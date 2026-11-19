package fssolomember

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fsmember"
	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// SceneRelay 是场景宿主承接面（官方 CGameScene::ProcessMsg 命令命名空间，
// §六；对象内核不复刻，框架层把场景命令透传给宿主）。
type SceneRelay interface {
	// SceneCommand 收到发往 sceneID 的场景命令（type 0x11/0x21../0x40../
	// 透传型）。返回 false 表示场景不存在（官方 SendToScene 静默丢弃）。
	SceneCommand(sceneID, typ uint32, payload []byte) bool
}

// Options Server 装配项。
type Options struct {
	Config Config
	// Logf 日志钩（官方 WriteLine/CORE_TRACE 呈现；可空）。
	Logf func(format string, args ...any)
	// Scenes 场景宿主（可空 = 场景命令仅入表观测）。
	Scenes SceneRelay
	// ManualTick true 时不启动 1s 定时器（测试手动 Tick）。
	ManualTick bool
	// OnExit id2=0xFFFE / 状态机 Closed 后的退出钩（官方 ICore 关停）。
	OnExit func()
}

// Server 复刻 SoloMember 实体（§二/§三）。
type Server struct {
	opt Options
	cfg Config

	r     *reactor.Reactor
	inner *InnerLink

	mu       sync.Mutex
	broadID  int
	state    memberState
	frameStr string // AppRecord frame 呈现位
	tickCnt  int

	// Loading 三闩（§七）
	pubReady   bool        // +0x434（PublicSpaces/PublicData 置）
	roleReady  bool        // +0x43C（CommandRoleInfo 置）
	manageData *ManageInfo // +0x440/+0x444

	// 注册表镜像（复用集群形态既有复刻）
	Scenes  *fsmember.SceneRegistry
	Players *fsmember.PlayerRegistry
	Pub     *fsmember.PubSpaces

	// 存档块表（store 面）
	chunks map[uint32]bool

	stopOnce sync.Once
	ticker   *time.Ticker
	tickDone chan struct{}
}

// New 创建 Server（ctor 缺省全对齐 DefaultConfig）。
func New(opt Options) *Server {
	if opt.Logf == nil {
		opt.Logf = func(string, ...any) {}
	}
	if opt.Config.InnerAddr == "" {
		opt.Config = DefaultConfig()
	}
	s := &Server{
		opt:      opt,
		cfg:      opt.Config,
		Scenes:   fsmember.NewSceneRegistry(),
		Players:  fsmember.NewPlayerRegistry(),
		Pub:      fsmember.NewPublicSpaces(),
		chunks:   map[uint32]bool{},
		tickDone: make(chan struct{}),
	}
	// 先建休眠链路（状态机/发送面在未 Start 时可安全调用——测试直驱）
	s.inner = NewInnerLink(nil, InnerEvents{
		OnMessage:         s.processInnerMsg,
		OnPublicConnected: s.sendMemberRegister,
		OnExit:            opt.OnExit,
	}, fmt.Sprintf("member_%d_%d", opt.Config.ServerID, opt.Config.MemberID), s.SelfComm(), opt.Logf)
	return s
}

// SelfComm 自身 comm ident（0x18+MemberID）。
func (s *Server) SelfComm() uint32 { return uint32(commMemberMin + s.cfg.MemberID) }

