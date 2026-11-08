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
//	 窄串技能ID, 窄串script标签("SkillLock"/"SkillNormal"), 窄串受害者模型,
//	 宽串攻击者名, i32 0]
//
// script 标签取技能编译口径 scriptLabel()（锁定族一律 SkillLock——
// 52905 抓包 myjf01 系 'SkillLock' 实证；此前恒 "SkillNormal" 是占位）。
func skillHitDamageFrame(victimID, victimOwner uint32, damage int32, skillID, scriptLabel, victimModel, attackerWideName string) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(victimID, victimOwner),
		customInt(17),
		customInt(damage),
		customInt(0),
		customInt(0),
		customString(skillID),
		customString(scriptLabel),
		customString(victimModel),
		customWideString(attackerWideName),
		customInt(0),
	)
}

// healFloatFrame 即 msg24 治疗/回蓝飘字（10 值形态，官服打坐
// 回复九跳逐帧实证，FsGame 官服抓包 00:11:50.408–00:12:00.882）：
//
//	[24, obj(受者), i32 16|18, i32 实际回复, i32 溢出截断, i32 0,
//	 窄串来源ID, 窄串script标签, 宽串受者名, 宽串受者名]
//
// 补满末跳官服实发 (162,1478)：实际 162 + 截断 1478 = 名义 1640；
// 未触顶跳恒 (回复量, 0)。同型帧亦用于 buff 回复（NPC
// buff_clone_recovery_5 飘字实证：脚本标签 "Buffer"、来源位为
// buff 名），script 标签区分来源族；打坐技能来源用 "SkillNormal"。
// 宽名双写为施法者/受者两位；打坐自愈时两者同为本人。
// kind：16=HP 治疗飘字、18=MP 回蓝飘字（49.4.14.150:7074 实时
// 实证打坐 MP 每跳 [24, obj, 18, 519, 0, 0, zs_default_01,
// SkillNormal, 名, 名]，与 c1=16 同十值形态）。
func healFloatFrame(targetID, targetOwner uint32, kind, applied, wasted int32, sourceID, scriptLabel, targetWideName string) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(targetID, targetOwner),
		customInt(kind),
		customInt(applied),
		customInt(wasted),
		customInt(0),
		customString(sourceID),
		customString(scriptLabel),
		customWideString(targetWideName),
		customWideString(targetWideName),
	)
}

// sitcrossActionFrame 即打坐进入/退出动作飘字（msg24 4 值形态）：
//
//	[24, obj(自己), i32 40|41, 窄串"dazuo_old"]
//
// 官服打坐进出簇与 LogicState 单属性帧同毫秒成对出现，c1=40 进、
// 41 出（FsGame 官服抓包 102 进出全量成对实证）；旁观者同广播，
// 这也是别人打坐时你能看到的动作提示来源。
func sitcrossActionFrame(id, owner uint32, enter bool) ([]byte, error) {
	action := int32(41)
	if enter {
		action = 40
	}
	return serverCustomIntMessage(24,
		customObject(id, owner),
		customInt(action),
		customString("dazuo_old"),
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
//
// 2026-11-07 战斗重写对齐（Java FightMessages.skillAction + 抓包 52905
// #3005 逐字节）：外壳一律 0x1E（抓包 42 份 0 次 msg20 走 0x27），尾两值
// 官方恒 i32 0 / f32 0.0（此前 1/1 是本地占位）。
func nativeSkillActionFrameFor(casterID, casterOwner uint32, skillID string, targetID, targetOwnerID uint32) ([]byte, error) {
	if skillID == "" {
		return nil, fmt.Errorf("native skill action ID is empty")
	}
	return serverCustomIntMessage(
		serverNativeSkillActionMessage,
		customObject(casterID, casterOwner),
		customString(skillID),
		customObject(targetID, targetOwnerID),
		customInt(0),
		customFloat32(0),
	)
}

// npcAttackHitFrame 即 msg24 NPC 攻击形态（九值）：
//
//	[24, obj(玩家), 7, i32 伤害, obj(NPC), 窄串NPC模型, 宽串玩家名,
//	 宽串NPC名, i32 20(名义攻击力)]
//
// 伤害=实扣值（参考服折算恒 1）、尾值 20=名义攻击力（Java
// npcDamageFloat、55866 抓包实证；NPC 名 Java/官方走宽串——抓包
// 12141 的窄串方言是另一台参考服的分歧，按 Java 口径）。
func npcAttackHitFrame(playerID, playerOwner uint32, damage int32, npcID, npcOwner uint32, npcModel, playerWideName, npcWideName string) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(playerID, playerOwner),
		customInt(7),
		customInt(damage),
		customObject(npcID, npcOwner),
		customString(npcModel),
		customWideString(playerWideName),
		customWideString(npcWideName),
		customInt(20),
	)
}

