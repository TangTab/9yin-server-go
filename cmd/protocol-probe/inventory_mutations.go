package main

// 物品域写穿 DB（移植 Java RoleInventoryStore 写侧）：moveBagSlot/
// moveBagToEquip/moveEquipToBag/swapEquipSlot/consumeBagItem/insertBagItem/
// upsertSkill。语义：先 DB 后内存（调用方在 DB 成功后才改会话模型）；
// uid/affixes 列缺席（0013 前参考库）按 Java 同款降级不写 uid。

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/local/9yin-go-server/internal/role"
)

// randomUid 生成实例身份（Java InventoryMessages.randomUid 同构 16-hex）。
func randomUid() string {
	var buffer [8]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return itemUID(0, "gen", "", 0, 0)
	}
	return hex.EncodeToString(buffer[:])
}

// effectiveUidCross 行 uid（空则跨表携带时新生成）。
func effectiveUidCross(uid string) string {
	if uid != "" {
		return uid
	}
	return randomUid()
}

// moveBagSlot 背包页之间移动（fromCategory 限定源页防跨页误删）。
func (s *inventoryStore) moveBagSlot(ctx context.Context, roleID role.RoleID, fromSlot, fromCategory, toSlot, toCategory int32) bool {
	if s == nil || s.db == nil {
		return true // 纯内存模式
	}
	result, err := s.db.ExecContext(ctx,
		"UPDATE role_bag_items SET slot = ?, view_id = ? WHERE role_id = ? AND slot = ? AND view_id = ?",
		toSlot, toCategory, roleID, fromSlot, fromCategory)
	if err != nil {
		log.Printf("moveBagSlot role=%d %d->%d: %v", roleID, fromSlot, toSlot, err)
		return false
	}
	affected, err := result.RowsAffected()
	return err == nil && affected > 0
}

// bagSlotMove 是一次重定位（seq 键控——role_bag_items 主键 (role_id,seq)，
// 槽位交换/整理批量重排时按槽位定位会连环误中，按实例身份定位与顺序无关）。
type bagSlotMove struct {
	seq    int64
	slot   int32
	viewID int32
}

