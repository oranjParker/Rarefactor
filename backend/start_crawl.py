import asyncio
import grpc
import logging
import sys
import os

sys.path.append(os.path.join(os.path.dirname(__file__), "generated"))

from generated import crawler_pb2, crawler_pb2_grpc

async def start_crawl():
    async with grpc.aio.insecure_channel('localhost:50051') as channel:
        stub = crawler_pb2_grpc.CrawlerServiceStub(channel)

        print("🚀 Sending Crawl Command to Server...")

        request = crawler_pb2.CrawlRequest(
            seed_url="https://docs.python.org/3/",
            max_pages=100,
            max_depth=3
        )

        try:
            response = await stub.Crawl(request)
            print("✅Crawl Complete!")
            print(f"Pages Crawled: {response.pages_crawled}")
            print(f"STATUS: {response.status}")

        except grpc.RpcError as e:
            print(f"RPC Failed: {e}")

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    asyncio.run(start_crawl())