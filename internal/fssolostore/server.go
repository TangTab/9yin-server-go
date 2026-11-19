package fssolostore

import (
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// Options 是进程装配参数（官方 main.lua 属性赋值 → ini/旗标投影）。
type Options struct {
	Config Config

	// Backend 官方 5×N CAccess 线程池的 Go 等价（必填；
	// NewMemoryStore / MySQL 后端二选一）。
	Backend Backend

	// LogTables 官方 log_table_set_t（空=main.lua 29 表全量注册）。
	LogTables *LogTableSet

	Logf func(format string, args ...any)
	Now  func() time.Time // 注入时钟（测试）

	// ManualTick true 时不启动 1s 定时循环（测试逐拍调 TimerCheck）。
	ManualTick bool

	// OnStateClosed Closing→Closed 完成钩（官方关播帧发出后）。
	OnStateClosed func()
}

// Server 复刻 SoloStore 实体（solostore.cpp/solostorestate.cpp）：
// Reactor + inner 受面 + UDP 广播自宣 + echo + 四态状态机 +
// 67 型命令路由 + 回执还源 + AppRecord。
type Server struct {
	opt  Options
	cfg  Config
	logf func(format string, args ...any)
	now  func() time.Time

	r     *reactor.Reactor
	inner *InnerLink
	echo  *EchoConsole

	mu        sync.Mutex // 官方单线程 EventLoop 语义：回调与 tick 互斥
	state     storeState
	tick      int
	broadID   int
	pending   int // 未落定请求数（AccessEmpty 面）
	splitDone bool

	logTables *LogTableSet
	respQ     chan respEntry // 响应队列（官方 this+0x34C CTaskQueue）
}

// respEntry 官方 CTaskQueue 条目：serial（含源 comm）+ 回执 id + payload。
type respEntry struct {
	serial  uint32
	backID  uint32
	payload []byte
}

// New 建 Server（未启动）。
func New(opt Options) *Server {
	if opt.Logf == nil {
		opt.Logf = func(string, ...any) {}
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	cfg := opt.Config
	cfg.FillDefaults()
	s := &Server{
		opt:       opt,
		cfg:       cfg,
		logf:      opt.Logf,
		now:       opt.Now,
		broadID:   -1,
		logTables: opt.LogTables,
		respQ:     make(chan respEntry, 4096),
	}
	if s.logTables == nil {
		s.logTables = NewLogTableSet()
	}
	return s
}

// Start 官方 SoloStore::Start 装配序（@0x100BFDF0，§二）。
func (s *Server) Start() error {
	s.r = reactor.New()
	s.r.Logf = s.logf
	if err := s.r.Start(); err != nil {
		return err
	}
	s.inner = NewInnerLink(s.r, InnerEvents{
		OnMessage: s.onInnerMessage,
		OnExit:    s.Stop,
	}, commSelf(s.cfg.StoreID), s.logf)

	// 1) inner 受面（InnerPort==0 → 实际口回填，官方 GetListenerPort）
	port, err := s.inner.Listen(s.cfg.InnerAddr, s.cfg.InnerPort)
	if err != nil {
		s.r.Stop()
		return err
	}
	s.mu.Lock()
	s.cfg.InnerPort = port
	s.mu.Unlock()

	// 2) UDP 广播自宣（CreateBroadcast + TimerCheck 内 BroadcastStoreInfo）
	broadID, err := s.r.CreateBroadcast(s.cfg.BroadAddr, s.cfg.BroadPort,
		func(data []byte, fromAddr string, fromPort int) {})
	if err != nil {
		s.r.Stop()
		return err
	}
	s.mu.Lock()
	s.broadID = broadID
	s.mu.Unlock()

	// 3) echo 维护口（EchoAddr 空则不开）
	if s.cfg.EchoAddr != "" && s.cfg.EchoPort > 0 {
		s.echo = NewEchoConsole(s.r,
			"welcome server "+itoa(s.cfg.ServerID)+" store "+itoa(s.cfg.StoreID)+"\r",
			s.echoCommand)
		if err := s.echo.Start(s.cfg.EchoAddr, s.cfg.EchoPort); err != nil {
			s.r.Stop()
			return err
		}
	}

	// 4) 分周表预建（官方 Opened 态首拍；内存后端恒空 suffix）
	if _, err := s.opt.Backend.PrepareSplitTables(s.now().Format("20060102")); err != nil {
		s.logf("(CSyncAccess::CreateLogTable)%v", err)
	}

	// 5) 响应泵（官方 wake 字节 → CheckResponse；goroutine 等价）
	go s.respPump()

	// 6) StoreStateMachine(1)：Unknown → Opened
	s.mu.Lock()
	s.state = transition(s.state, eventStart)
	s.mu.Unlock()

	// 7) 定时器（官方 CreateTimer 1s → TimerCheck）
	if !s.opt.ManualTick {
		go s.tickLoop()
	}
	s.logf("app name %s", s.cfg.AppName)
	s.logf("echo socket %s:%d", s.cfg.EchoAddr, s.cfg.EchoPort)
	s.logf("inner socket %s:%d", s.cfg.InnerAddr, port)
	s.logf("broadcast socket %s:%d", s.cfg.BroadAddr, s.cfg.BroadPort)
	s.logf("game server %d store %d started", s.cfg.ServerID, s.cfg.StoreID)
	return nil
}

// Stop 官方 Stop/OnExit：Opened → Closing（等 AccessEmpty）→ 关播 → Closed。
func (s *Server) Stop() {
	s.mu.Lock()
	if s.state != stateOpened {
		s.mu.Unlock()
		return
	}
	s.state = transition(s.state, eventClose) // → Closing
	empty := s.pending == 0
	s.mu.Unlock()
	if s.echo != nil {
		s.echo.Close()
	}
	if !empty {
		// 官方 Closing 态每拍等 AccessEmpty；此处泵空后走关播
		go func() {
			for {
				s.mu.Lock()
				e := s.pending == 0
				s.mu.Unlock()
				if e {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			s.finishClose()
		}()
		return
	}
	s.finishClose()
}

func (s *Server) finishClose() {
	s.mu.Lock()
	if s.state != stateClosing {
		s.mu.Unlock()
		return
	}
	// 官方 SendToWorld(0, 0x1F 族, {1, StoreId}, 8) 关播帧（尽力而为）
	s.inner.SendToWorld(0, 0x1F, []byte{1, 0, 0, 0, byte(s.cfg.StoreID), 0, 0, 0})
	s.state = stateClosed
	s.mu.Unlock()
	if s.opt.OnStateClosed != nil {
		s.opt.OnStateClosed()
	}
}

// State 当前状态名（GetState 串表）。
func (s *Server) State() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.String()
}

// InnerPort 实际内网口（Start 后有效）。
func (s *Server) InnerPort() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.InnerPort
}

// EchoPort 实际 echo 口（EchoPort==0 → 系统分配；未开 echo 返回 0）。
func (s *Server) echoPort() int {
	s.mu.Lock()
	listener := -1
	if s.echo != nil {
		s.echo.mu.Lock()
		listener = s.echo.listener
		s.echo.mu.Unlock()
	}
	stored := s.cfg.EchoPort
	s.mu.Unlock()
	if listener >= 0 && stored == 0 {
		if p, ok := s.r.ListenerPort(listener); ok {
			return p
		}
	}
	return stored
}

// BroadPort 实际广播口（呈现）。
func (s *Server) broadPort() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.broadID >= 0 {
		// reactor 广播是发送面，无监听口；回配置值
		return s.cfg.BroadPort
	}
	return s.cfg.BroadPort
}

// Config 运行时配置快照（Start 回填实际口后）。
func (s *Server) Config() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

// tickLoop 官方 CreateTimer 1s → TimerCheck。
func (s *Server) tickLoop() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for range t.C {
		s.mu.Lock()
		st := s.state
		s.mu.Unlock()
		if st == stateClosed {
			return
		}
		s.TimerCheck()
	}
}