// repositionBagRows 单事务批量重定位（拖拽交换=2 条 / 整理=N 条）：
// 逐条 UPDATE ... WHERE role_id=? AND seq=?；RowsAffected==0 视为内存
// 与 DB 脱钩，整批回滚拒绝（防止半途落盘造出新的重复槽位）。
func (s *inventoryStore) repositionBagRows(ctx context.Context, roleID role.RoleID, moves []bagSlotMove) bool {
	if s == nil || s.db == nil || len(moves) == 0 {
		return true
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("repositionBagRows role=%d begin: %v", roleID, err)
		return false
	}
	defer func() { _ = tx.Rollback() }()
	for _, move := range moves {
		result, err := tx.ExecContext(ctx,
			"UPDATE role_bag_items SET slot = ?, view_id = ? WHERE role_id = ? AND seq = ?",
			move.slot, move.viewID, roleID, move.seq)
		if err != nil {
			log.Printf("repositionBagRows role=%d seq=%d->slot=%d: %v", roleID, move.seq, move.slot, err)
			return false
		}
		affected, err := result.RowsAffected()
		if err != nil || affected == 0 {
			log.Printf("repositionBagRows role=%d seq=%d not found (affected=%d err=%v)", roleID, move.seq, affected, err)
			return false
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("repositionBagRows role=%d commit: %v", roleID, err)
		return false
	}
	return true
}

// mergeBagStack 同类叠堆合并（拖拽到同 configID 可叠行）：dst 行加数、
// src 行删除，均按 seq 键控；返回 dst 新总额（失败 ok=false）。
func (s *inventoryStore) mergeBagStack(ctx context.Context, roleID role.RoleID, srcSeq, dstSeq int64, addAmount int32) (int32, bool) {
	if s == nil || s.db == nil {
		return 0, true
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("mergeBagStack role=%d begin: %v", roleID, err)
		return 0, false
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx,
		"UPDATE role_bag_items SET amount = amount + ? WHERE role_id = ? AND seq = ?",
		addAmount, roleID, dstSeq)
	if err != nil {
		log.Printf("mergeBagStack role=%d dst seq=%d: %v", roleID, dstSeq, err)
		return 0, false
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		log.Printf("mergeBagStack role=%d dst seq=%d not found", roleID, dstSeq)
		return 0, false
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM role_bag_items WHERE role_id = ? AND seq = ?", roleID, srcSeq); err != nil {
		log.Printf("mergeBagStack role=%d src seq=%d: %v", roleID, srcSeq, err)
		return 0, false
	}
	var total int32
	if err := tx.QueryRowContext(ctx,
		"SELECT amount FROM role_bag_items WHERE role_id = ? AND seq = ?", roleID, dstSeq).Scan(&total); err != nil {
		log.Printf("mergeBagStack role=%d readback: %v", roleID, err)
		return 0, false
	}
	if err := tx.Commit(); err != nil {
		log.Printf("mergeBagStack role=%d commit: %v", roleID, err)
		return 0, false
	}
	return total, true
}

// moveBagToEquip 背包 → 装备栏（删背包行；装备行经 game_items 联表补
// equip_type/hardiness/max_hardiness/art_pack，uid 跨表携带）。
func (s *inventoryStore) moveBagToEquip(ctx context.Context, roleID role.RoleID, bagSlot, equipSlot, category int32) bool {
	if s == nil || s.db == nil {
		return true
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("moveBagToEquip role=%d begin: %v", roleID, err)
		return false
	}
	defer func() { _ = tx.Rollback() }()
	var configID string
	var itemType int32
	var carryUID sql.NullString
	if err := tx.QueryRowContext(ctx,
		"SELECT config_id, item_type, uid FROM role_bag_items WHERE role_id = ? AND slot = ? AND view_id = ?",
		roleID, bagSlot, category).Scan(&configID, &itemType, &carryUID); err != nil {
		log.Printf("moveBagToEquip role=%d select bag row: %v", roleID, err)
		return false
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM role_bag_items WHERE role_id = ? AND slot = ? AND view_id = ?",
		roleID, bagSlot, category); err != nil {
		log.Printf("moveBagToEquip role=%d delete bag row: %v", roleID, err)
		return false
	}
	if err := insertEquipRow(ctx, tx, roleID, equipSlot, configID, itemType, effectiveUidCross(carryUID.String)); err != nil {
		log.Printf("moveBagToEquip role=%d insert equip row: %v", roleID, err)
		return false
	}
	if err := tx.Commit(); err != nil {
		log.Printf("moveBagToEquip role=%d commit: %v", roleID, err)
		return false
	}
	return true
}

// moveEquipToBag 装备栏 → 背包（删装备行；背包行 uid 跨表携带）。
func (s *inventoryStore) moveEquipToBag(ctx context.Context, roleID role.RoleID, equipSlot, bagSlot, category int32) bool {
	if s == nil || s.db == nil {
		return true
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("moveEquipToBag role=%d begin: %v", roleID, err)
		return false
	}
	defer func() { _ = tx.Rollback() }()
	var configID string
	var itemType int32
	var carryUID sql.NullString
	var seq int64
	uidQuery := "SELECT config_id, item_type, uid, seq FROM role_equip_items WHERE role_id = ? AND slot = ?"
	if err := tx.QueryRowContext(ctx, uidQuery, roleID, equipSlot).
		Scan(&configID, &itemType, &carryUID, &seq); err != nil {
		if unknownColumn(err, "uid") {
			if err := tx.QueryRowContext(ctx,
				"SELECT config_id, item_type, seq FROM role_equip_items WHERE role_id = ? AND slot = ?",
				roleID, equipSlot).Scan(&configID, &itemType, &seq); err != nil {
				log.Printf("moveEquipToBag role=%d select equip row: %v", roleID, err)
				return false
			}
		} else {
			log.Printf("moveEquipToBag role=%d select equip row: %v", roleID, err)
			return false
		}
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM role_equip_items WHERE role_id = ? AND slot = ?", roleID, equipSlot); err != nil {
		log.Printf("moveEquipToBag role=%d delete equip row: %v", roleID, err)
		return false
	}
	if err := insertBagRow(ctx, tx, roleID, bagSlot, configID, itemType, 1, category, effectiveUidCross(carryUID.String)); err != nil {
		log.Printf("moveEquipToBag role=%d insert bag row: %v", roleID, err)
		return false
	}
	if err := tx.Commit(); err != nil {
		log.Printf("moveEquipToBag role=%d commit: %v", roleID, err)
		return false
	}
	return true
}

// swapEquipSlot 装备栏让位：旧装备入背包装备页（category=2）。
func (s *inventoryStore) swapEquipSlot(ctx context.Context, roleID role.RoleID, equipSlot, bagSlotForOld int32) bool {
	return s.moveEquipToBag(ctx, roleID, equipSlot, bagSlotForOld, 2)
}

// consumeBagItem 扣减背包行数量：减到 0 删行。返回剩余（删行=0）；
// 无行/不足返回 -1。（Java consumeBagItem 同构，含事务。）
func (s *inventoryStore) consumeBagItem(ctx context.Context, roleID role.RoleID, category, slot, count int32) (int32, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return -1, err
	}
	defer func() { _ = tx.Rollback() }()
	var amount int32
	err = tx.QueryRowContext(ctx,
		"SELECT amount FROM role_bag_items WHERE role_id = ? AND slot = ? AND view_id = ?",
		roleID, slot, category).Scan(&amount)
	if err == sql.ErrNoRows {
		return -1, nil
	}
	if err != nil {
		return -1, err
	}
	if amount < count {
		return -1, nil
	}
	if amount > count {
		if _, err = tx.ExecContext(ctx,
			"UPDATE role_bag_items SET amount = ? WHERE role_id = ? AND slot = ? AND view_id = ?",
			amount-count, roleID, slot, category); err != nil {
			return -1, err
		}
		if err = tx.Commit(); err != nil {
			return -1, err
		}
		return amount - count, nil
	}
	if _, err = tx.ExecContext(ctx,
		"DELETE FROM role_bag_items WHERE role_id = ? AND slot = ? AND view_id = ?",
		roleID, slot, category); err != nil {
		return -1, err
	}
	if err = tx.Commit(); err != nil {
		return -1, err
	}
	return 0, nil
}

