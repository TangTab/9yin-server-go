package main

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// 通道技能（怒招引导型）——挑战.pcapng 2026-11-12 寒心恨雪
// （CS_jh_hyjf08）官服定版，docs/寒心恨雪与太极剑舞引导机制.md §二。
//
// 官服模型与常规多段（scheduleSkillMultiHits，动画 hitback/hitbreak 帧
// 驱动）分轨：通道型 zhaoshi 单段 17 帧仅 1 个 hitbreak，跳数/节奏是纯
// 服务端引导逻辑——固定波数上限、逐波单目标自动重新索敌（每跳恰 1 个
// 受害者、死靶即换、双活目标也来回切）、每波扣 10 怒（SP 掉帧与伤害跳
// 逐拍对齐、无预扣）、SP 不足吞波、无活目标即停、受理 SP=0 整体静默吞
//（四连发零回帧）。角色引导中瞬移跟随当跳目标（客户端逐段回 m914 上报
// 位置，服务端回 0x21 成对停帧）。
const clientCustomChannelSync int32 = 914

// channelSkillSpec 通道参数（挑战.pcapng 实测：首波 +166~227ms、波间隔
// 390~443ms、8 波上限、每波 SP−10 后随命中 +2 回怒）。victimBuff/selfBuff
// =每波瞬挂瞬摘的 m169 对（受害者/自身），selfBuffStaticData 进 BufferInfo
// 槽 CSV 首段（buff_new StaticData，9019=0x233B 实证）。
type channelSkillSpec struct {
	waves             int
	firstWave         time.Duration
	interval          time.Duration
	spPerWave         int32
	victimBuff        string
	selfBuff          string
	selfBuffStatic    int32
}

// channelSkillSpecs 官服抓包种子。UseLimit=166 全目录恰 3 技
// （hyjf08/SM_hyjf08/tqjf08）：hyjf08 为抓包实证；tqjf08 zhaoshi 与
// hyjf08 逐位同构（单段 17 帧 wj_qzj_07_1 + hitbreak@15 + MovieScene）
// 按同构播种；SM 变体无 zhaoshi 段（missing_player_action）不可施放。
var channelSkillSpecs = map[string]channelSkillSpec{
	"CS_jh_hyjf08": {waves: 8, firstWave: 180 * time.Millisecond, interval: 440 * time.Millisecond, spPerWave: 10,
		victimBuff: "buf_CS_jh_hyjf08", selfBuff: "buf_CS_jh_hyjf08_03", selfBuffStatic: 9019},
	"CS_jh_tqjf08": {waves: 8, firstWave: 180 * time.Millisecond, interval: 440 * time.Millisecond, spPerWave: 10,
		victimBuff: "buf_CS_jh_tqjf08", selfBuff: "buf_CS_jh_tqjf08_03", selfBuffStatic: 32607},
}

// 通道帧簇官服方言常量（挑战.pcapng 波簇全帧实证）：
// m169 九值 i 极性=1 挂/0 摘（本形态实证）；v7=1601 纪元毫秒；
// BufferInfo CSV 六段=StaticData HEX、caster 双恒段、1、到期 1601 HEX、
// 剩余跳 0（seg2/seg3 恒定模板，跨玩家语义未考）；复位=末波+1s 五连归零。
const (
	windowsEpochDeltaMs int64 = 11644473600000
	channelCSVConstSegs       = "11078F4C,3DC2F97"
	channelHurtAction         = "hurt_f_1_l_half"
	channelResetDelay         = time.Second
)

var (
	propCantBeAttackIdx = clientdata.MustPlayerProperty("CantBeAttack").Index
	propBufferInfo5Idx  = clientdata.MustPlayerProperty("BufferInfo5").Index
)

// epoch1601Ms 当前时刻的 1601 纪元毫秒（官服 m169 v7/CSV 到期段口径）。
func epoch1601Ms(now time.Time) int64 {
	return now.UnixMilli() + windowsEpochDeltaMs
}

// channelBuffFrame m169 通道 buff 九值形态（官服波簇逐位）：
// [169, i, 窄串buff, obj(承载), obj(施法), 宽串承载名, 宽串施法名,
//  i64 1601纪元ms, i32 0]；i=1 挂 / 0 摘。
func channelBuffFrame(state int32, buffName string, bearerID, bearerOwner, casterID, casterOwner uint32, bearerWide, casterWide string, epochMs int64) ([]byte, error) {
	return serverCustomIntMessage(169,
		customInt(state),
		customString(buffName),
		customObject(bearerID, bearerOwner),
		customObject(casterID, casterOwner),
		customWideString(bearerWide),
		customWideString(casterWide),
		customInt64(epochMs),
		customInt(0),
	)
}

