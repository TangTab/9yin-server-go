package fssoloworld

// combatdamage.go 通用伤害落地：脚本 nx_damage、buff 周期伤害与治疗都走
// 这里（命中链的玩家→NPC 一击在 combathit.go，有完整的帧簇）。
//
// 目标是 NPC：m24[npc,22,伤害,0,0,技能|buff,"Buffer",NPC名,攻击者名]（22 =
// buff 伤害，到施法者名为止），致死走击杀簇
// m24[npc,29,攻击者,NPC名,攻击者名] → m18 → [m24 13]；目标是玩家：HP 帧 →
// m24 17。

import (
	"github.com/lizhanyin/9yin-server-go/internal/entergen"

	"github.com/lizhanyin/9yin-server-go/internal/script"
)

// applyDamage 结算一次脚本发起的伤害（amount<0 = 治疗），返回实际伤害。
func (s *Server) applyDamage(attacker, defender script.Handle, result script.HitResult, src damageSource) float64 {
	amount := int(result.Damage)
	switch defender.Kind {
	case script.KindNPC:
		f := s.fighterByHandle(defender.ID)
		if f == nil {
			return 0
		}
		_, attackerName, attackerPlayer := s.identOf(attacker)
		sink := &frameSink{s: s}
		if attackerPlayer != nil {
			sink = s.sinkFor(attackerPlayer)
		}
		f.mu.Lock()
		if f.dead {
			f.mu.Unlock()
			return 0
		}
		healed := amount < 0
		dealt, killed := 0, false
		if healed {
			f.hp = min(f.hp-amount, int(f.props["MaxHP"]))
		} else {
			dealt = min(amount, f.hp)
			f.hp -= dealt
			if killed = f.hp <= 0; killed {
				f.dead, f.deathAt = true, combatNow()
				f.cast = nil
			}
		}
		f.mu.Unlock()
		if healed || dealt <= 0 {
			return 0
		}
		sink.subject(f)
		// 到施法者名为止是 buff 伤害帧（22）；技能伤害帧（17）另有末位暴击
		// 标记——两份抓包 22 帧 22 条、17 帧 282 条，形态各自固定。
		values := []entergen.CustomValue{sink.npcValue(f), entergen.CustomInt(22), entergen.CustomInt(int32(dealt)),
			entergen.CustomInt(0), entergen.CustomInt(0), entergen.CustomString(src.skill), entergen.CustomString(src.class),
			entergen.CustomString(f.spawn.ConfigID), entergen.CustomString(attackerName)}
		if !src.byBuff {
			values[1] = entergen.CustomInt(17)
			values = append(values, entergen.CustomInt(int32(btoi(result.Crit))))
		}
		sink.pubCustom(24, values...)
		if killed {
			// 击杀簇同 hitNPC（摘 buff → m24 29 → m18 → [m24 13]，无连击帧）。
			for _, frame := range s.stripNpcBuffs(f) {
				sink.pub(frame, nil)
			}
			sink.pubCustom(24, sink.npcValue(f), entergen.CustomInt(29), s.objectValue(attacker),
				entergen.CustomString(f.spawn.ConfigID), entergen.CustomString(attackerName))
			sink.pubCustom(18, sink.npcValue(f), entergen.CustomString("hurt_half"), entergen.CustomInt(0), s.objectValue(attacker))
			if hitSpecOf(src.skill).deathMark {
				sink.pubCustom(24, sink.npcValue(f), entergen.CustomInt(13))
			}
			s.afterNpcDeath(f)
		}
		sink.flush()
		if !killed && attackerPlayer != nil {
			s.npcWake(f, attacker.ID)
		}
		return float64(dealt)
	case script.KindPlayer:
		pl := s.playerByKey(defender.ID)
		if pl == nil {
			return 0
		}
		_, attackerName, _ := s.identOf(attacker)
		st := &pl.combat
		sink := s.sinkFor(pl)
		st.mu.Lock()
		s.ensureLiveLocked(st, pl)
		dealt := 0
		if amount < 0 {
			st.live.hp = min(st.live.hp-amount, st.live.maxHP)
		} else {
			dealt = min(amount, st.live.hp-1) // 玩家死亡流程未接：保 1 点
			st.live.hp -= dealt
		}
		sink.priv("HPRatio", entergen.Int32Value(hpRatio(st.live.hp, st.live.maxHP)))
		sink.priv("HP", entergen.Int32Value(int32(st.live.hp)))
		st.mu.Unlock()
		if dealt > 0 {
			sink.custom(24, sink.playerObject(), entergen.CustomInt(17), entergen.CustomInt(int32(dealt)),
				entergen.CustomInt(0), entergen.CustomInt(0), entergen.CustomString(src.skill),
				entergen.CustomString(src.class), entergen.CustomString(pl.Name), entergen.CustomString(attackerName),
				entergen.CustomInt(int32(btoi(result.Crit))))
		}
		sink.flush()
		return float64(dealt)
	}
	return 0
}
