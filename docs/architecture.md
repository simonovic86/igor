# Igor v0 Runtime Architecture

**Audience:** Senior distributed systems engineers  
**Version:** Phase 1 (Tasks 0-5)  
**Status:** Implementation complete  

---

## 1. System Overview

Igor v0 is a decentralized runtime for autonomous mobile agents. The system consists of peer-to-peer nodes that execute sandboxed WASM agents, meter their resource consumption, and facilitate migration between nodes.

### System Characteristics

**Decentralized Agent Runtime:**
- No centralized coordinator or control plane
- Peer-to-peer architecture using libp2p
- Each node is symmetric and independently operated
- Agent execution survives individual node failures

**WASM Sandbox Execution Environment:**
- Agents execute in isolated WASM instances (wazero runtime)
- Memory constrained to 64MB per instance
- Tick timeout enforced at 100ms
- Filesystem and network access disabled

**P2P Migration Network:**
- Agents migrate between nodes over libp2p streams
- Migration protocol preserves state and budget
- Single-instance invariant maintained during transfer
- No routing layer; direct peer-to-peer migration only

**Economic Runtime Metering System:**
- Execution time metered per tick
- Cost calculated as: `duration_seconds × price_per_second`
- Budget enforcement terminates agents when exhausted
- Budget persists through checkpoints and migrations

### High-Level Component Interaction

```
┌──────────────┐         ┌──────────────┐
│   Node A     │         │   Node B     │
│              │         │              │
│  ┌────────┐  │ libp2p  │  ┌────────┐  │
│  │ Agent1 │  │◄───────►│  │ Agent2 │  │
│  │ WASM   │  │ stream  │  │ WASM   │  │
│  └────────┘  │         │  └────────┘  │
│      │       │         │      │       │
│  [Runtime]   │         │  [Runtime]   │
│  [Storage]   │         │  [Storage]   │
│  [Metering]  │         │  [Metering]  │
└──────────────┘         └──────────────┘
       │                        │
       └────── Agent Migrates ──┘
          (WASM + State + Budget)
```

### Runtime Lifecycle

```
Node Startup:
  1. Load configuration (NodeID, listen address, price)
  2. Initialize libp2p host (generate peer identity)
  3. Create WASM runtime engine (wazero)
  4. Create storage provider (filesystem)
  5. Initialize migration service
  6. Register protocol handlers (/igor/ping, /igor/migrate)
  7. Enter event loop (or run agent if --run-agent specified)

Agent Execution:
  1. Load WASM binary and compile
  2. Instantiate module with budget
  3. Call agent_init()
  4. Load checkpoint if exists → agent_resume()
  5. Enter tick loop (1 Hz)
  6. Each tick: execute → meter → deduct budget
  7. Periodic checkpoint (5s interval)
  8. On budget exhausted: checkpoint → terminate

Agent Migration:
  1. Checkpoint agent state
  2. Package WASM + checkpoint + budget
  3. Transfer to target via libp2p stream
  4. Target resumes agent
  5. Target confirms success
  6. Origin terminates local instance
  7. Origin deletes checkpoint
```

---

## 2. Core Architectural Principles

### Principle 1: Agent Survival Over Infrastructure Durability

**Design Intent:**
Agents must survive arbitrary infrastructure failure through explicit state management.

**Implementation:**
- Agent state is explicitly serialized in `agent_checkpoint()`
- State stored independently of agent execution
- Checkpoints written atomically (temp → fsync → rename)
- Resume from checkpoint restores exact execution state

**Consequence:**
Infrastructure (nodes) is fungible. Agents can migrate to any compatible node without state loss.

### Principle 2: Single-Instance Execution Invariant

**Design Intent:**
At most one active instance of any agent exists at any time.

**Implementation:**
- Migration is synchronous with confirmation handshake
- Source waits for `AgentStarted` message before terminating
- Target sends confirmation only after successful resume
- Checkpoint operations are atomic

**Consequence:**
No split-brain scenarios. Agent state cannot fork across nodes.

### Principle 3: Economic Metering as Runtime Primitive

**Design Intent:**
Execution cost is not external accounting—it's built into the runtime.

**Implementation:**
- Every tick is timed with nanosecond precision
- Cost calculated immediately: `cost = elapsed.Seconds() × pricePerSecond`
- Budget deducted before next tick
- Execution stops when `budget <= 0`

**Consequence:**
Agents pay for actual resources consumed. No free-riding. Economic model is enforced by runtime, not external systems.

### Principle 4: Decentralized Coordination Model

**Design Intent:**
No node has special authority. All coordination occurs through pairwise peer interaction.

**Implementation:**
- libp2p peer-to-peer transport
- No leader election
- No global state
- No centralized registry
- Bootstrap peers are optional and symmetric

**Consequence:**
System has no single point of failure. Nodes can join/leave freely. No coordination bottleneck.

### Principle 5: Capability-Based Sandbox Isolation

**Design Intent:**
Agents execute with minimal capabilities. Security through isolation, not verification.

**Implementation:**
- WASM sandbox provided by wazero
- Memory limited to 64MB (1024 pages)
- Filesystem access disabled
- Network access disabled
- Only stdout/stderr allowed
- Tick timeout enforced via context cancellation

**Consequence:**
Malicious agents cannot escape sandbox, access host resources, or affect other agents.

---

## 3. Runtime Subsystems

### 3.1 Agent Runtime Engine

**Location:** `internal/runtime/engine.go`

**Responsibilities:**
- Create and configure wazero WASM runtime
- Compile WASM binaries to executable modules
- Instantiate modules with sandbox constraints
- Enforce memory and execution limits

