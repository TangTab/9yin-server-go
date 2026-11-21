package fssoloauction

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// InnerEvents 是 InnerLink 上抛给宿主的事件（官方 CInnerHook::SetCallback
// 四回调 + store identify 上抛 InnerStoreConnected @0x100AEBF0）。
type InnerEvents struct {
	// OnMessage 是业务消息（identify/exit 之外的帧；payload 为拷贝）。
	OnMessage func(comm, id1, id2 uint32, payload []byte)
	// OnStoreIdentified 在 comm 4..7 完成 identify 时触发
	//（官方：状态机事件 id1=1 → 0x3F 注册）。
	OnStoreIdentified func()
	// OnExit 是 id2=0xFFFE 退出请求（官方调 ICore 关停；Go 版由宿主决定）。
	OnExit func()
}

// InnerLink 复刻 CInnerHook（innerhook.cpp）：内网 TCP 受/发面。
//
//	受面：CreateListener(InnerAddr, InnerPort, 10, 0x400000/0x400000/
//	      0x4000000)（Start @100AAAFA 起），对端连入后发 identify 自报 comm；
//	发面：CheckConnect(comm, addr, port)（广播发现驱动的外连，已连去重），
//	      外连 OnConnect 立即回发 identify {12, commSelf, 0, 0xFFFF}
//	     （OnConnect @0x1006CE40 栈帧实证）；
//	路由：m_ConnectorByComm[comm]→conn（SendMsg @0x1006DE80：comm_id∈
//	      [0,64) 断言，无连接 trace "connect not exists"）。
type InnerLink struct {
	r        *reactor.Reactor
	ev       InnerEvents
	appName  string
	logf     func(format string, args ...any)
	selfComm uint32

	mu       sync.Mutex
	listener int
	connComm map[int]uint32 // connID → 对端 comm（identify 后；0xFFFFFFFF=未知）
	commConn map[uint32]int // comm → connID
	pendComm map[int]uint32 // 外连 connID → 预期 comm（identify 确认前）
	acc      map[int]*frameAcc
	dialing  map[uint32]string // comm → "addr:port"（拨号中，防重复外连）
}

const commUnknown = 0xFFFFFFFF

// NewInnerLink 建链路（selfComm 官方=2）。
func NewInnerLink(r *reactor.Reactor, ev InnerEvents, appName string, selfComm uint32,
	logf func(format string, args ...any)) *InnerLink {
	return &InnerLink{
		r: r, ev: ev, appName: appName, logf: logf, selfComm: selfComm,
		listener: -1,
		connComm: map[int]uint32{},
		commConn: map[uint32]int{},
		pendComm: map[int]uint32{},
		acc:      map[int]*frameAcc{},
		dialing:  map[uint32]string{},
	}
}

// Listen 开内网受面（官方 CreateListener 参数：maxAccepts=10）。
func (l *InnerLink) Listen(addr string, port int) error {
	id, err := l.r.CreateListener(addr, port, 10, reactor.ConnHandler{
		OnOpen:  l.onOpen,
		OnClose: l.onClose,
		OnInput: l.onInput,
	})
	if err != nil {
		return err
	}
	l.mu.Lock()
	l.listener = id
	l.mu.Unlock()
	return nil
}

// CheckConnect 官方 CInnerHook::CheckConnect：comm 已有连接 → true；
// 否则发起外连（官方 CreateConnector(…, 0x400000, 0x400000, 0x4000000,
// 30, …)；失败 trace "create connect error"）。
func (l *InnerLink) CheckConnect(comm uint32, addr string, port int) bool {
	if comm >= commIDMax {
		l.logf("(CInnerHook::CheckConnect)(comm_id >= 0) && (comm_id < COMM_ID_MAX) comm=%d", comm)
		return false
	}
	l.mu.Lock()
	if conn, ok := l.commConn[comm]; ok && conn >= 0 {
		l.mu.Unlock()
		return true
	}
	target := fmt.Sprintf("%s:%d", addr, port)
	if l.dialing[comm] == target {
		l.mu.Unlock() // 同一目标拨号中
		return true
	}
	l.dialing[comm] = target
	l.mu.Unlock()

	id, err := l.r.CreateConnector(addr, port, reactor.ConnHandler{
		OnOpen:  l.onOpen,
		OnClose: l.onClose,
		OnInput: l.onInput,
	})
	if err != nil {
		l.logf("(CInnerHook::CheckConnect)create connect error: %v", err)
		l.mu.Lock()
		delete(l.dialing, comm)
		l.mu.Unlock()
		return false
	}
	l.mu.Lock()
	l.pendComm[id] = comm // OnOpen 时发 identify 并预登记
	l.mu.Unlock()
	return true
}

