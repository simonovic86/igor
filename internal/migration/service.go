// Package migration implements peer-to-peer agent relocation and transfer protocols.
// Coordinates autonomous agent migration between distributed nodes using libp2p streams
// while maintaining single-instance invariants and budget preservation guarantees.
package migration

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/internal/runtime"
	"github.com/simonovic86/igor/internal/storage"
	protomsg "github.com/simonovic86/igor/pkg/protocol"
)

// MigrateProtocol is the libp2p protocol ID for agent migration.
const MigrateProtocol protocol.ID = "/igor/migrate/1.0.0"

// Service coordinates agent migration between nodes.
type Service struct {
	host            host.Host
	runtimeEngine   *runtime.Engine
	storageProvider storage.Provider
	logger          *slog.Logger

	// Active agents running on this node
	activeAgents map[string]*agent.Instance
}

// NewService creates a new migration service.
func NewService(
	h host.Host,
	engine *runtime.Engine,
	storage storage.Provider,
	logger *slog.Logger,
) *Service {
	svc := &Service{
		host:            h,
		runtimeEngine:   engine,
		storageProvider: storage,
		logger:          logger,
		activeAgents:    make(map[string]*agent.Instance),
	}

	// Register migration protocol handler
	h.SetStreamHandler(MigrateProtocol, svc.handleIncomingMigration)

	logger.Info("Migration service initialized")
	return svc
}

// MigrateAgent migrates an agent to a target peer.
func (s *Service) MigrateAgent(
	ctx context.Context,
	agentID string,
	wasmPath string,
	targetPeerAddr string,
) error {
	s.logger.Info("Starting agent migration",
		"agent_id", agentID,
		"target", targetPeerAddr,
	)

	// Parse target multiaddr
	maddr, err := multiaddr.NewMultiaddr(targetPeerAddr)
	if err != nil {
		return fmt.Errorf("invalid target address: %w", err)
	}

	addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to extract peer info: %w", err)
	}

	// Connect to target peer
	if err := s.host.Connect(ctx, *addrInfo); err != nil {
		return fmt.Errorf("failed to connect to target peer: %w", err)
	}

	// Load WASM binary
	wasmBinary, err := os.ReadFile(wasmPath)
	if err != nil {
		return fmt.Errorf("failed to read WASM binary: %w", err)
	}

	// Load manifest from sidecar file (wasmPath with .json extension)
	manifestPath := wasmPath[:len(wasmPath)-len(".wasm")] + ".manifest.json"
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		// No manifest file — use empty capabilities (backward compatible)
		manifestData = []byte("{}")
		s.logger.Info("No manifest file found, using empty capabilities",
			"expected_path", manifestPath,
		)
	}

	// Load checkpoint from storage
	checkpoint, err := s.storageProvider.LoadCheckpoint(ctx, agentID)
	if err != nil {
		return fmt.Errorf("failed to load checkpoint: %w", err)
	}

	s.logger.Info("Checkpoint loaded for migration",
		"agent_id", agentID,
		"checkpoint_size", len(checkpoint),
	)

	// Extract budget metadata from checkpoint for package visibility
	// Checkpoint format: [budget:8][pricePerSecond:8][state:...]
	var budget, pricePerSecond float64
	if len(checkpoint) >= 16 {
		budget = math.Float64frombits(binary.LittleEndian.Uint64(checkpoint[0:8]))
		pricePerSecond = math.Float64frombits(binary.LittleEndian.Uint64(checkpoint[8:16]))
		s.logger.Info("Budget metadata extracted",
			"budget", fmt.Sprintf("%.6f", budget),
			"price_per_second", fmt.Sprintf("%.6f", pricePerSecond),
		)
	}

	// Create agent package
	pkg := protomsg.AgentPackage{
		AgentID:        agentID,
		WASMBinary:     wasmBinary,
		Checkpoint:     checkpoint,
		ManifestData:   manifestData,
		Budget:         budget,
		PricePerSecond: pricePerSecond,
	}

	transfer := protomsg.AgentTransfer{
		Package:      pkg,
		SourceNodeID: s.host.ID().String(),
	}

	// Open stream to target
	stream, err := s.host.NewStream(ctx, addrInfo.ID, MigrateProtocol)
	if err != nil {
		return fmt.Errorf("failed to open migration stream: %w", err)
	}
	defer stream.Close()

	// Send transfer message
	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(transfer); err != nil {
		return fmt.Errorf("failed to send transfer: %w", err)
	}

	s.logger.Info("Transfer sent", "agent_id", agentID)

	// Read confirmation
	decoder := json.NewDecoder(stream)
	var started protomsg.AgentStarted
	if err := decoder.Decode(&started); err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	if !started.Success {
		return fmt.Errorf("target failed to start agent: %s", started.Error)
	}

	s.logger.Info("Agent started on target",
		"agent_id", agentID,
		"target_node", started.NodeID,
	)

	// Terminate local instance if exists
	if instance, exists := s.activeAgents[agentID]; exists {
		if err := instance.Close(ctx); err != nil {
			s.logger.Error("Failed to close local instance", "error", err)
		}
		delete(s.activeAgents, agentID)
		s.logger.Info("Local agent instance terminated", "agent_id", agentID)
	}

	// Delete local checkpoint
	if err := s.storageProvider.DeleteCheckpoint(ctx, agentID); err != nil {
		s.logger.Error("Failed to delete local checkpoint", "error", err)
	} else {
		s.logger.Info("Local checkpoint deleted", "agent_id", agentID)
	}

	s.logger.Info("Migration completed successfully", "agent_id", agentID)
	return nil
}

