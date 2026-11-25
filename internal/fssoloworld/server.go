package fssoloworld

import (
	"encoding/binary"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// Options Server 装配项。
type Options struct {
	Config Config
	// AppName 覆盖 cfg.AppName()（空=缺省 "world_<sid>"）。合一形态
	// 官方 main.lua 置 world.AppName = "unite_"..server_id。
	AppName string
	// Logf 日志钩（官方 WriteLine/CORE_TRACE 呈现）。
	Logf func(format string, args ...any)
	// ManualTick true 时不启动 1s 定时器（测试手动 Tick）。
	ManualTick bool
	// OnClosed Closed 态 + QuitOnClosed 后的退出钩（官方 nx_quit）。
	OnClosed func()
	// Scenes scenes.ini 装配面（可空；Start 前注入）。
	Scenes []SceneSpec
}

// AppNameOf 本实体对广播自宣的 app name（Options.AppName > 缺省）。
func (o Options) AppNameOf() string {
	if o.AppName != "" {
		return o.AppName
	}
	return o.Config.AppName()
}

// SceneSpec AddScene/AddPrototypeScene 装配参数（main.lua load_scene_config）。
type SceneSpec struct {
	Index    int
	Member   int
	Script   string
	Config   string
	Clonable bool
}

// chargeAgent charge 代理观测面（复用 internal/charge 由宿主接线；此处仅
// 连接态供 PlayerStateLogging 回退判定）。
type chargeAgent interface {
	Connected() bool
}

// Server 复刻 SoloWorld 实体（§二/§三/§七）。
type Server struct {
	opt    Options
	cfg    Config
	r      *reactor.Reactor
	inner  *InnerLink
	echo   *EchoConsole
	charge chargeAgent

	mu        sync.Mutex
	state     WorldState
	session   uint32 // 本实例会话号（dc_accounts.logged 域；开机复位门）
	broadID   int
	timerID   int
	innerPort int
	tickCnt   int
	frameStr  string // AppRecord 心跳呈现

	players       *WorldPlayers
	scenes        *WorldScenes
	names         *NameUid
	pendingCreate map[uint32]createReq
	roleLocks     map[string]time.Time // +0x244 角色锁表（200s 自动解锁）
	accountLocks  map[string]time.Time // +0x254 账号锁表

	stopOnce sync.Once
	ticker   *time.Ticker
	tickDone chan struct{}

	watchStop chan struct{} // 发现监督器停口（startDiscoveryWatch 建）
	watchOnce sync.Once

	// 战斗接入层（combat.go）：事件队列与泵、Lua 宿主懒建、对象注册表。
	combatQ    combatQueue
	pumpDone   chan struct{}
	luaOnce    sync.Once
	lua        *combatLua
	combatMu   sync.Mutex
	fighters   map[uint64]*npcFighter // NPC 战斗体（按句柄）
	aux        map[uint64]*auxObject  // 技能/buff 辅助对象
	nextHandle uint64
}

// New 创建 Server（ctor 缺省对齐 DefaultConfig）。
func New(opt Options) *Server {
	if opt.Logf == nil {
		opt.Logf = func(string, ...any) {}
	}
	if opt.Config.InnerAddr == "" {
		opt.Config = DefaultConfig()
	}
	s := &Server{
		opt:           opt,
		cfg:           opt.Config,
		players:       NewWorldPlayers(opt.Config.MaxPlayers),
		scenes:        NewWorldScenes(),
		names:         NewNameUid(),
		pendingCreate: map[uint32]createReq{},
		roleLocks:     map[string]time.Time{},
		accountLocks:  map[string]time.Time{},
		tickDone:      make(chan struct{}),
		pumpDone:      make(chan struct{}),
		fighters:      map[uint64]*npcFighter{},
		aux:           map[uint64]*auxObject{},
		session:       uint32(time.Now().Unix()), // 本实例会话号（dc_accounts.logged 域）
	}
	for _, sc := range opt.Scenes {
		typ := 0
		if sc.Clonable {
			typ = 1
		}
		s.scenes.AddScene(sc.Index, sc.Member, typ, sc.Script, sc.Config)
	}
	return s
}

// trace 官方 WriteLine/CORE_TRACE 呈现。
func (s *Server) trace(format string, args ...any) {
	s.opt.Logf("(world)"+format, args...)
}

// State 当前状态名（GetState 语义）。
func (s *Server) State() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.String()
}

