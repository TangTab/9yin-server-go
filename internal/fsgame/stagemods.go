package fsgame

// 十五期 B：事件系统底座余量十六模块（stagemodule 管线——docs/official/
// FsGame全解与Go复刻.md §8 事件系统底座表余量）。官方源码树 fsgame\
// stagemodule\（17 文件泄露：stagemodule/trigger*event/processor*/
// processexecutor/stageeventhelper/creatormnger/codeline）。
//
// 架构定案（2026-11-20 侦察）：该族 loadres 全空——**不读静态配置表**，
// 触发/行为定义经 Lua 脚本（lua64/trigger/tr_*.lua、stage*.lua）与副本
// zone 运行时组合注入；Go 复刻同构：TriggerDataModule 登记面 + 六类事件
// 触发器 → TriggerProcesser 路由 → 八 Processor 方言 → 三 Executor。
//
// 真档接入（十五期 A）：share/area/areainfo.ini（区域禁则 12 键——
// TriggerAreaEvent 的区域方言）；share/clone/cloneinfo.ini（副本难度表）。

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
	"github.com/lizhanyin/9yin-server-go/internal/sharedata"
)

// ---- TriggerDataModule（Init 序 11）----

// TriggerDataModule 复刻官方 TriggerDataModule（Init @0x555970；符号 117）——
// 触发器登记面（官方 loadres 空——数据经 Lua/API 运行时注入，Go 同构）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddIntCustomHook ("player",270=0x10E)
type TriggerDataModule struct {
	BaseModule

	// Trigger 触发器定义（ID → 定义：源事件 + 条件 + 行为链）。
	triggers map[string]*StageTrigger
	// events 流水（测试面）。
	events []string
}

// StageTrigger 触发器定义（运行时注入面——官方经 Lua 脚本组装）。
type StageTrigger struct {
	ID         string           // 触发器号
	Source     string           // 源事件（player/npc/scene/area/buffer/item 六族 + 事件名）
	Once       bool             // 单发
	Fires      int              // 已触发次数
	Conditions []StageCondition // 条件链（全过才触发）
	Behaviors  []StageBehavior  // 行为链（按序执行）
}

// StageCondition 条件（八 Processor 的 Condition/Comparision 方言词汇）。
type StageCondition struct {
	Kind  string // prop/record/level/count…
	Prop  string
	Op    string // = != > >= < <=
	Value int32
}

// StageBehavior 行为（八 Processor 方言词汇）。
type StageBehavior struct {
	Kind  string // ai/clone/common/record/property/object/notice/buff
	Op    string // 子操作（如 set/add/summon/notice）
	Prop  string
	Value int32
	Text  string
}

// Name 模块名。
func (m *TriggerDataModule) Name() string { return "TriggerDataModule" }

// Init 注册期（对齐官方 0x555970）。
func (m *TriggerDataModule) Init(k *Kernel) error {
	m.triggers = make(map[string]*StageTrigger)
	k.Bus().OnCommand("player", 270, m.onCustomTrigger)
	return nil
}

// Register 触发器登记（Lua nx_* / API 注入面）。
func (m *TriggerDataModule) Register(t *StageTrigger) {
	m.triggers[t.ID] = t
	m.events = append(m.events, "register:"+t.ID)
}

// Trigger 取触发器定义。
func (m *TriggerDataModule) Trigger(id string) *StageTrigger { return m.triggers[id] }

// onCustomTrigger 官方 @0x555A30（cust 270=0x10E）：触发器 UI/查询消息。
func (m *TriggerDataModule) onCustomTrigger(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, fmt.Sprintf("cust:%d", args.Int(0)))
}

// Events 流水（测试面）。
func (m *TriggerDataModule) Events() []string { return m.events }

// ---- TriggerBaseModule + 六类事件触发器 ----

