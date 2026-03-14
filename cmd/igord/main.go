// SPDX-License-Identifier: Apache-2.0

// igord is the product CLI for Igor — the runtime for portable, immortal
// software agents. Subcommands: run, resume, verify, inspect.
//
// For the research/P2P CLI with migration, replay verification, and lease
// management, use igord-lab instead.
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
	"github.com/simonovic86/igor/internal/inspector"
	"github.com/simonovic86/igor/internal/logging"
	"github.com/simonovic86/igor/internal/runner"
	"github.com/simonovic86/igor/internal/runtime"
	"github.com/simonovic86/igor/internal/settlement"
	"github.com/simonovic86/igor/internal/storage"
	"github.com/simonovic86/igor/pkg/budget"
	"github.com/simonovic86/igor/pkg/identity"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		subcmdRun(os.Args[2:])
	case "resume":
		subcmdResume(os.Args[2:])
	case "verify":
		subcmdVerify(os.Args[2:])
	case "inspect":
		subcmdInspect(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: igord <command> [flags]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  run       Run a WASM agent with a new identity\n")
	fmt.Fprintf(os.Stderr, "  resume    Resume an agent from a checkpoint file\n")
	fmt.Fprintf(os.Stderr, "  verify    Verify checkpoint lineage chain\n")
	fmt.Fprintf(os.Stderr, "  inspect   Display checkpoint details\n")
	fmt.Fprintf(os.Stderr, "\nRun 'igord <command> -h' for help on a specific command.\n")
}

// subcmdRun implements "igord run <agent.wasm> [--budget N] [--manifest path]".
func subcmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	budgetFlag := fs.Float64("budget", 1.0, "Initial budget for agent execution")
	manifestPath := fs.String("manifest", "", "Path to capability manifest JSON")
	checkpointDir := fs.String("checkpoint-dir", "checkpoints", "Directory for checkpoint storage")
	agentID := fs.String("agent-id", "", "Agent ID (default: derived from WASM filename)")
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

	runStandalone(wasmPath, aid, *budgetFlag, *manifestPath, *checkpointDir)
}

// subcmdResume implements "igord resume --checkpoint <path> --wasm <path>".
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

	result, err := inspector.InspectFile(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if *wasmPath != "" {
		if verr := result.VerifyWASM(*wasmPath); verr != nil {
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

// runStandalone runs an agent without P2P, migration, or leases.
func runStandalone(wasmPath, agentID string, budgetVal float64, manifestPath, checkpointDir string) {
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
	agentIdent, err := identity.LoadOrGenerate(ctx, storageProvider, agentID, logger)
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
		agentIdent, err = identity.LoadOrGenerate(ctx, storageProvider, agentID, logger)
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

// agentIDFromPath derives an agent ID from a WASM file path.
func agentIDFromPath(wasmPath string) string {
	base := filepath.Base(wasmPath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	if name == "" || name == "agent" {
		name = filepath.Base(filepath.Dir(wasmPath))
	}
	if name == "" || name == "." {
		name = "agent"
	}
	return name
}
