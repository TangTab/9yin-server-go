package fscomm

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

// EchoSock 行缓冲上限：官方 0x3ff（echosock.cpp 0x10062825 处
// cmp ecx, 0x3ff），超长字节丢弃直至行终止符。
const echoLineMax = 1023

// EchoLineHandler 单命令回调（官方 "on_input_<cmd>" 动态回调）。
type EchoLineHandler func(args []string)

// EchoSockHandler 官方 EchoSockHook 回调面（echosockhook.cpp）。
type EchoSockHandler struct {
	OnOpen  func(addr string, port int) // "on_open"
	OnClose func(addr string, port int) // "on_close"
	OnInput func(line string)           // "on_input"：整行兜底
	// OnCommand 注册具名命令（"on_input_<首词>"）。EchoSock 命中
	// 即独占分发，不再走 OnInput——与官方优先级一致。
	OnCommand map[string]EchoLineHandler
}

// EchoSock 复刻官方 EchoSock+EchoSockHook：维护口 TCP 控制台。
// 单客户端模型（官方 Connected/RemoteAddress/RemotePort 三属性
// 印证同时只服务一个连接）。行协议：'\r' 或 '\n' 终止，空行不分发。
type EchoSock struct {
	addr string
	port int
	h    EchoSockHandler

	mu       sync.Mutex
	ln       net.Listener
	conn     net.Conn
	line     [echoLineMax + 1]byte
	lineLen  int
	wg       sync.WaitGroup
	stopOnce sync.Once
	stopped  chan struct{}
}

// NewEchoSock 创建；addr 空则回环。
func NewEchoSock(addr string, port int, h EchoSockHandler) *EchoSock {
	if addr == "" {
		addr = "127.0.0.1"
	}
	return &EchoSock{addr: addr, port: port, h: h, stopped: make(chan struct{})}
}

// Start 侦听（官方 EchoSock::Start；失败文案对齐
// "(EchoSock::Execute)bind error" 等）。
func (e *EchoSock) Start() error {
	ln, err := net.Listen("tcp", joinHostPort(e.addr, e.port))
	if err != nil {
		return fmt.Errorf("fscomm: (EchoSock::Execute)bind error: %w", err)
	}
	e.mu.Lock()
	e.ln = ln
	e.mu.Unlock()
	e.wg.Add(1)
	go e.serve(ln)
	return nil
}

// Addr 实际监听地址。
func (e *EchoSock) Addr() net.Addr {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.ln == nil {
		return nil
	}
	return e.ln.Addr()
}

func (e *EchoSock) serve(ln net.Listener) {
	defer e.wg.Done()
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		select {
		case <-e.stopped:
			c.Close()
			return
		default:
		}
		e.accept(c)
	}
}

// accept 官方单客户端语义：新连接顶掉旧连接。
func (e *EchoSock) accept(c net.Conn) {
	e.mu.Lock()
	old := e.conn
	e.conn = c
	e.lineLen = 0
	remote := c.RemoteAddr().String()
	e.mu.Unlock()
	if old != nil {
		old.Close()
	}
	host, port := splitHostPort(remote)
	if e.h.OnOpen != nil {
		e.h.OnOpen(host, port)
	}
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := c.Read(buf)
			if n > 0 {
				e.feed(buf[:n])
			}
			if err != nil {
				e.mu.Lock()
				if e.conn == c {
					e.conn = nil
				}
				e.mu.Unlock()
				if e.h.OnClose != nil {
					e.h.OnClose(host, port)
				}
				return
			}
		}
	}()
}

// feed 官方 0x10062790 逐字节行扫描的等价实现：'\n'/'\r' 终止，
// 行长 >0 才分发，超限丢弃。
func (e *EchoSock) feed(chunk []byte) {
	for _, b := range chunk {
		if b == '\n' || b == '\r' {
			e.mu.Lock()
			line := string(e.line[:e.lineLen])
			e.lineLen = 0
			e.mu.Unlock()
			if len(line) > 0 {
				e.dispatch(line)
			}
			continue
		}
		e.mu.Lock()
		if e.lineLen < echoLineMax {
			e.line[e.lineLen] = b
			e.lineLen++
		}
		e.mu.Unlock()
	}
}

// dispatch 官方 CEchoSockHook::OnReceive 分发序：切词（空格/制表）
// → 首词命中 "on_input_<cmd>" → 未命中回落 OnInput(整行)。
func (e *EchoSock) dispatch(line string) {
	fields := strings.FieldsFunc(line, func(r rune) bool {
		return r == ' ' || r == '\t'
	})
	if len(fields) > 0 {
		if h, ok := e.h.OnCommand[fields[0]]; ok && h != nil {
			h(fields[1:])
			return
		}
	}
	if e.h.OnInput != nil {
		e.h.OnInput(line)
	}
}

// SendLine 原样发送不补终止符（官方 EchoSock::SendLine→Send(info,0)
// 直通语义；对端换行由调用方自带，官方 Lua 侧拼 "\r"）。
func (e *EchoSock) SendLine(s string) error {
	e.mu.Lock()
	c := e.conn
	e.mu.Unlock()
	if c == nil {
		return fmt.Errorf("fscomm: (EchoSock::Send)not connected")
	}
	_, err := c.Write([]byte(s))
	return err
}

// Connected / Remote 官方只读属性。
func (e *EchoSock) Connected() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.conn != nil
}

// Stop 关停（官方 EchoSock::Stop/Close）。
func (e *EchoSock) Stop() {
	e.stopOnce.Do(func() { close(e.stopped) })
	e.mu.Lock()
	ln := e.ln
	e.ln = nil
	c := e.conn
	e.conn = nil
	e.mu.Unlock()
	if ln != nil {
		ln.Close()
	}
	if c != nil {
		c.Close()
	}
	e.wg.Wait()
}

func splitHostPort(addr string) (string, int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, 0
	}
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	return host, port
}
