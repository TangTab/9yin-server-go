package fssoloauction

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/auction"
	"github.com/lizhanyin/9yin-server-go/internal/proto"
	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// Options 是进程装配参数（官方 main.lua/Lua 属性 → ini/旗标投影）。
type Options struct {
	AppName    string // 官方 AppName（AppRecord 呈现名；缺省 SoloAuction）
	DistrictID uint32
	ServerID   uint32

	InnerAddr string // 内网受面地址（空=全部接口）
	InnerPort int    // 官方缺省 16003
	EchoAddr  string // 空=不开 echo 口
	EchoPort  int    // 官方缺省 16001
	BroadAddr string // UDP 广播目的地址（官方 127.255.255.255 系）
	BroadPort int    // 官方 15000（0=不开广播）

	SaveMinutes   int  // 官方 main.lua 缺省 20
	SaveBatchSize int  // 每 tick 退存批量（缺省 100；单数据对象一次即完）
	ExitForceSave bool // 退出全量退存（官方缺省开）

	AnnounceAddr string // 自宣包内地址（空=InnerAddr 或 127.0.0.1）
	ItemTypeXML  string // Manager.LoadResource 路径（空=不装载）
	LocalStore   *LocalStore

	Logf func(format string, args ...any)
	Now  func() time.Time // 注入时钟（测试）

	// ManualTick 测试钩子：不启动 1s 内部定时循环，由测试逐拍调
	// TimerCheck 驱动（等价官方 CreateTimer(1.0) 的手动化）。
	ManualTick bool
}

