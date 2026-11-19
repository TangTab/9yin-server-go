package fsserver

import (
	"fmt"
)

// processEntryMsg ProcessEntryMsg 跳表六型（§5.1）。
func (w *World) processEntryMsg(m msg) {
	switch m.typ {
	case EntryMsgTypeRegister:
		w.entryRegister(m)
	case EntryMsgTypeClientAccept:
		w.entryClientAccept(m)
	case EntryMsgTypeClientClosed:
		w.entryClientClosed(m)
	case EntryMsgTypeClientMessage:
		w.entryClientMessage(m)
	case EntryMsgTypeClientIllegal:
		w.entryClientIllegal(m)
	case EntryMsgTypeCloseEntry:
		w.entryCloseEntry(m)
	}
}

// entryRegister 校验 {DistrictID, ServerID, EntryIdx} → 回 init_connect。
func (w *World) entryRegister(m msg) {
	r, err := DecodeEntryRegister(m.bytes())
	if err != nil {
		w.logf("(GameWorld::EntryMsgRegister)%v", err)
		return
	}
	if r.DistrictID != w.DistrictID || r.ServerID != w.ServerID {
		w.logf("(GameWorld::EntryMsgRegister)entry ident error")
		return
	}
	open := uint32(0)
	if w.State == WStateOpened {
		open = 1
	}
	var b [4]byte
	b[0] = byte(open)
	_ = w.sendToEntry(r.EntryIdx, ForwInitConnect, b[:])
}

// entryClientAccept 28B → AddPlayer。
func (w *World) entryClientAccept(m msg) {
	a, err := DecodeEntryClientAccept(m.bytes())
	if err != nil {
		w.logf("(GameWorld::EntryMsgClientAccept)%v", err)
		return
	}
	w.Players.AddPlayer(a.Ident, a.Serial, entryIndexOf(m.ident))
}

// entryIndexOf entry ident → idx（8..0x17 → 0..15）。
func entryIndexOf(ident uint32) uint32 {
	if ident >= EntryIdentBase && ident < EntryIdentBase+MaxEntryNum {
		return ident - EntryIdentBase
	}
	return 0
}

// entryClientClosed 8B → 断线事件。
func (w *World) entryClientClosed(m msg) {
	if len(m.bytes()) != 8 {
		w.logf("len == sizeof(entry_back_client_closed_t) (%d)", len(m.bytes()))
		return
	}
	ident := leU32(m.bytes()[0:])
	serial := leU32(m.bytes()[4:])
	p := w.Players.FindByConnect(ident, serial)
	if p == nil {
		w.logf("(GameWorld::EntryMsgClientClosed)player not found")
		return
	}
	w.PlayerStateMachine(p, EventClosed, nil, 0)
}

// entryClientMessage {ident,serial}+帧 → opcode 分派（§4.2）。
func (w *World) entryClientMessage(m msg) {
	if len(m.bytes()) < 8 {
		w.logf("(GameWorld::EntryMsgClientMessage)message size error")
		return
	}
	ident := leU32(m.bytes()[0:])
	serial := leU32(m.bytes()[4:])
	frame := m.bytes()[8:]
	p := w.Players.FindByConnect(ident, serial)
	if p == nil {
		w.logf("(GameWorld::EntryMsgClientMessage)no this player")
		return
	}
	if len(frame) == 0 {
		return
	}
	var event uint32
	switch frame[0] {
	case ClientOpGetVerify:
		event = EventGetVerify
	case ClientOpLogin:
		event = EventLogin
	case ClientOpWorldInfo:
		event = EventWorldInfo
	case ClientOpChooseRole:
		event = EventChooseRole
	case ClientOpCreateRole:
		event = EventCreateRole
	case ClientOpDeleteRole:
		event = EventDeleteRole
	default:
		return // 官方言：type 1 及未知静默放行
	}
	w.PlayerStateMachine(p, event, frame, len(frame))
}

