package entergen

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

func testContext(t *testing.T) Context {
	t.Helper()
	state, err := rolestate.BuildRoleState(rolestate.DefaultBirthScene, 1, "icon\\players\\boy01_create.png", 0)
	if err != nil {
		t.Fatalf("build role state: %v", err)
	}
	return Context{
		Identity: Identity{ObjectID: 1, OwnerID: 1, Sequence: 0x2F05},
		Session:  Session{ClientIP: "1.2.3.4", ServerID: 7100037, ServerName: "侠骨丹心", Account: "lizy86", AccountID: 415016589},
		RoleName: "徐傠囃辭睼",
		State:    state,
	}
}

// TestEntrySceneFrameShape 0x0B 官方 4 属性形：idx178/1468=230/1447/1448，
// seq 注入（官方非常量实证）。
func TestEntrySceneFrameShape(t *testing.T) {
	frame, err := EntrySceneFrame(Identity{Sequence: 0x2F05, ObjectID: 1, OwnerID: 1},
		`ini\scene\born03_YanYuZhuang`, "YanYuZhuang")
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x0B || binary.LittleEndian.Uint32(frame[1:]) != 0x2F05 {
		t.Fatalf("head: %x", frame[:8])
	}
	if binary.LittleEndian.Uint32(frame[5:]) != 1 || binary.LittleEndian.Uint32(frame[9:]) != 1 {
		t.Fatalf("obj/own: %x", frame[5:13])
	}
	if binary.LittleEndian.Uint16(frame[13:]) != 4 {
		t.Fatalf("attrs = %d", binary.LittleEndian.Uint16(frame[13:]))
	}
	if !bytes.Contains(frame, []byte("ini\\scene\\born03_YanYuZhuang\x00")) ||
		!bytes.Contains(frame, []byte("YanYuZhuang\x00")) {
		t.Fatalf("scene strings missing: %x", frame)
	}
}

// TestRecordFrames 记录族：0x14 挂载 12B + 0x11 行（QingGongRec 单列串）。
func TestRecordFrames(t *testing.T) {
	rec := clientdata.MustRecordIndexByName("QingGongRec")
	mount := RecordMountFrame(1, 1, rec)
	if len(mount) != 12 || mount[0] != 0x14 || binary.LittleEndian.Uint16(mount[10:]) != rec {
		t.Fatalf("mount frame: %x", mount)
	}
	row, err := RecordAddCellsFrame(1, 1, rec, []RecordCell{RecordString("qinggong_1")})
	if err != nil {
		t.Fatal(err)
	}
	if row[0] != 0x11 || binary.LittleEndian.Uint16(row[10:]) != rec {
		t.Fatalf("row head: %x", row[:16])
	}
	if !bytes.Contains(row, []byte("qinggong_1\x00")) {
		t.Fatalf("row value: %x", row)
	}
}

// TestAvatarFrame 化身 0x46：头布局 + 字段数覆盖（41 基础 + 27 技能×17
// 列 + CurNeiGong）+ 女号 Photo 覆盖。
func TestAvatarFrame(t *testing.T) {
	ctx := testContext(t)
	frame, count := AvatarFrame(ctx, false)
	if frame[0] != 0x28 || frame[1] != 0x46 {
		t.Fatalf("head: %x", frame[:2])
	}
	bodyLen := binary.LittleEndian.Uint32(frame[6:])
	if int(bodyLen)+10 != len(frame) {
		t.Fatalf("body len %d vs frame %d", bodyLen, len(frame))
	}
	if binary.LittleEndian.Uint32(frame[10:]) != 1 || binary.LittleEndian.Uint32(frame[14:]) != 1 {
		t.Fatalf("obj/own: %x", frame[10:18])
	}
	if count != 41+27*17+1 {
		t.Fatalf("field count = %d (41 base + 27×17 rows + CurNeiGong)", count)
	}
	if !bytes.Contains(frame, []byte("ng_wd_001\x00")) {
		t.Fatal("CurNeiGong missing")
	}
	// literal-47 变体：头 + outlen 可读。
	f47, _ := AvatarFrame(ctx, true)
	if f47[0] != 0x28 || f47[1] != 0x47 || binary.LittleEndian.Uint32(f47[6:]) != bodyLen {
		t.Fatalf("47 head: %x", f47[:10])
	}
}

// TestSequenceShape 两段编排：PreReady 含 0x0B/化身/物理流；Ready 含
// 视图簇与 0x1F 尾帧。
func TestSequenceShape(t *testing.T) {
	ctx := testContext(t)
	now := time.Unix(1794415325, 0)
	pre, err := PreReadyFrames(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if pre[0][0] != 0x0B {
		t.Fatalf("pre head op = %#x", pre[0][0])
	}
	var hasAvatar bool
	physicsCount := 0
	for _, f := range pre {
		switch {
		case f[0] == 0x28 && f[1] == 0x46:
			hasAvatar = true
		case f[0] == 0x10 && len(f) >= 13:
			physicsCount++
		}
	}
	if !hasAvatar {
		t.Fatal("pre ready missing avatar 0x28/0x46")
	}
	if physicsCount < 36 {
		t.Fatalf("物理流 0x10 帧 = %d（应 ≥36）", physicsCount)
	}
	ready, err := ReadyFrames(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	last := ready[len(ready)-1]
	if last[0] != 0x1F || len(last) != 25 {
		t.Fatalf("ready tail = %#x len=%d（应 0x1F/25B 位置帧）", last[0], len(last))
	}
	// Task_Completed 位图行：49634 → row1551 → 1552 行 0x11。
	if len(pre) < 1552 {
		t.Fatalf("Task_Completed 位图应含 final-row 前 1552 行，pre 共 %d 帧", len(pre))
	}
}

// TestCustomMessage TVarList：m655 六值形态。
func TestCustomMessage(t *testing.T) {
	frame, err := CustomMessageFrame(MsgServerIdentity,
		CustomInt(7100037), CustomInt(7100037), CustomWideString("侠骨丹心"),
		CustomInt(7100037), CustomString("lizy86"), CustomInt64(415016589))
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x1E || binary.LittleEndian.Uint16(frame[1:]) != 7 {
		t.Fatalf("head: %x", frame[:3])
	}
}
