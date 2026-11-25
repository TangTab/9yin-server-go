package fssoloworld

// combathit.go 玩家技能的命中调度与结算（官服抓包逐帧定版，帧序见
// combat.go 文件头）。
//
// 命中节拍取用顺序：
//
//	① 招式表判定帧（ini/Action 产物；按 m20 下发的动作名查表）
//	② 流程包 HitTime / HitCount / LeadSepTime（NPC 的 default_normal_skill
//	   即此形：HitTime=1000）
//	③ 都没有 → 不结算（非伤害技）
//
// 分母击数（公式 HurtCount）= 计划击数。每击重新索敌（范围技以玩家当前
// 位为形状中心——2433 定案「技能期间进入范围的都要计算伤害」）。
//
// 锁序：玩家 combatState.mu → NPC npcFighter.mu；调 Lua 钩子/引擎前一律
// 放锁（脚本会经 nx_* 回到宿主取锁）。

import (
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/skilldata"

	"github.com/lizhanyin/9yin-server-go/internal/script"
)

// combatRoll 命中结算的随机源 [0,1)（未命中/暴击/区间掷值；测试可替换）。
var combatRoll = rand.Float64

// hitPlan 一次施放的命中计划。
type hitPlan struct {
	times []time.Duration // 各击时刻（自施放/落地起）
	hurt  int             // 分母击数（公式 HurtCount）
}

func (p hitPlan) valid() bool         { return len(p.times) > 0 }
func (p hitPlan) last() time.Duration { return p.times[len(p.times)-1] }

// skillHitPlan 命中计划：招式表判定帧 → 流程包 → 无。
func (s *Server) skillHitPlan(action string, def skilldata.Definition) hitPlan {
	if lua := s.combatLua(); lua.actions != nil && action != "" {
		if schedule, ok := lua.actions.Lookup(action); ok && len(schedule.HitsMS) > 0 {
			plan := hitPlan{hurt: len(schedule.HitsMS)}
			for _, ms := range schedule.HitsMS {
				plan.times = append(plan.times, time.Duration(ms)*time.Millisecond)
			}
			return plan
		}
	}
	if def.Flow.HitTimeMS <= 0 || def.DamageCalculate == 0 {
		return hitPlan{}
	}
	count := max(def.HitCount, 1)
	step := time.Duration(def.Flow.LeadSepTimeMS) * time.Millisecond
	if count > 1 && step <= 0 {
		step = time.Duration(def.Flow.HitTimeMS) * time.Millisecond / time.Duration(count)
	}
	plan := hitPlan{hurt: count}
	for i := 0; i < count; i++ {
		plan.times = append(plan.times, time.Duration(def.Flow.HitTimeMS)*time.Millisecond+time.Duration(i)*step)
	}
	return plan
}

// skillAction 施放时 m20 下发的动作名（招式表查表键；连招段按技能各自
// 轮转——与 planSkillCast 取 m20 动作同一计数器）。
func skillAction(st *combatState, def skilldata.Definition) string {
	if count := len(def.Flow.HitActions); count > 0 {
		if st.normalSteps == nil {
			st.normalSteps = map[string]int{}
		}
		action := def.Flow.HitActions[st.normalSteps[def.ID]%count]
		st.normalSteps[def.ID]++
		return action
	}
	return def.ID
}

// scheduleSkillHits 受理后登记命中计划，返回末击时刻（无计划返回 0）。
// base = 自受理到计划原点的偏移。
func (s *Server) scheduleSkillHits(pl *Player, def skilldata.Definition, cast skillCast, action string, base time.Duration) time.Duration {
	if !combatEnabled() {
		return 0
	}
	plan := s.skillHitPlan(action, def)
	if !plan.valid() {
		return 0
	}
	st := &pl.combat
	st.mu.Lock()
	defer st.mu.Unlock()
	st.live.hits = 0
	if aux := s.auxByHandle(st.skillObj); aux == nil || aux.config != def.ID {
		st.skillObj = s.registerAux(&auxObject{kind: script.HelperSkill, config: def.ID, script: def.LuaScript,
			def: def, owner: playerHandle(pl), props: map[string]script.Value{}, records: map[string][][]script.Value{}}).ID
	}
	seq := st.castSeq
	for i := range plan.times {
		index := i + 1
		s.after(base+plan.times[i], func() { s.resolveSkillHit(pl, def, cast, plan, index, seq) })
	}
	return plan.last()
}

