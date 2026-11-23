package entergen

// 0x0D 玩家物化帧（身体渲染模型集——初始号定版）。
//
// 源：captures/local 52626 会话 s2c #25（2386B/187 属性，docs/official/
// 初始号0x0D外观属性全表.txt + 初始号无装备身体模型定版.md，2433 表逐项
// 解析 consumed 2386/2386 定版）。帧布局：
//
//	[0]=0x0D [1:5]=obj [5:9]=own [9:13]=x [13:17]=y [17:21]=z
//	[21:25]=orient（f32）[25:61] 保留 36B 零 [61:63]=u16 属性数 + 属性区
//
// 定版要点（2026-11-23 官服穿/脱真机勘误后）：
//   - Cloth/Pants/Shoes 装备驱动（equipmodel.go）：槽空=jianghu000 裸身族
//     （官服实证「无装备≠空模型」），穿 {part}_{b|g}NNNN=jianghu 系号族；
//   - Hat = Hair 同值（无帽回填发型）；
//   - Photo/Face/Hair 属建角自定义（存档 appearance 槽 [1]/[2]/[4]，
//     捏脸二进制原样透传），档内缺席回退 captures/local 分支默认。
//
// 数值族：速度/物理常量取 #25 真机值；PhyDef 装备合计（官服 0 基线 +
// 布衣每件 +6）；五维/资源/内功从 rolestate 首本书推（官服白板口径
// RoleBaseProp + StarterSet ng_wd_001 加成）。

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// 建角自定义回退默认（captures/local 分支；Photo fork 线为 boy01）。
// 身体三件与 PhyDef 见 equipmodel.go（装备驱动）。
const (
	appearancePhotoB = `icon\players\boy03_create.png`
	appearancePhotoG = `icon\players\girl01_create.png`
	appearanceHairB  = `obj\char\b_hair\b_hair4`
	appearanceHairG  = `obj\char\g_hair\g_hair4`
	appearanceFace   = "e/330$33,'51\x1a4>=3333333333?5=<==\x19=33!F!1'6F3\t\x01" // 47B 捏脸方言形
)

// appearanceShortcutKey #25 真机键位串（659B；存档键位缺席时的回退）。
const appearanceShortcutKey = "1,87;2,83;3,81;4,69;5,65;6,68;7,32;8,90;9,144;10,111;11,2;12,16;13,17;14,88;15,18;" +
	"201,27;202,67;204,66;207,77;208,71;210,75;211,76;212,84;214,85;215,73;216,79;217,80;" +
	"224,123;225,86;226,78;227,72;228,74;229,89;" +
	"301,36;302,35;304,37;305,39;306,38;307,40;" +
	"401,13;402,33;403,34;404,1114146;405,1114145;502,9;" +
	"601,122;602,301989978;603,109;604,107;605,120;606,1114189;607,1114190;608,1114299;609,1114301;612,119;613,121;" +
	"701,189;702,187;712,49;713,50;714,51;715,52;716,53;717,54;718,55;719,56;720,57;721,48;" +
	"722,1114161;723,1114162;724,1114163;725,1114164;726,1114165;727,1114166;728,1114167;729,1114168;730,1114169;731,1114160;" +
	"732,112;733,113;734,114;735,115;736,116;"

// appearanceBufferList #25 登录 buff 串（新手标志 + 报伤 + 装备内功
// ng_wd_001——StarterSet 首本书 buff，建号态回放值）。
const appearanceBufferList = "buff_group_flag_101,local-1,1,285589731-64945070,0,1,0|" +
	"buf_baosd_02,local-2,0,285589731-64945070,0,1,0|" +
	"buf_ng_wd_001,local-4,3,285589731-64945070,1795354397233,1,0|"

// appearanceProp 一步一条（名→表项 + 值）。
func ap(name string, value PropertyValue) IndexedProperty {
	return IndexedProperty{Index: playerIdx(name), Name: name, Value: value}
}

