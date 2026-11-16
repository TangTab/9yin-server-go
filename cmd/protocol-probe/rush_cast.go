package main

// 冲锋全链外服私服帧级定版（f/g/b.pcapng 实证重写，2026-11-08；
// v29 分档重写 2026-11-16，官服观察定版——距离模型按释放距离分族）：
//
//	接近段（> 冲锋带上界）：受理簇暂存（CurSkillID→CurSkillTarget→
//	   msg25[30000]→PauseSkill→msg707[15]→状态四连）——位移由客户端
//	   自走（服务端零驱动），轻功未结束前扣住不授予（挂起-延迟授予链，
//	   技能4.pcapng §2.3 同款）。
//	① 直收带（3m 族 ≤castRange+0.5 / 6m 族 ≤castRange+1）：不走暂存链。
//	   直接执行簇 = [0x12 清扫过期 CD 行] + CurSkillID + CurSkillTarget +
//	   msg284[] + NPC.LogicState=1 + 0x11 rec114 fight_relation 行0 +
//	   玩家.LogicState=1 + CD 双行(0x11 空行+0x13 逐格) + MPRatio/MP +
//	   0x1F + msg337 + LastWGAttackNpc + c1=40 + msg19(14 值) +
//	   CantMove=1 + CantParry=1。CD/MP 此刻结、不扣轻功
//	   （g 3.06/2.45/1.48/1.30m 四样本；hyjf02 6.95m 直收）。
//	② 冲锋带（仅 3m 族：(直收带, GRushRange 5]，官服观察 3~5m 立即冲锋）：
//	   c61 授予簇 = c61[dest,flag,动作] → CantMove=1 → CantParry=1 →
//	   CD 双行 → MPRatio/MP（空中 +QingGongPoint、+40~80ms m707[29]）
//	   （技能4.pcapng 20.194 逐帧）→ 0x21 冲刺驱动（地面 25m/s + State=run
//	   跑步动画 / 空中 15m/s 奔袭不落地，ETA=距离/速度 到点释放）
//	   → 212[尾1] → 落地簇（双 0x1F、CD/MP 已结、State=idle 复位）。
//	   6m 族（castRange≥5）无冲锋带——<castRange 即直收，窗外客户端自走。
//
// msg19 定版 14 值：[19, obj攻, obj目, 技能, v, v, 打击点1xyz, 朝向, -2.5,
// 打击点2xyz]；朝向 = atan2(dx,dz) 指向目标；两打击点 = 玩家沿目标反方向
// 距目标恒 4.04m/4.55m 处（f/g 四样本 r+dist 逐位拟合）。
//
// 已知历史误判（详见 canonical 文档 §四/§五）不赘述；外服私服路径上的兜底
// （212 三分/6s 安全阀/过期释放）保留为静默兜底，仅客户端偏离外服私服方言
// 时触发并打日志。

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"sync"
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
	// rushAirborneAboveMeters：空中冲锋判定线（玩家 211 自报 y 高出目标
	// y 1.5m 即空中档——冲锋族 geometry 已用 211 自报目标坐标，两侧同为
	// 客户端坐标系，无 NPC 内部 y 口径问题；旧 1e9 禁用线针对服务端
	// 坐标基准时代，v29 撤销）。与 clientAirborneAbove/范围技高→低线同值。
	rushAirborneAboveMeters = float32(1.5)
	// rushGRushRangeMeters：玩家物理档案 GRushRange=5.0——3m 族冲锋带上界
	//（2026-11-16 官服观察定版：3m 技能在 3~5m 立即冲锋；技能4.pcapng
	// 玩家档案 LandRushSpeed=25/AirRushSpeed=15 分档同源）。
	rushGRushRangeMeters = 5.0
	// rushChargeHeightGateMeters 空/地混层闸（v30，2026-11-16 用户定版+
	// 二轮澄清）：|出招位置 y − 目标 y| 超过该值=角色/目标任一在高处
	//（角色空-目标地 / 目标空-角色地 / 双空）——近身也不原地直收（隔空
	// 打高处/低处目标），改走冲锋链受理即冲；≤1m 视为同层，距离分带与
	// v29 完全一致（平地带内照常地面冲锋，其它规则不动）。
	rushChargeHeightGateMeters = float32(1.0)
	// rushChargeUpliftAirborneMeters 低→高空中奔袭档阈值（v30 用户补充）：
	// 目标高于出招位置超过该值的冲锋走空中档（fastruning flag2 15m/s 奔袭
	// 动画，非地面跑步动画）；高→低维持 1.5m（rushAirborneAboveMeters，
	// 技能4.pcapng 实证）。范围冲锋（点地）同规则。
	rushChargeUpliftAirborneMeters = float32(0.5)
	// rushWalkKickDelay：受理簇到走踢帧的官方簇距（外服私服 f#0202 +155ms）。
	rushWalkKickDelay = 155 * time.Millisecond
)

// rushChargeAirborne 冲锋空中档判定（v30）：高→低 >1.5m（技能4.pcapng
// 俯冲实证）或 低→高 >0.5m（2026-11-16 用户补充：低地冲高地走空中奔袭
// 动画，非地面跑步）→ 空中档（fastruning flag2 / 15m/s / ETA 3D）。
func rushChargeAirborne(casterY, destY float32) bool {
	return casterY-destY > rushAirborneAboveMeters || destY-casterY > rushChargeUpliftAirborneMeters
}

// rushDashFallbackDelay：c61 发出后的落地哨首查延迟（外服私服无此机制；
// var 化供测试缩短）。c61 未被本客户端消费（2026-11-15 真机实证）时站立
// 按压会呆站——首查即兜底 bind 点直收；客户端已自行移动则让位 212 链。
var rushDashFallbackDelay = 1500 * time.Millisecond

