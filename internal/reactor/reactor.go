// Package reactor 对齐官方 Fs 框架 CReactor（FsSoloLister.dll 等的
// IOCP 网络引擎，BGserverCJ map 符号直读实证）的 Go 复刻。
//
// 复刻的是架构语义而非 syscall：Go 运行时在 Windows 下的 netpoller
// 本身就是 IOCP（CreateIoCompletionPort + GetQueuedCompletionStatus），
// 逐字移植完成端口等于重写运行时。对齐点：
//
//	官方符号（CReactor::）			Go 方法
//	---------------------------------------------------------------
//	CreateListener/AcceptEx			CreateListener
//	CreateConnector/ConnectEx		CreateConnector（异步拨号）
//	CreateBroadcast（UDP 广播）		CreateBroadcast（SO_BROADCAST
//					Go 对 DGRAM 套接字默认开启）
//	Send / Send2 / Send3			Send（异步入队，写侧独立排空）
//	EventLoop（单线程派发）			内部事件循环：所有回调串行，
//					官方单线程语义——业务侧免锁
//	ProcessRead/Write/Connect/Broad		读/写/拨号/广播路径的连接
//	的 identity error 校验			身份校验（id 单调不复用，
//					陈旧/伪造 id 一律拒绝）
//	ExpandOutBuffer 环形缓冲		每连接有界发送队列（上限
//					MaxOutBufferBytes，满则拒发并记
//					日志——官方无限扩容，此处护内存）
//	ShutdownConnector / CloseConnect	同名方法
//	DumpReactor				Stats（连接清单）
package reactor

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/sockopt"
)

// ConnHandler 连接三回调（官方 CreateListener/CreateConnector 的
// on_open/on_close/on_input 签名）。回调全部在事件循环串行派发。
type ConnHandler struct {
	OnOpen  func(connID int, addr string, port int)
	OnClose func(connID int, addr string, port int)
	OnInput func(connID int, data []byte)
}

// BroadHandler 广播接收回调（官方 CreateBroadcast 的 recv 回调）。
type BroadHandler func(data []byte, fromAddr string, fromPort int)

// MaxOutBufferBytes 每连接发送队列字节上限（护内存；官方为无限扩容）。
const MaxOutBufferBytes = 4 << 20

const (
	stateConnecting int32 = 0
	stateConnected  int32 = 1
	stateClosed     int32 = 2
)

// Reactor 是三通道网络引擎：TCP 监听（AcceptEx 语义）、TCP 主动外连
// （ConnectEx 语义）、UDP 广播（Broadcast 语义），统一身份校验与
// 单线程事件派发。
type Reactor struct {
	Logf func(format string, args ...any)

	events chan func()
	done   chan struct{}
	loopWG sync.WaitGroup
	objWG  sync.WaitGroup
	start  atomic.Bool

	mu        sync.Mutex
	closed    bool
	nextID    int
	conns     map[int]*rconn
	listeners map[int]net.Listener
	bcasts    map[int]*rbcast
}

// rconn 一条 TCP 连接（listener 接入或 connector 外连同表管理）。
type rconn struct {
	id    int
	state atomic.Int32
	c     net.Conn
	h     ConnHandler

	outMu     sync.Mutex // 保护 out 生命周期（入队与关闭互斥）
	outClosed bool
	out       chan []byte
	outLen    atomic.Int64

	closeOnce sync.Once
}

// rbcast 一个 UDP 广播口（收发同套接字，绑通配地址）。
type rbcast struct {
	id   int
	addr string
	port int
	pc   *net.UDPConn
	h    BroadHandler
}

// New 创建 Reactor（Logf 可空，日志串对齐官方 "(CReactor::…)" 前缀）。
func New() *Reactor {
	return &Reactor{
		events:    make(chan func(), 4096),
		done:      make(chan struct{}),
		conns:     make(map[int]*rconn),
		listeners: make(map[int]net.Listener),
		bcasts:    make(map[int]*rbcast),
	}
}

// Start 启动事件循环（幂等；官方 Start 语义）。
func (r *Reactor) Start() error {
	if !r.start.CompareAndSwap(false, true) {
		return nil
	}
	r.loopWG.Add(1)
	go r.eventLoop()
	return nil
}

