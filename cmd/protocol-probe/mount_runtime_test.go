package main

import (
	"strings"
	"testing"
)

// 坐骑运行域（menu2 mount.go 函数级对齐）：toggle/266 子令/技能闸/
// 拥有集纯函数。store=nil 纯内存口径（帧链照走，不落库）。

func newTestMountSession(t *testing.T) (*mountSession, *captureMessageConnection, *playerActor) {
	t.Helper()
	conn := &captureMessageConnection{}
	player := &playerActor{}
	session := &mountSession{
		link: conn, player: player, catalog: stubMountCatalog(t),
	}
	return session, conn, player
}

// rideCustomMessage 构造 243 帧（[msgId, flag]）。
func rideCustomMessage(flag int32) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomRide},
		{Type: 2, Int32: flag},
	}}
}

// newRideCustomMessage 构造 266 帧（[msgId, sub, arg...]）。
func newRideCustomMessage(sub int32, arg clientCustomValue) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomNewRide},
		{Type: 2, Int32: sub},
		arg,
	}}
}

func TestHandleRideCustomToggle(t *testing.T) {
	session, conn, player := newTestMountSession(t)

	// 无坐骑：拒绝，零帧。
	handled, err := handleRideCustom(session, rideCustomMessage(1))
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if frames := conn.Frames(); len(frames) != 0 {
		t.Fatalf("frames=%d, want 0", len(frames))
	}

	// 唯一拥有（未选中）→ soleOwnedMount 自动选中 + 上马（整推 + 属性组
	// + 239 + on_custom_new_ride）。
	player.restoreMounts([]mountEntry{{ConfigID: "MOUNT_B", PackID: 200}})
	if _, err = handleRideCustom(session, rideCustomMessage(1)); err != nil {
		t.Fatalf("ride: %v", err)
	}
	frames := conn.Frames()
	if len(frames) != 5 { // 整推（拥有行+骑乘态）+ 上马属性组 + 239 + on_custom_new_ride
		t.Fatalf("frames=%d, want 5", len(frames))
	}
	if !player.ridingMount() || player.ridingMountConfig() != "MOUNT_B" {
		t.Fatalf("riding=%v config=%q", player.riding, player.ridingConfig)
	}
	if pack := mountCurrentPack(player.mountsSnapshot()); pack != 200 {
		t.Fatalf("selected pack=%d, want 200 (soleOwned 自动选中)", pack)
	}
	if !strings.Contains(string(frames[2]), "ride_b") { // 上马属性组（0x10 模型串）
		t.Fatalf("mount model frame missing: %x", frames[2])
	}

	// 再按骑乘键 = 下马（属性组空串 + 242）。
	conn2Frames := len(frames)
	if _, err = handleRideCustom(session, rideCustomMessage(0)); err != nil {
		t.Fatalf("dismount: %v", err)
	}
	if player.ridingMount() {
		t.Fatalf("still riding after toggle")
	}
	tail := conn.Frames()[conn2Frames:]
	if len(tail) != 2 { // 骑乘态 0x10 + 242
		t.Fatalf("dismount frames=%d, want 2", len(tail))
	}
}

