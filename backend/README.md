# Backend — Rarefactor

This backend powers the Rarefactor search demo. It combines gRPC for high-performance service-to-service communication with a FastAPI HTTP façade used by the React client.

## Technologies

- FastAPI: Lightweight HTTP API used by the frontend (`/autocomplete`, `/search`).
- gRPC (grpc.aio): Primary RPC interface defined via Protocol Buffers in `protos/` and generated into `backend/generated/`.
- Protobuf: Message contracts for the search service (see `protos/search.proto`). Generation is configured in `buf.gen.yaml`.
- SQLModel + PostgreSQL: Persistence layer for storing documents and executing full‑text queries via PostgreSQL `to_tsvector`/`plainto_tsquery`.
- Redis: Caching for search results and a simple ranking signal for autocomplete suggestions.
- Async Python: End-to-end async I/O (database, Redis, and gRPC) for efficiency.

## Architecture

1. The gRPC server implements `SearchEngine` with methods like `Autocomplete`, `Search`, and `UpdateDocument` (see `app/services/search_service.py`).
2. A FastAPI app starts alongside the gRPC server and exposes HTTP routes that proxy to gRPC using an async channel. This makes the frontend simple to wire while keeping a strong gRPC contract.
3. Autocomplete uses an in‑memory Trie warmed from document titles plus a Redis-backed ranker to score suggestions.
4. Search queries PostgreSQL using `to_tsvector`/`plainto_tsquery` for basic full‑text search, then caches results in Redis.

## Key files

- `main.py`: App lifespan management; starts gRPC server and FastAPI, wires CORS and gRPC stub.
- `app/services/search_service.py`: Core service implementation (Trie autocomplete, Redis ranker, DB search, caching).
- `app/database.py`: Async SQLModel session configuration (PostgreSQL URL via `DATABASE_URL`).
- `app/structures/*`: Data structures like Trie and rank engine.
- `generated/`: Protobuf/gRPC Python stubs (generated — typically committed for convenience in demos).

## Running locally

Environment:
- Python 3.12, PostgreSQL, Redis.
- Set `DATABASE_URL` and any Redis connection envs as needed.

Commands (from `backend/`):

```bash
uv sync
uv run fastapi dev main.py
```

The FastAPI HTTP server defaults to `http://localhost:8000` and the gRPC server to `localhost:50051`.

## HTTP endpoints (via FastAPI)

- `GET /autocomplete?q=term&limit=10` → `{ suggestions: string[], duration_ms: number }`
- `GET /search?q=query` → `{ results: { url, title, snippet, score }[], total_hits }`

These call through to the gRPC service using an async `SearchEngineStub`.
