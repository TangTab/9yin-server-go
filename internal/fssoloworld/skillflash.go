package fssoloworld

// skillflash.go 闪击通道技（客户端 skill_static IsFlashAttack=1；现有实证技能
// = 寒心恨雪 CS_jh_hyjf08）：受理后由服务端逐波驱动的「闪到目标身上打一下」
// 引导。帧方言取自 挑战.pcapng（外服 103.143.19.173:7071，10 次施放 34 跳
// 逐帧解码），c71 的客户端语义取自 fxgamelogic.dll 反汇编；定版与取证见
// docs/combat/寒心恨雪与太极剑舞引导机制.md。
//
// 一次施放：
//
//	受理   通用受理簇（CurSkillID → CurSkillTarget → 锁态链 → 冷却行 → 0x1F
//	       → m337 → m20[自己,招式首段动作,目标]），怒气不预扣
//	波 1   命中结算 → 自身 buff 重挂链 → SP 扣 → m18[目标,受击动作]
//	       → m169 挂目标 buff → c71 → m205 → CurSkillID="" → CurSkillTarget=None
//	波 2+  m169 摘上一波目标 buff → 自身 buff 重挂链 → SP 扣 → m18
//	       → m169 挂目标 buff → c71 → 命中结算（m205 只在波 1 发一次）
//	收尾拍 m169 摘末波目标 buff（波数耗尽 / 怒气不足 / 无活目标，这一拍只做
//	       这件事；官服实测在末波后 423~502ms）
//	收束   末波 + 1s：m169 摘自身 buff → BufferInfoN="" → Cant* 五位清零
//
// c71 = m24 子码 71（客户端 player_flash_attack，玩家状态机状态 123）：客户端
// 把速度与动作名设到施法者身上，以 tm_speed（30 m/s）冲到目标实位，到达后
// 回 C2S m914 上报落点。怒气逐波扣（先扣后结算；被闪避的波照扣）；怒气不够
// 扣一次的施放整条静默吞。每波恰好打一个目标：首波 = m211 锁定目标，其后在
// 玩家最新上报位的射程内随机换靶（同一目标可以连中）。
//
// 锁序：玩家 combatState.mu → NPC npcFighter.mu；调 hitNPC（内部自取
// combatState.mu）前一律放锁。

import (
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/script"
	"github.com/lizhanyin/9yin-server-go/internal/skilldata"
)

const (
	// clientCustomZhaoshiEnd C2S m914 招式位移结束位置回报（客户端
	// CLIENT_CUSTOMMSG_ASY_POS_IN_ZHAOSHI_END）：[914, x, y, z, 朝向, 类型, 计数]。
	clientCustomZhaoshiEnd int32 = 914
	// zhaoshiEndSkillTranslate m914 类型 12：技能位移（闪击 / 冲锋）到达。计数位
	// 是角色当时的 skill_lock_count（卷入的技能锁定关系数），服务端只读不用。
	zhaoshiEndSkillTranslate = 12
	// channelResetDelay 末波到收束的间隔（挑战包：+1013 / +1018ms）。
	channelResetDelay = 1010 * time.Millisecond
	// m914 回显（0x21 成对停帧）的回显槽与标志位：首条朝向折到 [0, 2π)、
	// 标志 10；次条朝向原值、标志 1536。
	flashEchoMotion2        = 5.5
	flashEchoMotion3        = 9.8
	flashEchoFlagNormalized = 10
	flashEchoFlagRaw        = 1536
)

// channelSpec 闪击通道技参数（挑战包实证值）。
type channelSpec struct {
	waves      int           // 波数上限
	firstWave  time.Duration // 受理 → 首波（客户端侧观测量，含往返时延）
	interval   time.Duration // 波间隔（中位 438ms，范围 374~511）
	spPerWave  int           // 每波扣怒
	speed      float32       // c71 的 tm_speed（m/s）：实测时延 ∝ 跳距，拟合 31.7 m/s
	victimBuff string        // 每波挂到目标身上、下一拍摘的 buff
	selfBuff   string        // 引导期自身锁态 buff（每波重挂，收束时摘）
	hurtAction string        // 每波 m18 的受击动作（两值形态）
}

