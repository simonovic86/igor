# Genesis Commit Message

This document contains the canonical commit message for Igor's public genesis commit.

---

## Commit Message

```
feat: Igor v0.1.0-genesis - Runtime for Autonomous Economic Agents

Igor is a decentralized runtime enabling software agents to survive
infrastructure failure, migrate between peer nodes, and pay for their
own execution.

WHAT IGOR IS

Igor provides three primitives absent from existing infrastructure:

1. Survival: Agents checkpoint state and resume after node failure
2. Migration: Agents transfer between nodes over P2P streams (libp2p)
3. Economics: Agents pay for execution time from internal budgets

Implementation:

- WASM sandbox via wazero (64MB limit, no filesystem/network)
- Peer-to-peer migration via libp2p streams
- Runtime budget metering: cost = duration × price_per_second
- Atomic checkpoint persistence (budget + state)
- Single-instance invariant during migration

WHY IGOR EXISTS

Autonomous economic software—DeFi strategies, oracle participants,
AI service agents—can execute decisions autonomously but cannot
survive infrastructure failure autonomously.

These systems hold capital, operate continuously, and must persist
beyond operator attention spans. Current infrastructure couples
software survival to specific deployment locations. Igor decouples
survival from infrastructure by making agents responsible for their
own persistence through checkpointing and migration.

WHAT IGOR IS NOT

Igor is NOT:

- An AI reasoning framework
- A multi-agent coordination system
- A blockchain or consensus protocol
- An agent marketplace or discovery system
- A general-purpose orchestration platform

Igor is a minimal survival runtime. It implements checkpointing,
migration, and budget enforcement. Nothing more.

MATURITY LEVEL: EXPERIMENTAL

Igor v0 is research-stage software proving that autonomous agent
survival is technically feasible. It is NOT production-ready.

Known limitations:

- Trusted runtime accounting (no cryptographic receipts)
- Single-hop migration only (no routing)
- Local filesystem storage only
- No agent discovery protocol
- Limited security model

Do not deploy on public networks or with sensitive data.

ARCHITECTURE GUARANTEES

Igor maintains strict invariants:

- Single active instance per agent (no split-brain)
- Budget conservation (never created/destroyed)
- State persistence (survives shutdown/migration)
- Atomic checkpoints (all-or-nothing writes)
- Tick timeout enforcement (100ms max)

PHASE 1 COMPLETE

All 6 success criteria from PROJECT_CONTEXT.md are met:

1. ✓ Agent can run on Node A
2. ✓ Agent can checkpoint state explicitly
3. ✓ Agent can migrate to Node B
4. ✓ Agent resumes from checkpoint
5. ✓ Agent pays runtime rent
6. ✓ No centralized coordination

REPOSITORY STRUCTURE

cmd/igord/              # Node runtime
internal/agent/         # Agent lifecycle
internal/runtime/       # WASM engine (wazero)
internal/migration/     # P2P migration
internal/storage/       # Checkpoint persistence
internal/p2p/           # libp2p networking
pkg/protocol/           # Migration messages
agents/research/example/         # Counter agent (TinyGo)
docs/                   # Technical documentation

TECHNICAL STACK

- Runtime: Go 1.25.4
- WASM: wazero (pure Go sandbox)
- P2P: libp2p-go
- Agents: TinyGo → WASM

DEVELOPMENT PHILOSOPHY

From PROJECT_CONTEXT.md:

- Deterministic behavior preferred
- Explicit state over implicit memory
- Small testable increments
- Avoid feature creep
- Fail loudly on invariant violations

This commit establishes Igor as experimental infrastructure for
autonomous software survival research.
```

---

## Usage

When creating the genesis commit, use the message above verbatim or adapted as needed.

The message establishes:
- Technical scope
- Architectural guarantees
- Explicit limitations
- Experimental disclaimer
- Project philosophy

Do not modify runtime code after this baseline.
