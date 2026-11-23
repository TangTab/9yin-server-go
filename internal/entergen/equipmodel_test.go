package entergen

// 装备驱动身体模型推导测试（2026-11-23 官服穿/脱真机口径：空槽=
// jianghu000 裸身族、{part}_{b|g}NNNN=jianghu 系号族、PhyDef 0 基线+
// 布衣每件 +6；女号 g_ 镜像）。

import (
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

func TestEquipSlotModelBare(t *testing.T) {
	state := &rolestate.RoleState{Sex: 1}
	for slot, want := range map[int]string{
		EquipSlotCloth: `obj\char\b_jianghu000\b_cloth000`,
		EquipSlotPants: `obj\char\b_jianghu000\b_pants000`,
		EquipSlotShoes: `obj\char\b_jianghu000\b_shoes000`,
	} {
		if got := EquipSlotModel(state, slot); got != want {
			t.Fatalf("slot %d bare = %q want %q", slot, got, want)
		}
	}
	if got := EquipSlotModel(&rolestate.RoleState{Sex: 2}, EquipSlotCloth); got != `obj\char\g_jianghu000\g_cloth000` {
		t.Fatalf("female bare cloth = %q", got)
	}
}

func TestEquipSlotModelClothed(t *testing.T) {
	state := &rolestate.RoleState{Sex: 1, Equips: []rolestate.EquipItem{
		{Slot: EquipSlotCloth, ConfigID: "cloth_b0001"},
		{Slot: EquipSlotPants, ConfigID: "pants_b0003"}, // #25 混搭=穿装列实证
		{Slot: EquipSlotShoes, ConfigID: "shoes_b0003"},
	}}
	if got := EquipSlotModel(state, EquipSlotCloth); got != `obj\char\b_jianghu001\b_cloth001` {
		t.Fatalf("cloth = %q", got)
	}
	if got := EquipSlotModel(state, EquipSlotPants); got != `obj\char\b_jianghu003\b_pants003` {
		t.Fatalf("pants = %q", got)
	}
	if got := EquipSlotModel(state, EquipSlotShoes); got != `obj\char\b_jianghu003\b_shoes003` {
		t.Fatalf("shoes = %q", got)
	}
	// 女号布衣 g0001 → g_001 族（官服 capture-3 实证）。
	female := &rolestate.RoleState{Sex: 2, Equips: []rolestate.EquipItem{
		{Slot: EquipSlotShoes, ConfigID: "shoes_g0001"},
	}}
	if got := EquipSlotModel(female, EquipSlotShoes); got != `obj\char\g_jianghu001\g_shoes001` {
		t.Fatalf("female shoes = %q", got)
	}
	// 异常档：槽上挂着非本部位件 → 回落裸身。
	odd := &rolestate.RoleState{Sex: 1, Equips: []rolestate.EquipItem{
		{Slot: EquipSlotCloth, ConfigID: "weapon_x99"},
	}}
	if got := EquipSlotModel(odd, EquipSlotCloth); got != `obj\char\b_jianghu000\b_cloth000` {
		t.Fatalf("odd cloth = %q", got)
	}
	// 帽槽=无帽回填发型语义（建角外观槽 [4] 优先）。
	hat := &rolestate.RoleState{Sex: 1, Appearance: []string{"", "", "", "", `obj\char\b_hair\b_hair3`}}
	if got := EquipSlotModel(hat, EquipSlotHat); got != `obj\char\b_hair\b_hair3` {
		t.Fatalf("hat fallback hair = %q", got)
	}
}

func TestEquipPhyDef(t *testing.T) {
	if got := EquipPhyDef(&rolestate.RoleState{}); got != 0 {
		t.Fatalf("bare PhyDef = %d", got)
	}
	state := &rolestate.RoleState{Equips: []rolestate.EquipItem{
		{Slot: EquipSlotCloth, ConfigID: "cloth_b0001"},
		{Slot: EquipSlotPants, ConfigID: "pants_b0001"},
		{Slot: EquipSlotShoes, ConfigID: "shoes_b0001"},
	}}
	if got := EquipPhyDef(state); got != 18 {
		t.Fatalf("三件布衣 PhyDef = %d want 18", got)
	}
}
