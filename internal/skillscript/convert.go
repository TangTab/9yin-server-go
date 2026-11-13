package skillscript

import (
	"github.com/iceisfun/golua/v2/vm"
	"github.com/local/9yin-go-server/internal/modulevm"
)

// convertOverlay 按白名单转换 M.compile 返回表。返回 nil 表示无有效增量。
// 白名单外的键与畸形值 WARN 丢弃；合并语义由调用方实现（增量覆盖）。
func (s *Scripts) convertOverlay(skillID string, table vm.LuaTable) *Overlay {
	overlay := &Overlay{}
	set := false
	key := vm.Nil
	for {
		nextKey, value, err := table.Next(key)
		if err != nil || nextKey.IsNil() {
			break
		}
		key = nextKey
		if !nextKey.IsString() {
			s.host.WarnOnce(skillID+":key", "skillscript: %s compile 返回非字符串键 %s，已丢弃", skillID, modulevm.ValueString(nextKey))
			continue
		}
		name := nextKey.AsString()
		switch name {
		case "self_buff_lease_ms":
			ms, ok := modulevm.LuaInt(value)
			if !ok || ms <= 0 {
				s.host.WarnOnce(skillID+":"+name, "skillscript: %s self_buff_lease_ms %s 非正/畸形，已丢弃", skillID, modulevm.ValueString(value))
				continue
			}
			overlay.SelfBuffLeaseMS, set = ms, true
		case "self_buff_slot":
			slot, ok := modulevm.LuaInt(value)
			if !ok || slot < 0 {
				s.host.WarnOnce(skillID+":"+name, "skillscript: %s self_buff_slot %s 畸形，已丢弃", skillID, modulevm.ValueString(value))
				continue
			}
			overlay.SelfBuffSlot, set = slot, true
		case "red_armor":
			if value.IsBool() {
				overlay.RedArmor, set = value.AsBool(), true
			} else {
				s.host.WarnOnce(skillID+":"+name, "skillscript: %s red_armor %s 非布尔，已丢弃", skillID, modulevm.ValueString(value))
			}
		case "stance_cycle":
			if cycle, ok := s.convertStanceCycle(skillID, value); ok {
				overlay.StanceCycle, set = cycle, true
			}
		case "damage":
			if damage, ok := s.convertDamage(skillID, "compile", value); ok {
				overlay.Damage, set = damage, true
			}
		case "self_buffs":
			if buffs, ok := s.convertBuffList(skillID, name, value); ok {
				overlay.SelfBuffs, set = buffs, true
			}
		case "target_effects":
			if effects, ok := s.convertTargetEffects(skillID, name, value); ok {
				overlay.TargetEffects, set = effects, true
			}
		case "channel":
			if channel, ok := s.convertChannelSpec(skillID, value); ok {
				overlay.Channel, set = channel, true
			}
		default:
			s.host.WarnOnce(skillID+":"+name, "skillscript: %s compile 返回白名单外指令 %s=%s，已丢弃", skillID, name, modulevm.ValueString(value))
		}
	}
	if !set {
		return nil
	}
	return overlay
}

func (s *Scripts) convertStanceCycle(skillID string, value vm.Value) (*StanceCycle, bool) {
	if !value.IsTable() {
		s.host.WarnOnce(skillID+":stance_cycle", "skillscript: %s stance_cycle %s 非表，整条丢弃", skillID, modulevm.ValueString(value))
		return nil, false
	}
	variantsTable := modulevm.Field(value.AsTable(), "variants")
	if !variantsTable.IsTable() {
		s.host.WarnOnce(skillID+":stance_cycle", "skillscript: %s stance_cycle.variants 缺失/非表，整条丢弃", skillID)
		return nil, false
	}
	n := variantsTable.AsTable().Len()
	if n < 2 {
		s.host.WarnOnce(skillID+":stance_cycle", "skillscript: %s stance_cycle.variants 需 ≥2 个变体，整条丢弃", skillID)
		return nil, false
	}
	variants := make([]StanceVariant, 0, n)
	for i := 1; i <= n; i++ {
		entry := modulevm.Index(variantsTable.AsTable(), i)
		if !entry.IsTable() {
			s.host.WarnOnce(skillID+":stance_cycle", "skillscript: %s stance_cycle.variants[%d] 非表，已丢弃", skillID, i)
			continue
		}
		table := entry.AsTable()
		id := modulevm.Field(table, "id").AsString()
		life, lifeOK := modulevm.LuaInt(modulevm.Field(table, "life_ms"))
		if id == "" || !lifeOK || life <= 0 {
			s.host.WarnOnce(skillID+":stance_cycle", "skillscript: %s stance_cycle.variants[%d] {id=%q} 畸形，已丢弃", skillID, i, id)
			continue
		}
		staticData, _ := modulevm.LuaInt(modulevm.Field(table, "static_data"))
		level, _ := modulevm.LuaInt(modulevm.Field(table, "level"))
		slot, _ := modulevm.LuaInt(modulevm.Field(table, "slot"))
		variants = append(variants, StanceVariant{ID: id, StaticData: staticData, Level: level, LifeMS: life, Slot: slot})
	}
	if len(variants) < 2 {
		return nil, false
	}
	return &StanceCycle{Variants: variants}, true
}

