# Runtime Constitution

**Spec Layer:** Constitutional (Phase 1)  
**Stability:** High  
**Breaking Changes Require:** RFC + spec version bump  
**Related:** [SPEC_INDEX.md](../SPEC_INDEX.md), [INVARIANT_DEPENDENCY_GRAPH.md](../enforcement/INVARIANT_DEPENDENCY_GRAPH.md)

---

## Purpose

This document is the constitutional specification root for the Igor runtime. It declares the non-negotiable guarantees that define Igor as a continuity-preserving execution substrate for autonomous agents.

All runtime enforcement invariants MUST derive from the constitutional invariants defined in this document hierarchy. No implementation, optimization, or protocol extension may violate these guarantees.

This document does NOT prescribe wire formats, serialization schemas, cryptographic mechanisms, or distributed consensus protocols.

---

## Constitutional Declaration

Igor is a **continuity-preserving execution substrate**. Its purpose is to guarantee that autonomous agents maintain unbroken execution identity, linear checkpoint lineage, and singular authority across migration, restart, and failure.

The following properties are non-negotiable:

1. **At most one node ticks a given agent identity at any time.**
2. **Checkpoint lineage forms a single ordered chain with no forks.**
3. **Authority to advance state is held by exactly one node per agent identity.**
4. **Under uncertainty, the runtime halts rather than risks invariant violation.**

These properties hold across all operational states: normal execution, migration, recovery, and restart. No trade-off between performance, liveness, or convenience may weaken them.

---

## Foundational Separation

The Igor runtime rests on two orthogonal concepts:

**Checkpoint lineage defines state identity.**

The ordered chain of committed checkpoints — from initial state through every subsequent checkpoint — constitutes the canonical history of an agent. State identity is checkpoint lineage. Two agents with divergent checkpoint lineage are, by definition, different execution histories.

**Ownership defines authority to advance checkpoint lineage.**

Authority is the runtime-granted right held by exactly one node to execute ticks and produce new checkpoints for an agent identity. Authority is not conferred by possession of checkpoint data, proximity to the agent, or historical association. Authority flows through the defined ownership lifecycle.

These two concepts are independent:

- A node may hold durable checkpoint data without holding authority.
- A node may hold authority while checkpoint data is replicated elsewhere.
- Authority determines who may write; checkpoint lineage determines what was written.

---

## Single Active Authority — Formal Property

This section clarifies the constitutional "single active authority" property under explicit adversary and failure assumptions from [THREAT_MODEL.md](../runtime/THREAT_MODEL.md). It does not change migration ordering, checkpoint semantics, epoch advancement semantics, or cryptographic assumptions.

### 1. Definitions

**Authority Instance**

A node currently permitted to execute agent code under a valid ownership envelope and epoch.

**Active Execution**

A runtime state in which:

- Agent code is being executed.
- The node considers itself authoritative.
- The node may advance state or epoch.

**Authority Epoch**

A strictly monotonic logical version of ownership as defined by envelope supersession rules.

### 2. Property Statement — Adversary Qualified

#### Under A1 (Crash-Only, Honest Nodes)

At most one Authority Instance exists globally at any time for a given Authority Epoch.

If crashes occur:

- No two honest nodes will concurrently execute the same epoch.
- Restarted nodes must respect persisted envelope and epoch monotonicity rules.

#### Under F4 (Network Partition)

Igor guarantees:

- No two honest nodes may both legally claim authoritative execution of the same epoch without violating envelope monotonicity.

Igor does NOT guarantee:

- Global uniqueness under permanent partition.
- Immediate visibility of authority changes during communication loss.

Temporary split-brain visibility may occur. Convergence requires network communication.

#### Under F5 (Message Delay / Duplication / Reordering)

Igor guarantees:

- Honest nodes treat stale, delayed, duplicated, or reordered ownership envelopes as non-authoritative when they are superseded by higher valid epochs.
- Honest nodes do not regress accepted epoch because of delayed or duplicate delivery.

Igor does NOT guarantee:

- Bounded propagation delay for authority changes.
- Immediate global agreement after message disturbances.

#### Under A2 (Byzantine Node)

Igor does NOT guarantee prevention of malicious duplicate execution by a Byzantine node.

Igor guarantees only:

- Honest nodes will not accept conflicting authority envelopes if epoch monotonicity and supersession rules are respected.
- Byzantine behavior cannot force honest nodes to regress epoch.

### 3. Fork Classification

| Fork Type | Definition | Prevented | Detectable | Out of Scope |
|----------|------------|:---------:|:----------:|:------------:|
| **Illegal Fork** | Two honest nodes executing the same epoch under valid envelopes. | Yes (under A1 assumptions) | Yes | No |
| **Byzantine Fork** | A malicious node executing despite revocation or supersession. | No | Partially (through conflicting claims and lineage evidence) | Malicious prevention is out of scope in v0 |
| **Partition Fork** | Concurrent execution caused by temporary communication loss. | Not globally preventable under F4 | Yes, after communication restores | Permanent-partition global uniqueness is out of scope |

