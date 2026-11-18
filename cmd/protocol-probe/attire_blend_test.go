package main

// 梳妆阁图纸移植回归：目录装载 / blend_collect_rec 行帧 / C2S 692 保存 /
// 逐槽模型覆盖 / 视图图标替换对 / 存档 SQL。

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/local/9yin-go-server/internal/role"
)

// injectStubAttireCatalog 注入测试图纸目录（真实资源缺席时目录为空，
// 单测不依赖部署资源树）。
func injectStubAttireCatalog(t *testing.T) {
	t.Helper()
	oldOnce, oldLoaded := attireCatalogOnce, attireCatalogLoaded
	attireCatalogOnce = func() *attireCatalogData {
		return attireCatalogFromTable(iniTable{
			"Tab_b_head_180_b": {
				{key: "ItemType", value: "601"},
				{key: "ArtPack", value: "14233"},
			},
			"Tab_b_cloth_001": {
				{key: "ItemType", value: "602"},
				{key: "ArtPack", value: "20001"},
			},
			"Tab_g_cloth_001": {
				{key: "ItemType", value: "602"},
				{key: "ArtPack", value: "20002"},
			},
			"Tab_b_pants_001": {
				{key: "ItemType", value: "603"},
				{key: "ArtPack", value: "20003"},
			},
			"Tab_b_shoes_001": {
				{key: "ItemType", value: "604"},
				{key: "ArtPack", value: "20004"},
			},
			// 过滤项：非图纸 ItemType / 缺 ArtPack / 非 Tab 段。
			"Tab_b_badtype": {{key: "ItemType", value: "700"}, {key: "ArtPack", value: "9"}},
			"Tab_b_noart":   {{key: "ItemType", value: "601"}},
			"card_item_1":   {{key: "ItemType", value: "601"}, {key: "ArtPack", value: "9"}},
		})
	}
	attireCatalogLoaded = nil
	t.Cleanup(func() { attireCatalogOnce, attireCatalogLoaded = oldOnce, oldLoaded })
}

// injectStubBlueprintModels 往装备解析器补图纸数字美术包（在
// injectStubAppearanceResolver 之后叠加，不动既有装备链表值）。
func injectStubBlueprintModels(t *testing.T) {
	t.Helper()
	old := equipmentModels
	merged := *old
	merged.models = make(map[string]itemModels, len(old.models)+5)
	for key, models := range old.models {
		merged.models[key] = models
	}
	merged.models["14233"] = itemModels{male: `obj\b\bp_helmet`, female: `obj\g\gp_helmet`}
	merged.models["20001"] = itemModels{male: `obj\b\bp_cloth`, female: `obj\g\gp_cloth`}
	merged.models["20002"] = itemModels{male: `obj\b\g_cloth`, female: `obj\g\gg_cloth`}
	merged.models["20003"] = itemModels{male: `obj\b\bp_pants`}
	merged.models["20004"] = itemModels{male: `obj\b\bp_shoes`}
	equipmentModels = &merged
	t.Cleanup(func() { equipmentModels = old })
}

// 目录装载：Tab_b_/Tab_g_ 前缀定性别、ItemType 601-604 过滤、ArtPack
// 必填、order 升序、大小写不敏感检索。
func TestAttireCatalogFromFixtureTable(t *testing.T) {
	injectStubAttireCatalog(t)
	catalog := attireCatalogOf()
	if len(catalog.order) != 5 {
		t.Fatalf("catalog size=%d, want 5 (badtype/noart/non-tab filtered): %+v", len(catalog.order), catalog.order)
	}
	if got := catalog.order; got[0] != "tab_b_cloth_001" || got[4] != "tab_g_cloth_001" {
		t.Fatalf("order=%v, want 升序", got)
	}
	head, ok := catalog.lookup("Tab_B_Head_180_B") // 大小写不敏感
	if !ok || !head.male || head.itemType != 601 || head.artPack != "14233" {
		t.Fatalf("head entry=%+v ok=%t", head, ok)
	}
	female, ok := catalog.lookup("tab_g_cloth_001")
	if !ok || female.male {
		t.Fatalf("female entry=%+v ok=%t", female, ok)
	}
	for itemType, wantSlot := range map[int32]int32{601: slotHat, 602: slotCloth, 603: slotPants, 604: slotShoes} {
		if got := blendSlotOf(itemType); got != wantSlot {
			t.Fatalf("blendSlotOf(%d)=%d, want %d", itemType, got, wantSlot)
		}
	}
	if blendSlotOf(700) != 0 || isValidBlendSlot(slotSuit) {
		t.Fatal("blendSlotOf 过滤失效")
	}
}

