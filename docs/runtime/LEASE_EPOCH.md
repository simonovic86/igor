# Lease-Based Authority Epochs

**Spec Layer:** Runtime Implementation (Design Draft)
**Status:** Exploratory — not frozen, expected to evolve
**Builds on:** [AUTHORITY_STATE_MACHINE.md](../constitution/AUTHORITY_STATE_MACHINE.md), [OWNERSHIP_AND_AUTHORITY.md](../constitution/OWNERSHIP_AND_AUTHORITY.md)

---

## Purpose

This document describes how authority epochs can be time-bounded through leases, adding liveness guarantees to the safety guarantees already provided by the authority state machine.

The lease mechanism does not introduce new authority states or modify the constitutional state machine. It adds a time-based trigger for existing state transitions — specifically, a mechanism for detecting that an ACTIVE_OWNER has become unresponsive and transitioning to RECOVERY_REQUIRED.

---

## Motivation

The current authority model guarantees safety (at-most-one ticker, per EI-1 and EI-5) but provides no liveness bound. A crashed ACTIVE_OWNER holds authority indefinitely with no mechanism for other nodes to detect the failure and reclaim the agent.

In the current implementation, this is acceptable: if Node A crashes while holding authority, the operator restarts the node and the agent resumes from its last checkpoint. But in a multi-node permissionless network, this creates a liveness problem — the agent is stuck on a dead node with no automated recovery path.

Leases solve this: authority expires if not renewed, enabling automated detection of unresponsive nodes and transition to recovery.

---

## Lease Model

### Core Concepts

- **Lease:** A time-bounded grant of execution authority. The authority holder MUST renew the lease before it expires.
- **Lease duration:** Configurable period (e.g., 30 seconds, 5 minutes). Shorter leases improve failure detection but increase renewal overhead.
- **Lease renewal:** The authority holder extends its lease without changing authority. Renewal is a local operation that does not require network coordination.
- **Lease expiry:** When a lease expires without renewal, the agent transitions to RECOVERY_REQUIRED per the existing state machine.

### Relationship to Authority State Machine

The lease mechanism maps onto existing authority states without modification:

| Event | State Transition | State Machine Reference |
|-------|-----------------|------------------------|
| Lease granted | → ACTIVE_OWNER | Normal authority acquisition |
| Lease renewed | ACTIVE_OWNER → ACTIVE_OWNER | No state change; lease timer reset |
| Lease expired | ACTIVE_OWNER → RECOVERY_REQUIRED | Equivalent to "authority ambiguity detected" |
| Recovery resolved | RECOVERY_REQUIRED → ACTIVE_OWNER | Normal recovery flow |

The constitutional state machine already defines RECOVERY_REQUIRED as the state entered when authority is ambiguous. A lease expiry is one mechanism for detecting that ambiguity — the node is presumed unresponsive if it cannot renew its lease.

---

## Epoch Advancement

### Epoch Structure

The authority epoch is a strictly monotonic identifier that advances on authority transitions. With leases, epochs have two components:

```
epoch = (major_version, lease_generation)

major_version:     Increments on authority transfer (new node takes over)
lease_generation:  Increments on lease renewal (same node, new lease period)
```

### Epoch Ordering

Epochs are totally ordered:
- Higher major_version always supersedes lower
- Within the same major_version, higher lease_generation supersedes lower
- This ordering enables stale-node detection: a node presenting an older epoch is unambiguously outdated

### Epoch in Checkpoint Metadata

The current epoch is included in checkpoint metadata:
- Checkpoint signed with epoch identifier
- Nodes can verify checkpoint freshness by comparing epochs
- Stale checkpoints (from expired leases) are identifiable

---

## Anti-Clone Enforcement

Leases provide a mechanism for anti-clone enforcement without distributed consensus:

### Stale Node Detection

A node with an expired lease MUST NOT resume ticking. Before entering ACTIVE_OWNER, a node MUST verify:

1. It holds a valid (non-expired) lease
2. Its epoch is not superseded by a higher epoch from another node
3. No RECOVERY_REQUIRED state has been triggered for this agent

### Split-Brain Prevention

If two nodes both believe they hold authority (e.g., after a network partition):

