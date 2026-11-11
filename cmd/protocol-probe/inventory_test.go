package main

import (
	"context"
	"database/sql"
	"encoding/binary"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/local/9yin-go-server/internal/role"
)

// testContext 供 DB 层测试使用的短超时上下文。
func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// 镜像 Java InventoryMessages/RoleInventoryStore 口径：视图路由、UID
// 派生、17 视图容器、扩容锚点、纸娃娃/背包成对行、DB 富化与缺列降级。
// 物品静态表桩与 RoleVisualSexTest 同构（仓库内无 share/item 真表，
// 部署机由 NINEYIN_SERVER_ROOT 解析到 F 盘真表）。
func injectStubInventoryEquipment(t *testing.T) {
	t.Helper()
	old := equipmentModels
	equipmentModels = &equipmentResolver{
		artByItem: map[string]string{"cloth_g0003": "7", "sword_test": "9"},
		models:    map[string]itemModels{"7": {male: `obj\m\cloth`, female: `obj\f\cloth`}},
		statics: map[string]itemStatic{
			"sword_test": {photo: `icon\item\sword.png`, colorLevel: 3, level: 20,
				maxBuildCount: 4, effectPackID: 88, equipType: "Weapon"},
			"cloth_g0003": {equipType: "Cloth"},
		},
		actionSets: map[string]string{"9": "2h"},
	}
	t.Cleanup(func() { equipmentModels = old })
}

func TestBagViewOfRoutesCategories(t *testing.T) {
	cases := []struct {
		viewID int32
		want   uint16
	}{
		{1, viewToolPage}, {2, viewEquipBag}, {3, viewMaterialPage}, {4, viewTaskPage}, {0, viewToolPage}, {9, viewToolPage},
	}
	for _, c := range cases {
		if got := bagViewOf(c.viewID); got != c.want {
			t.Fatalf("bagViewOf(%d)=%d, want %d", c.viewID, got, c.want)
		}
	}
	if got := bagMirrorOf(viewEquipBag); got != 176 {
		t.Fatalf("bagMirrorOf(121)=%d, want 176", got)
	}
	if got := bagMirrorOf(viewDoll); got != viewDoll {
		t.Fatalf("bagMirrorOf(1)=%d, want 原视图", got)
	}
}

func TestItemUIDPrecedence(t *testing.T) {
	if got := itemUID(618, "bag", "3b9826399dff6fd9", 11, 22); got != "3b9826399dff6fd9" {
		t.Fatalf("列值优先: %q", got)
	}
	derived := itemUID(618, "equip", "", 3, 1)
	if len(derived) != 16 {
		t.Fatalf("seq 派生长度=%d, want 16 hex", len(derived))
	}
	if again := itemUID(618, "equip", "", 3, 99); again != derived {
		t.Fatalf("seq 派生应稳定: %q vs %q", derived, again)
	}
	if bySlot := itemUID(618, "bag", "", 0, 5); bySlot == derived || len(bySlot) != 16 {
		t.Fatalf("槽位兜底方言异常: %q", bySlot)
	}
}

func TestPropBuiltOfAffixesJSON(t *testing.T) {
	cases := []struct {
		json string
		want string
	}{
		{`[{"pack_rec":0,"prop_built":105},{"pack_rec":1,"prop_built":77}]`, "1,105;1,77"},
		{`[{"pack_rec":0,"prop_built":9}]`, "1,9"},
		{``, ""},
		{`not-json`, ""},
	}
	for _, c := range cases {
		nullString := sql.NullString{String: c.json, Valid: c.json != ""}
		if got := propBuiltOf(nullString); got != c.want {
			t.Fatalf("propBuiltOf(%q)=%q, want %q", c.json, got, c.want)
		}
	}
}

