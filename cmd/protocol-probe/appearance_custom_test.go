package main

import (
	"context"
	"encoding/binary"
	"testing"
	"time"
)

// 镜像 Java AppearanceModeFlowTest / SkinWardrobeFlowTest：
// C2S 61 显示类型切换、C2S 19 时装穿/脱/买、C2S 180 衣橱状态机。
// 全部走内存降级（store=nil），帧断言用 captureMessageConnection。

func injectStubSkinCatalog(t *testing.T) {
	t.Helper()
	old := skinCards
	skinCards = &skinCatalog{
		byID: map[int]skin{
			1: {id: 1, configID: "blade_card_001", male: []skinPartModel{{part: "Weapon", model: "blade_card_001"}}},
			2: {id: 2, configID: "sword_card_001", male: []skinPartModel{{part: "Weapon", model: "sword_card_001"}}},
			3: {id: 3, configID: "hair_card_001", male: []skinPartModel{{part: "Hat", model: "hair_card_001"}}},
			4: {id: 4, configID: "suit_card_001", male: []skinPartModel{
				{part: "Cloth", model: "suit_c"}, {part: "Pants", model: "suit_p"}, {part: "Shoes", model: "suit_s"}}},
		},
		ids:   []int{1, 2, 3, 4},
		byCfg: map[string]int{"blade_card_001": 1, "sword_card_001": 2, "hair_card_001": 3, "suit_card_001": 4},
	}
	t.Cleanup(func() { skinCards = old })
}

func injectStubCardItemCatalog(t *testing.T) {
	t.Helper()
	oldOnce := cardItemCatalogOnce
	oldLoaded := cardItemCatalogLoaded
	cardItemCatalogOnce = func() map[string]int32 {
		return map[string]int32{"card_item_1": 1, "card_item_2": 2, "card_item_3": 3, "card_item_4": 4}
	}
	cardItemCatalogLoaded = nil
	t.Cleanup(func() { cardItemCatalogOnce, cardItemCatalogLoaded = oldOnce, oldLoaded })
}

func newTestAppearanceSession(conn *captureMessageConnection, player *playerActor) *appearanceSession {
	return &appearanceSession{
		link: conn, player: player, remote: "test",
		ctx: context.Background(), roleID: 1,
	}
}

func newTestItemSessionForSkin(conn *captureMessageConnection, player *playerActor) *itemSession {
	return &itemSession{link: conn, player: player, remote: "test", ctx: context.Background(), roleID: 1, now: time.Now}
}

func showEquipTypeUpload(showEquipType int32) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomShowEquipType},
		{Type: 2, Int32: showEquipType},
	}}
}

func fashionSuitUpload(sub int32, text string) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomFashionSuit},
		{Type: 2, Int32: sub},
		{Type: 6, Text: text},
	}}
}

func wardrobeUpload(sub int32, skinID int32) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomWardrobeCard},
		{Type: 2, Int32: sub},
		{Type: 2, Int32: skinID},
	}}
}

// 61：合法值回推单帧 0x10（首属性 ShowEquipType）并更新内存；
// 非法值静默零帧。
func TestHandleShowEquipTypeCustomValidatesRange(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	session := newTestAppearanceSession(conn, player)

	handled, err := handleShowEquipTypeCustom(session, showEquipTypeUpload(2))
	if err != nil || !handled || len(conn.frames) != 1 {
		t.Fatalf("valid handled=%t err=%v frames=%d", handled, err, len(conn.frames))
	}
	frame := conn.frames[0]
	if frame[0] != 0x10 || binary.LittleEndian.Uint16(frame[12:]) != propShowEquipType || frame[14] != 2 {
		t.Fatalf("refresh frame=%x, want 0x10 ShowEquipType=2", frame)
	}
	if got := effectiveShowEquipType(player); got != showEquipFashion {
		t.Fatalf("effective=%d, want 2", got)
	}

	invalid := &captureMessageConnection{}
	handled, err = handleShowEquipTypeCustom(newTestAppearanceSession(invalid, player), showEquipTypeUpload(3))
	if err != nil || !handled || len(invalid.frames) != 0 {
		t.Fatalf("invalid handled=%t err=%v frames=%d", handled, err, len(invalid.frames))
	}
	if got := effectiveShowEquipType(player); got != showEquipFashion {
		t.Fatalf("invalid upload changed state: %d", got)
	}
}

