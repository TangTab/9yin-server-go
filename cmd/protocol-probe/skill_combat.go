package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/sim"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

const (
	clientCustomUseSkill int32 = 211

	// 当前 FxGameLogic 的 GameMessageHandler 绑定构造函数把
	// sub_110120E4 -> sub_129C0330 存放在配对 55/56 处，
	// 因此其现代数字消息号为 (55-15)/2 == 20。还原出的接收端读取：
	// [2] 施法者对象、[3] 招式 ConfigID、[4] 目标对象，
	// 然后进入 SkillEffect/CurSkillCanNoLimit 的后续流程。与
	// 消息 18 不同，这是招式动作路线而不是通用 DoAction。
	serverNativeSkillActionMessage int32 = 20

	// 已安装的 buff_new.ini/buff_static.ini：
	// buf_CS_yhwq_hsqs04 -> StaticData 15101，可见的自身 Buff。
	// 其一级的隐藏重触发租期为 35 秒（StaticData 15104）。
	yiHuaSkill04BuffStaticData uint32 = 15101
	yiHuaSkill04BuffLease             = 35 * time.Second
	// buff_new.ini/buff_static.ini/buff_varprop.ini 一级：
	// buf_CS_yhwq_hsqs05 -> StaticData 15105，可见，LifeTime 5000 ms。
	yiHuaSkill05BuffStaticData uint32 = 15105
	yiHuaSkill05BuffLease             = 5 * time.Second

	// 已安装的 太极拳(古谱) Buff 路线。招式 04 在两条
	// 持久可见的架势记录之间轮换；招式 05 和 07 使用其
	// 精确的一级 buff_varprop 生存期。
	taiJiDefenceStanceBuffStaticData uint32 = 6013
	taiJiAttackStanceBuffStaticData  uint32 = 6055
	taiJiSkill05BuffStaticData       uint32 = 6043
	taiJiSkill07BuffStaticData       uint32 = 6050
	taiJiStanceBuffLease                    = 24 * time.Hour
	taiJiSkill05BuffLease                   = 15 * time.Second
	taiJiSkill07BuffLease                   = 8 * time.Second
)

type skillActionSegment struct {
	action   string
	at       time.Duration
	duration time.Duration
}

type skillTargetMode uint8

const (
	skillTargetSelf skillTargetMode = iota
	skillTargetSelected
	skillTargetCasterCircle
	skillTargetCasterSector
	skillTargetCasterRectangle
	skillTargetSelectedArea
)

type skillEffectKind uint8

const (
	skillEffectDamage skillEffectKind = iota
	skillEffectPlayerBuff
	skillEffectTargetControl
	skillEffectRedArmor
	skillEffectStanceCycle
)

type skillBuffDefinition struct {
	configID      string
	staticData    uint32
	level         int32
	lifetime      time.Duration
	preferredSlot uint16
}

type combatSkillEffect struct {
	kind         skillEffectKind
	amount       int32
	buff         skillBuffDefinition
	buffVariants []skillBuffDefinition
}

type combatSkillDefinition struct {
	id               string
	staticData       int32
	script           string
	taoLu            string
	level            int32
	cooldownCategory int32
	cooldownTeam     int32
	personalCD       time.Duration
	publicCD         time.Duration
	mpCost           int32
	spCost           int32
	qgCost           int32
	baseDamage       int32
	requiresTarget   bool
	targetMode       skillTargetMode
	areaRadius       float32
	areaHeight       float32
	sectorAngle      float32
	areaWidth        float32
	actionName       string
	followupActions  []skillActionSegment
	actionDuration   time.Duration
	effects          []combatSkillEffect
	// 冲锋链字段（Java CombatSkill/SkillCastHandler 同源）：
	// castRange=TargetShapePara2>0 ? TSP2 : max(形状半径,8m)；
	// canRush=skill_static.CanRush；lockSkill=script=="skilllock"||canRush||
	// (TargetType==4&&DamageType!=0)（msg20 动作三分用）；
	// isDamageSkill=skill_static.IsDamage；bind/groundRushAction=
	// skill_lock_data[LockType] 的 BindDistance/LockGRushAction（缺行回退）。
	castRange        float32
	canRush          bool
	lockSkill        bool
	isDamageSkill    bool
	bindDistance     float32
	groundRushAction string
}

// stagedRushFamily 报告该技能是否走冲锋暂存链：
// Java 语义 canRush && script=="skilllock"（大小写不敏感）。
func (definition combatSkillDefinition) stagedRushFamily() bool {
	return definition.canRush && strings.EqualFold(definition.script, "skilllock")
}

func (definition combatSkillDefinition) totalActionDuration() time.Duration {
	duration := definition.actionDuration
	for _, segment := range definition.followupActions {
		if segment.at+segment.duration > duration {
			duration = segment.at + segment.duration
		}
	}
	return duration
}

