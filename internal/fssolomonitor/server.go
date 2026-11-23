package fssolomonitor

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
	AppName string // 官方 AppName（"monitor_%d"；缺省 monitor_0）

	OuterAddr string // 官方 [Outer] Addr
	OuterPort int    // 官方缺省 4008
	EchoAddr  string // 官方 [Echo] Addr（空=不开 echo 口）
	EchoPort  int    // 官方缺省 15012

	ServerID      int         // 官方 [System] ServerId（缺省 0）
	ConnectString string      // 官方 [Main]（Go 侧仅呈现）
	Limits        LimitConfig // 官方 Res\MonitorLimit.ini（LoadSystemConfig 映射）
	Sink          AlertSink   // 官方 CAccess→IDBSystem；空=内存兜底
	UseAppRecord  bool        // 官方 UseAppRecord（实配 true）→ 状态行日志
	GlobalSwitch  int         // 官方 +0x558（>0 才启用 alert_total_capital）

	Logf func(format string, args ...any)
	Now  func() time.Time // 注入时钟（测试）

	// ManualTick 测试钩子：不启动 1s 内部定时循环，由测试逐拍调
	// TimerCheck 驱动（等价官方 CreateTimer(1.0) 的手动化）。
	ManualTick bool
}

// Server 复刻 SoloMonitor 实体（solomonitor.cpp，sizeof 0x560）的进程
// 装配：Reactor + 外网命令受面（:4008）+ 多客户端 EchoConsole + 状态机
// + 7 监控表 + CAccess 告警落库线程。
type Server struct {
	opt    Options
	mu     sync.Mutex // 官方单线程 EventLoop 语义：回调与 tick 互斥
	logf   func(format string, args ...any)
	now    func() time.Time
	r      *reactor.Reactor
	echo   *EchoConsole
	sm     *StateMachine
	access *Access
	sink   AlertSink

	// 七表（官方 +0x35c..+0x374，ctor 均 SetWarningCallback→OnReportWarning）
	createItem       *CreateItemTable
	createItemDetail *CreateItemDetailTable
	roleData         *RoleDataTable
	recvLetter       *RecvLetterDetailTable
	tradeItem        *TradeItemDetailTable
	globalItem       *GlobalItemDetailTable
	loginDetail      *LoginDetailTable

	clients map[int]*client // 官方 TArrayPod<client_t*>（键=连接 id）
	total   int             // 官方 +0x334

	nLogType  int // 官方 +0x378（OnReportWarning 文本引用）
	clientIdx int // 官方 +0x37c（-1=无当前 client）

	faculityLimit  int      // 官方 +0x504
	livepointLimit int      // 官方 +0x508
	diffLimit      int64    // 官方 +0x510 PlayerDiffCapitalLimit
	diff           [8]int64 // 官方 +0x518..+0x550 资金差累加器

	outerListener int // 官方 +0x34c（-1=未建）
	tickCount     int // 官方 g_AppInfoTick（>5 复位 → 每 6 拍一报）
	capitalTick   int // 官方 g_CapitalTick（>0x3C 复位 → 每 61 拍巡检）
	tickTotal     int64
	dbBreak       bool // 官方 CLockVal +0x354
	dbErr         int  // 官方 CLockVal +0x358

	stopOnce sync.Once
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// client 复刻官方 client_t（0x42C）：+8 行缓冲 buf[0x400]（有效上限
// 0x3FF）、+0x408 len、+0x40c "ip:port"。
type client struct {
	buf  []byte
	addr string
}

// clientLineMax 官方行缓冲上限（0x3FF）。
const clientLineMax = 0x3FF

// New 建服（不含网络监听，Start 时建立）。
func New(opt Options) *Server {
	if opt.AppName == "" {
		opt.AppName = "monitor_0"
	}
	if opt.OuterPort == 0 {
		opt.OuterPort = 4008
	}
	if opt.EchoPort == 0 {
		opt.EchoPort = 15012
	}
	if opt.Limits.SysItemLimits == nil {
		opt.Limits = DefaultLimitConfig()
	}
	if opt.Sink == nil {
		opt.Sink = NewMemoryAlertSink()
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
		sink:    opt.Sink,
		clients: map[int]*client{},
		sm:      NewStateMachine(opt.Logf),
		r:       reactor.New(),
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
	// 官方 ctor：八表 new + SetWarningCallback(0x1009e580, this)。
	warn := func(role string, typ int, reason string) { s.onReportWarning(typ, role, reason) }
	now64 := func() int64 { return s.now().Unix() }
	s.createItem = NewCreateItemTable(now64, warn)
	s.createItemDetail = NewCreateItemDetailTable(now64, warn)
	s.roleData = NewRoleDataTable(now64, warn)
	s.recvLetter = NewRecvLetterDetailTable(now64, warn)
	s.tradeItem = NewTradeItemDetailTable(now64, warn)
	s.globalItem = NewGlobalItemDetailTable(now64, warn)
	s.loginDetail = NewLoginDetailTable(now64, warn)
	// 官方 SetTableCapacity 注册面默认容量（docs §五表）。
	s.createItemDetail.SetCapacity(1000000)
	s.recvLetter.SetCapacity(1000000)
	s.tradeItem.SetCapacity(100000)
	s.globalItem.SetCapacity(0) // ItemDetail 默认停用
	s.loginDetail.SetCapacity(100000)
	s.applyLimits(opt.Limits)
	s.r.Logf = func(format string, args ...any) { s.logf("[reactor] "+format, args...) }
	s.echo = NewEchoConsole(s.r, s.echoCommand)
	// 官方 CAccess 在 Start 建；Go 侧提前到 New（告警入队面常在，
	// Start 只拉写线程）。
	s.access = NewAccess(s.sink, s.logf, s.onDatabaseError, s.setDatabaseBreak)
	return s
}

// applyLimits 官方 LoadSystemConfig@0x100a5a80 键→落点映射。
func (s *Server) applyLimits(lc LimitConfig) {
	s.createItem.SetCheckTimerInterval(lc.CheckTimerInterval)
	s.roleData.SetCheckTimerInterval(lc.CheckTimerInterval)
	s.faculityLimit = lc.FaculityLimit
	s.livepointLimit = lc.LivepointLimit
	s.diffLimit = lc.DiffCapitalLimit
	s.roleData.SetTradeCounterLimit(lc.ExchangeAmountLimit)
	s.loginDetail.SetAmountLimit(lc.RoleNumLimit)
	s.loginDetail.SetOnlineAmountLimit(lc.OnlineRoleNumLimit)
	for cfg, v := range lc.SysItemLimits {
		s.createItem.SetItemCounterLimit(cfg, v)
	}
	for cfg, v := range lc.RoleItemLimits {
		s.roleData.SetItemCounterLimit(cfg, int64(v))
	}
	for i, v := range lc.RoleCapitalLimits {
		s.roleData.SetCapitalIncLimit(i, v)
	}
}

// Start 复刻官方 SoloMonitor::Start@0x100a0300（顺序语义；官方 1-3 为
// 跨进程运维件不复刻）：reactor → 外网受面 → echo → 定时器 →
// CAccess → 状态机 Start 事件。
func (s *Server) Start() error {
	if err := s.r.Start(); err != nil {
		s.logf("(SoloMonitor::Start)start reactor failed: %v", err)
		return err
	}
	id, err := s.r.CreateListener(s.opt.OuterAddr, s.opt.OuterPort, 100, reactor.ConnHandler{
		OnOpen:  s.onClientOpen,
		OnClose: s.onClientClose,
		OnInput: s.onClientInput,
	})
	if err != nil {
		s.logf("(SoloMonitor::Start)create outer listener failed: %v", err)
		return err
	}
	s.mu.Lock()
	s.outerListener = id
	s.mu.Unlock()
	if s.opt.EchoAddr != "" {
		if err := s.echo.Start(s.opt.EchoAddr, s.opt.EchoPort, s.logf); err != nil {
			s.logf("(SoloMonitor::Start)create echo listener failed: %v", err)
			return err
		}
	}
	// 官方 CreateTimer(1.0f, TimerCheck)：Go 侧固定 1s 循环。
	if !s.opt.ManualTick {
		go s.tickLoop()
	}
	// 官方 CAccess Start 失败仅 trace 不返回 false。
	if err := s.access.Start(); err != nil {
		s.logf("(SoloMonitor::Start)start access failed: %v", err)
	}
	s.sm.Dispatch(eventStart, 0, 0)
	// 官方 main.lua WriteLine 序列。
	s.logf("app name %s", s.opt.AppName)
	s.logf("echo socket %s:%d", s.opt.EchoAddr, s.opt.EchoPort)
	s.logf("outer socket %s:%d", s.opt.OuterAddr, s.opt.OuterPort)
	s.logf("monitor server started")
	return nil
}

// Stop 复刻官方 Stop@0x100a0e40（幂等）。
func (s *Server) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		s.logf("monitor server shutdown...")
		s.echo.Stop()
		s.r.Stop()
		if s.access != nil {
			s.access.Stop()
		} else if err := s.sink.Close(); err != nil {
			s.logf("(SoloMonitor::Stop)stop access failed: %v", err)
		}
		close(s.doneCh)
	})
}

