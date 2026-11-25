package fssoloworld

// skill.go C2S msg211 技能施放受理链（官服女号抓包逐帧定版，
// E:\Game\9yin\9yin-server-go2\资料\初始账号登陆-完整引导-女.pcapng 8032 线
// 197~398s 共 41 次施放）。技能配置来自 tools/datagen skill 构建产物
//（internal/skilldata 联查），本文件只管流程与帧序。
//
// 受理形态（按技能静态包与距离分流）：
//
//	普通技（SkillNormal）      [清扫] → CurSkillID → [CurSkillTarget] → 冷却行
//	                          → [MPRatio/MP] → 0x1F → m337 → m20
//	锁定技（SkillLock）直收带内 [清扫] → CurSkillID → CurSkillTarget → m284 →
//	                          冷却行 → MPRatio/MP → 0x1F → LastWGHit →
//	                          m24 目标 40 → m19 → CantMove/CantParry=1
//	锁定冲锋技超直收带         暂存簇（skillrush.go）：CurSkillID →
//	                          CurSkillTarget → m25 → PauseSkill → m707[15]
//	超距（非冲锋）             [清扫] → m24 目标 80 [目标位 + 朝向]
//
// 闸门（顺序即官服语义）：
//	  - 出招进行中再按 → m707[9000+33]（其他技能正在使用中）；
//	  - 冷却中 → 零应答（skillcooldown.go；客户端按冷却表自行拦键）；
//	  - MP 不足 → m707[9000+21]；
//	  - 需目标而无目标 → m707[9000+30]。
// 结果码 = 9000 + lua/skill/skill_error_define.lua 的 USESKILL_RESULT_*。
//
// 本阶段不含命中/伤害结算（需 NPC 战斗态：HP、仇恨、死亡），收招只清
// CurSkillID 与 CantMove/CantParry。

import (
	"encoding/binary"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/skilldata"
)

// C2S 自定义消息号。
const (
	clientCustomUseSkill     int32 = 211 // 技能施放
	clientCustomItemSkill    int32 = 270 // 技能道具施放（任务技能）[技能名]
	clientCustomDashLanding  int32 = 212 // 冲锋到位申报 [x, y, z, 朝向, 尾 1=冲刺/0=步行]
	clientCustomRushApproach int32 = 220 // 冲锋接近上报 [x, y, z, 朝向]
)

// 结果码 = useSkillResultBase + USESKILL_RESULT_*（skill_error_define.lua）。
// m211 链走 m707[码]；m270 技能道具链的多数拒绝是提示消息 m1[2,"码"]
// （女号包 721~815s 任务技能段 237 次施放的统计）。
const (
	useSkillResultBase              = 9000
	useSkillResultMPLess            = 21 // 能量不足
	useSkillResultNoTarget          = 30 // 没有选择技能使用目标
	useSkillResultOtherSkillHitting = 33 // 其他技能正在使用中
	useSkillResultCooldown          = 9  // 技能在冷却中（m270：m1[2,"9009"]）
	useSkillResultTargetDead        = 16 // 对象已死亡（m270：m707[9016]）
	useSkillResultSPLess            = 22 // 战意不足
	// useSkillResultHitStun 受击硬直中：定义文件无此码，抓包 725.942 被击
	// （atkhit3）后 80ms 施放任务技能即回 m1[2,"9023"]。
	useSkillResultHitStun = 23
	// useSkillStateNoTarget m270 无目标时紧跟 m1[2,"9030"] 的 m707（抓包
	// 722.992）。
	useSkillStateNoTarget = 7
	// useSkillStateStaged 暂存受理确认（m707[15]，抓包 206.558）。
	useSkillStateStaged = 15
	// useSkillResultDodged 目标闪避（抓包 226.276：锁定技被闪后的 m707[14]）。
	useSkillResultDodged = 14
)

// 出招锁缺省时长（抓包实测：普攻 473ms 收招；其余技能取流程包
// LockBuffTime 或锁定包 LockSrcTime）。
const (
	defaultNormalLockMS = 500
	// rushDirectBandExtra 锁定技直收带 = castRange + 此值（抓包 2.41m 直收、
	// 5.34m 暂存）。
	rushDirectBandExtra = 0.5
	// defaultQingGongPoint 轻功点上限/初值（登录面板 QingGongPoint=150）。
	defaultQingGongPoint = 150
)

// combatNow 战斗会话时钟（冷却/闸门判定用；测试可替换）。收招等定时器
// 仍走真实时钟。
var combatNow = time.Now

