package reactor

import (
	"fmt"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"
)

const testWait = 3 * time.Second

// coID 当前协程 id（事件循环恒为同一个）。
func coID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	var id uint64
	fmt.Sscanf(string(buf[:n]), "goroutine %d", &id)
	return id
}

// recvOrder 带超时的 order 读取（裸收会永久阻塞）。
func recvOrder(t *testing.T, ch chan int) int {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(testWait):
		t.Fatal("等回调超时")
		return -1
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// TestListenerConnectorEcho 三通道核心路径：listener 接入 + connector
// 外连 + Send 异步回程；回调全部串行（EventLoop 单线程语义）。
func TestListenerConnectorEcho(t *testing.T) {
	r := New()
	if err := r.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.Stop)

	port := freePort(t)
	srvEcho := make(chan []byte, 4)
	got := make(chan []byte, 4)
	opened := make(chan int, 4)

	_, err := r.CreateListener("127.0.0.1", port, 0, ConnHandler{
		OnInput: func(id int, data []byte) {
			srvEcho <- data
			r.Send(id, []byte("echo:"+string(data))) // 事件循环内回发
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	connID, err := r.CreateConnector("127.0.0.1", port, ConnHandler{
		OnOpen:  func(id int, addr string, p int) { opened <- id },
		OnInput: func(id int, data []byte) { got <- data },
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case id := <-opened:
		if id != connID {
			t.Fatalf("OnOpen id=%d want %d", id, connID)
		}
	case <-time.After(testWait):
		t.Fatal("等 OnOpen 超时")
	}
	if !r.GetConnected(connID) { // OnOpen 后必为 connected
		t.Fatal("OnOpen 后 GetConnected 应 true")
	}

	if !r.Send(connID, []byte("ping")) {
		t.Fatal("Send 失败")
	}
	select {
	case d := <-got:
		if string(d) != "echo:ping" {
			t.Fatalf("回程 %q want echo:ping", d)
		}
	case <-time.After(testWait):
		t.Fatal("等回程超时")
	}
	select {
	case d := <-srvEcho:
		if string(d) != "ping" {
			t.Fatalf("服务端收到 %q want ping", d)
		}
	default:
	}
}

// TestSerializedCallbacks 官方单线程 EventLoop 语义：所有回调在同一
// 协程串行执行，业务侧无需加锁。
func TestSerializedCallbacks(t *testing.T) {
	r := New()
	r.Start()
	t.Cleanup(r.Stop)

	port := freePort(t)
	var mu sync.Mutex
	goroutines := map[string]bool{}
	opened := make(chan int, 2)
	order := make(chan int, 16)

	_, err := r.CreateListener("127.0.0.1", port, 0, ConnHandler{
		OnOpen: func(id int, addr string, p int) {
			mu.Lock()
			goroutines[fmt.Sprint(coID())] = true
			mu.Unlock()
			order <- 1
		},
		OnInput: func(id int, data []byte) {
			mu.Lock()
			goroutines[fmt.Sprint(coID())] = true
			mu.Unlock()
			order <- 2
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	connID, _ := r.CreateConnector("127.0.0.1", port, ConnHandler{
		OnOpen: func(id int, addr string, p int) { opened <- id },
	})
	select {
	case <-opened:
	case <-time.After(testWait):
		t.Fatal("等 OnOpen 超时")
	}
	if !r.Send(connID, []byte("x")) {
		t.Fatal("Send 失败")
	}
	// 等两个回调都落地
	deadline := time.Now().Add(testWait)
	for len(order) < 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}

	mu.Lock()
	n := len(goroutines)
	mu.Unlock()
	if n != 1 {
		t.Fatalf("回调应恒在同一协程执行，实得 %d 个", n)
	}
	if v := recvOrder(t, order); v != 1 {
		t.Fatal("回调乱序")
	}
	if v := recvOrder(t, order); v != 2 {
		t.Fatal("回调乱序")
	}
}

// TestIdentityStaleID 陈旧 id 拒绝（官方 identity error 同效）：
// 关闭后的 id 再 Send/GetConnected 一律 false。
func TestIdentityStaleID(t *testing.T) {
	r := New()
	r.Start()
	t.Cleanup(r.Stop)

	port := freePort(t)
	closed := make(chan int, 4)
	_, err := r.CreateListener("127.0.0.1", port, 0, ConnHandler{
		OnClose: func(id int, addr string, p int) { closed <- id },
	})
	if err != nil {
		t.Fatal(err)
	}
	connClosed := make(chan int, 2)
	connID, _ := r.CreateConnector("127.0.0.1", port, ConnHandler{
		OnClose: func(id int, addr string, p int) { connClosed <- id },
	})
	time.Sleep(300 * time.Millisecond) // 等 OnOpen

	if !r.ShutdownConnector(connID) {
		t.Fatal("ShutdownConnector 应成功")
	}
	select {
	case id := <-connClosed:
		if id != connID {
			t.Fatalf("OnClose id=%d want %d", id, connID)
		}
	case <-time.After(testWait):
		t.Fatal("等 connector OnClose 超时")
	}

	if r.Send(connID, []byte("x")) {
		t.Fatal("陈旧 id Send 应拒绝")
	}
	if r.GetConnected(connID) {
		t.Fatal("陈旧 id GetConnected 应 false")
	}
	if r.ShutdownConnector(connID) {
		t.Fatal("陈旧 id ShutdownConnector 应拒绝")
	}
	if r.Send(0x7fffffff, []byte("x")) {
		t.Fatal("伪造 id Send 应拒绝")
	}
}

// TestBroadcastDirectedLoopback 广播口回环（定向到自身绑定端口）：
// 收发一体 + 回调带来源地址。真网段广播部署侧验证，单测用定向保稳定。
func TestBroadcastDirectedLoopback(t *testing.T) {
	r := New()
	r.Start()
	t.Cleanup(r.Stop)

	// 先占一个 UDP 端口
	probe, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := probe.LocalAddr().(*net.UDPAddr).Port
	probe.Close()

	got := make(chan string, 2)
	bid, err := r.CreateBroadcast("127.0.0.1", port, func(data []byte, from string, p int) {
		got <- fmt.Sprintf("%s|%d|%s", data, p, from)
	})
	if err != nil {
		t.Fatal(err)
	}
	if !r.SendBroadcast(bid, []byte("hello")) {
		t.Fatal("SendBroadcast 失败")
	}
	select {
	case g := <-got:
		if g != "hello|"+fmt.Sprint(port)+"|127.0.0.1" {
			t.Fatalf("广播自收 %q", g)
		}
	case <-time.After(testWait):
		t.Fatal("等广播自收超时")
	}
}

// TestStopSemantics Stop 幂等且排空在途回调（OnClose 必达）。
func TestStopSemantics(t *testing.T) {
	r := New()
	r.Start()

	port := freePort(t)
	onClose := make(chan int, 2)
	_, err := r.CreateListener("127.0.0.1", port, 0, ConnHandler{
		OnClose: func(id int, addr string, p int) { onClose <- id },
	})
	if err != nil {
		t.Fatal(err)
	}
	connID, _ := r.CreateConnector("127.0.0.1", port, ConnHandler{})
	time.Sleep(300 * time.Millisecond)

	r.Stop()
	r.Stop() // 幂等

	select {
	case <-onClose:
	case <-time.After(testWait):
		t.Fatal("Stop 后 OnClose 未达（回调未排空）")
	}
	if r.Send(connID, []byte("x")) {
		t.Fatal("Stop 后 Send 应拒绝")
	}
}

// TestBroadcastCoBind 多实体共绑同一广播口（SO_REUSEADDR 官方语义）：
// 官方集群多进程/合一进程内多实体同口互收广播——每个套接字都收到
// 双方报文（含发送者自收；Win11 实测 127.255.255.255 定向广播全投递）。
func TestBroadcastCoBind(t *testing.T) {
	probe, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := probe.LocalAddr().(*net.UDPAddr).Port
	probe.Close()

	r1, r2 := New(), New()
	r1.Start()
	r2.Start()
	t.Cleanup(r1.Stop)
	t.Cleanup(r2.Stop)

	got1, got2 := make(chan string, 4), make(chan string, 4)
	b1, err := r1.CreateBroadcast("127.255.255.255", port, func(data []byte, from string, p int) {
		got1 <- string(data)
	})
	if err != nil {
		t.Fatalf("第二前绑口: %v", err)
	}
	b2, err := r2.CreateBroadcast("127.255.255.255", port, func(data []byte, from string, p int) {
		got2 <- string(data)
	})
	if err != nil {
		t.Fatalf("共绑同口失败（REUSEADDR 未生效？）: %v", err)
	}

	r1.SendBroadcast(b1, []byte("m1"))
	r2.SendBroadcast(b2, []byte("m2"))

	collect := func(ch chan string) map[string]bool {
		got := map[string]bool{}
		for len(got) < 2 {
			select {
			case m := <-ch:
				got[m] = true
			case <-time.After(testWait):
				return got
			}
		}
		return got
	}
	for name, ch := range map[string]chan string{"r1": got1, "r2": got2} {
		got := collect(ch)
		if !got["m1"] || !got["m2"] {
			t.Fatalf("%s 未双收 m1/m2（得 %v）——共绑广播互收失效", name, got)
		}
	}
}