// Done 停机完成信号（测试/宿主同步用）。
func (s *Server) Done() <-chan struct{} { return s.doneCh }

// Run 便捷面：Start → 等 ctx/stopCh → Stop。
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

// ---- 外网受面（官方 AddClient/RemoveClient/ReceiveClientMsg）----

func (s *Server) onClientOpen(connID int, addr string, port int) {
	s.mu.Lock()
	prev := s.clients[connID]
	s.clients[connID] = &client{addr: fmt.Sprintf("%s:%d", addr, port)}
	s.total++
	total := s.total
	s.mu.Unlock()
	if prev != nil {
		s.logf("slot %d reused before close", connID)
	}
	s.logf("address %s:%d connected, total %d", addr, port, total)
}

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

// receiveCommand 官方 ReceiveCommand@0x100a19a0：text_decode →
// vl[0]=命令名 → 10 命令分派（docs §四表）。未知命令 trace 后返回
// true（不断链）。
func (s *Server) receiveCommand(connID int, line string) bool {
	vl, err := charge.DecodeLine(line)
	if err != nil {
		s.logf("(SoloGobal::ReceiveCommand)decode message error: %v", err)
		return false
	}
	s.mu.Lock()
	s.nLogType = 0
	_, ok := s.clients[connID]
	if ok {
		s.clientIdx = connID
	}
	s.mu.Unlock()
	if !ok {
		s.logf("(SoloGobal::ReceiveCommand)state error")
		return false
	}
	if len(vl) == 0 {
		s.logf("(SoloGobal::ReceiveCommand)message too short")
		return false
	}
	cmd := vl.StringVal(0)
	s.logf("recv message type:%s", cmd)
	switch cmd {
	case "createitem":
		if len(vl) < 4 {
			s.logf("(SoloGobal::ReceiveCommand)format failed")
			return false
		}
		return s.monitorCreateItem(vl.StringVal(2), wideVal(vl, 3))
	case "item_log":
		return s.processItemLog(vl)
	case "entry_scene":
		return s.entryCheckRoleData(vl, true)
	case "leave_scene":
		return s.entryCheckRoleData(vl, false)
	case "faculty":
		if len(vl) < 3 {
			s.logf("(SoloGobal::ReceiveCommand)format failed")
			return false
		}
		role := wideVal(vl, 1)
		val := vl.Int(2)
		s.logf("(SoloMonitor::ReceiveCommand)wsRole:%s, faculty:%d", role, val)
		if val > s.faculityLimit {
			s.onReportWarning(0, "S", fmt.Sprintf(
				"玩家修为值超过阀值!玩家:%s 当前值:%d 阀值:%d", role, val, s.faculityLimit))
		}
		return true
	case "alert_total_capital":
		return s.alertTotalCapital(vl)
	case "alert_total_capital_new":
		return s.alertTotalCapitalNew(vl)
	case "loginfo":
		if len(vl) < 5 {
			s.logf("(SoloGobal::ReceiveCommand)format failed")
			return false
		}
		s.logf("(SoloMonitor::ReceiveCommand)loginfo mac:%s addr:%s acc:%s role:%s",
			vl.StringVal(1), vl.StringVal(2), vl.StringVal(3), wideVal(vl, 4))
		s.loginDetail.Login(vl.StringVal(1), vl.StringVal(2), vl.StringVal(3), wideVal(vl, 4))
		return true
	case "logout":
		if len(vl) < 5 {
			s.logf("(SoloGobal::ReceiveCommand)format failed")
			return false
		}
		s.loginDetail.LoginOut(vl.StringVal(1), vl.StringVal(2), vl.StringVal(3), wideVal(vl, 4))
		return true
	default:
		s.logf("recv unknown message type:%s", cmd)
		return true
	}
}

