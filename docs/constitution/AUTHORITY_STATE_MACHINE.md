# Authority State Machine

> **Specification cross-references:** [Spec Index](../SPEC_INDEX.md) | [Invariant Dependency Graph](../enforcement/INVARIANT_DEPENDENCY_GRAPH.md) | [Runtime Constitution](./RUNTIME_CONSTITUTION.md)

**Spec Layer:** Constitutional (Phase 1)  
**Stability:** High  
**Breaking Changes Require:** RFC + spec version bump  
**Related:** [SPEC_INDEX.md](../SPEC_INDEX.md), [INVARIANT_DEPENDENCY_GRAPH.md](../enforcement/INVARIANT_DEPENDENCY_GRAPH.md)

---

## Section 1 — Overview

This document defines the formal authority lifecycle state machine for an agent identity in the Igor runtime. It codifies the conceptual states, transitions, and operational constraints that enforce the single-active-ticker guarantee.

The authority state machine derives from and operationalizes:

- [EXECUTION_INVARIANTS.md](./EXECUTION_INVARIANTS.md) — foundational runtime invariants (EI-1 through EI-11)
- [OWNERSHIP_AND_AUTHORITY.md](./OWNERSHIP_AND_AUTHORITY.md) — authority lifecycle model (OA-1 through OA-7)
- [MIGRATION_CONTINUITY.md](./MIGRATION_CONTINUITY.md) — migration continuity contracts (MC-1 through MC-9, FS-1 through FS-4)

This state machine clarifies how the single-active-ticker guarantee (EI-1) is enforced conceptually across authority transitions, migration, and recovery. It does NOT introduce protocol mechanics, wire message formats, or implementation algorithms.

The authority state machine is a **conceptual model** that describes the logical states an agent identity may occupy with respect to execution authority. Implementations must uphold the transitions and constraints defined here to maintain constitutional invariants.

---

## Section 2 — State Definitions

The authority state machine consists of five distinct states. Each state defines whether tick execution is permitted, what durable state expectations exist, and what safety intent the state serves.

### ACTIVE_OWNER

**Description:** A single node holds execution authority for the agent identity. Normal execution proceeds.

**Tick Permission:** Allowed

**Durable State Expectation:** The node holds a valid committed checkpoint and may produce new checkpoints through tick execution. The checkpoint lineage is actively advancing.

**Safety Intent:** This is the sole state in which tick execution is permitted. Exactly one node per agent identity may occupy this state at any time. The singularity of ACTIVE_OWNER enforces EI-1 (single active instance) and EI-5 (singular authority).

---

### HANDOFF_INITIATED

**Description:** The current authority holder has signaled intent to transfer authority. The source node is preparing for migration but has not yet relinquished authority.

**Tick Permission:** Forbidden (for new ticks)

**Durable State Expectation:** The source node holds the last committed checkpoint and must not advance state beyond this point. The checkpoint to be transferred is finalized. The target node has been identified but does not yet hold authority or durable state.

**Safety Intent:** This state ensures the source stops producing new state before authority transfer begins. It prevents the source from advancing checkpoint lineage during the handoff window. This enforces MC-8 (source node constraints during handoff) and contributes to MC-5 (single-authority guarantee).

---

### HANDOFF_PENDING

**Description:** Authority is in transit between nodes. The source has relinquished active execution, and the target has not yet confirmed assumption of authority.

**Tick Permission:** Forbidden

**Durable State Expectation:** The last committed checkpoint from the source is durable and consistent. Neither the source nor the target holds active execution authority. The checkpoint data is being transferred or has been transferred but not yet validated by the target.

**Safety Intent:** This state represents the serialization gap between source relinquishment and target assumption. No node may tick during this window, ensuring authority never overlaps. A liveness gap is acceptable per EI-6 (safety over liveness) and EI-9 (migration pause acceptability). This state enforces OA-5 (transfer serialization).

---

### RETIRED

**Description:** The source node has completed its role in the authority transfer. It no longer holds authority and has no further obligations for this agent identity.

**Tick Permission:** Forbidden

**Durable State Expectation:** The node may retain checkpoint data for a grace period but must not produce new checkpoints. Authority has been fully transferred to the target or the agent has been terminated. The node's checkpoint lineage participation has ended.

