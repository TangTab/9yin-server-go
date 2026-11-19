package fssolomonitor

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/charge"
	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// ---- 配置回归 ----

func TestParseGameMonitorINI(t *testing.T) {
	cfg, err := ParseGameMonitorINI(defaultGameMonitorINI)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.MainEntity != "SoloMonitor" || cfg.OuterPort != 4008 || cfg.EchoPort != 15012 {
		t.Fatalf("ini 投影异常: %+v", cfg)
	}
	if cfg.AppName() != "monitor_0" {
		t.Fatalf("AppName=%s", cfg.AppName())
	}
	// 官方原 ini（含 fssql.dll/OLEDB 串）亦可解析。
	official := "[System]\nSleep=-1\nMainEntity=SoloMonitor\nMainScript=main\n[Module]\nfskit.dll\nfssql.dll\nfscomm.dll\nFsSoloMonitor.dll\n" +
		"[Script]\nPath=..\\lua\\solomonitor\n[Main]\nConnectString=Provider=SQLOLEDB;Password=abc&4621;User ID=sa;Initial Catalog=nx_base1;Data Source=192.168.10.148:3306\n" +
		"[Outer]\nAddr=127.0.0.1\nPort=4008\n"
	cfg, err = ParseGameMonitorINI([]byte(official))
	if err != nil {
		t.Fatalf("official parse: %v", err)
	}
	if !cfg.IsOLEDB() || len(cfg.Modules) != 4 || cfg.OuterPort != 4008 {
		t.Fatalf("official 投影异常: %+v", cfg)
	}
}

// TestParseMonitorLimitINI 官方快照数值回归 + 大小写失配保持：
// [System] 五键中只有 CheckTimerInterval/FaculityLimit 被消费；
// Livepoint/DiffCapital 等走 [system]（不存在）→ 代码缺省。
func TestParseMonitorLimitINI(t *testing.T) {
	lc, err := ParseMonitorLimitINI(defaultMonitorLimitINI)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if lc.CheckTimerInterval != 60 {
		t.Errorf("CheckTimerInterval=%d want 60（[System] 生效）", lc.CheckTimerInterval)
	}
	if lc.FaculityLimit != 900000 {
		t.Errorf("FaculityLimit=%d want 900000", lc.FaculityLimit)
	}
	if lc.LivepointLimit != 1000 {
		t.Errorf("LivepointLimit=%d want 1000（[system].FaculityLimit 不存在→缺省）", lc.LivepointLimit)
	}
	if lc.DiffCapitalLimit != 15000000 {
		t.Errorf("DiffCapitalLimit=%d want 15000000（ini 1500000 在大写节，恒缺省）", lc.DiffCapitalLimit)
	}
	if lc.ExchangeAmountLimit != 0 || lc.RoleNumLimit != 10 || lc.OnlineRoleNumLimit != 3 {
		t.Errorf("system 五键缺省失配: %+v", lc)
	}
	if lc.SysItemLimits["ng_cjbook_sl_002"] != 20 || lc.SysItemLimits["cjbook_CS_sl_lhq"] != 40 {
		t.Errorf("SysItemLimits 异常: %v", lc.SysItemLimits)
	}
	if lc.RoleItemLimits["book_CS_sl_lhq01"] != 2 || lc.RoleItemLimits["growitem_1"] != 500 {
		t.Errorf("RoleItemLimits 异常: %v", lc.RoleItemLimits)
	}
	if lc.RoleCapitalLimits[0] != 1000 || lc.RoleCapitalLimits[1] != 3000000 || lc.RoleCapitalLimits[2] != 50000000 {
		t.Errorf("RoleCapitalLimits 异常: %v", lc.RoleCapitalLimits)
	}
	// 大小写敏感：小写 [system] 节存在时才消费五键。
	ini := "[system]\nDiffCapitalLimit = 42\n[System]\nDiffCapitalLimit = 9999\n"
	lc2, _ := ParseMonitorLimitINI([]byte(ini))
	if lc2.DiffCapitalLimit != 42 {
		t.Errorf("小写节命中=%d want 42", lc2.DiffCapitalLimit)
	}
}

func TestGetCapitalType(t *testing.T) {
	cases := map[string]int{"Capital3": 3, "CapitalType12": 2, "": -1, "CapitalX": 0}
	for in, want := range cases {
		if got := lastCharAtoi(in); got != want {
			t.Errorf("lastCharAtoi(%q)=%d want %d", in, got, want)
		}
	}
}