// combatState 玩家战斗会话态（连接生命周期，不落库）。
type combatState struct {
	mu        sync.Mutex
	cooldowns []cooldownRow
	lastTaoLu string // 上一次受理技能的套路（换套路共享冷却闸）
	curSkill  string // 进行中的技能（空 = 无）
	curTarget uint64 // 最近一次下发的 CurSkillTarget（0 = None）
	castSeq   int    // 施放序号（收招定时器防串）
	locked    bool   // CantMove/CantParry 置位中（锁定义簇 / 冲锋在途）
	staged    *stagedRush
	// selected 最近一次选靶（msg431）；m211 不带目标时回落。
	selected uint64
	// mpCredit 回复小数累积；qinggong 轻功点。
	mp       int
	mpReady  bool
	mpCredit float64
	qinggong int
	// selfBuff 施放期间挂在玩家身上的自身 buff（技能事件包；收招时摘）。
	selfBuff *activeSelfBuff
	// normalSteps 技能 ID → 连招动作轮转下标（流程包 HitAction 列表；按
	// 技能各自轮转——别的技能施放不推进普攻的连招序号）。
	normalSteps map[string]int
	// live 战斗生存态 + 当前技能的脚本对象句柄（combat.go）。
	live        playerLive
	skillObj    uint64
	targetSkill uint64
	// calmAt 已排的脱战检查时刻（零值 = 无；combatcore.go armCalmLocked）。
	calmAt time.Time
}

// skillCast m211 / m270 解码结果。
//
//	[211, 技能, x, y, z, 朝向, 0, 0, 0, "", [目标, 目标x, 目标y, 目标z]]
//	[270, 技能]
//
// item = m270 技能道具施放（任务技能）：消息不带坐标与目标，坐标取玩家
// 的服务端位置、目标回落最近选靶，两条入口此后走同一条受理链。
type skillCast struct {
	skill       string
	x, y, z     float32
	orient      float32
	targetID    uint32
	targetOwner uint32
	targetX     float32
	targetY     float32
	targetZ     float32
	hasTarget   bool
	item        bool
}

func (c skillCast) targetKey() uint64 {
	return uint64(c.targetID) | uint64(c.targetOwner)<<32
}

// finiteFloat TVarList 数值取有限 float32（f32/f64 双型）。
func finiteFloat(value entergen.CustomInboundValue) (float32, bool) {
	number, ok := customFloat(value)
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) || math.Abs(number) > 100000 {
		return 0, false
	}
	return float32(number), true
}

// parseSkillCast m211 值表解码（短式 10 值 / 长式 14 值）。
func parseSkillCast(values []entergen.CustomInboundValue) (skillCast, bool) {
	var cast skillCast
	if len(values) < 6 || values[0].Type != 2 || values[0].Int32 != clientCustomUseSkill || values[1].Type != 6 {
		return cast, false
	}
	cast.skill = strings.TrimSpace(values[1].Text)
	if cast.skill == "" {
		return cast, false
	}
	var coords [4]float32
	for i := range coords {
		value, ok := finiteFloat(values[2+i])
		if !ok {
			return cast, false
		}
		coords[i] = value
	}
	cast.x, cast.y, cast.z, cast.orient = coords[0], coords[1], coords[2], coords[3]
	if len(values) >= 11 && values[10].Type == 8 {
		cast.targetID = binary.LittleEndian.Uint32(values[10].Raw[:])
		cast.targetOwner = binary.LittleEndian.Uint32(values[10].Raw[4:])
		cast.hasTarget = cast.targetKey() != 0
		if len(values) >= 14 {
			cast.targetX, _ = finiteFloat(values[11])
			cast.targetY, _ = finiteFloat(values[12])
			cast.targetZ, _ = finiteFloat(values[13])
		}
	}
	return cast, true
}

// ---- 帧构造小件（纯函数，测试可直调） ----

// frameBuilder 帧簇累加器：首个编码错误即止，调用方统一判错。ident=玩家
// wire ident（newFrameBuilder 按会话注入；零值回落 fork 固定身份——测试直构）。
type frameBuilder struct {
	frames [][]byte
	err    error
	ident  wireIdent
}

// newFrameBuilder 会话帧簇（ident 取玩家当前值）。
func newFrameBuilder(pl *Player) *frameBuilder {
	return &frameBuilder{ident: pl.wireIdent()}
}

// playerIdent 累加器 ident。
func (b *frameBuilder) playerIdent() wireIdent { return b.ident.orFixed() }

