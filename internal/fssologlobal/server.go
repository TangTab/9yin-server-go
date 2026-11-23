package fssologlobal

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/charge"
	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// Options 是进程装配参数（官方 main.lua 属性赋值 → ini/旗标投影）。
type Options struct {
	AppName string // 官方 AppName（"mutex_%s" 名；缺省 global）

	OuterAddr string // 官方 [Outer] Addr（空=全部接口）
	OuterPort int    // 官方缺省 4003
	EchoAddr  string // 官方 [Echo] Addr（空=不开 echo 口）
	EchoPort  int    // 官方缺省 15012

	Domains *DomainTable // 空=官方 main.lua 实配（role/guild）
	Store   DBAccess     // 官方 CAccess→DBSystem；必填

	UseAppRecord bool // 官方 UseAppRecord（main.lua 实配 true）→ 状态行日志

	Logf func(format string, args ...any)
	Now  func() time.Time // 注入时钟（测试）

	// ManualTick 测试钩子：不启动 1s 内部定时循环，由测试逐拍调
	// TimerCheck 驱动（等价官方 CreateTimer(1.0) 的手动化）。
	ManualTick bool
}

// Server 复刻 SoloGlobal 实体（sologlobal.cpp）的进程装配：
// Reactor + 外网命令受面 + EchoConsole + 状态机 + 域表 + DBAccess。
type Server struct {
	opt     Options
	mu      sync.Mutex // 官方单线程 EventLoop 语义：回调与 tick 互斥
	logf    func(format string, args ...any)
	now     func() time.Time
	r       *reactor.Reactor
	echo    *EchoConsole
	domains *DomainTable
	store   DBAccess
	sm      *StateMachine

	clients map[int]*client // 官方 TArrayPod<client_t*> m_Clients（键=连接 id）
	total   int             // 官方 this+0x334 连接计数

	outerListener int   // 官方 this+0x34c（-1=未建）
	timerID       int   // 官方 this+0x350（-1=未建；Go 侧仅呈现
	tickCount     int   // 官方 g_counter（>5 复位 → 每 6 tick 一报）
	tickTotal     int64 // 累计 tick（呈现）
	dbBreak       bool  // 官方 CLockVal this+0x354（DatabaseBreak）
	dbErr         int   // 官方 CLockVal this+0x358（DatabaseError 计数）

	stopOnce sync.Once
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// client 复刻官方 client_t（0x40C）：+8 行缓冲 buf[0x400]（有效上限
// 0x3FF）、+0x408 len。
type client struct {
	buf []byte
}

// clientLineMax 官方行缓冲上限（0x3FF：len < 0x3FF 才收字节）。
const clientLineMax = 0x3FF

// encodeMax 官方 text_encode_msg 缓冲上限（0x3FE：n<0 即 encode 失败）。
const encodeMax = 0x3FE

// New 建服（不含网络监听，Start 时建立）。
func New(opt Options) *Server {
	if opt.AppName == "" {
		opt.AppName = "global"
	}
	if opt.OuterPort == 0 {
		opt.OuterPort = 4003
	}
	if opt.EchoPort == 0 {
		opt.EchoPort = 15012
	}
	if opt.Domains == nil {
		opt.Domains = DefaultDomains()
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
		domains: opt.Domains,
		store:   opt.Store,
		clients: map[int]*client{},
		sm:      NewStateMachine(opt.Logf),
		r:       reactor.New(),
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
	s.r.Logf = func(format string, args ...any) { s.logf("[reactor] "+format, args...) }
	s.echo = NewEchoConsole(s.r, s.echoCommand)
	return s
}

// Start 复刻官方 SoloGlobal::Start@0x1002cea0（顺序语义）：
// 受面 → echo → 定时器 → DB 连接 → 状态机 Start 事件 →
// "SoloGlobal pointer is 0x%p" trace。
func (s *Server) Start() error {
	// 1-3（mutex/quick log/app record）为跨进程运维件，不复刻（差异
	// 见包文档）；UseAppRecord 投影为状态行日志开关。
	// 4. reactor
	if err := s.r.Start(); err != nil {
		s.logf("(SoloGlobal::Start)start reactor failed: %v", err)
		return err
	}
	// 5. 外网受面（官方 CreateListener(…,100,0x400,0x400,0x4000,…)）
	id, err := s.r.CreateListener(s.opt.OuterAddr, s.opt.OuterPort, 100, reactor.ConnHandler{
		OnOpen:  s.onClientOpen,
		OnClose: s.onClientClose,
		OnInput: s.onClientInput,
	})
	if err != nil {
		s.logf("(SoloGlobal::Start)create outer listener failed: %v", err)
		return err
	}
	s.mu.Lock()
	s.outerListener = id
	s.mu.Unlock()
	// 6. echo 口（官方 CEchoHook::Start(EchoAddr, EchoPort)）
	if s.opt.EchoAddr != "" {
		if err := s.echo.Start(s.opt.EchoAddr, s.opt.EchoPort); err != nil {
			s.logf("(SoloGlobal::Start)create echo listener failed: %v", err)
			return err
		}
	}
	// 7. 定时器（官方 CreateTimer(TimerCheck, 1.0f)）
	if !s.opt.ManualTick {
		go s.tickLoop()
	}
	// 9. DB（官方 CAccess Start 失败仅 trace，不中断启动）
	if err := s.store.Ping(); err != nil {
		s.setDatabaseBreak()
		s.logf("(SoloGlobal::Start)start access failed: %v", err)
	}
	// 10. 状态机 Start 事件（Unknown→Opened）
	s.sm.Dispatch(eventStart, 0, 0)
	// 11. 官方 trace
	s.logf("SoloGlobal pointer is %p", s)
	s.logf("app name %s", s.opt.AppName)
	s.logf("echo socket %s:%d", s.opt.EchoAddr, s.opt.EchoPort)
	s.logf("outer socket %s:%d", s.opt.OuterAddr, s.opt.OuterPort)
	s.logf("game global server started")
	return nil
}

// Stop 复刻官方 Stop@0x1002da50：停受面/echo/reactor，DB Close
// 失败仅 trace。幂等。
func (s *Server) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		s.logf("global server shutdown...")
		s.echo.Stop()
		s.r.Stop()
		if err := s.store.Close(); err != nil {
			s.logf("(SoloGlobal::Stop)stop access failed: %v", err)
		}
		close(s.doneCh)
	})
}

