package main

// 冲锋全链官服帧级定版（f/g/b.pcapng 实证重写，2026-11-08，
// canonical 文档 §3.2.2——距离四桶模型）：
//
//	① 直收带 ≤ max(bind+2, 3.5)m：不走暂存链。直接执行簇 =
//	   [0x12 清扫过期 CD 行] + CurSkillID + CurSkillTarget + msg284[] +
//	   NPC.LogicState=1 + 0x11 rec114 fight_relation 行0 + 玩家.LogicState=1 +
//	   CD 双行(0x11 空行+0x13 逐格) + MPRatio/MP + 0x1F + msg337 +
//	   LastWGAttackNpc + c1=40 + msg19(14 值) + CantMove=1 + CantParry=1。
//	   CD/MP 此刻结、不扣轻功；无 msg25/26/707/c61/212/active_skill 行/msg20
//	   （g 3.06/2.45/1.48/1.30m 四样本；b 4.41m 走暂存钉死带边界）。
//	② 3.5m < 距离 ≤ 5.0(GRushRange)：受理簇暂存 → 客户端走近 →
//	   不发 c61（低于冲刺窗口）→ 走进 ≤3m 客户端发 212[尾0] → 落地簇
//	   （PauseSkill=""+msg26 前导、单 0x1F、CD/MP 落地结）（b 4.41m 实证）。
//	③ 冲刺窗口 (5.0, 6.5]：c61 冲刺簇 = c61 + CantMove/CantParry 置1 +
//	   CD 双行 + MPRatio/MP + QingGongPoint（CD/MP/轻功此刻结）
//	   → 客户端冲刺 → 212[尾1] → 落地簇（双 0x1F、CD/MP 已结）
//	   （f 5.71 / g 5.54 / b 5.9 触发、b 3.61 不触发钉死窗口）。
//	④ > 6.5m：同 ③ 但先走一段（客户端 ~6.2m 起自发 msg220 密报）。
//
// msg19 定版 14 值：[19, obj攻, obj目, 技能, v, v, 打击点1xyz, 朝向, -2.5,
// 打击点2xyz]；朝向 = atan2(dx,dz) 指向目标；两打击点 = 玩家沿目标反方向
// 距目标恒 4.04m/4.55m 处（f/g 四样本 r+dist 逐位拟合）。
//
// 已知历史误判（详见 canonical 文档 §四/§五）不赘述；官服路径上的兜底
// （212 三分/6s 安全阀/过期释放）保留为静默兜底，仅客户端偏离官服方言
// 时触发并打日志。

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
	clientCustomRushHitConfirm  int32 = 915 // 冲锋命中确认上报（官服无回应，静默消费）
	clientCustomCorpseQuery     int32 = 990 // 尸体拾取查询（同上，g 战后实证）
	clientCustomMapNavigate     int32 = 266 // 地图寻路标记（技能.pcapng：起跑前 ~73ms 一次，S2C 零回应）

	// 三层距离模型（g/f/b 联合定版 + Java SkillCastHandler 常量同源）。
	selectedRangeFallbackMeters = 8.0                // 无 TSP2/形状半径时的近战回落射程（skill_catalog 引用）
	rushStageMaxMeters          = 15.0               // RushRange 缺列时的回落（skill_catalog 引用，仅存档参考）
	rushArrivalRadiusMeters     = 3.0                // 落地到达半径（212 上报距目标 ≤3m = 落地）
	rushDirectCastSlackMeters   = 2.0                // bind 之上的直收余量
	rushPendingWindow           = 30 * time.Second   // 暂存出招有效期（官服 msg25 窗口=30000ms）
	defaultRushBindDistance     = float32(1.5)       // skill_lock_data 缺行时的身前绑定距离
	defaultGroundRushAction     = "qg_fastrun_2h_01" // 官服地面锁定冲锋动画
	convertDashLevel            = 1                  // 冲刺档位（官服锁定冲锋 c61=1）
	msgRushCommand              = 61                 // msg24 c1=61 冲刺/疾跑指令
	// rushDashLiftMeters：c61 终点在玩家上报 y 之上的抬升（恒 +1.0 小跳弧线）。
	rushDashLiftMeters = float32(1.0)
	// rushDashTimeout：c61 发出后等待 212 的静默安全阀（官服无此机制）。
	rushDashTimeout = 6 * time.Second
	// rushAirborneYDeltaMeters：空中冲锋判定（world 内部 NPC y 口径未校准，
	// 禁用中——一律走地面链）。
	rushAirborneYDeltaMeters = float32(1e9)
)

