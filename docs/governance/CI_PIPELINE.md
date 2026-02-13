# CI Pipeline Documentation

## Overview

This document describes Igor's continuous integration philosophy and intended pipeline structure. Actual CI workflows will be introduced in a future task.

**Status:** Documentation only. CI configuration not yet implemented.

## CI Goals

Igor's CI pipeline is designed to ensure:

**Deterministic Builds:**
- Same inputs produce identical outputs
- No dependency on build environment specifics
- Reproducible across developers and CI runners

**Lint Enforcement:**
- All code passes golangci-lint
- Formatting verified with gofmt
- Import organization checked with goimports

**Test Coverage Verification:**
- All tests pass
- No flaky tests
- Coverage metrics tracked (future)

**Dependency Hygiene:**
- `go.mod` and `go.sum` are tidy
- No unexpected dependency additions
- Security scanning of dependencies (future)

**Reproducible Binaries:**
- Build artifacts consistent across runs
- Version information embedded in binaries (future)
- Release artifacts generated automatically (future)

## Suggested Pipeline Stages

### Stage 1: Setup

**Actions:**
- Check out repository code
- Set up Go toolchain (version from go.mod)
- Cache Go modules for faster builds
- Install required tools (golangci-lint, goimports)

**Commands:**
```bash
go version
go env
```

### Stage 2: Dependency Validation

**Actions:**
- Verify go.mod and go.sum are tidy
- Check for security vulnerabilities in dependencies

**Commands:**
```bash
make tidy
git diff --exit-code go.mod go.sum
```

**Purpose:** Ensure dependency files are correct and committed.

### Stage 3: Formatting Check

**Actions:**
- Verify code is properly formatted
- Check import organization

**Commands:**
```bash
make fmt-check
```

**Purpose:** Enforce consistent code style.

### Stage 4: Static Analysis

**Actions:**
- Run go vet on all packages
- Run golangci-lint with full configuration

**Commands:**
```bash
make vet
make lint
```

**Purpose:** Catch potential bugs and code quality issues.

### Stage 5: Unit Testing

**Actions:**
- Run all unit tests
- Generate coverage report
- Verify coverage thresholds (future)

**Commands:**
```bash
make test
# Future: go test -coverprofile=coverage.txt -covermode=atomic ./...
```

**Purpose:** Ensure functionality correctness.

### Stage 6: Build Verification

**Actions:**
- Build igord binary
- Build example agent WASM
- Verify binaries are produced

**Commands:**
```bash
make build
make agent
```

**Purpose:** Ensure code compiles successfully.

### Stage 7: Integration Testing (Future)

**Actions:**
- Start test nodes
- Run agent migration tests
- Verify checkpoint survival
- Test budget enforcement

**Commands:**
```bash
# Future: make integration-test
```

**Purpose:** Validate end-to-end behavior.

## Pipeline Configuration

### Matrix Strategy

Test across:

**Operating Systems:**
- Ubuntu 22.04 (primary)
- macOS 13 (secondary)
- Windows (future, if needed)

**Go Versions:**
- Latest stable (1.25.x currently)
- Previous minor (1.24.x)

### Caching Strategy

Cache:
- Go modules (`~/go/pkg/mod`)
- golangci-lint cache (`~/.cache/golangci-lint`)
- Build cache (`~/.cache/go-build`)

Invalidate on:
- `go.mod` or `go.sum` changes
- `.golangci.yml` changes

### Failure Handling

**On Failure:**
- Mark build as failed
- Post detailed error logs
- Block merge if required check

**On Flaky Test:**
- Investigate immediately
- Fix or skip temporarily with issue tracking
- Aim for zero flaky tests

## CI Providers

Potential CI platforms for Igor:

**GitHub Actions:**
- Tight GitHub integration
- Free for public repos
- Easy matrix builds
- Good Go support

**GitLab CI:**
- Self-hostable
- Strong caching
- Docker-friendly

**CircleCI:**
- Fast build times
- Good free tier
- Complex workflows

**Decision pending.** GitHub Actions likely for simplicity.

## Branch Protection

### Main Branch Protection (Future)

Require:
- Status checks pass (lint, vet, test, build)
- At least one approval from code owner
- No direct pushes (all changes via PR)
- Linear history (rebase or squash merge)

### Development Workflow

```
feature branch → PR → review → checks pass → merge to main
```

No long-lived branches. Features merge quickly after validation.

## Performance Benchmarks (Future)

Track performance metrics:
- WASM compilation time
- Agent tick latency
- Migration duration
- Checkpoint write latency

Run benchmarks on:
- Every PR (compare against main)
- Daily (track trends)

Alert on:
- Regressions > 10%
- Increased memory usage
- Slower migration

## Security Scanning (Future)

