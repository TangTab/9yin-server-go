package fxcore

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/ecs"
	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// EntityID 框架实体句柄（官方 PERSISTID {u32 id; u32 serial} 同构，
// 复用 ecs.EntityID 形态：代际防悬挂）。NullEntity=0 永不签发。
type EntityID = ecs.EntityID

// PropDecl 对齐官方 IEntProperty 链节点（名 + 类型 + 读写面）。
type PropDecl struct {
	Name string
	Type string // "string"/"int"/"float"/"bool"/"widestr"…
	Get  bool   // 可读
	Set  bool   // 可写
}

// MethodDecl 对齐官方 IEntMethod 链节点（名 + 参数/返回元数据 + mid）。
type MethodDecl struct {
	Name       string
	ParamCount int
	Ret        bool
	Call       func(e *Entity, args *fskit.VarList, result *fskit.VarList) bool
}

// EntInfo 复刻官方 CEntInfo：实体类型描述（creator + 继承链 + 属性/方法链）。
// IsKindOf 沿 ParentName 上溯（官方 IsKindSameSpace 另校验域）。
type EntInfo struct {
	TypeName   string
	ParentName string // 继承（IsKindOf 链）；根类型为空
	Space      string // 模块域（官方 FxModule_GetSpace）
	Props      []PropDecl
	Methods    []MethodDecl
	New        func(args *fskit.VarList) any // 官方 IEntCreator::Create

	// lookup 由 EntityStore.AddCreator 注入（IsKindOf 上溯查表）。
	lookup func(name string) (*EntInfo, bool)
}

// IsKindOf 官方类型判断（含自身；沿父链上溯）。
func (info *EntInfo) IsKindOf(kind string) bool {
	for cur := info; cur != nil; {
		if strings.EqualFold(cur.TypeName, kind) {
			return true
		}
		if cur.ParentName == "" || strings.EqualFold(cur.ParentName, cur.TypeName) {
			return false
		}
		parent, ok := cur.lookup(cur.ParentName)
		if !ok {
			return false
		}
		cur = parent
	}
	return false
}

// Entity 复刻官方 IEntity 运行态：命名实体（CreateEntity 按类型名构造，
// 脚本/逻辑可绑定，删除走标记位——官方 SetDeleted 延迟销毁）。
type Entity struct {
	ID   EntityID
	Name string // 唯一名（官方 CreateEntity 后 SetName；查找键）
	Info *EntInfo

	ScriptName string // 官方 BindScript
	LogicName  string // 官方 BindLogic
	Callbacks  map[string]string

	userData    any
	deleted     bool
	canDelByScr bool
	props       map[string]any
}

// Deleted 官方 GetDeleted。
func (e *Entity) Deleted() bool { return e.deleted }

// CanDelByScript 官方标记（nx_destroy 只能删除此标记实体）。
func (e *Entity) CanDelByScript() bool { return e.canDelByScr }

// UserData 宿主挂载面（官方脚本 userdata 等价）。
func (e *Entity) UserData() any         { return e.userData }
func (e *Entity) SetUserData(v any)     { e.userData = v }

// EntityStore 复刻 CEntManager + CEntFactory + CEntInfoList：
// 类型注册表（AddCreator）、实体表（命名查找/前缀查找）、帧执行表、
// 单例表（SetSingle/GetSingle）。
type EntityStore struct {
	mu sync.Mutex

	creators map[string]*EntInfo // key=小写类型名
	ents     map[EntityID]*Entity
	byName   map[string]EntityID // key=小写实体名

	nextID   uint32
	gen      map[uint32]uint32 // index→generation（PERSISTID serial）
	deleted  []EntityID        // 官方延迟销毁队列（Destroy 于帧尾）

	frameExecutes map[string]FrameExecute
	frameOrder    []string
	singles       map[string]EntityID // key=小写类型名

	// lookupPool IsKindOf 上溯的查表注入。
	lookupPool func(name string) (*EntInfo, bool)
}

// FrameExecute 帧执行项（官方 AddFrameExecute(name, proc)；
// 定步泵每步调用，注册名可移除）。
type FrameExecute func(core *Core, step float64)

// NewEntityStore 创建空表。
func NewEntityStore() *EntityStore {
	s := &EntityStore{
		creators:      make(map[string]*EntInfo),
		ents:          make(map[EntityID]*Entity),
		byName:        make(map[string]EntityID),
		gen:           make(map[uint32]uint32),
		frameExecutes: make(map[string]FrameExecute),
		singles:       make(map[string]EntityID),
	}
	s.lookupPool = func(name string) (*EntInfo, bool) {
		info, ok := s.creators[strings.ToLower(name)]
		return info, ok
	}
	return s
}

// AddCreator 复刻 CEntManager::AddCreator：类型注册
//（重名=官方 "entity name repeat" 拒收）。
func (s *EntityStore) AddCreator(info *EntInfo) error {
	if info == nil || info.TypeName == "" {
		return fmt.Errorf("(CEntManager::AddCreator)empty entity name")
	}
	if info.New == nil {
		return fmt.Errorf("(CEntManager::AddCreator)creator nil: %s", info.TypeName)
	}
	key := strings.ToLower(info.TypeName)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.creators[key]; dup {
		return fmt.Errorf("(CEntManager::AddCreator)entity name repeat: %s", info.TypeName)
	}
	info.lookup = s.lookupPool
	s.creators[key] = info
	return nil
}

// FindInfo 类型查找（官方 CEntFactory::Find 前置）。
func (s *EntityStore) FindInfo(typeName string) (*EntInfo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, ok := s.creators[strings.ToLower(typeName)]
	return info, ok
}

