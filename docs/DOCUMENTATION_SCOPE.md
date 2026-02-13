# Documentation Scope

This document defines which documentation belongs in the public repository and which should be archived or excluded.

## Public Documentation

**Purpose:** Help contributors understand, use, and extend Igor.

**Location:** `/docs` (remains in repository)

### Constitutional Layer

Constitutional documents define non-negotiable runtime guarantees. They MUST remain mechanism-agnostic and field-agnostic. They MUST NOT reference protocol message fields, wire formats, or serialization schemas.

**RUNTIME_CONSTITUTION.md** - Constitutional specification root  
**EXECUTION_INVARIANTS.md** - Foundational runtime invariants  
**OWNERSHIP_AND_AUTHORITY.md** - Authority lifecycle model  
**MIGRATION_CONTINUITY.md** - Migration continuity contracts  

### Mechanism Design Layer

Mechanism design documents describe how constitutional guarantees are enforced. Every enforcement invariant must trace to one or more constitutional invariants. These documents MAY reference implementation-level concepts.

**RUNTIME_ENFORCEMENT_INVARIANTS.md** - Enforcement rules implementing constitutional guarantees  
**INVARIANT_DEPENDENCY_GRAPH.md** - Invariant dependency relationships  

### Runtime Implementation Layer

Implementation documents describe how the runtime is built. They contain protocol details, code-level guidance, and operational procedures. They MUST comply with all constitutional and enforcement invariants.

**ARCHITECTURE.md** - Runtime implementation details  
**AGENT_LIFECYCLE.md** - Building and deploying agents  
**MIGRATION_PROTOCOL.md** - P2P migration mechanics  
**BUDGET_MODEL.md** - Economic model and metering  
**SECURITY_MODEL.md** - Threat model and sandbox constraints  

### Conceptual Documentation

**OVERVIEW.md** - Introduction to Igor concepts  
**VISION.md** - Why autonomous software needs survival  

### Process Documentation

**DEVELOPMENT.md** - Developer setup and workflow  
**CI_PIPELINE.md** - Continuous integration documentation  
**RELEASE_PROCESS.md** - Release management  
**ROADMAP.md** - Future development phases (if technical, not speculative)  
**SPEC_GOVERNANCE.md** - Specification change control and classification

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
