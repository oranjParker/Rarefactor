from sqlmodel import SQLModel
from sqlalchemy.ext.asyncio import AsyncSession, create_async_engine, async_sessionmaker
from contextlib import asynccontextmanager

DATABASE_URL = "sqlite+aiosqlite:///./rarefactor.db"

engine = create_async_engine(
    DATABASE_URL, echo=True, future=True,
)

async_session_factory = async_sessionmaker(
    engine,
    class_=AsyncSession,
    expire_on_commit=False,
)
async def init_db():
    async with engine.begin() as conn:
        await conn.run_sync(SQLModel.metadata.create_all)

@asynccontextmanager
async def get_session_context():
    async with async_session_factory() as session:
        try:
            yield session
        finally:
            await session.close()

