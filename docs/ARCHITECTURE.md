# Igor Architecture

Igor v0 is a decentralized runtime for autonomous mobile agents. This document describes the implementation architecture.

## System Components

```
┌───────────────────────────────────────────┐
│           Igor Node (igord)               │
├───────────────────────────────────────────┤
│  Runtime     Migration    P2P     Storage │
│  (wazero)    (libp2p)    Layer   Provider │
└───────────────────────────────────────────┘
                    │
                    │ executes
                    ▼
         ┌────────────────────┐
         │   WASM Agent       │
         │   - init()         │
         │   - tick()         │
         │   - checkpoint()   │
         │   - resume()       │
         └────────────────────┘
```

## Runtime Layer

### WASM Sandbox

Agents execute in isolated WASM instances using wazero:

- **Memory limit:** 64MB (1024 pages × 64KB)
- **Filesystem access:** Disabled
- **Network access:** Disabled
- **Tick timeout:** 100ms enforced via context cancellation

Compilation and instantiation:
```
1. Load WASM binary
2. Compile with wazero
3. Instantiate module (no auto-start)
4. Verify exports exist
5. Create agent instance
```

### Tick-Based Execution

Agents do not run continuously. Instead:

- Runtime calls `agent_tick()` periodically (1 Hz)
- Each tick is independent and must complete quickly
- Execution is metered: time measured, cost calculated, budget deducted
- State persists between ticks via checkpoint

This model enables:
- Migration at tick boundaries
- Predictable resource consumption
- Fair scheduling
- Clean termination points

### Lifecycle Functions

Agents must export:

**`agent_init()`** - One-time initialization  
**`agent_tick()`** - Periodic execution  
**`agent_checkpoint()`** - Returns state size  
**`agent_checkpoint_ptr()`** - Returns pointer to state  
**`agent_resume(ptr, len)`** - Restores from state

The runtime calls these functions at appropriate times. Agents do not control their own lifecycle directly.

## Migration Layer

### P2P Protocol

**Protocol ID:** `/igor/migrate/1.0.0`  
**Transport:** libp2p bidirectional stream  
**Encoding:** JSON

### Migration Sequence

```
Source Node                    Target Node
     │                              │
     ├─ Checkpoint agent            │
     ├─ Load WASM binary            │
     ├─ Package: WASM + state       │
     │  + budget                    │
     ├─ Connect to target           │
     ├─ Open stream ────────────────>│
     ├─ Send AgentTransfer ─────────>│
     │                              ├─ Save checkpoint
     │                              ├─ Load agent
     │                              ├─ Resume from checkpoint
     │<─ Receive AgentStarted ───────┤
     ├─ Verify success              │
     ├─ Terminate local instance    │
     ├─ Delete checkpoint           │
     X                              ● (agent continues)
```

### Single-Instance Invariant

At most one active instance exists at any time:

- Source waits for confirmation before terminating
- Target starts and confirms success
- Only then does source terminate
- No concurrent execution occurs

If migration fails at any step, agent remains on source node.

### Agent Package Format

```go
type AgentPackage struct {
    AgentID        string   // Unique identifier
    WASMBinary     []byte   // Compiled module (~190KB)
    Checkpoint     []byte   // [budget:8][price:8][state:N]
    Budget         float64  // Remaining funds
    PricePerSecond float64  // Execution cost rate
}
```

Transferred over stream as JSON-encoded message.

## Economic Layer

### Runtime Metering

Every tick is timed and metered:

```go
start := time.Now()
agent_tick()  // Execute
elapsed := time.Since(start)

cost := elapsed.Seconds() × pricePerSecond
budget -= cost
```

Precision: nanosecond-level via Go's monotonic clock.

### Budget Enforcement

Before each tick:
```go
if budget <= 0 {
    return error("budget exhausted")
}
```

After successful tick:
```go
budget -= cost
log(cost, budget_remaining)
```

When budget exhausts:
1. Stop calling agent_tick()
2. Call agent_checkpoint()
3. Save final checkpoint
4. Terminate instance

### Budget Persistence

Checkpoints include budget as metadata:

```
Byte Layout:
┌──────────┬─────────────┬────────────┐
│  Budget  │ PricePerSec │ Agent State│
│ (8 bytes)│  (8 bytes)  │ (N bytes)  │
│ float64  │  float64    │            │
└──────────┴─────────────┴────────────┘
 0          8             16           16+N
```

