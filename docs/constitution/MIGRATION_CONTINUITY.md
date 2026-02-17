# Migration Continuity

> **Specification cross-references:** [Spec Index](../SPEC_INDEX.md) | [Invariant Dependency Graph](../enforcement/INVARIANT_DEPENDENCY_GRAPH.md) | [Runtime Constitution](./RUNTIME_CONSTITUTION.md)

**Spec Layer:** Constitutional (Phase 1)  
**Stability:** High  
**Breaking Changes Require:** RFC + spec version bump  
**Related:** [SPEC_INDEX.md](../SPEC_INDEX.md), [INVARIANT_DEPENDENCY_GRAPH.md](../enforcement/INVARIANT_DEPENDENCY_GRAPH.md)

---

## Purpose

This document defines the conceptual migration contract governing how Igor transfers execution continuity between nodes. It formalizes migration guarantees, overlap constraints, and failure safety invariants.

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

## Migration Definition

### MC-1: Migration as Continuity Transfer

Migration transfers execution continuity from a source node to a target node. It is a structured handoff of:

- **Agent identity** — the target inherits the same canonical identity.
- **Execution authority** — the target becomes the sole ACTIVE_OWNER.
- **Checkpoint lineage** — the target resumes from the source's last committed checkpoint.

Migration is not cloning, replication, or restart. It is a transfer of a single execution thread across physical boundaries.

### MC-2: Migration Scope

Migration encompasses:

- Authority lifecycle transition (see [OWNERSHIP_AND_AUTHORITY.md](./OWNERSHIP_AND_AUTHORITY.md)).
- Checkpoint data transfer from source to target.
- Execution resumption on target from committed checkpoint.
- Retirement of source node's role for this agent identity.

Migration does not encompass:

- Creation of new agent identities.
- Modification of checkpoint content.
- Parallel or speculative execution.

---

## Safe Migration Guarantees

### MC-3: Checkpoint Lineage Preservation

Migration MUST preserve checkpoint lineage.

The target node resumes from the exact committed checkpoint produced by the source node. The checkpoint chain before and after migration forms a single unbroken sequence. No checkpoint is lost, skipped, modified, or fabricated during migration.

**Relationship:** Enforces EI-3 (checkpoint lineage integrity) and EI-10 (migration checkpoint continuity).

### MC-4: Execution Identity Preservation

Migration MUST preserve execution identity.

The agent identity on the target node is identical to the agent identity on the source node. From the agent's perspective, migration is transparent — execution continues under the same identity with the same state.

**Relationship:** Enforces OA-1 (canonical logical identity).

### MC-5: Single-Authority Guarantee

Migration MUST maintain the single-authority guarantee as defined by the **Single Active Authority — Formal Property** in [RUNTIME_CONSTITUTION.md](./RUNTIME_CONSTITUTION.md).

For honest nodes, at no point during migration may two nodes simultaneously hold execution authority for the same agent identity and epoch. Authority transfer is serialized: the source relinquishes before the target assumes.

Under partition or Byzantine behavior, Igor's safety posture remains fail-stop on ambiguity; this specification does not imply guaranteed global uniqueness under permanent communication loss or malicious duplicate execution.

**Relationship:** Enforces EI-1 (single active instance), EI-5 (singular authority), and OA-5 (transfer serialization).

---

## Overlap Constraints

### MC-6: Permitted Preparation Activities

During migration preparation — after the target has been identified but before authority transfer — the target node MAY:

- **Load checkpoint data** — receive and store the checkpoint from the source.
- **Validate state** — verify checkpoint integrity and format.
- **Prepare environment** — initialize WASM sandbox, allocate resources, pre-load agent binary.

These activities are read-only with respect to agent state. They prepare the target to assume authority but do not constitute execution.

### MC-7: Prohibited Preparation Activities

During migration preparation, the target node MUST NOT:

- **Tick** — execute the agent's tick function.
- **Produce side effects** — perform any action attributable to the agent's execution.
- **Mutate durable state** — write new checkpoints, modify existing checkpoints, or alter any persistent state associated with the agent identity.

These prohibitions hold until the target has formally assumed authority (ACTIVE_OWNER state).

**Rationale:** Preparation activities that mutate state or produce side effects would constitute unauthorized execution, violating the single-authority invariant. The target node is a passive recipient until authority transfer completes.

### MC-8: Source Node Constraints During Handoff

After initiating handoff (HANDOFF_INITIATED), the source node MUST NOT:

- Begin new tick executions.
- Produce new checkpoints beyond the handoff checkpoint.

The source node MAY:

- Complete any in-progress checkpoint operation.
- Serve checkpoint data to the target.
- Maintain the agent's durable state until transfer completes.

---

## Lineage Fork Detection

### MC-9: Divergent Checkpoint Detection During Migration

Two ownership claims referencing divergent checkpoint digests for the same epoch MUST trigger RECOVERY_REQUIRED.

If during migration — or at any point where authority state is being evaluated — the runtime observes that two nodes reference different checkpoint states for the same agent identity at the same logical point in the checkpoint chain, this constitutes evidence of a lineage fork. The agent MUST immediately enter RECOVERY_REQUIRED state.

