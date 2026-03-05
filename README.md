# Igor

**Runtime for Survivable Autonomous Agents**

Igor is a decentralized execution runtime for autonomous software agents. It provides infrastructure primitives enabling agents to checkpoint state, migrate between peer nodes over libp2p, and pay for execution using internal budgets. Agents built on Igor persist independently of any infrastructure provider through WASM sandbox execution, peer-to-peer migration protocols, and runtime economics.

---

## About This Repository

**What:** Experimental infrastructure for autonomous agent survival  
**Status:** Research-stage — Phases 2–4 complete, Phase 5 (Hardening) complete. Agents run, checkpoint, migrate, resume, meter cost, enforce capability membranes, replay-verify, support multi-node chain migration, sign checkpoint lineage, recover from migration failures, and enforce lease-based authority. Task 15 (Permissionless Hardening) next.
**Purpose:** Demonstrate that software can checkpoint, migrate, and self-fund execution

**Read first:**
- [ANNOUNCEMENT.md](./ANNOUNCEMENT.md) - Public project introduction
- [docs/philosophy/OVERVIEW.md](./docs/philosophy/OVERVIEW.md) - Introduction to Igor concepts and status
- [docs/philosophy/VISION.md](./docs/philosophy/VISION.md) - Why autonomous software needs survival

**Contribute:**
- [CONTRIBUTING.md](./CONTRIBUTING.md) - Guidelines and workflow
- [docs/governance/DEVELOPMENT.md](./docs/governance/DEVELOPMENT.md) - Developer setup

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

WASM executables implementing five lifecycle functions:

- `agent_init()` - Initialize state
- `agent_tick()` - Execute one step (~1 Hz)
- `agent_checkpoint()` - Return checkpoint size
- `agent_checkpoint_ptr()` - Return pointer to checkpoint data
- `agent_resume(ptr, len)` - Restore from state

Agents interact with the runtime through the `igor` host module (clock, rand, log hostcalls), mediated by a capability manifest declared at load time.

Agents carry budgets. Cost calculated per tick: `duration_seconds × price_per_second`

### Nodes

Peer-to-peer participants providing execution services:

- Execute agents in wazero WASM sandbox
- Meter resource consumption per tick
- Charge agents per second of execution
- Facilitate migration via libp2p streams

Nodes operate autonomously without coordination.

### Checkpoints

Atomic snapshots preserving agent state, budget, and binary identity (209-byte header):

```
Offset  Size  Field
0       1     Version (0x04)
1       8     Budget (int64 microcents, little-endian)
9       8     PricePerSecond (int64 microcents, little-endian)
17      8     TickNumber (uint64, little-endian)
25      32    WASMHash (SHA-256 of agent binary)
57      8     MajorVersion (uint64, little-endian)
65      8     LeaseGeneration (uint64, little-endian)
73      8     LeaseExpiry (int64, little-endian)
81      32    PrevHash (SHA-256 of previous checkpoint)
113     32    AgentPubKey (Ed25519 public key)
145     64    Signature (Ed25519, covers bytes 0–144)
209     N     Agent State (application-defined)
```

Budget unit: 1 currency unit = 1,000,000 microcents. Integer arithmetic avoids float precision drift.

### Migration Protocol

Agents migrate via direct peer-to-peer streams:

1. Source checkpoints agent
2. Source packages: WASM + checkpoint + budget + manifest + WASM hash
3. Source includes replay verification data (if available)
4. Source transfers to target (libp2p stream)
5. Target verifies WASM hash integrity
6. Target replays last tick to verify checkpoint (if replay data present)
7. Target resumes agent
8. Target confirms success
9. Source terminates local instance

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

**Maturity:** Research-stage, proof-of-concept (Phase 3 complete, Phase 4 next)
**Production:** Not ready for production use  
**Security:** WASM hash identity binding; no full cryptographic attestation yet  

**Known limitations:**
- Trusted runtime accounting (no payment receipts)
- Chain migration tested (A→B→C→A) but no routing protocol
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

## Specification Overview

Igor's specification is organized into layered authority domains. See the full [Specification Index](./docs/SPEC_INDEX.md) for cross-references.

| Layer | Purpose | Location |
|-------|---------|----------|
| **Constitution** | WHAT Igor guarantees | [docs/constitution/](./docs/constitution/) |
| **Enforcement** | HOW guarantees are upheld | [docs/enforcement/](./docs/enforcement/) |
| **Runtime** | HOW Igor operates | [docs/runtime/](./docs/runtime/) |
| **Governance** | HOW Igor evolves | [docs/governance/](./docs/governance/) |
| **Philosophy** | WHY Igor exists | [docs/philosophy/](./docs/philosophy/) |

## Documentation

**Start here:**
- [docs/philosophy/OVERVIEW.md](./docs/philosophy/OVERVIEW.md) - Introduction to Igor concepts and status
- [docs/philosophy/VISION.md](./docs/philosophy/VISION.md) - Why autonomous agents need survival
- [docs/runtime/ARCHITECTURE.md](./docs/runtime/ARCHITECTURE.md) - Implementation details

**Technical reference:**
- [docs/runtime/AGENT_LIFECYCLE.md](./docs/runtime/AGENT_LIFECYCLE.md) - Building agents
- [docs/runtime/MIGRATION_PROTOCOL.md](./docs/runtime/MIGRATION_PROTOCOL.md) - P2P migration
- [docs/runtime/BUDGET_MODEL.md](./docs/runtime/BUDGET_MODEL.md) - Economic metering
- [docs/enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md](./docs/enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md) - System guarantees
- [docs/runtime/SECURITY_MODEL.md](./docs/runtime/SECURITY_MODEL.md) - Threat analysis

**Development:**
- [docs/governance/DEVELOPMENT.md](./docs/governance/DEVELOPMENT.md) - Setup and workflow
- [CONTRIBUTING.md](./CONTRIBUTING.md) - Contribution guidelines
- [docs/governance/ROADMAP.md](./docs/governance/ROADMAP.md) - Future phases

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

See [docs/governance/DEVELOPMENT.md](./docs/governance/DEVELOPMENT.md) for complete guide.

## Technology

- **Runtime:** Go 1.25.4
- **WASM Engine:** wazero (pure Go, deterministic)
- **P2P:** libp2p-go
- **Agents:** TinyGo → WASM

## Contributing

Contributions welcome. Please read:

- [CONTRIBUTING.md](./CONTRIBUTING.md) - Guidelines and workflow
- [docs/governance/DEVELOPMENT.md](./docs/governance/DEVELOPMENT.md) - Developer setup
- [docs/philosophy/OVERVIEW.md](./docs/philosophy/OVERVIEW.md) - Design philosophy

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

See [docs/governance/KEYWORDS.md](./docs/governance/KEYWORDS.md) for keyword governance policy.

## License

Apache-2.0 — see [LICENSE](./LICENSE) for details.