**Safety Intent:** This state ensures the source node cannot resume execution after transfer. Once retired, the node must not re-enter the authority lifecycle without explicit recovery action. This prevents stale nodes from violating EI-1 (single active instance) and enforces FS-4 (stale checkpoint restart constraints).

---

### RECOVERY_REQUIRED

**Description:** The runtime cannot determine which node holds authority. Execution is halted until authority is unambiguously resolved.

**Tick Permission:** Forbidden

**Durable State Expectation:** The last committed checkpoint from the verified authority chain serves as the recovery anchor. No node may produce new checkpoints. The checkpoint lineage is frozen at the last verifiable consistent state.

**Safety Intent:** This is the safety fence state. It is entered when:

- Authority transfer did not complete cleanly
- Conflicting authority claims are detected (OA-7)
- Divergent checkpoint digests are observed (EI-11, MC-9)
- The runtime cannot confirm singular authority for any reason

RECOVERY_REQUIRED enforces EI-6 (safety over liveness) and EI-7 (fail-stop on invariant violation). Execution must remain halted until explicit recovery action re-establishes singular authority. This state prevents silent state corruption, dual ticking, and lineage forks.

---

## Section 3 — Normative Transition Table

This table defines all permitted authority state transitions. Each transition specifies the trigger condition, preconditions that must hold, postconditions that result, and explanatory notes.

| FROM → TO | Trigger | Preconditions | Postconditions | Notes |
|-----------|---------|---------------|----------------|-------|
| **ACTIVE_OWNER → HANDOFF_INITIATED** | Source node initiates migration | Source holds singular authority; target node identified and reachable; last checkpoint committed | Source enters HANDOFF_INITIATED; source must not begin new ticks | Migration begins; authority remains with source but ticking ceases |
| **HANDOFF_INITIATED → HANDOFF_PENDING** | Source relinquishes authority | Checkpoint transfer to target initiated or completed; source confirms no in-progress ticks | Source no longer holds authority; neither node may tick | Authority serialization gap; liveness pause acceptable per EI-9 |
| **HANDOFF_PENDING → ACTIVE_OWNER (target)** | Target assumes authority | Target received and validated checkpoint; target confirms readiness; source confirmed relinquishment | Target becomes ACTIVE_OWNER; target may tick; exactly one ACTIVE_OWNER exists | Migration completes successfully; execution continuity transferred |
| **HANDOFF_PENDING → RETIRED (source)** | Transfer completion acknowledged | Target confirmed ACTIVE_OWNER; source acknowledged transfer success | Source enters RETIRED; source may not tick or produce checkpoints | Source obligation complete; source may clean up local state |
| **ACTIVE_OWNER → RETIRED** | Agent termination or explicit shutdown | Agent execution ended by request; final checkpoint committed if applicable | Node enters RETIRED; no node holds authority for this identity | Lifecycle termination; identity may not resume without recovery |
| **ANY → RECOVERY_REQUIRED** | Authority ambiguity detected | Conflicting authority claims observed; divergent checkpoint digests detected; transfer timeout exceeded; invariant violation suspected | All nodes enter RECOVERY_REQUIRED; all ticking ceases immediately; checkpoint lineage frozen | Safety fence activated per EI-6, EI-7, OA-7, EI-11 |
| **RECOVERY_REQUIRED → ACTIVE_OWNER** | Explicit recovery action | Authority unambiguously resolved; singular authority re-established; recovery anchor checkpoint identified | Exactly one node becomes ACTIVE_OWNER; execution may resume from recovery anchor | Recovery complete; single-ticker guarantee restored |

**Conceptual Triggers:** All triggers in this table are conceptual events. Implementations may realize these through various mechanisms (timeouts, explicit messages, consensus protocols, etc.), but the logical semantics must match this table.

**Precondition Enforcement:** Preconditions are invariant requirements. A transition attempted without its preconditions satisfied constitutes an invariant violation and must trigger RECOVERY_REQUIRED per EI-7.

**Postcondition Guarantees:** Postconditions are non-negotiable outcomes. Implementations must ensure these hold after each transition completes.

---

## Section 4 — Forbidden Transitions

The following transitions are explicitly forbidden. Attempting any of these constitutes a constitutional invariant violation.

### RETIRED → ACTIVE_OWNER (without recovery)

**Violation:** A node that has retired from authority may not spontaneously resume execution.

**Rationale:** RETIRED indicates the node's authority lifecycle has ended. Resuming without recovery would risk dual authority if the target is already ACTIVE_OWNER. This would violate EI-1 (single active instance) and EI-5 (singular authority).

