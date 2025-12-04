import asyncio
import logging
import signal

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
    search_service = SearchService()
    crawler_service = CrawlerService()
    search_pb2_grpc.add_SearchEngineServicer_to_server(
        search_service, server
    )

    crawler_pb2_grpc.add_CrawlerServiceServicer_to_server(
        crawler_service, server
    )

    await search_service.warm_up_trie()

    listen_addr = '0.0.0.0:50051'
    server.add_insecure_port(listen_addr)
    logging.info("Rarefactor Server listening on %s...", listen_addr)

    await server.start()
    await server.wait_for_termination()

    async def shutdown():
        logging.info("Shutting down server...")
        await server.stop(grace=5)
        loop.stop()

    loop = asyncio.get_event_loop()
    for sig in (signal.SIGTERM, signal.SIGINT):
        loop.add_signal_handler(
            sig,
            lambda s=sig: asyncio.create_task(shutdown())
        )

    try:
        await server.wait_for_termination()
    except asyncio.CancelledError:
        logging.info("Server task cancelled, shutting down...")
        await server.stop(grace=5)

if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO)
    try:
        asyncio.run(serve())
    except KeyboardInterrupt:
        pass