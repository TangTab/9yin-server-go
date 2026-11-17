package main

// 冲锋技能实现（v32 重写）——按《docs/combat/冲锋技能规格书.md》逐节实现：
//
//	§一  两段定界：接近段（到判距带前）=客户端职责（手动/自动靠近设置），
//	    服务端零驱动零假设；冲锋段（带内）=技能本身（c61→客户端冲刺→m212）。
//	§二  判距分带：直收带/空地混层闸/地面带(直收带,5] 2D/空中带(直收带,
//	    14.5] 3D；方言表与武技扩程覆盖。
//	§三  判断点消息：单体=m15/m220 进带申报；点地=m211 按压本身；stage
//	    期不发 c61。
//	§四  c61 授予簇：c61+CantMove/CantParry+CD 双行+MPRatio/MP（空中
//	    +QingGongPoint+m707[29]）+AOI 裸广播；dest.y 全档=目标 y+1.0（唯
//	    低→高空中档=平面）；成对 0x21 同步条目=方言事实但默认关（§4.1）。
//	§五  释放通道：m212 到位申报主通道+呆坐哨+30s 过期+取消/rearm 还原组；
//	    212 三分；轻功复位全出口。
//	§六  范围点地冲刺：c61 直冲点选点，ETA 终点重放 211 直收。
//	§七  直收执行簇（g 定版 17 帧）。
//	§八  受理簇（首发/rearm 两序+m25/m26 看门狗）。
//	§九  移动上报：位置收编+JumpTimes 镜像+回显方言；m220 进带判定。
//
// 历史演进、被推翻结论与逐帧考证见《冲锋与打坐外服方言总集.md》（档案）。

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

const (
	clientCustomMovementUpload  int32 = 15  // CLIENT_CUSTOMMSG_MOVE_SCENE
	clientCustomRushMoveUpload  int32 = 220 // 冲锋在途位移上报（4×f32，无 state 前缀）
	clientCustomCastDashLanding int32 = 212 // RESUME_SKILL 冲锋落地（尾值 1=冲刺 / 0=步行）
	clientCustomChangeWarning   int32 = 292 // CLIENT_CUSTOMMSG_CHANGEWARNING 预警切换
	clientCustomRushHitConfirm  int32 = 915 // 冲锋命中确认上报（外服私服无回应，静默消费）
	clientCustomCorpseQuery     int32 = 990 // 尸体拾取查询（同上，g 战后实证）
	clientCustomMapNavigate     int32 = 266 // 地图寻路标记（技能.pcapng：起跑前 ~73ms 一次，S2C 零回应）

	// 三层距离模型（g/f/b 联合定版 + Java SkillCastHandler 常量同源）。
	selectedRangeFallbackMeters = 8.0                // 无 TSP2/形状半径时的近战回落射程（skill_catalog 引用）
	rushStageMaxMeters          = 15.0               // RushRange 缺列时的回落（skill_catalog 引用，仅存档参考）
	rushArrivalRadiusMeters     = 3.0                // 落地到达半径（212 上报距目标 ≤3m = 落地）
	rushDirectCastSlackMeters   = 2.0                // bind 之上的直收余量
	rushPendingWindow           = 30 * time.Second   // 暂存出招有效期（外服私服 msg25 窗口=30000ms）
	defaultRushBindDistance     = float32(1.5)       // skill_lock_data 缺行时的身前绑定距离
	defaultGroundRushAction     = "qg_fastrun_2h_01" // 外服私服地面锁定冲锋动画
	convertDashLevel            = 1                  // 冲刺档位（外服私服锁定冲锋 c61=1）
	msgRushCommand              = 61                 // msg24 c1=61 冲刺/疾跑指令
	// rushDashLiftMeters：c61 终点在玩家上报 y 之上的抬升（恒 +1.0 小跳弧线）。
	rushDashLiftMeters = float32(1.0)
	// rushAirborneAboveMeters：高→低空中档判定线（玩家 211 自报 y 高出
	// 目标 y 0.5m 即空中档——2026-11-17 用户拍板：与低→高对称取 0.5，
	// (0.5,1.5] 浅下行亦走空中奔袭；技能4 俯冲样本均 >1.5 只是下界实证
	// 非上界）。冲锋族 geometry 已用 211 自报目标坐标，两侧同为客户端
	// 坐标系，无 NPC 内部 y 口径问题；旧 1e9 禁用线针对服务端坐标基准
	// 时代，v29 撤销。
	rushAirborneAboveMeters = float32(0.5)
	// rushGRushRangeMeters：玩家物理档案 GRushRange=5.0（player_actor.go
	// groundRushRange 同源；技能4.pcapng 玩家档案 LandRushSpeed=25/
	// AirRushSpeed=15 分档同源）。
	rushGRushRangeMeters = 5.0
	// rushGroundChargeHiMeters：3m 族地面冲锋带上界=GRushRange+Add **6.5**
	//（物理档案 groundRushRangeAdd=1.5 同源）。2026-11-17 真机定案：客户端
	// 自动接近稳定停在 5.19~5.43m 等授予并狂发 212——v29 的 hi=5.0 把带
	// 卡死在停点之外（「地面还是没有冲锋」根因）；技能.pcapng 官方三证
	// 5.91/6.09/6.10 与方言表下界 5.0（xytf02/mzjf02）同指 (5,6.5] 窗。
	// 下界维持直收带：「3~5m 立即冲锋」的按压/走进 4m 段照授。
	rushGroundChargeHiMeters = 6.5
	// rushChargeHeightGateMeters 空/地混层闸（v30，2026-11-16 用户定版+
	// 二轮澄清）：|出招位置 y − 目标 y| 超过该值=角色/目标任一在高处
	//（角色空-目标地 / 目标空-角色地 / 双空）——近身也不原地直收（隔空
	// 打高处/低处目标），改走冲锋链受理即冲；≤1m 视为同层，距离分带与
	// v29 完全一致（平地带内照常地面冲锋，其它规则不动）。
	rushChargeHeightGateMeters = float32(1.0)
	// rushChargeUpliftAirborneMeters 低→高空中奔袭档阈值（v30 用户补充）：
	// 目标高于出招位置超过该值的冲锋走空中档（fastruning flag2 15m/s 奔袭
	// 动画，非地面跑步动画）；高→低同 0.5m 对称（2026-11-17 用户拍板）。
	// 范围冲锋（点地）同规则。
	rushChargeUpliftAirborneMeters = float32(0.5)
	// rushAirChargeHiMeters 空中档授予窗上界（技能5.pcapng 六样本定版）：
	// ARushRange+Add=14.5，下界=直收带（与地面档连续无死区）；距离取 3D
	// 空间距离（样本 6.33~10.95 全落此窗；穿云纵 state7 飞行中 m220 即授，
	// c61 打断飞行转入奔袭）。玩家物理档案镜像 player_actor.go
	// airRushRange/airRushRangeAdd 同源。规格 §二。
	rushAirChargeHiMeters = 14.5
)

// rushChargeAirborne 冲锋空中档判定：高→低 >0.5m 或 低→高 >0.5m
//（2026-11-17 用户拍板双向对称 0.5；低→高线 v30 定版、高→低线由 1.5 收
// 至 0.5——技能4 俯冲样本均 >1.5 系下界实证非上界）→ 空中档（fastruning
// flag2 / 15m/s / ETA 3D）；|Δy|≤0.5m = 同层地面带。
func rushChargeAirborne(casterY, destY float32) bool {
	return casterY-destY > rushAirborneAboveMeters || destY-casterY > rushChargeUpliftAirborneMeters
}

// rushDashFallbackDelay 呆坐哨首查下限（规格 §五；var 化供测试缩短）：
// c61 后客户端 ≥max(ETA+500ms, 本值) 未自行移动（>0.5m）即在当前上报位
// 就地释放（2026-11-15 真机「c61 未消费呆站」实证形态）。
var rushDashFallbackDelay = 1500 * time.Millisecond

// rushDirectCastMeters 是直收带上界：≤该距离不走暂存链直接执行。
// v29（2026-11-16 官服观察定版）直收带=施法距离本位，按释放距离分族：
//   3m 族（castRange<5）= castRange+0.5——「< 施法距离阈值即立即释放」
//     （g 3.06m 直收实证带 ≥3.06、b 4.41m 暂存钉死带 <4.41，+0.5 恰容
//     两侧样本）；v14「近距也走两段式」撤销——近距按压立即释放。
//   6m 族（castRange≥5）= castRange+1（技能.pcapng 定版：hyjf02 6.95m
//     按下当毫秒 msg284 直收簇、kfkj02 1.19m 同形——宽容 1m）。
func rushDirectCastMeters(definition combatSkillDefinition) float64 {
	// v29 武技覆盖（lua rush.direct_band_m）优先。
	if definition.rushOverride != nil && definition.rushOverride.directBand > 0 {
		return definition.rushOverride.directBand
	}
	if definition.castRange < 5 {
		if near := float64(definition.castRange) + 0.5; near > 2 {
			return near
		}
		return 2
	}
	near := float64(definition.castRange) + 1
	if floor := rushArrivalRadiusMeters + 0.5; near < floor {
		near = floor
	}
	return near
}

// rushWalkBandAcceptMeters：212 走带认可半径=地面带上界（GRushRange+Add
// 6.5）。外服私服 walk-in 直收的 212 可落在 castRange+1 内任意点（技能.
// pcapng：hyjf02 212@5.81/5.58 认可落地）；真机自动接近停点 5.19~5.43m
// 的 212（尾0）也在此带内=走带内正常落地续招，不再误判超走带。
func rushWalkBandAcceptMeters(pending *pendingCastState) float64 {
	accept := rushGroundChargeHiMeters
	if bound := float64(pending.castRange) + 1; accept < bound {
		accept = bound
	}
	return accept
}

// rushDashDialect：per-skill c61 授予方言（技能1.pcapng 16 次授予观测，
// 2026-11-13 定版，docs/技能1pcapng多套路冲锋外服私服方言.md §1.2/§1.3）。
// c61 是服务端指令，外服私服在 m220 上报距目标进入该技能授予窗时 +30~90ms
// 下发；flag/动画随技能冲刺类别——lock 表三类动作 G=qg_fastrun(flag1)/
// A=qg_fastruning_s(flag2)/W=swim_rush，0h/2h 后缀=空手/持械。观测窗=各
// 技能 220 上报距目标带；未列技能回落 G 类（skill_lock_data
// LockGRushAction）+ 默认窗 (GRushRange, GRushRange+Add]。
type rushDashDialect struct {
	windowLo, windowHi float64
	level              int32
	action             string // 空=沿用 definition.groundRushAction（G 类）
}

var officialRushDashDialects = map[string]rushDashDialect{
	// hyjf02：220@8.1 授予，@6.1/7.3 窗外走入直收（19.1m 按压实证）。
	"CS_jh_hyjf02": {windowLo: 7.5, windowHi: 8.5},
	// hsjf02 观测窗 6.3~6.5 = 默认窗，不列。
	// 空手腿法（0h）与灭剑如虹走 A 类 fastruning flag2；sky 变体空中
	// 冲刺窗显著更宽（落差 4.2m/平距 7.1m 实证）。
	"CS_jz_xytf02":     {windowLo: 5.0, windowHi: 7.0, level: 2, action: "qg_fastruning_s_0h_01"},
	"CS_jz_xytf07_sky": {windowLo: 8.0, windowHi: 11.0, level: 2, action: "qg_fastruning_s_0h_01"},
	"CS_jz_xytf02_sky": {windowLo: 11.0, windowHi: 14.5, level: 2, action: "qg_fastruning_s_0h_01"},
	"CS_hs_kfkj02":     {windowLo: 9.0, windowHi: 11.0, level: 2, action: "qg_fastruning_s_2h_01"},
	"CS_jh_mzjf02":     {windowLo: 5.0, windowHi: 8.0},  // 220@13.8 窗外零授予
	"CS_jh_mzjf03":     {windowLo: 5.0, windowHi: 14.0}, // 220@13.2 仍授予
}

// rushDashTriggerOverrides 扩程冲刺武技表（skill_static.RushRange 实列）：
// 走到该距离即转 25m/s 冲刺段。默认 3m（技能3.pcapng：常规技 3m 带内
// c61+0x21 s5=25.0；wuji_lyfh05 实测 13m 外受理即冲=RushRange 20 生效）。
// rushDashChargeRangeSeeds 武技扩程手勘实据（v29 并入 rushChargeOverride
// 机制）：skill_static.RushRange 列缺行/部署侧数据不可得时的兜底实据。
// wuji_CS_jz_lyfh05（落英飞花·落尘）RushRange=20——技能4.pcapng 13m 外
// 受理即冲实证（「3m 冲锋改变到 20m」类武技）。
var rushDashChargeRangeSeeds = map[string]float64{
	"wuji_CS_jz_lyfh05": 20,
}

// rushChargeOverride 冲锋分档覆盖（v29 武技扩展点，2026-11-16）。来源
// 两级：① skill_static.RushRange 列实据（>GRushRange 即扩带——wuji_ 系
// 武技变体如 落英飞花·落尘 RushRange=20，技能4.pcapng 13m 外受理即冲
// 实证）；② lua 覆盖模块 M.compile 的 rush 指令（scripts/module/skill/
// <技能ID>.lua，热更走 lua_reload，字段级覆盖 ①）。零值字段=回落
// 方言表/族规则。
type rushChargeOverride struct {
	directBand   float64 // 直收带上界（0=族规则）
	chargeLo     float64 // 冲锋带下界（0=直收带）
	chargeHi     float64 // 冲锋带上界（0=族规则 GRushRange 5；落尘=20）
	level        int32   // c61 档（0=随地面/空中自动；1=G/2=A）
	groundAction string  // 地面动作名（空=lock 表）
	airAction    string  // 空中动作名（空=lock 表/推导）
}