// convertChannelSpec 解析 channel{waves, first_wave_ms, interval_ms,
// sp_per_wave, victim_buff, self_buff, self_buff_static}。任一字段畸形
// 整条丢弃（波次引擎参数错误会导致 SP/帧序错乱，不做部分采纳）。
func (s *Scripts) convertChannelSpec(skillID string, value vm.Value) (*ChannelSpec, bool) {
	if !value.IsTable() {
		s.host.WarnOnce(skillID+":channel", "skillscript: %s channel %s 非表，整条丢弃", skillID, modulevm.ValueString(value))
		return nil, false
	}
	table := value.AsTable()
	waves, wavesOK := modulevm.LuaInt(modulevm.Field(table, "waves"))
	firstWave, firstOK := modulevm.LuaInt(modulevm.Field(table, "first_wave_ms"))
	interval, intervalOK := modulevm.LuaInt(modulevm.Field(table, "interval_ms"))
	spPerWave, spOK := modulevm.LuaInt(modulevm.Field(table, "sp_per_wave"))
	victimBuff := modulevm.Field(table, "victim_buff").AsString()
	selfBuff := modulevm.Field(table, "self_buff").AsString()
	staticData, staticOK := modulevm.LuaInt(modulevm.Field(table, "self_buff_static"))
	// 上限 32 波/60s：官服定版 8 波 440ms，留出调参余量同时拦截把
	// AfterFunc 队列灌爆的笔误（waves=8000 之类）。
	if !wavesOK || waves < 1 || waves > 32 {
		s.host.WarnOnce(skillID+":channel", "skillscript: %s channel.waves %s 须为 1..32，整条丢弃",
			skillID, modulevm.ValueString(modulevm.Field(table, "waves")))
		return nil, false
	}
	if !firstOK || firstWave <= 0 || firstWave > 60000 || !intervalOK || interval <= 0 || interval > 60000 {
		s.host.WarnOnce(skillID+":channel", "skillscript: %s channel.first_wave_ms/interval_ms 须为 (0,60000] 毫秒，整条丢弃", skillID)
		return nil, false
	}
	if !spOK || spPerWave < 0 {
		s.host.WarnOnce(skillID+":channel", "skillscript: %s channel.sp_per_wave %s 须 ≥0，整条丢弃",
			skillID, modulevm.ValueString(modulevm.Field(table, "sp_per_wave")))
		return nil, false
	}
	if victimBuff == "" || selfBuff == "" {
		s.host.WarnOnce(skillID+":channel", "skillscript: %s channel.victim_buff/self_buff 不得为空，整条丢弃", skillID)
		return nil, false
	}
	if !staticOK || staticData < 0 {
		s.host.WarnOnce(skillID+":channel", "skillscript: %s channel.self_buff_static %s 须 ≥0，整条丢弃",
			skillID, modulevm.ValueString(modulevm.Field(table, "self_buff_static")))
		return nil, false
	}
	return &ChannelSpec{
		Waves: waves, FirstWaveMS: firstWave, IntervalMS: interval, SpPerWave: spPerWave,
		VictimBuff: victimBuff, SelfBuff: selfBuff, SelfBuffStatic: staticData,
	}, true
}