// channelSpecs 抓包复原的闪击通道技。CS_jh_tqjf08 的招式段与寒心恨雪逐位
// 同构（IsFlashAttack 同族），但它的逐波动作族（wj_qzj_07_N）在客户端动作表
// 里不存在、也没有抓包，未列入——取证后在此加一行即可接入。
var channelSpecs = map[string]channelSpec{
	"CS_jh_hyjf08": {waves: 8, firstWave: 180 * time.Millisecond, interval: 440 * time.Millisecond,
		spPerWave: 10, speed: 30, victimBuff: "buf_CS_jh_hyjf08", selfBuff: "buf_CS_jh_hyjf08_03",
		hurtAction: "hurt_f_1_l_half"},
}

// channelPick 后续波在候选目标里随机取一个的下标源（测试可替换）。
var channelPick = rand.IntN

// channelCantProps 锁态链置位的四个字节位。
var channelCantProps = []string{"CantBeSkillLocked", "CantBeSkillMustLocked", "CantBeAttack", "CantMove"}

// channelState 进行中的闪击通道（combatState.mu 护）。
type channelState struct {
	def    skilldata.Definition
	spec   channelSpec
	seq    int         // 施放序号（被打断 / 换招时对账）
	start  time.Time   // 受理时刻（各拍与收束的名义时刻都从它起算）
	scene  string      // 受理时的场景（切图即收束）
	action string      // m20 下发的首段动作名；逐波动作 = 基名_波号
	target uint64      // m211 锁定目标（首波）
	skill  uint64      // 技能脚本对象句柄
	wave   int         // 已发出的波数
	done   bool        // 收尾拍已发（不再出波；等收束定时器摘自身 buff）
	at     *npcFighter // 上一跳落到的目标（后续波 c71 的起点对象；nil = 还没跳）
	prev   *npcFighter // 上一波挂了目标 buff 的对象（下一拍摘）
}

// ---- 帧 ----

// flashAttackValues m24 子码 71（客户端 player_flash_attack）的值表（不含消息
// 号）。线路形态两种（挑战包逐字节）：
//
//	首波   [目标, 71, 目标, "", 1, 施法者, 速度, 0, "", 0, ""]
//	后续波 [上一目标, 71, 目标, "", 1, 施法者, 速度, 0, "<动作基名>_<波号>"]
//
// 客户端处理函数拿到的是去掉 [24, 对象, 子码] 后的子表，下标固定：[0] 目标
// 对象、[2] 必须为 1、[3] 施法者、[4] tm_speed、[5] tm_add、[6] tm_action（施法
// 者的位移动作）；值数 > 8 才读 [7] tm_back_dist 与 [8] tm_behit_action，> 9 才
// 读 [9] tm_stop_time（缺省 1.0 秒）。所以首波的空串占的是 tm_action，其后
// 两个是回撤距离与受击动作；后续波的动作名紧跟 tm_add——多塞一个空串它就
// 落到 tm_back_dist 槽里，客户端不会播。
func flashAttackValues(from, to uint64, caster wireIdent, speed float32, action string) []entergen.CustomValue {
	caster = caster.orFixed()
	values := []entergen.CustomValue{targetValue(from), entergen.CustomInt(71), targetValue(to),
		entergen.CustomString(""), entergen.CustomInt(1),
		entergen.CustomObject(caster.objectID, caster.ownerID),
		entergen.CustomFloat32(speed), entergen.CustomFloat32(0)}
	if action == "" {
		return append(values, entergen.CustomString(""), entergen.CustomFloat32(0), entergen.CustomString(""))
	}
	return append(values, entergen.CustomString(action))
}

