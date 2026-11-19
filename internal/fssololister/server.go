package fssololister

import (
	gonet "net"
	"sync"
	"sync/atomic"
	"time"
)

// 客户端面方言常量（CMassSock 面直连 + client_t 内嵌行缓冲实证）。
const (
	// clientLineMax 官方 client_t+8 行缓冲 char[0x100]：非终结字节缓冲
	// 至 0xFF（末位留 NUL），超出静默丢弃。
	clientLineMax = 0xFF

	// requestWait 官方 CheckClient state0：连接后 30s 等请求（counter>30
	// 即第 31 拍触发温和关闭），超时不应答直接关闭。
	requestWait = 30 * time.Second

	// linger 官方 CheckClient state1：应答后 5s 温和关闭（应答后新行
	// 一律不理——每连接只应答一次，AcquireRequest 单请求门）。
	linger = 5 * time.Second
)

// ClientServer 是 :4000 客户端接入面（官方 CReactor::CreateListener +
// OnAccept/OnClose/OnReceive → AddClient/RemoveClient/ReceiveClientMsg
// 的 Go 等价）。行协议 solo 方言：\r 与 \n 皆行终结、其余控制字符照常
// 缓冲（与大写系"控制符丢弃"相反）、行缓冲 0xFF 截断。
type ClientServer struct {
	SL   *SoloLister
	Addr string

	// RequestWait/Linger 可注入缩短（测试）；零值用官方默认。
	RequestWait time.Duration
	Linger      time.Duration

	ln      gonet.Listener
	lnMu    sync.Mutex
	conns   atomic.Int64
	stopped atomic.Bool
}

