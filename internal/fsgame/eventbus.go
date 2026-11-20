package fsgame

import "fmt"

// 六类回调签名（对应官方注册面；第一参恒为 Kernel）。

// ClassCallbackFn 类级回调（AddClassCallback；官方首事件 "OnCreateClass"，
// 类 schema 就绪时触发一次，模块在此声明类属性/记录——Go 面即装载 struct XML）。
type ClassCallbackFn func(k *Kernel, class string)

// EventFn 事件回调（AddEventCallback：类 + 事件名 → 处理器；
// 形参 (kernel, self, from, args) 对齐官方
// int fn(IKernel*, PERSISTID const&, PERSISTID const&, IVarList const&)）。
type EventFn func(k *Kernel, self, from PersistID, args VarList)

// CmdFn 整型命令回调（AddIntCustomHook/AddIntCommandHook：类 + 命令号 → 处理器；
// 客户端整型命令是玩家操作的主入口，PlayerBase 命令 0xC/0x33E/0x35C/0x3F7 等）。
type CmdFn func(k *Kernel, self, from PersistID, args VarList)

// HeartBeatFn 心跳（DeclareHeartBeat：命名定时器，官方按场景节拍驱动）。
type HeartBeatFn func(k *Kernel, self PersistID, beats int32)

// CriticalFn 临界变量回调（DeclareCritical：类属性变更即触发，
// 形参对齐 int fn(IKernel*, PERSISTID const&, char const*, IVar const&)）。
type CriticalFn func(k *Kernel, self PersistID, prop string, old Var)

// RecHookFn 记录钩子（DeclareRecHook：记录行变更触发；
// 官方 int fn(IKernel*, PERSISTID const&, int op, int row, int col)）。
type RecHookFn func(k *Kernel, self PersistID, record string, op, row int32)

// Bus 六类回调总线（官方 CEventCmdMnger + CKernel 注册面的等价物）。
// 命名心跳为全局单例；类事件按 (类,事件) 多播；命令按 (类,命令号) 单播——
// 官方分两条注册/分发通道：AddIntCustomHook（客户端 custom 通道）与
// AddIntCommandHook（命令通道），两表独立（同号不冲突，如 0x138 既登记
// TeamModule::OnCustomLeave（custom）又登记 FriendModule::OnTransDecTicketRes
// （command））；字符串命令（AddCommandHook，如 "chat"）按 (类,命令名) 单播。
type Bus struct {
	classEvents map[string][]ClassCallbackFn // (class|event) → 类级回调
	events      map[string][]EventFn         // (class|event) → 事件回调
	commands    map[int32]CmdFn              // 命令号 → 处理器（player 类主载）
	customCmds  map[string]map[int32]CmdFn   // 类 → 命令号 → AddIntCustomHook 处理器
	intCmds     map[string]map[int32]CmdFn   // 类 → 命令号 → AddIntCommandHook 处理器
	strCmds     map[string]CmdFn             // (class|name) → 处理器（字符串命令）
	heartbeats  []namedBeat
	criticals   map[string][]CriticalFn // (class|prop) → 回调
	recHooks    map[string][]RecHookFn  // 记录名（或 Class::Name）→ 回调
}

type namedBeat struct {
	name string
	fn   HeartBeatFn
}

func busKey(class, name string) string { return class + "\x00" + name }

// NewBus 创建空总线。
func NewBus() *Bus {
	return &Bus{
		classEvents: make(map[string][]ClassCallbackFn),
		events:      make(map[string][]EventFn),
		commands:    make(map[int32]CmdFn),
		customCmds:  make(map[string]map[int32]CmdFn),
		intCmds:     make(map[string]map[int32]CmdFn),
		strCmds:     make(map[string]CmdFn),
		criticals:   make(map[string][]CriticalFn),
		recHooks:    make(map[string][]RecHookFn),
	}
}

// OnClassEvent 类级回调注册（AddClassCallback(class, event, fn)）。
func (b *Bus) OnClassEvent(class, event string, fn ClassCallbackFn) {
	b.classEvents[busKey(class, event)] = append(b.classEvents[busKey(class, event)], fn)
}

// OnEvent 事件回调注册（AddEventCallback(class, event, fn)）。
func (b *Bus) OnEvent(class, event string, fn EventFn) {
	b.events[busKey(class, event)] = append(b.events[busKey(class, event)], fn)
}

// OnCommand 类命令注册（AddIntCustomHook(class, cmd, fn)；单播，后注册覆盖——
// 官方同名命令由装载序决定唯一处理器）。
func (b *Bus) OnCommand(class string, cmd int32, fn CmdFn) {
	if _, ok := b.customCmds[class]; !ok {
		b.customCmds[class] = make(map[int32]CmdFn)
	}
	b.customCmds[class][cmd] = fn
}

// OnIntCommand 命令通道注册（AddIntCommandHook(class, cmd, fn)；与
// OnCommand 分表，同号不冲突）。
func (b *Bus) OnIntCommand(class string, cmd int32, fn CmdFn) {
	if _, ok := b.intCmds[class]; !ok {
		b.intCmds[class] = make(map[int32]CmdFn)
	}
	b.intCmds[class][cmd] = fn
}

// OnStringCommand 字符串命令注册（AddCommandHook/AddCustomHook(class, name, fn)；
// 客户端字符串命令通道，如聊天发送 "chat"）。
func (b *Bus) OnStringCommand(class, name string, fn CmdFn) {
	b.strCmds[busKey(class, name)] = fn
}