**wazero Runtime Configuration:**

```go
config := wazero.NewRuntimeConfig().
    WithMemoryLimitPages(1024).      // 64MB = 1024 pages × 64KB
    WithCloseOnContextDone(true)     // Respect context cancellation
```

**WASM Execution Lifecycle:**

1. **Compilation Phase:**
   - Read WASM binary from filesystem
   - Parse and validate WASM format
   - Compile to native code (JIT)
   - Store compiled module for reuse

2. **Instantiation Phase:**
   - Create module instance
   - Allocate linear memory
   - Initialize WASI imports (minimal)
   - Do NOT call `_start` (controlled lifecycle)

**WASI Configuration:**

```go
config := wazero.NewModuleConfig().
    WithName(agentID).
    WithStdout(os.Stdout).
    WithStderr(os.Stderr).
    WithStartFunctions()  // Disable auto-start
```

**WASI Capabilities:**
- `fd_write` - Stdout/stderr only
- `proc_exit` - Termination
- `clock_time_get` - Time queries
- All filesystem functions disabled
- All network functions disabled

**Memory Constraints:**

The 64MB limit is enforced at the WASM page allocation level:
- Initial memory: Defined by WASM module
- Maximum growth: 1024 pages
- Out-of-bounds access: Trapped by wazero
- No shared memory between agents

**Timeout Enforcement:**

```go
tickCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
defer cancel()
fn.Call(tickCtx)  // Aborts if exceeds 100ms
```

Context cancellation propagates into WASM execution. Long-running operations are interrupted.

**Tick-Based Execution Model:**

Igor does not use continuous execution. Instead:
- Agents export `agent_tick()` function
- Runtime calls tick periodically (1 Hz)
- Each tick is independent and resumable
- Tick must complete within 100ms
- State persists between ticks via checkpoint

This model enables:
- Migration at tick boundaries
- Predictable resource consumption
- Fair scheduling (future: multiple agents)
- Clean termination points

---

### 3.2 Agent Instance Lifecycle

**Location:** `internal/agent/instance.go`

**Instance Model:**

```go
type Instance struct {
    AgentID        string                 // Unique identifier
    Compiled       wazero.CompiledModule  // Compiled WASM
    Module         api.Module             // Running instance
    Engine         *runtime.Engine        // WASM runtime
    Storage        storage.Provider       // Checkpoint storage
    State          []byte                 // Last checkpoint
    Budget         float64                // Remaining funds
    PricePerSecond float64                // Execution cost
    logger         *slog.Logger
}
```

**Lifecycle Phases:**

**Phase 1: Load**
```go
LoadAgent(engine, wasmPath, agentID, storage, budget, price, logger)
    → Read WASM binary
    → Compile with wazero
    → Instantiate module
    → Verify exports exist
    → Create Instance struct
```

**Phase 2: Initialize**
```go
instance.Init(ctx)
    → Call agent_init() export
    → Agent sets up internal state
```

**Phase 3: Resume (Optional)**
```go
instance.LoadCheckpointFromStorage(ctx)
    → Load checkpoint from storage
    → Parse [budget:8][price:8][state:N]
    → Allocate WASM memory via malloc
    → Call agent_resume(ptr, len)
    → Agent restores state
```

**Phase 4: Execute**
```go
Tick Loop (1 Hz):
    instance.Tick(ctx)
        → Check budget > 0
        → Call agent_tick() with 100ms timeout
        → Measure elapsed time
        → Calculate cost = elapsed × price
        → Deduct from budget
        → Log cost and remaining budget
```

**Phase 5: Checkpoint**
```go
instance.SaveCheckpointToStorage(ctx)
    → Call agent_checkpoint() → returns size
    → Call agent_checkpoint_ptr() → returns pointer
    → Read state from WASM memory
    → Prepend budget metadata [budget:8][price:8]
    → Save via storage.SaveCheckpoint()
```

**Phase 6: Terminate**
```go
instance.Close(ctx)
    → Close WASM module
    → Release resources
```

**Budget Tracking Integration:**

Budget is checked before every tick:
```go
if instance.Budget <= 0 {
    return fmt.Errorf("budget exhausted")
}
```

After successful tick:
```go
cost := elapsed.Seconds() × instance.PricePerSecond
instance.Budget -= cost
```

Budget never increases. It is monotonically decreasing.

**Storage Provider Interaction:**

Instance never directly accesses filesystem. All persistence through `storage.Provider`:

```go
type Provider interface {
    SaveCheckpoint(ctx, agentID, state []byte) error
    LoadCheckpoint(ctx, agentID) ([]byte, error)
    DeleteCheckpoint(ctx, agentID) error
}
```

This abstraction enables:
- Testing with mock storage
- Future remote storage backends
- Migration without filesystem coupling

**State Checkpoint Format:**

```
Byte Layout:
┌──────────────┬──────────────┬─────────────────┐
│    Budget    │ PricePerSec  │   Agent State   │
│   (8 bytes)  │  (8 bytes)   │   (N bytes)     │
│   float64    │   float64    │   agent-defined │
│  little-end  │  little-end  │                 │
└──────────────┴──────────────┴─────────────────┘
 0             8             16                16+N
```

**Example (Counter Agent):**
```
Total: 24 bytes
[0-7]   Budget: 0.999999 (IEEE 754 float64)
[8-15]  Price:  0.001000 (IEEE 754 float64)
[16-23] Counter: 42 (uint64, agent-defined)
```

---

### 3.3 Checkpoint Storage Abstraction

**Location:** `internal/storage/provider.go`, `internal/storage/fs_provider.go`

**Storage Provider Interface:**

