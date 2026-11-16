package main

import (
	"fmt"
	"log"
	"math"
	"path/filepath"
	"sync"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// GM 训练木桩（面板「战斗与演练」组 spawn_dummy，2026-11-16 落地）：
// 在角色身边生成纯沙包怪（官方木人桩 AttackNpc 模板 NpcClone005201，
// clonenpc/attacknpc.txt —— 跑步速度列官方即 0），可指定血量与数量，
// 环形摆放朝向角色。行为学=纯沙包：
//   - Passive + AggroRange=0：从不主动索敌；
//   - ChaseRange=0：受击锁定的攻击者在任意水平距离外都立刻按栓绳语义
//     清仇恨（木桩在出生点，回位分支零位移不发帧）——既不追击也不反击；
//   - AttackRange=-1：即便攻击者贴到 3D 距离 0（水平重合的极端情况）
//     命中分位判定也永不满足，不出招。
//
// 打空血后走 NPC 通用重生链（15s 满血原地复活），可反复测试输出。
const (
	dummyTemplateConfigID = "NpcClone005201"
	dummyFallbackModel    = `npc\brokennpc\brokenitem033`
	dummyDisplayName      = "训练木桩"
	// dummyRingRadius 多只木桩的环形摆放半径（米）；单只摆在角色面前。
	dummyRingRadius = float32(3)
	dummyFrontGap   = float32(2.5)
)

var (
	dummyTableOnce sync.Once
	dummyTable     *clientdata.NPCTemplateTable
)

// dummyTemplateTable 缓存装载 clonenpc/attacknpc.txt（官方木人桩行所在表）。
// 资源缺失（测试环境）返回 nil，生成走手工回落模板。
func dummyTemplateTable() *clientdata.NPCTemplateTable {
	dummyTableOnce.Do(func() {
		path := filepath.Join(defaultModernShareRoot, "npc", "npcconfig", "clonenpc", "attacknpc.txt")
		file, err := openResourceReader(path)
		if err != nil {
			log.Printf("dummy template table %s unavailable (%v); using fallback", path, err)
			return
		}
		table, loadErr := clientdata.LoadNPCTemplateTable(file, path)
		_ = file.Close()
		if loadErr != nil {
			log.Printf("dummy template table %s: %v; using fallback", path, loadErr)
			return
		}
		dummyTable = table
	})
	return dummyTable
}

// resolveDummySpawn 解析官方木人桩模板并按本次参数覆写血量与显示名。
// Runtime 层在模板/creator 之后应用且经官方属性表校验（MaxHP/HP/Name
// 均为官方场景域字段）；registerNPCLocked 从 MaxHP 建 Vitals。
func resolveDummySpawn(hp int32, name string) (npcSpawn, error) {
	if table := dummyTemplateTable(); table != nil {
		resolved, err := clientdata.ResolveNPC(table, clientdata.NPCCreatorInstance{TemplateID: dummyTemplateConfigID}, clientdata.ResolveOptions{
			Defaults: clientdata.VisibleNPCSpawnDefaultsCompatV1(),
			Runtime: map[string]clientdata.Value{
				"MaxHP": clientdata.Int32Value(hp),
				"HP":    clientdata.Int32Value(hp),
				"Name":  clientdata.WideStringValue(name),
			},
		})
		if err != nil {
			return npcSpawn{}, err
		}
		return npcSpawn{resolved: resolved}, nil
	}
	return fallbackDummySpawn(hp, name), nil
}

// fallbackDummySpawn 手工最小模板（资源缺失时；与 fallbackLootBoxSpawn
// 同属性面，ScriptClass=AttackNpc 保住战斗交互与可受击语义）。
func fallbackDummySpawn(hp int32, name string) npcSpawn {
	properties := map[string]clientdata.PropertyValue{}
	for propName, value := range map[string]clientdata.Value{
		"Type":          clientdata.ByteValue(4),
		"ConfigID":      clientdata.StringValue(dummyTemplateConfigID),
		"Resource":      clientdata.StringValue(dummyFallbackModel),
		"NpcType":       clientdata.Int32Value(172),
		"Level":         clientdata.Int32Value(45),
		"MaxHP":         clientdata.Int32Value(hp),
		"HP":            clientdata.Int32Value(hp),
		"NotPositive":   clientdata.Int32Value(1),
		"CollideRadius": clientdata.Float32Value(1.88),
		"Name":          clientdata.WideStringValue(name),
		"WeaponMode":    clientdata.StringValue("hand_0"),
	} {
		properties[propName] = clientdata.PropertyValue{Value: value}
	}
	return npcSpawn{resolved: clientdata.ResolvedNPC{
		ConfigID:    dummyTemplateConfigID,
		ScriptClass: "AttackNpc",
		Properties:  properties,
	}}
}

// neuterDummyBrain 把 registerNPCLocked 按模板建出的战斗组件改写成纯
// 沙包参数（见文件头注释三条闸）。调用方必须已持有 r.mu。
func (r *sceneRun) neuterDummyBrain(id uint32) {
	entityID := worldcore.EntityID(id)
	brain, exists := r.combat.Brain(entityID)
	if !exists {
		return
	}
	brain.Passive = true
	brain.AggroRange = 0
	brain.ChaseRange = 0
	brain.EngageRange = 0
	brain.AttackRange = -1
	brain.Threat = 0
	brain.TargetMember = 0
	r.combat.WriteBrain(entityID, brain)
}

// spawnTrainingDummiesLocked 在 center 周围生成 count 只训练木桩并登记
// gmDummies（clear_dummies 的清除清单）。单只=角色面前 dummyFrontGap 米，
// 多只=半径 dummyRingRadius 均匀环绕（首只对齐角色朝向，测 AOE 友好），
// 全部面朝角色。物化帧排队 pendingObjectFrames，由调用方锁外
// flushPendingObjectFrames 扇出（GM 宝箱同款）。调用方必须已持有 r.mu。
func (r *sceneRun) spawnTrainingDummiesLocked(center role.Position, hp int32, count int) []uint32 {
	ids := make([]uint32, 0, count)
	for i := 0; i < count; i++ {
		angle := center.Orient
		radius := dummyFrontGap
		if count > 1 {
			angle = center.Orient + float32(2*math.Pi)*float32(i)/float32(count)
			radius = dummyRingRadius
		}
		x := center.X + float32(math.Cos(float64(angle)))*radius
		z := center.Z + float32(math.Sin(float64(angle)))*radius
		// 朝向角色：官方朝向约定 atan2(dz, dx)（combat_ai 停止对同式）。
		orient := float32(math.Atan2(float64(center.Z-z), float64(center.X-x)))
		name := dummyDisplayName
		if count > 1 {
			name = fmt.Sprintf("%s·%d", dummyDisplayName, i+1)
		}
		spawn, err := resolveDummySpawn(hp, name)
		if err != nil {
			log.Printf("scene %s/%s: resolve training dummy: %v", r.key.config, r.key.resource, err)
			continue
		}
		spawn.x, spawn.y, spawn.z, spawn.orient = x, center.Y, z, orient
		spawn.route, spawn.schedule = nil, nil // 木桩不巡逻
		id := forkNPCID(r.nextWireID)
		if id == forkNPCID(0) {
			id = forkNPCID(2)
		}
		r.nextWireID++
		if err := r.registerNPCLocked(id, spawn); err != nil {
			log.Printf("scene %s/%s: register training dummy: %v", r.key.config, r.key.resource, err)
			continue
		}
		r.neuterDummyBrain(id)
		meta, ok := r.npcMeta(worldcore.EntityID(id))
		if !ok {
			continue
		}
		if r.gmDummies == nil {
			r.gmDummies = make(map[uint32]uint32)
		}
		r.gmDummies[id] = meta.ownerID
		transform := worldcore.Transform{X: x, Y: center.Y, Z: z, Orient: orient}
		r.pendingObjectFrames = append(r.pendingObjectFrames, pendingObjectFrame{
			id: id, ownerID: meta.ownerID, visibleWith: 0,
			payload: append([]byte(nil), meta.payload...), transform: transform,
		})
		ids = append(ids, id)
	}
	return ids
}

// clearTrainingDummies 清除本场景全部 GM 训练木桩（活体与尸体一并注销，
// 0x0E 扇出给看得见各木桩的成员），返回清除只数。despawnLootBox 同款
// 锁内注销/锁外发帧分工。
func (r *sceneRun) clearTrainingDummies() int {
	r.mu.Lock()
	if r.closed || len(r.gmDummies) == 0 {
		r.mu.Unlock()
		return 0
	}
	type dummyRemoval struct{ id, ownerID uint32 }
	removals := make([]dummyRemoval, 0, len(r.gmDummies))
	for id, ownerID := range r.gmDummies {
		if err := r.unregisterNPC(worldcore.EntityID(id)); err != nil {
			log.Printf("scene %s/%s: remove training dummy %d: %v", r.key.config, r.key.resource, id, err)
			continue
		}
		delete(r.deadNPCs, id)
		delete(r.fightRelation, id)
		delete(r.npcMoveState, id)
		removals = append(removals, dummyRemoval{id: id, ownerID: ownerID})
	}
	r.gmDummies = nil
	members := r.sortedMembersLocked()
	r.mu.Unlock()
	for _, removal := range removals {
		frame := sceneRemoveObject(removal.id, removal.ownerID)
		for _, member := range members {
			if member.view == nil || !member.view.npcVisible(removal.id) {
				continue
			}
			if err := member.view.conn.WriteFrame(frame); err != nil {
				log.Printf("%s: despawn training dummy %d: %v", member.view.remote, removal.id, err)
			}
		}
	}
	return len(removals)
}
