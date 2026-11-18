package login

import (
	"bytes"
	gonet "net"
	"os"
	"testing"
	"time"
)

// golden 转录（旧分支 testdata/lister 拷贝；入库时 CR 被 git 文本规范化
// 洗去，故比对按「去结尾换行逐字节一致」，结尾断言以实证的 CRLF 为准）。
const goldenResponse = "$svrlist #\\u672C\\u5730\\u6D4B\\u8BD5 $127.0.0.1 19061 0"

func TestEncodeServerListGolden(t *testing.T) {
	got := EncodeServerList(ServerEntry{Name: "本地测试", Host: "127.0.0.1", Port: 19061})
	if want := goldenResponse + "\r\n"; string(got) != want {
		t.Fatalf("编码 %q want %q", got, want)
	}

	// 与抓包 golden 交叉验证（本体一致，结尾换行差异为入库规范化伪影）
	raw, err := os.ReadFile("../../testdata/captures/svrlist-response.bin")
	if err != nil {
		t.Fatal(err)
	}
	body := got
	body = body[:len(body)-2] // 去掉 CRLF
	body = append(body, '\n') // 对齐 golden 的 LF 结尾
	if !bytes.Equal(body, raw) {
		t.Fatalf("与 golden 不一致:\n got  %q\n want %q", body, raw)
	}
}

func TestEscapeUnicode(t *testing.T) {
	cases := []struct{ in, want string }{
		{"本地测试", "\\u672C\\u5730\\u6D4B\\u8BD5"}, // golden 样本
		{"abc-1", "abc-1"},                           // ASCII 原样（golden 无样本，通用惯例假设）
		{"𐐷", "\\uD801\\uDC37"},                     // 增补面拆代理对
		{"", ""},
	}
	for _, c := range cases {
		if got := escapeUnicode(c.in); got != c.want {
			t.Fatalf("escapeUnicode(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func startTestLister(t *testing.T, entry ServerEntry) (*Lister, string) {
	t.Helper()
	l := &Lister{Addr: "127.0.0.1:0", Entry: entry}
	addr, err := l.Listen()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go l.Serve()
	return l, addr.String()
}

// TestListerRoundTrip 真 TCP：发 golden 请求字节，收到的应答必须与
// 编码纯函数输出逐字节一致（svrlist 是明文 ASCII 线路，不走 EE+XOR）。
func TestListerRoundTrip(t *testing.T) {
	entry := ServerEntry{Name: "本地测试", Host: "127.0.0.1", Port: 19061}
	_, addr := startTestLister(t, entry)

	req, err := os.ReadFile("../../testdata/captures/svrlist-request.bin")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := gonet.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write(req); err != nil {
		t.Fatal(err)
	}

	var got bytes.Buffer
	buf := make([]byte, 128)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		n, err := conn.Read(buf)
		got.Write(buf[:n])
		if err != nil { // 服务端应答后断连，EOF 即完整
			break
		}
	}
	if want := EncodeServerList(entry); !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("应答 %q want %q", got.String(), want)
	}
}

// TestListerCRRequest 也接受 \r 结尾的请求（PS1 同款宽容定界）。
func TestListerCRRequest(t *testing.T) {
	entry := ServerEntry{Name: "测试", Host: "127.0.0.1", Port: 9600}
	_, addr := startTestLister(t, entry)

	conn, err := gonet.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("svrlist\r")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 128)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ := conn.Read(buf)
	if want := EncodeServerList(entry); !bytes.Equal(buf[:n], want) {
		t.Fatalf("应答 %q want %q", buf[:n], want)
	}
}

func TestListerClose(t *testing.T) {
	entry := ServerEntry{Name: "测试", Host: "127.0.0.1", Port: 9600}
	l, addr := startTestLister(t, entry)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil { // 幂等
		t.Fatal(err)
	}
	if _, err := gonet.DialTimeout("tcp", addr, 500*time.Millisecond); err == nil {
		t.Fatal("Close 后不应再可拨号")
	}
}

func TestDefaultListerEntry(t *testing.T) {
	cases := []struct {
		gameAddr string
		host     string
		port     int
	}{
		{"127.0.0.1:9600", "127.0.0.1", 9600},
		{":9600", "127.0.0.1", 9600},
		{"0.0.0.0:9600", "127.0.0.1", 9600},
		{"192.168.1.5:12345", "192.168.1.5", 12345},
	}
	for _, c := range cases {
		e := DefaultListerEntry(c.gameAddr)
		if e.Host != c.host || e.Port != c.port {
			t.Fatalf("DefaultListerEntry(%s)=%+v want host=%s port=%d",
				c.gameAddr, e, c.host, c.port)
		}
	}
}
