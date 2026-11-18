package fscomm

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	fsnet "github.com/lizhanyin/9yin-server-go/internal/net"
)

// AdapterOptions 对齐官方 Adapter 实体属性表（adapter2.cpp）：
// Address/Port/ConnectMax/InBufLen/OutBufLen/ListenPendings/Closed。
// WorkThreads 不设——Go netpoller 即 IOCP，worker 数由运行时调度。
type AdapterOptions struct {
	Address        string // 监听地址（官方 AdapterAddress）
	Port           int    // 监听端口（官方 AdapterPort）
	ConnectMax     int    // 连接上限（官方 AdapterConnectMax，默认 64）
	InBufLen       int    // 单连接读缓冲上限（官方 AdapterInBufLen，默认 64KiB）
	OutBufLen      int    // 单连接发送 FIFO 字节上限（官方 AdapterOutBufLen，默认 256KiB）
	ListenPendings int    // 积压队列（官方 AdapterListenPendings → net backlog）
	Raw            bool   // true=MassSock 语义：读侧投递裸字节块，成帧留给宿主
}

// AdapterHandler 连接回调。Raw=false 时 OnRead 收到整帧帧体；
// Raw=true 时收到裸字节块（对齐官方 MultiSock/MassSock 读路径）。
// 所有回调在各自连接的读协程派发（官方在 IOCP worker 线程）。
type AdapterHandler struct {
	OnAccept func(idx int, addr string)          // 官方 ProcessAccept 上报
	OnRead   func(idx int, data []byte)          // 官方 hook vtable+0x30
	OnClose  func(idx int)                       // 官方 ClientClose 路径
	OnError  func(idx int, op string, err error) // 对齐 "(Adapter::…)…error" 日志串
}

// Adapter 复刻官方 Adapter/MassSock：内网 TCP 侦听实体。
// 帧模式（默认）读写均走 [u32 LE len][body]；Raw 模式读侧裸投递、
// 写侧仍带前缀（官方 MassSock::WriteOutBuf 同形）。
type Adapter struct {
	opts AdapterOptions
	h    AdapterHandler

	mu      sync.Mutex
	ln      net.Listener
	conns   map[int]*adapterConn
	nextIdx int
	closed  atomic.Bool
	wg      sync.WaitGroup
}

type adapterConn struct {
	idx    int
	c      net.Conn
	out    *msgFIFO
	wake   chan struct{}
	done   chan struct{} // closeOnce 关闭：唤醒并终止写协程
	once   sync.Once
	closed bool
}

// NewAdapter 创建；Start 前可改 Options。
func NewAdapter(opts AdapterOptions, h AdapterHandler) *Adapter {
	if opts.ConnectMax <= 0 {
		opts.ConnectMax = 64
	}
	if opts.InBufLen <= 0 {
		opts.InBufLen = 64 << 10
	}
	if opts.OutBufLen <= 0 {
		opts.OutBufLen = 256 << 10
	}
	if opts.ListenPendings <= 0 {
		opts.ListenPendings = 128
	}
	return &Adapter{opts: opts, h: h, conns: make(map[int]*adapterConn)}
}

// Start 绑定并开始侦听（官方 Adapter::Start；bind/listen 失败文案
// 对齐 "(Adapter::Start)bind failed"/"listen failed"）。
func (a *Adapter) Start() error {
	addr := joinHostPort(a.opts.Address, a.opts.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("fscomm: (Adapter::Start)listen failed: %w", err)
	}
	a.mu.Lock()
	if a.closed.Load() {
		a.mu.Unlock()
		ln.Close()
		return errors.New("fscomm: adapter closed")
	}
	a.ln = ln
	a.mu.Unlock()
	a.wg.Add(1)
	go a.acceptLoop(ln)
	return nil
}

// Addr 实际监听地址（Port=0 时取内核分配端口）。
func (a *Adapter) Addr() net.Addr {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ln == nil {
		return nil
	}
	return a.ln.Addr()
}

func joinHostPort(host string, port int) string {
	return net.JoinHostPort(host, fmt.Sprint(port))
}

func (a *Adapter) acceptLoop(ln net.Listener) {
	defer a.wg.Done()
	for {
		c, err := ln.Accept()
		if err != nil {
			if a.closed.Load() {
				return
			}
			if a.h.OnError != nil {
				a.h.OnError(-1, "accept", err)
			}
			time.Sleep(50 * time.Millisecond) // 对齐官方 PrepareAccept 失败后退避
			continue
		}
		a.setupConn(c)
	}
}

func (a *Adapter) setupConn(c net.Conn) {
	a.mu.Lock()
	if len(a.conns) >= a.opts.ConnectMax {
		a.mu.Unlock()
		c.Close() // 官方 ConnectMax 满即拒
		return
	}
	idx := a.nextIdx
	a.nextIdx++
	ac := &adapterConn{
		idx:  idx,
		c:    c,
		out:  newMsgFIFO(0, a.opts.OutBufLen),
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
	}
	a.conns[idx] = ac
	a.mu.Unlock()

	if a.h.OnAccept != nil {
		a.h.OnAccept(idx, c.RemoteAddr().String())
	}
	a.wg.Add(2)
	go a.readLoop(ac)
	go a.writeLoop(ac)
}

