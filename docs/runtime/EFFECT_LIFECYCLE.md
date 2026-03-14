# Effect Lifecycle Specification

Status: **Draft v1** — executable truth, not strategy.

Derives from: EI-6 (Safety Over Liveness), CM-1 (Total Mediation), RE-1 (Atomic Checkpoints).

## Problem

Agents perform irreversible external actions (transfers, API calls, state mutations).
When a crash occurs between initiating and confirming such an action, the agent
must know — on resume — that the action may have already happened. Blind retry
risks duplication. Blind skip risks omission. The ugly middle is the whole movie.

Today, the reconciler agent solves this manually with `IntentRecorded bool`,
`FinalizeExecuted bool`, `DupChecked bool`. Every agent that touches the outside
world needs the same machinery. Until effects are a runtime primitive, Igor's
core promise is social convention.

## Design Constraint: Effects Live in Agent State

The agent runs inside WASM. The SDK runs inside WASM. There is no durable path
from the SDK to the runtime except through `Marshal() []byte`. Therefore:

- Effect tracking state serializes as part of agent state.
- Effect tracking state checkpoints and resumes with the agent.
- Effect tracking state migrates with the agent automatically.
- The runtime does not need a separate effect journal for v1.

This is not a limitation. It is the correct design for portable actors.
The agent IS its checkpoint. Effects are part of what happened to the agent.

## Intent Lifecycle

An **intent** is a declared plan to perform an irreversible external action.

### States

```
  Recorded ──→ InFlight ──→ Confirmed
                  │
               [crash]
                  │
                  ▼
              Unresolved ──→ Confirmed     (it happened)
                         ──→ Compensated   (it didn't, or we fixed it)
```

| State | Meaning |
|---|---|
| **Recorded** | Agent has declared intent. Not yet attempted. Checkpoint captures this. |
| **InFlight** | Agent has begun the external action (HTTP call, tx submission). |
| **Confirmed** | External action verified complete. Terminal state. |
| **Unresolved** | On resume, any InFlight intent transitions here automatically. The agent does not know if the action happened. This is the swamp. |
| **Compensated** | Agent investigated an Unresolved intent and determined it either did not happen or has been compensated. Terminal state. |

### Allowed Transitions

```
Recorded    → InFlight      (agent begins execution)
InFlight    → Confirmed     (agent verifies success within same run)
InFlight    → Unresolved    (AUTOMATIC on resume — not agent-initiated)
Unresolved  → Confirmed     (agent checks external state, confirms it happened)
Unresolved  → Compensated   (agent checks external state, it didn't happen or was rolled back)
Confirmed   → (terminal)
Compensated → (terminal)
```

No other transitions are valid. In particular:
- `Unresolved` cannot go back to `InFlight`. You don't retry blindly. You reconcile.
- `Recorded` cannot skip to `Confirmed`. You must go through `InFlight`.
- `Confirmed` and `Compensated` are final. An intent does not reopen.

### The Resume Rule

On `Unmarshal`, every intent in `InFlight` state becomes `Unresolved`.

This is the single most important rule in the effect model. It encodes the
epistemological reality: after a crash, you do not know whether an in-flight
action completed. The runtime forces you to confront that uncertainty rather
than pretending it didn't happen.

## IntentID

Each intent has a unique key chosen by the agent. The SDK does not generate IDs
because the agent is the authority on what constitutes a distinct action.

- Type: `[]byte` (variable length, agent-chosen)
- Must be unique within the agent's effect log at any given time
- Typically: idempotency key, transaction hash, nonce, or descriptive tag
- Used for lookup, deduplication, and external correlation

## SDK API

### Types

```go
// IntentState represents the lifecycle state of an intent.
type IntentState uint8

const (
    Recorded    IntentState = 1
    InFlight    IntentState = 2
    Confirmed   IntentState = 3
    Unresolved  IntentState = 4
    Compensated IntentState = 5
)

// Intent is an individual effect intent tracked by the EffectLog.
type Intent struct {
    ID    []byte
    State IntentState
    Data  []byte  // Agent-defined payload (tx params, API call details, etc.)
}

// EffectLog tracks intents across checkpoint/resume cycles.
type EffectLog struct {
    intents []Intent
}
```

### Operations

```go
// Record declares a new intent. Returns error if ID already exists.
// The intent starts in Recorded state.
func (e *EffectLog) Record(id, data []byte) error

// Begin transitions an intent from Recorded to InFlight.
// Call this immediately before performing the external action.
func (e *EffectLog) Begin(id []byte) error

// Confirm transitions an intent from InFlight or Unresolved to Confirmed.
// Call this after verifying the external action succeeded.
func (e *EffectLog) Confirm(id []byte) error

// Compensate transitions an intent from Unresolved to Compensated.
// Call this after determining the action did not happen or was rolled back.
func (e *EffectLog) Compensate(id []byte) error

// Pending returns all intents not in a terminal state (Recorded, InFlight, Unresolved).
func (e *EffectLog) Pending() []Intent

// Unresolved returns only intents in Unresolved state.
// This is what the agent should check on resume.
func (e *EffectLog) Unresolved() []Intent

// Get returns the intent with the given ID, or nil.
func (e *EffectLog) Get(id []byte) *Intent

// Prune removes all terminal intents (Confirmed, Compensated).
// Call periodically to prevent unbounded growth.
// Returns the number of intents removed.
func (e *EffectLog) Prune() int
```

