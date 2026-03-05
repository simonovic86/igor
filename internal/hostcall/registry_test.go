// SPDX-License-Identifier: Apache-2.0

package hostcall

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/simonovic86/igor/internal/eventlog"
	"github.com/simonovic86/igor/pkg/manifest"
	"github.com/tetratelabs/wazero"
)

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	el := eventlog.NewEventLog(0)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewRegistry(logger, el)
}

func TestRegisterHostModule_AllCapabilities(t *testing.T) {
	ctx := context.Background()
	reg := newTestRegistry(t)

	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	m := &manifest.CapabilityManifest{
		Capabilities: map[string]manifest.CapabilityConfig{
			"clock": {Version: 1},
			"rand":  {Version: 1},
			"log":   {Version: 1},
		},
	}

	if err := reg.RegisterHostModule(ctx, rt, m); err != nil {
		t.Fatalf("RegisterHostModule failed: %v", err)
	}
}

func TestRegisterHostModule_SingleCapability(t *testing.T) {
	ctx := context.Background()
	reg := newTestRegistry(t)

	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	m := &manifest.CapabilityManifest{
		Capabilities: map[string]manifest.CapabilityConfig{
			"clock": {Version: 1},
		},
	}

	if err := reg.RegisterHostModule(ctx, rt, m); err != nil {
		t.Fatalf("RegisterHostModule failed: %v", err)
	}
}

func TestRegisterHostModule_EmptyManifest(t *testing.T) {
	ctx := context.Background()
	reg := newTestRegistry(t)

	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	m := &manifest.CapabilityManifest{
		Capabilities: map[string]manifest.CapabilityConfig{},
	}

	if err := reg.RegisterHostModule(ctx, rt, m); err != nil {
		t.Fatalf("RegisterHostModule failed: %v", err)
	}
}

func TestRegisterHostModule_NilManifest(t *testing.T) {
	ctx := context.Background()
	reg := newTestRegistry(t)

	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	m := &manifest.CapabilityManifest{}

	if err := reg.RegisterHostModule(ctx, rt, m); err != nil {
		t.Fatalf("RegisterHostModule failed: %v", err)
	}
}

func TestRegisterHostModule_UnknownCapabilityIgnored(t *testing.T) {
	ctx := context.Background()
	reg := newTestRegistry(t)

	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	// "net" is not implemented yet — should be ignored, not error
	m := &manifest.CapabilityManifest{
		Capabilities: map[string]manifest.CapabilityConfig{
			"clock": {Version: 1},
			"net":   {Version: 1},
		},
	}

	// This should succeed — ValidateAgainstNode catches unsupported caps,
	// not RegisterHostModule. RegisterHostModule only registers what it knows.
	if err := reg.RegisterHostModule(ctx, rt, m); err != nil {
		t.Fatalf("RegisterHostModule failed: %v", err)
	}
}
