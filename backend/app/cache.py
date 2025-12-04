import os
import redis.asyncio as redis
import logging

REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379")
pool = redis.ConnectionPool.from_url(REDIS_URL, decode_responses=True)

def get_redis_client() -> redis.Redis:
    return redis.Redis(connection_pool=pool)

async def check_redis_connection():
    client = get_redis_client()
    try:
        await client.ping()
        logging.info(f"✅ Connected to Redis at {REDIS_URL}")
    except Exception as e:
        logging.error(f"❌ Redis Connection Failed ({REDIS_URL}): {e}")
