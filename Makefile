.PHONY: help build build-no-cache run-dev run-binary test unit-test integration-test clean lint fmt install-tools docker-build docker-compose-build docker-up docker-down run-docker migrate-up migrate-down coverage vet tidy deps sqlc-generate swagger-generate

# Variables
GO := go
GOFLAGS := -v
COVERAGE_FILE := coverage.out
COVERAGE_THRESHOLD := 70

# Platform detection
ifeq ($(OS),Windows_NT)
    RM := del /Q
    RMDIR := rmdir /S /Q
    RACE_FLAG :=
else
    RM := rm -f
    RMDIR := rm -rf
    RACE_FLAG := -race
endif

help: ## Display this help message
	@echo Available targets:
	@echo.
	@echo   build                 - Build the application binary
	@echo   build-no-cache        - Build the application binary (no cache)
	@echo   run-dev               - Run the application with 'go run' (requires postgres in docker)
	@echo   run-binary            - Run the application binary (requires build and postgres in docker)
	@echo   install-tools         - Install required development tools
	@echo   unit-test             - Run unit tests only
	@echo   integration-test      - Run integration tests only (requires database)
	@echo   test                  - Run all tests (unit + integration)
	@echo   coverage              - Generate test coverage report
	@echo   lint                  - Run linters
	@echo   fmt                   - Format code
	@echo   vet                   - Run go vet
	@echo   migrate-up            - Run database migrations
	@echo   migrate-down          - Rollback database migrations
	@echo   docker-build          - Build Docker image of the app
	@echo   docker-compose-build  - Build docker-compose (postgres + app)
	@echo   docker-up             - Start Docker containers
	@echo   docker-down           - Stop Docker containers
	@echo   run-docker            - Run the app in docker (full stack)
	@echo   clean                 - Clean build artifacts
	@echo   deps                  - Download and verify dependencies
	@echo   tidy                  - Tidy go.mod and go.sum
	@echo   sqlc-generate         - Generate sqlc code
	@echo   swagger-generate      - Generate Swagger/OpenAPI documentation
	@echo   help                  - Display this help message
	@echo.

install-tools: ## Install required development tools
	$(GO) install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	$(GO) install github.com/swaggo/swag/cmd/swag@latest

build: swagger-generate ## Build the application binary
	$(GO) build -o bin/purchase-api ./cmd/purchase-api

build-no-cache: swagger-generate ## Build the application binary (no cache)
	$(GO) clean -cache
	$(GO) build -o bin/purchase-api ./cmd/purchase-api

run-dev: swagger-generate ## Run the application with 'go run' (requires postgres in docker)
	@echo Starting purchase-api with 'go run'...
	@echo Make sure postgres is running: make docker-up
	$(GO) run ./cmd/purchase-api/main.go

run-binary: build ## Run the application binary (requires build & postgres in docker)
	@echo Starting purchase-api binary...
	@echo Make sure postgres is running: make docker-up
	@echo.
	./bin/purchase-api

# Unit Tests
unit-test: ## Run unit tests only
	@echo Running unit tests...
	$(GO) test $(GOFLAGS) $(RACE_FLAG) -coverprofile=$(COVERAGE_FILE) \
		./internal/app \
		./internal/api \
		./internal/domain
	@$(GO) tool cover -func=$(COVERAGE_FILE) | find "total"

# Integration Tests
migrate-up: ## Run database migrations
	@echo Running database migrations...
	$(GO) run ./cmd/migrate -dir up

migrate-down: ## Rollback database migrations
	@echo Rolling back database migrations...
	$(GO) run ./cmd/migrate -dir down

integration-test: migrate-up ## Run integration tests only (requires database)
	@echo Running integration tests...
	$(GO) test $(GOFLAGS) $(RACE_FLAG) ./tests/...

# All Tests
test: unit-test integration-test ## Run all tests (unit + integration)
	@echo All tests completed successfully!

# Test Coverage
coverage: unit-test ## Generate and display test coverage report
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo Coverage report generated: coverage.html

# Code Quality
lint: ## Run linters (requires golangci-lint)
	@echo Running linters...
	@where golangci-lint >nul 2>&1 || (echo golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && exit 1)
	golangci-lint run ./...

fmt: ## Format code
	$(GO) fmt ./...
	@echo Code formatted

vet: ## Run go vet
	$(GO) vet ./...

# Docker
docker-build: ## Build Docker image of the app
	docker build -t purchase-api:latest .

docker-compose-build: ## Build docker-compose (postgres + app)
	docker-compose build

docker-up: ## Start Docker containers (docker-compose)
	docker-compose up -d

docker-down: ## Stop Docker containers
	docker-compose down

run-docker: docker-compose-build ## Run the app in docker (full stack)
	@echo Starting purchase-api in Docker...
	docker-compose up -d
ifeq ($(OS),Windows_NT)
	@timeout /t 5 /nobreak
else
	@sleep 5
endif
	@echo Purchase API is running at http://localhost:8080
	@echo Database: postgres://postgres:postgres@localhost:5432/purchase_api

# Utilities
clean: ## Clean build artifacts and test files
	$(GO) clean
	$(RM) $(COVERAGE_FILE) coverage.html
	$(RMDIR) bin
	@echo Cleaned

deps: ## Download and verify dependencies
	$(GO) mod download
	$(GO) mod verify
	@echo Dependencies verified

tidy: ## Tidy go.mod and go.sum
	$(GO) mod tidy
	@echo go.mod and go.sum tidied

sqlc-generate: ## Generate sqlc code
	sqlc generate

swagger-generate: ## Generate Swagger/OpenAPI documentation
	@echo Generating Swagger documentation...
	swag init -g cmd/purchase-api/main.go --parseInternal
	@echo Swagger documentation generated in docs/

.DEFAULT_GOAL := help