// AppearanceFrame 0x0D 玩家物化帧（进场首帧 0x0B 之后、位置帧之前——
// 52626 实证序 0x0B→0x0D→0x1F）。属性序全按 #25 真机序。
func AppearanceFrame(ctx Context) ([]byte, error) {
	state := ctx.State
	female := state.Sex == 2
	photoDefault := appearancePhotoB
	if female {
		photoDefault = appearancePhotoG
	}
	cloth := EquipSlotModel(&state, EquipSlotCloth)
	pants := EquipSlotModel(&state, EquipSlotPants)
	shoes := EquipSlotModel(&state, EquipSlotShoes)
	apSlot := func(i int) string {
		if i < len(state.Appearance) {
			return state.Appearance[i]
		}
		return ""
	}
	photo, face := photoDefault, appearanceFace
	if v := apSlot(1); v != "" {
		photo = v
	}
	if v := apSlot(2); v != "" {
		face = v
	}
	hair := HairModel(&state)
	shortcut := ctx.Keybind
	if shortcut == "" {
		shortcut = appearanceShortcutKey
	}

	// 首本装备内功（建号态 = StarterSet[0]，CurNeiGong 缺席回退零书）。
	var book rolestate.NeiGongBook
	for i := range state.NeiGong {
		if state.NeiGong[i].ConfigID == state.CurNeiGong {
			book = state.NeiGong[i]
			break
		}
	}
	if book.ConfigID == "" && len(state.NeiGong) > 0 {
		book = state.NeiGong[0]
	}
	curNeiGong := state.CurNeiGong
	if curNeiGong == "" && len(state.NeiGong) > 0 {
		curNeiGong = state.NeiGong[0].ConfigID
	}

	// 五维/资源（官服白板 + 首本书 Add）。
	str := state.Str + book.StrAdd
	sta := state.Sta + book.StaAdd
	dex := state.Dex + book.DexAdd
	ing := state.Ing + book.IngAdd
	spi := state.Spi + book.SpiAdd
	maxHP, maxMP, maxParry := state.HP+book.MaxHPAdd, state.MP+book.MaxMPAdd, state.Parry+book.MaxParryAdd

	objHex := fmt.Sprintf("%X", ctx.Identity.ObjectID)
	ownHex := fmt.Sprintf("%X", ctx.Identity.OwnerID)
	sexWire := byte(0)
	if female {
		sexWire = 1
	}

	props := []IndexedProperty{
		ap("Type", ByteValue(2)),
		ap("Sex", ByteValue(sexWire)),
		ap("Photo", StringValue(photo)),
		ap("Camp", ByteValue(0)),
		ap("Race", Int32Value(0)),
		ap("Job", StringValue("")),
		ap("Level", ByteValue(1)),
		ap("ShortcutKey", StringValue(shortcut)),
		ap("PosiX", Float32Value(float32(state.Position[0]))),
		ap("PosiY", Float32Value(float32(state.Position[1]))),
		ap("PosiZ", Float32Value(float32(state.Position[2]))),
		ap("Orient", Float32Value(float32(state.Position[3]))),
		ap("Hair", StringValue(hair)),
		ap("Face", StringValue(face)),
		ap("Cloth", StringValue(cloth)),
		ap("Pants", StringValue(pants)),
		ap("Shoes", StringValue(shoes)),
		ap("ActionSet", StringValue("0h")),
		ap("State", StringValue("stand")),
		ap("FloatingState", Int32Value(0)),
		ap("LogicState", ByteValue(0)),
		ap("JumpSpeed", Float32Value(8.0)),
		ap("MoveSpeed", Float32Value(3.9)),
		ap("WalkSpeed", Float32Value(1.28)),
		ap("RunSpeed", Float32Value(3.9)),
		ap("SpeedRatio", Int32Value(100)),
		ap("CantJump", ByteValue(0)),
		ap("HPRatio", Int32Value(100)),
		ap("MPRatio", Int32Value(100)),
		ap("MaxHP", Int32Value(int32(state.HP))),
		ap("MaxMP", Int32Value(int32(state.MP))),
		ap("HP", Int32Value(int32(maxHP))),
		ap("MP", Int32Value(int32(maxMP))),
		ap("HitHP", Int32Value(int32(state.HP))),
		ap("HitHPRatio", Int32Value(100)),
		ap("QingGongPoint", Int32Value(150)),
		ap("MaxQingGongPoint", Int32Value(150)),
		ap("MaxQingGongPointAdd", Int32Value(0)),
		ap("ParryRatio", Int32Value(100)),
		ap("InParry", Int32Value(0)),
		ap("CantParry", Int32Value(0)),
		ap("ParryType", Int32Value(0)),
		ap("InManualParry", Int32Value(0)),
		ap("ParryValue", Int32Value(int32(maxParry))),
		ap("ParryUpSpeed", Int32Value(4)),
		ap("NeigongPKStatus", WordValue(0)),
		ap("Dead", ByteValue(0)),
		ap("CurSkillID", StringValue("")),
		ap("CurSkillTarget", ObjectValue(0, 0)),
		ap("ModifySkillLockTime", Int32Value(200)),
		ap("BufferListStr", StringValue(appearanceBufferList)),
		ap("Gravity", Float32Value(9.8)),
		ap("GravityAdd", Float32Value(0)),
		ap("DropHeightPub", Float32Value(100000)),
		ap("RunSpeedAdd", Float32Value(0)),
		ap("JumpSpeedAdd", Float32Value(0)),
		ap("DriftSpeed", Float32Value(7.5)),
		ap("DriftSpeedAdd", Float32Value(0)),
		ap("DriftJumpSpeed", Float32Value(6.0)),
		ap("DriftJumpSpeedAdd", Float32Value(0)),
		ap("SndJumpSpeed", Float32Value(6.0)),
		ap("SndJumpSpeedAdd", Float32Value(0)),
		ap("ThdJumpSpeed", Float32Value(6.0)),
		ap("ThdJumpSpeedAdd", Float32Value(0)),
		ap("AirRushSpeed", Float32Value(15.0)),
		ap("AirRushSpeedAdd", Float32Value(0)),
		ap("AirRushDist", Float32Value(12.0)),
		ap("AirRushDistAdd", Float32Value(0)),
		ap("LandRushSpeed", Float32Value(25.0)),
		ap("LandRushSpeedAdd", Float32Value(0)),
		ap("LandRushDist", Float32Value(6.5)),
		ap("LandRushDistAdd", Float32Value(0)),
		ap("ClimbSpeed", Float32Value(7.5)),
		ap("ClimbSpeedAdd", Float32Value(0)),
		ap("GRushRange", Float32Value(5.0)),
		ap("GRushRangeAdd", Float32Value(1.5)),
		ap("ARushRange", Float32Value(8.0)),
		ap("ARushRangeAdd", Float32Value(6.5)),
		ap("WRushRange", Float32Value(8.0)),
		ap("WRushRangeAdd", Float32Value(6.5)),
		ap("AutoSelectObject", ObjectValue(0, 0)),
		ap("BufferInfo1", StringValue(fmt.Sprintf("6747,%s,%s,1,0,0", objHex, ownHex))),
		ap("BufferInfo2", StringValue(fmt.Sprintf("1E37,%s,%s,0,0,0", objHex, ownHex))),
		ap("BufferInfo3", StringValue("")),
		ap("BufferInfo4", StringValue(fmt.Sprintf("1742,%s,%s,3,C39342C7231,1", objHex, ownHex))),
		ap("BufferInfo5", StringValue("")),
		ap("BufferInfo6", StringValue("")),
		ap("BufferInfo7", StringValue("")),
		ap("BufferInfo8", StringValue("")),
		ap("BufferInfo9", StringValue("")),
		ap("BufferInfo10", StringValue("")),
		ap("BufferInfo11", StringValue("")),
		ap("BufferInfo12", StringValue("")),
		ap("BufferInfo13", StringValue("")),
		ap("BufferInfo14", StringValue("")),
		ap("BufferInfo15", StringValue("")),
		ap("BufferInfo16", StringValue("")),
		ap("BufferInfo17", StringValue("")),
		ap("BufferInfo18", StringValue("")),
		ap("BufferInfo19", StringValue("")),
		ap("BufferInfo20", StringValue("")),
		ap("BufferInfo21", StringValue("")),
		ap("BufferInfo22", StringValue("")),
		ap("BufferInfo23", StringValue("")),
		ap("BufferInfo24", StringValue("")),
		ap("CurNeiGong", StringValue(curNeiGong)),
		ap("Faculty", Int32Value(int32(state.FacultySeed))),
		ap("FacultyState", Int32Value(0)),
		ap("FacultyStyle", Int32Value(1)),
		ap("FacultyName", StringValue(curNeiGong)),
		ap("CurFillValue", Int32Value(0)),
		ap("TotalFillValue", Int32Value(int32(book.Total))),
		ap("CurLevel", Int32Value(int32(book.Level))),
		ap("FillSpeed", Int32Value(0)),
		ap("LiveGroove_1", Int32Value(0)),
		ap("LiveGroove_2", Int32Value(0)),
		ap("LiveGroove_3", Int32Value(0)),
		ap("LiveGroove_4", Int32Value(0)),
		ap("LiveGroove_5", Int32Value(0)),
		ap("LiveSpeedPower", Int32Value(0)),
		ap("HoldFaculty", Int32Value(0)),
		ap("TeamFacultyState", Int32Value(0)),
		ap("TeamFacultyValue", Int32Value(0)),
		ap("TeamFacultyLast", Int32Value(0)),
		ap("PowerValue", Int32Value(int32(book.Power))),
		ap("MaxPowerValue", Int32Value(int32(book.MaxPower))),
		ap("SP", Int32Value(int32(state.SP))),
		ap("Str", Int32Value(int32(str))),
		ap("Sta", Int32Value(int32(sta))),
		ap("Dex", Int32Value(int32(dex))),
		ap("Ing", Int32Value(int32(ing))),
		ap("Spi", Int32Value(int32(spi))),
		ap("MeleePower", Int32Value(80)),
		ap("ShotPower", Int32Value(80)),
		ap("MagicPower", Int32Value(80)),
		ap("MeleePowerAdd", Int32Value(int32(str))),
		ap("ShotPowerAdd", Int32Value(int32(dex))),
		ap("MagicPowerAdd", Int32Value(int32(ing))),
		ap("PhyHit", Int32Value(50)),
		ap("PhyHitAdd", Int32Value(int32(dex))),
		ap("MagicHit", Int32Value(50)),
		ap("MagicHitAdd", Int32Value(int32(dex))),
		ap("Dodge", Int32Value(35)),
		ap("DodgeAdd", Int32Value(int32(dex))),
		ap("PhyVa", Int32Value(16)),
		ap("PhyVaAdd", Int32Value(int32(str))),
		ap("MagicVa", Int32Value(16)),
		ap("MagicVaAdd", Int32Value(int32(dex))),
		ap("MaxParry", Int32Value(int32(state.Parry))),
		ap("PhyDef", Int32Value(EquipPhyDef(&state))),
		ap("StiDef", Int32Value(50)),
		ap("NegDef", Int32Value(50)),
		ap("MasDef", Int32Value(50)),
		ap("JujDef", Int32Value(50)),
		ap("MinMeleeDamage", Int32Value(20)),
		ap("MaxMeleeDamage", Int32Value(35)),
		ap("MaxHPAdd", Int32Value(int32(book.MaxHPAdd))),
		ap("MaxMPAdd", Int32Value(int32(book.MaxMPAdd))),
		ap("MaxParryAdd", Int32Value(int32(book.MaxParryAdd))),
		ap("MinMagicDefAdd", Int32Value(int32(book.MinMagicDefAdd))),
		ap("MaxLevel", Int32Value(int32(book.MaxLevel))),
		ap("CurNeiGongWuXing", Int32Value(5)), // ng_wd_001 五行土（StarterSet 首本书）
		ap("WuJiPoint", Int32Value(0)),
		ap("MaxWuJiPoint", Int32Value(0)),
		ap("CurJingMai", StringValue("")),
		ap("LastJingMai", StringValue("")),
		ap("JingMaiTotalLevel", Int32Value(0)),
		ap("jmActCount", Int32Value(3)),
		ap("ZhenQiDayValue", Int32Value(500)),
		ap("ZQPLValue", Int32Value(0)),
		ap("ZQActlValue", Int32Value(0)),
		ap("ZQUnUsedValue", Int32Value(0)),
		ap("Hat", StringValue(hair)), // 无帽回填发型（定版：Hat = Hair 同值）
		ap("Name", WideValue(ctx.RoleName)),
		ap("LastObject", ObjectValue(0, 0)),
		ap("CollideRadius", Float32Value(0.9)),
		ap("State", StringValue("stand")),
		ap("ActionSpeed", Float32Value(1.0)),
		ap("HPRatio", Int32Value(100)),
		ap("ParryRatio", Int32Value(100)),
		ap("HitHPRatio", Int32Value(100)),
		ap("InParry", Int32Value(0)),
		ap("CantParry", Int32Value(0)),
		ap("ParryType", Int32Value(0)),
		ap("InManualParry", Int32Value(0)),
		ap("ParryValue", Int32Value(int32(maxParry))),
		ap("ParryUpSpeed", Int32Value(4)),
	}

	msg := make([]byte, 63, 63+len(props)*10)
	msg[0] = 0x0D
	binary.LittleEndian.PutUint32(msg[1:], ctx.Identity.ObjectID)
	binary.LittleEndian.PutUint32(msg[5:], ctx.Identity.OwnerID)
	binary.LittleEndian.PutUint32(msg[9:], math.Float32bits(float32(state.Position[0])))
	binary.LittleEndian.PutUint32(msg[13:], math.Float32bits(float32(state.Position[1])))
	binary.LittleEndian.PutUint32(msg[17:], math.Float32bits(float32(state.Position[2])))
	binary.LittleEndian.PutUint32(msg[21:], math.Float32bits(float32(state.Position[3])))
	// [25:61] 保留区 36B 零（#25/#23 两会话均实证全零）。
	binary.LittleEndian.PutUint16(msg[61:], uint16(len(props)))
	for _, prop := range props {
		msg = appendProperty(msg, prop)
	}
	return msg, nil
}