// monitorCreateItem 官方 MonitorCreateItem@0x100a3de0 为 Debug 遗留基准
// （会摧毁真实 CreateItemDetail 表，恒返 false）。复刻按官方语义降级：
// no-op + 日志，保留表与已有记录。
func (s *Server) monitorCreateItem(uid, role string) bool {
	s.logf("(SoloMonitor::MonitorCreateItem),strItemUid:%s, wsRole:%s", uid, role)
	s.logf("(SoloMonitor::MonitorCreateItem)debug benchmark skipped (official destroys CreateItemDetail table)")
	return false
}

// processItemLog 官方 ProcessItemLog@0x100a4c60（item_log 10 变元分派）。
func (s *Server) processItemLog(vl charge.VarList) bool {
	if len(vl) < 11 {
		s.logf("(SoloMonitor::ProcessItemLog)format failed")
		return false
	}
	role := wideVal(vl, 1)
	cfg := wideVal(vl, 2)
	uid := vl.StringVal(3)
	amount := int64Val(vl, 4)
	logType := vl.Int(5)
	itemType := vl.Int(6)
	maxStack := int64(vl.Int(7))
	srcRole := wideVal(vl, 8)
	comment := wideVal(vl, 9)
	_ = itemType
	s.logf("(SoloMonitor::ProcessItemLog), wsRole:%s, wsConfigId:%s, nAmount:%d, nLogType:%d, nMaxAmonut:%d, wsSrcRole:%s, wsComment:%s",
		role, cfg, amount, logType, maxStack, srcRole, comment)
	capitalType := lastCharAtoi(cfg)
	s.mu.Lock()
	s.nLogType = logType
	// ProcessPlayerCapital@0x100a6120：全服净流量（+0x520=diff#2 直接交易槽）。
	if logType == 35001 {
		s.diff[1] -= amount
	} else if logType == 45001 {
		s.diff[1] += amount
	}
	s.mu.Unlock()

	switch {
	case logType > 10000 && logType < 20000: // 物品获得
		switch logType {
		case 10031, 10032, 10035, 10037: // 回购类
			s.roleData.BuyBackItem(role, uid)
		default:
			s.createItem.AddItem(cfg, amount, srcRole)
			if maxStack == 1 {
				s.createItemDetail.Add(cfg, uid, srcRole)
			}
		}
		s.roleData.AddItem(role, cfg, amount, logType)
		s.globalItem.BuyItem(cfg, amount, maxStack, role)
	case logType > 20000 && logType < 28000: // 物品消耗/交易
		if maxStack == 1 {
			s.roleData.TradeItem(role, cfg, uid, amount)
		}
		s.globalItem.SellItem(cfg, amount, maxStack, role)
	case logType > 30000 && logType < 34999: // 货币增加
		s.roleData.IncCapital(role, capitalType, amount, !(logType == 30010 || logType == 30027))
	case logType > 35000 && logType < 36029: // 货币增加2
		s.roleData.IncCapital(role, capitalType, amount, !(logType == 35002 || logType == 30027))
	case (logType >= 40000 && logType < 44999) || (logType >= 45000 && logType < 46029): // 货币减少
		s.roleData.DecCapital(role, capitalType, amount)
	}
	// 第二趟（明细）：
	switch logType {
	case 30010, 10042, 10032, 35002:
		s.recvLetter.Add(comment, srcRole, logType, amount)
	case 10031, 35001:
		s.tradeItem.BuyItem(cfg, amount, role, srcRole, uid, logType)
	case 20058, 45001:
		s.tradeItem.SellItem(cfg, amount, role, srcRole, uid, logType)
	}
	return true
}