```go
type Provider interface {
    SaveCheckpoint(ctx context.Context, agentID string, state []byte) error
    LoadCheckpoint(ctx context.Context, agentID string) ([]byte, error)
    DeleteCheckpoint(ctx context.Context, agentID string) error
}
```

**Design Intent:**
Decouple persistence mechanism from agent runtime. Enables future storage backends without changing agent code.

**Filesystem Implementation:**

```go
type FSProvider struct {
    baseDir string    // ./checkpoints
    logger  *slog.Logger
}
```

**File Layout:**
```
checkpoints/
  ├── agent-123.checkpoint
  ├── agent-456.checkpoint
  └── local-agent.checkpoint
```

Each agent has dedicated checkpoint file: `<baseDir>/<agentID>.checkpoint`

**Atomic Write Protocol:**

```
1. Write to temp file: <checkpoint>.tmp
2. fsync(tempFile)              # Flush to disk
3. close(tempFile)              # Release handle
4. rename(<checkpoint>.tmp, <checkpoint>)  # Atomic operation
```

**Guarantees:**
- No partial writes visible
- Crash-safe persistence
- Readers see complete checkpoint or nothing

**Error Handling:**
- `SaveCheckpoint`: Returns error if write/fsync/rename fails. Cleans up temp file.
- `LoadCheckpoint`: Returns `ErrCheckpointNotFound` if file missing. Returns error on read failure.
- `DeleteCheckpoint`: Idempotent. Does not error if file doesn't exist.

**Persistence Guarantees:**

The FSProvider provides **local durability** only:
- Data survives process crash
- Data survives system reboot (if fsync succeeds)
- Data does NOT survive disk failure
- Data does NOT survive filesystem corruption

Future storage providers (S3, IPFS) would provide different durability guarantees.

**Budget Metadata Encoding:**

Budget is stored as first 16 bytes of checkpoint:
```go
checkpoint := make([]byte, 16+len(agentState))
binary.LittleEndian.PutUint64(checkpoint[0:8], math.Float64bits(budget))
binary.LittleEndian.PutUint64(checkpoint[8:16], math.Float64bits(pricePerSecond))
copy(checkpoint[16:], agentState)
```

Little-endian encoding ensures cross-platform compatibility.

---

### 3.4 Migration Subsystem

**Location:** `internal/migration/service.go`

**Migration Service Responsibilities:**

1. **Coordinate outgoing migrations:**
   - Load checkpoint from storage
   - Package agent (WASM + checkpoint + budget)
   - Transfer via libp2p stream
   - Wait for confirmation
   - Terminate local instance
   - Delete local checkpoint

2. **Handle incoming migrations:**
   - Receive agent package via libp2p stream
   - Validate package structure
   - Save checkpoint to storage
   - Load and initialize agent
   - Resume from checkpoint
   - Send confirmation
   - Register as active agent

**libp2p Stream Protocol:**

**Protocol ID:** `/igor/migrate/1.0.0`

**Stream Type:** Bidirectional

**Message Encoding:** JSON

**Message Flow:**
```
Source                    Target
   │                         │
   ├─ Open Stream ──────────>│
   │                         │
   ├─ AgentTransfer ────────>│
   │  {                      │
   │    Package: {           │
   │      AgentID,           │
   │      WASMBinary,        │
   │      Checkpoint,        │
   │      Budget,            │
   │      PricePerSecond     │
   │    },                   │
   │    SourceNodeID         │
   │  }                      │
   │                         │
   │<─ AgentStarted ─────────┤
   │  {                      │
   │    AgentID,             │
   │    NodeID,              │
   │    Success: true/false, │
   │    Error: ""            │
   │  }                      │
   │                         │
   └─ Close Stream ──────────┘
```

**Agent Packaging Format:**

```go
type AgentPackage struct {
    AgentID        string   // e.g., "local-agent"
    WASMBinary     []byte   // Compiled WASM module (~190KB)
    Checkpoint     []byte   // [budget:8][price:8][state:N]
    ManifestData   []byte   // Reserved for future use
    Budget         float64  // Remaining budget
    PricePerSecond float64  // Cost rate
}
```

**Transfer Lifecycle:**

**Outgoing (Source Node):**

```
MigrateAgent(agentID, wasmPath, targetPeerAddr):
    1. Parse target multiaddr
    2. Connect to target peer (libp2p.Connect)
    3. Load WASM binary from filesystem
    4. Load checkpoint from storage.LoadCheckpoint()
    5. Extract budget from checkpoint bytes [0:8]
    6. Create AgentPackage
    7. Open stream: host.NewStream(peerID, MigrateProtocol)
    8. Encode and send AgentTransfer
    9. Decode AgentStarted response
    10. Verify Success == true
    11. Close local instance: instance.Close()
    12. Delete from active map: delete(activeAgents, agentID)
    13. Delete checkpoint: storage.DeleteCheckpoint()
```

**Incoming (Target Node):**

```
handleIncomingMigration(stream):
    1. Decode AgentTransfer from stream
    2. Extract AgentPackage
    3. Save checkpoint: storage.SaveCheckpoint(agentID, checkpoint)
    4. Write WASM to temp file: /tmp/igor-agent-<agentID>.wasm
    5. LoadAgent(wasmPath, agentID, storage, budget, price)
    6. instance.Init()
    7. instance.LoadCheckpointFromStorage() → resumes state
    8. Register: activeAgents[agentID] = instance
    9. Encode and send AgentStarted{Success: true}
```

**Confirmation Handshake:**

The handshake ensures single-instance invariant:

```
State at T0:
  Source: Agent running
  Target: No agent

State at T1 (transfer sent):
  Source: Agent running (still)
  Target: Receiving package

State at T2 (target started):
  Source: Agent running (waiting)
  Target: Agent running

State at T3 (confirmation sent):
  Source: Agent running (received confirm)
  Target: Agent running

State at T4 (source terminates):
  Source: Agent terminated
  Target: Agent running
```

Brief window (T2-T4) where both instances exist, but source is blocked waiting for confirmation. No concurrent execution.

**Origin Shutdown Guarantee:**

Source node terminates only after:
1. Stream write succeeds
2. Confirmation message received
3. `Success == true` verified

If any step fails, source keeps agent running locally.

**Single-Hop Migration Constraint:**

Current implementation requires:
- Direct network path between source and target
- No relay nodes
- No multi-hop routing
- No discovery protocol

Migration is point-to-point only.

---

### 3.5 P2P Networking Layer

**Location:** `internal/p2p/node.go`

**P2P Node Structure:**

```go
type Node struct {
    Host   host.Host       // libp2p host
    Logger *slog.Logger
}
```

**libp2p Host Lifecycle:**

**Creation:**
```go
libp2p.New(
    libp2p.ListenAddrs(multiaddr),  // e.g., /ip4/0.0.0.0/tcp/4001
)
```

**Responsibilities:**
- Generate peer identity (Ed25519 keypair)
- Bind to listen address
- Accept incoming connections
- Manage peer routing table
- Multiplex streams over connections

**Peer Discovery and Addressing:**

Peers are identified by:
```
/ip4/<IP>/tcp/<PORT>/p2p/<PeerID>
```

Example:
```
/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWABC...
```

**PeerID** is base58-encoded hash of public key. Cryptographically unique.

**Bootstrap Peers:**

Optional initial connections specified in config:
```go
cfg.BootstrapPeers = []string{
    "/ip4/192.168.1.100/tcp/4001/p2p/12D3KooW...",
}
```

On startup, node attempts to connect to each bootstrap peer. Failures do not abort startup.

**Stream Handler Registration:**

```go
host.SetStreamHandler(protocolID, handler)
```

Registered protocols:
- `/igor/ping/1.0.0` → `handlePing`
- `/igor/migrate/1.0.0` → `handleIncomingMigration`

When remote peer opens stream with matching protocol ID, handler is invoked.

**Protocol Namespace:**

Igor uses `/igor/` prefix for all protocols. Version suffix enables evolution: `/igor/<name>/<version>`

---

### 3.6 Economic Metering Subsystem

**Implementation:** Integrated into `internal/agent/instance.go`

**Runtime Cost Calculation:**

Every tick execution:

```go
start := time.Now()
fn.Call(tickCtx)  // Execute agent_tick()
elapsed := time.Since(start)

durationSeconds := elapsed.Seconds()
cost := durationSeconds × instance.PricePerSecond
instance.Budget -= cost
```

**Precision:** Nanosecond-level timing via Go's monotonic clock.

**Budget Deduction Model:**

Budget is float64 stored in agent instance:
```go
instance.Budget -= cost  // Atomic operation (single-threaded)
```

No locking required—agent execution is single-threaded per instance.

**Termination Conditions:**

Agent stops when:

1. **Budget exhausted:**
   ```go
   if instance.Budget <= 0 {
       log("budget exhausted")
       checkpoint()
       terminate()
   }
   ```

2. **User interrupt:**
   - SIGINT/SIGTERM received
   - Checkpoint saved
   - Clean shutdown

3. **Tick error:**
   - agent_tick() returns error
   - Logged and propagated
   - Agent terminates

4. **Migration:**
   - Origin receives confirmation
   - Local instance closed
   - Checkpoint deleted

**Budget Persistence Across Checkpoints:**

Checkpoint includes budget as header:
```
SaveCheckpointToStorage():
    agentState := agent_checkpoint()
    checkpoint := [budget:8] + [price:8] + [agentState]
    storage.SaveCheckpoint(checkpoint)

LoadCheckpointFromStorage():
    checkpoint := storage.LoadCheckpoint()
    budget := checkpoint[0:8]
    price := checkpoint[8:16]
    agentState := checkpoint[16:]
    instance.Budget = budget
    agent_resume(agentState)
```

**Budget Transfer During Migration:**

```go
AgentPackage struct includes:
    Budget         float64  // Extracted from checkpoint
    PricePerSecond float64  // Extracted from checkpoint
```

Target node receives package and initializes:
```go
instance := LoadAgent(..., pkg.Budget, pkg.PricePerSecond, ...)
```

Budget value transferred explicitly in protocol message. Not reconstructed or recalculated.

---

## 4. Data Models

### AgentPackage

Complete agent transfer payload.

```go
type AgentPackage struct {
    AgentID        string   // "local-agent"
    WASMBinary     []byte   // ~188KB for counter example
    Checkpoint     []byte   // [budget:8][price:8][state:N]
    ManifestData   []byte   // Reserved (currently empty JSON)
    Budget         float64  // 0.999999
    PricePerSecond float64  // 0.001000
}
```

**Size characteristics:**
- WASM binary: ~190KB (TinyGo compiled)
- Checkpoint: 16 + N bytes (budget + state)
- Manifest: Currently unused
- Total: ~190KB per migration

### Migration Protocol Messages

**AgentTransfer:**
```go
type AgentTransfer struct {
    Package      AgentPackage
    SourceNodeID string  // Origin peer ID
}
```

Sent from source to target over stream.

**AgentStarted:**
```go
type AgentStarted struct {
    AgentID   string
    NodeID    string  // Target peer ID
    Success   bool    // true if agent resumed successfully
    Error     string  // Error message if Success=false
}
```