func (b *frameBuilder) add(frame []byte, err error) {
	if b.err != nil {
		return
	}
	if err != nil {
		b.err = err
		return
	}
	b.frames = append(b.frames, frame)
}

// prop 单属性 0x10 帧（玩家对象）。
func (b *frameBuilder) prop(name string, value entergen.PropertyValue) {
	ident := b.playerIdent()
	frame, err := entergen.ObjectPropertyFrame(ident.objectID, ident.ownerID, 0,
		[]entergen.IndexedProperty{entergen.NamedProperty(name, value)})
	b.add(frame, err)
}

// custom 0x1E 自定义帧。
func (b *frameBuilder) custom(messageID int32, values ...entergen.CustomValue) {
	frame, err := entergen.CustomMessageFrame(messageID, values...)
	b.add(frame, err)
}

// location 玩家 0x1F 位置帧。
func (b *frameBuilder) location(x, y, z, orient float32) {
	ident := b.playerIdent()
	b.add(entergen.LocationFrame(ident.objectID, ident.ownerID, x, y, z, orient), nil)
}

// selfValue 玩家对象值（自定义消息里的自身 obj）。
func (b *frameBuilder) selfValue() entergen.CustomValue {
	ident := b.playerIdent()
	return entergen.CustomObject(ident.objectID, ident.ownerID)
}

// targetValue 对象键 → 对象值（0 = None）。
func targetValue(key uint64) entergen.CustomValue {
	return entergen.CustomObject(uint32(key), uint32(key>>32))
}

// towardOrient 自 (x,z) 指向目标 (tx,tz) 的朝向 atan2(dx, dz)。
func towardOrient(x, z, tx, tz float64) float32 {
	return float32(math.Atan2(tx-x, tz-z))
}

// normalizeOrient 朝向折到 [0, 2π)。
func normalizeOrient(orient float32) float32 {
	value := float64(orient)
	for value < 0 {
		value += 2 * math.Pi
	}
	for value >= 2*math.Pi {
		value -= 2 * math.Pi
	}
	return float32(value)
}

// mpRatio 官服 MPRatio 口径（截断：52/55 → 94）。
func mpRatio(mp, maxMP int) int32 {
	if maxMP <= 0 {
		return 0
	}
	return int32(mp * 100 / maxMP)
}

// mpRegenPerSecond 站立 MP 回复速率：登录面板已按官服公式下发
// MPUpSpeed = ceil(上限/10)（每 2 秒一档）——上限 55 时 3/秒，与抓包
// 386.5/391.6/396.6s 三次 +3 的观测一致。
func mpRegenPerSecond(maxMP int) float64 {
	if maxMP <= 0 {
		return 0
	}
	return float64((maxMP+9)/10) / 2
}

// ---- 技能配置入口 ----

var skillCatalogCache struct {
	mu      sync.Mutex
	loaded  bool
	catalog *skilldata.Catalog
	err     error
}

// skillDataRoots 技能产物数据根候选：数据根（npcfeed.ResolveRoot：
// NINEYIN_DATA_ROOT / 部署 Res / 仓库 testdata）之后再补仓内相对候选。
func skillDataRoots() []string {
	var roots []string
	if root := npcfeed.ResolveRoot(); root != "" {
		roots = append(roots, root)
	}
	return append(roots, "testdata", "../testdata", "../../testdata")
}

// skillCatalog 技能配置目录（进程级缓存）：首个含产物的数据根胜出；全缺
// 返回 nil（调用方按技能不可用处理，只 trace 一次不刷屏）。
func (s *Server) skillCatalog() *skilldata.Catalog {
	skillCatalogCache.mu.Lock()
	defer skillCatalogCache.mu.Unlock()
	if !skillCatalogCache.loaded {
		skillCatalogCache.loaded = true
		for _, root := range skillDataRoots() {
			catalog, err := skilldata.Load(root)
			if err == nil {
				skillCatalogCache.catalog = catalog
				s.trace("skill catalog loaded from %s", root)
				break
			}
			skillCatalogCache.err = err
		}
		if skillCatalogCache.catalog == nil {
			s.trace("skill catalog unavailable: %v（go run ./tools/datagen skill 构建）", skillCatalogCache.err)
		}
	}
	return skillCatalogCache.catalog
}

