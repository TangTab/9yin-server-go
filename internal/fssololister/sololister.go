// Package fssololister 复刻官方 FsSoloLister.dll（小写系 game_lister.exe
// 直启的区服列表服务器，BGserverCJ 全符号快照，2026-11-19 反汇编终局）。
//
// 与大写系 FsLister.dll（internal/fslister）同源分工但方言不同：
//
//	维度			FsSoloLister（本包）		FsLister（fslister）
//	------------------------------------------------------------------
//	进程形态		game_lister.exe 直启三件套	FsLoader.exe 装载
//	注册魔数		0x3001			0x300
//	world 包长		0x84（多 recommend@+0x78）	0x78
//	UDP 口		双口：本地环回中继 + 配置口	单 BroadSock
//	分区门		pkt+0x1C==DistrictId（配置口）	同
//	客户端行协议	\r 和 \n 皆终结、控制符不丢	仅 \n、<0x20 丢弃
//	行缓冲		0xFF（client_t 内嵌 0x100）	0x1000（hook 分配）
//	客户端机		30s 请求 / 5s 应答滞留 / 1s 关	60s / 5s / 关闭
//	版本门		ClientVersion（"svrlist <ver>"）	无
//	busy			recommend>10 直通，否则
//				服级 cur*5/max		服级 cur*5/max（无推荐）
//	选线初值		10.0f			1.0f
//	IP 黑白名单		BlockOrAllow + 范围表		无（OnBlockIP 恒 0）
//	MaxConnects 默认	ctor 8000 / lua 1024	1000
//
// 源码路径断言串：c:\wuyf\svn\serveriii\branches\publish20130508\
// server\fssololister\sololister.cpp（obj: SoloLister/Reactor/EchoHook/
// AppRecord/BlockIP/LogShareMem/TextProtocol/AppTool）。
package fssololister

import (
	"net/netip"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/login"
)

// 表容量与节拍（官方对象布局/TimerCheck@0x10064650 实证）。
const (
	MaxLines = 0x10 // line_t[16]（server_t+0x268 起，stride 0x28）

	// serverSweepTicks/recordTickTicks 官方 TimerCheck 静态计数器：
	// 客户端机每秒扫（CheckClient），服表每 5 拍扫（CheckServer），
	// AppRecord 信息行每 6 拍写（>5 触发即第 6 拍）。
	serverSweepTicks = 5
	recordTickTicks  = 6

	// bestRateInitial 官方选线初值 [0x100d0e78] = 10.0f：负载率严格小于
	// 当前最优才换线（等值取先者——与大写系"取后者"相反，实证见
	// ReceiveCommand fcompp/test ah,0x41 分支）。
	bestRateInitial = float32(10.0)

	// 官方日志方言（CORE_TRACE）。
	logAddServer    = "add server %s(%s)"    // RecvWorldServerInfo 新建服
	logRemoveServer = "remove server %s(%s)" // CheckServer 心跳丢失
	logAddLine      = "add line %d on server %s -> %s:%d (max %d)" // RecvEntryServerInfo 新建线（首跳打，心跳不刷屏）
	logClientCon    = "address %s:%d connected, total %d"
	logClientDis    = "address %s:%d disconnected, total %d"
	logRemoveClient = "remove client %d, total %d"
	logIPBlocked    = "address %s:%d ip blocked"
)

// Line 对应官方 line_t（0x28）：+0 valid / +4 type 闩锁（entry 置 1，
// 扫描清零）/ +8 line_id / +0xC max / +0x10 cur / +0x14 port / +0x18 ip[16]。
type Line struct {
	Valid  bool
	Latch  bool // 官方 type 字段的心跳闩锁语义
	LineID int32
	Max    int32
	Cur    int32
	Port   int32
	IP     string
}

// Server 对应官方 server_t（0x4E8）：+0 visible / +1 from_world /
// +4 server_id / +8 world_latch / +0xC max / +0x10 cur / +0x14 recommend /
// +0x18 ip[16] / +0x28 wchar name[0x40] / +0x68 wchar name2[0x100] /
// +0x268 line_t[16]。
type Server struct {
	Visible    bool
	FromWorld  bool
	ID         int32
	WorldLatch bool
	Max        int32
	Cur        int32
	Recommend  int32
	IP         string
	Name       string
	Lines      [MaxLines]Line
}