// TriggerBaseModule 复刻官方 TriggerBaseModule（Init @0xBEE810；符号 112；
// 零内核注册——触发器容器基座：求值 + 单发闸 + 行为链派发）。
type TriggerBaseModule struct {
	BaseModule

	// fired 触发流水（测试面："fire:<id>"）。
	fired []string
}

func (m *TriggerBaseModule) Name() string         { return "TriggerBaseModule" }
func (m *TriggerBaseModule) Init(k *Kernel) error { return nil }

// Fire 求值入口：条件链全过 → 行为链派发（TriggerProcesser 路由）→
// 单发闸（Once 已发不再）。
func (m *TriggerBaseModule) Fire(k *Kernel, t *StageTrigger, self, from PersistID, args VarList) bool {
	if t == nil || (t.Once && t.Fires > 0) {
		return false
	}
	if !ConditionsMet(k, t.Conditions, self) {
		return false
	}
	t.Fires++
	m.fired = append(m.fired, "fire:"+t.ID)
	stageExecute(k, t, self, from, args)
	return true
}

// Fired 触发流水（测试面）。
func (m *TriggerBaseModule) Fired() []string { return m.fired }

// ConditionsMet 条件链求值（ConditionModule 协作）。
func ConditionsMet(k *Kernel, conds []StageCondition, self PersistID) bool {
	cm, _ := k.GetLogicModule("ConditionModule").(*ConditionModule)
	if cm == nil {
		return true
	}
	for _, c := range conds {
		if !cm.Eval(k, self, c) {
			return false
		}
	}
	return true
}

// TriggerPlayerEvent 复刻官方 TriggerPlayerEvent（Init @0x581B20；符号 173；
// 官方零内核注册——player 源触发器，经 StageModule 内部分派；Go 面：六类
// Bus 事件直接挂接）。
type TriggerPlayerEvent struct {
	TriggerBaseModule
	// tdm 触发器登记面缓存。
	tdm *TriggerDataModule
}

func (m *TriggerPlayerEvent) Name() string { return "TriggerPlayerEvent" }

func (m *TriggerPlayerEvent) Init(k *Kernel) error {
	for _, ev := range []string{"OnReady", "OnEntry", "OnLevelChanged", "OnDead"} {
		ev := ev
		k.Bus().OnEvent("player", ev, func(k *Kernel, self, from PersistID, args VarList) {
			m.dispatch(k, "player", ev, self, args)
		})
	}
	return nil
}

// dispatch 源匹配派发（Source 方言「<族>」或「<族>:<事件>」精确）。
func (m *TriggerPlayerEvent) dispatch(k *Kernel, source, event string, self PersistID, args VarList) {
	tdm, _ := k.GetLogicModule("TriggerDataModule").(*TriggerDataModule)
	if tdm == nil {
		return
	}
	for _, t := range tdm.triggers {
		if t.Source == source || t.Source == source+":"+event {
			m.Fire(k, t, self, NullPersistID, args)
		}
	}
}

// TriggerSceneEvent 复刻官方 TriggerSceneEvent（Init @0x5853D0；符号 525）。
type TriggerSceneEvent struct {
	TriggerBaseModule
}

func (m *TriggerSceneEvent) Name() string { return "TriggerSceneEvent" }

func (m *TriggerSceneEvent) Init(k *Kernel) error {
	k.Bus().OnEvent("scene", "OnCreate", m.onSceneEvent)
	k.Bus().OnEvent("scene", "OnDestroy", m.onSceneEvent)
	return nil
}

func (m *TriggerSceneEvent) onSceneEvent(k *Kernel, self, from PersistID, args VarList) {
	tdm, _ := k.GetLogicModule("TriggerDataModule").(*TriggerDataModule)
	if tdm == nil {
		return
	}
	for _, t := range tdm.triggers {
		if strings.HasPrefix(t.Source, "scene") {
			m.Fire(k, t, self, NullPersistID, nil)
		}
	}
}