func TestHandleNewRideUseItemConsumesAndOwns(t *testing.T) {
	session, conn, player := newTestMountSession(t)
	def := session.catalog.defByConfig("MOUNT_A")

	// sub=1 无 pending：忽略。
	if _, err := handleNewRideCustom(session, newRideCustomMessage(1,
		clientCustomValue{Type: 6, Text: "MOUNT_A"})); err != nil {
		t.Fatalf("new ride: %v", err)
	}
	if frames := conn.Frames(); len(frames) != 0 {
		t.Fatalf("frames=%d, want 0 (无 pending 不动拥有)", len(frames))
	}

	// 挂起 + 背包无道具：mount item gone。
	player.mu.Lock()
	player.pendingMount = "MOUNT_A"
	player.mu.Unlock()
	if _, err := handleNewRideCustom(session, newRideCustomMessage(1,
		clientCustomValue{Type: 6, Text: "MOUNT_A"})); err != nil {
		t.Fatalf("new ride: %v", err)
	}
	if frames := conn.Frames(); len(frames) != 0 {
		t.Fatalf("frames=%d, want 0 (道具不在包)", len(frames))
	}
	if owns := ownsMountPack(player.mountsSnapshot(), def.packID); owns {
		t.Fatalf("owned without bag item")
	}

	// 挂起 + 道具在包（amount=1）：消耗（内存行删除 + 0x19 帧）+ 补拥有
	// + 整推；激活不自动骑（menu2 no auto-mount 口径）。
	player.mu.Lock()
	player.pendingMount = "MOUNT_A"
	player.bag = []bagRow{{viewID: 1, slot: 5, configID: "MOUNT_A", itemType: 200, amount: 1}}
	player.mu.Unlock()
	if _, err := handleNewRideCustom(session, newRideCustomMessage(1,
		clientCustomValue{Type: 6, Text: "MOUNT_A"})); err != nil {
		t.Fatalf("new ride: %v", err)
	}
	if owns := ownsMountPack(player.mountsSnapshot(), def.packID); !owns {
		t.Fatalf("not owned after use item")
	}
	if player.ridingMount() {
		t.Fatalf("sub=1 must not auto-mount")
	}
	player.mu.Lock()
	remaining := len(player.bag)
	player.mu.Unlock()
	if remaining != 0 {
		t.Fatalf("bag rows=%d, want 0 (consumed)", remaining)
	}
	// 0x19 remove + 拥有行 + 骑乘态帧。
	if frames := conn.Frames(); len(frames) < 3 {
		t.Fatalf("frames=%d, want >=3 (remove+own+state)", len(frames))
	}

	// 已拥有再 sub=1：不消耗（GM 口径）。
	player.mu.Lock()
	player.pendingMount = "MOUNT_A"
	player.bag = []bagRow{{viewID: 1, slot: 6, configID: "MOUNT_A", itemType: 200, amount: 1}}
	player.mu.Unlock()
	if _, err := handleNewRideCustom(session, newRideCustomMessage(1,
		clientCustomValue{Type: 6, Text: "MOUNT_A"})); err != nil {
		t.Fatalf("new ride: %v", err)
	}
	player.mu.Lock()
	remaining = len(player.bag)
	player.mu.Unlock()
	if remaining != 1 {
		t.Fatalf("bag rows=%d, want 1 (已拥有不消耗)", remaining)
	}
}

func TestHandleNewRideSelectSwitchesAndMounts(t *testing.T) {
	session, conn, player := newTestMountSession(t)
	player.restoreMounts([]mountEntry{
		{ConfigID: "MOUNT_A", PackID: 100},
		{ConfigID: "MOUNT_B", PackID: 200},
	})

	// sub=2 按包 id 选中：切换 Selected + 上马。
	if _, err := handleNewRideCustom(session, newRideCustomMessage(2,
		clientCustomValue{Type: 2, Int32: 200})); err != nil {
		t.Fatalf("select: %v", err)
	}
	if pack := mountCurrentPack(player.mountsSnapshot()); pack != 200 {
		t.Fatalf("selected=%d, want 200", pack)
	}
	if !player.ridingMount() || player.ridingMountConfig() != "MOUNT_B" {
		t.Fatalf("riding=%v config=%q", player.riding, player.ridingConfig)
	}

	// 骑乘中切换 sub=2 到另一匹：换骑（先下后上，尾部帧 = 新坐骑属性组）。
	if _, err := handleNewRideCustom(session, newRideCustomMessage(2,
		clientCustomValue{Type: 6, Text: "MOUNT_A"})); err != nil {
		t.Fatalf("switch: %v", err)
	}
	if pack := mountCurrentPack(player.mountsSnapshot()); pack != 100 {
		t.Fatalf("switched selected=%d, want 100", pack)
	}
	if player.ridingMountConfig() != "MOUNT_A" {
		t.Fatalf("riding config=%q, want MOUNT_A", player.ridingConfig)
	}

	// 未拥有 sub=2：拒绝（走 USE_ITEM 链入档）。
	conn.resetFrames()
	if _, err := handleNewRideCustom(session, newRideCustomMessage(2,
		clientCustomValue{Type: 2, Int32: 9999})); err != nil {
		t.Fatalf("unknown: %v", err)
	}
	if frames := conn.Frames(); len(frames) != 0 {
		t.Fatalf("frames=%d, want 0 (未知包 shape probe)", len(frames))
	}
}

