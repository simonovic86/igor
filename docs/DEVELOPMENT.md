# Igor Development Guide

## Prerequisites

### Required Tools

**Go 1.22+**

Igor requires Go 1.22 or later. Check your version:

```bash
go version
```

Install or update Go from: https://go.dev/dl/

**TinyGo** (for building agents)

Required for compiling agents to WASM:

```bash
# macOS
brew tap tinygo-org/tools
brew install tinygo

# Linux
wget https://github.com/tinygo-org/tinygo/releases/download/v0.40.1/tinygo_0.40.1_amd64.deb
sudo dpkg -i tinygo_0.40.1_amd64.deb
```

Verify installation:

```bash
tinygo version
```

### Recommended Tools

**golangci-lint** (code quality)

Install linter:

```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
  sh -s -- -b $(go env GOPATH)/bin
```

Verify installation:

```bash
golangci-lint --version
```

**goimports** (import formatting)

Install import formatter:

```bash
go install golang.org/x/tools/cmd/goimports@latest
```

Ensure `$(go env GOPATH)/bin` is in your PATH.

## Development Workflow

### Initial Setup

After cloning the repository, install Git hooks:

```bash
./scripts/install-hooks.sh
```

This installs pre-commit hooks that enforce code quality before each commit.

### Local Quality Gates

**Pre-commit hooks automatically run:**
- Code formatting check (`make fmt-check`)
- Static analysis (`make vet`)
- Linting (`make lint`)
- Tests (`make test`)

If any check fails, the commit is rejected. Fix issues before committing.

**To bypass hooks** (not recommended):
```bash
git commit --no-verify
```

Only bypass for work-in-progress commits on feature branches.

### Building Igor

Build the node runtime:

```bash
make build
```

Output: `bin/igord`

### Building Agents

Build the example agent:

```bash
make agent
```

Output: `agents/example/agent.wasm`

### Running Locally

Run an agent with default budget:

```bash
make run-agent
```

Or manually:

```bash
./bin/igord --run-agent agents/example/agent.wasm --budget 10.0
```

### Code Quality

**Format code:**

```bash
make fmt
```

**Check formatting without modifying:**

```bash
make fmt-check
```

**Run linters:**

```bash
make lint
```

**Run go vet:**

```bash
make vet
```

**Run all checks:**

```bash
make check
```

### Testing

Run tests:

```bash
make test
```

Run tests with coverage:

```bash
go test -cover ./...
```

### Cleaning Build Artifacts

Remove binaries and checkpoints:

```bash
make clean
```

## Makefile Targets

Run `make help` to see all available targets:

```bash
make help
```

Output:
```
Igor v0 Development Commands

Usage:
  make <target>

Targets:
  help         Show this help message
  build        Build igord binary
  clean        Remove build artifacts
  test         Run tests
  lint         Run golangci-lint
  vet          Run go vet
  fmt          Format code with gofmt and goimports
  fmt-check    Check if code is formatted correctly
  tidy         Tidy go.mod and go.sum
  agent        Build example agent WASM
  run-agent    Build and run example agent locally
  check        Run all checks (formatting, vet, lint)
  all          Clean, build, test, and run all checks
```

## Code Style Guidelines

### Formatting

Igor uses standard Go formatting:

- `gofmt` for code formatting
- `goimports` for import organization
- 80-character line length preferred (not enforced)

Run `make fmt` before committing.

### Linting

Igor uses golangci-lint with the following enabled linters:

- `govet` - Official Go static analysis
- `staticcheck` - Advanced correctness checks
- `errcheck` - Unchecked error detection
- `ineffassign` - Ineffectual assignment detection
- `gosimple` - Code simplification suggestions
- `unused` - Unused code detection
- `revive` - General code quality
- `gocyclo` - Cyclomatic complexity (max: 15)

Run `make lint` to check code quality.

### Error Handling

- Always check errors
- Use `fmt.Errorf` with `%w` for error wrapping
- Log errors with context
- Fail loudly on invariant violations

### Logging

Use structured logging with `slog`:

```go
logger.Info("Operation completed",
    "agent_id", agentID,
    "duration_ms", elapsed.Milliseconds(),
)
```

### Comments

- Document exported functions and types
- Explain non-obvious logic
- Avoid redundant comments
- Use godoc conventions

## Project Structure

