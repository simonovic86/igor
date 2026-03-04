package p2p

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/simonovic86/igor/internal/config"
)

// PingProtocol is the Igor ping protocol identifier.
const PingProtocol protocol.ID = "/igor/ping/1.0.0"

// Node represents a libp2p host for Igor.
type Node struct {
	Host   host.Host
	Logger *slog.Logger
}

// NewNode creates and initializes a new P2P node.
func NewNode(
	ctx context.Context,
	cfg *config.Config,
	logger *slog.Logger,
) (*Node, error) {
	// Parse listen address
	listenAddr, err := multiaddr.NewMultiaddr(cfg.ListenAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid listen address: %w", err)
	}

	// Create libp2p host with listen address
	h, err := libp2p.New(
		libp2p.ListenAddrs(listenAddr),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	node := &Node{
		Host:   h,
		Logger: logger,
	}

	// Register protocol handlers
	node.registerHandlers()

	// Log peer identity and listen addresses
	logger.Info("P2P node created",
		"peer_id", h.ID().String(),
	)
	for _, addr := range h.Addrs() {
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), h.ID().String())
		logger.Info("Listening on", "address", fullAddr)
	}

	// Bootstrap peer connections if configured
	if len(cfg.BootstrapPeers) > 0 {
		node.bootstrapPeers(ctx, cfg.BootstrapPeers)
	}

	return node, nil
}

// registerHandlers registers protocol handlers for the node.
func (n *Node) registerHandlers() {
	n.Host.SetStreamHandler(PingProtocol, n.handlePing)
}

// handlePing handles incoming ping requests.
func (n *Node) handlePing(s network.Stream) {
	defer s.Close()

	// Set read deadline to prevent slow/malicious peers from holding the connection.
	if err := s.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		n.Logger.Error("Failed to set ping read deadline", "error", err)
		return
	}

	remotePeer := s.Conn().RemotePeer()
	n.Logger.Info("Received ping", "from_peer", remotePeer.String())

	// Read "ping" message
	buf := make([]byte, 4)
	_, err := io.ReadFull(s, buf)
	if err != nil {
		n.Logger.Error("Failed to read ping", "error", err)
		return
	}

	if string(buf) != "ping" {
		n.Logger.Error("Invalid ping message", "received", string(buf))
		return
	}

	// Send "pong" response
	_, err = s.Write([]byte("pong"))
	if err != nil {
		n.Logger.Error("Failed to send pong", "error", err)
		return
	}

	n.Logger.Info("Sent pong", "to_peer", remotePeer.String())
}

// PingPeer sends a ping to a remote peer and expects a pong response.
func (n *Node) PingPeer(ctx context.Context, peerAddr string) error {
	// Parse multiaddr
	maddr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}

	// Extract peer ID and address info
	addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to extract peer info: %w", err)
	}

	// Connect to peer
	n.Logger.Info("Connecting to peer", "peer_id", addrInfo.ID.String())
	err = n.Host.Connect(ctx, *addrInfo)
	if err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	// Open stream
	s, err := n.Host.NewStream(ctx, addrInfo.ID, PingProtocol)
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}
	defer s.Close()

	// Send "ping"
	_, err = s.Write([]byte("ping"))
	if err != nil {
		return fmt.Errorf("failed to send ping: %w", err)
	}

	// Read "pong"
	buf := make([]byte, 4)
	_, err = io.ReadFull(s, buf)
	if err != nil {
		return fmt.Errorf("failed to read pong: %w", err)
	}

	if string(buf) != "pong" {
		return fmt.Errorf("unexpected response: %s", string(buf))
	}

	n.Logger.Info("Ping successful", "peer_id", addrInfo.ID.String())
	return nil
}

// bootstrapPeers attempts to connect to bootstrap peers.
func (n *Node) bootstrapPeers(ctx context.Context, peers []string) {
	n.Logger.Info("Attempting to connect to bootstrap peers", "count", len(peers))

	for _, peerAddr := range peers {
		peerCtx, peerCancel := context.WithTimeout(ctx, 30*time.Second)
		err := n.PingPeer(peerCtx, peerAddr)
		peerCancel()
		if err != nil {
			n.Logger.Error("Failed to bootstrap peer",
				"address", peerAddr,
				"error", err,
			)
			// Continue to next peer - failures must not crash node
			continue
		}
		n.Logger.Info("Successfully bootstrapped peer", "address", peerAddr)
	}
}

// Close closes the P2P node and releases resources.
func (n *Node) Close() error {
	return n.Host.Close()
}
