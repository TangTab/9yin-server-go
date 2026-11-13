// Package skillscript 装载技能模块脚本（scripts/module/skill/，自参考服拷贝）：
//
//   - 逐技能覆盖模块 `<技能ID>.lua`：M.compile(def) 编译期增量覆盖（整组
//     取代该技能的 INI 默认编译效果）+ M.on_hit(ctx) 出招倍率运行期钩子；
//   - effects_data.lua：全技能效果汇总表（local E … return E，生成器产出，
//     6,433 条），装载时整表快照为 Go 目录供查询。
//
// 宿主机制由 internal/modulevm 提供（坏脚本只禁用该模块、VM 不重建）。
// Overlay/HitRatio/EffectData 的 nil / (nil, false) 均表示"该技能无脚本
// 数据"——调用方（技能目录编译，落地时接线）回退 INI 默认编译，出招链
// 永不被脚本层拒绝。
//
// 契约要点（参考服 skill-script-framework.md skill 节）：随机数调用方预掷
// （on_hit 的 rand.r_adjust，每调用恒 1 枚——无钩子技能 RNG 序列零扰动）；
// by_level 的 damage_amount = 0 是显式有效值（缺字段才沿用 L1 基准）。
package skillscript

import (
	"sort"
	"sync"
	"time"

	"github.com/iceisfun/golua/v2/vm"
	"github.com/local/9yin-go-server/internal/modulevm"
)

// Logger 与标准库 log.Printf 兼容；宿主注入，包内不直接打日志。
type Logger = modulevm.Logger

// effectsDataModule 是效果汇总表模块名（effects_data.lua 的文件名主干）。
const effectsDataModule = "effects_data"

// descEffectsModule 是描述解析汇总表模块名（desc_effects.lua 的文件名主干）。
const descEffectsModule = "desc_effects"

// Scripts 是技能脚本宿主的域包装。零值不可用，经 New 创建。
type Scripts struct {
	host *modulevm.Host

	// effectsData/descData 在装载时（持 host 锁的 EachModule 回调内）
	// 整表快照；之后只读。mu 保护其生命周期（Close/Reload 置 nil）。
	mu          sync.Mutex
	effectsData map[string]*EffectEntry
	descData    map[string]*DescEntry
	overlays    []string // 存在 M.compile 的覆盖模块（排序后，目录用）
	onHits      []string // 存在 M.on_hit 的模块（排序后，目录用）
}

// New 创建技能脚本宿主。dir 为空表示禁用：所有查询未命中。
func New(dir string, logger Logger) *Scripts {
	return &Scripts{host: modulevm.New(dir, logger)}
}

// SetTimeout 调整单次调用预算（测试用）。
func (s *Scripts) SetTimeout(d time.Duration) { s.host.SetTimeout(d) }

// Enabled 报告宿主是否配置了脚本目录（-skill-scripts 非空）。
func (s *Scripts) Enabled() bool { return s.host.Enabled() }

// Loaded 返回最近一次装载成功的模块数（覆盖模块 + effects_data）。
func (s *Scripts) Loaded() int { return s.host.Loaded() }

// Count 是 Loaded 的别名，与目录型注册表的命名对齐。
func (s *Scripts) Count() int { return s.Loaded() }

// Broken 返回被禁用的模块名。
func (s *Scripts) Broken() []string { return s.host.Broken() }

// Reload 重新装载脚本目录（GM 热重载）。
func (s *Scripts) Reload() (int, error) { return s.Load() }

// Close 释放 VM 并清空效果快照。
func (s *Scripts) Close() error {
	s.mu.Lock()
	s.effectsData = nil
	s.descData = nil
	s.overlays = nil
	s.onHits = nil
	s.mu.Unlock()
	return s.host.Close()
}

