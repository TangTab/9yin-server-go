package fssoloworld

// 重登序方言单测（2026-11-22 晚五轮终案，对齐 2433 fork 官服 capture
// 61036）：进场簇 `09→0a→0b→0d→…`——零 0x0C/0x0E/0x32/0x14（fork 客户端
// 重登自清，无需服务端清场；0x32/0x14 是 aow 遗物，fork 客户端错误消费
// 致 Out range 越界与状态累积）。memberLeaveGame 离场零额外帧。

import (
	"encoding/binary"
	"testing"
)

// TestStoreLoadEnterDialectClean 重登进场序：无清场帧、无 aow 遗物，
// 表→进场簇正常推进至 PlOnline。
func TestStoreLoadEnterDialectClean(t *testing.T) {
	s, pl := newCustomTestServer(t)
	frames := captureFrames(t)

	// 0x02 load 回执 {client=7, serial=1, ok=1}（RST1 缺席保留预置 roleState）。
	p := make([]byte, 12)
	binary.LittleEndian.PutUint32(p[0:], 7)
	binary.LittleEndian.PutUint32(p[4:], 1)
	binary.LittleEndian.PutUint32(p[8:], 1)
	if !s.storeLoadEnter(0, p) {
		t.Fatal("storeLoadEnter rejected")
	}
	if pl.State != PlOnline {
		t.Fatalf("enter did not complete synchronously: state=%v", pl.State)
	}
	banned := map[byte]string{0x0C: "scene clear", 0x32: "aow domain block", 0x14: "aow record mount"}
	for i, f := range *frames {
		if reason, bad := banned[f[0]]; bad {
			t.Fatalf("frame #%d is banned %#x (%s) in fork relogin dialect", i, f[0], reason)
		}
	}
	ops := opOf(*frames)
	if len(ops) == 0 || ops[0] != 0x09 {
		t.Fatalf("first frame = % x, want 0x09 property table", ops)
	}
	has0B := false
	for _, op := range ops {
		if op == 0x0B {
			has0B = true
		}
	}
	if !has0B {
		t.Fatalf("no 0x0B entry frame (ops=% x)", ops)
	}
}

// TestMemberLeaveGameNoExtraFrames 离场零额外帧（对齐 2433）。
func TestMemberLeaveGameNoExtraFrames(t *testing.T) {
	s, pl := newCustomTestServer(t)
	frames := captureFrames(t)

	p := make([]byte, 8)
	binary.LittleEndian.PutUint32(p[0:], pl.Conn1)
	binary.LittleEndian.PutUint32(p[4:], pl.Conn2)
	if !s.memberLeaveGame(0, p) {
		t.Fatal("memberLeaveGame rejected")
	}
	if s.players.FindByConnect(pl.Conn1, pl.Conn2) != nil {
		t.Fatal("player not removed")
	}
	if n := len(*frames); n != 0 {
		t.Fatalf("leave sent %d frames, want 0 (ops=% x)", n, opOf(*frames))
	}
}
