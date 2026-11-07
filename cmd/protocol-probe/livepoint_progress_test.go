package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
	"time"
)

// 历练层（LiveGroove/HoldFaculty/LiveSpeedPower + custom 235）行为测试，
// 逐例移植 Java LivepointProgressTest：官方转化公式（form_faculty.lua:87-111：
// 槽速 10/20/50/100/200 ×(1+power%)，非名俊 ×0.7，1 毫历练 = 1 修为）、
// 100 秒累积、槽位上限、旧档兼容、潜能赎回 235 GETINFO/BUY 帧序与快照
// JSON 往返（docs/livepoint-conversion.md）。

var livepointTestNow = time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)

// newLivepointProgress 独立进度：绕开 newPlayerProgress 的 time.Now 基准，
// 直接以固定 NOW 为历练结算基准。
func newLivepointProgress() playerProgress {
	progress := newPlayerProgress()
	progress.lastLivepoint = livepointTestNow
	return progress
}

func TestLivepointAccrueFillsFirstGrooveEachStep(t *testing.T) {
	progress := newLivepointProgress()
	if !progress.livepointTick(livepointTestNow.Add(100*time.Second), true) {
		t.Fatal("槽空首步必有变化")
	}
	if got, want := progress.liveGroove[0], int32(1000-10); got != want {
		t.Fatalf("槽1 = %d, want +1000 累积 -10 转化（VIP 无损）= %d", got, want)
	}
	if got := progress.liveGroove[4]; got != 0 {
		t.Fatalf("槽5 = %d, want 0", got)
	}
}

func TestLivepointNormalTierSplitsSevenToThree(t *testing.T) {
	progress := newLivepointProgress()
	progress.liveGroove[4] = 200_000 // 槽5：200 点历练
	facultyBefore, holdBefore := progress.faculty, progress.holdFaculty
	if !progress.livepointTick(livepointTestNow.Add(100*time.Second), false) {
		t.Fatal("普通档首步必有变化")
	}
	// 槽1 累积 +1000、转化 10 → 990；槽5 转化 200：gain 140 修为 + 60 潜能。
	if got, want := progress.liveGroove[4], int32(200_000-200); got != want {
		t.Fatalf("槽5 = %d, want %d", got, want)
	}
	if got, want := progress.faculty, facultyBefore+140+7; got != want {
		t.Fatalf("faculty = %d, want %d", got, want)
	}
	if got, want := progress.holdFaculty, holdBefore+60+3; got != want {
		t.Fatalf("holdFaculty = %d, want %d", got, want)
	}
}

func TestLivepointVipTierConvertsLossless(t *testing.T) {
	progress := newLivepointProgress()
	progress.liveGroove[4] = 200_000
	facultyBefore := progress.faculty
	progress.livepointTick(livepointTestNow.Add(100*time.Second), true)
	if got, want := progress.faculty, facultyBefore+200+10; got != want {
		t.Fatalf("faculty = %d, want 名俊无损：槽5 全额 200 + 槽1 10 = %d", got, want)
	}
	if got := progress.holdFaculty; got != 0 {
		t.Fatalf("holdFaculty = %d, want 名俊不产生潜能 0", got)
	}
}

func TestLivepointSpeedPowerDoublesGrooveSpeed(t *testing.T) {
	progress := newLivepointProgress()
	progress.setLiveSpeedPower(100) // 卦象加成：速度 ×2
	progress.liveGroove[4] = 200_000
	facultyBefore := progress.faculty
	progress.livepointTick(livepointTestNow.Add(100*time.Second), false)
	// 槽5 速度 200×200/100=400 → 普通档 gain 280 / 潜能 120。
	if got, want := progress.faculty, facultyBefore+280+14; got != want {
		t.Fatalf("faculty = %d, want %d", got, want)
	}
	if got, want := progress.holdFaculty, int32(120+6); got != want {
		t.Fatalf("holdFaculty = %d, want %d", got, want)
	}
}

func TestLivepointGrooveCapStopsAccrual(t *testing.T) {
	progress := newLivepointProgress()
	for i := 0; i < grooveCount; i++ {
		progress.liveGroove[i] = grooveCap
	}
	total := int64(grooveCount) * int64(grooveCap)
	progress.livepointTick(livepointTestNow.Add(100*time.Second), true)
	after := int64(0)
	for i := 0; i < grooveCount; i++ {
		if got, want := progress.liveGroove[i], grooveCap-grooveBaseSpeed[i]; got != want {
			t.Fatalf("槽%d = %d, want 全满时累积丢弃、转化照扣 = %d", i+1, got, want)
		}
		after += int64(progress.liveGroove[i])
	}
	if after >= total {
		t.Fatalf("全满转化后总量 %d 应小于 %d", after, total)
	}
}

