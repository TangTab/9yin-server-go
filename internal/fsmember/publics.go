package fsmember

import (
	"fmt"
	"sort"
	"sync"
)

// PubRecord public 侧记录表镜像（列类型 + 行集）。
type PubRecord struct {
	Name  string
	Cols  []string // 列名
	Rows  [][]string
}

// PubData public 侧数据对象镜像：属性集 + 记录表集。
type PubData struct {
	Name  string
	Attrs map[string]string
	Recs  map[string]*PubRecord
}

// PubSpace public 命名空间（CPubSpace 等价）。
type PubSpace struct {
	Name  string
	Datas map[string]*PubData
}

// PublicSpaces 复刻 member 侧 public 数据镜像（§9 十四型的本地只读
// 副本）：space → data → {attr, rec}，官方断言串逐条对齐。
type PubSpaces struct {
	mu     sync.RWMutex
	spaces map[string]*PubSpace
}

// NewPublicSpaces 建镜像。
func NewPublicSpaces() *PubSpaces {
	return &PubSpaces{spaces: make(map[string]*PubSpace)}
}

func (p *PubSpaces) space(name string, create bool) (*PubSpace, error) {
	s, ok := p.spaces[name]
	if !ok {
		if !create {
			return nil, fmt.Errorf("(GameMember::%s)space not exists: %s", "PublicData", name)
		}
		s = &PubSpace{Name: name, Datas: make(map[string]*PubData)}
		p.spaces[name] = s
	}
	return s, nil
}

// DeclareSpaces 复刻 PublicSpaces(0x0)：空间声明清单；重名报
// "space name repeat"。
func (p *PubSpaces) DeclareSpaces(names []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	seen := map[string]bool{}
	for _, n := range names {
		if seen[n] {
			return fmt.Errorf("(GameMember::PublicSpaces)space name repeat: %s", n)
		}
		seen[n] = true
		if _, ok := p.spaces[n]; !ok {
			p.spaces[n] = &PubSpace{Name: n, Datas: make(map[string]*PubData)}
		}
	}
	return nil
}

// EnsureData 复刻 PublicData(0x1)：空间→数据对象建立。
func (p *PubSpaces) EnsureData(spaceName, dataName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	s, err := p.space(spaceName, false)
	if err != nil {
		return fmt.Errorf("(GameMember::PublicData)space not exists: %s", spaceName)
	}
	if _, ok := s.Datas[dataName]; !ok {
		s.Datas[dataName] = &PubData{Name: dataName,
			Attrs: make(map[string]string), Recs: make(map[string]*PubRecord)}
	}
	return nil
}

// AddData 复刻 PublicAdd(0x2)：已存在报 "data already exists"。
func (p *PubSpaces) AddData(spaceName, dataName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	s, err := p.space(spaceName, false)
	if err != nil {
		return fmt.Errorf("(GameMember::PublicAdd)space not exists: %s", spaceName)
	}
	if _, ok := s.Datas[dataName]; ok {
		return fmt.Errorf("(GameMember::PublicAdd)data already exists: %s", dataName)
	}
	s.Datas[dataName] = &PubData{Name: dataName,
		Attrs: make(map[string]string), Recs: make(map[string]*PubRecord)}
	return nil
}

// RemoveData 复刻 PublicRemove(0x3)。
func (p *PubSpaces) RemoveData(spaceName, dataName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	s, err := p.space(spaceName, false)
	if err != nil {
		return fmt.Errorf("(GameMember::PublicRemove)space not exists: %s", spaceName)
	}
	if _, ok := s.Datas[dataName]; !ok {
		return fmt.Errorf("(GameMember::PublicRemove)remove data failed: %s", dataName)
	}
	delete(s.Datas, dataName)
	return nil
}

func (p *PubSpaces) data(spaceName, dataName, who string) (*PubData, error) {
	s, err := p.space(spaceName, false)
	if err != nil {
		return nil, fmt.Errorf("(GameMember::%s)space not exists: %s", who, spaceName)
	}
	d, ok := s.Datas[dataName]
	if !ok {
		return nil, fmt.Errorf("(GameMember::%s)data not exists: %s", who, dataName)
	}
	return d, nil
}

// SetAttr 复刻 PublicAddAttr(0x4)；DelAttr 复刻 PublicRemoveAttr(0x5)。
func (p *PubSpaces) SetAttr(spaceName, dataName, key, value string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	d, err := p.data(spaceName, dataName, "PublicAddAttr")
	if err != nil {
		return err
	}
	d.Attrs[key] = value
	return nil
}

