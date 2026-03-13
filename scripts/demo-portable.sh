#!/usr/bin/env bash
# demo-portable.sh — Demonstrate Igor's portable, immortal agent.
#
# Flow:
#   1. Run heartbeat agent on "Machine A" (local dir A)
#   2. Let it tick for a few seconds, then stop it
#   3. Copy checkpoint to "Machine B" (local dir B)
#   4. Resume from checkpoint — same DID, continuous tick count
#   5. Verify cryptographic lineage across both "machines"
set -euo pipefail

IGORD="./bin/igord"
WASM="./agents/heartbeat/agent.wasm"
DIR_A="/tmp/igor-demo-machine-a"
DIR_B="/tmp/igor-demo-machine-b"

# Cleanup from previous runs.
rm -rf "$DIR_A" "$DIR_B"
mkdir -p "$DIR_A" "$DIR_B"

echo ""
echo "=========================================="
echo "  Igor: Portable Immortal Agent Demo"
echo "=========================================="
echo ""

# --- Machine A ---
echo "[Machine A] Starting heartbeat agent..."
$IGORD run --budget 100.0 --checkpoint-dir "$DIR_A" --agent-id heartbeat "$WASM" &
PID_A=$!

# Let it tick for 8 seconds.
sleep 8

echo ""
echo "[Machine A] Stopping agent (sending SIGINT)..."
kill -INT "$PID_A" 2>/dev/null || true
wait "$PID_A" 2>/dev/null || true

echo ""
echo "[Machine A] Checkpoint saved. Contents:"
ls -la "$DIR_A/"
echo ""

# --- Copy to Machine B ---
CKPT_FILE="$DIR_A/heartbeat.checkpoint"
if [ ! -f "$CKPT_FILE" ]; then
    echo "ERROR: Checkpoint file not found at $CKPT_FILE"
    exit 1
fi

echo "[Transfer] Copying checkpoint to Machine B..."
cp "$CKPT_FILE" "$DIR_B/"
# Also copy identity so the agent keeps its DID.
if [ -f "$DIR_A/heartbeat.identity" ]; then
    cp "$DIR_A/heartbeat.identity" "$DIR_B/"
fi
echo ""

# --- Machine B ---
echo "[Machine B] Resuming agent from checkpoint..."
$IGORD resume --checkpoint "$DIR_B/heartbeat.checkpoint" --wasm "$WASM" --checkpoint-dir "$DIR_B" --agent-id heartbeat &
PID_B=$!

# Let it tick for 6 seconds.
sleep 6

echo ""
echo "[Machine B] Stopping agent..."
kill -INT "$PID_B" 2>/dev/null || true
wait "$PID_B" 2>/dev/null || true

echo ""

# --- Verify Lineage ---
# Merge history from both machines for full chain verification.
VERIFY_DIR="/tmp/igor-demo-verify"
rm -rf "$VERIFY_DIR"
mkdir -p "$VERIFY_DIR"

if [ -d "$DIR_A/history/heartbeat" ]; then
    cp "$DIR_A/history/heartbeat/"*.ckpt "$VERIFY_DIR/" 2>/dev/null || true
fi
if [ -d "$DIR_B/history/heartbeat" ]; then
    cp "$DIR_B/history/heartbeat/"*.ckpt "$VERIFY_DIR/" 2>/dev/null || true
fi

CKPT_COUNT=$(ls "$VERIFY_DIR/"*.ckpt 2>/dev/null | wc -l | tr -d ' ')
echo "=========================================="
echo "  Lineage Verification ($CKPT_COUNT checkpoints)"
echo "=========================================="
echo ""

if [ "$CKPT_COUNT" -gt 0 ]; then
    $IGORD verify "$VERIFY_DIR"
else
    echo "No checkpoint history found for verification."
    echo "(Agent may not have run long enough to produce history.)"
fi

echo ""
echo "=========================================="
echo "  Demo Complete"
echo "=========================================="

# Cleanup.
rm -rf "$DIR_A" "$DIR_B" "$VERIFY_DIR"
