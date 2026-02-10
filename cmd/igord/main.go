package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/simonovic86/igor/internal/config"
	"github.com/simonovic86/igor/internal/logging"
	"github.com/simonovic86/igor/internal/p2p"
)

func main() {
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

	logging.Info(logger, "Igor Node ready")

	// Block until interrupted
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logging.Info(logger, "Igor Node shutting down...")
}
