package manifest

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
)

// NodeCapabilities lists capabilities available on the current node.
var NodeCapabilities = []string{"clock", "rand", "log", "wallet", "pricing"}

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

// manifestJSON is the raw JSON structure for a full manifest.
type manifestJSON struct {
	Capabilities    map[string]CapabilityConfig `json:"capabilities"`
	ResourceLimits  *resourceLimitsJSON         `json:"resource_limits,omitempty"`
	MigrationPolicy *migrationPolicyJSON        `json:"migration_policy,omitempty"`
}

type resourceLimitsJSON struct {
	MaxMemoryBytes  uint64 `json:"max_memory_bytes,omitempty"`
	MaxCPUMillis    uint64 `json:"max_cpu_millis,omitempty"`
	MaxStorageBytes uint64 `json:"max_storage_bytes,omitempty"`
}

type migrationPolicyJSON struct {
	Enabled           bool     `json:"enabled"`
	MaxPricePerSecond int64    `json:"max_price_per_second,omitempty"`
	PreferredRegions  []string `json:"preferred_regions,omitempty"`
}

// ParseManifest parses a full manifest from JSON bytes.
// Returns a Manifest with Capabilities, ResourceLimits, and MigrationPolicy.
// Missing sections get zero-value defaults. MigrationPolicy is nil when absent
// (meaning migration is allowed by default for backward compatibility).
func ParseManifest(data []byte) (*Manifest, error) {
	if len(data) == 0 {
		return &Manifest{
			Capabilities: &CapabilityManifest{
				Capabilities: make(map[string]CapabilityConfig),
			},
		}, nil
	}

	var raw manifestJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	// Build capability manifest
	caps := &CapabilityManifest{Capabilities: raw.Capabilities}
	if caps.Capabilities == nil {
		caps.Capabilities = make(map[string]CapabilityConfig)
	}
	for name, cfg := range caps.Capabilities {
		if cfg.Version < 1 {
			return nil, fmt.Errorf("capability %q: version must be >= 1, got %d", name, cfg.Version)
		}
	}

	m := &Manifest{Capabilities: caps}

	if raw.ResourceLimits != nil {
		m.ResourceLimits = ResourceLimits{
			MaxMemoryBytes:  raw.ResourceLimits.MaxMemoryBytes,
			MaxCPUMillis:    raw.ResourceLimits.MaxCPUMillis,
			MaxStorageBytes: raw.ResourceLimits.MaxStorageBytes,
		}
	}

	if raw.MigrationPolicy != nil {
		m.MigrationPolicy = &MigrationPolicy{
			Enabled:           raw.MigrationPolicy.Enabled,
			MaxPricePerSecond: raw.MigrationPolicy.MaxPricePerSecond,
			PreferredRegions:  raw.MigrationPolicy.PreferredRegions,
		}
	}

	return m, nil
}

// LoadSidecarData loads manifest data from a sidecar file. If manifestPath is
// non-empty it is used directly; otherwise the manifest path is derived by
// replacing the ".wasm" suffix of wasmPath with ".manifest.json". Returns
// "{}" when no manifest can be read.
func LoadSidecarData(wasmPath, manifestPath string, logger *slog.Logger) []byte {
	mPath := manifestPath
	if mPath == "" {
		if strings.HasSuffix(wasmPath, ".wasm") {
			mPath = strings.TrimSuffix(wasmPath, ".wasm") + ".manifest.json"
		}
	}
	data, err := os.ReadFile(mPath)
	if err != nil {
		logger.Info("No manifest file found, using empty capabilities",
			"expected_path", mPath,
		)
		return []byte("{}")
	}
	logger.Info("Manifest loaded", "path", mPath)
	return data
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