// channelBufferInfoFrame BufferInfo5 单属性帧：清空 "" 或六段 CSV
//（首段=selfBuff StaticData HEX，第五段=到期=当前 1601ms HEX）。
func channelBufferInfoFrame(csv string) ([]byte, error) {
	return objectSingleProperty(playerObjectID, playerOwnerID, propBufferInfo5Idx, "BufferInfo5",
		clientdata.StringValue(csv))
}

// writeChannelCantFrames Cant* 锁族单属帧五连。set=true 置位
//（1/2/1/1/1，官方通道期态）；set=false 波间翻转清零（0/1/0/0/0）；
// reset=true 通道收束全零（末波+1s 实证）。
func writeChannelCantFrames(link sceneMessageConnection, mode int, remote string) {
	values := map[int][]struct {
		index uint16
		name  string
		value int32
	}{
		0: { // 波间翻转清零（CantParry 翻 1）
			{propCantBeSkillLocked, "CantBeSkillLocked", 0},
			{propCantParry, "CantParry", 1},
			{propCantBeSkillMustLocked, "CantBeSkillMustLocked", 0},
			{propCantBeAttackIdx, "CantBeAttack", 0},
			{propRushCantMove, "CantMove", 0},
		},
		1: { // 通道置位（CantParry=2）
			{propCantBeSkillLocked, "CantBeSkillLocked", 1},
			{propCantParry, "CantParry", 2},
			{propCantBeSkillMustLocked, "CantBeSkillMustLocked", 1},
			{propCantBeAttackIdx, "CantBeAttack", 1},
			{propRushCantMove, "CantMove", 1},
		},
		2: { // 通道收束全零
			{propCantBeSkillLocked, "CantBeSkillLocked", 0},
			{propCantParry, "CantParry", 0},
			{propCantBeSkillMustLocked, "CantBeSkillMustLocked", 0},
			{propCantBeAttackIdx, "CantBeAttack", 0},
			{propRushCantMove, "CantMove", 0},
		},
	}[mode]
	for _, prop := range values {
		frame, err := objectSingleProperty(playerObjectID, playerOwnerID, prop.index, prop.name,
			clientdata.Int32Value(prop.value))
		if err != nil {
			continue
		}
		if err := link.WriteFrame(frame); err != nil {
			log.Printf("%s: channel cant frame %s: %v", remote, prop.name, err)
			return
		}
	}
}

// writeChannelAcceptStateFrames 受理簇通道态帧（官服序：CurSkillID →
// CurSkillTarget → Cant* 置位 → BufferInfo 槽 CSV；m337/m20 之前）。
// CurSkillID 设置+波1 清空的瞬时对是客户端通道演出触发链的组成部分。
func writeChannelAcceptStateFrames(link sceneMessageConnection, definition combatSkillDefinition, targetID, targetOwnerID uint32, now time.Time, remote string) {
	spec := definition.channel
	acceptFrames := []func() ([]byte, error){
		func() ([]byte, error) {
			return objectSingleProperty(playerObjectID, playerOwnerID, propCurSkillID, "CurSkillID",
				clientdata.StringValue(definition.id))
		},
		func() ([]byte, error) {
			return objectSingleProperty(playerObjectID, playerOwnerID, propCurSkillTarget, "CurSkillTarget",
				clientdata.ObjectValue(targetID, targetOwnerID))
		},
		func() ([]byte, error) {
			csv := fmt.Sprintf("%X,%s,1,%X,0", spec.selfBuffStatic, channelCSVConstSegs, epoch1601Ms(now))
			return channelBufferInfoFrame(csv)
		},
	}
	for _, build := range acceptFrames {
		frame, err := build()
		if err != nil {
			continue
		}
		if err := link.WriteFrame(frame); err != nil {
			log.Printf("%s: channel accept state frame: %v", remote, err)
			return
		}
	}
	writeChannelCantFrames(link, 1, remote)
}

