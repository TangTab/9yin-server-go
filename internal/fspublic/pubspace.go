package fspublic

import (
	"fmt"
	"sort"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// PubSpace 是 CPubSpace 等价：命名数据空间（name + store 表名 + 数据集）。
// vtable 定案（0x10126B0C）：[1]GetSpaceName [2]GetTableName [3]GetCanModify
// [4]FindPubData [5]GetPubDataCount [7]GetPubData [8]AddPubData [9]RemovePubData。
type PubSpace struct {
	Name      string // 空间名（"pubdata"/"guild"）
	TableName string // store 侧 DB 表名（"nx_pubdata"/"nx_guilds"）

	mu    sync.RWMutex
	datas []*PubData
	index map[string]int // name → 下标（官方 wchar TStringPod）
}

// NewPubSpace 官方 ctor（0x7C B）。
func NewPubSpace(name, table string) *PubSpace {
	return &PubSpace{Name: name, TableName: table, index: make(map[string]int)}
}

// GetCanModify 官方恒 1。
func (s *PubSpace) GetCanModify() bool { return true }

// Count 数据数（GetPubDataCount）。
func (s *PubSpace) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.datas)
}

// Find 按名查下标（FindIndex；-1=无）。
func (s *PubSpace) Find(name string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if i, ok := s.index[name]; ok {
		return i
	}
	return -1
}

// Get 按名取数据（FindPubData）。
func (s *PubSpace) Get(name string) (*PubData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	i, ok := s.index[name]
	if !ok {
		return nil, false
	}
	return s.datas[i], true
}

// GetByIndex 按下标取（GetPubData；MemberData 请求用 index）。
func (s *PubSpace) GetByIndex(i int) (*PubData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if i < 0 || i >= len(s.datas) {
		return nil, fmt.Errorf("(GamePublic::MemberData)data index error")
	}
	return s.datas[i], nil
}

// Add 建数据对象（CPubSpace::Add：内存对象 + uid 分配；不广播不持久化）。
func (s *PubSpace) Add(name, key string) (*PubData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.index[name]; ok {
		return nil, fmt.Errorf("(CPubData::Add)already exists: %s", name)
	}
	d := NewPubData(name, key)
	s.index[name] = len(s.datas)
	s.datas = append(s.datas, d)
	return d, nil
}

// AddPubData 官方 AddPubData 语义的数据面部分（广播/持久化由服务层
// OnAddPubData 钩子承接：member op2 推送 + store op 0x21）。
func (s *PubSpace) AddPubData(name, key string, uid string) (*PubData, error) {
	d, err := s.Add(name, key)
	if err != nil {
		return nil, err
	}
	if uid == "" {
		uid = CreateGUID()
	}
	d.UID = uid
	return d, nil
}

// Remove 删数据（RemovePubData 数据面）。
func (s *PubSpace) Remove(name string) (*PubData, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, ok := s.index[name]
	if !ok {
		return nil, false
	}
	d := s.datas[i]
	s.datas = append(s.datas[:i], s.datas[i+1:]...)
	delete(s.index, name)
	for n, idx := range s.index {
		if idx > i {
			s.index[n] = idx - 1
		}
	}
	return d, true
}

// Names 数据名清单（按下标序，镜像侧拉快照遍历用）。
func (s *PubSpace) Names() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.datas))
	for i, d := range s.datas {
		out[i] = d.Name
	}
	return out
}

// All 全量数据快照视图。
func (s *PubSpace) All() []*PubData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]*PubData(nil), s.datas...)
}

// Spaces 是空间注册表（GamePublic 的 vector<CPubSpace*> + TStringPod）。
type Spaces struct {
	mu      sync.RWMutex
	order   []*PubSpace
	byName  map[string]int
}

// NewSpaces 建注册表。
func NewSpaces() *Spaces {
	return &Spaces{byName: make(map[string]int)}
}

// Add 官方 AddPubSpace（重名拒绝 "space exists"）。
func (ss *Spaces) Add(name, table string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if _, ok := ss.byName[name]; ok {
		return fmt.Errorf("(GamePublic::AddPubSpace)space exists: %s", name)
	}
	ss.byName[name] = len(ss.order)
	ss.order = append(ss.order, NewPubSpace(name, table))
	return nil
}

// Find 官方 FindPubSpace（BOOL 语义）。
func (ss *Spaces) Find(name string) (*PubSpace, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	i, ok := ss.byName[name]
	if !ok {
		return nil, false
	}
	return ss.order[i], true
}

// Count 官方 GetPubSpaceCount。
func (ss *Spaces) Count() int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return len(ss.order)
}

// List 官方 GetPubSpaceList（注册序）。
func (ss *Spaces) List() []*PubSpace {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return append([]*PubSpace(nil), ss.order...)
}

// ListNames 空间名清单（排序稳定，日志面）。
func (ss *Spaces) ListNames() []string {
	names := make([]string, 0, ss.Count())
	for _, s := range ss.List() {
		names = append(names, s.Name)
	}
	sort.Strings(names)
	return names
}

// CreateGUID 官方 AddPubData 的 create_guid（fskit.GenUniqueName 同构：
// 32 位大写 hex v4 GUID）。
func CreateGUID() string { return fskit.GenUniqueName() }
