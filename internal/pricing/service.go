// SPDX-License-Identifier: Apache-2.0

// Package pricing implements the /igor/price/1.0.0 protocol for inter-node
// price discovery. Nodes respond to price queries with their current execution
// pricing, enabling agents to make cost-aware migration decisions.
package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

// PriceProtocol is the Igor price query protocol identifier.
const PriceProtocol protocol.ID = "/igor/price/1.0.0"

// PriceRequest is sent by the querying node.
type PriceRequest struct {
	// AgentID is optional: future use for agent-specific pricing.
	AgentID string `json:"agent_id,omitempty"`
}

// PriceResponse is returned by the responding node.
type PriceResponse struct {
	PricePerSecond int64  `json:"price_per_second"` // microcents/sec
	NodeID         string `json:"node_id"`          // peer ID of responding node
}

// Service handles price advertisement and queries over libp2p streams.
type Service struct {
	host           host.Host
	pricePerSecond int64
	logger         *slog.Logger
}

// NewService creates and registers the pricing service on the given host.
func NewService(h host.Host, pricePerSecond int64, logger *slog.Logger) *Service {
	svc := &Service{
		host:           h,
		pricePerSecond: pricePerSecond,
		logger:         logger,
	}
	h.SetStreamHandler(PriceProtocol, svc.handlePriceQuery)
	logger.Info("Pricing service initialized",
		"price_per_second", pricePerSecond,
	)
	return svc
}

// handlePriceQuery responds to incoming price queries.
func (s *Service) handlePriceQuery(stream network.Stream) {
	defer stream.Close()

	if err := stream.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		s.logger.Error("Failed to set price query read deadline", "error", err)
		return
	}

	remotePeer := stream.Conn().RemotePeer()

	var req PriceRequest
	if err := json.NewDecoder(stream).Decode(&req); err != nil {
		s.logger.Error("Failed to decode price request",
			"from_peer", remotePeer.String(),
			"error", err,
		)
		return
	}

	resp := PriceResponse{
		PricePerSecond: s.pricePerSecond,
		NodeID:         s.host.ID().String(),
	}

	if err := json.NewEncoder(stream).Encode(resp); err != nil {
		s.logger.Error("Failed to encode price response",
			"to_peer", remotePeer.String(),
			"error", err,
		)
		return
	}

	s.logger.Info("Served price query",
		"from_peer", remotePeer.String(),
		"price", s.pricePerSecond,
	)
}

// QueryPeerPrice queries a remote peer's execution price.
func (s *Service) QueryPeerPrice(ctx context.Context, peerAddr string) (*PriceResponse, error) {
	maddr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid peer address: %w", err)
	}

	addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return nil, fmt.Errorf("failed to extract peer info: %w", err)
	}

	if err := s.host.Connect(ctx, *addrInfo); err != nil {
		return nil, fmt.Errorf("failed to connect to peer: %w", err)
	}

	stream, err := s.host.NewStream(ctx, addrInfo.ID, PriceProtocol)
	if err != nil {
		return nil, fmt.Errorf("failed to open price stream: %w", err)
	}
	defer stream.Close()

	req := PriceRequest{}
	if err := json.NewEncoder(stream).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send price request: %w", err)
	}

	var resp PriceResponse
	if err := json.NewDecoder(stream).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read price response: %w", err)
	}

	s.logger.Info("Received peer price",
		"peer_id", addrInfo.ID.String(),
		"price_per_second", resp.PricePerSecond,
	)

	return &resp, nil
}

// ScanPeerPrices queries prices from multiple peers concurrently.
// Returns results for peers that respond within the timeout; individual
// peer errors are logged but do not fail the scan.
func (s *Service) ScanPeerPrices(ctx context.Context, peerAddrs []string) []PriceResponse {
	type result struct {
		resp PriceResponse
		err  error
	}

	results := make(chan result, len(peerAddrs))

	for _, addr := range peerAddrs {
		go func(peerAddr string) {
			peerCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			resp, err := s.QueryPeerPrice(peerCtx, peerAddr)
			if err != nil {
				s.logger.Debug("Price scan: peer unreachable",
					"peer", peerAddr,
					"error", err,
				)
				results <- result{err: err}
				return
			}
			results <- result{resp: *resp}
		}(addr)
	}

	var responses []PriceResponse
	for range peerAddrs {
		r := <-results
		if r.err == nil {
			responses = append(responses, r.resp)
		}
	}
	return responses
}