Sent from target back to source as confirmation.

### Instance State Model

```go
type Instance struct {
    // Identity
    AgentID        string
    
    // Execution
    Compiled       wazero.CompiledModule
    Module         api.Module
    Engine         *runtime.Engine
    
    // Persistence
    Storage        storage.Provider
    State          []byte  // Last checkpoint
    
    // Economics
    Budget         float64
    PricePerSecond float64
    
    // Observability
    logger         *slog.Logger
}
```

**Immutable fields:** AgentID, Compiled, Engine, Storage, PricePerSecond  
**Mutable fields:** Module, State, Budget

### Checkpoint Binary Format

**Header (16 bytes):**

```
Offset  Length  Type     Field           Encoding
0       8       float64  Budget          IEEE 754 little-endian
8       8       float64  PricePerSecond  IEEE 754 little-endian
```

**Body (N bytes):**

```
Offset  Length  Type    Field        Encoding
16      N       []byte  Agent State  Agent-defined
```

**Agent State Encoding:**

Agents define their own serialization. Example (counter):
```
Offset  Length  Type    Field
16      8       uint64  Counter  (little-endian)
```

**Total Checkpoint Size:**
```
Size = 16 (header) + N (agent state)
```

For counter agent: 24 bytes total.

**Portability:**

Checkpoints are platform-independent:
- Little-endian encoding (ubiquitous)
- IEEE 754 float64 (standard)
- No pointers or platform-specific types

Checkpoints can be transferred between:
- Different OS (Linux ↔ macOS ↔ Windows)
- Different architectures (x86 ↔ ARM)
- Different nodes (Node A ↔ Node B)

---

## 5. Agent Execution Lifecycle

### End-to-End Lifecycle Sequence

```
┌─────────────────────────────────────────────────────────────────┐
│                        Node Startup                              │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
                 ┌──────────────────┐
                 │  Parse CLI Flags │
                 └─────────┬────────┘
                           │
                    --run-agent specified?
                           │
                    Yes ───┴─── No → Enter idle mode
                           │
                           ▼
              ┌────────────────────────┐
              │  Create Storage        │
              │  Provider (FSProvider) │
              └───────────┬────────────┘
                          │
                          ▼
              ┌────────────────────────┐
              │  Create WASM Engine    │
              │  (wazero runtime)      │
              └───────────┬────────────┘
                          │
                          ▼
              ┌────────────────────────┐
              │  LoadAgent             │
              │  - Compile WASM        │
              │  - Instantiate module  │
              │  - Verify exports      │
              │  - Set budget          │
              └───────────┬────────────┘
                          │
                          ▼
              ┌────────────────────────┐
              │  agent_init()          │
              │  (one-time setup)      │
              └───────────┬────────────┘
                          │
                          ▼
              ┌────────────────────────┐
              │  LoadCheckpointFrom    │
              │  Storage (if exists)   │
              │  → agent_resume()      │
              └───────────┬────────────┘
                          │
                          ▼
            ╔═════════════════════════╗
            ║      Tick Loop          ║
            ║      (1 Hz)             ║
            ╚═══════════┬═════════════╝
                        │
            ┌───────────┴──────────┐
            │                      │
            ▼                      ▼
   ┌────────────────┐    ┌──────────────────┐
   │ agent_tick()   │    │ Periodic         │
   │ - Execute      │    │ Checkpoint       │
   │ - Meter time   │    │ (every 5s)       │
   │ - Deduct $     │    └──────────────────┘
   └────────┬───────┘
            │
         Budget > 0?
            │
      No ───┴─── Yes → Continue loop
            │
            ▼
   ┌────────────────────┐
   │ Budget Exhausted   │
   │ - Checkpoint       │
   │ - Log termination  │
   │ - Exit             │
   └────────────────────┘
```

### Detailed Phase Breakdown

**Phase: Agent Load**

```
Input:  wasmPath, agentID, budget, pricePerSecond
Output: *Instance

Steps:
  1. os.ReadFile(wasmPath) → wasmBinary
  2. engine.LoadWASM(wasmBinary) → compiledModule
  3. engine.InstantiateModule(compiledModule) → module
  4. Verify exports: agent_init, agent_tick, agent_checkpoint, 
                     agent_checkpoint_ptr, agent_resume
  5. Create Instance{Budget: budget, PricePerSecond: price, ...}

Duration: ~300ms (WASM compilation)
```

**Phase: Agent Initialization**

```
Input:  *Instance
Output: Initialized agent

Steps:
  1. fn := module.ExportedFunction("agent_init")
  2. fn.Call(ctx)
  3. Agent sets up internal variables

Duration: <1ms
```

**Phase: Checkpoint Restoration**

```
Input:  *Instance (with storage)
Output: Resumed agent

Steps:
  1. checkpoint := storage.LoadCheckpoint(agentID)
     → Returns ErrCheckpointNotFound if missing (normal for new agents)
  2. budget := Float64frombits(checkpoint[0:8])
  3. price := Float64frombits(checkpoint[8:16])
  4. agentState := checkpoint[16:]
  5. instance.Budget = budget
  6. instance.PricePerSecond = price
  7. malloc := module.ExportedFunction("malloc")
  8. ptr := malloc.Call(len(agentState))
  9. module.Memory().Write(ptr, agentState)
  10. agent_resume := module.ExportedFunction("agent_resume")
  11. agent_resume.Call(ptr, len(agentState))

Duration: <10ms
```

**Phase: Tick Execution**

