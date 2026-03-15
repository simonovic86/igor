# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Igor

Igor is the runtime for portable, immortal software agents. The checkpoint file IS the agent — copy it anywhere, run `igord resume`, it continues exactly where it left off. Every agent has a DID identity (`did:key:z6Mk...`) derived from its Ed25519 keypair, and a signed checkpoint lineage providing cryptographic proof of its entire life history. No infrastructure lock-in.

**Product Phase 1 (Portable Sovereign Agent)** is complete: DID identity, `igord run/resume/verify/inspect` subcommands, checkpoint history archival, lineage chain verification, heartbeat demo agent, portable demo script. Built on a research foundation (Phases 2–5) of WASM sandboxing, P2P migration, budget metering, replay verification, and signed checkpoint lineage.

**Stack:** Go 1.25 · wazero (pure Go WASM, no CGo) · libp2p-go · TinyGo (agent compilation)

## Build & Development Commands

```bash
make bootstrap       # Install toolchain (Go, golangci-lint, goimports, TinyGo)
make build           # Build igord → bin/igord
make agent              # Build example WASM agent → agents/research/example/agent.wasm
make agent-heartbeat    # Build heartbeat WASM agent → agents/heartbeat/agent.wasm
make agent-pricewatcher # Build price watcher WASM agent → agents/pricewatcher/agent.wasm
make agent-sentinel     # Build treasury sentinel WASM agent → agents/sentinel/agent.wasm
make agent-x402buyer   # Build x402 buyer WASM agent → agents/x402buyer/agent.wasm
make agent-deployer    # Build deployer WASM agent → agents/deployer/agent.wasm
make test            # Run tests: go test -v ./...
make lint            # golangci-lint (5m timeout)
make vet             # go vet
make fmt             # gofmt + goimports
make check           # fmt-check + vet + lint + test (same as precommit)
make run-agent       # Build + run example agent with budget 1.0
make demo              # Build + run bridge reconciliation demo
make demo-portable     # Build + run portable agent demo (run → stop → copy → resume → verify)
make demo-pricewatcher # Build + run price watcher demo (fetch prices → stop → resume → verify)
make demo-sentinel     # Build + run treasury sentinel demo (effect lifecycle → crash → reconcile)
make demo-x402         # Build + run x402 payment demo (pay for premium data → crash → reconcile)
make demo-deployer     # Build + run deployer demo (pay → deploy → monitor → crash → reconcile)
make clean           # Remove bin/, checkpoints/, agent.wasm
```

Run a single test: `go test -v -run TestName ./internal/agent/...`

Run manually (new subcommands):
```bash
./bin/igord run --budget 1.0 agents/heartbeat/agent.wasm
./bin/igord resume --checkpoint checkpoints/heartbeat/checkpoint.ckpt --wasm agents/heartbeat/agent.wasm
./bin/igord verify checkpoints/heartbeat/history/
./bin/igord inspect checkpoints/heartbeat/checkpoint.ckpt
```

Legacy mode (P2P/migration):
```bash
./bin/igord --run-agent agents/research/example/agent.wasm --budget 10.0
./bin/igord --migrate-agent local-agent --to /ip4/127.0.0.1/tcp/4002/p2p/<peerID> --wasm agent.wasm
```

## Architecture

### Execution model
Agents export 5 WASM functions: `agent_init`, `agent_tick`, `agent_checkpoint`, `agent_checkpoint_ptr`, `agent_resume`. TinyGo agents provide `malloc` automatically. The runtime drives an adaptive tick loop: 1 Hz default, 10ms fast path when `agent_tick` returns 1 (more work pending). Each tick is budgeted: `cost = elapsed_nanoseconds × price_per_second / 1e9`. Checkpoints save every 5 seconds. Tick timeout: 15s (increased from 100ms to accommodate HTTP hostcalls).

### Checkpoint format (binary, little-endian)
Current version is v0x04 (209-byte header). Supports reading v0x02 (57 bytes) and v0x03 (81 bytes).

`[version: 1 byte (0x04)][budget: 8 bytes int64 microcents][pricePerSecond: 8 bytes int64 microcents][tickNumber: 8 bytes uint64][wasmHash: 32 bytes SHA-256][majorVersion: 8 bytes uint64][leaseGeneration: 8 bytes uint64][leaseExpiry: 8 bytes uint64][prevHash: 32 bytes SHA-256][agentPubKey: 32 bytes Ed25519][signature: 64 bytes Ed25519][agent state: N bytes]`

Header is 209 bytes. Budget uses int64 microcents (1 currency unit = 1,000,000 microcents). WASM hash binds the checkpoint to the binary that created it; mismatch on resume is rejected. prevHash chains checkpoints into a tamper-evident lineage. Signature covers everything except the signature field itself. AgentPubKey encodes as DID: `did:key:z` + base58btc(0xed01 + pubkey).

