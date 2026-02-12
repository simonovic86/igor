#!/bin/bash
# Configure GitHub repository metadata
# Must be run after repository is pushed to GitHub

set -e

echo "🔧 Configuring Igor GitHub Repository Metadata"
echo ""

# Verify gh is available and authenticated
if ! command -v gh &> /dev/null; then
    echo "Error: GitHub CLI (gh) not installed"
    echo "Install from: https://cli.github.com/"
    exit 1
fi

if ! gh auth status > /dev/null 2>&1; then
    echo "Error: GitHub CLI not authenticated"
    echo "Run: gh auth login"
    exit 1
fi

# Set repository description
echo "→ Setting repository description..."
gh repo edit \
    --description "Runtime for survivable autonomous software agents using WASM, migration, and runtime economics."
echo "✓ Description set"

# Add repository topics
echo ""
echo "→ Adding repository topics..."
TOPICS=(
    "autonomous-agents"
    "distributed-systems"
    "wasm-runtime"
    "libp2p"
    "runtime-economics"
    "survivable-software"
    "agent-infrastructure"
    "peer-to-peer"
    "systems-research"
    "go"
)

for topic in "${TOPICS[@]}"; do
    gh repo edit --add-topic "$topic" > /dev/null 2>&1 || true
done
echo "✓ Topics added"

# Verify configuration
echo ""
echo "→ Verifying configuration..."
gh repo view --json description,repositoryTopics

echo ""
echo "✅ Repository metadata configured successfully"
echo ""
echo "View repository:"
echo "  gh repo view --web"