// 19 sub1 穿上：激活集校验 → 穿戴帧组（含镜像 40-43）→ 自动切显示类型；
// 未激活配置拒绝。
func TestHandleFashionSuitWearAndRemove(t *testing.T) {
	injectStubAppearanceResolver(t)
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	player.restoreFashionSuits([]string{"FASHION_SUIT_G027"})
	session := newTestAppearanceSession(conn, player)

	// 未激活：拒绝零帧。
	rejectConn := &captureMessageConnection{}
	if handled, err := handleFashionSuitCustom(newTestAppearanceSession(rejectConn, player), fashionSuitUpload(1, "unknown_suit")); err != nil || !handled || len(rejectConn.frames) != 0 {
		t.Fatalf("unactivated handled=%t err=%v frames=%d", handled, err, len(rejectConn.frames))
	}

	// 穿上 FashionCoat → 槽 17 + 镜像 41 + 外观帧 + 自动切 2。
	if _, err := handleFashionSuitCustom(session, fashionSuitUpload(1, "FASHION_SUIT_G027")); err != nil {
		t.Fatal(err)
	}
	if worn, _, ok := player.equipAtSlot(17); !ok || worn.configID != "FASHION_SUIT_G027" {
		t.Fatalf("worn=%+v ok=%t, want FASHION_SUIT_G027@17", worn, ok)
	}
	var sawMirror bool
	for _, frame := range conn.Frames() {
		if frame[0] == 0x19 && binary.LittleEndian.Uint16(frame[1:]) == viewDoll && binary.LittleEndian.Uint16(frame[3:]) == 41 {
			sawMirror = true
		}
	}
	if !sawMirror {
		t.Fatal("mirror slot 41 view remove missing")
	}
	if got := effectiveShowEquipType(player); got != showEquipFashion {
		t.Fatalf("effective=%d, want 穿上时装自动切 2", got)
	}

	// sub2 脱下（EquipType 槽名）：纸娃娃双槽清位 + 外观整刷。
	base := len(conn.Frames())
	if _, err := handleFashionSuitCustom(session, fashionSuitUpload(2, "FashionCoat")); err != nil {
		t.Fatal(err)
	}
	if _, _, still := player.equipAtSlot(17); still {
		t.Fatal("slot 17 not removed")
	}
	frames := conn.Frames()[base:]
	if len(frames) < 3 || frames[0][0] != 0x19 || binary.LittleEndian.Uint16(frames[0][3:]) != 17 ||
		binary.LittleEndian.Uint16(frames[1][3:]) != 41 {
		t.Fatalf("remove frames=%x", frames)
	}
	if frames[len(frames)-1][0] != 0x10 {
		t.Fatalf("appearance refresh missing: %x", frames[len(frames)-1])
	}

	// 空槽幂等：零帧。
	idle := &captureMessageConnection{}
	if handled, err := handleFashionSuitCustom(newTestAppearanceSession(idle, player), fashionSuitUpload(2, "FashionCoat")); err != nil || !handled || len(idle.frames) != 0 {
		t.Fatalf("idempotent remove handled=%t err=%v frames=%d", handled, err, len(idle.frames))
	}
}

