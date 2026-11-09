package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/sim"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

type combatTickResult struct {
	playerStateChanged bool
	playerRevived      bool
}

type npcMoveUpdate struct {
	id          uint32
	ownerID     uint32
	destination worldcore.Transform
	// delay 是该次移动的到达确认延时（scheduleCombatMoveArrival 用）。
	// 0＝旧默认 1s（重生/栓绳瞬移类）；追击折线按剩余路径长/速度。
	delay time.Duration
}

type npcPropertyUpdate struct {
	id         uint32
	ownerID    uint32
	properties []clientdata.IndexedProperty
}

const (
	combatTickMaxAggressors = 4
	defaultAggroRange       = float32(12)
	defaultChaseRange       = float32(30)
	defaultNPCMoveSpeed     = float32(4.5)
	defaultNPCRespawn       = 15 * time.Second
	playerAutoReviveDelay   = 8 * time.Second

	// NPC 攻击三常量（Java NpcCombatAi 逐值对齐、参考服 55866 实证）：
	// 反击近身 6m、周期 2.2s（抓包 12141 实测 ~2s 一轮）、每击实伤 1
	// （名义攻击力 20 经玩家防御折算恒 1，只出现在 msg24 c1=7 尾值）。
	// 此前 Go 用 MaxHP/50(≥10, Boss×2)/1500ms/3.5m 属本地自造，整体废弃。
	npcAttackRangeMeters = float32(6.0)
	npcAttackEvery       = 2200 * time.Millisecond
	npcAttackDamage      = int32(1)

	// baseLocalDodgeProbability 是受击裁定的基础闪避（百分点）；脚本
	// 被动（幻玉 evasion_add）在此之上叠加。
	baseLocalDodgeProbability = int32(10)
)

// newNPCCombatState 从模板属性构造敌对 NPC 的战斗决策组件。
// 模板仍然决定哪个对象是敌对的；此状态只保存运行期决策，而这些决策
// 是正式服务器绝不能下放给客户端的。
func newNPCCombatState(npc npcSpawn, home worldcore.Transform) sim.Brain {
	aggro := npc.float32Property("SpringRange")
	if aggro <= 0 {
		aggro = defaultAggroRange
	}
	chase := npc.float32Property("ChaseRange")
	if chase < aggro {
		chase = defaultChaseRange
	}
	move := npc.float32Property("RunSpeed")
	if move <= 0 {
		move = npc.float32Property("MoveSpeed")
	}
	if move <= 0 {
		move = defaultNPCMoveSpeed
	}
	return sim.Brain{
		Home: home, AggroRange: aggro, ChaseRange: chase,
		AttackRange: npcAttackRangeMeters, MoveSpeed: move, Damage: npcAttackDamage,
		AttackEvery: npcAttackEvery, RespawnAfter: npcRespawnAfter(npc),
		// NotPositive=1（官方属性表 idx 62，Int32）：creator 条目级
		// “不主动攻击”。城内 attacknpc.xml 大多带此标记（city05 219 条
		// 中 85 条，掷斧监工/背柴工人/剧情人物），野外表（scene01）全
		// 空即全主动。creator 属性后于模板列写入，两层同名列以 creator
		// 为准，与本读取一致。
		Passive: npc.int32Property("NotPositive") != 0,
	}
}

// npcRespawnAfter 取模板官方字段 RefreshTime（VisibleNPCModernV1 idx 93，
// 单位秒）作为死亡→重生间隔；worldnpc/bossnpc 实测数据该列基本为空，
// 空值/越界（<=0 或 >1 小时）一律回落默认 15s。只有 AttackNpc/BossNpc
// 挂 Brain，因此平民与任务怪（EventNpc 从不注册）天然不进入刷新循环。
func npcRespawnAfter(npc npcSpawn) time.Duration {
	if seconds := npc.int32Property("RefreshTime"); seconds > 0 && seconds <= 3600 {
		return time.Duration(seconds) * time.Second
	}
	return defaultNPCRespawn
}

func (n npcSpawn) float32Property(name string) float32 {
	return n.resolved.Properties[name].Value.F32
}

// combatTick 是场景运行时的每秒权威 NPC 模拟。它只发送当前客户端
// 已会消费的包：NPC 0x20/0x1F 移动以及普通的 0x10 属性包。
// 仇恨目标取"最近存活成员"并经 Brain.TargetMember 粘性保持，
// 直到目标死亡或被拉出栓绳距离；NotPositive（被动）NPC 例外——
// 只认 applySkillDamageToNPC 受击锁定的攻击者，从不主动索敌。
//
// 锁内结算（run.mu → p.mu，锁序与全局规则一致），帧写入在锁外
// 经各成员视图的 conn 完成（transport 写帧线程安全）。
type combatTickOutcome struct {
	members []*sceneMember
	moves   []npcMoveUpdate
	updates []npcPropertyUpdate
	changed map[uint32]bool
	// rageFrames 是受击得怒(格挡被揍 +2/直接被揍 +1)产生的 18 字节
	// SP 单帧,按受击成员收集,锁外写给属主连接。
	rageFrames map[uint32][]byte
	// attacks 是本 tick 发生的 NPC 攻击表现素材（F 盘参考服
	// writeNPCAttackFrames 三段式：msg337 预备→msg20 动作→msg24 命中）。
	// 伤害结算仍在锁内同步完成；这里只携带锁外补发表帧所需的只读快照。
	attacks []npcAttackPresentation
}

