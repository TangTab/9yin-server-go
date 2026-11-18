package login

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/net"
)

// startTestGateway 在随机端口起网关并进入 accept 循环，返回实际地址。
func startTestGateway(t *testing.T) (*Gateway, string) {
	t.Helper()
	g := &Gateway{Addr: "127.0.0.1:0"}
	addr, err := g.Listen()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { g.Close() })
	go g.Serve()
	return g, addr.String()
}

// dialProbe 以客户端装配（EE+XOR 初始键，零值默认）连上网关。
func dialProbe(t *testing.T, addr string) *net.Connection {
	t.Helper()
	conn, err := net.Dial(addr, net.Options{})
	if err != nil {
		t.Fatalf("拨号 %s: %v", addr, err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// TestGatewayHeartbeatGolden 真 TCP 回环：golden ping（fork 49840 抓包形态）
// 必须得到逐字节 golden pong——这是「握手+心跳往返」的验收线。
func TestGatewayHeartbeatGolden(t *testing.T) {
	_, addr := startTestGateway(t)
	conn := dialProbe(t, addr)

	cases := [][2]string{
		{"183154470000140000000000", "383154470000140000000000"},
		{"18f15b470000150000000000", "38f15b470000150000000000"},
		{"183154470000140000000000", "383154470000140000000000"},
	}
	for i, c := range cases {
		ping, _ := hex.DecodeString(c[0])
		want, _ := hex.DecodeString(c[1])
		if err := conn.WriteMsg(ping); err != nil {
			t.Fatalf("第 %d 轮发送: %v", i, err)
		}
		got, err := conn.ReadMsg()
		if err != nil {
			t.Fatalf("第 %d 轮接收: %v", i, err)
		}
		if string(got) != string(want) {
			t.Fatalf("第 %d 轮应答 %x，want %x", i, got, want)
		}
	}
}

// TestGatewayUnknownFrameTolerant 未知帧不致断会话：先发自定义帧再发心跳，
// 心跳仍按序应答（同连接帧序保持）。
func TestGatewayUnknownFrameTolerant(t *testing.T) {
	_, addr := startTestGateway(t)
	conn := dialProbe(t, addr)

	unknown := []byte{0x1E, 0x01, 0x00, 0x02, 0x05, 0x00, 0x00, 0x00}
	if err := conn.WriteMsg(unknown); err != nil {
		t.Fatal(err)
	}
	ping, _ := hex.DecodeString("183154470000140000000000")
	if err := conn.WriteMsg(ping); err != nil {
		t.Fatal(err)
	}
	got, err := conn.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 0x38 {
		t.Fatalf("未知帧后心跳失效: %x", got)
	}
}

// TestGatewayOnFrameHook 心跳之外的帧进 OnFrame 钩子（login 主干接入点）。
func TestGatewayOnFrameHook(t *testing.T) {
	g, addr := startTestGateway(t)

	got := make(chan []byte, 1)
	g.OnFrame = func(_ *Session, frame []byte) { got <- frame }

	conn := dialProbe(t, addr)
	custom := []byte{0x1E, 0x01, 0x00}
	if err := conn.WriteMsg(custom); err != nil {
		t.Fatal(err)
	}
	select {
	case frame := <-got:
		if string(frame) != string(custom) {
			t.Fatalf("钩子帧 %x want %x", frame, custom)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("OnFrame 未被调用")
	}
}

// TestGatewayClose Close 后：客户端读到错误、Serve 返回、二次 Close 幂等。
func TestGatewayClose(t *testing.T) {
	g, addr := startTestGateway(t)
	conn := dialProbe(t, addr)

	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ReadMsg(); err == nil {
		t.Fatal("Close 后客户端应读到错误")
	}
	if err := g.Close(); err != nil { // 幂等
		t.Fatal(err)
	}
}
