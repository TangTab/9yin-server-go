package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// 物品域只读 DB 访问，移植 Java RoleInventoryStore（读路径）：
// role_equip_items（穿戴）/role_bag_items（背包，game_items 富化）/
// role_bagpacks（扩容锚点）。失败由调用方回退空背包（不阻塞登录）。

// equipRow 是 role_equip_items 的穿戴行（身体槽位 1 帽/3 衣/4 裤/6 腕/
// 7 腿/8 鞋/22 武器…）。seq/uid=实例稳定身份；旧行 uid 空按 seq 派生。
type equipRow struct {
	configID     string
	itemType     int32
	equipType    string
	slot         int32
	hardiness    int32
	maxHardiness int32
	propBuilt    string
	seq          int64
	uid          string
}

// bagRow 是 role_bag_items 的背包行（slot=真实槽位 1..90；viewID 决定
// 进哪个视图页：game_items 分类 1 道具/2 装备/3 材料/4 任务）。
type bagRow struct {
	slot         int32
	configID     string
	itemType     int32
	amount       int32
	viewID       int32
	name         string
	equipType    string
	hardiness    int32
	maxHardiness int32
	seq          int64
	uid          string
}

// bagpackRow 是扩容包锚点（page 1..4；boost=扩容格数；永久=9999-12-31）。
type bagpackRow struct {
	page      int32
	anchor    int32
	configID  string
	boost     int32
	expiresAt *time.Time
}

type inventoryStore struct {
	db *sql.DB
}

func newInventoryStore(db *sql.DB) *inventoryStore {
	return &inventoryStore{db: db}
}

