package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// 官服 c.pcapng #111 的 414 gamestep 步骤帧逐字节（step=6）：
// 1e 04 00 02 9e010000 06 09..gamestep\0 02 06000000 02 01000000。
func TestJyzStepFrameMatchesOfficialCapture(t *testing.T) {
	frame, err := jyzStepFrame(defaultJyzConfig())
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{
		0x1e, 0x04, 0x00, // 0x1E 壳 + 4 值
		0x02, 0x9e, 0x01, 0x00, 0x00, // int 414
		0x06, 0x09, 0x00, 0x00, 0x00, // 窄串 len 9
		'g', 'a', 'm', 'e', 's', 't', 'e', 'p', 0x00,
		0x02, 0x06, 0x00, 0x00, 0x00, // int 6 = step
		0x02, 0x01, 0x00, 0x00, 0x00, // int 1 = flag
	}
	if !bytes.Equal(frame, want) {
		t.Fatalf("jyz step frame=% X, want % X", frame, want)
	}
}

// native 62311 #1680-#1742 帧序：quest49 簇(4 帧) → m416 域声明 → 419 步骤 → data_record 打包。
func TestPushJyzRecordsWritesQuestClusterStepAndPackedRows(t *testing.T) {
	conn := &captureMessageConnection{}
	if err := pushJyzRecords(conn, defaultJyzConfig(), nil); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 7 {
		t.Fatalf("frames=%d want=7 (quest4 + 416 + 419 + data)", len(frames))
	}
	// quest49 簇: Task_Accepted(29) 行 taskid=49、Task_Record(30)、NewRideRec(310)×2
	if got := binary.LittleEndian.Uint16(frames[0][10:]); got != recordTaskAccepted {
		t.Fatalf("frame0 record=%d want=%d", got, recordTaskAccepted)
	}
	if id := binary.LittleEndian.Uint32(frames[0][16:]); id != 49 {
		t.Fatalf("task49 row id=%d", id)
	}
	if !bytes.Contains(frames[0], []byte("Transbron03A")) || !bytes.Contains(frames[0], []byte("title_49")) {
		t.Fatalf("task49 row missing strings: % X", frames[0])
	}
	if got := binary.LittleEndian.Uint16(frames[1][10:]); got != recordTaskTrace {
		t.Fatalf("frame1 record=%d want Task_Record(30)", got)
	}
	if !bytes.Contains(frames[1], []byte("task_TrackInfo_49")) || !bytes.Contains(frames[1], []byte("49_9_Transbron03A")) {
		t.Fatalf("trace row missing strings: % X", frames[1])
	}
	for k, wantPack := range []uint32{9513, 41523} {
		f := frames[2+k]
		if got := binary.LittleEndian.Uint16(f[10:]); got != recordNewRide {
			t.Fatalf("ride frame record=%d", got)
		}
		if pack := binary.LittleEndian.Uint32(f[16:]); pack != wantPack {
			t.Fatalf("ride %d pack=%d want=%d", k, pack, wantPack)
		}
	}
	// m416 域声明（官服 c.pcapng 4.71s，419 前置）：["DomainJiuYinZhi", step]
	want416 := []byte{
		0x1e, 0x03, 0x00, // 0x1E 壳 + 3 值
		0x02, 0xa0, 0x01, 0x00, 0x00, // int 416
		0x06, 0x10, 0x00, 0x00, 0x00, // 窄串 len 16（含 NUL）
		'D', 'o', 'm', 'a', 'i', 'n', 'J', 'i', 'u', 'Y', 'i', 'n', 'Z', 'h', 'i', 0x00,
		0x02, 0x06, 0x00, 0x00, 0x00, // int 6 = step
	}
	if !bytes.Equal(frames[4], want416) {
		t.Fatalf("416 frame=% X want % X", frames[4], want416)
	}
	// 419 legacy 帧 = native #1683 逐字节
	want419 := []byte{0x1e, 0x02, 0x00, 0x02, 0xa3, 0x01, 0x00, 0x00, 0x02, 0x06, 0x00, 0x00, 0x00}
	if !bytes.Equal(frames[5], want419) {
		t.Fatalf("419 frame=% X want % X", frames[5], want419)
	}
	// data_record 打包 58 行
	row := frames[6]
	if len(row) != 16+2*jyzEventCount {
		t.Fatalf("packed row frame len=%d want=%d", len(row), 16+2*jyzEventCount)
	}
	if got := binary.LittleEndian.Uint16(row[10:]); got != recordDataRecord {
		t.Fatalf("record index=%d want=%d", got, recordDataRecord)
	}
	if got := binary.LittleEndian.Uint16(row[14:]); got != jyzEventCount {
		t.Fatalf("row count=%d want=%d", got, jyzEventCount)
	}
	for i := 0; i < jyzEventCount; i++ {
		if got := binary.LittleEndian.Uint16(row[16+2*i:]); got != 2 {
			t.Fatalf("row %d event state=%d want=2", i, got)
		}
	}
}

