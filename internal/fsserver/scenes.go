package fsserver

import (
	"fmt"
	"sync"
)

// SceneState 场景状态机 4 态（state@scene+0x10，§五 member 面 SceneState 回报驱动）。
type SceneState int32

const (
	SSceneUnknown SceneState = iota
	SSceneStarting
	SSceneRunning
	SSceneError
)

// String …
func (s SceneState) String() string {
	switch s {
	case SSceneUnknown:
		return "Unknown"
	case SSceneStarting:
		return "Starting"
	case SSceneRunning:
		return "Running"
	case SSceneError:
		return "Error"
	}
	return "SceneState(?)"
}

// Scene 复刻 scene_t（§6.2 SyncAllScene 写序定案：+0/+4/+8/+0xC 四 int +
// +0x28/+0x68 两 char[0x40]）。Go 侧语义字段承载。
type Scene struct {
	ID        uint32
	MemberIdx uint32 // 所属场景服
	Kind      uint32 // 0=普通 1=副本原型 2=副本实例（CloneScene 语义）
	State     SceneState
	Script    string
	Config    string
	ProtoID   uint32 // 副本实例的原型场景
	PlayerCnt int
}

// Scenes 复刻 CWorldScenes：普通场景槽 + 副本槽（官方 GetScene(i+0x400)）。
type Scenes struct {
	mu     sync.RWMutex
	normal map[uint32]*Scene
	clone  map[uint32]*Scene
	nextID uint32 // 副本实例 id 分配
}

// NewScenes …
func NewScenes() *Scenes {
	return &Scenes{normal: make(map[uint32]*Scene), clone: make(map[uint32]*Scene), nextID: 1}
}

// CloneBase 副本槽基址（官方 GetCloneScene 索引 +0x400 语义）。
const CloneBase = 0x400

// AddScene 普通场景（AddScene(sceneID, memberID, script, config)）。
func (ss *Scenes) AddScene(id, memberIdx uint32, script, config string) (*Scene, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if _, ok := ss.normal[id]; ok {
		return nil, fmt.Errorf("(GameWorld::AddScene)scene %d exists", id)
	}
	s := &Scene{ID: id, MemberIdx: memberIdx, Script: script, Config: config}
	ss.normal[id] = s
	return s, nil
}

// AddPrototypeScene 副本原型（Kind=1；Clonable 场景）。
func (ss *Scenes) AddPrototypeScene(id, memberIdx uint32, script, config string) (*Scene, error) {
	s, err := ss.AddScene(id, memberIdx, script, config)
	if err != nil {
		return nil, err
	}
	s.Kind = 1
	return s, nil
}

// GetScene …
func (ss *Scenes) GetScene(id uint32) *Scene {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	if s, ok := ss.clone[id]; ok {
		return s
	}
	return ss.normal[id]
}

// CreateClone 副本实例分配（MemberMsgCloneScene 反向；官方 CreateClone
// world→member type 1 的对偶）。返回新场景。
func (ss *Scenes) CreateClone(protoID uint32) (*Scene, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	proto, ok := ss.normal[protoID]
	if !ok || proto.Kind != 1 {
		return nil, fmt.Errorf("(GameWorld::MemberMsgCloneScene)prototype not found")
	}
	id := CloneBase + ss.nextID
	ss.nextID++
	s := &Scene{
		ID: id, MemberIdx: proto.MemberIdx, Kind: 2,
		Script: proto.Script, Config: proto.Config, ProtoID: protoID,
	}
	ss.clone[id] = s
	return s, nil
}

// DeleteClone …
func (ss *Scenes) DeleteClone(id uint32) bool {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if _, ok := ss.clone[id]; !ok {
		return false
	}
	delete(ss.clone, id)
	return true
}

// SetState …
func (ss *Scenes) SetState(id uint32, st SceneState) bool {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	s := ss.clone[id]
	if s == nil {
		s = ss.normal[id]
	}
	if s == nil {
		return false
	}
	s.State = st
	return true
}

// EachNormal / EachClone 遍历（WorldStateClearScenes 批语义）。
func (ss *Scenes) EachNormal(fn func(*Scene) bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	for _, s := range ss.normal {
		if !fn(s) {
			return
		}
	}
}

// EachClone …
func (ss *Scenes) EachClone(fn func(*Scene) bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	for _, s := range ss.clone {
		if !fn(s) {
			return
		}
	}
}

// Counts （normal, clone）。
func (ss *Scenes) Counts() (int, int) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return len(ss.normal), len(ss.clone)
}
