package fswatch

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Server 官方 MassSock（Outer 受面）+ EchoSock（维护心跳）+ DoWait tick
// 的 Go 等价合一宿主。
type Server struct {
	cfg   *Config
	watch *GameWatch

	listener net.Listener
	conns    sync.Map // cnx id -> net.Conn
	nextID   atomic.Int64
	closed   atomic.Bool

	echoConn net.Conn
	echoMu   sync.Mutex

	keepAliveStop chan struct{}
	wg            sync.WaitGroup

	logf func(string, ...any)
}

// NewServer 装配。
func NewServer(cfg *Config, watch *GameWatch, logf func(string, ...any)) *Server {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Server{cfg: cfg, watch: watch, logf: logf, keepAliveStop: make(chan struct{})}
}

// Addr 受面地址（测试用）。
func (s *Server) Addr() net.Addr { return s.listener.Addr() }

// Start 官方 main.lua 启动序：mass_sock Start（暂不接受）→ echo Start →
// game_watch Start → 开始接受连接。
func (s *Server) Start() error {
	watch := s.watch
	watch.SetSender(func(cnx int, l *VarList) bool {
		v, ok := s.conns.Load(cnx)
		if !ok {
			return false
		}
		c := v.(net.Conn)
		_, err := c.Write([]byte(l.EncodeText() + "\r\n"))
		return err == nil
	})
	ln, err := net.Listen("tcp", net.JoinHostPort(s.cfg.OuterAddr, strconv.Itoa(s.cfg.OuterPort)))
	if err != nil {
		return fmt.Errorf("fswatch: listen outer: %w", err)
	}
	s.listener = ln
	watch.Start()
	s.dialEcho()
	s.wg.Add(2)
	go s.acceptLoop()
	go s.keepAliveLoop()
	s.logf("Outer socket %s:%d", s.cfg.OuterAddr, s.cfg.OuterPort)
	s.logf("Echo socket %s:%d", s.cfg.EchoAddr, s.cfg.EchoPort)
	s.logf("Game watch server started")
	return nil
}

// dialEcho 官方 EchoSock：连维护口失败不阻断主服务（AppManager 缺位时
// 仅日志）。
func (s *Server) dialEcho() {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(s.cfg.EchoAddr, strconv.Itoa(s.cfg.EchoPort)), 2*time.Second)
	if err != nil {
		s.logf("echo socket not connected: %v", err)
		return
	}
	s.echoMu.Lock()
	s.echoConn = conn
	s.echoMu.Unlock()
	s.logf("Echo client %s connected", conn.RemoteAddr())
}

// acceptLoop 官方 MassSock 受面：MaxClient 上限 + IP 白名单
// （CMassSockHook::SetAllowAddress → FsComm 层 OnBlockIP 面）。
func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				return
			}
			continue
		}
		if s.cfg.MaxClient > 0 && int(s.nextID.Load()) > s.cfg.MaxClient*4 {
			// 惰性清点：活跃连接超上限即拒绝（官方 ConnectMax=16）。
			n := 0
			s.conns.Range(func(_, _ any) bool { n++; return true })
			if n >= s.cfg.MaxClient {
				conn.Close()
				continue
			}
		}
		if len(s.cfg.ClientAddresses) > 0 && !s.allowRemote(conn.RemoteAddr()) {
			s.logf("Address %s blocked", conn.RemoteAddr())
			conn.Close()
			continue
		}
		id := int(s.nextID.Add(1))
		s.conns.Store(id, conn)
		s.logf("Address %s connected, total %d", conn.RemoteAddr(), s.count())
		s.wg.Add(1)
		go s.serveConn(id, conn)
	}
}

func (s *Server) allowRemote(addr net.Addr) bool {
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		host = addr.String()
	}
	for _, a := range s.cfg.ClientAddresses {
		if strings.EqualFold(a, host) {
			return true
		}
	}
	return false
}

func (s *Server) count() int {
	n := 0
	s.conns.Range(func(_, _ any) bool { n++; return true })
	return n
}

// serveConn 官方 OnReceive → CMassSockHook::ReceiveMsg（\r\n 切帧）。
func (s *Server) serveConn(id int, conn net.Conn) {
	defer s.wg.Done()
	defer func() {
		s.conns.Delete(id)
		conn.Close()
	}()
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		if s.closed.Load() {
			return
		}
		line := strings.TrimRight(sc.Text(), "\r")
		s.watch.ProcessLine(id, line)
	}
}

// keepAliveLoop 官方 DoWait tick（1000ms 计数 >5）：①向全体运维客户端广播
// "\r\n"；②EchoSock SendLine `<info state="%s" connect="%d" />`。
func (s *Server) keepAliveLoop() {
	defer s.wg.Done()
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.keepAliveStop:
			return
		case <-t.C:
			s.conns.Range(func(_, v any) bool {
				c := v.(net.Conn)
				_, _ = c.Write([]byte("\r\n"))
				return true
			})
			s.echoMu.Lock()
			conn := s.echoConn
			s.echoMu.Unlock()
			if conn != nil {
				fmt.Fprintf(conn, "<info state=\"%s\" connect=\"%d\" />\r\n", s.watch.State(), s.count())
			}
		}
	}
}

// Stop 官方 main.lua close 序：客户端关闭 → echo 停 → game_watch 停。
func (s *Server) Stop() {
	if !s.closed.CompareAndSwap(false, true) {
		return
	}
	close(s.keepAliveStop)
	if s.listener != nil {
		s.listener.Close()
	}
	s.conns.Range(func(_, v any) bool {
		v.(net.Conn).Close()
		return true
	})
	s.echoMu.Lock()
	if s.echoConn != nil {
		s.echoConn.Close()
	}
	s.echoMu.Unlock()
	s.wg.Wait()
	s.watch.Stop()
}