This rule applies regardless of migration phase:

- During HANDOFF_INITIATED: if the source and target disagree on checkpoint state, migration MUST be aborted and the agent enters RECOVERY_REQUIRED.
- During HANDOFF_PENDING: if the target's received checkpoint does not match the source's committed checkpoint, migration MUST be aborted and the agent enters RECOVERY_REQUIRED.
- Post-migration: if a stale node claims authority with a divergent checkpoint, RECOVERY_REQUIRED is triggered per FS-4.

**Rationale:** Checkpoint divergence is conclusive evidence that the single-instance or checkpoint lineage invariant has been violated. Migration must explicitly preserve checkpoint lineage (MC-3), authority uniqueness (MC-5), and the single-ticker guarantee (EI-1). Divergent digests indicate one or more of these has failed.

---

## Failure Safety Matrix

The following matrix defines invariant outcomes for migration failure scenarios. Each scenario specifies what MUST be true regardless of implementation, not how the failure is handled.

### FS-1: Crash During Migration

**Scenario:** A node crashes during an active migration — either the source or target fails unexpectedly.

**Invariant outcomes:**

- At most one honest node may tick the agent after recovery.
- The last committed checkpoint from the verified authority chain is the recovery anchor.
- If the source crashed after relinquishing authority but the target did not confirm assumption, the agent enters RECOVERY_REQUIRED state.
- If the source crashed before relinquishing authority, the source retains authority upon restart and may resume from its last committed checkpoint.
- If the target crashed after assuming authority, the target retains authority upon restart and resumes from the transferred checkpoint.
- No state fabrication or interpolation is permitted during recovery.

### FS-2: Network Partition During Transfer

**Scenario:** Network connectivity between source and target is lost during authority transfer.

**Invariant outcomes:**

- Neither node may assume the transfer completed unless it received explicit confirmation.
- If the source cannot confirm target assumption, the agent enters RECOVERY_REQUIRED state.
- If the target cannot confirm source retirement, the target MUST NOT begin ticking until authority is unambiguously resolved.
- A partition MUST NOT cause both honest nodes to independently resume ticking.
- The last committed checkpoint remains the recovery anchor.
- Liveness may be lost for the duration of the partition — this is acceptable per safety-over-liveness (EI-6).

### FS-3: Duplicate Migration Attempts

**Scenario:** Multiple migration requests are issued for the same agent identity concurrently or in rapid succession.

**Invariant outcomes:**

- At most one migration may proceed in the honest authority chain for a given agent identity at any time.
- Concurrent migration attempts MUST be serialized or rejected.
- A second migration request while the agent is in HANDOFF_INITIATED or HANDOFF_PENDING state MUST be refused.
- No migration attempt may bypass the authority lifecycle.
- If conflicting migration attempts result in ambiguous authority, the agent enters RECOVERY_REQUIRED state.

### FS-4: Stale Checkpoint Restart

**Scenario:** A node attempts to resume an agent from a checkpoint that is not the latest in the authority chain — for example, due to a stale local copy after migration has occurred.

**Invariant outcomes:**

- A node MUST NOT tick an agent unless it holds execution authority for that agent identity.
- Possession of a checkpoint does not confer authority (EI-4).
- If a node detects that its checkpoint is not authoritative (e.g., authority has been transferred), it MUST NOT resume the agent.
- If a stale restart is detected after ticking has begun, the agent MUST enter RECOVERY_REQUIRED state.
- The authoritative checkpoint chain — not local storage — determines the valid recovery point.

---

## Failure Safety Summary

| Scenario | Invariant Outcome |
|----------|-------------------|
| Source crash before relinquishing | Source retains authority, resumes from checkpoint |
| Source crash after relinquishing | RECOVERY_REQUIRED until authority resolved |
| Target crash after assuming | Target retains authority, resumes from checkpoint |
| Network partition during transfer | RECOVERY_REQUIRED, no dual ticking |
| Duplicate migration attempts | Serialized or rejected, at most one proceeds |
| Stale checkpoint restart | Authority check required, no unauthorized ticking |

---

## Relationship to Other Specifications

This document operationalizes migration-specific guarantees defined in:

- **[EXECUTION_INVARIANTS.md](./EXECUTION_INVARIANTS.md)** — EI-1 (single instance), EI-6 (safety over liveness), EI-8 through EI-10 (migration invariants).
- **[OWNERSHIP_AND_AUTHORITY.md](./OWNERSHIP_AND_AUTHORITY.md)** — Authority lifecycle states and transfer serialization rules.
- **[RUNTIME_CONSTITUTION.md](./RUNTIME_CONSTITUTION.md)** — Single Active Authority formal property and adversary-qualified authority scope.

The failure safety matrix provides invariant outcomes for scenarios that exercise these contracts under adverse conditions.

---

## Document Status

**Type:** Constitutional Specification
**Scope:** Conceptual contracts only — no wire formats, serialization, or cryptographic mechanisms.
**Authority:** Normative for all future implementation of agent migration behavior and failure recovery. Part of the constitutional layer defined by [RUNTIME_CONSTITUTION.md](./RUNTIME_CONSTITUTION.md).
