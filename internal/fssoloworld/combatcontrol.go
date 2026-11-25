package fssoloworld

// combatcontrol.go 技能事件包里的受控效果（抓包复原，见 hitSpecs）。
//
// 现有一例：任务技能 skill_musejuben_4「推—拉」（女号包 731~800s，命中动作
// CS_wd_tjq08，判定帧 333 / 6000 / 7934ms）。每个受击目标在各拍收到的帧：
//
//	首击（333ms）  m169 挂 buf_skill_musejuben_41（击晕 4s）→ m169 挂 …_42（减速 8s）→
//	              目标正在出招：State="hurt_stun_2" → State="" → CurSkillID="" → m189 → m24 56
//	              目标没在出招：State="hurt_stun_2"（硬直 3s 后 State=""）
//	              → m169 挂 buf_<命中动作>（命中硬直 3s）
//	第 2 拍（6000ms）m169 重加 41 → m169 摘 42 → m169 挂 42 →
//	              目标正在出招：CurSkillID="" → m189
//	              → m24 56 → m24 78[目标,攻击者,"","",N,1e+08]
//	              目标被推离到离攻击者约 5.3m 处
//	第 3 拍（7934ms）同第 2 拍，目标被拉回攻击者身前约 0.7m
//
// N 是该目标自出生以来被击退的累计次数：首击退恒为 1（女号包 737.115 同
// 一瞬间五个 NPC 朝不同方向被击退，N 全为 1；35000160 连续 8 次为 1~8；
// 350001de 中途死亡重生后归零）。
//
// 位移经 AllDest 记录广播（0x21 明文与 0x2C 压缩是同一种 44 字节记录）：
// 击退瞬间一条「移动」记录（当前位 + 运动参数），到位后一条「停止」记录。
// 运动参数只看是第几拍，与 N 无关（抓包 737.174 / 738.264）。

import (
	"math"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/script"
	"github.com/lizhanyin/9yin-server-go/internal/skilldata"
)

// hitControl 命中附带的受控。
type hitControl struct {
	buffs []string // 每击挂到受击者身上的 buff：首击全挂，其后首个重加、其余先摘再挂
	knock bool     // 首击之外每击击退（c56 → c78），推离与拉回交替
}

const (
	// hitStunMS 命中硬直时长：命中硬直 buff buf_<命中动作> 在线上的挂载
	// 时长，与 m188[目标,3000] 同值（抓包 731.440 / 736.200）。
	hitStunMS = 3000 * time.Millisecond
	// stunState 受击硬直的动画状态名（State 是字符串属性）。
	stunState = "hurt_stun_2"

	// 击退运动参数（AllDest 记录）。推离：停在离攻击者 5.1~5.6m 处，约
	// 0.85s 到位（757.405 → 758.248）；拉回：停在攻击者身前 0.2~1.6m，约
	// 0.25s 到位（738.115 → 738.264~738.481）。
	knockPushDistance = 5.3
	knockPushMS       = 850 * time.Millisecond
	knockPushSpeed    = 2.0
	knockPushMotion2  = 4.0
	knockPushMotion3  = 8.0
	knockPullDistance = 0.7
	knockPullMS       = 250 * time.Millisecond
	knockPullSpeed    = 8.0
	knockPullMotion2  = 7.8
	knockPullMotion3  = 30.4
	// knockMoveFlags / knockStopFlags 移动 / 停止记录的移动标志位。
	knockMoveFlags = 0x2602
	knockStopFlags = 0x0a
	// knockSpeedMark c78 尾值：官服恒为 1e+08，含义未定。
	knockSpeedMark = 1e+08
)

// applyHitControl 命中附带的受控（受击者存活时；帧序见文件头）。beat 为
// 本招的第几击（1 基）。
func (s *Server) applyHitControl(sink *frameSink, pl *Player, def skilldata.Definition, f *npcFighter, beat int,
	action string, spec hitControl) {
	lua := s.combatLua()
	target := npcHandle(f)
	for i, id := range spec.buffs {
		if beat > 1 && i > 0 {
			s.takeBuff(sink, target, id)
		}
		s.grantBuff(sink, target, id, 0, playerHandle(pl), lua.vm)
	}

	now := combatNow()
	f.mu.Lock()
	casting := f.cast != nil
	if casting {
		// 打断：与 breakFlow 同款状态（出招对象清空、序号推进让余击落空）。
		f.cast, f.castSeq = nil, f.castSeq+1
	}
	f.stunSeq++
	seq := f.stunSeq
	f.stunUntil = now.Add(hitStunMS)
	f.mu.Unlock()
	if beat == 1 {
		sink.npcProp(f, "State", entergen.StringValue(stunState))
		if casting {
			sink.npcProp(f, "State", entergen.StringValue(""))
			sink.npcProp(f, "CurSkillID", entergen.StringValue(""))
			sink.pubCustom(189, sink.npcValue(f))
			sink.pubCustom(24, sink.npcValue(f), entergen.CustomInt(56))
		}
	} else if casting {
		sink.npcProp(f, "CurSkillID", entergen.StringValue(""))
		sink.pubCustom(189, sink.npcValue(f))
	}
	if beat == 1 && action != "" && lua.skills != nil && lua.skills.BuffExists("buf_"+action) {
		s.grantBuff(sink, target, "buf_"+action, hitStunMS, playerHandle(pl), lua.vm)
	}
	// 硬直接束：没在出招的目标回 State=""；被打断的起手不会再走它自己的
	// 收招回调，决策循环从这里接上。
	if beat == 1 || casting {
		s.after(hitStunMS, func() { s.npcStunEnd(f, seq, beat == 1 && !casting, casting) })
	}

	if spec.knock && beat >= 2 {
		s.knockback(sink, f, pl.roleState.Position, (beat%2 == 0))
	}
}

