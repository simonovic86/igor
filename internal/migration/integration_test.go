// SPDX-License-Identifier: Apache-2.0

package migration

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/internal/authority"
	"github.com/simonovic86/igor/internal/runtime"
	"github.com/simonovic86/igor/internal/storage"
	"github.com/simonovic86/igor/pkg/budget"
)

// integrationTestAgentSource is a minimal TinyGo agent that imports igor hostcalls.
const integrationTestAgentSource = `package main

import "unsafe"

//go:wasmimport igor clock_now
func clockNow() int64

//go:wasmimport igor rand_bytes
func randBytes(ptr uint32, length uint32) int32

//go:wasmimport igor log_emit
func logEmit(ptr uint32, length uint32)

var counter uint64

//export agent_init
func agent_init() { counter = 0 }

//export agent_tick
func agent_tick() {
	counter++
	_ = clockNow()
	var buf [4]byte
	randBytes(uint32(uintptr(unsafe.Pointer(&buf[0]))), 4)
	msg := []byte("tick")
	logEmit(uint32(uintptr(unsafe.Pointer(&msg[0]))), uint32(len(msg)))
}

//export agent_checkpoint
func agent_checkpoint() uint32 { return 8 }

//export agent_checkpoint_ptr
func agent_checkpoint_ptr() uint32 {
	return uint32(uintptr(unsafe.Pointer(&counter)))
}

//export agent_resume
func agent_resume(ptr, size uint32) {
	if size >= 8 {
		buf := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
		counter = *(*uint64)(unsafe.Pointer(&buf[0]))
	}
}

func main() {}
`

// buildIntegrationTestAgent compiles a TinyGo WASM agent and writes a manifest
// sidecar alongside it. Returns the WASM file path.
func buildIntegrationTestAgent(t *testing.T) string {
	t.Helper()

	tinygoTool, err := exec.LookPath("tinygo")
	if err != nil {
		t.Skip("tinygo not found, skipping integration test")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.go")
	wasmPath := filepath.Join(dir, "agent.wasm")

	if err := os.WriteFile(srcPath, []byte(integrationTestAgentSource), 0o644); err != nil {
		t.Fatalf("write agent source: %v", err)
	}

	goMod := "module testagent\n\ngo 1.24\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	cmd := exec.Command(tinygoTool, "build", "-target=wasi", "-no-debug", "-o", wasmPath, ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("failed to build test WASM agent: %s\n%s", err, out)
	}

	// Write manifest sidecar (required by MigrateAgent's manifest lookup)
	manifestPath := filepath.Join(dir, "agent.manifest.json")
	manifestJSON := `{"capabilities":{"clock":{"version":1},"rand":{"version":1},"log":{"version":1}}}`
	if err := os.WriteFile(manifestPath, []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	return wasmPath
}

func integrationTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// migrationEnv holds two in-process nodes wired for migration testing.
type migrationEnv struct {
	ctx     context.Context
	agentID string

	hostA, hostB       host.Host
	storageA, storageB *storage.FSProvider
	engineA, engineB   *runtime.Engine
	migSvcA, migSvcB   *Service

	wasmPath string
	instance *agent.Instance

	budgetBeforeMigration int64
}

func newMigrationEnv(t *testing.T) *migrationEnv {
	t.Helper()
	ctx := context.Background()
	logger := integrationTestLogger()

	wasmPath := buildIntegrationTestAgent(t)

	listenAddr, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/0")

	hostA, err := libp2p.New(libp2p.ListenAddrs(listenAddr))
	if err != nil {
		t.Fatalf("create hostA: %v", err)
	}
	t.Cleanup(func() { hostA.Close() })

	hostB, err := libp2p.New(libp2p.ListenAddrs(listenAddr))
	if err != nil {
		t.Fatalf("create hostB: %v", err)
	}
	t.Cleanup(func() { hostB.Close() })

	stA, err := storage.NewFSProvider(t.TempDir(), logger)
	if err != nil {
		t.Fatalf("create storageA: %v", err)
	}

	stB, err := storage.NewFSProvider(t.TempDir(), logger)
	if err != nil {
		t.Fatalf("create storageB: %v", err)
	}

	engA, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		t.Fatalf("create engineA: %v", err)
	}
	t.Cleanup(func() { engA.Close(ctx) })

	engB, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		t.Fatalf("create engineB: %v", err)
	}
	t.Cleanup(func() { engB.Close(ctx) })

	migA := NewService(hostA, engA, stA, "full", false, 1000, authority.LeaseConfig{}, logger)
	migB := NewService(hostB, engB, stB, "full", false, 1000, authority.LeaseConfig{}, logger)

	agentID := "integration-test-agent"
	manifestJSON := []byte(`{"capabilities":{"clock":{"version":1},"rand":{"version":1},"log":{"version":1}}}`)

	inst, err := agent.LoadAgent(ctx, engA, wasmPath, agentID, stA,
		budget.FromFloat(10.0), budget.FromFloat(0.001), manifestJSON, nil, "", nil, logger)
	if err != nil {
		t.Fatalf("LoadAgent: %v", err)
	}

	if err := inst.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}

	for i := 0; i < 3; i++ {
		if _, err := inst.Tick(ctx); err != nil {
			t.Fatalf("Tick %d: %v", i+1, err)
		}
	}

	if err := inst.SaveCheckpointToStorage(ctx); err != nil {
		t.Fatalf("SaveCheckpointToStorage: %v", err)
	}

	migA.RegisterAgent(agentID, inst)

	return &migrationEnv{
		ctx:                   ctx,
		agentID:               agentID,
		hostA:                 hostA,
		hostB:                 hostB,
		storageA:              stA,
		storageB:              stB,
		engineA:               engA,
		engineB:               engB,
		migSvcA:               migA,
		migSvcB:               migB,
		wasmPath:              wasmPath,
		instance:              inst,
		budgetBeforeMigration: inst.Budget,
	}
}

