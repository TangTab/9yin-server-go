package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/sim"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

const (
	playerObjectID uint32 = 1
	playerOwnerID  uint32 = 1

	// 这些属性紧跟在 playerProgressFields（171..207）之后追加。其顺序
	// 必须与 sceneVisiblePropertyFields 保持一致，因为 FxNet2 属性档案
	// 只携带数字索引，不携带名称。228 起已被 player_progress.go 的
	// MaxSP/MaxSPAdd 占用；新增属性只能取更高且未被占用的索引。
	propNeigongPKStatus uint16 = 208
	propDead            uint16 = 209
	propCantUseSkill    uint16 = 210
	propCurSkillID      uint16 = 211
	propPauseTime       uint16 = 212
	// 原生主角 UI 将这些属性绑定为整数。它们描述的是对外通告的恢复
	// 速率；真正的 HP/MP 变动仍以服务器为来源。
	propHPHeartSpeed    uint16 = 213
	propHPHeartSpeedAdd uint16 = 214
	propMPHeartSpeed    uint16 = 215
	propMPHeartSpeedAdd uint16 = 216
	// form_main_player 的恢复提示框从这些独立属性获取实际恢复量，而不
	// 是从 *HeartSpeed 读取。
	propHPUpSpeed    uint16 = 217
	propHPUpSpeedAdd uint16 = 218
	propMPUpSpeed    uint16 = 219
	propMPUpSpeedAdd uint16 = 220
	// 常规的服务器技能状态会发布完整的公开用技元组。FxGameLogic 在判定
	// 一个指向自身的技能能否进入其原生动作时，也会直接读取
	// ModifySkillLockTime。
	propCurSkillEffectID    uint16 = 221
	propCurSkillLevel       uint16 = 222
	propCurSkillTarget      uint16 = 223
	propModifySkillLockTime uint16 = 224
	// 已学技能对象对这个字节的使用独立于服务器的请求受理。现代快捷栏
	// UI 直接读取 CanUse；当它缺失时，即使受保护的战斗请求路径仍能到达
	// C2S 211，图标也会被渲染为不可用。
	propSkillCanUse uint16 = 225
	// 当前客户端并不通过 School 表达所有归属关系。其角色面板依次选用
	// Force，再 NewSchool，最后 School。因此必须让这些协商属性保持显式，
	// 后期势力和隐藏门派才不会被渲染成 \"无门派\"。
	propForce     uint16 = 226
	propNewSchool uint16 = 227

	// BufferListStr 是原生玩法的 BuffContainer 快照。BufferInfo 槽位是
	// 原生视觉存储的投影。两者必须描述同一批活动 Buff：仅有 UI 的
	// BufferInfo 不足以通过 qinggong_3 的原生 HasBuff("buf_qg_drift") 门控。
	bufferListPropertyIndex      uint16 = 143
	bufferInfoFirstPropertyIndex uint16 = bufferListPropertyIndex + 1
	bufferInfoSlotCount          uint16 = 24

	redSuperArmorBuffSlot uint16 = 1
	waterDriftBuffSlot    uint16 = 2
	waterJumpBuffSlot     uint16 = 3
	innerPowerBuffSlot    uint16 = 4
	yiHuaSkillBuffSlot    uint16 = 5
	yiHuaSkill05BuffSlot  uint16 = 6
	taiJiStanceBuffSlot   uint16 = 7
	taiJiSkill05BuffSlot  uint16 = 8
	taiJiSkill07BuffSlot  uint16 = 9
	// 明玉神功“料敌先机”是独立的短期闪避 Buff；它不能覆盖常驻内功
	// Buff，也不能挤占轻功/技能 Buff 的槽位。
	yihuaPassiveBuffSlot uint16 = 10

	redSuperArmorStaticData uint32 = 8682
	waterDriftStaticData    uint32 = 216
	waterJumpStaticData     uint32 = 5786

	// 装备中的内功使用同一个原生 BuffStore，但不得与临时战斗/轻功槽位
	// 共用。其租约在出生时以及每次 215 响应时都从权威装备状态续期。
	innerPowerBuffLease = 24 * time.Hour

	// SP 是当前现代客户端的怒气资源。已安装的技能定义和现网 211 处理
	// 都以 80 作为本地测试的满池值。所有 GM 及后续战斗获取路径都经由
	// 这些辅助函数，客户端属性永远不会成为该资源的权威。
	maxPlayerSP int32 = 80

	// 本地测试的打坐恢复按现有的服务器秒每秒结算一次。用户规则：存活
	// 的角色无论装备或内功推导出的总量如何，都在十秒内回满
	// MaxHP/MaxMP。
	sitcrossFullRecoverySeconds int32 = 10
	sitcrossHeartMillis         int32 = 1000

	// 战斗恢复从不恢复真实 HP。它只在每个权威服务器秒内削减一次原生
	// HitHP-HP 瘀血层。
	combatBruiseRecoveryPerSecond int32 = 10

	// 战斗怒气恢复：战斗中的角色每个权威服务器秒回复五点 SP，直到 80
	// 上限为止，与正式服战斗中的缓慢回复一致。
	combatRageRecoveryPerSecond int32 = 5
)

var nativeBuffSlots = func() []uint16 {
	slots := make([]uint16, bufferInfoSlotCount)
	for index := range slots {
		slots[index] = uint16(index + 1)
	}
	return slots
}()

func bufferInfoPropertyIndex(slot uint16) uint16 {
	if slot == 0 || slot > bufferInfoSlotCount {
		panic(fmt.Sprintf("invalid BufferInfo slot %d", slot))
	}
	return bufferInfoFirstPropertyIndex + slot - 1
}

func bufferInfoPropertyName(slot uint16) string {
	if slot == 0 || slot > bufferInfoSlotCount {
		panic(fmt.Sprintf("invalid BufferInfo slot %d", slot))
	}
	return fmt.Sprintf("BufferInfo%d", slot)
}

// playerActor 将权威运行时角色桥接到当前客户端协商出的
// 可见属性 schema。对当前本地测试服务器而言，货币的权威属于会话；
// 一旦引入物品/货币存储，购买事务将成为
// 它的持久化所有者。
//
// 锁纪律：p.mu 是玩家状态（下方全部字段，含三份 sim 组件值）的
// 唯一锁。跨对象锁序恒为 sceneLifecycle.mu（外）→ p.mu（内）；
// 持 p.mu 期间不得调用 sceneLifecycle 的任何加锁方法。异步 timer
// 路径（技能清除、buff 到期）一律经由本类型的持锁方法访问状态。
type playerActor struct {
	mu                    sync.Mutex
	name                  string
	vitals                sim.Vitals
	qinggong              sim.QingGong
	speed                 sim.Motion
	silver                int32
	faction               string
	motion                playerMotion
	buffs                 map[uint16]activePlayerBuff
	activeQingGong        map[string]struct{}
	shortcuts             []playerShortcut
	learnedSkills         map[string]int32
	skillCooldowns        map[string]time.Time
	skillTeamCooldowns    map[int32]time.Time
	currentSkillID        string
	currentSkillEffectID  string
	currentSkillLevel     int32
	currentSkillTarget    uint64
	activeParry           activeParryRequest
	skillLockUntil        time.Time
	reviveAt              time.Time
	innerPowerProcReadyAt time.Time
	innerPowerHealReadyAt time.Time
	progress              playerProgress
}

