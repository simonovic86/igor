#!/bin/bash
# Igor pre-commit hook
# Runs quality checks before allowing commits

set -e

echo "🔍 Running pre-commit quality checks..."
echo ""

# Format check
echo "→ Checking code formatting..."
if ! make fmt-check > /dev/null 2>&1; then
    echo "❌ Code formatting check failed"
    echo ""
    echo "Run 'make fmt' to fix formatting, then try committing again."
    exit 1
fi
echo "✓ Formatting OK"

# Vet
echo "→ Running static analysis (go vet)..."
if ! make vet > /dev/null 2>&1; then
    echo "❌ Static analysis failed"
    echo ""
    echo "Fix vet issues before committing."
    make vet
    exit 1
fi
echo "✓ Static analysis OK"

# Lint
echo "→ Running linters (golangci-lint)..."
if ! make lint > /dev/null 2>&1; then
    echo "❌ Linting failed"
    echo ""
    echo "Fix linter issues before committing:"
    make lint
    exit 1
fi
echo "✓ Linting OK"

# Tests
echo "→ Running tests..."
if ! make test > /dev/null 2>&1; then
    echo "❌ Tests failed"
    echo ""
    make test
    exit 1
fi
echo "✓ Tests OK"

echo ""
echo "✅ All pre-commit checks passed"
echo ""