// rushWalkSentinelInterval：走位驱动落地哨的复查间隔（var 化供测试缩短）。
var rushWalkSentinelInterval = 2500 * time.Millisecond

// rushWalkSentinelETASlack：落地哨首查 ETA 的富余量（distance/speed +
// slack；速度取 motion.moveSpeed 快照）。var 化供测试压缩。
var rushWalkSentinelETASlack = 1 * time.Second

// rushWalkRefreshInterval 步行段续拍间隔（官方 ~0.7s/拍取 500ms；var 化
// 供测试压缩）。
var rushWalkRefreshInterval = 500 * time.Millisecond

// rushEscapeRungDelay：rung1（skillrush 事件点 tick）到 rung2（逃逸门条目）
// 的间隔——v5 真机（16:11~16:12 三轮）证两者组合才能拖动主玩家（rung1
// 疑似负责点 skillrush 逐帧 tick、rung2 负责放行+写剩余距离），改近同拍
// （200ms）；rushEscapeRestoreDelay：LogicState 还原延迟——v5 的 800ms 还原
// 把拖动掐在 ~0.8s（「只拖一点距离」真凶），改按步行 ETA 推算（staging
// 处传入），落地的 LogicState=1 帧会自然覆盖。均 var 化供测试压缩。
var rushEscapeRungDelay = 200 * time.Millisecond

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

// rushWalkBandAcceptMeters：212 走带认可半径。外服私服 walk-in 直收的 212
// 可落在 castRange+1 内任意点（技能.pcapng：hyjf02 212@5.81/5.58 外服私服
// 认可落地——原走带全宽 5.0 对 6 米技能过窄，会把合法步行落地误判成
// 窗口外而驱动接近/取消）。
func rushWalkBandAcceptMeters(pending *pendingCastState) float64 {
	accept := rushGRushRangeMeters
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

// rushStopDistance 步行段停点距离（walk 族=castRange 下限 3；dash 族走
// 冲刺触发线另算）。
func rushStopDistance(castRange float32) float64 {
	stop := float64(castRange)
	if stop < rushArrivalRadiusMeters {
		stop = rushArrivalRadiusMeters
	}
	return stop
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
// v29 族规则（2026-11-16 官服观察定版）：
//   3m 族（castRange<5）：冲锋带=(直收带 castRange+0.5, GRushRange 5]——
//     官服观察「3m~5m 立即冲锋到 1~2m 处释放」；
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
			windowHi: rushGRushRangeMeters,
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

// noteApproachPhase 记录驱动接近当前段（phase timer 换段时更新），同时
// 复位打断闩（新一轮驱动开始）。
func (p *playerActor) noteApproachPhase(dest worldcore.Transform, speed float32) {
	p.mu.Lock()
	p.approachDest, p.approachSpeed, p.approachActive = dest, speed, true
	p.approachCancelled = false
	p.approachDashing = false
	p.mu.Unlock()
}

// approachPhaseSnapshot 当前驱动段快照。
func (p *playerActor) approachPhaseSnapshot() (worldcore.Transform, float32, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.approachDest, p.approachSpeed, p.approachActive
}

// cancelApproachPhase 取消驱动接近（v12：用户拍板允许手动打断）——原子
// 置 false + 置打断闩（v13：打断后旧 c61 瞬移兜底不得再把用户拽到 bind）。
// 返回此前是否在驱动中。
func (p *playerActor) cancelApproachPhase() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	was := p.approachActive
	p.approachActive = false
	if was {
		p.approachCancelled = true
	}
	return was
}

// noteApproachDashing 冲刺段闩（v14：dash 族步行段的 212 一律忽略——
// 客户端落地态噪声位置不可靠，真落地只认冲刺段后的带内 212/落地哨）。
func (p *playerActor) noteApproachDashing() {
	p.mu.Lock()
	p.approachDashing = true
	p.mu.Unlock()
}

func (p *playerActor) approachDashingSnapshot() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.approachDashing
}

// approachCancelledSnapshot 打断闩快照（issueRushDash 落地兜底门控）。
func (p *playerActor) approachCancelledSnapshot() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.approachCancelled
}

// clientAirborneAbove 已并入 rushChargeAirborne（v30：低→高 >0.5m 亦
// 空中档；原方法只识高→低 1.5m）。

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

