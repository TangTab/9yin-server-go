package ecs

import (
	"reflect"
	"sync"
)

type entitySlot struct {
	gen   uint32
	alive bool
}

// World 是实体与组件仓库的容器：实体槽位表 + 各组件的 sparse-set 存储。
// 设计为单线程使用（tick 模型）；并发分片属于 scene 层职责，本包不引入。
type World struct {
	tick     uint64
	slots    []entitySlot
	free     []uint32
	stores   []storage
	storeIDs map[reflect.Type]int
	bufPool  sync.Pool // 查询快照缓冲复用
}

// New 创建空 World。
func New() *World {
	return &World{storeIDs: make(map[reflect.Type]int)}
}

// Create 签发新实体。槽位复用时 generation 递增，旧句柄自动失效。
func (w *World) Create() EntityID {
	var idx uint32
	if n := len(w.free); n > 0 {
		idx = w.free[n-1]
		w.free = w.free[:n-1]
	} else {
		idx = uint32(len(w.slots))
		w.slots = append(w.slots, entitySlot{})
	}
	if w.slots[idx].gen == entityNullGen {
		w.slots[idx].gen = 1
	}
	w.slots[idx].alive = true
	return makeEntity(idx, w.slots[idx].gen)
}

// Alive 报告句柄是否指向存活实体（含 generation 校验）。
func (w *World) Alive(e EntityID) bool {
	i := e.Index()
	return int(i) < len(w.slots) && w.slots[i].alive && w.slots[i].gen == e.Gen()
}

// Destroy 销毁实体：从所有存储摘除组件、槽位置死、index 入空闲表。
func (w *World) Destroy(e EntityID) {
	if !w.Alive(e) {
		return
	}
	for _, st := range w.stores {
		st.erase(e)
	}
	w.slots[e.Index()].alive = false
	w.slots[e.Index()].gen++
	w.free = append(w.free, e.Index())
}

func storeID[T any](w *World) int {
	t := reflect.TypeOf((*T)(nil))
	if id, ok := w.storeIDs[t]; ok {
		return id
	}
	st := &Storage[T]{}
	w.storeIDs[t] = len(w.stores)
	w.stores = append(w.stores, st)
	return len(w.stores) - 1
}

func (w *World) store[T any]() *Storage[T] {
	return w.stores[storeID[T](w)].(*Storage[T])
}

// Store 返回组件 T 的存储（不存在则注册）。供需要直接操作存储的场景。
func (w *World) Store[T any]() *Storage[T] { return w.store[T]() }

// Add 为实体写入组件（已有则覆写并自增版本），返回值指针；实体不存活返回 nil。
func (w *World) Add[T any](e EntityID, v T) *T {
	if !w.Alive(e) {
		return nil
	}
	return w.store[T]().insert(e, v)
}

// Get 取组件副本。
func (w *World) Get[T any](e EntityID) (v T, ok bool) {
	if r := w.GetRef[T](e); r != nil {
		return *r, true
	}
	return
}

// GetRef 取组件可写指针；无组件或实体不存活返回 nil。
// 指针仅在下次结构性操作（同存储 Add/Remove/Destroy）前有效。
func (w *World) GetRef[T any](e EntityID) *T {
	return w.store[T]().GetRef(e)
}

// Has 报告实体是否持有组件 T。
func (w *World) Has[T any](e EntityID) bool { return w.store[T]().Has(e) }

// Set 覆写组件值并自增版本号；实体无组件返回 false（语义同 Storage.Set）。
func (w *World) Set[T any](e EntityID, v T) bool { return w.store[T]().Set(e, v) }

// Remove 摘除组件 T（无则空操作）。
func (w *World) Remove[T any](e EntityID) { w.store[T]().erase(e) }

// Version 返回组件版本号（sync 层脏检测入口）；无组件返回 0。
func (w *World) Version[T any](e EntityID) uint32 { return w.store[T]().Version(e) }

// Count 返回持有组件 T 的实体数。
func (w *World) Count[T any]() int { return w.store[T]().len() }

// CurrentTick 返回调度器推进到的帧号（从 1 起；0 表示尚未 tick）。
func (w *World) CurrentTick() uint64 { return w.tick }

// EntityCount 返回当前存活实体数。
func (w *World) EntityCount() int { return len(w.slots) - len(w.free) }

// ---- 查询 ----

// matchAll 求组件集合交集：选最小 dense 作驱动遍历，none 为排除集。
// 结果写入 buf（快照），回调期间的结构性变更不影响本轮遍历安全性。
func (w *World) matchAll(buf []EntityID, all []storage, none []storage) []EntityID {
	if len(all) == 0 {
		return buf
	}
	drv := all[0]
	for _, s := range all[1:] {
		if len(s.dense()) < len(drv.dense()) {
			drv = s
		}
	}
	for _, e := range drv.dense() {
		ok := true
		for _, s := range all {
			if s == drv || s.owns(e) {
				continue
			}
			ok = false
			break
		}
		if !ok {
			continue
		}
		for _, s := range none {
			if s.owns(e) {
				ok = false
				break
			}
		}
		if ok {
			buf = append(buf, e)
		}
	}
	return buf
}

func (w *World) acquire() []EntityID {
	if b, ok := w.bufPool.Get().([]EntityID); ok {
		return b[:0]
	}
	return make([]EntityID, 0, 64)
}

func (w *World) release(b []EntityID) { w.bufPool.Put(b) }

// Each1 遍历持有 A 的存活实体。回调内 Add/Remove/Destroy 均安全
// （基于快照 + 存活校验）；本实体被销毁时自动跳过后续回调。
func (w *World) Each1[A any](fn func(EntityID, *A)) {
	sa := w.store[A]()
	buf := w.matchAll(w.acquire(), []storage{sa}, nil)
	for _, e := range buf {
		if !w.Alive(e) {
			continue
		}
		fn(e, sa.GetRef(e))
	}
	w.release(buf)
}

// Each2 遍历同时持有 A、B 的实体，语义同 Each1。
func (w *World) Each2[A, B any](fn func(EntityID, *A, *B)) {
	sa, sb := w.store[A](), w.store[B]()
	buf := w.matchAll(w.acquire(), []storage{sa, sb}, nil)
	for _, e := range buf {
		if !w.Alive(e) {
			continue
		}
		fn(e, sa.GetRef(e), sb.GetRef(e))
	}
	w.release(buf)
}

// Each3 遍历同时持有 A、B、C 的实体，语义同 Each1。
func (w *World) Each3[A, B, C any](fn func(EntityID, *A, *B, *C)) {
	sa, sb, sc := w.store[A](), w.store[B](), w.store[C]()
	buf := w.matchAll(w.acquire(), []storage{sa, sb, sc}, nil)
	for _, e := range buf {
		if !w.Alive(e) {
			continue
		}
		fn(e, sa.GetRef(e), sb.GetRef(e), sc.GetRef(e))
	}
	w.release(buf)
}