// setState SetWorldState（+0x164）。
func (s *Server) setState(st WorldState) {
	s.mu.Lock()
	old := s.state
	if st <= old {
		// 只前进不倒退（13 态单调序：装载链与关闭链均为递增编号）：
		// 早到回执已直迁后，迟到的低态 setState 不得把状态打回。
		s.mu.Unlock()
		return
	}
	s.state = st
	s.mu.Unlock()
	if old != st {
		s.trace("world state %s -> %s", old, st)
	}
}

// InnerPort 实际内网端口（Start 后有效；0=未启）。
func (s *Server) InnerPort() int { return s.innerPort }

// Inner 公开 InnerLink（宿主接线 SendTo* 用）。
func (s *Server) Inner() *InnerLink { return s.inner }

// Players / Scenes / Names 公开数据面。
func (s *Server) Players() *WorldPlayers { return s.players }
func (s *Server) Scenes() *WorldScenes   { return s.scenes }
func (s *Server) Names() *NameUid        { return s.names }

// SetChargeAgent 注入 charge 代理观测面。
func (s *Server) SetChargeAgent(c chargeAgent) { s.charge = c }

// Start 官方 SoloWorld::Start（§三装配序）。
func (s *Server) Start() error {
	s.r = reactor.New()
	s.r.Start()

	s.inner = NewInnerLink(s.r, InnerEvents{
		OnMessage:    s.onInnerMessage,
		OnConnected:  s.onInnerConnected,
		OnDisconnect: s.onInnerDisconnect,
	}, s.opt.AppNameOf(), s.opt.Logf)

	// 广播受发面
	bid, err := s.r.CreateBroadcast(s.cfg.BroadAddr, s.cfg.BroadPort, s.onBroadcast)
	if err != nil {
		return err
	}
	s.broadID = bid

	// 内网受面（InnerPort==0 → 分配回填）
	port, err := s.inner.Listen(s.cfg.InnerAddr, s.cfg.InnerPort)
	if err != nil {
		return err
	}
	s.innerPort = port

	// echo 维护口
	if err := s.StartEcho(s.cfg.EchoAddr, s.cfg.EchoPort); err != nil {
		s.trace("(SoloWorld::Start)start echo failed: %v", err)
	}

	if !s.opt.ManualTick {
		s.ticker = time.NewTicker(time.Second)
		go s.tickLoop()
		if combatEnabled() {
			go s.combatPumpLoop()
		}
	}
	s.startDiscoveryWatch()
	s.trace("game server %d starting...", s.cfg.ServerID)
	s.setState(WsUnknown)
	return nil
}

// Stop 官方 SoloWorld::Stop + Shut。
func (s *Server) Stop() {
	s.stopOnce.Do(func() {
		s.watchOnce.Do(func() {
			if s.watchStop != nil {
				close(s.watchStop)
			}
		})
		if s.ticker != nil {
			s.ticker.Stop()
			select {
			case <-s.tickDone:
			case <-time.After(time.Second):
			}
		}
		if s.echo != nil {
			s.echo.Stop()
		}
		if s.r != nil {
			s.r.Stop()
		}
	})
}

// Tick 手动驱动一次 TimerCheck（ManualTick 模式）。
func (s *Server) Tick() { s.timerCheck() }

func (s *Server) tickLoop() {
	defer close(s.tickDone)
	for range s.ticker.C {
		s.timerCheck()
	}
}