// playerLocomotionStateFrame 主玩家移动状态行（0x10 State 属性；v8 真机
// 实证该通道在拖动期驱动本地玩家动画，"walk"=走/"run"=跑，客户端动画
// 状态机词汇见 lua form_test_action）。
func playerLocomotionStateFrame(state string) ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: npcStatePropertyIndex, Name: "State", Value: clientdata.StringValue(state)},
	})
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
	// msg25[30000,'skill_pause']（P0f 鉴别实验结论：撤除后 64006 轮仍零武装
	// 且体验更差，已回滚恢复）。
	// **stage 期不发 m24 c61**（技能1.pcapng 2026-11-13 定版：外服私服受理簇
	// 零 m24；c61 是服务端指令，仅在 m220 上报进技能授予窗时由
	// tryRushArrival→issueRushDash 下发——P0c/P0e 的受理期 stage-dash 档2
	// 系 Java 方言误植，真机实证客户端零消费且非外服私服帧序）。
	// msg25[30000,'skill_pause']（2026-11-15 曾撤、同日按令恢复：跳跃异常
	// 真凶=回放瘦身剥离外服物理帧档案（PHYSICS-16，另案已修），m25 无罪；
	// 远距按压的官方暂存 UX 保留）。
	// **stage 期不发 m24 c61**（技能1.pcapng 2026-11-13 定版：外服私服受理簇
	// 零 m24；c61 是服务端指令，仅在 m220 上报进技能授予窗时由
	// tryRushArrival→issueRushDash 下发——P0c/P0e 的受理期 stage-dash 档2
	// 系 Java 方言误植，真机实证客户端零消费且非外服私服帧序）。
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
	// 受理簇内的 msg25 同样要兜底配对（落地/取消/再受理的 m26 先到时
	// 看门狗只会多补一发无害的重复 m26）。
	player.armSkillPauseRelease(link, remote)
	// 走踢帧默认停发（NINEYIN_RUSH_WALK_KICK=1 显式开）：三方言真机全零走
	// （0xa0000f01 双同值 / 真方言 A/B 双点 0xA0000000+0x30000F01），且
	// 真方言会把客户端本地移动机打楔（跳不高/原地弹=服务端推的停帧与
	// 伪 WASD 条目被吞进本地 intent 状态机，2026-11-15 09:5x 真机）。
	// 客户端 OnMoving 闸（motion_func 白名单）未破之前不再自动发。
	if strings.EqualFold(strings.TrimSpace(os.Getenv("NINEYIN_RUSH_WALK_KICK")), "1") {
		scheduleRushWalkKick(link, player, definition, targetID, target, castAt, remote)
	}
	// v30 近身混层立即冲锋（2026-11-16 用户定版+二轮澄清）：直收带内但
	// |出招位 y−目标 y|>1m（空/地混层：角色空-目标地/目标空-角色地/双空）
	// ——原地普攻会隔空打高处/低处目标，受理簇后立即发冲锋（211 分流已
	// 保证此形态不再走 executeRushDirect；带外按压仍由客户端自走近、上报
	// 进带后 tryRushArrival 触发，同层 ≤1m 距离分带 v29 原样）。客户端
	// 自身飞行中扣住（airFlightActive 闸，与 tryRushArrival 同款；落地后
	// 行走上报重新触发）。
	if distance <= rushDirectCastMeters(definition) &&
		math.Abs(float64(castAt.Y-target.Y)) > float64(rushChargeHeightGateMeters) &&
		!player.airFlightActiveSnapshot() {
		rushDashDrive(link, player, world, target, castAt.X, castAt.Y, castAt.Z, remote)
	}
	// ★服务端驱动接近 v5（默认开，NINEYIN_RUSH_APPROACH=off 关闭）：本服
	// 登录方言下客户端无本地「按下即走」，远距受理后只会无限期挂起。已证伪
	// 通道：c61 受理即授予（15:55 六秒零位移）、0x21 白名单通用态 0x2B01
	//（15:39/15:41 两轮零移动——逆向二轮定案：通用态「只写不驱」，结构性
	// 死路）、v4 逃逸门+非官方槽值（15:49 零移动——s5 应为速度乘子 1.0~1.3
	// 非绝对速度 4.5，还原帧不能同拍发）。
	//
	// v5 两级梯子（逆向二轮配方，2026-11-15）：
	//   rung1 = 白名单态 0x44 skillrush（intent=0x2C01）：vt+0x250 flag==1
	//           写 dest/剩余距离后**额外发 skillrush 事件 0x71**→设
	//           skill_rush_type→0x8EA42C0 逐帧驱动——白名单里唯一自驱动
	//           的态，且语义就是技能冲刺。
	//   rung2 = 逃逸门官方式（1.5s 无位移再发）：LogicState=106 → 0x21
	//           intent=0x00000001 + 官方槽值（s5=1.2/s6=0/y 不取负）→
	//           事件 {21,17}+0x16 = 远程玩家同款驱动（官方大规模验证过的
	//           机制）→ +800ms 还原 LogicState（事件消费异步，同拍还原会被
	//           杀）。
	// 槽位定版（fxnet2 vt+0x140..178 线性）：s1/s2/s3=dest xyz、s4=朝向、
	// s5=速度乘子(官方 1.0~1.3)、s6=转向(0 可)、s7/s8=0、intent=flag|态<<8。
	// 落地哨兜底不变（位移推进→让位 212；ETA 后仍静止→bind 点直收）。
	return true, nil
}

// 0x21 主玩家驱动条目的 intent 槽值（逆向二轮配方）。
const (
	// rushSkillRushIntent：白名单态 0x44（skillrush）<<8 | flag 1——vt+0x250
	// 写完 dest 后对主玩家发 skillrush 事件 0x71，事件链设 skill_rush_type
	// 并启动逐帧驱动（白名单里唯一自驱动的态）。
	rushSkillRushIntent = uint32(0x44)<<8 | 1
	// rushEscapeDriveLogicState：OnMoving 白名单拒后放行分支要求的
	// LogicState 值（0x6A=106；游戏语义=LS_OFFLINEJOB，此处仅作位移放行
	// 开关，事后还原 0）。
	rushEscapeDriveLogicState byte = 0x6A
)

// rushDriveSpeedFallback 拖动速度兜底（motion.moveSpeed 缺省时）。
const rushDriveSpeedFallback = float32(3.9)

// rushDriveDashSpeedMeters 官方 0x21 冲刺同步条目的 s5 绝对速度（技能3
// .pcapng 8.876s 帧：c61 冲刺段 25.0；步行段 4.5）。
const rushDriveDashSpeedMeters = float32(25.0)