// rushDashTriggerMeters 是 c61 冲刺窗口上界：GRushRange(5.0)+GRushRangeAdd(1.5)。
func rushDashTriggerMeters() float64 { return 5.0 + 1.5 }

// rushDashWindowLoMeters 是 c61 冲刺窗口下界：GRushRange（再近冲刺无意义，
// b 4.41m 样本 220 上报 3.61m 不触发实证）。
func rushDashWindowLoMeters() float64 { return 5.0 }

// rushDirectCastMeters 是直收带上界：≤该距离不走暂存链直接执行。
// 三源合成：bind+2（g 3.06m 直收 / b 4.41m 暂存钉死边界∈(3.06,4.41]，
// Java 同源）；**castRange+1**（技能.pcapng 定版：hyjf02 6.95m 按下当毫秒
// msg284 直收簇、kfkj02 1.19m 同形——按下即执行窗口随面板射程走）；
// 下限 3.5。
func rushDirectCastMeters(definition combatSkillDefinition) float64 {
	near := float64(definition.bindDistance) + rushDirectCastSlackMeters
	if bound := float64(definition.castRange) + 1; near < bound {
		near = bound
	}
	if floor := rushArrivalRadiusMeters + 0.5; near < floor {
		near = floor
	}
	return near
}

// rushWalkBandAcceptMeters：212 走带认可半径。官服 walk-in 直收的 212
// 可落在 castRange+1 内任意点（技能.pcapng：hyjf02 212@5.81/5.58 官服
// 认可落地——原走带全宽 5.0 对 6 米技能过窄，会把合法步行落地误判成
// 窗口外而驱动接近/取消）。
func rushWalkBandAcceptMeters(pending *pendingCastState) float64 {
	accept := rushDashWindowLoMeters()
	if bound := float64(pending.castRange) + 1; accept < bound {
		accept = bound
	}
	return accept
}

// rushDashDialect：per-skill c61 授予方言（技能1.pcapng 16 次授予观测，
// 2026-11-13 定版，docs/技能1pcapng多套路冲锋官服方言.md §1.2/§1.3）。
// c61 是服务端指令，官服在 m220 上报距目标进入该技能授予窗时 +30~90ms
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

// rushDashDialectFor 解析技能授予方言（未列回落默认 G 类窗）。
func rushDashDialectFor(skillID string) rushDashDialect {
	if dialect, ok := officialRushDashDialects[skillID]; ok {
		return dialect
	}
	return rushDashDialect{windowLo: rushDashWindowLoMeters(), windowHi: rushDashTriggerMeters()}
}

// 冲锋簇属性索引（按名映射自本服表；官服现行表同名字 +2 偏移）。
var (
	propRushCantMove        = clientdata.MustPlayerProperty("CantMove").Index        // CantMove Byte（fork=203）
	propRushPauseSkill      = clientdata.MustPlayerProperty("PauseSkill").Index      // PauseSkill String（fork=220）
	propRushLogicState      = clientdata.MustPlayerProperty("LogicState").Index      // LogicState Byte（fork=463）
	propRushFP              = clientdata.MustPlayerProperty("FP").Index              // FP Int32（伤害段 FP/SP 对）（fork=462）
	propRushLastWGAttackNpc = clientdata.MustPlayerProperty("LastWGAttackNpc").Index // LastWGAttackNpc String（直收簇宣告目标模型）（fork=867）

	propQingGongPointIndex = clientdata.MustPlayerProperty("QingGongPoint").Index // QingGongPoint Int32（c61 扣后单帧）（fork=692）
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
	p.mu.Unlock()
}

func (p *playerActor) rushInFlightSnapshot() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rushInFlight
}

