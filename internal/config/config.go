package config

import (
	"github.com/google/uuid"
)

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
}

// Load returns a Config with default values applied.
func Load() (*Config, error) {
	cfg := &Config{
		NodeID:           generateNodeID(),
		ListenAddress:    "/ip4/0.0.0.0/tcp/4001",
		PricePerSecond:   1000, // 0.001 currency units = 1000 microcents
		BootstrapPeers:   []string{},
		CheckpointDir:    "./checkpoints",
		ReplayWindowSize: 16,
		VerifyInterval:   5,
		ReplayMode:       "full",
		ReplayCostLog:    false,
	}
	return cfg, nil
}

// generateNodeID creates a random UUID for the node if one is not provided.
func generateNodeID() string {
	return uuid.New().String()
}
