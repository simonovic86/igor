# Igor

**Runtime for Portable, Immortal Software Agents**

Igor makes any WASM program into a sovereign agent with its own identity, memory, and verifiable life history. The checkpoint file IS the agent — copy it anywhere, run `igord resume`, it continues exactly where it left off. No infrastructure lock-in.

---

## About This Repository

**What:** Runtime for portable, infrastructure-independent agents
**Status:** Product Phase 1 complete. Agents have DID identity, checkpoint/resume across machines, and cryptographic lineage verification. Built on a research foundation (Phases 2–5) of WASM sandboxing, P2P migration, budget metering, replay verification, and signed checkpoint lineage.
**Purpose:** Give software agents identity, memory, and continuity — independent of any machine, cloud, or operator

**Read first:**
- [ANNOUNCEMENT.md](./ANNOUNCEMENT.md) - Public project introduction
- [docs/philosophy/OVERVIEW.md](./docs/philosophy/OVERVIEW.md) - Introduction to Igor concepts and status
- [docs/philosophy/VISION.md](./docs/philosophy/VISION.md) - Why autonomous software needs survival

**Contribute:**
- [CONTRIBUTING.md](./CONTRIBUTING.md) - Guidelines and workflow
- [docs/governance/DEVELOPMENT.md](./docs/governance/DEVELOPMENT.md) - Developer setup

---

## Why Igor Exists

Agents today are tied to their infrastructure. Kill the server, the agent dies. Restart it, and it has to start from scratch — losing in-memory state, execution history, and continuity.

Kubernetes restarts processes but loses state. Temporal forces you into a workflow programming model. AO replays entire message histories from Arweave. An LLM with a wallet can rent a server but can't survive dying on it.

