package main

// 冲锋暂存/落地续招链（2026-11-08 官服 b/e.pcapng 帧级重解码终版，
// 地面锁定冲锋全链实证）：
//
// 211 按下（13/14 值带锁定目标，近距 < rushStageNearCastMeters 直接
// 正常施法，超 skill_static.RushRange 静默拒）→ 受理簇（CurSkillID +
// CurSkillTarget + msg25[30000] + PauseSkill=技能名 + msg707[15]）→
// 客户端自动跑向目标（msg15 state7，纯客户端行为，服务端零帧）→
// msg15/220 探测距目标 ≤6.5m（GRushRange+Add）时发最终冲刺簇（c61 档1
// LockGRushAction 至目标身前 bind 点、y+1.0 + CantMove/CantParry 置 1）
// → 客户端冲刺至终点上报 212 → 落地簇（状态清 + 0x1F 锚 + PauseSkill
// 清 + msg26 + msg337 + c1=40 目标受击预兆 + msg19 段伤害起始）→
// 重组 10 值短式 211 重入受理链出伤害。CD/资源在落地释放时才结
//（官服在 c61 时点结——已知时点差异）。
//
// 已纠正的历史误判（详见 docs/冲锋与打坐官服抓包分析及Go移植.md §四）：
// 受理簇首的 msg26/CurIsAirSkill=4 是重受理/空中方言，地面首发簇没有
// （55964 会话实证误发会导致客户端不进入自动接近）；c61 不会瞬移——
// 56989 会话的瞬移实为当时同发的 0x1F 锚（官服 0x1F 只在 212 落地簇）。
//
// 帧值序（官服 7071/7074 线实时实证）：
//   msg25 = [25, 窗口ms, "skill_pause"]（int 前、串后）
//   msg26 = [26]（空值）
//   msg337 = [337, obj(玩家), f32 x, f32 y, f32 z, f32 orient, i32 0]
//   msg24  = [24, obj(玩家), i32 61, f32 destX, f32 destY, f32 destZ, i32 档位, str 动画名]
//   c1=40 目标受击预兆 = [24, obj(目标), 40, i32 1, obj(玩家)]
//   msg19 段伤害起始 = [19, obj(玩家), obj(目标), str 技能, i32 v, i32 v, f32 x, f32 y, f32 z]
//
// 属性索引按名映射自本服 official_property_table.txt（官服现行 7071/7074
// 表在 460 区比本服表多 2 项——表位是相对的，帧索引必须跟本服表走，
// c.pcapng 抽出的 table09_7074.txt 仅作名字考证用）。

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
	clientCustomCastDashLanding int32 = 212 // RESUME_SKILL 冲锋落地
	clientCustomChangeWarning   int32 = 292 // CLIENT_CUSTOMMSG_CHANGEWARNING 预警切换

	// 三层距离模型（Java SkillCastHandler 常量同值 + 2026-11-08 官服
	// b/e.pcapng 帧级重解码修正）。
	selectedRangeFallbackMeters = 8.0              // 无 TSP2/形状半径时的近战回落射程
	rushStageMaxMeters          = 15.0             // RushRange 缺列时的暂存带上限回落
	rushArrivalRadiusMeters     = 3.0              // 到达半径（近身身位，非技能射程）
	rushStageBindSlackMeters    = 2.0              // bind 之上的直接施法余量（防落地重入再暂存）
	rushPendingWindow           = 30 * time.Second // 暂存出招有效期（官服 msg25 窗口=30000ms）
	defaultRushBindDistance     = float32(1.5)     // skill_lock_data 缺行时的身前绑定距离
	defaultGroundRushAction     = "qg_fastrun_2h_01" // 官服地面锁定冲锋动画（b RUSH2+e 地面双实证）
	convertDashLevel            = 1                   // 冲刺档位（官服锁定冲锋 c61=1）
	msgRushCommand              = 61                  // msg24 c1=61 冲刺/疾跑指令
	// rushDashLiftMeters：c61 终点在目标 y 之上的抬升（e.pcapng lyfh02
	// 目标 y=43.89 → c61 dest y=44.89，双样本恒 +1.0；冲刺带小跳弧线）。
	rushDashLiftMeters = float32(1.0)
	// rushDashTimeout：c61 发出后等待 212 落地的安全阀。官服无此机制
	//（依赖客户端必然回报），但 53831 会话有 CantMove 悬挂前科，超时
	// 补发状态清零簇防止卡死。
	rushDashTimeout = 6 * time.Second
	// rushAirborneYDeltaMeters：211 自报 y 高于目标该值即判空中冲锋
	//（e.pcapng 空中样本差 6m/1.5m，地面样本差 0；1.5 取地面样本上界）。
	// 2026-11-08 真机回归后禁用：world 内部 NPC y 与广播/地表 y 语义
	// 不一致（53831 会话 NPC 广播 y 差 0.13m、内部判定恒空中），误判
	// 空中=不发 c61=客户端零反应；空中方言（无 c61 自走）依赖客户端
	// 自走，一律走地面链反而安全。待内部 y 校准后再启用。
	rushAirborneYDeltaMeters = float32(1e9)
)

