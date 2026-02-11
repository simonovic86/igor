# Genesis Release Checklist

Pre-release verification checklist for Igor v0.1.0-genesis.

---

## Repository Integrity

### Code Quality

- [ ] CI passing on main branch (GitHub Actions green)
- [ ] `make check` passes locally (fmt, vet, lint, test)
- [ ] `make build` produces working binary
- [ ] `make agent` compiles example WASM agent
- [ ] No compiler warnings
- [ ] No linter errors

### Code Cleanliness

- [ ] No TODO markers in production code (`cmd/`, `internal/`, `pkg/`)
- [ ] No commented-out code blocks
- [ ] No debug print statements in runtime
- [ ] No placeholder functions
- [ ] All exported symbols documented

### Git Hygiene

- [ ] No merge conflicts
- [ ] No uncommitted changes
- [ ] Branch is `main` or `master`
- [ ] All commits have clear messages
- [ ] History is clean (or ready for squash)

---

## Documentation Integrity

### Content Verification

- [ ] README.md finalized and professional
- [ ] PROJECT_CONTEXT.md reflects actual implementation
- [ ] VISION.md explains motivation clearly
- [ ] ARCHITECTURE.md describes implementation accurately
- [ ] All docs/ files use UPPERCASE naming
- [ ] No broken internal links (`grep -r "docs/" *.md docs/*.md`)
- [ ] No references to deleted files

### Documentation Scope

- [ ] Public docs clearly defined (see DOCUMENTATION_SCOPE.md)
- [ ] Investor materials removed or archived
- [ ] No business/market positioning in technical docs
- [ ] No speculative roadmap promises
- [ ] All docs serve technical contributors

### Cross-Reference Check

- [ ] README links to docs/ are valid
- [ ] CONTRIBUTING.md references are valid
- [ ] SECURITY.md links work
- [ ] Docs cross-references are accurate
- [ ] No 404 links

---

## Release Tag Preparation

### Version Definition

- [ ] Version decided: `v0.1.0-genesis`
- [ ] Version follows semantic versioning
- [ ] Tag format: `vX.Y.Z-suffix`
- [ ] Version documented in commit message

### Tag Annotation

- [ ] Annotated tag (not lightweight)
- [ ] Tag message prepared (see GENESIS_COMMIT.md)
- [ ] Tag includes:
  - What Igor is
  - What Phase 1 achieves
  - Known limitations
  - Experimental status

### Release Notes

- [ ] Release notes drafted
- [ ] Highlights Phase 1 completion
- [ ] Lists all 6 success criteria met
- [ ] Explains known limitations clearly
- [ ] Links to documentation
- [ ] Includes installation instructions
- [ ] Includes quick start example

---

## Public Risk Disclosure

### Status Declarations

- [ ] README declares "Experimental, research-stage"
- [ ] README declares "Not ready for production use"
- [ ] SECURITY.md declares maturity level
- [ ] SECURITY.md lists known limitations
- [ ] Documentation avoids production-ready claims

### Limitation Documentation

- [ ] Trusted accounting model documented
- [ ] Single-hop migration constraint documented
- [ ] Local storage limitation documented
- [ ] Security model limitations documented
- [ ] Performance characteristics honest (no SLA claims)

### Use Case Warnings

- [ ] Do not use for production workloads
- [ ] Do not deploy on public networks
- [ ] Do not process sensitive data
- [ ] Do not use for financial transactions
- [ ] Suitable only for research/experimentation

Documented in: README.md, SECURITY.md, docs/SECURITY_MODEL.md

---

## Technical Verification

### Functionality

- [ ] Agent can start and run locally
- [ ] Agent survives restart (checkpoint/resume works)
- [ ] Agent can migrate between two nodes
- [ ] Budget metering functions correctly
- [ ] Budget exhaustion terminates agent gracefully
- [ ] Single-instance invariant maintained
- [ ] Checkpoints are atomic (verified via testing)

### Build Verification

- [ ] `make clean && make build` succeeds
- [ ] Binary runs: `./bin/igord`
- [ ] Agent example compiles: `make agent`
- [ ] Agent runs: `./bin/igord --run-agent agents/example/agent.wasm`
- [ ] No segfaults or panics during normal operation

### Platform Support

- [ ] Builds on Linux (verified in CI)
- [ ] Builds on macOS (if applicable)
- [ ] Go 1.25.4 required (documented)
- [ ] TinyGo 0.40.1+ required (documented)

---

## Communication Readiness

### Messaging

- [ ] ANNOUNCEMENT.md created and reviewed
- [ ] Explanation is technically accurate
- [ ] Tone is professional and measured
- [ ] No hype or exaggerated claims
- [ ] Invitation for collaboration clear

### Repository Presentation

- [ ] GitHub repository description set
- [ ] README is first-impression ready
- [ ] Repository topics/tags appropriate (if applicable)
- [ ] No placeholder content
- [ ] Professional appearance

---

## Pre-Release Actions

### Local Verification

```bash
# Clean build
make clean
make build

# Quality checks
make check

# Functionality test
./bin/igord --run-agent agents/example/agent.wasm --budget 1.0
# Verify: agent runs, ticks, checkpoints, terminates on interrupt
```

### Documentation Review

```bash
# Check for broken links
grep -r "\[.*\](.*.md)" README.md CONTRIBUTING.md SECURITY.md docs/*.md | \
  while read line; do
    # Verify each link resolves
  done

# Check for TODO markers
grep -r "TODO\|FIXME\|XXX" cmd/ internal/ pkg/ || echo "Clean"
```

### Git Status

```bash
git status  # Should be clean
git log --oneline -10  # Review recent commits
```

---

## Release Process

**Do NOT execute during this task. Manual actions only.**

### 1. Create Tag

```bash
git tag -a v0.1.0-genesis -F docs/GENESIS_COMMIT.md
```

### 2. Verify Tag

```bash
git tag -l -n20 v0.1.0-genesis
```

### 3. Push Tag

```bash
git push origin v0.1.0-genesis
```

### 4. Create GitHub Release

- Title: "Igor v0.1.0-genesis - Phase 1 (Survival) Complete"
- Body: Use release notes from GENESIS_COMMIT.md
- Mark as pre-release
- Attach binaries (if built)

### 5. Post Announcement

- Publish ANNOUNCEMENT.md contents
- Share in appropriate channels
- Invite technical feedback

---

## Post-Release Validation

- [ ] Tag visible on GitHub
- [ ] Release published successfully
- [ ] Documentation renders correctly
- [ ] Links work from GitHub UI
- [ ] CI badge shows passing (if added)
- [ ] Clone from GitHub works: `git clone https://github.com/simonovic86/igor.git`
- [ ] Fresh clone builds: `cd igor && make build`

---

## Current Status

**Checklist completion:** In progress  
**Ready for release:** Pending verification  
**Last updated:** 2026-02-11

Complete all checklist items before tagging v0.1.0-genesis.
