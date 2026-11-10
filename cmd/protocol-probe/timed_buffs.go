package main

// 限时 buff 域（2026-11-10）：超长时限增益（官服登录期 buf_baosd_02 族）
// 的存储、到期结算与带到期时间的下发。
//
//   - 存储：role_timed_buffs 快照表（migrations/0019，role_currency 族
//     同形：role_id PK / snapshot JSON / version / updated_at），条目含
//     到期时刻（RFC3339 UTC）——重启/断线不丢，登录按未过期集合重放。
//   - 下发（两层，官服口径）：
//       ① m169 九值时间形态（glideTimedBuffFrame 同构，官服 52795
//          buf_qg_light 实证）：v7 = i64 到期 unix 毫秒——「数据下发中
//          也有到期时间」的直接载体；
//       ② BufferListStr 玩家属性（0x0D/vitals 档自带）：条目尾段即
//          到期 unix 毫秒，客户端 BuffContainer 图标倒计时数据源。
//   - 到期结算：1s ticker 检查（main.handle），跨越到期即产 m169 摘帧
//     （addRemove=0，v7 取原到期毫秒）并从运行态摘除；登录装载时已
//     过期条目直接丢弃（零帧）。
//
// 运行态与战斗槽位租约（p.buffs / nativeBuffSlots 1..24）分离：限时
// 增益按 configID 键控，BufferListStr 槽段用 200+ 序号避开战斗槽。

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// timedBuffSlotBase 限时 buff 在 BufferListStr 的槽段基址（避开
// nativeBuffSlots 1..24 战斗租约段；200+ 与任何引擎保留段无碰撞）。
const timedBuffSlotBase = 200

const timedBuffDBQueryTimeout = 5 * time.Second

// timedBuffEntry 是 role_timed_buffs.snapshot 的 JSON 条目形态。
type timedBuffEntry struct {
	ConfigID  string `json:"configId"`
	Level     int32  `json:"level"`
	Category  int32  `json:"category"`
	ExpiresAt string `json:"expiresAt"` // RFC3339 UTC（官服 m169 v7 同口径的绝对时刻）
}

// timedBuffDBStore 把限时 buff 清单持久化到 role_timed_buffs 快照表。
// 缺表/缺库：Load 返回无存档、Save 记日志跳过（建表权威在
// migrations/0019，运行时不做 DDL；降级 = 会话态，重启即失）。
type timedBuffDBStore struct {
	db *sql.DB
}

func newTimedBuffDBStore(db *sql.DB) *timedBuffDBStore {
	return &timedBuffDBStore{db: db}
}

// Load 读取整表条目；无行/缺表/坏 JSON 一律按“无存档”处理。
func (store *timedBuffDBStore) Load(ctx context.Context, roleID role.RoleID) ([]timedBuffEntry, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return nil, false
	}
	queryCtx, cancel := context.WithTimeout(ctx, timedBuffDBQueryTimeout)
	defer cancel()
	var snapshot []byte
	err := store.db.QueryRowContext(queryCtx,
		"SELECT snapshot FROM role_timed_buffs WHERE role_id = ?", uint64(roleID)).Scan(&snapshot)
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		log.Printf("timed buffs: load role=%d: %v", uint64(roleID), err)
		return nil, false
	}
	var entries []timedBuffEntry
	if err := json.Unmarshal(snapshot, &entries); err != nil {
		log.Printf("timed buffs: decode role=%d: %v", uint64(roleID), err)
		return nil, false
	}
	return entries, true
}

