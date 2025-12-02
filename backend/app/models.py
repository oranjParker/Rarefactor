from typing import Optional
from sqlmodel import Field, SQLModel
from datetime import datetime

class Document(SQLModel, table=True):
    id: Optional[int] = Field(default=None, primary_key=True)
    url: str = Field(index=True, unique=True)
    title: str
    snippet: str
    content: str
    crawled_at: datetime = Field(default_factory=datetime.utcnow)
    score: float = Field(default=0.0)