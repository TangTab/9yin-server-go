package fscomm

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	fsnet "github.com/lizhanyin/9yin-server-go/internal/net"
)

// MultiSockOptions 对齐官方 MultiSock 实体属性：Address/Port/
// ConnectMax(=并发连接数)/AutoRetry/RetryDelay/InBufLen/OutBufLen。
type MultiSockOptions struct {
	Address    string        // 官方 MultiSockAddress
	Port       int           // 官方 MultiSockPort
	AutoRetry  bool          // 官方 MultiSockAutoRetry（断线自动重连）
	RetryDelay time.Duration // 官方 MultiSockRetryDelay（默认 3s）
	OutBufLen  int           // 官方 MultiSockOutBufLen（默认 256KiB）
}

// MultiSockHandler 官方四回调：ClientBegin/ClientConnected/
// ClientConnectFail/ClientClose（multisock.cpp 回调注册面）。
type MultiSockHandler struct {
	OnBegin        func()          // ClientBegin：发起拨号前
	OnConnected    func()          // ClientConnected：链路建立
	OnConnectFail  func(err error) // ClientConnectFail：拨号失败
	OnClose        func(err error) // ClientClose：链路断开
	OnRead         func(data []byte) // 读到整帧（Go 侧成帧，见 doc.go）
}

// MultiSock 复刻官方 MultiSock：带自动重连的 TCP 客户端。
// 官方支持 ConnectMax 路并发；内网对端场景每路对同一 (addr,port)，
// Go 以单路精简（ConnectMax>1 由上层建多个 MultiSock 表达）。
type MultiSock struct {
	opts MultiSockOptions
	h    MultiSockHandler

	mu     sync.Mutex
	c      net.Conn
	out    *msgFIFO
	wake   chan struct{}
	done   chan struct{}
	once   sync.Once
	stopped bool
	wg     sync.WaitGroup
}

// NewMultiSock 创建；随后 Start 开始首拨与重连循环。
func NewMultiSock(opts MultiSockOptions, h MultiSockHandler) *MultiSock {
	if opts.RetryDelay <= 0 {
		opts.RetryDelay = 3 * time.Second
	}
	if opts.OutBufLen <= 0 {
		opts.OutBufLen = 256 << 10
	}
	return &MultiSock{
		opts: opts,
		h:    h,
		out:  newMsgFIFO(0, opts.OutBufLen),
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
	}
}

// Start 启动连接循环（官方 MultiSock::Start）。
func (m *MultiSock) Start() {
	m.wg.Add(1)
	go m.loop()
}

// loop 官方 CheckConnect+AutoRetry 语义：拨号→读写→断开→
// （AutoRetry 时）退避 RetryDelay 再拨。
func (m *MultiSock) loop() {
	defer m.wg.Done()
	for {
		select {
		case <-m.done:
			return
		default:
		}
		if m.h.OnBegin != nil {
			m.h.OnBegin()
		}
		c, err := net.DialTimeout("tcp", joinHostPort(m.opts.Address, m.opts.Port), 5*time.Second)
		if err != nil {
			if m.h.OnConnectFail != nil {
				m.h.OnConnectFail(err)
			}
			if !m.opts.AutoRetry {
				return
			}
			if !m.sleepRetry() {
				return
			}
			continue
		}
		if m.h.OnConnected != nil {
			m.h.OnConnected()
		}
		m.setConn(c)
		readErr := m.serve(c)
		m.clearConn(c)
		if m.h.OnClose != nil {
			m.h.OnClose(readErr)
		}
		select {
		case <-m.done:
			return
		default:
		}
		if !m.opts.AutoRetry {
			return
		}
		if !m.sleepRetry() {
			return
		}
	}
}

func (m *MultiSock) sleepRetry() bool {
	t := time.NewTimer(m.opts.RetryDelay)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-m.done:
		return false
	}
}

func (m *MultiSock) setConn(c net.Conn) {
	m.mu.Lock()
	m.c = c
	m.mu.Unlock()
	m.wg.Add(1)
	go m.writeLoop(c)
}

func (m *MultiSock) clearConn(c net.Conn) {
	m.mu.Lock()
	if m.c == c {
		m.c = nil
	}
	m.mu.Unlock()
	c.Close()
}

// serve 读循环：成帧后回调 OnRead；返回读错误。
func (m *MultiSock) serve(c net.Conn) error {
	fr, err := fsnet.NewPrefixFramer(c, m.opts.OutBufLen-4)
	if err != nil {
		return err
	}
	for {
		body, err := fr.ReadFrame()
		if err != nil {
			return err
		}
		if m.h.OnRead != nil {
			m.h.OnRead(body)
		}
	}
}

func (m *MultiSock) writeLoop(c net.Conn) {
	defer m.wg.Done()
	for {
		msg := m.out.Pop()
		if msg == nil {
			select {
			case <-m.wake:
			case <-m.done:
				return
			}
			continue
		}
		if err := writeFrame(c, msg); err != nil {
			return // 连接已由读循环统一收口
		}
	}
}

// Connected 官方 MultiSockConnected 只读属性。
func (m *MultiSock) Connected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.c != nil
}

// Send 发一帧（官方 MultiSock::ClientSend）。未连接时入队等待
// 重连后写出（官方 FIFO 语义）；队列满/超限报错。
func (m *MultiSock) Send(body []byte) error {
	if len(body) > m.opts.OutBufLen-4 {
		return fmt.Errorf("fscomm: (MultiSock::ClientSend)message size error: %d", len(body))
	}
	m.mu.Lock()
	stopped := m.stopped
	m.mu.Unlock()
	if stopped {
		return errors.New("fscomm: multisock stopped")
	}
	if err := m.out.Put(body); err != nil {
		return fmt.Errorf("fscomm: (MultiSock::ClientSend)message size error: %w", err)
	}
	select {
	case m.wake <- struct{}{}:
	default:
	}
	return nil
}

// Stop 关停（官方 MultiSock::Stop）。
func (m *MultiSock) Stop() {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return
	}
	m.stopped = true
	c := m.c
	m.c = nil
	m.mu.Unlock()
	if c != nil {
		c.Close()
	}
	m.once.Do(func() { close(m.done) })
	m.wg.Wait()
}
