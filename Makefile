# Makefile for IntegracaoCron

.PHONY: build run test clean docker-build docker-run help

# Variables
APP_NAME=integracaocron
BINARY_DIR=bin
MAIN_PATH=./cmd/app
DOCKER_IMAGE=integracaocron:latest

# Default target
help:
	@echo "Available targets:"
	@echo "  build        - Build the Go application"
	@echo "  run          - Run the application locally"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run with Docker Compose"
	@echo "  docker-stop  - Stop Docker containers"
	@echo "  jaeger-up    - Start Jaeger tracing"
	@echo "  jaeger-down  - Stop Jaeger tracing"
	@echo "  jaeger-logs  - Show Jaeger logs"
	@echo "  full-stack   - Start full stack (app + Jaeger + RabbitMQ)"
	@echo "  mod-tidy     - Tidy Go modules"
	@echo "  fmt          - Format Go code"
	@echo "  vet          - Run go vet"
	@echo "  help         - Show this help message"

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BINARY_DIR)
	@go build -o $(BINARY_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BINARY_DIR)/$(APP_NAME)"

# Run the application locally
run: build
	@echo "Starting $(APP_NAME)..."
	@./$(BINARY_DIR)/$(APP_NAME)

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)
	@go clean

# Build Docker image
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	@docker build -t $(DOCKER_IMAGE) .

# Run with Docker Compose
docker-run:
	@echo "Starting with Docker Compose..."
	@docker-compose up -d

# Stop Docker containers
docker-stop:
	@echo "Stopping Docker containers..."
	@docker-compose down

# Start Jaeger tracing
jaeger-up:
	@echo "Starting Jaeger tracing..."
	@docker run -d --name jaeger \
		-p 16686:16686 \
		-p 14268:14268 \
		-p 14250:14250 \
		-p 6831:6831/udp \
		-p 6832:6832/udp \
		jaegertracing/all-in-one:latest
	@echo "Jaeger UI available at: http://localhost:16686"

# Stop Jaeger tracing
jaeger-down:
	@echo "Stopping Jaeger tracing..."
	@docker stop jaeger || true
	@docker rm jaeger || true

# Show Jaeger logs
jaeger-logs:
	@docker logs -f jaeger

# Start full stack with Jaeger
full-stack:
	@echo "Starting full stack with Jaeger..."
	@docker-compose -f docker-compose.jaeger.yml up -d
	@echo "Services started:"
	@echo "  - Jaeger UI: http://localhost:16686"
	@echo "  - RabbitMQ Management: http://localhost:15672 (admin/admin123)"
	@echo "  - Application API: http://localhost:8080"

# Tidy Go modules
mod-tidy:
	@echo "Tidying Go modules..."
	@go mod tidy

# Format Go code
fmt:
	@echo "Formatting Go code..."
	@go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Development workflow
dev: fmt vet test build

# Production build
prod-build:
	@echo "Building for production..."
	@mkdir -p $(BINARY_DIR)
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o $(BINARY_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Production build complete: $(BINARY_DIR)/$(APP_NAME)"