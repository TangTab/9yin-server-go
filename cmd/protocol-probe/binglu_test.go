package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// 兵录域：默认行/零进度/帧族/GM 子命令/943 扩档主链/984 查看。

func TestBingluDefaultsAndProgress(t *testing.T) {
	rows := bingluDefaults()
	if len(rows) != bingluSlotCount {
		t.Fatalf("rows=%d", len(rows))
	}
	if bingluHasProgress(rows) {
		t.Fatal("默认行应为零进度（0 帧口径）")
	}
	rows[3].Cur = 5
	if !bingluHasProgress(rows) {
		t.Fatal("cur>0 应视为有进度")
	}
	if got := bingluMaxCur(rows); got != 5 {
		t.Fatalf("maxCur=%d", got)
	}
	// 上限偏移也是进度。
	fresh := bingluDefaults()
	fresh[9].Max = 15
	if !bingluHasProgress(fresh) {
		t.Fatal("max≠10 应视为有进度")
	}
	if got := (bingluRow{}).effectiveMax(); got != bingluDefaultMaxLevel {
		t.Fatalf("effectiveMax=%d", got)
	}
}

func TestBingluRefreshFrames(t *testing.T) {
	conn := &captureMessageConnection{}
	rows := bingluDefaults()
	rows[1] = bingluRow{Cur: 7, Fill: 120, Max: 10}
	sent, err := bingluRefreshFrames(conn, rows, 0)
	if err != nil || sent != bingluSlotCount {
		t.Fatalf("sent=%d err=%v", sent, err)
	}
	frames := conn.Frames()
	if len(frames) != bingluSlotCount+1 { // 12 行 + MaxBingLuLevel 属性帧
		t.Fatalf("frames=%d, want 13", len(frames))
	}
	if frames[0][0] != 0x11 {
		t.Fatalf("row frame op=%#x", frames[0][0])
	}
	// 属性帧（MaxBingLuLevel=7）在尾部。
	if last := frames[len(frames)-1]; last[0] != 0x10 {
		t.Fatalf("prop frame op=%#x", last[0])
	}
	// 整刷：oldRows=12 → 12 清 + 12 行 + 1 属性 = 25。
	conn2 := &captureMessageConnection{}
	sent, err = bingluRefreshFrames(conn2, rows, bingluSlotCount)
	if err != nil || sent != bingluSlotCount {
		t.Fatalf("resend sent=%d err=%v", sent, err)
	}
	if frames := conn2.Frames(); len(frames) != 2*bingluSlotCount+1 {
		t.Fatalf("frames=%d, want 25", len(frames))
	}
}

func TestApplyBingluArgs(t *testing.T) {
	player := &playerActor{}
	player.restoreBinglu(bingluDefaults())

	// set：单行等级/进度，上限缺省不改。
	rows, text, ok := applyBingluArgs(player, "set 1 7 120")
	if !ok || rows[1].Cur != 7 || rows[1].Fill != 120 || rows[1].Max != 10 {
		t.Fatalf("set: rows[1]=%+v text=%q ok=%v", rows[1], text, ok)
	}
	// set 带上限。
	if rows, _, ok = applyBingluArgs(player, "set 1 12 0 15"); !ok || rows[1].Max != 15 {
		t.Fatalf("set max: %+v", rows[1])
	}
	// max 单行：cur 置有效上限、进度清零、补写上限。
	if rows, text, ok = applyBingluArgs(player, "max 3"); !ok || rows[3].Cur != 10 || rows[3].Fill != 0 || rows[3].Max != 10 {
		t.Fatalf("max 3: %+v text=%q", rows[3], text)
	}
	// max 全部。
	if rows, _, ok = applyBingluArgs(player, "max"); !ok {
		t.Fatalf("max all failed")
	}
	for index, row := range rows {
		if row.Cur != 10 || row.Fill != 0 || row.Max != 10 {
			t.Fatalf("row[%d]=%+v", index, row)
		}
	}
	// 无上限行（max=0）升满：按默认 10 补写。
	player.replaceBinglu([]bingluRow{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}})
	if rows, _, ok = applyBingluArgs(player, "max 0"); !ok || rows[0].Cur != 10 || rows[0].Max != 10 {
		t.Fatalf("max zero-cap: %+v", rows[0])
	}
	// 界值拒绝。
	for _, bad := range []string{"", "bogus", "set 99 1 1", "set 1 16 0", "set 1 1 65536", "set 1 1 0 16", "max 12"} {
		if _, _, ok := applyBingluArgs(player, bad); ok {
			t.Fatalf("args %q should fail", bad)
		}
	}
}

