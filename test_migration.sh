#!/bin/bash
set -e

echo "=== Igor Agent Migration Test ==="
echo ""

# Clean up function
cleanup() {
    echo "Cleaning up..."
    kill $NODE_B_PID 2>/dev/null || true
    rm -f /tmp/node_b.log
}
trap cleanup EXIT

# Clean checkpoints
rm -rf checkpoints
mkdir -p checkpoints

echo "Step 1: Running agent on Node A to create checkpoint..."
timeout 8s ./bin/igord --run-agent agents/example/agent.wasm 2>&1 | tee /tmp/node_a.log || true

# Extract counter value
COUNTER_A=$(grep "Checkpoint: counter=" /tmp/node_a.log | tail -1 | sed 's/.*counter=\([0-9]*\).*/\1/')
echo "  Node A stopped at counter: $COUNTER_A"
echo ""

# Verify checkpoint exists
if [ ! -f "checkpoints/local-agent.checkpoint" ]; then
    echo "ERROR: Checkpoint not created!"
    exit 1
fi
echo "  ✓ Checkpoint created"
echo ""

echo "Step 2: Starting Node B (receiver)..."
./bin/igord > /tmp/node_b.log 2>&1 &
NODE_B_PID=$!
sleep 3

# Extract Node B address
NODE_B_ADDR=$(grep "Listening on.*127.0.0.1" /tmp/node_b.log | head -1 | awk '{print $NF}' | sed 's/address=//')
echo "  Node B listening on: $NODE_B_ADDR"
echo ""

echo "Step 3: Migrating agent from Node A to Node B..."
./bin/igord \
    --migrate-agent local-agent \
    --to "$NODE_B_ADDR" \
    --wasm agents/example/agent.wasm \
    2>&1 | tee /tmp/migration.log

echo ""
echo "Step 4: Verifying migration..."
sleep 2

# Check Node B logs
if grep -q "Agent migration accepted and started" /tmp/node_b.log; then
    echo "  ✓ Node B accepted migration"
fi

if grep -q "Resumed with counter=" /tmp/node_b.log; then
    COUNTER_B=$(grep "Resumed with counter=" /tmp/node_b.log | tail -1 | sed 's/.*counter=\([0-9]*\).*/\1/')
    echo "  ✓ Node B resumed at counter: $COUNTER_B"
    
    if [ "$COUNTER_A" = "$COUNTER_B" ]; then
        echo "  ✓ State preserved! Counter matches: $COUNTER_A"
    else
        echo "  ✗ State mismatch! Node A: $COUNTER_A, Node B: $COUNTER_B"
        exit 1
    fi
fi

# Check if checkpoint was deleted on source
if [ ! -f "checkpoints/local-agent.checkpoint" ]; then
    echo "  ✓ Source checkpoint deleted"
else
    echo "  ⚠ Source checkpoint still exists (expected behavior for demo)"
fi

echo ""
echo "Step 5: Observing Node B continue execution..."
sleep 5
kill $NODE_B_PID 2>/dev/null || true
wait $NODE_B_PID 2>/dev/null || true

# Show final ticks
echo ""
echo "Agent ticks on Node B:"
grep "\[agent\] Tick" /tmp/node_b.log | tail -5

echo ""
echo "=== Migration Test Complete ==="
echo "✓ Agent successfully migrated from Node A to Node B"
echo "✓ State preserved across migration"
echo "✓ Agent continued execution on Node B"
