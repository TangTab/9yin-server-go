package fssololister

import (
	"fmt"
	gonet "net"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ---- codec：solo 方言报文门 + 编解码往返（字段偏移 = 反汇编实证） ----

func TestCodecRoundTrip(t *testing.T) {
	w := WorldInfo{ServerID: 7, IP: "10.1.2.3", Name: "一区·测试", Max: 1000,
		Cur: 250, Visible: true, Recommend: 12}
	pkt := BuildWorldPacket(1, w)
	if len(pkt) != WorldPacketLen {
		t.Fatalf("world 包长 %d，须 0x84", len(pkt))
	}
	if kind, ok := ParseRegistration(pkt, 1); !ok || kind != KindWorld {
		t.Fatalf("world 门未过：kind=%d ok=%v", kind, ok)
	}
	got := ParseWorldInfo(pkt)
	if got != w {
		t.Fatalf("world 往返失真：%+v ≠ %+v", got, w)
	}
	// 魔数字节实证：+4 = 0x3001（LE: 01 30 00 00）
	if pkt[4] != 0x01 || pkt[5] != 0x30 {
		t.Fatalf("魔数字节错位：% x", pkt[4:8])
	}

	e := EntryInfo{ServerID: 7, LineIndex: 3, LineID: 11, IP: "10.1.2.4", Port: 9600, Max: 500, Cur: 100}
	pkt = BuildEntryPacket(1, e)
	if len(pkt) != EntryPacketLen {
		t.Fatalf("entry 包长 %d，须 0x48", len(pkt))
	}
	if kind, ok := ParseRegistration(pkt, 1); !ok || kind != KindEntry {
		t.Fatalf("entry 门未过：kind=%d ok=%v", kind, ok)
	}
	if got := ParseEntryInfo(pkt); got != e {
		t.Fatalf("entry 往返失真：%+v ≠ %+v", got, e)
	}
}

// 配置口静默丢弃清单（全门）+ 本地口中继门（无分区/长度门）。
func TestRegistrationGate(t *testing.T) {
	good := BuildWorldPacket(1, WorldInfo{ServerID: 1, Name: "x", Visible: true})
	cases := []struct {
		name string
		pkt  []byte
	}{
		{"短包", good[:7]},
		{"魔数错", func() []byte { p := clone(good); p[4] = 0xFF; return p }()},
		{"魔数 0x300 大写系", func() []byte {
			p := clone(good)
			p[4], p[5] = 0x00, 0x03
			return p
		}()},
		{"kind 未知", func() []byte { p := clone(good); p[0] = 9; return p }()},
		{"分区错", BuildWorldPacket(2, WorldInfo{ServerID: 1, Visible: true})},
		{"world 长度 0x78 大写系", BuildWorldPacket(1, WorldInfo{ServerID: 1})[:0x78]},
		{"entry 长度错", BuildEntryPacket(1, EntryInfo{ServerID: 1})[:50]},
	}
	for _, c := range cases {
		if _, ok := ParseRegistration(c.pkt, 1); ok {
			t.Fatalf("%s：配置口门未拒绝", c.name)
		}
	}
	// 本地口：分区错/长度错仍可中继（魔数/kind 对即可）
	if _, ok := ParseLoopbackRegistration(BuildWorldPacket(2, WorldInfo{ServerID: 1})); !ok {
		t.Fatal("本地口误拒跨分区包（官方中继无分区门）")
	}
	if _, ok := ParseLoopbackRegistration(good[:0x50]); !ok {
		t.Fatal("本地口误拒短包（长度由目的端再验）")
	}
	if _, ok := ParseLoopbackRegistration(good[:7]); ok {
		t.Fatal("本地口未拒 len<8")
	}
}

func clone(b []byte) []byte { return append([]byte(nil), b...) }

// ---- 表生命周期：world 建服 / entry 建线 / 双心跳除名 / 空槽复用 ----

func newSL() *SoloLister {
	sl := New()
	sl.DistrictID = 1
	return sl
}

func mustWorld(t *testing.T, sl *SoloLister, w WorldInfo) {
	t.Helper()
	if !sl.RecvWorldInfo(w) {
		t.Fatalf("world 注册被拒：%+v", w)
	}
}

func mustEntry(t *testing.T, sl *SoloLister, e EntryInfo) {
	t.Helper()
	if !sl.RecvEntryInfo(e) {
		t.Fatalf("entry 注册被拒：%+v", e)
	}
}

func TestRegistrationLifecycle(t *testing.T) {
	sl := newSL()
	var logs []string
	sl.Logf = func(f string, a ...any) { logs = append(logs, sprintf(f, a...)) }

	// entry 先于 world：不建服（官方 RecvEntryServerInfo 丢弃）
	if sl.RecvEntryInfo(EntryInfo{ServerID: 5, LineIndex: 0}) {
		t.Fatal("entry 先注册不应建服")
	}
	// line_index ≥16 拒收
	mustWorld(t, sl, WorldInfo{ServerID: 5, Name: "五区", IP: "10.0.0.5", Visible: true, Max: 100})
	if sl.RecvEntryInfo(EntryInfo{ServerID: 5, LineIndex: 16}) {
		t.Fatal("line_index≥16 未拒")
	}
	mustEntry(t, sl, EntryInfo{ServerID: 5, LineIndex: 2, LineID: 1, IP: "10.0.0.5", Port: 9600, Max: 100, Cur: 30})

	s := sl.FindServer(5)
	if s == nil || !s.FromWorld || !s.WorldLatch || !s.Visible {
		t.Fatalf("world 注册后状态错：%+v", s)
	}
	if !s.Lines[2].Valid || !s.Lines[2].Latch || s.Lines[2].Max != 100 {
		t.Fatalf("line 注册后状态错：%+v", s.Lines[2])
	}
	if len(logs) != 2 || logs[0] != "add server 五区(10.0.0.5)" ||
		logs[1] != "add line 2 on server 五区 -> 10.0.0.5:9600 (max 100)" {
		t.Fatalf("add 日志方言错：%v", logs)
	}

	// 双心跳：一个扫描周期不重发 world → 闩锁被清，第二个周期除名
	// （槽回收，日志 remove）
	sl.TickSeconds(10)
	if sl.FindServer(5) != nil {
		t.Fatal("world 心跳丢失未除名")
	}
	if len(logs) != 3 || logs[2] != "remove server 五区(10.0.0.5)" {
		t.Fatalf("remove 日志方言错：%v", logs)
	}
	// 同 id 重注册 → 复用空槽（freeSlots 语义）
	mustWorld(t, sl, WorldInfo{ServerID: 6, Name: "六区", IP: "10.0.0.6", Visible: true})
	if sl.FindServer(6) == nil {
		t.Fatal("重注册失败")
	}
}

func TestDoubleHeartbeat(t *testing.T) {
	sl := newSL()
	beatW := func() { mustWorld(t, sl, WorldInfo{ServerID: 3, Name: "w", Visible: true, Max: 10, Cur: 2}) }
	beatE := func() { mustEntry(t, sl, EntryInfo{ServerID: 3, LineIndex: 1, LineID: 0, Max: 10, Cur: 1, Port: 1}) }
	beatW()
	beatE()
	// 只重发 world：线死（entry 心跳丢失一拍即摘线），服活
	for i := 0; i < 3; i++ {
		beatW()
		sl.TickSeconds(5)
	}
	s := sl.FindServer(3)
	if s == nil || s.Lines[1].Valid {
		t.Fatalf("entry 心跳丢失未摘线：%+v", s)
	}
	// 双跳齐发：线服俱活
	beatE()
	beatW()
	sl.TickSeconds(5)
	s = sl.FindServer(3)
	if s == nil || !s.Lines[1].Valid {
		t.Fatal("双心跳仍被除名")
	}
	// 全停：5s 后整服除名
	sl.TickSeconds(5)
	if sl.FindServer(3) != nil {
		t.Fatal("双心跳全停未除名")
	}
}

// ---- svrlist 应答：选线/busy/IDC 过滤/版本门 ----

func TestSelectionAndBusy(t *testing.T) {
	sl := newSL()
	sl.IDC = 0
	mustWorld(t, sl, WorldInfo{ServerID: 1, Name: "甲", Visible: true, Max: 100, Cur: 50, Recommend: 3})
	mustEntry(t, sl, EntryInfo{ServerID: 1, LineIndex: 0, LineID: 1, IP: "10.0.0.1", Port: 1001, Max: 100, Cur: 90})
	mustEntry(t, sl, EntryInfo{ServerID: 1, LineIndex: 1, LineID: 2, IP: "10.0.0.2", Port: 1002, Max: 100, Cur: 10})
	// 不可见服不出现；max==0 线不参选
	mustWorld(t, sl, WorldInfo{ServerID: 2, Name: "乙", Visible: false})
	mustWorld(t, sl, WorldInfo{ServerID: 3, Name: "丙", Visible: true, Max: 100, Cur: 99})
	mustEntry(t, sl, EntryInfo{ServerID: 3, LineIndex: 0, LineID: 1, IP: "10.0.0.3", Port: 1003, Max: 0, Cur: 0})

	es := sl.Entries()
	if len(es) != 1 {
		t.Fatalf("应答条目数 %d ≠ 1（乙不可见/丙无线）", len(es))
	}
	e := es[0]
	if e.Host != "10.0.0.2" || e.Port != 1002 {
		t.Fatalf("未选最小负载线：%+v", e)
	}
	// busy = recommend(3)≤10 → cur*5/max = 50*5/100 = 2
	if e.Busy != 2 {
		t.Fatalf("busy 公式错：%d ≠ 2", e.Busy)
	}

	// recommend>10 直通（solo 独有通道）
	mustWorld(t, sl, WorldInfo{ServerID: 1, Name: "甲", Visible: true, Max: 100, Cur: 50, Recommend: 42})
	if b := sl.Entries()[0].Busy; b != 42 {
		t.Fatalf("recommend>10 未直通：%d", b)
	}

	// 等负载率取先者（官方 solo fcompp 严格小于门，等值不换线）
	mustEntry(t, sl, EntryInfo{ServerID: 1, LineIndex: 2, LineID: 3, IP: "10.0.0.9", Port: 1009, Max: 100, Cur: 10})
	if e := sl.Entries()[0]; e.Host != "10.0.0.2" {
		t.Fatalf("等值应保持先者：%+v", e)
	}
}

func TestIDCLineFilter(t *testing.T) {
	sl := newSL()
	sl.IDC = 5
	mustWorld(t, sl, WorldInfo{ServerID: 1, Name: "甲", Visible: true, Max: 10, Cur: 0})
	mustEntry(t, sl, EntryInfo{ServerID: 1, LineIndex: 0, LineID: 4, IP: "10.0.0.4", Port: 1, Max: 10})
	mustEntry(t, sl, EntryInfo{ServerID: 1, LineIndex: 1, LineID: 5, IP: "10.0.0.5", Port: 2, Max: 10})
	// IDC=5 时仅 line_id==5 的线参选——即使负载率更差的 4 号线存在
	if e := sl.Entries()[0]; e.Host != "10.0.0.5" {
		t.Fatalf("IDC 过滤错：%+v", e)
	}
}

func TestVersionGate(t *testing.T) {
	sl := newSL()
	sl.ClientVersion = 20
	cases := []struct {
		line      string
		pass, low bool
	}{
		{"svrlist 20", true, false},
		{"svrlist 99", true, false},
		{"svrlist 19", false, true},
		{"svrlist", false, false},    // 长度 <8 静默
		{"svrlistx 20", true, false}, // strncmp(7) 语义：前 7 字节匹配即过
		{"svrlist 20x", true, false}, // atoi 截断方言 → 20
	}
	for _, c := range cases {
		pass, low := sl.CheckVersionGate(c.line)
		if pass != c.pass || low != c.low {
			t.Fatalf("%q：got(%v,%v) want(%v,%v)", c.line, pass, low, c.pass, c.low)
		}
	}
	// 无版本门：任意行过门（strcmp 精确匹配在 dispatch 层）
	sl.ClientVersion = 0
	if pass, low := sl.CheckVersionGate("whatever"); !pass || low {
		t.Fatal("无版本门时应恒过")
	}
}

// ---- 客户端面：行方言 + 单请求 + 超时关闭（真 TCP） ----

func startClientServer(t *testing.T, sl *SoloLister, wait, linger time.Duration) (*ClientServer, gonet.Addr) {
	t.Helper()
	cs := &ClientServer{SL: sl, Addr: "127.0.0.1:0", RequestWait: wait, Linger: linger}
	addr, err := cs.Listen()
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = cs.Serve() }()
	t.Cleanup(func() { _ = cs.Close() })
	return cs, addr
}

