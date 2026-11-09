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

// Scripts 是技能脚本宿主的域包装。零值不可用，经 New 创建。
type Scripts struct {
	host *modulevm.Host

	// effectsData 在装载时（持 host 锁的 EachModule 回调内）整表快照；
	// 之后只读。mu 保护其生命周期（Close/Reload 置 nil）。
	mu          sync.Mutex
	effectsData map[string]*EffectEntry
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
	s.overlays = nil
	s.onHits = nil
	s.mu.Unlock()
	return s.host.Close()
}

// Load 装载脚本目录：覆盖模块不要求 M.meta（参考服覆盖脚本即无 meta），
// effects_data 缺失或坏只导致 EffectData 查询恒 miss（降级 INI 编译），
// 不作为装载错误。
func (s *Scripts) Load() (int, error) {
	loaded, loadErr := s.host.Load()
	effects := make(map[string]*EffectEntry)
	var overlays, onHits []string
	s.host.EachModule(func(name string, table vm.LuaTable) error {
		if name == effectsDataModule {
			s.snapshotEffects(table, effects) // 畸形条目逐条 WARN 并跳过
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
	// 畸形条目在 snapshotEffects 内逐条 WARN 并跳过；effects_data 是纯数据
	// 模块（无可调函数），不 MarkBroken——合法条目照常服务。
	sort.Strings(overlays)
	sort.Strings(onHits)
	s.mu.Lock()
	s.effectsData = effects
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
