# Genesis Tag Annotation

This document contains the annotation message for the v0.1.0-genesis tag.

---

## Tag Creation Command

```bash
git tag -a v0.1.0-genesis -m "$(cat <<'EOF'
Igor v0.1.0-genesis - Phase 2 (Survival) Complete

Experimental decentralized runtime for survivable autonomous agents.

Phase 2 implements:
- WASM sandbox execution (wazero)
- Agent checkpointing and resume
- Peer-to-peer migration (libp2p)
- Runtime budget metering
- Decentralized node network

All 6 success criteria met:
✓ Agent runs on Node A
✓ Agent checkpoints state
✓ Agent migrates to Node B
✓ Agent resumes from checkpoint
✓ Agent pays runtime rent
✓ No centralized coordination

Status: Experimental, research-stage
Not production-ready.

See README.md and docs/runtime/ARCHITECTURE.md for details.
EOF
)"
```

---

## Short Tag Description

```
Igor v0.1.0-genesis - Phase 2 (Survival) Complete
```

---

## Tag Annotation Message

```
Igor v0.1.0-genesis - Phase 2 (Survival) Complete

Experimental decentralized runtime for survivable autonomous agents.

Phase 2 implements:
- WASM sandbox execution (wazero)
- Agent checkpointing and resume
- Peer-to-peer migration (libp2p)
- Runtime budget metering
- Decentralized node network

All 6 success criteria met:
✓ Agent runs on Node A
✓ Agent checkpoints state
✓ Agent migrates to Node B
✓ Agent resumes from checkpoint
✓ Agent pays runtime rent
✓ No centralized coordination

Status: Experimental, research-stage
Not production-ready.

See README.md and docs/runtime/ARCHITECTURE.md for details.
```

---

## Usage

**Create tag:**

```bash
git tag -a v0.1.0-genesis -F docs/archive/GENESIS_TAG_ANNOTATION.md
```

**Verify tag:**

```bash
git tag -l -n20 v0.1.0-genesis
git show v0.1.0-genesis
```

**Push tag (when ready):**

```bash
git push origin v0.1.0-genesis
```

---

## Release Notes

For GitHub release, use:

**Title:** Igor v0.1.0-genesis - Phase 2 (Survival) Complete

**Body:** Combine content from:
- ANNOUNCEMENT.md (introduction)
- This tag annotation (achievements)
- Known limitations from README.md

Mark as: **Pre-release** ✓
