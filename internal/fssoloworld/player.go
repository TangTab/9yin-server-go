package fssoloworld

import (
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