// npcAttackPresentation 是一次 NPC 攻击的表现快照。transform 已带
// 朝向目标后的 Orient；landed=false（被闪避/招架）时锁外只发预备+
// 动作两帧（Java：格挡零帧静默，Go 保留 337/20 预备表现）。victim
// 用于命中帧（obj(玩家) 用受害者本地身份 1-1）；hpRatio 是受击后
// 玩家血量百分比（0x2D 双写值）。
type npcAttackPresentation struct {
	npcID          uint32
	ownerID        uint32
	transform      worldcore.Transform
	landed         bool
	hpRatio        int32
	model          string
	playerWideName string
	npcWideName    string
	victim         *sceneMember
}

func (r *sceneRun) combatTick(now time.Time) {
	outcome := r.simulateLocked(now)
	if outcome == nil {
		return
	}
	// 锁外帧写入：受击得怒 SP 单帧、受击成员的 vitalUpdate、
	// 其远程属性帧扇出、以及 NPC 移动/属性帧扇出。
	for _, member := range outcome.members {
		if frame, has := outcome.rageFrames[member.objectID]; has && member.view != nil {
			if err := member.view.conn.WriteFrame(frame); err != nil {
				log.Printf("%s: publish rage gain: %v", member.view.remote, err)
			}
		}
		if !outcome.changed[member.objectID] {
			continue
		}
		// 战斗 tick 只推数值 vitals 小帧——全量面板（vitalUpdate）含
		// State="stand"/LogicState/CurSkillID/PauseSkill，会在冲锋受理簇
		// 之后把客户端的自动接近态洗掉（53051 会话实证：簇后 1s 全量面板
		// 到达 → CurSkillID 被清空 → 客户端不自走、220 冻结刷）。官服
		// 战斗中从不重推全量面板。
		if frame, err := member.player.combatVitalsFrame(); err == nil {
			if err := member.view.conn.WriteFrame(frame); err != nil {
				log.Printf("%s: publish combat vitals: %v", member.view.remote, err)
			}
		} else {
			log.Printf("%s: encode combat vitals: %v", member.view.remote, err)
		}
		// 远程可见者收到该玩家的战斗属性子集（掉血/死亡/状态）。
		// 连接循环驱动的状态变化（打坐/技能/复活）的远程广播
		// 属于下一轮范围。
		state := member.player.stateSnapshot()
		state.ObjectID, state.OwnerID = member.objectID, 1
		if frame, err := sceneObjectProperties(member.objectID, 1, 0, npcCombatProperties(state)); err == nil {
			for _, observer := range outcome.members {
				if observer == member || observer.view == nil || !observer.view.npcVisible(member.objectID) {
					continue
				}
				if err := observer.view.conn.WriteFrame(frame); err != nil {
					log.Printf("%s: relay combat vitals for %d: %v", observer.view.remote, member.objectID, err)
				}
			}
		}
	}
	r.broadcastNPCFrames(outcome.members, outcome.moves, outcome.updates)
	r.broadcastNPCAttacks(outcome.attacks)
}