1. Each node's lease has a different lease_generation
2. At most one lease is valid at any given time
3. The node with the expired lease detects the expiry and enters RECOVERY_REQUIRED
4. The node with the valid lease continues ticking

This relies on rough clock synchronization (see Open Questions).

### Migration with Leases

During migration:
1. Source node holds lease with epoch (M, G)
2. Source initiates handoff → HANDOFF_INITIATED
3. Target receives package, including lease metadata
4. Target acquires new lease with epoch (M+1, 0)
5. Source's lease is invalidated by the major_version increment
6. Source transitions to RETIRED

The major_version increment ensures the source's lease is unambiguously superseded, even if the source's old lease hasn't technically expired.

---

## Lease Renewal Protocol

### Local Renewal (v1)

In the simplest model, lease renewal is a local operation:

1. Node checks remaining lease time before each tick
2. If lease is within renewal window (e.g., 50% of duration remaining), node renews
3. Renewal updates the lease_generation in checkpoint metadata
4. Renewal is recorded in the checkpoint but does not require network communication

**Advantage:** Simple, no network overhead.
**Limitation:** No external observer can verify the lease is still held.

### Network-Confirmed Renewal (Future)

In a multi-node network with untrusted peers:

1. Node broadcasts lease renewal to known peers
2. Peers record the renewal with timestamp
3. If peers don't receive renewal before expiry, they can trigger recovery
4. This provides external verification of liveness

**Advantage:** External nodes can detect failure.
**Limitation:** Requires network connectivity for renewal; partition can cause false expiry.

---

## Configuration

Lease parameters are runtime configuration, not constitutional properties:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `lease_duration` | 60s | Time before lease expires without renewal |
| `renewal_window` | 0.5 | Fraction of duration at which renewal triggers |
| `grace_period` | 10s | Additional time after expiry before recovery triggers |

**Trade-offs:**
- **Short leases** (5-30s): fast failure detection, higher renewal overhead, sensitive to clock skew
- **Long leases** (1-10min): low overhead, slow failure detection, more tolerant of clock issues
- **Grace period:** absorbs transient delays (GC pauses, temporary load) that might cause missed renewals

---

## Open Questions

1. **Clock synchronization:** leases assume nodes have roughly synchronized clocks. How much clock skew is tolerable? NTP provides ~10ms accuracy; is that sufficient? Should lease durations be set conservatively to account for clock drift?

2. **Partition behavior:** if a node is partitioned but still functioning, it continues ticking with a valid local lease. When the partition heals, there may be a brief period where both the original node and a recovery node believe they have authority. How is this resolved? (The safety answer: both detect the conflict and enter RECOVERY_REQUIRED.)

3. **Lease storage:** where is the authoritative lease state stored? Options: in the checkpoint metadata (portable but only as fresh as the last checkpoint), in a separate lease store (more current but adds complexity), or broadcast to peers (distributed but eventually consistent).

4. **Interaction with budget:** should lease renewal cost budget? If so, agents with very low budgets might not be able to afford lease renewals. If not, lease renewals are "free" operations that don't reflect real resource consumption.

5. **Bootstrap:** when an agent starts for the first time (no prior lease), how is the initial lease acquired? Simplest answer: the loading node grants itself a lease at instantiation time.

6. **Lease transfer vs. new lease:** during migration, should the target receive the source's remaining lease time, or start a fresh lease? Fresh lease is simpler and avoids time-transfer complications.

---

## References

- [AUTHORITY_STATE_MACHINE.md](../constitution/AUTHORITY_STATE_MACHINE.md) — Authority states and transitions
- [OWNERSHIP_AND_AUTHORITY.md](../constitution/OWNERSHIP_AND_AUTHORITY.md) — Authority lifecycle model (OA-1 through OA-7)
- [EXECUTION_INVARIANTS.md](../constitution/EXECUTION_INVARIANTS.md) — EI-1 (single active instance), EI-5 (singular authority), EI-6 (safety over liveness)
- [MIGRATION_CONTINUITY.md](../constitution/MIGRATION_CONTINUITY.md) — Migration safety contracts
- [THREAT_MODEL.md](./THREAT_MODEL.md) — Failure classes and adversary model
