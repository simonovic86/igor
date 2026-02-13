# Execution Invariants

## Purpose

This document defines the foundational runtime invariants governing Igor agent execution. These are conceptual contracts specifying what the runtime MUST guarantee regardless of implementation strategy.

This document does NOT prescribe wire formats, serialization schemas, cryptographic mechanisms, or distributed consensus protocols.

---

## Terminology

| Term | Definition |
|------|-----------|
| **identity** | Canonical logical execution identity of an agent. Uniquely identifies an agent across all nodes and across time. |
| **authority** | The right held by exactly one node to advance an agent's checkpoint state. Authority is a runtime concept, separate from state durability. |
| **checkpoint** | Atomic durable snapshot of agent state and metadata at a committed boundary. The canonical durability anchor for execution continuity. |
| **migration** | Transfer of execution continuity — including identity, authority, and checkpoint lineage — from one node to another. |
| **ticking** | Active execution of an agent's tick function by the authoritative node. A ticking instance is one that may produce side effects and advance state. |
| **recovery-required** | A safety state entered when the runtime cannot determine unique authority. Execution is halted until authority is unambiguously resolved. |

---

## Runtime Identity Invariants

### EI-1: Single Active Instance

Igor guarantees **at-most-one active ticking instance** per agent identity at any point in time.

No two nodes may concurrently execute tick() for the same agent identity. This invariant holds across all operational states: normal execution, migration, recovery, and restart.

**Rationale:** Concurrent ticking would produce divergent state, violate checkpoint lineage, and break budget conservation. The single-instance invariant is the foundational safety property upon which all other guarantees depend.

---

## Continuity Invariants

### EI-2: Checkpoint Boundary Resumption

Agent execution MUST resume only from a committed checkpoint boundary.

A checkpoint represents a complete, durable, consistent snapshot of agent state. No execution may resume from partial state, uncommitted intermediary state, or reconstructed state.

**Rationale:** The checkpoint is the canonical durability anchor. All execution continuity flows through committed checkpoints. Resuming from any other source would violate state consistency guarantees.

### EI-3: Checkpoint Lineage Integrity

Each checkpoint is logically derived from the preceding checkpoint through a defined sequence of ticks. The lineage from initial state through all checkpoints forms a single ordered chain.

No checkpoint may exist outside this chain. No fork in checkpoint lineage is permitted.

**Rationale:** Forked lineage implies concurrent state mutation, which violates EI-1. Linear checkpoint lineage is both a consequence of and evidence for the single-instance invariant.

---

## Authority Invariants

### EI-4: Authority and Durability Separation

Execution authority and state durability are separate concepts.

- **Authority** defines which node may advance checkpoint state (i.e., tick and produce new checkpoints).
- **Durability** defines where checkpoint data is persisted.

A node may hold a durable copy of a checkpoint without holding authority to advance it. Authority is a runtime grant, not a storage property.

**Rationale:** Separating these concepts allows checkpoint data to be replicated or cached for migration preparation without granting execution rights. This distinction is essential for safe migration handoff.

### EI-5: Singular Authority

For any given agent identity, at most one node holds execution authority at any time.

Authority is never shared, split, or held concurrently by multiple nodes. If the runtime cannot determine which node holds authority, execution enters recovery-required state.

**Rationale:** Dual authority would permit concurrent ticking, violating EI-1.

---

## Safety Invariants

### EI-6: Safety Over Liveness

Under uncertainty, Igor MUST prefer safety over liveness.

When the runtime cannot confirm that all invariants hold — for example, during network partition, ambiguous authority, or incomplete migration — the correct response is to pause execution rather than risk invariant violation.

Temporary execution pauses are acceptable. Invariant violations are not.

**Rationale:** An agent that pauses temporarily loses only liveness. An agent that violates single-instance or checkpoint lineage invariants may suffer irrecoverable state corruption, budget divergence, or identity compromise.

### EI-7: Fail-Stop on Invariant Violation

If the runtime detects an invariant violation, the affected agent MUST be stopped immediately. The runtime MUST NOT attempt silent recovery or automatic state reconciliation.

**Rationale:** Silent recovery risks masking fundamental correctness failures. Igor's design philosophy is to fail loudly on invariant violations.

---

