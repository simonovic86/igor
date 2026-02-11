#!/bin/bash
# Igor Developer Environment Bootstrap
# Installs and verifies all required tooling

set -e

echo "🚀 Igor Developer Environment Bootstrap"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

REQUIRED_GO_VERSION="1.25.4"

# Check Go version
echo "→ Verifying Go version..."
if ! command -v go &> /dev/null; then
    echo -e "${RED}✗ Go not found${NC}"
    echo "  Install Go from: https://go.dev/dl/"
    exit 1
fi

CURRENT_GO=$(go version | awk '{print $3}' | sed 's/go//')
if [[ "$CURRENT_GO" != "$REQUIRED_GO_VERSION"* ]]; then
    echo -e "${YELLOW}⚠ Go version mismatch${NC}"
    echo "  Required: $REQUIRED_GO_VERSION"
    echo "  Current:  $CURRENT_GO"
    echo "  Please install Go $REQUIRED_GO_VERSION from https://go.dev/dl/"
    exit 1
fi
echo -e "${GREEN}✓ Go $CURRENT_GO${NC}"

# Download dependencies
echo ""
echo "→ Downloading Go module dependencies..."
go mod download
echo -e "${GREEN}✓ Dependencies downloaded${NC}"

# Install goimports
echo ""
echo "→ Installing goimports..."
go install golang.org/x/tools/cmd/goimports@latest
echo -e "${GREEN}✓ goimports installed${NC}"

# Install golangci-lint
echo ""
echo "→ Installing golangci-lint..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.63.4
echo -e "${GREEN}✓ golangci-lint v1.63.4 installed${NC}"

# Check TinyGo (optional for agent development)
echo ""
echo "→ Checking TinyGo (optional for agent development)..."
if command -v tinygo &> /dev/null; then
    TINYGO_VERSION=$(tinygo version | awk '{print $3}')
    echo -e "${GREEN}✓ TinyGo $TINYGO_VERSION installed${NC}"
else
    echo -e "${YELLOW}⚠ TinyGo not found (optional)${NC}"
    echo "  Install from: https://tinygo.org/getting-started/install/"
    echo "  Required only for building WASM agents"
fi

# Install Git hooks
echo ""
echo "→ Installing Git hooks..."
if [ -f "./scripts/install-hooks.sh" ]; then
    ./scripts/install-hooks.sh
    echo -e "${GREEN}✓ Git hooks installed${NC}"
else
    echo -e "${YELLOW}⚠ Hook install script not found${NC}"
fi

# Verify build
echo ""
echo "→ Verifying build..."
if make build > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Build successful${NC}"
else
    echo -e "${RED}✗ Build failed${NC}"
    echo "  Run 'make build' for details"
    exit 1
fi

# Run quality checks
echo ""
echo "→ Running quality checks..."
if make check > /dev/null 2>&1; then
    echo -e "${GREEN}✓ All quality checks passed${NC}"
else
    echo -e "${RED}✗ Quality checks failed${NC}"
    echo "  Run 'make check' for details"
    exit 1
fi

echo ""
echo -e "${GREEN}✅ Bootstrap complete!${NC}"
echo ""
echo "Ready to develop:"
echo "  make build      # Build igord"
echo "  make agent      # Build example agent"
echo "  make test       # Run tests"
echo "  make check      # Run all quality checks"
echo "  make precommit  # Verify before committing"
echo ""
echo "See docs/DEVELOPMENT.md for complete guide."
