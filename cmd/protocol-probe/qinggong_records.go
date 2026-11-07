package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"time"
	"unicode/utf16"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/qinggong"
)

type qinggongBuffSpec struct {
	slot       uint16
	staticData uint32
	duration   time.Duration
	clearOnEnd bool
}

// qinggong_9 轻身缓降的 msg169 链常量（Java PlayerActor.GLIDE_BUFF_*
// 同源）：BufferID=buff_qg_light、buff_static[218].Category=2048。
// 52795 抓包实证缓降挂/摘走 msg169 九值时间形态，而非其余轻功族的
// 纯槽帧路径。
const (
	qingGongGlideBuffID = "buf_qg_light"
	glideBuffCategory   = int32(2048)
)

// init 自检：记录下标字面值必须与官方表按名解析一致，防止
// official_record_table.bin 更新后常量静默漂移。
func init() {
	for name, want := range map[string]uint16{
		"Task_Accepted":      recordTaskAccepted,
		"Task_Completed":     recordTaskCompleted,
		"shortcut_rec":       recordShortcut,
		"title_rec":          recordTitle,
		"cooldown_rec":       recordCooldown,
		"fight_relation_rec": recordFightRelation,
		"active_skill_rec":   recordActiveSkill,
		"QingGongRec":        recordQingGong,
		"ActiveQGSkillRec":   recordActiveQingGong,
		"active_jingmai_rec": recordActiveJingMai,
		"FashionSuitRec":     recordFashionSuit,
		"CardRec":            recordCard,
		"UseCardRec":         recordUseCard,
		"NewRideRec":         recordNewRide,
		"RecvLetterRec":      recordRecvLetter,
	} {
		if got := clientdata.MustRecordIndexByName(name); got != want {
			panic(fmt.Sprintf("record index for %s drifted: official=%d constant=%d", name, got, want))
		}
	}
}

// qinggongBuffFor 根据已安装的现代 QGDefine.ini BufferID 推导原生移动
// 状态。这些 BuffStore 条目是客户端在移动数值和已受理的 216 资源扣费
// 之外还需要的东西。2026-11-07 跳段限制批次补齐高阶轻功链：StaticData/
// LifeTime 出自 F: 部署 buff_new.ini 实证（buf_DHadd_2rd=1578、
// buf_DHadd_3rd=1579 均无 LifeTime=常驻；buf_qg_airrush=6894 LifeTime
// 100ms；buf_qg_fast=217 / buf_qg_climb=219 10min；buf_qg_gliding=23070、
// buf_qg_dive=23292 无 LifeTime=持续到 end）。
func qinggongBuffFor(definition qinggong.Definition) (qinggongBuffSpec, bool) {
	switch definition.BufferID {
	case "buf_qg_drift":
		return qinggongBuffSpec{slot: waterDriftBuffSlot, staticData: waterDriftStaticData, duration: 10 * time.Minute, clearOnEnd: true}, true
	case "buf_qg_dst":
		return qinggongBuffSpec{slot: waterJumpBuffSlot, staticData: waterJumpStaticData, duration: 3 * time.Second}, true
	case "buf_qg_airrush": // qinggong_6 穿云纵 / 高阶 21-23 冲刺
		return qinggongBuffSpec{slot: qgAirRushBuffSlot, staticData: 6894, duration: 100 * time.Millisecond}, true
	case "buf_qg_fast": // qinggong_8 神行无踪
		// UseType=1 持续型（同踏波行/上天梯族）：buff_new LifeTime 10min
		// 只是兜底寿命，end 帧必须摘——漏 clearOnEnd 会把神行表现钉满
		// 10 分钟（2026-11-07 真机「可进入不可退出」根因）。
		return qinggongBuffSpec{slot: qgFastBuffSlot, staticData: 217, duration: 10 * time.Minute, clearOnEnd: true}, true
	case "buf_qg_climb": // qinggong_10 壁虎游墙 / qinggong_11 上天梯
		return qinggongBuffSpec{slot: qgClimbBuffSlot, staticData: 219, duration: 10 * time.Minute, clearOnEnd: true}, true
	case "buf_qg_gliding": // 高阶 24 滑翔
		return qinggongBuffSpec{slot: qgGlidingBuffSlot, staticData: 23070, duration: 10 * time.Minute, clearOnEnd: true}, true
	case "buf_qg_dive": // 高阶 25 急降
		return qinggongBuffSpec{slot: qgDiveBuffSlot, staticData: 23292, duration: 3 * time.Second}, true
	case qingGongGlideBuffID: // qinggong_9 轻身缓降（buff_static[218]）
		// Java GLIDE_BUFF_LIFETIME_MS=600s（buff_new LifeTime）；UseType=1
		// 秒耗 10/s 下点耗尽必然先到，10min 只是槽租约兜底寿命。end/
		// 耗尽两条既有清槽路径之外，还各伴随一帧 msg169 九值摘帧
		//（writeGlideTimedBuff*，52795 抓包 buf_qg_light 实证）。
		return qinggongBuffSpec{slot: qgLightBuffSlot, staticData: glideBuffStaticData, duration: 10 * time.Minute, clearOnEnd: true}, true
	default:
		// qinggong_20 大跳的 BufferID=buf_DHadd_2rd 与激活能力 Buff
		// 共槽（17）：使用路径不重复挂（能力已在，activateQingGongBuff
		// 的重复闸自然短路），更不能带 clearOnEnd——end 清槽会连能力
		// Buff 一起摘掉。能力链只经 qingGongCapabilityBuff 挂/摘。
		return qinggongBuffSpec{}, false
	}
}

