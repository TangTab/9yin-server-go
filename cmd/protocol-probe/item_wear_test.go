package main

import (
	"context"
	"database/sql/driver"
	"encoding/binary"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

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
	// 两阶段：3 次移动 = 先清 6 帧旧格（主视图+镜像各 3）→ 再 3×(目标预清
	// +add+目标镜像) = 15 帧。全部旧格 remove 先行，后行 remove 不再误清
	// 前行刚落的显示（撞格覆盖缺陷的帧型修复）。
	if len(frames) != 15 {
		t.Fatalf("arrange frames=%d, want 15", len(frames))
	}
	for index := 0; index < 6; index++ {
		if frames[index][0] != 0x19 {
			t.Fatalf("arrange phase-1 frame %d=%x, want remove", index, frames[index])
		}
	}
	for index := 6; index < len(frames); index += 3 {
		if frames[index][0] != 0x19 || frames[index+1][0] != 0x18 || frames[index+2][0] != 0x19 {
			t.Fatalf("arrange push triple %d=%x", index/3, frames[index:index+3])
		}
	}
	assertNoRemoveAfterAdd(t, frames, viewToolPage)

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

// assertNoRemoveAfterAdd 客户端安全不变量：任一 add(view,slot) 之后不得再
// 出现同 view+slot 的 remove（后行 remove 清掉前行刚落的显示——整理/拖拽
// 覆盖缺陷的帧型）。镜像页 view 不同天然排除。
func assertNoRemoveAfterAdd(t *testing.T, frames [][]byte, view uint16) {
	t.Helper()
	lastRemove := map[uint16]int{}
	for index, frame := range frames {
		if len(frame) < 5 || frame[0] != 0x19 {
			continue
		}
		if binary.LittleEndian.Uint16(frame[1:]) != view {
			continue
		}
		lastRemove[binary.LittleEndian.Uint16(frame[3:])] = index
	}
	for index, frame := range frames {
		if len(frame) < 5 || frame[0] != 0x18 {
			continue
		}
		if binary.LittleEndian.Uint16(frame[1:]) != view {
			continue
		}
		slot := binary.LittleEndian.Uint16(frame[3:])
		if removeIndex, taken := lastRemove[slot]; taken && removeIndex > index {
			t.Fatalf("add @%d (view=%d slot=%d) cleared by later remove @%d", index, view, slot, removeIndex)
		}
	}
}

// TestArrangeNoSlotCollisionWhenLaterGroupHoldsLowSlot 旧分配算法撞格回归：
// 早组行被分进未处理晚组行仍占用的低格 → 同格双物品 + add 覆盖原地行。
// 组序 g1（A@1,B@3）→ g2（C@2）；顺填结果 A@1、B@2、C@3，C 不被 B 覆盖。
func TestArrangeNoSlotCollisionWhenLaterGroupHoldsLowSlot(t *testing.T) {
	player := playerWithBag(
		bagRow{slot: 1, configID: "item_a", itemType: 1, amount: 1, viewID: 1},
		bagRow{slot: 3, configID: "item_a", itemType: 1, amount: 1, viewID: 1},
		bagRow{slot: 2, configID: "item_c", itemType: 1, amount: 1, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	if handled, err := handleArrangeItemCustom(h, arrangeMsg(2)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	seen := map[int32]string{}
	player.mu.Lock()
	for _, row := range player.bag {
		if row.viewID != 1 {
			continue
		}
		if existing, taken := seen[row.slot]; taken {
			t.Fatalf("slot %d duplicated: %s vs %s", row.slot, existing, row.configID)
		}
		seen[row.slot] = row.configID
	}
	player.mu.Unlock()
	if seen[1] != "item_a" || seen[2] != "item_a" || seen[3] != "item_c" {
		t.Fatalf("arranged slots=%v, want item_a@1 item_a@2 item_c@3", seen)
	}
	assertNoRemoveAfterAdd(t, conn.Frames(), viewToolPage)
}

// TestMoveItemOntoOccupiedSwaps 拖到占用格交换：旧实现直接落位与占用行同
// 格互覆。验证两行互换、无重复槽位、同格 remove 先于 add。
func TestMoveItemOntoOccupiedSwaps(t *testing.T) {
	player := playerWithBag(
		bagRow{slot: 1, configID: "yaopin_00100", itemType: 2101, amount: 2, viewID: 1},
		bagRow{slot: 5, configID: "caiyao_00200", itemType: 2201, amount: 3, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	if handled, err := handleMoveItemCustom(h, moveItemMsg(2, 1, 2, 5)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := bagSlotOf(t, player, "yaopin_00100", 1); got != 5 {
		t.Fatalf("yaopin slot=%d, want 5 (swapped into target cell)", got)
	}
	if got := bagSlotOf(t, player, "caiyao_00200", 1); got != 1 {
		t.Fatalf("caiyao slot=%d, want 1 (occupant moved to vacated source)", got)
	}
	assertNoRemoveAfterAdd(t, conn.Frames(), viewToolPage)
}

// TestMoveItemSameConfigEquipSwapsNotMerges 同 configID 装备是各自实例
// （uid/耐久独立），拖到同款上应交换而非合并。
func TestMoveItemSameConfigEquipSwapsNotMerges(t *testing.T) {
	player := playerWithBag(
		bagRow{slot: 1, configID: "sword_dup", itemType: 101, amount: 1, viewID: 2, equipType: "Weapon"},
		bagRow{slot: 5, configID: "sword_dup", itemType: 101, amount: 1, viewID: 2, equipType: "Weapon"},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	if handled, err := handleMoveItemCustom(h, moveItemMsg(121, 1, 121, 5)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	slots := map[int32]int{}
	player.mu.Lock()
	for _, row := range player.bag {
		if row.configID == "sword_dup" {
			slots[row.slot]++
		}
	}
	player.mu.Unlock()
	if len(slots) != 2 || slots[1] != 1 || slots[5] != 1 {
		t.Fatalf("equip rows slots=%v, want both rows kept at 1 and 5 (swapped)", slots)
	}
}

// TestMoveItemOntoSameStackMerges 拖到同 configID 可叠行：并入目标行（额
// 度合加），源格清空、行数减一。
func TestMoveItemOntoSameStackMerges(t *testing.T) {
	player := playerWithBag(
		bagRow{slot: 1, configID: "yaopin_00100", itemType: 2101, amount: 2, viewID: 1},
		bagRow{slot: 5, configID: "yaopin_00100", itemType: 2101, amount: 3, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	if handled, err := handleMoveItemCustom(h, moveItemMsg(2, 1, 2, 5)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := bagAmountOf(t, player, "yaopin_00100", 1); got != 5 {
		t.Fatalf("amount=%d, want merged 5", got)
	}
	player.mu.Lock()
	rows := 0
	for _, row := range player.bag {
		if row.configID == "yaopin_00100" && row.viewID == 1 {
			rows++
			if row.slot != 5 {
				t.Fatalf("merged row slot=%d, want 5", row.slot)
			}
		}
	}
	player.mu.Unlock()
	if rows != 1 {
		t.Fatalf("rows=%d, want single merged row", rows)
	}
	assertNoRemoveAfterAdd(t, conn.Frames(), viewToolPage)
}

// TestMoveItemRejectsSlotOutOfBounds 槽位越界（91+）拒绝：零帧、内存不动
// （登录侧对 91+ 槽位显示裁剪，不允许拖拽再写入越界槽）。
func TestMoveItemRejectsSlotOutOfBounds(t *testing.T) {
	player := playerWithBag(
		bagRow{slot: 1, configID: "yaopin_00100", itemType: 2101, amount: 1, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	if handled, err := handleMoveItemCustom(h, moveItemMsg(2, 1, 2, 91)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()); got != 0 {
		t.Fatalf("out-of-bounds move emitted %d frames", got)
	}
	if got := bagSlotOf(t, player, "yaopin_00100", 1); got != 1 {
		t.Fatalf("slot=%d, want unchanged 1", got)
	}
}

// TestMoveItemSwapPersistsBySeq 拖拽交换落盘：seq 键控（槽位键在中途态
// 双中两行），单事务两条 UPDATE。
func TestMoveItemSwapPersistsBySeq(t *testing.T) {
	store, mock := newMockInventoryDB(t)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE role_bag_items SET slot = \\?, view_id = \\?").
		WithArgs(int32(5), int32(1), int64(role.RoleID(901)), int64(11)).
		WillReturnResult(newSQLMockResult(0, 1))
	mock.ExpectExec("UPDATE role_bag_items SET slot = \\?, view_id = \\?").
		WithArgs(int32(1), int32(1), int64(role.RoleID(901)), int64(12)).
		WillReturnResult(newSQLMockResult(0, 1))
	mock.ExpectCommit()
	player := playerWithBag(
		bagRow{slot: 1, configID: "yaopin_00100", itemType: 2101, amount: 1, viewID: 1, seq: 11},
		bagRow{slot: 5, configID: "caiyao_00200", itemType: 2201, amount: 1, viewID: 1, seq: 12},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, store)

	if handled, err := handleMoveItemCustom(h, moveItemMsg(2, 1, 2, 5)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := bagSlotOf(t, player, "yaopin_00100", 1); got != 5 {
		t.Fatalf("yaopin slot=%d, want 5", got)
	}
	if got := bagSlotOf(t, player, "caiyao_00200", 1); got != 1 {
		t.Fatalf("caiyao slot=%d, want 1", got)
	}
}

// TestMoveItemMergePersistsBySeq 拖拽叠堆合并落盘：dst 加数 + src 删行 +
// 回读总额，单事务。
func TestMoveItemMergePersistsBySeq(t *testing.T) {
	store, mock := newMockInventoryDB(t)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE role_bag_items SET amount = amount \\+ \\?").
		WithArgs(int32(2), int64(role.RoleID(901)), int64(11)).
		WillReturnResult(newSQLMockResult(0, 1))
	mock.ExpectExec("DELETE FROM role_bag_items WHERE role_id = \\? AND seq = \\?").
		WithArgs(int64(role.RoleID(901)), int64(12)).
		WillReturnResult(newSQLMockResult(0, 1))
	readback := sqlmock.NewRows([]string{"amount"}).AddRow(5)
	mock.ExpectQuery("SELECT amount FROM role_bag_items WHERE role_id = \\? AND seq = \\?").
		WithArgs(int64(role.RoleID(901)), int64(11)).
		WillReturnRows(readback)
	mock.ExpectCommit()
	player := playerWithBag(
		bagRow{slot: 1, configID: "yaopin_00100", itemType: 2101, amount: 2, viewID: 1, seq: 12},
		bagRow{slot: 5, configID: "yaopin_00100", itemType: 2101, amount: 3, viewID: 1, seq: 11},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, store)

	if handled, err := handleMoveItemCustom(h, moveItemMsg(2, 1, 2, 5)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := bagAmountOf(t, player, "yaopin_00100", 1); got != 5 {
		t.Fatalf("amount=%d, want DB readback 5", got)
	}
}

// TestArrangePersistsThroughStore 整理落盘：每条移动一行 seq 键控 UPDATE
// （不再"不落盘"——内存/DB 槽位分叉会让后续槽位键写入命中错行）。
func TestArrangePersistsThroughStore(t *testing.T) {
	store, mock := newMockInventoryDB(t)
	// 移动 A(3→2)、A(7→3)、B(9→4)。
	mock.ExpectBegin()
	for _, args := range [][]driver.Value{
		{int32(2), int32(1), int64(role.RoleID(901)), int64(21)},
		{int32(3), int32(1), int64(role.RoleID(901)), int64(22)},
		{int32(4), int32(1), int64(role.RoleID(901)), int64(23)},
	} {
		mock.ExpectExec("UPDATE role_bag_items SET slot = \\?, view_id = \\?").
			WithArgs(args...).
			WillReturnResult(newSQLMockResult(0, 1))
	}
	mock.ExpectCommit()
	player := playerWithBag(
		bagRow{slot: 1, configID: "item_c", itemType: 1, amount: 1, viewID: 1, seq: 20},
		bagRow{slot: 3, configID: "item_a", itemType: 1, amount: 1, viewID: 1, seq: 21},
		bagRow{slot: 7, configID: "item_a", itemType: 1, amount: 1, viewID: 1, seq: 22},
		bagRow{slot: 9, configID: "item_b", itemType: 1, amount: 1, viewID: 1, seq: 23},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, store)

	if handled, err := handleArrangeItemCustom(h, arrangeMsg(2)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := bagSlotOf(t, player, "item_c", 1); got != 1 {
		t.Fatalf("item_c slot=%d, want 1", got)
	}
	if got := bagSlotOf(t, player, "item_b", 1); got != 4 {
		t.Fatalf("item_b slot=%d, want 4", got)
	}
}

// TestMoveSwapArrangeRePushModifyPackRecords 镇派兵器模板包 record（394/395）
// 挂在客户端 (view,slot) 上：移动/交换/整理清格后必须按新槽位重推，
// 否则背包内换位后属性消失（重登由 pushLoginRandomProps 才恢复——
// 2026-11-12 真机报告根因）。
func TestMoveSwapArrangeRePushModifyPackRecords(t *testing.T) {
	c := &randomEquipCatalog{
		itemFields: map[string]randomItemFields{
			"blade_zpsb_test": {propPacks: []int{95258}, skillPacks: []int{9659}},
			"yaopin_plain":    {},
		},
	}
	// 注入：先落定 Once 再覆写全局（TestPushModifyPackRecordsFrames 同法）。
	randomEquipCatalogOf()
	old := randomEquipCat
	randomEquipCat = c
	defer func() { randomEquipCat = old }()

	recordTargets := func(frames [][]byte, record uint16) []uint32 {
		var targets []uint32
		for _, f := range frames {
			if len(f) < 16 || f[0] != 0x11 || f[1] != 1 {
				continue
			}
			if binary.LittleEndian.Uint16(f[10:]) != record {
				continue
			}
			targets = append(targets, binary.LittleEndian.Uint32(f[6:]))
		}
		return targets
	}

	// 交换：装备页 slot1(镇派) <-> slot5(普通)。
	player := playerWithBag(
		bagRow{slot: 1, configID: "blade_zpsb_test", itemType: 2201, amount: 1, viewID: 2},
		bagRow{slot: 5, configID: "yaopin_plain", itemType: 2101, amount: 3, viewID: 2},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)
	if handled, err := handleMoveItemCustom(h, moveItemMsg(121, 1, 121, 5)); err != nil || !handled {
		t.Fatalf("swap handled=%t err=%v", handled, err)
	}
	if got := bagSlotOf(t, player, "blade_zpsb_test", 2); got != 5 {
		t.Fatalf("blade slot=%d, want 5", got)
	}
	frames := conn.Frames()
	if got := recordTargets(frames, 394); len(got) != 1 || got[0] != 5 {
		t.Fatalf("swap prop-pack targets=%v, want [5]（新槽位重推）", got)
	}
	if got := recordTargets(frames, 395); len(got) != 1 || got[0] != 5 {
		t.Fatalf("swap skill-pack targets=%v, want [5]", got)
	}

	// 移动到空格：slot5(镇派) -> slot9。
	player2 := playerWithBag(
		bagRow{slot: 5, configID: "blade_zpsb_test", itemType: 2201, amount: 1, viewID: 2},
	)
	conn2 := &captureMessageConnection{}
	h2 := newTestItemSession(conn2, player2, nil)
	if handled, err := handleMoveItemCustom(h2, moveItemMsg(121, 5, 121, 9)); err != nil || !handled {
		t.Fatalf("move handled=%t err=%v", handled, err)
	}
	if got := recordTargets(conn2.Frames(), 394); len(got) != 1 || got[0] != 9 {
		t.Fatalf("move prop-pack targets=%v, want [9]", got)
	}

	// 整理：yaopin@1 不动、镇派 slot3 -> slot2，record 落新槽位。
	player3 := playerWithBag(
		bagRow{slot: 1, configID: "yaopin_plain", itemType: 2101, amount: 3, viewID: 2},
		bagRow{slot: 3, configID: "blade_zpsb_test", itemType: 2201, amount: 1, viewID: 2},
	)
	conn3 := &captureMessageConnection{}
	h3 := newTestItemSession(conn3, player3, nil)
	if handled, err := handleArrangeItemCustom(h3, arrangeMsg(121)); err != nil || !handled {
		t.Fatalf("arrange handled=%t err=%v", handled, err)
	}
	if got := bagSlotOf(t, player3, "blade_zpsb_test", 2); got != 2 {
		t.Fatalf("arrange blade slot=%d, want 2", got)
	}
	if got := recordTargets(conn3.Frames(), 394); len(got) != 1 || got[0] != 2 {
		t.Fatalf("arrange prop-pack targets=%v, want [2]", got)
	}
}