// flashWaveAction 第 wave 波（1 基）的位移动作名：首波空（动画由受理 m20 的
// 首段动作驱动），其后 = 首段动作去段尾 "_1" 再接波号（qzj_07_1 → qzj_07_2 …）。
// 首段动作不以 "_1" 结尾的异构名：全波同名。
func flashWaveAction(first string, wave int) string {
	if wave <= 1 {
		return ""
	}
	base, ok := strings.CutSuffix(first, "_1")
	if !ok {
		return first
	}
	return base + "_" + strconv.Itoa(wave)
}

// channelCantFrames 锁态链的五个属性位（帧序即官服顺序）。CantParry 是「不可
// 招架来源数」：通道自身只占 1，与另四位同值——挑战包 6 次施放里 4 次读数是
// 置 1 / 清 0；另 2 次（249.640 / 386.312）读到 2 / 1，是因为玩家身上另有怪物
// 挂的 buf_CantParry（时间线里有它的摘除帧，摘后回落到 1），本服务端没有这个
// 来源。
func channelCantFrames(b *frameBuilder, value byte) {
	b.prop("CantBeSkillLocked", entergen.ByteValue(value))
	b.prop("CantParry", entergen.Int32Value(int32(value)))
	b.prop("CantBeSkillMustLocked", entergen.ByteValue(value))
	b.prop("CantBeAttack", entergen.ByteValue(value))
	b.prop("CantMove", entergen.ByteValue(value))
}

// timedBuff 摘 / 挂帧用的 buff 副本：时间值取此刻（官服闪击两处时间值都是
// 发帧时刻，不是到期时刻）。
func timedBuff(buff combatBuff) combatBuff {
	buff.expireAt = combatNow()
	return buff
}

// victimBuff 每波挂到目标身上的瞬挂 buff（不带类别与钩子）。
func victimBuff(spec channelSpec, pl *Player) combatBuff {
	return combatBuff{id: spec.victimBuff, caster: playerHandle(pl), expireAt: combatNow()}
}

func fighterKey(f *npcFighter) uint64 {
	if f == nil {
		return 0
	}
	return uint64(f.spawn.ObjectID) | uint64(f.spawn.OwnerID)<<32
}

func fighterHP(f *npcFighter) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.hp
}

// ---- 受理与收束 ----

// channelAction 闪击通道技的 m20 动作名：招式表该招的首段动作（寒心恨雪 =
// qzj_07_1，与技能 id 不同名）。招式节拍产物缺席或查不到时回落技能 id
// （与常规技一致），此时逐波不带动作名。
func (s *Server) channelAction(def skilldata.Definition) string {
	lua := s.combatLua()
	if lua.actions != nil {
		if schedule, ok := lua.actions.Lookup(def.ID); ok && len(schedule.Actions) > 0 && schedule.Actions[0] != "" {
			return schedule.Actions[0]
		}
	}
	return def.ID
}

// grantChannelLockLocked 受理时的锁态链（调用方持 combatState.mu），帧入帧簇。
func (s *Server) grantChannelLockLocked(b *frameBuilder, st *combatState, pl *Player, spec channelSpec) {
	b.frames = append(b.frames, s.channelChainLocked(st, pl, spec, true)...)
}

