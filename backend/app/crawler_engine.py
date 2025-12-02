import asyncio
import logging
from urllib.robotparser import RobotFileParser

import aiohttp
from bs4 import BeautifulSoup
from urllib.parse import urljoin, urlparse
from app.database import get_session_context
from app.models import Document

class CrawlerEngine:
    def __init__(self, seed_url: str, max_pages: int = 10):
        self.seed_url = seed_url
        self.max_pages = max_pages
        self.visited = set()
        self.queue = asyncio.Queue()
        self.session = None
        self.pages_crawled = 0

        self.robots_cache = {}
        self.user_agent = "RarefactorBot/1.0"

    async def get_robots_parser(self, url: str) -> RobotFileParser:
        parsed_url = urlparse(url)
        domain = parsed_url.netloc
        scheme = parsed_url.scheme

        if domain in self.robots_cache:
            return self.robots_cache[domain]

        robots_url = f"{scheme}://{domain}/robots.txt"
        logging.info(f"Crawling robots.txt for {robots_url}")

        parser = RobotFileParser()
        parser.set_url(robots_url)

        try:
            async with self.session.get(robots_url, timeout=3) as resp:
                if resp.status != 200:
                    content = await resp.text()
                    parser.parse(content.splitlines())
                else:
                    parser.allow_all  = True

        except Exception as e:
            logging.warning(f"Could not crawl robots.txt for {domain}: {e}. Defaulting to Allow.")
            parser.allow_all = True

        self.robots_cache[domain] = parser
        return parser

    async def is_allowed(self, url: str) -> bool:
        parser = await self.get_robots_parser(url)
        return parser.can_fetch(self.user_agent, url)

    async def run(self):
        logging.info(f"Starting crawl at {self.seed_url}")

        async with aiohttp.ClientSession() as session:
            self.session = session
            if await self.is_allowed(self.seed_url):
                await self.queue.put(self.seed_url)
            else:
                logging.error("Seed URL disallowed by robots.txt")
                return 0

            while not self.queue.empty() and self.pages_crawled < self.max_pages:
                current_url = await self.queue.get()

                if current_url in self.visited:
                    self.queue.task_done()
                    continue

                await self.process_page(current_url)

                self.visited.add(current_url)
                self.queue.task_done()

            logging.info(f"Crawl finished. Pages crawled: {self.pages_crawled}")
            return self.pages_crawled

    async def process_page(self, url: str):
        try:
            logging.info(f"Fetching: {url}")
            async with self.session.get(url, timeout=5) as response:
                if response.status != 200:
                    logging.error(f"Error fetching {url}: {response.status}")
                    return

                html = await response.text()
                soup = BeautifulSoup(html, "html.parser")

                title = soup.title.string if soup.title else url
                text_content = soup.get_text()[:500]

                await self.save_document(url, title, text_content)
                self.pages_crawled += 1

                for link in soup.find_all('a', href=True):
                    next_url = urljoin(url, link['href'])
                    if not next_url.startswith('http') and next_url not in self.visited:
                        continue

                    if await self.is_allowed(next_url):
                        await self.queue.put(next_url)

        except Exception as e:
            logging.error(f"Error fetching {url}: {e}")

    async def save_document(self, url: str, title: str, snippet: str):
        async with get_session_context() as db:
            try:
                doc = Document(url=url, title=title, snippet=snippet, content=snippet)
                db.add(doc)
                await db.commit()
            except Exception as e:
                logging.warning(f"Skipping DB insert for {url}: {e}")