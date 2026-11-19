package fssologlobal

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/charge"
)

// officialGameGlobalINI 是官方快照 game_global.ini 原文（方言回归
// 基准；关心的键均 ASCII，GBK 环境同样可解析）。
const officialGameGlobalINI = `[System]
// 循环休眠毫秒数（小于0则不休眠）
Sleep=-1
// 主实体名称
MainEntity=SoloGlobal
// 启动脚本名称
MainScript=main

// 指定加载的组件模块（为空则全部加载）
[Module]
fskit.dll
fssql.dll
fscomm.dll
fssologlobal.dll

// 脚本文件
[Script]
// 所在路径
Path=..\lua\sologlobal
// 是否即时重新加载
Reload=1

[Main]
// 数据库配置
ConnectString=Provider=SQLOLEDB;Password=abc;User ID=sa;Initial Catalog=nx_base1;Data Source=192.168.33.70

// 外网IP地址
[Outer]
Addr=192.168.33.70
Port=4005
`

func TestParseOfficialINI(t *testing.T) {
	cfg, err := ParseGameGlobalINI([]byte(officialGameGlobalINI))
	if err != nil {
		t.Fatalf("解析官方原文失败：%v", err)
	}
	if cfg.MainEntity != "SoloGlobal" || cfg.MainScript != "main" {
		t.Errorf("MainEntity/MainScript = %q/%q", cfg.MainEntity, cfg.MainScript)
	}
	if !cfg.IsOLEDB() {
		t.Errorf("官方 OLEDB 连接串应识别为 IsOLEDB")
	}
	wantModules := []string{"fskit.dll", "fssql.dll", "fscomm.dll", "fssologlobal.dll"}
	if len(cfg.Modules) != len(wantModules) {
		t.Fatalf("[Module] 列表 = %v，想要 %v", cfg.Modules, wantModules)
	}
	for i, m := range wantModules {
		if cfg.Modules[i] != m {
			t.Errorf("[Module][%d] = %q，想要 %q", i, cfg.Modules[i], m)
		}
	}
	if cfg.ScriptPath != `..\lua\sologlobal` {
		t.Errorf("ScriptPath = %q", cfg.ScriptPath)
	}
	if cfg.OuterAddr != "192.168.33.70" || cfg.OuterPort != 4005 {
		t.Errorf("Outer = %q:%d（快照部署值）", cfg.OuterAddr, cfg.OuterPort)
	}
	// 官方文件无 [Echo]：落到 main.lua 本地缺省
	if cfg.EchoAddr != "127.0.0.1" || cfg.EchoPort != 15012 {
		t.Errorf("Echo 缺省 = %q:%d，想要 127.0.0.1:15012", cfg.EchoAddr, cfg.EchoPort)
	}
}