// Save 全量覆写快照（grant/remove 后调用；autosave 周期同口径兜底）。
func (store *timedBuffDBStore) Save(ctx context.Context, roleID role.RoleID, entries []timedBuffEntry) error {
	if store == nil || store.db == nil || roleID == 0 {
		return nil
	}
	payload, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("encode snapshot: %w", err)
	}
	queryCtx, cancel := context.WithTimeout(ctx, timedBuffDBQueryTimeout)
	defer cancel()
	_, err = store.db.ExecContext(queryCtx,
		"INSERT INTO role_timed_buffs (role_id, snapshot) VALUES (?, ?) "+
			"ON DUPLICATE KEY UPDATE snapshot = VALUES(snapshot), version = version + 1",
		uint64(roleID), payload)
	if err != nil {
		return fmt.Errorf("upsert role=%d: %w", uint64(roleID), err)
	}
	return nil
}

// timedBuffRuntime 运行态条目（含到期是否已通告，防 ticker 重复发摘帧）。
type timedBuffRuntime struct {
	configID   string
	level      int32
	category   int32
	expiresUTC time.Time
	expired    bool // 已跨越到期（摘帧已发或登录时即判过期）
}

// timedBuffSnapshot 导出未过期条目（落库/autosave 用；调用方持 p.mu 或
// 经锁封装）。
func (p *playerActor) timedBuffSnapshotLocked() []timedBuffEntry {
	entries := make([]timedBuffEntry, 0, len(p.timedBuffs))
	for _, buff := range p.timedBuffs {
		if buff.expired {
			continue
		}
		entries = append(entries, timedBuffEntry{
			ConfigID:  buff.configID,
			Level:     buff.level,
			Category:  buff.category,
			ExpiresAt: buff.expiresUTC.Format(time.RFC3339Nano),
		})
	}
	return entries
}

// restoreTimedBuffs 登录装载：解析快照并丢弃已过期条目（零帧，落库由
// 随后的 Save 收敛）。
func (p *playerActor) restoreTimedBuffs(entries []timedBuffEntry, now time.Time) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.timedBuffs = p.timedBuffs[:0]
	live := 0
	for _, entry := range entries {
		expires, err := time.Parse(time.RFC3339Nano, entry.ExpiresAt)
		if err != nil || !expires.After(now.UTC()) {
			continue
		}
		if entry.ConfigID == "" {
			continue
		}
		level := entry.Level
		if level < 1 {
			level = 1
		}
		p.timedBuffs = append(p.timedBuffs, timedBuffRuntime{
			configID: entry.ConfigID, level: level, category: entry.Category,
			expiresUTC: expires.UTC(),
		})
		live++
	}
	return live
}

// grantTimedBuff 授予/续期一条限时 buff（同 configID 覆盖到期与等级）。
// 返回新的到期时刻。
func (p *playerActor) grantTimedBuff(configID string, level, category int32, ttl time.Duration, now time.Time) time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	expires := now.UTC().Add(ttl)
	if level < 1 {
		level = 1
	}
	for index := range p.timedBuffs {
		if p.timedBuffs[index].configID == configID {
			p.timedBuffs[index].level = level
			p.timedBuffs[index].category = category
			p.timedBuffs[index].expiresUTC = expires
			p.timedBuffs[index].expired = false
			return expires
		}
	}
	p.timedBuffs = append(p.timedBuffs, timedBuffRuntime{
		configID: configID, level: level, category: category, expiresUTC: expires,
	})
	return expires
}

// removeTimedBuff 摘除一条（返回是否存在且未过期；到期毫秒供摘帧 v7）。
func (p *playerActor) removeTimedBuff(configID string, now time.Time) (int64, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for index := range p.timedBuffs {
		if p.timedBuffs[index].configID != configID {
			continue
		}
		if p.timedBuffs[index].expired {
			return 0, false
		}
		endMs := p.timedBuffs[index].expiresUTC.UnixMilli()
		p.timedBuffs = append(p.timedBuffs[:index], p.timedBuffs[index+1:]...)
		return endMs, true
	}
	return 0, false
}

// timedBuffActiveCount 未过期条数（GM 面板/日志用）。
func (p *playerActor) timedBuffActiveCount(now time.Time) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	count := 0
	for _, buff := range p.timedBuffs {
		if !buff.expired && buff.expiresUTC.After(now.UTC()) {
			count++
		}
	}
	return count
}

