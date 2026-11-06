package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/local/9yin-go-server/internal/qinggong"
)

// 记录 schema 由操作码 0x0A 在场景连接上协商一次。当前客户端用
// QingGongRec（而非 QingGongPoint）来应答
// WuXueQuery:CheckLearn_QingGong，因此轻功值资源和已学技能列表刻意
// 走各自独立的协议路径。
type recordSchema struct {
	name     string
	colTypes []byte
}

type qinggongBuffSpec struct {
	slot       uint16
	staticData uint32
	duration   time.Duration
	clearOnEnd bool
}

// qinggongBuffFor 根据已安装的现代 QGDefine.ini BufferID 推导原生移动
// 状态。这些 BuffStore 条目是客户端在移动数值和已受理的 216 资源扣费
// 之外还需要的东西。
func qinggongBuffFor(definition qinggong.Definition) (qinggongBuffSpec, bool) {
	switch definition.BufferID {
	case "buf_qg_drift":
		return qinggongBuffSpec{slot: waterDriftBuffSlot, staticData: waterDriftStaticData, duration: 10 * time.Minute, clearOnEnd: true}, true
	case "buf_qg_dst":
		return qinggongBuffSpec{slot: waterJumpBuffSlot, staticData: waterJumpStaticData, duration: 3 * time.Second}, true
	default:
		return qinggongBuffSpec{}, false
	}
}

const (
	// FxNet2!GameReceiver::RecvRecordRow (0x1060BC40) 将 schema 类型 1
	// 解码为一个字节，类型 2 为两个字节，类型 3 为四字节整数。下面的
	// 记录行都用 AppendUint32 编码其整型单元格，包括 Task_Completed 的
	// 32 位位集；因此它们必须声明为类型 3。
	recordColumnInt = byte(3)

	// FxNet2!GameReceiver::RecvRecordRow 将 schema 类型 4 解码为带符号的
	// 八字节整数。cooldown_rec 的起始/结束列使用该类型。
	recordColumnInt64 = byte(4)

	// FxNet2!ServerRecordGrid (0x1060BC40) 把 schema 类型 7 解析为带长度
	// 前缀、以 NUL 结尾的窄字符串。类型 6 是 IEEE-754 double，因此把此列
	// 声明为 6 会让客户端把 "qinggong_*" 的前八个字节当作数字消费，并在
	// 语义上丢弃该行。
	recordColumnString = byte(7)

	recordQingGong       uint16 = 0
	recordActiveQingGong uint16 = 1
	recordTaskAccepted   uint16 = 2
	recordTaskCompleted  uint16 = 3
	recordShortcut       uint16 = 4
	recordCooldown       uint16 = 5
	viewportQingGong     uint16 = 46
	// View 对象使用原始的场景属性 ConfigID 字段。后续仅商店用的兼容字段
	// 105 有相同的显示名，但 QingGongContainer 的原生查找读取的不是它。
	qingGongConfigProperty uint16 = 7
)

var qingGongRecordSchemas = []recordSchema{
	{name: "QingGongRec", colTypes: []byte{recordColumnString}},
	{name: "ActiveQGSkillRec", colTypes: []byte{recordColumnString}},
	// 这刻意采用条件 23614 消费的七列兼容前缀：从 id 到 status。客户端
	// 根据此表协商记录宽度；若声明未还原的 26 列任务运行时却只发送这个
	// 前缀，RecvRecordRow 会提交这七个单元格，
	// 然后在第 7 列上抛出 Out range。
	{name: "Task_Accepted", colTypes: []byte{
		recordColumnInt, recordColumnInt, recordColumnInt,
		recordColumnString, recordColumnString, recordColumnString,
		recordColumnInt,
	}},
	{name: "Task_Completed", colTypes: []byte{recordColumnInt}},
	// shortcut_game.lua 在调用 custom_set_shortcut(200) 之前会查找这条
	// 记录。若没有这条记录本身，拖拽操作会在本地被拒绝，任何请求都
	// 到达不了服务器。
	{name: "shortcut_rec", colTypes: []byte{
		recordColumnInt, recordColumnString, recordColumnString,
	}},
	// CoolLogicManager 为这条玩家记录注册了原生回调。每个添加的行都会
	// 成为 StartACool(category, end-start, team, 0)。
	{name: "cooldown_rec", colTypes: []byte{
		recordColumnInt, recordColumnInt64, recordColumnInt64, recordColumnInt,
	}},
}

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

