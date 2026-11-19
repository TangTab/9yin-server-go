package fssolostore

// storeState 官方 solostorestate.cpp 四态（this+0x34；member 是五态，
// store 无 Loading——DB 就绪与否只反映在 AppRecord database 位）。
type storeState int

const (
	stateUnknown storeState = iota // 0
	stateOpened                    // 1
	stateClosing                   // 2
	stateClosed                    // 3
)

// stateNames GetState（@0x100C7E90）串表（echo/AppRecord 呈现）。
var stateNames = map[storeState]string{
	stateUnknown: "Unknown",
	stateOpened:  "Opened",
	stateClosing: "Closing",
	stateClosed:  "Closed",
}

func (s storeState) String() string {
	if n, ok := stateNames[s]; ok {
		return n
	}
	return "Unknown"
}

// storeEvent 状态机事件（StoreStateMachine @0x100C7F70；仅两迁移有动作）。
const (
	eventTimer = 0 // TimerCheck 每秒（Opened 内做分周表维护）
	eventStart = 1 // Start()（Unknown → Opened）
	eventClose = 2 // Stop()/OnExit（Opened → Closing）
)

// transition 官方四态迁移定案：
//
//	Unknown:  事件1 → Opened；事件0 无迁移；其余 trace。
//	Opened:   事件2 → Closing；事件0 = 分周表维护（state.go 不承载，
//	          由 Server.TimerCheck 承担——官方 Opened 内联逻辑）。
//	Closing:  事件2/0 且 AccessEmpty() → 发关播帧（失败重试留态）→ Closed。
//	          官方事件 0 不在 Closing 分支处理（无迁移）。
//	Closed:   终态无迁出（无迁入路径之外的分支，同构保留）。
func transition(cur storeState, event int) storeState {
	switch cur {
	case stateUnknown:
		if event == eventStart {
			return stateOpened
		}
	case stateOpened:
		if event == eventClose {
			return stateClosing
		}
	case stateClosing:
		// 迁移由 Server 承担（需 AccessEmpty 门 + 关播帧）。
	case stateClosed:
	}
	return cur
}
