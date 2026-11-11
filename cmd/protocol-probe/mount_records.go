package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// 坐骑域（Java MountStore/MountMessages/pushSavedMounts 全链对齐）：
//
//   - 拥有集合：NewRideRec 310 官方列型 [3,3,6,1]（col0=坐骑包 StaticID、
//     col1=UsedDate、col2=f64 ValidTime、col3>0=当前选中）；整刷先 0x12
//     倒序删旧行再重推（CardRec 同方言）。
//   - 骑乘态：0x10 属性族 Mount@77（空=步行）、NewRideID@1333 当前坐骑包、
//     CurRiderType@82（MountType）。重登不跨骑乘态：Mount 恒空串，只发
//     NewRideID/CurRiderType（Java「重登即步行」口径）。
//
// 存档 = role_mounts.snapshot（JSON 数组，元素即 NewRideRec 列镜像：
// ConfigID/StaticID/UsedDate/ValidTime/Selected——role 618 官服迁移形态）。
// 无存档零帧。目录（CurRiderType 查表）：tool_item.ini script=Mount 段
// （MountType/ArtPack）+ itemartstatic.ini MaleModel（npc\ride_XXX，无模型
// 不可骑）+ share/ride/new_ride.ini MountType 兜底（Java MountCatalog 同序）。
const (
	recordNewRide uint16 = 310 // 官方 447 schema：NewRideRec

	mountDBQueryTimeout = 5 * time.Second
)

// 属性索引按名解析（0x09 表方言门控换表自动跟随；fork 位见各行注记）。
var (
	propNewRideID    = clientdata.MustPlayerProperty("NewRideID").Index    // 当前坐骑包（fork=1333）
	propCurRiderType = clientdata.MustPlayerProperty("CurRiderType").Index // MountType（fork=82）
)

var (
	defaultToolItemINIPath   = filepath.Join(defaultModernShareRoot, "item", "tool_item.ini")
	defaultItemArtStaticPath = filepath.Join(defaultModernShareRoot, "item", "itemartstatic.ini")
	defaultNewRideINIPath    = filepath.Join(defaultModernShareRoot, "ride", "new_ride.ini")
)

// mountEntry 单条坐骑（NewRideRec 行镜像；ConfigID 仅服务端便捷字段）。
type mountEntry struct {
	ConfigID  string  `json:"ConfigID"`
	PackID    uint32  `json:"StaticID"`
	UsedDate  uint32  `json:"UsedDate"`
	ValidTime float64 `json:"ValidTime"`
	Selected  bool    `json:"Selected"`
}

// withOwnedMount 保证拥有（不存在则补 ValidTime=0 永久口径；packId 升序
// 插入保持行序稳定）。纯函数：原切片不动，返回新切片。
func withOwnedMount(mounts []mountEntry, packID uint32, configID string) []mountEntry {
	for _, entry := range mounts {
		if entry.PackID == packID {
			return mounts
		}
	}
	next := make([]mountEntry, 0, len(mounts)+1)
	inserted := false
	for _, entry := range mounts {
		if !inserted && packID < entry.PackID {
			next = append(next, mountEntry{ConfigID: configID, PackID: packID})
			inserted = true
		}
		next = append(next, entry)
	}
	if !inserted {
		next = append(next, mountEntry{ConfigID: configID, PackID: packID})
	}
	return next
}

// withMountSelected 切换选中（至多一条 Selected；packId 不在集合中原样返回）。
func withMountSelected(mounts []mountEntry, packID uint32) []mountEntry {
	owned := false
	for _, entry := range mounts {
		if entry.PackID == packID {
			owned = true
			break
		}
	}
	if !owned {
		return mounts
	}
	next := make([]mountEntry, len(mounts))
	for index, entry := range mounts {
		next[index] = entry
		next[index].Selected = entry.PackID == packID
	}
	return next
}

// ownsMountPack 拥有判定。
func ownsMountPack(mounts []mountEntry, packID uint32) bool {
	for _, entry := range mounts {
		if entry.PackID == packID {
			return true
		}
	}
	return false
}

// currentPack 当前选中坐骑包（无选中 0）。
func mountCurrentPack(mounts []mountEntry) uint32 {
	for _, entry := range mounts {
		if entry.Selected {
			return entry.PackID
		}
	}
	return 0
}

type mountDBStore struct {
	db *sql.DB
}

func newMountDBStore(db *sql.DB) *mountDBStore {
	return &mountDBStore{db: db}
}