// Load 装载脚本目录：覆盖模块不要求 M.meta（参考服覆盖脚本即无 meta），
// effects_data/desc_effects 缺失或坏只导致对应查询恒 miss（降级 INI
// 编译），不作为装载错误。
func (s *Scripts) Load() (int, error) {
	loaded, loadErr := s.host.Load()
	effects := make(map[string]*EffectEntry)
	descs := make(map[string]*DescEntry)
	var overlays, onHits []string
	s.host.EachModule(func(name string, table vm.LuaTable) error {
		if name == effectsDataModule {
			s.snapshotEffects(table, effects) // 畸形条目逐条 WARN 并跳过
			return nil
		}
		if name == descEffectsModule {
			s.snapshotDescEffects(table, descs) // 畸形条目逐条 WARN 并跳过
			return nil
		}
		if modulevm.Field(table, "compile").IsCallable() {
			overlays = append(overlays, name)
		}
		if modulevm.Field(table, "on_hit").IsCallable() {
			onHits = append(onHits, name)
		}
		return nil
	})
	// 畸形条目在 snapshot* 内逐条 WARN 并跳过；effects_data/desc_effects
	// 是纯数据模块（无可调函数），不 MarkBroken——合法条目照常服务。
	sort.Strings(overlays)
	sort.Strings(onHits)
	s.mu.Lock()
	s.effectsData = effects
	s.descData = descs
	s.overlays = overlays
	s.onHits = onHits
	s.mu.Unlock()
	return loaded, loadErr
}

// Overlays 返回存在 M.compile 的技能 ID 列表（排序后）。
func (s *Scripts) Overlays() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.overlays...)
}

// HasOverlay 报告技能是否有覆盖模块（存在且未被禁用）。
func (s *Scripts) HasOverlay(skillID string) bool {
	return s.host.Func(skillID, "compile").IsCallable()
}

// BuffDef 是 M.compile 的输入 def：INI 编译出的 buf_<技能ID> 精确 buff 快照。
type BuffDef struct {
	ConfigID   string
	StaticData int64
	Level      int64
	LifetimeMS int64
}

// DamageSpec 是伤害/段数（取整组时覆盖 INI 编译值）。
type DamageSpec struct {
	Amount int64
	Waves  int64
}

// StanceVariant 是架势轮换的一个变体。
type StanceVariant struct {
	ID         string
	StaticData int64
	Level      int64
	LifeMS     int64
	Slot       int64
}

// StanceCycle 是架势轮换指令（逐次施放循环变体）。
type StanceCycle struct {
	Variants []StanceVariant
}

// ChannelSpec 是 M.compile 的 channel 指令（通道型引导技参数，挑战
// .pcapng 2026-11-12 官服定版语义；消费方 cmd/protocol-probe 的
// channelSkillSpec——SP 逐波扣、m169 挂/摘对由波次引擎按此驱动）。
type ChannelSpec struct {
	Waves          int64  // 波数上限（官服 8）
	FirstWaveMS    int64  // 首波延迟（官服 180ms）
	IntervalMS     int64  // 波间隔（官服 440ms）
	SpPerWave      int64  // 每波扣怒（官服 10；0=免费波）
	VictimBuff     string // 每波瞬挂瞬摘的受害者 m169 buff（buf_<技能ID>）
	SelfBuff       string // 自身 m169 buff 对（buf_<技能ID>_03）
	SelfBuffStatic int64  // BufferInfo 槽 CSV 首段 StaticData（9019=0x233B 实证）
}

// BuffSpec 是一条精确 buff（自增益或目标效果共用的基形态）。
type BuffSpec struct {
	ConfigID   string
	StaticData int64
	Level      int64
	LifeMS     int64
	Slot       int64 // 0/缺省=自动找空槽
}

// TargetEffect 是目标效果（有害精确 buff 的目标控制）。
type TargetEffect struct {
	BuffSpec
	Control bool // 受控到租约终点
	Frame   bool // 加发 msg169 目标 debuff 帧
}

