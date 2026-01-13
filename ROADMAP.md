Rarefactor Engine: Roadmap & Future Considerations

This document tracks the architectural evolution and long-term goals of the Rarefactor General Engine.

## Phase 1: The High-Quality Foundation (Completed)
**Goal:** A reliable, single-node crawler with high-performance relational storage and semantic search.

### Key Achievements:
- **Go Refactor:** Successfully transitioned from Python MVP to a gRPC-first architecture with HTTP/1.1 Gateway support.
- **Vector Search:** Integrated Qdrant for semantic similarity search with local embedding generation.
- **Frontier Management:** Implemented a priority DomainHeap with Logarithmic Diversity Weighting to ensure fair domain crawling.
- **Budget Control:** Implemented eTLD+1 (Registered Domain) Budgeting to prevent subdomain "explosions" (e.g., Fandom/Tumblr wikis).
- **Sanitization:** Implemented UTF-8 enforcement to prevent gRPC marshalling panics on legacy-encoded sites.
- **Search/Autocomplete:** Fully functional frontend-to-backend pipeline with standardized parameter mapping.

## Phase 2: Observability & Resilience (Next Target)
**Goal:** Move from "fire-and-forget" to a persistent, manageable system.

- **State Persistence (Checkpointing):**
  - Move the visited map and domainQueue to Postgres or Redis so the crawler can resume after a crash or restart.
- **Job Management:**
  - Implement a `crawl_jobs` table to track progress, start/end times, and total pages found.
  - Assign a `job_id` to every crawl request and provide a `GET /v1/status/{id}` endpoint.
- **Control Plane:**
  - API endpoint to Pause, Resume, or Stop an active crawl without killing the backend process.
- **Content Quality Filters:**
  - Implement "Low Information Density" filters to discard "Under Maintenance" pages, image-only pages, and link farms.

## Phase 3: Transition to Distributed Systems
- **Shared Frontier:**
  - Decouple the Coordinator logic entirely into a Redis-backed manager.
  - Allow multiple instances of the Engine to consume from a shared task queue.
- **Distributed Politeness:**
  - Use Redis INCR or CELLAR to enforce domain rate-limiting across a fleet of 100+ workers.
- **Task Brokerage:**
  - Transition to a message broker (RabbitMQ or Redis Streams) for distributing URL tasks to specialized worker nodes.

## Phase 4: Advanced Engine Capabilities
- **Geographic & TLD Sharding:**
  - Categorize data and shard the Qdrant index based on Home Country/TLD (e.g., separating .cn, .ru, and .com nodes).
  - Allows for region-specific weighting and quality tuning.
- **Multi-Language Mastery:**
  - **Transcoding:** Use charset detection to convert legacy encodings (GBK, Shift-JIS) at the edge.
  - **Multilingual Embeddings:** Switch to models like `multilingual-e5` for cross-language search.
- **Duplicate Detection:**
  - Implement SimHash or MinHash to ignore near-duplicate content across different URLs.
- **Headless Rendering:**
  - Integrate Playwright or Chromedp for indexing JavaScript-heavy single-page applications.

## Critical Considerations
- **Cynical Performance Rule:** Raw SQL is preferred for all high-volume interactions.
- **Type Safety:** gRPC/Protobuf contract remains the absolute source of truth.
- **Privacy & Safety:** Evaluating safety-filters for indexed content to prevent illegal/harmful results.