**Constitutional Reference:** OA-3 (state transition rules), FS-4 (stale checkpoint restart)

---

### HANDOFF_PENDING → tick execution

**Violation:** No node may execute ticks while authority is in the HANDOFF_PENDING state.

**Rationale:** During HANDOFF_PENDING, neither the source nor the target holds authority. Ticking during this window would violate OA-5 (transfer serialization) and MC-5 (single-authority guarantee during migration).

**Constitutional Reference:** EI-1 (single active instance), MC-7 (prohibited preparation activities)

---

### Multiple concurrent ACTIVE_OWNER for same identity/epoch

**Violation:** Two or more nodes may not simultaneously hold ACTIVE_OWNER state for the same agent identity.

**Rationale:** This is the direct violation of the single-ticker law. Concurrent ACTIVE_OWNER states would permit concurrent ticking, forked checkpoint lineage, and budget divergence. This is the foundational invariant violation.

**Constitutional Reference:** EI-1 (single active instance), EI-5 (singular authority), OA-2 (authority states)

---

### Any state allowing tick outside ACTIVE_OWNER

**Violation:** Tick execution is permitted only in ACTIVE_OWNER state. No other state may authorize tick execution.

**Rationale:** ACTIVE_OWNER is the sole state that confers execution authority. Permitting ticks in any other state would bypass the authority lifecycle and risk dual authority, violating the single-ticker guarantee.

**Constitutional Reference:** EI-1 (single active instance), OA-2 (authority states)

---

### HANDOFF_INITIATED → ACTIVE_OWNER (skip HANDOFF_PENDING)

**Violation:** Authority transfer must pass through HANDOFF_PENDING. The source must relinquish before the target assumes.

**Rationale:** Skipping HANDOFF_PENDING would eliminate the serialization gap, risking overlapping authority if the handoff is not atomic. The HANDOFF_PENDING state enforces the source-stops-before-target-starts ordering.

**Constitutional Reference:** OA-3 (state transition rules), OA-5 (transfer serialization), MC-5 (single-authority guarantee)

---

### RECOVERY_REQUIRED → any state except ACTIVE_OWNER

**Violation:** Recovery must result in exactly one ACTIVE_OWNER or remain in RECOVERY_REQUIRED.

**Rationale:** RECOVERY_REQUIRED is entered due to authority ambiguity. Transitioning to HANDOFF_INITIATED, HANDOFF_PENDING, or RETIRED would not resolve the ambiguity. Recovery must explicitly re-establish singular authority.

**Constitutional Reference:** OA-7 (conflicting authority resolution), EI-6 (safety over liveness)

---

### Backward transitions (except via RECOVERY_REQUIRED)

**Violation:** State transitions must follow the forward progression defined in Section 3. Backward transitions are not permitted except through explicit recovery.

**Rationale:** The authority lifecycle is a one-way progression from ACTIVE_OWNER through handoff to RETIRED. Backward transitions would indicate state machine inconsistency and risk authority confusion.

**Constitutional Reference:** OA-3 (state transition rules)

---

## Section 5 — ASCII State Diagram

The following diagram illustrates the authority state machine structure and permitted transitions.

```
                    ┌─────────────────┐
                    │  ACTIVE_OWNER   │ ◄──┐
                    │                 │    │
                    │ (tick allowed)  │    │
                    └────────┬────────┘    │
                             │             │
                             │ initiate    │
                             │ handoff     │
                             │             │
                             ▼             │
                    ┌─────────────────┐    │
                    │ HANDOFF_        │    │
                    │ INITIATED       │    │
                    │                 │    │
                    │ (no new ticks)  │    │
                    └────────┬────────┘    │
                             │             │
                             │ source      │
                             │ relinquish  │
                             │             │
                             ▼             │
                    ┌─────────────────┐    │
                    │ HANDOFF_        │    │
                    │ PENDING         │    │
                    │                 │    │
                    │ (no ticks)      │    │
                    └────┬────────┬───┘    │
                         │        │        │
            target       │        │ source │
            assumes      │        │ acks   │
                         │        │        │
                         │        ▼        │
                         │   ┌─────────┐  │
                         │   │ RETIRED │  │
                         │   │         │  │
                         │   │(no ticks)  │
                         │   └─────────┘  │
                         │                │
                         └────────────────┘
                          (new ACTIVE_OWNER
                           on target node)


              ANY STATE ──────────────────┐
                                           │
                        authority          │
                        ambiguity          │
                        detected           │
                                           │
                                           ▼
                                 ┌──────────────────┐
                                 │ RECOVERY_        │
                                 │ REQUIRED         │
                                 │                  │
                                 │ (all ticks halt) │
                                 └────────┬─────────┘
                                          │
                                          │ explicit
                                          │ recovery
                                          │ action
                                          │
                                          ▼
                                 ┌─────────────────┐
                                 │ ACTIVE_OWNER    │
                                 │                 │
                                 │ (after recovery)│
                                 └─────────────────┘
```