// Server 复刻 SoloAuction 实体（soloauction.cpp）的进程装配：
// Reactor + InnerLink + EchoConsole + 广播 + 状态机 + world 注册表 +
// internal/auction 业务（官方 fsauctionlogic.dll 的 Go 等价）。
type Server struct {
	opt   Options
	mu    sync.Mutex // 官方单线程 EventLoop 语义：回调与 tick 互斥
	logf  func(format string, args ...any)
	now   func() time.Time
	r     *reactor.Reactor
	inner *InnerLink
	echo  *EchoConsole

	sm        *StateMachine
	mgr       *auction.Manager
	logic     *auction.Logic
	scenes    *Scenes
	players   *Players
	nameuid   *NameUid
	broadID   int
	tick      int // TimerCheck 计数（>5 复位 → 每 6 tick 一报）
	tickTotal int
	saveSec   int // g_SaveSeconds @0x101494CC
	saveIdx   int // m_SaveIndex @this+0x138（-1=扫完一轮）
	otMS      int // HCheckDueTime 累计（官方 0x493E0=300s 间隔）

	stopOnce sync.Once
	doneOnce sync.Once
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// New 建服（不含网络监听，Start 时建立）。
func New(opt Options) *Server {
	if opt.AppName == "" {
		opt.AppName = "SoloAuction"
	}
	if opt.InnerPort == 0 {
		opt.InnerPort = 16003
	}
	if opt.EchoPort == 0 {
		opt.EchoPort = 16001
	}
	if opt.SaveMinutes == 0 {
		opt.SaveMinutes = 20
	}
	if opt.SaveBatchSize == 0 {
		opt.SaveBatchSize = 100
	}
	if opt.Logf == nil {
		opt.Logf = func(string, ...any) {}
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	s := &Server{
		opt:     opt,
		logf:    opt.Logf,
		now:     opt.Now,
		broadID: -1,
		saveIdx: -1,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
		scenes:  NewScenes(),
		players: NewPlayers(),
		nameuid: NewNameUid(),
	}
	s.mgr = auction.NewManager()
	s.logic = auction.NewLogic(s.mgr)
	s.r = reactor.New()
	s.r.Logf = func(format string, args ...any) { s.logf("[reactor] "+format, args...) }
	s.inner = NewInnerLink(s.r, InnerEvents{
		OnMessage:         s.processInnerMsg,
		OnStoreIdentified: s.onStoreIdentified,
		OnExit:            func() { s.logf("inner: 收到退出请求（id2=0xFFFE）"); s.shutdown() },
	}, opt.AppName, commSelf, s.logf)
	s.echo = NewEchoConsole(s.r, s.onEchoLine)
	s.sm = &StateMachine{
		DistrictID:  opt.DistrictID,
		ServerID:    opt.ServerID,
		Logf:        s.logf,
		SendReg:     s.sendStoreReg,
		SendLoadReq: s.sendLoadRequest,
		SendClosed: func() bool { // SendToWorld(0, 1, {int 1}, 4)
			return s.inner.SendToWorld(1, 0, []byte{1, 0, 0, 0})
		},
		OnOpenedTick:  s.onOpenedTick,
		OnClosingTick: s.onClosingTick,
		OnServerClose: func() {}, // 官方 callee OnServerClose 为 no-op
	}
	return s
}

// Start 官方 SoloAuction::Start @0x100AA750 顺序：
// comm ident=2（NewInnerLink 构造）→ Reactor.Start → CreateBroadcast →
// EchoHook.Start → CreateListener(inner) → CreateTimer(1s) → 逻辑装载。
// 本实现逻辑装载=进程内装配 internal/auction；LocalStore 启用时立即发
// store-connected 事件（等价 store identify 完成）。
func (s *Server) Start() error {
	if err := s.r.Start(); err != nil {
		return fmt.Errorf("start reactor failed: %w", err)
	}
	if s.opt.BroadPort > 0 {
		addr := s.opt.BroadAddr
		if addr == "" {
			addr = "127.255.255.255"
		}
		id, err := s.r.CreateBroadcast(addr, s.opt.BroadPort, s.onBroadcast)
		if err != nil {
			return fmt.Errorf("create broadcast failed: %w", err)
		}
		s.broadID = id
	}
	if s.opt.EchoAddr != "" || s.opt.EchoPort > 0 {
		addr := s.opt.EchoAddr
		if addr == "" {
			addr = "127.0.0.1"
		}
		if err := s.echo.Start(addr, s.opt.EchoPort); err != nil {
			return fmt.Errorf("start echo failed: %w", err)
		}
	}
	if err := s.inner.Listen(s.opt.InnerAddr, s.opt.InnerPort); err != nil {
		return fmt.Errorf("create listener failed: %w", err)
	}
	if s.opt.ItemTypeXML != "" {
		if err := s.mgr.LoadResource(s.opt.ItemTypeXML); err != nil {
			s.logf("[Error]AuctionManager::LoadResource: %v（继续，官方缺表仅告警）", err)
		}
	}
	// 官方 LogicLoader 在 Start 尾装载逻辑 DLL 并调 OnAuctionStart（no-op）；
	// Go 等价：LocalStore 在场即视作 store 已连接（发送 store-connected 事件）。
	if s.opt.LocalStore != nil {
		s.sm.Event(evStoreConnected, 0)
	}
	if !s.opt.ManualTick {
		go s.tickLoop()
	}
	s.startDiscoveryWatch()
	return nil
}

// TimerCheck 导出的官方 TimerCheck（@0x100AEEF0）：ManualTick 模式下
// 由测试/宿主逐拍驱动；内部已加锁。
func (s *Server) TimerCheck(elapse float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tick++
	s.tickTotal++
	if s.tick > 5 {
		s.tick = 0
		s.logf("<info state=\"%s\" tick=\"%d\" />", s.sm.State(), s.tickTotal)
	}
	s.broadcastServerInfo()
	s.sm.Event(evTick, int(elapse))
}

// ConnectStore 是 Go 扩展：不经广播发现直连 store 0（comm 4）。
// 须在 Start 之后调用（reactor 已启动）。identify 完成即触发
// store-connected 事件（官方等价路径：广播发现 → CheckConnect）。
func (s *Server) ConnectStore(addr string) error {
	host, portS, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("store 地址须为 host:port: %w", err)
	}
	port, err := strconv.Atoi(portS)
	if err != nil {
		return fmt.Errorf("store 端口须为整数: %w", err)
	}
	s.inner.CheckConnect(commStoreMin, host, port)
	return nil
}

// Run 阻塞直至 ctx 结束后优雅关停（Stop）。
func (s *Server) Run(ctx context.Context) {
	<-ctx.Done()
	s.Stop()
}

// Stop 优雅关停：evClose（Opened→Closing）→ ExitForceSave 全量退存 →
// 通道关闭 → reactor 停止。ManualTick 模式由本调用直接完成关停
//（无内部定时循环可等）。
func (s *Server) Stop() {
	s.stopOnce.Do(func() { close(s.stopCh) })
	if s.opt.ManualTick {
		s.doneOnce.Do(func() {
			s.shutdown()
			s.r.Stop() // 回调线程并入，杜绝迟发日志
			close(s.doneCh)
		})
	}
	<-s.doneCh
}

// shutdown 是内部关停路径（evClose / Stop 共用）。
func (s *Server) shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sm.State() == StateOpened {
		s.sm.Event(evClose, 0)
	}
	if s.opt.ExitForceSave {
		s.innerSaveData(true)
	}
	s.sm.setState(StateClosed)
}

