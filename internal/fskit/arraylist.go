package fskit

import "sync/atomic"

// EntityID 是官方 PERSISTID 的 Go 等价（实体句柄）。0 = 空 ID。
type EntityID uint64

// ArrayList 复刻官方 fskit/arraylist.cpp 实体：命名子项树。
// 官方 CreateChild 走 Core::CreateEntity("ArrayList")，Go 侧由 Registry 发号。
type ArrayList struct {
	id       EntityID
	name     string
	hash     uint32
	children []*ArrayList
}

const ArrayListTypeName = "ArrayList"

// Registry 承担官方 Core 的实体发号/查号职责（ArrayList 子项按 id 引用）。
type Registry struct {
	next   atomic.Uint64
	byID   map[EntityID]*ArrayList
	byName map[string]*ArrayList // 顶层注册名（可选）
}

func NewRegistry() *Registry {
	return &Registry{byID: map[EntityID]*ArrayList{}, byName: map[string]*ArrayList{}}
}

// NewArrayList 建顶层实体（官方 EntCreator::Create → Core 注册）。
func (r *Registry) NewArrayList(name string) *ArrayList {
	a := &ArrayList{id: EntityID(r.next.Add(1)), name: name, hash: HashValueCase(name)}
	r.byID[a.id] = a
	return a
}

func (r *Registry) Get(id EntityID) *ArrayList { return r.byID[id] }

// CreateChild 官方 0x100490d0：创建子 ArrayList、SetName+SetHash（大小写敏感
// 131 哈希）、入列、返回其 id。失败路径（核心拒绝）Go 侧不发生。
func (a *ArrayList) CreateChild(r *Registry, name string) EntityID {
	child := &ArrayList{id: EntityID(r.next.Add(1)), name: name, hash: HashValueCase(name)}
	r.byID[child.id] = child
	a.children = append(a.children, child)
	return child.id
}

// AddChild 官方 0x100492f0：按 id 收养外部实体（查重：已在则失败）。
func (a *ArrayList) AddChild(r *Registry, id EntityID) bool {
	child := r.Get(id)
	if child == nil {
		return false
	}
	for _, c := range a.children {
		if c == child {
			return false
		}
	}
	a.children = append(a.children, child)
	return true
}

func (a *ArrayList) findChildIndex(name string) (int, bool) {
	h := HashValueCase(name)
	for i, c := range a.children {
		if c.hash == h && c.name == name {
			return i, true
		}
	}
	return 0, false
}

// RemoveChild 官方 0x10049460：按名删（大小写敏感）。
func (a *ArrayList) RemoveChild(name string) bool {
	i, ok := a.findChildIndex(name)
	if !ok {
		return false
	}
	a.children = append(a.children[:i], a.children[i+1:]...)
	return true
}

// RemoveChildByIndex 官方 0x10049520。
func (a *ArrayList) RemoveChildByIndex(i int) bool {
	if i < 0 || i >= len(a.children) {
		return false
	}
	a.children = append(a.children[:i], a.children[i+1:]...)
	return true
}

// ClearChild 官方 0x10049610。
func (a *ArrayList) ClearChild() bool {
	a.children = a.children[:0]
	return true
}

func (a *ArrayList) GetChildCount() int { return len(a.children) }

// GetChildList 官方 0x10049770：子名逐个 AddString。
func (a *ArrayList) GetChildList() []string {
	out := make([]string, 0, len(a.children))
	for _, c := range a.children {
		out = append(out, c.name)
	}
	return out
}

// FindChild 官方 0x10049840：按名查（大小写敏感）。
func (a *ArrayList) FindChild(name string) bool {
	_, ok := a.findChildIndex(name)
	return ok
}

// GetChild 官方 0x100498e0：按名取 id，miss → 0。
func (a *ArrayList) GetChild(name string) EntityID {
	if i, ok := a.findChildIndex(name); ok {
		return a.children[i].id
	}
	return 0
}

// GetChildByIndex 官方 0x100499b0：越界 → 0。
func (a *ArrayList) GetChildByIndex(i int) EntityID {
	if i < 0 || i >= len(a.children) {
		return 0
	}
	return a.children[i].id
}

// Name 属性 get/set（setter 同步 SetHash，官方 0x10048d80）。
func (a *ArrayList) SetName(name string) {
	a.name = name
	a.hash = HashValueCase(name)
}
func (a *ArrayList) GetName() string   { return a.name }
func (a *ArrayList) GetHash() uint32   { return a.hash }
