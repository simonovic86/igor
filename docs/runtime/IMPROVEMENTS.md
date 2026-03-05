# Runtime Improvements

Concrete improvements to the Igor runtime, organized by area. Each item describes the current behavior, the problem, and a proposed change.

---

## Replay Engine

### 1. Cache Compiled WASM in Replay Engine

**Current:** `ReplayTick` creates a fresh wazero runtime, compiles the WASM binary from bytes, instantiates it, runs one tick, and tears everything down — every single time.

**Problem:** WASM compilation is the dominant cost (~300ms for a 190KB module). Periodic verification recompiles the same bytes repeatedly.

**Proposed:** Cache the `wazero.CompiledModule` in the replay `Engine`, keyed by WASM hash. Recompile only when the hash changes. Instantiation + tick execution is ~1ms — this would make "full" replay mode cheap enough to always leave on.

```go
type Engine struct {
    logger      *slog.Logger
    cachedHash  [32]byte
    cachedRT    wazero.Runtime
    cachedMod   wazero.CompiledModule
}
```

**Impact:** ~300x speedup for periodic replay verification.

### 2. Hash-Based Post-State Comparison

**Current:** Every tick stores full copies of pre-tick and post-tick state in the `TickSnapshot`. During replay, the full replayed state is compared byte-by-byte against the stored post-state.

**Problem:** For agents with large state (megabytes), the replay window stores `2 × state_size × window_size` bytes. A 1MB state with a 16-slot window is 32MB of snapshot data.

**Proposed:** Store only the SHA-256 hash of the post-tick state. The full pre-tick state is still needed as replay input, but the post-tick state only needs a 32-byte hash for comparison.

```go
type TickSnapshot struct {
    TickNumber    uint64
    PreState      []byte      // full bytes — needed as replay input
    PostStateHash [32]byte    // hash only — compared against replay output
    TickLog       *eventlog.TickLog
}
```

**Impact:** Halves snapshot memory usage. State comparison becomes constant-time.

### 3. Observation-Weighted Snapshot Retention ✅

**Status:** Implemented in `internal/agent/instance.go` (weighted eviction) and `internal/runner/runner.go` (skip-empty verification).

**Current:** The replay window retains the last N snapshots and drops older ones regardless of content.

**Problem:** A tick with zero hostcalls is trivially deterministic — replaying it proves nothing. A tick with many observation hostcalls is the one worth verifying.

**Implemented:** Snapshots scored by observation count. On eviction, lowest-score snapshot is dropped (ties favor oldest). Verification skips ticks with empty event logs.

```go
func (snap TickSnapshot) observationScore() int {
    if snap.TickLog == nil {
        return 0
    }
    return len(snap.TickLog.Entries)
}
```

**Impact:** Better verification coverage without increasing window size.

### 4. Multi-Tick Chain Verification ✅

**Status:** Implemented in `internal/replay/engine.go` as `ReplayChain()`.

**Current:** Replay verifies a single tick at a time: `state_N + events → verify state_N+1`.

**Problem:** Single-tick replay catches per-tick nondeterminism but misses drift that accumulates across ticks (e.g., memory corruption, subtle state divergence that only manifests after many transitions).

**Implemented:** Chain verification replays N consecutive ticks in a single WASM instance using an `iteratorHolder` that swaps entry iterators between ticks. Compares only the final state hash.

```go
func (e *Engine) ReplayChain(
    ctx context.Context,
    wasmBytes []byte,
    capManifest *manifest.CapabilityManifest,
    initialState []byte,
    snapshots []ChainSnapshot,
    expectedFinalHash [32]byte,
) *ChainResult
```

**Impact:** Stronger integrity guarantees. Catches accumulated drift that single-tick replay misses.

### 5. Replay Failure Escalation Policy ✅

**Status:** Implemented in `internal/runner/runner.go` and `internal/config/config.go`.

**Current:** When replay detects divergence, the system logs a warning and continues execution.

**Problem:** No escalation path. A divergence could indicate a runtime bug, a nondeterministic agent, or a corrupted checkpoint — all of which warrant different responses.

**Implemented:** Four configurable escalation policies via `--replay-on-divergence` flag:

| Policy | Behavior |
|--------|----------|
| `log` | Log warning, continue (current default) |
| `pause` | Stop ticking, preserve checkpoint, exit cleanly |
| `intensify` | Reduce VerifyInterval to 1 (verify every tick) |
| `migrate` | Trigger migration via `MigrationTrigger` callback, fallback to pause |

Config validated at startup. `HandleDivergenceAction()` executes the policy. Full test coverage.

**Impact:** Operators can choose the appropriate safety/liveness tradeoff.

---

## Tick Loop

### 6. Adaptive Tick Rate ✅

**Status:** Implemented in `cmd/igord/main.go` (tick loop) and `internal/agent/instance.go` (return value handling).

**Current:** The tick loop runs at a fixed 1 Hz. An agent that finishes its tick in 0.1ms waits 999.9ms idle.

**Problem:** Agents with bursty workloads (e.g., processing a batch of events) are bottlenecked at 1 tick/second regardless of how fast they complete.

**Implemented:** `agent_tick()` returns uint32 hint: 0 = no more work (1s interval), 1 = more work pending (10ms interval). Backward compatible — legacy void-returning agents default to "no more work". SDK `Agent.Tick() bool` maps cleanly.