// channelRetargetRadius 逐波索敌半径：官服换靶样本目标间 ~5-7m、角色
// 瞬移幅度 ≤7.7m；castRange（近战 3-5m）容不下换靶，按样本上限放宽。
func (definition combatSkillDefinition) channelRetargetRadius() float32 {
	radius := definition.castRange + 6
	if radius < 10 {
		radius = 10
	}
	return radius
}

// setChannelWindow 打开通道窗口（受理时调用）：waves×interval+2s 余量。
func (p *playerActor) setChannelWindow(now time.Time, spec *channelSkillSpec, castAt worldcore.Transform, hasPos bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.channelUntil = now.Add(spec.firstWave + time.Duration(spec.waves)*spec.interval + 2*time.Second)
	p.channelTransform = castAt
	p.channelHasPos = hasPos
}

// noteChannelSync m914 逐段位置上报：窗口内更新权威位置并返回 true。
func (p *playerActor) noteChannelSync(now time.Time, transform worldcore.Transform) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if now.After(p.channelUntil) {
		return false
	}
	p.channelTransform = transform
	p.channelHasPos = true
	return true
}

// channelOriginSnapshot 逐波索敌基准：m914 最新上报优先，回落 211 自报。
func (p *playerActor) channelOriginSnapshot(fallback worldcore.Transform) worldcore.Transform {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.channelHasPos {
		return p.channelTransform
	}
	return fallback
}

// channelWaveTarget 逐波单目标索敌（用户 2026-11-12 口径 + 抓包互证）：
// 每波从附近存活战斗 NPC 中**随机**选一个、一次伤害换一个（424.7 六连
// 跳六杀换了六个目标、474.0 双目标来回切均无最近优先规律）；首波=211
// 受理帧带的锁定目标（6/6 施放首跳受害者==受理目标）。无活目标即停
//（官服 1 跳击杀后无目标同帧收束）。
func (s *sceneLifecycle) channelWaveTarget(origin worldcore.Transform, radius float32, prefer skillDamageTargetRef) (skillDamageTargetRef, bool) {
	if s == nil {
		return skillDamageTargetRef{}, false
	}
	refs := s.queryShapedSkillTargets(origin, radius, 0, 0, 0)
	if len(refs) == 0 {
		return skillDamageTargetRef{}, false
	}
	if prefer.id != 0 {
		for _, ref := range refs {
			if ref.id == prefer.id {
				return ref, true // 首波/偏好目标存活即用
			}
		}
	}
	// 随机选靶：combatRandomFloat64 与伤害掷值同源（测试可 stub）。
	return refs[int(combatRandomFloat64()*float64(len(refs)))%len(refs)], true
}