// Start 官方 SoloMember::Start 装配序（§三；mutex 单实例/AppRecord/
// LogShareMem 呈现等价不复刻）。
func (s *Server) Start() error {
	s.r = reactor.New()

	if err := s.r.Start(); err != nil {
		return fmt.Errorf("(SoloMember::Start)start reactor failed: %w", err)
	}
	s.inner.Attach(s.r)
	broadID, err := s.r.CreateBroadcast(s.cfg.InnerAddr, s.cfg.BroadPort, s.onBroadcast)
	if err != nil {
		return fmt.Errorf("(SoloMember::Start)create broadcast failed: %w", err)
	}
	s.mu.Lock()
	s.broadID = broadID
	s.mu.Unlock()

	port, err := s.inner.Listen(s.cfg.InnerAddr, s.cfg.InnerPort)
	if err != nil {
		return fmt.Errorf("(SoloMember::Start)create inner listener failed: %w", err)
	}
	if s.cfg.InnerPort == 0 {
		s.cfg.InnerPort = port // 官方 GetListenerPort 回填
	}
	s.opt.Logf("inner socket %s:%d", s.cfg.InnerAddr, s.cfg.InnerPort)

	if !s.opt.ManualTick {
		s.ticker = time.NewTicker(time.Second)
		go func() {
			defer close(s.tickDone)
			for range s.ticker.C {
				s.Tick()
			}
		}()
	}
	s.opt.Logf("game server %d member %d started", s.cfg.ServerID, s.cfg.MemberID)
	return nil
}

// Stop 官方 SoloMember::Stop 停机序（§三）。
func (s *Server) Stop() {
	s.stopOnce.Do(func() {
		if s.ticker != nil {
			s.ticker.Stop()
			// ticker.Stop 不关 channel（for range 永不返回），超时兜底
			// 防从 tick 协程内重入/泄漏阻塞（fssoloworld.Stop 同范式）
			select {
			case <-s.tickDone:
			case <-time.After(time.Second):
			}
		}
		if s.r != nil {
			s.r.Stop()
		}
	})
}

// Tick 官方 TimerCheck（1s，@0x10295600）：AppRecord 行（6s 节流）→
// BroadcastMemberInfo → MemberStateMachine(0,0,0)。手动测试可直调。
func (s *Server) Tick() {
	s.mu.Lock()
	s.tickCnt++
	needInfo := s.tickCnt > 5
	if needInfo {
		s.tickCnt = 0
	}
	s.mu.Unlock()
	if needInfo {
		s.opt.Logf(`<info state="%s" frame="%s" tick="%d" />`, s.StateName(), s.frameStr, nowTick())
	}
	s.broadcastMemberInfo()
	s.StateMachine(0, 0, nil)
}

// broadcastMemberInfo 官方 BroadcastMemberInfo（每 TimerCheck 一拍）。
func (s *Server) broadcastMemberInfo() {
	s.mu.Lock()
	id := s.broadID
	s.mu.Unlock()
	if id < 0 {
		return
	}
	s.r.SendBroadcast(id, buildMemberAnnounce(s.cfg.InnerAddr, uint32(s.cfg.InnerPort),
		s.cfg.DistrictID, s.cfg.ServerID, s.cfg.MemberID))
}

// onBroadcast 官方 on_broadcast（@0x10290E70）：len≥8 ∧ magic==0x3001 ∧
// district/server 双门后按 kind 外连（§四）。
func (s *Server) onBroadcast(data []byte, fromAddr string, fromPort int) {
	info := parseAnnounce(data)
	if info == nil {
		return
	}
	if info.district != s.cfg.DistrictID || info.server != s.cfg.ServerID {
		return
	}
	switch info.kind {
	case KindPublic:
		s.inner.CheckPublicConnect(info.ip, int(info.port))
	case KindStore:
		s.inner.CheckStoreConnect(info.comm, info.ip, int(info.port))
	case KindEntry:
		s.inner.CheckEntryConnect(int(info.entryIndex), info.ip, int(info.port))
	case KindAuction:
		s.inner.CheckAuctionConnect(info.ip, int(info.port))
	}
}

// sendMemberRegister 官方 on_connect：对端 comm==1（public）→
// SendToPublic(0, 0, {DistrictId,ServerId}, 8)（@0x10291320）。
func (s *Server) sendMemberRegister() {
	if !s.inner.SendToPublic(0, 0, encodeRegisterPayload(s.cfg.DistrictID, s.cfg.ServerID)) {
		s.opt.Logf("(on_inner_connect)send member register failed")
	}
}