// rushDriveAirDashSpeedMeters 空中/高对低冲刺的 0x21 s5 速度（技能4.pcapng
// 分档定版：物理档案 AirRushSpeed=15 vs LandRushSpeed=25；同技 hyjf07 空中
// v平均值 17.8 vs 地面 24.9，空中耗时均值 599ms vs 地面 449ms——空中恒慢
// ~40%，垂直分量由 fastfall 主导（实测 v垂至 -22.3，非匀速冲刺））。
const rushDriveAirDashSpeedMeters = float32(15.0)

// rushDriveMoveIDStraightRun（弃用，v19 起）=13316=0x3404：态 0x34 ∈ 白名单
// 但 vt+0x250 flag4 分支「只写不驱」（不发动作事件、mover tick 不派发，
// 逆向终报+18:16 真机 7 秒零位移实证）。保留常量仅作历史对照。
const rushDriveMoveIDStraightRun = uint32(13316)

// rushDriveEntryFull 构造主玩家 0x21 单条目驱动帧——完整方言（技能3.pcapng
// 官方主玩家条目逐槽实证 2026-11-15 + 另一服 encodePresenceWaypointSelf 同构）：
// s1..s4=dest xyz/朝向、s5=速度（官方步行 4.5 / 冲刺 25.0 绝对值）、
// s6=转向符号（官方步行 -π / 冲刺 +2π）、s7(Motion2)=速度副本、
// s8(Motion3)=重力（地面 **9.9** 官方值 / 空中 0 交客户端插值）、
// intent=moveID（13316 白名单态 0x34 直线跑）。speed=0 即停止条目。
func rushDriveEntryFull(dest worldcore.Transform, speed float32, airborne bool) []byte {
	if speed < 0 {
		speed = 0
	}
	gravity := float32(9.9)
	rotate := float32(-math.Pi)
	if airborne {
		gravity = 0
		rotate = float32(2 * math.Pi)
	}
	return serverAllDest([]allDestEntry{{
		objectID: playerObjectID, ownerID: playerOwnerID,
		destX: dest.X, destY: dest.Y, destZ: dest.Z,
		destOrient: dest.Orient,
		speed:      speed,
		rotate:     rotate,
		motion2:    speed,
		motion3:    gravity,
		moveFlags:  1,
	}})
}

// normalizeOrient 朝向归一 [0,2π)（官方 s4 实测 0.13~5.94）。

// normalizeOrient 朝向归一 [0,2π)。
func normalizeOrient(o float32) float32 {
	for o < 0 {
		o += 2 * float32(math.Pi)
	}
	for o >= 2*float32(math.Pi) {
		o -= 2 * float32(math.Pi)
	}
	return o
}

// rushDriveEntryV13 v13 方言复刻单条目（用户令回归实测）：s1..s4=dest
// xyz/朝向、s5=速度、s6..s8=0、intent 参数（0x4401/0x00000001）——与
// normalizeOrient 朝向归一 [0,2π)（官方 s4 实测 0.13~5.94）。

// rushDriveEntryV13 v13 方言复刻单条目（用户令回归实测）：s1..s4=dest
// xyz/朝向、s5=速度、s6..s8=0、intent 入参（rung1=0x4401 / 逃逸门条目=1）
// ——v6~v13 唯一被真机证明有真实位移的形状（后加的 rotate/motion/对帧
// 槽位全部撤回，与当年逐字节一致）。
func rushDriveEntryV13(dest worldcore.Transform, speed float32, intent uint32) []byte {
	if speed <= 0 {
		speed = rushDriveSpeedFallback
	}
	return serverAllDest([]allDestEntry{{
		objectID: playerObjectID, ownerID: playerOwnerID,
		destX: dest.X, destY: dest.Y, destZ: dest.Z,
		destOrient: normalizeOrient(dest.Orient),
		speed:      speed,
		moveFlags:  intent,
	}})
}

// normalizeOrient 朝向归一 [0,2π)（官方 s4 实测 0.13~5.94）。

// rushDriveEntryV13 v13 方言复刻单条目（2026-11-15 用户令回归实测）：
// rushDriveEntryRotate：带显式转向槽的条目（冲刺 +2π 官方式）。
func rushDriveEntryRotate(dest worldcore.Transform, speed float32, rotate, gravity float32) []byte {
	return serverAllDest([]allDestEntry{{
		objectID: playerObjectID, ownerID: playerOwnerID,
		destX: dest.X, destY: dest.Y, destZ: dest.Z,
		destOrient: dest.Orient,
		speed:      speed,
		rotate:     rotate,
		motion2:    speed,
		motion3:    gravity,
		moveFlags:  1,
	}})
}

// rushDriveStopEntry 停止条目值（打断取消用，配对官方双条目帧）：
// dest=当前位、速度 0——必须真 0（驱动 tick 0m/s 停走）。
func rushDriveStopEntry(x, y, z, orient float32) allDestEntry {
	return allDestEntry{
		objectID: playerObjectID, ownerID: playerOwnerID,
		destX: x, destY: y, destZ: z, destOrient: normalizeOrient(orient),
		speed: 0, motion3: 9.9, moveFlags: 1,
	}
}

// rushApproachDashEnabled 服务端驱动接近开关（默认开；NINEYIN_RUSH_APPROACH=off 关闭）。
func rushApproachDashEnabled() bool {
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("NINEYIN_RUSH_APPROACH")), "off")
}

