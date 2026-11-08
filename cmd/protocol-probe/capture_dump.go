package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// captureTap 把一条连接的原始线上字节（XOR+0xEE 转义后的最终形态）逐字节
// tee 进 captures/local/<ip-pid>_c2s.bin 与 _s2c.bin。命名与 F 盘参考服/
// 新版服务端留下的抓包完全一致（如 127-0-0-1-55866_c2s.bin），因此
// tools/smoke/capture_deep.py 等现成解析器可以直接对拍本服会话。
//
// 记录点刻意放在 transport 编解码之外：tee 的是 socket 上的真实字节，
// 与抓包器看到的流完全相同；密钥中途轮换也会被如实保留。
type captureTap struct {
	conn net.Conn
	c2s  *bufio.Writer
	s2c  *bufio.Writer
	c2sF *os.File
	s2cF *os.File
}

// captureSessionPrefix 把远端地址 127.0.0.1:55866 规范成抓包文件前缀
// 127-0-0-1-55866。
func captureSessionPrefix(remote net.Addr) string {
	host, port, err := net.SplitHostPort(remote.String())
	if err != nil {
		host, port = remote.String(), "x"
	}
	sanitize := func(s string) string {
		var b strings.Builder
		for _, r := range s {
			switch {
			case r >= '0' && r <= '9', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '-':
				b.WriteRune(r)
			default:
				b.WriteRune('-')
			}
		}
		return b.String()
	}
	return sanitize(host) + "-" + sanitize(port)
}

// newCaptureTap 在 baseDir 下为该连接建立一对截获文件；失败时返回 nil，
// 绝不因为记录问题影响游戏会话本身。
func newCaptureTap(conn net.Conn, baseDir string) *captureTap {
	if baseDir == "" || conn.RemoteAddr() == nil {
		return nil
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		log.Printf("capture: create %s: %v", baseDir, err)
		return nil
	}
	prefix := filepath.Join(baseDir, captureSessionPrefix(conn.RemoteAddr()))
	c2sF, err := os.Create(prefix + "_c2s.bin")
	if err != nil {
		log.Printf("capture: create c2s dump: %v", err)
		return nil
	}
	s2cF, err := os.Create(prefix + "_s2c.bin")
	if err != nil {
		_ = c2sF.Close()
		log.Printf("capture: create s2c dump: %v", err)
		return nil
	}
	return &captureTap{
		conn: conn,
		c2s:  bufio.NewWriterSize(c2sF, 1<<16),
		s2c:  bufio.NewWriterSize(s2cF, 1<<16),
		c2sF: c2sF,
		s2cF: s2cF,
	}
}

func (t *captureTap) Read(p []byte) (int, error) {
	n, err := t.conn.Read(p)
	if n > 0 {
		_, _ = t.c2s.Write(p[:n])
		// 逐读即刷：上行流量小（几 KB/分钟），64KB 缓冲若只靠 Close
		// 落盘，会话进行中 c2s 恒为 0 字节——真机回归时正在测的会话
		// 拿不到上行数据（55052 会话诊断受阻）。
		_ = t.c2s.Flush()
	}
	return n, err
}

func (t *captureTap) Write(p []byte) (int, error) {
	n, err := t.conn.Write(p)
	if n > 0 {
		_, _ = t.s2c.Write(p[:n])
		_ = t.s2c.Flush()
	}
	return n, err
}

func (t *captureTap) Close() error {
	_ = t.c2s.Flush()
	_ = t.s2c.Flush()
	_ = t.c2sF.Close()
	_ = t.s2cF.Close()
	return t.conn.Close()
}

// captureTapLogName 仅供日志展示。
func (t *captureTap) logName(baseDir string) string {
	return filepath.Join(baseDir, captureSessionPrefix(t.conn.RemoteAddr()))
}

// capturedConn 把 tap 伪装回 net.Conn：地址与超时控制透传底层连接，
// 只有 Read/Write/Close 走截获逻辑。
type capturedConn struct {
	net.Conn
	tap *captureTap
}

func newCapturedConn(conn net.Conn, baseDir string) (net.Conn, *captureTap) {
	tap := newCaptureTap(conn, baseDir)
	if tap == nil {
		return conn, nil
	}
	return &capturedConn{Conn: conn, tap: tap}, tap
}

func (c *capturedConn) Read(p []byte) (int, error) { return c.tap.Read(p) }

func (c *capturedConn) Write(p []byte) (int, error) { return c.tap.Write(p) }

func (c *capturedConn) Close() error { return c.tap.Close() }

var _ net.Conn = (*capturedConn)(nil)
