# Capability Membrane

> **Specification cross-references:** [Spec Index](../SPEC_INDEX.md) | [Invariant Dependency Graph](../enforcement/INVARIANT_DEPENDENCY_GRAPH.md) | [Runtime Constitution](./RUNTIME_CONSTITUTION.md)

**Spec Layer:** Constitutional
**Stability:** High
**Breaking Changes Require:** RFC + spec version bump
**Related:** [EXECUTION_INVARIANTS.md](./EXECUTION_INVARIANTS.md), [AUTHORITY_STATE_MACHINE.md](./AUTHORITY_STATE_MACHINE.md), [MIGRATION_CONTINUITY.md](./MIGRATION_CONTINUITY.md)

---

## Purpose

This document defines constitutional guarantees governing agent I/O: how agents observe the external world and produce side effects.

These invariants complement the execution invariants (which govern state continuity) with I/O invariants (which govern information flow). Together, they define the complete trust boundary between agent code and the Igor runtime.

This document does NOT prescribe hostcall names, function signatures, encoding formats, or implementation details. See [HOSTCALL_ABI.md](../runtime/HOSTCALL_ABI.md) for the runtime-layer implementation design.

---

## Terminology

| Term | Definition |
|------|-----------|
| **hostcall** | A runtime-mediated function callable by agent WASM code. The sole channel through which agents interact with anything outside their linear memory. |
| **capability** | A named class of hostcall (e.g., clock, randomness, storage, network). Capabilities are declared, granted, and enforced by the runtime. |
| **capability manifest** | Declaration by an agent of which capabilities it requires. Evaluated at load time. |
| **capability grant** | Runtime decision to provide a specific capability to a specific agent instance. Grants are determined at instantiation and fixed for the agent's lifetime on that node. |
| **observation** | A hostcall that returns information from outside the agent's linear memory (e.g., current time, random bytes, stored values). Observations are inputs to tick execution. |
| **side effect** | A hostcall that modifies state outside the agent's linear memory (e.g., writing to storage, sending a network request). Side effects are outputs of tick execution. |

---

## Boundary Invariants

### CM-1: Total Mediation

All agent interaction with resources outside WASM linear memory MUST pass through runtime-provided hostcalls. There MUST be no unmediated channel between agent code and host resources.

**Rationale:** Without total mediation, the runtime cannot enforce resource limits, meter costs, record observations for replay, or maintain determinism. Total mediation is the architectural foundation that enables every other capability invariant.

### CM-2: Explicit Declaration

An agent MUST declare its required capabilities before execution begins. The runtime MUST NOT provide capabilities that were not declared and granted.

**Rationale:** Undeclared capabilities cannot be audited, metered, or reproduced during replay. Explicit declaration enables the runtime to verify that a host can satisfy an agent's requirements before accepting it.

### CM-3: Deny by Default

Capabilities not explicitly granted MUST be unavailable to the agent. The default capability set is empty.

**Rationale:** Least privilege. Agents receive only what they declare and the runtime approves. This prevents accidental exposure of host resources and limits the attack surface of the agent-runtime boundary.

---

## Integrity Invariants

### CM-4: Observation Determinism

Hostcalls that return observations MUST return values that can be recorded and replayed deterministically. The runtime MUST be able to reproduce the exact sequence of observation values an agent received during any tick.

**Rationale:** Without observation determinism, checkpoint lineage verification via replay is impossible. This invariant is the constitutional foundation for the replay engine. It extends the determinism contract established by EI-3 (checkpoint lineage integrity) and RE-6 (tick determinism) from agent-contract responsibility to runtime-enforced guarantee.

### CM-6: Capability Immutability During Tick

The set of capabilities available to an agent MUST NOT change during a tick execution. Capabilities may only be modified at tick boundaries.

**Rationale:** Mid-tick capability changes would create non-deterministic execution paths that cannot be replayed. A tick is an atomic unit of execution; its environment must be stable.

---

## Authority Invariants

### CM-5: Side Effect Attribution

Every side effect produced through a hostcall MUST be attributable to the agent identity that invoked it. Side-effect hostcalls MUST only execute while the agent is in ACTIVE_OWNER state per the [Authority State Machine](./AUTHORITY_STATE_MACHINE.md).

**Rationale:** Unattributed side effects break accountability and make it impossible to determine which agent caused which external state change. Side effects outside ACTIVE_OWNER state would violate the authority lifecycle — only the authoritative node may produce effects on behalf of an agent.

---

## Migration Invariants

### CM-7: Capability Survival Through Migration

An agent's declared capability requirements MUST be preserved through migration. The target node MUST verify it can satisfy the agent's capability requirements before accepting authority.

**Rationale:** Migration that silently drops capabilities would change the agent's execution environment, potentially causing failures that the agent cannot anticipate. If a target cannot provide a required capability, it MUST reject the migration rather than accept with degraded capability.

---

## Relationship to Existing Invariants

| Capability Invariant | Relates To | Relationship |
|---------------------|-----------|--------------|
| CM-1 (total mediation) | RE-7 (storage isolation), WASM sandbox | Strengthens the sandbox boundary from passive isolation to active mediation |
| CM-2 (explicit declaration) | MC-2 (migration scope) | Capability manifest becomes part of agent package |
| CM-4 (observation determinism) | EI-3 (lineage integrity), RE-6 (tick determinism) | Extends determinism from agent contract to runtime-enforced observation recording |
| CM-5 (side effect attribution) | AUTHORITY_STATE_MACHINE.md tick permission matrix | Extends authority enforcement to cover I/O side effects |
| CM-6 (capability immutability) | RE-6 (tick determinism) | Strengthens tick determinism by fixing the execution environment |
| CM-7 (capability survival) | MC-2 (migration scope), MC-4 (identity preservation) | Extends migration scope to include capability requirements |

---

## Normative Statements

### Hostcalls as the Sole I/O Channel

The capability membrane defines a complete boundary between agent computation and external interaction. An agent's WASM linear memory is private and opaque. The hostcall interface is the only window through which information crosses this boundary in either direction.

### Observation Recording as a Runtime Duty

CM-4 places the obligation of observation determinism on the runtime, not the agent. The agent does not need to know whether it is being replayed. The runtime MUST ensure that observation hostcalls behave identically during live execution and replay.

### Capability Declaration as an Integrity Mechanism

CM-2 is not merely a convenience for the runtime — it is an integrity mechanism. The declared capability set defines the agent's external interface. Any change to this set changes the agent's behavior contract. Capability declarations are therefore part of the agent's identity metadata, alongside its WASM binary and checkpoint lineage.

---

## Invariant Summary

| ID | Invariant | Category |
|----|-----------|----------|
| CM-1 | All agent I/O passes through runtime hostcalls | Boundary |
| CM-2 | Agents declare required capabilities before execution | Boundary |
| CM-3 | Undeclared capabilities are unavailable (deny by default) | Boundary |
| CM-4 | Observation hostcalls are deterministically replayable | Integrity |
| CM-5 | Side effects attributed to agent identity, gated on ACTIVE_OWNER | Authority |
| CM-6 | Capability set fixed during tick execution | Integrity |
| CM-7 | Capability requirements preserved through migration | Migration |

---

## Document Status

**Type:** Constitutional Specification
**Scope:** Conceptual contracts only — no hostcall names, function signatures, encoding formats, or implementation details.
**Authority:** Normative for all future implementation of agent I/O, capability enforcement, and observation recording. Part of the constitutional layer defined by [RUNTIME_CONSTITUTION.md](./RUNTIME_CONSTITUTION.md).
