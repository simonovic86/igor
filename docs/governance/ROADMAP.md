# Igor Roadmap

## Current Status: Product Phase 2.5 Next

Igor has completed its research foundation (**Phases 2–5**) and product phases 1–2. The runtime supports **portable, infrastructure-independent agents** with DID identity, cryptographic lineage verification, effect-safe side effects, HTTP/payment hostcalls, and clean CLI workflows (`igord run`, `resume`, `verify`).

**Product Phase 1 (Portable Sovereign Agent)** and **Phase 2 (Agent Self-Provisioning)** feature work are complete. **Product Phase 2.5 (Protocol Foundation)** is next — making the checkpoint format, verifier, and effect primitives independently adoptable.

---

## Research Foundation

### Completed Tasks

- ✅ **Task 0** - Repository scaffold
- ✅ **Task 1** - P2P bootstrap & ping
- ✅ **Task 2** - WASM sandbox runtime & local survivable agents
- ✅ **Task 3** - Checkpoint persistence abstraction & storage provider
- ✅ **Task 4** - Migration protocol over libp2p
- ✅ **Task 5** - Rent metering & runtime accounting
- ✅ **Task 6** - Capability membrane MVP (clock/rand/log hostcalls, manifest, event log, deny-by-default)
- ✅ **Task 7** - Replay engine (single-tick verification, configurable modes, sliding window)
- ✅ **Task 8** - Agent SDK & developer experience (SDK, mocks, simulator, inspector, template)
- ✅ **Task 9** - Multi-node mobility testing (chain migration A→B→C→A, capability preservation, budget conservation, stress testing)

**Phase 2 result:** Agents can survive, migrate, and pay for execution.
**Phase 3 result:** Capability membrane, replay verification, developer tooling, and multi-node mobility validated.

---

## Phase 3: Autonomy

**Goal:** Enable agents to make autonomous decisions about where to run.

### Task 6: Capability Membrane & Hostcall ABI ✅

**Status:** Complete (MVP). Implemented capability manifest, `igor` host module (clock, rand, log), observation event log, and deny-by-default enforcement.

**Delivered:**
- `igor` WASM host module with clock, rand, log hostcalls (`internal/hostcall/`)
- Capability manifest parsing and validation at load time (`pkg/manifest/`)
- Per-tick observation event log recording (`internal/eventlog/`)
- Deny-by-default: undeclared capabilities cause load failure (CE-1, CE-2, CE-3)
- TinyGo-compiled agents with `//go:wasmimport` hostcall imports

**Deferred to follow-on tasks:**
- KV storage hostcalls (needs authority gating, CE-4)
- Pre-migration capability verification (CE-5)
- Hostcall cost accounting (CE-6)
- Replay verification (Task 7)

**Specs:** [CAPABILITY_MEMBRANE.md](../constitution/CAPABILITY_MEMBRANE.md), [CAPABILITY_ENFORCEMENT.md](../enforcement/CAPABILITY_ENFORCEMENT.md), [HOSTCALL_ABI.md](../runtime/HOSTCALL_ABI.md)

### Task 7: Replay Engine (Basic) ✅

**Status:** Complete. Single-tick replay verification fully implemented with configurable modes.

**Delivered:**
- Replay sandbox with isolated wazero runtime (`internal/replay/engine.go`)
- Replay-mode hostcalls that feed recorded observation values
- Single-tick verification: `checkpoint_N + event_log → verify checkpoint_N+1`
- Divergence detection with byte-level diff reporting
- Periodic self-verification in tick loop (`cmd/igord/main.go` `verifyNextTick`)
- Migration-time replay verification (`internal/migration/service.go` `verifyMigrationReplay`)
- Sliding replay window with configurable size (`--replay-window`)
- Configurable verification interval (`--verify-interval`)
- Formal replay modes: off, periodic, on-migrate, full (`--replay-mode`)
- Replay cost metering with optional logging (`--replay-cost-log`)

**Specs:** [REPLAY_ENGINE.md](../runtime/REPLAY_ENGINE.md)

### Task 8: Agent SDK & Developer Experience ✅

**Status:** Complete. Agent SDK, developer tooling, and testing infrastructure fully implemented.

