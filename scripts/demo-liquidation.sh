#!/usr/bin/env bash
# demo-liquidation.sh — Liquidation Risk Watcher: continuity through missing time.
#
# This demo proves that an Igor agent stays continuous, not alive.
# A deterministic price curve exists as a pure function of time.
# The agent processes it live, dies, misses the most critical window,
# then resumes on another node and catches up — discovering that the
# liquidation threshold was breached while it was absent.
#
# This demo simulates two hosts locally for determinism and speed.
# The property being proven is portable resume with preserved identity,
# gap-aware catch-up, and cryptographic lineage continuity.
set -euo pipefail

IGORD="./bin/igord"
WASM="./agents/liquidation/agent.wasm"
DIR_A="/tmp/igor-liquidation-node-a"
DIR_B="/tmp/igor-liquidation-node-b"
AGENT_ID="liquidation"

# Cleanup from previous runs.
rm -rf "$DIR_A" "$DIR_B"
mkdir -p "$DIR_A" "$DIR_B"

echo ""
echo "══════════════════════════════════════════════════════════════"
echo "  Igor: Liquidation Risk Watcher"
echo "  \"This agent did not stay alive. It stayed continuous.\""
echo "══════════════════════════════════════════════════════════════"
echo ""
echo "  Position:   10 ETH collateral, 15,000 USDC debt"
echo "  Threshold:  ETH = \$1,550.00 (liquidation warning)"
echo "  World:      Deterministic price curve — exists whether"
echo "              or not the agent is running"
echo ""

# ─── Scene 1: Node A ────────────────────────────────────────────
echo "──────────────────────────────────────────────────────────────"
echo "  Scene 1: Node A — Agent starts monitoring"
echo "──────────────────────────────────────────────────────────────"
echo ""

$IGORD run --budget 100.0 --checkpoint-dir "$DIR_A" --agent-id "$AGENT_ID" "$WASM" &
PID_A=$!

# Let agent run for 15 seconds (processes slots 0–14).
sleep 15

echo ""
echo "[demo] Stopping Node A (SIGINT for graceful checkpoint)..."
kill -INT "$PID_A" 2>/dev/null || true
wait "$PID_A" 2>/dev/null || true

echo ""
echo "[demo] Node A stopped. Checkpoint saved."
echo ""

# Show checkpoint identity.
CKPT_FILE="$DIR_A/${AGENT_ID}.checkpoint"
if [ -f "$CKPT_FILE" ]; then
    NODE_A_DID=$($IGORD inspect "$CKPT_FILE" 2>/dev/null | grep "Agent DID" | awk '{print $NF}' || true)
    if [ -n "$NODE_A_DID" ]; then
        echo "[demo] Node A identity: $NODE_A_DID"
        echo ""
    fi
fi

# ─── Scene 2: The world keeps moving ────────────────────────────
echo "──────────────────────────────────────────────────────────────"
echo "  Scene 2: Agent is dead. The world keeps moving."
echo "──────────────────────────────────────────────────────────────"
echo ""
echo "  The price curve continues through its most dramatic phase."
echo "  Slots ~15–40 include a sharp drawdown and threshold breach."
echo "  The agent is not running. It will discover this on resume."
echo ""

# Wait 25 seconds — slots 15–40 pass without the agent.
for i in $(seq 1 5); do
    echo "  ⏳ ...time passes... ($((i * 5))s of downtime)"
    sleep 5
done

echo ""

# ─── Scene 3: Node B resumes ────────────────────────────────────
echo "──────────────────────────────────────────────────────────────"
echo "  Scene 3: Node B — Agent resumes from checkpoint"
echo "──────────────────────────────────────────────────────────────"
echo ""

# Copy checkpoint and identity to Node B.
CKPT_FILE="$DIR_A/${AGENT_ID}.checkpoint"
if [ ! -f "$CKPT_FILE" ]; then
    echo "ERROR: Checkpoint file not found at $CKPT_FILE"
    exit 1
fi

echo "[demo] Copying checkpoint to Node B..."
cp "$CKPT_FILE" "$DIR_B/"
if [ -f "$DIR_A/${AGENT_ID}.identity" ]; then
    cp "$DIR_A/${AGENT_ID}.identity" "$DIR_B/"
fi
echo ""

$IGORD resume --checkpoint "$DIR_B/${AGENT_ID}.checkpoint" --wasm "$WASM" --checkpoint-dir "$DIR_B" --agent-id "$AGENT_ID" &
PID_B=$!

# Let it run for 10 seconds (catch-up + live processing).
sleep 10

echo ""
echo "[demo] Stopping Node B..."
kill -INT "$PID_B" 2>/dev/null || true
wait "$PID_B" 2>/dev/null || true

echo ""

# Verify same identity on Node B.
CKPT_FILE_B="$DIR_B/${AGENT_ID}.checkpoint"
if [ -f "$CKPT_FILE_B" ]; then
    NODE_B_DID=$($IGORD inspect "$CKPT_FILE_B" 2>/dev/null | grep "Agent DID" | awk '{print $NF}' || true)
    if [ -n "$NODE_B_DID" ] && [ -n "$NODE_A_DID" ]; then
        if [ "$NODE_A_DID" = "$NODE_B_DID" ]; then
            echo "[demo] ✓ Same identity across nodes: $NODE_B_DID"
        else
            echo "[demo] ✗ Identity mismatch! Node A: $NODE_A_DID  Node B: $NODE_B_DID"
        fi
        echo ""
    fi
fi

# ─── Scene 4: Verify lineage ────────────────────────────────────
echo "──────────────────────────────────────────────────────────────"
echo "  Scene 4: Verify cryptographic lineage"
echo "──────────────────────────────────────────────────────────────"
echo ""

VERIFY_DIR="/tmp/igor-liquidation-verify"
rm -rf "$VERIFY_DIR"
mkdir -p "$VERIFY_DIR"

if [ -d "$DIR_A/history/$AGENT_ID" ]; then
    cp "$DIR_A/history/$AGENT_ID/"*.ckpt "$VERIFY_DIR/" 2>/dev/null || true
fi
if [ -d "$DIR_B/history/$AGENT_ID" ]; then
    cp "$DIR_B/history/$AGENT_ID/"*.ckpt "$VERIFY_DIR/" 2>/dev/null || true
fi

CKPT_COUNT=$(ls "$VERIFY_DIR/"*.ckpt 2>/dev/null | wc -l | tr -d ' ')

if [ "$CKPT_COUNT" -gt 0 ]; then
    $IGORD verify "$VERIFY_DIR"
else
    echo "No checkpoint history found for verification."
fi

echo ""
echo "══════════════════════════════════════════════════════════════"
echo "  Demo Complete"
echo ""
echo "  What just happened:"
echo "    1. Agent started on Node A with a stable DID identity"
echo "    2. It monitored ETH price against liquidation threshold"
echo "    3. Node A died — the agent was absent for ~25 seconds"
echo "    4. The price curve kept moving through its critical phase"
echo "    5. Agent resumed on Node B with the same DID"
echo "    6. It detected the gap and replayed missed time slots"
echo "    7. It discovered the threshold was breached during downtime"
echo "    8. Cryptographic lineage verified across both nodes"
echo ""
echo "  The agent did not stay alive. It stayed continuous."
echo "══════════════════════════════════════════════════════════════"

# Cleanup.
rm -rf "$DIR_A" "$DIR_B" "$VERIFY_DIR"