// castEndFrames 结束当前施放：清 CurSkillID；alsoTarget 时另清
// CurSkillTarget。自然收招只清 CurSkillID（抓包 217.239：CurSkillTarget
// 保持），被闪避收束时两者同清（抓包 220.371 / 226.276）。出招锁位
// （CantMove/CantParry）由收招定时器 finishCast 清。
func (s *Server) castEndFrames(pl *Player, def skilldata.Definition, sink *frameSink, alsoTarget bool) {
	st := &pl.combat
	st.mu.Lock()
	active := st.curSkill == def.ID
	if active {
		st.curSkill = ""
	}
	st.mu.Unlock()
	if !active {
		return
	}
	sink.priv("CurSkillID", entergen.StringValue(""))
	if alsoTarget {
		sink.priv("CurSkillTarget", entergen.ObjectValue(0, 0))
	}
}

// armCalmLocked 战斗活动刷新后排一次脱战检查（恰在最后一次活动
// playerLeaveFight 之后；调用方持 combatState.mu）。
func (s *Server) armCalmLocked(st *combatState, pl *Player, now time.Time) {
	at := now.Add(playerLeaveFight)
	if st.calmAt.Equal(at) {
		return
	}
	st.calmAt = at
	s.after(playerLeaveFight, func() { s.playerCalm(pl) })
}

// resolveSkillHit 第 index 击（1 基）。帧序（抓包 208.229 / 216.675 /
// 挑战包 483.077）：
//
//	[首击] 技能行 CurTargetCount=目标数 → [攻击者 FP → SP] →
//	       各目标：[首次交手：NPC.LogicState=1 → 关系行加行 → 玩家
//	       LogicState=1 → m24 37] → 关系行刷新 → m24 17 → [m24 40] →
//	       [击杀：关系行删 → m24 29 → m18 → m24 13] →//	[第 2 击起] m24[攻击者,10,累计击数] → MaxHits → m205[攻击者,技能,目标…] →
//	[末击] 收招
//
// 各目标的帧先入各自的子汇，命中确定后按上次序合并（FP/SP 只在确有
// 命中时发，且先于全部目标帧）。
func (s *Server) resolveSkillHit(pl *Player, def skilldata.Definition, cast skillCast, plan hitPlan, index, seq int) {
	st := &pl.combat
	st.mu.Lock()
	active := st.castSeq == seq && st.curSkill == def.ID
	s.ensureLiveLocked(st, pl)
	skillObj := st.skillObj
	st.mu.Unlock()
	if !active || pl.State != PlOnline {
		return
	}
	lua := s.combatLua()
	// 技能脚本钩子：命中之前（返回 0 = 不再结算）。
	if hook := lua.vm.Function(def.LuaScript, "skill_before_hit"); hook != nil {
		out, err := hook.Call(script.Ref(playerHandle(pl)),
			script.Ref(script.Handle{Kind: script.KindHelper, Helper: script.HelperSkill, ID: skillObj}))
		if err == nil && len(out) > 0 && out[0].Number() == 0 {
			return
		}
	}

	targets := s.skillTargets(pl, def, cast)
	sink := s.sinkFor(pl)
	if index == 1 {
		view, row := skillView(pl, def)
		sink.pri(entergen.ObjectPropertyFrame(view, row, 1, []entergen.IndexedProperty{
			entergen.NamedProperty("CurTargetCount", entergen.Int32Value(int32(len(targets))))}))
	}
	type landing struct {
		f    *npcFighter
		sink *frameSink
	}
	var hit []landing
	for _, f := range targets {
		sub := s.sinkFor(pl)
		if s.hitNPC(pl, def, f, plan.hurt, len(targets), skillObj, sub) {
			hit = append(hit, landing{f: f, sink: sub})
		} else {
			sink.absorb(sub) // 闪避等帧原位保留
		}
	}
	if len(hit) == 0 {
		// 本击无有效命中（目标闪避/已死亡）：施放就此收束，CurSkillID 与
		// CurSkillTarget 同清（抓包 220.371 / 226.276）。
		s.castEndFrames(pl, def, sink, true)
		sink.flush()
		return
	}
	// 命中一次攻击者得怒/战意：FP/SP 成对 +2（每击一次，与目标数无关——
	// 挑战包一招双杀同样只发一对）。
	st.mu.Lock()
	st.live.hits++
	total := st.live.hits
	s.gainSPLocked(st, hitSPGain, sink)
	st.mu.Unlock()
	for _, item := range hit {
		sink.absorb(item.sink)
	}
	if total >= 2 {
		sink.custom(24, sink.playerObject(), entergen.CustomInt(10), entergen.CustomInt(int32(total)))
		sink.priv("MaxHits", entergen.Int32Value(int32(total)))
	}
	values := []entergen.CustomValue{sink.playerObject(), entergen.CustomString(def.ID)}
	for _, item := range hit {
		values = append(values, sink.npcValue(item.f))
	}
	sink.pubCustom(205, values...)

	// 末击：收招（抓包 208.741：CurSkillID="" → [m169 挂 buff] →
	// CurSkillTarget=None → m189；非锁定技只清 CurSkillID）。
	if index == len(plan.times) {
		s.castEndFrames(pl, def, sink, false)
		if def.Script == skilldata.ScriptLock {
			var landed []*npcFighter
			for _, item := range hit {
				landed = append(landed, item.f)
			}
			s.emitLockBuff(sink, pl, def, landed)
			sink.priv("CurSkillTarget", entergen.ObjectValue(0, 0))
			sink.pubCustom(189, sink.playerObject())
		}
	}
	sink.flush()
}

