.PHONY: help build run test clean deps tidy migrate-up migrate-down

# Variables
BINARY_NAME=server
BUILD_DIR=build
CMD_DIR=cmd/server
GO=go
GOCMD=go

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Download dependencies
	$(GO) mod download
	$(GO) mod tidy

tidy: ## Tidy go.mod and go.sum
	$(GO) mod tidy

build: ## Build the binary
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

run: ## Run the server
	$(GO) run ./$(CMD_DIR)

test: ## Run all tests
	$(GO) test -v ./...

test-coverage: ## Run tests with coverage
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

migrate-up: ## Run database migrations (requires psql)
	@echo "Running migrations..."
	psql -U postgres -d nvide_live -f migrations/001_initial_schema.sql
	psql -U postgres -d nvide_live -f migrations/002_seed_data.sql

migrate-down: ## Drop and recreate database (WARNING: destructive)
	@echo "Dropping and recreating database..."
	-dropdb nvide_live
	createdb nvide_live
	make migrate-up

dev: ## Run in development mode with air (if installed)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "air not found, installing..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

fmt: ## Format code
	$(GO) fmt ./...

vet: ## Run go vet
	$(GO) vet ./...

lint: ## Run golangci-lint (if installed)
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

.DEFAULT_GOAL := help