// TriggerNpcEvent 复刻官方 TriggerNpcEvent（Init @0x57E810；符号 169）。
type TriggerNpcEvent struct {
	TriggerBaseModule
}

func (m *TriggerNpcEvent) Name() string { return "TriggerNpcEvent" }

func (m *TriggerNpcEvent) Init(k *Kernel) error {
	k.Bus().OnIntCommand("AttackNpc", 616, m.onNpcKilled)
	k.Bus().OnIntCommand("CommonNpc", 616, m.onNpcKilled)
	return nil
}

func (m *TriggerNpcEvent) onNpcKilled(k *Kernel, self, from PersistID, args VarList) {
	tdm, _ := k.GetLogicModule("TriggerDataModule").(*TriggerDataModule)
	if tdm == nil {
		return
	}
	for _, t := range tdm.triggers {
		if strings.HasPrefix(t.Source, "npc") {
			m.Fire(k, t, from, self, args)
		}
	}
}

// TriggerAreaEvent 复刻官方 TriggerAreaEvent（Init @0x56EF40；符号 272）——
// 区域触发器 + areainfo 禁则真档（十五期 A 接入）。
type TriggerAreaEvent struct {
	TriggerBaseModule

	// AreaRules 区域禁则真档（share/area/areainfo.ini：区域号 →
	// forbid_* 12 键，Y=禁/N=允）。
	AreaRules map[string]map[string]string
}

func (m *TriggerAreaEvent) Name() string { return "TriggerAreaEvent" }

func (m *TriggerAreaEvent) Init(k *Kernel) error {
	k.Bus().OnEvent("player", "OnMapChanged", m.onMapChanged)
	return nil
}

// LoadResource 区域禁则真档装载（缺档静默；fskit.ParseItems 官方方言）。
func (m *TriggerAreaEvent) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return nil
	}
	// 客户端档是 GBK（随 share 树原样进数据根），经编码自适应入口读。
	data, err := sharedata.ReadTextFile(filepath.Join(k.ResourcePath, "share", "area", "areainfo.ini"))
	if err != nil {
		return nil
	}
	rules := make(map[string]map[string]string)
	err = fskit.ParseItems(data, func(sec, key, value string) error {
		if sec == "" || key == "" || key == "Comment" {
			return nil
		}
		if rules[sec] == nil {
			rules[sec] = make(map[string]string)
		}
		rules[sec][key] = value
		return nil
	})
	if err != nil {
		return err
	}
	m.AreaRules = rules
	return nil
}

// Forbid 区域禁则查询（Y=禁）。
func (m *TriggerAreaEvent) Forbid(area, rule string) bool {
	return m.AreaRules[area]["forbid_"+rule] == "Y"
}

// onMapChanged 官方面：进区触发（区域族 Source）。
func (m *TriggerAreaEvent) onMapChanged(k *Kernel, self, from PersistID, args VarList) {
	tdm, _ := k.GetLogicModule("TriggerDataModule").(*TriggerDataModule)
	if tdm == nil {
		return
	}
	for _, t := range tdm.triggers {
		if strings.HasPrefix(t.Source, "area") {
			m.Fire(k, t, self, NullPersistID, args)
		}
	}
}

// TriggerBufferEvent 复刻官方 TriggerBufferEvent（Init @0x577760；符号 196）。
type TriggerBufferEvent struct {
	TriggerBaseModule
}

func (m *TriggerBufferEvent) Name() string { return "TriggerBufferEvent" }

func (m *TriggerBufferEvent) Init(k *Kernel) error {
	k.Bus().OnEvent("BufferContainer", "OnAfterAdd", m.onBufferAdd)
	return nil
}

func (m *TriggerBufferEvent) onBufferAdd(k *Kernel, self, from PersistID, args VarList) {
	tdm, _ := k.GetLogicModule("TriggerDataModule").(*TriggerDataModule)
	if tdm == nil {
		return
	}
	for _, t := range tdm.triggers {
		if strings.HasPrefix(t.Source, "buffer") {
			m.Fire(k, t, self, NullPersistID, args)
		}
	}
}

