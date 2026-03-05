package config

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TickTimeout is the maximum duration for a single agent tick.
// Used by agent execution, replay verification, and the simulator.
const TickTimeout = 100 * time.Millisecond

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
}

// Load returns a Config with default values applied.
func Load() (*Config, error) {
	cfg := &Config{
		NodeID:             generateNodeID(),
		ListenAddress:      "/ip4/0.0.0.0/tcp/4001",
		PricePerSecond:     1000, // 0.001 currency units = 1000 microcents
		BootstrapPeers:     []string{},
		CheckpointDir:      "./checkpoints",
		ReplayWindowSize:   16,
		VerifyInterval:     5,
		ReplayMode:         "full",
		ReplayCostLog:      false,
		ReplayOnDivergence: "log",
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
	return nil
}

// generateNodeID creates a random UUID for the node if one is not provided.
func generateNodeID() string {
	return uuid.New().String()
}
