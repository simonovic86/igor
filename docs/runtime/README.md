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
| [THREAT_MODEL.md](./THREAT_MODEL.md) | Canonical threat assumptions and adversary classes |
| [HOSTCALL_ABI.md](./HOSTCALL_ABI.md) | Hostcall interface design and wazero integration |
| [REPLAY_ENGINE.md](./REPLAY_ENGINE.md) | Replay engine design (draft) |
| [LEASE_EPOCH.md](./LEASE_EPOCH.md) | Lease-based authority epochs design (draft) |

## Relationship to Constitution

All runtime implementation must comply with the constitutional invariants. Runtime documents describe enforcement mechanisms; constitutional documents define what must be enforced.
