package migration

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"github.com/simonovic86/igor/internal/agent"
	"github.com/simonovic86/igor/internal/runtime"
	"github.com/simonovic86/igor/internal/storage"
	"github.com/simonovic86/igor/pkg/budget"
)

// multiNodeEnv holds N in-process nodes wired for chain migration testing.
type multiNodeEnv struct {
	ctx     context.Context
	agentID string

	hosts    []host.Host
	storages []*storage.FSProvider
	engines  []*runtime.Engine
	migSvcs  []*Service

	wasmPath     string
	manifestJSON []byte
}

func multiNodeLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// newMultiNodeEnv creates nodeCount in-process libp2p nodes with independent
// storage, runtime engines, and migration services.
func newMultiNodeEnv(t *testing.T, nodeCount int) *multiNodeEnv {
	t.Helper()
	ctx := context.Background()
	logger := multiNodeLogger()

	wasmPath := buildIntegrationTestAgent(t)

	listenAddr, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/0")

	env := &multiNodeEnv{
		ctx:          ctx,
		agentID:      "multinode-test-agent",
		hosts:        make([]host.Host, nodeCount),
		storages:     make([]*storage.FSProvider, nodeCount),
		engines:      make([]*runtime.Engine, nodeCount),
		migSvcs:      make([]*Service, nodeCount),
		wasmPath:     wasmPath,
		manifestJSON: []byte(`{"capabilities":{"clock":{"version":1},"rand":{"version":1},"log":{"version":1}}}`),
	}

	for i := 0; i < nodeCount; i++ {
		h, err := libp2p.New(libp2p.ListenAddrs(listenAddr))
		if err != nil {
			t.Fatalf("create host[%d]: %v", i, err)
		}
		t.Cleanup(func() { h.Close() })
		env.hosts[i] = h

		st, err := storage.NewFSProvider(t.TempDir(), logger)
		if err != nil {
			t.Fatalf("create storage[%d]: %v", i, err)
		}
		env.storages[i] = st

		eng, err := runtime.NewEngine(ctx, logger)
		if err != nil {
			t.Fatalf("create engine[%d]: %v", i, err)
		}
		t.Cleanup(func() { eng.Close(ctx) })
		env.engines[i] = eng

		env.migSvcs[i] = NewService(h, eng, st, "full", false, logger)
	}

	return env
}

// addr returns the full multiaddr string for node i (including peer ID).
func (env *multiNodeEnv) addr(i int) string {
	return fmt.Sprintf("%s/p2p/%s", env.hosts[i].Addrs()[0].String(), env.hosts[i].ID().String())
}

// loadAndInitAgent creates an agent on node i, runs ticks, saves checkpoint,
// and registers with the migration service. Returns the instance.
func (env *multiNodeEnv) loadAndInitAgent(t *testing.T, nodeIdx int, budgetVal, price int64, ticks int) *agent.Instance { //nolint:unparam // nodeIdx is always 0 today but the helper is designed for any node
	t.Helper()
	logger := multiNodeLogger()

	inst, err := agent.LoadAgent(env.ctx, env.engines[nodeIdx], env.wasmPath, env.agentID,
		env.storages[nodeIdx], budgetVal, price, env.manifestJSON, logger)
	if err != nil {
		t.Fatalf("LoadAgent on node[%d]: %v", nodeIdx, err)
	}

	if err := inst.Init(env.ctx); err != nil {
		t.Fatalf("Init on node[%d]: %v", nodeIdx, err)
	}

	for i := 0; i < ticks; i++ {
		if _, err := inst.Tick(env.ctx); err != nil {
			t.Fatalf("Tick %d on node[%d]: %v", i+1, nodeIdx, err)
		}
	}

	if err := inst.SaveCheckpointToStorage(env.ctx); err != nil {
		t.Fatalf("SaveCheckpointToStorage on node[%d]: %v", nodeIdx, err)
	}

	env.migSvcs[nodeIdx].RegisterAgent(env.agentID, inst)
	return inst
}

