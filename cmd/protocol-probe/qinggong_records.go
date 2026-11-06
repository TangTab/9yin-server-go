package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/local/9yin-go-server/internal/qinggong"
)

// A record schema is negotiated once for a scene connection by opcode 0x0A.
// The current client uses QingGongRec (not QingGongPoint) to answer
// WuXueQuery:CheckLearn_QingGong, so the lightness resource and the learned
// skill list deliberately travel on separate protocol paths.
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

// qinggongBuffFor derives native movement state from the installed modern
// QGDefine.ini BufferID.  These BuffStore entries are what the client needs
// in addition to movement stats and the accepted 216 resource charge.
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
	// FxNet2!GameReceiver::RecvRecordRow (0x1060BC40) decodes schema type 1
	// as one byte, type 2 as two bytes, and type 3 as a four-byte integer.
	// Record rows below encode their integer cells with AppendUint32, including
	// Task_Completed's 32-bit bitset; they must therefore be declared as type 3.
	recordColumnInt = byte(3)

	// FxNet2!GameReceiver::RecvRecordRow decodes schema type 4 as a signed
	// eight-byte integer.  cooldown_rec uses this for its start/end columns.
	recordColumnInt64 = byte(4)

	// FxNet2!ServerRecordGrid (0x1060BC40) parses schema type 7 as a
	// length-prefixed, NUL-terminated narrow string. Type 6 is an IEEE-754
	// double, so declaring this column as 6 makes the client consume the first
	// eight bytes of "qinggong_*" as a number and discard the row semantically.
	recordColumnString = byte(7)

	recordQingGong       uint16 = 0
	recordActiveQingGong uint16 = 1
	recordTaskAccepted   uint16 = 2
	recordTaskCompleted  uint16 = 3
	recordShortcut       uint16 = 4
	recordCooldown       uint16 = 5
	viewportQingGong     uint16 = 46
	// View objects use the original scene-property ConfigID field.  The later
	// shop-only compatibility field 105 has the same display name but is not
	// what QingGongContainer's native lookup reads.
	qingGongConfigProperty uint16 = 7
)

var qingGongRecordSchemas = []recordSchema{
	{name: "QingGongRec", colTypes: []byte{recordColumnString}},
	{name: "ActiveQGSkillRec", colTypes: []byte{recordColumnString}},
	// This is deliberately the seven-column compatibility prefix consumed by
	// condition 23614: id through status.  The client negotiates record width
	// from this table; declaring the unrecovered 26-column task runtime while
	// sending only this prefix makes RecvRecordRow commit these seven cells and
	// then throw Out range on column 7.
	{name: "Task_Accepted", colTypes: []byte{
		recordColumnInt, recordColumnInt, recordColumnInt,
		recordColumnString, recordColumnString, recordColumnString,
		recordColumnInt,
	}},
	{name: "Task_Completed", colTypes: []byte{recordColumnInt}},
	// shortcut_game.lua looks this record up before it calls
	// custom_set_shortcut(200). Without the record itself, the drag operation
	// is rejected locally and no request ever reaches the server.
	{name: "shortcut_rec", colTypes: []byte{
		recordColumnInt, recordColumnString, recordColumnString,
	}},
	// CoolLogicManager registers a native callback for this player record.
	// Each added row becomes StartACool(category, end-start, team, 0).
	{name: "cooldown_rec", colTypes: []byte{
		recordColumnInt, recordColumnInt64, recordColumnInt64, recordColumnInt,
	}},
}

// starterQingGongIDs are the current client's base categories.  They are
// deliberately config IDs, not visual labels, so the local client resolves
// their current QGDefine/QGSkill data itself.
var starterQingGongIDs = []string{
	"qinggong_1", "qinggong_2", "qinggong_3", "qinggong_4", "qinggong_5",
	"qinggong_6", "qinggong_7", "qinggong_8", "qinggong_9", "qinggong_10",
	"qinggong_11", "qinggong_12", "qinggong_14", "qinggong_15", "qinggong_16",
	"qinggong_17", "qinggong_26", "qinggong_gaojie",
}

