# Rarefactor Backend (Go Refactor)

## The Refactor: From Python MVP to Go Production Engine
If you wish to view the Python MVP, please visit the `demo` branch. While the Python version was successful for proving the concept, the Go refactor was undertaken to solve three critical issues:

1.  **Concurrency & Throughput:** Moving from AsyncIO to Go's native CSP model (Goroutines/Channels) for 10x crawling performance.
2.  **From Lexical to Semantic:** Moving from Postgres `tsvector` (keyword matching) to Vector Embeddings via Qdrant (semantic matching).
3.  **The Frontier Problem:** Replacing simple URL queues with a priority-weighted Domain Frontier.

## Advanced Frontier Management
The crawler's intelligence resides in the `coordinator.go`, which manages the discovery frontier:

-   **Domain Diversity:** Using a `DomainHeap` (Priority Queue), the system ensures we aren't just hammering one domain, distributing load across the web.
-   **Logarithmic Penalty Weighting:** To prevent large sites from starving the crawler, we implement a fairness algorithm: `math.Log1p(float64(h[i].PageCount)) * 10`. This ensures a diverse index by increasing the "cost" of crawling subsequent pages from the same host.
-   **eTLD+1 Budgeting:** We use `publicsuffix` logic to treat subdomains (e.g., `user1.tumblr.com` and `user2.tumblr.com`) as a single registered domain for budgeting purposes, preventing "infinite" subdomain crawls from consuming all resources.

## Semantic Search & Local Embeddings
-   **Local Embedding Generation:** We utilize the Infinity engine to serve `nomic-embed-text-v1.5` locally. This removes dependency on expensive third-party APIs (like OpenAI) and ensures data privacy and low latency.
-   **Vector DB:** We utilize Qdrant to perform high-dimensional similarity searches, allowing the engine to understand "context" rather than just "keywords."

## The Communication Layer
-   **gRPC-First Philosophy:** The `protos/v1` contract is the Single Source of Truth for the entire system. All internal communication is strictly typed and high-performance.
-   **gRPC-Gateway:** We use gRPC-Gateway to automatically generate a RESTful HTTP/1.1 API from our proto definitions. This allows the frontend to remain simple while the backend retains the performance of gRPC.

## Modern Tooling
-   **UTF-8 Enforcement:** The crawler includes strict sanitization logic (`SanitizeUTF8`) as a necessary guardrail for crawling the "Wild Web," where legacy encodings or malformed byte sequences often crash modern gRPC marshallers.

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
