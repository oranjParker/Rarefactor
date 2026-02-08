# Rarefactor Engine: Roadmap & Future Considerations

This document tracks the architectural evolution and long-term goals of the Rarefactor General Engine.

## Phase 1: The High-Quality Foundation (Completed)

**Goal:** A reliable, single-node crawler with high-performance relational storage and semantic search.

### Key Achievements:

- **Go Refactor:** Successfully transitioned from Python MVP to a gRPC-first architecture with HTTP/1.1 Gateway support.
- **Vector Search:** Integrated Qdrant for semantic similarity search with local embedding generation.
- **Frontier Management:** Implemented a priority DomainHeap with Logarithmic Diversity Weighting.
- **Budget Control:** Implemented eTLD+1 (Registered Domain) Budgeting via publicsuffix.
- **Sanitization:** Strict UTF-8 enforcement for gRPC stability.

## Phase 2: Distributed DAG Architecture (Current Focus)

**Goal:** Transform the linear pipeline into an Event-Driven Directed Acyclic Graph (DAG) for independent scaling and type-safe parallelism.

### Graph Orchestration (The Core):

- **Implement GraphRunner** to manage parallel branches (e.g., Security vs. Discovery).
- **Immutability Contract:** Implement `.Clone()` on `Document[T]` to ensure thread-safe fork processing.
- **Hybrid Nodes:** Support nodes that act as both Processors (Transform) and Sinks (Side-Effect) for the Discovery loop.

### Distributed State & Politeness:

- **Redis Frontier:** Moved the "Visited Map" and Domain Counters to Redis for distributed coordination.
- **Logarithmic Fairness:** Implemented Redis Lua scripts to enforce logarithmic time penalties on aggressive domains.
- **Event-Driven Queue:** Replaced in-memory heaps with NATS JetStream for persistent work distribution.

### Functional Refactor:

- Convert all processors to "Pure Functions" (Input -> Output) with no internal state or side effects.

### Smart Crawling:

- **Headless Fallback:** Integrate `chromedp` for indexing JavaScript-heavy SPAs when heuristic checks fail.
- **Boilerplate Removal:** Optimize vector quality by stripping nav, footer, and script tags pre-ingestion.

## Phase 3: Observability & Resilience (Next Target)

**Goal:** Move from "fire-and-forget" to a persistent, manageable system with a basic UI.

### Control Plane v1 (The Command Center):

- **Backend:** Implement `crawl_jobs` table and `GET /v1/status/{id}` to track execution history.
- **Frontend:** Create a dedicated "Admin" dashboard with live progress bars and URL discovery feeds.
- **Control:** Enable Pause/Resume/Stop functionality via NATS control signals.

### Advanced Metrics:

- Integrate Prometheus for worker throughput and latency monitoring.

## Phase 4: Advanced Engine & AI Integration

**Goal:** transform Rarefactor into an "Intelligence Infrastructure" for AI Agents.

### MCP (Model Context Protocol) Server:

- Implement an MCP interface to allow LLMs (Claude/Gemini) to use Rarefactor as a native "Search Tool."

### Content Quality Filters:

- Implement "Low Information Density" filters to discard maintenance pages and link farms before embedding.

### Advanced Visualization:

- Implement D3.js or Three.js "Crawl Graphs" showing domain interlinking.

### Geographic & TLD Sharding:

- Shard the Qdrant index based on Home Country/TLD (e.g., .cn, .ru).

## Phase 5: Search Service & RAG Showcase (Long Term)

**Goal:** The public-facing demonstration of the RAG infrastructure.

### Search Service V2:

- Implement the Frequency/Recency Decay ranking algorithm.
- **Hybrid Retrieval:** Merge Vector Search results with Redis "Hot Cache" data.

### Cache Invalidation:

- Implement Event-Based Invalidation (Delete-on-Update) to ensure search results are never stale.

### Public Demo:

- A polished UI demonstrating the speed and accuracy of the RAG pipeline.

## Critical Considerations

- **The "Pointer Trap":** Ensure all forks in the DAG use deep copies of metadata to prevent data races.
- **Resource Accessibility:** Maintain a "Lite" configuration profile (Single Binary + Redis) alongside the "Pro" Distributed setup.
- **Cynical Performance Rule:** Raw SQL for high-volume interactions; avoid ORM overhead in the hot path.
- **Type Safety:** gRPC/Protobuf contract as the absolute source of truth for inter-service communication.