// qingGongCapabilityBuffLease 是激活轻功记录挂的能力 Buff 租约：
// activateBuff 的 lifetime=0 会立即过期，常驻语义用内功槽同款 24h
// （会话级——activeQingGong 本就不跨登录持久）。
const qingGongCapabilityBuffLease = 24 * time.Hour

// qingGongCapabilityBuff 是激活轻功记录（217）挂的常驻能力 Buff
// （native-menu qinggongBuffFor 硬编码同款映射）：激活
// qinggong_15（纤云步法·凌空踏虚）→ buf_DHadd_2rd（二段跳存在），
// qinggong_17（纤云步法·穿云纵）→ buf_DHadd_3rd（三段跳存在）。
func qingGongCapabilityBuff(qingGongID string) (qinggongBuffSpec, bool) {
	switch qingGongID {
	case "qinggong_15":
		return qinggongBuffSpec{slot: qgDHadd2rdBuffSlot, staticData: 1578, duration: qingGongCapabilityBuffLease}, true
	case "qinggong_17":
		return qinggongBuffSpec{slot: qgDHadd3rdBuffSlot, staticData: 1579, duration: qingGongCapabilityBuffLease}, true
	default:
		return qinggongBuffSpec{}, false
	}
}

// isJumpStageQingGong 判定 216 begin 是否计入腾空跳段（服务端
// 反作弊计段集合）：凌空踏虚（Type 4，二段跳）、扶摇步法（Type 5，
// 三段跳）、高阶 20 大跳。健步功（Type 1）客户端不发 216 或免费，
// 不计段——服务端对不发 216 的动作看不见落地，计段会误伤地面连跳。
// 冲刺类（6/21-23）已有 500ms 重报闸+扣费约束，不进本集合。
func isJumpStageQingGong(definition qinggong.Definition) bool {
	switch definition.Type {
	case 4, 5, 20:
		return true
	default:
		return false
	}
}

// qingGongExistBuffSpec 解析 ExistBufID 能力门槛：仅对已实现能力来源
// 的 Buff（buf_DHadd_2rd/3rd，激活 qinggong_15/17 常驻挂）设门槛；
// 其余（如 qinggong_9 的 buf_DHadd_light，能力来源未实现）返回
// known=false 不设门槛，保旧表/缓降链现状。
func qingGongExistBuffSpec(existBufID string) (qinggongBuffSpec, bool) {
	switch existBufID {
	case "buf_DHadd_2rd":
		return qinggongBuffSpec{slot: qgDHadd2rdBuffSlot, staticData: 1578}, true
	case "buf_DHadd_3rd":
		return qinggongBuffSpec{slot: qgDHadd3rdBuffSlot, staticData: 1579}, true
	default:
		return qinggongBuffSpec{}, false
	}
}

// qingGongDrainPerSecond 是持续型轻功（UseType=1）租约的每秒扣点量，
// = qgdefine BufferID → buff_new/buff_static/buff_varprop 的 |CCQGP|
// （Java CombatSkillCatalog.qingGongDrainPerSecond 运行时读表；本服
// 未装载 buff 表，按 Java 已实证值内联——玩家资料：踏波行/上天梯
// 开局 10 点、其后每秒 7 点；qinggong_9 缓降每秒 10 点。神行无踪
// buf_qg_fast=7 出自官方 buff_varprop[22001] L1 CCQGP=-7 实证）。
// 表缺 CCQGP 返回 0：租约仍挂（保续约/回复暂停语义）。
func qingGongDrainPerSecond(definition qinggong.Definition) int32 {
	switch definition.BufferID {
	case "buf_qg_light": // qinggong_9 轻身缓降（buff_varprop[23001] L1 CCQGP=-10）
		return 10
	case "buf_qg_drift": // qinggong_2 踏波行
		return 7
	case "buf_qg_fast": // qinggong_8 神行无踪（buff_varprop[22001] L1 CCQGP=-7）
		return 7
	case "buf_qg_climb": // qinggong_10 壁虎游墙 / qinggong_11 上天梯
		return 7
	default:
		return 0
	}
}

