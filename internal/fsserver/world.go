package fsserver

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"
)

// WorldState 世界状态机 13 态（state@this+0x74，§三）。
type WorldState int32

const (
	WStateUnknown WorldState = iota
	WStateLoadRoles
	WStateLoadScenes
	WStateCanOpen
	WStateOpened
	WStateCloseEntry
	WStateClearPlayers
	WStateClearScenes
	WStateCloseMember
	WStateClosePublic
	WStateCloseStore
	WStateClosed
	WStateError
)

// String 状态名（官方处理器名尾词，trace 同款）。
func (s WorldState) String() string {
	switch s {
	case WStateUnknown:
		return "Unknown"
	case WStateLoadRoles:
		return "LoadRoles"
	case WStateLoadScenes:
		return "LoadScenes"
	case WStateCanOpen:
		return "CanOpen"
	case WStateOpened:
		return "Opened"
	case WStateCloseEntry:
		return "CloseEntry"
	case WStateClearPlayers:
		return "ClearPlayers"
	case WStateClearScenes:
		return "ClearScenes"
	case WStateCloseMember:
		return "CloseMember"
	case WStateClosePublic:
		return "ClosePublic"
	case WStateCloseStore:
		return "CloseStore"
	case WStateClosed:
		return "Closed"
	case WStateError:
		return "Error"
	}
	return "WorldState(?)"
}

// msg 队列条目（六队列同构）。
type msg struct {
	ident   uint32
	typ     uint32
	payload any // []byte 或 charge.VarList（charge 队列文本形态）
}

// World 复刻 GameWorld 服务本体（§一/§二）。
type World struct {
	mu sync.Mutex // 六队列与状态机串行（DoWait 单线程语义）

	DistrictID           uint32
	ServerID             uint32
	ServerName           string // wchar 属性
	MaxPlayers           int    // ctor 缺省 0x800；main.lua 默认 8192
	SaveMinutes          int
	AccountRoles         int
	ClearPlayerBatchSize int
	ClearSceneBatchSize  int

	State WorldState

	Scenes  *Scenes
	Players *Players
	Names   *NameUid
	Matrix  *MatrixCode
	IC      *InterConnect

	// 六队列（DoWait 排空序：inner → store → entry → member → public → charge）
	qInner, qStore, qEntry, qMember, qPublic, qCharge []msg

	// 登录排队（PlayerQueue 门）
	queue []*Player

	// 场景分配游标（GetFirstSceneId 轮转）
	firstScene int

	// SendToClient 客户端帧下发（entry_forw type 7 组装在 SendToClient 方法）。
	// 由装配层注入：SendToEntry2 语义。
	SendToEntryFn func(entryIdx, dst, typ uint32, payload []byte) error

	// Logf 日志面（CORE_TRACE 语义，断言文案逐字保留在调用点）。
	Logf func(format string, args ...any)

	// 关闭完成回调（WorldStateClosed 落位后触发一次）。
	OnClosed func()

	matrixTick  int // DoWait 1s 计数（>0x1E 门 → IncMatrixPos）
	chargeTick  int
	closedFired bool
}

// NewWorld 缺省构造（ctor 0x10079CA0 常量）。
func NewWorld() *World {
	return &World{
		MaxPlayers:           0x800,
		SaveMinutes:          10,
		AccountRoles:         1,
		ClearPlayerBatchSize: 100,
		ClearSceneBatchSize:  10,
		State:                WStateUnknown,
		Scenes:               NewScenes(),
		Players:              NewPlayers(),
		Names:                NewNameUid(),
		Matrix:               NewMatrixCode(),
	}
}

func (w *World) logf(format string, args ...any) {
	if w.Logf != nil {
		w.Logf(format, args...)
	}
}

// ---- 受面入口（CAdapterHook::ReceiveMsg 源路由四分支 + charge 独立口）----

// PutInnerMessage …（inner 面：注册帧外的 gmcc/stat/manage/extra 直达消息
// 与运维事件，经 ProcessInnerMsg 分派）。
func (w *World) PutInnerMessage(ident, typ uint32, payload []byte) {
	w.mu.Lock()
	w.qInner = append(w.qInner, msg{ident, typ, payload})
	w.mu.Unlock()
}

// PutStoreMessage …
func (w *World) PutStoreMessage(ident, typ uint32, payload []byte) {
	w.mu.Lock()
	w.qStore = append(w.qStore, msg{ident, typ, payload})
	w.mu.Unlock()
}