// entryCheckRoleData 官方 Entry/LeaveCheckRoleData@0x100a4420/0x100a4840：
// [1]=名 [2..4]=三货币 I64 [5..]=逐背包数；上线建基线、下线核对。
func (s *Server) entryCheckRoleData(vl charge.VarList, entry bool) bool {
	if len(vl) < 5 {
		s.logf("(SoloGobal::ReceiveCommand)format failed")
		return false
	}
	name := wideVal(vl, 1)
	caps := [3]int64{int64Val(vl, 2), int64Val(vl, 3), int64Val(vl, 4)}
	s.logf("(SoloMonitor::MonitorCreateItem),wsName:%s, nAmount0:%d, nAmount1:%d, nAmount2:%d, nRemain:%d",
		name, caps[0], caps[1], caps[2], len(vl)-5)
	for i, v := range caps {
		if entry {
			s.roleData.UpdateCapital(name, i, v)
		} else {
			s.roleData.CheckCapital(name, i, v)
		}
	}
	for i := 5; i < len(vl); i++ {
		v := int64Val(vl, i)
		if entry {
			s.roleData.UpdateItem(name, i-5, v)
		} else {
			s.roleData.CheckItem(name, i-5, v)
		}
	}
	return true
}

// alertTotalCapital 官方 alert_total_capital：每记录 8 元素×N
// （(len-1)%8==0 否则 trace format failed）；GlobalSwitch>0 → 全局
// 资金不平衡 type 0 告警（注意官方文案 #5 离线值不进告警）。
func (s *Server) alertTotalCapital(vl charge.VarList) bool {
	if (len(vl)-1)%8 != 0 {
		s.logf("(SoloGobal::ReceiveCommand)format failed")
		return false
	}
	for i := 1; i+7 < len(vl); i += 8 {
		capType := vl.Int(i)
		flag := vl.Int(i + 1)
		sys := int64Val(vl, i+2)
		canBuy := int64Val(vl, i+3)
		canPrize := int64Val(vl, i+4)
		userCur := int64Val(vl, i+5)
		onway := int64Val(vl, i+7)
		s.logf("alert_total_capital type:%d flag:%d sys:%d canBuy:%d canPrize:%d userCur:%d onway:%d",
			capType, flag, sys, canBuy, canPrize, userCur, onway)
		if s.globalSwitchVal() > 0 {
			sum := canBuy + canPrize + userCur + onway
			s.onReportWarning(0, "S", fmt.Sprintf(
				"全局资金不平衡! 资金类型CptialType:%d 开关状态:%d 系统总可发行资金:%d 玩家可以购买的资金:%d 系统可以用来奖励的资金:%d 所有玩家当前拥有资金:%d 流通中货币:%d 实际总量:%d",
				capType, flag, sys, canBuy, canPrize, userCur, onway, sum))
		}
	}
	return true
}