// effective 带覆盖合并进方言（增量语义：零值字段保留底层值）。
func (o *rushChargeOverride) effective(base rushDashDialect, directBand float64) rushDashDialect {
	merged := base
	if o.chargeLo > 0 {
		merged.windowLo = o.chargeLo
	}
	if o.chargeHi > 0 {
		merged.windowHi = o.chargeHi
		// 6m 族底窗无带（windowLo=0）被扩程时，下界回落直收带。
		if merged.windowLo <= 0 {
			merged.windowLo = directBand
		}
	}
	if o.level != 0 {
		merged.level = o.level
	}
	if o.groundAction != "" {
		merged.action = o.groundAction
	}
	return merged
}

// rushDashDialectFor 解析技能授予方言。优先级：lua 武技覆盖
// （definition.rushOverride）> 方言表（技能1.pcapng 16 次授予观测）>
// 族规则（2026-11-17 修正定版）：
//   3m 族（castRange<5）：冲锋带=(直收带 castRange+0.5, GRushRange+Add
//     **6.5]——官服观察「3m~5m 立即冲锋」（下界本位）+ 技能.pcapng 官方
//     三证 5.91/6.09/6.10 + 真机自动接近停点 5.19/5.43m（上界本位；v29
//     的 hi=GRushRange 5.0 卡死在停点外=「地面没冲锋」根因）；
//   6m 族（castRange≥5）：无冲锋带（windowLo=Hi=0）——「< 施法距离阈值
//     即直收」，带外客户端自走，进直收带重按/停点 212 落地续招。
// 武技扩程（RushRange=20 类）经 rushOverride 把带放宽。
func rushDashDialectFor(definition combatSkillDefinition) rushDashDialect {
	base := rushDashDialect{}
	if dialect, ok := officialRushDashDialects[definition.id]; ok {
		base = dialect
	} else if definition.castRange < 5 {
		base = rushDashDialect{
			windowLo: rushDirectCastMeters(definition), // 直收带即冲锋带下界锚
			windowHi: rushGroundChargeHiMeters,
		}
	}
	if definition.rushOverride != nil {
		return definition.rushOverride.effective(base, rushDirectCastMeters(definition))
	}
	return base
}

// 冲锋簇属性索引（按名映射自本服表；外服私服现行表同名字 +2 偏移）。
var (
	propRushCantMove        = clientdata.MustPlayerProperty("CantMove").Index        // CantMove Byte（fork=203）
	propRushPauseSkill      = clientdata.MustPlayerProperty("PauseSkill").Index      // PauseSkill String（fork=220）
	propRushLogicState      = clientdata.MustPlayerProperty("LogicState").Index      // LogicState Byte（fork=463）
	propRushFP              = clientdata.MustPlayerProperty("FP").Index              // FP Int32（伤害段 FP/SP 对）（fork=462）
	propRushLastWGAttackNpc = clientdata.MustPlayerProperty("LastWGAttackNpc").Index // LastWGAttackNpc String（直收簇宣告目标模型）（fork=867）

	propQingGongPointIndex = clientdata.MustPlayerProperty("QingGongPoint").Index // QingGongPoint Int32（c61 扣后单帧）（fork=692）
	propJumpTimesIndex     = clientdata.MustPlayerProperty("JumpTimes").Index     // JumpTimes Int32（飞行计数镜像，起飞授/消耗归零）
)

// msg19 打击点拟合常数：两打击点分别在玩家沿「目标→玩家」延长线、
// 距目标恒定距离处（f/g 四样本 r+dist 拟合 4.04/4.55，±0.03）。
const (
	msg19StrikePointFar1 = 4.04
	msg19StrikePointFar2 = 4.55
	msg19PitchConstant   = float32(-2.5)
)

// pendingCastState 是一次越程按下暂存的出招。
type pendingCastState struct {
	skillID          string
	targetID         uint32
	targetOwner      uint32
	expiresUTC       time.Time
	canRush          bool
	castRange        float32
	bindDistance     float32
	groundRushAction string
	airRushAction    string
	airborne         bool
	// c61 授予方言（技能1.pcapng per-skill；未列技能= G 类默认窗）。
	dashWindowLo, dashWindowHi float64
	dashLevel                  int32
	dashAction                 string
	// geometry：冲锋链几何基准（211 自报目标坐标优先，缺省服务端坐标）。
	geometry worldcore.Transform
	// settled：CD/MP 已在 c61 时点结算；落地重入据此跳过重复结算。
	settled bool
	// 结算快照（staging 时从 definition 复制）。
	mpCost, spCost, qgCost int32
	cooldownCategory       int32
	cooldownTeam           int32
	personalCD, publicCD   time.Duration
	taoLu                  string
	nominalDamage          int32
}

// rushStageEnabled 对齐 Java -Dnineyin.rush.stage（默认开）。
func rushStageEnabled() bool {
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("NINEYIN_RUSH_STAGE")), "off")
}

// ---- playerActor 冲锋状态（锁纪律：全部经 p.mu） ----

func (p *playerActor) stagePendingCast(pending *pendingCastState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pendingCast = pending
	p.rushDashIssued = false
	p.rushDashAirborne = false
}

func (p *playerActor) peekPendingCast(now time.Time) *pendingCastState {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pendingCast == nil || now.After(p.pendingCast.expiresUTC) {
		return nil
	}
	return p.pendingCast
}

func (p *playerActor) consumePendingCast(now time.Time) *pendingCastState {
	p.mu.Lock()
	defer p.mu.Unlock()
	pending := p.pendingCast
	p.pendingCast = nil
	if pending == nil || now.After(pending.expiresUTC) {
		return nil
	}
	return pending
}

func (p *playerActor) markRushInFlight() {
	p.mu.Lock()
	p.rushInFlight = true
	p.mu.Unlock()
}

func (p *playerActor) clearRushInFlight() {
	p.mu.Lock()
	p.rushInFlight = false
	p.rushDashIssued = false
	p.rushDashAirborne = false
	p.mu.Unlock()
}

func (p *playerActor) rushInFlightSnapshot() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rushInFlight
}

func (p *playerActor) markRushDashIssued(airborne bool) {
	p.mu.Lock()
	p.rushDashIssued = true
	p.rushDashAirborne = airborne
	p.mu.Unlock()
}

// rushDashAirborneSnapshot 冲刺发出时刻的空中档标记（consume/clear 前取）。
func (p *playerActor) rushDashAirborneSnapshot() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rushDashAirborne
}

func (p *playerActor) rushDashIssuedSnapshot() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rushDashIssued
}

func (p *playerActor) setResumeSettleSkip(v bool) {
	p.mu.Lock()
	p.resumeSettleSkip = v
	p.mu.Unlock()
}

func (p *playerActor) resumeSettleSkipSnapshot() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.resumeSettleSkip
}

// mirrorJumpTimes 官方 JumpTimes 飞行计数镜像（技能4.pcapng 62 帧对 m15
// 迁移全解，2026-11-16）：起飞（地面→state6）授 1（挂三段跳能力 Buff
// buf_DHadd_3rd 时授 2）；空中动作消耗（已腾空后 state6 再上行 >0.3m=
// 二段跳，或腾空期 state7=空中水平冲刺/滑翔段）归 0；落地（state4/5）
// 静默复位。官服对每次飞行都发授予帧——客户端空中动作配额以该属性为
// 权威，我方不发则配额耗尽后永不复活=「穿云纵过去后卡状态、后续只有
// 原地垂直跳」根因（58540 会话 m15 铁证：3 次空中冲刺后进入 xz 恒定的
// 自动垂直弹跳循环）。
func (p *playerActor) mirrorJumpTimes(link sceneMessageConnection, state int32, y float32) {
	// 能力查询须在持锁前（hasActiveBuff 自取 p.mu）。
	hasTriple := p.hasActiveBuff(qgDHadd3rdBuffSlot, 1579, time.Now())
	p.mu.Lock()
	wasAir, consumed, lastY := p.airFlightActive, p.airFlightConsumed, p.airFlightLastY
	var grant int32  // >0=授予值
	var consume bool // 归零
	switch {
	case state == 6 && !wasAir:
		p.airFlightActive = true
		p.airFlightConsumed = false
		p.airFlightLastY = y
		grant = 1
		if hasTriple {
			grant = 2
		}
	case state == 6 && wasAir:
		if !consumed && y > lastY+0.3 {
			consume = true
			p.airFlightConsumed = true
		}
		if y > lastY {
			p.airFlightLastY = y
		}
	case state == 7 && wasAir && !consumed:
		// 空中水平段（穿云纵冲刺/滑翔报告 state7，73.184 JT=0 铁证）。
		consume = true
		p.airFlightConsumed = true
	case state == 4 || state == 5:
		p.airFlightActive = false
		p.airFlightConsumed = false
		p.airFlightLastY = y
	}
	p.mu.Unlock()
	if grant == 0 && !consume {
		return
	}
	value := grant
	if consume {
		value = 0
	}
	if frame, err := objectSingleProperty(playerObjectID, playerOwnerID, propJumpTimesIndex, "JumpTimes", clientdata.Int32Value(value)); err == nil {
		_ = link.WriteFrame(frame)
	}
}

// airFlightActiveSnapshot 客户端自身飞行是否进行中（state6 起飞→state4/5
// 落地）。逃逸门拖动与客户端自己的轻功冲刺并发会打断其移动状态机
//（f.pcapng 逆向 motion_func 白名单拒冲突 intent；2026-11-16 真机定版
// 「穿云纵过程中按冲锋技能→后续穿云纵卡死」=服务端 0x21 拖动插进客户端
// 自身空中冲刺所致）——冲锋驱动须等其落地。
func (p *playerActor) airFlightActiveSnapshot() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.airFlightActive
}

// noteClientReport 记录客户端最近位移上报（m15/220 均走此），供冲锋落地哨
// 判定 c61 后客户端是否自行移动（c61 零消费实证 2026-11-15）。
func (p *playerActor) noteClientReport(x, y, z float32, at time.Time) {
	p.mu.Lock()
	p.clientReportX, p.clientReportY, p.clientReportZ, p.clientReportAt = x, y, z, at
	p.mu.Unlock()
}

// clientMovedSince c61 发出后客户端是否自行位移（>0.5m 的新上报）。
func (p *playerActor) clientMovedSince(issuedAt time.Time, fromX, fromZ float32) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.clientReportAt.IsZero() || p.clientReportAt.Before(issuedAt) {
		return false
	}
	return math.Sqrt(float64(square(p.clientReportX-fromX)+square(p.clientReportZ-fromZ))) > 0.5
}

// moveSpeedSnapshot 角色当前移动速度（motion.moveSpeed=官方档案值 3.9，
// GM set_move_speed 可调）——拖动驱动与落地哨 ETA 的速度基准（v10 起取代
// 写死的 6.0，用户拍板「按当前角色速度跑」）。
func (p *playerActor) moveSpeedSnapshot() float32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.motion.moveSpeed <= 0 {
		return 3.9
	}
	return p.motion.moveSpeed
}

// qinggongPointSnapshot 轻功点余量快照（c61 空中授予簇尾 QingGongPoint
// 表现帧取值，v29）。
func (p *playerActor) qinggongPointSnapshot() int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.qinggong.Point
}

// clientReportSnapshot 客户端最近位移上报位置（走位驱动落地哨用）。
func (p *playerActor) clientReportSnapshot() (float32, float32, time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.clientReportX, p.clientReportZ, p.clientReportAt
}

// clientReportPositionSnapshot 客户端最近位移上报的三维位置（v31 呆坐
// 哨就地释放的锚点——绝不写 bind，防位置跳变）。
func (p *playerActor) clientReportPositionSnapshot() (float32, float32, float32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.clientReportX, p.clientReportY, p.clientReportZ
}

// rushPressGate 冲锋族按压入口闸：beginSkillUse 非 settled 分支的锁招/
// 个人 CD / 公共 GCD 三闸只读版。冲锋链在 211 入口提前分叉（直收 return /
// 暂存挂起），不补这道闸则四个结算点均无 CD 校验（commitRushDashCosts
// 只写不查）。MP/SP/轻功不在此查——仍按外服私服模型留在执行提交点。
func (p *playerActor) rushPressGate(definition combatSkillDefinition, now time.Time) (bool, string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.skillLockUntil.After(now) {
		return false, fmt.Sprintf("skill locked remaining=%s", p.skillLockUntil.Sub(now).Round(time.Millisecond))
	}
	if ready := p.skillCooldowns[definition.id]; ready.After(now) {
		return false, fmt.Sprintf("personal cooldown remaining=%s", ready.Sub(now).Round(time.Millisecond))
	}
	if p.publicCDUntil.After(now) && definition.taoLu != p.publicCDTaolu {
		return false, fmt.Sprintf("public cooldown (taolu switch from=%s) remaining=%s",
			p.publicCDTaolu, p.publicCDUntil.Sub(now).Round(time.Millisecond))
	}
	return true, ""
}

// commitRushDashCosts 在执行提交点（c61 / 直收簇 / 步行落地簇）结算
// CD/MP/SP 并做出招簿记；includeQingGong 仅冲刺路径为真（近距直收与步行
// 落地外服私服均不扣轻功点——g 直收簇无 QingGongPoint 帧、b 步行落地同）。
func (p *playerActor) commitRushDashCosts(pending *pendingCastState, now time.Time, includeQingGong bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	state := p.snapshotLocked()
	if state.HP <= 0 {
		return fmt.Errorf("player is dead")
	}
	if state.MP < pending.mpCost {
		return fmt.Errorf("MP %d < %d", state.MP, pending.mpCost)
	}
	if p.progress.attributes.sp < pending.spCost {
		return fmt.Errorf("SP %d < %d", p.progress.attributes.sp, pending.spCost)
	}
	if includeQingGong && pending.qgCost > 0 {
		// 轻功点不足不硬闸（65533 会话实证：QG<10 时整条冲锋链被拒）——
		// 尽力扣，扣不动跳过。外服私服 c61 簇的 QG 扣是表现帧，无不足拒绝样本。
		_, _ = p.qinggong.Use(pending.qgCost, now)
	}
	if !p.vitals.SpendMP(pending.mpCost) {
		return fmt.Errorf("MP spend rejected")
	}
	p.progress.attributes.sp -= pending.spCost
	p.skillCooldowns[pending.skillID] = now.Add(pending.personalCD)
	p.publicCDUntil = now.Add(pending.publicCD)
	p.publicCDTaolu = pending.taoLu
	p.currentSkillID = pending.skillID
	p.currentSkillEffectID = pending.skillID
	p.currentSkillTarget = uint64(pending.targetID) | uint64(pending.targetOwner)<<32
	p.currentSkillEpoch++
	// 直收路径的 pending 是本地构造（未入暂存槽），仅同实例时回写标记。
	if p.pendingCast == pending {
		p.pendingCast.settled = true
	}
	return nil
}