// processInnerMsg 官方 ProcessInnerMsg 路由（@0x10295270，§五）。
func (s *Server) processInnerMsg(src, id1, id2 uint32, payload []byte) {
	switch {
	case src == commWorld:
		s.processWorldMsg(id1, id2, payload)
	case src == commPublic:
		s.processPublicMsg(id1, id2, payload)
	case src == commAuction:
		s.processAuctionMsg(id1, id2, payload)
	case src >= commEntryMin && src < commEntryMax:
		s.sendToScene(id1, sceneCmdClientMessage, payload)
	case src >= commStoreMin && src < commStoreMax:
		if cmd, ok := storeID2ToSceneCmd[id2]; ok {
			s.sendToScene(id1, cmd, payload)
			return
		}
		if storeID2Local(id2) {
			s.processStoreMsg(id1, id2, payload)
			return
		}
		s.sendToScene(id1, id2, payload) // 其余 store id2 原样透传
	default:
		s.opt.Logf("(SoloMember::ProcessInnerMsg)message source error! comm:%d", src)
	}
}

// processAuctionMsg 官方 ProcessAuctionMsg（0xD→AuctionMessage 场景中继；
// 0xE→AuctionCommand；其余断言）。
func (s *Server) processAuctionMsg(id1, id2 uint32, payload []byte) {
	switch id2 {
	case 0xD:
		s.sendToScene(id1, 0xD, payload)
	case 0xE:
		s.sendToScene(id1, 0xE, payload)
	default:
		s.opt.Logf("(SoloMember::ProcessAuctionMsg)assert: id2=%d", id2)
	}
}

// sendToScene 官方 SendToScene（@0x10295080）：查场景 → CGameScene::
// ProcessMsg；无场景 false。
func (s *Server) sendToScene(sceneID, typ uint32, payload []byte) bool {
	if s.opt.Scenes != nil {
		return s.opt.Scenes.SceneCommand(sceneID, typ, payload)
	}
	return false
}

// SendToClient 官方 SoloMember::SendToClient（§八）：serial 高 16 位=
// EntryIdx；头 {serial,type}(8B)+data → SendToEntry2(idx, 0, 6, …)。
func (s *Server) SendToClient(serial, typ uint32, data []byte) bool {
	hdr := make([]byte, 8+len(data))
	binary.LittleEndian.PutUint32(hdr[0:], serial)
	binary.LittleEndian.PutUint32(hdr[4:], typ)
	copy(hdr[8:], data)
	if !s.inner.SendToEntry(int((serial>>16)&0xFFFF), 0, forwSendClient, hdr) {
		s.opt.Logf("(SoloMember::SendToClient)send to entry failed")
		return false
	}
	return true
}

// SendClientBroadcast 官方 SendClientBroadcast：SendToAllEntry(0, 8, data)。
func (s *Server) SendClientBroadcast(data []byte) bool {
	if !s.inner.SendToAllEntry(0, forwBroadcast, data) {
		s.opt.Logf("(SoloMember::SendClientBroadcast)send to entry failed")
		return false
	}
	return true
}

// SendToChannel 官方 SendToChannel：{channel}+data → SendToAllEntry2(0, 13, …)。
func (s *Server) SendToChannel(channel uint32, data []byte) bool {
	payload := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint32(payload[0:], channel)
	copy(payload[4:], data)
	if !s.inner.SendToAllEntry(0, forwSendChannel, payload) {
		s.opt.Logf("(SoloMember::SendToChannel)send to entry failed")
		return false
	}
	return true
}

// StateName 官方 GetState（@0x1029EB50 五名跳表）。
func (s *Server) StateName() string { return s.state.name() }

// InnerPort 实际内网口（0 口侦听回填后；官方 GetListenerPort 语义）。
func (s *Server) InnerPort() int { return s.cfg.InnerPort }

func nowTick() uint32 { return uint32(time.Now().UnixMilli()) }