// blend_collect_rec 全解锁行帧：目录序逐行、0x11 场景对象形态、
// index 179、[string,1,0,byte 1] 列布局（末列 ACTIVE 是字节）。
func TestBlendCollectRecordPush(t *testing.T) {
	injectStubAttireCatalog(t)
	conn := &captureMessageConnection{}
	if err := pushBlendCollectRows(conn); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	catalog := attireCatalogOf()
	if len(frames) != len(catalog.order) {
		t.Fatalf("frames=%d, want %d", len(frames), len(catalog.order))
	}
	first := frames[0]
	if first[0] != 0x11 || first[1] != 0 {
		t.Fatalf("header=% x", first[:2])
	}
	if binary.LittleEndian.Uint32(first[2:]) != playerObjectID ||
		binary.LittleEndian.Uint32(first[6:]) != playerOwnerID ||
		binary.LittleEndian.Uint16(first[10:]) != recordBlendCollect ||
		binary.LittleEndian.Uint16(first[12:]) != 0 || binary.LittleEndian.Uint16(first[14:]) != 1 {
		t.Fatalf("blend row addressing=% x", first[:16])
	}
	// 列布局：string(configID) + int(1) + int(0) + byte(1)。
	configID := catalog.order[0]
	wantLen := 16 + 4 + len(configID) + 1 + 4 + 4 + 1
	if len(first) != wantLen {
		t.Fatalf("row len=%d, want %d", len(first), wantLen)
	}
	if got := string(first[20 : 20+len(configID)]); got != configID {
		t.Fatalf("configID cell=%q, want %q", got, configID)
	}
	if own := binary.LittleEndian.Uint32(first[16+4+len(configID)+1:]); own != 1 {
		t.Fatalf("OWNNUM=%d", own)
	}
	if last := first[len(first)-1]; last != 1 {
		t.Fatalf("ACTIVE byte=%d, want 1", last)
	}
}

// 692 保存：合法对落账（绑定 key=该穿着件 UniqueID 口径）+ 0x10 外观帧
// + sysinfo；性别不匹配/未知 uid 的对忽略；部件不符忽略；空载荷静默。
func TestAttireBlendParses692(t *testing.T) {
	injectStubAttireCatalog(t)
	injectStubAppearanceResolver(t)
	injectStubBlueprintModels(t)
	player := newPlayerActor("tester", 0)
	player.roleID = 1
	// 男号穿一件自带 uid 的上衣（uid 非空 → 绑定 key 原样）。
	worn := equipRow{configID: "cloth_gear", equipType: "Cloth", slot: slotCloth, uid: "equip-cloth-1"}
	player.replaceEquipSlot(worn)
	conn := &captureMessageConnection{}
	session := newTestAppearanceSession(conn, player)

	// 一对合法（男上衣图纸→槽3穿着件）+ 一对性别不匹配 + 一对未知 uid。
	info := "Tab_b_cloth_001,equip-cloth-1,Tab_g_cloth_001,equip-cloth-1,Tab_b_head_180_b,nowhere"
	msg := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomAttireEquipBlend},
		{Type: 2, Int32: 2},
		{Type: 6, Text: info},
	}}
	handled, err := handleAttireEquipBlendCustom(session, msg)
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	snapshot := player.blendSnapshot()
	if len(snapshot) != 1 || snapshot["equip-cloth-1"] != "Tab_b_cloth_001" {
		t.Fatalf("snapshot=%v, want 单条合法绑定", snapshot)
	}
	hasAppearance, hasSysinfo := false, false
	for _, frame := range conn.Frames() {
		if frame[0] == 0x10 {
			hasAppearance = true
		}
		// 0x1E 帧布局：opcode + u16 值数 + 首值 tag(2=int32) + 377。
		if frame[0] == 0x1E && len(frame) >= 8 && frame[3] == 2 &&
			binary.LittleEndian.Uint32(frame[4:]) == 377 {
			hasSysinfo = true
		}
	}
	if !hasAppearance {
		t.Fatal("保存后未推 0x10 外观帧")
	}
	if !hasSysinfo {
		t.Fatal("保存后未发 sysinfo 377 成功提示")
	}

	// 部件不符（帽图纸绑到衣槽穿着件）整对忽略。
	conn.resetFrames()
	msg.Values[2].Text = "Tab_b_head_180_b,equip-cloth-1"
	if _, err := handleAttireEquipBlendCustom(session, msg); err != nil {
		t.Fatal(err)
	}
	if snapshot := player.blendSnapshot(); len(snapshot) != 1 {
		t.Fatalf("部件不符未被忽略: %v", snapshot)
	}

	// 空载荷/残对拒绝（零改动零帧）。
	conn.resetFrames()
	msg.Values[2].Text = ";1;0"
	if _, err := handleAttireEquipBlendCustom(session, msg); err != nil {
		t.Fatal(err)
	}
	if len(conn.Frames()) != 0 || len(player.blendSnapshot()) != 1 {
		t.Fatal("空载荷应静默拒绝")
	}
}

