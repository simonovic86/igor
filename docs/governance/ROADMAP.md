# Igor v0 Roadmap

## Current Status: Phase 3 In Progress

Igor v0 has completed **Phase 2 (Survival)** and begun **Phase 3 (Autonomy)**.

### Completed Tasks

- ✅ **Task 0** - Repository scaffold
- ✅ **Task 1** - P2P bootstrap & ping
- ✅ **Task 2** - WASM sandbox runtime & local survivable agents
- ✅ **Task 3** - Checkpoint persistence abstraction & storage provider
- ✅ **Task 4** - Migration protocol over libp2p
- ✅ **Task 5** - Rent metering & runtime accounting
- ✅ **Task 6** - Capability membrane MVP (clock/rand/log hostcalls, manifest, event log, deny-by-default)

**Phase 2 result:** Agents can survive, migrate, and pay for execution.
**Phase 3 progress:** Agents interact through runtime-mediated hostcalls with observation recording.

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

### Task 8: Agent SDK & Developer Experience

**Objective:** Make agent authoring accessible with SDK and tooling.

**Scope:**
- Agent SDK (Go/TinyGo first) wrapping hostcall interface
- Local simulator (single-process deterministic replay)
- Capability mocks for testing
- Agent template / starter project
- Checkpoint inspector

**Outcome:** Developers can build agents without manually managing WASM exports, memory, and hostcall signatures.

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

### Task 10: Payment Receipt Signing

**Objective:** Nodes provide cryptographic proof of execution.

**Scope:**
- Node signs receipts with peer key
- Receipt includes execution time, cost, and epoch
- Agent verifies receipts via wallet hostcalls
- Store receipts for audit trail

**Components:**
- Receipt data structure tied to checkpoints/epochs
- Signing with libp2p identity
- `wallet.*` hostcall implementation
- Receipt storage

**Outcome:** Auditable payment trail with hostcall-mediated access.

### Task 11: Node Pricing & Economic Settlement

**Objective:** Implement economic settlement interface with external payment rails.

**Scope:**
- Nodes advertise pricing via libp2p gossip
- Agents query prices through hostcalls
- Budget adapter interface (mock + real EVM settlement)
- Runtime tick gating on budget validity

**Components:**
- Price advertisement protocol
- Budget adapter (pluggable: mock, EVM L2/stablecoin)
- Settlement interface
- Economic receipt infrastructure

**Outcome:** Agents can survive/die economically with real payment rails.

---

## Phase 5: Hardening

**Goal:** Production-grade reliability and security.

### Task 12: Lease-Based Authority Epochs

**Objective:** Time-bound execution authority with leases for automated failure detection.

**Scope:**
- Lease grant/renewal/expiry integrated with authority state machine
- Epoch advancement (major version on transfer, lease generation on renewal)
- Anti-clone enforcement: expired leases cannot resume ticking
- Lease metadata in checkpoint

**Specs:** [LEASE_EPOCH.md](../runtime/LEASE_EPOCH.md)

**Outcome:** Automated detection of unresponsive nodes; liveness guarantee on top of existing safety.

### Task 13: Signed Checkpoint Lineage

**Objective:** Cryptographic identity for agents and signed checkpoint chains.

**Scope:**
- Ed25519 agent keypairs
- Signed checkpoint lineage (each checkpoint signed by agent identity)
- WASM binary hash verification
- Checkpoint content-addressed storage (IPFS/CID compatible)

**Outcome:** Verifiable checkpoint lineage; foundation for trustless operation.

### Task 14: Migration Failure Recovery

**Objective:** Handle migration failures gracefully with lease-aware recovery.

**Scope:**
- Retry failed migrations with exponential backoff
- Lease-aware recovery: expired lease triggers re-election
- Fallback to alternative nodes
- Cross-node replay verification for migrated checkpoints

**Outcome:** Robust migration under adverse conditions.

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

Phase 3 began after Phase 2 validation. Task 6 is complete.

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

**Status: Phase 2 validated. Phase 3 in progress.**

---

## Next Immediate Steps

With Phase 3 Tasks 6 (Capability Membrane) and 7 (Replay Engine) complete:

1. **Task 8: Agent SDK** - Hostcall wrappers, lifecycle helpers, developer experience
2. **Task 9: Multi-Node Mobility Testing** - Chain migration, capability preservation
3. **Hardening** - Bug fixes, test coverage, documentation accuracy
4. **Extended testing** - Run agents with hostcalls for hours/days
