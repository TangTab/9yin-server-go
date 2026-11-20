package fslister

import (
	gonet "net"
	"strings"
	"testing"
	"time"
)

// ---- codec：报文门 + 编解码往返（字段偏移 = 反汇编实证） ----

func TestCodecRoundTrip(t *testing.T) {
	w := WorldInfo{ServerID: 7, IP: "10.1.2.3", Name: "一区·测试", Max: 1000, Cur: 250, Visible: true}
	pkt := BuildWorldPacket(1, w)
	if len(pkt) != worldPacketSize {
		t.Fatalf("world 包长 %d，须 0x78", len(pkt))
	}
	if kind, ok := ParseRegistration(pkt, 1); !ok || kind != KindWorld {
		t.Fatalf("world 门未过：kind=%d ok=%v", kind, ok)
	}
	got := ParseWorldInfo(pkt)
	if got != w {
		t.Fatalf("world 往返失真：%+v ≠ %+v", got, w)
	}

	e := EntryInfo{ServerID: 7, LineIndex: 3, LineID: 11, IP: "10.1.2.4", Port: 9600, Max: 500, Cur: 100}
	pkt = BuildEntryPacket(1, e)
	if len(pkt) != entryPacketSize {
		t.Fatalf("entry 包长 %d，须 0x48", len(pkt))
	}
	if kind, ok := ParseRegistration(pkt, 1); !ok || kind != KindEntry {
		t.Fatalf("entry 门未过：kind=%d ok=%v", kind, ok)
	}
	if got := ParseEntryInfo(pkt); got != e {
		t.Fatalf("entry 往返失真：%+v ≠ %+v", got, e)
	}
}

// 门语义：官方 OnReceive 静默丢弃清单。
func TestRegistrationGate(t *testing.T) {
	good := BuildWorldPacket(1, WorldInfo{ServerID: 1, Name: "x", Visible: true})
	cases := []struct {
		name string
		pkt  []byte
	}{
		{"短包", good[:7]},
		{"魔数错", func() []byte { p := clone(good); p[4] = 0xFF; return p }()},
		{"kind 未知", func() []byte { p := clone(good); p[0] = 9; return p }()},
		{"分区错", BuildWorldPacket(2, WorldInfo{ServerID: 1, Visible: true})},
		{"world 长度错", good[:100]},
		{"entry 长度错", BuildEntryPacket(1, EntryInfo{ServerID: 1})[:50]},
	}
	for _, c := range cases {
		if _, ok := ParseRegistration(c.pkt, 1); ok {
			t.Fatalf("%s：门未拒绝", c.name)
		}
	}
	// entry 包按 world 门检长度也应拒绝（72≠120）
	if _, ok := ParseRegistration(BuildEntryPacket(1, EntryInfo{ServerID: 1}), 1); !ok {
		t.Fatal("合法 entry 包被误拒")
	}
}

func clone(b []byte) []byte { return append([]byte(nil), b...) }

// ---- 表生命周期：注册/心跳/除名 ----

func newGL() *GameLister { return NewGameLister(0, 1, 0, 1024) }

func mustWorld(t *testing.T, gl *GameLister, w WorldInfo) {
	t.Helper()
	if !gl.RecvWorldInfo(w) {
		t.Fatalf("world 注册被拒：%+v", w)
	}
}

func mustEntry(t *testing.T, gl *GameLister, e EntryInfo) {
	t.Helper()
	if !gl.RecvEntryInfo(e) {
		t.Fatalf("entry 注册被拒：%+v", e)
	}
}