func readResp(t *testing.T, conn gonet.Conn) string {
	t.Helper()
	buf := make([]byte, 4096)
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	return string(buf[:n])
}

func TestClientProtocolSoloDialect(t *testing.T) {
	sl := newSL()
	mustWorld(t, sl, WorldInfo{ServerID: 1, Name: "本地", Visible: true, Max: 100, Cur: 0})
	mustEntry(t, sl, EntryInfo{ServerID: 1, LineIndex: 0, LineID: 0, IP: "127.0.0.1", Port: 9600, Max: 100})
	_, addr := startClientServer(t, sl, 30*time.Second, 30*time.Second)

	conn, err := gonet.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// solo 方言：\r 也是行终结符（大写系会丢弃）
	_, _ = conn.Write([]byte("svrlist\r"))
	resp := readResp(t, conn)
	want := "$svrlist #\\u672C\\u5730 $127.0.0.1 9600 0\r\n"
	if resp != want {
		t.Fatalf("应答方言错：%q ≠ %q", resp, want)
	}

	// 单请求：应答后的 svrlist 不再理
	_, _ = conn.Write([]byte("svrlist\n"))
	_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	buf := make([]byte, 64)
	if n, err := conn.Read(buf); err == nil {
		t.Fatalf("应答后仍有回应：%q", buf[:n])
	}

	// 空行不发（行缓冲空时分发跳过——官方 len>0 门）
	_, _ = conn.Write([]byte("\r\nsvrlist\r"))
	_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if n, err := conn.Read(buf); err == nil {
		t.Fatalf("重复请求被回应：%q", buf[:n])
	}
}