// serverRecordTable 是当前 FxNet2 的记录 schema 包：
// 0x0A | u16 count | (C string name | u16 column count | type bytes)*。
func serverRecordTable(schemas []recordSchema) ([]byte, error) {
	if len(schemas) > math.MaxUint16 {
		return nil, fmt.Errorf("record schema count %d exceeds u16", len(schemas))
	}
	msg := make([]byte, 3)
	msg[0] = 0x0A
	binary.LittleEndian.PutUint16(msg[1:], uint16(len(schemas)))
	for _, schema := range schemas {
		if schema.name == "" {
			return nil, fmt.Errorf("record schema name is empty")
		}
		if len(schema.colTypes) > math.MaxUint16 {
			return nil, fmt.Errorf("record schema %q columns %d exceeds u16", schema.name, len(schema.colTypes))
		}
		msg = append(msg, schema.name...)
		msg = append(msg, 0)
		msg = binary.LittleEndian.AppendUint16(msg, uint16(len(schema.colTypes)))
		msg = append(msg, schema.colTypes...)
	}
	return msg, nil
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
	msg := make([]byte, 20)
	msg[0] = 0x11
	msg[1] = 0 // 场景对象
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], 0) // 第一列
	binary.LittleEndian.PutUint16(msg[14:], 1) // 一个整型列
	binary.LittleEndian.PutUint32(msg[16:], value)
	return msg
}

type recordCell struct {
	intValue   *uint32
	int64Value *int64
	text       *string
}

func recordInt(value uint32) recordCell    { return recordCell{intValue: &value} }
func recordInt64(value int64) recordCell   { return recordCell{int64Value: &value} }
func recordString(value string) recordCell { return recordCell{text: &value} }

// serverRecordAddCells 按协商的 schema 写入一整行。包内 +0x0C/+0x0E 是
// FxNet2::RecvRecordRow 消费的源行起始/数量，不是列的起始/数量对。
// 这里的每次调用都恰好携带一整行，因此该数量
// 必须保持为 1。
func serverRecordAddCells(objectID, ownerID uint32, recordIndex uint16, cells []recordCell) ([]byte, error) {
	if int(recordIndex) >= len(qingGongRecordSchemas) {
		return nil, fmt.Errorf("record index %d has no negotiated schema", recordIndex)
	}
	colTypes := qingGongRecordSchemas[recordIndex].colTypes
	if len(cells) != len(colTypes) {
		return nil, fmt.Errorf("record %q has %d cells, want complete row of %d", qingGongRecordSchemas[recordIndex].name, len(cells), len(colTypes))
	}
	msg := make([]byte, 16)
	msg[0] = 0x11
	msg[1] = 0
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], 0) // 第一个源行
	binary.LittleEndian.PutUint16(msg[14:], 1) // 一整行
	for column, cell := range cells {
		switch colTypes[column] {
		case recordColumnInt:
			if cell.intValue == nil || cell.int64Value != nil || cell.text != nil {
				return nil, fmt.Errorf("record column %d requires int32", column)
			}
			msg = binary.LittleEndian.AppendUint32(msg, *cell.intValue)
		case recordColumnInt64:
			if cell.int64Value == nil || cell.intValue != nil || cell.text != nil {
				return nil, fmt.Errorf("record column %d requires int64", column)
			}
			msg = binary.LittleEndian.AppendUint64(msg, uint64(*cell.int64Value))
		case recordColumnString:
			if cell.text == nil || cell.intValue != nil || cell.int64Value != nil {
				return nil, fmt.Errorf("record column %d requires string", column)
			}
			msg = binary.LittleEndian.AppendUint32(msg, uint32(len(*cell.text)+1))
			msg = append(msg, (*cell.text)...)
			msg = append(msg, 0)
		default:
			return nil, fmt.Errorf("record column %d has unsupported negotiated type %d", column, colTypes[column])
		}
	}
	return msg, nil
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
		frame, err := serverRecordAddCells(playerObjectID, playerOwnerID, recordTaskAccepted, []recordCell{
			recordInt(taskID), recordInt(9), recordInt(0),
			recordString(""), recordString(""), recordString(""), recordInt(0),
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
	if err := conn.WriteFrame(serverRecordAddString(playerObjectID, playerOwnerID, recordActiveQingGong, id)); err != nil {
		return false, fmt.Errorf("add ActiveQGSkillRec %s: %w", id, err)
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
	})
}