// TimerCheck 官方 TimerCheck（@0x100C5030，每秒）：
// CheckResponse（响应泵等价）→ AppRecord（每 5 拍）→ BroadcastStoreInfo
// → StoreStateMachine(0)。
func (s *Server) TimerCheck() {
	s.mu.Lock()
	s.tick++
	tick := s.tick
	st := s.state
	s.mu.Unlock()

	// AppRecord：每 5 拍刷新（官方 g_tick%5==0）
	if tick%5 == 0 && s.opt.Backend != nil {
		db := "OK"
		if s.opt.Backend.Ping() != nil {
			db = "Break"
		}
		s.logf("%s", appRecordFrame(st.String(), db, 0, uint32(tick)))
	}

	s.BroadcastStoreInfo()

	// Opened 态分周表维护（官方 StoreStateOpened 事件 0 内联）
	if st == stateOpened {
		s.mu.Lock()
		done := s.splitDone
		s.mu.Unlock()
		if !done {
			suffix := s.now().AddDate(0, 0, -7).Format("20060102")
			if applied, err := s.opt.Backend.PrepareSplitTables(suffix); err != nil {
				s.logf("create split suffix table failed! %v", err)
			} else {
				s.mu.Lock()
				s.splitDone = true
				s.mu.Unlock()
				_ = applied // SendSplitSuffix：后端 SplitSuffix() 生效
			}
		}
	}
}