// skillDefinition 按 ConfigID 取技能定义（等级取玩家已学等级，未学按 1 级）。
func (s *Server) skillDefinition(pl *Player, id string) (skilldata.Definition, bool) {
	catalog := s.skillCatalog()
	if catalog == nil {
		return skilldata.Definition{}, false
	}
	level := 1
	for _, learned := range pl.roleState.Skills {
		if learned.ConfigID == id && learned.Level > 0 {
			level = learned.Level
			break
		}
	}
	def, err := catalog.Definition(id, level)
	if err != nil {
		s.trace("skill %s unavailable client=%d: %v", id, pl.Conn1, err)
		return def, false
	}
	return def, true
}

// playerMaxMP MP 上限（entergen 登录面板同口径：基础值 + 首本内功加成）。
func playerMaxMP(pl *Player) int {
	maxMP := pl.roleState.MP
	if len(pl.roleState.NeiGong) > 0 {
		maxMP += pl.roleState.NeiGong[0].MaxMPAdd
	}
	return maxMP
}

// ensureVitals 首次用到时按登录面板口径初始化会话 MP/轻功点（调用方持锁）。
func (st *combatState) ensureVitals(pl *Player) {
	if st.mpReady {
		return
	}
	st.mp, st.mpReady, st.qinggong = playerMaxMP(pl), true, defaultQingGongPoint
}

// castLockMS 出招锁时长（CurSkillID 置位 → 收招；期间再按 → m707 9033）。
func castLockMS(def skilldata.Definition) time.Duration {
	ms := defaultNormalLockMS
	switch {
	case def.Script == skilldata.ScriptLock && def.Lock.SrcTime > 0:
		ms = def.Lock.SrcTime
	case def.Flow.LockBuffTimeMS > 0:
		ms = def.Flow.LockBuffTimeMS
	}
	return time.Duration(ms) * time.Millisecond
}

// skillTarget 技能目标的服务端口径（位置 + 朝向 + 名字）。
type skillTarget struct {
	pos    [3]float64
	orient float32 // 服务端记录的朝向（弧度，atan2(dx,dz) 约定）
	name   string
	known  bool
}

// resolveSkillTarget 目标的服务端位置/朝向/名字（NPC 目录；未知对象 known=false）。
func (s *Server) resolveSkillTarget(pl *Player, cast *skillCast) skillTarget {
	if !cast.hasTarget {
		return skillTarget{}
	}
	pl.npcMu.Lock()
	scene := pl.npcScene
	pl.npcMu.Unlock()
	spawn, ok := s.findNPCSpawn(scene, cast.targetID)
	if !ok || spawn.OwnerID != cast.targetOwner {
		return skillTarget{}
	}
	return skillTarget{pos: [3]float64{float64(spawn.X), float64(spawn.Y), float64(spawn.Z)},
		orient: spawn.Orient, name: spawn.ConfigID, known: true}
}

// ---- 主入口 ----

// handleSkillCast C2S msg211 技能施放：判定与状态变更在会话锁内完成，只
// 产出「待发帧 + 发送后的动作」，锁外发送——每条出口同一收口。
func (s *Server) handleSkillCast(pl *Player, message entergen.CustomInbound) {
	cast, ok := parseSkillCast(message.Values)
	if !ok {
		s.trace("skill cast malformed client=%d", pl.Conn1)
		return
	}
	def, ok := s.skillDefinition(pl, cast.skill)
	if !ok {
		return
	}
	frames, followUp := s.planSkillCast(pl, def, cast)
	if len(frames) > 0 {
		s.sendFrames(pl, frames)
	}
	if followUp != nil {
		followUp()
	}
}

// handleItemSkillCast C2S m270 技能道具施放（任务技能）：[270, 技能名]。女号
// 包 721~815s 的任务战斗 237 次施放全走它、没有一次 m211。消息不带坐标与
// 目标：坐标取玩家的服务端位置，需目标的技能回落最近选靶，此后与 m211 同
// 一条受理链（差别见 planSkillCast 的 item 分支）。
func (s *Server) handleItemSkillCast(pl *Player, message entergen.CustomInbound) {
	if len(message.Values) < 2 || message.Values[1].Type != 6 {
		s.trace("item skill cast malformed client=%d", pl.Conn1)
		return
	}
	def, ok := s.skillDefinition(pl, message.Values[1].Text)
	if !ok {
		return
	}
	pos := pl.roleState.Position
	cast := skillCast{skill: def.ID, item: true, x: float32(pos[0]), y: float32(pos[1]), z: float32(pos[2]),
		orient: float32(pos[3])}
	frames, followUp := s.planSkillCast(pl, def, cast)
	if len(frames) > 0 {
		s.sendFrames(pl, frames)
	}
	if followUp != nil {
		followUp()
	}
}