// DelAttr 删除属性；不存在报 "remove attr failed"。
func (p *PubSpaces) DelAttr(spaceName, dataName, key string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	d, err := p.data(spaceName, dataName, "PublicRemoveAttr")
	if err != nil {
		return err
	}
	if _, ok := d.Attrs[key]; !ok {
		return fmt.Errorf("(GameMember::PublicRemoveAttr)remove attr failed: %s", key)
	}
	delete(d.Attrs, key)
	return nil
}

// AddRec 复刻 PublicAddRec(0x6)；DelRec 复刻 PublicRemoveRec(0x7)。
func (p *PubSpaces) AddRec(spaceName, dataName, recName string, cols []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	d, err := p.data(spaceName, dataName, "PublicAddRec")
	if err != nil {
		return err
	}
	if _, ok := d.Recs[recName]; ok {
		return fmt.Errorf("(GameMember::PublicAddRec)add rec failed: %s", recName)
	}
	d.Recs[recName] = &PubRecord{Name: recName, Cols: append([]string(nil), cols...)}
	return nil
}

// DelRec 删除记录表。
func (p *PubSpaces) DelRec(spaceName, dataName, recName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	d, err := p.data(spaceName, dataName, "PublicRemoveRec")
	if err != nil {
		return err
	}
	if _, ok := d.Recs[recName]; !ok {
		return fmt.Errorf("(GameMember::PublicRemoveRec)remove rec failed: %s", recName)
	}
	delete(d.Recs, recName)
	return nil
}

func (p *PubSpaces) rec(spaceName, dataName, recName, who string) (*PubData, *PubRecord, error) {
	d, err := p.data(spaceName, dataName, who)
	if err != nil {
		return nil, nil, err
	}
	rc, ok := d.Recs[recName]
	if !ok {
		return nil, nil, fmt.Errorf("(GameMember::%s)rec not exists: %s", who, recName)
	}
	return d, rc, nil
}

// RecAddRow 复刻 PublicRecAddRow(0x8)；RecRemoveRow(0x9)；
// RecClear(0xA)；RecChange(0xB)；RecSetRow(0xC)。
func (p *PubSpaces) RecAddRow(spaceName, dataName, recName string, row []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, rc, err := p.rec(spaceName, dataName, recName, "PublicRecAddRow")
	if err != nil {
		return err
	}
	if len(row) != len(rc.Cols) {
		return fmt.Errorf("(GameMember::PublicRecAddRow)rec load row value failed: %s", recName)
	}
	rc.Rows = append(rc.Rows, append([]string(nil), row...))
	return nil
}

// RecRemoveRow 按下标删行。
func (p *PubSpaces) RecRemoveRow(spaceName, dataName, recName string, index int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, rc, err := p.rec(spaceName, dataName, recName, "PublicRecRemoveRow")
	if err != nil {
		return err
	}
	if index < 0 || index >= len(rc.Rows) {
		return fmt.Errorf("(GameMember::PublicRecRemoveRow)rec remove row failed: %d", index)
	}
	rc.Rows = append(rc.Rows[:index], rc.Rows[index+1:]...)
	return nil
}

// RecClear 清行。
func (p *PubSpaces) RecClear(spaceName, dataName, recName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, rc, err := p.rec(spaceName, dataName, recName, "PublicRecClear")
	if err != nil {
		return err
	}
	rc.Rows = nil
	return nil
}

// RecChange 改单格。
func (p *PubSpaces) RecChange(spaceName, dataName, recName string, row, col int, value string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, rc, err := p.rec(spaceName, dataName, recName, "PublicRecChange")
	if err != nil {
		return err
	}
	if row < 0 || row >= len(rc.Rows) || col < 0 || col >= len(rc.Cols) {
		return fmt.Errorf("(GameMember::PublicRecChange)rec data error: %d,%d", row, col)
	}
	rc.Rows[row][col] = value
	return nil
}

// RecSetRow 整行覆写。
func (p *PubSpaces) RecSetRow(spaceName, dataName, recName string, row int, values []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, rc, err := p.rec(spaceName, dataName, recName, "PublicRecSetRow")
	if err != nil {
		return err
	}
	if row < 0 || row >= len(rc.Rows) || len(values) != len(rc.Cols) {
		return fmt.Errorf("(GameMember::PublicRecSetRow)rec load row failed: %d", row)
	}
	rc.Rows[row] = append([]string(nil), values...)
	return nil
}

// SpaceCount/SpaceList 官方 GetPubSpaceCount/GetPubSpaceList。
func (p *PubSpaces) SpaceCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.spaces)
}

// SpaceList 空间名清单（排序稳定）。
func (p *PubSpaces) SpaceList() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]string, 0, len(p.spaces))
	for n := range p.spaces {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// FindSpace 官方 FindPubSpace。
func (p *PubSpaces) FindSpace(name string) (*PubSpace, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	s, ok := p.spaces[name]
	return s, ok
}
