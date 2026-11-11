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
	"github.com/local/9yin-go-server/internal/combatcalc"
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
	// damageRows 是 r 行原始编译产物（2026-11-08 数值重写批次）：combatcalc
	// 每击公式路径按 formulaID 取 formula.ini 公式、按位喂 params（P1 基值/
	// P2 波动/分摊系数等），legacy ratio 路径保留为回退。hitCount 是 varprop
	// HitCount 列（hurtCount() 的第二优先级计数来源）。
	// hitFrameMs 是 zhaoshi trigger_custom_N 编译出的多段判定时刻
	// （30fps 换算毫秒）：trigger_hitbreak 帧位优先（2026-11-11 官服
	// pcapng 定版，段内相对帧+前序段全时长累加），无 hitbreak 回落
	// trigger_effect _hurt/核心前缀帧（段相对同口径），空表=单段即时
	// 结算。levelupRelevance（wuji_* 前缀=武技族，出招链带 169 段）与
	// packCategory（2=怒招）驱动武技连击段下发。
	damageWaves      int32
	damageRows       []skillDamageRow
	hitCount         int32
	hitFrameMs       []time.Duration
	// channel 非空 = 怒招引导通道型（skill_channel.go，挑战.pcapng
	// 2026-11-12 寒心恨雪定版）：跳数/节奏由服务端波次引擎驱动而非
	// hitFrameMs（通道型 zhaoshi 单段仅 1 hitbreak），SP 逐波扣无预扣。
	channel *channelSkillSpec
	// noLimit = skill_static NoLimit（施法中不锁行动）：受理簇须在 m20
	// 前发 CurSkillCanNoLimit=1（挑战.pcapng tjj07 受理簇实证）——客户端
	// m20 处理链据此放行移动，缺它=太极剑舞等可移动技被定死原地。
	noLimit bool
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
	// rushRange=skill_static.RushRange（冲锋发起上限，缺列回退
	// max(castRange,15m)——TargetShape 半径不是冲锋技的可攻击距离）；
	// canRush=skill_static.CanRush；lockSkill=script=="skilllock"||canRush||
	// (TargetType==4&&DamageType!=0)（msg20 动作三分用）；
	// isDamageSkill=skill_static.IsDamage；bind/groundRushAction=
	// skill_lock_data[LockType] 的 BindDistance/LockGRushAction（缺行回退）。
	castRange        float32
	rushRange        float32
	canRush          bool
	lockSkill        bool
	isDamageSkill    bool
	bindDistance     float32
	groundRushAction string
}