// increaseBagItem 行内加数（商店购买叠堆路径）：无行返回 -1；返回新额。
func (s *inventoryStore) increaseBagItem(ctx context.Context, roleID role.RoleID, category, slot, count int32) (int32, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return -1, err
	}
	defer func() { _ = tx.Rollback() }()
	var amount int32
	err = tx.QueryRowContext(ctx,
		"SELECT amount FROM role_bag_items WHERE role_id = ? AND slot = ? AND view_id = ?",
		roleID, slot, category).Scan(&amount)
	if err == sql.ErrNoRows {
		return -1, nil
	}
	if err != nil {
		return -1, err
	}
	if _, err = tx.ExecContext(ctx,
		"UPDATE role_bag_items SET amount = ? WHERE role_id = ? AND slot = ? AND view_id = ?",
		amount+count, roleID, slot, category); err != nil {
		return -1, err
	}
	if err = tx.Commit(); err != nil {
		return -1, err
	}
	return amount + count, nil
}

// insertBagItem 新行入包：slot<=0 自动取 1..90 首空格；返回落库行
// （seq/uid 回填，调用方以返回值为帧与内存的权威）。
func (s *inventoryStore) insertBagItem(ctx context.Context, roleID role.RoleID, item bagRow) (bagRow, error) {
	if s == nil || s.db == nil {
		return item, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return item, err
	}
	defer func() { _ = tx.Rollback() }()
	used := make(map[int32]bool)
	rows, err := tx.QueryContext(ctx, "SELECT slot FROM role_bag_items WHERE role_id = ?", roleID)
	if err != nil {
		return item, err
	}
	for rows.Next() {
		var slot int32
		if err := rows.Scan(&slot); err != nil {
			rows.Close()
			return item, err
		}
		used[slot] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return item, err
	}
	slot := item.slot
	if slot <= 0 {
		slot = 0
		for candidate := int32(1); candidate <= int32(bagSlotMax); candidate++ {
			if !used[candidate] {
				slot = candidate
				break
			}
		}
		if slot == 0 {
			return item, fmt.Errorf("bag full (1..%d)", bagSlotMax)
		}
	}
	var seq int64
	if err := tx.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(seq), 0) + 1 FROM role_bag_items WHERE role_id = ?", roleID).Scan(&seq); err != nil {
		return item, err
	}
	uid := randomUid()
	insert := "INSERT INTO role_bag_items (role_id, seq, slot, config_id, item_type, amount, view_id, name, script, uid) " +
		"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	if _, err := tx.ExecContext(ctx, insert,
		roleID, seq, slot, item.configID, item.itemType, item.amount, item.viewID,
		nullString(item.name), nullString(""), uid); err != nil {
		if !unknownColumn(err, "uid") {
			return item, err
		}
		insert = "INSERT INTO role_bag_items (role_id, seq, slot, config_id, item_type, amount, view_id, name, script) " +
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
		if _, err := tx.ExecContext(ctx, insert,
			roleID, seq, slot, item.configID, item.itemType, item.amount, item.viewID,
			nullString(item.name), nullString("")); err != nil {
			return item, err
		}
		uid = ""
	}
	if err := tx.Commit(); err != nil {
		return item, err
	}
	item.slot = slot
	item.seq = seq
	item.uid = uid
	return item, nil
}

