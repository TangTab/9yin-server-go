package proc

import (
	"errors"
	"sync"
)

// InprocBus 是 Bus 的进程内实现（solo 形态）：
//   - 同步直调投递，延迟最低、顺序确定
//   - handler 在 Publish 调用方 goroutine 执行——handler 内禁止再 Publish
//     回同一 Topic（会递归）；跨 Topic 转发合法
//   - net 层接入后，连接 goroutine 可直接 Publish，内部有锁保护订阅表
type InprocBus struct {
	mu      sync.RWMutex
	closed  bool
	subs    map[string][]subscription
	allSubs []subscription // "*" 广播订阅
}

type subscription struct {
	role    Role
	handler func(Envelope)
}

// TopicWildcard 订阅全部消息的通配主题（审计/日志）。
const TopicWildcard = "*"

// NewInprocBus 创建进程内总线。
func NewInprocBus() *InprocBus {
	return &InprocBus{subs: make(map[string][]subscription)}
}

// Subscribe 注册订阅。重复注册同一 handler 会多次投递（按需去责）。
func (b *InprocBus) Subscribe(role Role, topic string, handler func(Envelope)) error {
	if handler == nil {
		return errors.New("proc: handler 为空")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errors.New("proc: 总线已关闭")
	}
	if topic == TopicWildcard {
		b.allSubs = append(b.allSubs, subscription{role: role, handler: handler})
		return nil
	}
	b.subs[topic] = append(b.subs[topic], subscription{role: role, handler: handler})
	return nil
}

// Publish 同步投递：精确 Topic 订阅者先收，"*" 广播订阅者后收。
func (b *InprocBus) Publish(env Envelope) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return errors.New("proc: 总线已关闭")
	}
	targets := make([]subscription, 0, len(b.subs[env.Topic])+len(b.allSubs))
	targets = append(targets, b.subs[env.Topic]...)
	targets = append(targets, b.allSubs...)
	b.mu.RUnlock()

	for _, s := range targets {
		s.handler(env)
	}
	return nil
}

// Close 关闭总线；已注册 handler 不再被调用。
func (b *InprocBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	b.subs = nil
	b.allSubs = nil
}
