package fssoloworld

// combatnpc.go NPC 战斗 AI：仇恨 → 追击 → 起手 → 命中玩家 → 脱战回归；
// 尸体移除与重生。
//
// 官服抓包（女号包 196~232s）：
//
//	发现目标  NPC.LogicState=1 → 玩家关系行加行 → 玩家.LogicState=1 →
//	          m24[npc,37,"npc_foundplayer"]
//	追击      NPC.State="run"（位置经 0x1F 广播）
//	起手      NPC.CurSkillID=技能 → CurSkillTarget=玩家 → [0x1F] →
//	          m20[npc,动作,玩家,0,0.0]（动作 = 流程包 HitAction 轮转）
//	命中      m24[玩家,17,伤害,0,0,技能,"SkillNormal",玩家名,NPC名,0]
//	          → 玩家 HitHPRatio/HitHP → m205[npc,技能,玩家] → 收招
//	          （NPC.CurSkillID="" → CurSkillTarget=None → m189[npc]）
//	招架      m24[玩家,8,1,npc,技能,玩家名,NPC名] → 招架耐力 −5、怒 +4 →
//	          穿透伤害照走 m24 17
//	闪避      玩家 LastAttackerNpc/LastDodge → m24[玩家,7,1,npc,技能,…,100]
//	脱战      NPC.LogicState=0
//
// NPC 的 HP 不以属性帧下发，起手周期 npcAttackPeriod。

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/skilldata"

	"github.com/lizhanyin/9yin-server-go/internal/script"
)

// npcSkillRef NPC 可用的一个技能（战斗行技能列联查后的定义 + 起手动作）。
type npcSkillRef struct {
	def    skilldata.Definition
	action string
}

// npcSkills NPC 的技能表：战斗行 SkillID1..8（静态包号 → ConfigID）按
// SkillLevelN 取定义；全缺回落 default_normal_skill。
func (s *Server) npcSkills(f *npcFighter) []npcSkillRef {
	if f.skills != nil {
		return f.skills
	}
	lua := s.combatLua()
	var refs []npcSkillRef
	if lua.skills != nil && f.spawn.Combat != nil {
		row := f.spawn.Combat
		for i := 1; i <= 8; i++ {
			raw := strings.TrimSpace(row.Attrs["SkillID"+strconv.Itoa(i)])
			staticData, err := strconv.Atoi(raw)
			if err != nil || staticData <= 0 {
				continue
			}
			configID, ok := lua.skills.SkillIDByStaticData(staticData)
			if !ok {
				continue
			}
			def, err := lua.skills.Definition(configID, row.Int("SkillLevel"+strconv.Itoa(i)))
			if err != nil || def.DamageCalculate == 0 {
				continue
			}
			action := def.ID
			if len(def.Flow.HitActions) > 0 {
				action = def.Flow.HitActions[0]
			}
			refs = append(refs, npcSkillRef{def: def, action: action})
		}
	}
	if len(refs) == 0 && lua.skills != nil {
		if def, err := lua.skills.Definition("default_normal_skill", 1); err == nil {
			refs = append(refs, npcSkillRef{def: def, action: "atkhit1"})
		}
	}
	return refs
}

// npcWake NPC 进入战斗（被击/主动索敌）：登记关系并排下一拍决策。
func (s *Server) npcWake(f *npcFighter, target uint64) {
	f.mu.Lock()
	if f.dead {
		f.mu.Unlock()
		return
	}
	f.logic = logicFighting
	f.target = target
	f.fightAt = combatNow()
	f.mu.Unlock()
	s.npcReact(f)
}

// npcReact 反应延迟后排一次决策（被击/发现目标后不立即还手）。
func (s *Server) npcReact(f *npcFighter) {
	if npcAIEnabled() {
		s.after(npcReactDelay, func() { s.npcThink(f) })
	}
}

