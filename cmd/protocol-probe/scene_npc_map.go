package main

import (
	"fmt"
	"log"
)

// writeSceneNPCMapFrames 下发 msg229 130 场景 NPC 点位表（大地图图层）。
//
// 官服方言（c.pcapng 登录爆发段 62 帧逐值统计，2026-11-10）：
//
//	[229, 130, obj, "npc\配置名", x, y, z, orient,
//	 缩放串(54/62 为空), 703|701(各半), 图标类(12 多数), 0, 0,
//	 W:名(47/62 空), "", "0.0,<orient>,0.0"]
//
// 大地图的 NPC/传送点标记与点击寻路目标来自这张表——本服此前零下发，
// 大地图图层为空、点击无反应（msg266 都不发起）。最小可行版取多数值：
// v9=703/v10=1/v11=0/v12=0/名空/缩放空。
func writeSceneNPCMapFrames(conn sceneMessageConnection, world *sceneLifecycle, remote string) {
	if world == nil {
		return
	}
	type npcPoint struct {
		id                                                               uint32
		owner                                                            uint32
		configID                                                         string
		x, y, z, orient                                                  float32
	}
	world.run.mu.Lock()
	if world.run.closed {
		world.run.mu.Unlock()
		return
	}
	snapshot := world.run.scene.Snapshot()
	points := make([]npcPoint, 0, len(snapshot.Entities))
	for _, entity := range snapshot.Entities {
		meta, ok := world.run.npcMeta(entity.Entity.ID)
		if !ok {
			continue
		}
		points = append(points, npcPoint{
			id: uint32(entity.Entity.ID), owner: meta.ownerID, configID: meta.configID,
			x: entity.Entity.Transform.X, y: entity.Entity.Transform.Y, z: entity.Entity.Transform.Z,
			orient: entity.Entity.Transform.Orient,
		})
	}
	world.run.mu.Unlock()

	sent := 0
	// 官服事务序（c.pcapng 71 帧序列压缩定版）：[144,145,154] 前导 →
	// 130×N → [144,145,154] 收尾 → 再一组 [144,145,154]。144/145/154
	// 是表事务定界符——裸发 130 客户端不提交（06:59 会话 3016 帧实证：
	// 发了但大地图仍空）。
	writeNPCMapDelimiter := func() bool {
		specs := []struct {
			id int32
		}{{144}, {145}, {154}}
		for _, del := range specs {
			var frame []byte
			var err error
			if del.id == 144 {
				frame, err = serverCustomIntMessage(229, customInt(144), customInt(0))
			} else {
				frame, err = serverCustomIntMessage(229, customInt(del.id))
			}
			if err != nil {
				log.Printf("%s: scene npc map delimiter %d: %v", remote, del.id, err)
				return false
			}
			if err := conn.WriteFrame(frame); err != nil {
				log.Printf("%s: scene npc map delimiter %d: %v", remote, del.id, err)
				return false
			}
		}
		return true
	}
	if !writeNPCMapDelimiter() {
		return
	}
	for _, p := range points {
		frame, err := serverCustomIntMessage(229,
			customInt(130),
			customObject(p.id, p.owner),
			customString("npc\\"+p.configID),
			customFloat32(p.x), customFloat32(p.y), customFloat32(p.z), customFloat32(p.orient),
			customString(""),
			customInt(703),
			customInt(1),
			customInt(0),
			customInt(0),
			customWideString(""),
			customString(""),
			customString(fmt.Sprintf("0.0,%.3f,0.0", p.orient)),
		)
		if err != nil {
			log.Printf("%s: scene npc map frame %s: %v", remote, p.configID, err)
			continue
		}
		if err := conn.WriteFrame(frame); err != nil {
			log.Printf("%s: scene npc map frame %s: %v", remote, p.configID, err)
			return
		}
		sent++
	}
	writeNPCMapDelimiter()
	writeNPCMapDelimiter()
	log.Printf("%s: scene npc map frames sent=%d (delimited)", remote, sent)
}
