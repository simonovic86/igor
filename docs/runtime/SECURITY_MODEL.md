# Security Model

## Overview

Igor v0 implements a **minimal viable security model** focused on sandbox isolation and basic trust boundaries. Perfect security is explicitly not required in v0.

## Threat Model

### What Igor Protects Against

1. **Malicious Agents**
   - Cannot escape WASM sandbox
   - Cannot access host filesystem
   - Cannot make network calls
   - Cannot consume unlimited resources

2. **Resource Exhaustion**
   - Memory capped at 64MB per agent
   - Execution timeout at 100ms per tick
   - Budget enforcement prevents infinite execution

### What Igor Does NOT Protect Against

1. **Malicious Nodes**
   - Nodes can lie about execution time
   - Nodes can steal agent state
   - Nodes can refuse to forward migrations
   - Nodes can ignore budget limits

2. **Network Attacks**
   - No authentication between peers
   - No transport encryption (relies on libp2p)
   - No DoS protection
   - No rate limiting

3. **Data Integrity**
   - No cryptographic verification of checkpoints
   - No proof of execution
   - No tamper detection

**Assumption:** Igor v0 operates in semi-trusted environments (development, research, friendly networks).

## Sandbox Boundaries

### WASM Sandbox (wazero)

**Enforced by runtime:**

```go
config := wazero.NewRuntimeConfig().
    WithMemoryLimitPages(1024).  // 64MB limit
    WithCloseOnContextDone(true)
```

**Capabilities disabled:**
- Filesystem access (read/write)
- Network sockets
- System calls beyond WASI
- Raw memory access outside linear memory

**Capabilities enabled:**
- Linear memory (64MB max)
- Stack operations
- Local variables
- Stdout/stderr (logged)

### Execution Limits

**Tick Timeout:**
```go
tickCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
defer cancel()
```

**Result:**
- Agent tick must complete in 100ms
- Context cancellation if exceeded
- Prevents infinite loops
- Prevents CPU hogging

**Memory Limit:**
```
1024 pages × 64KB/page = 64MB
```

**Result:**
- Agent cannot allocate beyond 64MB
- Out-of-memory panics caught by runtime
- Prevents memory exhaustion

## Trust Boundaries

### Agents Trust Nodes For:

- Executing code correctly
- Metering honestly
- Facilitating migration
- Preserving checkpoints temporarily

**Mitigation (future):**
- Cryptographic receipts
- Redundant execution
- Checkpoint verification

### Nodes Trust Agents For:

- Paying for execution (budget enforcement)
- Not attacking runtime (sandbox enforcement)
- Cooperating with migration (lifecycle compliance)

**Mitigation (current):**
- WASM sandbox isolation
- Resource limits
- Timeout enforcement

### Peers Do Not Trust Each Other

- No peer authentication (relies on libp2p peer ID)
- No transport-level encryption (libp2p provides this)
- No reputation system
- No coordination requirements

## Isolation Mechanisms

### Process Isolation

Each Igor node runs in its own process:
- Separate memory space
- Separate filesystem namespace
- No shared state between nodes

### Agent Isolation

Agents are isolated from:
- Host filesystem
- Host network
- Other agents
- Node internals

Agents communicate only through:
- WASM exports (lifecycle functions)
- Stdout/stderr (logged)

### State Isolation

Each agent has:
- Private WASM linear memory
- Private checkpoint storage
- Private budget accounting
- No shared state

## Attack Vectors

### Agent → Node Attacks

**Possible:**
- CPU consumption (mitigated: 100ms timeout)
- Memory exhaustion (mitigated: 64MB limit)
- Budget depletion (intended behavior)

**Not possible:**
- Filesystem access (disabled)
- Network access (disabled)
- Escape sandbox (WASM isolation)
- Affect other agents (process isolation)

### Node → Agent Attacks

**Possible:**
- Steal agent state (node has full access)
- Lie about metering (no verification)
- Refuse migration (no enforcement)
- Corrupt checkpoint (no integrity check)

**Mitigation (v0):**
- Choose trusted nodes
- Verify migration manually
- Check logs

**Mitigation (future):**
- Cryptographic receipts
- State encryption
- Multi-node verification

### Peer → Peer Attacks

**Possible:**
- Send malformed messages (gracefully rejected)
- Flood with connections (no rate limiting)
- Refuse cooperation (no penalties)

