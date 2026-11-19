package fssolopublic

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fspublic"
	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// Server 是 SoloPublic 实体（solopublic.cpp）的 Go 等价：装配
// Reactor/InnerLink/EchoConsole/广播/状态机/数据空间/注册表/Relation/
// Trade，吸收官方 main.lua 装配序与 lua\solopublic 脚本面。
type Server struct {
	cfg    Config
	r      *reactor.Reactor
	inner  *InnerLink
	echo   *EchoConsole
	callee fspublic.Callee

	mu        sync.Mutex
	spaces    []*fspublic.PubSpace
	spaceIdx  map[string]int
	broadID   int
	innerPort int

	state     atomic.Int32 // State（官方 +0x34）
	relations atomic.Bool  // +0x174 关系装载完
	feeds     atomic.Bool  // +0x175 动态装载完
	nameuids  atomic.Bool  // +0x176 名表装载完（world RoleInfo）
	trades    atomic.Bool  // +0x177 寄售装载完

	loadSpaceCursor int // +0x168 装载链空间游标
	saveSpaceCursor int // +0x16C 回存空间游标
	saveDataCursor  int // +0x170 回存数据游标
	lastSaveMinute  int64

	Names    *NameUid
	Players  *PubPlayers
	Scenes   *PubScenes
	Relation *Relation
	Trade    *Trade

	RoleProtectBase float64 // world type0 广播（double@0x7C）

	logf    func(format string, args ...any)
	stopped atomic.Bool
}

// Options 装配可选项。
type Options struct {
	// Callee 是官方 LogicLoader(fspublogic.dll, 版本门 0x300F) 的 Go
	// 等价；nil = fspublic.DefaultCallee（1000..1107 内建修改原语）。
	Callee fspublic.Callee
	// Logf 日志回调（缺省丢弃）。
	Logf func(format string, args ...any)
}

// New 建实体（官方 ctor + main.lua 属性注入）。
func New(cfg Config, opt Options) *Server {
	if opt.Logf == nil {
		opt.Logf = func(string, ...any) {}
	}
	callee := opt.Callee
	if callee == nil {
		callee = fspublic.NewDefaultCallee()
	}
	s := &Server{
		cfg:      cfg,
		callee:   callee,
		spaceIdx: map[string]int{},
		broadID:  -1,
		Names:    NewNameUid(),
		Players:  NewPubPlayers(),
		Scenes:   NewPubScenes(),
		Relation: NewRelation(),
		Trade:    NewTrade(),
		logf:     opt.Logf,
	}
	s.state.Store(int32(StateUnknown))
	s.r = reactor.New()
	s.inner = NewInnerLink(s.r, InnerEvents{
		OnMessage:         s.processInnerMsg,
		OnStoreIdentified: s.onStoreIdentified,
		OnExit:            func() { s.logf("inner exit 请求（官方 nx_quit）") },
	}, AppName(cfg.ServerID), CommSelf, s.logf)
	s.echo = NewEchoConsole(s.r, cfg.ServerID, s.echoCommand)
	return s
}

// Start 官方 SoloPublic::Start + main.lua 装配序。
func (s *Server) Start() error {
	if err := s.r.Start(); err != nil {
		return fmt.Errorf("(SoloPublic::Start)start reactor failed: %w", err)
	}
	// 官方 CreateBroadcast(InnerAddr, BroadAddr, BroadPort, OnBroadcast)
	broadID, err := s.r.CreateBroadcast(s.cfg.BroadAddr, s.cfg.BroadPort, s.onBroadcast)
	if err != nil {
		return fmt.Errorf("(SoloPublic::Start)create broadcast failed: %w", err)
	}
	s.mu.Lock()
	s.broadID = broadID
	s.mu.Unlock()
	// 官方 EchoHook::Start(EchoAddr, EchoPort)
	if err := s.echo.Start(s.cfg.EchoAddr, s.cfg.EchoPort); err != nil {
		return fmt.Errorf("(SoloPublic::Start)start echo failed: %w", err)
	}
	// 官方 CreateListener(InnerAddr, InnerPort=0 分配回填)
	port, err := s.inner.Listen(s.cfg.InnerAddr, s.cfg.InnerPort)
	if err != nil {
		return fmt.Errorf("(SoloPublic::Start)create inner listener failed: %w", err)
	}
	s.mu.Lock()
	s.innerPort = port
	s.mu.Unlock()
	// 官方缺省空间注册：domain/nx_domains + guild/nx_guilds
	if s.FindPubSpace("domain") == nil {
		if _, err := s.AddPubSpace("domain", "nx_domains"); err != nil {
			return err
		}
	}
	if s.FindPubSpace("guild") == nil {
		if _, err := s.AddPubSpace("guild", "nx_guilds"); err != nil {
			return err
		}
	}
	// 官方 Callee->OnPublicStart
	if k := s.kernel(); k != nil {
		s.callee.OnPublicStart(k)
	}
	s.logf("app name %s", AppName(s.cfg.ServerID))
	s.logf("echo socket %s:%d", s.cfg.EchoAddr, s.cfg.EchoPort)
	s.logf("inner socket %s:%d", s.cfg.InnerAddr, port)
	s.logf("broadcast socket %s:%d", s.cfg.BroadAddr, s.cfg.BroadPort)
	s.logf("game public server %d started", s.cfg.ServerID)
	return nil
}

