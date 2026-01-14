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
- **Control Plane v1 (The Command Center):**
  - **Backend:** Implement crawl_jobs table to track execution history and success rates.
  - **Frontend:** Create a dedicated "Admin" dashboard in the React app.
  - **Visualization:** Basic progress bars, "URLs per second" counters, and a live-updating table of discovered URLs.
  - **Control:** Enable Pause/Resume/Stop functionality via the UI.
- **Content Quality Filters:**
  - Implement "Low Information Density" filters to discard maintenance pages and link farms.

## Phase 3: Transition to Distributed Systems (The Event Era)
- **Event-Driven Backbone (Kafka/Redpanda):**
  - Transition from internal Go channels to a Kafka-compatible message bus for URL discovery.
  - **Efficiency Note:** Prioritize Redpanda for lower resource overhead (C++ based) while maintaining full Kafka API compatibility.
  - Enable "Message Replay" to allow re-indexing without re-fetching content.
- **Shared Frontier:**
  - Decouple the Coordinator into a distributed manager where multiple Engine instances consume from the message bus.
- **Distributed Politeness:**
  - Use Redis-based global rate limiting to enforce domain politeness across multiple crawler nodes.
- **Control Plane v2 (System Health):**
  - Integrate Prometheus for metrics collection and Grafana for deep infrastructure monitoring.
  - Track consumer lag, worker throughput, and database latency.

## Phase 4: Advanced Engine & AI Integration
- **MCP (Model Context Protocol) Server:**
  - Implement an MCP interface to allow LLMs and AI Agents to use Rarefactor as a native "Search Tool."
- **Control Plane v3 (Data Insights & Marketing):**
  - **Advanced Visualization:** Implement D3.js or Three.js "Crawl Graphs" showing domain interlinking.
  - **Analytics:** Dashboard for "Embedding Drift" and search quality metrics to ensure the AI pipeline remains accurate.
- **Multi-Language Mastery:**
  - **Transcoding:** Edge-level charset detection for legacy encodings.
  - **Multilingual Embeddings:** Transition to multilingual-e5 models.
- **Geographic & TLD Sharding:**
  - Shard the Qdrant index based on Home Country/TLD (e.g., .cn, .ru) to optimize for region-specific content.
- **Headless Rendering:**
  - Integrate Chromedp for indexing JavaScript-heavy SPAs, managed through the Control Plane.

## Critical Considerations
- **Resource Accessibility:** Maintain a "Lite" configuration profile (Redis-only) alongside the "Pro" Kafka-based setup.
- **Cynical Performance Rule:** Raw SQL for high-volume interactions.
- **Type Safety:** gRPC/Protobuf contract as the absolute source of truth.
