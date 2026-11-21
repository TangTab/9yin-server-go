package fssolopublic

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// EchoConsole 复刻官方 CEchoHook 承载的 echo 维护口（TCP，官方 [Echo]
// 缺省 127.0.0.1:15003）。线协议与 SoloGlobal 同方言：逐字节扫描、
// '\n'/'\r' 任一即行终止、行缓冲上限 1023、空行不分发、SendLine 原样
// 发送不补终止符。
//
// 官方把行交脚本层 on_input（lua\solopublic\main.lua：quit/end +
// 实体自省族十二回调，欢迎语 "welcome public server <S>\r"）；Go 复刻
// 为内置命令集（差异见包文档）。
type EchoConsole struct {
	r        *reactor.Reactor
	serverID uint32
	commands func(line string) (string, bool) // 内置命令集（close 联动）

	mu       sync.Mutex
	listener int
	conns    map[int]struct{}
	bufs     map[int][]byte
}

const echoLineMax = 1023

func NewEchoConsole(r *reactor.Reactor, serverID uint32, commands func(line string) (string, bool)) *EchoConsole {
	return &EchoConsole{
		r: r, serverID: serverID, commands: commands,
		listener: -1,
		conns:    map[int]struct{}{},
		bufs:     map[int][]byte{},
	}
}

// Start 开 echo 受面；欢迎语 "welcome public server <S>\r"。
func (e *EchoConsole) Start(addr string, port int) error {
	id, err := e.r.CreateListener(addr, port, 4, reactor.ConnHandler{
		OnOpen: func(connID int, a string, p int) {
			e.mu.Lock()
			e.conns[connID] = struct{}{}
			e.bufs[connID] = nil
			e.mu.Unlock()
			e.r.Send(connID, []byte(fmt.Sprintf("welcome public server %d\r", e.serverID)))
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

// ListenerID echo 受面 reactor listener 句柄（-1=未开）。
func (e *EchoConsole) ListenerID() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.listener
}

// Stop 关 echo 受面（监听随 Reactor::Stop 统一关闭，此处清引用）。
func (e *EchoConsole) Stop() {
	e.mu.Lock()
	e.listener = -1
	e.conns = map[int]struct{}{}
	e.bufs = map[int][]byte{}
	e.mu.Unlock()
}

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
				reply, closeConn := e.dispatch(string(line))
				if reply != "" {
					e.r.Send(connID, []byte(reply))
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
		}
	}
	e.bufs[connID] = buf
	e.mu.Unlock()
}

// dispatch 官方语义：quit→停服、end→断连、未定义命令回
// ">undefined command: %s\r"；其余交内置命令集。
func (e *EchoConsole) dispatch(line string) (string, bool) {
	switch line {
	case "quit":
		if e.commands != nil {
			if reply, _ := e.commands(line); reply != "" {
				e.SendLine(reply)
			}
		}
		return ">server will shutdown...\r", false
	case "end":
		return "", true
	}
	if e.commands != nil {
		if reply, handled := e.commands(line); handled {
			return reply, false
		}
	}
	return ">undefined command: " + line + "\r", false
}

// SendLine 官方 SendEchoLine → 向全部 echo 连接原样发送。
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

// echoHelpText 内置命令集帮助。
const echoHelpText = `commands: state | spaces | datas <space> | players | scenes | names | relations | feeds | trades | help | quit | end`

// formatKV 辅助呈现（键值对排序拼接）。
func formatKV(kv map[string]int) string {
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s=%d", k, kv[k])
	}
	return b.String()
}
