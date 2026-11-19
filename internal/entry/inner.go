package entry

import (
	"fmt"
	"net"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/fscomm"
)

// Inner 是内网面（CInterConnect + CAdapterHook 复刻）：FsComm Adapter
// 线路（u32 前缀帧）之上的 {src,dst,type} 头 + ident 注册 + world/member
// 路由。world/member 反向连入 entry 的内网端口并注册 ident。
type Inner struct {
	adapter *fscomm.Adapter

	mu       sync.Mutex
	identOf  map[int]uint32 // connIdx → 注册的 src ident
	connOf   map[uint32]int // ident → connIdx
	worldIdx int            // world 连接（-1 未连）
	ownIdent uint32         // entry 自身 ident（发送侧 src）

	// ---- solo 变体旋钮（FsSoloEntry.dll innerhook.cpp；nil/零值=FsEntry
	// 官方语义，不改变网关版行为）----

	// SourceCheck 覆盖默认源校验（solo 注入 {0,1,2}∪[0x18,0x2F]）。
	SourceCheck func(src uint32) bool
	// OnQuitFrame 入站 QuitFrame（0xFFFE）回调（官方 Core 退出）。
	OnQuitFrame func()
	// KickDuplicate ident 已被其它连接持有时：向新连接回 QuitFrame 并
	// 拒绝其注册（官方 "comm_id duplicate"，保旧踢新）。
	KickDuplicate bool
	// MemberSpace 覆盖 member ident 空间（缺省 0x18 起 16 个；solo 为
	// 0x18..0x2F 共 24 个）。零值用官方缺省。
	MemberBase  uint32
	MemberCount int
}

func (in *Inner) memberBase() uint32 {
	if in.MemberBase != 0 {
		return in.MemberBase
	}
	return MemberIdentBase
}

func (in *Inner) memberCount() int {
	if in.MemberCount != 0 {
		return in.MemberCount
	}
	return MaxMemberNum
}

func (in *Inner) sourceOK(src uint32) bool {
	if in.SourceCheck != nil {
		return in.SourceCheck(src)
	}
	return validSource(src)
}

func (in *Inner) memberIdentOf(idx int) uint32 { return in.memberBase() + uint32(idx) }

// InnerHandler 内网事件回调（ProcessMsg 分发后的宿主逻辑）。
type InnerHandler struct {
	// OnRegister ident 注册完成（world=0 / member=0x18+idx）。
	OnRegister func(ident uint32)
	// OnDrop 连接断开（ident 已解绑）。
	OnDrop func(ident uint32)
	// OnMessage 合法源消息：ProcessMsg(this, srcIdent, type, payload)。
	OnMessage func(src uint32, typ uint32, payload []byte)
}

// NewInner 创建内网面；ownIdent 为 entry 自身标识（源地址域）。
func NewInner(addr string, port int, ownIdent uint32, h InnerHandler) *Inner {
	in := &Inner{
		identOf:  make(map[int]uint32),
		connOf:   make(map[uint32]int),
		worldIdx: -1,
		ownIdent: ownIdent,
	}
	in.adapter = fscomm.NewAdapter(fscomm.AdapterOptions{
		Address:    addr,
		Port:       port,
		ConnectMax: 64, // 官方 adapter.ConnectMax
		InBufLen:   1024 * 1024,
		OutBufLen:  1024 * 1024,
	}, fscomm.AdapterHandler{
		OnAccept: func(idx int, addr string) {},
		OnRead: func(idx int, data []byte) {
			in.receive(idx, data, h)
		},
		OnClose: func(idx int) {
			in.dropConn(idx, h)
		},
		OnError: func(idx int, op string, err error) {},
	})
	return in
}

// Start 启动内网侦听。
func (in *Inner) Start() error { return in.adapter.Start() }

// Stop 停止。
func (in *Inner) Stop() { in.adapter.Stop() }

// Addr 实际监听地址。
func (in *Inner) Addr() net.Addr { return in.adapter.Addr() }

