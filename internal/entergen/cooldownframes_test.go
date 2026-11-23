package entergen

// 冷却表记录帧 golden：女号官服抓包（初始账号登陆-完整引导-女.pcapng，
// 8032 线 t=197.729 普攻受理簇）逐字节原样。玩家 obj=0x350000F3 own=0x102A，
// 记录 109 cooldown_rec schema [int32,int64,int64,u16]。

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

const (
	goldenPlayerObject uint32 = 0x350000F3
	goldenPlayerOwner  uint32 = 0x0000102A
	goldenCooldownRec  uint16 = 109
)

// hexBytes 空白分隔的十六进制串 → 字节。
func hexBytes(t *testing.T, s string) []byte {
	t.Helper()
	raw, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return raw
}

// TestRecordInsertRowFrameGolden 0x11 插入空行：抓包 #2737/#2742（行 0/行 1）。
// 16B 头 + 22 个零（int32 0 / int64 0 / int64 0 / u16 0），共 38 字节。
func TestRecordInsertRowFrameGolden(t *testing.T) {
	zeros := strings.Repeat(" 00", 22)
	wantRow0 := "11 00 f3 00 00 35 2a 10 00 00 6d 00 00 00 01 00" + zeros
	wantRow1 := "11 00 f3 00 00 35 2a 10 00 00 6d 00 01 00 01 00" + zeros
	for row, want := range map[uint16]string{0: wantRow0, 1: wantRow1} {
		got, err := RecordInsertRowFrame(goldenPlayerObject, goldenPlayerOwner, goldenCooldownRec, row)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 38 || !bytes.Equal(got, hexBytes(t, want)) {
			t.Errorf("row %d:\n got % x\nwant %s", row, got, want)
		}
	}
}

// TestRecordGridFrameGolden 0x13 逐格写：抓包 #2738..#2741（行 0 四列，每帧一格）
// 与 #2743（行 1 类别 1——注意不是 256：格数/行号各 u16 之后才是列号与值）。
func TestRecordGridFrameGolden(t *testing.T) {
	start := int64(0x0C3934E4DD97) // FILETIME 纪元毫秒
	cases := []struct {
		row  uint16
		cell RecordGridCell
		want string
	}{
		{0, GridInt(0, 2012), "13 00 f3 00 00 35 2a 10 00 00 6d 00 01 00 00 00 00 dc 07 00 00"},
		{0, GridInt64(1, start), "13 00 f3 00 00 35 2a 10 00 00 6d 00 01 00 00 00 01 97 dd e4 34 39 0c 00 00"},
		{0, GridInt64(2, start+1200), "13 00 f3 00 00 35 2a 10 00 00 6d 00 01 00 00 00 02 47 e2 e4 34 39 0c 00 00"},
		{0, GridWord(3, 0xFFFF), "13 00 f3 00 00 35 2a 10 00 00 6d 00 01 00 00 00 03 ff ff"},
		{1, GridInt(0, 1), "13 00 f3 00 00 35 2a 10 00 00 6d 00 01 00 01 00 00 01 00 00 00"},
	}
	for _, c := range cases {
		got, err := RecordGridFrame(goldenPlayerObject, goldenPlayerOwner, goldenCooldownRec, c.row, c.cell)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, hexBytes(t, c.want)) {
			t.Errorf("row %d col %d:\n got % x\nwant %s", c.row, c.cell.Column, got, c.want)
		}
	}
}

// 多格连写：两格同帧 = 两个 [列 u8][值] 顺序拼接；空格集/列越界报错。
func TestRecordGridFrameBatchAndErrors(t *testing.T) {
	got, err := RecordGridFrame(goldenPlayerObject, goldenPlayerOwner, goldenCooldownRec, 0,
		GridInt(0, 50000), GridInt64(1, 7))
	if err != nil {
		t.Fatal(err)
	}
	want := "13 00 f3 00 00 35 2a 10 00 00 6d 00 02 00 00 00 00 50 c3 00 00 01 07 00 00 00 00 00 00 00"
	if !bytes.Equal(got, hexBytes(t, want)) {
		t.Errorf("\n got % x\nwant %s", got, want)
	}
	if _, err := RecordGridFrame(1, 1, goldenCooldownRec, 0); err == nil {
		t.Error("空格集应报错")
	}
	if _, err := RecordGridFrame(1, 1, goldenCooldownRec, 0, GridInt(4, 1)); err == nil {
		t.Error("列越界应报错")
	}
	if _, err := RecordGridFrame(1, 1, 5000, 0, GridInt(0, 1)); err == nil {
		t.Error("记录越界应报错")
	}
}
