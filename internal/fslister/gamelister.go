package fslister

import (
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/login"
)

// 表容量（官方对象布局实证）：server_t[0x100] × line_t[16]，
// client_t[0x2000]（客户端槽在 ClientServer 侧等价实现）。
const (
	MaxServers = 0x100
	MaxLines   = 0x10

	// sweepPeriodSec 官方主循环节拍：CTickTimer 每 1000ms 一拍，
	// 每 5 拍全表扫描（CheckServer/CheckClient）。
	sweepPeriodSec = 5

	// bestRateInitial 选线初值（官方常量 [0x100b03f4] = 10.0f）：
	// 真实 cur/max 负载率恒 ≤10 即可入围，等值时取后者。
	bestRateInitial = float32(10.0)
)

// Line 对应官方 line_t（server_t+0x27C 起 16 槽 × 0x28）。
type Line struct {
	Valid  bool
	LineID int32
	Max    int32
	Cur    int32
	Port   int32
	IP     string

	// latch 是官方 line_t+0x04 的 type 闩锁：entry 注册置位，每次
	// 扫描清零——entry 必须在每个扫描周期内重发，否则线被摘除。
	latch bool
}

// Server 对应官方 server_t（0x4FC）。
type Server struct {
	ID    int32
	FromWorld bool
	Visible   bool
	IP        string
	Name      string // 宽显示名（应答 # 转义来源）
	Max       int32
	Cur       int32
	Lines     [MaxLines]Line

	// worldLatch 官方 server_t+0x20：world 注册置位，扫描清零——
	// world 必须在每个扫描周期内重发，否则整服除名。
	worldLatch bool
}

// GameLister 对应官方 GameLister 实体的表与生命周期语义
// （对象布局/函数地址见权威文档）。线程安全：注册（UDP 面）与
// 应答（TCP 面）可并发，锁粒度等价官方每结构 CRITICAL_SECTION。
type GameLister struct {
	mu          sync.Mutex
	IDC         int
	DistrictID  int32
	ListerID    int
	MaxConnects int

	servers   []*Server // len ≤ MaxServers；nil = 空槽
	highWater int       // 官方 server 计数高水位（FindServer 扫描界）

	sweep int // 秒拍计数（0..sweepPeriodSec-1）

	// Logf 官方 Console 日志（"Add server %s(%s)" 等）；nil 静默。
	Logf func(format string, args ...any)
}

// NewGameLister 建表。字段对应官方属性：IDC(+0x24)/DistrictId(+0x28)/
// ListerId(+0x2C)/MaxConnects(+0x30，默认 1000)。
func NewGameLister(idc, districtID, listerID, maxConnects int) *GameLister {
	if maxConnects <= 0 {
		maxConnects = 1000
	}
	return &GameLister{
		IDC:         idc,
		DistrictID:  int32(districtID),
		ListerID:    listerID,
		MaxConnects: maxConnects,
		servers:     make([]*Server, 0, MaxServers),
	}
}

func (gl *GameLister) logf(format string, args ...any) {
	if gl.Logf != nil {
		gl.Logf(format, args...)
	}
}

// FindServer 官方 FindServer@100468E0：按 server_id 在高水位内线性查。
func (gl *GameLister) FindServer(id int32) *Server {
	gl.mu.Lock()
	defer gl.mu.Unlock()
	return gl.findServerLocked(id)
}

func (gl *GameLister) findServerLocked(id int32) *Server {
	n := gl.highWater
	if n > len(gl.servers) {
		n = len(gl.servers)
	}
	for i := 0; i < n; i++ {
		if s := gl.servers[i]; s != nil && s.ID == id {
			return s
		}
	}
	return nil
}

// AddServer 官方 AddServer@100469A0：取首个空槽，valid=1、记 id、
// 清零 16 条线；表满（0x100）返 nil。
func (gl *GameLister) AddServer(id int32) *Server {
	gl.mu.Lock()
	defer gl.mu.Unlock()
	return gl.addServerLocked(id)
}

func (gl *GameLister) addServerLocked(id int32) *Server {
	for i := 0; i < MaxServers; i++ {
		if i < len(gl.servers) && gl.servers[i] != nil {
			continue
		}
		s := &Server{ID: id, FromWorld: true}
		if i < len(gl.servers) {
			gl.servers[i] = s
		} else {
			gl.servers = append(gl.servers, s)
		}
		if gl.highWater < i+1 {
			gl.highWater = i + 1
		}
		return s
	}
	return nil
}

// RecvWorldInfo 官方 RecvWorldServerInfo@10046BD0：未命中则 AddServer
// （日志 "Add server %s(%s)"），随后刷新 from_world/闩锁/visible/
// cur/max/ip/name。表满拒收（false）。
func (gl *GameLister) RecvWorldInfo(w WorldInfo) bool {
	gl.mu.Lock()
	s := gl.findServerLocked(w.ServerID)
	added := false
	if s == nil {
		s = gl.addServerLocked(w.ServerID)
		if s == nil {
			gl.mu.Unlock()
			return false
		}
		added = true
	}
	s.FromWorld = true
	s.worldLatch = true
	s.Visible = w.Visible
	s.Cur = w.Cur
	s.Max = w.Max
	s.IP = w.IP
	s.Name = w.Name
	gl.mu.Unlock()
	if added {
		gl.logf("Add server %s(%s)", w.Name, w.IP)
	}
	return true
}