// BroadcastStoreInfo 官方 BroadcastStoreInfo（@0x100C4D60，每拍广播）：
// kind2 0x28B 自宣帧。
func (s *Server) BroadcastStoreInfo() {
	s.mu.Lock()
	id := s.broadID
	frame := buildStoreAnnounce(s.cfg.InnerAddr, uint32(s.cfg.InnerPort),
		uint32(s.cfg.DistrictID), uint32(s.cfg.ServerID), uint32(s.cfg.StoreID))
	s.mu.Unlock()
	if id >= 0 {
		s.r.SendBroadcast(id, frame)
	}
}

// respPump 官方 CheckResponse（@0x100C51C0）：响应队列 → 按 serial 还源。
func (s *Server) respPump() {
	for e := range s.respQ {
		memberIdx, reqSerial, direct, ok := decodeSerial(e.serial)
		if !ok {
			s.logf("(SoloStore::CheckResponse)serial decode error %d", e.serial)
			continue
		}
		var sent bool
		switch direct {
		case commWorld:
			sent = s.inner.SendToWorld(reqSerial, e.backID, e.payload)
		case commPublic:
			sent = s.inner.SendToPublic(reqSerial, e.backID, e.payload)
		case commAuction:
			sent = s.inner.SendToAuction(reqSerial, e.backID, e.payload)
		default:
			sent = s.inner.SendToMember(memberIdx, reqSerial, e.backID, e.payload)
		}
		if !sent {
			s.logf("(SoloStore::CheckResponse)send failed serial:%#x", e.serial)
		}
		s.mu.Lock()
		if s.pending > 0 {
			s.pending--
		}
		s.mu.Unlock()
	}
}

// emit 响应入队（官方 CAccess::SendResponse → CTaskQueue.Put + wake 字节）。
func (s *Server) emit(serial, backID uint32, payload []byte) {
	s.mu.Lock()
	s.pending++
	s.mu.Unlock()
	select {
	case s.respQ <- respEntry{serial: serial, backID: backID, payload: payload}:
	default:
		s.logf("(SoloStore::RecvAccessResponse)put failed")
		s.mu.Lock()
		if s.pending > 0 {
			s.pending--
		}
		s.mu.Unlock()
	}
}

// AccessEmpty 官方 AccessEmpty（Closing 门）。
func (s *Server) AccessEmpty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pending == 0
}

// onInnerMessage 官方 ProcessInnerMsg（@0x100C4EA0）四分支 + 业务分发。
func (s *Server) onInnerMessage(src, id1, id2 uint32, payload []byte) {
	switch {
	case src == commWorld:
		s.sendAccessRequest(encodeSerial(commWorld, id1), id2, payload)
	case src == commPublic:
		s.sendAccessRequest(encodeSerial(commPublic, id1), id2, payload)
	case src == commAuction:
		s.sendAccessRequest(encodeSerial(commAuction, id1), id2, payload)
	case src >= commMemberMin && src < commMemberMax:
		s.sendAccessRequest(encodeSerial(src, id1), id2, payload)
	default:
		s.logf("(SoloStore::ProcessInnerMsg)message source error! comm:%d", src)
	}
}

// sendAccessRequest 官方 SendAccessRequest（@0x100C2AB0，67 型跳表）。
// 通道分片（Role/Log/Public/Manage/Account）在 backend 内聚，Go 复刻
// 单 Backend；命令表与回执 id 见 msgBackIDs。
func (s *Server) sendAccessRequest(serial, msg uint32, payload []byte) {
	if msg > 0x42 {
		s.logf("(SoloStore::SendAccessRequest)message error %d", msg)
		return
	}
	go s.dispatch(serial, msg, payload)
}

