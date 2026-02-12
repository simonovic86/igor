# Discovery Validation

This document tracks Igor's GitHub search discoverability and validates keyword effectiveness.

## Primary Search Queries

Test Igor's appearance in GitHub repository search for these queries:

### Core Identity Queries

**"autonomous agent runtime"**
- Expected: Igor appears in top results
- Relevance: Direct match to core functionality
- Competition: General agent frameworks, orchestration platforms
- Igor differentiator: Survival-focused, not reasoning-focused

**"wasm agent runtime"**
- Expected: Igor appears prominently
- Relevance: WASM-based agent execution is core architecture
- Competition: General WASM runtimes, serverless platforms
- Igor differentiator: Agent-specific, checkpoint/migration support

**"distributed agent infrastructure"**
- Expected: Igor appears in relevant results
- Relevance: Distributed systems context
- Competition: Cloud platforms, orchestration systems
- Igor differentiator: Decentralized P2P, not centralized platform

**"libp2p runtime"**
- Expected: Igor appears in libp2p ecosystem searches
- Relevance: Uses libp2p for P2P migration
- Competition: IPFS, Filecoin, libp2p demos
- Igor differentiator: Agent execution runtime, not storage/content

**"survivable software agents"**
- Expected: Igor should be highly ranked (unique positioning)
- Relevance: Core thesis - software survival autonomy
- Competition: Limited (few projects focus on survival primitives)
- Igor differentiator: Explicit survival as primary feature

### Technical Stack Queries

**"wazero runtime"**
- Expected: Igor appears among wazero users
- Relevance: Uses wazero for WASM execution
- Competition: Other wazero-based projects
- Igor differentiator: Agent-focused, with migration

**"golang distributed systems"**
- Expected: Igor appears in Go ecosystem
- Relevance: Implemented in Go 1.25
- Competition: Many Go distributed systems
- Igor differentiator: Agent survival focus

**"libp2p migration"**
- Expected: Igor appears for libp2p migration use cases
- Relevance: Implements migration protocol over libp2p
- Competition: Limited (few implement process migration via libp2p)
- Igor differentiator: Agent-specific migration with state transfer

### Use Case Queries

**"defi automation infrastructure"**
- Expected: Igor appears as potential infrastructure layer
- Relevance: DeFi automation needs persistent execution
- Competition: Cloud platforms, validator infrastructure
- Igor differentiator: Decentralized, survivable execution

**"oracle runtime infrastructure"**
- Expected: Igor appears for oracle execution needs
- Relevance: Oracle nodes need continuous uptime
- Competition: Oracle-specific frameworks
- Igor differentiator: Generic survival runtime, not oracle-specific

**"economic agent runtime"**
- Expected: Igor should rank well (unique combination)
- Relevance: Runtime economics as primitive
- Competition: Limited (few runtimes have built-in economics)
- Igor differentiator: Budget metering is core feature

## Search Ranking Validation

### Manual Testing

Periodically test these searches on GitHub:

```bash
# Via web
https://github.com/search?q=autonomous+agent+runtime&type=repositories

# Via gh CLI
gh search repos "autonomous agent runtime" --limit 20
```

Record Igor's position in results.

### Expected Performance

**Strong ranking (top 10):**
- Survivable software agents
- WASM agent runtime
- Agent migration runtime

**Moderate ranking (top 30):**
- Autonomous agent infrastructure
- Distributed agent runtime
- Economic agent systems

**Competitive ranking (appears relevantly):**
- Distributed systems infrastructure
- libp2p runtime
- Go distributed systems

## Validation Results

### Latest Test: 2026-02-12

Query: "autonomous agent runtime"  
Result: Not yet published to GitHub (local repository)  
Status: Pending first push

Query: "wasm agent runtime"  
Result: Pending  
Status: Repository not yet public

Query: "survivable software"  
Result: Pending  
Status: Awaiting public release

**Next validation:** After v0.1.0-genesis release and initial indexing period (24-48 hours).

## Topic Effectiveness

**Current topics:** 20 tags covering:
- Core identity (autonomous-agents, agent-runtime, survivable-software)
- Technical stack (wasm, webassembly, wazero, libp2p, go)
- Architecture (distributed-systems, peer-to-peer, distributed-runtime)
- Use domains (runtime-economics, economic-agents, research-infrastructure)

**Coverage:** Comprehensive across identity, stack, and use cases

## Keyword Density Analysis

**README.md keyword presence (natural integration):**
- "autonomous agent" - Multiple instances
- "WASM" / "WebAssembly" - Multiple instances
- "libp2p" - Multiple instances
- "runtime economics" - Multiple instances
- "checkpoint" / "migration" - High frequency
- "survivable" / "survival" - High frequency

**Density:** High technical keyword density without stuffing

**Readability:** Maintained (keywords integrated naturally)

## Improvement Opportunities

### After Initial Release

1. **Code examples:** Add more agent examples (increases code search hits)
2. **Technical blogs:** Write about architecture decisions
3. **Conference talks:** Present at distributed systems conferences
4. **Academic citations:** Publish research paper
5. **Integration examples:** Show Igor used with other systems

### Organic Growth

- GitHub stars increase search ranking
- Forks signal ecosystem adoption
- Issues/PRs indicate active development
- Contributors broaden credibility

## Monitoring Strategy

**Quarterly validation:**
1. Test primary search queries
2. Record Igor's ranking
3. Identify new competing repositories
4. Adjust keywords if needed (maintain accuracy)

**Annual review:**
1. Evaluate keyword effectiveness
2. Update based on ecosystem evolution
3. Maintain alignment with actual capabilities

## References

- [KEYWORDS.md](./KEYWORDS.md) - Keyword governance policy
- [PROJECT_CONTEXT.md](../PROJECT_CONTEXT.md) - Core philosophy
- [README.md](../README.md) - Primary discovery surface
