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
	"os"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/internal/replay"
	"github.com/simonovic86/igor/internal/runtime"
	"github.com/simonovic86/igor/internal/storage"
	"github.com/simonovic86/igor/pkg/budget"
	"github.com/simonovic86/igor/pkg/manifest"
	protomsg "github.com/simonovic86/igor/pkg/protocol"
)

// MigrateProtocol is the libp2p protocol ID for agent migration.
const MigrateProtocol protocol.ID = "/igor/migrate/1.0.0"

// Service coordinates agent migration between nodes.
type Service struct {
	host            host.Host
	runtimeEngine   *runtime.Engine
	storageProvider storage.Provider
	replayEngine    *replay.Engine
	logger          *slog.Logger

	// Active agents running on this node.
	// Protected by mu — accessed from main goroutine and libp2p stream handlers.
	mu           sync.RWMutex
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
		replayEngine:    replay.NewEngine(logger),
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
	// Checkpoint v1 format: [version:1][budget:8][pricePerSecond:8][state:...]
	var budgetVal, pricePerSecond int64
	if len(checkpoint) >= 17 && checkpoint[0] == 0x01 {
		budgetVal = int64(binary.LittleEndian.Uint64(checkpoint[1:9]))
		pricePerSecond = int64(binary.LittleEndian.Uint64(checkpoint[9:17]))
		s.logger.Info("Budget metadata extracted",
			"budget", budget.Format(budgetVal),
			"price_per_second", budget.Format(pricePerSecond),
		)
	}

	// Create agent package
	pkg := protomsg.AgentPackage{
		AgentID:        agentID,
		WASMBinary:     wasmBinary,
		Checkpoint:     checkpoint,
		ManifestData:   manifestData,
		Budget:         budgetVal,
		PricePerSecond: pricePerSecond,
	}

	// Include replay verification data from the active instance (if available).
	// The staleness guard in replayDataFromInstance ensures the replay data
	// corresponds to the checkpoint we're sending.
	s.mu.RLock()
	instance, hasInstance := s.activeAgents[agentID]
	s.mu.RUnlock()
	if hasInstance {
		pkg.ReplayData = replayDataFromInstance(instance, checkpoint)
		if pkg.ReplayData != nil {
			s.logger.Info("Replay data included in migration package",
				"agent_id", agentID,
				"tick", pkg.ReplayData.TickNumber,
				"entries", len(pkg.ReplayData.Entries),
			)
		} else {
			s.logger.Info("No replay data available for migration package",
				"agent_id", agentID,
			)
		}
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
	s.mu.Lock()
	instance, exists := s.activeAgents[agentID]
	if exists {
		delete(s.activeAgents, agentID)
	}
	s.mu.Unlock()
	if exists {
		if err := instance.Close(ctx); err != nil {
			s.logger.Error("Failed to close local instance", "error", err)
		}
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
		"budget", budget.Format(pkg.Budget),
		"price_per_second", budget.Format(pkg.PricePerSecond),
	)

	// Replay verification: verify checkpoint integrity before accepting (CM-4)
	if pkg.ReplayData != nil {
		if reject := s.verifyMigrationReplay(ctx, stream, &pkg); reject {
			return
		}
	} else {
		s.logger.Info("No replay data in package, skipping verification",
			"agent_id", pkg.AgentID,
		)
	}

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
	s.mu.Lock()
	s.activeAgents[pkg.AgentID] = instance
	s.mu.Unlock()

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

// verifyMigrationReplay replays the last tick from the migration package and
// verifies the result matches the checkpoint. Returns true if migration should
// be rejected (verification failed or errored).
func (s *Service) verifyMigrationReplay(
	ctx context.Context,
	stream io.Writer,
	pkg *protomsg.AgentPackage,
) bool {
	s.logger.Info("Performing replay verification",
		"agent_id", pkg.AgentID,
		"tick", pkg.ReplayData.TickNumber,
		"entries", len(pkg.ReplayData.Entries),
	)

	// Parse manifest for replay engine
	capManifest, err := manifest.ParseCapabilityManifest(pkg.ManifestData)
	if err != nil {
		s.logger.Error("Failed to parse manifest for replay", "error", err)
		s.sendStartConfirmation(stream, pkg.AgentID, false,
			fmt.Sprintf("replay verification failed: invalid manifest: %v", err))
		return true
	}

	// Extract post-tick state from checkpoint
	postTickState, err := agent.ExtractAgentState(pkg.Checkpoint)
	if err != nil {
		s.logger.Error("Invalid checkpoint for replay verification", "error", err)
		s.sendStartConfirmation(stream, pkg.AgentID, false,
			fmt.Sprintf("replay verification failed: %v", err))
		return true
	}

	// Convert protocol replay data to eventlog types
	tickLog := toTickLog(pkg.ReplayData)

	// Execute replay
	result := s.replayEngine.ReplayTick(
		ctx,
		pkg.WASMBinary,
		capManifest,
		pkg.ReplayData.PreTickState,
		tickLog,
		postTickState,
	)

	if result.Error != nil {
		s.logger.Error("Replay verification error — rejecting migration",
			"agent_id", pkg.AgentID,
			"tick", result.TickNumber,
			"error", result.Error,
		)
		s.sendStartConfirmation(stream, pkg.AgentID, false,
			fmt.Sprintf("replay verification failed: %v", result.Error))
		return true
	}

	if !result.Verified {
		s.logger.Error("Replay divergence — rejecting migration",
			"agent_id", pkg.AgentID,
			"tick", result.TickNumber,
			"first_diff_byte", result.FirstDiffByte,
		)
		s.sendStartConfirmation(stream, pkg.AgentID, false,
			fmt.Sprintf("replay divergence at tick %d, first diff byte %d",
				result.TickNumber, result.FirstDiffByte))
		return true
	}

	s.logger.Info("Replay verification passed",
		"agent_id", pkg.AgentID,
		"tick", result.TickNumber,
	)
	return false
}

// RegisterAgent registers an actively running agent with the migration service.
func (s *Service) RegisterAgent(agentID string, instance *agent.Instance) {
	s.mu.Lock()
	s.activeAgents[agentID] = instance
	s.mu.Unlock()
	s.logger.Info("Agent registered with migration service", "agent_id", agentID)
}

// GetActiveInstance returns the active instance for the given agent ID, or nil.
func (s *Service) GetActiveInstance(agentID string) *agent.Instance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeAgents[agentID]
}

// GetActiveAgents returns the list of active agent IDs.
func (s *Service) GetActiveAgents() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	agents := make([]string, 0, len(s.activeAgents))
	for id := range s.activeAgents {
		agents = append(agents, id)
	}
	return agents
}
