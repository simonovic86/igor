#!/usr/bin/env bash
# demo-deployer.sh — Compute Self-Provisioning: agent pays for and deploys itself.
#
# Demonstrates Igor's self-provisioning capabilities:
#   1. Start a mock compute provider (returns 402 without payment)
#   2. Run deployer agent — it checks budget, pays, deploys, monitors
#   3. KILL the process mid-monitoring
#   4. Resume on "Machine B" — reconcile unresolved intents
#   5. Verify cryptographic lineage across both machines
#
# The compute provider is simulated — no real cloud. The agent pays from
# its budget via wallet_pay and uses the receipt to deploy.
set -euo pipefail

IGORD="./bin/igord"
WASM="./agents/deployer/agent.wasm"
MOCKCLOUD="./bin/mockcloud"
DIR_A="/tmp/igor-demo-deployer-a"
DIR_B="/tmp/igor-demo-deployer-b"

# Cleanup from previous runs.
rm -rf "$DIR_A" "$DIR_B"
mkdir -p "$DIR_A" "$DIR_B"

echo ""
echo "=========================================="
echo "  Igor: Compute Self-Provisioning Demo"
echo "  Agent pays for and deploys itself"
echo "=========================================="
echo ""
echo "  Scenario: Agent needs compute infrastructure."
echo "  Provider returns 402. Agent pays. Deploys."
echo "  Monitors until running. Crash. Resume. Reconcile."
echo ""

# --- Start mock cloud server ---
echo "[MockCloud] Starting mock compute provider on :8500..."
$MOCKCLOUD &
PID_CLOUD=$!

# Wait for server to be ready.
for i in $(seq 1 10); do
    if curl -s http://localhost:8500/health > /dev/null 2>&1; then
        break
    fi
    sleep 0.5
done
echo "[MockCloud] Ready."
echo ""

# --- Machine A: Run agent ---
echo "[Machine A] Starting deployer agent..."
echo "[Machine A] Agent will check budget, pay provider, deploy, and monitor."
echo ""
$IGORD run --budget 10.0 --checkpoint-dir "$DIR_A" --agent-id deployer "$WASM" &
PID_A=$!

# Let it run long enough to: check budget → pay → deploy → monitor a few cycles.
sleep 25

echo ""
echo "[Machine A] KILLING process (simulating crash mid-monitoring)..."
kill -9 "$PID_A" 2>/dev/null || true
wait "$PID_A" 2>/dev/null || true

echo "[Machine A] Process killed. Checkpoint is the last known good state."
echo ""

# --- Copy checkpoint to Machine B ---
CKPT_FILE="$DIR_A/deployer.checkpoint"
if [ ! -f "$CKPT_FILE" ]; then
    echo "ERROR: Checkpoint file not found at $CKPT_FILE"
    echo "(Agent may not have run long enough to produce a checkpoint.)"
    kill "$PID_CLOUD" 2>/dev/null || true
    rm -rf "$DIR_A" "$DIR_B"
    exit 1
fi

echo "=========================================="
echo "  Simulating Transfer to Machine B"
echo "=========================================="
echo ""
echo "[Transfer] Copying checkpoint + identity to Machine B..."
cp "$CKPT_FILE" "$DIR_B/"
if [ -f "$DIR_A/deployer.identity" ]; then
    cp "$DIR_A/deployer.identity" "$DIR_B/"
fi
echo "[Transfer] Done. The checkpoint IS the agent."
echo ""

# --- Machine B: Resume and reconcile ---
echo "=========================================="
echo "  Machine B: Resume + Reconcile"
echo "=========================================="
echo ""
echo "[Machine B] Resuming deployer from checkpoint..."
echo "[Machine B] Any in-flight payments or deployments become UNRESOLVED."
echo "[Machine B] The agent will check status, not blindly retry."
echo ""
$IGORD resume --checkpoint "$DIR_B/deployer.checkpoint" --wasm "$WASM" --checkpoint-dir "$DIR_B" --agent-id deployer &
PID_B=$!

# Let it reconcile and continue monitoring.
sleep 15

echo ""
echo "[Machine B] Stopping agent gracefully..."
kill -INT "$PID_B" 2>/dev/null || true
wait "$PID_B" 2>/dev/null || true

# Stop mock cloud server.
kill "$PID_CLOUD" 2>/dev/null || true
wait "$PID_CLOUD" 2>/dev/null || true
echo ""

# --- Verify Lineage ---
VERIFY_DIR="/tmp/igor-demo-deployer-verify"
rm -rf "$VERIFY_DIR"
mkdir -p "$VERIFY_DIR"

if [ -d "$DIR_A/history/deployer" ]; then
    cp "$DIR_A/history/deployer/"*.ckpt "$VERIFY_DIR/" 2>/dev/null || true
fi
if [ -d "$DIR_B/history/deployer" ]; then
    cp "$DIR_B/history/deployer/"*.ckpt "$VERIFY_DIR/" 2>/dev/null || true
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
echo "  - Agent checked its budget against compute costs"
echo "  - Provider returned HTTP 402 Payment Required"
echo "  - Agent paid from its budget via wallet_pay hostcall"
echo "  - Agent deployed itself with the payment receipt"
echo "  - Agent monitored deployment: pending → provisioning → running"
echo "  - Process KILLED mid-monitoring (SIGKILL)"
echo "  - Resumed on Machine B from checkpoint"
echo "  - In-flight intents became UNRESOLVED (the resume rule)"
echo "  - Agent RECONCILED — checked payment and deployment status"
echo "  - No duplicate payments. Budget conserved. Continuity."
echo ""
echo "This is compute self-provisioning:"
echo "  The agent chooses, pays for, and deploys its own infrastructure."
echo "  And survives the process."
echo ""

# Cleanup.
rm -rf "$DIR_A" "$DIR_B" "$VERIFY_DIR"