// npcThink NPC 战斗决策：目标失效/超时 → 脱战；超距 → 追击；贴身 → 起手。
func (s *Server) npcThink(f *npcFighter) {
	pl := s.playerByKey(f.target)
	now := combatNow()
	f.mu.Lock()
	if passive(f.spawn) {
		// 不还手（CantAttack=1，训练木桩）：战斗态只用于交手帧，不决策。
		f.mu.Unlock()
		return
	}
	valid := pl != nil && pl.State == PlOnline
	if valid {
		pl.npcMu.Lock()
		valid = pl.npcScene == f.scene
		pl.npcMu.Unlock()
	}
	if f.dead || f.logic != logicFighting || !valid || f.cast != nil {
		f.mu.Unlock()
		if f.logic == logicFighting && !f.dead {
			s.npcCalm(f, "target gone")
		}
		return
	}
	if now.Before(f.stunUntil) {
		// 受击硬直 / 击退在途（combatcontrol.go）：不动手，硬直接束时再接上。
		f.mu.Unlock()
		return
	}
	target := pl.roleState.Position
	dx, dz := target[0]-f.pos[0], target[2]-f.pos[2]
	distance := math.Hypot(dx, dz)
	// 目标拉开且久未交手 → 脱战回归。
	if now.Sub(f.fightAt) >= npcCalmDelay && distance > npcAttackRange*4 {
		f.mu.Unlock()
		s.npcCalm(f, "target escaped")
		return
	}
	sink := s.sinkFor(pl)
	if distance > npcAttackRange {
		// 追击：向目标推进一步，位置经 0x1F 广播，State=run。
		speed := npcRunSpeed(f)
		step := min(distance-2.0, speed*npcChaseStep.Seconds())
		if norm := math.Hypot(dx, dz); norm > 1e-6 {
			f.pos[0] += dx / norm * step
			f.pos[2] += dz / norm * step
			f.pos[3] = math.Atan2(dx, dz)
		}
		if !f.moving {
			f.moving = true
			sink.npcProp(f, "State", entergen.StringValue(enterRUN))
		}
		sink.add(entergen.LocationFrame(f.spawn.ObjectID, f.spawn.OwnerID,
			float32(f.pos[0]), float32(f.pos[1]), float32(f.pos[2]), float32(f.pos[3])), nil, false)
		f.mu.Unlock()
		s.after(npcChaseStep, func() { s.npcThink(f) })
		sink.flush()
		return
	}
	// 贴身且过了起手间隔：起手。
	if now.Before(f.nextCast) {
		f.mu.Unlock()
		s.after(npcChaseStep, func() { s.npcThink(f) })
		return
	}
	refs := s.npcSkills(f)
	if len(refs) == 0 {
		f.mu.Unlock()
		return
	}
	f.step++
	ref := refs[(f.step-1)%len(refs)]
	if len(ref.def.Flow.HitActions) > 0 {
		ref.action = ref.def.Flow.HitActions[(f.step-1)%len(ref.def.Flow.HitActions)]
	}
	wasMoving := f.moving
	f.moving = false
	f.castSeq++
	f.cast = &npcCast{skill: ref.def, action: ref.action, seq: f.castSeq}
	f.cast.aux = s.registerAux(&auxObject{kind: script.HelperSkill, config: ref.def.ID, script: ref.def.LuaScript,
		def: ref.def, props: map[string]script.Value{}, records: map[string][][]script.Value{}}).ID
	f.nextCast = now.Add(npcAttackPeriod)
	f.fightAt = now
	ident := pl.wireIdent().orFixed()
	sink.subject(f)
	sink.npcProp(f, "CurSkillID", entergen.StringValue(ref.def.ID))
	sink.npcProp(f, "CurSkillTarget", entergen.ObjectValue(ident.objectID, ident.ownerID))
	if wasMoving {
		sink.npcProp(f, "State", entergen.StringValue(""))
	}
	sink.add(entergen.LocationFrame(f.spawn.ObjectID, f.spawn.OwnerID,
		float32(f.pos[0]), float32(f.pos[1]), float32(f.pos[2]), float32(f.pos[3])), nil, false)
	sink.pubCustom(20, sink.npcValue(f), entergen.CustomString(ref.action), sink.playerObject(),
		entergen.CustomInt(0), entergen.CustomFloat32(0))
	plan := s.skillHitPlan(ref.action, ref.def)
	seq := f.castSeq
	for i := range plan.times {
		index := i + 1
		s.after(plan.times[i], func() { s.npcResolveHit(f, seq, ref, plan, index) })
	}
	f.mu.Unlock()
	sink.flush()
}

