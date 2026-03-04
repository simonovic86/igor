# Hostcall ABI

**Spec Layer:** Runtime Implementation
**Status:** Design Draft
**Derives from:** [CAPABILITY_MEMBRANE.md](../constitution/CAPABILITY_MEMBRANE.md) (CM-1 through CM-7), [CAPABILITY_ENFORCEMENT.md](../enforcement/CAPABILITY_ENFORCEMENT.md) (CE-1 through CE-6)

---

## Purpose

This document defines the concrete hostcall interface between agents and the Igor runtime. It specifies the WASM module namespace, capability namespaces, function signatures, error conventions, and integration approach with the wazero engine.

This is a runtime-layer design document. It may evolve as implementation proceeds. The constitutional guarantees it implements (CM-1 through CM-7) are stable; the specific ABI described here is not frozen.

---

## Module Namespace

All Igor hostcalls are registered under the `igor` WASM module namespace, distinct from `wasi_snapshot_preview1`. Agents import hostcalls as:

```wat
(import "igor" "clock_now" (func $clock_now (result i64)))
(import "igor" "rand_bytes" (func $rand_bytes (param i32 i32) (result i32)))
```

WASI remains available for stdout/stderr (debugging). The `igor` namespace is the structured capability interface; WASI is the minimal compatibility layer.

---

## Capability Namespaces

### clock — Time Observation

Provides wall-clock time. Observation hostcall — recorded in event log for replay (CE-3).

| Function | Signature | Description |
|----------|-----------|-------------|
| `clock_now` | `() -> (i64)` | Returns current time as Unix nanoseconds |

**Replay behavior:** During replay, returns the recorded value from the event log instead of live time.

### rand — Randomness Observation

Provides cryptographically random bytes. Observation hostcall — recorded in event log for replay (CE-3).

| Function | Signature | Description |
|----------|-----------|-------------|
| `rand_bytes` | `(ptr: i32, len: i32) -> (i32)` | Fills buffer at `ptr` with `len` random bytes. Returns 0 on success, negative on error. |

**Replay behavior:** During replay, writes the recorded bytes to the buffer.

### kv — Key-Value Storage

Provides persistent key-value storage scoped to the agent. Read operations are observations; write operations are side effects gated on ACTIVE_OWNER (CE-4).

| Function | Signature | Description |
|----------|-----------|-------------|
| `kv_get` | `(key_ptr: i32, key_len: i32, val_ptr: i32, val_cap: i32) -> (i32)` | Reads value for key into buffer. Returns bytes written, 0 if not found, negative on error. |
| `kv_put` | `(key_ptr: i32, key_len: i32, val_ptr: i32, val_len: i32) -> (i32)` | Writes value for key. Side effect. Returns 0 on success, negative on error. |
| `kv_delete` | `(key_ptr: i32, key_len: i32) -> (i32)` | Deletes key. Side effect. Returns 0 on success, negative on error. |

**Storage scope:** Per-agent. Agents cannot access each other's KV space (RE-7).
**Persistence:** KV state is included in checkpoint data.
**Replay behavior:** Reads return recorded values. Writes are recorded but not re-executed during replay.

### log — Structured Logging

Provides structured log output. Not a side effect — logging is an observation that produces no externally visible state change.

| Function | Signature | Description |
|----------|-----------|-------------|
| `log_emit` | `(ptr: i32, len: i32) -> ()` | Emits a structured log entry. |

**Replay behavior:** Log entries are silently discarded during replay.

### net — Network Requests (Future: Phase 3+)

Provides HTTP-like request/response capability. Side-effect hostcall — gated on ACTIVE_OWNER (CE-4), recorded for replay (CE-3).

| Function | Signature | Description |
|----------|-----------|-------------|
| `net_request` | `(req_ptr: i32, req_len: i32, resp_ptr: i32, resp_cap: i32) -> (i32)` | Sends request, writes response to buffer. Returns bytes written or negative on error. |

**Not implemented in current phase.** Included to demonstrate namespace reservation and design direction.

### wallet — Economic Operations

Provides budget introspection and receipt access. All wallet hostcalls are observations — recorded in event log for replay (CE-3).

| Function | Signature | Description |
|----------|-----------|-------------|
| `wallet_balance` | `() -> (i64)` | Returns current budget in microcents. Observation. |
| `wallet_receipt_count` | `() -> (i32)` | Returns number of receipts accumulated by the agent. Observation. |
| `wallet_receipt` | `(index: i32, buf_ptr: i32, buf_len: i32) -> (i32)` | Copies receipt at `index` into buffer. Returns bytes written, -1 on invalid index, -4 if buffer too small. Observation. |

**Replay behavior:** During replay, returns the recorded values from the event log.

**Receipts:** Created by the runtime after each checkpoint epoch. Each receipt is signed by the node's Ed25519 peer key and attests to the execution cost for a range of ticks. Receipts travel with the agent during migration for audit trail continuity.

---

## Error Convention

All hostcalls returning `i32` follow this convention:

| Value | Meaning |
|-------|---------|
| `>= 0` | Success. For read operations, the number of bytes written. |
| `-1` | Generic error |
| `-2` | Capability not granted |
| `-3` | Authority state violation (not ACTIVE_OWNER) |
| `-4` | Buffer too small |
| `-5` | Key not found (kv_get) |
| `-6` | Budget insufficient |

Agents MUST check return values. The runtime does not abort on hostcall errors — it returns error codes and lets the agent decide how to respond.

---

## Capability Manifest

Agents declare required capabilities in their manifest. The manifest is evaluated at load time (CE-2) and during pre-migration checks (CE-5).

```json
{
  "capabilities": {
    "clock": { "version": 1 },
    "rand": { "version": 1 },
    "kv": { "version": 1, "max_key_size": 256, "max_value_size": 65536 },
    "log": { "version": 1 }
  }
}
```

**Rules:**
- Only declared capabilities are registered in the WASM import namespace (CE-1)
- Missing required imports cause WASM instantiation failure (expected, not an error)
- Manifest format is part of the agent package alongside the WASM binary

---

## Observation Event Log

Per CM-4 and CE-3, the runtime records all observation hostcall return values in a per-tick event log.

**Log entry format (conceptual):**

```
[tick_number: u64][entry_count: u32][entries...]

Entry:
  [hostcall_id: u16][payload_length: u32][payload: bytes]
```

**Properties:**
- Append-only during tick execution
- Sealed at tick boundary (before checkpoint commit)
- Carried alongside checkpoint data through migration
- Sufficient to replay any tick from its starting checkpoint

**Open questions:**
- Event log size budget per tick (may need limits)
- Compression strategy for large observation payloads
- Retention policy (keep all history vs sliding window)

---

## wazero Integration

Hostcalls are registered using wazero's `RuntimeConfig` and host module builder:

```go
// Conceptual — not final API
igor := r.NewHostModuleBuilder("igor")

if manifest.Has("clock") {
    igor.NewFunctionBuilder().
        WithFunc(func(ctx context.Context) int64 {
            now := time.Now().UnixNano()
            eventLog.Record(ClockNow, now)
            return now
        }).
        Export("clock_now")
}

if manifest.Has("rand") {
    igor.NewFunctionBuilder().
        WithFunc(func(ctx context.Context, ptr, length uint32) int32 {
            buf := make([]byte, length)
            rand.Read(buf)
            eventLog.Record(RandBytes, buf)
            mem.Write(ptr, buf)
            return 0
        }).
        Export("rand_bytes")
}

igor.Instantiate(ctx)
```

**Key integration points:**
- `internal/runtime/engine.go` — currently calls `wasi.MustInstantiate`; hostcall module registered alongside WASI
- `internal/agent/instance.go` — manifest loaded from agent package, passed to engine for selective registration

---

## Design Decisions

**JSON capability manifest over Protobuf:** The manifest is a small, static declaration read once at load time. JSON is human-readable, requires no code generation, and is trivially debuggable. Protobuf would add a schema file, a compilation step, and a generated code dependency for a structure that fits in a few lines. If manifest complexity grows significantly, this decision can be revisited without breaking the constitutional layer (manifests are a runtime concern).

**`igor` namespace over WASI extensions:** Hostcalls live in a custom `igor` WASM module rather than extending WASI. This keeps the Igor capability model independent of the WASI specification evolution, avoids conflicting with WASI-P2 standardization, and makes capability auditing unambiguous — anything in `igor.*` is Igor-mediated, anything in `wasi_*` is the minimal compatibility layer.

**Negative error codes over exceptions/traps:** Hostcalls return negative integers for errors rather than trapping the WASM instance. This keeps error handling in agent code (the agent decides how to respond to a failed hostcall) and preserves the agent's ability to checkpoint cleanly after errors. A trap would abort the tick and potentially lose in-progress state.

## Relationship to Current Implementation

The current runtime (`internal/runtime/engine.go`) provides WASI with filesystem/network disabled. Agents interact via stdout/stderr only. The hostcall ABI replaces this limited interface with structured, auditable, replayable I/O channels.

**Migration path:**
1. Add `igor` host module alongside existing WASI
2. Implement clock and rand hostcalls first (simplest, most useful)
3. Add kv hostcalls (enables persistent agent state beyond checkpoint blob)
4. Deprecate direct stdout for agent output; prefer `log_emit`
5. Add net/wallet in later phases

The existing 4-function agent contract (`agent_init`, `agent_tick`, `agent_checkpoint`, `agent_resume`) remains unchanged. Hostcalls add new imports, not new exports.

---

## References

- [CAPABILITY_MEMBRANE.md](../constitution/CAPABILITY_MEMBRANE.md) — Constitutional invariants (CM-1 through CM-7)
- [CAPABILITY_ENFORCEMENT.md](../enforcement/CAPABILITY_ENFORCEMENT.md) — Enforcement rules (CE-1 through CE-6)
- [ARCHITECTURE.md](./ARCHITECTURE.md) — System architecture
- [AGENT_LIFECYCLE.md](./AGENT_LIFECYCLE.md) — Agent development guide