// convertDamage 解析 damage{amount, waves}。
func (s *Scripts) convertDamage(owner, field string, value vm.Value) (*DamageSpec, bool) {
	if !value.IsTable() {
		s.host.WarnOnce(owner+":"+field, "skillscript: %s %s.damage %s 非表，整条丢弃", owner, field, modulevm.ValueString(value))
		return nil, false
	}
	table := value.AsTable()
	amount, amountOK := modulevm.LuaInt(modulevm.Field(table, "amount"))
	waves, wavesOK := modulevm.LuaInt(modulevm.Field(table, "waves"))
	if !amountOK || amount < 0 || !wavesOK || waves < 1 {
		s.host.WarnOnce(owner+":"+field, "skillscript: %s %s.damage {amount=%s, waves=%s} 畸形，整条丢弃",
			owner, field, modulevm.ValueString(modulevm.Field(table, "amount")), modulevm.ValueString(modulevm.Field(table, "waves")))
		return nil, false
	}
	return &DamageSpec{Amount: amount, Waves: waves}, true
}

func (s *Scripts) convertBuffList(owner, field string, value vm.Value) ([]BuffSpec, bool) {
	if !value.IsTable() {
		s.host.WarnOnce(owner+":"+field, "skillscript: %s %s %s 非表，整条丢弃", owner, field, modulevm.ValueString(value))
		return nil, false
	}
	table := value.AsTable()
	n := table.Len()
	if n <= 0 {
		return nil, false
	}
	buffs := make([]BuffSpec, 0, n)
	for i := 1; i <= n; i++ {
		if spec, ok := s.readBuffSpec(modulevm.Index(table, i)); ok {
			buffs = append(buffs, spec)
			continue
		}
		s.host.WarnOnce(owner+":"+field, "skillscript: %s %s[%d] 畸形，已丢弃", owner, field, i)
	}
	if len(buffs) == 0 {
		return nil, false
	}
	return buffs, true
}

func (s *Scripts) convertTargetEffects(owner, field string, value vm.Value) ([]TargetEffect, bool) {
	if !value.IsTable() {
		s.host.WarnOnce(owner+":"+field, "skillscript: %s %s %s 非表，整条丢弃", owner, field, modulevm.ValueString(value))
		return nil, false
	}
	table := value.AsTable()
	n := table.Len()
	if n <= 0 {
		return nil, false
	}
	effects := make([]TargetEffect, 0, n)
	for i := 1; i <= n; i++ {
		entry := modulevm.Index(table, i)
		if !entry.IsTable() {
			s.host.WarnOnce(owner+":"+field, "skillscript: %s %s[%d] 非表，已丢弃", owner, field, i)
			continue
		}
		spec, ok := s.readBuffSpec(entry)
		if !ok {
			s.host.WarnOnce(owner+":"+field, "skillscript: %s %s[%d] 畸形，已丢弃", owner, field, i)
			continue
		}
		inner := entry.AsTable()
		control := modulevm.Field(inner, "control")
		frame := modulevm.Field(inner, "frame")
		effects = append(effects, TargetEffect{
			BuffSpec: spec,
			Control:  control.IsBool() && control.AsBool(),
			Frame:    frame.IsBool() && frame.AsBool(),
		})
	}
	if len(effects) == 0 {
		return nil, false
	}
	return effects, true
}

// readBuffSpec 读一条精确 buff：config_id 必填、life_ms 须正；static_data/
// level/slot 缺省 0（slot 0=自动找空槽）。
func (s *Scripts) readBuffSpec(value vm.Value) (BuffSpec, bool) {
	if !value.IsTable() {
		return BuffSpec{}, false
	}
	table := value.AsTable()
	spec := BuffSpec{ConfigID: modulevm.Field(table, "config_id").AsString()}
	life, lifeOK := modulevm.LuaInt(modulevm.Field(table, "life_ms"))
	if spec.ConfigID == "" || !lifeOK || life <= 0 {
		return BuffSpec{}, false
	}
	spec.LifeMS = life
	spec.StaticData, _ = modulevm.LuaInt(modulevm.Field(table, "static_data"))
	spec.Level, _ = modulevm.LuaInt(modulevm.Field(table, "level"))
	spec.Slot, _ = modulevm.LuaInt(modulevm.Field(table, "slot"))
	return spec, true
}

