# Igor v0 — Autonomous Mobile Agent Runtime

Igor v0 is an experimental distributed systems runtime that enables portable autonomous agents to migrate between peer nodes and execute independently.

## Core Concept

Igor separates two fundamental concerns:

### Runtime (Node)

The **Igor runtime** (`igord`) is untrusted infrastructure. Each node:

- Provides sandboxed execution environment for agents
- Meters resource consumption (CPU, memory, storage)
- Facilitates peer-to-peer agent migration
- Charges agents for runtime usage
- Discovers and communicates with peer nodes

Nodes are passive infrastructure providers. They do not coordinate or have special authority.

### Agent

**Agents** are portable, autonomous execution packages that:

- Carry their own cryptographic identity
- Own and manage their execution state
- Control their own budget
- Decide when and where to migrate
- Execute in WASM sandboxes
- Survive infrastructure churn

Agents are active entities. They decide, nodes comply.

## Success Criteria

Igor v0 is complete when:

1. ✓ An agent can run on Node A
2. ✓ The agent can checkpoint its state explicitly
3. ✓ The agent can migrate to Node B
4. ✓ The agent resumes execution from checkpoint on Node B
5. ✓ The agent pays runtime rent to the hosting node
6. ✓ No centralized coordination is required

## Technology Stack

- **Runtime Language**: Go 1.22+
- **P2P Transport**: libp2p-go
- **Sandbox**: wazero (WASM runtime)
- **Agent Language**: TinyGo → WASM

## Project Structure

```
cmd/
  igord/           # Node runtime executable
internal/
  p2p/             # Peer-to-peer networking
  runtime/         # Agent execution sandbox
  agent/           # Agent lifecycle management
  migration/       # Migration coordination
  payment/         # Payment accounting
  config/          # Configuration loading
  logging/         # Structured logging
pkg/
  protocol/        # P2P protocol messages
  manifest/        # Agent manifest model
agents/
  example/         # Example agent implementations
docs/              # Additional documentation
```

## Building

```bash
go build -o bin/igord ./cmd/igord
```

## Running

```bash
./bin/igord
```

The node will start and display:

```
Igor Node starting...
NodeID: <uuid>
P2P node created peer_id=<peer_id>
Listening on address=/ip4/127.0.0.1/tcp/4001/p2p/<peer_id>
Listening on address=/ip4/<your_ip>/tcp/4001/p2p/<peer_id>
Igor Node ready
```

## P2P Networking

Igor nodes use **libp2p** for peer-to-peer communication. Each node:

- Generates a persistent peer identity on startup
- Listens for incoming connections on a configurable multiaddr
- Can connect to other nodes via bootstrap peers
- Implements the `/igor/ping/1.0.0` protocol for connectivity testing

### Peer Identity

Each Igor node has a unique libp2p peer ID derived from its cryptographic keypair. This identity is automatically generated on node startup.

### Connectivity Testing

To test connectivity between two nodes:

**Terminal A - Start first node:**
```bash
./bin/igord
```

Note the peer multiaddr from the logs, for example:
```
Listening on address=/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWABC...
```

**Terminal B - Start second node with bootstrap peer:**

Currently, bootstrap peers must be configured programmatically in the config. In a future iteration, command-line flags will be supported.

When nodes connect successfully, you'll see logs indicating:
```
Connecting to peer peer_id=12D3KooWABC...
Ping successful peer_id=12D3KooWABC...
```

### Ping Protocol

The `/igor/ping/1.0.0` protocol is the first connectivity test:

- Client sends "ping" (4 bytes)
- Server responds "pong" (4 bytes)
- Logs confirm successful peer interaction

This protocol verifies that:
- Nodes can establish libp2p connections
- Protocol negotiation works correctly
- Basic stream communication functions

## Agent Runtime

Igor nodes execute autonomous agents in a sandboxed WASM environment using **wazero**.

### Running an Agent Locally

```bash
./bin/igord --run-agent agents/example/agent.wasm
```

The runtime will:
- Load and compile the WASM module
- Initialize the agent
- Execute tick() every second
- Checkpoint state every 5 seconds
- Save checkpoint to disk
- Resume from checkpoint on restart

### Agent Lifecycle

Agents must implement four lifecycle functions:

**`agent_init()`**
Called once when the agent first starts. Initialize state here.

**`agent_tick()`**
Called periodically (every 1 second). Must be short-lived (<100ms timeout enforced).

**`agent_checkpoint() -> size`**
Returns the size of serialized state. State data location provided by `agent_checkpoint_ptr()`.

**`agent_checkpoint_ptr() -> ptr`**
Returns pointer to checkpoint data in WASM memory.

**`agent_resume(ptr, size)`**
Restores agent from previously checkpointed state.

### Sandbox Constraints

The WASM runtime enforces:
- **Memory limit**: 64MB per agent
- **No filesystem access**: Agents cannot read/write files
- **No network access**: Agents cannot make network calls
- **Execution timeout**: Each tick must complete within 100ms

### Example Agent

