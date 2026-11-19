package fssolomonitor

import (
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// EchoConsole 复刻官方 FsSoloMonitor 版 CEchoHook（echohook.cpp 多客户
// 户端形态，[Echo] 缺省 127.0.0.1:15012）。
//
// 与 fssologlobal 单客户端版的差异（官方 echohook.cpp 差异定案）：
//   - 上限 64 连接（m_Clients 池 + connector 下标映射；超出 trace
//     "client array too big" 并断开后来者）；
//   - 每客户端独立 0x400 行缓冲（上限 0x3FF、\r/\n 皆终止、空行不分发）；
//   - SendLine(0,0,info) → 广播全部 echo 客户端（告警 alert_monitor 下行）；
//   - Close() → 批量关闭（官方 end 命令 → CloseEcho 的整批语义）。
//
// 欢迎语/命令集来自官方 Lua\solomonitor\main.lua：welcome monitor
// server\r；quit/end/undefined command（">"+info+"\r" 应答格式）。
type EchoConsole struct {
	r      *reactor.Reactor
	onLine func(line string) (reply string, closeAll bool)

	mu       sync.Mutex
	listener int
	conns    map[int]struct{}
	bufs     map[int][]byte
}

// echoMaxClients 官方上限 64（0x40）。
const echoMaxClients = 64

// echoLineMax 官方行缓冲上限（0x3FF）。
const echoLineMax = 0x3FF

// NewEchoConsole 建控制台。onLine 返回 (应答, 是否整批关闭)——
// closeAll 对应官方 end → CloseEcho()（批量断开）。
func NewEchoConsole(r *reactor.Reactor, onLine func(line string) (string, bool)) *EchoConsole {
	return &EchoConsole{
		r: r, onLine: onLine,
		listener: -1,
		conns:    map[int]struct{}{},
		bufs:     map[int][]byte{},
	}
}

// Start 开 echo 受面（官方 CEchoHook::Start）。
func (e *EchoConsole) Start(addr string, port int, logf func(string, ...any)) error {
	id, err := e.r.CreateListener(addr, port, echoMaxClients, reactor.ConnHandler{
		OnOpen: func(connID int, a string, p int) {
			e.mu.Lock()
			if len(e.conns) >= echoMaxClients {
				e.mu.Unlock()
				if logf != nil {
					logf("client array too big")
				}
				e.r.ShutdownConnector(connID) // 官方：超限拒后来者
				return
			}
			e.conns[connID] = struct{}{}
			e.bufs[connID] = nil
			e.mu.Unlock()
			if logf != nil {
				logf("echo client %s:%d connected", a, p)
			}
			// 官方 main.lua echo_open → SendEchoLine("welcome monitor
			// server\r")（hook 补 \r\n → 线上 "...server\r\r\n"）。
			e.sendTo(connID, "welcome monitor server\r")
		},
		OnClose: func(connID int, a string, p int) {
			e.mu.Lock()
			delete(e.conns, connID)
			delete(e.bufs, connID)
			e.mu.Unlock()
			if logf != nil {
				logf("echo client %s:%d closed", a, p)
			}
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

// Stop 清引用与缓冲（监听随 Reactor::Stop 统一关闭）。
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
				reply, closeAll := e.onLine(string(line))
				if reply != "" {
					e.SendLine(reply)
				}
				if closeAll {
					e.Close() // 官方 end → CloseEcho（批量关闭）
					return
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
	if _, ok := e.bufs[connID]; ok {
		e.bufs[connID] = buf
	}
	e.mu.Unlock()
}

// sendTo 官方 CEchoHook::SendLine 单发形态：负载后补 "\r\n"。
func (e *EchoConsole) sendTo(connID int, text string) {
	e.r.Send(connID, []byte(text+"\r\n"))
}

// SendLine 官方 SendLine(0,0,info)：负载后补 "\r\n"（echohook.cpp
// 差异定案：TAutoMem(len+2) memcpy + "\r\n"），广播全部 echo 连接。
func (e *EchoConsole) SendLine(text string) {
	e.mu.Lock()
	conns := make([]int, 0, len(e.conns))
	for id := range e.conns {
		conns = append(conns, id)
	}
	e.mu.Unlock()
	for _, id := range conns {
		e.sendTo(id, text)
	}
}

// Close 官方 Close()：批量关闭全部 echo 连接。
func (e *EchoConsole) Close() {
	e.mu.Lock()
	conns := make([]int, 0, len(e.conns))
	for id := range e.conns {
		conns = append(conns, id)
	}
	e.conns = map[int]struct{}{}
	e.bufs = map[int][]byte{}
	e.mu.Unlock()
	for _, id := range conns {
		e.r.ShutdownConnector(id)
	}
}

// ConnCount 呈现用连接数。
func (e *EchoConsole) ConnCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.conns)
}
