import asyncio
import logging
import grpc
import sys
import os

sys.path.append(os.path.join(os.path.dirname(__file__), "generated"))

from app.database import init_db
from generated import search_pb2_grpc, crawler_pb2_grpc
from app.services.search_service import SearchService
from app.services.crawler_service import CrawlerService

async def serve():
    logging.info("Starting server...")
    await init_db()

    server = grpc.aio.server()
    search_pb2_grpc.add_SearchEngineServicer_to_server(
        SearchService(), server
    )

    crawler_pb2_grpc.add_CrawlerServiceServicer_to_server(
        CrawlerService(), server
    )

    listen_addr = '[::]:50051'
    server.add_insecure_port(listen_addr)
    logging.info("Rarefactor Server listening on %s...", listen_addr)

    await server.start()
    await server.wait_for_termination()

if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO)
    asyncio.run(serve())