const (
	// FxNet2!GameReceiver::RecvRecordRow (0x1060BC40) 的列线型码，
	// 与官方 0x0A 表声明一致：1=byte、2=u16 Word、3=int32、
	// 4=int64、6=IEEE-754 double、7=窄字符串、8=宽字符串
	// （UTF-16LE，u32 字节数+2 + 尾 u16 NUL——RecvLetterRec col0/3/4）。
	recordColumnByte   = byte(1)
	recordColumnWord   = byte(2)
	recordColumnInt    = byte(3)
	recordColumnInt64  = byte(4)
	recordColumnDouble = byte(6)
	recordColumnString = byte(7)

	recordColumnWideString = byte(8)

	// 记录下标 = 官方 447 schema 表的帧内位置（Java RecordSchemas
	// official_record_table.bin 同源）。此前本服自协商 11 张表
	// （shortcut_rec 落在自定下标 4），真机客户端的原生记录消费者
	// （CoolLogicManager、快捷栏等）按官方下标绑定——行帧字节级
	// 正确却不上显的根因。改为整表透传官方方言（Java 同款）。
	// 字面值须与 official_record_table.bin 一致（init 自检）。
	recordTaskAccepted   uint16 = 29  // Task_Accepted（28 列）
	recordTaskCompleted  uint16 = 31  // Task_Completed
	recordShortcut       uint16 = 93  // shortcut_rec（[3,7,7]）
	recordTitle          uint16 = 98  // title_rec（[3,1,4]）
	recordCooldown       uint16 = 109 // cooldown_rec（[3,4,4,2]）
	recordFightRelation  uint16 = 114 // fight_relation_rec（[9,2]，0x13 战斗关系格）
	recordQingGong       uint16 = 116 // QingGongRec（[7]）
	recordActiveQingGong uint16 = 117 // ActiveQGSkillRec（[7,1]）
	recordActiveSkill    uint16 = 446 // active_skill_rec（[7]，出招行，动作总时长后 0x12 删）
	recordActiveJingMai  uint16 = 162 // active_jingmai_rec（[7]）
	recordFashionSuit    uint16 = 315 // FashionSuitRec（[7,3,3,4,3,3]）
	recordCard           uint16 = 322 // CardRec（[3]）
	recordUseCard        uint16 = 324 // UseCardRec（[2,2]）
	viewportQingGong     uint16 = 46
	// View 对象使用原始的场景属性 ConfigID 字段。后续仅商店用的兼容字段
	// 105 有相同的显示名，但 QingGongContainer 的原生查找读取的不是它。
	qingGongConfigProperty uint16 = 7
)

// starterQingGongIDs 是当前客户端的基础类别。它们刻意使用 config ID
// 而非视觉标签，让本地客户端自行解析其当前的 QGDefine/QGSkill
// 数据。
var starterQingGongIDs = []string{
	"qinggong_1", "qinggong_2", "qinggong_3", "qinggong_4", "qinggong_5",
	"qinggong_6", "qinggong_7", "qinggong_8", "qinggong_9", "qinggong_10",
	"qinggong_11", "qinggong_12", "qinggong_14", "qinggong_15", "qinggong_16",
	"qinggong_17", "qinggong_26", "qinggong_gaojie",
}

// starterQingGongSkillIDs 是实际的轻功技能实例，不是上面的类别 ID。
// QingGongRec 解锁一个类别，而 WuXueQuery:GetLearnID_QGSkill 从
// QingGongContainer（View 46）读取这些对象来填充卡片。
var starterQingGongSkillIDs = []string{
	"QG_JH_001", "QG_JH_002", "QG_JH_003", "QG_JH_004", "QG_JH_005", "QG_JH_006", "QG_JH_007", "QG_JH_008", "QG_JH_009", "QG_JH_010",
	"QG_JH_011", "QG_JH_012", "QG_JH_013", "QG_JH_014", "QG_JH_015", "QG_JH_016", "QG_JH_017", "QG_JH_018", "QG_JH_019", "QG_JH_020",
	"QG_JH_021", "QG_JH_022", "QG_JH_023", "QG_JH_024", "QG_JH_025", "QG_JH_026", "QG_JH_027", "QG_JH_028", "QG_JH_029", "QG_JH_02_001",
	"QG_JH_030", "QG_JH_031", "QG_JH_032", "QG_JH_033", "QG_JH_034", "QG_JH_035", "QG_JH_036", "QG_JH_037", "QG_JH_038", "QG_JH_039",
	"QG_JH_03_001", "QG_JH_040", "QG_JH_041", "QG_JH_04_001", "QG_JH_05_001", "QG_JH_06_001", "QG_JH_gj_001", "QG_JH_gj_002", "QG_JH_gj_003", "QG_JH_gj_004", "QG_JH_gj_006", "QG_JH_gjskill_001",
}

