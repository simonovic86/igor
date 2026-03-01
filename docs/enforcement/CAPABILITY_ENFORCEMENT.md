# Capability Enforcement Invariants

> **Specification cross-references:** [Spec Index](../SPEC_INDEX.md) | [Invariant Dependency Graph](./INVARIANT_DEPENDENCY_GRAPH.md) | [Runtime Constitution](../constitution/RUNTIME_CONSTITUTION.md)
>
> This document defines enforcement rules derived from the constitutional guarantees in [CAPABILITY_MEMBRANE.md](../constitution/CAPABILITY_MEMBRANE.md). Enforcement invariants operationalize constitutional contracts; they do not introduce new guarantees.

## Overview

This document defines capability enforcement rules that implement the constitutional guarantees specified in [CAPABILITY_MEMBRANE.md](../constitution/CAPABILITY_MEMBRANE.md).

These invariants describe **how the runtime upholds capability membrane guarantees** through concrete enforcement mechanisms. They are derived from, and subordinate to, the capability membrane invariants (CM-1 through CM-7) and the existing runtime enforcement rules in [RUNTIME_ENFORCEMENT_INVARIANTS.md](./RUNTIME_ENFORCEMENT_INVARIANTS.md).

---

## Hostcall Registration

### CE-1: Hostcall Namespace Isolation

**Invariant:** The runtime MUST register only declared and granted hostcalls in the WASM module import namespace. Hostcalls not in the granted set MUST NOT be importable by agent code.

**Derives from:** CM-1 (total mediation), CM-3 (deny by default)

**Why:**
- Prevents agents from accessing undeclared capabilities
- Enforces least privilege at the WASM instantiation boundary
- Makes capability violations impossible rather than merely detectable

**Enforcement:**
- Parse capability manifest at agent load time
- Build hostcall module with only granted functions
- WASM instantiation fails if agent imports ungrantable functions
- No runtime fallback for missing capabilities

**Violation consequences:**
- Agent gains access to unaudited, unmetered external resources
- Observation recording incomplete, replay impossible
- Budget accounting inaccurate

**Detected by:**
- WASM link errors (expected behavior for denied capabilities)
- Audit of registered host module functions

**Status:** Implemented — `internal/hostcall/registry.go` registers only declared capabilities; wazero link error enforces deny-by-default at instantiation. Covered by `TestLoadAgent_EmptyManifest`.

---

### CE-2: Manifest Validation at Load

**Invariant:** The runtime MUST validate the capability manifest against available node capabilities before agent instantiation. Agents declaring unavailable capabilities MUST be rejected at load time.

**Derives from:** CM-2 (explicit declaration)

**Why:**
- Prevents runtime failures from missing capabilities
- Fail-fast over fail-during-execution
- Enables pre-migration capability checking

**Enforcement:**
- Manifest parsed and validated before WASM compilation
- Node advertises available capabilities
- Mismatch between manifest and node capabilities produces immediate rejection
- Rejection logged with specific capability names

**Violation consequences:**
- Agent instantiated without required capabilities
- Runtime errors during tick execution
- Unpredictable agent behavior

**Detected by:**
- Load-time validation errors
- Agent crash during tick due to missing hostcall

**Status:** Implemented — `agent.LoadAgent()` calls `manifest.ParseCapabilityManifest()` and `manifest.ValidateAgainstNode()` before WASM compilation. Agents declaring unavailable capabilities are rejected at load time.

---

## Observation Recording

### CE-3: Tick Observation Log

**Invariant:** The runtime MUST record the return value of every observation hostcall in the tick's event log. Recorded values MUST be sufficient to replay the tick deterministically.

**Derives from:** CM-4 (observation determinism)

**Why:**
- Enables replay-based verification of tick execution
- Provides audit trail for agent behavior
- Separates observation source (live vs replay) from agent logic

**Enforcement:**
- Each observation hostcall writes (hostcall_id, return_value) to per-tick event log
- Event log appended atomically during tick execution
- Event log sealed at tick boundary (before checkpoint)
- During replay, hostcalls return recorded values instead of live data

**Violation consequences:**
- Replay produces different results than live execution
- Cannot verify tick correctness
- Lineage integrity claims unverifiable

**Detected by:**
- Replay divergence (different state after replay vs live execution)
- Missing entries in observation log
- Event log format validation errors

**Status:** Partially implemented — all observation hostcalls record to the per-tick EventLog; `BeginTick`/`SealTick` bound each tick; the replay engine consumes the log deterministically. NOT YET: tick logs are not persisted to disk and only the most recent tick's replay data is transferred during migration.

