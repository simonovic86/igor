# Specification Governance

## Purpose

This document defines the governance rules for Igor's specification documents. It establishes change control processes, specification layer classification, and constraints that prevent specification drift.

---

## Specification Layers

Igor's specification is organized into three layers. Each layer has distinct responsibilities and change constraints.

### Constitutional Layer

**Documents:**
- [RUNTIME_CONSTITUTION.md](../constitution/RUNTIME_CONSTITUTION.md)
- [EXECUTION_INVARIANTS.md](../constitution/EXECUTION_INVARIANTS.md)
- [OWNERSHIP_AND_AUTHORITY.md](../constitution/OWNERSHIP_AND_AUTHORITY.md)
- [MIGRATION_CONTINUITY.md](../constitution/MIGRATION_CONTINUITY.md)

**Contains:**
- Non-negotiable runtime guarantees
- Formal invariant definitions
- Authority lifecycle contracts
- Migration continuity contracts
- Safety semantics

**Constraints:**
- MUST remain mechanism-agnostic.
- MUST remain field-agnostic.
- MUST NOT reference protocol message fields, wire formats, or serialization schemas.
- MUST NOT contain implementation hints, code examples, or enforcement mechanisms.
- MUST use RFC 2119 language (MUST, MUST NOT, MAY, SHOULD) precisely.

### Mechanism Design Layer

**Documents:**
- [RUNTIME_ENFORCEMENT_INVARIANTS.md](../enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md)
- [INVARIANT_DEPENDENCY_GRAPH.md](../enforcement/INVARIANT_DEPENDENCY_GRAPH.md)

**Contains:**
- Enforcement rules implementing constitutional guarantees
- Invariant derivation chains
- Detection and verification mechanisms
- Operational failure modes and responses

**Constraints:**
- Every enforcement invariant MUST trace to one or more constitutional invariants.
- MUST NOT introduce guarantees that are not derivable from the constitutional layer.
- MAY reference implementation-level concepts (e.g., timeout values, file operations) to describe enforcement mechanisms.
- MUST NOT contradict constitutional invariants.

### Runtime Implementation Layer

**Documents:**
- [ARCHITECTURE.md](../runtime/ARCHITECTURE.md)
- [MIGRATION_PROTOCOL.md](../runtime/MIGRATION_PROTOCOL.md)
- [AGENT_LIFECYCLE.md](../runtime/AGENT_LIFECYCLE.md)
- [BUDGET_MODEL.md](../runtime/BUDGET_MODEL.md)
- [SECURITY_MODEL.md](../runtime/SECURITY_MODEL.md)
- Source code and tests

**Contains:**
- Implementation details
- Protocol message formats
- Wire formats and serialization
- Code-level enforcement
- Operational procedures

**Constraints:**
- MUST comply with all constitutional and enforcement invariants.
- MAY define implementation-specific details not covered by higher layers.
- MUST NOT weaken guarantees defined at higher layers.

---

## Specification Freeze Rules

### Phase 1 Constitutional Freeze

Constitutional specification documents are under specification freeze. Changes to constitutional documents require the following process:

#### Change Requirements

1. **RFC Proposal** — A written proposal describing the proposed change, its motivation, and its impact on existing invariants. The proposal must identify which invariants are affected and why the change is necessary.

2. **Compatibility Impact Statement** — An explicit assessment of how the proposed change affects:
   - Existing constitutional invariants (additions, modifications, removals)
   - Enforcement invariants that derive from affected constitutional invariants
   - Implementation code that relies on affected invariants
   - Specification documents that cross-reference affected definitions

3. **Version Bump** — Constitutional document changes MUST increment the specification version. The version is tracked in the document status section of each constitutional document.

4. **Explicit Changelog Entry** — Every constitutional change MUST be recorded with:
   - Date of change
   - Invariant(s) affected
   - Nature of change (addition, modification, removal, clarification)
   - Justification

#### What Constitutes a Constitutional Change

The following require the full change process:

- Adding a new constitutional invariant
- Modifying the definition of an existing invariant
- Removing or weakening an existing invariant
- Changing authority lifecycle states or transitions
- Modifying migration safety guarantees
- Altering the foundational separation (checkpoint lineage / ownership authority)

The following do NOT require the full change process:

- Fixing typographical errors
- Improving clarity without changing meaning
- Adding cross-references to existing documents
- Updating document status metadata

---

## Prohibition: Constitutional Field References

Constitutional documents MUST NOT reference protocol message fields.

This means constitutional documents MUST NOT contain:

- Protocol buffer field names or numbers
- JSON field names or schemas
- Binary encoding formats or byte offsets
- Serialization library references
- Wire protocol opcodes or message types

Constitutional invariants describe *what* must be true, not *how* it is encoded or transmitted. Field-level details belong in the runtime implementation layer.

---

## Governance Summary

| Layer | Change Process | Field References | Mechanism Detail |
|-------|---------------|-----------------|-----------------|
| Constitutional | RFC + impact + version bump + changelog | Prohibited | Prohibited |
| Mechanism Design | Review required, must trace to constitutional | Permitted in enforcement context | Required |
| Runtime Implementation | Standard development process | Required | Required |

---

## Document Status

**Type:** Specification Governance
**Scope:** Change control and classification rules for all Igor specification documents.
**Authority:** Governs the process by which all specification documents may be modified.