// alertTotalCapitalNew 官方 alert_total_capital_new：子型 1..7 七种
// 告警（文案按反汇编片段重构）。
func (s *Server) alertTotalCapitalNew(vl charge.VarList) bool {
	if len(vl) < 5 {
		s.logf("(SoloGobal::ReceiveCommand)format failed")
		return false
	}
	subType := vl.Int(1)
	capType := vl.Int(2)
	cur := int64Val(vl, 3)
	limit := int64Val(vl, 4)
	s.logf("alert_total_capital_new sub:%d type:%d cur:%d limit:%d", subType, capType, cur, limit)
	var msg string
	switch subType {
	case 1:
		msg = fmt.Sprintf("系统资金小于0! 资金类型CptialType%d", capType)
	case 2:
		msg = fmt.Sprintf("可购买资金小于阀值! 资金类型CptialType:%d 当期值:%d 阀值:%d", capType, cur, limit)
	case 3:
		msg = fmt.Sprintf("可奖励资金小于阀值! 资金类型CptialType:%d 当期值:%d 阀值:%d", capType, cur, limit)
	case 4:
		msg = fmt.Sprintf("离线资金小于阀值! 资金类型CptialType:%d 当期值:%d 阀值:%d", capType, cur, limit)
	case 5:
		msg = fmt.Sprintf("用户资金小于1锭! 资金类型CptialType:%d 当期值:%d", capType, cur)
	case 6:
		msg = fmt.Sprintf("资金不平衡 差额超过1锭! 资金类型CptialType:%d 可购买+奖励+用户保有:%d 总值:%d", capType, cur, limit)
	case 7:
		msg = fmt.Sprintf("在路上的资金小于1锭! 资金类型CptialType:%d 当前值:%d", capType, cur)
	default:
		return false
	}
	s.onReportWarning(0, "S", msg)
	return true
}

