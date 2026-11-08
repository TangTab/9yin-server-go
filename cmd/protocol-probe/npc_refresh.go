package main

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/local/9yin-go-server/internal/sim"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// 怪物死亡→刷新周期参数。批量模式下死了的怪在扫描点统一重生，
// 没死的一律不动；尸体超过 TTL 直接清除（0x0E），重生时在出生点
// 重新物化（0x0D）。main() 的 flag 写初值（生产默认 30s/10s）；零值
// =两项都关（每怪独立 RefreshTime 计时、尸体保留），测试环境即此
// 基线。GM 面板运行时可调。
var (
	npcRespawnBatchIntervalNanos atomic.Int64
	npcCorpseTTLNanos            atomic.Int64
	lootBoxTTLNanos              atomic.Int64
)

// lootBoxDefaultTTL 掉落宝箱在场默认时长。官方 attacknpc DestroyTime2
// （最大消失时间）= 60s；本服默认放宽到 5 分钟，-lootbox-ttl / GM 面板
// 可调（0=保留到拾空）。过期结算每 tick 现读全局参数（按宝箱生成时刻
// 推算），改值对在场宝箱即刻生效。
const lootBoxDefaultTTL = 5 * time.Minute

func npcRespawnBatchInterval() time.Duration {
	return time.Duration(npcRespawnBatchIntervalNanos.Load())
}

func npcCorpseTTL() time.Duration {
	return time.Duration(npcCorpseTTLNanos.Load())
}

func lootBoxTTL() time.Duration {
	return time.Duration(lootBoxTTLNanos.Load())
}

func setNpcRespawnBatchInterval(d time.Duration) { npcRespawnBatchIntervalNanos.Store(int64(d)) }
func setNpcCorpseTTL(d time.Duration)            { npcCorpseTTLNanos.Store(int64(d)) }
func setLootBoxTTL(d time.Duration)              { lootBoxTTLNanos.Store(int64(d)) }

// deadNPCRecord 登记一具等待重生的怪物尸体。spawnIndex 是目录下标
// （重生实体号 = spawnIndex+2 保持不变）；removed 表示尸体已被 TTL
// 清除，重生走重注册路径。
type deadNPCRecord struct {
	spawnIndex int
	diedAt     time.Time
	respawnAt  time.Time // 每怪模式（批量关闭时）的重生时刻
	removed    bool
}

type npcDespawnUpdate struct {
	id      uint32
	ownerID uint32
}

type npcReaddUpdate struct {
	id        uint32
	ownerID   uint32
	payload   []byte
	transform worldcore.Transform
}

// pendingObjectFrame 由 apply* 击杀路径在锁内排队的动态对象物化帧
// （掉落宝箱 0x0D+0x1F），visibleWith 是可见性判定的锚对象（尸体）。
type pendingObjectFrame struct {
	id          uint32
	ownerID     uint32
	visibleWith uint32
	payload     []byte
	transform   worldcore.Transform
}

type refreshTickOutcome struct {
	members  []*sceneMember
	despawns []npcDespawnUpdate
	readds   []npcReaddUpdate
	moves    []npcMoveUpdate
	updates  []npcPropertyUpdate
}

// refreshTick 是 tickOnce 的第三段：尸体 TTL 清除、批量重生扫描与
// 宝箱过期。锁内结算、锁外发帧（与 combatTick 同纪律）。开头先冲排
// 击杀路径排队的宝箱物化帧（正常由击杀中继即时冲排，这里是兜底）。
func (r *sceneRun) refreshTick(now time.Time) {
	r.flushPendingObjectFrames()
	outcome := r.refreshSettleLocked(now)
	if outcome == nil {
		return
	}
	r.flushRefreshFrames(outcome)
}

