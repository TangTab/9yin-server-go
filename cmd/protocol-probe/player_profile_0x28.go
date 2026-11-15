package main

// 玩家档案 0x28 全量帧——mode=0x46 明文生成器（2026-11-12 定版）。
//
// 权威规范：docs/official/外服玩家档案0x28全量帧.md。外服私服登录进场簇在 0x0B 进场、
// 0x14/0x11 对与 0x29 记录批之后，向玩家下发一条 5KB 级 mode=0x47（LZ）
// 的 0x28 全量档案帧（count 292~393），obj/own=玩家实体，紧随其后才是
// 玩家 0x14/0x11 视图行爆发；换景时整帧重发（k.pcapng #1978 实证）。
// 本生成器以 mode=0x46 纯 memcpy 明文方言程序化构造（与 npcDetail0x28V3Frame
// 同族，官方 l.pcapng 自有 0x46 先例，任何会话字典状态安全）。
//
// 帧布局（外服私服 #132/#1978 逐字节定版）：
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
// 是官方语义（k#1978 相对 k#15 即少 idx162）。BaseFace（外服私服 len=0
// 变体）v1 起省略。ShortcutKey（官方 f#042，str176 '1,87;2,83;…'）v2
// 起按存档键位嵌入（2026-11-15「保存的按键配置下发有问题」：外服家族
// 客户端在 AddObject 时从档案字段初始化键位管理器，后到的 0x10 idx198
// 只同步属性视图不重建引擎键表——官方唯一回放通道就是本字段；官方
// 抓包里的「主段+增量续段」是 LZ 编码器字典复用手法，语义即完整串，
// 0x46 明文单段=标准编码）。
//
// 值来源：六包（c/k/登陆/登陆2/登陆-all）独立会话恒定值——Name/Face 为
// 外服私服遮蔽占位（跨角色恒定，非真名）；LastLineTask 族/复活点表为登录
// 场景常量。真机验证渲染通过后再接技能行（ConfigID+16 列模板）与本服
// 真值（复活点/键位）。门控：默认开（2026-11-15 内置，NINEYIN_PLAYER_
// PROFILE_0X28=0 可关——fork 客户端丢弃 0x28 帧，开着无害）。

