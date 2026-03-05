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
	"os"
	"strings"

	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/internal/config"
	"github.com/simonovic86/igor/internal/p2p"
	"github.com/simonovic86/igor/internal/replay"
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

// HandleTickFailure logs the failure reason, saves a final checkpoint, and returns the error.
func HandleTickFailure(ctx context.Context, instance *agent.Instance, tickErr error, logger *slog.Logger) error {
	if instance.Budget <= 0 {
		logger.Info("Agent budget exhausted, terminating",
			"agent_id", "local-agent",
			"reason", "budget_exhausted",
		)
	} else {
		logger.Error("Tick failed", "error", tickErr)
	}
	if err := instance.SaveCheckpointToStorage(ctx); err != nil {
		logger.Error("Failed to save checkpoint on termination", "error", err)
	}
	return tickErr
}

// HandleDivergenceAction acts on the escalation policy returned by VerifyNextTick.
// Returns true if the tick loop should exit.
func HandleDivergenceAction(ctx context.Context, instance *agent.Instance, cfg *config.Config, action DivergenceAction, logger *slog.Logger) bool {
	switch action {
	case DivergencePause:
		logger.Info("Agent paused due to replay divergence (EI-6), saving checkpoint")
		if err := instance.SaveCheckpointToStorage(ctx); err != nil {
			logger.Error("Failed to save checkpoint on pause", "error", err)
		}
		return true
	case DivergenceIntensify:
		logger.Info("Verification frequency intensified to every tick")
		cfg.VerifyInterval = 1
	case DivergenceMigrate:
		// Full migration-trigger requires peer selection; fall through to pause.
		logger.Info("Migration escalation triggered — pausing (peer selection not yet implemented)")
		if err := instance.SaveCheckpointToStorage(ctx); err != nil {
			logger.Error("Failed to save checkpoint on migrate-pause", "error", err)
		}
		return true
	}
	return false
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
	mPath := manifestPathFlag
	if mPath == "" {
		// Default: look for <agent>.manifest.json alongside the WASM file
		if strings.HasSuffix(wasmPath, ".wasm") {
			mPath = strings.TrimSuffix(wasmPath, ".wasm") + ".manifest.json"
		}
	}
	manifestData, err := os.ReadFile(mPath)
	if err != nil {
		logger.Info("No manifest file found, using empty capabilities",
			"expected_path", mPath,
		)
		return []byte("{}")
	}
	logger.Info("Manifest loaded", "path", mPath)
	return manifestData
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
