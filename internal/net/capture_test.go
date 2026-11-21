package net

import (
	gonet "net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCaptureBaseName(t *testing.T) {
	cases := []struct {
		host, port, want string
	}{
		{"127.0.0.1", "51512", "127-0-0-1-51512"}, // 对拍基准同构：点换杠+杠拼端口
		{"192.168.1.23", "4000", "192-168-1-23-4000"},
	}
	for _, c := range cases {
		addr, err := gonet.ResolveTCPAddr("tcp", c.host+":"+c.port)
		if err != nil {
			t.Fatalf("resolve %s: %v", c.host, err)
		}
		if got := captureBaseName(addr); got != c.want {
			t.Errorf("captureBaseName(%v) = %q, want %q", addr, got, c.want)
		}
	}
	if got := captureBaseName(nil); got == "" {
		t.Error("captureBaseName(nil) 应有兜底名")
	}
}

func TestCaptureDisabledWhenEnvUnset(t *testing.T) {
	t.Setenv(captureEnv, "")
	p := newLoopPipe()
	defer p.Close()
	if got := captureStream(p); got != p {
		t.Fatalf("未设 %s 时应原样返回流（got %T）", captureEnv, got)
	}
}

// TestCaptureRecordsBothDirections 走完整装配路径：NewConnection 在
// NINEYIN_CAPTURE_DIR 生效时包住服务端流，c2s/s2c 两方向 wire 字节
// （成帧后、按 PrefixFramer 布局）按序落盘，文件名取自客户端侧地址。
func TestCaptureRecordsBothDirections(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(captureEnv, dir)

	ln, err := gonet.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	accepted := make(chan gonet.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		accepted <- c
	}()
	cli, err := gonet.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer cli.Close()
	srv := <-accepted

	conn, err := NewConnection(srv, Options{
		Framer:      PrefixFramerFactory,
		ReadCipher:  IdentityCipher{},
		WriteCipher: IdentityCipher{},
	})
	if err != nil {
		t.Fatalf("NewConnection: %v", err)
	}

	// s2c：服务端连发两帧（[4B LE len][body]，Identity 直通）
	if err := conn.WriteMsg([]byte{0x01, 0x02}); err != nil {
		t.Fatalf("WriteMsg#1: %v", err)
	}
	if err := conn.WriteMsg([]byte{0x03}); err != nil {
		t.Fatalf("WriteMsg#2: %v", err)
	}
	// c2s：客户端裸写一帧
	if _, err := cli.Write([]byte{0x02, 0, 0, 0, 0xAA, 0xBB}); err != nil {
		t.Fatalf("client write: %v", err)
	}
	cli.SetReadDeadline(time.Now().Add(2 * time.Second))
	got := make([]byte, 6)
	if _, err := readFull(cli, got); err != nil {
		t.Fatalf("client read: %v", err)
	}
	// 读回 c2s 帧确保字节已过三通
	frame, err := conn.ReadMsg()
	if err != nil {
		t.Fatalf("ReadMsg: %v", err)
	}
	if string(frame) != "\xAA\xBB" {
		t.Fatalf("ReadMsg = % X, want AA BB", frame)
	}
	conn.Close()

	base := captureBaseName(srv.RemoteAddr())
	c2s, err := os.ReadFile(filepath.Join(dir, base+"_c2s.bin"))
	if err != nil {
		t.Fatalf("读 c2s 落盘: %v", err)
	}
	s2c, err := os.ReadFile(filepath.Join(dir, base+"_s2c.bin"))
	if err != nil {
		t.Fatalf("读 s2c 落盘: %v", err)
	}
	wantC2S := []byte{0x02, 0, 0, 0, 0xAA, 0xBB}
	wantS2C := []byte{
		0x02, 0, 0, 0, 0x01, 0x02, // 帧1：len=2 body=01 02
		0x01, 0, 0, 0, 0x03, // 帧2：len=1 body=03
	}
	if string(c2s) != string(wantC2S) {
		t.Errorf("c2s = % X, want % X", c2s, wantC2S)
	}
	if string(s2c) != string(wantS2C) {
		t.Errorf("s2c = % X, want % X", s2c, wantS2C)
	}
}

func readFull(c gonet.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := c.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}