// insertBagRow 事务内新行（Java insertBagRow 同构；uid 列缺席降级不写）。
func insertBagRow(ctx context.Context, tx *sql.Tx, roleID role.RoleID, slot int32, configID string, itemType, amount, category int32, carryUid string) error {
	var seq int64
	if err := tx.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(seq), 0) + 1 FROM role_bag_items WHERE role_id = ?", roleID).Scan(&seq); err != nil {
		return err
	}
	uid := effectiveUidCross(carryUid)
	insert := "INSERT INTO role_bag_items (role_id, seq, slot, config_id, item_type, amount, view_id, name, script, uid) " +
		"VALUES (?, ?, ?, ?, ?, ?, ?, '', '', ?)"
	if _, err := tx.ExecContext(ctx, insert, roleID, seq, slot, configID, itemType, amount, category, uid); err != nil {
		if !unknownColumn(err, "uid") {
			return err
		}
		_, err = tx.ExecContext(ctx,
			"INSERT INTO role_bag_items (role_id, seq, slot, config_id, item_type, amount, view_id, name, script) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?, '', '')",
			roleID, seq, slot, configID, itemType, amount, category)
		return err
	}
	return nil
}

// insertEquipRow 事务内装备行（game_items 联表补全；uid/affixes 列缺席
// 逐级降级）。镇派兵器（@PropModifyPackRec 条件动态包）在此单点写入
// affixes 包列表 JSON——读侧 propBuiltOf → PropBuilt@1954 下发 tooltip。
func insertEquipRow(ctx context.Context, tx *sql.Tx, roleID role.RoleID, slot int32, configID string, itemType int32, carryUid string) error {
	var seq int64
	if err := tx.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(seq), 0) + 1 FROM role_equip_items WHERE role_id = ?", roleID).Scan(&seq); err != nil {
		return err
	}
	uid := effectiveUidCross(carryUid)
	affixes := randomEquipCatalogOf().zhenpaiAffixesJSON(configID)
	if affixes == "" {
		affixes = "[]"
	}
	insert := "INSERT INTO role_equip_items (role_id, seq, config_id, item_type, equip_type, hardiness, max_hardiness, slot, art_pack, uid, affixes) " +
		"SELECT ?, ?, g.item_id, g.item_type, g.equip_type, g.hardiness, g.max_hardiness, ?, g.art_pack, ?, ? FROM game_items g WHERE g.item_id = ?"
	result, err := tx.ExecContext(ctx, insert, roleID, seq, slot, uid, affixes, configID)
	if err == nil {
		if affected, affErr := result.RowsAffected(); affErr == nil && affected == 0 {
			// game_items 无行：按背包行类型兜底直插（equip_type 由调用方内存行给出）。
			_, err = tx.ExecContext(ctx,
				"INSERT INTO role_equip_items (role_id, seq, config_id, item_type, equip_type, hardiness, max_hardiness, slot, uid, affixes) "+
					"VALUES (?, ?, ?, ?, ?, 100, 100, ?, ?, ?)",
				roleID, seq, configID, itemType, "", slot, uid, affixes)
			if err != nil && unknownColumn(err, "uid") {
				_, err = tx.ExecContext(ctx,
					"INSERT INTO role_equip_items (role_id, seq, config_id, item_type, equip_type, hardiness, max_hardiness, slot, affixes) "+
						"VALUES (?, ?, ?, ?, ?, 100, 100, ?, ?)",
					roleID, seq, configID, itemType, "", slot, affixes)
				if err != nil && unknownColumn(err, "affixes") {
					_, err = tx.ExecContext(ctx,
						"INSERT INTO role_equip_items (role_id, seq, config_id, item_type, equip_type, hardiness, max_hardiness, slot) "+
							"VALUES (?, ?, ?, ?, ?, 100, 100, ?)",
						roleID, seq, configID, itemType, "", slot)
				}
			} else if err != nil && unknownColumn(err, "affixes") {
				_, err = tx.ExecContext(ctx,
					"INSERT INTO role_equip_items (role_id, seq, config_id, item_type, equip_type, hardiness, max_hardiness, slot, uid) "+
						"VALUES (?, ?, ?, ?, ?, 100, 100, ?, ?)",
					roleID, seq, configID, itemType, "", slot, uid)
			}
			return err
		}
		return nil
	}
	switch {
	case unknownColumn(err, "affixes"):
		insert = "INSERT INTO role_equip_items (role_id, seq, config_id, item_type, equip_type, hardiness, max_hardiness, slot, art_pack, uid) " +
			"SELECT ?, ?, g.item_id, g.item_type, g.equip_type, g.hardiness, g.max_hardiness, ?, g.art_pack, ? FROM game_items g WHERE g.item_id = ?"
		_, err = tx.ExecContext(ctx, insert, roleID, seq, slot, uid, configID)
	case unknownColumn(err, "uid"):
		insert = "INSERT INTO role_equip_items (role_id, seq, config_id, item_type, equip_type, hardiness, max_hardiness, slot, art_pack, affixes) " +
			"SELECT ?, ?, g.item_id, g.item_type, g.equip_type, g.hardiness, g.max_hardiness, ?, g.art_pack, ? FROM game_items g WHERE g.item_id = ?"
		_, err = tx.ExecContext(ctx, insert, roleID, seq, slot, affixes, configID)
		if err != nil && unknownColumn(err, "affixes") {
			insert = "INSERT INTO role_equip_items (role_id, seq, config_id, item_type, equip_type, hardiness, max_hardiness, slot, art_pack) " +
				"SELECT ?, ?, g.item_id, g.item_type, g.equip_type, g.hardiness, g.max_hardiness, ?, g.art_pack FROM game_items g WHERE g.item_id = ?"
			_, err = tx.ExecContext(ctx, insert, roleID, seq, slot, configID)
		}
	}
	return err
}

