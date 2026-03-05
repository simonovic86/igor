package authority

import "testing"

func TestEpoch_Supersedes(t *testing.T) {
	tests := []struct {
		name string
		a, b Epoch
		want bool
	}{
		{"higher major wins", Epoch{2, 0}, Epoch{1, 5}, true},
		{"lower major loses", Epoch{1, 5}, Epoch{2, 0}, false},
		{"same major higher gen wins", Epoch{1, 5}, Epoch{1, 3}, true},
		{"same major lower gen loses", Epoch{1, 3}, Epoch{1, 5}, false},
		{"equal epochs no supersede", Epoch{1, 3}, Epoch{1, 3}, false},
		{"zero epoch no supersede self", Epoch{0, 0}, Epoch{0, 0}, false},
		{"first gen supersedes zero", Epoch{0, 1}, Epoch{0, 0}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Supersedes(tt.b); got != tt.want {
				t.Errorf("Epoch%s.Supersedes(Epoch%s) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestEpoch_Equal(t *testing.T) {
	tests := []struct {
		name string
		a, b Epoch
		want bool
	}{
		{"equal", Epoch{1, 3}, Epoch{1, 3}, true},
		{"diff major", Epoch{1, 3}, Epoch{2, 3}, false},
		{"diff gen", Epoch{1, 3}, Epoch{1, 4}, false},
		{"zero equal", Epoch{0, 0}, Epoch{0, 0}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equal(tt.b); got != tt.want {
				t.Errorf("Epoch%s.Equal(Epoch%s) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestEpoch_String(t *testing.T) {
	e := Epoch{3, 7}
	if s := e.String(); s != "(3,7)" {
		t.Errorf("Epoch.String() = %q, want %q", s, "(3,7)")
	}
}

func TestState_CanTick(t *testing.T) {
	tests := []struct {
		state State
		want  bool
	}{
		{StateActiveOwner, true},
		{StateHandoffInitiated, false},
		{StateHandoffPending, false},
		{StateRetired, false},
		{StateRecoveryRequired, false},
	}
	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			if got := tt.state.CanTick(); got != tt.want {
				t.Errorf("State(%s).CanTick() = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestState_String(t *testing.T) {
	if s := StateActiveOwner.String(); s != "ACTIVE_OWNER" {
		t.Errorf("StateActiveOwner.String() = %q", s)
	}
	if s := StateRecoveryRequired.String(); s != "RECOVERY_REQUIRED" {
		t.Errorf("StateRecoveryRequired.String() = %q", s)
	}
	if s := State(99).String(); s != "UNKNOWN(99)" {
		t.Errorf("State(99).String() = %q", s)
	}
}
