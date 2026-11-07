package main

// 冲锋暂存/落地续招链（fork qinggong_gj.go + Java SkillCastHandler 九轮终版）：
//
// 211 越程按下 → 暂存疾跑（CurSkillID 帧 + msg25 skill_pause + msg24 c61 档2
// 疾跑去目标位）→ msg15 上报进入射程阈值 → 转冲刺（msg24 c61 档1 +
// LockGRushAction + 目标身前终点 + 0x1F 锚）→ 到目标面前（≤3m）由 msg15/212
// 落地释放（msg337 回显 + msg26 解除 + 重组 10 值短式 211 重入受理链）。
// CD/资源在落地释放时才结。>15m 静默拒（零帧）。
//
// 帧值序以 Java FightMessages 注释（对齐参考服 61653 抓包）为准：
//   msg25 = [25, 窗口ms, "skill_pause"]（int 前、串后）
//   msg26 = [26]（空值）
//   msg337 = [337, obj(玩家), f32 x, f32 y, f32 z, f32 orient, i32 0]
//   msg24  = [24, obj(玩家), i32 61, f32 destX, f32 destY, f32 destZ, i32 档位, str 动画名]

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
	clientCustomCastDashLanding int32 = 212 // RESUME_SKILL 冲锋落地
	clientCustomChangeWarning   int32 = 292 // CLIENT_CUSTOMMSG_CHANGEWARNING 预警切换

	// 三层距离模型（Java SkillCastHandler 常量同值）。
	selectedRangeFallbackMeters = 8.0                     // 无 TSP2/形状半径时的近战回落射程
	rushStageMaxMeters          = 15.0                    // 暂存带上限；越程超出即静默拒
	rushArrivalRadiusMeters     = 3.0                     // 到达半径（近身身位，非技能射程）
	rushPendingWindow           = 30 * time.Second        // 暂存出招有效期
	defaultRushBindDistance     = float32(1.5)            // skill_lock_data 缺行时的身前绑定距离
	defaultGroundRushAction     = "qg_fastrun_0h_01"      // 同上缺省冲刺动画
	stagingDashAction           = "qg_fastruning_s_2h_01" // 暂存疾跑方言（参考服 61653）
	stagingDashLevel            = 2                       // 疾跑档位（官服冲刺=1）
	convertDashLevel            = 1
	msgRushCommand              = 61 // msg24 c1=61 冲刺/疾跑指令
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

// stageRushCast 暂存疾跑：CurSkillID 帧 + msg25 暂存 + msg24 c61 档2 疾跑
// （目的地=目标位置）+ 暂存出招 30s + 标记冲锋在途。CD/资源落地才结。
func stageRushCast(link sceneMessageConnection, player *playerActor, definition combatSkillDefinition,
	targetID, targetOwner uint32, target worldcore.Transform, distance float64, remote string) (bool, error) {
	now := time.Now().UTC()
	player.stagePendingCast(&pendingCastState{
		skillID: definition.id, targetID: targetID, targetOwner: targetOwner,
		expiresUTC: now.Add(rushPendingWindow),
		canRush:    definition.canRush, castRange: definition.castRange,
		bindDistance: definition.bindDistance, groundRushAction: definition.groundRushAction,
	})
	frames := make([][]byte, 0, 3)
	for index, build := range []func() ([]byte, error){
		func() ([]byte, error) { return curSkillIDFrame(definition.id) },
		func() ([]byte, error) { return skillPauseFrame(int32(rushPendingWindow / time.Millisecond)) },
		func() ([]byte, error) {
			return dashRushFrame(target.X, target.Y, target.Z, stagingDashLevel, stagingDashAction)
		},
	} {
		frame, err := build()
		if err != nil {
			return true, fmt.Errorf("encode rush staging frame %d for %s: %w", index, definition.id, err)
		}
		frames = append(frames, frame)
	}
	player.markRushInFlight()
	for _, frame := range frames {
		if err := link.WriteFrame(frame); err != nil {
			return true, err
		}
	}
	log.Printf("%s: rush staged skill id=%s dist=%.0fm window=%s dest=(%.2f,%.2f,%.2f)",
		remote, definition.id, distance, rushPendingWindow, target.X, target.Y, target.Z)
	return true, nil
}

// ---- 212 冲锋落地 ----

// handleCastDashLanding 还原 fork handleCastDashLanding / Java handleDashLanding：
// [212, f32 x, f32 y, f32 z, (f32 朝向)] → 清冲锋态 + 位置收编 + msg337 回显 +
// msg26 解除 + 暂存出招重组 10 值短式 211 重入受理链。
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
	resumePendingCastAt(link, player, world,
		custom.Values[1].Float32, custom.Values[2].Float32, custom.Values[3].Float32, orient,
		"dash-landing", remote)
	return true, nil
}

