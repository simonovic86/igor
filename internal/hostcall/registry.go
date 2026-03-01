// Package hostcall implements the igor WASM host module containing
// runtime-mediated capability hostcalls per the capability membrane spec.
// Agents import functions from the "igor" namespace; only capabilities
// declared in the agent's manifest are registered (CE-1, CE-2).
package hostcall

import (
	"context"
	"log/slog"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/simonovic86/igor/pkg/manifest"
	"github.com/tetratelabs/wazero"
)

// Registry builds and manages the igor host module for a single agent.
type Registry struct {
	logger   *slog.Logger
	eventLog *eventlog.EventLog
}

// NewRegistry creates a hostcall registry bound to the given event log.
func NewRegistry(logger *slog.Logger, eventLog *eventlog.EventLog) *Registry {
	return &Registry{
		logger:   logger,
		eventLog: eventLog,
	}
}

// RegisterHostModule builds and instantiates the "igor" WASM host module
// with only the capabilities declared in the manifest.
// Must be called after WASI instantiation and before agent module instantiation.
func (r *Registry) RegisterHostModule(
	ctx context.Context,
	rt wazero.Runtime,
	m *manifest.CapabilityManifest,
) error {
	builder := rt.NewHostModuleBuilder("igor")
	registered := 0

	if m.Has("clock") {
		r.registerClock(builder)
		registered++
	}

	if m.Has("rand") {
		r.registerRand(builder)
		registered++
	}

	if m.Has("log") {
		r.registerLog(builder)
		registered++
	}

	// Only instantiate if at least one capability was registered.
	// If the agent has an empty manifest, skip module creation entirely.
	// If the agent's WASM imports from "igor", instantiation will fail
	// with a clear error about the missing module.
	if registered == 0 {
		r.logger.Info("No capabilities declared, igor host module not registered")
		return nil
	}

	if _, err := builder.Instantiate(ctx); err != nil {
		return err
	}

	r.logger.Info("igor host module registered",
		"capabilities", registered,
	)
	return nil
}
