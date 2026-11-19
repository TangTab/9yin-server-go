package fsserver

import (
	"sync"
)

// NameUid 复刻 CNameUid：角色名 → {uid, account, deleted} 注册表
// （StoreMsgRoleInfo 全量重建、StoreMsgCreate/Delete 增删、SyncAllRoleInfo
// 广播源）。官方 find/add 皆有 bool 回报，重名拒绝。
type nameUidEntry struct {
	Name    string // 宽串（角色名）
	Account string // 窄串（所属账号）
	UID     int32
	Deleted bool
}

// NameUid …
type NameUid struct {
	mu   sync.RWMutex
	ents []nameUidEntry
	idx  map[string]int
}

// NewNameUid …
func NewNameUid() *NameUid {
	return &NameUid{idx: make(map[string]int)}
}

// Clear 全量重建起点（StoreMsgRoleInfo 头部动作）。
func (n *NameUid) Clear() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.ents = nil
	n.idx = make(map[string]int)
}

// Add 增名（重名拒绝——"(GameWorld::StoreMsgCreate)add name uid failed"）。
func (n *NameUid) Add(name, account string, uid int32) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, ok := n.idx[name]; ok {
		return false
	}
	n.idx[name] = len(n.ents)
	n.ents = append(n.ents, nameUidEntry{Name: name, Account: account, UID: uid})
	return true
}

// Remove 摘名。
func (n *NameUid) Remove(name string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	i, ok := n.idx[name]
	if !ok {
		return false
	}
	n.ents[i].Deleted = true
	delete(n.idx, name)
	return true
}

// Count …
func (n *NameUid) Count() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.ents)
}

// Snapshot 遍历（SyncAllRoleInfo 写序：Name/Account/UID/Deleted）。
func (n *NameUid) Snapshot(fn func(name, account string, uid int32, deleted bool)) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, e := range n.ents {
		fn(e.Name, e.Account, e.UID, e.Deleted)
	}
}
