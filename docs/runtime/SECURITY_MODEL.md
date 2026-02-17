# Security Model

## Overview

Igor v0 implements a **minimal viable security model** focused on sandbox isolation, runtime resource bounding, and explicit limitation disclosure. Perfect security is explicitly not required in v0.

## Canonical Threat Assumptions

See [THREAT_MODEL.md](./THREAT_MODEL.md) for canonical threat assumptions, including:

- system model and failure classes
- adversary classes (A1-A4)
- network assumptions
- trust assumptions
- security goals and non-goals

This document focuses on **current mechanisms** and **current limitations** under those assumptions.

## Current Protection Scope

### What Igor v0 Actively Enforces

1. **Agent containment**
   - WASM sandbox isolation via wazero
   - No direct host filesystem/network capability in guest code

2. **Resource bounding**
   - Memory capped per agent
   - Tick timeout enforced
   - Budget-gated execution

3. **Operational fail-fast behavior**
   - Runtime surfaces errors directly
   - Agent termination on unrecoverable execution failures

### What Igor v0 Does Not Defend Against

1. **Malicious node resistance**
   - Nodes may lie about metering
   - Nodes may inspect/tamper with plaintext state they host
   - Nodes may refuse migration cooperation

2. **Strong economic integrity**
   - No fraud-proof metering
   - No cryptographic payment verification
   - No dispute resolution rails

3. **Hostile-public-network hardening**
   - No robust DoS/rate-limit framework
   - No reputation- or policy-based peer exclusion framework
   - No application-layer authorization model beyond current protocol checks

## Sandbox Boundaries

### WASM Sandbox (wazero)

**Runtime constraints in v0:**

```go
config := wazero.NewRuntimeConfig().
    WithMemoryLimitPages(1024).  // 64MB limit
    WithCloseOnContextDone(true)
```

**Guest capabilities effectively unavailable by default runtime integration:**
- Host filesystem access
- Arbitrary host networking
- Host process memory access

**Guest capabilities used:**
- WASM linear memory
- Lifecycle export invocation
- Stdout/stderr output (captured by runtime logging)

## Execution Limits

### Tick Timeout

```go
tickCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
defer cancel()
```

### Memory Limit

```
1024 pages × 64KB/page = 64MB
```

### Budget Gate

Execution is bounded by budget exhaustion behavior at runtime.

## Trust Boundaries

### Local Trust Domain

Locally, the runtime assumes:
- node process control flow is not actively subverted
- local operator controls deployment/configuration
- local persistence behaves according to local failure classes

### Untrusted Domain

The runtime does **not** assume trust in:
- remote peers
- remote authority claims
- remote metering claims
- remote checkpoint custody
- network timing/order behavior

## Isolation Mechanisms

### Process Isolation

Each node runs as a separate process with its own local resources.

### Agent Isolation

Each agent executes in isolated WASM module context with bounded resources.

### State Isolation

Checkpoint files are separated by agent ID pathing at storage layer.

## Attack Surface and Current Limits

### Agent -> Node

**Mitigated in v0:**
- unbounded CPU loop behavior (timeout)
- unbounded guest memory growth (memory cap)
- direct host capability use (sandbox boundary)

**Residual risk:**
- high churn workload patterns that remain within configured bounds

### Node -> Agent

**Not mitigated in v0:**
- state inspection by host
- dishonest metering
- refusal to cooperate in migration
- local checkpoint tampering

### Peer -> Peer

**Partially handled in v0:**
- malformed payload rejection in handlers

**Not robustly handled in v0:**
- flooding/spam/resource pressure
- strategic liveness griefing

## Checkpoint Security

### Current State

Checkpoints are plaintext and host-visible:

- no encryption
- no cryptographic integrity proof
- no authenticated origin proof

This means host-level actors can read/alter/delete local checkpoint files.

### Directional Future Work

Potential directions (not committed by this document):
- state encryption
- checkpoint integrity signing
- authenticated checkpoint provenance

## Budget Security

### Current State

Budget accounting is trusted runtime accounting:

- no cryptographic receipts in v0
- no independent verification
- no built-in fraud proofing

### Directional Future Work

Potential directions (not committed by this document):
- signed execution receipts
- verifiable accounting proofs
- external settlement/dispute integration

## Identity and Authentication

### Current State

**Node identity:**
- libp2p peer identity primitives are used by transport layer

**Agent identity:**
- runtime agent IDs are string identifiers in current implementation
- no full cryptographic identity lifecycle enforcement in v0

## Network Security

### Transport Baseline

Igor relies on libp2p transport defaults for channel security properties provided by that stack.

### Remaining Gaps

v0 does not provide a complete application-layer security policy framework for:
- authorization
- anti-DoS controls
- anti-replay economics

## Operational Security Guidance

### Appropriate Environments

- local development
- research/test networks
- trusted operator environments

### Not Recommended

- hostile public internet operation
- production-critical workloads
- sensitive data handling
- value-critical financial operations

## Vulnerability Disclosure

Igor v0 is experimental software with known limitations. Security reports should follow the repository security policy in [SECURITY.md](../../SECURITY.md).

## Future Security Roadmap (Directional)

### Phase 3 (Autonomy)
- manifest/capability policy surfaces

### Phase 4 (Economics)
- receipt/signing-oriented economics controls

### Phase 5 (Hardening)
- stronger isolation
- failure-recovery hardening
- agent integrity/identity verification

These are roadmap directions, not guarantees.

## Security Philosophy

Igor v0 prioritizes:

1. containment over complexity
2. explicit limits over implied guarantees
3. fail-stop safety behavior over optimistic liveness
4. honest disclosure over production claims

From [PROJECT_CONTEXT.md](../../PROJECT_CONTEXT.md):

> "Perfect security is NOT required in v0."

