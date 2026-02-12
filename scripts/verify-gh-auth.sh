#!/bin/bash
# Verify GitHub CLI authentication
# Safe to run in CI (exits successfully if gh not available)

set -e

# Check if gh is installed
if ! command -v gh &> /dev/null; then
    echo "GitHub CLI (gh) not installed"
    echo "Install from: https://cli.github.com/"
    echo ""
    echo "This is optional for local development."
    echo "Required only for repository metadata management."
    exit 0  # Not a failure
fi

# Check if authenticated
if gh auth status > /dev/null 2>&1; then
    echo "✓ GitHub CLI authenticated"
    gh auth status
else
    echo "⚠ GitHub CLI not authenticated"
    echo ""
    echo "Authenticate with:"
    echo "  gh auth login"
    echo ""
    echo "This is required for repository management tasks."
    echo "Not required for local development (build, test, lint)."
    exit 0  # Not a failure
fi