// broadcastNPCAttacks 补发本 tick 的 NPC 攻击表现帧（Java NpcCombatAi
// 出招簇 + 抓包 12141 四连实证）：msg337 预备（NPC 坐标/朝向）→ msg20
// 动作（施法者=NPC，招式 jn_mon053_02）→ msg24 命中（c1=7，仅实际命中；
// obj 用受害者本地玩家身份 1-1，只写受害者连接）→ 0x2D 受击特效（玩家
// 受击后 HPRatio——官方血条由 0x2D 驱动，参考服不随 NPC 命中发玩家
// HP 面板）。预备与动作帧以 NPC 为中心，受害者首发后对视口内可见该
// NPC 的其他成员中继。伤害已在锁内同步结算，这里纯表现。
func (r *sceneRun) broadcastNPCAttacks(attacks []npcAttackPresentation) {
	r.autoSelectAttackerTargets(attacks)
	for _, attack := range attacks {
		if attack.victim == nil || attack.victim.view == nil {
			continue
		}
		prepFrame, err := npcCastPreparationFrame(attack.npcID, attack.ownerID,
			attack.transform.X, attack.transform.Y, attack.transform.Z, attack.transform.Orient)
		if err != nil {
			log.Printf("%s: encode NPC cast preparation %d: %v", attack.victim.view.remote, attack.npcID, err)
			continue
		}
		animFrame, err := nativeSkillActionFrameFor(attack.npcID, attack.ownerID, npcAttackSkillJN, playerObjectID, playerOwnerID)
		if err != nil {
			log.Printf("%s: encode NPC attack action %d: %v", attack.victim.view.remote, attack.npcID, err)
			continue
		}
		for _, frame := range [][]byte{prepFrame, animFrame} {
			if err := attack.victim.view.conn.WriteFrame(frame); err != nil {
				log.Printf("%s: write NPC attack frame %d: %v", attack.victim.view.remote, attack.npcID, err)
			}
		}
		r.relayCombatFrame(attack.npcID, prepFrame, attack.victim.view, "NPC attack")
		r.relayCombatFrame(attack.npcID, animFrame, attack.victim.view, "NPC attack")
		if !attack.landed {
			continue
		}
		hitFrame, err := npcAttackHitFrame(playerObjectID, playerOwnerID,
			npcAttackDamage, attack.npcID, attack.ownerID, attack.model, attack.playerWideName, attack.npcWideName)
		if err != nil {
			log.Printf("%s: encode NPC attack hit %d: %v", attack.victim.view.remote, attack.npcID, err)
			continue
		}
		effectFrame := hurtEffectFrame(playerObjectID, playerOwnerID, attack.hpRatio)
		for _, frame := range [][]byte{hitFrame, effectFrame} {
			if err := attack.victim.view.conn.WriteFrame(frame); err != nil {
				log.Printf("%s: write NPC attack hit %d: %v", attack.victim.view.remote, attack.npcID, err)
			}
		}
	}
}

// autoSelectAttackerTargets 在 NPC 出手时给没有战斗目标的受害者自动
// 选中攻击者：LastObject(414)+AutoSelectObject(415) 双帧 + 服务端
// selectedObject 镜像，与 0x07/431 手动选中（objectRequest/
// handleSelectTargetCustom）走同一条 writePlayerTargetLocked 路径，
// 使 211 技能选靶（skillTargetSelected/SelectedArea）无需玩家先点击
// 即可反击。仅在受害者当前没有可用战斗目标时接管——玩家手动选中
// 的目标绝不被覆盖；陈旧格（目标已死/非战斗交互）与空选同权，可被
// 攻击者顶替。官方服无此行为（55866 实证 414/415 全为点击回声，
// 挨打事件 ±8 帧 0 次自发推送），这是本服的便利性扩展。
func (r *sceneRun) autoSelectAttackerTargets(attacks []npcAttackPresentation) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	for _, attack := range attacks {
		if attack.victim == nil || attack.victim.view == nil {
			continue
		}
		view := attack.victim.view
		if _, _, alive := view.livingCombatSelectionLocked(); alive {
			continue
		}
		id := worldcore.EntityID(attack.npcID)
		meta, exists := r.npcMeta(id)
		if !exists || meta.ownerID != attack.ownerID || meta.interaction != npcInteractionCombat {
			continue
		}
		if vitals, ok := r.combat.Vitals(id); !ok || vitals.HP <= 0 {
			continue
		}
		if err := view.writePlayerTargetLocked(attack.npcID, attack.ownerID); err != nil {
			log.Printf("%s: auto-select attacker id=%d: %v", view.remote, attack.npcID, err)
			continue
		}
		view.selectedObject = attack.npcID
		view.selectedOwner = attack.ownerID
		log.Printf("%s: auto-selected attacker id=%d owner=%d (victim had no living target)",
			view.remote, attack.npcID, attack.ownerID)
	}
}

// npcChaseDisabled 进程级追击开关（true=关）。初值取启动环境
// NINEYIN_NPC_CHASE=off（部署方言，起服时读一次），GM 面板
// set_npc_chase 运行期覆写（重启回落 env 初值）。
var npcChaseDisabled atomic.Bool

func init() {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("NINEYIN_NPC_CHASE")), "off") {
		npcChaseDisabled.Store(true)
	}
}

// npcChaseEnabled 追击位移总闸（默认开）：关闭后怪坐标恒停出生点——
// pending.geometry（211 快照）/服务端权威/客户端认知三方一致，用于
// 隔离追击 0x1F 广播节流（客户端滞后 4m+）造成的冲锋 c61 终点漂移。
// 仇恨照常持有（受击锁定/威胁表不动），栓绳回位/重生都以 Home 为
// 目的地，怪未挪窝即原地 no-op，无需另闸。
func npcChaseEnabled() bool {
	return !npcChaseDisabled.Load()
}

// setNPCChaseEnabled GM 面板运行期覆写追击开关，返回覆写后的状态。
func setNPCChaseEnabled(enabled bool) bool {
	npcChaseDisabled.Store(!enabled)
	return npcChaseEnabled()
}

