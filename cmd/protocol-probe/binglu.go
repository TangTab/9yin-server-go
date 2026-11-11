package main

// 兵录域（binglu_rank_rec=178 [2,2,2]；Java BingluStore/applyBinglu 蓝本 +
// 客户端 form_binglu.lua / form_new_binglu.lua / form_binglu_chakan.lua 方言，
// 2026-11-11）：
//
// 数据下发：12 行密集（行号=槽位，槽序=binglubaseinfo.ini 节序：徒手/
// 单刀/单剑/匕首/双刀/双剑/双刺/长棍/短棍/笛(ditze，UI 跳过但行保留)/
// 暗器/奇门），列 [CurLevel,FillValue,MaxLevel]。零进度（全默认行）
// 0 帧——官服对拍口径：零进度角色捕获无兵录行。面板域 MaxBingLuLevel@
// 2154 = 各槽 cur 最大值；BingLuType@2153 恒 0（官服未观测到非零下发）。
//
// 使用链（C2S）：
//   - 943 MODIFY_BINGLU(sub, item_id, binglu_type, add_num)：sub=0
//     ADD_MAX_LEVEL——BingLuExpandInfo.ini [当前上限] 档耗修为
//     （use_faculty，如 [10] 200 万修为 → 15）；其余子令日志静默。
//   - 984 GET_BINGLU_INFO(name)：查看他人兵录 → 命名 custom
//     custom_show_binglu_info [name, cur,fill,max ×12 摊平]；-1=查无
//    （一期回本会话玩家自己的 12 行，跨玩家查询二期）。
//
// 存档：role_binglu.snapshot（JSON 数组 [[cur,fill,max]×12] 密集，role_
// mounts 同方言；Java 侧逐行表 schema 0002，本服统一快照列）。缺行/
// 缺表/解码失败回默认 12 行 [0,0,10]。
//
// GM binglu：`set <row> <cur> <fill> [max]` / `max [row]`（Java
// applyBinglu 同语义：上限缺省不改；升满=cur 置有效上限、进度清零）。

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

// C2S 消息号（client_custom_define.lua:642/677）。
const (
	clientCustomModifyBinglu int32 = 943 // CLIENT_CUSTOMMSG_MODIFY_BINGLU
	clientCustomGetBinglu    int32 = 984 // CLIENT_CUSTOMMSG_GET_BINGLU_INFO
)

// 记录号与槽位常量。
const (
	recordBingluRank = 178 // 官方 447 schema：binglu_rank_rec [2,2,2]

	bingluSlotCount       = 12
	bingluDefaultMaxLevel = 10 // binglubaseinfo.ini 全 12 节 MaxLevel
)

// bingluSlotNames 槽位中文名（binglubaseinfo.ini 节序；9=ditze 客户端
// UI 不展示但行保留——"注意不要修改武器的配置顺序"）。
var bingluSlotNames = [bingluSlotCount]string{
	"徒手", "单刀", "单剑", "匕首", "双刀", "双剑",
	"双刺", "长棍", "短棍", "笛", "暗器", "奇门",
}

// bingluSlotName 槽位文案（越界回退）。
func bingluSlotName(index int) string {
	if index >= 0 && index < bingluSlotCount {
		return bingluSlotNames[index]
	}
	return "槽位" + strconv.Itoa(index)
}

// bingluRow 单槽进度（binglu_rank_rec 行镜像）。
type bingluRow struct {
	Cur  int32 `json:"cur"`
	Fill int64 `json:"fill"`
	Max  int32 `json:"max"`
}

// effectiveMax 单行上限（无上限行保守默认 10）。
func (row bingluRow) effectiveMax() int32 {
	if row.Max > 0 {
		return row.Max
	}
	return bingluDefaultMaxLevel
}

// bingluDefaults 默认密集 12 行 [0,0,10]。
func bingluDefaults() []bingluRow {
	rows := make([]bingluRow, bingluSlotCount)
	for index := range rows {
		rows[index] = bingluRow{Max: bingluDefaultMaxLevel}
	}
	return rows
}