// Load 读 role_mounts 快照（裸 JSON 数组）；无行/缺表/解码失败返回空。
func (store *mountDBStore) Load(roleID role.RoleID) []mountEntry {
	if store == nil || store.db == nil || roleID == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), mountDBQueryTimeout)
	defer cancel()
	var raw []byte
	err := store.db.QueryRowContext(ctx,
		`SELECT snapshot FROM role_mounts WHERE role_id = ?`, roleID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		log.Printf("mount store: load role=%d: %v", roleID, err)
		return nil
	}
	mounts, err := unmarshalMountSnapshot(string(raw))
	if err != nil {
		log.Printf("mount store: decode role=%d snapshot: %v", roleID, err)
		return nil
	}
	return mounts
}

// unmarshalMountSnapshot 解析裸 JSON 数组快照（NewRideRec 列镜像）。
func unmarshalMountSnapshot(raw string) ([]mountEntry, error) {
	var mounts []mountEntry
	if err := json.Unmarshal([]byte(raw), &mounts); err != nil {
		return nil, err
	}
	return mounts, nil
}

// Save upsert 坐骑快照（mount_ride/266 骑乘链变更后落库；role_currency
// 同款方言，失败由调用方记日志不影响内存权威）。
func (store *mountDBStore) Save(roleID role.RoleID, mounts []mountEntry) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("mount store: no database")
	}
	if roleID == 0 {
		return errors.New("mount store: zero role id")
	}
	raw, err := json.Marshal(mounts)
	if err != nil {
		return fmt.Errorf("encode mount snapshot: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), mountDBQueryTimeout)
	defer cancel()
	if _, err := store.db.ExecContext(ctx, `INSERT INTO role_mounts (role_id, snapshot, updated_at)
VALUES (?, ?, UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE snapshot = VALUES(snapshot), updated_at = UTC_TIMESTAMP(6)`,
		roleID, raw); err != nil {
		return fmt.Errorf("persist mounts role=%d: %w", roleID, err)
	}
	return nil
}

// ---- 目录（packId → MountType；三表合并，Java MountCatalog 同序） ----

// mountDefinition 单条可骑坐骑（menu2 mountCatalog.load 同构）：
// tool_item script=Mount 主源（Name/MountType/RideSkillID 列表）+
// itemartstatic 模型（存在即骑）+ new_ride 兜底/补强（MountType/
// RideSkillID/SpeedQuality）。Name 取 GM 目录同源中文名（GBK）。
type mountDefinition struct {
	configID   string
	name       string
	packID     uint32  // ArtPack
	mountType  uint32   // CurRiderType 值源
	model      string   // npc\ride_XXX（引擎 ini\npc\<model>.ini 加载）
	speed      float64  // SpeedQuality（骑乘速度；未接下发，日志/GM 回显用）
	rideSkills []string // RideSkillID 逗号列表（211 出招闸白名单）
}

type mountCatalog struct {
	byPack map[uint32]uint32 // ArtPack → MountType（仅可骑条目）

	byConfig  map[string]*mountDefinition // 道具 config → 定义
	byPackDef map[uint32]*mountDefinition // ArtPack → 定义
	rideSkills map[string]struct{}        // 全目录骑乘技能合集（211 闸）
}

// defByConfig 道具 config 查定义（未知 nil）。
func (catalog *mountCatalog) defByConfig(configID string) *mountDefinition {
	if catalog == nil {
		return nil
	}
	return catalog.byConfig[configID]
}

// defByPack 坐骑包查定义（未知 nil）。
func (catalog *mountCatalog) defByPack(packID uint32) *mountDefinition {
	if catalog == nil {
		return nil
	}
	return catalog.byPackDef[packID]
}

// hasRideSkill 骑乘技能白名单判定（全目录合集；菜单/快捷栏技能随坐骑
// 授予，客户端任意时刻可发——按合集而非当前坐骑判定，menu2 merge 同口径）。
func (catalog *mountCatalog) hasRideSkill(skillID string) bool {
	if catalog == nil {
		return false
	}
	_, ok := catalog.rideSkills[skillID]
	return ok
}

// newRideMountSection new_ride.ini 单节兜底字段（MountType 主源在
// tool_item；RideSkillID/SpeedQuality 两表都可能有，合并去重）。
type newRideMountSection struct {
	mountType  uint32
	hasType    bool
	rideSkills []string
	speed      float64
	hasSpeed   bool
}

// loadMountCatalog 合并 itemartstatic（pack→模型，存在即可骑）+
// new_ride（pack→MountType/RideSkillID/SpeedQuality 兜底）+ tool_item
// script=Mount（MountType/Name/RideSkillID 主源）。与 Java 一致：
// ArtPack≤0 或无模型映射的条目不入表（不可骑）。
func loadMountCatalog(toolItemPath, itemArtStaticPath, newRidePath string) *mountCatalog {
	models := make(map[uint32]string)
	if file, err := openResourceReader(itemArtStaticPath); err == nil {
		parseINISections(file, func(section string, key, value string) {
			if pack, ok := parseUintSection(section); ok &&
				strings.EqualFold(key, "malemodel") && value != "" {
				models[pack] = value
			}
		})
		file.Close()
	} else {
		log.Printf("mount catalog: itemartstatic unavailable: %v", err)
	}
	newRide := make(map[uint32]newRideMountSection)
	if file, err := openResourceReader(newRidePath); err == nil {
		sections := make(map[uint32]*newRideMountSection)
		parseINISections(file, func(section string, key, value string) {
			pack, ok := parseUintSection(section)
			if !ok {
				return
			}
			entry := sections[pack]
			if entry == nil {
				entry = &newRideMountSection{}
				sections[pack] = entry
			}
			switch strings.ToLower(key) {
			case "mounttype":
				if parsed, err := strconv.ParseUint(value, 10, 32); err == nil {
					entry.mountType = uint32(parsed)
					entry.hasType = true
				}
			case "rideskillid":
				entry.rideSkills = append(entry.rideSkills, splitRideSkills(value)...)
			case "speedquality":
				if parsed, err := strconv.ParseFloat(value, 64); err == nil {
					entry.speed = parsed
					entry.hasSpeed = true
				}
			}
		})
		file.Close()
		for pack, entry := range sections {
			newRide[pack] = *entry
		}
	} else {
		log.Printf("mount catalog: new_ride unavailable: %v", err)
	}

	catalog := &mountCatalog{
		byPack:     make(map[uint32]uint32),
		byConfig:   make(map[string]*mountDefinition),
		byPackDef:  make(map[uint32]*mountDefinition),
		rideSkills: make(map[string]struct{}),
	}
	file, err := openResourceReader(toolItemPath)
	if err != nil {
		log.Printf("mount catalog: tool_item unavailable (CurRideType degrades to 0): %v", err)
		return catalog
	}
	defer file.Close()
	toolItemMounts(file, func(entry toolItemMountEntry) {
		if entry.artPack == 0 {
			return
		}
		model, ride := models[entry.artPack]
		if !ride || model == "" {
			return // 无模型映射的坐骑不可骑（Java 同款过滤）
		}
		def := &mountDefinition{
			configID:   entry.configID,
			name:       entry.name,
			packID:     entry.artPack,
			mountType:  entry.mountType,
			model:      model,
			rideSkills: entry.rideSkills,
		}
		if fallback := newRide[entry.artPack]; fallback.hasType && def.mountType == 0 {
			def.mountType = fallback.mountType
		}
		if fallback := newRide[entry.artPack]; len(fallback.rideSkills) > 0 {
			def.rideSkills = mergeRideSkills(def.rideSkills, fallback.rideSkills)
		}
		if fallback := newRide[entry.artPack]; fallback.hasSpeed {
			def.speed = fallback.speed
		}
		catalog.byPack[entry.artPack] = def.mountType
		catalog.byConfig[entry.configID] = def
		catalog.byPackDef[entry.artPack] = def
		for _, skill := range def.rideSkills {
			catalog.rideSkills[skill] = struct{}{}
		}
	})
	return catalog
}

// splitRideSkills 拆 RideSkillID 逗号/分号列表（空白项丢弃）。
func splitRideSkills(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t'
	})
	skills := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			skills = append(skills, part)
		}
	}
	return skills
}