// serverRecordTable 逐字节透传官方 0x0A 记录表协商帧
// （clientdata.OfficialRecordTableFrame，447 schema）。客户端按
// 官方方言绑定记录下标，自建子表会让行帧错绑——与 0x09 属性表
// 同一透明策略。
func serverRecordTable() ([]byte, error) {
	frame := clientdata.OfficialRecordTableFrame()
	if len(frame) < 3 || frame[0] != 0x0A {
		return nil, fmt.Errorf("official record table frame malformed (%d bytes)", len(frame))
	}
	return frame, nil
}

// serverRecordAddString 为单列的场景对象记录编码 SERVER_RECORD_ADD
// (0x11)。玩家标识必须与 0x0D 对象标识（1-1）一致，否则
// GetPlayer()/QueryRecord 看不到新行。
func serverRecordAddString(objectID, ownerID uint32, recordIndex uint16, value string) []byte {
	msg := make([]byte, 16, 16+len(value)+5)
	msg[0] = 0x11
	msg[1] = 0 // 场景对象
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], 0) // 第一列
	binary.LittleEndian.PutUint16(msg[14:], 1) // 一个字符串列
	msg = binary.LittleEndian.AppendUint32(msg, uint32(len(value)+1))
	msg = append(msg, value...)
	return append(msg, 0)
}

// serverRecordAddInt32 编码一个整型记录单元格。SERVER_RECORD_ADD 追加
// 行，因此需要稀疏行的调用方必须先发出更早的行。
func serverRecordAddInt32(objectID, ownerID uint32, recordIndex uint16, value uint32) []byte {
	return serverRecordAddInt32Row(objectID, ownerID, recordIndex, 0, value)
}

// serverRecordAddInt32Row 带显式行号的整型记录单元格（CardRec 位图行
// 专用：FxNet2 按行号对齐 32 皮肤/行的位集，缺行会让后续行整体前移）。
func serverRecordAddInt32Row(objectID, ownerID uint32, recordIndex uint16, row, value uint32) []byte {
	msg := make([]byte, 20)
	msg[0] = 0x11
	msg[1] = 0 // 场景对象
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], uint16(row)) // 源行号
	binary.LittleEndian.PutUint16(msg[14:], 1)           // 一行
	binary.LittleEndian.PutUint32(msg[16:], value)
	return msg
}

type recordCell struct {
	intValue    *uint32
	int64Value  *int64
	byteValue   *uint32
	doubleValue *float64
	text        *string
}

func recordInt(value uint32) recordCell      { return recordCell{intValue: &value} }
func recordInt64(value int64) recordCell     { return recordCell{int64Value: &value} }
func recordByte(value uint32) recordCell     { return recordCell{byteValue: &value} }
func recordFloat64(value float64) recordCell { return recordCell{doubleValue: &value} }
func recordString(value string) recordCell   { return recordCell{text: &value} }

