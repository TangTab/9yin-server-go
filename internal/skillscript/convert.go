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
