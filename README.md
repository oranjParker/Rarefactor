# Rarefactor: Event-Driven RAG Ingestion Engine

![Rarefactor Demo](assets/rarefactor_demo.gif)

Rarefactor is a high-performance, distributed, event-driven DAG (Directed Acyclic Graph) designed for large-scale RAG (Retrieval-Augmented Generation) ingestion.

## Project Status
Rarefactor has successfully transitioned to a Distributed, Event-Driven Architecture (Phase 2). It features a gRPC-first Go backend, high-performance web discovery, and seamless integration with vector databases for semantic retrieval.

For a detailed look at future phases, see the [ROADMAP.md](./ROADMAP.md).

## Core Capabilities
- **Event-Driven DAG:** Orchestrate complex ingestion pipelines using `GraphRunner` with strict immutability contracts.
- **Smart Crawling:** Support for both standard HTML scraping (`goquery`) and headless SPA rendering (`chromedp`).
- **Distributed State:** Redis-backed frontier management with Lua scripts for global politeness and eTLD+1 budgeting.
- **High-Performance Ingestion:** NATS JetStream for persistent message queuing and asynchronous enrichment.

## Tech Stack
- **Backend:** Go (1.25+)
- **API:** gRPC (Internal) & gRPC-Gateway (REST)
- **State/Metadata:** PostgreSQL
- **Distributed Coordination:** Redis (Frontier & Politeness)
- **Message Queuing:** NATS JetStream
- **Vector DB:** Qdrant (Semantic Search)
- **Frontend:** React (Vite + TypeScript + Tailwind CSS)

## Sub-application READMEs
- See [backend/README.md](./backend/README.md) for backend internal architecture, processors, and testing.
- See [client/README.md](./client/README.md) for frontend stack, components, and scripts.

## Getting Started

### Prerequisites
- Go 1.25+
- Node.js 18+
- Docker & Docker Compose

### 1. Infrastructure
Start the full distributed stack (Postgres, Redis, Qdrant, NATS, and Embeddings):
```bash
docker-compose up -d
```

### 2. Backend (Manual/Development)
If you prefer to run the Go server manually (after starting infrastructure):
```bash
cd backend
go run cmd/server/main.go
```

### 3. Frontend
Navigate to the `client` directory and start the development server:
```bash
cd client
npm install
npm run dev
```

### 4. Initial Crawl
Trigger an ingestion job via grpcurl:
```powershell
grpcurl -plaintext -d '{\"seed_url\": \"https://go.dev\", \"max_pages\": 100, \"max_depth\": 2, \"crawl_mode\": \"broad\"}' localhost:50051 protos.v1.CrawlerService/Crawl
```
