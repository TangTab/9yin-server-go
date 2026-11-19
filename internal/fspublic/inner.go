package fspublic

import (
	"net"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/fscomm"
)

// InnerHandler 内网受面回调（对应 GamePublic 的队列/事件入口）。
type InnerHandler struct {
	// OnRegister ident 注册完成（0xFFFF 特例帧）。store#0(4) 注册时
	// 内部已触发 OnStoreConnected（官方 OnConnected→PutMessage(0,1)）。
	OnRegister func(ident uint32)
	// OnDrop 连接断开（ident 解绑）。
	OnDrop func(ident uint32)
	// OnError 非法源/短帧等丢弃事件（日志面）。
	OnError func(format string, args ...any)
}

// Inner 复刻 public 的 Adapter 受面（CAdapterHook）：u32 前缀帧之上的
// {src,dst,type} 头 + 0xFFFF 注册 + 源路由（world/store/member）。
type Inner struct {
	adapter *fscomm.Adapter

	mu       sync.Mutex
	identOf  map[int]uint32
	connOf   map[uint32]int
	h        InnerHandler
	handler  func(ident, typ uint32, payload []byte, src uint32)
}

// NewInner 创建受面（handler 为三队列投递函数，按 src 分道）。
func NewInner(addr string, port int, h InnerHandler, dispatch func(src, ident, typ uint32, payload []byte)) *Inner {
	in := &Inner{
		identOf: make(map[int]uint32),
		connOf:  make(map[uint32]int),
		h:       h,
	}
	in.adapter = fscomm.NewAdapter(fscomm.AdapterOptions{
		Address:    addr,
		Port:       port,
		ConnectMax: 64, // main.lua adapter.ConnectMax
		InBufLen:   1024 * 1024,
		OutBufLen:  1024 * 1024,
	}, fscomm.AdapterHandler{
		OnAccept: func(idx int, addr string) {},
		OnRead:   func(idx int, data []byte) { in.receive(idx, data) },
		OnClose:  func(idx int) { in.dropConn(idx) },
		OnError:  func(idx int, op string, err error) {},
	})
	in.handler = func(ident, typ uint32, payload []byte, src uint32) {
		dispatch(src, ident, typ, payload)
	}
	return in
}

// Start/Stop/Addr 侦听生命周期。
func (in *Inner) Start() error   { return in.adapter.Start() }
func (in *Inner) Stop()          { in.adapter.Stop() }
func (in *Inner) Addr() net.Addr { return in.adapter.Addr() }

// receive 复刻 CAdapterHook::ReceiveMsg（0x1efa0）：长度门 → 0xFFFF 注册
// → 源校验 → 三队列分派。
func (in *Inner) receive(idx int, body []byte) {
	hdr, payload, err := decodeWire(body)
	if err != nil {
		in.warn("%v", err)
		return
	}
	if hdr.Type == IdentRegister {
		in.register(idx, hdr.Src)
		return
	}
	if !validSource(hdr.Src) {
		in.warn("(CAdapterHook::ReceiveMsg)message source error: %#x", hdr.Src)
		return
	}
	in.handler(hdr.Src, hdr.Type, payload, hdr.Src)
}

func (in *Inner) register(idx int, ident uint32) {
	in.mu.Lock()
	if old, ok := in.identOf[idx]; ok {
		delete(in.connOf, old)
	}
	in.identOf[idx] = ident
	in.connOf[ident] = idx
	in.mu.Unlock()
	if in.h.OnRegister != nil {
		in.h.OnRegister(ident)
	}
}

func (in *Inner) dropConn(idx int) {
	in.mu.Lock()
	ident, ok := in.identOf[idx]
	if ok {
		delete(in.identOf, idx)
		delete(in.connOf, ident)
	}
	in.mu.Unlock()
	if ok && in.h.OnDrop != nil {
		in.h.OnDrop(ident)
	}
}

// MemberIdents 在网 member ident 集（SendToAllMember 寻址用）。
func (in *Inner) MemberIdents() []uint32 {
	in.mu.Lock()
	defer in.mu.Unlock()
	out := make([]uint32, 0, 16)
	for id := range in.connOf {
		if id >= MemberIdentBase && id <= MemberIdentMax {
			out = append(out, id)
		}
	}
	return out
}

// StoreIdents 在网 store ident 集。
func (in *Inner) StoreIdents() []uint32 {
	in.mu.Lock()
	defer in.mu.Unlock()
	out := make([]uint32, 0, 4)
	for id := range in.connOf {
		if id >= StoreIdentBase && id < StoreIdentBase+4 {
			out = append(out, id)
		}
	}
	return out
}

// Connected 指定 ident 是否在网。
func (in *Inner) Connected(ident uint32) bool {
	in.mu.Lock()
	defer in.mu.Unlock()
	_, ok := in.connOf[ident]
	return ok
}

// send 打包发往已注册 ident（官方 SendMsg(12B 头)——dst 恒 0）。
func (in *Inner) send(dstIdent, typ uint32, payload []byte) error {
	in.mu.Lock()
	idx, ok := in.connOf[dstIdent]
	in.mu.Unlock()
	if !ok {
		return errNotConnected(dstIdent)
	}
	return in.adapter.Send(idx, encodeWire(PublicIdent, 0, typ, payload))
}

// SendRaw 直发已注册 ident。
func (in *Inner) SendRaw(ident, typ uint32, payload []byte) error {
	return in.send(ident, typ, payload)
}

// ConnectedCount 在网连接数。
func (in *Inner) ConnectedCount() int { return in.adapter.ConnectCount() }

func (in *Inner) warn(format string, args ...any) {
	if in.h.OnError != nil {
		in.h.OnError(format, args...)
	}
}
