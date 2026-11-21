package gmpanel

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

// EchoCli 各服 echo 维护口客户端（行协议：连接即收 `welcome server N\r`，
// 发 `cmd\r`，应答 `>xxx\r`）。plist 一类多行应答以 `>end` 收尾
// （fssoloworld.EchoConsole 方言，本次为面板新增）；单行应答（status/
// players 等）无 end，靠空闲超时收束。
type EchoCli struct {
	host    string
	port    int
	timeout time.Duration
	idle    time.Duration // 应答行间空闲上限（无 >end 的单行应答收束）
}

// NewEchoCli 默认超时：连接 2s、行间空闲 800ms。
func NewEchoCli(host string, port int) *EchoCli {
	return &EchoCli{host: host, port: port, timeout: 2 * time.Second, idle: 800 * time.Millisecond}
}

// Command 发一行命令，返回应答行（不含前导 `>`）。
// 首行 welcome 不返回；读到 `>end` 或空闲超时即收。
func (c *EchoCli) Command(line string) ([]string, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.host, c.port), c.timeout)
	if err != nil {
		return nil, fmt.Errorf("echo %s:%d 不可达: %w", c.host, c.port, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(c.timeout))
	r := bufio.NewReader(conn)
	// welcome 握手（`welcome server N\r`）；读不到也不致命（方言差异）
	if _, err := r.ReadString('\r'); err != nil {
		return nil, fmt.Errorf("echo %s:%d 握手失败: %w", c.host, c.port, err)
	}
	if _, err := conn.Write([]byte(line + "\r")); err != nil {
		return nil, err
	}
	var out []string
	for {
		_ = conn.SetReadDeadline(time.Now().Add(c.idle))
		s, err := r.ReadString('\r')
		if e, ok := err.(net.Error); ok && e.Timeout() {
			break // 空闲收束（单行应答无 >end）
		}
		if err != nil {
			break // EOF / 连接关闭（quit 族应答后对端可能直接断）
		}
		s = trimEOL(s)
		if s == "" {
			continue
		}
		if s == ">end" {
			break
		}
		if len(s) > 1 && s[0] == '>' {
			s = s[1:]
		}
		if s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}

func trimEOL(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\r' || s[len(s)-1] == '\n') {
		s = s[:len(s)-1]
	}
	return s
}