// resultText m1[2,"<结果码>"] 的码值（m270 链的提示消息形态）。
func resultText(code int) entergen.CustomValue {
	return entergen.CustomString(strconv.Itoa(useSkillResultBase + code))
}

// planSkillCast 受理规划（返回 nil, nil = 帧编码失败，零发送）。
//
// m270（cast.item）与 m211 的差别（女号包任务技能段）：
//
//	冷却中       m1[2,"9009"]（m211 零应答）
//	无目标       m1[2,"9030"] → m707[7]（m211：m707[9030]）
//	目标已死亡   m707[9016]
//	受理帧序     普通技：CurSkillID → 冷却行 → MP/SP → [自身 buff] → m20
//	             （没有 m211 的 0x1F 与 m337——消息不带坐标）
//	             锁定技：… → 0x1F（服务端位置）→ LastWGAttackNpc=目标名 →
//	             m24 40 → m19（技能名位是命中动作名）→ CantMove → CantParry
func (s *Server) planSkillCast(pl *Player, def skilldata.Definition, cast skillCast) ([][]byte, func()) {
	st := &pl.combat
	st.mu.Lock()
	defer st.mu.Unlock()
	b := newFrameBuilder(pl)
	nowMS := filetimeMS(combatNow())

	// 暂存中的冲锋技被新按键覆盖：三清（CurSkillID="" → m26 → PauseSkill=""）
	// 后按新技能重新分流（2433 参考 rearm 同序）。
	if st.staged != nil {
		b.frames = append(b.frames, s.clearStagedLocked(b.ident, st)...)
	}
	// m270 的冷却闸门先于出招闸门：施放途中再按同技能，官服回的是冷却提示
	// 而不是 9033（抓包 722.139 受理 → 722.286 再按 → 722.324 m1[2,"9009"]，
	// 其前没有清扫帧）。
	if cast.item {
		if reason := cooldownGate(st.cooldowns, st.lastTaoLu, def, nowMS); reason != "" {
			s.trace("skill %s blocked by %s client=%d", def.ID, reason, pl.Conn1)
			b.custom(1, entergen.CustomInt(2), resultText(useSkillResultCooldown))
			return b.done()
		}
	}
	// 出招进行中：其他技能正在使用中。
	if st.curSkill != "" {
		b.custom(707, entergen.CustomInt(useSkillResultBase+useSkillResultOtherSkillHitting))
		return b.done()
	}
	// 目标回落：需目标的技能在 m211 不带目标时取最近选靶（客户端先 m431
	// 再 m211）。自身范围技不回落——抓包里它们的 m20 目标恒为空、朝向取
	// 玩家自报朝向。
	if !cast.hasTarget && def.NeedsTarget() && st.selected != 0 {
		cast.targetID, cast.targetOwner = uint32(st.selected), uint32(st.selected>>32)
		cast.hasTarget = true
	}
	if cast.hasTarget {
		st.selected = cast.targetKey()
	}
	// 清扫对每次被处理的按键都发生，受理/拒绝同款。
	sweep, rows := sweepCooldowns(b.ident.objectID, b.ident.ownerID, st.cooldowns, def, nowMS)
	st.cooldowns = rows
	b.frames = append(b.frames, sweep...)

	// 冷却中：m211 零应答（客户端按冷却表自行拦键）；m270 提示冷却中。
	if reason := cooldownGate(st.cooldowns, st.lastTaoLu, def, nowMS); reason != "" {
		s.trace("skill %s blocked by %s client=%d", def.ID, reason, pl.Conn1)
		if cast.item {
			b.custom(1, entergen.CustomInt(2), resultText(useSkillResultCooldown))
		}
		return b.done()
	}
	// MP。
	st.ensureVitals(pl)
	if def.Consume.MP > st.mp {
		b.custom(707, entergen.CustomInt(useSkillResultBase+useSkillResultMPLess))
		return b.done()
	}
	// 战意（任务技能 musejuben_4/7/8/9 的消耗是 AConsumeSP）。
	s.ensureLiveLocked(st, pl)
	if def.Consume.SP > st.live.sp {
		b.custom(707, entergen.CustomInt(useSkillResultBase+useSkillResultSPLess))
		return b.done()
	}
	// 目标与射程。目标不在服务端目录时位置回落 m211 自报位（朝向无从得知，
	// 保持 0）。
	target := s.resolveSkillTarget(pl, &cast)
	if !target.known {
		target.pos = [3]float64{float64(cast.targetX), float64(cast.targetY), float64(cast.targetZ)}
	}
	needsTarget := def.NeedsTarget()
	distance := math.Hypot(target.pos[0]-float64(cast.x), target.pos[2]-float64(cast.z))
	if needsTarget && !cast.hasTarget {
		if cast.item {
			b.custom(1, entergen.CustomInt(2), resultText(useSkillResultNoTarget))
			b.custom(707, entergen.CustomInt(useSkillStateNoTarget))
		} else {
			b.custom(707, entergen.CustomInt(useSkillResultBase+useSkillResultNoTarget))
		}
		return b.done()
	}
	if cast.item && cast.hasTarget && needsTarget {
		pl.npcMu.Lock()
		scene := pl.npcScene
		pl.npcMu.Unlock()
		if f := s.fighterBySpawn(scene, cast.targetKey()); f != nil && f.isDead() {
			b.custom(707, entergen.CustomInt(useSkillResultBase+useSkillResultTargetDead))
			return b.done()
		}
	}
	if cast.hasTarget && needsTarget {
		if def.CanRush && distance > def.CastRange+rushDirectBandExtra {
			// 冲锋暂存：冷却/MP 在授予（c61）时结算，此处只发暂存簇。
			window := def.PauseTimeMS
			if window <= 0 {
				window = 30000
			}
			b.prop("CurSkillID", entergen.StringValue(def.ID))
			b.skillTarget(cast.targetKey(), st)
			b.custom(25, entergen.CustomInt(int32(window)), entergen.CustomString("skill_pause"))
			b.prop("PauseSkill", entergen.StringValue(def.ID))
			b.custom(707, entergen.CustomInt(useSkillStateStaged))
			st.castSeq++
			stage := &stagedRush{def: def, seq: st.castSeq, target: cast.targetKey(),
				targetPos: target.pos, targetName: target.name}
			stage.expiry = time.AfterFunc(time.Duration(window)*time.Millisecond,
				func() { s.expireStaged(pl, stage.seq) })
			st.staged, st.curSkill, st.curTarget = stage, def.ID, cast.targetKey()
			s.trace("skill %s staged rush dist=%.2f client=%d", def.ID, distance, pl.Conn1)
			return b.done()
		}
		if distance > def.CastRange {
			// 超距：回目标当前的服务端位置与朝向，客户端据此接近。末位浮点是
			// 目标朝向——女号抓包 7 条拒绝帧均与此前该目标最近一条服务端位置
			// 记录（0x1F / 0x21 条目 / m337）的朝向逐位相同
			//（tools/pcapskill/reject80.py）。
			b.custom(24, targetValue(cast.targetKey()), entergen.CustomInt(80),
				entergen.CustomFloat32(float32(target.pos[0])), entergen.CustomFloat32(float32(target.pos[1])),
				entergen.CustomFloat32(float32(target.pos[2])), entergen.CustomFloat32(target.orient))
			s.trace("skill %s out of range dist=%.2f range=%.2f client=%d", def.ID, distance, def.CastRange, pl.Conn1)
			return b.done()
		}
	}

	// ---- 受理：冷却行在写入成功后才落会话 ----
	rowFrames, rows, err := writeCooldownRows(b.ident.objectID, b.ident.ownerID, st.cooldowns, def, nowMS)
	if err != nil {
		s.trace("skill %s cooldown rows encode failed client=%d: %v", def.ID, pl.Conn1, err)
		return nil, nil
	}
	b.prop("CurSkillID", entergen.StringValue(def.ID))
	b.skillTarget(cast.targetKey(), st)
	if def.Script == skilldata.ScriptLock {
		b.custom(284)
	}
	b.frames = append(b.frames, rowFrames...)
	st.cooldowns = rows
	if def.Consume.MP > 0 {
		b.prop("MPRatio", entergen.Int32Value(mpRatio(st.mp-def.Consume.MP, playerMaxMP(pl))))
		b.prop("MP", entergen.Int32Value(int32(st.mp-def.Consume.MP)))
	}
	st.mp -= def.Consume.MP
	if def.Consume.QingGong > 0 && st.qinggong > 0 {
		st.qinggong = max(st.qinggong-def.Consume.QingGong, 0)
		b.prop("QingGongPoint", entergen.Int32Value(int32(st.qinggong)))
	}
	if def.Consume.SP > 0 {
		st.live.sp = max(st.live.sp-def.Consume.SP, 0)
		b.prop("SP", entergen.Int32Value(int32(st.live.sp)))
	}
	// 施放期间的自身 buff（技能事件包）：属性位 → buff 槽串 → m169 挂载。
	if spec := hitSpecOf(def.ID).selfBuff; spec != nil {
		s.grantSelfBuffLocked(b, st, pl, *spec)
	}
	st.lastTaoLu, st.curSkill, st.curTarget = def.TaoLu, def.ID, cast.targetKey()
	st.castSeq++
	seq := st.castSeq

	// m20 动作名 = 招式表查表键（连招段按技能各自轮转）。
	action := skillAction(st, def)
	orient := cast.orient
	if cast.hasTarget {
		orient = towardOrient(float64(cast.x), float64(cast.z), target.pos[0], target.pos[2])
	}
	if def.Script == skilldata.ScriptLock {
		st.locked = true
		b.location(cast.x, cast.y, cast.z, normalizeOrient(orient))
		if cast.item {
			b.prop("LastWGAttackNpc", entergen.StringValue(target.name))
		} else {
			b.prop("LastWGHit", entergen.Float32Value(100))
		}
		b.custom(24, targetValue(cast.targetKey()), entergen.CustomInt(40), entergen.CustomInt(1), b.selfValue())
		b.custom(19, skillLockMessageValues(b.ident, def, action, cast, target.pos, orient)...)
		b.prop("CantMove", entergen.ByteValue(1))
		b.prop("CantParry", entergen.Int32Value(1))
	} else {
		if !cast.item {
			b.location(cast.x, cast.y, cast.z, orient)
			b.custom(337, b.selfValue(), entergen.CustomFloat32(cast.x), entergen.CustomFloat32(cast.y),
				entergen.CustomFloat32(cast.z), entergen.CustomFloat32(orient), entergen.CustomInt(0))
		}
		aim := entergen.CustomObject(0, 0)
		if cast.hasTarget {
			aim = targetValue(cast.targetKey())
		}
		b.custom(20, b.selfValue(), entergen.CustomString(action), aim, entergen.CustomInt(0), entergen.CustomFloat32(0))
	}
	locked := st.locked
	s.trace("skill %s accepted dist=%.2f mp=%d client=%d", def.ID, distance, st.mp, pl.Conn1)
	return b.frames, func() {
		// 命中计划在受理帧落地后登记（帧编码失败时不登记）；出招锁至少
		// 撑到末击（抓包：CurSkillID 在末击帧之后才清）。
		lastHit := s.scheduleSkillHits(pl, def, cast, action, 0)
		s.scheduleFinish(pl, def, seq, locked, lastHit)
	}
}