// playerMotion 是刻意由服务器持有的 GM 状态。其值通过常规的现代对象
// 属性接收端发送，绝不通过 S->C 0x1E 发送。
type playerMotion struct {
	moveSpeed float32
	runSpeed  float32
	gravity   float32
}

// activePlayerBuff 是单个原生 BufferInfo 槽位在服务器侧的对应物。渲染
// 归客户端所有，但玩法决策必须读取这份状态，而不是信任客户端侧的
// 图标。
type activePlayerBuff struct {
	staticData uint32
	expiresUTC time.Time
	configID   string
	level      int32
}

// playerBuffRules 镜像 buf_fixed_red -> proppack 9679。它刻意不声明
// 伤害免疫：当前配置只授予受击特效免疫和技能锁定免疫。战斗/控制
// 入口将在这些 C2S 路径引入时
// 查询这份状态。
type playerBuffRules struct {
	CantHitEffect     bool
	CantBeSkillLocked bool
}

func newPlayerActor(name string, silver int32) *playerActor {
	learnedSkills := make(map[string]int32, len(starterSkillViews))
	for _, skill := range starterSkillViews {
		learnedSkills[skill.configID] = skill.level
	}
	now := time.Now()
	p := &playerActor{
		name:               name,
		vitals:             sim.NewVitals(1000, 500),
		qinggong:           sim.NewQingGong(now),
		speed:              sim.NewMotion(),
		silver:             silver,
		faction:            defaultRoleFaction,
		motion:             playerMotion{moveSpeed: 25, runSpeed: 25, gravity: 9.8},
		buffs:              make(map[uint16]activePlayerBuff),
		activeQingGong:     make(map[string]struct{}),
		learnedSkills:      learnedSkills,
		skillCooldowns:     make(map[string]time.Time),
		skillTeamCooldowns: make(map[int32]time.Time),
		progress:           newPlayerProgress(),
	}
	p.syncEquippedNeiGongBuffLocked(now)
	p.syncEquippedNeiGongResourcesLocked()
	return p
}

// snapshotLocked 把三份组件值组合为协议层消费的 entity.ActorState
// DTO，保证 0x10/0x0D 属性投影与迁移前字节一致。
// 调用方必须已持有 p.mu。
func (p *playerActor) snapshotLocked() entity.ActorState {
	state := p.vitals.ActorState(playerObjectID, playerOwnerID)
	state.Name = p.name
	state.QingGongPoint = p.qinggong.Point
	state.MaxQingGongPoint = p.qinggong.Max
	state.MaxQingGongPointAdd = p.qinggong.MaxAdd
	state.QingGongConsumeMul = p.qinggong.ConsumeMul
	state.SpeedRatio = p.speed.SpeedRatio
	return state
}

// stateSnapshot 供不持 p.mu 的调用方（连接循环、combatTick 的
// 日志路径）获取一份一致快照。
func (p *playerActor) stateSnapshot() entity.ActorState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.snapshotLocked()
}

func (p *playerActor) logicState() uint8 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.vitals.Logic
}

func (p *playerActor) setLogicState(state uint8) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.vitals.SetLogicState(state)
}

func (p *playerActor) applyDamage(amount int32) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.vitals.ApplyDamage(amount)
}

func (p *playerActor) spendMP(cost int32) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.vitals.SpendMP(cost)
}

func (p *playerActor) useQingGong(baseCost int32, now time.Time) (int32, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.qinggong.Use(baseCost, now)
}

func (p *playerActor) learnedSkillLevel(configID string) (int32, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	level, exists := p.learnedSkills[configID]
	return level, exists && level > 0
}

func (p *playerActor) learnSkill(configID string, level int32) {
	if configID == "" || level < 1 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.learnedSkills[configID] = level
}

func (p *playerActor) progressProperties() []clientdata.IndexedProperty {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.properties()
}

// setFaction 只在 GM 白名单和持久化外观更新都成功后才修改原生的
// School 属性。现代客户端对初始门派、势力和新门派统一使用这一个
// 属性。
func (p *playerActor) setFaction(faction string) ([]byte, error) {
	p.mu.Lock()
	p.faction = faction
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, factionProperties(faction))
}

// factionProperties 镜像 form_main_player.refresh_role_changed 的逻辑：
// 初始门派填充 School，江湖势力填充 Force，后期门派填充 NewSchool。
// 留空的伴随字段用于刻意清除先前的 GM 选择。
func factionProperties(faction string) []clientdata.IndexedProperty {
	school, force, newSchool := faction, "", ""
	switch {
	case strings.HasPrefix(faction, "force_"):
		school, force = "", faction
	case strings.HasPrefix(faction, "newschool_"):
		school, newSchool = "", faction
	}
	return []clientdata.IndexedProperty{
		{Index: 53, Name: "School", Value: clientdata.StringValue(school)},
		{Index: propForce, Name: "Force", Value: clientdata.StringValue(force)},
		{Index: propNewSchool, Name: "NewSchool", Value: clientdata.StringValue(newSchool)},
	}
}

// setActiveParry 记录一条客户端发起的 218 请求，供未来权威的受击裁定
// 器使用。它自身没有任何客户端视觉副作用：视觉上的招架/受击动作要等
// NPC 或玩家的攻击到达该裁定器之后
// 才会被选定。
func (p *playerActor) setActiveParry(value activeParryRequest) {
	p.mu.Lock()
	p.activeParry = value
	p.mu.Unlock()
}

func (p *playerActor) activeParrySnapshot() activeParryRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.activeParry
}

type npcAttackOutcome struct {
	stateChanged        bool
	parried             bool
	dodged              bool
	dead                bool
	innerPowerTriggered bool
	innerPowerHeal      int32
}