// Stop 关闭全部对象并退出事件循环（幂等；在途回调排空后才返回）。
func (r *Reactor) Stop() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	listeners := make([]net.Listener, 0, len(r.listeners))
	for _, ln := range r.listeners {
		listeners = append(listeners, ln)
	}
	conns := make([]*rconn, 0, len(r.conns))
	for _, c := range r.conns {
		conns = append(conns, c)
	}
	bcs := make([]*rbcast, 0, len(r.bcasts))
	for _, b := range r.bcasts {
		bcs = append(bcs, b)
	}
	r.mu.Unlock()

	for _, ln := range listeners {
		ln.Close()
	}
	for _, b := range bcs {
		b.pc.Close()
	}
	for _, c := range conns {
		c.shutdown(r)
	}
	r.objWG.Wait()
	close(r.done)
	r.loopWG.Wait()
}

// eventLoop 官方 EventLoop 单线程派发语义：所有回调按投递序串行执行。
func (r *Reactor) eventLoop() {
	defer r.loopWG.Done()
	for {
		select {
		case fn := <-r.events:
			fn()
		case <-r.done:
			for {
				select {
				case fn := <-r.events:
					fn()
				default:
					return
				}
			}
		}
	}
}

// post 把回调投递进事件循环；缓冲满则另起协程兜底（回调绝不丢弃）。
func (r *Reactor) post(fn func()) {
	select {
	case r.events <- fn:
	default:
		go func() {
			select {
			case r.events <- fn:
			case <-r.done:
			}
		}()
	}
}

func (r *Reactor) logf(format string, args ...any) {
	if r.Logf != nil {
		r.Logf("(Reactor::"+format+")", args...)
	}
}

// newID 对象 id 单调递增、永不复用：任何陈旧/伪造 id 都查无此物，
// 即官方 ProcessRead/Write/Connect/Broad 各处 "identity error" 的
// 拒绝效果（且无槽位复用窗口）。
func (r *Reactor) newID() int {
	r.nextID++
	return r.nextID
}

func hostPort(addr net.Addr) (string, int) {
	host, portS, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String(), 0
	}
	port, _ := strconv.Atoi(portS)
	return host, port
}

// CreateListener 在 addr:port 开 TCP 监听（AcceptEx 语义）：每个接入
// 记为一条连接，OnOpen/OnInput/OnClose 串行派发。maxAccepts 为官方
// 签名兼容保留（Go accept 循环无预投递概念），可为 0。
func (r *Reactor) CreateListener(addr string, port int, maxAccepts int, h ConnHandler) (int, error) {
	if !r.start.Load() {
		return 0, fmt.Errorf("reactor 未 Start")
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(addr, strconv.Itoa(port)))
	if err != nil {
		r.logf("CreateListener bind error: %v", err)
		return 0, err
	}
	id := r.newID()
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		ln.Close()
		return 0, fmt.Errorf("reactor 已停止")
	}
	r.listeners[id] = ln
	r.mu.Unlock()

	r.objWG.Add(1)
	go r.acceptLoop(ln, h)
	return id, nil
}

// ListenerPort 返回 listener 实际侦听口（官方 CReactor::GetListenerPort；
// 0 口侦听时回填临时口）。id 无效返回 false。
func (r *Reactor) ListenerPort(id int) (int, bool) {
	r.mu.Lock()
	ln, ok := r.listeners[id]
	r.mu.Unlock()
	if !ok {
		return 0, false
	}
	if ta, ok := ln.Addr().(*net.TCPAddr); ok {
		return ta.Port, true
	}
	return 0, false
}

func (r *Reactor) acceptLoop(ln net.Listener, h ConnHandler) {
	defer r.objWG.Done()
	for {
		nc, err := ln.Accept()
		if err != nil {
			r.logf("EventLoop listener error: %v", err)
			return
		}
		r.registerConn(nc, h)
	}
}

