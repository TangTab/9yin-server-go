package fssoloworld

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

// EchoConsole 复刻 CEchoHook（§十一）：行协议维护控制台。
// 欢迎语 `welcome server %d\r`；命令 quit/fquit/end/players/open/close/status
// + 调试命令族框架（getp/setp/getc/setc/method/listp/listc/listm/listg/
// function/debugger —— 官方走 nx_property 反射，此处框架保留）。
type EchoConsole struct {
	srv     *Server
	mu      sync.Mutex
	ln      net.Listener
	clients map[net.Conn]chan struct{}
	closed  bool
	logf    func(format string, args ...any)
}

// StartEcho 开维护受面。
func (s *Server) StartEcho(addr string, port int) error {
	if addr == "" {
		return nil // 官方 EchoAddr 空 → 不启
	}
	ec := &EchoConsole{srv: s, clients: map[net.Conn]chan struct{}{}, logf: s.opt.Logf}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return err
	}
	ec.ln = ln
	s.echo = ec
	go ec.accept()
	return nil
}

func (e *EchoConsole) accept() {
	for {
		nc, err := e.ln.Accept()
		if err != nil {
			return
		}
		done := make(chan struct{})
		e.mu.Lock()
		if e.closed {
			e.mu.Unlock()
			nc.Close()
			return
		}
		e.clients[nc] = done
		e.mu.Unlock()
		go e.serve(nc, done)
	}
}

func (e *EchoConsole) serve(nc net.Conn, done chan struct{}) {
	defer func() {
		e.mu.Lock()
		delete(e.clients, nc)
		e.mu.Unlock()
		nc.Close()
	}()
	s := e.srv
	s.trace("echo client %s connected", nc.RemoteAddr())
	fmt.Fprintf(nc, "welcome server %d\r", s.cfg.ServerID)
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 256)
	for {
		n, err := nc.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			for {
				i := strings.IndexAny(string(buf), "\r\n")
				if i < 0 {
					break
				}
				line := strings.TrimSpace(string(buf[:i]))
				buf = buf[i+1:]
				if line == "" {
					continue
				}
				if quit := e.command(nc, line); quit {
					return
				}
			}
		}
		if err != nil {
			return
		}
	}
}

// command 返回 true = 结束该连接（end/quit/fquit）。
func (e *EchoConsole) command(nc net.Conn, line string) bool {
	s := e.srv
	resp := func(info string) {
		fmt.Fprintf(nc, ">%s\r", info)
	}
	// kick 是带参命令（switch 是字符串 case，前缀匹配走 if）。
	if strings.HasPrefix(line, "kick ") {
		// 面板 GM 踢人：名字/账号皆可命中（FindByName 先，FindByAccount
		// 兜底——封号联动按账号踢）。官方语义：只向 entry 下发断开帧，
		// 清理由 entryClientClosed 回报链（store 0x37 logoff + RemovePlayer），
		// 不在此手动移除（对齐 login.go 拒登路径）。
		name := strings.TrimSpace(strings.TrimPrefix(line, "kick "))
		pl := s.players.FindByName(name)
		if pl == nil {
			pl = s.players.FindByAccount(name)
		}
		if pl == nil {
			resp("kick " + name + " not found")
			return false
		}
		s.inner.SendToAllEntry(0, entryBreakClient, entryClientKey(pl))
		resp("kick " + name + " sent")
		return false
	}
	// gm 测试命令族（带参，switch 字符串 case 之外的同款 if 前缀模式）
	if strings.HasPrefix(line, "gm ") || line == "gm" {
		e.gmCommand(nc, strings.Fields(line)[1:])
		return false
	}
	switch line {
	case "quit":
		resp("server will shutdown...")
		go func() {
			// 官方 close()：world.QuitOnClosed = true（Closed 后自动
			// 退出进程）→ world:CloseServer()。
			s.mu.Lock()
			s.cfg.QuitOnClosed = true
			s.mu.Unlock()
			s.CloseServer()
		}()
		return false
	case "fquit":
		resp("server will shutdown...")
		go s.Stop()
		return true
	case "end":
		return true
	case "players":
		resp(fmt.Sprintf("player: %d, online: %d, offline: %d, active: %d, queuing: %d",
			s.players.Count(), s.players.CountOnline(), s.players.CountOffline(),
			s.players.CountActive(), s.players.CountQueue()))
	case "plist":
		// 面板（gmpanel）在线名单方言：计数头 + 每玩家一行 + >end 收尾。
		// 字段 account|name|state|scene|loginAt|activeAt（时间为 unix 秒）。
		list := s.players.Snapshot()
		fmt.Fprintf(nc, ">player %d\r", len(list))
		for _, p := range list {
			fmt.Fprintf(nc, "%s|%s|%s|%d|%d|%d\r",
				p.Account, p.Name, p.State, p.sceneID, p.loginAt.Unix(), p.activeAt.Unix())
		}
		fmt.Fprintf(nc, ">end\r")
	case "open":
		if s.State() == "Opened" {
			resp("game server already opened")
		} else if s.State() != "CanOpen" {
			resp("game server not ready, can't open")
		} else if s.OpenServer() {
			resp("send open command to game server")
		}
	case "close":
		if s.State() != "Opened" {
			resp("game server not open")
		} else if s.CloseServer() {
			resp("send close command to game server")
		}
	case "status":
		resp("game server state is " + s.State())
	case "perf":
		// 官方 solounite echo 独有：world:DumpPerformance("") 导出
		// 逻辑模块性能统计（Go 复刻无 LogicLoader，呈现应答面）。
		resp("dump logic module performance...")
	default:
		// 调试命令族（官方 nx_property 反射）：getp/setp/listm 等
		// 框架保留，返回未定义
		resp("undefined command: " + line)
	}
	return false
}

// Stop 关闭控制台。
func (e *EchoConsole) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return
	}
	e.closed = true
	e.ln.Close()
	for nc, done := range e.clients {
		close(done)
		nc.Close()
	}
}
