# Rarefactor Backend (Go Refactor)

## Technology Stack
- **Language:** Go (1.21+)
- **Communication:** gRPC for internal calls; gRPC-Gateway for HTTP/1.1 REST.
- **Relational DB:** Postgres with pgxpool.
- **Cache/Queue:** Redis.
- **Vector DB:** Qdrant for semantic search.
- **Embeddings:** Infinity engine (`michaelf34/infinity`) serving `nomic-embed-text-v1.5`.

## Running with Docker
The backend stack is fully containerized. To start all services including the Go server:
```bash
docker-compose up -d
```
To stop all services:
```bash
docker-compose down
```
This starts:
- `rarefactor-server`: The Go gRPC/REST gateway.
- `rarefactor-embeddings`: Infinity embedding server (port 7997).
- `rarefactor-postgres`: Relational storage.
- `rarefactor-redis`: Task queue and cache.
- `rarefactor-qdrant`: Vector database.

## Architecture Highlights
- **Logarithmic Penalty Weighting:** Prevents large domains from starving smaller sites.
- **eTLD+1 Budgeting:** Tracks crawl depth across entire subdomain families (e.g., fandom.com) using publicsuffix.
- **UTF-8 Enforcement:** Strict sanitization to prevent gRPC marshalling errors.
