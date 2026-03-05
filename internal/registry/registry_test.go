// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nil, nil))
}

func testPeerID(suffix byte) peer.ID {
	// Create a deterministic peer ID for testing.
	b := make([]byte, 32)
	b[0] = suffix
	id, _ := peer.IDFromBytes(append([]byte{0x00, 0x24, 0x08, 0x01, 0x12, 0x20}, b...))
	return id
}

func TestAdd_NewPeer(t *testing.T) {
	r := New(testLogger())
	id := testPeerID(1)

	r.Add(PeerInfo{
		ID:             id,
		Addrs:          []string{"/ip4/127.0.0.1/tcp/4001"},
		PricePerSecond: 1000,
	})

	got := r.Get(id)
	if got == nil {
		t.Fatal("expected peer to be found")
	}
	if got.PricePerSecond != 1000 {
		t.Errorf("expected price 1000, got %d", got.PricePerSecond)
	}
	if r.Len() != 1 {
		t.Errorf("expected len 1, got %d", r.Len())
	}
}

func TestAdd_UpdateExisting(t *testing.T) {
	r := New(testLogger())
	id := testPeerID(1)

	r.Add(PeerInfo{ID: id, PricePerSecond: 1000})
	r.Add(PeerInfo{ID: id, PricePerSecond: 2000})

	got := r.Get(id)
	if got.PricePerSecond != 2000 {
		t.Errorf("expected updated price 2000, got %d", got.PricePerSecond)
	}
	if r.Len() != 1 {
		t.Errorf("expected len 1, got %d", r.Len())
	}
}

func TestAdd_MergePreservesExistingFields(t *testing.T) {
	r := New(testLogger())
	id := testPeerID(1)

	r.Add(PeerInfo{
		ID:             id,
		Addrs:          []string{"/ip4/127.0.0.1/tcp/4001"},
		PricePerSecond: 1000,
	})
	// Update price only — addrs should be preserved.
	r.Add(PeerInfo{ID: id, PricePerSecond: 2000})

	got := r.Get(id)
	if len(got.Addrs) != 1 {
		t.Errorf("expected addrs preserved, got %v", got.Addrs)
	}
	if got.PricePerSecond != 2000 {
		t.Errorf("expected price 2000, got %d", got.PricePerSecond)
	}
}

func TestRemove(t *testing.T) {
	r := New(testLogger())
	id := testPeerID(1)
	r.Add(PeerInfo{ID: id})
	r.Remove(id)

	if r.Get(id) != nil {
		t.Error("expected peer to be removed")
	}
	if r.Len() != 0 {
		t.Errorf("expected len 0, got %d", r.Len())
	}
}

func TestGet_NotFound(t *testing.T) {
	r := New(testLogger())
	if r.Get(testPeerID(99)) != nil {
		t.Error("expected nil for unknown peer")
	}
}

func TestRecordSuccess_ResetsFailCount(t *testing.T) {
	r := New(testLogger())
	id := testPeerID(1)
	r.Add(PeerInfo{ID: id})

	r.RecordFailure(id)
	r.RecordFailure(id)
	r.RecordSuccess(id)

	got := r.Get(id)
	if got.FailCount != 0 {
		t.Errorf("expected fail count 0 after success, got %d", got.FailCount)
	}
	if got.LastSeen.IsZero() {
		t.Error("expected LastSeen to be set")
	}
}

func TestRecordFailure_IncrementsCount(t *testing.T) {
	r := New(testLogger())
	id := testPeerID(1)
	r.Add(PeerInfo{ID: id})

	r.RecordFailure(id)
	r.RecordFailure(id)

	got := r.Get(id)
	if got.FailCount != 2 {
		t.Errorf("expected fail count 2, got %d", got.FailCount)
	}
}

func TestSelectCandidates_FilterByPrice(t *testing.T) {
	r := New(testLogger())
	cheap := testPeerID(1)
	expensive := testPeerID(2)

	r.Add(PeerInfo{ID: cheap, PricePerSecond: 500})
	r.Add(PeerInfo{ID: expensive, PricePerSecond: 5000})

	candidates := r.SelectCandidates(1000, nil, nil, -1)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].ID != cheap {
		t.Error("expected cheap peer")
	}
}

func TestSelectCandidates_FilterByCapabilities(t *testing.T) {
	r := New(testLogger())
	full := testPeerID(1)
	partial := testPeerID(2)

	r.Add(PeerInfo{ID: full, Capabilities: []string{"clock", "rand", "log"}})
	r.Add(PeerInfo{ID: partial, Capabilities: []string{"clock"}})

	candidates := r.SelectCandidates(0, []string{"clock", "rand"}, nil, -1)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].ID != full {
		t.Error("expected full-capability peer")
	}
}