// migrateAndVerify migrates the agent from src to dst, then asserts source
// cleanup and target reception.
func (env *multiNodeEnv) migrateAndVerify(t *testing.T, src, dst int) {
	t.Helper()

	if err := env.migSvcs[src].MigrateAgent(env.ctx, env.agentID, env.wasmPath, env.addr(dst)); err != nil {
		t.Fatalf("MigrateAgent %d→%d: %v", src, dst, err)
	}

	// Source must be cleaned up
	if agents := env.migSvcs[src].GetActiveAgents(); len(agents) != 0 {
		t.Errorf("node[%d] should have no active agents after migration, got %v", src, agents)
	}
	if _, err := env.storages[src].LoadCheckpoint(env.ctx, env.agentID); err != storage.ErrCheckpointNotFound {
		t.Errorf("node[%d] checkpoint should be deleted, got err=%v", src, err)
	}

	// Target must have the agent
	agents := env.migSvcs[dst].GetActiveAgents()
	if len(agents) != 1 || agents[0] != env.agentID {
		t.Fatalf("node[%d] should have agent %q, got %v", dst, env.agentID, agents)
	}
}

// readCheckpoint loads and parses the checkpoint from node i.
// Returns budget, tick number, counter (agent state), and WASM hash.
func (env *multiNodeEnv) readCheckpoint(t *testing.T, nodeIdx int) (budgetVal int64, tick uint64, counter uint64) {
	t.Helper()

	checkpoint, err := env.storages[nodeIdx].LoadCheckpoint(env.ctx, env.agentID)
	if err != nil {
		t.Fatalf("LoadCheckpoint on node[%d]: %v", nodeIdx, err)
	}

	b, _, tk, _, state, err := agent.ParseCheckpointHeader(checkpoint)
	if err != nil {
		t.Fatalf("ParseCheckpointHeader on node[%d]: %v", nodeIdx, err)
	}

	var cnt uint64
	if len(state) >= 8 {
		cnt = binary.LittleEndian.Uint64(state[:8])
	}

	return b, tk, cnt
}

// tickOnNode ticks the agent on node i for the given number of ticks and saves
// the checkpoint. Returns the instance's budget after ticks.
func (env *multiNodeEnv) tickOnNode(t *testing.T, nodeIdx int, ticks int) int64 {
	t.Helper()

	inst := env.migSvcs[nodeIdx].GetActiveInstance(env.agentID)
	if inst == nil {
		t.Fatalf("no active instance on node[%d]", nodeIdx)
	}

	for i := 0; i < ticks; i++ {
		if _, err := inst.Tick(env.ctx); err != nil {
			t.Fatalf("Tick on node[%d]: %v", nodeIdx, err)
		}
	}

	if err := inst.SaveCheckpointToStorage(env.ctx); err != nil {
		t.Fatalf("SaveCheckpointToStorage on node[%d]: %v", nodeIdx, err)
	}

	return inst.Budget
}

// budgetBeforeMigration returns the budget of the active instance on node i.
func (env *multiNodeEnv) budgetBeforeMigration(t *testing.T, nodeIdx int) int64 {
	t.Helper()
	inst := env.migSvcs[nodeIdx].GetActiveInstance(env.agentID)
	if inst == nil {
		t.Fatalf("no active instance on node[%d]", nodeIdx)
	}
	return inst.Budget
}