// absorb 并入另一帧汇的帧与广播主体（保持次序）。
func (k *frameSink) absorb(other *frameSink) {
	k.frames = append(k.frames, other.frames...)
	for _, f := range other.subjects {
		k.subject(f)
	}
	other.frames = nil
}

// skillView 技能行视图对象（官服：View40 行 = 技能在 roleState.Skills 的
// 0 基下标 + 1；不在 View40 的普攻族落 View41 行 1——抓包 zs_normal_00）。
func skillView(pl *Player, def skilldata.Definition) (view, row uint32) {
	for i, learned := range pl.roleState.Skills {
		if learned.ConfigID == def.ID {
			return 40, uint32(i + 1)
		}
	}
	return 41, 1
}

// fighterBySpawn 场景内一条摆放的战斗体（对象号 + own 校验）。
func (s *Server) fighterBySpawn(scene string, key uint64) *npcFighter {
	spawn, ok := s.findNPCSpawn(scene, uint32(key))
	if !ok || spawn.OwnerID != uint32(key>>32) || !attackable(spawn) {
		return nil
	}
	return s.fighterFor(scene, spawn)
}

// skillTargets 本击的结算目标：单体技取 m211 目标；范围技以玩家当前位
// 为中心按作用范围包（HitShape）与高度差重索敌，上限 HitCount。
func (s *Server) skillTargets(pl *Player, def skilldata.Definition, cast skillCast) []*npcFighter {
	pl.npcMu.Lock()
	scene := pl.npcScene
	pl.npcMu.Unlock()
	if scene == "" {
		return nil
	}
	if def.HitShape.Kind == 0 {
		if !cast.hasTarget {
			return nil
		}
		if f := s.fighterBySpawn(scene, cast.targetKey()); f != nil && !f.isDead() {
			return []*npcFighter{f}
		}
		return nil
	}
	origin := pl.roleState.Position
	var out []*npcFighter
	for _, spawn := range s.sceneNPCCatalog(scene).spawns {
		if !attackable(spawn) {
			continue
		}
		f := s.fighterFor(scene, spawn)
		f.mu.Lock()
		pos, dead := f.pos, f.dead
		f.mu.Unlock()
		if dead || !inShape(def, origin, pos) {
			continue
		}
		out = append(out, f)
		if def.HitCount > 0 && len(out) >= def.HitCount {
			break
		}
	}
	return out
}

