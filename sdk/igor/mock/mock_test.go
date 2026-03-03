package mock_test

import (
	"encoding/binary"
	"testing"

	igor "github.com/simonovic86/igor/sdk/igor"
	"github.com/simonovic86/igor/sdk/igor/mock"
)

func TestClockNow_Deterministic(t *testing.T) {
	rt := mock.NewDeterministic(1_000_000_000, 42)
	defer rt.Enable()()

	v1 := igor.ClockNow()
	v2 := igor.ClockNow()

	if v1 != 1_000_000_000 {
		t.Errorf("first call: got %d, want 1000000000", v1)
	}
	if v2 != 2_000_000_000 {
		t.Errorf("second call: got %d, want 2000000000", v2)
	}
}

func TestClockNow_CustomClock(t *testing.T) {
	rt := mock.New()
	defer rt.Enable()()

	rt.SetClock(func() int64 { return 42 })

	if v := igor.ClockNow(); v != 42 {
		t.Errorf("got %d, want 42", v)
	}
}

func TestClockNow_FixedClock(t *testing.T) {
	rt := mock.New()
	defer rt.Enable()()

	rt.SetFixedClock(100, 10)

	v1 := igor.ClockNow()
	v2 := igor.ClockNow()
	v3 := igor.ClockNow()

	if v1 != 100 {
		t.Errorf("v1: got %d, want 100", v1)
	}
	if v2 != 110 {
		t.Errorf("v2: got %d, want 110", v2)
	}
	if v3 != 120 {
		t.Errorf("v3: got %d, want 120", v3)
	}
}

func TestRandBytes_Deterministic(t *testing.T) {
	rt := mock.NewDeterministic(0, 42)
	defer rt.Enable()()

	buf1 := make([]byte, 16)
	if err := igor.RandBytes(buf1); err != nil {
		t.Fatal(err)
	}

	// Same seed should produce same output.
	rt2 := mock.NewDeterministic(0, 42)
	defer rt2.Enable()()

	buf2 := make([]byte, 16)
	if err := igor.RandBytes(buf2); err != nil {
		t.Fatal(err)
	}

	for i := range buf1 {
		if buf1[i] != buf2[i] {
			t.Fatalf("byte %d differs: %d vs %d", i, buf1[i], buf2[i])
		}
	}
}

func TestRandBytes_Empty(t *testing.T) {
	rt := mock.New()
	defer rt.Enable()()

	// Empty buffer should be a no-op.
	if err := igor.RandBytes(nil); err != nil {
		t.Fatal(err)
	}
	if err := igor.RandBytes([]byte{}); err != nil {
		t.Fatal(err)
	}
}

func TestRandBytes_NonZero(t *testing.T) {
	rt := mock.NewDeterministic(0, 1)
	defer rt.Enable()()

	buf := make([]byte, 32)
	if err := igor.RandBytes(buf); err != nil {
		t.Fatal(err)
	}

	allZero := true
	for _, b := range buf {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("rand bytes are all zero")
	}
}

func TestLog_Capture(t *testing.T) {
	rt := mock.New()
	defer rt.Enable()()

	igor.Log("hello")
	igor.Logf("count %d", 42)

	logs := rt.Logs()
	if len(logs) != 2 {
		t.Fatalf("got %d logs, want 2", len(logs))
	}
	if logs[0] != "hello" {
		t.Errorf("log[0]: got %q, want %q", logs[0], "hello")
	}
	if logs[1] != "count 42" {
		t.Errorf("log[1]: got %q, want %q", logs[1], "count 42")
	}
}

func TestLog_Empty(t *testing.T) {
	rt := mock.New()
	defer rt.Enable()()

	igor.Log("") // should be a no-op
	if len(rt.Logs()) != 0 {
		t.Error("empty log should not be captured")
	}
}

func TestClearLogs(t *testing.T) {
	rt := mock.New()
	defer rt.Enable()()

	igor.Log("a")
	igor.Log("b")
	rt.ClearLogs()

	if len(rt.Logs()) != 0 {
		t.Errorf("got %d logs after clear, want 0", len(rt.Logs()))
	}

	igor.Log("c")
	if len(rt.Logs()) != 1 {
		t.Errorf("got %d logs, want 1", len(rt.Logs()))
	}
}

func TestDisable_RestoresPanic(t *testing.T) {
	rt := mock.New()
	cleanup := rt.Enable()
	cleanup()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic after Disable")
		}
	}()
	igor.ClockNow()
}

func TestPanic_WithoutMock(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic without mock")
		}
	}()
	igor.ClockNow()
}

// TestAgentLifecycle demonstrates the primary use case: testing an agent natively.
func TestAgentLifecycle(t *testing.T) {
	rt := mock.NewDeterministic(1_000_000_000, 42)
	defer rt.Enable()()

	agent := &testAgent{}
	agent.Init()
	agent.Tick()
	agent.Tick()
	agent.Tick()

	if agent.TickCount != 3 {
		t.Errorf("tick count: got %d, want 3", agent.TickCount)
	}

	// Verify checkpoint round-trip.
	state := agent.Marshal()
	agent2 := &testAgent{}
	agent2.Unmarshal(state)

	if agent2.TickCount != 3 {
		t.Errorf("restored tick count: got %d, want 3", agent2.TickCount)
	}
	if agent2.LastClock != agent.LastClock {
		t.Errorf("restored clock: got %d, want %d", agent2.LastClock, agent.LastClock)
	}
}

// testAgent is a minimal agent for testing.
type testAgent struct {
	TickCount uint64
	LastClock int64
	Luck      uint32
}

func (a *testAgent) Init() {}

func (a *testAgent) Tick() {
	a.TickCount++
	a.LastClock = igor.ClockNow()
	buf := make([]byte, 4)
	_ = igor.RandBytes(buf)
	a.Luck ^= binary.LittleEndian.Uint32(buf)
	igor.Logf("tick %d", a.TickCount)
}

func (a *testAgent) Marshal() []byte {
	buf := make([]byte, 20)
	binary.LittleEndian.PutUint64(buf[0:8], a.TickCount)
	binary.LittleEndian.PutUint64(buf[8:16], uint64(a.LastClock))
	binary.LittleEndian.PutUint32(buf[16:20], a.Luck)
	return buf
}

func (a *testAgent) Unmarshal(data []byte) {
	a.TickCount = binary.LittleEndian.Uint64(data[0:8])
	a.LastClock = int64(binary.LittleEndian.Uint64(data[8:16]))
	a.Luck = binary.LittleEndian.Uint32(data[16:20])
}