// 逐槽模型覆盖：穿着被绑定件才替换（卸下即恢复）；身份装套装分件对
// 绑定部件让位（图纸 > 套装）；女号取 FemaleModel。
func TestBlendOverlayReplacesWornModel(t *testing.T) {
	injectStubAttireCatalog(t)
	injectStubAppearanceResolver(t)
	injectStubBlueprintModels(t)
	roleID := role.RoleID(1)
	hat := equipRow{configID: "helmet_gear", equipType: "Hat", slot: slotHat, uid: "equip-hat-1"}
	equips := append(stubAppearanceEquips(), hat)
	blends := map[string]string{"equip-hat-1": "tab_b_head_180_b"}

	overlay := blendOverlayFor(roleID, blends, 0, equips)
	if got := overlay[slotHat]; got != `obj\b\bp_helmet` {
		t.Fatalf("hat overlay=%q, want 图纸男模型", got)
	}
	if len(overlay) != 1 {
		t.Fatalf("overlay=%v, want 仅帽槽", overlay)
	}
	female := blendOverlayFor(roleID, blends, 1, equips)
	if got := female[slotHat]; got != `obj\g\gp_helmet` {
		t.Fatalf("female overlay=%q, want 图纸女模型", got)
	}
	// 卸下即恢复：穿着集不含该件 → 零覆盖。
	if overlay := blendOverlayFor(roleID, blends, 0, []equipRow{}); overlay != nil {
		t.Fatalf("未穿着被绑定件 overlay=%v, want nil", overlay)
	}
	// 身份装档：套装帽分件让位给图纸，其余分件照常覆盖。
	finals := finalModels(showEquipIdentity, 0, equips, overlay)
	if finals[partHat] != `obj\b\bp_helmet` {
		t.Fatalf("identity hat=%q, want 图纸模型（套装让位）", finals[partHat])
	}
	if finals[partPants] != `obj\m\suit_pants` {
		t.Fatalf("identity pants=%q, want 套装分件照常", finals[partPants])
	}
	// 江湖装档：绑定槽直接图纸模型。
	gear := finalModels(showEquipGear, 0, equips, overlay)
	if gear[partHat] != `obj\b\bp_helmet` {
		t.Fatalf("gear hat=%q, want 图纸模型", gear[partHat])
	}
}

