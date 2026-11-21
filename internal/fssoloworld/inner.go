package fssoloworld

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// InnerEvents 上抛事件（官方 CInnerHook::SetCallback 四回调 + 五类
// 连接事件 InnerXXXConnected/Disconnect @SoloWorldInnerMsg.obj）。
type InnerEvents struct {
	// OnMessage 业务帧（identify 之外）。comm = 发送方 comm。
	OnMessage func(comm, id1, id2 uint32, payload []byte)
	// OnConnected 对端 identify 登记后按类回调（public=1/auction=2/
	// store=4+idx/entry=8+idx/member=0x18+idx）。
	OnConnected func(comm uint32)
	// OnDisconnect 连接断开（comm 已登记时）。
	OnDisconnect func(comm uint32)
}

// InnerLink 复刻 world 侧 CInnerHook（§六）：
//
//	受面：CreateListener（InnerPort==0 → 系统分配回填）；
//	发面：Check*Connect（广播发现驱动）外连，OnConnect 立即发 identify
//	      {12, 0, 0, 0xFFFF}（world comm=0）；
//	寻址：SendMsg 按 comm；SendToStore/Entry/Member/Public/Auction 家族。
type InnerLink struct {
	r        *reactor.Reactor
	ev       InnerEvents
	appName  string
	logf     func(format string, args ...any)
	selfComm uint32 // = 0（COMM_WORLD）

	mu       sync.Mutex
	listener int
	connComm map[int]uint32
	commConn map[uint32]int
	pendComm map[int]uint32
	acc      map[int]*frameAcc
	dialing  map[uint32]string
}

const commUnknown = 0xFFFFFFFF

// NewInnerLink world comm 恒 0（Start @0x100b1f04 SetCommIdent(0) 定案）。
func NewInnerLink(r *reactor.Reactor, ev InnerEvents, appName string,
	logf func(format string, args ...any)) *InnerLink {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &InnerLink{
		r: r, ev: ev, appName: appName, logf: logf, selfComm: commWorld,
		listener: -1,
		connComm: map[int]uint32{},
		commConn: map[uint32]int{},
		pendComm: map[int]uint32{},
		acc:      map[int]*frameAcc{},
		dialing:  map[uint32]string{},
	}
}

// Listen 内网受面（InnerPort==0 → 临时口回填，官方 GetListenerPort 语义）。
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

// CheckConnect 官方 CInnerHook::CheckConnect。
func (l *InnerLink) CheckConnect(comm uint32, addr string, port int) bool {
	if l.r == nil {
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

// Check*Connect 广播发现五分支（§五接收表）。
func (l *InnerLink) CheckPublicConnect(addr string, port int) bool {
	return l.CheckConnect(commPublic, addr, port)
}

func (l *InnerLink) CheckAuctionConnect(addr string, port int) bool {
	return l.CheckConnect(commAuction, addr, port)
}

func (l *InnerLink) CheckStoreConnect(idx int, addr string, port int) bool {
	if idx < 0 || idx >= storeNumMax {
		l.logf("(CInnerHook::CheckStoreConnect)store idx out bound %d", idx)
		return false
	}
	return l.CheckConnect(uint32(commStoreMin+idx), addr, port)
}

func (l *InnerLink) CheckEntryConnect(idx int, addr string, port int) bool {
	if idx < 0 || idx >= entryIndexMax {
		l.logf("(CInnerHook::CheckEntryConnect)size_t(index) < MAX_ENTRY_NUM index=%d", idx)
		return false
	}
	return l.CheckConnect(uint32(commEntryMin+idx), addr, port)
}

func (l *InnerLink) CheckMemberConnect(idx int, addr string, port int) bool {
	if idx < 0 || idx >= memberNumMax {
		l.logf("(CInnerHook::CheckMemberConnect)member idx out bound %d", idx)
		return false
	}
	return l.CheckConnect(uint32(commMemberMin+idx), addr, port)
}

// onOpen OnAccept/OnConnect 合并：外连方预挂 comm 并立即发 identify。
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
	l.logf("%s connector_id %d address %s:%d%s", l.appName, connID, addr, port,
		map[bool]string{true: " connected", false: " accept"}[outbound])
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
	if hadComm && comm != commUnknown && l.ev.OnDisconnect != nil {
		l.ev.OnDisconnect(comm)
	}
	l.logf("%s connector_id %d address %s:%d closed", l.appName, connID, addr, port)
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
		if id2 == innerIdentifyID2 {
			l.mu.Lock()
			l.connComm[connID] = comm
			l.commConn[comm] = connID
			l.mu.Unlock()
			l.logf("%s connector_id %d comm_id %d received", l.appName, connID, comm)
			if l.ev.OnConnected != nil {
				l.ev.OnConnected(comm)
			}
			return
		}
		if l.ev.OnMessage != nil {
			l.ev.OnMessage(comm, id1, id2, payload)
		}
	})
	if err != nil {
		l.logf("(CInnerHook::ReceiveMsg)%v", err)
	}
}