// ---- 监控表行为 ----

// mockSink 收集告警（测试串行调用）。
type mockSink struct{ alerts []Alert }

func (m *mockSink) Ping() error { return nil }
func (m *mockSink) InsertAlert(_ context.Context, a Alert) error {
	m.alerts = append(m.alerts, a)
	return nil
}
func (m *mockSink) Close() error { return nil }

func newTestServer(t *testing.T) (*Server, *mockSink) {
	t.Helper()
	sink := &mockSink{}
	s := New(Options{
		AppName: "monitor_0", OuterAddr: "127.0.0.1", OuterPort: 0,
		EchoAddr: "", Sink: sink, UseAppRecord: true, ManualTick: true,
		Now: func() time.Time { return time.Unix(1000, 0) },
	})
	// 预注册一个外网 client（测试直调 receiveCommand 用）。
	s.onClientOpen(0, "127.0.0.1", 40000)
	return s, sink
}

func wide(s string) charge.Value { return charge.Value{Kind: charge.KindWideString, S: s} }

// waitAlerts 等待异步 Access 写线程落库至少 n 条（2s 上限）。
func waitAlerts(t *testing.T, sink *mockSink, n int) {
	t.Helper()
	for i := 0; i < 400 && len(sink.alerts) < n; i++ {
		time.Sleep(5 * time.Millisecond)
	}
}

func itemLogVL(role, cfg, uid string, amount int64, logType, maxStack int, srcRole, comment string) charge.VarList {
	return charge.VarList{
		{Kind: charge.KindString, S: "item_log"},
		wide(role), wide(cfg), {Kind: charge.KindString, S: uid},
		{Kind: charge.KindInt64, I: amount}, {Kind: charge.KindInt, I: int64(logType)},
		{Kind: charge.KindInt, I: 0}, {Kind: charge.KindInt, I: int64(maxStack)},
		wide(srcRole), wide(comment), wide(""),
	}
}

// TestRoleDataCapitalWindow 官方窗口清算语义：inc_total 超 limit 告警
// type 0 且清零；下线核对 type 8。
func TestRoleDataCapitalWindow(t *testing.T) {
	s, sink := newTestServer(t)
	s.roleData.SetCapitalIncLimit(1, 100)
	s.roleData.SetCheckTimerInterval(10)
	// 上线基线（expected=login=50）。
	s.roleData.UpdateCapital("张三", 1, 50)
	// +30：首次 inc 的惰性 check（last_check=0→now）把 30 清零（官方
	// 惰性清算边界语义），不告警。
	s.roleData.IncCapital("张三", 1, 30, true)
	if len(sink.alerts) != 0 {
		t.Fatalf("early alert: %+v", sink.alerts)
	}
	// 窗口内（5s<10s）+80 不触发 check；窗口外（11s）再 +80 →
	// check 时累计 160>100 → type 0 且清零。
	s.now = func() time.Time { return time.Unix(1005, 0) }
	s.roleData.IncCapital("张三", 1, 80, true)
	if len(sink.alerts) != 0 {
		t.Fatalf("in-window alert: %+v", sink.alerts)
	}
	s.now = func() time.Time { return time.Unix(1011, 0) }
	s.roleData.IncCapital("张三", 1, 80, true)
	waitAlerts(t, sink, 1)
	if len(sink.alerts) != 1 || sink.alerts[0].Type != 0 {
		t.Fatalf("alerts=%+v", sink.alerts)
	}
	// 清零后同窗口再 +80 不告警。
	s.roleData.IncCapital("张三", 1, 80, true)
	if len(sink.alerts) != 1 {
		t.Fatalf("window not cleared: %+v", sink.alerts)
	}
	// 下线核对偏差 → type 8（expected=50+30+80+80=240 vs val=40）。
	s.roleData.CheckCapital("张三", 1, 40)
	if len(sink.alerts) != 2 || sink.alerts[1].Type != 8 {
		t.Fatalf("check capital alerts=%+v", sink.alerts)
	}
}

