# Budget Model

## Overview

Igor implements a simple metering-based budget model where agents pay for execution time consumed on nodes.

**Formula:** `cost = runtime_seconds × price_per_second`

## Budget Mechanics

### Budget Units

Budget is denominated in **arbitrary currency units**. There is no specific currency (USD, tokens, etc.) in v0.

- Budget: `int64` microcents (1 currency unit = 1,000,000 microcents)
- Price: `int64` microcents per second (e.g., 1000 = 0.001 units/sec)
- Cost: `int64` microcents (calculated with integer arithmetic, no float precision loss)

### Price Configuration

Each node sets its own price per second in config:

```go
cfg := &config.Config{
    PricePerSecond: 1000, // 0.001 units/sec = 1000 microcents/sec
}
```

**Default:** 0.001 units per second

Nodes are free to set any price. Agents choose nodes based on price (in future phases).

### Initial Budget

Agents receive initial budget via CLI flag:

```bash
./bin/igord --run-agent agent.wasm --budget 10.0
```

**Default:** 1.0 units

Budget is provided by the user launching the agent. In production systems, agents would manage their own budgets cryptographically.

## Metering Implementation

### Per-Tick Metering

Every tick execution is metered:

```go
start := time.Now()
agent_tick()
elapsed := time.Since(start)

costMicrocents := elapsed.Microseconds() * pricePerSecond / budget.MicrocentScale
budget -= costMicrocents
```

