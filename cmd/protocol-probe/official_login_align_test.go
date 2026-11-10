package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
	"time"
)

// m294 官服形态应答（c.pcapng PAIR 实证）：
// [294, client_ms 回显(i64), obj(玩家), unix 秒(i64), uptime 秒(f64)]。
func TestOfficialTimeSyncReplyFrameShape(t *testing.T) {
	start := time.Unix(1000, 0)
	now := start.Add(90 * time.Second)
	frame, err := officialTimeSyncReplyFrame(72290612, start, now)
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x1e {
		t.Fatalf("opcode=%02X want 0x1E", frame[0])
	}
	if count := binary.LittleEndian.Uint16(frame[1:]); count != 5 {
		t.Fatalf("value count=%d want=5", count)
	}
	if id := binary.LittleEndian.Uint32(frame[4:]); id != 294 {
		t.Fatalf("msg id=%d want=294", id)
	}
	// 第二值：i64 client_ms 回显（tag3）
	if frame[8] != 3 {
		t.Fatalf("client_ms tag=%d want=3 (i64)", frame[8])
	}
	if echo := binary.LittleEndian.Uint64(frame[9:]); echo != 72290612 {
		t.Fatalf("client_ms echo=%d want=72290612", echo)
	}
	// 第三值：obj(玩家)（tag8 低32=对象ID 高32=ownerID）
	off := 17
	if frame[off] != 8 {
		t.Fatalf("obj tag=%d want=8", frame[off])
	}
	if obj := binary.LittleEndian.Uint32(frame[off+1:]); obj != playerObjectID {
		t.Fatalf("obj id=%#x want=%#x", obj, playerObjectID)
	}
	if owner := binary.LittleEndian.Uint32(frame[off+5:]); owner != playerOwnerID {
		t.Fatalf("obj owner=%#x want=%#x", owner, playerOwnerID)
	}
	// 第四值：unix 秒（tag3）
	off += 9
	if frame[off] != 3 {
		t.Fatalf("unix tag=%d want=3", frame[off])
	}
	if unix := binary.LittleEndian.Uint64(frame[off+1:]); unix != uint64(now.Unix()) {
		t.Fatalf("unix=%d want=%d", unix, now.Unix())
	}
	// 第五值：uptime 秒（tag5 f64，~90）
	off += 9
	if frame[off] != 5 {
		t.Fatalf("uptime tag=%d want=5 (f64)", frame[off])
	}
	uptime := math.Float64frombits(binary.LittleEndian.Uint64(frame[off+1:]))
	if uptime < 89.9 || uptime > 90.1 {
		t.Fatalf("uptime=%.3f want≈90", uptime)
	}
}

// 登录信息帧簇：m366/m623/m655 共三帧，msg id 与首值形态逐项校验。
func TestWriteOfficialLoginInfoFrames(t *testing.T) {
	conn := &captureMessageConnection{}
	writeOfficialLoginInfoFrames(conn, "1.2.3.4:5678", "lizy86", "徐测试", 42)
	frames := conn.Frames()
	if len(frames) != 3 {
		t.Fatalf("frames=%d want=3", len(frames))
	}
	wantIDs := []uint32{366, 623, 655}
	for i, want := range wantIDs {
		if frames[i][0] != 0x1e {
			t.Fatalf("frame%d opcode=%02X", i, frames[i][0])
		}
		if id := binary.LittleEndian.Uint32(frames[i][4:]); id != want {
			t.Fatalf("frame%d msg id=%d want=%d", i, id, want)
		}
	}
	if !bytes.Contains(frames[0], []byte("1.2.3.4")) {
		t.Fatalf("m366 missing client ip: % X", frames[0])
	}
	if !bytes.Contains(frames[2], []byte("lizy86")) || !bytes.Contains(frames[2], []byte("侠骨丹心")) {
		t.Fatalf("m655 missing server identity strings: % X", frames[2])
	}
	// m655 账号 ID = roleID（i64 尾值）
	if !bytes.Contains(frames[2], []byte{0x03, 42, 0, 0, 0, 0, 0, 0, 0}) {
		t.Fatalf("m655 account id tail missing: % X", frames[2])
	}
}

