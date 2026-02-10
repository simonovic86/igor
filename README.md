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

## Development Status

**Current Phase**: P2P foundation

The repository now includes:
- ✓ Basic P2P networking with libp2p
- ✓ Peer identity and connection management
- ✓ Ping protocol for connectivity testing

Not yet implemented:
- Agent execution (WASM runtime)
- Migration logic
- Payment system
- Agent storage and persistence

## Design Philosophy

Igor v0 follows strict design principles:

- **Deterministic behavior** over emergent complexity
- **Explicit state** over implicit memory
- **Small testable increments** over large refactors
- **Minimal scope** over feature richness
- **Fail loudly** on invariant violations

## Authoritative Design Document

See [PROJECT_CONTEXT.md](./PROJECT_CONTEXT.md) for the complete and authoritative design specification.

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