// snapshotEffects 在装载期（持 host 锁）把效果汇总表整表转为 Go 快照。
// 畸形条目 WARN 并跳过；返回畸形计数（>0 时 effects_data 模块被标记禁用，
// 但快照内合法条目照常服务）。
func (s *Scripts) snapshotEffects(table vm.LuaTable, out map[string]*EffectEntry) int {
	malformed := 0
	key := vm.Nil
	for {
		nextKey, value, err := table.Next(key)
		if err != nil || nextKey.IsNil() {
			break
		}
		key = nextKey
		if !nextKey.IsString() || !value.IsTable() {
			malformed++
			s.host.Warnf("skillscript: effects_data 非法条目 %s=%s，已跳过", modulevm.ValueString(nextKey), modulevm.ValueString(value))
			continue
		}
		skillID := nextKey.AsString()
		entry := &EffectEntry{}
		entryTable := value.AsTable()
		entryKey := vm.Nil
		for {
			entryNext, entryValue, entryErr := entryTable.Next(entryKey)
			if entryErr != nil || entryNext.IsNil() {
				break
			}
			entryKey = entryNext
			if !entryNext.IsString() {
				continue
			}
			switch entryNext.AsString() {
			case "damage":
				if damage, ok := s.convertDamage(skillID, "effects_data", entryValue); ok {
					entry.Damage = damage
				}
			case "self_buffs":
				if buffs, ok := s.convertBuffList(skillID, "self_buffs", entryValue); ok {
					entry.SelfBuffs = buffs
				}
			case "target_effects":
				if effects, ok := s.convertTargetEffects(skillID, "target_effects", entryValue); ok {
					entry.TargetEffects = effects
				}
			case "by_level":
				if deltas, ok := s.convertByLevel(skillID, entryValue); ok {
					entry.ByLevel = deltas
				}
			}
		}
		if entry.Damage == nil && entry.SelfBuffs == nil && entry.TargetEffects == nil && entry.ByLevel == nil {
			malformed++
			s.host.Warnf("skillscript: effects_data[%s] 无任何有效字段，已跳过", skillID)
			continue
		}
		out[skillID] = entry
	}
	return malformed
}

// snapshotDescEffects 在装载期（持 host 锁）把描述解析表整表转为 Go
// 快照。畸形条目 WARN 并跳过（键须为技能 ID、值须为表）。
func (s *Scripts) snapshotDescEffects(table vm.LuaTable, out map[string]*DescEntry) {
	key := vm.Nil
	for {
		nextKey, value, err := table.Next(key)
		if err != nil || nextKey.IsNil() {
			break
		}
		key = nextKey
		if !nextKey.IsString() || !value.IsTable() {
			s.host.Warnf("skillscript: desc_effects 非法条目 %s=%s，已跳过", modulevm.ValueString(nextKey), modulevm.ValueString(value))
			continue
		}
		skillID := nextKey.AsString()
		entry := &DescEntry{}
		entryTable := value.AsTable()
		entryKey := vm.Nil
		for {
			entryNext, entryValue, entryErr := entryTable.Next(entryKey)
			if entryErr != nil || entryNext.IsNil() {
				break
			}
			entryKey = entryNext
			if !entryNext.IsString() {
				continue
			}
			switch entryNext.AsString() {
			case "compile":
				if compile, ok := s.convertDescCompile(skillID, entryValue); ok {
					entry.Compile = compile
				}
			case "on_hit", "on_parry", "on_parry_break", "on_victim", "on_periodic", "self_state":
				rules := s.convertDescRules(skillID, entryNext.AsString(), entryValue)
				switch entryNext.AsString() {
				case "on_hit":
					entry.OnHit = rules
				case "on_parry":
					entry.OnParry = rules
				case "on_parry_break":
					entry.OnParryBreak = rules
				case "on_victim":
					entry.OnVictim = rules
				case "on_periodic":
					entry.OnPeriodic = rules
				case "self_state":
					entry.SelfState = rules
				}
			case "by_level":
				if deltas := s.convertDescByLevel(skillID, entryValue); len(deltas) > 0 {
					entry.ByLevel = deltas
				}
			case "unparsed":
				if list, ok := s.convertStringList(entryValue); ok {
					entry.Unparsed = list
				}
			}
		}
		if entry.Compile == nil && entry.Unparsed == nil && entry.OnHit == nil &&
			entry.OnParry == nil && entry.OnParryBreak == nil && entry.OnVictim == nil &&
			entry.OnPeriodic == nil && entry.SelfState == nil && entry.ByLevel == nil {
			s.host.Warnf("skillscript: desc_effects[%s] 无任何有效字段，已跳过", skillID)
			continue
		}
		out[skillID] = entry
	}
}