// TestItemLogDispatch item_log 全分支（官方 §四参数序）。
func TestItemLogDispatch(t *testing.T) {
	s, sink := newTestServer(t)
	// 系统创建超限：[SysItemLimit] ng_x=2，创建累计超限 → type 1。
	s.applyLimits(LimitConfig{CheckTimerInterval: 3600, SysItemLimits: map[string]int{"ng_x": 2}})
	itemLog := func(role, cfg, uid string, amount int64, logType, maxStack int, srcRole, comment string) bool {
		return s.processItemLog(itemLogVL(role, cfg, uid, amount, logType, maxStack, srcRole, comment))
	}
	// ①普通获得：窗口累计 amount 超 2 → type 1。
	if !itemLog("李四", "ng_x", "uuid-1", 2, 10001, 1, "系统", "") {
		t.Fatal("first item_log rejected")
	}
	if !itemLog("李四", "ng_x", "uuid-2", 2, 10001, 1, "系统", "") {
		t.Fatal("second item_log rejected")
	}
	waitAlerts(t, sink, 1)
	found1 := false
	for _, a := range sink.alerts {
		if a.Type == 1 {
			found1 = true
		}
	}
	if !found1 {
		t.Fatalf("type 1 missing: %+v", sink.alerts)
	}
	// ②重复序列号：同 uuid 再走一发给 CreateItemDetail → type 9。
	sink.alerts = nil
	itemLog("李四", "ng_x", "uuid-1", 1, 10001, 1, "系统", "")
	waitAlerts(t, sink, 2)
	found9 := false
	for _, a := range sink.alerts {
		if a.Type == 9 {
			found9 = true
		}
	}
	if !found9 {
		t.Fatalf("type 9 missing: %+v", sink.alerts)
	}
	// ③交易：maxStack=1 → 同 uuid 重复交易 → type 2。
	sink.alerts = nil
	itemLog("李四", "book_a", "u-trade", 1, 20001, 1, "王五", "")
	itemLog("李四", "book_a", "u-trade", 1, 20001, 1, "王五", "")
	waitAlerts(t, sink, 1)
	found2 := false
	for _, a := range sink.alerts {
		if a.Type == 2 {
			found2 = true
		}
	}
	if !found2 {
		t.Fatalf("type 2 missing: %+v", sink.alerts)
	}
	// ④货币增加：Capital1 +30001 计数（无阀值不告警）。
	sink.alerts = nil
	itemLog("李四", "Capital1", "", 60, 30001, 0, "系统", "")
	if len(sink.alerts) != 0 {
		t.Fatalf("capital inc alerts=%+v", sink.alerts)
	}
	// ⑤邮件附件重复：30010 走 RecvLetter（备注=letterId）。
	itemLog("李四", "gold", "", 5, 30010, 0, "张三", "letter-77")
	itemLog("李四", "gold", "", 5, 30010, 0, "张三", "letter-77")
	waitAlerts(t, sink, 1)
	found4 := false
	for _, a := range sink.alerts {
		if a.Type == 4 && strings.Contains(a.Reason, "letter-77") {
			found4 = true
		}
	}
	if !found4 {
		t.Fatalf("recv letter type 4 missing: %+v", sink.alerts)
	}
	// ⑥边界值落空：官方开区间边界全不处理。
	for _, lt := range []int{10000, 20000, 28000, 30000, 34999, 35000, 36029, 40000, 44999, 45000, 46029} {
		sink.alerts = nil
		itemLog("李四", "Capital1", "u", 1, lt, 0, "系统", "")
		if len(sink.alerts) != 0 {
			t.Errorf("boundary logType %d produced %+v", lt, sink.alerts)
		}
	}
}

// TestLoginDetail 同机器账号数/在线数双告警。
func TestLoginDetail(t *testing.T) {
	s, sink := newTestServer(t)
	s.loginDetail.SetAmountLimit(2)
	s.loginDetail.SetOnlineAmountLimit(1)
	s.loginDetail.Login("MAC-A", "10.0.0.1", "acc1", "角色-acc1")
	if len(sink.alerts) != 0 {
		t.Fatalf("pre alerts=%+v", sink.alerts)
	}
	// 第 2 登录：distinct=2 不超 2；online=2>1 → 0xf。
	s.loginDetail.Login("MAC-A", "10.0.0.1", "acc2", "角色-acc2")
	waitAlerts(t, sink, 1)
	if len(sink.alerts) != 1 || sink.alerts[0].Type != 0xf {
		t.Fatalf("second login alerts=%+v", sink.alerts)
	}
	// 第 3 登录：distinct=3>2 → 0xe；online=3>1 → 0xf。
	s.loginDetail.Login("MAC-A", "10.0.0.1", "acc3", "角色-acc3")
	waitAlerts(t, sink, 3)
	if len(sink.alerts) != 3 {
		t.Fatalf("alerts=%+v", sink.alerts)
	}
	if sink.alerts[1].Type != 0xe || sink.alerts[2].Type != 0xf {
		t.Fatalf("type 顺序异常: %+v", sink.alerts)
	}
	s.loginDetail.LoginOut("MAC-A", "10.0.0.1", "acc3", "角色-acc3")
}