// Done 暴露停机完成信号（测试/宿主同步用）。
func (s *Server) Done() <-chan struct{} { return s.doneCh }

// RunStartRun 便捷面：Start → 等 ctx/stopCh → Stop。
func (s *Server) Run(ctx context.Context) error {
	if err := s.Start(); err != nil {
		s.Stop()
		return err
	}
	select {
	case <-ctx.Done():
	case <-s.stopCh:
	}
	s.Stop()
	return nil
}

// tickLoop 官方 CreateTimer(1.0f) 等价：每秒 TimerCheck。
func (s *Server) tickLoop() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-t.C:
			s.TimerCheck()
		}
	}
}

// onClientOpen 官方 AddClient@0x1002dfa0：new client_t → 落槽 →
// 计数++ → "address %s:%d connected, total %d"。
func (s *Server) onClientOpen(connID int, addr string, port int) {
	s.mu.Lock()
	prev := s.clients[connID] // 官方断言槽空；Go 侧防御直接顶替
	s.clients[connID] = &client{}
	s.total++
	total := s.total
	s.mu.Unlock()
	if prev != nil {
		s.logf("slot %d reused before close", connID)
	}
	s.logf("address %s:%d connected, total %d", addr, port, total)
}

// onClientClose 官方 RemoveClient@0x1002e1e0：delete + 置空 + 计数--。
func (s *Server) onClientClose(connID int, addr string, port int) {
	s.mu.Lock()
	if _, ok := s.clients[connID]; ok {
		delete(s.clients, connID)
		s.total--
	}
	total := s.total
	s.mu.Unlock()
	s.logf("address %s:%d closed, total %d", addr, port, total)
}