// convertDescCompile 解析 compile{aoe{…}, max_hits, cast_armor, can_move,
// can_qinggong, rush_note, air}。can_move 是三态（false 有效）故 *bool。
func (s *Scripts) convertDescCompile(skillID string, value vm.Value) (*DescCompile, bool) {
	if !value.IsTable() {
		s.host.WarnOnce(skillID+":desc.compile", "skillscript: %s desc compile %s 非表，已丢弃", skillID, modulevm.ValueString(value))
		return nil, false
	}
	compile := &DescCompile{}
	table := value.AsTable()
	if aoe := modulevm.Field(table, "aoe"); aoe.IsTable() {
		compile.AOEShape = modulevm.Field(aoe.AsTable(), "aoe_shape").AsString()
		compile.AOERadiusM, _ = modulevm.LuaFloat(modulevm.Field(aoe.AsTable(), "aoe_radius_m"))
		compile.AOEWidthM, _ = modulevm.LuaFloat(modulevm.Field(aoe.AsTable(), "aoe_width_m"))
	}
	compile.MaxHits, _ = modulevm.LuaInt(modulevm.Field(table, "max_hits"))
	compile.CastArmor = modulevm.Field(table, "cast_armor").AsString()
	if canMove := modulevm.Field(table, "can_move"); canMove.IsBool() {
		value := canMove.AsBool()
		compile.CanMove = &value
	}
	compile.CanQinggong = modulevm.LuaBool(modulevm.Field(table, "can_qinggong"))
	compile.RushNote = modulevm.LuaBool(modulevm.Field(table, "rush_note"))
	compile.Air = modulevm.Field(table, "air").AsString()
	return compile, true
}

// convertDescRules 解析触发钩子的规则数组（逐条白名单平铺字段；畸形
// 条目 WARN 跳过不连坐）。
func (s *Scripts) convertDescRules(skillID, hook string, value vm.Value) []*DescRule {
	if !value.IsTable() {
		return nil
	}
	table := value.AsTable()
	n := table.Len()
	rules := make([]*DescRule, 0, n)
	for i := 1; i <= n; i++ {
		item := modulevm.Index(table, i)
		if !item.IsTable() {
			s.host.WarnOnce(skillID+":"+hook, "skillscript: %s desc %s[%d] 非表，已跳过", skillID, hook, i)
			continue
		}
		rule := &DescRule{}
		ruleTable := item.AsTable()
		rule.PerHit = modulevm.LuaBool(modulevm.Field(ruleTable, "per_hit"))
		rule.Chance, _ = modulevm.LuaFloat(modulevm.Field(ruleTable, "chance"))
		if cond := modulevm.Field(ruleTable, "condition"); cond.IsTable() {
			condition := &DescCondition{
				Unparried:    modulevm.LuaBool(modulevm.Field(cond.AsTable(), "unparried")),
				ParryBroken:  modulevm.LuaBool(modulevm.Field(cond.AsTable(), "parry_broken")),
				TargetState:  modulevm.Field(cond.AsTable(), "target_state").AsString(),
				Unparryable:  modulevm.LuaBool(modulevm.Field(cond.AsTable(), "unparryable")),
			}
			rule.Condition = condition
		}
		rule.ControlKind = modulevm.Field(ruleTable, "control_kind").AsString()
		rule.ControlDurMS, _ = modulevm.LuaInt(modulevm.Field(ruleTable, "control_dur_ms"))
		rule.ControlWithinMS, _ = modulevm.LuaInt(modulevm.Field(ruleTable, "control_within_ms"))
		rule.DisplacementKind = modulevm.Field(ruleTable, "displacement_kind").AsString()
		rule.DisplacementDistanceM, _ = modulevm.LuaFloat(modulevm.Field(ruleTable, "displacement_distance_m"))
		rule.MPDrain = modulevm.LuaBool(modulevm.Field(ruleTable, "mp_drain"))
		rule.MPDrainDurMS, _ = modulevm.LuaInt(modulevm.Field(ruleTable, "mp_drain_dur_ms"))
		if restore, ok := modulevm.LuaInt(modulevm.Field(ruleTable, "mp_restore")); ok {
			rule.MPRestoreSet, rule.MPRestore = true, restore
		}
		rule.Heal = modulevm.LuaBool(modulevm.Field(ruleTable, "heal"))
		rule.ApplyBuff = modulevm.Field(ruleTable, "apply_buff").AsString()
		rule.ApplyResolved = modulevm.LuaBool(modulevm.Field(ruleTable, "apply_resolved"))
		rule.ApplyStacks, _ = modulevm.LuaInt(modulevm.Field(ruleTable, "apply_stacks"))
		rule.ApplyLifeMS, _ = modulevm.LuaInt(modulevm.Field(ruleTable, "apply_life_ms"))
		rule.BurstAt, _ = modulevm.LuaInt(modulevm.Field(ruleTable, "burst_at"))
		rule.BurstKind = modulevm.Field(ruleTable, "burst_kind").AsString()
		rule.BurstDurMS, _ = modulevm.LuaInt(modulevm.Field(ruleTable, "burst_dur_ms"))
		rule.StackBuff = modulevm.Field(ruleTable, "stack_buff").AsString()
		rule.StackResolved = modulevm.LuaBool(modulevm.Field(ruleTable, "stack_resolved"))
		rule.StackPerHit, _ = modulevm.LuaInt(modulevm.Field(ruleTable, "stack_per_hit"))
		rule.PeriodicEveryMS, _ = modulevm.LuaInt(modulevm.Field(ruleTable, "periodic_every_ms"))
		rule.PeriodicAOE = modulevm.LuaBool(modulevm.Field(ruleTable, "periodic_aoe"))
		rules = append(rules, rule)
	}
	return rules
}