func (s *Server) tickLoop() {
	defer s.doneOnce.Do(func() {
		s.shutdown()
		s.r.Stop()
		close(s.doneCh)
	})
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-t.C:
			s.timerCheck(1.0)
		}
	}
}

// timerCheck 官方 TimerCheck @0x100AEEF0（内部定时循环驱动）。
func (s *Server) timerCheck(elapse float32) { s.TimerCheck(elapse) }

// broadcastServerInfo 官方 @0x100AEC50：0x24 字节自宣包（type=5/
// magic=0x3001/addr[0x10]/port/district/server）。
func (s *Server) broadcastServerInfo() {
	if s.broadID < 0 {
		return
	}
	addr := s.opt.AnnounceAddr
	if addr == "" {
		addr = s.opt.InnerAddr
	}
	if addr == "" {
		addr = "127.0.0.1"
	}
	s.r.SendBroadcast(s.broadID, encodeAnnounce(addr, uint32(s.opt.InnerPort),
		s.opt.DistrictID, s.opt.ServerID))
}

// onBroadcast 官方 OnBroadcast @0x100AB780：只认 magic 0x3001；
// type1=World 自宣（分区/服校验 + 序号<0x10 → 外连 comm 0）、
// type2=Store 自宣（[0x24]=comm 4..7 → 外连）。
func (s *Server) onBroadcast(data []byte, from string, fromPort int) {
	a, ok := decodeAnnounce(data)
	if !ok {
		return
	}
	switch a.Type {
	case announceWorld:
		if a.District != s.opt.DistrictID || a.Index >= worldIndexMax {
			return
		}
		s.inner.CheckConnect(commWorld, a.Addr, int(a.Port))
	case announceStore:
		// 官方门要求 +0x24 为完整 comm 4..7，而 store 自宣（本仓
		// fssolostore 与官方抓包同构）写裸 StoreID 0..3——member/
		// public/world 收侧均自行 +4，官方此分支实为永不命中的失配
		// （勘误：按裸 StoreID 收，+4 基址与其余三包统一）。
		if a.District != s.opt.DistrictID || a.Index >= commStoreMax-commStoreMin {
			return
		}
		s.inner.CheckConnect(commStoreMin+a.Index, a.Addr, int(a.Port))
	}
}

// onStoreIdentified store identify 完成（官方 InnerStoreConnected
// @0x100AEBF0 → 状态机事件 1）。Lock 已由事件循环/调用方语义保证，
// 此处独立加锁防 tick 并发。
func (s *Server) onStoreIdentified() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sm.Event(evStoreConnected, 0)
}

// sendStoreReg 官方 Unknown 态：SendToStore(0,0,0x3F,{serverId},4)。
// 无网络 store 且 LocalStore 在场时由垫片应答成功。
func (s *Server) sendStoreReg() bool {
	payload := []byte{byte(s.opt.ServerID), byte(s.opt.ServerID >> 8),
		byte(s.opt.ServerID >> 16), byte(s.opt.ServerID >> 24)}
	if s.inner.SendToStore(0, 0, 0x3F, payload) {
		return true
	}
	return s.opt.LocalStore != nil
}

// sendLoadRequest 官方 LoadAuctionHall 态每 tick：
// SendToStore(0,0,0x40,{district,server},8)；LocalStore 在场且无网络
// store 时由垫片完成 0x29→0x2A→0x2B 加载序。
func (s *Server) sendLoadRequest() bool {
	req := encodeLoadRequest(s.opt.DistrictID, s.opt.ServerID)
	if s.inner.SendToStore(0, 0, storeMsgLoad, req) {
		return true
	}
	if s.opt.LocalStore != nil {
		s.opt.LocalStore.Load(s.onStoreAttrBlock, s.onStoreRecordBlock, func() {
			s.sm.LoadOK(true, s.onAuctionDataLoaded)
		})
		return true
	}
	return false
}

