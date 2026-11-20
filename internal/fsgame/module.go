package fsgame

import (
	"bytes"
	"fmt"
)

// bytesReader 小工具（kernel.LoadStruct 入参用）。
func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }

// Module 复刻 ILogicModule 五虚槽的 Go 面（dtor 由 GC 承担）：
// Init（注册期）→ BeforeLaunch → LoadResource（静态配置）→ 运行 → Shut。
type Module interface {
	// Name 模块名（IModuleCreator::Name，官方返回类名字符串）。
	Name() string
	// Init 注册期（官方 ?Init@Xxx@@UAE_NPAVIKernel@@@Z）。
	Init(k *Kernel) error
	// BeforeLaunch 预启动（官方默认实现恒真，可覆盖）。
	BeforeLaunch(k *Kernel) error
	// LoadResource 静态配置装载（ini/*.xml 经 PreloadConfig）。
	LoadResource(k *Kernel) error
	// Shut 关停。
	Shut(k *Kernel) error
}

// BaseModule 默认实现（官方 ILogicModule 默认槽：BeforeLaunch/LoadResource 恒真）。
type BaseModule struct{}

// BeforeLaunch 默认恒真（官方 0x3A3590 默认实现）。
func (BaseModule) BeforeLaunch(*Kernel) error { return nil }

// LoadResource 默认恒真（官方 0x3A3600 默认实现）。
func (BaseModule) LoadResource(*Kernel) error { return nil }

// Shut 默认恒真。
func (BaseModule) Shut(*Kernel) error { return nil }

// Creator 复刻 IModuleCreator：Name/Create/Destroy。
type Creator struct {
	NameStr string
	Create  func() Module
}

// Registry 模块注册链（官方 g_pModuleCreator 头插链；Go 面按序登记，
// Assemble 以登记序执行 Init——即官方「构造逆序 = Init 序」的等价表达：
// 玩法表 §8 的 Init 序即本表推荐登记序：LogicClass → StaticDataQuery → …）。
type Registry struct {
	creators []Creator
}

// Register 登记创建器（重复名报错——官方链内同名模块互斥）。
func (r *Registry) Register(c Creator) error {
	for _, e := range r.creators {
		if e.NameStr == c.NameStr {
			return fmt.Errorf("fsgame: 模块 %s 重复注册", c.NameStr)
		}
	}
	r.creators = append(r.creators, c)
	return nil
}

// Assemble 全量装配：创建 → Init → BeforeLaunch → LoadResource → Shut 回调登记。
// 返回内核（模块表已注入，GetLogicModule 可用）。
func (r *Registry) Assemble(k *Kernel) ([]Module, error) {
	var mods []Module
	for _, c := range r.creators {
		m := c.Create()
		if err := m.Init(k); err != nil {
			return mods, fmt.Errorf("fsgame: %s.Init: %w", c.NameStr, err)
		}
		k.modules[c.NameStr] = m
		mods = append(mods, m)
	}
	for _, m := range mods {
		if err := m.BeforeLaunch(k); err != nil {
			return mods, fmt.Errorf("fsgame: %s.BeforeLaunch: %w", m.Name(), err)
		}
	}
	for _, m := range mods {
		if err := m.LoadResource(k); err != nil {
			return mods, fmt.Errorf("fsgame: %s.LoadResource: %w", m.Name(), err)
		}
	}
	return mods, nil
}

// ShutAll 逆序关停。
func ShutAll(mods []Module, k *Kernel) {
	for i := len(mods) - 1; i >= 0; i-- {
		_ = mods[i].Shut(k)
	}
}

// RegisterFirstBatch 登记第一批（登录可玩最小闭环）十模块。
// 顺序对齐官方 Init 序（玩法表 §8：容器/物品先于 PlayerBase，
// PlayerBase(94) → CreateRole(133)/Motion(134) → Level … Request(180)）。
func RegisterFirstBatch(r *Registry) error {
	seq := []Creator{
		{"ContainerBaseModule", func() Module { return &ContainerBaseModule{} }},
		{"ItemBaseModule", func() Module { return &ItemBaseModule{} }},
		{"CapitalModule", func() Module { return &CapitalModule{} }},
		{"SceneBaseModule", func() Module { return &SceneBaseModule{} }},
		{"LocationModule", func() Module { return &LocationModule{} }},
		{"PlayerBaseModule", func() Module { return &PlayerBaseModule{} }},
		{"CreateRoleModule", func() Module { return &CreateRoleModule{} }},
		{"MotionModule", func() Module { return &MotionModule{} }},
		{"LevelModule", func() Module { return &LevelModule{} }},
		{"RequestModule", func() Module { return &RequestModule{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RegisterSecondBatch 登记第二批（社交底座）：Gmcc/Post/Team/Friend/Chat/Log
// 六模块（docs/official/FsGame全解与Go复刻.md §7 后续批次定案）。
// 顺序对齐官方 Init 序（玩法表 §8）：Gmcc(13) → Post(178) → Team(182) →
// Friend(183) → Chat(184) → Log(240)。
func RegisterSecondBatch(r *Registry) error {
	seq := []Creator{
		{"GmccModule", func() Module { return &GmccModule{} }},
		{"PostModule", func() Module { return &PostModule{} }},
		{"TeamModule", func() Module { return &TeamModule{} }},
		{"FriendModule", func() Module { return &FriendModule{} }},
		{"ChatModule", func() Module { return &ChatModule{} }},
		{"LogModule", func() Module { return &LogModule{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}