func TestClientLineBufferDialect(t *testing.T) {
	sl := newSL()
	sl.ClientVersion = 1
	_, addr := startClientServer(t, sl, 30*time.Second, 30*time.Second)
	conn, err := gonet.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// 控制字符不丢弃（大写系丢弃）：0x01 夹在行里使 strcmp 不匹配 → 静默
	_, _ = conn.Write([]byte("sv\x01list 1\r"))
	_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	buf := make([]byte, 256)
	if n, err := conn.Read(buf); err == nil {
		t.Fatalf("控制符入行后应静默：%q", buf[:n])
	}
	// 版本不达标：version_error（首行）
	_, _ = conn.Write([]byte("svrlist 0\r"))
	if got := readResp(t, conn); got != "$version_error\r\n" {
		t.Fatalf("version_error 方言错：%q", got)
	}
	// version_error 也占用单请求位
	_, _ = conn.Write([]byte("svrlist 1\r"))
	_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if n, err := conn.Read(buf); err == nil {
		t.Fatalf("version_error 后不应再答：%q", buf[:n])
	}
}

func TestClientTimeouts(t *testing.T) {
	sl := newSL()
	mustWorld(t, sl, WorldInfo{ServerID: 1, Name: "本地", Visible: true, Max: 10, Cur: 0})
	mustEntry(t, sl, EntryInfo{ServerID: 1, LineIndex: 0, LineID: 0, IP: "127.0.0.1", Port: 9600, Max: 10})
	_, addr := startClientServer(t, sl, 120*time.Millisecond, 120*time.Millisecond)

	// 无请求：30s 期限（此处 120ms）到即断
	conn, err := gonet.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 64)
	if _, err := conn.Read(buf); err == nil {
		t.Fatal("无请求连接未被关闭")
	}

	// 应答后：5s 期限（120ms）滞留关闭
	conn2, err := gonet.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close()
	_, _ = conn2.Write([]byte("svrlist\n"))
	if got := readResp(t, conn2); !strings.HasPrefix(got, "$svrlist") {
		t.Fatalf("应答错：%q", got)
	}
	_ = conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn2.Read(buf); err == nil {
		t.Fatal("应答滞留连接未被关闭")
	}
	waitFor(t, 2*time.Second, func() bool { return sl.ClientCount() == 0 })
}

