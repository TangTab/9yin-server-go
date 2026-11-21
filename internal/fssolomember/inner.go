package fssolomember

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// InnerEvents 是 InnerLink 上抛事件（官方 CInnerHook::SetCallback 四回调；
// 回调体 @0x102912B0..0x10291520 静态 thunk 直转）。
type InnerEvents struct {
	// OnMessage 业务帧（identify/exit 之外；payload 为拷贝）。
	OnMessage func(comm, id1, id2 uint32, payload []byte)
	// OnPublicConnected 对端 comm==1（public）外连成功后发注册帧
	//（官方 on_connect @0x10291320 内联，此处拆为事件供测试）。
	OnPublicConnected func()
	// OnExit id2=0xFFFE 退出请求。
	OnExit func()
}

// InnerLink 复刻 member 侧 CInnerHook（§五）：
//
//	受面：CreateListener(InnerAddr, InnerPort, pendings=10, 4MB/4MB)——
//	      world/entry 等连入后以 identify 自报 comm；
//	发面：Check*Connect 广播发现驱动外连，OnConnect 立即回发 identify
//	      {12, 0x18+MemberID, 0, 0xFFFF}；对端 comm==1 时补发 member 注册
//	      {DistrictId, ServerId}(8B)（SendToPublic id2=0）；
//	寻址：SendMsg 按 comm（非 conn id），无连接 trace 后 false。
type InnerLink struct {
	r        *reactor.Reactor
	ev       InnerEvents
	appName  string
	logf     func(format string, args ...any)
	selfComm uint32 // 0x18 + MemberID

	mu       sync.Mutex
	listener int
	connComm map[int]uint32 // connID → 对端 comm（identify 后）
	commConn map[uint32]int // comm → connID
	pendComm map[int]uint32 // 外连 connID → 预期 comm
	acc      map[int]*frameAcc
	dialing  map[uint32]string
}

const commUnknown = 0xFFFFFFFF