// TestGlobalItemDisabled ItemDetail 注册面默认容量 0=停用。
func TestGlobalItemDisabled(t *testing.T) {
	s, sink := newTestServer(t)
	if s.globalItem.BuyItem("book_a", 999, 10, "李四") {
		t.Fatal("cap=0 应返回 false")
	}
	if len(sink.alerts) != 0 {
		t.Fatalf("cap=0 不应告警: %+v", sink.alerts)
	}
	s.SetTableCapacity("ItemDetail", 4)
	s.globalItem.BuyItem("book_a", 3, 10, "李四")
	s.globalItem.BuyItem("book_a", 9, 10, "李四") // 净 12>10 → type 4
	waitAlerts(t, sink, 1)
	last := sink.alerts[len(sink.alerts)-1]
	if last.Type != 4 || !strings.Contains(last.Reason, "增加") {
		t.Fatalf("buy stack alert=%+v", last)
	}
	s.globalItem.SellItem("book_a", -1, 10, "李四") // n<0 → type 0xc + |11|>10 → type 4
	waitAlerts(t, sink, 3)
	if sink.alerts[1].Type != 0xc || sink.alerts[2].Type != 4 {
		t.Fatalf("sell alerts=%+v", sink.alerts)
	}
}

// TestTradeItemDetail 同玩家同编号重复交易 type 0xd。
func TestTradeItemDetail(t *testing.T) {
	s, sink := newTestServer(t)
	s.tradeItem.BuyItem("book_a", 100, "李四", "王五", "u-1", 10031)
	if len(sink.alerts) != 0 {
		t.Fatalf("first trade alerted: %+v", sink.alerts)
	}
	s.tradeItem.SellItem("book_a", 50, "李四", "王五", "u-2", 20058)
	waitAlerts(t, sink, 1)
	if len(sink.alerts) != 1 || sink.alerts[0].Type != 0xd {
		t.Fatalf("repeat trade alerts=%+v", sink.alerts)
	}
}

// TestRingOverwrite 环表回绕覆盖（官方 cursor 回绕 + Remove 旧键）。
func TestRingOverwrite(t *testing.T) {
	// SetCapacity 官方不缩容（n<=cap 忽略），独立建小环表验证回绕。
	tbl := NewCreateItemDetailTable(nil, func(string, int, string) {})
	tbl.SetCapacity(2)
	tbl.Add("cfg", "u1", "r")
	tbl.Add("cfg", "u2", "r")
	tbl.Add("cfg", "u3", "r") // 覆盖 u1
	if tbl.ring.lookup("u1") != nil {
		t.Fatal("u1 应被回绕覆盖")
	}
	if tbl.ring.lookup("u3") == nil {
		t.Fatal("u3 应在环内")
	}
}

// TestDiffCapitalTimer 61 拍资金差巡检（type 3；+0x520=diff#2 直接交易槽）。
func TestDiffCapitalTimer(t *testing.T) {
	s, sink := newTestServer(t)
	s.SetPlayerDiffCapitalLimit(100)
	// 45001（货币增加）净流量进 diff#2。
	s.processItemLog(itemLogVL("r", "Capital1", "", 500, 45001, 0, "sys", ""))
	for i := 0; i < 61; i++ {
		s.TimerCheck()
	}
	waitAlerts(t, sink, 1)
	found := false
	for _, a := range sink.alerts {
		if a.Type == 3 && strings.Contains(a.Reason, "直接交易") {
			found = true
		}
	}
	if !found {
		t.Fatalf("diff capital type 3 missing: %+v", sink.alerts)
	}
}

