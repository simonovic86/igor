// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
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
	// Handle subcommands before flag parsing.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "run":
			subcmdRun(os.Args[2:])
			return
		case "resume":
			subcmdResume(os.Args[2:])
			return
		case "verify":
			subcmdVerify(os.Args[2:])
			return
		case "inspect":
			subcmdInspect(os.Args[2:])
			return
		}
	}

	// Legacy flag-based CLI (backwards compatible).
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
	migrationRetries := flag.Int("migration-retries", 0, "Max retries per migration target (0 = use config default)")
	migrationRetryDelay := flag.Duration("migration-retry-delay", 0, "Initial backoff delay between migration retries (0 = use config default)")
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
	applyCLIOverrides(cfg, *replayWindow, *verifyInterval, *replayMode, *replayCostLog,
		*replayOnDivergence, *leaseDuration, *leaseGrace, *migrationRetries, *migrationRetryDelay)

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
			result, err := handleTick(ctx, instance, cfg, replayEngine, periodicVerify,
				&ticksSinceVerify, &lastVerifiedTick, logger)
			if err != nil {
				return err
			}
			switch result {
			case tickRecovered:
				tickTimer.Reset(normalTickInterval)
				continue
			case tickStopped:
				return nil
			case tickFastPath:
				tickTimer.Reset(minTickInterval)
			default:
				tickTimer.Reset(normalTickInterval)
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

// tickResult indicates the outcome of a single tick iteration.
type tickResult int

const (
	tickNormal    tickResult = iota // Normal tick, use standard interval.
	tickFastPath                    // Agent has more work, use fast interval.
	tickRecovered                   // Lease recovered, continue immediately.
	tickStopped                     // Divergence action requires stopping.
)

// handleTick processes a single tick: lease check, agent tick, verification.
func handleTick(
	ctx context.Context,
	instance *agent.Instance,
	cfg *config.Config,
	replayEngine *replay.Engine,
	periodicVerify bool,
	ticksSinceVerify *int,
	lastVerifiedTick *uint64,
	logger *slog.Logger,
) (tickResult, error) {
	// Pre-tick lease validation (EI-6: safety over liveness)
	if leaseErr := runner.CheckAndRenewLease(instance, logger); leaseErr != nil {
		if instance.Lease != nil && instance.Lease.State == authority.StateRecoveryRequired {
			if recoverErr := runner.AttemptLeaseRecovery(ctx, instance, logger); recoverErr == nil {
				return tickRecovered, nil
			}
		}
		return tickNormal, runner.HandleLeaseExpiry(ctx, instance, leaseErr, logger)
	}

	hasMoreWork, tickErr := runner.SafeTick(ctx, instance)
	if tickErr != nil {
		return tickNormal, runner.HandleTickFailure(ctx, instance, tickErr, logger)
	}

	*ticksSinceVerify++
	if periodicVerify && cfg.VerifyInterval > 0 && *ticksSinceVerify >= cfg.VerifyInterval {
		*ticksSinceVerify = 0
		var action runner.DivergenceAction
		*lastVerifiedTick, action = runner.VerifyNextTick(ctx, instance, replayEngine, *lastVerifiedTick, cfg.ReplayCostLog, cfg.ReplayOnDivergence, logger)
		if stop := runner.HandleDivergenceAction(ctx, instance, cfg, action, nil, logger); stop {
			return tickStopped, nil
		}
	}

	if hasMoreWork {
		return tickFastPath, nil
	}
	return tickNormal, nil
}

// applyCLIOverrides applies command-line flag values to the configuration.
func applyCLIOverrides(cfg *config.Config, replayWindow, verifyInterval int, replayMode string,
	replayCostLog bool, replayOnDivergence string, leaseDuration, leaseGrace time.Duration,
	migrationRetries int, migrationRetryDelay time.Duration) {
	if replayWindow > 0 {
		cfg.ReplayWindowSize = replayWindow
	}
	if verifyInterval > 0 {
		cfg.VerifyInterval = verifyInterval
	}
	if replayMode != "" {
		cfg.ReplayMode = replayMode
	}
	if replayCostLog {
		cfg.ReplayCostLog = true
	}
	if replayOnDivergence != "" {
		cfg.ReplayOnDivergence = replayOnDivergence
	}
	cfg.LeaseDuration = leaseDuration
	cfg.LeaseGracePeriod = leaseGrace
	if migrationRetries > 0 {
		cfg.MigrationMaxRetries = migrationRetries
	}
	if migrationRetryDelay > 0 {
		cfg.MigrationRetryDelay = migrationRetryDelay
	}
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

// subcmdRun implements "igord run <agent.wasm> [--budget N] [--manifest path]".
// Runs an agent locally with a simplified setup (no P2P, no migration).
func subcmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	budgetFlag := fs.Float64("budget", 1.0, "Initial budget for agent execution")
	manifestPath := fs.String("manifest", "", "Path to capability manifest JSON")
	checkpointDir := fs.String("checkpoint-dir", "checkpoints", "Directory for checkpoint storage")
	agentID := fs.String("agent-id", "", "Agent ID (default: derived from WASM filename)")
	leaseDuration := fs.Duration("lease-duration", 0, "Lease validity period (0 = disabled)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: igord run <agent.wasm> [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Run a WASM agent locally. The agent gets a DID identity,\n")
		fmt.Fprintf(os.Stderr, "checkpoints periodically, and can be resumed from checkpoint.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}
	wasmPath := fs.Arg(0)

	aid := *agentID
	if aid == "" {
		aid = agentIDFromPath(wasmPath)
	}

	runStandalone(wasmPath, aid, *budgetFlag, *manifestPath, *checkpointDir, *leaseDuration)
}

// subcmdResume implements "igord resume --checkpoint <path> --wasm <path>".
// Resumes an agent from a checkpoint file.
func subcmdResume(args []string) {
	fs := flag.NewFlagSet("resume", flag.ExitOnError)
	checkpointPath := fs.String("checkpoint", "", "Path to checkpoint file")
	wasmPath := fs.String("wasm", "", "Path to WASM binary")
	budgetFlag := fs.Float64("budget", 0, "Override budget (0 = use checkpoint budget)")
	manifestPath := fs.String("manifest", "", "Path to capability manifest JSON")
	checkpointDir := fs.String("checkpoint-dir", "checkpoints", "Directory for checkpoint storage")
	agentID := fs.String("agent-id", "", "Agent ID (default: derived from WASM filename)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: igord resume --checkpoint <path> --wasm <path> [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Resume an agent from a checkpoint file. The agent continues\n")
		fmt.Fprintf(os.Stderr, "from exactly where it left off with the same DID identity.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if *checkpointPath == "" || *wasmPath == "" {
		fs.Usage()
		os.Exit(1)
	}

	aid := *agentID
	if aid == "" {
		aid = agentIDFromPath(*wasmPath)
	}

	resumeFromCheckpoint(*checkpointPath, *wasmPath, aid, *budgetFlag, *manifestPath, *checkpointDir)
}

// subcmdVerify implements "igord verify <history-dir>".
func subcmdVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: igord verify <history-dir>\n\n")
		fmt.Fprintf(os.Stderr, "Verify the cryptographic lineage of an agent's checkpoint history.\n")
		fmt.Fprintf(os.Stderr, "The history directory contains numbered .ckpt files.\n\n")
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	result, err := inspector.VerifyChain(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	result.PrintChain(os.Stdout)
	if !result.ChainValid {
		os.Exit(1)
	}
}

// subcmdInspect implements "igord inspect <checkpoint-file> [--wasm path]".
func subcmdInspect(args []string) {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	wasmPath := fs.String("wasm", "", "Optional WASM binary to verify against checkpoint hash")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: igord inspect <checkpoint-file> [--wasm path]\n\n")
		fmt.Fprintf(os.Stderr, "Parse and display a checkpoint file, including DID identity.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}
	runInspectorWithDID(fs.Arg(0), *wasmPath)
}

// runStandalone runs an agent without P2P, migration, or leases.
func runStandalone(wasmPath, agentID string, budgetVal float64, manifestPath, checkpointDir string, leaseDuration time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := logging.NewLogger()

	storageProvider, err := storage.NewFSProvider(checkpointDir, logger)
	if err != nil {
		logger.Error("Failed to create storage provider", "error", err)
		os.Exit(1)
	}

	engine, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		logger.Error("Failed to create runtime engine", "error", err)
		os.Exit(1)
	}
	defer engine.Close(ctx)

	// Load or generate agent identity.
	agentIdent, err := loadOrGenerateIdentity(ctx, storageProvider, agentID, logger)
	if err != nil {
		logger.Error("Agent identity error", "error", err)
		os.Exit(1)
	}

	logger.Info("Agent identity",
		"did", agentIdent.DID(),
		"did_short", agentIdent.DIDShort(),
	)

	manifestData := runner.LoadManifestData(wasmPath, manifestPath, logger)
	budgetMicrocents := budget.FromFloat(budgetVal)

	instance, err := agent.LoadAgent(
		ctx, engine, wasmPath, agentID, storageProvider,
		budgetMicrocents, 1000000, // default price: 1.0/s
		manifestData, nil, "", agentIdent, logger,
	)
	if err != nil {
		logger.Error("Failed to load agent", "error", err)
		os.Exit(1)
	}
	defer instance.Close(ctx)

	instance.BudgetAdapter = settlement.NewMockAdapter(logger)

	if err := instance.Init(ctx); err != nil {
		logger.Error("Failed to initialize agent", "error", err)
		os.Exit(1)
	}

	// Try to load existing checkpoint.
	if loadErr := instance.LoadCheckpointFromStorage(ctx); loadErr != nil {
		logger.Info("No existing checkpoint, starting fresh")
	}

	if leaseDuration > 0 {
		leaseCfg := authority.LeaseConfig{
			Duration:      leaseDuration,
			RenewalWindow: 0.5,
			GracePeriod:   10 * time.Second,
		}
		instance.Lease = authority.NewLease(leaseCfg)
	}

	logger.Info("Agent started",
		"agent_id", agentID,
		"did", agentIdent.DIDShort(),
		"budget", budget.Format(budgetMicrocents),
		"tick", instance.TickNumber,
	)

	runTickLoop(ctx, instance, logger)
}

// resumeFromCheckpoint resumes an agent from a specific checkpoint file.
func resumeFromCheckpoint(checkpointPath, wasmPath, agentID string, budgetVal float64, manifestPath, checkpointDir string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := logging.NewLogger()

	// Copy checkpoint file to storage directory so the agent can find it.
	storageProvider, err := storage.NewFSProvider(checkpointDir, logger)
	if err != nil {
		logger.Error("Failed to create storage provider", "error", err)
		os.Exit(1)
	}

	// Read the provided checkpoint.
	ckptData, err := os.ReadFile(checkpointPath)
	if err != nil {
		logger.Error("Failed to read checkpoint", "path", checkpointPath, "error", err)
		os.Exit(1)
	}

	// Save it to the storage directory so LoadCheckpointFromStorage finds it.
	if err := storageProvider.SaveCheckpoint(ctx, agentID, ckptData); err != nil {
		logger.Error("Failed to stage checkpoint", "error", err)
		os.Exit(1)
	}

	// Extract identity from checkpoint if present.
	hdr, _, parseErr := agent.ParseCheckpointHeader(ckptData)
	if parseErr != nil {
		logger.Error("Failed to parse checkpoint", "error", parseErr)
		os.Exit(1)
	}

	engine, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		logger.Error("Failed to create runtime engine", "error", err)
		os.Exit(1)
	}
	defer engine.Close(ctx)

	// Try to load identity from checkpoint's agent pubkey or from storage.
	var agentIdent *identity.AgentIdentity
	idData, idErr := storageProvider.LoadIdentity(ctx, agentID)
	if idErr == nil {
		agentIdent, _ = identity.UnmarshalBinary(idData)
	}
	if agentIdent == nil {
		agentIdent, err = loadOrGenerateIdentity(ctx, storageProvider, agentID, logger)
		if err != nil {
			logger.Error("Agent identity error", "error", err)
			os.Exit(1)
		}
	}

	logger.Info("Resuming agent",
		"did", agentIdent.DID(),
		"from_tick", hdr.TickNumber,
		"checkpoint", checkpointPath,
	)

	manifestData := runner.LoadManifestData(wasmPath, manifestPath, logger)

	b := budget.FromFloat(budgetVal)
	if b == 0 {
		b = hdr.Budget
	}

	instance, err := agent.LoadAgent(
		ctx, engine, wasmPath, agentID, storageProvider,
		b, hdr.PricePerSecond,
		manifestData, nil, "", agentIdent, logger,
	)
	if err != nil {
		logger.Error("Failed to load agent", "error", err)
		os.Exit(1)
	}
	defer instance.Close(ctx)

	instance.BudgetAdapter = settlement.NewMockAdapter(logger)

	if err := instance.Init(ctx); err != nil {
		logger.Error("Failed to initialize agent", "error", err)
		os.Exit(1)
	}

	if err := instance.LoadCheckpointFromStorage(ctx); err != nil {
		logger.Error("Failed to resume from checkpoint", "error", err)
		os.Exit(1)
	}

	logger.Info("Agent resumed",
		"agent_id", agentID,
		"did", agentIdent.DIDShort(),
		"tick", instance.TickNumber,
		"budget", budget.Format(b),
	)

	runTickLoop(ctx, instance, logger)
}

// runTickLoop runs the simplified tick loop (no replay, no lease, no migration).
func runTickLoop(ctx context.Context, instance *agent.Instance, logger *slog.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	const (
		normalInterval = 1 * time.Second
		fastInterval   = 10 * time.Millisecond
	)

	checkpointTicker := time.NewTicker(5 * time.Second)
	defer checkpointTicker.Stop()

	tickTimer := time.NewTimer(normalInterval)
	defer tickTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sigChan:
			logger.Info("Shutting down, saving final checkpoint...")
			if err := instance.SaveCheckpointToStorage(ctx); err != nil {
				logger.Error("Failed to save checkpoint on shutdown", "error", err)
			} else {
				logger.Info("Final checkpoint saved")
			}
			return
		case <-tickTimer.C:
			hasMore, err := runner.SafeTick(ctx, instance)
			if err != nil {
				logger.Error("Tick failed", "error", err)
				if saveErr := instance.SaveCheckpointToStorage(ctx); saveErr != nil {
					logger.Error("Failed to save checkpoint after error", "error", saveErr)
				}
				return
			}
			if hasMore {
				tickTimer.Reset(fastInterval)
			} else {
				tickTimer.Reset(normalInterval)
			}
		case <-checkpointTicker.C:
			if err := instance.SaveCheckpointToStorage(ctx); err != nil {
				logger.Error("Failed to save checkpoint", "error", err)
			}
		}
	}
}

// runInspectorWithDID inspects a checkpoint and displays DID identity.
func runInspectorWithDID(checkpointPath, wasmPath string) {
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

	// Show DID if checkpoint has lineage.
	if result.HasLineage && len(result.AgentPubKey) == 32 {
		id := &identity.AgentIdentity{PublicKey: result.AgentPubKey}
		fmt.Fprintf(os.Stdout, "Agent DID:        %s\n", id.DID())
	}
}

// agentIDFromPath derives an agent ID from a WASM file path.
func agentIDFromPath(wasmPath string) string {
	base := filepath.Base(wasmPath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	if name == "" || name == "agent" {
		// Use parent directory name.
		name = filepath.Base(filepath.Dir(wasmPath))
	}
	if name == "" || name == "." {
		name = "agent"
	}
	return name
}