// launchChannelWaves 通道波次引擎（受理簇之后调用，替代常规多段调度）。
// 每波原子序（挑战.pcapng 波 2 全帧实证）：SP−10 帧 → 复用
// writeSkillSettleFrames（FP/SP+2 → 0x13 → c1=17 → c1=10 段序 → 击杀
// 簇 c1=29）→ c1=71 每波效果帧。官服另有 m169 buf 对与 Cant* 锁族帧
//（buf_CS_jh_hyjf08(_03) 瞬挂瞬摘、CantMove/CantBeAttack 翻转），暂不
// 镜像——登记在 docs §六 缺口清单。
func launchChannelWaves(link sceneMessageConnection, world *sceneLifecycle, definition combatSkillDefinition,
	player *playerActor, custom clientCustomMessage, targetID, targetOwnerID uint32, remote string) {
	spec := definition.channel
	if spec == nil || world == nil {
		return
	}
	now := time.Now()
	castAt, hasCast := useSkillCastTransform(custom)
	player.setChannelWindow(now, spec, castAt, hasCast)
	radius := definition.channelRetargetRadius()
	playerName := player.snapshotName()
	// 通道收束：末波+1s Cant* 五连归零（官服 390.509 实证；提前中断的
	// 通道也按全长时刻收束，Cant 态略悬但必然清除）。
	time.AfterFunc(spec.firstWave+time.Duration(spec.waves)*spec.interval+channelResetDelay, func() {
		writeChannelCantFrames(link, 2, remote)
	})
	for wave := 0; wave < spec.waves; wave++ {
		waveIndex := wave
		time.AfterFunc(spec.firstWave+time.Duration(waveIndex)*spec.interval, func() {
			// SP 闸：不足本波扣费即整通道静默收束（官服吞波零回帧）。
			if player.skillSP() < spec.spPerWave {
				return
			}
			origin := player.channelOriginSnapshot(castAt)
			// 首波=211 锁定目标；后续每波随机换靶（官服形态）。
			var prefer skillDamageTargetRef
			if waveIndex == 0 {
				prefer = skillDamageTargetRef{targetID, targetOwnerID}
			}
			ref, ok := world.channelWaveTarget(origin, radius, prefer)
			if !ok {
				return // 无活目标：官服同帧收束（c1=29 击杀后不再补帧）
			}
			roll := rollSkillHit(player, definition, ref, world, 1)
			if roll.damage <= 0 {
				return // 本波未命中：不发帧不扣费（官服 miss 波形态未观测，按零帧处理）
			}
			hitAt := time.Now()
			outcome, err := world.applySkillDamageToNPC(ref.id, ref.ownerID, roll.damage, player, hitAt)
			if err != nil {
				log.Printf("%s: channel wave %d/%d apply id=%d: %v", remote, waveIndex+1, spec.waves, ref.id, err)
				return
			}
			if outcome.frame == nil {
				return // 目标已失效
			}
			player.noteCombatActivity(hitAt)
			waveWrite := func(frame []byte, ferr error, what string) {
				if ferr != nil {
					return
				}
				if err := link.WriteFrame(frame); err != nil {
					log.Printf("%s: channel wave %s: %v", remote, what, err)
				}
			}
			// 波前簇（官服波2 全帧序逐位镜像）：波1 先清 CurSkillID——
			// 受理设置→波1 清空的瞬时对是客户端通道演出触发链的组成。
			if waveIndex == 0 {
				clearSkill, clearErr := objectSingleProperty(playerObjectID, playerOwnerID, propCurSkillID, "CurSkillID",
					clientdata.StringValue(""))
				waveWrite(clearSkill, clearErr, "cur skill clear")
			}
			epoch := epoch1601Ms(hitAt)
			victimApply, victimErr := channelBuffFrame(1, spec.victimBuff, ref.id, ref.ownerID, playerObjectID, playerOwnerID,
				outcome.wideName, playerName, epoch)
			waveWrite(victimApply, victimErr, "victim buff apply")
			selfApply, selfErr := channelBuffFrame(1, spec.selfBuff, playerObjectID, playerOwnerID, playerObjectID, playerOwnerID,
				playerName, playerName, epoch)
			waveWrite(selfApply, selfErr, "self buff apply")
			clearFrame, clearErr := channelBufferInfoFrame("")
			waveWrite(clearFrame, clearErr, "buffer clear")
			writeChannelCantFrames(link, 0, remote)
			writeChannelCantFrames(link, 1, remote)
			slotFrame, slotErr := channelBufferInfoFrame(fmt.Sprintf("%X,%s,1,%X,0",
				spec.selfBuffStatic, channelCSVConstSegs, epoch))
			waveWrite(slotFrame, slotErr, "buffer slot csv")
			selfRemove, selfRemoveErr := channelBuffFrame(0, spec.selfBuff, playerObjectID, playerOwnerID, playerObjectID, playerOwnerID,
				playerName, playerName, epoch)
			waveWrite(selfRemove, selfRemoveErr, "self buff remove")
			// SP−10（先扣，官服 SP 掉帧先于伤害飘字）。
			if _, spFrame, spErr := player.addSP(-spec.spPerWave); spErr == nil && spFrame != nil {
				if err := link.WriteFrame(spFrame); err != nil {
					log.Printf("%s: channel wave SP frame: %v", remote, err)
				}
			}
			// m18 受击动作（官服带方向变体名 hurt_f_1_l_half）。
			hurtFrame, hurtErr := serverCustomIntMessage(18,
				customObject(ref.id, ref.ownerID),
				customString(channelHurtAction),
				customInt(0),
				customObject(playerObjectID, playerOwnerID))
			waveWrite(hurtFrame, hurtErr, "hurt action")
			victimRemove, victimRemoveErr := channelBuffFrame(0, spec.victimBuff, ref.id, ref.ownerID, playerObjectID, playerOwnerID,
				outcome.wideName, playerName, epoch)
			waveWrite(victimRemove, victimRemoveErr, "victim buff remove")
			// c1=71 每波效果帧（帧数与跳数严格相等的官服专属帧；第 11 值=
			// 当波动作名——客户端逐波播放挥砍动画的指令位）。
			effectFrame, effectErr := channelWaveEffectFrame(ref.id, ref.ownerID,
				channelWaveActionName(definition.actionName, waveIndex))
			waveWrite(effectFrame, effectErr, "wave effect")
			// 结算簇（FP/SP+2 → 0x13 → c1=17 → c1=10 段序 → 击杀簇 c1=29）。
			hits := []skillSettleHit{{ref: ref, damage: roll.damage, outcome: outcome}}
			if _, err := writeSkillSettleFrames(link, world, definition, player, hits, waveIndex, remote); err != nil {
				log.Printf("%s: channel wave %d/%d frames: %v", remote, waveIndex+1, spec.waves, err)
			}
			log.Printf("%s: channel wave %d/%d skill=%s victim=%d damage=%d sp_left=%d",
				remote, waveIndex+1, spec.waves, definition.id, ref.id, roll.damage, player.skillSP())
		})
	}
}

