//go:build tools

// Package tools tracks development tool dependencies via Go modules.
// This ensures golangci-lint and other tools are versioned consistently.
package tools

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "golang.org/x/tools/cmd/goimports"
)
