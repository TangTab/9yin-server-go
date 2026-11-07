package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	modernSkillRoot       = filepath.Join(defaultModernShareRoot, "skill")
	modernSkillActionRoot = defaultModernActionRoot
)

var modernPlayerSkillActionFiles = []string{
	"zhaoshi_player.ini",
	"zhaoshi_player_2.ini",
	"zhaoshi_player_dodge.ini",
	"zhaoshi_player_parry.ini",
}

type iniField struct {
	key   string
	value string
}

type iniTable map[string][]iniField

func loadINISections(path string) (iniTable, error) {
	file, err := openResourceReader(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	table := make(iniTable)
	section := ""
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			if _, exists := table[section]; !exists {
				table[section] = nil
			}
			continue
		}
		if section == "" {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		table[section] = append(table[section], iniField{
			key: strings.TrimSpace(key), value: strings.TrimSpace(value),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return table, nil
}

func iniValue(fields []iniField, key string) string {
	for i := len(fields) - 1; i >= 0; i-- {
		if strings.EqualFold(fields[i].key, key) {
			return fields[i].value
		}
	}
	return ""
}

func iniInt(fields []iniField, key string) int32 {
	value, _ := strconv.ParseInt(strings.TrimSpace(iniValue(fields, key)), 10, 32)
	return int32(value)
}

func iniFloat(fields []iniField, key string) float32 {
	value, _ := strconv.ParseFloat(strings.TrimSpace(iniValue(fields, key)), 32)
	return float32(value)
}

type skillResourceTables struct {
	skillNew    iniTable
	skillStatic iniTable
	normalVar   iniTable
	lockVar     iniTable
	consume     iniTable
	damage      iniTable
	hitShape    iniTable
	targetShape iniTable
	lockData    iniTable
	actions     iniTable
	buffNew     iniTable
	buffStatic  iniTable
	buffVar     iniTable
}

func loadSkillResourceTables() (skillResourceTables, error) {
	load := func(path string) (iniTable, error) {
		table, err := loadINISections(path)
		if err != nil {
			return nil, fmt.Errorf("load skill resource %s: %w", filepath.Base(path), err)
		}
		return table, nil
	}
	var result skillResourceTables
	var err error
	if result.skillNew, err = load(filepath.Join(modernSkillRoot, "skill_new.ini")); err != nil {
		return result, err
	}
	if result.skillStatic, err = load(filepath.Join(modernSkillRoot, "skill_static.ini")); err != nil {
		return result, err
	}
	if result.normalVar, err = load(filepath.Join(modernSkillRoot, "skill_normal_varprop.ini")); err != nil {
		return result, err
	}
	if result.lockVar, err = load(filepath.Join(modernSkillRoot, "skill_lock_varprop.ini")); err != nil {
		return result, err
	}
	if result.consume, err = load(filepath.Join(modernSkillRoot, "skill_consume.ini")); err != nil {
		return result, err
	}
	if result.damage, err = load(filepath.Join(modernSkillRoot, "damage_calculate.ini")); err != nil {
		return result, err
	}
	if result.hitShape, err = load(filepath.Join(modernSkillRoot, "attack_hitshape.ini")); err != nil {
		return result, err
	}
	// attack_targetshape.ini / skill_lock_data.ini：冲锋链（TargetShapePara2
	// 真实射程、LockGRushAction 冲刺动画/BindDistance）所需。测试夹具仓与
	// 精简部署可能缺这两张表：缺失按空表降级（射程回退 max(形状半径,8m)、
	// 冲刺参数回退默认值），与 Java 行缺失时的 fallback 语义一致。
	loadOptional := func(name string) (iniTable, error) {
		table, loadErr := load(filepath.Join(modernSkillRoot, name))
		if loadErr != nil {
			if errors.Is(loadErr, fs.ErrNotExist) {
				log.Printf("skill resource %s missing; rush cast falls back to default range/params", name)
				return make(iniTable), nil
			}
			return nil, loadErr
		}
		return table, nil
	}
	if result.targetShape, err = loadOptional("attack_targetshape.ini"); err != nil {
		return result, err
	}
	if result.lockData, err = loadOptional("skill_lock_data.ini"); err != nil {
		return result, err
	}
	result.actions = make(iniTable)
	for _, name := range modernPlayerSkillActionFiles {
		actionTable, loadErr := load(filepath.Join(modernSkillActionRoot, name))
		if loadErr != nil {
			return result, loadErr
		}
		for section, fields := range actionTable {
			// zhaoshi_list.ini 把这些文件包含进同一个逻辑动作
			// 命名空间。后出现的重名小节遵循 include 顺序。
			result.actions[section] = fields
		}
	}
	if result.buffNew, err = load(filepath.Join(modernSkillRoot, "buff_new.ini")); err != nil {
		return result, err
	}
	if result.buffStatic, err = load(filepath.Join(modernSkillRoot, "buff_static.ini")); err != nil {
		return result, err
	}
	if result.buffVar, err = load(filepath.Join(modernSkillRoot, "buff_varprop.ini")); err != nil {
		return result, err
	}
	return result, nil
}

func levelVarProp(static []iniField, script string, level int32, tables skillResourceTables) []iniField {
	_, fields := findVarPropRow(static, script, level, tables)
	return fields
}

// findVarPropRow 是 levelVarProp 的带行号版（Java findVarProp 同构）：返回
// 匹配 varprop 的节号——View40 行 VarPropNo 与客户端 varpkg 取同值。
func findVarPropRow(static []iniField, script string, level int32, tables skillResourceTables) (int32, []iniField) {
	var table iniTable
	if strings.EqualFold(script, "SkillLock") {
		table = tables.lockVar
	} else {
		table = tables.normalVar
	}
	minimum := iniInt(static, "MinVarPropNo")
	maximum := iniInt(static, "MaxVarPropNo")
	for id := minimum; id <= maximum && id > 0; id++ {
		fields := table[strconv.FormatInt(int64(id), 10)]
		if iniInt(fields, "Level") == level {
			return id, fields
		}
	}
	return 0, nil
}

// compileSkillDamage 返回 damage_calculate 行的伤害编译对：baseDamage=
// 全部 r 字段第 3 段正值之和；waves=正 r 段数（Java damageWaves——多段
// 技能的倍率分母）。
func compileSkillDamage(fields []iniField) (baseDamage, waves int32) {
	var total float64
	for _, field := range fields {
		if !strings.EqualFold(field.key, "r") {
			continue
		}
		parts := strings.Split(field.value, ",")
		if len(parts) < 3 {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err == nil && value > 0 {
			total += value
			waves++
		}
	}
	return int32(math.Round(total)), waves
}

func compileTargetMode(fields []iniField, isDamage bool) (skillTargetMode, float32, float32, float32) {
	if !isDamage {
		return skillTargetSelf, 0, 0, 0
	}
	switch iniInt(fields, "HitShape") {
	case 1:
		return skillTargetCasterCircle, iniFloat(fields, "HitShapePara2"), 0, 0
	case 2, 11:
		return skillTargetSelectedArea, iniFloat(fields, "HitShapePara2"), 0, 0
	case 3, 9, 10:
		return skillTargetCasterSector, iniFloat(fields, "HitShapePara2"), iniFloat(fields, "HitShapePara1"), 0
	case 4, 12:
		return skillTargetCasterRectangle, iniFloat(fields, "HitShapePara2"), 0, iniFloat(fields, "HitShapePara1")
	default:
		return skillTargetSelected, 0, 0, 0
	}
}

type rawActionSegment struct {
	order    int
	action   string
	endFrame float64
	inBlend  float64
}

// compileSkillActions 编译 zhaoshi 时间线：动作段（此前已有）+ 多段判定
// 时刻 hitFrames（Java CombatSkillCatalog.compileSkillActions 同构）。
// trigger_custom_N=段号;trigger_effect;帧号;效果名，过滤序（52795 抓包
// 实证）：①效果名含 _hurt 的帧=伤害段（gxfhj02 的 ×4=每招 4 段）；②无
// hurt 帧时取效果名以技能核心名（尾段去数字取前 3 字母，如 sfgp01→sfg）
// 开头的帧（shl08 的 shl08_NN×6=6 段）；刀光/fly/特效帧与技能全名施法帧
// 剔除；皆无 → 单段即时结算。
func compileSkillActions(fields []iniField, skillID string) (string, time.Duration, []skillActionSegment, []time.Duration) {
	raw := make([]rawActionSegment, 0, 3)
	hurtFrames := make([]float64, 0, 4)
	coreFrames := make([]float64, 0, 4)
	corePrefix := skillCorePrefix(skillID)
	for _, field := range fields {
		if order, err := strconv.Atoi(field.key); err == nil {
			parts := strings.Split(field.value, ";")
			if len(parts) < 5 || strings.TrimSpace(parts[0]) == "" {
				continue
			}
			endFrame, _ := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
			inBlend, _ := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
			raw = append(raw, rawActionSegment{
				order: order, action: strings.TrimSpace(parts[0]), endFrame: endFrame, inBlend: inBlend,
			})
			continue
		}
		if !strings.HasPrefix(field.key, "trigger_custom_") {
			continue
		}
		parts := strings.Split(field.value, ";")
		// 布局：段号;trigger_effect;帧号;效果名
		if len(parts) < 4 || !strings.EqualFold(strings.TrimSpace(parts[1]), "trigger_effect") {
			continue
		}
		effectName := strings.TrimSpace(parts[3])
		if effectName == "" || effectName == skillID {
			continue // 施法特效帧（与技能同名）不是伤害段
		}
		frame, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err != nil || frame < 0 {
			continue
		}
		if strings.Contains(effectName, "_hurt") {
			hurtFrames = append(hurtFrames, frame)
		} else if corePrefix != "" && strings.HasPrefix(effectName, corePrefix) {
			coreFrames = append(coreFrames, frame)
		}
	}
	hitFrames := hurtFrames
	if len(hitFrames) == 0 {
		hitFrames = coreFrames
	}
	sort.Float64s(hitFrames)
	hitFrameMs := make([]time.Duration, len(hitFrames))
	for i, frame := range hitFrames {
		hitFrameMs[i] = time.Duration(frame / 30 * float64(time.Second))
	}
	sort.Slice(raw, func(i, j int) bool { return raw[i].order < raw[j].order })
	if len(raw) == 0 {
		return "", 0, nil, hitFrameMs
	}
	duration := time.Duration(raw[0].endFrame / 30 * float64(time.Second))
	at := time.Duration(0)
	followups := make([]skillActionSegment, 0, len(raw)-1)
	for i := 1; i < len(raw); i++ {
		overlap := time.Duration(math.Max(raw[i].inBlend, 0) * float64(time.Second))
		at += time.Duration(raw[i-1].endFrame/30*float64(time.Second)) - overlap
		followups = append(followups, skillActionSegment{
			action: raw[i].action,
			at:     at,
			// State 会播放完整的具名资源。它无法选择
			// zhaoshi_player.ini 中编码的起止片段。
			duration: time.Duration(raw[i].endFrame / 30 * float64(time.Second)),
		})
	}
	return raw[0].action, duration, followups, hitFrameMs
}

// skillCorePrefix 技能核心名前缀（CS_jh_sfgp01→"sfg"、CS_jhgp_gxfhj02→
// "gxf"、CS_xy_shl08→"shl"；Java skillCorePrefix 同构）：最后一个含字母
// 且含数字的分段，剔数字后取前 3 字母。
func skillCorePrefix(skillID string) string {
	segments := strings.Split(skillID, "_")
	for i := len(segments) - 1; i >= 0; i-- {
		hasLetter, hasDigit := false, false
		for _, r := range segments[i] {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				hasLetter = true
			}
			if r >= '0' && r <= '9' {
				hasDigit = true
			}
		}
		if !hasLetter || !hasDigit {
			continue
		}
		letters := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				return r
			}
			return -1
		}, segments[i])
		if len(letters) >= 3 {
			return letters[:3]
		}
		return letters
	}
	return ""
}