// entryClientIllegal 非法回报 → 踢线。
func (w *World) entryClientIllegal(m msg) {
	p := w.playerOf8(m)
	if p == nil {
		w.logf("(GameWorld::EntryMsgClientIllegal)player not found")
		return
	}
	w.breakClient(p)
	w.PlayerStateMachine(p, EventClosed, nil, 0)
}

// entryCloseEntry 关服应答 → WorldStateMachine。
func (w *World) entryCloseEntry(m msg) {
	w.WorldStateMachine(0, 0, nil)
}

func (w *World) playerOf8(m msg) *Player {
	if len(m.bytes()) < 8 {
		return nil
	}
	return w.Players.FindByConnect(leU32(m.bytes()[0:]), leU32(m.bytes()[4:]))
}

// processStoreMsg ProcessStoreMsg 跳表 13 型（§5.3）。
func (w *World) processStoreMsg(m msg) {
	switch m.typ {
	case StoreMsgRoleInfo:
		w.storeRoleInfo(m)
	case StoreMsgLoad:
		p := w.playerOf8(m)
		if p == nil {
			w.logf("(GameWorld::StoreMsgLoad)This player not found")
			return
		}
		w.PlayerStateMachine(p, EventTick, m.bytes()[8:], 0)
	case StoreMsgSave:
		p := w.playerOf8(m)
		if p == nil {
			w.logf("(GameWorld::StoreMsgSave)can't store player")
			return
		}
		w.logf("(GameWorld::StoreMsgSave)player stored")
	case StoreMsgRoles:
		p := w.playerOf8(m)
		if p == nil {
			w.logf("(GameWorld::StoreMsgRoles)player not found")
			return
		}
		p.RoleBlob = m.bytes()[8:]
		if p.State == PStateGetting {
			w.setPlayerState(p, PStateLogged)
		}
		w.sendPlayerRoles(p)
	case StoreMsgCreate:
		w.storeCreate(m)
	case StoreMsgDelete:
		w.storeDelete(m)
	case StoreMsgAccountLogon, StoreMsgAccountValidate:
		p := w.playerOf8(m)
		if p == nil {
			w.logf("(GameWorld::StoreMsgAccountLogon)can't find this player")
			return
		}
		w.PlayerStateMachine(p, EventTick, m.bytes()[8:], 0)
	case StoreMsgCloseStore:
		w.WorldStateMachine(0, 0, nil)
	}
}

// storeRoleInfo 角色名全量重建 → LoadRoles 推进。
func (w *World) storeRoleInfo(m msg) {
	w.Names.Clear()
	r := NewLoadReader(m.bytes())
	for r.Remaining() >= 12 {
		name, err1 := r.ReadWideStr()
		acct, err2 := r.ReadString()
		deleted, err3 := r.ReadBool()
		if err1 != nil || err2 != nil || err3 != nil {
			w.logf("(GameWorld::StoreMsgRoleInfo)load role info failed")
			break
		}
		_ = w.Names.Add(name, acct, 0)
		_ = deleted
	}
	if w.State == WStateUnknown {
		w.SetWorldState(WStateLoadRoles)
	}
}

// storeCreate 建角应答：RoleAdd 广播 + 角色列表重发。
func (w *World) storeCreate(m msg) {
	p := w.playerOf8(m)
	if p == nil {
		w.logf("(GameWorld::StoreMsgCreate)this player not found")
		return
	}
	r := NewLoadReader(m.bytes()[8:])
	role, err1 := r.ReadWideStr()
	if err1 != nil {
		return
	}
	w.Names.Add(role, p.Account, 0)
	w.sendToAllMembers(0, MemberRoleAdd, buildRoleMsg(p.Account, role))
	w.setPlayerState(p, PStateLogged)
	w.sendPlayerRoles(p)
}

// storeDelete 删角应答。
func (w *World) storeDelete(m msg) {
	p := w.playerOf8(m)
	if p == nil {
		w.logf("(GameWorld::StoreMsgDelete)this player not found")
		return
	}
	r := NewLoadReader(m.bytes()[8:])
	role, err1 := r.ReadWideStr()
	if err1 != nil {
		return
	}
	w.Names.Remove(role)
	w.sendToAllMembers(0, MemberRoleRemove, buildRoleMsg(p.Account, role))
	w.setPlayerState(p, PStateLogged)
	w.sendPlayerRoles(p)
}

