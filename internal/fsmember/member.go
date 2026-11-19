package fsmember

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
)

// Member 本地状态机（MemberStateMachine，this+0x24 四态定案，§6）。
type MemberState int32

const (
	StateUnknown MemberState = iota // 0：启动中（场景装载）
	StateOpened                     // 1：开放服务
	StateClosing                    // 2：关服中（场景清空门）
	StateClosed                     // 3：已关闭
)

// String 状态名（日志用）。
func (s MemberState) String() string {
	switch s {
	case StateUnknown:
		return "Unknown"
	case StateOpened:
		return "Opened"
	case StateClosing:
		return "Closing"
	case StateClosed:
		return "Closed"
	}
	return fmt.Sprintf("State(%d)", int32(s))
}

// 广播包定案（BroadcastMemberInfo，0x28B，§8）：kind=3/magic=0x300 头 +
// ip[16] + 端口 + ServerId/MemberId（DistrictId 死存于包外，不入包）。
const (
	broadcastKind  = 3
	broadcastMagic = 0x300
)

// msg 内网消息（三队列条目）。
type msg struct {
	ident   uint32
	typ     uint32
	payload []byte
}

// Member 复刻 GameMember 服务本体：三消息队列 + 状态机 + 注册表编排。
type Member struct {
	State atomic.Int32 // MemberState

	DistrictID uint32
	ServerID   uint32
	MemberID   uint32

	MaxLetterNum   int
	MaxRunningTime int // DoWait 单帧预算毫秒（官方同名属性）

	Scenes  *SceneRegistry
	Players *PlayerRegistry
	Pub     *PubSpaces
	Chunks  *ChunkStore

	// OnSceneTick 每帧场景回调（CGameScene::DoWait 等价挂点）。
	OnSceneTick func()
	// OnStateChange 状态迁移通知（日志/广播联动）。
	OnStateChange func(from, to MemberState)
	// OnSceneCommand 场景命令投递（SendToScene 等价挂点：entry 流与
	// PublicMessage 透传共用，§7/§9）。
	OnSceneCommand func(sceneID uint32, typ uint32, payload []byte)
	// SendToWorld/SendToPublic/SendToStorage/SendToEntry 外发面
	//（CInterConnect 寻路；nil 时外发丢弃）。
	SendToWorld    func(typ uint32, payload []byte) error
	SendToPublic   func(typ uint32, payload []byte) error
	SendToStorage  func(ident uint32, typ uint32, payload []byte) error
	SendToAllStor  func(typ uint32, payload []byte) error
	SendToEntry    func(ident uint32, typ uint32, payload []byte) error
	// Logf 日志面。
	Logf func(format string, args ...any)

	mu      sync.Mutex
	worldQ  []msg
	storeQ  []msg
	publicQ []msg

	playerBufMu sync.Mutex
	playerBufs  []msg // entry_forw 3 → CPlayerBuffers（PlayerSingleBuffer 语义）

	closeLogMu   sync.Mutex
	lastCloseLog int // Closing 门日志去重（-1=未记）
}

// New 服务构造（属性缺省对齐官方：MaxLetterNum=0、MaxRunningTime=100）。
func New(districtID, serverID, memberID uint32) *Member {
	m := &Member{
		DistrictID:     districtID,
		ServerID:       serverID,
		MemberID:       memberID,
		MaxRunningTime: 100,
		Scenes:         NewSceneRegistry(),
		Players:        NewPlayerRegistry(),
		Pub:            NewPublicSpaces(),
		Chunks:         NewChunkStore(),
	}
	m.State.Store(int32(StateUnknown))
	return m
}

// OwnIdent 本 member 内网标识（0x18+MemberID）。
func (m *Member) OwnIdent() uint32 { return MemberIdentBase + m.MemberID }

func (m *Member) logf(format string, args ...any) {
	if m.Logf != nil {
		m.Logf(format, args...)
	}
}

// ---- 三队列入口（PutWorldMessage/PutStoreMessage/PutPublicMessage） ----

// PutWorld world 队列入队。
func (m *Member) PutWorld(ident, typ uint32, payload []byte) {
	m.mu.Lock()
	m.worldQ = append(m.worldQ, msg{ident, typ, payload})
	m.mu.Unlock()
}

// PutStore store 队列入队。
func (m *Member) PutStore(ident, typ uint32, payload []byte) {
	m.mu.Lock()
	m.storeQ = append(m.storeQ, msg{ident, typ, payload})
	m.mu.Unlock()
}