// combatAfterFunc 收招定时器的排定入口：线上恒为真实时钟；战斗夹具把它
// 换到战斗时钟上，收招与命中同钟，用例不随真实时间与机器负载漂移。
var combatAfterFunc = time.AfterFunc

// scheduleFinish 排收招定时器：castLockMS 与「末击 + 余量」取长者。余量
// 须盖过战斗泵的节拍抖动——命中由泵按 combatPumpStep 放行，可能晚于名义
// 时刻；收招一旦抢到末击前面，此刻受理的新施放会顶掉序号，末击被丢。
func (s *Server) scheduleFinish(pl *Player, def skilldata.Definition, seq int, locked bool, lastHit time.Duration) {
	lock := castLockMS(def)
	if lastHit+finishMargin > lock {
		lock = lastHit + finishMargin
	}
	combatAfterFunc(lock, func() { s.finishCast(pl, seq, locked) })
}

// skillTarget CurSkillTarget 帧——仅当目标变化时发（抓包：值变才推）。
func (b *frameBuilder) skillTarget(key uint64, st *combatState) {
	if key == st.curTarget {
		return
	}
	st.curTarget = key
	b.prop("CurSkillTarget", entergen.ObjectValue(uint32(key), uint32(key>>32)))
}

// done 收口：编码失败返回零帧。
func (b *frameBuilder) done() ([][]byte, func()) {
	if b.err != nil {
		return nil, nil
	}
	return b.frames, nil
}

