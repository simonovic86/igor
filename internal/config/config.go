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

	// PricePerSecond is the cost in arbitrary units per second of runtime.
	PricePerSecond float64

	// BootstrapPeers is a list of multiaddrs to connect to on startup.
	BootstrapPeers []string

	// CheckpointDir is the directory where agent checkpoints are stored.
	CheckpointDir string
}

// Load returns a Config with default values applied.
func Load() (*Config, error) {
	cfg := &Config{
		NodeID:         generateNodeID(),
		ListenAddress:  "/ip4/0.0.0.0/tcp/4001",
		PricePerSecond: 0.001,
		BootstrapPeers: []string{},
		CheckpointDir:  "./checkpoints",
	}
	return cfg, nil
}

// generateNodeID creates a random UUID for the node if one is not provided.
func generateNodeID() string {
	return uuid.New().String()
}
