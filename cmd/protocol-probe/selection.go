package main

import (
	"encoding/binary"
	"fmt"
	"github.com/local/9yin-go-server/internal/clientdata"
	"log"

	worldcore "github.com/local/9yin-go-server/internal/world"
)

// CLIENT_CUSTOMMSG_SELECT_TARGET：客户端每次点击目标发
// [i32 431, raw16 packed]，取消选择发 msgId=440（空）+ 431 obj(0,0)。
// 官方语义见 docs/fork-binary-message-inventory.md 与
// docs/combat-traffic-compare.md（49840 抓包钉死帧形）。
const clientCustomSelectTarget int32 = 431

// CLIENT_CUSTOMMSG_ATTACK_TARGET（298）：客户端请求自动攻击/接近目标。
// 官服应答 msg209 [接近距离15, obj(目标), 0]（a.pcapng 14.258/46.489
// 双样本）——这是客户端「自动战斗接近」的授权帧：点怪后客户端自动跑向
// 目标、跑入射程自动出招都依赖它；缺 209 时客户端发完 211 不起跑
// （2026-11-08 全天回归的最终根因：官服的"跑"是自动攻击接近系统，
// 不是冲锋链独有行为）。
const clientCustomAttackTarget int32 = 298

// handleAttackTargetCustom 处理 msg298 自动攻击目标请求：
// [298, i32 0, obj(目标)] → 回 msg209 [15, obj(目标), 0] 授权。
// 官服应答后续的 LogicState/移速帧（463/386/390）跟随各自系统，
// 不随 209 固定下发。
func handleAttackTargetCustom(world *sceneLifecycle, custom clientCustomMessage, remote string) error {
	if len(custom.Values) < 3 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomAttackTarget {
		return fmt.Errorf("attack target message mismatch: %v", custom.Values)
	}
	if custom.Values[2].Type != 8 {
		return fmt.Errorf("attack target argument must be object, got %s", custom.Values[2])
	}
	packed := binary.LittleEndian.Uint64(custom.Values[2].Raw[:])
	targetID := uint32(packed)
	targetOwner := uint32(packed >> 32)
	if targetID == 0 {
		return nil
	}
	world.run.mu.Lock()
	defer world.run.mu.Unlock()
	if world.closed {
		return nil
	}
	if _, exists := world.npcMeta(worldcore.EntityID(targetID)); !exists {
		// 未知对象：回零目标授权（客户端清接近态），不报错断链。
		frame, err := serverCustomIntMessage(209, customInt(15), customObject(0, 0), customInt(0))
		if err != nil {
			return err
		}
		return world.conn.WriteFrame(frame)
	}
	frame, err := serverCustomIntMessage(209, customInt(15), customObject(targetID, targetOwner), customInt(0))
	if err != nil {
		return fmt.Errorf("encode attack target ack: %w", err)
	}
	if err := world.conn.WriteFrame(frame); err != nil {
		return fmt.Errorf("send attack target ack: %w", err)
	}
	// 298 语义即攻击意图：镜像选中（211 选靶读取）。
	world.selectedObject = targetID
	world.selectedOwner = targetOwner
	log.Printf("%s: attack target ack id=%d owner=%d", remote, targetID, targetOwner)
	return nil
}

// AutoSelectObject 与 LastObject 配对：每次选择目标回两帧属性（按名
// 解析——fork 414/415，官服 2433 表天然落 416/417=官服 m431 应答原号
// §四-5）。
var propAutoSelectObject = clientdata.MustPlayerProperty("AutoSelectObject").Index // fork=415

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
	// 431 ack = 选中双帧（416/417）仅此而已——技能.pcapng（103.143.19.173
	// :7072 线）官服实证：选目标全程无 msg209，客户端照样锁定+自走。
	// 此前的「选目标即主动下发 msg209 接近授权」是 a.pcapng（7074 线）
	// 方言的试验性加戏；7072 线客户端上无此消息，多发反而可疑干扰锁定
	// 态（本服实测：客户端 431 风暴重选、211 恒短式无目标）。298 自请求
	// 的应答（handleAttackTargetCustom）保留——那是客户端主动要求的。
	log.Printf("%s: selected target id=%d owner=%d config=%s interaction=%s",
		remote, entity.id, entity.ownerID, entity.configID, entity.interaction)
	return nil
}

// writePlayerTargetLocked 回玩家对象的 LastObject + AutoSelectObject
// 属性帧（fork scene_lifecycle.go:1090-1094 writePlayerTargetLocked；
// 49840 抓包 s2c 0x10 双帧逐字节对应）。
func (s *sceneLifecycle) writePlayerTargetLocked(targetID, targetOwner uint32) error {
	// LastObject 按名解析：fork=414、官服=416。
	if err := s.conn.WriteFrame(sceneObjectObjectProperty(playerObjectID, playerOwnerID, uint32(clientdata.MustPlayerProperty("LastObject").Index), targetID, targetOwner)); err != nil {
		return err
	}
	return s.conn.WriteFrame(sceneObjectObjectProperty(playerObjectID, playerOwnerID, uint32(propAutoSelectObject), targetID, targetOwner))
}
