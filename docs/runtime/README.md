# Runtime

This folder contains Igor's runtime architecture and operational documentation.

## Scope

Runtime documents describe **HOW** Igor implements the guarantees defined in the [constitutional specification](../constitution/).

These documents may include architecture descriptions, algorithms, protocol details, operational flows, and implementation-level constraints.

## Documents

| Document | Purpose |
|----------|---------|
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Runtime implementation details and system structure |
| [AGENT_LIFECYCLE.md](./AGENT_LIFECYCLE.md) | Building and deploying agents |
| [MIGRATION_PROTOCOL.md](./MIGRATION_PROTOCOL.md) | P2P migration protocol mechanics |
| [BUDGET_MODEL.md](./BUDGET_MODEL.md) | Economic model and execution metering |
| [SECURITY_MODEL.md](./SECURITY_MODEL.md) | Threat model and sandbox constraints |

## Relationship to Constitution

All runtime implementation must comply with the constitutional invariants. Runtime documents describe enforcement mechanisms; constitutional documents define what must be enforced.
