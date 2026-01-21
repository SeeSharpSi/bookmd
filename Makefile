.PHONY: help build run dev test test-verbose lint fmt clean docker-build docker-run install-tools templ-generate templ-fmt

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build commands
build: ## Build the application
	go build -o bin/htmx_quickstart .

run: ## Run the application
	go run .

dev: ## Run with hot reload using air
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "Air is not installed. Run 'make install-tools' to install it."; \
		exit 1; \
	fi

# Testing commands
test: ## Run all tests
	go test ./...

test-verbose: ## Run tests with verbose output
	go test -v ./...

test-cover: ## Run tests with coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Code quality commands
lint: ## Run linter
	go vet ./...

fmt: ## Format Go code
	gofmt -w .
	templ fmt

# Templ commands
templ-generate: ## Generate Go code from templ files
	templ generate

templ-fmt: ## Format templ files
	templ fmt

# Cleanup commands
clean: ## Clean build artifacts
	rm -rf bin/ tmp/ coverage.out coverage.html
	go clean

clean-all: clean ## Clean all artifacts including vendor
	rm -rf vendor/

# Docker commands
docker-build: ## Build Docker image
	docker build -t htmx-quickstart .
	docker run -p 9779:9779 --env-file .env htmx-quickstart
	docker run -p 9779:9779 -v $(PWD):/app --env-file .env htmx-quickstart

# Installation commands
install-tools: ## Install development tools
	go install github.com/air-verse/air@latest
	go install github.com/a-h/templ/cmd/templ@latest

# Environment setup
setup: ## Setup development environment
	cp .env.example .env
	@echo "Environment file created. Edit .env with your settings."

# Health check
health: ## Check if the application is running
	@if curl -s http://localhost:9779/health > /dev/null; then \
		echo "Application is healthy"; \
	else \
		echo "Application is not responding"; \
		exit 1; \
	fi