func buildRoleMsg(account, role string) []byte {
	out := NewStoreWriter(0x80)
	out.Seek(4)
	out.WriteString(account)
	out.WriteWideStr(role)
	return out.Bytes()
}

// processMemberMsg ProcessMemberMsg 跳表 27 型（§5.2；礼品/点卡族转发
// charge 文本协议，Gmcc/Stat/Manage/Extra 族透传连接器）。
func (w *World) processMemberMsg(m msg) {
	switch m.typ {
	case MemberMsgTypeSceneState:
		w.memberSceneState(m)
	case MemberMsgTypeBreakPlayer:
		p := w.playerOf8(m)
		if p == nil {
			w.logf("(GameWorld::MemberMsgBreakPlayer)player not found")
			return
		}
		w.PlayerStateMachine(p, EventClosed, nil, 0)
	case MemberMsgTypeLeaveGame:
		p := w.playerOf8(m)
		if p == nil {
			w.logf("(GameWorld::MemberMsgLeaveGame)player not found")
			return
		}
		w.setPlayerState(p, PStateOffline)
		w.PlayerStateMachine(p, EventTick, nil, 0)
	case MemberMsgTypeCreateClone:
		// member 侧副本实例建立回报（场景表登记）
		if len(m.bytes()) >= 8 {
			id := leU32(m.bytes()[0:])
			_ = id
		}
	case MemberMsgTypeCloseMember:
		w.WorldStateMachine(0, 0, nil)
	}
}

// memberSceneState 场景状态回报（Starting→Running/Error）。
func (w *World) memberSceneState(m msg) {
	if len(m.bytes()) < 8 {
		w.logf("len == sizeof(member_back_scene_state_t)")
		return
	}
	id := leU32(m.bytes()[0:])
	st := SceneState(int32(leU32(m.bytes()[4:])))
	if !w.Scenes.SetState(id, st) {
		w.logf("(GameWorld::MemberMsgSceneState)scene not found")
		return
	}
	if st == SSceneRunning {
		w.sendToAllMembers(0, MemberSceneAdd, buildSceneMsg(w.Scenes.GetScene(id)))
	}
}

// buildSceneMsg SceneAdd/Remove（{id, member, kind, state, script, config}）。
func buildSceneMsg(s *Scene) []byte {
	if s == nil {
		return nil
	}
	out := NewStoreWriter(0x100)
	out.Seek(4)
	out.WriteInt(int32(s.ID))
	out.WriteInt(int32(s.MemberIdx))
	out.WriteInt(int32(s.Kind))
	out.WriteInt(int32(s.State))
	out.WriteString(s.Script)
	out.WriteString(s.Config)
	return out.Bytes()
}

// processPublicMsg ProcessPublicMsg 三型（§5.4）。
func (w *World) processPublicMsg(m msg) {
	switch m.typ {
	case PublicMsgLoadPlayer:
		// public 代载完成 → InnerLoadPlayer 语义：物化玩家并驱动
		if len(m.bytes()) < 12 {
			return
		}
		acct := cstr(m.bytes()[0:])
		role := wstrAt(m.bytes()[8:])
		w.innerLoadPlayer(acct, role)
	case PublicMsgClosePublic:
		w.WorldStateMachine(0, 0, nil)
	}
}

// innerLoadPlayer 代载物化（§5.5 InnerLoadPlayer）。
func (w *World) innerLoadPlayer(acct, role string) {
	if acct == "" || role == "" {
		w.logf("(GameWorld::InnerLoadLoadPlayer)parameter error")
		return
	}
	if w.Players.FindByAccount(acct) != nil {
		w.logf("(GameWorld::InnerLoadLoadPlayer)account exists")
		return
	}
	if w.Players.FindByName(role) != nil {
		w.logf("(GameWorld::InnerLoadLoadPlayer)role exists")
		return
	}
	p := w.Players.AddPlayer(0, 0, 0)
	w.Players.SetPlayerAccount(p, acct)
	w.Players.SetupPlayerName(p, role)
	p.RoleName = role
	p.InGame = true
	w.logf("inner load player %s/%s", acct, role)
}

