package main

// 受控机制（击飞 launch / 击退 push / 击倒 knockdown / 昏迷 coma 族）：
// 规格书《击飞击退击倒机制规格书.md》——章节以「§」引用，帧族证据=
// 群战.pcapng（规格书附录样本索引）。
//
//	§三  帧族总表：c56 伴随标记 / c79 击飞 / c78 击退 / State(183) 整型对
//	§四  帧型细则：juggle 裸帧、招架条件门、链式 c71（已有 wujiComboFloatFrame）
//	§五  控制规格解析链：M.compile control > M.on_hit control > desc_effects
//	§六  服务端权威：controlLease 租约（封招/封轻功/收编窗口/起身定时）
//	§七  Lua 挂钩：内功 control/break_control/control_immune
//
// 服务端零 0x21 驱动帧（§一铁律）：受控位移全部客户端动画自驱，事后
// m15 收编；观察者纯动画渲染（AOI 中继与命中帧族同路径）。

import (
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/neigong"
	"github.com/local/9yin-go-server/internal/skillscript"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// ---- 环境开关（§八）----

// controlEnabled 总闸（NINEYIN_CONTROL=off 关闭全部受控帧簇+租约，
// 回到纯 hurt_half 现状）。
func controlEnabled() bool {
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("NINEYIN_CONTROL")), "off")
}

// controlDescEnabled 描述数据面闸（NINEYIN_CONTROL_DESC=off 只认
// M.compile/on_hit 显式控制，desc_effects 层静默）。
func controlDescEnabled() bool {
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("NINEYIN_CONTROL_DESC")), "off")
}

// controlNPCFramesEnabled NPC 受害者帧族闸（NINEYIN_CONTROL_NPC=off
// 时 NPC 只冻结 AI 不发受控帧——客户端无倒地渲染、纯逻辑受控）。
func controlNPCFramesEnabled() bool {
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("NINEYIN_CONTROL_NPC")), "off")
}

// ---- 控制族常量（§二/§九）----

// 客户端受控状态机（State idx183 整型态，群战.pcapng 定版）：6 腾空 /
// 12 倒地 / 13 浮空受击·昏迷 / 1 常态（起身终点）。
const (
	controlStateNormal int32 = 1
	controlStateAir    int32 = 6
	controlStateDown   int32 = 12
	controlStateFloat  int32 = 13
)

// 击退帧尾常量（§三：1e+08 恒值原样复刻，疑似「无衰减速度」标记，
// 与 wujiComboFloatFrame 的 30.0 同款待钉死处理）。
const knockbackSpeedMarker = 100000000.0

// 击飞首发/浮空受击动画名（客户端 hurt_f 族，bin2 fxgamelogic.dll
// 字符串实证；c79 带 _l 长变体、m18 带 _s 短变体）。
const (
	launchFloatAction = "hurt_f_1_l"
	launchHurtAction  = "hurt_f_1_s"
)

// controlLeaseDefaultMS 是 §五缺省租约表（dur_ms=0 时按 kind 回落）。
func controlLeaseDefaultMS(kind string) int64 {
	switch kind {
	case "knockdown":
		return 2000
	case "launch":
		return 2200 // air 1200 + down 1000
	case "coma":
		return 3000 // yydf03 desc 实测 3s
	case "stun", "dizzy":
		return 2000
	case "push":
		return 800 // 滑行动画时长，不封招（§五表）
	}
	return 0
}

// controlAirDefaultMS 击飞空中段缺省（13→12 定时，§九）。
const controlAirDefaultMS = 1200

// controlBlocksCasting 报告该控制族是否封招/封轻功（§六：push 只滑行
// 不封——群战 hurtouting 四连窗口受害者可立即反击）。
func controlBlocksCasting(kind string) bool {
	switch kind {
	case "launch", "knockdown", "coma", "stun", "dizzy":
		return true
	}
	return false
}

// ---- 帧族构建（§三/§四）----

// controlStatePropertyIndex 是 State(183)（与 npcStatePropertyIndex 同一
// 属性；int 值经 WireInt32 原样落位=官方 float 位型方言）。
var controlStatePropertyIndex = clientdata.MustPlayerProperty("State").Index

// controlMarkerFrame 即 msg24 c1=56 受控段伴随标记（群战.pcapng 58 例
// 恒 ×2~3 簇发，取 2）：[24, obj(受害), 56]。
func controlMarkerFrame(victimID, victimOwner uint32) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(victimID, victimOwner),
		customInt(56),
	)
}

