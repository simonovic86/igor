#!/usr/bin/env bash
# demo-pricewatcher.sh — Demonstrate a portable agent that watches crypto prices.
#
# This agent fetches BTC and ETH prices from CoinGecko on every tick,
# tracks all-time high/low, and accumulates observations. When stopped
# and resumed on a different "machine", it continues with its full
# price history intact — same DID, same memory, cryptographically verified.
#
# Flow:
#   1. Run pricewatcher on "Machine A"
#   2. Let it fetch prices for several ticks
#   3. Stop it, copy checkpoint to "Machine B"
#   4. Resume — continuous observation count, same DID
#   5. Verify cryptographic lineage across both machines
set -euo pipefail

IGORD="./bin/igord"
WASM="./agents/pricewatcher/agent.wasm"
DIR_A="/tmp/igor-demo-prices-a"
DIR_B="/tmp/igor-demo-prices-b"

# Cleanup from previous runs.
rm -rf "$DIR_A" "$DIR_B"
mkdir -p "$DIR_A" "$DIR_B"

echo ""
echo "=========================================="
echo "  Igor: Price Watcher Demo"
echo "  Portable agent with real-world memory"
echo "=========================================="
echo ""

# --- Machine A ---
echo "[Machine A] Starting price watcher agent..."
echo "[Machine A] Fetching BTC & ETH prices from CoinGecko..."
echo ""
$IGORD run --budget 100.0 --checkpoint-dir "$DIR_A" --agent-id pricewatcher "$WASM" &
PID_A=$!

# Let it fetch prices for ~15 seconds.
sleep 15

echo ""
echo "[Machine A] Stopping agent (sending SIGINT)..."
kill -INT "$PID_A" 2>/dev/null || true
wait "$PID_A" 2>/dev/null || true

echo ""
echo "[Machine A] Checkpoint saved. Agent's memory preserved."
echo ""

# --- Copy to Machine B ---
CKPT_FILE="$DIR_A/pricewatcher.checkpoint"
if [ ! -f "$CKPT_FILE" ]; then
    echo "ERROR: Checkpoint file not found at $CKPT_FILE"
    exit 1
fi

echo "=========================================="
echo "  Simulating Transfer to Machine B"
echo "=========================================="
echo ""
echo "[Transfer] Copying checkpoint + identity to Machine B..."
cp "$CKPT_FILE" "$DIR_B/"
if [ -f "$DIR_A/pricewatcher.identity" ]; then
    cp "$DIR_A/pricewatcher.identity" "$DIR_B/"
fi
echo "[Transfer] Done. The checkpoint IS the agent."
echo ""

# --- Machine B ---
echo "=========================================="
echo "  Machine B: Resuming Agent"
echo "=========================================="
echo ""
echo "[Machine B] Resuming price watcher from checkpoint..."
echo "[Machine B] Same DID, continuous observation count, all price history intact."
echo ""
$IGORD resume --checkpoint "$DIR_B/pricewatcher.checkpoint" --wasm "$WASM" --checkpoint-dir "$DIR_B" --agent-id pricewatcher &
PID_B=$!

# Let it fetch more prices for ~10 seconds.
sleep 10

echo ""
echo "[Machine B] Stopping agent..."
kill -INT "$PID_B" 2>/dev/null || true
wait "$PID_B" 2>/dev/null || true

echo ""

# --- Verify Lineage ---
VERIFY_DIR="/tmp/igor-demo-prices-verify"
rm -rf "$VERIFY_DIR"
mkdir -p "$VERIFY_DIR"

if [ -d "$DIR_A/history/pricewatcher" ]; then
    cp "$DIR_A/history/pricewatcher/"*.ckpt "$VERIFY_DIR/" 2>/dev/null || true
fi
if [ -d "$DIR_B/history/pricewatcher" ]; then
    cp "$DIR_B/history/pricewatcher/"*.ckpt "$VERIFY_DIR/" 2>/dev/null || true
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
echo ""
echo "What you just saw:"
echo "  - An agent fetching real crypto prices from CoinGecko"
echo "  - Stopped on Machine A, copied to Machine B"
echo "  - Resumed with full price history + same DID identity"
echo "  - Cryptographic lineage verified across both machines"
echo "  - The checkpoint IS the agent. Copy it anywhere."
echo ""

# Cleanup.
rm -rf "$DIR_A" "$DIR_B" "$VERIFY_DIR"