// Overlay 是 M.compile 返回的增量覆盖。nil 字段/零值表示该增量未下发；
// 合并进技能编译效果由调用方负责（增量语义）。
type Overlay struct {
	SelfBuffLeaseMS int64          // self_buff_lease_ms：自 buff 租约覆盖
	SelfBuffSlot    int64          // self_buff_slot：指定 BufferInfo 槽位（0=自动）
	RedArmor        bool           // red_armor：红霸体（起手霸体帧）
	StanceCycle     *StanceCycle   // stance_cycle
	Damage          *DamageSpec    // damage：伤害/段数覆盖
	SelfBuffs       []BuffSpec     // self_buffs：多自 buff（整组取代）
	TargetEffects   []TargetEffect // target_effects（整组取代）
	Channel         *ChannelSpec   // channel：通道型引导参数（spCost 预扣清零、逐波扣）
}

// LevelDelta 是 by_level[level] 与 L1 基准的差集。字段全集按 effects_data
// 实测：damage_amount/damage_waves/target_life_ms/self_life_ms；显式 0 有效
// （缺字段才沿用基准），故用指针区分。
type LevelDelta struct {
	DamageAmount *int64
	DamageWaves  *int64
	TargetLifeMS *int64
	SelfLifeMS   *int64
}

// EffectEntry 是 effects_data 的一个技能条目（E["<技能ID>"]）。
type EffectEntry struct {
	Damage        *DamageSpec
	SelfBuffs     []BuffSpec
	TargetEffects []TargetEffect
	ByLevel       map[int64]*LevelDelta
}

// DescCompile 是 desc_effects 的编译期静态轴（"周围N米/最多斩击N次/
// 黄霸体/不可移动/只能对空"这类不含触发条件的部分）。零值字段=描述未标注。
type DescCompile struct {
	AOEShape    string  // circle|sector|rect
	AOERadiusM  float64
	AOEWidthM   float64
	MaxHits     int64   // 最多斩击/连击 N 次（寒心恨雪族波数上限的描述来源）
	CastArmor   string  // red|yellow（出招过程为红/黄霸体状态）
	CanMove     *bool   // （出招时不可移动）显式标注才非 nil——false 是有效值
	CanQinggong bool    // （可施展/利用轻功…）
	RushNote    bool    // 轻功快速接近旁注（冲锋族标记）
	Air         string  // only=只能针对空中目标 / never=无法在空中施展
}

// DescCondition 是规则的触发条件（"未招架/已被破防/存有'X'状态/无视招架"）。
type DescCondition struct {
	Unparried    bool
	ParryBroken  bool
	TargetState  string
	Unparryable  bool
}

// DescRule 是一条触发规则（平铺载荷，键与生成器一一对应；mp_restore
// 显式 0 有效——"恢复内力"未标数值——故 MPRestoreSet 区分存在性）。
type DescRule struct {
	PerHit    bool    // 每击
	Chance    float64 // 有 N% 几率（0=未标注；二期接线走调用方预掷）
	Condition *DescCondition

	ControlKind     string // stun/coma/dizzy/launch/knockdown/root/silence/ban_qinggong/slow/stiffen
	ControlDurMS    int64
	ControlWithinMS int64 // "6秒内出招后封招3秒"的外层窗口

	DisplacementKind      string  // push 击退（受害者后移）/ pull 拉近
	DisplacementDistanceM float64 // 0=描述未标距离

	MPDrain      bool // 流失内力（目标减蓝）
	MPDrainDurMS int64
	MPRestoreSet bool
	MPRestore    int64
	Heal         bool

	ApplyBuff     string // 给XX添加"X"（状态名反查的 buf_ 配置；未反查到=中文名）
	ApplyResolved bool   // ApplyBuff 是否成功落到 buf_ 配置
	ApplyStacks   int64
	ApplyLifeMS   int64
	BurstAt       int64 // 叠加至 N 层后爆发
	BurstKind     string
	BurstDurMS    int64

	StackBuff     string // 被攻击则叠加（感染族）
	StackResolved bool
	StackPerHit   int64

	PeriodicEveryMS int64 // 每 N 秒
	PeriodicAOE     bool  // 给自身周围
}

