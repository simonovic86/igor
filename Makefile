.PHONY: help build clean test lint vet fmt fmt-check tidy agent run-agent

.DEFAULT_GOAL := help

# Build configuration
BINARY_NAME := igord
BINARY_DIR := bin
AGENT_DIR := agents/example

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOVET := $(GOCMD) vet
GOFMT := gofmt
GOIMPORTS := goimports

help: ## Show this help message
	@echo 'Igor v0 Development Commands'
	@echo ''
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build igord binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/igord
	@echo "Built $(BINARY_DIR)/$(BINARY_NAME)"

clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BINARY_DIR)
	rm -rf checkpoints
	rm -f agents/example/agent.wasm
	rm -f agents/example/agent.wasm.checkpoint
	@echo "Clean complete"

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

lint: ## Run golangci-lint
	@echo "Running linters..."
	@which golangci-lint > /dev/null || \
		(echo "golangci-lint not found. Install: brew install golangci-lint" && exit 1)
	golangci-lint run ./...

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./cmd/... ./internal/... ./pkg/...

fmt: ## Format code with gofmt and goimports
	@echo "Formatting code..."
	@which $(GOIMPORTS) > /dev/null || \
		(echo "goimports not found. Install: go install golang.org/x/tools/cmd/goimports@latest" && exit 1)
	@find . -name '*.go' -not -path './vendor/*' -exec $(GOFMT) -w -s {} \;
	@find . -name '*.go' -not -path './vendor/*' -exec $(GOIMPORTS) -w {} \;
	@echo "Formatting complete"

fmt-check: ## Check if code is formatted correctly
	@echo "Checking code formatting..."
	@FMT_FILES=$$($(GOFMT) -l $$(find . -name '*.go' -not -path './vendor/*')); \
	if [ -n "$$FMT_FILES" ]; then \
		echo "The following files are not formatted:"; \
		echo "$$FMT_FILES"; \
		exit 1; \
	fi
	@echo "All files are properly formatted"

tidy: ## Tidy go.mod and go.sum
	@echo "Tidying go modules..."
	$(GOMOD) tidy
	@echo "Modules tidied"

agent: ## Build example agent WASM
	@echo "Building example agent..."
	@which tinygo > /dev/null || \
		(echo "tinygo not found. Install: brew tap tinygo-org/tools && brew install tinygo" && exit 1)
	cd $(AGENT_DIR) && $(MAKE) build
	@echo "Agent built: $(AGENT_DIR)/agent.wasm"

run-agent: build agent ## Build and run example agent locally
	@echo "Running agent with default budget (1.0)..."
	./$(BINARY_DIR)/$(BINARY_NAME) --run-agent $(AGENT_DIR)/agent.wasm --budget 1.0

check: fmt-check vet lint ## Run all checks (formatting, vet, lint)
	@echo "All checks passed"

all: clean build test check ## Clean, build, test, and run all checks
	@echo "Build and checks complete"
