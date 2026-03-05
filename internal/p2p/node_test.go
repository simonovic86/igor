// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/simonovic86/igor/internal/config"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func TestNewNode_ListensAndCloses(t *testing.T) {
	cfg := &config.Config{
		NodeID:             "test-node",
		ListenAddress:      "/ip4/127.0.0.1/tcp/0",
		PricePerSecond:     1000,
		CheckpointDir:      t.TempDir(),
		ReplayWindowSize:   16,
		VerifyInterval:     5,
		ReplayMode:         "full",
		ReplayOnDivergence: "log",
	}

	ctx := context.Background()
	node, err := NewNode(ctx, cfg, testLogger())
	if err != nil {
		t.Fatalf("NewNode: %v", err)
	}
	defer node.Close()

	if node.Host == nil {
		t.Fatal("expected non-nil Host")
	}
	if node.Host.ID() == "" {
		t.Fatal("expected non-empty peer ID")
	}
	if len(node.Host.Addrs()) == 0 {
		t.Fatal("expected at least one listen address")
	}
}

func TestPingPeer_RoundTrip(t *testing.T) {
	ctx := context.Background()
	cfg := func() *config.Config {
		return &config.Config{
			NodeID:             "test",
			ListenAddress:      "/ip4/127.0.0.1/tcp/0",
			PricePerSecond:     1000,
			CheckpointDir:      t.TempDir(),
			ReplayWindowSize:   16,
			VerifyInterval:     5,
			ReplayMode:         "full",
			ReplayOnDivergence: "log",
		}
	}

	nodeA, err := NewNode(ctx, cfg(), testLogger())
	if err != nil {
		t.Fatalf("NewNode A: %v", err)
	}
	defer nodeA.Close()

	nodeB, err := NewNode(ctx, cfg(), testLogger())
	if err != nil {
		t.Fatalf("NewNode B: %v", err)
	}
	defer nodeB.Close()

	// Build multiaddr for nodeB
	addrs := nodeB.Host.Addrs()
	if len(addrs) == 0 {
		t.Fatal("nodeB has no addresses")
	}
	peerAddr := fmt.Sprintf("%s/p2p/%s", addrs[0].String(), nodeB.Host.ID().String())

	// Ping B from A
	if err := nodeA.PingPeer(ctx, peerAddr); err != nil {
		t.Fatalf("PingPeer: %v", err)
	}
}

func TestPingPeer_InvalidAddress(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		NodeID:             "test-node",
		ListenAddress:      "/ip4/127.0.0.1/tcp/0",
		PricePerSecond:     1000,
		CheckpointDir:      t.TempDir(),
		ReplayWindowSize:   16,
		VerifyInterval:     5,
		ReplayMode:         "full",
		ReplayOnDivergence: "log",
	}

	node, err := NewNode(ctx, cfg, testLogger())
	if err != nil {
		t.Fatalf("NewNode: %v", err)
	}
	defer node.Close()

	err = node.PingPeer(ctx, "not-a-valid-multiaddr")
	if err == nil {
		t.Fatal("expected error for invalid address")
	}
}

func TestNewNode_InvalidListenAddress(t *testing.T) {
	cfg := &config.Config{
		NodeID:             "test",
		ListenAddress:      "garbage",
		PricePerSecond:     1000,
		CheckpointDir:      t.TempDir(),
		ReplayWindowSize:   16,
		VerifyInterval:     5,
		ReplayMode:         "full",
		ReplayOnDivergence: "log",
	}

	_, err := NewNode(context.Background(), cfg, testLogger())
	if err == nil {
		t.Fatal("expected error for invalid listen address")
	}
}

func TestBootstrapPeers_UnreachablePeerContinues(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		NodeID:             "test",
		ListenAddress:      "/ip4/127.0.0.1/tcp/0",
		PricePerSecond:     1000,
		BootstrapPeers:     []string{"/ip4/127.0.0.1/tcp/59999/p2p/12D3KooWDpJ7As7BWAwRMfu1VU2WCqNjvq387JEYKDBj4kx6nXTN"},
		CheckpointDir:      t.TempDir(),
		ReplayWindowSize:   16,
		VerifyInterval:     5,
		ReplayMode:         "full",
		ReplayOnDivergence: "log",
	}

	// Should not panic or error — unreachable peers are logged and skipped
	node, err := NewNode(ctx, cfg, testLogger())
	if err != nil {
		t.Fatalf("NewNode with unreachable bootstrap: %v", err)
	}
	defer node.Close()
}