import (
	"encoding/binary"
	"math"
	"os"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

// playerProfile0x28Enabled 控制玩家档案 0x28 帧下发（默认开；
// NINEYIN_PLAYER_PROFILE_0X28=0 显式关闭——2026-11-15 起内置，bat 不再设参）。
func playerProfile0x28Enabled() bool {
	return os.Getenv("NINEYIN_PLAYER_PROFILE_0X28") != "0"
}

// 外服私服恒定字节值（六包跨会话逐字节一致；Name/Face 为遮蔽占位，见文件头）。
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

// playerProfilePropSpecs v1 字段清单（41 项，官方 idx 序；省略 BaseFace，
// ShortcutKey 由调用方存档串动态嵌入——均见文件头）。值=六包恒定外服私服
// 常量。
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

// playerProfileFengShuiDate 外服私服 HomeFengShuiDate 恒值（f64 日期序列，
// k/c/登陆 三包同值 42952.0）。
const playerProfileFengShuiDate = 42952.0

// profilePropEntry 按名解析档案字段条目。t6（f64 日期族）行被按名映射
// 排除（线型不可编码镜像 Java 语义，property_table.go:217），但档案帧
// 外服私服方言实发 8B——回落全表扫（末现，与 PlayerProperty 同语义）。
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

// playerProfileSkillRowNames 技能行 17 列（外服私服 k#1978 模板序：ConfigID
// 打头、四倍率收尾）。同一 idx 帧内重复合法（每技能一组）——外服私服档案
// 帧计数含行字段实例（count 393 ≈ 44 基础 + 17×行 + 尾区）。
var playerProfileSkillRowNames = []string{
	"ConfigID", "CanUse", "StaticData", "Level", "VarPropNo", "PauseTime",
	"MaxLevel", "TotalFillValue", "Quality", "WuXing", "AConsumeMP",
	"TargetShapePara2", "CoolDownTime", "HPHealMul", "MPHealMul",
	"PhyDamageMul", "StiDamageMul",
}

// appendProfileSkillRows 按外服私服模板逐技能追加行字段，返回追加字段数。
// 真值来源与 View40 skillRowProperties 同源：learnedSkillView 档值 +
// combatSkillCatalog 补集（VarPropNo/CD/消耗/射程）；Quality/倍率固定族
// 对齐 View40（=1；外服私服 k#1978 样本 Quality=2 系该技能个案，全行取 1 与
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
// pos=登录落点（posA）；scale/posB 按外服私服玩家形恒零。skills=已学集合
// （View40 merge 集同源），逐技能附外服私服模板 17 列行。keybind=存档键位
// 串（scheme 0 当前启用，195 上传原串；空=省略字段，稀疏官方语义）。
// curNeiGong=当前装备内功（官方档案 @0x110b 实证 CurNeiGong@655 唯一通道
// 就是本档案——0x10/View43 均不携带，引擎按装备内功 varprop 推导实力档；
// 空=省略）。curWuXing=CurNeiGongWuXing@656。
// 当前表缺席的字段名静默跳过（fork 表兼容，字段数按实际写入计）。
func playerProfile0x28Frame(objectID, ownerID uint32, pos role.Position, skills []learnedSkillView, keybind, curNeiGong string, curWuXing int32) ([]byte, int) {
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
	mirror := bisectActive("avatar-mirror")
	for _, spec := range playerProfilePropSpecs {
		if mirror && spec.name == "DongHaiRelivePositon" {
			// 镜像轮：DongHai 起整段换官档字节（见 avatarMirrorFields）
			continue
		}
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
	// 二分 A/B 位 avatar-bare：官方 f#036 BaseFace 空串（标准编码
	// [len=1][NUL]）补位——对照官档 43 字段形（2026-11-15「重登回走」
	// 生成档 vs 透传档差异定位）。
	if bisectActive("avatar-bare") {
		if entry, ok := profilePropEntry("BaseFace"); ok {
			body = binary.LittleEndian.AppendUint16(body, entry.Index)
			body = binary.LittleEndian.AppendUint32(body, 1)
			body = append(body, 0)
			count++
		}
	}
	// ShortcutKey（官方 f#042 尾字段）：存档键位原串 + NUL（appendStringValue
	// 同形——len 含 NUL）。空串省略。
	if keybind != "" {
		if entry, ok := profilePropEntry("ShortcutKey"); ok {
			body = binary.LittleEndian.AppendUint16(body, entry.Index)
			body = binary.LittleEndian.AppendUint32(body, uint32(len(keybind)+1))
			body = append(body, keybind...)
			body = append(body, 0)
			count++
		}
	}
	if mirror {
		body = append(body, avatarMirrorFields...)
		count += 3588
	} else {
		rowBytes, rowCount := appendProfileSkillRows(body, skills)
		body = rowBytes
		count += rowCount
	}
	// 装备内功身份对（官方档案尾区实证：k#1978 @0x110b idx655
	// CurNeiGong="ng_jz_006" + idx656 CurNeiGongWuXing——引擎实力档的
	// 装备内功锚点，0x28 档案是官方唯一下发通道）。空 ID 省略（稀疏语义）。
	if curNeiGong != "" {
		if entry, ok := profilePropEntry("CurNeiGong"); ok {
			body = binary.LittleEndian.AppendUint16(body, entry.Index)
			body = binary.LittleEndian.AppendUint32(body, uint32(len(curNeiGong)+1))
			body = append(body, curNeiGong...)
			body = append(body, 0)
			count++
		}
		if entry, ok := profilePropEntry("CurNeiGongWuXing"); ok && curWuXing != 0 {
			body = binary.LittleEndian.AppendUint16(body, entry.Index)
			body = binary.LittleEndian.AppendUint32(body, uint32(curWuXing))
			count++
		}
	}
	// 字段数回填 body[60:62]（变换块定长 52B 保证）。
	binary.LittleEndian.PutUint16(body[60:], uint16(count))

	if bisectActive("avatar-47") {
		// 二分 A/B 位 avatar-47：同样内容改走 mode-0x47 全字面量 LZ 包装
		//（零池依赖、解码输出=原文；tools/lz47_literal.py 同语义 Python
		// 版已 lzdec 往返验证）——判别「重登回走」触发器是否在 0x46
		// memcpy vs 0x47 解压路径本身（2026-11-15 终局实验）。
		frame := wrapProfile0x28As47Literal(body)
		return frame, count
	}
	frame := make([]byte, 0, len(body)+10)
	frame = append(frame, 0x28, 0x46)
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)+9)) // len-1
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)))   // 明文体长
	frame = append(frame, body...)
	return frame, count
}

