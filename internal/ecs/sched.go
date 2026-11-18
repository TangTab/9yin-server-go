package ecs

// System 是每帧执行的逻辑单元。命名用于调度审计与重复注册检测。
type System interface {
	Name() string
	Tick(w *World, dt float64)
}

type systemFunc struct {
	name string
	fn   func(*World, float64)
}

func (s systemFunc) Name() string                { return s.name }
func (s systemFunc) Tick(w *World, dt float64)   { s.fn(w, dt) }

// Func 把函数包装为 System。
func Func(name string, fn func(*World, float64)) System {
	return systemFunc{name: name, fn: fn}
}

// Scheduler 按注册顺序执行 System，每次 Tick 前推进 World 帧号。
type Scheduler struct {
	systems []System
	names   map[string]struct{}
}

func NewScheduler() *Scheduler {
	return &Scheduler{names: make(map[string]struct{})}
}

// Add 注册 System；重名 panic（调度表错误应在启动期暴露）。
func (s *Scheduler) Add(sys System) {
	if _, dup := s.names[sys.Name()]; dup {
		panic("ecs: duplicate system name: " + sys.Name())
	}
	s.names[sys.Name()] = struct{}{}
	s.systems = append(s.systems, sys)
}

// AddFunc 注册函数型 System。
func (s *Scheduler) AddFunc(name string, fn func(*World, float64)) {
	s.Add(Func(name, fn))
}

// Tick 推进一帧：帧号自增后按序执行全部 System。
func (s *Scheduler) Tick(w *World, dt float64) {
	w.tick++
	for _, sys := range s.systems {
		sys.Tick(w, dt)
	}
}

// Len 返回已注册 System 数。
func (s *Scheduler) Len() int { return len(s.systems) }
