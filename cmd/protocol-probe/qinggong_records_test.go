package main

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/qinggong"
)

func TestQingGongRecordSchemaAndRowLayout(t *testing.T) {
	table, err := serverRecordTable()
	if err != nil {
		t.Fatal(err)
	}
	// 官方 447 schema 方言整表透传：帧头 + 关键下标按名可解析。
	if table[0] != 0x0A || binary.LittleEndian.Uint16(table[1:]) != 447 {
		t.Fatalf("record table header=%x", table[:3])
	}
	for name, want := range map[string]uint16{
		"Task_Accepted":      recordTaskAccepted,
		"Task_Completed":     recordTaskCompleted,
		"shortcut_rec":       recordShortcut,
		"title_rec":          recordTitle,
		"cooldown_rec":       recordCooldown,
		"QingGongRec":        recordQingGong,
		"ActiveQGSkillRec":   recordActiveQingGong,
		"active_jingmai_rec": recordActiveJingMai,
		"FashionSuitRec":     recordFashionSuit,
		"CardRec":            recordCard,
		"UseCardRec":         recordUseCard,
	} {
		if got := clientdata.MustRecordIndexByName(name); got != want {
			t.Fatalf("record %s official index=%d constant=%d", name, got, want)
		}
	}
	// 官方列型抽查：Task_Accepted 28 列、cooldown_rec 末列 u16、
	// title_rec [3,1,4]、ActiveQGSkillRec [7,1]。
	if schema, _ := clientdata.RecordSchemaByIndex(int(recordTaskAccepted)); len(schema.ColumnTypes) != 28 {
		t.Fatalf("Task_Accepted official columns=%d, want 28", len(schema.ColumnTypes))
	}
	if schema, _ := clientdata.RecordSchemaByIndex(int(recordCooldown)); len(schema.ColumnTypes) != 4 || schema.ColumnTypes[3] != recordColumnWord {
		t.Fatalf("cooldown schema=%+v", schema)
	}
	if schema, _ := clientdata.RecordSchemaByIndex(int(recordTitle)); len(schema.ColumnTypes) != 3 ||
		schema.ColumnTypes[0] != recordColumnInt || schema.ColumnTypes[1] != recordColumnByte || schema.ColumnTypes[2] != recordColumnInt64 {
		t.Fatalf("title schema=%+v", schema)
	}
	if schema, _ := clientdata.RecordSchemaByIndex(int(recordActiveQingGong)); len(schema.ColumnTypes) != 2 ||
		schema.ColumnTypes[0] != recordColumnString || schema.ColumnTypes[1] != recordColumnByte {
		t.Fatalf("active qinggong schema=%+v", schema)
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

func TestRecordTablePassesThroughOfficialSchemas(t *testing.T) {
	// 整表透传后官方 447 schema 全量在场（含 Repute_Record 等
	// 未消费记录）——行帧下标才有官方口径可绑定。
	frame, err := serverRecordTable()
	if err != nil {
		t.Fatal(err)
	}
	if string(frame) != string(clientdata.OfficialRecordTableFrame()) {
		t.Fatal("negotiated record table diverged from official frame")
	}
	found := false
	for _, schema := range clientdata.OfficialRecordSchemas() {
		if schema.Name == "Repute_Record" {
			found = true
		}
	}
	if !found {
		t.Fatal("official table missing Repute_Record")
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