See `agents/example/` for a minimal counter agent that:
- Maintains a counter in state
- Increments on each tick
- Checkpoints state
- Survives restarts

Build with TinyGo:
```bash
cd agents/example
make build
```

### Survival Test

**First run:**
```bash
./bin/igord --run-agent agents/example/agent.wasm
# Agent ticks: 1, 2, 3, 4, 5, 6, 7
# Ctrl+C to stop
```

**Second run (restart):**
```bash
./bin/igord --run-agent agents/example/agent.wasm
# Agent resumes from counter=7
# Agent continues: 8, 9, 10, 11...
```

The agent survives infrastructure restarts by checkpointing state to disk and resuming automatically.

## Checkpoint Storage Abstraction

Igor decouples agent state persistence from the storage implementation through the `storage.Provider` interface. This architectural pattern enables:

- **Migration-ready state handling**: Agent checkpoints can be transferred between nodes
- **Pluggable storage backends**: Filesystem, remote storage, or distributed systems
- **Atomic writes**: Guaranteed consistency through temp-file-fsync-rename pattern
- **Clean separation of concerns**: Agent runtime doesn't know about filesystems

### Storage Provider Interface

```go
type Provider interface {
    SaveCheckpoint(ctx context.Context, agentID string, state []byte) error
    LoadCheckpoint(ctx context.Context, agentID string) ([]byte, error)
    DeleteCheckpoint(ctx context.Context, agentID string) error
}
```

### Filesystem Provider

The reference implementation (`FSProvider`) stores checkpoints locally:

- **Location**: `./checkpoints/<agentID>.checkpoint`
- **Atomic writes**: Uses temp file + fsync + rename to prevent partial state
- **Directory auto-creation**: Creates checkpoint directory if missing
- **Logging**: All operations logged with agent ID

### Configuration

Set checkpoint directory in config (default: `./checkpoints`):

```go
cfg := &config.Config{
    CheckpointDir: "./checkpoints",
    // ...
}
```

### Architecture Benefits

The storage abstraction prepares Igor for migration by ensuring:

1. **No direct file I/O in agent runtime**: All persistence through Provider interface
2. **Testable**: Easy to mock storage for unit tests
3. **Extensible**: Can add remote storage (S3, IPFS, libp2p) without changing agent code
4. **Atomic guarantees**: fsync ensures durability before visibility

Future migration will use the same interface to transfer checkpoints between nodes over libp2p.

## Development Status

**Current Phase**: Phase 1 Complete ✅

Igor v0 has completed all Phase 1 (Survival) tasks:

- ✅ P2P networking with libp2p
- ✅ Peer identity and connection management
- ✅ Connectivity testing (ping protocol)
- ✅ WASM agent runtime with wazero
- ✅ Agent lifecycle (init, tick, checkpoint, resume)
- ✅ Local agent execution with survival through restarts
- ✅ Sandboxed execution (memory limits, timeouts, no filesystem/network)
- ✅ Checkpoint storage abstraction
- ✅ Agent migration between nodes over P2P
- ✅ Runtime rent metering and budget enforcement

**All 6 success criteria from PROJECT_CONTEXT.md are met:**

1. ✅ Agent can run on Node A
2. ✅ Agent can checkpoint state explicitly
3. ✅ Agent can migrate to Node B
4. ✅ Agent resumes execution from checkpoint
5. ✅ Agent pays runtime rent to hosting node
6. ✅ System has no centralized coordination

**Next Phase:** Autonomy - Enable agents to make migration decisions independently.

## Design Philosophy

Igor v0 follows strict design principles:

- **Deterministic behavior** over emergent complexity
- **Explicit state** over implicit memory
- **Small testable increments** over large refactors
- **Minimal scope** over feature richness
- **Fail loudly** on invariant violations

## Documentation

### Core Documents

- [PROJECT_CONTEXT.md](./PROJECT_CONTEXT.md) - Authoritative design specification
- [TASKS.md](./TASKS.md) - Development task tracker

### Detailed Documentation

- [Overview](./docs/OVERVIEW.md) - System introduction and quick start
- [Architecture](./docs/ARCHITECTURE.md) - System structure and components
- [Agent Lifecycle](./docs/AGENT_LIFECYCLE.md) - How agents execute and survive
- [Migration Protocol](./docs/MIGRATION_PROTOCOL.md) - How agents move between nodes
- [Budget Model](./docs/BUDGET_MODEL.md) - How agents pay for execution
- [Security Model](./docs/SECURITY_MODEL.md) - Sandbox and trust boundaries
- [Invariants](./docs/INVARIANTS.md) - System guarantees and constraints
- [Roadmap](./docs/ROADMAP.md) - Future development phases

## What Igor v0 Is NOT

Igor v0 explicitly does **not** implement:

- Agent marketplaces
- Reputation systems
- Staking or token economics
- AI / LLM functionality
- Multi-agent coordination frameworks
- Advanced security systems
- Distributed consensus

These are out of scope.

## License

MIT (or specify your license)