// NewInnerLink 建链路（selfComm 官方 = 0x18+MemberID，Start @0x1028F45E 段）。
func NewInnerLink(r *reactor.Reactor, ev InnerEvents, appName string, selfComm uint32,
	logf func(format string, args ...any)) *InnerLink {
	if logf == nil {
		logf = func(string, ...any) {}
	}
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

// Attach 挂接 reactor（Server.New 先建休眠链路，Start 时接线）。
func (l *InnerLink) Attach(r *reactor.Reactor) { l.r = r }

// Listen 开内网受面（官方 pendings=10；InnerPort==0 → 临时口，回填实际口
// ——官方 GetListenerPort 语义）。
func (l *InnerLink) Listen(addr string, port int) (actualPort int, err error) {
	id, err := l.r.CreateListener(addr, port, 10, reactor.ConnHandler{
		OnOpen:  l.onOpen,
		OnClose: l.onClose,
		OnInput: l.onInput,
	})
	if err != nil {
		return 0, err
	}
	l.mu.Lock()
	l.listener = id
	l.mu.Unlock()
	if port == 0 {
		if ap, ok := l.r.ListenerPort(id); ok {
			return ap, nil
		}
	}
	return port, nil
}

// CheckConnect 官方 CInnerHook::CheckConnect：已有连接 → true；否则外连。
func (l *InnerLink) CheckConnect(comm uint32, addr string, port int) bool {
	if l.r == nil { // 休眠链路（未 Start）：仅查表
		l.mu.Lock()
		defer l.mu.Unlock()
		_, ok := l.commConn[comm]
		return ok
	}
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
		l.mu.Unlock()
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
	l.pendComm[id] = comm
	l.mu.Unlock()
	return true
}

// CheckPublicConnect / CheckStoreConnect / CheckEntryConnect / CheckAuctionConnect
// on_broadcast 四分支（§四）——CheckEntryConnect 断言 idx < MAX_ENTRY_NUM(0x10)。
func (l *InnerLink) CheckPublicConnect(addr string, port int) bool {
	return l.CheckConnect(commPublic, addr, port)
}

// CheckStoreConnect 官方 @0x10291190：入参是广播包 +0x24 的 StoreId 原值
// （FsSoloStore BroadcastStoreInfo @0x100C4D60 实证），断言
// < MAX_STORE_NUM(4) 后 CheckConnect(index+4) —— comm = 4..7。
func (l *InnerLink) CheckStoreConnect(storeID uint32, addr string, port int) bool {
	if storeID >= storeNumMax {
		l.logf("(CInnerHook::CheckStoreConnect)size_t(index) < MAX_STORE_NUM index=%d", storeID)
		return false
	}
	return l.CheckConnect(commStoreMin+storeID, addr, port)
}

func (l *InnerLink) CheckEntryConnect(index int, addr string, port int) bool {
	if index < 0 || index >= entryIndexMax {
		l.logf("(CInnerHook::CheckEntryConnect)size_t(index) < MAX_ENTRY_NUM index=%d", index)
		return false
	}
	return l.CheckConnect(uint32(commEntryMin+index), addr, port)
}

func (l *InnerLink) CheckAuctionConnect(addr string, port int) bool {
	return l.CheckConnect(commAuction, addr, port)
}

// onOpen 官方 OnAccept/OnConnect 合并语义：接入方 comm 未知（等 identify）；
// 外连方立即发 identify；identify 前预登记 comm 使发送可用。
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
		l.commConn[comm] = connID
		delete(l.dialing, comm)
	}
	l.mu.Unlock()
	if outbound {
		if frame, err := encodeInner(l.selfComm, 0, innerIdentifyID2, nil); err == nil {
			l.r.Send(connID, frame)
		}
	}
	l.logf("inner: conn %d open %s:%d%s", connID, addr, port,
		map[bool]string{true: "（外连，已发 identify）", false: ""}[outbound])
}

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
			if comm == commPublic && l.ev.OnPublicConnected != nil {
				l.ev.OnPublicConnected()
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

// SendMsg comm 寻址发送；无连接 trace 后 false。
func (l *InnerLink) SendMsg(comm, id1, id2 uint32, payload []byte) bool {
	if l.r == nil { // 休眠链路：无连接可达
		return false
	}
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

func (l *InnerLink) SendToWorld(id1, id2 uint32, payload []byte) bool {
	return l.SendMsg(commWorld, id1, id2, payload)
}

// SendToPublic 官方 SendToPublic：comm 1（on_connect 注册帧即此，id2=0）。
func (l *InnerLink) SendToPublic(id1, id2 uint32, payload []byte) bool {
	return l.SendMsg(commPublic, id1, id2, payload)
}

func (l *InnerLink) SendToAuction(id1, id2 uint32, payload []byte) bool {
	return l.SendMsg(commAuction, id1, id2, payload)
}

// SendToStore index<MAX_STORE_NUM(4)：comm 4+index。
func (l *InnerLink) SendToStore(index int, id1, id2 uint32, payload []byte) bool {
	if index < 0 || index >= storeNumMax {
		l.logf("(CInnerHook::SendToStore)size_t(index) < MAX_STORE_NUM index=%d", index)
		return false
	}
	return l.SendMsg(uint32(commStoreMin+index), id1, id2, payload)
}

// SendToEntry index<MAX_ENTRY_NUM(0x10)：comm 8+index。
func (l *InnerLink) SendToEntry(index int, id1, id2 uint32, payload []byte) bool {
	if index < 0 || index >= entryIndexMax {
		l.logf("(CInnerHook::SendToEntry)size_t(index) < MAX_ENTRY_NUM index=%d", index)
		return false
	}
	return l.SendMsg(uint32(commEntryMin+index), id1, id2, payload)
}

// SendToAllEntry 官方 SendToAllEntry：遍历 comm 8..0x17 逐个 SendMsg。
func (l *InnerLink) SendToAllEntry(id1, id2 uint32, payload []byte) bool {
	sent := false
	for c := commEntryMin; c < commEntryMax; c++ {
		if l.SendMsg(uint32(c), id1, id2, payload) {
			sent = true
		}
	}
	return sent
}

// GetStoreConnected 官方 GetStoreConnected(index)：comm 4+index 是否登记。
func (l *InnerLink) GetStoreConnected(index int) bool {
	if index < 0 || index >= storeNumMax {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.commConn[uint32(commStoreMin+index)]
	return ok
}

// GetConnected comm 是否登记（GetStoreConnected 泛化；发现就绪
// 判定用）。
func (l *InnerLink) GetConnected(comm uint32) bool {
	if comm >= commIDMax {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.commConn[comm]
	return ok
}

// GetPublicConnected public(comm 1) 是否登记（发现就绪判定用）。
func (l *InnerLink) GetPublicConnected() bool { return l.GetConnected(commPublic) }

// GetEntryConnected entry index 是否登记（发现就绪判定用）。
func (l *InnerLink) GetEntryConnected(index int) bool {
	if index < 0 || index >= entryIndexMax {
		return false
	}
	return l.GetConnected(uint32(commEntryMin + index))
}

// ConnCommOf 测试/呈现用：connID → 对端 comm。
func (l *InnerLink) ConnCommOf(connID int) (uint32, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	c, ok := l.connComm[connID]
	return c, ok && c != commUnknown
}

// encodeRegisterPayload on_connect member 注册帧体 {DistrictId, ServerId}(8B)。
func encodeRegisterPayload(district, server uint32) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b[0:], district)
	binary.LittleEndian.PutUint32(b[4:], server)
	return b
}