func newBingluTestSession(t *testing.T) (*bingluSession, *captureMessageConnection, *playerActor) {
	t.Helper()
	conn := &captureMessageConnection{}
	player := &playerActor{}
	player.name = "tester"
	session := &bingluSession{link: conn, player: player, remote: "test"}
	return session, conn, player
}

func bingluCustomMessage(msg int32, values ...clientCustomValue) clientCustomMessage {
	all := append([]clientCustomValue{{Type: 2, Int32: msg}}, values...)
	return clientCustomMessage{Values: all}
}

func setBingluFaculty(player *playerActor, value int32) {
	player.mu.Lock()
	player.progress.faculty = value
	player.mu.Unlock()
}

func TestModifyBingluExpand(t *testing.T) {
	setBingluExpandCatalogForTest(map[int32]bingluExpandTier{
		10: {useFaculty: 2_000_000, newMax: 15},
	})
	t.Cleanup(func() { setBingluExpandCatalogForTest(nil) })
	m, conn, player := newBingluTestSession(t)
	player.restoreBinglu(bingluDefaults())
	setBingluFaculty(player, 3_000_000)

	expand := currentBingluExpandCatalog()[10]
	if expand.newMax != 15 || expand.useFaculty != 2_000_000 {
		t.Fatalf("expand tier [10]=%+v", expand)
	}

	message := bingluCustomMessage(clientCustomModifyBinglu,
		clientCustomValue{Type: 2, Int32: 0}, // sub=0 ADD_MAX_LEVEL
		clientCustomValue{Type: 6, Text: ""}, // item_id（未用）
		clientCustomValue{Type: 2, Int32: 1}, // binglu_type=槽位 1
		clientCustomValue{Type: 2, Int32: 0}, // add_num
	)
	if handled, err := handleModifyBingluCustom(m, message); !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	rows := player.bingluSnapshot()
	if rows[1].Max != 15 {
		t.Fatalf("slot1 max=%d, want 15", rows[1].Max)
	}
	if got := player.facultyValue(); got != int32(3_000_000-expand.useFaculty) {
		t.Fatalf("faculty=%d, want %d", got, 3_000_000-expand.useFaculty)
	}
	// 整刷帧：12 行 + MaxBingLuLevel（零进度闸不适用——已有进度）。
	if frames := conn.Frames(); len(frames) < 13 {
		t.Fatalf("frames=%d, want >=13", len(frames))
	}

	// 修为不足：拒绝不动账。
	conn.resetFrames()
	player.restoreBinglu(bingluDefaults())
	setBingluFaculty(player, 100)
	if handled, _ := handleModifyBingluCustom(m, message); !handled {
		t.Fatal("handled=false")
	}
	if rows := player.bingluSnapshot(); rows[1].Max != 10 {
		t.Fatalf("unpaid expand changed max: %+v", rows[1])
	}
	if got := player.facultyValue(); got != 100 {
		t.Fatalf("unpaid faculty=%d", got)
	}
}

func TestGetBingluInfo(t *testing.T) {
	m, conn, player := newBingluTestSession(t)
	player.restoreBinglu(bingluDefaults())
	player.replaceBinglu(func() []bingluRow {
		rows := bingluDefaults()
		rows[0].Cur = 5
		return rows
	}())

	// 查自己：custom_show_binglu_info [name, cur,fill,max ×12]。
	self := bingluCustomMessage(clientCustomGetBinglu, clientCustomValue{Type: 6, Text: "tester"})
	if handled, err := handleGetBingluCustom(m, self); !handled || err != nil {
		t.Fatalf("self: handled=%v err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 1 || !strings.Contains(string(frames[0]), "custom_show_binglu_info") {
		t.Fatalf("frames=%d", len(frames))
	}
	if !strings.Contains(string(frames[0]), "tester") {
		t.Fatal("payload missing name")
	}

	// 查他人：-1 回执（单机无跨会话目录）。
	conn.resetFrames()
	other := bingluCustomMessage(clientCustomGetBinglu, clientCustomValue{Type: 6, Text: "stranger"})
	if handled, err := handleGetBingluCustom(m, other); !handled || err != nil {
		t.Fatalf("other: %v", err)
	}
	if frames := conn.Frames(); len(frames) != 1 || !strings.Contains(string(frames[0]), "custom_show_binglu_info") {
		t.Fatalf("miss frames=%d", len(frames))
	}
}

func TestBingluSnapshotJSONDialect(t *testing.T) {
	rows := bingluDefaults()
	rows[2] = bingluRow{Cur: 9, Fill: 999, Max: 15}
	raw, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded []bingluRow
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded[2].Cur != 9 || decoded[2].Fill != 999 || decoded[2].Max != 15 {
		t.Fatalf("decoded=%+v", decoded[2])
	}
}
