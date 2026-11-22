package fssoloworld

import (
	"bytes"
	"encoding/binary"
	"time"
)

// ProcessEntryMsg 官方 id2 0..5 分发表（§5.1 跳表 @0x10084D78；编号与
// fsserver/wire.go EntryMsgType* 同源钉死：0=register/1=accept/2=closed/
// 3=message/4=illegal/5=close。旧载 "0=accept/1=closed" 为误读——accept
// 误接 closed 曾让断线清理永不触发，重登同名角色被 byName 陈旧项卡死）。
func (s *Server) ProcessEntryMsg(id1, id2 uint32, payload []byte) bool {
	switch id2 {
	case 0:
		return s.entryClientRegister(id1, payload)
	case 1:
		return s.entryClientAccept(id1, payload)
	case 2:
		return s.entryClientClosed(id1, payload)
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

// entryClientRegister entry 注册帧（官方 12B {DistrictID,ServerID,EntryIdx}
// 校验 + 回 init_connect 旗标）。Go 端注册在 InnerLink identify 层消化、
// 开闸由 OpenServer 批量 init_connect 承担，业务帧到达此处仅簿记。
func (s *Server) entryClientRegister(id1 uint32, p []byte) bool {
	s.trace("entry register id=%d len=%d", id1, len(p))
	return true
}

// entryClientAccept：官方 28B {client_id, serial, addr[16], port}（首 8B
// 即连接键）→ AddPlayer(Unknown)。
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

// entryClientClosed：连接关闭/失联回报（网关读循环退出与 shake 未命中
// 两源同型）→ 0x37 logoff + playerLeave 清场——byName 陈旧项即在此释放，
// 重登同名角色不再被 "player already in game" 静默拒绝。
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
	// 官方 0x37 AccountLogoff：logged 会话清零（store 侧 dc_accounts）
	if pl.Account != "" {
		s.mu.Lock()
		session := s.session
		s.mu.Unlock()
		s.inner.SendToAllStore(0, 0x37,
			accountLogoffFrame(clientID, serial, session, pl.Account))
	}
	// 0x04 下线存档（GM 热改后的 roleState 权威快照落库；无 roleState
	// 的未进场会话跳过）。存档发出后不等待回执（0x03 trace 簿记）。
	if pl.haveRoleState {
		s.sendRoleSave(pl)
	}
	s.playerLeave(pl)
	return true
}

// entryClientMessage：8B 头 + msgtype 子分发（§十 msgtype 0..0xC）。
func (s *Server) entryClientMessage(id1 uint32, p []byte) bool {
	clientID, serial, mt, body, ok := decodeClientMsg(p)
	s.trace("entry client message %d/%d mt=0x%X len=%d ok=%v", clientID, serial, mt, len(body), ok)
	if !ok {
		return false
	}
	if mt >= 1 {
		mt-- // 客户端 opcode → 官方内网 msgtype（op01→0 verify/op02→1 login/…）
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
		// auth 解析要完整 op02 帧（message[0]==0x02 头校验）——重组
		// opcode 号（mt 已折算内网号，还原 +1）
		return s.clientLogin(pl, append([]byte{byte(mt + 1)}, body...))
	case clientMsgWorldInfo:
		return s.clientWorldInfo(pl)
	case clientMsgChooseRole:
		return s.clientChooseRole(pl, body)
	case clientMsgCreateRole:
		return s.clientCreateRole(pl, body)
	case clientMsgDeleteRole:
		return s.clientDeleteRole(pl, body)
	case clientMsgReviveRole:
		// op0x07 双用法分流：33B 定长 object request（点击场景对象——
		// NPC 交互链，2433 定版形态）优先；其余形态维持复活占位。
		if req, ok := parseNPCObjectRequest(append([]byte{0x07}, body...)); ok {
			return s.handleNPCObjectRequest(pl, req)
		}
		return s.clientReviveRole(pl, body)
	case clientMsgReady:
		// op09 ClientReady（单字节）→ 新手视图簇（View43 内功书/
		// View40 技能行；View43 缺发 = 「普通开始」按钮禁用）
		s.clientReady(pl)
		return true
	case clientMsgActivity:
		// op0x0A 活动信封：仅 21B 头 + TVarList 形态按自定义消息消费
		//（107/195/99/199/200/201 等可经此路），其余活动流量放行。
		return s.clientActivity(pl, append([]byte{0x0A}, body...))
	case clientMsgCustom:
		// op0x1E 自定义消息（GameVisual:CustomSend TVarList）→
		// 自定义配置域接收链（键位/快捷栏/个性配置上传）。
		return s.clientCustom(pl, append([]byte{0x1E}, body...))
	case clientMsgPing:
		// op0x18 心跳 → 0x38 外服形态（30B）：12B 回显 + 18B 尾
		// （u16 4 + uptime_ms×2 + 计数——外服 s2c#224 逆向；12B 纯回显
		// 形态客户端稳定态崩，2026-11-21 capture 对拍）
		echo := append([]byte{0x38}, body...)
		ms := uint32(time.Since(processStart).Milliseconds())
		tail := []byte{0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00}
		binary.LittleEndian.PutUint32(tail[3:], ms)
		binary.LittleEndian.PutUint32(tail[13:], ms)
		s.sendToClient(pl, append(echo, tail...))
		return true
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

// processStart 心跳尾 uptime_ms 基准（进程启动）。
var processStart = time.Now()

// clientLogin 载荷：length-prefixed account/password（CLoadArchive 双 ReadString）。
func (s *Server) clientLogin(pl *Player, body []byte) bool {
	// 官方通路：op02 双槽经 internal/auth（b-line 双域 DES 阶梯）解出
	// 明文 → store 0x36 AccountLogon（首登自动建号/密码等值/logged 门）
	// → 0x25 回执驱动（accountLogonReply）。PlayerQueue 满员照旧排队。
	if s.cfg.PlayerQueue && s.players.CountActive() >= s.cfg.MaxPlayers {
		s.players.SetPlayerState(pl, PlQueuing)
		return true
	}
	s.players.SetPlayerState(pl, PlLogging)
	if !s.sendAccountLogon(pl, body) {
		s.trace("(SoloWorld::PlayerStateUnknown)account logon send failed")
		s.inner.SendToAllEntry(0, entryBreakClient, entryClientKey(pl))
	}
	return true
}

// entryClientKey 下行/断开帧的 {client_id, serial} 头。
func entryClientKey(pl *Player) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b[0:], pl.Conn1)
	binary.LittleEndian.PutUint32(b[4:], pl.Conn2)
	return b
}

// clientWorldInfo 下发世界信息（ServerName/DistrictId/...）。
func (s *Server) clientWorldInfo(pl *Player) bool {
	// op03 世界信息 → 下行 0x05（infoType 0 + 五书开放串，真机回放值）
	s.players.SetPlayerState(pl, PlGetting)
	s.sendToClient(pl, worldInfoFrame(0, openBooks))
	return true
}

// clientChooseRole 载荷：UCS2 角色名（真机定长 0x54 帧 [1:0x43] 名字
// 区→body[0:0x42]，FsSoloRobot/fsverify/id-system 三源互证）→
// PlChoosing + store 0x02 存档载入（回执 case 2 推进进场帧簇）。
func (s *Server) clientChooseRole(pl *Player, body []byte) bool {
	end := 0x42
	if len(body) < end {
		end = len(body)
	}
	name := wideNulNameOf(body[:end])
	if name == "" {
		return false
	}
	if !s.names.FindName(name) || s.names.GetDeleted(name) {
		s.trace("(SoloWorld::PlayerStateLogged)choose role unknown %q", name)
		return false
	}
	if q := s.players.FindByName(name); q != nil && q != pl {
		s.trace("(SoloWorld::PlayerStateLogged)player already in game")
		return false
	}
	s.players.SetupPlayerName(pl.Conn1, pl.Conn2, name)
	s.players.SetPlayerState(pl, PlChoosing)
	frame := buildLoadRoleStore(pl.Conn1, pl.Conn2, uint32(s.cfg.ServerID), pl.Account, name)
	ok := s.inner.SendToAllStore(0, 0x02, frame)
	s.trace("load role sent(%v) name=%q client=%d", ok, name, pl.Conn1)
	return true
}

// clientCreateRole 载荷：{name, config...}（parseCreateRoleBody 双读法）
// → PlCreating + store 0x0A 建角请求（store_forw_create_t ≥0xD4，solo
// 断言 212）。0x06 回执经 storeCreate 推进（下行有角 0x04）。
func (s *Server) clientCreateRole(pl *Player, body []byte) bool {
	name, appearance := parseCreateRoleBody(body)
	if name == "" {
		return false
	}
	if pl.Account == "" { // 未走 op02 绑定：建角会落空账号（重连竞态）
		s.trace("(SoloWorld::PlayerStateLogged)create role without account client=%d", pl.Conn1)
		return false
	}
	if len([]rune(name)) > s.cfg.RoleNameLimit {
		return false
	}
	if s.names.FindName(name) || s.players.FindByName(name) != nil {
		return false // 名字占用
	}
	s.players.SetPlayerState(pl, PlCreating)
	s.pendingCreate[pl.Conn1] = createReq{name: name, appearance: appearance}
	frame := buildCreateRoleStore(pl.Conn1, pl.Conn2, uint32(s.cfg.ServerID),
		pl.Account, name, joinAppearance(appearance))
	ok := s.inner.SendToAllStore(0, 0x0A, frame)
	s.trace("create role sent(%v) name=%q client=%d", ok, name, pl.Conn1)
	return true
}

// clientDeleteRole 载荷（真机方言未实证，双读防御）：{uid arch-str} /
// 裸窄串扫 NUL / 宽串角色名——先按 uid 反查名册、再按名字查。→
// PlDeleting + store 0x0B 删角请求（store_forw_delete_t ==0xAC，与
// create 同头 + uid@0x71）。0x07 回执经 storeDelete 推进。
func (s *Server) clientDeleteRole(pl *Player, body []byte) bool {
	if len(body) < 4 {
		return false
	}
	var uid, name string
	if v, n, _ := readArchiveString(body); n > 4 { // 读法一：archive 串
		uid = v
	}
	// 读法二：op04 同构定长 UCS2 名字帧（0x54 方言——[0:0x42] 裸 UTF-16LE
	// 扫 NUL）——优先于裸窄串（后者会把 UCS2 字节当垃圾 uid 截断后续解析）
	{
		end := 0x42
		if len(body) < end {
			end = len(body)
		}
		if v := wideNulNameOf(body[:end]); v != "" {
			name = v
		}
	}
	if uid == "" && name == "" { // 读法三：裸窄串扫 NUL（纯 uid 形态）
		end := 0
		for end < len(body) && body[end] != 0 {
			end++
		}
		uid = string(body[:end])
	}
	if name == "" { // 读法四：archive 宽串
		if v, n, _ := readArchiveWidestr(body); n > 4 {
			name = v
		}
	}
	if uid != "" { // uid 反查优先
		if v, ok := s.names.GetName(uid); ok {
			name = v
		}
	}
	if name == "" && uid == "" {
		return false
	}
	if name == "" {
		name = uid // 名字串形态：uid 槽即角色名
		uid = ""
	}
	if v, ok := s.names.GetUid(name); ok {
		uid = v
	} else if uid == "" || s.names.GetDeleted(name) {
		s.trace("(SoloWorld::PlayerStateLogged)role unknown %q", name)
		return false
	}
	if s.names.GetDeleted(name) {
		return false
	}
	s.players.SetPlayerState(pl, PlDeleting)
	frame := buildDeleteRoleStore(pl.Conn1, pl.Conn2, uint32(s.cfg.ServerID),
		pl.Account, name, uid)
	sent := s.inner.SendToAllStore(0, 0x0B, frame)
	s.trace("delete role sent(%v) name=%q uid=%q client=%d", sent, name, uid, pl.Conn1)
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
	case 0x24: // AccountReset 回执（开机复位；簿记面）
		return true
	case 0x25: // AccountLogon 回执 → 下行 0x04/0x02
		return s.accountLogonReply(payload)
	case 0x26: // AccountLogoff 回执
		return s.accountLogoffReply(payload)
	case 0: // RoleInfo：NameUid 全量同步（SyncRoleInfo 源）
		return s.storeRoleInfo(id1, payload)
	case 1: // RoleOffline 回执（§四 0x01；簿记面）
		return true
	case 2: // Load 回执：存档载入完成 → 进场帧簇（0x09/0x0A/0x0B/0x0D）
		return s.storeLoadEnter(id1, payload)
	case 3: // Save 回执（0x04 存档应答；簿记面）
		return s.storeSaveReply(id1, payload)
	case 0x0D: // QueryAccount 回执：账号角色清单 → 下行有角 0x04
		return s.storeRolesList(id1, payload)
	case 6: // Create/Roles 共用回执：建角应答
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
	// 回环极速竞态：0x00 回执可早于本实体 Unknown→LoadRoles 的 setState
	// 落地（同进程内微秒级往返）——两态皆认，早到即直迁。
	if s.state == WsLoadRoles || s.state == WsUnknown {
		s.setState(WsLoadScenes)
		s.syncRoleInfoToMembers() // 名册就绪 → 全量推 member（+0x43C 闩源）
		s.pushRoleInfoToPublic()  // 同步推 public（+0x176 nameuids 门源）
	}
	return true
}

// buildRoleInfoForPublic public 版 RoleInfo（type10/0x0A，0x34B 定长帧）：
// {int@0, int@4=1, count@8, 项 0x28B{wstr name[0x20], char uid[8]}×count}
// ——fssolopublic.worldRoleInfo 按定长 0x34 验帧（空名册也须 0x34B），
// 项区最多 1 项，不足补零。
func (s *Server) buildRoleInfoForPublic() []byte {
	snap := s.names.Snapshot()
	b := make([]byte, 0x34) // 定长帧：0xC 头 + 1×0x28 项槽
	binary.LittleEndian.PutUint32(b[0:], 0)
	binary.LittleEndian.PutUint32(b[4:], 1)
	binary.LittleEndian.PutUint32(b[8:], uint32(len(snap)))
	if len(snap) > 1 {
		snap = snap[:1]
	}
	for i, nd := range snap {
		ent := b[0xC+i*0x28 : 0xC+(i+1)*0x28]
		u := encodeUTF16([]rune(nd.Name))
		for j, c := range u {
			if j >= 0x1F {
				break
			}
			binary.LittleEndian.PutUint16(ent[j*2:], uint16(c))
		}
		copy(ent[0x20:], nd.Uid)
	}
	return b
}

// pushRoleInfoToPublic RoleInfo(type10) 全量推 public。
func (s *Server) pushRoleInfoToPublic() {
	s.inner.SendToPublic(0, 0x0A, s.buildRoleInfoForPublic())
}

// buildRoleInfoPayload CommandRoleInfo(0x0B) 载荷：{count,{wstr name,
// str uid,u8 deleted}}——与 store RoleInfo 回执同构（member 端
// worldRoleInfo 装载序）。
func (s *Server) buildRoleInfoPayload() []byte {
	snap := s.names.Snapshot()
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(len(snap)))
	for _, nd := range snap {
		u := encodeUTF16([]rune(nd.Name))
		u = append(u, 0)
		b = binary.LittleEndian.AppendUint32(b, uint32(len(u)))
		for _, c := range u {
			b = binary.LittleEndian.AppendUint16(b, uint16(c))
		}
		ub := append([]byte(nd.Uid), 0)
		b = binary.LittleEndian.AppendUint32(b, uint32(len(ub)))
		b = append(b, ub...)
		if nd.Deleted {
			b = append(b, 1)
		} else {
			b = append(b, 0)
		}
	}
	return b
}

