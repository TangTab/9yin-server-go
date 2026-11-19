package fssoloworld

import (
	"encoding/binary"
)

// ProcessEntryMsg 官方 id2 0..5 分发表（§十；跳表 @0x100c2688）。
func (s *Server) ProcessEntryMsg(id1, id2 uint32, payload []byte) bool {
	switch id2 {
	case 0:
		return s.entryClientAccept(id1, payload)
	case 1:
		return s.entryClientClosed(id1, payload)
	case 2:
		return s.entryClientLost(id1, payload)
	case 3:
		return s.entryClientMessage(id1, payload)
	case 4:
		return s.entryClientIllegal(id1, payload)
	case 5:
		return s.entryCloseEntry(id1, payload)
	default:
		return false
	}
}

// entryClientAccept：{client_id, entry_idx?} 8B → AddPlayer(Unknown)。
func (s *Server) entryClientAccept(id1 uint32, p []byte) bool {
	if len(p) < 8 {
		s.trace("(SoloWorld::EntryMsgClientAccept)len != sizeof(entry_back_client_t)")
		return false
	}
	clientID := binary.LittleEndian.Uint32(p[0:])
	serial := binary.LittleEndian.Uint32(p[4:])
	if s.players.FindByConnect(clientID, serial) != nil {
		return true
	}
	s.players.AddPlayer(clientID, serial, "")
	s.trace("entry client %d accept", clientID)
	return true
}

// entryClientClosed：玩家断开（Online→下线链或直删）。
func (s *Server) entryClientClosed(id1 uint32, p []byte) bool {
	if len(p) < 8 {
		return false
	}
	clientID := binary.LittleEndian.Uint32(p[0:])
	serial := binary.LittleEndian.Uint32(p[4:])
	pl := s.players.FindByConnect(clientID, serial)
	if pl == nil {
		return true
	}
	s.playerLeave(pl)
	return true
}

// entryClientLost：entry 丢失客户端（更强的清理，断 charge 侧会话）。
func (s *Server) entryClientLost(id1 uint32, p []byte) bool {
	return s.entryClientClosed(id1, p)
}

// entryClientMessage：8B 头 + msgtype 子分发（§十 msgtype 0..0xC）。
func (s *Server) entryClientMessage(id1 uint32, p []byte) bool {
	clientID, serial, mt, body, ok := decodeClientMsg(p)
	if !ok {
		s.trace("(SoloWorld::EntryMsgClientMessage)len <= sizeof(entry_back_client_t)")
		return false
	}
	pl := s.players.FindByConnect(clientID, serial)
	if pl == nil {
		pl = s.players.AddPlayer(clientID, serial, "")
	}
	switch mt {
	case clientMsgGetVerify:
		// UseVerifyImage=false 时直接跳过（MatrixCode 不复刻）
		return true
	case clientMsgLogin:
		return s.clientLogin(pl, body)
	case clientMsgWorldInfo:
		return s.clientWorldInfo(pl)
	case clientMsgChooseRole:
		return s.clientChooseRole(pl, body)
	case clientMsgCreateRole:
		return s.clientCreateRole(pl, body)
	case clientMsgDeleteRole:
		return s.clientDeleteRole(pl, body)
	case clientMsgReviveRole:
		return s.clientReviveRole(pl, body)
	default:
		return true
	}
}

// entryClientIllegal：非法客户端（直接断）。
func (s *Server) entryClientIllegal(id1 uint32, p []byte) bool {
	if len(p) < 8 {
		return false
	}
	clientID := binary.LittleEndian.Uint32(p[0:])
	serial := binary.LittleEndian.Uint32(p[4:])
	if pl := s.players.FindByConnect(clientID, serial); pl != nil {
		s.players.RemovePlayer(clientID, serial)
	}
	return true
}

// entryCloseEntry：entry 关闭（ClearScenes 链的一环）。
func (s *Server) entryCloseEntry(id1 uint32, p []byte) bool {
	s.trace("entry server close succeed")
	return true
}

// clientLogin 载荷：length-prefixed account/password（CLoadArchive 双 ReadString）。
func (s *Server) clientLogin(pl *Player, body []byte) bool {
	account, _, rest := readArchiveString(body)
	if account == "" {
		s.trace("(SoloWorld::PlayerStateUnknown)get account password error")
		return false
	}
	password, _, _ := readArchiveString(rest) // 校验域（charge 侧对账；此处保留解析）
	_ = password
	if pl.Account == "" {
		s.players.mu.Lock()
		if pl.Account == "" {
			pl.Account = account
			s.players.byAcc[account] = pl
		}
		s.players.mu.Unlock()
	}
	// PlayerQueue 且满员 → Queuing，否则进 Logging（charge login 由
	// ChargeHook 代理；未接 charge 直进 Getting 下发角色列表）
	if s.cfg.PlayerQueue && s.players.CountActive() >= s.cfg.MaxPlayers {
		s.players.SetPlayerState(pl, PlQueuing)
		return true
	}
	s.players.SetPlayerState(pl, PlLogging)
	if s.charge == nil || !s.charge.Connected() {
		// WaitChargeSeconds 超时回退路径：直进 Getting
		s.players.SetPlayerState(pl, PlGetting)
	}
	return true
}

