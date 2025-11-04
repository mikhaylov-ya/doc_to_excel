# Multi-stage build for smaller final image
FROM golang:1.23-alpine AS builder

# Build argument for Go parallelism
ARG GOMAXPROCS=1

# Install build dependencies
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    poppler-utils \
    antiword

WORKDIR /app

# Set Go build concurrency
ENV GOMAXPROCS=${GOMAXPROCS}

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies with cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY *.go ./

# Build with optimizations for speed and smaller binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o server .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies for document conversion
RUN apk add --no-cache \
    ca-certificates \
    poppler-utils \
    tesseract-ocr \
    tzdata \
    wv \
    && \
    # Install catdoc from edge/community repo
    apk add --no-cache catdoc --repository=http://dl-cdn.alpinelinux.org/alpine/edge/community || \
    echo "catdoc not available"

# Set timezone (optional)
ENV TZ=UTC

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/server .

# Copy frontend files
COPY frontend/ ./frontend/

# Create temp directory for file processing
RUN mkdir -p ./temp

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the server
CMD ["./server"]