func TestBagViewCreateFramesLayout(t *testing.T) {
	frames := bagViewCreateFrames()
	if len(frames) != 17 {
		t.Fatalf("视图容器数=%d, want 17（8 主页+8 扩展+纸娃娃）", len(frames))
	}
	for i, frame := range frames {
		if frame[0] != 0x15 {
			t.Fatalf("frame[%d] opcode=%#x, want 0x15", i, frame[0])
		}
		viewID := binary.LittleEndian.Uint16(frame[1:])
		wantCap := uint16(2)
		wantProps := uint16(0)
		if i < 8 {
			wantCap, wantProps = bagSlotMax, 1
		} else if i == 16 {
			wantCap = dollSlotMax
		}
		if got := binary.LittleEndian.Uint16(frame[3:]); got != wantCap {
			t.Fatalf("视图 %d capacity=%d, want %d", viewID, got, wantCap)
		}
		if got := binary.LittleEndian.Uint16(frame[5:]); got != wantProps {
			t.Fatalf("视图 %d 属性数=%d, want %d", viewID, got, wantProps)
		}
		if wantProps == 1 {
			if got := binary.LittleEndian.Uint16(frame[7:]); got != propBaseCap {
				t.Fatalf("主页首属性索引=%d, want BaseCap@%d", got, propBaseCap)
			}
			if got := int32(binary.LittleEndian.Uint32(frame[9:])); got != 18 {
				t.Fatalf("BaseCap=%d, want 18", got)
			}
		}
	}
	if got := binary.LittleEndian.Uint16(frames[16][1:]); got != viewDoll {
		t.Fatalf("末帧视图=%d, want View1 纸娃娃", got)
	}
}

func TestItemBagFramesAnchors(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	permanent := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	expired := now.Add(-time.Hour)
	packs := []bagpackRow{
		{page: 1, anchor: 1, configID: "ITEMBAG_006", boost: 36, expiresAt: &permanent},
		{page: 1, anchor: 2, configID: "ITEMBAG_006", boost: 36, expiresAt: &permanent},
		{page: 2, anchor: 1, configID: "EQUIPBAG_005", boost: 36, expiresAt: &expired},
	}
	frames := itemBagFrames(packs, now)
	if len(frames) != 2 {
		t.Fatalf("锚点帧数=%d, want 2（过期不下发）", len(frames))
	}
	// anchorProps 按已知 schema 顺序解锚点帧属性：
	// ConfigID(String: u32 长度+字节)、BeginPos(Word)、EndPos(Word)、Hardiness(Int32)。
	anchorProps := func(frame []byte) (string, uint16, uint16, int32) {
		if frame[0] != 0x18 {
			t.Fatalf("锚点帧 opcode=%#x, want 0x18", frame[0])
		}
		off := 7
		readIndex := func() uint16 {
			index := binary.LittleEndian.Uint16(frame[off:])
			off += 2
			return index
		}
		if got := readIndex(); got != propItemConfigID {
			t.Fatalf("首属性=%d, want ConfigID@%d", got, propItemConfigID)
		}
		size := int(binary.LittleEndian.Uint32(frame[off:]))
		off += 4
		config := string(frame[off : off+size-1])
		off += size
		if got := readIndex(); got != propBeginPos {
			t.Fatalf("次属性=%d, want BeginPos@%d", got, propBeginPos)
		}
		begin := binary.LittleEndian.Uint16(frame[off:])
		off += 2
		if got := readIndex(); got != propEndPos {
			t.Fatalf("三属性=%d, want EndPos@%d", got, propEndPos)
		}
		end := binary.LittleEndian.Uint16(frame[off:])
		off += 2
		if got := readIndex(); got != propHardiness {
			t.Fatalf("末属性=%d, want Hardiness@%d", got, propHardiness)
		}
		hardiness := int32(binary.LittleEndian.Uint32(frame[off:]))
		return config, begin, end, hardiness
	}
	first := frames[0]
	if got := binary.LittleEndian.Uint16(first[1:]); got != 3 {
		t.Fatalf("页1 扩展视图=%d, want 3", got)
	}
	if got := binary.LittleEndian.Uint16(first[3:]); got != 1 {
		t.Fatalf("锚点位=%d, want 1", got)
	}
	if got := binary.LittleEndian.Uint16(first[5:]); got != 4 {
		t.Fatalf("锚点属性数=%d, want 4", got)
	}
	config, begin, end, hardiness := anchorProps(first)
	if config != "ITEMBAG_006" || begin != 19 || end != 54 {
		t.Fatalf("首锚 解析异常: config=%s begin=%d end=%d, want 19/54", config, begin, end)
	}
	if hardiness != 720 {
		t.Fatalf("永久锚 Hardiness=%d, want 720", hardiness)
	}
	// 第二锚累计 boost=36：BeginPos=19+36=55、EndPos=18+72=90。
	_, begin2, end2, _ := anchorProps(frames[1])
	if begin2 != 55 || end2 != 90 {
		t.Fatalf("次锚 begin=%d end=%d, want 55/90", begin2, end2)
	}
}