// skillCooldownFrames 产出出招 CD 双行（§3 行 1/3 顺序，中间夹
// active_skill_rec 行由调用方写入）：个人行 team 恒 0xFFFF；GCD 行
// 类别固定 50000、时长取 varprop AddCoolDownTime（publicCD）、team 取
// skill_static CoolDownTeam。
func skillCooldownFrames(definition combatSkillDefinition, now time.Time) (personal, public []byte, err error) {
	start := now.UnixMilli()
	// 官方 cooldown_rec 第 4 列 team 是 u16（0xFFFF 个人行 /
	// CoolDownTeam GCD 行），此前自协商方言误声明为 u32。
	personal, err = serverRecordAddCells(playerObjectID, playerOwnerID, recordCooldown, []recordCell{
		recordInt(uint32(definition.cooldownCategory)),
		recordInt64(start),
		recordInt64(start + definition.personalCD.Milliseconds()),
		recordInt(uint32(0xFFFF)),
	})
	if err != nil {
		return nil, nil, err
	}
	public, err = serverRecordAddCells(playerObjectID, playerOwnerID, recordCooldown, []recordCell{
		recordInt(uint32(50000)),
		recordInt64(start),
		recordInt64(start + definition.publicCD.Milliseconds()),
		recordInt(uint32(definition.cooldownTeam)),
	})
	return personal, public, err
}

// writeSkillHitEventFrames 按 §4 每目标序列补发命中表现帧族：
// 0x13 战斗关系（玩家对象上的记录格，仅施法者自己）→ msg24 伤害
// 飘字 → 0x2D 受击特效 → msg18 受击动作 → msg205 命中确认。
// 后四帧经 link 首发后对视口内可见受害 NPC 的其他成员中继。
func writeSkillHitEventFrames(link sceneMessageConnection, world *sceneLifecycle, definition combatSkillDefinition, player *playerActor, victimID, victimOwner uint32, damage, hpRatio int32, victimModel string, cumulative int32) error {
	if err := link.WriteFrame(fightRelationDamageFrame(playerObjectID, playerOwnerID, cumulative)); err != nil {
		return fmt.Errorf("write fight relation frame: %w", err)
	}
	hitFrame, err := skillHitDamageFrame(victimID, victimOwner, damage, definition.id, victimModel, player.snapshotName())
	if err != nil {
		return fmt.Errorf("encode skill hit frame: %w", err)
	}
	effectFrame := hurtEffectFrame(victimID, victimOwner, hpRatio)
	hurtFrame, err := skillHurtActionFrame(victimID, victimOwner, playerObjectID, playerOwnerID)
	if err != nil {
		return fmt.Errorf("encode skill hurt frame: %w", err)
	}
	confirmFrame, err := skillHitConfirmFrame(playerObjectID, playerOwnerID, definition.id, victimID, victimOwner)
	if err != nil {
		return fmt.Errorf("encode skill confirm frame: %w", err)
	}
	for _, frame := range [][]byte{hitFrame, effectFrame, hurtFrame, confirmFrame} {
		if err := link.WriteFrame(frame); err != nil {
			return fmt.Errorf("write skill hit event frame: %w", err)
		}
	}
	// 命中表现帧族中继：飘字/受击特效/动作/确认对可见受害 NPC 的
	// 旁观者原样转发（0x13 是玩家私有记录格，不中继）。
	if world != nil && world.run != nil {
		for _, frame := range [][]byte{hitFrame, effectFrame, hurtFrame, confirmFrame} {
			world.run.relayCombatFrame(victimID, frame, world, "skill hit")
		}
	}
	return nil
}

func nativeSkillActionFrame(skillID string, targetID, targetOwnerID uint32) ([]byte, error) {
	if skillID == "" {
		return nil, fmt.Errorf("native skill action ID is empty")
	}
	return serverModernCustomIntMessage(
		serverNativeSkillActionMessage,
		customObject(playerObjectID, playerOwnerID),
		customString(skillID),
		customObject(targetID, targetOwnerID),
		// 受保护的接收端接受这两个可选字段。提供完整的
		// 现代形态可以让逐行的 show runner 保留角色
		// 位移和动作时序，而不是依赖省略字段时的默认值。
		customInt(1),
		customFloat32(1),
	)
}

// skillActionPreparationFrame 在 ID20 之前立即重放两条原生玩家动作属性。
// 客户端注册了真实的 on_actionset 回调：它会进入战斗动作服务
// 并同步重建动作前缀/当前动作状态。PlayerAddObject 已携带相同的值，
// 但那份初始归档可能在异步的角色模型/动作组件就绪之前到达；
// 在 ClientReady 之后显式刷新一次对象属性
// 可以弥补这个时序缺口，且不会改变选定的外观。
func skillActionPreparationFrame() ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 406, Name: "ActionSet", Value: clientdata.StringValue("0h")},
		{Index: 183, Name: "State", Value: clientdata.StringValue("stand")},
	})
}

// playerActionStateFrame 驱动普通的玩家 State 属性回调。
// 它与 skill_action 刻意分离：已安装的生活技能 Lua 会对
// 玩家动作组件使用相同的动作名（例如钓鱼的 interact268），
// 因此它是一个有用的独立可达性探针。
func playerActionStateFrame(action string) ([]byte, error) {
	if action == "" {
		return nil, fmt.Errorf("player action state is empty")
	}
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 183, Name: "State", Value: clientdata.StringValue(action)},
	})
}