// ---- 帧构造 ----

func skillPauseFrame(windowMs int32) ([]byte, error) {
	return serverCustomIntMessage(25, customInt(windowMs), customString("skill_pause"))
}

func skillPauseClearFrame() ([]byte, error) {
	return serverCustomIntMessage(26)
}

// ---- m25/m26 配对看门狗 ----

// skillPauseWatchdogDelay 选靶路径 m25 的 m26 兜底延迟：官服 m25 从不
// 孤悬（a-k.pcapng 全扫 0.4~4.6s 必配对 m26），取官方上界外一拍——
// 选靶 m25 是客户端自动接近的武装前提不能不发，也不宜掐太早。
const skillPauseWatchdogDelay = 5 * time.Second

// skillPauseExpiryGrace 冲锋暂存过期后的补发宽限：受理链自身的 m26
// （落地/取消/再受理）会先到，看门狗只兜「过期无人收尾」的尾巴。
const skillPauseExpiryGrace = time.Second

// armSkillPauseRelease 在每次发出 msg25（skill_pause 诅咒）后武装 m26
// 兜底看门狗。m25=custom_begin_curse 锁客户端全部输入且客户端不自愈
// （30s 窗）；本服选靶/冲锋受理两路裸发曾致「连选目标后格挡/出招全哑」
// （2026-11-12 定案，兜底当日按用户要求撤回；2026-11-15 真机复发：58929
// 会话 m25×29/m26×17、12 枚孤悬最长约 2300 帧的输入锁窗）。到点仍有
// 未过期暂存冲锋则改守其过期点+宽限；重复 m26 无害（官服重选即同毫秒
// m26+m25 背靠背连发）。
func (p *playerActor) armSkillPauseRelease(link sceneMessageConnection, remote string) {
	delay := skillPauseWatchdogDelay
	if pending := p.peekPendingCast(time.Now().UTC()); pending != nil {
		if remain := time.Until(pending.expiresUTC.Add(skillPauseExpiryGrace)); remain > delay {
			delay = remain
		}
	}
	fire := func() { p.skillPauseReleaseTick(link, remote) }
	p.mu.Lock()
	if p.skillPauseTimer != nil {
		p.skillPauseTimer.Stop()
	}
	p.skillPauseTimer = time.AfterFunc(delay, fire)
	p.mu.Unlock()
}

// skillPauseReleaseTick 看门狗到点：无暂存=裸补 m26（选靶路径，或已被
// 受理链 m26 收尾后的冗余帧）；暂存仍在窗内=改守过期点；过期未清=按
// 取消方言补 m26+PauseSkill 清（resumePendingCastAt 对过期暂存零回应、
// peekPendingCast 不清槽，无人收尾时的唯一出口）。
func (p *playerActor) skillPauseReleaseTick(link sceneMessageConnection, remote string) {
	now := time.Now().UTC()
	p.mu.Lock()
	pending := p.pendingCast
	p.skillPauseTimer = nil
	p.mu.Unlock()
	if pending != nil && now.Before(pending.expiresUTC) {
		// 暂存冲锋仍在窗内：其自身链路的 m26 会先到，改守过期点。
		p.mu.Lock()
		p.skillPauseTimer = time.AfterFunc(
			time.Until(pending.expiresUTC.Add(skillPauseExpiryGrace)),
			func() { p.skillPauseReleaseTick(link, remote) })
		p.mu.Unlock()
		return
	}
	frame, err := skillPauseClearFrame()
	if err != nil {
		log.Printf("%s: encode deferred msg26: %v", remote, err)
		return
	}
	if err := link.WriteFrame(frame); err != nil {
		log.Printf("%s: write deferred msg26: %v", remote, err)
		return
	}
	if pending != nil {
		if clear, cerr := rushPropertyFrame(propRushPauseSkill, "PauseSkill", clientdata.StringValue("")); cerr == nil {
			if werr := link.WriteFrame(clear); werr != nil {
				log.Printf("%s: write deferred PauseSkill clear: %v", remote, werr)
			}
		}
		// v31 补（2026-11-17 用户令核对轻功复位）：空中档已授予但 212
		// 未到的过期收尾——轻功绑定、授予对（CantMove/CantParry=1）与
		// rush 在途标记一并还原，否则轻功状态残留（「穿云纵后续有概率
		// 用不出」复发）+ 玩家被 CantMove 冻住。
		if p.rushDashIssuedSnapshot() {
			if p.rushDashAirborneSnapshot() {
				qx, qy, qz := p.clientReportPositionSnapshot()
				writeQinggongReleaseFrame(link, qx, qy, qz)
			}
			if clear, cerr := rushPropertyFrame(propRushCantMove, "CantMove", clientdata.ByteValue(0)); cerr == nil {
				_ = link.WriteFrame(clear)
			}
			if clear, cerr := rushPropertyFrame(propCantParry, "CantParry", clientdata.Int32Value(0)); cerr == nil {
				_ = link.WriteFrame(clear)
			}
			p.clearRushInFlight()
		}
		// 过期暂存正式出清（原实现只发帧不清槽，残留 pending 会被下一次
		// staging 覆盖但窗口语义上应即时出清）。
		p.consumePendingCast(time.Now().UTC())
		log.Printf("%s: deferred msg26 + PauseSkill clear for expired staged skill id=%s", remote, pending.skillID)
		return
	}
	log.Printf("%s: deferred msg26 (skill_pause pairing watchdog)", remote)
}

func casterTransformSyncFrame(x, y, z, orient float32) ([]byte, error) {
	return serverCustomIntMessage(337,
		customObject(playerObjectID, playerOwnerID),
		customFloat32(x), customFloat32(y), customFloat32(z),
		customFloat32(orient),
		customInt(0),
	)
}

func dashRushFrame(x, y, z float32, level int32, action string) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(playerObjectID, playerOwnerID),
		customInt(msgRushCommand),
		customFloat32(x), customFloat32(y), customFloat32(z),
		customInt(level),
		customString(action),
	)
}

func curSkillIDFrame(skillID string) ([]byte, error) {
	return objectSingleProperty(playerObjectID, playerOwnerID, propCurSkillID, "CurSkillID", clientdata.StringValue(skillID))
}

// objectSingleProperty 任意对象的 n=1 单属性 0x10 帧（外服私服簇逐帧单属性，
// 绝不整帧重推——坐姿/冲锋态下整帧重推会重入动画）。
func objectSingleProperty(objectID, ownerID uint32, index uint16, name string, value clientdata.Value) ([]byte, error) {
	return sceneObjectProperties(objectID, ownerID, 0, []clientdata.IndexedProperty{
		{Index: index, Name: name, Value: value},
	})
}

// rushPropertyFrame 保留旧名：玩家对象的 n=1 单属性帧。
func rushPropertyFrame(index uint16, name string, value clientdata.Value) ([]byte, error) {
	return objectSingleProperty(playerObjectID, playerOwnerID, index, name, value)
}

// rushBindPoint 目标朝玩家方向 BindDistance 处的冲锋终点。
func rushBindPoint(target worldcore.Transform, x, z float32, bind float32) (float32, float32, float32) {
	return rushApproachPoint(target, x, z, bind)
}

// rushApproachPoint 目标朝玩家方向 dist 米处的接近停点（v11：6 米技停在
// 射程处出招、3 米技停在射程后补冲刺；y=target.Y 地面口径）。
func rushApproachPoint(target worldcore.Transform, x, z, dist float32) (float32, float32, float32) {
	dx := x - target.X
	dz := z - target.Z
	horiz := math.Sqrt(float64(dx*dx + dz*dz))
	if horiz < 0.01 {
		dx = 1
		horiz = 1
	}
	return target.X + float32(float64(dx)/horiz)*dist, target.Y, target.Z + float32(float64(dz)/horiz)*dist
}

// orientTowardTarget 外服私服朝向公式：atan2(dx, dz) 指向目标（msg337/msg19
// 共用；f 3.758 / g 2.84/2.414 / b 1.638 四样本逐位验证）。
func orientTowardTarget(x, z, targetX, targetZ float32) float32 {
	return float32(math.Atan2(float64(targetX-x), float64(targetZ-z)))
}

func rushTargetHitPrecastFrame(targetID, targetOwner uint32) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(targetID, targetOwner),
		customInt(40),
		customInt(1),
		customObject(playerObjectID, playerOwnerID),
	)
}

// segmentDamageOpenFrame 即 msg19 段伤害起始帧（外服私服 14 值）：
//
//	[19, obj(攻), obj(目), str 技能, i32 v, i32 v,
//	 f32 打击点1 x,y,z, f32 朝向, f32 -2.5, f32 打击点2 x,y,z]
//
// 打击点=玩家沿「目标→玩家」延长线、距目标恒 4.04/4.55m（外服私服拟合常数）。
func segmentDamageOpenFrame(targetID, targetOwner uint32, skillID string, value int32,
	x, y, z, orient float32, distance float64) ([]byte, error) {
	sin, cos := math.Sin(float64(orient)), math.Cos(float64(orient))
	// 玩家→目标方向单位向量 = (sin, cos)；反向延伸 (K-distance) 米。
	point := func(k float64) (float32, float32) {
		r := k - distance
		if r < 0 {
			r = 0
		}
		return x - float32(sin*r), z - float32(cos*r)
	}
	x1, z1 := point(msg19StrikePointFar1)
	x2, z2 := point(msg19StrikePointFar2)
	return serverCustomIntMessage(19,
		customObject(playerObjectID, playerOwnerID),
		customObject(targetID, targetOwner),
		customString(skillID),
		customInt(value),
		customInt(value),
		customFloat32(x1), customFloat32(y), customFloat32(z1),
		customFloat32(orient),
		customFloat32(msg19PitchConstant),
		customFloat32(x2), customFloat32(y), customFloat32(z2),
	)
}

// ---- sceneLifecycle 战斗 NPC 查询 ----

func (s *sceneLifecycle) combatNPCAlive(id, owner uint32) bool {
	if s == nil {
		return false
	}
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	entity := worldcore.EntityID(id)
	vitals, hasVitals := s.run.combat.Vitals(entity)
	if !hasVitals || vitals.HP <= 0 {
		return false
	}
	meta, exists := s.npcMeta(entity)
	return exists && meta.ownerID == owner && meta.interaction == npcInteractionCombat
}

func (s *sceneLifecycle) combatNPCTransform(id uint32) (worldcore.Transform, bool) {
	if s == nil {
		return worldcore.Transform{}, false
	}
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	entity := worldcore.EntityID(id)
	transform, exists := s.npcTransform(entity)
	vitals, hasVitals := s.run.combat.Vitals(entity)
	return transform, exists && hasVitals && vitals.HP > 0
}

// combatNPCModel 返回存活战斗 NPC 的模型窄串（直收簇 LastWGAttackNpc 与
// msg24 战斗帧共用；npcMeta.model 即 Material 列）。
func (s *sceneLifecycle) combatNPCModel(id uint32) (string, bool) {
	if s == nil {
		return "", false
	}
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	meta, exists := s.npcMeta(worldcore.EntityID(id))
	if !exists || meta.interaction != npcInteractionCombat {
		return "", false
	}
	return meta.model, true
}

func rushCastStateFrame() ([]byte, error) {
	return serverCustomIntMessage(707, customInt(15))
}

// rushAirGrantFollowFrame 空中 c61 授予后的 m707[29] 随帧（技能4.pcapng：
// 3/3 空中授予全带、地面授予零样本；官方时点=授予后 +40~80ms）。
func rushAirGrantFollowFrame() ([]byte, error) {
	return serverCustomIntMessage(707, customInt(29))
}

// ---- 211 暂存受理（桶②③④） ----