func (r *sceneRun) refreshSettleLocked(now time.Time) *refreshTickOutcome {
	interval, ttl := npcRespawnBatchInterval(), npcCorpseTTL()
	sweepDue := interval > 0 && !now.Before(r.nextSweepAt)
	if !sweepDue && ttl <= 0 && len(r.deadNPCs) == 0 && len(r.lootBoxes) == 0 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || len(r.members) == 0 {
		return nil
	}
	out := &refreshTickOutcome{members: r.sortedMembersLocked()}
	// 尸体 TTL：超时清除（0x0E 直发 + 注销实体），等批量/定时重生重注册。
	// 挂尸掉落（r.lootBoxes[id]）的尸体不走本分支——由掉落 TTL 接管
	//（拾空/超时清尸时 removeLootBoxLocked 同步 removed，重生随之放行）。
	if ttl > 0 {
		for id, rec := range r.deadNPCs {
			if rec.removed || now.Before(rec.diedAt.Add(ttl)) {
				continue
			}
			if _, hasLoot := r.lootBoxes[id]; hasLoot {
				continue
			}
			ownerID := uint32(1)
			if meta, ok := r.npcMeta(worldcore.EntityID(id)); ok {
				ownerID = meta.ownerID
			}
			if err := r.unregisterNPC(worldcore.EntityID(id)); err != nil {
				log.Printf("scene %s/%s: remove corpse id=%d: %v", r.key.config, r.key.resource, id, err)
				continue
			}
			rec.removed = true
			r.deadNPCs[id] = rec
			out.despawns = append(out.despawns, npcDespawnUpdate{id: id, ownerID: ownerID})
		}
	}
	// 每怪模式（批量关闭）：已被 TTL 清除的尸体按各自 respawnAt 重注册。
	// 批量模式期间被杀（无 respawnAt）的尸体按死亡时刻+默认间隔兜底。
	if interval <= 0 {
		for id, rec := range r.deadNPCs {
			if !rec.removed {
				continue
			}
			deadline := rec.respawnAt
			if deadline.IsZero() {
				deadline = rec.diedAt.Add(defaultNPCRespawn)
			}
			if now.Before(deadline) {
				continue
			}
			if r.respawnRemovedNPCLocked(out, id, rec) {
				delete(r.deadNPCs, id)
			}
		}
	}
	// 批量扫描：死了的统一重生（尸体在场=原地复活帧组；已清除=重注册），
	// 没死的一律不动。带未拾掉落的尸体跳过复活——掉落清除（拾空/超时）
	// 标记 removed 后按重注册路径重生。
	if sweepDue {
		r.nextSweepAt = now.Add(interval)
		for id, rec := range r.deadNPCs {
			if rec.removed {
				if r.respawnRemovedNPCLocked(out, id, rec) {
					delete(r.deadNPCs, id)
				}
				continue
			}
			if _, hasLoot := r.lootBoxes[id]; hasLoot {
				continue
			}
			entityID := worldcore.EntityID(id)
			vitals, hasVitals := r.combat.Vitals(entityID)
			brain, hasBrain := r.combat.Brain(entityID)
			if !hasVitals || !hasBrain {
				delete(r.deadNPCs, id)
				continue
			}
			ownerID := uint32(1)
			if meta, ok := r.npcMeta(entityID); ok {
				ownerID = meta.ownerID
			}
			if !r.reviveNPCLocked(entityID, &vitals, &brain) {
				continue
			}
			out.moves = append(out.moves, npcMoveUpdate{id: id, ownerID: ownerID, destination: brain.Home})
			out.updates = append(out.updates, npcPropertyUpdate{id: id, ownerID: ownerID, properties: npcCombatProperties(vitals.ActorState(id, ownerID))})
			delete(r.deadNPCs, id)
		}
	}
	// 掉落过期：在场超 lootBoxTTL（每 tick 现读，改值即刻生效；0=保留
	// 到拾空）直接消失。挂尸形态=清尸体（removeLootBoxLocked 内部同步
	// deadNPCs removed，重生随之放行）；实体宝箱=注销宝箱本体。
	if boxTTL := lootBoxTTL(); boxTTL > 0 {
		for id, box := range r.lootBoxes {
			if now.Sub(box.spawnedAt) < boxTTL {
				continue
			}
			r.removeLootBoxLocked(out, id, box.ownerID)
		}
	}
	if len(out.despawns) == 0 && len(out.readds) == 0 && len(out.moves) == 0 && len(out.updates) == 0 {
		return nil
	}
	return out
}

// reviveNPCLocked 与 simulateLocked 每怪复活分支同帧语义：满血复活、
// 清仇恨、传送回出生点。调用方必须已持有 r.mu。
func (r *sceneRun) reviveNPCLocked(entityID worldcore.EntityID, vitals *sim.Vitals, brain *sim.Brain) bool {
	vitals.Revive()
	brain.Threat = 0
	brain.TargetMember = 0
	brain.NextAttack = time.Time{}
	brain.RespawnAt = time.Time{}
	r.combat.WriteVitals(entityID, *vitals)
	r.combat.WriteBrain(entityID, *brain)
	if err := r.setNPCTransformLocked(entityID, brain.Home); err != nil {
		log.Printf("scene %s/%s: respawn transform id=%d: %v", r.key.config, r.key.resource, entityID, err)
		return false
	}
	return true
}