**Delivered:**
- Agent SDK (`sdk/igor/`): `Agent` interface (Init, Tick, Marshal, Unmarshal), hostcall wrappers (ClockNow, RandBytes, Log, Logf), build-tag split for WASM and native builds
- Capability mocks (`sdk/igor/mock/`): pluggable `MockBackend` for native testing without WASM, deterministic clock/rand, log capture
- Local simulator (`internal/simulator/`): single-process WASM runner with deterministic hostcalls, per-tick replay verification, checkpoint round-trip verification
- Checkpoint inspector (`internal/inspector/`): parse and display checkpoint files, WASM hash verification
- Agent template (`agents/research/example/`): Survivor agent demonstrating SDK usage, hostcall patterns, and state serialization
- CLI flags: `--simulate`, `--ticks`, `--verify`, `--deterministic`, `--seed`, `--inspect-checkpoint`, `--inspect-wasm`

**Outcome:** Developers can build agents without manually managing WASM exports, memory, and hostcall signatures. Agents can be tested natively with mocks or as compiled WASM in the simulator.

### Task 9: Multi-Node Mobility Testing

**Objective:** Verify agents can hop between multiple nodes with capability-aware migration.

**Scope:**
- Test agent migrating A → B → C → A with capability verification
- Verify state + capability preservation across hops
- Verify budget conservation
- Stress test migration frequency

**Outcome:** Confidence in capability-aware agent mobility at scale.

---

## Phase 4: Economics

**Goal:** Implement cryptographic payment proofs and pricing mechanisms.

### Task 10: Payment Receipt Signing ✅

**Status:** Complete. Cryptographic payment receipts fully implemented.

**Delivered:**
- Receipt data structure with Ed25519 signing (`pkg/receipt/`)
- `wallet_balance`, `wallet_receipt_count`, `wallet_receipt` hostcalls (`internal/hostcall/wallet.go`)
- Receipt storage persistence (`internal/storage/` — SaveReceipts/LoadReceipts/DeleteReceipts)
- Receipt creation per checkpoint epoch with cost tracking (`internal/agent/instance.go`)
- Receipts travel with agents during migration (`internal/migration/service.go`)
- Wallet replay support for deterministic verification (`internal/replay/engine.go`)
- SDK wrappers and mocks for agent development (`sdk/igor/`)
- Simulator wallet hostcall support (`internal/simulator/hostcalls.go`)

**Outcome:** Auditable payment trail with hostcall-mediated access. Agents introspect budget and receipts. Receipts signed by node peer key, verified by anyone with the public key.

### Task 11: Node Pricing & Economic Settlement ✅

**Status:** Complete. Pricing discovery and settlement infrastructure implemented.

**Delivered:**
- Price discovery protocol over libp2p stream `/igor/pricing/1.0.0` (`internal/pricing/`)
- Budget adapter interface with mock implementation (`internal/settlement/`)
- Runtime tick gating on budget validity (`internal/agent/instance.go`)
- Bulk peer price scanning for migration decisions (`internal/pricing/service.go`)

**Outcome:** Nodes advertise prices, agents query prices, budget adapters gate execution.

---

## Phase 5: Hardening

**Goal:** Production-grade reliability and security.

### Task 12: Lease-Based Authority Epochs ✅

**Status:** Complete. Lease-based authority with epoch versioning fully implemented.

**Delivered:**
- Lease grant/renewal/expiry integrated with authority state machine (`internal/authority/`)
- Epoch advancement: major version on transfer, lease generation on renewal
- Anti-clone enforcement: expired leases cannot resume ticking
- Lease metadata in checkpoint format (v0x04)
- CLI flags: `--lease-duration`, `--lease-grace`

**Specs:** [LEASE_EPOCH.md](../runtime/LEASE_EPOCH.md)

**Outcome:** Automated detection of unresponsive nodes; liveness guarantee on top of existing safety.

### Task 13: Signed Checkpoint Lineage ✅

**Status:** Complete. Agent cryptographic identity and signed checkpoint chains implemented.

**Delivered:**
- Ed25519 agent keypairs with persistent storage (`pkg/identity/`)
- Signed checkpoint lineage: each checkpoint signed by agent identity (`pkg/lineage/`)
- WASM binary hash verification in checkpoint header
- Content hashing for tamper-evident checkpoint chains
- Checkpoint format v0x04 with prevHash, agentPubKey, signature fields

**Outcome:** Verifiable checkpoint lineage; foundation for trustless operation.

### Task 14: Migration Failure Recovery ✅

**Status:** Complete. Robust migration with retry, fallback, and lease-aware recovery.

