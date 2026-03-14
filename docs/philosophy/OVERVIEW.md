# Igor v0 Overview

## What is Igor?

Igor v0 is an experimental distributed systems runtime that enables portable autonomous agents to migrate between peer nodes and execute independently.

## Core Vision

Igor achieves **true execution autonomy**:

- **Agents own identity** - Each agent has its own cryptographic identity
- **Agents own state** - State is explicit and agent-controlled
- **Agents own budget** - Agents pay for their own execution
- **Agents decide where to run** - Migration is agent-initiated
- **Agents survive infrastructure churn** - Execution persists across node failures

## Fundamental Separation

Igor separates two concerns that are traditionally coupled:

### Runtime (Infrastructure)

The **Igor node** (`igord`) is untrusted infrastructure that:
- Provides sandboxed WASM execution environment
- Meters resource consumption
- Facilitates P2P agent migration
- Charges agents for runtime usage
- Discovers and communicates with peer nodes

Nodes are passive infrastructure providers with no special authority.

### Agent (Autonomous Entity)

**Agents** are portable execution packages that:
- Carry cryptographic identity
- Own and manage execution state
- Control their own budget
- Decide when and where to migrate
- Execute in WASM sandboxes
- Survive infrastructure failures

Agents are active entities. They decide, nodes comply.

## Why Igor Exists

Traditional cloud computing couples execution to infrastructure:
- Applications depend on specific servers
- State lives on specific disks
- Failure of infrastructure means failure of application

Igor inverts this relationship:
- Agents carry their own state
- Agents can migrate to any compatible node
- Infrastructure is fungible and untrusted
- Agent survival is independent of any single node

## What Igor Is NOT

Igor v0 explicitly does **not** implement:

- Agent marketplaces or discovery
- Reputation systems
- Staking or token economics
- AI / LLM functionality
- Multi-agent coordination frameworks
- Advanced security systems
- Distributed consensus protocols

These are out of scope by design.

## Current Status (Phase 2 Complete)

Igor v0 Phase 2 implements:

✅ P2P networking with libp2p  
✅ WASM agent sandbox execution  
✅ Agent lifecycle (init, tick, checkpoint, resume)  
✅ Agent migration between nodes  
✅ Runtime rent metering and budget enforcement  
✅ Checkpoint storage abstraction  

Phase 2 proves that **agents can survive and migrate autonomously**.

## Technology Stack

- **Runtime**: Go 1.22+
- **P2P Transport**: libp2p-go
- **Sandbox**: wazero (WASM runtime)
- **Agent Language**: TinyGo → WASM
- **Serialization**: Binary protocols (JSON for P2P messages)

## Design Philosophy

Igor follows strict principles:

- **Deterministic behavior** over emergent complexity
- **Explicit state** over implicit memory
- **Small testable increments** over large refactors
- **Minimal scope** over feature richness
- **Fail loudly** on invariant violations

## Quick Start

### Run a Node

```bash
go build -o bin/igord ./cmd/igord
./bin/igord
```

### Run an Agent

```bash
./bin/igord --run-agent agents/research/example/agent.wasm --budget 10.0
```

### Migrate an Agent

```bash
# Terminal A: Start receiving node
./bin/igord

# Terminal B: Migrate agent (after checkpoint exists)
./bin/igord \
  --migrate-agent local-agent \
  --to /ip4/127.0.0.1/tcp/4002/p2p/<peer_id> \
  --wasm agents/research/example/agent.wasm
```

## Further Reading

- [Architecture](../runtime/ARCHITECTURE.md) - System structure and components
- [Agent Lifecycle](../runtime/AGENT_LIFECYCLE.md) - How agents execute and survive
- [Migration Protocol](../runtime/MIGRATION_PROTOCOL.md) - How agents move between nodes
- [Budget Model](../runtime/BUDGET_MODEL.md) - How agents pay for execution
- [Security Model](../runtime/SECURITY_MODEL.md) - Sandbox and trust boundaries
- [Runtime Constitution](../constitution/RUNTIME_CONSTITUTION.md) - Constitutional specification root
- [Runtime Enforcement Invariants](../enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md) - System guarantees and enforcement
- [Roadmap](../governance/ROADMAP.md) - Future development phases