```
igor/
├── cmd/igord/           # Node runtime entry point
├── internal/            # Internal packages (not importable)
│   ├── agent/          # Agent instance management
│   ├── config/         # Configuration
│   ├── logging/        # Structured logging
│   ├── migration/      # Migration coordination
│   ├── p2p/            # P2P networking
│   ├── runtime/        # WASM execution engine
│   └── storage/        # Checkpoint storage
├── pkg/                 # Public packages (importable)
│   ├── manifest/       # Agent manifest schema
│   └── protocol/       # P2P message types
├── agents/example/      # Example agent
├── docs/               # Documentation
└── bin/                # Compiled binaries (gitignored)
```

### Package Organization

**internal/** - Private implementation packages. Cannot be imported by external code.

**pkg/** - Public API packages. Can be imported by agents or external tools.

**cmd/** - Entry points for binaries.

## Development Best Practices

### Before Committing

Run quality checks:

```bash
make check
```

This runs:
1. `make fmt-check` - Verify formatting
2. `make vet` - Run static analysis
3. `make lint` - Run linters

Fix any issues before committing.

### Commit Messages

Follow conventional commits format:

```
<type>(<scope>): <subject>

<body>
```

Types:
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation only
- `chore` - Tooling, dependencies
- `refactor` - Code restructuring
- `test` - Test additions

Examples:
```
feat(migration): add multi-hop migration support
fix(agent): correct budget calculation rounding
docs(architecture): clarify checkpoint format
chore(deps): update libp2p to v0.48
```

### Testing

Write tests for:
- Core logic in `internal/` packages
- Protocol message encoding/decoding
- Storage provider implementations
- Migration flows

Place tests alongside code:
```
internal/agent/
  instance.go
  instance_test.go
```

Run tests before committing:
```bash
make test
```

## Troubleshooting

### golangci-lint errors

If linter fails:

1. Check which linter reported the issue
2. Read the error message carefully
3. Fix the issue or suppress if false positive
4. Re-run `make lint`

Suppress false positives with `//nolint` comments:

```go
//nolint:errcheck // Intentionally ignoring error
_ = stream.Close()
```

### goimports issues

If imports are not organizing correctly:

```bash
# Manual fix
goimports -w .
```

### Build failures

If build fails after pulling changes:

```bash
make clean
make tidy
make build
```

### Agent compilation issues

If TinyGo compilation fails:

```bash
# Check TinyGo version
tinygo version

# Clean and rebuild
cd agents/example
make clean
make build
```

## Editor Integration

### VS Code

Install Go extension:
```
code --install-extension golang.go
```

Configure settings (`.vscode/settings.json`):

```json
{
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "editor.formatOnSave": true,
  "go.formatTool": "goimports"
}
```

### Vim/Neovim

Use `vim-go` plugin with:

```vim
let g:go_fmt_command = "goimports"
let g:go_metalinter_command = "golangci-lint"
```

### GoLand/IntelliJ

1. Preferences → Tools → File Watchers
2. Add `goimports` file watcher
3. Enable golangci-lint inspection

## Performance Profiling

Profile CPU usage:

```bash
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof
```

Profile memory:

```bash
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

## Debugging

### Runtime Debugging

Use delve debugger:

```bash
# Install
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug igord
dlv exec ./bin/igord -- --run-agent agents/example/agent.wasm
```

### Agent Debugging

Add debug output in agent code:

```go
import "fmt"

func agent_tick() {
    fmt.Printf("[debug] Counter: %d\n", state.Counter)
    // ...
}
```

Output appears in igord logs.

## Contributing

### Pull Request Checklist

Before submitting PR:

- [ ] `make check` passes
- [ ] `make test` passes
- [ ] New tests added for new functionality
- [ ] Documentation updated
- [ ] Commit messages follow conventions
- [ ] No unrelated changes included

### Code Review Focus

Reviews focus on:

1. **Correctness** - Does it work as intended?
2. **Invariants** - Are system guarantees maintained?
3. **Clarity** - Is code understandable?
4. **Testing** - Is it adequately tested?

Performance optimization is secondary to correctness.

## Additional Resources

- [PROJECT_CONTEXT.md](../PROJECT_CONTEXT.md) - Authoritative design specification
- [docs/ARCHITECTURE.md](./ARCHITECTURE.md) - Technical architecture
- [docs/AGENT_LIFECYCLE.md](./AGENT_LIFECYCLE.md) - Agent development guide
- [TASKS.md](../TASKS.md) - Development roadmap
