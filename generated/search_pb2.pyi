from google.protobuf import field_mask_pb2 as _field_mask_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AutocompleteRequest(_message.Message):
    __slots__ = ()
    PREFIX_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    prefix: str
    limit: int
    def __init__(self, prefix: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class AutocompleteResponse(_message.Message):
    __slots__ = ()
    SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    suggestions: _containers.RepeatedScalarFieldContainer[str]
    duration_ms: float
    def __init__(self, suggestions: _Optional[_Iterable[str]] = ..., duration_ms: _Optional[float] = ...) -> None: ...

class SearchRequest(_message.Message):
    __slots__ = ()
    QUERY_FIELD_NUMBER: _ClassVar[int]
    query: str
    def __init__(self, query: _Optional[str] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ()
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_HITS_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[Document]
    total_hits: int
    def __init__(self, results: _Optional[_Iterable[_Union[Document, _Mapping]]] = ..., total_hits: _Optional[int] = ...) -> None: ...

class Document(_message.Message):
    __slots__ = ()
    URL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    url: str
    title: str
    snippet: str
    score: float
    def __init__(self, url: _Optional[str] = ..., title: _Optional[str] = ..., snippet: _Optional[str] = ..., score: _Optional[float] = ...) -> None: ...

class UpdateDocumentRequest(_message.Message):
    __slots__ = ()
    URL_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    UPDATE_MASK_FIELD_NUMBER: _ClassVar[int]
    url: str
    document: Document
    update_mask: _field_mask_pb2.FieldMask
    def __init__(self, url: _Optional[str] = ..., document: _Optional[_Union[Document, _Mapping]] = ..., update_mask: _Optional[_Union[_field_mask_pb2.FieldMask, _Mapping]] = ...) -> None: ...