**Delivered:**
- Peer registry with health tracking and candidate selection (`internal/registry/`)
- Retry policy with error classification: retriable, fatal, ambiguous (`internal/migration/retry.go`)
- Exponential backoff with configurable max attempts and delay
- `MigrateAgentWithRetry`: orchestrates retry loop with fallback to alternative peers
- FS-2 safety: ambiguous transfer (sent but no confirmation) enters RECOVERY_REQUIRED, no retry to different target
- Lease state transitions: `RevertHandoff()` (HANDOFF_INITIATED → ACTIVE_OWNER), `Recover()` (RECOVERY_REQUIRED → ACTIVE_OWNER at epoch major+1)
- Lease recovery in tick loop: RECOVERY_REQUIRED state auto-recovers
- `DivergenceMigrate` escalation wired to `MigrateAgentWithRetry`
- CLI flags: `--migration-retries`, `--migration-retry-delay`

**Outcome:** Robust migration under adverse conditions with single-instance invariant preserved.

### Task 15: Permissionless Hardening (Deprioritized)

**Status:** Deferred. Deprioritized in favor of product pivot.

**Scope:** Sybil resistance, host attestation, anti-withholding, cross-node replay. May be revisited when agents self-provision infrastructure (Product Phase 2+).

---

## Product Pivot

Igor's research foundation (Phases 2–5) proved that agents can checkpoint, migrate, resume, and maintain cryptographic lineage. The pivot focuses on making agents **portable and infrastructure-independent** — the agent is a digital object, not a deployment.

---

## Product Phase 1: Portable Sovereign Agent ✅

**Goal:** Make the agent a portable digital object with identity and verifiable history.

**Status:** Complete.

**Delivered:**
- ✅ **Task P1** - DID identity encoding (`pkg/identity/did.go`) — agents get `did:key:z6Mk...` identifiers from their Ed25519 keys
- ✅ **Task P2** - CLI subcommands — `igord run`, `igord resume`, `igord verify`, `igord inspect` for clean developer workflow
- ✅ **Task P3** - Checkpoint history archival (`internal/storage/fs_provider.go`) — every checkpoint archived for lineage verification
- ✅ **Task P4** - Lineage chain verifier (`internal/inspector/chain.go`) — walk checkpoint history, verify all signatures and hash chain
- ✅ **Task P5** - Heartbeat demo agent (`agents/heartbeat/`) — visible agent that demonstrates continuity across machines
- ✅ **Task P6** - Portable agent demo (`scripts/demo-portable.sh`) — end-to-end: run, stop, copy, resume on different machine, verify lineage

**Outcome:** An agent runs on Machine A, gets stopped, its checkpoint is copied to Machine B, and it resumes with the same DID, continuous tick count, and unbroken cryptographic lineage. The checkpoint file IS the agent.

---

## Product Phase 2: Agent Self-Provisioning

**Goal:** Agents choose and pay for their own infrastructure.

**Status:** In progress.

**Scope:**
- ✅ **HTTP hostcall** — agent calls external APIs (REST, webhooks) — `internal/hostcall/http.go`, allowed_hosts, timeout, max response size
- ✅ **Effect lifecycle model** — intent state machine in SDK for crash-safe effect tracking — `sdk/igor/effects.go`
- ✅ **Price watcher demo** — agent fetching live crypto prices via HTTP hostcall — `agents/pricewatcher/`
- ✅ **Treasury sentinel demo** — effect-safe treasury monitoring with crash recovery — `agents/sentinel/`
- ✅ **x402/USDC wallet hostcall** — agent pays for services from budget via `wallet_pay` hostcall — `internal/hostcall/payment.go`, signed receipts, crash-safe via effect lifecycle — `agents/x402buyer/`
- ✅ **Compute provider demo** — agent pays for and deploys itself to mock compute provider — `agents/deployer/`, `cmd/mockcloud/`
- **Self-migration** — agent decides when and where to move based on price/performance

**Outcome:** Agents are economically autonomous — they rent infrastructure, pay for it, and move when they choose.

---

## Product Phase 2.5: Protocol Foundation

**Goal:** Make Igor's checkpoint format, verification tools, and effect primitives independently adoptable — before building more runtime features.

**Status:** Not started.

**Strategic context:** Igor is a protocol, not a platform. The competitive moat is agent sovereignty (identity + lineage + portability), not durable execution. Durable execution platforms (Golem, Temporal, Restate) are potential deployment targets for Igor agents, not competitors. Adoption comes bottom-up through verification — publish the spec, ship the verifier, let verification pull developers toward the runtime. The analogy: the checkpoint format spec is JWT (RFC 7519), the standalone verifier is jwt.io, the Igor runtime is Auth0.

