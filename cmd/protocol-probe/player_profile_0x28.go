package main

// 玩家档案 0x28 全量帧——mode=0x46 明文生成器（2026-11-12 定版）。
//
// 权威规范：docs/官服玩家档案0x28全量帧.md。官服登录进场簇在 0x0B 进场、
// 0x14/0x11 对与 0x29 记录批之后，向玩家下发一条 5KB 级 mode=0x47（LZ）
// 的 0x28 全量档案帧（count 292~393），obj/own=玩家实体，紧随其后才是
// 玩家 0x14/0x11 视图行爆发；换景时整帧重发（k.pcapng #1978 实证）。
// 本生成器以 mode=0x46 纯 memcpy 明文方言程序化构造（与 npcDetail0x28V3Frame
// 同族，官方 l.pcapng 自有 0x46 先例，任何会话字典状态安全）。
//
// 帧布局（官服 #132/#1978 逐字节定版）：
//
//	[0]=0x28 [1]=0x46 [2:6]=len-1 u32 [6:10]=明文体长 u32
//	[10:]=[obj u32][own u32][13×f32 变换块 52B][u16 字段数][字段×N]
//	变换块：posA=落点 x,y,z,yaw；posB 全零；scale=0（玩家模型缩放客户端
//	本地定，与 NPC 帧 1.25/5.0 不同）；保留零 + 尾 u32 1。
//	字段=[idx u16][值按 0x09 表列型]：t3=u32 t4=u64 t6=8B f64、
//	t7/t8=[len u32 含NUL][字节]。
//
// 官读 LZ 编码器四捷径（len=0→cstring、[idx_lo][FF] flag、[00跑+01 00]
// 分节标记、NUL 槽增量续段）全部不复刻——纯标准编码，字段稀疏可缺席
// 是官方语义（k#1978 相对 k#15 即少 idx162）。BaseFace（官服 len=0 变体）
// 与 ShortcutKey（176B 主段后接增量续段，单发主段为未经官方验证的截断
// 形态；键位另有 0x10 快捷配置通道）v1 均省略。
//
// 值来源：六包（c/k/登陆/登陆2/登陆-all）独立会话恒定值——Name/Face 为
// 官服遮蔽占位（跨角色恒定，非真名）；LastLineTask 族/复活点表为登录
// 场景常量。真机验证渲染通过后再接技能行（ConfigID+16 列模板）与本服
// 真值（复活点/键位）。门控：NINEYIN_PLAYER_PROFILE_0X28=1（默认关）。

