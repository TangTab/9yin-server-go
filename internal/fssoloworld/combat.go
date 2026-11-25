package fssoloworld

// combat.go 战斗接入层：战斗体状态、nx_* 对象视图、事件队列与 Lua World 适配。
//
// 官服帧方言取自 E:\Game\9yin\9yin-server-go2\资料\ 的抓包逐帧解码
// （tools/pcapevents dumprec）：女号完整引导包 195~232s / 345~400s、
// 挑战.pcapng 447~484s、技能4.pcapng 268s。要点：
//
//	战斗关系  NPC 仇恨/被击 → NPC.LogicState=1 → 玩家 fight_relation_rec
//	          （rec114，行=[对手, word]）加行 → 玩家.LogicState=1 →
//	          m24[npc,37,"npc_foundplayer"]；每次命中刷新该行（0x13）；
//	          对手死亡/脱战删行（0x12）。
//	命中      技能行 CurTargetCount → 攻击者 FP/SP → 关系行刷新 →
//	          m24[目标,17,伤害,0,0,技能,逻辑类,目标名,攻击者名,暴击] →
//	          [第 2 击起 m24[攻击者,10,累计击数] + MaxHits] →
//	          [非普攻 m24[目标,40,"body_hit_3_self",攻击者]] → m205[攻击者,技能,目标…]。
//	击杀      m24[目标,17…] → 关系行删 → m24[目标,29,攻击者,目标名,攻击者名]
//	          → m18[目标,"hurt_half",0,攻击者] → m24[目标,13] → m205。
//	闪避      m24[目标,7,1,攻击者,技能,目标名,攻击者名,100]（锁定技另配
//	          m20[目标,"<技能>_dodge",攻击者]）。
//	招架      m24[目标,8,1,攻击者,技能,目标名,攻击者名] → 招架耐力 −5 →
//	          m24[目标,17,穿透后伤害…]。
//	buff      m169[0 挂/1 摘,buffID,目标,施法者,目标名,施法者名,到期毫秒,Category]。
//
// NPC 的 HP 不以属性帧下发（客户端凭 m24 17 自行扣减）；HP/HPRatio 帧
// 只发给受击玩家本人。
//
// 每击结算与技能钩子走 Lua 引擎（internal/script）：公式 → 未命中 → 暴击
// → 招架；引擎不可用时回退 Go 面板×倍率。NINEYIN_COMBAT=off 关停全部
// 结算（受理链照旧）。

import (
	"container/heap"
	"os"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/skilldata"

	"github.com/lizhanyin/9yin-server-go/internal/script"
)

// 逻辑态（share/logicstate_define.lua）。
const (
	logicNormal   = 0 // LS_NORMAL
	logicFighting = 1 // LS_FIGHTING
	logicWarning  = 2 // LS_WARNING（脱战过渡）
)

// 战斗节拍常量（抓包观测；标注推断处无直接帧证）。时长一律是
// time.Duration 类型——与裸毫秒/纳秒混用即编译报错。
const (
	// hitSPGain 命中一次攻击者得怒/战意（FP/SP 成对帧，恒 +2）。
	hitSPGain = 2
	// hitTakenSPGain 被击未招架 +1、招架 +4（抓包 225.883：19 → 23）。
	hitTakenSPGain   = 1
	parryTakenSPGain = 4
	parryValuePerHit = 5 // 招架一次掉 5 点耐力（72 → 67）
	// fightRelationLeadMS fight_relation_rec 第 2 列 = (now + 此值) 的低 16 位
	// （女号包 49 个样本对钟后中位 36862ms，抖动 ±50ms）。
	fightRelationLeadMS = 36860
	// playerLeaveFight / playerCalmDelay 玩家脱战：最后一次战斗活动 3s 后
	// LogicState=2、FP/SP 清零（抓包 228.138 → 231.152/231.303），再 2s 回 0。
	playerLeaveFight = 3 * time.Second
	playerCalmDelay  = 5 * time.Second
	// npcCalmDelay NPC 脱战（抓包 228.138 → 230.166）。
	npcCalmDelay = 2 * time.Second
	// corpseKeep / npcRespawnDelay 尸体停留与重生（推断：抓包未覆盖）。
	corpseKeep      = 5 * time.Second
	npcRespawnDelay = 15 * time.Second
	// npcReactDelay NPC 被击/发现目标后的反应延迟（推断：抓包 215.577 发现
	// → 217.043 首次起手，其间含接近时间）。
	npcReactDelay = 600 * time.Millisecond
	// npcAttackRange / npcAttackPeriod NPC 近身攻击距离与起手周期
	// （2433 审计 A6：周期 2200ms、近身 6m；按 3m 贴身取更贴近抓包）。
	npcAttackRange  = 3.0
	npcAttackPeriod = 2200 * time.Millisecond
	// npcChaseStep 追击/回归位置推进节拍。
	npcChaseStep = 500 * time.Millisecond
	// parryFactor 招架穿透比（引擎回退面；Lua 引擎内同值）。
	parryFactor = 0.10
	// combatPumpStep 战斗事件泵节拍。
	combatPumpStep = 50 * time.Millisecond
	// finishMargin 末击之后到收招的余量：盖过泵的节拍抖动，再加一帧
	// （抓包：CurSkillID 在末击帧之后才清）。
	finishMargin = combatPumpStep + 40*time.Millisecond
)

