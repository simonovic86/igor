# Igor

**Runtime for Survivable Autonomous Agents**

Igor is a decentralized runtime enabling software agents to checkpoint state, migrate between peer nodes, and pay for their own execution. Agents persist independently of any infrastructure provider.

---

## Why Igor Exists

Autonomous economic software can execute decisions independently but cannot survive infrastructure failure independently.

Trading strategies manage capital without human approval yet stop when servers fail. Oracle participants must maintain continuous uptime yet depend entirely on operators. AI service agents process requests autonomously yet cannot survive the loss of their hosting infrastructure.

Igor provides survival primitives: explicit state checkpointing, peer-to-peer migration, and runtime budget enforcement. Agents persist across infrastructure changes. Infrastructure becomes fungible; agents become persistent.

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

## Architecture

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

**Phase 1 (Survival) - Complete**

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
- [docs/INVARIANTS.md](./docs/INVARIANTS.md) - System guarantees
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

## License

MIT