// mergeRideSkills 合并两来源技能列表（保序去重，menu2 mergeMountSkills 同义）。
func mergeRideSkills(primary, secondary []string) []string {
	seen := make(map[string]struct{}, len(primary)+len(secondary))
	merged := make([]string, 0, len(primary)+len(secondary))
	for _, list := range [][]string{primary, secondary} {
		for _, skill := range list {
			if _, ok := seen[skill]; ok {
				continue
			}
			seen[skill] = struct{}{}
			merged = append(merged, skill)
		}
	}
	return merged
}

// riderTypeOf CurRiderType 查表（未知包 0）。
func (catalog *mountCatalog) riderTypeOf(packID uint32) uint32 {
	if catalog == nil {
		return 0
	}
	return catalog.byPack[packID]
}

// parseUintSection 解析数字小节名（非数字小节 false）。
func parseUintSection(section string) (uint32, bool) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(section), 10, 32)
	if err != nil {
		return 0, false
	}
	return uint32(parsed), true
}

// parseINISections 逐行回调 (section, key, value)；小节外/注释行跳过。
// 与 jingmai 目录同一方言（GBK 文件只读 ASCII 字段无损）。
func parseINISections(file io.Reader, visit func(section, key, value string)) {
	scanner := bufio.NewScanner(file)
	section := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.Trim(line, "[]")
			continue
		}
		if section == "" {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		visit(section, strings.TrimSpace(key), strings.TrimSpace(value))
	}
}

// toolItemMountEntry tool_item.ini script=Mount 单条聚合（Name 中文名
// GBK 解码后读取——流式 transform 包一层，方言与 gm_item_catalog 同源）。
type toolItemMountEntry struct {
	configID   string
	name       string
	mountType  uint32
	artPack    uint32
	rideSkills []string
}

