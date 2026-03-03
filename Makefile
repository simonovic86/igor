.PHONY: help bootstrap build clean test lint vet fmt fmt-check tidy agent run-agent gh-check gh-metadata gh-release

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

bootstrap: ## Install development toolchain and verify environment
	@echo "Running developer environment bootstrap..."
	@./scripts/bootstrap.sh

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

test: ## Run tests (with race detector)
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

lint: ## Run golangci-lint
	@echo "Running linters..."
	@which golangci-lint > /dev/null || \
		(echo "golangci-lint not found. Run: make bootstrap" && exit 1)
	golangci-lint run --timeout=5m

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./cmd/... ./internal/... ./pkg/...

fmt: ## Format code with gofmt and goimports
	@echo "Formatting code..."
	@which $(GOIMPORTS) > /dev/null || \
		(echo "goimports not found. Run: make bootstrap" && exit 1)
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
		(echo "tinygo not found. See docs/governance/DEVELOPMENT.md for installation" && exit 1)
	cd $(AGENT_DIR) && $(MAKE) build
	@echo "Agent built: $(AGENT_DIR)/agent.wasm"

run-agent: build agent ## Build and run example agent locally
	@echo "Running agent with default budget (1.0)..."
	./$(BINARY_DIR)/$(BINARY_NAME) --run-agent $(AGENT_DIR)/agent.wasm --budget 1.0

check: fmt-check vet lint test ## Run all checks (formatting, vet, lint, tests)
	@echo "All checks passed"

precommit: check ## Alias for check (use before committing)
	@echo "Pre-commit checks complete"

all: clean build test check ## Clean, build, test, and run all checks
	@echo "Build and checks complete"

gh-check: ## Verify GitHub CLI authentication
	@./scripts/verify-gh-auth.sh

gh-metadata: ## Configure GitHub repository metadata (requires gh auth)
	@./scripts/configure-repo-metadata.sh

gh-release: ## Prepare GitHub release draft (usage: make gh-release VERSION=v0.1.0)
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION required"; \
		echo "Usage: make gh-release VERSION=v0.1.0-genesis"; \
		exit 1; \
	fi
	@./scripts/prepare-release.sh $(VERSION)