// Stop 官方 SoloPublic::Stop 序（广播→侦听→定时器→echo→reactor）。
func (s *Server) Stop() {
	if !s.stopped.CompareAndSwap(false, true) {
		return
	}
	s.echo.Stop()
	s.r.Stop()
	s.logf("public server shutdown...")
}

// InnerPort 实际侦听端口（官方 main.lua Start 后读回）。
func (s *Server) InnerPort() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.innerPort
}

// State 当前状态机态。
func (s *Server) State() State { return State(s.state.Load()) }

// AddPubSpace 官方 AddPubSpace(space, table)（main.lua 注册 domain/guild）。
func (s *Server) AddPubSpace(name, table string) (*fspublic.PubSpace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.spaceIdx[name]; ok {
		return nil, fmt.Errorf("(SoloPublic::AddPubSpace)space already exists: %s", name)
	}
	sp := fspublic.NewPubSpace(name, table)
	s.spaceIdx[name] = len(s.spaces)
	s.spaces = append(s.spaces, sp)
	return sp, nil
}

// FindPubSpace 官方 FindPubSpace(name)。
func (s *Server) FindPubSpace(name string) *fspublic.PubSpace {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i, ok := s.spaceIdx[name]; ok {
		return s.spaces[i]
	}
	return nil
}

// PubSpaces 空间清单快照。
func (s *Server) PubSpaces() []*fspublic.PubSpace {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*fspublic.PubSpace(nil), s.spaces...)
}

func (s *Server) kernel() *fspublic.Kernel { return nil } // Callee 观测槽占位（DefaultCallee 不依赖）

// ---- 定时（官方 CReactor::CreateTimer(1s, TimerCheck)）----

// Tick 官方 TimerCheck：AppRecord 信息 + BroadcastServerInfo +
// StateMachine(0, elapsed, 4)。由宿主每秒调用（cmd 入口的 1s 心跳）。
func (s *Server) Tick() {
	if s.stopped.Load() {
		return
	}
	s.broadcastServerInfo()
	s.stateMachine(evTick, 4)
}

// broadcastServerInfo 官方 0x24B 自宣包（kind=4/magic=0x3001）。
func (s *Server) broadcastServerInfo() {
	s.mu.Lock()
	broadID, port := s.broadID, s.innerPort
	s.mu.Unlock()
	if broadID < 0 {
		return
	}
	s.r.SendBroadcast(broadID, encodeAnnounce(s.cfg.InnerAddr, uint32(port),
		s.cfg.DistrictID, s.cfg.ServerID))
}

// onBroadcast 官方 OnBroadcast：DistrictId/ServerId 双匹配门 +
// type2 store / type1 entry 幂等外连 / type0 RoleProtectBase。
func (s *Server) onBroadcast(data []byte, fromAddr string, fromPort int) {
	a, ok := decodeAnnounce(data)
	if !ok {
		return
	}
	if a.District != s.cfg.DistrictID || a.Server != s.cfg.ServerID {
		return
	}
	switch a.Type {
	case announceStore:
		if a.Index >= CommStoreMin && a.Index < CommStoreMax {
			s.inner.CheckConnect(a.Index, a.Addr, int(a.Port))
		}
	case announceEntry:
		if a.Index < worldIndexMax {
			s.inner.CheckEntryConnect(int(a.Index), a.Addr, int(a.Port))
		}
	case announceWorldEx:
		if v, ok := worldRoleProtectBase(data); ok {
			s.RoleProtectBase = v
			s.logf("role protect base = %v", v)
		}
	}
}