// upsertEquipSlot 时装/身份装穿戴（Java RoleInventoryStore.upsertEquip 的
// 槽位直写形式：时装不走背包源，直接 upsert 纸娃娃槽 16-19）。同槽同款
// 重穿的 uid 保留（客户端实例稳定身份）；异 configID 覆盖生成新 uid——
// 继承旧 uid 会让新物品命中旧物品的随机属性快照（role_item_random_props
// 按 uid 键控，GM 连发宝物/娃娃到同槽时属性串号），调用方负责先让位旧装
// 备（GM 路径走 unequipFromDoll）。delete+insert 事务内完成；落库后
// 回读权威行（seq/uid/equip_type 以 DB 为准）供内存与帧使用。无 DB
// （内存降级）返回调用方构造的行。
func (s *inventoryStore) upsertEquipSlot(ctx context.Context, roleID role.RoleID, slot int32, configID string, itemType int32, equipType string) (equipRow, error) {
	row := equipRow{
		configID: configID, itemType: itemType, equipType: equipType,
		slot: slot, hardiness: 100, maxHardiness: 100,
	}
	if s == nil || s.db == nil {
		return row, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return row, err
	}
	defer func() { _ = tx.Rollback() }()
	var carryUid, oldConfigID sql.NullString
	err = tx.QueryRowContext(ctx,
		"SELECT uid, config_id FROM role_equip_items WHERE role_id = ? AND slot = ?", roleID, slot).
		Scan(&carryUid, &oldConfigID)
	if err != nil && err != sql.ErrNoRows && !unknownColumn(err, "uid") {
		return row, err
	}
	if carryUid.Valid && oldConfigID.Valid && oldConfigID.String != configID {
		carryUid = sql.NullString{} // 异款覆盖：新实例身份，不带旧 uid（String 一并清空）
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM role_equip_items WHERE role_id = ? AND slot = ?", roleID, slot); err != nil {
		return row, err
	}
	if err := insertEquipRow(ctx, tx, roleID, slot, configID, itemType, carryUid.String); err != nil {
		return row, err
	}
	if err := tx.Commit(); err != nil {
		return row, err
	}
	// 回读权威行（game_items 联表已补 equip_type/hardiness；affixes 随行
	// 回读——镇派兵器的 PropBuilt 必须随穿戴帧即时下发，否则 tooltip 要
	// 等重登才出词条）；回读失败不回滚穿戴（行已落库），用已知字段兜底。
	var affixes sql.NullString
	err = s.db.QueryRowContext(ctx,
		"SELECT config_id, item_type, equip_type, slot, hardiness, max_hardiness, seq, uid, affixes "+
			"FROM role_equip_items WHERE role_id = ? AND slot = ?", roleID, slot).
		Scan(&row.configID, &row.itemType, &row.equipType, &row.slot,
			&row.hardiness, &row.maxHardiness, &row.seq, &carryUid, &affixes)
	if err != nil {
		if !unknownColumn(err, "affixes") {
			err = s.db.QueryRowContext(ctx,
				"SELECT config_id, item_type, equip_type, slot, hardiness, max_hardiness, seq, uid "+
					"FROM role_equip_items WHERE role_id = ? AND slot = ?", roleID, slot).
				Scan(&row.configID, &row.itemType, &row.equipType, &row.slot,
					&row.hardiness, &row.maxHardiness, &row.seq, &carryUid)
		}
	}
	if err != nil {
		row.configID, row.itemType, row.equipType, row.slot = configID, itemType, equipType, slot
		row.propBuilt = randomEquipCatalogOf().zhenpaiPropBuiltString(configID)
	} else {
		row.propBuilt = propBuiltOf(affixes)
	}
	row.uid = carryUid.String
	return row, nil
}