// 就绪信息帧簇：m607/m52/m586 共三帧。
func TestWriteOfficialReadyInfoFrames(t *testing.T) {
	conn := &captureMessageConnection{}
	now := time.Unix(1794116366, 0)
	writeOfficialReadyInfoFrames(conn, "test", "lizy86", "徐测试", now)
	frames := conn.Frames()
	if len(frames) != 3 {
		t.Fatalf("frames=%d want=3", len(frames))
	}
	for i, want := range []uint32{607, 52, 586} {
		if id := binary.LittleEndian.Uint32(frames[i][4:]); id != want {
			t.Fatalf("frame%d msg id=%d want=%d", i, id, want)
		}
	}
	// m52 unix=1794116366=0x6AF00B0E → i64 LE
	if !bytes.Contains(frames[1], []byte{0x03, 0x0E, 0x0B, 0xF0, 0x6A, 0x00, 0x00, 0x00, 0x00}) {
		t.Fatalf("m52 unix seconds payload missing: % X", frames[1])
	}
}

// 白名单已知集：924/327 新入列，未知 id 不在其中。
func TestActivityCustomMsgKnown(t *testing.T) {
	for _, id := range []int32{924, 327, 211, 431, 770} {
		if !activityCustomMsgKnown(id) {
			t.Fatalf("msg %d should be known", id)
		}
	}
	for _, id := range []int32{999, 82, 301, 876} {
		if activityCustomMsgKnown(id) {
			t.Fatalf("msg %d should NOT be known", id)
		}
	}
}

// m327 应答：以登录链同款 0x11 加行帧重放 Task_Accepted（record 29）。
// 官服方言为 0x13 写格 ×9（行已在客户端时的幂等覆写）；本服复用
// grantFunctionUnlock 已真机验证的加行形态，语义等价（对照文档 §十一）。
func TestRepushAcceptedTaskFrames(t *testing.T) {
	conn := &captureMessageConnection{}
	if err := repushAcceptedTaskFrames(conn, "test"); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) == 0 {
		t.Fatal("no frames written")
	}
	for _, f := range frames {
		if f[0] != 0x11 {
			t.Fatalf("frame opcode=%02X want 0x11 (record add row)", f[0])
		}
		if rec := binary.LittleEndian.Uint16(f[10:]); rec != recordTaskAccepted {
			t.Fatalf("record=%d want=%d", rec, recordTaskAccepted)
		}
	}
}

// 换装战斗属性增量：属性变化 → 恰好一个变化项的单属性 0x10；无变化 → 零帧。
func TestPushCombatAttributeDelta(t *testing.T) {
	player := playerWithBag()
	before := combatPanelSnapshot(player)
	if len(before) == 0 {
		t.Fatal("snapshot empty")
	}

	// 无变化：零帧。
	conn := &captureMessageConnection{}
	pushCombatAttributeDelta(conn, player, before, "test")
	if n := len(conn.Frames()); n != 0 {
		t.Fatalf("no-change delta frames=%d want=0", n)
	}

	// 五维变化：Str 及其级联项（MeleePowerAdd/PhyVaAdd/MaxHPAdd 等
	// 以 str 为源头的派生值，官服同理会全部重推）各自单属性帧。
	player.mu.Lock()
	player.progress.attributes.str += 7
	player.mu.Unlock()
	conn = &captureMessageConnection{}
	pushCombatAttributeDelta(conn, player, before, "test")
	frames := conn.Frames()
	if len(frames) == 0 {
		t.Fatal("changed attributes produced no delta frames")
	}
	strIndex := propStr // 0x10 帧只带索引（Str=485=0x01E5），常量转 byte 需经变量
	strIdx := []byte{byte(strIndex), byte(strIndex >> 8)}
	hasStr := false
	for _, f := range frames {
		if f[0] != 0x10 {
			t.Fatalf("delta frame opcode=%02X want 0x10", f[0])
		}
		if bytes.Contains(f, strIdx) {
			hasStr = true
		}
	}
	if !hasStr {
		t.Fatalf("delta frames missing Str index %d: % X", propStr, frames)
	}
}