// TestChainMigration_ABC_A verifies an agent can migrate through a 3-node
// chain (A→B→C→A) with state and capability preservation at every hop.
func TestChainMigration_ABC_A(t *testing.T) {
	env := newMultiNodeEnv(t, 3)

	// Load agent on node 0, run 3 ticks
	env.loadAndInitAgent(t, 0, budget.FromFloat(10.0), budget.FromFloat(0.001), 3)

	// Hop 1: 0→1
	env.migrateAndVerify(t, 0, 1)

	b, tick, counter := env.readCheckpoint(t, 1)
	if counter != 3 {
		t.Errorf("hop 0→1: counter = %d, want 3", counter)
	}
	if tick != 3 {
		t.Errorf("hop 0→1: tick = %d, want 3", tick)
	}
	if b <= 0 {
		t.Errorf("hop 0→1: budget should be positive, got %d", b)
	}

	// Run 2 ticks on node 1
	env.tickOnNode(t, 1, 2)

	// Hop 2: 1→2
	env.migrateAndVerify(t, 1, 2)

	_, tick, counter = env.readCheckpoint(t, 2)
	if counter != 5 {
		t.Errorf("hop 1→2: counter = %d, want 5", counter)
	}
	if tick != 5 {
		t.Errorf("hop 1→2: tick = %d, want 5", tick)
	}

	// Run 2 ticks on node 2
	env.tickOnNode(t, 2, 2)

	// Hop 3: 2→0 (back to origin)
	env.migrateAndVerify(t, 2, 0)

	_, tick, counter = env.readCheckpoint(t, 0)
	if counter != 7 {
		t.Errorf("hop 2→0: counter = %d, want 7", counter)
	}
	if tick != 7 {
		t.Errorf("hop 2→0: tick = %d, want 7", tick)
	}

	// Verify agent continues on the original node
	env.tickOnNode(t, 0, 1)

	_, _, counter = env.readCheckpoint(t, 0)
	if counter != 8 {
		t.Errorf("after return: counter = %d, want 8", counter)
	}
}

// TestChainMigration_BudgetConservation verifies RE-3 (budget never
// created/destroyed) across multiple migration hops. The core invariant is
// that migrations never alter the budget — only ticks deduct cost. Tick cost
// may round to 0 for sub-microsecond executions, so we verify non-increase
// rather than strict decrease.
func TestChainMigration_BudgetConservation(t *testing.T) {
	env := newMultiNodeEnv(t, 3)

	initialBudget := budget.FromFloat(100.0)
	env.loadAndInitAgent(t, 0, initialBudget, budget.FromFloat(0.01), 5)

	// Budget after initial ticks must not exceed starting budget
	b0 := env.budgetBeforeMigration(t, 0)
	if b0 > initialBudget {
		t.Errorf("budget increased after ticks: initial=%d, after=%d", initialBudget, b0)
	}

	// Hop 0→1: migration must not alter budget
	env.migrateAndVerify(t, 0, 1)
	b1, _, _ := env.readCheckpoint(t, 1)
	if b1 != b0 {
		t.Errorf("budget changed during migration 0→1: pre=%d, post=%d", b0, b1)
	}

	// Run 5 ticks on node 1
	b2 := env.tickOnNode(t, 1, 5)
	if b2 > b1 {
		t.Errorf("budget increased after ticks on node 1: pre=%d, post=%d", b1, b2)
	}

	// Hop 1→2: migration must not alter budget
	preMigrate := env.budgetBeforeMigration(t, 1)
	env.migrateAndVerify(t, 1, 2)
	b3, _, _ := env.readCheckpoint(t, 2)
	if b3 != preMigrate {
		t.Errorf("budget changed during migration 1→2: pre=%d, post=%d", preMigrate, b3)
	}

	// Run 5 ticks on node 2
	b4 := env.tickOnNode(t, 2, 5)
	if b4 > b3 {
		t.Errorf("budget increased after ticks on node 2: pre=%d, post=%d", b3, b4)
	}

	// Hop 2→0: migration must not alter budget
	preMigrate = env.budgetBeforeMigration(t, 2)
	env.migrateAndVerify(t, 2, 0)
	b5, _, _ := env.readCheckpoint(t, 0)
	if b5 != preMigrate {
		t.Errorf("budget changed during migration 2→0: pre=%d, post=%d", preMigrate, b5)
	}

	// Overall: budget never exceeded initial, never went negative
	if b5 > initialBudget {
		t.Errorf("final budget exceeds initial: initial=%d, final=%d", initialBudget, b5)
	}
	if b5 <= 0 {
		t.Errorf("budget should still be positive, got %d", b5)
	}
}