// storeRolesList：QueryAccount 回执 0x0D {client_id, serial, count@8,
// {uid, name wchar, para wchar}...} → PlGetting + 下行有角 0x04（重登
// 角色列表；无角色时登录链已发无角 0x04，不再重发）。
func (s *Server) storeRolesList(id1 uint32, p []byte) bool {
	if len(p) < 12 {
		return false
	}
	clientID := binary.LittleEndian.Uint32(p[0:])
	serial := binary.LittleEndian.Uint32(p[4:])
	n := int(binary.LittleEndian.Uint32(p[8:]))
	rest := p[12:]
	pl := s.players.FindByConnect(clientID, serial)
	if pl == nil {
		return true
	}
	s.players.SetPlayerState(pl, PlGetting)
	var roles []roleEntry
	for i := 0; i < n; i++ {
		_, ur, r1 := readArchiveString(rest)
		name, nr, r2 := readArchiveWidestr(r1)
		if ur == 0 || nr == 0 || name == "" {
			break
		}
		para, _, r3 := readArchiveWidestr(r2)
		roles = append(roles, roleEntry{name: name, appearance: splitAppearance(para)})
		rest = r3
	}
	// 无条件下行：有角→角色列表；空→无角 0x04 基座（count=0）——
	// 删角后 0x10 刷新依赖空清单也回帧（客户端界面回建角页）
	s.trace("roles list client=%d count=%d", clientID, len(roles))
	s.sendToClient(pl, rolesFrame(time.Now(), roles))
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
	pl := s.players.FindByConnect(clientID, serial)
	if uid == "" || name == "" {
		// 建角被拒（重名/超数竞态）：回无角 0x04 让客户端回建角界面
		s.trace("(SoloWorld::StoreMsgCreate)create role denied client=%d", clientID)
		delete(s.pendingCreate, clientID)
		if pl != nil {
			s.sendToClient(pl, rolesFrame(time.Now(), nil))
		}
		return true
	}
	if !s.names.Add(name, uid, false) {
		s.trace("(SoloWorld::StoreMsgCreate)add name uid failed")
		return false
	}
	// 官方：角色新增广播 member/public/auction 三端 + 统计上报
	s.broadcastRoleChange(name, uid, true)
	if pl != nil {
		s.players.SetupPlayerName(clientID, serial, name)
		s.players.SetPlayerState(pl, PlLogged)
		req := s.pendingCreate[clientID]
		delete(s.pendingCreate, clientID)
		s.sendToClient(pl, rolesFrame(time.Now(),
			[]roleEntry{{name: name, appearance: req.appearance}}))
	}
	return true
}