// 19 sub3 购买：会话级扣 1 金 + 激活 + 补行；余额不足拒绝。
func TestHandleFashionSuitBuyDebitsGold(t *testing.T) {
	injectStubAppearanceResolver(t)
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	player.setGold(2)
	session := newTestAppearanceSession(conn, player)

	if _, err := handleFashionSuitCustom(session, fashionSuitUpload(3, "FASHION_SUIT_G027")); err != nil {
		t.Fatal(err)
	}
	if got := player.goldSnapshot(); got != 1 {
		t.Fatalf("gold=%d, want 1（扣 1 金）", got)
	}
	if !containsIgnoreCase(player.fashionSuitSnapshot(), "FASHION_SUIT_G027") {
		t.Fatalf("activated=%v", player.fashionSuitSnapshot())
	}
	// 重复购买幂等（不再扣金）。
	if _, err := handleFashionSuitCustom(session, fashionSuitUpload(3, "FASHION_SUIT_G027")); err != nil {
		t.Fatal(err)
	}
	if got := player.goldSnapshot(); got != 1 {
		t.Fatalf("gold=%d after repeat buy, want 1", got)
	}
	// 余额不足拒绝（清零后再买）。
	player.setGold(0)
	if _, err := handleFashionSuitCustom(session, fashionSuitUpload(3, "suit_shaolin01")); err != nil {
		t.Fatal(err)
	}
	if containsIgnoreCase(player.fashionSuitSnapshot(), "suit_shaolin01") {
		t.Fatal("insufficient gold should not activate")
	}
}

