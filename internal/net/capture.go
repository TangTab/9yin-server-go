package net

import (
	"fmt"
	"io"
	gonet "net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// 线路抓取：把客户端线路两个方向的原始字节（成帧+编码后的 wire 流，
// 未解码）按到达顺序追加落盘，文件名与外服对拍基准（captures/local，
// 见 docs/fork-era/fork-object-id-migration.md §2.1）完全同构：
//
//	{ip点换杠}-{端口}_c2s.bin   客户端→服务端原始字节
//	{ip点换杠}-{端口}_s2c.bin   服务端→客户端原始字节
//
// 开关：环境变量 NINEYIN_CAPTURE_DIR 非空即启用（值为落盘目录，相对
// 路径按进程工作目录解析）。抓取点在 Framer/Cipher 之外——直接包住
// 底层 TCP 流，因此内容与 pcapng-split.py 拆流产出的基准文件逐字节
// 同构，EE 转义、XOR 编码原样保留，可直接喂给现有解帧/对拍工具链。
// 抓取尽力而为：开文件失败只告警一次，该连接退化为明线路继续服务。
const captureEnv = "NINEYIN_CAPTURE_DIR"

// captureSeq 地址不可解析时的兜底序号（保证文件名仍唯一）。
var captureSeq uint64

// captureStream 在启用抓取时把 TCP 流包成双向落盘的三通流；未启用、
// 非 TCP 连接（如进程内管道）或落盘初始化失败时原样返回。
// NewConnection 是唯一调用方——login/entry 两条客户端线都从那里装配。
func captureStream(stream io.ReadWriteCloser) io.ReadWriteCloser {
	dir := strings.TrimSpace(os.Getenv(captureEnv))
	if dir == "" {
		return stream
	}
	tc, ok := stream.(gonet.Conn)
	if !ok {
		return stream
	}
	base := captureBaseName(tc.RemoteAddr())
	c2s, err1 := openCaptureSink(dir, base+"_c2s.bin")
	s2c, err2 := openCaptureSink(dir, base+"_s2c.bin")
	if err1 != nil || err2 != nil {
		if err1 == nil {
			c2s.close()
		}
		if err2 == nil {
			s2c.close()
		}
		fmt.Fprintf(os.Stderr, "net: capture %s 打开失败（%v/%v），本连接不抓取\n", base, err1, err2)
		return stream
	}
	fmt.Fprintf(os.Stderr, "net: capture %s → %s\n", base, dir)
	return &captureConn{Conn: tc, c2s: c2s, s2c: s2c}
}

// captureBaseName 由对端地址生成文件名基：IP 中的点与其余非字母数字
// 字符统一折成杠，再以杠拼端口——127.0.0.1:51512 → 127-0-0-1-51512，
// 与对拍基准命名逐字符一致。地址缺失/不可解析时退化为 conn-序号。
func captureBaseName(addr gonet.Addr) string {
	var s string
	if addr != nil {
		s = addr.String()
	}
	host, port, err := gonet.SplitHostPort(s)
	if err != nil || host == "" {
		return fmt.Sprintf("conn-%d", atomic.AddUint64(&captureSeq, 1))
	}
	var b strings.Builder
	for _, r := range host {
		if r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	b.WriteByte('-')
	b.WriteString(port)
	return b.String()
}

// captureSink 单方向追加落盘（O_APPEND：同源端口重连时续写同一文件，
// 与基准「一会话一对文件、按时间序拼接」的用法兼容）。
type captureSink struct {
	mu sync.Mutex
	f  *os.File
}

// openCaptureSink 建目录并开追加句柄。
func openCaptureSink(dir, name string) (*captureSink, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	return &captureSink{f: f}, nil
}

// append 同步追加一截原始字节后才返回，调用方缓冲可安全复用；
// 写失败静默（抓取不干扰线路）。
func (s *captureSink) append(p []byte) {
	if len(p) == 0 {
		return
	}
	s.mu.Lock()
	_, _ = s.f.Write(p)
	s.mu.Unlock()
}

func (s *captureSink) close() { _ = s.f.Close() }

// captureConn 三通流：Read 进来的字节落 c2s 盘、Write 出去的字节落
// s2c 盘，其余行为（截止时间/本地地址等）由内嵌 TCP 连接原样提供。
// 两方向各持独立锁——读循环与服务端推送可并发写各自的盘。
type captureConn struct {
	gonet.Conn
	c2s, s2c  *captureSink
	closeOnce sync.Once
}

func (c *captureConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		c.c2s.append(p[:n])
	}
	return n, err
}

func (c *captureConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	if n > 0 {
		c.s2c.append(p[:n])
	}
	return n, err
}

// Close 收口两个落盘句柄再关底层流（幂等）。
func (c *captureConn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.c2s.close()
		c.s2c.close()
		err = c.Conn.Close()
	})
	return err
}