// ---- 状态机（PublicStateMachine）----

func (s *Server) stateMachine(event int, elapsed int) {
	switch State(s.state.Load()) {
	case StateUnknown:
		s.stateUnknown(event)
	case StateLoadRelations:
		s.stateLoadRelations(event)
	case StateLoadFeeds:
		s.stateLoadFeeds(event)
	case StateLoadTrades:
		s.stateLoadTrades(event)
	case StateLoadSpaces:
		s.stateLoadSpaces(event)
	case StateOpened:
		s.stateOpened(event)
	case StateClosing:
		s.stateClosing(event)
	case StateClosed:
		// 官方终态空转（仍广播）
	default:
		s.logf("(SoloPublic::PublicState%s)error input %d", State(s.state.Load()), event)
	}
}

// stateUnknown 官方事件1（store identify）→ 0x2D 关系装载请求。
func (s *Server) stateUnknown(ev int) {
	if ev == evStoreConn {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], s.cfg.ServerID)
		if s.inner.SendToStore(0, 0, storeRelationReq, buf[:]) {
			s.state.Store(int32(StateLoadRelations))
			return
		}
		s.logf("(SoloPublic::PublicStateUnknown)send failed")
	}
}

// stateLoadRelations +0x174 门 → 0x30 feed 存活期配置 {id; double×levels}。
func (s *Server) stateLoadRelations(ev int) {
	if ev != evTick {
		s.logf("(SoloPublic::PublicState%s)error input %d", StateLoadRelations, ev)
		return
	}
	if !s.relations.Load() {
		return
	}
	buf := make([]byte, 0x108)
	binary.LittleEndian.PutUint32(buf, s.cfg.ServerID)
	levels := s.Relation.GetFeedLevels()
	for i := 0; i < levels && 8+i*8+8 <= len(buf); i++ {
		binary.LittleEndian.PutUint64(buf[8+i*8:], uint64(s.Relation.GetFeedLiveTime(i)))
	}
	if s.inner.SendToStore(0, 0, storeFeedConfig, buf) {
		s.state.Store(int32(StateLoadFeeds))
		return
	}
	s.logf("(SoloPublic::PublicStateRelations)send failed")
}

// stateLoadFeeds +0x175 门 → 0x3A 动态装载请求 {ServerId}。
func (s *Server) stateLoadFeeds(ev int) {
	if ev != evTick {
		s.logf("(SoloPublic::PublicState%s)error input %d", StateLoadFeeds, ev)
		return
	}
	if !s.feeds.Load() {
		return
	}
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], s.cfg.ServerID)
	if s.inner.SendToStore(0, 0, storeFeedReq, buf[:]) {
		s.state.Store(int32(StateLoadTrades))
		return
	}
	s.logf("(SoloPublic::PublicStateLoadFeeds)send failed")
}

// stateLoadTrades 双门（+0x176 名表 && +0x177 寄售）→ 首空间 0x24 请求。
func (s *Server) stateLoadTrades(ev int) {
	if ev != evTick {
		s.logf("(SoloPublic::PublicState%s)error input %d", StateLoadTrades, ev)
		return
	}
	if !s.trades.Load() || !s.nameuids.Load() {
		return
	}
	if s.requestPubInfo(0) {
		s.state.Store(int32(StateLoadSpaces))
		return
	}
	s.logf("(SoloPublic::PublicStateLoadTrades)send failed")
}

// requestPubInfo 官方 0x24 请求 {space[0x24], table[0x24]}。
func (s *Server) requestPubInfo(index int) bool {
	s.mu.Lock()
	if index < 0 || index >= len(s.spaces) {
		s.mu.Unlock()
		return false
	}
	sp := s.spaces[index]
	s.mu.Unlock()
	buf := make([]byte, 0x48)
	putFixedStr(buf[0:0x24], sp.Name)
	putFixedStr(buf[0x24:0x48], sp.TableName)
	return s.inner.SendToStore(0, 0, storePubInfoReq, buf)
}

