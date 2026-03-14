# Agent Lifecycle

## Overview

Igor agents implement a deterministic lifecycle with five required functions:

1. `agent_init()` - One-time initialization
2. `agent_tick()` - Periodic execution (returns `uint32`: nonzero = has more work)
3. `agent_checkpoint()` - Returns state size
4. `agent_checkpoint_ptr()` - Returns pointer to state
5. `agent_resume(ptr, len)` - State restoration

All agents must export these functions for the runtime to call. TinyGo agents using the SDK (`sdk/igor`) get all five exports automatically.

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
- Enforce 15s timeout
- Measure execution duration
- Calculate cost
- Deduct from budget
- Log metrics
- Checkpoint every 5 seconds

**Agent Actions:**
- Execute one unit of work
- Update internal state
- Must complete within tick timeout (15s)

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
costMicrocents := elapsed.Nanoseconds() * pricePerSecond / 1_000_000_000
budget -= costMicrocents
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
- Add checkpoint header (209 bytes v0x04): version, budget, price, tick, WASM hash, lease metadata, lineage hash, signature
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
0       1     Version (0x04)
1       8     Budget (int64 microcents, little-endian)
9       8     PricePerSecond (int64 microcents, little-endian)
17      8     TickNumber (uint64, little-endian)
25      32    WASMHash (SHA-256 of agent binary)
57      8     MajorVersion (uint64, little-endian)
65      8     LeaseGeneration (uint64, little-endian)
73      8     LeaseExpiry (uint64, little-endian)
81      32    PrevHash (SHA-256 of previous checkpoint)
113     32    AgentPubKey (Ed25519 public key)
145     64    Signature (Ed25519 over header minus signature)
209     N     Agent State (agent-defined)
```

Header: 209 bytes (v0x04). Supports reading v0x02 (57 bytes) and v0x03 (81 bytes). Budget unit: 1 currency unit = 1,000,000 microcents.

### Example (Counter Agent)

```
Total: 217 bytes (209-byte header + 8-byte state)
[0]       0x04 (version)
[1-8]     1000000 (budget = 1.0 units in microcents)
[9-16]    1000 (price = 0.001 units/sec in microcents)
[17-24]   42 (tick number)
[25-56]   <SHA-256 hash of agent WASM binary>
[57-64]   1 (major version)
[65-72]   1 (lease generation)
[73-80]   <lease expiry timestamp>
[81-112]  <SHA-256 of previous checkpoint>
[113-144] <Ed25519 public key>
[145-208] <Ed25519 signature>
[209-216] 42 (counter value as uint64)
```

### Portability

Checkpoints are:
- Platform-independent (little-endian encoding)
- Self-contained (include budget metadata)
- Migration-ready (transferable between nodes)
- Atomic (all-or-nothing writes)

## Execution Constraints

### Time Limits

- **Tick timeout**: 15s per execution
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

costMicrocents := elapsed.Nanoseconds() * pricePerSecond / 1_000_000_000
budget -= costMicrocents
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

## Example Agent (Survivor)

Complete implementation in `agents/example/main.go` using the Igor SDK (`sdk/igor`):

```go
type Survivor struct {
    TickCount uint64
    BirthNano int64
    LastNano  int64
    Luck      uint32
}

func (s *Survivor) Init() {}

func (s *Survivor) Tick() bool {
    s.TickCount++
    now := igor.ClockNow()
    if s.BirthNano == 0 { s.BirthNano = now }
    s.LastNano = now
    buf := make([]byte, 4)
    igor.RandBytes(buf)
    s.Luck ^= binary.LittleEndian.Uint32(buf)
    igor.Logf("[survivor] tick %d | age %ds | luck 0x%08x",
        s.TickCount, (s.LastNano-s.BirthNano)/1e9, s.Luck)
    return false
}

func (s *Survivor) Marshal() []byte   { /* 28-byte LE encoding */ }
func (s *Survivor) Unmarshal(data []byte) { /* reverse */ }

func init() { igor.Run(&Survivor{}) }
func main() {}
```

The SDK provides all five required WASM exports (`agent_init`, `agent_tick`, `agent_checkpoint`, `agent_checkpoint_ptr`, `agent_resume`) automatically, delegating to the `Agent` interface methods.

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

See [RUNTIME_ENFORCEMENT_INVARIANTS.md](../enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md) for enforcement invariants and [EXECUTION_INVARIANTS.md](../constitution/EXECUTION_INVARIANTS.md) for constitutional invariants.