// resolveNPCAttack 是服务器权威的受击裁定。当前客户端已提供 C2S 的
// 主动招架入口（218），但其确切的战斗枚举尚未还原；对本地服务器而言，
// 每条新的 218 请求都授予一个 600ms 的招架窗口。闪避按攻击者和秒
// 确定性生成，重连无法重掷已接受的判定结果。
func (p *playerActor) resolveNPCAttack(attackerID uint32, damage int32, now time.Time) npcAttackOutcome {
	p.mu.Lock()
	deadAlready := p.vitals.HP <= 0
	p.mu.Unlock()
	if damage <= 0 || deadAlready {
		return npcAttackOutcome{}
	}
	parry := p.activeParrySnapshot()
	if !parry.At.IsZero() && now.Sub(parry.At) >= 0 && now.Sub(parry.At) <= 600*time.Millisecond {
		return npcAttackOutcome{parried: true}
	}
	profile, hasProfile := p.equippedInnerPowerPassive()
	activePassive, triggered := false, false
	if hasProfile && profile.mode == innerPowerModeIncomingDodgeHeal {
		activePassive, triggered = p.applyIncomingDodgePassive(profile, attackerID, now)
	}
	dodgeChance := baseLocalDodgeProbability
	if activePassive {
		dodgeChance += profile.dodgeAdd
	}
	if combatPercentRoll(attackerID, now, 18) < dodgeChance {
		healed := int32(0)
		if hasProfile && profile.mode == innerPowerModeIncomingDodgeHeal && !p.innerPowerHealCoolingDown(now) {
			p.mu.Lock()
			amount := (p.vitals.MaxHP*profile.healPercent + 99) / 100
			if p.vitals.ApplyHeal(amount) {
				healed = amount
			}
			p.setInnerPowerHealCooldownLocked(now.Add(profile.healCooldown))
			p.mu.Unlock()
		}
		return npcAttackOutcome{stateChanged: triggered || healed > 0, dodged: true, innerPowerTriggered: triggered, innerPowerHeal: healed}
	}
	p.mu.Lock()
	changed := p.vitals.ApplyDamage(damage)
	if !changed {
		p.mu.Unlock()
		return npcAttackOutcome{}
	}
	dead := p.vitals.HP <= 0
	if dead {
		p.reviveAt = now.Add(playerAutoReviveDelay)
	}
	p.mu.Unlock()
	rules := p.buffRules(now)
	if !rules.CantBeSkillLocked {
		p.mu.Lock()
		p.skillLockUntil = now.Add(500 * time.Millisecond)
		p.mu.Unlock()
	}
	return npcAttackOutcome{stateChanged: true, dead: dead}
}

// combatPercentRoll 对同一攻击者和同一服务器 tick 是确定性的。它让
// 本地测试保持可复现，同时不让重连重掷一次已经结算的 NPC 攻击。
func combatPercentRoll(attackerID uint32, now time.Time, salt uint64) int32 {
	value := uint64(attackerID)*1103515245 + uint64(now.Unix())*2654435761 + salt*97
	return int32(value % 100)
}

// applyIncomingDodgePassive 是内功档案的事件模式通用执行器，该档案由其
// 已安装的描述和子 Buff 编译而来。其他被动模式都接入同一个受击前的
// 派发点。
func (p *playerActor) applyIncomingDodgePassive(profile innerPowerPassiveProfile, attackerID uint32, now time.Time) (active, triggered bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	current := p.buffs[yihuaPassiveBuffSlot]
	active = current.staticData == profile.procStatic && current.expiresUTC.After(now.UTC())
	if active || p.innerPowerProcReadyAt.After(now) || combatPercentRoll(attackerID, now, 17) >= profile.procChance {
		return active, false
	}
	expires := now.UTC().Add(profile.procLifetime)
	p.buffs[yihuaPassiveBuffSlot] = activePlayerBuff{staticData: profile.procStatic, expiresUTC: expires, configID: profile.procBuffID, level: profile.buffLevel}
	p.innerPowerProcReadyAt = now.Add(profile.procCooldown)
	return true, true
}

func (p *playerActor) innerPowerHealCoolingDown(now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.innerPowerHealReadyAt.After(now)
}

func (p *playerActor) setInnerPowerHealCooldown(next time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.setInnerPowerHealCooldownLocked(next)
}

// setInnerPowerHealCooldownLocked 供已持 p.mu 的受击裁定路径使用。
// 调用方必须已持有 p.mu。
func (p *playerActor) setInnerPowerHealCooldownLocked(next time.Time) {
	p.innerPowerHealReadyAt = next
}

// expireInnerPowerPassive 只清除料敌先机。随后常规的连接 tick 会发送
// 一个常规 0x10 属性档案，同时移除 BufferInfo10 和与之匹配的
// BufferListStr 条目。
func (p *playerActor) expireInnerPowerPassive(now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[yihuaPassiveBuffSlot]
	if !ok || buff.expiresUTC.After(now.UTC()) {
		return false
	}
	delete(p.buffs, yihuaPassiveBuffSlot)
	return true
}

func (p *playerActor) skillLocked(now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.skillLockUntil.After(now)
}

// reviveWhenDue 实现本地测试的复活策略。它刻意使用常规角色属性；官方
// 确切的复活 UI/传送路由是一个独立的协议特性，此处不做虚构。
func (p *playerActor) reviveWhenDue(now time.Time) bool {
	p.mu.Lock()
	if p.reviveAt.IsZero() || now.Before(p.reviveAt) {
		p.mu.Unlock()
		return false
	}
	p.reviveAt = time.Time{}
	p.skillLockUntil = time.Time{}
	// 完整复活迁移：数值回满 + 轻功条回满 + 速度比重置，
	// 等价于原 entity.Actor.Revive 的全语义。
	p.vitals.Revive()
	p.qinggong.Reset(now)
	p.speed = sim.NewMotion()
	p.mu.Unlock()
	return true
}

func (p *playerActor) progressViewFrames() ([][]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.viewFrames()
}

func (p *playerActor) equipNeiGong(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.progress.equip(id); err != nil {
		return err
	}
	p.syncEquippedNeiGongBuffLocked(time.Now())
	p.syncEquippedNeiGongResourcesLocked()
	return nil
}

// syncEquippedNeiGongResourcesLocked 将已装备秘籍的资源修正值应用到
// 真正的服务器 Actor 上。可见协议仍会发送基础 MaxHP/MaxMP 加原始的
// *Add 字段，但所有服务器端的消耗、伤害和恢复都使用 Actor.MaxHP/
// MaxMP（即推导后的总上限）。当前现代客户端直接按 MaxMP + MaxMPAdd
// 刷新 MP；线上值没有乘数。
//
// 调用方必须已持有 p.mu。
func (p *playerActor) syncEquippedNeiGongResourcesLocked() {
	hpAdd, mpAddRaw := p.progress.equippedResourceAdds()
	p.vitals.SetDerivedMaxima(p.vitals.BaseMaxHP+hpAdd, p.vitals.BaseMaxMP+mpAddRaw)
}

// syncEquippedNeiGongBuffLocked 将已装备秘籍由已安装 varprop 推导出的
// 确切效果镜像到一个专用原生 BufferInfo 槽位。这刻意不用服务器发起的
// 0x1E 自定义消息：FxGameLogic 通过普通对象属性消费 BufferInfo 和
// BufferListStr。
func (p *playerActor) syncEquippedNeiGongBuffLocked(now time.Time) {
	book := p.progress.book(p.progress.curNeiGong)
	if book == nil || book.buffID == "" || book.buffStaticData == 0 || book.buffLevel <= 0 {
		delete(p.buffs, innerPowerBuffSlot)
		return
	}
	p.buffs[innerPowerBuffSlot] = activePlayerBuff{
		staticData: book.buffStaticData,
		expiresUTC: now.UTC().Add(innerPowerBuffLease),
		configID:   book.buffID,
		level:      book.buffLevel,
	}
}

func (p *playerActor) facultyReady(id string) (uint16, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.facultyReady(id)
}

