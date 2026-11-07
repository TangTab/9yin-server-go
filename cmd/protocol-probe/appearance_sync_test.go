package main

import (
	"encoding/binary"
	"strings"
	"testing"
)

// 镜像 Java AppearanceSyncTest：三模式裁决 / 登录合并 / 0x10 帧属性序。
// 桩表覆盖装备链（ArtPack→男女模型）与套装分件（HatArtPack/PlantsArtPack/
// ShoesArtPack，含 -1 哨兵）。

func injectStubAppearanceResolver(t *testing.T) {
	t.Helper()
	old := equipmentModels
	equipmentModels = &equipmentResolver{
		artByItem: map[string]string{
			"cloth_gear": "gear_pack", "suit_shaolin01": "suit_pack",
			"FASHION_SUIT_G027": "fashion_pack", "fashion_hat_a": "hat_pack",
			"fashion_shoes_a": "shoes_pack",
		},
		models: map[string]itemModels{
			"gear_pack":    {male: `obj\m\gear_cloth`, female: `obj\f\gear_cloth`},
			"suit_pack":    {male: `obj\m\suit_cloth`, female: `obj\f\suit_cloth`},
			"fashion_pack": {male: `obj\m\fashion_coat`, female: `obj\f\fashion_coat`},
			"hat_pack":     {male: `obj\m\fashion_hat`, female: `obj\f\fashion_hat`},
			"shoes_pack":   {male: `obj\m\fashion_shoes`, female: `obj\f\fashion_shoes`},
			"suit_hat":     {male: `obj\m\suit_hat`},
			"suit_plants":  {male: `obj\m\suit_pants`},
			"suit_shoes":   {male: `obj\m\suit_shoes`},
			"coat_hat":     {male: `obj\m\coat_hat`},
			"coat_shoes":   {male: `obj\m\coat_shoes`},
		},
		packs: map[string]artifactPacks{
			"suit_shaolin01":    {hat: "suit_hat", plants: "suit_plants", shoes: "suit_shoes"},
			"FASHION_SUIT_G027": {hat: "coat_hat", plants: "-1", shoes: "coat_shoes"},
		},
		statics: map[string]itemStatic{
			"fashion_suit_g027": {equipType: "FashionCoat"},
			"suit_shaolin01":    {equipType: "Suit"},
		},
	}
	t.Cleanup(func() { equipmentModels = old })
}

func stubAppearanceEquips() []equipRow {
	return []equipRow{
		{configID: "cloth_gear", equipType: "Cloth", slot: 3},
		{configID: "suit_shaolin01", equipType: "Suit", slot: 16},
		{configID: "FASHION_SUIT_G027", equipType: "FashionCoat", slot: 17},
		{configID: "fashion_hat_a", equipType: "FashionHat", slot: 18},
	}
}

// 江湖装（0）：衣取装备槽 3，身份/时装槽不参与。
func TestFinalModelsGearUsesWornEquipment(t *testing.T) {
	injectStubAppearanceResolver(t)
	finals := finalModels(showEquipGear, 0, stubAppearanceEquips())
	if finals[partCloth] != `obj\m\gear_cloth` {
		t.Fatalf("cloth=%q, want 装备模型", finals[partCloth])
	}
	if _, has := finals[partHat]; has {
		t.Fatalf("hat=%q, want 无帽部位", finals[partHat])
	}
}

// 身份装（1）：槽 16 整套——主 ArtPack=衣 + Hat/Plants/Shoes 分件覆盖。
func TestFinalModelsIdentitySuitOverridesAllParts(t *testing.T) {
	injectStubAppearanceResolver(t)
	finals := finalModels(showEquipIdentity, 0, stubAppearanceEquips())
	if finals[partCloth] != `obj\m\suit_cloth` || finals[partHat] != `obj\m\suit_hat` ||
		finals[partPants] != `obj\m\suit_pants` || finals[partShoes] != `obj\m\suit_shoes` {
		t.Fatalf("identity finals=%+v, want 整套四件全覆", finals)
	}
}

// 时装（2）：套装衣自带帽/鞋 ArtPack 先行，单件帽/鞋槽再覆盖；-1 分件
// 不产生部位（套装衣无裤件——Pants 保持装备值）。
func TestFinalModelsFashionCoatWithSinglePieceOverride(t *testing.T) {
	injectStubAppearanceResolver(t)
	finals := finalModels(showEquipFashion, 0, stubAppearanceEquips())
	if finals[partCloth] != `obj\m\fashion_coat` {
		t.Fatalf("cloth=%q, want 套装衣", finals[partCloth])
	}
	if finals[partHat] != `obj\m\fashion_hat` {
		t.Fatalf("hat=%q, want 单件帽覆盖套装自带帽", finals[partHat])
	}
	if finals[partShoes] != `obj\m\coat_shoes` {
		t.Fatalf("shoes=%q, want 套装自带鞋（无单件鞋槽）", finals[partShoes])
	}
	if _, has := finals[partPants]; has {
		t.Fatalf("pants=%q, want 时装无裤件", finals[partPants])
	}
}