// timedBuffGrantFrame 授予帧：m169 九值时间形态（官服 buf_qg_light 实证
// 布局；v7=i64 到期 unix 毫秒，v8=Category）。
func timedBuffGrantFrame(buff timedBuffRuntime, playerName string) ([]byte, error) {
	return serverCustomIntMessage(169,
		customInt(1),
		customString(buff.configID),
		customObject(playerObjectID, playerOwnerID),
		customObject(playerObjectID, playerOwnerID),
		customWideString(playerName),
		customWideString(playerName),
		customInt64(buff.expiresUTC.UnixMilli()),
		customInt(buff.category),
	)
}

// timedBuffRemoveFrame 摘除帧（addRemove=0，v7 取原到期毫秒）。
func timedBuffRemoveFrame(configID string, endEpochMs int64, category int32, playerName string) ([]byte, error) {
	return serverCustomIntMessage(169,
		customInt(0),
		customString(configID),
		customObject(playerObjectID, playerOwnerID),
		customObject(playerObjectID, playerOwnerID),
		customWideString(playerName),
		customWideString(playerName),
		customInt64(endEpochMs),
		customInt(category),
	)
}

// timedBuffLoginReplayFrames 登录/就绪重放：未过期条目逐条授予帧
//（官服登录期 buf_baosd_02 语义的带时间形态对应物）。
func (p *playerActor) timedBuffLoginReplayFrames(now time.Time) ([][]byte, error) {
	p.mu.Lock()
	buffs := make([]timedBuffRuntime, 0, len(p.timedBuffs))
	for _, buff := range p.timedBuffs {
		if !buff.expired && buff.expiresUTC.After(now.UTC()) {
			buffs = append(buffs, buff)
		}
	}
	name := p.name
	p.mu.Unlock()
	frames := make([][]byte, 0, len(buffs))
	for _, buff := range buffs {
		frame, err := timedBuffGrantFrame(buff, name)
		if err != nil {
			return nil, fmt.Errorf("encode timed buff %s: %w", buff.configID, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// timedBuffExpiryFrames 到期结算（1s ticker）：跨越到期的条目产摘帧并
// 标记 expired（运行态保留至落库收敛，避免 ticker 与 Save 竞态丢条目）。
func (p *playerActor) timedBuffExpiryFrames(now time.Time) [][]byte {
	p.mu.Lock()
	var expired []timedBuffRuntime
	for index := range p.timedBuffs {
		if p.timedBuffs[index].expired || p.timedBuffs[index].expiresUTC.After(now.UTC()) {
			continue
		}
		p.timedBuffs[index].expired = true
		expired = append(expired, p.timedBuffs[index])
	}
	name := p.name
	p.mu.Unlock()
	if len(expired) == 0 {
		return nil
	}
	frames := make([][]byte, 0, len(expired))
	for _, buff := range expired {
		frame, err := timedBuffRemoveFrame(buff.configID, buff.expiresUTC.UnixMilli(), buff.category, name)
		if err != nil {
			log.Printf("timed buffs: encode removal %s: %v", buff.configID, err)
			continue
		}
		frames = append(frames, frame)
	}
	return frames
}

// appendTimedBuffsLocked 向 BufferListStr 追加限时段（调用方持 p.mu）。
// 格式与战斗租约条目同构，槽段 200+；到期毫秒即条目尾段时间列。
func (p *playerActor) appendTimedBuffsLocked(list *strings.Builder, now time.Time) {
	for index, buff := range p.timedBuffs {
		if buff.expired || !buff.expiresUTC.After(now.UTC()) {
			continue
		}
		fmt.Fprintf(list, "%s,local-%d,%d,%d-%d,%d,1,0|",
			buff.configID, timedBuffSlotBase+index, buff.level,
			playerObjectID, playerOwnerID, buff.expiresUTC.UnixMilli())
	}
}


