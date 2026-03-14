#!/usr/bin/env bash
# demo-x402.sh — x402 Payment Protocol: agent pays for premium data.
#
# Demonstrates Igor's payment capabilities:
#   1. Start a mock paywall server (returns 402 without payment)
#   2. Run x402buyer agent — it encounters 402, pays, gets premium data
#   3. KILL the process mid-payment cycle
#   4. Resume on "Machine B" — reconcile unresolved payments
#   5. Verify cryptographic lineage across both machines
#
# The paywall is simulated — no real blockchain. The agent pays from its
# budget via the wallet_pay hostcall and receives a signed receipt.
set -euo pipefail

IGORD="./bin/igord"
WASM="./agents/x402buyer/agent.wasm"
PAYWALL="./bin/paywall"
DIR_A="/tmp/igor-demo-x402-a"
DIR_B="/tmp/igor-demo-x402-b"

# Cleanup from previous runs.
rm -rf "$DIR_A" "$DIR_B"
mkdir -p "$DIR_A" "$DIR_B"

echo ""
echo "=========================================="
echo "  Igor: x402 Payment Protocol Demo"
echo "  Agent pays for premium data"
echo "=========================================="
echo ""
echo "  Scenario: Agent fetches premium market data."
echo "  Paywall returns 402. Agent pays. Gets data."
echo "  Crash mid-payment. Resume. Reconcile. No duplicates."
echo ""

# --- Start paywall server ---
echo "[Paywall] Starting mock x402 paywall server on :8402..."
$PAYWALL &
PID_PAYWALL=$!

# Wait for paywall to be ready.
for i in $(seq 1 10); do
    if curl -s http://localhost:8402/health > /dev/null 2>&1; then
        break
    fi
    sleep 0.5
done
echo "[Paywall] Ready."
echo ""

# --- Machine A: Run agent ---
echo "[Machine A] Starting x402buyer agent..."
echo "[Machine A] Agent will encounter 402, pay, and fetch premium data."
echo ""
$IGORD run --budget 100.0 --checkpoint-dir "$DIR_A" --agent-id x402buyer "$WASM" &
PID_A=$!

# Let it run long enough to complete a few payment cycles.
sleep 20

echo ""
echo "[Machine A] KILLING process (simulating crash mid-payment)..."
kill -9 "$PID_A" 2>/dev/null || true
wait "$PID_A" 2>/dev/null || true

echo "[Machine A] Process killed. Checkpoint is the last known good state."
echo ""

# --- Copy checkpoint to Machine B ---
CKPT_FILE="$DIR_A/x402buyer.checkpoint"
if [ ! -f "$CKPT_FILE" ]; then
    echo "ERROR: Checkpoint file not found at $CKPT_FILE"
    echo "(Agent may not have run long enough to produce a checkpoint.)"
    kill "$PID_PAYWALL" 2>/dev/null || true
    rm -rf "$DIR_A" "$DIR_B"
    exit 1
fi

echo "=========================================="
echo "  Simulating Transfer to Machine B"
echo "=========================================="
echo ""
echo "[Transfer] Copying checkpoint + identity to Machine B..."
cp "$CKPT_FILE" "$DIR_B/"
if [ -f "$DIR_A/x402buyer.identity" ]; then
    cp "$DIR_A/x402buyer.identity" "$DIR_B/"
fi
echo "[Transfer] Done. The checkpoint IS the agent."
echo ""

# --- Machine B: Resume and reconcile ---
echo "=========================================="
echo "  Machine B: Resume + Reconcile"
echo "=========================================="
echo ""
echo "[Machine B] Resuming x402buyer from checkpoint..."
echo "[Machine B] If there's an in-flight payment, it becomes UNRESOLVED."
echo "[Machine B] The agent will check payment status, not blindly retry."
echo ""
$IGORD resume --checkpoint "$DIR_B/x402buyer.checkpoint" --wasm "$WASM" --checkpoint-dir "$DIR_B" --agent-id x402buyer &
PID_B=$!

# Let it reconcile and run a few more payment cycles.
sleep 15

echo ""
echo "[Machine B] Stopping agent gracefully..."
kill -INT "$PID_B" 2>/dev/null || true
wait "$PID_B" 2>/dev/null || true

# Stop paywall server.
kill "$PID_PAYWALL" 2>/dev/null || true
wait "$PID_PAYWALL" 2>/dev/null || true
echo ""

# --- Verify Lineage ---
VERIFY_DIR="/tmp/igor-demo-x402-verify"
rm -rf "$VERIFY_DIR"
mkdir -p "$VERIFY_DIR"

if [ -d "$DIR_A/history/x402buyer" ]; then
    cp "$DIR_A/history/x402buyer/"*.ckpt "$VERIFY_DIR/" 2>/dev/null || true
fi
if [ -d "$DIR_B/history/x402buyer" ]; then
    cp "$DIR_B/history/x402buyer/"*.ckpt "$VERIFY_DIR/" 2>/dev/null || true
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
echo "  - Agent requested premium data from a paywall"
echo "  - Server returned HTTP 402 Payment Required"
echo "  - Agent paid from its budget via wallet_pay hostcall"
echo "  - Agent received signed payment receipt"
echo "  - Agent retried with receipt, got premium data"
echo "  - Process KILLED mid-payment (SIGKILL)"
echo "  - Resumed on Machine B from checkpoint"
echo "  - In-flight payments became UNRESOLVED (the resume rule)"
echo "  - Agent RECONCILED — checked if payment settled"
echo "  - No duplicate payment. Budget conserved. Continuity."
echo ""
echo "This is x402 for autonomous agents:"
echo "  The agent pays for what it needs. And survives the paying."
echo ""

# Cleanup.
rm -rf "$DIR_A" "$DIR_B" "$VERIFY_DIR"
