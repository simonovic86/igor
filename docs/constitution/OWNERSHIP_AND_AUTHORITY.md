# Ownership and Authority

> **Specification cross-references:** [Spec Index](../SPEC_INDEX.md) | [Invariant Dependency Graph](../enforcement/INVARIANT_DEPENDENCY_GRAPH.md) | [Runtime Constitution](./RUNTIME_CONSTITUTION.md)

**Spec Layer:** Constitutional (Phase 1)  
**Stability:** High  
**Breaking Changes Require:** RFC + spec version bump  
**Related:** [SPEC_INDEX.md](../SPEC_INDEX.md), [INVARIANT_DEPENDENCY_GRAPH.md](../enforcement/INVARIANT_DEPENDENCY_GRAPH.md)

---

## Purpose

This document defines the authority lifecycle model governing Igor agent execution identity. It formalizes agent identity, authority states, authority transfer rules, and conflict resolution semantics.

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

## Agent Identity Definition

### OA-1: Canonical Logical Identity

An agent identity is the canonical logical execution identity of a single agent. It uniquely identifies the agent across all nodes and across the agent's entire lifetime.

Agent identity is:

- **Singular** — one identity corresponds to exactly one logical agent.
- **Persistent** — identity survives migration, restart, and node failure.
- **Non-forkable** — identity cannot be duplicated or split.
- **Node-independent** — identity is not bound to any specific node.

Agent identity is the anchor for authority, checkpoint lineage, and execution continuity. All runtime invariants (see [EXECUTION_INVARIANTS.md](./EXECUTION_INVARIANTS.md)) are scoped per agent identity.

---

## Authority Lifecycle

### OA-2: Authority States

At any point in time, the authority relationship between an agent identity and the runtime exists in exactly one of the following conceptual states:

#### ACTIVE_OWNER

A single node holds execution authority for the agent identity. The node MAY tick the agent and produce new checkpoints.

- Exactly one node is in ACTIVE_OWNER state per agent identity.
- The node may advance checkpoint state.
- Normal execution proceeds.

#### HANDOFF_INITIATED

The current authority holder has begun the process of transferring authority. The source node has signaled intent to relinquish authority but has not yet completed the transfer.

- The source node still holds authority.
- The source node MUST NOT begin new ticks after initiating handoff.
- The target node has been identified but does not yet hold authority.

#### HANDOFF_PENDING

Authority is in transit. The source node has relinquished active execution but the target node has not yet confirmed assumption of authority.

- All nodes MUST NOT tick the agent.
- The agent's last committed checkpoint is durable and consistent.
- A liveness gap exists — this is acceptable per safety-over-liveness (EI-6).

#### RETIRED

The source node has completed its role in the authority transfer. It no longer holds authority, checkpoint data may be cleaned up, and it has no further obligations for this agent identity.

- The source node MUST NOT tick the agent.
- The source node MUST NOT produce checkpoints for this agent.
- Authority has been fully transferred or the agent has been terminated.

#### RECOVERY_REQUIRED

The runtime cannot determine which node holds authority. This state is entered when:

- Authority transfer did not complete cleanly.
- Conflicting authority claims are detected.
- The runtime cannot confirm singular authority.

In RECOVERY_REQUIRED state:

- All nodes MUST NOT tick the agent. All ticking MUST cease immediately.
- All nodes MUST NOT produce new checkpoints. All checkpoint production MUST cease.
- Execution MUST remain halted until authority is unambiguously resolved.
- Resolution requires explicit recovery action.

### OA-3: State Transition Rules

Authority state transitions follow strict ordering:

```
ACTIVE_OWNER → HANDOFF_INITIATED → HANDOFF_PENDING → ACTIVE_OWNER (new node)
                                                    ↘
                                                     RETIRED (old node)
```

From any state, a transition to RECOVERY_REQUIRED is possible if the runtime detects ambiguity.

```
Any State → RECOVERY_REQUIRED
RECOVERY_REQUIRED → ACTIVE_OWNER (after explicit resolution)
```