// TriggerItemEvent 复刻官方 TriggerItemEvent（Init @0x57B680；符号 211）。
type TriggerItemEvent struct {
	TriggerBaseModule
}

func (m *TriggerItemEvent) Name() string { return "TriggerItemEvent" }

func (m *TriggerItemEvent) Init(k *Kernel) error {
	k.Bus().OnEvent("ToolBox", "OnAfterAdd", m.onItemAdd)
	return nil
}

func (m *TriggerItemEvent) onItemAdd(k *Kernel, self, from PersistID, args VarList) {
	tdm, _ := k.GetLogicModule("TriggerDataModule").(*TriggerDataModule)
	if tdm == nil {
		return
	}
	for _, t := range tdm.triggers {
		if strings.HasPrefix(t.Source, "item") {
			m.Fire(k, t, self, NullPersistID, args)
		}
	}
}

// ---- 八 Processor（行为方言）----

// ProcessorCommonBehavior 复刻官方 ProcessorCommonBehavior（Init @0x5AB800；
// 符号 376；零内核注册）——通用行为（属性 set/add + 公告）。
type ProcessorCommonBehavior struct {
	BaseModule

	// ops 流水（测试面："common:<op>:<prop>"）。
	ops []string
}

func (m *ProcessorCommonBehavior) Name() string         { return "ProcessorCommonBehavior" }
func (m *ProcessorCommonBehavior) Init(k *Kernel) error { return nil }

// Process 通用行为执行：property set/add / notice。
func (m *ProcessorCommonBehavior) Process(k *Kernel, b StageBehavior, self, from PersistID) {
	o := k.GetGameObj(self)
	switch b.Op {
	case "set":
		if o != nil {
			if _, ok := o.Class().Prop(b.Prop); ok {
				_ = k.SetInt(self, b.Prop, b.Value)
			}
		}
	case "add":
		if o != nil {
			if _, ok := o.Class().Prop(b.Prop); ok {
				_ = k.SetInt(self, b.Prop, o.GetInt(b.Prop)+b.Value)
			}
		}
	case "notice":
		if nm, _ := k.GetLogicModule("NewsModule").(*NewsModule); nm != nil {
			nm.NxPublishNews(0, b.Text)
		}
	}
	m.ops = append(m.ops, fmt.Sprintf("common:%s:%s", b.Op, b.Prop))
}

// Ops 流水（测试面）。
func (m *ProcessorCommonBehavior) Ops() []string { return m.ops }

// ProcessorPropertyBehavior 复刻官方 ProcessorPropertyBehavior（Init @0x5E5610；
// 符号 132；零内核注册）——属性行为（与 Common 的 property 面分工：含
// 比例/上限语义）。
type ProcessorPropertyBehavior struct {
	BaseModule

	// ops 流水（测试面）。
	ops []string
}

func (m *ProcessorPropertyBehavior) Name() string         { return "ProcessorPropertyBehavior" }
func (m *ProcessorPropertyBehavior) Init(k *Kernel) error { return nil }

// Process 属性行为：ratio（按当前值百分比增减）。
func (m *ProcessorPropertyBehavior) Process(k *Kernel, b StageBehavior, self, from PersistID) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if b.Op == "ratio" {
		if _, ok := o.Class().Prop(b.Prop); ok {
			cur := o.GetInt(b.Prop)
			_ = k.SetInt(self, b.Prop, cur+cur*b.Value/100)
		}
	}
	m.ops = append(m.ops, fmt.Sprintf("property:%s:%s", b.Op, b.Prop))
}

// Ops 流水（测试面）。
func (m *ProcessorPropertyBehavior) Ops() []string { return m.ops }

