package fssologlobal

// State 是官方状态机枚举（sologlobalstate.cpp，this+0x30）：
// 0=Unknown / 1=Opened / 2=Closed。
type State int

// 官方三态（GetState@0x100307b0 返回的名字串同此）。
const (
	StateUnknown State = iota
	StateOpened
	StateClosed
)

// String 官方 GetState 的呈现名。
func (s State) String() string {
	switch s {
	case StateOpened:
		return "Opened"
	case StateClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

// 官方事件值（GlobalStateMachine 第一参）：
// 1 = 启动（SoloGlobal::Start 尾 GlobalStateMachine(1,0,0)）；
// 0 = 定时 tick（TimerCheck 每秒 GlobalStateMachine(0,0,0)）。
const (
	eventTick  = 0
	eventStart = 1
)

// StateMachine 复刻官方 SoloGlobal::GlobalStateMachine@0x10030880 +
// 三态处理器（Unknown@0x10030980 / Opened@0x10030ae0 /
// Closed@0x10030b60）。迁移仅一条：Unknown --(1)--> Opened；
// 三态对事件 0 均"接受但不动"；其余事件 trace error input 后拒绝。
// Closed 无官方迁入路径（仅枚举/呈现，同构保留）。
type StateMachine struct {
	state   State
	onTrace func(format string, args ...any)
}

// NewStateMachine 建态机（初始 Unknown）。
func NewStateMachine(onTrace func(string, ...any)) *StateMachine {
	if onTrace == nil {
		onTrace = func(string, ...any) {}
	}
	return &StateMachine{state: StateUnknown, onTrace: onTrace}
}

// State 当前态。
func (m *StateMachine) State() State { return m.state }

// Dispatch 官方 GlobalStateMachine(event, arg1, arg2)。返回值 =
// 状态处理器返回值（接受 true / 拒绝 false）。
func (m *StateMachine) Dispatch(event, arg1, arg2 int) bool {
	switch m.state {
	case StateUnknown: // GlobalStateUnknown
		if event == eventStart {
			m.state = StateOpened
			return true
		}
		if event == eventTick {
			return true
		}
		m.traceError("Unknown", event)
		return false
	case StateOpened: // GlobalStateOpened
		if event == eventTick {
			return true
		}
		m.traceError("Opened", event)
		return false
	case StateClosed: // GlobalStateClosed
		if event == eventTick {
			return true
		}
		m.traceError("Closed", event)
		return false
	}
	return false
}

// traceError 官方 "(SoloGlobal::GlobalState%s)error input %d"。
func (m *StateMachine) traceError(stateName string, event int) {
	m.onTrace("(SoloGlobal::GlobalState%s)error input %d", stateName, event)
}
