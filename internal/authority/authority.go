package authority

import "fmt"

// Epoch identifies a unique authority period. Totally ordered:
// higher MajorVersion always supersedes; within the same MajorVersion,
// higher LeaseGeneration supersedes.
type Epoch struct {
	MajorVersion    uint64 // Increments on authority transfer (new node)
	LeaseGeneration uint64 // Increments on lease renewal (same node)
}

// Supersedes returns true if e is strictly greater than other.
func (e Epoch) Supersedes(other Epoch) bool {
	if e.MajorVersion != other.MajorVersion {
		return e.MajorVersion > other.MajorVersion
	}
	return e.LeaseGeneration > other.LeaseGeneration
}

// Equal returns true if both components match.
func (e Epoch) Equal(other Epoch) bool {
	return e.MajorVersion == other.MajorVersion &&
		e.LeaseGeneration == other.LeaseGeneration
}

// String returns a human-readable representation of the epoch.
func (e Epoch) String() string {
	return fmt.Sprintf("(%d,%d)", e.MajorVersion, e.LeaseGeneration)
}

// State represents the authority state per AUTHORITY_STATE_MACHINE.md.
type State int

const (
	StateActiveOwner      State = iota // Tick execution allowed
	StateHandoffInitiated              // Migration initiated, no new ticks
	StateHandoffPending                // Authority in transit
	StateRetired                       // Source completed transfer
	StateRecoveryRequired              // Authority ambiguous, all ticks halt
)

// String returns a human-readable representation of the state.
func (s State) String() string {
	switch s {
	case StateActiveOwner:
		return "ACTIVE_OWNER"
	case StateHandoffInitiated:
		return "HANDOFF_INITIATED"
	case StateHandoffPending:
		return "HANDOFF_PENDING"
	case StateRetired:
		return "RETIRED"
	case StateRecoveryRequired:
		return "RECOVERY_REQUIRED"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(s))
	}
}

// CanTick returns true if tick execution is permitted in this state.
func (s State) CanTick() bool {
	return s == StateActiveOwner
}