// PutEntryMessage …
func (w *World) PutEntryMessage(ident, typ uint32, payload []byte) {
	w.mu.Lock()
	w.qEntry = append(w.qEntry, msg{ident, typ, payload})
	w.mu.Unlock()
}

// PutMemberMessage …
func (w *World) PutMemberMessage(ident, typ uint32, payload []byte) {
	w.mu.Lock()
	w.qMember = append(w.qMember, msg{ident, typ, payload})
	w.mu.Unlock()
}

// PutPublicMessage …
func (w *World) PutPublicMessage(ident, typ uint32, payload []byte) {
	w.mu.Lock()
	w.qPublic = append(w.qPublic, msg{ident, typ, payload})
	w.mu.Unlock()
}

// PutChargeMessage …（CChargeSockHook::OnReceive 文本行入队）。
func (w *World) PutChargeMessage(ident, typ uint32, payload []byte) {
	w.mu.Lock()
	w.qCharge = append(w.qCharge, msg{ident, typ, payload})
	w.mu.Unlock()
}

// bytes 取载荷字节形态（charge 队列为文本 VarList，另走类型断言）。
func (m msg) bytes() []byte {
	b, _ := m.payload.([]byte)
	return b
}

// Pump 复刻 DoWait 单帧：六队列排空 + CheckQueue。
// 不含 1s tick（TickSeconds 承担——保持泵纯碎便于测试）。
func (w *World) Pump() {
	for {
		w.mu.Lock()
		if len(w.qInner) == 0 {
			w.mu.Unlock()
			break
		}
		m := w.qInner[0]
		w.qInner = w.qInner[1:]
		w.mu.Unlock()
		w.processInnerMsg(m)
	}
	w.drain(&w.qStore, w.processStoreMsg)
	w.drain(&w.qEntry, w.processEntryMsg)
	w.drain(&w.qMember, w.processMemberMsg)
	w.drain(&w.qPublic, w.processPublicMsg)
	w.drain(&w.qCharge, w.processChargeMsg)
	w.checkQueue()
}

func (w *World) drain(q *[]msg, fn func(m msg)) {
	for {
		w.mu.Lock()
		if len(*q) == 0 {
			w.mu.Unlock()
			return
		}
		m := (*q)[0]
		*q = (*q)[1:]
		w.mu.Unlock()
		fn(m)
	}
}

// TickSeconds 复刻 DoWait 1000ms 分支：矩阵卡轮转 + 世界状态机 + 广播。
// （charge/gmcc/stat/extra 保活由装配层的连接器承担。）
func (w *World) TickSeconds() {
	w.mu.Lock()
	w.matrixTick++
	w.chargeTick++
	if w.matrixTick > 0x1E {
		w.Matrix.IncMatrixPos()
	}
	w.mu.Unlock()
	w.WorldStateMachine(0, 0, nil)
}

// ---- 世界状态机（§三）----

// SetWorldState 迁移 + trace + 再驱动（官方 handler 迁移后递归
// WorldStateMachine 的语义；链式状态仅在 Unknown..Closed 前向单程，
// 无环）。
func (w *World) SetWorldState(s WorldState) {
	if w.State == s {
		return
	}
	w.logf("world state %s -> %s", w.State, s)
	w.State = s
	if s == WStateClosed && !w.closedFired {
		w.closedFired = true
		if w.OnClosed != nil {
			w.OnClosed()
		}
	}
	w.WorldStateMachine(0, 0, nil)
}

// GetState …
func (w *World) GetState() WorldState { return w.State }

// WorldStateMachine 事件驱动（事件参数按官方 (a,b,c) 占位，Go 侧
// 以内部条件推进——应答由各消息面回调进入）。
func (w *World) WorldStateMachine(_, _ uint32, _ []byte) {
	switch w.State {
	case WStateUnknown:
		w.worldStateUnknown()
	case WStateLoadRoles:
		w.worldStateLoadRoles()
	case WStateLoadScenes:
		w.worldStateLoadScenes()
	case WStateCanOpen:
		w.worldStateCanOpen()
	case WStateOpened:
		w.checkAllScene()
	case WStateCloseEntry:
		w.worldStateCloseEntry()
	case WStateClearPlayers:
		w.worldStateClearPlayers()
	case WStateClearScenes:
		w.worldStateClearScenes()
	case WStateCloseMember:
		w.worldStateCloseMember()
	case WStateClosePublic:
		w.worldStateClosePublic()
	case WStateCloseStore:
		w.worldStateCloseStore()
	}
}