func compileExactSkillBuff(skillID string, level int32, tables skillResourceTables) (skillBuffDefinition, bool, bool) {
	configID := "buf_" + skillID
	buffNew := tables.buffNew[configID]
	staticData := iniInt(buffNew, "StaticData")
	if staticData <= 0 {
		return skillBuffDefinition{}, false, false
	}
	static := tables.buffStatic[strconv.FormatInt(int64(staticData), 10)]
	lifetime := time.Duration(iniInt(buffNew, "LifeTime")) * time.Millisecond
	minimum := iniInt(static, "MinVarPropNo")
	maximum := iniInt(static, "MaxVarPropNo")
	for id := minimum; id <= maximum && id > 0; id++ {
		fields := tables.buffVar[strconv.FormatInt(int64(id), 10)]
		if iniInt(fields, "Level") != level {
			continue
		}
		if milliseconds := iniInt(fields, "LifeTime"); milliseconds > 0 {
			lifetime = time.Duration(milliseconds) * time.Millisecond
		}
		break
	}
	if lifetime <= 0 {
		lifetime = 24 * time.Hour
	}
	return skillBuffDefinition{
		configID: configID, staticData: uint32(staticData), level: level, lifetime: lifetime,
	}, iniInt(static, "IsDamage") != 0, true
}