// npcRunSpeed 追击速度（m/s）：战斗行 RunSpeed（十分之一米单位）/10，缺省 4.5。
func npcRunSpeed(f *npcFighter) float64 {
	speed := 4.5
	if f.spawn.Combat != nil {
		if value := f.spawn.Combat.Int("RunSpeed"); value > 0 {
			speed = float64(value) / 10
		}
	}
	return math.Max(speed, 1.0)
}

// npcResolveHit NPC 一次起手的第 index 击（1 基）：对仇恨目标结算。
func (s *Server) npcResolveHit(f *npcFighter, seq int, ref npcSkillRef, plan hitPlan, index int) {
	pl := s.playerByKey(f.target)
	f.mu.Lock()
	active := !f.dead && f.cast != nil && f.castSeq == seq
	npcPos := f.pos
	f.mu.Unlock()
	if !active || pl == nil || pl.State != PlOnline {
		return
	}
	pl.npcMu.Lock()
	sceneOK := pl.npcScene == f.scene
	pl.npcMu.Unlock()
	if !sceneOK {
		return
	}

	now := combatNow()
	st := &pl.combat
	key := uint64(f.spawn.ObjectID) | uint64(f.spawn.OwnerID)<<32
	st.mu.Lock()
	s.ensureLiveLocked(st, pl)
	playerPos := pl.roleState.Position
	entered := noteFightLocked(st, now)
	relation := relationIndex(st.live.relations, key)
	if relation < 0 {
		st.live.relations = append(st.live.relations, fightRelation{target: key, at: now})
	} else {
		st.live.relations[relation].at = now
	}
	props := make(map[string]float64, len(s.playerPanel(pl))+6)
	for name, value := range s.playerPanel(pl) {
		props[name] = value
	}
	props["HP"], props["MaxHP"] = float64(st.live.hp), float64(st.live.maxHP)
	props["MP"], props["MaxMP"] = float64(st.mp), float64(playerMaxMP(pl))
	props["SP"], props["MaxSP"] = float64(st.live.sp), float64(st.live.maxSP) // 面板里的 SP 是进场值，怒气读会话态
	if level := s.playerPanel(pl)["Level"]; level > 0 {
		props["Level"] = level
	}
	maxParry := s.parryValueTotal(pl)
	st.mu.Unlock()
	pl.parryMu.Lock()
	parrying := pl.parrying && maxParry > 0
	pl.parryMu.Unlock()

	distance := math.Hypot(playerPos[0]-npcPos[0], playerPos[2]-npcPos[2])
	// 攻方 = NPC、守方 = 玩家。
	ctx := s.hitContext(f.props, props, plan.hurt, 1, distance, parrying)
	result := s.npcRoll(ref.def, ctx)
	class := string(ref.def.Script)
	if class == "" {
		class = string(skilldata.ScriptNormal)
	}

	sink := s.sinkFor(pl)
	sink.subject(f)
	st.mu.Lock()
	// 帧序按结果分形（抓包 218.058 / 225.883 / 223.954）：
	//
	//	闪避  LastAttackerNpc → LastDodge → m24 7 → 关系行
	//	招架  m24 8 → 关系行 → 怒 +4 → 招架耐力 −5 → 战意 +2 → HP
	//	命中  怒 +1 → HP → 关系行
	//
	// 其后（未闪避）m24 17。
	relationFrame := func() {
		s.armCalmLocked(st, pl, now)
		if relation < 0 {
			sink.relationAdd(uint16(len(st.live.relations)-1), key, now)
		} else {
			sink.relationTouch(uint16(relation), now)
		}
		if entered {
			sink.priv("LogicState", entergen.ByteValue(logicFighting))
		}
	}
	applyHit := func() {
		// 玩家死亡流程未接：致命伤钳到 1 点。
		st.live.hp = max(st.live.hp-int(result.Damage), 1)
		sink.priv("HPRatio", entergen.Int32Value(hpRatio(st.live.hp, st.live.maxHP)))
		sink.priv("HP", entergen.Int32Value(int32(st.live.hp)))
	}
	switch {
	case result.Miss:
		sink.priv("LastAttackerNpc", entergen.StringValue(f.spawn.ConfigID))
		sink.priv("LastDodge", entergen.Float32Value(10))
		sink.pubCustom(24, sink.playerObject(), entergen.CustomInt(7), entergen.CustomInt(1),
			sink.npcValue(f), entergen.CustomString(ref.def.ID), entergen.CustomString(pl.Name),
			entergen.CustomString(f.spawn.ConfigID), entergen.CustomInt(100))
		relationFrame()
	case result.Parried:
		sink.pubCustom(24, sink.playerObject(), entergen.CustomInt(8), entergen.CustomInt(1),
			sink.npcValue(f), entergen.CustomString(ref.def.ID), entergen.CustomString(pl.Name),
			entergen.CustomString(f.spawn.ConfigID))
		relationFrame()
		if st.live.sp < st.live.maxSP {
			st.live.sp = min(st.live.sp+parryTakenSPGain, st.live.maxSP)
			sink.priv("SP", entergen.Int32Value(int32(st.live.sp)))
		}
		st.live.parry = max(st.live.parry-parryValuePerHit, 0)
		sink.priv("ParryRatio", entergen.Int32Value(hpRatio(st.live.parry, int(maxParry))))
		sink.priv("ParryValue", entergen.Int32Value(int32(st.live.parry)))
		if st.live.fp < st.live.maxSP {
			st.live.fp = min(st.live.fp+hitSPGain, st.live.maxSP)
			sink.priv("FP", entergen.Int32Value(int32(st.live.fp)))
		}
		applyHit()
	default:
		if st.live.sp < st.live.maxSP {
			st.live.sp = min(st.live.sp+hitTakenSPGain, st.live.maxSP)
			sink.priv("SP", entergen.Int32Value(int32(st.live.sp)))
		}
		applyHit()
		relationFrame()
	}
	if !result.Miss {
		sink.pubCustom(24, sink.playerObject(), entergen.CustomInt(17), entergen.CustomInt(int32(result.Damage)),
			entergen.CustomInt(0), entergen.CustomInt(0), entergen.CustomString(ref.def.ID),
			entergen.CustomString(class), entergen.CustomString(pl.Name), entergen.CustomString(f.spawn.ConfigID),
			entergen.CustomInt(int32(btoi(result.Crit))))
	}
	st.mu.Unlock()
	sink.pubCustom(205, sink.npcValue(f), entergen.CustomString(ref.def.ID), sink.playerObject())

	// 末击收招：NPC.State=""（追击过来时）→ CurSkillID="" →
	// CurSkillTarget=None → m189[npc]；隔一个起手周期再决策。
	if index == len(plan.times) {
		f.mu.Lock()
		if f.castSeq == seq {
			f.cast = nil
		}
		wasMoving := f.moving
		f.moving = false
		f.mu.Unlock()
		if wasMoving {
			sink.npcProp(f, "State", entergen.StringValue(""))
		}
		sink.npcProp(f, "CurSkillID", entergen.StringValue(""))
		sink.npcProp(f, "CurSkillTarget", entergen.ObjectValue(0, 0))
		sink.pubCustom(189, sink.npcValue(f))
		s.after(npcAttackPeriod-plan.last(), func() { s.npcThink(f) })
	}
	sink.flush()
}

