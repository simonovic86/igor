# Hostcall ABI

**Spec Layer:** Runtime Implementation
**Status:** Implemented
**Derives from:** [CAPABILITY_MEMBRANE.md](../constitution/CAPABILITY_MEMBRANE.md) (CM-1 through CM-7), [CAPABILITY_ENFORCEMENT.md](../enforcement/CAPABILITY_ENFORCEMENT.md) (CE-1 through CE-6)

---

## Purpose

This document defines the concrete hostcall interface between agents and the Igor runtime. It specifies the WASM module namespace, capability namespaces, function signatures, error conventions, and integration with the wazero engine.

The constitutional guarantees it implements (CM-1 through CM-7) are stable; the specific ABI described here may evolve as new capabilities are added.

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

**Implementation:** `internal/hostcall/clock.go` — calls `time.Now().UnixNano()`, records 8-byte little-endian payload in event log.

**Replay behavior:** During replay, returns the recorded value from the event log instead of live time.

### rand — Randomness Observation

Provides cryptographically random bytes. Observation hostcall — recorded in event log for replay (CE-3).

| Function | Signature | Description |
|----------|-----------|-------------|
| `rand_bytes` | `(ptr: i32, len: i32) -> (i32)` | Fills buffer at `ptr` with `len` random bytes. Returns 0 on success, negative on error. |

**Implementation:** `internal/hostcall/rand.go` — calls `crypto/rand.Read()`, writes bytes to WASM memory, records raw bytes in event log.

**Replay behavior:** During replay, writes the recorded bytes to the buffer.

### log — Structured Logging

Provides structured log output. Observation hostcall — recorded in event log for replay (CE-3). Not a side effect — logging produces no externally visible state change.

| Function | Signature | Description |
|----------|-----------|-------------|
| `log_emit` | `(ptr: i32, len: i32) -> ()` | Emits a structured log entry. |

**Implementation:** `internal/hostcall/log.go` — reads message from WASM memory (capped at 4096 bytes), logs via `slog`, records message bytes in event log.

**Replay behavior:** Log entries are silently consumed during replay (entry iterator advances but no output is produced).

### wallet — Economic Operations

Provides budget introspection and receipt access. All wallet hostcalls are observations — recorded in event log for replay (CE-3).

| Function | Signature | Description |
|----------|-----------|-------------|
| `wallet_balance` | `() -> (i64)` | Returns current budget in microcents. Observation. |
| `wallet_receipt_count` | `() -> (i32)` | Returns number of receipts accumulated by the agent. Observation. |
| `wallet_receipt` | `(index: i32, buf_ptr: i32, buf_len: i32) -> (i32)` | Copies receipt at `index` into buffer. Returns bytes written, -1 on invalid index, -4 if buffer too small. Observation. |

**Implementation:** `internal/hostcall/wallet.go` — reads from the `WalletState` interface (budget, receipt count, receipt bytes), records values in event log.

**Replay behavior:** During replay, returns the recorded values from the event log.

**Receipts:** Created by the runtime after each checkpoint epoch. Each receipt is signed by the node's Ed25519 peer key and attests to the execution cost for a range of ticks. Receipts travel with the agent during migration for audit trail continuity.

### kv — Key-Value Storage (Not Yet Implemented)

Persistent key-value storage scoped to the agent. Reads are observations; writes are side effects gated on ACTIVE_OWNER (CE-4). Reserved for a future task.

| Function | Signature | Description |
|----------|-----------|-------------|
| `kv_get` | `(key_ptr: i32, key_len: i32, val_ptr: i32, val_cap: i32) -> (i32)` | Reads value for key into buffer. Returns bytes written, 0 if not found, negative on error. |
| `kv_put` | `(key_ptr: i32, key_len: i32, val_ptr: i32, val_len: i32) -> (i32)` | Writes value for key. Side effect. Returns 0 on success, negative on error. |
| `kv_delete` | `(key_ptr: i32, key_len: i32) -> (i32)` | Deletes key. Side effect. Returns 0 on success, negative on error. |

### x402 — Payment Operations

Payment capability for agents to pay for services from their budget. Side-effect hostcall. Recorded in event log for replay (CE-3). Declared under the `x402` manifest capability key.

| Function | Signature | Description |
|----------|-----------|-------------|
| `wallet_pay` | `(amount: i64, recipient_ptr: i32, recipient_len: i32, memo_ptr: i32, memo_len: i32, receipt_ptr: i32, receipt_cap: i32) -> (i32)` | Deducts `amount` microcents from budget, creates a signed payment receipt, writes receipt to buffer. Returns bytes written on success, negative error code on failure. |