// worldStateUnknown 启动：向 store 索取角色名全量（type 0，8B）。
func (w *World) worldStateUnknown() {
	var b [8]byte
	if err := w.IC.SendToStore(0, 0, StoreRoleInfoReq, b[:]); err != nil {
		w.logf("(CInterConnect::SendMsg)not connected: %v", err)
	}
	// 应答 StoreMsgRoleInfo → SetWorldState(LoadRoles)
}

// worldStateLoadRoles 角色名就绪 → 场景装载。
func (w *World) worldStateLoadRoles() { w.SetWorldState(WStateLoadScenes) }

// worldStateLoadScenes 全场景 Starting 下发（member 建场景）。
func (w *World) worldStateLoadScenes() {
	w.checkAllScene()
	w.sendOpenConnect()
	w.SetWorldState(WStateCanOpen)
}

// worldStateCanOpen …
func (w *World) worldStateCanOpen() {
	w.sendOpenConnect()
	w.checkAllScene()
	w.SetWorldState(WStateOpened)
	w.logf("Game server ready")
}

// checkAllScene 场景状态巡检：未建场景补发 CreateScene。
func (w *World) checkAllScene() {
	w.Scenes.EachNormal(func(s *Scene) bool {
		if s.State == SSceneUnknown {
			w.sendSceneCreate(s)
		}
		return true
	})
}

// sendSceneCreate world→member CreateScene（type 0：{sceneID, kind, …}）。
func (w *World) sendSceneCreate(s *Scene) {
	b := make([]byte, 0x10)
	binary.LittleEndian.PutUint32(b[0:], s.ID)
	binary.LittleEndian.PutUint32(b[4:], s.Kind)
	// +8/+0xC：脚本/配置索引位（官方 WriteInt×4 + WriteString×2 的
	// 前四 int 段；字符串段由 SyncAllScene 全量路径承载）
	if err := w.IC.SendToMember(s.MemberIdx, 0, MemberCreateScene, b); err != nil {
		w.logf("(GameWorld::CheckAllScene)send create scene %d: %v", s.ID, err)
	}
	s.State = SSceneStarting
}

// SendOpenConnect 全 entry 开放旗标（entry_forw type 0）。
func (w *World) sendOpenConnect() {
	open := uint32(0)
	if w.State == WStateOpened {
		open = 1
	}
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], open)
	_ = w.IC.SendToAllEntry(0, ForwInitConnect, b[:])
}

// SendCloseEntry/Member/Public/Store 关服序四连（§三 5..10 态轮询重发）。
func (w *World) sendCloseEntry() {
	_ = w.IC.SendToAllEntry(0, ForwInitConnect, []byte{0, 0, 0, 0})
}

// worldStateCloseEntry …
func (w *World) worldStateCloseEntry() {
	w.sendCloseEntry()
	// entry_back_close_entry_t 应答计数在 processEntryMsg 维护；
	// 全部 entry 应答（或无 entry 在线）即推进。
	if w.entriesClosed() {
		w.logf("Entry server close all")
		w.SetWorldState(WStateClearPlayers)
	}
}

// worldStateClearPlayers 批量驱动玩家离场（ClearPlayerBatchSize/帧）。
func (w *World) worldStateClearPlayers() {
	n := 0
	w.Players.Each(func(p *Player) bool {
		if n >= w.ClearPlayerBatchSize {
			return false
		}
		n++
		w.PlayerStateMachine(p, EventTick, nil, 0)
		return true
	})
	if w.Players.Count() == 0 {
		w.SetWorldState(WStateClearScenes)
	}
}

// worldStateClearScenes 副本场景清扫 + member 关闭下发。
func (w *World) worldStateClearScenes() {
	left := false
	w.Scenes.EachClone(func(s *Scene) bool {
		if s.PlayerCnt > 0 {
			left = true
		}
		return false // EachClone 早退：仅探测
	})
	if !left {
		w.logf("Member server close all")
		w.SetWorldState(WStateCloseMember)
	}
}

// worldStateCloseMember …
func (w *World) worldStateCloseMember() {
	// member_back_close_member_t 应答计数推进；无 member 在线直达
	w.logf("Member server close succeed")
	w.SetWorldState(WStateClosePublic)
}

