package fssolopublic

import (
	"fmt"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// InnerEvents 是 InnerLink 上抛给宿主的事件（官方 CInnerHook::SetCallback
// 四回调；store identify → 状态机事件 1 → 0x2D 装载请求）。
type InnerEvents struct {
	// OnMessage 业务消息（identify/exit 之外的帧；payload 为拷贝）。
	OnMessage func(comm, id1, id2 uint32, payload []byte)
	// OnStoreIdentified comm 4..7 identify 完成时触发（官方状态机事件1）。
	OnStoreIdentified func()
	// OnExit id2=0xFFFE 退出请求。
	OnExit func()
}

// InnerLink 复刻 CInnerHook（innerhook.h）：inner TCP 受/发面。
// 受面：CreateListener(InnerAddr, InnerPort=0 分配回填)；发面：
// CheckConnect(comm, addr, port) 广播发现驱动外连（已连去重），外连
// OnConnect 立即回发 identify {12, commSelf, 0, 0xFFFF}。
type InnerLink struct {
	r        *reactor.Reactor
	ev       InnerEvents
	appName  string
	logf     func(format string, args ...any)
	selfComm uint32

	mu       sync.Mutex
	listener int
	connComm map[int]uint32
	commConn map[uint32]int
	pendComm map[int]uint32
	acc      map[int]*frameAcc
	dialing  map[uint32]string
}

const commUnknown = 0xFFFFFFFF

func NewInnerLink(r *reactor.Reactor, ev InnerEvents, appName string, selfComm uint32,
	logf func(format string, args ...any)) *InnerLink {
	return &InnerLink{
		r: r, ev: ev, appName: appName, selfComm: selfComm, logf: logf,
		listener: -1,
		connComm: map[int]uint32{},
		commConn: map[uint32]int{},
		pendComm: map[int]uint32{},
		acc:      map[int]*frameAcc{},
		dialing:  map[uint32]string{},
	}
}

// Listen 开内网受面；返回实际侦听端口（官方 Port=0 → GetListenerPort 回填）。
func (l *InnerLink) Listen(addr string, port int) (int, error) {
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
	actual, ok := l.r.ListenerPort(id)
	if !ok {
		actual = port
	}
	return actual, nil
}

// CheckConnect 官方幂等外连（已连/拨号中 → true）。
func (l *InnerLink) CheckConnect(comm uint32, addr string, port int) bool {
	if comm >= CommIDMax {
		l.logf("(CInnerHook::CheckConnect)comm_id 越界 comm=%d", comm)
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

// CheckStoreConnect 官方包装：comm 4+index（MAX_STORE_NUM=4）。
func (l *InnerLink) CheckStoreConnect(index int, addr string, port int) bool {
	if index < 0 || index >= CommStoreMax-CommStoreMin {
		l.logf("(CInnerHook::CheckStoreConnect)size_t(index) < MAX_STORE_NUM index=%d", index)
		return false
	}
	return l.CheckConnect(uint32(CommStoreMin+index), addr, port)
}

// CheckEntryConnect 官方包装：comm 8+index（MAX_ENTRY_NUM=16，solo 特有）。
func (l *InnerLink) CheckEntryConnect(index int, addr string, port int) bool {
	if index < 0 || index >= CommEntryMax-CommEntryMin {
		l.logf("(CInnerHook::CheckEntryConnect)size_t(index) < MAX_ENTRY_NUM index=%d", index)
		return false
	}
	return l.CheckConnect(uint32(CommEntryMin+index), addr, port)
}

func (l *InnerLink) onOpen(connID int, addr string, port int) {
	l.mu.Lock()
	l.connComm[connID] = commUnknown
	l.acc[connID] = &frameAcc{}
	comm, outbound := l.pendComm[connID]
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
	delete(l.pendComm, connID)
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
			if comm >= CommStoreMin && comm < CommStoreMax && l.ev.OnStoreIdentified != nil {
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

// SendMsg 官方 SendMsg(comm_id, id1, id2, payload)：comm 寻址。
func (l *InnerLink) SendMsg(comm, id1, id2 uint32, payload []byte) bool {
	if comm >= CommIDMax {
		l.logf("(CInnerHook::SendMsg)comm_id 越界 comm=%d", comm)
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

// SendToWorld / SendToStore / SendToAllStore / SendToEntry / SendToAllEntry
// / SendToMember / SendToAllMember 官方发送族包装。
func (l *InnerLink) SendToWorld(id1, id2 uint32, payload []byte) bool {
	return l.SendMsg(CommWorld, id1, id2, payload)
}

func (l *InnerLink) SendToStore(index int, id1, id2 uint32, payload []byte) bool {
	if index < 0 || index >= CommStoreMax-CommStoreMin {
		l.logf("(CInnerHook::SendToStore)size_t(index) < MAX_STORE_NUM index=%d", index)
		return false
	}
	return l.SendMsg(uint32(CommStoreMin+index), id1, id2, payload)
}

func (l *InnerLink) SendToAllStore(id1, id2 uint32, payload []byte) bool {
	sent := false
	for c := CommStoreMin; c < CommStoreMax; c++ {
		if l.SendMsg(uint32(c), id1, id2, payload) {
			sent = true
		}
	}
	return sent
}

func (l *InnerLink) SendToEntry(index int, id1, id2 uint32, payload []byte) bool {
	if index < 0 || index >= CommEntryMax-CommEntryMin {
		l.logf("(CInnerHook::SendToEntry)size_t(index) < MAX_ENTRY_NUM index=%d", index)
		return false
	}
	return l.SendMsg(uint32(CommEntryMin+index), id1, id2, payload)
}

func (l *InnerLink) SendToAllEntry(id1, id2 uint32, payload []byte) bool {
	sent := false
	for c := CommEntryMin; c < CommEntryMax; c++ {
		if l.SendMsg(uint32(c), id1, id2, payload) {
			sent = true
		}
	}
	return sent
}

func (l *InnerLink) SendToMember(index int, id1, id2 uint32, payload []byte) bool {
	if index < 0 || index >= CommMemberMax-CommMemberMin {
		l.logf("(CInnerHook::SendToMember)size_t(index) < MAX_MEMBER_NUM index=%d", index)
		return false
	}
	return l.SendMsg(uint32(CommMemberMin+index), id1, id2, payload)
}

func (l *InnerLink) SendToAllMember(id1, id2 uint32, payload []byte) bool {
	sent := false
	for c := CommMemberMin; c < CommMemberMax; c++ {
		if l.SendMsg(uint32(c), id1, id2, payload) {
			sent = true
		}
	}
	return sent
}

// HasStore 报告是否有 store 连接（comm 4..7 任一登记）。
func (l *InnerLink) HasStore() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for c := CommStoreMin; c < CommStoreMax; c++ {
		if _, ok := l.commConn[uint32(c)]; ok {
			return true
		}
	}
	return false
}

// ConnStats 呈现用：各命名空间连接数。
func (l *InnerLink) ConnStats() (world, store, entry, member int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, comm := range l.connComm {
		switch {
		case comm == CommWorld:
			world++
		case comm >= CommStoreMin && comm < CommStoreMax:
			store++
		case comm >= CommEntryMin && comm < CommEntryMax:
			entry++
		case comm >= CommMemberMin && comm < CommMemberMax:
			member++
		}
	}
	return
}