// handleIncomingMigration handles an incoming agent transfer.
func (s *Service) handleIncomingMigration(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	s.logger.Info("Receiving agent migration", "from_peer", remotePeer.String())

	ctx := context.Background()

	// Decode transfer message
	decoder := json.NewDecoder(stream)
	var transfer protomsg.AgentTransfer
	if err := decoder.Decode(&transfer); err != nil {
		s.logger.Error("Failed to decode transfer", "error", err)
		s.sendStartConfirmation(stream, "", false, err.Error())
		return
	}

	pkg := transfer.Package
	s.logger.Info("Agent package received",
		"agent_id", pkg.AgentID,
		"wasm_size", len(pkg.WASMBinary),
		"checkpoint_size", len(pkg.Checkpoint),
		"budget", fmt.Sprintf("%.6f", pkg.Budget),
		"price_per_second", fmt.Sprintf("%.6f", pkg.PricePerSecond),
	)

	// Save checkpoint to storage
	if err := s.storageProvider.SaveCheckpoint(ctx, pkg.AgentID, pkg.Checkpoint); err != nil {
		s.logger.Error("Failed to save checkpoint", "error", err)
		s.sendStartConfirmation(stream, pkg.AgentID, false, err.Error())
		return
	}

	// Write WASM binary to temporary file
	wasmPath := fmt.Sprintf("/tmp/igor-agent-%s.wasm", pkg.AgentID)
	if err := os.WriteFile(wasmPath, pkg.WASMBinary, 0644); err != nil {
		s.logger.Error("Failed to write WASM binary", "error", err)
		s.sendStartConfirmation(stream, pkg.AgentID, false, err.Error())
		return
	}

	// Load agent with budget and manifest from package
	instance, err := agent.LoadAgent(
		ctx,
		s.runtimeEngine,
		wasmPath,
		pkg.AgentID,
		s.storageProvider,
		pkg.Budget,
		pkg.PricePerSecond,
		pkg.ManifestData,
		s.logger,
	)
	if err != nil {
		s.logger.Error("Failed to load agent", "error", err)
		s.sendStartConfirmation(stream, pkg.AgentID, false, err.Error())
		return
	}

	// Initialize agent
	if err := instance.Init(ctx); err != nil {
		s.logger.Error("Failed to initialize agent", "error", err)
		instance.Close(ctx)
		s.sendStartConfirmation(stream, pkg.AgentID, false, err.Error())
		return
	}

	// Resume from checkpoint
	if err := instance.LoadCheckpointFromStorage(ctx); err != nil {
		s.logger.Error("Failed to resume agent", "error", err)
		instance.Close(ctx)
		s.sendStartConfirmation(stream, pkg.AgentID, false, err.Error())
		return
	}

	// Store as active agent
	s.activeAgents[pkg.AgentID] = instance

	s.logger.Info("Agent migration accepted and started",
		"agent_id", pkg.AgentID,
		"from_node", transfer.SourceNodeID,
	)

	// Send success confirmation
	s.sendStartConfirmation(stream, pkg.AgentID, true, "")
}

// sendStartConfirmation sends an AgentStarted message.
func (s *Service) sendStartConfirmation(
	stream io.Writer,
	agentID string,
	success bool,
	errorMsg string,
) {
	started := protomsg.AgentStarted{
		AgentID: agentID,
		NodeID:  s.host.ID().String(),
		Success: success,
		Error:   errorMsg,
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(started); err != nil {
		s.logger.Error("Failed to send confirmation", "error", err)
	}
}

// RegisterAgent registers an actively running agent with the migration service.
func (s *Service) RegisterAgent(agentID string, instance *agent.Instance) {
	s.activeAgents[agentID] = instance
	s.logger.Info("Agent registered with migration service", "agent_id", agentID)
}

// GetActiveAgents returns the list of active agent IDs.
func (s *Service) GetActiveAgents() []string {
	agents := make([]string, 0, len(s.activeAgents))
	for id := range s.activeAgents {
		agents = append(agents, id)
	}
	return agents
}