// rushDashTriggerMeters 是 c61 冲刺的触发距离：客户端自动接近到该距离
// 内时服务端发最终冲刺指令。= rush_range.ini GRushRange(5.0)+GRushRangeAdd
// (1.5)，即客户端轻功地面冲刺距离（b/e 双样本 5.9/6.1m 触发一致；
// 同值见 player_actor.go 冲锋属性块 873/876 列下发）。
func rushDashTriggerMeters() float64 {
	return 5.0 + 1.5
}

// rushStageNearCastMeters 是冲锋技不再走冲锋链、直接正常施法的近距阈值
//（官服方言：bind 身位内直接出招）。取 max(bind+余量, 到达半径+0.5)——
// 下界保证 212/msg15 落地重入（≤rushArrivalRadiusMeters）不会因阈值
// 卡在带内被重新暂存成死循环。
func rushStageNearCastMeters(definition combatSkillDefinition) float64 {
	near := float64(definition.bindDistance) + rushStageBindSlackMeters
	if floor := rushArrivalRadiusMeters + 0.5; near < floor {
		near = floor
	}
	return near
}

// 冲锋簇属性索引（按名映射自本服表；官服现行表同名字为 203/220/623/627/690/692）。
const (
	propRushCantMove      uint16 = 203 // CantMove Byte
	propRushPauseSkill    uint16 = 220 // PauseSkill String
	propRushCurIsAirSkill uint16 = 623 // CurIsAirSkill Byte
	propRushJumpTimes     uint16 = 690 // JumpTimes Byte

	propQingGongPointIndex uint16 = 692 // QingGongPoint Int32（轻功点，c61 扣后单帧）
)

// pendingCastState 是一次越程按下暂存的出招（Java PlayerActor.PendingCast
// 同源；冲锋参数在暂存时从已解析定义快照，落地/转冲刺不再回查目录）。
type pendingCastState struct {
	skillID          string
	targetID         uint32
	targetOwner      uint32
	expiresUTC       time.Time
	canRush          bool
	castRange        float32
	bindDistance     float32
	groundRushAction string
	// airborne：空中冲锋方言（211 自报 y 高于目标 1.5m+）。空中冲刺簇
	// 不发 c61、落地簇补 JumpTimes=0（e.pcapng 空中 5/6 实证）。
	airborne bool
	// geometry：冲锋链几何基准（211 自报目标坐标优先，缺省服务端坐标）。
	// 距离判定/c61 终点/212 三分全部按它算——客户端按自己的世界观执行
	// 冲刺，终点偏离其认知目标会被拒冲（57587 会话实证）。伤害结算仍
	// 走服务端实体，与此解耦。
	geometry worldcore.Transform
	// settled：CD/MP/轻功已在 c61 时点结算（官服 b/e/f 三源实证触发簇
	// 含 CD 双行+MP 对+轻功扣，60536 会话实证缺它们客户端不执行冲刺）。
	// 落地重入据此跳过重复结算。
	settled bool
	// 结算快照（staging 时从 definition 复制，c61 时点 commitRushDashCosts 用）。
	mpCost, spCost, qgCost   int32
	cooldownCategory         int32
	cooldownTeam             int32
	personalCD, publicCD     time.Duration
	taoLu                    string
	// nominalDamage：msg19 段伤害起始帧的基准值（definition.baseDamage
	// 暂存快照；落地释放簇使用）。
	nominalDamage int32
}

// rushStageEnabled 对齐 Java -Dnineyin.rush.stage（默认开）；
// NINEYIN_RUSH_STAGE=off 时 212 仅回显、211 越程按普通超距拒。
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