// deleteEquipSlot 按槽位删除装备行（时装脱下；Java deleteEquip 同构）。
func (s *inventoryStore) deleteEquipSlot(ctx context.Context, roleID role.RoleID, slot int32) error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM role_equip_items WHERE role_id = ? AND slot = ?", roleID, slot)
	return err
}
func (s *inventoryStore) upsertSkill(ctx context.Context, roleID role.RoleID, skillID, skillType string) error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx,
		"INSERT IGNORE INTO role_skills (role_id, skill_type, skill_id) VALUES (?, ?, ?)",
		roleID, skillType, skillID)
	if err != nil {
		log.Printf("upsertSkill role=%d %s: %v", roleID, skillID, err)
	}
	return err
}

// deleteSkillsByTypes 删除角色的指定类型技能行（GM forget_all_taolu 删
// wuxue/anqi；Java RoleInventoryStore.deleteSkills 同构，Go 的 View40 恒为
// starter∪DB 并集模型，暗器行一并清才能与重登基线一致）。
func (s *inventoryStore) deleteSkillsByTypes(ctx context.Context, roleID role.RoleID, skillTypes []string) error {
	if s == nil || s.db == nil || len(skillTypes) == 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM role_skills WHERE role_id = ? AND skill_type IN ("+strings.TrimRight(strings.Repeat("?,", len(skillTypes)), ",")+")",
		append([]any{roleID}, anySlice(skillTypes)...)...)
	return err
}