// inShape 目标是否落在技能作用范围内（attack_hitshape：1 圆 / 3 扇形 /
// 4 矩形；高度差上限 HeightDiff）。
func inShape(def skilldata.Definition, origin, target [4]float64) bool {
	if def.HeightDiff > 0 && math.Abs(origin[1]-target[1]) > def.HeightDiff {
		return false
	}
	dx, dz := target[0]-origin[0], target[2]-origin[2]
	shape := def.HitShape
	switch shape.Kind {
	case 1:
		return math.Hypot(dx, dz) <= shape.Para2
	case 3, 4:
		// 朝向约定 atan2(dx, dz)：前向 (sin, cos)、右向 (cos, -sin)。
		sin, cos := math.Sin(float64(origin[3])), math.Cos(float64(origin[3]))
		forward, side := dx*sin+dz*cos, dx*cos-dz*sin
		if shape.Kind == 3 {
			if math.Hypot(dx, dz) > shape.Para2 {
				return false
			}
			half := shape.Para1 * 0.5 * math.Pi / 180
			return math.Abs(math.Atan2(side, forward)) <= half
		}
		return forward >= 0 && forward <= shape.Para2 && math.Abs(side) <= shape.Para1*0.5
	}
	return math.Hypot(dx, dz) <= shape.Para2
}

// hitNPC 对一个 NPC 结算一击；返回是否有效命中（闪避/已死亡为假）。
func (s *Server) hitNPC(pl *Player, def skilldata.Definition, f *npcFighter, hurt, count int, skillObj uint64, sink *frameSink) bool {
	st := &pl.combat
	key := uint64(f.spawn.ObjectID) | uint64(f.spawn.OwnerID)<<32
	now := combatNow()
	st.mu.Lock()
	f.mu.Lock()
	if f.dead {
		f.mu.Unlock()
		st.mu.Unlock()
		return false
	}
	first := f.logic != logicFighting
	if first {
		f.logic = logicFighting
		f.target = persistKey(pl.Conn1, pl.Conn2)
	}
	f.fightAt = now
	f.mu.Unlock()
	s.ensureLiveLocked(st, pl)
	entered := noteFightLocked(st, now)
	row := relationIndex(st.live.relations, key)
	if row < 0 {
		st.live.relations = append(st.live.relations, fightRelation{target: key, at: now})
	} else {
		st.live.relations[row].at = now
	}
	if first {
		sink.npcProp(f, "LogicState", entergen.ByteValue(logicFighting))
	}
	if row < 0 {
		sink.relationAdd(uint16(len(st.live.relations)-1), key, now)
	} else {
		sink.relationTouch(uint16(row), now)
	}
	if entered {
		sink.priv("LogicState", entergen.ByteValue(logicFighting))
	}
	if first {
		sink.pubCustom(24, sink.npcValue(f), entergen.CustomInt(37), entergen.CustomString("npc_foundplayer"))
	}
	s.armCalmLocked(st, pl, now)
	st.mu.Unlock()
	if first {
		s.npcReact(f)
	}

	targetName := f.spawn.ConfigID
	result := s.rollHit(pl, def, f, hurt, count, skillObj)
	if result.Miss {
		// 闪避：抓包 226.276（锁定技另配 <技能>_dodge 动作、m707[14]）。
		sink.pubCustom(24, sink.npcValue(f), entergen.CustomInt(7), entergen.CustomInt(1),
			sink.playerObject(), entergen.CustomString(def.ID), entergen.CustomString(targetName),
			entergen.CustomString(pl.Name), entergen.CustomInt(100))
		if def.Script == skilldata.ScriptLock {
			sink.pubCustom(20, sink.npcValue(f), entergen.CustomString(def.ID+"_dodge"), sink.playerObject())
			sink.custom(707, entergen.CustomInt(useSkillResultDodged))
		}
		return false
	}

	dealt := int(result.Damage)
	f.mu.Lock()
	dealt = min(dealt, f.hp)
	f.hp -= dealt
	killed := f.hp <= 0
	if killed {
		f.dead, f.deathAt, f.cast = true, now, nil
	}
	f.mu.Unlock()
	sink.subject(f)
	sink.pubCustom(24, sink.npcValue(f), entergen.CustomInt(17), entergen.CustomInt(int32(dealt)),
		entergen.CustomInt(0), entergen.CustomInt(0), entergen.CustomString(def.ID),
		entergen.CustomString(string(def.Script)), entergen.CustomString(targetName),
		entergen.CustomString(pl.Name), entergen.CustomInt(int32(btoi(result.Crit))))
	if def.Script != skilldata.ScriptLock && !strings.HasPrefix(def.ID, "zs_normal_") {
		// 非普攻：受击特效帧（抓包 cqgf04/cqgf06：body_hit_3_self）。
		sink.pubCustom(24, sink.npcValue(f), entergen.CustomInt(40),
			entergen.CustomString("body_hit_3_self"), sink.playerObject())
	}
	if killed {
		// 击杀簇（挑战包 483.077）：关系行删 → m24 29 → m18 → m24 13。
		st.mu.Lock()
		if row := relationIndex(st.live.relations, key); row >= 0 {
			sink.relationDel(uint16(row))
			st.live.relations = append(st.live.relations[:row], st.live.relations[row+1:]...)
		}
		st.mu.Unlock()
		sink.pubCustom(24, sink.npcValue(f), entergen.CustomInt(29), sink.playerObject(),
			entergen.CustomString(targetName), entergen.CustomString(pl.Name))
		sink.pubCustom(18, sink.npcValue(f), entergen.CustomString("hurt_half"), entergen.CustomInt(0), sink.playerObject())
		sink.pubCustom(24, sink.npcValue(f), entergen.CustomInt(13))
		s.after(corpseKeep, func() { s.npcCorpseRemove(f) })
		s.after(corpseKeep+npcRespawnDelay, func() { s.npcRespawn(f) })
	}
	if hook := s.combatLua().vm.Function(def.LuaScript, "skill_after_hit"); hook != nil {
		hook.Call(script.Ref(playerHandle(pl)),
			script.Ref(script.Handle{Kind: script.KindHelper, Helper: script.HelperSkill, ID: skillObj}),
			script.Ref(npcHandle(f)))
	}
	return true
}

