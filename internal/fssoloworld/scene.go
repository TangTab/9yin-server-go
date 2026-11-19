package fssoloworld

import (
	"sort"
	"sync"
)

// SceneState 场景状态机 8 态（SoloWorldScene.obj rdata 定案）。
type SceneState int

const (
	ScUnknown SceneState = iota
	ScStarting
	ScRunning
	ScCloning
	ScCloned
	ScDeleting
	ScClosed
	ScError
)

var sceneStateNames = map[SceneState]string{
	ScUnknown: "Unknown", ScStarting: "Starting", ScRunning: "Running",
	ScCloning: "Cloning", ScCloned: "Cloned", ScDeleting: "Deleting",
	ScClosed: "Closed", ScError: "Error",
}

func (s SceneState) String() string { return sceneStateNames[s] }

// Scene 复刻 scene_t（§九）：index/归属 member/脚本/配置/克隆原型。
type Scene struct {
	Index   int    // 场景编号（scenes.ini 节名）
	Member  int    // 所在场景服编号
	Type    int    // 0 普通 / 1 克隆原型
	Script  string // 逻辑类名（缺省 "scene"）
	Config  string // 场景配置名
	State   SceneState
	Players int // Inc/DecScenePlayers 计数
}

// WorldScenes 复刻 CWorldScenes（0x178）：normal+clone 两段。
type WorldScenes struct {
	mu    sync.Mutex
	byIdx map[int]*Scene
	order []int
}

// NewWorldScenes ctor @0x100e10e0。
func NewWorldScenes() *WorldScenes {
	return &WorldScenes{byIdx: map[int]*Scene{}}
}

// AddScene 官方 AddScene(index, member, type, ?, script, config)。
func (w *WorldScenes) AddScene(index, member, typ int, script, config string) bool {
	if script == "" || config == "" {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, dup := w.byIdx[index]; dup {
		return false
	}
	w.byIdx[index] = &Scene{Index: index, Member: member, Type: typ,
		Script: script, Config: config, State: ScUnknown}
	w.order = append(w.order, index)
	return true
}

// RemoveScene 官方 RemoveScene(index)。
func (w *WorldScenes) RemoveScene(index int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, ok := w.byIdx[index]; !ok {
		return false
	}
	delete(w.byIdx, index)
	for i, v := range w.order {
		if v == index {
			w.order = append(w.order[:i], w.order[i+1:]...)
			break
		}
	}
	return true
}

// GetScene 官方 GetScene(index)。
func (w *WorldScenes) GetScene(index int) *Scene {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.byIdx[index]
}

// Snapshot 有序（GetFirstSceneId/遍历语义）。
func (w *WorldScenes) Snapshot() []*Scene {
	w.mu.Lock()
	defer w.mu.Unlock()
	idxs := make([]int, len(w.order))
	copy(idxs, w.order)
	sort.Ints(idxs)
	out := make([]*Scene, 0, len(idxs))
	for _, i := range idxs {
		out = append(out, w.byIdx[i])
	}
	return out
}

// CountNormal / CountClone / CountAll 计数族。
func (w *WorldScenes) CountNormal() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for _, s := range w.byIdx {
		if s.Type == 0 {
			n++
		}
	}
	return n
}

func (w *WorldScenes) CountClone() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for _, s := range w.byIdx {
		if s.Type != 0 {
			n++
		}
	}
	return n
}

// CountByState 场景状态计数（CheckAllScene 判据）。
func (w *WorldScenes) CountByState(st SceneState) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for _, s := range w.byIdx {
		if s.State == st {
			n++
		}
	}
	return n
}

// AllRunning CheckAllScene 语义：非空且全部 Running（克隆态允许 Cloned）。
func (w *WorldScenes) AllRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.byIdx) == 0 {
		return false
	}
	for _, s := range w.byIdx {
		if s.State != ScRunning {
			return false
		}
	}
	return true
}

// SetState / IncPlayers / DecPlayers 维护族。
func (w *WorldScenes) SetState(index int, st SceneState) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if s, ok := w.byIdx[index]; ok {
		s.State = st
		return true
	}
	return false
}

func (w *WorldScenes) IncPlayers(index int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if s, ok := w.byIdx[index]; ok {
		s.Players++
		return true
	}
	return false
}

func (w *WorldScenes) DecPlayers(index int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	s, ok := w.byIdx[index]
	if ok && s.Players > 0 {
		s.Players--
		return true
	}
	return false
}
