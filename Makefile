.PHONY: help generate migrate-up migrate-down migrate-create migrate-force run lint

# Default target
help:
	@echo "Available targets:"
	@echo "  generate     - Generate code from proto and sqlc"
	@echo "  migrate-up   - Run database migrations up"
	@echo "  migrate-down - Rollback database migrations (1 step)"
	@echo "  migrate-create - Create a new migration file"
	@echo "  migrate-force - Force migration to specific version"
	@echo "  run          - Run the payment service"
	@echo "  lint         - Run linter and formatter"

# Generate code from proto and sqlc
generate:
	@echo "Generating code..."
	@go generate ./...
	@echo "Code generation complete"

# Run database migrations up
migrate-up:
	@echo "Running migrations up..."
	@docker run --rm -v $(PWD)/migrations:/migrations --network host migrate/migrate:v4 \
		-path=/migrations \
		-database="postgres://paymentservice:paymentservice123@localhost:5432/paymentservice?sslmode=disable" \
		up
	@echo "Migrations completed"

# Rollback database migrations (rollback 1 step)
migrate-down:
	@echo "Rolling back migrations..."
	@docker run --rm -v $(PWD)/migrations:/migrations --network host migrate/migrate:v4 \
		-path=/migrations \
		-database="postgres://paymentservice:paymentservice123@localhost:5432/paymentservice?sslmode=disable" \
		down 1
	@echo "Migrations rolled back"

# Create a new migration file
migrate-create:
	@read -p "Enter migration name: " name; \
	docker run --rm -v $(PWD)/migrations:/migrations migrate/migrate:v4 \
		create -ext sql -dir /migrations -seq $$name

# Force migration version (use with caution)
migrate-force:
	@read -p "Enter version to force: " version; \
	docker run --rm -v $(PWD)/migrations:/migrations --network host migrate/migrate:v4 \
		-path=/migrations \
		-database="postgres://paymentservice:paymentservice123@localhost:5432/paymentservice?sslmode=disable" \
		force $$version

# Run the payment service
run:
	@echo "Starting payment service..."
	@go run cmd/paymentservice/main.go

# Run linter and formatter
lint:
	@echo "Running linter..."
	@golangci-lint run
	@echo "Running formatter..."
	@gofmt -s -w .
	@echo "Linting complete"
