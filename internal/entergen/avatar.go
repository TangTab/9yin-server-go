package entergen

// 玩家档案 0x28 全量帧生成器（b-line player_profile_0x28.go 移植）。
//
// 布局（外服 #132/#1978 逐字节定版）：
//	[0]=0x28 [1]=mode(0x46 明文) [2:6]=len-1 [6:10]=体长
//	[10:]=[obj][own][13×f32 变换块 52B][u16 字段数][字段×N]
// 字段=[idx u16][值按 0x09 表列型]。值=六包恒定外服常量 + rolestate
// 真值（Photo 按性别/技能行/键位/装备内功）。

import (
	"encoding/binary"
	"math"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

// 外服恒定字节值（六包跨会话逐字节一致；Name/Face 为遮蔽占位）。
var (
	profileNameWide = []byte{0x90, 0x5f, 0xa0, 0x50, 0xc3, 0x56, 0xad, 0x8f, 0x7c, 0x77, 0x00, 0x00}
	profileFaceBlob = []byte("22222$22,\x106<\x01d\x00\x00\x002dddddd)\x00\x00\x00\x002)4T(!=)OEI\x01\x1dM'\x05\x17\x00")
	profileHelpStep = []byte("q\x00\x00\x00dengzhen_02\x00")
	profileRelive   = []byte("296,-89.360001,3.3\x00\x00\x0000,338.45401\x00\x00\x00.138\x00\x00\x00" +
		",ui_pos_Relive\x00\x00\x00\x00\x00\x0011,HomePoint\x00\x00\x00\x00\x00\x00\x00\x00A,0\x00")
	profileDongHaiRelive = []byte("604,557.224976,371.278992,-408.0490\x00\x00\x000.355\x00" +
		"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00CJ1\x00" +
		"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00ene16A,1\x00")
)

const profileFengShuiDate = 42952.0

type profileProp struct {
	name string
	num  uint64
	f64  float64
	raw  []byte
}

// profilePropSpecs 基础字段清单（b-line 41 项，官方 idx 序）。
var profilePropSpecs = []profileProp{
	{"Name", 0, 0, profileNameWide},
	{"GroupID", 0xFFFFFFFF, 0, nil},
	{"VisualPlayers", 80, 0, nil},
	{"HomeFengShuiDate", 0, profileFengShuiDate, nil},
	{"CurHomeInOutType", 0, 0, nil},
	{"GuildKillLevel", 1, 0, nil},
	{"SchoolVoteKey", 13108398789392, 0, nil},
	{"GuildSuitLevel", 3, 0, nil},
	{"GuildMemberAuthStep", 4, 0, nil},
	{"GuildMemberAuthSubStep", 5, 0, nil},
	{"GuildHelpCDTime", 1405001020, 0, nil},
	{"FestivalLFTime", 1424275200, 0, nil},
	{"FestivalProgress", 2, 0, nil},
	{"FestivalVersionID", 0, 0, []byte("20150212BaiDaNian\x00")},
	{"DigTreasureTime", 244, 0, nil},
	{"HelpStepID", 0, 0, profileHelpStep},
	{"LastLineTask", 24104, 0, nil},
	{"LastLineTaskSchool", 46048, 0, nil},
	{"LastLineTaskWY", 72342, 0, nil},
	{"SkyHillStar", 19, 0, nil},
	{"NewTerritoryCampIndex", 0, 0, nil},
	{"NewTerritoryPKVal", 52, 0, nil},
	{"CurOriginAchievePoint", 20, 0, nil},
	{"CurPrizeAchieveIndex", 0, 0, nil},
	{"IsHaveSecondWord", 1, 0, nil},
	{"WordBeLock", 1, 0, nil},
	{"IsOpenTimeProtect", 1, 0, nil},
	{"NormalProtectTime", 0, 0, nil},
	{"ClashProtectTime", 0, 0, nil},
	{"VipValidTime", 1357481587, 0, nil},
	{"VipCreateTime", 2062566, 0, nil},
	{"VipFlag", 0, 0, nil},
	{"Book", 0, 0, []byte("book1\x00")},
	{"Type", 2, 0, nil},
	{"Photo", 0, 0, []byte("icon\\players\\boy01.png\x00")}, // 调用方按性别覆盖
	{"Face", 0, 0, profileFaceBlob},
	{"Hair", 0, 0, []byte("obj\\char\\b_hai\x00")},
	{"LeaveSchoolType", 0, 0, nil},
	{"LastSchool", 0, 0, []byte("school_junzita\x00")},
	{"RelivePositon", 0, 0, profileRelive},
	{"DongHaiRelivePositon", 0, 0, profileDongHaiRelive},
}

// profileSkillRowNames 技能行 17 列（外服 k#1978 模板序）。
var profileSkillRowNames = []string{
	"ConfigID", "CanUse", "StaticData", "Level", "VarPropNo", "PauseTime",
	"MaxLevel", "TotalFillValue", "Quality", "WuXing", "AConsumeMP",
	"TargetShapePara2", "CoolDownTime", "HPHealMul", "MPHealMul",
	"PhyDamageMul", "StiDamageMul",
}

// profilePropEntry 按名解析（含 t6 f64 日期族的全表回落）。
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

func appendProfileNumeric(body []byte, typeCode int, spec profileProp) []byte {
	switch typeCode {
	case 1:
		return append(body, byte(spec.num))
	case 2:
		return binary.LittleEndian.AppendUint16(body, uint16(spec.num))
	case 3:
		return binary.LittleEndian.AppendUint32(body, uint32(spec.num))
	case 4, 6:
		if spec.f64 != 0 {
			return binary.LittleEndian.AppendUint64(body, math.Float64bits(spec.f64))
		}
		return binary.LittleEndian.AppendUint64(body, spec.num)
	default:
		return binary.LittleEndian.AppendUint32(body, uint32(spec.num))
	}
}

// AvatarFrame 生成玩家档案 0x28 mode=0x46 明文帧（b-line
// playerProfile0x28Frame 同构：41 基础字段 + 每技能 17 列行 +
// ShortcutKey + CurNeiGong 对）。literal47=true 改走 0x47 全字面量包装。
func AvatarFrame(ctx Context, literal47 bool) ([]byte, int) {
	state := ctx.State
	pos := state.Position // x,y,z,orient
	body := make([]byte, 0, 640)
	body = binary.LittleEndian.AppendUint32(body, ctx.Identity.ObjectID)
	body = binary.LittleEndian.AppendUint32(body, ctx.Identity.OwnerID)
	for _, f := range []float32{float32(pos[0]), float32(pos[1]), float32(pos[2]), float32(pos[3]),
		0, 0, 0, 0, 0, 0, 0, 0} {
		body = binary.LittleEndian.AppendUint32(body, math.Float32bits(f))
	}
	body = binary.LittleEndian.AppendUint32(body, 1)
	body = binary.LittleEndian.AppendUint16(body, 0) // 字段数占位 @60

	appendField := func(body []byte, entry clientdata.PropertyEntry, num uint64, f64 float64, raw []byte, f32 float32) []byte {
		body = binary.LittleEndian.AppendUint16(body, entry.Index)
		if raw != nil {
			body = binary.LittleEndian.AppendUint32(body, uint32(len(raw)))
			return append(body, raw...)
		}
		switch entry.TypeCode {
		case 1:
			return append(body, byte(num))
		case 2:
			return binary.LittleEndian.AppendUint16(body, uint16(num))
		case 3:
			return binary.LittleEndian.AppendUint32(body, uint32(num))
		case 4, 6:
			if f64 != 0 {
				return binary.LittleEndian.AppendUint64(body, math.Float64bits(f64))
			}
			return binary.LittleEndian.AppendUint64(body, num)
		default:
			return binary.LittleEndian.AppendUint32(body, math.Float32bits(f32))
		}
	}

	count := 0
	for _, spec := range profilePropSpecs {
		entry, ok := profilePropEntry(spec.name)
		if !ok {
			continue
		}
		raw := spec.raw
		if spec.name == "Photo" && state.Sex == 2 {
			raw = []byte("icon\\players\\girl01.png\x00")
		}
		body = binary.LittleEndian.AppendUint16(body, entry.Index)
		if raw != nil {
			body = binary.LittleEndian.AppendUint32(body, uint32(len(raw)))
			body = append(body, raw...)
		} else {
			body = appendProfileNumeric(body, entry.TypeCode, spec)
		}
		count++
	}
	// ShortcutKey（官方 f#042 尾字段）：存档键位原串。
	if ctx.Keybind != "" {
		if entry, ok := profilePropEntry("ShortcutKey"); ok {
			body = binary.LittleEndian.AppendUint16(body, entry.Index)
			body = binary.LittleEndian.AppendUint32(body, uint32(len(ctx.Keybind)+1))
			body = append(body, ctx.Keybind...)
			body = append(body, 0)
			count++
		}
	}
	// 技能行（每技能 17 列，b-line appendProfileSkillRows 同构；
	// VarPropNo/CD/消耗/射程无资源目录时置零）。
	entries := make([]clientdata.PropertyEntry, len(profileSkillRowNames))
	valid := make([]bool, len(profileSkillRowNames))
	for i, name := range profileSkillRowNames {
		if entry, ok := profilePropEntry(name); ok {
			entries[i], valid[i] = entry, true
		}
	}
	for _, skill := range state.Skills {
		configRaw := append([]byte(skill.ConfigID), 0)
		for i := range profileSkillRowNames {
			if !valid[i] {
				continue
			}
			switch profileSkillRowNames[i] {
			case "ConfigID":
				body = appendField(body, entries[i], 0, 0, configRaw, 0)
			case "CanUse":
				body = appendField(body, entries[i], 1, 0, nil, 0)
			case "StaticData":
				body = appendField(body, entries[i], uint64(uint32(skill.StaticData)), 0, nil, 0)
			case "Level":
				body = appendField(body, entries[i], uint64(clampU8(skill.Level)), 0, nil, 0)
			case "VarPropNo":
				body = appendField(body, entries[i], 0, 0, nil, 0)
			case "PauseTime":
				body = appendField(body, entries[i], uint64(uint32(skill.PauseTime)), 0, nil, 0)
			case "MaxLevel":
				body = appendField(body, entries[i], uint64(uint32(skill.MaxLevel)), 0, nil, 0)
			case "TotalFillValue":
				body = appendField(body, entries[i], uint64(uint32(skill.Total)), 0, nil, 0)
			case "Quality":
				body = appendField(body, entries[i], 1, 0, nil, 0)
			case "WuXing":
				body = appendField(body, entries[i], 0, 0, nil, 0)
			case "AConsumeMP":
				body = appendField(body, entries[i], 0, 0, nil, 0)
			case "TargetShapePara2":
				body = appendField(body, entries[i], 0, 0, nil, 0)
			case "CoolDownTime":
				body = appendField(body, entries[i], 0, 0, nil, 0)
			case "HPHealMul", "MPHealMul", "PhyDamageMul", "StiDamageMul":
				body = appendField(body, entries[i], 0, 0, nil, 1)
			}
			count++
		}
	}
	// 装备内功身份对（idx655 CurNeiGong + idx656 WuXing——官方档案
	// 唯一下发通道）。
	if state.CurNeiGong != "" {
		if entry, ok := profilePropEntry("CurNeiGong"); ok {
			body = binary.LittleEndian.AppendUint16(body, entry.Index)
			body = binary.LittleEndian.AppendUint32(body, uint32(len(state.CurNeiGong)+1))
			body = append(body, state.CurNeiGong...)
			body = append(body, 0)
			count++
		}
	}
	binary.LittleEndian.PutUint16(body[60:], uint16(count))

	if literal47 {
		return WrapAs47Literal(0x28, body), count
	}
	return WrapAs46Plain(0x28, body), count
}

func clampU8(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}