// channelChainLocked 自身锁态 buff 的挂载 / 重挂链（调用方持 combatState.mu）。
// 首次挂载：Cant* 置位 → BufferInfoN 槽串 → m169 挂；每波重挂：m169 摘 → 槽串
// 清空 → Cant* 清除档 → Cant* 置位 → 槽串 → m169 挂。帧里的时间值两处都是
// 「此刻」（挑战包 m169 的 i64 与槽串第 5 段同值，不是到期时刻）。槽位 = 首个
// 空槽（挑战包 BufferInfo4 / BufferInfo5 两例，随当时在身 buff 数变）。buff
// 目录里没有这个 buff 时只发 Cant* 位。
func (s *Server) channelChainLocked(st *combatState, pl *Player, spec channelSpec, apply bool) [][]byte {
	s.ensureLiveLocked(st, pl)
	b := newFrameBuilder(pl)
	// 只动本通道自己的 buff：在身的不是它（理论不会发生）就不碰。
	active := st.selfBuff
	if active != nil && active.buff.id != spec.selfBuff {
		active = nil
	} else {
		st.selfBuff = nil
	}
	if active != nil && !apply {
		b.frames = append(b.frames, s.buffFrame(false, playerHandle(pl), timedBuff(active.buff)))
		b.frames = append(b.frames, s.buffSlotFrame(pl, active.slot, combatBuff{}, 0))
	}
	if apply {
		channelCantFrames(b, 1)
	} else {
		channelCantFrames(b, 0)
		channelCantFrames(b, 1)
	}
	buff, staticData, ok := s.channelBuffDef(spec.selfBuff)
	if !ok {
		return b.frames
	}
	slot := bufferSlotBase + len(st.live.buffs)
	if active != nil {
		slot = active.slot
	} else {
		for i := range st.live.buffs {
			if st.live.buffs[i].id == buff.ID {
				slot = bufferSlotBase + i
				break
			}
		}
	}
	live := timedBuff(combatBuff{id: buff.ID, level: 1, category: buff.Category, lua: buff.LuaScript,
		caster: playerHandle(pl)})
	st.live.buffs = upsertBuff(st.live.buffs, live)
	st.selfBuff = &activeSelfBuff{buff: live, slot: slot, props: channelCantProps}
	b.frames = append(b.frames, s.buffSlotFrame(pl, slot, live, staticData))
	b.frames = append(b.frames, s.buffFrame(true, playerHandle(pl), live))
	return b.frames
}

// channelBuffDef 锁态 / 目标 buff 的定义（buff 目录缺席时 ok=false）。
func (s *Server) channelBuffDef(id string) (skilldata.BuffDefinition, int, bool) {
	lua := s.combatLua()
	if lua.skills == nil {
		return skilldata.BuffDefinition{}, 0, false
	}
	buff, err := lua.skills.Buff(id, 1)
	if err != nil {
		return skilldata.BuffDefinition{}, 0, false
	}
	return buff, buff.StaticData, true
}

// closeChannelLocked 通道收束（调用方持 combatState.mu）：m169 摘自身 buff →
// 槽串清空 → Cant* 五位清零（CantParry 归 0）。只动本通道自己的状态——通道
// 被打断后玩家可能已开始别的施放，那一路的 CurSkillID / 自身 buff 不碰。
func (s *Server) closeChannelLocked(pl *Player, st *combatState, state *channelState) [][]byte {
	st.channel = nil
	if st.curSkill == state.def.ID {
		st.curSkill, st.locked = "", false
	}
	b := newFrameBuilder(pl)
	if active := st.selfBuff; active != nil && active.buff.id == state.spec.selfBuff {
		st.selfBuff = nil
		buffs := &st.live.buffs
		for i := range *buffs {
			if (*buffs)[i].id == active.buff.id {
				*buffs = append((*buffs)[:i], (*buffs)[i+1:]...)
				break
			}
		}
		b.frames = append(b.frames, s.buffFrame(false, playerHandle(pl), timedBuff(active.buff)))
		b.frames = append(b.frames, s.buffSlotFrame(pl, active.slot, combatBuff{}, 0))
	}
	channelCantFrames(b, 0)
	return b.frames
}

// ---- 波次 ----

