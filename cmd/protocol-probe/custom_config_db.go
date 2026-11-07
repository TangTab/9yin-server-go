package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// 自定义配置域的三个 MySQL 存储，读写与 Java 参考服逐字对齐
// （docs/customizing-shortcut-persist.md 定稿），共享 nineyin 库可与其
// 多服务端共存。降级语义与 facultyDBStore 一致：缺表/坏行读作
// 「无存档」，登录链走默认初始化；写入错误由调用方决定是否降级日志。
const customConfigQueryTimeout = 5 * time.Second

// customConfigStores 聚合自定义配置域的三个存储（与 facultyStore/
// inventoryStore 同一装配层），避免 handle 参数表继续膨胀。
type customConfigStores struct {
	customizing *customizingDBStore
	keybind     *keybindDBStore
	shortcuts   *shortcutDBStore
}

func newCustomConfigStores(db *sql.DB) customConfigStores {
	return customConfigStores{
		customizing: newCustomizingDBStore(db),
		keybind:     newKeybindDBStore(db),
		shortcuts:   newShortcutDBStore(db),
	}
}

// customizingDBStore 持久化引擎个性配置串（customizing_manager 上传的
// C2S 107 整串）。列名 customizing_value 对齐 F 数据同步存量表；串含
// 控制字节（0x01-0x65），Go string 原样保留，线上以 []byte 往返。
type customizingDBStore struct {
	db *sql.DB
}

func newCustomizingDBStore(db *sql.DB) *customizingDBStore {
	return &customizingDBStore{db: db}
}

// Load 读取角色个性配置；无行、DB 错误都按「无存档」处理并记日志
// （无存档时客户端保持本地默认并在登录期上传基线）。
func (store *customizingDBStore) Load(roleID role.RoleID) (string, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), customConfigQueryTimeout)
	defer cancel()
	var raw []byte
	err := store.db.QueryRowContext(ctx,
		`SELECT customizing_value FROM role_customizing WHERE role_id = ?`, roleID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false
	}
	if err != nil {
		log.Printf("customizing store: load role=%d: %v", roleID, err)
		return "", false
	}
	return string(raw), true
}

// Save upsert 个性配置串（updated_at 由库维护）。
func (store *customizingDBStore) Save(roleID role.RoleID, value string) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("customizing store: no database")
	}
	if roleID == 0 {
		return errors.New("customizing store: zero role id")
	}
	ctx, cancel := context.WithTimeout(context.Background(), customConfigQueryTimeout)
	defer cancel()
	if _, err := store.db.ExecContext(ctx,
		`INSERT INTO role_customizing (role_id, customizing_value) VALUES (?, ?)
ON DUPLICATE KEY UPDATE customizing_value = VALUES(customizing_value)`,
		roleID, []byte(value)); err != nil {
		return fmt.Errorf("persist customizing role=%d: %w", roleID, err)
	}
	return nil
}

// keybindDBStore 持久化键位主串/方案一/方案二与操作模式
// （role_keybinds 单行）。bind_value NOT NULL 且无默认值：任何不含
// bind_value 的首插都必须补 ”，否则整行插入失败——Java 参考服真机
// 踩实过的坑（RoleInventoryStore.saveKeybindScheme）。
type keybindDBStore struct {
	db *sql.DB
}

func newKeybindDBStore(db *sql.DB) *keybindDBStore {
	return &keybindDBStore{db: db}
}

// Load 读取键位域整行；operate_mode NULL 表示从未设置（不下发，
// 客户端保持本地默认 2.5D）。无行或 DB 错误按「无存档」处理。
func (store *keybindDBStore) Load(roleID role.RoleID) (playerKeybind, bool) {
	var record playerKeybind
	if store == nil || store.db == nil || roleID == 0 {
		return record, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), customConfigQueryTimeout)
	defer cancel()
	var operateMode sql.NullInt32
	err := store.db.QueryRowContext(ctx,
		`SELECT bind_value, shortcut_key1, shortcut_key2, operate_mode FROM role_keybinds WHERE role_id = ?`,
		roleID).Scan(&record.bind, &record.scheme1, &record.scheme2, &operateMode)
	if errors.Is(err, sql.ErrNoRows) {
		return record, false
	}
	if err != nil {
		log.Printf("keybind store: load role=%d: %v", roleID, err)
		return record, false
	}
	if operateMode.Valid {
		record.operateMode = operateMode.Int32
		record.operateModeSet = true
	}
	return record, true
}

// SaveScheme 落一份键位串：scheme 0=当前启用（bind_value）、
// 1=方案一（shortcut_key1）、2=方案二（shortcut_key2）。
func (store *keybindDBStore) SaveScheme(roleID role.RoleID, scheme int32, value string) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("keybind store: no database")
	}
	if roleID == 0 {
		return errors.New("keybind store: zero role id")
	}
	var statement string
	var arguments []any
	switch scheme {
	case 0:
		// scheme 0 的目标列即 INSERT 列单里的 bind_value，两处列名
		// 必须只出现一次，否则 MySQL 报 "Column specified twice"。
		statement = `INSERT INTO role_keybinds (role_id, bind_value) VALUES (?, ?)
ON DUPLICATE KEY UPDATE bind_value = VALUES(bind_value)`
		arguments = []any{roleID, value}
	case 1:
		statement = `INSERT INTO role_keybinds (role_id, bind_value, shortcut_key1) VALUES (?, '', ?)
ON DUPLICATE KEY UPDATE shortcut_key1 = VALUES(shortcut_key1)`
		arguments = []any{roleID, value}
	case 2:
		statement = `INSERT INTO role_keybinds (role_id, bind_value, shortcut_key2) VALUES (?, '', ?)
ON DUPLICATE KEY UPDATE shortcut_key2 = VALUES(shortcut_key2)`
		arguments = []any{roleID, value}
	default:
		return fmt.Errorf("keybind store: unknown scheme %d", scheme)
	}
	ctx, cancel := context.WithTimeout(context.Background(), customConfigQueryTimeout)
	defer cancel()
	if _, err := store.db.ExecContext(ctx, statement, arguments...); err != nil {
		return fmt.Errorf("persist keybind role=%d scheme=%d: %w", roleID, scheme, err)
	}
	return nil
}