// serverRecordAddCells 按官方 schema 写满一整行。包内 +0x0C/+0x0E 是
// FxNet2::RecvRecordRow 消费的源行起始/数量，不是列的起始/数量对。
// 调用方可给前缀列（窄于官方声明宽度时缺失尾列按线型补零/空串——
// Java RecordMessages.recordAddCells 同语义，Task_Accepted 官方
// 28 列而功能解锁只填前 7 列）；超出声明宽度报错。
func serverRecordAddCells(objectID, ownerID uint32, recordIndex uint16, cells []recordCell) ([]byte, error) {
	schema, ok := clientdata.RecordSchemaByIndex(int(recordIndex))
	if !ok {
		return nil, fmt.Errorf("record index %d outside official table", recordIndex)
	}
	colTypes := schema.ColumnTypes
	if len(cells) > len(colTypes) {
		return nil, fmt.Errorf("record %q has %d cells, want at most %d", schema.Name, len(cells), len(colTypes))
	}
	msg := make([]byte, 16)
	msg[0] = 0x11
	msg[1] = 0
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], 0) // 第一个源行
	binary.LittleEndian.PutUint16(msg[14:], 1) // 一整行
	for column := range colTypes {
		var cell *recordCell
		if column < len(cells) {
			cell = &cells[column]
		}
		switch colTypes[column] {
		case recordColumnInt:
			if cell == nil || cell.intValue == nil {
				msg = binary.LittleEndian.AppendUint32(msg, 0)
				continue
			}
			if !cell.onlyInt() {
				return nil, fmt.Errorf("record %q column %d requires int32", schema.Name, column)
			}
			msg = binary.LittleEndian.AppendUint32(msg, *cell.intValue)
		case recordColumnWord:
			if cell == nil || cell.intValue == nil {
				msg = binary.LittleEndian.AppendUint16(msg, 0)
				continue
			}
			if !cell.onlyInt() {
				return nil, fmt.Errorf("record %q column %d requires word", schema.Name, column)
			}
			// 与 Java recordAddCells case2 相同的低 16 位截断
			// （cooldown_rec 第 4 列 team 0xFFFF 个人行等负值口径）。
			msg = binary.LittleEndian.AppendUint16(msg, uint16(*cell.intValue&0xFFFF))
		case recordColumnByte:
			if cell == nil || cell.byteValue == nil {
				msg = append(msg, 0)
				continue
			}
			if !cell.onlyByte() {
				return nil, fmt.Errorf("record %q column %d requires byte", schema.Name, column)
			}
			if *cell.byteValue > math.MaxUint8 {
				return nil, fmt.Errorf("record %q column %d byte %d out of range", schema.Name, column, *cell.byteValue)
			}
			msg = append(msg, byte(*cell.byteValue))
		case recordColumnInt64:
			if cell == nil || cell.int64Value == nil {
				msg = binary.LittleEndian.AppendUint64(msg, 0)
				continue
			}
			if !cell.onlyInt64() {
				return nil, fmt.Errorf("record %q column %d requires int64", schema.Name, column)
			}
			msg = binary.LittleEndian.AppendUint64(msg, uint64(*cell.int64Value))
		case recordColumnDouble:
			if cell == nil || cell.doubleValue == nil {
				msg = binary.LittleEndian.AppendUint64(msg, 0)
				continue
			}
			if !cell.onlyDouble() {
				return nil, fmt.Errorf("record %q column %d requires double", schema.Name, column)
			}
			msg = binary.LittleEndian.AppendUint64(msg, math.Float64bits(*cell.doubleValue))
		case recordColumnString:
			if cell == nil || cell.text == nil {
				msg = binary.LittleEndian.AppendUint32(msg, 1)
				msg = append(msg, 0)
				continue
			}
			if !cell.onlyText() {
				return nil, fmt.Errorf("record %q column %d requires string", schema.Name, column)
			}
			msg = binary.LittleEndian.AppendUint32(msg, uint32(len(*cell.text)+1))
			msg = append(msg, (*cell.text)...)
			msg = append(msg, 0)
		case recordColumnWideString:
			if cell == nil || cell.text == nil {
				msg = binary.LittleEndian.AppendUint32(msg, 2)
				msg = binary.LittleEndian.AppendUint16(msg, 0)
				continue
			}
			if !cell.onlyText() {
				return nil, fmt.Errorf("record %q column %d requires wide string", schema.Name, column)
			}
			// Java RecordMessages case 8 同构：u32(UTF-16LE 字节数+2) +
			// 宽串字节 + 尾 u16 NUL。
			wide := utf16.Encode([]rune(*cell.text))
			msg = binary.LittleEndian.AppendUint32(msg, uint32(len(wide)*2+2))
			for _, unit := range wide {
				msg = binary.LittleEndian.AppendUint16(msg, unit)
			}
			msg = binary.LittleEndian.AppendUint16(msg, 0)
		default:
			return nil, fmt.Errorf("record %q column %d has unsupported official type %d", schema.Name, column, colTypes[column])
		}
	}
	return msg, nil
}

// only* 谓词保证每个 recordCell 恰好携带一种类型的值。
func (c recordCell) onlyInt() bool {
	return c.int64Value == nil && c.byteValue == nil && c.doubleValue == nil && c.text == nil
}

func (c recordCell) onlyByte() bool {
	return c.intValue == nil && c.int64Value == nil && c.doubleValue == nil && c.text == nil
}

func (c recordCell) onlyInt64() bool {
	return c.intValue == nil && c.byteValue == nil && c.doubleValue == nil && c.text == nil
}

func (c recordCell) onlyDouble() bool {
	return c.intValue == nil && c.int64Value == nil && c.byteValue == nil && c.text == nil
}

func (c recordCell) onlyText() bool {
	return c.intValue == nil && c.int64Value == nil && c.byteValue == nil && c.doubleValue == nil
}

const (
	functionUnlockTaskID uint32 = 1346

	// 条件 23614 的直连分支针对 1069--1073 和 1344 读取 Task_Accepted 的
	// 状态列 6。已完成任务分支（26303）覆盖同一家族，因此兼容服务器把
	// 两个已验证的来源都暴露出来。
	facultyGuideTaskFirst uint32 = 1069
	facultyGuideTaskLast  uint32 = 1073
	facultyAltTaskID      uint32 = 1344

	// 条件 23613 在内功面板的演武按钮发出 1006/21 请求之前对其门控。其
	// 直连分支针对 1140--1147 和 51002 读取 Task_Accepted 的状态列 6；
	// 嵌套公式 23647 检查它们的已完成位。两者都要保留，因为实机证据
	// 表明仅已完成位无法启用 btn_act。
	facultyActTaskFirst uint32 = 1140
	facultyActTaskLast  uint32 = 1147
	facultyActAltTaskID uint32 = 51002
)

