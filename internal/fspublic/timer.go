package fspublic

import (
	"fmt"
	"sort"
	"sync"
)

// PubTimer 复刻 CPubTimer（pubtimer.cpp）：命名定时器集合，Opened 态
// 每秒 Run(elapsed) 驱动，超 100ms 记 "Timer running exceed %dms"。
type PubTimer struct {
	mu    sync.Mutex
	beats map[string]*pubBeat
}

// TimerFunc 定时器回调（官方 cdecl cb(kernel, funcName, elapsed, data)）。
type TimerFunc func(name string, elapsed int, data []Value)

type pubBeat struct {
	name     string
	interval int // 毫秒
	acc      int // 累计
	cb       TimerFunc
	data     []Value
	dead     bool
}

// NewPubTimer 建定时器。
func NewPubTimer() *PubTimer {
	return &PubTimer{beats: make(map[string]*pubBeat)}
}

// Add 官方 AddTimer（"timer already exists"/"time > 0" 断言语义）。
func (t *PubTimer) Add(name string, interval int, cb TimerFunc, data []Value) error {
	if interval <= 0 {
		return fmt.Errorf("(CPubTimer::Add)time > 0")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.beats[name]; ok && !t.beats[name].dead {
		return fmt.Errorf("(CPubTimer::Add)timer already exists: %s", name)
	}
	t.beats[name] = &pubBeat{name: name, interval: interval, cb: cb, data: data}
	return nil
}

// Remove 删定时器（官方 Remove→标记 deleted，Run 后清理）。
func (t *PubTimer) Remove(name string) {
	t.mu.Lock()
	if b, ok := t.beats[name]; ok {
		b.dead = true
	}
	t.mu.Unlock()
}

// Run 每秒驱动（官方 Execute：cb(kernel, funcName, elapsed, data)；
// >100ms 告警）。
func (t *PubTimer) Run(elapsedMS int) {
	t.mu.Lock()
	names := make([]string, 0, len(t.beats))
	for n := range t.beats {
		names = append(names, n)
	}
	sort.Strings(names)
	fire := make([]*pubBeat, 0)
	for _, n := range names {
		b := t.beats[n]
		if b.dead {
			delete(t.beats, n)
			continue
		}
		b.acc += elapsedMS
		if b.acc >= b.interval {
			b.acc -= b.interval
			fire = append(fire, b)
		}
	}
	t.mu.Unlock()
	for _, b := range fire {
		if b.cb != nil {
			b.cb(b.name, elapsedMS, b.data)
		}
	}
}

// Count 在网定时器数。
func (t *PubTimer) Count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := 0
	for _, b := range t.beats {
		if !b.dead {
			n++
		}
	}
	return n
}