// launchChannel 闪击通道技登记（受理帧已落地后调用）：各拍按受理时刻的绝对
// 偏移排在战斗时钟上（逐拍相对排会累积泵的节拍抖动）。index 0..waves-1 是
// 波，waves 是只摘 buff 的收尾拍。
func (s *Server) launchChannel(pl *Player, def skilldata.Definition, cast skillCast, action string, seq int) {
	st := &pl.combat
	st.mu.Lock()
	s.ensureLiveLocked(st, pl)
	if aux := s.auxByHandle(st.skillObj); aux == nil || aux.config != def.ID {
		st.skillObj = s.registerAux(&auxObject{kind: script.HelperSkill, config: def.ID, script: def.LuaScript,
			def: def, owner: playerHandle(pl), props: map[string]script.Value{}, records: map[string][][]script.Value{}}).ID
	}
	st.live.hits = 0
	skillObj := st.skillObj
	st.mu.Unlock()
	pl.npcMu.Lock()
	scene := pl.npcScene
	pl.npcMu.Unlock()
	state := &channelState{def: def, spec: channelSpecs[def.ID], seq: seq, start: combatNow(), scene: scene,
		action: action, target: cast.targetKey(), skill: skillObj}
	st.mu.Lock()
	st.channel = state
	st.mu.Unlock()
	for k := 0; k <= state.spec.waves; k++ {
		s.after(state.spec.firstWave+time.Duration(k)*state.spec.interval, func() { s.channelTick(pl, state, k) })
	}
	s.trace("skill %s channel launched action=%s target=%d client=%d", def.ID, action, cast.targetKey(), pl.Conn1)
}

// channelVictim 第 index 波（0 基）的目标：首波 = m211 锁定目标（已死或已
// 失效则回落随机索敌）；其后在玩家最新上报位的射程内的活目标里随机。
func (s *Server) channelVictim(pl *Player, state *channelState, index int) *npcFighter {
	pl.npcMu.Lock()
	scene := pl.npcScene
	pl.npcMu.Unlock()
	if scene == "" {
		return nil
	}
	if index == 0 && state.target != 0 {
		if f := s.fighterBySpawn(scene, state.target); f != nil && !f.isDead() {
			return f
		}
	}
	origin := pl.roleState.Position
	var pool []*npcFighter
	for _, spawn := range s.skillTargetSpawns(pl, scene) {
		if !attackable(spawn) {
			continue
		}
		f := s.fighterFor(scene, spawn)
		f.mu.Lock()
		pos, dead := f.pos, f.dead
		f.mu.Unlock()
		if dead || math.Hypot(pos[0]-origin[0], pos[2]-origin[2]) > state.def.CastRange {
			continue
		}
		pool = append(pool, f)
	}
	if len(pool) == 0 {
		return nil
	}
	return pool[channelPick(len(pool))]
}

// channelTick 第 index 拍（0 基；index == spec.waves 是收尾拍）。通道已被收束、
// 或此前已到收尾拍时直接返回。
func (s *Server) channelTick(pl *Player, state *channelState, index int) {
	st := &pl.combat
	st.mu.Lock()
	if st.channel != state || state.done {
		st.mu.Unlock()
		return
	}
	online := pl.State == PlOnline
	pl.npcMu.Lock()
	sameScene := pl.npcScene == state.scene
	pl.npcMu.Unlock()
	if st.castSeq != state.seq || st.curSkill != state.def.ID || !sameScene {
		// 打断 / 换招 / 切图：立即收束。离线或已切图时旧场景的帧没有接收方。
		wave := state.wave
		frames := s.closeChannelLocked(pl, st, state)
		st.mu.Unlock()
		if online && sameScene {
			s.sendFrames(pl, frames)
		}
		s.trace("skill %s channel closed early wave=%d client=%d", state.def.ID, wave, pl.Conn1)
		return
	}
	sink := s.sinkFor(pl)
	if state.prev != nil {
		// 摘上一波目标的 buff：换没换目标都摘（挑战包每一拍都有，被击杀的
		// 上一波目标也补摘）。
		sink.pub(s.buffFrame(false, npcHandle(state.prev), timedBuff(combatBuff{id: state.spec.victimBuff,
			caster: playerHandle(pl)})), nil)
		sink.subject(state.prev)
		state.prev = nil
	}
	s.ensureLiveLocked(st, pl)
	ended := ""
	if index >= state.spec.waves {
		ended = "waves exhausted"
	} else if st.live.sp < state.spec.spPerWave {
		ended = "sp ran out"
	}
	sp := st.live.sp
	st.mu.Unlock()

	if ended == "" {
		if victim := s.channelVictim(pl, state, index); victim != nil {
			s.channelWave(pl, state, index, victim, sink)
			return
		}
		ended = "no target"
	}
	// 收尾：这一拍只摘 buff（已入 sink）。怒气不足时不补扣、不发帧。
	sink.flush()
	st.mu.Lock()
	state.done = true
	waves := state.wave
	var frames [][]byte
	if waves == 0 {
		// 一招都没打出去：没有排过收束定时器，当场收束。
		frames = s.closeChannelLocked(pl, st, state)
	}
	st.mu.Unlock()
	if len(frames) > 0 {
		s.sendFrames(pl, frames)
	}
	s.trace("skill %s channel done (%s) waves=%d sp=%d client=%d", state.def.ID, ended, waves, sp, pl.Conn1)
}