// respawnRemovedNPCLocked 把已被 TTL 清除的尸体按目录行重注册回共享
// 世界（实体号不变），并登记 0x0D+0x1F 直发帧。返回是否终结该记录。
// 调用方必须已持有 r.mu。
func (r *sceneRun) respawnRemovedNPCLocked(out *refreshTickOutcome, id uint32, rec deadNPCRecord) bool {
	if rec.spawnIndex < 0 || rec.spawnIndex >= len(r.catalog) {
		log.Printf("scene %s/%s: removed NPC %d has no catalog row; dropping dead record", r.key.config, r.key.resource, id)
		return true
	}
	if err := r.registerNPCLocked(id, r.catalog[rec.spawnIndex]); err != nil {
		log.Printf("scene %s/%s: respawn register id=%d: %v", r.key.config, r.key.resource, id, err)
		return false
	}
	meta, ok := r.npcMeta(worldcore.EntityID(id))
	if !ok {
		return false
	}
	transform, ok := r.npcTransform(worldcore.EntityID(id))
	if !ok {
		return false
	}
	out.readds = append(out.readds, npcReaddUpdate{
		id: id, ownerID: meta.ownerID,
		payload: append([]byte(nil), meta.payload...), transform: transform,
	})
	return true
}

// flushRefreshFrames 在锁外把 refreshTick 产出的清除/重生帧扇出给
// 视口内可见对应对象的成员。
func (r *sceneRun) flushRefreshFrames(out *refreshTickOutcome) {
	for _, despawn := range out.despawns {
		frame := sceneRemoveObject(despawn.id, despawn.ownerID)
		for _, member := range out.members {
			if member.view == nil || !member.view.npcVisible(despawn.id) {
				continue
			}
			if err := member.view.conn.WriteFrame(frame); err != nil {
				log.Printf("%s: despawn object %d: %v", member.view.remote, despawn.id, err)
			}
		}
	}
	for _, readd := range out.readds {
		location := serverLocation(readd.id, readd.ownerID, readd.transform)
		for _, member := range out.members {
			if member.view == nil || !member.view.npcVisible(readd.id) {
				continue
			}
			if err := member.view.conn.WriteFrame(readd.payload); err != nil {
				log.Printf("%s: rematerialize NPC %d: %v", member.view.remote, readd.id, err)
			}
			if err := member.view.conn.WriteFrame(location); err != nil {
				log.Printf("%s: locate NPC %d: %v", member.view.remote, readd.id, err)
			}
		}
	}
	if len(out.moves) > 0 || len(out.updates) > 0 {
		r.broadcastNPCFrames(out.members, out.moves, out.updates)
	}
}

// recordNPCDeathLocked 在击杀点登记尸体记录，并按官服双形态掉落方言
// 分叉：AIEditorPackID 登记了死亡生箱事件 → 独立宝箱实体
//（spawnBoxNPCEntityLocked）；否则 DropID roll 非空 → 掉落挂尸体
//（attachLootToCorpseLocked，CanPick/PickMember 驱动客户端冒光）；
// roll 为空 = 纯尸体（corpse TTL 清除）。调用方必须已持有 r.mu
//（apply* 伤害路径）。
func (r *sceneRun) recordNPCDeathLocked(id, ownerID uint32, brain sim.Brain, now time.Time) {
	rec := deadNPCRecord{spawnIndex: int(id) - 2, diedAt: now, respawnAt: brain.RespawnAt}
	if rec.spawnIndex < 0 || rec.spawnIndex >= len(r.catalog) {
		// 目录外实体（测试/动态摆放）：无重注册来源，仅支持原地复活路径。
		rec.spawnIndex = -1
	}
	if existing, ok := r.deadNPCs[id]; ok && existing.removed {
		rec.removed = true
	}
	r.deadNPCs[id] = rec
	if rec.spawnIndex >= 0 {
		if templates := chestSpawnConfigForNPC(r.catalog[rec.spawnIndex]); len(templates) > 0 {
			r.spawnBoxNPCEntityLocked(id, templates[0], now)
			return
		}
	}
	r.attachLootToCorpseLocked(id, ownerID, now)
}
