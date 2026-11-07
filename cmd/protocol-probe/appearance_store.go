package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// 外观域 MySQL 存储：role_appearance（显示类型偏好，0007 迁移权威）+
// role_fashion_suits（时装页签激活集，0001 基线表，Java
// RoleInventoryStore.activateFashionSuits/fashionSuits 同口径）。
// 降级语义与 customConfigStores 一致：缺表/坏行读作「无存档」，登录链
// 走默认（显示类型 0），写入错误由调用方降级日志。
const appearanceQueryTimeout = 5 * time.Second

// appearanceStores 聚合外观域三个存储（customConfigStores 同款装配层）。
type appearanceStores struct {
	appearance *appearanceDBStore
	fashion    *fashionSuitDBStore
	skins      *skinDBStore
}

func newAppearanceStores(db *sql.DB) appearanceStores {
	return appearanceStores{
		appearance: newAppearanceDBStore(db),
		fashion:    newFashionSuitDBStore(db),
		skins:      newSkinDBStore(db),
	}
}

// appearanceDBStore 持久化 ShowEquipType 偏好（custom 61 三 radio）。
type appearanceDBStore struct {
	db *sql.DB
}

func newAppearanceDBStore(db *sql.DB) *appearanceDBStore {
	return &appearanceDBStore{db: db}
}

// Load 读取显示类型偏好（无行/DB 错误按「无存档」处理，调用方按 0 江湖装）。
func (store *appearanceDBStore) Load(roleID role.RoleID) (int, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return showEquipGear, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), appearanceQueryTimeout)
	defer cancel()
	var showEquipType int
	err := store.db.QueryRowContext(ctx,
		`SELECT show_equip_type FROM role_appearance WHERE role_id = ?`, roleID).Scan(&showEquipType)
	if errors.Is(err, sql.ErrNoRows) {
		return showEquipGear, false
	}
	if err != nil {
		log.Printf("appearance store: load role=%d: %v", roleID, err)
		return showEquipGear, false
	}
	if showEquipType < showEquipGear || showEquipType > showEquipFashion {
		return showEquipGear, false
	}
	return showEquipType, true
}

// Save upsert 显示类型偏好（Java AppearanceStore.upsertDatabase 同 SQL，
// version 逐次 +1；updated_at 由库维护）。
func (store *appearanceDBStore) Save(roleID role.RoleID, showEquipType int) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("appearance store: no database")
	}
	if roleID == 0 {
		return errors.New("appearance store: zero role id")
	}
	if showEquipType < showEquipGear || showEquipType > showEquipFashion {
		return fmt.Errorf("appearance store: show equip type %d out of range", showEquipType)
	}
	ctx, cancel := context.WithTimeout(context.Background(), appearanceQueryTimeout)
	defer cancel()
	if _, err := store.db.ExecContext(ctx,
		`INSERT INTO role_appearance (role_id, show_equip_type, version, updated_at)
VALUES (?, ?, 1, UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE show_equip_type = VALUES(show_equip_type),
  version = version + 1, updated_at = UTC_TIMESTAMP(6)`,
		roleID, showEquipType); err != nil {
		return fmt.Errorf("persist appearance role=%d: %w", roleID, err)
	}
	return nil
}

// fashionSuitDBStore 持久化时装/身份装激活集（role_fashion_suits，
// 主键 (role_id, suit_id)；FashionSuitRec 行的 DB 源）。
type fashionSuitDBStore struct {
	db *sql.DB
}

func newFashionSuitDBStore(db *sql.DB) *fashionSuitDBStore {
	return &fashionSuitDBStore{db: db}
}

// List 按 suit_id 升序回放激活集（内存集合保持排序稳定，收录判定
// 大小写不敏感）。
func (store *fashionSuitDBStore) List(roleID role.RoleID) ([]string, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), appearanceQueryTimeout)
	defer cancel()
	rows, err := store.db.QueryContext(ctx,
		`SELECT suit_id FROM role_fashion_suits WHERE role_id = ? ORDER BY suit_id`, roleID)
	if err != nil {
		log.Printf("fashion suit store: list role=%d: %v", roleID, err)
		return nil, false
	}
	defer rows.Close()
	var suits []string
	for rows.Next() {
		var suitID string
		if err := rows.Scan(&suitID); err != nil {
			log.Printf("fashion suit store: scan role=%d: %v", roleID, err)
			return nil, false
		}
		suits = append(suits, suitID)
	}
	if err := rows.Err(); err != nil {
		log.Printf("fashion suit store: iterate role=%d: %v", roleID, err)
		return nil, false
	}
	return suits, true
}

// Activate 幂等激活一批套装（INSERT IGNORE，Java activateFashionSuits
// 同口径——已存在行静默跳过）。
func (store *fashionSuitDBStore) Activate(roleID role.RoleID, suitIDs []string) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("fashion suit store: no database")
	}
	if roleID == 0 {
		return errors.New("fashion suit store: zero role id")
	}
	if len(suitIDs) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), appearanceQueryTimeout)
	defer cancel()
	statement, err := store.db.PrepareContext(ctx,
		`INSERT IGNORE INTO role_fashion_suits (role_id, suit_id) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare activate fashion suits role=%d: %w", roleID, err)
	}
	defer statement.Close()
	for _, suitID := range suitIDs {
		if _, err := statement.ExecContext(ctx, roleID, suitID); err != nil {
			return fmt.Errorf("activate fashion suit role=%d suit=%s: %w", roleID, suitID, err)
		}
	}
	return nil
}

// mergeFashionSuits 合并激活集（去重、升序；登录自动发放/购买共用）。
func mergeFashionSuits(existing, added []string) []string {
	seen := make(map[string]bool, len(existing)+len(added))
	merged := make([]string, 0, len(existing)+len(added))
	for _, suit := range append(append([]string(nil), existing...), added...) {
		if !seen[suit] {
			seen[suit] = true
			merged = append(merged, suit)
		}
	}
	sort.Strings(merged)
	return merged
}

// containsIgnoreCase 激活集收录判定（Java containsIgnoreCase 同口径）。
func containsIgnoreCase(haystack []string, needle string) bool {
	for _, value := range haystack {
		if strings.EqualFold(value, needle) {
			return true
		}
	}
	return false
}