```go
results, _ := tickFn.Call(ctx)
hasMoreWork := results[0] != 0
if hasMoreWork {
    nextTick = minTickInterval  // 10ms floor
} else {
    nextTick = normalInterval   // 1s
}
```

**Impact:** Up to 100x throughput for bursty workloads without changing the metering model. Budget still drains per wall-clock time consumed.

---

## Agent Developer Experience

### 7. SDK Checkpoint Serialization ✅

**Status:** Implemented in `sdk/igor/` — lifecycle exports, `Encoder`/`Decoder` helpers, `Raw`/`FixedBytes`/`ReadInto` for fixed-size fields.

**Current:** Agents manually serialize state using `unsafe.Pointer`, `binary.LittleEndian`, and raw byte buffers.

**Problem:** Error-prone, tedious, and hostile to new contributors. Every agent reinvents serialization.

**Implemented:** The SDK handles `agent_checkpoint`, `agent_checkpoint_ptr`, `agent_resume`, and `malloc` exports automatically. Agents implement the `Agent` interface with chainable `Encoder`/`Decoder` helpers:

```go
type Agent interface {
    Init()
    Tick() bool
    Marshal() []byte       // igor.NewEncoder(n).Uint64(v).Raw(id[:]).Finish()
    Unmarshal(data []byte)  // d := igor.NewDecoder(data); d.ReadInto(id[:])
}
func init() { igor.Run(&MyAgent{}) }
```

Encoding methods: `Uint64`, `Int64`, `Uint32`, `Int32`, `Bool`, `Bytes` (length-prefixed), `String`, `Raw` (no prefix). Decoding methods: matching reads plus `FixedBytes(n)` and `ReadInto(dst)` for fixed-size arrays.

**Impact:** Agent authoring reduced from manual binary manipulation to ~15-line idiomatic Go. Lower barrier to entry.

---

## Resource Efficiency

### 8. Shared Runtime Engine

**Current:** `main.go` creates two separate `runtime.Engine` instances — one for the migration service (line 86) and one inside `runLocalAgent` (line 158). Each contains its own wazero runtime and module cache.

**Problem:** Double memory footprint for compiled module caches. Two independent wazero runtimes doing the same work.

**Proposed:** Pass the same `*runtime.Engine` into both `runLocalAgent` and the migration service. wazero runtimes are safe for concurrent use.

**Impact:** Halves baseline memory for compiled WASM caches. Simple refactor.

### 9. Event Log Allocation Optimization ✅

**Status:** Implemented in `internal/eventlog/eventlog.go`.

**Current:** Every `Record()` call allocates a new byte slice and copies the payload.

**Problem:** For high-frequency hostcalls (e.g., `clock_now` called every tick), this creates many small allocations that pressure the GC.

**Implemented:** 4KB arena pre-allocated per tick at `BeginTick`. `Record()` sub-allocates from the arena; falls back to heap when the arena is full. Eviction copies retained slice to release references.

```go
type TickLog struct {
    TickNumber uint64
    Entries    []Entry
    arena      []byte  // pre-allocated backing store (4KB)
    offset     int     // next free position
}
```

**Impact:** Reduces GC pressure. Marginal for current single-agent workloads, meaningful for multi-agent nodes.

---

## Budget Precision

### 10. Sub-Microsecond Tick Metering

**Current:**

```go
costMicrocents := elapsed.Microseconds() * pricePerSecond / budget.MicrocentScale
```

**Problem:** `elapsed.Microseconds()` truncates to whole microseconds. For very short ticks (sub-microsecond on fast hardware), cost rounds to zero — the agent runs for free.

**Proposed:** Use nanosecond precision with adjusted arithmetic:

```go
costMicrocents := elapsed.Nanoseconds() * pricePerSecond / (budget.MicrocentScale * 1000)
```

Or introduce a higher-precision intermediate calculation to avoid integer overflow for long ticks.

**Impact:** Correct metering for fast agents. No behavioral change for typical tick durations (~1ms).

---

## Priority Order

Ranked by impact-to-effort ratio:

1. ~~**Shared Runtime Engine** (#8)~~ — **DONE**. Single engine shared between migration service and local agent.
2. ~~**Cached WASM in Replay** (#1)~~ — **DONE**. `wazero.CompilationCache` shared across replay invocations.
3. ~~**Hash-Based Post-State** (#2)~~ — **DONE**. `TickSnapshot.PostStateHash [32]byte` replaces full state copy.
4. ~~**Sub-Microsecond Metering** (#10)~~ — **DONE**. Nanosecond precision via `elapsed.Nanoseconds()`.
5. ~~**Replay Failure Escalation** (#5)~~ — **DONE**. Configurable `--replay-on-divergence` policy: log, pause, intensify, migrate.
6. ~~**Observation-Weighted Retention** (#3)~~ — **DONE**. `observationScore()` weighted eviction; empty tick logs skipped during verification.
7. ~~**SDK Checkpoint Serialization** (#7)~~ — **DONE**. `Encoder`/`Decoder` helpers with `Raw`/`FixedBytes`/`ReadInto` for fixed-size fields.
8. ~~**Adaptive Tick Rate** (#6)~~ — **DONE**. `agent_tick() → uint32` return hint; tick loop uses 10ms fast path / 1s normal interval.
9. ~~**Multi-Tick Chain Verification** (#4)~~ — **DONE**. `ReplayChain()` replays N consecutive ticks with hash-based final state comparison.
10. ~~**Event Log Allocation** (#9)~~ — **DONE**. 4KB arena per tick with heap fallback; eviction copies to release references.
