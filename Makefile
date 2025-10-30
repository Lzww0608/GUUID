.PHONY: all build test bench coverage lint fmt vet clean help

# Variables
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)

# Color output
BLUE=\033[0;34m
NC=\033[0m # No Color

all: fmt vet test

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

fmt: ## Run go fmt on all source files
	@echo "$(BLUE)Running go fmt...$(NC)"
	@go fmt ./...

vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	@go vet ./...

lint: ## Run golangci-lint
	@echo "$(BLUE)Running golangci-lint...$(NC)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

test: ## Run tests
	@echo "$(BLUE)Running tests...$(NC)"
	@go test -v -race -count=1 ./...

test-short: ## Run short tests
	@echo "$(BLUE)Running short tests...$(NC)"
	@go test -v -short ./...

bench: ## Run benchmarks
	@echo "$(BLUE)Running benchmarks...$(NC)"
	@go test -bench=. -benchmem -run=^$$ ./...

coverage: ## Generate coverage report
	@echo "$(BLUE)Generating coverage report...$(NC)"
	@go test -race -coverprofile=coverage.txt -covermode=atomic ./...
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

build: ## Build examples
	@echo "$(BLUE)Building examples...$(NC)"
	@mkdir -p $(GOBIN)
	@go build -o $(GOBIN)/ ./examples/...

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning...$(NC)"
	@rm -rf $(GOBIN)
	@rm -f coverage.txt coverage.html
	@go clean -testcache

tidy: ## Run go mod tidy
	@echo "$(BLUE)Running go mod tidy...$(NC)"
	@go mod tidy

deps: ## Download dependencies
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	@go mod download

check: fmt vet lint test ## Run all checks (fmt, vet, lint, test)

.DEFAULT_GOAL := help