// clientWorldInfo 下发世界信息（ServerName/DistrictId/...）。
func (s *Server) clientWorldInfo(pl *Player) bool {
	s.players.SetPlayerState(pl, PlGetting)
	return true
}

// clientChooseRole 载荷：{role uid[33]} → Loading→member。
func (s *Server) clientChooseRole(pl *Player, body []byte) bool {
	uid, _, _ := readArchiveString(body)
	if uid == "" {
		return false
	}
	s.players.SetPlayerState(pl, PlLogged)
	return true
}

// clientCreateRole 载荷：{name wchar, config...} → Creating（store create）。
func (s *Server) clientCreateRole(pl *Player, body []byte) bool {
	name, _, _ := readArchiveWidestr(body)
	if name == "" {
		return false
	}
	if len([]rune(name)) > s.cfg.RoleNameLimit {
		return false
	}
	if s.names.FindName(name) || s.players.FindByName(name) != nil {
		return false // 名字占用
	}
	s.players.SetPlayerState(pl, PlCreating)
	s.pendingCreate[pl.Conn1] = name
	return true
}

func (s *Server) clientDeleteRole(pl *Player, body []byte) bool {
	uid, _, _ := readArchiveString(body)
	s.players.SetPlayerState(pl, PlDeleting)
	_ = uid
	return true
}

func (s *Server) clientReviveRole(pl *Player, body []byte) bool {
	uid, _, _ := readArchiveString(body)
	s.players.SetPlayerState(pl, PlReviving)
	_ = uid
	return true
}

// ProcessStoreMsg 官方 id2 0..0xE 分发表（§十；跳表 @0x100de300）。
func (s *Server) ProcessStoreMsg(id1, id2 uint32, payload []byte) bool {
	switch id2 {
	case 0: // RoleInfo：NameUid 全量同步（SyncRoleInfo 源）
		return s.storeRoleInfo(id1, payload)
	case 1: // Load：账号角色列表
		return s.storeLoad(id1, payload)
	case 6: // Create：建角应答
		return s.storeCreate(id1, payload)
	case 7: // Delete
		return s.storeDelete(id1, payload)
	case 8: // Revive
		return s.storeRevive(id1, payload)
	case 0xE: // CloseStore：关闭链推进
		if s.state == WsCloseStore {
			s.setState(WsClosed)
		}
		return true
	default:
		// 2 Save / 3 Block / 4 Resume / 5 Roles / 9 Recreate /
		// A AccountReset / B AccountLogon / C AccountLogoff：
		// 存档细节域，框架层记录后透传
		return true
	}
}

// storeRoleInfo：{count, {name wchar, uid, deleted}...} 全量。
func (s *Server) storeRoleInfo(id1 uint32, p []byte) bool {
	if len(p) < 4 {
		return false
	}
	n := int(binary.LittleEndian.Uint32(p[0:]))
	rest := p[4:]
	for i := 0; i < n; i++ {
		name, nr, r1 := readArchiveWidestr(rest)
		uid, ur, r2 := readArchiveString(r1)
		if nr == 0 || ur == 0 {
			break
		}
		deleted := false
		if len(r2) >= 1 {
			deleted = r2[0] != 0
		}
		s.names.Add(name, uid, deleted)
		rest = r2[1:]
	}
	if s.state == WsLoadRoles {
		s.setState(WsLoadScenes)
	}
	return true
}

// storeLoad：{client_id, serial, account, count, {uid,name,level,...}...}。
func (s *Server) storeLoad(id1 uint32, p []byte) bool {
	if len(p) < 8 {
		return false
	}
	clientID := binary.LittleEndian.Uint32(p[0:])
	serial := binary.LittleEndian.Uint32(p[4:])
	pl := s.players.FindByConnect(clientID, serial)
	if pl != nil {
		s.players.SetPlayerState(pl, PlGetting)
	}
	return true
}

// storeCreate：store_back_create_t {client_id, serial, uid, ...}。
func (s *Server) storeCreate(id1 uint32, p []byte) bool {
	if len(p) < 8 {
		return false
	}
	clientID := binary.LittleEndian.Uint32(p[0:])
	serial := binary.LittleEndian.Uint32(p[4:])
	rest := p[8:]
	uid, _, r := readArchiveString(rest)
	name, _, _ := readArchiveWidestr(r)
	if uid == "" || name == "" {
		return false
	}
	if !s.names.Add(name, uid, false) {
		s.trace("(SoloWorld::StoreMsgCreate)add name uid failed")
		return false
	}
	// 官方：角色新增广播 member/public/auction 三端 + 统计上报
	s.broadcastRoleChange(name, uid, true)
	pl := s.players.FindByConnect(clientID, serial)
	if pl != nil {
		s.players.SetupPlayerName(clientID, serial, name)
		s.players.SetPlayerState(pl, PlLogged)
	}
	return true
}