// 装备随机属性折算：按 uid 汇总、跨件求和、跳过空 uid。
func TestEquipAttributeSums(t *testing.T) {
	equips := []equipRow{
		{slot: 1, uid: "u1"},
		{slot: 2, uid: "u2"},
		{slot: 3, uid: ""}, // starter 无随机属性
	}
	props := map[string]itemRandomProps{
		"u1": {PropRows: []randomPropLine{{PropID: 485, Value: 5}, {PropID: 581, Value: 100}}},
		"u2": {PropRows: []randomPropLine{{PropID: 485, Value: 7}, {PropID: 99999, Value: 1}}},
	}
	sums := equipAttributeSums(equips, props)
	if got := sums[485]; got != 12 {
		t.Fatalf("Str add sum=%d want=12", got)
	}
	if got := sums[581]; got != 100 {
		t.Fatalf("MaxHPAdd sum=%d want=100", got)
	}
	if _, ok := sums[uint16(65535)]; ok {
		t.Fatal("out-of-range prop id should be skipped")
	}
}

// combatPanelProperties 合并：int32 属性叠加 equipAdds，其余不动。
func TestCombatPanelPropertiesMergesEquipAdds(t *testing.T) {
	player := playerWithBag()
	base := player.combatPanelProperties()
	var baseStr int32
	for _, prop := range base {
		if prop.Index == propStr {
			baseStr = prop.Value.I32
		}
	}
	player.setEquipAttributeAdds(map[uint16]int32{propStr: 12})
	merged := player.combatPanelProperties()
	for _, prop := range merged {
		if prop.Index == propStr && prop.Value.I32 != baseStr+12 {
			t.Fatalf("Str merged=%d want=%d(+12)", prop.Value.I32, baseStr+12)
		}
	}
	// 换装增量联动：快照应包含加成后的帧字节（与登录全量同口径）。
	snapshot := combatPanelSnapshot(player)
	strIndex := propStr
	idxBytes := []byte{byte(strIndex), byte(strIndex >> 8)}
	found := false
	for _, frame := range snapshot {
		if bytes.Contains(frame, idxBytes) {
			found = true
		}
	}
	if !found {
		t.Fatal("snapshot missing Str after equip adds")
	}
}

// 阴阳太极分桶 + 武技点：属性在列；目录不可用时零值不 panic。
func TestPropertiesIncludeWuxingBucketsAndWuJi(t *testing.T) {
	player := playerWithBag()
	props := player.combatPanelProperties()
	want := map[uint16]bool{propYinLevel: false, propYangLevel: false, propTaijiLevel: false, propWuJiPoint: false, propMaxWuJiPoint: false}
	for _, prop := range props {
		if _, ok := want[prop.Index]; ok {
			want[prop.Index] = true
		}
	}
	for index, seen := range want {
		if !seen {
			t.Fatalf("property index %d missing from panel", index)
		}
	}
}

// 登录 buff：至少 buff_group_flag_101 一帧 m169 两值形态；装备内功存在时叠加其 buffID。
func TestWriteOfficialLoginBuffFrames(t *testing.T) {
	player := playerWithBag()
	conn := &captureMessageConnection{}
	writeOfficialLoginBuffFrames(conn, player, "test")
	frames := conn.Frames()
	if len(frames) < 1 {
		t.Fatal("no login buff frames")
	}
	for _, f := range frames {
		if f[0] != 0x1e {
			t.Fatalf("buff frame opcode=%02X want 0x1E", f[0])
		}
		if id := binary.LittleEndian.Uint32(f[4:]); id != 169 {
			t.Fatalf("buff msg id=%d want=169", id)
		}
	}
	if !bytes.Contains(frames[0], []byte("buff_group_flag_101")) {
		t.Fatalf("first buff frame missing group flag: % X", frames[0])
	}
}