// storeDelete：store_back_delete_t {client, serial, arch-str uid,
// arch-wstr name}（与 0x06 同构）。成功 → 名册 Remove + 三端角色删除
// 广播 + PlLogged + 重发 0x10 清单（0x0D 驱动界面回有角/无角 0x04）；
// 拒（uid 空）→ 仅回 PlLogged。
func (s *Server) storeDelete(id1 uint32, p []byte) bool {
	if len(p) < 8 {
		return false
	}
	clientID := binary.LittleEndian.Uint32(p[0:])
	serial := binary.LittleEndian.Uint32(p[4:])
	pl := s.players.FindByConnect(clientID, serial)
	uid, _, r := readArchiveString(p[8:])
	name, _, _ := readArchiveWidestr(r)
	if uid == "" || name == "" {
		s.trace("(SoloWorld::StoreMsgDelete)delete role denied client=%d", clientID)
		if pl != nil {
			s.players.SetPlayerState(pl, PlLogged)
		}
		return true
	}
	s.names.Remove(name, uid)
	// 官方：角色删除广播 member/public/auction 三端 + 统计上报
	s.broadcastRoleChange(name, uid, false)
	if pl != nil {
		if pl.Name == name {
			s.players.SetupPlayerName(clientID, serial, "") // 清绑定（同名可再建）
		}
		s.players.SetPlayerState(pl, PlLogged)
	}
	// 清单刷新（0x10 → 0x0D → 有角/无角 0x04）
	req := make([]byte, 72)
	binary.LittleEndian.PutUint32(req[0:], clientID)
	binary.LittleEndian.PutUint32(req[4:], serial)
	if pl != nil {
		copy(req[8:], pl.Account)
	}
	s.inner.SendToAllStore(0, 0x10, req)
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

// readArchiveString CLoadArchive::ReadString：u32 长度 + 字节。尾 NUL
// 统一滤除（store 侧 appendArchStr 方言长度含 NUL——uid char[33] 等
// 名册键不得带 NUL，否则 FindName/Remove 永远失配）。
func readArchiveString(p []byte) (val string, n int, rest []byte) {
	if len(p) < 4 {
		return "", 0, p
	}
	l := int(binary.LittleEndian.Uint32(p[0:]))
	if l < 0 || 4+l > len(p) {
		return "", 0, p
	}
	return string(bytes.TrimRight(p[4:4+l], "\x00")), 4 + l, p[4+l:]
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
	// 尾 NUL 统一滤除（appendArchWide 码元数含 NUL 同源）
	for len(u16) > 0 && u16[len(u16)-1] == 0 {
		u16 = u16[:len(u16)-1]
	}
	return string(decodeUTF16(u16)), 4 + l*2, p[4+l*2:]
}
