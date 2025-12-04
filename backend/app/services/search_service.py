import json
import logging
import sys
import os
import time

import grpc
from ..structures.trie import Trie
from ..structures.redis_rank_engine import RedisRankEngine
from sqlmodel import select, text
from ..database import get_session_context
from ..models import Document as DB_Document
from ..utils import apply_field_mask
from ..cache import get_redis_client


from typing import List
sys.path.append(os.path.join(os.path.dirname(__file__), "generated"))
from generated import search_pb2, search_pb2_grpc

logger = logging.getLogger(__name__)
class SearchService(search_pb2_grpc.SearchEngineServicer):
    def __init__(self):
        self.trie = Trie()
        self.ranker = RedisRankEngine()
        self.CACHE_TTL_SECONDS = 3600

    async def warm_up_trie(self):
        count = 0
        try:
            async with get_session_context() as session:
                statement = select(DB_Document.title)
                results = await session.execute(statement)
                titles = results.all()

                for title in titles:
                    if title:
                        self.trie.insert(title)
                        count += 1

            logger.info(f"Trie hot n' ready. Loaded {count} keys")
        except Exception as e:
            logger.error(f"This trie is cold: {e}")

    async def Autocomplete(self, request: search_pb2.AutocompleteRequest, context: grpc.aio.ServicerContext) -> search_pb2.AutocompleteResponse:
        start_time: float = time.perf_counter()
        prefix: str = request.prefix
        limit: int = 10

        candidates: List[str] = self.trie.autocomplete(prefix, limit)
        if not candidates:
            duration: float = (time.perf_counter() - start_time) * 1000
            return search_pb2.AutocompleteResponse(suggestions=[], duration_ms=duration)

        scored_candidates: List[tuple[str, float]] = await self.ranker.get_scores(candidates)

        scored_candidates.sort(key=lambda x: x.score, reverse=True)

        top_candidates = [term for term, score in scored_candidates]

        duration: float = (time.perf_counter() - start_time) * 1000
        return search_pb2.AutocompleteResponse(
            suggestions=top_candidates,
            duration_ms=duration
        )

    async def Search(
            self,
            request: search_pb2.SearchRequest,
            context: grpc.aio.ServicerContext
    ) -> search_pb2.SearchResponse:
        query = request.query.strip().lower()
        if not query:
            return search_pb2.SearchResponse(results=[], total_hits=0)
        cache_key: str = f"search_results:{request.query}"
        redis_client = get_redis_client()

        try:
            cached_data = redis_client.get(cache_key)
            if cached_data:
                logger.info(f"Found cached data for query {query}")

                results_list = json.loads(cached_data)
                proto_docs: List[DB_Document] = [
                    search_pb2.Document(
                        url=item.get("url", ""),
                        title = item.get("title", ""),
                        snippet = item.get("snippet", ""),
                        score = item.get("score", 0),
                    ) for item in results_list
                ]
                return search_pb2.SearchResponse(results=proto_docs, total_hits=len(proto_docs))

        except Exception as e:
            logger.error(f"Failed to get cached data for query {query}")

        logger.info(f"Querying DB for {query}")
        results_for_cache =  []
        proto_results = []

        try:
            async with get_session_context() as session:
                statement = select(DB_Document).where(
                    text("to_tsvector('english', title || ' ' || content) @@ plainto_tsquery('english', :q)")
                ).params(q=query).limit(20)

                db_results = await session.execute(statement)
                docs = db_results.all()
                for doc in docs:
                    snippet_preview = (doc.content[:200] + '...') if len(doc.content) > 200 else doc.content
                    pb_doc = search_pb2.Document(
                        url=doc.url,
                        title=doc.title,
                        snippet=snippet_preview,
                        score=doc.score if hasattr(doc, 'score') and doc.score else 1.0
                    )
                    proto_results.append(pb_doc)
                    results_for_cache.append({
                        "url": pb_doc.url,
                        "title": pb_doc.title,
                        "snippet": snippet_preview,
                        "score": pb_doc.score if hasattr(doc, 'score') and doc.score else 1.0,
                    })

                if results_for_cache:
                    await redis_client.setex(cache_key, self.CACHE_TTL_SECONDS, json.dumps(results_for_cache))

                    try:
                        await self.ranker.increment_score(query)
                    except Exception as rank_err:
                        logger.error(f"Failed to increment score for query '{query}' : {rank_err}")

        except Exception as e:
            logger.error(f"Database search failed: {e}")
            return search_pb2.SearchResponse(results=[], total_hits=0)

        return search_pb2.SearchResponse(
            results=proto_results,
            total_hits=len(proto_results)
        )

    async def UpdateDocument(
            self,
            request: search_pb2.UpdateDocumentRequest,
            context: grpc.aio.ServicerContext) -> search_pb2.Document:
        logger.info(f"Updating document: {request.url} with mask: {request.update_mask.paths}")

        async with get_session_context() as session:
            try:
                statement = select(DB_Document).where(DB_Document.url == request.url)
                results = await session.execute(statement)
                db_doc = results.first()

                if not db_doc:
                    await context.abort(grpc.StatusCode.NOT_FOUND, "Document not found")

                apply_field_mask(request.document, db_doc, request.update_mask)

                session.add(db_doc)
                await session.commit()
                await session.refresh(db_doc)

                return search_pb2.Document(
                    url=db_doc.url,
                    title=db_doc.title,
                    score=db_doc.score
                )

            except Exception as e:
                logger.error(f"Failed to update document {request.url}: {e}")
                await context.abort(grpc.StatusCode.INTERNAL, f"Failed to update document: {e}")


async def serve_grpc():
    server = grpc.aio.server()

    servicer = SearchService()

    # Register the servicer
    search_pb2_grpc.add_SearchServiceServicer_to_server(servicer, server)

    server.add_insecure_port(':50051')

    # CRITICAL: Run the warmup BEFORE we start accepting requests.
    # Since serve_grpc is awaited in main.py, this will pause startup until DB is read.
    await servicer.warm_up_trie()

    logger.info("Starting gRPC Server on port 50051...")
    await server.start()
    return server