// rollHit 一次命中的结算：技能脚本 get_damage（返回 5 系伤害）优先，否则
// 伤害计算包 × 公式引擎（Lua），引擎回退时走 Go 面板×倍率。
func (s *Server) rollHit(pl *Player, def skilldata.Definition, f *npcFighter, hurt, count int, skillObj uint64) script.HitResult {
	lua := s.combatLua()
	ctx := s.hitContext(playerCombatProps(pl, s), f.props, hurt, count,
		math.Hypot(pl.roleState.Position[0]-f.pos[0], pl.roleState.Position[2]-f.pos[2]), false)
	if hook := lua.vm.Function(def.LuaScript, "get_damage"); hook != nil {
		if out, err := hook.Call(script.Ref(playerHandle(pl)),
			script.Ref(script.Handle{Kind: script.KindHelper, Helper: script.HelperSkill, ID: skillObj}),
			script.Ref(npcHandle(f))); err == nil && len(out) >= 5 {
			raw := 0.0
			for _, v := range out[:5] {
				raw += v.Number()
			}
			return lua.engine.Settle(ctx, raw)
		}
	}
	if def.DamageCalculate == 0 || lua.skills == nil {
		return script.HitResult{}
	}
	total, crit, fallback := 0.0, false, false
	var last skilldata.DamageEntry
	for _, entry := range lua.skills.DamageCalculate(def.DamageCalculate) {
		last = entry
		ctx.FormulaID, ctx.Params = entry.Formula, entry.Args
		applyDamageType(&ctx, entry.DamageType)
		result := lua.engine.CalcHit(ctx)
		if result.Fallback {
			fallback = true
			continue
		}
		if result.Miss {
			return result
		}
		total += result.Damage
		crit = crit || result.Crit
	}
	if fallback {
		total += legacyDamage(ctx.Attacker, last, hurt)
	}
	return script.HitResult{Damage: total, Crit: crit}
}

// hitContext 组装每击求值输入（随机数宿主预掷）。
func (s *Server) hitContext(attacker, target map[string]float64, hurt, count int, distance float64, parrying bool) script.HitContext {
	ctx := script.HitContext{HurtCount: hurt, TargetCount: count, Attacker: attacker, Target: target,
		Distance: distance, Parrying: parrying,
		Accuracy: attacker["PhyHit"] + attacker["PhyHitAdd"], CritValue: attacker["PhyVa"] + attacker["PhyVaAdd"],
		Defense: target["PhyDef"], Evasion: target["Dodge"] + target["DodgeAdd"], Resilience: target["VaDef"]}
	for i := 0; i < 8; i++ {
		ctx.Rand = append(ctx.Rand, combatRoll())
	}
	if level := attacker["Level"]; level > 0 {
		ctx.SkillLevel = int(level)
	}
	return ctx
}

