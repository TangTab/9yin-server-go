package session

import (
	"errors"
	"fmt"
	"sync/atomic"
)

// Phase 是服务端对单个客户端连接的视角。它刻意建模
// 协议就绪状态而非 UI 表单名称，使 world 服务可以依赖它
// 而不必导入客户端表现层的细节。
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

// TransitionError 表示一个已知的握手操作码到达了某个阶段，若在该阶段
// 接受它会破坏连接状态。未知的玩法操作码
// 有意留给所属子系统处理。
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

// Validate 只校验那些顺序已由现代客户端 Lua 流程和可用的回环服务器
// 证实的握手操作码。业务操作码不会仅因
// 尚未分类而在此被拒绝。
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
		// 当前客户端在进入场景后可能重复发送其单字节 ready 通知。
		// 所有者可将其视为幂等操作；
		// 若在此拒绝，会在业务分发器看到它之前
		// 关闭一个健康的游戏连接。
		allowed = m.phase == PhaseEnteringWorld || m.phase == PhaseInWorld
	case ClientActivity:
		// 抓包的现代客户端会话会在 ChooseRole 之后、单字节 ClientReady 之前
		// 发送若干 0x0A 自定义消息。它们是有效的
		// ready 前流量，但不得打开场景对象屏障。
		allowed = m.phase == PhaseEnteringWorld || m.phase == PhaseAwaitingActivity || m.phase == PhaseInWorld
	default:
		// 运行时抓包中还包含 RoleSelection 和 EnteringWorld 阶段的
		// 操作码 0x18。在其布局被分类之前，保留可用
		// 客户端的流量并留给业务分发器处理。
		return nil
	}
	if !allowed {
		return &TransitionError{Phase: m.phase, Opcode: opcode}
	}
	return nil
}

// ValidateMessage 应用由当前发送方实现直接证实的固定或最小长度。
// 它有意不为 0x03、0x0A 或未分类的玩法消息
// 臆造布局。
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

// ReenterScene 在不重建网络会话或角色的前提下开启新的场景世代。
// 下一对 ClientReady/ClientActivity 使用与首次进入场景
// 相同的保守对象实体化屏障。
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