// scheduleRushWalkKick 受理簇后 +155ms 下发走踢帧（真方言双入口，见
// rushWalkKickFrame）。发送前复核暂存态仍是本次受理且未发冲刺——
// 155ms 内换了技能/取消/已进窗发 c61 的，旧踢帧作废。
func scheduleRushWalkKick(link sceneMessageConnection, player *playerActor, definition combatSkillDefinition,
	targetID uint32, target, castAt worldcore.Transform, remote string) {
	time.AfterFunc(rushWalkKickDelay, func() {
		pending := player.peekPendingCast(time.Now().UTC())
		if pending == nil || pending.skillID != definition.id || pending.targetID != targetID {
			return
		}
		if player.rushDashIssuedSnapshot() {
			return
		}
		bindX, bindY, bindZ := rushBindPoint(target, castAt.X, castAt.Z, definition.bindDistance)
		if err := link.WriteFrame(rushWalkKickFrame(castAt, target, definition.bindDistance)); err != nil {
			log.Printf("%s: rush walk kick skill id=%s: %v", remote, definition.id, err)
			return
		}
		log.Printf("%s: rush walk kick sent skill id=%s bind=(%.1f,%.1f,%.1f)",
			remote, definition.id, bindX, bindY, bindZ)
	})
}

// rushWalkKickFrame 冲锋走踢帧：双入口 0x21（44B 条目形，f.pcapng #69
// 逐槽三解定版 2026-11-15 凌晨——**2026-11-12「二轮勘误」系字节错位误读
// 全废**：两入口=当前位→目的地 A/B 双点（首轮解读才是对的），slot3=角度
// （非 4.689 常量）、尾双槽=0/0（非 5.5/9.8）、orient=+π（非 -π）、
// intent 按 entry 分族（非两入口同 0xa0000f01））：
//
//	entry0 停帧形：pos=当前位、angle=当前朝向、speed=0、orient=0、尾 0/0、
//	              intent=0xA0000000
//	entry1 走帧形：pos=bind 点、angle=atan2(dest−cur)（dx,dz）、speed=4.5、
//	              orient=+π、尾 0/0、intent=0x30000F01（WASD 键盘族）
//
// 走起后逐帧行进回显（#78 形）：单条目 speed=1.77 orient=+π intent=0xF01。
// 踢帧被客户端消费要求钟基正确（m294=m924 单路应答）。
// ⚠ 真机实证（09:2x）：真方言也会把客户端移动机打楔（跳不高/原地弹），
// 仅 NINEYIN_RUSH_WALK_KICK=1 显式开启时才发。
func rushWalkKickFrame(castAt, target worldcore.Transform, bind float32) []byte {
	bindX, bindY, bindZ := rushBindPoint(target, castAt.X, castAt.Z, bind)
	angle := float32(math.Atan2(float64(bindX-castAt.X), float64(bindZ-castAt.Z)))
	payload := make([]byte, 3, 3+2*44)
	payload[0] = 0x21
	binary.LittleEndian.PutUint16(payload[1:], 2)
	// entry0：停帧形（当前位 + 当前朝向 + speed 0）
	payload = binary.LittleEndian.AppendUint32(payload, playerObjectID)
	payload = binary.LittleEndian.AppendUint32(payload, playerOwnerID)
	payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(castAt.X))
	payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(castAt.Y))
	payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(castAt.Z))
	payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(castAt.Orient))
	payload = binary.LittleEndian.AppendUint32(payload, 0) // speed 0
	payload = binary.LittleEndian.AppendUint32(payload, 0) // orient 0
	payload = binary.LittleEndian.AppendUint32(payload, 0) // 尾槽 0
	payload = binary.LittleEndian.AppendUint32(payload, 0) // 尾槽 0
	payload = binary.LittleEndian.AppendUint32(payload, 0xa0000000)
	// entry1：走帧形（bind 点 + 移动角 + speed 4.5 + orient +π）
	payload = binary.LittleEndian.AppendUint32(payload, playerObjectID)
	payload = binary.LittleEndian.AppendUint32(payload, playerOwnerID)
	payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(bindX))
	payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(bindY))
	payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(bindZ))
	payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(angle))
	payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(4.5))
	payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(math.Pi)) // 外服样本 3.14=+π
	payload = binary.LittleEndian.AppendUint32(payload, 0)                         // 尾槽 0
	payload = binary.LittleEndian.AppendUint32(payload, 0)                         // 尾槽 0
	payload = binary.LittleEndian.AppendUint32(payload, 0x30000f01)                // WASD 键盘族
	return payload
}

// maybeBareWalkKickProbe 裸踢帧判别实验（NINEYIN_RUSH_KICK_PROBE=1）：
// 首个 m15 后 45s，无任何受理簇/msg25/PauseSkill 前置，单发一帧踢帧
// （目的地=玩家东侧 8m）。判别：客户端起走（m15 坐标推进）=踢帧机制
// 活着；零走=本客户端模式对本地玩家的 AllDest 服务端移动死路
// （2026-11-15 判别定案=死路，三方言+裸踢全零走）。默认关。
var bareKickProbeSeen sync.Map // *playerActor -> time.Time（首见 m15/哨兵）

func maybeBareWalkKickProbe(link sceneMessageConnection, player *playerActor, x, y, z float32, remote string) {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("NINEYIN_RUSH_KICK_PROBE")), "1") {
		return
	}
	now := time.Now()
	first, loaded := bareKickProbeSeen.LoadOrStore(player, now)
	if !loaded {
		return // 首见只登记
	}
	firstAt := first.(time.Time)
	if firstAt.After(now.Add(time.Hour)) {
		return // 哨兵=已发过
	}
	if now.Sub(firstAt) < 45*time.Second {
		return
	}
	castAt := worldcore.Transform{X: x, Y: y, Z: z}
	target := worldcore.Transform{X: x + 8 + defaultRushBindDistance, Y: y, Z: z}
	if err := link.WriteFrame(rushWalkKickFrame(castAt, target, defaultRushBindDistance)); err != nil {
		log.Printf("%s: bare walk kick probe: %v", remote, err)
		return
	}
	bareKickProbeSeen.Store(player, now.Add(24*time.Hour)) // 哨兵=已发
	log.Printf("%s: bare walk kick probe sent from=(%.1f,%.1f,%.1f) dest=(%.1f,%.1f,%.1f)",
		remote, x, y, z, x+8, y, z)
}

