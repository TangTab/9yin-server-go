package main

import (
	"encoding/binary"
	"slices"
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

// native-menu grantFunctionUnlock 的 23 行 Task_Accepted 顺序清单
// （62311/49840 两会话 #124-#146 实证）：门派引导链 1555--1562 铺在
// 1344 之后、1140--1147 之前，51002 收尾。
func TestFacultyAcceptedTaskIDsIncludeSchoolLeadFamily(t *testing.T) {
	ids := facultyAcceptedTaskIDs()
	want := []uint32{
		1069, 1070, 1071, 1072, 1073, 1344,
		1555, 1556, 1557, 1558, 1559, 1560, 1561, 1562,
		1140, 1141, 1142, 1143, 1144, 1145, 1146, 1147,
		51002,
	}
	if !slices.Equal(ids, want) {
		t.Fatalf("accepted ids=%v want %v", ids, want)
	}
	// 门派引导链只进 Task_Accepted，不进 Task_Completed 位集：
	// 位图行值与既有断言保持不变（42 行仍只有 1344/1346 位）。
	taskIDs := []uint32{functionUnlockTaskID, facultyAltTaskID, facultyActAltTaskID}
	for taskID := facultyGuideTaskFirst; taskID <= facultyGuideTaskLast; taskID++ {
		taskIDs = append(taskIDs, taskID)
	}
	for taskID := facultyActTaskFirst; taskID <= facultyActTaskLast; taskID++ {
		taskIDs = append(taskIDs, taskID)
	}
	rows, finalRow := taskCompletedRows(taskIDs)
	if finalRow != 1593 || rows[42] != 0x00000005 {
		t.Fatalf("completed bitmap drifted: finalRow=%d row42=0x%08X", finalRow, rows[42])
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
	if got := player.bufferListString(now); !strings.Contains(got, "buf_qg_drift,local-2,1,285212673-8018955,") {
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
	if got := player.bufferListString(now); !strings.Contains(got, "buf_qg_dst,local-3,1,285212673-8018955,") {
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
	// 功能解锁行已拆到 grantFunctionUnlock（选角链内下发），此处只有
	// QingGongRec 行 + View46 容器 + 容器项。
	wantFrames := len(starterQingGongIDs) + 1 + len(starterQingGongIDs) + len(starterQingGongSkillIDs)
	if len(conn.frames) != wantFrames {
		t.Fatalf("frames=%d want=%d", len(conn.frames), wantFrames)
	}
	if conn.frames[0][0] != 0x11 || conn.frames[len(starterQingGongIDs)][0] != 0x15 || conn.frames[len(starterQingGongIDs)+1][0] != 0x18 {
		t.Fatalf("unexpected grant sequence first=%x create=%x add=%x", conn.frames[0], conn.frames[len(starterQingGongIDs)], conn.frames[len(starterQingGongIDs)+1])
	}
}

// grantFunctionUnlock 独立于 ClientReady（选角链内）: 23 任务行 + 204 采集
// 行 + Task_Completed 位图 + 19 门任务值行（E 模式；顺序无关断言——并行
// 会话会重排帧序）。
func TestGrantFunctionUnlockStagesTaskBitmap(t *testing.T) {
	conn := &captureMessageConnection{}
	if err := grantFunctionUnlock(conn); err != nil {
		t.Fatal(err)
	}
	_, finalTaskRow := taskCompletedRows([]uint32{facultyActAltTaskID})
	// 值行(E)已被证伪默认关停；位图含功能开放门任务 49634(row1551|0x4)。
	wantFrames := len(facultyAcceptedTaskIDs()) + len(collectSkillIDs) + int(finalTaskRow) + 1
	if len(conn.frames) != wantFrames {
		t.Fatalf("frames=%d want=%d", len(conn.frames), wantFrames)
	}
	var bitmapRows []uint32
	for _, f := range conn.frames {
		if f[0] == 0x11 && binary.LittleEndian.Uint16(f[10:]) == recordTaskCompleted {
			bitmapRows = append(bitmapRows, binary.LittleEndian.Uint32(f[16:]))
		}
	}
	if len(bitmapRows) != int(finalTaskRow)+1 {
		t.Fatalf("bitmap rows=%d want %d", len(bitmapRows), int(finalTaskRow)+1)
	}
	// 暮色初章族 1344/1346 → row 42 bits 0x1/0x4。
	if got := bitmapRows[42]; got&0x5 != 0x5 {
		t.Fatalf("bitmap row42=%#x want bits 0x5 (task 1344+1346)", got)
	}
	// 功能开放总门 task 49634 → row 1551 mask 0x4（真机二分定案）。
	if got := bitmapRows[1551]; got&0x4 != 0x4 {
		t.Fatalf("bitmap row1551=%#x want bit 0x4 (gate task 49634)", got)
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
	if !strings.Contains(string(frame), "21EA,11000001,7A5C0B,1,") {
		t.Fatalf("missing native BufferInfo serialization: %q", frame)
	}
	if !strings.Contains(string(frame), "buf_fixed_red,local-1,1,285212673-8018955,") {
		t.Fatalf("missing native BufferListStr serialization: %q", frame)
	}
}