// simulateLocked 是 combatTick 的锁内结算段，独立出来供测试路径
// 复用（测试只消费 NPC 帧，不写玩家 vital 帧）。
func (r *sceneRun) simulateLocked(now time.Time) *combatTickOutcome {
	r.mu.Lock()
	if r.closed || len(r.members) == 0 || r.combat.CombatCount() == 0 {
		r.mu.Unlock()
		return nil
	}
	members := r.sortedMembersLocked()
	memberByID := make(map[uint32]*sceneMember, len(members))
	targets := make(map[uint32]worldcore.Transform, len(members))
	for _, member := range members {
		memberByID[member.objectID] = member
		if !member.active || member.player == nil {
			continue
		}
		// 死亡成员不是仇恨目标；其复活由属主连接循环负责。
		if member.player.stateSnapshot().HP <= 0 {
			continue
		}
		targets[member.objectID] = worldcore.Transform{
			X: member.position.X, Y: member.position.Y, Z: member.position.Z, Orient: member.position.Orient,
		}
	}
	moves := make([]npcMoveUpdate, 0, combatTickMaxAggressors)
	updates := make([]npcPropertyUpdate, 0, 2)
	aggressors := 0
	threatened := make(map[uint32]bool, len(targets))
	var rage map[uint32][]byte
	attacks := make([]npcAttackPresentation, 0, 2)
	// eachCombatNPCFullLocked 以 EntityID 升序一次遍历交付全部组件指针；
	// 分支逻辑与单连接时代逐行等价，就地修改即生效。
	r.eachCombatNPCFullLocked(func(id worldcore.EntityID, vitals *sim.Vitals, state *sim.Brain, transform *worldcore.Transform, meta *sceneEntity) {
		actorState := vitals.ActorState(meta.id, meta.ownerID)
		if actorState.HP <= 0 {
			if npcRespawnBatchInterval() > 0 {
				// 批量刷新模式：重生归 refreshTick 周期扫描统一结算
				//（尸体 TTL 清除 + 批量复活），这里只保留尸体。
				return
			}
			if !state.RespawnAt.IsZero() && !now.Before(state.RespawnAt) {
				if _, hasLoot := r.lootBoxes[meta.id]; hasLoot {
					// 尸体带未拾掉落：重生推迟到掉落清除（拾空/超时标
					// removed 后走重注册路径），避免"掉落还在、怪已复活"。
					return
				}
				vitals.Revive()
				state.Threat = 0
				state.TargetMember = 0
				state.NextAttack = time.Time{}
				state.RespawnAt = time.Time{}
				// 追击导航状态随重生废弃（Home 位置的折线对新生命周期无意义）。
				meta.chase = nil
				if err := r.setNPCTransformLocked(id, state.Home); err != nil {
					log.Printf("scene %s/%s: respawn transform id=%d: %v", r.key.config, r.key.resource, meta.id, err)
					return
				}
				moves = append(moves, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, destination: state.Home})
				updates = append(updates, npcPropertyUpdate{id: meta.id, ownerID: meta.ownerID, properties: npcCombatProperties(vitals.ActorState(meta.id, meta.ownerID))})
			}
			return
		}

		if state.Threat == 0 || state.TargetMember == 0 {
			// NotPositive（被动）NPC 只认受击路径锁定的目标
			// （applySkillDamageToNPC 写入攻击者），绝不做近身主动索敌
			// ——否则城里白名监工/工人会追咬路过玩家（Java NpcCombatAi
			// 全量即纯反击模型）。无目标即原地待命。
			if state.Passive {
				return
			}
			state.TargetMember = nearestTargetLocked(targets, *transform, state.AggroRange)
			if state.TargetMember == 0 {
				return
			}
			state.Threat = 1
		}
		position, engaged := targets[state.TargetMember]
		distance := horizontalDistance(*transform, position)
		if !engaged || distance > state.ChaseRange {
			// 目标死亡/离场，或被拉出栓绳距离：清除仇恨并按栓绳规则回位。
			r.appendNPCReturnHomeLocked(id, meta, vitals, state, transform, &moves, &updates)
			return
		}
		// 该 NPC 仍持有有效目标，即便它此刻正在等待
		// 攻击冷却结束，或已超出本 tick 的攻击者数量上限。
		threatened[state.TargetMember] = true
		if now.Before(state.ControlledUntil) {
			return
		}
		if !npcChaseEnabled() && distance > state.AttackRange {
			// 追击总闸关闭（NINEYIN_NPC_CHASE=off 启动方言或 GM 面板
			// set_npc_chase=0）：追击位移整段静默，仇恨照常持有。
			// 放在攻击者名额闸之前——远距怪不占每 tick 的攻击名额，
			// 玩家走进 AttackRange 后攻击逻辑照常走。
			return
		}
		if aggressors >= combatTickMaxAggressors {
			return
		}
		aggressors++
		target := memberByID[state.TargetMember]
		if distance > state.AttackRange {
			// 路点图追击（mapnav_chase.go）：A* 折线推进 + y 插值替代
			// 旧水平直线 stepToward——穿墙（无寻路）与穿地/悬空（y 不
			// 随地形）两个根因在此一并消除（docs/map-path-waypoint-graph.md
			// §5/§6.3）。无图场景内部回落直线贴地。
			destination, delay, gaveUp := r.advanceChaseLocked(meta, state, transform, position, now)
			if gaveUp {
				// 跨分量连续规划失败：放弃仇恨按栓绳语义回位。
				r.appendNPCReturnHomeLocked(id, meta, vitals, state, transform, &moves, &updates)
				return
			}
			if err := r.setNPCTransformLocked(id, destination); err != nil {
				log.Printf("scene %s/%s: chase transform id=%d: %v", r.key.config, r.key.resource, meta.id, err)
				return
			}
			moves = append(moves, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, destination: destination, delay: delay})
			return
		}
		if now.Before(state.NextAttack) {
			return
		}
		state.NextAttack = now.Add(state.AttackEvery)
		outcome := target.player.resolveNPCAttack(meta.id, state.Damage, now)
		// on_victim 特效（reflect 指令）反打 NPC：走同一 NPC 伤亡链
		//（威胁清理/重生排程/属性帧）。
		if outcome.neigongReflect > 0 && vitals.HP > 0 {
			if vitals.ApplyDamage(outcome.neigongReflect) {
				state.Threat += outcome.neigongReflect
				if vitals.HP <= 0 {
					state.Threat = 0
					if npcRespawnBatchInterval() > 0 {
						// 批量刷新模式：重生归 refreshTick 周期扫描统一复活。
						state.RespawnAt = time.Time{}
					} else {
						state.RespawnAt = now.Add(state.RespawnAfter)
					}
				}
				updates = append(updates, npcPropertyUpdate{id: meta.id, ownerID: meta.ownerID, properties: npcCombatProperties(vitals.ActorState(meta.id, meta.ownerID))})
			}
		}
		// 攻击表现素材（锁外补发 msg337/msg20/msg24/0x2D）；朝向重算
		// 为面向目标。素材取自 NPC meta 与受害者成员，全部只读。
		// hpRatio 取受击后玩家血量百分比（0x2D 双写值；maxHP 兜底 1）。
		playerState := target.player.stateSnapshot()
		hpRatio := int32(0)
		if playerState.MaxHP > 0 {
			hpRatio = int32(int64(playerState.HP) * 100 / int64(playerState.MaxHP))
		}
		attacks = append(attacks, npcAttackPresentation{
			npcID:   meta.id,
			ownerID: meta.ownerID,
			transform: worldcore.Transform{
				X: transform.X, Y: transform.Y, Z: transform.Z,
				Orient: float32(math.Atan2(float64(position.Z-transform.Z), float64(position.X-transform.X))),
			},
			landed:         !outcome.dodged && !outcome.parried,
			hpRatio:        hpRatio,
			model:          meta.model,
			playerWideName: target.player.snapshotName(),
			npcWideName:    meta.wideName,
			victim:         target,
		})
		if outcome.rageDelta != 0 {
			if _, spFrame, spErr := target.player.addSP(outcome.rageDelta); spErr == nil && spFrame != nil {
				if rage == nil {
					rage = map[uint32][]byte{}
				}
				rage[target.objectID] = spFrame
			}
		}
		remote := target.view.remote
		if outcome.dodged {
			log.Printf("%s: NPC attack dodged id=%d config=%s", remote, meta.id, meta.configID)
		} else if outcome.stateChanged {
			playerState := target.player.stateSnapshot()
			log.Printf("%s: NPC attack id=%d config=%s damage=%d player_hp=%d/%d", remote, meta.id, meta.configID, state.Damage, playerState.HP, playerState.MaxHP)
		} else if outcome.parried {
			log.Printf("%s: NPC attack parried id=%d config=%s", remote, meta.id, meta.configID)
		}
	})
	// 玩家侧战斗临场感（Fighting/Normal 迁移）、内功特效周期节拍
	//（on_periodic，仅交战成员、脱战自停）与闪避窗过期清扫（含料敌
	// 先机原生 Buff 摘除）按成员分别结算。
	changed := make(map[uint32]bool, len(members))
	for _, member := range members {
		if member.player == nil {
			continue
		}
		if member.player.setCombatPresence(threatened[member.objectID], now) {
			changed[member.objectID] = true
		}
		if threatened[member.objectID] && member.player.dispatchNeiGongPeriodic(now) {
			changed[member.objectID] = true
		}
		if member.player.expireNeiGongDodgeWindow(now) {
			changed[member.objectID] = true
		}
	}
	r.mu.Unlock()
	return &combatTickOutcome{members: members, moves: moves, updates: updates, changed: changed, rageFrames: rage, attacks: attacks}
}

