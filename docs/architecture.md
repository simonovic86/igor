# Igor v0 Architecture

## System Components

Igor v0 consists of four primary components:

```
┌─────────────────────────────────────────────────────┐
│                   Igor Node (igord)                  │
├─────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌────────────┐  ┌─────────────┐ │
│  │  P2P Network │  │   WASM     │  │  Migration  │ │
│  │   (libp2p)   │  │  Runtime   │  │   Service   │ │
│  │              │  │  (wazero)  │  │             │ │
│  └──────────────┘  └────────────┘  └─────────────┘ │
│  ┌──────────────┐  ┌────────────┐  ┌─────────────┐ │
│  │   Storage    │  │   Agent    │  │   Config    │ │
│  │   Provider   │  │  Instance  │  │             │ │
│  └──────────────┘  └────────────┘  └─────────────┘ │
└─────────────────────────────────────────────────────┘
                         │
                         │ runs
                         ▼
              ┌──────────────────┐
              │  WASM Agent      │
              │  - init()        │
              │  - tick()        │
              │  - checkpoint()  │
              │  - resume()      │
              └──────────────────┘
```

## Component Details

### 1. P2P Network Layer (`internal/p2p`)

**Responsibilities:**
- Peer discovery and connection management
- libp2p host lifecycle
- Protocol handler registration
- Stream-based communication

**Key Protocols:**
- `/igor/ping/1.0.0` - Connectivity testing
- `/igor/migrate/1.0.0` - Agent migration

**Implementation:**
- Uses libp2p-go for transport
- Generates unique peer ID per node
- Listens on configurable multiaddr
- Supports bootstrap peer connections

### 2. WASM Runtime (`internal/runtime`)

**Responsibilities:**
- WASM module compilation and instantiation
- Sandbox enforcement
- Memory limits
- Execution timeout enforcement

**Security Constraints:**
- Memory limit: 64MB per agent
- No filesystem access
- No network access
- Tick timeout: 100ms

**Implementation:**
- Uses wazero pure-Go WASM runtime
- WASI support with minimal capabilities
- Deterministic execution

### 3. Agent Instance (`internal/agent`)

**Responsibilities:**
- Agent lifecycle management
- Calling WASM lifecycle functions
- Budget tracking and metering
- Checkpoint serialization
- State restoration

**Lifecycle Functions:**
- `agent_init()` - One-time initialization
- `agent_tick()` - Periodic execution
- `agent_checkpoint()` - State serialization
- `agent_resume(ptr, len)` - State restoration

**Budget Metering:**
- Measures actual tick execution time
- Calculates cost: `duration_seconds × price_per_second`
- Deducts from agent budget
- Terminates agent when budget exhausted

### 4. Migration Service (`internal/migration`)

**Responsibilities:**
- Coordinate agent migration between nodes
- Package agent for transfer
- Handle incoming migrations
- Ensure single-instance invariant

**Outgoing Flow:**
1. Load checkpoint from storage
2. Package WASM + checkpoint + budget
3. Transfer via libp2p stream
4. Wait for confirmation
5. Terminate local instance
6. Delete local checkpoint

**Incoming Flow:**
1. Receive agent package
2. Save checkpoint to storage
3. Load and initialize agent
4. Resume from checkpoint
5. Send confirmation
6. Register as active agent

### 5. Storage Provider (`internal/storage`)

**Responsibilities:**
- Abstract checkpoint persistence
- Atomic write guarantees
- Agent state isolation

**Interface:**
```go
type Provider interface {
    SaveCheckpoint(ctx, agentID, state) error
    LoadCheckpoint(ctx, agentID) ([]byte, error)
    DeleteCheckpoint(ctx, agentID) error
}
```

**Filesystem Implementation:**
- Atomic writes: temp file → fsync → rename
- Per-agent checkpoint files
- Directory auto-creation
- Idempotent operations

### 6. Configuration (`internal/config`)

**Node Configuration:**
- `NodeID` - Unique identifier (UUID)
- `ListenAddress` - P2P multiaddr
- `PricePerSecond` - Cost per second of execution
- `BootstrapPeers` - Initial peer connections
- `CheckpointDir` - Checkpoint storage location

**Defaults:**
- ListenAddress: `/ip4/0.0.0.0/tcp/4001`
- PricePerSecond: `0.001`
- CheckpointDir: `./checkpoints`

## Data Flow

### Agent Execution Flow

```
Start → LoadAgent → Init → Tick Loop → Checkpoint → Shutdown
                             │            │
                             └─ Meter ────┘
                             └─ Budget ───┘
```

### Migration Flow

```
Node A                          Node B
  │                               │
  ├─ Checkpoint ─────────────────>│
  ├─ Package (WASM+State) ───────>│
  │                               ├─ Store Checkpoint
  │                               ├─ Load Agent
  │                               ├─ Resume
  │<── Confirmation ──────────────┤
  ├─ Terminate Local              │
  ├─ Delete Checkpoint            │
  X                               ● (Agent Running)
```

### Checkpoint Flow

```
Agent State (WASM Memory)
        ↓
  agent_checkpoint()
        ↓
  [budget:8][price:8][state:N]
        ↓
  storage.SaveCheckpoint()
        ↓
  Atomic Write (temp → fsync → rename)
        ↓
  Disk (checkpoints/<agentID>.checkpoint)
```

## Protocol Layers

### Layer 1: P2P Transport (libp2p)

- Peer discovery
- Connection management
- Stream multiplexing
- Protocol negotiation

### Layer 2: Igor Protocols

- Ping: Connectivity testing
- Migration: Agent transfer

### Layer 3: Agent Runtime

- WASM execution
- Budget metering
- Checkpoint/resume
- State management

## Directory Structure

```
cmd/
  igord/              # Main node executable
internal/
  p2p/                # P2P networking
  runtime/            # WASM execution engine
  agent/              # Agent instance management
  migration/          # Migration coordination
  storage/            # Checkpoint storage abstraction
  config/             # Configuration
  logging/            # Structured logging
pkg/
  protocol/           # P2P message types
  manifest/           # Agent manifest schema
agents/
  example/            # Example counter agent
docs/                 # Documentation
```

## Key Design Decisions

### Why WASM?

- Portable across platforms
- Sandboxed by default
- Deterministic execution
- Small binary size
- No runtime dependencies

### Why libp2p?

- Peer-to-peer by design
- No centralized coordination
- Battle-tested in IPFS/Filecoin
- Protocol multiplexing
- NAT traversal support

### Why Storage Abstraction?

- Decouples persistence from filesystem
- Enables future remote storage
- Simplifies migration
- Testable with mock providers

### Why Budget Enforcement?

- Agents pay for resources consumed
- Prevents runaway execution
- Enables economic incentives
- No free-riding on infrastructure

## Scalability Considerations

Igor v0 is **not designed for scale**. It is a proof-of-concept runtime.

Current limits:
- Single agent per node (in `--run-agent` mode)
- No agent multiplexing
- No connection pooling
- No sophisticated peer discovery

Future phases may address scalability, but v0 prioritizes correctness over performance.

## Trust Model

**Nodes do not trust:**
- Other nodes (peers are untrusted)
- Agents (sandboxed execution)

**Agents do not trust:**
- Nodes (assume malicious host)
- Other agents (no coordination)

**Trusted components:**
- WASM sandbox (wazero)
- libp2p transport layer
- Local node configuration

This trust model assumes agents can defend themselves through cryptographic identity and explicit state management.