// resumePendingCastAt 冲锋落地续招本体（212 与 msg15 到位探测共用）：
// 清冲锋态 + 位置收编 + msg337 回显坐标 + msg26 解除 skill_pause + 取暂存出招
// （30s 过期作废）→ 重组 211（10 值短形，目标回落选中态）重入受理链正式出招。
// 已知简化（对照 Java）：重组 211 不带目标对象，重入后沿用当前选中目标。
func resumePendingCastAt(link sceneMessageConnection, player *playerActor, world *sceneLifecycle,
	x, y, z, orient float32, trigger, remote string) error {
	player.clearRushInFlight()
	// fork actor.updatePosition：只收编成员位置，不做视口重算
	//（短距落点；下一次移动上报自然对账）。
	if world != nil && world.wireID != 0 {
		world.run.updateMemberPosition(world.wireID, role.Position{X: x, Y: y, Z: z, Orient: orient})
	}
	sync, err := casterTransformSyncFrame(x, y, z, orient)
	if err != nil {
		return fmt.Errorf("encode caster transform sync: %w", err)
	}
	if err := link.WriteFrame(sync); err != nil {
		return err
	}
	pending := player.consumePendingCast(time.Now().UTC())
	if pending == nil {
		log.Printf("%s: dash landing without live pending cast (expired or none) via %s", remote, trigger)
		return nil
	}
	clear, err := skillPauseClearFrame()
	if err != nil {
		return fmt.Errorf("encode skill pause clear: %w", err)
	}
	if err := link.WriteFrame(clear); err != nil {
		return err
	}
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
	}}
	log.Printf("%s: dash landing resume skill id=%s via %s", remote, pending.skillID, trigger)
	_, err = handleSkillCustom(link, player, world, resume, remote)
	return err
}

// ---- msg15 移动上报 ----

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

// tryRushArrival msg15 驱动的到位探测（Java tryRushArrival 同源）：进入射程
// 阈值先转冲刺（每暂存一次），到目标面前（≤3m 到达半径）才落地续招。
func tryRushArrival(link sceneMessageConnection, player *playerActor, world *sceneLifecycle,
	x, y, z, orient float32, remote string) {
	pending := player.peekPendingCast(time.Now().UTC())
	if pending == nil {
		return
	}
	target, ok := world.combatNPCTransform(pending.targetID)
	if !ok {
		return
	}
	distance := math.Sqrt(float64(square(x-target.X) + square(y-target.Y) + square(z-target.Z)))
	player.mu.Lock()
	dashIssued := player.rushDashIssued
	player.mu.Unlock()
	if !dashIssued && distance > rushArrivalRadiusMeters {
		if pending.canRush && distance <= float64(pending.castRange) {
			issueStageRushDash(link, pending, target, x, y, z)
			player.mu.Lock()
			player.rushDashIssued = true
			player.mu.Unlock()
		}
		return
	}
	if distance > rushArrivalRadiusMeters {
		return
	}
	log.Printf("%s: rush arrival at target skill id=%s dist=%.0fm <= %.0fm", remote, pending.skillID, distance, rushArrivalRadiusMeters)
	if err := resumePendingCastAt(link, player, world, x, y, z, orient, "msg15-arrival", remote); err != nil {
		log.Printf("%s: rush arrival resume: %v", remote, err)
	}
}

// issueStageRushDash 暂存疾跑 → 冲刺转换：msg24 c1=61 档1 + LockGRushAction
// 动画名，终点=目标朝玩家方向 BindDistance 处（暂存带语义恒为接近），
// 随后 0x1F 终点锚——客户端疾跑动画切换为冲刺，把最后一段冲完。
func issueStageRushDash(link sceneMessageConnection, pending *pendingCastState,
	target worldcore.Transform, x, y, z float32) {
	bind := pending.bindDistance
	dx := x - target.X
	dz := z - target.Z
	horiz := math.Sqrt(float64(dx*dx + dz*dz))
	if horiz < 0.01 {
		dx = 1
		horiz = 1
	}
	endX := target.X + float32(float64(dx)/horiz)*bind
	endZ := target.Z + float32(float64(dz)/horiz)*bind
	endY := target.Y
	dash, err := dashRushFrame(endX, endY, endZ, convertDashLevel, pending.groundRushAction)
	if err == nil {
		err = link.WriteFrame(dash)
	}
	if err == nil {
		anchor := serverLocation(playerObjectID, playerOwnerID,
			worldcore.Transform{X: endX, Y: endY, Z: endZ})
		err = link.WriteFrame(anchor)
	}
	if err != nil {
		log.Printf("stage rush dash convert skill id=%s: %v", pending.skillID, err)
		return
	}
	log.Printf("stage rush dash convert skill id=%s dest=(%.2f,%.2f,%.2f) anim=%s",
		pending.skillID, endX, endY, endZ, pending.groundRushAction)
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