func (p *playerActor) markRushDashIssued() {
	p.mu.Lock()
	p.rushDashIssued = true
	p.mu.Unlock()
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

// rushPressGate 冲锋族按压入口闸：beginSkillUse 非 settled 分支的锁招/
// 个人 CD / 公共 GCD 三闸只读版。冲锋链在 211 入口提前分叉（直收 return /
// 暂存挂起），不补这道闸则四个结算点均无 CD 校验（commitRushDashCosts
// 只写不查）。MP/SP/轻功不在此查——仍按官服模型留在执行提交点。
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
// 落地官服均不扣轻功点——g 直收簇无 QingGongPoint 帧、b 步行落地同）。
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
		// 尽力扣，扣不动跳过。官服 c61 簇的 QG 扣是表现帧，无不足拒绝样本。
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

// objectSingleProperty 任意对象的 n=1 单属性 0x10 帧（官服簇逐帧单属性，
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
	dx := x - target.X
	dz := z - target.Z
	horiz := math.Sqrt(float64(dx*dx + dz*dz))
	if horiz < 0.01 {
		dx = 1
		horiz = 1
	}
	return target.X + float32(float64(dx)/horiz)*bind, target.Y, target.Z + float32(float64(dz)/horiz)*bind
}

// orientTowardTarget 官服朝向公式：atan2(dx, dz) 指向目标（msg337/msg19
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

// segmentDamageOpenFrame 即 msg19 段伤害起始帧（官服 14 值）：
//
//	[19, obj(攻), obj(目), str 技能, i32 v, i32 v,
//	 f32 打击点1 x,y,z, f32 朝向, f32 -2.5, f32 打击点2 x,y,z]
//
// 打击点=玩家沿「目标→玩家」延长线、距目标恒 4.04/4.55m（官服拟合常数）。
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

// ---- 211 暂存受理（桶②③④） ----