// channelWaveEffectFrame 即 msg24 c1=71 通道逐波效果帧
// （挑战.pcapng 满跳 4×8+2=34 帧与跳数严格相等）。**第 11 值=当波
// 动作名**（2026-11-12 真机三轮「瞬移有动作但无动画」根因：官服波 2~8
// 尾值=qzj_07_2..qzj_07_8 递增——客户端按名逐波播放挥砍动作，首波空串
// =动画由受理 m20 的段 1 动作 qzj_07_1 驱动）：
// [24, victim, 71, victim, '', 1, caster, 30.0, 0.0, '', waveAction]。
func channelWaveEffectFrame(victimID, victimOwner uint32, waveAction string) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(victimID, victimOwner),
		customInt(71),
		customObject(victimID, victimOwner),
		customString(""),
		customInt(1),
		customObject(playerObjectID, playerOwnerID),
		customFloat32(30.0),
		customFloat32(0.0),
		customString(""),
		customString(waveAction),
	)
}

// channelWaveActionName 波 N 的动作名：首波空串（受理 m20 已驱动），
// 波 2..N=<动作基名>_<波号>（qzj_07_1 → qzj_07_2..qzj_07_8 递增，
// 官服 4 满跳逐位复现；基名=definition.actionName 去尾 "_1"）。
func channelWaveActionName(actionName string, waveIndex int) string {
	if waveIndex == 0 {
		return ""
	}
	base := strings.TrimSuffix(actionName, "_1")
	if base == actionName {
		return actionName // 无 _1 后缀的异构动作名：全波同名兜底
	}
	return fmt.Sprintf("%s_%d", base, waveIndex+1)
}

// handleChannelSyncCustom 受理 C2S m914 通道逐段位置上报
// [914, x, y, z, orient, 12, 3]（挑战.pcapng：段数与跳数严格相等、跳后
// 0~200ms；尾参 12/3 恒定语义未定位）。窗口内更新通道权威位置并回
// 0x21 成对停帧（官服成对：归一 [0,2π) 角 + 原始朝向，spd=0——瞬移无
// 移动过程帧）；窗口外静默吞。
func handleChannelSyncCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomChannelSync {
		return false, nil
	}
	if player == nil {
		return true, nil
	}
	values := custom.Values
	if len(values) < 4 || values[1].Type != 4 || values[2].Type != 4 || values[3].Type != 4 {
		return true, nil // 宽容解析：坐标缺省静默吞
	}
	transform := worldcore.Transform{
		X: values[1].Float32,
		Y: values[2].Float32,
		Z: values[3].Float32,
	}
	if len(values) > 4 && values[4].Type == 4 {
		transform.Orient = values[4].Float32
	}
	if !player.noteChannelSync(time.Now(), transform) {
		return true, nil // 非通道窗口：静默吞
	}
	angle := transform.Orient
	for angle < 0 {
		angle += 2 * math.Pi
	}
	for angle >= 2*math.Pi {
		angle -= 2 * math.Pi
	}
	for _, echo := range []playerMoveEcho{
		{objectID: playerObjectID, ownerID: playerOwnerID, destX: transform.X, destY: transform.Y, destZ: transform.Z, angle: angle},
		{objectID: playerObjectID, ownerID: playerOwnerID, destX: transform.X, destY: transform.Y, destZ: transform.Z, angle: transform.Orient},
	} {
		if err := link.WriteFrame(serverAllDestPlayerEcho(echo)); err != nil {
			log.Printf("%s: channel sync echo: %v", remote, err)
			return true, nil
		}
	}
	return true, nil
}