// appendNPCReturnHomeLocked 清仇恨并按栓绳规则回位：正常脱战（目标
// 死亡/离场/拉出 ChaseRange）与路点图追击放弃（跨分量连续规划失败）
// 共用同一语义。受击带入的 Fighting 态随清仇恨完全重置（HP/瘀血/MP
// 回满 + 463=0）并补发 0x10 属性帧——否则 NPC 的脱战只发生在重生，
// 怪回家后永远顶着战斗态。官方 463 生命周期 0→1→120→0（55866/
// 12141/52905 抓包一致），0 恒以重置属性帧下发且伴随资源回满；玩家
// 侧同语义迁移见 setCombatPresence（威胁/10s 迟滞）。
//
// moves/updates 以指针接收（append 扩容对调用方可见）。调用方必须
// 已持有 r.mu。
func (r *sceneRun) appendNPCReturnHomeLocked(
	id worldcore.EntityID,
	meta *sceneEntity,
	vitals *sim.Vitals,
	state *sim.Brain,
	transform *worldcore.Transform,
	moves *[]npcMoveUpdate,
	updates *[]npcPropertyUpdate,
) {
	state.Threat = 0
	state.TargetMember = 0
	meta.chase = nil
	if horizontalDistance(*transform, state.Home) > 0.25 {
		if err := r.setNPCTransformLocked(id, state.Home); err != nil {
			log.Printf("scene %s/%s: leash transform id=%d: %v", r.key.config, r.key.resource, meta.id, err)
			return
		}
		*moves = append(*moves, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, destination: state.Home})
	}
	if vitals.Logic == entity.LogicStateFighting {
		vitals.ResetAfterLeash()
		*updates = append(*updates, npcPropertyUpdate{id: meta.id, ownerID: meta.ownerID, properties: npcCombatProperties(vitals.ActorState(meta.id, meta.ownerID))})
	}
}

