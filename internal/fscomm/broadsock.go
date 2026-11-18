package fscomm

import (
	"fmt"
	"net"
	"sync"
)

// BroadSockOptions 对齐官方 BroadSock 实体属性：
// BindAddress/BroadcastAddress/BroadcastPort（broadsock.cpp）。
type BroadSockOptions struct {
	BindAddress      string // 本地绑定（官方 BroadSockBindAddress；空=通配）
	BindPort         int    // 本地端口（官方与广播口同用一 socket，收发同口）
	BroadcastAddress string // 广播目的地址（官方 BroadSockBroadcastAddress）
	BroadcastPort    int    // 广播目的端口（官方 BroadSockBroadcastPort）
}

// BroadSockHandler 官方 Receive 回调面。
type BroadSockHandler struct {
	OnReceive func(data []byte, fromAddr string, fromPort int) // 官方 (BroadSock::Receive)
	OnError   func(op string, err error)
}

// BroadSock 复刻官方 BroadSock：UDP 广播口（SO_BROADCAST +
// SO_REUSEADDR + 收发同 socket + 独立线程 select 收包）。
// 载荷语义由宿主定义（官方九阴侧=lister 线路注册 payload）。
type BroadSock struct {
	opts BroadSockOptions
	h    BroadSockHandler

	mu       sync.Mutex
	pc       *net.UDPConn
	wg       sync.WaitGroup
	stopOnce sync.Once
	stopped  chan struct{}
}

// NewBroadSock 创建。
func NewBroadSock(opts BroadSockOptions, h BroadSockHandler) *BroadSock {
	return &BroadSock{opts: opts, h: h, stopped: make(chan struct{})}
}

// Start 绑定并进入收包循环（官方 BroadSock::Start；bind 失败文案
// 对齐 "(BroadSock::Start)bind failed"）。Go 的 UDP 套接字对
// DGRAM 默认开启广播，等价官方 setsockopt(SO_BROADCAST)。
func (b *BroadSock) Start() error {
	bindAddr := net.UDPAddr{Port: b.opts.BindPort}
	if b.opts.BindAddress != "" {
		bindAddr.IP = net.ParseIP(b.opts.BindAddress)
	}
	pc, err := net.ListenUDP("udp4", &bindAddr)
	if err != nil {
		return fmt.Errorf("fscomm: (BroadSock::Start)bind failed: %w", err)
	}
	b.mu.Lock()
	b.pc = pc
	b.mu.Unlock()
	b.wg.Add(1)
	go b.receiveLoop(pc)
	return nil
}

func (b *BroadSock) receiveLoop(pc *net.UDPConn) {
	defer b.wg.Done()
	buf := make([]byte, 64<<10)
	for {
		n, from, err := pc.ReadFromUDP(buf)
		if err != nil {
			return
		}
		if b.h.OnReceive != nil {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			host, port := splitHostPort(from.String())
			b.h.OnReceive(chunk, host, port)
		}
	}
}

// conn 取活动 socket。
func (b *BroadSock) conn() *net.UDPConn {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.pc
}

// Send 向指定目的地发包（官方 BroadSock::Send(addr,port,data,len)）。
func (b *BroadSock) Send(addr string, port int, data []byte) error {
	pc := b.conn()
	if pc == nil {
		return fmt.Errorf("fscomm: (BroadSock::Send)not started")
	}
	if _, err := pc.WriteToUDP(data, &net.UDPAddr{IP: net.ParseIP(addr), Port: port}); err != nil {
		return fmt.Errorf("fscomm: (BroadSock::Send)send data failed: %w", err)
	}
	return nil
}

// SendBroad 向 BroadcastAddress:BroadcastPort 广播
// （官方 BroadSock::SendBroad）。
func (b *BroadSock) SendBroad(data []byte) error {
	return b.Send(b.opts.BroadcastAddress, b.opts.BroadcastPort, data)
}

// Stop 关停（官方 BroadSock::Stop）。
func (b *BroadSock) Stop() {
	b.stopOnce.Do(func() { close(b.stopped) })
	b.mu.Lock()
	pc := b.pc
	b.pc = nil
	b.mu.Unlock()
	if pc != nil {
		pc.Close()
	}
	b.wg.Wait()
}
