package main

import (
	"encoding/binary"
	"fmt"
	"log"

	worldcore "github.com/local/9yin-go-server/internal/world"
)

// CLIENT_CUSTOMMSG_SELECT_TARGET：客户端每次点击目标发
// [i32 431, raw16 packed]，取消选择发 msgId=440（空）+ 431 obj(0,0)。
// 官方语义见 docs/fork-binary-message-inventory.md 与
// docs/combat-traffic-compare.md（49840 抓包钉死帧形）。
const clientCustomSelectTarget int32 = 431

// AutoSelectObject 与 LastObject（表位 100）配对：fork 每次选择目标
// 回两帧属性（官方表 414/415；本仓协商表按追加约定放在表尾 230）。
const propAutoSelectObject uint16 = 230

// handleSelectTargetCustom 还原 fork selection.go:102-129：
// packed raw16 = objId(低 32) | owner(高 32)。清零目标走
// clearSelectedTarget 等价路径；有效目标校验实体存在后回
// LastObject + AutoSelectObject 双帧，并把选择镜像到
// selectedObject/selectedOwner 供 211 技能选靶读取。
func handleSelectTargetCustom(world *sceneLifecycle, custom clientCustomMessage, remote string) error {
	if len(custom.Values) < 2 {
		return fmt.Errorf("select target message needs id and object, got %d values", len(custom.Values))
	}
	if custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomSelectTarget {
		return fmt.Errorf("select target message id mismatch: %s", custom.Values[0])
	}
	if custom.Values[1].Type != 8 {
		return fmt.Errorf("select target argument must be object, got %s", custom.Values[1])
	}
	packed := binary.LittleEndian.Uint64(custom.Values[1].Raw[:])
	targetID := uint32(packed)
	targetOwner := uint32(packed >> 32)

	world.run.mu.Lock()
	defer world.run.mu.Unlock()
	if world.closed {
		return nil
	}
	if targetID == 0 {
		if err := world.writePlayerTargetLocked(0, 0); err != nil {
			return fmt.Errorf("clear player target: %w", err)
		}
		world.selectedObject = 0
		world.selectedOwner = 0
		log.Printf("%s: cleared selected target", remote)
		return nil
	}
	entity, exists := world.npcMeta(worldcore.EntityID(targetID))
	if !exists || entity.ownerID != targetOwner {
		return fmt.Errorf("select target unknown scene object id=%d owner=%d", targetID, targetOwner)
	}
	if err := world.writePlayerTargetLocked(entity.id, entity.ownerID); err != nil {
		return fmt.Errorf("select target id=%d owner=%d: %w", entity.id, entity.ownerID, err)
	}
	world.selectedObject = entity.id
	world.selectedOwner = entity.ownerID
	log.Printf("%s: selected target id=%d owner=%d config=%s interaction=%s",
		remote, entity.id, entity.ownerID, entity.configID, entity.interaction)
	return nil
}

// writePlayerTargetLocked 回玩家对象的 LastObject + AutoSelectObject
// 属性帧（fork scene_lifecycle.go:1090-1094 writePlayerTargetLocked；
// 49840 抓包 s2c 0x10 双帧逐字节对应）。
func (s *sceneLifecycle) writePlayerTargetLocked(targetID, targetOwner uint32) error {
	if err := s.conn.WriteFrame(sceneObjectObjectProperty(1, 1, 100, targetID, targetOwner)); err != nil {
		return err
	}
	return s.conn.WriteFrame(sceneObjectObjectProperty(1, 1, uint32(propAutoSelectObject), targetID, targetOwner))
}