**Scope:**
- **Effects library extraction** (`igor.dev/effects`) — extract `sdk/igor/effects.go` + `encoding.go` as standalone Go module with zero Igor-specific dependencies. Blog post: "Crash-safe side effects in 200 lines." Tests bottom-up adoption thesis with minimal risk. (~1 day)
- **Checkpoint format spec** — formal one-page spec in `docs/runtime/CHECKPOINT_FORMAT.md`: byte-level offset table, signing domain construction, chain rule, DID encoding, version dispatch, reserved epoch fields, CID compatibility note. Language-agnostic — enables third-party implementations without reading Go code. (~1-2 days)
- **Standalone verifier** (`igor-verify`) — extract `ParseCheckpointHeader` from `internal/agent/instance.go` into `pkg/checkpoint` (fixes existing layering violation). Build zero-dependency binary: no wazero, no libp2p, no hostcalls (~5MB vs ~30MB). Bundle sample liquidation watcher history for zero-to-verification in 30 seconds. (~1-2 days)
- **Browser WASM verifier** — compile the verify path (`pkg/lineage/`, `pkg/identity/`, `pkg/checkpoint/`) to WASM via `GOOS=js GOARCH=wasm` (pure stdlib, no TinyGo needed). Embeddable in protocol dashboards: "agent lineage: verified, 847 checkpoints, no gaps." (~2-3 days)

**Binary model after this phase:**
- `igor` (renamed from `igord`) — agent operators — run, resume, verify, inspect
- `igor-verify` (new) — anyone verifying lineage — checkpoint parsing + chain verification only
- `igor-lab` (existing, unchanged) — protocol developers — migration, replay, leases, simulator

**Design principle:** No configurability, no extension points, no generalization. Five operations and one rule. A fixed offset table. Green or red. "I understand this completely in 5 minutes."

**Outcome:** The checkpoint spec is an adoptable protocol artifact. The standalone verifier is the on-ramp. The effects library generates Go ecosystem mindshare. All three are independently valuable and create the conditions for runtime adoption — without requiring anyone to run a WASM agent first.

---

## Product Phase 3: Permanent Memory

**Goal:** Agents have tamper-evident, publicly verifiable life histories.

**Scope:**
- **Arweave checkpoint archival** — permanent storage tier for checkpoint lineage
- **Two-tier storage** — fast local checkpoints + periodic Arweave archival (async, not in critical path)
- **Content-addressed checkpoints** — anyone can verify an agent's history from its content hash

**Outcome:** An agent's entire execution history is publicly verifiable and permanent.

---

## Product Phase 4: Ecosystem

**Goal:** Multi-language support, tooling, and community infrastructure.

**Scope:**
- **Multi-language SDK** — Rust and AssemblyScript compilation targets (beyond TinyGo)
- **Agent registry** — discover and share agents
- **Supervisor tooling** — optional auto-resurrection across node pools
- **Dashboard** — deploy, monitor, and fund agents via web UI

**Outcome:** A developer ecosystem around portable agents.

---

## Development Philosophy

Igor development follows these principles:

### Small Increments

- Each task is independently useful
- No monolithic rewrites
- Testable at each step

### Validate Before Scaling

- Prove correctness first
- Performance later
- Security iteratively

### Stay Minimal

- Resist feature creep
- Explicit over clever
- Delete more than add

### Fail Loudly

- Don't hide problems
- Error visibility over resilience
- Debug-friendly

---

## Release Strategy

Igor v0 is **not ready for production** and may never be.

### Version Semantics

- **v0.x** - Experimental, breaking changes expected
- **v1.x** - Stable APIs, production-ready (maybe)
- **v2.x+** - Advanced features, ecosystem

### Breaking Changes

Product phase transitions may break:
- Checkpoint format (new fields)
- CLI flags (new subcommands)
- Agent lifecycle (new hostcalls)
- SDK API (new types)

**No compatibility guarantees in v0.**

### Deprecation Policy

None. v0 is experimental. Things may be:
- Removed without warning
- Changed radically
- Replaced entirely

---

## Success Metrics

### Research Foundation (Phases 2–5) — All Met

- ✅ Agent runs, checkpoints, migrates, resumes
- ✅ Budget metering and payment receipts
- ✅ Capability membrane and replay verification
- ✅ Signed checkpoint lineage
- ✅ Lease-based authority and migration failure recovery

### Product Phase 1 (Complete)

- ✅ Agent has DID identity (`did:key:z6Mk...`)
- ✅ Agent checkpoints and resumes on a different machine
- ✅ Same DID, continuous tick count across machines
- ✅ Cryptographic lineage verified across machines
- ✅ Clean CLI: `igord run`, `resume`, `verify`