// peekPendingCast 返回未过期的暂存出招（不清除）。
func (p *playerActor) peekPendingCast(now time.Time) *pendingCastState {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pendingCast == nil || now.After(p.pendingCast.expiresUTC) {
		return nil
	}
	return p.pendingCast
}

// consumePendingCast 幂等取出暂存出招：过期/已消费返回 nil。
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

// markRushDashIssued 标记最终冲刺指令（c61）已发出：此后 msg15/220 到位
// 探测静默，落地只认 212（官服方言——冲刺在途不做距离续招，防中途打断
// 客户端冲刺动画）。
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

// setResumeSettleSkip / resumeSettleSkipSnapshot 是落地重入与受理链之间
// 的一次性跳过标记：pending.settled（c61 时点已结算）的重组 211 重入
// handleSkillCustom 时置位——受理链据此跳过 CD 闸/扣费/CD 行（防止
// 个人 CD 拒掉自己的落地续招、防双扣）。同连接同 goroutine，无竞态。
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

// commitRushDashCosts 在 c61 冲刺时点结算 CD/MP/SP/轻功并做出招簿记
//（官服 b/e/f 三源实证：触发簇含 CD 双行+MP 对+轻功扣；60536 会话实证
// 缺它们客户端不执行冲刺）。失败（资源不足）返回错误——调用方应作废
// 暂存，不学官服「受理即扣」的事前闸。
func (p *playerActor) commitRushDashCosts(pending *pendingCastState, now time.Time) error {
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
	if pending.qgCost > 0 {
		if _, ok := p.qinggong.Use(pending.qgCost, now); !ok {
			return fmt.Errorf("QingGongPoint cannot pay %d", pending.qgCost)
		}
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
	p.pendingCast.settled = true
	return nil
}

// ---- 帧构造（全部 0x1E 外壳数值 custom，Java customIntMessage 同构） ----

// skillPauseFrame 即 msg25：[25, 窗口毫秒, "skill_pause"]。
func skillPauseFrame(windowMs int32) ([]byte, error) {
	return serverCustomIntMessage(25, customInt(windowMs), customString("skill_pause"))
}

// skillPauseClearFrame 即 msg26：空值解除 skill_pause。
func skillPauseClearFrame() ([]byte, error) {
	return serverCustomIntMessage(26)
}

// casterTransformSyncFrame 即 msg337：施法者位置/朝向同步（值取 211/212 请求）。
func casterTransformSyncFrame(x, y, z, orient float32) ([]byte, error) {
	return serverCustomIntMessage(337,
		customObject(playerObjectID, playerOwnerID),
		customFloat32(x), customFloat32(y), customFloat32(z),
		customFloat32(orient),
		customInt(0),
	)
}

// dashRushFrame 即 msg24 c1=61 冲刺/疾跑指令：
// [24, obj, 61, f32 dest, 档位(官服冲刺=1/参考服疾跑=2), 动画名(LockG/A/WRushAction)]。
func dashRushFrame(x, y, z float32, level int32, action string) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(playerObjectID, playerOwnerID),
		customInt(msgRushCommand),
		customFloat32(x), customFloat32(y), customFloat32(z),
		customInt(level),
		customString(action),
	)
}

// curSkillIDFrame 暂存按下时单属性 CurSkillID 帧（Java curSkillIdFrame 同构）。
func curSkillIDFrame(skillID string) ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propCurSkillID, Name: "CurSkillID", Value: clientdata.StringValue(skillID)},
	})
}

// rushPropertyFrame 生成冲锋簇的 n=1 单属性 0x10 帧（官服簇逐帧单属性，
// 绝不整帧重推——坐姿/冲锋态下整帧重推会重入动画）。
func rushPropertyFrame(index uint16, name string, value clientdata.Value) ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: index, Name: name, Value: value},
	})
}

// rushBindPoint 目标朝玩家方向 BindDistance 处的冲锋终点（官服 b RUSH2
// c61 dest 与目标差恒 1.5m 实证），地面/空中冲锋共用。
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

// rushTargetHitPrecastFrame 即 msg24 c1=40 目标受击预兆（5 值 obj 变体，
// e.pcapng 空中冲锋落地簇实证）：
//
//	[24, obj(目标), 40, i32 1, obj(攻击者)]
//
// 与 c1=40 的字符串变体（dazuo_old/hurt_*）同族不同形：锁定冲锋释放
// 簇用 obj 变体预告目标受击。
func rushTargetHitPrecastFrame(targetID, targetOwner uint32) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(targetID, targetOwner),
		customInt(40),
		customInt(1),
		customObject(playerObjectID, playerOwnerID),
	)
}