// launchControlFrame 即 msg24 c1=79 击飞帧（§4.1）：
// [24, obj(受害), 79, obj(攻击者), 窄串动画名]。首发带 "hurt_f_1_l"；
// 空中追击（juggle）传空串=裸帧维持浮空、不刷租约。
func launchControlFrame(victimID, victimOwner, attackerID, attackerOwner uint32, action string) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(victimID, victimOwner),
		customInt(79),
		customObject(attackerID, attackerOwner),
		customString(action),
	)
}

// knockbackControlFrame 即 msg24 c1=78 击退帧（§4.3/§4.4）：
// [24, obj(受害), 78, obj(攻击者), 窄串"", 窄串"", i32 N, f64 1e8]。
// N=方位参数（观测 0..180，疑攻击线偏角度；0=按攻击线 atan2 现算）。
func knockbackControlFrame(victimID, victimOwner, attackerID, attackerOwner uint32, angleDeg int32) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(victimID, victimOwner),
		customInt(78),
		customObject(attackerID, attackerOwner),
		customString(""),
		customString(""),
		customInt(angleDeg),
		customFloat64(knockbackSpeedMarker),
	)
}

// controlStateFrame 是 State(183) 整型推送（int 按 WireInt32 落位=官方
// float 位型方言）。击倒=6+12 同毫秒两推送（§4.2 官方 138 例全纯对）。
func controlStateFrame(objectID, ownerID uint32, state int32) ([]byte, error) {
	return sceneObjectProperties(objectID, ownerID, 0, []clientdata.IndexedProperty{
		{Index: controlStatePropertyIndex, Name: "State", Value: clientdata.Int32Value(state)},
	})
}

// skillHurtActionFrameAnim 是 msg18 受击动作的动画变体（§三：hurt_half
// 硬直 / hurt_f_1_s 浮空首发）。
func skillHurtActionFrameAnim(victimID, victimOwner, attackerID, attackerOwner uint32, action string) ([]byte, error) {
	if action == "" {
		action = "hurt_half"
	}
	return serverCustomIntMessage(18,
		customObject(victimID, victimOwner),
		customString(action),
		customInt(0),
		customObject(attackerID, attackerOwner),
	)
}

// writeControlFramesRelayed 把受控帧首发给攻方 link 并对视口内其它成员
// 中继（与命中表现帧族同路径；c61 AOI 裸广播哲学，§一铁律 2）。构建
// 错误记日志跳过、不中断余簇。
func (s *sceneLifecycle) writeControlFramesRelayed(link sceneMessageConnection, victimID uint32, frame []byte, err error) {
	if err != nil {
		log.Printf("control frame encode victim=%d: %v", victimID, err)
		return
	}
	if s == nil || s.run == nil {
		return
	}
	if link != nil {
		if werr := link.WriteFrame(frame); werr != nil {
			log.Printf("control frame victim=%d: %v", victimID, werr)
		}
	}
	s.run.relayCombatFrame(victimID, frame, s, "control")
}

// ---- 控制规格解析链（§五）----

// controlRollSalt 受控几率掷盐（与内功 31..35、既有 17/18 掷错开）。
const controlRollSalt = 41

// resolveControlSpec 解析一次命中的受控规格：① definition.control
// （M.compile control 指令，编译期合并）> ② M.on_hit 运行期 control 键
// > ③ desc_effects DescRule（NINEYIN_CONTROL_DESC 闸）。三者全 miss
// 或总闸关闭 → nil（不受控，纯 hurt_half 现状）。on_hit 调用恒 1 枚
// 预掷（RNG 契约同 HitRatio）。
func resolveControlSpec(definition combatSkillDefinition, player *playerActor, victimID uint32, now time.Time) *skillscript.ControlSpec {
	if !controlEnabled() {
		return nil
	}
	if definition.control != nil {
		return definition.control
	}
	if skillScripts != nil && skillScripts.Enabled() {
		if result := skillScripts.OnHit(definition.id, &skillscript.HitCtx{
			Level:   int64(definition.level),
			RAdjust: float64(combatPercentRoll(victimID, now, controlRollSalt)) / 100,
		}); result != nil {
			return result.Control
		}
	}
	if !controlDescEnabled() || skillScripts == nil || !skillScripts.Enabled() {
		return nil
	}
	entry := skillScripts.DescEffects(definition.id)
	if entry == nil {
		return nil
	}
	for _, rule := range entry.OnHit {
		if spec := controlSpecFromDescRule(rule); spec != nil {
			return spec
		}
	}
	return nil
}