// onAuctionDataLoaded 官方 callee OnAuctionDataLoaded → AuctionManager::
// OnAuctionDataLoaded（LoadResource→InitAuctionData→InitRecords→InitTimers）；
// Go 等价：装载资源 + 重建排序视图 + 挂 300s 超时扫描。
func (s *Server) onAuctionDataLoaded() {
	if s.opt.ItemTypeXML != "" && len(s.mgr.Sorted) > 0 {
		_ = s.mgr.LoadResource(s.opt.ItemTypeXML)
	}
	s.mgr.RebuildRecs()
	s.otMS = 0
	s.logf("auction data loaded：%d 行在拍（排序视图已重建）", s.mgr.GetItemNum())
}

// onOpenedTick 官方 Opened 态 tick：OnFrameExecute（no-op）+
// CAuctionTimer::Run（HCheckDueTime 300s/0x493E0 间隔，超时结算广播）
// + SaveMinutes 存盘周期（g_SaveSeconds/SaveIndex/SaveBatchSize）。
func (s *Server) onOpenedTick(elapsedSec int) {
	// CAuctionTimer：300s 一扫（HCheckDueTime @InitTimers push 0x493E0）
	s.otMS += elapsedSec * 1000
	if s.otMS >= 0x493E0 {
		s.otMS = 0
		for _, out := range s.logic.CheckOverTime() {
			s.sendAuctionMessage(out)
		}
	}
	// 存盘周期（AuctionStateOpened @0x100B1D00 段）
	s.saveSec += elapsedSec
	if s.saveSec < s.opt.SaveMinutes*60 {
		return
	}
	s.saveSec = 0
	saved := 0
	for i := s.saveIdx + 1; i < s.dataCount(); i++ {
		s.saveIdx = i
		if i == 0 { // 单数据对象（Main）；官方按 GetSave() 过滤，此处恒存
			s.innerSaveData(false)
			saved++
		}
		if saved >= s.opt.SaveBatchSize {
			break
		}
	}
	if s.saveIdx >= s.dataCount()-1 {
		s.saveIdx = -1 // 扫完一轮
	}
}

// onClosingTick 官方 Closing 态 tick：exit=true 批量退存，全部完成返 true。
// 单数据对象：一轮即完。
func (s *Server) onClosingTick(elapsedSec int) bool {
	s.innerSaveData(true)
	return true
}

// dataCount 官方 GetAuctionDataCount（本实现恒 1：Main）。
func (s *Server) dataCount() int { return 1 }

// innerSaveData 官方 InnerSaveAuctionData @0x100ADAC0：attr 块(0x41)+
// record 分片(0x42) → SendToAllStore；LocalStore 双写镜像。
func (s *Server) innerSaveData(exit bool) {
	attr := encodeAttrBlock("soloauction", "Main", []storeAttr{
		{Name: "DistrictID", Value: proto.Value{Type: proto.VarInt32, I64: int64(s.opt.DistrictID)}},
		{Name: "ServerID", Value: proto.Value{Type: proto.VarInt32, I64: int64(s.opt.ServerID)}},
		{Name: "SavedAt", Value: proto.Value{Type: proto.VarInt64, I64: s.now().Unix()}},
	})
	s.inner.SendToAllStore(0, storeMsgAttr, attr)
	if s.opt.LocalStore != nil {
		s.opt.LocalStore.SaveAttr(attr)
	}

	rows := make([][]proto.Value, 0, s.mgr.Main.Used)
	for i := range s.mgr.Main.Rows {
		if r := &s.mgr.Main.Rows[i]; r.UniqueID != "" {
			rows = append(rows, rowValues(*r))
		}
	}
	for _, block := range encodeRecordBlocks("Main", auction.MaxTotalRows, mainCols, rows) {
		s.inner.SendToAllStore(0, storeMsgRecord, block)
		if s.opt.LocalStore != nil {
			s.opt.LocalStore.SaveRecord(block)
		}
	}
}