func TestAnchorHardiness(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	permanent := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	if got := anchorHardiness(&permanent, now); got != 720 {
		t.Fatalf("永久锚=%d, want 720", got)
	}
	if got := anchorHardiness(nil, now); got != 720 {
		t.Fatalf("无到期=%d, want 720", got)
	}
	if got := anchorHardiness(&[]time.Time{now.Add(10 * time.Hour)}[0], now); got != 10 {
		t.Fatalf("剩余小时=%d, want 10", got)
	}
	if got := anchorHardiness(&[]time.Time{now.Add(-time.Hour)}[0], now); got != 0 {
		t.Fatalf("已过期=%d, want 0", got)
	}
}

func TestWeaponVisualOf(t *testing.T) {
	injectStubInventoryEquipment(t)
	visual := weaponVisualOf(0, []equipRow{
		{configID: "shot_x", itemType: 110, equipType: "ShotWeapon"},
		{configID: "sword_test", itemType: 102, equipType: "Weapon"},
		{configID: "inner_y", itemType: 190, equipType: "InnerWeapon"},
	})
	if visual.weapon != "sword_test" {
		t.Fatalf("主手=%q, want 首件 Weapon（表缺模型回退裸名）", visual.weapon)
	}
	if visual.shotWeapon != "shot_x" {
		t.Fatalf("副手=%q, want shot_x", visual.shotWeapon)
	}
	// sword_test→ArtPack9→ActionSet=2h（表值优先）。
	if visual.actionSet != "2h" {
		t.Fatalf("actionSet=%q, want 2h（表值）", visual.actionSet)
	}
	if !visual.armed() {
		t.Fatal("有武器应 armed")
	}
	// 无表值武器回退 ItemType-100+"h"。
	fallback := weaponVisualOf(0, []equipRow{{configID: "no_table", itemType: 103, equipType: "Weapon"}})
	if fallback.actionSet != "3h" {
		t.Fatalf("回退 actionSet=%q, want 3h", fallback.actionSet)
	}
	empty := weaponVisualOf(0, nil)
	if empty.armed() {
		t.Fatal("空手不应 armed")
	}
}

func TestInventoryFramesPairedRows(t *testing.T) {
	injectStubInventoryEquipment(t)
	equips := []equipRow{
		{configID: "sword_test", itemType: 102, equipType: "Weapon", slot: 22,
			hardiness: 100, maxHardiness: 100, propBuilt: "1,105", seq: 11, uid: "abc123def4567890"},
	}
	items := []bagRow{
		{slot: 3, configID: "yaopin_00500", itemType: 1, amount: 2, viewID: 1, seq: 143},
		{slot: 91, configID: "overflow_x", itemType: 1, amount: 1, viewID: 1, seq: 200},
		{slot: 2, configID: "sword_test", itemType: 102, amount: 1, viewID: 2, equipType: "Weapon", seq: 3},
	}
	frames, err := inventoryFrames(618, 0, equips, items)
	if err != nil {
		t.Fatal(err)
	}
	// 纸娃娃 1 + 背包 2 行×3 帧（91+ 溢出裁剪）+ 武器 0x10 = 8。
	if len(frames) != 8 {
		t.Fatalf("帧数=%d, want 8", len(frames))
	}
	doll := frames[0]
	if doll[0] != 0x18 || binary.LittleEndian.Uint16(doll[1:]) != viewDoll {
		t.Fatalf("首帧应为 View1 纸娃娃，opcode=%#x", doll[0])
	}
	if got := binary.LittleEndian.Uint16(doll[3:]); got != 22 {
		t.Fatalf("纸娃娃槽位=%d, want 身体槽位 22", got)
	}
	if !strings.Contains(string(doll), "1,105") {
		t.Fatal("词缀 PropBuilt 应下发")
	}
	// 背包行成对：remove(道具页,3) add remove(镜像) remove(装备页,2) add remove(镜像)。
	wantOps := []struct {
		opcode byte
		view   uint16
		index  uint16
	}{
		{0x19, viewToolPage, 3}, {0x18, viewToolPage, 3}, {0x19, bagMirrorOf(viewToolPage), 3},
		{0x19, viewEquipBag, 2}, {0x18, viewEquipBag, 2}, {0x19, bagMirrorOf(viewEquipBag), 2},
	}
	for i, want := range wantOps {
		frame := frames[1+i]
		if frame[0] != want.opcode {
			t.Fatalf("frames[%d] opcode=%#x, want %#x", 1+i, frame[0], want.opcode)
		}
		view := binary.LittleEndian.Uint16(frame[1:])
		index := binary.LittleEndian.Uint16(frame[3:])
		if view != want.view || index != want.index {
			t.Fatalf("frames[%d] view=%d index=%d, want view=%d index=%d", 1+i, view, index, want.view, want.index)
		}
	}
	weaponFrame := frames[7]
	if weaponFrame[0] != 0x10 {
		t.Fatalf("末帧 opcode=%#x, want 武器外观 0x10", weaponFrame[0])
	}
	if !strings.Contains(string(weaponFrame), "sword_test") {
		t.Fatal("0x10 应携带 Weapon 模型名")
	}
}