func TestRegistrationLifecycle(t *testing.T) {
	gl := newGL()
	// entry 先于 world：官方语义=丢弃（不建服）
	if gl.RecvEntryInfo(EntryInfo{ServerID: 5, LineIndex: 0, Max: 10}) {
		t.Fatal("entry 不得先于 world 建线")
	}
	mustWorld(t, gl, WorldInfo{ServerID: 5, IP: "1.1.1.1", Name: "五服", Max: 100, Cur: 20, Visible: true})
	if got := gl.FindServer(5); got == nil {
		t.Fatal("world 建服失败")
	}
	// line_index 越界拒绝
	if gl.RecvEntryInfo(EntryInfo{ServerID: 5, LineIndex: MaxLines, Max: 1}) {
		t.Fatal("line_index≥16 须拒绝")
	}
	mustEntry(t, gl, EntryInfo{ServerID: 5, LineIndex: 0, LineID: 5, IP: "1.1.1.2", Port: 7000, Max: 100, Cur: 20})
	mustEntry(t, gl, EntryInfo{ServerID: 5, LineIndex: 1, LineID: 6, IP: "1.1.1.3", Port: 7001, Max: 100, Cur: 80})
	if n := len(gl.Snapshot()); n != 1 {
		t.Fatalf("服数 %d ≠ 1", n)
	}

	// 心跳闩锁语义（官方 CheckServer）：注册置位 → 存活一个扫描周期
	// （sweep 保活并清零）→ 下一个周期无刷新才除名。
	gl.TickSeconds(5) // sweep#1：注册闩锁仍在 → 保活并清零
	if n := len(gl.Snapshot()); n != 1 {
		t.Fatalf("注册后首个扫描周期须保活，剩 %d", n)
	}

	// 线级单独死亡：只刷 world，entry 断更 → 服留线亡
	for i := 0; i < 5; i++ {
		mustWorld(t, gl, WorldInfo{ServerID: 5, Name: "五服", Visible: true, Max: 100, Cur: 20})
		gl.TickSeconds(1) // 第 5 拍触发 sweep#2
	}
	snap := gl.Snapshot()
	if len(snap) != 1 {
		t.Fatal("world 在刷新，服不得除名")
	}
	if snap[0].Lines[0].Valid || snap[0].Lines[1].Valid {
		t.Fatal("entry 断更须摘线")
	}

	// 服级死亡：全断更一个周期 → 整服除名（闩锁已被 sweep#2 清零）
	gl.TickSeconds(5) // sweep#3
	if n := len(gl.Snapshot()); n != 0 {
		t.Fatalf("world 断更须整服除名，剩 %d", n)
	}
}

// 表满：256 服上限（官方 server_t[0x100]）。
func TestServerTableCap(t *testing.T) {
	gl := newGL()
	for i := int32(1); i <= MaxServers; i++ {
		if !gl.RecvWorldInfo(WorldInfo{ServerID: i, Name: "s", Visible: true}) {
			t.Fatalf("第 %d 服注册被拒（未到上限）", i)
		}
	}
	if gl.RecvWorldInfo(WorldInfo{ServerID: 999, Name: "s", Visible: true}) {
		t.Fatal("超 256 服须拒绝")
	}
	// 死一个腾槽后可复用
	gl.mu.Lock()
	gl.servers[0].worldLatch = false
	gl.mu.Unlock()
	gl.Sweep()
	if !gl.RecvWorldInfo(WorldInfo{ServerID: 999, Name: "s", Visible: true}) {
		t.Fatal("空槽应可复用")
	}
}

// ---- 选线与 busy ----

func TestSelectionAndBusy(t *testing.T) {
	gl := newGL()
	mustWorld(t, gl, WorldInfo{ServerID: 1, Name: "甲", Visible: true, Max: 500, Cur: 100})
	mustEntry(t, gl, EntryInfo{ServerID: 1, LineIndex: 0, LineID: 1, IP: "2.2.2.2", Port: 7100, Max: 100, Cur: 50})  // 50%负载
	mustEntry(t, gl, EntryInfo{ServerID: 1, LineIndex: 1, LineID: 2, IP: "2.2.2.3", Port: 7101, Max: 100, Cur: 10}) // 10% ← 最小
	mustEntry(t, gl, EntryInfo{ServerID: 1, LineIndex: 2, LineID: 3, IP: "2.2.2.4", Port: 7102, Max: 100, Cur: 90}) // 90%

	es := gl.Entries()
	if len(es) != 1 {
		t.Fatalf("条目数 %d ≠ 1", len(es))
	}
	if es[0].Host != "2.2.2.3" || es[0].Port != 7101 {
		t.Fatalf("未选最小负载线：%+v", es[0])
	}
	if es[0].Busy != 1 { // 服级 cur*5/max = 100*5/500
		t.Fatalf("busy=%d ≠ 1", es[0].Busy)
	}

	// 隐身服不出应答；无线可用服跳过
	gl2 := newGL()
	mustWorld(t, gl2, WorldInfo{ServerID: 1, Name: "隐", Visible: false, Max: 10, Cur: 0})
	mustEntry(t, gl2, EntryInfo{ServerID: 1, LineIndex: 0, LineID: 1, IP: "3.3.3.3", Port: 1, Max: 10})
	if n := len(gl2.Entries()); n != 0 {
		t.Fatalf("不可见服条目 %d ≠ 0", n)
	}
	gl2.RecvWorldInfo(WorldInfo{ServerID: 2, Name: "无线", Visible: true, Max: 10})
	if n := len(gl2.Entries()); n != 0 {
		t.Fatal("无线服不得出现在应答")
	}
	// max=0 线不可选
	gl2.RecvEntryInfo(EntryInfo{ServerID: 2, LineIndex: 0, LineID: 1, IP: "3.3.3.4", Port: 2, Max: 0})
	if n := len(gl2.Entries()); n != 0 {
		t.Fatal("max=0 线不得入选")
	}
	// 空表应答形态：仅 $svrlist（官方 VarList 首元素）
	if got := string(newGL().EncodeResponse()); got != "$svrlist\r\n" {
		t.Fatalf("空表应答 %q ≠ \"$svrlist\\r\\n\"", got)
	}
}