Include security tools:

**Dependency Scanning:**
```bash
go list -json -m all | nancy sleuth
```

**Static Security Analysis:**
```bash
gosec ./...
```

**License Compliance:**
```bash
go-licenses check ./...
```

Run on every PR and nightly.

## Release Pipeline (Future)

Automated release process:

1. Tag version (vX.Y.Z)
2. Trigger release pipeline
3. Build binaries for multiple platforms
4. Generate checksums
5. Create GitHub release
6. Publish release notes
7. Update documentation

See [RELEASE_PROCESS.md](./RELEASE_PROCESS.md) for details.

## Monitoring and Observability

**Build Metrics:**
- Build success rate
- Average build duration
- Test execution time
- Cache hit rates

**Dashboard (future):**
- Visualize trends
- Track flakiness
- Identify bottlenecks

## GitHub Actions Implementation

**Status:** Implemented in `.github/workflows/ci.yml`

### Pipeline Overview

The GitHub Actions CI pipeline implements all stages described above:

**Triggers:**
- Push to main/master branch
- Pull requests to main/master

**Matrix Build:**
- OS: ubuntu-latest, macos-latest
- Go version: Read from go.mod (ensures toolchain alignment)

**Concurrency Control:**
- Cancels previous runs on same PR/branch
- Reduces unnecessary CI resource usage

**Bootstrap-Driven Toolchain:**
- Single source of truth: `scripts/bootstrap.sh`
- Tool versions locked in `tools.go` and `go.mod`
- CI and local environments use identical bootstrap procedure
- No external linter actions (prevents version mismatches)

**Toolchain Alignment:**
- Go 1.25.4 pinned explicitly
- golangci-lint v1.63.4 installed via `go install` (not external action)
- goimports installed via `go install`
- All tool versions tracked in go.mod/go.sum
- Prevents version drift by design

### Pipeline Stages

**Single Job: Quality Checks**

1. Checkout code (actions/checkout@v4)
2. Setup Go 1.25.4 (actions/setup-go@v5 with caching)
3. Verify toolchain infrastructure (tools.go, bootstrap.sh, pre-commit.sh exist)
4. Bootstrap environment (`make bootstrap`)
   - Installs golangci-lint v1.63.4
   - Installs goimports
   - Installs Git hooks
   - Verifies build
5. Validate dependencies (`go mod tidy` + diff check)
6. Run all quality checks (`make check`)
   - Format check
   - Static analysis (go vet)
   - Linting (golangci-lint)
   - Tests
7. Verify binary (build succeeded during bootstrap)

**Why no external linter action:**

External linter binaries are pre-compiled with unknown Go versions, causing version mismatches. Installing via `go install` ensures the linter is built with the project's Go version (1.25.4), eliminating compatibility issues.

### Caching Strategy

GitHub Actions auto-caches:
- Go modules via `setup-go` action
- Go build cache

golangci-lint cache is managed by the tool itself (not via external action).

Cache invalidates on:
- go.mod/go.sum changes
- .golangci.yml changes
- tools.go changes

### Local Quality Enforcement

**Pre-commit hooks:**

Install hooks to enforce quality before commits:

```bash
./scripts/install-hooks.sh
```

The hook runs automatically on `git commit` and checks:
- Formatting (`make fmt-check`)
- Static analysis (`make vet`)
- Linting (`make lint`)
- Tests (`make test`)

Commits are rejected if any check fails.

**Bypass hook** (use sparingly):
```bash
git commit --no-verify
```

### Debugging CI Failures

**Reproduce locally:**

```bash
# Run exact CI checks
make check      # Runs fmt-check, vet, lint
make test       # Runs unit tests
make build      # Verifies compilation
```

**Check specific failures:**

```bash
# Format issues
make fmt-check
make fmt  # Auto-fix

# Lint issues
make lint

# Test failures
make test
```

All CI checks run through Makefile targets, ensuring local/CI parity.

## Current State

**Implemented:**
- Makefile with quality check targets ✓
- golangci-lint configuration ✓
- Formatting enforcement tools ✓
- Developer documentation ✓
- GitHub Actions CI pipeline ✓
- Matrix builds (OS + Go versions) ✓
- Automated checks on PR ✓

**Not Yet Implemented:**
- Branch protection rules (manual GitHub configuration)
- Release automation
- Performance benchmarks
- Security scanning
- Coverage reporting

## References

- [DEVELOPMENT.md](./DEVELOPMENT.md) - Local development workflow
- [CONTRIBUTING.md](../../CONTRIBUTING.md) - Contribution guidelines
- [PROJECT_CONTEXT.md](../../PROJECT_CONTEXT.md) - Design principles
- [.github/workflows/ci.yml](../../.github/workflows/ci.yml) - CI configuration