// ---- IP 黑白名单 ----

func TestBlockIP(t *testing.T) {
	sl := newSL()
	if sl.IsBlockIP("1.2.3.4") {
		t.Fatal("默认 -1 不拦")
	}
	if !sl.AddBlockIP("10.0.0.1", "10.0.0.9") || !sl.AddBlockIP("10.1.0.0", "10.1.255.255") {
		t.Fatal("合法区间被拒")
	}
	if sl.AddBlockIP("bogus", "10.0.0.1") {
		t.Fatal("非法 IP 应拒")
	}
	sl.BlockOrAllow = BlockBlack
	if !sl.IsBlockIP("10.0.0.5") || sl.IsBlockIP("10.2.0.1") {
		t.Fatal("黑名单语义错")
	}
	sl.BlockOrAllow = BlockWhite
	if sl.IsBlockIP("10.0.0.5") || !sl.IsBlockIP("10.2.0.1") {
		t.Fatal("白名单语义错")
	}
	sl.ClearBlockIP()
	if !sl.IsBlockIP("10.2.0.1") {
		t.Fatal("白名单清表后应全拒")
	}
}

func TestBlockIPOnAccept(t *testing.T) {
	sl := newSL()
	sl.AddBlockIP("127.0.0.1", "127.0.0.1")
	sl.BlockOrAllow = BlockBlack
	_, addr := startClientServer(t, sl, time.Second, time.Second)
	conn, err := gonet.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 64)
	if n, err := conn.Read(buf); err == nil || n != 0 {
		t.Fatal("黑名单连接未被立即断开")
	}
	if sl.ClientCount() != 0 {
		t.Fatal("被拦连接不应入表")
	}
}

