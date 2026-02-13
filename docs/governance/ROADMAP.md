# Igor v0 Roadmap

## Current Status: Phase 2 Complete ✅

Igor v0 has completed **Phase 2 (Survival)**, implementing all core functionality needed for autonomous mobile agents.

### Completed Tasks (Phase 2)

- ✅ **Task 0** - Repository scaffold
- ✅ **Task 1** - P2P bootstrap & ping
- ✅ **Task 2** - WASM sandbox runtime & local survivable agents
- ✅ **Task 3** - Checkpoint persistence abstraction & storage provider
- ✅ **Task 4** - Migration protocol over libp2p
- ✅ **Task 5** - Rent metering & runtime accounting

**Result:** Agents can survive, migrate, and pay for execution.

---

## Phase 3: Autonomy

**Goal:** Enable agents to make autonomous decisions about where to run.

### Task 6: Agent Manifest Validation & Capability Enforcement

**Objective:** Implement agent manifests that declare capabilities and resource requirements.

**Scope:**
- Define manifest schema (`pkg/manifest`)
- Validate manifest on agent load
- Enforce capability matching
- Reject incompatible agents

**Components:**
- Manifest parser and validator
- Capability registry per node
- Resource requirement checking
- Rejection protocol

**Outcome:** Nodes can verify they can host an agent before accepting migration.

### Task 7: Migration Decision Interface (Agent → Runtime)

**Objective:** Allow agents to request migration autonomously.

**Scope:**
- WASM host function for migration request
- Agent can query available nodes
- Agent can evaluate pricing
- Agent can trigger migration

**Components:**
- `request_migration()` host function
- Peer discovery from WASM
- Price comparison logic
- Autonomous decision making

**Outcome:** Agents can migrate without human intervention.

### Task 8: Multi-Node Agent Mobility Testing

**Objective:** Verify agents can hop between multiple nodes successfully.

**Scope:**
- Test agent migrating A → B → C → A
- Verify state preservation
- Verify budget conservation
- Stress test migration frequency

**Components:**
- Multi-node test harness
- Migration orchestration tests
- State verification tools
- Budget tracking across hops

**Outcome:** Confidence in agent mobility at scale.

---

## Phase 4: Economics

**Goal:** Implement cryptographic payment proofs and pricing mechanisms.

### Task 9: Payment Receipt Signing

**Objective:** Nodes provide cryptographic proof of execution.

**Scope:**
- Node signs receipts with peer key
- Receipt includes execution time + cost
- Agent verifies receipts
- Store receipts for audit

**Components:**
- Receipt data structure
- Signing with libp2p identity
- Verification logic
- Receipt storage

**Outcome:** Auditable payment trail.

### Task 10: Node Pricing Negotiation

**Objective:** Agents and nodes negotiate execution price.

**Scope:**
- Nodes advertise pricing
- Agents query prices from peers
- Negotiation protocol
- Dynamic price adjustment

**Components:**
- Price advertisement protocol
- Agent price comparison
- Negotiation messages
- Market discovery

**Outcome:** Competitive pricing market for execution.

---

## Phase 5: Hardening

**Goal:** Production-grade reliability and security.

### Task 11: Runtime Resource Isolation

**Objective:** Enhance sandbox with additional protections.

**Scope:**
- CPU usage limits
- I/O bandwidth limits
- System call filtering
- Enhanced WASM validation

**Components:**
- Resource accounting
- cgroup integration (Linux)
- Syscall filtering
- Performance monitoring

**Outcome:** Safer execution environment.

### Task 12: Migration Failure Recovery

**Objective:** Handle migration failures gracefully.

**Scope:**
- Retry failed migrations
- Fallback to alternative nodes
- Rollback on target failure
- Timeout handling

**Components:**
- Retry logic with exponential backoff
- Peer ranking/reputation
- Transaction rollback
- Health monitoring

**Outcome:** Robust migration under adverse conditions.

### Task 13: Agent Integrity Verification

**Objective:** Verify agent identity and code integrity.

**Scope:**
- Agent cryptographic identity
- WASM binary signing
- Checkpoint signing
- Identity verification

**Components:**
- Public key infrastructure
- Signature generation/verification
- Trust chain
- Revocation mechanism

**Outcome:** Trustworthy agent identity.

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

- **Agent SDK** - Libraries for common patterns
- **Node operator tools** - Monitoring dashboards
- **Agent debugger** - Inspect state and execution
- **Migration visualizer** - Track agent movement
- **Economic analytics** - Budget and pricing insights

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

- Agent autonomously chooses node
- Agent evaluates node pricing
- Agent migrates without human intervention
- Multi-hop migration works reliably

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

Phase 3 begins when Phase 2 is validated through extended testing.

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

**Status: Phase 2 implemented, validation ongoing.**

---

## Next Immediate Steps

With Phase 2 complete, the immediate focus is:

1. **Extended testing** - Run agents for hours/days
2. **Bug fixing** - Address issues found in testing
3. **Documentation review** - Ensure accuracy
4. **Community feedback** - Gather early user input
5. **Phase 3 planning** - Design agent autonomy features

**No new features until Phase 2 is validated.**
