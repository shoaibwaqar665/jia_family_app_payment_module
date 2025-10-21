# Makefile for Jia Payment Service

.PHONY: help build test test-unit test-integration test-e2e lint clean docker-build docker-run migrate-up migrate-down proto-generate

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the application
	@echo "Building payment service..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/payment-service ./cmd/paymentservice

build-local: ## Build the application for local development
	@echo "Building payment service for local development..."
	go build -o bin/payment-service ./cmd/paymentservice

# Test targets
test: test-unit test-integration ## Run all tests

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	go test -v -tags=integration ./...

test-e2e: ## Run end-to-end tests
	@echo "Running end-to-end tests..."
	go test -v -tags=e2e ./...

test-coverage: test-unit ## Generate test coverage report
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Linting and formatting
lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

lint-fix: ## Run linter with auto-fix
	@echo "Running linter with auto-fix..."
	golangci-lint run --fix

format: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Database targets
migrate-up: ## Run database migrations up
	@echo "Running database migrations up..."
	@if [ -z "$(DSN)" ]; then echo "Please set DSN environment variable"; exit 1; fi
	migrate -path migrations -database "$(DSN)" up

migrate-down: ## Run database migrations down
	@echo "Running database migrations down..."
	@if [ -z "$(DSN)" ]; then echo "Please set DSN environment variable"; exit 1; fi
	migrate -path migrations -database "$(DSN)" down

migrate-force: ## Force migration version
	@echo "Forcing migration version..."
	@if [ -z "$(DSN)" ] || [ -z "$(VERSION)" ]; then echo "Please set DSN and VERSION environment variables"; exit 1; fi
	migrate -path migrations -database "$(DSN)" force $(VERSION)

# Protocol buffer targets
proto-generate: ## Generate Go code from protobuf files
	@echo "Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		--validate_out="lang=go:." \
		proto/payment/v1/payment_service.proto

proto-descriptor: ## Generate protobuf descriptor file
	@echo "Generating protobuf descriptor..."
	protoc --descriptor_set_out=proto/payment/v1/payment_service.pb \
		--include_imports \
		proto/payment/v1/payment_service.proto

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t payment-service:latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -p 8081:8081 --env-file .env payment-service:latest

docker-compose-up: ## Start services with docker-compose
	@echo "Starting services with docker-compose..."
	docker-compose -f docker/docker-compose.yaml up -d

docker-compose-down: ## Stop services with docker-compose
	@echo "Stopping services with docker-compose..."
	docker-compose -f docker/docker-compose.yaml down

docker-compose-logs: ## Show docker-compose logs
	@echo "Showing docker-compose logs..."
	docker-compose -f docker/docker-compose.yaml logs -f

# Development targets
dev: ## Start development environment
	@echo "Starting development environment..."
	docker-compose -f docker/docker-compose.yaml up -d postgres redis
	@echo "Waiting for services to be ready..."
	sleep 10
	@echo "Running migrations..."
	DSN="postgres://postgres:postgres@localhost:5432/payment_service?sslmode=disable" make migrate-up
	@echo "Starting application..."
	go run ./cmd/paymentservice

dev-clean: ## Clean development environment
	@echo "Cleaning development environment..."
	docker-compose -f docker/docker-compose.yaml down -v
	docker system prune -f

# Security targets
security-scan: ## Run security scan
	@echo "Running security scan..."
	gosec ./...

security-audit: ## Run security audit
	@echo "Running security audit..."
	go list -json -deps ./... | nancy sleuth

# Performance targets
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

profile-cpu: ## Generate CPU profile
	@echo "Generating CPU profile..."
	go test -cpuprofile=cpu.prof -bench=. ./...

profile-mem: ## Generate memory profile
	@echo "Generating memory profile..."
	go test -memprofile=mem.prof -bench=. ./...

# Cleanup targets
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f *.prof
	go clean -cache

# Dependencies
deps: ## Install dependencies
	@echo "Installing dependencies..."
	go mod download
	go mod verify

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

deps-vendor: ## Vendor dependencies
	@echo "Vendoring dependencies..."
	go mod vendor

# Tools installation
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install github.com/nancy-org/nancy@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/envoyproxy/protoc-gen-validate@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Environment setup
setup: install-tools deps ## Setup development environment
	@echo "Setting up development environment..."
	@if [ ! -f .env ]; then cp example.env .env; fi
	@echo "Development environment setup complete!"
	@echo "Please update .env file with your configuration"

# Health check
health: ## Check service health
	@echo "Checking service health..."
	@curl -f http://localhost:8081/health || echo "Service is not running"

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	godoc -http=:6060 &
	@echo "Documentation available at http://localhost:6060"

# Release
release: build test lint ## Create a release build
	@echo "Creating release build..."
	@if [ -z "$(VERSION)" ]; then echo "Please set VERSION environment variable"; exit 1; fi
	docker build -t payment-service:$(VERSION) .
	docker tag payment-service:$(VERSION) payment-service:latest
	@echo "Release $(VERSION) created successfully!"