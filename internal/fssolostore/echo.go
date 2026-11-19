package fssolostore

import (
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// EchoConsole 复刻官方 echo 维护口（CEchoHook 承载 + Lua\solostore
// \main.lua echo_* 命令集；缺省 15100+StoreID）。
//
// 线协议与 fssologlobal/fssoloentry 同方言：\r 或 \n 任一即行终止、
// 行缓冲 1023、空行不分发、回显不补终止符（Lua 侧自带 "\r"）。
type EchoConsole struct {
	r       *reactor.Reactor
	onLine  func(line string) (reply string, closeConn bool)
	welcome string

	mu       sync.Mutex
	listener int
	conns    map[int]struct{}
	bufs     map[int][]byte
}

const echoLineMax = 1023

// NewEchoConsole 建控制台（welcome=官方 echo_open 欢迎语，
// onLine 返回回复与是否断开）。
func NewEchoConsole(r *reactor.Reactor, welcome string,
	onLine func(line string) (string, bool)) *EchoConsole {
	return &EchoConsole{
		r: r, onLine: onLine, welcome: welcome,
		listener: -1,
		conns:    map[int]struct{}{},
		bufs:     map[int][]byte{},
	}
}

// Start 开受面（官方 CEchoHook::Start）。
func (e *EchoConsole) Start(addr string, port int) error {
	id, err := e.r.CreateListener(addr, port, 4, reactor.ConnHandler{
		OnOpen:  e.onOpen,
		OnClose: e.onClose,
		OnInput: e.onInput,
	})
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.listener = id
	e.mu.Unlock()
	return nil
}

func (e *EchoConsole) onOpen(connID int, host string, port int) {
	e.mu.Lock()
	e.conns[connID] = struct{}{}
	e.bufs[connID] = nil
	e.mu.Unlock()
	e.r.Send(connID, []byte(e.welcome))
}

func (e *EchoConsole) onClose(connID int, host string, port int) {
	e.mu.Lock()
	delete(e.conns, connID)
	delete(e.bufs, connID)
	e.mu.Unlock()
}

func (e *EchoConsole) onInput(connID int, data []byte) {
	e.mu.Lock()
	e.conns[connID] = struct{}{} // OnInput 早于 OnOpen 的防御（reactor 语义）
	buf := append(e.bufs[connID], data...)
	e.bufs[connID] = buf
	e.mu.Unlock()
	for {
		idx := -1
		for i, c := range buf {
			if c == '\r' || c == '\n' {
				idx = i
				break
			}
		}
		if idx < 0 {
			if len(buf) > echoLineMax {
				e.mu.Lock()
				e.bufs[connID] = nil
				e.mu.Unlock()
			}
			return
		}
		line := string(buf[:idx])
		rest := append([]byte(nil), buf[idx+1:]...)
		e.mu.Lock()
		e.bufs[connID] = rest
		e.mu.Unlock()
		buf = rest
		if line == "" {
			continue
		}
		if e.onLine != nil {
			reply, closeConn := e.onLine(line)
			if reply != "" {
				e.r.Send(connID, []byte(reply))
			}
			if closeConn {
				e.r.CloseConnect(connID)
				return
			}
		}
	}
}

// SendLine 广播一行（官方 SendEchoLine）。
func (e *EchoConsole) SendLine(line string) {
	e.mu.Lock()
	conns := make([]int, 0, len(e.conns))
	for id := range e.conns {
		conns = append(conns, id)
	}
	e.mu.Unlock()
	for _, id := range conns {
		e.r.Send(id, []byte(line))
	}
}

// Close 关受面（CloseEcho）。
func (e *EchoConsole) Close() {
	e.mu.Lock()
	id := e.listener
	e.listener = -1
	e.mu.Unlock()
	if id >= 0 {
		e.r.CloseConnect(id)
	}
}