// worldStateClosePublic …
func (w *World) worldStateClosePublic() {
	w.logf("Public server close succeed")
	w.SetWorldState(WStateCloseStore)
}

// worldStateCloseStore …
func (w *World) worldStateCloseStore() {
	w.logf("Store server close succeed")
	w.logf("Game server close succeed")
	w.SetWorldState(WStateClosed)
}

// ---- 登录排队（§四 Logging/Queuing）----

func (w *World) onlineCount() int {
	n := 0
	w.Players.Each(func(p *Player) bool {
		if p.InGame {
			n++
		}
		return true
	})
	return n
}

// OnlineCount 在线数（echo "players" 面同款）。
func (w *World) OnlineCount() int { return w.onlineCount() }

// GetQueueCount …
func (w *World) GetQueueCount() int { return len(w.queue) }

// AddToQueue …（队尾）。
func (w *World) AddToQueue(p *Player) { w.queue = append(w.queue, p) }

// RemoveFromQueue …
func (w *World) RemoveFromQueue(p *Player) {
	for i, q := range w.queue {
		if q == p {
			w.queue = append(w.queue[:i], w.queue[i+1:]...)
			return
		}
	}
}

// GetInQueuePos 队列位置（1 起）。
func (w *World) GetInQueuePos(p *Player) int {
	for i, q := range w.queue {
		if q == p {
			return i + 1
		}
	}
	return 0
}

// checkQueue 放行门：在线 < Max 时队首放行到 Getting。
func (w *World) checkQueue() {
	for len(w.queue) > 0 && w.onlineCount() < w.MaxPlayers {
		p := w.queue[0]
		w.queue = w.queue[1:]
		w.setPlayerState(p, PStateGetting)
		w.sendPlayerRoles(p)
	}
}

// ClearQueue …（WorldStateClearPlayers 联动）。
func (w *World) ClearQueue() { w.queue = nil }

// ---- 场景计数（IncScenePlayers/DecScenePlayers）----

// IncScenePlayers …
func (w *World) IncScenePlayers(id uint32) {
	if s := w.Scenes.GetScene(id); s != nil {
		s.PlayerCnt++
	}
}

// DecScenePlayers …
func (w *World) DecScenePlayers(id uint32) {
	if s := w.Scenes.GetScene(id); s != nil && s.PlayerCnt > 0 {
		s.PlayerCnt--
	}
}

// GetFirstSceneId 轮转选场景（进场分配）。
func (w *World) GetFirstSceneId() (uint32, bool) {
	var ids []uint32
	w.Scenes.EachNormal(func(s *Scene) bool {
		if s.Kind == 0 {
			ids = append(ids, s.ID)
		}
		return true
	})
	if len(ids) == 0 {
		return 0, false
	}
	id := ids[w.firstScene%len(ids)]
	w.firstScene++
	return id, true
}

// ---- 玩家状态机（§四）----

func (w *World) setPlayerState(p *Player, s PlayerState) {
	if p.State == s {
		return
	}
	w.logf("player %08X:%08X %s -> %s", p.Ident, p.Serial, p.State, s)
	p.State = s
}

// PlayerStateMachine(player, msgID, data, len)：14 态 × 事件路由。
func (w *World) PlayerStateMachine(p *Player, event uint32, data []byte, _ int) {
	switch p.State {
	case PStateUnknown:
		w.playerStateUnknown(p, event, data)
	case PStateWaitBreak:
		w.breakClient(p)
		w.Players.RemovePlayer(p.Ident, p.Serial)
	case PStateLogging:
		w.playerStateLogging(p, event, data)
	case PStateValidate:
		w.playerStateValidate(p, event, data)
	case PStateQueuing:
		w.playerStateQueuing(p, event, data)
	case PStateGetting:
		// 等待角色列表应答（store Roles）；断线即摘除
		if event == EventClosed {
			w.disconnectPlayer(p)
		}
	case PStateLogged:
		w.playerStateLogged(p, event, data)
	case PStateChoosing:
		w.playerStateChoosing(p, event, data)
	case PStateCreating:
		if event == EventClosed {
			w.disconnectPlayer(p)
		}
	case PStateDeleting:
		if event == EventClosed {
			w.disconnectPlayer(p)
		}
	case PStateOnline:
		w.playerStateOnline(p, event, data)
	case PStateOffline:
		w.playerStateOffline(p)
	case PStateSaving:
		w.playerStateOffline(p)
	case PStateLeaving:
		w.breakClient(p)
		w.Players.RemovePlayer(p.Ident, p.Serial)
	}
}