// CreateConnector 异步外连（ConnectEx 语义）：立即返回连接 id，拨号
// 成功后 OnOpen；失败仅记日志（官方 "(CReactor::CreateConnector)
// ConnectEx error" 同款），id 停留 connecting 态，Send 返回 false。
func (r *Reactor) CreateConnector(addr string, port int, h ConnHandler) (int, error) {
	if !r.start.Load() {
		return 0, fmt.Errorf("reactor 未 Start")
	}
	c := &rconn{id: r.newID(), out: make(chan []byte, 256), h: h}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return 0, fmt.Errorf("reactor 已停止")
	}
	r.conns[c.id] = c
	r.mu.Unlock()

	r.objWG.Add(1)
	go func() {
		defer r.objWG.Done()
		nc, err := net.DialTimeout("tcp", net.JoinHostPort(addr, strconv.Itoa(port)), 15*time.Second)
		if err != nil {
			r.logf("CreateConnector ConnectEx error: %v", err)
			c.state.Store(stateClosed)
			r.mu.Lock()
			delete(r.conns, c.id)
			r.mu.Unlock()
			return
		}
		r.wireConn(c, nc)
	}()
	return c.id, nil
}

// registerConn 接入侧连接直接落表（accept 已成，直接进入 connected）。
func (r *Reactor) registerConn(nc net.Conn, h ConnHandler) {
	c := &rconn{id: r.newID(), out: make(chan []byte, 256), h: h}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		nc.Close()
		return
	}
	r.conns[c.id] = c
	r.mu.Unlock()
	r.wireConn(c, nc)
}

// wireConn 挂读写协程并派发 OnOpen。
func (r *Reactor) wireConn(c *rconn, nc net.Conn) {
	c.c = nc
	c.state.Store(stateConnected)
	host, port := hostPort(nc.RemoteAddr())

	r.objWG.Add(2)
	go c.readLoop(r, host, port)
	go c.writeLoop(r)

	r.post(func() {
		if c.h.OnOpen != nil {
			c.h.OnOpen(c.id, host, port)
		}
	})
}

func (c *rconn) readLoop(r *Reactor, host string, port int) {
	defer r.objWG.Done()
	buf := make([]byte, 64<<10)
	for {
		n, err := c.c.Read(buf)
		if n > 0 {
			data := append([]byte(nil), buf[:n]...)
			r.post(func() {
				if c.h.OnInput != nil {
					c.h.OnInput(c.id, data)
				}
			})
		}
		if err != nil {
			c.shutdown(r)
			return
		}
	}
}

func (c *rconn) writeLoop(r *Reactor) {
	defer r.objWG.Done()
	for b := range c.out {
		if _, err := c.c.Write(b); err != nil {
			r.logf("ProcessWrite error: %v", err)
			c.shutdown(r)
			return
		}
		c.outLen.Add(-int64(len(b)))
	}
}

// Send 异步发送（官方 Send/PostQueuedCompletionStatus 语义）：入队即返。
// id 不存在/未连/已关/队列超限 → false（identity error 同效拒发）。
func (r *Reactor) Send(connID int, data []byte) bool {
	c := r.lookup(connID)
	if c == nil || c.state.Load() != stateConnected {
		r.logf("Send connect not exists: %d", connID)
		return false
	}
	c.outMu.Lock()
	defer c.outMu.Unlock()
	if c.outClosed {
		r.logf("Send connect not exists: %d", connID)
		return false
	}
	if c.outLen.Load()+int64(len(data)) > MaxOutBufferBytes {
		r.logf("Send buffer overflow %d bytes: %d", c.outLen.Load(), connID)
		return false
	}
	select {
	case c.out <- append([]byte(nil), data...):
		c.outLen.Add(int64(len(data)))
		return true
	default:
		r.logf("Send queue full: %d", connID)
		return false
	}
}

// ShutdownConnector 关闭连接（官方 ShutdownConnector 语义）：
// OnClose 恰好一次。
func (r *Reactor) ShutdownConnector(connID int) bool {
	c := r.lookup(connID)
	if c == nil {
		r.logf("ShutdownConnector connect not exists: %d", connID)
		return false
	}
	c.shutdown(r)
	return true
}

// CloseConnect 立即关闭连接（OnClose 恰好一次）。
func (r *Reactor) CloseConnect(connID int) bool {
	return r.ShutdownConnector(connID)
}

