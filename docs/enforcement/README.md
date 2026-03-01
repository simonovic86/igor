# Enforcement

This folder contains Igor's enforcement specification documents.

## Scope

Enforcement documents define **HOW** constitutional guarantees are upheld through runtime enforcement rules and invariant derivation mappings.

These documents bridge the gap between constitutional guarantees (WHAT must be true) and runtime implementation (HOW the system operates). They describe the enforcement invariants that operationalize constitutional contracts, and the dependency relationships between invariants.

## Constraints

Enforcement documents:

- **Must derive from constitutional guarantees** — every enforcement invariant traces to one or more constitutional invariants.
- **Must not introduce new guarantees** — only constitutional documents may define guarantees.
- **Must not contradict constitutional invariants** — in case of conflict, the constitutional invariant prevails.
- **May describe runtime mechanisms** — enforcement rules may reference implementation-level concepts (e.g., timeout values, file operations) to describe how guarantees are upheld.

## Documents

| Document | Purpose |
|----------|---------|
| [RUNTIME_ENFORCEMENT_INVARIANTS.md](./RUNTIME_ENFORCEMENT_INVARIANTS.md) | Enforcement rules implementing constitutional guarantees (RE-1 through RE-8) |
| [INVARIANT_DEPENDENCY_GRAPH.md](./INVARIANT_DEPENDENCY_GRAPH.md) | Invariant dependency relationships, derivation chains, and cross-document traceability |
| [CAPABILITY_ENFORCEMENT.md](./CAPABILITY_ENFORCEMENT.md) | Capability enforcement rules (CE-1 through CE-6) derived from CAPABILITY_MEMBRANE.md |

## Relationship to Other Layers

- **Constitutional layer** ([docs/constitution/](../constitution/)) — defines WHAT Igor guarantees. Enforcement derives from this.
- **Runtime layer** ([docs/runtime/](../runtime/)) — describes HOW Igor operates. Implementation realizes enforcement rules.
- **Governance layer** ([docs/governance/](../governance/)) — defines HOW Igor evolves. Spec changes governed by [SPEC_GOVERNANCE.md](../governance/SPEC_GOVERNANCE.md).