// RecvEntryInfo 官方 RecvEntryServerInfo@100471D0：line_index 必须
// <0x10；server 必须已由 world 包建立（entry 不建服）；随后全量
// 覆盖该线槽并置闩锁。
func (gl *GameLister) RecvEntryInfo(e EntryInfo) bool {
	if e.LineIndex < 0 || e.LineIndex >= MaxLines {
		return false
	}
	gl.mu.Lock()
	defer gl.mu.Unlock()
	s := gl.findServerLocked(e.ServerID)
	if s == nil {
		return false
	}
	l := &s.Lines[e.LineIndex]
	l.Valid = true
	l.latch = true
	l.LineID = e.LineID
	l.Max = e.Max
	l.Cur = e.Cur
	l.Port = e.Port
	l.IP = e.IP
	return true
}

// HandlePacket 官方 CBroadSockHook::OnReceive 报文门 + 分发。
// 返回是否被接受（false=静默丢弃，官方同款不回错）。
func (gl *GameLister) HandlePacket(pkt []byte) bool {
	kind, ok := ParseRegistration(pkt, gl.DistrictID)
	if !ok {
		return false
	}
	if kind == KindWorld {
		return gl.RecvWorldInfo(ParseWorldInfo(pkt))
	}
	return gl.RecvEntryInfo(ParseEntryInfo(pkt))
}

// TickSeconds 推进 n 个秒拍（官方 DoWait 主循环每 1000ms 一拍），
// 第 sweepPeriodSec 拍执行全表扫描。幂等于挂钟，测试可单拍驱动。
func (gl *GameLister) TickSeconds(n int) {
	for i := 0; i < n; i++ {
		gl.mu.Lock()
		gl.sweep++
		do := gl.sweep >= sweepPeriodSec
		if do {
			gl.sweep = 0
		}
		gl.mu.Unlock()
		if do {
			gl.Sweep()
		}
	}
}

// Sweep 官方 CheckServer@100475D0（单次全表扫描）：
//
//	线级：entry 闩锁未在本周期内置位 → 线除名
//	服级：world 闩锁未置位 → 整服除名（槽回收，日志 Remove server）
//
// 官方对死服是槽位 valid=0 而非压缩数组——空槽可被 AddServer 复用；
// Go 侧置 nil 等价（FindServer/AddServer 均按 nil 判空）。
func (gl *GameLister) Sweep() {
	type removed struct{ name, ip string }
	var gone []removed
	var dead []int
	gl.mu.Lock()
	for i, s := range gl.servers {
		if s == nil {
			continue
		}
		for j := range s.Lines {
			l := &s.Lines[j]
			if !l.Valid {
				continue
			}
			if l.latch {
				l.latch = false
			} else {
				l.Valid = false
			}
		}
		if s.FromWorld {
			if s.worldLatch {
				s.worldLatch = false
			} else {
				gone = append(gone, removed{s.Name, s.IP})
				dead = append(dead, i)
			}
		}
	}
	for _, i := range dead {
		gl.servers[i] = nil
	}
	gl.mu.Unlock()
	for _, r := range gone {
		gl.logf("Remove server %s(%s)", r.name, r.ip)
	}
}

// Entries 官方 SendServerList@10044320 的条目投影：每个 valid &&
// visible 的服，在其 16 线中取 cur/max 负载率最小者（IDC>0 时仅
// line_id==IDC 的线），busy = 服级 cur*5/max（login.BusyLevel 同公式）。
// 无可用线的服不出现在应答中。
func (gl *GameLister) Entries() []login.ServerEntry {
	gl.mu.Lock()
	defer gl.mu.Unlock()
	entries := make([]login.ServerEntry, 0, len(gl.servers))
	for _, s := range gl.servers {
		if s == nil || !s.FromWorld || !s.Visible {
			continue
		}
		best := -1
		bestRate := bestRateInitial
		for i := range s.Lines {
			l := &s.Lines[i]
			if !l.Valid || l.Max == 0 {
				continue
			}
			if gl.IDC > 0 && l.LineID != int32(gl.IDC) {
				continue
			}
			rate := float32(l.Cur) / float32(l.Max)
			if rate <= bestRate {
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
			Busy: login.BusyLevel(int(s.Cur), int(s.Max), 0),
		})
	}
	return entries
}

// EncodeResponse 生成 svrlist 应答行（含 CRLF）——官方 text_encode_msg
// 方言（与 FsSoloLister golden 同源）。空表应答 "$svrlist\r\n"
// （官方 VarList 仅含首元素时的编码形态）。
func (gl *GameLister) EncodeResponse() []byte {
	return login.EncodeServerList(gl.Entries()...)
}

// Snapshot 深拷贝当前服表（状态呈现/测试用；行级闩锁不外泄）。
func (gl *GameLister) Snapshot() []Server {
	gl.mu.Lock()
	defer gl.mu.Unlock()
	out := make([]Server, 0, len(gl.servers))
	for _, s := range gl.servers {
		if s == nil {
			continue
		}
		out = append(out, *s)
	}
	return out
}