**Precision:** Nanosecond-level timing (Go's `time.Now()`)

**Granularity:** Per-tick basis (every ~1 second)

### Cost Calculation Example

Given:
- Tick duration: 0.5ms = 0.0005 seconds
- Price: 0.001 per second

Calculation:
```
cost = 0.0005 × 0.001 = 0.0000005 units
```

At this rate, an agent with 1.0 budget can run for:
```
1.0 ÷ 0.0000005 = 2,000,000 ticks
2,000,000 seconds = ~23 days of continuous execution
```

## Budget Lifecycle

### 1. Initialization

```
CLI --budget 10.0
    ↓
budget.FromFloat(10.0) → 10000000 microcents
    ↓
instance.Budget = 10000000
```

### 2. Execution

```
Tick 1: budget 10.0 → cost 0.000001 → budget 9.999999
Tick 2: budget 9.999999 → cost 0.000001 → budget 9.999998
...
```

### 3. Checkpointing

Budget is persisted in checkpoint:

```
Checkpoint Format (v0x04, 209-byte header):
[0]       version (0x04)
[1-8]     budget (int64 microcents)
[9-16]    pricePerSecond (int64 microcents)
[17-24]   tickNumber (uint64)
[25-56]   wasmHash (SHA-256)
[57-64]   majorVersion (uint64)
[65-72]   leaseGeneration (uint64)
[73-80]   leaseExpiry (int64)
[81-112]  prevHash (SHA-256 of previous checkpoint)
[113-144] agentPubKey (Ed25519 public key)
[145-208] signature (Ed25519)
[209+]    agent state
```

### 4. Restoration

When agent resumes (restart or migration):

```
Load checkpoint
    ↓
ParseCheckpointHeader(checkpoint)
    → budget (int64 microcents)
    → pricePerSecond (int64 microcents)
    → tickNumber (uint64)
    → wasmHash ([32]byte)
    → agentState ([]byte)
Verify wasmHash matches loaded WASM binary
    ↓
instance.Budget = budget
instance.PricePerSecond = pricePerSecond
instance.TickNumber = tickNumber
```

Budget continues from previous value.

### 5. Migration

Budget transfers with agent:

```
Source Node:
  checkpoint → budget = 5.5

Migration Package:
  Budget: 5.5
  PricePerSecond: 0.001

Target Node:
  instance.Budget = 5.5
  continues execution
```

### 6. Exhaustion

When budget ≤ 0:

```
if budget <= 0 {
    log("budget exhausted")
    checkpoint()
    terminate()
}
```

Agent stops execution but checkpoint is saved with budget=0.

## Budget Enforcement

### Pre-Execution Check

Before each tick:

```go
if instance.Budget <= 0 {
    return fmt.Errorf("budget exhausted: %s", budget.Format(instance.Budget))
}
```

No tick executes if budget exhausted.

### Post-Execution Deduction

After each tick:

```go
costMicrocents := elapsed.Microseconds() * instance.PricePerSecond / budget.MicrocentScale
instance.Budget -= costMicrocents
```

Budget decreases monotonically.

### Graceful Termination

On budget exhaustion:

1. Detect `budget <= 0` after tick
2. Log termination reason
3. Call `agent_checkpoint()`
4. Save checkpoint to storage
5. Terminate agent instance
6. Return error to tick loop

The checkpoint preserves the exhausted state, allowing inspection or potential refunding (outside v0 scope).

## Budget Persistence

### Storage Format

Checkpoints include budget as metadata:

```
Binary Layout (209-byte header, v0x04):
┌─────────┬──────────┬────────────────┬────────────┬──────────┬──────────────┬─────────────────┬─────────────┬──────────┬─────────────┬───────────┬─────────────┐
│ Version │  Budget  │ PricePerSecond │ TickNumber │ WASMHash │ MajorVersion │ LeaseGeneration │ LeaseExpiry │ PrevHash │ AgentPubKey │ Signature │ Agent State │
│ (1 byte)│ (8 bytes)│   (8 bytes)    │ (8 bytes)  │(32 bytes)│  (8 bytes)   │   (8 bytes)     │  (8 bytes)  │(32 bytes)│  (32 bytes) │ (64 bytes)│  (N bytes)  │
└─────────┴──────────┴────────────────┴────────────┴──────────┴──────────────┴─────────────────┴─────────────┴──────────┴─────────────┴───────────┴─────────────┘

Encoding: Little-endian integers (int64 microcents for budget/price, uint64 for tick)
```

### Survival Scenarios

**Local Restart:**
```
Run 1: budget 10.0 → 9.5 (checkpoint saved)
Shutdown
Run 2: budget 9.5 (restored) → 9.0 → ...
```

**Migration:**
```
Node A: budget 5.0 → 4.5 (checkpoint)
Migrate to Node B
Node B: budget 4.5 (restored) → 4.0 → ...
```

**Budget survives:**
- Process restarts
- Node crashes
- Migrations
- Checkpoint/resume cycles

## Logging

### Budget Tracking

Every tick logs:

```
Tick completed
  agent_id=local-agent
  duration_ms=0
  cost=0.000001
  budget_remaining=9.999999
```

### Budget Exhaustion

```
Agent budget exhausted, terminating
  agent_id=local-agent
  reason=budget_exhausted
```

### Budget Restoration

```
Budget restored from checkpoint
  agent_id=local-agent
  budget=9.999999
  price_per_second=0.001000
```

## Economic Implications

### Agent Behavior

Agents with limited budgets will:
- Seek cheaper nodes (future: price comparison)
- Minimize computation (optimize tick efficiency)
- Migrate strategically (balance price vs. latency)

### Node Behavior

Nodes can:
- Set competitive prices to attract agents
- Adjust prices based on load
- Refuse expensive agents (future: capability enforcement)

### Market Dynamics

In a functioning Igor ecosystem:
- Price discovery through competition
- Agents vote with their budget
- Nodes compete on price and reliability
- No centralized price-setting

**Note:** Market dynamics are not implemented in v0. This describes intended behavior for future phases.

## Current Limitations

### MVP Simplifications

1. **No cryptographic receipts**
   - Budget is trusted accounting
   - No proof of payment
   - No dispute resolution

2. **No refunds**
   - Budget only decreases
   - No credit for unused time
   - No rebates for poor performance

3. **Single price model**
   - Flat rate per second
   - No tiered pricing
   - No priority queues

4. **No payment negotiation**
   - Agents accept node price
   - No bidding
   - No auctions

### Security Considerations

**Budget is not secure in v0:**

- Stored in plaintext
- No cryptographic proofs
- Nodes could lie about consumption
- No external audit trail

Future phases will add:
- Cryptographic receipts
- Signed payment proofs
- Third-party verification
- On-chain settlement (optional)

## Budget Examples

### Example 1: Long-Running Agent

```
Initial budget: 100.0
Price: 0.001 per second
Tick duration: 1ms = 0.001 seconds

Cost per tick: 0.001 × 0.001 = 0.000001
Ticks per budget: 100.0 ÷ 0.000001 = 100,000,000 ticks

At 1 tick/second: ~27,777 hours = ~1,157 days
```

### Example 2: Expensive Node

```
Initial budget: 1.0
Price: 1.0 per second (1000× default)
Tick duration: 1ms = 0.001 seconds

Cost per tick: 0.001 × 1.0 = 0.001
Ticks per budget: 1.0 ÷ 0.001 = 1,000 ticks

At 1 tick/second: ~16 minutes
```

### Example 3: Micro-Budget Agent

```
Initial budget: 0.001
Price: 0.001 per second
Tick duration: 1ms

Cost per tick: 0.000001
Ticks per budget: 0.001 ÷ 0.000001 = 1,000 ticks

At 1 tick/second: ~16 minutes
```

## Testing Budget Behavior

### Test 1: Normal Execution

```bash
./bin/igord --run-agent agent.wasm --budget 1.0
```

**Expected:**
- Agent runs normally
- Budget decreases gradually
- Checkpoints include budget
- Agent survives restart

### Test 2: Budget Exhaustion

```bash
./bin/igord --run-agent agent.wasm --budget 0.000001
```

**Expected:**
- Agent runs 1-2 ticks
- Budget reaches 0
- Agent terminates gracefully
- Final checkpoint saved

### Test 3: Budget Through Migration

```bash
# Node A: Run agent with budget
./bin/igord --run-agent agent.wasm --budget 5.0

# Node B: Receive migration
./bin/igord  # listening

# Migrate (from Node A)
./bin/igord --migrate-agent local-agent --to <node_b> --wasm agent.wasm
```

**Expected:**
- Budget transfers to Node B
- Node B continues metering
- Total budget preserved
- No budget created/destroyed

## Future Enhancements

Potential improvements for future phases:

- **Dynamic pricing** - Nodes adjust price based on load
- **Budget pools** - Shared budget across agent instances
- **Credit system** - Prepaid execution credits
- **Payment channels** - Off-chain micropayments
- **Receipt signing** - Cryptographic proof of payment
- **Dispute resolution** - Third-party arbitration

These are **not implemented in v0** and are listed only as possibilities.