// controlSpecFromDescRule 把 desc_effects 的 DescRule 折成受控规格
// （§五②映射表）：ControlKind launch/knockdown/coma/stun/dizzy 直取
// （位移副键 push 并入 launch 不另发帧）；纯 displacement push 独立成
// 击退；pull 二期不受理。条件门（unparried/parry_broken）与几率透传，
// 由 controlAllowed 裁定。
func controlSpecFromDescRule(rule *skillscript.DescRule) *skillscript.ControlSpec {
	if rule == nil {
		return nil
	}
	spec := &skillscript.ControlSpec{}
	switch rule.ControlKind {
	case "launch", "knockdown", "coma", "stun", "dizzy":
		spec.Kind = rule.ControlKind
	case "":
		if rule.DisplacementKind != "push" {
			return nil // pull/无控制：二期或不受控
		}
		spec.Kind = "push"
	default:
		// slow/root/silence/ban_qinggong/stiffen：无状态机推送，纯 buff
		// 租约路径（skillEffectTargetControl），不在本规格。
		return nil
	}
	spec.DurMS = rule.ControlDurMS
	spec.DistM = rule.DisplacementDistanceM
	spec.Chance = rule.Chance
	if rule.Condition != nil {
		switch {
		case rule.Condition.Unparried:
			spec.Condition = "unparried"
		case rule.Condition.ParryBroken:
			spec.Condition = "parry_broken"
		}
	}
	return spec
}

// controlAllowed 条件门与几率掷（§4.5）。NPC 受害者无招架位
// （unparried 恒过）；parry_broken 对 NPC 恒 false（假人格挡属二期）。
// 玩家受害者的招架/霸体/免控窗在 applyControlToPlayer 内再判。
func controlAllowed(spec *skillscript.ControlSpec, victimID uint32, now time.Time) bool {
	if spec == nil || spec.Kind == "" {
		return false
	}
	if spec.Chance > 0 {
		// [0,1) 掷：预掷百分点值 < chance*100。
		if float64(combatPercentRoll(victimID, now, controlRollSalt)) >= spec.Chance*100 {
			return false
		}
	}
	switch spec.Condition {
	case "unparried":
		return true // NPC 受害者恒过；玩家侧在 applyControlToPlayer 判
	case "parry_broken":
		return false // NPC 无破防态（二期接线）
	}
	return true
}

// ---- NPC 受控租约（§六；sceneRun.npcControl 表）----

// npcControlLease 是受害 NPC 的受控租约（run.mu 保护；§六模型）。
// recoverAt 是恢复定时器的对账戳：租约被刷新后旧定时器按戳失配退场。
type npcControlLease struct {
	kind       string
	until      time.Time
	attackerID uint32
	recoverAt  time.Time
}

// applyControlFrames 对一条命中落受控帧簇+租约（命中链在
// writeSkillSettleFrames 每击调用；击杀命中跳过——死亡链优先，§六）。
// 返回是否落控（日志锚 §十）。
func applyControlFrames(link sceneMessageConnection, world *sceneLifecycle, definition combatSkillDefinition, player *playerActor, victimID, victimOwner uint32, now time.Time) bool {
	if !controlEnabled() || world == nil || world.run == nil {
		return false
	}
	if !controlNPCFramesEnabled() {
		return applyNPCControlLeaseOnly(world, definition.id, victimID, now)
	}
	spec := resolveControlSpec(definition, player, victimID, now)
	return applyControlSpecFrames(link, world, spec, player, victimID, victimOwner, now, definition.id)
}