// channelWave 一波（第 index + 1 波；调用方持 combatState.mu 的两段之间放锁调
// hitNPC）。帧序见文件头。
func (s *Server) channelWave(pl *Player, state *channelState, index int, victim *npcFighter, sink *frameSink) {
	st := &pl.combat
	st.mu.Lock()
	if st.channel != state || state.done {
		st.mu.Unlock()
		sink.flush()
		return
	}
	spec, first := state.spec, index == 0
	hpBefore, skillObj := fighterHP(victim), state.skill
	// c71 的起点 = 上一跳落到的目标；还没跳过（首波，或首波被闪避）就用目标
	// 自身——即官服首波形态。
	at := state.at
	if at == nil {
		at = victim
	}
	// 扣怒的先后随波别：波 2 起先扣后结算（SP 帧 = 扣后值，得怒帧在其后）；
	// 波 1 先结算再扣（SP 帧序 = 得怒 +2 在前、扣 10 在后：挑战包 97 → 99 →
	// 89）。被闪避的波照扣。
	if !first {
		st.live.sp -= spec.spPerWave
	}
	landed := true
	if first {
		// 未命中（被闪避）时只留交手帧与闪避帧。
		st.mu.Unlock()
		landed = s.hitNPC(pl, state.def, victim, 1, 1, skillObj, false, index+1, state.action, sink)
		st.mu.Lock()
		if st.channel != state {
			st.mu.Unlock()
			sink.flush()
			return
		}
		st.live.sp -= spec.spPerWave
	}
	// 自身 buff 每波都是重挂链（受理时已挂过）。
	for _, frame := range s.channelChainLocked(st, pl, spec, false) {
		sink.pri(frame, nil)
	}
	sink.priv("SP", entergen.Int32Value(int32(st.live.sp)))
	st.mu.Unlock()

	// 受击动作 → 挂目标 buff → c71。波 2 起这三帧先于结算帧；结算未命中时撤回。
	mark := len(sink.frames)
	if landed {
		sink.pubCustom(18, sink.npcValue(victim), entergen.CustomString(spec.hurtAction))
		sink.pub(s.buffFrame(true, npcHandle(victim), victimBuff(spec, pl)), nil)
		sink.pubCustom(24, flashAttackValues(fighterKey(at), fighterKey(victim), sink.ident, spec.speed,
			flashWaveAction(state.action, index+1))...)
		sink.subject(victim)
	}
	settled := len(sink.frames)
	if !first {
		landed = s.hitNPC(pl, state.def, victim, 1, 1, skillObj, false, index+1, state.action, sink)
		if !landed {
			sink.frames = append(sink.frames[:mark], sink.frames[settled:]...)
		}
	}
	if landed && first {
		// m205 一次施放只发一次（挑战包 6 次施放逐帧：都在波 1 的 c71 之后）。
		sink.pubCustom(205, sink.playerObject(), entergen.CustomString(state.def.ID), sink.npcValue(victim))
	}
	st.mu.Lock()
	open := st.channel == state
	if open {
		state.wave = index + 1
		if landed {
			state.prev, state.at = victim, victim
		}
	}
	if first {
		// 受理设置 → 波 1 清空的瞬时对：客户端通道演出的触发链。只改线上的
		// CurSkillTarget；服务端的出招态留到收束（期间再按技能回 9033）。
		sink.priv("CurSkillID", entergen.StringValue(""))
		sink.priv("CurSkillTarget", entergen.ObjectValue(0, 0))
		st.curTarget = 0
	}
	st.mu.Unlock()
	sink.flush()
	s.trace("skill %s channel wave %d/%d victim=%s damage=%d client=%d", state.def.ID, index+1, spec.waves,
		victim.spawn.ConfigID, hpBefore-fighterHP(victim), pl.Conn1)
	if open {
		// 收束按名义波时刻 + 1s 排：这一拍实际触发得晚（泵的节拍抖动）不推迟收束。
		due := state.start.Add(spec.firstWave + time.Duration(index)*spec.interval + channelResetDelay)
		s.after(due.Sub(combatNow()), func() { s.channelCleanup(pl, state, index+1) })
	}
}