// onClientInput 官方 ReceiveClientMsg@0x1002e370：逐字节 \r/\n 终止、
// 行缓冲上限 0x3FF、空行不分发。
func (s *Server) onClientInput(connID int, data []byte) {
	s.mu.Lock()
	c := s.clients[connID]
	if c == nil {
		s.mu.Unlock()
		return
	}
	for _, b := range data {
		if b == '\n' || b == '\r' {
			if len(c.buf) > 0 {
				line := string(c.buf)
				c.buf = nil
				s.mu.Unlock()
				s.receiveCommand(connID, line)
				s.mu.Lock()
			}
			c.buf = nil
			continue
		}
		if len(c.buf) < clientLineMax {
			c.buf = append(c.buf, b)
		} // 超长字节丢弃
	}
	s.mu.Unlock()
}

// receiveCommand 官方 ReceiveCommand@0x1002e570（行命令分派）。
// 解码失败/参数不足/域未知 → 仅 trace 无应答；正常 → 应答行 + CRLF。
func (s *Server) receiveCommand(connID int, line string) bool {
	s.mu.Lock()
	_, ok := s.clients[connID]
	s.mu.Unlock()
	if !ok {
		s.logf("(SoloGlobal::ReceiveCommand)state error")
		return false
	}
	vl, err := charge.DecodeLine(line)
	if err != nil {
		s.logf("(SoloGlobal::ReceiveCommand)decode message error: %v", err)
		return false
	}
	if len(vl) < 2 {
		s.logf("(SoloGobal::ReceiveCommand)message too short")
		return false
	}
	seq, cmd := vl.StringVal(0), vl.StringVal(1)
	out := charge.VarList{}
	out.AddString(seq)
	out.AddString(cmd)
	switch cmd {
	case "add_name":
		if len(vl) < 6 {
			s.logf("(SoloGlobal::ReceiveCommand)need 6 arguments")
			return false
		}
		domain, name := vl.StringVal(2), wideVal(vl, 3)
		owner, serverID := wideVal(vl, 4), vl.Int(5)
		table := s.domains.Lookup(domain)
		if table == "" {
			s.logf("(SoloGlobal::ReceiveCommand)no domain table %s", domain)
			return false
		}
		out.AddString(table)
		out.AddWideStr(name)
		ok, err := s.store.AddName(table, name, owner, serverID)
		if err != nil {
			s.onDatabaseError()
		}
		out.AddInt(b2i(ok))
	case "remove_name":
		if len(vl) < 4 {
			s.logf("(SoloGlobal::ReceiveCommand)need 4 arguments")
			return false
		}
		domain, name := vl.StringVal(2), wideVal(vl, 3)
		table := s.domains.Lookup(domain)
		if table == "" {
			s.logf("(SoloGlobal::ReceiveCommand)no domain table %s", domain)
			return false
		}
		out.AddString(table)
		out.AddWideStr(name)
		ok, err := s.store.RemoveName(table, name)
		if err != nil {
			s.onDatabaseError()
		}
		out.AddInt(b2i(ok))
	default:
		// 未知命令：应答仅 [seq, cmd]（官方同构回包）。
	}
	encoded := out.EncodeLine()
	if len(encoded) > encodeMax {
		s.logf("(SoloGlobal::ReceiveCommand)encode msg failed")
		return false
	}
	frame := encoded + "\r\n"
	if !s.r.Send(connID, []byte(frame)) {
		s.logf("(SoloGlobal::ReceiveCommand)send msg failed")
		return false
	}
	return true
}

