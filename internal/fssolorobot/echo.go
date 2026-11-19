package fssolorobot

import (
	"fmt"
	"net"
	"sync"
)

// EchoConsole 复刻官方 CEchoHook（tools/echohook.cpp）承载的 echo
// 维护口：TCP 受面（[Echo]，缺省 127.0.0.1:15009），**单客户端**
// （已有连接时后来者直接断开，官方 OnAccept@0x10046200 DeleteConnector
// 拒绝语义），'\r'/'\n' 任一即行终止。
//
// 官方把行交脚本层 on_input（Lua\solorobot\main.lua）：echo_open 回
// "welcome game robot\r"；echo_input：quit → "server will shutdown..."
// 后关停进程、end → 断本连接、其余 "undefined command: %s"；应答
// 格式 echo_response = ">" .. info .. "\r"。
type EchoConsole struct {
	onLine func(line string) (reply string, closeConn bool)

	mu       sync.Mutex
	listener net.Listener
	client   net.Conn
	closed   bool
}

// NewEchoConsole 建控制台；onLine 返回回复行（不含 ">" 前缀与 "\r"）
// 与是否断开本连接；nil 回复 = 不应答。
func NewEchoConsole(onLine func(string) (string, bool)) *EchoConsole {
	return &EchoConsole{onLine: onLine}
}

// Start 开受面（官方 CEchoHook::Start(addr, port)）。
func (e *EchoConsole) Start(addr string, port int) error {
	ln, err := net.Listen("tcp", net.JoinHostPort(addr, fmt.Sprint(port)))
	if err != nil {
		return err
	}
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		ln.Close()
		return fmt.Errorf("echo console stopped")
	}
	e.listener = ln
	e.mu.Unlock()
	go e.acceptLoop(ln)
	return nil
}

func (e *EchoConsole) acceptLoop(ln net.Listener) {
	for {
		nc, err := ln.Accept()
		if err != nil {
			return
		}
		e.mu.Lock()
		if e.closed {
			e.mu.Unlock()
			nc.Close()
			return
		}
		if e.client != nil {
			// 官方单客户端：拒绝后来者。
			e.mu.Unlock()
			nc.Close()
			continue
		}
		e.client = nc
		e.mu.Unlock()
		go e.serve(nc)
	}
}

func (e *EchoConsole) serve(nc net.Conn) {
	// 官方 main.lua echo_open：welcome game robot
	nc.Write([]byte("welcome game robot\r"))
	// 官方逐字节入行缓冲，'\r'(0x0D) 或 '\n'(0x0A) 任一即行终止。
	buf := make([]byte, 0, 128)
	rb := make([]byte, 1)
	for {
		_, err := nc.Read(rb)
		if err != nil {
			e.dropClient(nc)
			return
		}
		b := rb[0]
		if b == '\r' || b == '\n' {
			line := string(buf)
			buf = buf[:0]
			if line == "" {
				continue // 官方空行不分发
			}
			reply, closeConn := e.onLine(line)
			if reply != "" {
				if _, err := nc.Write([]byte(">" + reply + "\r")); err != nil {
					e.dropClient(nc)
					return
				}
			}
			if closeConn {
				nc.Close()
				e.dropClient(nc)
				return
			}
			continue
		}
		if len(buf) < 1024 {
			buf = append(buf, b)
		}
	}
}

func (e *EchoConsole) dropClient(nc net.Conn) {
	e.mu.Lock()
	if e.client == nc {
		e.client = nil
	}
	e.mu.Unlock()
	nc.Close()
}

// SendLine 官方 SendEchoLine → CEchoHook::SendLine：向 echo 客户端
// 原样发送（Lua 侧自带 "\r"）。
func (e *EchoConsole) SendLine(s string) bool {
	e.mu.Lock()
	nc := e.client
	e.mu.Unlock()
	if nc == nil {
		return false
	}
	_, err := nc.Write([]byte(s))
	return err == nil
}

// Close 官方 CloseEcho → CEchoHook::Close：断当前 echo 客户端。
func (e *EchoConsole) Close() {
	e.mu.Lock()
	nc := e.client
	e.client = nil
	e.mu.Unlock()
	if nc != nil {
		nc.Close()
	}
}

// Stop 官方 CEchoHook::Stop：关受面与全部连接（随 SoloRobot::Stop）。
func (e *EchoConsole) Stop() {
	e.mu.Lock()
	e.closed = true
	ln := e.listener
	e.listener = nil
	nc := e.client
	e.client = nil
	e.mu.Unlock()
	if ln != nil {
		ln.Close()
	}
	if nc != nil {
		nc.Close()
	}
}
