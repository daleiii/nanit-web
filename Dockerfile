# Frontend build stage
FROM node:18-alpine AS frontend-build

WORKDIR /app/frontend

# Copy frontend package files
COPY frontend/package.json frontend/package-lock.json* ./

# Install dependencies (including dev dependencies for build)
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi

# Copy frontend source
COPY frontend/ ./

# Build the frontend
RUN npm run build

# Backend build stage
FROM --platform=$BUILDPLATFORM golang:1.24.0 AS backend-build

# Install build dependencies for SQLite and cross-compilation
RUN apt-get update && apt-get install -y \
    gcc libc6-dev sqlite3 libsqlite3-dev \
    gcc-aarch64-linux-gnu libc6-dev-arm64-cross \
    && rm -rf /var/lib/apt/lists/*

ADD cmd /app/cmd
ADD pkg /app/pkg
ADD go.mod /app/
ADD go.sum /app/
ADD scripts /app/scripts

# Copy built frontend files to replace old web directory
COPY --from=frontend-build /app/frontend/dist /app/web

WORKDIR /app

# Download dependencies
RUN go mod download

ARG CI_COMMIT_SHORT_SHA
ARG TARGETOS TARGETARCH

# Enable CGO for SQLite support with cross-compilation
RUN if [ "$TARGETARCH" = "arm64" ]; then \
        export CC=aarch64-linux-gnu-gcc; \
    fi && \
    CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags "-X main.GitCommit=$CI_COMMIT_SHORT_SHA" -o ./bin/nanit ./cmd/nanit/*.go

# Final production stage
FROM debian:bookworm-slim

COPY --from=backend-build /app/bin/nanit /app/bin/nanit
COPY --from=backend-build /app/scripts /app/scripts
COPY --from=backend-build /app/web /app/web

RUN apt-get -yqq update && \
    apt-get install -yq --no-install-recommends ca-certificates ffmpeg bash curl jq sqlite3 libsqlite3-0 && \
    apt-get autoremove -y && \
    apt-get clean -y

RUN mkdir -p /data && \
    chmod +x /app/scripts/*.sh

WORKDIR /app
ENTRYPOINT ["/app/bin/nanit"]