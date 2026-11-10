package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
	"time"
)

// 授予/续期/快照往返：到期时刻随快照持久，过期条目装载即丢弃。
func TestTimedBuffGrantSnapshotRestore(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	player := newPlayerActor("限时侠", 0)
	player.grantTimedBuff("buf_baosd_02", 3, 0, 24*time.Hour, now)
	player.grantTimedBuff("buf_vip_week", 1, 0, 2*time.Hour, now)

	player.mu.Lock()
	snapshot := player.timedBuffSnapshotLocked()
	player.mu.Unlock()
	if len(snapshot) != 2 {
		t.Fatalf("snapshot entries=%d want=2", len(snapshot))
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []timedBuffEntry
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}

	// 24h 后重放：只剩未过期……此场景两条都过期 → 0；分两次验证。
	if live := player.restoreTimedBuffs(decoded, now.Add(25*time.Hour)); live != 0 {
		t.Fatalf("all expired restore live=%d want=0", live)
	}
	if live := player.restoreTimedBuffs(decoded, now.Add(3*time.Hour)); live != 1 {
		t.Fatalf("partial restore live=%d want=1 (vip_week 剩)", live)
	}
}

// 授予帧形态：m169 九值，v7=i64 到期 unix 毫秒。
func TestTimedBuffGrantFrameCarriesExpiry(t *testing.T) {
	expires := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	frame, err := timedBuffGrantFrame(timedBuffRuntime{
		configID: "buf_baosd_02", level: 3, category: 0, expiresUTC: expires,
	}, "限时侠")
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x1e {
		t.Fatalf("opcode=%02X", frame[0])
	}
	if id := binary.LittleEndian.Uint32(frame[4:]); id != 169 {
		t.Fatalf("msg id=%d want=169", id)
	}
	if !bytes.Contains(frame, []byte("buf_baosd_02")) {
		t.Fatalf("missing config: % X", frame)
	}
	want := expires.UnixMilli()
	if !bytes.Contains(frame, func() []byte {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(want))
		return b
	}()) {
		t.Fatalf("missing expiry ms %d: % X", want, frame)
	}
}

// 到期结算：跨越产摘帧（addRemove=0、v7=原到期毫秒）且只发一次。
func TestTimedBuffExpiryFrames(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	player := newPlayerActor("到期侠", 0)
	player.grantTimedBuff("buf_test_1h", 1, 0, time.Hour, now)

	if frames := player.timedBuffExpiryFrames(now.Add(10*time.Minute)); len(frames) != 0 {
		t.Fatalf("premature removal frames=%d", len(frames))
	}
	frames := player.timedBuffExpiryFrames(now.Add(2 * time.Hour))
	if len(frames) != 1 {
		t.Fatalf("expiry frames=%d want=1", len(frames))
	}
	// addRemove=0（时间形态 0=摘）：帧内首个值 tag2 = 0
	if got := binary.LittleEndian.Uint32(frameArgOffset(frames[0])); got != 0 {
		t.Fatalf("addRemove=%d want=0", got)
	}
	if again := player.timedBuffExpiryFrames(now.Add(3 * time.Hour)); len(again) != 0 {
		t.Fatalf("duplicate removal frames=%d", len(again))
	}
	if count := player.timedBuffActiveCount(now.Add(3 * time.Hour)); count != 0 {
		t.Fatalf("active after expiry=%d", count)
	}
}

// frameArgOffset 返回 [msgid 后第一个 i32 值]——布局：op|count u16|tag2 id u32|tag2 v。
func frameArgOffset(frame []byte) []byte {
	return frame[4+4+1 : 4+4+1+4]
}

// BufferListStr 追加限时段：槽 200+、尾段到期毫秒。
func TestBufferListStringIncludesTimedBuffs(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	player := newPlayerActor("列表侠", 0)
	player.grantTimedBuff("buf_list_check", 2, 0, 5*time.Hour, now)
	got := player.bufferListString(now)
	if !bytes.Contains([]byte(got), []byte("buf_list_check,local-200,2,")) {
		t.Fatalf("timed segment missing: %q", got)
	}
	// 到期毫秒在条目内
	expiryMs := now.Add(5 * time.Hour).UnixMilli()
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(expiryMs))
	// BufferListStr 是文本属性：毫秒以十进制出现在串里
	if !bytes.Contains([]byte(got), []byte(formatInt64(expiryMs))) {
		t.Fatalf("expiry ms %d missing: %q", expiryMs, got)
	}
	// 过期后不再出现
	if bytes.Contains([]byte(player.bufferListString(now.Add(6*time.Hour))), []byte("buf_list_check")) {
		t.Fatalf("expired buff still listed")
	}
}

func formatInt64(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [24]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
