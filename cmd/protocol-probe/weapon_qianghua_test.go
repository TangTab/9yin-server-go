package main

import (
	"os"
	"path/filepath"
	"testing"
)

// 武器强化（淬炼）域（C2S 861）：equip_prop_rand 规则装载/定位 + 主链
//（穿戴武器 + 融合石消耗 + 碎银扣减 + 词条落档）。

func stubQiangHuaCatalog(t *testing.T) *qiangHuaCatalog {
	t.Helper()
	dir := t.TempDir()
	ruleINI := "[1]\nCost=1000\nItem=ronghe_30001,3,0\nJobName=sh_cf\nJobLevel=1\n" +
		"PropPack=63305,63306,63307\n\n[2]\nCost=2000\nItem=ronghe_30002,1,0\nJobName=sh_cf\nJobLevel=4\n" +
		"PropPack=63405,63406\n\n[3]\nCost=500\nItem=ronghe_50003,2,0\nJobName=sh_ds\nJobLevel=4\n" +
		"PropPack=62924,62926\n\n[4]\nCost=10\nItem=ronghe_empty,1,0\nJobName=sh_cf\n"
	if err := os.WriteFile(filepath.Join(dir, "equip_prop_rand.ini"), []byte(ruleINI), 0o644); err != nil {
		t.Fatalf("write rule: %v", err)
	}
	return loadQiangHuaCatalog(filepath.Join(dir, "equip_prop_rand.ini"))
}

func TestQiangHuaCatalogRules(t *testing.T) {
	catalog := stubQiangHuaCatalog(t)
	// [4] 无 PropPack 不入册。
	if len(catalog.rules) != 3 {
		t.Fatalf("rules=%d, want 3 (no-pack rule dropped)", len(catalog.rules))
	}
	if got := len(catalog.byJob["sh_cf"]); got != 2 {
		t.Fatalf("sh_cf rules=%d", got)
	}
	// 默认定位：jobid 首条。
	rule := catalog.resolveRule("sh_cf", "")
	if rule == nil || rule.id != 1 || rule.cost != 1000 || rule.item != "ronghe_30001" || rule.itemNum != 3 {
		t.Fatalf("resolveRule(sh_cf,\"\")=%+v", rule)
	}
	// itemreduce 命中变体。
	if rule = catalog.resolveRule("sh_cf", "ronghe_30002"); rule == nil || rule.id != 2 {
		t.Fatalf("resolveRule itemreduce=%+v", rule)
	}
	// 融合石跨职业不命中（sh_cf 请求 ds 的石头 → 回落首条）。
	if rule = catalog.resolveRule("sh_ds", ""); rule == nil || rule.id != 3 {
		t.Fatalf("resolveRule(sh_ds)=%+v", rule)
	}
	if catalog.resolveRule("sh_xx", "") != nil {
		t.Fatalf("unknown job should resolve nil")
	}
}

func TestQiangHuaCustomMainChain(t *testing.T) {
	setWeaponRecastCatalogForTest(stubWeaponRecastCatalog(t))
	t.Cleanup(func() { setWeaponRecastCatalogForTest(nil) })
	setQiangHuaCatalogForTest(stubQiangHuaCatalog(t))
	t.Cleanup(func() { setQiangHuaCatalogForTest(nil) })

	conn := &captureMessageConnection{}
	player := &playerActor{}
	player.silver = 5000
	player.equip = []equipRow{{
		configID: "blade_test", itemType: 200, equipType: "weapon",
		slot: 22, hardiness: 100, maxHardiness: 100, propBuilt: "1,13040;1,13041",
	}}
	player.bag = []bagRow{{viewID: 3, slot: 7, configID: "ronghe_30001", itemType: 1150, amount: 5}}
	h := &itemSession{link: conn, player: player, remote: "test"}

	message := func(view, index int32, jobID string) clientCustomMessage {
		return clientCustomMessage{Values: []clientCustomValue{
			{Type: 2, Int32: clientCustomBeginQiangHua},
			{Type: 2, Int32: view}, {Type: 2, Int32: index},
			{Type: 6, Text: jobID},
		}}
	}

	// 非武器槽拒绝（21 非武器）。
	player.equip = append(player.equip, equipRow{configID: "hat_a", equipType: "hat", slot: 1})
	if handled, err := handleQiangHuaCustom(h, message(recastViewEquipWorn, 1, "sh_cf")); err != nil || !handled {
		t.Fatalf("non-weapon handled=%v err=%v", handled, err)
	}
	if player.silver != 5000 {
		t.Fatalf("silver=%d (non-weapon must not bill)", player.silver)
	}

	// 材料不足拒绝（需 3 颗只有 5？——足。先测碎银不足路径用 sh_ds 高费
	// 场景；此处直接跑主链）。
	if _, err := handleQiangHuaCustom(h, message(recastViewEquipWorn, 22, "sh_cf")); err != nil {
		t.Fatalf("main chain: %v", err)
	}
	if player.silver != 4000 {
		t.Fatalf("silver=%d, want 5000-1000", player.silver)
	}
	player.mu.Lock()
	remaining := len(player.bag)
	player.mu.Unlock()
	if remaining != 1 { // 5-3=2 叠回一行
		t.Fatalf("bag rows=%d, want 1 (5-3 stacked)", remaining)
	}
	if row, ok := player.wornItemAt(22); !ok || row.propBuilt == "1,13040;1,13041" {
		t.Fatalf("propBuilt not rolled: %q", row.propBuilt)
	}
	// 背包视图一期拒绝。
	if _, err := handleQiangHuaCustom(h, message(recastViewEquipBag, 5, "sh_cf")); err != nil {
		t.Fatalf("bag view: %v", err)
	}
	if frames := conn.Frames(); len(frames) == 0 {
		t.Fatalf("no frames pushed (currency + wear refresh)")
	}
}

