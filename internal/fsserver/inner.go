package fsserver

import (
	"fmt"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/fscomm"
)

// Inner 复刻世界服的 Adapter 受面（CAdapterHook）：u32 前缀帧之上的
// {src,dst,type} 头 + 0xFFFF 注册 + 源路由四分支（member/entry/store/
// public，§2.4）；注册完成即向 inner 队列投 Connected 事件
// （InnerMsg*Connected 族——member 连接触发三段全量同步）。
type Inner struct {
	adapter *fscomm.Adapter

	mu      sync.Mutex
	identOf map[int]uint32
	connOf  map[uint32]int

	world *World
}

// NewInner 创建受面（world 侦听 16001 语义）。
func NewInner(w *World, addr string, port int) *Inner {
	in := &Inner{
		identOf: make(map[int]uint32),
		connOf:  make(map[uint32]int),
		world:   w,
	}
	in.adapter = fscomm.NewAdapter(fscomm.AdapterOptions{
		Address:    addr,
		Port:       port,
		ConnectMax: 64, // main.lua adapter.ConnectMax
		InBufLen:   1024 * 1024,
		OutBufLen:  1024 * 1024,
	}, fscomm.AdapterHandler{
		OnAccept: func(idx int, addr string) {
			w.logf("World adapter %s accept, totalConn=%d", addr, idx+1)
		},
		OnRead:  func(idx int, data []byte) { in.receive(idx, data) },
		OnClose: func(idx int) { in.dropConn(idx) },
		OnError: func(idx int, op string, err error) {},
	})
	return in
}

// Start/Stop …
func (in *Inner) Start() error { return in.adapter.Start() }

// Stop …
func (in *Inner) Stop() { in.adapter.Stop() }

// receive 一帧 → 头解析 + 源路由（官方 ReceiveMsg 逐分支）。
func (in *Inner) receive(idx int, frame []byte) {
	hdr, payload, err := DecodeInner(frame)
	if err != nil {
		return // "message length error"
	}
	if hdr.Type == IdentRegister {
		in.register(idx, hdr.Src)
		return
	}
	w := in.world
	switch {
	case hdr.Src >= MemberIdentBase && hdr.Src < MemberIdentBase+MaxMemberNum:
		w.PutMemberMessage(hdr.Src, hdr.Type, payload)
	case hdr.Src >= EntryIdentBase && hdr.Src < EntryIdentBase+MaxEntryNum:
		w.PutEntryMessage(hdr.Src, hdr.Type, payload)
	case hdr.Src >= StoreIdentBase && hdr.Src < StoreIdentBase+MaxStoreNum:
		w.PutStoreMessage(hdr.Src, hdr.Type, payload)
	case hdr.Src == PublicIdent:
		w.PutPublicMessage(hdr.Src, hdr.Type, payload)
	default:
		w.logf("(CAdapterHook::ReceiveMsg)message source error (%d)", hdr.Src)
	}
}

// register 0xFFFF 特例帧 → 连接 ↔ ident 绑定 + Connected 事件入队。
func (in *Inner) register(idx int, ident uint32) {
	in.mu.Lock()
	if old, ok := in.identOf[idx]; ok {
		delete(in.connOf, old)
	}
	in.identOf[idx] = ident
	in.connOf[ident] = idx
	in.mu.Unlock()

	w := in.world
	var b [4]byte
	b[0] = byte(idx)
	switch {
	case ident >= MemberIdentBase && ident < MemberIdentBase+MaxMemberNum:
		b[0] = byte(ident - MemberIdentBase)
		w.PutInnerMessage(ident, InnerMsgTypeMemberConnected, b[:])
	case ident >= EntryIdentBase && ident < EntryIdentBase+MaxEntryNum:
		b[0] = byte(ident - EntryIdentBase)
		w.PutInnerMessage(ident, InnerMsgTypeEntryConnected, b[:])
	case ident >= StoreIdentBase && ident < StoreIdentBase+MaxStoreNum:
		b[0] = byte(ident - StoreIdentBase)
		w.PutInnerMessage(ident, InnerMsgTypeStoreConnected, b[:])
	case ident == PublicIdent:
		w.PutInnerMessage(ident, InnerMsgTypeFromExtra, nil)
	}
}

// dropConn 断连解绑 + Disconnect 事件。
func (in *Inner) dropConn(idx int) {
	in.mu.Lock()
	ident, ok := in.identOf[idx]
	if ok {
		delete(in.identOf, idx)
		delete(in.connOf, ident)
	}
	in.mu.Unlock()
	if !ok {
		return
	}
	w := in.world
	switch {
	case ident >= MemberIdentBase && ident < MemberIdentBase+MaxMemberNum:
		w.PutInnerMessage(ident, InnerMsgTypeMemberDisconnect, nil)
	case ident >= EntryIdentBase && ident < EntryIdentBase+MaxEntryNum:
		w.PutInnerMessage(ident, InnerMsgTypeEntryDisconnect, nil)
	case ident >= StoreIdentBase && ident < StoreIdentBase+MaxStoreNum:
		w.PutInnerMessage(ident, InnerMsgTypeStoreDisconnect, nil)
	}
}

// Send 按目标 ident 外发（未连接 = "(CInterConnect::SendMsg)not connected"）。
func (in *Inner) Send(dst, typ uint32, payload []byte) error {
	return in.sendIdent(dst, 0, typ, payload)
}

// sendIdent …（dst 字段透传，SendToScene 用场景 id 作 dst）。
func (in *Inner) sendIdent(ident, dst, typ uint32, payload []byte) error {
	in.mu.Lock()
	idx, ok := in.connOf[ident]
	in.mu.Unlock()
	if !ok {
		return errNotConnected(ident)
	}
	return in.adapter.Send(idx, EncodeInner(InnerHeader{Src: WorldIdent, Dst: dst, Type: typ}, payload))
}

// Connected ident 在册查询（CheckConnect 族）。
func (in *Inner) Connected(ident uint32) bool {
	in.mu.Lock()
	defer in.mu.Unlock()
	_, ok := in.connOf[ident]
	return ok
}

type notConnectedError uint32

func (e notConnectedError) Error() string {
	return fmt.Sprintf("comm_id: %d (CInterConnect::SendMsg)not connected", uint32(e))
}

func errNotConnected(ident uint32) error { return notConnectedError(ident) }
