#!/bin/bash
# Install Git hooks for Igor repository

set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HOOKS_DIR="$REPO_ROOT/.git/hooks"
PRE_COMMIT_SCRIPT="$REPO_ROOT/scripts/pre-commit.sh"

echo "Installing Igor Git hooks..."
echo ""

# Check if pre-commit script exists
if [ ! -f "$PRE_COMMIT_SCRIPT" ]; then
    echo "Error: pre-commit script not found at $PRE_COMMIT_SCRIPT"
    exit 1
fi

# Make pre-commit script executable
chmod +x "$PRE_COMMIT_SCRIPT"

# Create symlink to pre-commit hook
ln -sf "../../scripts/pre-commit.sh" "$HOOKS_DIR/pre-commit"

echo "✓ Pre-commit hook installed at $HOOKS_DIR/pre-commit"
echo ""
echo "The hook will run automatically before each commit to enforce:"
echo "  - Code formatting (gofmt)"
echo "  - Static analysis (go vet)"
echo "  - Linting (golangci-lint)"
echo "  - Tests (go test)"
echo ""
echo "To bypass the hook (not recommended), use: git commit --no-verify"
echo ""
echo "Done!"