func TestParseINIDefaults(t *testing.T) {
	// 空 [Outer] → main.lua 缺省 127.0.0.1:4003
	cfg, err := ParseGameGlobalINI([]byte("[Main]\nConnectString=x\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OuterAddr != "127.0.0.1" || cfg.OuterPort != 4003 {
		t.Errorf("Outer 缺省 = %q:%d", cfg.OuterAddr, cfg.OuterPort)
	}
	// 注释方言：// ; # 三形整行注释
	cfg, err = ParseGameGlobalINI([]byte(
		"; c1\n# c2\n// c3\n[Outer]\nAddr=10.0.0.1\nPort=5001\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OuterAddr != "10.0.0.1" || cfg.OuterPort != 5001 {
		t.Errorf("Outer = %q:%d", cfg.OuterAddr, cfg.OuterPort)
	}
	// 非法端口
	if _, err := ParseGameGlobalINI([]byte("[Outer]\nPort=99999\n")); err == nil {
		t.Errorf("非法端口应报错")
	}
}

func TestEmbeddedDefaultINI(t *testing.T) {
	cfg, err := ParseGameGlobalINI(defaultGameGlobalINI)
	if err != nil {
		t.Fatalf("内置默认解析失败：%v", err)
	}
	if cfg.MainEntity != "SoloGlobal" {
		t.Errorf("MainEntity = %q", cfg.MainEntity)
	}
	if cfg.OuterAddr != "127.0.0.1" || cfg.OuterPort != 4003 {
		t.Errorf("内置默认 Outer = %q:%d（应回环可用）", cfg.OuterAddr, cfg.OuterPort)
	}
	if cfg.EchoPort != 15012 {
		t.Errorf("内置默认 EchoPort = %d", cfg.EchoPort)
	}
}

func TestLoadGameGlobalINICreateAndReuse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "game_global.ini")
	_, data1, created, err := LoadGameGlobalINI(path)
	if err != nil || !created {
		t.Fatalf("首跑应释放默认：%v created=%v", err, created)
	}
	// 手工改写后复用：绝不覆盖
	overlay := strings.Replace(string(data1), "Port=4003", "Port=4100", 1)
	if err := os.WriteFile(path, []byte(overlay), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _, created, err := LoadGameGlobalINI(path)
	if err != nil || created {
		t.Fatalf("二跑应复用现存文件：%v created=%v", err, created)
	}
	if cfg.OuterPort != 4100 {
		t.Errorf("改写端口 = %d，想要 4100", cfg.OuterPort)
	}
}

func TestDomainTable(t *testing.T) {
	d := DefaultDomains()
	if got := d.Lookup("role"); got != "nx_role_names" {
		t.Errorf("role → %q", got)
	}
	if got := d.Lookup("guild"); got != "nx_guild_names" {
		t.Errorf("guild → %q", got)
	}
	if got := d.Lookup("team"); got != "" {
		t.Errorf("未知域应返回空串，得 %q", got)
	}
	// 8 槽边界
	if d.Set(8, "x", "y") {
		t.Errorf("index=8 应拒绝")
	}
	if !d.Set(7, "x", "y") {
		t.Errorf("index=7 应接受")
	}
	if got := d.Lookup("x"); got != "y" {
		t.Errorf("x → %q", got)
	}
	// 0x24 定长截断
	long := strings.Repeat("a", 0x30)
	d.Set(0, long, long)
	if n := d.Lookup(strings.Repeat("a", 0x23)); n != strings.Repeat("a", 0x23) {
		t.Errorf("超长字段应截断到 0x23 字节")
	}
	// 覆盖槽位
	d.Set(0, "role2", "t2")
	if d.Lookup("role") != "" || d.Lookup("role2") != "t2" {
		t.Errorf("槽位覆盖语义错误")
	}
}

func TestStateMachine(t *testing.T) {
	var traces []string
	sm := NewStateMachine(func(f string, a ...any) { traces = append(traces, fmt.Sprintf(f, a...)) })
	if sm.State() != StateUnknown || StateUnknown.String() != "Unknown" {
		t.Fatalf("初始态应为 Unknown")
	}
	// tick 在 Unknown 被接受但不迁移
	if !sm.Dispatch(eventTick, 0, 0) || sm.State() != StateUnknown {
		t.Fatalf("Unknown tick 语义错误")
	}
	// 非法事件
	if sm.Dispatch(9, 0, 0) {
		t.Errorf("非法事件应拒绝")
	}
	// start → Opened
	if !sm.Dispatch(eventStart, 0, 0) || sm.State() != StateOpened {
		t.Fatalf("start 应迁移 Opened")
	}
	if StateOpened.String() != "Opened" {
		t.Errorf("Opened 呈现名")
	}
	// Opened：tick 接受；start 拒绝
	if !sm.Dispatch(eventTick, 0, 0) {
		t.Errorf("Opened tick 应接受")
	}
	if sm.Dispatch(eventStart, 0, 0) {
		t.Errorf("Opened start 应拒绝")
	}
	if len(traces) == 0 || !strings.Contains(traces[len(traces)-1], "error input 1") {
		t.Errorf("应 trace error input，得 %v", traces)
	}
	// Closed 态（直接构造）
	sm2 := NewStateMachine(nil)
	sm2.Dispatch(eventStart, 0, 0)
	if !sm2.Dispatch(eventTick, 0, 0) {
		t.Errorf("Closed tick 应接受")
	}
	if sm2.Dispatch(2, 0, 0) {
		t.Errorf("Closed 非法事件应拒绝")
	}
}

// wire 形状测试：官方 ReceiveCommand 的请求/应答 VarList 编码
// （charge 文本方言：$ 窄串、# 宽串、int 裸十进制）。
func TestWireShapes(t *testing.T) {
	req := charge.VarList{}
	req.AddString("seq1")
	req.AddString("add_name")
	req.AddString("role")
	req.AddWideStr("张三")
	req.AddWideStr("李四")
	req.AddInt(7)
	line := req.EncodeLine()
	// 宽串按 UTF-16 码元转义（charge 方言），窄串 $ 前缀
	if !strings.HasPrefix(line, "$seq1 $add_name $role #") {
		t.Errorf("请求行形状 = %q", line)
	}
	back, err := charge.DecodeLine(line)
	if err != nil {
		t.Fatal(err)
	}
	if back.StringVal(0) != "seq1" || back.StringVal(1) != "add_name" ||
		back.StringVal(2) != "role" || back.Int(5) != 7 {
		t.Errorf("往返失真：%v", back)
	}
	if back[3].S != "张三" || back[3].Kind != charge.KindWideString {
		t.Errorf("宽串往返失真：%+v", back[3])
	}
}

func TestLocalStoreSemantics(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "names.json")
	s, err := NewLocalStore(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	// AddName：名不存在才登记
	if ok, _ := s.AddName("nx_role_names", "张三", "李四", 7); !ok {
		t.Errorf("首登应成功")
	}
	if ok, _ := s.AddName("nx_role_names", "张三", "别人", 8); ok {
		t.Errorf("重名应失败")
	}
	if s.Count("nx_role_names") != 1 {
		t.Errorf("登记数 = %d", s.Count("nx_role_names"))
	}
	// RemoveName：名存在才释放
	if ok, _ := s.RemoveName("nx_role_names", "不存在"); ok {
		t.Errorf("释放不存在的名应失败")
	}
	if ok, _ := s.RemoveName("nx_role_names", "张三"); !ok {
		t.Errorf("释放存在的名应成功")
	}
	if s.Count("nx_role_names") != 0 {
		t.Errorf("释放后登记数 = %d", s.Count("nx_role_names"))
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	// 快照往返
	s2, err := NewLocalStore(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := s2.AddName("nx_role_names", "王五", "赵六", 9); !ok {
		t.Fatal("快照装载后新登记应成功")
	}
	if ok, _ := s2.AddName("nx_guild_names", "天下会", "帮主", 1); !ok {
		t.Fatal("guild 域登记")
	}
	_ = s2.Close()
	s3, err := NewLocalStore(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s3.Count("nx_role_names") != 1 || s3.Count("nx_guild_names") != 1 {
		t.Errorf("二次快照往返失真：%d/%d", s3.Count("nx_role_names"), s3.Count("nx_guild_names"))
	}
}

// ---- 真 TCP 全链 ----

// testServerFixed 起本地名册后端的全链服（固定空闲端口，ManualTick
// 手动驱动定时面）。
func testServerFixed(t *testing.T) (*Server, string, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := NewLocalStore(filepath.Join(dir, "names.json"), nil)
	if err != nil {
		t.Fatal(err)
	}
	outerPort, echoP := freePort(t), freePort(t)
	s := New(Options{
		OuterAddr:  "127.0.0.1",
		OuterPort:  outerPort,
		EchoAddr:   "127.0.0.1",
		EchoPort:   echoP,
		Store:      store,
		Logf:       func(string, ...any) {},
		ManualTick: true,
	})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	return s, fmt.Sprintf("127.0.0.1:%d", outerPort), fmt.Sprintf("127.0.0.1:%d", echoP)
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// outerClient 是外网命令通道测试客户端（行读，\n 终止）。
type outerClient struct {
	conn net.Conn
	rd   *bufio.Reader
}

func dialOuter(t *testing.T, addr string) *outerClient {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return &outerClient{conn: conn, rd: bufio.NewReader(conn)}
}

func (c *outerClient) send(line string) {
	c.conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := c.conn.Write([]byte(line + "\r\n")); err != nil {
		panic(err)
	}
}

// roundtrip 发一条命令并读一帧应答（到 \n）。
func (c *outerClient) roundtrip(line string) string {
	c.send(line)
	return c.readLine()
}

func (c *outerClient) readLine() string {
	c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	line, err := c.rd.ReadString('\n')
	if err != nil {
		panic(err)
	}
	return strings.TrimRight(line, "\r\n")
}

// expectReply 断言应答帧 = [seq, cmd, …] 形状并返回解码结果。
func expectReply(t *testing.T, frame string) charge.VarList {
	t.Helper()
	vl, err := charge.DecodeLine(frame)
	if err != nil {
		t.Fatalf("应答解码失败 %q：%v", frame, err)
	}
	return vl
}

func TestServerOuterFullChain(t *testing.T) {
	_, outer, _ := testServerFixed(t)
	c := dialOuter(t, outer)

	// add_name 首登 → 应答 [seq, add_name, 表, 名(宽), 1]
	req := charge.VarList{}
	req.AddString("s1")
	req.AddString("add_name")
	req.AddString("role")
	req.AddWideStr("张三")
	req.AddWideStr("李四")
	req.AddInt(7)
	vl := expectReply(t, c.roundtrip(req.EncodeLine()))
	if vl.StringVal(0) != "s1" || vl.StringVal(1) != "add_name" {
		t.Fatalf("应答头 = %q/%q", vl.StringVal(0), vl.StringVal(1))
	}
	if vl.StringVal(2) != "nx_role_names" {
		t.Errorf("应答表名 = %q", vl.StringVal(2))
	}
	if vl[3].S != "张三" || vl[3].Kind != charge.KindWideString {
		t.Errorf("应答名 = %+v", vl[3])
	}
	if vl.Int(4) != 1 {
		t.Errorf("首登结果 = %d", vl.Int(4))
	}

	// 重名 → 0
	vl = expectReply(t, c.roundtrip(req.EncodeLine()))
	if vl.Int(4) != 0 {
		t.Errorf("重名结果 = %d", vl.Int(4))
	}

	// guild 域独立表
	req2 := charge.VarList{}
	req2.AddString("s2")
	req2.AddString("add_name")
	req2.AddString("guild")
	req2.AddWideStr("天下会")
	req2.AddWideStr("帮主")
	req2.AddInt(1)
	vl = expectReply(t, c.roundtrip(req2.EncodeLine()))
	if vl.StringVal(2) != "nx_guild_names" || vl.Int(4) != 1 {
		t.Errorf("guild 域应答失真：%v", vl)
	}

	// remove_name 存在 → 1；不存在 → 0
	rm := charge.VarList{}
	rm.AddString("s3")
	rm.AddString("remove_name")
	rm.AddString("role")
	rm.AddWideStr("张三")
	vl = expectReply(t, c.roundtrip(rm.EncodeLine()))
	if vl.Int(4) != 1 {
		t.Errorf("释放存在名结果 = %d", vl.Int(4))
	}
	vl = expectReply(t, c.roundtrip(rm.EncodeLine()))
	if vl.Int(4) != 0 {
		t.Errorf("释放不存在名结果 = %d", vl.Int(4))
	}
	// 释放后可再登
	vl = expectReply(t, c.roundtrip(req.EncodeLine()))
	if vl.Int(4) != 1 {
		t.Errorf("释放后再登结果 = %d", vl.Int(4))
	}
}

func TestServerOuterErrorPaths(t *testing.T) {
	_, outer, _ := testServerFixed(t)
	c := dialOuter(t, outer)

	// 未知命令：应答仅 [seq, cmd]（官方空结果同构回包）
	q := charge.VarList{}
	q.AddString("s9")
	q.AddString("bogus_cmd")
	vl := expectReply(t, c.roundtrip(q.EncodeLine()))
	if len(vl) != 2 || vl.StringVal(1) != "bogus_cmd" {
		t.Errorf("未知命令应答 = %v", vl)
	}

	// 参数不足：无应答（官方 trace 后 return false）
	short := "$sA $add_name $role"
	c.send(short)
	if reply, ok := tryReadLine(c); ok {
		t.Errorf("参数不足不应回包，得 %q", reply)
	}

	// 未知域：无应答
	noDomain := charge.VarList{}
	noDomain.AddString("sB")
	noDomain.AddString("add_name")
	noDomain.AddString("nosuch")
	noDomain.AddWideStr("x")
	noDomain.AddWideStr("y")
	noDomain.AddInt(1)
	c.send(noDomain.EncodeLine())
	if reply, ok := tryReadLine(c); ok {
		t.Errorf("未知域不应回包，得 %q", reply)
	}

	// 消息过短：无应答
	c.send("$onlyone")
	if reply, ok := tryReadLine(c); ok {
		t.Errorf("过短消息不应回包，得 %q", reply)
	}

	// 非法 token：无应答（decode error）
	c.send("$s $add_name 12x34")
	if r2, ok := tryReadLine(c); ok {
		t.Errorf("解码失败不应回包，得 %q", r2)
	}

	// 解码失败后连接仍可用（官方行为：return false 不断链）
	vl3 := expectReply(t, c.roundtrip("$sC $bogus"))
	if vl3.StringVal(0) != "sC" {
		t.Errorf("解码失败后连接应存活")
	}
}

// tryReadLine 短超时试探读一帧（ok=false = 无应答）。
func tryReadLine(c *outerClient) (string, bool) {
	c.conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	line, err := c.rd.ReadString('\n')
	if err != nil {
		return "", false
	}
	return strings.TrimRight(line, "\r\n"), true
}

func TestServerLineTerminators(t *testing.T) {
	_, outer, _ := testServerFixed(t)
	c := dialOuter(t, outer)
	// 官方 ReceiveClientMsg：\r 与 \n 任一即行终止——连续两命令用
	// 裸 \r 分隔也应各自成帧
	c.conn.SetDeadline(time.Now().Add(2 * time.Second))
	frame := "$s1 $bogus\r$s2 $bogus\r\n"
	if _, err := c.conn.Write([]byte(frame)); err != nil {
		t.Fatal(err)
	}
	first := c.readLine()
	second := c.readLine()
	v1, _ := charge.DecodeLine(first)
	v2, _ := charge.DecodeLine(second)
	if v1.StringVal(0) != "s1" || v2.StringVal(0) != "s2" {
		t.Errorf("\\r 终止语义：%q / %q", first, second)
	}
}

func TestServerOversizeLine(t *testing.T) {
	_, outer, _ := testServerFixed(t)
	c := dialOuter(t, outer)
	// 超过 0x3FF 的行：字节丢弃，行尾截断后仍分派（官方同语义）
	long := strings.Repeat("a", clientLineMax+50)
	req := "$seq $bogus " + long
	vl := expectReply(t, c.roundtrip(req))
	if len(vl) != 2 {
		t.Errorf("超长行截断分派应答 = %v", vl)
	}
}

func TestServerMultiClient(t *testing.T) {
	s, outer, _ := testServerFixed(t)
	c1 := dialOuter(t, outer)
	c2 := dialOuter(t, outer)
	waitCount(t, s, 2) // OnOpen 异步回调，轮询到位
	if n := s.ConnCount(); n != 2 {
		t.Errorf("连接计数 = %d", n)
	}
	// 两客户端互不干扰
	v1 := expectReply(t, c1.roundtrip("$c1 $bogus"))
	v2 := expectReply(t, c2.roundtrip("$c2 $bogus"))
	if v1.StringVal(0) != "c1" || v2.StringVal(0) != "c2" {
		t.Errorf("多客户端串扰：%v %v", v1, v2)
	}
	// 断连计数回落
	c1.conn.Close()
	waitCount(t, s, 1)
	if s.ConnCount() != 1 {
		t.Errorf("断连后计数 = %d", s.ConnCount())
	}
}

// waitCount 轮询连接计数到位（reactor 回调异步）。
func waitCount(t *testing.T, s *Server, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.ConnCount() == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestServerEchoConsole(t *testing.T) {
	_, _, echoAddr := testServerFixed(t)
	conn, err := net.DialTimeout("tcp", echoAddr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	rd := bufio.NewReader(conn)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	welcome, err := rd.ReadString('\r')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(welcome, "welcome global server") {
		t.Errorf("欢迎语 = %q", welcome)
	}
	// state 命令 → `<info …/>` 状态行
	sendRecv := func(line string) string {
		conn.SetDeadline(time.Now().Add(2 * time.Second))
		if _, err := conn.Write([]byte(line + "\r\n")); err != nil {
			t.Fatal(err)
		}
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		resp, err := rd.ReadString('\r')
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	if got := sendRecv("state"); !strings.Contains(got, `<info state="Opened"`) {
		t.Errorf("state 应答 = %q", got)
	}
	// 未定义命令
	if got := sendRecv("frobnicate"); !strings.Contains(got, "undefined command: frobnicate") {
		t.Errorf("未定义命令应答 = %q", got)
	}
	// end → 无应答直接断链（官方 CloseEcho）
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write([]byte("end\r\n")); err != nil {
		t.Fatal(err)
	}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := rd.ReadByte(); err == nil {
		t.Errorf("end 后应断链")
	}
}

func TestServerTimerCheckStatusLine(t *testing.T) {
	s, _, _ := testServerFixed(t)
	// Start 后状态 Opened；TimerCheck tick 不迁移
	for i := 0; i < 7; i++ {
		s.TimerCheck()
	}
	if s.State() != StateOpened {
		t.Errorf("tick 后状态 = %v", s.State())
	}
	line := s.StatusLine()
	if !strings.Contains(line, `state="Opened"`) || !strings.Contains(line, `database="OK"`) {
		t.Errorf("状态行 = %q", line)
	}
	if !strings.Contains(line, `db_err="0"`) || !strings.Contains(line, `tick="`) {
		t.Errorf("状态行字段缺失：%q", line)
	}
}

func TestServerQuitDirect(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStore(filepath.Join(dir, "names.json"), nil)
	if err != nil {
		t.Fatal(err)
	}
	s := New(Options{
		OuterAddr: "127.0.0.1", OuterPort: freePort(t),
		EchoAddr: "127.0.0.1", EchoPort: freePort(t),
		Store: store, Logf: func(string, ...any) {}, ManualTick: true,
	})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	// 官方 quit：echo 回应 "server will shutdown..." + Stop
	if reply, closeConn := s.echoCommand("quit"); !strings.Contains(reply, "server will shutdown") || closeConn {
		t.Errorf("quit 应答 = %q", reply)
	}
	select {
	case <-s.Done():
	case <-time.After(2 * time.Second):
		t.Fatalf("quit 应触发关停")
	}
}