### Serialization

The EffectLog participates in the agent's `Marshal`/`Unmarshal` cycle.

```go
// Marshal serializes the effect log. Append this to your agent state.
func (e *EffectLog) Marshal() []byte

// Unmarshal restores the effect log from serialized bytes.
// CRITICAL: All InFlight intents become Unresolved during unmarshal.
// This is the resume rule.
func (e *EffectLog) Unmarshal(data []byte)
```

Wire format (little-endian):
```
[count: uint32]
  for each intent:
    [state: uint8]
    [id_len: uint32][id: N bytes]
    [data_len: uint32][data: N bytes]
```

### Checkpoint Relationship

The EffectLog is part of agent state. The checkpoint boundary defines
what the runtime considers durable truth:

- **Before checkpoint**: intent state changes are volatile. A crash
  reverts to the last checkpoint.
- **After checkpoint**: intent state is durable. Resume restores it.

This means the correct pattern is:

```
Tick N:   Record intent → return (checkpoint happens)
Tick N+1: Begin intent → perform action → Confirm intent → return
```

If the agent records an intent and performs the action in the same tick,
a crash after the action but before the next checkpoint loses the
Recorded→InFlight transition. The intent won't appear on resume at all.
The agent performed an untracked external action. This is a bug.

**Rule: Always checkpoint between Record and Begin.**

The runtime checkpoints every 5 seconds (configurable). At 1 Hz tick rate,
this means at least 4 ticks between actions. In practice, Record in one tick,
Begin in the next tick, and trust the checkpoint interval.

## Agent Usage Pattern

```go
type Sentinel struct {
    Effects  igor.EffectLog
    // ... other state
}

func (s *Sentinel) Tick() bool {
    // On resume, handle unresolved intents first.
    for _, intent := range s.Effects.Unresolved() {
        s.reconcile(intent)
        return true
    }

    // Normal operation...
    if needsRefill() {
        id := generateIdempotencyKey()
        s.Effects.Record(id, transferParams)
        return true  // checkpoint will happen before next tick
    }

    // In a later tick, execute the recorded intent.
    for _, intent := range s.Effects.Pending() {
        if intent.State == igor.Recorded {
            s.Effects.Begin(intent.ID)
            result := executeTransfer(intent.Data)
            if result.OK {
                s.Effects.Confirm(intent.ID)
            }
            return true
        }
    }
    return false
}

func (s *Sentinel) reconcile(intent igor.Intent) {
    // Check external state to determine if the action happened.
    status := checkBridgeStatus(intent.Data)
    if status.Completed {
        s.Effects.Confirm(intent.ID)
    } else if status.NotFound {
        s.Effects.Compensate(intent.ID)
    }
    // If status is ambiguous, leave as Unresolved and check next tick.
}

func (s *Sentinel) Marshal() []byte {
    return igor.NewEncoder(256).
        Bytes(s.Effects.Marshal()).
        // ... other fields
        Finish()
}

func (s *Sentinel) Unmarshal(data []byte) {
    d := igor.NewDecoder(data)
    s.Effects.Unmarshal(d.Bytes())
    // ... other fields
}
```

## What the Runtime Owns vs. What the SDK Owns

| Concern | Owner |
|---|---|
| Checkpoint timing | Runtime |
| Checkpoint atomicity (RE-1) | Runtime |
| Effect state storage | SDK (inside agent state) |
| Intent state transitions | SDK (validates transitions) |
| InFlight→Unresolved on resume | SDK (automatic in Unmarshal) |
| Reconciliation logic | Agent (application-specific) |
| IntentID generation | Agent |
| External action execution | Agent (via hostcalls) |
| Pruning terminal intents | Agent (calls Prune) |

## Future: Runtime Awareness

In a later phase, we may add a hostcall for the agent to report effects
to the runtime for audit and inspection:

```
igor.effect_report(intent_id, state, data) → void
```

This would let `igord inspect` show the effect journal without parsing
agent-specific state. But this is not required for v1. The agent's own
state is the source of truth.

## Open Design Questions (Resolved)

**Q: Should the runtime manage effects outside agent state?**
A: No. The WASM boundary means agent state is the only durable path.
   External effect journals are a future audit feature, not the truth source.

**Q: Can an agent have multiple in-flight intents?**
A: Yes. The EffectLog is a list, not a single slot. An agent monitoring
   multiple positions might have several concurrent intents.

**Q: What about nested effects (effect B depends on effect A)?**
A: Model as separate intents with application-level ordering.
   The SDK provides a flat list; the agent owns the dependency graph.

**Q: Should the SDK enforce Record-before-Begin across ticks?**
A: No. The SDK validates state transitions (can't Begin from Confirmed).
   Enforcing the "checkpoint between Record and Begin" rule is a convention,
   not a runtime constraint. Agents that violate it risk untracked effects.
   Documentation makes this clear; the SDK does not police tick boundaries.