// channelCleanup 末波 + channelResetDelay 的收束定时器：只有最后一波的这一只
// 生效（更晚的波已另排），此时才摘自身 buff 并放开行动锁。
func (s *Server) channelCleanup(pl *Player, state *channelState, wave int) {
	st := &pl.combat
	st.mu.Lock()
	if st.channel != state || state.wave != wave {
		st.mu.Unlock()
		return
	}
	frames := s.closeChannelLocked(pl, st, state)
	st.mu.Unlock()
	if pl.State == PlOnline {
		s.sendFrames(pl, frames)
	}
	s.trace("skill %s channel closed waves=%d client=%d", state.def.ID, wave, pl.Conn1)
}

// handleZhaoshiEnd C2S m914 招式位移结束位置回报 [914, x, y, z, 朝向, 类型,
// 计数]（挑战包寒心恨雪 34 跳逐拍；技能包里每次冲锋 / 轻功收招后各一报）。
// 位置收编为服务端权威位（absorbPlayerPosition）；类型 12（技能位移到达）另
// 回 0x21 成对停帧——官服 39/39 跳都有回显，首条在 44~321ms 内到，首条朝向
// 折到 [0, 2π)、次条朝向原值，速度恒 0（闪击没有移动过程帧）。其余类型只收编不回显。
func (s *Server) handleZhaoshiEnd(pl *Player, message entergen.CustomInbound) {
	values := message.Values
	if len(values) < 7 || values[0].Type != 2 || values[0].Int32 != clientCustomZhaoshiEnd {
		return
	}
	var pos [4]float64
	for i := range pos {
		value, ok := finiteFloat(values[1+i])
		if !ok {
			s.trace("zhaoshi end malformed client=%d", pl.Conn1)
			return
		}
		pos[i] = float64(value)
	}
	if !validPlayerPosition(pos) {
		return
	}
	s.absorbPlayerPosition(pl, pos)
	kind, _ := values[5].ExactInt32()
	if kind != zhaoshiEndSkillTranslate || pl.State != PlOnline {
		return
	}
	ident := pl.wireIdent().orFixed()
	echo := entergen.ServerAllDestFrame([]entergen.AllDestEntry{
		{ObjectID: ident.objectID, OwnerID: ident.ownerID,
			DestX: float32(pos[0]), DestY: float32(pos[1]), DestZ: float32(pos[2]),
			DestOrient: normalizeOrient(float32(pos[3])), Motion2: flashEchoMotion2, Motion3: flashEchoMotion3,
			MoveFlags: flashEchoFlagNormalized},
		{ObjectID: ident.objectID, OwnerID: ident.ownerID,
			DestX: float32(pos[0]), DestY: float32(pos[1]), DestZ: float32(pos[2]),
			DestOrient: float32(pos[3]), Motion2: flashEchoMotion2, Motion3: flashEchoMotion3,
			MoveFlags: flashEchoFlagRaw},
	})
	if echo != nil {
		s.sendFrames(pl, [][]byte{echo})
	}
}
