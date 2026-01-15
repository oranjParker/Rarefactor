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

## Phase 2: Observability & Resilience (Next Target)

**Goal:** Move from "fire-and-forget" to a persistent, manageable system with a basic UI.

- **State Persistence (Checkpointing):**
  - Move the visited map and domainQueue to Postgres or Redis to support crawl resumption.
- **Service Modularization (Prep for Decoupling):**
  - Refactor the backend into distinct internal packages so that the Crawler and Search services can eventually be compiled into separate binaries.
  - Implement shared internal interfaces for database access to ensure consistency across services.
- **Control Plane v1 (The Command Center):**
  - **Backend:** Implement `crawl_jobs` table and `GET /v1/status/{id}` to track execution history.
  - **Frontend:** Create a dedicated "Admin" dashboard in the React app with live progress bars and URL discovery feeds.
  - **Control:** Enable Pause/Resume/Stop functionality via the UI.
- **Content Quality Filters:**
  - Implement "Low Information Density" filters to discard maintenance pages and link farms.

## Phase 3: Transition to Distributed Systems (The Event Era)

**Goal:** Physically decouple services for independent scaling and fault tolerance.

- **Service Decoupling:**
  - Separate the Crawler and Search services into distinct Docker containers.
  - **Crawler Service:** Focused on fetching, parsing, and embedding generation.
  - **Search Service:** Focused on Qdrant retrieval, Autocomplete, and user-facing API.
- **Event-Driven Backbone (Kafka/Redpanda):**
  - Transition from internal Go channels to a Kafka-compatible message bus (Redpanda) to bridge the decoupled services.
  - Enable "Message Replay" to allow re-indexing without re-fetching content.
- **Distributed Politeness:**
  - Use Redis-based global rate limiting to enforce domain politeness across multiple crawler nodes.
- **Control Plane v2 (System Health):**
  - Integrate Prometheus for metrics collection and Grafana for deep infrastructure monitoring.

## Phase 4: Advanced Engine & AI Integration

- **MCP (Model Context Protocol) Server:**
  - Implement an MCP interface to allow LLMs and AI Agents to use Rarefactor as a native "Search Tool."
- **Control Plane v3 (Data Insights & Marketing):**
  - **Advanced Visualization:** Implement D3.js or Three.js "Crawl Graphs" showing domain interlinking.
  - **Analytics:** Dashboard for "Embedding Drift" and search quality metrics.
- **Geographic & TLD Sharding:**
  - Shard the Qdrant index based on Home Country/TLD (e.g., .cn, .ru).
- **Headless Rendering:**
  - Integrate Chromedp for indexing JavaScript-heavy SPAs.

## Critical Considerations

- **Resource Accessibility:** Maintain a "Lite" configuration profile (Single Binary + Redis) alongside the "Pro" Distributed setup.
- **Independent Scaling:** Allow the Search service to scale horizontally during high traffic without increasing Crawler resources.
- **Cynical Performance Rule:** Raw SQL for high-volume interactions.
- **Type Safety:** gRPC/Protobuf contract as the absolute source of truth.