// TestStressMigration_RapidRoundTrips stress tests resource cleanup across
// many rapid back-and-forth migrations between 2 nodes.
func TestStressMigration_RapidRoundTrips(t *testing.T) {
	env := newMultiNodeEnv(t, 2)

	const rounds = 20

	// Load agent on node 0, run 1 initial tick
	env.loadAndInitAgent(t, 0, budget.FromFloat(100.0), budget.FromFloat(0.001), 1)

	current := 0
	for i := 0; i < rounds; i++ {
		target := 1 - current

		env.migrateAndVerify(t, current, target)
		env.tickOnNode(t, target, 1)

		current = target
	}

	// Verify final counter: 1 initial tick + 20 round ticks = 21
	_, _, counter := env.readCheckpoint(t, current)
	if counter != uint64(1+rounds) {
		t.Errorf("final counter = %d, want %d", counter, 1+rounds)
	}
}

// TestCapabilityRejection_MigrationFails verifies CE-5: migration is rejected
// when the target node lacks required capabilities. The source retains the
// agent per EI-6 (safety over liveness).
func TestCapabilityRejection_MigrationFails(t *testing.T) {
	env := newMultiNodeEnv(t, 2)

	// Configure node 1 to only support "clock" (missing rand, log)
	env.migSvcs[1].SetNodeCapabilities([]string{"clock"})

	// Load agent with full manifest (clock+rand+log) on node 0
	env.loadAndInitAgent(t, 0, budget.FromFloat(10.0), budget.FromFloat(0.001), 3)

	// Attempt migration — should fail
	err := env.migSvcs[0].MigrateAgent(env.ctx, env.agentID, env.wasmPath, env.addr(1))
	if err == nil {
		t.Fatal("migration should have failed due to capability mismatch")
	}

	// Source must retain the agent (EI-6)
	agents := env.migSvcs[0].GetActiveAgents()
	if len(agents) != 1 || agents[0] != env.agentID {
		t.Errorf("source should still have agent, got %v", agents)
	}

	// Source checkpoint must still exist
	if _, err := env.storages[0].LoadCheckpoint(env.ctx, env.agentID); err != nil {
		t.Errorf("source checkpoint should exist, got err=%v", err)
	}

	// Target must have nothing
	if agents := env.migSvcs[1].GetActiveAgents(); len(agents) != 0 {
		t.Errorf("target should have no agents, got %v", agents)
	}
	if _, err := env.storages[1].LoadCheckpoint(env.ctx, env.agentID); err != storage.ErrCheckpointNotFound {
		t.Errorf("target should have no checkpoint, got err=%v", err)
	}

	// Agent should still be able to tick on source
	env.tickOnNode(t, 0, 1)
	_, _, counter := env.readCheckpoint(t, 0)
	if counter != 4 {
		t.Errorf("counter after continued tick = %d, want 4", counter)
	}
}

// TestCapabilityPreservation_AcrossHops verifies the capability manifest is
// faithfully transmitted across multiple migration hops and that hostcalls
// remain functional at each destination.
func TestCapabilityPreservation_AcrossHops(t *testing.T) {
	env := newMultiNodeEnv(t, 3)

	env.loadAndInitAgent(t, 0, budget.FromFloat(10.0), budget.FromFloat(0.001), 1)

	hops := [][2]int{{0, 1}, {1, 2}, {2, 0}}
	for _, hop := range hops {
		src, dst := hop[0], hop[1]
		env.migrateAndVerify(t, src, dst)

		// Verify manifest is preserved on the target
		inst := env.migSvcs[dst].GetActiveInstance(env.agentID)
		if inst == nil {
			t.Fatalf("no instance on node[%d] after hop %d→%d", dst, src, dst)
		}
		if inst.Manifest == nil {
			t.Fatalf("manifest nil on node[%d] after hop %d→%d", dst, src, dst)
		}
		for _, cap := range []string{"clock", "rand", "log"} {
			if !inst.Manifest.Has(cap) {
				t.Errorf("node[%d] missing capability %q after hop %d→%d", dst, cap, src, dst)
			}
		}

		// Verify agent can still tick (hostcalls are registered correctly)
		env.tickOnNode(t, dst, 1)
	}
}