// combatEnabled NINEYIN_COMBAT=off 关停战斗结算。
func combatEnabled() bool { return os.Getenv("NINEYIN_COMBAT") != "off" }

// npcAIEnabled NINEYIN_NPC_AI=off 关停 NPC 战斗 AI（仇恨/追击/起手/脱战；
// 玩家侧命中结算照常）——真机 A/B 开关：NPC 还手表现不对时不必回滚代码
// 即可对照。
func npcAIEnabled() bool { return os.Getenv("NINEYIN_NPC_AI") != "off" }

// ---- 战斗体状态 ----

// combatBuff 战斗体身上的一项 buff。
type combatBuff struct {
	id       string // ConfigID
	level    int
	category int           // buff_static.Category（m169 末位）
	expireAt time.Time     // 到期时刻（m169 的时间值）
	lua      string        // 钩子脚本
	dotHP    float64       // ContinuePropertyRec：每次触发的 HP 增减
	handle   uint64        // 脚本建的 buff 对象句柄（锁定技挂的为 0）
	caster   script.Handle // 施法者
}

// fightRelation fight_relation_rec 一行：对手对象 → 最近刷新时刻。
type fightRelation struct {
	target uint64
	at     time.Time
}

// playerLive 玩家战斗生存态（combatState.mu 护；连接生命周期，不落库）。
type playerLive struct {
	ready        bool
	hp, maxHP    int
	sp, maxSP    int
	fp           int
	parry        int
	parryCredit  float64
	logic        int
	fightAt      time.Time // 最近一次战斗活动（零值 = 非战斗）
	relations    []fightRelation
	hits         int // 当前技能累计击数
	attacker     uint64
	attackerName string
	buffs        []combatBuff
}

// npcCast NPC 一次起手。
type npcCast struct {
	skill  skilldata.Definition
	action string
	seq    int
	aux    uint64 // 技能脚本对象句柄（SelSkill 查询用）
}

// npcFighter NPC 战斗体（场景共享：同场景全部玩家看同一个；npcCatalog.combat）。
type npcFighter struct {
	mu       sync.Mutex
	handle   uint64
	spawn    npcfeed.Spawn
	scene    string
	pos      [4]float64 // 当前位（出生位起步，追击时推进）
	props    map[string]float64
	hp       int
	dead     bool
	deathAt  time.Time
	logic    int
	target   uint64 // 仇恨目标（玩家连接键；0 = 无）
	fightAt  time.Time
	cast     *npcCast
	castSeq  int
	nextCast time.Time
	chaseAt  time.Time
	step     int
	buffs    []combatBuff
	hurtBy   map[uint64]bool // 近期攻击者（击杀归属）
	skills   []npcSkillRef   // 技能表（npcSkills 懒建）
	moving   bool            // 追击/回归中（State=run 已下发）
}

// combatFrame 带可见性标记的下行帧：private 只发攻击者本人，public 同时
// 广播给同场景看得见该 NPC 的其它玩家。
type combatFrame struct {
	data    []byte
	private bool
}

