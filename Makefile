.PHONY: build run test clean help

# Build the application
build:
	@echo "Building..."
	@go build -o bin/api example/main.go

# Run the application
run:
	@echo "Running..."
	@go run example/main.go

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

# Run the application in development mode with auto-reload (requires air)
dev:
	@echo "Running in development mode..."
	@air

# Generate OpenAPI specification
generate-spec:
	@echo "Generating OpenAPI specification..."
	@cd examples/with-swagger && go run . -generate-spec -spec-file=openapi.json

# Help command
help:
	@echo "Available commands:"
	@echo "  make build          - Build the application"
	@echo "  make run            - Run the application"
	@echo "  make test           - Run tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Run linter"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make deps           - Install dependencies"
	@echo "  make dev            - Run in development mode with auto-reload"
	@echo "  make generate-spec  - Generate OpenAPI specification"
	@echo "  make help           - Show this help message"