// segmentDamageOpenFrame 即 msg19 段伤害起始帧（9 值，e.pcapng 空中
// 冲锋落地簇实证：wuji_CS_jz_lyfh05 [1500,1500] / lyfh02 [1066,1066]
// 双同值——攻击起始时刻的基准值，与后续 c1=17 实际伤害独立）：
//
//	[19, obj(攻击者), obj(目标), str 技能, i32 v, i32 v, f32 x, f32 y, f32 z]
func segmentDamageOpenFrame(targetID, targetOwner uint32, skillID string, value int32, x, y, z float32) ([]byte, error) {
	return serverCustomIntMessage(19,
		customObject(playerObjectID, playerOwnerID),
		customObject(targetID, targetOwner),
		customString(skillID),
		customInt(value),
		customInt(value),
		customFloat32(x), customFloat32(y), customFloat32(z),
	)
}

// ---- sceneLifecycle 战斗 NPC 查询 ----

// combatNPCAlive 校验对象是否存活战斗 NPC（owner 匹配）。
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

// combatNPCTransform 返回存活战斗 NPC 的位姿（射程/到达判定共用）。
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

// ---- 211 越程暂存 ----

// rushCastStateFrame 即 msg707[15]：官服受理簇/重受理簇尾帧（b RUSH2、
// e lyfh02/05 地面锁定 + a hyjf07 范围全形态实证恒 [707,15]；方向型无）。
// 语义未定位（同族变体 8/13/15/29），按官服原样镜像——受理簇缺它时
// 55964 真机会话客户端不进入自动接近（state7），照抄后恢复正常。
func rushCastStateFrame() ([]byte, error) {
	return serverCustomIntMessage(707, customInt(15))
}

