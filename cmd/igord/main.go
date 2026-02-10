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
	"github.com/simonovic86/igor/internal/config"
	"github.com/simonovic86/igor/internal/logging"
	"github.com/simonovic86/igor/internal/migration"
	"github.com/simonovic86/igor/internal/p2p"
	"github.com/simonovic86/igor/internal/runtime"
	"github.com/simonovic86/igor/internal/storage"
)

func main() {
	// Parse CLI flags
	runAgent := flag.String("run-agent", "", "Path to WASM agent to run locally")
	migrateAgent := flag.String("migrate-agent", "", "Agent ID to migrate")
	targetPeer := flag.String("to", "", "Target peer multiaddr for migration")
	wasmPath := flag.String("wasm", "", "WASM binary path for migration")
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
	migrationSvc := migration.NewService(node.Host, engine, storageProvider, logger)

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
		if err := runLocalAgent(ctx, cfg, *runAgent, migrationSvc, logger); err != nil {
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
	wasmPath string,
	migrationSvc *migration.Service,
	logger *slog.Logger,
) error {
	// Create storage provider
	storageProvider, err := storage.NewFSProvider(cfg.CheckpointDir, logger)
	if err != nil {
		return fmt.Errorf("failed to create storage provider: %w", err)
	}

	// Create WASM runtime engine
	engine, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		return fmt.Errorf("failed to create runtime engine: %w", err)
	}
	defer engine.Close(ctx)

	// Load agent
	instance, err := agent.LoadAgent(ctx, engine, wasmPath, "local-agent", storageProvider, logger)
	if err != nil {
		return fmt.Errorf("failed to load agent: %w", err)
	}
	defer instance.Close(ctx)

	// Register agent with migration service
	migrationSvc.RegisterAgent("local-agent", instance)

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

	// Tick loop
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	checkpointTicker := time.NewTicker(5 * time.Second)
	defer checkpointTicker.Stop()

	logger.Info("Starting agent tick loop")

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
			// Execute tick
			if err := instance.Tick(ctx); err != nil {
				logger.Error("Tick failed", "error", err)
				return err
			}

		case <-checkpointTicker.C:
			// Periodic checkpoint
			if err := instance.SaveCheckpointToStorage(ctx); err != nil {
				logger.Error("Failed to save checkpoint", "error", err)
			}
		}
	}
}
