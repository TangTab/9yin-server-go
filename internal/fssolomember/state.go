package fssolomember

import (
	"encoding/binary"
)

// memberState 官方状态机五态（SoloMemberState.obj，state@+0x34；§七）。
type memberState int

const (
	stateUnknown memberState = iota // 0
	stateLoading                    // 1（solo 独有引导态）
	stateOpened                     // 2
	stateClosing                    // 3
	stateClosed                     // 4
)

func (st memberState) name() string {
	switch st {
	case stateUnknown:
		return "Unknown"
	case stateLoading:
		return "Loading"
	case stateOpened:
		return "Opened"
	case stateClosing:
		return "Closing"
	default:
		return "Closed"
	}
}

// 状态机事件（MemberStateMachine(id1=事件, id2, data) 的 id1；TimerCheck
// 每拍以事件 0 驱动）。
const (
	evTick  = 0 // 周期驱动（含启动引导推进）
	evClose = 1 // CloseMember / 世界关服（CommandCloseMember → 事件 1）
)

// StateMachine 官方 MemberStateMachine（@0x1029EC40 跳表）+ 五处理器。
// 返回 true 表示状态发生了推进。
func (s *Server) StateMachine(event uint32, _ uint32, _ []byte) bool {
	switch s.state {
	case stateUnknown:
		return s.stateUnknownHandler(event)
	case stateLoading:
		return s.stateLoadingHandler(event)
	case stateOpened:
		return s.stateOpenedHandler(event)
	case stateClosing:
		return s.stateClosingHandler(event)
	case stateClosed:
		return s.stateClosedHandler(event)
	}
	return false
}

// stateUnknownHandler @0x1029ED90：事件 1 → Closing；事件 0 且 store0
// 已连 → SendToStore(0, 0, 0x29, {ServerId}, 4) → Loading。
func (s *Server) stateUnknownHandler(event uint32) bool {
	if event == evClose {
		s.state = stateClosing
		return true
	}
	if event != evTick {
		return false
	}
	if !s.inner.GetStoreConnected(0) {
		return false
	}
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload[0:], s.cfg.ServerID)
	if !s.inner.SendToStore(0, 0, 0x29, payload) {
		s.opt.Logf("(SoloMember::MemberStateUnknown)send load manage data failed")
		return false
	}
	s.opt.Logf("start load manage config data...")
	s.state = stateLoading
	return true
}

// stateLoadingHandler @0x1029F020：事件 1 → Closing；三闩
// （pub +0x434 / role +0x43C / manage +0x444）全置 → Opened。
func (s *Server) stateLoadingHandler(event uint32) bool {
	if event == evClose {
		s.state = stateClosing
		return true
	}
	if s.pubReady && s.roleReady && s.manageData != nil {
		s.state = stateOpened
		return true
	}
	return false
}

// stateOpenedHandler @0x1029F0F0：事件 1 → Closing。
func (s *Server) stateOpenedHandler(event uint32) bool {
	if event == evClose {
		s.state = stateClosing
		return true
	}
	return false
}

// stateClosingHandler @0x1029F180：事件 0 每拍扫场景表——任一场景有玩家
// （"scene has player" + 场景名）则阻塞；全空 → SendToWorld(0, 0x1F,
// {1, MemberId}, 8) → Closed（失败 "send close member failed" 重试）。
func (s *Server) stateClosingHandler(event uint32) bool {
	if event != evTick {
		return false
	}
	for _, sc := range s.Scenes.Snapshot() {
		if n := s.Players.ScenePlayers(sc.ID); len(n) > 0 {
			s.opt.Logf("SoloMember::MemberStateClosing)scene has player")
			s.opt.Logf("%s", sc.Name)
			return false
		}
	}
	payload := make([]byte, 8)
	binary.LittleEndian.PutUint32(payload[0:], 1)
	binary.LittleEndian.PutUint32(payload[4:], s.cfg.MemberID)
	if !s.inner.SendToWorld(0, 0x1F, payload) {
		s.opt.Logf("SoloMember::MemberStateClosing)send close member failed")
		return false
	}
	s.state = stateClosed
	if s.opt.OnExit != nil {
		s.opt.OnExit()
	}
	return true
}

// stateClosedHandler @0x1029F320：终态（官方仅呈现）。
func (s *Server) stateClosedHandler(event uint32) bool { return false }
