// SPDX-License-Identifier: Apache-2.0

package igor

// Agent is the interface that agent authors implement.
// The SDK handles all WASM lifecycle exports and memory management;
// agents only need to provide application logic and serialization.
type Agent interface {
	// Init initializes the agent's state. Called once on first start.
	// Do not call observation hostcalls (ClockNow, RandBytes, Log) here —
	// only agent_tick should produce observations for replay correctness (CM-4).
	Init()

	// Tick executes one step of the agent's logic.
	// Must complete within 100ms.
	// Return true if there is more work pending (tick again soon, subject to
	// 10ms minimum interval). Return false to sleep until the next normal
	// interval (~1 Hz).
	Tick() bool

	// Marshal serializes the agent's state for checkpointing.
	// The returned bytes must fully capture the agent's state so that
	// Unmarshal can restore it exactly.
	Marshal() []byte

	// Unmarshal restores the agent's state from a previous Marshal output.
	// Called during resume (restart or migration). Must be side-effect-free
	// since it is also called during replay verification (CM-4).
	Unmarshal(data []byte)
}

// Global agent instance and checkpoint buffer, set by Run.
var (
	registeredAgent Agent
	ckptBuf         []byte
)

// Run registers an Agent implementation with the SDK.
// Must be called from the agent's init() function.
//
//	func init() { igor.Run(&MyAgent{}) }
//	func main() {}
func Run(a Agent) {
	registeredAgent = a
}