func TestInventoryFramesCommonProps(t *testing.T) {
	injectStubInventoryEquipment(t)
	frames, err := inventoryFrames(665, 1, nil, []bagRow{
		{slot: 4, configID: "box_wnhd_dllb01", itemType: 2102, amount: 1, viewID: 1, seq: 0, uid: ""},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 3 {
		t.Fatalf("帧数=%d, want 3", len(frames))
	}
	add := frames[1]
	if got := binary.LittleEndian.Uint16(add[5:]); got != 6 {
		t.Fatalf("非装备行属性数=%d, want 通用 6 项", got)
	}
	// UniqueID 兜底槽位方言 MD5(role:bag:slot)。
	uid := itemUID(uint64(role.RoleID(665)), "bag", "", 0, 4)
	if !strings.Contains(string(add), uid) {
		t.Fatal("背包行应携带 UniqueID")
	}
}

func newMockInventoryDB(t *testing.T) (*inventoryStore, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		_ = db.Close()
	})
	return newInventoryStore(db), mock
}

func TestEquipsDegradeWithoutUIDColumn(t *testing.T) {
	store, mock := newMockInventoryDB(t)
	mock.ExpectQuery("SELECT config_id.*uid.*FROM role_equip_items").
		WithArgs(int64(role.RoleID(618))).
		WillReturnError(&mysqlError{message: "Unknown column 'uid' in 'field list'"})
	rows := sqlmock.NewRows([]string{"config_id", "item_type", "equip_type", "slot",
		"hardiness", "max_hardiness", "affixes", "seq"}).
		AddRow("sword_ytj_001", 102, "Weapon", 22, 100, 100, nil, 11)
	mock.ExpectQuery("SELECT config_id.*FROM role_equip_items").
		WithArgs(int64(role.RoleID(618))).
		WillReturnRows(rows)
	equips, err := store.equips(testContext(t), role.RoleID(618))
	if err != nil {
		t.Fatal(err)
	}
	if len(equips) != 1 || equips[0].configID != "sword_ytj_001" || equips[0].propBuilt != "" {
		t.Fatalf("降级读取异常: %+v", equips)
	}
	if equips[0].uid != "" {
		t.Fatalf("uid 缺席库应空: %q", equips[0].uid)
	}
}

type mysqlError struct{ message string }

func (e *mysqlError) Error() string { return e.message }

func TestBagItemsEnrichedFromGameItems(t *testing.T) {
	store, mock := newMockInventoryDB(t)
	rows := sqlmock.NewRows([]string{"slot", "config_id", "item_type", "amount", "view_id",
		"name", "equip_type", "hardiness", "max_hardiness", "seq", "uid", "affixes"}).
		AddRow(1, "box_wnhd_dllb01", 0, 1, 0, nil, nil, nil, nil, 0, nil, nil)
	mock.ExpectQuery("SELECT slot.*FROM role_bag_items").
		WithArgs(int64(role.RoleID(665))).
		WillReturnRows(rows)
	catalog := sqlmock.NewRows([]string{"item_id", "item_type", "view_id", "item_name", "equip_type"}).
		AddRow("box_wnhd_dllb01", 2102, 1, "登陆大礼包", "")
	mock.ExpectQuery("SELECT item_id, item_type, view_id, item_name, equip_type FROM game_items").
		WithArgs("box_wnhd_dllb01").
		WillReturnRows(catalog)
	items, err := store.bagItems(testContext(t), role.RoleID(665))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("行数=%d, want 1", len(items))
	}
	if items[0].itemType != 2102 || items[0].viewID != 1 || items[0].name != "登陆大礼包" {
		t.Fatalf("富化异常: %+v", items[0])
	}
}