// Client 对应官方 client_t（0x10C）：+0 state / +4 counter / +8 行缓冲
// char[0x100] / +0x108 line_len。状态机（CheckClient@0x10064140）：
//
//	0 连接待请求：counter>30（秒）→ 温和关闭（ShutdownConnector）→ 2
//	1 已应答：    counter>5 → 温和关闭 → 2
//	2 关闭中：    counter>1 → 硬关（DeleteConnector）+ 删表 + 日志
type Client struct {
	State   int32
	Counter int32
}

// 客户端机状态。
const (
	ClientWaitRequest int32 = 0
	ClientAnswered    int32 = 1
	ClientClosing     int32 = 2
)

// BlockMode 官方 BlockOrAllow（默认 -1）。
const (
	BlockDisable = -1 // 不拦
	BlockBlack   = 0  // 黑名单模式：命中拒绝
	BlockWhite   = 1  // 白名单模式：未命中拒绝
)

// ipRange 官方 block_addr_t：uint32 起/止（reverse_ip 字节序归一后比较）。
type ipRange struct {
	beg, end uint32
}

// SoloLister 官方 SoloLister 实体的表与生命周期语义。字段偏移实证：
// IDC(+0x34)/DistrictId(+0x38)/ListerId(+0x3C)/MaxConnects(+0x40)/
// ClientVersion(+0x44)/BlockOrAllow(+0x48)。
// 线程安全：UDP 注册面与 TCP 应答面可并发（官方 reactor 单线程派发，
// Go 侧锁粒度等价）。
type SoloLister struct {
	mu sync.Mutex

	IDC           int32
	DistrictID    int32
	ListerID      int32 // 官方 SetListerId 0..7 钳制
	MaxConnects   int32
	ClientVersion int32
	BlockOrAllow  int
	InnerAddr     string // 配置广播口绑定地址（[Inner] Addr）
	BroadAddr     string // 广播目标地址（[Broadcast] Addr）
	BroadPort     int    // 双 UDP 口共用端口（[Broadcast] Port）

	servers   []*Server // nil = 空槽（官方 free 索引表复用）
	freeSlots []int32   // 官方 +0x16C TArrayPod<unsigned>：删除下标回收
	clients   []*Client // 下标 = connector id 槽

	stateVal int32 // 官方 +0x30（Start 置 1"Paused"，GetState 跳表）
	sweep    int   // TimerCheck 秒拍（5 拍全表扫描）

	ipRanges []ipRange

	// Logf 官方 Console 日志（CORE_TRACE 语义）；nil 静默。
	Logf func(format string, args ...any)
}

// New 建实体。默认值对齐官方 ctor@0x1005FD60：MaxConnects=8000、
// BlockOrAllow=-1、其余 0；lua main.lua 会以 ini 值覆写（默认 1024）。
func New() *SoloLister {
	return &SoloLister{
		MaxConnects:  8000,
		BlockOrAllow: BlockDisable,
	}
}

func (sl *SoloLister) logf(format string, args ...any) {
	if sl.Logf != nil {
		sl.Logf(format, args...)
	}
}

// SetListerID 官方 SetListerId@0x100610A0：仅 0≤id<8 接受，越界忽略。
func (sl *SoloLister) SetListerID(id int32) {
	if id >= 0 && id < 8 {
		sl.ListerID = id
	}
}

// ---- IP 黑白名单（CBlockIP + AddBlockIP@0x10061100）----

// AddBlockIP 官方 CBlockIP::AddBlockIP(beg,end)：点分十进制区间入表。
// 解析失败返回 false（lua "add test ip failed" 同款）。
func (sl *SoloLister) AddBlockIP(beg, end string) bool {
	b, err := netip.ParseAddr(beg)
	if err != nil {
		return false
	}
	e, err := netip.ParseAddr(end)
	if err != nil {
		return false
	}
	if !b.Is4() || !e.Is4() {
		return false
	}
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.ipRanges = append(sl.ipRanges, ipRange{reverseIP(b.As4()), reverseIP(e.As4())})
	return true
}

// ClearBlockIP 官方 ClearBlockIP。
func (sl *SoloLister) ClearBlockIP() {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.ipRanges = nil
}