// stageRushCast 冲锋暂存受理（外服私服序，g/f 实证）：
//
//	首发：[0x12 清扫过期 CD 行] + CurSkillID + CurSkillTarget + msg25 +
//	      PauseSkill + msg707[15]
//	重受理：CurSkillID="" + msg26 + PauseSkill="" + [0x12 清扫] +
//	      CurSkillID + msg25 + PauseSkill + msg707[15]（不重发 CurSkillTarget）
//
// 受理与距离解耦；拉回/走动期间服务端零帧零取消，唯一出口=重按或 30s 过期
// （2026-11-15 起受理尾追加服务端驱动接近 c61，见函数尾注释）。
func stageRushCast(link sceneMessageConnection, player *playerActor, world *sceneLifecycle, definition combatSkillDefinition,
	targetID, targetOwner uint32, target worldcore.Transform, castAt worldcore.Transform,
	distance float64, remote string) (bool, error) {
	now := time.Now().UTC()
	// v29：空中判定启用（旧 1e9 禁用线撤销——其根因是 NPC 服务端 y 口径
	// 时代，而冲锋族 geometry 已用 211 自报目标坐标，与 castAt 同为客户端
	// 坐标系；官服观察「空中奔袭凭空飞过不落地、速度 15」即此分支）。
	// v30：低→高 >0.5m 亦空中档（rushChargeAirborne 统一判定）。
	airborne := rushChargeAirborne(castAt.Y, target.Y)
	rearm := player.peekPendingCast(now) != nil
	// 冲刺在途标记必须在 stagePendingCast 复位前捕获（旧代码复位后再读
	// 恒 false，换技能还原分支实际不可达——v29 修正）。
	rearmDash := rearm && player.rushDashIssuedSnapshot()
	rearmDashAirborne := rearmDash && player.rushDashAirborneSnapshot()
	dialect := rushDashDialectFor(definition)
	dashLevel := dialect.level
	if dashLevel == 0 {
		dashLevel = convertDashLevel
	}
	dashAction := dialect.action
	if dashAction == "" {
		dashAction = definition.groundRushAction
	}
	// v29 武技动作覆盖（lua rush.action/air_action；level/带已在 dialect 合并）。
	groundAction, airAction := definition.groundRushAction, definition.airRushAction
	if o := definition.rushOverride; o != nil {
		if o.groundAction != "" {
			groundAction = o.groundAction
			if dashAction == definition.groundRushAction {
				dashAction = o.groundAction
			}
		}
		if o.airAction != "" {
			airAction = o.airAction
		}
	}
	player.stagePendingCast(&pendingCastState{
		skillID: definition.id, targetID: targetID, targetOwner: targetOwner,
		expiresUTC: now.Add(rushPendingWindow),
		canRush:    definition.canRush, castRange: definition.castRange,
		bindDistance: definition.bindDistance, groundRushAction: groundAction,
		airRushAction: airAction,
		airborne:      airborne,
		geometry:      target,
		nominalDamage: definition.baseDamage,
		mpCost:        definition.mpCost, spCost: definition.spCost, qgCost: definition.qgCost,
		cooldownCategory: definition.cooldownCategory, cooldownTeam: definition.cooldownTeam,
		personalCD: definition.personalCD, publicCD: definition.publicCD,
		taoLu:        definition.taoLu,
		dashWindowLo: dialect.windowLo, dashWindowHi: dialect.windowHi,
		dashLevel: dashLevel, dashAction: dashAction,
	})
	type frameBuilder struct {
		name  string
		build func() ([]byte, error)
	}
	var builders []frameBuilder
	if rearmDash {
		// 冲刺在途重按换技能（rearm）：旧 dash 的 ETA AfterFunc 会因 pending
		// 被替换而早退——逃逸门 106、轻功绑定与 c61 授予对（CantMove/
		// CantParry，v29 起授予簇置 1）在此出口统一还原
		//（2026-11-16「穿云纵后续有概率用不出」同类出口补全）。
		if rearmDashAirborne {
			writeQinggongReleaseFrame(link, castAt.X, castAt.Y, castAt.Z)
		}
		if restore, err := objectSingleProperty(playerObjectID, playerOwnerID, propRushLogicState, "LogicState", clientdata.ByteValue(0)); err == nil {
			_ = link.WriteFrame(restore)
		}
		if clear, err := rushPropertyFrame(propRushCantMove, "CantMove", clientdata.ByteValue(0)); err == nil {
			_ = link.WriteFrame(clear)
		}
		if clear, err := rushPropertyFrame(propCantParry, "CantParry", clientdata.Int32Value(0)); err == nil {
			_ = link.WriteFrame(clear)
		}
	}
	if rearm {
		builders = append(builders,
			frameBuilder{"CurSkillID-clear-rearm", func() ([]byte, error) { return curSkillIDFrame("") }},
			frameBuilder{"msg26-rearm", func() ([]byte, error) { return skillPauseClearFrame() }},
			frameBuilder{"PauseSkill-clear-rearm", func() ([]byte, error) {
				return rushPropertyFrame(propRushPauseSkill, "PauseSkill", clientdata.StringValue(""))
			}},
		)
	}
	// 惰性清扫过期 CD 行（外服私服在受理簇头发 0x12，f 7.764/15.649 实证）。
	for _, sweep := range player.sweepExpiredCooldownRows(now) {
		builders = append(builders, frameBuilder{"cd-sweep", func() ([]byte, error) { return sweep, nil }})
	}
	// 首受理（非 rearm）官方方言（技能.pcapng 32.642 定版逐帧，2026-11-12
	// 复核勘误）：**CurSkillID(462)=staged 技能串 → CurSkillTarget(629) →
	// msg25(25,30000,'skill_pause') → PauseSkill(220)=技能 → msg707[15] →
	// 四连**。旧结论「首受理不发 CurSkillID」系对 f#0194 的误读（其 idx462
	// 实为 'CS_jh_hyjf02\0'=staged 技能）；当年「多发致施法中不自走」的
	// 对照实验缺的是走踢帧，CurSkillID 被错怪。
	// PauseSkill 首受理必发（技能1.pcapng 登录锚定 5/5 样本：141.199/
	// 148.530/157.764/181.869/793.617 全在 m25 后带 PauseSkill=技能；
	// 技能.pcapng 32.642「首受理不发」系中途开抓、实为 rearm 形态的误判）。
	if !rearm {
		builders = append(builders,
			frameBuilder{"CurSkillID", func() ([]byte, error) { return curSkillIDFrame(definition.id) }},
			frameBuilder{"CurSkillTarget", func() ([]byte, error) {
				return rushPropertyFrame(propCurSkillTarget, "CurSkillTarget", clientdata.ObjectValue(targetID, targetOwner))
			}})
	} else {
		builders = append(builders, frameBuilder{"CurSkillID", func() ([]byte, error) { return curSkillIDFrame(definition.id) }})
	}
	// v31 空中受理标记（技能4 20.044 首受理/技能5 44.029 rearm 双证）：
	// CurIsAirSkill=4 紧随 CurSkillID——空中暂存的官方受理簇成员。
	if airborne {
		builders = append(builders, frameBuilder{"CurIsAirSkill", func() ([]byte, error) {
			return rushPropertyFrame(propCurIsAirSkill, "CurIsAirSkill", clientdata.Int32Value(4))
		}})
	}
	// msg25[30000,'skill_pause'] + PauseSkill=技能成对（技能1.pcapng 5/5
	// 首受理样本皆有，m25 之后、m707 之前）——远距按压的官方暂存 UX。
	// **stage 期不发 m24 c61**（规格 §三：受理簇零 m24，c61 只在进带时由
	// tryRushArrival 下发）。
	builders = append(builders,
		frameBuilder{"msg25", func() ([]byte, error) { return skillPauseFrame(int32(rushPendingWindow / time.Millisecond)) }},
	)
	// PauseSkill=技能：首受理与 rearm 均发（技能1.pcapng 5/5 首受理样本
	// 皆有，m25 之后、m707 之前——技能暂存状态机 msg25+PauseSkill 成对）。
	builders = append(builders, frameBuilder{"PauseSkill-set", func() ([]byte, error) {
		return rushPropertyFrame(propRushPauseSkill, "PauseSkill", clientdata.StringValue(definition.id))
	}})
	builders = append(builders, frameBuilder{"msg707", func() ([]byte, error) { return rushCastStateFrame() }})
	// 状态四连（586/645/646/750）：恢复。此前「暂撤」的对照推理（55052
	// 无四连会走 / 64785 有四连不自走）因果倒置——那批不自走会话处于
	// 受理簇方言错位期。技能.pcapng 外服私服三证（32.642/415.577/1290.300）
	// 钉死：外服私服每次受理簇后 +42ms 必发四连，且客户端同毫秒起跑——
	// 四连与自走共存，是簇的组成部分而非干扰。maybeRushStatePing 的
	// 在途 3s 周期重发仍停用（最小变量，先验簇尾四连）。
	for _, ping := range officialStatePingFrames() {
		frame := ping
		if frame == nil {
			continue
		}
		builders = append(builders, frameBuilder{"state-ping", func() ([]byte, error) { return frame, nil }})
	}
	frames := make([][]byte, 0, len(builders))
	for _, builder := range builders {
		frame, err := builder.build()
		if err != nil {
			return true, fmt.Errorf("encode rush %s frame for %s: %w", builder.name, definition.id, err)
		}
		frames = append(frames, frame)
	}
	player.markRushInFlight()
	player.noteRushStatePing(now)
	for _, frame := range frames {
		if err := link.WriteFrame(frame); err != nil {
			return true, err
		}
	}
	log.Printf("%s: rush staged skill id=%s dist=%.1fm air=%t rearm=%t window=%s dest=(%.1f,%.1f,%.1f)",
		remote, definition.id, distance, airborne, rearm, rushPendingWindow, target.X, target.Y, target.Z)
	// 受理簇内的 msg25 同样要兜底配对（规格 §八：官服 m25 从不孤悬；落地/
	// 取消/再受理的 m26 先到时看门狗只会多补一发无害的重复 m26）。
	player.armSkillPauseRelease(link, remote)
	// 空/地混层闸（规格 §二）：直收带内但 |出招位 y−目标 y|>1m——原地普攻
	// 会隔空打高处/低处目标，受理后立即授予冲锋；客户端自身飞行中扣住
	//（airFlightActive，落地后行走上报重新触发）。
	if distance <= rushDirectCastMeters(definition) &&
		math.Abs(float64(castAt.Y-target.Y)) > float64(rushChargeHeightGateMeters) &&
		!player.airFlightActiveSnapshot() {
		rushDashDrive(link, player, world, target, castAt.X, castAt.Y, castAt.Z, remote)
		return true, nil
	}
	// 地面冲锋带按压即授（规格 §二/§三；2026-11-17 真机「地面带无效果」
	// 修复）：同层且按压 2D 距离已在授予窗 (lo, hi] 内（3m 族 (直收带,
	// GRushRange 5]、方言表窗、武技扩程窗同构）——进带判定以按压 211 自带
	// 位置即刻完成。本服登录方言下客户端无「按下即走」，站立按压零
	// m15/m220 上报，单等上报=带死；官服「3~5m 立即冲锋」/落尘「受理即冲」
	// 均此形态（技能4 #5：受理簇→c61，m220 只是同义的接近申报）。轻功在途
	// 同款扣住；空中档（混层）不在此列——空中授予官方为 m220 驱动（技能5）。
	if !airborne && dialect.windowLo > 0 && dialect.windowHi > 0 &&
		distance > dialect.windowLo && distance <= dialect.windowHi &&
		!player.airFlightActiveSnapshot() {
		rushDashDrive(link, player, world, target, castAt.X, castAt.Y, castAt.Z, remote)
	}
	return true, nil
}

// rushDriveDashSpeedMeters 地面档档速（物理档案 LandRushSpeed=25；规格
// §十一——ETA/呆坐哨/同步条目 s5 共用）。
const rushDriveDashSpeedMeters = float32(25.0)

// rushDriveAirDashSpeedMeters 空中档档速（物理档案 AirRushSpeed=15；同技
// 空中耗时均值 599ms vs 地面 449ms——垂直分量由 fastfall 主导）。
const rushDriveAirDashSpeedMeters = float32(15.0)

// normalizeOrient 朝向归一 [0,2π)（官方 s4 实测 0.13~5.94）。
func normalizeOrient(o float32) float32 {
	for o < 0 {
		o += 2 * float32(math.Pi)
	}
	for o >= 2 * float32(math.Pi) {
		o -= 2 * float32(math.Pi)
	}
	return o
}

// noteRushStatePing 记录状态四连下发时刻（受理/直收簇内联发送后调用）。
func (p *playerActor) noteRushStatePing(now time.Time) {
	p.mu.Lock()
	p.rushStatePingAt = now
	p.mu.Unlock()
}

// ---- 直收执行簇（桶①） ----

