// SPDX-License-Identifier: Apache-2.0

// Package storage provides checkpoint persistence abstraction for survivable agents.
// Implements atomic write guarantees and storage provider interfaces enabling
// agent state to persist across infrastructure failures and migrations.
package storage

import (
	"context"
	"errors"
)

// ErrCheckpointNotFound is returned when a checkpoint does not exist.
var ErrCheckpointNotFound = errors.New("checkpoint not found")

// ErrReceiptsNotFound is returned when receipts do not exist.
var ErrReceiptsNotFound = errors.New("receipts not found")

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

	// SaveReceipts persists an agent's serialized receipt chain.
	// Must be atomic - either the entire data is saved or none of it.
	SaveReceipts(ctx context.Context, agentID string, data []byte) error

	// LoadReceipts retrieves an agent's serialized receipt chain.
	// Returns ErrReceiptsNotFound if no receipts exist for the agent.
	LoadReceipts(ctx context.Context, agentID string) ([]byte, error)

	// DeleteReceipts removes an agent's receipt chain.
	// Does not return error if receipts don't exist (idempotent).
	DeleteReceipts(ctx context.Context, agentID string) error
}