func (a *Adapter) readLoop(ac *adapterConn) {
	defer a.wg.Done()
	if a.opts.Raw {
		a.readRaw(ac)
		return
	}
	fr, err := fsnet.NewPrefixFramer(ac.c, a.maxPayload())
	if err != nil {
		a.closeConn(ac, "init", err)
		return
	}
	for {
		body, err := fr.ReadFrame()
		if err != nil {
			a.closeConn(ac, "read", err)
			return
		}
		if a.h.OnRead != nil {
			a.h.OnRead(ac.idx, body)
		}
	}
}

// readRaw MassSock 读路径：整块裸投递，无成帧。
func (a *Adapter) readRaw(ac *adapterConn) {
	buf := make([]byte, 32<<10)
	for {
		n, err := ac.c.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			if a.h.OnRead != nil {
				a.h.OnRead(ac.idx, chunk)
			}
		}
		if err != nil {
			a.closeConn(ac, "read", err)
			return
		}
	}
}

func (a *Adapter) writeLoop(ac *adapterConn) {
	defer a.wg.Done()
	for {
		msg := ac.out.Pop()
		if msg == nil {
			select {
			case <-ac.wake:
			case <-ac.done:
				return
			}
			continue
		}
		if err := writeFrame(ac.c, msg); err != nil {
			a.closeConn(ac, "write", err)
			return
		}
	}
}

// writeFrame 官方 WriteOutBuf 语义：[u32 len][body] 连续写出。
func writeFrame(w io.Writer, body []byte) error {
	var head [4]byte
	binary.LittleEndian.PutUint32(head[:], uint32(len(body)))
	if _, err := w.Write(head[:]); err != nil {
		return err
	}
	_, err := w.Write(body)
	return err
}

// maxPayload 官方 get_max_send_size = OutBufLen - 4（0x100551a0）。
func (a *Adapter) maxPayload() int { return a.opts.OutBufLen - 4 }

// Send 向指定连接发送一帧（官方 Adapter::ClientSend(idx,msg)）。
// 消息超限/队列满/连接不在册均报错，语义对齐官方三错误串。
func (a *Adapter) Send(idx int, body []byte) error {
	ac := a.conn(idx)
	if ac == nil {
		return fmt.Errorf("fscomm: (Adapter::ClientSend)socket invalid: idx %d", idx)
	}
	if len(body) > a.maxPayload() {
		return fmt.Errorf("fscomm: (Adapter::ClientSend)message size error: %d > %d", len(body), a.maxPayload())
	}
	if err := ac.out.Put(body); err != nil {
		return fmt.Errorf("fscomm: (Adapter::ClientSend)put message error: %w", err)
	}
	select {
	case ac.wake <- struct{}{}:
	default:
	}
	return nil
}

// Broadcast 向全部在册连接广播（官方 Adapter::Broadcast）。
func (a *Adapter) Broadcast(body []byte) error {
	a.mu.Lock()
	idxs := make([]int, 0, len(a.conns))
	for idx := range a.conns {
		idxs = append(idxs, idx)
	}
	a.mu.Unlock()
	var firstErr error
	for _, idx := range idxs {
		if err := a.Send(idx, body); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// ConnectCount 在册连接数（官方 AdapterConnectCount 只读属性）。
func (a *Adapter) ConnectCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.conns)
}

func (a *Adapter) conn(idx int) *adapterConn {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.conns[idx]
}

func (a *Adapter) closeConn(ac *adapterConn, op string, err error) {
	a.mu.Lock()
	if ac.closed {
		a.mu.Unlock()
		return
	}
	ac.closed = true
	delete(a.conns, ac.idx)
	a.mu.Unlock()
	ac.c.Close()
	ac.once.Do(func() { close(ac.done) })
	if a.h.OnError != nil && err != nil && !isConnClosed(err) {
		a.h.OnError(ac.idx, op, err)
	}
	if a.h.OnClose != nil {
		a.h.OnClose(ac.idx)
	}
}

func isConnClosed(err error) bool {
	return err == io.EOF || errors.Is(err, net.ErrClosed)
}

// Close 关闭指定连接（官方 Adapter::Close(idx)）。
func (a *Adapter) Close(idx int) {
	if ac := a.conn(idx); ac != nil {
		a.closeConn(ac, "close", net.ErrClosed)
	}
}

// Stop 关停全部（官方 Adapter::Stop/CloseAll：断侦听+逐连关闭）。
func (a *Adapter) Stop() {
	a.closed.Store(true)
	a.mu.Lock()
	ln := a.ln
	a.ln = nil
	conns := a.conns
	a.conns = make(map[int]*adapterConn)
	a.mu.Unlock()
	if ln != nil {
		ln.Close()
	}
	for _, ac := range conns {
		ac.c.Close()
		ac.once.Do(func() { close(ac.done) })
		select {
		case ac.wake <- struct{}{}:
		default:
		}
	}
	a.wg.Wait()
}