// starterQingGongSkillIDs are actual QG skill instances, not the category
// IDs above. QingGongRec unlocks a category, while WuXueQuery:GetLearnID_QGSkill
// reads these objects from QingGongContainer (View 46) to populate the cards.
var starterQingGongSkillIDs = []string{
	"QG_JH_001", "QG_JH_002", "QG_JH_003", "QG_JH_004", "QG_JH_005", "QG_JH_006", "QG_JH_007", "QG_JH_008", "QG_JH_009", "QG_JH_010",
	"QG_JH_011", "QG_JH_012", "QG_JH_013", "QG_JH_014", "QG_JH_015", "QG_JH_016", "QG_JH_017", "QG_JH_018", "QG_JH_019", "QG_JH_020",
	"QG_JH_021", "QG_JH_022", "QG_JH_023", "QG_JH_024", "QG_JH_025", "QG_JH_026", "QG_JH_027", "QG_JH_028", "QG_JH_029", "QG_JH_02_001",
	"QG_JH_030", "QG_JH_031", "QG_JH_032", "QG_JH_033", "QG_JH_034", "QG_JH_035", "QG_JH_036", "QG_JH_037", "QG_JH_038", "QG_JH_039",
	"QG_JH_03_001", "QG_JH_040", "QG_JH_041", "QG_JH_04_001", "QG_JH_05_001", "QG_JH_06_001", "QG_JH_gj_001", "QG_JH_gj_002", "QG_JH_gj_003", "QG_JH_gj_004", "QG_JH_gj_006", "QG_JH_gjskill_001",
}

// serverRecordTable is the current FxNet2 record schema packet:
// 0x0A | u16 count | (C string name | u16 column count | type bytes)*.
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

// serverRecordAddString encodes SERVER_RECORD_ADD (0x11) for a one-column
// scene-object record.  Player identity must match the 0x0D object identity
// (1-1), otherwise GetPlayer()/QueryRecord cannot see the new row.
func serverRecordAddString(objectID, ownerID uint32, recordIndex uint16, value string) []byte {
	msg := make([]byte, 16, 16+len(value)+5)
	msg[0] = 0x11
	msg[1] = 0 // scene object
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], 0) // first column
	binary.LittleEndian.PutUint16(msg[14:], 1) // one string column
	msg = binary.LittleEndian.AppendUint32(msg, uint32(len(value)+1))
	msg = append(msg, value...)
	return append(msg, 0)
}

// serverRecordAddInt32 encodes one integer record cell. SERVER_RECORD_ADD
// appends a row, so callers which need a sparse row must emit earlier rows.
func serverRecordAddInt32(objectID, ownerID uint32, recordIndex uint16, value uint32) []byte {
	msg := make([]byte, 20)
	msg[0] = 0x11
	msg[1] = 0 // scene object
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], 0) // first column
	binary.LittleEndian.PutUint16(msg[14:], 1) // one int column
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

// serverRecordAddCells writes one complete row using the negotiated schema.
// Packet +0x0C/+0x0E are the source row start/count consumed by
// FxNet2::RecvRecordRow, not a column start/count pair.  Every call here
// carries exactly one full row, so the count must remain 1.
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
	binary.LittleEndian.PutUint16(msg[12:], 0) // first source row
	binary.LittleEndian.PutUint16(msg[14:], 1) // one complete row
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

	// The direct branches of condition 23614 read Task_Accepted status column
	// 6 for 1069--1073 and 1344.  The completed-task branch (26303) covers the
	// same family, so the compatibility server exposes both proven sources.
	facultyGuideTaskFirst uint32 = 1069
	facultyGuideTaskLast  uint32 = 1073
	facultyAltTaskID      uint32 = 1344

	// Condition 23613 gates the 内功面板的演武 button before it emits the
	// 1006/21 request.  Its direct branches read Task_Accepted status column 6
	// for 1140--1147 and 51002; nested formula 23647 checks their completed
	// bits.  Keep both because live evidence showed the completed bits alone
	// did not enable btn_act.
	facultyActTaskFirst uint32 = 1140
	facultyActTaskLast  uint32 = 1147
	facultyActAltTaskID uint32 = 51002
)

// taskCompletedBit follows the modern FxGameLogic predicate at 0x11854870.
// It maps one-based task serials to 32-bit Task_Completed record rows/bits.
// Task 1346 resolves to row 42, bit 3: mask 0x00000004.
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

// grantFunctionUnlock completes the standard main-function prerequisite and
// both modern faculty prerequisite families.  The bounded seven-column
// Task_Accepted rows attach the active-task gates to the player, while the
// verified Task_Completed bitset retains their alternate completed branches.
// Because Task_Completed is append-only, every preceding row must be emitted,
// including zero rows before the rows that contain the required bits.
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

// grantStarterQingGong writes both client-side truth sources after ClientReady:
// QingGongRec powers CheckLearn_QingGong while View 46 backs condition checks
// against QingGongContainer.
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

// activateQingGongRecord implements CLIENT_CUSTOMMSG_ACTIVE_QINGGONG (217).
// The current Lua UI's check_qgskill_is_avtive queries ActiveQGSkillRec
// before it permits an equipped qinggong_gaojie action. This is distinct from
// 216, which spends the movement resource only after an action is already
// active.
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

// grantRedSuperArmor updates BufferInfo1, which enters FxGameLogic's native
// BuffStore setter. S->C custom 0x1E is not this server-to-client route.
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

// scheduleRedSuperArmorExpiry clears the same native state at expiry.
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