// bingluHasProgress 零进度判定（全默认行=0 帧，官服对拍口径）。
func bingluHasProgress(rows []bingluRow) bool {
	for _, row := range rows {
		if row.Cur > 0 || row.Fill > 0 || row.Max != bingluDefaultMaxLevel {
			return true
		}
	}
	return false
}

// bingluMaxCur 各槽 cur 最大值（MaxBingLuLevel 属性值源）。
func bingluMaxCur(rows []bingluRow) int32 {
	maxCur := int32(0)
	for _, row := range rows {
		if row.Cur > maxCur {
			maxCur = row.Cur
		}
	}
	return maxCur
}

// ---- 扩档目录（BingLuExpandInfo.ini）----

// bingluExpandTier 升档条件（[当前上限] 节）。
type bingluExpandTier struct {
	useFaculty int64 // 修为消耗
	newMax     int32 // 提升后上限
}

// bingluExpandCatalog 当前上限 → 升档（当前资源仅 [10]：200 万修为 → 15）。
var (
	bingluExpandOnce    sync.Once
	bingluExpandCatalog map[int32]bingluExpandTier
)

// currentBingluExpandCatalog 懒加载扩档目录（weaponRecast 同模式；
// 测试注入优先——setBingluExpandCatalogForTest）。
func currentBingluExpandCatalog() map[int32]bingluExpandTier {
	bingluExpandOnce.Do(func() {
		if bingluExpandCatalog != nil {
			return
		}
		bingluExpandCatalog = loadBingluExpandCatalog()
	})
	return bingluExpandCatalog
}

// setBingluExpandCatalogForTest 单测注入合成扩档表（生产不得调用）。
func setBingluExpandCatalogForTest(tiers map[int32]bingluExpandTier) {
	bingluExpandOnce.Do(func() {})
	bingluExpandCatalog = tiers
}

// loadBingluExpandCatalog 装载 BingLuExpandInfo.ini（[当前上限] 节）。
func loadBingluExpandCatalog() map[int32]bingluExpandTier {
	tiers := make(map[int32]bingluExpandTier)
	file, err := openResourceReader(filepath.Join(defaultModernShareRoot,
		"Skill", "BingLu", "BingLuExpandInfo.ini"))
	if err != nil {
		log.Printf("binglu expand catalog unavailable: %v", err)
		return tiers
	}
	defer file.Close()
	parseINISections(file, func(section, key, value string) {
		curMax, err := strconv.ParseInt(strings.TrimSpace(section), 10, 32)
		if err != nil {
			return
		}
		tier := tiers[int32(curMax)]
		switch strings.ToLower(key) {
		case "use_faculty":
			if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
				tier.useFaculty = parsed
			}
		case "new_max_level":
			if parsed, err := strconv.ParseInt(value, 10, 32); err == nil {
				tier.newMax = int32(parsed)
			}
		}
		tiers[int32(curMax)] = tier
	})
	return tiers
}

// ---- 存档 ----

const bingluDBQueryTimeout = 5 * time.Second

// bingluDBStore role_binglu 快照存取（role_mounts 同方言）。
type bingluDBStore struct{ db *sql.DB }

func newBingluDBStore(db *sql.DB) *bingluDBStore { return &bingluDBStore{db: db} }

// Load 读快照（[[cur,fill,max]×12]）；无行/缺表/解码失败/行数不足回默认。
func (store *bingluDBStore) Load(roleID role.RoleID) []bingluRow {
	rows := bingluDefaults()
	if store == nil || store.db == nil || roleID == 0 {
		return rows
	}
	ctx, cancel := context.WithTimeout(context.Background(), bingluDBQueryTimeout)
	defer cancel()
	var raw []byte
	err := store.db.QueryRowContext(ctx,
		`SELECT snapshot FROM role_binglu WHERE role_id = ?`, roleID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return rows
	}
	if err != nil {
		log.Printf("binglu store: load role=%d: %v", roleID, err)
		return rows
	}
	var decoded []bingluRow
	if err := json.Unmarshal(raw, &decoded); err != nil {
		log.Printf("binglu store: decode role=%d snapshot: %v", roleID, err)
		return rows
	}
	for index := 0; index < bingluSlotCount && index < len(decoded); index++ {
		rows[index] = decoded[index]
	}
	return rows
}