func TestSelectCandidates_UnknownCapsOptimistic(t *testing.T) {
	r := New(testLogger())
	unknown := testPeerID(1)
	r.Add(PeerInfo{ID: unknown}) // nil capabilities

	candidates := r.SelectCandidates(0, []string{"clock"}, nil, -1)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate (optimistic), got %d", len(candidates))
	}
}

func TestSelectCandidates_FilterByFailCount(t *testing.T) {
	r := New(testLogger())
	healthy := testPeerID(1)
	failing := testPeerID(2)

	r.Add(PeerInfo{ID: healthy})
	r.Add(PeerInfo{ID: failing})
	r.RecordFailure(failing)
	r.RecordFailure(failing)
	r.RecordFailure(failing)

	candidates := r.SelectCandidates(0, nil, nil, 2)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].ID != healthy {
		t.Error("expected healthy peer")
	}
}

func TestSelectCandidates_ExcludeIDs(t *testing.T) {
	r := New(testLogger())
	a := testPeerID(1)
	b := testPeerID(2)
	c := testPeerID(3)

	r.Add(PeerInfo{ID: a})
	r.Add(PeerInfo{ID: b})
	r.Add(PeerInfo{ID: c})

	candidates := r.SelectCandidates(0, nil, []peer.ID{b}, -1)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	for _, cand := range candidates {
		if cand.ID == b {
			t.Error("excluded peer should not be in candidates")
		}
	}
}

func TestSelectCandidates_SortOrder(t *testing.T) {
	r := New(testLogger())
	cheap := testPeerID(1)
	mid := testPeerID(2)
	expensive := testPeerID(3)

	r.Add(PeerInfo{ID: expensive, PricePerSecond: 3000})
	r.Add(PeerInfo{ID: cheap, PricePerSecond: 1000})
	r.Add(PeerInfo{ID: mid, PricePerSecond: 2000})

	candidates := r.SelectCandidates(0, nil, nil, -1)
	if len(candidates) != 3 {
		t.Fatalf("expected 3 candidates, got %d", len(candidates))
	}
	if candidates[0].PricePerSecond != 1000 {
		t.Errorf("expected cheapest first, got price %d", candidates[0].PricePerSecond)
	}
	if candidates[1].PricePerSecond != 2000 {
		t.Errorf("expected mid second, got price %d", candidates[1].PricePerSecond)
	}
	if candidates[2].PricePerSecond != 3000 {
		t.Errorf("expected expensive last, got price %d", candidates[2].PricePerSecond)
	}
}

func TestSelectCandidates_UnknownPriceSortedLast(t *testing.T) {
	r := New(testLogger())
	known := testPeerID(1)
	unknown := testPeerID(2)

	r.Add(PeerInfo{ID: known, PricePerSecond: 1000})
	r.Add(PeerInfo{ID: unknown, PricePerSecond: 0})

	candidates := r.SelectCandidates(0, nil, nil, -1)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].ID != known {
		t.Error("expected known-price peer first")
	}
}

func TestSelectCandidates_SortByFailCountThenRecency(t *testing.T) {
	r := New(testLogger())
	a := testPeerID(1)
	b := testPeerID(2)

	now := time.Now()
	r.Add(PeerInfo{ID: a, PricePerSecond: 1000, LastSeen: now.Add(-time.Minute)})
	r.Add(PeerInfo{ID: b, PricePerSecond: 1000, LastSeen: now})

	candidates := r.SelectCandidates(0, nil, nil, -1)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	// Same price, same fail count → most recently seen first.
	if candidates[0].ID != b {
		t.Error("expected most recently seen peer first")
	}
}

func TestAll(t *testing.T) {
	r := New(testLogger())
	r.Add(PeerInfo{ID: testPeerID(1)})
	r.Add(PeerInfo{ID: testPeerID(2)})

	all := r.All()
	if len(all) != 2 {
		t.Errorf("expected 2 peers, got %d", len(all))
	}
}

func TestSelectCandidates_NoPriceFilter(t *testing.T) {
	r := New(testLogger())
	r.Add(PeerInfo{ID: testPeerID(1), PricePerSecond: 999999})

	// maxPrice <= 0 means no price filter.
	candidates := r.SelectCandidates(0, nil, nil, -1)
	if len(candidates) != 1 {
		t.Errorf("expected 1 candidate with no price filter, got %d", len(candidates))
	}
}