// sendAuctionMessage 官方 kernel vt+0x20 → SoloAuction::SendAuctionMessage
// @0x100AE1F0：forw{param2=sub, count, vary} → SendToMember(target,0,0xD)。
func (s *Server) sendAuctionMessage(out auction.Outbound) bool {
	body := encodeForw(uint32(out.Sub), out.Body)
	if len(body) > maxForwBytes {
		s.logf("(SoloAuction::SendAuctionMessage)message exceed %d bytes", maxForwBytes)
	}
	if !s.inner.SendToMember(int(out.Target), 0, 0xD, body) {
		s.logf("(SoloAuction::SendAuctionMessage)send to member failed（target=%d）", out.Target)
		return false
	}
	return true
}

// ---- 内网消息分发（ProcessInnerMsg @0x100AED80）----

func (s *Server) processInnerMsg(comm, id1, id2 uint32, payload []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case comm == commWorld:
		s.processWorldMsg(id1, id2, payload)
	case comm >= commStoreMin && comm < commStoreMax:
		s.processStoreMsg(id1, id2, payload)
	case comm >= commMemberMin && comm < commMemberMax:
		s.processMemberMsg(id1, id2, payload)
	default:
		s.logf("(SoloAuction::ProcessInnerMsg)message source error! comm:%d", comm)
	}
}

// processMemberMsg 官方 @0x100B1620：id2=0 MemberRegister（len==8、
// 返回 state==Opened）/ id2=2 MemberMessage（forw → callee
// OnAuctionMessage(kernel, id1, param2, vary)）；其它忽略。
func (s *Server) processMemberMsg(id1, id2 uint32, payload []byte) {
	switch id2 {
	case 0: // MemberRegister @0x100B0C60：payload 恒 8 字节，仅按状态应答
		if len(payload) != 8 {
			s.logf("(SoloAuction::MemberRegister)len == sizeof(auction_forw_register_t) 断言失败：%d", len(payload))
			return
		}
		if s.sm.State() != StateOpened {
			s.logf("member register：状态 %s 非 Opened，member 稍后重试", s.sm.State())
		}
	case 2: // MemberMessage @0x100B0CF0
		param2, vl, err := decodeForw(payload)
		if err != nil {
			s.logf("(SoloAuction::MemberMessage)load message failed: %v", err)
			return
		}
		for _, out := range s.logic.Dispatch(int32(id1), int32(param2), vl) {
			s.sendAuctionMessage(out)
		}
	}
}

// processStoreMsg 官方 @0x100B2890：id2 0x29..0x2D 跳表。
func (s *Server) processStoreMsg(id1, id2 uint32, payload []byte) {
	if len(payload) < 4 {
		return // 各 handler 均断言 len ≥ 4 头
	}
	result := binary.LittleEndian.Uint32(payload[0:])
	block := payload[4:]
	switch id2 {
	case storeBackAttr:
		s.onStoreAttrBlock(block)
	case storeBackRec:
		s.onStoreRecordBlock(block)
	case storeBackOK:
		s.sm.LoadOK(result != 0, s.onAuctionDataLoaded)
	case 0x2C, 0x2D: // SaveAttr/SaveRecord 空应答：官方无动作
	}
}

// onStoreAttrBlock 官方 StoreAuctionLoadAttr @0x100B22F0：result==0 →
// "load pub data failed"；块名建数据对象 → LoadAttr。本实现为呈现性
// 校验（数据对象即 Main）。
func (s *Server) onStoreAttrBlock(block []byte) {
	if len(block) == 0 {
		return // 空块=空库首启
	}
	if _, name, _, err := decodeAttrBlock(block); err != nil {
		s.logf("(SoloAuction::StoreAuctionLoadAttr)load pub data failed: %v", err)
		return
	} else {
		s.logf("store：attr 块 %s 已回放", name)
	}
}

