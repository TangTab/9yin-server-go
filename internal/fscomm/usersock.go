package fscomm

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// UserSock 状态机：对齐官方 UserSock this+0x38 的
// 0=未连/1=连接中/2=已连/3=关闭 四态。
const (
	userStateNew        int32 = 0
	userStateConnecting int32 = 1
	userStateConnected  int32 = 2
	userStateClosed     int32 = 3
)

// UserSockOptions 对齐官方 UserSock 实体属性：Address/Port/Async/
// OutBufLen；官方另有 InBufLen（读侧成帧由宿主 hook 做，此处读侧
// 直接裸投递，与官方 UserSock::ProcessRead 转发语义一致）。
type UserSockOptions struct {
	Address   string
	Port      int
	Async     bool          // 官方 UserSockAsync：true=FIFO 异步排空，false=同步直发
	OutBufLen int           // 官方 UserSockOutBufLen（默认 64KiB）
	DialTimeout time.Duration
}

// UserSockHandler 官方 hook 回调面：OnOpen/OnClose/OnRead。
type UserSockHandler struct {
	OnOpen  func(addr string)
	OnClose func(err error)
	OnRead  func(data []byte) // 裸字节块（官方 UserSock 读路径不成帧）
	OnError func(op string, err error)
}

// UserSock 复刻官方 UserSock：单路 TCP 客户端。
// 发送侧：Async=false 走同步直发（官方 SendMsg 内联 send 路径），
// Async=true 走有界 FIFO（官方 "send buffer overflow" 语义）。
type UserSock struct {
	opts UserSockOptions
	h    UserSockHandler

	mu    sync.Mutex
	c     net.Conn
	out   *msgFIFO
	wake  chan struct{}
	done  chan struct{}
	once  sync.Once
	state atomic.Int32
	wg    sync.WaitGroup
}

// NewUserSock 创建未连接实例（官方实体创建后设属性再 Connect）。
func NewUserSock(opts UserSockOptions, h UserSockHandler) *UserSock {
	if opts.OutBufLen <= 0 {
		opts.OutBufLen = 64 << 10
	}
	if opts.DialTimeout <= 0 {
		opts.DialTimeout = 5 * time.Second
	}
	return &UserSock{
		opts: opts,
		h:    h,
		out:  newMsgFIFO(0, opts.OutBufLen),
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
	}
}

// Connect 拨号并进入服务循环（官方 UserSock::Connect/Start）。
// 幂等：已连接/连接中返回 nil。
func (u *UserSock) Connect() error {
	if !u.state.CompareAndSwap(userStateNew, userStateConnecting) &&
		!u.state.CompareAndSwap(userStateClosed, userStateConnecting) {
		return nil
	}
	c, err := net.DialTimeout("tcp", joinHostPort(u.opts.Address, u.opts.Port), u.opts.DialTimeout)
	if err != nil {
		u.state.Store(userStateClosed)
		if u.h.OnError != nil {
			u.h.OnError("connect", err)
		}
		return fmt.Errorf("fscomm: (UserSock::InnerConnect)connect error: %w", err)
	}
	u.mu.Lock()
	u.c = c
	u.mu.Unlock()
	u.state.Store(userStateConnected)
	if u.h.OnOpen != nil {
		u.h.OnOpen(c.RemoteAddr().String())
	}
	u.wg.Add(2)
	go u.readLoop(c)
	if u.opts.Async {
		go u.writeLoop(c)
	}
	return nil
}

func (u *UserSock) readLoop(c net.Conn) {
	defer u.wg.Done()
	buf := make([]byte, 32<<10)
	for {
		n, err := c.Read(buf)
		if n > 0 && u.h.OnRead != nil {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			u.h.OnRead(chunk)
		}
		if err != nil {
			u.teardown(c, err)
			return
		}
	}
}

func (u *UserSock) writeLoop(c net.Conn) {
	defer u.wg.Done()
	for {
		msg := u.out.Pop()
		if msg == nil {
			select {
			case <-u.wake:
			case <-u.done:
				return
			}
			continue
		}
		if _, err := c.Write(msg); err != nil {
			return // 收口在读循环 teardown
		}
	}
}

func (u *UserSock) teardown(c net.Conn, err error) {
	u.mu.Lock()
	if u.c != c {
		u.mu.Unlock()
		return
	}
	u.c = nil
	u.mu.Unlock()
	c.Close()
	u.state.Store(userStateClosed)
	u.once.Do(func() { close(u.done) })
	if u.h.OnClose != nil {
		u.h.OnClose(err)
	}
}

// Connected 官方 UserSockConnected 只读属性。
func (u *UserSock) Connected() bool { return u.state.Load() == userStateConnected }

// Send 发送一段字节（官方 UserSock::SendMsg；无成帧——UserSock 载荷
// 语义由宿主定义，内网对拍时宿主两端用同一编码即可）。
func (u *UserSock) Send(data []byte) error {
	if u.state.Load() != userStateConnected {
		return errors.New("fscomm: (UserSock::SendMsg)not connected")
	}
	if len(data) > u.opts.OutBufLen {
		return fmt.Errorf("fscomm: (UserSock::SendMsg)message too long: %d", len(data))
	}
	if u.opts.Async {
		if err := u.out.Put(data); err != nil {
			return fmt.Errorf("fscomm: (UserSock::SendMsg)send buffer overflow: %w", err)
		}
		select {
		case u.wake <- struct{}{}:
		default:
		}
		return nil
	}
	u.mu.Lock()
	c := u.c
	u.mu.Unlock()
	if c == nil {
		return errors.New("fscomm: (UserSock::SendMsg)not connected")
	}
	if _, err := c.Write(data); err != nil {
		u.teardown(c, err)
		return fmt.Errorf("fscomm: (UserSock::InnerSend)send failed: %w", err)
	}
	return nil
}

// Close 断开（官方 UserSock::Close）。
func (u *UserSock) Close() {
	u.mu.Lock()
	c := u.c
	u.mu.Unlock()
	if c != nil {
		u.teardown(c, net.ErrClosed)
	}
}
