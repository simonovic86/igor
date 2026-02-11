# Security Policy

## Project Status

Igor v0 is experimental research software currently in Phase 1 (Survival) development. It is **not production-ready** and has known security limitations by design.

**Current Status:** Early development, active testing, not suitable for production use.

## Supported Versions

Only the `main` branch is actively maintained. No stable releases have been tagged yet.

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| v0.x    | :construction:     |

## Security Scope

Igor includes security-sensitive components:

### WASM Sandbox Execution
- Agent isolation via wazero runtime
- Memory limits (64MB per agent)
- Capability restrictions (no filesystem, no network)
- Execution timeouts (100ms per tick)

### Agent Migration Transport
- Agent package transfer over libp2p streams
- Checkpoint serialization and transmission
- Confirmation handshake protocol

### Checkpoint Persistence
- State serialization and storage
- Budget metadata encoding
- Atomic write operations

### Runtime Metering Logic
- Execution time measurement
- Budget calculation and enforcement
- Cost deduction mechanisms

## Known Limitations

Igor v0 has intentional security limitations:

**Budget Security:**
- Budget accounting is trusted, not cryptographically verified
- Nodes can lie about execution time
- No payment receipts or proof of work
- No dispute resolution mechanism

**Checkpoint Security:**
- Checkpoints stored in plaintext
- No encryption of agent state
- No integrity verification
- No authentication of checkpoint origin

**Network Security:**
- No peer authentication beyond libp2p peer ID
- No authorization for migration requests
- No rate limiting or DoS protection
- Relies on trusted network environment

**Identity Security:**
- Agents lack cryptographic identity in v0
- No signing capability
- No access control
- Identity not carried in migrations

These limitations are documented in [docs/SECURITY_MODEL.md](./docs/SECURITY_MODEL.md).

**Do not deploy Igor v0 on public networks or with sensitive data.**

## Reporting Vulnerabilities

If you discover a security vulnerability in Igor, please report it responsibly.

### Preferred Method: GitHub Security Advisory

1. Go to https://github.com/simonovic86/igor/security/advisories
2. Click "Report a vulnerability"
3. Provide detailed description
4. Allow time for assessment and patch development

### Alternative Method: Email

Email security issues to: `security@igor-project.org` (placeholder - use GitHub issues for now)

### What to Include

- Description of vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if applicable)
- Your contact information (for coordination)

### What to Report

**Critical:**
- WASM sandbox escapes
- Arbitrary code execution on host
- Memory corruption or buffer overflows
- Resource exhaustion attacks that crash nodes
- Migration protocol attacks that create duplicate agents
- Checkpoint corruption that breaks recovery

**Important:**
- Denial of service vulnerabilities
- Information disclosure
- Privilege escalation within sandbox
- Budget manipulation exploits
- Migration abuse scenarios

**Lower Priority (but still valuable):**
- Logging security issues
- Configuration weaknesses
- Documentation improvements

## Disclosure Policy

We follow coordinated disclosure:

1. **Report received** - We acknowledge within 48 hours
2. **Assessment** - We evaluate severity and impact
3. **Fix development** - We develop and test a patch
4. **Disclosure timeline** - We agree on public disclosure date
5. **Patch release** - We release fix to main branch
6. **Public disclosure** - We publish security advisory

Typical timeline: 30-90 days from report to disclosure, depending on severity.

## Security Expectations

### What Igor v0 Protects Against

- **Malicious agents** escaping WASM sandbox
- **Resource exhaustion** by single agent (memory, CPU)
- **State corruption** through atomic checkpoint writes

### What Igor v0 Does NOT Protect Against

- **Malicious nodes** lying about metering or stealing state
- **Network attacks** beyond basic libp2p security
- **Cryptographic vulnerabilities** (minimal crypto in v0)
- **Economic attacks** (no payment verification)
- **Data privacy** (checkpoints in plaintext)

Igor v0 is suitable only for:
- Development and testing
- Research environments
- Trusted network deployments
- Non-sensitive agent workloads

## Security Roadmap

Future phases may address current limitations:

**Phase 2 (Autonomy):**
- Agent manifest validation
- Capability enforcement
- Basic integrity checks

**Phase 3 (Economics):**
- Cryptographic receipts
- Payment verification
- Fraud detection

**Phase 4 (Hardening):**
- State encryption
- Checkpoint signing
- Advanced sandbox hardening
- Multi-party verification

No timeline or commitment. Listed for context only.

## Security Resources

- [docs/SECURITY_MODEL.md](./docs/SECURITY_MODEL.md) - Detailed threat model
- [docs/INVARIANTS.md](./docs/INVARIANTS.md) - System guarantees
- [PROJECT_CONTEXT.md](./PROJECT_CONTEXT.md) - Design philosophy

## Contact

For non-sensitive questions about Igor security:
- Open a GitHub issue
- Use "security" label
- Public discussion welcome

For sensitive vulnerability reports:
- Use GitHub Security Advisory
- Private disclosure until patched