// SaveOperateMode 落 3D/2.5D 操作模式（0=3D、1=2.5D）。
func (store *keybindDBStore) SaveOperateMode(roleID role.RoleID, mode int32) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("keybind store: no database")
	}
	if roleID == 0 {
		return errors.New("keybind store: zero role id")
	}
	ctx, cancel := context.WithTimeout(context.Background(), customConfigQueryTimeout)
	defer cancel()
	if _, err := store.db.ExecContext(ctx,
		`INSERT INTO role_keybinds (role_id, bind_value, operate_mode) VALUES (?, '', ?)
ON DUPLICATE KEY UPDATE operate_mode = VALUES(operate_mode)`,
		roleID, mode); err != nil {
		return fmt.Errorf("persist operate mode role=%d: %w", roleID, err)
	}
	return nil
}

// shortcutDBStore 持久化玩家自编辑快捷栏（role_shortcuts，
// 主键 (role_id, seq)，shortcut_index 是可见栏位索引）。
type shortcutDBStore struct {
	db *sql.DB
}

func newShortcutDBStore(db *sql.DB) *shortcutDBStore {
	return &shortcutDBStore{db: db}
}

// List 按 shortcut_index 升序回放快捷栏基线（内存表顺序与栏位序
// 一致，grantShortcutRows 逐行重放 0x11）。
func (store *shortcutDBStore) List(roleID role.RoleID) ([]playerShortcut, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), customConfigQueryTimeout)
	defer cancel()
	rows, err := store.db.QueryContext(ctx,
		`SELECT shortcut_index, kind, id FROM role_shortcuts WHERE role_id = ? ORDER BY shortcut_index`,
		roleID)
	if err != nil {
		log.Printf("shortcut store: list role=%d: %v", roleID, err)
		return nil, false
	}
	defer rows.Close()
	var shortcuts []playerShortcut
	for rows.Next() {
		var index int64
		var kind, id string
		if err := rows.Scan(&index, &kind, &id); err != nil {
			log.Printf("shortcut store: scan role=%d: %v", roleID, err)
			return nil, false
		}
		shortcuts = append(shortcuts, playerShortcut{index: int32(index), kind: kind, id: id})
	}
	if err := rows.Err(); err != nil {
		log.Printf("shortcut store: iterate role=%d: %v", roleID, err)
		return nil, false
	}
	return shortcuts, true
}

// Upsert 写一个栏位：已有 shortcut_index 原位 UPDATE，否则追加
// seq=max+1。追加用 INSERT..SELECT 聚合保持原子，避免并发会话
// 抢同一 seq。
func (store *shortcutDBStore) Upsert(roleID role.RoleID, shortcut playerShortcut) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("shortcut store: no database")
	}
	if roleID == 0 {
		return errors.New("shortcut store: zero role id")
	}
	ctx, cancel := context.WithTimeout(context.Background(), customConfigQueryTimeout)
	defer cancel()
	result, err := store.db.ExecContext(ctx,
		`UPDATE role_shortcuts SET kind = ?, id = ? WHERE role_id = ? AND shortcut_index = ?`,
		shortcut.kind, shortcut.id, roleID, shortcut.index)
	if err != nil {
		return fmt.Errorf("update shortcut role=%d index=%d: %w", roleID, shortcut.index, err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected > 0 {
		return nil
	}
	if _, err := store.db.ExecContext(ctx,
		`INSERT INTO role_shortcuts (role_id, seq, shortcut_index, kind, id)
SELECT ?, IFNULL(MAX(seq), 0) + 1, ?, ?, ? FROM role_shortcuts WHERE role_id = ?`,
		roleID, shortcut.index, shortcut.kind, shortcut.id, roleID); err != nil {
		return fmt.Errorf("append shortcut role=%d index=%d: %w", roleID, shortcut.index, err)
	}
	return nil
}

// DeleteIndex 按可见栏位索引删除一行（客户端 C2S 201 传的就是
// 栏位索引，不是记录行号）。
func (store *shortcutDBStore) DeleteIndex(roleID role.RoleID, index int32) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("shortcut store: no database")
	}
	if roleID == 0 {
		return errors.New("shortcut store: zero role id")
	}
	ctx, cancel := context.WithTimeout(context.Background(), customConfigQueryTimeout)
	defer cancel()
	if _, err := store.db.ExecContext(ctx,
		`DELETE FROM role_shortcuts WHERE role_id = ? AND shortcut_index = ?`,
		roleID, index); err != nil {
		return fmt.Errorf("delete shortcut role=%d index=%d: %w", roleID, index, err)
	}
	return nil
}
