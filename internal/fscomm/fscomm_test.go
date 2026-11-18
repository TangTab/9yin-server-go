package fscomm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// waitCond 轮询等待条件成立（网络测试异步回调同步面）。
func waitCond(t *testing.T, timeout time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

// ---- msgFIFO ----

func TestMsgFIFOBounded(t *testing.T) {
	f := newMsgFIFO(2, 10)
	if err := f.Put([]byte("aaa")); err != nil {
		t.Fatal(err)
	}
	if err := f.Put([]byte("bb")); err != nil {
		t.Fatal(err)
	}
	if err := f.Put([]byte("c")); err != ErrFIFOFull { // 条数满
		t.Fatalf("want ErrFIFOFull, got %v", err)
	}
	f.Pop() // 剩 "bb"（2 字节）
	if err := f.Put([]byte("ccccccccc")); err != ErrFIFOFull { // 字节满（2+9>10）
		t.Fatalf("want ErrFIFOFull, got %v", err)
	}
	if got := string(f.Pop()); got != "bb" {
		t.Fatalf("fifo order broken: %q", got)
	}
	if f.Len() != 0 || f.Bytes() != 0 {
		t.Fatal("fifo not empty")
	}
	if f.Pop() != nil {
		t.Fatal("pop on empty must be nil")
	}
}

// ---- Adapter 帧模式 ↔ 手写客户端（官方线格式对拍）----

func TestAdapterFramedEcho(t *testing.T) {
	var mu sync.Mutex
	var ad *Adapter
	ad = NewAdapter(AdapterOptions{Address: "127.0.0.1", Port: 0}, AdapterHandler{
		OnAccept: func(idx int, addr string) {
			mu.Lock()
			defer mu.Unlock()
			if err := ad.Send(idx, []byte("welcome")); err != nil {
				t.Errorf("send: %v", err)
			}
		},
		OnRead: func(idx int, data []byte) {
			if err := ad.Send(idx, append([]byte("ack:"), data...)); err != nil {
				t.Errorf("send: %v", err)
			}
		},
	})
	if err := ad.Start(); err != nil {
		t.Fatal(err)
	}
	defer ad.Stop()

	c, err := net.Dial("tcp", ad.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// 官方线格式手写：[u32 LE len][body]
	writeOne := func(body string) {
		var head [4]byte
		binary.LittleEndian.PutUint32(head[:], uint32(len(body)))
		c.Write(head[:])
		c.Write([]byte(body))
	}
	readOne := func() []byte {
		var head [4]byte
		if _, err := io.ReadFull(c, head[:]); err != nil {
			t.Fatal(err)
		}
		n := binary.LittleEndian.Uint32(head[:])
		body := make([]byte, n)
		if _, err := io.ReadFull(c, body); err != nil {
			t.Fatal(err)
		}
		return body
	}

	if got := string(readOne()); got != "welcome" {
		t.Fatalf("welcome want %q got %q", "welcome", got)
	}
	writeOne("hello")
	if got := string(readOne()); got != "ack:hello" {
		t.Fatalf("echo want %q got %q", "ack:hello", got)
	}
	// 半帧粘包：一次写 1.5 帧，帧器应正确切分
	var head [4]byte
	binary.LittleEndian.PutUint32(head[:], 4)
	c.Write(head[:])
	c.Write([]byte("abcd"))
	binary.LittleEndian.PutUint32(head[:], 2)
	c.Write(head[:])
	c.Write([]byte("ef"))
	if got := string(readOne()); got != "ack:abcd" {
		t.Fatalf("split frame want ack:abcd got %q", got)
	}
}

func TestAdapterBroadcastAndClose(t *testing.T) {
	var mu sync.Mutex
	var accepted []int
	ad := NewAdapter(AdapterOptions{Address: "127.0.0.1", Port: 0, ConnectMax: 4}, AdapterHandler{
		OnAccept: func(idx int, addr string) {
			mu.Lock()
			accepted = append(accepted, idx)
			mu.Unlock()
		},
	})
	if err := ad.Start(); err != nil {
		t.Fatal(err)
	}
	defer ad.Stop()

	var conns []net.Conn
	for i := 0; i < 3; i++ {
		c, err := net.Dial("tcp", ad.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		conns = append(conns, c)
	}
	if !waitCond(t, 3*time.Second, func() bool { return ad.ConnectCount() == 3 }) {
		t.Fatalf("accept count %d", ad.ConnectCount())
	}
	if err := ad.Broadcast([]byte("bc")); err != nil {
		t.Fatal(err)
	}
	for i, c := range conns {
		var head [4]byte
		if _, err := io.ReadFull(c, head[:]); err != nil {
			t.Fatal(i, err)
		}
		if n := binary.LittleEndian.Uint32(head[:]); n != 2 {
			t.Fatalf("conn %d frame len %d", i, n)
		}
		var body [2]byte
		io.ReadFull(c, body[:])
		if string(body[:]) != "bc" {
			t.Fatalf("conn %d body %q", i, body)
		}
	}
	// 关闭单连接
	ad.Close(0)
	if !waitCond(t, 3*time.Second, func() bool { return ad.ConnectCount() == 2 }) {
		t.Fatalf("after close count %d", ad.ConnectCount())
	}
	// 超限消息被拒
	if err := ad.Send(999, make([]byte, 16)); err == nil {
		t.Fatal("send to invalid idx must fail")
	}
}

// ---- Adapter Raw 模式（MassSock 语义）----

func TestAdapterRawMode(t *testing.T) {
	got := make(chan []byte, 4)
	ad := NewAdapter(AdapterOptions{Address: "127.0.0.1", Port: 0, Raw: true}, AdapterHandler{
		OnRead: func(idx int, data []byte) { got <- data },
	})
	if err := ad.Start(); err != nil {
		t.Fatal(err)
	}
	defer ad.Stop()

	c, err := net.Dial("tcp", ad.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Write([]byte("rawbytes-no-frame"))
	select {
	case d := <-got:
		if string(d) != "rawbytes-no-frame" {
			t.Fatalf("raw payload %q", d)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("raw read timeout")
	}
}

// ---- MultiSock 自动重连 + 帧互拍 ----

func TestMultiSockReconnectAndFraming(t *testing.T) {
	var events sync.Map
	var ad *Adapter
	ad = NewAdapter(AdapterOptions{Address: "127.0.0.1", Port: 0}, AdapterHandler{
		OnRead: func(idx int, data []byte) {
			_ = ad.Send(idx, bytes.ToUpper(data))
		},
	})
	if err := ad.Start(); err != nil {
		t.Fatal(err)
	}
	_, portStr, _ := net.SplitHostPort(ad.Addr().String())

	beginN, connN, failN, closeN := 0, 0, 0, 0
	var mu sync.Mutex
	ms := NewMultiSock(MultiSockOptions{
		Address:   "127.0.0.1",
		Port:      mustPort(t, portStr),
		AutoRetry: true,
		RetryDelay: 100 * time.Millisecond,
	}, MultiSockHandler{
		OnBegin:       func() { mu.Lock(); beginN++; mu.Unlock() },
		OnConnected:   func() { mu.Lock(); connN++; mu.Unlock() },
		OnConnectFail: func(err error) { mu.Lock(); failN++; mu.Unlock() },
		OnClose:       func(err error) { mu.Lock(); closeN++; mu.Unlock() },
		OnRead: func(data []byte) {
			events.Store("last", string(data))
		},
	})
	ms.Start()
	defer ms.Stop()

	if !waitCond(t, 3*time.Second, func() bool { return ms.Connected() }) {
		t.Fatal("multisock not connected")
	}
	if err := ms.Send([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	if !waitCond(t, 3*time.Second, func() bool {
		v, _ := events.Load("last")
		return v == "PING"
	}) {
		t.Fatal("framed roundtrip missing")
	}

	// 掐断服务端 → 客户端应自动重连（AutoRetry）
	ad.Stop()
	time.Sleep(250 * time.Millisecond) // 覆盖 RetryDelay
	if !waitCond(t, 3*time.Second, func() bool { return !ms.Connected() }) {
		t.Fatal("should be down after server stop")
	}
	ad2 := NewAdapter(AdapterOptions{Address: "127.0.0.1", Port: mustPort(t, portStr)}, AdapterHandler{})
	if err := ad2.Start(); err != nil {
		t.Fatal(err)
	}
	defer ad2.Stop()
	if !waitCond(t, 5*time.Second, func() bool { return ms.Connected() }) {
		mu.Lock()
		defer mu.Unlock()
		t.Fatalf("no reconnect: begin=%d conn=%d fail=%d close=%d", beginN, connN, failN, closeN)
	}
	mu.Lock()
	defer mu.Unlock()
	if connN < 2 || closeN < 1 {
		t.Fatalf("lifecycle counts: begin=%d conn=%d fail=%d close=%d", beginN, connN, failN, closeN)
	}
}

func mustPort(t *testing.T, s string) int {
	t.Helper()
	var p int
	fmt.Sscanf(s, "%d", &p)
	return p
}

// ---- UserSock 同步/异步 ----

func TestUserSockSyncSend(t *testing.T) {
	var got []byte
	var mu sync.Mutex
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		b := make([]byte, 64)
		n, _ := c.Read(b)
		mu.Lock()
		got = append([]byte(nil), b[:n]...)
		mu.Unlock()
	}()

	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	u := NewUserSock(UserSockOptions{Address: "127.0.0.1", Port: mustPort(t, portStr)}, UserSockHandler{})
	if err := u.Connect(); err != nil {
		t.Fatal(err)
	}
	defer u.Close()
	if !u.Connected() {
		t.Fatal("not connected")
	}
	if err := u.Send([]byte("plain")); err != nil {
		t.Fatal(err)
	}
	if !waitCond(t, 3*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return string(got) == "plain"
	}) {
		t.Fatalf("server got %q", got)
	}
	if err := u.Send([]byte("x")); err != nil {
		t.Fatal(err)
	}
}

// ---- EchoSock 行协议 + 分发 ----

func TestEchoSockLineProtocol(t *testing.T) {
	var mu sync.Mutex
	var lines []string
	var openPort int
	cmdArgs := make(chan []string, 4)
	es := NewEchoSock("127.0.0.1", 0, EchoSockHandler{
		OnOpen: func(addr string, port int) { mu.Lock(); openPort = port; mu.Unlock() },
		OnInput: func(line string) {
			mu.Lock()
			lines = append(lines, line)
			mu.Unlock()
		},
		OnCommand: map[string]EchoLineHandler{
			"quit": func(args []string) { cmdArgs <- args },
		},
	})
	if err := es.Start(); err != nil {
		t.Fatal(err)
	}
	defer es.Stop()

	c, err := net.Dial("tcp", es.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// 官方方言：\r 或 \n 皆终止；空行不分发；具名命令独占
	c.Write([]byte("hello world\r\n"))
	c.Write([]byte("\n"))
	c.Write([]byte("quit now\r"))
	c.Write([]byte("\ntail line\n"))
	select {
	case args := <-cmdArgs:
		if len(args) != 1 || args[0] != "now" {
			t.Fatalf("quit args %v", args)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("on_input_quit not dispatched")
	}
	if !waitCond(t, 3*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(lines) == 2
	}) {
		mu.Lock()
		ls := append([]string(nil), lines...)
		mu.Unlock()
		t.Fatalf("fallback lines %v", ls)
	}
	mu.Lock()
	first, second := lines[0], lines[1]
	mu.Unlock()
	if first != "hello world" || second != "tail line" {
		t.Fatalf("lines %v", lines)
	}
	if openPort == 0 {
		t.Fatal("on_open not fired")
	}
	// SendLine 不补终止符
	if err := es.SendLine(">ok\r"); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 16)
	c.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := c.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != ">ok\r" {
		t.Fatalf("sendline payload %q", buf[:n])
	}
	// 超长行截断到 1023（第 1024 字节起丢弃，终止符后分发截断行）
	long := strings.Repeat("a", 1100)
	c.Write([]byte(long + "\n"))
	if !waitCond(t, 3*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(lines) == 3 && len(lines[2]) == echoLineMax
	}) {
		mu.Lock()
		defer mu.Unlock()
		t.Fatalf("truncate: n=%d len2=%d", len(lines), len(lines[2]))
	}
}

// ---- BroadSock UDP ----

func TestBroadSockRoundtrip(t *testing.T) {
	got := make(chan []byte, 4)
	b := NewBroadSock(BroadSockOptions{
		BindAddress:     "127.0.0.1",
		BroadcastAddress: "127.0.0.1",
	}, BroadSockHandler{
		OnReceive: func(data []byte, addr string, port int) { got <- data },
	})
	if err := b.Start(); err != nil {
		t.Fatal(err)
	}
	defer b.Stop()

	bindPort := b.conn().LocalAddr().(*net.UDPAddr).Port
	if err := b.Send("127.0.0.1", bindPort, []byte("udp-ping")); err != nil {
		t.Fatal(err)
	}
	select {
	case d := <-got:
		if string(d) != "udp-ping" {
			t.Fatalf("udp payload %q", d)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("udp roundtrip timeout")
	}
}

// ---- AppManager ----

func TestAppManagerRegisterAndSweep(t *testing.T) {
	dir := t.TempDir()
	am, err := NewAppManager("testbook", dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := am.Register("world_1", "127.0.0.1:5000", 4); err != nil {
		t.Fatal(err)
	}
	defer am.Close()

	if am.GetAppCount() != 1 {
		t.Fatalf("count %d", am.GetAppCount())
	}
	if addr, ok := am.GetAppAddress("world_1"); !ok || addr != "127.0.0.1:5000" {
		t.Fatalf("addr %q %v", addr, ok)
	}
	_ = am.SetAppState(5)
	list := am.GetAppList()
	if len(list) != 1 || list[0].State != 5 {
		t.Fatalf("state not updated: %+v", list)
	}

	// 单实例：同名第二实例被拒（官方 mutex exclusive 语义）
	am2, err := NewAppManager("testbook", dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := am2.Register("world_1", "x", 0); err == nil {
		t.Fatal("duplicate app name must be rejected")
	}
	// 异名注册可见
	if err := am2.Register("entry_1", "127.0.0.1:2001", 4); err != nil {
		t.Fatal(err)
	}
	if am2.GetAppCount() != 2 {
		t.Fatalf("shared book count %d", am2.GetAppCount())
	}

	// 死进程清扫：伪造死亡 pid 槽（pid 4000000 近乎必不存在）
	dead := AppInfo{Name: "ghost", ProcID: 4000000, Addr: "g:1"}
	if err := mutateForTest(am, dead); err != nil {
		t.Fatal(err)
	}
	if am.GetAppCount() != 3 {
		t.Fatalf("pre-sweep count %d", am.GetAppCount())
	}
	am.Sweep()
	if am.GetAppCount() != 2 {
		t.Fatalf("post-sweep count %d", am.GetAppCount())
	}
	if _, ok := am.GetAppAddress("ghost"); ok {
		t.Fatal("ghost survived sweep")
	}
	// Close 摘除自身槽
	am2.Close()
	if am.GetAppCount() != 1 {
		t.Fatalf("after am2 close count %d", am.GetAppCount())
	}
}

// mutateForTest 直接塞一条槽（测试专用口）。
func mutateForTest(am *AppManager, info AppInfo) error {
	return am.mutate(func(apps *[]AppInfo) {
		*apps = append(*apps, info)
	})
}

// ---- LogPool ----

func TestLogPoolDailyNaming(t *testing.T) {
	dir := t.TempDir()
	p := NewLogPool(dir)
	p.FilePrefix = "world_"
	p.EveryDayAlone = true
	if err := p.WriteLine("game server starting"); err != nil {
		t.Fatal(err)
	}
	if n := p.LineCount(); n != 1 {
		t.Fatalf("lines %d", n)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("files %d", len(entries))
	}
	name := entries[0].Name()
	// 官方命名 "%s%04d%02d%02d%02d%02d%02d.log"：前缀+14位时间戳+.log
	if !strings.HasPrefix(name, "world_") || !strings.HasSuffix(name, ".log") || len(name) != len("world_")+14+len(".log") {
		t.Fatalf("daily name %q", name)
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "] game server starting\n") {
		t.Fatalf("line format %q", s)
	}
	if s[0] != '[' {
		t.Fatalf("missing timestamp prefix %q", s)
	}
	p.Stop()

	// 固定文件模式
	p2 := NewLogPool(dir)
	p2.FileName = "fixed.log"
	if err := p2.WriteLine("a"); err != nil {
		t.Fatal(err)
	}
	if err := p2.WriteLine("b"); err != nil {
		t.Fatal(err)
	}
	p2.Stop()
	data, _ = os.ReadFile(filepath.Join(dir, "fixed.log"))
	if !strings.Contains(string(data), "] a\n") || !strings.Contains(string(data), "] b\n") {
		t.Fatalf("fixed file %q", data)
	}
}

// ---- LogicLoader ----

func TestLogicLoader(t *testing.T) {
	l := NewLogicLoader()
	if err := l.Register("fsgame", LogicGame, func() any { return "game-logic" }); err != nil {
		t.Fatal(err)
	}
	if err := l.Register("bad", "FxOther", func() any { return nil }); err == nil {
		t.Fatal("wrong type must be rejected")
	}
	v, err := l.Load("fsgame")
	if err != nil || v != "game-logic" {
		t.Fatalf("load %v %v", v, err)
	}
	if _, err := l.Load("nope"); err == nil {
		t.Fatal("unknown module must fail")
	}
	if l.Count() != 1 || len(l.Modules()) != 1 {
		t.Fatalf("count/modules mismatch: %d %v", l.Count(), l.Modules())
	}
}