// onReportWarning 官方 OnReportWarning@0x100a3960：trace（有当前
// client 带连接地址）→ CAccess 入队 → alert_monitor echo 广播。
func (s *Server) onReportWarning(typ int, role, reason string) {
	s.mu.Lock()
	s.reportWarningLocked(typ, role, reason)
	s.mu.Unlock()
}

// reportWarningLocked onReportWarning 主体（调用方持 s.mu——官方单
// 线程 EventLoop 语义下无重入问题；Go 侧 TimerCheck 持锁巡检路径复用）。
func (s *Server) reportWarningLocked(typ int, role, reason string) {
	addr := ""
	if c, ok := s.clients[s.clientIdx]; ok {
		addr = c.addr
	}
	logType := s.nLogType
	if addr != "" {
		s.logf("[Warning],role:%s type:%d conn addr:%s logtype:%d  %s", role, typ, addr, logType, reason)
	} else {
		s.logf("[Warning],role:%s type:%d logtype:%d  %s", role, typ, logType, reason)
	}
	if s.access != nil {
		s.access.PutMessage(Alert{Type: typ, Name: role, Reason: reason})
	}
	vl := charge.VarList{}
	vl.AddString("alert_monitor")
	vl.AddWideStr(role)
	vl.AddInt(typ)
	vl.AddString(reason)
	s.echo.SendLine(vl.EncodeLine())
}

// TimerCheck 官方 TimerCheck@0x100a30d0（1.0s）：
// ①每 6 拍 AppRecord 状态行（Go=日志）②状态机 tick ③
// PlayerDiffCapitalLimit>0 时每 61 拍 8 槽资金差巡检（type 3）。
func (s *Server) TimerCheck() {
	s.mu.Lock()
	s.tickTotal++
	if s.opt.UseAppRecord {
		s.tickCount++
		if s.tickCount > 5 {
			s.tickCount = 0
			s.logf("%s", s.statusLineLocked())
		}
	}
	if s.diffLimit > 0 {
		s.capitalTick++
		if s.capitalTick > 0x3C {
			s.capitalTick = 0
			texts := [8]string{
				"玩家间强化装备资金差值超过阀值! 当前差值:%d 阀值:%d",
				"玩家间直接交易的资金差值超过阀值! 当前差值:%d 阀值:%d",
				"玩家间乞讨的资金差值超过阀值! 当前差值:%d 阀值:%d",
				"玩家间乞讨的资金差值超过阀值! 当前差值:%d 阀值:%d", // 官方文案重复，原文如此
				"玩家间画像的资金差值超过阀值! 当前差值:%d 阀值:%d",
				"玩家间书生服务的资金差值超过阀值! 当前差值:%d 阀值:%d",
				"玩家间摆摊中卖道具的资金差值超过阀值! 当前差值:%d 阀值:%d",
				"合成服务的资金差值超过阀值! 当前差值:%d 阀值:%d",
			}
			for i, text := range texts {
				d := s.diff[i]
				if d < 0 {
					d = -d
				}
				if d > s.diffLimit {
					s.reportWarningLocked(3, "S", fmt.Sprintf(text, s.diff[i], s.diffLimit))
				}
			}
		}
	}
	s.mu.Unlock()
	s.healDatabaseBreak()
	s.sm.Dispatch(eventTick, 0, 0)
}

// healDatabaseBreak Go 扩展：DatabaseBreak 期间逐拍探活，探通即摘掉
// Break（官方只有 echo 的 ClearDatabaseError 能清——库重启后状态行会
// 一直挂在 Break 上）。dbErr 是断连期间的累计，保留供事后查。
func (s *Server) healDatabaseBreak() {
	if !s.DatabaseBreak() || s.access == nil {
		return
	}
	if err := s.access.Ping(); err != nil {
		return
	}
	s.mu.Lock()
	s.dbBreak = false
	s.mu.Unlock()
	s.logf("(CAccess::CheckConnect)database recovered")
}

// StatusLine 呈现官方 `<info …/>` 状态行（测试用）。
func (s *Server) StatusLine() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statusLineLocked()
}

