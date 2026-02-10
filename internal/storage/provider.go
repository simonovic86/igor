package storage

import (
	"context"
	"errors"
)

// ErrCheckpointNotFound is returned when a checkpoint does not exist.
var ErrCheckpointNotFound = errors.New("checkpoint not found")

// Provider defines the interface for agent checkpoint persistence.
// Implementations must be safe for concurrent use.
type Provider interface {
	// SaveCheckpoint persists an agent's checkpoint state.
	// Must be atomic - either the entire state is saved or none of it.
	SaveCheckpoint(ctx context.Context, agentID string, state []byte) error

	// LoadCheckpoint retrieves an agent's checkpoint state.
	// Returns ErrCheckpointNotFound if no checkpoint exists for the agent.
	LoadCheckpoint(ctx context.Context, agentID string) ([]byte, error)

	// DeleteCheckpoint removes an agent's checkpoint state.
	// Does not return error if checkpoint doesn't exist (idempotent).
	DeleteCheckpoint(ctx context.Context, agentID string) error
}
