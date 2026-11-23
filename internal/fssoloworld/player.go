package fssoloworld

import (
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"

	"sync"
	"time"
)

// PlayerState 玩家状态机 14 态（SoloWorldPlayerState.obj rdata 顺序定案）。
type PlayerState int

const (
	PlUnknown PlayerState = iota
	PlWaitBreak
	PlLogging
	PlQueuing
	PlGetting
	PlLogged
	PlChoosing
	PlCreating
	PlDeleting
	PlReviving
	PlOnline
	PlOffline
	PlSaving
	PlLeaving
)

var playerStateNames = map[PlayerState]string{
	PlUnknown: "Unknown", PlWaitBreak: "WaitBreak", PlLogging: "Logging",
	PlQueuing: "Queuing", PlGetting: "Getting", PlLogged: "Logged",
	PlChoosing: "Choosing", PlCreating: "Creating", PlDeleting: "Deleting",
	PlReviving: "Reviving", PlOnline: "Online", PlOffline: "Offline",
	PlSaving: "Saving", PlLeaving: "Leaving",
}

func (s PlayerState) String() string { return playerStateNames[s] }

// Player 复刻 player_t（0x1A0，§八）：PERSISTID(conn1,conn2)=entry 客户端键。
type Player struct {
	Conn1, Conn2 uint32 // entry_client_id / entry idx 侧键（map key）
	Account      string // char[16]@+0x18C
	Name         string // wchar 区
	State        PlayerState
	loginAt      time.Time
	activeAt     time.Time
	sceneID      int
	queued       bool
	vip          bool
	// roleState（store 0x02 回执 RST1 尾段）：进场原生生成的数据源。
	// 缺席（官方 store/旧档）时 entergen 回退 replay 模式。
	roleState     rolestate.RoleState
	haveRoleState bool
	// 自定义配置域会话闸（连接生命周期，不落库）：客户端登录期会先
	// 上传默认键位/个性配置转储（恢复帧未必已应用）——首传且库存有档
	// 时跳过落库防默认覆盖玩家配置（custom.go 首传闸）。
	customizingSeen bool
	keybindSeen     bool
	// posSaveAt 位置节流存档水位（position.go absorbPlayerPosition；
	// 2s 窗口内只更新内存权威位，窗口到点/下线才发 store 0x04）。
	posSaveAt time.Time
	// NPC 视口/巡逻簿记（连接生命周期）：npcVisible=已物化 ident 集，
	// npcPos=当前平面位置（op0x0A 67B 位移包更新）——npcReconcile 按
	// 此 diff 增删；patrolReadyAt=巡逻首拍时刻（进场物化 +5s）；
	// npcMoveState=巡逻 NPC 行走/驻停水位（State 翻转帧用）。
	npcMu         sync.Mutex
	npcVisible    map[uint32]bool
	npcPos        [4]float64
	npcScene      string
	patrolReadyAt time.Time
	npcMoveState  map[uint32]bool
	// sceneSwitch 切图在途态（sceneswitch.go；nil=不在切换）：0x0C 已发
	// 到目标场景 0x09 就绪屏障之间，位移上报全丢弃（旧场景过期 0x0A）。
	sceneSwitchMu sync.Mutex
	sceneSwitch   *sceneSwitchState
	// portalCoolAt 传送门触发冷却水位（portals.go；场景级 2s）。
	portalCoolAt time.Time
	// 手动格挡会话态（parry.go；连接生命周期，不落库）：parryBaseline=
	// 官服族 m54 状态上报的基调值（0=未采集；首条上报即基调=未格挡）；
	// parrying=当前格挡权威位。双方言：0/1 直判（0=按下）与令牌上报
	//（按压=基调+parryTokenDelta）共用这两个字段；基调随副本切景轮换，
	// 由 K 形判别自动采纳。
	parryMu       sync.Mutex
	parryBaseline int64
	parrying      bool
}

// WorldPlayers 复刻 CWorldPlayers（0x48）：数组 + 空闲槽 + PERSISTID 索引。
type WorldPlayers struct {
	mu     sync.Mutex
	byKey  map[uint64]*Player
	byAcc  map[string]*Player
	byName map[string]*Player
	order  []*Player
	max    int
}

