# Agent Lifecycle

## Overview

Igor agents implement a deterministic lifecycle with four required functions:

1. `agent_init()` - One-time initialization
2. `agent_tick()` - Periodic execution
3. `agent_checkpoint()` - State serialization
4. `agent_resume(ptr, len)` - State restoration

All agents must export these functions for the runtime to call.

## Lifecycle Phases

### Phase 1: Loading

```
WASM Binary → Compile → Instantiate → Verify Exports → Instance Created
```

**Runtime Actions:**
- Read WASM binary from file
- Compile with wazero
- Instantiate module (no auto-start)
- Verify required exports exist
- Create agent instance with budget

**Agent Actions:**
- None (not yet running)

### Phase 2: Initialization

```
Instance → agent_init() → Initialized
```

**Runtime Actions:**
- Call `agent_init()` export
- Log initialization

**Agent Actions:**
- Initialize internal state
- Set up data structures
- Prepare for first tick

**Example:**
```go
//export agent_init
func agent_init() {
    state.Counter = 0
    fmt.Println("[agent] Initialized")
}
```

### Phase 3: Execution (Tick Loop)

```
Loop: Tick → Meter → Budget Check → Checkpoint (periodic) → Tick ...
```

**Runtime Actions:**
- Call `agent_tick()` every 1 second
- Enforce 100ms timeout
- Measure execution duration
- Calculate cost
- Deduct from budget
- Log metrics
- Checkpoint every 5 seconds

**Agent Actions:**
- Execute one unit of work
- Update internal state
- Must complete quickly (<100ms)

**Example:**
```go
//export agent_tick
func agent_tick() {
    state.Counter++
    fmt.Printf("[agent] Tick %d\n", state.Counter)
}
```

**Budget Metering:**
```go
durationSeconds := elapsed.Seconds()
cost := durationSeconds × pricePerSecond
budget -= cost
```

### Phase 4: Checkpointing

```
agent_checkpoint() → Size
agent_checkpoint_ptr() → Pointer
Read from WASM Memory → Serialize
```

**Runtime Actions:**
- Call `agent_checkpoint()` to get size
- Call `agent_checkpoint_ptr()` to get pointer
- Read state from WASM memory
- Add budget metadata: `[budget:8][price:8][state:N]`
- Save via storage provider (atomic write)

**Agent Actions:**
- Serialize internal state
- Return pointer and size

**Example:**
```go
var stateBuf [8]byte

//export agent_checkpoint
func agent_checkpoint() uint32 {
    binary.LittleEndian.PutUint64(stateBuf[:], state.Counter)
    return 8 // size
}

//export agent_checkpoint_ptr
func agent_checkpoint_ptr() uint32 {
    return uint32(uintptr(unsafe.Pointer(&stateBuf[0])))
}
```

### Phase 5: Resumption

```
Load Checkpoint → Parse Metadata → agent_resume(ptr, len) → Resumed
```

**Runtime Actions:**
- Load checkpoint from storage
- Parse budget metadata
- Restore budget and price
- Allocate WASM memory via `malloc`
- Copy state to WASM memory
- Call `agent_resume(ptr, len)`

**Agent Actions:**
- Read state from memory
- Restore internal structures
- Continue from previous state

**Example:**
```go
//export agent_resume
func agent_resume(ptr, size uint32) {
    if size == 0 {
        return
    }
    buf := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
    state.Counter = binary.LittleEndian.Uint64(buf)
    fmt.Printf("[agent] Resumed with counter=%d\n", state.Counter)
}
```

### Phase 6: Termination

**Triggers:**
- Budget exhausted
- User interrupt (Ctrl+C)
- Migration (origin node)
- Fatal error

**Runtime Actions:**
- Final checkpoint
- Save to storage
- Close WASM module
- Log termination reason

**Agent Actions:**
- None (runtime controls termination)

## Lifecycle State Machine

```
     ┌──────────┐
     │  LOADED  │
     └────┬─────┘
          │ agent_init()
          ▼
   ┌─────────────┐
   │ INITIALIZED │
   └──────┬──────┘
          │ start tick loop
          ▼
   ┌─────────────┐◄─────────┐
   │   RUNNING   │          │
   └──────┬──────┘          │
          │                 │
          ├─ agent_tick()  ─┤
          │                 │
          ├─ checkpoint() ──┤
          │                 │
          ├─ budget check ──┘
          │
          ▼
   ┌─────────────┐
   │ TERMINATED  │
   └─────────────┘
```

