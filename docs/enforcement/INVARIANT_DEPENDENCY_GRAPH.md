# Invariant Dependency Graph

> **Specification cross-references:** [Spec Index](../SPEC_INDEX.md) | [Runtime Enforcement Invariants](./RUNTIME_ENFORCEMENT_INVARIANTS.md) | [Runtime Constitution](../constitution/RUNTIME_CONSTITUTION.md)
>
> This document maps relationships between invariants without redefining them. All invariant definitions reside in their authoritative constitutional or enforcement documents.

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

## Cross-Document Traceability

### Root Constitutional Invariants

#### Single Active Ticker Law (EI-1)

- **Defined in:** EXECUTION_INVARIANTS.md
- **Referenced in:** OWNERSHIP_AND_AUTHORITY.md, MIGRATION_CONTINUITY.md, RUNTIME_CONSTITUTION.md
- **Enforced by:** RE-5 (tick duration limit)
- **Validated by:** Single-instance detection via log analysis, process counting

#### Checkpoint Lineage Canonicality (EI-3)

- **Defined in:** EXECUTION_INVARIANTS.md
- **Referenced in:** MIGRATION_CONTINUITY.md (MC-3), RUNTIME_CONSTITUTION.md
- **Enforced by:** RE-1 (atomic checkpoints), RE-3 (budget conservation), RE-4 (budget monotonicity), RE-6 (tick determinism)
- **Validated by:** Checkpoint format verification, lineage chain integrity checks

#### Authority Uniqueness per Epoch (EI-5)

- **Defined in:** EXECUTION_INVARIANTS.md
- **Referenced in:** OWNERSHIP_AND_AUTHORITY.md (OA-2 through OA-7), MIGRATION_CONTINUITY.md (MC-5)
- **Enforced by:** Authority lifecycle state machine (OA-2, OA-3), transfer serialization (OA-5)
- **Validated by:** Conflicting authority detection triggering RECOVERY_REQUIRED (OA-7, EI-11)

#### Safety-Over-Liveness Rule (EI-6)

- **Defined in:** EXECUTION_INVARIANTS.md
- **Referenced in:** OWNERSHIP_AND_AUTHORITY.md, MIGRATION_CONTINUITY.md (FS-1 through FS-4)
- **Enforced by:** RECOVERY_REQUIRED state (OA-7), fail-stop behavior (EI-7)
- **Validated by:** Execution halt under ambiguity, migration pause acceptance (EI-9)

### Derived Enforcement Invariants

#### RE-1: Atomic Checkpoints

- **Defined in:** RUNTIME_ENFORCEMENT_INVARIANTS.md
- **Derives from:** EI-2 (checkpoint boundary resumption), EI-3 (lineage integrity)
- **Referenced in:** INVARIANT_DEPENDENCY_GRAPH.md
- **Validated by:** Checkpoint read/write error detection, format validation

#### RE-2: State Persistence

- **Defined in:** RUNTIME_ENFORCEMENT_INVARIANTS.md
- **Derives from:** EI-2 (checkpoint boundary resumption), EI-10 (migration checkpoint continuity)
- **Referenced in:** INVARIANT_DEPENDENCY_GRAPH.md
- **Validated by:** Resume success verification, checkpoint presence after shutdown

#### RE-3: Budget Conservation

- **Defined in:** RUNTIME_ENFORCEMENT_INVARIANTS.md
- **Derives from:** EI-3 (lineage integrity)
- **Referenced in:** INVARIANT_DEPENDENCY_GRAPH.md
- **Validated by:** Budget sum tracking across migrations

#### RE-4: Budget Monotonicity

- **Defined in:** RUNTIME_ENFORCEMENT_INVARIANTS.md
- **Derives from:** EI-3 (lineage integrity)
- **Referenced in:** INVARIANT_DEPENDENCY_GRAPH.md
- **Validated by:** Budget increase detection in logs

#### RE-5: Tick Duration Limit

- **Defined in:** RUNTIME_ENFORCEMENT_INVARIANTS.md
- **Derives from:** EI-1 (single active instance)
- **Referenced in:** INVARIANT_DEPENDENCY_GRAPH.md
- **Validated by:** Timeout error detection, agent termination logs

#### RE-6: Tick Determinism

- **Defined in:** RUNTIME_ENFORCEMENT_INVARIANTS.md
- **Derives from:** EI-3 (lineage integrity)
- **Referenced in:** INVARIANT_DEPENDENCY_GRAPH.md
- **Validated by:** Manual testing, cross-run comparison (agent contract)

#### RE-7: Storage Isolation

- **Defined in:** RUNTIME_ENFORCEMENT_INVARIANTS.md
- **Derives from:** EI-4 (authority/durability separation), OA-1 (canonical identity)
- **Referenced in:** INVARIANT_DEPENDENCY_GRAPH.md
- **Validated by:** Checkpoint collision detection, wrong-state-loaded errors

#### RE-8: Lifecycle Order

- **Defined in:** RUNTIME_ENFORCEMENT_INVARIANTS.md
- **Derives from:** EI-2 (checkpoint boundary resumption), OA-2 (authority lifecycle states)
- **Referenced in:** INVARIANT_DEPENDENCY_GRAPH.md
- **Validated by:** Agent crash detection, undefined behavior logs

---

## Document Status

**Type:** Enforcement Specification
**Scope:** Invariant dependency relationships, derivation chains, and cross-document traceability.
**Authority:** Descriptive — documents relationships between invariants defined in constitutional and enforcement specifications. Part of the enforcement layer; see [SPEC_INDEX.md](../SPEC_INDEX.md) for full specification map.