// onOpen 官方 OnAccept/OnConnect 合并语义：接入连接 comm 未知（等对端
// identify）；外连连接立即发 identify 自报。
func (l *InnerLink) onOpen(connID int, addr string, port int) {
	l.mu.Lock()
	l.connComm[connID] = commUnknown
	l.acc[connID] = &frameAcc{}
	comm, outbound := l.pendComm[connID]
	if !outbound {
		// 回环极速拨号竞态：onOpen 可能先于 CheckConnect 的 pendComm
		// 登记到达——按拨号目标反查 dialing 判定出向（竞态免疫）。
		target := fmt.Sprintf("%s:%d", addr, port)
		for c, t := range l.dialing {
			if t == target {
				comm, outbound = c, true
				l.pendComm[connID] = comm
				break
			}
		}
	}
	if outbound {
		// 外连：预登记 comm（发送可用），identify 确认后兜底重写
		l.commConn[comm] = connID
		delete(l.dialing, comm)
	}
	l.mu.Unlock()
	if outbound {
		if frame, err := encodeInner(l.selfComm, 0, innerIdentifyID2, nil); err == nil {
			l.r.Send(connID, frame)
		}
	}
	l.logf("inner: conn %d open %s:%d%s", connID, addr, port, map[bool]string{true: "（外连，已发 identify）", false: ""}[outbound])
}

// onClose 清连接表：conn 是某 comm 当前连接时摘除，允许 announce 重连
// （官方 OnClose 后 CheckConnect 依 m_ConnectorByComm 判定重连）。
func (l *InnerLink) onClose(connID int, addr string, port int) {
	l.mu.Lock()
	comm, hadComm := l.connComm[connID]
	delete(l.connComm, connID)
	delete(l.acc, connID)
	if pcomm, ok := l.pendComm[connID]; ok {
		delete(l.pendComm, connID)
		delete(l.dialing, pcomm) // 拨号失败/断线后允许广播发现重拨
		if l.commConn[pcomm] == connID {
			delete(l.commConn, pcomm) // 未及 identify 即断：乐观登记一并回收
		}
	}
	if hadComm && comm != commUnknown {
		if cur, ok := l.commConn[comm]; ok && cur == connID {
			delete(l.commConn, comm)
		}
	}
	l.mu.Unlock()
	l.logf("inner: conn %d close %s:%d", connID, addr, port)
}

// onInput 帧累积 + 分发（官方 OnReceive→ReceiveMsg：identify 登记 /
// exit 上抛 / 业务回调；日志 "%s connector_id %d comm_id %d received"）。
func (l *InnerLink) onInput(connID int, data []byte) {
	acc := func() *frameAcc {
		l.mu.Lock()
		defer l.mu.Unlock()
		a := l.acc[connID]
		if a == nil {
			a = &frameAcc{}
			l.acc[connID] = a
		}
		return a
	}()
	err := acc.feed(data, func(comm, id1, id2 uint32, payload []byte) {
		switch id2 {
		case innerIdentifyID2:
			l.mu.Lock()
			l.connComm[connID] = comm
			l.commConn[comm] = connID
			l.mu.Unlock()
			l.logf("%s connector_id %d comm_id %d received", l.appName, connID, comm)
			if comm >= commStoreMin && comm < commStoreMax && l.ev.OnStoreIdentified != nil {
				l.ev.OnStoreIdentified()
			}
		case innerExitID2:
			if l.ev.OnExit != nil {
				l.ev.OnExit()
			}
		default:
			if l.ev.OnMessage != nil {
				l.ev.OnMessage(comm, id1, id2, payload)
			}
		}
	})
	if err != nil {
		l.logf("(CInnerHook::ReceiveMsg)%v", err)
	}
}