// NewWorldPlayers max=MaxPlayers（GetPlayerMax）。
func NewWorldPlayers(max int) *WorldPlayers {
	if max <= 0 {
		max = 8192
	}
	return &WorldPlayers{
		byKey:  map[uint64]*Player{},
		byAcc:  map[string]*Player{},
		byName: map[string]*Player{},
		max:    max,
	}
}

func persistKey(conn1, conn2 uint32) uint64 { return uint64(conn1)<<32 | uint64(conn2) }

// AddPlayer 官方 AddPlayer(conn1, conn2, account, flag)。
func (w *WorldPlayers) AddPlayer(conn1, conn2 uint32, account string) *Player {
	w.mu.Lock()
	defer w.mu.Unlock()
	key := persistKey(conn1, conn2)
	if p, ok := w.byKey[key]; ok {
		return p
	}
	now := time.Now()
	p := &Player{Conn1: conn1, Conn2: conn2, Account: account, State: PlUnknown,
		loginAt: now, activeAt: now}
	w.byKey[key] = p
	if account != "" {
		w.byAcc[account] = p
	}
	w.order = append(w.order, p)
	return p
}

// RemovePlayer by connect 键。
func (w *WorldPlayers) RemovePlayer(conn1, conn2 uint32) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.removeLocked(persistKey(conn1, conn2))
}

// RemovePlayerByName 官方 RemovePlayerByName(wchar)。
func (w *WorldPlayers) RemovePlayerByName(name string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	p, ok := w.byName[name]
	if !ok {
		return false
	}
	return w.removeLocked(persistKey(p.Conn1, p.Conn2))
}

func (w *WorldPlayers) removeLocked(key uint64) bool {
	p, ok := w.byKey[key]
	if !ok {
		return false
	}
	delete(w.byKey, key)
	if p.Account != "" {
		delete(w.byAcc, p.Account)
	}
	if p.Name != "" {
		delete(w.byName, p.Name)
	}
	for i, q := range w.order {
		if q == p {
			w.order = append(w.order[:i], w.order[i+1:]...)
			break
		}
	}
	return true
}

// SetupPlayerName / CleanPlayerName 官方对（名字索引维护）。
func (w *WorldPlayers) SetupPlayerName(conn1, conn2 uint32, name string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	p, ok := w.byKey[persistKey(conn1, conn2)]
	if !ok {
		return false
	}
	if p.Name != "" {
		delete(w.byName, p.Name)
	}
	p.Name = name
	if name != "" {
		w.byName[name] = p
	}
	return true
}

// FindByConnect / FindByName / FindByAccount / FindByAccountId 查询族。
func (w *WorldPlayers) FindByConnect(conn1, conn2 uint32) *Player {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.byKey[persistKey(conn1, conn2)]
}

func (w *WorldPlayers) FindByName(name string) *Player {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.byName[name]
}

func (w *WorldPlayers) FindByAccount(acc string) *Player {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.byAcc[acc]
}

// Snapshot 有序遍历（GetPlayers 语义）。
func (w *WorldPlayers) Snapshot() []*Player {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]*Player, len(w.order))
	copy(out, w.order)
	return out
}

// Count 族（GetPlayerCount 等九计数的实现基准）。
func (w *WorldPlayers) Count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.order)
}

// CountBy 按状态计数。
func (w *WorldPlayers) CountBy(st PlayerState) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for _, p := range w.order {
		if p.State == st {
			n++
		}
	}
	return n
}

// CountActive Active 语义：Online+Offline+Saving+Leaving（在册）。
func (w *WorldPlayers) CountActive() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for _, p := range w.order {
		switch p.State {
		case PlOnline, PlOffline, PlSaving, PlLeaving:
			n++
		}
	}
	return n
}

// CountOnline / CountOffline / CountQueue 五计数（echo players）。
func (w *WorldPlayers) CountOnline() int  { return w.CountBy(PlOnline) }
func (w *WorldPlayers) CountOffline() int { return w.CountBy(PlOffline) }
func (w *WorldPlayers) CountQueue() int   { return w.CountBy(PlQueuing) }

// SetPlayerState / SetPlayerActive 状态维护（§八 SoloWorldPlayer.obj）。
func (w *WorldPlayers) SetPlayerState(p *Player, st PlayerState) {
	w.mu.Lock()
	p.State = st
	w.mu.Unlock()
}

func (w *WorldPlayers) SetPlayerActive(p *Player) {
	w.mu.Lock()
	p.activeAt = time.Now()
	w.mu.Unlock()
}