// 女性按 FemaleModel 取列；分件同样按性别。
func TestFinalModelsFemaleColumns(t *testing.T) {
	injectStubAppearanceResolver(t)
	finals := finalModels(showEquipFashion, 1, stubAppearanceEquips())
	if finals[partCloth] != `obj\f\fashion_coat` || finals[partHat] != `obj\f\fashion_hat` {
		t.Fatalf("female finals=%+v", finals)
	}
}

// 登录合并：已穿装备覆盖创角初始 roleVisual（其余部位保持）。
func TestMergeVisualOverridesWornPartsOnly(t *testing.T) {
	injectStubAppearanceResolver(t)
	base := roleVisual{sex: 0, cloth: "cloth_b0001", pants: "pants_b0001", shoes: "shoes_b0001"}
	merged := mergeVisual(base, []equipRow{{configID: "cloth_gear", slot: 3}})
	if merged.cloth != `obj\m\gear_cloth` {
		t.Fatalf("cloth=%q, want 装备模型覆盖初始装", merged.cloth)
	}
	if merged.pants != "pants_b0001" || merged.shoes != "shoes_b0001" {
		t.Fatalf("pants/shoes 被意外改写: %+v", merged)
	}
}

// 0x10 帧属性序：ShowEquipType(408) 首、衣裤鞋帽/Fashion* 空值跳过、
// 武器三属性恒写（空手 ActionSet 回 1h 基线）。
func TestAppearanceFramePropertyOrderAndSkips(t *testing.T) {
	injectStubAppearanceResolver(t)
	frame, err := appearanceFrame(playerObjectID, playerOwnerID, showEquipIdentity, 0, stubAppearanceEquips())
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x10 {
		t.Fatalf("opcode=%#x", frame[0])
	}
	if count := binary.LittleEndian.Uint16(frame[10:]); count != 10 {
		t.Fatalf("property count=%d, want 10（408 + 衣裤鞋帽 4 + FashionCoat/Hat 2 + 武器 3）", count)
	}
	// 首属性 ShowEquipType(408) byte=1。
	if index := binary.LittleEndian.Uint16(frame[12:]); index != propShowEquipType || frame[14] != 1 {
		t.Fatalf("first property index=%d value=%d, want 408/1", index, frame[14])
	}
	// 尾部武器三属性恒写（无武器 → 空 Weapon/ShotWeapon + 1h）。
	text := string(frame)
	if !strings.Contains(text, "1h") {
		t.Fatalf("unarmed action set fallback missing: %x", frame[len(frame)-24:])
	}

	// 空穿戴：只剩 ShowEquipType + 武器三属性（衣裤鞋帽全跳过）。
	bare, err := appearanceFrame(playerObjectID, playerOwnerID, showEquipGear, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if count := binary.LittleEndian.Uint16(bare[10:]); count != 4 {
		t.Fatalf("bare property count=%d, want 4", count)
	}
}

// 登录补帧门槛：帽/16-19 任一占用或非江湖装 → 需要补帧；纯衣裤鞋+0 型零帧。
func TestAppearanceOverlayOccupiedGate(t *testing.T) {
	if !appearanceOverlayOccupied(stubAppearanceEquips(), showEquipGear) {
		t.Fatal("fashion slots occupied should require top-up")
	}
	if appearanceOverlayOccupied([]equipRow{{configID: "cloth_gear", slot: 3}}, showEquipGear) {
		t.Fatal("plain gear at type 0 should skip top-up（帧序对齐 golden）")
	}
	if !appearanceOverlayOccupied([]equipRow{{configID: "cloth_gear", slot: 3}}, showEquipFashion) {
		t.Fatal("non-gear display type always requires top-up")
	}
}

// mirrorDollSlot 镜像映射（16-19 → 40-43）。
func TestMirrorDollSlotMapping(t *testing.T) {
	for slot, mirror := range map[int32]int32{16: 40, 17: 41, 18: 42, 19: 43, 3: 3, 22: 22} {
		if got := mirrorDollSlot(slot); got != mirror {
			t.Fatalf("mirror(%d)=%d, want %d", slot, got, mirror)
		}
	}
}
