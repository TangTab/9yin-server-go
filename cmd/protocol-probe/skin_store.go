package main

// 皮肤衣橱存档（Java SkinStore 的 Go 移植，MySQL-only：Java 的
// data/skins.json 主档在 Go 侧按 customConfigStores 惯例省略，role_skins
// 单源）：collected（收录位图源，升序去重）/ mounted（挂载列表，保序）。
// JSON 列以文本数组 "[]"、"[1,2]" 形式往返。

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"

	"github.com/local/9yin-go-server/internal/role"
)

// skinState 一名玩家的衣橱状态（playerActor.skins 的值类型）。
type skinState struct {
	collected []int32 // 升序去重（collectedContains 二分判定可换线性，量级千级）
	mounted   []int32 // 挂载序（后挂覆盖同部件；衣橱 UI 顺序敏感）
}

func (s skinState) clone() skinState {
	return skinState{
		collected: append([]int32(nil), s.collected...),
		mounted:   append([]int32(nil), s.mounted...),
	}
}

func (s skinState) collectedContains(id int32) bool {
	for _, collected := range s.collected {
		if collected == id {
			return true
		}
	}
	return false
}

func (s skinState) mountedContains(id int32) bool {
	for _, mounted := range s.mounted {
		if mounted == id {
			return true
		}
	}
	return false
}

// addCollected 收录（已收录幂等；保持升序去重）。
func (s *skinState) addCollected(id int32) {
	if s.collectedContains(id) {
		return
	}
	s.collected = append(s.collected, id)
	sort.Slice(s.collected, func(i, j int) bool { return s.collected[i] < s.collected[j] })
}

// removeMounted 卸载指定皮肤（未挂载幂等）。
func (s *skinState) removeMounted(id int32) {
	for index, mounted := range s.mounted {
		if mounted == id {
			s.mounted = append(s.mounted[:index], s.mounted[index+1:]...)
			return
		}
	}
}

// marshalSkinIDs 写库/线上数组形态：升序去重（Java writeIds 同口径）。
func marshalSkinIDs(ids []int32) string {
	seen := make(map[int32]bool, len(ids))
	unique := make([]int32, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	sort.Slice(unique, func(i, j int) bool { return unique[i] < unique[j] })
	if len(unique) == 0 {
		return "[]"
	}
	encoded, err := json.Marshal(unique)
	if err != nil {
		return "[]"
	}
	return string(encoded)
}

// parseSkinIDs 解析 JSON 列（空/坏行返回 nil）。
func parseSkinIDs(raw string) []int32 {
	trimmed := trimSpaceASCII(raw)
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	var ids []int32
	if err := json.Unmarshal([]byte(trimmed), &ids); err != nil {
		return nil
	}
	return ids
}

func trimSpaceASCII(value string) string {
	start, end := 0, len(value)
	for start < end && (value[start] == ' ' || value[start] == '\t' || value[start] == '\n' || value[start] == '\r') {
		start++
	}
	for end > start && (value[end-1] == ' ' || value[end-1] == '\t' || value[end-1] == '\n' || value[end-1] == '\r') {
		end--
	}
	return value[start:end]
}

// skinDBStore 持久化 role_skins（0006 迁移权威 DDL；降级同 customConfig）。
type skinDBStore struct {
	db *sql.DB
}

func newSkinDBStore(db *sql.DB) *skinDBStore {
	return &skinDBStore{db: db}
}

// Load 读取衣橱状态；无行/DB 错误按「无存档」处理（空态零帧）。
func (store *skinDBStore) Load(roleID role.RoleID) (skinState, bool) {
	var state skinState
	if store == nil || store.db == nil || roleID == 0 {
		return state, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), appearanceQueryTimeout)
	defer cancel()
	var collected, mounted string
	err := store.db.QueryRowContext(ctx,
		`SELECT collected, mounted FROM role_skins WHERE role_id = ?`, roleID).Scan(&collected, &mounted)
	if errors.Is(err, sql.ErrNoRows) {
		return state, false
	}
	if err != nil {
		log.Printf("skin store: load role=%d: %v", roleID, err)
		return state, false
	}
	state.collected = parseSkinIDs(collected)
	state.mounted = parseSkinIDs(mounted)
	return state, true
}

// Save upsert 衣橱（collected 落库升序去重、mounted 保内存序但写库去重
// 排序——Java writeIds 同口径，内存序权威在会话内）。
func (store *skinDBStore) Save(roleID role.RoleID, state skinState) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("skin store: no database")
	}
	if roleID == 0 {
		return errors.New("skin store: zero role id")
	}
	ctx, cancel := context.WithTimeout(context.Background(), appearanceQueryTimeout)
	defer cancel()
	if _, err := store.db.ExecContext(ctx,
		`INSERT INTO role_skins (role_id, collected, mounted, version, updated_at)
VALUES (?, CAST(? AS JSON), CAST(? AS JSON), 1, UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE collected = VALUES(collected), mounted = VALUES(mounted),
  version = version + 1, updated_at = UTC_TIMESTAMP(6)`,
		roleID, marshalSkinIDs(state.collected), marshalSkinIDs(state.mounted)); err != nil {
		return fmt.Errorf("persist skins role=%d: %w", roleID, err)
	}
	return nil
}

// ---- playerActor 衣橱访问器 ----

func (p *playerActor) restoreSkins(state skinState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.skins = state.clone()
}

// applySkins 原子替换衣橱状态并返回快照（挂载/卸载/收录后）。
func (p *playerActor) applySkins(state skinState) skinState {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.skins = state
	return p.skins.clone()
}

func (p *playerActor) skinSnapshot() skinState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.skins.clone()
}