// onStoreRecordBlock 官方 StoreAuctionLoadRecord @0x100B2570：块 →
// LoadRecord 追加行。分片行号即 AddItem 序（空行复用链语义保持）。
func (s *Server) onStoreRecordBlock(block []byte) {
	if len(block) == 0 {
		return
	}
	name, _, rows, err := decodeRecordBlock(block)
	if err != nil {
		s.logf("(SoloAuction::StoreAuctionLoadRecord)load record failed: %v", err)
		return
	}
	if name != "Main" {
		s.logf("store：忽略非 Main 记录块 %q", name)
		return
	}
	for _, vs := range rows {
		r, err := valuesRow(vs)
		if err != nil {
			s.logf("(SoloAuction::StoreAuctionLoadRecord)行解码失败: %v", err)
			continue
		}
		if row := s.mgr.Main.AddItem(r); row < 0 {
			s.logf("(SoloAuction::StoreAuctionLoadRecord)表满，丢弃后续 %d 行", len(rows))
			return
		}
	}
}

// processWorldMsg 官方 @0x100B4230：id2 0..0xC 跳表维护三注册表。
func (s *Server) processWorldMsg(id1, id2 uint32, payload []byte) {
	switch id2 {
	case 0: // WorldServerState：callee OnServerState(str,5×int)（呈现）
		s.logf("world: server state 变更（%d 字节）", len(payload))
	case 1: // WorldClose → 关服事件
		s.sm.Event(evClose, 0)
	case 2: // WorldSceneAll：全量快照（0x90 定长循环）
		s.scenes.Clear()
		s.processSceneAll(payload)
	case 3: // WorldSceneAdd
		if srv, mem, sc, clone, name, res, ok := parseSceneAdd(payload); ok {
			if !s.scenes.Add(srv, mem, sc, clone, name, res) {
				s.logf("(SoloAuction::WorldSceneAdd)scene 已存在 %d", sc)
			}
		} else {
			s.logf("(SoloAuction::WorldSceneAdd)len == sizeof(member_forw_scene_add_t) 断言失败：%d", len(payload))
		}
	case 4: // WorldSceneRemove
		if len(payload) >= 4 {
			id := int32(binary.LittleEndian.Uint32(payload))
			if !s.scenes.Remove(id) {
				s.logf("(SoloAuction::WorldSceneRemove)scene not found %d", id)
			}
		}
	case 5: // WorldPlayerAll：全量快照（0x9C 定长循环）
		s.players = NewPlayers()
		s.processPlayerAll(payload)
	case 6: // WorldPlayerAdd
		if con, srv, pid, on, name, acct, ok := parsePlayerAdd(payload); ok {
			if !s.players.Add(con, srv, pid, on, name, acct) {
				s.logf("(SoloAuction::WorldPlayerAdd)add failed %s", name)
			}
		} else {
			s.logf("(SoloAuction::WorldPlayerAdd)len == sizeof(member_forw_player_add_t) 断言失败：%d", len(payload))
		}
	case 7: // WorldPlayerRemove
		if name, srv, pid, ok := parsePlayerRef(payload); ok {
			if !s.players.Remove(srv, pid, name) {
				s.logf("(SoloAuction::WorldPlayerRemove)player not found")
			}
		}
	case 8: // WorldPlayerChange：在线位翻转（布局按对称推断）
		if name, srv, pid, ok := parsePlayerRef(payload); ok {
			s.players.SetOnline(srv, pid, name, true)
		}
	case 9: // WorldRoleInfo：CLoadArchive 全量重建 {widestr,str,bool}*
		s.loadRoles(payload)
	case 0xA: // WorldRoleAdd
		if name, uid, del, ok := parseRoleRef(payload); ok {
			if !s.nameuid.Add(name, uid, del) {
				s.logf("(SoloAuction::WorldRoleInfo)add failed %s", name)
			}
		}
	case 0xB: // WorldRoleRemove
		if name, uid, _, ok := parseRoleRef(payload); ok {
			s.nameuid.Remove(name, uid)
		}
	case 0xC: // WorldRoleRecreate
		if name, uid, _, ok := parseRoleRef(payload); ok {
			s.nameuid.Add(name, uid, false)
		}
	}
}

// processSceneAll 全量场景（0x90 步进）。
func (s *Server) processSceneAll(payload []byte) {
	for off := 0; off+sceneAddSize <= len(payload); off += sceneAddSize {
		if srv, mem, sc, clone, name, res, ok := parseSceneAdd(payload[off : off+sceneAddSize]); ok {
			s.scenes.Add(srv, mem, sc, clone, name, res)
		}
	}
}

