# Payment Service Makefile
# 
# Environment Variables:
#   POSTGRES_DSN - PostgreSQL connection string (default: postgres://app:app@localhost:5432/payments?sslmode=disable)
#   REDIS_ADDR   - Redis address (default: localhost:6379)
#   GRPC_ADDR    - gRPC server address (default: :8081)
#   ENV          - Environment (dev, test, prod)
#
# Examples:
#   export POSTGRES_DSN="postgres://user:pass@localhost:5432/db?sslmode=disable"
#   export REDIS_ADDR="localhost:6379"
#   export GRPC_ADDR=":8081"
#   export ENV="dev"
#
#   make run                    # Run with default config
#   make run POSTGRES_DSN="..." # Run with custom DB
#   make migrate-up             # Run migrations with POSTGRES_DSN
#   make generate               # Generate proto and sqlc code

.PHONY: help generate sqlc-validate migrate-up migrate-down migrate-create migrate-force run lint

# Default target
help:
	@echo "Available targets:"
	@echo "  generate     - Generate code from proto and sqlc"
	@echo "  sqlc-validate - Validate sqlc configuration and queries"
	@echo "  migrate-up   - Run database migrations up"
	@echo "  migrate-down - Rollback database migrations (1 step)"
	@echo "  migrate-create - Create a new migration file"
	@echo "  migrate-force - Force migration to specific version"
	@echo "  run          - Run the payment service"
	@echo "  lint         - Run linter and formatter (optional)"

# Generate code from proto and sqlc
generate:
	@echo "Generating code..."
	@echo "Generating Go code from proto files..."
	@protoc --plugin=protoc-gen-go=$(shell go env GOPATH)/bin/protoc-gen-go \
		--plugin=protoc-gen-go-grpc=$(shell go env GOPATH)/bin/protoc-gen-go-grpc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/payment/v1/*.proto
	@echo "Generating SQL code with sqlc..."
	@$(shell go env GOPATH)/bin/sqlc generate -f sqlc.yaml
	@echo "Code generation complete"

# Validate sqlc configuration and queries
sqlc-validate:
	@echo "Validating sqlc configuration..."
	@docker-compose run --rm sqlc validate -f sqlc.yaml
	@echo "Validation complete"

# Run database migrations up
migrate-up:
	@echo "Running migrations up..."
	@POSTGRES_DSN="$${POSTGRES_DSN:-postgres://app:app@localhost:5432/payments?sslmode=disable}" \
	docker-compose run --rm migrate -path=/migrations -database "$$POSTGRES_DSN" up
	@echo "Migrations completed"

# Rollback database migrations (rollback 1 step)
migrate-down:
	@echo "Rolling back migrations..."
	@POSTGRES_DSN="$${POSTGRES_DSN:-postgres://app:app@localhost:5432/payments?sslmode=disable}" \
	docker-compose run --rm migrate -path=/migrations -database "$$POSTGRES_DSN" down 1
	@echo "Migrations rolled back"

# Create a new migration file
migrate-create:
	@read -p "Enter migration name: " name; \
	docker-compose run --rm migrate create -ext sql -dir /migrations -seq $$name

# Force migration version (use with caution)
migrate-force:
	@read -p "Enter version to force: " version; \
	@POSTGRES_DSN="$${POSTGRES_DSN:-postgres://app:app@localhost:5432/payments?sslmode=disable}" \
	docker-compose run --rm migrate -path=/migrations -database "$$POSTGRES_DSN" force $$version

# Run the payment service
run:
	@echo "Starting payment service..."
	@go run ./cmd/paymentservice

# Run linter and formatter (optional)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping..."; \
	fi
	@echo "Running formatter..."
	@if command -v gofmt >/dev/null 2>&1; then \
		gofmt -s -w .; \
	else \
		echo "gofmt not found, skipping..."; \
	fi
	@echo "Linting complete"