// Save upsert 快照（GM binglu / 943 扩档后落库；失败由调用方记日志）。
func (store *bingluDBStore) Save(roleID role.RoleID, rows []bingluRow) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("binglu store: no database")
	}
	if roleID == 0 {
		return errors.New("binglu store: zero role id")
	}
	raw, err := json.Marshal(rows)
	if err != nil {
		return fmt.Errorf("encode binglu snapshot: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), bingluDBQueryTimeout)
	defer cancel()
	if _, err := store.db.ExecContext(ctx, `INSERT INTO role_binglu (role_id, snapshot, updated_at)
VALUES (?, ?, UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE snapshot = VALUES(snapshot), updated_at = UTC_TIMESTAMP(6)`,
		roleID, raw); err != nil {
		return fmt.Errorf("persist binglu role=%d: %w", roleID, err)
	}
	return nil
}

// ---- 会话助手（p.mu 保护）----

func (p *playerActor) restoreBinglu(rows []bingluRow) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(rows) == bingluSlotCount {
		p.binglu = rows
	} else {
		p.binglu = bingluDefaults()
	}
}

func (p *playerActor) bingluSnapshot() []bingluRow {
	p.mu.Lock()
	defer p.mu.Unlock()
	snapshot := make([]bingluRow, len(p.binglu))
	copy(snapshot, p.binglu)
	return snapshot
}

func (p *playerActor) replaceBinglu(rows []bingluRow) {
	p.mu.Lock()
	p.binglu = rows
	p.mu.Unlock()
}

// resetBingluRows 0x0C 重入后行记账复位（整表重放前调）。
func (p *playerActor) resetBingluRows() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.bingluRowsSent = 0
}

// trySpendFaculty 修为扣减（Faculty@798；不足不动账）。返回扣减后值
// 与 Faculty 单属性帧（0x10——不走 vitalUpdate 全量，避免对未完整初始化
// 会话的重依赖；修为单值变化单帧足够）。
func (p *playerActor) trySpendFaculty(cost int64) (bool, int32, []byte, error) {
	p.mu.Lock()
	if int64(p.progress.faculty) < cost {
		faculty := p.progress.faculty
		p.mu.Unlock()
		return false, faculty, nil, nil
	}
	p.progress.faculty -= int32(cost)
	faculty := p.progress.faculty
	p.mu.Unlock()
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propFaculty, Name: "Faculty", Value: clientdata.Int32Value(faculty)},
	})
	return true, faculty, frame, err
}

// ---- 帧族 ----

// bingluRefreshFrames 兵录整刷：逆序删旧行 → 12 行密集重推 →
// MaxBingLuLevel 属性帧。返回新行数（下轮清场记账）。
func bingluRefreshFrames(link sceneMessageConnection, rows []bingluRow, oldRows int) (int, error) {
	for row := oldRows - 1; row >= 0; row-- {
		if err := link.WriteFrame(serverRecordDelRow(playerObjectID, playerOwnerID, recordBingluRank, uint16(row))); err != nil {
			return oldRows, fmt.Errorf("clear binglu row %d: %w", row, err)
		}
	}
	for _, row := range rows {
		frame, err := serverRecordAddCells(playerObjectID, playerOwnerID, recordBingluRank, []recordCell{
			recordInt(uint32(row.Cur)), recordInt(uint32(row.Fill)), recordInt(uint32(row.Max)),
		})
		if err != nil {
			return oldRows, fmt.Errorf("encode binglu row: %w", err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return oldRows, fmt.Errorf("write binglu row: %w", err)
		}
	}
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propMaxBingLuLevel, Name: "MaxBingLuLevel", Value: clientdata.Int32Value(bingluMaxCur(rows))},
	})
	if err != nil {
		return len(rows), fmt.Errorf("encode MaxBingLuLevel: %w", err)
	}
	if err := link.WriteFrame(frame); err != nil {
		return len(rows), fmt.Errorf("write MaxBingLuLevel: %w", err)
	}
	return len(rows), nil
}

