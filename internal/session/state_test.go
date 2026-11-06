package session

import (
	"errors"
	"testing"
)

func TestExistingRoleFlow(t *testing.T) {
	m := New()
	if err := m.Validate(ClientLogin); err != nil {
		t.Fatal(err)
	}
	if err := m.LoginCompleted(true); err != nil {
		t.Fatal(err)
	}
	if got := m.Phase(); got != PhaseRoleSelection {
		t.Fatalf("phase = %s, want role_selection", got)
	}
	if err := m.Validate(ClientChooseRole); err != nil {
		t.Fatal(err)
	}
	if err := m.RoleChosen(); err != nil {
		t.Fatal(err)
	}
	if err := m.Ready(); err != nil {
		t.Fatal(err)
	}
	if err := m.Activity(); err != nil {
		t.Fatal(err)
	}
	if got := m.Phase(); got != PhaseInWorld {
		t.Fatalf("phase = %s, want in_world", got)
	}
}

func TestCreateRoleFlow(t *testing.T) {
	m := New()
	if err := m.LoginCompleted(false); err != nil {
		t.Fatal(err)
	}
	if got := m.Phase(); got != PhaseRoleCreation {
		t.Fatalf("phase = %s, want role_creation", got)
	}
	if err := m.Validate(ClientCreateRole); err != nil {
		t.Fatal(err)
	}
	if err := m.RoleCreated(); err != nil {
		t.Fatal(err)
	}
	if got := m.Phase(); got != PhaseRoleSelection {
		t.Fatalf("phase = %s, want role_selection", got)
	}
}

func TestRejectsKnownOpcodeOutOfPhase(t *testing.T) {
	m := New()
	err := m.Validate(ClientChooseRole)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("error = %v, want ErrInvalidTransition", err)
	}
	var transition *TransitionError
	if !errors.As(err, &transition) || transition.Phase != PhaseConnected || transition.Opcode != ClientChooseRole {
		t.Fatalf("transition = %#v", transition)
	}
}

func TestUnknownOpcodeIsOwnedByBusinessDispatcher(t *testing.T) {
	if err := New().Validate(0x7F); err != nil {
		t.Fatalf("unknown opcode rejected: %v", err)
	}
}

func TestPreReadyActivityDoesNotOpenBarrier(t *testing.T) {
	m := New()
	_ = m.LoginCompleted(true)
	_ = m.RoleChosen()
	if err := m.Activity(); err != nil {
		t.Fatal(err)
	}
	if got := m.Phase(); got != PhaseEnteringWorld {
		t.Fatalf("phase = %s, want entering_world", got)
	}
}

func TestReadyCannotBeRepeatedWithoutNewSceneGeneration(t *testing.T) {
	m := New()
	_ = m.LoginCompleted(true)
	_ = m.RoleChosen()
	_ = m.Ready()
	if err := m.Ready(); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("repeated ready error = %v", err)
	}
}

func TestReadyIsPermittedAsAnInWorldIdempotentNotification(t *testing.T) {
	m := New()
	_ = m.LoginCompleted(true)
	_ = m.RoleChosen()
	_ = m.Ready()
	_ = m.Activity()
	if err := m.Validate(ClientReady); err != nil {
		t.Fatalf("in-world duplicate ready rejected: %v", err)
	}
}

func TestReenterSceneRestoresReadyBarrier(t *testing.T) {
	m := New()
	_ = m.LoginCompleted(true)
	_ = m.RoleChosen()
	_ = m.Ready()
	_ = m.Activity()
	if err := m.ReenterScene(); err != nil {
		t.Fatal(err)
	}
	if got := m.Phase(); got != PhaseEnteringWorld {
		t.Fatalf("phase = %s, want entering_world", got)
	}
	if err := m.Ready(); err != nil {
		t.Fatalf("ready after re-entry: %v", err)
	}
}

func TestObservedUnknownOpcodeBeforeWorldIsPreserved(t *testing.T) {
	m := New()
	_ = m.LoginCompleted(true)
	if err := m.Validate(0x18); err != nil {
		t.Fatalf("observed role-stage opcode rejected: %v", err)
	}
	_ = m.RoleChosen()
	if err := m.Validate(0x18); err != nil {
		t.Fatalf("observed entry-stage opcode rejected: %v", err)
	}
}

func TestClosedMachineRejectsAllInput(t *testing.T) {
	m := New()
	m.Close()
	if err := m.Validate(0x7F); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("closed validation error = %v", err)
	}
}

func TestMachinesHaveDistinctConnectionIDs(t *testing.T) {
	first, second := New(), New()
	if first.ID() == 0 || second.ID() == 0 || first.ID() == second.ID() {
		t.Fatalf("machine IDs = %d, %d", first.ID(), second.ID())
	}
}

func TestKnownMessageLengthGates(t *testing.T) {
	tests := []struct {
		name    string
		phase   func(*Machine)
		message []byte
	}{
		{"short login", func(*Machine) {}, make([]byte, 4)},
		{"short choose role", func(m *Machine) { _ = m.LoginCompleted(true) }, make([]byte, 0x53)},
		{"short create role", func(m *Machine) { _ = m.LoginCompleted(false) }, make([]byte, 0x48)},
		{"long client ready", func(m *Machine) { _ = m.LoginCompleted(true); _ = m.RoleChosen() }, []byte{ClientReady, 0}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := New()
			test.phase(m)
			test.message[0] = map[string]byte{
				"short login": ClientLogin, "short choose role": ClientChooseRole,
				"short create role": ClientCreateRole, "long client ready": ClientReady,
			}[test.name]
			if err := m.ValidateMessage(test.message); !errors.Is(err, ErrInvalidMessage) {
				t.Fatalf("error = %v, want ErrInvalidMessage", err)
			}
		})
	}
}
