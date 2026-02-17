# Runtime Threat Model

This document is the canonical Runtime-layer statement of Igor's threat assumptions, failure model, adversary classes, and trust boundaries.

It defines the threat universe Igor v0 operates in.  
It does not define protocol mechanics, wire formats, cryptographic schemes, or implementation algorithms.

Related:
- [SECURITY_MODEL.md](./SECURITY_MODEL.md) - current mechanisms and limitations
- [EXECUTION_INVARIANTS.md](../constitution/EXECUTION_INVARIANTS.md) - constitutional safety guarantees
- [MIGRATION_CONTINUITY.md](../constitution/MIGRATION_CONTINUITY.md) - migration continuity contracts

---

## System Model

### Conceptual Entities

- **Node**: Runtime host process (`igord`) providing sandboxed execution, checkpoint persistence, migration transport, and metering.
- **Agent**: Portable WASM execution package with explicit state, runtime budget, and logical identity.
- **Authority**: Runtime-granted right to advance an agent's checkpoint lineage (tick and produce new checkpoints).
- **Checkpoint**: Committed durable state boundary used for resume, restart, and migration continuity.
- **Migration**: Transfer of execution continuity (identity, authority, checkpoint lineage) from source node to target node.

### Single-Instance Meaning Under Igor Assumptions

In Igor, "single-instance" means:

- At most one node is authoritative to advance canonical checkpoint lineage for a given agent identity at a time.
- Migration and lifecycle transitions are intended to preserve that property through serialized handoff and fail-stop behavior on ambiguity.

This is a runtime safety objective, not a global impossibility claim.  
Without global consensus and cryptographic global authority proofs, Byzantine or partitioned stale nodes may still attempt concurrent execution. When such risk or evidence appears, the safe response is halt/recovery, not optimistic continuation.

---

## Failure Classes

The runtime threat universe includes the following failure classes:

1. **Crash (process exit)**
   - Node process stops unexpectedly.
   - In-memory state is lost.
   - Durability and continuity rely on the last committed checkpoint.

2. **Restart with persistent storage**
   - Node restarts with checkpoint directory intact.
   - Agent may resume from committed checkpoint boundary.

3. **Restart with partial loss**
   - Node restarts with missing/corrupted subset of prior local state.
   - Includes stale checkpoint copies and incomplete local migration artifacts.
   - Safety requires no unauthorized state advancement from stale data.

4. **Network partition**
   - Communication between peers is unavailable for some duration.
   - Peers cannot safely assume transfer completion without explicit confirmation.

5. **Message delay / duplication / reordering**
   - Transport can delay, duplicate, or reorder messages.
   - Runtime correctness cannot depend on strict real-time delivery order.

---

## Adversary Classes

### A1 Honest-but-failing node

Node follows protocol intent but fails due to crashes, storage faults, restarts, or connectivity loss.

### A2 Byzantine node

Node can deviate arbitrarily, including:
- Lying about authority state
- Replaying stale messages
- Attempting forked execution
- Refusing cooperation during handoff/recovery

### A3 Economic griefer

Participant can induce cost/liveness pressure without necessarily violating protocol syntax, such as:
- Handshake spam
- Livelock-style migration churn
- Budget-drain attempts through repeated failed interactions

### A4 Identity forger

Adversary attempts impersonation or replay of identity-bearing envelopes/messages.

### v0 Scope Declaration

| Adversary Class | v0 Coverage | Future Coverage Direction |
|----------------|-------------|---------------------------|
| A1 Honest-but-failing | In scope (primary) | Continue hardening restart/migration recovery paths |
| A2 Byzantine | Threat acknowledged; robust resistance is out of scope in v0 | Phase 5 hardening + integrity/auth mechanisms |
| A3 Economic griefer | Threat acknowledged; robust anti-grief controls are out of scope in v0 | Phase 3/4/5 controls (policy, pricing, limits) |
| A4 Identity forger | Threat acknowledged; strong identity/auth guarantees are out of scope in v0 | Phase 5 integrity + identity verification work |

---

## Network Assumptions

Igor's runtime model assumes an asynchronous network with failure:

- No bound on delivery latency
- No guarantee of in-order delivery
- Possible duplication and loss
- Possible prolonged partition

For practical liveness, Igor assumes eventual periods of connectivity where cooperating peers can exchange required handoff messages.  
For safety, Igor assumes those liveness conditions may be absent at any time and therefore favors halt under ambiguity.

### Boundary Without Consensus

Igor v0 does not implement distributed consensus. Therefore:

- It cannot globally prove unique authority under arbitrary Byzantine behavior plus partition.
- It cannot guarantee global ordering beyond what participants can directly verify.
- It treats uncertainty as a safety condition requiring pause/recovery, not automatic reconciliation.

---

## Trust Assumptions

### Trusted (local trust domain)

- Local node process and its immediate runtime control flow
- Local operator policy/configuration
- Local persistence substrate to the extent it behaves as expected (with failure classes explicitly acknowledged)

### Not Trusted

- Remote peers (authority claims, cooperation, liveness behavior)
- Remote metering correctness
- Remote checkpoint custody
- Network timing/ordering behavior
- Any assumption that another node is honest by default

Igor's baseline posture: nodes are untrusted infrastructure providers.

---

## Security Goals

Security goals are stated as properties, not mechanisms:

1. **Sandbox containment (agent -> node boundary)**
   - Agent code must remain contained within runtime isolation boundaries.

2. **Resource bounds**
   - Agent execution must remain bounded by configured runtime limits (time/memory/budget).

3. **Continuity goals**
   - Agents should survive crash/restart/migration via committed checkpoints and resumable execution.

4. **Safety goals**
   - Prevent unsafe concurrent authoritative advancement of a single identity's lineage.
   - Under uncertainty, prefer halt/recovery over unsafe continuation.

These goals do not modify constitutional guarantees; they operationalize runtime intent under explicit threat assumptions.

---

## Non-Goals (v0)

The following are explicitly out of scope for Igor v0:

- Public hostile-network deployment guarantees
- Robust malicious-node resistance
- Fraud-proof or cryptographically verifiable metering
- Cryptographic checkpoint integrity guarantees
- Global authority proofs via consensus

v0 is a survival-runtime experiment, not a production security platform.

---

## Phase Mapping

Threat coverage aligns with repository phases in [TASKS.md](../../TASKS.md):

| Phase | Relevant Tasks | Threat Coverage Intent |
|------|----------------|------------------------|
| Phase 2 - Survival | Tasks 0-5.11 | Baseline containment, checkpointing, migration continuity under A1 failures |
| Phase 3 - Autonomy | Tasks 6-8 | Capability/policy surfaces that can reduce unsafe placement and some grief vectors |
| Phase 4 - Economics | Tasks 9-10 | Receipt/signing and pricing mechanisms to reduce metering and economic abuse risk |
| Phase 5 - Hardening | Tasks 11-13 | Stronger isolation, recovery robustness, and integrity/identity controls for A2/A4-class threats |

This mapping describes direction only. It does not commit Igor to specific protocol or cryptographic implementations.