// reverse_ip 官方 @0x1005c460：点分十进制转 u32 时字节序取反
// （ntohl 语义），使区间比较与同序无关——Go 侧统一 big-endian。
func reverseIP(b [4]byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// IsBlockIP 官方 IsBlockIP@0x100611D0 + OnAccept 拦截语义：
// BlockOrAllow=-1 恒不拦；0=黑名单命中拦；1=白名单未命中拦。
func (sl *SoloLister) IsBlockIP(addr string) bool {
	ip, err := netip.ParseAddr(addr)
	if err != nil {
		return sl.BlockOrAllow == BlockWhite
	}
	if !ip.Is4() {
		ip = ip.Unmap()
	}
	if !ip.Is4() {
		return sl.BlockOrAllow == BlockWhite
	}
	v := reverseIP(ip.As4())
	sl.mu.Lock()
	hit := false
	for _, r := range sl.ipRanges {
		if v >= r.beg && v <= r.end {
			hit = true
			break
		}
	}
	mode := sl.BlockOrAllow
	sl.mu.Unlock()
	switch mode {
	case BlockBlack:
		return hit
	case BlockWhite:
		return !hit
	default:
		return false
	}
}

// ---- 服表（FindServer@0x10063940 / AddServer@0x10063A00）----

// FindServer 按 server_id 线性查（+4 字段为键）。
func (sl *SoloLister) FindServer(id int32) *Server {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	return sl.findServerLocked(id)
}

func (sl *SoloLister) findServerLocked(id int32) *Server {
	for _, s := range sl.servers {
		if s != nil && s.ID == id {
			return s
		}
	}
	return nil
}

// addServerLocked 官方语义：优先复用 free 索引表里的空槽，否则追加。
func (sl *SoloLister) addServerLocked(id int32) *Server {
	s := &Server{ID: id}
	if n := len(sl.freeSlots); n > 0 {
		idx := sl.freeSlots[0]
		sl.freeSlots = sl.freeSlots[1:]
		if int(idx) < len(sl.servers) {
			sl.servers[idx] = s
			return s
		}
	}
	sl.servers = append(sl.servers, s)
	return s
}

// RecvWorldInfo 官方 RecvWorldServerInfo@0x10063B60：未命中则 AddServer
// （日志 "add server %s(%s)"）；随后 from_world=1、world_latch=1、
// visible=包[+0x74]≠0、cur/max/recommend/ip/name 全量覆盖、name2 清空。
func (sl *SoloLister) RecvWorldInfo(w WorldInfo) bool {
	sl.mu.Lock()
	s := sl.findServerLocked(w.ServerID)
	added := false
	if s == nil {
		s = sl.addServerLocked(w.ServerID)
		added = true
	}
	s.FromWorld = true
	s.WorldLatch = true
	s.Visible = w.Visible
	s.Max = w.Max
	s.Cur = w.Cur
	s.Recommend = w.Recommend
	s.IP = w.IP
	s.Name = w.Name
	sl.mu.Unlock()
	if added {
		sl.logf(logAddServer, w.Name, w.IP)
	}
	return true
}

// RecvEntryInfo 官方 RecvEntryServerInfo@0x10063FB0：line_index 必须
// <0x10；server 必须已由 world 包建立（entry 不建服）；随后线槽
// valid=1、闩锁=1（type 1）、line_id/max/cur/port/ip 全量覆盖。
func (sl *SoloLister) RecvEntryInfo(e EntryInfo) bool {
	if e.LineIndex < 0 || e.LineIndex >= MaxLines {
		return false
	}
	sl.mu.Lock()
	s := sl.findServerLocked(e.ServerID)
	if s == nil {
		sl.mu.Unlock()
		return false
	}
	l := &s.Lines[e.LineIndex]
	first := !l.Valid
	l.Valid = true
	l.Latch = true
	l.LineID = e.LineID
	l.Max = e.Max
	l.Cur = e.Cur
	l.Port = e.Port
	l.IP = e.IP
	name := s.Name
	sl.mu.Unlock()
	if first {
		sl.logf(logAddLine, e.LineIndex, name, e.IP, e.Port, e.Max)
	}
	return true
}

// HandleRegistrationPacket 配置口收包处理（broad_cfg_cb 全门 + 分发）。
func (sl *SoloLister) HandleRegistrationPacket(pkt []byte) bool {
	kind, ok := ParseRegistration(pkt, sl.DistrictID)
	if !ok {
		return false
	}
	if kind == KindWorld {
		return sl.RecvWorldInfo(ParseWorldInfo(pkt))
	}
	return sl.RecvEntryInfo(ParseEntryInfo(pkt))
}

// ---- 客户端机 ----

// AddClient 官方 AddClient@0x10062B70：新 client_t（全零）入槽、计数 +1、
// 日志 "address %s:%d connected, total %d"。返回槽下标。
func (sl *SoloLister) AddClient(addr string, port int) int {
	sl.mu.Lock()
	idx := -1
	for i, c := range sl.clients {
		if c == nil {
			idx = i
			break
		}
	}
	c := &Client{}
	if idx < 0 {
		sl.clients = append(sl.clients, c)
		idx = len(sl.clients) - 1
	} else {
		sl.clients[idx] = c
	}
	total := 0
	for _, cc := range sl.clients {
		if cc != nil {
			total++
		}
	}
	sl.mu.Unlock()
	sl.logf(logClientCon, addr, port, total)
	return idx
}

// RemoveClient 官方 RemoveClient@0x10062DB0：删槽、计数 -1、日志。
func (sl *SoloLister) RemoveClient(idx int, addr string, port int) {
	sl.mu.Lock()
	if idx < 0 || idx >= len(sl.clients) {
		sl.mu.Unlock()
		return
	}
	sl.clients[idx] = nil
	for len(sl.clients) > 0 && sl.clients[len(sl.clients)-1] == nil {
		sl.clients = sl.clients[:len(sl.clients)-1]
	}
	total := 0
	for _, cc := range sl.clients {
		if cc != nil {
			total++
		}
	}
	sl.mu.Unlock()
	sl.logf(logClientDis, addr, port, total)
}

// ClientCount 官方 GetClientCount（+0x140 计数）。
func (sl *SoloLister) ClientCount() int {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	n := 0
	for _, c := range sl.clients {
		if c != nil {
			n++
		}
	}
	return n
}

// MarkStarted 官方 Start 尾部 [+0x30]=1（GetState 跳表 1="Paused"——
// 官方即如此置位，Opened 保留给暂停恢复路径）。
func (sl *SoloLister) MarkStarted() {
	sl.mu.Lock()
	sl.stateVal = 1
	sl.mu.Unlock()
}

// AcquireRequest 官方 ReceiveCommand 的单请求门：state==0 才受理，
// 受理即占位（应答成功后置 1）。返回 false = 拒绝（已应答/关闭中）。
func (sl *SoloLister) AcquireRequest(idx int) bool {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if idx < 0 || idx >= len(sl.clients) {
		return false
	}
	c := sl.clients[idx]
	if c == nil || c.State != ClientWaitRequest {
		return false
	}
	return true
}

// MarkAnswered 应答写回成功后置位（官方 [client]=1,[client+4]=0）。
func (sl *SoloLister) MarkAnswered(idx int) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if idx < 0 || idx >= len(sl.clients) {
		return
	}
	if c := sl.clients[idx]; c != nil {
		c.State = ClientAnswered
		c.Counter = 0
	}
}

