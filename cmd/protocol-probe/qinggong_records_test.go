package main

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/qinggong"
)

func TestQingGongRecordSchemaAndRowLayout(t *testing.T) {
	table, err := serverRecordTable(qingGongRecordSchemas)
	if err != nil {
		t.Fatal(err)
	}
	if table[0] != 0x0A || binary.LittleEndian.Uint16(table[1:]) != 9 {
		t.Fatalf("record table header=%x", table)
	}
	wantPrefix := []byte{0x0A, 9, 0,
		'Q', 'i', 'n', 'g', 'G', 'o', 'n', 'g', 'R', 'e', 'c', 0, 1, 0, 7,
		'A', 'c', 't', 'i', 'v', 'e', 'Q', 'G', 'S', 'k', 'i', 'l', 'l', 'R', 'e', 'c', 0, 1, 0, 7}
	if len(table) < len(wantPrefix) || string(table[:len(wantPrefix)]) != string(wantPrefix) {
		t.Fatalf("record table prefix=%x want=%x", table, wantPrefix)
	}
	if got := len(qingGongRecordSchemas[recordTaskAccepted].colTypes); got != 7 {
		t.Fatalf("Task_Accepted compatibility column count=%d, want 7", got)
	}
	if schema := qingGongRecordSchemas[recordShortcut]; schema.name != "shortcut_rec" || len(schema.colTypes) != 3 ||
		schema.colTypes[0] != recordColumnInt || schema.colTypes[1] != recordColumnString || schema.colTypes[2] != recordColumnString {
		t.Fatalf("shortcut schema=%+v", schema)
	}
	if schema := qingGongRecordSchemas[recordCooldown]; schema.name != "cooldown_rec" ||
		len(schema.colTypes) != 4 || schema.colTypes[0] != recordColumnInt ||
		schema.colTypes[1] != recordColumnInt64 || schema.colTypes[2] != recordColumnInt64 ||
		schema.colTypes[3] != recordColumnInt {
		t.Fatalf("cooldown schema=%+v", schema)
	}
	// 外观域三记录（docs/装备外观与风物志挂载.md：FashionSuitRec
	// [7,3,3,4,3,3] / CardRec [3] 位图 / UseCardRec [2,2] Word 双列）。
	if schema := qingGongRecordSchemas[recordFashionSuit]; schema.name != "FashionSuitRec" ||
		len(schema.colTypes) != 6 || schema.colTypes[0] != recordColumnString ||
		schema.colTypes[1] != recordColumnInt || schema.colTypes[2] != recordColumnInt ||
		schema.colTypes[3] != recordColumnInt64 || schema.colTypes[4] != recordColumnInt ||
		schema.colTypes[5] != recordColumnInt {
		t.Fatalf("fashion suit schema=%+v", schema)
	}
	if schema := qingGongRecordSchemas[recordCard]; schema.name != "CardRec" ||
		len(schema.colTypes) != 1 || schema.colTypes[0] != recordColumnInt {
		t.Fatalf("card schema=%+v", schema)
	}
	if schema := qingGongRecordSchemas[recordUseCard]; schema.name != "UseCardRec" ||
		len(schema.colTypes) != 2 || schema.colTypes[0] != recordColumnWord ||
		schema.colTypes[1] != recordColumnWord {
		t.Fatalf("use card schema=%+v", schema)
	}
	row := serverRecordAddString(1, 1, recordQingGong, "qinggong_8")
	if len(row) != 16+4+len("qinggong_8")+1 || row[0] != 0x11 || row[1] != 0 {
		t.Fatalf("record row=%x", row)
	}
	if binary.LittleEndian.Uint32(row[2:]) != 1 || binary.LittleEndian.Uint32(row[6:]) != 1 || binary.LittleEndian.Uint16(row[10:]) != recordQingGong || binary.LittleEndian.Uint16(row[12:]) != 0 || binary.LittleEndian.Uint16(row[14:]) != 1 {
		t.Fatalf("record row header=%x", row[:16])
	}
	if n := binary.LittleEndian.Uint32(row[16:]); n != uint32(len("qinggong_8")+1) || row[len(row)-1] != 0 {
		t.Fatalf("record row string archive=%x", row[16:])
	}
}

