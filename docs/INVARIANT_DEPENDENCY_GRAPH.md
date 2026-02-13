# Invariant Dependency Graph

## Purpose

This document maps the dependency relationships between Igor's constitutional invariants and runtime enforcement invariants. It shows how enforcement rules derive from constitutional guarantees and how constitutional invariants depend on each other.

---

## Constitutional Invariant Dependencies

The following graph describes how constitutional invariants relate to and depend on each other.

### Root Invariant: Single Ticker Law

```
EI-1: Single Active Instance (at-most-one ticking instance per identity)
 │
 ├── EI-3: Checkpoint Lineage Integrity
 │    │    (single ticker → no concurrent mutation → no lineage fork)
 │    │
 │    ├── EI-2: Checkpoint Boundary Resumption
 │    │         (lineage integrity → resume only from committed boundary)
 │    │
 │    └── EI-11: Divergent Lineage Detection
 │              (lineage integrity → divergent digests trigger recovery)
 │
 ├── EI-5: Singular Authority
 │    │    (single ticker → at most one node holds authority)
 │    │
 │    ├── EI-4: Authority and Durability Separation
 │    │         (singular authority → authority is not storage property)
 │    │
 │    ├── OA-1: Canonical Logical Identity
 │    │         (singular authority scoped per identity)
 │    │
 │    ├── OA-2: Authority Lifecycle States
 │    │    │    (singular authority → defined state machine)
 │    │    │
 │    │    ├── OA-3: State Transition Rules
 │    │    │         (lifecycle states → strict ordering)
 │    │    │
 │    │    └── OA-7: Conflicting Authority Resolution
 │    │              (lifecycle states → ambiguity triggers RECOVERY_REQUIRED)
 │    │
 │    ├── OA-4: Explicit Transfer Events
 │    │         (singular authority → authority never implicitly assumed)
 │    │
 │    ├── OA-5: Transfer Serialization
 │    │         (singular authority → source relinquishes before target assumes)
 │    │
 │    └── OA-6: Transfer Completeness
 │              (singular authority → transfer complete only when confirmed)
 │
 ├── EI-8: Migration Single-Instance Preservation
 │    │    (single ticker → migration must not produce concurrent ticking)
 │    │
 │    ├── MC-1: Migration as Continuity Transfer
 │    │         (single instance → migration is transfer, not duplication)
 │    │
 │    ├── MC-3: Checkpoint Lineage Preservation
 │    │         (single instance + lineage integrity → migration preserves lineage)
 │    │
 │    ├── MC-5: Single-Authority Guarantee
 │    │         (single instance → authority serialized during migration)
 │    │
 │    ├── MC-6/MC-7: Overlap Constraints
 │    │         (single instance → target must not tick during preparation)
 │    │
 │    ├── MC-8: Source Node Constraints
 │    │         (single instance → source must not tick after handoff initiation)
 │    │
 │    └── MC-9: Divergent Checkpoint Detection
 │              (single instance → divergent digests during migration trigger recovery)
 │
 ├── EI-9: Migration Pause Acceptability
 │         (derived from EI-6: safety over liveness)
 │
 └── EI-10: Migration Checkpoint Continuity
            (single ticker + lineage integrity → migration preserves checkpoint chain)
```

### Safety Foundation

```
EI-6: Safety Over Liveness
 │    (under uncertainty, halt rather than risk violation)
 │
 ├── EI-7: Fail-Stop on Invariant Violation
 │         (safety → stop immediately, no silent recovery)
 │
 ├── EI-9: Migration Pause Acceptability
 │         (safety → liveness gap during migration is acceptable)
 │
 ├── OA-7: Conflicting Authority Resolution
 │         (safety → ambiguity triggers RECOVERY_REQUIRED)
 │
 ├── EI-11: Divergent Lineage Detection
 │         (safety → divergent digests trigger RECOVERY_REQUIRED)
 │
 └── Failure Safety Matrix (FS-1 through FS-4)
          (safety → invariant outcomes for all failure scenarios)
```

---

## Enforcement Derivation Chain

The following shows how each runtime enforcement invariant derives from constitutional invariants.

```
Constitutional                          Enforcement
─────────────                          ─────────────

EI-2 (checkpoint boundary)      ──►    RE-1 (atomic checkpoints)
EI-3 (lineage integrity)        ──►    RE-1 (atomic checkpoints)

EI-2 (checkpoint boundary)      ──►    RE-2 (state persistence)
EI-10 (migration continuity)    ──►    RE-2 (state persistence)

EI-3 (lineage integrity)        ──►    RE-3 (budget conservation)

EI-3 (lineage integrity)        ──►    RE-4 (budget monotonicity)

EI-1 (single active instance)   ──►    RE-5 (tick duration limit)

EI-3 (lineage integrity)        ──►    RE-6 (tick determinism)

EI-4 (authority/durability)     ──►    RE-7 (storage isolation)
OA-1 (canonical identity)       ──►    RE-7 (storage isolation)

EI-2 (checkpoint boundary)      ──►    RE-8 (lifecycle order)
OA-2 (authority states)         ──►    RE-8 (lifecycle order)
```

---

## Derivation Explanations

### Single Ticker → Authority Lifecycle → Ownership Sidecar → Migration Ordering → Recovery Fencing

This is the primary dependency chain through the specification:

1. **Single ticker law (EI-1):** The foundational invariant. At most one node ticks a given agent identity.

2. **Authority lifecycle (OA-2, OA-3):** To enforce the single ticker law across nodes, authority must follow a defined state machine with explicit transitions. The lifecycle states (ACTIVE_OWNER, HANDOFF_INITIATED, HANDOFF_PENDING, RETIRED, RECOVERY_REQUIRED) exist to maintain single-ticker across time.

3. **Ownership sidecar (OA-4, OA-5, OA-6):** Authority transfer rules ensure that ownership transitions are explicit, serialized, and complete. The "sidecar" of ownership metadata — who holds authority, when it was granted, and how transfer proceeds — exists to make the single-ticker guarantee verifiable during migration.

4. **Migration ordering (MC-5, MC-6, MC-7, MC-8):** Migration overlap constraints derive from the authority lifecycle. The ordering rules (source stops before target starts, target must not tick during preparation) are the migration-specific expression of authority serialization.

5. **Recovery fencing (OA-7, EI-11, MC-9, FS-1 through FS-4):** When the chain breaks — when authority cannot be determined, when checkpoints diverge, when migration fails — the RECOVERY_REQUIRED state serves as a safety fence. It halts all execution until the single-ticker invariant can be re-established with certainty.

---

## Invariant Coverage Matrix

| Enforcement Invariant | Constitutional Dependencies |
|----------------------|---------------------------|
| RE-1: Atomic checkpoints | EI-2, EI-3 |
| RE-2: State persistence | EI-2, EI-10 |
| RE-3: Budget conservation | EI-3 |
| RE-4: Budget monotonicity | EI-3 |
| RE-5: Tick duration limit | EI-1 |
| RE-6: Tick determinism | EI-3 |
| RE-7: Storage isolation | EI-4, OA-1 |
| RE-8: Lifecycle order | EI-2, OA-2 |

Every enforcement invariant (RE-*) traces to at least one constitutional invariant (EI-*, OA-*, MC-*). No enforcement invariant exists without constitutional justification.

---

## Document Status

**Type:** Mechanism Design Specification
**Scope:** Invariant dependency relationships and derivation chains.
**Authority:** Descriptive — documents relationships between invariants defined in constitutional and enforcement specifications.
