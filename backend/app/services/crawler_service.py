import grpc
from generated import crawler_pb2, crawler_pb2_grpc
from ..crawler_engine import CrawlerEngine


class CrawlerService(crawler_pb2_grpc.CrawlerServiceServicer):
    async def Crawl(self, request, context):
        engine = CrawlerEngine(
            seed_url=request.seed_url,
            max_pages=request.max_pages
        )

        count = await engine.run()

        return crawler_pb2.CrawlResponse(
            pages_crawled=count,
            status="COMPLETED"
        )