// damageSource 一次伤害的来源描述（m24 17 的技能段）。
type damageSource struct {
	skill   string
	class   string // SkillNormal / SkillLock / Buffer
	crit    bool
	parried bool
	byBuff  bool
}

// ---- 事件队列 ----

type combatEvent struct {
	at  time.Time
	seq int
	run func()
}

type combatQueue struct {
	mu     sync.Mutex
	events []combatEvent
	seq    int
}

func (q *combatQueue) push(at time.Time, run func()) {
	q.mu.Lock()
	q.seq++
	heap.Push(q, combatEvent{at: at, seq: q.seq, run: run})
	q.mu.Unlock()
}

// drain 取出到期事件（at <= now）。
func (q *combatQueue) drain(now time.Time) []combatEvent {
	q.mu.Lock()
	defer q.mu.Unlock()
	var out []combatEvent
	for len(q.events) > 0 && !q.events[0].at.After(now) {
		out = append(out, heap.Pop(q).(combatEvent))
	}
	return out
}

func (q *combatQueue) Len() int { return len(q.events) }
func (q *combatQueue) Less(i, j int) bool {
	if !q.events[i].at.Equal(q.events[j].at) {
		return q.events[i].at.Before(q.events[j].at)
	}
	return q.events[i].seq < q.events[j].seq
}
func (q *combatQueue) Swap(i, j int) { q.events[i], q.events[j] = q.events[j], q.events[i] }
func (q *combatQueue) Push(x any)    { q.events = append(q.events, x.(combatEvent)) }
func (q *combatQueue) Pop() any {
	old := q.events
	n := len(old)
	event := old[n-1]
	q.events = old[:n-1]
	return event
}

// after 战斗时钟上的一次性事件（时长常量都是 time.Duration 类型，见上方
// 常量区）。
func (s *Server) after(d time.Duration, run func()) {
	s.combatQ.push(combatNow().Add(d), run)
}

// CombatPump 推进一次战斗事件（到点事件依次执行；测试手动驱动）。
func (s *Server) CombatPump() {
	for _, event := range s.combatQ.drain(combatNow()) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					s.trace("combat: 事件处理异常已兜住: %v", r)
				}
			}()
			event.run()
		}()
	}
}

func (s *Server) combatPumpLoop() {
	ticker := time.NewTicker(combatPumpStep)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.CombatPump()
		case <-s.pumpDone:
			return
		}
	}
}

// ---- 对象视图（nx_* 面） ----

// playerHandle 玩家句柄。
func playerHandle(pl *Player) script.Handle {
	return script.Handle{Kind: script.KindPlayer, ID: persistKey(pl.Conn1, pl.Conn2)}
}

// npcHandle NPC 句柄。
func npcHandle(f *npcFighter) script.Handle {
	return script.Handle{Kind: script.KindNPC, ID: f.handle}
}

// ---- 玩家对象视图 ----

type playerObject struct {
	s  *Server
	pl *Player
}

func (o *playerObject) Type() int      { return int(script.KindPlayer) }
func (o *playerObject) Config() string { return o.pl.Name }

// live 玩家运行态属性（生存值/逻辑态/当前技能）。
func (o *playerObject) live(name string) (script.Value, bool) {
	st := &o.pl.combat
	st.mu.Lock()
	defer st.mu.Unlock()
	o.s.ensureLiveLocked(st, o.pl)
	live := &st.live
	switch name {
	case "HP":
		return script.Int(int64(live.hp)), true
	case "MaxHP":
		return script.Int(int64(live.maxHP)), true
	case "MP":
		return script.Int(int64(st.mp)), true
	case "MaxMP":
		return script.Int(int64(playerMaxMP(o.pl))), true
	case "SP":
		return script.Int(int64(live.sp)), true
	case "MaxSP":
		return script.Int(int64(live.maxSP)), true
	case "FP":
		return script.Int(int64(live.fp)), true
	case "ParryValue":
		return script.Int(int64(live.parry)), true
	case "MaxParry":
		return script.Int(int64(o.s.parryValueTotal(o.pl))), true
	case "Dead":
		return script.Int(0), true // 玩家死亡未接（致命伤钳到 1 点，见 damagePlayer）
	case "LogicState":
		return script.Int(int64(live.logic)), true
	case "Hits":
		return script.Int(int64(live.hits)), true
	case "InParry":
		o.pl.parryMu.Lock()
		parrying := o.pl.parrying
		o.pl.parryMu.Unlock()
		return script.Bool(parrying), true
	case "CurSkillID":
		return script.String(st.curSkill), true
	case "CurSkill":
		if st.curSkill == "" || st.skillObj == 0 {
			return script.Nil(), false
		}
		return script.Ref(script.Handle{Kind: script.KindHelper, Helper: script.HelperSkill, ID: st.skillObj}), true
	case "TargetSkill":
		if st.targetSkill == 0 {
			return script.Nil(), false
		}
		return script.Ref(script.Handle{Kind: script.KindHelper, Helper: script.HelperSkill, ID: st.targetSkill}), true
	}
	return script.Nil(), false
}