// applyControlSpecFrames 落一条已解析的受控规格（§四帧簇+§六租约）。
// juggle 规则（§4.1）：已在 launch 租约内再被 launch 命中=裸 c79 维持、
// 不刷租约。skillID 仅日志用。
func applyControlSpecFrames(link sceneMessageConnection, world *sceneLifecycle, spec *skillscript.ControlSpec, player *playerActor, victimID, victimOwner uint32, now time.Time, skillID string) bool {
	if world == nil || world.run == nil {
		return false
	}
	if !controlAllowed(spec, victimID, now) {
		return false
	}
	// 默认值回落（§五缺省表）。
	duration := spec.DurMS
	if duration <= 0 {
		duration = controlLeaseDefaultMS(spec.Kind)
	}
	if duration <= 0 {
		return false
	}
	attacker, attackerOwner := playerObjectID, playerOwnerID
	if player == nil {
		attacker, attackerOwner = 0, 0
	}

	world.run.mu.Lock()
	lease := world.run.npcControl[victimID]
	juggle := lease != nil && lease.kind == "launch" && spec.Kind == "launch" && now.Before(lease.until)
	if juggle {
		world.run.mu.Unlock()
		// 空中追击：裸 c79 维持浮空，租约不刷（群战 5.279~6.776 六连发）。
		frame, err := launchControlFrame(victimID, victimOwner, attacker, attackerOwner, "")
		world.writeControlFramesRelayed(link, victimID, frame, err)
		log.Printf("control juggle victim=%d skill=%s (lease kept)", victimID, skillID)
		return true
	}
	until := now.Add(time.Duration(duration) * time.Millisecond)
	world.run.setNPCControlLocked(victimID, &npcControlLease{
		kind: spec.Kind, until: until, attackerID: attacker, recoverAt: until,
	})
	// AI 冻结（§六）：Brain.ControlledUntil 取大者；push 不冻结。
	world.run.freezeNPCBrainLocked(victimID, spec.Kind, until)
	world.run.mu.Unlock()

	// 帧簇按 §四逐 kind（首发全走攻方 link + AOI 中继）。
	angle := knockbackAngleDeg(world, player, victimID, spec)
	switch spec.Kind {
	case "launch":
		frame, err := skillHurtActionFrameAnim(victimID, victimOwner, attacker, attackerOwner, launchHurtAction)
		world.writeControlFramesRelayed(link, victimID, frame, err)
		marker, err := controlMarkerFrame(victimID, victimOwner)
		world.writeControlFramesRelayed(link, victimID, marker, err)
		world.writeControlFramesRelayed(link, victimID, marker, err)
		frame, err = launchControlFrame(victimID, victimOwner, attacker, attackerOwner, launchFloatAction)
		world.writeControlFramesRelayed(link, victimID, frame, err)
		frame, err = controlStateFrame(victimID, victimOwner, controlStateFloat)
		world.writeControlFramesRelayed(link, victimID, frame, err)
	case "knockdown":
		marker, err := controlMarkerFrame(victimID, victimOwner)
		world.writeControlFramesRelayed(link, victimID, marker, err)
		frame, err := controlStateFrame(victimID, victimOwner, controlStateAir)
		world.writeControlFramesRelayed(link, victimID, frame, err)
		frame, err = controlStateFrame(victimID, victimOwner, controlStateDown)
		world.writeControlFramesRelayed(link, victimID, frame, err)
	case "push":
		marker, err := controlMarkerFrame(victimID, victimOwner)
		world.writeControlFramesRelayed(link, victimID, marker, err)
		world.writeControlFramesRelayed(link, victimID, marker, err)
		frame, err := knockbackControlFrame(victimID, victimOwner, attacker, attackerOwner, angle)
		world.writeControlFramesRelayed(link, victimID, frame, err)
	case "coma", "stun", "dizzy":
		marker, err := controlMarkerFrame(victimID, victimOwner)
		world.writeControlFramesRelayed(link, victimID, marker, err)
		world.writeControlFramesRelayed(link, victimID, marker, err)
		frame, err := controlStateFrame(victimID, victimOwner, controlStateFloat)
		world.writeControlFramesRelayed(link, victimID, frame, err)
	}

	// 定时链（§六）：launch 的 airMS 后 12；租约到推 State=1（官方起身
	// 零客户端申报，服务端定时权威）。
	if spec.Kind == "launch" {
		airMS := spec.AirMS
		if airMS <= 0 {
			airMS = controlAirDefaultMS
		}
		scheduleNPCControlState(link, world, victimID, victimOwner, controlStateDown, now.Add(time.Duration(airMS)*time.Millisecond))
	}
	scheduleNPCControlState(link, world, victimID, victimOwner, controlStateNormal, until)
	log.Printf("control applied kind=%s skill=%s victim=%d lease=%dms", spec.Kind, skillID, victimID, duration)
	return true
}

// npcControlEvent 是战斗 tick 收集的内功反震受控事件（run.mu 内产生、
// 锁外结算——applyControlSpecFrames 自持 run.mu）。view/conn 取自受害
// 成员快照（首发 link），与 parryBreaks 的收集模式一致。
type npcControlEvent struct {
	npcID   uint32
	ownerID uint32
	player  *playerActor
	view    *sceneLifecycle
	control *neigong.ControlEffect
}

