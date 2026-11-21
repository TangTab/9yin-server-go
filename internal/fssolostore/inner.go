package fssolostore

import (
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// InnerEvents 是官方 CInnerHook::SetCallback 四回调（Start @0x100C012A
// 挂 0x100C0B50 族静态 thunk）的上抛面。
type InnerEvents struct {
	// OnMessage 业务帧（identify/exit 之外；payload 为拷贝）。
	// comm=对端身份，id1=请求方序号，id2=业务消息号。
	OnMessage func(comm, id1, id2 uint32, payload []byte)
	// OnExit id2=0xFFFE 退出请求（对端要求关停）。
	OnExit func()
}

// InnerLink 复刻 SoloStore 侧 CInnerHook 受面：
//
//	Listen(addr, port=0) → CreateListener（官方 pendings 语义；
//	  InnerPort==0 → 临时口回填实际口，GetListenerPort 语义）；
//	连入方以 identify 帧 {12?, comm, id1, 0xFFFF} 自报身份
//	（world=0/public=1/auction=2/entry=8+idx/member=0x18+idx）；
//	SendMsg 按对端 comm 寻址（无连接 → trace 后 false，
//	官方 SendToWorld/SendToPublic/SendToAuction/SendToMember 同一底座）。
type InnerLink struct {
	r    *reactor.Reactor
	ev   InnerEvents
	self uint32 // StoreId+4
	logf func(format string, args ...any)

	mu       sync.Mutex
	listener int
	connComm map[int]uint32 // connID → 对端 comm
	commConn map[uint32]int // comm → connID
	acc      map[int]*frameAcc
}

const commUnknown = 0xFFFFFFFF

// NewInnerLink 建链路（self=StoreId+4）。
func NewInnerLink(r *reactor.Reactor, ev InnerEvents, self uint32,
	logf func(format string, args ...any)) *InnerLink {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &InnerLink{
		r: r, ev: ev, self: self, logf: logf,
		listener: -1,
		connComm: map[int]uint32{},
		commConn: map[uint32]int{},
		acc:      map[int]*frameAcc{},
	}
}

// Listen 开内网受面（port==0 → 系统分配并回填）。
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
	if port == 0 {
		if ap, ok := l.r.ListenerPort(id); ok {
			return ap, nil
		}
	}
	return port, nil
}

func (l *InnerLink) onOpen(connID int, host string, port int) {
	l.mu.Lock()
	l.connComm[connID] = commUnknown
	l.acc[connID] = &frameAcc{}
	l.mu.Unlock()
	// 官方双向 identify：受面接受连接即回自身 identify——
	// public 端 OnStoreIdentified（装载链事件1）依赖此帧。
	if frame, err := encodeInner(l.self, 0, innerIdentifyID2, nil); err == nil {
		l.r.Send(connID, frame)
	}
}

func (l *InnerLink) onClose(connID int, host string, port int) {
	l.mu.Lock()
	if c, ok := l.connComm[connID]; ok && c != commUnknown {
		delete(l.commConn, c)
	}
	delete(l.connComm, connID)
	delete(l.acc, connID)
	l.mu.Unlock()
}

func (l *InnerLink) onInput(connID int, data []byte) {
	l.mu.Lock()
	acc := l.acc[connID]
	l.mu.Unlock()
	if acc == nil {
		return
	}
	var deliver []func()
	err := acc.feed(data, func(comm, id1, id2 uint32, payload []byte) {
		deliver = append(deliver, func() { l.frame(connID, comm, id1, id2, payload) })
	})
	if err != nil {
		l.logf("%v", err)
	}
	for _, fn := range deliver {
		fn()
	}
}

// frame 官方 OnReceive 分流：identify 登记身份 / exit 上抛 / 业务回调。
func (l *InnerLink) frame(connID int, comm, id1, id2 uint32, payload []byte) {
	switch id2 {
	case innerIdentifyID2:
		l.mu.Lock()
		old := l.connComm[connID]
		if old != commUnknown {
			delete(l.commConn, old)
		}
		l.connComm[connID] = comm
		l.commConn[comm] = connID
		l.mu.Unlock()
		return
	case innerExitID2:
		if l.ev.OnExit != nil {
			l.ev.OnExit()
		}
		return
	}
	if l.ev.OnMessage != nil {
		l.ev.OnMessage(comm, id1, id2, payload)
	}
}

// SelfComm 自身 comm（StoreId+4）。
func (l *InnerLink) SelfComm() uint32 { return l.self }

// Connected 对端 comm 是否在线（CheckResponse 失败 trace 面）。
func (l *InnerLink) Connected(comm uint32) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.commConn[comm]
	return ok
}

// AnyConnected 是否已有任一对端 identify 连入（store 纯被连，发现
// 就绪判定用）。
func (l *InnerLink) AnyConnected() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.commConn) > 0
}

// SendMsg 按对端 comm 发帧（id2=业务回执号；payload 官方回执体）。
func (l *InnerLink) SendMsg(dst, id1, id2 uint32, payload []byte) bool {
	l.mu.Lock()
	connID, ok := l.commConn[dst]
	l.mu.Unlock()
	if !ok {
		l.logf("(SoloStore::CheckResponse)send failed comm:%d", dst)
		return false
	}
	frame, err := encodeInner(l.self, id1, id2, payload)
	if err != nil {
		return false
	}
	return l.r.Send(connID, frame)
}

// SendToWorld/SendToPublic/SendToAuction/SendToMember 官方
// CInnerHook 四寻址面（CheckResponse 0/1/2/else 分支逐字对应）。
func (l *InnerLink) SendToWorld(id1, id2 uint32, payload []byte) bool {
	return l.SendMsg(commWorld, id1, id2, payload)
}

func (l *InnerLink) SendToPublic(id1, id2 uint32, payload []byte) bool {
	return l.SendMsg(commPublic, id1, id2, payload)
}

func (l *InnerLink) SendToAuction(id1, id2 uint32, payload []byte) bool {
	return l.SendMsg(commAuction, id1, id2, payload)
}

func (l *InnerLink) SendToMember(memberIdx int, id1, id2 uint32, payload []byte) bool {
	if memberIdx < 0 || memberIdx >= memberNumMax {
		l.logf("(SoloStore::CheckResponse)send to member failed idx:%d", memberIdx)
		return false
	}
	return l.SendMsg(uint32(commMemberMin+memberIdx), id1, id2, payload)
}