func (o *playerObject) Query(name string) (script.Value, bool) {
	if v, ok := o.live(name); ok {
		return v, true
	}
	if value, ok := o.s.playerPanel(o.pl)[name]; ok {
		return script.Float(value), true
	}
	return script.Nil(), false
}

// Set 可写运行态（HP/MP/SP/FP/ParryValue/LogicState），变化即推帧。
func (o *playerObject) Set(name string, v script.Value) bool {
	st := &o.pl.combat
	st.mu.Lock()
	o.s.ensureLiveLocked(st, o.pl)
	live := &st.live
	b := newFrameBuilder(o.pl)
	switch name {
	case "HP":
		live.hp = clampInt(int(v.Number()), 0, live.maxHP)
		b.prop("HPRatio", entergen.Int32Value(hpRatio(live.hp, live.maxHP)))
		b.prop("HP", entergen.Int32Value(int32(live.hp)))
	case "MP":
		st.mp = clampInt(int(v.Number()), 0, playerMaxMP(o.pl))
		b.prop("MPRatio", entergen.Int32Value(mpRatio(st.mp, playerMaxMP(o.pl))))
		b.prop("MP", entergen.Int32Value(int32(st.mp)))
	case "SP":
		live.sp = clampInt(int(v.Number()), 0, live.maxSP)
		b.prop("SP", entergen.Int32Value(int32(live.sp)))
	case "FP":
		live.fp = clampInt(int(v.Number()), 0, live.maxSP)
		b.prop("FP", entergen.Int32Value(int32(live.fp)))
	case "ParryValue":
		live.parry = clampInt(int(v.Number()), 0, int(o.s.parryValueTotal(o.pl)))
		b.prop("ParryValue", entergen.Int32Value(int32(live.parry)))
	case "LogicState":
		live.logic = int(v.Number())
		if live.logic == logicNormal {
			live.fightAt = time.Time{}
		}
		b.prop("LogicState", entergen.ByteValue(byte(live.logic)))
	default:
		st.mu.Unlock()
		return false
	}
	st.mu.Unlock()
	o.s.sendFrames(o.pl, b.frames)
	return true
}

func (o *playerObject) Position() ([4]float64, bool) { return o.pl.roleState.Position, true }

// ---- NPC 对象视图 ----

type npcObject struct {
	s *Server
	f *npcFighter
}

func (o *npcObject) Type() int      { return int(script.KindNPC) }
func (o *npcObject) Config() string { return o.f.spawn.ConfigID }
func (o *npcObject) Script() string { return o.f.spawn.Script }

func (o *npcObject) Query(name string) (script.Value, bool) {
	f := o.f
	f.mu.Lock()
	defer f.mu.Unlock()
	switch name {
	case "HP":
		return script.Int(int64(f.hp)), true
	case "Dead":
		return script.Int(int64(btoi(f.dead))), true
	case "LogicState":
		return script.Int(int64(f.logic)), true
	case "StopAI":
		return script.Int(0), true
	case "CurSkillID":
		if f.cast == nil {
			return script.String(""), true
		}
		return script.String(f.cast.skill.ID), true
	case "CurSkillTarget", "SelSkill":
		if f.cast == nil {
			return script.Nil(), false
		}
		if name == "SelSkill" {
			return script.Ref(script.Handle{Kind: script.KindHelper, Helper: script.HelperSkill, ID: f.cast.aux}), true
		}
		if pl := o.s.playerByKey(f.target); pl != nil {
			return script.Ref(playerHandle(pl)), true
		}
		return script.Nil(), false
	}
	if value, ok := f.props[name]; ok {
		return script.Float(value), true
	}
	return script.Nil(), false
}