// pushBingluFrames 会话整推入口（行记账走 player.bingluRowsSent）。
func pushBingluFrames(link sceneMessageConnection, player *playerActor) error {
	player.mu.Lock()
	oldRows := player.bingluRowsSent
	rows := make([]bingluRow, len(player.binglu))
	copy(rows, player.binglu)
	player.mu.Unlock()
	sent, err := bingluRefreshFrames(link, rows, oldRows)
	if err != nil {
		return err
	}
	player.mu.Lock()
	player.bingluRowsSent = sent
	player.mu.Unlock()
	return nil
}

// ---- C2S 943 / 984 ----

// bingluSession 兵录域会话上下文（mountSession 同款式参数包）。
type bingluSession struct {
	link   sceneMessageConnection
	player *playerActor
	store  *bingluDBStore
	roleID role.RoleID
	remote string
}

// handleModifyBingluCustom 943 入口：sub=0 ADD_MAX_LEVEL——当前有效上限
// 命中扩档目录 → 扣修为 → 上限提升 → 落档 + 整刷。
func handleModifyBingluCustom(m *bingluSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 ||
		custom.Values[0].Int32 != clientCustomModifyBinglu {
		return false, nil
	}
	if m == nil || m.player == nil {
		return true, nil
	}
	if len(custom.Values) < 4 || custom.Values[1].Type != 2 {
		log.Printf("%s: modify binglu 943 malformed args=%d", m.remote, len(custom.Values))
		return true, nil
	}
	sub := custom.Values[1].Int32
	slot, ok := custom.Values[3].exactInt32()
	if !ok || slot < 0 || slot >= bingluSlotCount {
		log.Printf("%s: modify binglu sub=%d slot=%d invalid", m.remote, sub, slot)
		return true, nil
	}
	switch sub {
	case 0: // ADD_MAX_LEVEL
		rows := m.player.bingluSnapshot()
		tier, known := currentBingluExpandCatalog()[rows[slot].effectiveMax()]
		if !known || tier.newMax <= 0 {
			log.Printf("%s: modify binglu slot=%d max=%d no expand tier", m.remote, slot, rows[slot].effectiveMax())
			return true, nil
		}
		paid, _, frame, err := m.player.trySpendFaculty(tier.useFaculty)
		if err == nil && frame != nil {
			if writeErr := m.link.WriteFrame(frame); writeErr != nil {
				log.Printf("%s: modify binglu faculty frame: %v", m.remote, writeErr)
			}
		}
		if !paid {
			log.Printf("%s: modify binglu slot=%d needs faculty %d", m.remote, slot, tier.useFaculty)
			return true, nil
		}
		rows[slot].Max = tier.newMax
		m.player.replaceBinglu(rows)
		m.persistBinglu("modify binglu expand")
		if err := pushBingluFrames(m.link, m.player); err != nil {
			log.Printf("%s: modify binglu refresh: %v", m.remote, err)
		}
		log.Printf("%s: binglu expand slot=%d(%s) max %d -> %d faculty=%d",
			m.remote, slot, bingluSlotName(int(slot)), rows[slot].effectiveMax(), tier.newMax, tier.useFaculty)
	default:
		log.Printf("%s: modify binglu sub=%d not implemented (slot=%d)", m.remote, sub, slot)
	}
	return true, nil
}

// handleGetBingluCustom 984 入口：查看兵录 → custom_show_binglu_info
// [name, cur,fill,max ×12 摊平]；-1=查无（一期仅本会话玩家）。
func handleGetBingluCustom(m *bingluSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 ||
		custom.Values[0].Int32 != clientCustomGetBinglu {
		return false, nil
	}
	if m == nil || m.player == nil {
		return true, nil
	}
	name := ""
	if len(custom.Values) > 1 {
		name = strings.TrimSpace(custom.Values[1].Text)
	}
	queried := m.player.name
	if name == "" || name == queried {
		values := []serverCustomValue{customString(queried)}
		for _, row := range m.player.bingluSnapshot() {
			values = append(values,
				customInt(row.Cur), customInt(int32(row.Fill)), customInt(row.Max))
		}
		if frame, err := serverCustomStringMessage("custom_show_binglu_info", values...); err == nil {
			if err := m.link.WriteFrame(frame); err != nil {
				log.Printf("%s: get binglu write: %v", m.remote, err)
			}
		} else {
			log.Printf("%s: get binglu encode: %v", m.remote, err)
		}
		log.Printf("%s: get binglu info self", m.remote)
		return true, nil
	}
	// 查他人：单机服无跨会话目录，按查无回执（客户端 -1 → 提示文案 7910）。
	if frame, err := serverCustomStringMessage("custom_show_binglu_info", customInt(-1)); err == nil {
		if err := m.link.WriteFrame(frame); err != nil {
			log.Printf("%s: get binglu miss write: %v", m.remote, err)
		}
	}
	log.Printf("%s: get binglu info %q not found (single-node)", m.remote, name)
	return true, nil
}