// 180 衣橱状态机：收录→挂载幂等；同部件互斥、异部件叠加；未收录 USE
// 忽略；卸载回落存档偏好。
func TestWardrobeCardStateMachine(t *testing.T) {
	injectStubSkinCatalog(t)
	injectStubCardItemCatalog(t)
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	player.setShowEquipTypePref(showEquipIdentity)
	session := newTestItemSessionForSkin(conn, player)

	// 未收录 USE 忽略（USE 从不自动收录）。
	if handled, err := handleWardrobeCardCustom(session, wardrobeUpload(0, 1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	state := player.skinSnapshot()
	if len(state.collected) != 0 || len(state.mounted) != 0 {
		t.Fatalf("uncollected use mutated state: %+v", state)
	}

	// 收录 1 → collected=[1]，无自动挂载。
	if _, err := handleWardrobeCardCustom(session, wardrobeUpload(7, 1)); err != nil {
		t.Fatal(err)
	}
	state = player.skinSnapshot()
	if len(state.collected) != 1 || len(state.mounted) != 0 {
		t.Fatalf("collect state=%+v", state)
	}
	// 重复收录幂等。
	if _, err := handleWardrobeCardCustom(session, wardrobeUpload(7, 1)); err != nil {
		t.Fatal(err)
	}
	if got := len(player.skinSnapshot().collected); got != 1 {
		t.Fatalf("repeat collect=%d, want 1", got)
	}

	// 挂载 1 → mounted=[1]，effective 切 2。
	if _, err := handleWardrobeCardCustom(session, wardrobeUpload(0, 1)); err != nil {
		t.Fatal(err)
	}
	if got := effectiveShowEquipType(player); got != showEquipFashion {
		t.Fatalf("effective=%d, want mounted→2", got)
	}
	// 重复挂载幂等。
	if _, err := handleWardrobeCardCustom(session, wardrobeUpload(0, 1)); err != nil {
		t.Fatal(err)
	}

	// 同部件互斥：收录+挂载皮肤 2（同 Weapon）替换 1；异部件（发型 3）叠加。
	for _, id := range []int32{2, 3} {
		if _, err := handleWardrobeCardCustom(session, wardrobeUpload(7, id)); err != nil {
			t.Fatal(err)
		}
		if _, err := handleWardrobeCardCustom(session, wardrobeUpload(0, id)); err != nil {
			t.Fatal(err)
		}
	}
	state = player.skinSnapshot()
	if len(state.mounted) != 2 || state.mounted[0] != 2 || state.mounted[1] != 3 {
		t.Fatalf("mounted=%v, want [2 3]（武器互斥替换、发型叠加）", state.mounted)
	}

	// 卸载 2：mounted=[3]；effective 仍 2（3 还挂着）。
	if _, err := handleWardrobeCardCustom(session, wardrobeUpload(1, 2)); err != nil {
		t.Fatal(err)
	}
	state = player.skinSnapshot()
	if len(state.mounted) != 1 || state.mounted[0] != 3 || len(state.collected) != 3 {
		t.Fatalf("cancel state=%+v", state)
	}
	// 卸载 3：全部卸下 → effective 回落存档偏好 1。
	if _, err := handleWardrobeCardCustom(session, wardrobeUpload(1, 3)); err != nil {
		t.Fatal(err)
	}
	if got := effectiveShowEquipType(player); got != showEquipIdentity {
		t.Fatalf("effective=%d after unmount all, want 回落偏好 1", got)
	}

	// 未知皮肤号全子操作忽略。
	idle := &captureMessageConnection{}
	idleSession := newTestItemSessionForSkin(idle, player)
	for _, sub := range []int32{0, 1, 7} {
		if handled, err := handleWardrobeCardCustom(idleSession, wardrobeUpload(sub, 999999)); err != nil || !handled {
			t.Fatalf("unknown sub=%d handled=%t err=%v", sub, handled, err)
		}
	}
	if len(idle.frames) != 0 {
		t.Fatalf("unknown skin frames=%d, want 0", len(idle.frames))
	}
}

// 31 使用卡片道具 → 消耗 1 张收录（不自动挂载）。
func TestCollectSkinFromItemUse(t *testing.T) {
	injectStubSkinCatalog(t)
	injectStubCardItemCatalog(t)
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	player.restoreSessionInventory(0, nil, []bagRow{
		{slot: 4, configID: "card_item_1", itemType: 1, amount: 2, viewID: 1},
	}, nil)
	session := newTestItemSessionForSkin(conn, player)

	collectSkinFromItemUse(session, bagRow{slot: 4, configID: "card_item_1", itemType: 1, amount: 2, viewID: 1})
	state := player.skinSnapshot()
	if len(state.collected) != 1 || state.collected[0] != 1 {
		t.Fatalf("collected=%v, want [1]", state.collected)
	}
	if len(state.mounted) != 0 {
		t.Fatalf("mounted=%v, use 不自动挂载", state.mounted)
	}
	// 消耗后背包行 amount 2→1。
	if got := player.bag[0].amount; got != 1 {
		t.Fatalf("bag amount=%d, want 1", got)
	}
}

// 有效显示类型优先级：挂载 > 存档偏好 > 默认 0。
func TestEffectiveShowEquipTypePrecedence(t *testing.T) {
	player := newPlayerActor("测试", 0)
	if got := effectiveShowEquipType(player); got != showEquipGear {
		t.Fatalf("default=%d, want 0", got)
	}
	player.setShowEquipTypePref(showEquipIdentity)
	if got := effectiveShowEquipType(player); got != showEquipIdentity {
		t.Fatalf("pref=%d, want 1", got)
	}
	player.applySkins(skinState{mounted: []int32{9}})
	if got := effectiveShowEquipType(player); got != showEquipFashion {
		t.Fatalf("mounted=%d, want 2", got)
	}
}

// 皮肤部件模型：选角合并只取 Cloth/Pants/Shoes，后挂覆盖同部件。
func TestSkinCardModelsPartsAndOverride(t *testing.T) {
	injectStubSkinCatalog(t)
	models := skinCardModels([]int32{4, 1}, 0)
	if len(models) != 3 || models[partCloth] != "suit_c" || models[partPants] != "suit_p" || models[partShoes] != "suit_s" {
		t.Fatalf("models=%v", models)
	}
	// 再挂一个 Cloth 皮肤覆盖同部件（Weapon 不进选角模型）。
	skinCards.byID[5] = skin{id: 5, male: []skinPartModel{{part: "Cloth", model: "late_cloth"}}}
	skinCards.ids = append(skinCards.ids, 5)
	overridden := skinCardModels([]int32{4, 5}, 0)
	if overridden[partCloth] != "late_cloth" {
		t.Fatalf("cloth=%v, want 后挂覆盖", overridden[partCloth])
	}
}
