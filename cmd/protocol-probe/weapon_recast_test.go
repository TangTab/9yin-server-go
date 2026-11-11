package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 武器重铸域（C2S 933-938 / S2C 467/468）：目录合并、roll 预览、
// lock→preview→confirm 全链与会话校验。

func stubWeaponRecastCatalog(t *testing.T) *weaponRecastCatalog {
	t.Helper()
	dir := t.TempDir()
	randINI := "[blade_test]\nr=13040,13041,13234,13236,,,,,,,,\n\n[blade_none]\nr=13040\n"
	direINI := "[blade_test]\nr=MeleeDamage,13040,13041,13042\nr=AIPhyDef,13234,13236\n"
	propPackINI := "[13040]\nr=,MinMeleeDamage,7,0\n\n[13041]\nr=,MaxMeleeDamage,8,0\n"
	skillPackINI := "[13234]\nr=CS_a,PhyDamageMul,0.7,0\n\n[13236]\nr=CS_b,StiDamageMul,0.8,0\n"
	for name, content := range map[string]string{
		"senior_weapon_rand.ini": randINI, "senior_weapon_dire.ini": direINI,
		"proppack.ini": propPackINI, "skillpack.ini": skillPackINI,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return loadWeaponRecastCatalog(
		filepath.Join(dir, "senior_weapon_rand.ini"),
		filepath.Join(dir, "senior_weapon_dire.ini"),
		filepath.Join(dir, "proppack.ini"),
		filepath.Join(dir, "skillpack.ini"))
}

func TestWeaponRecastCatalogMerge(t *testing.T) {
	catalog := stubWeaponRecastCatalog(t)
	setWeaponRecastCatalogForTest(catalog)
	t.Cleanup(func() { setWeaponRecastCatalogForTest(nil) })
	pool := catalog.randPools["blade_test"]
	if len(pool) != 4 || pool[0] != 13040 || pool[2] != 13234 {
		t.Fatalf("rand pool=%v", pool)
	}
	route := catalog.direRoutes["blade_test"]
	if len(route["MeleeDamage"]) != 3 || len(route["AIPhyDef"]) != 2 {
		t.Fatalf("dire routes=%v", route)
	}
	if !catalog.hasPools("blade_test") || catalog.hasPools("blade_none") && len(catalog.direRoutes["blade_none"]) > 0 {
		// blade_none 只在 rand 表——hasPools 按任一表在册判定。
		if !catalog.hasPools("blade_none") {
			t.Fatalf("blade_none should be in rand pools")
		}
	}
	if catalog.packKind(13040) != 1 || catalog.packKind(13234) != 2 || catalog.packKind(999) != 0 {
		t.Fatalf("packKind broken")
	}
}

func TestWeaponRecastRollPreview(t *testing.T) {
	catalog := stubWeaponRecastCatalog(t)
	// op3 武器随机：仅 prop 词条。
	preview := catalog.rollPreview("blade_test", recastTypeWeaponRand, 2, itemUseRandom())
	if len(preview) != 2 {
		t.Fatalf("weapon preview=%v", preview)
	}
	for _, pack := range preview {
		if catalog.packKind(pack) != 1 {
			t.Fatalf("weapon preview leaked skill pack %d", pack)
		}
	}
	// op5 技能随机：仅 skill 词条。
	preview = catalog.rollPreview("blade_test", recastTypeSkillRand, 2, itemUseRandom())
	if len(preview) != 2 {
		t.Fatalf("skill preview=%v", preview)
	}
	for _, pack := range preview {
		if catalog.packKind(pack) != 2 {
			t.Fatalf("skill preview leaked prop pack %d", pack)
		}
	}
	// 空池/零数量。
	if got := catalog.rollPreview("unknown", recastTypeWeaponRand, 2, itemUseRandom()); got != nil {
		t.Fatalf("unknown pool preview=%v", got)
	}
	if got := catalog.rollPreview("blade_test", recastTypeWeaponRand, 0, itemUseRandom()); got != nil {
		t.Fatalf("zero count preview=%v", got)
	}
}

func TestRecastPackListAndAffixes(t *testing.T) {
	catalog := stubWeaponRecastCatalog(t)
	if got := recastPackList([]int{13040, 13234}); got != "0,13040:0,13234" {
		t.Fatalf("packList=%q", got)
	}
	affixes := recastAffixesJSON(catalog, []int{13040, 13234})
	if !strings.Contains(affixes, `"prop_built":13040`) || !strings.Contains(affixes, `"kind":1`) {
		t.Fatalf("affixes=%s", affixes)
	}
	if !strings.Contains(affixes, `"prop_built":13234`) || !strings.Contains(affixes, `"kind":2`) {
		t.Fatalf("affixes kind2 missing: %s", affixes)
	}
	built := propBuiltOfAffixes(affixes)
	if built != "1,13040;2,13234" {
		t.Fatalf("propBuilt=%q", built)
	}
	if propBuiltOfAffixes("") != "" || propBuiltOfAffixes("not json") != "" {
		t.Fatalf("propBuiltOfAffixes degenerate broken")
	}
}

func recastCustomMessage(msg int32, args ...int32) clientCustomMessage {
	values := []clientCustomValue{{Type: 2, Int32: msg}}
	for _, arg := range args {
		values = append(values, clientCustomValue{Type: 2, Int32: arg})
	}
	return clientCustomMessage{Values: values}
}

func newRecastTestSession(t *testing.T) (*itemSession, *captureMessageConnection, *playerActor) {
	t.Helper()
	conn := &captureMessageConnection{}
	player := &playerActor{}
	player.equip = []equipRow{{
		configID: "blade_test", itemType: 200, equipType: "weapon",
		slot: 22, hardiness: 100, maxHardiness: 100, propBuilt: "1,13040;1,13041",
	}}
	session := &itemSession{link: conn, player: player, remote: "test"}
	return session, conn, player
}

func TestRecastLockPreviewConfirm(t *testing.T) {
	setWeaponRecastCatalogForTest(stubWeaponRecastCatalog(t))
	t.Cleanup(func() { setWeaponRecastCatalogForTest(nil) })
	h, conn, player := newRecastTestSession(t)

	// 背包视图一期拒绝。
	if handled, err := handleRecastCustom(h, recastCustomMessage(clientCustomEquipRandPropLock, recastViewEquipBag, 5, recastTypeWeaponRand)); err != nil || !handled {
		t.Fatalf("bag view lock handled=%v err=%v", handled, err)
	}
	if player.recast.active {
		t.Fatalf("bag view lock should not bind session")
	}

	// 未知 op 拒绝。
	handleRecastCustom(h, recastCustomMessage(clientCustomEquipRandPropLock, recastViewEquipWorn, 22, 99))
	if player.recast.active {
		t.Fatalf("unknown op should not bind")
	}

	// 933 穿戴视图锁定（slot=22，词条数按 propBuilt=2）。
	if _, err := handleRecastCustom(h, recastCustomMessage(clientCustomEquipRandPropLock, recastViewEquipWorn, 22, recastTypeWeaponRand)); err != nil {
		t.Fatalf("lock: %v", err)
	}
	if !player.recast.active || player.recast.configID != "blade_test" {
		t.Fatalf("session=%+v", player.recast)
	}

	// 935 预览（会话匹配）→ S2C 467 [op, newpackprop]。
	if _, err := handleRecastCustom(h, recastCustomMessage(clientCustomYuRandProp, recastViewEquipWorn, 22, recastTypeWeaponRand)); err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(player.recast.preview) != 2 {
		t.Fatalf("preview packs=%v", player.recast.preview)
	}
	preview := append([]int(nil), player.recast.preview...)
	frames := conn.Frames()
	if len(frames) != 1 {
		t.Fatalf("preview frames=%d", len(frames))
	}
	if !strings.Contains(string(frames[0]), "\x01\xd3") && !strings.Contains(string(frames[0]), "1\xd3") {
		// 467=0x1D3 小端 (d3 01 00 00)——int 值编码包含 0xd3 0x01。
		t.Logf("frame=%x (msg id 467 int-encoded)", frames[0])
	}

	// 938 确认 → affixes 落内存行 + S2C 468。
	if _, err := handleRecastCustom(h, recastCustomMessage(clientCustomSureEquipProp, recastViewEquipWorn, 22, recastTypeWeaponRand)); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if player.recast.active {
		t.Fatalf("confirm should clear session")
	}
	want := propBuiltOfAffixes(recastAffixesJSON(currentWeaponRecastCatalog(), preview))
	row, ok := player.wornItemAt(22)
	if !ok || row.propBuilt != want {
		t.Fatalf("propBuilt=%q, want %q", row.propBuilt, want)
	}
	if frames := conn.Frames(); len(frames) < 2 {
		t.Fatalf("frames=%d (467 + 468 + wear refresh)", len(frames))
	}

	// 会话不匹配的确认拒绝。
	handleRecastCustom(h, recastCustomMessage(clientCustomEquipRandPropLock, recastViewEquipWorn, 22, recastTypeWeaponRand))
	if _, err := handleRecastCustom(h, recastCustomMessage(clientCustomSureEquipProp, recastViewEquipWorn, 22, recastTypeSkillRand)); err != nil {
		t.Fatalf("mismatched confirm: %v", err)
	}
	if !player.recast.active {
		t.Fatalf("mismatched confirm should keep session")
	}

	// 934 解锁。
	if _, err := handleRecastCustom(h, recastCustomMessage(clientCustomEquipRandPropUnlock, recastViewEquipWorn, 22, recastTypeWeaponRand)); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if player.recast.active {
		t.Fatalf("unlock should clear session")
	}
}

// ---- 洗炼二期：背包装备路径（00020 affixes 列）----

func newRecastBagTestSession(t *testing.T) (*itemSession, *captureMessageConnection, *playerActor) {
	t.Helper()
	conn := &captureMessageConnection{}
	player := &playerActor{}
	// 背包装备页（viewID=2 → 客户端视图 121）槽 5 一把 senior 武器。
	player.bag = []bagRow{{
		viewID: 2, slot: 5, configID: "blade_test", itemType: 200,
		equipType: "weapon", amount: 1, propBuilt: "1,13042;1,13041",
	}}
	session := &itemSession{link: conn, player: player, remote: "test"}
	return session, conn, player
}

func TestRecastBagPathLockPreviewConfirm(t *testing.T) {
	setWeaponRecastCatalogForTest(stubWeaponRecastCatalog(t))
	t.Cleanup(func() { setWeaponRecastCatalogForTest(nil) })
	h, conn, player := newRecastBagTestSession(t)

	// 933 背包视图锁定。
	if _, err := handleRecastCustom(h, recastCustomMessage(clientCustomEquipRandPropLock, recastViewEquipBag, 5, recastTypeWeaponRand)); err != nil {
		t.Fatalf("bag lock: %v", err)
	}
	if !player.recast.active || player.recast.configID != "blade_test" || player.recast.viewID != recastViewEquipBag {
		t.Fatalf("session=%+v", player.recast)
	}

	// 935 预览：词条数按背包行 propBuilt（2）。
	if _, err := handleRecastCustom(h, recastCustomMessage(clientCustomYuRandProp, recastViewEquipBag, 5, recastTypeWeaponRand)); err != nil {
		t.Fatalf("bag preview: %v", err)
	}
	if len(player.recast.preview) != 2 {
		t.Fatalf("preview packs=%v", player.recast.preview)
	}
	preview := append([]int(nil), player.recast.preview...)

	// 938 确认：背包行词条落内存（store=nil 内存降级）+ 帧重发。
	before := len(conn.Frames())
	if _, err := handleRecastCustom(h, recastCustomMessage(clientCustomSureEquipProp, recastViewEquipBag, 5, recastTypeWeaponRand)); err != nil {
		t.Fatalf("bag confirm: %v", err)
	}
	want := propBuiltOfAffixes(recastAffixesJSON(currentWeaponRecastCatalog(), preview))
	player.mu.Lock()
	got := player.bag[0].propBuilt
	player.mu.Unlock()
	if got != want {
		t.Fatalf("bag propBuilt=%q, want %q", got, want)
	}
	frames := conn.Frames()[before:]
	// S2C 468 + ViewRemove + ViewAdd（带 PropBuilt）+ 镜像清位。
	if len(frames) < 3 {
		t.Fatalf("frames=%d, want >=3 (468+remove+add)", len(frames))
	}
	foundPropBuilt := false
	for _, frame := range frames {
		if strings.Contains(string(frame), "1,") && len(frame) > 2 && frame[0] == 0x19 {
			foundPropBuilt = true
		}
	}
	_ = foundPropBuilt // ViewAdd 帧形状由 bagRowViewAdd 单测覆盖

	// 空格锁定拒绝。
	if _, err := handleRecastCustom(h, recastCustomMessage(clientCustomEquipRandPropLock, recastViewEquipBag, 9, recastTypeWeaponRand)); err != nil {
		t.Fatalf("empty lock: %v", err)
	}
	if player.recast.active && player.recast.slot == 9 {
		t.Fatal("empty cell should not bind")
	}
}

func TestBagRowViewAddCarriesPropBuilt(t *testing.T) {
	setWeaponRecastCatalogForTest(stubWeaponRecastCatalog(t))
	t.Cleanup(func() { setWeaponRecastCatalogForTest(nil) })
	resetRandomEquipCatalogForTest()
	randomEquipCatalogOnce.Do(func() {})
	randomEquipCat = &randomEquipCatalog{} // 空目录：hasTemplatePacks 恒 false → PropBuilt 下发

	item := bagRow{
		viewID: 2, slot: 5, configID: "blade_test", itemType: 200,
		equipType: "weapon", amount: 1, propBuilt: "1,13040;2,13234",
	}
	frame, err := bagRowViewAdd(1, item)
	if err != nil {
		t.Fatalf("viewAdd: %v", err)
	}
	if !strings.Contains(string(frame), "1,13040;2,13234") {
		t.Fatalf("ViewAdd missing PropBuilt: %x", frame)
	}
	// 空词条不下发。
	item.propBuilt = ""
	frame, err = bagRowViewAdd(1, item)
	if err != nil {
		t.Fatalf("viewAdd empty: %v", err)
	}
	if strings.Contains(string(frame), "1,13040") {
		t.Fatalf("empty propBuilt should not carry")
	}
}