// receive 复刻 CAdapterHook::ReceiveMsg：长度门、ident 注册特例、
// 源校验（0 | 0x18..0x27）、合法即投 ProcessMsg。
func (in *Inner) receive(idx int, body []byte, h InnerHandler) {
	hdr, payload, err := decodeInner(body)
	if err != nil {
		return // "message size error"
	}
	if hdr.Type == IdentRegister {
		in.register(idx, hdr.Src, h)
		return
	}
	if hdr.Type == QuitFrame {
		if in.OnQuitFrame != nil {
			in.OnQuitFrame()
		}
		return // 官方：GetCore()->Quit(1)
	}
	if !in.sourceOK(hdr.Src) {
		return // "message source error"
	}
	if h.OnMessage != nil {
		h.OnMessage(hdr.Src, hdr.Type, payload)
	}
}

func (in *Inner) register(idx int, ident uint32, h InnerHandler) {
	in.mu.Lock()
	if in.KickDuplicate {
		if old, ok := in.connOf[ident]; ok && old != idx {
			// 官方 "comm_id duplicate"：向新连接回 QuitFrame 拒绝注册，保旧绑定
			in.mu.Unlock()
			_ = in.adapter.Send(idx, encodeInner(in.ownIdent, WorldIdent, QuitFrame, nil))
			return
		}
	}
	if old, ok := in.identOf[idx]; ok {
		delete(in.connOf, old)
	}
	in.identOf[idx] = ident
	in.connOf[ident] = idx
	if ident == WorldIdent {
		in.worldIdx = idx
	}
	in.mu.Unlock()
	if h.OnRegister != nil {
		h.OnRegister(ident)
	}
}

func (in *Inner) dropConn(idx int, h InnerHandler) {
	in.mu.Lock()
	ident, ok := in.identOf[idx]
	if ok {
		delete(in.identOf, idx)
		delete(in.connOf, ident)
	}
	if in.worldIdx == idx {
		in.worldIdx = -1
	}
	in.mu.Unlock()
	if ok && h.OnDrop != nil {
		h.OnDrop(ident)
	}
}

// send 打包并发往 ident 对应连接（未连接返回错误，对应
// "(CInterConnect::SendMsg)not connected"）。
func (in *Inner) send(dstIdent, typ uint32, payload []byte) error {
	in.mu.Lock()
	idx, ok := in.connOf[dstIdent]
	in.mu.Unlock()
	if !ok {
		return fmt.Errorf("(CInterConnect::SendMsg)not connected: comm_id %d", dstIdent)
	}
	return in.adapter.Send(idx, encodeInner(in.ownIdent, dstIdent, typ, payload))
}

// SendToWorld 发往 world（ident 0）。
func (in *Inner) SendToWorld(typ uint32, payload []byte) error {
	return in.send(WorldIdent, typ, payload)
}

// SendToMember 发往 member idx（ident 0x18+idx）。
func (in *Inner) SendToMember(idx int, typ uint32, payload []byte) error {
	if idx < 0 || idx >= in.memberCount() {
		return fmt.Errorf("member index out of range: %d", idx)
	}
	return in.send(in.memberIdentOf(idx), typ, payload)
}

// SendRaw 直发已注册 ident。
func (in *Inner) SendRaw(ident, typ uint32, payload []byte) error {
	return in.send(ident, typ, payload)
}

// WorldConnected world 是否在网。
func (in *Inner) WorldConnected() bool {
	in.mu.Lock()
	defer in.mu.Unlock()
	return in.worldIdx >= 0
}

// RegisterSelf 对外声明自身 ident（连接端注册帧本地回放，测试用）。
func (in *Inner) RegisterSelf(idx int, ident uint32) { in.register(idx, ident, InnerHandler{}) }

// ConnectedCount 在网连接数。
func (in *Inner) ConnectedCount() int { return in.adapter.ConnectCount() }