func (s *Server) storeDelete(id1 uint32, p []byte) bool {
	if len(p) < 8 {
		return false
	}
	clientID := binary.LittleEndian.Uint32(p[0:])
	serial := binary.LittleEndian.Uint32(p[4:])
	if pl := s.players.FindByConnect(clientID, serial); pl != nil {
		s.players.SetPlayerState(pl, PlLogged)
	}
	return true
}

func (s *Server) storeRevive(id1 uint32, p []byte) bool {
	return s.storeDelete(id1, p)
}

// ProcessMemberMsg 官方 id2 0..0x24 分发表（§十；跳表 @0x100cb9bc）。
// 框架层覆盖场景/玩家联动核心 case，其余记录透传。
func (s *Server) ProcessMemberMsg(id1, id2 uint32, payload []byte) bool {
	switch id2 {
	case 4: // RoleInfo：member 在册角色（SyncAllPlayer 源）
		return true
	case 5: // SceneState：场景状态推进（§九）
		return s.memberSceneState(id1, payload)
	case 0x1F: // CloseMember：关闭链推进
		if s.state == WsCloseMember {
			s.setState(WsClosePublic)
		}
		return true
	case 8: // BreakPlayer：场景踢人
		return s.memberBreakPlayer(id1, payload)
	case 0x20: // LeaveGame
		return s.memberLeaveGame(id1, payload)
	default:
		return true
	}
}

// memberSceneState：{scene_index, state}。
func (s *Server) memberSceneState(id1 uint32, p []byte) bool {
	if len(p) < 8 {
		return false
	}
	index := int(binary.LittleEndian.Uint32(p[0:]))
	st := int(binary.LittleEndian.Uint32(p[4:]))
	var ss SceneState
	switch st {
	case 1:
		ss = ScStarting
	case 2:
		ss = ScRunning
	case 3:
		ss = ScCloning
	case 4:
		ss = ScCloned
	case 5:
		ss = ScDeleting
	case 7:
		ss = ScError
	default:
		ss = ScUnknown
	}
	ok := s.scenes.SetState(index, ss)
	if ok && s.state == WsLoadScenes && s.scenes.AllRunning() {
		s.setState(WsCanOpen)
	}
	return ok
}

func (s *Server) memberBreakPlayer(id1 uint32, p []byte) bool {
	if len(p) < 8 {
		return false
	}
	clientID := binary.LittleEndian.Uint32(p[0:])
	serial := binary.LittleEndian.Uint32(p[4:])
	if pl := s.players.FindByConnect(clientID, serial); pl != nil {
		s.players.SetPlayerState(pl, PlWaitBreak)
	}
	return true
}

func (s *Server) memberLeaveGame(id1 uint32, p []byte) bool {
	if len(p) < 8 {
		return false
	}
	clientID := binary.LittleEndian.Uint32(p[0:])
	serial := binary.LittleEndian.Uint32(p[4:])
	if pl := s.players.FindByConnect(clientID, serial); pl != nil {
		s.playerLeave(pl)
	}
	return true
}

// broadcastRoleChange 角色增删三端广播（StoreMsgCreate/Delete 语义）。
func (s *Server) broadcastRoleChange(name, uid string, add bool) {
	if s.inner == nil {
		return
	}
	// 官方 SendToAllMember/SendToPublic/SendToAuction 角色增删帧
	s.inner.SendToAllMember(0, 6, encodeRoleChange(name, uid, add))
	s.inner.SendToPublic(0, 6, encodeRoleChange(name, uid, add))
	s.inner.SendToAuction(0, 6, encodeRoleChange(name, uid, add))
}

func encodeRoleChange(name, uid string, add bool) []byte {
	nb := []byte(name)
	ub := []byte(uid)
	b := make([]byte, 4+len(nb)+1+len(ub)+1)
	binary.LittleEndian.PutUint32(b[0:], uint32(len(nb)))
	copy(b[4:], nb)
	b[4+len(nb)] = byte(map[bool]int{true: 1, false: 0}[add])
	copy(b[5+len(nb):], ub)
	return b
}

// readArchiveString CLoadArchive::ReadString：u32 长度 + 字节（NUL 结尾可选）。
func readArchiveString(p []byte) (val string, n int, rest []byte) {
	if len(p) < 4 {
		return "", 0, p
	}
	l := int(binary.LittleEndian.Uint32(p[0:]))
	if l < 0 || 4+l > len(p) {
		return "", 0, p
	}
	return string(p[4 : 4+l]), 4 + l, p[4+l:]
}

// readArchiveWidestr UCS2→string（Port_UCS2ToWideStr 语义）。
func readArchiveWidestr(p []byte) (val string, n int, rest []byte) {
	if len(p) < 4 {
		return "", 0, p
	}
	l := int(binary.LittleEndian.Uint32(p[0:])) // 字符数
	if l < 0 || 4+l*2 > len(p) {
		return "", 0, p
	}
	b := p[4 : 4+l*2]
	u16 := make([]uint16, l)
	for i := 0; i < l; i++ {
		u16[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	return string(decodeUTF16(u16)), 4 + l*2, p[4+l*2:]
}
