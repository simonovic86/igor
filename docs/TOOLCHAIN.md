# Toolchain Management

Igor uses explicit toolchain version locking to ensure deterministic builds across all environments.

## Version Policy

### Go Version

**Required:** Go 1.25.4

**Specified in:**
- `go.mod` - Module Go version
- `.golangci.yml` - Linter Go target
- `.github/workflows/ci.yml` - CI Go version
- `scripts/bootstrap.sh` - Bootstrap verification

**Policy:**
- Minor version updates allowed (1.25.x → 1.26.x) with testing
- Major version updates require architecture review
- All environments must use identical Go version

### golangci-lint Version

**Required:** v1.63.4

**Specified in:**
- `tools.go` - Go module dependency
- `.github/workflows/ci.yml` - CI action version
- `scripts/bootstrap.sh` - Bootstrap installation

**Policy:**
- Pin specific patch version (v1.63.4)
- Update only when new linters needed or Go version requires
- Test thoroughly before updating

### goimports Version

**Required:** Latest from golang.org/x/tools

**Specified in:**
- `tools.go` - Go module dependency
- `.github/workflows/ci.yml` - CI installation
- `scripts/bootstrap.sh` - Bootstrap installation

**Policy:**
- Always use latest (stable API)
- Automatically updates with Go toolchain

### TinyGo Version

**Required:** 0.40.1+ (optional)

**Used for:** Compiling agents to WASM

**Policy:**
- Not required for runtime development
- Required only for building new agents
- Version specified in agent documentation

## Toolchain Installation

### Automated Bootstrap

Run the bootstrap script to install all required tools:

```bash
./scripts/bootstrap.sh
```

The script:
1. Verifies Go 1.25.4 installed
2. Downloads module dependencies
3. Installs goimports (latest)
4. Installs golangci-lint v1.63.4
5. Checks TinyGo presence (optional)
6. Installs Git hooks
7. Verifies build
8. Runs quality checks

### Manual Installation

**Go 1.25.4:**
```bash
# Download from https://go.dev/dl/
# Or use version manager:
gvm install go1.25.4
gvm use go1.25.4
```

**golangci-lint v1.63.4:**
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.63.4
```

**goimports:**
```bash
go install golang.org/x/tools/cmd/goimports@latest
```

**TinyGo (optional):**
```bash
# macOS
brew install tinygo

# Linux
wget https://github.com/tinygo-org/tinygo/releases/download/v0.40.1/tinygo_0.40.1_amd64.deb
sudo dpkg -i tinygo_0.40.1_amd64.deb
```

## Dependency Locking

### tools.go

Igor uses `tools.go` to track development tool dependencies:

```go
//go:build tools

package tools

import (
    _ "github.com/golangci/golangci-lint/cmd/golangci-lint"
    _ "golang.org/x/tools/cmd/goimports"
)
```

This ensures:
- Tool versions recorded in `go.mod` and `go.sum`
- Reproducible across machines
- Explicit version upgrades via `go get`

### Module Management

**Install tools from lock:**
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint
go install golang.org/x/tools/cmd/goimports
```

Reads versions from `go.mod`.

## Version Upgrade Procedure

### Upgrading Go

1. **Test locally:**
   ```bash
   go get toolchain@go1.26.0
   make check
   make test
   ```

2. **Update all references:**
   - `go.mod`
   - `.golangci.yml` (run.go)
   - `.github/workflows/ci.yml`
   - `scripts/bootstrap.sh`
   - `docs/TOOLCHAIN.md`

3. **Verify CI passes:**
   Push to branch, ensure green build

4. **Update documentation:**
   Note compatibility changes

### Upgrading golangci-lint

1. **Check compatibility with Go version:**
   https://github.com/golangci/golangci-lint/releases

2. **Update version:**
   ```bash
   go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.0
   ```

3. **Update references:**
   - `tools.go` (implicit via go.mod)
   - `.github/workflows/ci.yml` (action version)
   - `scripts/bootstrap.sh` (install command)

4. **Test locally:**
   ```bash
   make lint
   ```

5. **Verify CI:**
   Check GitHub Actions passes

## Compatibility Rules

### Go Backward Compatibility

Igor requires exact minor version (1.25.x):
- `go.mod` specifies `go 1.25.4`
- Earlier versions (1.24.x) may work but are unsupported
- Later patches (1.25.5+) should work but must be tested

### Linter Compatibility

golangci-lint must support project Go version:
- v1.63.4 supports Go 1.25+
- Check release notes before upgrading
- Test on legacy-history branch first

### Module Compatibility

All dependencies must support Go 1.25+:
- Review `go.mod` changes carefully
- Test upgrades in isolation
- Verify no breaking changes in APIs

## Environment Verification

### Check Versions

```bash
go version                 # Should show 1.25.4
golangci-lint version      # Should show 1.63.4
goimports -help            # Should succeed
tinygo version             # Optional
```

### Verify Parity

**Local:**
```bash
make check
```

**CI:**
Check GitHub Actions output matches local results exactly.

**Differences indicate drift.** Resolve immediately.

## Drift Detection

### Preventing Drift

**In CI:**
```yaml
- name: Verify dependencies are tidy
  run: |
    go mod tidy
    git diff --exit-code go.mod go.sum
```

Fails if dependencies changed unexpectedly.

**Locally:**
```bash
go mod tidy
git diff go.mod go.sum  # Should show no changes
```

### Resolving Drift

If drift detected:

1. **Determine cause:**
   - Dependency update?
   - Tool version change?
   - Platform-specific behavior?

2. **Fix consistently:**
   - Update go.mod explicitly
   - Document change
   - Update all references

3. **Verify:**
   ```bash
   make check  # Local
   # Push and check CI
   ```

## Troubleshooting

### "golangci-lint not found"

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.63.4
# Ensure $(go env GOPATH)/bin in PATH
export PATH="$PATH:$(go env GOPATH)/bin"
```

### "Go version mismatch"

Download correct Go version:
```bash
# Visit https://go.dev/dl/
# Download go1.25.4.darwin-amd64.pkg (macOS)
# Or: go1.25.4.linux-amd64.tar.gz (Linux)
```

### "Module checksum mismatch"

```bash
go clean -modcache
go mod download
```

### "CI lint fails but local passes"

Check versions:
```bash
go version
golangci-lint version
```

Compare with `.github/workflows/ci.yml`.

## Toolchain Philosophy

Igor requires **deterministic builds**:

- Same inputs → same outputs
- No version drift
- No environment-specific behavior
- Reproducible across developers and CI

This philosophy aligns with PROJECT_CONTEXT.md:
> "Deterministic behavior preferred"

Toolchain determinism ensures:
- Reliable CI results
- Consistent developer experience
- Predictable builds
- Simplified debugging

## References

- [DEVELOPMENT.md](./DEVELOPMENT.md) - Development workflow
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution guidelines
- [PROJECT_CONTEXT.md](../PROJECT_CONTEXT.md) - Design philosophy
