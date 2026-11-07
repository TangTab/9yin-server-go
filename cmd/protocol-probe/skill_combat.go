package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"sync/atomic"
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
	// damageWaves 是 damage_calculate 正 r 段数（伤害倍率分母：
	// ratio = baseDamage/100/waves，Java CombatSkill.damageRatio 同式）。
	// hitFrameMs 是 zhaoshi trigger_custom_N（段号;trigger_effect;帧号;
	// 效果名）编译出的多段判定时刻（30fps 换算毫秒），空表=单段即时
	// 结算。levelupRelevance（wuji_* 前缀=武技族，出招链带 169 段）与
	// packCategory（2=怒招）驱动武技连击段下发。
	damageWaves      int32
	hitFrameMs       []time.Duration
	levelupRelevance string
	packCategory     int32
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

// scriptLabel 是 msg24 飘字第 7 值（Java CombatSkill.scriptLabel）：锁定族
// 一律 "SkillLock"，其余原样 script。抓包 52905：myjf01 系 'SkillLock'、
// 其余 'SkillNormal'——此前 Go 恒发 "SkillNormal" 属本地占位。
func (definition combatSkillDefinition) scriptLabel() string {
	if definition.lockSkill {
		return "SkillLock"
	}
	return definition.script
}

// damageRatio 是公式倍率（formula.lua 的 ratio；<=0 由公式按 1.0 处理）。
func (definition combatSkillDefinition) damageRatio() float64 {
	if definition.damageWaves <= 0 {
		return 0
	}
	return float64(definition.baseDamage) / 100 / float64(definition.damageWaves)
}

// wujiRelevance 报告武技族（LevelupRelevance=wuji_* 前缀）：出招链带
// 169 buff 段（首跳摘 buf_<技能>_01）。
func (definition combatSkillDefinition) wujiRelevance() bool {
	return strings.HasPrefix(strings.ToLower(definition.levelupRelevance), "wuji_")
}

