package fssoloworld

import "sync"

// NameUid 复刻 CNameUid（NameUid.obj，§九）：角色名(wchar)↔uid(char[33])
// 双向映射 + deleted 标志。用于建角名预占与 NameUid 同步。
type NameUid struct {
	mu     sync.Mutex
	byName map[string]*nameUidNode
	byUid  map[string]*nameUidNode
	count  int
}

type nameUidNode struct {
	name    string // UTF-8 呈现（官方 wchar）
	uid     string // 官方 char[33]（CreateGuid）
	deleted bool
}

// NewNameUid 桶 0x7F（ctor @0x10085480）。
func NewNameUid() *NameUid {
	return &NameUid{byName: map[string]*nameUidNode{}, byUid: map[string]*nameUidNode{}}
}

// Add 官方 Add(name, uid, deleted)：双向登记，重复 false。
func (n *NameUid) Add(name, uid string, deleted bool) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, dup := n.byName[name]; dup {
		return false
	}
	if _, dup := n.byUid[uid]; dup {
		return false
	}
	nd := &nameUidNode{name: name, uid: uid, deleted: deleted}
	n.byName[name] = nd
	n.byUid[uid] = nd
	n.count++
	return true
}

// Remove 官方 Remove(name, uid)：双侧都在才删。
func (n *NameUid) Remove(name, uid string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	nd, ok := n.byName[name]
	if !ok || nd.uid != uid {
		return false
	}
	delete(n.byName, name)
	delete(n.byUid, uid)
	n.count--
	return true
}

// RemoveName / RemoveUid 单侧删除变体。
func (n *NameUid) RemoveName(name string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	nd, ok := n.byName[name]
	if !ok {
		return false
	}
	delete(n.byName, name)
	delete(n.byUid, nd.uid)
	n.count--
	return true
}

func (n *NameUid) RemoveUid(uid string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	nd, ok := n.byUid[uid]
	if !ok {
		return false
	}
	delete(n.byUid, uid)
	delete(n.byName, nd.name)
	n.count--
	return true
}

// FindName / FindUid 存在性。
func (n *NameUid) FindName(name string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	_, ok := n.byName[name]
	return ok
}

func (n *NameUid) FindUid(uid string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	_, ok := n.byUid[uid]
	return ok
}

// GetUid / GetName / GetDeleted 查询族。
func (n *NameUid) GetUid(name string) (string, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	nd, ok := n.byName[name]
	if !ok {
		return "", false
	}
	return nd.uid, true
}

func (n *NameUid) GetName(uid string) (string, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	nd, ok := n.byUid[uid]
	if !ok {
		return "", false
	}
	return nd.name, true
}

func (n *NameUid) GetDeleted(name string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if nd, ok := n.byName[name]; ok {
		return nd.deleted
	}
	return false
}

// GetCount 数量。
func (n *NameUid) GetCount() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.count
}

// Snapshot 供 SyncRoleInfo 全量同步（StoreMsg RoleInfo 载荷构建用）。
func (n *NameUid) Snapshot() []struct {
	Name, Uid string
	Deleted   bool
} {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := make([]struct {
		Name, Uid string
		Deleted   bool
	}, 0, n.count)
	for _, nd := range n.byName {
		out = append(out, struct {
			Name, Uid string
			Deleted   bool
		}{nd.name, nd.uid, nd.deleted})
	}
	return out
}