// persistBinglu 落档（失败仅记日志——内存/帧已生效）。
func (m *bingluSession) persistBinglu(reason string) {
	if m.store == nil || m.roleID == 0 {
		return
	}
	if err := m.store.Save(m.roleID, m.player.bingluSnapshot()); err != nil {
		log.Printf("%s: persist binglu after %s: %v", m.remote, reason, err)
	}
}

// applyBingluArgs GM binglu 子命令（Java applyBinglu 同语义）：
// `set <row> <cur> <fill> [max]`（上限缺省=不改）/ `max [row]`（缺省全部
// ——cur 置有效上限、进度清零；无上限行按默认 10 补写上限）。
// 返回（新行集，回显文案，是否成功）。
func applyBingluArgs(player *playerActor, args string) ([]bingluRow, string, bool) {
	parts := strings.Fields(strings.TrimSpace(args))
	rows := player.bingluSnapshot()
	usage := "用法: set <row> <cur> <fill> [max] / max [row]"
	if len(parts) == 0 {
		return nil, usage, false
	}
	atoi := func(raw string) (int, bool) {
		value, err := strconv.Atoi(raw)
		return value, err == nil
	}
	switch parts[0] {
	case "set":
		if len(parts) < 4 || len(parts) > 5 {
			return nil, "用法: set <row> <cur> <fill> [max]", false
		}
		row, ok1 := atoi(parts[1])
		cur, ok2 := atoi(parts[2])
		fill, ok3 := atoi(parts[3])
		if !ok1 || !ok2 || !ok3 || row < 0 || row >= bingluSlotCount {
			return nil, "row 须为 0-11 槽位（9=ditze 客户端不展示）", false
		}
		if cur < 0 || cur > 15 {
			return nil, "cur(等级) 须为 0-15", false
		}
		if fill < 0 || fill > 0xFFFF {
			return nil, "fill(进度) 须为 0-65535（binglu_rank_rec 列宽 u16）", false
		}
		maxLevel := rows[row].Max
		if len(parts) == 5 {
			parsed, ok := atoi(parts[4])
			if !ok || parsed < 0 || parsed > 15 {
				return nil, "max(上限) 须为 0-15", false
			}
			maxLevel = int32(parsed)
		}
		rows[row] = bingluRow{Cur: int32(cur), Fill: int64(fill), Max: maxLevel}
		return rows, fmt.Sprintf("兵录 %s[%d] %d/%d 进度 %d",
			bingluSlotName(row), row, cur, rows[row].effectiveMax(), fill), true
	case "max":
		if len(parts) > 2 {
			return nil, "用法: max [row]（row 0-11，缺省=全部）", false
		}
		from, to := 0, bingluSlotCount
		if len(parts) == 2 {
			row, ok := atoi(parts[1])
			if !ok || row < 0 || row >= bingluSlotCount {
				return nil, "用法: max [row]（row 0-11，缺省=全部）", false
			}
			from, to = row, row+1
		}
		for index := from; index < to; index++ {
			cap := rows[index].effectiveMax()
			rows[index] = bingluRow{Cur: cap, Fill: 0, Max: cap}
		}
		text := fmt.Sprintf("兵录 %s已升满（等级上限 %d）",
			bingluSlotName(from), rows[from].effectiveMax())
		if to-from == bingluSlotCount {
			text = "兵录 12 行已全部升满（等级上限 " +
				strconv.FormatInt(int64(rows[0].effectiveMax()), 10) + "）"
		}
		return rows, text, true
	default:
		return nil, "未知子命令: " + parts[0] + "（支持 set / max）", false
	}
}