// buildIntegrationTestAgentWithManifest compiles a TinyGo WASM agent and writes
// a custom manifest sidecar alongside it. Returns the WASM file path.
func buildIntegrationTestAgentWithManifest(t *testing.T, manifestJSON string) string {
	t.Helper()

	tinygoTool, err := exec.LookPath("tinygo")
	if err != nil {
		t.Skip("tinygo not found, skipping integration test")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.go")
	wasmPath := filepath.Join(dir, "agent.wasm")

	if err := os.WriteFile(srcPath, []byte(integrationTestAgentSource), 0o644); err != nil {
		t.Fatalf("write agent source: %v", err)
	}

	goMod := "module testagent\n\ngo 1.24\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	cmd := exec.Command(tinygoTool, "build", "-target=wasi", "-no-debug", "-o", wasmPath, ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("failed to build test WASM agent: %s\n%s", err, out)
	}

	manifestPath := filepath.Join(dir, "agent.manifest.json")
	if err := os.WriteFile(manifestPath, []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	return wasmPath
}

func (env *migrationEnv) targetAddr() string {
	return fmt.Sprintf("%s/p2p/%s", env.hostB.Addrs()[0].String(), env.hostB.ID().String())
}

func TestMultiNodeMigration(t *testing.T) {
	env := newMigrationEnv(t)

	// Verify replay data is present before migration
	if len(env.instance.ReplayWindow) == 0 {
		t.Fatal("expected ReplayWindow to be populated after ticking")
	}

	// Migrate from A to B (replay verification happens on the target)
	if err := env.migSvcA.MigrateAgent(env.ctx, env.agentID, env.wasmPath, env.targetAddr()); err != nil {
		t.Fatalf("MigrateAgent: %v", err)
	}

	t.Run("source_cleaned_up", func(t *testing.T) {
		if agents := env.migSvcA.GetActiveAgents(); len(agents) != 0 {
			t.Errorf("source should have no active agents, got %v", agents)
		}

		_, err := env.storageA.LoadCheckpoint(env.ctx, env.agentID)
		if err != storage.ErrCheckpointNotFound {
			t.Errorf("source checkpoint should be deleted, got err=%v", err)
		}
	})

	t.Run("target_received_agent", func(t *testing.T) {
		agents := env.migSvcB.GetActiveAgents()
		if len(agents) != 1 || agents[0] != env.agentID {
			t.Fatalf("target should have agent %q, got %v", env.agentID, agents)
		}
	})

	t.Run("checkpoint_state_preserved", func(t *testing.T) {
		checkpoint, err := env.storageB.LoadCheckpoint(env.ctx, env.agentID)
		if err != nil {
			t.Fatalf("target LoadCheckpoint: %v", err)
		}

		if len(checkpoint) < 81 || checkpoint[0] != 0x03 {
			t.Fatalf("invalid checkpoint: len=%d, version=%d", len(checkpoint), checkpoint[0])
		}

		targetBudget := int64(binary.LittleEndian.Uint64(checkpoint[1:9]))
		if targetBudget != env.budgetBeforeMigration {
			t.Errorf("budget: got %d, want %d", targetBudget, env.budgetBeforeMigration)
		}

		counter := binary.LittleEndian.Uint64(checkpoint[81:])
		if counter != 3 {
			t.Errorf("counter: got %d, want 3", counter)
		}
	})

	t.Run("agent_continues_on_target", func(t *testing.T) {
		inst := env.migSvcB.GetActiveInstance(env.agentID)
		if inst == nil {
			t.Fatal("target instance should not be nil")
		}
		defer inst.Close(env.ctx)

		if _, err := inst.Tick(env.ctx); err != nil {
			t.Fatalf("target Tick: %v", err)
		}

		if err := inst.SaveCheckpointToStorage(env.ctx); err != nil {
			t.Fatalf("target SaveCheckpointToStorage: %v", err)
		}

		checkpoint, err := env.storageB.LoadCheckpoint(env.ctx, env.agentID)
		if err != nil {
			t.Fatalf("target LoadCheckpoint after tick: %v", err)
		}

		counter := binary.LittleEndian.Uint64(checkpoint[81:])
		if counter != 4 {
			t.Errorf("counter after tick: got %d, want 4", counter)
		}
	})
}

// newPolicyTestEnv creates a minimal two-node migration environment with a
// custom manifest. Uses the simpler "off" replay mode for faster tests.
func newPolicyTestEnv(t *testing.T, manifestJSON string) *migrationEnv {
	t.Helper()
	ctx := context.Background()
	logger := integrationTestLogger()

	wasmPath := buildIntegrationTestAgentWithManifest(t, manifestJSON)

	listenAddr, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/0")

	hostA, err := libp2p.New(libp2p.ListenAddrs(listenAddr))
	if err != nil {
		t.Fatalf("create hostA: %v", err)
	}
	t.Cleanup(func() { hostA.Close() })

	hostB, err := libp2p.New(libp2p.ListenAddrs(listenAddr))
	if err != nil {
		t.Fatalf("create hostB: %v", err)
	}
	t.Cleanup(func() { hostB.Close() })

	stA, err := storage.NewFSProvider(t.TempDir(), logger)
	if err != nil {
		t.Fatalf("create storageA: %v", err)
	}
	stB, err := storage.NewFSProvider(t.TempDir(), logger)
	if err != nil {
		t.Fatalf("create storageB: %v", err)
	}

	engA, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		t.Fatalf("create engineA: %v", err)
	}
	t.Cleanup(func() { engA.Close(ctx) })

	engB, err := runtime.NewEngine(ctx, logger)
	if err != nil {
		t.Fatalf("create engineB: %v", err)
	}
	t.Cleanup(func() { engB.Close(ctx) })

	// Target node charges 2000 microcents/sec
	migA := NewService(hostA, engA, stA, "off", false, 1000, authority.LeaseConfig{}, logger)
	migB := NewService(hostB, engB, stB, "off", false, 2000, authority.LeaseConfig{}, logger)

	agentID := "policy-test-agent"
	inst, err := agent.LoadAgent(ctx, engA, wasmPath, agentID, stA,
		budget.FromFloat(10.0), budget.FromFloat(0.001), []byte(manifestJSON), nil, "", nil, logger)
	if err != nil {
		t.Fatalf("LoadAgent: %v", err)
	}

	if err := inst.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if _, err := inst.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}

	if err := inst.SaveCheckpointToStorage(ctx); err != nil {
		t.Fatalf("SaveCheckpointToStorage: %v", err)
	}

	migA.RegisterAgent(agentID, inst)

	return &migrationEnv{
		ctx:      ctx,
		agentID:  agentID,
		hostA:    hostA,
		hostB:    hostB,
		storageA: stA,
		storageB: stB,
		engineA:  engA,
		engineB:  engB,
		migSvcA:  migA,
		migSvcB:  migB,
		wasmPath: wasmPath,
		instance: inst,
	}
}