**Diagram Legend:**

- **Solid boxes:** Authority states
- **Arrows (→):** Permitted transitions
- **Labels:** Transition triggers (conceptual)
- **Parenthetical notes:** Tick permission summary

**Diagram Consistency:** This diagram exactly reflects the transition table in Section 3. Every arrow corresponds to a table row. No additional transitions are depicted.

---

## Section 6 — Tick Permission Matrix

This matrix defines operational permissions for each authority state. Each row represents a state; each column represents an operation. The matrix entries specify whether the operation is Allowed or Forbidden in that state.

| State | Tick Execution | Produce Durable Checkpoint | Initiate Handoff | Accept Handoff | Perform Side Effects |
|-------|:--------------:|:--------------------------:|:----------------:|:--------------:|:--------------------:|
| **ACTIVE_OWNER** | **Allowed** | **Allowed** | **Allowed** | Forbidden | **Allowed** |
| **HANDOFF_INITIATED** | Forbidden | Forbidden (new) | Forbidden | Forbidden | Forbidden |
| **HANDOFF_PENDING** | Forbidden | Forbidden | Forbidden | **Allowed** (target) | Forbidden |
| **RETIRED** | Forbidden | Forbidden | Forbidden | Forbidden | Forbidden |
| **RECOVERY_REQUIRED** | Forbidden | Forbidden | Forbidden | Forbidden | Forbidden |

### Matrix Definitions

**Tick Execution:** Execution of the agent's tick() function. Only ACTIVE_OWNER may tick.

**Produce Durable Checkpoint:** Writing a new committed checkpoint to durable storage, advancing the checkpoint lineage. Only ACTIVE_OWNER may produce new checkpoints. HANDOFF_INITIATED permits completion of an in-progress checkpoint but forbids starting new checkpoints.

**Initiate Handoff:** Beginning the authority transfer process by identifying a target and entering HANDOFF_INITIATED state. Only ACTIVE_OWNER may initiate handoff.

**Accept Handoff:** Receiving checkpoint data from the source, validating it, and transitioning to ACTIVE_OWNER. Only a node in HANDOFF_PENDING (from the target's perspective, awaiting authority) may accept handoff.

**Perform Side Effects:** Any operation attributable to the agent's execution, including network requests, file I/O, budget expenditure, or observable state mutation. Only ACTIVE_OWNER may perform side effects. Preparation activities during HANDOFF_PENDING (MC-6) are explicitly read-only and do not constitute side effects.

### Derivation From Existing Invariants

All matrix rules derive from existing constitutional invariants:

- **Tick execution restricted to ACTIVE_OWNER:** EI-1 (single active instance), OA-2 (authority states)
- **Checkpoint production restricted to ACTIVE_OWNER:** EI-3 (checkpoint lineage integrity), MC-3 (checkpoint lineage preservation)
- **Handoff initiation restricted to ACTIVE_OWNER:** OA-4 (explicit transfer events), OA-5 (transfer serialization)
- **Handoff acceptance restricted to HANDOFF_PENDING:** OA-3 (state transition rules), MC-5 (single-authority guarantee)
- **Side effects restricted to ACTIVE_OWNER:** MC-7 (prohibited preparation activities)

No new operational guarantees are introduced by this matrix. It is a tabular representation of permissions already implied by the constitutional documents.

---

## Document Status

**Type:** Constitutional Specification  
**Scope:** Formal authority lifecycle model — conceptual states, transitions, and constraints only. No wire formats, serialization, or implementation algorithms.  
**Authority:** Normative for all implementations of authority lifecycle and migration handoff. Part of the constitutional layer defined by [RUNTIME_CONSTITUTION.md](./RUNTIME_CONSTITUTION.md).
