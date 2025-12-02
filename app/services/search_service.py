import logging
from sys import prefix

import grpc
from generated import search_pb2, search_pb2_grpc
from app.structures.trie import Trie
from sqlmodel import select
from app.database import get_session_context
from app.models import Document as DB_Document
from app.utils import apply_field_mask

from typing import List
from generated import search_pb2, search_pb2_grpc

GLOBAL_TRIE = Trie()

async def warm_up_trie():
    logging.info("Warming up trie")
    count = 0
    try:
        async with get_session_context() as session:
            statement = select(DB_Document.title)
            results = await session.exec(statement)
            titles = results.all()

            for title in titles:
                if title:
                    GLOBAL_TRIE.insert(title)
                    count += 1

        logging.info(f"Trie hot n' ready. Loaded {count} keys")
    except Exception as e:
        logging.error(f"This trie is cold: {e}")


class SearchService(search_pb2_grpc.SearchEngineServicer):
    def __init__(self):
        pass

    async def Autocomplete(self, request: search_pb2.AutocompleteRequest, context: grpc.aio.ServicerContext) -> search_pb2.AutocompleteResponse:
        prefix: str = request.prefix
        limit: int = request.limit

        suggestions: List[str] = GLOBAL_TRIE.autocomplete(prefix=prefix, limit=limit)

        return search_pb2.AutocompleteResponse(
            suggestions=suggestions,
            duration_ms=0
        )

    async def Search(
            self,
            request: search_pb2.SearchRequest,
            context: grpc.aio.ServicerContext
    ) -> search_pb2.SearchResponse:
        logging.info(f"Received Query: {request.query}")

        dummy_doc = search_pb2.Document(
            url="https://python.org",
            title="Python Official",
            snippet="Python is a programming language...",
            score=0.99
        )

        return search_pb2.SearchResponse(
            results=[dummy_doc],
            total_hits=1
        )

    async def UpdateDocument(
            self,
            request: search_pb2.UpdateDocumentRequest,
            context: grpc.aio.ServicerContext) -> search_pb2.Document:
        logging.info(f"Updating document: {request.url} with mask: {request.update_mask.paths}")

        async with get_session_context() as session:
            try:
                statement = select(DB_Document).where(DB_Document.url == request.url)
                results = await session.exec(statement)
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

            finally:
                await session.close()