// stagedRushFamily 报告该技能是否走冲锋暂存链。
// 2026-11-08 50514 会话回归：本服 skill_static.ini 的 Script 列普遍为
// None（hyjf07 [7648] CanRush=1、Script=None 实测），旧条件
// `canRush && script=="skilllock"` 恒 false → 全部冲锋 211 被静默拒
// （零帧，客户端无动画不动）。放宽为 canRush 单条件——lockSkill 判定
// 本来就有 canRush 兜底，冲锋族以 CanRush 列为准。
func (definition combatSkillDefinition) stagedRushFamily() bool {
	return definition.canRush
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

// installedCombatCalc 是每击公式引擎（main 按 -combat-scripts 装配；nil 或
// 未 Ready 时 rollSkillHit 全走 legacy 掷骰——测试环境不装配即维持旧行为）。
var installedCombatCalc *combatcalc.Engine

// combatCalcSnapshot 返回攻方属性/生存值快照（combatcalc ctx.props/vitals）。
// MeleePowerAdd 等加成位按五维衍生系数取对应维（attr-derived 官方系数）。
func (p *playerActor) combatCalcSnapshot() (props, vitals map[string]int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	a := p.progress.attributes
	v := p.vitals
	props = map[string]int64{
		"Str": int64(a.str), "Dex": int64(a.dex), "Ing": int64(a.ing),
		"Spi": int64(a.spi), "Sta": int64(a.sta),
		"MeleePower": int64(a.meleePower), "MeleePowerAdd": int64(a.str),
		"ShotPower": int64(a.shotPower), "ShotPowerAdd": int64(a.dex),
		"MagicPower": int64(a.magicPower), "MagicPowerAdd": int64(a.ing),
		"PhyVa": int64(a.phyVa), "MagicVa": int64(a.magicVa),
		"PhyHit": int64(a.phyHit), "MagicHit": int64(a.magicHit), "Dodge": int64(a.dodge),
		"MinMeleeDamage": int64(a.minMeleeDamage), "MaxMeleeDamage": int64(a.maxMeleeDamage),
	}
	vitals = map[string]int64{
		"HP": int64(v.HP), "MaxHP": int64(v.MaxHP),
		"MP": int64(v.MP), "MaxMP": int64(v.MaxMP),
	}
	return props, vitals
}

// rollSkillHit 单击伤害结算（2026-11-08 数值重写）：优先 combatcalc 每击
// 公式路径——damage_calculate r 行 × formula.ini 公式（P1 基值+P2 波动+
// 威力×系数+分摊），每击×每目标独立掷值；暴击值/命中值按伤害系取
// PhyVa/MagicVa、PhyHit/MagicHit（§十六②：基础暴率=5%+10%×(暴击值−封劲)/
// (暴击值+1)，默认 16 点暴击值 vs 零封劲 ≈14.4%，正对 §七 PVE 实测的
// 14–15%）。守方取自 NPC 实力包推导（npcCombatStats：staticprop Base×
// Level，按伤害系选防；封劲暂 0 见其注释）；无 stats/非本场景目标按零值
// （与参考服"守方侧缺省"口径一致）。引擎未装载/公式不支持/运行期回退时
// 走 legacy 面板×倍率掷骰（rollSkillDamage 原样保留）。
func rollSkillHit(player *playerActor, definition combatSkillDefinition, ref skillDamageTargetRef, world *sceneLifecycle, targetCount int) combatRoll {
	if engine := installedCombatCalc; engine != nil && engine.Ready() {
		if row, ok := definition.primaryDamageRow(); ok && engine.Supports(row.formulaID) {
			props, vitals := player.combatCalcSnapshot()
			critValue, accuracy := props["PhyVa"], props["PhyHit"]
			if !strings.EqualFold(row.damageType, "PhyDamage") {
				critValue, accuracy = props["MagicVa"], props["MagicHit"]
			}
			input := &combatcalc.HitInput{
				FormulaID:      row.formulaID,
				Params:         row.params,
				HurtCount:      definition.hurtCount(),
				CurTargetCount: targetCount,
				Props:          props,
				Vitals:         vitals,
				CritValue:      critValue,
				Accuracy:       accuracy,
				Defender:       defenderStatsOf(world, ref, row.damageType),
				MissRoll:       randFloat64(),
				CritRoll:       randFloat64(),
				ExtraRolls: []float64{
					randFloat64(), randFloat64(), randFloat64(),
					randFloat64(), randFloat64(), randFloat64(),
				},
			}
			if result, handled := engine.CalcHit(input); handled {
				return combatRoll{damage: result.Damage, crit: result.Crit}
			}
		}
	}
	attackMin, attackMax := player.meleeDamageRange()
	return rollSkillDamage(attackMin, attackMax, definition.damageRatio())
}

// defenderStatsOf 按伤害系选防构建守方快照（PhyDamage→外防，阴/阳/刚/
// 柔→对应分系防）。无推导数据返回 nil（引擎按零值守方）。
func defenderStatsOf(world *sceneLifecycle, ref skillDamageTargetRef, damageType string) *combatcalc.DefenderStats {
	stats := world.npcCombatStatsOf(ref)
	if stats == nil {
		return nil
	}
	defense := stats.phyDef
	switch {
	case strings.EqualFold(damageType, "StiDamage"):
		defense = stats.stiDef
	case strings.EqualFold(damageType, "JujDamage"):
		defense = stats.jujDef
	case strings.EqualFold(damageType, "NegDamage"):
		defense = stats.negDef
	case strings.EqualFold(damageType, "MasDamage"):
		defense = stats.masDef
	}
	return &combatcalc.DefenderStats{
		Defense:    int64(defense),
		Evasion:    int64(stats.evasion),
		Resilience: 0, // 封劲暂不推导（npcCombatStats 注释）
	}
}

// damageRatio 是公式倍率（formula.lua 的 ratio；<=0 由公式按 1.0 处理）。
func (definition combatSkillDefinition) damageRatio() float64 {
	if definition.damageWaves <= 0 {
		return 0
	}
	return float64(definition.baseDamage) / 100 / float64(definition.damageWaves)
}

// hurtCount 是公式 HurtCount 计数器的解析值（客户端 Src=1 上下文）：
// 多段判定帧数优先，其次 varprop HitCount 列，最后回落正 r 段数。
func (definition combatSkillDefinition) hurtCount() int {
	if n := len(definition.hitFrameMs); n > 0 {
		return n
	}
	if definition.hitCount > 0 {
		return int(definition.hitCount)
	}
	if definition.damageWaves > 0 {
		return int(definition.damageWaves)
	}
	return 1
}

// primaryDamageRow 返回每击公式路径使用的 r 行：取第一条公式号 >0 的行
// （绝大多数玩家技能单 r 行；多 r 行的 NPC/buff 伤害包属异形，走 legacy）。
func (definition combatSkillDefinition) primaryDamageRow() (skillDamageRow, bool) {
	for _, row := range definition.damageRows {
		if row.formulaID > 0 {
			return row, true
		}
	}
	return skillDamageRow{}, false
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

// skillCooldownFrames 产出出招 CD 行（§3 行 1/3 顺序，中间夹
// active_skill_rec 行由调用方写入）：个人行 team 恒 0xFFFF；GCD 行
// 类别固定 50000、时长取 varprop AddCoolDownTime（publicCD）、team 取
// skill_static CoolDownTeam。GCD 行仅在配置同时给出 CoolDownTeam 与
// AddCoolDownTime 时下发（无我剑诀等无 team 套路无公共行——官方从不发
// team=0 的 50000 行，客户端会把它当全组公共 CD，一招转全场）。
func skillCooldownFrames(definition combatSkillDefinition, now time.Time) (personal, public []byte, err error) {
	// 纪元口径：native-menu 反编译同方言（skillCooldownFrameWithDuration，
	// UnixMilli）真机转圈亲证——客户端 CD 倒计时只消费 col2-col1 时长，
	// 绝对纪元不参与。官服逐格方言用 1601(FILETIME) 纪元毫秒（g 14.660
	// col1=UnixMilli+11644473600000 逐位吻合），待该方言启用时同口径。
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
	if definition.cooldownTeam <= 0 || definition.publicCD <= 0 {
		return personal, nil, nil
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
		{Index: clientdata.MustPlayerProperty("ActionSet").Index, Name: "ActionSet", Value: clientdata.StringValue("0h")},
		{Index: clientdata.MustPlayerProperty("State").Index, Name: "State", Value: clientdata.StringValue("stand")},
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
		{Index: clientdata.MustPlayerProperty("State").Index, Name: "State", Value: clientdata.StringValue(action)},
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

// beginSkillUse 受理一笔出招：CD/GCD 闸 + MP/SP/轻功结算 + 出招簿记。
// settled=true 为冲锋落地重入专用——CD/MP/轻功已在 c61 时点
// （commitRushDashCosts）结算过，跳过闸与扣费，只补簿记（防止个人 CD
// 把自己的落地续招拒掉）。
func (p *playerActor) beginSkillUse(definition combatSkillDefinition, target uint64, now time.Time, settled bool) (bool, string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !settled {
		if p.skillLockUntil.After(now) {
			return false, fmt.Sprintf("skill locked remaining=%s", p.skillLockUntil.Sub(now).Round(time.Millisecond))
		}
		if ready := p.skillCooldowns[definition.id]; ready.After(now) {
			return false, fmt.Sprintf("personal cooldown remaining=%s", ready.Sub(now).Round(time.Millisecond))
		}
		// 公共 GCD 只拦套路切换（Java PlayerActor.beginSkillUse:1061-1064
		// 同构、bug.txt:8「同套路连招不在触发公共7秒冷却，不同套路会看到7秒CD
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
	}
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
	return s.livingCombatSelectionLocked()
}

// livingCombatSelectionLocked 是 selectedCombatTarget 的锁内形态
// （要求调用方已持有 s.run.mu）：选中对象须仍是存活战斗 NPC 才可用；
// 陈旧格（对象已消亡/非战斗交互/owner 不符）与空选同一待遇，视为
// 没有目标。
func (s *sceneLifecycle) livingCombatSelectionLocked() (uint32, uint32, bool) {
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
// （历史测试路径保留：伤害直落选中战斗 NPC；无玩家上下文，跳过内功
// on_attack 分发）。
func (s *sceneLifecycle) applySelectedSkillDamage(amount int32, now time.Time) (skillDamageOutcome, error) {
	id, ownerID, ok := s.selectedCombatTarget()
	if !ok {
		return skillDamageOutcome{}, nil
	}
	return s.applySkillDamageToNPC(id, ownerID, amount, nil, now)
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
// 与迟到者看到一致的死亡/重生周期；挂尸掉落（非实体宝箱）随后补发
// CanPick/PickMember 冒光属性（击杀者首发+其他成员中继）；最后把击杀
// 点排队的宝箱实体物化帧（0x0D+0x1F）锁外扇出。apply* 内部自持
// run.mu，调用点在锁外，各扇出独立加锁安全。
func (s *sceneLifecycle) relayNPCKillFrame(id uint32, frame []byte, killerName string) {
	if s == nil || s.run == nil || frame == nil {
		return
	}
	s.run.relayNPCPropertyFrame(id, frame, s)
	s.relayCorpsePickFrames(id, killerName)
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

// skillRowCurTargetCountFrame 即 View40 技能行 CurTargetCount 帧（官服
// 伤害段簇首击前=命中数、击杀后=0，f 43.432/44.926 实证；isView=1）。
func skillRowCurTargetCountFrame(row uint16, count int32) ([]byte, error) {
	return sceneObjectProperties(uint32(viewportSkill), uint32(row), 1, []clientdata.IndexedProperty{
		{Index: propSkillCurTargetCount, Name: "CurTargetCount", Value: clientdata.Int32Value(count)},
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

// refreshTargetsPerTick 范围型（以施法者为中心的圆/扇/矩形）逐段重索敌
//（用户口径：技能期间进入范围的敌人也要吃伤害——官服剑阵跟随移动的
// 语义）；锁定单体/目标区域保持出招快照。
func (definition combatSkillDefinition) refreshTargetsPerTick() bool {
	switch definition.targetMode {
	case skillTargetCasterCircle, skillTargetCasterSector, skillTargetCasterRectangle:
		return true
	}
	return false
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
// 整段跳过）。累计威胁进 0x13，死亡清仇恨排重生。player 非空时先走
// 内功特效 on_attack 分发（实中目标才到这；extra_damage 并入落血值）。
func (s *sceneLifecycle) applySkillDamageToNPC(id, ownerID uint32, amount int32, player *playerActor, now time.Time) (skillDamageOutcome, error) {
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	entityID := worldcore.EntityID(id)
	vitals, hasVitals := s.run.combat.Vitals(entityID)
	meta, exists := s.npcMeta(entityID)
	if !hasVitals || !exists || meta.ownerID != ownerID || meta.interaction != npcInteractionCombat || vitals.HP <= 0 {
		return skillDamageOutcome{}, nil
	}
	if player != nil {
		amount = player.dispatchNeiGongAttack(id, vitals.HP, vitals.MaxHP, amount, now)
	}
	if amount <= 0 || !vitals.ApplyDamage(amount) {
		return skillDamageOutcome{}, nil
	}
	s.run.combat.WriteVitals(entityID, vitals)
	state := vitals.ActorState(id, ownerID)
	if brain, isCombat := s.run.combat.Brain(entityID); isCombat {
		brain.Threat += amount
		// 受击锁定攻击者（Java NpcCombatAi 反击入口）：NotPositive 被动
		// NPC 的唯一索敌来源——无目标时把攻击者写进 TargetMember，tick
		// 随后按普通追击/攻击/栓绳流程反击。已有目标（粘性）不覆盖。
		if brain.TargetMember == 0 && state.HP > 0 && s.wireID != 0 {
			brain.TargetMember = s.wireID
		}
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
// 官服伤害段簇（f 43.432~44.539 / g 22.337~56.468 定版）：
// 首段 View40 行 CurTargetCount=命中数 → 每击 [FP → SP → 0x13 rec114
// 行0列1(u16,+500/击) → c1=17 → (第2击起 c1=10 段序) → msg205] →
// NPC 血量 0x2D 多属性（HPRatio/HitHPRatio）→ 击杀簇 [0x12 del rec114
// 行0 → c1=29 击退] → 击杀后 CurTargetCount=0。
// 旧方言的每击 msg18/0x2D(28/112) 已删（官服 0 次出现）。
func writeSkillSettleFrames(link sceneMessageConnection, world *sceneLifecycle, definition combatSkillDefinition, player *playerActor, hits []skillSettleHit, tickIndex int, remote string) (totalLoot int32, err error) {
	if len(hits) == 0 {
		return 0, nil
	}
	playerName := player.snapshotName()
	// 首段：View40 技能行 CurTargetCount（官服 g/f 首击前恒发）。
	if tickIndex == 0 {
		if row := player.skillViewRow(definition.id); row > 0 {
			if frame, ferr := skillRowCurTargetCountFrame(row, int32(len(hits))); ferr == nil {
				if err := link.WriteFrame(frame); err != nil {
					return totalLoot, err
				}
			}
		}
	}
	hitOrdinal := int32(tickIndex + 1)
	for _, hit := range hits {
		// 每击 +2 怒：FP/SP 双帧（官服恒同值成对）。
		sp, spFrame, spErr := player.addSP(ragGainCleanHit)
		if spErr != nil {
			return totalLoot, spErr
		}
		if fpFrame, ferr := objectSingleProperty(playerObjectID, playerOwnerID, propRushFP, "FP", clientdata.Int32Value(sp)); ferr == nil {
			if err := link.WriteFrame(fpFrame); err != nil {
				return totalLoot, err
			}
		}
		if spFrame != nil {
			if err := link.WriteFrame(spFrame); err != nil {
				return totalLoot, err
			}
		}
		// 0x13 rec114 行0列1 官服形态（u16 +500/击累计）。
		if err := link.WriteFrame(fightRelationHitCellFrame(hitOrdinal)); err != nil {
			return totalLoot, err
		}
		// c1=17 伤害飘字 + msg205 命中确认（对旁观者中继）。
		hitFrame, herr := skillHitDamageFrame(hit.ref.id, hit.ref.ownerID, hit.damage, definition.id, definition.scriptLabel(), hit.outcome.model, playerName)
		if herr != nil {
			return totalLoot, herr
		}
		confirmFrame, cerr := skillHitConfirmFrame(playerObjectID, playerOwnerID, definition.id, hit.ref.id, hit.ref.ownerID)
		if cerr != nil {
			return totalLoot, cerr
		}
		for _, frame := range [][]byte{hitFrame, confirmFrame} {
			if err := link.WriteFrame(frame); err != nil {
				return totalLoot, err
			}
			if world != nil && world.run != nil {
				world.run.relayCombatFrame(hit.ref.id, frame, world, "skill hit")
			}
		}
		// NPC 血量 0x2D 多属性帧（官服形态；替代旧 0x10 面板整帧）。
		if hpFrame, ferr := multiPropUpdateFrame(hit.ref.id, hit.ref.ownerID, []clientdata.IndexedProperty{
			{Index: propHPRatio, Name: "HPRatio", Value: clientdata.Int32Value(hit.outcome.hpRatio)},
			{Index: propHitHPRatio, Name: "HitHPRatio", Value: clientdata.Int32Value(hit.outcome.hpRatio)},
		}); ferr == nil {
			if err := link.WriteFrame(hpFrame); err != nil {
				return totalLoot, err
			}
		}
		// 第 2 击起：c1=10 怒气段序（值=段序 2/3/4）。
		if hitOrdinal >= 2 {
			if frame, rerr := rageComboFloatFrame(hitOrdinal); rerr == nil {
				if err := link.WriteFrame(frame); err != nil {
					return totalLoot, err
				}
			}
		}
		if hit.outcome.killed {
			// 击杀簇（官服序）：fight_relation 撤销 → c1=29 击退；
			// 死亡面板/尸体拾取走既有 relayNPCKillFrame 链。
			if err := link.WriteFrame(fightRelationRowDeleteFrame()); err != nil {
				return totalLoot, err
			}
			totalLoot += hit.outcome.loot
			if knock, kerr := knockbackFloatFrame(hit.ref.id, hit.ref.ownerID, playerObjectID, playerOwnerID, hit.outcome.model, playerName); kerr == nil {
				if err := link.WriteFrame(knock); err != nil {
					return totalLoot, err
				}
			}
			world.relayNPCKillFrame(hit.ref.id, hit.outcome.frame, playerName)
		}
	}
	// 击杀后 View40 行 CurTargetCount=0（官服 44.926 实证）。
	killed := false
	for _, hit := range hits {
		if hit.outcome.killed {
			killed = true
			break
		}
	}
	if killed {
		if row := player.skillViewRow(definition.id); row > 0 {
			if frame, ferr := skillRowCurTargetCountFrame(row, 0); ferr == nil {
				if err := link.WriteFrame(frame); err != nil {
					return totalLoot, err
				}
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
// 实证形态）：按 trigger_effect 帧位逐段调度——每击×每目标独立掷值
// （rollSkillHit：combatcalc 每击公式路径优先，legacy 回退），每段对仍
// 存活目标发全套命中帧；首段命中进战斗+目标控制，后续段刷新战斗活跃
// 时刻；每段每目标 +2 回怒；末段后按整招累计伤害喂经脉真气。
func scheduleSkillMultiHits(link sceneMessageConnection, world *sceneLifecycle, definition combatSkillDefinition,
	player *playerActor, targets []skillDamageTargetRef, custom clientCustomMessage, remote string) {
	if len(targets) == 0 {
		return
	}
	times := skillHitTimes(definition)
	if len(times) == 0 {
		return
	}
	// 范围型逐段重索敌（refreshTargetsPerTick）：以玩家实时位置为形状
	// 中心重查——段间进入范围的敌人也吃该段伤害；玩家坐标优先取
	// combat world 权威值（m15/212 持续更新），回落 211 施法位。
	perTickRefresh := definition.refreshTargetsPerTick()
	castAt, _ := useSkillCastTransform(custom)
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
			tickTargets := targets
			if perTickRefresh {
				origin := castAt
				if live, ok := world.combatNPCTransform(playerObjectID); ok {
					origin = live
				}
				tickTargets = world.queryShapedSkillTargets(origin, definition.areaRadius, definition.areaHeight,
					definition.sectorAngleForShape(), definition.areaWidthForShape())
				if len(tickTargets) == 0 {
					return // 本段无目标（都死了/没人进范围）：零帧跳过
				}
			}
			hits := make([]skillSettleHit, 0, len(tickTargets))
			killed := false
			stageDamage := int64(0)
			for _, ref := range tickTargets {
				roll := rollSkillHit(player, definition, ref, world, len(targets))
				if roll.damage <= 0 {
					continue // 本击未命中：该目标本段不发帧
				}
				outcome, err := world.applySkillDamageToNPC(ref.id, ref.ownerID, roll.damage, player, now)
				if err != nil {
					log.Printf("%s: multi-hit stage %d apply id=%d: %v", remote, tick, ref.id, err)
					continue
				}
				if outcome.frame == nil {
					continue // 前段已击杀/目标失效：本段跳过（不发帧）
				}
				hits = append(hits, skillSettleHit{ref: ref, damage: roll.damage, outcome: outcome})
				stageDamage += int64(roll.damage)
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
			totalLoot, err := writeSkillSettleFrames(link, world, definition, player, hits, tick, remote)
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
			log.Printf("%s: multi-hit stage %d/%d skill=%s hits=%d damage_sum=%d", remote, tick+1, len(times), definition.id, len(hits), stageDamage)
		})
	}
}

func handleSkillCustom(link sceneMessageConnection, player *playerActor, world *sceneLifecycle, mount *mountSession, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomUseSkill {
		return false, nil
	}
	// 骑乘技能先于武学目录闸分流（menu2 handleRideSkillCustom :695-738
	// 同位）：riding_* 不在 zhaoshi 编译目录（无动作时间线），落在
	// parseUseSkillCustom 恒 false——旧闸位在其后实际从未到达（骑乘技能
	// 全被当 unlearned 吞掉）。dismount 族转下马；步行发骑乘技/骑乘发非
	// 白名单技忽略；其余骑乘技能发官服受理簇（mount_skills.go）。
	// riding_start（骑术精通）= 召唤技，不在 RideSkillID 白名单：步行按
	// 选中上马（soleOwned 同 handleRideCustom 口径）；骑乘中忽略。
	if mount != nil && player != nil {
		if len(custom.Values) >= 2 && custom.Values[0].Type == 2 &&
			custom.Values[0].Int32 == clientCustomUseSkill && custom.Values[1].Type == 6 {
			skillID := custom.Values[1].Text
			if skillID == ridingStartSkillID {
				if player.ridingMount() {
					return true, nil // 骑乘中召唤技无操作
				}
				if err := mount.startSelectedMount(); err != nil {
					log.Printf("%s: riding start mount: %v", remote, err)
				}
				return true, nil
			}
			if skillID != "" && mount.catalog.hasRideSkill(skillID) {
				if mountSkillGate(mount, skillID) != mountSkillPass {
					return true, nil // 下马已收尾 / 忽略已记日志
				}
				if err := mount.emitRideSkillCast(skillID); err != nil {
					log.Printf("%s: ride skill cast frames: %v", remote, err)
				}
				return true, nil
			}
		}
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
	// 骑乘中发非白名单（普通武学）技能：闸拒绝（骑乘白名单外的出招
	// 一律忽略，menu2 "ride skill not in mount skills" 同口径）。
	if mount != nil && player.ridingMountConfig() != "" {
		mountSkillGate(mount, definition.id)
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
		// 瞄准对象优先（211 长式 idx10；落地重组 211 也带它），选中目标
		// 兜底——冲锋落地续招时选中目标可能已换/丢失。
		if aimID, aimOwner := dashLandingAimObject(custom); aimID != 0 && world.combatNPCAlive(aimID, aimOwner) {
			targetID, targetOwnerID = aimID, aimOwner
		} else if selectedID, selectedOwner, ok := world.selectedCombatTarget(); ok {
			targetID, targetOwnerID = selectedID, selectedOwner
		} else {
			log.Printf("%s: reject skill id=%s without living selected target", remote, definition.id)
			return true, nil
		}
	}
	// 冲锋族按压闸（2026-11-11）：锁定目标的冲锋分支（直收/暂存）提前
	// return，永远到不了 beginSkillUse 的 CD/GCD 闸——commitRushDashCosts
	// 只起 CD 不查，冷却中按压会照常受理→走近→照常出招（贴脸即无视 CD
	// 连发）。官服「受理自由」（f 十连按全受理）的样本全部无 CD 在途，
	// 不能外推到冷却期；此处按 beginSkillUse 同语义先过锁招/个人 CD/
	// 公共 GCD 三闸，MP/SP/轻功仍留在执行提交点（「CD/MP 在执行时点结」
	// 模型不变）。
	if rushStageEnabled() && definition.canRush {
		if ok, reason := player.rushPressGate(definition, time.Now()); !ok {
			log.Printf("%s: reject skill id=%s: %s", remote, definition.id, reason)
			return true, nil
		}
	}
	// 攻击距离闸（Java 八轮拍板「按技能真实射程控制释放距离」）：施法者
	// 坐标取 211 自带上报（宽容解析，缺省原点同 Java）；距离=施法者→目标。
	// 冲锋族距离模型（官服 b/e.pcapng 重解码 + f.pcapng 十连按修正
	// 2026-11-08）：面板射程（castRange/TargetShapePara2）不是可攻击距离；
	// 身前近距（≤rushStageNearCastMeters）直接施法，更远一律走暂存冲锋链
	//（受理簇→客户端自动接近→6.5m 内 c61 冲刺→212 落地出伤害）。
	// **受理与距离解耦**（f 十连按实证：14.6~19.8m 全受理零拒绝，kfkj02
	// 无 RushRange 列）——不做发起上限拒包，RushRange 仅存档参考。
	if definition.requiresTarget && world != nil {
		if targetTransform, hasTransform := world.combatNPCTransform(targetID); hasTransform {
			castAt, _ := useSkillCastTransform(custom)
			// 冲锋族几何基准 = 211 自报目标坐标（客户端世界观）；NPC 服务端
			// 走动不广播，两边可差 5m+，按服务端坐标算 c61 终点会被客户端
			// 校验拒冲（57587 会话实证）。非冲锋族仍按服务端坐标。
			geometry := targetTransform
			claimed, hasClaimed := useSkillClaimedTargetTransform(custom)
			if definition.stagedRushFamily() && hasClaimed {
				geometry = claimed
			}
			// xz 平面距离（与 212 三分 / tryRushArrival 同口径；技能.pcapng
			// 触发三证均平地，统一 2D 防坡地 y 差误判）。
			distance := math.Sqrt(float64(
				square(castAt.X-geometry.X) +
					square(castAt.Z-geometry.Z)))
			if rushStageEnabled() && definition.stagedRushFamily() {
				// 距离四桶（g/f/b 定版）：≤直收带直接执行簇（不走暂存链、
				// CD/MP 即刻结、不扣轻功）；其余一律暂存冲锋链（受理与距离
				// 解耦，f 十连按 14.6~19.8m 全受理零拒绝）。
				if distance <= rushDirectCastMeters(definition) {
					return executeRushDirect(link, player, world, definition, targetID, targetOwnerID, geometry, castAt, distance, remote)
				}
				return stageRushCast(link, player, world, definition, targetID, targetOwnerID, geometry, castAt, distance, remote)
			} else if distance > float64(definition.castRange) {
				log.Printf("%s: reject skill id=%s out of range %.0fm > %.0fm",
					remote, definition.id, distance, definition.castRange)
				return true, nil
			}
		}
	}
	// 范围型点地冲锋（技能.pcapng 定版）：hyjf07 型（CanRush + 非锁定
	// 目标 + 211 短式带地面点）。分带：≤3.5 受理即直收（55.5@1.45 无 c61
	// 直接 msg20+伤害，主出招路径已覆盖）；>3.5 受理即 c61 冲向地面点
	// 身前 bind（3.96/5.52 档1 qg_fastrun、b/e 8.2 档2 qg_fastruning_s，
	// 档1|档2 分界=GRushRange+Add 6.5；1216 终点=ground 身前 1.5，
	// dash=dist-1.5 双证）。
	if rushStageEnabled() && definition.canRush && !definition.requiresTarget && world != nil {
		if ground, ok := useSkillGroundPointTransform(custom); ok {
			if castAt, hasCast := useSkillCastTransform(custom); hasCast {
				issueAreaGroundRushDash(link, player, definition, castAt, ground, remote)
			}
		}
	}
	now := time.Now()
	target := uint64(targetID) | uint64(targetOwnerID)<<32
	// 冲锋落地重入（pending.settled）：CD/MP 已在 c61 时点结算，跳过
	// 闸/扣费/CD 行（官服落地簇无 CD 行；个人 CD 会拒掉自己的续招）。
	settledResume := player.resumeSettleSkipSnapshot()
	// 顶掉补删（Java §D8 对拍）：GCD 不硬闸，连招会把上一招的收尾定时器
	// 顶掉；新招受理时上一招仍挂着的，立即补发它的 0x12+清除帧——
	// active_skill_rec 的 add/del 恒 1:1（参考服 8/8、52905 14/14）。
	// 通道型 SP 闸前置（挑战.pcapng：SP=0 四连受理零回帧——CD 行/簿记
	// 全不发，静默吞；SP 逐波扣故按「付得起首波」判）。
	if definition.channel != nil && player.skillSP() < definition.channel.spPerWave {
		log.Printf("%s: swallow channel skill id=%s: SP %d < %d (official silent)",
			remote, definition.id, player.skillSP(), definition.channel.spPerWave)
		return true, nil
	}
	superseded := player.currentSkillSnapshot()
	if ok, reason := player.beginSkillUse(definition, target, now, settledResume); !ok {
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
	// §3 出招帧序（Java handleValues 逐位同序）：个人 CD 行 →
	// active_skill_rec 行 → GCD 行 → View40 行 CanUse=0 → 玩家全量 →
	// msg337 施法回显 → msg20。
	// 通道型受理态帧（挑战.pcapng 官序：CurSkillID → CurSkillTarget →
	// Cant* 置位 → BufferInfo 槽 CSV，均先于 CD 行；CurSkillID 设置→
	// 波1 清空的瞬时对是客户端通道演出触发链的组成）。
	if definition.channel != nil {
		writeChannelAcceptStateFrames(link, definition, targetID, targetOwnerID, now, remote)
	}
	if !settledResume {
		// CD 行官服方言（0x11 空行+0x13 逐格+槽位覆写）+ 受理时惰性清扫
		// 过期行（f 6.224 kfkj01 官方普通出招同构）。
		for _, sweep := range player.sweepExpiredCooldownRows(now) {
			if err := link.WriteFrame(sweep); err != nil {
				return true, err
			}
		}
		if err := player.writeOfficialCooldownRows(link, definition.cooldownCategory, definition.cooldownTeam,
			definition.personalCD, definition.publicCD, now); err != nil {
			return true, err
		}
	}
	if err := link.WriteFrame(serverRecordAddString(playerObjectID, playerOwnerID, recordActiveSkill, definition.id)); err != nil {
		return true, fmt.Errorf("write active skill row %s: %w", definition.id, err)
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
	// 施法中不锁行动标志（挑战.pcapng tjj07 受理簇：NoLimit=1 技能在
	// m20 前发 CurSkillCanNoLimit=1，客户端 m20 处理链据此放行移动——
	// 缺它=太极剑舞等可移动技被定死原地，2026-11-12 真机四轮实证；
	// 收招清除簇 currentSkillClearUpdate 已含归零帧）。
	if definition.noLimit {
		if noLimitFrame, nlErr := objectSingleProperty(playerObjectID, playerOwnerID,
			propCurSkillCanNoLimit, "CurSkillCanNoLimit", clientdata.ByteValue(1)); nlErr == nil {
			if err := link.WriteFrame(noLimitFrame); err != nil {
				return true, err
			}
		}
	}
	// msg20 技能动作帧：目标对象三分（resolveSkillActionTarget——锁定族
	// =211 瞄准/选中目标、自增益=自己、其余=空对象）。锁定族官方走
	// msg19 锁定帧（c61 冲刺+0x1F+24 值大帧），但 42 份抓包 0 次出现且
	// 此前实现有「无冲刺动画」回归史——继续 msg20 路线（分歧记录在
	// docs/combat-rewrite-hardcoded-audit.md）。通道型名字=zhaoshi 段 1
	// 动作名（挑战.pcapng：hyjf08 官方 m20='qzj_07_1' 非技能 id，受理簇
	// +122ms；常规技能仍是技能 id）。
	actionTargetID, actionTargetOwner := resolveSkillActionTarget(definition, custom, world, targetID, targetOwnerID)
	actionName := definition.id
	if definition.channel != nil && definition.actionName != "" {
		actionName = definition.actionName
	}
	actionFrame, err := nativeSkillActionFrame(actionName, actionTargetID, actionTargetOwner)
	if err != nil {
		return true, fmt.Errorf("encode native skill action %s: %w", definition.id, err)
	}
	if err := link.WriteFrame(actionFrame); err != nil {
		return true, err
	}
	log.Printf("%s: started skill action skill=%s script=%s total=%s caster=%d-%d target=%d-%d waves=%d ticks=%d",
		remote, definition.id, definition.scriptLabel(), definition.totalActionDuration(),
		playerObjectID, playerOwnerID, targetID, targetOwnerID, definition.damageWaves, len(definition.hitFrameMs))
	launchSkillDamagePhase(link, player, world, definition, custom, targetID, targetOwnerID, castEpoch, remote)
	state := player.stateSnapshot()
	log.Printf("%s: accepted skill id=%s mp=%d sp=%d qg=%d base_damage=%d waves=%d target=%d personal_cd=%s public_cd=%s",
		remote, definition.id, state.MP, player.skillSP(), state.QingGongPoint, definition.baseDamage, definition.damageWaves, targetID, definition.personalCD, definition.publicCD)
	return true, nil
}

// launchSkillDamagePhase 出招后的伤害阶段（普通出招 msg20 后、冲锋直收簇 /
// 落地簇尾共用）：目标快照 → 多段调度 / 单段即结 → buff 到期挂接 →
// 跳跃重置 → 收招清簇。
func launchSkillDamagePhase(link sceneMessageConnection, player *playerActor, world *sceneLifecycle,
	definition combatSkillDefinition, custom clientCustomMessage, targetID, targetOwnerID uint32,
	castEpoch uint64, remote string) {
	now := time.Now()
	appliedBuffs := stagePlayerSkillEffects(player, definition, now, remote)
	// 通道型（寒心恨雪族）：服务端波次引擎替代动画帧驱动——逐波单目标
	// 重新索敌、每波扣 SP、无活目标即停（skill_channel.go）。
	if definition.channel != nil {
		launchChannelWaves(link, world, definition, player, custom, targetID, targetOwnerID, remote)
		log.Printf("%s: launched channel waves skill=%s waves=%d interval=%s sp_per_wave=%d",
			remote, definition.id, definition.channel.waves, definition.channel.interval, definition.channel.spPerWave)
		return
	}
	// 出招时刻解析目标快照（多段按此逐段结算，段间不追补新目标）。
	targets := world.resolveSkillDamageTargets(definition, custom, targetID, targetOwnerID)
	if len(definition.hitFrameMs) > 0 {
		// 多段技能：按 trigger_effect 帧位逐段调度（52795 抓包：每段全套
		// 命中帧、死亡目标跳过）。2026-11-08 数值重写：每击×每目标独立
		// 掷值结算（此前"出招掷一次、同值复用各段各目标"）。
		scheduleSkillMultiHits(link, world, definition, player, targets, custom, remote)
		log.Printf("%s: scheduled multi-hit skill id=%s ticks=%d targets=%d per_hit_roll=true",
			remote, definition.id, len(definition.hitFrameMs), len(targets))
	} else if len(targets) > 0 {
		// 单段技能：逐目标掷值落血（Java settleTargets——每目标独立
		// 掷骰），未命中（damage=0）整段不发帧。
		hits := make([]skillSettleHit, 0, len(targets))
		for _, ref := range targets {
			roll := rollSkillHit(player, definition, ref, world, len(targets))
			if roll.damage <= 0 {
				continue
			}
			outcome, err := world.applySkillDamageToNPC(ref.id, ref.ownerID, roll.damage, player, now)
			if err != nil {
				log.Printf("%s: settle damage %s: %v", remote, definition.id, err)
				return
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
			totalLoot, err := writeSkillSettleFrames(link, world, definition, player, hits, 0, remote)
			if err != nil {
				log.Printf("%s: settle frames %s: %v", remote, definition.id, err)
				return
			}
			awardSettleLoot(link, player, totalLoot, fmt.Sprintf("targets=%d", len(hits)), remote)
			if err := writeWujiSequenceFrames(link, definition, player, hits, 0); err != nil {
				log.Printf("%s: wuji frames %s: %v", remote, definition.id, err)
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
					log.Printf("%s: jingmai feed %s: %v", remote, definition.id, err)
					return
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
