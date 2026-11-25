package fssoloworld

// combatbuff.go buff 挂载/到期/周期触发，以及脚本侧入口（nx_add_buffer /
// nx_find_buffer / nx_remove_buffer / nx_damage_target / nx_break_any_flow）。
//
// 官服 buff 帧方言（抓包）：
//
//	m169[0 挂/1 摘, buffID, 目标, 施法者, 目标名, 施法者名, 到期毫秒(1601
//	     纪元), Category]   Category = buff_static.Category；到期 = 挂载
//	     时刻 + LifeTime（女号包 buf_CS_jh_cqgf01：+5000ms、尾值 73）
//	m24[目标, 22, 伤害, 0, 0, buffID, "Buffer", 目标名, 施法者名]
//	     buff 伤害，到施法者名为止（没有 17 帧那样的末位暴击标记）；
//	     挑战包 5.219 / 8.739：eqp_lzzy_003 每跳 1
//
// buff 记录表（官方 BUFF常用函数.txt）：DirectPropertyRec = 挂载即改的
// 属性（列 3 属性名、列 4 值），ContinuePropertyRec = 每次触发连续改
// （如连续扣血）。钩子 buffer_active / buffer_deactive / buffer_timer。
//
// 锁序：挂载点的锁（玩家 combatState.mu / NPC npcFighter.mu）只在
// withBuffs 内短暂持有；调 Lua 钩子与落帧都在锁外。

import (
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"

	"github.com/lizhanyin/9yin-server-go/internal/script"
)

