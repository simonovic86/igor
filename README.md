# Igor v0

**Experimental decentralized runtime for survivable autonomous agents**

Igor enables software agents to checkpoint state, migrate between peer nodes, and pay for execution using internal budgets. Agents persist independently of any infrastructure provider.

---

## Project Status

**Phase:** Phase 1 (Survival) — Complete  
**Maturity:** Experimental, research-stage  
**Production:** Not ready for production use

Igor v0 demonstrates that agents can survive infrastructure failure through checkpointing and migration. It is a proof-of-concept runtime, intentionally minimal.

**All 6 success criteria met:**

1. ✓ Agent can run on Node A
2. ✓ Agent can checkpoint state explicitly
3. ✓ Agent can migrate to Node B
4. ✓ Agent resumes execution from checkpoint
5. ✓ Agent pays runtime rent to hosting node
6. ✓ No centralized coordination required

---

## Quick Start

### Build

```bash
make build
```

### Run a Node

```bash
./bin/igord
```

### Run an Agent Locally

```bash
./bin/igord --run-agent agents/example/agent.wasm --budget 10.0
```

### Build Example Agent

```bash
make agent
```

---

## Core Concepts

**Agents** are portable WASM executables that:
- Checkpoint their own state explicitly
- Migrate between nodes over P2P networks
- Pay for execution time from internal budgets
- Survive infrastructure failure without external intervention

**Nodes** are peer-to-peer participants that:
- Execute agents in WASM sandboxes (wazero)
- Meter resource consumption
- Charge agents per second of execution
- Facilitate migration via libp2p streams

**Checkpoints** are atomic snapshots containing:
- Agent state (application-defined)
- Budget remaining (float64)
- Price per second (float64)

Checkpoints enable agents to resume after node failure or migration.

---

## Documentation

**Essential:**
- [OVERVIEW.md](./docs/OVERVIEW.md) - What Igor is and does
- [ARCHITECTURE.md](./docs/ARCHITECTURE.md) - How it works
- [VISION.md](./docs/VISION.md) - Why it exists

**Technical:**
- [AGENT_LIFECYCLE.md](./docs/AGENT_LIFECYCLE.md) - Building agents
- [MIGRATION_PROTOCOL.md](./docs/MIGRATION_PROTOCOL.md) - Migration mechanics
- [BUDGET_MODEL.md](./docs/BUDGET_MODEL.md) - Economic model
- [SECURITY_MODEL.md](./docs/SECURITY_MODEL.md) - Threat model
- [INVARIANTS.md](./docs/INVARIANTS.md) - System guarantees

**Process:**
- [DEVELOPMENT.md](./docs/DEVELOPMENT.md) - Developer setup
- [CONTRIBUTING.md](./CONTRIBUTING.md) - How to contribute
- [SECURITY.md](./SECURITY.md) - Security policy
- [ROADMAP.md](./docs/ROADMAP.md) - Future phases

**Authoritative:**
- [PROJECT_CONTEXT.md](./PROJECT_CONTEXT.md) - Design specification

---

## Technology Stack

- **Runtime:** Go 1.22+
- **WASM Engine:** wazero (pure Go sandbox)
- **P2P Transport:** libp2p-go
- **Agent Compilation:** TinyGo → WASM

---

## Design Philosophy

Igor follows strict principles from [PROJECT_CONTEXT.md](./PROJECT_CONTEXT.md):

- **Deterministic behavior** over emergent complexity
- **Explicit state** over implicit memory
- **Small testable increments** over large refactors
- **Minimal scope** over feature richness
- **Fail loudly** on invariant violations

---

## What Igor Is NOT

Igor v0 explicitly does **not** implement:

- Agent marketplaces
- Reputation systems
- Staking or token economics
- AI / LLM functionality
- Multi-agent coordination frameworks
- Advanced security systems
- Distributed consensus

These are out of scope by design.

---

## Development

### Prerequisites

- Go 1.22+
- TinyGo (for building agents)
- golangci-lint
- goimports

See [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) for complete setup.

### Makefile Targets

```bash
make help        # Show all targets
make build       # Build igord
make test        # Run tests
make lint        # Run linters
make fmt         # Format code
make check       # Run all quality checks
make agent       # Build example agent
make run-agent   # Build and run example agent
```

### CI

GitHub Actions runs on every push and PR:
- Formatting check
- Linting (golangci-lint)
- Static analysis (go vet)
- Tests
- Build verification

See [docs/CI_PIPELINE.md](./docs/CI_PIPELINE.md) for details.

---

## Examples

### Counter Agent

A minimal agent that increments a counter, checkpoints state, and survives restarts:

```bash
# First run
./bin/igord --run-agent agents/example/agent.wasm --budget 1.0
# Agent ticks: 1, 2, 3, 4, 5...
# Ctrl+C to stop (checkpoint saved)

# Second run (restart)
./bin/igord --run-agent agents/example/agent.wasm --budget 1.0
# Agent resumes: 6, 7, 8, 9, 10...
```

Source: `agents/example/main.go`

---

## Architecture Highlights

**WASM Sandbox:**
- Memory limited to 64MB per agent
- No filesystem or network access
- Tick timeout at 100ms

**Migration Protocol:**
- Agent packages transferred via libp2p streams
- Single-instance invariant maintained
- Budget transfers with agent

**Budget Model:**
- Execution metered per tick: `cost = duration × price_per_second`
- Budget deducted automatically
- Agent terminates when exhausted

**Storage Abstraction:**
- Checkpoints persist via Provider interface
- Atomic writes guarantee consistency
- Migration-ready design

---

## Known Limitations

- Single-hop migration only (no relay nodes)
- Trusted runtime accounting (no cryptographic proofs)
- Local filesystem storage only
- No agent discovery protocol
- No payment receipt verification

See [SECURITY_MODEL.md](./docs/SECURITY_MODEL.md) for security analysis.

---

## Contributing

Contributions welcome. Please read:

- [CONTRIBUTING.md](./CONTRIBUTING.md) - Contribution workflow
- [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) - Development setup
- [PROJECT_CONTEXT.md](./PROJECT_CONTEXT.md) - Design philosophy

---

## License

MIT (or specify your license)
