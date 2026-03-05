# Igor v0 Roadmap

## Current Status: Phase 5 In Progress

Igor v0 has completed **Phase 2 (Survival)**, **Phase 3 (Autonomy)**, **Phase 4 (Economics)**, and started **Phase 5 (Hardening)** with Tasks 12–14.

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
- Agent template (`agents/example/`): Survivor agent demonstrating SDK usage, hostcall patterns, and state serialization
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

### Task 15: Permissionless Hardening

**Objective:** Security and incentive mechanisms for untrusted networks.

**Scope:**
- Sybil resistance (stake/reputation/cost-to-advertise)
- Host attestation options
- Anti-withholding incentives
- Full replay verification (cross-node)

**Outcome:** Foundation for permissionless deployment.

---

## Beyond Phase 5

Potential future directions (not committed):

### Advanced Features

- **Agent communication** - Peer-to-peer agent messaging
- **Agent composition** - Agents spawning sub-agents
- **Persistent agent storage** - Long-term state persistence
- **Agent marketplaces** - Discovery and matching
- **Reputation systems** - Node and agent ratings
- **Advanced payment rails** - Payment channels, L2s

### Performance Optimizations

- **Multi-agent nodes** - Run multiple agents concurrently
- **Hot migration** - Migrate without stopping ticks
- **Checkpoint compression** - Reduce transfer size
- **Connection pooling** - Reuse peer connections
- **WASM compilation cache** - Faster agent loading

### Ecosystem Tools

- **Node operator tools** - Monitoring dashboards
- **Agent debugger** - Inspect state and execution via replay
- **Migration visualizer** - Track agent movement
- **Economic analytics** - Budget and pricing insights
- **Checkpoint inspector** - Browse lineage, verify signatures

**Important:** These are speculative. Focus remains on v0 core functionality.

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

Phase 2 → Phase 3 may break:
- Checkpoint format (add manifest)
- Protocol messages (add capabilities)
- CLI flags (restructure)
- Agent lifecycle (new host functions)

**No compatibility guarantees in v0.**

### Deprecation Policy

None. v0 is experimental. Things may be:
- Removed without warning
- Changed radically
- Replaced entirely

---

## Success Metrics

### Phase 2 (Complete)

- ✅ Agent runs on Node A
- ✅ Agent checkpoints state
- ✅ Agent migrates to Node B
- ✅ Agent resumes from checkpoint
- ✅ Agent pays for execution
- ✅ No centralized coordination

**All 6 success criteria met.**

### Phase 3 Goals

- All agent I/O through capability membrane (hostcalls)
- Observation event log enables deterministic replay
- Basic replay verification working
- Agent SDK makes agent authoring accessible
- Capability-aware multi-node migration

### Phase 4 Goals

- Cryptographic payment receipts
- Auditable execution costs
- Competitive pricing market
- Economic incentives aligned

### Phase 5 Goals

- Production-ready reliability
- Security hardening complete
- Failure recovery robust
- Performance acceptable

---

## Timeline

**No timeline provided.**

Igor development follows "done when it's done" philosophy:
- Quality over speed
- Correctness over features
- Learning over shipping

Phase 3 began after Phase 2 validation. Tasks 6, 7, and 8 are complete.

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

Focus: Validate Phase 2 before expanding scope.

---

## Long-Term Vision

### Year 1: Proof of Concept

- Complete Phases 2-3
- Validate autonomous migration
- Run in research environments
- Publish findings

### Year 2: Economics

- Complete Phase 4
- Add payment infrastructure
- Test with real economic incentives
- Build small ecosystem

### Year 3: Hardening

- Complete Phase 5
- Security audit
- Production-grade reliability
- Consider v1.0

**Highly speculative.** Depends on validation results and community interest.

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

---

## Open Questions

Questions to answer through v0 development:

1. **Can agents practically migrate fast enough?**
2. **Is budget accounting sufficient without cryptographic proofs?**
3. **Do agents need more host functions?**
4. **Is WASM overhead acceptable?**
5. **Can checkpoint sizes stay small?**
6. **Will nodes actually compete on price?**

Answers will inform future phases.

---

## Success Criteria for v0

Igor v0 is **complete** when all Phase 2 tasks are done and validated.

Phase 2 is **validated** when:

- Agents run for days without issues
- Migration works reliably (>95% success)
- Budget accounting is accurate
- No critical bugs remain
- Documentation is comprehensive

**Status: Phase 2 validated. Phase 5 in progress.**

---

## Next Immediate Steps

Phase 4 complete. Phase 5 in progress (Tasks 12–14 complete). Next:

1. **Task 15: Permissionless Hardening** - Sybil resistance, host attestation, anti-withholding
2. **Extended testing** - Run agents with migration retry under adverse conditions
3. **Hardening** - Bug fixes, test coverage, documentation accuracy