// playerStateUnknown 登录主态。
func (w *World) playerStateUnknown(p *Player, event uint32, data []byte) {
	if event == EventClosed {
		w.disconnectPlayer(p)
		return
	}
	if event != EventLogin {
		return
	}
	acct, pass, err := parseClientLogin(data)
	if err != nil || acct == "" || pass == "" {
		w.logf("(GameWorld::PlayerStateUnknown)get account password error")
		w.sendErrClient(p, ErrLoginParse)
		w.breakClient(p)
		w.Players.RemovePlayer(p.Ident, p.Serial)
		return
	}
	// 满员 / 重复登录
	if q := w.Players.FindByAccount(acct); q != nil && q != p {
		w.logf("(GameWorld::ClientMsgLogin)account repeat login")
		w.sendErrClient(p, ErrRepeatLogin)
		return
	}
	if w.onlineCount() >= w.MaxPlayers {
		w.sendErrClient(p, ErrServerFull)
		return
	}
	w.Players.SetPlayerAccount(p, acct)
	p.Password = pass
	w.setPlayerState(p, PStateLogging)
	w.PlayerStateMachine(p, EventTick, nil, 0)
}

// playerStateLogging …
func (w *World) playerStateLogging(p *Player, event uint32, _ []byte) {
	if event == EventClosed {
		w.disconnectPlayer(p)
		return
	}
	// 账号下线（账号 logon 应答通过 store charge 链路，此处直达取角列表）
	w.sendToStorage(0, StoreAccountLogon, w.buildAccountLogon(p))
	w.setPlayerState(p, PStateGetting)
	w.sendPlayerRoles(p)
}

// playerStateValidate 矩阵卡校验等待（挑战已发）。
func (w *World) playerStateValidate(p *Player, event uint32, data []byte) {
	if event == EventClosed {
		w.disconnectPlayer(p)
		return
	}
	if event != EventGetVerify {
		return
	}
	// 客户端挑战应答核对（md5(password+challenge) 由 charge 链承担；
	// Go 侧简化：非空应答即通过 → Logging）
	if len(data) > 0 {
		w.setPlayerState(p, PStateLogging)
		w.PlayerStateMachine(p, EventTick, nil, 0)
	}
}

// playerStateQueuing …
func (w *World) playerStateQueuing(p *Player, event uint32, _ []byte) {
	if event == EventClosed {
		w.RemoveFromQueue(p)
		w.disconnectPlayer(p)
		return
	}
	if pos := w.GetInQueuePos(p); pos == 0 {
		w.logf("(GameWorld::PlayerStateQueuing)queue error")
		w.disconnectPlayer(p)
	}
}

// playerStateLogged 选角/建角/删角。
func (w *World) playerStateLogged(p *Player, event uint32, data []byte) {
	switch event {
	case EventClosed:
		w.disconnectPlayer(p)
	case EventChooseRole:
		w.chooseRole(p, data)
	case EventCreateRole:
		w.createRole(p, data)
	case EventDeleteRole:
		w.deleteRole(p, data)
	}
}

// chooseRole 选角（§四 Logged 校验链）。
func (w *World) chooseRole(p *Player, data []byte) {
	role, err := parseClientChooseRole(data)
	if err != nil {
		w.logf("(GameWorld::PlayerStateLogged)player role invalid")
		w.sendErrClient(p, ErrLoginParse)
		return
	}
	if q := w.Players.FindByName(role); q != nil && q != p {
		w.logf("(GameWorld::PlayerStateLogged)player already in game")
		w.sendErrClient(p, ErrRepeatLogin)
		return
	}
	sceneID, ok := w.GetFirstSceneId()
	if !ok {
		w.logf("(GameWorld::PlayerStateLogged)no any scene")
		w.sendErrClient(p, ErrServerFull)
		return
	}
	s := w.Scenes.GetScene(sceneID)
	if s == nil || s.State != SSceneRunning {
		w.logf("(GameWorld::PlayerStateLogged)scene busy")
		w.sendErrClient(p, ErrServerFull)
		return
	}
	p.RoleName = role
	p.SceneID = sceneID
	w.setPlayerState(p, PStateChoosing)
	// 角色数据请求（store LoadPlayer 45B 语义；Go 侧直发场景装载）
	w.sendToStorage(0, StoreLoadPlayer, w.buildLoadPlayer(p.Account, role))
}

