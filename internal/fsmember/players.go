package fsmember

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Player 玩家注册表条目（CMemberPlayers::player_t 等价）。
type Player struct {
	Name    string // 角色名（宽字符语义，Go 存 UTF-8；线上按 UTF-16 定长编解码）
	Account string // 账号
	SceneID uint32 // 所在场景
	Ident   uint32 // entry 侧客户端 ident（conn 标识）
	Serial  uint32 // entry 侧客户端 serial（entryIdx<<16|序号）
	Online  bool   // 在线（CleanPlayerConnect 置离线）
}

// PlayerRegistry 复刻 CMemberPlayers：AddPlayer/RemovePlayer/GetPlayer/
// FindByName/FindByAccount/ChangePlayerSceneId/CleanPlayerConnect。
// 索引同官方：名字（wchar）→ 槽、账号（char）→ 槽、双向同步。
type PlayerRegistry struct {
	mu      sync.RWMutex
	byName  map[string]*Player
	byAcct  map[string]*Player
	players map[string]*Player // name → player（主存）
}

// NewPlayerRegistry 建表。
func NewPlayerRegistry() *PlayerRegistry {
	return &PlayerRegistry{
		byName:  make(map[string]*Player),
		byAcct:  make(map[string]*Player),
		players: make(map[string]*Player),
	}
}

// AddPlayer 复刻 CMemberPlayers::AddPlayer(sceneID, ident, serial, name,
// account)；重名/重账号拒绝（官方 player_t 单槽语义）。
func (r *PlayerRegistry) AddPlayer(sceneID uint32, ident, serial uint32, name, account string) error {
	if name == "" {
		return fmt.Errorf("(CMemberPlayers::AddPlayer)name empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.byName[name]; dup {
		return fmt.Errorf("(CMemberPlayers::AddPlayer)name repeat: %s", name)
	}
	if account != "" {
		if _, dup := r.byAcct[account]; dup {
			return fmt.Errorf("(CMemberPlayers::AddPlayer)account repeat: %s", account)
		}
	}
	p := &Player{Name: name, Account: account, SceneID: sceneID, Ident: ident, Serial: serial, Online: true}
	r.players[name] = p
	r.byName[name] = p
	if account != "" {
		r.byAcct[account] = p
	}
	return nil
}

// RemovePlayer 复刻 RemovePlayer(sceneID, name)：查无此人报错同官方
// "(GameMember::CommandPlayerRemove)player not found"。
func (r *PlayerRegistry) RemovePlayer(sceneID uint32, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.players[name]
	if !ok {
		return fmt.Errorf("(GameMember::CommandPlayerRemove)player not found: %s", name)
	}
	if p.SceneID != sceneID && sceneID != 0 {
		return fmt.Errorf("(CMemberPlayers::RemovePlayer)scene mismatch: %d != %d", p.SceneID, sceneID)
	}
	delete(r.players, name)
	delete(r.byName, name)
	if p.Account != "" {
		delete(r.byAcct, p.Account)
	}
	return nil
}

// GetPlayer 复刻 GetPlayer(sceneID, ?, name)。
func (r *PlayerRegistry) GetPlayer(sceneID uint32, name string) (Player, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.players[name]
	if !ok || (p.SceneID != sceneID && sceneID != 0) {
		return Player{}, false
	}
	return *p, true
}

// FindByName 按名查。
func (r *PlayerRegistry) FindByName(name string) (Player, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.players[name]
	if !ok {
		return Player{}, false
	}
	return *p, true
}

// FindByAccount 按账号查。
func (r *PlayerRegistry) FindByAccount(account string) (Player, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.byAcct[account]
	if !ok {
		return Player{}, false
	}
	return *p, true
}

// ChangePlayerSceneId 复刻 ChangePlayerSceneId(player, sceneID)。
func (r *PlayerRegistry) ChangePlayerSceneId(name string, sceneID uint32) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.players[name]
	if !ok {
		return fmt.Errorf("(CMemberPlayers::ChangePlayerSceneId)player not found: %s", name)
	}
	p.SceneID = sceneID
	return nil
}

// CleanPlayerConnect 复刻 CleanPlayerConnect(name)：断线清在线标记。
func (r *PlayerRegistry) CleanPlayerConnect(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.players[name]
	if !ok {
		return fmt.Errorf("(CMemberPlayers::CleanPlayerConnect)player not found: %s", name)
	}
	p.Online = false
	return nil
}

// SetOnline entry_forw 3（set_client_online）后置在线。
func (r *PlayerRegistry) SetOnline(ident, serial uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.players {
		if p.Ident == ident && p.Serial == serial {
			p.Online = true
			return
		}
	}
}

// Count 官方 GetPlayerCount。
func (r *PlayerRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.players)
}

// CountOnline 在线数（场景/上报用）。
func (r *PlayerRegistry) CountOnline() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := 0
	for _, p := range r.players {
		if p.Online {
			n++
		}
	}
	return n
}

// ScenePlayers 场景在场名单（GetScenePlayerList）。
func (r *PlayerRegistry) ScenePlayers(sceneID uint32) []Player {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Player
	for _, p := range r.players {
		if p.SceneID == sceneID {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Dump 复刻 CMemberPlayers::Dump（all_player%d.txt 行格式）。
func (r *PlayerRegistry) Dump() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var b strings.Builder
	for _, p := range r.Snapshot() {
		fmt.Fprintf(&b, "%s\t%s\tscene %d\tident %#x serial %#x online %v\n",
			p.Name, p.Account, p.SceneID, p.Ident, p.Serial, p.Online)
	}
	return b.String()
}

// Snapshot 全量（DumpPlayers/PlayerAll 上报源）。
func (r *PlayerRegistry) Snapshot() []Player {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Player, 0, len(r.players))
	for _, p := range r.players {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