// stageRushCast 冲锋暂存受理（官服序，g/f 实证）：
//
//	首发：[0x12 清扫过期 CD 行] + CurSkillID + CurSkillTarget + msg25 +
//	      PauseSkill + msg707[15]
//	重受理：CurSkillID="" + msg26 + PauseSkill="" + [0x12 清扫] +
//	      CurSkillID + msg25 + PauseSkill + msg707[15]（不重发 CurSkillTarget）
//
// 受理与距离解耦；拉回/走动期间服务端零帧零取消，唯一出口=重按或 30s 过期。
func stageRushCast(link sceneMessageConnection, player *playerActor, definition combatSkillDefinition,
	targetID, targetOwner uint32, target worldcore.Transform, castAt worldcore.Transform,
	distance float64, remote string) (bool, error) {
	now := time.Now().UTC()
	airborne := castAt.Y-target.Y > rushAirborneYDeltaMeters
	rearm := player.peekPendingCast(now) != nil
	dialect := rushDashDialectFor(definition.id)
	dashLevel := dialect.level
	if dashLevel == 0 {
		dashLevel = convertDashLevel
	}
	dashAction := dialect.action
	if dashAction == "" {
		dashAction = definition.groundRushAction
	}
	player.stagePendingCast(&pendingCastState{
		skillID: definition.id, targetID: targetID, targetOwner: targetOwner,
		expiresUTC: now.Add(rushPendingWindow),
		canRush:    definition.canRush, castRange: definition.castRange,
		bindDistance: definition.bindDistance, groundRushAction: definition.groundRushAction,
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
	if rearm {
		builders = append(builders,
			frameBuilder{"CurSkillID-clear-rearm", func() ([]byte, error) { return curSkillIDFrame("") }},
			frameBuilder{"msg26-rearm", func() ([]byte, error) { return skillPauseClearFrame() }},
			frameBuilder{"PauseSkill-clear-rearm", func() ([]byte, error) {
				return rushPropertyFrame(propRushPauseSkill, "PauseSkill", clientdata.StringValue(""))
			}},
		)
	}
	// 惰性清扫过期 CD 行（官服在受理簇头发 0x12，f 7.764/15.649 实证）。
	for _, sweep := range player.sweepExpiredCooldownRows(now) {
		builders = append(builders, frameBuilder{"cd-sweep", func() ([]byte, error) { return sweep, nil }})
	}
	// 首受理（非 rearm）官方方言（技能.pcapng 32.642 定版逐帧，2026-11-12
	// 复核勘误）：**CurSkillID(462)=staged 技能串 → CurSkillTarget(629) →
	// msg25(25,30000,'skill_pause') → msg707[15] → 四连**，+155ms 双入口
	// 0x21 走踢帧（见 stageRushCast 尾部）。旧结论「首受理不发 CurSkillID」
	// 系对 f#0194 的误读（其 idx462 实为 'CS_jh_hyjf02\0'=staged 技能）；
	// 当年「多发致施法中不自走」的对照实验缺的是走踢帧，CurSkillID 被错怪。
	// PauseSkill 首受理仍不发（32.6-33.8 全窗无该行），重受理（f.pcapng
	// P 系列）官方才发清+设全家桶，见上方 rearm 分支。
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
	// **stage 期不发 m24 c61**（技能1.pcapng 2026-11-13 定版：官服受理簇
	// 零 m24；c61 是服务端指令，仅在 m220 上报进技能授予窗时由
	// tryRushArrival→issueRushDash 下发——P0c/P0e 的受理期 stage-dash 档2
	// 系 Java 方言误植，真机实证客户端零消费且非官服帧序）。
	builders = append(builders,
		frameBuilder{"msg25", func() ([]byte, error) { return skillPauseFrame(int32(rushPendingWindow / time.Millisecond)) }},
	)
	if rearm {
		builders = append(builders, frameBuilder{"PauseSkill-set", func() ([]byte, error) {
			return rushPropertyFrame(propRushPauseSkill, "PauseSkill", clientdata.StringValue(definition.id))
		}})
	}
	builders = append(builders, frameBuilder{"msg707", func() ([]byte, error) { return rushCastStateFrame() }})
	// 状态四连（586/645/646/750）：恢复。此前「暂撤」的对照推理（55052
	// 无四连会走 / 64785 有四连不自走）因果倒置——那批不自走会话处于
	// 受理簇方言错位期。技能.pcapng 官服三证（32.642/415.577/1290.300）
	// 钉死：官服每次受理簇后 +42ms 必发四连，且客户端同毫秒起跑——
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
	return true, nil
}

// rushWalkKickFrame 冲锋走踢帧：双入口 0x21（44B playerMoveEcho 条目形，
// 技能.pcapng f#0202 逐槽精解 2026-11-12 二轮勘误）：**两入口=同一目的
// 地**（官服样本 947.8/947.9≈bind 点重复两次，非「当前位→目的地」——
// 一轮的 A→B 双点解读系字节错位误读），angle/speed/orient/尾槽两入口
// 全同；speed=4.5、orient 恒 **-π**（官服两样本 -3.14，非朝向）、尾双槽
// 5.5/**9.8**(0x411CCCCD)、intentID 两入口均 0xa0000f01（寻路族）。
// 官服踢后 0.485s 客户端即起走（其间 s2c 零帧）——踢帧单独触发走位。
// 注意：踢帧被客户端消费要求钟基正确（m294=m924 单路应答），错方言
// 下踢帧被当回显吃掉不执行（凌晨三轮实证）。
func rushWalkKickFrame(castAt, target worldcore.Transform, bind float32) []byte {
	bindX, bindY, bindZ := rushBindPoint(target, castAt.X, castAt.Z, bind)
	payload := make([]byte, 3, 3+2*44)
	payload[0] = 0x21
	binary.LittleEndian.PutUint16(payload[1:], 2)
	for i := 0; i < 2; i++ {
		payload = binary.LittleEndian.AppendUint32(payload, playerObjectID)
		payload = binary.LittleEndian.AppendUint32(payload, playerOwnerID)
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(bindX))
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(bindY))
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(bindZ))
		payload = binary.LittleEndian.AppendUint32(payload, 0x4096268F) // slot3 官服四样本恒 4.689（常量槽非角度，三轮真机+字节 diff 定版）
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(4.5))
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(-math.Pi)) // 官服恒 -π
		payload = binary.LittleEndian.AppendUint32(payload, 0x40B00000)                // 5.5 官服恒定
		payload = binary.LittleEndian.AppendUint32(payload, 0x411CCCCD)                // 9.8 官服恒定
		payload = binary.LittleEndian.AppendUint32(payload, 0xa0000f01)                // 寻路族 intentID 两入口同值
	}
	return payload
}

// noteRushStatePing 记录状态四连下发时刻（受理/直收簇内联发送后调用，
// 接近段重发按 ~3s 节流从此起算）。
func (p *playerActor) noteRushStatePing(now time.Time) {
	p.mu.Lock()
	p.rushStatePingAt = now
	p.mu.Unlock()
}

// ---- 直收执行簇（桶①） ----

