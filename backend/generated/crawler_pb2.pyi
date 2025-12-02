from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class CrawlRequest(_message.Message):
    __slots__ = ()
    SEED_URL_FIELD_NUMBER: _ClassVar[int]
    MAX_PAGES_FIELD_NUMBER: _ClassVar[int]
    MAX_DEPTH_FIELD_NUMBER: _ClassVar[int]
    seed_url: str
    max_pages: int
    max_depth: int
    def __init__(self, seed_url: _Optional[str] = ..., max_pages: _Optional[int] = ..., max_depth: _Optional[int] = ...) -> None: ...

class CrawlResponse(_message.Message):
    __slots__ = ()
    PAGES_CRAWLED_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    pages_crawled: int
    status: str
    def __init__(self, pages_crawled: _Optional[int] = ..., status: _Optional[str] = ...) -> None: ...
