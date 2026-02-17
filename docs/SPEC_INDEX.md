# Igor Specification Index

This document serves as the cross-reference index for all Igor specification documents. It maps every document to its specification layer, describes its purpose, and lists cross-references to related documents.

---

## Constitutional Guarantees

Constitutional documents define **WHAT** Igor must guarantee — non-negotiable execution invariants that hold regardless of implementation.

| Document | Description | Category | Cross-References |
|----------|-------------|----------|-----------------|
| [RUNTIME_CONSTITUTION.md](./constitution/RUNTIME_CONSTITUTION.md) | Constitutional specification root. Declares non-negotiable guarantees and the foundational separation between checkpoint lineage and ownership authority. | Root | [EXECUTION_INVARIANTS](./constitution/EXECUTION_INVARIANTS.md), [OWNERSHIP_AND_AUTHORITY](./constitution/OWNERSHIP_AND_AUTHORITY.md), [MIGRATION_CONTINUITY](./constitution/MIGRATION_CONTINUITY.md), [SPEC_GOVERNANCE](./governance/SPEC_GOVERNANCE.md) |
| [EXECUTION_INVARIANTS.md](./constitution/EXECUTION_INVARIANTS.md) | Foundational runtime invariants (EI-1 through EI-11): single active instance, checkpoint lineage integrity, authority separation, safety over liveness, migration preservation. | Identity, Continuity, Authority, Safety, Migration | [RUNTIME_CONSTITUTION](./constitution/RUNTIME_CONSTITUTION.md), [OWNERSHIP_AND_AUTHORITY](./constitution/OWNERSHIP_AND_AUTHORITY.md), [RUNTIME_ENFORCEMENT_INVARIANTS](./enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md) |
| [OWNERSHIP_AND_AUTHORITY.md](./constitution/OWNERSHIP_AND_AUTHORITY.md) | Authority lifecycle model (OA-1 through OA-7): canonical identity, lifecycle states (ACTIVE_OWNER, HANDOFF_INITIATED, HANDOFF_PENDING, RETIRED, RECOVERY_REQUIRED), transfer rules, conflict resolution. | Identity, Authority | [EXECUTION_INVARIANTS](./constitution/EXECUTION_INVARIANTS.md), [RUNTIME_CONSTITUTION](./constitution/RUNTIME_CONSTITUTION.md) |
| [MIGRATION_CONTINUITY.md](./constitution/MIGRATION_CONTINUITY.md) | Migration continuity contracts (MC-1 through MC-9, FS-1 through FS-4): migration as continuity transfer, safe migration guarantees, overlap constraints, lineage fork detection, failure safety matrix. | Migration, Safety | [EXECUTION_INVARIANTS](./constitution/EXECUTION_INVARIANTS.md), [OWNERSHIP_AND_AUTHORITY](./constitution/OWNERSHIP_AND_AUTHORITY.md), [RUNTIME_CONSTITUTION](./constitution/RUNTIME_CONSTITUTION.md) |
| [AUTHORITY_STATE_MACHINE.md](./constitution/AUTHORITY_STATE_MACHINE.md) | Formal authority lifecycle state machine: state definitions, normative transition table, forbidden transitions, tick permission matrix. Operationalizes single-active-ticker guarantee. | Authority, Lifecycle | [EXECUTION_INVARIANTS](./constitution/EXECUTION_INVARIANTS.md), [OWNERSHIP_AND_AUTHORITY](./constitution/OWNERSHIP_AND_AUTHORITY.md), [MIGRATION_CONTINUITY](./constitution/MIGRATION_CONTINUITY.md), [RUNTIME_CONSTITUTION](./constitution/RUNTIME_CONSTITUTION.md) |

---

## Enforcement Layer

Enforcement documents define **HOW** constitutional guarantees are upheld through runtime enforcement rules and invariant derivation mappings. Every enforcement invariant derives from one or more constitutional invariants.

| Document | Description | Cross-References |
|----------|-------------|-----------------|
| [RUNTIME_ENFORCEMENT_INVARIANTS.md](./enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md) | Enforcement rules (RE-1 through RE-8): checkpoint atomicity, state persistence, budget conservation and monotonicity, tick duration and determinism, storage isolation, lifecycle ordering. | [RUNTIME_CONSTITUTION](./constitution/RUNTIME_CONSTITUTION.md), [EXECUTION_INVARIANTS](./constitution/EXECUTION_INVARIANTS.md), [OWNERSHIP_AND_AUTHORITY](./constitution/OWNERSHIP_AND_AUTHORITY.md), [MIGRATION_CONTINUITY](./constitution/MIGRATION_CONTINUITY.md) |
| [INVARIANT_DEPENDENCY_GRAPH.md](./enforcement/INVARIANT_DEPENDENCY_GRAPH.md) | Maps dependency relationships between constitutional and enforcement invariants. Traces the derivation chain from single-ticker law through authority lifecycle, ownership sidecar, migration ordering, and recovery fencing. | [EXECUTION_INVARIANTS](./constitution/EXECUTION_INVARIANTS.md), [OWNERSHIP_AND_AUTHORITY](./constitution/OWNERSHIP_AND_AUTHORITY.md), [MIGRATION_CONTINUITY](./constitution/MIGRATION_CONTINUITY.md), [RUNTIME_ENFORCEMENT_INVARIANTS](./enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md) |