func TestLivepointAddFillsSequentiallyAndOverflows(t *testing.T) {
	progress := newLivepointProgress()
	if got, want := progress.addLivepoint(5), int32(5_000); got != want {
		t.Fatalf("addLivepoint(5) = %d, want 5 点 → 5000 毫全入槽1 = %d", got, want)
	}
	if got := progress.liveGroove[0]; got != 5_000 {
		t.Fatalf("槽1 = %d, want 5000", got)
	}
	progress.liveGroove[0] = grooveCap - 500
	if got, want := progress.addLivepoint(2), int32(2_000); got != want {
		t.Fatalf("addLivepoint(2) = %d, want 槽1 差 500 补满，余 1500 进槽2 = %d", got, want)
	}
	if got := progress.liveGroove[0]; got != grooveCap {
		t.Fatalf("槽1 = %d, want 满 %d", got, grooveCap)
	}
	if got, want := progress.liveGroove[1], int32(1_500); got != want {
		t.Fatalf("槽2 = %d, want %d", got, want)
	}
}

func TestLivepointSubHundredSecondsDoesNotSettle(t *testing.T) {
	progress := newLivepointProgress()
	progress.liveGroove[4] = 200_000
	if progress.livepointTick(livepointTestNow.Add(99*time.Second), false) {
		t.Fatal("99 秒不结算：余数留在 lastLivepoint")
	}
	if got := progress.liveGroove[4]; got != 200_000 {
		t.Fatalf("槽5 = %d, want 200000", got)
	}
	progress.livepointTick(livepointTestNow.Add(299*time.Second), false)
	if got, want := progress.liveGroove[4], int32(200_000-2*200); got != want {
		t.Fatalf("槽5 = %d, want 两个整步各扣一次槽速（99 秒那次不计步）= %d", got, want)
	}
}

func TestLivepointLegacySnapshotFallsBackToDefaults(t *testing.T) {
	progress := newLivepointProgress()
	progress.faculty = 5000
	saved := progress.snapshot()
	// 旧档缺历练层：槽/潜能/加成归零，时间戳无效（零值哨兵）。
	saved.LiveGroove1, saved.LiveGroove2 = 0, 0
	saved.LiveGroove3, saved.LiveGroove4, saved.LiveGroove5 = 0, 0, 0
	saved.HoldFaculty, saved.LiveSpeedPower = 0, 0
	saved.LastLivepointUTC = time.Time{}
	progress.restore(saved, livepointTestNow)
	if got := progress.liveGroove; got != ([grooveCount]int32{}) {
		t.Fatalf("旧档历练槽 = %v, want 全零", got)
	}
	if progress.holdFaculty != 0 || progress.liveSpeedPower != 0 {
		t.Fatalf("旧档 holdFaculty=%d liveSpeedPower=%d, want 0/0", progress.holdFaculty, progress.liveSpeedPower)
	}
	if got, want := progress.faculty, int32(5000); got != want {
		t.Fatalf("旧档修为存量 = %d, want 原样保留 %d", got, want)
	}
}

// 离线补结算（docs/livepoint-conversion.md 边界节）：重登时对
// last_livepoint_utc 以来的整步照结（累积+转化）。
func TestLivepointRestoreSettlesOfflineSteps(t *testing.T) {
	progress := newLivepointProgress()
	progress.liveGroove[4] = 200_000
	saved := progress.snapshot()
	saved.LastLivepointUTC = livepointTestNow.Add(-200 * time.Second)
	restored := newLivepointProgress()
	restored.restore(saved, livepointTestNow)
	// 200 秒 = 2 步：槽5 每步转化 200（普通档 70/30）。
	if got, want := restored.liveGroove[4], int32(200_000-2*200); got != want {
		t.Fatalf("离线槽5 = %d, want %d", got, want)
	}
	if got, want := restored.holdFaculty, int32(2*(60+3)); got != want {
		t.Fatalf("离线潜能 = %d, want 两步各（槽5 30%%=60 + 槽1 30%%=3）= %d", got, want)
	}
	if !restored.lastLivepoint.Equal(livepointTestNow) {
		t.Fatalf("离线结算后基准 = %v, want %v", restored.lastLivepoint, livepointTestNow)
	}
}

func TestLivepointGetInfoRepliesPriceFrame(t *testing.T) {
	player := newPlayerActor("李逍遥", 0)
	conn := &captureMessageConnection{}
	handled, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: livepointInfoMsg}, {Type: 2, Int32: 1}, {Type: 2, Int32: 0},
	}})
	if err != nil || !handled {
		t.Fatalf("235/1 handled=%v err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 1 || frames[0][0] != 0x1E {
		t.Fatalf("235/1 reply=%x", frames)
	}
	// 1E 03 00 | 02 <235> | 02 <sub=1> | 02 <price=500>
	if got := leInt32At(frames[0], 4); got != 235 {
		t.Fatalf("message id = %d, want 235", got)
	}
	if got := leInt32At(frames[0], 9); got != 1 {
		t.Fatalf("sub = %d, want 1", got)
	}
	if got := leInt32At(frames[0], 14); got != 500 {
		t.Fatalf("price = %d, want 500", got)
	}
}