func (s *inventoryStore) query(ctx context.Context, roleID role.RoleID, sqlText string, scan func(*sql.Rows) error) error {
	rows, err := s.db.QueryContext(ctx, sqlText, roleID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// unknownColumn 判断 MySQL 报错是否为缺列（F 版参考服还原库可能没有
// 本服迁移加的 uid/affixes 列），用于降级重查。
func unknownColumn(err error, column string) bool {
	return err != nil && strings.Contains(err.Error(), "Unknown column '"+column+"'")
}

// equips 读取穿戴行，按 seq 升序。uid/affixes 列缺席时逐级降级重查。
func (s *inventoryStore) equips(ctx context.Context, roleID role.RoleID) ([]equipRow, error) {
	const full = "SELECT config_id, item_type, equip_type, slot, hardiness, max_hardiness, affixes, seq, uid " +
		"FROM role_equip_items WHERE role_id = ? ORDER BY seq"
	const noUID = "SELECT config_id, item_type, equip_type, slot, hardiness, max_hardiness, affixes, seq " +
		"FROM role_equip_items WHERE role_id = ? ORDER BY seq"
	const noAffixes = "SELECT config_id, item_type, equip_type, slot, hardiness, max_hardiness, seq " +
		"FROM role_equip_items WHERE role_id = ? ORDER BY seq"
	equips := make([]equipRow, 0, 8)
	scanWith := func(withAffixes, withUID bool) func(*sql.Rows) error {
		return func(rows *sql.Rows) error {
			var row equipRow
			var affixes sql.NullString
			var uid sql.NullString
			dest := []any{&row.configID, &row.itemType, &row.equipType, &row.slot,
				&row.hardiness, &row.maxHardiness}
			if withAffixes {
				dest = append(dest, &affixes)
			}
			dest = append(dest, &row.seq)
			if withUID {
				dest = append(dest, &uid)
			}
			if err := rows.Scan(dest...); err != nil {
				return err
			}
			row.propBuilt = propBuiltOf(affixes)
			row.uid = uid.String
			equips = append(equips, row)
			return nil
		}
	}
	err := s.query(ctx, roleID, full, scanWith(true, true))
	switch {
	case err == nil:
		return equips, nil
	case unknownColumn(err, "uid"):
		err = s.query(ctx, roleID, noUID, scanWith(true, false))
		if unknownColumn(err, "affixes") {
			err = s.query(ctx, roleID, noAffixes, scanWith(false, false))
		}
		return equips, err
	case unknownColumn(err, "affixes"):
		return equips, s.query(ctx, roleID, noAffixes, scanWith(false, false))
	default:
		return equips, err
	}
}

// propBuiltOf 把 affixes JSON [{"pack_rec":0,"prop_built":id,"kind":N}] 解成
// 协议 PropBuilt 串 "N,id;N,id"。段首数字是表类型列（tips_func_equip.lua
// get_text_by_table_id：1=proppack 属性包 / 2=skillpack 套路威力包 / 3-5=
// buff/equip/task 包）；kind 缺省按 1（Java 旧存档 [{"pack_rec":0,
// "prop_built":id}] 兼容）。空/坏 JSON 返回空串。
func propBuiltOf(affixes sql.NullString) string {
	if !affixes.Valid || affixes.String == "" {
		return ""
	}
	var entries []struct {
		PackRec   int64 `json:"pack_rec"`
		PropBuilt int64 `json:"prop_built"`
		Kind      int64 `json:"kind"`
	}
	if err := json.Unmarshal([]byte(affixes.String), &entries); err != nil {
		return ""
	}
	segments := make([]string, 0, len(entries))
	for _, entry := range entries {
		kind := entry.Kind
		if kind < 1 || kind > 5 {
			kind = 1
		}
		segments = append(segments, fmt.Sprintf("%d,%d", kind, entry.PropBuilt))
	}
	return strings.Join(segments, ";")
}

// bagItems 读取背包行（seq 升序）并用 game_items 富化 itemType/viewID/
// name/equipType 的缺省值。uid 列缺席降级重查。
func (s *inventoryStore) bagItems(ctx context.Context, roleID role.RoleID) ([]bagRow, error) {
	const full = "SELECT slot, config_id, item_type, amount, view_id, name, equip_type, " +
		"hardiness, max_hardiness, seq, uid FROM role_bag_items WHERE role_id = ? ORDER BY seq"
	const noUID = "SELECT slot, config_id, item_type, amount, view_id, name, equip_type, " +
		"hardiness, max_hardiness, seq FROM role_bag_items WHERE role_id = ? ORDER BY seq"
	items := make([]bagRow, 0, 16)
	scanWith := func(withUID bool) func(*sql.Rows) error {
		return func(rows *sql.Rows) error {
			var row bagRow
			var name, equipType, uid sql.NullString
			// hardiness/max_hardiness 列可空（礼包类物品为 NULL）。
			var hardiness, maxHardiness sql.NullInt32
			dest := []any{&row.slot, &row.configID, &row.itemType, &row.amount, &row.viewID,
				&name, &equipType, &hardiness, &maxHardiness, &row.seq}
			if withUID {
				dest = append(dest, &uid)
			}
			if err := rows.Scan(dest...); err != nil {
				return err
			}
			row.name, row.equipType, row.uid = name.String, equipType.String, uid.String
			row.hardiness, row.maxHardiness = hardiness.Int32, maxHardiness.Int32
			items = append(items, row)
			return nil
		}
	}
	err := s.query(ctx, roleID, full, scanWith(true))
	if unknownColumn(err, "uid") {
		err = s.query(ctx, roleID, noUID, scanWith(false))
	}
	if err != nil {
		return items, err
	}
	enrichFromGameItems(ctx, s.db, items)
	return items, nil
}

// gameItemCatalog 是 game_items 的富化信息（item_type/view_id/item_name/
// equip_type），configId 缺行时保留 DB 行原值。
type gameItemCatalog struct {
	itemType  int32
	viewID    int32
	name      string
	equipType string
}

// enrichFromGameItems 按一次 IN 批查（200/批）富化背包行的缺省字段。
func enrichFromGameItems(ctx context.Context, db *sql.DB, items []bagRow) {
	if len(items) == 0 {
		return
	}
	ids := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		if !seen[item.configID] {
			seen[item.configID] = true
			ids = append(ids, item.configID)
		}
	}
	catalog := map[string]gameItemCatalog{}
	for start := 0; start < len(ids); start += 200 {
		end := start + 200
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]
		placeholders := strings.TrimRight(strings.Repeat("?,", len(chunk)), ",")
		args := make([]any, 0, len(chunk))
		for _, id := range chunk {
			args = append(args, id)
		}
		rows, err := db.QueryContext(ctx,
			"SELECT item_id, item_type, view_id, item_name, equip_type FROM game_items WHERE item_id IN ("+placeholders+")",
			args...)
		if err != nil {
			// 富化失败保留 DB 行原值。
			return
		}
		for rows.Next() {
			var id string
			var info gameItemCatalog
			if err := rows.Scan(&id, &info.itemType, &info.viewID, &info.name, &info.equipType); err != nil {
				_ = rows.Close()
				return
			}
			catalog[id] = info
		}
		_ = rows.Close()
	}
	for i := range items {
		info, ok := catalog[items[i].configID]
		if !ok {
			continue
		}
		if items[i].itemType == 0 {
			items[i].itemType = info.itemType
		}
		if items[i].viewID == 0 {
			items[i].viewID = info.viewID
		}
		if items[i].name == "" {
			items[i].name = info.name
		}
		if items[i].equipType == "" {
			items[i].equipType = info.equipType
		}
	}
}

// bagpacks 读取扩容锚点（page, anchor 升序）。
func (s *inventoryStore) bagpacks(ctx context.Context, roleID role.RoleID) ([]bagpackRow, error) {
	packs := make([]bagpackRow, 0, 4)
	err := s.query(ctx, roleID,
		"SELECT page, anchor, config_id, boost, expires_at FROM role_bagpacks WHERE role_id = ? ORDER BY page, anchor",
		func(rows *sql.Rows) error {
			var pack bagpackRow
			var expires sql.NullTime
			if err := rows.Scan(&pack.page, &pack.anchor, &pack.configID, &pack.boost, &expires); err != nil {
				return err
			}
			if expires.Valid {
				pack.expiresAt = &expires.Time
			}
			packs = append(packs, pack)
			return nil
		})
	return packs, err
}