// playerStateChoosing 载入完成（store Load / public LoadPlayer 应答进入）。
func (w *World) playerStateChoosing(p *Player, event uint32, _ []byte) {
	if event == EventClosed {
		w.disconnectPlayer(p)
		return
	}
	// 进场：PlayerAdd 全 member 广播 + entry set_online
	w.Players.SetupPlayerName(p, p.RoleName)
	p.InGame = true
	w.IncScenePlayers(p.SceneID)
	w.sendToAllMembers(0, MemberPlayerAdd, w.buildPlayerAdd(p))
	w.sendSetOnline(p)
	w.setPlayerState(p, PStateOnline)
}

// createRole 建角（store Create 请求 → Creating；应答回 Logged 重发列表）。
func (w *World) createRole(p *Player, data []byte) {
	role, para, err := parseClientCreateRole(data)
	if err != nil || role == "" {
		w.logf("(GameWorld::PlayerStateLogged)error name length")
		return
	}
	if !w.Names.Add(role, p.Account, 0) {
		w.logf("(GameWorld::StoreMsgCreate)add name uid failed")
		return
	}
	w.setPlayerState(p, PStateCreating)
	w.sendToAllStores(0, StoreCreateRole, w.buildCreateRole(p.Account, role, para))
}

// deleteRole 删角。
func (w *World) deleteRole(p *Player, data []byte) {
	role, err := parseClientChooseRole(data)
	if err != nil {
		return
	}
	w.setPlayerState(p, PStateDeleting)
	w.sendToAllStores(0, StoreCreateRole, w.buildDeleteRole(p.Account, role)) // 类型由 storeface Create/Delete 分流
}

// playerStateOnline 在线态。
func (w *World) playerStateOnline(p *Player, event uint32, data []byte) {
	switch event {
	case EventClosed:
		w.setPlayerState(p, PStateOffline)
		w.PlayerStateMachine(p, EventTick, nil, 0)
	case EventTick:
		// 续费检查（renew warning/break 由 charge 链驱动；超时离线）
		if time.Since(p.ActiveTime) > time.Duration(w.SaveMinutes)*time.Minute*60 {
			w.setPlayerState(p, PStateOffline)
			w.PlayerStateMachine(p, EventTick, nil, 0)
		}
	default:
		// 客户端帧透传场景（CommandClientMessage 语义）
		if len(data) > 0 {
			w.sendToScene(p.SceneID, 0x10, data)
		}
	}
}

// playerStateOffline 离线善后（§四 Offline 全链）。
func (w *World) playerStateOffline(p *Player) {
	w.sendPlayerSetOffline(p)
	p.InGame = false
	w.DecScenePlayers(p.SceneID)
	w.Players.CleanPlayerName(p)
	w.sendToAllMembers(0, MemberPlayerRemove, w.buildPlayerRemove(p))
	w.savePlayer(p)
	w.Players.RemovePlayer(p.Ident, p.Serial)
}

// disconnectPlayer 非在线态断线（ErasePlayer 轻路径）。
func (w *World) disconnectPlayer(p *Player) {
	w.RemoveFromQueue(p)
	w.Players.RemovePlayer(p.Ident, p.Serial)
}

// ---- 客户端面（entry_forw 组装）----

func (w *World) sendErrClient(p *Player, code uint32) {
	w.SendToClient(p, ErrFrame(code))
}

// SendToClient entry_forw type 7：{ident,serial}8B + 帧。
func (w *World) SendToClient(p *Player, frame []byte) error {
	b := make([]byte, 8+len(frame))
	binary.LittleEndian.PutUint32(b[0:], p.Ident)
	binary.LittleEndian.PutUint32(b[4:], p.Serial)
	copy(b[8:], frame)
	return w.sendToEntry(p.EntryIdx, ForwSendClient, b)
}

// breakClient type 1 踢线。
func (w *World) breakClient(p *Player) {
	var b [8]byte
	binary.LittleEndian.PutUint32(b[0:], p.Ident)
	binary.LittleEndian.PutUint32(b[4:], p.Serial)
	_ = w.sendToEntry(p.EntryIdx, ForwBreakClient, b[:])
}

