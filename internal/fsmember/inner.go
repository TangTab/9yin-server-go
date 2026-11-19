package fsmember

import (
	"net"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/fscomm"
)

// InnerHandler 内网受面回调（对应 GameMember 的四路投递）。
type InnerHandler struct {
	// OnRegister ident 注册完成（0xFFFF 特例帧）。
	OnRegister func(ident uint32)
	// OnDrop 连接断开（ident 已解绑）。
	OnDrop func(ident uint32)
	// OnWorld/OnStore/OnPublic 三队列入口（ident, type, payload）。
	OnWorld  func(ident, typ uint32, payload []byte)
	OnStore  func(ident, typ uint32, payload []byte)
	OnPublic func(ident, typ uint32, payload []byte)
	// OnEntry entry 消息直投（不进队列，官方 PutEntryMessage 语义）。
	OnEntry func(ident, typ uint32, payload []byte)
	// OnError 非法源/短帧等丢弃事件（日志面）。
	OnError func(format string, args ...any)
}

// Inner 复刻 member 的 Adapter 受面（CAdapterHook）：u32 前缀帧之上的
// {src,dst,type} 头 + 0xFFFF 注册 + 源路由（world/public/store/entry）。
type Inner struct {
	adapter *fscomm.Adapter

	mu       sync.Mutex
	identOf  map[int]uint32
	connOf   map[uint32]int
	ownIdent uint32
	h        InnerHandler
}

// NewInner 创建受面；ownIdent 为本 member 的内网标识（0x18+MemberID）。
func NewInner(addr string, port int, ownIdent uint32, h InnerHandler) *Inner {
	in := &Inner{
		identOf:  make(map[int]uint32),
		connOf:   make(map[uint32]int),
		ownIdent: ownIdent,
		h:        h,
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
	return in
}

// Start/Stop/Addr 侦听生命周期。
func (in *Inner) Start() error { return in.adapter.Start() }
func (in *Inner) Stop()        { in.adapter.Stop() }
func (in *Inner) Addr() net.Addr { return in.adapter.Addr() }

// receive 复刻 CAdapterHook::ReceiveMsg（§4.1）：长度门 → 0xFFFF 注册 →
// 源校验 → 四路投递。
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
	switch {
	case hdr.Src == WorldIdent:
		if in.h.OnWorld != nil {
			in.h.OnWorld(hdr.Src, hdr.Type, payload)
		}
	case hdr.Src == PublicIdent:
		if in.h.OnPublic != nil {
			in.h.OnPublic(hdr.Src, hdr.Type, payload)
		}
	case hdr.Src >= StoreIdentBase && hdr.Src < StoreIdentBase+4:
		if in.h.OnStore != nil {
			in.h.OnStore(hdr.Src, hdr.Type, payload)
		}
	default: // entry 8..0x17
		if in.h.OnEntry != nil {
			in.h.OnEntry(hdr.Src, hdr.Type, payload)
		}
	}
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

// send 打包发往已注册 ident（未连接报 "(CInterConnect::SendMsg)not
// connected"，官方同义）。
func (in *Inner) send(dstIdent, typ uint32, payload []byte) error {
	in.mu.Lock()
	idx, ok := in.connOf[dstIdent]
	in.mu.Unlock()
	if !ok {
		return errNotConnected(dstIdent)
	}
	return in.adapter.Send(idx, encodeWire(in.ownIdent, dstIdent, typ, payload))
}

// SendRaw 直发已注册 ident（受面等价发面）。
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