type skillEffectOverlay struct {
	selfBuffLease time.Duration
	preferredSlot uint16
	redArmor      bool
	stance        []skillBuffDefinition
}

var verifiedSkillEffectOverlays = map[string]skillEffectOverlay{
	"CS_yhwq_hsqs04": {selfBuffLease: 35 * time.Second, preferredSlot: yiHuaSkillBuffSlot},
	"CS_yhwq_hsqs05": {preferredSlot: yiHuaSkill05BuffSlot, redArmor: true},
	"CS_wd_tjq04": {
		preferredSlot: taiJiStanceBuffSlot,
		stance: []skillBuffDefinition{
			{configID: "buf_CS_wd_tjq04_1", staticData: taiJiDefenceStanceBuffStaticData, lifetime: taiJiStanceBuffLease, preferredSlot: taiJiStanceBuffSlot},
			{configID: "buf_CS_wd_tjq04_2", staticData: taiJiAttackStanceBuffStaticData, lifetime: taiJiStanceBuffLease, preferredSlot: taiJiStanceBuffSlot},
		},
	},
	"CS_wd_tjq05": {preferredSlot: taiJiSkill05BuffSlot},
	"CS_wd_tjq07": {preferredSlot: taiJiSkill07BuffSlot},
	"CS_wd_tjq08": {redArmor: true},
}

