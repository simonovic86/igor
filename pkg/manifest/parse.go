package manifest

import (
	"encoding/json"
	"fmt"
	"sort"
)

// NodeCapabilities lists capabilities available on the current node.
// MVP supports: clock, rand, log.
var NodeCapabilities = []string{"clock", "rand", "log"}

// ParseCapabilityManifest parses a capability manifest from JSON bytes.
// An empty or nil input returns an empty manifest (no capabilities declared).
func ParseCapabilityManifest(data []byte) (*CapabilityManifest, error) {
	if len(data) == 0 {
		return &CapabilityManifest{}, nil
	}

	var m CapabilityManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse capability manifest: %w", err)
	}

	if m.Capabilities == nil {
		m.Capabilities = make(map[string]CapabilityConfig)
	}

	// Validate version fields
	for name, cfg := range m.Capabilities {
		if cfg.Version < 1 {
			return nil, fmt.Errorf("capability %q: version must be >= 1, got %d", name, cfg.Version)
		}
	}

	return &m, nil
}

// ValidateAgainstNode checks that every declared capability is available on
// the given node. Returns an error listing all unsatisfied capabilities.
func ValidateAgainstNode(manifest *CapabilityManifest, nodeCapabilities []string) error {
	if manifest == nil || len(manifest.Capabilities) == 0 {
		return nil
	}

	available := make(map[string]bool, len(nodeCapabilities))
	for _, cap := range nodeCapabilities {
		available[cap] = true
	}

	var missing []string
	for name := range manifest.Capabilities {
		if !available[name] {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("node cannot satisfy capabilities: %v", missing)
	}

	return nil
}
