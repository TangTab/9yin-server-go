package snaileditornet

import (
	"bytes"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFrameCodec(t *testing.T) {
	payload := []byte("lua\\main.lua*ini\\scene.xml")
	raw := EncodeFrame(FMsgServerSvnUpdate, payload)
	if len(raw) != 8+len(payload) {
		t.Fatalf("帧长 %d", len(raw))
	}
	// 头部：totalLen(LE) + msgId(LE)
	if le32(raw[0:4]) != uint32(8+len(payload)) || le32(raw[4:8]) != 0x3F1 {
		t.Fatalf("头不符 %v", raw[:8])
	}
	// 回读
	r := &bytes.Buffer{}
	r.Write(raw)
	f, err := ReadFrame(r)
	if err != nil || f.MsgID != FMsgServerSvnUpdate || string(f.Payload) != string(payload) {
		t.Fatalf("回读不符: %v %v", f, err)
	}
	// 空载荷
	r.Reset()
	r.Write(EncodeFrame(0x3E8, nil))
	f, err = ReadFrame(r)
	if err != nil || len(f.Payload) != 0 {
		t.Fatalf("空帧: %v %v", f, err)
	}
}

func le32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func TestSvnHelpers(t *testing.T) {
	if got := NormalizeSvnPath("a/b/c.xml"); got != `a\b\c.xml` {
		t.Fatal(got)
	}
	got := SplitPaths("lua/main.lua\r\nini\\scene.xml alpha.xml")
	if len(got) != 3 || got[0] != `lua\main.lua` || got[1] != `ini\scene.xml` || got[2] != `alpha.xml` {
		t.Fatalf("%q", got)
	}
	args := BuildSvnUpdateArgs(`D:\ws`)
	want := []string{"/command:update", `/path:"D:\ws"`, "/notempfile", "/closeonend:1"}
	for i := range args {
		if args[i] != want[i] {
			t.Fatalf("args %v", args)
		}
	}
}

func TestClientServerLoopback(t *testing.T) {
	srv, err := NewServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	var mu sync.Mutex
	var commands, conflicts []string

	cli := NewClient(Config{
		SvnServerIP: "127.0.0.1",
		WatchPaths:  []string{`lua\`, `ini\`},
	}, Callbacks{
		OnDelaySvnUpdate: func(p string) { mu.Lock(); commands = append(commands, "delay:"+p); mu.Unlock() },
		OnEditorCommand:  func(d string) { mu.Lock(); commands = append(commands, "cmd:"+d); mu.Unlock() },
		OnConflict:       func(f string) { mu.Lock(); conflicts = append(conflicts, f); mu.Unlock() },
		OnSvnUpdateComplete: func(ps []string) {
			mu.Lock()
			for _, p := range ps {
				commands = append(commands, "done:"+p)
			}
			mu.Unlock()
		},
	})
	if err := cli.Connect(srv.Addr().String()); err != nil {
		t.Fatal(err)
	}
	cli.StartMsgLoop()
	defer cli.Close()

	// GUI → 游戏：SVN 更新路径（in-watch / out-of-watch）+ 命令帧 + 占用
	if err := WriteFrameTo(t, srv, FMsgServerSvnUpdate, []byte("lua/quest/a.xml ini\\cfg.xml res\\big.pack")); err != nil {
		t.Fatal(err)
	}
	if err := WriteFrameTo(t, srv, FMsgEditorCommand, []byte("CustomEditorWriteObject id=7")); err != nil {
		t.Fatal(err)
	}
	if err := WriteFrameTo(t, srv, FMsgFileInUse, []byte("scene.xml")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(commands) >= 3 && len(conflicts) == 1
	})
	mu.Lock()
	defer mu.Unlock()
	joined := strings.Join(commands, "|")
	if !strings.Contains(joined, "delay:lua\\quest\\a.xml") || !strings.Contains(joined, "delay:ini\\cfg.xml") {
		t.Fatalf("in-watch 路径未回调: %v", commands)
	}
	if strings.Contains(joined, "res\\big.pack") {
		t.Fatalf("out-of-watch 路径不应回调: %v", commands)
	}
	if !strings.Contains(joined, "cmd:CustomEditorWriteObject id=7") {
		t.Fatalf("命令帧未回调: %v", commands)
	}
	if conflicts[0] != "scene.xml" {
		t.Fatalf("占用回调: %v", conflicts)
	}

	// 游戏 → GUI：0x3F1 广播（经 OnFrame 观察面收，避免与 serveConn 抢流）
	ch := make(chan *Frame, 4)
	srv.OnFrame(func(id uint32, payload []byte) { ch <- &Frame{MsgID: id, Payload: payload} })
	if err := cli.SendSvnUpdate([]string{"lua/x.lua"}); err != nil {
		t.Fatal(err)
	}
	select {
	case f := <-ch:
		if f.MsgID != FMsgServerSvnUpdate || string(f.Payload) != `lua\x.lua` {
			t.Fatalf("广播帧不符: %#x %q", f.MsgID, f.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("广播超时")
	}
}

// WriteFrameTo 等服务器侧 accept 完成后广播一帧。
func WriteFrameTo(t *testing.T, srv *Server, msgID uint32, payload []byte) error {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		srv.mu.Lock()
		n := len(srv.conns)
		srv.mu.Unlock()
		if n > 0 {
			srv.SendAll(msgID, payload)
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return net.ErrClosed
}

func TestFileTransferLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	type result struct {
		name string
		data []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			ch <- result{err: err}
			return
		}
		defer conn.Close()
		err = GetFile(conn, func(name string, r io.Reader) error {
			b, err := io.ReadAll(r)
			ch <- result{name: name, data: b, err: err}
			return err
		})
		if err != nil {
			ch <- result{err: err}
		}
	}()
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	content := bytes.Repeat([]byte("9yin-editor-file."), 5000) // 85KB > ChunkLarge
	if err := SendFile(conn, "res\\test.bin", bytes.NewReader(content)); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatal(r.err)
		}
		if r.name != "res\\test.bin" || !bytes.Equal(r.data, content) {
			t.Fatalf("文件不符 %q len=%d", r.name, len(r.data))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("传输超时")
	}
}

func waitFor(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
