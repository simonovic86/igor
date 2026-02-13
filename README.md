# Igor

**Runtime for Survivable Autonomous Agents**

Igor is a decentralized execution runtime for autonomous software agents. It provides infrastructure primitives enabling agents to checkpoint state, migrate between peer nodes over libp2p, and pay for execution using internal budgets. Agents built on Igor persist independently of any infrastructure provider through WASM sandbox execution, peer-to-peer migration protocols, and runtime economics.

---

## About This Repository

**What:** Experimental infrastructure for autonomous agent survival  
**Status:** Research-stage (Phase 2 complete)  
**Purpose:** Demonstrate that software can checkpoint, migrate, and self-fund execution

**Read first:**
- [ANNOUNCEMENT.md](./ANNOUNCEMENT.md) - Public project introduction
- [PROJECT_CONTEXT.md](./PROJECT_CONTEXT.md) - Authoritative design specification
- [docs/VISION.md](./docs/VISION.md) - Why autonomous software needs survival

**Contribute:**
- [CONTRIBUTING.md](./CONTRIBUTING.md) - Guidelines and workflow
- [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) - Developer setup

---

## Why Igor Exists

Autonomous economic software—including DeFi automation, oracle networks, and AI service agents—can execute decisions independently but cannot survive infrastructure failure autonomously. These distributed systems hold capital, operate continuously, yet remain existentially tied to specific infrastructure.

Trading strategies manage capital without human approval yet stop when servers fail. Oracle participants must maintain continuous uptime yet depend entirely on operators. AI service agents process requests autonomously yet cannot survive the loss of their hosting infrastructure.

Igor provides survivable agent runtime primitives: explicit state checkpointing, peer-to-peer agent migration, and runtime economic metering. Agents persist across distributed infrastructure changes. Infrastructure becomes fungible; agents become persistent.

## Technical Domains

Igor addresses challenges in:

* **Autonomous agent infrastructure** - Runtime for self-managing software entities
* **Survivable distributed systems** - Execution persistence across infrastructure failure
* **WASM sandbox runtimes** - Portable deterministic agent execution (wazero)
* **Peer-to-peer compute mobility** - Agent migration over libp2p networks
* **Runtime economic accounting** - Budget-based execution metering and enforcement
* **Self-hosting autonomous software** - Agents that pay for their own compute
* **Distributed systems research** - Experimental infrastructure for agent survival

## Related Ecosystem Areas

Igor occupies a lower-level runtime infrastructure layer, distinct from agent reasoning or orchestration frameworks.

Relevant adjacent domains:
* Autonomous AI agent infrastructure and runtimes
* Distributed compute execution environments
* WebAssembly execution engines and sandbox runtimes
* Peer-to-peer service fabrics and distributed protocols
* DeFi solver network infrastructure
* Oracle and economic automation service infrastructure
* Mobile code and process migration research

Igor provides execution survival primitives that these higher-level systems could build upon.

## Core Guarantees

- **Survival:** Agents checkpoint state and resume after node failure
- **Migration:** Agents transfer between nodes over libp2p streams
- **Single-instance:** At most one active instance exists (no split-brain)
- **Budget enforcement:** Execution metered per-tick, cost deducted automatically
- **Sandboxing:** WASM isolation, 64MB memory limit, no filesystem/network access
- **Decentralization:** Peer-to-peer coordination, no central authority

## What Igor Does Not Provide

Igor is **not**:

- An AI reasoning framework
- An agent marketplace or discovery protocol
- A blockchain or consensus system
- A multi-agent coordination platform
- A general-purpose orchestration system

Igor is a minimal survival runtime. It implements checkpointing, migration, and budget metering. Nothing more.

## Architecture Overview

```
┌─────────────────────────────────────────────────┐
│  Autonomous Agent (WASM)                        │
│  ┌──────────┐  ┌───────────┐  ┌──────────────┐ │
│  │  Init    │→ │ Tick Loop │→ │  Checkpoint  │ │
│  └──────────┘  └─────┬─────┘  └──────┬───────┘ │
│                      │                │         │
│                      ↓                ↓         │
│              ┌──────────────┐  ┌─────────────┐ │
│              │ Budget       │  │ State       │ │
│              │ Metering     │  │ Persistence │ │
│              └──────────────┘  └─────────────┘ │
└─────────────────────────────────────────────────┘
                       │
                       │ Migration (libp2p)
                       ↓
┌─────────────────────────────────────────────────┐
│  Igor Runtime (Node)                            │
│  ┌──────────┐  ┌───────────┐  ┌──────────────┐ │
│  │  WASM    │  │ Migration │  │  P2P         │ │
│  │  Sandbox │  │ Protocol  │  │  Network     │ │
│  └──────────┘  └───────────┘  └──────────────┘ │
└─────────────────────────────────────────────────┘
```

### Agents

WASM executables implementing four lifecycle functions:

- `agent_init()` - Initialize state
- `agent_tick()` - Execute one step (~1 Hz)
- `agent_checkpoint()` - Serialize state
- `agent_resume(ptr, len)` - Restore from state

Agents carry budgets. Cost calculated per tick: `duration_seconds × price_per_second`

### Nodes

Peer-to-peer participants providing execution services:

- Execute agents in wazero WASM sandbox
- Meter resource consumption per tick
- Charge agents per second of execution
- Facilitate migration via libp2p streams

Nodes operate autonomously without coordination.

### Checkpoints

Atomic snapshots preserving agent state and budget:

```
[0-7]   Budget (float64)
[8-15]  PricePerSecond (float64)
[16+]   Agent State (application-defined)
```

### Migration Protocol

Agents migrate via direct peer-to-peer streams:

1. Source checkpoints agent
2. Source packages: WASM + checkpoint + budget
3. Source transfers to target (libp2p stream)
4. Target resumes agent
5. Target confirms success
6. Source terminates local instance

Single-instance invariant maintained throughout.

## Current Capabilities

**Phase 2 (Survival) - Complete**

All 6 success criteria met:

1. ✓ Agent runs on Node A
2. ✓ Agent checkpoints state explicitly
3. ✓ Agent migrates to Node B
4. ✓ Agent resumes from checkpoint
5. ✓ Agent pays runtime rent
6. ✓ No centralized coordination

## Project Status: Experimental

**Maturity:** Research-stage, proof-of-concept  
**Production:** Not ready for production use  
**Security:** Limited threat model, no cryptographic verification  

**Known limitations:**
- Trusted runtime accounting (no payment receipts)
- Single-hop migration (no routing)
- Local filesystem storage (no distribution)
- No agent discovery protocol
- Minimal security hardening

**Do not use for:**
- Production workloads
- Public network deployments
- Sensitive data
- Financial transactions

**Suitable for:**
- Research and experimentation
- Trusted network environments
- Understanding autonomous agent patterns
- Academic exploration

See [SECURITY.md](./SECURITY.md) for complete security model.

## Quick Start

### Prerequisites

- Go 1.25.4+
- TinyGo 0.40.1+ (for agents)
- golangci-lint (for development)

### Build and Run

```bash
# Build node runtime
make build

# Build example agent
make agent

# Run agent with budget
./bin/igord --run-agent agents/example/agent.wasm --budget 10.0
```

### Survival Demonstration

```bash
# First run
./bin/igord --run-agent agents/example/agent.wasm --budget 1.0
# Agent ticks: 1, 2, 3, 4, 5...
# Ctrl+C (checkpoint saved to ./checkpoints/)

# Second run (restart)
./bin/igord --run-agent agents/example/agent.wasm --budget 1.0
# Agent resumes: 6, 7, 8, 9...
```

Agent state survives restart via checkpoint.

## Documentation

**Start here:**
- [PROJECT_CONTEXT.md](./PROJECT_CONTEXT.md) - Authoritative design specification
- [docs/VISION.md](./docs/VISION.md) - Why autonomous agents need survival
- [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) - Implementation details

**Technical reference:**
- [docs/AGENT_LIFECYCLE.md](./docs/AGENT_LIFECYCLE.md) - Building agents
- [docs/MIGRATION_PROTOCOL.md](./docs/MIGRATION_PROTOCOL.md) - P2P migration
- [docs/BUDGET_MODEL.md](./docs/BUDGET_MODEL.md) - Economic metering
- [docs/RUNTIME_ENFORCEMENT_INVARIANTS.md](./docs/RUNTIME_ENFORCEMENT_INVARIANTS.md) - System guarantees
- [docs/SECURITY_MODEL.md](./docs/SECURITY_MODEL.md) - Threat analysis

**Development:**
- [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) - Setup and workflow
- [CONTRIBUTING.md](./CONTRIBUTING.md) - Contribution guidelines
- [docs/ROADMAP.md](./docs/ROADMAP.md) - Future phases

## Development

### Setup

Install Git hooks for local quality enforcement:

```bash
./scripts/install-hooks.sh
```

### Quality Checks

Run before committing:

```bash
make check      # fmt-check + vet + lint + test
make precommit  # alias for check
```

Pre-commit hooks automatically enforce quality.

See [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) for complete guide.

## Technology

- **Runtime:** Go 1.25.4
- **WASM Engine:** wazero (pure Go, deterministic)
- **P2P:** libp2p-go
- **Agents:** TinyGo → WASM

## Contributing

Contributions welcome. Please read:

- [CONTRIBUTING.md](./CONTRIBUTING.md) - Guidelines and workflow
- [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) - Developer setup
- [PROJECT_CONTEXT.md](./PROJECT_CONTEXT.md) - Design philosophy

Security issues: [SECURITY.md](./SECURITY.md)

---

## Discovery Keywords

**Core Identity:**
Autonomous agent runtime | Survivable distributed systems | WASM agent execution | Peer-to-peer agent migration | Runtime economics | Decentralized execution infrastructure

**Technical Stack:**
WebAssembly sandbox | wazero runtime | libp2p networking | Go distributed systems | Agent checkpoint persistence | Budget metering infrastructure

**Use Cases:**
DeFi automation infrastructure | Oracle network runtime | AI agent execution platform | Economic software agents | Autonomous service infrastructure | Distributed compute mobility

**Research Areas:**
Agent survival primitives | Mobile code execution | Process migration protocols | Runtime accounting systems | Survivable software research | Experimental distributed infrastructure

See [docs/KEYWORDS.md](./docs/KEYWORDS.md) for keyword governance policy.

## License

MIT