// toolItemMounts 流式扫 tool_item.ini 的 script=Mount 条目（GBK 大表，
// 10 万+ 节——GBK→UTF-8 流式转码后逐节聚合 Name/MountType/ArtPack/
// RideSkillID/script）。configID=小节名。
func toolItemMounts(file io.Reader, visit func(entry toolItemMountEntry)) {
	type mountItem struct {
		script    string
		name      string
		hasName   bool
		mountType uint32
		artPack   uint32
		skills    []string
	}
	entries := make(map[string]*mountItem)
	parseINISections(gbkReader(file), func(section, key, value string) {
		if strings.HasPrefix(section, "//") || strings.HasPrefix(section, ";") {
			return // 注释节
		}
		entry, ok := entries[section]
		if !ok {
			entry = &mountItem{}
			entries[section] = entry
		}
		switch strings.ToLower(key) {
		case "script":
			entry.script = value
		case "name":
			if !entry.hasName {
				entry.name = value
				entry.hasName = true
			}
		case "mounttype":
			if parsed, err := strconv.ParseUint(value, 10, 32); err == nil {
				entry.mountType = uint32(parsed)
			}
		case "artpack":
			if parsed, err := strconv.ParseUint(value, 10, 32); err == nil {
				entry.artPack = uint32(parsed)
			}
		case "rideskillid":
			entry.skills = append(entry.skills, splitRideSkills(value)...)
		}
	})
	for configID, entry := range entries {
		if !strings.EqualFold(entry.script, "Mount") {
			continue
		}
		visit(toolItemMountEntry{
			configID:   configID,
			name:       entry.name,
			mountType:  entry.mountType,
			artPack:    entry.artPack,
			rideSkills: entry.skills,
		})
	}
}

// gbkReader GBK→UTF-8 流式转码（GM 物品目录同款 decoder；转码错误
// 以替换符吞掉，键/节名 ASCII 无损）。
func gbkReader(file io.Reader) io.Reader {
	return transform.NewReader(file, simplifiedchinese.GBK.NewDecoder())
}

// ---- 帧族 ----

// mountOwnedClearFrames 倒序删除已存在的拥有行（整刷前清场）。
func mountOwnedClearFrames(link sceneMessageConnection, oldRows int) error {
	for row := oldRows - 1; row >= 0; row-- {
		if err := link.WriteFrame(serverRecordDelRow(playerObjectID, playerOwnerID, recordNewRide, uint16(row))); err != nil {
			return fmt.Errorf("clear mount row %d: %w", row, err)
		}
	}
	return nil
}

// mountRideStateFrame 骑乘态单帧 0x10：上马（model 非空 npc\ride_XXX +
// pack/riderType 实值）或下马/重登步行（model 空串 + 双 0）。
func mountRideStateFrame(model string, packID, riderType uint32) ([]byte, error) {
	if model == "" {
		packID, riderType = 0, 0
	}
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propMount, Name: "Mount", Value: clientdata.StringValue(model)},
		{Index: propNewRideID, Name: "NewRideID", Value: clientdata.Int32Value(int32(packID))},
		{Index: propCurRiderType, Name: "CurRiderType", Value: clientdata.Int32Value(int32(riderType))},
	})
}

// pushMountFrames 坐骑整推：清旧 NewRideRec 行 → 重推行（Selected 位随行）
// → 骑乘态单帧。返回新行数（供下轮清场；空库零帧）。
func pushMountFrames(link sceneMessageConnection, mounts []mountEntry, catalog *mountCatalog, oldRows int) (int, error) {
	if err := mountOwnedClearFrames(link, oldRows); err != nil {
		return oldRows, err
	}
	for _, entry := range mounts {
		selected := uint32(0)
		if entry.Selected {
			selected = 1
		}
		frame, err := serverRecordAddCells(playerObjectID, playerOwnerID, recordNewRide, []recordCell{
			recordInt(entry.PackID),
			recordInt(entry.UsedDate),
			recordFloat64(entry.ValidTime),
			recordByte(selected),
		})
		if err != nil {
			return oldRows, fmt.Errorf("encode mount row %s: %w", entry.ConfigID, err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return oldRows, fmt.Errorf("write mount row %s: %w", entry.ConfigID, err)
		}
	}
	if len(mounts) == 0 {
		return 0, nil
	}
	currentPack := mountCurrentPack(mounts)
	rideState, err := mountRideStateFrame("", currentPack, catalog.riderTypeOf(currentPack))
	if err != nil {
		return len(mounts), fmt.Errorf("encode ride state: %w", err)
	}
	if err := link.WriteFrame(rideState); err != nil {
		return len(mounts), fmt.Errorf("write ride state: %w", err)
	}
	return len(mounts), nil
}