// timerCheck 官方 TimerCheck（§四九步）。
func (s *Server) timerCheck() {
	s.mu.Lock()
	s.tickCnt++
	s.mu.Unlock()

	// 排队放行（CheckQueue：满员判定随下线自然回退）
	s.checkQueue()

	// 锁超时（≈200s，天单位 0.002315）
	now := time.Now()
	for name, t := range s.roleLocks {
		if now.Sub(t) > 200*time.Second {
			delete(s.roleLocks, name)
			s.trace("(SoloWorld::TimerCheck)role %s auto unlocked", name)
		}
	}
	for acc, t := range s.accountLocks {
		if now.Sub(t) > 200*time.Second {
			delete(s.accountLocks, acc)
			s.trace("(SoloWorld::TimerCheck)account %s auto unlocked", acc)
		}
	}

	// AppRecord 心跳呈现（XML info）
	s.frameStr = s.appRecordInfo()

	// 广播自宣
	s.broadcastWorldInfo()

	// 世界状态机驱动
	s.worldStateMachine()

	// NPC 巡逻广播（0x21 批量 + State 翻转）
	s.npcPatrolTick(time.Now())

	// 战斗会话每秒驱动（MP 回复、招架耐力回复；skill.go）+ NPC 主动索敌
	// （combatnpc.go）
	s.combatTick()
	s.npcAggroTick()
}

// checkQueue CheckQueue/CheckVipQueue：排队者放行进 Logging。
func (s *Server) checkQueue() {
	for _, p := range s.players.Snapshot() {
		if p.State == PlQueuing && s.players.CountActive() < s.cfg.MaxPlayers {
			s.players.SetPlayerState(p, PlLogging)
		}
	}
}

// appRecordInfo `<info state=.. scenes=.. clones=.. player=.. .../>`。
func (s *Server) appRecordInfo() string {
	return "<info state=\"" + s.State() + "\" scenes=\"" +
		itoa(s.scenes.CountNormal()) + "\" clones=\"" +
		itoa(s.scenes.CountClone()) + "\" player=\"" +
		itoa(s.players.Count()) + "\" online=\"" +
		itoa(s.players.CountOnline()) + "\" offline=\"" +
		itoa(s.players.CountOffline()) + "\" active=\"" +
		itoa(s.players.CountActive()) + "\" queue=\"" +
		itoa(s.players.CountQueue()) + "\"/>"
}

// AppRecordInfo 心跳快照（测试/宿主呈现）。
func (s *Server) AppRecordInfo() string { return s.frameStr }

// broadcastWorldInfo 官方自宣帧（§五 0x84B）。
func (s *Server) broadcastWorldInfo() {
	if s.r == nil || s.broadID < 0 {
		return
	}
	s.mu.Lock()
	opened := s.state == WsOpened
	s.mu.Unlock()
	pkt := buildWorldAnnounce(s.cfg.InnerAddr, uint32(s.innerPort),
		uint32(s.cfg.DistrictID), uint32(s.cfg.ServerID), s.cfg.ServerName,
		s.cfg.MaxPlayers, 0, opened, uint32(s.players.CountOnline()))
	s.r.SendBroadcast(s.broadID, pkt)
}

// onBroadcast 广播接收五分支（§五接收表）。
func (s *Server) onBroadcast(data []byte, fromAddr string, fromPort int) {
	h, ok := parseAnnounceHead(data)
	if !ok {
		return
	}
	if h.district != uint32(s.cfg.DistrictID) || h.server != uint32(s.cfg.ServerID) {
		return // 同区同服判定
	}
	if s.inner == nil {
		return
	}
	switch h.kind {
	case KindPublic:
		s.inner.CheckPublicConnect(h.addr, int(h.port))
	case KindStore:
		s.inner.CheckStoreConnect(int(h.idx), h.addr, int(h.port))
	case KindMember:
		if h.idx >= memberNumMax {
			return
		}
		s.inner.CheckMemberConnect(int(h.idx), h.addr, int(h.port))
	case KindEntry:
		if h.idx >= entryIndexMax {
			return
		}
		s.inner.CheckEntryConnect(int(h.idx), h.addr, int(h.port))
	case KindAuction:
		s.inner.CheckAuctionConnect(h.addr, int(h.port))
	}
}

