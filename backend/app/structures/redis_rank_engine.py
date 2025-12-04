import logging
from typing import List, Tuple

import redis
from redis import Redis

from ..cache import get_redis_client

class RedisRankEngine:
    def __init__(self):
        pass

    async def increment_score(self, term: str) -> None:
        try:
            await self.redis_client.zincrby(self.TRENDING_KEY, 1.0, term.lower())
        except redis.exceptions.ResponseError:
            logging.error(f"Failed to increment score for {term}")

    async def get_scores(self, terms: List[str]) -> List[Tuple[str, float]]:
        if not terms:
            return []
        try:
            redis = get_redis_client()
            async with redis.pipeline() as pipeline:
                for term in terms:
                    pipeline.zscore("global_search_scores", term)

                scores = await pipeline.execute()

            ranked_results = []
            for term, score in zip(terms, scores):
                final_score = float(score) if score else 0.0
                ranked_results.append((term, final_score))

            ranked_results.sort(key=lambda x: x[1], reverse=True)

            return ranked_results

        except Exception as e:
            logging.error(f"RedisRanker get_scores failed: {e}")
            return [(term, 0.0) for term in terms]