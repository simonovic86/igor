// SPDX-License-Identifier: Apache-2.0

// Package research provides research-specific tick loop functions that depend
// on P2P, replay verification, and lease management. These are used by
// igord-lab but not by the product igord binary.
package research

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
	"github.com/simonovic86/igor/internal/runner"
)

// HandleDivergenceAction acts on the escalation policy returned by VerifyNextTick.
// migrateFn is optional — passing nil preserves the existing "fall through to pause"
// behavior for DivergenceMigrate.
// Returns true if the tick loop should exit.
func HandleDivergenceAction(ctx context.Context, instance *agent.Instance, cfg *config.Config, action runner.DivergenceAction, migrateFn runner.MigrationTrigger, logger *slog.Logger) bool {
	switch action {
	case runner.DivergencePause:
		logger.Info("Agent paused due to replay divergence (EI-6), saving checkpoint")
		trySaveCheckpoint(ctx, instance, logger)
		return true
	case runner.DivergenceIntensify:
		logger.Info("Verification frequency intensified to every tick")
		cfg.VerifyInterval = 1
	case runner.DivergenceMigrate:
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

// AttemptLeaseRecovery tries to recover from RECOVERY_REQUIRED state.
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
// action if divergence is detected.
func VerifyNextTick(
	ctx context.Context,
	instance *agent.Instance,
	replayEngine *replay.Engine,
	lastVerified uint64,
	logCost bool,
	policy string,
	logger *slog.Logger,
) (uint64, runner.DivergenceAction) {
	for _, snap := range instance.ReplayWindow {
		if snap.TickNumber <= lastVerified {
			continue
		}
		if snap.TickLog == nil || len(snap.TickLog.Entries) == 0 {
			continue
		}

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
			return snap.TickNumber, runner.EscalationForPolicy(policy)
		}

		replayedHash := sha256.Sum256(result.ReplayedState)
		if replayedHash != snap.PostStateHash {
			logger.Error("Replay divergence detected",
				"tick", result.TickNumber,
				"state_bytes", len(result.ReplayedState),
			)
			return snap.TickNumber, runner.EscalationForPolicy(policy)
		}

		attrs := []any{
			"tick", result.TickNumber,
			"state_bytes", len(result.ReplayedState),
		}
		if logCost {
			attrs = append(attrs, "replay_duration", result.Duration)
		}
		logger.Info("Replay verified", attrs...)
		return snap.TickNumber, runner.DivergenceNone
	}
	return lastVerified, runner.DivergenceNone
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

// trySaveCheckpoint attempts to save a checkpoint, logging errors.
func trySaveCheckpoint(ctx context.Context, instance *agent.Instance, logger *slog.Logger) {
	if instance == nil || instance.Storage == nil {
		return
	}
	if err := instance.SaveCheckpointToStorage(ctx); err != nil {
		logger.Error("Failed to save checkpoint", "error", err)
	}
}