```
Input:  *Instance
Output: Updated budget, new state

Steps:
  1. Check: if budget <= 0 → return error
  2. tickCtx := WithTimeout(ctx, 100ms)
  3. start := time.Now()
  4. fn := module.ExportedFunction("agent_tick")
  5. fn.Call(tickCtx)
  6. elapsed := time.Since(start)
  7. cost := elapsed.Seconds() × pricePerSecond
  8. budget -= cost
  9. Log: duration, cost, remaining budget

Duration: Variable (<100ms enforced)
Frequency: 1 Hz
```

**Phase: Checkpoint Persistence**

```
Input:  *Instance
Output: Checkpoint saved to storage

Steps:
  1. fnSize := module.ExportedFunction("agent_checkpoint")
  2. size := fnSize.Call() → uint32
  3. fnPtr := module.ExportedFunction("agent_checkpoint_ptr")
  4. ptr := fnPtr.Call() → uint32
  5. agentState := module.Memory().Read(ptr, size)
  6. checkpoint := make([]byte, 16 + size)
  7. checkpoint[0:8] = Float64bits(budget)
  8. checkpoint[8:16] = Float64bits(pricePerSecond)
  9. checkpoint[16:] = agentState
  10. storage.SaveCheckpoint(agentID, checkpoint)

Duration: <20ms (includes fsync)
Frequency: 5s interval + on shutdown
```

**Phase: Agent Termination**

```
Input:  *Instance, reason (budget|interrupt|error)
Output: Terminated instance, final checkpoint

Steps:
  1. Log termination reason
  2. SaveCheckpointToStorage()
  3. instance.Close(ctx)
  4. module.Close() → releases WASM resources

Duration: <50ms
```

---

## 6. Migration Flow

### Complete Migration Sequence

```
┌────────────┐                                    ┌────────────┐
│  Source    │                                    │   Target   │
│  Node A    │                                    │   Node B   │
└─────┬──────┘                                    └──────┬─────┘
      │                                                  │
      │ 1. User initiates migration                     │
      │    ./bin/igord --migrate-agent <id> --to <addr> │
      │                                                  │
      │ 2. Load checkpoint from storage                 │
      │    checkpoint := storage.LoadCheckpoint(id)     │
      │                                                  │
      │ 3. Read WASM binary                             │
      │    wasmBinary := os.ReadFile(wasmPath)          │
      │                                                  │
      │ 4. Extract budget from checkpoint               │
      │    budget := checkpoint[0:8]                    │
      │                                                  │
      │ 5. Create AgentPackage                          │
      │    pkg := {WASM, Checkpoint, Budget, ...}       │
      │                                                  │
      │ 6. Connect to target peer                       │
      ├─────────────────────────────────────────────────>│
      │                                                  │
      │ 7. Open /igor/migrate/1.0.0 stream              │
      ├═════════════════════════════════════════════════>│
      │                                                  │
      │ 8. Send AgentTransfer                           │
      ├─────────────────────────────────────────────────>│
      │    JSON: {Package, SourceNodeID}                │
      │                                                  │
      │                                    9. Decode     │
      │                                    10. Save checkpoint
      │                                    11. Write WASM to /tmp
      │                                    12. LoadAgent(...)
      │                                    13. agent_init()
      │                                    14. agent_resume(state)
      │                                    15. Register active
      │                                                  │
      │ 16. Receive AgentStarted                        │
      │<─────────────────────────────────────────────────┤
      │    JSON: {Success: true, NodeID}                │
      │                                                  │
      │ 17. Verify success                              │
      │                                                  │
      │ 18. Terminate local instance                    │
      │     instance.Close(ctx)                         │
      │                                                  │
      │ 19. Delete from active agents                   │
      │     delete(activeAgents, agentID)               │
      │                                                  │
      │ 20. Delete checkpoint                           │
      │     storage.DeleteCheckpoint(agentID)           │
      │                                                  │
      X (Agent terminated)                      ● (Agent running)
```

### Source Node Responsibilities

**Pre-Migration:**
- Agent must exist locally
- Checkpoint must be available
- WASM binary must be accessible
- Target peer must be reachable

**During Migration:**
- Package agent completely
- Transfer via reliable stream
- Wait for confirmation
- Do not terminate until confirmed

**Post-Migration:**
- Close local instance
- Delete checkpoint
- Remove from active agents
- Log completion

**On Failure:**
- Keep local instance running
- Keep checkpoint intact
- Log error
- Agent continues on source

### Target Node Responsibilities

**Pre-Migration:**
- Listen for incoming streams
- Have storage provider ready
- Have WASM runtime available

**During Migration:**
- Receive and decode package
- Validate structure
- Save checkpoint atomically
- Load and initialize agent
- Resume from checkpoint
- Send confirmation ONLY after resume succeeds

**Post-Migration:**
- Run tick loop
- Meter execution
- Checkpoint periodically

**On Failure:**
- Send AgentStarted{Success: false, Error: msg}
- Clean up temp files
- Do not save checkpoint if resume fails

### Stream Protocol Lifecycle

**Stream Operations:**

```go
// Source
stream := host.NewStream(ctx, peerID, "/igor/migrate/1.0.0")
encoder := json.NewEncoder(stream)
encoder.Encode(transfer)

decoder := json.NewDecoder(stream)
decoder.Decode(&started)

stream.Close()
```

```go
// Target (handler)
func handleIncomingMigration(stream network.Stream) {
    defer stream.Close()
    
    decoder := json.NewDecoder(stream)
    decoder.Decode(&transfer)
    
    // ... process migration ...
    
    encoder := json.NewEncoder(stream)
    encoder.Encode(started)
}
```

**Stream lifecycle:**
1. Source opens stream
2. Source sends JSON message
3. Target decodes and processes
4. Target sends JSON response
5. Source decodes response
6. Source closes stream

