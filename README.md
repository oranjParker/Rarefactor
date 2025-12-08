# Rarefactor

Rarefactor is a two-part demo application that showcases a Python backend with gRPC + FastAPI and a modern React (Vite + TypeScript + Tailwind CSS v3) client. This root README explains the repository layout and the sub-applications inside.

## Repository structure

- backend
  - Python application that exposes a gRPC service (search) and an HTTP API via FastAPI which proxies to gRPC.
  - Responsible for autocomplete and search over stored documents, ranking, and caching.
- client
  - React + TypeScript + Vite single-page app for search UI and results, styled with Tailwind CSS v3.
  - Talks to the FastAPI HTTP endpoints.
- protos
  - Protocol Buffers definitions for the search service. Code generation is configured via `buf.gen.yaml`.
- data
  - Sample or seed data used by the backend (e.g., to populate a database or index).
- docker-compose.yml
  - Optional services and local orchestration during development.
- Makefile
  - Common developer commands to build, run, and generate code.

## How the pieces fit together

1. Protobuf contracts in `protos/` define the gRPC search interface.
2. The backend implements the gRPC service and also runs a FastAPI app that offers convenient HTTP routes for the frontend.
3. The client calls the FastAPI HTTP endpoints for autocomplete and search, and renders results.

## Quickstart

Prerequisites: Python 3.12, Node 18+, and optionally Docker.

Backend (from `backend/`):

```bash
uv sync  # or pip install -r requirements.txt equivalents if you prefer
uv run fastapi dev main.py
```

Client (from `client/`):

```bash
npm install
npm run dev
```

Open the client at http://localhost:5173 and ensure the backend is serving at http://localhost:8000.

Notes:
- The frontend uses Tailwind CSS v3 with a standard PostCSS setup. See `client/README.md` for details.
- If you see a Node.js version warning from Vite, it is informational; development and builds still work with Node 20.x in this demo.

## Sub-application READMEs

- See `backend/README.md` for backend technologies, architecture, and endpoints.
- See `client/README.md` for frontend stack (React + Vite + Tailwind), components, and scripts.
