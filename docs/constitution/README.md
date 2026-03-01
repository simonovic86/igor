# Constitution

This folder contains Igor's constitutional specification documents.

## Scope

Constitutional documents define Igor's **non-negotiable execution guarantees** — the invariants that the runtime MUST uphold regardless of implementation strategy.

These documents describe **WHAT** Igor must guarantee. They do not describe how guarantees are enforced or implemented.

## Constraints

Constitutional documents:

- **Must remain mechanism-agnostic** — no code, no algorithms, no performance characteristics.
- **Must remain implementation-agnostic** — no references to specific runtime components or operational procedures.
- **Must remain field-agnostic** — no protocol message fields, wire formats, or serialization schemas.
- **Must use precise normative language** — MUST, MUST NOT, MAY per RFC 2119.
- **Must not include enforcement derivation** — enforcement rules and derivation mappings belong in the [enforcement layer](../enforcement/).
- **Must not include runtime mechanics** — implementation details belong in the [runtime layer](../runtime/).

## Documents

| Document | Purpose |
|----------|---------|
| [RUNTIME_CONSTITUTION.md](./RUNTIME_CONSTITUTION.md) | Constitutional specification root |
| [EXECUTION_INVARIANTS.md](./EXECUTION_INVARIANTS.md) | Foundational runtime invariants (EI-1 through EI-11) |
| [OWNERSHIP_AND_AUTHORITY.md](./OWNERSHIP_AND_AUTHORITY.md) | Authority lifecycle model (OA-1 through OA-7) |
| [MIGRATION_CONTINUITY.md](./MIGRATION_CONTINUITY.md) | Migration continuity contracts (MC-1 through MC-9, FS-1 through FS-4) |
| [AUTHORITY_STATE_MACHINE.md](./AUTHORITY_STATE_MACHINE.md) | Formal authority lifecycle state machine and tick permission matrix |
| [CAPABILITY_MEMBRANE.md](./CAPABILITY_MEMBRANE.md) | Capability membrane invariants (CM-1 through CM-7) |

## Related Layers

- **Enforcement layer** ([docs/enforcement/](../enforcement/)) — defines HOW constitutional guarantees are upheld.
- **Specification index** ([docs/SPEC_INDEX.md](../SPEC_INDEX.md)) — cross-reference index for all specification documents.

## Change Process

Changes to constitutional documents are governed by [SPEC_GOVERNANCE.md](../governance/SPEC_GOVERNANCE.md) and require formal review.
