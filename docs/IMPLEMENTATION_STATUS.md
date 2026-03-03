# Implementation Status

Truth matrix mapping spec documents to current code. Source of truth is the code; docs have been updated to match.

Last updated: 2026-03-03

## Checkpoint Format

| Aspect | Status | Code Reference |
|--------|--------|----------------|
| Version byte (0x02) | Implemented | `internal/agent/instance.go` `checkpointVersion` |
| Budget as int64 microcents | Implemented | `internal/agent/instance.go` `Instance.Budget` |
| PricePerSecond as int64 microcents | Implemented | `internal/agent/instance.go` `Instance.PricePerSecond` |
| TickNumber in header | Implemented | `internal/agent/instance.go` offset 17-25 |
| WASM SHA-256 hash in header | Implemented | `internal/agent/instance.go` offset 25-57, `Instance.WASMHash` |
| 57-byte header | Implemented | `internal/agent/instance.go` `checkpointHeaderLen` |
| Atomic writes (temp+fsync+rename) | Implemented | `internal/storage/fs_provider.go` |
| Hash verification on resume | Implemented | `internal/agent/instance.go` `LoadCheckpointFromStorage` |
| v1 backward compatibility | Removed | Single format, no version negotiation |

## Budget System

| Aspect | Status | Code Reference |
|--------|--------|----------------|
| int64 microcents (1 unit = 1M) | Implemented | `pkg/budget/budget.go` `MicrocentScale` |
| `FromFloat` conversion at CLI boundary | Implemented | `pkg/budget/budget.go` `FromFloat` |
| `Format` for human-readable display | Implemented | `pkg/budget/budget.go` `Format` |
| Integer cost calculation | Implemented | `internal/agent/instance.go` line 249 |
| Budget monotonically decreasing | Implemented | `internal/agent/instance.go` `Tick` |
| Pre-tick budget check | Implemented | `internal/agent/instance.go` `Tick` |

## Protocol Messages

| Message | Field | Status | Code Reference |
|---------|-------|--------|----------------|
| AgentPackage | AgentID, WASMBinary, Checkpoint | Implemented | `pkg/protocol/messages.go` |
| AgentPackage | WASMHash | Implemented | `pkg/protocol/messages.go` |
| AgentPackage | ManifestData | Implemented | `pkg/protocol/messages.go` |
| AgentPackage | Budget, PricePerSecond (int64) | Implemented | `pkg/protocol/messages.go` |
| AgentPackage | ReplayData | Implemented | `pkg/protocol/messages.go` |
| ReplayData | PreTickState, TickNumber, Entries | Implemented | `pkg/protocol/messages.go` |
| ReplayEntry | HostcallID, Payload | Implemented | `pkg/protocol/messages.go` |
| AgentStarted | StartTime | Implemented | `pkg/protocol/messages.go` |
| AgentTransfer | Package, SourceNodeID | Implemented | `pkg/protocol/messages.go` |

## Replay Verification

| Aspect | Status | Code Reference |
|--------|--------|----------------|
| Event log recording (per-tick) | Implemented | `internal/eventlog/eventlog.go` |
| Replay engine (isolated wazero runtime) | Implemented | `internal/replay/engine.go` |
| Replay hostcalls (clock, rand, log) | Implemented | `internal/replay/engine.go` |
| Periodic self-verification | Implemented | `cmd/igord/main.go` `verifyNextTick` |
| Migration-time replay verification | Implemented | `internal/migration/service.go` `verifyMigrationReplay` |
| Replay window (sliding buffer) | Implemented | `internal/agent/instance.go` `ReplayWindow` |
| `--replay-window` CLI flag | Implemented | `cmd/igord/main.go` |
| `--verify-interval` CLI flag | Implemented | `cmd/igord/main.go` |
| Formal replay modes (off/periodic/on-migrate/full) | Implemented | `internal/config/config.go` `ReplayMode`, `cmd/igord/main.go` `--replay-mode` |
| Replay cost metering | Implemented | `internal/replay/engine.go` `Result.Duration`, `cmd/igord/main.go` `--replay-cost-log` |

