.PHONY: clean build install run run-debug fmt test lint help dev-tools

BINARY_NAME=mcp-elasticsearch
BUILD_DIR=bin
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT_HASH?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

help:
	@echo "mcp-elasticsearch development commands:"
	@echo "Version: $(VERSION)"
	@echo
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT_HASH)"
	@echo "Build Time: $(BUILD_TIME)"

build: deps ## Build the binary
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT_HASH) -X main.buildTime=$(BUILD_TIME) -s -w -extldflags '-static'" \
		-a -installsuffix cgo \
		-o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME) ($(VERSION))"

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	go clean
	@echo "Cleaned build artifacts"

deps: ## Download and tidy dependencies
	go mod download
	go mod tidy
	@echo "Dependencies installed"

install: build ## Install binary to system
	install -c $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to /usr/local/bin/"

uninstall: ## Uninstall binary from system
	rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME) from /usr/local/bin/"

run: ## Run the server with default configuration
	CGO_ENABLED=0 go run -ldflags "-X main.version=$(VERSION) -s -w" .

run-debug: ## Run with debug logging
	MCP_ES_LOG_LEVEL=debug CGO_ENABLED=0 go run -ldflags "-X main.version=$(VERSION) -s -w" .

fmt: ## Format code
	gofmt -w .
	@command -v goimports >/dev/null 2>&1 && goimports -w . || echo "goimports not found, install with: make dev-tools"
	@command -v golines >/dev/null 2>&1 && golines -w . || echo "golines not found, install with: make dev-tools"
	@echo "Code formatted"

test: ## Run tests
	go test -v ./...

lint: ## Run linter
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not found, install from https://golangci-lint.run/"

dev-tools: ## Install development tools
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/segmentio/golines@latest
	@command -v jq >/dev/null 2>&1 || (echo "Installing jq..." && \
		case "$$(uname -s)" in \
			Linux*) sudo apt-get update && sudo apt-get install -y jq || sudo yum install -y jq ;; \
			Darwin*) brew install jq ;; \
			*) echo "Please install jq manually" ;; \
		esac)
	@echo "Development tools installed"

# Example usage targets
example-list: ## Example: List all indices
	@echo '{"tool": "list_indices", "parameters": {"pattern": "*"}}'

example-mappings: ## Example: Get mappings for logs indices
	@echo '{"tool": "get_index_mappings", "parameters": {"index": "logs-*"}}'

example-search: ## Example: Search for errors in last 24h
	@echo '{"tool": "search", "parameters": {"index": "logs-*", "query": "{\"bool\": {\"must\": [{\"term\": {\"log.level\": \"ERROR\"}}, {\"range\": {\"@timestamp\": {\"gte\": \"now-24h\"}}}]}}", "size": 10}}'
