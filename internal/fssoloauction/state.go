package fssoloauction

import "fmt"

// State 官方五态（GetState @0x100B1710 跳表序：this+0x30 索引 0..4）。
type State int

// 官方状态索引。
const (
	StateUnknown         State = iota // "Unknown"
	StateLoadAuctionHall              // "LoadAuctionHall"
	StateOpened                       // "Opened"
	StateClosing                      // "Closing"
	StateClosed                       // "Closed"
)

// String 官方状态名（SetAppInfo "<info state=\"%s\" ...>" 呈现用）。
func (s State) String() string {
	switch s {
	case StateUnknown:
		return "Unknown"
	case StateLoadAuctionHall:
		return "LoadAuctionHall"
	case StateOpened:
		return "Opened"
	case StateClosing:
		return "Closing"
	case StateClosed:
		return "Closed"
	}
	return fmt.Sprintf("State(%d)", int(s))
}

// 状态机事件（官方 AuctionStateMachine 第二参 id1；TimerCheck 每 1s 发
// id1=0、payload=elapsed 秒；InnerStoreConnected 发 id1=1；关服 id1=3）。
const (
	evTick           = 0
	evStoreConnected = 1
	evClose          = 3
)

// StateMachine 依赖注入面（SoloAuction 各通道的 Go 等价）。
type StateMachine struct {
	state     State
	storeSent bool // Unknown 态 0x3F 一次性闩锁（官方 this+0x13E）

	DistrictID uint32
	ServerID   uint32
	Logf       func(format string, args ...any)

	// SendReg 官方 Unknown 态：SendToStore(0, 0, 0x3F, {serverId}, 4)
	//（@0x100B19B1；成功才转 LoadAuctionHall，失败 trace
	// "(SoloAuction::AuctionStateLoadAuctionHall)send failed"——串复用）。
	SendReg func() bool
	// SendLoadReq 官方 LoadAuctionHall 态每 tick：
	// SendToStore(0, 0, 0x40, {district,server}, 8)（@0x100B1C51）。
	SendLoadReq func() bool
	// SendClosed 官方 Closing 完成后：SendToWorld(0, 1, {int 1}, 4)
	//（@0x100B217F；成功转 Closed，失败 trace "send close auction failed"）。
	SendClosed func() bool
	// OnOpenedTick 官方 Opened 态 tick：OnFrameExecute（no-op）+
	// CAuctionTimer::Run(elapsed) + SaveMinutes 存盘周期（exit=false）。
	OnOpenedTick func(elapsed int)
	// OnClosingTick 官方 Closing 态 tick：exit=true 批量退存，完成返回 true。
	OnClosingTick func(elapsed int) (allSaved bool)
	// OnServerClose 官方 Opened 收 evClose：callee OnServerClose（no-op）。
	OnServerClose func()
}

// State 当前态。
func (m *StateMachine) State() State { return m.state }

// setState 换态呈现。
func (m *StateMachine) setState(s State) {
	if m.state != s {
		m.Logf("state: %s → %s", m.state, s)
		m.state = s
	}
}

// Event 官方 AuctionStateMachine(id1, payload, size) 分发
// （TimerCheck @0x100AEFAB 传 (0, &elapsedSec, 4)）。
func (m *StateMachine) Event(ev int, elapsedSec int) {
	switch m.state {
	case StateUnknown:
		// @0x100B1973：id1==1（store 连接，一次性）→ 0x3F 注册；
		// id1==0 无事；其它事件 trace（state, id）。
		if ev == evStoreConnected {
			if m.storeSent {
				return
			}
			m.storeSent = true
			if m.SendReg() {
				m.setState(StateLoadAuctionHall)
			} else {
				m.Logf("(SoloAuction::AuctionStateLoadAuctionHall)send failed")
			}
			return
		}
		if ev != evTick {
			m.traceUnexpected(ev)
		}
	case StateLoadAuctionHall:
		// @0x100B1C13：id1==0 每 tick 重发 0x40 加载请求；
		// 转 Opened 由 0x2B 消息（LoadOK）驱动，非状态机自身。
		if ev == evTick {
			if !m.SendLoadReq() {
				m.Logf("(SoloAuction::AuctionStateLoadAuctionHall)send failed")
			}
			return
		}
		m.traceUnexpected(ev)
	case StateOpened:
		// @0x100B1D46：id1==3 → OnServerClose + SaveIndex=-1 → Closing；
		// id1==0 → OnFrameExecute + Timer::Run + 存盘周期。
		if ev == evClose {
			if m.OnServerClose != nil {
				m.OnServerClose()
			}
			m.setState(StateClosing)
			return
		}
		if ev == evTick {
			if m.OnOpenedTick != nil {
				m.OnOpenedTick(elapsedSec)
			}
			return
		}
		m.traceUnexpected(ev)
	case StateClosing:
		// @0x100B2063：id1==0 退存批（exit=true）；扫完 → SendToWorld
		// 关播 → Closed；id1==3（再次关服）无事。
		if ev == evTick {
			if m.OnClosingTick != nil && m.OnClosingTick(elapsedSec) {
				if m.SendClosed() {
					m.setState(StateClosed)
				} else {
					m.Logf("(SoloAuction.AuctionStateClosing)send close auction failed)")
				}
			}
			return
		}
		if ev != evClose {
			m.traceUnexpected(ev)
		}
	case StateClosed:
		if ev != evTick {
			m.traceUnexpected(ev)
		}
	}
}

// LoadOK 官方 0x2B StoreAuctionLoadRecordOk @0x100B276A：result!=0 →
// Opened + callee OnAuctionDataLoaded；result==0 trace
// "(SoloAuction::StoreAuctionLoadRecordOk)load pub data failed"。
func (m *StateMachine) LoadOK(result bool, onLoaded func()) {
	if !result {
		m.Logf("(SoloAuction::StoreAuctionLoadRecordOk)load pub data failed")
		return
	}
	m.setState(StateOpened)
	if onLoaded != nil {
		onLoaded()
	}
}

// traceUnexpected 官方非预期事件 trace（SendToStore+0xA0 呈现助手）。
func (m *StateMachine) traceUnexpected(ev int) {
	m.Logf("(SoloAuction)unexpected event %d in state %s", ev, m.state)
}