// applyNeiGongControls 在战斗 tick 锁外结算内功反震受控事件（§7.2：
// on_parry control 指令 → 攻击 NPC 吃受控帧簇+租约）。
func applyNeiGongControls(events []npcControlEvent) {
	for _, event := range events {
		if event.control == nil || event.player == nil || event.view == nil {
			continue
		}
		spec := &skillscript.ControlSpec{
			Kind:  event.control.Kind,
			DurMS: event.control.DurMS,
			DistM: event.control.DistM,
		}
		applyControlSpecFrames(event.view.conn, event.view, spec, event.player, event.npcID, event.ownerID, time.Now(), "neigong_parry")
	}
}

// applyNPCControlLeaseOnly 是 NINEYIN_CONTROL_NPC=off 的纯租约形：
// 只冻结 AI（§六）不发帧（客户端无渲染、逻辑受控）。
func applyNPCControlLeaseOnly(world *sceneLifecycle, skillID string, victimID uint32, now time.Time) bool {
	spec := resolveControlSpecQuiet(skillID, victimID, now)
	if !controlAllowed(spec, victimID, now) {
		return false
	}
	duration := spec.DurMS
	if duration <= 0 {
		duration = controlLeaseDefaultMS(spec.Kind)
	}
	if duration <= 0 {
		return false
	}
	until := now.Add(time.Duration(duration) * time.Millisecond)
	world.run.mu.Lock()
	world.run.setNPCControlLocked(victimID, &npcControlLease{kind: spec.Kind, until: until, recoverAt: until})
	world.run.freezeNPCBrainLocked(victimID, spec.Kind, until)
	world.run.mu.Unlock()
	log.Printf("control applied kind=%s skill=%s victim=%d lease=%dms (frames off)", spec.Kind, skillID, victimID, duration)
	return true
}

// resolveControlSpecQuiet 是 resolveControlSpec 的无玩家形（on_hit 层需要
// player 上下文，纯租约形跳过——M.compile/desc 两层照常）。
func resolveControlSpecQuiet(skillID string, victimID uint32, now time.Time) *skillscript.ControlSpec {
	if !controlEnabled() {
		return nil
	}
	if skillScripts != nil && skillScripts.Enabled() {
		if overlay := skillScripts.Overlay(skillID, nil); overlay != nil && overlay.Control != nil {
			return overlay.Control
		}
	}
	if !controlDescEnabled() || skillScripts == nil || !skillScripts.Enabled() {
		return nil
	}
	entry := skillScripts.DescEffects(skillID)
	if entry == nil {
		return nil
	}
	for _, rule := range entry.OnHit {
		if spec := controlSpecFromDescRule(rule); spec != nil {
			return spec
		}
	}
	return nil
}

// freezeNPCBrainLocked 把 Brain.ControlledUntil 抬到租约终点（push 族
// 不冻结，§六）。run.mu 内。
func (r *sceneRun) freezeNPCBrainLocked(victimID uint32, kind string, until time.Time) {
	if !controlBlocksCasting(kind) {
		return
	}
	id := worldcore.EntityID(victimID)
	brain, ok := r.combat.Brain(id)
	if !ok {
		return
	}
	if until.After(brain.ControlledUntil) {
		brain.ControlledUntil = until
		r.combat.WriteBrain(id, brain)
	}
}

// knockbackAngleDeg 是 c78 方位参数：显式 angle_deg 优先，缺省按攻击线
// atan2 现算（0=正北顺时针——§4.4 方位角假说，真机回归翻案位）。
func knockbackAngleDeg(world *sceneLifecycle, player *playerActor, victimID uint32, spec *skillscript.ControlSpec) int32 {
	if spec.AngleDeg > 0 {
		return int32(spec.AngleDeg)
	}
	if player == nil || world == nil {
		return 0
	}
	transform, exists := world.npcTransform(worldcore.EntityID(victimID))
	if !exists {
		return 0
	}
	px, _, pz := player.clientReportPositionSnapshot()
	return int32(math.Mod(math.Atan2(float64(transform.X-px), float64(transform.Z-pz))*180/math.Pi+360, 360))
}