func (o *npcObject) Set(name string, v script.Value) bool {
	f := o.f
	f.mu.Lock()
	defer f.mu.Unlock()
	switch name {
	case "HP":
		f.hp = clampInt(int(v.Number()), 0, int(f.props["MaxHP"]))
	case "LogicState":
		f.logic = int(v.Number())
		if f.logic == logicNormal {
			f.fightAt = time.Time{}
			f.target = 0
		}
	default:
		return false
	}
	return true
}

func (o *npcObject) Position() ([4]float64, bool) {
	o.f.mu.Lock()
	defer o.f.mu.Unlock()
	return o.f.pos, true
}

// ---- 辅助对象（技能/buff）视图 ----

// auxObject 技能或 buff 对象：技能只读（定义的字段），buff 可写（脚本
// nx_set_property 设 Level/LifeTime 等）并带记录表（DirectPropertyRec 等）。
type auxObject struct {
	mu      sync.Mutex
	kind    script.Helper
	config  string
	script  string // LuaScript（钩子脚本）
	def     skilldata.Definition
	props   map[string]script.Value
	records map[string][][]script.Value
	owner   script.Handle // 技能：施法者；buff：Sender
}

func (o *auxObject) Type() int      { return int(script.KindHelper) }
func (o *auxObject) Config() string { return o.config }
func (o *auxObject) Script() string { return o.script }
func (o *auxObject) Position() ([4]float64, bool) {
	return [4]float64{}, false
}

func (o *auxObject) Query(name string) (script.Value, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.queryLocked(name)
}

// queryLocked 属性查询（调用方持 o.mu）：Sender → 可写属性表 → 技能静态
// 包列。锁内只调同样不加锁的 staticLocked（调加锁的 SkillStatic 会自锁）。
func (o *auxObject) queryLocked(name string) (script.Value, bool) {
	if name == "Sender" && o.owner.Valid() {
		return script.Ref(o.owner), true
	}
	if v, ok := o.props[name]; ok {
		return v, true
	}
	if o.kind == script.HelperSkill {
		return o.staticLocked(o.def, name)
	}
	return script.Nil(), false
}

func (o *auxObject) Set(name string, v script.Value) bool {
	if o.kind != script.HelperBuffer {
		return false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.props[name] = v
	return true
}

// SkillStatic 技能静态包列（官服 Lua 按源表列名取技能对象属性）。
func (o *auxObject) SkillStatic(staticData int, column string) (script.Value, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.staticLocked(o.def, column)
}

// staticLocked 静态包列取值（调用方持 o.mu）。
func (o *auxObject) staticLocked(d skilldata.Definition, column string) (script.Value, bool) {
	switch column {
	case "EffectType":
		return script.Int(int64(d.EffectType)), true
	case "DamageType":
		return script.Int(int64(d.DamageType)), true
	case "IsDamage":
		return script.Int(int64(d.IsDamage)), true
	case "TargetType":
		return script.Int(int64(d.TargetType)), true
	case "LockType":
		return script.Int(int64(btoi(d.Lock.OK))), true
	case "LockSrcTime":
		return script.Int(int64(d.Lock.SrcTime)), true
	case "LockTargetTime":
		return script.Int(int64(d.Lock.TargetTime)), true
	case "LockRatio":
		return script.Int(int64(d.Lock.Ratio)), true
	case "LuaScript":
		return script.String(d.LuaScript), true
	case "TaoLu":
		return script.String(d.TaoLu), true
	case "CoolDownCategory":
		return script.Int(int64(d.CoolDownCategory)), true
	case "MinVarPropNo":
		return script.Int(int64(d.VarPropNo)), true
	case "Level":
		return script.Int(int64(d.Level)), true
	case "Hits", "HitCount":
		return script.Int(int64(d.HitCount)), true
	case "AConsumeMP":
		return script.Int(int64(d.Consume.MP)), true
	case "PConsumeMP":
		return script.Float(d.Consume.PercentMP), true
	}
	return script.Nil(), false
}
