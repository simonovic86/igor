# Keyword Governance

This document defines approved terminology for describing Igor in documentation, metadata, and public communications.

## Purpose

Maintain discoverability through technical accuracy. Keywords must:
- Reflect actual implementation
- Aid GitHub search indexing
- Avoid marketing exaggeration
- Preserve technical credibility

## Core Identity Keywords

**Primary (required in main description):**
- Runtime for survivable autonomous agents
- Decentralized execution runtime
- Autonomous agent infrastructure
- WASM agent runtime

**Secondary (use in detailed descriptions):**
- Peer-to-peer agent migration
- Runtime economic accounting
- Distributed systems infrastructure
- Survivable software primitives

## Technical Stack Keywords

**Approved:**
- WASM sandbox execution
- WebAssembly runtime (wazero specifically)
- libp2p networking
- Peer-to-peer migration protocol
- Go distributed systems
- Agent checkpoint persistence
- Budget metering infrastructure

**Context-required:**
- Agent runtime (specify "survival-focused")
- Distributed runtime (specify "for agents")
- Economic agents (specify "with budget metering")

## Use Case Keywords

**Accurate:**
- DeFi automation infrastructure
- Oracle network execution
- AI service agent runtime
- Economic software infrastructure
- Autonomous service execution

**Requires disclaimer:**
- "Suitable for" not "designed for"
- "Infrastructure layer for" not "complete solution for"
- "Experimental" status must accompany production use cases

## Ecosystem Alignment Keywords

**Adjacent domains (use with context):**
- Autonomous AI agent infrastructure
- Distributed compute runtimes
- WebAssembly execution engines
- Peer-to-peer service fabrics
- Economic automation systems
- Solver network infrastructure

**Always include:**
"Igor provides lower-level runtime infrastructure, not application-level frameworks."

## Disallowed Language

**Marketing buzzwords:**
- Revolutionary
- Groundbreaking
- Next-generation
- Cutting-edge (use "experimental" instead)
- Industry-leading
- Enterprise-grade (Igor is explicitly NOT production-ready)

**Hype phrases:**
- "The future of..."
- "Reimagining..."
- "Transforming the industry..."
- "Unprecedented capabilities..."
- "Game-changing technology..."

**Investor language:**
- Market opportunity
- Total addressable market
- Go-to-market strategy
- Competitive advantages
- Monetization potential

**Exaggerated technical claims:**
- Production-ready (Igor is experimental)
- Battle-tested (Igor is research-stage)
- Enterprise security (Igor has limited security)
- Proven at scale (Igor is proof-of-concept)

## Approved Descriptive Phrases

**For README/documentation:**
- "Experimental distributed systems runtime"
- "Research-stage infrastructure"
- "Proof-of-concept survival runtime"
- "Minimal survival primitives"
- "Demonstrates feasibility of autonomous agent survival"

**For GitHub description:**
- "Runtime for survivable autonomous software agents using WASM, migration, and runtime economics"

**For technical discussions:**
- "Implements checkpointing, migration, and budget enforcement"
- "Enables agents to survive infrastructure failure"
- "Provides survival autonomy for economic software"

## Topic Tag Policy

**GitHub repository topics must be:**
- Technically accurate
- Searchably relevant
- Not misleading about maturity
- Aligned with actual implementation

**Approved topics:**
autonomous-agents, distributed-systems, wasm-runtime, webassembly, libp2p, runtime-economics, survivable-software, agent-infrastructure, peer-to-peer, systems-research, go, distributed-runtime, agent-runtime, survivable-systems, p2p-runtime, economic-agents, execution-runtime, research-infrastructure, distributed-compute, wasm

**Rejected topics:**
production, enterprise, blockchain, ai-platform, agent-marketplace

## Search Query Alignment

Igor should be discoverable via these queries (test periodically):

**Primary:**
- "autonomous agent runtime"
- "wasm agent runtime"
- "distributed agent infrastructure"
- "libp2p runtime"
- "survivable software agents"

**Secondary:**
- "agent migration runtime"
- "runtime economics"
- "peer-to-peer execution"
- "wasm distributed systems"
- "autonomous software infrastructure"

**Tertiary:**
- "defi automation infrastructure"
- "oracle runtime"
- "economic agent runtime"
- "process migration runtime"

## Tagline Consistency

**Canonical tagline:**
"Runtime for survivable autonomous agents"

**Variations (context-appropriate):**
- "Runtime for survivable autonomous software agents" (full form)
- "Autonomous agent survival runtime" (action-oriented)
- "Decentralized runtime for autonomous agents" (architecture-focused)

**Disallowed variations:**
- Adding "advanced", "powerful", "revolutionary"
- Removing "survivable" (core differentiator)
- Removing "runtime" (identity anchor)

## Maintenance Rules

**When updating README or metadata:**
1. Check keywords against this document
2. Ensure technical accuracy
3. Avoid introducing hype
4. Preserve experimental status clarity
5. Maintain alignment with PROJECT_CONTEXT.md

**When adding new keywords:**
1. Verify technical accuracy
2. Test GitHub search relevance
3. Update this document
4. Commit with justification

**Quarterly review:**
- Verify keyword accuracy against implementation
- Remove deprecated or misleading terms
- Add new relevant technical domains
- Maintain credibility over discoverability

## Philosophy Alignment

From PROJECT_CONTEXT.md:
> "Igor v0 is intentionally minimal. It is a proof-of-survival runtime."

Keywords must reflect:
- Minimalism (not comprehensiveness)
- Survival focus (not general-purpose)
- Experimental status (not production-ready)
- Research orientation (not commercial)

Discoverability through truthful precision, not exaggeration.
