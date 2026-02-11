# Contributing to Igor

Thank you for your interest in contributing to Igor v0. This document provides guidelines for contributing to the project.

## Project Philosophy

Igor is an experimental runtime for autonomous mobile agents. Development follows core principles:

**Survival-Oriented Runtime:**
Agents must be able to checkpoint state, migrate between nodes, and survive infrastructure failure without external intervention.

**Decentralized Infrastructure:**
No centralized coordinator or platform. Nodes are peer-to-peer participants that provide execution services autonomously.

**Deterministic Execution:**
Agent behavior must be reproducible given the same state. Explicit state management over implicit assumptions.

**Security-First Sandboxing:**
Agents execute in WASM sandboxes with strict memory limits and capability restrictions. Safety through isolation.

**Minimal Scope:**
Igor v0 focuses on proving autonomous agent survival is feasible. Features outside this scope are deferred or rejected.

**Fail Loudly:**
Invariant violations cause immediate errors. Correctness over graceful degradation.

See [PROJECT_CONTEXT.md](./PROJECT_CONTEXT.md) for authoritative design specification.

## Development Setup

### Prerequisites

**Required:**
- Go 1.22+ ([install](https://go.dev/dl/))
- TinyGo ([install](https://tinygo.org/getting-started/install/))
- golangci-lint ([install](https://golangci-lint.run/welcome/install/))
- goimports: `go install golang.org/x/tools/cmd/goimports@latest`

### Clone and Build

```bash
git clone https://github.com/simonovic86/igor.git
cd igor
make build
```

### Run Tests

```bash
make test
```

### Run Quality Checks

```bash
make check  # Runs fmt-check, vet, lint
```

See [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) for detailed development guide.

## Pull Request Guidelines

### Before Submitting

1. **Run quality checks:**
   ```bash
   make check
   ```

2. **Run tests:**
   ```bash
   make test
   ```

3. **Build successfully:**
   ```bash
   make build
   ```

4. **Update documentation** if adding features or changing behavior

### PR Requirements

**Small, Focused Changes:**
- One logical change per PR
- Avoid mixing refactoring with feature addition
- Split large changes into sequential PRs

**Tests Required:**
- Add tests for new functionality
- Ensure existing tests pass
- Aim for test coverage of critical paths

**Documentation Updates:**
- Update relevant docs/ files if behavior changes
- Update DEVELOPMENT.md if workflow changes
- Update PROJECT_CONTEXT.md only with maintainer approval

**Clear Description:**
- Explain what the PR does and why
- Reference related issues if applicable
- Describe testing performed

## Commit Message Guidelines

Use conventional commits format:

```
<type>(<scope>): <subject>

<optional body>

<optional footer>
```

### Types

- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation changes
- `chore` - Tooling, dependencies, configuration
- `refactor` - Code restructuring without behavior change
- `test` - Test additions or corrections
- `perf` - Performance improvements

### Scopes

Examples:
- `runtime` - WASM execution engine
- `agent` - Agent lifecycle
- `migration` - Migration protocol
- `storage` - Checkpoint storage
- `p2p` - Networking layer
- `dev` - Developer tooling

### Examples

```
feat(migration): add multi-hop routing support

Implements relay-based migration for agents crossing
network boundaries.

Closes #42

---

fix(agent): correct budget calculation during tick timeout

Previously, timed-out ticks still deducted budget. Now budget
deduction only occurs for successful tick execution.

Fixes #38

---

docs(architecture): clarify checkpoint binary format

Added byte-level layout diagram for checkpoint structure
including budget metadata encoding.

---

chore(deps): update libp2p to v0.48.0

Includes security fixes and performance improvements.
```

### Commit Message Best Practices

- Use imperative mood ("add" not "added")
- Keep subject line under 72 characters
- Capitalize subject line
- No period at end of subject
- Separate subject from body with blank line
- Wrap body at 72 characters
- Explain what and why, not how

## Code Review Expectations

### Review Focus

Reviews prioritize:

1. **Correctness** - Does the code work as intended?
2. **Safety** - Are invariants maintained?
3. **Maintainability** - Can future contributors understand this?
4. **Clarity** - Is the intent obvious?

Performance optimization is secondary to correctness.

### Review Process

1. Reviewers provide feedback on code, design, and documentation
2. Author addresses feedback through discussion or changes
3. Approval indicates code meets quality standards
4. Merge after approval from code owners

### Addressing Feedback

- Respond to all review comments (even if just acknowledging)
- Push updates to the same branch
- Mark conversations as resolved when addressed
- Request re-review after substantial changes

## Code Style

### Formatting

- Use `gofmt` (enforced by CI)
- Use `goimports` for import organization
- Run `make fmt` before committing

### Naming

- Use clear, descriptive names
- Avoid abbreviations unless standard (ctx, err, pkg)
- Exported names must be clear without package context

### Error Handling

- Always check errors
- Use `fmt.Errorf` with `%w` for wrapping
- Provide context in error messages
- Log errors with structured logging

### Comments

- Document all exported functions and types
- Explain non-obvious logic
- Avoid redundant comments
- Use godoc conventions

## Communication

### Discussion Channels

- **GitHub Issues** - Bug reports, feature proposals, questions
- **Pull Requests** - Code review and technical discussion
- **Discussions** - General project direction (if enabled)

### Asking Questions

- Search existing issues first
- Provide context and specific details
- Include relevant logs or error messages
- Be respectful and patient

### Reporting Bugs

Include:
- Steps to reproduce
- Expected behavior
- Actual behavior
- Environment (OS, Go version, etc.)
- Relevant logs or error output

Use issue template if available.

## What to Contribute

### Welcome Contributions

- Bug fixes
- Documentation improvements
- Test coverage additions
- Example agents
- Performance analysis
- Security audits

### Discuss First

- New features
- Breaking changes
- Architecture modifications
- Major refactoring

Open an issue to discuss before investing significant effort.

### Out of Scope

Per PROJECT_CONTEXT.md, the following are explicitly out of scope:

- Agent marketplaces
- Reputation systems
- Staking or token economics
- AI / LLM functionality
- Multi-agent coordination frameworks
- Advanced security systems
- Distributed consensus

Proposals in these areas will be declined.

## Getting Help

- Read [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) for setup help
- Read [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) for system understanding
- Check existing issues for similar questions
- Open a new issue if stuck

## Recognition

Contributors are recognized through:
- Git commit authorship
- Pull request history
- CONTRIBUTORS file (if established)
- Release notes mentions

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

(License to be specified in LICENSE file)