// taskCompletedBit 遵循 0x11854870 处的现代 FxGameLogic 谓词。它把一
// 起始的任务序号映射到 32 位的 Task_Completed 记录行/位。任务 1346
// 解析为第 42 行第 3 位：掩码 0x00000004。
func taskCompletedBit(taskID uint32) (row uint32, mask uint32) {
	serial := taskID + 1
	row = serial / 32
	bit := serial % 32
	if bit == 0 {
		row--
		bit = 32
	}
	return row, uint32(1) << (bit - 1)
}

func taskCompletedRows(taskIDs []uint32) (map[uint32]uint32, uint32) {
	rowValues := make(map[uint32]uint32)
	var finalRow uint32
	for _, taskID := range taskIDs {
		row, mask := taskCompletedBit(taskID)
		rowValues[row] |= mask
		if row > finalRow {
			finalRow = row
		}
	}
	return rowValues, finalRow
}

func facultyAcceptedTaskIDs() []uint32 {
	taskIDs := make([]uint32, 0, int(facultyGuideTaskLast-facultyGuideTaskFirst)+int(facultyActTaskLast-facultyActTaskFirst)+4)
	for taskID := facultyGuideTaskFirst; taskID <= facultyGuideTaskLast; taskID++ {
		taskIDs = append(taskIDs, taskID)
	}
	taskIDs = append(taskIDs, facultyAltTaskID)
	for taskID := facultyActTaskFirst; taskID <= facultyActTaskLast; taskID++ {
		taskIDs = append(taskIDs, taskID)
	}
	return append(taskIDs, facultyActAltTaskID)
}

// grantFunctionUnlock 补齐标准的主功能前置条件以及两个现代修炼前置
// 家族。有界的七列 Task_Accepted 行把活动任务门控挂到玩家身上，而
// 已验证的 Task_Completed 位集保留它们对应的已完成分支。由于
// Task_Completed 只能追加，前面的每一行都必须发出，包括包含所需位
// 的各行之前的零值行。
func grantFunctionUnlock(conn sceneMessageConnection) error {
	taskIDs := []uint32{functionUnlockTaskID, facultyAltTaskID, facultyActAltTaskID}
	for taskID := facultyGuideTaskFirst; taskID <= facultyGuideTaskLast; taskID++ {
		taskIDs = append(taskIDs, taskID)
	}
	for taskID := facultyActTaskFirst; taskID <= facultyActTaskLast; taskID++ {
		taskIDs = append(taskIDs, taskID)
	}

	for _, taskID := range facultyAcceptedTaskIDs() {
		// 官方 28 列方言的前 7 列（id,line,server,submit_scene,submit_npc,
		// title,status），尾列由 serverRecordAddCells 按线型补零——与 Java
		// StarterQingGong.functionUnlockFrames / 52795 抓包 task 1069 行
		// 144B 全零形状逐列对齐。
		frame, err := serverRecordAddCells(playerObjectID, playerOwnerID, recordTaskAccepted, []recordCell{
			recordInt(taskID), recordInt(0), recordString(""),
			recordInt(0), recordString(""), recordString(""), recordInt(0),
		})
		if err != nil {
			return fmt.Errorf("encode Task_Accepted task %d: %w", taskID, err)
		}
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("add Task_Accepted task %d: %w", taskID, err)
		}
	}

	rowValues, finalRow := taskCompletedRows(taskIDs)
	for currentRow := uint32(0); currentRow <= finalRow; currentRow++ {
		value := rowValues[currentRow]
		if err := conn.WriteFrame(serverRecordAddInt32(playerObjectID, playerOwnerID, recordTaskCompleted, value)); err != nil {
			return fmt.Errorf("add Task_Completed row %d: %w", currentRow, err)
		}
	}
	log.Printf("Function/faculty unlock staged Task_Accepted tasks=%v and Task_Completed tasks=%v final-row=%d", facultyAcceptedTaskIDs(), taskIDs, finalRow)
	return nil
}