// executeRushDirect 近距直收执行（外服私服 g 定版，无暂存链）：
// [0x12 清扫] → CurSkillID → CurSkillTarget → msg284 → NPC.LogicState=1 →
// fight_relation 行0 → 玩家.LogicState=1 → CD 双行 → MPRatio → MP → 0x1F →
// msg337 → LastWGAttackNpc → c1=40 → msg19 → CantMove=1 → CantParry=1。
// CD/MP 此刻结、不扣轻功、无 active_skill 行/msg20/CanUse。
func executeRushDirect(link sceneMessageConnection, player *playerActor, world *sceneLifecycle,
	definition combatSkillDefinition, targetID, targetOwner uint32, geometry worldcore.Transform,
	castAt worldcore.Transform, distance float64, remote string) (bool, error) {
	now := time.Now().UTC()
	pending := &pendingCastState{
		skillID: definition.id, targetID: targetID, targetOwner: targetOwner,
		canRush: definition.canRush, castRange: definition.castRange,
		bindDistance: definition.bindDistance, groundRushAction: definition.groundRushAction,
		airRushAction: definition.airRushAction,
		geometry:      geometry, nominalDamage: definition.baseDamage,
		mpCost: definition.mpCost, spCost: definition.spCost, qgCost: definition.qgCost,
		cooldownCategory: definition.cooldownCategory, cooldownTeam: definition.cooldownTeam,
		personalCD: definition.personalCD, publicCD: definition.publicCD,
		taoLu: definition.taoLu, expiresUTC: now.Add(rushPendingWindow),
	}
	if err := player.commitRushDashCosts(pending, now, false); err != nil {
		log.Printf("%s: rush direct settle failed skill id=%s: %v", remote, definition.id, err)
		return true, nil
	}
	model, _ := world.combatNPCModel(targetID)
	orient := orientTowardTarget(castAt.X, castAt.Z, geometry.X, geometry.Z)
	state := player.stateSnapshot()
	mpRatio := int32(0)
	if state.MaxMP > 0 {
		mpRatio = state.MP * 100 / state.MaxMP
	}
	// 外服私服直收簇逐帧序（g 22.157 定版）：[0x12 清扫] → CurSkillID →
	// CurSkillTarget → msg284 → NPC.LogicState → fight_relation →
	// 玩家.LogicState → CD 双行 → MPRatio → MP → 0x1F → msg337 →
	// LastWGAttackNpc → c1=40 → msg19 → CantMove=1 → CantParry=1。
	step := func(name string, build func() ([]byte, error)) error {
		frame, err := build()
		if err != nil {
			return fmt.Errorf("encode rush direct %s for %s: %w", name, definition.id, err)
		}
		return link.WriteFrame(frame)
	}
	for _, sweep := range player.sweepExpiredCooldownRows(now) {
		if err := step("cd-sweep", func() ([]byte, error) { return sweep, nil }); err != nil {
			return true, err
		}
	}
	prelude := []struct {
		name  string
		build func() ([]byte, error)
	}{
		{"CurSkillID", func() ([]byte, error) { return curSkillIDFrame(definition.id) }},
		{"CurSkillTarget", func() ([]byte, error) {
			return rushPropertyFrame(propCurSkillTarget, "CurSkillTarget", clientdata.ObjectValue(targetID, targetOwner))
		}},
		{"msg284", func() ([]byte, error) { return rushCastMark284Frame() }},
		{"NPC-LogicState", func() ([]byte, error) {
			return objectSingleProperty(targetID, targetOwner, propRushLogicState, "LogicState", clientdata.ByteValue(1))
		}},
		{"fight-relation", func() ([]byte, error) { return fightRelationRowAddFrame(targetID, targetOwner), nil }},
		{"LogicState", func() ([]byte, error) {
			return rushPropertyFrame(propRushLogicState, "LogicState", clientdata.ByteValue(1))
		}},
	}
	for _, s := range prelude {
		if err := step(s.name, s.build); err != nil {
			return true, err
		}
	}
	if err := player.writeOfficialCooldownRows(link, definition.cooldownCategory, definition.cooldownTeam,
		definition.personalCD, definition.publicCD, now); err != nil {
		return true, fmt.Errorf("write rush direct cooldown rows %s: %w", definition.id, err)
	}
	tail := []struct {
		name  string
		build func() ([]byte, error)
	}{
		{"MPRatio", func() ([]byte, error) {
			return rushPropertyFrame(propMPRatio, "MPRatio", clientdata.Int32Value(mpRatio))
		}},
		{"MP", func() ([]byte, error) {
			return rushPropertyFrame(propMP, "MP", clientdata.Int32Value(state.MP))
		}},
		{"location-anchor", func() ([]byte, error) {
			return serverLocation(playerObjectID, playerOwnerID, worldcore.Transform{X: castAt.X, Y: castAt.Y, Z: castAt.Z}), nil
		}},
		{"msg337", func() ([]byte, error) { return casterTransformSyncFrame(castAt.X, castAt.Y, castAt.Z, orient) }},
	}
	if model != "" {
		tail = append(tail, struct {
			name  string
			build func() ([]byte, error)
		}{"LastWGAttackNpc", func() ([]byte, error) {
			return rushPropertyFrame(propRushLastWGAttackNpc, "LastWGAttackNpc", clientdata.StringValue(model))
		}})
	}
	tail = append(tail,
		struct {
			name  string
			build func() ([]byte, error)
		}{"c1=40-precast", func() ([]byte, error) { return rushTargetHitPrecastFrame(targetID, targetOwner) }},
		struct {
			name  string
			build func() ([]byte, error)
		}{"msg19-open", func() ([]byte, error) {
			return segmentDamageOpenFrame(targetID, targetOwner, definition.id, definition.baseDamage,
				castAt.X, castAt.Y, castAt.Z, orient, distance)
		}},
		struct {
			name  string
			build func() ([]byte, error)
		}{"CantMove-set", func() ([]byte, error) {
			return rushPropertyFrame(propRushCantMove, "CantMove", clientdata.ByteValue(1))
		}},
		struct {
			name  string
			build func() ([]byte, error)
		}{"CantParry-set", func() ([]byte, error) {
			return rushPropertyFrame(propCantParry, "CantParry", clientdata.Int32Value(1))
		}},
	)
	for _, s := range tail {
		if err := step(s.name, s.build); err != nil {
			return true, err
		}
	}
	// 状态四连（外服私服 367.971 直收簇 → 368.020 +49ms 四连实证；与受理簇
	// 尾四连同源，恢复理由见 stageRushCast 注释）。
	for _, ping := range officialStatePingFrames() {
		if ping == nil {
			continue
		}
		if err := link.WriteFrame(ping); err != nil {
			return true, err
		}
	}
	player.noteRushStatePing(now)
	// 出招簿记 + 伤害链（settled：CD/MP 已在簇内结算）
	target := uint64(targetID) | uint64(targetOwner)<<32
	if ok, reason := player.beginSkillUse(definition, target, now, true); !ok {
		log.Printf("%s: rush direct begin skill use rejected id=%s: %s", remote, definition.id, reason)
		return true, nil
	}
	castEpoch := player.currentSkillEpochSnapshot()
	launchSkillDamagePhase(link, player, world, definition, clientCustomMessage{}, targetID, targetOwner, castEpoch, remote)
	log.Printf("%s: rush direct execute skill id=%s dist=%.1fm orient=%.3f", remote, definition.id, distance, orient)
	return true, nil
}

// ---- c61 冲刺簇（桶③） ----

// ---- 212 落地 ----

// handleCastDashLanding 外服私服 212 方言：
// [212, x, y, z, 朝向, 尾值]——尾 1=冲刺落地 / 0=步行落地（b 51.348 实证），
// 两者都是落地信号（距目标 ≤3m 即落地）。无暂存 212 外服私服零回应（静默）。
// 212 三分（远距未冲=就地执行请求→发 c61；远距已冲=安全释放）保留为
// 静默兜底——仅客户端偏离外服私服方言时触发。
func handleCastDashLanding(link sceneMessageConnection, player *playerActor, world *sceneLifecycle,
	custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomCastDashLanding {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject dash landing before player spawn", remote)
		return true, nil
	}
	if len(custom.Values) < 5 || custom.Values[1].Type != 4 || custom.Values[2].Type != 4 || custom.Values[3].Type != 4 {
		log.Printf("%s: reject dash landing malformed values=%d", remote, len(custom.Values))
		return true, nil
	}
	orient := float32(0)
	if len(custom.Values) > 4 && custom.Values[4].Type == 4 {
		orient = custom.Values[4].Float32
	}
	x, y, z := custom.Values[1].Float32, custom.Values[2].Float32, custom.Values[3].Float32
	if pending := player.peekPendingCast(time.Now().UTC()); pending != nil {
		// 212 距离基准 = 当前服务端 NPC 坐标（0x1F 节流广播使客户端认知
		// 与之对齐；pending.geometry 是按下时刻快照，NPC 追击后会漂移）。
		geometry := pending.geometry
		if target, ok := world.combatNPCTransform(pending.targetID); ok {
			geometry = target
		}
		if geometry.X != 0 || geometry.Z != 0 {
			distance := math.Sqrt(float64(square(x-geometry.X) + square(z-geometry.Z)))
			if distance > rushWalkBandAcceptMeters(pending) {
				// 技能4.pcapng #78 定版：m212=客户端「冲锋到位申报」而非
				// 「落地」——距 dest 平距 6.9m/垂 4.6m 的悬空 212 官服照发
				// m26+40ms 释放（27/27 释放=m212+39~78ms、零取消样本）。
				// 冲刺已发后的远带 212 一律按到位释放（空中/悬空出招是特性
				// 非 bug）；走位驱动在途的噪声 212 仍静默忽略。
				if player.rushDashIssuedSnapshot() {
					log.Printf("%s: dash landing beyond band with dash issued; released as arrival declaration skill id=%s dist=%.1fm",
						remote, pending.skillID, distance)
				}
				// 超走带的 212 无条件落地（Java b 线方言：handleDashLanding
				// 无距离闸直接 resumePendingCastAt）。54949 会话实证走带闸
				// 静默忽略致客户端狂发 212×278 无响应——客户端停点即其
				// 认定的射程内（其目标认知与服务端 NPC 实时位可差 5m+，
				// 按 211 自报目标坐标算距才与客户端同口径）。
				if !player.rushDashIssuedSnapshot() {
					log.Printf("%s: dash landing beyond walk band accepted skill id=%s dist=%.1fm",
						remote, pending.skillID, distance)
				}
			} else {
				// 走带内的 212[尾0/1] = 步行/冲刺落地（b 4.41m 走到 3.17m 落地
				// 实证；本服客户端 ~3.6m 即停连发 212[0]，63073 会话实证——
				// 认可半径 max(GRushRange, castRange+1)：hyjf02 型 6 米技能官方
				// 212 落在 5.5~5.8m 全认可，技能.pcapng 定版）。
				log.Printf("%s: dash landing in walk band skill id=%s dist=%.1fm dash=%t",
					remote, pending.skillID, distance, player.rushDashIssuedSnapshot())
			}
		}
		// 目标不可测距（自增益目标/world 未就绪）：维持落地续招。
		resumePendingCastAt(link, player, world, x, y, z, orient, "dash-landing", remote)
		return true, nil
	}
	// 外服私服对无暂存 212 零回应（b 51.4+ 连发 15 条 212[尾0] 全程静默）；
	// 位置收编仍按上报对账，不发帧。
	// v31 补（2026-11-17 用户令核对轻功复位）：c61 已授予（dashIssued
	// 残留）而 pending 已过期/被消费——迟到 212 是该轮冲锋的唯一收尾
	// 出口：还原轻功绑定与授予对（CantMove/CantParry=1）+ 复位 rush 标记
	//（官方对纯噪声 212 零回应，dashIssued 残留形态是异常收尾不是噪声）。
	if player.rushDashIssuedSnapshot() {
		if player.rushDashAirborneSnapshot() {
			writeQinggongReleaseFrame(link, x, y, z)
		}
		if clear, err := rushPropertyFrame(propRushCantMove, "CantMove", clientdata.ByteValue(0)); err == nil {
			_ = link.WriteFrame(clear)
		}
		if clear, err := rushPropertyFrame(propCantParry, "CantParry", clientdata.Int32Value(0)); err == nil {
			_ = link.WriteFrame(clear)
		}
		player.clearRushInFlight()
	}
	if world != nil && world.wireID != 0 {
		world.run.updateMemberPosition(world.wireID, role.Position{X: x, Y: y, Z: z, Orient: orient})
	}
	return true, nil
}

// cancelPendingCast 作废暂存出招（不结算不伤害）——兜底路径专用。
// c61 已发出时必须一并清 CantMove/CantParry（63073 会话实证：取消不清
// 状态对 → 客户端被冻原地、后续按下只刷 212 不走动）。
func cancelPendingCast(link sceneMessageConnection, player *playerActor,
	x, y, z, orient float32, remote string) {
	dashIssued := player.rushDashIssuedSnapshot()
	dashAirborne := player.rushDashAirborneSnapshot()
	pending := player.consumePendingCast(time.Now().UTC())
	player.clearRushInFlight()
	if pending != nil {
		log.Printf("%s: dash landing outside walk band; cancelled pending skill id=%s dash=%t", remote, pending.skillID, dashIssued)
	}
	if dashIssued {
		// dashIssued ⇒ 逃逸门 LogicState=106 已发——取消路径必须还原 0
		//（旧行只清 CantMove/CantParry，106 残留同样锁移动/轻功）；空中档
		// 一并释放轻功绑定（writeQinggongReleaseFrame，2026-11-16 统一出口）。
		if dashAirborne {
			writeQinggongReleaseFrame(link, x, y, z)
		}
		if restore, err := objectSingleProperty(playerObjectID, playerOwnerID, propRushLogicState, "LogicState", clientdata.ByteValue(0)); err == nil {
			_ = link.WriteFrame(restore)
		}
	}
	release := []struct {
		name  string
		build func() ([]byte, error)
	}{
		{"msg26", func() ([]byte, error) { return skillPauseClearFrame() }},
		{"PauseSkill-clear", func() ([]byte, error) {
			return rushPropertyFrame(propRushPauseSkill, "PauseSkill", clientdata.StringValue(""))
		}},
		{"msg337", func() ([]byte, error) { return casterTransformSyncFrame(x, y, z, orient) }},
	}
	if dashIssued {
		release = append([]struct {
			name  string
			build func() ([]byte, error)
		}{
			{"CantMove-clear", func() ([]byte, error) {
				return rushPropertyFrame(propRushCantMove, "CantMove", clientdata.ByteValue(0))
			}},
			{"CantParry-clear", func() ([]byte, error) {
				return rushPropertyFrame(propCantParry, "CantParry", clientdata.Int32Value(0))
			}},
		}, release...)
	}
	for _, step := range release {
		frame, err := step.build()
		if err != nil {
			log.Printf("%s: encode dash cancel %s: %v", remote, step.name, err)
			return
		}
		if err := link.WriteFrame(frame); err != nil {
			log.Printf("%s: write dash cancel %s: %v", remote, step.name, err)
			return
		}
	}
}

