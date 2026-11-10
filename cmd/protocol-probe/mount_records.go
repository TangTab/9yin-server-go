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

// ---- 目录（packId → MountType；三表合并，Java MountCatalog.load 同序） ----

type mountCatalog struct {
	byPack map[uint32]uint32 // ArtPack → MountType（仅可骑条目）
}

// loadMountCatalog 合并 itemartstatic（pack→模型，存在即可骑）+
// new_ride（pack→MountType 兜底）+ tool_item script=Mount（MountType 主源）。
// 与 Java 一致：ArtPack≤0 或无模型映射的条目不入表（不可骑）。
func loadMountCatalog(toolItemPath, itemArtStaticPath, newRidePath string) *mountCatalog {
	models := make(map[uint32]bool)
	if file, err := openResourceReader(itemArtStaticPath); err == nil {
		parseINISections(file, func(section string, key, value string) {
			if pack, ok := parseUintSection(section); ok &&
				strings.EqualFold(key, "malemodel") && value != "" {
				models[pack] = true
			}
		})
		file.Close()
	} else {
		log.Printf("mount catalog: itemartstatic unavailable: %v", err)
	}
	newRideTypes := make(map[uint32]uint32)
	if file, err := openResourceReader(newRidePath); err == nil {
		parseINISections(file, func(section string, key, value string) {
			if pack, ok := parseUintSection(section); ok && strings.EqualFold(key, "mounttype") {
				if parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 32); err == nil {
					newRideTypes[pack] = uint32(parsed)
				}
			}
		})
		file.Close()
	} else {
		log.Printf("mount catalog: new_ride unavailable: %v", err)
	}

	catalog := &mountCatalog{byPack: make(map[uint32]uint32)}
	file, err := openResourceReader(toolItemPath)
	if err != nil {
		log.Printf("mount catalog: tool_item unavailable (CurRideType degrades to 0): %v", err)
		return catalog
	}
	defer file.Close()
	toolItemMounts(file, func(configID string, mountType, artPack uint32) {
		if artPack == 0 || !models[artPack] {
			return // 无模型映射的坐骑不可骑（Java 同款过滤）
		}
		if mountType == 0 {
			mountType = newRideTypes[artPack]
		}
		catalog.byPack[artPack] = mountType
	})
	return catalog
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

// toolItemMounts 流式扫 tool_item.ini 的 script=Mount 条目（GBK 大表，
// 只取 ASCII 字段 MountType/ArtPack/Script）。configID=小节名。
func toolItemMounts(file io.Reader, visit func(configID string, mountType, artPack uint32)) {
	type mountItem struct {
		script             string
		mountType, artPack uint32
	}
	entries := make(map[string]*mountItem)
	parseINISections(file, func(section, key, value string) {
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
		case "mounttype":
			if parsed, err := strconv.ParseUint(value, 10, 32); err == nil {
				entry.mountType = uint32(parsed)
			}
		case "artpack":
			if parsed, err := strconv.ParseUint(value, 10, 32); err == nil {
				entry.artPack = uint32(parsed)
			}
		}
	})
	for configID, entry := range entries {
		if !strings.EqualFold(entry.script, "Mount") {
			continue
		}
		visit(configID, entry.mountType, entry.artPack)
	}
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

// mountRideStateFrame 骑乘态单帧 0x10（重登步行口径：Mount 恒空串）。
func mountRideStateFrame(packID, riderType uint32) ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propMount, Name: "Mount", Value: clientdata.StringValue("")},
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
	rideState, err := mountRideStateFrame(currentPack, catalog.riderTypeOf(currentPack))
	if err != nil {
		return len(mounts), fmt.Errorf("encode ride state: %w", err)
	}
	if err := link.WriteFrame(rideState); err != nil {
		return len(mounts), fmt.Errorf("write ride state: %w", err)
	}
	return len(mounts), nil
}