**Implementation:** `internal/hostcall/payment.go` — validates recipient against `allowed_recipients`, validates amount against `max_payment_microcents`, deducts from budget, creates Ed25519-signed receipt (amount, timestamp, recipient, memo, agent pubkey), records amount and receipt in event log.

**Replay behavior:** During replay, returns the recorded amount and receipt from the event log.

**Manifest configuration:**
```json
{
  "x402": {
    "version": 1,
    "options": {
      "allowed_recipients": ["service-provider"],
      "max_payment_microcents": 1000000
    }
  }
}
```

**Options:**
- `allowed_recipients` — list of permitted payment recipients (empty = all recipients allowed)
- `max_payment_microcents` — maximum amount per payment (0 = no limit)

**Error codes:**
| Value | Meaning |
|-------|---------|
| `-1` | Insufficient budget |
| `-2` | Input too long (recipient > 256 bytes, memo > 1KB) |
| `-3` | Recipient not in allowed_recipients |
| `-4` | Amount exceeds max_payment_microcents |
| `-5` | Receipt buffer too small (first 4 bytes contain required size as LE uint32) |
| `-6` | Processing error |

### http — HTTP Requests

HTTP request/response capability for calling external APIs. Side-effect hostcall. Recorded in event log for replay (CE-3).

| Function | Signature | Description |
|----------|-----------|-------------|
| `http_request` | `(method_ptr: i32, method_len: i32, url_ptr: i32, url_len: i32, headers_ptr: i32, headers_len: i32, body_ptr: i32, body_len: i32, resp_ptr: i32, resp_cap: i32) -> (i32)` | Sends HTTP request, writes response to buffer. Returns HTTP status code (>0) on success, negative error code on failure. |

**Implementation:** `internal/hostcall/http.go` — reads method, URL, headers, and body from WASM memory, executes the request via `http.Client`, writes response as `[body_len: 4 bytes LE][body: N bytes]` to the response buffer, records status code and body in event log.

**Replay behavior:** During replay, returns the recorded status code and response body from the event log.

**Manifest configuration:**
```json
{
  "http": {
    "version": 1,
    "options": {
      "allowed_hosts": ["api.coingecko.com"],
      "timeout_ms": 10000,
      "max_response_bytes": 1048576
    }
  }
}
```

**Options:**
- `allowed_hosts` — list of permitted hostnames (empty = all hosts allowed)
- `timeout_ms` — per-request timeout in milliseconds (default: 10000)
- `max_response_bytes` — maximum response body size (default: 1MB)

**Error codes:**
| Value | Meaning |
|-------|---------|
| `-1` | Network error |
| `-2` | Input too long (URL > 8KB, headers > 32KB, body > 1MB) |
| `-3` | Host not in allowed_hosts |
| `-4` | Request timeout |
| `-5` | Response body exceeds max_response_bytes |

**Size hint:** When response buffer is too small (`-5`), the first 4 bytes of the response buffer contain the required body length as LE uint32, allowing the agent to retry with a larger allocation.

---

## Error Convention

All hostcalls returning `i32` follow this convention:

| Value | Meaning | Status |
|-------|---------|--------|
| `>= 0` | Success. For read operations, the number of bytes written. | Active |
| `-1` | Generic error | Active |
| `-4` | Buffer too small | Active |
| `-2` | Capability not granted | Reserved (undeclared capabilities cause WASM instantiation failure instead) |
| `-3` | Authority state violation (not ACTIVE_OWNER) | Reserved (authority state machine not yet implemented) |
| `-5` | Key not found (kv_get) | Reserved (kv not yet implemented) |
| `-6` | Budget insufficient | Reserved |

**Note:** Error codes `-2` and `-3` are defined for future use. Currently, capabilities not declared in the manifest are simply not registered in the `igor` host module, causing WASM instantiation to fail (the import cannot be resolved). When the authority state machine is implemented (Phase 5), `-3` will be returned for side-effect hostcalls invoked outside ACTIVE_OWNER state.

Agents MUST check return values. The runtime does not abort on hostcall errors — it returns error codes and lets the agent decide how to respond.

---

## Capability Manifest

Agents declare required capabilities in their manifest. The manifest is evaluated at load time (CE-2) and during pre-migration checks (CE-5).

