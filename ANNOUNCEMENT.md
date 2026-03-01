# Introducing Igor: A Runtime for Survivable Autonomous Agents

Software today can execute autonomously but cannot survive autonomously. Igor addresses this gap.

## Why Igor Exists

Autonomous economic software—trading strategies, oracle networks, AI service agents—operates independently but remains existentially tied to specific infrastructure. When hosting servers fail, these systems stop. When operators disengage, they vanish. The software can manage capital and make decisions, but it cannot ensure its own survival.

This dependency creates fragility for systems that must persist continuously. A DeFi strategy cannot "redeploy" without losing position context. An oracle participant cannot restart without losing reputation. An AI agent cannot reset without losing client relationships.

Igor provides the missing capability: agents that checkpoint their own state, migrate when infrastructure fails, and pay for execution from internal budgets. Survival becomes a property of the software itself, not of operational practices surrounding it.

## What Igor Introduces

Igor implements three primitives not present in existing infrastructure:

**Survival through explicit checkpointing:** Agents serialize state at regular intervals. When infrastructure fails or migration occurs, agents resume from the last checkpoint. State persists independently of the hosting node.

**Migration over peer-to-peer networks:** Agents package their WASM code, state, and budget, then transfer to target nodes via libp2p streams. The target resumes the agent and confirms success. The source then terminates. Migration is protocol-supported, not operator-managed.

**Runtime economic metering:** Agents carry budgets. Every tick of execution is metered, its cost calculated as `duration × price_per_second`, and deducted from budget. When budget exhausts, agents terminate gracefully. Economics are enforced by the runtime, not external systems.

These primitives combine to enable software that persists across infrastructure changes: agents run on Node A, checkpoint state, migrate to Node B, resume execution, and continue operating until their budget exhausts or they decide to stop.

## Technical Implementation

Igor v0 is implemented in Go using proven components:

- **WASM sandbox:** wazero provides isolated execution (64MB memory, no filesystem/network access)
- **P2P transport:** libp2p handles peer discovery and agent transfer
- **Checkpointing:** Atomic writes (temp file → fsync → rename) guarantee consistency
- **Budget tracking:** Nanosecond-precision timing, immediate cost deduction

The implementation is minimal: ~3,000 lines of runtime code, ~200 lines for the example agent. This validates that survival primitives can be realized without architectural complexity.

## Current Capabilities (Phase 2)

Igor v0 demonstrates:

- Agents execute in WASM sandboxes with deterministic behavior
- Agents checkpoint state and resume after restart
- Agents migrate between nodes preserving state and budget
- Agents pay for execution time from internal budgets
- Agents terminate gracefully when budget exhausts
- Nodes coordinate peer-to-peer without centralized authority

All 6 success criteria from the design specification are met. Phase 2 proves that autonomous agent survival is feasible.

## Project Maturity: Experimental

Igor v0 is research-stage software. It validates concepts, not production deployments.

**Known limitations:**
- Trusted runtime accounting (nodes self-report honestly)
- Single-hop migration (no routing or relay)
- Local filesystem storage (no distributed checkpoints)
- Limited security model (sandbox only, no encryption)
- No agent discovery or coordination protocols

Do not use Igor for:
- Production workloads
- Public network deployments
- Sensitive data processing
- Financial transactions

Igor is suitable for:
- Research and experimentation
- Trusted network environments
- Understanding autonomous agent architecture
- Validating survival-centric design patterns

## Why This Matters Now

Three technological capabilities mature simultaneously:

WASM has evolved from browser sandbox to production-ready portable execution format. Runtimes are stable, performant, and widely adopted.

libp2p has been proven at scale through IPFS and Filecoin deployments. It handles millions of nodes and provides production-grade decentralized networking.

Programmable capital through DeFi demonstrates that software can autonomously manage billions in assets. This creates demand for infrastructure enabling economically self-sustaining agents.

These technologies existed separately. Igor combines them into a runtime for autonomous agent survival.

## Invitation

Igor is an experiment in distributed systems infrastructure. We invite technical collaboration from engineers, researchers, and architects interested in:

- Autonomous software persistence
- Decentralized execution infrastructure
- Runtime economic models
- Mobile agent architectures
- Survival-centric system design

Contributions welcome in:
- Implementation improvements
- Security analysis
- Documentation
- Example agents
- Performance profiling
- Bug reports

See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## What Happens Next

Igor v0 completes Phase 2 (Survival). Immediate priorities:

- Extended stability testing
- Community feedback integration
- Security review and hardening
- Performance characterization

Future phases (not committed):
- Phase 3: Agent autonomy (capability enforcement, migration decisions)
- Phase 4: Economic verification (payment receipts, cryptographic proofs)
- Phase 5: Production hardening (security audit, failure recovery)

Development remains minimal and iterative. Features are added only when validated as necessary.

## Repository

**GitHub:** https://github.com/simonovic86/igor  
**Documentation:** [docs/](./docs/)  
**Specification:** [docs/philosophy/OVERVIEW.md](./docs/philosophy/OVERVIEW.md)

---

Igor establishes a baseline for runtime infrastructure enabling autonomous software survival. Whether this capability proves valuable can only be determined through real-world deployment and testing.

We invite technically-minded collaborators to explore this question with us.