## Migration Invariants

### EI-8: Migration Single-Instance Preservation

Migration MUST NOT produce concurrent ticking instances at any point during the transfer process.

At no moment may both the source node and target node be ticking the same agent identity. The handoff of execution authority is a serialized event: the source must relinquish authority before the target begins ticking.

**Rationale:** Migration is a transfer of authority, not a duplication. Concurrent ticking during migration would violate EI-1 and could fork checkpoint lineage.

### EI-9: Migration Pause Acceptability

Migration MAY temporarily pause execution.

A period during which no node is ticking the agent is acceptable during migration. The agent is not lost — its state is durable in the committed checkpoint. Liveness is temporarily sacrificed to preserve safety.

**Rationale:** Follows from EI-6. The gap between source ceasing and target beginning is a safe liveness pause, not a failure.

### EI-10: Migration Checkpoint Continuity

Migration MUST preserve checkpoint lineage.

The target node resumes from the same committed checkpoint that the source node last produced. No state is lost, invented, or interpolated during migration.

**Rationale:** Migration transfers execution continuity. The checkpoint is the continuity anchor. Breaking checkpoint lineage would make migration indistinguishable from state corruption.

---

## Normative Statements

### Checkpoint Lineage as State Identity

Checkpoint lineage defines canonical state identity.

The ordered chain of committed checkpoints — from initial state through every subsequent checkpoint — constitutes the canonical history of an agent. Two execution histories with divergent checkpoint lineage are, by definition, different state identities. There is no mechanism to reconcile divergent lineage.

### Ownership as Authority to Advance Lineage

Ownership defines authority to advance checkpoint lineage.

The right to produce new checkpoints — and thereby extend an agent's state identity — is held by exactly one node at any time. This right flows through the ownership lifecycle defined in [OWNERSHIP_AND_AUTHORITY.md](./OWNERSHIP_AND_AUTHORITY.md). No node may advance lineage without holding authority.

### EI-11: Divergent Lineage Detection

Conflicting ownership claims referencing divergent checkpoint digests for the same epoch MUST transition to RECOVERY_REQUIRED.

If the runtime detects or infers that two nodes claim authority for the same agent identity and their claimed checkpoint states are not identical, this constitutes evidence of a lineage fork. The agent MUST immediately enter RECOVERY_REQUIRED state. No node may tick the agent until the fork is resolved and singular authority is re-established.

**Rationale:** Divergent checkpoint digests under concurrent ownership claims are conclusive evidence that EI-1 (single active instance) or EI-3 (checkpoint lineage integrity) has been violated. The only safe response is to halt execution and require explicit recovery.

---

## Relationship to Specification Hierarchy

This document is part of the constitutional specification layer defined by the [Runtime Constitution](./RUNTIME_CONSTITUTION.md). Constitutional invariants defined here are implemented by enforcement rules in [RUNTIME_ENFORCEMENT_INVARIANTS.md](./RUNTIME_ENFORCEMENT_INVARIANTS.md).

---

## Invariant Summary

| ID | Invariant | Category |
|----|-----------|----------|
| EI-1 | At-most-one active ticking instance per identity | Runtime Identity |
| EI-2 | Resume only from committed checkpoint boundary | Continuity |
| EI-3 | Checkpoint lineage forms single ordered chain | Continuity |
| EI-4 | Authority and durability are separate concepts | Authority |
| EI-5 | At most one node holds execution authority | Authority |
| EI-6 | Prefer safety over liveness under uncertainty | Safety |
| EI-7 | Fail-stop on invariant violation | Safety |
| EI-8 | Migration must not produce concurrent ticking | Migration |
| EI-9 | Migration may temporarily pause execution | Migration |
| EI-10 | Migration must preserve checkpoint lineage | Migration |
| EI-11 | Divergent lineage under conflicting ownership triggers RECOVERY_REQUIRED | Safety |

---

## Document Status

**Type:** Constitutional Specification
**Scope:** Conceptual contracts only — no wire formats, serialization, or cryptographic mechanisms.
**Authority:** Normative for all future implementation of execution identity, authority, and migration behavior. Part of the constitutional layer defined by [RUNTIME_CONSTITUTION.md](./RUNTIME_CONSTITUTION.md).