Little-endian IEEE 754 encoding.

On resume:
```go
budget := Float64frombits(checkpoint[0:8])
pricePerSecond := Float64frombits(checkpoint[8:16])
agentState := checkpoint[16:]
```

Budget transfers with agent during migration via `AgentPackage`.

## Storage Layer

### Storage Provider Interface

```go
type Provider interface {
    SaveCheckpoint(ctx, agentID, state []byte) error
    LoadCheckpoint(ctx, agentID) ([]byte, error)
    DeleteCheckpoint(ctx, agentID) error
}
```

Abstraction enables:
- Testing with mock storage
- Future remote storage backends
- Migration without filesystem coupling

### Filesystem Implementation

Checkpoints stored as: `checkpoints/<agentID>.checkpoint`

**Atomic write pattern:**
1. Write to temp file
2. fsync() to flush to disk
3. close() file handle
4. rename() atomically to final path

Guarantees: no partial writes visible, crash-safe persistence.

## Node Responsibilities

Each Igor node:

**Discovery:**
- Generate unique peer ID (libp2p)
- Listen on configured multiaddr
- Connect to bootstrap peers if provided

**Execution:**
- Load and compile WASM agents
- Execute tick loop with timeout enforcement
- Meter execution time
- Enforce budget constraints

**Migration:**
- Handle incoming agent transfers
- Provide outgoing agent transfer capability
- Maintain single-instance invariant

**Accounting:**
- Calculate execution costs
- Deduct from agent budgets
- Log all financial operations

Nodes operate autonomously. No centralized coordination.

## System Invariants

**I1: Single Active Instance**  
At most one active instance of any agent exists at any time.

**I2: Budget Conservation**  
Budget is never created or destroyed, only transferred.

**I3: State Persistence**  
Agent state survives any shutdown or migration.

**I4: Atomic Checkpoints**  
Checkpoints are fully written or not at all.

**I5: Budget Monotonicity**  
Agent budget never increases during execution.

**I6: Execution Determinism**  
Given same state and inputs, tick produces same outputs (agent responsibility).

**I7: Tick Timeout**  
Each tick completes within 100ms or is aborted.

See [INVARIANTS.md](./INVARIANTS.md) for complete specifications.

## Technology Stack

- **Runtime:** Go 1.22+
- **WASM Engine:** wazero (pure Go, no CGO)
- **P2P Transport:** libp2p-go
- **Agent Compilation:** TinyGo → WASM
- **Serialization:** Binary protocols, JSON for P2P messages

## Architectural Constraints

**Single-hop migration:**
Direct source-to-target only. No relay or multi-hop routing.

**Trusted accounting:**
Budget metering is not cryptographically verified. Nodes self-report honestly.

**Local checkpoints:**
Storage provider uses local filesystem. No distributed storage in v0.

**No agent discovery:**
Agents specify target nodes explicitly. No automatic discovery protocol.

## Extension Points

The architecture is designed for future expansion:

**Payment receipts:** Add cryptographic signing to execution costs  
**Multi-hop migration:** Route through intermediate nodes  
**Capability matching:** Nodes advertise capabilities, agents filter  
**Agent autonomy:** Host functions for agents to discover and evaluate nodes  
**Remote storage:** Replace FSProvider with distributed storage

These are architectural hooks, not committed features.

## Security Model

**WASM Sandbox:** Agents cannot escape sandbox, access filesystem, or make network calls.

**Resource Limits:** Memory capped at 64MB, tick timeout at 100ms.

**Assumptions:** Nodes are semi-trusted. Agent state is visible to nodes. Budget accounting is not cryptographically verified.

**Not secure for:** Public networks, sensitive data, production deployments.

See [SECURITY_MODEL.md](./SECURITY_MODEL.md) for complete threat model.

## References

- [PROJECT_CONTEXT.md](../PROJECT_CONTEXT.md) - Authoritative design specification
- [AGENT_LIFECYCLE.md](./AGENT_LIFECYCLE.md) - Agent development guide
- [MIGRATION_PROTOCOL.md](./MIGRATION_PROTOCOL.md) - Protocol details
- [BUDGET_MODEL.md](./BUDGET_MODEL.md) - Economic model