func TestFunctionUnlockTaskCompletedRecord(t *testing.T) {
	row, mask := taskCompletedBit(functionUnlockTaskID)
	if row != 42 || mask != 0x4 {
		t.Fatalf("task %d mapping = row %d mask 0x%X, want row 42 mask 0x4", functionUnlockTaskID, row, mask)
	}
	frame := serverRecordAddInt32(1, 1, recordTaskCompleted, mask)
	if len(frame) != 20 || frame[0] != 0x11 {
		t.Fatalf("invalid int record frame: % X", frame)
	}
	if got := binary.LittleEndian.Uint16(frame[10:]); got != recordTaskCompleted {
		t.Fatalf("record index = %d, want %d", got, recordTaskCompleted)
	}
	if got := binary.LittleEndian.Uint32(frame[16:]); got != mask {
		t.Fatalf("record value = 0x%X, want 0x%X", got, mask)
	}
}

func TestFacultyGuideCompletedTaskRows(t *testing.T) {
	taskIDs := []uint32{functionUnlockTaskID, facultyAltTaskID, facultyActAltTaskID}
	for taskID := facultyGuideTaskFirst; taskID <= facultyGuideTaskLast; taskID++ {
		taskIDs = append(taskIDs, taskID)
	}
	for taskID := facultyActTaskFirst; taskID <= facultyActTaskLast; taskID++ {
		taskIDs = append(taskIDs, taskID)
	}
	rows, finalRow := taskCompletedRows(taskIDs)
	if finalRow != 1593 {
		t.Fatalf("final completed-task row = %d, want 1593", finalRow)
	}
	// 1069-1073 占据第 33 行的第 13-17 位。1344 和 1346 占据第 42 行的
	// 第 0 位和第 2 位。
	if got := rows[33]; got != 0x0003E000 {
		t.Fatalf("guide row 33 = 0x%08X, want 0x0003E000", got)
	}
	if got := rows[42]; got != 0x00000005 {
		t.Fatalf("unlock row 42 = 0x%08X, want 0x00000005", got)
	}
	// 完整的 23613 分支由任务 1140--1147（第 35 行）加上 51002（第 1593
	// 行）组成；它们共同门控修炼面板的演武按钮。
	if got := rows[35]; got != 0x0FF00000 {
		t.Fatalf("activity-faculty row 35 = 0x%08X, want 0x0FF00000", got)
	}
	if got := rows[1593]; got != 0x04000000 {
		t.Fatalf("activity-faculty row 1593 = 0x%08X, want 0x04000000", got)
	}
}

func TestRecordTableExcludesUnrecoveredReputeRecord(t *testing.T) {
	for _, schema := range qingGongRecordSchemas {
		if schema.name == "Repute_Record" {
			t.Fatalf("unrecovered condition record leaked into current schema: %s", schema.name)
		}
	}
}

func TestQingGongWaterBuffsUseSeparateNativeSlots(t *testing.T) {
	player := newPlayerActor("测试", 0)
	drift := qinggong.Definition{ID: "qinggong_2", BufferID: "buf_qg_drift"}
	jump := qinggong.Definition{ID: "qinggong_3", BufferID: "buf_qg_dst"}

	conn := &captureMessageConnection{}
	now := time.Now()
	already, expires, err := activateQingGongBuff(player, drift, now)
	if err != nil || already || expires.IsZero() {
		t.Fatalf("activate drift already=%t expires=%v err=%v", already, expires, err)
	}
	if got := player.bufferListString(now); !strings.Contains(got, "buf_qg_drift,local-2,1,1-1,") {
		t.Fatalf("drift BufferListStr=%q", got)
	}
	already, _, err = activateQingGongBuff(player, drift, now)
	if err != nil || !already {
		t.Fatalf("duplicate drift already=%t err=%v", already, err)
	}
	if err := endQingGongBuff(conn, player, drift); err != nil || len(conn.frames) != 1 || conn.frames[0][0] != 0x10 || binary.LittleEndian.Uint16(conn.frames[0][10:]) < 2 {
		t.Fatalf("end drift err=%v frames=%x", err, conn.frames)
	}

	conn = &captureMessageConnection{}
	if already, expires, err = activateQingGongBuff(player, jump, now); err != nil || already || expires.IsZero() {
		t.Fatalf("activate jump already=%t expires=%v err=%v", already, expires, err)
	}
	if got := player.bufferListString(now); !strings.Contains(got, "buf_qg_dst,local-3,1,1-1,") {
		t.Fatalf("jump BufferListStr=%q", got)
	}
}

func TestRedSuperArmorRulesMatchPropPack9679(t *testing.T) {
	player := newPlayerActor("测试", 0)
	now := time.Now()
	if rules := player.buffRules(now); rules.CantHitEffect || rules.CantBeSkillLocked {
		t.Fatalf("inactive red rules=%+v", rules)
	}
	player.activateRedSuperArmor(now)
	if rules := player.buffRules(now); !rules.CantHitEffect || !rules.CantBeSkillLocked {
		t.Fatalf("active red rules=%+v", rules)
	}
}

