package pricing

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func TestPriceQuery_RoundTrip(t *testing.T) {
	ctx := context.Background()

	// Create two libp2p hosts
	hostA, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("create host A: %v", err)
	}
	defer hostA.Close()

	hostB, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("create host B: %v", err)
	}
	defer hostB.Close()

	// Register pricing service on host B with price 5000 microcents/sec
	_ = NewService(hostB, 5000, testLogger())

	// Create pricing service on host A to query B
	svcA := NewService(hostA, 1000, testLogger())

	// Build multiaddr for host B
	addrs := hostB.Addrs()
	if len(addrs) == 0 {
		t.Fatal("host B has no addresses")
	}
	peerAddr := fmt.Sprintf("%s/p2p/%s", addrs[0].String(), hostB.ID().String())

	// Query B's price from A
	resp, err := svcA.QueryPeerPrice(ctx, peerAddr)
	if err != nil {
		t.Fatalf("QueryPeerPrice: %v", err)
	}

	if resp.PricePerSecond != 5000 {
		t.Errorf("expected price 5000, got %d", resp.PricePerSecond)
	}
	if resp.NodeID != hostB.ID().String() {
		t.Errorf("expected node ID %s, got %s", hostB.ID().String(), resp.NodeID)
	}
}

func TestPriceQuery_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	hostA, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("create host A: %v", err)
	}
	defer hostA.Close()

	svcA := NewService(hostA, 1000, testLogger())

	// Query an unreachable peer — should fail
	_, err = svcA.QueryPeerPrice(ctx, "/ip4/127.0.0.1/tcp/59999/p2p/12D3KooWDpJ7As7BWAwRMfu1VU2WCqNjvq387JEYKDBj4kx6nXTN")
	if err == nil {
		t.Fatal("expected error for unreachable peer")
	}
}

func TestPriceQuery_InvalidAddress(t *testing.T) {
	hostA, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("create host A: %v", err)
	}
	defer hostA.Close()

	svcA := NewService(hostA, 1000, testLogger())

	_, err = svcA.QueryPeerPrice(context.Background(), "not-a-valid-multiaddr")
	if err == nil {
		t.Fatal("expected error for invalid address")
	}
}
