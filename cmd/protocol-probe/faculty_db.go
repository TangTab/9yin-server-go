package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

const facultyDBQueryTimeout = 5 * time.Second

// facultyDBStore 把内功进度快照持久化到共享库 role_faculty 表
// （role_id PK / snapshot JSON / version / updated_at）。读写 SQL 与
// Java 参考服 FacultyStore 逐字对齐（读写同表、upsert 递增 version），
// snapshot 的超集字段经 extras passthrough 保真，多服务端共存不互蚀。
type facultyDBStore struct {
	db *sql.DB
}

func newFacultyDBStore(db *sql.DB) *facultyDBStore {
	return &facultyDBStore{db: db}
}

// Load 读取角色内功进度；无行、DB 错误或坏快照都按“无存档”处理并
// 记日志（与旧文件存档缺失语义一致，登录链走默认初始化）。
func (store *facultyDBStore) Load(roleID role.RoleID) (facultyProgressSnapshot, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return facultyProgressSnapshot{}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), facultyDBQueryTimeout)
	defer cancel()
	var raw []byte
	err := store.db.QueryRowContext(ctx,
		`SELECT snapshot FROM role_faculty WHERE role_id = ?`, roleID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return facultyProgressSnapshot{}, false
	}
	if err != nil {
		log.Printf("faculty store: load role=%d: %v", roleID, err)
		return facultyProgressSnapshot{}, false
	}
	var snapshot facultyProgressSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		log.Printf("faculty store: decode role=%d snapshot: %v", roleID, err)
		return facultyProgressSnapshot{}, false
	}
	return snapshot, true
}

// Save upsert 快照（version 自增、updated_at 由库维护）。
func (store *facultyDBStore) Save(roleID role.RoleID, value facultyProgressSnapshot) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("faculty store: no database")
	}
	if roleID == 0 {
		return errors.New("faculty store: zero role id")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode faculty snapshot: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), facultyDBQueryTimeout)
	defer cancel()
	if _, err := store.db.ExecContext(ctx, `INSERT INTO role_faculty (role_id, snapshot, version, updated_at)
VALUES (?, ?, 1, UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE snapshot = VALUES(snapshot), version = version + 1, updated_at = UTC_TIMESTAMP(6)`,
		roleID, raw); err != nil {
		return fmt.Errorf("persist faculty role=%d: %w", roleID, err)
	}
	return nil
}
