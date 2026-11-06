package session

import (
	"errors"
	"fmt"
	"sync/atomic"
)

// Phase is the server-side view of one client connection. It deliberately
// models protocol readiness, not UI form names, so world services can depend on
// it without importing client presentation details.
type Phase uint8

const (
	PhaseConnected Phase = iota
	PhaseRoleCreation
	PhaseRoleSelection
	PhaseEnteringWorld
	PhaseAwaitingActivity
	PhaseInWorld
	PhaseClosed
)

func (p Phase) String() string {
	switch p {
	case PhaseConnected:
		return "connected"
	case PhaseRoleCreation:
		return "role_creation"
	case PhaseRoleSelection:
		return "role_selection"
	case PhaseEnteringWorld:
		return "entering_world"
	case PhaseAwaitingActivity:
		return "awaiting_activity"
	case PhaseInWorld:
		return "in_world"
	case PhaseClosed:
		return "closed"
	default:
		return fmt.Sprintf("phase(%d)", uint8(p))
	}
}

const (
	ClientLogin      byte = 0x02
	ClientWorldInfo  byte = 0x03
	ClientChooseRole byte = 0x04
	ClientCreateRole byte = 0x05
	ClientReady      byte = 0x09
	ClientActivity   byte = 0x0A
)

var ErrInvalidTransition = errors.New("invalid session transition")
var ErrInvalidMessage = errors.New("invalid session message")

// TransitionError means that a known handshake opcode arrived in a phase in
// which accepting it would corrupt the connection state. Unknown gameplay
// opcodes are intentionally left to the owning subsystem.
type TransitionError struct {
	Phase  Phase
	Opcode byte
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("%v: client opcode 0x%02X in %s", ErrInvalidTransition, e.Opcode, e.Phase)
}

func (e *TransitionError) Unwrap() error { return ErrInvalidTransition }

type MessageError struct {
	Opcode byte
	Length int
	Rule   string
}

func (e *MessageError) Error() string {
	return fmt.Sprintf("%v: client opcode 0x%02X length %d (%s)", ErrInvalidMessage, e.Opcode, e.Length, e.Rule)
}

func (e *MessageError) Unwrap() error { return ErrInvalidMessage }

type Machine struct {
	id    uint64
	phase Phase
}

var nextID atomic.Uint64

func New() *Machine { return &Machine{id: nextID.Add(1), phase: PhaseConnected} }

func (m *Machine) ID() uint64 { return m.id }

func (m *Machine) Phase() Phase { return m.phase }

// Validate checks only the handshake opcodes whose ordering is already backed
// by the modern Lua flow and the working loopback server. Business opcodes are
// not rejected here merely because they have not been classified yet.
func (m *Machine) Validate(opcode byte) error {
	if m.phase == PhaseClosed {
		return &TransitionError{Phase: m.phase, Opcode: opcode}
	}

	allowed := false
	switch opcode {
	case ClientLogin:
		allowed = m.phase == PhaseConnected
	case ClientWorldInfo:
		allowed = m.phase == PhaseRoleCreation || m.phase == PhaseRoleSelection
	case ClientCreateRole:
		allowed = m.phase == PhaseRoleCreation
	case ClientChooseRole:
		allowed = m.phase == PhaseRoleSelection
	case ClientReady:
		// The current client may repeat its one-byte ready notification after
		// it is already in the scene. The owner can treat that as idempotent;
		// rejecting it here would close a healthy game connection before the
		// business dispatcher sees it.
		allowed = m.phase == PhaseEnteringWorld || m.phase == PhaseInWorld
	case ClientActivity:
		// Captured modern-client sessions send several 0x0A custom messages
		// after ChooseRole and before the one-byte ClientReady. They are valid
		// pre-ready traffic but must not open the scene-object barrier.
		allowed = m.phase == PhaseEnteringWorld || m.phase == PhaseAwaitingActivity || m.phase == PhaseInWorld
	default:
		// Runtime captures also contain opcode 0x18 in RoleSelection and
		// EnteringWorld. Until its layout is classified, preserve the working
		// client's traffic and leave it to the business dispatcher.
		return nil
	}
	if !allowed {
		return &TransitionError{Phase: m.phase, Opcode: opcode}
	}
	return nil
}

// ValidateMessage applies the fixed or minimum lengths directly proven by the
// current sender implementations. It intentionally does not invent layouts
// for 0x03, 0x0A, or unclassified gameplay messages.
func (m *Machine) ValidateMessage(message []byte) error {
	if len(message) == 0 {
		return &MessageError{Length: 0, Rule: "message must contain an opcode"}
	}
	opcode := message[0]
	if err := m.Validate(opcode); err != nil {
		return err
	}
	minimum := 1
	exact := 0
	switch opcode {
	case ClientLogin:
		minimum = 5
	case ClientChooseRole:
		minimum = 0x54
	case ClientCreateRole:
		minimum = 0x49
	case ClientReady:
		exact = 1
	}
	if exact != 0 && len(message) != exact {
		return &MessageError{Opcode: opcode, Length: len(message), Rule: fmt.Sprintf("want exactly %d", exact)}
	}
	if len(message) < minimum {
		return &MessageError{Opcode: opcode, Length: len(message), Rule: fmt.Sprintf("want at least %d", minimum)}
	}
	return nil
}

func (m *Machine) LoginCompleted(hasRole bool) error {
	if err := m.require(PhaseConnected, ClientLogin); err != nil {
		return err
	}
	if hasRole {
		m.phase = PhaseRoleSelection
	} else {
		m.phase = PhaseRoleCreation
	}
	return nil
}

func (m *Machine) RoleCreated() error {
	if m.phase != PhaseRoleCreation && m.phase != PhaseRoleSelection {
		return &TransitionError{Phase: m.phase, Opcode: ClientCreateRole}
	}
	m.phase = PhaseRoleSelection
	return nil
}

func (m *Machine) RoleChosen() error {
	if err := m.require(PhaseRoleSelection, ClientChooseRole); err != nil {
		return err
	}
	m.phase = PhaseEnteringWorld
	return nil
}

func (m *Machine) Ready() error {
	if err := m.Validate(ClientReady); err != nil {
		return err
	}
	m.phase = PhaseAwaitingActivity
	return nil
}

func (m *Machine) Activity() error {
	if err := m.Validate(ClientActivity); err != nil {
		return err
	}
	if m.phase == PhaseAwaitingActivity {
		m.phase = PhaseInWorld
	}
	return nil
}

// ReenterScene starts a new scene generation without recreating the network
// session or role. The next ClientReady/ClientActivity pair uses the same
// conservative object-materialization barrier as the first scene entry.
func (m *Machine) ReenterScene() error {
	if err := m.require(PhaseInWorld, 0x0C); err != nil {
		return err
	}
	m.phase = PhaseEnteringWorld
	return nil
}

func (m *Machine) Close() { m.phase = PhaseClosed }

func (m *Machine) require(phase Phase, opcode byte) error {
	if m.phase != phase {
		return &TransitionError{Phase: m.phase, Opcode: opcode}
	}
	return nil
}
