package ecs

// storage 是 World 组件仓库的统一接口，供查询/销毁时跨类型操作。
// 仅由本包使用；业务代码一律走 World / Storage[T] 的类型化入口。
type storage interface {
	owns(e EntityID) bool
	erase(e EntityID)
	dense() []EntityID
}

// Storage[T] 是组件 T 的 sparse-set 存储：
//
//	sparse[entityIndex] → dense 下标（-1 表示无）
//	denseE / denseV     → 紧凑的实体与值数组（swap-remove 维持紧凑）
//	ver                 → 每个在存实体的组件版本号，Set 时自增（sync 层脏检测用）
//
// 引用约束：GetRef 返回的指针只在下一次对同一存储的结构性操作
// （insert/erase）前有效，禁止跨帧持有。
type Storage[T any] struct {
	sparse []int32
	denseE []EntityID
	denseV []T
	ver    []uint32
}

// Has 报告实体是否持有该组件。
func (s *Storage[T]) Has(e EntityID) bool {
	i := e.Index()
	return int(i) < len(s.sparse) && s.sparse[i] >= 0
}

// GetRef 返回组件可写指针；无组件时返回 nil。
func (s *Storage[T]) GetRef(e EntityID) *T {
	if !s.Has(e) {
		return nil
	}
	d := s.sparse[e.Index()]
	return &s.denseV[d]
}

// Set 覆写组件值并自增版本号；实体无组件时返回 false。
func (s *Storage[T]) Set(e EntityID, v T) bool {
	if !s.Has(e) {
		return false
	}
	d := s.sparse[e.Index()]
	s.denseV[d] = v
	s.ver[d]++
	return true
}

// Version 返回组件当前版本号；无组件时返回 0。
// 版本从 1 起，每次 Set 自增；重新 Add 后版本重新从 1 起。
func (s *Storage[T]) Version(e EntityID) uint32 {
	if !s.Has(e) {
		return 0
	}
	return s.ver[s.sparse[e.Index()]]
}

func (s *Storage[T]) len() int { return len(s.denseE) }

func (s *Storage[T]) owns(e EntityID) bool { return s.Has(e) }

func (s *Storage[T]) dense() []EntityID { return s.denseE }

// insert 写入组件（已有则覆写并自增版本），返回值指针。
func (s *Storage[T]) insert(e EntityID, v T) *T {
	i := e.Index()
	for int(i) >= len(s.sparse) {
		s.sparse = append(s.sparse, -1)
	}
	if d := s.sparse[i]; d >= 0 {
		s.denseV[d] = v
		s.ver[d]++
		return &s.denseV[d]
	}
	s.sparse[i] = int32(len(s.denseE))
	s.denseE = append(s.denseE, e)
	s.denseV = append(s.denseV, v)
	s.ver = append(s.ver, 1)
	return &s.denseV[len(s.denseV)-1]
}

// erase swap-remove 组件，被移动的实体带走自己的版本号。
func (s *Storage[T]) erase(e EntityID) {
	if !s.Has(e) {
		return
	}
	d := s.sparse[e.Index()]
	last := len(s.denseE) - 1
	if d != int32(last) {
		s.denseE[d] = s.denseE[last]
		s.denseV[d] = s.denseV[last]
		s.ver[d] = s.ver[last]
		s.sparse[s.denseE[d].Index()] = d
	}
	s.denseE = s.denseE[:last]
	s.denseV = s.denseV[:last]
	s.ver = s.ver[:last]
	s.sparse[e.Index()] = -1
}
