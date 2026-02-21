# Rarefactor Backend: Distributed DAG Engine

## Core Architecture: `internal/core`
The backbone of the Rarefactor engine is an event-driven Directed Acyclic Graph (DAG) that allows for type-safe, concurrent processing of documents.

- **`GraphRunner[T]`**: The primary orchestrator that manages the flow of data through the DAG. It supports parallel execution branches and ensures that the system can scale horizontally.
- **`Node[T]`**: Individual units of work within the graph. Nodes can be **Processors** (transforming data), **Sinks** (side-effects like storage), or **Hybrids**.
- **`Document[T]`**: The generic unit of data that flows through the system. It carries the primary content, metadata, and includes a `.Clone()` method to satisfy the **Immutability Contract** when the DAG forks into multiple branches.

## Processing Pipeline: `internal/processor`
Rarefactor utilizes a series of specialized processors to transform raw web data into high-quality vector embeddings:

- **SmartCrawler**: A heuristic-based crawler that decides between standard HTML fetching and headless rendering.
- **SPACrawler**: Uses `chromedp` for full headless browser rendering, ensuring JavaScript-heavy sites are correctly indexed.
- **Security**: Validates URLs and enforces safety constraints (e.g., avoiding internal IP ranges).
- **Politeness**: Enforces domain-specific crawl delays using Redis-backed distributed state and Lua scripts.
- **Chunker**: Breaks down large documents into manageable segments for embedding, with strict UTF-8 enforcement.
- **Embedding**: Generates high-dimensional vectors using local models (e.g., via the Infinity engine).
- **Metadata**: Extracts and normalizes structured information (titles, summaries, etc.) from crawled content.

## AI & LLM Integration
We support multiple LLM providers through a unified interface for enrichment and analysis:
- **Gemini**: Google's high-performance multimodal models.
- **Ollama**: For local LLM inference.
- **Mock**: Used for high-speed testing and CI/CD without external dependencies.

## Testing & Quality Assurance
We maintain >85% unit test coverage to ensure system reliability:

- **Database Testing**: Requires `pgxmock` for mocking PostgreSQL interactions.
- **API Mocks**: Uses `httptest` for simulating external web servers and gRPC-Gateway endpoints.
- **SPA Testing**: Headless browser tests using `chromedp` will **automatically skip** if Chrome is not detected locally, preventing CI failures in restricted environments.

## Running the Backend
The backend is fully containerized via `docker-compose.yml`. To start the distributed stack for local development with GPU support:
```bash
docker compose --profile gpu up -d
```
This includes the Go server, PostgreSQL, Redis, NATS JetStream, and Qdrant.
