# Replay Engine Design

**Spec Layer:** Runtime Implementation (Design Draft)
**Status:** Exploratory — not frozen, expected to evolve
**Derives from:** [CAPABILITY_MEMBRANE.md](../constitution/CAPABILITY_MEMBRANE.md) (CM-4), [CAPABILITY_ENFORCEMENT.md](../enforcement/CAPABILITY_ENFORCEMENT.md) (CE-3)

---

## Purpose

This document describes how the Igor runtime can verify tick execution correctness by replaying ticks from a checkpoint and recorded observations, then comparing results against the next checkpoint.

The replay engine exploits the observation determinism guarantee (CM-4) and the tick observation log (CE-3) to provide post-hoc verification of agent behavior. It does not modify the execution model — it adds a verification capability on top of the existing tick/checkpoint cycle.

---

## Conceptual Model

A tick is a deterministic function:

```
f(checkpoint_N, observations) → (checkpoint_N+1, side_effects)
```

If the runtime records all observations during a tick (per CE-3), it can replay the tick from the same checkpoint with the same observations and verify that the resulting state matches the actual checkpoint at N+1.

### What Replay Verifies

- **Agent determinism:** same inputs produce same state transition
- **Checkpoint lineage integrity:** each checkpoint is derivable from its predecessor
- **No hidden state:** agent state is fully captured in checkpoints

### What Replay Does NOT Verify

- **Side effect execution:** replay does not re-execute side effects (network requests, KV writes)
- **Node honesty:** replay verifies agent behavior, not host behavior
- **Observation correctness:** replay verifies that the agent processed observations correctly, not that the observations themselves were truthful

---

## Event Log Structure

The tick observation log records every observation hostcall return value during a tick. This is the input data required for replay.

### Per-Tick Event Log (Conceptual Format)

```
TickEventLog {
  tick_number:   u64          // Monotonic tick counter
  checkpoint_id: [32]byte     // Hash of starting checkpoint
  entry_count:   u32          // Number of observation entries
  entries:       []EventEntry // Ordered sequence of observations
}

EventEntry {
  hostcall_id:    u16    // Identifies which hostcall (clock_now, rand_bytes, etc.)
  payload_length: u32    // Length of recorded return value
  payload:        []byte // Serialized return value
}
```

### Properties

- **Append-only during tick:** entries added as hostcalls execute, never modified
- **Sealed at tick boundary:** log closed before checkpoint commit
- **Ordered:** entries reflect the exact sequence of hostcall invocations
- **Complete:** every observation hostcall is recorded (no sampling)
- **Portable:** event logs are transferred alongside checkpoints during migration

---

## Replay Verification Flow

### Single-Tick Replay

```
1. Load checkpoint at tick N (checkpoint_N)
2. Load event log for tick N+1
3. Initialize replay sandbox:
   - Same WASM binary
   - Same capability set
   - Event log provides observation values (instead of live hostcalls)
4. Execute agent_tick() in replay mode
5. Call agent_checkpoint() to capture resulting state
6. Compare resulting state against checkpoint_N+1
7. Match → verified. Mismatch → divergence detected.
```

### Divergence Response

If replay produces different state than the recorded checkpoint:

- Log divergence with full context (tick number, checkpoint hashes, first differing byte)
- Flag the checkpoint lineage as unverified
- Do NOT automatically halt the agent (replay is verification, not enforcement)
- Report to operator for investigation

Divergence can indicate:
- Non-deterministic agent code (violates RE-6)
- Incomplete observation recording (violates CE-3)
- Checkpoint corruption
- Bug in replay engine itself

---

## Replay Modes

### Local Self-Verification

The hosting node replays its own recent ticks to verify execution correctness. This catches non-deterministic agent behavior and observation recording bugs.

**When to run:** periodically (e.g., every N ticks), on checkpoint rotation, or on demand.

### Cross-Node Verification (Future)

A different node replays ticks from received checkpoints + event logs to verify the originating node's execution. This provides independent verification without trusting the hosting node.

**Requires:**
- Event logs transferred alongside checkpoints
- Verifier has the same WASM binary
- Verifier can instantiate a replay sandbox

**Trust model:** the verifier trusts the WASM binary and the event log format, but does not trust the originating node's execution.

### Selective Replay (Future)

Rather than replaying every tick, replay a statistical sample. This reduces verification cost while still providing probabilistic assurance.

**Trade-off:** lower cost vs. reduced coverage. Sampling strategy is a policy decision, not a constitutional property.

---

## Integration with Existing Components

### Checkpoint Format Extension

The current checkpoint format is:
```
[budget: 8 bytes][pricePerSecond: 8 bytes][agent state: N bytes]
```

For replay support, checkpoints need:
- Tick number (for ordering)
- Checkpoint content hash (for verification)
- Reference to associated event log (or inline event log)

The exact format extension is an implementation decision.

### Migration

Event logs SHOULD be transferred alongside checkpoints during migration. This enables the target node to verify the checkpoint lineage it receives. If event logs are not available, the target node accepts the checkpoint on trust (current behavior).

### Budget Model

Replay execution consumes compute resources. Replay cost is borne by the verifier, not the agent's budget. Replay is a runtime service, not an agent operation.

---

## Open Questions

1. **Event log size budget:** should there be a per-tick limit on event log size? Large observation payloads (e.g., network responses) could produce unbounded logs.

2. **Retention policy:** how long are event logs kept? Options: keep all (full history), sliding window (last N ticks), checkpoint-aligned (keep log for each retained checkpoint).

3. **Compression:** should event logs be compressed? Content-addressed storage (IPFS) would deduplicate identical observations, but inline compression may be needed for network transfer.

4. **Cross-node protocol:** how does a verifier request event logs from the originating node? New libp2p protocol stream, or bundled with checkpoint transfer?

5. **Incentives:** who pays for cross-node verification? Is there an economic incentive for honest verification? This connects to the broader economic model.

6. **Partial replay:** can replay start from any checkpoint, or only from the genesis state? Starting from any checkpoint requires trusting that checkpoint as a valid starting point.

---

## References

- [CAPABILITY_MEMBRANE.md](../constitution/CAPABILITY_MEMBRANE.md) — CM-4 (observation determinism)
- [CAPABILITY_ENFORCEMENT.md](../enforcement/CAPABILITY_ENFORCEMENT.md) — CE-3 (tick observation log)
- [EXECUTION_INVARIANTS.md](../constitution/EXECUTION_INVARIANTS.md) — EI-3 (checkpoint lineage integrity)
- [RUNTIME_ENFORCEMENT_INVARIANTS.md](../enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md) — RE-6 (tick determinism)
- [ARCHITECTURE.md](./ARCHITECTURE.md) — current checkpoint format and execution model