// nativeGenericActionFrame 触达当前客户端的通用 "action"
// 命令接收端（sub_129C0E90）。其原生实现从 TVarList[2] 读取目标、
// 从 [3] 读取动作，可选择清除当前动作，
// 然后调用动作服务。注册表槽位并不是数字形式的线上传输 ID：
// TVarList 的首个值必须是字符串形式的命令名。
func nativeGenericActionFrame(action string) ([]byte, error) {
	if action == "" {
		return nil, fmt.Errorf("native generic action is empty")
	}
	return serverModernCustomStringMessage(
		"action",
		customObject(playerObjectID, playerOwnerID),
		customString(action),
		customString("1"),
	)
}

func parseUseSkillCustom(custom clientCustomMessage) (combatSkillDefinition, bool) {
	if len(custom.Values) < 2 || custom.Values[0].Type != 2 ||
		custom.Values[0].Int32 != clientCustomUseSkill ||
		custom.Values[1].Type != 6 || custom.Values[1].Text == "" {
		return combatSkillDefinition{}, false
	}
	definition, exists := combatSkills[custom.Values[1].Text]
	return definition, exists
}

func useSkillCastTransform(custom clientCustomMessage) (worldcore.Transform, bool) {
	if len(custom.Values) < 5 ||
		custom.Values[2].Type != 4 ||
		custom.Values[3].Type != 4 ||
		custom.Values[4].Type != 4 {
		return worldcore.Transform{}, false
	}
	transform := worldcore.Transform{
		X: custom.Values[2].Float32,
		Y: custom.Values[3].Float32,
		Z: custom.Values[4].Float32,
	}
	// 现代版 C2S 211 在可选的目标位置之前还跟随施法者朝向。
	// 扇形技能的朝向判定需要这个值。
	if len(custom.Values) > 5 && custom.Values[5].Type == 4 {
		transform.Orient = custom.Values[5].Float32
	}
	return transform, true
}

// resolveSkillActionTarget 是 msg20/msg19 动作帧的目标对象三分
// （Java resolveActionTarget 同源）：锁定/冲锋技=211 瞄准对象（校验存活
// 战斗 NPC，失效回落选中目标）——客户端据此本地执行冲锋位移；自增益招
// （IsDamage=0）=施法者自己；其余伤害技=空对象 obj(0,0)。
func resolveSkillActionTarget(definition combatSkillDefinition, custom clientCustomMessage,
	world *sceneLifecycle, selectedID, selectedOwner uint32) (uint32, uint32) {
	if definition.lockSkill {
		if aimID, aimOwner := dashLandingAimObject(custom); aimID != 0 && world.combatNPCAlive(aimID, aimOwner) {
			return aimID, aimOwner
		}
		if selectedID != 0 && selectedID != playerObjectID {
			return selectedID, selectedOwner
		}
		return 0, 0
	}
	if !definition.isDamageSkill {
		return playerObjectID, playerOwnerID
	}
	return 0, 0
}

func (p *playerActor) beginSkillUse(definition combatSkillDefinition, target uint64, now time.Time) (bool, string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.skillLockUntil.After(now) {
		return false, fmt.Sprintf("skill locked remaining=%s", p.skillLockUntil.Sub(now).Round(time.Millisecond))
	}
	if ready := p.skillCooldowns[definition.id]; ready.After(now) {
		return false, fmt.Sprintf("personal cooldown remaining=%s", ready.Sub(now).Round(time.Millisecond))
	}
	teamKey := definition.cooldownTeam
	if teamKey == 0 {
		teamKey = -definition.cooldownCategory
	}
	if ready := p.skillTeamCooldowns[teamKey]; ready.After(now) {
		return false, fmt.Sprintf("public cooldown team=%d remaining=%s", teamKey, ready.Sub(now).Round(time.Millisecond))
	}
	state := p.snapshotLocked()
	if state.LogicState == entity.LogicStateDied || state.HP <= 0 {
		return false, "player is dead"
	}
	if state.MP < definition.mpCost {
		return false, fmt.Sprintf("MP %d < %d", state.MP, definition.mpCost)
	}
	if p.progress.attributes.sp < definition.spCost {
		return false, fmt.Sprintf("SP %d < %d", p.progress.attributes.sp, definition.spCost)
	}
	if definition.qgCost > 0 {
		if cost, ok := p.qinggong.Use(definition.qgCost, now); !ok {
			return false, fmt.Sprintf("QingGongPoint cannot pay %d (resolved cost %d)", definition.qgCost, cost)
		}
	}
	if !p.vitals.SpendMP(definition.mpCost) {
		return false, "MP spend rejected"
	}
	p.progress.attributes.sp -= definition.spCost
	p.skillCooldowns[definition.id] = now.Add(definition.personalCD)
	p.skillTeamCooldowns[teamKey] = now.Add(definition.publicCD)
	p.currentSkillID = definition.id
	p.currentSkillEffectID = definition.id
	p.currentSkillLevel = definition.level
	p.currentSkillTarget = target
	p.vitals.SetLogicState(entity.LogicStateFighting)
	return true, ""
}

