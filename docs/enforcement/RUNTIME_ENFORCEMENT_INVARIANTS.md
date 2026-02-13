# Runtime Enforcement Invariants

> **Specification cross-references:** [Spec Index](../SPEC_INDEX.md) | [Invariant Dependency Graph](./INVARIANT_DEPENDENCY_GRAPH.md) | [Runtime Constitution](../constitution/RUNTIME_CONSTITUTION.md)
>
> This document defines enforcement rules derived from the constitutional guarantees in [RUNTIME_CONSTITUTION.md](../constitution/RUNTIME_CONSTITUTION.md). Enforcement invariants operationalize constitutional contracts; they do not introduce new guarantees.

## Overview

This document defines runtime enforcement rules that implement the constitutional guarantees specified in the [Runtime Constitution](../constitution/RUNTIME_CONSTITUTION.md) and its referenced specification documents.

These invariants describe **how the runtime upholds constitutional guarantees** through concrete enforcement mechanisms. They are derived from, and subordinate to, the constitutional invariants defined in [EXECUTION_INVARIANTS.md](../constitution/EXECUTION_INVARIANTS.md), [OWNERSHIP_AND_AUTHORITY.md](../constitution/OWNERSHIP_AND_AUTHORITY.md), and [MIGRATION_CONTINUITY.md](../constitution/MIGRATION_CONTINUITY.md).

The system is designed to **fail loudly** when enforcement invariants are violated.

---

## Checkpoint Enforcement

### RE-1: Atomic Checkpoints

**Invariant:** Checkpoints are never partial. They are fully written or not at all.

**Derives from:** EI-2 (checkpoint boundary resumption), EI-3 (checkpoint lineage integrity)

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

### RE-2: State Persistence

**Invariant:** Agent state survives any shutdown or migration.

**Derives from:** EI-2 (checkpoint boundary resumption), EI-10 (migration checkpoint continuity)

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

## Budget Enforcement

### RE-3: Budget Conservation

**Invariant:** Budget is never created or destroyed, only transferred.

**Derives from:** EI-3 (checkpoint lineage integrity)

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

### RE-4: Budget Monotonicity

**Invariant:** Agent budget never increases during execution.

**Derives from:** EI-3 (checkpoint lineage integrity)

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

## Tick Enforcement

### RE-5: Tick Duration Limit

**Invariant:** Each tick completes within 100ms.

**Derives from:** EI-1 (single active instance — responsiveness supports migration handoff)

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

### RE-6: Tick Determinism

**Invariant:** Given same state and input, tick produces same output.

**Derives from:** EI-3 (checkpoint lineage integrity — determinism supports lineage verification)

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

## Sandbox and Resource Isolation

### RE-7: Storage Isolation

**Invariant:** Agents cannot access each other's checkpoints.

**Derives from:** EI-4 (authority and durability separation), OA-1 (canonical logical identity)

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

## Lifecycle Enforcement

### RE-8: Lifecycle Order

**Invariant:** Lifecycle functions called in strict order:

```
init → [tick* → checkpoint*] → resume (optional)
```

**Derives from:** EI-2 (checkpoint boundary resumption), OA-2 (authority lifecycle states)

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

## Enforcement Summary

| ID | Invariant | Category | Derives From |
|----|-----------|----------|-------------|
| RE-1 | Checkpoints are atomic (fully written or not at all) | Checkpoint | EI-2, EI-3 |
| RE-2 | Agent state survives shutdown and migration | Checkpoint | EI-2, EI-10 |
| RE-3 | Budget is conserved (never created or destroyed) | Budget | EI-3 |
| RE-4 | Budget never increases during execution | Budget | EI-3 |
| RE-5 | Tick completes within 100ms | Tick | EI-1 |
| RE-6 | Tick is deterministic given same state | Tick | EI-3 |
| RE-7 | Agents cannot access each other's checkpoints | Sandbox | EI-4, OA-1 |
| RE-8 | Lifecycle functions called in strict order | Lifecycle | EI-2, OA-2 |

---

## Runtime Guarantees

### What Igor Guarantees

1. **Agents execute in WASM sandbox**
2. **Ticks timeout at 100ms**
3. **Memory limited to 64MB**
4. **Budget decreases monotonically**
5. **Checkpoints are atomic**
6. **At most one instance exists** (constitutional — see [EI-1](../constitution/EXECUTION_INVARIANTS.md))
7. **State persists through restart**
8. **Migration preserves budget**

### What Igor Does NOT Guarantee

1. **Node honesty** — Nodes can cheat
2. **Network reliability** — Connections can fail
3. **Data privacy** — Nodes see plaintext state
4. **Execution speed** — No performance SLA
5. **Migration success** — Target can refuse
6. **Budget security** — No cryptographic proofs

---

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
2. **Double instance** → Split-brain (constitutional violation — see [EI-1](../constitution/EXECUTION_INVARIANTS.md))
3. **Budget increase** → Economic model broken
4. **Sandbox escape** → Host compromise

**Detection:** Loud failures with error logs

---

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

---

## Invariant Violations

### How to Detect

1. **Monitor logs** — Check for error patterns
2. **Verify checkpoints** — Ensure format valid
3. **Track budgets** — Sum should be constant
4. **Count instances** — Should be ≤ 1 per agent

### How to Respond

If invariant violated:

1. **Stop affected agent** — Prevent further damage
2. **Inspect logs** — Understand root cause
3. **Check checkpoint** — Verify integrity
4. **Restore from backup** — If state corrupted
5. **Report bug** — Invariant violation is always a bug

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

---

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

# RE-7: Storage isolation
COUNT=$(pgrep -f "run-agent.*agent-123" | wc -l)
if [ $COUNT -gt 1 ]; then
    echo "VIOLATION: Multiple instances detected"
    exit 1
fi

# RE-3: Budget conservation
# (requires tracking across migrations)

# RE-1: Atomic checkpoints
for f in checkpoints/*.checkpoint; do
    if [ $(stat -f%z "$f") -lt 16 ]; then
        echo "VIOLATION: Partial checkpoint $f"
        exit 1
    fi
done
```

---

## Design Principles for Enforcement

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

---

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

---

## Enforcement Invariant Documentation

Each enforcement invariant should have:

1. **Statement** — What must be true
2. **Derives from** — Which constitutional invariant(s) justify this enforcement rule
3. **Why** — Reason for invariant
4. **Enforcement** — How it's maintained
5. **Consequences** — What breaks if violated
6. **Detection** — How to verify
7. **Status** — Enforced or documented

---

## Document Status

**Type:** Runtime Enforcement Specification
**Scope:** Enforcement rules implementing constitutional guarantees — includes implementation-level detail.
**Authority:** Subordinate to [RUNTIME_CONSTITUTION.md](../constitution/RUNTIME_CONSTITUTION.md) and its referenced constitutional specifications.