func (p *playerActor) facultyBegin(style int32) (uint16, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.progress.facultyBegin(style, time.Now()); err != nil {
		return 0, false, err
	}
	// 普通内修是一种持久的后台转换，不得强制角色进入 LS_FACULTY。只有
	// 主动的演武路径才拥有那个可见的逻辑/动作状态。
	if style == facultyStyleAct {
		p.vitals.SetLogicState(logicStateFaculty)
	} else {
		p.vitals.SetLogicState(0)
	}
	return 0, false, nil
}

func (p *playerActor) facultyExit() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.progress.facultyExit()
	p.vitals.SetLogicState(0)
}

func (p *playerActor) progressBookFrame(slot uint16) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.bookFrame(slot)
}

func (p *playerActor) facultyTick(now time.Time) (uint16, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	slot, changed := p.progress.advanceFaculty(now)
	if changed {
		// 一次结算可能跨越一级或多级。确切的 varprop 记录已由
		// advanceFaculty 重新求值；在调用方发送常规的角色属性/View 刷新
		// 之前，先发布它的原生 BuffStore 状态，并让新的推导上限
		// 成为权威值。
		p.syncEquippedNeiGongBuffLocked(now)
		p.syncEquippedNeiGongResourcesLocked()
	}
	return slot, changed
}

func (p *playerActor) facultySnapshot() facultyProgressSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.snapshot()
}

func (p *playerActor) restoreFaculty(value facultyProgressSnapshot, now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.progress.restore(value, now)
	if p.progress.facultyState == facultyStateConvert && p.progress.facultyStyle == facultyStyleAct {
		p.vitals.SetLogicState(logicStateFaculty)
	} else {
		p.vitals.SetLogicState(0)
	}
	p.syncEquippedNeiGongBuffLocked(now)
	p.syncEquippedNeiGongResourcesLocked()
}

// activateQingGong 为本场景会话记住一条服务器批准的 ActiveQGSkillRec
// 条目。客户端在启用已装备的 qinggong_gaojie（高级轻功）技能之前会
// 检查这张表。
func (p *playerActor) activateQingGong(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.activeQingGong[id]; exists {
		return false
	}
	p.activeQingGong[id] = struct{}{}
	return true
}

func (p *playerActor) stateProperties() []clientdata.IndexedProperty {
	p.mu.Lock()
	defer p.mu.Unlock()
	state := p.snapshotLocked()
	now := time.Now()
	hpRecovery := sitcrossRecoveryPerSecond(state.MaxHP)
	mpRecovery := sitcrossRecoveryPerSecond(state.MaxMP)
	currentSkillID, currentSkillEffectID, currentSkillLevel, currentSkillTarget := p.currentSkillID, p.currentSkillEffectID, p.currentSkillLevel, p.currentSkillTarget
	hpRatio := int32(int64(state.HP) * 100 / int64(state.MaxHP))
	hitHPRatio := int32(int64(state.HitHP) * 100 / int64(state.MaxHP))
	mpRatio := int32(int64(state.MP) * 100 / int64(state.MaxMP))
	properties := []clientdata.IndexedProperty{
		{Index: 21, Name: "LogicState", Value: clientdata.ByteValue(state.LogicState)},
		{Index: 28, Name: "HPRatio", Value: clientdata.Int32Value(hpRatio)},
		{Index: 29, Name: "MPRatio", Value: clientdata.Int32Value(mpRatio)},
		// 客户端用这些基础值加上 MaxHPAdd/MaxMPAdd 推导显示的总量。
		// Actor.Max* 仍是服务器侧的总上限。
		{Index: 30, Name: "MaxHP", Value: clientdata.Int32Value(state.BaseMaxHP)},
		{Index: 31, Name: "MaxMP", Value: clientdata.Int32Value(state.BaseMaxMP)},
		{Index: 32, Name: "HP", Value: clientdata.Int32Value(state.HP)},
		{Index: 33, Name: "MP", Value: clientdata.Int32Value(state.MP)},
		{Index: 26, Name: "SpeedRatio", Value: clientdata.Int32Value(state.SpeedRatio)},
		{Index: 112, Name: "HitHP", Value: clientdata.Int32Value(state.HitHP)},
		{Index: 113, Name: "HitHPRatio", Value: clientdata.Int32Value(hitHPRatio)},
		{Index: 114, Name: "QingGongPoint", Value: clientdata.Int32Value(state.QingGongPoint)},
		{Index: 115, Name: "MaxQingGongPoint", Value: clientdata.Int32Value(state.MaxQingGongPoint)},
		{Index: 116, Name: "MaxQingGongPointAdd", Value: clientdata.Int32Value(state.MaxQingGongPointAdd)},
		{Index: propNeigongPKStatus, Name: "NeigongPKStatus", Value: clientdata.Int32Value(0)},
		{Index: propDead, Name: "Dead", Value: clientdata.Int32Value(boolInt32(state.HP <= 0))},
		{Index: propCantUseSkill, Name: "CantUseSkill", Value: clientdata.Int32Value(boolInt32(p.skillLockUntil.After(now)))},
		{Index: propCurSkillID, Name: "CurSkillID", Value: clientdata.StringValue(currentSkillID)},
		{Index: propCurSkillEffectID, Name: "CurSkillEffectID", Value: clientdata.StringValue(currentSkillEffectID)},
		{Index: propCurSkillLevel, Name: "CurSkillLevel", Value: clientdata.Int32Value(currentSkillLevel)},
		{Index: propCurSkillTarget, Name: "CurSkillTarget", Value: clientdata.Value{Type: clientdata.WireObject, U64: currentSkillTarget}},
		// 本地回环绝不应因某个属性缺失（为零）而无法通过客户端的技能/
		// 动作同步容差检查。
		{Index: propModifySkillLockTime, Name: "ModifySkillLockTime", Value: clientdata.Int32Value(5000)},
		{Index: propHPHeartSpeed, Name: "HPHeartSpeed", Value: clientdata.Int32Value(sitcrossHeartMillis)},
		{Index: propHPHeartSpeedAdd, Name: "HPHeartSpeedAdd", Value: clientdata.Int32Value(0)},
		{Index: propMPHeartSpeed, Name: "MPHeartSpeed", Value: clientdata.Int32Value(sitcrossHeartMillis)},
		{Index: propMPHeartSpeedAdd, Name: "MPHeartSpeedAdd", Value: clientdata.Int32Value(0)},
		{Index: propHPUpSpeed, Name: "HPUpSpeed", Value: clientdata.Int32Value(hpRecovery)},
		{Index: propHPUpSpeedAdd, Name: "HPUpSpeedAdd", Value: clientdata.Int32Value(0)},
		{Index: propMPUpSpeed, Name: "MPUpSpeed", Value: clientdata.Int32Value(mpRecovery)},
		{Index: propMPUpSpeedAdd, Name: "MPUpSpeedAdd", Value: clientdata.Int32Value(0)},
		{Index: 36, Name: "Name", Value: clientdata.WideStringValue(state.Name)},
		{Index: 111, Name: "CapitalType1", Value: clientdata.Int32Value(p.silver)},
		{Index: bufferListPropertyIndex, Name: "BufferListStr", Value: clientdata.StringValue(p.bufferListStringLocked(now))},
	}
	properties = append(properties, factionProperties(p.faction)...)
	for _, slot := range nativeBuffSlots {
		properties = append(properties, clientdata.IndexedProperty{
			Index: bufferInfoPropertyIndex(slot), Name: bufferInfoPropertyName(slot), Value: clientdata.StringValue(p.bufferInfoLocked(slot, now)),
		})
	}
	properties = append(properties, baseMotionProperties(p.motion)...)
	properties = append(properties, qinggongMotionProperties()...)
	return append(properties, p.progress.properties()...)
}

