package fsserver

import (
	"sync"
	"time"
)

// PlayerState 玩家状态机 14 态（state@player+0xC，§四）。
type PlayerState int32

const (
	PStateUnknown PlayerState = iota
	PStateWaitBreak
	PStateLogging
	PStateValidate
	PStateQueuing
	PStateGetting
	PStateLogged
	PStateChoosing
	PStateCreating
	PStateDeleting
	PStateOnline
	PStateOffline
	PStateSaving
	PStateLeaving
)

// String 状态名（官方处理器名尾词）。
func (s PlayerState) String() string {
	switch s {
	case PStateUnknown:
		return "Unknown"
	case PStateWaitBreak:
		return "WaitBreak"
	case PStateLogging:
		return "Logging"
	case PStateValidate:
		return "Validate"
	case PStateQueuing:
		return "Queuing"
	case PStateGetting:
		return "Getting"
	case PStateLogged:
		return "Logged"
	case PStateChoosing:
		return "Choosing"
	case PStateCreating:
		return "Creating"
	case PStateDeleting:
		return "Deleting"
	case PStateOnline:
		return "Online"
	case PStateOffline:
		return "Offline"
	case PStateSaving:
		return "Saving"
	case PStateLeaving:
		return "Leaving"
	}
	return "PlayerState(?)"
}

// Player 复刻 player_t（0x170B，§4.1）。Go 侧以语义字段承载官方偏移。
type Player struct {
	Ident    uint32 // connect id（PERSISTID 首 4B）
	Serial   uint32 // PERSISTID 次 4B
	State    PlayerState
	EntryIdx uint32 // 所属 entry

	Account  string // +0x15C char[16]
	Password string // 登录帧解出（校验期驻留）
	RoleName string // 当前角色名（SetupPlayerName）
	RoleBlob []byte // store Roles 应答原文（SendPlayerRoles/Choosing 消费）
	SceneID  uint32 // 当前场景
	InGame   bool   // SetPlayerEntryGame/LeaveGame
	Matrix   bool   // 矩阵卡挑战已发

	LoginTime  time.Time // +0x40
	ActiveTime time.Time // +0x48
	SaveTime   time.Time // +0x50
}

// Players 复刻 CWorldPlayers：指针数组 + PERSISTID 哈希 + 空闲槽复用。
type Players struct {
	mu     sync.RWMutex
	list   []*Player // nil = 空闲槽
	free   []int
	byConn map[uint64]*Player // {ident,serial} → player
	byName map[string]*Player // 角色名（SetupPlayerName 登记）
	byAcct map[string]*Player // 账号（SetPlayerAccount 登记）
}

// NewPlayers …
func NewPlayers() *Players {
	return &Players{
		byConn: make(map[uint64]*Player),
		byName: make(map[string]*Player),
		byAcct: make(map[string]*Player),
	}
}

func pid(ident, serial uint32) uint64 { return uint64(ident)<<32 | uint64(serial) }

// AddPlayer 复刻 CWorldPlayers::AddPlayer（连接接入；同 connect 先移除）。
func (ps *Players) AddPlayer(ident, serial uint32, entryIdx uint32) *Player {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.removeLocked(ident, serial)
	var p = &Player{Ident: ident, Serial: serial, EntryIdx: entryIdx, LoginTime: time.Now()}
	p.ActiveTime, p.SaveTime = p.LoginTime, p.LoginTime
	idx := len(ps.list)
	if n := len(ps.free); n > 0 {
		idx = ps.free[n-1]
		ps.free = ps.free[:n-1]
		ps.list[idx] = p
	} else {
		ps.list = append(ps.list, p)
	}
	ps.byConn[pid(ident, serial)] = p
	return p
}

func (ps *Players) removeLocked(ident, serial uint32) bool {
	p, ok := ps.byConn[pid(ident, serial)]
	if !ok {
		return false
	}
	delete(ps.byConn, pid(ident, serial))
	if p.RoleName != "" {
		delete(ps.byName, p.RoleName)
	}
	if p.Account != "" && ps.byAcct[p.Account] == p {
		delete(ps.byAcct, p.Account)
	}
	for i, q := range ps.list {
		if q == p {
			ps.list[i] = nil
			ps.free = append(ps.free, i)
			break
		}
	}
	return true
}

// RemovePlayer …
func (ps *Players) RemovePlayer(ident, serial uint32) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.removeLocked(ident, serial)
}

// Count 玩家表规模（GetPlayerMax 语义：槽数）。
func (ps *Players) Count() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	n := 0
	for _, p := range ps.list {
		if p != nil {
			n++
		}
	}
	return n
}

// FindByConnect PERSISTID 查找。
func (ps *Players) FindByConnect(ident, serial uint32) *Player {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.byConn[pid(ident, serial)]
}

// FindByName 角色名查找（FindPlayerInGame 语义）。
func (ps *Players) FindByName(role string) *Player {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.byName[role]
}

// FindByAccount 账号查找（同账号唯一在线）。
func (ps *Players) FindByAccount(acct string) *Player {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.byAcct[acct]
}

// SetupPlayerName 角色名登记（进场/换名）。
func (ps *Players) SetupPlayerName(p *Player, role string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if p.RoleName != "" {
		delete(ps.byName, p.RoleName)
	}
	p.RoleName = role
	if role != "" {
		ps.byName[role] = p
	}
}

// CleanPlayerName 角色名摘除（离场）。
func (ps *Players) CleanPlayerName(p *Player) { ps.SetupPlayerName(p, "") }

// SetPlayerAccount 账号登记（校验通过后）。
func (ps *Players) SetPlayerAccount(p *Player, acct string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if p.Account != "" && ps.byAcct[p.Account] == p {
		delete(ps.byAcct, p.Account)
	}
	p.Account = acct
	if acct != "" {
		ps.byAcct[acct] = p
	}
}

// Each 遍历（回调内增删不安全——官方 WorldStateClearPlayers 批语义，
// Go 侧以快照遍历 + 二次收集实现）。
func (ps *Players) Each(fn func(*Player) bool) {
	ps.mu.RLock()
	snap := make([]*Player, 0, len(ps.list))
	for _, p := range ps.list {
		if p != nil {
			snap = append(snap, p)
		}
	}
	ps.mu.RUnlock()
	for _, p := range snap {
		if !fn(p) {
			return
		}
	}
}
