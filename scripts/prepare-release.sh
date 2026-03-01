#!/bin/bash
# Prepare GitHub release draft
# Usage: ./scripts/prepare-release.sh v0.1.0-genesis

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <version-tag>"
    echo "Example: $0 v0.1.0-genesis"
    exit 1
fi

VERSION="$1"

echo "📦 Preparing GitHub Release: $VERSION"
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

# Check if tag exists
if ! git rev-parse "$VERSION" > /dev/null 2>&1; then
    echo "Error: Tag $VERSION does not exist"
    echo "Create tag first:"
    echo "  git tag -a $VERSION -F docs/archive/GENESIS_TAG_ANNOTATION.md"
    exit 1
fi

# Prepare release notes
RELEASE_NOTES_FILE=$(mktemp)

cat > "$RELEASE_NOTES_FILE" <<'EOF'
Experimental decentralized runtime for survivable autonomous agents.

## Phase 1 Complete

Igor v0.1.0-genesis implements:
- **WASM sandbox execution** (wazero)
- **Agent checkpointing and resume** (atomic persistence)
- **Peer-to-peer migration** (libp2p streams)
- **Runtime budget metering** (cost = duration × price)
- **Decentralized node network** (no coordination)

All 6 success criteria met:
- ✓ Agent runs on Node A
- ✓ Agent checkpoints state
- ✓ Agent migrates to Node B
- ✓ Agent resumes from checkpoint
- ✓ Agent pays runtime rent
- ✓ No centralized coordination

## Status: Experimental

**Not production-ready.**

Known limitations:
- Trusted runtime accounting (no cryptographic receipts)
- Single-hop migration (no routing)
- Local filesystem storage (no distribution)
- Limited security model

See [SECURITY.md](./SECURITY.md) for complete threat model.

## Quick Start

```bash
make build
./bin/igord --run-agent agents/example/agent.wasm --budget 10.0
```

## Documentation

- [README.md](./README.md) - Overview and quick start
- [docs/runtime/ARCHITECTURE.md](./docs/runtime/ARCHITECTURE.md) - Technical details
- [docs/philosophy/VISION.md](./docs/philosophy/VISION.md) - Why Igor exists
- [docs/philosophy/OVERVIEW.md](./docs/philosophy/OVERVIEW.md) - Design philosophy

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

---

**Igor is experimental infrastructure for autonomous software survival research.**
EOF

# Create draft release
echo "→ Creating draft release..."
gh release create "$VERSION" \
    --draft \
    --title "Igor $VERSION - Phase 1 (Survival) Complete" \
    --notes-file "$RELEASE_NOTES_FILE" \
    --prerelease

rm "$RELEASE_NOTES_FILE"

echo ""
echo "✅ Draft release created: $VERSION"
echo ""
echo "Review and publish:"
echo "  gh release view $VERSION --web"
echo ""
echo "Or edit draft:"
echo "  gh release edit $VERSION"
