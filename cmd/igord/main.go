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
		if err := runLocalAgent(ctx, *runAgent, logger); err != nil {
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
func runLocalAgent(ctx context.Context, wasmPath string, logger *slog.Logger) error {
	// Create WASM runtime engine
	engine, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		return fmt.Errorf("failed to create runtime engine: %w", err)
	}
	defer engine.Close(ctx)

	// Load agent
	instance, err := agent.LoadAgent(ctx, engine, wasmPath, "local-agent", logger)
	if err != nil {
		return fmt.Errorf("failed to load agent: %w", err)
	}
	defer instance.Close(ctx)

	// Initialize agent
	if err := instance.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize agent: %w", err)
	}

	// Check for existing checkpoint file
	checkpointPath := wasmPath + ".checkpoint"
	if data, err := os.ReadFile(checkpointPath); err == nil {
		logger.Info("Found existing checkpoint, resuming agent", "bytes", len(data))
		if err := instance.Resume(ctx, data); err != nil {
			logger.Error("Failed to resume from checkpoint", "error", err)
			// Continue anyway with fresh state
		}
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
			state, err := instance.Checkpoint(ctx)
			if err != nil {
				logger.Error("Failed to checkpoint on shutdown", "error", err)
			} else if err := os.WriteFile(checkpointPath, state, 0644); err != nil {
				logger.Error("Failed to save checkpoint", "error", err)
			} else {
				logger.Info("Checkpoint saved", "path", checkpointPath)
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
			state, err := instance.Checkpoint(ctx)
			if err != nil {
				logger.Error("Checkpoint failed", "error", err)
				continue
			}

			// Save to file
			if err := os.WriteFile(checkpointPath, state, 0644); err != nil {
				logger.Error("Failed to save checkpoint", "error", err)
			} else {
				logger.Info("Checkpoint saved", "path", checkpointPath)
			}
		}
	}
}
