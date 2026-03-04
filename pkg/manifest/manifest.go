package manifest

import "sort"

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
	MigrationPolicy MigrationPolicy

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