// Create 复刻 CCore::CreateEntity(type[, args])：按类型名构造，
// userData=creator 产物。名重复/类型缺失返回空句柄。
func (s *EntityStore) Create(typeName, name string, args *fskit.VarList) (*Entity, error) {
	info, ok := s.FindInfo(typeName)
	if !ok {
		return nil, fmt.Errorf("(CEntFactory::Create)class %s not found", typeName)
	}
	if name != "" {
		s.mu.Lock()
		_, dup := s.byName[strings.ToLower(name)]
		s.mu.Unlock()
		if dup {
			return nil, fmt.Errorf("(CEntFactory::Create)entity name repeat: %s", name)
		}
	}
	body := info.New(args)
	id := s.allocID()
	e := &Entity{
		ID:           id,
		Name:         name,
		Info:         info,
		Callbacks:    make(map[string]string),
		canDelByScr:  false,
		props:        make(map[string]any),
	}
	e.userData = body
	s.mu.Lock()
	s.ents[id] = e
	if name != "" {
		s.byName[strings.ToLower(name)] = id
	}
	s.mu.Unlock()
	return e, nil
}

func (s *EntityStore) allocID() EntityID {
	s.nextID++
	idx := s.nextID
	gen := s.gen[idx] + 1
	s.gen[idx] = gen
	return EntityID(uint64(idx)&0xFFFFFFFF | uint64(gen)<<32)
}

// Lookup 复刻 CCore::LookupEntity(name)（大小写不敏感，官方哈希方言）。
func (s *EntityStore) Lookup(name string) *Entity {
	s.mu.Lock()
	id, ok := s.byName[strings.ToLower(name)]
	var e *Entity
	if ok {
		e = s.ents[id]
	}
	s.mu.Unlock()
	return e
}

// LookupMore 复刻 CCore::LookupEntityMore(prefix, out)：前缀匹配计数回填。
func (s *EntityStore) LookupMore(prefix string, out *[]string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	names := make([]string, 0, len(s.byName))
	for lower, id := range s.byName {
		e := s.ents[id]
		if e == nil || e.deleted {
			continue
		}
		if prefix == "" || strings.HasPrefix(lower, strings.ToLower(prefix)) {
			names = append(names, e.Name)
			n++
		}
	}
	sort.Strings(names)
	if out != nil {
		*out = append(*out, names...)
	}
	return n
}

// GetByID 句柄取实体（代际安全：悬挂句柄 nil）。
func (s *EntityStore) GetByID(id EntityID) *Entity {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.ents[id]
	if e == nil || e.deleted {
		return nil
	}
	return e
}

// SetDeleted 复刻官方 SetDeleted：标记删除（帧尾 Destroy）。
func (s *EntityStore) SetDeleted(e *Entity) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e == nil || e.deleted {
		return
	}
	e.deleted = true
	s.deleted = append(s.deleted, e.ID)
	if e.Name != "" {
		delete(s.byName, strings.ToLower(e.Name))
	}
}

// SetCanDelByScript 官方标记位。
func (s *EntityStore) SetCanDelByScript(e *Entity, can bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e.canDelByScr = can
}

// FlushDeleted 帧尾销毁（官方 Destroy 队列消费；泵 Async 二遍后调用）。
// 返回本帧销毁数。
func (s *EntityStore) FlushDeleted() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, id := range s.deleted {
		if e := s.ents[id]; e != nil {
			delete(s.ents, id)
			for k, v := range s.singles {
				if v == id {
					delete(s.singles, k)
				}
			}
			n++
		}
	}
	s.deleted = s.deleted[:0]
	return n
}

// Count 存活实体数。
func (s *EntityStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, e := range s.ents {
		if !e.deleted {
			n++
		}
	}
	return n
}

// Entities 全量快照（Dump 面）。
func (s *EntityStore) Entities() []*Entity {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Entity, 0, len(s.ents))
	for _, e := range s.ents {
		if !e.deleted {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// SetSingle/GetSingle 官方单例表（按类型名）。
func (s *EntityStore) SetSingle(e *Entity) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.singles[strings.ToLower(e.Info.TypeName)] = e.ID
}

func (s *EntityStore) GetSingle(typeName string) *Entity {
	s.mu.Lock()
	id, ok := s.singles[strings.ToLower(typeName)]
	var e *Entity
	if ok {
		e = s.ents[id]
	}
	s.mu.Unlock()
	if e == nil || e.deleted {
		return nil
	}
	return e
}

// AddFrameExecute 复刻 CEntManager::AddFrameExecute(name, proc)。
func (s *EntityStore) AddFrameExecute(name string, fn FrameExecute) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.frameExecutes[name]; !exists {
		s.frameOrder = append(s.frameOrder, name)
	}
	s.frameExecutes[name] = fn
}

// RemoveFrameExecute 官方移除。
func (s *EntityStore) RemoveFrameExecute(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.frameExecutes, name)
	for i, n := range s.frameOrder {
		if n == name {
			s.frameOrder = append(s.frameOrder[:i], s.frameOrder[i+1:]...)
			break
		}
	}
}

// executeFrames 定步执行（注册序稳定；官方 TArrayPod 顺序语义）。
func (s *EntityStore) executeFrames(core *Core, step float64) {
	s.mu.Lock()
	names := make([]string, len(s.frameOrder))
	copy(names, s.frameOrder)
	fns := make([]FrameExecute, 0, len(names))
	for _, n := range names {
		if fn, ok := s.frameExecutes[n]; ok {
			fns = append(fns, fn)
		}
	}
	s.mu.Unlock()
	for _, fn := range fns {
		fn(core, step)
	}
}
