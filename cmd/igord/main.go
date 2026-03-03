package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/internal/config"
	"github.com/simonovic86/igor/internal/logging"
	"github.com/simonovic86/igor/internal/migration"
	"github.com/simonovic86/igor/internal/p2p"
	"github.com/simonovic86/igor/internal/replay"
	"github.com/simonovic86/igor/internal/runtime"
	"github.com/simonovic86/igor/internal/storage"
	"github.com/simonovic86/igor/pkg/budget"
)

func main() {
	// Parse CLI flags
	runAgent := flag.String("run-agent", "", "Path to WASM agent to run locally")
	budgetFlag := flag.Float64("budget", 1.0, "Initial budget for agent execution")
	manifestPath := flag.String("manifest", "", "Path to capability manifest JSON (default: <agent>.manifest.json)")
	migrateAgent := flag.String("migrate-agent", "", "Agent ID to migrate")
	targetPeer := flag.String("to", "", "Target peer multiaddr for migration")
	wasmPath := flag.String("wasm", "", "WASM binary path for migration")
	replayWindow := flag.Int("replay-window", 0, "Number of recent tick snapshots to retain for verification (0 = use config default)")
	verifyInterval := flag.Int("verify-interval", 0, "Ticks between self-verification passes (0 = use config default)")
	replayMode := flag.String("replay-mode", "", "Replay verification mode: off, periodic, on-migrate, full (default: full)")
	replayCostLog := flag.Bool("replay-cost-log", false, "Log replay compute duration for economic observability")
	flag.Parse()

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Apply CLI overrides
	if *replayWindow > 0 {
		cfg.ReplayWindowSize = *replayWindow
	}
	if *verifyInterval > 0 {
		cfg.VerifyInterval = *verifyInterval
	}
	if *replayMode != "" {
		cfg.ReplayMode = *replayMode
	}
	if *replayCostLog {
		cfg.ReplayCostLog = true
	}

	// Initialize logging
	logger := logging.NewLogger()

	// Print startup banner
	logging.Info(logger, "Igor Node starting...")
	logging.Info(logger, fmt.Sprintf("NodeID: %s", cfg.NodeID))

	// Initialize P2P node
	node, err := p2p.NewNode(ctx, cfg, logger)
	if err != nil {
		logging.Error(logger, "Failed to create P2P node", "error", err)
		os.Exit(1)
	}
	defer node.Close()

	// Create storage provider
	storageProvider, err := storage.NewFSProvider(cfg.CheckpointDir, logger)
	if err != nil {
		logging.Error(logger, "Failed to create storage provider", "error", err)
		os.Exit(1)
	}

	// Create WASM runtime engine for migration service
	engine, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		logging.Error(logger, "Failed to create runtime engine", "error", err)
		os.Exit(1)
	}
	defer engine.Close(ctx)

	// Initialize migration service
	migrationSvc := migration.NewService(node.Host, engine, storageProvider, cfg.ReplayMode, cfg.ReplayCostLog, logger)

	// If --migrate-agent flag is provided, perform migration
	if *migrateAgent != "" {
		if *targetPeer == "" {
			logging.Error(logger, "Migration requires --to flag with target peer address")
			os.Exit(1)
		}
		if *wasmPath == "" {
			logging.Error(logger, "Migration requires --wasm flag with WASM binary path")
			os.Exit(1)
		}

		logging.Info(logger, "Initiating agent migration",
			"agent_id", *migrateAgent,
			"target", *targetPeer,
		)

		if err := migrationSvc.MigrateAgent(ctx, *migrateAgent, *wasmPath, *targetPeer); err != nil {
			logging.Error(logger, "Migration failed", "error", err)
			os.Exit(1)
		}

		logging.Info(logger, "Migration completed successfully")
		return
	}

	// If --run-agent flag is provided, run agent locally
	if *runAgent != "" {
		budgetMicrocents := budget.FromFloat(*budgetFlag)
		if err := runLocalAgent(ctx, cfg, engine, storageProvider, *runAgent, budgetMicrocents, *manifestPath, migrationSvc, logger); err != nil {
			logging.Error(logger, "Failed to run agent", "error", err)
			os.Exit(1)
		}
		return
	}

	logging.Info(logger, "Igor Node ready")

	// Block until interrupted
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logging.Info(logger, "Igor Node shutting down...")
}

