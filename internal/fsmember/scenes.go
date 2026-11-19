package fsmember

import (
	"fmt"
	"sort"
	"sync"
)

// SceneType 场景型别（官方 prototype/clone 二分，§6）。
type SceneType uint32

const (
	ScenePrototype SceneType = iota // 原型场景（静态表创建）
	SceneClone                      // 克隆场景（副本类，动态复制原型）
)

// Scene 场景注册表条目。
type Scene struct {
	ID     uint32
	Type   SceneType
	Name   string   // 场景名（创建器名/配置名）
	Proto  uint32   // clone 型的原型场景 id（prototype 型为 0）
	Active bool     // 场景是否已激活（Load 完成）
	MaxPlayers int  // 配额（CanCreateScene 依据）
	CurPlayers int  // 当前在场（DeleteClone 门）
}

// String 场景摘要。
func (s Scene) String() string {
	return fmt.Sprintf("scene %d type %d name %q players %d/%d", s.ID, s.Type, s.Name, s.CurPlayers, s.MaxPlayers)
}

// SceneRegistry 复刻 CMemberScenes：场景注册表 + 官方生命周期断言
//（CreateScene/CloneScene/DeleteClone/CanCreateScene 错误串逐条对齐）。
type SceneRegistry struct {
	mu     sync.RWMutex
	scenes map[uint32]*Scene
}

// NewSceneRegistry 建表。
func NewSceneRegistry() *SceneRegistry {
	return &SceneRegistry{scenes: make(map[uint32]*Scene)}
}

// CreateScene 复刻 GameMember::CreateScene：原型/静态场景创建。
// 断言序：no scene creator → scene creator type error → scene load error →
// scene initialize error（空名对应 no scene creator）。
func (r *SceneRegistry) CreateScene(id uint32, typ SceneType, name string, maxPlayers int) (*Scene, error) {
	if name == "" {
		return nil, fmt.Errorf("(GameMember::CreateScene)no scene creator")
	}
	if typ != ScenePrototype && typ != SceneClone {
		return nil, fmt.Errorf("(GameMember::CreateScene)scene creator type error")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.scenes[id]; ok && old.Active {
		return nil, fmt.Errorf("(GameMember::CreateScene)scene initialize error: scene %d exists", id)
	}
	s := &Scene{ID: id, Type: typ, Name: name, MaxPlayers: maxPlayers, Active: true}
	r.scenes[id] = s
	return s, nil
}

// CloneScene 复刻 GameMember::CloneScene：断言序 prototype not found →
// prototype not created → prototype error → scene invalid → create scene
// failed。
func (r *SceneRegistry) CloneScene(protoID, cloneID uint32, name string, maxPlayers int) (*Scene, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	proto, ok := r.scenes[protoID]
	if !ok {
		return nil, fmt.Errorf("(GameMember::CloneScene)prototype not found: %d", protoID)
	}
	if !proto.Active {
		return nil, fmt.Errorf("(GameMember::CloneScene)prototype not created: %d", protoID)
	}
	if proto.Type != ScenePrototype {
		return nil, fmt.Errorf("(GameMember::CloneScene)prototype error: %d is clone", protoID)
	}
	if _, exists := r.scenes[cloneID]; exists {
		return nil, fmt.Errorf("(GameMember::CloneScene)scene invalid: %d exists", cloneID)
	}
	s := &Scene{ID: cloneID, Type: SceneClone, Name: name, Proto: protoID, MaxPlayers: maxPlayers, Active: true}
	r.scenes[cloneID] = s
	return s, nil
}

// DeleteClone 复刻 GameMember::DeleteClone：scene not exists → scene type
// not clone → scene has player → scene not created。
func (r *SceneRegistry) DeleteClone(id uint32) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.scenes[id]
	if !ok {
		return fmt.Errorf("(GameMember::DeleteClone)scene not exists")
	}
	if s.Type != SceneClone {
		return fmt.Errorf("(GameMember::DeleteClone)scene type not clone")
	}
	if s.CurPlayers > 0 {
		return fmt.Errorf("(GameMember::DeleteClone)scene has player")
	}
	if !s.Active {
		return fmt.Errorf("(GameMember::DeleteClone)scene not created")
	}
	delete(r.scenes, id)
	return nil
}

// CanCreateScene 复刻 GameMember::CanCreateScene：场景存在且激活即可克隆。
func (r *SceneRegistry) CanCreateScene(id uint32) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.scenes[id]
	return ok && s.Active
}

// Remove 移除任意场景（CloseMember 清场用）。
func (r *SceneRegistry) Remove(id uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.scenes, id)
}

// Get 查询。
func (r *SceneRegistry) Get(id uint32) (Scene, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.scenes[id]
	if !ok {
		return Scene{}, false
	}
	return *s, true
}

// SetPlayerCount 维护在场计数（玩家进出场景时调用）。
func (r *SceneRegistry) SetPlayerCount(id uint32, n int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.scenes[id]; ok {
		s.CurPlayers = n
	}
}

// IsPrototypeScene 官方同名谓词。
func (r *SceneRegistry) IsPrototypeScene(id uint32) bool {
	s, ok := r.Get(id)
	return ok && s.Type == ScenePrototype
}

// Snapshot 全量场景（id 升序；SceneAll/SceneAdd 上报数据源）。
func (r *SceneRegistry) Snapshot() []Scene {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Scene, 0, len(r.scenes))
	for _, s := range r.scenes {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// CloneList 克隆场景清单（GetCloneSceneList）。
func (r *SceneRegistry) CloneList() []Scene {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Scene
	for _, s := range r.scenes {
		if s.Type == SceneClone {
			out = append(out, *s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Empty 是否无场景（Closing→Closed 门：场景清空才可落定）。
func (r *SceneRegistry) Empty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.scenes) == 0
}