// grantStarterQingGong 在 ClientReady 之后写入两个客户端侧的事实来源：
// QingGongRec 驱动 CheckLearn_QingGong，而 View 46 支撑针对
// QingGongContainer 的条件检查。
func grantStarterQingGong(conn sceneMessageConnection) error {
	for _, id := range starterQingGongIDs {
		if err := conn.WriteFrame(serverRecordAddString(playerObjectID, playerOwnerID, recordQingGong, id)); err != nil {
			return fmt.Errorf("add QingGongRec %s: %w", id, err)
		}
	}
	viewItems := append(append([]string(nil), starterQingGongIDs...), starterQingGongSkillIDs...)
	if err := conn.WriteFrame(serverCreateView(serverViewSpec{ID: viewportQingGong, Capacity: uint16(len(viewItems))})); err != nil {
		return fmt.Errorf("create QingGongContainer: %w", err)
	}
	for slot, id := range viewItems {
		frame, err := serverViewAdd(viewportQingGong, uint16(slot+1), []serverViewProperty{viewString(qingGongConfigProperty, id)})
		if err != nil {
			return fmt.Errorf("encode QingGongContainer %s: %w", id, err)
		}
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("add QingGongContainer %s: %w", id, err)
		}
	}
	if err := grantFunctionUnlock(conn); err != nil {
		return err
	}
	return nil
}

// activateQingGongRecord 实现 CLIENT_CUSTOMMSG_ACTIVE_QINGGONG (217)。
// 当前 Lua UI 的 check_qgskill_is_avtive 在允许已装备的 qinggong_gaojie
// 动作之前会查询 ActiveQGSkillRec。这与 216 不同，后者只在动作已经
// 激活之后才消耗移动资源。
func activateQingGongRecord(conn sceneMessageConnection, player *playerActor, id string) (bool, error) {
	if player == nil {
		return false, fmt.Errorf("character has not entered a scene")
	}
	if id == "" {
		return false, fmt.Errorf("active qinggong id is empty")
	}
	if !player.activateQingGong(id) {
		return false, nil
	}
	// 官方 ActiveQGSkillRec 是 [string, byte] 两列：只填 id，尾列 byte
	// 由 serverRecordAddCells 补零（此前自协商单列表下用
	// serverRecordAddString，官方方言下会变成短行触发 Out range）。
	frame, err := serverRecordAddCells(playerObjectID, playerOwnerID, recordActiveQingGong, []recordCell{recordString(id)})
	if err != nil {
		return false, fmt.Errorf("encode ActiveQGSkillRec %s: %w", id, err)
	}
	if err := conn.WriteFrame(frame); err != nil {
		return false, fmt.Errorf("add ActiveQGSkillRec %s: %w", id, err)
	}
	// 能力 Buff 链（2026-11-07 跳段限制批次）：激活 qinggong_15/17 挂
	// buf_DHadd_2rd/3rd 常驻（24h 会话租约）——这是 qinggong_4/5 的
	// ExistBufID 能力门槛来源，也是客户端跳段物理读的存在 Buff。
	if spec, known := qingGongCapabilityBuff(id); known && !player.hasActiveBuff(spec.slot, spec.staticData, time.Now()) {
		buffState, expires := player.activateBuff(spec.slot, spec.staticData, spec.duration, time.Now())
		update, err := player.vitalUpdate()
		if err != nil {
			return true, fmt.Errorf("encode qinggong capability Buff %s: %w", id, err)
		}
		if err := conn.WriteFrame(update); err != nil {
			return true, fmt.Errorf("write qinggong capability Buff %s: %w", id, err)
		}
		log.Printf("QingGong capability Buff staged id=%s property=%d value=%q expiry=%d", id, bufferInfoPropertyIndex(spec.slot), buffState, expires.UnixMilli())
	}
	return true, nil
}

// grantRedSuperArmor 更新 BufferInfo1，它进入 FxGameLogic 的原生
// BuffStore 设置器。S->C 自定义 0x1E 并不是这条服务器到客户端的路由。
func grantRedSuperArmor(conn sceneMessageConnection, player *playerActor) (time.Time, error) {
	if player == nil {
		return time.Time{}, fmt.Errorf("character has not entered a scene")
	}
	now := time.Now()
	buffState, expires := player.activateRedSuperArmor(now)
	log.Printf("Buff add state player=%d-%d property=%d value=%q expiry=%d", playerObjectID, playerOwnerID, bufferInfoPropertyIndex(1), buffState, expires.UnixMilli())
	update, err := player.vitalUpdate()
	if err != nil {
		return time.Time{}, fmt.Errorf("encode red Buff state: %w", err)
	}
	if err := conn.WriteFrame(update); err != nil {
		return time.Time{}, fmt.Errorf("write red Buff state: %w", err)
	}
	return expires, nil
}

func activateQingGongBuff(player *playerActor, definition qinggong.Definition, now time.Time) (alreadyActive bool, expires time.Time, err error) {
	spec, exists := qinggongBuffFor(definition)
	if !exists {
		return false, time.Time{}, nil
	}
	if spec.clearOnEnd && player.hasActiveBuff(spec.slot, spec.staticData, now) {
		return true, time.Time{}, nil
	}
	buffState, expires := player.activateBuff(spec.slot, spec.staticData, spec.duration, now)
	log.Printf("QingGong Buff staged id=%s property=%d value=%q expiry=%d", definition.ID, bufferInfoPropertyIndex(spec.slot), buffState, expires.UnixMilli())
	return false, expires, nil
}

