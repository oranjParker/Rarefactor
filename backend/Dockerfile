FROM python:3.12-slim

RUN apt-get update && apt-get install -y \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

COPY --from=ghcr.io/astral-sh/uv:latest /uv /bin/uv

RUN BIN="/usr/local/bin" && \
    VERSION="1.54.0" && \
    curl -sSL \
    "https://github.com/bufbuild/buf/releases/download/v${VERSION}/buf-$(uname -s)-$(uname -m)" \
    -o "${BIN}/buf" && \
    chmod +x "${BIN}/buf"

WORKDIR /app

COPY backend/pyproject.toml backend/uv.lock ./

RUN uv sync --frozen

COPY . .

RUN buf generate protos

WORKDIR /app/backend

EXPOSE 50051

CMD ["uv", "run", "server.py"]