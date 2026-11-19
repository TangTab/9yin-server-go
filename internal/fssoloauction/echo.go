package fssoloauction

import (
	"fmt"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// EchoConsole 复刻 CEchoHook（echohook.cpp）：echo 调试口（TCP 16001）。
//
// 线协议（官方 OnReceive @0x10067190 循环实证）：
//   - recv 缓冲逐字节扫描，'\n'(0x0A) 或 '\r'(0x0D) 任一即行终止
//   - 行缓冲上限 1023 字节（超长字节丢弃，继续扫描终止符）
//   - 空行不分发
//   - SendLine 原样发送不补终止符（Lua 侧自带 "\r"）
//
// 官方把行交 IEntScript 当 Lua 表达式求值，结果调全局 on_input(...)；
// Go 复刻为 OnLine 回调（宿主提供命令集），差异见包文档。
type EchoConsole struct {
	r      *reactor.Reactor
	onLine func(line string) string // 返回应答文本（空串=不回）

	mu       sync.Mutex
	listener int
	conns    map[int]struct{}
	bufs     map[int][]byte
}

// echoLineMax 官方行缓冲上限（1023）。
const echoLineMax = 1023

// NewEchoConsole 建控制台。
func NewEchoConsole(r *reactor.Reactor, onLine func(line string) string) *EchoConsole {
	return &EchoConsole{
		r: r, onLine: onLine,
		listener: -1,
		conns:    map[int]struct{}{},
		bufs:     map[int][]byte{},
	}
}

// Start 开 echo 受面（官方 Start(addr, port)，EchoAddr 空则不开）。
func (e *EchoConsole) Start(addr string, port int) error {
	id, err := e.r.CreateListener(addr, port, 4, reactor.ConnHandler{
		OnOpen: func(connID int, a string, p int) {
			e.mu.Lock()
			e.conns[connID] = struct{}{}
			e.bufs[connID] = nil
			e.mu.Unlock()
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
				if reply := e.onLine(string(line)); reply != "" {
					e.SendLine(reply)
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

// SendLine 官方 SendLine：向全部 echo 连接原样发送（不补终止符）。
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

// echoWelcome 连接后首行提示（Go 扩展；官方无）。
func echoWelcome() string {
	return fmt.Sprintf("9yin auction echo（help 查看命令；行协议 \\r/\\n 终止，缓冲上限 %d）\r\n", echoLineMax)
}
