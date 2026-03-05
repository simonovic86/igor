// Package runtime implements survivable autonomous agent execution using WASM sandboxing.
// Provides deterministic tick-based execution, resource isolation, and timeout enforcement
// for portable agent runtimes that can checkpoint and migrate across infrastructure.
package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Engine manages WASM agent execution.
type Engine struct {
	runtime wazero.Runtime
	logger  *slog.Logger
}

// NewEngine creates a new WASM execution engine with safe sandbox configuration.
func NewEngine(ctx context.Context, logger *slog.Logger) (*Engine, error) {
	// Create runtime with safe configuration
	config := wazero.NewRuntimeConfig().
		// Limit memory to 64MB per agent
		WithMemoryLimitPages(1024). // 1024 pages * 64KB = 64MB
		// Disable most dangerous features
		WithCloseOnContextDone(true)

	rt := wazero.NewRuntimeWithConfig(ctx, config)

	// Instantiate WASI with minimal capabilities
	// Disable filesystem and network access
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("instantiate WASI: %w", err)
	}

	logger.Info("WASM runtime engine created",
		"memory_limit_mb", 64,
		"filesystem", "disabled",
		"network", "disabled",
	)

	return &Engine{
		runtime: rt,
		logger:  logger,
	}, nil
}

// LoadWASM compiles a WASM binary from a file path.
func (e *Engine) LoadWASM(ctx context.Context, wasmPath string) (wazero.CompiledModule, error) {
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read WASM file: %w", err)
	}

	e.logger.Info("Compiling WASM module",
		"path", wasmPath,
		"size_bytes", len(wasmBytes),
	)

	return e.CompileWASMBytes(ctx, wasmBytes)
}

// CompileWASMBytes compiles a WASM binary from raw bytes.
func (e *Engine) CompileWASMBytes(ctx context.Context, wasmBytes []byte) (wazero.CompiledModule, error) {
	compiled, err := e.runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to compile WASM module: %w", err)
	}

	return compiled, nil
}

// InstantiateModule instantiates a compiled WASM module.
func (e *Engine) InstantiateModule(
	ctx context.Context,
	compiled wazero.CompiledModule,
	moduleName string,
) (api.Module, error) {
	config := wazero.NewModuleConfig().
		WithName(moduleName).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		// Skip auto-start so we can call _start manually. Go 1.24+ with
		// //go:wasmexport needs _start for runtime init, but wazero's start
		// function mechanism closes the module after _start returns.
		// Calling _start manually as a regular export keeps the module alive.
		WithStartFunctions()

	module, err := e.runtime.InstantiateModule(ctx, compiled, config)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate module: %w", err)
	}

	return module, nil
}

// Runtime returns the underlying wazero runtime for host module registration.
// This must be called after NewEngine (WASI is already instantiated) and
// before InstantiateModule (so the agent can import from host modules).
func (e *Engine) Runtime() wazero.Runtime {
	return e.runtime
}

// Close releases all runtime resources.
func (e *Engine) Close(ctx context.Context) error {
	return e.runtime.Close(ctx)
}