Atomic writes via temp file → fsync → rename. Every checkpoint is also archived to `history/{agentID}/{tickNumber}.ckpt` for lineage verification.

### Key packages
- `cmd/igord/` — CLI entry point, subcommand dispatch (`run`, `resume`, `verify`, `inspect`), tick loop
- `internal/agent/` — Agent lifecycle: load WASM, init, tick, checkpoint, resume, budget deduction
- `internal/runtime/` — wazero sandbox: 64MB memory limit, WASI with fs/net disabled
- `internal/hostcall/` — `igor` host module: clock, rand, log, wallet, http, x402 payment hostcall implementations
- `internal/inspector/` — Checkpoint inspection and lineage chain verification (`chain.go`: `VerifyChain`)
- `internal/storage/` — `CheckpointProvider` interface + filesystem impl + checkpoint history archival
- `internal/eventlog/` — Per-tick observation event log for deterministic replay
- `internal/replay/` — Deterministic replay verification: single-tick (`ReplayTick`) and chain (`ReplayChain`)
- `internal/runner/` — Tick loop orchestration, divergence escalation policies, lease management
- `internal/authority/` — Lease-based authority epochs, state machine (Active→Expired→RecoveryRequired)
- `internal/migration/` — P2P migration over libp2p stream protocol `/igor/migrate/1.0.0`, retry with backoff
- `internal/registry/` — Peer registry with health tracking for migration target selection
- `internal/p2p/` — libp2p host setup, bootstrap peers, protocol handlers
- `pkg/identity/` — Agent Ed25519 keypair management, DID encoding (`did:key:z6Mk...`), DID parsing
- `pkg/lineage/` — Signed checkpoint types, content hashing, signature verification
- `pkg/manifest/` — Capability manifest parsing and validation
- `pkg/protocol/` — Message types: `AgentPackage`, `AgentTransfer`, `AgentStarted`
- `pkg/receipt/` — Payment receipt data structure, Ed25519 signing, binary serialization
- `sdk/igor/` — Agent SDK: hostcall wrappers (ClockNow, RandBytes, Log, WalletBalance, WalletPay, HTTPRequest), lifecycle plumbing (Agent interface), Encoder/Decoder with Raw/FixedBytes/ReadInto for checkpoint serialization, EffectLog for intent tracking across checkpoint/resume
- `sdk/igor/effects.go` — Effect lifecycle primitives: EffectLog, IntentState (Recorded→InFlight→Confirmed/Unresolved→Compensated), the resume rule (InFlight→Unresolved on Unmarshal)
- `agents/heartbeat/` — Demo agent: logs heartbeat with tick count and age, milestones every 10 ticks
- `agents/pricewatcher/` — Demo agent: fetches BTC/ETH prices from CoinGecko, tracks high/low/latest across checkpoint/resume
- `agents/sentinel/` — Treasury sentinel: monitors simulated treasury balance, triggers refills with effect-safe intent tracking, demonstrates crash recovery and reconciliation
- `agents/x402buyer/` — x402 payment demo: encounters HTTP 402 paywall, pays from budget via wallet_pay hostcall, receives premium data, crash-safe payment reconciliation
- `agents/deployer/` — Deployer demo: pays compute provider, deploys itself, monitors deployment status, multi-step effect-safe crash recovery
- `agents/research/example/` — Original demo agent (Survivor) from research phases
- `agents/research/reconciliation/` — Bridge reconciliation demo agent (research phase)
- `scripts/demo-portable.sh` — End-to-end portable agent demo

### Migration flow
Source checkpoints → packages (WASM + checkpoint + budget) → transfers over libp2p → target instantiates + resumes → target confirms → source terminates + deletes local checkpoint. Single-instance invariant maintained throughout. Failures classified as retriable/fatal/ambiguous; ambiguous transfers enter RECOVERY_REQUIRED state (EI-6). Retry with exponential backoff; peer registry tracks health for target selection.

### CLI subcommands (Product Phase 1)
- `igord run [flags] <agent.wasm>` — run agent with new identity (`--budget`, `--checkpoint-dir`, `--agent-id`)
- `igord resume --checkpoint <path> --wasm <path>` — resume agent from checkpoint file
- `igord verify <history-dir>` — verify checkpoint lineage chain
- `igord inspect <checkpoint.ckpt>` — display checkpoint details with DID identity

### Legacy CLI flags (research/P2P mode)
- `--replay-mode off|periodic|on-migrate|full` — when to run replay verification (default: full)
- `--replay-on-divergence log|pause|intensify|migrate` — escalation policy on divergence (default: log)
- `--verify-interval N` — ticks between verification passes (default: 5)
- `--lease-duration 60s` — authority lease validity period (default: 60s, 0 = disabled)
- `--simulate` — run agent in local simulator mode (no P2P)
- `--inspect-checkpoint <path>` — parse and display checkpoint file

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
