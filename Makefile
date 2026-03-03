.PHONY: help generate build run test clean docker-up docker-down migrate-up migrate-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

generate: ## Generate code from OpenAPI spec
	@echo "Installing oapi-codegen if not present..."
	@go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@echo "Generating Go code from OpenAPI spec..."
	@mkdir -p internal/generated/api
	@$$(go env GOPATH)/bin/oapi-codegen -package api -generate types,server,spec -o internal/generated/api/api.gen.go api/openapi/marketplace.yaml
	@echo "Code generation complete!"

build: generate ## Build the application
	@echo "Building application..."
	@go build -o bin/marketplace-server cmd/server/main.go
	@echo "Build complete! Binary: bin/marketplace-server"

run: ## Run the application locally
	@go run cmd/server/main.go

test: ## Run tests
	@go test -v -race -coverprofile=coverage.out ./...
integration-test: ## Run integration tests with Docker
	@echo "Running integration tests..."
	@cd test && go test -v -timeout 5m

	@go tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts and generated code
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf internal/generated/
	@rm -f coverage.out coverage.html
	@echo "Clean complete!"

docker-up: ## Start all services with docker-compose
	@docker-compose up -d
	@echo "Services started! API available at http://localhost:8080"

docker-down: ## Stop all services
	@docker-compose down

docker-build: ## Build docker image
	@docker build -t marketplace-backend:latest .

migrate-up: ## Run database migrations up
	@echo "Running migrations..."
	@docker-compose exec -T postgres psql -U marketplace -d marketplace < migrations/init.sql || true
	@echo "Migrations complete!"

migrate-down: ## Run database migrations down
	@echo "Rolling back migrations..."
	@docker-compose exec -T postgres psql -U marketplace -d marketplace -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@echo "Rollback complete!"

deps: ## Install Go dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies installed!"

lint: ## Run linter
	@golangci-lint run ./...

.DEFAULT_GOAL := help