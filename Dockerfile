# ── Stage 1: Build frontend ─────────────────────────────────────────
FROM node:26-alpine AS frontend-builder
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# ── Stage 2: Build backend ──────────────────────────────────────────
FROM golang:1.26-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev  # for modernc.org/sqlite (pure Go, but needs C linker for sqlite)
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 go build -o /sharknado ./cmd/sharknado/

# ── Stage 3: Runtime ───────────────────────────────────────────────
FROM python:3.13-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*
RUN groupadd -g 1000 shark && useradd -m -u 1000 -g shark shark

# Install Python tools: streamrip for Qobuz/Deezer search+download, tidalapi for Tidal search
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc g++ libjpeg62-turbo-dev zlib1g-dev libffi-dev && \
    pip install --no-cache-dir streamrip tidalapi && \
    apt-get purge -y gcc g++ && apt-get autoremove -y && \
    rm -rf /var/lib/apt/lists/*

# Install tidal-dl-ng from vendored tarball (removed from PyPI)
COPY tidal_dl_ng.tar.gz /tmp/
RUN pip install --no-cache-dir /tmp/tidal_dl_ng.tar.gz && rm /tmp/tidal_dl_ng.tar.gz

# Copy backend binary
COPY --from=backend-builder --chown=shark:shark /sharknado /sharknado

# Copy frontend build
COPY --from=frontend-builder --chown=shark:shark /src/dist /frontend/dist

# Copy search scripts
COPY --chown=shark:shark scripts/ /app/scripts/

# Runtime directories
RUN mkdir -p /data /downloads /library /cache /app/scripts && chown -R shark:shark /data /downloads /library /cache /app/scripts

USER shark
WORKDIR /home/shark

ENV PORT=8000
ENV DOWNLOAD_DIR=/downloads
ENV LIBRARY_DIR=/library
ENV CACHE_DIR=/cache
ENV DB_PATH=/data/sharknado.db
ENV FRONTEND_DIR=/frontend/dist

EXPOSE 8000
ENTRYPOINT ["/sharknado"]