No state may be skipped. No backward transitions are permitted except through RECOVERY_REQUIRED resolution.

---

## Authority Transfer Rules

### OA-4: Explicit Transfer Events

Authority transfer is an explicit runtime event. Authority is never implicitly assumed, inferred, or acquired by proximity, timing, or data possession.

A node holding a copy of an agent's checkpoint does not hold authority. A node that previously held authority does not retain residual authority. Authority is granted through the defined transfer lifecycle, not through state possession.

### OA-5: Transfer Serialization

Authority transfer is a serialized operation. At no point during transfer may two nodes simultaneously hold authority for the same agent identity.

The transfer sequence is:

1. Source node holds authority (ACTIVE_OWNER).
2. Source node initiates handoff (HANDOFF_INITIATED).
3. Source node ceases ticking and relinquishes authority (HANDOFF_PENDING).
4. Target node assumes authority (ACTIVE_OWNER on target).
5. Source node retires (RETIRED).

Steps 3 and 4 are ordered: the source MUST relinquish before the target assumes.

### OA-6: Transfer Completeness

An authority transfer is complete only when:

- The target node has confirmed assumption of authority.
- The source node has transitioned to RETIRED.
- Exactly one node holds ACTIVE_OWNER for the agent identity.

An incomplete transfer — where the source has relinquished but the target has not confirmed — results in HANDOFF_PENDING. If this state persists beyond expected bounds, the runtime transitions to RECOVERY_REQUIRED.

---

## Conflict Safety Rule

### OA-7: Conflicting Authority Resolution

If at any point the runtime detects or suspects that more than one node claims authority for the same agent identity, the agent MUST transition to RECOVERY_REQUIRED state.

Conflicting authority is defined as any situation where:

- Two nodes believe they are ACTIVE_OWNER for the same identity.
- A node begins ticking an agent for which another node has not confirmed retirement.
- Authority state cannot be unambiguously determined.

In RECOVERY_REQUIRED state:

- All nodes MUST cease ticking the affected agent.
- All nodes MUST cease producing checkpoints for the affected agent.
- The last committed checkpoint from the verified authority chain is the recovery anchor.
- Execution resumes only after authority is explicitly and unambiguously reassigned to a single node.

**Rationale:** Conflicting authority is a precursor to concurrent ticking, checkpoint forking, and state divergence. The only safe response is to halt and resolve explicitly. This follows from safety-over-liveness (EI-6).

---

## Authority Lifecycle Summary

| State | Node May Tick | Node May Checkpoint | Notes |
|-------|:---:|:---:|-------|
| ACTIVE_OWNER | Yes | Yes | Normal execution |
| HANDOFF_INITIATED | No (new) | No (new) | Source preparing to transfer |
| HANDOFF_PENDING | No | No | Authority in transit |
| RETIRED | No | No | Source obligation complete |
| RECOVERY_REQUIRED | No | No | Ambiguity detected, halted |

---

## Relationship to Execution Invariants

This document defines the authority lifecycle that enforces several execution invariants from [EXECUTION_INVARIANTS.md](./EXECUTION_INVARIANTS.md):

- **EI-1** (single active instance) — enforced by ACTIVE_OWNER singularity.
- **EI-4** (authority/durability separation) — authority states are independent of checkpoint storage.
- **EI-5** (singular authority) — at most one ACTIVE_OWNER per identity.
- **EI-6** (safety over liveness) — RECOVERY_REQUIRED halts execution under ambiguity.
- **EI-8** (migration single-instance) — transfer serialization prevents concurrent ticking.
- **EI-11** (divergent lineage detection) — conflicting ownership with divergent checkpoint digests triggers RECOVERY_REQUIRED.

---

## Document Status

**Type:** Constitutional Specification
**Scope:** Conceptual contracts only — no wire formats, serialization, or cryptographic mechanisms.
**Authority:** Normative for all future implementation of agent identity and authority lifecycle behavior. Part of the constitutional layer defined by [RUNTIME_CONSTITUTION.md](./RUNTIME_CONSTITUTION.md).