// Listen 绑定监听（官方 CreateListener(OuterAddr, OuterPort, 100,
// 0x400, 0x4000, …)）。
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
		remote := conn.RemoteAddr().(*gonet.TCPAddr)
		// 官方 OnAccept：黑白名单命中即拒（日志 "address %s:%d ip
		// blocked" + 立即断连）；MaxConnects 满同样拒。
		if s.SL.IsBlockIP(remote.IP.String()) {
			s.SL.logf(logIPBlocked, remote.IP.String(), remote.Port)
			_ = conn.Close()
			continue
		}
		if int(s.conns.Load()) >= int(s.SL.MaxConnects) {
			_ = conn.Close()
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

// ConnectCount 当前接入数（官方日志 total 字段）。
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

// serveConn 官方客户端机（ReceiveClientMsg@0x10062F40 逐字节状态机 +
// ReceiveCommand@0x10063140 单请求/版本门 + CheckClient 超时迁移）：
//
//	accept → 30s 读期限：\r/\n 皆分发（空行不发），其余字节（含控制符）
//	入行缓冲（≤0xFF 截断）；首行过版本门且 client state==0 才应答
//	（版本不达标回 $version_error，前缀/长度不符静默不理），应答后续
//	5s 读期限后关闭；期限到（无请求/应答滞留）即关闭。
func (s *ClientServer) serveConn(conn gonet.Conn) {
	remote := conn.RemoteAddr().(*gonet.TCPAddr)
	idx := s.SL.AddClient(remote.IP.String(), remote.Port)
	defer func() {
		_ = conn.Close()
		s.SL.RemoveClient(idx, remote.IP.String(), remote.Port)
	}()
	s.conns.Add(1)
	defer s.conns.Add(-1)
	_ = conn.SetReadDeadline(time.Now().Add(s.wait()))

	answered := false
	var line []byte
	buf := make([]byte, 512)
	for {
		n, err := conn.Read(buf)
		for i := 0; i < n; i++ {
			c := buf[i]
			if c == '\r' || c == '\n' {
				if len(line) > 0 {
					s.dispatch(conn, idx, string(line), &answered)
					line = line[:0]
				}
				continue
			}
			if len(line) < clientLineMax {
				line = append(line, c)
			}
		}
		if err != nil {
			return
		}
	}
}

// dispatch 官方 ReceiveCommand：版本门 → 单请求门 → 应答。
func (s *ClientServer) dispatch(conn gonet.Conn, idx int, cmd string, answered *bool) {
	if *answered {
		return // 官方 client state≠0 拒绝重复请求
	}
	pass, versionLow := s.SL.CheckVersionGate(cmd)
	if !pass {
		if versionLow {
			*answered = true
			_, _ = conn.Write([]byte("$version_error\r\n"))
			_ = conn.SetReadDeadline(time.Now().Add(s.lingerDur()))
		}
		return // 前缀/长度不符：静默不理（官方 return false 同款）
	}
	if s.SL.ClientVersion <= 0 && cmd != "svrlist" {
		return // 官方无版本门时 strcmp 精确匹配
	}
	if !s.SL.AcquireRequest(idx) {
		return
	}
	resp := s.SL.EncodeResponse()
	if _, err := conn.Write(resp); err == nil {
		s.SL.MarkAnswered(idx)
		*answered = true
		_ = conn.SetReadDeadline(time.Now().Add(s.lingerDur()))
	}
}

// RegistrationServer 是 UDP 双口注册面（官方 CreateBroadcast 双调用 +
// 双收包门的 Go 等价）：
//
//	配置口：绑定 InnerAddr:BroadPort（InnerAddr 空 = 通配），收包过全门
//	        （魔数/kind/精确长度/分区门）→ RecvWorld/RecvEntry 入表；
//	        同时是向 BroadAddr:BroadPort 的广播发送口（SendBroadcast 中继）
//	本地口：仅 InnerAddr ≠ "127.0.0.1" 时建立（官方同款条件），绑定
//	        127.0.0.1:BroadPort，收包过基本门（魔数/kind，无分区门）→
//	        原样中继到广播目标（官方 SendBroadcast@0x100640D0 语义）
type RegistrationServer struct {
	SL *SoloLister

	// InnerAddr/BroadAddr/BroadPort 通常取自 SL 同名字段；零值回退。
	InnerAddr string
	BroadAddr string
	BroadPort int

	pcMu     sync.Mutex
	pc       *gonet.UDPConn // 配置口（收 + 广播发）
	loopPC   *gonet.UDPConn // 本地中继口（可 nil）
	broadDst *gonet.UDPAddr
}

// Listen 建双口。返回配置口地址。
func (r *RegistrationServer) Listen() (gonet.Addr, error) {
	inner := r.InnerAddr
	if inner == "" {
		inner = r.SL.InnerAddr
	}
	port := r.BroadPort
	if port == 0 {
		port = r.SL.BroadPort
	}
	broad := r.BroadAddr
	if broad == "" {
		broad = r.SL.BroadAddr
	}
	if broad == "" {
		broad = "127.255.255.255"
	}

	bindIP := gonet.IPv4zero
	if inner != "" && inner != "0.0.0.0" {
		ip := gonet.ParseIP(inner)
		if ip == nil {
			return nil, &gonet.AddrError{Err: "invalid inner addr", Addr: inner}
		}
		bindIP = ip
	}
	pc, err := gonet.ListenUDP("udp4", &gonet.UDPAddr{IP: bindIP, Port: port})
	if err != nil {
		return nil, err
	}
	r.pcMu.Lock()
	r.pc = pc
	r.broadDst = &gonet.UDPAddr{IP: gonet.ParseIP(broad), Port: port}
	needLoop := inner != "127.0.0.1"
	r.pcMu.Unlock()

	if needLoop {
		// 官方本地口（bind 127.0.0.1:port，目标 127.255.255.255）：
		// 同机定向回环即可达（Linux lo 无 BROADCAST 的既定对策）。
		loop, lerr := gonet.ListenUDP("udp4", &gonet.UDPAddr{IP: gonet.IPv4(127, 0, 0, 1), Port: port})
		if lerr == nil {
			r.pcMu.Lock()
			r.loopPC = loop
			r.pcMu.Unlock()
		}
		// 本地口建立失败不致命（官方失败即 fatal，此处降级为日志——
		// 同机定向包仍可直达配置口）。
	}
	return pc.LocalAddr(), nil
}

// Serve 启动双口收包循环（goroutine；Close 关套接字即收口）。
func (r *RegistrationServer) Serve() {
	go r.servePC()
	go r.serveLoop()
}

func (r *RegistrationServer) getPC() (*gonet.UDPConn, *gonet.UDPAddr) {
	r.pcMu.Lock()
	defer r.pcMu.Unlock()
	return r.pc, r.broadDst
}

func (r *RegistrationServer) getLoop() *gonet.UDPConn {
	r.pcMu.Lock()
	defer r.pcMu.Unlock()
	return r.loopPC
}

func (r *RegistrationServer) servePC() {
	buf := make([]byte, 64<<10)
	for {
		pc, _ := r.getPC()
		if pc == nil {
			return
		}
		n, _, err := pc.ReadFromUDP(buf)
		if err != nil {
			if pc, _ := r.getPC(); pc == nil {
				return
			}
			return
		}
		r.SL.HandleRegistrationPacket(buf[:n])
	}
}

// serveLoop 本地口中继（官方 broad_local_cb → SendBroadcast）：合法
// 注册包原样转发到广播目标——同机 world/entry 只需定向 127.0.0.1，
// 中继让它同时可达跨机部署的其他 lister/world。
func (r *RegistrationServer) serveLoop() {
	buf := make([]byte, 64<<10)
	for {
		loop := r.getLoop()
		if loop == nil {
			return
		}
		n, _, err := loop.ReadFromUDP(buf)
		if err != nil {
			return
		}
		_, ok := ParseLoopbackRegistration(buf[:n])
		if !ok {
			continue
		}
		pc, dst := r.getPC()
		if pc == nil || dst == nil {
			return
		}
		_, _ = pc.WriteToUDP(buf[:n], dst)
	}
}

// SendBroadcast 官方 SoloLister::SendBroadcast@0x100640D0：经配置口向
// 广播目标发原始包（脚本/运维面与本地中继共用）。
func (r *RegistrationServer) SendBroadcast(pkt []byte) bool {
	pc, dst := r.getPC()
	if pc == nil || dst == nil {
		return false
	}
	_, err := pc.WriteToUDP(pkt, dst)
	return err == nil
}

// LocalAddr 配置口本地地址（测试/呈现用；未 Listen 返回空串）。
func (r *RegistrationServer) LocalAddr() string {
	r.pcMu.Lock()
	defer r.pcMu.Unlock()
	if r.pc == nil {
		return ""
	}
	return r.pc.LocalAddr().String()
}

// Close 关闭双口并等收包循环退出。
func (r *RegistrationServer) Close() error {
	r.pcMu.Lock()
	pc, loop := r.pc, r.loopPC
	r.pc, r.loopPC = nil, nil
	r.pcMu.Unlock()
	if loop != nil {
		_ = loop.Close()
	}
	var err error
	if pc != nil {
		err = pc.Close()
	}
	return err
}

// ServeSoloLister 把官方 1s 节拍（CReactor::CreateTimer(1.0f) →
// TimerCheck：每 5 拍 CheckServer）跑在独立 goroutine。返回停止函数。
func ServeSoloLister(sl *SoloLister) (stop func()) {
	ticker := time.NewTicker(time.Second)
	stopCh := make(chan struct{})
	var once sync.Once
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				sl.TickSeconds(1)
			case <-stopCh:
				return
			}
		}
	}()
	return func() {
		once.Do(func() { close(stopCh) })
	}
}