// npcRoll NPC 一击的结算：伤害计算包 × 公式引擎，引擎回退走 Go 面。
func (s *Server) npcRoll(def skilldata.Definition, ctx script.HitContext) script.HitResult {
	lua := s.combatLua()
	if def.DamageCalculate == 0 || lua.skills == nil {
		return script.HitResult{Damage: 0}
	}
	total, miss, parried, crit, fallback := 0.0, false, false, false, false
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
		miss, parried = miss || result.Miss, parried || result.Parried
		crit = crit || result.Crit
		total += result.Damage
	}
	if miss {
		return script.HitResult{Miss: true}
	}
	if fallback {
		if ctx.Parrying {
			total += math.Max(1, math.Floor(legacyDamage(ctx.Attacker, last, ctx.HurtCount)*parryFactor))
		} else {
			total += legacyDamage(ctx.Attacker, last, ctx.HurtCount)
		}
	}
	return script.HitResult{Damage: total, Parried: parried, Crit: crit}
}

// npcCalm NPC 脱战：回常态、关系清空并走回出生点（出生点处回满血）。
func (s *Server) npcCalm(f *npcFighter, reason string) {
	f.mu.Lock()
	if f.dead || f.logic != logicFighting {
		f.mu.Unlock()
		return
	}
	f.logic, f.target, f.cast, f.fightAt = logicNormal, 0, nil, time.Time{}
	f.mu.Unlock()
	s.npcBroadcast(f, npcProperty(f, "LogicState", entergen.ByteValue(logicNormal)))
	s.trace("combat: npc %s 脱战（%s）", f.spawn.ConfigID, reason)
	if !passive(f.spawn) {
		s.npcReturn(f) // 训练木桩不归位——也就不走出生点回满血
	}
}