// ---- UDP 注册面：定向直达 + 本地口中继链路 ----

func TestUDPRegistrationDirect(t *testing.T) {
	sl := newSL()
	sl.InnerAddr = "127.0.0.1"
	sl.BroadPort = freeUDPPort(t)
	r := &RegistrationServer{SL: sl}
	if _, err := r.Listen(); err != nil {
		t.Fatal(err)
	}
	r.Serve()
	t.Cleanup(func() { _ = r.Close() })

	pc, err := gonet.Dial("udp4", r.LocalAddr())
	if err != nil {
		t.Fatal(err)
	}
	// InnerAddr==127.0.0.1 → 不建本地中继口（官方同款条件）
	if r.getLoop() != nil {
		t.Fatal("InnerAddr=127.0.0.1 时不应建本地口")
	}
	w := BuildWorldPacket(1, WorldInfo{ServerID: 9, Name: "直连", IP: "10.9.9.9", Visible: true, Max: 7, Cur: 1})
	if _, err := pc.Write(w); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 2*time.Second, func() bool { return sl.FindServer(9) != nil })
	if s := sl.FindServer(9); s == nil || s.Name != "直连" {
		t.Fatalf("定向注册未入表：%+v", s)
	}
	// 分区错包静默丢弃
	if _, err := pc.Write(BuildWorldPacket(2, WorldInfo{ServerID: 10, Visible: true})); err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	if sl.FindServer(10) != nil {
		t.Fatal("跨分区包被错误接受")
	}
}