// IDC 线过滤（官方 SendServerList 用 IDC 而非 ListerId——反汇编实证）。
func TestIDCLineFilter(t *testing.T) {
	gl := NewGameLister(9, 1, 0, 100) // IDC=9
	mustWorld(t, gl, WorldInfo{ServerID: 1, Name: "甲", Visible: true, Max: 100, Cur: 0})
	mustEntry(t, gl, EntryInfo{ServerID: 1, LineIndex: 0, LineID: 8, IP: "4.4.4.8", Port: 7200, Max: 10})
	mustEntry(t, gl, EntryInfo{ServerID: 1, LineIndex: 1, LineID: 9, IP: "4.4.4.9", Port: 7201, Max: 10})
	es := gl.Entries()
	if len(es) != 1 || es[0].Host != "4.4.4.9" {
		t.Fatalf("IDC 过滤失效：%+v", es)
	}
}

// 应答形态与 solo golden 同源（login.EncodeServerList 复用）。
func TestResponseDialect(t *testing.T) {
	gl := newGL()
	mustWorld(t, gl, WorldInfo{ServerID: 1, Name: "本地测试", Visible: true, Max: 1000, Cur: 0})
	mustEntry(t, gl, EntryInfo{ServerID: 1, LineIndex: 0, LineID: 1, IP: "127.0.0.1", Port: 19061, Max: 1000})
	want := "$svrlist #\\u672C\\u5730\\u6D4B\\u8BD5 $127.0.0.1 19061 0\r\n"
	if got := string(gl.EncodeResponse()); got != want {
		t.Fatalf("应答方言失真：\n got %q\nwant %q", got, want)
	}
}

// ---- 客户端面：真 TCP 行协议 ----

func startClientServer(t *testing.T, gl *GameLister, wait, linger time.Duration) (addr string, cs *ClientServer) {
	t.Helper()
	cs = &ClientServer{GL: gl, Addr: "127.0.0.1:0", RequestWait: wait, Linger: linger}
	a, err := cs.Listen()
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = cs.Serve() }()
	t.Cleanup(func() { _ = cs.Close() })
	return a.String(), cs
}

func dialRead(t *testing.T, addr string, send string, readWait time.Duration) (string, error) {
	t.Helper()
	conn, err := gonet.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if send != "" {
		_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
		if _, err := conn.Write([]byte(send)); err != nil {
			return "", err
		}
	}
	_ = conn.SetReadDeadline(time.Now().Add(readWait))
	var sb strings.Builder
	buf := make([]byte, 512)
	for {
		n, err := conn.Read(buf)
		sb.Write(buf[:n])
		if err != nil || n == 0 {
			return sb.String(), err
		}
	}
}

func TestClientProtocol(t *testing.T) {
	gl := newGL()
	mustWorld(t, gl, WorldInfo{ServerID: 1, Name: "甲", Visible: true, Max: 400, Cur: 80})
	mustEntry(t, gl, EntryInfo{ServerID: 1, LineIndex: 0, LineID: 1, IP: "127.0.0.1", Port: 9600, Max: 400, Cur: 80})
	addr, _ := startClientServer(t, gl, 0, 0)

	// CRLF 请求（\r 是控制符须被丢弃）→ 应答行尾 CRLF
	got, err := dialRead(t, addr, "svrlist\r\n", time.Second)
	if err != nil && !isTimeout(err) {
		t.Fatal(err)
	}
	if want := "$svrlist #\\u7532 $127.0.0.1 9600 1\r\n"; got != want {
		t.Fatalf("应答 %q ≠ %q", got, want)
	}

	// 非请求行一概不理
	got, err = dialRead(t, addr, "hello\nserverlist\nsvrlists\n", 300*time.Millisecond)
	if err == nil || !isTimeout(err) {
		t.Fatalf("垃圾行不得有应答（got %q err %v）", got, err)
	}
	// 控制字符（\x01/\t/\x00）丢弃后仍精确匹配 "svrlist"
	got, err = dialRead(t, addr, "sv\x01r\x02list\t\n", 500*time.Millisecond)
	if err != nil && !isTimeout(err) {
		t.Fatal(err)
	}
	if want := "$svrlist #\\u7532 $127.0.0.1 9600 1\r\n"; got != want {
		t.Fatalf("控制字符净化后应答 %q ≠ %q", got, want)
	}

	// 二次请求不重复应答（每连接一次）
	conn, err := gonet.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := conn.Write([]byte("svrlist\nsvrlist\n")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 512)
	n1, _ := conn.Read(buf)
	first := string(buf[:n1])
	_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if n2, err := conn.Read(buf); err == nil || n2 != 0 {
		t.Fatalf("第二次请求不得再应答（读回 %d 字节）", n2)
	}
	if !strings.HasSuffix(first, "\r\n") || !strings.HasPrefix(first, "$svrlist") {
		t.Fatalf("首应答形态异常 %q", first)
	}
}

