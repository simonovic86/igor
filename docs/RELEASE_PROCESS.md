# Release Process

## Overview

This document defines the release process for Igor v0. As an experimental project in early development, the release process is lightweight and focused on clear versioning and communication.

**Current Status:** Pre-release. No official versions tagged yet.

## Versioning Strategy

Igor uses **semantic versioning**: `MAJOR.MINOR.PATCH`

### Version Semantics

**MAJOR (0.x.x):**
- Breaking changes to APIs or protocols
- Checkpoint format changes
- Migration protocol changes
- Not backward compatible

**MINOR (x.Y.x):**
- New features
- New capabilities
- Backward compatible additions

**PATCH (x.x.Z):**
- Bug fixes
- Documentation updates
- Performance improvements
- No API changes

### Pre-1.0 Semantics

While in 0.x versions:
- Breaking changes allowed in MINOR versions
- API stability not guaranteed
- Migration between versions may require manual intervention

**v1.0.0** will signify:
- API stability commitment
- Production readiness (if achieved)
- Backward compatibility guarantees

## Tagging Format

**Tags:** `vX.Y.Z`

Examples:
- `v0.1.0` - First tagged release
- `v0.2.0` - Added capability enforcement
- `v0.2.1` - Bug fix for migration protocol
- `v1.0.0` - First stable release (future)

**Annotated tags required:**
```bash
git tag -a v0.1.0 -m "Release v0.1.0: Phase 2 (Survival) complete"
```

## Release Criteria

### Pre-Release Checklist

Before tagging a release:

- [ ] All Phase N tasks complete (per TASKS.md)
- [ ] Documentation updated and accurate
- [ ] `make check` passes (lint, vet, fmt-check)
- [ ] `make test` passes all tests
- [ ] `make build` produces working binary
- [ ] Example agent compiles and runs
- [ ] Migration demonstrated successfully
- [ ] No known critical bugs
- [ ] CHANGELOG.md updated (future)

### Release Readiness Gates

**v0.1.0 Requirements:**
- Phase 2 complete
- Core survival capabilities validated
- Documentation comprehensive
- Basic stability demonstrated

**v0.2.0 Requirements:**
- Phase 3 complete
- Agent autonomy implemented
- Multi-node testing successful

**v1.0.0 Requirements:**
- All phases complete
- Security audit passed
- Production deployments successful
- Backward compatibility commitment ready

## Release Flow

### 1. Preparation

**Update documentation:**
```bash
# Review and update
vim README.md
vim docs/ARCHITECTURE.md
vim TASKS.md
```

**Ensure clean state:**
```bash
make clean
make all  # Build, test, and check everything
```

### 2. Version Decision

Determine version bump:
- Breaking changes? → MAJOR
- New features? → MINOR
- Bug fixes only? → PATCH

### 3. Changelog Update (Future)

Update CHANGELOG.md:

```markdown
## [0.2.0] - 2026-03-15

### Added
- Agent capability validation and enforcement
- Multi-node migration optimization
- Dynamic infrastructure selection

### Fixed
- Migration timeout handling
- Checkpoint race condition

### Changed
- Migration protocol format (breaking)
```

Follow [Keep a Changelog](https://keepachangelog.com/) format.

### 4. Create Annotated Tag

```bash
# Tag release
git tag -a v0.1.0 -m "Release v0.1.0: Phase 2 (Survival) complete

- WASM sandbox runtime
- Agent checkpointing and resume
- P2P migration protocol
- Runtime budget metering
- Decentralized node network"

# Verify tag
git tag -l -n9 v0.1.0

# Push tag
git push origin v0.1.0
```

### 5. Build Release Artifacts (Future)

Build for multiple platforms:

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build -o bin/igord-linux-amd64 ./cmd/igord

# macOS amd64
GOOS=darwin GOARCH=amd64 go build -o bin/igord-darwin-amd64 ./cmd/igord

# macOS arm64
GOOS=darwin GOARCH=arm64 go build -o bin/igord-darwin-arm64 ./cmd/igord
```

Generate checksums:
```bash
shasum -a 256 bin/igord-* > bin/checksums.txt
```

### 6. Publish GitHub Release

Create release on GitHub:

1. Go to Releases page
2. Click "Draft a new release"
3. Select tag: `v0.1.0`
4. Title: `Igor v0.1.0 - Phase 2 (Survival)`
5. Description: Copy from CHANGELOG.md
6. Attach binaries and checksums
7. Mark as pre-release if v0.x
8. Publish release

### 7. Announce

Post announcement:
- GitHub Discussions (if enabled)
- Project website (if exists)
- Community channels (if established)

## Release Notes Format

### Title

```
Igor v0.X.Y - [Phase Name or Theme]
```

### Structure

```markdown
## Highlights

[2-3 sentences describing major changes]

## What's New

### Features
- Feature 1 with brief description
- Feature 2 with brief description

### Improvements
- Improvement 1
- Improvement 2

### Bug Fixes
- Fix 1
- Fix 2

## Breaking Changes

[If any, list explicitly with migration guide]

## Installation

[Download links and installation instructions]

## Documentation

[Links to updated docs]

## Contributors

[Thank contributors if external contributions exist]
```

## Version History

| Version | Date | Description |
|---------|------|-------------|
| -       | -    | No releases yet |

Future releases will be tracked here.

## Rollback Procedure

If a release has critical issues:

1. **Identify issue** - Confirm severity requires rollback
2. **Communicate** - Notify users immediately
3. **Revert tag** - Delete bad tag (if not widely pulled)
4. **Fix forward** - Issue patch release (preferred)
5. **Document** - Add postmortem to release notes

**Prefer fix-forward over rollback** when possible.

## Hotfix Process

For critical bugs in released versions:

1. Branch from release tag: `git checkout -b hotfix-0.1.1 v0.1.0`
2. Fix bug with minimal changes
3. Test thoroughly
4. Merge to main
5. Cherry-pick to hotfix branch
6. Tag patch release: `v0.1.1`
7. Release notes explain hotfix

## Release Automation (Future)

Potential automation using GitHub Actions:

```yaml
# .github/workflows/release.yml (not yet created)
name: Release
on:
  push:
    tags:
      - 'v*'
steps:
  - Build binaries
  - Generate checksums
  - Create GitHub release
  - Upload artifacts
  - Publish release notes
```

## Deprecation Policy

For v0.x versions:
- No deprecation guarantees
- Breaking changes allowed
- Advance notice encouraged but not required

For v1.x versions (future):
- Deprecation warnings in advance
- Migration guides provided
- Deprecation period of at least one minor version

## Support Policy

**Main branch:**
- Continuously maintained
- Bug fixes applied immediately
- New features added regularly

**Released versions:**
- No backports to old versions in v0.x
- Users should upgrade to latest

**v1.x versions (future):**
- Security fixes backported
- Critical bug fixes backported
- Feature additions only in new versions

## Communication Channels

Release announcements via:
- GitHub Releases (primary)
- Repository README badges (future)
- Project discussions (if enabled)

## Metrics and Monitoring

Track per release:
- Download counts (future)
- Adoption metrics (future)
- Bug reports frequency
- Documentation completeness

## References

- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution workflow
- [docs/DEVELOPMENT.md](./DEVELOPMENT.md) - Development setup
- [TASKS.md](../TASKS.md) - Development roadmap
- [PROJECT_CONTEXT.md](../PROJECT_CONTEXT.md) - Design philosophy