// shakeClient type 10 保活。
func (w *World) shakeClient(p *Player) {
	var b [8]byte
	binary.LittleEndian.PutUint32(b[0:], p.Ident)
	binary.LittleEndian.PutUint32(b[4:], p.Serial)
	_ = w.sendToEntry(p.EntryIdx, ForwShakeClient, b[:])
}

// sendSetOnline type 3 {ident,serial,memberID,memberIdent}。
func (w *World) sendSetOnline(p *Player) {
	s := w.Scenes.GetScene(p.SceneID)
	if s == nil {
		return
	}
	b := make([]byte, 16)
	binary.LittleEndian.PutUint32(b[0:], p.Ident)
	binary.LittleEndian.PutUint32(b[4:], p.Serial)
	binary.LittleEndian.PutUint32(b[8:], s.MemberIdx)
	binary.LittleEndian.PutUint32(b[12:], MemberIdent(s.MemberIdx))
	_ = w.sendToEntry(p.EntryIdx, ForwSetOnline, b)
}

// sendPlayerRoles 客户端 0x04（§4.2 布局）。
func (w *World) sendPlayerRoles(p *Player) {
	roles := decodeRoleBlob(p.RoleBlob)
	now := time.Now()
	out := NewStoreWriter(0x100)
	out.PutByte(0x04)
	out.WriteInt(int32(p.Ident))
	out.WriteInt(int32(p.Serial))
	out.WriteInt(int32(now.Year()))
	out.WriteInt(int32(now.Month()))
	out.WriteInt(int32(now.Day()))
	out.WriteInt(int32(now.Hour()))
	out.WriteInt(int32(now.Minute()))
	out.WriteInt(int32(now.Second()))
	out.WriteInt(int32(p.Serial))
	out.WriteInt(int32(len(roles)))
	out.Seek(0x29)
	for _, r := range roles {
		out.WriteWord(4)
		out.PutByte(VaryTagInt)
		out.WriteInt(r.A)
		out.PutByte(VaryTagInt)
		out.WriteInt(r.B)
		out.PutByte(VaryTagWideStr)
		out.WriteWideStr(r.Name)
		out.PutByte(VaryTagWideStr)
		out.WriteWideStr(r.Para)
	}
	_ = w.SendToClient(p, out.Bytes())
}

// RoleEntry 角色条目（store Roles blob 单项）。
type RoleEntry struct {
	A, B       int32
	Name, Para string
}

// decodeRoleBlob 解 store Roles 应答（Seek(0x10) 起 per-role：i32/i32/wstr/wstr）。
func decodeRoleBlob(blob []byte) []RoleEntry {
	if len(blob) < 0x10 {
		return nil
	}
	r := NewLoadReader(blob)
	r.Seek(0x10)
	var out []RoleEntry
	for r.Remaining() >= 12 {
		a, err1 := r.ReadInt()
		b, err2 := r.ReadInt()
		name, err3 := r.ReadWideStr()
		para, err4 := r.ReadWideStr()
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			break
		}
		out = append(out, RoleEntry{A: a, B: b, Name: name, Para: para})
	}
	return out
}

// ---- 外发包装 ----

func (w *World) sendToEntry(idx, typ uint32, payload []byte) error {
	if w.SendToEntryFn != nil {
		return w.SendToEntryFn(idx, 0, typ, payload)
	}
	if w.IC != nil {
		return w.IC.SendToEntry(idx, 0, typ, payload)
	}
	return fmt.Errorf("(CInterConnect::SendMsg)not connected")
}

func (w *World) sendToStorage(dst, typ uint32, payload []byte) {
	if w.IC != nil {
		_ = w.IC.SendToStore(0, dst, typ, payload)
	}
}

func (w *World) sendToAllStores(dst, typ uint32, payload []byte) {
	if w.IC != nil {
		_ = w.IC.SendToAllStore(dst, typ, payload)
	}
}

func (w *World) sendToAllMembers(dst, typ uint32, payload []byte) {
	if w.IC != nil {
		_ = w.IC.SendToAllMember(dst, typ, payload)
	}
}

// sendToScene 场景路由（§2.3：场景须 Running）。
func (w *World) sendToScene(sceneID, typ uint32, payload []byte) bool {
	s := w.Scenes.GetScene(sceneID)
	if s == nil || s.State != SSceneRunning {
		return false
	}
	if w.IC != nil {
		return w.IC.SendToMember(s.MemberIdx, sceneID, typ, payload) == nil
	}
	return false
}