// processInnerMsg ProcessInnerMsg 跳表 14 型（§5.5）。
func (w *World) processInnerMsg(m msg) {
	switch m.typ {
	case InnerMsgTypeMemberConnected:
		// {memberIdx} → 三段全量同步
		idx := uint32(0)
		if len(m.bytes()) >= 4 {
			idx = leU32(m.bytes()[0:])
		}
		w.syncAllScene(idx)
		w.syncAllPlayer(idx)
		w.syncAllRoleInfo(idx)
	case InnerMsgTypeMemberDisconnect, InnerMsgTypeStoreDisconnect, InnerMsgTypeEntryDisconnect:
		// 连接层已解绑（assemble 注入）；场景状态回置
	case InnerMsgTypeOpenServer:
		if w.State == WStateOpened {
			w.sendOpenConnect()
		}
	case InnerMsgTypeCloseServer:
		w.SetWorldState(WStateCloseEntry)
	case InnerMsgTypeDumpPlayers:
		w.dumpPlayers()
	case InnerMsgTypeDumpScenes:
		w.dumpScenes()
	}
}

// syncAllScene world→member SceneAll（type 4，§6.2 写序）。
func (w *World) syncAllScene(memberIdx uint32) {
	out := NewStoreWriter(0x400)
	out.Seek(4)
	dump := func(s *Scene) bool {
		out.WriteInt(int32(s.ID))
		out.WriteInt(int32(s.MemberIdx))
		out.WriteInt(int32(s.Kind))
		out.WriteInt(int32(s.State))
		out.WriteString(s.Script)
		out.WriteString(s.Config)
		return true
	}
	w.Scenes.EachNormal(dump)
	w.Scenes.EachClone(dump)
	_ = w.IC.SendToMember(memberIdx, 0, MemberSceneAll, out.Bytes())
}

// syncAllPlayer world→member PlayerAll（type 7）。
func (w *World) syncAllPlayer(memberIdx uint32) {
	out := NewStoreWriter(0x400)
	out.Seek(4)
	w.Players.Each(func(p *Player) bool {
		if p.InGame {
			out.WriteInt(int32(p.SceneID))
			out.WriteInt(int32(p.Ident))
			out.WriteInt(int32(p.Serial))
			out.WriteWideStr(p.RoleName)
			out.WriteString(p.Account)
		}
		return true
	})
	_ = w.IC.SendToMember(memberIdx, 0, MemberPlayerAll, out.Bytes())
}

// syncAllRoleInfo world→member RoleInfo（type 0xB；定向 + 全量双发）。
func (w *World) syncAllRoleInfo(memberIdx uint32) {
	out := NewStoreWriter(0x400)
	out.Seek(4)
	w.Names.Snapshot(func(name, account string, uid int32, deleted bool) {
		out.WriteWideStr(name)
		out.WriteString(account)
		out.WriteBool(deleted)
	})
	_ = w.IC.SendToMember(memberIdx, 0, MemberRoleInfo, out.Bytes())
	_ = w.IC.SendToAllMember(0, MemberRoleInfo, out.Bytes())
}

// dumpPlayers / dumpScenes 运维面（echo "players" 同源）。
func (w *World) dumpPlayers() {
	w.logf("player: %d, online: %d, queue: %d", w.Players.Count(), w.onlineCount(), w.GetQueueCount())
}

// dumpScenes …
func (w *World) dumpScenes() {
	n, c := w.Scenes.Counts()
	w.logf("scene: %d, clone: %d", n, c)
}

// ---- 小工具 ----

func leU32(b []byte) uint32 {
	_ = b[3]
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func cstr(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func wstrAt(b []byte) string {
	if len(b) < 2 {
		return ""
	}
	r := NewLoadReader(b)
	s, err := r.ReadWideStr()
	if err != nil {
		return ""
	}
	return s
}

var _ = fmt.Sprintf