// TickSeconds 推进 n 个秒拍（官方 TimerCheck@0x10064650：每 5 拍全服表
// CheckServer；客户端机每秒的 CheckClient 由 server.go 连接层以
// deadline 等价驱动——30s/5s/1s 时序同源）。幂等，测试可单拍驱动。
func (sl *SoloLister) TickSeconds(n int) {
	for i := 0; i < n; i++ {
		sl.mu.Lock()
		sl.sweep++
		doSweep := sl.sweep >= serverSweepTicks
		if doSweep {
			sl.sweep = 0
		}
		sl.mu.Unlock()
		if doSweep {
			sl.CheckServer()
		}
	}
}

// CheckServer 官方 CheckServer@0x10064400（5s 扫）：
//
//	线级：valid && 闩锁未置 → 线除名；否则闩锁清零（entry 每 5s 必重发）
//	服级：from_world && world_latch 未置 → 整服删除（槽回收进 free 表，
//	      日志 "remove server %s(%s)"）；否则 world_latch 清零
func (sl *SoloLister) CheckServer() {
	type gone struct{ name, ip string }
	var dead []int
	var gones []gone
	sl.mu.Lock()
	for i, s := range sl.servers {
		if s == nil {
			continue
		}
		for j := range s.Lines {
			l := &s.Lines[j]
			if !l.Valid {
				continue
			}
			if l.Latch {
				l.Latch = false
			} else {
				l.Valid = false
			}
		}
		if s.FromWorld {
			if s.WorldLatch {
				s.WorldLatch = false
			} else {
				s.FromWorld = false
				gones = append(gones, gone{s.Name, s.IP})
				dead = append(dead, i)
			}
		}
	}
	for _, i := range dead {
		sl.servers[i] = nil
		sl.freeSlots = append(sl.freeSlots, int32(i))
	}
	sl.mu.Unlock()
	for _, g := range gones {
		sl.logf(logRemoveServer, g.name, g.ip)
	}
}

