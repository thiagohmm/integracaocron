# Multi-stage build for Go application
# Stage 1: Builder
FROM golang:1.25.1-alpine AS builder

# Metadata
LABEL maintainer="integracaocron"
LABEL description="IntegracaoCron - Sistema de integração com Oracle e RabbitMQ"

# Set working directory
WORKDIR /app

# ✅ OTIMIZAÇÃO: Instalar dependências e limpar cache em um único layer
RUN apk add --no-cache git ca-certificates tzdata && \
  rm -rf /var/cache/apk/*

# ✅ OTIMIZAÇÃO: Copiar apenas go.mod e go.sum primeiro para melhor cache
COPY go.mod go.sum ./

# ✅ OTIMIZAÇÃO: Download de dependências com cache mount (BuildKit)
# Cache persiste entre builds, acelerando rebuilds significativamente
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  go mod download && \
  go mod verify

# ✅ OTIMIZAÇÃO: Copiar apenas arquivos necessários (usando .dockerignore)
# Isso reduz o contexto de build e acelera o COPY
COPY . .

# ✅ OTIMIZAÇÃO: Build com todas as otimizações e cache
# -ldflags="-w -s" remove debug info (reduz ~30-40% do tamanho)
# -trimpath remove caminhos do sistema de arquivos
# -buildvcs=false desabilita versionamento (mais rápido)
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -buildvcs=false \
  -ldflags="-w -s -extldflags '-static'" \
  -trimpath \
  -a -installsuffix cgo \
  -o integracaocron ./cmd/app/

# Stage 2: Minimal runtime image
FROM alpine:3.19

# Metadata
LABEL maintainer="integracaocron"
LABEL description="IntegracaoCron - Sistema de integração com Oracle e RabbitMQ"
LABEL org.opencontainers.image.source="https://github.com/thiagohmm/integracaocron"

# ✅ OTIMIZAÇÃO: Combinar instalação e criação de usuário em menos layers
RUN apk --no-cache --update add \
  ca-certificates \
  tzdata && \
  addgroup -g 1000 appuser && \
  adduser -D -u 1000 -G appuser appuser && \
  rm -rf /var/cache/apk/*

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