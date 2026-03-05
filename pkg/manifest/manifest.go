// SPDX-License-Identifier: Apache-2.0

package manifest

import "sort"

const (
	// DefaultMaxMemoryBytes is the default WASM memory limit (64MB).
	DefaultMaxMemoryBytes = 64 * 1024 * 1024

	// WASMPageSize is the size of a single WASM memory page.
	WASMPageSize = 65536
)

// Manifest describes an agent's identity, requirements, and policies.
type Manifest struct {
	// AgentID is the unique identifier for this agent.
	AgentID string

	// OwnerPublicKey is the cryptographic public key of the agent's owner.
	OwnerPublicKey []byte

	// Capabilities declares the runtime capabilities this agent requires.
	// Parsed from the agent's manifest.json.
	Capabilities *CapabilityManifest

	// MigrationPolicy defines rules for when and how the agent can migrate.
	// nil means no policy specified — migration is allowed by default.
	MigrationPolicy *MigrationPolicy

	// ResourceLimits defines the maximum resources the agent may consume.
	ResourceLimits ResourceLimits
}

// CapabilityManifest declares which capabilities an agent requires.
// Per CM-2, agents must declare capabilities before execution begins.
// Per CM-3, undeclared capabilities are unavailable (deny by default).
type CapabilityManifest struct {
	Capabilities map[string]CapabilityConfig `json:"capabilities"`
}

// CapabilityConfig specifies version and options for a single capability.
type CapabilityConfig struct {
	Version int            `json:"version"`
	Options map[string]any `json:"options,omitempty"`
}

// Has returns true if the manifest declares the named capability.
func (m *CapabilityManifest) Has(name string) bool {
	if m == nil || m.Capabilities == nil {
		return false
	}
	_, ok := m.Capabilities[name]
	return ok
}

// Names returns the sorted list of declared capability names.
func (m *CapabilityManifest) Names() []string {
	if m == nil || m.Capabilities == nil {
		return nil
	}
	names := make([]string, 0, len(m.Capabilities))
	for name := range m.Capabilities {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// MigrationPolicy specifies migration behavior constraints.
type MigrationPolicy struct {
	// Enabled indicates whether migration is allowed.
	Enabled bool

	// MaxPricePerSecond is the maximum price the agent is willing to pay (microcents).
	MaxPricePerSecond int64

	// PreferredRegions is a list of geographic preferences (optional).
	PreferredRegions []string
}

// ResourceLimits defines runtime resource constraints.
type ResourceLimits struct {
	// MaxMemoryBytes is the maximum memory the agent may use.
	MaxMemoryBytes uint64

	// MaxCPUMillis is the maximum CPU time per tick in milliseconds.
	MaxCPUMillis uint64

	// MaxStorageBytes is the maximum persistent storage allowed.
	MaxStorageBytes uint64
}

// MemoryLimitPages returns the memory limit in WASM pages.
// Returns the default (1024 pages = 64MB) if MaxMemoryBytes is 0.
func (r ResourceLimits) MemoryLimitPages() uint32 {
	if r.MaxMemoryBytes == 0 {
		return uint32(DefaultMaxMemoryBytes / WASMPageSize)
	}
	pages := r.MaxMemoryBytes / WASMPageSize
	if pages == 0 {
		pages = 1
	}
	if pages > 65536 {
		pages = 65536
	}
	return uint32(pages)
}