---

## Runtime Architecture

Runtime documents describe **HOW** Igor operates — implementation details, protocol mechanics, and operational flows.

| Document | Description | Cross-References |
|----------|-------------|-----------------|
| [ARCHITECTURE.md](./runtime/ARCHITECTURE.md) | System structure, component overview, and implementation details. | [RUNTIME_ENFORCEMENT_INVARIANTS](./enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md), [EXECUTION_INVARIANTS](./constitution/EXECUTION_INVARIANTS.md) |
| [AGENT_LIFECYCLE.md](./runtime/AGENT_LIFECYCLE.md) | Agent development guide: lifecycle functions, building and deploying agents. | [RUNTIME_ENFORCEMENT_INVARIANTS](./enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md), [EXECUTION_INVARIANTS](./constitution/EXECUTION_INVARIANTS.md) |
| [MIGRATION_PROTOCOL.md](./runtime/MIGRATION_PROTOCOL.md) | P2P migration protocol mechanics and message flows. | [MIGRATION_CONTINUITY](./constitution/MIGRATION_CONTINUITY.md) |
| [BUDGET_MODEL.md](./runtime/BUDGET_MODEL.md) | Economic model, execution metering, and budget enforcement. | [RUNTIME_ENFORCEMENT_INVARIANTS](./enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md) |
| [THREAT_MODEL.md](./runtime/THREAT_MODEL.md) | Canonical runtime threat assumptions: system/failure model, adversary classes, network assumptions, and trust boundaries. | [SECURITY_MODEL](./runtime/SECURITY_MODEL.md), [EXECUTION_INVARIANTS](./constitution/EXECUTION_INVARIANTS.md), [MIGRATION_CONTINUITY](./constitution/MIGRATION_CONTINUITY.md) |
| [SECURITY_MODEL.md](./runtime/SECURITY_MODEL.md) | Current security mechanisms and explicit limitations under the threat assumptions. | [THREAT_MODEL](./runtime/THREAT_MODEL.md) |

---

## Governance and Evolution

Governance documents define **HOW Igor evolves** — processes, workflows, and change control.

| Document | Description |
|----------|-------------|
| [SPEC_GOVERNANCE.md](./governance/SPEC_GOVERNANCE.md) | Specification change control, layer classification, and constitutional freeze rules. |
| [DOCUMENTATION_SCOPE.md](./governance/DOCUMENTATION_SCOPE.md) | Documentation boundaries and layer definitions. |
| [DEVELOPMENT.md](./governance/DEVELOPMENT.md) | Developer setup and workflow. |
| [CI_PIPELINE.md](./governance/CI_PIPELINE.md) | Continuous integration documentation. |
| [RELEASE_PROCESS.md](./governance/RELEASE_PROCESS.md) | Release management procedures. |
| [TOOLCHAIN.md](./governance/TOOLCHAIN.md) | Build toolchain and version requirements. |
| [ROADMAP.md](./governance/ROADMAP.md) | Future development phases. |
| [KEYWORDS.md](./governance/KEYWORDS.md) | Keyword governance policy. |

---

## Philosophy

Philosophy documents explain **WHY Igor exists** — motivation and conceptual framing.

| Document | Description |
|----------|-------------|
| [VISION.md](./philosophy/VISION.md) | Why autonomous software needs survival. |
| [OVERVIEW.md](./philosophy/OVERVIEW.md) | Introduction to Igor concepts and current status. |

---

## Layer Boundary Definitions

| Layer | Purpose | Scope |
|-------|---------|-------|
| **Constitution** | WHAT Igor guarantees | Non-negotiable invariants. Mechanism-agnostic, field-agnostic, implementation-agnostic. |
| **Enforcement** | HOW guarantees are upheld | Enforcement rules derived from constitutional invariants. May reference runtime concepts. |
| **Runtime** | HOW Igor operates | Implementation details, protocols, algorithms, operational flows. Must comply with all higher layers. |
| **Governance** | HOW Igor evolves | Change control, development process, release management. |
| **Philosophy** | WHY Igor exists | Motivation, worldview, conceptual framing. |

Each layer derives authority from the layer above. No lower layer may introduce or weaken guarantees defined at a higher layer.