// runLocalAgent loads and executes an agent locally with tick loop and checkpointing.
func runLocalAgent(
	ctx context.Context,
	cfg *config.Config,
	engine *runtime.Engine,
	storageProvider storage.Provider,
	wasmPath string,
	budgetMicrocents int64,
	manifestPathFlag string,
	migrationSvc *migration.Service,
	logger *slog.Logger,
) error {
	// Load manifest from file
	mPath := manifestPathFlag
	if mPath == "" {
		// Default: look for <agent>.manifest.json alongside the WASM file
		mPath = wasmPath[:len(wasmPath)-len(".wasm")] + ".manifest.json"
	}
	manifestData, err := os.ReadFile(mPath)
	if err != nil {
		// No manifest file — backward compatible, empty capabilities
		manifestData = []byte("{}")
		logger.Info("No manifest file found, using empty capabilities",
			"expected_path", mPath,
		)
	} else {
		logger.Info("Manifest loaded", "path", mPath)
	}

	// Load agent with budget and manifest
	instance, err := agent.LoadAgent(
		ctx,
		engine,
		wasmPath,
		"local-agent",
		storageProvider,
		budgetMicrocents,
		cfg.PricePerSecond,
		manifestData,
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to load agent: %w", err)
	}
	defer instance.Close(ctx)

	// Configure replay window size
	instance.SetReplayWindowSize(cfg.ReplayWindowSize)

	// Register agent with migration service
	migrationSvc.RegisterAgent("local-agent", instance)

	logger.Info("Agent loaded with budget",
		"budget", budget.Format(budgetMicrocents),
		"price_per_second", budget.Format(cfg.PricePerSecond),
	)

	// Initialize agent
	if err := instance.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize agent: %w", err)
	}

	// Load checkpoint from storage if it exists
	if err := instance.LoadCheckpointFromStorage(ctx); err != nil {
		logger.Error("Failed to load checkpoint", "error", err)
		// Continue anyway with fresh state
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create replay engine for CM-4 verification
	replayEngine := replay.NewEngine(logger)

	// Tick loop
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	checkpointTicker := time.NewTicker(5 * time.Second)
	defer checkpointTicker.Stop()

	// Self-verification state (CM-4: Observation Determinism)
	var ticksSinceVerify int
	var lastVerifiedTick uint64

	periodicVerify := cfg.ReplayMode == "periodic" || cfg.ReplayMode == "full"

	logger.Info("Starting agent tick loop",
		"replay_window", cfg.ReplayWindowSize,
		"verify_interval", cfg.VerifyInterval,
		"replay_mode", cfg.ReplayMode,
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-sigChan:
			logger.Info("Received interrupt signal, checkpointing and shutting down...")

			// Final checkpoint before exit
			if err := instance.SaveCheckpointToStorage(ctx); err != nil {
				logger.Error("Failed to save checkpoint on shutdown", "error", err)
			}
			return nil

		case <-ticker.C:
			if tickErr := safeTick(ctx, instance); tickErr != nil {
				return handleTickFailure(ctx, instance, tickErr, logger)
			}

			ticksSinceVerify++
			if periodicVerify && cfg.VerifyInterval > 0 && ticksSinceVerify >= cfg.VerifyInterval {
				ticksSinceVerify = 0
				lastVerifiedTick = verifyNextTick(ctx, instance, replayEngine, lastVerifiedTick, cfg.ReplayCostLog, logger)
			}

		case <-checkpointTicker.C:
			// Periodic checkpoint
			if err := instance.SaveCheckpointToStorage(ctx); err != nil {
				logger.Error("Failed to save checkpoint", "error", err)
			}
		}
	}
}

// handleTickFailure logs the failure reason, saves a final checkpoint, and returns the error.
func handleTickFailure(ctx context.Context, instance *agent.Instance, tickErr error, logger *slog.Logger) error {
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

// safeTick executes one tick with panic recovery (EI-6: Safety Over Liveness).
// A WASM trap or runtime bug must not crash the node.
func safeTick(ctx context.Context, instance *agent.Instance) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tick panicked: %v", r)
		}
	}()
	return instance.Tick(ctx)
}

// verifyNextTick replays the oldest unverified tick in the replay window.
// Returns the tick number of the verified tick (for tracking), or lastVerified
// if nothing was verified.
// Replay failures are logged but do not halt execution (EI-6: Safety Over Liveness).
func verifyNextTick(
	ctx context.Context,
	instance *agent.Instance,
	replayEngine *replay.Engine,
	lastVerified uint64,
	logCost bool,
	logger *slog.Logger,
) uint64 {
	for _, snap := range instance.ReplayWindow {
		if snap.TickNumber <= lastVerified {
			continue
		}
		if snap.TickLog == nil {
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
			return snap.TickNumber
		}

		// Hash-based post-state comparison (IMPROVEMENTS #2).
		replayedHash := sha256.Sum256(result.ReplayedState)
		if replayedHash != snap.PostStateHash {
			logger.Error("Replay divergence detected",
				"tick", result.TickNumber,
				"state_bytes", len(result.ReplayedState),
			)
			return snap.TickNumber
		}

		attrs := []any{
			"tick", result.TickNumber,
			"state_bytes", len(result.ReplayedState),
		}
		if logCost {
			attrs = append(attrs, "replay_duration", result.Duration)
		}
		logger.Info("Replay verified", attrs...)
		return snap.TickNumber
	}
	return lastVerified
}
