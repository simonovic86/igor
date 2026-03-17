.PHONY: help bootstrap build build-lab clean test lint vet fmt fmt-check tidy agent agent-heartbeat agent-reconciliation agent-pricewatcher agent-sentinel agent-x402buyer agent-deployer agent-liquidation run-agent demo demo-portable demo-pricewatcher demo-sentinel demo-x402 demo-deployer demo-liquidation gh-check gh-metadata gh-release

.DEFAULT_GOAL := help

# Build configuration
BINARY_NAME := igord
BINARY_DIR := bin
AGENT_DIR := agents/research/example
HEARTBEAT_AGENT_DIR := agents/heartbeat
RECONCILIATION_AGENT_DIR := agents/research/reconciliation
PRICEWATCHER_AGENT_DIR := agents/pricewatcher
SENTINEL_AGENT_DIR := agents/sentinel
X402BUYER_AGENT_DIR := agents/x402buyer
DEPLOYER_AGENT_DIR := agents/deployer
LIQUIDATION_AGENT_DIR := agents/liquidation

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

build: ## Build igord binary (product CLI)
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/igord
	@echo "Built $(BINARY_DIR)/$(BINARY_NAME)"

build-lab: ## Build igord-lab binary (research/P2P CLI)
	@echo "Building igord-lab..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(BINARY_DIR)/igord-lab ./cmd/igord-lab
	@echo "Built $(BINARY_DIR)/igord-lab"

clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BINARY_DIR)
	rm -rf checkpoints
	rm -f agents/research/example/agent.wasm
	rm -f agents/research/example/agent.wasm.checkpoint
	rm -f agents/heartbeat/agent.wasm
	rm -f agents/research/reconciliation/agent.wasm
	rm -f agents/pricewatcher/agent.wasm
	rm -f agents/sentinel/agent.wasm
	rm -f agents/x402buyer/agent.wasm
	rm -f agents/deployer/agent.wasm
	rm -f agents/liquidation/agent.wasm
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
	$(GOVET) ./cmd/... ./internal/... ./pkg/... ./sdk/...

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

run-agent: build-lab agent ## Build and run example agent locally (uses igord-lab)
	@echo "Running agent with default budget (1.0)..."
	./$(BINARY_DIR)/igord-lab --run-agent $(AGENT_DIR)/agent.wasm --budget 1.0

agent-heartbeat: ## Build heartbeat demo agent WASM
	@echo "Building heartbeat agent..."
	@which tinygo > /dev/null || \
		(echo "tinygo not found. See docs/governance/DEVELOPMENT.md for installation" && exit 1)
	cd $(HEARTBEAT_AGENT_DIR) && $(MAKE) build
	@echo "Agent built: $(HEARTBEAT_AGENT_DIR)/agent.wasm"

agent-reconciliation: ## Build reconciliation agent WASM
	@echo "Building reconciliation agent..."
	@which tinygo > /dev/null || \
		(echo "tinygo not found. See docs/governance/DEVELOPMENT.md for installation" && exit 1)
	cd $(RECONCILIATION_AGENT_DIR) && $(MAKE) build
	@echo "Agent built: $(RECONCILIATION_AGENT_DIR)/agent.wasm"

agent-pricewatcher: ## Build price watcher demo agent WASM
	@echo "Building pricewatcher agent..."
	@which tinygo > /dev/null || \
		(echo "tinygo not found. See docs/governance/DEVELOPMENT.md for installation" && exit 1)
	cd $(PRICEWATCHER_AGENT_DIR) && $(MAKE) build
	@echo "Agent built: $(PRICEWATCHER_AGENT_DIR)/agent.wasm"

agent-sentinel: ## Build treasury sentinel demo agent WASM
	@echo "Building sentinel agent..."
	@which tinygo > /dev/null || \
		(echo "tinygo not found. See docs/governance/DEVELOPMENT.md for installation" && exit 1)
	cd $(SENTINEL_AGENT_DIR) && $(MAKE) build
	@echo "Agent built: $(SENTINEL_AGENT_DIR)/agent.wasm"

agent-x402buyer: ## Build x402 buyer demo agent WASM
	@echo "Building x402buyer agent..."
	@which tinygo > /dev/null || \
		(echo "tinygo not found. See docs/governance/DEVELOPMENT.md for installation" && exit 1)
	cd $(X402BUYER_AGENT_DIR) && $(MAKE) build
	@echo "Agent built: $(X402BUYER_AGENT_DIR)/agent.wasm"

agent-deployer: ## Build deployer demo agent WASM
	@echo "Building deployer agent..."
	@which tinygo > /dev/null || \
		(echo "tinygo not found. See docs/governance/DEVELOPMENT.md for installation" && exit 1)
	cd $(DEPLOYER_AGENT_DIR) && $(MAKE) build
	@echo "Agent built: $(DEPLOYER_AGENT_DIR)/agent.wasm"

agent-liquidation: ## Build liquidation watcher demo agent WASM
	@echo "Building liquidation watcher agent..."
	@which tinygo > /dev/null || \
		(echo "tinygo not found. See docs/governance/DEVELOPMENT.md for installation" && exit 1)
	cd $(LIQUIDATION_AGENT_DIR) && $(MAKE) build
	@echo "Agent built: $(LIQUIDATION_AGENT_DIR)/agent.wasm"

demo: build agent-reconciliation ## Build and run reconciliation demo
	@echo "Building demo runner..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(BINARY_DIR)/demo-reconciliation ./agents/research/reconciliation/cmd/demo
	@echo "Running Bridge Reconciliation Demo..."
	./$(BINARY_DIR)/demo-reconciliation --wasm $(RECONCILIATION_AGENT_DIR)/agent.wasm

demo-portable: build agent-heartbeat ## Run the portable agent demo (run, stop, resume, verify)
	@echo "Running Portable Agent Demo..."
	@chmod +x scripts/demo-portable.sh
	@./scripts/demo-portable.sh

demo-pricewatcher: build agent-pricewatcher ## Run the price watcher demo (fetch prices, stop, resume, verify)
	@echo "Running Price Watcher Demo..."
	@chmod +x scripts/demo-pricewatcher.sh
	@./scripts/demo-pricewatcher.sh

demo-sentinel: build agent-sentinel ## Run the treasury sentinel demo (effect lifecycle, crash recovery)
	@echo "Running Treasury Sentinel Demo..."
	@chmod +x scripts/demo-sentinel.sh
	@./scripts/demo-sentinel.sh

demo-x402: build agent-x402buyer ## Run the x402 payment demo (pay for premium data, crash recovery)
	@echo "Building paywall server..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(BINARY_DIR)/paywall ./cmd/paywall
	@echo "Running x402 Payment Demo..."
	@chmod +x scripts/demo-x402.sh
	@./scripts/demo-x402.sh

demo-deployer: build agent-deployer ## Run the deployer demo (pay, deploy, monitor, crash recovery)
	@echo "Building mockcloud server..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(BINARY_DIR)/mockcloud ./cmd/mockcloud
	@echo "Running Deployer Demo..."
	@chmod +x scripts/demo-deployer.sh
	@./scripts/demo-deployer.sh

demo-liquidation: build agent-liquidation ## Run the liquidation watcher demo (gap-aware continuity proof)
	@echo "Running Liquidation Watcher Demo..."
	@chmod +x scripts/demo-liquidation.sh
	@./scripts/demo-liquidation.sh

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