// nearestTargetLocked 返回 aggroRange 内最近的存活目标；没有则返回 0。
func nearestTargetLocked(targets map[uint32]worldcore.Transform, from worldcore.Transform, aggroRange float32) uint32 {
	best := uint32(0)
	bestDistance := aggroRange
	for memberID, position := range targets {
		distance := horizontalDistance(from, position)
		if distance <= bestDistance {
			// 同距离时取成员 ID 较小者，保证线上行为确定。
			if distance == bestDistance && best != 0 && memberID > best {
				continue
			}
			best = memberID
			bestDistance = distance
		}
	}
	return best
}

// combatTickForTest 保留单视图独占场景的测试语义：把玩家登记为
// 唯一成员（位置即其坐标）后同步驱动一次场景 tick。
func (s *sceneLifecycle) combatTickForTest(player *playerActor, position role.Position, now time.Time) combatTickResult {
	if s == nil || player == nil {
		return combatTickResult{}
	}
	before := player.stateSnapshot()
	wireID := s.run.attachMember(&sceneMember{selfID: playerObjectID, player: player, view: s, position: position, active: true})
	s.mu.Lock()
	s.wireID = wireID
	s.mu.Unlock()
	// 独占测试视图没有目录驱动的可见集；按旧行为把全部已注册
	// NPC 视为可见（帧无过滤直发）。
	s.run.mu.Lock()
	s.run.eachNPCLocked(func(id worldcore.EntityID, meta *sceneEntity, _ *worldcore.Transform) {
		s.mu.Lock()
		s.visibleNPCs[meta.id] = struct{}{}
		s.mu.Unlock()
	})
	s.run.mu.Unlock()
	outcome := s.run.simulateLocked(now)
	if outcome != nil {
		s.run.broadcastNPCFrames(outcome.members, outcome.moves, outcome.updates)
		s.run.broadcastNPCAttacks(outcome.attacks)
	}
	after := player.stateSnapshot()
	return combatTickResult{playerStateChanged: after.LogicState != before.LogicState || after.HP != before.HP}
}

// scheduleCombatMoveArrival 只在客户端拥有一个移动间隔
// 来朝目的地播放动画之后，才确认最终位置。transform
// 比较即是取消令牌：持续追击每 tick 都会更新它，
// 因此过期的定时器绝不会确认一个陈旧的目的地——但持续追击
// 期间也不能让客户端对 NPC 的位置认知一直停在出生点（官服
// a.pcapng 追击中的怪拿到 6 次坐标变化的 0x1F；本服 57587 会话
// 追击全程零 0x1F，客户端认知滞后 4m+，冲锋 c61 终点随之错位）。
// 因此被更新的定时器不再静默丢弃：距上次广播位移超阈值时改发
// 当前权威坐标的 0x1F（节流 ≥1m 且 ≥500ms，与官服追逐中几秒一帧
// 的节奏同量级；同 tick 与 0x20 成对发送的瞬移观感只在无节流的
// 逐 tick 直发时出现）。
// 定时器归场景运行时所有（epoch 防护），确认帧扇出给
// 当前视口内可见该 NPC 的成员。
func (r *sceneRun) scheduleCombatMoveArrival(members []*sceneMember, move npcMoveUpdate, delay time.Duration) {
	if delay <= 0 {
		delay = time.Second
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	epoch := r.epoch
	timer := time.AfterFunc(delay, func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.closed || r.epoch != epoch {
			return
		}
		meta, exists := r.npcMeta(worldcore.EntityID(move.id))
		if !exists || meta.ownerID != move.ownerID {
			return
		}
		transform, transformOK := r.npcTransform(worldcore.EntityID(move.id))
		if !transformOK {
			return
		}
		if !sameCombatTransform(transform, move.destination) {
			// 被更新的移动取代（追击继续）：节流补发当前权威坐标，
			// 防客户端认知长期滞留在旧位置。
			r.throttledNPCLocationBroadcast(members, meta, transform)
			return
		}
		r.throttledNPCLocationBroadcast(members, meta, transform)
	})
	r.timers = append(r.timers, timer)
	r.mu.Unlock()
}