// 对实库物品域的只读校验（不写库）：role_bag_items/role_equip_items/
// role_bagpacks 全部行可读、可编码成帧（线上实证 role 618=30 背包+9 穿戴、
// role 665=4 礼包行）。NINEYIN_TEST_MYSQL_DSN 未设置时跳过。
func TestInventoryDBLiveRowsEncodeFrames(t *testing.T) {
	dsn := os.Getenv("NINEYIN_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("NINEYIN_TEST_MYSQL_DSN not set")
	}
	// 生产口径连接（parseTime=true/UTC），与 role.OpenMySQLRepository
	// 交给 inventoryStore 的 db 同参数面。
	productionDSN, err := role.ProductionDSN(dsn)
	if err != nil {
		t.Fatalf("normalize dsn: %v", err)
	}
	db, err := sql.Open("mysql", productionDSN)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	defer db.Close()
	store := newInventoryStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, "SELECT DISTINCT role_id FROM role_bag_items")
	if err != nil {
		t.Fatalf("select bag roles: %v", err)
	}
	var roleIDs []role.RoleID
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		roleIDs = append(roleIDs, role.RoleID(id))
	}
	rows.Close()
	if len(roleIDs) == 0 {
		t.Skip("live db has no bag rows")
	}
	encoded := 0
	for _, id := range roleIDs {
		equips, equipErr := store.equips(ctx, id)
		if equipErr != nil {
			t.Fatalf("role %d equips: %v", id, equipErr)
		}
		items, bagErr := store.bagItems(ctx, id)
		if bagErr != nil {
			t.Fatalf("role %d bag: %v", id, bagErr)
		}
		packs, packErr := store.bagpacks(ctx, id)
		if packErr != nil {
			t.Fatalf("role %d bagpacks: %v", id, packErr)
		}
		if len(items) == 0 && len(equips) == 0 {
			continue
		}
		frames, frameErr := inventoryFrames(id, 0, equips, items)
		if frameErr != nil {
			t.Fatalf("role %d frames: %v", id, frameErr)
		}
		anchorFrames := itemBagFrames(packs, time.Now())
		if len(frames) == 0 || len(anchorFrames) == 0 {
			t.Fatalf("role %d produced empty frames (equips=%d items=%d packs=%d)",
				id, len(equips), len(items), len(packs))
		}
		encoded++
		t.Logf("role=%d equips=%d bagItems=%d anchors=%d frames=%d", id, len(equips), len(items), len(anchorFrames), len(frames))
	}
	if encoded == 0 {
		t.Fatal("no role produced inventory frames")
	}
}


// 0020 前旧库（affixes 列缺）：bagItems 降级重查且 propBuilt 恒空。
func TestBagItemsDegradeWithoutAffixesColumn(t *testing.T) {
	store, mock := newMockInventoryDB(t)
	mock.ExpectQuery("SELECT slot.*affixes.*FROM role_bag_items").
		WithArgs(int64(role.RoleID(665))).
		WillReturnError(&mysqlError{message: "Unknown column 'affixes' in 'field list'"})
	rows := sqlmock.NewRows([]string{"slot", "config_id", "item_type", "amount", "view_id",
		"name", "equip_type", "hardiness", "max_hardiness", "seq", "uid"}).
		AddRow(1, "box_wnhd_dllb01", 2102, 1, 1, "登陆大礼包", "", nil, nil, 0, nil)
	mock.ExpectQuery("SELECT slot.*FROM role_bag_items").
		WithArgs(int64(role.RoleID(665))).
		WillReturnRows(rows)
	catalog := sqlmock.NewRows([]string{"item_id", "item_type", "view_id", "item_name", "equip_type"})
	mock.ExpectQuery("SELECT item_id, item_type, view_id, item_name, equip_type FROM game_items").
		WithArgs("box_wnhd_dllb01").
		WillReturnRows(catalog)
	items, err := store.bagItems(testContext(t), role.RoleID(665))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].propBuilt != "" {
		t.Fatalf("降级读取异常: %+v", items)
	}
}
