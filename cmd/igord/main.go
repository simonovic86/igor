// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/internal/authority"
	"github.com/simonovic86/igor/internal/config"
	"github.com/simonovic86/igor/internal/inspector"
	"github.com/simonovic86/igor/internal/logging"
	"github.com/simonovic86/igor/internal/migration"
	"github.com/simonovic86/igor/internal/p2p"
	"github.com/simonovic86/igor/internal/pricing"
	"github.com/simonovic86/igor/internal/replay"
	"github.com/simonovic86/igor/internal/runner"
	"github.com/simonovic86/igor/internal/runtime"
	"github.com/simonovic86/igor/internal/settlement"
	"github.com/simonovic86/igor/internal/simulator"
	"github.com/simonovic86/igor/internal/storage"
	"github.com/simonovic86/igor/pkg/budget"
	"github.com/simonovic86/igor/pkg/identity"
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
	replayOnDivergence := flag.String("replay-on-divergence", "", "Escalation policy on replay divergence: log, pause, intensify, migrate (default: log)")
	inspectCheckpoint := flag.String("inspect-checkpoint", "", "Path to checkpoint file to inspect")
	inspectWASM := flag.String("inspect-wasm", "", "Optional WASM binary to verify against checkpoint hash")
	simulate := flag.Bool("simulate", false, "Run agent in local simulator mode (no P2P)")
	simTicks := flag.Int("ticks", 0, "Number of ticks to simulate (0 = until budget exhausted)")
	simVerify := flag.Bool("verify", false, "Per-tick replay verification during simulation")
	simDeterministic := flag.Bool("deterministic", false, "Use fixed clock and seeded rand for reproducible simulation")
	simSeed := flag.Uint64("seed", 0, "Random seed for deterministic simulation")
	leaseDuration := flag.Duration("lease-duration", 60*time.Second, "Lease validity period (0 = disabled)")
	leaseGrace := flag.Duration("lease-grace", 10*time.Second, "Grace period after lease expiry")
	flag.Parse()

	// Checkpoint inspector — standalone, no config/P2P/engine needed
	if *inspectCheckpoint != "" {
		runInspector(*inspectCheckpoint, *inspectWASM)
		return
	}

	// Local simulator — standalone, no config/P2P needed
	if *simulate && *runAgent != "" {
		runSimulator(*runAgent, *manifestPath, *budgetFlag, *simTicks, *simVerify, *simDeterministic, *simSeed)
		return
	}

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
	if *replayOnDivergence != "" {
		cfg.ReplayOnDivergence = *replayOnDivergence
	}
	cfg.LeaseDuration = *leaseDuration
	cfg.LeaseGracePeriod = *leaseGrace

	// Initialize logging
	logger := logging.NewLogger()

	// Print startup banner
	logger.Info("Igor Node starting...")
	logger.Info("NodeID: " + cfg.NodeID)

	// Initialize P2P node
	node, err := p2p.NewNode(ctx, cfg, logger)
	if err != nil {
		logger.Error("Failed to create P2P node", "error", err)
		os.Exit(1)
	}
	defer node.Close()

	// Create storage provider
	storageProvider, err := storage.NewFSProvider(cfg.CheckpointDir, logger)
	if err != nil {
		logger.Error("Failed to create storage provider", "error", err)
		os.Exit(1)
	}

	// Create WASM runtime engine for migration service
	engine, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		logger.Error("Failed to create runtime engine", "error", err)
		os.Exit(1)
	}
	defer engine.Close(ctx)

	// Initialize migration service
	leaseCfg := authority.LeaseConfig{
		Duration:      cfg.LeaseDuration,
		RenewalWindow: cfg.LeaseRenewalWindow,
		GracePeriod:   cfg.LeaseGracePeriod,
	}
	migrationSvc := migration.NewService(node.Host, engine, storageProvider, cfg.ReplayMode, cfg.ReplayCostLog, cfg.PricePerSecond, leaseCfg, logger)

	// Initialize pricing service for inter-node price discovery
	_ = pricing.NewService(node.Host, cfg.PricePerSecond, logger)

	// If --migrate-agent flag is provided, perform migration
	if *migrateAgent != "" {
		if *targetPeer == "" {
			logger.Error("Migration requires --to flag with target peer address")
			os.Exit(1)
		}
		if *wasmPath == "" {
			logger.Error("Migration requires --wasm flag with WASM binary path")
			os.Exit(1)
		}

		logger.Info("Initiating agent migration",
			"agent_id", *migrateAgent,
			"target", *targetPeer,
		)

		if err := migrationSvc.MigrateAgent(ctx, *migrateAgent, *wasmPath, *targetPeer); err != nil {
			logger.Error("Migration failed", "error", err)
			os.Exit(1)
		}

		logger.Info("Migration completed successfully")
		return
	}

	// If --run-agent flag is provided, run agent locally
	if *runAgent != "" {
		budgetMicrocents := budget.FromFloat(*budgetFlag)
		if err := runLocalAgent(ctx, cfg, engine, storageProvider, *runAgent, budgetMicrocents, *manifestPath, migrationSvc, node, logger); err != nil {
			logger.Error("Failed to run agent", "error", err)
			os.Exit(1)
		}
		return
	}

	logger.Info("Igor Node ready")

	// Block until interrupted
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Igor Node shutting down...")
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
	node *p2p.Node,
	logger *slog.Logger,
) error {
	manifestData := runner.LoadManifestData(wasmPath, manifestPathFlag, logger)

	signingKey, nodeID := runner.ExtractSigningKey(node)

	// Load or generate agent cryptographic identity for signed checkpoint lineage.
	agentIdent, err := loadOrGenerateIdentity(ctx, storageProvider, "local-agent", logger)
	if err != nil {
		return fmt.Errorf("agent identity: %w", err)
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
		signingKey,
		nodeID,
		agentIdent,
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to load agent: %w", err)
	}
	defer instance.Close(ctx)

	if err := initLocalAgent(ctx, cfg, instance, migrationSvc, budgetMicrocents, logger); err != nil {
		return err
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create replay engine for CM-4 verification
	replayEngine := replay.NewEngine(logger)
	defer replayEngine.Close(ctx)

	// Adaptive tick loop constants.
	const (
		normalTickInterval = 1 * time.Second
		minTickInterval    = 10 * time.Millisecond
	)

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

	tickTimer := time.NewTimer(normalTickInterval)
	defer tickTimer.Stop()

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

		case <-tickTimer.C:
			// Pre-tick lease validation (EI-6: safety over liveness)
			if leaseErr := runner.CheckAndRenewLease(instance, logger); leaseErr != nil {
				return runner.HandleLeaseExpiry(ctx, instance, leaseErr, logger)
			}

			hasMoreWork, tickErr := runner.SafeTick(ctx, instance)
			if tickErr != nil {
				return runner.HandleTickFailure(ctx, instance, tickErr, logger)
			}

			// Adaptive tick scheduling: fast-path if agent has more work.
			if hasMoreWork {
				tickTimer.Reset(minTickInterval)
			} else {
				tickTimer.Reset(normalTickInterval)
			}

			ticksSinceVerify++
			if periodicVerify && cfg.VerifyInterval > 0 && ticksSinceVerify >= cfg.VerifyInterval {
				ticksSinceVerify = 0
				var action runner.DivergenceAction
				lastVerifiedTick, action = runner.VerifyNextTick(ctx, instance, replayEngine, lastVerifiedTick, cfg.ReplayCostLog, cfg.ReplayOnDivergence, logger)
				if stop := runner.HandleDivergenceAction(ctx, instance, cfg, action, logger); stop {
					return nil
				}
			}

		case <-checkpointTicker.C:
			// Periodic checkpoint
			if err := instance.SaveCheckpointToStorage(ctx); err != nil {
				logger.Error("Failed to save checkpoint", "error", err)
			}
		}
	}
}

