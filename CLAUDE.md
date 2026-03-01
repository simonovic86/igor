# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Igor

Igor is a decentralized runtime for autonomous, survivable software agents. Agents are WASM binaries that checkpoint their state, migrate between peer nodes over libp2p, and pay for execution from internal budgets. Phase 2 (Survival) is complete — agents run, checkpoint, migrate, resume, and meter cost. Research-stage, not production-ready.

**Stack:** Go 1.25 · wazero (pure Go WASM, no CGo) · libp2p-go · TinyGo (agent compilation)

## Build & Development Commands

```bash
make bootstrap       # Install toolchain (Go, golangci-lint, goimports, TinyGo)
make build           # Build igord → bin/igord
make agent           # Build example WASM agent → agents/example/agent.wasm
make test            # Run tests: go test -v ./...
make lint            # golangci-lint (5m timeout)
make vet             # go vet
make fmt             # gofmt + goimports
make check           # fmt-check + vet + lint + test (same as precommit)
make run-agent       # Build + run example agent with budget 1.0
make clean           # Remove bin/, checkpoints/, agent.wasm
```

Run a single test: `go test -v -run TestName ./internal/agent/...`

Run the node manually:
```bash
./bin/igord --run-agent agents/example/agent.wasm --budget 10.0
./bin/igord --migrate-agent local-agent --to /ip4/127.0.0.1/tcp/4002/p2p/<peerID> --wasm agent.wasm
```

## Architecture

### Execution model
Agents export 5 WASM functions: `agent_init`, `agent_tick`, `agent_checkpoint`, `agent_checkpoint_ptr`, `agent_resume` (plus `malloc`). The runtime drives a 1 Hz tick loop. Each tick is budgeted: `cost = elapsed_seconds × price_per_second`. Checkpoints save every 5 seconds. Tick timeout: 100ms.

### Checkpoint format (binary, little-endian)
`[budget: 8 bytes float64][pricePerSecond: 8 bytes float64][agent state: N bytes]`

Atomic writes via temp file → fsync → rename.

### Key packages
- `cmd/igord/` — CLI entry point, flag parsing, tick loop orchestration
- `internal/agent/` — Agent lifecycle: load WASM, init, tick, checkpoint, resume, budget deduction
- `internal/runtime/` — wazero sandbox: 64MB memory limit, WASI with fs/net disabled
- `internal/migration/` — P2P migration over libp2p stream protocol `/igor/migrate/1.0.0`
- `internal/storage/` — `CheckpointProvider` interface + filesystem implementation
- `internal/p2p/` — libp2p host setup, bootstrap peers, protocol handlers
- `pkg/protocol/` — Message types: `AgentPackage`, `AgentTransfer`, `AgentStarted`

### Migration flow
Source checkpoints → packages (WASM + checkpoint + budget) → transfers over libp2p → target instantiates + resumes → target confirms → source terminates + deletes local checkpoint. Single-instance invariant maintained throughout.

## Specification Layers

Igor uses a three-layer specification governance model. Understanding this is critical before proposing changes.

1. **Constitutional** (`docs/constitution/`) — WHAT Igor guarantees. Frozen; changes require RFC + impact statement. Must remain mechanism-agnostic.
2. **Enforcement** (`docs/enforcement/`) — HOW guarantees are upheld. Every rule derives from a constitutional invariant.
3. **Runtime** (`docs/runtime/`) — Implementation details. Must comply with layers above.

Cross-reference: `docs/SPEC_INDEX.md`

## Invariants That Must Never Be Violated

- **EI-1: Single Active Instance** — At most one node ticks an agent at any time
- **EI-3: Checkpoint Lineage Integrity** — Single ordered chain, no forks
- **EI-6: Safety Over Liveness** — Pause rather than violate invariants
- **CM-1: Total Mediation** — All agent I/O passes through runtime hostcalls, no unmediated channels
- **CM-4: Observation Determinism** — Observation hostcalls are deterministically replayable
- **RE-1: Atomic Checkpoints** — Never partial writes
- **RE-3: Budget Conservation** — Budget never created or destroyed, only transferred

Full list: `docs/constitution/EXECUTION_INVARIANTS.md`, `docs/constitution/CAPABILITY_MEMBRANE.md`, `docs/enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md`

## Conventions

- Conventional commits: `<type>(<scope>): <subject>` — types: feat, fix, docs, chore, refactor, test; scopes: runtime, agent, migration, storage, p2p, dev
- Pre-commit hook runs `make check`; CI enforces the same
- Error handling: always check errors, wrap with `fmt.Errorf("context: %w", err)`, log with `slog`
- golangci-lint config: `.golangci.yml` (cyclomatic complexity max 20, errcheck, staticcheck, revive, etc.)
- Spec documents use RFC 2119 keywords (MUST, MUST NOT, MAY, SHOULD)