func TestMountSkillGate(t *testing.T) {
	session, _, player := newTestMountSession(t)

	// 步行发骑乘技能：拒绝。
	if got := mountSkillGate(session, "riding_dismount"); got != mountSkillReject {
		t.Fatalf("walk + ride skill = %d, want reject", got)
	}
	// 步行发普通技能：放行。
	if got := mountSkillGate(session, "cs_normal_hit"); got != mountSkillPass {
		t.Fatalf("walk + normal skill = %d, want pass", got)
	}

	// 骑乘中发普通技能：拒绝。
	player.mu.Lock()
	player.riding = true
	player.mu.Unlock()
	if got := mountSkillGate(session, "cs_normal_hit"); got != mountSkillReject {
		t.Fatalf("riding + normal skill = %d, want reject", got)
	}
	// 骑乘中发 dismount 技能：下马。
	if got := mountSkillGate(session, "riding_dismount"); got != mountSkillDismount {
		t.Fatalf("riding + dismount = %d, want dismount", got)
	}
	if player.ridingMount() {
		t.Fatalf("still riding after dismount skill")
	}
	// 骑乘中发白名单非下马技能：放行。
	player.mu.Lock()
	player.riding = true
	player.mu.Unlock()
	if got := mountSkillGate(session, "riding_spur"); got != mountSkillPass {
		t.Fatalf("riding + whitelisted = %d, want pass", got)
	}
}

func TestMountEntrySetOps(t *testing.T) {
	mounts := []mountEntry{{ConfigID: "A", PackID: 100}, {ConfigID: "C", PackID: 300}}

	// withOwnedMount：升序插入。
	owned := withOwnedMount(mounts, 200, "B")
	if len(owned) != 3 || owned[1].PackID != 200 || owned[1].ConfigID != "B" {
		t.Fatalf("owned=%+v", owned)
	}
	// 已拥有幂等。
	if again := withOwnedMount(owned, 200, "B"); len(again) != 3 {
		t.Fatalf("idempotent own failed: %+v", again)
	}
	// withMountSelected：唯一选中。
	selected := withMountSelected(owned, 200)
	for index, entry := range selected {
		want := entry.PackID == 200
		if entry.Selected != want {
			t.Fatalf("selected[%d]=%v, want %v", index, entry.Selected, want)
		}
	}
	// 未拥有选中：原样。
	if same := withMountSelected(selected, 999); len(same) != 3 || !same[1].Selected {
		t.Fatalf("unknown select changed set: %+v", same)
	}
	// removeMountPack：剔除回滚。
	if removed := removeMountPack(selected, 200); len(removed) != 2 {
		t.Fatalf("removed=%+v", removed)
	}
	// ownsMountPack。
	if !ownsMountPack(selected, 100) || ownsMountPack(selected, 999) {
		t.Fatalf("ownsMountPack broken")
	}
}

func TestSplitAndMergeRideSkills(t *testing.T) {
	skills := splitRideSkills("riding_dismount, riding_spur;riding_jiben_camel")
	if len(skills) != 3 || skills[0] != "riding_dismount" {
		t.Fatalf("split=%v", skills)
	}
	merged := mergeRideSkills([]string{"a", "b"}, []string{"b", "c"})
	if len(merged) != 3 || merged[2] != "c" {
		t.Fatalf("merge=%v", merged)
	}
}
