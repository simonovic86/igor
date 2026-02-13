# Budget Model

## Overview

Igor implements a simple metering-based budget model where agents pay for execution time consumed on nodes.

**Formula:** `cost = runtime_seconds × price_per_second`

## Budget Mechanics

### Budget Units

Budget is denominated in **arbitrary currency units**. There is no specific currency (USD, tokens, etc.) in v0.

- Budget: `float64` (e.g., 1.0, 10.5, 0.000001)
- Price: `float64` per second (e.g., 0.001)
- Cost: `float64` (calculated)

### Price Configuration

Each node sets its own price per second in config:

```go
cfg := &config.Config{
    PricePerSecond: 0.001, // Default
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
agent_tick()  // Execute agent code
elapsed := time.Since(start)

durationSeconds := elapsed.Seconds()
cost := durationSeconds × pricePerSecond
budget -= cost
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
LoadAgent(budget=10.0)
    ↓
instance.Budget = 10.0
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
Checkpoint Format:
[0-7]   budget (float64)
[8-15]  pricePerSecond (float64)
[16+]   agent state
```

### 4. Restoration

When agent resumes (restart or migration):

```
Load checkpoint
    ↓
Parse bytes 0-7 → budget
Parse bytes 8-15 → pricePerSecond
    ↓
instance.Budget = budget
instance.PricePerSecond = pricePerSecond
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
    return fmt.Errorf("budget exhausted: %.6f", instance.Budget)
}
```

No tick executes if budget exhausted.

### Post-Execution Deduction

After each tick:

```go
cost := elapsed.Seconds() × pricePerSecond
instance.Budget -= cost
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
Binary Layout:
┌──────────┬────────────────┬─────────────┐
│  Budget  │ PricePerSecond │ Agent State │
│ (8 bytes)│   (8 bytes)    │  (N bytes)  │
└──────────┴────────────────┴─────────────┘
```

**Encoding:** Little-endian IEEE 754 float64

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