// applyDamageType 按伤害系（外功/刚柔阴阳）选对抗值。
func applyDamageType(ctx *script.HitContext, damageType int) {
	if damageType == 0 {
		return
	}
	ctx.Accuracy = ctx.Attacker["MagicHit"] + ctx.Attacker["MagicHitAdd"]
	ctx.CritValue = ctx.Attacker["MagicVa"] + ctx.Attacker["MagicVaAdd"]
	ctx.Defense = ctx.Target["MagicDef"]
}

// playerCombatProps 玩家战斗属性视图：登录面板数值 + 运行态生存值。
func playerCombatProps(pl *Player, s *Server) map[string]float64 {
	panel := s.playerPanel(pl)
	props := make(map[string]float64, len(panel)+6)
	for name, value := range panel {
		props[name] = value
	}
	st := &pl.combat
	st.mu.Lock()
	s.ensureLiveLocked(st, pl)
	props["HP"], props["MaxHP"] = float64(st.live.hp), float64(st.live.maxHP)
	props["MP"], props["MaxMP"] = float64(st.mp), float64(playerMaxMP(pl))
	props["SP"], props["MaxSP"] = float64(st.live.sp), float64(st.live.maxSP)
	st.mu.Unlock()
	return props
}

// legacyDamage 引擎回退面：基值 + 面板攻击区间×倍率，除以击数，保底 1。
func legacyDamage(props map[string]float64, entry skilldata.DamageEntry, hurt int) float64 {
	base, _ := strconv.ParseFloat(strings.TrimSpace(first(entry.Args)), 64)
	ratio := 1.0
	if len(entry.Args) > 1 {
		if value, err := strconv.ParseFloat(strings.TrimSpace(entry.Args[1]), 64); err == nil && value > 0 {
			ratio = value
		}
	}
	raw := base + ratio*(props["MinMeleeDamage"]+props["MaxMeleeDamage"])*0.15
	if hurt > 0 {
		raw /= float64(hurt)
	}
	return math.Max(1, math.Floor(raw))
}

func first(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

// emitLockBuff 锁定技末击给目标挂 buf_<技能>（抓包 208.741 的 m169：
// 到期 = 现在 + LifeTime，末位 = Category）。
func (s *Server) emitLockBuff(sink *frameSink, pl *Player, def skilldata.Definition, targets []*npcFighter) {
	lua := s.combatLua()
	if lua.skills == nil {
		return
	}
	buffID := "buf_" + def.ID
	if !lua.skills.BuffExists(buffID) {
		return
	}
	buff, err := lua.skills.Buff(buffID, def.Level)
	if err != nil || buff.LifeTimeMS <= 0 {
		return
	}
	for _, f := range targets {
		if f.isDead() {
			continue
		}
		_, frame := s.addBuff(npcHandle(f), combatBuff{id: buff.ID, level: def.Level, category: buff.Category,
			lua: buff.LuaScript, caster: playerHandle(pl)},
			time.Duration(buff.LifeTimeMS)*time.Millisecond, buff.EffectType, buff.EffectTimerMS, nil,
			s.combatLua().vm)
		sink.pub(frame, nil)
	}
}

// gainSPLocked 攻击者得怒/战意：FP → SP 成对 +（抓包帧序 FP 先于 SP；
// 调用方持 combatState.mu）。
func (s *Server) gainSPLocked(st *combatState, gain int, sink *frameSink) {
	live := &st.live
	if live.fp < live.maxSP {
		live.fp = min(live.fp+gain, live.maxSP)
		sink.priv("FP", entergen.Int32Value(int32(live.fp)))
	}
	if live.sp < live.maxSP {
		live.sp = min(live.sp+gain, live.maxSP)
		sink.priv("SP", entergen.Int32Value(int32(live.sp)))
	}
}

func (f *npcFighter) isDead() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.dead
}

func relationIndex(rows []fightRelation, target uint64) int {
	for i, row := range rows {
		if row.target == target {
			return i
		}
	}
	return -1
}
