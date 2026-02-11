# System Invariants

## Overview

Igor v0 maintains strict invariants to ensure correct agent survival and migration. The system is designed to **fail loudly** when invariants are violated.

## Critical Invariants

### I1: Single Active Instance

**Invariant:** At most one active instance of any agent exists at any time.

**Why:**
- Prevents split-brain scenarios
- Ensures state consistency
- Avoids budget double-spending

**Enforcement:**
- Source waits for target confirmation before terminating
- Target starts before sending confirmation
- Atomic checkpoint operations
- Migration is synchronous

**Violation consequences:**
- Multiple instances could diverge
- Budget could be spent twice
- State could fork

**Detected by:**
- Log analysis (two nodes reporting same agent)
- Budget tracking (spending exceeds available)

**Status:** ✅ Enforced

---

### I2: Budget Conservation

**Invariant:** Budget is never created or destroyed, only transferred.

**Why:**
- Prevents inflation
- Ensures fair payment
- Enables economic model

**Enforcement:**
- Budget loaded from checkpoint
- Budget transferred in AgentPackage
- Budget saved in checkpoint
- No refunds or credits

**Violation consequences:**
- Economic model breaks
- Unlimited execution
- Unfair node compensation

**Detected by:**
- Sum of all agent budgets remains constant
- Budget cannot increase except via external funding

**Status:** ✅ Enforced

---

### I3: State Persistence

**Invariant:** Agent state survives any shutdown or migration.

**Why:**
- Agents must survive infrastructure churn
- Execution must be resumable
- No data loss

**Enforcement:**
- Checkpoint before shutdown
- Checkpoint before migration
- Atomic checkpoint writes
- Resume from checkpoint on restart

**Violation consequences:**
- Agent loses progress
- State corruption
- Execution cannot resume

**Detected by:**
- Missing checkpoints after shutdown
- Resume failures
- State inconsistencies

**Status:** ✅ Enforced

---

### I4: Atomic Checkpoints

**Invariant:** Checkpoints are never partial. They are fully written or not at all.

**Why:**
- Prevents state corruption
- Ensures consistency
- Enables safe recovery

**Enforcement:**
- Write to temp file
- fsync to disk
- Atomic rename
- No partial writes visible

**Violation consequences:**
- Corrupted checkpoints
- Agent cannot resume
- Data loss

**Detected by:**
- Checkpoint read errors
- Invalid format errors
- Missing metadata

**Status:** ✅ Enforced (FSProvider)

---

### I5: Tick Determinism

**Invariant:** Given same state and input, tick produces same output.

**Why:**
- Predictable behavior
- Reproducible execution
- Debugging and testing

**Enforcement:**
- Agent responsibility (not enforced by runtime)
- No random without seeded RNG
- No external dependencies

**Violation consequences:**
- Non-reproducible bugs
- Migration inconsistencies
- Difficult debugging

**Detected by:**
- Manual testing
- Comparing execution across runs

**Status:** ⚠️ Not enforced (agent contract)

---

### I6: Tick Short Duration

**Invariant:** Each tick completes within 100ms.

**Why:**
- Responsive system
- Fair scheduling (future: multiple agents)
- Migration responsiveness

**Enforcement:**
- Context timeout on tick
- Error if exceeded
- Agent terminated on timeout

**Violation consequences:**
- Slow agent execution
- Poor migration experience
- Resource hogging

**Detected by:**
- Timeout errors
- Agent termination logs

**Status:** ✅ Enforced

---

### I7: Budget Monotonicity

**Invariant:** Agent budget never increases during execution.

**Why:**
- Reflects resource consumption
- No free execution
- Economic model consistency

**Enforcement:**
- Budget only decremented
- Cost always positive (or zero)
- No refund mechanism

**Violation consequences:**
- Perpetual execution
- Economic model breaks
- Node exploitation

**Detected by:**
- Budget increasing in logs
- Budget > initial value

**Status:** ✅ Enforced

---

### I8: Storage Isolation

**Invariant:** Agents cannot access each other's checkpoints.

**Why:**
- Privacy
- State integrity
- No cross-agent interference

**Enforcement:**
- Checkpoints keyed by AgentID
- Separate files per agent
- No shared storage

**Violation consequences:**
- State corruption
- Privacy breach
- Agent interference

**Detected by:**
- Wrong state loaded
- Checkpoint collisions

**Status:** ✅ Enforced

---

### I9: Lifecycle Order

**Invariant:** Lifecycle functions called in strict order:

```
init → [tick* → checkpoint*] → resume (optional)
```

**Why:**
- Predictable execution
- Correct state initialization
- Safe migration

**Enforcement:**
- Runtime controls call order
- Init called once before ticks
- Resume only after init
- Checkpoint callable anytime

**Violation consequences:**
- Uninitialized state
- Corrupted execution
- Resume without init

**Detected by:**
- Agent crashes
- Undefined behavior
- State corruption

**Status:** ✅ Enforced

---

### I10: Migration Atomicity

**Invariant:** Migration either fully succeeds or fully fails. No partial state.

**Why:**
- No agent loss
- No duplicate instances
- Clean error recovery

**Enforcement:**
- Synchronous migration
- Confirmation required
- Source waits for target
- Checkpoint atomic

**Violation consequences:**
- Lost agent
- Duplicate agent
- Split state

**Detected by:**
- Migration timeout
- No confirmation received
- Agent missing