// resumePendingCastAt 落地续招本体（外服私服定版序，f 43.233 冲刺落地 /
// b 51.387 步行落地双形态）：
//
//	[冲刺] CantMove=0 → CantParry=0
//	PauseSkill="" → msg26 → NPC.LogicState=1 → fight_relation 行0 →
//	玩家.LogicState=1 → [步行且未结算：CD 双行+MP 对] → 0x1F（冲刺×2/步行×1）
//	→ msg337(朝向=atan2 指向目标) → c1=40 → msg19(14 值) →
//	CantMove=1 → CantParry=1
func resumePendingCastAt(link sceneMessageConnection, player *playerActor, world *sceneLifecycle,
	x, y, z, orient float32, trigger, remote string) error {
	dashIssued := player.rushDashIssuedSnapshot()
	dashAirborne := player.rushDashAirborneSnapshot()
	player.clearRushInFlight()
	if world != nil && world.wireID != 0 {
		world.run.updateMemberPosition(world.wireID, role.Position{X: x, Y: y, Z: z, Orient: orient})
	}
	pending := player.consumePendingCast(time.Now().UTC())
	if pending == nil {
		// 无暂存出招（过期/重复落地）：外服私服零回应（b 实证），静默对账。
		// v31 补：空中档已授予而 212 迟到于 30s 窗后——轻功绑定与授予对
		//（CantMove/CantParry=1）仍须还原，否则轻功状态残留+玩家被冻
		//（「穿云纵后续有概率用不出」同族出口补全，2026-11-17 用户令
		// 核对轻功复位）。
		if dashIssued {
			if dashAirborne {
				writeQinggongReleaseFrame(link, x, y, z)
			}
			if clear, err := rushPropertyFrame(propRushCantMove, "CantMove", clientdata.ByteValue(0)); err == nil {
				_ = link.WriteFrame(clear)
			}
			if clear, err := rushPropertyFrame(propCantParry, "CantParry", clientdata.Int32Value(0)); err == nil {
				_ = link.WriteFrame(clear)
			}
		}
		log.Printf("%s: dash landing without live pending cast (expired or none) via %s", remote, trigger)
		return nil
	}
	now := time.Now().UTC()
	geometry := pending.geometry
	orientToTarget := orientTowardTarget(x, z, geometry.X, geometry.Z)
	distance := math.Sqrt(float64(square(x-geometry.X) + square(z-geometry.Z)))
	model, _ := world.combatNPCModel(pending.targetID)
	// 步行落地（未结算）：CD/MP 在落地簇内结算（b 51.387 外服私服序）。
	// 字面/零成本定义（自增益测试目标）跳过结算与 CD 行。
	settledNow := false
	if !pending.settled && (pending.cooldownCategory != 0 || pending.mpCost > 0 || pending.personalCD > 0) {
		if err := player.commitRushDashCosts(pending, now, false); err != nil {
			log.Printf("%s: rush walk-in settle failed skill id=%s: %v", remote, pending.skillID, err)
		} else {
			settledNow = true
		}
	}
	type frameBuilder struct {
		name  string
		build func() ([]byte, error)
	}
	var builders []frameBuilder
	if dashIssued {
		if dashAirborne {
			// 空中档冲刺经 212 到位申报出口时同样必须释放轻功绑定
			//（2026-11-16「穿云纵后续有概率用不出」根因：该帧原只在
			// ETA 路径发，212 抢先→AfterFunc 早退→状态残留）。
			writeQinggongReleaseFrame(link, x, y, z)
		}
		builders = append(builders,
			frameBuilder{"CantMove-clear", func() ([]byte, error) {
				return rushPropertyFrame(propRushCantMove, "CantMove", clientdata.ByteValue(0))
			}},
			frameBuilder{"CantParry-clear", func() ([]byte, error) {
				return rushPropertyFrame(propCantParry, "CantParry", clientdata.Int32Value(0))
			}},
		)
	}
	builders = append(builders,
		frameBuilder{"PauseSkill-clear", func() ([]byte, error) {
			return rushPropertyFrame(propRushPauseSkill, "PauseSkill", clientdata.StringValue(""))
		}},
		frameBuilder{"msg26", func() ([]byte, error) { return skillPauseClearFrame() }},
		frameBuilder{"NPC-LogicState", func() ([]byte, error) {
			return objectSingleProperty(pending.targetID, pending.targetOwner, propRushLogicState, "LogicState", clientdata.ByteValue(1))
		}},
		frameBuilder{"fight-relation", func() ([]byte, error) { return fightRelationRowAddFrame(pending.targetID, pending.targetOwner), nil }},
		frameBuilder{"LogicState", func() ([]byte, error) {
			return rushPropertyFrame(propRushLogicState, "LogicState", clientdata.ByteValue(1))
		}},
	)
	for _, builder := range builders {
		frame, err := builder.build()
		if err != nil {
			return fmt.Errorf("encode landing %s: %w", builder.name, err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return err
		}
	}
	if settledNow {
		state := player.stateSnapshot()
		mpRatio := int32(0)
		if state.MaxMP > 0 {
			mpRatio = state.MP * 100 / state.MaxMP
		}
		if err := player.writeOfficialCooldownRows(link, pending.cooldownCategory, pending.cooldownTeam,
			pending.personalCD, pending.publicCD, now); err != nil {
			return fmt.Errorf("write landing cooldown rows %s: %w", pending.skillID, err)
		}
		for _, step := range []struct {
			name  string
			build func() ([]byte, error)
		}{
			{"MPRatio", func() ([]byte, error) {
				return rushPropertyFrame(propMPRatio, "MPRatio", clientdata.Int32Value(mpRatio))
			}},
			{"MP", func() ([]byte, error) {
				return rushPropertyFrame(propMP, "MP", clientdata.Int32Value(state.MP))
			}},
		} {
			frame, err := step.build()
			if err != nil {
				return fmt.Errorf("encode landing %s: %w", step.name, err)
			}
			if err := link.WriteFrame(frame); err != nil {
				return err
			}
		}
	}
	anchor, anchorErr := serverLocation(playerObjectID, playerOwnerID, worldcore.Transform{X: x, Y: y, Z: z}), error(nil)
	tailCount := 1
	if dashIssued {
		tailCount = 2
	}
	for i := 0; i < tailCount; i++ {
		if anchorErr != nil {
			return anchorErr
		}
		if err := link.WriteFrame(anchor); err != nil {
			return err
		}
	}
	tail := []struct {
		name  string
		build func() ([]byte, error)
	}{
		{"msg337", func() ([]byte, error) { return casterTransformSyncFrame(x, y, z, orientToTarget) }},
		{"c1=40-precast", func() ([]byte, error) { return rushTargetHitPrecastFrame(pending.targetID, pending.targetOwner) }},
		{"msg19-open", func() ([]byte, error) {
			return segmentDamageOpenFrame(pending.targetID, pending.targetOwner, pending.skillID, pending.nominalDamage,
				x, y, z, orientToTarget, distance)
		}},
		{"CantMove-set", func() ([]byte, error) {
			return rushPropertyFrame(propRushCantMove, "CantMove", clientdata.ByteValue(1))
		}},
		{"CantParry-set", func() ([]byte, error) {
			return rushPropertyFrame(propCantParry, "CantParry", clientdata.Int32Value(1))
		}},
	}
	for _, step := range tail {
		frame, err := step.build()
		if err != nil {
			return fmt.Errorf("encode landing %s: %w", step.name, err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return err
		}
	}
	// 状态四连（外服私服 416.162 落地簇 → 416.321 +159ms 四连实证）。
	for _, ping := range officialStatePingFrames() {
		if ping == nil {
			continue
		}
		if err := link.WriteFrame(ping); err != nil {
			return err
		}
	}
	_ = model
	// 出招簿记 + 伤害链（settled 跳过 CD/MP 闸防自拒）。
	target := uint64(pending.targetID) | uint64(pending.targetOwner)<<32
	level, learned := player.learnedSkillLevel(pending.skillID)
	definition, valid := installedCombatSkillCatalog.definition(pending.skillID, level)
	if !learned || !valid {
		log.Printf("%s: dash landing resume lost definition skill id=%s", remote, pending.skillID)
		return nil
	}
	if ok, reason := player.beginSkillUse(definition, target, now, true); !ok {
		log.Printf("%s: dash landing begin skill use rejected id=%s: %s", remote, pending.skillID, reason)
		return nil
	}
	castEpoch := player.currentSkillEpochSnapshot()
	log.Printf("%s: dash landing resume skill id=%s via %s dash=%t orient=%.3f",
		remote, pending.skillID, trigger, dashIssued, orientToTarget)
	launchSkillDamagePhase(link, player, world, definition, clientCustomMessage{},
		pending.targetID, pending.targetOwner, castEpoch, remote)
	return nil
}

// ---- msg15 / msg220 移动上报 ----

func handleMovementUpload(link sceneMessageConnection, player *playerActor, world *sceneLifecycle,
	custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomMovementUpload {
		return false, nil
	}
	values := custom.Values
	if len(values) < 6 || values[3].Type != 4 || values[4].Type != 4 || values[5].Type != 4 {
		return true, nil
	}
	// 外服私服 msg15 全形态（走 82B / 停 67B / 跳 105B）idx3..5 恒为 x,y,z，
	// 非冲锋在途也收编位置（技能.pcapng 定版：0x21 回显含玩家条目 =
	// 服务端收编证据；追击 NPC 测距依赖服务端侧玩家坐标）。
	x, y, z := values[3].Float32, values[4].Float32, values[5].Float32
	orient := movementUploadOrient(values)
	absorbClientPosition(world, x, y, z, orient)
	if player == nil {
		return true, nil
	}
	player.noteClientReport(x, y, z, time.Now())
	// JumpTimes 飞行计数官方镜像（全路径独立于冲锋态，含冲刺在途段）。
	if len(values) > 2 && values[2].Type == 2 {
		player.mirrorJumpTimes(link, values[2].Int32, y)
	}
	if player.rushInFlightSnapshot() {
		if player.rushDashIssuedSnapshot() {
			// c61 冲刺段在途位移：不回显，消费方为 212/c61 落地链
			//（外服私服方言）。
			tryRushArrival(link, player, world, x, y, z, orient, remote)
			return true, nil
		}
		echoPlayerMoveIntent(link, player, values, x, y, z, remote)
		tryRushArrival(link, player, world, x, y, z, orient, remote)
		return true, nil
	}
	// 外服私服移动应答方言（技能.pcapng §一）：逐帧 0x21 玩家回显 +
	// 走↔停迁移状态四连——客户端等这对帧才执行点击移动。
	echoPlayerMoveIntent(link, player, values, x, y, z, remote)
	return true, nil
}

// echoPlayerMoveIntent 外服私服移动应答（技能.pcapng §一 全样本实证）：每帧
// 寻路族（id 高 16 位 0xA000——点击地面/地图点击/技能接近）与 WASD 族
// （0x0000/0x3000）的 state7/4/6/8 回显一帧单条目 0x21（走帧
// dest=(idx6, y, idx7)、角度=atan2(destX−x, destZ−z)、速度=idx8、朝向=
// idx9；停帧 dest=上报终点、角度=朝向增量 idx6、速度/朝向=0）；state5
// 跳跃双帧不回显。state7↔4 走停迁移沿先补发外服私服状态四连
// （586/645/646/750——外服私服对每次点击起跑/到位停帧 +0~80ms 必发，
// 64785 会话实证缺它客户端停滞不自走），同态重发只回 0x21（外服私服追怪
// 改向五连重报无 ping）。冲锋在途段不回显（消费方为 tryRushArrival，
// 外服私服该段位移走 c61/212 落地链）。
func echoPlayerMoveIntent(link sceneMessageConnection, player *playerActor, values []clientCustomValue,
	x, y, z float32, remote string) {
	if len(values) < 7 || values[1].Type != 2 || values[2].Type != 2 || values[6].Type != 4 {
		return
	}
	// 手动格挡期间整段静默（2026-11-16 真机 54455 会话定案：按住右键时
	// 客户端的微小漂移上报被回显成带行走 intent（0x30000F01 等）的 0x21，
	// 客户端移动机看到行走意图即撤架势——七次「受理→1 秒内塌」全此根因；
	// 官服格挡期不回显位移，受理簇后那帧 0x21 是格挡减速同步非行走意图）。
	if player.parryingSnapshot() {
		return
	}
	intentID := uint32(values[1].Int32)
	state := values[2].Int32
	echo := playerMoveEcho{objectID: playerObjectID, ownerID: playerOwnerID, intentID: intentID}
	walking := false
	switch state {
	case 7, 6, 8: // 行进态（6/8=下落/滑翔变体，外服私服同回显）
		if len(values) < 8 || values[7].Type != 4 {
			return
		}
		echo.destX, echo.destY, echo.destZ = values[6].Float32, y, values[7].Float32
		echo.angle = moveEchoAngle(echo.destX-x, echo.destZ-z)
		if len(values) > 8 && values[8].Type == 4 {
			echo.speed = values[8].Float32
		}
		echo.orient = movementUploadOrient(values)
		walking = true
	case 4: // 停帧：外服私服角度槽=朝向增量（0.07→ang=0.07 三证）
		echo.destX, echo.destY, echo.destZ = x, y, z
		echo.angle = values[6].Float32
	default:
		return
	}
	if !player.noteMoveEchoWalking(walking) {
		if err := link.WriteFrame(serverAllDestPlayerEcho(echo)); err != nil {
			log.Printf("%s: player move echo: %v", remote, err)
		}
		return
	}
	for _, ping := range officialStatePingFrames() {
		if ping == nil {
			continue
		}
		if err := link.WriteFrame(ping); err != nil {
			log.Printf("%s: player move echo ping: %v", remote, err)
			return
		}
	}
	if err := link.WriteFrame(serverAllDestPlayerEcho(echo)); err != nil {
		log.Printf("%s: player move echo: %v", remote, err)
	}
}

// moveEchoAngle 外服私服移动角：atan2(dx, dz) 归一 [0, 2π)（技能.pcapng
// 外服私服样本 0.07/2.12/5.21/4.69 四证——负角补 2π，如 -1.076→5.21）。
func moveEchoAngle(dx, dz float32) float32 {
	angle := float32(math.Atan2(float64(dx), float64(dz)))
	if angle < 0 {
		angle += 2 * float32(math.Pi)
	}
	return angle
}

// noteMoveEchoWalking 记录走停态并报告是否发生迁移（迁移=状态四连
// 触发器；锁 p.mu）。
func (p *playerActor) noteMoveEchoWalking(walking bool) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	transitioned := p.moveEchoWalking != walking
	p.moveEchoWalking = walking
	return transitioned
}

// movementUploadOrient 提取 msg15 的绝对朝向：仅 state7 长式带
// （[15, id, 7, x, y, z, destx, desty, speed, orient, clock]，idx9）；
// state4 停帧 idx6 是增量朝向、跳帧无朝向——均不取。旧实现取 idx6
// （实为 destx）= 字段错位。
func movementUploadOrient(values []clientCustomValue) float32 {
	if len(values) > 9 && values[2].Type == 2 && values[2].Int32 == 7 && values[9].Type == 4 {
		return values[9].Float32
	}
	return 0
}

// absorbClientPosition 把客户端移动上报收编为场景侧权威位置。
func absorbClientPosition(world *sceneLifecycle, x, y, z, orient float32) {
	if world == nil || world.wireID == 0 {
		return
	}
	world.run.updateMemberPosition(world.wireID, role.Position{X: x, Y: y, Z: z, Orient: orient})
}

// areaRushDirectCastMeters：范围型点地直收带上界（技能.pcapng 分带：
// 1.45m 直收无 c61 / 3.96m、5.52m 档1 冲——分界 ∈ (1.45, 3.96]，取与
// 锁定型直收下限同源的 3.5）。规格 §二。
const areaRushDirectCastMeters = 3.5

// useSkillGroundPointTransform 解析 211 短式 idx6..8 的地面点（范围型
// hyjf07 点地形态；锁定式此三值为 0）。
func useSkillGroundPointTransform(custom clientCustomMessage) (worldcore.Transform, bool) {
	if len(custom.Values) < 9 ||
		custom.Values[6].Type != 4 || custom.Values[7].Type != 4 || custom.Values[8].Type != 4 {
		return worldcore.Transform{}, false
	}
	transform := worldcore.Transform{
		X: custom.Values[6].Float32,
		Y: custom.Values[7].Float32,
		Z: custom.Values[8].Float32,
	}
	if transform.X == 0 && transform.Y == 0 && transform.Z == 0 {
		return worldcore.Transform{}, false
	}
	return transform, true
}

// issueAreaGroundRushDash 范围型点地冲刺（规格 §六）：c61 直冲点选平面点
// 本体（dest.y=点+1.0，唯空中档=平面）+ AOI 裸广播，位移由客户端自走；
// ETA=3D 距离/档速（地面 25+120ms / 空中 15+250ms）后落地簇（空中档先补
// 轻功终结符）锚=客户端当前上报位（绝不写 dest 防瞬移）；调用方以同 ETA
// 在终点重放 211 走主施法直收。CD/MP 仍由主出招路径结算。
func issueAreaGroundRushDash(link sceneMessageConnection, player *playerActor, world *sceneLifecycle,
	definition combatSkillDefinition, castAt, ground worldcore.Transform, remote string) time.Duration {
	distance := math.Sqrt(float64(square(castAt.X-ground.X) + square(castAt.Z-ground.Z)))
	if distance <= areaRushDirectCastMeters {
		return 0
	}
	destX, destY, destZ := ground.X, ground.Y, ground.Z
	// 空中档判定（规格 §二）：castAt/ground 均 211 客户端自报坐标（同一坐
	// 标系）——高→低 >1m 或 低→高 >0.5m → 空中档（flag2+A 类动画）。
	airborne := rushChargeAirborne(castAt.Y, ground.Y)
	// v31.2（技能4 点地 #2/#3/#4 三样本定版）：c61 dest.y=点选平面+1.0
	//（4.975=3.975+1.0 全带，与单体地面档同方言）；destY 平面值保留给
	// ETA 数学与同步条目。空中档维持点选平面（v28 定版，无反证）。
	c61Y := destY + rushDashLiftMeters
	if airborne {
		c61Y = destY
	}
	speed := rushDriveDashSpeedMeters
	slack := 120 * time.Millisecond
	if airborne {
		speed = rushDriveAirDashSpeedMeters
		slack = 250 * time.Millisecond
	}
	dashDur := time.Duration(math.Sqrt(float64(square(destX-castAt.X)+square(destZ-castAt.Z)+square(destY-castAt.Y)))/float64(speed)*float64(time.Second)) + slack
	// v26：范围技地面=G 类 flag1（技能3.pcapng 107.95 hyjf07 点地实发
	// qg_fastrun_2h_01 flag1）；高→低=A 类 flag2（技能4.pcapng 空中方言）。
	rushAnim := definition.groundRushAction
	rushFlag := int32(1)
	if airborne {
		if definition.airRushAction != "" {
			rushAnim = definition.airRushAction
		}
		rushFlag = 2
	}
	// v31（用户令「去掉 0x21」）：范围点地冲刺同转官方自走模型——c61 +
	// AOI 裸广播后零位移驱动（官方 240.571 到位锚 op1F 跟随客户端 m212
	// 位）。落地兜底锚=客户端当前上报位（绝不写 dest，防位置跳变=瞬移）。
	if rushAnim != "" {
		if frame, err := dashRushFrame(destX, c61Y, destZ, rushFlag, rushAnim); err == nil {
			_ = link.WriteFrame(frame)
			broadcastRushDashAOI(world, player, frame)
		}
	}
	// v31.2 成对 0x21 冲刺同步条目（技能4 点地 #2/#3/#4：47B 单条目、
	// dest 与 c61 dest xz 完全一致、y=点选平面、s5=35、s6=-2π、s7=10.8、
	// s8=9.9、intent=0x4701；空中档 s5=15/intent=0x4704、y=c61 dest）。
	// 仅发施法者本人（观察者零位移流，群战.pcapng）。
	syncSpeed := rushAreaSyncSpeedMeters
	if airborne {
		syncSpeed = rushDriveAirDashSpeedMeters
	}
	syncDest := worldcore.Transform{X: destX, Y: destY, Z: destZ,
		Orient: orientTowardTarget(castAt.X, castAt.Z, ground.X, ground.Z)}
	if airborne {
		syncDest.Y = c61Y
	}
	if rushSyncPairEnabled() {
		_ = link.WriteFrame(rushDashSyncFrame(airborne, syncSpeed, syncDest, worldcore.Transform{}, false))
	}
	airTag := ""
	if airborne {
		airTag = " air"
	}
	log.Printf("%s: area rush dash skill id=%s ground=(%.2f,%.2f,%.2f) dist=%.1fm%s eta=%s (client self-run, no 0x21)",
		remote, definition.id, ground.X, ground.Y, ground.Z, distance, airTag, dashDur)
	time.AfterFunc(dashDur, func() {
		anchorX, anchorY, anchorZ := player.clientReportPositionSnapshot()
		if anchorX == 0 && anchorY == 0 && anchorZ == 0 {
			anchorX, anchorY, anchorZ = castAt.X, castAt.Y, castAt.Z
		}
		anchor := worldcore.Transform{
			X: anchorX, Y: anchorY, Z: anchorZ,
			Orient: orientTowardTarget(anchorX, anchorZ, ground.X, ground.Z),
		}
		// v31 补（2026-11-17 用户令核对轻功复位）：空中档（高→低俯冲/
		// 低→高奔袭）落地必须发轻功绑定终止符——点地冲锋打断了客户端
		// 飞行/轻功状态，不还终结符=「穿云纵后续有概率用不出」复发
		//（writeQinggongReleaseFrame 统一出口方言）。
		if airborne {
			writeQinggongReleaseFrame(link, anchor.X, anchor.Y, anchor.Z)
		}
		steps := []struct {
			name  string
			build func() ([]byte, error)
		}{
			{"msg26", func() ([]byte, error) { return skillPauseClearFrame() }},
			{"0x1F", func() ([]byte, error) { return serverLocation(playerObjectID, playerOwnerID, anchor), nil }},
			{"msg337", func() ([]byte, error) { return casterTransformSyncFrame(anchor.X, anchor.Y, anchor.Z, anchor.Orient) }},
		}
		for _, step := range steps {
			frame, err := step.build()
			if err != nil {
				log.Printf("%s: area rush landing %s: %v", remote, step.name, err)
				return
			}
			if err := link.WriteFrame(frame); err != nil {
				log.Printf("%s: area rush landing %s: %v", remote, step.name, err)
				return
			}
		}
	})
	return dashDur
}

// tryRushArrival 移动上报驱动的冲刺探测：距目标进入 (直收带, GRushRange+Add
// 6.5]（方言窗优先）时发 c61——客户端自动接近停点 5.19~5.43m 落此窗
//（2026-11-17 真机定案）。低于下界由客户端自行走进走带发 212 落地。
// 冲刺已发出后静默——落地只认 212。
func handleRushMovementUpload(link sceneMessageConnection, player *playerActor, world *sceneLifecycle,
	custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomRushMoveUpload {
		return false, nil
	}
	values := custom.Values
	if len(values) < 4 || values[1].Type != 4 || values[2].Type != 4 || values[3].Type != 4 {
		return true, nil
	}
	orient := float32(0)
	if len(values) > 4 && values[4].Type == 4 {
		orient = values[4].Float32
	}
	// msg220 [220, x, y, z, orient?]：位置收编（官方接近段上报）。
	absorbClientPosition(world, values[1].Float32, values[2].Float32, values[3].Float32, orient)
	if player != nil {
		player.noteClientReport(values[1].Float32, values[2].Float32, values[3].Float32, time.Now())
	}
	if player == nil || !player.rushInFlightSnapshot() {
		return true, nil
	}
	tryRushArrival(link, player, world, values[1].Float32, values[2].Float32, values[3].Float32, orient, remote)
	return true, nil
}

func tryRushArrival(link sceneMessageConnection, player *playerActor, world *sceneLifecycle,
	x, y, z, orient float32, remote string) {
	pending := player.peekPendingCast(time.Now().UTC())
	if pending == nil {
		if player.rushInFlightSnapshot() {
			player.clearRushInFlight()
			log.Printf("%s: rush pending expired; cleared in-flight state", remote)
		}
		return
	}
	// maybeRushStatePing 停用（对照构建：外服私服四连是行走上报的回执，
	// 无走不发；多余下发疑似破坏接近态——定位中）。
	target, ok := world.combatNPCTransform(pending.targetID)
	if !ok {
		return
	}
	geometry := pending.geometry
	if geometry.X == 0 && geometry.Y == 0 && geometry.Z == 0 {
		geometry = target
	}
	// xz 平面距离（与 212 三分同口径；技能.pcapng c61 触发三证
	// 6.09/6.10/5.91 均平地未暴露 y，统一 2D 防坡地 y 差误判）。
	distance := math.Sqrt(float64(square(x-geometry.X) + square(z-geometry.Z)))
	if player.rushDashIssuedSnapshot() {
		return
	}
	// v31（技能5.pcapng 定版，2026-11-17）：空/地分带分口径——
	//   空中档（rushChargeAirborne：高→低 >1m 或 低→高 >0.5m）：授予窗
	//     =(直收带, ARushRange+Add 14.5]，距离取 **3D 空间距离**（技能5
	//     六样本 6.33~10.95 全落此窗；直收带下界与地面档连续无死区——
	//     「近点立即释放、远处冲锋」二分；穿云纵 state7 飞行中 m220 即授，
	//     c61 打断飞行转入奔袭——官方实证，不再 airFlightActive 扣住）；
	//   地面档（同层）：2D 距离 + v29 分档带（pending.dashWindowLo/Hi，
	//     方言表/族规则 staging 时填好；6m 族无带）。
	airborne := rushChargeAirborne(y, geometry.Y)
	lo, hi := pending.dashWindowLo, pending.dashWindowHi
	if airborne {
		hi = rushAirChargeHiMeters
		if lo <= 0 {
			// 6m 族（无地面带）：空中下界=直收带 castRange+1。
			lo = float64(pending.castRange) + 1
		}
		distance = math.Sqrt(float64(square(x-geometry.X) + square(z-geometry.Z) + square(y-geometry.Y)))
	}
	if hi <= 0 || lo <= 0 {
		return
	}
	if distance > hi || distance <= lo {
		return
	}
	log.Printf("%s: rush within dash band skill id=%s dist=%.1fm in (%.1f, %.1f] air=%t",
		remote, pending.skillID, distance, lo, hi, airborne)
	rushDashDrive(link, player, world, geometry, x, y, z, remote)
}

// rushDashDrive c61 授予（规格 §四/§五）：授予簇=c61[dest,flag,动作]（dest
// 全档 y+1.0，唯低→高空中档=平面）→ CantMove/CantParry=1 → CD 双行 →
// MPRatio/MP（空中 +QingGongPoint、+60ms m707[29]）→ [可选同步条目 §4.1]
// + AOI 裸广播；此后零服务端位移帧，位移/动画全由客户端执行（穿云纵被
// c61 打断转入奔袭）。释放：m212 到位申报主通道（resumePendingCastAt 锚
// 212 自报位）+ 呆坐哨兜底（≥max(ETA+500ms,1.5s) 未动在客户端当前上报位
// 就地释放，不写 bind 防瞬移）+ 30s 过期/取消/rearm 出口（§5.2 还原组）。
func rushDashDrive(link sceneMessageConnection, player *playerActor, world *sceneLifecycle,
	geometry worldcore.Transform, fromX, fromY, fromZ float32, remote string) {
	pending := player.peekPendingCast(time.Now().UTC())
	if pending == nil || player.rushDashIssuedSnapshot() {
		return
	}
	if err := player.commitRushDashCosts(pending, time.Now().UTC(), true); err != nil {
		log.Printf("%s: rush dash cost settle failed skill id=%s: %v", remote, pending.skillID, err)
		cancelPendingCast(link, player, fromX, fromY, fromZ, 0, remote)
		return
	}
	bindX, _, bindZ := rushBindPoint(geometry, fromX, fromZ, pending.bindDistance)
	// v30 空中档（2026-11-16 用户补充）：低→高 >0.5m 亦空中奔袭——原
	// clientAirborneAbove 只识高→低 >1.5m，低地冲高地被判地面档=贴地
	// 跑步爬墙根因。
	airborne := rushChargeAirborne(fromY, geometry.Y)
	// v31.2（技能4 地面五样本定版，2026-11-17）：c61 dest.y=目标 y+1.0
	// 为全档方言（地面五样本 4.975=3.975+1.0 全带；高→低俯冲同，技能5
	// 六样本 0.68~1.08）；仅低→高空中档=目标平面本体（v30 用户定版——
	// 再抬 1.0 会冲过平台面悬空，无反证）。
	destY := geometry.Y + rushDashLiftMeters
	if airborne && fromY <= geometry.Y {
		destY = geometry.Y
	}
	// 呆坐哨延迟：官方冲刺耗时 0.21~0.99s（技能5 六样本），ETA 按档速
	// 3D 推算 + 富余，且不低于 rushDashFallbackDelay（1.5s，2026-11-15
	// 真机「c61 未消费呆站」首查线）。
	speed := rushDriveDashSpeedMeters
	if airborne {
		speed = rushDriveAirDashSpeedMeters
	}
	dashDist := math.Sqrt(float64(square(bindX-fromX) + square(bindZ-fromZ)))
	dashDur := time.Duration(math.Sqrt(dashDist*dashDist+
		float64(square(destY-fromY)))/float64(speed)*float64(time.Second)) + 500*time.Millisecond
	if dashDur < rushDashFallbackDelay {
		dashDur = rushDashFallbackDelay
	}
	// v25：官方式 c61 动作帧（flag1=G 类 qg_fastrun_{W}h_01 地面 /
	// flag2=A 类 qg_fastruning_s_{W}h_01 空中；动作名按技能 skill_lock_data
	// 的 LockG/LockARushAction）。
	rushAnim := pending.groundRushAction
	rushFlag := int32(1)
	if airborne {
		rushAnim = pending.airRushAction // v26：A 类 fastruning_s（表 LockARushAction）
		if rushAnim == "" {
			rushAnim = pending.dashAction
		}
		rushFlag = 2
	}
	if rushAnim != "" {
		if frame, err := dashRushFrame(bindX, destY, bindZ, rushFlag, rushAnim); err == nil {
			_ = link.WriteFrame(frame)
			// v31 AOI 广播（群战.pcapng 定版）：他人冲锋=裸 c61（观察者
			// 不收 CantMove/CD/MP 私有簇、冲刺期零 0x21 位移流，远端渲染
			// 纯靠 c61）。此前只写施法者自己的 link——别人看不到冲锋。
			broadcastRushDashAOI(world, player, frame)
		}
	}
	// v29 c61 授予簇补齐（技能4.pcapng 20.194 空中授予逐帧：c61 →
	// CantMove=1 → CantParry=1 → CD 记录簇 → MPRatio/MP（空中 +
	// QingGongPoint）；落地簇/取消/换技能出口的清 0 对已在各自路径）。
	// CantMove=1 冲刺不可打断（官方同毫秒方言）；CD/MP 已在上方
	// commitRushDashCosts 结算，此处为表现帧。
	if frame, err := rushPropertyFrame(propRushCantMove, "CantMove", clientdata.ByteValue(1)); err == nil {
		_ = link.WriteFrame(frame)
	}
	if frame, err := rushPropertyFrame(propCantParry, "CantParry", clientdata.Int32Value(1)); err == nil {
		_ = link.WriteFrame(frame)
	}
	if err := player.writeOfficialCooldownRows(link, pending.cooldownCategory, pending.cooldownTeam,
		pending.personalCD, pending.publicCD, time.Now().UTC()); err != nil {
		log.Printf("%s: rush dash cooldown rows skill id=%s: %v", remote, pending.skillID, err)
	}
	state := player.stateSnapshot()
	mpRatio := int32(0)
	if state.MaxMP > 0 {
		mpRatio = state.MP * 100 / state.MaxMP
	}
	if frame, err := rushPropertyFrame(propMPRatio, "MPRatio", clientdata.Int32Value(mpRatio)); err == nil {
		_ = link.WriteFrame(frame)
	}
	if frame, err := rushPropertyFrame(propMP, "MP", clientdata.Int32Value(state.MP)); err == nil {
		_ = link.WriteFrame(frame)
	}
	if airborne && pending.qgCost > 0 {
		// 官方空中授予簇尾 QingGongPoint 单帧（20.194 QingGongPoint=148）。
		if frame, err := rushPropertyFrame(propQingGongPointIndex, "QingGongPoint",
			clientdata.Int32Value(player.qinggongPointSnapshot())); err == nil {
			_ = link.WriteFrame(frame)
		}
	}
	if airborne {
		// 技能4.pcapng：空中授予后 +40~80ms 补 m707[29]（3/3 空中授予全带，
		// 地面授予零样本）——空中受理专属随帧。
		time.AfterFunc(60*time.Millisecond, func() {
			if frame, err := rushAirGrantFollowFrame(); err == nil {
				_ = link.WriteFrame(frame)
			}
		})
	}
	// v31.2 成对 0x21 冲刺同步条目（技能4 #5 91B 双条目 / 技能5 空中 47B
	// 逐字节定版，详见 rushDashSyncFrame 注释）：地面单体=[同步条 s5=25
	// intent=0x4701 dest=c61 dest xz、y=地面平面]+[空置回显 dest=按压位
	// 0xA0000700]；空中=47B s5=15 intent=0x4704 dest=c61 dest 本体。
	// 仅发施法者本人（观察者零位移流，群战.pcapng）。
	syncY := geometry.Y
	if airborne {
		syncY = destY
	}
	syncDest := worldcore.Transform{X: bindX, Y: syncY, Z: bindZ,
		Orient: orientTowardTarget(bindX, bindZ, geometry.X, geometry.Z)}
	pressEcho := worldcore.Transform{X: fromX, Y: fromY, Z: fromZ,
		Orient: orientTowardTarget(fromX, fromZ, geometry.X, geometry.Z)}
	if rushSyncPairEnabled() {
		_ = link.WriteFrame(rushDashSyncFrame(airborne, speed, syncDest, pressEcho, !airborne))
	}
	player.markRushDashIssued(airborne)
	grantedAt := time.Now().UTC()
	airTag := ""
	if airborne {
		airTag = " air"
	}
	syncTag := "sync-pair mover"
	if !rushSyncPairEnabled() {
		syncTag = "bare-c61"
	}
	log.Printf("%s: rush dash grant dest=(%.2f,%.2f,%.2f)%s action=%s eta-watchdog=%s (%s)",
		remote, bindX, destY, bindZ, airTag, rushAnim, dashDur, syncTag)
	time.AfterFunc(dashDur, func() {
		if player.peekPendingCast(time.Now().UTC()) != pending {
			return // 已 212 到位/取消/rearm（轻功释放帧由 resume/cancel 出口发）
		}
		// 客户端已自行移动=自走在途（官方模型），让位 212 链；30s 暂存
		// 窗与 skillPauseReleaseTick 兜底收尾。
		if player.clientMovedSince(grantedAt, fromX, fromZ) {
			return
		}
		// 呆坐兜底（c61 未被消费，2026-11-15 真机实证形态）：在客户端
		// 当前上报位就地释放——绝不写 bind 点（位置跳变=瞬移观感根因）。
		cx, cy, cz := player.clientReportPositionSnapshot()
		if cx == 0 && cy == 0 && cz == 0 {
			cx, cy, cz = fromX, fromY, fromZ
		}
		orient := orientTowardTarget(cx, cz, geometry.X, geometry.Z)
		if err := resumePendingCastAt(link, player, world, cx, cy, cz, orient, "dash-idle-fallback", remote); err != nil {
			log.Printf("%s: rush dash idle fallback: %v", remote, err)
		}
	})
}

// broadcastRushDashAOI 把施法者的 c61 冲锋帧广播给同场景其他成员
//（群战.pcapng 定版：观察者收到的他人冲锋=裸 c61，无私有属性簇、无
// 0x21 位移流）。施法者本人的 link 已收完整授予簇，跳过。
func broadcastRushDashAOI(world *sceneLifecycle, caster *playerActor, frame []byte) {
	if world == nil || world.run == nil {
		return
	}
	world.run.mu.Lock()
	views := make([]*sceneLifecycle, 0, len(world.run.members))
	for _, member := range world.run.members {
		if member.player == caster || member.view == nil || !member.active {
			continue
		}
		views = append(views, member.view)
	}
	world.run.mu.Unlock()
	for _, view := range views {
		if err := view.conn.WriteFrame(frame); err != nil {
			log.Printf("%s: broadcast rush c61: %v", view.remote, err)
		}
	}
}

func square(value float32) float32 { return value * value }

// writeQinggongReleaseFrame 轻功绑定释放帧（v26 官方式终结符：0x21 空置
// 条目 speed=0/motion3=9.9/intent=0xA0000700，另一服 self_waypoint 同款）。
// 空中档冲刺的所有出口都必须发（2026-11-16「穿云纵后续有概率用不出」
// 根因：该帧原只在 ETA 落地路径发，客户端 212 抢先到→resumePendingCastAt
// 出招后 AfterFunc 早退→轻功状态残留锁死后续轻功）。
// rushAreaSyncSpeedMeters 点地档 0x21 冲刺同步条目 s5（技能4.pcapng
// #2/#3/#4 三样本逐字节定版 35.0——点地档与地面单体档 25 分档）。
const rushAreaSyncSpeedMeters = float32(35.0)

// rushSyncPairEnabled v31.2 成对 0x21 冲刺同步条目开关（**默认开**，
// NINEYIN_RUSH_SYNC_PAIR=off 关闭）。2026-11-17 23:22 真机翻案：授予后
// 客户端 state4 停一帧即恢复 state7 普通走路（3.9m/s）——**地面档裸 c61
// 零位移**（v24「c61 受理即授予=六秒零位移」同证），0x4701/0x4704 skillrush
// 自驱 mover（0x21 白名单里唯一逐帧自走的态）就是官方地面冲刺位移本体
// （技能4 地面 4/5 样本 c61 后 63~214ms 必发）；此前「客户端自行冲刺」
// 系误判，「歪歪扭扭」=mover 与客户端接近行走的抢道，待带内短程（≤6.5m
// 授予、0.2s 级驱动）复测观感，异常走 off 回退。
func rushSyncPairEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("NINEYIN_RUSH_SYNC_PAIR")))
	return v != "off" && v != "0"
}

