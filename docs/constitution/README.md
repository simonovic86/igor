# Constitution

This folder contains Igor's constitutional specification documents.

## Scope

Constitutional documents define Igor's **non-negotiable execution guarantees** — the invariants that the runtime MUST uphold regardless of implementation strategy.

These documents describe **WHAT** Igor must guarantee, not how those guarantees are implemented.

## Constraints

Constitutional documents:

- **Must remain implementation-agnostic** — no code, no algorithms, no performance characteristics.
- **Must remain field-agnostic** — no protocol message fields, wire formats, or serialization schemas.
- **Must use precise normative language** — MUST, MUST NOT, MAY per RFC 2119.
- **Must not reference enforcement mechanisms** — those belong in the enforcement invariants.

## Documents

| Document | Purpose |
|----------|---------|
| [RUNTIME_CONSTITUTION.md](./RUNTIME_CONSTITUTION.md) | Constitutional specification root |
| [EXECUTION_INVARIANTS.md](./EXECUTION_INVARIANTS.md) | Foundational runtime invariants (EI-1 through EI-11) |
| [OWNERSHIP_AND_AUTHORITY.md](./OWNERSHIP_AND_AUTHORITY.md) | Authority lifecycle model (OA-1 through OA-7) |
| [MIGRATION_CONTINUITY.md](./MIGRATION_CONTINUITY.md) | Migration continuity contracts (MC-1 through MC-9, FS-1 through FS-4) |
| [RUNTIME_ENFORCEMENT_INVARIANTS.md](./RUNTIME_ENFORCEMENT_INVARIANTS.md) | Enforcement rules implementing constitutional guarantees (RE-1 through RE-8) |
| [INVARIANT_DEPENDENCY_GRAPH.md](./INVARIANT_DEPENDENCY_GRAPH.md) | Invariant dependency relationships and derivation chains |

## Change Process

Changes to constitutional documents are governed by [SPEC_GOVERNANCE.md](../governance/SPEC_GOVERNANCE.md) and require formal review.