// ProcessorRecordBehavior 复刻官方 ProcessorRecordBehavior（Init @0x5EFEF0；
// 符号 96；零内核注册）——记录行为（行追加/删除）。
type ProcessorRecordBehavior struct {
	BaseModule

	// ops 流水（测试面）。
	ops []string
}

func (m *ProcessorRecordBehavior) Name() string         { return "ProcessorRecordBehavior" }
func (m *ProcessorRecordBehavior) Init(k *Kernel) error { return nil }

// Process 记录行为：add（记录 b.Prop 追加 {b.Value} 行）/ clear。
func (m *ProcessorRecordBehavior) Process(k *Kernel, b StageBehavior, self, from PersistID) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if r := o.Record(b.Prop); r != nil {
		switch b.Op {
		case "add":
			if r.Rows() < r.Def().MaxRows {
				r.AppendRow(VarList{b.Value})
			}
		case "clear":
			r.Clear()
		}
	}
	m.ops = append(m.ops, fmt.Sprintf("record:%s:%s", b.Op, b.Prop))
}

// Ops 流水（测试面）。
func (m *ProcessorRecordBehavior) Ops() []string { return m.ops }

// ProcessorObjectBehavior 复刻官方 ProcessorObjectBehavior（Init @0x5D6A30；
// 符号 237；零内核注册）——对象行为（物化/销毁）。
type ProcessorObjectBehavior struct {
	BaseModule

	// ops 流水（测试面）。
	ops []string
}

func (m *ProcessorObjectBehavior) Name() string         { return "ProcessorObjectBehavior" }
func (m *ProcessorObjectBehavior) Init(k *Kernel) error { return nil }

// Process 对象行为：summon（NpcResource 物化）/ destroy。
func (m *ProcessorObjectBehavior) Process(k *Kernel, b StageBehavior, self, from PersistID) {
	switch b.Op {
	case "summon":
		if nrm, _ := k.GetLogicModule("NpcResourceModule").(*NpcResourceModule); nrm != nil {
			nrm.Spawn(k, b.Text)
		}
	case "destroy":
		if o := k.GetGameObj(self); o != nil && b.Value != 0 {
			k.RemoveObject(self)
		}
	}
	m.ops = append(m.ops, fmt.Sprintf("object:%s", b.Op))
}

// Ops 流水（测试面）。
func (m *ProcessorObjectBehavior) Ops() []string { return m.ops }

// ProcessorAIBehavior 复刻官方 ProcessorAIBehavior（Init @0x59B080；符号 148；
// 零内核注册）——AI 行为（状态迁移/仇恨面）。
type ProcessorAIBehavior struct {
	BaseModule

	// ops 流水（测试面）。
	ops []string
}

func (m *ProcessorAIBehavior) Name() string         { return "ProcessorAIBehavior" }
func (m *ProcessorAIBehavior) Init(k *Kernel) error { return nil }

// Process AI 行为：state（AIModule 迁移）/ hate（HateModule 入表）。
func (m *ProcessorAIBehavior) Process(k *Kernel, b StageBehavior, self, from PersistID) {
	switch b.Op {
	case "state":
		if am, _ := k.GetLogicModule("AIModule").(*AIModule); am != nil {
			am.Transition(k, self, b.Value)
		}
	case "hate":
		if hm, _ := k.GetLogicModule("HateModule").(*HateModule); hm != nil {
			hm.AddHate(k, self, from, b.Value)
		}
	}
	m.ops = append(m.ops, fmt.Sprintf("ai:%s", b.Op))
}

// Ops 流水（测试面）。
func (m *ProcessorAIBehavior) Ops() []string { return m.ops }

// ProcessorCloneSceneBehavior 复刻官方 ProcessorCloneSceneBehavior（Init
// @0x5A5760；符号 108；零内核注册）——副本行为（建副本/难度面——
// cloneinfo 真档门槛）。
type ProcessorCloneSceneBehavior struct {
	BaseModule

	// ops 流水（测试面）。
	ops []string
}