func compileCombatSkill(skillID string, level int32, tables skillResourceTables) (combatSkillDefinition, error) {
	if level < 1 {
		return combatSkillDefinition{}, fmt.Errorf("%s has invalid level %d", skillID, level)
	}
	newFields := tables.skillNew[skillID]
	staticData := iniInt(newFields, "StaticData")
	if staticData <= 0 {
		return combatSkillDefinition{}, fmt.Errorf("%s has no StaticData", skillID)
	}
	script := iniValue(newFields, "script")
	if !strings.EqualFold(script, "SkillNormal") && !strings.EqualFold(script, "SkillLock") {
		return combatSkillDefinition{}, fmt.Errorf("%s uses unsupported script %q", skillID, script)
	}
	static := tables.skillStatic[strconv.FormatInt(int64(staticData), 10)]
	if len(static) == 0 {
		return combatSkillDefinition{}, fmt.Errorf("%s has no skill_static section %d", skillID, staticData)
	}
	varProp := levelVarProp(static, script, level, tables)
	if len(varProp) == 0 {
		return combatSkillDefinition{}, fmt.Errorf("%s has no level-%d varprop", skillID, level)
	}
	consume := tables.consume[strconv.FormatInt(int64(iniInt(varProp, "Consume")), 10)]
	damage, damageWaves := compileSkillDamage(tables.damage[strconv.FormatInt(int64(iniInt(varProp, "DamageCalculate")), 10)])
	isDamage := iniInt(static, "IsDamage") != 0
	hitShape := tables.hitShape[strconv.FormatInt(int64(iniInt(varProp, "HitShapePkg")), 10)]
	targetMode, radius, angle, width := compileTargetMode(hitShape, isDamage)
	action, actionDuration, followups, hitFrameMs := compileSkillActions(tables.actions[skillID], skillID)
	if action == "" {
		return combatSkillDefinition{}, fmt.Errorf("%s has no player action timeline", skillID)
	}

	publicCooldown := iniInt(varProp, "AddCoolDownTime")
	if publicCooldown <= 0 {
		publicCooldown = iniInt(newFields, "AddCoolDownTime")
	}
	// 冲锋链编译（Java CombatSkillCatalog 同源）：TargetShapePara2 即技能
	// 面板真实射程；缺表/缺行回退 max(形状半径, 8m)。锁定参数经
	// skill_static.LockType 索引 skill_lock_data 行。
	targetShape := tables.targetShape[strconv.FormatInt(int64(iniInt(varProp, "TargetShapePkg")), 10)]
	castRange := iniFloat(targetShape, "TargetShapePara2")
	if castRange <= 0 {
		castRange = float32(math.Max(float64(radius), selectedRangeFallbackMeters))
	}
	lockRow := tables.lockData[strconv.FormatInt(int64(iniInt(static, "LockType")), 10)]
	bindDistance := iniFloat(lockRow, "BindDistance")
	if bindDistance <= 0 {
		bindDistance = defaultRushBindDistance
	}
	groundRushAction := iniValue(lockRow, "LockGRushAction")
	if groundRushAction == "" {
		groundRushAction = defaultGroundRushAction
	}
	definition := combatSkillDefinition{
		id: skillID, staticData: staticData, script: script,
		taoLu: iniValue(static, "TaoLu"), level: level,
		cooldownCategory: iniInt(static, "CoolDownCategory"),
		cooldownTeam:     iniInt(static, "CoolDownTeam"),
		personalCD:       time.Duration(iniInt(varProp, "CoolDownTime")) * time.Millisecond,
		publicCD:         time.Duration(publicCooldown) * time.Millisecond,
		mpCost:           iniInt(consume, "AConsumeMP"),
		spCost:           iniInt(consume, "AConsumeSP"),
		qgCost:           iniInt(consume, "AConsumeQGP"),
		baseDamage:       damage,
		damageWaves:      damageWaves,
		hitFrameMs:       hitFrameMs,
		levelupRelevance: iniValue(newFields, "LevelupRelevance"),
		packCategory:     iniInt(newFields, "PackCategory"),
		requiresTarget:   targetMode == skillTargetSelected || targetMode == skillTargetSelectedArea,
		targetMode:       targetMode, areaRadius: radius, areaHeight: iniFloat(varProp, "HeightDiff"),
		sectorAngle: angle, areaWidth: width,
		actionName: action, actionDuration: actionDuration, followupActions: followups,
		castRange: castRange,
		canRush:   iniInt(static, "CanRush") != 0,
		lockSkill: strings.EqualFold(script, "skilllock") || iniInt(static, "CanRush") != 0 ||
			(iniInt(static, "TargetType") == 4 && iniInt(static, "DamageType") != 0),
		isDamageSkill:    isDamage,
		bindDistance:     bindDistance,
		groundRushAction: groundRushAction,
	}
	if definition.publicCD <= 0 {
		definition.publicCD = definition.personalCD
	}
	if damage > 0 {
		definition.effects = append(definition.effects, combatSkillEffect{
			kind: skillEffectDamage, amount: damage,
		})
	}
	if buff, harmful, exists := compileExactSkillBuff(skillID, level, tables); exists {
		overlay := verifiedSkillEffectOverlays[skillID]
		if overlay.selfBuffLease > 0 {
			buff.lifetime = overlay.selfBuffLease
		}
		buff.preferredSlot = overlay.preferredSlot
		kind := skillEffectPlayerBuff
		if harmful {
			kind = skillEffectTargetControl
		}
		definition.effects = append(definition.effects, combatSkillEffect{kind: kind, buff: buff})
	}
	if overlay := verifiedSkillEffectOverlays[skillID]; len(overlay.stance) > 0 {
		definition.effects = append(definition.effects, combatSkillEffect{
			kind: skillEffectStanceCycle, buffVariants: overlay.stance,
		})
	}
	if verifiedSkillEffectOverlays[skillID].redArmor {
		definition.effects = append(definition.effects, combatSkillEffect{kind: skillEffectRedArmor})
	}
	return definition, nil
}

