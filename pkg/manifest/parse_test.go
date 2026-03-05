package manifest

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestParseCapabilityManifest_Valid(t *testing.T) {
	data := []byte(`{
		"capabilities": {
			"clock": {"version": 1},
			"rand": {"version": 1},
			"log": {"version": 1}
		}
	}`)

	m, err := ParseCapabilityManifest(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !m.Has("clock") {
		t.Error("expected clock capability")
	}
	if !m.Has("rand") {
		t.Error("expected rand capability")
	}
	if !m.Has("log") {
		t.Error("expected log capability")
	}
	if m.Has("net") {
		t.Error("unexpected net capability")
	}
}

func TestParseCapabilityManifest_Empty(t *testing.T) {
	m, err := ParseCapabilityManifest(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Has("clock") {
		t.Error("empty manifest should have no capabilities")
	}
}

func TestParseCapabilityManifest_EmptyJSON(t *testing.T) {
	m, err := ParseCapabilityManifest([]byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.Names()) != 0 {
		t.Errorf("expected 0 capabilities, got %d", len(m.Names()))
	}
}

func TestParseCapabilityManifest_InvalidJSON(t *testing.T) {
	_, err := ParseCapabilityManifest([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseCapabilityManifest_InvalidVersion(t *testing.T) {
	data := []byte(`{"capabilities": {"clock": {"version": 0}}}`)
	_, err := ParseCapabilityManifest(data)
	if err == nil {
		t.Error("expected error for version 0")
	}
}

func TestParseCapabilityManifest_WithOptions(t *testing.T) {
	data := []byte(`{
		"capabilities": {
			"kv": {"version": 1, "options": {"max_key_size": 256}}
		}
	}`)

	m, err := ParseCapabilityManifest(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !m.Has("kv") {
		t.Error("expected kv capability")
	}

	cfg := m.Capabilities["kv"]
	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.Options["max_key_size"] != float64(256) {
		t.Errorf("expected max_key_size 256, got %v", cfg.Options["max_key_size"])
	}
}

func TestValidateAgainstNode_AllSatisfied(t *testing.T) {
	m, _ := ParseCapabilityManifest([]byte(`{
		"capabilities": {
			"clock": {"version": 1},
			"rand": {"version": 1}
		}
	}`))

	err := ValidateAgainstNode(m, []string{"clock", "rand", "log"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAgainstNode_Missing(t *testing.T) {
	m, _ := ParseCapabilityManifest([]byte(`{
		"capabilities": {
			"clock": {"version": 1},
			"net": {"version": 1}
		}
	}`))

	err := ValidateAgainstNode(m, []string{"clock", "rand", "log"})
	if err == nil {
		t.Error("expected error for missing 'net' capability")
	}
}

func TestValidateAgainstNode_EmptyManifest(t *testing.T) {
	err := ValidateAgainstNode(nil, []string{"clock"})
	if err != nil {
		t.Fatalf("unexpected error for nil manifest: %v", err)
	}

	m, _ := ParseCapabilityManifest([]byte(`{}`))
	err = ValidateAgainstNode(m, []string{"clock"})
	if err != nil {
		t.Fatalf("unexpected error for empty manifest: %v", err)
	}
}

func TestCapabilityManifest_Names(t *testing.T) {
	m, _ := ParseCapabilityManifest([]byte(`{
		"capabilities": {
			"rand": {"version": 1},
			"clock": {"version": 1}
		}
	}`))

	names := m.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
}

func TestCapabilityManifest_NilSafety(t *testing.T) {
	var m *CapabilityManifest
	if m.Has("anything") {
		t.Error("nil manifest should not have any capabilities")
	}
	if m.Names() != nil {
		t.Error("nil manifest should return nil names")
	}
}

// --- ParseManifest tests ---

func TestParseManifest_Full(t *testing.T) {
	data := []byte(`{
		"capabilities": {
			"clock": {"version": 1},
			"rand": {"version": 1}
		},
		"resource_limits": {
			"max_memory_bytes": 33554432,
			"max_cpu_millis": 50,
			"max_storage_bytes": 1048576
		},
		"migration_policy": {
			"enabled": true,
			"max_price_per_second": 5000,
			"preferred_regions": ["us-east", "eu-west"]
		}
	}`)

	m, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !m.Capabilities.Has("clock") {
		t.Error("expected clock capability")
	}
	if !m.Capabilities.Has("rand") {
		t.Error("expected rand capability")
	}

	if m.ResourceLimits.MaxMemoryBytes != 33554432 {
		t.Errorf("MaxMemoryBytes: got %d, want 33554432", m.ResourceLimits.MaxMemoryBytes)
	}
	if m.ResourceLimits.MaxCPUMillis != 50 {
		t.Errorf("MaxCPUMillis: got %d, want 50", m.ResourceLimits.MaxCPUMillis)
	}
	if m.ResourceLimits.MaxStorageBytes != 1048576 {
		t.Errorf("MaxStorageBytes: got %d, want 1048576", m.ResourceLimits.MaxStorageBytes)
	}

	if m.MigrationPolicy == nil {
		t.Fatal("MigrationPolicy should not be nil")
	}
	if !m.MigrationPolicy.Enabled {
		t.Error("MigrationPolicy.Enabled: expected true")
	}
	if m.MigrationPolicy.MaxPricePerSecond != 5000 {
		t.Errorf("MaxPricePerSecond: got %d, want 5000", m.MigrationPolicy.MaxPricePerSecond)
	}
	if len(m.MigrationPolicy.PreferredRegions) != 2 {
		t.Errorf("PreferredRegions: got %d, want 2", len(m.MigrationPolicy.PreferredRegions))
	}
}

func TestParseManifest_CapabilitiesOnly(t *testing.T) {
	data := []byte(`{"capabilities": {"log": {"version": 1}}}`)

	m, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !m.Capabilities.Has("log") {
		t.Error("expected log capability")
	}
	if m.MigrationPolicy != nil {
		t.Error("MigrationPolicy should be nil when absent")
	}
	if m.ResourceLimits.MaxMemoryBytes != 0 {
		t.Error("ResourceLimits should be zero-value when absent")
	}
}

func TestParseManifest_Empty(t *testing.T) {
	m, err := ParseManifest(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Capabilities == nil {
		t.Error("Capabilities should not be nil for empty input")
	}
	if m.MigrationPolicy != nil {
		t.Error("MigrationPolicy should be nil for empty input")
	}
}

func TestParseManifest_EmptyJSON(t *testing.T) {
	m, err := ParseManifest([]byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.Capabilities.Names()) != 0 {
		t.Errorf("expected 0 capabilities, got %d", len(m.Capabilities.Names()))
	}
}

func TestParseManifest_InvalidJSON(t *testing.T) {
	_, err := ParseManifest([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseManifest_InvalidCapabilityVersion(t *testing.T) {
	data := []byte(`{"capabilities": {"clock": {"version": 0}}}`)
	_, err := ParseManifest(data)
	if err == nil {
		t.Error("expected error for version 0")
	}
}

func TestParseManifest_MigrationDisabled(t *testing.T) {
	data := []byte(`{
		"capabilities": {},
		"migration_policy": {"enabled": false}
	}`)

	m, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.MigrationPolicy == nil {
		t.Fatal("MigrationPolicy should not be nil when present")
	}
	if m.MigrationPolicy.Enabled {
		t.Error("MigrationPolicy.Enabled: expected false")
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestLoadSidecarData_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	mPath := filepath.Join(dir, "custom.manifest.json")
	if err := os.WriteFile(mPath, []byte(`{"capabilities":{"clock":{"version":1}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	data := LoadSidecarData("irrelevant.wasm", mPath, testLogger())
	if string(data) != `{"capabilities":{"clock":{"version":1}}}` {
		t.Fatalf("unexpected manifest: %s", data)
	}
}

func TestLoadSidecarData_DerivedPath(t *testing.T) {
	dir := t.TempDir()
	mPath := filepath.Join(dir, "agent.manifest.json")
	if err := os.WriteFile(mPath, []byte(`{"capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	data := LoadSidecarData(filepath.Join(dir, "agent.wasm"), "", testLogger())
	if string(data) != `{"capabilities":{}}` {
		t.Fatalf("unexpected manifest: %s", data)
	}
}

func TestLoadSidecarData_NoFile(t *testing.T) {
	data := LoadSidecarData("/tmp/nonexistent.wasm", "", testLogger())
	if string(data) != "{}" {
		t.Fatalf("expected empty JSON, got: %s", data)
	}
}

func TestLoadSidecarData_NonWASMPath(t *testing.T) {
	data := LoadSidecarData("agent", "", testLogger())
	if string(data) != "{}" {
		t.Fatalf("expected empty JSON for non-.wasm path, got: %s", data)
	}
}

func TestResourceLimits_MemoryLimitPages(t *testing.T) {
	tests := []struct {
		name           string
		maxMemoryBytes uint64
		wantPages      uint32
	}{
		{"zero_uses_default", 0, 1024},
		{"exact_64MB", 64 * 1024 * 1024, 1024},
		{"32MB", 32 * 1024 * 1024, 512},
		{"1_page", 65536, 1},
		{"less_than_1_page", 100, 1},
		{"max_capped", 65536 * 65536 * 2, 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := ResourceLimits{MaxMemoryBytes: tt.maxMemoryBytes}
			got := rl.MemoryLimitPages()
			if got != tt.wantPages {
				t.Errorf("MemoryLimitPages(): got %d, want %d", got, tt.wantPages)
			}
		})
	}
}
