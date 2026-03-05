// SPDX-License-Identifier: Apache-2.0

// Package registry tracks known peers with cached prices, capabilities,
// and health information. It supports filtered peer selection for migration
// fallback and divergence-triggered migration.
package registry

import (
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// PeerInfo holds cached information about a known peer.
type PeerInfo struct {
	ID             peer.ID
	Addrs          []string // multiaddrs
	PricePerSecond int64    // last known price in microcents; 0 = unknown
	Capabilities   []string // last known capabilities; nil = unknown
	LastSeen       time.Time
	FailCount      int // consecutive failures
}

// Registry tracks known peers with cached prices and health.
type Registry struct {
	mu     sync.RWMutex
	peers  map[peer.ID]*PeerInfo
	logger *slog.Logger
}

// New creates a peer registry.
func New(logger *slog.Logger) *Registry {
	return &Registry{
		peers:  make(map[peer.ID]*PeerInfo),
		logger: logger,
	}
}

// Add registers or updates a peer. Idempotent — merges with existing data.
func (r *Registry) Add(info PeerInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.peers[info.ID]
	if !ok {
		cp := info
		r.peers[info.ID] = &cp
		return
	}

	// Merge: update non-zero fields.
	if len(info.Addrs) > 0 {
		existing.Addrs = info.Addrs
	}
	if info.PricePerSecond > 0 {
		existing.PricePerSecond = info.PricePerSecond
	}
	if info.Capabilities != nil {
		existing.Capabilities = info.Capabilities
	}
	if !info.LastSeen.IsZero() {
		existing.LastSeen = info.LastSeen
	}
}

// Remove deletes a peer from the registry.
func (r *Registry) Remove(id peer.ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.peers, id)
}

// Get returns peer info if known, nil otherwise.
func (r *Registry) Get(id peer.ID) *PeerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.peers[id]
	if !ok {
		return nil
	}
	cp := *p
	return &cp
}

// RecordSuccess resets fail count and updates LastSeen.
func (r *Registry) RecordSuccess(id peer.ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.peers[id]; ok {
		p.FailCount = 0
		p.LastSeen = time.Now()
	}
}

// RecordFailure increments fail count for a peer.
func (r *Registry) RecordFailure(id peer.ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.peers[id]; ok {
		p.FailCount++
	}
}

// SelectCandidates returns peers matching the given constraints, sorted by
// preference (lowest price first, then fewest failures, then most recently seen).
// Excludes peers above maxFailures and those in excludeIDs.
// maxPrice <= 0 means no price filter. maxFailures < 0 means no failure filter.
func (r *Registry) SelectCandidates(
	maxPrice int64,
	requiredCaps []string,
	excludeIDs []peer.ID,
	maxFailures int,
) []*PeerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	excludeSet := make(map[peer.ID]struct{}, len(excludeIDs))
	for _, id := range excludeIDs {
		excludeSet[id] = struct{}{}
	}

	var candidates []*PeerInfo
	for _, p := range r.peers {
		if _, excluded := excludeSet[p.ID]; excluded {
			continue
		}
		if maxFailures >= 0 && p.FailCount > maxFailures {
			continue
		}
		if maxPrice > 0 && p.PricePerSecond > 0 && p.PricePerSecond > maxPrice {
			continue
		}
		if !hasCapabilities(p.Capabilities, requiredCaps) {
			continue
		}
		cp := *p
		candidates = append(candidates, &cp)
	}

	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		// Prefer known prices over unknown.
		if a.PricePerSecond != b.PricePerSecond {
			// 0 = unknown, sort last.
			if a.PricePerSecond == 0 {
				return false
			}
			if b.PricePerSecond == 0 {
				return true
			}
			return a.PricePerSecond < b.PricePerSecond
		}
		if a.FailCount != b.FailCount {
			return a.FailCount < b.FailCount
		}
		return a.LastSeen.After(b.LastSeen)
	})

	return candidates
}

// All returns a snapshot of all known peers.
func (r *Registry) All() []*PeerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*PeerInfo, 0, len(r.peers))
	for _, p := range r.peers {
		cp := *p
		out = append(out, &cp)
	}
	return out
}

// Len returns the number of known peers.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.peers)
}

// hasCapabilities returns true if peerCaps contains all of requiredCaps.
// If requiredCaps is empty, always returns true.
// If peerCaps is nil (unknown), returns true (optimistic — let the target reject).
func hasCapabilities(peerCaps, requiredCaps []string) bool {
	if len(requiredCaps) == 0 {
		return true
	}
	if peerCaps == nil {
		return true // unknown caps — optimistic
	}
	capSet := make(map[string]struct{}, len(peerCaps))
	for _, c := range peerCaps {
		capSet[c] = struct{}{}
	}
	for _, req := range requiredCaps {
		if _, ok := capSet[req]; !ok {
			return false
		}
	}
	return true
}