// TestAlertTotalCapital 格式门 + GlobalSwitch 门 + 文案。
func TestAlertTotalCapital(t *testing.T) {
	s, sink := newTestServer(t)
	// 格式失败：(len-1)%8 != 0。
	bad := charge.VarList{{Kind: charge.KindString, S: "alert_total_capital"}, {Kind: charge.KindInt, I: 1}}
	if s.alertTotalCapital(bad) {
		t.Fatal("format gate failed")
	}
	mk := func() charge.VarList {
		vl := charge.VarList{{Kind: charge.KindString, S: "alert_total_capital"},
			{Kind: charge.KindInt, I: 1}, {Kind: charge.KindInt, I: 0}}
		for _, v := range []int64{100, 30, 20, 40, 5, 10} {
			vl = append(vl, charge.Value{Kind: charge.KindInt64, I: v})
		}
		return vl
	}
	if !s.alertTotalCapital(mk()) || len(sink.alerts) != 0 {
		t.Fatal("GlobalSwitch=0 不应告警")
	}
	s.SetGlobalSwitch(1)
	if !s.alertTotalCapital(mk()) {
		t.Fatal("告警失败")
	}
	waitAlerts(t, sink, 1)
	if len(sink.alerts) != 1 {
		t.Fatalf("alerts=%+v", sink.alerts)
	}
	a := sink.alerts[0]
	if a.Type != 0 || a.Name != "S" || !strings.Contains(a.Reason, "CptialType:1") {
		t.Fatalf("alert=%+v", a)
	}
}

// TestAlertTotalCapitalNew 七子型 + default 拒绝。
func TestAlertTotalCapitalNew(t *testing.T) {
	s, sink := newTestServer(t)
	mk := func(sub int) charge.VarList {
		return charge.VarList{{Kind: charge.KindString, S: "alert_total_capital_new"},
			{Kind: charge.KindInt, I: int64(sub)}, {Kind: charge.KindInt, I: 2},
			{Kind: charge.KindInt64, I: 7}, {Kind: charge.KindInt64, I: 9}}
	}
	for sub := 1; sub <= 7; sub++ {
		if !s.alertTotalCapitalNew(mk(sub)) {
			t.Fatalf("subtype %d rejected", sub)
		}
	}
	waitAlerts(t, sink, 7)
	if len(sink.alerts) != 7 {
		t.Fatalf("alerts=%d want 7", len(sink.alerts))
	}
	for _, a := range sink.alerts {
		if a.Type != 0 || a.Name != "S" {
			t.Fatalf("alert=%+v", a)
		}
	}
	if s.alertTotalCapitalNew(mk(8)) {
		t.Fatal("subtype 8 应被拒绝")
	}
}

// TestFacultyCommand faculty > FaculityLimit → type 0。
func TestFacultyCommand(t *testing.T) {
	s, sink := newTestServer(t)
	s.applyLimits(DefaultLimitConfig()) // FaculityLimit=900000
	vl := charge.VarList{{Kind: charge.KindString, S: "faculty"}, wide("张三"), {Kind: charge.KindInt, I: 900001}}
	if !s.receiveCommand(0, vl.EncodeLine()) {
		t.Fatal("faculty rejected")
	}
	waitAlerts(t, sink, 1)
	if len(sink.alerts) != 1 || sink.alerts[0].Type != 0 || !strings.Contains(sink.alerts[0].Reason, "修为值") {
		t.Fatalf("alerts=%+v", sink.alerts)
	}
}

// TestMonitorCreateItemNoop 官方 Debug 基准降级为 no-op（保留表）。
func TestMonitorCreateItemNoop(t *testing.T) {
	s, _ := newTestServer(t)
	s.createItemDetail.Add("cfg", "u1", "r")
	if s.monitorCreateItem("u9", "张三") {
		t.Fatal("官方恒返 false")
	}
	if s.createItemDetail.ring.lookup("u1") == nil {
		t.Fatal("no-op 不应摧毁 CreateItemDetail 表")
	}
}

