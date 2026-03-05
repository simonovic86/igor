// Package migration implements peer-to-peer agent relocation and transfer protocols.
// Coordinates autonomous agent migration between distributed nodes using libp2p streams
// while maintaining single-instance invariants and budget preservation guarantees.
package migration

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

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
	replayMode      string // "off", "periodic", "on-migrate", "full"
	replayCostLog   bool
	pricePerSecond  int64 // This node's price per second in microcents
	logger          *slog.Logger

	// nodeCapabilities overrides manifest.NodeCapabilities for this node when
	// non-nil. Enables heterogeneous capability sets across nodes.
	nodeCapabilities []string

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
	replayMode string,
	replayCostLog bool,
	pricePerSecond int64,
	logger *slog.Logger,
) *Service {
	svc := &Service{
		host:            h,
		runtimeEngine:   engine,
		storageProvider: storage,
		replayEngine:    replay.NewEngine(logger),
		replayMode:      replayMode,
		replayCostLog:   replayCostLog,
		pricePerSecond:  pricePerSecond,
		logger:          logger,
		activeAgents:    make(map[string]*agent.Instance),
	}

	// Register migration protocol handler
	h.SetStreamHandler(MigrateProtocol, svc.handleIncomingMigration)

	logger.Info("Migration service initialized")
	return svc
}

// signingKey extracts the Ed25519 private key from the libp2p host's peerstore.
// Returns nil if the key is unavailable or not Ed25519.
func (s *Service) signingKey() ed25519.PrivateKey {
	privKey := s.host.Peerstore().PrivKey(s.host.ID())
	if privKey == nil {
		return nil
	}
	raw, err := privKey.Raw()
	if err != nil {
		s.logger.Warn("Failed to extract raw signing key", "error", err)
		return nil
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil
	}
	return ed25519.PrivateKey(raw)
}