func boolInt32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

// baseMotionProperties 提供 FxGameLogic 在创建普通跳跃动作时读取的
// 玩家物理参数。客户端对 Gravity 没有服务器侧的兜底来源：只有当这个
// 协商属性存在时它才会应用该值。9.8 是当前客户端使用的
// 标准跳跃重力。
func baseMotionProperties(motion playerMotion) []clientdata.IndexedProperty {
	return []clientdata.IndexedProperty{
		{Index: 168, Name: "Gravity", Value: clientdata.Float32Value(motion.gravity)},
		{Index: 169, Name: "GravityAdd", Value: clientdata.Float32Value(0)},
		// 现代 FxGameLogic 的 FallDownManager 会从本地玩家对象查询
		// DropHeightPub。它的兜底值只有 5.0 个世界单位；这里刻意使用
		// 偏高的本地测试安全阈值，让普通的地形降落退出回着地动作，而非
		// 进入倒地动作。 TODO: 后面改GM参数控制，默认 5.0
		{Index: 170, Name: "DropHeightPub", Value: clientdata.Float32Value(100000)},
	}
}

func (p *playerActor) motionSnapshot() playerMotion {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.motion
}

// setMoveSpeed 和 setGravity 返回单个常规 0x10 更新档案。MoveSpeed 与
// RunSpeed 一并修改，使 GM 控制对行走和奔跑只有一个明确的含义。
func (p *playerActor) setMoveSpeed(value float32) ([]byte, error) {
	p.mu.Lock()
	p.motion.moveSpeed = value
	p.motion.runSpeed = value
	motion := p.motion
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 23, Name: "MoveSpeed", Value: clientdata.Float32Value(motion.moveSpeed)},
		{Index: 25, Name: "RunSpeed", Value: clientdata.Float32Value(motion.runSpeed)},
	})
}

func (p *playerActor) setGravity(value float32) ([]byte, error) {
	p.mu.Lock()
	p.motion.gravity = value
	motion := p.motion
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 168, Name: "Gravity", Value: clientdata.Float32Value(motion.gravity)},
	})
}

// activateRedSuperArmor 开启一段权威的红色霸体租约。其返回值是供
// FxGameLogic 的 BuffStore 设置器消费的原生 BufferInfo1 载荷：
// StaticData,OwnerID,OwnerSerial,Level,EndTimeMs,Count（均为十六进制）。
func (p *playerActor) activateRedSuperArmor(now time.Time) (string, time.Time) {
	return p.activateBuff(redSuperArmorBuffSlot, redSuperArmorStaticData, 20*time.Second, now)
}

func (p *playerActor) activateBuff(slot uint16, staticData uint32, lifetime time.Duration, now time.Time) (string, time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	expires := now.UTC().Add(lifetime)
	p.buffs[slot] = activePlayerBuff{staticData: staticData, expiresUTC: expires, level: 1}
	return p.bufferInfoLocked(slot, now), expires
}

func (p *playerActor) activateNamedBuff(slot uint16, staticData uint32, configID string, lifetime time.Duration, now time.Time) (string, time.Time) {
	return p.activateNamedBuffAtLevel(slot, staticData, configID, 1, lifetime, now)
}

func (p *playerActor) activateNamedBuffAtLevel(slot uint16, staticData uint32, configID string, level int32, lifetime time.Duration, now time.Time) (string, time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if level < 1 {
		level = 1
	}
	expires := now.UTC().Add(lifetime)
	p.buffs[slot] = activePlayerBuff{
		staticData: staticData,
		expiresUTC: expires,
		configID:   configID,
		level:      level,
	}
	return p.bufferInfoLocked(slot, now), expires
}

// activateSkillBuff 将由资源编译出的技能 Buff 投影到原生 BufferInfo1..24
// 存储中。系统租约保留 1..10 槽位；常规 Buff 复用其现有 ConfigID 槽位，
// 否则从 11..24 分配。
func (p *playerActor) activateSkillBuff(definition skillBuffDefinition, now time.Time) (uint16, string, time.Time, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	slot := definition.preferredSlot
	if slot == 0 {
		for candidate, active := range p.buffs {
			if active.configID == definition.configID {
				slot = candidate
				break
			}
		}
	}
	if slot == 0 {
		for candidate := yihuaPassiveBuffSlot + 1; candidate <= bufferInfoSlotCount; candidate++ {
			active, occupied := p.buffs[candidate]
			if !occupied || !active.expiresUTC.After(now.UTC()) {
				slot = candidate
				break
			}
		}
	}
	if slot == 0 || slot > bufferInfoSlotCount {
		return 0, "", time.Time{}, false
	}
	lifetime := definition.lifetime
	if lifetime <= 0 {
		lifetime = time.Millisecond
	}
	expires := now.UTC().Add(lifetime)
	level := definition.level
	if level < 1 {
		level = 1
	}
	p.buffs[slot] = activePlayerBuff{
		staticData: definition.staticData,
		expiresUTC: expires,
		configID:   definition.configID,
		level:      level,
	}
	return slot, p.bufferInfoLocked(slot, now), expires, true
}

func (p *playerActor) nextSkillBuffVariant(variants []skillBuffDefinition, now time.Time) (skillBuffDefinition, bool) {
	if len(variants) == 0 {
		return skillBuffDefinition{}, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	slot := variants[0].preferredSlot
	active, exists := p.buffs[slot]
	if !exists || !active.expiresUTC.After(now.UTC()) {
		return variants[0], true
	}
	for index, variant := range variants {
		if active.staticData == variant.staticData {
			return variants[(index+1)%len(variants)], true
		}
	}
	return variants[0], true
}

func (p *playerActor) bufferInfo(slot uint16, now time.Time) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.bufferInfoLocked(slot, now)
}

func (p *playerActor) bufferInfoLocked(slot uint16, now time.Time) string {
	buff, ok := p.buffs[slot]
	if !ok || !buff.expiresUTC.After(now.UTC()) {
		return ""
	}
	level := buff.level
	if level < 1 {
		level = 1
	}
	return fmt.Sprintf("%X,%X,%X,%X,%X,%X", buff.staticData, playerObjectID, playerOwnerID, level, buff.expiresUTC.UnixMilli(), 1)
}