// scheduleNPCControlState 受控恢复定时（schedulePlayerBuffExpiry 同款
// AfterFunc 模式）。到期时租约仍在（recoverAt 对账戳匹配、未被新受控
// 刷新）才发 State 帧；空中段（12）迟到于租约终点即静默退场。
func scheduleNPCControlState(link sceneMessageConnection, world *sceneLifecycle, victimID, victimOwner uint32, state int32, at time.Time) {
	delay := time.Until(at)
	if delay < 0 {
		delay = 0
	}
	time.AfterFunc(delay, func() {
		if world == nil || world.run == nil {
			return
		}
		world.run.mu.Lock()
		lease := world.run.npcControl[victimID]
		if lease == nil {
			world.run.mu.Unlock()
			return
		}
		if state == controlStateNormal {
			if !at.Equal(lease.recoverAt) {
				world.run.mu.Unlock()
				return // 租约已被刷新：旧恢复定时器退场
			}
			delete(world.run.npcControl, victimID)
		} else if time.Now().After(lease.until) {
			world.run.mu.Unlock()
			return // 空中段迟到：租约已到期走恢复
		}
		world.run.mu.Unlock()
		frame, err := controlStateFrame(victimID, victimOwner, state)
		world.writeControlFramesRelayed(link, victimID, frame, err)
		if state == controlStateNormal {
			log.Printf("control recover victim=%d state=1", victimID)
		}
	})
}

// setNPCControlLocked 写/刷新 NPC 受控租约（run.mu 内；惰性建表）。
// 后到者优先整租约替换——旧恢复定时器按 recoverAt 对账戳自动退场。
func (r *sceneRun) setNPCControlLocked(victimID uint32, lease *npcControlLease) {
	if r.npcControl == nil {
		r.npcControl = make(map[uint32]*npcControlLease)
	}
	r.npcControl[victimID] = lease
}

// clearNPCControlLocked 清除 NPC 受控租约（死亡/注销；run.mu 内）。
func (r *sceneRun) clearNPCControlLocked(victimID uint32) {
	if r.npcControl == nil {
		return
	}
	delete(r.npcControl, victimID)
}

// ---- 玩家侧租约（§六；当前无受控施加源——PvP/NPC 控制技二期接入，
// 闸门先行：m211 封招/JumpTimes 封轻功/位置收编窗口/起身定时）----

// playerControlLease 是玩家受害者的受控租约（p.mu 保护）。
type playerControlLease struct {
	kind       string
	until      time.Time
	attackerID uint32
	recoverAt  time.Time
}

// controlCastBlocked 是 m211 受理闸（§六封招）：封招族租约活跃时返回
// 控制族名（日志用），未受控返回空串。
func (p *playerActor) controlCastBlocked(now time.Time) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.controlCastBlockedLocked(now) {
		return p.controlLease.kind
	}
	return ""
}

// controlCastBlockedLocked 是封招闸的锁内实现。调用方必须已持有 p.mu。
func (p *playerActor) controlCastBlockedLocked(now time.Time) bool {
	return p.controlLease.kind != "" && controlBlocksCasting(p.controlLease.kind) && now.Before(p.controlLease.until)
}

// controlSnapshotLocked 返回受控租约快照（neigong Ctx.Controlled/
// ControlKind/ControlRemainMS 的真值来源；调用方必须已持有 p.mu）。
func (p *playerActor) controlSnapshotLocked(now time.Time) (kind string, remainMS int64, controlled bool) {
	if p.controlLease.kind == "" || now.After(p.controlLease.until) {
		return "", -1, false
	}
	remain := p.controlLease.until.Sub(now).Milliseconds()
	if remain < 0 {
		remain = 0
	}
	return p.controlLease.kind, remain, controlBlocksCasting(p.controlLease.kind)
}

// applyPlayerControlImmune 落免控窗（§7.2 control_immune；p.mu 内由
// hookNeiGong 调用方持锁）。
func (p *playerActor) applyPlayerControlImmune(lifeMS int64, now time.Time) {
	until := now.Add(time.Duration(lifeMS) * time.Millisecond)
	if until.After(p.controlImmuneUntil) {
		p.controlImmuneUntil = until
	}
}

// controlImmunedLocked 报告免控窗是否活跃（§4.5 条件门恒过）。
// 调用方必须已持有 p.mu。
func (p *playerActor) controlImmunedLocked(now time.Time) bool {
	return now.Before(p.controlImmuneUntil)
}

// breakPlayerControlLocked 解控（§7.2 break_control：清租约并起身）。
// 返回是否确有租约被清。调用方必须已持有 p.mu。
func (p *playerActor) breakPlayerControlLocked(now time.Time) bool {
	if p.controlLease.kind == "" {
		return false
	}
	p.controlLease = playerControlLease{}
	return true
}