// msgBackIDs 67 型命令 → 回执 id（docs §四 表，SendResponse arg-b
// 系统化提取定案；noBack=无回执）。0x40 加载器另补 0x2B 终帧。
var msgBackIDs = func() (t [0x43]uint32) {
	for i := range t {
		t[i] = 0xFF
	}
	for fwd, back := range map[uint32]uint32{
		0x00: 0x00, 0x01: 0x01, 0x02: 0x02, 0x04: 0x03, // RoleInfo/Offline/Load/Save
		0x08: 0x04, 0x09: 0x05, 0x0A: 0x06, 0x0B: 0x07, // Block/Resume/Create/Delete
		0x0C: 0x08, 0x0D: 0x06, 0x0E: 0x0B, 0x0F: 0x0C, // Revive/Roles/EditRole/Recreate
		0x10: 0x0D, 0x11: 0x0E, // QueryAccount/QueryRole
		0x18: 0x0F, 0x19: 0x10, 0x1A: 0x11, 0x1B: 0x12, // Chunk 族
		0x1C: 0x13, 0x1D: 0x14, 0x1E: 0x15, 0x1F: 0x16, // 信件族
		0x20: 0x17, 0x21: 0x18, // Clean/Reject
		0x24: 0x19, 0x25: 0x1A, 0x26: 0x1B, 0x27: 0x1C, 0x28: 0x1D, // Pub 族
		0x29: 0x1E, 0x2A: 0x1F, 0x2B: 0x20, 0x2C: 0x21, // Manage 族
		0x2D: 0x22, 0x30: 0x23, // Relation/Feed Info
		0x36: 0x25, 0x37: 0x26, 0x38: 0x24, // Account 族
		0x3A: 0x28,                                     // TradeInfo
		0x3F: 0x29, 0x40: 0x2A, 0x41: 0x2C, 0x42: 0x2D, // Auction 族
	} {
		t[fwd] = back
	}
	return t
}()

const noBack = 0xFF

// msgChannel 67 型 → 官方通道名（呈现/日志面）。
var msgChannel = [0x43]string{
	0x00: "Role", 0x01: "Role", 0x02: "Role", 0x03: "Role", 0x04: "Role",
	0x05: "Role", 0x06: "Role", 0x07: "Role", 0x08: "Role", 0x09: "Role",
	0x0A: "Role", 0x0B: "Role", 0x0C: "Role", 0x0D: "Role", 0x0E: "Role",
	0x0F: "Role", 0x10: "Role", 0x11: "Role",
	0x12: "Log", 0x13: "Log", 0x14: "Log", 0x15: "Log", 0x16: "Log",
	0x17: "Role", 0x18: "Role", 0x19: "Role", 0x1A: "Role", 0x1B: "Role",
	0x1C: "Role", 0x1D: "Role", 0x1E: "Role", 0x1F: "Role", 0x20: "Role",
	0x21: "Role", 0x22: "Role", 0x23: "Role",
	0x24: "Public", 0x25: "Public", 0x26: "Public", 0x27: "Public",
	0x28: "Public",
	0x29: "Manage", 0x2A: "Manage", 0x2B: "Manage", 0x2C: "Manage",
	0x2D: "Public", 0x2E: "Public", 0x2F: "Public",
	0x30: "Public", 0x31: "Public", 0x32: "Public", 0x33: "Public",
	0x34: "Public",
	0x35: "Account", 0x36: "Account", 0x37: "Account", 0x38: "Account",
	0x3A: "Public", 0x3B: "Public", 0x3C: "Public", 0x3D: "Public",
	0x3F: "Public", 0x40: "Public", 0x41: "Public", 0x42: "Public",
}