---

## Side Effect Gating

### CE-4: Authority-Gated Side Effects

**Invariant:** Hostcalls that produce side effects MUST verify the agent is in ACTIVE_OWNER state before executing. Side-effect hostcalls invoked outside ACTIVE_OWNER state MUST fail immediately with an error.

**Derives from:** CM-5 (side effect attribution)

**Why:**
- Prevents unauthorized state changes from non-authoritative nodes
- Ensures side effects are attributable to a single authority holder
- Prevents split-brain effects during migration or recovery

**Enforcement:**
- Side-effect hostcalls check authority state before execution
- Return error code on authority state mismatch
- Authority state cached per tick (immutable during tick per CM-6)

**Violation consequences:**
- Side effects produced by non-authoritative node
- Conflicting external state changes
- Attribution broken

**Detected by:**
- Side-effect hostcall succeeding outside ACTIVE_OWNER
- Audit log showing effects without authority

**Status:** Not implemented — no authority state machine exists in the runtime. `AUTHORITY_STATE_MACHINE.md` is specified but the `ACTIVE_OWNER` check in hostcalls has not been built.

---

## Migration Enforcement

### CE-5: Pre-Migration Capability Verification

**Invariant:** Before accepting a migration, the target node MUST verify it can satisfy all capabilities declared in the agent's manifest. Migration to a node that cannot satisfy declared capabilities MUST be rejected.

**Derives from:** CM-7 (capability survival through migration)

**Why:**
- Prevents silent capability degradation on migration
- Agent behavior contract preserved across nodes
- Fail-fast: reject at migration time, not at tick time

**Enforcement:**
- Migration handshake includes capability manifest
- Target verifies manifest against local capabilities
- Rejection message includes specific unsatisfied capabilities
- Source node retains authority on migration rejection

**Violation consequences:**
- Agent migrated to node that cannot satisfy its requirements
- Tick execution fails due to missing hostcalls
- Agent becomes non-functional after migration

**Detected by:**
- Post-migration tick failures
- Capability mismatch in migration logs

**Status:** Not implemented — target performs CE-2 validation at `agent.LoadAgent()` time (after checkpoint is already stored), but there is no pre-handshake capability negotiation. The source cannot verify target capability compatibility before initiating the transfer.

---

## Metering

### CE-6: Hostcall Cost Accounting

**Invariant:** Hostcall execution time MUST be included in tick cost calculation. The budget model MUST account for hostcall overhead.

**Derives from:** CM-1 (total mediation), RE-3 (budget conservation)

**Why:**
- Hostcalls consume real resources (I/O, compute, network)
- Agents cannot bypass budget enforcement via expensive hostcalls
- Fair metering requires complete cost accounting

**Enforcement:**
- Tick duration includes all hostcall execution time
- No separate hostcall budget (unified with tick budget)
- Timeout enforcement (RE-5) covers hostcall duration

**Violation consequences:**
- Agents consume resources beyond their budget
- Budget model underestimates actual costs
- Node undercompensated for execution

**Detected by:**
- Tick cost lower than expected given hostcall activity
- Budget discrepancies across migrations

**Status:** Effectively implemented — tick duration is measured across the full `fn.Call()` invocation which executes all hostcalls synchronously, so hostcall execution time is included in tick cost by design.

---

## Enforcement Summary

| ID | Invariant | Category | Derives From |
|----|-----------|----------|-------------|
| CE-1 | Only declared/granted hostcalls importable by agent | Registration | CM-1, CM-3 |
| CE-2 | Manifest validated against node capabilities at load | Registration | CM-2 |
| CE-3 | All observation hostcall returns recorded in event log | Observation | CM-4 |
| CE-4 | Side-effect hostcalls gated on ACTIVE_OWNER state | Authority | CM-5 |
| CE-5 | Migration target verifies capability satisfaction | Migration | CM-7 |
| CE-6 | Hostcall execution time included in tick cost | Metering | CM-1, RE-3 |

---

## Document Status

**Type:** Enforcement Specification
**Scope:** Enforcement rules implementing capability membrane guarantees — includes implementation-level detail.
**Authority:** Subordinate to [CAPABILITY_MEMBRANE.md](../constitution/CAPABILITY_MEMBRANE.md) and [RUNTIME_CONSTITUTION.md](../constitution/RUNTIME_CONSTITUTION.md).