// stageRushCast 冲锋受理（2026-11-08 官服 b/e.pcapng 帧级重解码 + 62622
// 真机回归终版）：
//
// 受理簇（官服地面锁定序，e lyfh02 +38ms 实证）：
//   CurSkillID → CurSkillTarget → msg25[30000] → PauseSkill=技能名 → msg707[15]
// 客户端收到本簇即自行跑向目标（msg15 state7，速度 4.5~5m/s，同刻起跑）。
// 旧方言的 msg26 首帧与 CurIsAirSkill=4 已删：msg26 是官服重受理时的
// 前导清帧（再次按键时才发），CurIsAirSkill=4 是空中样本方言，地面簇
// 无此帧——2026-11-08 前误发导致客户端不自走（55964 会话零 state7）。
//
// **冲刺指令一律由 msg15/220 移动上报触发**（62622 真机教训）：官服 c61
// 从不在受理簇里同刻发出——即使按下时已 ≤6.5m，客户端也是先起走、
// 上报后才收到 c61。受理即发 c61+CantMove 会把刚起走的客户端冻在原地
// （CantMove 先于客户端接近状态机生效），6s 安全阀反复释放即「顺序乱」。
// CD/MP/轻功在落地释放时才结（官服在 c61 时点结——已知时点差异，
// 避免与落地重入的受理链双重扣费，刻意后移）。
func stageRushCast(link sceneMessageConnection, player *playerActor, definition combatSkillDefinition,
	targetID, targetOwner uint32, target worldcore.Transform, castAt worldcore.Transform,
	distance float64, remote string) (bool, error) {
	now := time.Now().UTC()
	airborne := castAt.Y-target.Y > rushAirborneYDeltaMeters
	rearm := player.peekPendingCast(now) != nil
	player.stagePendingCast(&pendingCastState{
		skillID: definition.id, targetID: targetID, targetOwner: targetOwner,
		expiresUTC: now.Add(rushPendingWindow),
		canRush:    definition.canRush, castRange: definition.castRange,
		bindDistance: definition.bindDistance, groundRushAction: definition.groundRushAction,
		airborne:      airborne,
		geometry:      target,
		nominalDamage: definition.baseDamage,
		mpCost:         definition.mpCost, spCost: definition.spCost, qgCost: definition.qgCost,
		cooldownCategory: definition.cooldownCategory, cooldownTeam: definition.cooldownTeam,
		personalCD: definition.personalCD, publicCD: definition.publicCD,
		taoLu: definition.taoLu,
	})
	type frameBuilder struct {
		name  string
		build func() ([]byte, error)
	}
	builders := []frameBuilder{
		{"CurSkillID", func() ([]byte, error) { return curSkillIDFrame(definition.id) }},
		{"CurSkillTarget", func() ([]byte, error) {
			return rushPropertyFrame(propCurSkillTarget, "CurSkillTarget", clientdata.ObjectValue(targetID, targetOwner))
		}},
		{"msg25", func() ([]byte, error) { return skillPauseFrame(int32(rushPendingWindow / time.Millisecond)) }},
		{"PauseSkill-set", func() ([]byte, error) {
			return rushPropertyFrame(propRushPauseSkill, "PauseSkill", clientdata.StringValue(definition.id))
		}},
		{"msg707", func() ([]byte, error) { return rushCastStateFrame() }},
	}
	// 重受理（暂存期再次按下）：官服 f 十连按实证前导三帧——CurSkillID=""
	// + msg26[] + PauseSkill="" 清旧窗，再重铺受理簇。
	if rearm {
		builders = append([]frameBuilder{
			{"CurSkillID-clear-rearm", func() ([]byte, error) { return curSkillIDFrame("") }},
			{"msg26-rearm", func() ([]byte, error) { return skillPauseClearFrame() }},
			{"PauseSkill-clear-rearm", func() ([]byte, error) {
				return rushPropertyFrame(propRushPauseSkill, "PauseSkill", clientdata.StringValue(""))
			}},
		}, builders...)
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
	for _, frame := range frames {
		if err := link.WriteFrame(frame); err != nil {
			return true, err
		}
	}
	log.Printf("%s: rush staged skill id=%s dist=%.0fm air=%t rearm=%t window=%s",
		remote, definition.id, distance, airborne, rearm, rushPendingWindow)
	return true, nil
}

// issueRushDash 最终冲刺指令（官服 c61 冲刺簇，e lyfh02 +470ms 实证）：
// msg24 c61 [obj, dest(目标身前 bind、y+1.0), 档1, LockGRushAction] +
// CantMove/CantParry 置 1。官服同簇还结 CD 行/MP/轻功（我们后移到落地，
// 已知时点差异）。冲刺终点 y = 玩家上报 y + 1.0（双样本恒值抬升，冲刺带
// 小跳弧线）——**绝不能用目标/world 内部 y**：NPC 内部 y 与玩家地面 y
// 差 ~1.2m（54992 会话实证），终点悬空 2m 客户端寻路拒冲、只剩 CantMove
// 冻住（「阻止走动与冲刺」根因）。此刻不发 0x1F 锚——官服 0x1F 只在
// 212 落地簇出现，提前发会被客户端执行成瞬移（56989 会话教训，当时
// 误归咎于 c61 本身）。发出后挂安全阀：rushDashTimeout 内未收到 212 则
// 补发状态清零，防 CantMove 悬挂。
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
	destY := fromY + rushDashLiftMeters
	// c61 时点结算 CD/MP/SP/轻功（官服 b/e/f 三源实证；60536 会话实证
	// 缺它们客户端不执行冲刺）。结算失败（资源不足）作废暂存，不发冲刺。
	if err := player.commitRushDashCosts(pending, time.Now().UTC()); err != nil {
		log.Printf("%s: rush dash cost settle failed skill id=%s: %v", remote, pending.skillID, err)
		cancelPendingCast(link, player, fromX, fromY, fromZ, 0, remote)
		return nil
	}
	state := player.stateSnapshot()
	mpRatio := int32(0)
	if state.MaxMP > 0 {
		mpRatio = state.MP * 100 / state.MaxMP
	}
	cdStart := time.Now().UnixMilli()
	cooldownRows := [][]recordCell{{
		recordInt(uint32(pending.cooldownCategory)),
		recordInt64(cdStart),
		recordInt64(cdStart + pending.personalCD.Milliseconds()),
		recordInt(uint32(0xFFFF)),
	}}
	if pending.cooldownTeam > 0 && pending.publicCD > 0 {
		cooldownRows = append(cooldownRows, []recordCell{
			recordInt(uint32(50000)),
			recordInt64(cdStart),
			recordInt64(cdStart + pending.publicCD.Milliseconds()),
			recordInt(uint32(pending.cooldownTeam)),
		})
	}
	type dashStep struct {
		name  string
		build func() ([]byte, error)
	}
	steps := []dashStep{
		{"c61-dash", func() ([]byte, error) {
			return dashRushFrame(destX, destY, destZ, convertDashLevel, pending.groundRushAction)
		}},
		{"CantMove-set", func() ([]byte, error) {
			return rushPropertyFrame(propRushCantMove, "CantMove", clientdata.ByteValue(1))
		}},
		{"CantParry-set", func() ([]byte, error) {
			return rushPropertyFrame(propCantParry, "CantParry", clientdata.Int32Value(1))
		}},
	}
	for _, row := range cooldownRows {
		cells := row
		steps = append(steps, dashStep{"cd-row", func() ([]byte, error) {
			return serverRecordAddCells(playerObjectID, playerOwnerID, recordCooldown, cells)
		}})
	}
	steps = append(steps,
		dashStep{"MPRatio", func() ([]byte, error) {
			return rushPropertyFrame(propMPRatio, "MPRatio", clientdata.Int32Value(mpRatio))
		}},
		dashStep{"MP", func() ([]byte, error) {
			return rushPropertyFrame(propMP, "MP", clientdata.Int32Value(state.MP))
		}},
		dashStep{"QingGongPoint", func() ([]byte, error) {
			return rushPropertyFrame(propQingGongPointIndex, "QingGongPoint", clientdata.Int32Value(state.QingGongPoint))
		}},
	)
	for _, step := range steps {
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
		// 实例绑定（55052 会话教训）：只释放本次冲刺自己的暂存。重受理会
		// 换新 pendingCastState 实例——旧定时器若只看标志位，会把正在
		// 冲刺的新一轮误释放（18:10:30 定时器在 :36 杀掉 :32 那轮）。
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

// ---- 212 冲锋落地 ----

// handleCastDashLanding 还原 fork handleCastDashLanding / Java handleDashLanding：
// [212, f32 x, f32 y, f32 z, (f32 朝向)] → 按上报位置三分（见函数体）：
// ≤3m 落地续招（清冲锋态+位置收编+msg337+msg26+重组 211 重入受理链）；
// 远处未冲刺=就地执行请求 → 发 c61 命其冲刺；远处已冲刺未执行 → 安全
// 释放（不出伤害防远距结算）。无暂存只回 337 回显。
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
	// 212 三分（55052 会话实证）：上报位置距目标 ≤3m=落地续招（无论 c61
	// 是否发出——位置即真相，覆盖服务端 c61 与客户端自冲两条路）；远处
	// 且未发 c61=客户端就地执行请求（近距按下不走路，官服 b 样本 5.9m
	// 停止状态 c61 照样触发冲刺）→ **发 c61 命其冲刺，不取消不伤害**；
	// 远处且 c61 已发=客户端没执行冲刺 → 安全释放（不出伤害防远距结算）。
	if pending := player.peekPendingCast(time.Now().UTC()); pending != nil {
		if target, ok := world.combatNPCTransform(pending.targetID); ok {
			geometry := pending.geometry
			if geometry.X == 0 && geometry.Y == 0 && geometry.Z == 0 {
				geometry = target
			}
			distance := math.Sqrt(float64(square(x-geometry.X) + square(y-geometry.Y) + square(z-geometry.Z)))
			if distance > rushArrivalRadiusMeters {
				if !player.rushDashIssuedSnapshot() {
					log.Printf("%s: execute request away from target skill id=%s dist=%.0fm; issuing dash",
						remote, pending.skillID, distance)
					if err := issueRushDash(link, player, geometry, x, y, z, remote); err != nil {
						log.Printf("%s: rush dash issue: %v", remote, err)
					}
					return true, nil
				}
				cancelPendingCast(link, player, x, y, z, orient, remote)
				return true, nil
			}
			if !player.rushDashIssuedSnapshot() {
				log.Printf("%s: self-dash landing skill id=%s dist=%.0fm <= %.0fm (no c61 issued)",
					remote, pending.skillID, distance, rushArrivalRadiusMeters)
			}
		}
		// 目标不可测距（自增益目标/world 未就绪）：维持旧行为直接落地续招。
	}
	resumePendingCastAt(link, player, world, x, y, z, orient, "dash-landing", remote)
	return true, nil
}

// cancelPendingCast 作废暂存出招（不结算不伤害）：取消型 212 与「c61 已
// 发但客户端未执行冲刺」的兜底释放共用。清 skill_pause 态（msg26 +
// PauseSkill 空串）+ msg337 位置回显。
func cancelPendingCast(link sceneMessageConnection, player *playerActor,
	x, y, z, orient float32, remote string) {
	pending := player.consumePendingCast(time.Now().UTC())
	player.clearRushInFlight()
	if pending != nil {
		log.Printf("%s: dash landing without issued dash; cancelled pending skill id=%s", remote, pending.skillID)
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

// resumePendingCastAt 冲锋落地续招本体（212 与 msg15/msg220 到位探测共用，
// 官服落地簇形态：CantMove/CantParry 清零 + 0x1F 锚 + PauseSkill 清空 +
// msg26 解除 + msg337 到位 + c1=40 目标受击预兆 + msg19 段伤害起始）：
// 取暂存出招（30s 过期作废）→ 重组 211（10 值短形）重入受理链出伤害。
// 已知简化（对照官服）：重组 211 不带目标对象，重入后沿用当前选中目标。
func resumePendingCastAt(link sceneMessageConnection, player *playerActor, world *sceneLifecycle,
	x, y, z, orient float32, trigger, remote string) error {
	player.clearRushInFlight()
	// fork actor.updatePosition：只收编成员位置，不做视口重算
	//（短距落点；下一次移动上报自然对账）。
	if world != nil && world.wireID != 0 {
		world.run.updateMemberPosition(world.wireID, role.Position{X: x, Y: y, Z: z, Orient: orient})
	}
	pending := player.consumePendingCast(time.Now().UTC())
	if pending == nil {
		// 无暂存出招（过期/重复落地）：只回 msg337 位置收编回显，不刷
		// 清零簇（2026-11-08 53831 会话回归：msg15/220 每次都触发落地
		// 链，485 次清零簇对客户端是纯刷帧噪音；受理已不置位，无需清）。
		sync, err := casterTransformSyncFrame(x, y, z, orient)
		if err != nil {
			return fmt.Errorf("encode caster transform sync: %w", err)
		}
		if err := link.WriteFrame(sync); err != nil {
			return err
		}
		log.Printf("%s: dash landing without live pending cast (expired or none) via %s", remote, trigger)
		return nil
	}
	// 落地簇前导（官服序）：状态位清零 → 0x1F 锚 → PauseSkill 清空。
	prelude := []struct {
		name  string
		build func() ([]byte, error)
	}{
		{"CantMove-clear", func() ([]byte, error) {
			return rushPropertyFrame(propRushCantMove, "CantMove", clientdata.ByteValue(0))
		}},
		{"CantParry-clear", func() ([]byte, error) {
			return rushPropertyFrame(propCantParry, "CantParry", clientdata.Int32Value(0))
		}},
		{"location-anchor", func() ([]byte, error) {
			return serverLocation(playerObjectID, playerOwnerID, worldcore.Transform{X: x, Y: y, Z: z}), nil
		}},
		{"PauseSkill-clear", func() ([]byte, error) {
			return rushPropertyFrame(propRushPauseSkill, "PauseSkill", clientdata.StringValue(""))
		}},
		{"msg26", func() ([]byte, error) { return skillPauseClearFrame() }},
		{"msg337", func() ([]byte, error) { return casterTransformSyncFrame(x, y, z, orient) }},
		{"c1=40-precast", func() ([]byte, error) { return rushTargetHitPrecastFrame(pending.targetID, pending.targetOwner) }},
		{"msg19-open", func() ([]byte, error) {
			return segmentDamageOpenFrame(pending.targetID, pending.targetOwner, pending.skillID, pending.nominalDamage, x, y, z)
		}},
	}
	for _, step := range prelude {
		frame, err := step.build()
		if err != nil {
			return fmt.Errorf("encode landing %s: %w", step.name, err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return err
		}
	}
	// 重组 14 值长式 211：带瞄准对象与几何目标坐标（客户端世界观）。
	// 短式（无目标）重入时，距离闸按服务端坐标测距，落地若离服务端
	// NPC >近距阈值会被再次暂存成死循环；带自报目标后距离闸与冲锋链
	// 同基准，落地即正常出招。
	resume := clientCustomMessage{Opcode: 0x1E, Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomUseSkill},
		{Type: 6, Text: pending.skillID},
		{Type: 4, Float32: x},
		{Type: 4, Float32: y},
		{Type: 4, Float32: z},
		{Type: 4, Float32: orient},
		{Type: 4, Float32: 0},
		{Type: 4, Float32: 0},
		{Type: 4, Float32: 0},
		{Type: 6, Text: ""},
		{Type: 8, Raw: packPlayerObjectRaw(pending.targetID, pending.targetOwner)},
		{Type: 4, Float32: pending.geometry.X},
		{Type: 4, Float32: pending.geometry.Y},
		{Type: 4, Float32: pending.geometry.Z},
	}}
	log.Printf("%s: dash landing resume skill id=%s via %s settled=%t", remote, pending.skillID, trigger, pending.settled)
	// settled 重入跳过 CD 闸/扣费/CD 行（c61 时点已结算）；完事必清，
	// 防标记泄漏到后续普通出招。
	player.setResumeSettleSkip(pending.settled)
	defer player.setResumeSettleSkip(false)
	_, err := handleSkillCustom(link, player, world, resume, remote)
	return err
}

// ---- msg15 / msg220 移动上报 ----

// handleMovementUpload 还原 Java handleMovementUpload：9 值形态
// [15, i32 0xA0005000, i32 state, f32 x, f32 y, f32 z, f32 orient, str, i32 tick]。
// 非冲锋在途时静默返回（高频路径不记日志；常规位置同步由 0x0A motion blob
// 承担，此处只驱动冲锋到位探测）。
func handleMovementUpload(link sceneMessageConnection, player *playerActor, world *sceneLifecycle,
	custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomMovementUpload {
		return false, nil
	}
	if player == nil || !player.rushInFlightSnapshot() {
		return true, nil
	}
	values := custom.Values
	if len(values) < 6 || values[3].Type != 4 || values[4].Type != 4 || values[5].Type != 4 {
		return true, nil
	}
	orient := float32(0)
	if len(values) > 6 && values[6].Type == 4 {
		orient = values[6].Float32
	}
	tryRushArrival(link, player, world, values[3].Float32, values[4].Float32, values[5].Float32, orient, remote)
	return true, nil
}

// handleRushMovementUpload 处理冲锋在途的 msg220 位移上报（官服 b/e
// 抓包实证：冲锋中客户端用 220 而非 15 上报位置）：4×f32 形态
// [220, f32 x, f32 y, f32 z, f32 orient]。非冲锋在途静默返回。
func handleRushMovementUpload(link sceneMessageConnection, player *playerActor, world *sceneLifecycle,
	custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomRushMoveUpload {
		return false, nil
	}
	if player == nil || !player.rushInFlightSnapshot() {
		return true, nil
	}
	values := custom.Values
	if len(values) < 4 || values[1].Type != 4 || values[2].Type != 4 || values[3].Type != 4 {
		return true, nil
	}
	orient := float32(0)
	if len(values) > 4 && values[4].Type == 4 {
		orient = values[4].Float32
	}
	tryRushArrival(link, player, world, values[1].Float32, values[2].Float32, values[3].Float32, orient, remote)
	return true, nil
}

// tryRushArrival msg15/msg220 驱动的接近探测（官服接近段模型）：客户端
// 受理后自动跑向目标（state7 上报流），距目标进入冲刺距离（≤6.5m）时
// 发最终冲刺 c61（每暂存一次）；冲刺已发出后本探测静默——落地只认 212
//（官服冲刺在途无距离续招，中途 resume 会打断客户端冲刺动画）。
// 暂存过期时安全释放冲锋态（防 CantMove 类状态悬挂）。
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
	target, ok := world.combatNPCTransform(pending.targetID)
	if !ok {
		return
	}
	geometry := pending.geometry
	if geometry.X == 0 && geometry.Y == 0 && geometry.Z == 0 {
		geometry = target
	}
	distance := math.Sqrt(float64(square(x-geometry.X) + square(y-geometry.Y) + square(z-geometry.Z)))
	if player.rushDashIssuedSnapshot() {
		return
	}
	if distance > rushDashTriggerMeters() {
		return
	}
	log.Printf("%s: rush within dash range skill id=%s dist=%.0fm <= %.0fm",
		remote, pending.skillID, distance, rushDashTriggerMeters())
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
//（客户端世界观里的目标位置）。冲锋链全部几何（距离判定/c61 终点/212
// 三分）以它为准——服务端 NPC 会走动但移动不经 337 广播，客户端认知
// 可滞后 5m+；按服务端坐标发 c61 时，终点在客户端世界里偏离其认知目标
// 数米，客户端校验不过、拒绝执行冲刺（57587 会话实证）。伤害结算仍以
// 服务端实体为准，与客户端几何解耦。
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