// SendMsg comm 寻址发送。
func (l *InnerLink) SendMsg(comm, id1, id2 uint32, payload []byte) bool {
	if l.r == nil {
		return false
	}
	if comm >= commIDMax {
		l.logf("(CInnerHook::SendMsg)comm id error comm=%d", comm)
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
		return false
	}
	return l.r.Send(conn, frame)
}

// SendToMember index<MAX_MEMBER_NUM(0x18)：comm 0x18+index（§六定案）。
func (l *InnerLink) SendToMember(index int, id1, id2 uint32, payload []byte) bool {
	if index < 0 || index >= memberNumMax {
		l.logf("(CInnerHook::SendToMember)size_t(index) < MAX_MEMBER_NUM index=%d", index)
		return false
	}
	return l.SendMsg(uint32(commMemberMin+index), id1, id2, payload)
}

// SendToAllMember 遍历 comm 0x18..0x2F。
func (l *InnerLink) SendToAllMember(id1, id2 uint32, payload []byte) bool {
	sent := false
	for c := commMemberMin; c < commMemberMax; c++ {
		if l.SendMsg(uint32(c), id1, id2, payload) {
			sent = true
		}
	}
	return sent
}

// SendToStore index<MAX_STORE_NUM(4)：comm 4+index。
func (l *InnerLink) SendToStore(index int, id1, id2 uint32, payload []byte) bool {
	if index < 0 || index >= storeNumMax {
		l.logf("(CInnerHook::SendToStore)size_t(index) < MAX_STORE_NUM index=%d", index)
		return false
	}
	return l.SendMsg(uint32(commStoreMin+index), id1, id2, payload)
}

// SendToAllStore 遍历 comm 4..7。
func (l *InnerLink) SendToAllStore(id1, id2 uint32, payload []byte) bool {
	sent := false
	for c := commStoreMin; c < commStoreMax; c++ {
		if l.SendMsg(uint32(c), id1, id2, payload) {
			sent = true
		}
	}
	return sent
}

// SendToEntry index<MAX_ENTRY_NUM(0x10)：comm 8+index。
func (l *InnerLink) SendToEntry(index int, id1, id2 uint32, payload []byte) bool {
	if index < 0 || index >= entryIndexMax {
		l.logf("(CInnerHook::SendToEntry)size_t(index) < MAX_ENTRY_NUM index=%d", index)
		return false
	}
	return l.SendMsg(uint32(commEntryMin+index), id1, id2, payload)
}

// SendToAllEntry 遍历 comm 8..0x17（SendOpenConnect 使用）。
func (l *InnerLink) SendToAllEntry(id1, id2 uint32, payload []byte) bool {
	sent := false
	for c := commEntryMin; c < commEntryMax; c++ {
		if l.SendMsg(uint32(c), id1, id2, payload) {
			sent = true
		}
	}
	return sent
}

func (l *InnerLink) SendToPublic(id1, id2 uint32, payload []byte) bool {
	return l.SendMsg(commPublic, id1, id2, payload)
}

func (l *InnerLink) SendToAuction(id1, id2 uint32, payload []byte) bool {
	return l.SendMsg(commAuction, id1, id2, payload)
}

// GetConnected comm 是否登记（GetStoreConnected/GetEntryConnected 族）。
func (l *InnerLink) GetConnected(comm uint32) bool {
	if comm >= commIDMax {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.commConn[comm]
	return ok
}

// ConnCommOf 测试/呈现用。
func (l *InnerLink) ConnCommOf(connID int) (uint32, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	c, ok := l.connComm[connID]
	return c, ok && c != commUnknown
}

// encodeLoadRoles StoreMsg 请求体（WorldStateUnknown → SendToStore；
// 首域 ServerId——store 端 ResetOnline 的复位范围）。
func encodeLoadRoles(sid int) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b[0:], uint32(sid))
	return b
}
