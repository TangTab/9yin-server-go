package main

// 战斗命中表现帧族（0x13 战斗关系 / msg24 伤害飘字 / 0x2D 受击特效 /
// msg18 受击动作 / msg205 命中确认 / NPC 三段攻击帧）。
//
// 规格双源钉死：F 盘 native-menu.exe 反汇编（writeSkillHitEvents /
// fightRelationDamageFrame / writeNPCAttackFrames）与 Java 仓抓包文档
// docs/combat-skill-211.md §4（capture 127-0-0-1-62433）。两源冲突以
// 抓包为准；0x13 行/列语义按字面复刻（恒 1/1）。

import (
	"encoding/binary"
	"fmt"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// fightRelationDamageFrame 编码 FxNet2 ServerRecordGrid (0x13) 的
// fight_relation_rec 单格累计伤害（21B）：
//
//	13 | u8 0 | u32 玩家obj@2 | u32 owner@6 | u16 记录索引@10
//	  | u16 0x0001 @12 | u16 0x0001 @14 | u8 0x00 @16 | u32 累计伤害@17
//
// 累计值是「该玩家对该目标」的逐击累加（抓包 143→210→277 每击 +67）；
// row/col 恒 1/1，col0(obj) 从不写。
func fightRelationDamageFrame(playerObjectID, playerOwnerID uint32, cumulativeDamage int32) []byte {
	msg := make([]byte, 21)
	msg[0] = 0x13
	msg[1] = 0
	binary.LittleEndian.PutUint32(msg[2:], playerObjectID)
	binary.LittleEndian.PutUint32(msg[6:], playerOwnerID)
	binary.LittleEndian.PutUint16(msg[10:], recordFightRelation)
	binary.LittleEndian.PutUint16(msg[12:], 1)
	binary.LittleEndian.PutUint16(msg[14:], 1)
	msg[16] = 0
	binary.LittleEndian.PutUint32(msg[17:], uint32(cumulativeDamage))
	return msg
}

// skillHitDamageFrame 即 msg24 技能形态伤害飘字（11 值）：
//
//	[24, obj(受害者), i32 17, i32 伤害值, i32 0, i32 0,
//	 窄串技能ID, 窄串"SkillNormal", 窄串受害者模型, 宽串攻击者名, i32 0]
func skillHitDamageFrame(victimID, victimOwner uint32, damage int32, skillID, victimModel, attackerWideName string) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(victimID, victimOwner),
		customInt(17),
		customInt(damage),
		customInt(0),
		customInt(0),
		customString(skillID),
		customString("SkillNormal"),
		customString(victimModel),
		customWideString(attackerWideName),
		customInt(0),
	)
}

// hurtEffectFrame 编码 0x2D 受击特效（25B）：
//
//	2D | 01 | 00 | u32 受害者obj@3 | u32 owner@7 | u16 2 @11 | u16 28 @13
//	  | u32 受害者HPRatio@15 | u16 112 @19 | u32 受害者HPRatio@21
//
// 28/112 两常量全程不变；HPRatio=受击后血量百分比整数，出现两次。
func hurtEffectFrame(victimID, victimOwner uint32, hpRatio int32) []byte {
	msg := make([]byte, 25)
	msg[0] = 0x2D
	msg[1] = 1
	msg[2] = 0
	binary.LittleEndian.PutUint32(msg[3:], victimID)
	binary.LittleEndian.PutUint32(msg[7:], victimOwner)
	binary.LittleEndian.PutUint16(msg[11:], 2)
	binary.LittleEndian.PutUint16(msg[13:], 28)
	binary.LittleEndian.PutUint32(msg[15:], uint32(hpRatio))
	binary.LittleEndian.PutUint16(msg[19:], 112)
	binary.LittleEndian.PutUint32(msg[21:], uint32(hpRatio))
	return msg
}

// skillHurtActionFrame 即 msg18 通用受击动作：
// [18, obj(目标), 窄串"hurt_half", i32 0, obj(攻击者)]。
func skillHurtActionFrame(victimID, victimOwner, attackerID, attackerOwner uint32) ([]byte, error) {
	return serverCustomIntMessage(18,
		customObject(victimID, victimOwner),
		customString("hurt_half"),
		customInt(0),
		customObject(attackerID, attackerOwner),
	)
}

// skillHitConfirmFrame 即 msg205 命中确认：
// [205, obj(施法者), 窄串技能ID, obj(目标)]。
func skillHitConfirmFrame(casterID, casterOwner uint32, skillID string, victimID, victimOwner uint32) ([]byte, error) {
	return serverCustomIntMessage(205,
		customObject(casterID, casterOwner),
		customString(skillID),
		customObject(victimID, victimOwner),
	)
}

// npcCastPreparationFrame 即 msg337 NPC 形态施法预备（与玩家
// casterTransformSyncFrame 同布局，对象换 NPC）：
// [337, obj(NPC), f32 x, f32 y, f32 z, f32 orient, i32 0]。
func npcCastPreparationFrame(npcID, npcOwner uint32, x, y, z, orient float32) ([]byte, error) {
	return serverCustomIntMessage(337,
		customObject(npcID, npcOwner),
		customFloat32(x), customFloat32(y), customFloat32(z),
		customFloat32(orient),
		customInt(0),
	)
}

// nativeSkillActionFrameFor 是 nativeSkillActionFrame 的施法者泛化版：
// NPC 攻击动画帧需要以 NPC 为施法者走同一条 msg20 路线。
func nativeSkillActionFrameFor(casterID, casterOwner uint32, skillID string, targetID, targetOwnerID uint32) ([]byte, error) {
	if skillID == "" {
		return nil, fmt.Errorf("native skill action ID is empty")
	}
	return serverModernCustomIntMessage(
		serverNativeSkillActionMessage,
		customObject(casterID, casterOwner),
		customString(skillID),
		customObject(targetID, targetOwnerID),
		customInt(1),
		customFloat32(1),
	)
}

// npcAttackHitFrame 即 msg24 NPC 攻击形态（9 值）：
//
//	[24, obj(玩家), 7, 1, obj(NPC), 窄串NPC模型, 宽串玩家名, 宽串NPC名, 20]
func npcAttackHitFrame(playerID, playerOwner, npcID, npcOwner uint32, npcModel, playerWideName, npcWideName string) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(playerID, playerOwner),
		customInt(7),
		customInt(1),
		customObject(npcID, npcOwner),
		customString(npcModel),
		customWideString(playerWideName),
		customWideString(npcWideName),
		customInt(20),
	)
}

// npcDefaultAttackSkill 是 NPC 攻击动画帧携带的招式串。官方表无
// NPC 专属攻击技能列，F 盘参考服对普攻动画统一走 SkillNormal 脚本。
const npcDefaultAttackSkill = "SkillNormal"

// playerStandStateFrame 是动作总时长后的 State(idx19)="stand" 复位帧
// （§3 行 9；区别于 idx183 的动作段 State 帧）。
func playerStandStateFrame() ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propState, Name: "State", Value: clientdata.StringValue("stand")},
	})
}