### 4. Convergence Property

If the network eventually reconnects and no Byzantine node controls all reachable peers:

- Honest nodes must converge to the highest valid epoch envelope.
- Lower epoch envelopes must be considered superseded.

This statement defines a constitutional property expectation only. It does not define a consensus algorithm.

---

## Constitutional Specification Documents

The constitutional invariants are defined across three normative specification documents. Together with this document, they form the constitutional layer of the Igor specification.

### [EXECUTION_INVARIANTS.md](./EXECUTION_INVARIANTS.md)

Defines foundational runtime invariants governing agent execution:

- Single active instance guarantee (EI-1)
- Checkpoint boundary resumption (EI-2)
- Checkpoint lineage integrity (EI-3)
- Authority and durability separation (EI-4)
- Singular authority (EI-5)
- Safety over liveness (EI-6)
- Fail-stop on violation (EI-7)
- Migration single-instance preservation (EI-8)
- Migration pause acceptability (EI-9)
- Migration checkpoint continuity (EI-10)

### [OWNERSHIP_AND_AUTHORITY.md](./OWNERSHIP_AND_AUTHORITY.md)

Defines the authority lifecycle model:

- Canonical logical identity (OA-1)
- Authority lifecycle states: ACTIVE_OWNER, HANDOFF_INITIATED, HANDOFF_PENDING, RETIRED, RECOVERY_REQUIRED (OA-2)
- State transition rules (OA-3)
- Explicit transfer events (OA-4)
- Transfer serialization (OA-5)
- Transfer completeness (OA-6)
- Conflicting authority resolution (OA-7)

### [MIGRATION_CONTINUITY.md](./MIGRATION_CONTINUITY.md)

Defines the migration continuity contract:

- Migration as continuity transfer (MC-1)
- Migration scope (MC-2)
- Checkpoint lineage preservation (MC-3)
- Execution identity preservation (MC-4)
- Single-authority guarantee (MC-5)
- Permitted and prohibited preparation activities (MC-6, MC-7)
- Source node constraints during handoff (MC-8)
- Failure safety matrix (FS-1 through FS-4)

### [CAPABILITY_MEMBRANE.md](./CAPABILITY_MEMBRANE.md)

Defines the constitutional guarantees governing agent I/O — the trust boundary between agent code and host resources:

- Total mediation of agent I/O (CM-1)
- Explicit capability declaration (CM-2)
- Deny by default (CM-3)
- Observation determinism (CM-4)
- Side effect attribution (CM-5)
- Capability immutability during tick (CM-6)
- Capability survival through migration (CM-7)

### [AUTHORITY_STATE_MACHINE.md](./AUTHORITY_STATE_MACHINE.md)

Formalizes the authority lifecycle as a state machine with normative transition table, forbidden transitions, and tick permission matrix. Operationalizes the single-active-ticker guarantee from EI-1 and the authority lifecycle from OA-2.

---

## Derivation Rule

All runtime enforcement invariants MUST derive from constitutional invariants.

Runtime enforcement rules — such as checkpoint atomicity mechanisms, budget monotonicity enforcement, tick duration limits, and sandbox isolation — implement and operationalize the constitutional guarantees. They describe *how* the runtime upholds the constitution, not *what* the constitution requires.

The enforcement layer is defined in [RUNTIME_ENFORCEMENT_INVARIANTS.md](../enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md). Every enforcement invariant must trace its justification to one or more constitutional invariants defined in the documents listed above.

No enforcement invariant may contradict a constitutional invariant. In case of conflict, the constitutional invariant prevails.

---

## Specification Hierarchy

```
Constitutional Layer (this document + referenced specs)
    │
    │  defines non-negotiable guarantees
    │
    ▼
Enforcement Layer (RUNTIME_ENFORCEMENT_INVARIANTS.md)
    │
    │  implements constitutional guarantees via runtime rules
    │
    ▼
Implementation Layer (source code, protocol handlers, storage providers)
    │
    │  realizes enforcement rules in executable form
    │
    ▼
Operational Layer (deployment, monitoring, recovery procedures)
```

Each layer derives authority from the layer above. No lower layer may introduce guarantees or weaken guarantees defined at a higher layer.

---

## Constitutional Amendments

Constitutional invariants are subject to specification governance rules defined in [SPEC_GOVERNANCE.md](../governance/SPEC_GOVERNANCE.md). Changes to constitutional documents require formal review and explicit version bumps.

---

## Document Status

**Type:** Constitutional Specification Root
**Scope:** Non-negotiable runtime guarantees — no wire formats, serialization, or cryptographic mechanisms.
**Authority:** Supreme normative authority over all Igor runtime specification and implementation.