// throttledNPCLocationBroadcast 带节流的 NPC 0x1F 广播：距上次已广播
// 位置水平位移 ≥npcLocationBroadcastMinMeters 且距上次发送 ≥
// npcLocationBroadcastMinInterval 才发。调用方必须已持有 r.mu。
func (r *sceneRun) throttledNPCLocationBroadcast(members []*sceneMember, meta sceneEntity, transform worldcore.Transform) {
	now := time.Now()
	if last, ok := r.npcLastLocation[meta.id]; ok {
		if now.Sub(last.at) < npcLocationBroadcastMinInterval ||
			horizontalDistance(last.transform, transform) < npcLocationBroadcastMinMeters {
			return
		}
	}
	if r.npcLastLocation == nil {
		r.npcLastLocation = make(map[uint32]npcLocationBroadcast)
	}
	r.npcLastLocation[meta.id] = npcLocationBroadcast{transform: transform, at: now}
	frame := serverLocation(meta.id, meta.ownerID, transform)
	for _, member := range members {
		if member.view == nil || !member.view.npcVisible(meta.id) {
			continue
		}
		if err := member.view.conn.WriteFrame(frame); err != nil {
			log.Printf("%s: sync combat NPC location id=%d: %v", member.view.remote, meta.id, err)
		}
	}
}

// broadcastNPCFrames 在锁外把战斗 tick 产出的 NPC 移动/属性帧
// 扇出给视口内可见对应 NPC 的成员，并排布到达确认定时器。
func (r *sceneRun) broadcastNPCFrames(members []*sceneMember, moves []npcMoveUpdate, updates []npcPropertyUpdate) {
	for _, move := range moves {
		frame := serverMoving(move.id, move.ownerID, motionFromDest(move.destination))
		for _, member := range members {
			if member.view == nil || !member.view.npcVisible(move.id) {
				continue
			}
			if err := member.view.conn.WriteFrame(frame); err != nil {
				log.Printf("%s: combat move NPC %d: %v", member.view.remote, move.id, err)
			}
		}
		// ServerLocation 在当前客户端中是到达确认。不要
		// 在同一个战斗 tick 中让它与 ServerMoving 成对发送：那样会使
		// actor 瞬移到目的地而不是播放移动动作。之后
		// 的追击目的地会在该定时器执行之前更改权威
		// transform，从而令此次确认失效。到达延时按折线路径长/速度
		//（docs/map-path-waypoint-graph.md §6.3：折线长于直线，固定
		// 1s 会提前确认回到漂移）；重生/栓绳瞬移类保持默认 1s。
		r.scheduleCombatMoveArrival(members, move, move.delay)
	}
	for _, update := range updates {
		frame, err := sceneObjectProperties(update.id, update.ownerID, 0, update.properties)
		if err != nil {
			log.Printf("scene %s/%s: encode combat NPC state %d: %v", r.key.config, r.key.resource, update.id, err)
			continue
		}
		for _, member := range members {
			if member.view == nil || !member.view.npcVisible(update.id) {
				continue
			}
			if err := member.view.conn.WriteFrame(frame); err != nil {
				log.Printf("%s: combat NPC state %d: %v", member.view.remote, update.id, err)
			}
		}
	}
}

// relayNPCPropertyFrame 把连接循环里产生的单条 NPC 属性帧（目前只有
// 击杀致死帧）中继给视口内可见该 NPC 的其他成员；击杀者自己的视图经
// exclude 指针跳过——首发帧已由其连接循环写入。普通掉血帧仍仅击杀者
// 可见（完整战斗可见性是独立专题，不并入重生批次）。
func (r *sceneRun) relayNPCPropertyFrame(id uint32, frame []byte, exclude *sceneLifecycle) {
	r.relayCombatFrame(id, frame, exclude, "NPC property")
}

// relayCombatFrame 是通用战斗帧中继：把首发成员连接循环已写入的帧
// 原样发给视口内可见该 NPC 的其他成员。命中表现帧族（msg24/0x2D/
// msg18/msg205、NPC 三段攻击帧）与致死帧共用同一条扇出路径。
func (r *sceneRun) relayCombatFrame(id uint32, frame []byte, exclude *sceneLifecycle, kind string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	for _, member := range r.members {
		if member.view == nil || member.view == exclude {
			continue
		}
		if !member.view.npcVisible(id) {
			continue
		}
		if err := member.view.conn.WriteFrame(frame); err != nil {
			log.Printf("%s: relay %s %d: %v", member.view.remote, kind, id, err)
		}
	}
}