// ---- svrlist 应答（ReceiveCommand@0x10063140）----

// CheckVersionGate 官方版本门：ClientVersion>0 时请求行长 ≥8、前 7 字节
// "svrlist"、atoi(第 9 字节起) ≥ ClientVersion。返回 (通过, 是版本不达标)；
// 前缀/长度不符 = 静默不理（false,false），版本低 = version_error（false,true）。
func (sl *SoloLister) CheckVersionGate(line string) (pass bool, versionLow bool) {
	if sl.ClientVersion <= 0 {
		return true, false
	}
	if len(line) < 8 || line[:7] != "svrlist" {
		return false, false
	}
	ver := atoiOfficial(line[8:])
	if ver >= sl.ClientVersion {
		return true, false
	}
	return false, true
}

// atoiOfficial C 运行库 atoi 方言（前导符号 + 数字，遇非数字截断，空=0）。
func atoiOfficial(s string) int32 {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	neg := false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	var v int32
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		v = v*10 + int32(s[i]-'0')
		i++
	}
	if neg {
		return -v
	}
	return v
}

// Entries 官方 svrlist 应答条目投影：每个 visible 的服在其 16 线中取
// cur/max 负载率严格最小者（IDC>0 时仅 line_id==IDC 参选；
// 官方门 = line.max≠0 且 line 首字节非 0）；busy = recommend>10 直通，
// 否则 max>0 时 cur*5/max（login.BusyLevel 同公式）。无可用线的服跳过。
func (sl *SoloLister) Entries() []login.ServerEntry {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	entries := make([]login.ServerEntry, 0, len(sl.servers))
	for _, s := range sl.servers {
		if s == nil || !s.Visible {
			continue
		}
		best := -1
		bestRate := bestRateInitial
		for i := range s.Lines {
			l := &s.Lines[i]
			if !l.Valid || l.Max == 0 {
				continue
			}
			if sl.IDC > 0 && l.LineID != sl.IDC {
				continue
			}
			rate := float32(l.Cur) / float32(l.Max)
			if rate < bestRate { // 官方严格小于（fcompp 门），等值取先者
				bestRate = rate
				best = i
			}
		}
		if best < 0 {
			continue
		}
		l := &s.Lines[best]
		entries = append(entries, login.ServerEntry{
			Name: s.Name,
			Host: l.IP,
			Port: int(l.Port),
			Busy: login.BusyLevel(int(s.Cur), int(s.Max), int(s.Recommend)),
		})
	}
	return entries
}

// EncodeResponse svrlist 应答行（含 CRLF）——官方 text_encode_msg 方言
// （internal/login 与 golden 同源双重钉死）。
func (sl *SoloLister) EncodeResponse() []byte {
	return login.EncodeServerList(sl.Entries()...)
}

// StateName 官方 GetState@0x10060FC0：+0x30 状态字 → 字符串
// （0=Closed/1=Paused/2=Opened/3=Unknown；Start 后为 1"Paused"——
// 官方即如此，2 留给暂停恢复路径）。
func (sl *SoloLister) StateName() string {
	switch sl.state() {
	case 0:
		return "Closed"
	case 1:
		return "Paused"
	case 2:
		return "Opened"
	case 3:
		return "Unknown"
	default:
		return ""
	}
}

func (sl *SoloLister) state() int32 {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	return sl.stateVal
}
