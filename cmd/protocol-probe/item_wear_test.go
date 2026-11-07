package main

import (
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// 背包四件套（msg30/34/36 + 31 穿戴分支）：Java LoginFlow :4559-4821 同源
// 语义。内存模型（store=nil 纯内存）为主；写穿持久化用 sqlmock 断言。

func newTestItemSession(conn sceneMessageConnection, player *playerActor, store *inventoryStore) *itemSession {
	return &itemSession{
		link: conn, player: player, store: store,
		ctx: context.Background(), roleID: role.RoleID(901), remote: "test",
		now: func() time.Time { return time.Unix(1700000000, 0).UTC() },
	}
}

func moveItemMsg(fromView, fromSlot, toView, toSlot int32) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomMoveItem},
		{Type: 2, Int32: fromView},
		{Type: 2, Int32: fromSlot},
		{Type: 2, Int32: toView},
		{Type: 2, Int32: toSlot},
	}}
}

func deleteItemMsg(view, slot, amount int32) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomDeleteItem},
		{Type: 2, Int32: view},
		{Type: 2, Int32: slot},
		{Type: 2, Int32: amount},
	}}
}

func arrangeMsg(view int32) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomArrangeBag},
		{Type: 2, Int32: view},
	}}
}

func playerWithBag(rows ...bagRow) *playerActor {
	player := newPlayerActor("背包侠", 0)
	player.restoreSessionInventory(0, nil, rows, nil)
	return player
}

// bagHas 非致命存在性检查（穿装消耗断言用）。
func bagHas(player *playerActor, configID string, category int32) bool {
	player.mu.Lock()
	defer player.mu.Unlock()
	for _, row := range player.bag {
		if row.configID == configID && row.viewID == category {
			return true
		}
	}
	return false
}

func bagSlotOf(t *testing.T, player *playerActor, configID string, category int32) int32 {
	t.Helper()
	player.mu.Lock()
	defer player.mu.Unlock()
	for _, row := range player.bag {
		if row.configID == configID && row.viewID == category {
			return row.slot
		}
	}
	t.Fatalf("bag row %s(cat %d) missing: %+v", configID, category, player.bag)
	return 0
}

func bagAmountOf(t *testing.T, player *playerActor, configID string, category int32) int32 {
	t.Helper()
	player.mu.Lock()
	defer player.mu.Unlock()
	for _, row := range player.bag {
		if row.configID == configID && row.viewID == category {
			return row.amount
		}
	}
	t.Fatalf("bag row %s(cat %d) missing", configID, category)
	return 0
}