// onInnerConnected 五类连接事件（InnerXXXConnected）。
func (s *Server) onInnerConnected(comm uint32) {
	switch {
	case comm == commPublic:
		s.trace("public connected")
		// 名册已装载而 public 后连：补推 RoleInfo(type10)（+0x176 门源）
		if s.rolesLoaded() {
			s.pushRoleInfoToPublic()
		}
	case comm == commAuction:
		s.trace("auction connected")
	case comm >= commStoreMin && comm < commStoreMax:
		s.trace("store %d connected", comm-commStoreMin)
	case comm >= commEntryMin && comm < commEntryMax:
		s.trace("entry %d connected", comm-commEntryMin)
	case comm >= commMemberMin && comm < commMemberMax:
		idx := int(comm - commMemberMin)
		s.trace("member %d connected", idx)
		// 名册已装载而 member 后连：补推 CommandRoleInfo（+0x43C 闩源）
		if s.rolesLoaded() {
			s.inner.SendToMember(idx, 0, 0x0B, s.buildRoleInfoPayload())
		}
	}
}

// sendSceneCreate CommandCreateScene(0x00) 下发：{scene u32, type u32,
// name cstr}（member 端 worldCreateScene 装载序；name=Script 逻辑类名）。
func (s *Server) sendSceneCreate(index, member, typ int, name string) bool {
	b := make([]byte, 8+len(name)+1)
	binary.LittleEndian.PutUint32(b[0:], uint32(index))
	binary.LittleEndian.PutUint32(b[4:], uint32(typ))
	copy(b[8:], name)
	return s.inner.SendToMember(member, 0, 0x00, b)
}

// ConnectMember 宿主直连口（合一形态确定性启动）：绕过广播发现，
// 以已知地址幂等拨 member（已连则 no-op）。返回是否已在线。
func (s *Server) ConnectMember(addr string, port int) bool {
	if s.inner == nil {
		return false
	}
	s.inner.CheckMemberConnect(0, addr, port) // 幂等：已连/dialing 中即跳过
	return s.MemberConnected()
}

// MemberConnected world↔member 0 号边是否在线（宿主启动门用）。
func (s *Server) MemberConnected() bool {
	if s.inner == nil {
		return false
	}
	s.inner.mu.Lock()
	_, ok := s.inner.commConn[uint32(commMemberMin)]
	s.inner.mu.Unlock()
	return ok
}

// rolesLoaded 角色名册装载完毕（WsLoadScenes..WsOpened 运行区间）。
func (s *Server) rolesLoaded() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state >= WsLoadScenes && s.state <= WsOpened
}

// syncRoleInfoToMembers CommandRoleInfo(0x0B) 全量推所有已连 member。
func (s *Server) syncRoleInfoToMembers() {
	s.inner.SendToAllMember(0, 0x0B, s.buildRoleInfoPayload())
}

func (s *Server) onInnerDisconnect(comm uint32) {
	s.trace("comm %d disconnected", comm)
}

// onInnerMessage ProcessInnerMsg（§十分发中枢：按 comm 源路由）。
func (s *Server) onInnerMessage(comm, id1, id2 uint32, payload []byte) {
	switch {
	case comm >= commMemberMin && comm < commMemberMax:
		s.ProcessMemberMsg(id1, id2, payload)
	case comm >= commEntryMin && comm < commEntryMax:
		s.ProcessEntryMsg(id1, id2, payload)
	case comm >= commStoreMin && comm < commStoreMax:
		s.ProcessStoreMsg(id1, id2, payload)
	case comm == commPublic:
		// ProcessPublicMsg：角色增删同步对端确认（框架层记录）
	case comm == commAuction:
		// ProcessAuctionMsg：同上
	default:
		s.trace("(CSoloWorld::ProcessInnerMsg)world message source error! comm:%d", comm)
	}
}