// npcBroadcast 给同场景看得见该 NPC 的全部玩家下发帧。
func (s *Server) npcBroadcast(f *npcFighter, frames ...[]byte) {
	for _, pl := range s.players.Snapshot() {
		if pl.State != PlOnline {
			continue
		}
		pl.npcMu.Lock()
		ok := pl.npcScene == f.scene && (pl.npcVisible == nil || pl.npcVisible[f.spawn.ObjectID])
		pl.npcMu.Unlock()
		if !ok {
			continue
		}
		for _, frame := range frames {
			s.sendToClient(pl, frame)
		}
	}
}

func npcProperty(f *npcFighter, name string, value entergen.PropertyValue) []byte {
	frame, err := entergen.ObjectPropertyFrame(f.spawn.ObjectID, f.spawn.OwnerID, 0,
		[]entergen.IndexedProperty{entergen.NamedProperty(name, value)})
	if err != nil {
		return nil
	}
	return frame
}

// npcReturn 走回出生点（每拍推进一步；到位后恢复姿态并回满血）。
func (s *Server) npcReturn(f *npcFighter) {
	f.mu.Lock()
	if f.dead || f.logic == logicFighting {
		f.mu.Unlock()
		return
	}
	home := [4]float64{float64(f.spawn.X), float64(f.spawn.Y), float64(f.spawn.Z), float64(f.spawn.Orient)}
	dx, dz := home[0]-f.pos[0], home[2]-f.pos[2]
	var frames [][]byte
	var removals [][]byte
	if math.Hypot(dx, dz) <= 0.5 {
		// 回满血前把身上的 buff 逐个摘掉（发 m169 摘帧），不静默清空。
		for _, buff := range f.buffs {
			removals = append(removals, s.buffFrame(false, npcHandle(f), buff))
		}
		f.pos, f.hp, f.buffs, f.moving = home, int(f.props["MaxHP"]), nil, false
		rest := f.spawn.RestAction()
		if frame := npcProperty(f, "State", entergen.StringValue(rest)); frame != nil {
			frames = append(frames, frame)
		}
	} else {
		step := min(math.Hypot(dx, dz), npcRunSpeed(f)*npcChaseStep.Seconds())
		f.pos[0] += dx / math.Hypot(dx, dz) * step
		f.pos[2] += dz / math.Hypot(dx, dz) * step
		f.pos[3] = math.Atan2(dx, dz)
		if !f.moving {
			f.moving = true
			frames = append(frames, npcProperty(f, "State", entergen.StringValue(enterRUN)))
		}
		frames = append(frames, entergen.LocationFrame(f.spawn.ObjectID, f.spawn.OwnerID,
			float32(f.pos[0]), float32(f.pos[1]), float32(f.pos[2]), float32(f.pos[3])))
		s.after(npcChaseStep, func() { s.npcReturn(f) })
	}
	f.mu.Unlock()
	s.npcBroadcast(f, append(frames, removals...)...)
}

