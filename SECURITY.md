# Security Policy

## Project Status

Igor v0 is experimental research software currently in Phase 2 (Survival) development. It is **not production-ready** and has known security limitations by design.

**Do not deploy Igor v0 on public networks or with sensitive data.**

## Supported Versions

Only the `main` branch is actively maintained. No stable releases have been tagged yet.

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| v0.x    | :construction:     |

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

## Technical Security Documentation

For detailed threat analysis and security mechanisms:

- [docs/runtime/THREAT_MODEL.md](./docs/runtime/THREAT_MODEL.md) - Threat assumptions, adversary classes, trust boundaries
- [docs/runtime/SECURITY_MODEL.md](./docs/runtime/SECURITY_MODEL.md) - Current security mechanisms and limitations
- [docs/enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md](./docs/enforcement/RUNTIME_ENFORCEMENT_INVARIANTS.md) - System guarantees

## Contact

For non-sensitive questions about Igor security:
- Open a GitHub issue
- Use "security" label
- Public discussion welcome

For sensitive vulnerability reports:
- Use GitHub Security Advisory
- Private disclosure until patched
