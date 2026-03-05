// Package mock provides mock hostcall implementations for testing Igor agents
// natively (without WASM compilation).
//
// Usage:
//
//	func TestMyAgent(t *testing.T) {
//	    rt := mock.NewDeterministic(1_000_000_000, 42)
//	    defer rt.Enable()()
//
//	    agent := &MyAgent{}
//	    agent.Init()
//	    agent.Tick()
//
//	    if rt.Logs()[0] != "expected message" {
//	        t.Error("unexpected log")
//	    }
//	}
package mock

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	igor "github.com/simonovic86/igor/sdk/igor"
)

// Runtime provides mock implementations of Igor hostcalls for native testing.
type Runtime struct {
	mu        sync.Mutex
	clock     func() int64
	randSrc   *rand.Rand
	logs      []string
	budget    int64
	receipts  [][]byte
	nodePrice int64
}

// New creates a mock runtime using the real system clock and crypto-seeded rand.
func New() *Runtime {
	return &Runtime{
		clock:   func() int64 { return time.Now().UnixNano() },
		randSrc: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
	}
}

// NewDeterministic creates a mock runtime with a fixed clock that advances
// by 1 second per call and a seeded PRNG for fully reproducible tests.
func NewDeterministic(clockStartNano int64, randSeed uint64) *Runtime {
	callCount := int64(0)
	return &Runtime{
		clock: func() int64 {
			v := clockStartNano + callCount*1_000_000_000
			callCount++
			return v
		},
		randSrc: rand.New(rand.NewPCG(randSeed, randSeed)),
	}
}

// Enable installs this runtime as the active hostcall backend.
// Returns a cleanup function that calls Disable.
//
//	defer rt.Enable()()
func (r *Runtime) Enable() func() {
	igor.SetMockBackend(r)
	return func() { r.Disable() }
}

// Disable uninstalls this runtime, restoring panic stubs.
func (r *Runtime) Disable() {
	igor.SetMockBackend(nil)
}

// SetClock configures a custom clock function.
func (r *Runtime) SetClock(fn func() int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clock = fn
}

// SetFixedClock sets the clock to return a fixed value that advances by
// delta nanoseconds on each call.
func (r *Runtime) SetFixedClock(startNano, deltaNano int64) {
	callCount := int64(0)
	r.SetClock(func() int64 {
		v := startNano + callCount*deltaNano
		callCount++
		return v
	})
}

// Logs returns all log messages captured since creation or last ClearLogs.
func (r *Runtime) Logs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.logs))
	copy(out, r.logs)
	return out
}

// ClearLogs resets the captured log buffer.
func (r *Runtime) ClearLogs() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = nil
}

// ClockNow implements MockBackend.
func (r *Runtime) ClockNow() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.clock()
}

// RandBytes implements MockBackend.
func (r *Runtime) RandBytes(buf []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range buf {
		buf[i] = byte(r.randSrc.Uint32())
	}
	return nil
}

// LogEmit implements MockBackend.
func (r *Runtime) LogEmit(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, msg)
}

// WalletBalance implements MockBackend.
func (r *Runtime) WalletBalance() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.budget
}

// WalletReceiptCount implements MockBackend.
func (r *Runtime) WalletReceiptCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.receipts)
}

// WalletReceipt implements MockBackend.
func (r *Runtime) WalletReceipt(index int) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if index < 0 || index >= len(r.receipts) {
		return nil, fmt.Errorf("receipt index %d out of range", index)
	}
	return r.receipts[index], nil
}

// SetBudget sets the mock budget value returned by WalletBalance.
func (r *Runtime) SetBudget(b int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.budget = b
}

// AddReceipt adds a serialized receipt for WalletReceipt to return.
func (r *Runtime) AddReceipt(data []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.receipts = append(r.receipts, data)
}

// NodePrice implements MockBackend.
func (r *Runtime) NodePrice() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.nodePrice
}

// SetNodePrice sets the mock node price returned by NodePrice.
func (r *Runtime) SetNodePrice(p int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodePrice = p
}
