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