func (m *ProcessorCloneSceneBehavior) Name() string         { return "ProcessorCloneSceneBehavior" }
func (m *ProcessorCloneSceneBehavior) Init(k *Kernel) error { return nil }

// Process 副本行为：create（CloneSceneModule 建副本，b.Text=场景资源路径）。
func (m *ProcessorCloneSceneBehavior) Process(k *Kernel, b StageBehavior, self, from PersistID) {
	if b.Op != "create" {
		return
	}
	csm, _ := k.GetLogicModule("CloneSceneModule").(*CloneSceneModule)
	if csm == nil {
		return
	}
	info := csm.Clones[b.Text]
	if info != nil && b.Value > 0 {
		// 难度门槛（Easy 档战力——真档面；b.Value=调用面战力）。
		if power := atoi32(info["EasyPowerLevel"]); power > 0 && b.Value < power {
			m.ops = append(m.ops, fmt.Sprintf("clone-gate:%s", b.Text))
			return
		}
	}
	// 场景号：资源路径数字（clone001→1）。
	var proto int32
	for i := 0; i < len(b.Text); i++ {
		if b.Text[i] >= '0' && b.Text[i] <= '9' {
			proto = proto*10 + int32(b.Text[i]-'0')
		}
	}
	csm.CreateClone(k, proto, 1, self, 1)
	m.ops = append(m.ops, fmt.Sprintf("clone:%s", b.Text))
}

// Ops 流水（测试面）。
func (m *ProcessorCloneSceneBehavior) Ops() []string { return m.ops }

// ProcessorConditionBehavior 复刻官方 ProcessorConditionBehavior（Init
// @0x5CAE10；符号 105；零内核注册）——条件行为（运行时门：不满足则截断
// 行为链）。
type ProcessorConditionBehavior struct {
	BaseModule

	// ops 流水（测试面："cond-pass"/"cond-cut"）。
	ops []string
}

func (m *ProcessorConditionBehavior) Name() string         { return "ProcessorConditionBehavior" }
func (m *ProcessorConditionBehavior) Init(k *Kernel) error { return nil }

// Check 条件行为门（ConditionModule 求值）。
func (m *ProcessorConditionBehavior) Check(k *Kernel, c StageCondition, self PersistID) bool {
	cm, _ := k.GetLogicModule("ConditionModule").(*ConditionModule)
	ok := cm != nil && cm.Eval(k, self, c)
	if ok {
		m.ops = append(m.ops, "cond-pass")
	} else {
		m.ops = append(m.ops, "cond-cut")
	}
	return ok
}

// Ops 流水（测试面）。
func (m *ProcessorConditionBehavior) Ops() []string { return m.ops }

// ProcessorComparisionBehavior 复刻官方 ProcessorComparisionBehavior（Init
// @0x5C7570；符号 33；零内核注册）——比较行为（六则比较门）。
type ProcessorComparisionBehavior struct {
	BaseModule

	// ops 流水（测试面）。
	ops []string
}

func (m *ProcessorComparisionBehavior) Name() string         { return "ProcessorComparisionBehavior" }
func (m *ProcessorComparisionBehavior) Init(k *Kernel) error { return nil }

// Compare 六则比较（= != > >= < <=）。
func (m *ProcessorComparisionBehavior) Compare(op string, a, b int32) bool {
	m.ops = append(m.ops, fmt.Sprintf("cmp:%s", op))
	switch op {
	case "=":
		return a == b
	case "!=":
		return a != b
	case ">":
		return a > b
	case ">=":
		return a >= b
	case "<":
		return a < b
	case "<=":
		return a <= b
	}
	return false
}

// Ops 流水（测试面）。
func (m *ProcessorComparisionBehavior) Ops() []string { return m.ops }

// ---- 三 Executor + ConditionModule ----