// anySlice 把 []string 抬成 []any（SQL 占位参数拼接用）。
func anySlice(values []string) []any {
	result := make([]any, len(values))
	for index, value := range values {
		result[index] = value
	}
	return result
}

// grantableItem 是发放目录行（game_items 查询结果；ToolItemCatalog.ItemInfo 对应物）。
type grantableItem struct {
	configID  string
	itemType  int32
	viewID    int32
	name      string
	equipType string
	script    string
}

// lookupGrantable 查 game_items 发放目录（无命中返回 ok=false）。
func (s *inventoryStore) lookupGrantable(ctx context.Context, configID string) (grantableItem, bool) {
	var entry grantableItem
	if s == nil || s.db == nil || configID == "" {
		return entry, false
	}
	var name, equipType, script sql.NullString
	err := s.db.QueryRowContext(ctx,
		"SELECT item_id, item_type, view_id, item_name, equip_type, script FROM game_items WHERE item_id = ?", configID).
		Scan(&entry.configID, &entry.itemType, &entry.viewID, &name, &equipType, &script)
	if err == sql.ErrNoRows {
		return entry, false
	}
	if err != nil {
		log.Printf("lookupGrantable %s: %v", configID, err)
		return entry, false
	}
	entry.name, entry.equipType, entry.script = name.String, equipType.String, script.String
	return entry, true
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// itemNames 批量解析 game_items 中文名（GM 目录端点用；空表/无 DB 返回
// 空映射，条目名由调用方回退 ConfigID）。
func (s *inventoryStore) itemNames(ctx context.Context, configIDs []string) map[string]string {
	names := make(map[string]string, len(configIDs))
	if s == nil || s.db == nil || len(configIDs) == 0 {
		return names
	}
	const batch = 64
	for start := 0; start < len(configIDs); start += batch {
		end := start + batch
		if end > len(configIDs) {
			end = len(configIDs)
		}
		batchIDs := configIDs[start:end]
		placeholders := make([]string, len(batchIDs))
		args := make([]any, len(batchIDs))
		for index, configID := range batchIDs {
			placeholders[index] = "?"
			args[index] = configID
		}
		rows, err := s.db.QueryContext(ctx,
			"SELECT item_id, item_name FROM game_items WHERE item_id IN ("+strings.Join(placeholders, ",")+")",
			args...)
		if err != nil {
			log.Printf("itemNames batch lookup: %v", err)
			return names
		}
		for rows.Next() {
			var configID, name string
			if err := rows.Scan(&configID, &name); err != nil {
				rows.Close()
				return names
			}
			if name != "" {
				names[configID] = name
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return names
		}
	}
	return names
}
