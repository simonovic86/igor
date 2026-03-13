// SPDX-License-Identifier: Apache-2.0

// Package runner provides the tick loop orchestration logic extracted from
// cmd/igord/main.go. This enables direct unit testing of verification,
// divergence escalation, and tick lifecycle management.
package runner

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/pkg/manifest"
)

// DivergenceAction indicates what the tick loop should do after replay verification.
type DivergenceAction int

const (
	DivergenceNone      DivergenceAction = iota // no divergence detected
	DivergenceLog                               // log and continue
	DivergencePause                             // stop ticking, preserve checkpoint
	DivergenceIntensify                         // increase verify frequency
	DivergenceMigrate                           // trigger migration
)

// MigrationTrigger is called when divergence escalation requests migration.
// The implementation should attempt migration with retry and fallback.
// Returns nil if migration succeeded, error if it failed.
type MigrationTrigger func(ctx context.Context, agentID string) error

// SafeTick executes one tick with panic recovery (EI-6: Safety Over Liveness).
// A WASM trap or runtime bug must not crash the node.
func SafeTick(ctx context.Context, instance *agent.Instance) (hasMore bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tick panicked: %v", r)
		}
	}()
	return instance.Tick(ctx)
}

// HandleTickFailure logs the failure reason, saves a final checkpoint, and returns the error.
func HandleTickFailure(ctx context.Context, instance *agent.Instance, tickErr error, logger *slog.Logger) error {
	if instance.Budget <= 0 {
		logger.Info("Agent budget exhausted, terminating",
			"agent_id", instance.AgentID,
			"reason", "budget_exhausted",
		)
	} else {
		logger.Error("Tick failed", "error", tickErr)
	}
	trySaveCheckpoint(ctx, instance, logger)
	return tickErr
}

// trySaveCheckpoint attempts to save a checkpoint, logging errors. Guards against
// nil instances or missing storage (e.g., during unit tests).
func trySaveCheckpoint(ctx context.Context, instance *agent.Instance, logger *slog.Logger) {
	if instance == nil || instance.Storage == nil {
		return
	}
	if err := instance.SaveCheckpointToStorage(ctx); err != nil {
		logger.Error("Failed to save checkpoint", "error", err)
	}
}

// EscalationForPolicy maps a policy string to a DivergenceAction.
func EscalationForPolicy(policy string) DivergenceAction {
	switch policy {
	case "pause":
		return DivergencePause
	case "intensify":
		return DivergenceIntensify
	case "migrate":
		return DivergenceMigrate
	default:
		return DivergenceLog
	}
}

// LoadManifestData reads the manifest file for the given WASM path and flags.
// Returns empty JSON capabilities if no manifest is found.
func LoadManifestData(wasmPath, manifestPathFlag string, logger *slog.Logger) []byte {
	return manifest.LoadSidecarData(wasmPath, manifestPathFlag, logger)
}
