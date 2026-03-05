# Implementation Status

Truth matrix mapping spec documents to current code. Source of truth is the code; docs have been updated to match.

Last updated: 2026-03-05

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
| AgentStarted | StartTime | Removed | Unused field removed |
| AgentTransfer | Package, SourceNodeID | Implemented | `pkg/protocol/messages.go` |

## Replay Verification

| Aspect | Status | Code Reference |
|--------|--------|----------------|
| Event log recording (per-tick) | Implemented | `internal/eventlog/eventlog.go` |
| Replay engine (isolated wazero runtime) | Implemented | `internal/replay/engine.go` |
| Replay hostcalls (clock, rand, log) | Implemented | `internal/replay/engine.go` |
| Periodic self-verification | Implemented | `internal/runner/runner.go` `VerifyNextTick` |
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
| Side-effect authority gating (CM-5) | Not implemented | Requires authority state machine (Phase 5). Currently no side-effect hostcalls exist (kv/net are unimplemented). When added, side-effect hostcalls MUST only execute in ACTIVE_OWNER state. |
| KV storage hostcalls | Not implemented | Roadmap future task |
| Network hostcalls | Not implemented | Roadmap future phase |

## Identity and Authority

| Aspect | Status | Code Reference |
|--------|--------|----------------|
| Agent logical ID (string) | Implemented | `internal/agent/instance.go` `AgentID` |
| WASM binary hash in checkpoint | Implemented | `internal/agent/instance.go` `WASMHash` |
| WASM hash verification on resume | Implemented | `internal/agent/instance.go` `LoadCheckpointFromStorage` |
| WASM hash in migration package | Implemented | `pkg/protocol/messages.go` `WASMHash` |
| WASM hash verification on migration | Implemented | `internal/migration/service.go` |
| OA-2 authority lifecycle states | Not implemented | States (ACTIVE_OWNER, HANDOFF_INITIATED, HANDOFF_PENDING, RETIRED, RECOVERY_REQUIRED) exist as spec concepts only. Migration tracks ownership implicitly via `activeAgents` map presence. Requires Task 12. |
| EI-11 divergent lineage detection | Not implemented | No distributed protocol for detecting concurrent instances across nodes. RECOVERY_REQUIRED state transition is specified but not implemented. Requires Tasks 12-13. |
| Signed checkpoint lineage | Not implemented | Roadmap Task 13 |
| Lease-based authority epochs | Not implemented | Roadmap Task 12 |
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
| Host module re-registration on return | Implemented | `internal/hostcall/registry.go` (close existing before re-instantiate) |
| Orphaned checkpoint cleanup on failure | Implemented | `internal/migration/service.go` `deleteOrphanedCheckpoint` |
| Per-node capability overrides | Implemented | `internal/migration/service.go` `SetNodeCapabilities` |
| Chain migration (A→B→C→A) | Tested | `internal/migration/multinode_test.go` `TestChainMigration_ABC_A` |
| Budget conservation across hops | Tested | `internal/migration/multinode_test.go` `TestChainMigration_BudgetConservation` |
| Capability rejection on migration | Tested | `internal/migration/multinode_test.go` `TestCapabilityRejection_MigrationFails` |
| Capability preservation across hops | Tested | `internal/migration/multinode_test.go` `TestCapabilityPreservation_AcrossHops` |
| Stress migration (20 round-trips) | Tested | `internal/migration/multinode_test.go` `TestStressMigration_RapidRoundTrips` |

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

## Runtime Optimizations

