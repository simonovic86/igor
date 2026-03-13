// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TickTimeout is the maximum duration for a single agent tick.
// Used by agent execution, replay verification, and the simulator.
// Set to 15s to accommodate agents making HTTP requests during ticks.
const TickTimeout = 15 * time.Second

// Config holds the runtime configuration for an Igor node.
type Config struct {
	// NodeID is the unique identifier for this node.
	// If empty, a random UUID will be generated.
	NodeID string

	// ListenAddress is the multiaddr for the P2P listener.
	ListenAddress string

	// PricePerSecond is the cost in microcents per second of runtime.
	// 1 currency unit = 1,000,000 microcents.
	PricePerSecond int64

	// BootstrapPeers is a list of multiaddrs to connect to on startup.
	BootstrapPeers []string

	// CheckpointDir is the directory where agent checkpoints are stored.
	CheckpointDir string

	// ReplayWindowSize is the number of recent tick snapshots retained
	// for sliding replay verification (CM-4). Default: 16.
	ReplayWindowSize int

	// VerifyInterval is the number of ticks between self-verification passes.
	// 0 disables periodic verification. Default: 5.
	VerifyInterval int

	// ReplayMode controls when replay verification runs.
	// "off" = no verification; "periodic" = self-verify every VerifyInterval ticks;
	// "on-migrate" = verify only on incoming migration; "full" = both (default).
	ReplayMode string

	// ReplayCostLog enables logging of replay compute duration for economic observability.
	ReplayCostLog bool

	// ReplayOnDivergence controls the escalation policy when replay verification
	// detects state divergence. Valid values:
	//   "log"       - Log error and continue (default)
	//   "pause"     - Stop ticking, preserve checkpoint, exit cleanly
	//   "intensify" - Temporarily reduce VerifyInterval to 1 (verify every tick)
	//   "migrate"   - Log intent to migrate (full implementation requires peer selection)
	ReplayOnDivergence string

	// LeaseDuration is the validity period for authority leases.
	// 0 disables leases (backward compatible). Default: 60s.
	LeaseDuration time.Duration

	// LeaseRenewalWindow is the fraction of lease duration at which
	// automatic renewal triggers. Must be in (0, 1). Default: 0.5.
	LeaseRenewalWindow float64

	// LeaseGracePeriod is the additional time after lease expiry before
	// the agent transitions to RECOVERY_REQUIRED. Default: 10s.
	LeaseGracePeriod time.Duration

	// MigrationMaxRetries is the maximum number of retry attempts per
	// migration target before moving to the next candidate. Default: 3.
	MigrationMaxRetries int

	// MigrationRetryDelay is the initial backoff delay between migration
	// retry attempts. Subsequent attempts use exponential backoff. Default: 1s.
	MigrationRetryDelay time.Duration
}

// Load returns a Config with default values applied.
func Load() (*Config, error) {
	cfg := &Config{
		NodeID:              generateNodeID(),
		ListenAddress:       "/ip4/0.0.0.0/tcp/4001",
		PricePerSecond:      1000, // 0.001 currency units = 1000 microcents
		BootstrapPeers:      []string{},
		CheckpointDir:       "./checkpoints",
		ReplayWindowSize:    16,
		VerifyInterval:      5,
		ReplayMode:          "full",
		ReplayCostLog:       false,
		ReplayOnDivergence:  "log",
		LeaseDuration:       60 * time.Second,
		LeaseRenewalWindow:  0.5,
		LeaseGracePeriod:    10 * time.Second,
		MigrationMaxRetries: 3,
		MigrationRetryDelay: 1 * time.Second,
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}

// Validate checks config invariants.
func (c *Config) Validate() error {
	if c.PricePerSecond <= 0 {
		return fmt.Errorf("PricePerSecond must be positive, got %d", c.PricePerSecond)
	}
	if c.ReplayWindowSize < 0 {
		return fmt.Errorf("ReplayWindowSize must be non-negative, got %d", c.ReplayWindowSize)
	}
	if c.VerifyInterval < 0 {
		return fmt.Errorf("VerifyInterval must be non-negative, got %d", c.VerifyInterval)
	}
	validModes := map[string]bool{"off": true, "periodic": true, "on-migrate": true, "full": true}
	if !validModes[c.ReplayMode] {
		return fmt.Errorf("ReplayMode must be one of off/periodic/on-migrate/full, got %q", c.ReplayMode)
	}
	validPolicies := map[string]bool{"log": true, "pause": true, "intensify": true, "migrate": true}
	if !validPolicies[c.ReplayOnDivergence] {
		return fmt.Errorf("ReplayOnDivergence must be one of log/pause/intensify/migrate, got %q", c.ReplayOnDivergence)
	}
	// Migration retry validation
	if c.MigrationMaxRetries < 0 {
		return fmt.Errorf("MigrationMaxRetries must be non-negative, got %d", c.MigrationMaxRetries)
	}
	if c.MigrationRetryDelay < 0 {
		return fmt.Errorf("MigrationRetryDelay must be non-negative, got %v", c.MigrationRetryDelay)
	}
	// Lease validation (LeaseDuration == 0 disables leases)
	if c.LeaseDuration < 0 {
		return fmt.Errorf("LeaseDuration must be non-negative, got %v", c.LeaseDuration)
	}
	if c.LeaseDuration > 0 {
		if c.LeaseRenewalWindow <= 0 || c.LeaseRenewalWindow >= 1.0 {
			return fmt.Errorf("LeaseRenewalWindow must be in (0, 1), got %f", c.LeaseRenewalWindow)
		}
		if c.LeaseGracePeriod < 0 {
			return fmt.Errorf("LeaseGracePeriod must be non-negative, got %v", c.LeaseGracePeriod)
		}
	}
	return nil
}

// generateNodeID creates a random UUID for the node if one is not provided.
func generateNodeID() string {
	return uuid.New().String()
}
