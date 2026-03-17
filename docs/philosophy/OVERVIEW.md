# Igor Overview

## What is Igor?

Igor is a runtime for portable, continuous software agents. The checkpoint file IS the agent — copy it anywhere, run `igord resume`, it continues exactly where it left off. Every agent has a DID identity (`did:key:z6Mk...`) and a signed checkpoint lineage providing cryptographic proof of its entire life history.

## The Core Idea

An agent treats its checkpoint as a boundary between processed and unprocessed history. When it resumes after downtime, it detects the gap, replays missed time slots, and discovers what happened while it was absent. The agent does not stay alive — it stays continuous.

The canonical demonstration is the liquidation watcher (`make demo-liquidation`): a DeFi monitoring agent that dies during a critical price drawdown, resumes on a different node, catches up on missed history, and discovers the liquidation threshold was breached during downtime.

## Core Vision

Igor achieves **execution continuity**:

- **Agents own identity** - Each agent has a DID derived from its Ed25519 keypair
- **Agents own state** - State is explicit, checkpointed, and portable
- **Agents own budget** - Agents pay for their own execution
- **Agents survive infrastructure churn** - Execution persists across node failures
- **Agents stay continuous** - Gap-aware catch-up reconstructs missed history

## Fundamental Separation

Igor separates two concerns that are traditionally coupled:

### Runtime (Infrastructure)

The **Igor node** (`igord`) is untrusted infrastructure that:
- Provides sandboxed WASM execution environment
- Meters resource consumption
- Facilitates agent migration (research foundation)
- Charges agents for runtime usage

Nodes are passive infrastructure providers with no special authority.

### Agent (Autonomous Entity)

**Agents** are portable execution packages that:
- Carry cryptographic identity
- Own and manage execution state
- Control their own budget
- Execute in WASM sandboxes
- Survive infrastructure failures
- Detect and replay missed history on resume

## Why Igor Exists

Traditional cloud computing couples execution to infrastructure:
- Applications depend on specific servers
- State lives on specific disks
- Failure of infrastructure means failure of application

Igor inverts this relationship:
- Agents carry their own state
- Agents can resume on any compatible node
- Infrastructure is fungible and untrusted
- Agent continuity is independent of any single node

## What Igor Is NOT

Igor explicitly does **not** implement:

- Agent marketplaces or discovery
- Reputation systems or token economics
- AI / LLM functionality
- Multi-agent coordination frameworks
- Distributed consensus protocols
- Production-grade trustless security (v0 limitation)

These are out of scope by design.

## Current Status

Early product stage. Not production-ready for value-critical workloads.

What works today:
- DID identity (`did:key:z6Mk...`) with Ed25519 keypair
- Checkpoint/resume across machines with same identity
- Gap-aware catch-up (replay missed time slots on resume)
- Signed checkpoint lineage (cryptographic proof of life history)
- CLI: `igord run`, `resume`, `verify`, `inspect`
- Demo agents: liquidation watcher, price tracker, heartbeat, treasury sentinel, deployer

Research foundation (complete): WASM sandboxing, P2P migration, budget metering, replay verification, capability membranes, lease-based authority.

## Technology Stack

- **Runtime**: Go 1.25
- **P2P Transport**: libp2p-go
- **Sandbox**: wazero (WASM runtime, pure Go)
- **Agent Language**: TinyGo → WASM
- **Serialization**: Binary protocols

## Design Philosophy

Igor follows strict principles:

- **Deterministic behavior** over emergent complexity
- **Explicit state** over implicit memory
- **Small testable increments** over large refactors
- **Minimal scope** over feature richness
- **Fail loudly** on invariant violations

## Further Reading

- [Architecture](../runtime/ARCHITECTURE.md) - System structure and components
- [Agent Lifecycle](../runtime/AGENT_LIFECYCLE.md) - How agents execute and survive
- [Migration Protocol](../runtime/MIGRATION_PROTOCOL.md) - How agents move between nodes
- [Budget Model](../runtime/BUDGET_MODEL.md) - How agents pay for execution
- [Security Model](../runtime/SECURITY_MODEL.md) - Sandbox and trust boundaries
- [Runtime Constitution](../constitution/RUNTIME_CONSTITUTION.md) - Constitutional specification root
- [Runtime Enforcement Invariants](../enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md) - System guarantees and enforcement
- [Roadmap](../governance/ROADMAP.md) - Future development phases