// 60s 无请求超时与应答后 5s 温和关闭（缩短注入验证迁移本身）。
func TestClientTimeouts(t *testing.T) {
	gl := newGL()
	addr, _ := startClientServer(t, gl, 200*time.Millisecond, 200*time.Millisecond)

	// 无请求：超时关闭且无应答（服务端 deadline 关闭在客户端表现为
	// EOF 或读超时——两者都是"被关"）
	start := time.Now()
	got, err := dialRead(t, addr, "", time.Second)
	if err == nil || !(isTimeout(err) || err.Error() == "EOF") {
		t.Fatalf("超时关闭路径异常 err=%v", err)
	}
	if got != "" {
		t.Fatalf("超时不应答，却收到 %q", got)
	}
	if d := time.Since(start); d < 150*time.Millisecond {
		t.Fatalf("关闭过早 %v（须等待请求期限）", d)
	}

	// 应答后滞留：200ms 后服务端主动关闭
	conn, err := gonet.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
	if _, err := conn.Write([]byte("svrlist\n")); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 512)
	if _, err := conn.Read(buf); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	start = time.Now()
	if _, err := conn.Read(buf); err == nil || !(isTimeout(err) || err.Error() == "EOF") {
		t.Fatalf("滞留关闭路径异常 err=%v", err)
	}
	if d := time.Since(start); d >= time.Second {
		t.Fatal("滞留期限未生效（等满读期限才关）")
	}
}

// ---- 注册面：真 UDP ----

func TestUDPRegistration(t *testing.T) {
	gl := newGL()
	rs := &RegistrationServer{GL: gl, Port: 0}
	a, err := rs.Listen()
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = rs.Serve() }()
	t.Cleanup(func() { _ = rs.Close() })

	// 绑定地址是 0.0.0.0:port——定向回环须改写为 127.0.0.1
	//（Windows 对 0.0.0.0 目标不保证回环）
	ua := a.(*gonet.UDPAddr)
	dst := &gonet.UDPAddr{IP: gonet.IPv4(127, 0, 0, 1), Port: ua.Port}
	pc, err := gonet.DialUDP("udp4", nil, dst)
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()

	send := func(pkt []byte) {
		if _, err := pc.Write(pkt); err != nil {
			t.Fatal(err)
		}
	}
	send(BuildWorldPacket(1, WorldInfo{ServerID: 42, IP: "5.5.5.5", Name: "UDP服", Visible: true, Max: 10, Cur: 2}))
	send(BuildEntryPacket(1, EntryInfo{ServerID: 42, LineIndex: 0, LineID: 1, IP: "5.5.5.6", Port: 7300, Max: 10, Cur: 2}))
	// 坏包（分区错）必须被静默丢弃不影响后续
	send(BuildWorldPacket(9, WorldInfo{ServerID: 43, Visible: true}))

	deadline := time.Now().Add(time.Second)
	want := "$svrlist #UDP\\u670D $5.5.5.6 7300 1\r\n"
	got, snap := "", 0
	for time.Now().Before(deadline) {
		// got/snap 同拍取样：只盯快照非空再断言会有 TOCTOU——满载时
		// 包恰在 EncodeResponse 与 Fatalf 取快照之间落地，报出
		// 「应答异常（快照 1 服）」的假阳性。
		snap = len(gl.Snapshot())
		got = string(gl.EncodeResponse())
		if got == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("UDP 注册后应答 %q 异常（快照 %d 服）", got, snap)
}

// 主循环驱动：ServeGameLister 按官方 1s 节拍推进（5s 扫描编排已由
// TickSeconds 确定性覆盖，这里只验节拍真实走表）。
func TestServeGameListerTicks(t *testing.T) {
	gl := newGL()
	stop := ServeGameLister(gl)
	defer stop()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		gl.mu.Lock()
		ticked := gl.sweep > 0
		gl.mu.Unlock()
		if ticked {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("ServeGameLister 3s 内未推进任何秒拍")
}

func isTimeout(err error) bool {
	return err != nil && strings.Contains(err.Error(), "i/o timeout")
}