## Checkpoint Format

### Structure

```
Offset  Size  Field
0       8     Budget (float64, little-endian)
8       8     PricePerSecond (float64, little-endian)
16      N     Agent State (agent-defined)
```

### Example (Counter Agent)

```
Total: 24 bytes
[0-7]   1.000000 (budget)
[8-15]  0.001000 (price per second)
[16-23] 42 (counter value as uint64)
```

### Portability

Checkpoints are:
- Platform-independent (little-endian encoding)
- Self-contained (include budget metadata)
- Migration-ready (transferable between nodes)
- Atomic (all-or-nothing writes)

## Execution Constraints

### Time Limits

- **Tick timeout**: 100ms per execution
- **Context cancellation**: Respected by runtime
- **No blocking**: Ticks must be short and resumable

### Memory Limits

- **Per-agent limit**: 64MB (1024 pages × 64KB)
- **WASM linear memory**: Limited by runtime config
- **No memory sharing**: Agents are isolated

### I/O Restrictions

- **No filesystem**: Read/write disabled in WASM
- **No network**: Socket access disabled
- **Stdout/stderr**: Allowed (logged by runtime)

## State Management

### Explicit State

Agents must explicitly serialize all state in `checkpoint()`.

**Bad (won't survive):**
```go
// Static variable not checkpointed
var cache = make(map[string]string)
```

**Good (survives):**
```go
type State struct {
    Counter uint64
    Cache   map[string]string
}
var state State

func checkpoint() {
    // Serialize entire state including cache
}
```

### Determinism

Agents should be deterministic given the same state:
- No random without seeded RNG
- No time.Now() unless checkpointed
- No external dependencies

### State Size

Keep state minimal:
- Checkpoint on every migration
- Transferred over network
- Stored by nodes
- Impacts performance

## Budget Management

### Initial Budget

Set via CLI flag:
```bash
igord --run-agent agent.wasm --budget 10.0
```

### Metering

Every tick:
```go
start := time.Now()
agent_tick()
elapsed := time.Since(start)

cost := elapsed.Seconds() × pricePerSecond
budget -= cost
```

### Exhaustion

When budget ≤ 0:
1. Stop calling `agent_tick()`
2. Call `agent_checkpoint()`
3. Save checkpoint
4. Terminate instance
5. Log reason: `budget_exhausted`

### Restoration

Budget persists through:
- **Local restarts**: Loaded from checkpoint
- **Migration**: Transferred in AgentPackage

## Error Handling

### Tick Errors

If `agent_tick()` returns error:
- Log error
- Terminate agent
- Save final checkpoint (if budget permits)

### Checkpoint Errors

If `agent_checkpoint()` fails:
- Log error
- Continue execution
- Retry on next interval
- Final attempt on shutdown

### Resume Errors

If `agent_resume()` fails:
- Abort agent startup
- Log error
- Keep checkpoint intact

## Example Agent (Counter)

Complete implementation in `agents/example/main.go`:

```go
type State struct {
    Counter uint64
}
var state State

//export agent_init
func agent_init() {
    state.Counter = 0
}

//export agent_tick
func agent_tick() {
    state.Counter++
    fmt.Printf("Tick %d\n", state.Counter)
}

//export agent_checkpoint
func agent_checkpoint() uint32 {
    return 8 // size of counter
}

//export agent_checkpoint_ptr
func agent_checkpoint_ptr() uint32 {
    return uint32(uintptr(unsafe.Pointer(&state.Counter)))
}

//export agent_resume
func agent_resume(ptr, size uint32) {
    buf := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
    state.Counter = binary.LittleEndian.Uint64(buf)
}
```

## Building Agents

### Requirements

- TinyGo compiler
- WASI target support

### Build Command

```bash
cd agents/example
tinygo build -o agent.wasm -target=wasi -no-debug .
```

### Output

- `agent.wasm` - Compiled WASM binary (~190KB for counter example)
- Platform-independent
- Ready to run on any Igor node

## Lifecycle Invariants

1. **agent_init()** called exactly once per instance
2. **agent_tick()** called only when budget > 0
3. **agent_checkpoint()** called before any shutdown
4. **agent_resume()** called at most once per instance
5. **Budget monotonically decreases** (no refunds in runtime)
6. **State persists** through checkpoint/resume cycle

See [INVARIANTS.md](./INVARIANTS.md) for complete list.