// skillLockMessageValues m19 锁定段起始帧的值表（不含消息号；抓包
// 207.777/355.780 逐值，整帧 144 字节）：
// [攻击者, 目标, 技能名, 己方锁定ms, 目标锁定ms, 打击点1 xyz, 朝向,
// 锁定己方位移, 打击点2 xyz, 常量 2.6166286, 锁定对方位移, 0, 0.0, 1, 1,
// 0.0, 目标位 xyz]。打击点 = 出招位 / 目标位各沿攻击方向前进对应位移。
// 技能名位填命中动作名（锁定包 HitAction；常规锁定技与技能 ID 同名，任务
// 技能不同：抓包 728.313 的 m19 技能名位是 CS_sl_lhq02）。
func skillLockMessageValues(ident wireIdent, def skilldata.Definition, name string, cast skillCast, target [3]float64, orient float32) []entergen.CustomValue {
	ident = ident.orFixed()
	lock := def.Lock
	if lock.SrcTime <= 0 {
		lock.SrcTime = 1500
	}
	if lock.TargetTime <= 0 {
		lock.TargetTime = 1500
	}
	srcDist, targetDist := lock.SrcDist, lock.TargetDist
	dx, dz := math.Sin(float64(orient)), math.Cos(float64(orient))
	f32 := entergen.CustomFloat32
	return []entergen.CustomValue{
		entergen.CustomObject(ident.objectID, ident.ownerID), targetValue(cast.targetKey()), entergen.CustomString(name),
		entergen.CustomInt(int32(lock.SrcTime)), entergen.CustomInt(int32(lock.TargetTime)),
		f32(cast.x + float32(dx*srcDist)), f32(cast.y), f32(cast.z + float32(dz*srcDist)),
		f32(orient), f32(float32(srcDist)),
		f32(float32(target[0] + dx*targetDist)), f32(float32(target[1])), f32(float32(target[2] + dz*targetDist)),
		f32(2.6166286), f32(float32(targetDist)),
		entergen.CustomInt(0), f32(0), entergen.CustomInt(1), entergen.CustomInt(1), f32(0),
		f32(float32(target[0])), f32(float32(target[1])), f32(float32(target[2])),
	}
}