// TimerCheck 官方 TimerCheck@0x1002f010：每 6 tick 报一次状态行
// （官方经 CAppRecord::SetAppInfo 写共享内存；Go 为日志），
// 随后状态机 tick 事件。
func (s *Server) TimerCheck() {
	s.mu.Lock()
	s.tickTotal++
	if s.opt.UseAppRecord {
		s.tickCount++
		if s.tickCount > 5 {
			s.tickCount = 0
			db := "OK"
			if s.dbBreak {
				db = "Break"
			}
			s.logf(`<info state="%s" database="%s" db_err="%d" tick="%d" />`,
				s.sm.State().String(), db, s.dbErr, tickCount32())
		}
	}
	s.mu.Unlock()
	s.healDatabaseBreak()
	s.sm.Dispatch(eventTick, 0, 0)
}

// StatusLine 呈现官方 `<info …/>` 状态行（测试用，与 TimerCheck 同构）。
func (s *Server) StatusLine() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	db := "OK"
	if s.dbBreak {
		db = "Break"
	}
	return fmt.Sprintf(`<info state="%s" database="%s" db_err="%d" tick="%d" />`,
		s.sm.State().String(), db, s.dbErr, tickCount32())
}

// setDatabaseBreak/onDatabaseError 复刻官方 SetDatabaseBreak(1) /
// IncDatabaseError（CLockVal this+0x354/0x358）。
func (s *Server) setDatabaseBreak() {
	s.mu.Lock()
	s.dbBreak = true
	s.mu.Unlock()
}

func (s *Server) onDatabaseError() {
	s.mu.Lock()
	s.dbErr++
	s.mu.Unlock()
}

// ClearDatabaseError 官方同名方法（echo/内核面）。
func (s *Server) ClearDatabaseError() {
	s.mu.Lock()
	s.dbBreak = false
	s.dbErr = 0
	s.mu.Unlock()
}

// healDatabaseBreak Go 扩展：DatabaseBreak 期间逐拍探活，探通即摘掉
// Break（官方只有 echo 的 ClearDatabaseError 能清——库重启后状态行会
// 一直挂在 Break 上）。dbErr 是断连期间的累计，保留供事后查。
func (s *Server) healDatabaseBreak() {
	s.mu.Lock()
	broken := s.dbBreak
	s.mu.Unlock()
	if !broken {
		return
	}
	if err := s.store.Ping(); err != nil {
		return
	}
	s.mu.Lock()
	s.dbBreak = false
	s.mu.Unlock()
	s.logf("(SoloGlobal)database recovered")
}

// State 当前状态机态（官方 GetState）。
func (s *Server) State() State { return s.sm.State() }

// ConnCount 外网受面现行连接数（官方 this+0x334）。
func (s *Server) ConnCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.total
}

// tickCount32 官方 Port_GetTickCount 的呈现等价（毫秒）。
func tickCount32() int {
	return int(time.Now().UnixMilli() & 0xFFFFFFFF)
}

// wideVal 官方 WideStrVal(i)：宽串元素取值；对窄串宽容取 S
// （超集兼容，见包文档差异节）。
func wideVal(vl charge.VarList, i int) string {
	if i < 0 || i >= len(vl) {
		return ""
	}
	return vl[i].S
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// echoCommand 是 echo 口命令集（官方 main.lua echo_input 族：quit/end
// + Go 扩展 help/state/status/domains；quit 触发关停、end 关连接）。
// 返回（应答文本，是否关闭该连接）。
func (s *Server) echoCommand(line string) (string, bool) {
	cmd := strings.TrimSpace(line)
	switch cmd {
	case "quit":
		s.logf("server will shutdown...")
		go s.Stop() // 官方 close(): WriteLine → Stop → nx_quit
		return ">server will shutdown...\r", false
	case "end":
		return "", true // 官方 end → CloseEcho（无应答直接断开）
	case "help":
		return ">quit shutdown; end close; state/status/domains info\r", false
	case "state", "status":
		return ">" + s.StatusLine() + "\r", false
	case "domains":
		parts := []string{}
		for idx, nt := range s.domains.Entries() {
			parts = append(parts, fmt.Sprintf("%d:%s->%s", idx, nt[0], nt[1]))
		}
		return ">" + strings.Join(parts, " ") + "\r", false
	default:
		return ">undefined command: " + cmd + "\r", false
	}
}