```json
{
  "capabilities": {
    "clock": { "version": 1 },
    "rand": { "version": 1 },
    "log": { "version": 1 },
    "wallet": { "version": 1 },
    "http": { "version": 1, "options": { "allowed_hosts": ["api.example.com"] } },
    "x402": { "version": 1, "options": { "allowed_recipients": ["service"], "max_payment_microcents": 1000000 } }
  },
  "resource_limits": {
    "max_memory_bytes": 33554432
  },
  "migration_policy": {
    "enabled": true,
    "max_price_per_second": 5000
  }
}
```

**Rules:**
- Only declared capabilities are registered in the WASM import namespace (CE-1)
- Missing required imports cause WASM instantiation failure (expected, not an error)
- Manifest format is part of the agent package alongside the WASM binary
- `resource_limits.max_memory_bytes` is validated against the node's 64MB limit at load time
- `migration_policy` controls whether the agent accepts migration and at what price ceiling

---

## Observation Event Log

Per CM-4 and CE-3, the runtime records all observation hostcall return values in a per-tick event log.

**Implementation:** `internal/eventlog/eventlog.go`

**Entry format:**

```go
type Entry struct {
    HostcallID HostcallID // ClockNow=1, RandBytes=2, LogEmit=3, WalletBalance=4, ..., HTTPRequest=8, WalletPay=9
    Payload    []byte     // Return value bytes (e.g., 8-byte timestamp, N random bytes)
}
```

**Properties:**
- Append-only during tick execution
- Sealed at tick boundary (before checkpoint commit)
- Carried alongside checkpoint data through migration (as `ReplayData` in `AgentPackage`)
- Sufficient to replay any tick from its starting checkpoint
- Per-tick arena allocation (4KB default) to minimize GC pressure
- Sliding window retention (configurable via `--replay-window`, default 16 snapshots)
- Observation-weighted eviction: low-observation ticks are evicted first

---

## wazero Integration

Hostcalls are registered using wazero's host module builder. The `internal/hostcall/registry.go` Registry selectively registers capabilities based on the agent's manifest:

```go
// From internal/hostcall/registry.go
registry := hostcall.NewRegistry(logger, eventLog)
registry.SetWalletState(walletState)
registry.SetWalletPayState(walletPayState) // for x402 payments
if err := registry.RegisterHostModule(ctx, rt, capManifest); err != nil {
    return err
}
```

Each capability registers its functions under the `igor` module namespace:

```go
// From internal/hostcall/clock.go
builder.NewFunctionBuilder().
    WithFunc(func(ctx context.Context) int64 {
        now := time.Now().UnixNano()
        el.Record(eventlog.ClockNow, now)
        return now
    }).
    Export("clock_now")
```

**Key integration points:**
- `internal/runtime/engine.go` — creates the wazero runtime with 64MB memory limit, WASI instantiated alongside the `igor` host module
- `internal/agent/instance.go` — manifest loaded from agent package, passed to Registry for selective registration
- `internal/hostcall/registry.go` — deny-by-default: only capabilities declared in the manifest are registered
- `internal/replay/engine.go` — replay-mode hostcalls return recorded values from the event log iterator

---

## Design Decisions

**JSON capability manifest over Protobuf:** The manifest is a small, static declaration read once at load time. JSON is human-readable, requires no code generation, and is trivially debuggable. Protobuf would add a schema file, a compilation step, and a generated code dependency for a structure that fits in a few lines. If manifest complexity grows significantly, this decision can be revisited without breaking the constitutional layer (manifests are a runtime concern).

**`igor` namespace over WASI extensions:** Hostcalls live in a custom `igor` WASM module rather than extending WASI. This keeps the Igor capability model independent of the WASI specification evolution, avoids conflicting with WASI-P2 standardization, and makes capability auditing unambiguous — anything in `igor.*` is Igor-mediated, anything in `wasi_*` is the minimal compatibility layer.

**Negative error codes over exceptions/traps:** Hostcalls return negative integers for errors rather than trapping the WASM instance. This keeps error handling in agent code (the agent decides how to respond to a failed hostcall) and preserves the agent's ability to checkpoint cleanly after errors. A trap would abort the tick and potentially lose in-progress state.

---

## References

- [CAPABILITY_MEMBRANE.md](../constitution/CAPABILITY_MEMBRANE.md) — Constitutional invariants (CM-1 through CM-7)
- [CAPABILITY_ENFORCEMENT.md](../enforcement/CAPABILITY_ENFORCEMENT.md) — Enforcement rules (CE-1 through CE-6)
- [ARCHITECTURE.md](./ARCHITECTURE.md) — System architecture
- [AGENT_LIFECYCLE.md](./AGENT_LIFECYCLE.md) — Agent development guide
- [IMPROVEMENTS.md](./IMPROVEMENTS.md) — Runtime optimization decisions
