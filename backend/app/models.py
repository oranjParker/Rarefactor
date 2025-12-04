from typing import Optional
from sqlmodel import Field, SQLModel
from sqlalchemy import Index
from sqlalchemy.dialects.postgresql import TSVECTOR
from sqlalchemy import text as sa_text
from datetime import datetime

class Document(SQLModel, table=True):
    id: Optional[int] = Field(default=None, primary_key=True)
    url: str = Field(index=True, unique=True)
    title: str
    snippet: str
    content: str

    __table_args__ = (
        Index(
            "ix_document_search_vector",
            sa_text("to_tsvector('english', title || ' ' || content)"),
            postgresql_using="gin"
        ),
    )

    crawled_at: datetime = Field(default_factory=datetime.utcnow)
    score: float = Field(default=0.0)