// TestEntryLeaveScene 上线建基线（无告警）/下线偏差告警（type 8/0xb）。
func TestEntryLeaveScene(t *testing.T) {
	s, sink := newTestServer(t)
	entry := charge.VarList{{Kind: charge.KindString, S: "entry_scene"}, wide("张三"),
		{Kind: charge.KindInt64, I: 100}, {Kind: charge.KindInt64, I: 200}, {Kind: charge.KindInt64, I: 300},
		{Kind: charge.KindInt64, I: 5}, {Kind: charge.KindInt64, I: 6}}
	if !s.receiveCommand(0, entry.EncodeLine()) {
		t.Fatal("entry rejected")
	}
	if len(sink.alerts) != 0 {
		t.Fatalf("entry 不应告警: %+v", sink.alerts)
	}
	// 下线：货币 1 偏差 → type 8；背包 0 偏差（下线只写值，无 0xb）。
	leave := charge.VarList{{Kind: charge.KindString, S: "leave_scene"}, wide("张三"),
		{Kind: charge.KindInt64, I: 100}, {Kind: charge.KindInt64, I: 250}, {Kind: charge.KindInt64, I: 300},
		{Kind: charge.KindInt64, I: 9}, {Kind: charge.KindInt64, I: 6}}
	if !s.receiveCommand(0, leave.EncodeLine()) {
		t.Fatal("leave rejected")
	}
	waitAlerts(t, sink, 1)
	if len(sink.alerts) != 1 || sink.alerts[0].Type != 8 || !strings.Contains(sink.alerts[0].Reason, "CapitalType1") {
		t.Fatalf("leave alerts=%+v", sink.alerts)
	}
	// 再上线：官方 CheckCapital 只读不改基线 → login 仍 200 vs 250 →
	// type 7（上线资金检查异常）后基线重置为 250。
	reentry := charge.VarList{{Kind: charge.KindString, S: "entry_scene"}, wide("张三"),
		{Kind: charge.KindInt64, I: 100}, {Kind: charge.KindInt64, I: 250}, {Kind: charge.KindInt64, I: 300},
		{Kind: charge.KindInt64, I: 9}, {Kind: charge.KindInt64, I: 6}}
	if !s.receiveCommand(0, reentry.EncodeLine()) {
		t.Fatal("re-entry rejected")
	}
	waitAlerts(t, sink, 2)
	if len(sink.alerts) != 2 || sink.alerts[1].Type != 7 {
		t.Fatalf("re-entry alerts=%+v", sink.alerts)
	}
}

// TestEndToEndOuterEcho 真 TCP 端到端：外网行协议命令 → echo 口收到
// alert_monitor 广播；未知命令不断链。
func TestEndToEndOuterEcho(t *testing.T) {
	s, sink := newTestServer(t)
	if err := s.r.Start(); err != nil {
		t.Fatalf("reactor: %v", err)
	}
	id, err := s.r.CreateListener("127.0.0.1", 0, 10, reactor.ConnHandler{
		OnOpen: s.onClientOpen, OnClose: s.onClientClose, OnInput: s.onClientInput,
	})
	if err != nil {
		t.Fatalf("outer listener: %v", err)
	}
	s.mu.Lock()
	s.outerListener = id
	s.mu.Unlock()
	outerPort, _ := s.r.ListenerPort(id)
	if err := s.echo.Start("127.0.0.1", 0, s.logf); err != nil {
		t.Fatalf("echo: %v", err)
	}
	s.echo.mu.Lock()
	echoListener := s.echo.listener
	s.echo.mu.Unlock()
	echoPort, _ := s.r.ListenerPort(echoListener)
	s.sm.Dispatch(eventStart, 0, 0)

	echoConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", echoPort))
	if err != nil {
		t.Fatalf("echo dial: %v", err)
	}
	defer echoConn.Close()
	echoReader := bufio.NewReader(echoConn)
	if line, err := echoReader.ReadString('\n'); err != nil ||
		strings.TrimRight(line, "\r\n") != "welcome monitor server" {
		t.Fatalf("welcome=%q err=%v", line, err)
	}

	outerConn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", outerPort))
	if err != nil {
		t.Fatalf("outer dial: %v", err)
	}
	defer outerConn.Close()
	// 未知命令 → 不断链。
	fmt.Fprintf(outerConn, "nonsense_cmd\r\n")
	// faculty 超限 → 告警。
	vl := charge.VarList{{Kind: charge.KindString, S: "faculty"}, wide("张三"), {Kind: charge.KindInt, I: 99999999}}
	fmt.Fprintf(outerConn, "%s\r\n", vl.EncodeLine())
	// echo 侧应收到 alert_monitor 广播行（方言编码，解码后断言）。
	line, err := echoReader.ReadString('\n')
	if err != nil {
		t.Fatalf("broadcast read: %v", err)
	}
	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, "$alert_monitor") {
		t.Fatalf("echo broadcast=%q", line)
	}
	decoded, err := charge.DecodeLine(line)
	if err != nil {
		t.Fatalf("broadcast decode: %v", err)
	}
	if len(decoded) < 4 || decoded[0].S != "alert_monitor" || decoded[1].S != "S" ||
		decoded[2].I != 0 || !strings.Contains(decoded[3].S, "修为值") {
		t.Fatalf("broadcast decoded=%+v", decoded)
	}
	waitAlerts(t, sink, 1)
	if len(sink.alerts) == 0 {
		t.Fatal("sink 未收到告警")
	}
	s.Stop()
}