**Mitigation (v0):**
- Robust error handling
- Close malformed streams
- Continue operation

**Mitigation (future):**
- Peer reputation
- Rate limiting
- Connection throttling

## Checkpoint Security

### Current Implementation

Checkpoints are **not secured**:

- Stored in plaintext
- No encryption
- No integrity verification
- No authentication

**Format:**
```
[budget:8][price:8][state:N]
```

Anyone with filesystem access can:
- Read agent state
- Modify budget
- Tamper with state
- Delete checkpoints

### Future Improvements

Potential enhancements:

1. **Encryption**
   - Encrypt state with agent's key
   - Only agent can decrypt
   - Nodes cannot read state

2. **Integrity**
   - Hash or sign checkpoint
   - Verify on load
   - Detect tampering

3. **Authentication**
   - Agent signs checkpoint
   - Node verifies signature
   - Prevents checkpoint injection

**Not implemented in v0.**

## Budget Security

### Current Model

Budget is **trusted accounting**:

- Nodes meter honestly (assumed)
- No cryptographic proofs
- No external verification
- No audit trail

**Vulnerabilities:**
- Node can overcharge
- Node can undercharge
- No proof of consumption
- No dispute resolution

### Future Improvements

1. **Cryptographic Receipts**
   - Node signs execution time
   - Agent verifies signature
   - Third-party auditable

2. **Payment Channels**
   - Off-chain micropayments
   - Periodic settlement
   - Fraud proofs

3. **Multi-Node Verification**
   - Multiple nodes execute same agent
   - Compare results
   - Detect dishonest metering

**Not implemented in v0.**

## Identity and Authentication

### Current Implementation

**Node Identity:**
- libp2p peer ID (cryptographic)
- Derived from keypair
- Not persistent across restarts

**Agent Identity:**
- String AgentID only
- No cryptographic identity
- No signatures
- No access control

### What's Missing

Agents do not have:
- Cryptographic keypair
- Signing capability
- Identity verification
- Access control lists

This is a known limitation of v0.

## Network Security

### libp2p Security

Igor inherits libp2p's security properties:

- **Transport encryption** - Noise protocol by default
- **Peer authentication** - Peer IDs are cryptographic
- **Protocol negotiation** - Multistream-select

### What's NOT Protected

- **No authorization** - Any peer can migrate agents
- **No rate limiting** - Peers can flood requests
- **No DoS protection** - Resource exhaustion possible

## Operational Security

### Running Igor v0 Safely

**Development/Testing:**
- Run on isolated networks
- Use trusted peers only
- Monitor resource consumption
- Check logs frequently

**Not Recommended:**
- Public internet deployment
- Production workloads
- Financial transactions
- Sensitive data processing

### Security Checklist

Before deploying Igor v0:

- [ ] Trusted network environment?
- [ ] Known peer nodes?
- [ ] Non-sensitive agent state?
- [ ] Acceptable data loss risk?
- [ ] Monitoring in place?

If any answer is "No", **do not deploy**.

## Vulnerability Disclosure

Igor v0 is **experimental software** with known security limitations.

Expected vulnerabilities:
- Budget manipulation
- State tampering
- Checkpoint corruption
- Resource exhaustion
- Network flooding

**This is acceptable for v0.** The goal is to prove autonomous agent survival, not to build a production system.

## Future Security Roadmap

Potential improvements beyond v0:

**Phase 3 (Autonomy):**
- Agent manifest validation
- Capability enforcement
- Basic integrity checks

**Phase 4 (Economics):**
- Payment receipt signing
- Cryptographic proofs
- Basic fraud detection

**Phase 5 (Hardening):**
- State encryption
- Checkpoint signing
- Multi-party verification
- Advanced sandbox hardening

**Not committed.** Listed only as possibilities.

## Security Philosophy

Igor v0 follows a **pragmatic security approach**:

1. **Sandbox first** - Prevent agent escape
2. **Isolate resources** - Limit damage from bugs
3. **Fail safely** - Errors terminate agents, not nodes
4. **Log everything** - Auditable behavior
5. **Accept limitations** - Don't pretend v0 is production-ready

**Quote from PROJECT_CONTEXT.md:**

> "Perfect security is NOT required in v0."

The focus is on proving the autonomous agent model works, not on production-grade security.
