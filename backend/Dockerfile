# Use a lightweight Python base
FROM python:3.12-slim

# 1. Install System Dependencies
# curl: to download uv and buf
# git: often needed for pip dependencies
RUN apt-get update && apt-get install -y \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

# 2. Install uv (The Modern Python Manager)
# We use the official installer and add it to PATH
COPY --from=ghcr.io/astral-sh/uv:latest /uv /bin/uv

# 3. Install Buf (The Modern Proto Manager)
# This makes the build architecture-agnostic
RUN BIN="/usr/local/bin" && \
    VERSION="1.28.1" && \
    curl -sSL \
    "https://github.com/bufbuild/buf/releases/download/v${VERSION}/buf-$(uname -s)-$(uname -m)" \
    -o "${BIN}/buf" && \
    chmod +x "${BIN}/buf"

# 4. Set Working Directory
WORKDIR /app

# 5. Copy Dependency Definitions first (Caching Layer)
COPY pyproject.toml uv.lock ./

# 6. Install Python Dependencies
# --frozen ensures we stick exactly to the lockfile
RUN uv sync --frozen

# 7. Copy the rest of the application
COPY . .

# 8. Generate Protobufs (The "Build" Step)
# This ensures fresh generated code inside the image
RUN buf generate protos

# 9. Expose the gRPC Port
EXPOSE 50051

# 10. Run the Server
# We use 'uv run' to ensure we use the virtual environment created in step 6
CMD ["uv", "run", "server.py"]