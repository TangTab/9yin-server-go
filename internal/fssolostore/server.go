package fssolostore

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
	"sync"
	"time"
	"unicode/utf16"

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
	s.startDiscoveryWatch()
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

// boolByteU8 布尔 → 单字节 0/1（archive 尾域）。
func boolByteU8(v bool) byte {
	if v {
		return 1
	}
	return 0
}

// emptyStoreBackHeader 0x22/0x23/0x28 应答公共头（result=1, count=0）——
// 空库合法形态，fssolopublic parseStoreBackHeader 互证（len≥0x10 且
// result≠0 即过门）。
func emptyStoreBackHeader() []byte {
	b := make([]byte, 0x10)
	putLe32(b, 0, 1)
	return b
}

// cstrFixed 定长域取 NUL 前 C 串。
func cstrFixed(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// wideNulStr UTF-16LE 定长槽扫 NUL 解码（建角帧 name/para 槽）。
func wideNulStr(p []byte) string {
	end := 0
	for end+1 < len(p) {
		if p[end] == 0 && p[end+1] == 0 {
			break
		}
		end += 2
	}
	u := make([]uint16, end/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(p[i*2:])
	}
	return string(utf16.Decode(u))
}

// newRoleUID 官方 CAccess::CreateGuid（char[33] 槽）：16B 随机 hex。
func newRoleUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%032x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// 建角出生场景缺省（官方口径：默认烟雨庄；testdata/location.ini
// BornCJ000003 + rolePara f24 scene_name_017 互证，world 端
// fssoloworld/roles.go 同源）。
const (
	birthSceneID       = 17
	birthSceneLocation = `ini\scene\born03_YanYuZhuang`

	// defaultFacultySeed 建号修为种子（b-line facultySeed 口径：外服私服
	// 修为由历练转化供给，新角色种子可配）。
	defaultFacultySeed = 0
)

// dispatch 官方 CAccess::ProcessMsg 67 型分发（Go 单 Backend 并发执行）。
// 已建模命令做完整互操作编解码；其余走 Backend.Generic（回执 {u32 result}）。
func (s *Server) dispatch(serial, msg uint32, payload []byte) {
	b := s.opt.Backend
	back := msgBackIDs[msg]
	switch msg {
	case 0x00: // role_info：开机在线复位 + 名册全量（world 引导 WsUnknown→LoadRoles；
		// 回执 archive {count,{wstr name,str uid,u8 deleted}}，world 端 storeRoleInfo 装载序）
		sid := int(sceneOf(payload))
		if sid == 0 {
			sid = s.cfg.ServerID
		}
		briefs, err := b.ResetOnline(sid)
		if err != nil {
			s.logf("(CAccess::MessageRoleInfo)connection error")
			return
		}
		body := appendArchInt(nil, int32(len(briefs)))
		for _, br := range briefs {
			body = appendArchWide(body, br.Name)
			body = appendArchStr(body, br.UID)
			body = append(body, boolByteU8(br.Deleted))
		}
		s.emit(serial, back, body)
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
			// LoadRecordOk 终帧：result≠0 = 装载成功（fssoloauction
			// LoadOK 按 result 开闸 Opened——发 0 会被当失败告警）
			s.emit(serial, 0x2B, []byte{1, 0, 0, 0})
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
	case 0x24: // pub_info：{space[0x24],table[0x24]} → store_back_pub_info_t
		// {space@0,table@0x24,result@0x48,count@0x4C,archive{wstr}×}
		// （fssolopublic.storePubInfo 版式互证）。
		if len(payload) < 0x48 {
			s.logf("(CAccess::MessagePubInfo)len < 0x48")
			return
		}
		space := cstrFixed(payload[0:0x24])
		table := cstrFixed(payload[0x24:0x48])
		names, err := b.PubLoad(table)
		if err != nil {
			s.logf("(CAccess::MessagePubInfo)failed")
			return
		}
		body := make([]byte, 0x50, 0x50+len(names)*4)
		copy(body[0:0x24], space)
		copy(body[0x24:0x48], table)
		putLe32(body, 0x48, 1) // result≠0
		putLe32(body, 0x4C, uint32(len(names)))
		for name := range names {
			body = appendArchWide(body, name)
		}
		s.emit(serial, back, body)
	case 0x2D: // relation_info：{ServerId} → 0x22 空表头（result≠0）
		s.emit(serial, back, emptyStoreBackHeader())
	case 0x30: // feed 存活期配置 → 0x23 空表头
		s.emit(serial, back, emptyStoreBackHeader())
	case 0x3A: // trade_info：{ServerId} → 0x28 空表头
		s.emit(serial, back, emptyStoreBackHeader())
	case 0x02: // load（solo ==0xBC）：与 create 同头 → MessageLoad
		// （SELECT 存档）→ 回执 0x02 {client,serial,u32 ok,
		// arch-wstr para}（空档=ok 1 无 blob；save_data 全量 0xD0B
		// 形态留存档里程碑——新角初建本就 NULL）。
		if len(payload) < 0x2D+2 {
			s.logf("(CAccess::MessageLoad)len == sizeof(store_forw_load_t)=0xBC，得 %#x", len(payload))
			return
		}
		_, meta, ok, err := b.LoadRole(cstrFixed(payload[0xC:0x2D]),
			wideNulStr(payload[0x2D:0x2D+0x44]), int(sceneOf(payload[8:12])))
		if err != nil {
			s.logf("(CAccess::MessageLoad)connection error")
			return
		}
		body := make([]byte, 12)
		putLe32(body, 0x00, sceneOf(payload[0:4]))
		putLe32(body, 0x04, sceneOf(payload[4:8]))
		putLe32(body, 0x08, uint32(boolByteU8(ok)))
		if ok && meta.RoleStateJSON == "" {
			// 旧档补建（RST1 机制上线的存量行 role_state_json=NULL）：
			// 按存档外观推性别即时重建并回写——否则 world 回落 golden
			// replay（2086 方言帧在 2433 客户端上 ConfigID 语义错位 →
			// 场景加载 ter0 失败）。
			if st, berr := rolestate.BuildRoleState(rolestate.DefaultBirthScene,
				rolestate.InferSexFromPara(meta.SavePara1), meta.SavePara1, defaultFacultySeed); berr == nil {
				st.Equips = rolestate.StarterEquips(st.Sex, meta.UID, time.Now().Unix())
				st.EquipsSeeded = true
				if js, jerr := rolestate.MarshalRoleState(st); jerr == nil {
					meta.RoleStateJSON = js
					_ = b.SaveRole(cstrFixed(payload[0xC:0x2D]), meta, nil)
				}
			}
		}
		if ok && meta.RoleStateJSON != "" {
			// 初始布衣补穿（equipsSeeded 闸上线前的存量档：RST1 在但
			// 建号早于 StarterEquips——按官方初始号口径穿着进场补一次；
			// 玩家主动全脱的档已有闸位不再复穿）。
			if st, uerr := rolestate.UnmarshalRoleState(meta.RoleStateJSON); uerr == nil && !st.EquipsSeeded {
				st.Equips = rolestate.StarterEquips(st.Sex, meta.UID, time.Now().Unix())
				st.EquipsSeeded = true
				if js, jerr := rolestate.MarshalRoleState(st); jerr == nil {
					meta.RoleStateJSON = js
					_ = b.SaveRole(cstrFixed(payload[0xC:0x2D]), meta, nil)
				}
			}
		}
		if ok {
			body = appendArchWide(body, meta.SavePara1)
			// Go 内网扩展尾段（官方 store 不带）：建号/运行状态 JSON。
			// magic 'RST1' 识别——world 端缺席该段时回退 replay 模式。
			if meta.RoleStateJSON != "" {
				body = binary.LittleEndian.AppendUint32(body, 0x31545352) // 'RST1'
				body = binary.LittleEndian.AppendUint32(body, uint32(len(meta.RoleStateJSON)))
				body = append(body, meta.RoleStateJSON...)
			}
			// 'ACC1' 账户尾段（GM 离线邮件 name-only 查档时 world 回写
			// 0x04 需要真实账户定位；常规链路 world 自知账户，多余无害）。
			if meta.Account != "" {
				body = binary.LittleEndian.AppendUint32(body, 0x31434341) // 'ACC1'
				body = binary.LittleEndian.AppendUint32(body, uint32(len(meta.Account)))
				body = append(body, meta.Account...)
			}
		}
		s.emit(serial, back, body)
	case 0x04: // save（gmpanel GM 链路：world 0x04 存档，官方锚点头
		// +Go TLV 段——见 fssoloworld/roles.go buildSaveStore）→
		// merge 库内 meta（world 不可知列不冲掉）→ SaveRole →
		// 回执 0x03 {client,serial,u32 result}（官方 MessageSave→0x03）。
		name, pos, tlv, ok := decodeSaveStore(payload)
		if !ok {
			s.logf("(CAccess::MessageSave)len == sizeof(store_forw_save_t)≥0xA0，得 %#x", len(payload))
			return
		}
		account := string(tlv[1])
		serverID := 0
		if len(tlv[2]) >= 4 {
			serverID = int(binary.LittleEndian.Uint32(tlv[2]))
		}
		result := uint32(1) // 官方方言：0 = 失败，1 = 成功
		_, base, found, err := b.LoadRole(account, name, serverID)
		switch {
		case err != nil:
			s.logf("(CAccess::MessageSave)connection error")
			return
		case !found:
			s.logf("(CAccess::MessageSave)role %q not found (account %q)", name, account)
			result = 0
		default:
			// merge：location（world 提供新位置）/role_state_json/
			// save_data 覆盖基底；其余 meta 列（total_sec/letter_max/
			// scene_id/role_index/sys_flags/save_para1/uid）world 不可
			// 知，一律保留库内值。
			if pos != nil {
				base.Location = fmt.Sprintf("%.3f,%.3f,%.3f,%.3f", pos[0], pos[1], pos[2], pos[3])
			}
			if rst1 := string(tlv[3]); rst1 != "" {
				base.RoleStateJSON = rst1
			}
			if len(tlv[4]) > 0 {
				if err := b.SaveRole(name, base, tlv[4]); err != nil {
					s.logf("(CAccess::MessageSave)save error: %v", err)
					result = 0
				}
			} else if err := b.SaveRole(name, base, nil); err != nil {
				s.logf("(CAccess::MessageSave)save error: %v", err)
				result = 0
			}
		}
		body := make([]byte, 12)
		putLe32(body, 0x00, sceneOf(payload[0:4]))
		putLe32(body, 0x04, sceneOf(payload[4:8]))
		putLe32(body, 0x08, result)
		s.emit(serial, back, body)
	case 0x0A: // create（solo ≥0xD4）：{client_id@0, serial@4,
		// server_id@8, account char[33]@0xC, name wchar 34槽@0x2D,
		// para wchar@0x71} → MessageCreate（INSERT；重名 false）→
		// 回执 0x06 {client,serial,arch-str uid,arch-wstr name}；拒绝回
		// {client,serial,u32 1}（uid 空，world 端 storeCreate 拒绝分支）。
		if len(payload) < 0x2D+2 {
			s.logf("(CAccess::MessageCreate)len == sizeof(store_forw_create_t)≥0xD4，得 %#x", len(payload))
			return
		}
		serverID := int(sceneOf(payload[8:12]))
		account := cstrFixed(payload[0xC:0x2D])
		name := wideNulStr(payload[0x2D : 0x2D+0x44])
		para := ""
		if len(payload) > 0x71 {
			para = wideNulStr(payload[0x71:])
		}
		existing, _ := b.QueryAccount(account, serverID)
		// 建号初始状态（rolestate.go：RoleBaseProp.xml 五维/资源初值 +
		// StarterSet.xml 授予集 + location.ini 出生点；初始布衣三件穿着
		// 进场——官服初始号 view1 口径）→ role_state_json。
		uid := newRoleUID()
		state, err := rolestate.BuildRoleState(rolestate.DefaultBirthScene, rolestate.InferSexFromPara(para), para, defaultFacultySeed)
		if err != nil {
			s.logf("(CAccess::MessageCreate)build role state: %v", err)
			return
		}
		state.Equips = rolestate.StarterEquips(state.Sex, uid, time.Now().Unix())
		state.EquipsSeeded = true
		stateJSON, err := rolestate.MarshalRoleState(state)
		if err != nil {
			s.logf("(CAccess::MessageCreate)marshal role state: %v", err)
			return
		}
		meta := RoleMeta{
			SceneID:       birthSceneID,
			Location:      birthSceneLocation,
			Index:         int32(len(existing)),
			SavePara1:     para,
			RoleStateJSON: stateJSON,
		}
		ok, err := b.CreateRole(account, name, uid, serverID, meta, nil)
		if err != nil {
			s.logf("(CAccess::MessageCreate)connection error")
			return
		}
		if !ok {
			s.logf("create role %s failed (duplicate)", name)
			body := make([]byte, 0xC)
			putLe32(body, 0x00, sceneOf(payload[0:4]))
			putLe32(body, 0x04, sceneOf(payload[4:8]))
			putLe32(body, 0x08, 1)
			s.emit(serial, back, body)
			return
		}
		s.logf("create role %s uid %s", name, uid)
		body := make([]byte, 8)
		putLe32(body, 0x00, sceneOf(payload[0:4]))
		putLe32(body, 0x04, sceneOf(payload[4:8]))
		body = appendArchStr(body, uid)
		body = appendArchWide(body, name)
		s.emit(serial, back, body)
	case 0x10: // query_account（≥72B）：{client_id@0, serial@4,
		// account 裸串@8} → MessageQueryAccount（角色清单）→ 回执
		// 0x0D {client,serial,count@8,{arch-str uid,arch-wstr name,
		// arch-wstr para}×n}（world 端 storeRolesList 装载序）。
		account := ""
		if len(payload) > 8 {
			account = cstrFixed(payload[8:])
		}
		briefs, err := b.QueryAccount(account, s.cfg.ServerID)
		if err != nil {
			s.logf("(CAccess::MessageQueryAccount)connection error")
			return
		}
		body := make([]byte, 8)
		if len(payload) >= 8 {
			putLe32(body, 0x00, sceneOf(payload[0:4]))
			putLe32(body, 0x04, sceneOf(payload[4:8]))
		}
		body = appendU32(body, uint32(len(briefs)))
		for _, br := range briefs {
			body = appendArchStr(body, br.UID)
			body = appendArchWide(body, br.Name)
			body = appendArchWide(body, br.Para)
		}
		s.emit(serial, back, body)
	case 0x0B: // delete（solo ==0xAC）：与 create 同头 + uid@0x71 →
		// MessageDelete（软删 UPDATE deleted=1, delete_time=NOW()
		// WHERE name AND account）→ 回执 0x07 {client,serial,arch-str
		// uid,arch-wstr name}（与 0x06 同构）；拒回 {client,serial,u32 1}。
		if len(payload) < 0x71+2 {
			s.logf("(CAccess::MessageDelete)len == sizeof(store_forw_delete_t)=0xAC，得 %#x", len(payload))
			return
		}
		dServerID := int(sceneOf(payload[8:12]))
		dAccount := cstrFixed(payload[0xC:0x2D])
		dName := wideNulStr(payload[0x2D : 0x2D+0x44])
		dUID := cstrFixed(payload[0x71:])
		ok, err := b.DeleteRole(dAccount, dName, dServerID)
		if err != nil {
			s.logf("(CAccess::MessageDelete)connection error")
			return
		}
		if !ok {
			s.logf("delete role %s failed (absent)", dName)
			body := make([]byte, 0xC)
			putLe32(body, 0x00, sceneOf(payload[0:4]))
			putLe32(body, 0x04, sceneOf(payload[4:8]))
			putLe32(body, 0x08, 1)
			s.emit(serial, back, body)
			return
		}
		s.logf("delete role %s uid %s", dName, dUID)
		body := make([]byte, 8)
		putLe32(body, 0x00, sceneOf(payload[0:4]))
		putLe32(body, 0x04, sceneOf(payload[4:8]))
		body = appendArchStr(body, dUID)
		body = appendArchWide(body, dName)
		s.emit(serial, back, body)
	case 0x36: // account_logon（==0x98 官方断言）：find-or-create +
		// 密码等值 + logged 门 → 回执 0x25 {client,serial,result,
		// user_type,points,used_points}
		if len(payload) != 0x98 {
			s.logf("(CAccess::MessageAccountLogon)len == sizeof(account_logon_t)=0x98，得 %#x", len(payload))
			return
		}
		name := cstrFixed(payload[0xC : 0xC+0x24])
		password := cstrFixed(payload[0x30 : 0x30+0x24])
		addr := cstrFixed(payload[0x54 : 0x54+0x20])
		info, ok, err := b.AccountLogon(name, password, addr, sceneOf(payload[8:12]))
		if err != nil {
			s.logf("(CAccess::MessageAccountLogon)connection error: %v", err)
			return
		}
		if !ok && len(name) > 0 {
			s.logf("logon %s failed (password/logged gate)", name)
		}
		body := make([]byte, 0x18)
		putLe32(body, 0x00, sceneOf(payload[0:4]))   // client_id
		putLe32(body, 0x04, sceneOf(payload[4:8]))   // serial
		putLe32(body, 0x08, uint32(boolByteU8(ok)))  // result
		putLe32(body, 0x0C, uint32(info.UserType))   // user_type
		putLe32(body, 0x10, uint32(info.Points))     // points
		putLe32(body, 0x14, uint32(info.UsedPoints)) // used_points
		s.emit(serial, back, body)
	case 0x37: // account_logoff（==0x7C 官方断言）→ 回执 0x26
		if len(payload) != 0x7C {
			s.logf("(CAccess::MessageAccountLogoff)len == sizeof(account_logoff_t)=0x7C，得 %#x", len(payload))
			return
		}
		err := b.AccountLogoff(cstrFixed(payload[0xC:0xC+0x24]), sceneOf(payload[8:12]))
		if err != nil {
			s.logf("(CAccess::MessageAccountLogoff)connection error")
			return
		}
		body := make([]byte, 0xC)
		putLe32(body, 0x00, sceneOf(payload[0:4]))
		putLe32(body, 0x04, sceneOf(payload[4:8]))
		putLe32(body, 0x08, 1)
		s.emit(serial, back, body)
	case 0x38: // account_reset → 回执 0x24
		if err := b.AccountReset(sceneOf(payload)); err != nil {
			s.logf("(CAccess::MessageAccountReset)connection error")
			return
		}
		s.emit(serial, back, []byte{1, 0, 0, 0})
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

// decodeSaveStore 解 0x04 存档包（fssoloworld/roles.go buildSaveStore
// 对偶）：官方锚点头（name 宽串@0x08 0x48 区、charge@0x50、location
// 4×double@0x80）+ 0xA0 后 TLV 段（1=account 2=server_id 3=RST1 json
// 4=save_data）。pos 全零返回 nil（无新位置，merge 保留基底）。
func decodeSaveStore(p []byte) (name string, pos *[4]float64, tlv map[uint32][]byte, ok bool) {
	if len(p) < 0xA0 {
		return "", nil, nil, false
	}
	tlv = map[uint32][]byte{}
	for off := 0xA0; off+8 <= len(p); {
		tag := le32(p, off)
		n := int(le32(p, off+4))
		off += 8
		if off+n > len(p) {
			return "", nil, nil, false // 截断段：整包拒绝
		}
		tlv[tag] = p[off : off+n]
		off += n
	}
	var loc [4]float64
	any := false
	for i := 0; i < 4; i++ {
		bits := binary.LittleEndian.Uint64(p[0x80+i*8:])
		loc[i] = math.Float64frombits(bits)
		if loc[i] != 0 {
			any = true
		}
	}
	if any {
		pos = &loc
	}
	return wideNulStr(p[0x08 : 0x08+0x48]), pos, tlv, true
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
