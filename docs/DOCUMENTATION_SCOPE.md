# Documentation Scope

This document defines which documentation belongs in the public repository and which should be archived or excluded.

## Public Documentation

**Purpose:** Help contributors understand, use, and extend Igor.

**Location:** `/docs` (remains in repository)

### Technical Documentation

**ARCHITECTURE.md** - Runtime implementation details  
**AGENT_LIFECYCLE.md** - Building and deploying agents  
**MIGRATION_PROTOCOL.md** - P2P migration mechanics  
**BUDGET_MODEL.md** - Economic model and metering  
**SECURITY_MODEL.md** - Threat model and sandbox constraints  
**INVARIANTS.md** - System guarantees and verification  

### Conceptual Documentation

**OVERVIEW.md** - Introduction to Igor concepts  
**VISION.md** - Why autonomous software needs survival  

### Process Documentation

**DEVELOPMENT.md** - Developer setup and workflow  
**CI_PIPELINE.md** - Continuous integration documentation  
**RELEASE_PROCESS.md** - Release management  
**ROADMAP.md** - Future development phases (if technical, not speculative)

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

## Documentation Maintenance Rules

### Adding New Documentation

New documentation must:

1. **Serve technical contributors** - Helps someone understand or extend Igor
2. **Be technically grounded** - Based on actual implementation, not speculation
3. **Avoid redundancy** - Does not duplicate existing docs
4. **Follow naming convention** - UPPERCASE.md in /docs
5. **Use technical tone** - Infrastructure documentation, not marketing

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