// fightRelationAddLocked 累计「玩家→受害 NPC」伤害并返回新累计值
// （0x13 fight_relation_rec 单格语义：抓包每击 +固定伤害 递增）。
// 要求调用方已持 r.mu。NPC 死亡走 fightRelationResetLocked 清零。
func (r *sceneRun) fightRelationAddLocked(victimID uint32, damage int32) int32 {
	if r.fightRelation == nil {
		r.fightRelation = make(map[uint32]int32)
	}
	r.fightRelation[victimID] += damage
	return r.fightRelation[victimID]
}

// fightRelationResetLocked 清除受害 NPC 的战斗关系累计格（致死/
// 注销时调用，重生后从零累计）。
func (r *sceneRun) fightRelationResetLocked(victimID uint32) {
	delete(r.fightRelation, victimID)
}

func sameCombatTransform(a, b worldcore.Transform) bool {
	const epsilon = 0.01
	return math.Abs(float64(a.X-b.X)) < epsilon &&
		math.Abs(float64(a.Y-b.Y)) < epsilon &&
		math.Abs(float64(a.Z-b.Z)) < epsilon &&
		math.Abs(float64(a.Orient-b.Orient)) < epsilon
}

// NPC 0x1F 位置广播节流参数（官服 a.pcapng 追击中的怪 ~几秒一帧）。
const (
	npcLocationBroadcastMinMeters   = 1.0
	npcLocationBroadcastMinInterval = 500 * time.Millisecond
)

// npcLocationBroadcast 是单个 NPC 最近一次 0x1F 广播水位。
type npcLocationBroadcast struct {
	transform worldcore.Transform
	at        time.Time
}

// setNPCTransformLocked 更新 NPC 权威坐标：可视权威 Scene 实体
// （revision 抬升，供各视图下一次 reconcile 出 Updates）与
// Transform 组件同步写。调用方必须已持有 r.mu。
func (r *sceneRun) setNPCTransformLocked(id worldcore.EntityID, destination worldcore.Transform) error {
	if _, exists := r.npcMeta(id); !exists {
		return fmt.Errorf("set transform for unknown NPC %d", id)
	}
	current, err := r.scene.Get(id)
	if err != nil {
		return err
	}
	current.Entity.Transform = destination
	for _, property := range []struct {
		name  string
		value float32
	}{{"PosiX", destination.X}, {"PosiY", destination.Y}, {"PosiZ", destination.Z}, {"Orient", destination.Orient}} {
		if err := current.Entity.Properties.Set(property.name, worldcore.Float32(property.value)); err != nil {
			return err
		}
	}
	if err := r.scene.Replace(current.Entity); err != nil {
		return err
	}
	r.writeNpcTransform(id, destination)
	return nil
}

// npcCombatProperties 是 NPC 战斗面板 0x10 属性集（抓包 52905 #3027
// 实证 9 属性 57B 逐字节对齐；Java npcCombatFrame 只发其中 HP 族 5 项，
// 抓包口径是双索引族超集——新旧两域 HP/HPRatio/MaxHP 各发一遍 + 恒 1
// 的 Prop21[Byte]。玩家瘀血 HitHP@481 不在官方 NPC 帧（抓包 0 次），
// 归玩家面板专属）。
func npcCombatProperties(state entity.ActorState) []clientdata.IndexedProperty {
	hpRatio := int32(0)
	if state.MaxHP > 0 {
		hpRatio = int32(int64(state.HP) * 100 / int64(state.MaxHP))
	}
	return []clientdata.IndexedProperty{
		{Index: 21, Name: "Prop21", Value: clientdata.ByteValue(1)},
		{Index: 28, Name: "HPRatio", Value: clientdata.Int32Value(hpRatio)},
		{Index: 30, Name: "MaxHP", Value: clientdata.Int32Value(state.BaseMaxHP)},
		{Index: 32, Name: "HP", Value: clientdata.Int32Value(state.HP)},
		{Index: 463, Name: "LogicState", Value: clientdata.ByteValue(state.LogicState)},
		{Index: 465, Name: "HP", Value: clientdata.Int32Value(state.HP)},
		{Index: 469, Name: "HPRatio", Value: clientdata.Int32Value(hpRatio)},
		{Index: 578, Name: "MaxHP", Value: clientdata.Int32Value(state.BaseMaxHP)},
		{Index: propDead, Name: "Dead", Value: clientdata.ByteValue(boolByte(state.HP <= 0))},
	}
}

func horizontalDistance(a, b worldcore.Transform) float32 {
	dx, dz := a.X-b.X, a.Z-b.Z
	return float32(math.Hypot(float64(dx), float64(dz)))
}

func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