// wujiCombo 报告武技连击段（wuji_* 且 PackCategory=2 怒招）：每跳对每个
// 受害 NPC 追加目标 debuff 169（带到期）+ msg24 c1=71 连击段飘字。
func (definition combatSkillDefinition) wujiCombo() bool {
	return definition.wujiRelevance() && definition.packCategory == 2
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
	hitFrame, err := skillHitDamageFrame(victimID, victimOwner, damage, definition.id, definition.scriptLabel(), victimModel, player.snapshotName())
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
	// 外壳 0x1E、尾两值 0/0.0（Java FightMessages.skillAction + 抓包
	// 52905 #3005 逐字节；42 份抓包 msg20 从未走 0x27，尾值恒 0——
	// 此前 0x27+1/1 是本地占位）。
	return serverCustomIntMessage(
		serverNativeSkillActionMessage,
		customObject(playerObjectID, playerOwnerID),
		customString(skillID),
		customObject(targetID, targetOwnerID),
		customInt(0),
		customFloat32(0),
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
	// 公共 GCD 只拦套路切换（Java PlayerActor.beginSkillUse:1061-1064
	// 同构、bug.txt:8「同套路不在触发公共7秒冷却，不同套路会看到7秒CD
	// 转圈回复」）：同套路连招与出招者自身永不拦——个人 CD<7s 的技能
	// 不再被压到 7s。此前 per-team 硬闸（skillTeamCooldowns[CoolDownTeam]）
	// 把同套路五招（共享 CoolDownTeam）连同施法者全部锁 7s，是本地自造
	// 模型，已废弃。
	if p.publicCDUntil.After(now) && definition.taoLu != p.publicCDTaolu {
		return false, fmt.Sprintf("public cooldown (taolu switch from=%s) remaining=%s",
			p.publicCDTaolu, p.publicCDUntil.Sub(now).Round(time.Millisecond))
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
	// 每次受理都刷新 GCD 窗口与发起套路（62433 抓包：每招恒发个人+GCD
	// 双行；客户端按 team 列过滤，同套路技能不显示 7s——服务端同样只拦
	// 跨套路）。连续同套路连招会把切换锁窗口后推，Java 同款。
	p.publicCDUntil = now.Add(definition.publicCD)
	p.publicCDTaolu = definition.taoLu
	p.currentSkillID = definition.id
	p.currentSkillEffectID = definition.id
	p.currentSkillLevel = definition.level
	p.currentSkillTarget = target
	// 出招代次自增（Java beginSkillUse 同构）：收尾定时器按 (id, epoch)
	// 配对，旧招定时器失配空转。
	p.currentSkillEpoch++
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

// applySelectedSkillDamage 是 applySkillDamageToNPC 的选中目标快捷方式
// （历史测试路径保留：伤害直落选中战斗 NPC）。
func (s *sceneLifecycle) applySelectedSkillDamage(amount int32, now time.Time) (skillDamageOutcome, error) {
	id, ownerID, ok := s.selectedCombatTarget()
	if !ok {
		return skillDamageOutcome{}, nil
	}
	return s.applySkillDamageToNPC(id, ownerID, amount, now)
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

// skillDamageTargetRef 是出招时刻解析出的伤害目标引用（Java
// FightState.HitTarget 的 Go 对应物；多段路径按此快照逐段结算，段间
// 新进范围的 NPC 不追补——Java 语义）。
type skillDamageTargetRef struct {
	id      uint32
	ownerID uint32
}

// skillSettleHit 是单次结算批里的一条命中（Java Hit + 结算产物合体）。
type skillSettleHit struct {
	ref     skillDamageTargetRef
	damage  int32
	outcome skillDamageOutcome
}

// skillRowCanUseFrame 是 View40 技能行可用性行帧（Java FightMessages.
// skillRowCanUse、62433 抓包 2159 帧实证）：0x10 | isView=1 | u32 40 |
// u32 行号 | u16 1 | u16 CanUse(2236) | u8 0/1。出招置 0、个人 CD 尽时
// 置 1——不置则真机技能图标恒灰；官方值恒 1 不按资源门控，资源灰由
// 客户端按 SP+AConsumeSP 自算。
func skillRowCanUseFrame(row uint16, canUse bool) ([]byte, error) {
	value := byte(0)
	if canUse {
		value = 1
	}
	return sceneObjectProperties(uint32(viewportSkill), uint32(row), 1, []clientdata.IndexedProperty{
		{Index: propSkillCanUse, Name: "CanUse", Value: clientdata.ByteValue(value)},
	})
}

// resolveSkillDamageTargets 出招时刻的目标解析（Java resolveTargets 同
// 构）：无伤害效果返回空表；单体=选中目标；以己形状=211 上报坐标为
// 中心；以目标为中心=选中 NPC 位置为圆心。只解析引用不落伤害。
func (s *sceneLifecycle) resolveSkillDamageTargets(definition combatSkillDefinition, custom clientCustomMessage, selectedID, selectedOwner uint32) []skillDamageTargetRef {
	hasDamage := false
	for _, effect := range definition.effects {
		if effect.kind == skillEffectDamage && effect.amount > 0 {
			hasDamage = true
			break
		}
	}
	if !hasDamage || s == nil {
		return nil
	}
	switch definition.targetMode {
	case skillTargetSelected:
		if selectedID != 0 && selectedID != playerObjectID {
			return []skillDamageTargetRef{{id: selectedID, ownerID: selectedOwner}}
		}
		return nil
	case skillTargetCasterCircle, skillTargetCasterSector, skillTargetCasterRectangle:
		center, ok := useSkillCastTransform(custom)
		if !ok {
			return nil
		}
		return s.queryShapedSkillTargets(center, definition.areaRadius, definition.areaHeight,
			definition.sectorAngleForShape(), definition.areaWidthForShape())
	case skillTargetSelectedArea:
		center, ok := s.selectedCombatTargetTransform()
		if !ok {
			return nil
		}
		return s.queryShapedSkillTargets(center, definition.areaRadius, definition.areaHeight, 0, 0)
	default:
		return nil
	}
}

// sectorAngleForShape / areaWidthForShape：形状查询参数收口（扇形取
// sectorAngle、矩形取 areaWidth，其余 0——与 applyShapedSkillDamage
// 的旧分支语义一致）。
func (definition combatSkillDefinition) sectorAngleForShape() float32 {
	if definition.targetMode == skillTargetCasterSector {
		return definition.sectorAngle
	}
	return 0
}

func (definition combatSkillDefinition) areaWidthForShape() float32 {
	if definition.targetMode == skillTargetCasterRectangle {
		return definition.areaWidth
	}
	return 0
}

// queryShapedSkillTargets 是 applyShapedSkillDamage 的只查询形态：
// 返回形状命中的存活战斗 NPC 引用（EntityID 升序），不落伤害。
func (s *sceneLifecycle) queryShapedSkillTargets(center worldcore.Transform, radius, height, sectorAngle, rectangleWidth float32) []skillDamageTargetRef {
	if s == nil || radius <= 0 {
		return nil
	}
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	radiusSquared := radius * radius
	refs := make([]skillDamageTargetRef, 0, 4)
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
		if meta, exists := s.npcMeta(id); exists {
			refs = append(refs, skillDamageTargetRef{id: uint32(id), ownerID: meta.ownerID})
		}
	})
	sort.Slice(refs, func(i, j int) bool { return refs[i].id < refs[j].id })
	return refs
}

// applySkillDamageToNPC 对指定战斗 NPC 落一击伤害（Java FightState.
// applyDamage + applySelectedSkillDamage 合并的参数化形态）：目标失效/
// 已死/未命中 ApplyDamage 拒绝时返回零值 outcome（frame=nil，发帧阶段
// 整段跳过）。累计威胁进 0x13，死亡清仇恨排重生。
func (s *sceneLifecycle) applySkillDamageToNPC(id, ownerID uint32, amount int32, now time.Time) (skillDamageOutcome, error) {
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	entityID := worldcore.EntityID(id)
	vitals, hasVitals := s.run.combat.Vitals(entityID)
	meta, exists := s.npcMeta(entityID)
	if !hasVitals || !exists || meta.ownerID != ownerID || meta.interaction != npcInteractionCombat || vitals.HP <= 0 {
		return skillDamageOutcome{}, nil
	}
	if !vitals.ApplyDamage(amount) {
		return skillDamageOutcome{}, nil
	}
	s.run.combat.WriteVitals(entityID, vitals)
	state := vitals.ActorState(id, ownerID)
	if brain, isCombat := s.run.combat.Brain(entityID); isCombat {
		brain.Threat += amount
		if state.HP <= 0 {
			brain.Threat = 0
			if npcRespawnBatchInterval() > 0 {
				// 批量刷新：不设每怪计时，refreshTick 扫描统一复活。
				brain.RespawnAt = time.Time{}
			} else {
				brain.RespawnAt = now.Add(brain.RespawnAfter)
			}
			s.run.recordNPCDeathLocked(id, ownerID, brain, now)
		}
		s.run.combat.WriteBrain(entityID, brain)
	}
	frame, err := sceneObjectProperties(id, ownerID, 0, npcCombatProperties(state))
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
		frame: frame, id: id, ownerID: ownerID,
		loot: loot, killed: state.HP <= 0,
		hpRatio:    int32(int64(state.HP) * 100 / int64(state.MaxHP)),
		model:      meta.model,
		wideName:   meta.wideName,
		cumulative: s.run.fightRelationAddLocked(id, amount),
	}
	if outcome.killed {
		s.run.fightRelationResetLocked(id)
	}
	return outcome, nil
}

// skillHitTimes 多段调度时刻（Java hitTimes 同构）：trigger_effect 帧位
// 毫秒升序、去重，首段至少 1ms 后落拍。
func skillHitTimes(definition combatSkillDefinition) []time.Duration {
	times := make([]time.Duration, 0, len(definition.hitFrameMs))
	var previous time.Duration = -1
	for _, frameMs := range definition.hitFrameMs {
		at := frameMs
		if at < time.Millisecond {
			at = time.Millisecond
		}
		if at == previous {
			continue
		}
		times = append(times, at)
		previous = at
	}
	return times
}

// writeSkillSettleFrames 写一批命中的全套帧（Java emitPerTargetSequence
// 同序）：逐目标 [0x10 NPC 面板 → 0x13 → msg24(17) → 0x2D → msg18 →
// msg205]，致死帧中继；随后按命中数一次性 +2/跳 回怒（官方按打击次数
// 不按伤害，多段招每段每目标各算）。掉银与击杀中继随批结算。
func writeSkillSettleFrames(link sceneMessageConnection, world *sceneLifecycle, definition combatSkillDefinition, player *playerActor, hits []skillSettleHit, remote string) (totalLoot int32, err error) {
	for _, hit := range hits {
		if err := link.WriteFrame(hit.outcome.frame); err != nil {
			return totalLoot, err
		}
		if err := writeSkillHitEventFrames(link, world, definition, player,
			hit.ref.id, hit.ref.ownerID, hit.damage, hit.outcome.hpRatio, hit.outcome.model, hit.outcome.cumulative); err != nil {
			return totalLoot, err
		}
		if hit.outcome.killed {
			totalLoot += hit.outcome.loot
			world.relayNPCKillFrame(hit.ref.id, hit.outcome.frame)
		}
	}
	if len(hits) > 0 {
		if _, spFrame, spErr := player.addSP(int32(len(hits)) * ragGainCleanHit); spErr != nil {
			return totalLoot, spErr
		} else if spFrame != nil {
			if err := link.WriteFrame(spFrame); err != nil {
				return totalLoot, err
			}
		}
	}
	return totalLoot, nil
}

// writeWujiSequenceFrames 武技族出招附加帧（Java emitWujiSequence、
// 参考服 55866 myjf03/myjf08 实证）：wuji_* 族首跳摘玩家自身
// buf_<技能>_01（169 v0=0）；wuji_* + 怒招（PackCategory=2）每跳对每个
// 受害 NPC 发目标 debuff 169（v0=1 带到期）+ 一条 msg24 c1=71 连击段
// （段名 ”/“_2”/“_3”/“_4”上限四段，受害/第二目标轮转）。
func writeWujiSequenceFrames(link sceneMessageConnection, definition combatSkillDefinition, player *playerActor, hits []skillSettleHit, tickIndex int) error {
	if !definition.wujiRelevance() || len(hits) == 0 {
		return nil
	}
	playerName := player.snapshotName()
	if tickIndex == 0 {
		frame, err := skillBuffFrame(0, playerObjectID, playerOwnerID, playerObjectID, playerOwnerID,
			"buf_"+definition.id+"_01", playerName)
		if err != nil {
			return err
		}
		if err := link.WriteFrame(frame); err != nil {
			return err
		}
	}
	if !definition.wujiCombo() {
		return nil
	}
	// 武技连击 NPC debuff 租约 ~2s 量级（参考服 55866 逐跳滚动）。
	const wujiDebuffLease = 2 * time.Second
	now := time.Now()
	for _, hit := range hits {
		frame, err := targetDebuffTimedFrame("buf_"+definition.id,
			hit.ref.id, hit.ref.ownerID, playerObjectID, playerOwnerID, playerName,
			now.Add(wujiDebuffLease).UnixMilli())
		if err != nil {
			return err
		}
		if err := link.WriteFrame(frame); err != nil {
			return err
		}
	}
	if tickIndex > 3 {
		return nil
	}
	stage := ""
	if tickIndex > 0 {
		stage = fmt.Sprintf("%s_%d", definition.levelupRelevance, tickIndex+1)
	}
	first := hits[0]
	second := hits[0]
	if len(hits) > 1 {
		second = hits[1]
	}
	frame, err := wujiComboFloatFrame(first.ref.id, first.ref.ownerID,
		second.ref.id, second.ref.ownerID, playerObjectID, playerOwnerID, stage)
	if err != nil {
		return err
	}
	return link.WriteFrame(frame)
}

// awardSettleLoot 把击杀掉银入账并推货币帧（Go 本地特色，Java 无此
// 路径——保留既有经济特性）。
func awardSettleLoot(link sceneMessageConnection, player *playerActor, totalLoot int32, targets, remote string) {
	if totalLoot <= 0 {
		return
	}
	player.addSilver(totalLoot)
	silverFrame, err := player.silverUpdate()
	if err == nil {
		if err := link.WriteFrame(silverFrame); err != nil {
			log.Printf("%s: write silver update: %v", remote, err)
		}
	}
	log.Printf("%s: skill defeated NPCs %s silver_drop=%d", remote, targets, totalLoot)
}

// scheduleSkillMultiHits 多段结算（Java scheduleMultiHits、52795 抓包
// 实证形态）：出招时掷一次伤害值（同招各段各目标同值），按
// trigger_effect 帧位逐段调度——每段对仍存活目标发全套命中帧；首段
// 命中进战斗+目标控制，后续段刷新战斗活跃时刻；每段每目标 +2 回怒；
// 末段后按整招累计伤害喂经脉真气。
func scheduleSkillMultiHits(link sceneMessageConnection, world *sceneLifecycle, definition combatSkillDefinition,
	player *playerActor, targets []skillDamageTargetRef, roll combatRoll, remote string) {
	if len(targets) == 0 || roll.damage <= 0 {
		return
	}
	times := skillHitTimes(definition)
	if len(times) == 0 {
		return
	}
	lastAt := times[len(times)-1]
	// 段间可能并发到达（帧位相近时）：跨段共享态用原子量（firstTick 防重
	// 复入战、landedTotal 汇总喂真气）。
	var landedTotal atomic.Int64
	var firstTick atomic.Bool
	for tickIndex, at := range times {
		tick := tickIndex
		lastTick := at == lastAt
		time.AfterFunc(at, func() {
			now := time.Now()
			hits := make([]skillSettleHit, 0, len(targets))
			killed := false
			for _, ref := range targets {
				outcome, err := world.applySkillDamageToNPC(ref.id, ref.ownerID, roll.damage, now)
				if err != nil {
					log.Printf("%s: multi-hit stage %d apply id=%d: %v", remote, tick, ref.id, err)
					continue
				}
				if outcome.frame == nil {
					continue // 前段已击杀/目标失效：本段跳过（不发帧）
				}
				hits = append(hits, skillSettleHit{ref: ref, damage: roll.damage, outcome: outcome})
				if outcome.killed {
					killed = true
				}
			}
			if len(hits) == 0 {
				return
			}
			// 命中/每段都刷新战斗活跃时刻（Java enterCombat/touchCombat），
			// 防段间 10s 闲退。
			player.noteCombatActivity(now)
			if firstTick.CompareAndSwap(true, false) {
				for _, effect := range definition.effects {
					if effect.kind == skillEffectTargetControl && effect.buff.lifetime > 0 {
						ids := make([]uint32, 0, len(hits))
						for _, hit := range hits {
							ids = append(ids, hit.ref.id)
						}
						world.applyNPCControl(ids, effect.buff.lifetime, now)
					}
				}
			}
			totalLoot, err := writeSkillSettleFrames(link, world, definition, player, hits, remote)
			if err != nil {
				log.Printf("%s: multi-hit stage %d frames: %v", remote, tick, err)
			}
			awardSettleLoot(link, player, totalLoot, fmt.Sprintf("stage=%d targets=%d", tick, len(hits)), remote)
			if err := writeWujiSequenceFrames(link, definition, player, hits, tick); err != nil {
				log.Printf("%s: multi-hit stage %d wuji frames: %v", remote, tick, err)
			}
			if killed && definition.wujiCombo() {
				// 击杀红名标记（参考服 55866：myjf08 击杀后 [1, buf_fixed_red]）。
				if frame, err := skillBuffFrame(1, playerObjectID, playerOwnerID, playerObjectID, playerOwnerID,
					"buf_fixed_red", player.snapshotName()); err == nil {
					if err := link.WriteFrame(frame); err != nil {
						log.Printf("%s: write kill red marker: %v", remote, err)
					}
				}
			}
			for _, hit := range hits {
				landedTotal.Add(int64(hit.damage))
			}
			if lastTick && landedTotal.Load() > 0 {
				for _, frame := range player.jingMaiCombatFeed(int32(landedTotal.Load())) {
					if err := link.WriteFrame(frame); err != nil {
						log.Printf("%s: write combat zhenqi frame: %v", remote, err)
					}
				}
			}
			log.Printf("%s: multi-hit stage %d/%d skill=%s hits=%d damage_each=%d", remote, tick+1, len(times), definition.id, len(hits), roll.damage)
		})
	}
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
	// 顶掉补删（Java §D8 对拍）：GCD 不硬闸，连招会把上一招的收尾定时器
	// 顶掉；新招受理时上一招仍挂着的，立即补发它的 0x12+清除帧——
	// active_skill_rec 的 add/del 恒 1:1（参考服 8/8、52905 14/14）。
	superseded := player.currentSkillSnapshot()
	if ok, reason := player.beginSkillUse(definition, target, now); !ok {
		log.Printf("%s: reject skill id=%s: %s", remote, definition.id, reason)
		return true, nil
	}
	castEpoch := player.currentSkillEpochSnapshot()
	if superseded != "" {
		if err := link.WriteFrame(serverRecordDelRow(playerObjectID, playerOwnerID, recordActiveSkill, 0)); err != nil {
			log.Printf("%s: close superseded skill row %s: %v", remote, superseded, err)
		}
		if clear, err := player.currentSkillClearUpdate(); err == nil {
			if err := link.WriteFrame(clear); err != nil {
				log.Printf("%s: close superseded skill clear %s: %v", remote, superseded, err)
			}
		}
		log.Printf("%s: close superseded skill id=%s early (delete row + clear frame)", remote, superseded)
	}
	appliedBuffs := stagePlayerSkillEffects(player, definition, now, remote)
	// §3 出招帧序（Java handleValues 逐位同序）：个人 CD 行 →
	// active_skill_rec 行 → GCD 行 → View40 行 CanUse=0 → 玩家全量 →
	// msg337 施法回显 → msg20。
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
	// CanUse 置 0（View40 行；不置则真机技能图标恒灰）；个人 CD 尽时
	// 置 1（官方 SP 变化不触发行帧，图标点亮由客户端按 SP 自算）。
	if row := player.skillViewRow(definition.id); row > 0 {
		if frame, canUseErr := skillRowCanUseFrame(row, false); canUseErr == nil {
			if err := link.WriteFrame(frame); err != nil {
				return true, err
			}
		}
		restore := definition.personalCD
		if restore < time.Millisecond {
			restore = time.Millisecond
		}
		time.AfterFunc(restore, func() {
			if frame, canUseErr := skillRowCanUseFrame(row, true); canUseErr == nil {
				if err := link.WriteFrame(frame); err != nil {
					log.Printf("skill CanUse restore id=%s: %v", definition.id, err)
				}
			}
		})
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
	// msg20 技能动作帧：目标对象三分（resolveSkillActionTarget——锁定族
	// =211 瞄准/选中目标、自增益=自己、其余=空对象）。锁定族官方走
	// msg19 锁定帧（c61 冲刺+0x1F+24 值大帧），但 42 份抓包 0 次出现且
	// 此前实现有「无冲刺动画」回归史——继续 msg20 路线（分歧记录在
	// docs/combat-rewrite-hardcoded-audit.md）。
	actionTargetID, actionTargetOwner := resolveSkillActionTarget(definition, custom, world, targetID, targetOwnerID)
	actionFrame, err := nativeSkillActionFrame(definition.id, actionTargetID, actionTargetOwner)
	if err != nil {
		return true, fmt.Errorf("encode native skill action %s: %w", definition.id, err)
	}
	if err := link.WriteFrame(actionFrame); err != nil {
		return true, err
	}
	log.Printf("%s: started skill action skill=%s script=%s total=%s caster=%d-%d target=%d-%d waves=%d ticks=%d",
		remote, definition.id, definition.scriptLabel(), definition.totalActionDuration(),
		playerObjectID, playerOwnerID, targetID, targetOwnerID, definition.damageWaves, len(definition.hitFrameMs))
	// 出招时刻解析目标快照（多段按此逐段结算，段间不追补新目标）。
	targets := world.resolveSkillDamageTargets(definition, custom, targetID, targetOwnerID)
	if len(definition.hitFrameMs) > 0 {
		// 多段技能：出招掷一次（同值复用各段），按 trigger_effect 帧位
		// 逐段调度（52795 抓包：每段全套命中帧、死亡目标跳过）。
		attackMin, attackMax := player.meleeDamageRange()
		roll := rollSkillDamage(attackMin, attackMax, definition.damageRatio())
		scheduleSkillMultiHits(link, world, definition, player, targets, roll, remote)
		log.Printf("%s: scheduled multi-hit skill id=%s ticks=%d targets=%d roll=%d miss=%t",
			remote, definition.id, len(definition.hitFrameMs), len(targets), roll.damage, roll.damage == 0)
	} else if len(targets) > 0 {
		// 单段技能：逐目标掷值落血（Java settleTargets——每目标独立
		// 掷骰），未命中（damage=0）整段不发帧。
		attackMin, attackMax := player.meleeDamageRange()
		hits := make([]skillSettleHit, 0, len(targets))
		for _, ref := range targets {
			roll := rollSkillDamage(attackMin, attackMax, definition.damageRatio())
			if roll.damage <= 0 {
				continue
			}
			outcome, err := world.applySkillDamageToNPC(ref.id, ref.ownerID, roll.damage, now)
			if err != nil {
				return true, err
			}
			if outcome.frame == nil {
				continue
			}
			hits = append(hits, skillSettleHit{ref: ref, damage: roll.damage, outcome: outcome})
		}
		if len(hits) > 0 {
			// 命中过敌人即战斗活动（官方 §9.1「命中入战」）：驱动 10s
			// 脱战迟滞窗口，击杀清仇恨/怪未及锁定自己时不再闪回 Normal。
			player.noteCombatActivity(now)
			totalLoot, err := writeSkillSettleFrames(link, world, definition, player, hits, remote)
			if err != nil {
				return true, err
			}
			awardSettleLoot(link, player, totalLoot, fmt.Sprintf("targets=%d", len(hits)), remote)
			if err := writeWujiSequenceFrames(link, definition, player, hits, 0); err != nil {
				return true, err
			}
			hitIDs := make([]uint32, 0, len(hits))
			var damageSum int32
			for _, hit := range hits {
				hitIDs = append(hitIDs, hit.ref.id)
				damageSum += hit.damage
			}
			for _, effect := range definition.effects {
				if effect.kind == skillEffectTargetControl {
					world.applyNPCControl(hitIDs, effect.buff.lifetime, now)
					log.Printf("%s: settled compiled target control skill=%s buff=%s targets=%d lifetime=%s",
						remote, definition.id, effect.buff.configID, len(hitIDs), effect.buff.lifetime)
				}
			}
			// 战斗真气（Java feedMeridianZhenQi：伤害合计×0.1 折真气）。
			for _, frame := range player.jingMaiCombatFeed(damageSum) {
				if err := link.WriteFrame(frame); err != nil {
					return true, err
				}
			}
			log.Printf("%s: settled skill id=%s targets=%d damage_ratio=%.3f",
				remote, definition.id, len(hits), definition.damageRatio())
		}
	}
	for _, buff := range appliedBuffs {
		schedulePlayerBuffExpiry(link, player, buff.slot, buff.staticData, buff.expires)
	}
	// 技能重置跳跃（2026-11-07 跳段限制批次，用户钉死规则）：腾空中
	// 施放技能成功 → 跳段计数重置但只授予一段（grantAirJumpReset 内部
	// 仅对腾空状态生效；着地本就全量重置）。
	player.grantAirJumpReset()
	scheduleCurrentSkillClear(link, player, definition, castEpoch)
	state := player.stateSnapshot()
	log.Printf("%s: accepted skill id=%s mp=%d sp=%d qg=%d base_damage=%d waves=%d target=%d targets=%d personal_cd=%s public_cd=%s",
		remote, definition.id, state.MP, player.skillSP(), state.QingGongPoint, definition.baseDamage, definition.damageWaves, targetID, len(targets), definition.personalCD, definition.publicCD)
	return true, nil
}

// scheduleCurrentSkillClear 动作总时长后的收尾（Java scheduleEndOfAction
// + 52905 抓包 14/14 收招簇）：0x12 删 active_skill_rec 行 0 → 六属性
// 技能态清除帧 → State(idx19)="stand" 复位。删行按 (id, epoch) 配对——
// 被新招顶掉的旧招在此空转；State='stand' 只走官方模板通道 idx19 一条
// （抓包收尾簇恰一条 24B State 帧）。
func scheduleCurrentSkillClear(conn sceneMessageConnection, player *playerActor, definition combatSkillDefinition, castEpoch uint64) {
	delay := definition.totalActionDuration()
	if delay <= 0 {
		delay = time.Millisecond
	}
	time.AfterFunc(delay, func() {
		if !player.clearCurrentSkill(definition.id, castEpoch) {
			return
		}
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
		if standState, err := playerStandStateFrame(); err == nil {
			if err := conn.WriteFrame(standState); err != nil {
				log.Printf("skill stand state write id=%s: %v", definition.id, err)
			}
		}
		log.Printf("skill state cleared id=%s total_action=%s", definition.id, delay)
	})
}