// npcStunEnd 硬直接束：清状态、把 NPC 放回决策循环。序号不符 = 期间又被
// 硬直，交给后一次。
func (s *Server) npcStunEnd(f *npcFighter, seq int, clearState, resume bool) {
	f.mu.Lock()
	skip := f.dead || f.stunSeq != seq || f.cast != nil
	f.mu.Unlock()
	if skip {
		return
	}
	if clearState {
		s.npcBroadcast(f, npcProperty(f, "State", entergen.StringValue("")))
	}
	if resume {
		s.npcReact(f)
	}
}

// knockback 击退一击：c56 → c78 → 移动记录；到位后落位并发停止记录。
// push = 推离攻击者，否则拉回身前。
func (s *Server) knockback(sink *frameSink, f *npcFighter, attacker [4]float64, push bool) {
	f.mu.Lock()
	f.knockbacks++
	count := f.knockbacks
	from := f.pos
	f.mu.Unlock()
	dx, dz := from[0]-attacker[0], from[2]-attacker[2]
	if dist := math.Hypot(dx, dz); dist > 0.05 {
		dx, dz = dx/dist, dz/dist
	} else {
		// 贴身无法定向：沿攻击者朝向。
		dx, dz = math.Sin(attacker[3]), math.Cos(attacker[3])
	}
	distance, duration, speed, motion2, motion3 := knockPullDistance, knockPullMS, knockPullSpeed, knockPullMotion2, knockPullMotion3
	if push {
		distance, duration, speed, motion2, motion3 = knockPushDistance, knockPushMS, knockPushSpeed, knockPushMotion2, knockPushMotion3
	}
	to := from
	to[0], to[2] = attacker[0]+dx*distance, attacker[2]+dz*distance
	if push {
		// 推离后朝向 = 运动方向（抓包 738.264 停止记录 o=3.888）；拉回不变。
		to[3] = math.Atan2(dx, dz)
	}
	sink.pubCustom(24, sink.npcValue(f), entergen.CustomInt(56))
	sink.pubCustom(24, sink.npcValue(f), entergen.CustomInt(78), sink.playerObject(),
		entergen.CustomString(""), entergen.CustomString(""), entergen.CustomInt(int32(count)),
		entergen.CustomFloat32(knockSpeedMark))
	sink.pub(entergen.ServerAllDestFrame([]entergen.AllDestEntry{
		allDestRecord(f, from, float32(speed), float32(-2*math.Pi), float32(motion2), float32(motion3), knockMoveFlags)}), nil)
	s.after(duration, func() { s.knockbackSettle(f, to, float32(motion2), float32(motion3)) })
}

// knockbackSettle 击退到位：服务端位置落到终点，发停止记录。
func (s *Server) knockbackSettle(f *npcFighter, to [4]float64, motion2, motion3 float32) {
	f.mu.Lock()
	if f.dead {
		f.mu.Unlock()
		return
	}
	f.pos = to
	f.mu.Unlock()
	s.npcBroadcast(f, entergen.ServerAllDestFrame([]entergen.AllDestEntry{
		allDestRecord(f, to, 0, 0, motion2, motion3, knockStopFlags)}))
}

// allDestRecord AllDest 记录（entergen.ServerAllDestFrame 的单条）。
func allDestRecord(f *npcFighter, at [4]float64, speed, rotate, motion2, motion3 float32, flags uint32) entergen.AllDestEntry {
	return entergen.AllDestEntry{ObjectID: f.spawn.ObjectID, OwnerID: f.spawn.OwnerID,
		DestX: float32(at[0]), DestY: float32(at[1]), DestZ: float32(at[2]), DestOrient: float32(at[3]),
		Speed: speed, Rotate: rotate, Motion2: motion2, Motion3: motion3, MoveFlags: flags}
}

// grantBuff 给目标挂 buff 并把 m169 挂载帧入汇（life ≤ 0 取定义里的生命期）。
func (s *Server) grantBuff(sink *frameSink, target script.Handle, id string, life time.Duration, caster script.Handle, lua script.Caller) {
	skills := s.combatLua().skills
	if skills == nil {
		return
	}
	def, err := skills.Buff(id, 1)
	if err != nil {
		return
	}
	if life <= 0 {
		life = time.Duration(def.LifeTimeMS) * time.Millisecond
	}
	buff := combatBuff{id: def.ID, level: 1, category: def.Category, lua: def.LuaScript, caster: caster}
	_, frame := s.addBuff(target, buff, life, def.EffectType, def.EffectTimerMS, nil, lua)
	sink.pub(frame, nil)
}

// takeBuff 从目标身上摘下指定 ConfigID 的 buff，摘帧入汇（不发）。
func (s *Server) takeBuff(sink *frameSink, target script.Handle, id string) {
	var removed *combatBuff
	s.withBuffs(target, func(list *[]combatBuff) {
		for i := range *list {
			if (*list)[i].id == id {
				taken := (*list)[i]
				removed = &taken
				*list = append((*list)[:i], (*list)[i+1:]...)
				return
			}
		}
	})
	if removed != nil {
		sink.pub(s.buffFrame(false, target, *removed), nil)
	}
}