// initLocalAgent configures, initializes, and resumes a local agent instance.
func initLocalAgent(
	ctx context.Context,
	cfg *config.Config,
	instance *agent.Instance,
	migrationSvc *migration.Service,
	budgetMicrocents int64,
	logger *slog.Logger,
) error {
	instance.SetReplayWindowSize(cfg.ReplayWindowSize)
	instance.BudgetAdapter = settlement.NewMockAdapter(logger)
	migrationSvc.RegisterAgent("local-agent", instance)

	logger.Info("Agent loaded with budget",
		"budget", budget.Format(budgetMicrocents),
		"price_per_second", budget.Format(cfg.PricePerSecond),
	)

	if err := instance.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize agent: %w", err)
	}

	if err := instance.LoadCheckpointFromStorage(ctx); err != nil {
		logger.Error("Failed to load checkpoint", "error", err)
	}

	if cfg.LeaseDuration > 0 {
		leaseCfg := authority.LeaseConfig{
			Duration:      cfg.LeaseDuration,
			RenewalWindow: cfg.LeaseRenewalWindow,
			GracePeriod:   cfg.LeaseGracePeriod,
		}
		instance.Lease = authority.NewLease(leaseCfg)
		logger.Info("Lease granted",
			"epoch", instance.Lease.Epoch,
			"expiry", instance.Lease.Expiry,
			"duration", cfg.LeaseDuration,
		)
	}
	return nil
}

// runInspector parses and displays a checkpoint file.
func runInspector(checkpointPath, wasmPath string) {
	result, err := inspector.InspectFile(checkpointPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if wasmPath != "" {
		if verr := result.VerifyWASM(wasmPath); verr != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", verr)
		}
	}
	result.Print(os.Stdout)
}

// runSimulator executes an agent in local simulator mode (no P2P).
func runSimulator(wasmPath, manifestPath string, budgetVal float64, ticks int, verify, deterministic bool, seed uint64) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := logging.NewLogger()
	cfg := simulator.Config{
		WASMPath:      wasmPath,
		ManifestPath:  manifestPath,
		Budget:        budgetVal,
		Ticks:         ticks,
		Verify:        verify,
		Deterministic: deterministic,
		RandSeed:      seed,
	}
	result, err := simulator.Run(ctx, cfg, logger)
	if err != nil {
		logger.Error("Simulation failed", "error", err)
		os.Exit(1)
	}
	simulator.PrintSummary(result, logger)
	if len(result.Errors) > 0 {
		os.Exit(1)
	}
}

// loadOrGenerateIdentity loads an existing agent identity from storage,
// or generates a new one and persists it. The identity is the agent's Ed25519
// keypair used for signed checkpoint lineage (Task 13).
func loadOrGenerateIdentity(
	ctx context.Context,
	storageProvider storage.Provider,
	agentID string,
	logger *slog.Logger,
) (*identity.AgentIdentity, error) {
	data, err := storageProvider.LoadIdentity(ctx, agentID)
	if err == nil {
		id, parseErr := identity.UnmarshalBinary(data)
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

	id, err := identity.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate identity: %w", err)
	}

	if err := storageProvider.SaveIdentity(ctx, agentID, id.MarshalBinary()); err != nil {
		return nil, fmt.Errorf("save identity: %w", err)
	}

	logger.Info("Agent identity generated and saved", "agent_id", agentID)
	return id, nil
}