// wrapProfile0x28As47Literal mode-0x47 全字面量编码：字面量 token（k∈
// {1,2,3,4} 字节，n 位型 0010/0100/1000/0000，耗 k bit）+ 尾 11 字节逐
// 字节 1bit；每 flags 词在已用位最高位置哨兵 1（寄存器==1 触发重填）。
func wrapProfile0x28As47Literal(body []byte) []byte {
	var out []byte
	var wordBits []bool
	var wordData []byte
	flush := func() {
		w := uint32(0)
		for i, b := range wordBits {
			if b {
				w |= 1 << uint(i)
			}
		}
		w |= 1 << uint(len(wordBits)) // 哨兵位
		out = binary.LittleEndian.AppendUint32(out, w)
		out = append(out, wordData...)
		wordBits, wordData = nil, nil
	}
	emitLiteral := func(k int, chunk []byte) {
		need := k
		if len(wordBits)+need > 31 {
			flush()
		}
		var n uint8
		switch k {
		case 1:
			n = 0b0010
		case 2:
			n = 0b0100
		case 3:
			n = 0b1000
		default:
			n = 0
		}
		for j := 0; j < 4; j++ {
			wordBits = append(wordBits, n&(1<<uint(j)) != 0)
		}
		wordData = append(wordData, chunk...)
	}
	emitTailByte := func(b byte) {
		if len(wordBits)+1 > 31 {
			flush()
		}
		wordBits = append(wordBits, false)
		wordData = append(wordData, b)
	}
	mainLen := len(body) - 0xb
	if mainLen < 0 {
		mainLen = len(body)
	}
	for i := 0; i < mainLen; {
		k := 4
		if r := mainLen - i; r < k {
			k = r
		}
		emitLiteral(k, body[i:i+k])
		i += k
	}
	for _, b := range body[mainLen:] {
		emitTailByte(b)
	}
	flush()
	frame := make([]byte, 0, len(out)+10)
	frame = append(frame, 0x28, 0x47)
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(out)+9)) // len-1（0x47 客户端以 outlen 为准）
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)))
	frame = append(frame, out...)
	return frame
}

// curNeiGongProfilePair 档案装备内功身份对：CurNeiGong（progress 权威）
// + 五行（目录缺表回落 0=省略字段）。
func (p *playerActor) curNeiGongProfilePair() (string, int32) {
	p.mu.Lock()
	id := p.progress.curNeiGong
	p.mu.Unlock()
	if id == "" {
		return "", 0
	}
	wuXing := int32(0)
	if catalog, err := loadModernNeiGongCatalog(); err == nil {
		wuXing = catalog.wuXingOf(id)
	}
	return id, wuXing
}

// appendProfileNumeric 按官方表列型码写数值。t6（f64 日期族）特殊：
// OfficialWireTypeOf 映射为不可编码 0，但档案帧外服私服方言实发 8B f64
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