func (p *playerActor) skillSP() int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.attributes.sp
}

func (s *sceneLifecycle) selectedCombatTarget() (uint32, uint32, bool) {
	if s == nil {
		return 0, 0, false
	}
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	id := worldcore.EntityID(s.selectedObject)
	vitals, hasVitals := s.run.combat.Vitals(id)
	if !hasVitals || vitals.HP <= 0 {
		return 0, 0, false
	}
	meta, exists := s.npcMeta(id)
	if !exists || meta.ownerID != s.selectedOwner || meta.interaction != npcInteractionCombat {
		return 0, 0, false
	}
	return uint32(id), meta.ownerID, true
}

// skillDamageOutcome 是一次技能伤害结算的完整产物：除扣血 0x10 帧
// （frame）与掉落外，还携带命中表现帧族所需的素材——受击后
// HPRatio（0x2D 两次）、受害 NPC 模型/宽名（msg24）、以及 0x13
// fight_relation_rec 的最新累计伤害。
type skillDamageOutcome struct {
	frame      []byte
	id         uint32
	ownerID    uint32
	loot       int32
	killed     bool
	hpRatio    int32
	model      string
	wideName   string
	cumulative int32
}

func (s *sceneLifecycle) applySelectedSkillDamage(amount int32, now time.Time) (skillDamageOutcome, error) {
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	id := worldcore.EntityID(s.selectedObject)
	vitals, hasVitals := s.run.combat.Vitals(id)
	meta, exists := s.npcMeta(id)
	if !hasVitals || !exists || meta.ownerID != s.selectedOwner || meta.interaction != npcInteractionCombat || vitals.HP <= 0 {
		return skillDamageOutcome{}, nil
	}
	if !vitals.ApplyDamage(amount) {
		return skillDamageOutcome{}, nil
	}
	s.run.combat.WriteVitals(id, vitals)
	state := vitals.ActorState(uint32(id), meta.ownerID)
	if brain, isCombat := s.run.combat.Brain(id); isCombat {
		brain.Threat += amount
		if state.HP <= 0 {
			brain.Threat = 0
			if npcRespawnBatchInterval() > 0 {
				// 批量刷新：不设每怪计时，refreshTick 扫描统一复活。
				brain.RespawnAt = time.Time{}
			} else {
				brain.RespawnAt = now.Add(brain.RespawnAfter)
			}
			s.run.recordNPCDeathLocked(uint32(id), meta.ownerID, brain, now)
		}
		s.run.combat.WriteBrain(id, brain)
	}
	frame, err := sceneObjectProperties(uint32(id), meta.ownerID, 0, npcCombatProperties(state))
	if err != nil {
		return skillDamageOutcome{}, err
	}
	loot := int32(0)
	if state.HP <= 0 {
		loot = state.MaxHP / 100
		if loot < 1 {
			loot = 1
		}
		if loot > 99 {
			loot = 99
		}
	}
	outcome := skillDamageOutcome{
		frame: frame, id: uint32(id), ownerID: meta.ownerID,
		loot: loot, killed: state.HP <= 0,
		hpRatio: int32(int64(state.HP) * 100 / int64(state.MaxHP)),
		model:   meta.model, wideName: meta.wideName,
		cumulative: s.run.fightRelationAddLocked(uint32(id), amount),
	}
	if outcome.killed {
		s.run.fightRelationResetLocked(uint32(id))
	}
	return outcome, nil
}

type areaSkillDamageResult struct {
	frame      []byte
	id         uint32
	ownerID    uint32
	loot       int32
	killed     bool
	hpRatio    int32
	model      string
	wideName   string
	cumulative int32
}

// relayNPCKillFrame 把致死 0x10 帧转发给同场景其他可见成员，让旁观者
// 与迟到者看到一致的死亡/重生周期；随后把击杀点排队的掉落宝箱物化
// 帧（0x0D+0x1F）锁外扇出。apply* 内部自持 run.mu，调用点在锁外，
// 两个扇出各自加锁安全。
func (s *sceneLifecycle) relayNPCKillFrame(id uint32, frame []byte) {
	if s == nil || s.run == nil || frame == nil {
		return
	}
	s.run.relayNPCPropertyFrame(id, frame, s)
	s.run.flushPendingObjectFrames()
}

func (s *sceneLifecycle) applyAreaSkillDamage(center worldcore.Transform, radius, height float32, amount int32, now time.Time) ([]areaSkillDamageResult, error) {
	return s.applyShapedSkillDamage(center, radius, height, 0, 0, amount, now)
}