// GetConnected 官方 GetConnected 语义。
func (r *Reactor) GetConnected(connID int) bool {
	c := r.lookup(connID)
	return c != nil && c.state.Load() == stateConnected
}

// CreateBroadcast 开 UDP 广播口（官方 CreateBroadcast 语义）：绑通配
// 地址收发一体，既收定向包也收广播包（Go 对 DGRAM 默认置
// SO_BROADCAST；官方双口=环回+网段广播，Go 单口即可双收）。
// SO_REUSEADDR 共绑：官方集群多进程/合一进程内多实体同绑一口互收
// 广播（Win11 实测 127.255.255.255 定向广播全量投递、发送者亦自收）。
func (r *Reactor) CreateBroadcast(broadAddr string, broadPort int, onRecv BroadHandler) (int, error) {
	if !r.start.Load() {
		return 0, fmt.Errorf("reactor 未 Start")
	}
	pc, err := sockopt.ListenUDPReusable("udp4", (&net.UDPAddr{IP: net.IPv4zero, Port: broadPort}).String())
	if err != nil {
		r.logf("CreateBroadcast bind failed: %v", err)
		return 0, err
	}
	b := &rbcast{id: r.newID(), addr: broadAddr, port: broadPort, pc: pc, h: onRecv}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		pc.Close()
		return 0, fmt.Errorf("reactor 已停止")
	}
	r.bcasts[b.id] = b
	r.mu.Unlock()

	r.objWG.Add(1)
	go b.recvLoop(r)
	return b.id, nil
}

func (b *rbcast) recvLoop(r *Reactor) {
	defer r.objWG.Done()
	buf := make([]byte, 64<<10)
	for {
		n, from, err := b.pc.ReadFromUDP(buf)
		if err != nil {
			return
		}
		data := append([]byte(nil), buf[:n]...)
		fhost, fport := hostPort(from)
		r.post(func() {
			if b.h != nil {
				b.h(data, fhost, fport)
			}
		})
	}
}

// SendBroadcast 官方 SendBroadcast 语义：向该口的广播地址发送。
func (r *Reactor) SendBroadcast(broadID int, data []byte) bool {
	r.mu.Lock()
	b := r.bcasts[broadID]
	r.mu.Unlock()
	if b == nil {
		r.logf("SendBroadcast connect not exists: %d", broadID)
		return false
	}
	if _, err := b.pc.WriteToUDP(data, &net.UDPAddr{IP: net.ParseIP(b.addr), Port: b.port}); err != nil {
		r.logf("SendBroadcast send data failed: %v", err)
		return false
	}
	return true
}

// Stats 官方 DumpReactor 语义：对象与连接清单。
func (r *Reactor) Stats() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := fmt.Sprintf("conns=%d listeners=%d broadcasts=%d\n",
		len(r.conns), len(r.listeners), len(r.bcasts))
	for id, c := range r.conns {
		remote := "?"
		if c.c != nil {
			remote = c.c.RemoteAddr().String()
		}
		out += fmt.Sprintf("  conn %d state=%d %s out=%dB\n", id, c.state.Load(), remote, c.outLen.Load())
	}
	return out
}

// lookup 连接查询（identity 校验拒点：查无此物即陈旧/伪造 id）。
func (r *Reactor) lookup(connID int) *rconn {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.conns[connID]
}

// shutdown 关闭连接并恰好一次派发 OnClose（读退出/写失败/主动关共用）。
func (c *rconn) shutdown(r *Reactor) {
	c.closeOnce.Do(func() {
		c.state.Store(stateClosed)
		if c.c != nil {
			c.c.Close()
		}
		c.outMu.Lock()
		c.outClosed = true
		close(c.out)
		c.outMu.Unlock()

		h, p := "", 0
		if c.c != nil {
			h, p = hostPort(c.c.RemoteAddr())
		}
		r.post(func() {
			r.mu.Lock()
			if r.conns[c.id] == c {
				delete(r.conns, c.id)
			}
			r.mu.Unlock()
			if c.h.OnClose != nil {
				c.h.OnClose(c.id, h, p)
			}
		})
	})
}