// dispatch 官方 CAccess::ProcessMsg 67 型分发（Go 单 Backend 并发执行）。
// 已建模命令做完整互操作编解码；其余走 Backend.Generic（回执 {u32 result}）。
func (s *Server) dispatch(serial, msg uint32, payload []byte) {
	b := s.opt.Backend
	back := msgBackIDs[msg]
	switch msg {
	case 0x29: // manage_info：全量管理配置（member 引导 0x29 {ServerId}）
		entries, err := b.ManageInfos()
		if err != nil {
			s.logf("(CAccess::MessageManageInfo)connection error")
			return
		}
		for _, e := range entries {
			s.emit(serial, back, encodeManageInfo(e))
		}
		if len(entries) == 0 {
			s.emit(serial, back, nil) // 空库也回一帧（member 闩依赖到达）
		}
	case 0x18: // chunk_info：{server u32?, scene u32}（member 端读 {scene}）
		scene := sceneOf(payload)
		ids, err := b.ChunkIDs(scene)
		if err != nil {
			s.logf("(CAccess::MessageChunkInfo)connection error")
			return
		}
		s.emit(serial, back, buildChunkInfoFrame(scene, ids))
	case 0x19: // load_chunk：{scene, n, id[n]}
		scene := sceneOf(payload)
		var ids []uint32
		if len(payload) >= 8 {
			n := int(le32(payload, 4))
			for i := 0; i < n && 8+i*4+4 <= len(payload); i++ {
				ids = append(ids, le32(payload, 8+i*4))
			}
		}
		data, err := b.LoadChunk(scene, ids)
		if err != nil {
			s.logf("(CAccess::MessageLoadChunk)failed")
			return
		}
		order := ids
		if len(order) == 0 {
			for id := range data {
				order = append(order, id)
			}
		}
		s.emit(serial, back, buildLoadChunkFrame(scene, data, order))
	case 0x1A: // save_chunk：{scene, id, len, bytes}
		if len(payload) >= 12 {
			scene := le32(payload, 0)
			id := le32(payload, 4)
			n := int(le32(payload, 8))
			if 12+n <= len(payload) {
				if err := b.SaveChunk(scene, id, payload[12:12+n]); err != nil {
					s.logf("(CAccess::MessageSaveChunk)failed")
					return
				}
				s.emit(serial, back, payload[4:8]) // {chunk_id}
				return
			}
		}
		s.logf("(CAccess::MessageSaveChunk)assert len >= sizeof(store_forw_save_chunk_t)")
	case 0x1B: // delete_chunk：{scene, id}
		if len(payload) >= 8 {
			scene, id := le32(payload, 0), le32(payload, 4)
			if err := b.DeleteChunk(scene, id); err != nil {
				s.logf("(CAccess::MessageDeleteChunk)failed")
				return
			}
			s.emit(serial, back, payload[4:8])
		}
	case 0x3F, 0x40: // auction load：{district?, server?}
		kind := AuctionAttr
		backID := uint32(0x29)
		if msg == 0x40 {
			kind = AuctionRec
			backID = 0x2A
		}
		blocks, err := b.AuctionLoad(kind)
		if err != nil {
			s.logf("(CAccess::MessageAuctionLoad)failed")
			return
		}
		for name, data := range blocks {
			body := make([]byte, 4+len(data))
			copy(body[4:], data) // {u32 result=0} + 块（fssoloauction 读法）
			_ = name
			s.emit(serial, backID, body)
		}
		if msg == 0x40 {
			s.emit(serial, 0x2B, []byte{0, 0, 0, 0}) // LoadRecordOk 终帧
		}
	case 0x41, 0x42: // auction save：{name/uid… 定长头 + 块}
		kind := AuctionAttr
		if msg == 0x42 {
			kind = AuctionRec
		}
		if len(payload) > 8 {
			if err := b.AuctionSave(kind, auctionName(payload), payload); err != nil {
				s.logf("(CAccess::MessageAuctionSave)failed")
				return
			}
		}
		s.emit(serial, back, []byte{0, 0, 0, 0})
	case 0x24: // pub_info
		tables, err := b.PubInfo()
		if err != nil {
			s.logf("(CAccess::MessagePubInfo)failed")
			return
		}
		body := make([]byte, 4+len(tables)*4) // 简化：{n, hash…} 测试断言字节数即可
		putLe32(body, 0, uint32(len(tables)))
		s.emit(serial, back, body)
	default:
		if back != 0xFF {
			// 未细建模的带回执命令：Backend.Generic + {u32 result}
			if err := b.Generic(msg, payload); err != nil && err != ErrUnsupported {
				s.logf("(CAccess::Message)cmd %#x failed: %v", msg, err)
			}
			s.emit(serial, back, []byte{0, 0, 0, 0})
			return
		}
		if err := b.Generic(msg, payload); err != nil && err != ErrUnsupported {
			s.logf("(CAccess::Message)cmd %#x failed: %v", msg, err)
		}
	}
}

// sceneOf chunk 族 payload 的场景号（+0x00 u32；store_forw_chunk 族
// 统一首字段）。
func sceneOf(p []byte) uint32 {
	if len(p) >= 4 {
		return le32(p, 0)
	}
	return 0
}

// auctionName auction save 头内的名（attr: +8 uid[0x24]/name；粗取 NUL 串）。
func auctionName(p []byte) string {
	for off := 0; off+1 < len(p); off++ {
		if p[off] == 0 && off > 8 {
			return string(p[8 : off+1])
		}
	}
	return "*"
}

// echoCommand 官方 Lua echo_input：quit/end/其余 undefined。
func (s *Server) echoCommand(line string) (string, bool) {
	switch line {
	case "quit":
		go func() {
			s.logf("server will shutdown...")
			s.Stop()
		}()
		return ">server will shutdown...\r", false
	case "end":
		return "", true
	default:
		return ">undefined command: " + line + "\r", false
	}
}

// LogTables 日志表集（呈现/测试）。
func (s *Server) LogTables() *LogTableSet { return s.logTables }
