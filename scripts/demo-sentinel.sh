#!/usr/bin/env bash
# demo-sentinel.sh — Treasury Sentinel: crash recovery with effect tracking.
#
# Demonstrates Igor's core value proposition:
#   Critical automations fail in the middle. Igor is built for the middle.
#
# Flow:
#   1. Run sentinel on "Machine A" — treasury depletes over time
#   2. Agent detects low balance, records a refill intent
#   3. KILL the process mid-flight (simulating crash)
#   4. Resume on "Machine B" from checkpoint
#   5. Agent finds unresolved intent, reconciles instead of blindly retrying
#   6. Verify cryptographic lineage across both machines
#
# The key insight: after crash, the agent doesn't know if the transfer
# completed. It checks. It doesn't guess. That's the whole movie.
set -euo pipefail

IGORD="./bin/igord"
WASM="./agents/sentinel/agent.wasm"
DIR_A="/tmp/igor-demo-sentinel-a"
DIR_B="/tmp/igor-demo-sentinel-b"

# Cleanup from previous runs.
rm -rf "$DIR_A" "$DIR_B"
mkdir -p "$DIR_A" "$DIR_B"

echo ""
echo "=========================================="
echo "  Igor: Treasury Sentinel Demo"
echo "  Effect-safe crash recovery"
echo "=========================================="
echo ""
echo "  Scenario: Treasury balance depletes. Agent refills."
echo "  Crash mid-refill. Resume. Reconcile. No duplicates."
echo ""

# --- Machine A: Run until low balance triggers refill ---
echo "[Machine A] Starting treasury sentinel..."
echo "[Machine A] Watching treasury balance, spending simulated..."
echo ""
$IGORD run --budget 100.0 --checkpoint-dir "$DIR_A" --agent-id sentinel "$WASM" &
PID_A=$!

# Let it run until balance drops and a refill intent is recorded.
# At ~$50-250/tick at 1Hz, $10,000 treasury hits $3,000 threshold in ~28-50 ticks.
# We give it enough time to trigger at least one refill cycle.
sleep 45

echo ""
echo "[Machine A] KILLING process (simulating crash mid-operation)..."
kill -9 "$PID_A" 2>/dev/null || true
wait "$PID_A" 2>/dev/null || true

echo "[Machine A] Process killed. Checkpoint is the last known good state."
echo ""

# --- Copy checkpoint to Machine B ---
CKPT_FILE="$DIR_A/sentinel.checkpoint"
if [ ! -f "$CKPT_FILE" ]; then
    echo "ERROR: Checkpoint file not found at $CKPT_FILE"
    echo "(Agent may not have run long enough to produce a checkpoint.)"
    rm -rf "$DIR_A" "$DIR_B"
    exit 1
fi

echo "=========================================="
echo "  Simulating Transfer to Machine B"
echo "=========================================="
echo ""
echo "[Transfer] Copying checkpoint + identity to Machine B..."
cp "$CKPT_FILE" "$DIR_B/"
if [ -f "$DIR_A/sentinel.identity" ]; then
    cp "$DIR_A/sentinel.identity" "$DIR_B/"
fi
echo "[Transfer] Done. The checkpoint IS the agent."
echo ""

# --- Machine B: Resume and reconcile ---
echo "=========================================="
echo "  Machine B: Resume + Reconcile"
echo "=========================================="
echo ""
echo "[Machine B] Resuming sentinel from checkpoint..."
echo "[Machine B] If there's an in-flight refill, it becomes UNRESOLVED."
echo "[Machine B] The agent will check bridge status, not blindly retry."
echo ""
$IGORD resume --checkpoint "$DIR_B/sentinel.checkpoint" --wasm "$WASM" --checkpoint-dir "$DIR_B" --agent-id sentinel &
PID_B=$!

# Let it reconcile and run a few more cycles.
sleep 20

echo ""
echo "[Machine B] Stopping agent gracefully..."
kill -INT "$PID_B" 2>/dev/null || true
wait "$PID_B" 2>/dev/null || true
echo ""

# --- Verify Lineage ---
VERIFY_DIR="/tmp/igor-demo-sentinel-verify"
rm -rf "$VERIFY_DIR"
mkdir -p "$VERIFY_DIR"

if [ -d "$DIR_A/history/sentinel" ]; then
    cp "$DIR_A/history/sentinel/"*.ckpt "$VERIFY_DIR/" 2>/dev/null || true
fi
if [ -d "$DIR_B/history/sentinel" ]; then
    cp "$DIR_B/history/sentinel/"*.ckpt "$VERIFY_DIR/" 2>/dev/null || true
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
fi

echo ""
echo "=========================================="
echo "  Demo Complete"
echo "=========================================="
echo ""
echo "What you just saw:"
echo "  - A treasury sentinel monitoring balance and executing refills"
echo "  - Process KILLED mid-operation (SIGKILL, not graceful)"
echo "  - Resumed on Machine B from last checkpoint"
echo "  - In-flight intents became UNRESOLVED (the resume rule)"
echo "  - Agent RECONCILED by checking bridge status"
echo "  - No blind retry. No duplicate transfer. Continuity."
echo ""
echo "This is what makes Igor different from retry logic:"
echo "  The agent knows it doesn't know. And it acts accordingly."
echo ""

# Cleanup.
rm -rf "$DIR_A" "$DIR_B" "$VERIFY_DIR"