// npcCorpseRemove 尸体消失：对看得见它的玩家发 0x0E 移除并清视口簿记。
func (s *Server) npcCorpseRemove(f *npcFighter) {
	if !f.isDead() {
		return
	}
	frame := entergen.SceneRemoveObjectFrame(f.spawn.ObjectID, f.spawn.OwnerID)
	for _, pl := range s.players.Snapshot() {
		if pl.State != PlOnline {
			continue
		}
		pl.npcMu.Lock()
		ok := pl.npcScene == f.scene && pl.npcVisible != nil && pl.npcVisible[f.spawn.ObjectID]
		if ok {
			delete(pl.npcVisible, f.spawn.ObjectID)
			delete(pl.taskMarks, f.spawn.ObjectID)
		}
		pl.npcMu.Unlock()
		if ok {
			s.sendToClient(pl, frame)
		}
	}
}

// npcRespawn 重生：回出生位满血，对同场景玩家重新物化。
func (s *Server) npcRespawn(f *npcFighter) {
	f.mu.Lock()
	if !f.dead {
		f.mu.Unlock()
		return
	}
	f.dead, f.hp, f.buffs = false, int(f.props["MaxHP"]), nil
	f.logic, f.target, f.cast, f.fightAt = logicNormal, 0, nil, time.Time{}
	f.stunUntil, f.knockbacks = time.Time{}, 0
	f.stunSeq++ // 死前的硬直收尾回调作废
	f.pos = [4]float64{float64(f.spawn.X), float64(f.spawn.Y), float64(f.spawn.Z), float64(f.spawn.Orient)}
	spawn := f.spawn
	f.mu.Unlock()
	frames, err := npcfeed.Frames([]npcfeed.Spawn{spawn}, npcForm())
	if err != nil {
		return
	}
	radius := npcRadius()
	for _, pl := range s.players.Snapshot() {
		if pl.State != PlOnline {
			continue
		}
		pl.npcMu.Lock()
		ok := pl.npcScene == f.scene
		if ok && radius > 0 && pl.npcVisible != nil {
			dx, dz := pl.npcPos[0]-float64(spawn.X), pl.npcPos[2]-float64(spawn.Z)
			ok = dx*dx+dz*dz <= radius*radius
		}
		if ok && pl.npcVisible != nil {
			pl.npcVisible[spawn.ObjectID] = true
		}
		pl.npcMu.Unlock()
		if ok {
			for _, frame := range frames {
				s.sendToClient(pl, frame)
			}
		}
	}
}

// npcAggroTick 主动索敌（每秒）：战斗类 NPC 的警戒半径内有在线玩家即
// 进入战斗。警戒半径取战斗行 AlarmRange（分米），缺省不主动索敌。
func (s *Server) npcAggroTick() {
	if !combatEnabled() {
		return
	}
	for _, pl := range s.players.Snapshot() {
		if pl.State != PlOnline {
			continue
		}
		pl.npcMu.Lock()
		scene := pl.npcScene
		pl.npcMu.Unlock()
		if scene == "" {
			continue
		}
		pos := pl.roleState.Position
		for _, spawn := range s.sceneNPCCatalog(scene).spawns {
			if !attackable(spawn) || spawn.Combat == nil {
				continue
			}
			alarm := spawn.Combat.Int("AlarmRange")
			if alarm <= 0 {
				continue
			}
			radius := float64(alarm) / 10
			dx, dz := pos[0]-float64(spawn.X), pos[2]-float64(spawn.Z)
			if dx*dx+dz*dz > radius*radius {
				continue
			}
			if f := s.fighterFor(scene, spawn); !f.isDead() && f.logicState() == logicNormal {
				s.npcWake(f, persistKey(pl.Conn1, pl.Conn2))
			}
		}
	}
}

func (f *npcFighter) logicState() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.logic
}

const enterRUN = "run"