// stateLoadSpaces 事件2（全空间装载完）→ OnPubDataLoaded 全通知 +
// member 清单广播 + Opened。
func (s *Server) stateLoadSpaces(ev int) {
	if ev == evSpacesDone {
		if s.callee != nil {
			for _, sp := range s.PubSpaces() {
				for _, d := range sp.All() {
					s.callee.OnPubDataLoaded(nil, sp.Name, d.Name)
				}
			}
		}
		body, err := encodeSpaceList(s.PubSpaces())
		if err == nil && s.inner.SendToAllMember(0, pushSpaceList, body) {
			s.state.Store(int32(StateOpened))
			s.logf("public opened（spaces broadcast）")
			return
		}
		s.logf("(SoloPublic::PublicStateLoadSpaces)send to member failed")
		return
	}
	if ev != evTick {
		s.logf("(SoloPublic::PublicState%s)error input %d", StateLoadSpaces, ev)
	}
}

// stateOpened tick：OnFrameExecute + 周期存盘；事件3：OnServerClose。
func (s *Server) stateOpened(ev int) {
	switch ev {
	case evWorldClose:
		s.callee.OnServerClose(nil)
		s.state.Store(int32(StateClosing))
		s.resetSaveCursors()
	case evTick:
		s.callee.OnFrameExecute(nil, 0)
		s.periodicSave(false)
	}
}

// stateClosing tick：游标批量回存至尽 → SendToWorld(0,2,{1}) → Closed。
func (s *Server) stateClosing(ev int) {
	if ev != evTick {
		return
	}
	if !s.periodicSave(true) {
		return
	}
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], 1)
	if s.inner.SendToWorld(0, worldCloseDone, buf[:]) {
		s.state.Store(int32(StateClosed))
		s.logf("public closed（close done 已回 world）")
	}
}

// resetSaveCursors 重置回存游标（+0x16C/+0x170）。
func (s *Server) resetSaveCursors() {
	s.mu.Lock()
	s.saveSpaceCursor, s.saveDataCursor = 0, 0
	s.mu.Unlock()
}

// periodicSave 官方 Opened 周期存盘/Closing 批量回存共用的游标扫描：
// GetModify 的 CPubData → InnerSavePubData（0x26 全 store），每帧至多
// SaveBatchSize 条（ctor 缺省 20）。返回是否扫描完成（Closing 推进依据）。
func (s *Server) periodicSave(closing bool) bool {
	if !closing {
		minute := time.Now().Unix() / 60
		s.mu.Lock()
		due := s.cfg.SaveMinutes > 0 && minute-s.lastSaveMinute >= int64(s.cfg.SaveMinutes)
		if !due {
			s.mu.Unlock()
			return true
		}
		s.lastSaveMinute = minute
		s.mu.Unlock()
	}
	const batchSize = 20 // 官方 SaveBatchSize ctor 缺省
	sent := 0
	for {
		s.mu.Lock()
		if s.saveSpaceCursor >= len(s.spaces) {
			s.mu.Unlock()
			s.resetSaveCursors()
			return true
		}
		sp := s.spaces[s.saveSpaceCursor]
		datas := sp.All()
		if s.saveDataCursor >= len(datas) {
			s.saveSpaceCursor++
			s.saveDataCursor = 0
			s.mu.Unlock()
			continue
		}
		d := datas[s.saveDataCursor]
		s.mu.Unlock()
		if d.Modify {
			s.innerSavePubData(sp, d)
			d.Modify = false
		}
		s.mu.Lock()
		s.saveDataCursor++
		s.mu.Unlock()
		sent++
		if sent >= batchSize {
			return false // 本帧额度尽，下帧续扫
		}
	}
}