func TestMigration_PolicyDisabled(t *testing.T) {
	manifestJSON := `{
		"capabilities": {"clock":{"version":1},"rand":{"version":1},"log":{"version":1}},
		"migration_policy": {"enabled": false}
	}`
	env := newPolicyTestEnv(t, manifestJSON)

	err := env.migSvcA.MigrateAgent(env.ctx, env.agentID, env.wasmPath, env.targetAddr())
	if err == nil {
		t.Fatal("expected error when migration policy is disabled")
	}
	t.Logf("Got expected error: %v", err)

	// Target should NOT have the agent
	if agents := env.migSvcB.GetActiveAgents(); len(agents) != 0 {
		t.Errorf("target should have no active agents, got %v", agents)
	}
}

func TestMigration_PriceTooHigh(t *testing.T) {
	// Agent allows max 1500 microcents/sec, but target charges 2000
	manifestJSON := `{
		"capabilities": {"clock":{"version":1},"rand":{"version":1},"log":{"version":1}},
		"migration_policy": {"enabled": true, "max_price_per_second": 1500}
	}`
	env := newPolicyTestEnv(t, manifestJSON)

	err := env.migSvcA.MigrateAgent(env.ctx, env.agentID, env.wasmPath, env.targetAddr())
	if err == nil {
		t.Fatal("expected error when node price exceeds agent max")
	}
	t.Logf("Got expected error: %v", err)
}