// TriggerProcesserModule 扩展：Execute 行为链路由（八 Processor 分派——
// Kind → Processor 方言映射）。
// （TriggerProcesserModule 本体在九期 eventmods.go——此处补路由面。）

// stageRoute Kind → Processor 映射表。
func stageExecute(k *Kernel, t *StageTrigger, self, from PersistID, args VarList) {
	for _, b := range t.Behaviors {
		switch b.Kind {
		case "common":
			if p, _ := k.GetLogicModule("ProcessorCommonBehavior").(*ProcessorCommonBehavior); p != nil {
				p.Process(k, b, self, from)
			}
		case "property":
			if p, _ := k.GetLogicModule("ProcessorPropertyBehavior").(*ProcessorPropertyBehavior); p != nil {
				p.Process(k, b, self, from)
			}
		case "record":
			if p, _ := k.GetLogicModule("ProcessorRecordBehavior").(*ProcessorRecordBehavior); p != nil {
				p.Process(k, b, self, from)
			}
		case "object":
			if p, _ := k.GetLogicModule("ProcessorObjectBehavior").(*ProcessorObjectBehavior); p != nil {
				p.Process(k, b, self, from)
			}
		case "ai":
			if p, _ := k.GetLogicModule("ProcessorAIBehavior").(*ProcessorAIBehavior); p != nil {
				p.Process(k, b, self, from)
			}
		case "clone":
			if p, _ := k.GetLogicModule("ProcessorCloneSceneBehavior").(*ProcessorCloneSceneBehavior); p != nil {
				p.Process(k, b, self, from)
			}
		case "notice":
			if p, _ := k.GetLogicModule("ProcessorCommonBehavior").(*ProcessorCommonBehavior); p != nil {
				p.Process(k, StageBehavior{Op: "notice", Text: b.Text}, self, from)
			}
		}
	}
}

// ConditionModule 复刻官方 ConditionModule（Init @0x6F8F20；符号 231；
// commonmodule；零内核注册——条件求值 API）。
type ConditionModule struct {
	BaseModule

	// evals 求值计数（测试面）。
	evals int
}

func (m *ConditionModule) Name() string         { return "ConditionModule" }
func (m *ConditionModule) Init(k *Kernel) error { return nil }

// Eval 条件求值：prop 面（六则比较）——其余 Kind 缺省真（扩展面）。
func (m *ConditionModule) Eval(k *Kernel, self PersistID, c StageCondition) bool {
	m.evals++
	if c.Kind != "prop" {
		return true
	}
	o := k.GetGameObj(self)
	if o == nil {
		return false
	}
	cur := o.GetInt(c.Prop)
	switch c.Op {
	case "=":
		return cur == c.Value
	case "!=":
		return cur != c.Value
	case ">":
		return cur > c.Value
	case ">=":
		return cur >= c.Value
	case "<":
		return cur < c.Value
	case "<=":
		return cur <= c.Value
	}
	return false
}

// Evals 求值计数（测试面）。
func (m *ConditionModule) Evals() int { return m.evals }

// NoticeExecutor 复刻官方 NoticeExecutor（Init @0xA8E6C0；符号 65；
// eventspringmodule\；注册面为扫描噪声行——公告执行 API 面）。
type NoticeExecutor struct {
	BaseModule

	// Sent 公告出口（NewsModule 转发）。
	// ops 流水（测试面）。
	ops []string
}

func (m *NoticeExecutor) Name() string         { return "NoticeExecutor" }
func (m *NoticeExecutor) Init(k *Kernel) error { return nil }

// Execute 公告执行（channel + 文案）。
func (m *NoticeExecutor) Execute(k *Kernel, channel int32, text string) {
	if nm, _ := k.GetLogicModule("NewsModule").(*NewsModule); nm != nil {
		nm.NxPublishNews(channel, text)
	}
	m.ops = append(m.ops, fmt.Sprintf("notice:%d", channel))
}