// innerSavePubData 官方 0x26 回存：{space[0x24],table[0x24],wname[0x48]}
// 0x90 头 + PUB4 blob（solo flag=0 全量）→ SendToAllStore。
func (s *Server) innerSavePubData(sp *fspublic.PubSpace, d *fspublic.PubData) {
	blob, err := d.StoreArchiveBlob(false)
	if err != nil {
		s.logf("(SoloPublic::InnerSavePubData)store failed: %v", err)
		return
	}
	if len(blob) > 0x1000000 {
		s.logf("(SoloPublic::InnerSavePubData)data too big: %d", len(blob))
		return
	}
	out := make([]byte, 0x90, 0x90+len(blob))
	putFixedStr(out[0:0x24], sp.Name)
	putFixedStr(out[0x24:0x48], sp.TableName)
	putFixedWide(out[0x48:0x90], d.Name)
	out = append(out, blob...)
	if !s.inner.SendToAllStore(0, storePubSave, out) {
		s.logf("(SoloPublic::InnerSavePubData)send to store failed")
	}
}

// onStoreIdentify store identify 完成（状态机事件1）。
func (s *Server) onStoreIdentified() {
	s.stateMachine(evStoreConn, 0)
}

// ---- 内网路由（ProcessInnerMsg）----

func (s *Server) processInnerMsg(comm, id1, id2 uint32, payload []byte) {
	switch {
	case comm >= CommMemberMin && comm < CommMemberMax:
		s.processMemberMsg(int(comm-CommMemberMin), id2, payload)
	case comm >= CommStoreMin && comm < CommStoreMax:
		s.processStoreMsg(id2, payload)
	case comm == CommWorld:
		s.processWorldMsg(id2, payload)
	default:
		s.logf("(SoloPublic::ProcessInnerMsg)public message source error comm=%d", comm)
	}
}

// encodeSpaceList member 清单（Register 应答 / LoadSpaces 广播共用）：
// archive {str space, str table, int count}×。
func encodeSpaceList(spaces []*fspublic.PubSpace) ([]byte, error) {
	arc := fspublic.NewStoreArchive(64)
	for _, sp := range spaces {
		if err := arc.WriteString(sp.Name); err != nil {
			return nil, err
		}
		if err := arc.WriteString(sp.TableName); err != nil {
			return nil, err
		}
		if err := arc.WriteInt(int32(sp.Count())); err != nil {
			return nil, err
		}
	}
	return arc.Bytes(), nil
}

// echoCommand echo 内置命令集（官方为 Lua 十二调试回调的 Go 等价）。
func (s *Server) echoCommand(line string) (string, bool) {
	fields := splitFields(line)
	if len(fields) == 0 {
		return echoHelpText + "\r", true
	}
	switch fields[0] {
	case "help":
		return echoHelpText + "\r", true
	case "state":
		return fmt.Sprintf("state=%s(%d) relations=%v feeds=%v names=%v trades=%v roleProtect=%v\r",
			s.State(), s.State(), s.relations.Load(), s.feeds.Load(),
			s.nameuids.Load(), s.trades.Load(), s.RoleProtectBase), true
	case "quit":
		go func() {
			time.Sleep(50 * time.Millisecond)
			s.Stop()
		}()
		return "server will shutdown...\r", true
	case "spaces":
		var b []byte
		for _, sp := range s.PubSpaces() {
			b = append(b, fmt.Sprintf("%s->%s(%d) ", sp.Name, sp.TableName, sp.Count())...)
		}
		return string(b) + "\r", true
	case "players":
		total, online := s.Players.Count()
		return fmt.Sprintf("players total=%d online=%d\r", total, online), true
	case "scenes":
		normal, clone := s.Scenes.Count()
		return fmt.Sprintf("scenes normal=%d clone=%d\r", normal, clone), true
	case "names":
		return fmt.Sprintf("nameuid count=%d\r", s.Names.GetCount()), true
	case "relations":
		return fmt.Sprintf("relations=%d feeds=%d\r", s.Relation.RelationCount(), s.Relation.FeedCount()), true
	case "trades":
		return fmt.Sprintf("trades=%d\r", s.Trade.Count()), true
	case "datas":
		if len(fields) < 2 {
			return "usage: datas <space>\r", true
		}
		sp := s.FindPubSpace(fields[1])
		if sp == nil {
			return "space not exists\r", true
		}
		var b []byte
		for _, n := range sp.Names() {
			b = append(b, (n + " ")...)
		}
		return string(b) + "\r", true
	}
	return "", false
}

func splitFields(line string) []string {
	var out []string
	cur := ""
	for _, c := range line {
		if c == ' ' || c == '\t' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(c)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