// buildMigrationPackage assembles the AgentPackage for migration, including
// WASM binary, checkpoint, manifest, receipts, and replay data.
func (s *Service) buildMigrationPackage(
	ctx context.Context,
	agentID string,
	wasmPath string,
) (protomsg.AgentPackage, *agent.Instance, error) {
	wasmBinary, err := os.ReadFile(wasmPath)
	if err != nil {
		return protomsg.AgentPackage{}, nil, fmt.Errorf("failed to read WASM binary: %w", err)
	}

	// Load manifest from sidecar file (wasmPath with .json extension)
	manifestData := manifest.LoadSidecarData(wasmPath, "", s.logger)

	checkpoint, err := s.storageProvider.LoadCheckpoint(ctx, agentID)
	if err != nil {
		return protomsg.AgentPackage{}, nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	s.logger.Info("Checkpoint loaded for migration",
		"agent_id", agentID,
		"checkpoint_size", len(checkpoint),
	)

	var budgetVal, pricePerSecond int64
	if parsedBudget, parsedPrice, _, _, _, parseErr := agent.ParseCheckpointHeader(checkpoint); parseErr == nil {
		budgetVal = parsedBudget
		pricePerSecond = parsedPrice
		s.logger.Info("Budget metadata extracted",
			"budget", budget.Format(budgetVal),
			"price_per_second", budget.Format(pricePerSecond),
		)
	} else {
		s.logger.Warn("Could not parse checkpoint header for budget extraction",
			"agent_id", agentID,
			"error", parseErr,
		)
	}

	receiptData, err := s.storageProvider.LoadReceipts(ctx, agentID)
	if err != nil {
		receiptData = nil
	}

	wasmHashArr := sha256.Sum256(wasmBinary)
	pkg := protomsg.AgentPackage{
		AgentID:        agentID,
		WASMBinary:     wasmBinary,
		WASMHash:       wasmHashArr[:],
		Checkpoint:     checkpoint,
		ManifestData:   manifestData,
		Budget:         budgetVal,
		PricePerSecond: pricePerSecond,
		Receipts:       receiptData,
	}

	s.mu.RLock()
	localInstance, hasInstance := s.activeAgents[agentID]
	s.mu.RUnlock()
	if hasInstance {
		pkg.ReplayData = replayDataFromInstance(localInstance, checkpoint)
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

	return pkg, localInstance, nil
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

	pkg, localInstance, err := s.buildMigrationPackage(ctx, agentID, wasmPath)
	if err != nil {
		return err
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

	// Terminate local instance using compare-and-delete: only remove if the
	// map still holds the same instance we read earlier. A concurrent incoming
	// migration could have registered a different instance for this agent ID;
	// deleting that would violate EI-1 (Single Active Instance).
	// Close is held under the lock to prevent concurrent access to a closing
	// instance (wazero Module.Close is fast — it just marks the module closed).
	s.mu.Lock()
	if localInstance != nil && s.activeAgents[agentID] == localInstance {
		delete(s.activeAgents, agentID)
		if err := localInstance.Close(ctx); err != nil {
			s.logger.Error("Failed to close local instance", "error", err)
		}
		s.logger.Info("Local agent instance terminated", "agent_id", agentID)
	}
	s.mu.Unlock()

	// Delete local checkpoint — failure is an error because a stale checkpoint
	// could cause EI-1 violations if this node restarts and resumes the agent.
	if err := s.storageProvider.DeleteCheckpoint(ctx, agentID); err != nil {
		return fmt.Errorf("migration succeeded but failed to delete local checkpoint: %w", err)
	}
	s.logger.Info("Local checkpoint deleted", "agent_id", agentID)

	// Delete local receipts (non-fatal).
	if err := s.storageProvider.DeleteReceipts(ctx, agentID); err != nil {
		s.logger.Error("Failed to delete local receipts", "error", err)
	}

	s.logger.Info("Migration completed successfully", "agent_id", agentID)
	return nil
}

// handleIncomingMigration handles an incoming agent transfer.
func (s *Service) handleIncomingMigration(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	s.logger.Info("Receiving agent migration", "from_peer", remotePeer.String())

	// Bound the entire handler to prevent resource exhaustion from slow
	// or malicious peers. 2 minutes allows time for replay verification.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Set a read deadline for the initial transfer decode. Large WASM
	// binaries may take time to arrive but 30s is generous for local/LAN.
	if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		s.logger.Error("Failed to set stream read deadline", "error", err)
	}

	// Decode transfer message
	decoder := json.NewDecoder(stream)
	var transfer protomsg.AgentTransfer
	if err := decoder.Decode(&transfer); err != nil {
		s.logger.Error("Failed to decode transfer", "error", err)
		s.sendStartConfirmation(stream, "", false, err.Error())
		return
	}

	// Clear read deadline for remaining operations (checkpoint save, agent init, etc.)
	if err := stream.SetReadDeadline(time.Time{}); err != nil {
		s.logger.Error("Failed to clear stream read deadline", "error", err)
	}

	pkg := transfer.Package
	s.logger.Info("Agent package received",
		"agent_id", pkg.AgentID,
		"wasm_size", len(pkg.WASMBinary),
		"checkpoint_size", len(pkg.Checkpoint),
		"budget", budget.Format(pkg.Budget),
		"price_per_second", budget.Format(pkg.PricePerSecond),
	)

	// Verify WASM binary integrity — reject if hash is missing or malformed
	if len(pkg.WASMHash) != 32 {
		s.logger.Error("WASM hash missing or malformed — rejecting migration",
			"agent_id", pkg.AgentID,
			"hash_len", len(pkg.WASMHash),
		)
		s.sendStartConfirmation(stream, pkg.AgentID, false, "WASM hash missing or invalid length")
		return
	}
	computed := sha256.Sum256(pkg.WASMBinary)
	if computed != [32]byte(pkg.WASMHash) {
		s.logger.Error("WASM hash mismatch — rejecting migration", "agent_id", pkg.AgentID)
		s.sendStartConfirmation(stream, pkg.AgentID, false, "WASM binary hash mismatch")
		return
	}

	// Replay verification: verify checkpoint integrity before accepting (CM-4)
	migrateVerify := s.replayMode == "on-migrate" || s.replayMode == "full"
	if migrateVerify && pkg.ReplayData != nil {
		if reject := s.verifyMigrationReplay(ctx, stream, &pkg); reject {
			return
		}
	} else if pkg.ReplayData == nil {
		s.logger.Info("No replay data in package, skipping verification",
			"agent_id", pkg.AgentID,
		)
	} else {
		s.logger.Info("Replay verification skipped by mode",
			"agent_id", pkg.AgentID,
			"replay_mode", s.replayMode,
		)
	}

	// Parse and validate manifest (capabilities, migration policy, resource limits).
	fullManifest, err := manifest.ParseManifest(pkg.ManifestData)
	if err != nil {
		s.logger.Error("Invalid manifest in migration package", "error", err)
		s.sendStartConfirmation(stream, pkg.AgentID, false, "invalid manifest: "+err.Error())
		return
	}

	if rejectMsg := s.validateIncomingManifest(fullManifest, pkg.AgentID); rejectMsg != "" {
		s.sendStartConfirmation(stream, pkg.AgentID, false, rejectMsg)
		return
	}

	// Save checkpoint to storage
	if err := s.storageProvider.SaveCheckpoint(ctx, pkg.AgentID, pkg.Checkpoint); err != nil {
		s.logger.Error("Failed to save checkpoint", "error", err)
		s.sendStartConfirmation(stream, pkg.AgentID, false, err.Error())
		return
	}

	// Save receipts alongside checkpoint (non-fatal).
	if len(pkg.Receipts) > 0 {
		if err := s.storageProvider.SaveReceipts(ctx, pkg.AgentID, pkg.Receipts); err != nil {
			s.logger.Error("Failed to save receipts", "error", err)
		}
	}

	instance, err := agent.LoadAgentFromBytes(
		ctx,
		s.runtimeEngine,
		pkg.WASMBinary,
		pkg.AgentID,
		s.storageProvider,
		pkg.Budget,
		pkg.PricePerSecond,
		pkg.ManifestData,
		s.signingKey(),
		s.host.ID().String(),
		s.logger,
	)
	if err != nil {
		s.logger.Error("Failed to load agent", "error", err)
		s.deleteOrphanedCheckpoint(ctx, pkg.AgentID)
		s.sendStartConfirmation(stream, pkg.AgentID, false, err.Error())
		return
	}

	// Initialize agent
	if err := instance.Init(ctx); err != nil {
		s.logger.Error("Failed to initialize agent", "error", err)
		instance.Close(ctx)
		s.deleteOrphanedCheckpoint(ctx, pkg.AgentID)
		s.sendStartConfirmation(stream, pkg.AgentID, false, err.Error())
		return
	}

	// Resume from checkpoint
	if err := instance.LoadCheckpointFromStorage(ctx); err != nil {
		s.logger.Error("Failed to resume agent", "error", err)
		instance.Close(ctx)
		s.deleteOrphanedCheckpoint(ctx, pkg.AgentID)
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

// validateIncomingManifest checks migration policy, resource limits, and
// capability requirements. Returns a non-empty rejection message on failure.
func (s *Service) validateIncomingManifest(m *manifest.Manifest, agentID string) string {
	// Check migration policy: reject if agent explicitly disables migration.
	if m.MigrationPolicy != nil && !m.MigrationPolicy.Enabled {
		s.logger.Error("Migration policy disabled", "agent_id", agentID)
		return "agent migration policy disabled"
	}

	// Check migration policy: reject if node price exceeds agent's maximum.
	if m.MigrationPolicy != nil &&
		m.MigrationPolicy.MaxPricePerSecond > 0 &&
		s.pricePerSecond > m.MigrationPolicy.MaxPricePerSecond {
		s.logger.Error("Node price exceeds agent maximum",
			"agent_id", agentID,
			"node_price", s.pricePerSecond,
			"agent_max", m.MigrationPolicy.MaxPricePerSecond,
		)
		return fmt.Sprintf("node price %d exceeds agent max %d",
			s.pricePerSecond, m.MigrationPolicy.MaxPricePerSecond)
	}

	// Validate resource limits: reject agents that require more memory than the node provides.
	if m.ResourceLimits.MaxMemoryBytes > 0 &&
		m.ResourceLimits.MaxMemoryBytes > manifest.DefaultMaxMemoryBytes {
		s.logger.Error("Agent memory requirement exceeds node capacity",
			"agent_id", agentID,
			"required", m.ResourceLimits.MaxMemoryBytes,
			"available", manifest.DefaultMaxMemoryBytes,
		)
		return fmt.Sprintf("agent requires %d bytes memory, node provides %d",
			m.ResourceLimits.MaxMemoryBytes, manifest.DefaultMaxMemoryBytes)
	}

	// CE-5: Verify target node can satisfy agent's declared capabilities.
	s.mu.RLock()
	nodeCaps := manifest.NodeCapabilities
	if s.nodeCapabilities != nil {
		nodeCaps = s.nodeCapabilities
	}
	s.mu.RUnlock()
	if err := manifest.ValidateAgainstNode(m.Capabilities, nodeCaps); err != nil {
		s.logger.Error("Capability check failed", "agent_id", agentID, "error", err)
		return "capability check failed: " + err.Error()
	}

	return ""
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

	attrs := []any{
		"agent_id", pkg.AgentID,
		"tick", result.TickNumber,
	}
	if s.replayCostLog {
		attrs = append(attrs, "replay_duration", result.Duration)
	}
	s.logger.Info("Replay verification passed", attrs...)
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

// deleteOrphanedCheckpoint removes a checkpoint that was saved during an
// incoming migration but whose agent failed to fully initialize. Without
// cleanup, a stale checkpoint would block future migrations for this agent ID.
func (s *Service) deleteOrphanedCheckpoint(ctx context.Context, agentID string) {
	if err := s.storageProvider.DeleteCheckpoint(ctx, agentID); err != nil {
		s.logger.Error("Failed to delete orphaned checkpoint", "agent_id", agentID, "error", err)
	} else {
		s.logger.Info("Orphaned checkpoint cleaned up", "agent_id", agentID)
	}
}

// SetNodeCapabilities overrides the default node capabilities for this service.
// When set, incoming migrations validate against these capabilities instead of
// manifest.NodeCapabilities. Pass nil to restore default behavior.
func (s *Service) SetNodeCapabilities(caps []string) {
	s.mu.Lock()
	s.nodeCapabilities = caps
	s.mu.Unlock()
}
