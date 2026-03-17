# Igor

**This agent did not stay alive. It stayed continuous.**

An agent monitors a DeFi position for liquidation risk. Node A dies. The price keeps moving. The threshold is breached while the agent is absent. Node B picks up the checkpoint, detects the gap, replays the missed time slots, and discovers what happened during downtime. Same DID identity. Cryptographic proof of the entire life history. No state lost.

```bash
make demo-liquidation    # See it happen in 60 seconds
```

---

Igor is a runtime for portable, continuous software agents. The checkpoint file IS the agent — copy it anywhere, run `igord resume`, it continues exactly where it left off. Every agent has a DID identity (`did:key:z6Mk...`), and a signed checkpoint lineage providing cryptographic proof of its entire life history.

**Status:** Early product stage. Agents have DID identity, checkpoint/resume across machines, gap-aware catch-up, and cryptographic lineage verification. Not production-ready for value-critical workloads.

## Why Igor Exists

Agents today are tied to their infrastructure. Kill the server, the agent dies. Restart it, and it starts from scratch — losing state, execution history, and continuity.

Igor gives agents three things nothing else provides together: **identity** (DID), **memory** (checkpointed state that survives infrastructure failure), and **verifiable continuity** (cryptographic proof of the agent's entire life history). The agent is a portable digital object — not a deployment tied to specific infrastructure.

## Core Guarantees

- **Continuous:** Agents detect gaps in their history and reconstruct what they missed
- **Portable:** The checkpoint file IS the agent — copy it anywhere, resume it
- **Identity:** Every agent has a DID (`did:key:z6Mk...`) derived from its Ed25519 keypair
- **Verifiable:** Signed checkpoint lineage — cryptographic proof of entire life history
- **Sandboxed:** WASM isolation, 64MB memory limit, no filesystem/network access

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

## Project Status

Early product stage. Not production-ready for value-critical workloads.

What works today: DID identity, checkpoint/resume across machines, gap-aware catch-up, signed checkpoint lineage, CLI (`igord run/resume/verify/inspect`). Built on a research foundation of WASM sandboxing, P2P migration, budget metering, and replay verification.

See [SECURITY.md](./SECURITY.md) for the complete security model and known limitations.

## Quick Start

### Prerequisites

- Go 1.25.4+
- TinyGo 0.40.1+ (for agents)
- golangci-lint (for development)

### The Canonical Demo

```bash
make demo-liquidation
```

This runs the full continuity proof: agent starts on Node A monitoring a simulated ETH position, Node A dies during a critical price drawdown, Node B resumes from checkpoint, detects the gap, replays missed time slots, discovers the threshold was breached during downtime, and verifies the cryptographic lineage across both nodes.

### Manual Usage

```bash
# Build runtime and agent
make build
make agent-liquidation

# Run agent (creates DID identity, starts monitoring)
./bin/igord run --budget 100.0 agents/liquidation/agent.wasm

# Resume on same or different machine
./bin/igord resume --checkpoint checkpoints/liquidation/liquidation.checkpoint \
    --wasm agents/liquidation/agent.wasm

# Verify the agent's entire life history
./bin/igord verify checkpoints/liquidation/history/

# Inspect a checkpoint
./bin/igord inspect checkpoints/liquidation/liquidation.checkpoint
```

### Other Demos

```bash
make demo-portable      # Basic portable resume (heartbeat agent)
make demo-pricewatcher  # Price tracking across resume
make demo-sentinel      # Effect-safe crash recovery
make demo-x402          # Payment with crash reconciliation
make demo-deployer      # Multi-step deployment with crash recovery
```

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