// bufferListString 遵循已保留的 BufferModule::RefreshBuffersInfo 布局：
// ConfigID,UniqueID,Level,OwnerIdent-OwnerSerial,EndTime,EffectType,
// OverlayRows|。它与 BufferInfo 刻意由同一批服务器权威租约生成，因此
// 玩法与视觉的 Buff 状态不会分叉。
func (p *playerActor) bufferListString(now time.Time) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.bufferListStringLocked(now)
}

func (p *playerActor) bufferListStringLocked(now time.Time) string {
	var list strings.Builder
	for _, slot := range nativeBuffSlots {
		buff, ok := p.buffs[slot]
		if !ok || !buff.expiresUTC.After(now.UTC()) {
			continue
		}
		configID := buff.configID
		if configID == "" {
			configID, ok = activeBuffConfigID(buff.staticData)
		}
		if !ok {
			continue
		}
		level := buff.level
		if level < 1 {
			level = 1
		}
		fmt.Fprintf(&list, "%s,local-%d,%d,%d-%d,%d,1,0|", configID, slot, level, playerObjectID, playerOwnerID, buff.expiresUTC.UnixMilli())
	}
	return list.String()
}

func activeBuffConfigID(staticData uint32) (string, bool) {
	switch staticData {
	case redSuperArmorStaticData:
		return "buf_fixed_red", true
	case waterDriftStaticData:
		return "buf_qg_drift", true
	case waterJumpStaticData:
		return "buf_qg_dst", true
	default:
		return "", false
	}
}

func (p *playerActor) hasActiveBuff(slot uint16, staticData uint32, now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[slot]
	return ok && buff.staticData == staticData && buff.expiresUTC.After(now.UTC())
}

func (p *playerActor) clearBuff(slot uint16, expectedStaticData uint32) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[slot]
	if !ok || buff.staticData != expectedStaticData {
		return "", false
	}
	delete(p.buffs, slot)
	return "", true
}

// expireBuff 只清除创建此定时器的那份确切租约。后续的刷新绝不会被一个
// 已过期的定时器移除。
func (p *playerActor) expireBuff(slot uint16, expectedStaticData uint32, expected time.Time, now time.Time) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[slot]
	if !ok || buff.staticData != expectedStaticData || !buff.expiresUTC.Equal(expected) || buff.expiresUTC.After(now.UTC()) {
		return "", false
	}
	delete(p.buffs, slot)
	return "", true
}

func (p *playerActor) buffRules(now time.Time) playerBuffRules {
	red := p.hasActiveBuff(redSuperArmorBuffSlot, redSuperArmorStaticData, now)
	return playerBuffRules{CantHitEffect: red, CantBeSkillLocked: red}
}

// qinggongMotionProperties 的取值来源是当前已安装的
// share/Skill/QingGong/QGDefine.ini。客户端不会从 216 动作请求推导
// 这些值：它在解析动作之前先从自己的玩家对象读取。请保持基础属性与
// Add 属性分离，因为 Lua 和 FxGameLogic 都会成对查询两者。
func qinggongMotionProperties() []clientdata.IndexedProperty {
	const (
		waterSpeed      float32 = 7.5 // qinggong_2 Para1
		waterJumpSpeed  float32 = 6.0 // qinggong_3 Para1
		secondJumpSpeed float32 = 6.0 // qinggong_4 Para1
		thirdJumpSpeed  float32 = 6.0 // qinggong_5 Para1
		airRushSpeed    float32 = 15  // qinggong_6 Para1
		airRushDist     float32 = 12  // qinggong_6 Para2
		// 实机测试确认类型 7 的 Para 顺序是先速度、后距离：Para1=25 产生
		// 预期的快速雁行功冲刺，而 Para2=6.5 是它的行进距离。这与类型 6
		// 的空中冲刺参数顺序不同。
		landRushSpeed float32 = 25  // qinggong_7 Para1
		landRushDist  float32 = 6.5 // qinggong_7 Para2
		climbSpeed    float32 = 7.5 // qinggong_11 Para1
		// 这些并不是上面三个动作速度值的别名。FxGameLogic 在冲刺动作的
		// 门控中使用 share/Skill/rush_range.ini，包括 qinggong_3 / 点水跳。
		// 在这里保留历史上以速度冒充距离的占位值，会让那个原生门控按
		// 错误的水上距离求值。
		groundRushRange    float32 = 5.0 // rush_range.ini GRushRange
		airRushRange       float32 = 8.0 // rush_range.ini ARushRange
		waterRushRange     float32 = 8.0 // rush_range.ini WRushRange
		groundRushRangeAdd float32 = 1.5 // rush_range.ini GRushRangeAdd
		airRushRangeAdd    float32 = 6.5 // rush_range.ini ARushRangeAdd
		waterRushRangeAdd  float32 = 6.5 // rush_range.ini WRushRangeAdd
	)
	f := func(index uint16, name string, value float32) clientdata.IndexedProperty {
		return clientdata.IndexedProperty{Index: index, Name: name, Value: clientdata.Float32Value(value)}
	}
	return []clientdata.IndexedProperty{
		f(117, "RunSpeedAdd", 0), f(118, "JumpSpeedAdd", 0),
		f(119, "DriftSpeed", waterSpeed), f(120, "DriftSpeedAdd", 0),
		f(121, "DriftJumpSpeed", waterJumpSpeed), f(122, "DriftJumpSpeedAdd", 0),
		f(123, "SndJumpSpeed", secondJumpSpeed), f(124, "SndJumpSpeedAdd", 0),
		f(125, "ThdJumpSpeed", thirdJumpSpeed), f(126, "ThdJumpSpeedAdd", 0),
		f(127, "AirRushSpeed", airRushSpeed), f(128, "AirRushSpeedAdd", 0),
		f(129, "AirRushDist", airRushDist), f(130, "AirRushDistAdd", 0),
		f(131, "LandRushSpeed", landRushSpeed), f(132, "LandRushSpeedAdd", 0),
		f(133, "LandRushDist", landRushDist), f(134, "LandRushDistAdd", 0),
		f(135, "ClimbSpeed", climbSpeed), f(136, "ClimbSpeedAdd", 0),
		f(137, "GRushRange", groundRushRange), f(138, "GRushRangeAdd", groundRushRangeAdd),
		f(139, "ARushRange", airRushRange), f(140, "ARushRangeAdd", airRushRangeAdd),
		f(141, "WRushRange", waterRushRange), f(142, "WRushRangeAdd", waterRushRangeAdd),
	}
}

func (p *playerActor) vitalUpdate() ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, p.stateProperties())
}

// sitcrossTick 只推进服务器持有的 HP/MP 状态。调用方会发送生成的常规
// 0x10 档案，因此客户端绝不会自行凭 HPHeartSpeed/MPHeartSpeed 虚构
// 资源值。
func (p *playerActor) sitcrossTick() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.vitals.RestoreWhileSit(sitcrossRecoveryPerSecond(p.vitals.MaxHP), sitcrossRecoveryPerSecond(p.vitals.MaxMP))
}