// gamestep 收尾帧独立函数（sync 之后调用）。
func TestPushJyzGameStepFinalMatchesOfficialCapture(t *testing.T) {
	conn := &captureMessageConnection{}
	if err := pushJyzGameStepFinal(conn, defaultJyzConfig()); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 1 {
		t.Fatalf("frames=%d want=1", len(frames))
	}
	want := []byte{
		0x1e, 0x04, 0x00,
		0x02, 0x9e, 0x01, 0x00, 0x00,
		0x06, 0x09, 0x00, 0x00, 0x00,
		'g', 'a', 'm', 'e', 's', 't', 'e', 'p', 0x00,
		0x02, 0x06, 0x00, 0x00, 0x00,
		0x02, 0x01, 0x00, 0x00, 0x00,
	}
	if !bytes.Equal(frames[0], want) {
		t.Fatalf("gamestep frame=% X, want % X", frames[0], want)
	}
}

// 官服 #2026：770 明细应答 [0, (id,0,1,state)×8] 逐字节前缀。
func TestJyzDetailFrameMatchesOfficialCapture(t *testing.T) {
	frame, err := jyzDetailFrame()
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{
		0x1e, 0x22, 0x00, // 0x1E 壳 + 34 值（2+8×4）
		0x02, 0x02, 0x03, 0x00, 0x00, // int 770
		0x02, 0x00, 0x00, 0x00, 0x00, // int 0
		0x02, 0x51, 0x27, 0x01, 0x00, // int 75601
		0x02, 0x00, 0x00, 0x00, 0x00, 0x02, 0x01, 0x00, 0x00, 0x00, 0x02, 0x01, 0x00, 0x00, 0x00,
		0x02, 0x52, 0x27, 0x01, 0x00, // int 75602
	}
	if !bytes.Equal(frame[:len(want)], want) {
		t.Fatalf("jyz detail frame=% X, want prefix % X", frame, want)
	}
	if len(frame) != 3+5*(2+4*len(jyzDetailEntries)) {
		t.Fatalf("detail frame len=%d want=173 (官服 #2026 同长)", len(frame))
	}
}

// C2S 770 (player_name, msgID) 应答：m416+419+data 数据部 + gamestep + 770 明细，共五帧（quest 簇不重发）。
func TestHandleJyzQueryCustomAnswersStepAndDetail(t *testing.T) {
	custom := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomJyzQuery},
		{Type: 6, Text: ""},
		{Type: 2, Int32: 6},
	}}
	conn := &captureMessageConnection{}
	if err := handleJyzQueryCustom(conn, custom, "test"); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 5 {
		t.Fatalf("frames=%d want=5", len(frames))
	}
	if frames[0][0] != 0x1e || binary.LittleEndian.Uint32(frames[0][4:8]) != uint32(officialMsgJyzDomain) {
		t.Fatalf("416 frame=% X", frames[0])
	}
	if !bytes.Contains(frames[0], []byte("DomainJiuYinZhi")) {
		t.Fatalf("416 frame missing domain name: % X", frames[0])
	}
	if frames[1][0] != 0x1e || binary.LittleEndian.Uint32(frames[1][4:8]) != uint32(serverJyzLegacyStepMessage) {
		t.Fatalf("419 frame=% X", frames[1])
	}
	if frames[2][0] != 0x11 || binary.LittleEndian.Uint16(frames[2][10:]) != recordDataRecord {
		t.Fatalf("data_record frame=% X", frames[2])
	}
	if frames[3][0] != 0x1e || binary.LittleEndian.Uint32(frames[3][4:8]) != uint32(serverJyzGameStepMessage) {
		t.Fatalf("gamestep frame=% X", frames[3])
	}
	if frames[4][0] != 0x1e || binary.LittleEndian.Uint32(frames[4][4:8]) != uint32(serverJyzDetailMessage) {
		t.Fatalf("detail frame=% X", frames[4])
	}
}

// 复刻 native-menu init 语义：STEP 仅 >0 生效、EVENT_STATE >=0 生效且
// 低 8 位截断，非法值回退缺省。
func TestResolveJyzConfigEnvOverrides(t *testing.T) {
	env := map[string]string{}
	getenv := func(key string) string { return env[key] }

	if got := resolveJyzConfig(getenv); got.step != 6 || got.eventState != 2 {
		t.Fatalf("defaults=%+v", got)
	}
	env["NINEYIN_JYZ_STEP"] = "9"
	if got := resolveJyzConfig(getenv); got.step != 9 || got.eventState != 2 {
		t.Fatalf("step override=%+v", got)
	}
	for _, raw := range []string{"0", "-1", "abc", ""} {
		env["NINEYIN_JYZ_STEP"] = raw
		if got := resolveJyzConfig(getenv); got.step != 6 {
			t.Fatalf("step %q should fall back, got %+v", raw, got)
		}
	}
	env["NINEYIN_JYZ_STEP"] = "9"
	env["NINEYIN_JYZ_EVENT_STATE"] = "0"
	if got := resolveJyzConfig(getenv); got.step != 9 || got.eventState != 0 {
		t.Fatalf("event-state 0=%+v", got)
	}
	env["NINEYIN_JYZ_EVENT_STATE"] = "-1"
	if got := resolveJyzConfig(getenv); got.eventState != 2 {
		t.Fatalf("event-state -1 should fall back, got %+v", got)
	}
	env["NINEYIN_JYZ_EVENT_STATE"] = "300"
	if got := resolveJyzConfig(getenv); got.eventState != 44 {
		t.Fatalf("event-state 300 should truncate to low byte, got %+v", got)
	}
}