// convertDescByLevel 解析 by_level{[level]={["dotted.path"]=数值}}（描述
// 数值随级变化的差集；路径键是生成器约定的点路径）。
func (s *Scripts) convertDescByLevel(skillID string, value vm.Value) map[int64]map[string]int64 {
	if !value.IsTable() {
		return nil
	}
	table := value.AsTable()
	out := make(map[int64]map[string]int64)
	key := vm.Nil
	for {
		nextKey, levelValue, err := table.Next(key)
		if err != nil || nextKey.IsNil() {
			break
		}
		key = nextKey
		level, levelOK := modulevm.LuaInt(nextKey)
		if !levelOK || level < 2 || !levelValue.IsTable() {
			continue
		}
		deltas := make(map[string]int64)
		deltaKey := vm.Nil
		for {
			deltaNext, deltaValue, deltaErr := levelValue.AsTable().Next(deltaKey)
			if deltaErr != nil || deltaNext.IsNil() {
				break
			}
			deltaKey = deltaNext
			if !deltaNext.IsString() {
				continue
			}
			if number, ok := modulevm.LuaInt(deltaValue); ok {
				deltas[deltaNext.AsString()] = number
			}
		}
		if len(deltas) > 0 {
			out[level] = deltas
		}
	}
	return out
}

// convertStringList 解析字符串数组（unparsed 留痕）。
func (s *Scripts) convertStringList(value vm.Value) ([]string, bool) {
	if !value.IsTable() {
		return nil, false
	}
	table := value.AsTable()
	n := table.Len()
	list := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		if item := modulevm.Index(table, i); item.IsString() {
			list = append(list, item.AsString())
		}
	}
	return list, len(list) > 0
}

// convertByLevel 转换 by_level{[level]={damage_amount=…, damage_waves=…,
// target_life_ms=…, self_life_ms=…}}。显式 0 是有效值（如该级无伤害），
// 缺字段沿用 L1 基准（不建该字段的指针）。
func (s *Scripts) convertByLevel(skillID string, value vm.Value) (map[int64]*LevelDelta, bool) {
	if !value.IsTable() {
		return nil, false
	}
	table := value.AsTable()
	out := make(map[int64]*LevelDelta)
	key := vm.Nil
	for {
		nextKey, deltaValue, err := table.Next(key)
		if err != nil || nextKey.IsNil() {
			break
		}
		key = nextKey
		level, levelOK := modulevm.LuaInt(nextKey)
		if !levelOK || level < 1 || !deltaValue.IsTable() {
			continue
		}
		delta := &LevelDelta{}
		set := false
		deltaTable := deltaValue.AsTable()
		deltaKey := vm.Nil
		for {
			deltaNext, deltaField, deltaErr := deltaTable.Next(deltaKey)
			if deltaErr != nil || deltaNext.IsNil() {
				break
			}
			deltaKey = deltaNext
			if !deltaNext.IsString() {
				continue
			}
			number, ok := modulevm.LuaInt(deltaField)
			if !ok {
				continue
			}
			switch deltaNext.AsString() {
			case "damage_amount":
				delta.DamageAmount, set = &number, true
			case "damage_waves":
				delta.DamageWaves, set = &number, true
			case "target_life_ms":
				delta.TargetLifeMS, set = &number, true
			case "self_life_ms":
				delta.SelfLifeMS, set = &number, true
			}
		}
		if set {
			out[level] = delta
		}
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}