// sitcrossRecoveryPerSecond 向上取整，使无法整除的推导上限仍能在至多
// 十次一秒结算内回满。
func sitcrossRecoveryPerSecond(maximum int32) int32 {
	if maximum <= 0 {
		return 0
	}
	return (maximum + sitcrossFullRecoverySeconds - 1) / sitcrossFullRecoverySeconds
}

// combatBruiseTick 在角色仍处于战斗中时结算次要的 HitHP 层。常规的
// 属性档案更新已包含 HitHP 和 HitHPRatio，因此这里沿用已被验证的常规
// 0x10 属性路由。
func (p *playerActor) combatBruiseTick() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.vitals.RecoverBruisedHPWhileFighting(combatBruiseRecoveryPerSecond)
}

// combatRageTick 在角色仍处于战斗中时结算固定的每秒 +5 怒气恢复。任何
// 进入战斗的转换——使用技能或残留的 NPC 仇恨——都会让角色保持资格；
// 打坐和死亡则永远不算。常规的属性档案更新已包含 SP，因此这里与其他
// tick 层一样，沿用同一条已被验证的常规 0x10 属性路由。
func (p *playerActor) combatRageTick() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.vitals.Logic != entity.LogicStateFighting {
		return false
	}
	before := p.progress.attributes.sp
	p.progress.attributes.sp = clampPlayerSP(before + combatRageRecoveryPerSecond)
	return p.progress.attributes.sp != before
}

// loginVitalUpdate 是登录/切图链的首个权威 0x10，也是 MaxSP 唯一的
// 下发点：真机与新版服务端全量帧实证，怒气上限只在登录全量批出现一次
//（61653 抓包全场 3964 帧仅 #88 一处，值 80），之后的任何恢复批/事件批
// 都不再携带，MaxSPAdd 则从不发送。把上限塞进 0x0D 归档或周期批会让
// 客户端在进世界时崩溃。
func (p *playerActor) loginVitalUpdate() ([]byte, error) {
	properties := p.stateProperties()
	properties = append(properties, clientdata.IndexedProperty{
		Index: propMaxSP, Name: "MaxSP", Value: clientdata.Int32Value(maxPlayerSP),
	})
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, properties)
}

// setCombatPresence 负责本地敌对 AI 的玩家侧生命周期。任何残留的 NPC
// 仇恨都会在首次受击结算之前打断打坐；只有当每个仇恨都已清除后才
// 恢复常规状态。这使恢复、伤害和客户端 LogicState 保持在同一个权威
// 转换之上。
func (p *playerActor) setCombatPresence(active bool) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.vitals.HP <= 0 {
		return false
	}
	if active {
		if p.vitals.Logic == entity.LogicStateFighting {
			return false
		}
		p.vitals.SetLogicState(entity.LogicStateFighting)
		return true
	}
	if p.vitals.Logic != entity.LogicStateFighting {
		return false
	}
	p.vitals.SetLogicState(entity.LogicStateNormal)
	return true
}

func (p *playerActor) currentSkillSnapshot() string {
	id, _, _, _ := p.currentSkillStateSnapshot()
	return id
}

func (p *playerActor) currentSkillStateSnapshot() (string, string, int32, uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentSkillID, p.currentSkillEffectID, p.currentSkillLevel, p.currentSkillTarget
}

// clearCurrentSkill 只清除调度该定时器的那一次确切技能调用。后续的调用
// 绝不能被更早的定时器取消。
func (p *playerActor) clearCurrentSkill(expected string) bool {
	p.mu.Lock()
	if p.currentSkillID != expected {
		p.mu.Unlock()
		return false
	}
	p.currentSkillID = ""
	p.currentSkillEffectID = ""
	p.currentSkillLevel = 0
	p.currentSkillTarget = 0
	// 技能清除定时器在独立 goroutine 触发；状态字段与 LogicState
	// 必须在同一临界区内落定，这与锁纪律注释一致。
	p.vitals.SetLogicState(0)
	p.mu.Unlock()
	return true
}

func (p *playerActor) currentSkillClearUpdate() ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 21, Name: "LogicState", Value: clientdata.ByteValue(0)},
		{Index: propCurSkillID, Name: "CurSkillID", Value: clientdata.StringValue("")},
		{Index: propCurSkillEffectID, Name: "CurSkillEffectID", Value: clientdata.StringValue("")},
		{Index: propCurSkillLevel, Name: "CurSkillLevel", Value: clientdata.Int32Value(0)},
		{Index: propCurSkillTarget, Name: "CurSkillTarget", Value: clientdata.ObjectValue(0, 0)},
	})
}

// facultyNameResetFrame 只清除客户端侧的 FacultyName 属性。打开
// form_wuxue_faculty 发生在 1006/1 选择响应之后，因此它的 FacultyName
// 绑定器可能错过最初的值变更。1006/24 刷新会在常规的属性档案更新之前
// 立即发送这个清空帧；随后恢复真实名称即可可靠地执行
// set_control_state，既不改变服务器持有的选择，也不假装演武已经开始。
func (p *playerActor) facultyNameResetFrame() ([]byte, bool, error) {
	p.mu.Lock()
	hasFacultyName := p.progress.facultyName != ""
	p.mu.Unlock()
	if !hasFacultyName {
		return nil, false, nil
	}
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propFacultyName, Name: "FacultyName", Value: clientdata.StringValue("")},
	})
	return frame, true, err
}

func (p *playerActor) setSilver(value int32) {
	p.mu.Lock()
	p.silver = value
	p.mu.Unlock()
}

func clampPlayerSP(value int32) int32 {
	if value < 0 {
		return 0
	}
	if value > maxPlayerSP {
		return maxPlayerSP
	}
	return value
}

// setSP 修改服务器权威的怒气值，并立即返回主角 UI 消费的常规 0x10
// 属性更新。
func (p *playerActor) setSP(value int32) (int32, []byte, error) {
	p.mu.Lock()
	p.progress.attributes.sp = clampPlayerSP(value)
	sp := p.progress.attributes.sp
	p.mu.Unlock()
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propSP, Name: "SP", Value: clientdata.Int32Value(sp)},
	})
	return sp, frame, err
}

// addSP 是战斗/技能怒气奖励以及 GM 调整未来唯一的入口。在这里做饱和
// 处理可保证 211 的消耗检查读到的正是客户端刚显示过的值。
func (p *playerActor) addSP(delta int32) (int32, []byte, error) {
	p.mu.Lock()
	next := int64(p.progress.attributes.sp) + int64(delta)
	if next < 0 {
		next = 0
	}
	if next > int64(maxPlayerSP) {
		next = int64(maxPlayerSP)
	}
	p.progress.attributes.sp = int32(next)
	sp := p.progress.attributes.sp
	p.mu.Unlock()
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propSP, Name: "SP", Value: clientdata.Int32Value(sp)},
	})
	return sp, frame, err
}

func (p *playerActor) addSilver(amount int32) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if amount > 0 && p.silver > math.MaxInt32-amount {
		p.silver = math.MaxInt32
	} else {
		p.silver += amount
	}
	return p.silver
}

