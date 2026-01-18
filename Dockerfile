# Multi-stage build for Go application
# Stage 1: Builder
FROM golang:1.25.1-alpine AS builder

# Metadata
LABEL maintainer="integracaocron"
LABEL description="IntegracaoCron - Sistema de integração com Oracle e RabbitMQ"

# Set working directory
WORKDIR /app

# Install build dependencies (git needed for some Go modules)
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies (using BuildKit cache mount for faster rebuilds)
# This will cache the module downloads between builds
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download && \
  go mod verify

# Copy source code
COPY . .

# Build the application with optimizations
# -ldflags="-w -s" removes debug info and symbol tables (smaller binary)
# -trimpath removes file system paths from the binary
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-w -s -extldflags '-static'" \
  -trimpath \
  -a -installsuffix cgo \
  -o integracaocron ./cmd/app/

# Stage 2: Minimal runtime image
FROM alpine:3.19

# Metadata
LABEL maintainer="integracaocron"
LABEL description="IntegracaoCron - Sistema de integração com Oracle e RabbitMQ"
LABEL org.opencontainers.image.source="https://github.com/thiagohmm/integracaocron"

# Install only runtime dependencies (ca-certificates for HTTPS/TLS, tzdata for timezone)
RUN apk --no-cache --update add \
  ca-certificates \
  tzdata \
  && rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -g 1000 appuser && \
  adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder --chown=appuser:appuser /app/integracaocron /app/integracaocron

# Switch to non-root user
USER appuser

# Expose HTTP port
EXPOSE 3013

# Health check - verify process is running
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
  CMD pgrep -f integracaocron > /dev/null || exit 1

# Run the application
ENTRYPOINT ["/app/integracaocron"]