### Product Phase 2 Goals

- ✅ Agent calls external HTTP APIs via hostcall
- ✅ Effect lifecycle model for crash-safe side effects
- ✅ Agent pays for compute with real money (x402/USDC)
- ✅ Agent deploys itself to compute provider (mock)
- Agent decides when to migrate

### Product Phase 2.5 Goals

- Checkpoint format has a formal one-page spec (`docs/runtime/CHECKPOINT_FORMAT.md`)
- Effects library published as standalone Go module (`igor.dev/effects`)
- `igor-verify` binary verifies lineage chains with zero runtime dependencies
- Browser WASM verifier can parse and verify checkpoints
- Binary rename: `igord` → `igor`

---

## Timeline

**No timeline provided.**

Igor development follows "done when it's done" philosophy:
- Quality over speed
- Correctness over features
- Learning over shipping

Research phases (2–5) complete. Product Phase 2 feature work complete. Phase 2.5 (Protocol Foundation) is next.

---

## Contributing

Igor v0 is experimental research software.

**Contributions welcome:**
- Bug reports
- Documentation improvements
- Test cases
- Example agents

**Not accepting yet:**
- Major feature additions (scope creep)
- Performance optimizations (premature)
- Production deployments (not ready)

Focus: Complete Product Phase 2.5 (Protocol Foundation) before expanding scope.

---

## Long-Term Vision

Agents will pick their own infrastructure. Igor makes them portable enough to do so.

- **Now:** Agents are portable digital objects with identity and verifiable history
- **Next:** Checkpoint format spec, standalone verifier, and effects library enable verification-first adoption
- **Then:** Agents pay for their own compute; Golem/Temporal/Restate become deployment targets, not competitors
- **Later:** Agents have permanent, publicly verifiable memory on Arweave
- **Eventually:** A multi-language ecosystem of sovereign agents built on the Igor protocol

---

## What Could Derail Igor

Potential reasons to abandon or pivot:

1. **Fundamental flaw discovered** - Agent survival model doesn't work
2. **Performance unacceptable** - Too slow for practical use
3. **Security unfixable** - Trust model fundamentally broken
4. **No use cases** - Nobody wants autonomous agents
5. **Better alternatives** - Someone builds this better

Igor is an **experiment**. It may fail. That's acceptable.

---

## Related Work

Igor builds on ideas from:

- **Actor Model** - Isolated computation units
- **IPFS/Filecoin** - P2P infrastructure
- **Erlang/BEAM** - Process migration
- **WebAssembly** - Portable sandboxed code
- **Bitcoin Lightning** - Micropayment channels

Igor is **not novel**. It combines existing ideas in a specific way to explore autonomous agent survival.

**Competitive positioning:** Golem Cloud, Temporal, and Restate provide durable execution platforms. Igor's differentiator is agent-owned identity, cryptographic lineage, and true portability — the agent is a protocol-level entity, not a platform deployment. These platforms are potential deployment targets for Igor agents, not competitors. Igor answers "who IS the agent?" while they answer "where does the agent run?"

---

## Open Questions

1. **Can the checkpoint format spec gain adoption independently of the runtime?** (The JWT analogy test — does verification pull people toward the protocol?)
2. **Is the checkpoint format efficient enough for large agent state?** (MB+ state sizes; content-addressed external storage pointers as escape hatch)
3. **Will DeFi teams adopt igor-verify for agent lineage proofs before the full runtime?** (Bottom-up adoption test)
4. **Is WASM overhead acceptable for latency-sensitive agents?** (not for HFT, but for long-running?)
5. **Can two-tier storage (local + Arweave) work without slowing the critical path?**
6. **What hostcalls do agents actually need?** (HTTP, storage, payments — what else?)
7. **Should Golem/Temporal/Restate be treated as deployment targets rather than competitors?** (Protocol-not-platform thesis)

---

## Next Immediate Steps

Product Phase 2 feature work complete. Protocol foundation is next — verification-first adoption.

1. **Effects library extraction** — standalone Go module at `igor.dev/effects` (~1 day)
2. **Checkpoint format spec** — formal spec in `docs/runtime/CHECKPOINT_FORMAT.md` (~1-2 days)
3. **Standalone verifier** (`igor-verify`) — zero-dependency lineage verification binary (~1-2 days)
4. **Browser WASM verifier** — checkpoint verification compiled to WASM (~2-3 days)

Then:

5. **Self-migration** — agent decides when and where to move based on price/performance
6. **Real compute provider** — Akash or Golem as deployment targets (not competitors)