func (p *playerActor) silverUpdate() ([]byte, error) {
	p.mu.Lock()
	silver := p.silver
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 111, Name: "CapitalType1", Value: clientdata.Int32Value(silver)},
	})
}

func (p *playerActor) addObject(location role.Position, visual roleVisual) []byte {
	// 本机视角的玩家对象恒为 1-1：stage_main 开图与 0x11 记录绑定
	// 都依赖这一身份。
	return p.addObjectAs(playerObjectID, playerOwnerID, location, visual)
}

// addObjectAs 以给定的线上对象身份生成 0x0D 玩家档案。远程玩家
// 的物化走本入口：帧头换成其在场景内的 wire objectID，帧体仍是
// 该玩家（p 自身）的实时状态快照。
func (p *playerActor) addObjectAs(objectID, ownerID uint32, location role.Position, visual roleVisual) []byte {
	p.mu.Lock()
	state := p.snapshotLocked()
	state.ObjectID, state.OwnerID = objectID, ownerID
	motion := p.motion
	now := time.Now()
	msg := make([]byte, 0x3f, 0x120)
	msg[0] = 0x0D
	binary.LittleEndian.PutUint32(msg[1:], state.ObjectID)
	binary.LittleEndian.PutUint32(msg[5:], state.OwnerID)
	binary.LittleEndian.PutUint32(msg[0x09:], math.Float32bits(location.X))
	binary.LittleEndian.PutUint32(msg[0x0D:], math.Float32bits(location.Y))
	binary.LittleEndian.PutUint32(msg[0x11:], math.Float32bits(location.Z))
	binary.LittleEndian.PutUint32(msg[0x15:], math.Float32bits(location.Orient))

	count := uint16(0)
	addByte := func(index uint16, value byte) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = append(msg, value)
		count++
	}
	addInt := func(index uint16, value int32) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint32(msg, uint32(value))
		count++
	}
	addFloat := func(index uint16, value float32) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint32(msg, math.Float32bits(value))
		count++
	}
	addString := func(index uint16, value string) { msg = appendStringProperty(msg, index, value); count++ }
	addObject := func(index uint16, value uint64) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint64(msg, value)
		count++
	}

	addByte(0, 2) // Type = TYPE_PLAYER
	addByte(1, visual.sex)
	addString(2, visual.photo)
	addInt(3, 0) // Camp
	addInt(4, 0) // Race
	addInt(5, 0) // Job
	addInt(6, 1) // Level
	addFloat(9, location.X)
	addFloat(10, location.Y)
	addFloat(11, location.Z)
	addFloat(12, location.Orient)
	addString(13, visual.hair)
	addString(14, visual.face)
	addString(15, visual.cloth)
	addString(16, visual.pants)
	addString(17, visual.shoes)
	addString(18, visual.actionSet)
	addString(19, "stand")
	addInt(20, 0)
	addByte(21, state.LogicState)
	addFloat(22, 8.0)
	addFloat(23, motion.moveSpeed)
	addFloat(24, 7.5)
	addFloat(25, motion.runSpeed)
	addInt(26, state.SpeedRatio)
	addByte(27, 0)
	addInt(28, int32(int64(state.HP)*100/int64(state.MaxHP)))
	addInt(29, int32(int64(state.MP)*100/int64(state.MaxMP)))
	addInt(30, state.BaseMaxHP)
	addInt(31, state.BaseMaxMP)
	addInt(32, state.HP)
	addInt(33, state.MP)
	addInt(112, state.HitHP)
	addInt(113, int32(int64(state.HitHP)*100/int64(state.MaxHP)))
	addInt(114, state.QingGongPoint)
	addInt(115, state.MaxQingGongPoint)
	addInt(116, state.MaxQingGongPointAdd)
	addInt(propNeigongPKStatus, 0)
	addInt(propDead, boolInt32(state.HP <= 0))
	addInt(propCantUseSkill, boolInt32(p.skillLockUntil.After(now)))
	addString(propCurSkillID, p.currentSkillID)
	addString(propCurSkillEffectID, p.currentSkillEffectID)
	addInt(propCurSkillLevel, p.currentSkillLevel)
	addObject(propCurSkillTarget, p.currentSkillTarget)
	addInt(propModifySkillLockTime, 5000)
	addString(bufferListPropertyIndex, p.bufferListStringLocked(now))
	for _, property := range baseMotionProperties(motion) {
		addFloat(property.Index, property.Value.F32)
	}
	for _, property := range qinggongMotionProperties() {
		addFloat(property.Index, property.Value.F32)
	}
	addInt(111, p.silver) // CapitalType1: 碎银；已在场景属性表中协商。
	for _, slot := range nativeBuffSlots {
		addString(bufferInfoPropertyIndex(slot), p.bufferInfoLocked(slot, now))
	}
	for _, property := range p.progress.properties() {
		msg = appendNPCProperty(msg, property)
		count++
	}
	addString(35, visual.hair)
	msg = appendWideStringProperty(msg, 36, state.Name)
	count++
	msg = appendObjectProperty(msg, 100, 0, 0)
	count++
	binary.LittleEndian.PutUint16(msg[0x3d:], count)
	p.mu.Unlock()
	return msg
}

func sendPlayerSpawn(conn sceneMessageConnection, player *playerActor, location role.Position, visual roleVisual) error {
	// 在新建的玩家对象中保留仍处于活动状态的原生 BuffStore 状态。
	// BufferInfo 由客户端的 C++ 属性绑定器处理。
	log.Printf("native Buff spawn player=%d-%d red=%q drift=%q jump=%q neigong=%q", playerObjectID, playerOwnerID, player.bufferInfo(redSuperArmorBuffSlot, time.Now()), player.bufferInfo(waterDriftBuffSlot, time.Now()), player.bufferInfo(waterJumpBuffSlot, time.Now()), player.bufferInfo(innerPowerBuffSlot, time.Now()))
	if err := sendPlayerAddObject(conn, player, location, visual); err != nil {
		return err
	}
	return sendPlayerLocationAndVitals(conn, player, location)
}

// sendPlayerAddObject 创建原生玩家对象，但不发出单独的 ServerLocation
// 移动更新。现场场景替换需要这个拆分：stage_main 要求先收到 1-1 才
// 会打开目标场景，而 0x1F 视觉重定位必须等到目标 ClientReady 之后。
func sendPlayerAddObject(conn sceneMessageConnection, player *playerActor, location role.Position, visual roleVisual) error {
	return conn.WriteFrame(player.addObject(location, visual))
}

func sendPlayerLocationAndVitals(conn sceneMessageConnection, player *playerActor, location role.Position) error {
	if err := conn.WriteFrame(serverLocation(playerObjectID, playerOwnerID, worldTransform(location))); err != nil {
		return err
	}
	vitals, err := player.loginVitalUpdate()
	if err != nil {
		return err
	}
	return conn.WriteFrame(vitals)
}

func worldTransform(position role.Position) worldcore.Transform {
	return worldcore.Transform{X: position.X, Y: position.Y, Z: position.Z, Orient: position.Orient}
}