// withBuffs 在持有挂载点锁的情况下读写 buff 列表；目标不存在返回 false。
func (s *Server) withBuffs(target script.Handle, fn func(list *[]combatBuff)) bool {
	switch target.Kind {
	case script.KindNPC:
		f := s.fighterByHandle(target.ID)
		if f == nil {
			return false
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		fn(&f.buffs)
	case script.KindPlayer:
		pl := s.playerByKey(target.ID)
		if pl == nil {
			return false
		}
		st := &pl.combat
		st.mu.Lock()
		defer st.mu.Unlock()
		s.ensureLiveLocked(st, pl)
		fn(&st.live.buffs)
	default:
		return false
	}
	return true
}

// buffsOf 目标身上 buff 的快照（只读面：测试与查询）。
func (s *Server) buffsOf(target script.Handle) []combatBuff {
	var out []combatBuff
	s.withBuffs(target, func(list *[]combatBuff) { out = append(out, *list...) })
	return out
}

func upsertBuff(list []combatBuff, buff combatBuff) []combatBuff {
	for i := range list {
		if list[i].id == buff.id {
			list[i] = buff
			return list
		}
	}
	return append(list, buff)
}

// addBuff buff 落挂载点：到期事件与周期触发（effectType 3 且 timerMS>0）
// 排队，记录表效果落地；返回到期时刻与 m169 挂载帧（调用方决定入汇还是
// 直发）。
func (s *Server) addBuff(target script.Handle, buff combatBuff, life time.Duration, effectType, timerMS int,
	direct []propEffect, lua script.Caller) (time.Time, []byte) {
	buff.expireAt = combatNow().Add(life)
	s.withBuffs(target, func(list *[]combatBuff) { *list = upsertBuff(*list, buff) })
	if fn := lua.Function(buff.lua, "buffer_active"); fn != nil && buff.handle != 0 {
		fn.Call(script.Ref(target), script.Ref(script.Handle{Kind: script.KindHelper, Helper: script.HelperBuffer, ID: buff.handle}))
	}
	for _, effect := range direct {
		s.applyProperty(buff.caster, target, effect)
	}
	s.after(life, func() { s.expireBuff(target, buff) })
	expiry := buff.expireAt
	if effectType == 3 && timerMS > 0 {
		s.after(time.Duration(timerMS)*time.Millisecond, func() { s.buffTick(target, buff, timerMS) })
	}
	return expiry, s.buffFrame(true, target, buff)
}

// expireBuff 到期摘除（到期时刻不符 = 已被刷新，丢弃）：钩子
// buffer_deactive → 关系帧 m169 摘。
func (s *Server) expireBuff(target script.Handle, buff combatBuff) {
	removed := false
	s.withBuffs(target, func(list *[]combatBuff) {
		for i := range *list {
			if (*list)[i].id == buff.id && (*list)[i].expireAt.Equal(buff.expireAt) {
				*list = append((*list)[:i], (*list)[i+1:]...)
				removed = true
				return
			}
		}
	})
	if !removed {
		return
	}
	if fn := s.combatLua().vm.Function(buff.lua, "buffer_deactive"); fn != nil && buff.handle != 0 {
		fn.Call(script.Ref(target), script.Ref(script.Handle{Kind: script.KindHelper, Helper: script.HelperBuffer, ID: buff.handle}))
	}
	s.deliverBuffFrame(target, s.buffFrame(false, target, buff))
}

// buffTick 周期触发（EffectType 3：X 秒内每 EffectTimer 一次）：钩子
// buffer_timer（计次型返回 0 = 本次不计）→ 持续变化的 HP 增减 → 排下一拍。
func (s *Server) buffTick(target script.Handle, buff combatBuff, timerMS int) {
	found := false
	s.withBuffs(target, func(list *[]combatBuff) {
		for _, current := range *list {
			found = found || (current.id == buff.id && current.expireAt.Equal(buff.expireAt))
		}
	})
	if !found {
		return
	}
	counted := true
	if fn := s.combatLua().vm.Function(buff.lua, "buffer_timer"); fn != nil && buff.handle != 0 {
		if out, err := fn.Call(script.Ref(target),
			script.Ref(script.Handle{Kind: script.KindHelper, Helper: script.HelperBuffer, ID: buff.handle})); err == nil && len(out) > 0 {
			counted = out[0].Number() != 0
		}
	}
	if counted && buff.dotHP != 0 {
		s.applyDamage(buff.caster, target, script.HitResult{Damage: -buff.dotHP},
			damageSource{skill: buff.id, class: "Buffer", byBuff: true})
	}
	s.after(time.Duration(timerMS)*time.Millisecond, func() { s.buffTick(target, buff, timerMS) })
}

// propEffect 记录表里的一条属性效果。
type propEffect struct {
	name  string
	value float64
}

// applyProperty 记录表效果落地：当前只接 HP（正 = 治疗、负 = 伤害）。
func (s *Server) applyProperty(caster, target script.Handle, effect propEffect) {
	if effect.name != "HP" || effect.value == 0 {
		return
	}
	s.applyDamage(caster, target, script.HitResult{Damage: -effect.value},
		damageSource{class: "Buffer", byBuff: true})
}

// buffFrame m169 加/摘帧：[169, 0 挂/1 摘, buffID, 目标, 施法者, 目标名,
// 施法者名, 到期毫秒(1601 纪元), Category]。
func (s *Server) buffFrame(add bool, target script.Handle, buff combatBuff) []byte {
	targetIdent, targetName, _ := s.identOf(target)
	casterIdent, casterName, _ := s.identOf(buff.caster)
	flag := int32(1)
	if add {
		flag = 0
	}
	frame, err := entergen.CustomMessageFrame(169, entergen.CustomInt(flag), entergen.CustomString(buff.id),
		entergen.CustomObject(targetIdent.objectID, targetIdent.ownerID),
		entergen.CustomObject(casterIdent.objectID, casterIdent.ownerID),
		entergen.CustomString(targetName), entergen.CustomString(casterName),
		entergen.CustomInt64(filetimeMS(buff.expireAt)), entergen.CustomInt(int32(buff.category)))
	if err != nil {
		return nil
	}
	return frame
}

// deliverBuffFrame m169 直发：目标是 NPC → 广播给看得见它的玩家；是玩家
// → 只发本人。
func (s *Server) deliverBuffFrame(target script.Handle, frame []byte) {
	if frame == nil {
		return
	}
	if target.Kind == script.KindPlayer {
		if pl := s.playerByKey(target.ID); pl != nil {
			s.sendToClient(pl, frame)
		}
		return
	}
	if f := s.fighterByHandle(target.ID); f != nil {
		s.npcBroadcast(f, frame)
	}
}

// identOf 句柄的 wire 身份与显示名（第三个返回值：玩家本人，非玩家为 nil）。
func (s *Server) identOf(h script.Handle) (ident wireIdent, name string, pl *Player) {
	switch h.Kind {
	case script.KindPlayer:
		if p := s.playerByKey(h.ID); p != nil {
			return p.wireIdent().orFixed(), p.Name, p
		}
	case script.KindNPC:
		if f := s.fighterByHandle(h.ID); f != nil {
			return wireIdent{objectID: f.spawn.ObjectID, ownerID: f.spawn.OwnerID}, f.spawn.ConfigID, nil
		}
	}
	return wireIdent{}, "", nil
}

// ---- 脚本侧入口 ----

// attachBuffAux nx_add_buffer：把脚本建好并设置过的 buff 对象挂到目标。
func (s *Server) attachBuffAux(target, caster script.Handle, auxID uint64) bool {
	aux := s.auxByHandle(auxID)
	lua := s.combatLua()
	if aux == nil || aux.kind != script.HelperBuffer || !target.Valid() || lua.skills == nil {
		return false
	}
	aux.mu.Lock()
	config, scriptPath := aux.config, aux.script
	number := func(name string) int { return int(aux.props[name].Number()) }
	level, life, effectType, timerMS := number("Level"), number("LifeTime"), number("EffectType"), number("EffectTimer")
	var direct []propEffect
	dot := 0.0
	for _, record := range []string{"DirectPropertyRec", "ContinuePropertyRec"} {
		for row := 0; row < len(aux.records[record]); row++ {
			cells := aux.records[record][row]
			if len(cells) < 5 || cells[3].Text != "HP" {
				continue
			}
			if record == "DirectPropertyRec" {
				direct = append(direct, propEffect{name: "HP", value: cells[4].Number()})
			} else {
				dot += cells[4].Number()
			}
		}
	}
	aux.owner = caster
	aux.mu.Unlock()
	if level < 1 {
		level = 1
	}
	def, err := lua.skills.Buff(config, level)
	if err != nil {
		return false
	}
	if life <= 0 {
		life = def.LifeTimeMS
	}
	buff := combatBuff{id: config, level: level, category: def.Category, lua: scriptPath,
		dotHP: dot, handle: auxID, caster: caster}
	// 本函数运行在 nx_add_buffer 的回调里（VM 锁已由外层脚本调用持有）：
	// 钩子经回调语境调用面取。
	_, frame := s.addBuff(target, buff, time.Duration(max(life, 1))*time.Millisecond, effectType, timerMS, direct,
		script.Reentrant{VM: lua.vm})
	s.deliverBuffFrame(target, frame)
	return true
}

// findBuff nx_find_buffer：目标身上该 ConfigID 的 buff 对象句柄。
func (s *Server) findBuff(target script.Handle, configID string) (script.Handle, bool) {
	var found *combatBuff
	s.withBuffs(target, func(list *[]combatBuff) {
		for i := range *list {
			if (*list)[i].id == configID {
				found = &(*list)[i]
			}
		}
	})
	if found == nil {
		return script.NullHandle, false
	}
	if found.handle == 0 {
		// 锁定技挂的 buff 没有脚本对象：按需补建，脚本可查可摘。
		aux := &auxObject{kind: script.HelperBuffer, config: found.id, script: found.lua, owner: found.caster,
			props:   map[string]script.Value{"ConfigID": script.String(found.id), "Level": script.Int(int64(found.level))},
			records: map[string][][]script.Value{}}
		s.withBuffs(target, func(list *[]combatBuff) {
			for i := range *list {
				if (*list)[i].id == configID {
					(*list)[i].handle = s.registerAux(aux).ID
					found = &(*list)[i]
				}
			}
		})
	}
	return script.Handle{Kind: script.KindHelper, Helper: script.HelperBuffer, ID: found.handle}, true
}

// detachBuffByHandle 按脚本对象摘 buff（nx_remove_buffer / nx_destroy_self）。
func (s *Server) detachBuffByHandle(auxID uint64) {
	for _, f := range s.allFighters() {
		s.detachBuffWhere(npcHandle(f), func(buff combatBuff) bool { return buff.handle == auxID })
	}
	for _, pl := range s.players.Snapshot() {
		s.detachBuffWhere(playerHandle(pl), func(buff combatBuff) bool { return buff.handle == auxID })
	}
}

// detachBuffWhere 摘除目标身上满足条件的 buff（逐个发 m169 摘帧）。
func (s *Server) detachBuffWhere(target script.Handle, match func(combatBuff) bool) {
	for {
		var removed *combatBuff
		s.withBuffs(target, func(list *[]combatBuff) {
			for i := range *list {
				if match((*list)[i]) {
					removed = &(*list)[i]
					*list = append((*list)[:i], (*list)[i+1:]...)
					return
				}
			}
		})
		if removed == nil {
			return
		}
		s.deliverBuffFrame(target, s.buffFrame(false, target, *removed))
	}
}

// allFighters 全部 NPC 战斗体快照。
func (s *Server) allFighters() []*npcFighter {
	s.combatMu.Lock()
	defer s.combatMu.Unlock()
	out := make([]*npcFighter, 0, len(s.fighters))
	for _, f := range s.fighters {
		out = append(out, f)
	}
	return out
}

// damageByBuffer nx_damage_target：buff 脚本的 get_damage（5 系伤害求和）。
func (s *Server) damageByBuffer(attacker, target, buffer script.Handle) float64 {
	aux := s.auxByHandle(buffer.ID)
	if aux == nil {
		return 0
	}
	fn := script.Reentrant{VM: s.combatLua().vm}.Function(aux.script, "get_damage")
	if fn == nil {
		return 0
	}
	out, err := fn.Call(script.Ref(attacker), script.Ref(buffer), script.Ref(target))
	if err != nil || len(out) < 5 {
		return 0
	}
	raw := 0.0
	for _, v := range out[:5] {
		raw += v.Number()
	}
	return s.applyDamage(attacker, target, script.HitResult{Damage: raw},
		damageSource{skill: aux.config, class: "Buffer", byBuff: true})
}

// breakFlow nx_break_any_flow / nx_stop：打断当前施放。
func (s *Server) breakFlow(h script.Handle) {
	switch h.Kind {
	case script.KindPlayer:
		pl := s.playerByKey(h.ID)
		if pl == nil {
			return
		}
		st := &pl.combat
		st.mu.Lock()
		active := st.curSkill != ""
		st.curSkill, st.castSeq = "", st.castSeq+1
		st.mu.Unlock()
		if active {
			b := newFrameBuilder(pl)
			b.prop("CurSkillID", entergen.StringValue(""))
			s.sendFrames(pl, b.frames)
		}
	case script.KindNPC:
		f := s.fighterByHandle(h.ID)
		if f == nil {
			return
		}
		f.mu.Lock()
		active := f.cast != nil
		f.cast, f.castSeq = nil, f.castSeq+1
		f.mu.Unlock()
		if active {
			s.npcBroadcast(f, npcProperty(f, "CurSkillID", entergen.StringValue("")))
		}
	}
}