func TestMigration_NoPolicyAllowed(t *testing.T) {
	// No migration_policy — backward compatible, migration should succeed
	manifestJSON := `{
		"capabilities": {"clock":{"version":1},"rand":{"version":1},"log":{"version":1}}
	}`
	env := newPolicyTestEnv(t, manifestJSON)

	err := env.migSvcA.MigrateAgent(env.ctx, env.agentID, env.wasmPath, env.targetAddr())
	if err != nil {
		t.Fatalf("MigrateAgent: %v (expected success with no policy)", err)
	}

	// Target should have the agent
	agents := env.migSvcB.GetActiveAgents()
	if len(agents) != 1 || agents[0] != env.agentID {
		t.Errorf("target should have agent %q, got %v", env.agentID, agents)
	}
}

func TestMigration_PriceWithinLimit(t *testing.T) {
	// Agent allows max 5000 microcents/sec, target charges 2000 — should succeed
	manifestJSON := `{
		"capabilities": {"clock":{"version":1},"rand":{"version":1},"log":{"version":1}},
		"migration_policy": {"enabled": true, "max_price_per_second": 5000}
	}`
	env := newPolicyTestEnv(t, manifestJSON)

	err := env.migSvcA.MigrateAgent(env.ctx, env.agentID, env.wasmPath, env.targetAddr())
	if err != nil {
		t.Fatalf("MigrateAgent: %v (expected success with price within limit)", err)
	}

	agents := env.migSvcB.GetActiveAgents()
	if len(agents) != 1 || agents[0] != env.agentID {
		t.Errorf("target should have agent %q, got %v", env.agentID, agents)
	}
}
