# Runtime Constitution

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

---

## Derivation Rule

All runtime enforcement invariants MUST derive from constitutional invariants.

Runtime enforcement rules — such as checkpoint atomicity mechanisms, budget monotonicity enforcement, tick duration limits, and sandbox isolation — implement and operationalize the constitutional guarantees. They describe *how* the runtime upholds the constitution, not *what* the constitution requires.

The enforcement layer is defined in [RUNTIME_ENFORCEMENT_INVARIANTS.md](./RUNTIME_ENFORCEMENT_INVARIANTS.md). Every enforcement invariant must trace its justification to one or more constitutional invariants defined in the documents listed above.

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
