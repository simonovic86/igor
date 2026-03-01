# Documentation Scope

This document defines which documentation belongs in the public repository and which should be archived or excluded.

## Public Documentation

**Purpose:** Help contributors understand, use, and extend Igor.

**Location:** `/docs` organized into authority-layer subfolders

### Constitutional Layer — `docs/constitution/`

Constitutional documents define non-negotiable runtime guarantees. They MUST remain mechanism-agnostic and field-agnostic. They MUST NOT reference protocol message fields, wire formats, or serialization schemas.

**constitution/RUNTIME_CONSTITUTION.md** - Constitutional specification root
**constitution/EXECUTION_INVARIANTS.md** - Foundational runtime invariants
**constitution/OWNERSHIP_AND_AUTHORITY.md** - Authority lifecycle model
**constitution/MIGRATION_CONTINUITY.md** - Migration continuity contracts
**constitution/AUTHORITY_STATE_MACHINE.md** - Formal authority lifecycle state machine
**constitution/CAPABILITY_MEMBRANE.md** - Capability membrane invariants

### Enforcement Layer — `docs/enforcement/`

Enforcement documents define how constitutional guarantees are upheld through runtime enforcement rules and invariant derivation mappings. Every enforcement invariant derives from one or more constitutional invariants.

**enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md** - Enforcement rules implementing constitutional guarantees
**enforcement/INVARIANT_DEPENDENCY_GRAPH.md** - Invariant dependency relationships and cross-document traceability
**enforcement/CAPABILITY_ENFORCEMENT.md** - Capability enforcement rules

### Runtime Implementation Layer — `docs/runtime/`

Implementation documents describe how the runtime is built. They contain protocol details, code-level guidance, and operational procedures. They MUST comply with all constitutional and enforcement invariants.

**runtime/ARCHITECTURE.md** - Runtime implementation details  
**runtime/AGENT_LIFECYCLE.md** - Building and deploying agents  
**runtime/MIGRATION_PROTOCOL.md** - P2P migration mechanics  
**runtime/BUDGET_MODEL.md** - Economic model and metering  
**runtime/SECURITY_MODEL.md** - Threat model and sandbox constraints
**runtime/THREAT_MODEL.md** - Canonical threat assumptions and adversary classes
**runtime/HOSTCALL_ABI.md** - Hostcall interface design
**runtime/REPLAY_ENGINE.md** - Replay engine design (draft)
**runtime/LEASE_EPOCH.md** - Lease-based authority epochs design (draft)

### Philosophy Layer — `docs/philosophy/`

**philosophy/OVERVIEW.md** - Introduction to Igor concepts  
**philosophy/VISION.md** - Why autonomous software needs survival  

### Governance Layer — `docs/governance/`

**governance/DEVELOPMENT.md** - Developer setup and workflow  
**governance/CI_PIPELINE.md** - Continuous integration documentation  
**governance/RELEASE_PROCESS.md** - Release management  
**governance/ROADMAP.md** - Future development phases (if technical, not speculative)  
**governance/SPEC_GOVERNANCE.md** - Specification change control and classification  
**governance/DOCUMENTATION_SCOPE.md** - This document  
**governance/TOOLCHAIN.md** - Build toolchain and version requirements  
**governance/DISCOVERY_VALIDATION.md** - Search and discovery validation  
**governance/KEYWORDS.md** - Keyword governance policy  

### Archive — `docs/archive/`

**archive/GENESIS_COMMIT.md** - Genesis commit message source  
**archive/GENESIS_TAG_ANNOTATION.md** - Genesis tag annotation content  
**archive/GENESIS_RELEASE_CHECKLIST.md** - Release verification checklist  
**archive/HISTORY_REWRITE.md** - History rewrite rationale and process

### Criteria for Public Documentation

Must satisfy at least one:

- Explains how Igor works technically
- Helps developers build agents or extend runtime
- Documents design decisions and invariants
- Provides operational or security guidance
- Explains project philosophy grounded in technical reality

Must NOT:

- Focus on business positioning
- Target investor audiences
- Emphasize market opportunity
- Contain fundraising language
- Speculate about business models

---

## Archived Documentation

**Purpose:** Preserve historical positioning materials not relevant to technical contributors.

**Location:** `docs/archive/` or deleted entirely

### Investor/Business Materials

Documents focused on:
- Market opportunity analysis
- Business model speculation
- Investor positioning
- Fundraising narratives
- Competitive positioning
- Economic projections

**Examples (if they existed):**
- INVESTOR_MEMO.md
- INVESTOR_PITCH.md
- MARKET_ANALYSIS.md
- BUSINESS_MODEL.md

**Decision:** Already deleted in previous cleanup. Do not recreate.

### Redundant Materials

Documents that duplicate content better covered elsewhere:
- Multiple overlapping vision documents
- Redundant architecture descriptions
- Duplicate getting-started guides

**Resolution:** Consolidated into single focused documents.

---

## Specification Layer Boundaries

### What Belongs in the Constitutional Layer

- Non-negotiable runtime guarantees (invariants)
- Authority lifecycle state definitions
- Migration safety contracts
- Failure safety invariant outcomes
- Shared terminology definitions

Constitutional documents MUST remain mechanism-agnostic and field-agnostic. They define *what* must be true, never *how* it is achieved.

### What Belongs in the Mechanism Design Layer

- Enforcement rules that implement constitutional invariants
- Invariant derivation chains and dependency graphs
- Detection and verification mechanisms
- Operational failure modes

Mechanism design documents MUST trace every rule to constitutional justification.

### What Belongs in the Runtime Implementation Layer

- Protocol message formats and wire encoding
- Code architecture and module structure
- Serialization schemas and byte layouts
- Operational procedures and deployment guidance
- Performance characteristics and resource limits

Implementation documents MUST comply with higher layers but are otherwise unconstrained in technical detail.

---

## Documentation Maintenance Rules

### Adding New Documentation

New documentation must:

1. **Serve technical contributors** - Helps someone understand or extend Igor
2. **Be technically grounded** - Based on actual implementation, not speculation
3. **Avoid redundancy** - Does not duplicate existing docs
4. **Follow naming convention** - UPPERCASE.md in /docs
5. **Use technical tone** - Infrastructure documentation, not marketing
6. **Respect layer boundaries** - Content must be placed in the correct specification layer

### Removing Documentation

Remove documentation if:

1. **Duplicative** - Content covered better elsewhere
2. **Speculative** - Not grounded in current implementation
3. **Business-focused** - Targets investors/customers, not contributors
4. **Outdated** - Describes deprecated or removed functionality

### Documentation Review

Quarterly review of /docs to ensure:

- No redundant content
- All references valid
- Technical accuracy maintained
- Scope remains contributor-focused

---

## Current Status

**Public documentation:** 12 files, ~128KB  
**Archived documentation:** None (investor materials deleted)  
**Last review:** 2026-02-11 (genesis preparation)

All current documentation serves technical contributors and maintains focus on Igor as a survival runtime for autonomous economic agents.
