// SPDX-License-Identifier: Apache-2.0

// Package hostcall implements the igor WASM host module containing
// runtime-mediated capability hostcalls per the capability membrane spec.
// Agents import functions from the "igor" namespace; only capabilities
// declared in the agent's manifest are registered (CE-1, CE-2).
package hostcall

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/simonovic86/igor/pkg/manifest"
	"github.com/tetratelabs/wazero"
)

// Registry builds and manages the igor host module for a single agent.
type Registry struct {
	logger         *slog.Logger
	eventLog       *eventlog.EventLog
	walletState    WalletState    // optional; nil = wallet hostcalls not available
	pricingState   PricingState   // optional; nil = pricing hostcalls not available
	httpClient     HTTPClient     // optional; nil = use http.DefaultClient
	walletPayState WalletPayState // optional; nil = payment hostcalls not available
}

// NewRegistry creates a hostcall registry bound to the given event log.
func NewRegistry(logger *slog.Logger, eventLog *eventlog.EventLog) *Registry {
	return &Registry{
		logger:   logger,
		eventLog: eventLog,
	}
}

// SetWalletState installs the wallet state provider for wallet.* hostcalls.
// Must be called before RegisterHostModule if the agent declares "wallet" capability.
func (r *Registry) SetWalletState(ws WalletState) {
	r.walletState = ws
}

// SetPricingState installs the pricing state provider for pricing hostcalls.
// Must be called before RegisterHostModule if the agent declares "pricing" capability.
func (r *Registry) SetPricingState(ps PricingState) {
	r.pricingState = ps
}

// SetHTTPClient installs a custom HTTP client for the http_request hostcall.
// If not set, http.DefaultClient is used. Useful for testing.
func (r *Registry) SetHTTPClient(c HTTPClient) {
	r.httpClient = c
}

// SetWalletPayState installs the payment state provider for the wallet_pay hostcall.
// Must be called before RegisterHostModule if the agent declares "x402" capability.
func (r *Registry) SetWalletPayState(wps WalletPayState) {
	r.walletPayState = wps
}

// RegisterHostModule builds and instantiates the "igor" WASM host module
// with only the capabilities declared in the manifest.
// Must be called after WASI instantiation and before agent module instantiation.
func (r *Registry) RegisterHostModule(
	ctx context.Context,
	rt wazero.Runtime,
	m *manifest.CapabilityManifest,
) error {
	// Close any previously instantiated igor module. This happens when a node
	// receives a second agent (e.g. after the first migrated away and another
	// migrates in). The old guest module is already closed, so the host module
	// can be safely replaced with fresh closures bound to the new event log.
	if existing := rt.Module("igor"); existing != nil {
		if err := existing.Close(ctx); err != nil {
			return fmt.Errorf("failed to close existing igor module: %w", err)
		}
	}

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

	if m.Has("wallet") && r.walletState != nil {
		r.registerWallet(builder, r.walletState)
		registered++
	}

	if m.Has("pricing") && r.pricingState != nil {
		r.registerPricing(builder, r.pricingState)
		registered++
	}

	if m.Has("http") {
		capCfg := m.Capabilities["http"]
		r.registerHTTP(builder, capCfg)
		registered++
	}

	if m.Has("x402") && r.walletPayState != nil {
		capCfg := m.Capabilities["x402"]
		r.registerPayment(builder, r.walletPayState, capCfg)
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
		return fmt.Errorf("failed to instantiate igor host module: %w", err)
	}

	r.logger.Info("igor host module registered",
		"capabilities", registered,
	)
	return nil
}