// Ops 流水（测试面）。
func (m *NoticeExecutor) Ops() []string { return m.ops }

// BehaviorExecutor 复刻官方 BehaviorExecutor（Init @0xA7DAD0；符号 393；
// eventspringmodule\behaviorexecutor.cpp）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("scene","OnCreate")
//	DeclareHeartBeat ("BehaviorExecutor::HB_CheckBehavior")
type BehaviorExecutor struct {
	BaseModule

	// pending 待执行行为队列（scene 物化后排程）。
	pending []queuedBehavior
	// ops 流水（测试面）。
	ops []string
}

type queuedBehavior struct {
	behavior StageBehavior
	self     PersistID
}

func (m *BehaviorExecutor) Name() string { return "BehaviorExecutor" }

func (m *BehaviorExecutor) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("scene", "OnCreate", m.onSceneCreate)
	b.HeartBeat("BehaviorExecutor::HB_CheckBehavior", m.hbCheckBehavior)
	return nil
}

// Queue 行为入队（延迟执行面——官方 HB_CheckBehavior 泵）。
func (m *BehaviorExecutor) Queue(b StageBehavior, self PersistID) {
	m.pending = append(m.pending, queuedBehavior{b, self})
}

// onSceneCreate 官方：场景物化 → 排队行为落地（副本开场演出面）。
func (m *BehaviorExecutor) onSceneCreate(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, "scene")
}

// hbCheckBehavior 官方：行为泵——顺序落一条/拍（顺序保证；无排队即空转）。
func (m *BehaviorExecutor) hbCheckBehavior(k *Kernel, self PersistID, beats int32) {
	if len(m.pending) == 0 || self.ID() != 1 {
		return // 单锚对象泵（首物化对象——避免逐对象重复落）
	}
	queued := m.pending[0]
	m.pending = m.pending[1:]
	stageExecute(k, &StageTrigger{ID: "queued", Behaviors: []StageBehavior{queued.behavior}}, queued.self, NullPersistID, nil)
	m.ops = append(m.ops, "drain")
}

// Ops 流水（测试面）。
func (m *BehaviorExecutor) Ops() []string { return m.ops }

// ConditionExecutor 复刻官方 ConditionExecutor（Init @0xA79530；符号 60；
// eventspringmodule\）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddIntCommandHook ("scene",752=0x2F0) / ("AttackNpc"/"BattleCarrier"/
//	    "player",616) / ("player",932=0x3A4 / 930=0x3A2)
//	AddEventCallback ×1（扫描噪声）
type ConditionExecutor struct {
	BaseModule

	// ops 流水（测试面）。
	ops []string
}

func (m *ConditionExecutor) Name() string { return "ConditionExecutor" }

func (m *ConditionExecutor) Init(k *Kernel) error {
	b := k.Bus()
	b.OnIntCommand("scene", 752, m.onCheck)
	b.OnIntCommand("AttackNpc", 616, m.onBeKilled)
	b.OnIntCommand("BattleCarrier", 616, m.onBeKilled)
	b.OnIntCommand("player", 616, m.onBeKilled)
	b.OnIntCommand("player", 932, m.onUpdate)
	b.OnIntCommand("player", 930, m.onCheck)
	return nil
}

// onCheck 官方 @0xA79660（cmd 752/930）：条件检查（args: 条件面）。
func (m *ConditionExecutor) onCheck(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, "check")
}

// onBeKilled 官方 @0xA79800（616）：被杀条件推进（击杀类条件）。
func (m *ConditionExecutor) onBeKilled(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, "killed")
}

// onUpdate 官方 @0xA79D00（cmd 932=0x3A4）：通用进度推进。
func (m *ConditionExecutor) onUpdate(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, "update")
}

// Ops 流水（测试面）。
func (m *ConditionExecutor) Ops() []string { return m.ops }
