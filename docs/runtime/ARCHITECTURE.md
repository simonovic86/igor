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
- **Filesystem access:** Disabled (via WASI restrictions)
- **Network access:** Disabled (via WASI restrictions)
- **Tick timeout:** 15s enforced via context cancellation
- **Agent I/O:** Through `igor` hostcall module (see [HOSTCALL_ABI.md](./HOSTCALL_ABI.md)); raw WASI filesystem/network remain disabled

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

## Capability Layer

Agents interact with the outside world exclusively through runtime-provided hostcalls registered under the `igor` WASM module namespace. This is the **capability membrane** — a constitutional guarantee (CM-1 through CM-7) that all agent I/O is mediated, auditable, and replayable.

### Capability Namespaces

| Namespace | Type | Purpose | Phase |
|-----------|------|---------|-------|
| `clock` | Observation | Wall-clock time | Implemented |
| `rand` | Observation | Cryptographic randomness | Implemented |
| `kv` | Mixed | Per-agent key-value storage | Not implemented |
| `log` | Observation | Structured logging | Implemented |
| `http` | Side effect | HTTP requests to external APIs | Implemented |
| `wallet` | Mixed | Budget introspection, transfers | Implemented |

Agents declare required capabilities in a manifest. The runtime grants only declared capabilities (deny by default, CM-3). All observation hostcalls are recorded in an event log for deterministic replay (CM-4, CE-3).

See [HOSTCALL_ABI.md](./HOSTCALL_ABI.md) for function signatures and [CAPABILITY_MEMBRANE.md](../constitution/CAPABILITY_MEMBRANE.md) for constitutional invariants.

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
     ├─ Package: WASM + hash +      │
     │  state + budget + manifest   │
     │  + replay data               │
     ├─ Connect to target           │
     ├─ Open stream ────────────────>│
     ├─ Send AgentTransfer ─────────>│
     │                              ├─ Verify WASM hash
     │                              ├─ Verify replay (if present)
     │                              ├─ Save checkpoint
     │                              ├─ Load agent from bytes
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
    AgentID        string      // Unique identifier
    WASMBinary     []byte      // Compiled module (~190KB)
    WASMHash       []byte      // SHA-256 of WASMBinary
    Checkpoint     []byte      // 209-byte header (v0x04) + agent state
    ManifestData   []byte      // Capability manifest JSON
    Budget         int64       // Remaining budget in microcents
    PricePerSecond int64       // Cost per second in microcents
    ReplayData     *ReplayData // Replay verification data (optional)
}
```

Transferred over stream as JSON-encoded message.

## Economic Layer

### Runtime Metering

Every tick is timed and metered:

```go
start := time.Now()
agent_tick()
elapsed := time.Since(start)

costMicrocents := elapsed.Nanoseconds() * pricePerSecond / 1_000_000_000
budget -= costMicrocents
```

Precision: nanosecond-level via Go's monotonic clock. Overflow guard caps cost at remaining budget.

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

Checkpoints include budget, identity, and lineage as metadata (209-byte header for v0x04):

```
Byte Layout (v0x04):
┌─────────┬──────────┬─────────────┬────────────┬──────────┬──────────────┐
│ Version │  Budget  │ PricePerSec │ TickNumber │ WASMHash │ MajorVersion │
│ (1 byte)│ (8 bytes)│  (8 bytes)  │ (8 bytes)  │(32 bytes)│  (8 bytes)   │
├─────────┴──────────┼─────────────┼────────────┼──────────┼──────────────┤
│ LeaseGeneration    │ LeaseExpiry │  PrevHash  │ PubKey   │  Signature   │
│    (8 bytes)       │  (8 bytes)  │ (32 bytes) │(32 bytes)│  (64 bytes)  │
├────────────────────┴─────────────┴────────────┴──────────┴──────────────┤
│ Agent State (N bytes)                                                   │
└─────────────────────────────────────────────────────────────────────────┘
 Header: 209 bytes. Supports reading v0x02 (57 bytes) and v0x03 (81 bytes).
```

Little-endian encoding. Budget and price are int64 microcents (1 unit = 1,000,000 microcents).

On resume:
```go
budgetVal, pricePerSecond, tickNumber, wasmHash, agentState, err := ParseCheckpointHeader(checkpoint)
// Verify wasmHash matches loaded WASM binary
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
Each tick completes within 15s or is aborted.

See [RUNTIME_ENFORCEMENT_INVARIANTS.md](../enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md) for enforcement specifications and [EXECUTION_INVARIANTS.md](../constitution/EXECUTION_INVARIANTS.md) for constitutional invariants.

## Technology Stack

- **Runtime:** Go 1.25+
- **WASM Engine:** wazero (pure Go, no CGO)
- **P2P Transport:** libp2p-go
- **Agent Compilation:** TinyGo → WASM
- **Serialization:** Binary protocols, JSON for P2P messages

### Design Decisions

**wazero over Wasmtime:** wazero is a pure-Go WASM runtime with zero CGo dependencies. This keeps the build toolchain simple (`go build` just works), avoids C library linking issues across platforms, and aligns with the Go-native stack (libp2p-go, TinyGo). Wasmtime would provide WASI-P2 and component model support, but adds CGo complexity and a C toolchain dependency for marginal v0 benefit. If WASI-P2 becomes necessary, the `internal/runtime` package isolates the engine choice.

**JSON over Protobuf for P2P messages:** P2P protocol messages and capability manifests use JSON encoding. Protobuf would add schema compilation, codegen dependencies, and versioning complexity. At v0 message volumes (migration is rare, manifests are small), JSON's readability and debuggability outweigh Protobuf's size/speed advantages. The binary checkpoint format remains hand-packed for performance where it matters.

**Go over Rust:** Go provides garbage collection, fast compilation, and a rich standard library. The libp2p-go and wazero ecosystems are mature. Rust would provide memory safety guarantees and potentially better performance, but at the cost of longer compilation times, steeper learning curve, and losing the pure-Go toolchain simplicity. For a research-stage project, Go's iteration speed matters more than Rust's safety guarantees.

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

**Resource Limits:** Memory capped at 64MB, tick timeout at 15s.

**Assumptions:** Nodes are semi-trusted. Agent state is visible to nodes. Budget accounting is not cryptographically verified.

**Not secure for:** Public networks, sensitive data, production deployments.

See [SECURITY_MODEL.md](./SECURITY_MODEL.md) for complete threat model.

## References

- [docs/philosophy/OVERVIEW.md](../philosophy/OVERVIEW.md) - Project overview and design philosophy
- [AGENT_LIFECYCLE.md](./AGENT_LIFECYCLE.md) - Agent development guide
- [MIGRATION_PROTOCOL.md](./MIGRATION_PROTOCOL.md) - Protocol details
- [BUDGET_MODEL.md](./BUDGET_MODEL.md) - Economic model