## Capability Membrane

| Aspect | Status | Code Reference |
|--------|--------|----------------|
| Capability manifest parsing | Implemented | `pkg/manifest/manifest.go` |
| Deny-by-default (CM-3) | Implemented | `internal/hostcall/registry.go` |
| Host module registration per manifest | Implemented | `internal/hostcall/registry.go` |
| Observation recording (CM-4) | Implemented | `internal/eventlog/eventlog.go` |
| Manifest in migration package | Implemented | `pkg/protocol/messages.go` `ManifestData` |
| Pre-migration capability check (CE-5) | Implemented | `internal/migration/service.go` `handleIncomingMigration` |
| KV storage hostcalls | Not implemented | Roadmap future task |
| Network hostcalls | Not implemented | Roadmap Phase 3+ |

## Identity and Authority

| Aspect | Status | Code Reference |
|--------|--------|----------------|
| Agent logical ID (string) | Implemented | `internal/agent/instance.go` `AgentID` |
| WASM binary hash in checkpoint | Implemented | `internal/agent/instance.go` `WASMHash` |
| WASM hash verification on resume | Implemented | `internal/agent/instance.go` `LoadCheckpointFromStorage` |
| WASM hash in migration package | Implemented | `pkg/protocol/messages.go` `WASMHash` |
| WASM hash verification on migration | Implemented | `internal/migration/service.go` |
| Signed checkpoint lineage | Not implemented | Roadmap Task 13 |
| Lease-based authority epochs | Not implemented | Roadmap Phase 5 |
| Cryptographic agent identity | Not implemented | Roadmap Phase 5 |

## Migration

| Aspect | Status | Code Reference |
|--------|--------|----------------|
| libp2p stream protocol | Implemented | `internal/migration/service.go` |
| Source: checkpoint + package + send | Implemented | `internal/migration/service.go` `MigrateAgent` |
| Target: verify + resume + confirm | Implemented | `internal/migration/service.go` `handleIncomingMigration` |
| Single-instance handoff | Implemented | Source deletes only after confirmation |
| /tmp WASM write on target | Removed | Replaced by `agent.LoadAgentFromBytes` (no temp file) |
| Replay data in migration package | Implemented | `internal/migration/replay.go` |
| Staleness guard for replay data | Implemented | `internal/migration/replay.go` |

## Agent SDK & Developer Experience

| Aspect | Status | Code Reference |
|--------|--------|----------------|
| Agent SDK (Agent interface, hostcall wrappers) | Implemented | `sdk/igor/lifecycle.go`, `sdk/igor/hostcalls_wrappers_wasm.go` |
| Build-tag split (WASM vs native) | Implemented | `sdk/igor/hostcalls_wrappers_wasm.go`, `sdk/igor/hostcalls_wrappers_stub.go` |
| Capability mocks (MockBackend, Runtime) | Implemented | `sdk/igor/mock_backend.go`, `sdk/igor/mock/mock.go` |
| Deterministic mock (fixed clock, seeded rand) | Implemented | `sdk/igor/mock/mock.go` `NewDeterministic` |
| Local simulator (single-process WASM runner) | Implemented | `internal/simulator/simulator.go` |
| Simulator deterministic hostcalls | Implemented | `internal/simulator/hostcalls.go` |
| Simulator replay verification | Implemented | `internal/simulator/simulator.go` `verifyTick` |
| Checkpoint inspector | Implemented | `internal/inspector/inspector.go` |
| WASM hash verification in inspector | Implemented | `internal/inspector/inspector.go` `VerifyWASM` |
| Agent template (Survivor example) | Implemented | `agents/example/` |
| `--simulate` CLI flag | Implemented | `cmd/igord/main.go` |
| `--inspect-checkpoint` CLI flag | Implemented | `cmd/igord/main.go` |
