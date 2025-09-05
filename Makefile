.PHONY: build run test clean docker-build docker-run docker-clean help

# Build variables
BINARY_NAME=miwifi-exporter
VERSION=$(shell git describe --tags --always --dirty)
COMMIT=$(shell git rev-parse --short HEAD)
DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
LDFLAGS=-ldflags="-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Default target
all: build

# Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Run the binary
run: build
	./$(BINARY_NAME)

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

# Build Docker image
docker-build:
	docker build -t $(BINARY_NAME):latest .

# Run Docker container
docker-run: docker-build
	docker run -d --name $(BINARY_NAME) -p 9001:9001 $(BINARY_NAME):latest

# Stop and remove Docker container
docker-stop:
	docker stop $(BINARY_NAME) || true
	docker rm $(BINARY_NAME) || true

# Clean Docker images
docker-clean: docker-stop
	docker rmi $(BINARY_NAME):latest || true

# Run with docker-compose
docker-compose-up:
	docker-compose up -d --build

# Stop docker-compose
docker-compose-down:
	docker-compose down

# View logs
logs:
	docker-compose logs -f

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Check for vulnerabilities
sec:
	gosec ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Show help
help:
	@echo "Available targets:"
	@echo "  build           - Build the binary"
	@echo "  run             - Build and run the binary"
	@echo "  test            - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  clean           - Clean build artifacts"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-run      - Run Docker container"
	@echo "  docker-stop     - Stop and remove Docker container"
	@echo "  docker-clean    - Clean Docker images"
	@echo "  docker-compose-up  - Start with docker-compose"
	@echo "  docker-compose-down - Stop docker-compose"
	@echo "  logs            - View docker-compose logs"
	@echo "  fmt             - Format code"
	@echo "  lint            - Run linter"
	@echo "  sec             - Check for vulnerabilities"
	@echo "  deps            - Download dependencies"
	@echo "  help            - Show this help"