import (
	"encoding/binary"
	"math"
	"os"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

// playerProfile0x28Enabled 控制玩家档案 0x28 帧下发（NINEYIN_PLAYER_PROFILE_0X28=1）。
func playerProfile0x28Enabled() bool {
	return os.Getenv("NINEYIN_PLAYER_PROFILE_0X28") == "1"
}

// 官服恒定字节值（六包跨会话逐字节一致；Name/Face 为遮蔽占位，见文件头）。
var (
	playerProfileNameWide = []byte{0x90, 0x5f, 0xa0, 0x50, 0xc3, 0x56, 0xad, 0x8f, 0x7c, 0x77, 0x00, 0x00}
	playerProfileFaceBlob = []byte("22222$22,\x106<\x01d\x00\x00\x002dddddd)\x00\x00\x00\x002)4T(!=)OEI\x01\x1dM'\x05\x17\x00")
	playerProfileHelpStep = []byte("q\x00\x00\x00dengzhen_02\x00")
	playerProfileRelive   = []byte("296,-89.360001,3.3\x00\x00\x0000,338.45401\x00\x00\x00.138\x00\x00\x00" +
		",ui_pos_Relive\x00\x00\x00\x00\x00\x0011,HomePoint\x00\x00\x00\x00\x00\x00\x00\x00A,0\x00")
	playerProfileDongHaiRelive = []byte("604,557.224976,371.278992,-408.0490\x00\x00\x000.355\x00" +
		"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00CJ1\x00" +
		"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00ene16A,1\x00")
)

// profileProp 一个档案字段：按名解析到当前 0x09 表条目 + 线值。
// 数值走 num（t1/t2/t3/t4/t6 均覆盖；t6 按 f64 写 8B），串走 raw
// （须含尾 NUL，len=含 NUL 总长——官方方言）。
type profileProp struct {
	name string
	num  uint64
	raw  []byte
}

// playerProfilePropSpecs v1 字段清单（41 项，官方 idx 序；省略 BaseFace/
// ShortcutKey，见文件头）。值=六包恒定官服常量。
var playerProfilePropSpecs = []profileProp{
	{"Name", 0, playerProfileNameWide},
	{"GroupID", 0xFFFFFFFF, nil},
	{"VisualPlayers", 80, nil},
	{"HomeFengShuiDate", 0, nil}, // f64=42952.0（日期序列），num 由 kind 决定
	{"CurHomeInOutType", 0, nil},
	{"GuildKillLevel", 1, nil},
	{"SchoolVoteKey", 13108398789392, nil},
	{"GuildSuitLevel", 3, nil},
	{"GuildMemberAuthStep", 4, nil},
	{"GuildMemberAuthSubStep", 5, nil},
	{"GuildHelpCDTime", 1405001020, nil},
	{"FestivalLFTime", 1424275200, nil},
	{"FestivalProgress", 2, nil},
	{"FestivalVersionID", 0, []byte("20150212BaiDaNian\x00")},
	{"DigTreasureTime", 244, nil},
	{"HelpStepID", 0, playerProfileHelpStep},
	{"LastLineTask", 24104, nil},
	{"LastLineTaskSchool", 46048, nil},
	{"LastLineTaskWY", 72342, nil},
	{"SkyHillStar", 19, nil},
	{"NewTerritoryCampIndex", 0, nil},
	{"NewTerritoryPKVal", 52, nil},
	{"CurOriginAchievePoint", 20, nil},
	{"CurPrizeAchieveIndex", 0, nil},
	{"IsHaveSecondWord", 1, nil},
	{"WordBeLock", 1, nil},
	{"IsOpenTimeProtect", 1, nil},
	{"NormalProtectTime", 0, nil},
	{"ClashProtectTime", 0, nil},
	{"VipValidTime", 1357481587, nil},
	{"VipCreateTime", 2062566, nil},
	{"VipFlag", 0, nil},
	{"Book", 0, []byte("book1\x00")},
	{"Type", 2, nil},
	{"Photo", 0, []byte("icon\\players\\boy01.png\x00")},
	{"Face", 0, playerProfileFaceBlob},
	{"Hair", 0, []byte("obj\\char\\b_hai\x00")},
	{"LeaveSchoolType", 0, nil},
	{"LastSchool", 0, []byte("school_junzita\x00")},
	{"RelivePositon", 0, playerProfileRelive},
	{"DongHaiRelivePositon", 0, playerProfileDongHaiRelive},
}

// playerProfileFengShuiDate 官服 HomeFengShuiDate 恒值（f64 日期序列，
// k/c/登陆 三包同值 42952.0）。
const playerProfileFengShuiDate = 42952.0

// profilePropEntry 按名解析档案字段条目。t6（f64 日期族）行被按名映射
// 排除（线型不可编码镜像 Java 语义，property_table.go:217），但档案帧
// 官服方言实发 8B——回落全表扫（末现，与 PlayerProperty 同语义）。
func profilePropEntry(name string) (clientdata.PropertyEntry, bool) {
	if entry, ok := clientdata.PlayerProperty(name); ok {
		return entry, true
	}
	var found clientdata.PropertyEntry
	ok := false
	for _, entry := range clientdata.OfficialPropertyEntries() {
		if entry.Name == name && entry.TypeCode == 6 {
			found, ok = entry, true
		}
	}
	return found, ok
}

// playerProfileSkillRowNames 技能行 17 列（官服 k#1978 模板序：ConfigID
// 打头、四倍率收尾）。同一 idx 帧内重复合法（每技能一组）——官服档案
// 帧计数含行字段实例（count 393 ≈ 44 基础 + 17×行 + 尾区）。
var playerProfileSkillRowNames = []string{
	"ConfigID", "CanUse", "StaticData", "Level", "VarPropNo", "PauseTime",
	"MaxLevel", "TotalFillValue", "Quality", "WuXing", "AConsumeMP",
	"TargetShapePara2", "CoolDownTime", "HPHealMul", "MPHealMul",
	"PhyDamageMul", "StiDamageMul",
}

// appendProfileSkillRows 按官服模板逐技能追加行字段，返回追加字段数。
// 真值来源与 View40 skillRowProperties 同源：learnedSkillView 档值 +
// combatSkillCatalog 补集（VarPropNo/CD/消耗/射程）；Quality/倍率固定族
// 对齐 View40（=1；官服 k#1978 样本 Quality=2 系该技能个案，全行取 1 与
// 本服 View40 行自洽）。行字段在当前表缺席时整行跳过该列（稀疏语义）。
func appendProfileSkillRows(body []byte, skills []learnedSkillView) ([]byte, int) {
	entries := make([]clientdata.PropertyEntry, len(playerProfileSkillRowNames))
	valid := make([]bool, len(playerProfileSkillRowNames))
	for i, name := range playerProfileSkillRowNames {
		if entry, ok := profilePropEntry(name); ok {
			entries[i], valid[i] = entry, true
		}
	}
	appendField := func(body []byte, i int, num uint64, raw []byte, f32 float32) []byte {
		body = binary.LittleEndian.AppendUint16(body, entries[i].Index)
		if raw != nil {
			body = binary.LittleEndian.AppendUint32(body, uint32(len(raw)))
			return append(body, raw...)
		}
		switch entries[i].TypeCode {
		case 1:
			return append(body, byte(num))
		case 2:
			return binary.LittleEndian.AppendUint16(body, uint16(num))
		case 3:
			return binary.LittleEndian.AppendUint32(body, uint32(num))
		case 4, 6:
			return binary.LittleEndian.AppendUint64(body, num)
		default: // 5=f32
			return binary.LittleEndian.AppendUint32(body, math.Float32bits(f32))
		}
	}
	count := 0
	for _, skill := range skills {
		var supplement *skillRowSupplement
		if catalog := installedCombatSkillCatalog; catalog != nil {
			supplement = catalog.skillRowData(skill.configID, skill.level)
		}
		varPropNo, coolDownMs, aConsumeMP, targetShapePara2 := int32(0), int32(0), int32(0), float32(0)
		if supplement != nil {
			varPropNo, coolDownMs, aConsumeMP, targetShapePara2 =
				supplement.varPropNo, supplement.coolDownMs, supplement.aConsumeMP, supplement.targetShapePara2
		}
		configRaw := append([]byte(skill.configID), 0)
		for i := range playerProfileSkillRowNames {
			if !valid[i] {
				continue
			}
			switch playerProfileSkillRowNames[i] {
			case "ConfigID":
				body = appendField(body, i, 0, configRaw, 0)
			case "CanUse":
				body = appendField(body, i, 1, nil, 0)
			case "StaticData":
				body = appendField(body, i, uint64(uint32(skill.staticData)), nil, 0)
			case "Level":
				body = appendField(body, i, uint64(uint8(clampU8(int(skill.level)))), nil, 0)
			case "VarPropNo":
				body = appendField(body, i, uint64(uint32(varPropNo)), nil, 0)
			case "PauseTime":
				body = appendField(body, i, uint64(uint32(skill.pauseTime)), nil, 0)
			case "MaxLevel":
				body = appendField(body, i, uint64(uint32(skill.maxLevel)), nil, 0)
			case "TotalFillValue":
				body = appendField(body, i, uint64(uint32(skill.total)), nil, 0)
			case "Quality":
				body = appendField(body, i, 1, nil, 0)
			case "WuXing":
				body = appendField(body, i, uint64(uint32(skillWuXing(skill.configID))), nil, 0)
			case "AConsumeMP":
				body = appendField(body, i, uint64(uint32(aConsumeMP)), nil, 0)
			case "TargetShapePara2":
				body = appendField(body, i, 0, nil, targetShapePara2)
			case "CoolDownTime":
				body = appendField(body, i, uint64(uint32(coolDownMs)), nil, 0)
			case "HPHealMul", "MPHealMul", "PhyDamageMul", "StiDamageMul":
				body = appendField(body, i, 0, nil, 1)
			}
			count++
		}
	}
	return body, count
}

// clampU8 等级档收 u8 线宽（Level 列 t1）。
func clampU8(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

// playerProfile0x28Frame 生成玩家档案 mode=0x46 明文帧，返回 (帧, 字段数)。
// pos=登录落点（posA）；scale/posB 按官服玩家形恒零。skills=已学集合
// （View40 merge 集同源），逐技能附官服模板 17 列行。当前表缺席的字段名
// 静默跳过（fork 表兼容，字段数按实际写入计——官服稀疏语义）。
func playerProfile0x28Frame(objectID, ownerID uint32, pos role.Position, skills []learnedSkillView) ([]byte, int) {
	body := make([]byte, 0, 640)
	body = binary.LittleEndian.AppendUint32(body, objectID)
	body = binary.LittleEndian.AppendUint32(body, ownerID)
	// 变换块 13 槽：posA=落点、posB 零、scale=0（玩家形）、保留零 ×3、尾 u32 1。
	for _, f := range []float32{pos.X, pos.Y, pos.Z, pos.Orient, 0, 0, 0, 0, 0, 0, 0, 0} {
		body = binary.LittleEndian.AppendUint32(body, math.Float32bits(f))
	}
	body = binary.LittleEndian.AppendUint32(body, 1)
	body = binary.LittleEndian.AppendUint16(body, 0) // 字段数占位（body[60:62]）

	count := 0
	for _, spec := range playerProfilePropSpecs {
		entry, ok := profilePropEntry(spec.name)
		if !ok {
			continue
		}
		body = binary.LittleEndian.AppendUint16(body, entry.Index)
		if spec.raw != nil {
			body = binary.LittleEndian.AppendUint32(body, uint32(len(spec.raw)))
			body = append(body, spec.raw...)
		} else {
			body = appendProfileNumeric(body, entry.TypeCode, spec)
		}
		count++
	}
	body, rowCount := appendProfileSkillRows(body, skills)
	count += rowCount
	// 字段数回填 body[60:62]（变换块定长 52B 保证）。
	binary.LittleEndian.PutUint16(body[60:], uint16(count))

	frame := make([]byte, 0, len(body)+10)
	frame = append(frame, 0x28, 0x46)
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)+9)) // len-1
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)))   // 明文体长
	frame = append(frame, body...)
	return frame, count
}