type combatSkillCacheKey struct {
	id    string
	level int32
}

type combatSkillCatalog struct {
	tables      skillResourceTables
	levelOne    map[string]combatSkillDefinition
	indexed     map[string]struct{}
	skipReasons map[string]int
	cache       map[combatSkillCacheKey]combatSkillDefinition
	cacheMu     sync.RWMutex
}

func skillCompileReason(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "unsupported script"):
		return "unsupported_script"
	case strings.Contains(message, "no StaticData"):
		return "missing_static_id"
	case strings.Contains(message, "no skill_static"):
		return "missing_static"
	case strings.Contains(message, "varprop"):
		return "missing_level_varprop"
	case strings.Contains(message, "action timeline"):
		return "missing_player_action"
	default:
		return "other"
	}
}

func loadCombatSkillCatalog() (*combatSkillCatalog, error) {
	tables, err := loadSkillResourceTables()
	if err != nil {
		return nil, err
	}
	catalog := &combatSkillCatalog{
		tables: tables, levelOne: make(map[string]combatSkillDefinition),
		indexed: make(map[string]struct{}), skipReasons: make(map[string]int),
		cache: make(map[combatSkillCacheKey]combatSkillDefinition),
	}
	ids := make([]string, 0, len(tables.skillNew))
	for id := range tables.skillNew {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		script := iniValue(tables.skillNew[id], "script")
		if !strings.EqualFold(script, "SkillNormal") && !strings.EqualFold(script, "SkillLock") {
			catalog.skipReasons["unsupported_script"]++
			continue
		}
		catalog.indexed[id] = struct{}{}
		definition, compileErr := compileCombatSkill(id, 1, tables)
		if compileErr != nil {
			catalog.skipReasons[skillCompileReason(compileErr)]++
			continue
		}
		catalog.levelOne[id] = definition
		catalog.cache[combatSkillCacheKey{id: id, level: 1}] = definition
	}
	return catalog, nil
}