// PutPublic public 队列入队。
func (m *Member) PutPublic(ident, typ uint32, payload []byte) {
	m.mu.Lock()
	m.publicQ = append(m.publicQ, msg{ident, typ, payload})
	m.mu.Unlock()
}

// PutPlayerBuffer entry_forw 3 → 玩家缓冲池（CPlayerBuffers::PutMessage，
// 官方剥 8B {ident,serial} 头后入池，这里保头重组）。
func (m *Member) PutPlayerBuffer(ident, serial uint32, payload []byte) {
	body := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint32(body[0:], ident)
	binary.LittleEndian.PutUint32(body[4:], serial)
	copy(body[8:], payload)
	m.playerBufMu.Lock()
	m.playerBufs = append(m.playerBufs, msg{ident, ForwSetOnline, body})
	m.playerBufMu.Unlock()
}

// DrainPlayerBuffers 取空玩家缓冲池（进场流程消费）。
func (m *Member) DrainPlayerBuffers() []msg {
	m.playerBufMu.Lock()
	out := m.playerBufs
	m.playerBufs = nil
	m.playerBufMu.Unlock()
	return out
}

// DoWait 复刻 DoWait 主循环一帧：world→store→public 三队列 → 场景帧 →
// 状态机 → 广播（§4.2 官方次序）。
func (m *Member) DoWait() {
	m.drain(&m.worldQ, m.ProcessWorld)
	m.drain(&m.storeQ, m.ProcessStore)
	m.drain(&m.publicQ, m.ProcessPublic)
	if m.OnSceneTick != nil {
		m.OnSceneTick()
	}
	m.stateMachine()
}

func (m *Member) drain(q *[]msg, fn func(ident, typ uint32, payload []byte)) {
	m.mu.Lock()
	pending := *q
	*q = nil
	m.mu.Unlock()
	for _, g := range pending {
		fn(g.ident, g.typ, g.payload)
	}
}

// SetState 状态迁移（含 Opened→Closing 的 CloseMember 入口）。
func (m *Member) SetState(s MemberState) {
	old := MemberState(m.State.Swap(int32(s)))
	if old != s && m.OnStateChange != nil {
		m.OnStateChange(old, s)
	}
}

// stateMachine 复刻 MemberStateMachine：Closing 态下场景清空才落
// Closed（"(GameMember::MemberStateClosing)scene has player" 门；
// 日志仅在余量变化时打点，避免逐帧刷屏）。
func (m *Member) stateMachine() {
	switch MemberState(m.State.Load()) {
	case StateClosing:
		n := len(m.Scenes.Snapshot())
		if n == 0 {
			m.SetState(StateClosed)
			return
		}
		m.closeLogMu.Lock()
		if m.lastCloseLog != n {
			m.lastCloseLog = n
			m.logf("(GameMember::MemberStateClosing)scene has player: %d scenes", n)
		}
		m.closeLogMu.Unlock()
	default:
		m.closeLogMu.Lock()
		m.lastCloseLog = -1
		m.closeLogMu.Unlock()
	}
}

// Open 开放服务（启动期场景装载完成 → Opened）。
func (m *Member) Open() { m.SetState(StateOpened) }

// BeginClose 复刻 CommandCloseMember：进入 Closing（踢线/存档流程）。
func (m *Member) BeginClose() { m.SetState(StateClosing) }

// BuildBroadcast 复刻 BroadcastMemberInfo 的 0x28B 包
//（kind=3/magic=0x300，ip[16]@0x08、port@0x18、ServerId@0x20、
// MemberId@0x24，§8 定案布局）。
func (m *Member) BuildBroadcast(ip string, port int) []byte {
	pkt := make([]byte, 0x28)
	binary.LittleEndian.PutUint32(pkt[0x00:], broadcastKind)
	binary.LittleEndian.PutUint32(pkt[0x04:], broadcastMagic)
	copy(pkt[0x08:], ip) // CopyString 定长 16
	if len(ip) < 16 {
		pkt[0x08+len(ip)] = 0
	}
	binary.LittleEndian.PutUint32(pkt[0x18:], uint32(port))
	binary.LittleEndian.PutUint32(pkt[0x20:], m.ServerID)
	binary.LittleEndian.PutUint32(pkt[0x24:], m.MemberID)
	return pkt
}

// String 摘要（console:WriteLine 同级信息）。
func (m *Member) String() string {
	return fmt.Sprintf("GameMember state %s scenes %d players %d pub spaces %d",
		MemberState(m.State.Load()), len(m.Scenes.Snapshot()), m.Players.Count(), m.Pub.SpaceCount())
}
