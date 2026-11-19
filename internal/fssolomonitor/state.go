package fssolomonitor

// State 是官方状态机枚举（solomonitorstate.cpp，实体 +0x30）：
// 0=Unknown / 1=Opened / 2=Closed。
type State int

// 官方三态（GetState 返回的名字串同此）。
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

// 官方事件值（MonitorStateMachine 第一参）：
// 1 = 启动（SoloMonitor::Start 尾 MonitorStateMachine(1,NULL,0)）；
// 0 = 定时 tick（TimerCheck 每秒 MonitorStateMachine(0,0,0)）。
const (
	eventTick  = 0
	eventStart = 1
)

// StateMachine 复刻官方 SoloMonitor::MonitorStateMachine + 三态处理器
// （solomonitorstate.cpp）。迁移仅一条：Unknown --(1)--> Opened；三态对
// 事件 0 均接受但空转；其余事件 trace "error input %d" 拒绝。
// Closed 无官方迁入路径（保留态，同 fssologlobal）。
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

// Dispatch 官方 MonitorStateMachine(event, arg1, arg2)。
func (m *StateMachine) Dispatch(event, arg1, arg2 int) bool {
	switch m.state {
	case StateUnknown:
		if event == eventStart {
			m.state = StateOpened
			return true
		}
		if event == eventTick {
			return true
		}
		m.traceError("Unknown", event)
		return false
	case StateOpened:
		if event == eventTick {
			return true
		}
		m.traceError("Opened", event)
		return false
	case StateClosed:
		if event == eventTick {
			return true
		}
		m.traceError("Closed", event)
		return false
	}
	return false
}

// traceError 官方 error input trace（状态名替换 %s）。
func (m *StateMachine) traceError(stateName string, event int) {
	m.onTrace("(SoloMonitor::MonitorState%s)error input %d", stateName, event)
}
