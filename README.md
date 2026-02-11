# Igor

**Runtime for Autonomous Economic Agents**

---

## What Igor Is

Igor is a decentralized runtime that enables software agents to survive infrastructure failure, migrate between execution nodes, and pay for their own operation. Agents carry explicit state, internal budgets, and migrate across peer-to-peer networks without centralized coordination.

Igor implements a survival-centric execution model. Agents checkpoint their state explicitly. When infrastructure fails, agents resume from checkpoints. When execution becomes expensive, agents migrate to cheaper nodes. When budget exhausts, agents terminate gracefully. The agent persists; the infrastructure is transient.

## What Igor Is Not

Igor v0 explicitly does **not** implement:

- Agent marketplaces or discovery protocols
- Reputation systems
- Staking, tokens, or blockchain integration
- AI / LLM reasoning capabilities
- Multi-agent coordination frameworks
- Advanced security or cryptographic systems
- Distributed consensus protocols

These are out of scope by design. Igor is a minimal proof-of-survival runtime focused on demonstrating that autonomous agents can checkpoint, migrate, and self-fund execution.

## Core Guarantees

- **Survival:** Agents checkpoint state and resume after node failure
- **Migration:** Agents transfer between nodes over P2P streams (libp2p)
- **Single-instance:** At most one active instance exists at any time
- **Budgets:** Execution metered per-tick, cost deducted from agent budget
- **Sandboxing:** Agents execute in WASM with 64MB memory limit, no filesystem/network access
- **Decentralization:** No centralized coordinator, nodes are symmetric peers

## Architecture

### Agents

WASM executables that implement four lifecycle functions:

- `agent_init()` - Initialize state
- `agent_tick()` - Execute one step (called ~1 Hz)
- `agent_checkpoint()` - Serialize state
- `agent_resume(ptr, len)` - Restore from state

Agents carry budgets and pay for execution time: `cost = duration × price_per_second`

### Nodes

Peer-to-peer participants that:

- Execute agents in wazero WASM sandbox
- Meter resource consumption per tick
- Charge agents per second of execution
- Facilitate migration via libp2p streams
- Operate autonomously without coordination

### Checkpoints

Atomic snapshots containing:

```
[0-7]   Budget (float64)
[8-15]  PricePerSecond (float64)
[16+]   Agent State (application-defined)
```

Checkpoints enable agents to resume after failure or migration.

### Migration

Agents migrate via direct peer-to-peer protocol:

1. Source checkpoints agent
2. Source packages: WASM + checkpoint + budget
3. Source transfers to target via libp2p stream
4. Target resumes agent from checkpoint
5. Target confirms success
6. Source terminates local instance

Single-instance invariant maintained throughout.

## Project Status

**Current:** Phase 1 (Survival) complete  
**Maturity:** Experimental, research-stage  
**Production:** Not ready for production use

All 6 success criteria from [PROJECT_CONTEXT.md](./PROJECT_CONTEXT.md) are met. Phase 1 validates that agents can survive, migrate, and self-fund execution.

**Known limitations:**
- Single-hop migration only (no routing)
- Trusted runtime accounting (no cryptographic receipts)
- Local filesystem storage only
- No agent discovery protocol
- No payment verification

See [docs/SECURITY_MODEL.md](./docs/SECURITY_MODEL.md) for complete threat model.

## Running Locally

### Prerequisites

- Go 1.25.4+
- TinyGo (for building agents)
- golangci-lint (for development)

### Build and Run

```bash
# Build node runtime
make build

# Build example agent
make agent

# Run agent locally
./bin/igord --run-agent agents/example/agent.wasm --budget 10.0
```

The agent will tick every second, checkpointing every 5 seconds, until budget exhausts or interrupted.

### Survival Demo

```bash
# First run
./bin/igord --run-agent agents/example/agent.wasm --budget 1.0
# Agent ticks: 1, 2, 3, 4, 5, 6, 7
# Ctrl+C (checkpoint saved)

# Second run (restart)
./bin/igord --run-agent agents/example/agent.wasm --budget 1.0
# Agent resumes: 8, 9, 10, 11...
```

Agent state survives restart via checkpoint.

## Documentation

**Essential:**
- [PROJECT_CONTEXT.md](./PROJECT_CONTEXT.md) - Authoritative design specification
- [docs/VISION.md](./docs/VISION.md) - Why autonomous software needs survival
- [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) - Runtime implementation details

**Technical:**
- [docs/AGENT_LIFECYCLE.md](./docs/AGENT_LIFECYCLE.md) - Building agents
- [docs/MIGRATION_PROTOCOL.md](./docs/MIGRATION_PROTOCOL.md) - Migration mechanics
- [docs/BUDGET_MODEL.md](./docs/BUDGET_MODEL.md) - Economic model
- [docs/INVARIANTS.md](./docs/INVARIANTS.md) - System guarantees
- [docs/SECURITY_MODEL.md](./docs/SECURITY_MODEL.md) - Threat model

**Development:**
- [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) - Developer setup
- [CONTRIBUTING.md](./CONTRIBUTING.md) - Contribution workflow
- [docs/CI_PIPELINE.md](./docs/CI_PIPELINE.md) - CI documentation
- [docs/ROADMAP.md](./docs/ROADMAP.md) - Future phases

## Development

Install Git hooks for local quality enforcement:

```bash
./scripts/install-hooks.sh
```

Run quality checks before committing:

```bash
make check      # Runs fmt-check, vet, lint, test
make precommit  # Alias for check
```

See [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) for complete developer guide.

## Technology Stack

- **Runtime:** Go 1.25.4
- **WASM Engine:** wazero (pure Go, deterministic sandbox)
- **P2P Transport:** libp2p-go
- **Agent Compilation:** TinyGo → WASM

## Contributing

Contributions welcome. Please read:

- [CONTRIBUTING.md](./CONTRIBUTING.md) - Guidelines
- [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) - Setup
- [PROJECT_CONTEXT.md](./PROJECT_CONTEXT.md) - Design philosophy

Report security issues via [SECURITY.md](./SECURITY.md).

## License

MIT