func TestLivepointBuyExchangesGoldForHoldFaculty(t *testing.T) {
	player := newPlayerActor("李逍遥", 0)
	player.setGold(100)
	player.setHoldFaculty(2_000)
	facultyBefore := player.progress.faculty
	conn := &captureMessageConnection{}
	handled, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: livepointInfoMsg}, {Type: 2, Int32: 2}, {Type: 2, Int32: 3},
	}})
	if err != nil || !handled {
		t.Fatalf("235/2 handled=%v err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 2 || frames[0][0] != 0x10 || frames[1][0] != 0x10 {
		t.Fatalf("235/2 reply=%x, want [CapitalType0 帧, 0x10 vital]", frames)
	}
	if got := player.goldSnapshot(); got != 97 {
		t.Fatalf("gold = %d, want 扣 3 黄金 = 97", got)
	}
	if got, want := player.progress.holdFaculty, int32(500); got != want {
		t.Fatalf("holdFaculty = %d, want 2000-3×500 = %d", got, want)
	}
	if got, want := player.progress.faculty, facultyBefore+1_500; got != want {
		t.Fatalf("faculty = %d, want %d", got, want)
	}
}

func TestLivepointBuyBeyondGoldOrHoldIsSilentlyIgnored(t *testing.T) {
	player := newPlayerActor("李逍遥", 0)
	player.setGold(2)
	player.setHoldFaculty(10_000)
	conn := &captureMessageConnection{}
	handled, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: livepointInfoMsg}, {Type: 2, Int32: 2}, {Type: 2, Int32: 0},
	}})
	if err != nil || !handled {
		t.Fatalf("235/2 count=0 handled=%v err=%v", handled, err)
	}
	if frames := conn.Frames(); len(frames) != 0 {
		t.Fatalf("count=0 静默，reply=%x", frames)
	}
	handled, err = handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: livepointInfoMsg}, {Type: 2, Int32: 2}, {Type: 2, Int32: 3},
	}})
	if err != nil || !handled {
		t.Fatalf("235/2 黄金不足 handled=%v err=%v", handled, err)
	}
	if frames := conn.Frames(); len(frames) != 0 {
		t.Fatalf("黄金不足静默，reply=%x", frames)
	}
	if player.goldSnapshot() != 2 || player.progress.holdFaculty != 10_000 {
		t.Fatalf("静默拒绝不得改状态：gold=%d hold=%d", player.goldSnapshot(), player.progress.holdFaculty)
	}
}

// 快照 JSON 往返：历练层 8 字段（live_groove_1..5 / hold_faculty /
// live_speed_power / last_livepoint_utc）与 Java FacultyStore 键名逐字一致。
func TestLivepointSnapshotRoundTrip(t *testing.T) {
	progress := newLivepointProgress()
	progress.addLivepoint(4)
	progress.setHoldFaculty(1_234)
	progress.setLiveSpeedPower(25)
	progress.livepointTick(livepointTestNow, false) // 不结算，仅固定基准
	saved := progress.snapshot()

	raw, err := json.Marshal(saved)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	var loaded facultyProgressSnapshot
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	for _, key := range []string{
		"live_groove_1", "live_groove_2", "live_groove_3", "live_groove_4", "live_groove_5",
		"hold_faculty", "live_speed_power", "last_livepoint_utc",
	} {
		if !jsonHasKey(raw, key) {
			t.Fatalf("snapshot json misses Java key %q: %s", key, raw)
		}
	}
	if loaded.LiveGroove1 != saved.LiveGroove1 || loaded.LiveGroove5 != saved.LiveGroove5 {
		t.Fatalf("live groove round trip: %v vs %v", loaded, saved)
	}
	if loaded.HoldFaculty != saved.HoldFaculty || loaded.LiveSpeedPower != saved.LiveSpeedPower {
		t.Fatalf("hold/power round trip: hold=%d/%d power=%d/%d",
			loaded.HoldFaculty, saved.HoldFaculty, loaded.LiveSpeedPower, saved.LiveSpeedPower)
	}
	if !loaded.LastLivepointUTC.Equal(saved.LastLivepointUTC) {
		t.Fatalf("last livepoint round trip: %v vs %v", loaded.LastLivepointUTC, saved.LastLivepointUTC)
	}
}

func leInt32At(frame []byte, offset int) int32 {
	return int32(binary.LittleEndian.Uint32(frame[offset:]))
}

func jsonHasKey(raw []byte, key string) bool {
	needle := []byte("\"" + key + "\":")
	return bytes.Contains(raw, needle)
}