// 视图图标替换：绑定行的 Photo 撤发、ReplacePack(int)/ReplaceID(string)
// 补发；未绑定行为不变（Photo 照发）。
func TestEquipViewPropsUseBlendReplaceID(t *testing.T) {
	injectStubAttireCatalog(t)
	old := equipmentModels
	equipmentModels = &equipmentResolver{
		statics: map[string]itemStatic{
			"cloth_gear": {equipType: "Cloth", photo: `icon\equip\gear.png`},
		},
	}
	t.Cleanup(func() { equipmentModels = old })

	roleID := role.RoleID(665)
	equip := equipRow{configID: "cloth_gear", equipType: "Cloth", slot: slotCloth, uid: "equip-cloth-1", itemType: 60702}
	plain, err := equipmentViewAdd(roleID, equip, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !viewFrameContainsProp(plain, propItemPhoto) {
		t.Fatal("未绑定行应照发 Photo")
	}
	if viewFrameContainsProp(plain, propReplacePack) {
		t.Fatal("未绑定行不应发 ReplacePack")
	}

	blended, err := equipmentViewAdd(roleID, equip, map[string]string{"equip-cloth-1": "tab_b_head_180_b"})
	if err != nil {
		t.Fatal(err)
	}
	if viewFrameContainsProp(blended, propItemPhoto) {
		t.Fatal("绑定行须撤 Photo（直发 Photo 会遮住 ReplacePack）")
	}
	if !viewFrameContainsProp(blended, propReplacePack) || !viewFrameContainsProp(blended, propReplaceID) {
		t.Fatal("绑定行缺 ReplacePack/ReplaceID")
	}
	if !strings.Contains(string(blended), "Tab_b_head_180_b") {
		t.Fatal("ReplaceID 值应为图纸 configID（原大小写，官方抓包同形）")
	}

	// 背包行同链：绑定 uid 键口径 = itemUID("bag")。
	bag := bagRow{slot: 7, configID: "cloth_gear", itemType: 60702, viewID: 2, equipType: "Cloth", uid: "bag-cloth-1"}
	bagBlended, err := bagRowViewAdd(roleID, bag, map[string]string{"bag-cloth-1": "tab_b_cloth_001"})
	if err != nil {
		t.Fatal(err)
	}
	if !viewFrameContainsProp(bagBlended, propReplaceID) {
		t.Fatal("背包绑定行缺 ReplaceID")
	}
}

// viewFrameContainsProp 粗查 0x18 行帧是否带某属性下标（u16 下标扫描，
// 值区不做解码——存在性足够本测试）。
func viewFrameContainsProp(frame []byte, propIndex uint16) bool {
	if len(frame) < 7 || frame[0] != 0x18 {
		return false
	}
	count := int(binary.LittleEndian.Uint16(frame[5:]))
	offset := 7
	for i := 0; i < count; i++ {
		if offset+2 > len(frame) {
			return false
		}
		index := binary.LittleEndian.Uint16(frame[offset:])
		if index == propIndex {
			return true
		}
		offset += 2
		// 按官方线型跳值区。
		switch index {
		case propItemLevel, propRecoverFlag: // byte
			offset += 1
		case propBeginPos, propEndPos: // word
			offset += 2
		case propItemPhoto, propPropBuilt, propEquipType, propItemConfigID, propUniqueID, propReplaceID: // string
			if offset+4 > len(frame) {
				return false
			}
			offset += 4 + int(binary.LittleEndian.Uint32(frame[offset:]))
		default: // int32 / 其余按 u32
			offset += 4
		}
	}
	return false
}

// role_attire_blends 存档 SQL（skinDBStore 同款 upsert/降级口径）。
func TestAttireBlendStoreSQL(t *testing.T) {
	mock, stores := newAppearanceMockStores(t)
	mock.ExpectExec(`INSERT INTO role_attire_blends \(role_id, blends, version, updated_at\)`+
		` VALUES \(\?, CAST\(\? AS JSON\), 1, UTC_TIMESTAMP\(6\)\)`+
		` ON DUPLICATE KEY UPDATE blends = VALUES\(blends\),`+
		` version = version \+ 1, updated_at = UTC_TIMESTAMP\(6\)`).
		WithArgs(customConfigTestRoleID, `{"u1":"tab_b_cloth_001"}`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := stores.blends.Save(customConfigTestRoleID, map[string]string{"u1": "tab_b_cloth_001"}); err != nil {
		t.Fatal(err)
	}

	mock, stores = newAppearanceMockStores(t)
	mock.ExpectQuery(`SELECT blends FROM role_attire_blends WHERE role_id = \?`).
		WithArgs(customConfigTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"blends"}).AddRow(`{"u1":"tab_b_cloth_001"}`))
	loaded, ok := stores.blends.Load(customConfigTestRoleID)
	if !ok || loaded["u1"] != "tab_b_cloth_001" {
		t.Fatalf("load=(%v,%t)", loaded, ok)
	}

	// 空行/坏 JSON 降级为无存档。
	mock, stores = newAppearanceMockStores(t)
	mock.ExpectQuery(`SELECT blends FROM role_attire_blends WHERE role_id = \?`).
		WithArgs(customConfigTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"blends"}))
	if loaded, ok := stores.blends.Load(customConfigTestRoleID); ok || loaded != nil {
		t.Fatalf("no-row load=(%v,%t), want (nil,false)", loaded, ok)
	}
	if parsed := parseBlends("not-json"); parsed != nil {
		t.Fatalf("parseBlends bad json=%v, want nil", parsed)
	}
	if got := marshalBlends(nil); got != "{}" {
		t.Fatalf("marshalBlends nil=%q, want {}", got)
	}

	// nil-store 降级：Save 无操作、Load 无存档（httptest 同款内存态）。
	var nilStore *blendDBStore
	if err := nilStore.Save(customConfigTestRoleID, nil); err != nil {
		t.Fatal(err)
	}
	if _, ok := nilStore.Load(customConfigTestRoleID); ok {
		t.Fatal("nil store load should degrade")
	}
}