func (catalog *combatSkillCatalog) definition(id string, level int32) (combatSkillDefinition, bool) {
	if catalog == nil || level < 1 {
		return combatSkillDefinition{}, false
	}
	key := combatSkillCacheKey{id: id, level: level}
	catalog.cacheMu.RLock()
	definition, exists := catalog.cache[key]
	catalog.cacheMu.RUnlock()
	if exists {
		return definition, true
	}
	if _, indexed := catalog.indexed[id]; !indexed {
		return combatSkillDefinition{}, false
	}
	definition, err := compileCombatSkill(id, level, catalog.tables)
	if err != nil {
		return combatSkillDefinition{}, false
	}
	catalog.cacheMu.Lock()
	catalog.cache[key] = definition
	catalog.cacheMu.Unlock()
	return definition, true
}

// skillRowSupplement 是 View40 技能行的官方补集（Java CombatSkillCatalog.SkillRowData
// 同构，另折入 personalCdMs 的 CoolDownTime 显示值）：官方行携带而最小集未发的
// VarPropNo/UseLimit/消耗族/形状参数——快捷栏 tooltip 的消耗与最大射程、施法限距、
// 怒招怒气不足灰图标、CD 文本都读它们（bug.txt「怒招的消耗没有下发」项）。
type skillRowSupplement struct {
	varPropNo        int32
	useLimit         int32
	aConsumeHP       int32
	aConsumeMP       int32
	aConsumeSP       int32
	aConsumeQGP      int32
	targetShapePara2 float32
	hitShapePara1    float32
	hitShapePara2    float32
	coolDownMs       int32
}

