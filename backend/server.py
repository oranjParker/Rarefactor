import logging
import grpc
import sys
import os

sys.path.append(os.path.join(os.path.dirname(__file__), "generated"))

from app.database import init_db
from generated import search_pb2_grpc, crawler_pb2_grpc

from app.services.search_service import SearchService
from app.services.crawler_service import CrawlerService

async def serve_grpc():
    logging.info("Starting server...")
    await init_db()

    server = grpc.aio.server()

    servicer = SearchService()
    crawler_service = CrawlerService()

    search_pb2_grpc.add_SearchEngineServicer_to_server(
        servicer, server
    )

    crawler_pb2_grpc.add_CrawlerServiceServicer_to_server(
        crawler_service, server
    )

    listen_addr = '0.0.0.0:50051'
    server.add_insecure_port(listen_addr)

    await servicer.warm_up_trie()

    logging.info("Starting gRPC Server on port 50051...")
    await server.start()
    return server