// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"context"
	"fmt"
	"log/slog"
)

// Store is the subset of storage.Provider needed for identity
// persistence. Both internal/storage.FSProvider and any future
// implementations satisfy this interface implicitly.
type Store interface {
	LoadIdentity(ctx context.Context, agentID string) ([]byte, error)
	SaveIdentity(ctx context.Context, agentID string, data []byte) error
}

// LoadOrGenerate loads an existing agent identity from storage, or generates
// a new one if none exists (or the stored data is corrupted).
func LoadOrGenerate(
	ctx context.Context,
	store Store,
	agentID string,
	logger *slog.Logger,
) (*AgentIdentity, error) {
	data, err := store.LoadIdentity(ctx, agentID)
	if err == nil {
		id, parseErr := UnmarshalBinary(data)
		if parseErr != nil {
			logger.Warn("Corrupted agent identity, generating new", "error", parseErr)
		} else {
			logger.Info("Agent identity loaded",
				"agent_id", agentID,
				"pub_key_size", len(id.PublicKey),
			)
			return id, nil
		}
	}

	id, err := Generate()
	if err != nil {
		return nil, fmt.Errorf("generate identity: %w", err)
	}

	if err := store.SaveIdentity(ctx, agentID, id.MarshalBinary()); err != nil {
		return nil, fmt.Errorf("save identity: %w", err)
	}

	logger.Info("Agent identity generated and saved", "agent_id", agentID)
	return id, nil
}