// skillRowData 按 (id, level) 取 View40 行补集；查不到链路返回 nil，调用方回落。
// 不走完整 compile——缺动作/伤害表的技术也能取值（打坐调息等）。发射侧沿用
// 官方稀疏风格（参考服 207 行抓包分桶实证 2026-11-05）：VarPropNo/CoolDownTime
// 每行必发；消耗/射程/UseLimit 仅 >0 才发；HitShapePara1 仅扇形族
// （HitShape=3/9/10）发、HitShapePara2 仅 HitShape>0 发（htype=0 无形状不发）；
// TargetShapePara1 官方从不发（207 行 0 次），不取。
func (catalog *combatSkillCatalog) skillRowData(id string, level int32) *skillRowSupplement {
	if catalog == nil || id == "" || level < 1 {
		return nil
	}
	newFields, indexed := catalog.tables.skillNew[id]
	if !indexed {
		return nil
	}
	static, ok := catalog.tables.skillStatic[strconv.FormatInt(int64(iniInt(newFields, "StaticData")), 10)]
	if !ok {
		return nil
	}
	rowNo, varProp := findVarPropRow(static, iniValue(newFields, "script"), level, catalog.tables)
	if varProp == nil {
		return nil
	}
	consume := catalog.tables.consume[strconv.FormatInt(int64(iniInt(varProp, "Consume")), 10)]
	targetShape := catalog.tables.targetShape[strconv.FormatInt(int64(iniInt(varProp, "TargetShapePkg")), 10)]
	hitShape := catalog.tables.hitShape[strconv.FormatInt(int64(iniInt(varProp, "HitShapePkg")), 10)]
	hitShapeType := iniInt(hitShape, "HitShape")
	supplement := &skillRowSupplement{
		varPropNo:        rowNo,
		useLimit:         iniInt(newFields, "UseLimit"),
		aConsumeHP:       iniInt(consume, "AConsumeHP"),
		aConsumeMP:       iniInt(consume, "AConsumeMP"),
		aConsumeSP:       iniInt(consume, "AConsumeSP"),
		aConsumeQGP:      iniInt(consume, "AConsumeQGP"),
		targetShapePara2: iniFloat(targetShape, "TargetShapePara2"),
		// 个人 CD 显示值：varprop 按级 CoolDownTime 毫秒（skill_init.ini cooldown
		// 官方全 0 不作源——下发 0 客户端 CD 文本兜底显示 1s；127-0-0-1-59719
		// View40 行抓包 17/17 与 varprop 对齐）。
		coolDownMs: iniInt(varProp, "CoolDownTime"),
	}
	if hitShapeType == 3 || hitShapeType == 9 || hitShapeType == 10 {
		supplement.hitShapePara1 = iniFloat(hitShape, "HitShapePara1")
	}
	if hitShapeType > 0 {
		supplement.hitShapePara2 = iniFloat(hitShape, "HitShapePara2")
	}
	return supplement
}

// maxLevelOf 技能满级（秘籍书直达层夹制用；Java maxLevelOf 同构）：
// skill_static [Min,Max]VarPropNo 区间内 varprop（script=skilllock 走
// lockVar 表，否则 normalVar）Level 的最大值；未索引/无区间/空表 → 1。
func (catalog *combatSkillCatalog) maxLevelOf(id string) int32 {
	newFields, indexed := catalog.tables.skillNew[id]
	if !indexed {
		return 1
	}
	static := catalog.tables.skillStatic[strconv.FormatInt(int64(iniInt(newFields, "StaticData")), 10)]
	varTable := catalog.tables.normalVar
	if strings.EqualFold(iniValue(newFields, "script"), "skilllock") {
		varTable = catalog.tables.lockVar
	}
	best := int32(1)
	for varID := iniInt(static, "MinVarPropNo"); varID <= iniInt(static, "MaxVarPropNo") && varID > 0; varID++ {
		fields := varTable[strconv.FormatInt(int64(varID), 10)]
		if level := iniInt(fields, "Level"); level > best {
			best = level
		}
	}
	return best
}

func mustLoadCombatSkillCatalog() *combatSkillCatalog {
	catalog, err := loadCombatSkillCatalog()
	if err != nil {
		panic(fmt.Sprintf("compile modern combat skill catalog: %v", err))
	}
	log.Printf("modern universal skill catalog ready indexed=%d executable_level1=%d skipped=%v action_files=%d",
		len(catalog.indexed), len(catalog.levelOne), catalog.skipReasons, len(modernPlayerSkillActionFiles))
	return catalog
}

func skillCatalogFamily(all map[string]combatSkillDefinition, prefix string) map[string]combatSkillDefinition {
	result := make(map[string]combatSkillDefinition)
	for id, definition := range all {
		if strings.HasPrefix(id, prefix) {
			result[id] = definition
		}
	}
	return result
}

var installedCombatSkillCatalog = mustLoadCombatSkillCatalog()
var combatSkills = installedCombatSkillCatalog.levelOne
var yiHuaWuQueCombatSkills = skillCatalogFamily(combatSkills, "CS_yhwq_hsqs")
var taiJiQuanGuPuCombatSkills = skillCatalogFamily(combatSkills, "CS_wd_tjq")
var heartBuddhaPalmCombatSkills = skillCatalogFamily(combatSkills, "CS_jh_xfz")