// worldStateMachine 13 态驱动（§七）。
func (s *Server) worldStateMachine() {
	s.mu.Lock()
	st := s.state
	s.mu.Unlock()
	switch st {
	case WsUnknown:
		// SendToStore(load roles 请求) → LoadRoles
		if s.inner != nil && s.inner.SendToAllStore(0, 0, encodeLoadRoles(s.cfg.ServerID)) {
			// 0x38 AccountReset 开机复位（logged=session 清零）
			s.mu.Lock()
			session := s.session
			s.mu.Unlock()
			s.inner.SendToAllStore(0, 0x38, sessionFrame(session))
			s.setState(WsLoadRoles)
		}
	case WsLoadRoles:
		// 等 StoreMsgRoleInfo 同步（storeRoleInfo 推进）
	case WsLoadScenes:
		// SendSceneAdd（AddScene 装配后逐场景下发 member）→ 等 SceneState。
		// member 未连时 SendToMember 失败、ScUnknown 保持，下一拍重试。
		for _, sc := range s.scenes.Snapshot() {
			if sc.State != ScUnknown || sc.Member < 0 {
				continue
			}
			if s.sendSceneCreate(sc.Index, sc.Member, sc.Type, sc.Script) {
				s.scenes.SetState(sc.Index, ScStarting)
			} else {
				s.trace("send scene %d to member %d failed", sc.Index, sc.Member)
			}
		}
		if s.scenes.CountNormal() == 0 && s.scenes.CountClone() == 0 {
			s.setState(WsCanOpen)
		}
	case WsCanOpen:
		if s.cfg.AutoOpen {
			s.OpenServer()
		}
	case WsOpened:
		// CheckAllScene + CheckAllPlayer + 逐玩家驱动（省略——玩家由消息驱动）
	case WsCloseEntry:
		s.inner.SendToAllEntry(0, entryClose, nil)
		s.setState(WsClearPlayers)
	case WsClearPlayers:
		if s.players.Count() == 0 {
			s.setState(WsClearScenes)
		} else {
			s.clearPlayerBatch()
		}
	case WsClearScenes:
		if s.scenes.CountNormal()+s.scenes.CountClone() == 0 {
			s.setState(WsCloseMember)
		} else {
			s.clearSceneBatch()
		}
	case WsCloseMember:
		s.inner.SendToAllMember(0, 0x1F, nil)
		s.setState(WsClosePublic)
	case WsClosePublic:
		s.inner.SendToPublic(0, 0, nil)
		s.setState(WsCloseAuction)
	case WsCloseAuction:
		s.inner.SendToAuction(0, 0, nil)
		s.setState(WsCloseStore)
	case WsCloseStore:
		s.inner.SendToAllStore(0, 0xE, nil)
		// 等 StoreMsgCloseStore 应答或直进
		s.setState(WsClosed)
	case WsClosed:
		if s.cfg.QuitOnClosed {
			s.Stop()
			if s.opt.OnClosed != nil {
				s.opt.OnClosed()
			}
		}
	}
}

// clearPlayerBatch ClearPlayerBatchSize 批量下线。
func (s *Server) clearPlayerBatch() {
	batch := 10
	for _, p := range s.players.Snapshot() {
		if batch <= 0 {
			break
		}
		s.playerLeave(p)
		batch--
	}
}

func (s *Server) clearSceneBatch() {
	for _, sc := range s.scenes.Snapshot() {
		s.scenes.RemoveScene(sc.Index)
		break // ClearSceneBatchSize=1 官方缺省
	}
}

// playerLeave 官方离线链（Offline 态清理语义）。
func (s *Server) playerLeave(p *Player) {
	s.players.RemovePlayer(p.Conn1, p.Conn2)
}

// OpenServer 官方 OpenServer：CanOpen→Opened + SendOpenConnect(entry id2=0)。
func (s *Server) OpenServer() bool {
	s.mu.Lock()
	ready := s.state == WsCanOpen
	s.mu.Unlock()
	if !ready {
		return false
	}
	// entry_forw_init_connect_t ==8 {flag,window}：flag≠0 → entry 开闸
	// （Unknown→Opened、SetClosed(false)——客户端面自此收连）。
	ic := make([]byte, 8)
	binary.LittleEndian.PutUint32(ic[0:], 1)
	binary.LittleEndian.PutUint32(ic[4:], uint32(time.Now().Unix()))
	s.inner.SendToAllEntry(0, entryInitConnect, ic)
	s.setState(WsOpened)
	s.trace("game server opened")
	return true
}

// CloseServer 官方 CloseServer：Opened→CloseEntry（关闭链启动）。
func (s *Server) CloseServer() bool {
	s.mu.Lock()
	opened := s.state == WsOpened
	s.mu.Unlock()
	if !opened {
		return false
	}
	s.setState(WsCloseEntry)
	s.trace("waiting server close...")
	return true
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