Igor gives agents three things nothing else provides together: **identity** (DID), **memory** (checkpointed state that survives infrastructure failure), and **verifiable continuity** (cryptographic proof of the agent's entire life history). The agent is a portable digital object — not a deployment tied to specific infrastructure.

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

- **Portable:** The checkpoint file IS the agent — copy it anywhere, resume it
- **Identity:** Every agent has a DID (`did:key:z6Mk...`) derived from its Ed25519 keypair
- **Verifiable:** Signed checkpoint lineage — cryptographic proof of entire life history
- **Survival:** Agents checkpoint state and resume after infrastructure failure
- **Sandboxing:** WASM isolation, 64MB memory limit, no filesystem/network access
- **Migration:** Agents transfer between nodes over libp2p streams (research foundation)

## What Igor Does Not Provide

Igor is **not**:

- An AI reasoning framework
- An agent marketplace or discovery protocol
- A blockchain or consensus system
- A multi-agent coordination platform
- A general-purpose orchestration system

Igor is a minimal runtime for portable, sovereign agents. It provides identity, checkpointing, and verifiable continuity. Nothing more.

## Architecture Overview

```
┌─────────────────────────────────────────────────┐
│  Agent (WASM binary)                            │
│  ┌──────────┐  ┌───────────┐  ┌──────────────┐ │
│  │  Init    │→ │ Tick Loop │→ │  Checkpoint  │ │
│  └──────────┘  └───────────┘  └──────────────┘ │
└─────────────────────────────────────────────────┘
         │                             │
         ↓                             ↓
┌─────────────────┐  ┌────────────────────────────┐
│  Igor Runtime   │  │  Checkpoint File (.ckpt)   │
│  (igord)        │  │  ┌──────────────────────┐  │
│  ┌───────────┐  │  │  │ Identity (Ed25519)   │  │
│  │ WASM      │  │  │  │ State + Budget       │  │
│  │ Sandbox   │  │  │  │ Signed Lineage       │  │
│  └───────────┘  │  │  │ WASM Hash Binding    │  │
│  ┌───────────┐  │  │  └──────────────────────┘  │
│  │ Hostcalls │  │  │  ↕ Copy to any machine     │
│  └───────────┘  │  │  ↕ igord resume → continues │
└─────────────────┘  └────────────────────────────┘
```

### Agents

WASM executables implementing five lifecycle functions:

- `agent_init()` - Initialize state
- `agent_tick()` - Execute one step (~1 Hz)
- `agent_checkpoint()` - Return checkpoint size
- `agent_checkpoint_ptr()` - Return pointer to checkpoint data
- `agent_resume(ptr, len)` - Restore from state

Agents interact with the runtime through the `igor` host module (clock, rand, log hostcalls), mediated by a capability manifest declared at load time.

### Checkpoints

The checkpoint file IS the agent. It contains everything needed to resume:

- **Identity:** Ed25519 public key → DID (`did:key:z6Mk...`)
- **State:** Application-defined agent memory
- **Budget:** Remaining execution budget in microcents
- **Lineage:** PrevHash chain + Ed25519 signature for tamper-evident history
- **Binding:** SHA-256 hash of the WASM binary that created it

Every checkpoint is archived to `history/{agentID}/{tickNumber}.ckpt` for full lineage verification.

## Current Capabilities

**Product Phase 1 (Portable Sovereign Agent) - Complete**

- Agent runs with DID identity (`did:key:z6Mk...`)
- Agent checkpoints and resumes on any machine — same DID, continuous tick count
- Signed checkpoint lineage — cryptographic proof of entire life history
- Checkpoint history archival for lineage verification
- CLI subcommands: `igord run`, `resume`, `verify`, `inspect`

**Research Foundation (Phases 2–5) - Complete**

- WASM sandboxing, P2P migration, budget metering, replay verification
- Capability membranes, lease-based authority, signed lineage, migration failure recovery

## Project Status

**Maturity:** Product Phase 1 complete. Built on research foundation (Phases 2–5).
**Production:** Not yet production-ready — early product stage.
**Security:** Ed25519 signed checkpoint lineage with DID identity.

**Known limitations:**
- Local filesystem storage only (no permanent archival yet)
- No HTTP or payment hostcalls (agents can't call external APIs yet)
- No self-provisioning (agents can't deploy themselves yet)
- Minimal security hardening

**Suitable for:**
- Building and running portable agents
- Experimenting with agent identity and continuity
- Understanding infrastructure-independent agent patterns

See [SECURITY.md](./SECURITY.md) for complete security model.

## Quick Start

### Prerequisites

- Go 1.25.4+
- TinyGo 0.40.1+ (for agents)
- golangci-lint (for development)

### Build and Run

```bash
# Build runtime and heartbeat agent
make build
make agent-heartbeat

# Run agent (creates identity, starts ticking)
./bin/igord run --budget 1.0 agents/heartbeat/agent.wasm
# [heartbeat] tick=1 age=1s
# [heartbeat] tick=2 age=2s
# Ctrl+C → checkpoint saved

# Resume on same or different machine
./bin/igord resume checkpoints/heartbeat/checkpoint.ckpt agents/heartbeat/agent.wasm
# [heartbeat] tick=3 age=3s  ← continues where it left off

# Verify the agent's entire life history
./bin/igord verify checkpoints/heartbeat/history/

# Inspect a checkpoint
./bin/igord inspect checkpoints/heartbeat/checkpoint.ckpt
```

### Portable Agent Demo

```bash
# Full demo: run → stop → copy → resume → verify
make demo-portable
```

The demo shows an agent running on "Machine A", checkpoint copied to "Machine B", resuming with the same DID identity and continuous tick count, then verifying the cryptographic lineage across both machines.

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
Portable agent runtime | Immortal software agents | Agent identity (did:key) | Sovereign agents | Infrastructure-independent agents | WASM agent execution | Verifiable agent continuity

**Technical Stack:**
WebAssembly sandbox | wazero runtime | Ed25519 signed lineage | DID identity | Go distributed systems | Agent checkpoint persistence | TinyGo WASM agents

**Use Cases:**
Long-running autonomous agents | Self-provisioning compute | AI agent execution platform | Portable stateful agents | Agent survival across infrastructure | Decentralized agent deployment

**Research Foundation:**
Agent survival primitives | Mobile code execution | Process migration protocols | P2P agent migration | Runtime accounting systems | Survivable software research

See [docs/governance/KEYWORDS.md](./docs/governance/KEYWORDS.md) for keyword governance policy.

## License

Apache-2.0 — see [LICENSE](./LICENSE) for details.