// 中继链路：本地口(127.0.0.1)收 → 配置口(127.0.0.2)发往广播目标
// （BroadAddr=127.0.0.2 定向）→ 配置口收包入表。全链 127/8 回环，
// 双平台稳定（官方双口设计本就为此）。
func TestUDPRegistrationRelay(t *testing.T) {
	sl := newSL()
	sl.InnerAddr = "127.0.0.2"
	sl.BroadAddr = "127.0.0.2"
	sl.BroadPort = freeUDPPort(t)
	r := &RegistrationServer{SL: sl}
	if _, err := r.Listen(); err != nil {
		t.Fatal(err)
	}
	r.Serve()
	t.Cleanup(func() { _ = r.Close() })
	if r.getLoop() == nil {
		t.Fatal("InnerAddr≠127.0.0.1 应建本地中继口")
	}

	pc, err := gonet.Dial("udp4", "127.0.0.1:"+itoa(sl.BroadPort))
	if err != nil {
		t.Fatal(err)
	}
	e := BuildEntryPacket(1, EntryInfo{ServerID: 21, LineIndex: 0, LineID: 0, IP: "10.0.0.21", Port: 21, Max: 10, Cur: 1})
	// entry 先发：官方语义 world 建服——中继不改变这点
	if _, err := pc.Write(e); err != nil {
		t.Fatal(err)
	}
	w := BuildWorldPacket(1, WorldInfo{ServerID: 21, Name: "中继", IP: "10.0.0.21", Visible: true, Max: 10, Cur: 1})
	if _, err := pc.Write(w); err != nil {
		t.Fatal(err)
	}
	// 再补 entry（中继后由配置口入表）
	if _, err := pc.Write(e); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 2*time.Second, func() bool {
		s := sl.FindServer(21)
		return s != nil && s.Lines[0].Valid
	})
}

// ---- 状态与杂项 ----

func TestMiscOfficialSemantics(t *testing.T) {
	sl := New()
	if sl.MaxConnects != 8000 || sl.BlockOrAllow != BlockDisable {
		t.Fatalf("ctor 默认值错：%+v", sl)
	}
	for _, id := range []int32{-1, 8, 100} {
		sl.SetListerID(id)
	}
	if sl.ListerID != 0 {
		t.Fatal("ListerID 越界未忽略")
	}
	sl.SetListerID(7)
	if sl.ListerID != 7 {
		t.Fatal("ListerID=7 未接受（官方 0..7）")
	}
	sl.MarkStarted()
	if sl.StateName() != "Paused" {
		t.Fatalf("Start 后状态应 Paused：%s", sl.StateName())
	}
	if got := atoiOfficial(" 42abc"); got != 42 {
		t.Fatalf("atoi 方言错：%d", got)
	}
	if got := atoiOfficial(""); got != 0 {
		t.Fatalf("atoi 空串方言错：%d", got)
	}
}

// ---- 小工具 ----

func sprintf(f string, a ...any) string { return fmt.Sprintf(f, a...) }

func itoa(n int) string { return strconv.Itoa(n) }

func waitFor(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("等待条件超时")
}

func freeUDPPort(t *testing.T) int {
	t.Helper()
	pc, err := gonet.ListenUDP("udp4", &gonet.UDPAddr{IP: gonet.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	port := pc.LocalAddr().(*gonet.UDPAddr).Port
	_ = pc.Close()
	return port
}