func TestRollQiangHuaPacks(t *testing.T) {
	if got := rollQiangHuaPacks(nil, 3); got != nil {
		t.Fatalf("empty pool=%v", got)
	}
	pool := []int{1, 2, 3}
	got := rollQiangHuaPacks(pool, 5)
	if len(got) != 3 {
		t.Fatalf("capped roll=%v", got)
	}
	seen := map[int]bool{}
	for _, id := range got {
		if seen[id] {
			t.Fatalf("duplicate %d in %v", id, got)
		}
		seen[id] = true
	}
}

// 洗炼二期：背包装备路径（00020 affixes 列）。
func TestQiangHuaBagPath(t *testing.T) {
	setWeaponRecastCatalogForTest(stubWeaponRecastCatalog(t))
	t.Cleanup(func() { setWeaponRecastCatalogForTest(nil) })
	setQiangHuaCatalogForTest(stubQiangHuaCatalog(t))
	t.Cleanup(func() { setQiangHuaCatalogForTest(nil) })

	conn := &captureMessageConnection{}
	player := &playerActor{}
	player.silver = 5000
	player.bag = []bagRow{
		{viewID: 2, slot: 5, configID: "blade_test", itemType: 200, equipType: "weapon", amount: 1, propBuilt: "1,13040"},
		{viewID: 3, slot: 7, configID: "ronghe_30001", itemType: 1150, amount: 5},
	}
	h := &itemSession{link: conn, player: player, remote: "test"}

	message := func(view, index int32, jobID string) clientCustomMessage {
		return clientCustomMessage{Values: []clientCustomValue{
			{Type: 2, Int32: clientCustomBeginQiangHua},
			{Type: 2, Int32: view}, {Type: 2, Int32: index},
			{Type: 6, Text: jobID},
		}}
	}

	// 背包武器强化主链：碎银/融合石扣减 + 背包行词条重掷。
	if _, err := handleQiangHuaCustom(h, message(recastViewEquipBag, 5, "sh_cf")); err != nil {
		t.Fatalf("bag qianghua: %v", err)
	}
	if player.silver != 4000 {
		t.Fatalf("silver=%d, want 4000", player.silver)
	}
	player.mu.Lock()
	built := player.bag[0].propBuilt
	stone := player.bag[1].amount
	player.mu.Unlock()
	if built == "" || built == "1,13040" {
		t.Fatalf("bag propBuilt not rolled: %q", built)
	}
	if stone != 2 {
		t.Fatalf("stone=%d, want 5-3=2", stone)
	}

	// 背包非武器拒绝（equipType 过滤）。
	player.bag = append(player.bag, bagRow{viewID: 2, slot: 8, configID: "hat_a", equipType: "hat", amount: 1})
	player.silver = 5000
	if handled, _ := handleQiangHuaCustom(h, message(recastViewEquipBag, 8, "sh_cf")); !handled {
		t.Fatal("handled=false")
	}
	if player.silver != 5000 {
		t.Fatalf("non-weapon bag target billed: %d", player.silver)
	}

	// 空格拒绝。
	if handled, _ := handleQiangHuaCustom(h, message(recastViewEquipBag, 99, "sh_cf")); !handled {
		t.Fatal("handled=false")
	}
}
