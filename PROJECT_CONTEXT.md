# Igor v0 — Autonomous Mobile Agent Runtime

## Vision

Igor v0 is an experimental distributed systems runtime that enables
portable autonomous agents to migrate between peer nodes and execute
independently.

The goal is to implement true execution autonomy:
- agents own identity
- agents own state
- agents own budget
- agents decide where to run
- agents survive infrastructure churn

Igor v0 is intentionally minimal. It is a proof-of-survival runtime.

---

## Non-Goals (Strict)

Igor v0 must NOT implement:

- agent marketplaces
- reputation systems
- staking or token economics
- AI / LLM functionality
- multi-agent coordination frameworks
- advanced security systems
- distributed consensus

These are explicitly out of scope.

---

## Success Criteria

Igor v0 is complete only if:

1. Agent can run on Node A
2. Agent can checkpoint state explicitly
3. Agent can migrate to Node B
4. Agent resumes execution from checkpoint
5. Agent pays runtime rent to hosting node
6. System has no centralized coordination

---

## Architecture Overview

### Node Runtime (`igord`)

Each node is a peer in a P2P network responsible for:

- peer discovery
- agent sandbox execution
- runtime resource metering
- migration transport
- payment accounting

Nodes are untrusted infrastructure providers.

---

### Agent

Agents are portable execution packages composed of:

- WASM executable
- explicit serialized state
- manifest metadata
- cryptographic identity

Agents must be migration-cooperative.

---

### Agent Lifecycle

Agents must implement:

- init()
- tick()
- checkpoint()
- resume(state)

tick() must be short and resumable.

Agents may request migration but cannot perform it directly.

---

## Technology Stack

Runtime Language:
- Go

P2P Transport:
- libp2p-go

Sandbox:
- wazero

Agent Compilation:
- TinyGo → WASM

---

## Security Model (MVP)

Nodes must:
- enforce WASM sandbox
- enforce resource limits
- meter execution time

Agents must:
- carry identity keypair
- explicitly serialize state
- assume host is untrusted

Perfect security is NOT required in v0.

---

## Payment Model (MVP)

Nodes charge agents based on:

runtime_seconds × price_per_second

Payments are implemented using signed receipts.

No blockchain integration in v0.

---

## Migration Model

Migration must follow:

1. Agent requests migration
2. Node finds candidate peers
3. Target peer accepts
4. Node checkpoints agent
5. Agent package transferred
6. Target resumes agent
7. Source deletes instance

At most one active instance must exist.

---

## Development Philosophy

- deterministic behavior preferred
- explicit state over implicit memory
- small testable increments
- avoid feature creep
- fail loudly on invariant violations