func (s *sceneLifecycle) applyShapedSkillDamage(center worldcore.Transform, radius, height, sectorAngle, rectangleWidth float32, amount int32, now time.Time) ([]areaSkillDamageResult, error) {
	if s == nil || radius <= 0 || amount <= 0 {
		return nil, nil
	}
	s.run.mu.Lock()
	defer s.run.mu.Unlock()

	radiusSquared := radius * radius
	// 只有 attacknpc/bossnpc 携带 Brain，与旧循环中
	// interaction == npcInteractionCombat 的过滤等价。单次遍历直接
	// 拿到 Transform 指针，无需逐 ID Read。
	ids := make([]int, 0, s.run.combat.CombatCount())
	s.run.combat.EachCombatNPCTransform(func(id worldcore.EntityID, vitals *sim.Vitals, _ *sim.Brain, transform *worldcore.Transform) {
		if vitals.HP <= 0 {
			return
		}
		dx := transform.X - center.X
		dz := transform.Z - center.Z
		dy := transform.Y - center.Y
		if dy < 0 {
			dy = -dy
		}
		if height > 0 && dy > height {
			return
		}
		if rectangleWidth > 0 {
			facingX, facingZ := math.Cos(float64(center.Orient)), math.Sin(float64(center.Orient))
			forward := float64(dx)*facingX + float64(dz)*facingZ
			side := -float64(dx)*facingZ + float64(dz)*facingX
			if forward < 0 || forward > float64(radius) || math.Abs(side) > float64(rectangleWidth)/2 {
				return
			}
		} else if dx*dx+dz*dz > radiusSquared {
			return
		} else if sectorAngle > 0 {
			targetAngle := math.Atan2(float64(dz), float64(dx))
			delta := math.Atan2(math.Sin(targetAngle-float64(center.Orient)), math.Cos(targetAngle-float64(center.Orient)))
			if math.Abs(delta) > float64(sectorAngle)*math.Pi/360 {
				return
			}
		}
		ids = append(ids, int(id))
	})
	// EachCombatNPC 本身按 EntityID 升序回调，这里的排序是
	// 对该保证的显式防御。
	sort.Ints(ids)

	results := make([]areaSkillDamageResult, 0, len(ids))
	for _, rawID := range ids {
		id := worldcore.EntityID(rawID)
		vitals, ok := s.run.combat.Vitals(id)
		if !ok {
			continue
		}
		// 第一段筛选已保证存在；与原 map 直读的零检查语义保持一致。
		meta, _ := s.npcMeta(id)
		if !vitals.ApplyDamage(amount) {
			continue
		}
		s.run.combat.WriteVitals(id, vitals)
		state := vitals.ActorState(uint32(id), meta.ownerID)
		if brain, exists := s.run.combat.Brain(id); exists {
			brain.Threat += amount
			if state.HP <= 0 {
				brain.Threat = 0
				if npcRespawnBatchInterval() > 0 {
					// 批量刷新：不设每怪计时，refreshTick 扫描统一复活。
					brain.RespawnAt = time.Time{}
				} else {
					brain.RespawnAt = now.Add(brain.RespawnAfter)
				}
				s.run.recordNPCDeathLocked(uint32(id), meta.ownerID, brain, now)
			}
			s.run.combat.WriteBrain(id, brain)
		}
		frame, err := sceneObjectProperties(uint32(id), meta.ownerID, 0, npcCombatProperties(state))
		if err != nil {
			return nil, err
		}
		loot := int32(0)
		if state.HP <= 0 {
			loot = state.MaxHP / 100
			if loot < 1 {
				loot = 1
			}
			if loot > 99 {
				loot = 99
			}
		}
		result := areaSkillDamageResult{
			frame: frame, id: uint32(id), ownerID: meta.ownerID, loot: loot, killed: state.HP <= 0,
			hpRatio: int32(int64(state.HP) * 100 / int64(state.MaxHP)),
			model:   meta.model, wideName: meta.wideName,
			cumulative: s.run.fightRelationAddLocked(uint32(id), amount),
		}
		if result.killed {
			s.run.fightRelationResetLocked(uint32(id))
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *sceneLifecycle) selectedCombatTargetTransform() (worldcore.Transform, bool) {
	if s == nil {
		return worldcore.Transform{}, false
	}
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	id := worldcore.EntityID(s.selectedObject)
	transform, exists := s.npcTransform(id)
	vitals, hasVitals := s.run.combat.Vitals(id)
	return transform, exists && hasVitals && vitals.HP > 0
}

func (s *sceneLifecycle) applyNPCControl(ids []uint32, lifetime time.Duration, now time.Time) {
	if s == nil || lifetime <= 0 || len(ids) == 0 {
		return
	}
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	until := now.Add(lifetime)
	for _, rawID := range ids {
		id := worldcore.EntityID(rawID)
		brain, exists := s.run.combat.Brain(id)
		if !exists {
			continue
		}
		if until.After(brain.ControlledUntil) {
			brain.ControlledUntil = until
			s.run.combat.WriteBrain(id, brain)
		}
	}
}

type appliedSkillBuff struct {
	slot       uint16
	staticData uint32
	expires    time.Time
}

func stagePlayerSkillEffects(player *playerActor, definition combatSkillDefinition, now time.Time, remote string) []appliedSkillBuff {
	applied := make([]appliedSkillBuff, 0, 3)
	for _, effect := range definition.effects {
		var buff skillBuffDefinition
		switch effect.kind {
		case skillEffectPlayerBuff:
			buff = effect.buff
		case skillEffectStanceCycle:
			var ok bool
			buff, ok = player.nextSkillBuffVariant(effect.buffVariants, now)
			if !ok {
				continue
			}
		case skillEffectRedArmor:
			buff = skillBuffDefinition{
				configID: "buf_fixed_red", staticData: redSuperArmorStaticData,
				lifetime: definition.totalActionDuration(), preferredSlot: redSuperArmorBuffSlot,
			}
		default:
			continue
		}
		slot, state, expires, ok := player.activateSkillBuff(buff, now)
		if !ok {
			log.Printf("%s: no native BuffInfo slot available skill=%s buff=%s", remote, definition.id, buff.configID)
			continue
		}
		applied = append(applied, appliedSkillBuff{slot: slot, staticData: buff.staticData, expires: expires})
		log.Printf("%s: staged compiled skill Buff skill=%s buff=%s property=%d value=%q expiry=%d",
			remote, definition.id, buff.configID, bufferInfoPropertyIndex(slot), state, expires.UnixMilli())
	}
	return applied
}

func handleSkillCustom(link sceneMessageConnection, player *playerActor, world *sceneLifecycle, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomUseSkill {
		return false, nil
	}
	definition, valid := parseUseSkillCustom(custom)
	if !valid {
		log.Printf("%s: reject malformed or unlearned USE_SKILL values=%v", remote, custom.Values)
		return true, nil
	}
	if player == nil {
		log.Printf("%s: reject skill before player spawn id=%s", remote, definition.id)
		return true, nil
	}
	level, learned := player.learnedSkillLevel(definition.id)
	if !learned {
		log.Printf("%s: reject unlearned installed skill id=%s", remote, definition.id)
		return true, nil
	}
	resolved, executable := installedCombatSkillCatalog.definition(definition.id, level)
	if !executable {
		log.Printf("%s: reject learned skill without compiled level response id=%s level=%d", remote, definition.id, level)
		return true, nil
	}
	definition = resolved
	targetID := playerObjectID
	targetOwnerID := playerOwnerID
	if definition.requiresTarget {
		var ok bool
		if targetID, targetOwnerID, ok = world.selectedCombatTarget(); !ok {
			log.Printf("%s: reject skill id=%s without living selected target", remote, definition.id)
			return true, nil
		}
	}
	// 攻击距离闸（Java 八轮拍板「按技能真实射程控制释放距离」）：施法者
	// 坐标取 211 自带上报（宽容解析，缺省原点同 Java）；距离=施法者→目标。
	// 冲锋三段式：≤TSP2 立即受理；TSP2<距离≤15m 且冲锋族走暂存疾跑链
	// （CurSkillID+msg25+msg24 档2，CD/资源落地才结）；>15m 静默拒（零帧）。
	if definition.requiresTarget && world != nil {
		if targetTransform, hasTransform := world.combatNPCTransform(targetID); hasTransform {
			castAt, _ := useSkillCastTransform(custom)
			distance := math.Sqrt(float64(
				square(castAt.X-targetTransform.X) +
					square(castAt.Y-targetTransform.Y) +
					square(castAt.Z-targetTransform.Z)))
			if rushStageEnabled() && definition.stagedRushFamily() &&
				distance > float64(definition.castRange) && distance <= rushStageMaxMeters {
				return stageRushCast(link, player, definition, targetID, targetOwnerID, targetTransform, distance, remote)
			}
			if distance > float64(definition.castRange) {
				log.Printf("%s: reject skill id=%s out of range %.0fm > %.0fm",
					remote, definition.id, distance, definition.castRange)
				return true, nil
			}
		}
	}
	now := time.Now()
	target := uint64(targetID) | uint64(targetOwnerID)<<32
	if ok, reason := player.beginSkillUse(definition, target, now); !ok {
		log.Printf("%s: reject skill id=%s: %s", remote, definition.id, reason)
		return true, nil
	}
	appliedBuffs := stagePlayerSkillEffects(player, definition, now, remote)
	// §3 出招帧序：个人 CD 行 → active_skill_rec 行（动作总时长后
	// 0x12 删）→ GCD 行 → 玩家全量 → msg337 施法回显 → msg20。
	personalCDFrame, publicCDFrame, err := skillCooldownFrames(definition, now)
	if err != nil {
		return true, fmt.Errorf("encode skill cooldown %s: %w", definition.id, err)
	}
	if err := link.WriteFrame(personalCDFrame); err != nil {
		return true, err
	}
	if err := link.WriteFrame(serverRecordAddString(playerObjectID, playerOwnerID, recordActiveSkill, definition.id)); err != nil {
		return true, fmt.Errorf("write active skill row %s: %w", definition.id, err)
	}
	if err := link.WriteFrame(publicCDFrame); err != nil {
		return true, err
	}
	playerFrame, err := player.vitalUpdate()
	if err != nil {
		return true, err
	}
	if err := link.WriteFrame(playerFrame); err != nil {
		return true, err
	}
	// msg337 施法者位置/朝向回显（值取 211 请求；缺省坐标不回显）。
	if castAt, ok := useSkillCastTransform(custom); ok {
		syncFrame, syncErr := casterTransformSyncFrame(castAt.X, castAt.Y, castAt.Z, castAt.Orient)
		if syncErr != nil {
			return true, fmt.Errorf("encode caster transform sync %s: %w", definition.id, syncErr)
		}
		if err := link.WriteFrame(syncFrame); err != nil {
			return true, err
		}
	}
	if definition.actionName != "" {
		actionFrame, err := playerActionStateFrame(definition.actionName)
		if err != nil {
			return true, fmt.Errorf("encode skill State action %s (%s): %w", definition.id, definition.actionName, err)
		}
		if err := link.WriteFrame(actionFrame); err != nil {
			return true, err
		}
		if len(definition.followupActions) > 0 {
			scheduleSkillFollowupActions(link, definition)
		}
	}
	// msg20 技能动作帧（官服 out2 时间线定谳的两分规则）：锁定族
	// （SkillLock/CanRush）带真实 NPC 目标走 msg19 锁定帧（24 值大帧，
	// combat-gap 后续项——本轮不发，避免「无冲刺动画」回归）；其余
	// （非锁定技/自增益/无目标）走 msg20，目标对象按三分：自增益招
	// （IsDamage=0）=施法者自己，其余伤害技=空对象（57378/52795 实证）。
	actionTargetID, actionTargetOwner := resolveSkillActionTarget(definition, custom, world, targetID, targetOwnerID)
	lockCastPending := definition.lockSkill &&
		actionTargetID != 0 && actionTargetID != playerObjectID
	if !lockCastPending {
		actionFrame, err := nativeSkillActionFrame(definition.id, actionTargetID, actionTargetOwner)
		if err != nil {
			return true, fmt.Errorf("encode native skill action %s: %w", definition.id, err)
		}
		if err := link.WriteFrame(actionFrame); err != nil {
			return true, err
		}
	}
	log.Printf("%s: started verified State skill action skill=%s action=%s followups=%d total=%s caster=%d-%d target=%d-%d",
		remote, definition.id, definition.actionName, len(definition.followupActions), definition.totalActionDuration(),
		playerObjectID, playerOwnerID, targetID, targetOwnerID)
	damageTargetID := uint32(0)
	damageTargetCount := 0
	hitIDs := make([]uint32, 0, 8)
	var totalLoot int32
	for _, effect := range definition.effects {
		if effect.kind != skillEffectDamage || effect.amount <= 0 || world == nil {
			continue
		}
		switch definition.targetMode {
		case skillTargetSelected:
			outcome, err := world.applySelectedSkillDamage(effect.amount, now)
			if err != nil {
				return true, err
			}
			if outcome.frame != nil {
				damageTargetID, damageTargetCount = outcome.id, damageTargetCount+1
				hitIDs = append(hitIDs, outcome.id)
				if err := link.WriteFrame(outcome.frame); err != nil {
					return true, err
				}
				// 命中表现帧族（0x13/24/0x2D/18/205）。
				if err := writeSkillHitEventFrames(link, world, definition, player,
					outcome.id, outcome.ownerID, effect.amount, outcome.hpRatio, outcome.model, outcome.cumulative); err != nil {
					return true, err
				}
				// 实招命中(目标未格挡;NPC 无格挡态)→ 攻击者 +2 怒,
				// 18 字节 SP 单帧,满值不重推。
				if _, spFrame, spErr := player.addSP(ragGainCleanHit); spErr != nil {
					return true, spErr
				} else if spFrame != nil {
					if err := link.WriteFrame(spFrame); err != nil {
						return true, err
					}
				}
				if outcome.killed {
					// 致死帧对视口内其他可见成员中继,死亡表现全场景一致。
					world.relayNPCKillFrame(outcome.id, outcome.frame)
				}
			}
			if outcome.killed {
				totalLoot += outcome.loot
			}
		case skillTargetCasterCircle, skillTargetCasterSector, skillTargetCasterRectangle, skillTargetSelectedArea:
			center, ok := useSkillCastTransform(custom)
			if definition.targetMode == skillTargetSelectedArea {
				center, ok = world.selectedCombatTargetTransform()
			}
			if !ok {
				log.Printf("%s: compiled area skill id=%s has no center; no targets settled", remote, definition.id)
				continue
			}
			angle := float32(0)
			width := float32(0)
			if definition.targetMode == skillTargetCasterSector {
				angle = definition.sectorAngle
			} else if definition.targetMode == skillTargetCasterRectangle {
				width = definition.areaWidth
			}
			results, err := world.applyShapedSkillDamage(center, definition.areaRadius, definition.areaHeight, angle, width, effect.amount, now)
			if err != nil {
				return true, err
			}
			for _, result := range results {
				if damageTargetID == 0 {
					damageTargetID = result.id
				}
				damageTargetCount++
				hitIDs = append(hitIDs, result.id)
				if err := link.WriteFrame(result.frame); err != nil {
					return true, err
				}
				// 群体命中同样走完整表现帧族（§4 逐目标序列）。
				if err := writeSkillHitEventFrames(link, world, definition, player,
					result.id, result.ownerID, effect.amount, result.hpRatio, result.model, result.cumulative); err != nil {
					return true, err
				}
				if result.killed {
					totalLoot += result.loot
					// 群杀致死帧同样中继给其他可见成员。
					world.relayNPCKillFrame(result.id, result.frame)
				}
				// 群体技能同样按命中逐跳 +2(与单体一致)。
				if _, spFrame, spErr := player.addSP(ragGainCleanHit); spErr != nil {
					return true, spErr
				} else if spFrame != nil {
					if err := link.WriteFrame(spFrame); err != nil {
						return true, err
					}
				}
			}
			log.Printf("%s: settled compiled area skill id=%s mode=%d center=(%.2f,%.2f,%.2f) radius=%.1f angle=%.1f width=%.1f hits=%d damage_each=%d",
				remote, definition.id, definition.targetMode, center.X, center.Y, center.Z,
				definition.areaRadius, definition.sectorAngle, definition.areaWidth, len(results), effect.amount)
		}
	}
	for _, effect := range definition.effects {
		if effect.kind == skillEffectTargetControl {
			world.applyNPCControl(hitIDs, effect.buff.lifetime, now)
			log.Printf("%s: settled compiled target control skill=%s buff=%s targets=%d lifetime=%s",
				remote, definition.id, effect.buff.configID, len(hitIDs), effect.buff.lifetime)
		}
	}
	if totalLoot > 0 {
		player.addSilver(totalLoot)
		silverFrame, err := player.silverUpdate()
		if err != nil {
			return true, err
		}
		if err := link.WriteFrame(silverFrame); err != nil {
			return true, err
		}
		log.Printf("%s: skill defeated NPCs targets=%d silver_drop=%d", remote, damageTargetCount, totalLoot)
	}
	for _, buff := range appliedBuffs {
		schedulePlayerBuffExpiry(link, player, buff.slot, buff.staticData, buff.expires)
	}
	// 命中过敌人即战斗活动（官方 §9.1「命中入战」）：驱动 10s 脱战
	// 迟滞窗口，击杀清仇恨/怪未及锁定自己时不再闪回 Normal。
	if damageTargetCount > 0 {
		player.noteCombatActivity(now)
	}
	// 技能重置跳跃（2026-11-07 跳段限制批次，用户钉死规则）：腾空中
	// 施放技能成功 → 跳段计数重置但只授予一段（grantAirJumpReset 内部
	// 仅对腾空状态生效；着地本就全量重置）。
	player.grantAirJumpReset()
	scheduleCurrentSkillClear(link, player, definition)
	state := player.stateSnapshot()
	log.Printf("%s: accepted skill id=%s mp=%d sp=%d qg=%d damage=%d target=%d targets=%d personal_cd=%s public_cd=%s",
		remote, definition.id, state.MP, player.skillSP(), state.QingGongPoint, definition.baseDamage, damageTargetID, damageTargetCount, definition.personalCD, definition.publicCD)
	return true, nil
}

func scheduleSkillFollowupActions(conn sceneMessageConnection, definition combatSkillDefinition) {
	for _, scheduled := range definition.followupActions {
		segment := scheduled
		delay := segment.at
		if delay < 0 {
			delay = 0
		}
		time.AfterFunc(delay, func() {
			frame, err := playerActionStateFrame(segment.action)
			if err != nil {
				log.Printf("skill follow-up State encode id=%s action=%s: %v", definition.id, segment.action, err)
				return
			}
			if err := conn.WriteFrame(frame); err != nil {
				log.Printf("skill follow-up State write id=%s action=%s: %v", definition.id, segment.action, err)
				return
			}
			log.Printf("skill follow-up State sent id=%s action=%s at=%s", definition.id, segment.action, delay)
		})
	}
}

func scheduleCurrentSkillClear(conn sceneMessageConnection, player *playerActor, definition combatSkillDefinition) {
	delay := definition.totalActionDuration()
	if delay <= 0 {
		delay = time.Millisecond
	}
	time.AfterFunc(delay, func() {
		if !player.clearCurrentSkill(definition.id) {
			return
		}
		// §3 行 9：动作总时长后 0x12 删 active_skill_rec 行 0 +
		// State(idx19)="stand"（183 是我方已验证的动作段复位通道，并行保留）。
		if err := conn.WriteFrame(serverRecordDelRow(playerObjectID, playerOwnerID, recordActiveSkill, 0)); err != nil {
			log.Printf("skill active row delete id=%s: %v", definition.id, err)
		}
		update, err := player.currentSkillClearUpdate()
		if err != nil {
			log.Printf("skill state clear encode id=%s: %v", definition.id, err)
			return
		}
		if err := conn.WriteFrame(update); err != nil {
			log.Printf("skill state clear write id=%s: %v", definition.id, err)
			return
		}
		stand, err := playerActionStateFrame("stand")
		if err != nil {
			log.Printf("skill stand restore encode id=%s: %v", definition.id, err)
			return
		}
		if err := conn.WriteFrame(stand); err != nil {
			log.Printf("skill stand restore write id=%s: %v", definition.id, err)
			return
		}
		if standState, err := playerStandStateFrame(); err == nil {
			if err := conn.WriteFrame(standState); err != nil {
				log.Printf("skill stand state write id=%s: %v", definition.id, err)
			}
		}
		log.Printf("skill state cleared and stand restored id=%s total_action=%s", definition.id, delay)
	})
}