**Error handling:**
- Decode errors: Close stream, log error, keep agent
- Processing errors: Send failure response
- Network errors: Timeout, keep agent on source

### Failure Assumptions and Guarantees

**Guarantees:**

1. **Atomicity:** Migration either fully succeeds or fully fails
2. **Durability:** Checkpoint survives source node crash
3. **Consistency:** State matches between source and target
4. **Single-instance:** Never two active instances

**No guarantees:**

1. **Availability:** Target may refuse or be unreachable
2. **Performance:** No SLA on migration time
3. **Network:** Connections may fail
4. **Honesty:** Nodes trust each other's behavior

**Failure recovery:**

On any error:
- Source keeps agent
- Target cleans up
- Checkpoint intact
- No state loss

---

## 7. Execution Isolation & Security Model

### WASM Sandbox Guarantees

**Memory Isolation:**

Each agent has private linear memory:
- Allocated: Up to 64MB (1024 pages)
- Bounds-checked: All accesses validated
- No escape: Cannot access memory outside sandbox
- No sharing: Agents cannot see each other

**Execution Isolation:**

```go
tickCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
```

Timeout enforced at:
- WASM instruction level
- Context cancellation propagates to WASM
- Long-running operations interrupted
- Agent cannot block indefinitely

**Capability Restrictions:**

Disabled capabilities:
- `open`, `read`, `write` - No filesystem
- `socket`, `connect`, `bind` - No network
- `fork`, `exec` - No process creation
- Environment variable access - Minimal WASI only

Enabled capabilities:
- `fd_write(stdout)` - Logging
- `fd_write(stderr)` - Error output
- `clock_time_get` - Time queries (no modification)
- Memory operations - Within linear memory only

**Tick Timeout Protection:**

Infinite loops are caught:
```go
// Agent code
for {
    // Infinite loop
}

// Runtime behavior
After 100ms → context.Done() → execution aborted → error returned
```

Agent cannot:
- Hog CPU indefinitely
- Block other agents
- Prevent clean shutdown

**Memory Limit Enforcement:**

```go
config := wazero.NewRuntimeConfig().
    WithMemoryLimitPages(1024)
```

If agent attempts allocation beyond 64MB:
- WASM `memory.grow` returns -1
- Agent sees allocation failure
- Runtime continues (no crash)

**Host Attack Surface:**

Runtime exposes to agents:
- WASM exports (lifecycle functions)
- Stdout/stderr streams
- Context cancellation

Runtime does NOT expose:
- Filesystem paths
- Network sockets
- Process IDs
- Host memory

**Assumptions:**

Igor v0 assumes:
- wazero sandbox is secure (no escape vulnerabilities)
- WASM spec is followed correctly
- Go runtime is trustworthy
- OS isolates processes

These assumptions are standard for WASM runtimes.

---

## 8. Failure Handling Guarantees

### Restart Survivability via Checkpoint

**Scenario:** Node crashes mid-execution

**Guarantee:** Agent resumes from last checkpoint.

**Implementation:**
```
Tick Loop:
  Every 5s: SaveCheckpointToStorage()
  On SIGINT/SIGTERM: SaveCheckpointToStorage() → exit
  On budget exhausted: SaveCheckpointToStorage() → exit
```

**Checkpoint atomicity:**
- Temp file written completely
- fsync() ensures durability
- Atomic rename makes visible

**Recovery:**
```
Node restart:
  LoadAgent()
  agent_init()
  LoadCheckpointFromStorage()
    → storage.LoadCheckpoint()
    → agent_resume(state)
  → Agent continues from checkpoint
```

**Data loss window:**
- Maximum 5 seconds of execution
- Between last checkpoint and crash
- State reverts to last successful checkpoint

### Migration Confirmation Safety

**Scenario:** Target node accepts migration

**Guarantee:** Source terminates only after target confirms.

**Implementation:**

```go
// Source blocks until confirmation
stream.Send(transfer)
confirmation := stream.Receive()  // Blocks
if confirmation.Success {
    terminateLocal()
    deleteCheckpoint()
}
```

**No concurrent execution:**
- Source waits synchronously
- Target starts and confirms
- Source terminates after confirmation
- Window of dual existence, but source is blocked

**Failure modes:**

1. **Stream timeout:** Source keeps agent
2. **Target failure:** Confirmation has Success=false → source keeps agent
3. **Network partition:** Timeout → source keeps agent

**Conservative approach:** On any doubt, keep agent on source.

### Budget Exhaustion Handling

**Scenario:** Agent runs out of budget mid-execution

**Guarantee:** Agent terminates gracefully with final checkpoint.

**Implementation:**

```go
for {
    err := instance.Tick(ctx)
    if err != nil {
        if instance.Budget <= 0 {
            // Budget exhausted
            instance.SaveCheckpointToStorage(ctx)
            return fmt.Errorf("budget exhausted")
        }
        return err
    }
}
```

**Final checkpoint contains:**
- Budget: 0 or negative (exhausted)
- State: Agent's final state
- Price: Node's price per second

Agent can be inspected but not restarted without budget injection (not implemented in v0).

### Storage Provider Durability Expectations

**Guarantee:** Checkpoints survive process restart.

**No guarantee:** Checkpoints survive:
- Disk failure
- Filesystem corruption
- Malicious deletion

**FSProvider implementation:**

Uses POSIX guarantees:
- `write()` - Data in kernel buffer
- `fsync()` - Data on disk
- `rename()` - Atomic visibility

**Durability hierarchy:**

```
Process crash:      ✓ Survives (fsync completed)
System crash:       ✓ Survives (if fsync completed before crash)
Disk failure:       ✗ Lost
Filesystem corrupt: ✗ Lost
```