// DispatchStringCommand 字符串命令路由（类继承链上溯；返回是否命中）。
func (b *Bus) DispatchStringCommand(k *Kernel, class, name string, self, from PersistID, args VarList) bool {
	for c := class; c != ""; {
		if fn, ok := b.strCmds[busKey(c, name)]; ok {
			fn(k, self, from, args)
			return true
		}
		cd := k.Class(c)
		if cd == nil {
			break
		}
		c = cd.Parent
	}
	return false
}

// HeartBeat 心跳注册（DeclareHeartBeat(name, fn)）。
func (b *Bus) HeartBeat(name string, fn HeartBeatFn) {
	b.heartbeats = append(b.heartbeats, namedBeat{name: name, fn: fn})
}

// Critical 临界注册（DeclareCritical(class, prop, fn)）。
func (b *Bus) Critical(class, prop string, fn CriticalFn) {
	k := busKey(class, prop)
	b.criticals[k] = append(b.criticals[k], fn)
}

// RecHook 记录钩子注册（DeclareRecHook(name, fn)）。
func (b *Bus) RecHook(name string, fn RecHookFn) {
	b.recHooks[name] = append(b.recHooks[name], fn)
}

// FireClassEvent 触发类级回调（类继承链上溯多播：role 事件 player 类也收）。
func (b *Bus) FireClassEvent(k *Kernel, class, event string) {
	for c := class; c != ""; {
		for _, fn := range b.classEvents[busKey(c, event)] {
			fn(k, class)
		}
		cd := k.Class(c)
		if cd == nil {
			break
		}
		c = cd.Parent
	}
}

// FireEvent 触发事件（类继承链上溯多播；返回命中的处理器数）。
func (b *Bus) FireEvent(k *Kernel, class, event string, self, from PersistID, args VarList) int {
	n := 0
	for c := class; c != ""; {
		for _, fn := range b.events[busKey(c, event)] {
			fn(k, self, from, args)
			n++
		}
		cd := k.Class(c)
		if cd == nil {
			break
		}
		c = cd.Parent
	}
	return n
}

// DispatchCommand 整型命令路由（类继承链上溯；custom 通道优先，未命中回落
// 命令通道——Go 面兼容桥；返回是否命中）。
func (b *Bus) DispatchCommand(k *Kernel, class string, cmd int32, self, from PersistID, args VarList) bool {
	if b.dispatchIn(k, b.customCmds, class, cmd, self, from, args) {
		return true
	}
	return b.dispatchIn(k, b.intCmds, class, cmd, self, from, args)
}

// DispatchIntCommand 命令通道路由（命令通道优先，未命中回落 custom 通道）。
func (b *Bus) DispatchIntCommand(k *Kernel, class string, cmd int32, self, from PersistID, args VarList) bool {
	if b.dispatchIn(k, b.intCmds, class, cmd, self, from, args) {
		return true
	}
	return b.dispatchIn(k, b.customCmds, class, cmd, self, from, args)
}

// dispatchIn 单表路由（类继承链上溯）。
func (b *Bus) dispatchIn(k *Kernel, table map[string]map[int32]CmdFn, class string, cmd int32, self, from PersistID, args VarList) bool {
	for c := class; c != ""; {
		if m := table[c]; m != nil {
			if fn, ok := m[cmd]; ok {
				fn(k, self, from, args)
				return true
			}
		}
		cd := k.Class(c)
		if cd == nil {
			break
		}
		c = cd.Parent
	}
	return false
}

// FireCritical 属性变更通知（类继承链上溯多播）。
func (b *Bus) FireCritical(k *Kernel, class, prop string, self PersistID, old Var) {
	for c := class; c != ""; {
		for _, fn := range b.criticals[busKey(c, prop)] {
			fn(k, self, prop, old)
		}
		cd := k.Class(c)
		if cd == nil {
			break
		}
		c = cd.Parent
	}
}

// FireRecHook 记录行变更通知。
func (b *Bus) FireRecHook(k *Kernel, self PersistID, record string, op, row int32) {
	for _, fn := range b.recHooks[record] {
		fn(k, self, record, op, row)
	}
}

// TickHeartBeats 心跳泵：对每个存活对象驱动全部命名心跳一拍
// （官方按 FxCore 定步泵驱动；beats 为累计拍数）。
func (b *Bus) TickHeartBeats(k *Kernel, beats int32) {
	for _, hb := range b.heartbeats {
		for _, o := range k.objects {
			hb.fn(k, o.id, beats)
		}
	}
}

// HeartBeatNames 已注册心跳名（测试与对照用）。
func (b *Bus) HeartBeatNames() []string {
	out := make([]string, 0, len(b.heartbeats))
	for _, hb := range b.heartbeats {
		out = append(out, hb.name)
	}
	return out
}

// String 总线注册面概览。
func (b *Bus) String() string {
	n := 0
	for _, m := range b.customCmds {
		n += len(m)
	}
	for _, m := range b.intCmds {
		n += len(m)
	}
	return fmt.Sprintf("Bus{events=%d cmds=%d heartbeats=%d criticals=%d rechoks=%d}",
		len(b.events), len(b.commands)+n, len(b.heartbeats), len(b.criticals), len(b.recHooks))
}
