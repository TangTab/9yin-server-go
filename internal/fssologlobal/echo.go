package fssologlobal

import (
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// EchoConsole 复刻官方 CEchoHook（echohook.cpp）承载的 echo 维护口
// （TCP，官方 [Echo] 缺省 127.0.0.1:15012）。
//
// 线协议与外网命令通道同方言（SoloGlobal::ReceiveClientMsg 实证的
// 官方语义）：逐字节扫描、'\n'(0x0A) 或 '\r'(0x0D) 任一即行终止、
// 行缓冲上限 1023 字节（超长丢弃）、空行不分发；SendLine 原样发送
// 不补终止符（官方 Lua 侧自带 "\r"）。
//
// 官方把行交脚本层 on_input（main.lua：quit/end + 实体自省族）；
// Go 复刻为 OnLine 回调（宿主命令集见 Server.echoCommand），差异
// 见包文档。
type EchoConsole struct {
	r      *reactor.Reactor
	onLine func(line string) (reply string, closeConn bool)

	mu       sync.Mutex
	listener int
	conns    map[int]struct{}
	bufs     map[int][]byte
}

// echoLineMax 官方行缓冲上限（1023）。
const echoLineMax = 1023

// NewEchoConsole 建控制台。
func NewEchoConsole(r *reactor.Reactor, onLine func(line string) (string, bool)) *EchoConsole {
	return &EchoConsole{
		r: r, onLine: onLine,
		listener: -1,
		conns:    map[int]struct{}{},
		bufs:     map[int][]byte{},
	}
}

// Start 开 echo 受面（官方 CEchoHook::Start(addr, port)）。
func (e *EchoConsole) Start(addr string, port int) error {
	id, err := e.r.CreateListener(addr, port, 4, reactor.ConnHandler{
		OnOpen: func(connID int, a string, p int) {
			e.mu.Lock()
			e.conns[connID] = struct{}{}
			e.bufs[connID] = nil
			e.mu.Unlock()
			// 官方 main.lua echo_open：welcome global server
			e.r.Send(connID, []byte("welcome global server\r"))
		},
		OnClose: func(connID int, a string, p int) {
			e.mu.Lock()
			delete(e.conns, connID)
			delete(e.bufs, connID)
			e.mu.Unlock()
		},
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

// Stop 关 echo 受面（官方 CEchoHook::Stop；Go reactor 的监听随
// Reactor::Stop 统一关闭，此处清引用与缓冲）。
func (e *EchoConsole) Stop() {
	e.mu.Lock()
	e.listener = -1
	e.conns = map[int]struct{}{}
	e.bufs = map[int][]byte{}
	e.mu.Unlock()
}

// onInput 逐字节行扫描（官方语义：\r/\n 皆终止、超长丢弃、空行不分发）。
func (e *EchoConsole) onInput(connID int, data []byte) {
	e.mu.Lock()
	buf := e.bufs[connID]
	for _, c := range data {
		if c == '\n' || c == '\r' {
			line := buf
			buf = nil
			e.bufs[connID] = buf
			e.mu.Unlock()
			if len(line) > 0 {
				reply, closeConn := e.onLine(string(line))
				if reply != "" {
					e.SendLine(reply)
				}
				if closeConn {
					e.r.ShutdownConnector(connID) // 官方 end → CloseEcho
				}
			}
			e.mu.Lock()
			buf = e.bufs[connID]
			continue
		}
		if len(buf) < echoLineMax {
			buf = append(buf, c)
		} // 超长字节丢弃
	}
	e.bufs[connID] = buf
	e.mu.Unlock()
}

// SendLine 官方 SendEchoLine → CEchoHook::SendLine：向全部 echo 连接
// 原样发送（不补终止符）。
func (e *EchoConsole) SendLine(text string) {
	e.mu.Lock()
	conns := make([]int, 0, len(e.conns))
	for id := range e.conns {
		conns = append(conns, id)
	}
	e.mu.Unlock()
	for _, id := range conns {
		e.r.Send(id, []byte(text))
	}
}

// ConnCount 呈现用连接数。
func (e *EchoConsole) ConnCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.conns)
}