**Status:** ✅ Enforced

---

## Runtime Guarantees

### What Igor Guarantees

1. **Agents execute in WASM sandbox**
2. **Ticks timeout at 100ms**
3. **Memory limited to 64MB**
4. **Budget decreases monotonically**
5. **Checkpoints are atomic**
6. **At most one instance exists**
7. **State persists through restart**
8. **Migration preserves budget**

### What Igor Does NOT Guarantee

1. **Node honesty** - Nodes can cheat
2. **Network reliability** - Connections can fail
3. **Data privacy** - Nodes see plaintext state
4. **Execution speed** - No performance SLA
5. **Migration success** - Target can refuse
6. **Budget security** - No cryptographic proofs

## Failure Modes

### Safe Failures

These failures preserve invariants:

1. **Connection timeout** → Migration fails, agent stays on source
2. **Checkpoint error** → Execution continues, retries later
3. **Budget exhausted** → Agent terminates gracefully
4. **WASM timeout** → Tick aborted, error logged
5. **Out of memory** → Agent terminates, checkpoint saved

### Unsafe Failures

These violate invariants (should never happen):

1. **Partial checkpoint write** → State corruption
2. **Double instance** → Split-brain
3. **Budget increase** → Economic model broken
4. **Sandbox escape** → Host compromise

**Detection:** Loud failures with error logs

## Audit Trail

### What Gets Logged

Every significant event:

- Agent load/init/terminate
- Tick execution with cost
- Budget deductions and balance
- Checkpoint save/load
- Migration start/complete/fail
- Errors with full context

### Log Format

Structured logging with slog:

```
time=... level=INFO msg="..." key=value key2=value2
```

**Queryable by:**
- agent_id
- operation type
- timestamp
- error status

### Audit Queries

**Find all ticks for an agent:**
```bash
grep "agent_id=local-agent" logs.txt | grep "Tick completed"
```

**Track budget over time:**
```bash
grep "budget_remaining" logs.txt | awk '{print $NF}'
```

**Detect budget exhaustion:**
```bash
grep "budget exhausted" logs.txt
```

## Invariant Violations

### How to Detect

1. **Monitor logs** - Check for error patterns
2. **Verify checkpoints** - Ensure format valid
3. **Track budgets** - Sum should be constant
4. **Count instances** - Should be ≤ 1 per agent

### How to Respond

If invariant violated:

1. **Stop affected agent** - Prevent further damage
2. **Inspect logs** - Understand root cause
3. **Check checkpoint** - Verify integrity
4. **Restore from backup** - If state corrupted
5. **Report bug** - Invariant violation is always a bug

### Example Violation

**Scenario:** Two nodes both running same agent

**Detection:**
```bash
# Node A logs
Tick completed agent_id=agent-123 counter=10

# Node B logs (same time)
Tick completed agent_id=agent-123 counter=10
```

**Response:**
1. Terminate both instances
2. Determine which checkpoint is authoritative
3. Migrate to single node
4. Investigate how split occurred

## Verification Tools

### Manual Verification

```bash
# Check checkpoint format
hexdump -C checkpoints/local-agent.checkpoint

# Verify budget in checkpoint
od -t f8 -N 8 checkpoints/local-agent.checkpoint

# Count active agents
ps aux | grep igord | wc -l
```

### Automated Verification

Create test script:

```bash
#!/bin/bash
# verify-invariants.sh

# I1: Single instance
COUNT=$(pgrep -f "run-agent.*agent-123" | wc -l)
if [ $COUNT -gt 1 ]; then
    echo "VIOLATION: Multiple instances detected"
    exit 1
fi

# I2: Budget conservation
# (requires tracking across migrations)

# I4: Atomic checkpoints
for f in checkpoints/*.checkpoint; do
    if [ $(stat -f%z "$f") -lt 16 ]; then
        echo "VIOLATION: Partial checkpoint $f"
        exit 1
    fi
done
```

## Design Principles for Invariants

### 1. Fail Loudly

When invariant violated:
- Log error immediately
- Terminate affected component
- Don't try to recover silently

### 2. Make Violations Obvious

- Clear error messages
- Detailed logging
- Explicit checks

### 3. Prevent, Don't Detect

- Atomic operations
- Synchronous protocols
- No race conditions

### 4. Simple Over Clever

- Straightforward invariants
- Easy to verify
- Easy to maintain

## Testing Invariants

### Unit Tests

Test individual invariants in isolation:

```go
func TestBudgetMonotonicity(t *testing.T) {
    initial := instance.Budget
    instance.Tick(ctx)
    if instance.Budget > initial {
        t.Error("Budget increased!")
    }
}
```

### Integration Tests

Test invariants across components:

```go
func TestSingleInstanceMigration(t *testing.T) {
    // Start agent on Node A
    // Migrate to Node B
    // Verify Node A terminated
    // Verify Node B running
    // Verify no overlap
}
```

### Chaos Tests

Test invariants under failure:

- Kill node during migration
- Corrupt checkpoint mid-write
- Disconnect during transfer
- Exhaust budget mid-tick

**Expected:** All invariants still hold.

## Invariant Documentation

Each invariant should have:

1. **Statement** - What must be true
2. **Why** - Reason for invariant
3. **Enforcement** - How it's maintained
4. **Consequences** - What breaks if violated
5. **Detection** - How to verify
6. **Status** - Enforced or documented

This document serves as the authoritative invariant specification for Igor v0.
