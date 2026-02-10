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
	"github.com/simonovic86/igor/internal/p2p"
	"github.com/simonovic86/igor/internal/runtime"
	"github.com/simonovic86/igor/internal/storage"
)

func main() {
	// Parse CLI flags
	runAgent := flag.String("run-agent", "", "Path to WASM agent to run locally")
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

	// If --run-agent flag is provided, run agent locally
	if *runAgent != "" {
		if err := runLocalAgent(ctx, cfg, *runAgent, logger); err != nil {
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
func runLocalAgent(ctx context.Context, cfg *config.Config, wasmPath string, logger *slog.Logger) error {
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