// noteRushStatePing 记录状态四连下发时刻（受理/直收簇内联发送后调用，
// 接近段重发按 ~3s 节流从此起算）。
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
				} else if rushApproachDashEnabled() {
					// v13：服务端驱动接近在途的远带 212 = 客户端落地态噪声
					//（17:22 真机：按压 1s 后 27.5m 外 212 抢跑出招）——静默
					// 忽略，真正的落地续招由停点处带内 212 / 落地哨完成。
					if _, _, active := player.approachPhaseSnapshot(); active {
						log.Printf("%s: ignore far dash landing during approach drive dist=%.1fm", remote, distance)
						return true, nil
					}
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
	maybeBareWalkKickProbe(link, player, x, y, z, remote)
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
// 锁定型直收下限同源的 3.5）。
const areaRushDirectCastMeters = 3.5

// areaRushTier2Action：范围型长距疾跑动画（b/e 8.2m 档2 方言）。
const areaRushTier2Action = "qg_fastruning_s_2h_01"

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

// issueAreaGroundRushDash 范围型点地冲刺（v28，2026-11-16 用户真机定版）：
// ①点选平面点=落点本体（高处可选低处平面——地面/屋顶，低处选高处须直线
//   无碰撞=客户端点选已校验、服务端直线驱动；单体锁定技的「目标身前 bind
//   1.5m」不适用于范围技）；②同平面/上行=25 高速跑（0x10 State=run 驱动
//   跑步动画——v8 实证拖动期动画走该通道，旧版未发=「立着滑行」根因）；
//   高→低落差>1.5m=空中档 15 俯冲（技能4.pcapng
//   Air/Land 分档，c61 flag2+A 类 anim）。ETA=3D 距离，落地=stop-reset+
//   State=idle+msg26+0x1F+msg337+还原；CD/MP 仍主出招路径即时结算。
func issueAreaGroundRushDash(link sceneMessageConnection, player *playerActor,
	definition combatSkillDefinition, castAt, ground worldcore.Transform, remote string) time.Duration {
	distance := math.Sqrt(float64(square(castAt.X-ground.X) + square(castAt.Z-ground.Z)))
	if distance <= areaRushDirectCastMeters {
		return 0
	}
	destX, destY, destZ := ground.X, ground.Y, ground.Z
	// 空中档判定（v30，用户补充「范围冲锋也是一样」）：castAt/ground 均
	// 211 客户端自报坐标（同一坐标系，无 NPC 内部 y 口径问题）。高→低
	// >1.5m（rushAirborneAboveMeters，技能4.pcapng 实证）或 低→高 >0.5m
	//（rushChargeUpliftAirborneMeters，低地冲高地=空中奔袭非地面跑步）
	// → 空中档 15m/s + c61 flag2；同平面/小落差=25 高速跑 + State=run。
	airborne := rushChargeAirborne(castAt.Y, ground.Y)
	speed := rushDriveDashSpeedMeters
	slack := 120 * time.Millisecond
	if airborne {
		speed = rushDriveAirDashSpeedMeters
		slack = 250 * time.Millisecond
	}
	dashDur := time.Duration(math.Sqrt(float64(square(destX-castAt.X)+square(destZ-castAt.Z)+square(destY-castAt.Y)))/float64(speed)*float64(time.Second)) + slack
	if escapeSet, err := objectSingleProperty(playerObjectID, playerOwnerID, propRushLogicState, "LogicState", clientdata.ByteValue(rushEscapeDriveLogicState)); err == nil {
		_ = link.WriteFrame(escapeSet)
	}
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
	if rushAnim != "" {
		if frame, err := dashRushFrame(destX, destY, destZ, rushFlag, rushAnim); err == nil {
			_ = link.WriteFrame(frame)
		}
	}
	if !airborne {
		// 高速跑动画：拖动期本地玩家动画走 0x10 State 行（v8 真机实证；
		// form_test_action 状态机词汇 idle/run，run 态速度插值带 blend）。
		if frame, err := playerLocomotionStateFrame("run"); err == nil {
			_ = link.WriteFrame(frame)
		}
	}
	dashDest := worldcore.Transform{X: destX, Y: destY, Z: destZ, Orient: orientTowardTarget(castAt.X, castAt.Z, ground.X, ground.Z)}
	driveRushDest(link, player, nil, castAt, dashDest, speed, airborne)
	airTag := ""
	if airborne {
		airTag = " air"
	}
	log.Printf("%s: area rush dash skill id=%s ground=(%.2f,%.2f,%.2f) dist=%.1fm speed=%.0f%s eta=%s (drive dialect)",
		remote, definition.id, ground.X, ground.Y, ground.Z, distance, speed, airTag, dashDur)
	dest := dashDest
	time.AfterFunc(dashDur, func() {
		steps := []struct {
			name  string
			build func() ([]byte, error)
		}{
			{"stop-reset", func() ([]byte, error) { return rushDriveEntryV13(dest, 0, 1), nil }},
			{"msg26", func() ([]byte, error) { return skillPauseClearFrame() }},
			{"0x1F", func() ([]byte, error) { return serverLocation(playerObjectID, playerOwnerID, dest), nil }},
			{"msg337", func() ([]byte, error) { return casterTransformSyncFrame(dest.X, dest.Y, dest.Z, dest.Orient) }},
			{"restore", func() ([]byte, error) {
				return objectSingleProperty(playerObjectID, playerOwnerID, propRushLogicState, "LogicState", clientdata.ByteValue(0))
			}},
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
		if !airborne {
			// 复位跑步动画（form_test_action 复位即 control.State="idle"）。
			if frame, err := playerLocomotionStateFrame("idle"); err == nil {
				_ = link.WriteFrame(frame)
			}
		}
	})
	return dashDur
}

// tryRushArrival 移动上报驱动的冲刺探测：仅当距目标进入冲刺窗口
// (GRushRange 5.0, +Add 6.5] 时发 c61；低于下界由客户端自行走进走带
// 发 212 落地。冲刺已发出后静默——落地只认 212。
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
	// v13：服务端驱动接近在途时不做窗口 c61 授予——c61 对本客户端已证伪
	//（零消费），且授予置 dashIssued 会劫持两段式流（phase timer/落地哨
	// 双双早退，17:21 真机实证 dash-timeout-fallback 抢跑瞬移）。
	if rushApproachDashEnabled() {
		if _, _, active := player.approachPhaseSnapshot(); active {
			return
		}
	}
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
	// v30 二轮勘定（2026-11-16 用户澄清）：高度规则只作用于空/地混层
	//（角色空-目标地 / 目标空-角色地 / 双空，可测信号=|Δy|>1m）；双方
	// 同层地面（|Δy|≤1m）距离分带行为与 v29 完全一致——平地带内照常
	// 地面冲锋，本处不再设闸。
	// v29 分档带（staging 时按方言表/族规则填入 pending.dashWindowLo/Hi）：
	// 3m 族=(直收带, GRushRange 5]——进入即驱动冲刺到目标身前 bind
	//（1~1.5m，空中终点 y+1.0），ETA 到点释放；6m 族=无带（hi<=0 直接
	// 返回——无冲锋段，客户端跑到 <castRange 重按直收或停点 212 落地
	// 续招）。方言表技能（xytf02/kfkj02 等）用表窗。
	if pending.dashWindowHi <= 0 || pending.dashWindowLo <= 0 {
		return
	}
	if distance > pending.dashWindowHi || distance <= pending.dashWindowLo {
		return
	}
	// 客户端自身飞行（穿云纵等）进行中：扣住不发逃逸门拖动——服务端
	// 0x21 拖动与其自身空中冲刺并发会打断移动状态机，穿云纵后续卡死
	//（2026-11-16 真机确定性复现；落地后 state4/5 复位，随后的行走上报
	// 重新进带触发地面档冲刺，pending 由 30s 窗兜底）。
	if player.airFlightActiveSnapshot() {
		log.Printf("%s: rush dash hold during client flight skill id=%s dist=%.1fm", remote, pending.skillID, distance)
		return
	}
	log.Printf("%s: rush within dash band skill id=%s dist=%.1fm in (%.1f, %.1f]",
		remote, pending.skillID, distance, pending.dashWindowLo, pending.dashWindowHi)
	rushDashDrive(link, player, world, geometry, x, y, z, remote)
}

// rushAirDiagonalStep 空中奔袭斜线路标的下发间隔（var 化供测试压缩）。
var rushAirDiagonalStep = 80 * time.Millisecond

// driveRushDest 冲锋位移条目下发（v30.1 斜线路标驱动，2026-11-16 用户
// 反馈「空中奔袭=斜线冲向目标，非粗暴直拉/直接拉到目标近前」）：单条目
// 直写终点时垂直分量无过渡——客户端 self-mover 平面插值，y 全靠落地
// 0x1F 锚一次吞掉=瞬移观感。空中档改为沿起→终点线性插值逐段发路标：
// t=0 即发第一档，此后每 step 一档（前瞻一档让 mover 恒有距离可吃），
// y 随平距同步爬升/俯冲=斜线奔袭；ETA 落地锚与末档终点重合、无残差
// 瞬移。地面档（同层无 y 差）与超短程（≤step）维持单条目。逐段重发
// 只用已证真的 v13 条目形状，与官方 NPC 0x21 移动流同构；pending 非
// nil 时被 212 落地/取消/rearm 收走即停发。
func driveRushDest(link sceneMessageConnection, player *playerActor, pending *pendingCastState,
	start, dest worldcore.Transform, speed float32, airborne bool) {
	if !airborne {
		_ = link.WriteFrame(rushDriveEntryV13(dest, speed, 1))
		return
	}
	pureSec := math.Sqrt(float64(square(dest.X-start.X)+square(dest.Z-start.Z)+square(dest.Y-start.Y))) / float64(speed)
	if pureSec <= rushAirDiagonalStep.Seconds() {
		_ = link.WriteFrame(rushDriveEntryV13(dest, speed, 1))
		return
	}
	n := int(math.Ceil(pureSec / rushAirDiagonalStep.Seconds()))
	waypoint := func(k int) worldcore.Transform {
		t := float32(k) / float32(n)
		return worldcore.Transform{
			X: start.X + (dest.X-start.X)*t,
			Y: start.Y + (dest.Y-start.Y)*t,
			Z: start.Z + (dest.Z-start.Z)*t,
			Orient: dest.Orient,
		}
	}
	_ = link.WriteFrame(rushDriveEntryV13(waypoint(1), speed, 1))
	for k := 2; k <= n; k++ {
		k := k
		time.AfterFunc(time.Duration(float64(k-1)*pureSec/float64(n)*float64(time.Second)), func() {
			if pending != nil && player.peekPendingCast(time.Now().UTC()) != pending {
				return
			}
			_ = link.WriteFrame(rushDriveEntryV13(waypoint(k), speed, 1))
		})
	}
}

// rushDashDrive 冲刺驱动（v24 终版）：逃逸门三明治 [LogicState=106] +
// [0x21 intent=1 s5=25 dest=bind]（唯一真位移方言），106 保持到 ETA+富余
// （冲刺不可打断=用户定版）；ETA=距离/25 到点直接 resumePendingCastAt 释放
// （1~1.5m，空中 y+1.0）。CD/MP/轻功此刻结（官方 c61 时点语义）。
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
	destY := geometry.Y
	if airborne && fromY > geometry.Y {
		// 高→低俯冲：到点 +1.0 小跳悬停弧线（官方）；低→高：终点=目标
		// 所在平面本体（再抬 1.0 会冲过平台面悬空）。
		destY = geometry.Y + rushDashLiftMeters
	}
	// v27（技能4.pcapng 分档）：空中/高对低走 AirRushSpeed=15、ETA 取 3D
	// 距离（垂直分量 fastfall 主导，官方空中段实测 0.22~0.89s）；地面维持
	// 25 平面 ETA。slack：地面 120ms 保持；空中 250ms（官方 27/27 释放=
	// m212+39~78ms，此 ETA 仅作 212 未到时的兜底）。
	speed := rushDriveDashSpeedMeters
	slack := 120 * time.Millisecond
	if airborne {
		speed = rushDriveAirDashSpeedMeters
		slack = 250 * time.Millisecond
	}
	dashDist := math.Sqrt(float64(square(bindX-fromX) + square(bindZ-fromZ)))
	dashDur := time.Duration(math.Sqrt(dashDist*dashDist+
		float64(square(destY-fromY)))/float64(speed)*float64(time.Second)) + slack
	if escapeSet, err := objectSingleProperty(playerObjectID, playerOwnerID, propRushLogicState, "LogicState", clientdata.ByteValue(rushEscapeDriveLogicState)); err == nil {
		_ = link.WriteFrame(escapeSet)
	}
	// v25：官方式 c61 动作帧（技能3.pcapng：c61 与 0x21 冲刺同步成对，flag1=
	// G 类 qg_fastrun_{W}h_01 地面 / flag2=A 类 qg_fastruning_s_{W}h_01 空中；
	// 动作名按技能 skill_lock_data 的 LockG/LockARushAction）。
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
	} else {
		// 地面静止起冲=跑步动作（官服观察；v28 范围技真机定版同通道——
		// 0x10 State=run 驱动本地玩家跑步动画，0x21 s5=25 高速位移）。
		if frame, err := playerLocomotionStateFrame("run"); err == nil {
			_ = link.WriteFrame(frame)
		}
	}
	dashDest := worldcore.Transform{X: bindX, Y: destY, Z: bindZ, Orient: orientTowardTarget(bindX, bindZ, geometry.X, geometry.Z)}
	driveRushDest(link, player, pending, worldcore.Transform{X: fromX, Y: fromY, Z: fromZ}, dashDest, speed, airborne)
	player.markRushDashIssued(airborne)
	airTag := ""
	if airborne {
		airTag = " air"
	}
	log.Printf("%s: rush dash drive dest=(%.2f,%.2f,%.2f) speed=%.0f%s eta=%s (escape dialect)",
		remote, bindX, destY, bindZ, speed, airTag, dashDur)
	time.AfterFunc(dashDur, func() {
		if player.peekPendingCast(time.Now().UTC()) != pending {
			return // 已 212 落地/取消/rearm（轻功释放帧由 resume/cancel 出口发）
		}
		// v26：轻功接近中触发冲刺的落地重置——官方式 qinggong 喂帧终止符
		// 释放轻功绑定并重置状态（仅空中档；2026-11-16 起所有出口统一走
		// writeQinggongReleaseFrame）。
		if airborne {
			writeQinggongReleaseFrame(link, bindX, geometry.Y, bindZ)
		}
		if restore, err := objectSingleProperty(playerObjectID, playerOwnerID, propRushLogicState, "LogicState", clientdata.ByteValue(0)); err == nil {
			_ = link.WriteFrame(restore) // 冲刺结束关逃逸门（落地簇 LogicState=1 随后覆盖）
		}
		if !airborne {
			// v29：地面档落地复位跑步动画（form_test_action 复位即
			// control.State="idle"；与授予簇 State=run 配对）。
			if frame, err := playerLocomotionStateFrame("idle"); err == nil {
				_ = link.WriteFrame(frame)
			}
		}
		live := geometry
		if t, ok := world.combatNPCTransform(pending.targetID); ok {
			live = t
		}
		bindX2, _, bindZ2 := rushBindPoint(live, fromX, fromZ, pending.bindDistance)
		orient := orientTowardTarget(bindX2, bindZ2, live.X, live.Z)
		if err := resumePendingCastAt(link, player, world, bindX2, live.Y, bindZ2, orient, "dash-drive-landing", remote); err != nil {
			log.Printf("%s: rush dash drive landing: %v", remote, err)
		}
	})
}

func square(value float32) float32 { return value * value }

// writeQinggongReleaseFrame 轻功绑定释放帧（v26 官方式终结符：0x21 空置
// 条目 speed=0/motion3=9.9/intent=0xA0000700，另一服 self_waypoint 同款）。
// 空中档冲刺的所有出口都必须发（2026-11-16「穿云纵后续有概率用不出」
// 根因：该帧原只在 ETA 落地路径发，客户端 212 抢先到→resumePendingCastAt
// 出招后 AfterFunc 早退→轻功状态残留锁死后续轻功）。
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