// rushSyncMotion2 冲刺同步条目 s7(Motion2) 官方主值（技能4 4/5 样本
// 10.8、#2 一例 0——语义未证，按主值复刻）。
const rushSyncMotion2 = float32(10.8)

// rushDashSyncFrame 官方式 c61 成对 0x21 冲刺同步条目（v31.2，2026-11-17
// 技能4 地面五样本 + 技能5 空中样本逐字节定版）：官方 c61 授予簇后
// 63~214ms 必发一条本人 0x21——
//   地面单体（#5，91B 双条目）：同步条 dest=c61 dest 的 xz、y=地面平面
//    （非 c61 的 +1.0 抬升值）、s5=25(LandRushSpeed)、s6=-2π、s7=10.8、
//     s8=9.9、intent=0x4701；第二成员=空置回显（dest=按压位、s5=0、
//     s6=0、s7=10.8、s8=9.9、intent=0xA0000700）。
//   地面点地（#2/#3/#4，47B 单条目）：dest 与 c61 dest 完全一致、
//     y=点选平面、s5=35、s6=-2π、intent=0x4701。
//   空中（技能5 #2，47B 单条目）：dest=c61 dest 本体（y 同 c61）、
//     s5=15(AirRushSpeed)、intent=0x4704（flag4=空中变体）。
// intent 高字节 0x47=skillrush 事件 id(0x71) 态家族。仅发施法者本人
// （群战.pcapng：观察者零位移流，c61 才是远端渲染源）。
func rushDashSyncFrame(airborne bool, speed float32, dest, press worldcore.Transform, withIdleEcho bool) []byte {
	intent := uint32(0x4701)
	if airborne {
		intent = 0x4704
	}
	sync := allDestEntry{
		objectID: playerObjectID, ownerID: playerOwnerID,
		destX: dest.X, destY: dest.Y, destZ: dest.Z,
		destOrient: normalizeOrient(dest.Orient),
		speed:      speed,
		rotate:     float32(-2 * math.Pi),
		motion2:    rushSyncMotion2,
		motion3:    9.9,
		moveFlags:  intent,
	}
	entries := []allDestEntry{sync}
	if withIdleEcho {
		entries = append(entries, allDestEntry{
			objectID: playerObjectID, ownerID: playerOwnerID,
			destX: press.X, destY: press.Y, destZ: press.Z,
			destOrient: normalizeOrient(press.Orient),
			motion2:   rushSyncMotion2,
			motion3:   9.9,
			moveFlags: 0xA0000700,
		})
	}
	return serverAllDest(entries)
}