// appendProfileNumeric 按官方表列型码写数值。t6（f64 日期族）特殊：
// OfficialWireTypeOf 映射为不可编码 0，但档案帧官服方言实发 8B f64
// （HomeFengShuiDate=42952.0 实证），此处按线型直写。
func appendProfileNumeric(out []byte, typeCode int, spec profileProp) []byte {
	switch typeCode {
	case 1:
		return append(out, byte(spec.num))
	case 2:
		return binary.LittleEndian.AppendUint16(out, uint16(spec.num))
	case 3:
		return binary.LittleEndian.AppendUint32(out, uint32(spec.num))
	case 4:
		return binary.LittleEndian.AppendUint64(out, spec.num)
	case 5:
		return binary.LittleEndian.AppendUint32(out, math.Float32bits(float32(spec.num)))
	case 6:
		f := playerProfileFengShuiDate
		if spec.name != "HomeFengShuiDate" {
			f = float64(spec.num)
		}
		return binary.LittleEndian.AppendUint64(out, math.Float64bits(f))
	default:
		// 列型缺席（0/9 等）写 u32 兜底——调用侧已按 entry.Type!=0 过滤，
		// 此分支仅防御未来清单扩列。
		return binary.LittleEndian.AppendUint32(out, uint32(spec.num))
	}
}