// DescEntry 是 desc_effects.lua 的一个技能条目（旁路数据层：仅查询，
// 不进 effects_data 优先级链；消费接线见 docs/skill-desc-ir.md 路线节）。
type DescEntry struct {
	Compile      *DescCompile
	OnHit        []*DescRule
	OnParry      []*DescRule
	OnParryBreak []*DescRule
	OnVictim     []*DescRule
	OnPeriodic   []*DescRule
	SelfState    []*DescRule
	ByLevel      map[int64]map[string]int64 // 等级 → 点路径 → 数值（差集）
	Unparsed     []string                   // 含关键词但无模板命中的段落留痕
}

// Overlay 调用技能的 M.compile。nil 表示该技能无覆盖（无模块/无 compile/
// 已禁用/返回 nil/返回值全无效）——调用方走 effects_data → INI 默认编译。
func (s *Scripts) Overlay(skillID string, def *BuffDef) *Overlay {
	if def == nil {
		def = &BuffDef{}
	}
	result, ok := s.host.Invoke(skillID, []string{"compile"}, func(v *vm.VM) vm.Value {
		defTable := vm.NewEmptyTable()
		defTable.SetString("config_id", vm.NewString(def.ConfigID))
		defTable.SetString("static_data", vm.NewInt(def.StaticData))
		defTable.SetString("level", vm.NewInt(def.Level))
		defTable.SetString("lifetime_ms", vm.NewInt(def.LifetimeMS))
		return vm.NewTable(defTable)
	})
	if !ok || result.IsNil() || !result.IsTable() {
		return nil
	}
	return s.convertOverlay(skillID, result.AsTable())
}

// HitCtx 是 M.on_hit 的输入（只读，调用方预填；每调用恒 1 枚预掷）。
type HitCtx struct {
	Ratio   float64 // 本波倍率
	Level   int64   // 技能等级
	RAdjust float64 // rand.r_adjust：倍率调整掷 [0,1)
}

// HitRatio 调用技能的 M.on_hit，返回调整后的倍率（夹 [0,100]）。nil 表示
// 不调整（无钩子/返回 nil/非正数）——无钩子技能 RNG 序列零扰动。
func (s *Scripts) HitRatio(skillID string, ctx *HitCtx) *float64 {
	if ctx == nil {
		ctx = &HitCtx{}
	}
	result, ok := s.host.Invoke(skillID, []string{"on_hit"}, func(v *vm.VM) vm.Value {
		t := vm.NewEmptyTable()
		t.SetString("ratio", vm.NewFloat(ctx.Ratio))
		t.SetString("level", vm.NewInt(ctx.Level))
		rand := vm.NewEmptyTable()
		rand.SetString("r_adjust", vm.NewFloat(ctx.RAdjust))
		t.SetString("rand", vm.NewTable(rand))
		return vm.NewTable(t)
	})
	if !ok || result.IsNil() || !result.IsTable() {
		return nil
	}
	ratio, ratioOK := modulevm.LuaFloat(modulevm.Field(result.AsTable(), "ratio"))
	if !ratioOK || ratio <= 0 {
		return nil
	}
	if ratio > 100 {
		ratio = 100
	}
	return &ratio
}

// EffectData 查询效果汇总表（装载时快照）。nil 表示无条目——调用方回退
// INI 默认编译。返回值为快照引用，调用方只读。
func (s *Scripts) EffectData(skillID string) *EffectEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.effectsData[skillID]
}

// EffectDataCount 返回效果汇总表条目数（目录/冒烟用）。
func (s *Scripts) EffectDataCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.effectsData)
}

// DescEffects 查询描述解析汇总表（装载时快照）。nil 表示无条目——
// 调用方按无描述数据处理。返回值为快照引用，调用方只读。
func (s *Scripts) DescEffects(skillID string) *DescEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.descData[skillID]
}

// DescEffectsCount 返回描述解析表条目数（目录/冒烟用）。
func (s *Scripts) DescEffectsCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.descData)
}
