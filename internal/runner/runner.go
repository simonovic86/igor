// SPDX-License-Identifier: Apache-2.0

// Package runner provides the tick loop orchestration logic extracted from
// cmd/igord/main.go. This enables direct unit testing of verification,
// divergence escalation, and tick lifecycle management.
package runner

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"log/slog"

	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/internal/config"
	"github.com/simonovic86/igor/internal/p2p"
	"github.com/simonovic86/igor/internal/replay"
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

// CheckAndRenewLease validates the lease before a tick and renews if needed.
// Returns nil if ticking is allowed, or an error if the lease has expired.
// No-op if leases are disabled (instance.Lease == nil).
func CheckAndRenewLease(instance *agent.Instance, logger *slog.Logger) error {
	if instance.Lease == nil {
		return nil
	}
	if err := instance.Lease.ValidateForTick(); err != nil {
		return err
	}
	if instance.Lease.NeedsRenewal() {
		if err := instance.Lease.Renew(); err != nil {
			logger.Error("Lease renewal failed", "error", err)
		} else {
			logger.Info("Lease renewed",
				"epoch", instance.Lease.Epoch,
				"expiry", instance.Lease.Expiry,
			)
		}
	}
	return nil
}

// HandleLeaseExpiry saves a final checkpoint and returns the lease error.
// EI-6: safety over liveness — we stop rather than tick without authority.
func HandleLeaseExpiry(ctx context.Context, instance *agent.Instance, leaseErr error, logger *slog.Logger) error {
	logger.Error("Lease expired, halting agent",
		"agent_id", instance.AgentID,
		"error", leaseErr,
	)
	trySaveCheckpoint(ctx, instance, logger)
	return leaseErr
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

// MigrationTrigger is called when divergence escalation requests migration.
// The implementation should attempt migration with retry and fallback.
// Returns nil if migration succeeded, error if it failed.
type MigrationTrigger func(ctx context.Context, agentID string) error

// HandleDivergenceAction acts on the escalation policy returned by VerifyNextTick.
// migrateFn is optional — passing nil preserves the existing "fall through to pause"
// behavior for DivergenceMigrate.
// Returns true if the tick loop should exit.
func HandleDivergenceAction(ctx context.Context, instance *agent.Instance, cfg *config.Config, action DivergenceAction, migrateFn MigrationTrigger, logger *slog.Logger) bool {
	switch action {
	case DivergencePause:
		logger.Info("Agent paused due to replay divergence (EI-6), saving checkpoint")
		trySaveCheckpoint(ctx, instance, logger)
		return true
	case DivergenceIntensify:
		logger.Info("Verification frequency intensified to every tick")
		cfg.VerifyInterval = 1
	case DivergenceMigrate:
		if migrateFn != nil {
			logger.Info("Divergence-triggered migration starting",
				"agent_id", instance.AgentID,
			)
			if err := migrateFn(ctx, instance.AgentID); err != nil {
				logger.Error("Divergence migration failed, pausing",
					"error", err,
				)
			} else {
				logger.Info("Divergence migration succeeded")
				return true // exit tick loop — agent is on another node
			}
		} else {
			logger.Info("Migration escalation triggered — pausing (no migration function available)")
		}
		trySaveCheckpoint(ctx, instance, logger)
		return true
	}
	return false
}

// AttemptLeaseRecovery tries to recover from RECOVERY_REQUIRED state.
// For v0, this is a local-only operation: the node re-grants itself a fresh
// lease with an incremented major version, ensuring stale leases are superseded.
// Returns nil on success, error if recovery is not possible.
func AttemptLeaseRecovery(ctx context.Context, instance *agent.Instance, logger *slog.Logger) error {
	if instance.Lease == nil {
		return fmt.Errorf("lease recovery: no lease configured")
	}
	if err := instance.Lease.Recover(); err != nil {
		return fmt.Errorf("lease recovery: %w", err)
	}
	logger.Info("Lease recovered",
		"agent_id", instance.AgentID,
		"epoch", instance.Lease.Epoch,
		"expiry", instance.Lease.Expiry,
	)
	trySaveCheckpoint(ctx, instance, logger)
	return nil
}

// VerifyNextTick replays the oldest unverified tick in the replay window.
// Returns the tick number of the verified tick (for tracking) and an escalation
// action if divergence is detected. Returns DivergenceNone when verification passes.
func VerifyNextTick(
	ctx context.Context,
	instance *agent.Instance,
	replayEngine *replay.Engine,
	lastVerified uint64,
	logCost bool,
	policy string,
	logger *slog.Logger,
) (uint64, DivergenceAction) {
	for _, snap := range instance.ReplayWindow {
		if snap.TickNumber <= lastVerified {
			continue
		}
		if snap.TickLog == nil || len(snap.TickLog.Entries) == 0 {
			continue
		}

		// Pass nil expectedState — hash-based verification (IMPROVEMENTS #2).
		result := replayEngine.ReplayTick(
			ctx,
			instance.WASMBytes,
			instance.Manifest,
			snap.PreState,
			snap.TickLog,
			nil,
		)

		if result.Error != nil {
			logger.Error("Replay verification failed",
				"tick", result.TickNumber,
				"error", result.Error,
			)
			return snap.TickNumber, EscalationForPolicy(policy)
		}

		// Hash-based post-state comparison (IMPROVEMENTS #2).
		replayedHash := sha256.Sum256(result.ReplayedState)
		if replayedHash != snap.PostStateHash {
			logger.Error("Replay divergence detected",
				"tick", result.TickNumber,
				"state_bytes", len(result.ReplayedState),
			)
			return snap.TickNumber, EscalationForPolicy(policy)
		}

		attrs := []any{
			"tick", result.TickNumber,
			"state_bytes", len(result.ReplayedState),
		}
		if logCost {
			attrs = append(attrs, "replay_duration", result.Duration)
		}
		logger.Info("Replay verified", attrs...)
		return snap.TickNumber, DivergenceNone
	}
	return lastVerified, DivergenceNone
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

// ExtractSigningKey returns the Ed25519 private key and peer ID from the node,
// or nil/"" if the key is not available or not Ed25519.
func ExtractSigningKey(node *p2p.Node) (ed25519.PrivateKey, string) {
	privKey := node.Host.Peerstore().PrivKey(node.Host.ID())
	if privKey == nil {
		return nil, ""
	}
	raw, err := privKey.Raw()
	if err != nil || len(raw) != ed25519.PrivateKeySize {
		return nil, ""
	}
	return ed25519.PrivateKey(raw), node.Host.ID().String()
}