func TestActivateQingGongWritesActiveSkillRecord(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	changed, err := activateQingGongRecord(conn, player, "QG_JH_gj_004")
	if err != nil || !changed || len(conn.frames) != 1 {
		t.Fatalf("activate changed=%t err=%v frames=%d", changed, err, len(conn.frames))
	}
	frame := conn.frames[0]
	if frame[0] != 0x11 || binary.LittleEndian.Uint16(frame[10:]) != recordActiveQingGong || !strings.Contains(string(frame), "QG_JH_gj_004") {
		t.Fatalf("active qinggong record frame=%x", frame)
	}
	changed, err = activateQingGongRecord(conn, player, "QG_JH_gj_004")
	if err != nil || changed || len(conn.frames) != 1 {
		t.Fatalf("duplicate activation changed=%t err=%v frames=%d", changed, err, len(conn.frames))
	}
}

func TestGrantStarterQingGongUsesRecordAndView(t *testing.T) {
	conn := &captureMessageConnection{}
	if err := grantStarterQingGong(conn); err != nil {
		t.Fatal(err)
	}
	_, finalTaskRow := taskCompletedRows([]uint32{facultyActAltTaskID})
	facultyAcceptedRows := len(facultyAcceptedTaskIDs())
	wantFrames := len(starterQingGongIDs) + 1 + len(starterQingGongIDs) + len(starterQingGongSkillIDs) + facultyAcceptedRows + int(finalTaskRow) + 1
	if len(conn.frames) != wantFrames {
		t.Fatalf("frames=%d want=%d", len(conn.frames), wantFrames)
	}
	if conn.frames[0][0] != 0x11 || conn.frames[len(starterQingGongIDs)][0] != 0x15 || conn.frames[len(starterQingGongIDs)+1][0] != 0x18 {
		t.Fatalf("unexpected grant sequence first=%x create=%x add=%x", conn.frames[0], conn.frames[len(starterQingGongIDs)], conn.frames[len(starterQingGongIDs)+1])
	}
	firstViewItem := conn.frames[len(starterQingGongIDs)+1]
	if got := binary.LittleEndian.Uint16(firstViewItem[7:]); got != qingGongConfigProperty {
		t.Fatalf("QingGongContainer ConfigID property=%d want=%d", got, qingGongConfigProperty)
	}
	completedRowsStart := len(conn.frames) - (int(finalTaskRow) + 1)
	accepted := conn.frames[completedRowsStart-facultyAcceptedRows]
	if accepted[0] != 0x11 || binary.LittleEndian.Uint16(accepted[10:]) != recordTaskAccepted || binary.LittleEndian.Uint16(accepted[14:]) != 1 {
		t.Fatalf("faculty Task_Accepted row=%x", accepted)
	}
	if got := binary.LittleEndian.Uint32(accepted[16:]); got != facultyGuideTaskFirst {
		t.Fatalf("first faculty Task_Accepted id=%d want=%d", got, facultyGuideTaskFirst)
	}
	lastAccepted := conn.frames[completedRowsStart-1]
	if got := binary.LittleEndian.Uint32(lastAccepted[16:]); got != facultyActAltTaskID {
		t.Fatalf("last faculty Task_Accepted id=%d want=%d", got, facultyActAltTaskID)
	}
	guideRow := conn.frames[completedRowsStart+33]
	if got := binary.LittleEndian.Uint32(guideRow[16:]); got != 0x0003E000 {
		t.Fatalf("faculty-guide completed row=%x", guideRow)
	}
	last := conn.frames[completedRowsStart+42]
	if last[0] != 0x11 || binary.LittleEndian.Uint16(last[10:]) != recordTaskCompleted || binary.LittleEndian.Uint32(last[16:]) != 0x5 {
		t.Fatalf("last function/faculty-unlock record frame=%x", last)
	}
}

func TestGrantRedSuperArmorUsesNativeBufferInfoProperty(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	if _, err := grantRedSuperArmor(conn, player); err != nil {
		t.Fatal(err)
	}
	if len(conn.frames) != 1 {
		t.Fatalf("frames=%d want=1", len(conn.frames))
	}
	frame := conn.frames[0]
	if frame[0] != 0x10 || binary.LittleEndian.Uint16(frame[10:]) < 2 {
		t.Fatalf("BufferInfo update header=%x", frame)
	}
	if !strings.Contains(string(frame), "21EA,1,1,1,") {
		t.Fatalf("missing native BufferInfo serialization: %q", frame)
	}
	if !strings.Contains(string(frame), "buf_fixed_red,local-1,1,1-1,") {
		t.Fatalf("missing native BufferListStr serialization: %q", frame)
	}
}
