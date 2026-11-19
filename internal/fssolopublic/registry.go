package fssolopublic

import (
	"fmt"
	"sync"
)

// NameUid 复刻 CNameUid（nameuid.cpp，ctor(0x7f)）：wchar 角色名 ↔
// char UID 双向哈希索引，由 world 广播维护（WorldRoleInfo 全量 /
// RoleAdd/Remove/Recreate 增量）。装载完成门 = SoloPublic+0x176。
type NameUid struct {
	mu      sync.RWMutex
	byName  map[string]string // wchar 名 → uid
	byUid   map[string]string // uid → wchar 名
	deleted map[string]bool   // 名字墓碑（Remove 后重建判定）
}

func NewNameUid() *NameUid {
	return &NameUid{
		byName:  map[string]string{},
		byUid:   map[string]string{},
		deleted: map[string]bool{},
	}
}

// Add 官方 CNameUid::Add(wname, uid, keepDeleted)。
func (n *NameUid) Add(name, uid string) bool {
	if name == "" || uid == "" {
		return false
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.RemoveLocked(name, uid)
	n.byName[name] = uid
	n.byUid[uid] = name
	delete(n.deleted, name)
	return true
}

// Remove 官方 CNameUid::Remove(wname, uid)。
func (n *NameUid) Remove(name, uid string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.RemoveLocked(name, uid)
}

func (n *NameUid) RemoveLocked(name, uid string) bool {
	ok := false
	if cur, had := n.byName[name]; had && (uid == "" || cur == uid) {
		delete(n.byName, name)
		delete(n.byUid, cur)
		n.deleted[name] = true
		ok = true
	}
	if uid != "" {
		if cur, had := n.byUid[uid]; had && (name == "" || cur == name) {
			delete(n.byUid, uid)
			delete(n.byName, cur)
			n.deleted[cur] = true
			ok = true
		}
	}
	return ok
}

// Clear 官方 Clear（WorldRoleInfo 全量重灌前置）。
func (n *NameUid) Clear() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.byName = map[string]string{}
	n.byUid = map[string]string{}
	n.deleted = map[string]bool{}
}

// GetUid / GetName / GetCount / GetDeleted 官方查询面。
func (n *NameUid) GetUid(name string) (string, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	uid, ok := n.byName[name]
	return uid, ok
}

func (n *NameUid) GetName(uid string) (string, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	name, ok := n.byUid[uid]
	return name, ok
}

func (n *NameUid) GetCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.byName)
}

func (n *NameUid) GetDeleted(name string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.deleted[name]
}

// Player 是 CPubPlayers 的 player_t（world 6/7/8/9 型维护）。
type Player struct {
	MemberID uint32
	SceneID  uint32
	Ident    uint32
	Name     string // wstr
	Account  string
	Online   bool
}

// PubPlayers 复刻 CPubPlayers（pubplayers.cpp）：在线/离线玩家镜像，
// member↔scene 归属（CommandByName/SendCustom* 路由依据）。
type PubPlayers struct {
	mu      sync.RWMutex
	players map[string]*Player // key = wname
}

func NewPubPlayers() *PubPlayers {
	return &PubPlayers{players: map[string]*Player{}}
}

// AddPlayer 官方 AddPlayer(member, scene, ident, ?, wname, account)。
func (p *PubPlayers) AddPlayer(memberID, sceneID, ident uint32, name, account string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if old, ok := p.players[name]; ok && old.Name == name {
		old.MemberID, old.SceneID, old.Ident = memberID, sceneID, ident
		old.Account = account
		old.Online = true
		return
	}
	p.players[name] = &Player{
		MemberID: memberID, SceneID: sceneID, Ident: ident,
		Name: name, Account: account, Online: true,
	}
}

// RemovePlayer 官方 RemovePlayer(member, scene, wname)。
func (p *PubPlayers) RemovePlayer(memberID, sceneID uint32, name string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if pl, ok := p.players[name]; ok {
		pl.Online = false
		pl.MemberID, pl.SceneID = memberID, sceneID
		return true
	}
	return false
}

// CleanPlayerConnect 官方 WorldPlayerChange（换线清理归属）。
func (p *PubPlayers) CleanPlayerConnect(name string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if pl, ok := p.players[name]; ok {
		pl.SceneID = 0
		return true
	}
	return false
}

// GetPlayer / FindByName / FindByAccount / Count 官方查询面。
func (p *PubPlayers) GetPlayer(name string) (*Player, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	pl, ok := p.players[name]
	if !ok {
		return nil, false
	}
	cp := *pl
	return &cp, true
}

func (p *PubPlayers) FindByAccount(account string) (*Player, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, pl := range p.players {
		if pl.Account == account {
			cp := *pl
			return &cp, true
		}
	}
	return nil, false
}

func (p *PubPlayers) Count() (total, online int) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, pl := range p.players {
		if pl.Online {
			online++
		}
	}
	return len(p.players), online
}

// Scene 是 CPubScenes 的 scene_t（world 2/3/4 型维护）。
type Scene struct {
	ID       uint32
	MemberID uint32
	Ident    uint32
	Clone    bool
	Name     string
	ResName  string
}

// PubScenes 复刻 CPubScenes（pubscenes.cpp）：普通/克隆场景注册表。
type PubScenes struct {
	mu     sync.RWMutex
	scenes map[uint32]*Scene
}

func NewPubScenes() *PubScenes {
	return &PubScenes{scenes: map[uint32]*Scene{}}
}

// AddScene 官方 AddScene(sceneID, memberID, ident, clone, name, resName)：
// 已存在则更新（幂等，WorldSceneAll 重灌安全）。
func (s *PubScenes) AddScene(sceneID, memberID, ident, clone uint32, name, resName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sc, ok := s.scenes[sceneID]; ok {
		sc.MemberID, sc.Ident, sc.Clone = memberID, ident, clone != 0
		sc.Name, sc.ResName = name, resName
		return
	}
	s.scenes[sceneID] = &Scene{
		ID: sceneID, MemberID: memberID, Ident: ident,
		Clone: clone != 0, Name: name, ResName: resName,
	}
}

// RemoveScene 官方 RemoveScene(sceneID)。
func (s *PubScenes) RemoveScene(sceneID uint32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.scenes[sceneID]; !ok {
		return false
	}
	delete(s.scenes, sceneID)
	return true
}

// GetScene / GetSceneMemberId / Count 官方查询面。
func (s *PubScenes) GetScene(sceneID uint32) (*Scene, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.scenes[sceneID]
	if !ok {
		return nil, false
	}
	cp := *sc
	return &cp, true
}

func (s *PubScenes) GetSceneMemberId(sceneID uint32) (uint32, error) {
	sc, ok := s.GetScene(sceneID)
	if !ok {
		return 0, fmt.Errorf("scene %d not exists", sceneID)
	}
	return sc.MemberID, nil
}

func (s *PubScenes) Count() (normal, clone int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sc := range s.scenes {
		if sc.Clone {
			clone++
		} else {
			normal++
		}
	}
	return
}