func (s *Server) statusLineLocked() string {
	db := "OK"
	if s.dbBreak {
		db = "Break"
	}
	return fmt.Sprintf(`<info state="%s" database="%s" db_err="%d" tick="%d" />`,
		s.sm.State().String(), db, s.dbErr, tickCount32())
}

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

// DatabaseBreak/DatabaseError 呈现面（CLockVal +0x354/+0x358）。
func (s *Server) DatabaseBreak() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dbBreak
}

func (s *Server) DatabaseError() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dbErr
}

func (s *Server) globalSwitchVal() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.opt.GlobalSwitch
}

// State 当前状态机态（官方 GetState）。
func (s *Server) State() State { return s.sm.State() }

// ConnCount 外网受面现行连接数（官方 +0x334）。
func (s *Server) ConnCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.total
}

// SetGlobalSwitch 官方 SetGlobalSwitch@0x100a61f0（echo/属性面）。
func (s *Server) SetGlobalSwitch(v int) {
	s.mu.Lock()
	s.opt.GlobalSwitch = v
	s.mu.Unlock()
}

// SetPlayerDiffCapitalLimit 官方同名（echo/属性面）。
func (s *Server) SetPlayerDiffCapitalLimit(v int64) {
	s.mu.Lock()
	s.diffLimit = v
	s.mu.Unlock()
}

// SetTableCapacity 官方 SetTableCapacity@0x1009f8d0（7 注册名分派）。
func (s *Server) SetTableCapacity(name string, cap int) bool {
	switch name {
	case "CreateItem":
		s.createItem.SetCapacity(cap)
	case "CreateItemDetail":
		s.createItemDetail.SetCapacity(cap)
	case "RoleData":
		s.roleData.SetCapacity(cap)
	case "RecvLetterDetail":
		s.recvLetter.SetCapacity(cap)
	case "TradeItemDetail":
		s.tradeItem.SetCapacity(cap)
	case "ItemDetail":
		s.globalItem.SetCapacity(cap)
	case "LoginDetail":
		s.loginDetail.SetCapacity(cap)
	default:
		return false
	}
	return true
}

// SetDomain 官方 SetDomain@0x1009f6f0（8 槽名→表；Go 侧保留装配面）。
func (s *Server) SetDomain(idx int, name, table string) bool { return true }

// echoCommand 官方 main.lua echo_input：quit（关停）/end（批量断开）/
// undefined；Go 扩展 help/state/status/tables。返回（应答，是否批量
// 关闭 echo）。
func (s *Server) echoCommand(line string) (string, bool) {
	cmd := strings.TrimSpace(line)
	switch cmd {
	case "quit":
		s.logf("server will shutdown...")
		go s.Stop()
		return ">server will shutdown...\r", false
	case "end":
		return "", true // 官方 end → CloseEcho（批量断开，无应答）
	case "help":
		return ">quit shutdown; end close echo; state/status info; tables capacities\r", false
	case "state", "status":
		return ">" + s.StatusLine() + "\r", false
	case "tables":
		return fmt.Sprintf(">CreateItem:%d CreateItemDetail:%d RoleData:%d RecvLetterDetail:%d TradeItemDetail:%d ItemDetail:%d LoginDetail:%d\r",
			s.createItem.Capacity(), s.createItemDetail.Capacity(), s.roleData.Capacity(),
			s.recvLetter.Capacity(), s.tradeItem.Capacity(), s.globalItem.Capacity(),
			s.loginDetail.Capacity()), false
	default:
		return ">undefined command: " + cmd + "\r", false
	}
}

// tickCount32 官方 Port_GetTickCount 呈现等价（毫秒）。
func tickCount32() int {
	return int(time.Now().UnixMilli() & 0xFFFFFFFF)
}

// wideVal 官方 WideStrVal(i)：宽串元素取值；对窄串宽容取 S。
func wideVal(vl charge.VarList, i int) string {
	if i < 0 || i >= len(vl) {
		return ""
	}
	return vl[i].S
}

// int64Val 官方 Int64Val：int/int64 域归一。
func int64Val(vl charge.VarList, i int) int64 {
	if i < 0 || i >= len(vl) {
		return 0
	}
	return vl[i].I
}