func TestMoveItemBagToBagReSequencesRow(t *testing.T) {
	player := playerWithBag(
		bagRow{slot: 1, configID: "yaopin_00100", itemType: 2101, amount: 1, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	if handled, err := handleMoveItemCustom(h, moveItemMsg(2, 1, 123, 5)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := bagSlotOf(t, player, "yaopin_00100", 3); got != 5 {
		t.Fatalf("moved slot=%d, want 5 (category 3)", got)
	}
	// 帧序（Java 同序）：remove(源 2@1) → remove(源镜像 174@1) →
	// 入包成对帧 remove(123@5) → 0x18 行 → remove(镜像 179@5)。
	frames := conn.Frames()
	if len(frames) != 5 {
		t.Fatalf("move frames=%d: %x", len(frames), frames)
	}
	if got := binary.LittleEndian.Uint16(frames[0][1:]); got != 2 || binary.LittleEndian.Uint16(frames[0][3:]) != 1 {
		t.Fatalf("source remove=%x", frames[0])
	}
	if got := binary.LittleEndian.Uint16(frames[1][1:]); got != bagMirrorOf(2) || binary.LittleEndian.Uint16(frames[1][3:]) != 1 {
		t.Fatalf("source mirror remove=%x", frames[1])
	}
	if got := binary.LittleEndian.Uint16(frames[2][1:]); got != 123 || binary.LittleEndian.Uint16(frames[2][3:]) != 5 {
		t.Fatalf("target pre-clear=%x", frames[2])
	}
	if frames[3][0] != 0x18 || binary.LittleEndian.Uint16(frames[3][1:]) != 123 || binary.LittleEndian.Uint16(frames[3][3:]) != 5 {
		t.Fatalf("target add=%x", frames[3])
	}
	if got := binary.LittleEndian.Uint16(frames[4][1:]); got != bagMirrorOf(123) || binary.LittleEndian.Uint16(frames[4][3:]) != 5 {
		t.Fatalf("target mirror remove=%x", frames[4])
	}
}

func TestWearWeaponToDollSlot22AndUnequipBack(t *testing.T) {
	player := playerWithBag(
		bagRow{slot: 3, configID: "sword_test_001", itemType: 101, amount: 1, viewID: 2, equipType: "Weapon"},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	// 穿（装备页 121 → 自动槽 22）。
	if handled, err := handleMoveItemCustom(h, moveItemMsg(121, 3, 1, 0)); err != nil || !handled {
		t.Fatalf("wear handled=%t err=%v", handled, err)
	}
	worn, _, ok := player.equipAtSlot(22)
	if !ok || worn.configID != "sword_test_001" {
		t.Fatalf("worn=%+v ok=%t, want sword_test_001 @22", worn, ok)
	}
	if bagHas(player, "sword_test_001", 2) {
		t.Fatal("bag row not consumed on wear")
	}
	// 帧组：remove(121@3) → remove(176@3) → doll remove+add（pushWearFrames）→ 武器 0x10。
	frames := conn.Frames()
	dollAdd := false
	weaponFrame := false
	for _, frame := range frames {
		if frame[0] == 0x18 && binary.LittleEndian.Uint16(frame[1:]) == 1 &&
			binary.LittleEndian.Uint16(frame[3:]) == 22 {
			dollAdd = true
		}
		if frame[0] == 0x10 {
			weaponFrame = true
		}
	}
	if !dollAdd || !weaponFrame {
		t.Fatalf("wear frames=%x dollAdd=%v weapon=%v", frames, dollAdd, weaponFrame)
	}

	// 脱（doll 22 → 装备页自动空格）。
	base := len(conn.Frames())
	if handled, err := handleMoveItemCustom(h, moveItemMsg(1, 22, 121, 0)); err != nil || !handled {
		t.Fatalf("unequip handled=%t err=%v", handled, err)
	}
	if _, _, still := player.equipAtSlot(22); still {
		t.Fatal("unequip kept doll slot 22")
	}
	if got := bagSlotOf(t, player, "sword_test_001", 2); got == 0 {
		t.Fatal("unequip did not land weapon in bag")
	}
	unequipFrames := conn.Frames()[base:]
	// 帧组：doll remove@22 → 入包成对帧 → 脱后武器视觉 0x10（空手 1h）。
	if len(unequipFrames) < 4 || unequipFrames[0][0] != 0x19 ||
		binary.LittleEndian.Uint16(unequipFrames[0][1:]) != 1 || binary.LittleEndian.Uint16(unequipFrames[0][3:]) != 22 {
		t.Fatalf("unequip frames=%x", unequipFrames)
	}
}

func TestWearDualSlotEarringPicksFreeThenSwaps(t *testing.T) {
	player := playerWithBag(
		bagRow{slot: 1, configID: "earring_a", itemType: 103, amount: 1, viewID: 2, equipType: "Earring"},
		bagRow{slot: 2, configID: "earring_b", itemType: 103, amount: 1, viewID: 2, equipType: "Earring"},
		bagRow{slot: 3, configID: "earring_c", itemType: 103, amount: 1, viewID: 2, equipType: "Earring"},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	for _, slot := range []int32{1, 2} {
		if handled, err := handleMoveItemCustom(h, moveItemMsg(121, slot, 1, 0)); err != nil || !handled {
			t.Fatalf("wear %d handled=%t err=%v", slot, handled, err)
		}
	}
	if _, _, ok := player.equipAtSlot(11); !ok {
		t.Fatal("first earring should land slot 11")
	}
	if _, _, ok := player.equipAtSlot(12); !ok {
		t.Fatal("second earring should land slot 12")
	}
	// 第三只：双槽全占 → 换出第一槽（earring_a 回装备页背包）。
	if handled, err := handleMoveItemCustom(h, moveItemMsg(121, 3, 1, 0)); err != nil || !handled {
		t.Fatalf("swap wear handled=%t err=%v", handled, err)
	}
	worn, _, ok := player.equipAtSlot(11)
	if !ok || worn.configID != "earring_c" {
		t.Fatalf("slot 11 worn=%+v, want earring_c", worn)
	}
	if got := bagSlotOf(t, player, "earring_a", 2); got == 0 {
		t.Fatal("swapped-out earring_a not landed in bag")
	}
}

func TestDeleteItemConsumesStackAndRemovesWhenEmpty(t *testing.T) {
	player := playerWithBag(
		bagRow{slot: 4, configID: "yaopin_00300", itemType: 2102, amount: 5, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	// 扣 2：剩余 3 → remove+add+remove 三帧。
	if handled, err := handleDeleteItemCustom(h, deleteItemMsg(2, 4, 2)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := bagAmountOf(t, player, "yaopin_00300", 1); got != 3 {
		t.Fatalf("amount=%d, want 3", got)
	}
	if frames := conn.Frames(); len(frames) != 3 || frames[0][0] != 0x19 || frames[1][0] != 0x18 || frames[2][0] != 0x19 {
		t.Fatalf("partial delete frames=%x", frames)
	}

	// 扣 3：清行 → 双 remove（页 + 镜像）。
	base := len(conn.Frames())
	if handled, err := handleDeleteItemCustom(h, deleteItemMsg(2, 4, 3)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if bagCount := len(player.sessionBag()); bagCount != 0 {
		t.Fatalf("row not removed: %d rows left", bagCount)
	}
	if frames := conn.Frames()[base:]; len(frames) != 2 || frames[0][0] != 0x19 || frames[1][0] != 0x19 {
		t.Fatalf("full delete frames=%x", frames)
	}

	// 超量拒绝：空格。
	if handled, err := handleDeleteItemCustom(h, deleteItemMsg(2, 4, 1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()); got != base+2 {
		t.Fatalf("empty delete appended frames: %d", got)
	}
}

func TestArrangeGroupsByConfigAndCompacts(t *testing.T) {
	player := playerWithBag(
		bagRow{slot: 1, configID: "item_c", itemType: 1, amount: 1, viewID: 1},
		bagRow{slot: 3, configID: "item_a", itemType: 1, amount: 1, viewID: 1},
		bagRow{slot: 7, configID: "item_a", itemType: 1, amount: 1, viewID: 1},
		bagRow{slot: 9, configID: "item_b", itemType: 1, amount: 1, viewID: 1},
		// 别页行不受影响。
		bagRow{slot: 3, configID: "item_x", itemType: 1, amount: 1, viewID: 2},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	if handled, err := handleArrangeItemCustom(h, arrangeMsg(2)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	// 组序=首现（C@1, A@3, B@9）→ C@1, A@2, A@3, B@4；三格移动。
	if got := bagSlotOf(t, player, "item_c", 1); got != 1 {
		t.Fatalf("item_c slot=%d, want 1", got)
	}
	slotsA := []int32{bagSlotOf(t, player, "item_a", 1)}
	player.mu.Lock()
	for _, row := range player.bag {
		if row.configID == "item_a" && row.viewID == 1 && row.slot != slotsA[0] {
			slotsA = append(slotsA, row.slot)
		}
	}
	player.mu.Unlock()
	if len(slotsA) != 2 {
		t.Fatalf("item_a slots=%v, want two adjacent", slotsA)
	}
	if got := bagSlotOf(t, player, "item_b", 1); got != 4 {
		t.Fatalf("item_b slot=%d, want 4", got)
	}
	if got := bagSlotOf(t, player, "item_x", 2); got != 3 {
		t.Fatalf("item_x (other page) slot=%d, want 3 untouched", got)
	}
	frames := conn.Frames()
	// 3 次移动 × (remove 旧 + remove 新预清 + add + remove 镜像) = 12 帧。
	if len(frames) != 12 {
		t.Fatalf("arrange frames=%d, want 12", len(frames))
	}
	for index := 0; index < len(frames); index += 4 {
		if frames[index][0] != 0x19 || frames[index+1][0] != 0x19 ||
			frames[index+2][0] != 0x18 || frames[index+3][0] != 0x19 {
			t.Fatalf("arrange frame quad %d=%x", index/4, frames[index:index+4])
		}
	}

	// 再整理：已紧凑 → 无变更零帧。
	base := len(conn.Frames())
	if handled, err := handleArrangeItemCustom(h, arrangeMsg(2)); err != nil || !handled {
		t.Fatalf("re-arrange handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()) - base; got != 0 {
		t.Fatalf("re-arrange appended %d frames", got)
	}
}

func TestMoveItemPersistsThroughStore(t *testing.T) {
	store, mock := newMockInventoryDB(t)
	mock.ExpectExec("UPDATE role_bag_items SET slot = \\?, view_id = \\?").
		WithArgs(int32(5), int32(1), int64(role.RoleID(901)), int32(1), int32(1)).
		WillReturnResult(newSQLMockResult(0, 1))
	player := playerWithBag(
		bagRow{slot: 1, configID: "yaopin_00100", itemType: 2101, amount: 1, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, store)

	if handled, err := handleMoveItemCustom(h, moveItemMsg(2, 1, 2, 5)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := bagSlotOf(t, player, "yaopin_00100", 1); got != 5 {
		t.Fatalf("slot=%d", got)
	}
}

func TestWearRefusedForNonBagSource(t *testing.T) {
	player := newPlayerActor("背包侠", 0)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)
	// toView=1 但源非背包页（view 999）。
	if handled, err := handleMoveItemCustom(h, moveItemMsg(999, 1, 1, 0)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()); got != 0 {
		t.Fatalf("invalid source emitted %d frames", got)
	}
}
