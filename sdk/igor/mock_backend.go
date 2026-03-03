package igor

// MockBackend is the interface for mock hostcall implementations.
// Used by sdk/igor/mock to enable native (non-WASM) testing of agent logic.
type MockBackend interface {
	ClockNow() int64
	RandBytes(buf []byte) error
	LogEmit(msg string)
}

// activeMock is set by mock.Enable() and cleared by mock.Disable().
// Only referenced from hostcalls_wrappers_stub.go (non-WASM builds).
var activeMock MockBackend

// SetMockBackend installs or removes a mock hostcall backend.
// Pass nil to remove. Only effective in non-WASM builds.
func SetMockBackend(m MockBackend) {
	activeMock = m
}