// processPlayerAll 全量玩家（0x9C 步进）。
func (s *Server) processPlayerAll(payload []byte) {
	for off := 0; off+playerAddSize <= len(payload); off += playerAddSize {
		if con, srv, pid, on, name, acct, ok := parsePlayerAdd(payload[off : off+playerAddSize]); ok {
			s.players.Add(con, srv, pid, on, name, acct)
		}
	}
}

// parsePlayerRef 解析 {i32 server, i32 player, wchar name[]}（Remove/
// Change 布局按 Add 对称推断）。
func parsePlayerRef(b []byte) (name string, serverID, playerID int32, ok bool) {
	if len(b) < 8 {
		return "", 0, 0, false
	}
	serverID = int32(binary.LittleEndian.Uint32(b[0:]))
	playerID = int32(binary.LittleEndian.Uint32(b[4:]))
	return fixedWide(b[8:]), serverID, playerID, true
}

// parseRoleRef 解析角色引用（archive 流 {widestr, str, bool?}；RoleInfo
// 实证元素序，Add 附带 deleted 布尔）。
func parseRoleRef(b []byte) (name, uid string, deleted bool, ok bool) {
	r := &storeReader{b: b}
	name, err := r.readWide()
	if err != nil {
		return "", "", false, false
	}
	uid, err = r.readStr()
	if err != nil {
		return "", "", false, false
	}
	if r.remain() >= 1 {
		d, _ := r.readByte()
		deleted = d != 0
	}
	return name, uid, deleted, true
}

// loadRoles 官方 WorldRoleInfo @0x100B3AD3：Clear 后循环
// {ReadWideStr, ReadString, ReadBool} → Add。
func (s *Server) loadRoles(payload []byte) {
	s.nameuid.Clear()
	r := &storeReader{b: payload}
	for r.remain() > 0 {
		name, err := r.readWide()
		if err != nil {
			return
		}
		uid, err := r.readStr()
		if err != nil {
			return
		}
		del, err := r.readByte()
		if err != nil {
			return
		}
		s.nameuid.Add(name, uid, del != 0)
	}
}

// ---- echo 命令面（官方 Lua on_input 的 Go 等价）----

func (s *Server) onEchoLine(line string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	cmd := line
	var arg string
	for i := 0; i < len(line); i++ {
		if line[i] == ' ' || line[i] == '\t' {
			cmd, arg = line[:i], line[i+1:]
			break
		}
	}
	switch cmd {
	case "help":
		return "命令：state stat items save rebuild clear roles scenes players help\r\n"
	case "state":
		return fmt.Sprintf("state=%s tick=%d saveIdx=%d saveSec=%d/%ds\r\n",
			s.sm.State(), s.tickTotal, s.saveIdx, s.saveSec, s.opt.SaveMinutes*60)
	case "stat":
		w, st, m := s.inner.ConnStats()
		return fmt.Sprintf("items=%d conns(world/store/member)=%d/%d/%d echo=%d roles=%d\r\n",
			s.mgr.GetItemNum(), w, st, m, s.echo.ConnCount(), s.nameuid.Count())
	case "items":
		n, c := 0, 0
		for bt := 1; bt <= 4; bt++ {
			if sr := s.mgr.GetSortedRecord(int32(bt), 1); sr != nil {
				n += sr.GetItemNum()
				if sr.GetItemNum() > 0 {
					c++
				}
			}
		}
		return fmt.Sprintf("total=%d usedBigTypes=%d\r\n", s.mgr.GetItemNum(), c)
	case "save":
		s.innerSaveData(true)
		return "已触发全量退存（0x41/0x42 → store + local）\r\n"
	case "rebuild":
		s.mgr.RebuildRecs()
		return "排序视图已重建\r\n"
	case "clear":
		s.mgr.ClearData()
		return "全表已清空（GM msg 6 语义）\r\n"
	case "roles":
		return fmt.Sprintf("roles=%d uid 示例=%s\r\n", s.nameuid.Count(), s.nameuid.GetName(arg))
	case "scenes":
		return s.scenes.Dump()
	case "players":
		return s.players.Dump()
	default:
		return fmt.Sprintf("[echo] 未知命令 %q（help 查看命令表）\r\n", cmd)
	}
}
