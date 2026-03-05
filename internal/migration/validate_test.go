// SPDX-License-Identifier: Apache-2.0

package migration

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/pkg/manifest"
)

func testValidateService(pricePerSecond int64, nodeCaps []string) *Service {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	s := &Service{
		pricePerSecond: pricePerSecond,
		logger:         logger,
		activeAgents:   make(map[string]*agent.Instance),
	}
	if nodeCaps != nil {
		s.nodeCapabilities = nodeCaps
	}
	return s
}

func TestValidateIncomingManifest_Accept(t *testing.T) {
	s := testValidateService(1000, nil)
	m := &manifest.Manifest{
		Capabilities: &manifest.CapabilityManifest{
			Capabilities: map[string]manifest.CapabilityConfig{
				"clock": {Version: 1},
			},
		},
	}
	if msg := s.validateIncomingManifest(m, "test-agent"); msg != "" {
		t.Errorf("expected acceptance, got rejection: %s", msg)
	}
}

func TestValidateIncomingManifest_MigrationDisabled(t *testing.T) {
	s := testValidateService(1000, nil)
	m := &manifest.Manifest{
		Capabilities: &manifest.CapabilityManifest{
			Capabilities: map[string]manifest.CapabilityConfig{},
		},
		MigrationPolicy: &manifest.MigrationPolicy{
			Enabled: false,
		},
	}
	msg := s.validateIncomingManifest(m, "test-agent")
	if msg == "" {
		t.Fatal("expected rejection for disabled migration")
	}
	if !strings.Contains(msg, "disabled") {
		t.Errorf("rejection should mention disabled, got: %s", msg)
	}
}

func TestValidateIncomingManifest_PriceExceeded(t *testing.T) {
	// Node charges 5000, agent max is 2000
	s := testValidateService(5000, nil)
	m := &manifest.Manifest{
		Capabilities: &manifest.CapabilityManifest{
			Capabilities: map[string]manifest.CapabilityConfig{},
		},
		MigrationPolicy: &manifest.MigrationPolicy{
			Enabled:           true,
			MaxPricePerSecond: 2000,
		},
	}
	msg := s.validateIncomingManifest(m, "test-agent")
	if msg == "" {
		t.Fatal("expected rejection for price exceeding max")
	}
	if !strings.Contains(msg, "exceeds") {
		t.Errorf("rejection should mention price exceeding, got: %s", msg)
	}
}

func TestValidateIncomingManifest_PriceWithinLimit(t *testing.T) {
	// Node charges 1000, agent max is 5000
	s := testValidateService(1000, nil)
	m := &manifest.Manifest{
		Capabilities: &manifest.CapabilityManifest{
			Capabilities: map[string]manifest.CapabilityConfig{
				"clock": {Version: 1},
			},
		},
		MigrationPolicy: &manifest.MigrationPolicy{
			Enabled:           true,
			MaxPricePerSecond: 5000,
		},
	}
	if msg := s.validateIncomingManifest(m, "test-agent"); msg != "" {
		t.Errorf("expected acceptance, got rejection: %s", msg)
	}
}

func TestValidateIncomingManifest_MemoryExceeded(t *testing.T) {
	s := testValidateService(1000, nil)
	m := &manifest.Manifest{
		Capabilities: &manifest.CapabilityManifest{
			Capabilities: map[string]manifest.CapabilityConfig{},
		},
		ResourceLimits: manifest.ResourceLimits{
			MaxMemoryBytes: manifest.DefaultMaxMemoryBytes + 1, // exceeds 64MB
		},
	}
	msg := s.validateIncomingManifest(m, "test-agent")
	if msg == "" {
		t.Fatal("expected rejection for excessive memory requirement")
	}
	if !strings.Contains(msg, "memory") {
		t.Errorf("rejection should mention memory, got: %s", msg)
	}
}

func TestValidateIncomingManifest_CapabilityMissing(t *testing.T) {
	// Node only has clock, agent wants "kv" which isn't available
	s := testValidateService(1000, []string{"clock"})
	m := &manifest.Manifest{
		Capabilities: &manifest.CapabilityManifest{
			Capabilities: map[string]manifest.CapabilityConfig{
				"clock": {Version: 1},
				"kv":    {Version: 1},
			},
		},
	}
	msg := s.validateIncomingManifest(m, "test-agent")
	if msg == "" {
		t.Fatal("expected rejection for missing capability")
	}
	if !strings.Contains(msg, "capability") {
		t.Errorf("rejection should mention capability, got: %s", msg)
	}
}

func TestValidateIncomingManifest_NilPolicy(t *testing.T) {
	// Nil migration policy means migration is allowed by default
	s := testValidateService(1000, nil)
	m := &manifest.Manifest{
		Capabilities: &manifest.CapabilityManifest{
			Capabilities: map[string]manifest.CapabilityConfig{
				"clock": {Version: 1},
			},
		},
		MigrationPolicy: nil,
	}
	if msg := s.validateIncomingManifest(m, "test-agent"); msg != "" {
		t.Errorf("nil policy should allow migration, got rejection: %s", msg)
	}
}