// Connected comm 是否登记（外连乐观登记面；发现就绪判定用）。
func (l *InnerLink) Connected(comm uint32) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.commConn[comm]
	return ok
}

// SendMsg 官方 CInnerHook::SendMsg(comm_id, id1, id2, payload, len)：
// comm 寻址（非 conn id）；无连接 trace 后 false。
func (l *InnerLink) SendMsg(comm, id1, id2 uint32, payload []byte) bool {
	if comm >= commIDMax {
		l.logf("(CInnerHook::SendMsg)(comm_id >= 0) && (comm_id < COMM_ID_MAX) comm=%d", comm)
		return false
	}
	l.mu.Lock()
	conn, ok := l.commConn[comm]
	l.mu.Unlock()
	if !ok || conn < 0 {
		l.logf("(CInnerHook::SendMsg)connect not exists comm=%d", comm)
		return false
	}
	frame, err := encodeInner(l.selfComm, id1, id2, payload)
	if err != nil {
		l.logf("(CInnerHook::SendMsg)%v", err)
		return false
	}
	return l.r.Send(conn, frame)
}

// SendToWorld 官方 SendToWorld(id1, id2, payload)：comm 0。
func (l *InnerLink) SendToWorld(id1, id2 uint32, payload []byte) bool {
	return l.SendMsg(commWorld, id1, id2, payload)
}

// SendToStore 官方 SendToStore(index<MAX_STORE_NUM=4)：comm 4+index。
func (l *InnerLink) SendToStore(index int, id1, id2 uint32, payload []byte) bool {
	if index < 0 || index >= commStoreMax-commStoreMin {
		l.logf("(CInnerHook::SendToStore)size_t(index) < MAX_STORE_NUM index=%d", index)
		return false
	}
	return l.SendMsg(uint32(commStoreMin+index), id1, id2, payload)
}

// SendToAllStore 官方 SendToAllStore：遍历 comm 4..7 逐个 SendMsg。
func (l *InnerLink) SendToAllStore(id1, id2 uint32, payload []byte) bool {
	sent := false
	for c := commStoreMin; c < commStoreMax; c++ {
		if l.SendMsg(uint32(c), id1, id2, payload) {
			sent = true
		}
	}
	return sent
}

// SendToMember 官方 SendToMember(index<MAX_MEMBER_NUM=0x18)：comm
// 0x18+index。SendAuctionMessage 出站即此（id1=0, id2=0xD）。
func (l *InnerLink) SendToMember(index int, id1, id2 uint32, payload []byte) bool {
	if index < 0 || index >= memberIndexMax {
		l.logf("(CInnerHook::SendToMember)size_t(index) < MAX_MEMBER_NUM index=%d", index)
		return false
	}
	return l.SendMsg(uint32(commMemberMin+index), id1, id2, payload)
}

// HasStore 报告是否有 store 连接（comm 4..7 任一登记）。
func (l *InnerLink) HasStore() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for c := commStoreMin; c < commStoreMax; c++ {
		if _, ok := l.commConn[uint32(c)]; ok {
			return true
		}
	}
	return false
}

// ConnStats 呈现用：各命名空间连接数。
func (l *InnerLink) ConnStats() (world, store, member int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, comm := range l.connComm {
		switch {
		case comm == commWorld:
			world++
		case comm >= commStoreMin && comm < commStoreMax:
			store++
		case comm >= commMemberMin && comm < commMemberMax:
			member++
		}
	}
	return
}

// encodeLoadRequest 构造 {district, server} 8 字节加载请求体
// （0x40 用；0x3F 只带 serverId 4 字节，见 state.go）。
func encodeLoadRequest(district, server uint32) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b[0:], district)
	binary.LittleEndian.PutUint32(b[4:], server)
	return b
}
