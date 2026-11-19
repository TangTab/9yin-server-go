package fslister

import (
	gonet "net"
	"sync"
	"sync/atomic"
	"time"
)

// 客户端面方言常量（CMassSockHook 反汇编定案）。
const (
	// clientLineMax 官方行缓冲 0x1000（CreateContext 的 4 KiB）：
	// 可打印字节缓冲到此为止，超出静默丢弃，\n 仍分发截断行。
	clientLineMax = 0x1000

	// requestWait 官方 CheckClient state0→2：连接后 60s 等待请求，
	// 超时不应答直接进入关闭。linger 是 state2→3：应答后 5s 温和关闭
	// （也应答后新行一律忽略——每连接只应答一次）。
	requestWait = 60 * time.Second
	linger     = 5 * time.Second

	// maxClients 官方 client_t[0x2000]（8192 槽）。
	maxClients = 0x2000
)

// ClientServer 是 :4000 客户端接入面（官方 FsComm MassSock 实体 +
// CMassSockHook 的 Go 等价）：明文 TCP 行协议，请求精确 "svrlist"，
// 应答 = GameLister.EncodeResponse（每行尾 CRLF）。
type ClientServer struct {
	GL   *GameLister
	Addr string

	// RequestWait/Linger 可注入缩短（测试）；零值用官方默认。
	RequestWait time.Duration
	Linger     time.Duration

	ln      gonet.Listener
	lnMu    sync.Mutex
	conns   atomic.Int64 // ConnectCount（官方日志 total 字段）
	stopped atomic.Bool
}

// Listen 绑定监听（官方 MassSock:Start + Closed=false）。
func (s *ClientServer) Listen() (gonet.Addr, error) {
	ln, err := gonet.Listen("tcp", s.Addr)
	if err != nil {
		return nil, err
	}
	s.lnMu.Lock()
	s.ln = ln
	s.lnMu.Unlock()
	return ln.Addr(), nil
}

// Serve accept 循环；返回 nil 表示已 Close。
func (s *ClientServer) Serve() error {
	for {
		s.lnMu.Lock()
		ln := s.ln
		s.lnMu.Unlock()
		if ln == nil || s.stopped.Load() {
			return nil
		}
		conn, err := ln.Accept()
		if err != nil {
			if s.stopped.Load() {
				return nil
			}
			return err
		}
		if int(s.conns.Load()) >= maxClients {
			// 官方 AddClient 边界（client 下标 < 0x2000）等价：满则拒接。
			conn.Close()
			continue
		}
		go s.serveConn(conn)
	}
}

// Close 停止监听（已建连接由各自 goroutine 的 deadline 收口）。
func (s *ClientServer) Close() error {
	s.stopped.Store(true)
	s.lnMu.Lock()
	ln := s.ln
	s.ln = nil
	s.lnMu.Unlock()
	if ln != nil {
		return ln.Close()
	}
	return nil
}

// ConnectCount 当前接入数（官方 OnAccept/OnClose 日志的 total）。
func (s *ClientServer) ConnectCount() int { return int(s.conns.Load()) }

func (s *ClientServer) wait() time.Duration {
	if s.RequestWait > 0 {
		return s.RequestWait
	}
	return requestWait
}

func (s *ClientServer) lingerDur() time.Duration {
	if s.Linger > 0 {
		return s.Linger
	}
	return linger
}

// serveConn 官方客户端机（CMassSockHook::OnReceive 逐字节状态机 +
// ReceiveClientMsg 单请求门 + CheckClient 超时迁移）：
//
//	accept → 60s 读期限：可打印字节（≥0x20）入行缓冲（≤0x1000 截断），
//	控制字符（含 \r）丢弃；'\n' 分发——首个且精确 "svrlist" 的行应答
//	并续 5s 读期限，其余行一概不理；期限到（无请求/已应答滞留）即关闭。
func (s *ClientServer) serveConn(conn gonet.Conn) {
	defer conn.Close()
	s.conns.Add(1)
	defer s.conns.Add(-1)
	_ = conn.SetReadDeadline(time.Now().Add(s.wait()))

	line := make([]byte, 0, 64)
	answered := false
	buf := make([]byte, 512)
	for {
		n, err := conn.Read(buf)
		for i := 0; i < n; i++ {
			c := buf[i]
			switch {
			case c == '\n':
				if !answered && len(line) > 0 && string(line) == "svrlist" {
					answered = true
					resp := s.GL.EncodeResponse() // 含 CRLF（官方 MakeSendData）
					_, _ = conn.Write(resp)
					_ = conn.SetReadDeadline(time.Now().Add(s.lingerDur()))
				}
				line = line[:0]
			case c >= 0x20 && len(line) < clientLineMax:
				line = append(line, c)
			}
		}
		if err != nil {
			return
		}
	}
}

// RegistrationServer 是 :15000 注册面（官方 FsComm BroadSock 实体 +
// CBroadSockHook 的 Go 等价）：绑定 0.0.0.0:Port，定向/广播包皆收
// （reactor.CreateBroadcast 同款绑定语义），逐包过 GameLister.HandlePacket。
type RegistrationServer struct {
	GL   *GameLister
	Port int

	pcMu sync.Mutex
	pc   *gonet.UDPConn
	wg   sync.WaitGroup
}

func (r *RegistrationServer) get() *gonet.UDPConn {
	r.pcMu.Lock()
	defer r.pcMu.Unlock()
	return r.pc
}

// Listen 绑定 UDP 口（官方 BroadSock:Start）。
func (r *RegistrationServer) Listen() (gonet.Addr, error) {
	pc, err := gonet.ListenUDP("udp4", &gonet.UDPAddr{IP: gonet.IPv4zero, Port: r.Port})
	if err != nil {
		return nil, err
	}
	r.pcMu.Lock()
	r.pc = pc
	r.pcMu.Unlock()
	return pc.LocalAddr(), nil
}

// Serve 收包循环；返回 nil 表示已 Close。
func (r *RegistrationServer) Serve() error {
	r.wg.Add(1)
	defer r.wg.Done()
	buf := make([]byte, 64<<10)
	for {
		pc := r.get()
		if pc == nil {
			return nil
		}
		n, _, err := pc.ReadFromUDP(buf)
		if err != nil {
			if r.get() == nil {
				return nil
			}
			return err
		}
		r.GL.HandlePacket(buf[:n])
	}
}

// Close 关闭收包口并等 Serve 退出。
func (r *RegistrationServer) Close() error {
	r.pcMu.Lock()
	pc := r.pc
	r.pc = nil
	r.pcMu.Unlock()
	if pc == nil {
		return nil
	}
	err := pc.Close()
	r.wg.Wait()
	return err
}

// ServeGameLister 把官方主循环（DoWait：CTickTimer 每秒一拍、
// 每 5 拍全表扫描）跑在独立 goroutine。返回停止函数（幂等）。
func ServeGameLister(gl *GameLister) (stop func()) {
	ticker := time.NewTicker(time.Second)
	stopCh := make(chan struct{})
	var once sync.Once
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				gl.TickSeconds(1)
			case <-stopCh:
				return
			}
		}
	}()
	return func() {
		once.Do(func() {
			close(stopCh)
		})
	}
}