// savePlayer 存档请求（SendToScene type 0x15 语义 + Offline 落库）。
func (w *World) savePlayer(p *Player) {
	if w.Names != nil {
		// 存档落库由 store 后端承担（接缝）；此处仅保序
	}
}

// buildAccountLogon 143B 语义（窄串定长域按官方槽位填）。
func (w *World) buildAccountLogon(p *Player) []byte {
	out := NewStoreWriter(0x8F)
	out.Seek(4)
	out.WriteString(p.Account)
	out.WriteString(p.Password)
	return out.Bytes()[:0x8F]
}

// buildLoadPlayer 45B 语义。
func (w *World) buildLoadPlayer(account, role string) []byte {
	out := NewStoreWriter(0x2D)
	out.Seek(4)
	out.WriteString(account)
	out.WriteWideStr(role)
	b := out.Bytes()
	if len(b) < 0x2D {
		b = append(b, make([]byte, 0x2D-len(b))...)
	}
	return b[:0x2D]
}

// buildCreateRole …
func (w *World) buildCreateRole(account, role, para string) []byte {
	out := NewStoreWriter(0x9B)
	out.Seek(4)
	out.WriteString(account)
	out.WriteWideStr(role)
	out.WriteWideStr(para)
	return out.Bytes()
}

// buildDeleteRole …
func (w *World) buildDeleteRole(account, role string) []byte {
	out := NewStoreWriter(0x40)
	out.Seek(4)
	out.WriteString(account)
	out.WriteWideStr(role)
	return out.Bytes()
}

// buildPlayerAdd world→member PlayerAdd（{scene,ident,serial,name wchar[36],account}）。
func (w *World) buildPlayerAdd(p *Player) []byte {
	out := NewStoreWriter(0xA8)
	out.Seek(4)
	out.WriteInt(int32(p.SceneID))
	out.WriteInt(int32(p.Ident))
	out.WriteInt(int32(p.Serial))
	units := []rune(p.RoleName)
	for i := 0; i < 36; i++ {
		if i < len(units) {
			out.WriteWord(uint16(units[i]))
		} else {
			out.WriteWord(0)
		}
	}
	out.WriteString(p.Account)
	return out.Bytes()
}

// buildPlayerRemove {ident,serial} …
func (w *World) buildPlayerRemove(p *Player) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b[0:], p.Ident)
	binary.LittleEndian.PutUint32(b[4:], p.Serial)
	return b
}

// sendPlayerSetOffline store type 6 …
func (w *World) sendPlayerSetOffline(p *Player) {
	w.sendToStorage(0, StoreSetOffline, w.buildAccountLogon(p))
}

// entriesClosed 关服序 entry 应答计数（assembly 注入；缺省全关即真）。
func (w *World) entriesClosed() bool { return true }

// ---- 客户端帧解析（world 层方言，§4.2）----

// parseClientLogin 登录帧：[0x15=21B 头][u32 len][account][u32 len][password]。
func parseClientLogin(frame []byte) (acct, pass string, err error) {
	if len(frame) < 0x15 {
		return "", "", fmt.Errorf("message size error")
	}
	r := NewLoadReader(frame)
	r.Seek(0x15)
	if acct, err = r.ReadString(); err != nil {
		return "", "", err
	}
	if pass, err = r.ReadString(); err != nil {
		return "", "", err
	}
	return acct, pass, nil
}

// parseClientChooseRole 选角/删角帧：跳 21B 头读宽串角色名。
func parseClientChooseRole(frame []byte) (string, error) {
	if len(frame) < 0x15 {
		return "", fmt.Errorf("message size error")
	}
	r := NewLoadReader(frame)
	r.Seek(0x15)
	return r.ReadWideStr()
}

// parseClientCreateRole 建角帧：角色名 + 外观参数串。
func parseClientCreateRole(frame []byte) (role, para string, err error) {
	if len(frame) < 0x15 {
		return "", "", fmt.Errorf("message size error")
	}
	r := NewLoadReader(frame)
	r.Seek(0x15)
	if role, err = r.ReadWideStr(); err != nil {
		return "", "", err
	}
	if para, err = r.ReadWideStr(); err != nil {
		return "", "", err
	}
	return role, para, nil
}
