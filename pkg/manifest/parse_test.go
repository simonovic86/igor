package manifest

import (
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