// executeRushDirect 近距直收执行（官服 g 定版，无暂存链）：
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
		geometry: geometry, nominalDamage: definition.baseDamage,
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
	// 官服直收簇逐帧序（g 22.157 定版）：[0x12 清扫] → CurSkillID →
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
	// 状态四连（官服 367.971 直收簇 → 368.020 +49ms 四连实证；与受理簇
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

// issueRushDash 最终冲刺指令（官服 c61 簇序，f 42.948 定版）：
// c61 → CantMove=1 → CantParry=1 → CD 双行（官方方言）→ MPRatio → MP →
// QingGongPoint。终点 y=玩家上报 y+1.0；此刻不发 0x1F（官服锚只在落地簇）。
func issueRushDash(link sceneMessageConnection, player *playerActor, target worldcore.Transform,
	fromX, fromY, fromZ float32, remote string) error {
	if player.rushDashIssuedSnapshot() {
		return nil
	}
	pending := player.peekPendingCast(time.Now().UTC())
	if pending == nil {
		return nil
	}
	destX, _, destZ := rushBindPoint(target, fromX, fromZ, pending.bindDistance)
	// dest y=目标 y+1.0（技能1.pcapng 四样本拟合：#2 target38.6→39.59、
	// #5 39.9→40.91、#3 38.6→39.37、#17 4.3→5.33；旧「玩家 y+1.0」系
	// f 平地样本巧合，坡地会偏）。
	destY := target.Y + rushDashLiftMeters
	now := time.Now().UTC()
	if err := player.commitRushDashCosts(pending, now, true); err != nil {
		log.Printf("%s: rush dash cost settle failed skill id=%s: %v", remote, pending.skillID, err)
		cancelPendingCast(link, player, fromX, fromY, fromZ, 0, remote)
		return nil
	}
	state := player.stateSnapshot()
	mpRatio := int32(0)
	if state.MaxMP > 0 {
		mpRatio = state.MP * 100 / state.MaxMP
	}
	steps := []struct {
		name  string
		build func() ([]byte, error)
	}{
		{"c61-dash", func() ([]byte, error) {
			level := pending.dashLevel
			if level == 0 {
				level = convertDashLevel
			}
			action := pending.dashAction
			if action == "" {
				action = pending.groundRushAction
			}
			return dashRushFrame(destX, destY, destZ, level, action)
		}},
		{"CantMove-set", func() ([]byte, error) {
			return rushPropertyFrame(propRushCantMove, "CantMove", clientdata.ByteValue(1))
		}},
		{"CantParry-set", func() ([]byte, error) {
			return rushPropertyFrame(propCantParry, "CantParry", clientdata.Int32Value(1))
		}},
	}
	for _, step := range steps {
		frame, err := step.build()
		if err != nil {
			return fmt.Errorf("encode rush dash %s: %w", step.name, err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return err
		}
	}
	if err := player.writeOfficialCooldownRows(link, pending.cooldownCategory, pending.cooldownTeam,
		pending.personalCD, pending.publicCD, now); err != nil {
		return fmt.Errorf("write rush dash cooldown rows %s: %w", pending.skillID, err)
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
		{"QingGongPoint", func() ([]byte, error) {
			return rushPropertyFrame(propQingGongPointIndex, "QingGongPoint", clientdata.Int32Value(state.QingGongPoint))
		}},
	}
	for _, step := range tail {
		frame, err := step.build()
		if err != nil {
			return fmt.Errorf("encode rush dash %s: %w", step.name, err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return err
		}
	}
	player.markRushDashIssued()
	time.AfterFunc(rushDashTimeout, func() {
		if !player.rushInFlightSnapshot() || !player.rushDashIssuedSnapshot() {
			return
		}
		if player.peekPendingCast(time.Now().UTC()) != pending {
			return
		}
		player.clearRushInFlight()
		release := []struct {
			name  string
			build func() ([]byte, error)
		}{
			{"CantMove-clear", func() ([]byte, error) {
				return rushPropertyFrame(propRushCantMove, "CantMove", clientdata.ByteValue(0))
			}},
			{"CantParry-clear", func() ([]byte, error) {
				return rushPropertyFrame(propCantParry, "CantParry", clientdata.Int32Value(0))
			}},
			{"msg26", func() ([]byte, error) { return skillPauseClearFrame() }},
			{"PauseSkill-clear", func() ([]byte, error) {
				return rushPropertyFrame(propRushPauseSkill, "PauseSkill", clientdata.StringValue(""))
			}},
		}
		for _, step := range release {
			if frame, err := step.build(); err == nil {
				if err := link.WriteFrame(frame); err != nil {
					log.Printf("rush dash timeout %s: %v", step.name, err)
					return
				}
			}
		}
		log.Printf("%s: rush dash landing timeout; released rush state", remote)
	})
	log.Printf("%s: rush dash issued dest=(%.2f,%.2f,%.2f) action=%s",
		remote, destX, destY, destZ, pending.groundRushAction)
	return nil
}

// ---- 212 落地 ----

// handleCastDashLanding 官服 212 方言：
// [212, x, y, z, 朝向, 尾值]——尾 1=冲刺落地 / 0=步行落地（b 51.348 实证），
// 两者都是落地信号（距目标 ≤3m 即落地）。无暂存 212 官服零回应（静默）。
// 212 三分（远距未冲=就地执行请求→发 c61；远距已冲=安全释放）保留为
// 静默兜底——仅客户端偏离官服方言时触发。
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
				// 冲刺已发未执行（客户端拉回/暂存期按了别的技能）——安全作废。
				if player.rushDashIssuedSnapshot() {
					cancelPendingCast(link, player, x, y, z, orient, remote)
					return true, nil
				}
				// 超走带的 212 无条件落地（Java b 线方言：handleDashLanding
				// 无距离闸直接 resumePendingCastAt）。54949 会话实证走带闸
				// 静默忽略致客户端狂发 212×278 无响应——客户端停点即其
				// 认定的射程内（其目标认知与服务端 NPC 实时位可差 5m+，
				// 按 211 自报目标坐标算距才与客户端同口径）。
				log.Printf("%s: dash landing beyond walk band accepted skill id=%s dist=%.1fm",
					remote, pending.skillID, distance)
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
	// 官服对无暂存 212 零回应（b 51.4+ 连发 15 条 212[尾0] 全程静默）；
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
	pending := player.consumePendingCast(time.Now().UTC())
	player.clearRushInFlight()
	if pending != nil {
		log.Printf("%s: dash landing outside walk band; cancelled pending skill id=%s dash=%t", remote, pending.skillID, dashIssued)
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

// resumePendingCastAt 落地续招本体（官服定版序，f 43.233 冲刺落地 /
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
	player.clearRushInFlight()
	if world != nil && world.wireID != 0 {
		world.run.updateMemberPosition(world.wireID, role.Position{X: x, Y: y, Z: z, Orient: orient})
	}
	pending := player.consumePendingCast(time.Now().UTC())
	if pending == nil {
		// 无暂存出招（过期/重复落地）：官服零回应（b 实证），静默对账。
		log.Printf("%s: dash landing without live pending cast (expired or none) via %s", remote, trigger)
		return nil
	}
	now := time.Now().UTC()
	geometry := pending.geometry
	orientToTarget := orientTowardTarget(x, z, geometry.X, geometry.Z)
	distance := math.Sqrt(float64(square(x-geometry.X) + square(z-geometry.Z)))
	model, _ := world.combatNPCModel(pending.targetID)
	// 步行落地（未结算）：CD/MP 在落地簇内结算（b 51.387 官服序）。
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
	// 状态四连（官服 416.162 落地簇 → 416.321 +159ms 四连实证）。
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
	// 官服 msg15 全形态（走 82B / 停 67B / 跳 105B）idx3..5 恒为 x,y,z，
	// 非冲锋在途也收编位置（技能.pcapng 定版：0x21 回显含玩家条目 =
	// 服务端收编证据；追击 NPC 测距依赖服务端侧玩家坐标）。
	x, y, z := values[3].Float32, values[4].Float32, values[5].Float32
	orient := movementUploadOrient(values)
	absorbClientPosition(world, x, y, z, orient)
	if player == nil {
		return true, nil
	}
	if player.rushInFlightSnapshot() {
		if player.rushDashIssuedSnapshot() {
			// c61 冲刺段在途位移：不回显，消费方为 212/c61 落地链
			//（官服方言）。
			tryRushArrival(link, player, world, x, y, z, orient, remote)
			return true, nil
		}
		// 受理后的技能接近段（c61 未发）：官服逐帧 0x21 玩家回显 +
		// 走停迁移四连照发（jn 34.105/34.833 接近段持续 0x21 样本；
		// 64785 实证缺四连客户端中途停滞不自走），位置照常收编进
		// 到达检测。
		echoPlayerMoveIntent(link, player, values, x, y, z, remote)
		tryRushArrival(link, player, world, x, y, z, orient, remote)
		return true, nil
	}
	// 官服移动应答方言（技能.pcapng §一）：逐帧 0x21 玩家回显 +
	// 走↔停迁移状态四连——客户端等这对帧才执行点击移动。
	echoPlayerMoveIntent(link, player, values, x, y, z, remote)
	return true, nil
}

// echoPlayerMoveIntent 官服移动应答（技能.pcapng §一 全样本实证）：每帧
// 寻路族（id 高 16 位 0xA000——点击地面/地图点击/技能接近）与 WASD 族
// （0x0000/0x3000）的 state7/4/6/8 回显一帧单条目 0x21（走帧
// dest=(idx6, y, idx7)、角度=atan2(destX−x, destZ−z)、速度=idx8、朝向=
// idx9；停帧 dest=上报终点、角度=朝向增量 idx6、速度/朝向=0）；state5
// 跳跃双帧不回显。state7↔4 走停迁移沿先补发官服状态四连
// （586/645/646/750——官服对每次点击起跑/到位停帧 +0~80ms 必发，
// 64785 会话实证缺它客户端停滞不自走），同态重发只回 0x21（官服追怪
// 改向五连重报无 ping）。冲锋在途段不回显（消费方为 tryRushArrival，
// 官服该段位移走 c61/212 落地链）。
func echoPlayerMoveIntent(link sceneMessageConnection, player *playerActor, values []clientCustomValue,
	x, y, z float32, remote string) {
	if len(values) < 7 || values[1].Type != 2 || values[2].Type != 2 || values[6].Type != 4 {
		return
	}
	intentID := uint32(values[1].Int32)
	state := values[2].Int32
	echo := playerMoveEcho{objectID: playerObjectID, ownerID: playerOwnerID, intentID: intentID}
	walking := false
	switch state {
	case 7, 6, 8: // 行进态（6/8=下落/滑翔变体，官服同回显）
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
	case 4: // 停帧：官服角度槽=朝向增量（0.07→ang=0.07 三证）
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

// moveEchoAngle 官服移动角：atan2(dx, dz) 归一 [0, 2π)（技能.pcapng
// 官服样本 0.07/2.12/5.21/4.69 四证——负角补 2π，如 -1.076→5.21）。
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

// issueAreaGroundRushDash 范围型点地受理即冲刺（技能.pcapng 1216.284
// 官服簇：0x1F 锚 → c61（+48ms），**无 CantMove/CantParry（锁定 c61
// 专属）、无 msg25/PauseSkill**；CD/MP/伤害仍由主出招路径即时结算——
// 官服伤害在落地 212 后 ~400ms，单机无对拍方先即时，真机验证后再官服
// 化）。~400ms 后补 msg26 + 0x1F + msg337 落地三件套（近似官服 1216.720
// 落地簇，位置取 c61 终点）。
func issueAreaGroundRushDash(link sceneMessageConnection, player *playerActor,
	definition combatSkillDefinition, castAt, ground worldcore.Transform, remote string) {
	distance := math.Sqrt(float64(square(castAt.X-ground.X) + square(castAt.Z-ground.Z)))
	if distance <= areaRushDirectCastMeters {
		return
	}
	level := int32(2)
	action := areaRushTier2Action
	if distance <= rushDashTriggerMeters() {
		level = 1
		action = defaultGroundRushAction
	}
	// c61 终点 = 地面点身前 bind（官服 1216 dash=dist-1.5、1275 同式）。
	destX, _, destZ := rushBindPoint(ground, castAt.X, castAt.Z, defaultRushBindDistance)
	destY := castAt.Y + rushDashLiftMeters
	if err := link.WriteFrame(serverLocation(playerObjectID, playerOwnerID, castAt)); err != nil {
		log.Printf("%s: area rush anchor: %v", remote, err)
		return
	}
	frame, err := dashRushFrame(destX, destY, destZ, level, action)
	if err == nil {
		err = link.WriteFrame(frame)
	}
	if err != nil {
		log.Printf("%s: area rush dash skill id=%s: %v", remote, definition.id, err)
		return
	}
	log.Printf("%s: area rush dash skill id=%s ground=(%.2f,%.2f) dist=%.1fm tier=%d",
		remote, definition.id, ground.X, ground.Z, distance, level)
	dest := worldcore.Transform{X: destX, Y: castAt.Y, Z: destZ, Orient: orientTowardTarget(castAt.X, castAt.Z, ground.X, ground.Z)}
	time.AfterFunc(400*time.Millisecond, func() {
		steps := []struct {
			name  string
			build func() ([]byte, error)
		}{
			{"msg26", func() ([]byte, error) { return skillPauseClearFrame() }},
			{"0x1F", func() ([]byte, error) { return serverLocation(playerObjectID, playerOwnerID, dest), nil }},
			{"msg337", func() ([]byte, error) { return casterTransformSyncFrame(dest.X, dest.Y, dest.Z, dest.Orient) }},
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
}

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
	// msg220 [220, x, y, z, orient?]：位置同样收编（官服仅在接近段发出，
	// 收编无害且保持与 msg15 同口径）。
	absorbClientPosition(world, values[1].Float32, values[2].Float32, values[3].Float32, orient)
	if player == nil || !player.rushInFlightSnapshot() {
		return true, nil
	}
	tryRushArrival(link, player, world, values[1].Float32, values[2].Float32, values[3].Float32, orient, remote)
	return true, nil
}

// tryRushArrival 移动上报驱动的冲刺探测：仅当距目标进入冲刺窗口
// (GRushRange 5.0, +Add 6.5] 时发 c61（g 5.54 触发 / b 3.61 不触发钉死
// 窗口下界）；低于下界由客户端自行走进 ≤3m 发 212[尾0] 步行落地。
// 冲刺已发出后静默——落地只认 212。接近在途时按官服节奏 ~3s 重发
// 状态四连（586/645/646/750——官服全程周期下发，接近段保持喂养）。
func (p *playerActor) maybeRushStatePing(link sceneMessageConnection, now time.Time) {
	p.mu.Lock()
	due := now.Sub(p.rushStatePingAt) >= 3*time.Second
	if due {
		p.rushStatePingAt = now
	}
	p.mu.Unlock()
	if !due {
		return
	}
	for _, frame := range officialStatePingFrames() {
		if frame == nil {
			continue
		}
		if err := link.WriteFrame(frame); err != nil {
			return
		}
	}
}

// tryRushArrival 移动上报驱动的冲刺探测：仅当距目标进入冲刺窗口
// (GRushRange 5.0, +Add 6.5] 时发 c61；低于下界由客户端自行走进走带
// 发 212 落地。冲刺已发出后静默——落地只认 212。
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
	// maybeRushStatePing 停用（对照构建：官服四连是行走上报的回执，
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
	// 授予窗 per-skill（技能1.pcapng 2026-11-13：官服对 m220 进窗应答 c61，
	// 窗外零应答——hyjf02 220@8.1 授予 / @6.1、7.3 走入直收；xytf_sky 窗
	// 达 14m）。未列表回落 (GRushRange, +Add]。
	windowLo, windowHi := pending.dashWindowLo, pending.dashWindowHi
	if windowHi <= 0 {
		windowLo, windowHi = rushDashWindowLoMeters(), rushDashTriggerMeters()
	}
	if distance > windowHi || distance <= windowLo {
		// 官方方言：接近段由客户端自走（jn 32.642 受理簇同毫秒 state7；
		// 65210 会话 23:02:27 官服簇下方客户端实发自走帧实证）。旧的
		// 服务端 c61/0x1F 驱动（「fork 客户端从未自走」2026-11-08 定论
		// 的产物）会把玩家一下一下瞬移（23:02 真机实证），已撤除——
		// 等客户端走进冲刺窗口由 issueRushDash 接手。
		return
	}
	log.Printf("%s: rush within dash window skill id=%s dist=%.1fm in (%.1f, %.1f]",
		remote, pending.skillID, distance, windowLo, windowHi)
	if err := issueRushDash(link, player, geometry, x, y, z, remote); err != nil {
		log.Printf("%s: rush dash issue: %v", remote, err)
	}
}

func square(value float32) float32 { return value * value }

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
