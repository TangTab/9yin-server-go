package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 坐骑域（NewRideRec 310 [3,3,6,1]）：行帧/清行/骑乘态帧 + 三表目录
//（tool_item script=Mount + itemartstatic MaleModel + new_ride MountType 兜底）。

func TestPushMountFramesClearsReaddsAndSendsRideState(t *testing.T) {
	conn := &captureMessageConnection{}
	mounts := []mountEntry{
		{ConfigID: "MOUNT_HAWK_003", PackID: 41523, ValidTime: 30},
		{ConfigID: "HAIWAI_MOUNT_6_009", PackID: 9513, ValidTime: 30, Selected: true},
	}
	rows, err := pushMountFrames(conn, mounts, &mountCatalog{}, 0)
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if rows != 2 {
		t.Fatalf("rows=%d, want 2", rows)
	}
	frames := conn.Frames()
	// 2 拥有行 + 1 骑乘态帧。
	if len(frames) != 3 {
		t.Fatalf("frames=%d, want 3", len(frames))
	}
	for _, frame := range frames[:2] {
		if frame[0] != 0x11 || binary.LittleEndian.Uint16(frame[10:]) != 310 {
			t.Fatalf("row=%x (want 0x11 idx 310)", frame)
		}
	}
	rideState := frames[2]
	if rideState[0] != 0x10 {
		t.Fatalf("ride state=%#x (want 0x10)", rideState[0])
	}

	// 整刷：oldRows=2 → 先 0x12 倒序清 2 行再重推。
	conn2 := &captureMessageConnection{}
	rows, err = pushMountFrames(conn2, mounts, &mountCatalog{}, 2)
	if err != nil {
		t.Fatalf("repush: %v", err)
	}
	if rows != 2 {
		t.Fatalf("rows=%d after repush", rows)
	}
	frames = conn2.Frames()
	if len(frames) != 5 {
		t.Fatalf("frames=%d, want 2 clear + 2 add + 1 ride = 5", len(frames))
	}
	for row, frame := range frames[:2] {
		if frame[0] != 0x12 || binary.LittleEndian.Uint16(frame[12:]) != uint16(1-row) {
			t.Fatalf("clear frame[%d]=%x (want row %d)", row, frame, 1-row)
		}
	}
	// 空存档零帧（含清场后归零）。
	conn3 := &captureMessageConnection{}
	if rows, err = pushMountFrames(conn3, nil, &mountCatalog{}, 0); err != nil || rows != 0 {
		t.Fatalf("empty push rows=%d err=%v", rows, err)
	}
	if frames := conn3.Frames(); len(frames) != 0 {
		t.Fatalf("frames=%d, want 0 (无存档零帧)", len(frames))
	}
}

func TestMountRideStateFrameWalkStyle(t *testing.T) {
	frame, err := mountRideStateFrame(41523, 2)
	if err != nil {
		t.Fatalf("ride state: %v", err)
	}
	if frame[0] != 0x10 {
		t.Fatalf("opcode=%#x", frame[0])
	}
	// 重登步行口径：Mount@77 空串，NewRideID@1333=包号，CurRiderType@82。
	raw := string(frame)
	if !containsPropertyIndex(raw, 77) || !containsPropertyIndex(raw, 1333) || !containsPropertyIndex(raw, 82) {
		t.Fatalf("ride state missing property indices: %x", frame)
	}
}

func containsPropertyIndex(frame string, index uint16) bool {
	needle := []byte{byte(index), byte(index >> 8)}
	return strings.Contains(frame, string(needle))
}

func stubMountCatalog(t *testing.T) *mountCatalog {
	t.Helper()
	dir := t.TempDir()
	toolItem := "[MOUNT_A]\nScript=Mount\nMountType=0\nArtPack=100\n\n" +
		"[MOUNT_B]\nScript=Mount\nMountType=7\nArtPack=200\n\n" +
		"[NOT_MOUNT]\nScript=Other\nArtPack=300\n\n" +
		"[MOUNT_NOMODEL]\nScript=Mount\nMountType=3\nArtPack=400\n"
	itemArtStatic := "[100]\nMaleModel=npc\\ride_a\n\n[200]\nMaleModel=npc\\ride_b\n\n[400]\nFemaleModel=npc\\ride_c\n"
	newRide := "[100]\nMountType=5\n\n[400]\nMountType=9\n"
	for name, content := range map[string]string{"tool_item.ini": toolItem, "itemartstatic.ini": itemArtStatic, "new_ride.ini": newRide} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return loadMountCatalog(
		filepath.Join(dir, "tool_item.ini"),
		filepath.Join(dir, "itemartstatic.ini"),
		filepath.Join(dir, "new_ride.ini"))
}

func TestLoadMountCatalogThreeTableMerge(t *testing.T) {
	catalog := stubMountCatalog(t)
	// MOUNT_A：tool_item MountType=0 → new_ride 兜底 5。
	if got := catalog.riderTypeOf(100); got != 5 {
		t.Fatalf("riderType(100)=%d, want new_ride fallback 5", got)
	}
	// MOUNT_B：tool_item 主源 7。
	if got := catalog.riderTypeOf(200); got != 7 {
		t.Fatalf("riderType(200)=%d, want 7", got)
	}
	// NOT_MOUNT（script≠Mount）与 MOUNT_NOMODEL（itemartstatic 只有
	// FemaleModel，无 MaleModel → 不可骑）均不入表。
	if got := catalog.riderTypeOf(300); got != 0 {
		t.Fatalf("riderType(300)=%d, want 0 (非 Mount script)", got)
	}
	if got := catalog.riderTypeOf(400); got != 0 {
		t.Fatalf("riderType(400)=%d, want 0 (无 MaleModel 不可骑)", got)
	}
	if got := catalog.riderTypeOf(9999); got != 0 {
		t.Fatalf("riderType(9999)=%d, want 0", got)
	}
}

func TestMountCurrentPack(t *testing.T) {
	mounts := []mountEntry{{PackID: 1}, {PackID: 2, Selected: true}}
	if got := mountCurrentPack(mounts); got != 2 {
		t.Fatalf("current=%d, want 2", got)
	}
	if got := mountCurrentPack([]mountEntry{{PackID: 1}}); got != 0 {
		t.Fatalf("current=%d, want 0", got)
	}
}

func TestMountSnapshotJSONDialect(t *testing.T) {
	// role_mounts.snapshot 实际形态（role 618 dump）：裸 JSON 数组，
	// 字段即 NewRideRec 列镜像。
	raw := `[{"ConfigID": "MOUNT_HAWK_003", "Selected": true, "StaticID": 41523, "UsedDate": 0, "ValidTime": 30},` +
		`{"ConfigID": "HAIWAI_MOUNT_6_009", "Selected": false, "StaticID": 9513, "UsedDate": 0, "ValidTime": 30}]`
	mounts, err := unmarshalMountSnapshot(raw)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(mounts) != 2 || mounts[0].PackID != 41523 || !mounts[0].Selected || mounts[1].ValidTime != 30 {
		t.Fatalf("mounts=%+v", mounts)
	}
}