func endQingGongBuff(conn sceneMessageConnection, player *playerActor, definition qinggong.Definition) error {
	spec, exists := qinggongBuffFor(definition)
	if !exists || !spec.clearOnEnd {
		return nil
	}
	if _, changed := player.clearBuff(spec.slot, spec.staticData); !changed {
		return nil
	}
	update, err := player.vitalUpdate()
	if err != nil {
		return fmt.Errorf("encode qinggong Buff removal: %w", err)
	}
	if err := conn.WriteFrame(update); err != nil {
		return fmt.Errorf("write qinggong Buff removal: %w", err)
	}
	log.Printf("QingGong Buff remove id=%s property=%d", definition.ID, bufferInfoPropertyIndex(spec.slot))
	return nil
}

// writeGlideTimedBuffAdd 推 msg169 缓降挂帧（addRemove=0，v7=租约到期
// Unix 毫秒）。帧序对齐 Java engageGlide：msg169 先于槽帧（vitalUpdate）。
func writeGlideTimedBuffAdd(conn sceneMessageConnection, player *playerActor, expires time.Time) error {
	frame, err := glideTimedBuffFrame(0, expires.UnixMilli(), playerObjectID, playerOwnerID, player.snapshotName())
	if err != nil {
		return fmt.Errorf("encode glide buff add: %w", err)
	}
	if err := conn.WriteFrame(frame); err != nil {
		return fmt.Errorf("write glide buff add: %w", err)
	}
	return nil
}

// writeGlideTimedBuffRemove 推 msg169 缓降摘帧（addRemove=1）。endAt
// 取租约原到期（buffExpiry 命中时），游离摘（租约已不在）回落当前
// 时刻但仍回帧——客户端状态重同步（Java releaseGlide 同口径）。
func writeGlideTimedBuffRemove(conn sceneMessageConnection, player *playerActor, now time.Time) error {
	endAt := now
	if expires, ok := player.buffExpiry(qgLightBuffSlot, glideBuffStaticData, now); ok {
		endAt = expires
	}
	frame, err := glideTimedBuffFrame(1, endAt.UnixMilli(), playerObjectID, playerOwnerID, player.snapshotName())
	if err != nil {
		return fmt.Errorf("encode glide buff removal: %w", err)
	}
	if err := conn.WriteFrame(frame); err != nil {
		return fmt.Errorf("write glide buff removal: %w", err)
	}
	return nil
}

// writeGlideBuffRemovalFor 耗尽强摘/槽租约到期共用的摘帧入口：先读
// buffExpiry 再清槽（endQingGongBuff 清槽后到期时刻就丢了）。
func writeGlideBuffRemovalFor(conn sceneMessageConnection, player *playerActor, definition qinggong.Definition, now time.Time) {
	if definition.BufferID != qingGongGlideBuffID {
		return
	}
	if err := writeGlideTimedBuffRemove(conn, player, now); err != nil {
		log.Printf("glide buff removal frame: %v", err)
	}
}

// scheduleRedSuperArmorExpiry 在到期时清除同一份原生状态。
func scheduleRedSuperArmorExpiry(conn sceneMessageConnection, player *playerActor, expires time.Time) {
	schedulePlayerBuffExpiry(conn, player, redSuperArmorBuffSlot, redSuperArmorStaticData, expires)
}

func schedulePlayerBuffExpiry(conn sceneMessageConnection, player *playerActor, slot uint16, staticData uint32, expires time.Time) {
	delay := time.Until(expires)
	if delay < 0 {
		delay = 0
	}
	time.AfterFunc(delay, func() {
		if _, active := player.expireBuff(slot, staticData, expires, time.Now()); !active {
			return
		}
		update, err := player.vitalUpdate()
		if err != nil {
			log.Printf("Buff remove encode: %v", err)
			return
		}
		if err := conn.WriteFrame(update); err != nil {
			log.Printf("Buff remove state: %v", err)
			return
		}
		log.Printf("Buff remove state player=%d-%d property=%d expiry=%d", playerObjectID, playerOwnerID, bufferInfoPropertyIndex(slot), expires.UnixMilli())
		// 缓降槽租约到期（Java glideTick lease-expired → releaseGlide）：
		// 空槽帧之后补 msg169 摘帧，v7 取本定时器钉死的原到期时刻。
		if slot == qgLightBuffSlot && staticData == glideBuffStaticData {
			if frame, err := glideTimedBuffFrame(1, expires.UnixMilli(), playerObjectID, playerOwnerID, player.snapshotName()); err == nil {
				if err := conn.WriteFrame(frame); err != nil {
					log.Printf("glide buff removal state: %v", err)
				}
			} else {
				log.Printf("glide buff removal encode: %v", err)
			}
		}
	})
}