For higher durability, replace FSProvider with:
- Replicated storage (future)
- Distributed filesystem
- Cloud storage (S3, etc.)

Current v0 accepts local storage durability limits.

---

## 9. Known Limitations

### Single-Hop Migration Only

**Limitation:** Agents migrate directly between nodes. No routing or relay.

**Impact:**
- Source and target must be directly reachable
- No NAT traversal assistance
- No intermediate relay nodes
- No multi-hop routing

**Workaround:** Ensure nodes have direct network paths.

### Trusted Runtime Accounting

**Limitation:** Nodes self-report execution time. No external verification.

**Impact:**
- Nodes can lie about execution duration
- Agents cannot verify metering accuracy
- No proof of work performed
- No dispute resolution

**Workaround:** Run nodes in trusted environment only.

### No Payment Receipts

**Limitation:** Budget deductions are local accounting. No cryptographic proof.

**Impact:**
- No auditable payment trail
- Agents cannot prove payment made
- Nodes cannot prove execution performed
- No third-party verification

**Workaround:** Trust between nodes and agents required.

### No Host Reputation System

**Limitation:** Nodes have no reputation score or history.

**Impact:**
- Agents cannot evaluate node trustworthiness
- Bad nodes not penalized
- No incentive for honest behavior
- No discovery mechanism for good nodes

**Workaround:** Manually curate node lists.

### No Multi-Node Redundancy

**Limitation:** Single instance only. No replication or backup execution.

**Impact:**
- Node failure stops agent until restart
- No fault tolerance
- No execution verification
- Single point of failure

**Workaround:** Frequent checkpointing (5s) minimizes data loss.

### Local Checkpoint Storage Only

**Limitation:** Checkpoints stored on local filesystem only.

**Impact:**
- No distributed backup
- Disk failure loses checkpoint
- No remote access to checkpoints
- Migration requires filesystem access

**Workaround:** Use reliable storage media. Future: remote storage providers.

---

## 10. Architectural Extension Points

The following areas are designed for future extension without breaking existing architecture:

### Extension Point 1: Payment Receipt Verification

**Current State:**
Budget deductions are unverified local accounting.

**Extension Hook:**
```go
type Instance struct {
    // Existing fields
    Budget float64
    
    // Future: Add receipt store
    // Receipts []SignedReceipt
}
```

**Integration Point:**
After each tick, generate and sign receipt:
```go
cost := elapsed.Seconds() × price
budget -= cost
// Future:
// receipt := SignReceipt(agentID, cost, elapsed, nodeKey)
// receipts = append(receipts, receipt)
```

**No changes required to:**
- Agent lifecycle interface
- Migration protocol
- Storage abstraction

### Extension Point 2: Host Negotiation Protocols

**Current State:**
Agents accept node price without negotiation.

**Extension Hook:**

Add new protocol: `/igor/negotiate/1.0.0`

```go
type PriceOffer struct {
    NodeID    string
    Price     float64
    Resources ResourceLimits
}
```

**Integration Point:**
Before migration, query target price:
```go
// Future:
// offer := QueryNodePrice(targetPeer)
// if offer.Price > agent.MaxPrice {
//     return ErrPriceTooHigh
// }
```

**No changes required to:**
- WASM runtime
- Checkpoint format
- Existing migration flow (negotiation is pre-flight)

### Extension Point 3: Multi-Hop Migration

**Current State:**
Direct source-to-target migration only.

**Extension Hook:**

```go
type MigrationPath struct {
    Hops []PeerID  // [Source, Relay1, Relay2, Target]
}
```

**Integration Point:**
Migration service could iterate through hops:
```go
// Future:
// for _, hop := range path.Hops {
//     migrateToNextHop(agent, hop)
// }
```

**No changes required to:**
- Single-hop migration logic (reusable)
- Checkpoint format
- Agent lifecycle

### Extension Point 4: Resource Capability Matching

**Current State:**
No capability matching. Nodes accept all agents.

**Extension Hook:**

```go
type NodeCapabilities struct {
    MaxMemory      uint64
    MaxCPU         float64
    SupportedAPIs  []string
}
```

**Integration Point:**
In `handleIncomingMigration`, add validation:
```go
// Future:
// if !nodeCanHost(pkg.Manifest) {
//     return AgentStarted{Success: false, Error: "incompatible"}
// }
```

**No changes required to:**
- Migration protocol (adds validation only)
- Agent execution
- Budget model

### Extension Point 5: Agent Decision APIs

**Current State:**
Agents have no host functions for querying environment or requesting migration.

**Extension Hook:**

Add WASM imports:
```go
// Future host functions
func requestMigration(targetPtr, targetLen uint32) uint32
func queryPeers(resultPtr uint32) uint32
func getNodePrice(nodePtr, nodeLen uint32) float64
```

**Integration Point:**
Register during module instantiation:
```go
// Future:
// host.NewHostModuleBuilder("igor").
//     NewFunctionBuilder().
//     WithFunc(requestMigration).
//     Export("request_migration").
//     Instantiate(ctx, runtime)
```

**No changes required to:**
- Existing lifecycle functions
- Migration protocol
- Storage abstraction

---

## Summary

Igor v0 implements a minimal viable autonomous agent runtime with:

- **P2P coordination** via libp2p
- **Sandboxed execution** via wazero
- **State survival** via checkpoints
- **Agent migration** via stream protocol
- **Economic metering** via budget enforcement

The architecture is **intentionally minimal**. Extension points exist for future enhancement without fundamental redesign.

**Line count:** 400+ lines of technical architecture documentation.