| Optimization | Status | Code Reference |
|-------------|--------|----------------|
| Cached WASM in replay (~300x speedup) | Implemented | `internal/replay/engine.go` `wazero.CompilationCache` |
| Hash-based post-state comparison | Implemented | `internal/agent/instance.go` `TickSnapshot.PostStateHash` |
| Observation-weighted snapshot retention | Implemented | `internal/agent/instance.go` `observationScore` eviction |
| Multi-tick chain verification | Implemented | `internal/replay/engine.go` `ReplayChain` |
| Replay failure escalation policy | Implemented | `internal/runner/runner.go` `EscalationForPolicy`, `--replay-on-divergence` |
| Adaptive tick rate | Implemented | `cmd/igord/main.go` `hasMoreWork` hint, 10ms minimum interval |
| `migrate` escalation policy | Partial | `internal/runner/runner.go` `DivergenceMigrate`. Falls through to pause because peer selection is not yet implemented. Users setting `--replay-on-divergence=migrate` get pause behavior. |
| SDK checkpoint serialization | Implemented | `sdk/igor/encoder.go` `Encoder`/`Decoder` |
| Shared runtime engine (migration) | Implemented | `internal/migration/service.go` shares `runtime.Engine` |
| Event log arena allocation | Implemented | `internal/eventlog/eventlog.go` per-tick arena, 4KB default |
| Sub-microsecond metering | Implemented | `internal/agent/instance.go` nanosecond cost calculation |

## Hardening (Code Review Fixes)

| Fix | Status | Code Reference |
|-----|--------|----------------|
| WASM hash verification: reject malformed hash | Fixed | `internal/migration/service.go` |
| `errors.Is` for sentinel error comparison | Fixed | `internal/agent/instance.go` `LoadCheckpointFromStorage` |
| Safe manifest path derivation (no panic) | Fixed | `cmd/igord/main.go`, `internal/migration/service.go` |
| Replay engine resource cleanup | Fixed | `cmd/igord/main.go` `defer replayEngine.Close` |
| LatestSnapshot returns value copy | Fixed | `internal/agent/instance.go` `LatestSnapshot` |
| P2P ping read deadline (10s) | Fixed | `internal/p2p/node.go` `handlePing` |
| Bootstrap per-peer timeout (30s) | Fixed | `internal/p2p/node.go` `bootstrapPeers` |
| nodeCapabilities mutex protection | Fixed | `internal/migration/service.go` `SetNodeCapabilities` |
| `Names()` sorted output | Fixed | `pkg/manifest/manifest.go` |
| `History()` returns slice copy | Fixed | `internal/eventlog/eventlog.go` |
| Hostcall registry error wrapping | Fixed | `internal/hostcall/registry.go` |
| Oversized log_emit warning | Fixed | `internal/hostcall/log.go` |
| Unused `StartTime` field removed | Fixed | `pkg/protocol/messages.go` |
| P2P package tests | Added | `internal/p2p/node_test.go` |
| CLI entry point tests | Added | `cmd/igord/main_test.go` |
| WASM agent build in CI | Added | `.github/workflows/ci.yml` |
| Tick loop extracted to `internal/runner` | Refactored | `internal/runner/runner.go` — testable in isolation |
| `captureState`/`resumeAgent` deduplicated | Refactored | `internal/wasmutil/wasmutil.go` — shared by agent, replay, simulator |
| WASM hash mismatch test | Added | `internal/agent/instance_test.go` `TestLoadCheckpointFromStorage_WASMHashMismatch` |
| Receipt corruption tests | Added | `pkg/receipt/receipt_test.go` — truncated entries, signatures, fields |
| `MustInstantiate` → `Instantiate` | Fixed | `internal/runtime/engine.go`, `internal/replay/engine.go`, `internal/simulator/simulator.go` — returns error instead of panicking |
| Shared tick timeout constant | Fixed | `internal/config/config.go` `TickTimeout` — used by agent, replay, simulator |
| Manifest sidecar loading unified | Refactored | `pkg/manifest/parse.go` `LoadSidecarData` — shared by runner, migration, simulator |
| `validateIncomingManifest` tests | Added | `internal/migration/validate_test.go` — 7 cases: accept, disabled, price, memory, capability, nil policy |
| `LoadSidecarData` tests | Added | `pkg/manifest/parse_test.go` — explicit path, derived path, no file, non-WASM |
| CI: TinyGo before tests | Fixed | `.github/workflows/ci.yml` — WASM integration tests now run in CI |
| CI: test coverage reporting | Added | `.github/workflows/ci.yml` — `go test -coverprofile` + summary |