// finishCast 收招：锁定位先清，再清 CurSkillID（抓包：收招只清 CurSkillID，
// CurSkillTarget 保持）。序号不符 = 已被新一轮施放覆盖，丢弃。
func (s *Server) finishCast(pl *Player, seq int, locked bool) {
	st := &pl.combat
	st.mu.Lock()
	if st.castSeq != seq {
		st.mu.Unlock()
		return
	}
	b := newFrameBuilder(pl)
	if locked {
		b.prop("CantMove", entergen.ByteValue(0))
		b.prop("CantParry", entergen.Int32Value(0))
	}
	// CurSkillID 可能已由命中引擎在末击/闪避时清掉（combat.go）：只清
	// 尚未清的，出招锁位照常释放。
	if st.curSkill != "" {
		b.prop("CurSkillID", entergen.StringValue(""))
	}
	// 施放期间的自身 buff 随收招摘掉（CurSkillID 之后；命中链若已摘过则空）。
	b.frames = append(b.frames, s.releaseSelfBuffLocked(pl, st)...)
	st.curSkill, st.locked = "", false
	st.mu.Unlock()
	s.sendFrames(pl, b.frames)
}

// combatTick 每秒 MP 回复（站立：登录面板 MPUpSpeed = ceil(上限/10) 每 2 秒
// 一档）。变化即推 MP 单属性帧（抓包 386.537/391.570/396.562 同形）。
func (s *Server) combatTick() {
	for _, pl := range s.players.Snapshot() {
		if pl.State != PlOnline {
			continue
		}
		st := &pl.combat
		st.mu.Lock()
		if !st.mpReady {
			st.mu.Unlock()
			continue
		}
		maxMP := playerMaxMP(pl)
		before, mp := st.mp, st.mp
		credit := st.mpCredit + mpRegenPerSecond(maxMP)
		for credit >= 1 && mp < maxMP {
			credit--
			mp++
		}
		st.mpCredit, st.mp = credit, mp
		st.mu.Unlock()
		if mp != before {
			s.sendFrames(pl, mpFrame(pl.wireIdent(), mp))
		}
		// 招架耐力回复（combatcore.go；抓包 226.834 起每秒 +1 回到上限）。
		if frames := s.parryRegen(pl); len(frames) > 0 {
			s.sendFrames(pl, frames)
		}
	}
}

// mpFrame MP 单属性帧（ident=玩家 wire ident）。
func mpFrame(ident wireIdent, mp int) [][]byte {
	b := &frameBuilder{ident: ident}
	b.prop("MP", entergen.Int32Value(int32(mp)))
	return b.frames
}

// sendFrames 依序下行一批帧。
func (s *Server) sendFrames(pl *Player, frames [][]byte) {
	for _, frame := range frames {
		s.sendToClient(pl, frame)
	}
}
