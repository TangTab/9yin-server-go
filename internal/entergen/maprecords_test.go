package entergen

// 开图三件套测试：OpenMapRec 24 行位图帧（外服 len=112 形）、MapMaskRec
// 迷雾行、AutoTrace 单属性帧、PreReady/Ready 序位（b-line map_records.go
// 定版口径）。

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

// TestOpenMapRecFrameShape 外服 c.pcapng 同形：0x11 头 16B + 24 行 ×int32
// = 112B；rec=51、起始行 0、行数 24；行值逐位等于位图。
func TestOpenMapRecFrameShape(t *testing.T) {
	bits := officialOpenMapBits
	frame := openMapRecFrame(1, 1, bits)
	if len(frame) != 112 {
		t.Fatalf("len=%d, want 112（16B 头 + 24×4）", len(frame))
	}
	if frame[0] != 0x11 || frame[1] != 0 {
		t.Fatalf("op/sub = %#x %#x", frame[0], frame[1])
	}
	if got := binary.LittleEndian.Uint32(frame[2:]); got != 1 {
		t.Fatalf("obj=%d", got)
	}
	if got := binary.LittleEndian.Uint32(frame[6:]); got != 1 {
		t.Fatalf("own=%d", got)
	}
	if rec := clientdata.MustRecordIndexByName("OpenMapRec"); binary.LittleEndian.Uint16(frame[10:]) != rec {
		t.Fatalf("rec=%d, want %d（OpenMapRec）", binary.LittleEndian.Uint16(frame[10:]), rec)
	}
	if start, rows := binary.LittleEndian.Uint16(frame[12:]), binary.LittleEndian.Uint16(frame[14:]); start != 0 || rows != 24 {
		t.Fatalf("start/rows=%d/%d, want 0/24", start, rows)
	}
	for i, want := range bits {
		if got := binary.LittleEndian.Uint32(frame[16+4*i:]); got != want {
			t.Fatalf("row %d=%#x, want %#x", i, got, want)
		}
	}
}

// TestMapRecordsFramesOrderAndOverlay sceneIndex≥0：迷雾行先、位图后；
// 位图叠加当前场景位（born03=3 → 行 0 bit3）；-1 降级只发位图。
func TestMapRecordsFramesOrderAndOverlay(t *testing.T) {
	ctx := testContext(t)
	ctx.SceneIndex = 3 // born03_YanYuZhuang 全局段号（scenes.ini [3]）
	frames, err := MapRecordsFrames(ctx, ctx.SceneIndex)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("frames=%d, want 2（迷雾行+位图）", len(frames))
	}
	mask, bitmap := frames[0], frames[1]
	if mask[0] != 0x11 || binary.LittleEndian.Uint16(mask[10:]) != clientdata.MustRecordIndexByName("MapMaskRec") {
		t.Fatalf("迷雾帧头 rec=%d", binary.LittleEndian.Uint16(mask[10:]))
	}
	// 单行方言：行数=1；体 [int32 3][len u32 7]["born03\0"][int32 1]。
	if rows := binary.LittleEndian.Uint16(mask[14:]); rows != 1 {
		t.Fatalf("迷雾行数=%d, want 1", rows)
	}
	off := 16
	if got := binary.LittleEndian.Uint32(mask[off:]); got != 3 {
		t.Fatalf("迷雾 col0=%d, want 3（born03 段号）", got)
	}
	off += 4
	if n := binary.LittleEndian.Uint32(mask[off:]); n != 7 {
		t.Fatalf("迷雾 col1 len=%d, want 7（born03\\0）", n)
	}
	if !bytes.Equal(mask[off+4:off+4+7], append([]byte(ctx.State.Resource), 0)) {
		t.Fatalf("迷雾 col1=%q", mask[off+4:off+4+7])
	}
	if open := binary.LittleEndian.Uint32(mask[off+4+7:]); open != 1 {
		t.Fatalf("迷雾 col2=%d, want 1（已开）", open)
	}
	// 位图叠加位：行 0 bit3（基线 0x7a03fffe | 8 = 0x7a03fffe 已含？校验并集）。
	wantRow0 := officialOpenMapBits[0] | 1<<3
	if got := binary.LittleEndian.Uint32(bitmap[16:]); got != wantRow0 {
		t.Fatalf("位图行 0=%#x, want %#x（叠加 born03 位）", got, wantRow0)
	}

	degraded, err := MapRecordsFrames(ctx, -1)
	if err != nil {
		t.Fatal(err)
	}
	if len(degraded) != 1 || degraded[0][0] != 0x11 {
		t.Fatalf("降级 frames=%d（只发基线位图）", len(degraded))
	}
	if got := binary.LittleEndian.Uint32(degraded[0][16:]); got != officialOpenMapBits[0] {
		t.Fatalf("降级位图行 0=%#x（不叠加）", got)
	}
}

// TestAutoTraceFrame idx234=1 单属性 0x10。
func TestAutoTraceFrame(t *testing.T) {
	frame, err := AutoTraceFrame(testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x10 {
		t.Fatalf("op=%#x", frame[0])
	}
	entry, ok := clientdata.PlayerProperty("AutoTrace")
	if !ok {
		t.Fatal("AutoTrace 不在 2433 表")
	}
	if idx := binary.LittleEndian.Uint16(frame[12:]); idx != entry.Index {
		t.Fatalf("idx=%d, want %d", idx, entry.Index)
	}
	if v := binary.LittleEndian.Uint32(frame[14:]); v != 1 {
		t.Fatalf("AutoTrace=%d, want 1", v)
	}
}

// TestPreReadyReadyContainMapRecords 序位：PreReady 段尾（NPC 物化前）含
// 迷雾行+位图；Ready 段 m52 前含 AutoTrace。
func TestPreReadyReadyContainMapRecords(t *testing.T) {
	ctx := testContext(t)
	ctx.SceneIndex = 3
	pre, err := PreReadyFrames(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	openMap, mask := 0, 0
	for _, f := range pre {
		if f[0] != 0x11 {
			continue
		}
		switch binary.LittleEndian.Uint16(f[10:]) {
		case clientdata.MustRecordIndexByName("OpenMapRec"):
			openMap++
		case clientdata.MustRecordIndexByName("MapMaskRec"):
			mask++
		}
	}
	if openMap != 1 || mask != 1 {
		t.Fatalf("PreReady 开图帧 openMap=%d mask=%d, want 1/1", openMap, mask)
	}
	ready, err := ReadyFrames(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	entry, _ := clientdata.PlayerProperty("AutoTrace")
	autoTrace := 0
	for _, f := range ready {
		if f[0] == 0x10 && len(f) >= 16 && binary.LittleEndian.Uint16(f[12:]) == entry.Index {
			autoTrace++
		}
	}
	if autoTrace != 1 {
		t.Fatalf("Ready AutoTrace 帧=%d, want 1", autoTrace)
	}
}
