package fssololog

import (
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// EchoConsole 复刻 CEchoHook（echohook.cpp）单连接形态：
// solo 集群的 echo 维护口与其它模块同源，但连接模型为**单主连接**
// （新连接到来先删旧主连接、行缓冲清零），SendLine 追加 "\r\n"
// （官方 TAutoMem len+2）——Lua 回显串自带 "\r" 时客户端见双换行
// （官方行为）。
//
// 线协议（官方 OnReceive 循环实证）：逐字节扫描，'\n'/'\r' 任一即
// 行终止；行缓冲上限 1023（0x400-1，超长丢弃）；空行不分发；
// 仅处理主连接。
type EchoConsole struct {
	r    *reactor.Reactor
	disp func(line string) (reply string, closeConn bool)

	mu        sync.Mutex
	listener  int
	connector int    // m_nConnector（-1=无主连接）
	buf       []byte // 0x400 行缓冲
}

// echoLineMax 官方行缓冲上限（1023）。
const echoLineMax = 1023

// NewEchoConsole 建控制台（disp = 命令分发，宿主提供；quit 的停服
// 动作由宿主在 disp 内异步触发——官方 close()：WriteLine+Stop+nx_quit）。
func NewEchoConsole(r *reactor.Reactor, disp func(line string) (string, bool)) *EchoConsole {
	return &EchoConsole{r: r, disp: disp, listener: -1, connector: -1}
}

// Start 开 echo 受面（官方 Start(addr, port) 断言 addr 非空且 port>0；
// 监听 backlog 官方传 1）。连接建立即换主并推欢迎行（main.lua
// on_open → SendEchoLine("welcome log server")）。
func (e *EchoConsole) Start(addr string, port int) error {
	id, err := e.r.CreateListener(addr, port, 1, reactor.ConnHandler{
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

// Stop 清面（监听随 Reactor::Stop 统一关闭）。
func (e *EchoConsole) Stop() {
	e.mu.Lock()
	e.listener = -1
	e.connector = -1
	e.buf = nil
	e.mu.Unlock()
}

// onOpen 官方 OnAccept：删旧主连接、换主、行缓冲清零、触发
// on_open（welcome log server）。
func (e *EchoConsole) onOpen(connID int, addr string, port int) {
	e.mu.Lock()
	old := e.connector
	e.connector = connID
	e.buf = nil
	e.mu.Unlock()
	if old >= 0 && old != connID {
		e.r.CloseConnect(old) // DeleteConnector 旧主
	}
	e.SendLine("welcome log server")
}

// onClose 官方 OnClose：仅当 id == m_nConnector 才算主连接离线。
func (e *EchoConsole) onClose(connID int, addr string, port int) {
	e.mu.Lock()
	if e.connector == connID {
		e.connector = -1
		e.buf = nil
	}
	e.mu.Unlock()
}

// onInput 官方 OnReceive：仅处理主连接；逐字节扫描，\r/\n 任一即
// 行终止；行缓冲非空才分发（空行不分发）。
func (e *EchoConsole) onInput(connID int, data []byte) {
	e.mu.Lock()
	if connID != e.connector {
		e.mu.Unlock()
		return
	}
	var lines []string
	for _, c := range data {
		if c == '\n' || c == '\r' {
			if len(e.buf) > 0 {
				lines = append(lines, string(e.buf))
			}
			e.buf = nil
			continue
		}
		if len(e.buf) < echoLineMax {
			e.buf = append(e.buf, c)
		} // 超长字节丢弃
	}
	e.mu.Unlock()
	for _, line := range lines {
		reply, closeConn := e.disp(line)
		if reply != "" {
			e.SendLine(reply)
		}
		if closeConn {
			e.Close() // 官方 end → CloseEcho
		}
	}
}

// SendLine 官方 SendLine：主连接非空才发（memcpy + "\r\n"，len+2）。
func (e *EchoConsole) SendLine(text string) bool {
	e.mu.Lock()
	id := e.connector
	e.mu.Unlock()
	if id < 0 {
		return false
	}
	return e.r.Send(id, append([]byte(text), "\r\n"...))
}

// Close 官方 CloseEcho：ShutdownConnector 优雅半关闭主连接。
func (e *EchoConsole) Close() {
	e.mu.Lock()
	id := e.connector
	e.connector = -1
	e.buf = nil
	e.mu.Unlock()
	if id >= 0 {
		e.r.ShutdownConnector(id)
	}
}

// ConnCount 呈现用（0/1，单连接模型）。
func (e *EchoConsole) ConnCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.connector >= 0 {
		return 1
	}
	return 0
}