func writeQinggongReleaseFrame(link sceneMessageConnection, x, y, z float32) {
	idle := allDestEntry{
		objectID: playerObjectID, ownerID: playerOwnerID,
		destX: x, destY: y, destZ: z,
		speed: 0, motion3: 9.9, moveFlags: 0xA0000700,
	}
	_ = link.WriteFrame(serverAllDest([]allDestEntry{idle}))
}

// dashLandingAimObject 解出 211 长格式 idx10 的 tag8 瞄准对象（无则 0）。
func dashLandingAimObject(custom clientCustomMessage) (uint32, uint32) {
	if len(custom.Values) <= 10 || custom.Values[10].Type != 8 {
		return 0, 0
	}
	packed := binary.LittleEndian.Uint64(custom.Values[10].Raw[:])
	return uint32(packed), uint32(packed >> 32)
}

// packPlayerObjectRaw 打包 tag8 对象值（低 32 位对象号、高 32 位 owner）。
func packPlayerObjectRaw(objectID, ownerID uint32) [8]byte {
	var raw [8]byte
	binary.LittleEndian.PutUint64(raw[:], uint64(objectID)|uint64(ownerID)<<32)
	return raw
}

// useSkillClaimedTargetTransform 解出 211 长格式 idx11..13 的目标坐标
// （客户端世界观里的目标位置）。冲锋链全部几何以它为准。
func useSkillClaimedTargetTransform(custom clientCustomMessage) (worldcore.Transform, bool) {
	if len(custom.Values) < 14 ||
		custom.Values[11].Type != 4 ||
		custom.Values[12].Type != 4 ||
		custom.Values[13].Type != 4 {
		return worldcore.Transform{}, false
	}
	transform := worldcore.Transform{
		X: custom.Values[11].Float32,
		Y: custom.Values[12].Float32,
		Z: custom.Values[13].Float32,
	}
	if transform.X == 0 && transform.Y == 0 && transform.Z == 0 {
		return worldcore.Transform{}, false
	}
	return transform, true
}