// npcAttackSkillJN 是 NPC 攻击动画帧携带的招式串（Java NpcCombatAi.
// NPC_ATTACK_SKILL 同值；参考服 55866 追风士 jn_mon053_02 逐位对齐。
// 抓包 12141 另见 jn_mon043_02（按 NPC 模型取招式的官方列未定位，
// 统一走 Java 常量）。此前 Go 发 "SkillNormal" 是本地占位——msg20 的
// 技能串要能索引客户端招式动画表，"SkillNormal" 不是招式 ID。
const npcAttackSkillJN = "jn_mon053_02"

// skillBuffFrame 即 msg169 技能 buff 九值形态（Java FightMessages.skillBuff）：
//
//	[169, i32 addRemove(0=摘/1=挂), 窄串buffConfigId, obj(施法者), obj(承载者),
//	 宽串名, 宽串名, i32 0, i32 0]
//
// 武技族首跳摘 buf_<技能>_01（addRemove=0）与击杀红名标记（addRemove=1,
// buf_fixed_red）共用本形态。
func skillBuffFrame(addRemove int32, casterID, casterOwner, bearerID, bearerOwner uint32, buffConfigID, playerName string) ([]byte, error) {
	return serverCustomIntMessage(169,
		customInt(addRemove),
		customString(buffConfigID),
		customObject(casterID, casterOwner),
		customObject(bearerID, bearerOwner),
		customWideString(playerName),
		customWideString(playerName),
		customInt(0),
		customInt(0),
	)
}

// targetDebuffTimedFrame 即 msg169 目标 debuff 时间形态（Java
// FightMessages.targetDebuffTimed；参考服 55866 myjf08 每跳实证）：
//
//	[169, i32 1, 窄串buff串, obj(承载NPC), obj(施法者玩家), 宽串"",
//	 宽串施法者名, i64 到期unix毫秒, i32 0]
func targetDebuffTimedFrame(buffConfigID string, bearerID, bearerOwner, casterID, casterOwner uint32, casterName string, endEpochMs int64) ([]byte, error) {
	return serverCustomIntMessage(169,
		customInt(1),
		customString(buffConfigID),
		customObject(bearerID, bearerOwner),
		customObject(casterID, casterOwner),
		customWideString(""),
		customWideString(casterName),
		customInt64(endEpochMs),
		customInt(0),
	)
}

// glideTimedBuffFrame 即 msg169 缓降九值时间形态（Java
// FightMessages.timedBuff；52795 抓包 buf_qg_light 实证）：
//
//	[169, i32 addRemove, 窄串"buf_qg_light", obj(施), obj(承),
//	 宽名, 宽名, i64 到期unix毫秒, i32 2048]
//
// ⚠️ addRemove 极性与六值 skillBuffFrame 相反：本形态 0=挂/1=摘。
// v7 是普通 Unix 毫秒（非 BufferInfo 槽串的 FILETIME 口径），v8 是
// buff_static[218].Category。
func glideTimedBuffFrame(addRemove int32, endEpochMs int64, bearerID, bearerOwner uint32, playerName string) ([]byte, error) {
	return serverCustomIntMessage(169,
		customInt(addRemove),
		customString(qingGongGlideBuffID),
		customObject(bearerID, bearerOwner),
		customObject(bearerID, bearerOwner),
		customWideString(playerName),
		customWideString(playerName),
		customInt64(endEpochMs),
		customInt(glideBuffCategory),
	)
}

// wujiComboFloatFrame 即 msg24 武技连击段（c1=71 十值形态；Java
// FightMessages.wujiComboFloat、参考服 55866 myjf08 实证）：
//
//	[24, obj(受害), 71, obj(第二目标), 窄串"", i32 1, obj(攻击者),
//	 f64 30.0, f64 0.0, 窄串段名("" / "<LevelupRelevance>_2/_3/_4")]
//
// 30.0 常量含义未钉死（疑似段间隔秒），逐字节复刻官方。
func wujiComboFloatFrame(victimID, victimOwner, secondID, secondOwner, casterID, casterOwner uint32, wujiName string) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(victimID, victimOwner),
		customInt(71),
		customObject(secondID, secondOwner),
		customString(""),
		customInt(1),
		customObject(casterID, casterOwner),
		customFloat64(30.0),
		customFloat64(0.0),
		customString(wujiName),
	)
}

// playerStandStateFrame 是动作总时长后的 State(idx19)="stand" 复位帧
// （§3 行 9；区别于 idx183 的动作段 State 帧）。
func playerStandStateFrame() ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propState, Name: "State", Value: clientdata.StringValue("stand")},
	})
}
