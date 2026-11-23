// Package itemdata 物品目录装载：share/item/*.ini（GBK ini 方言，物品
// 定义节 [configID] 携 ItemType/MaxAmount/script/EquipType——商品发放
// 的 ViewID 分类与 ItemType 来源；用户定向 2026-11-23）。
//
// 分类规则（bag 四页：1 道具/2 装备/3 材料/4 任务）：
//   - script=Equipment 或 EquipType 非空 或 equipment.ini/innerequip → 2
//   - item_material_static.ini → 3
//   - item_task_static.ini → 4
//   - 其余 → 1
package itemdata

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// Entry 一条物品定义。
type Entry struct {
	ConfigID   string
	ItemType   int
	MaxAmount  int    // 堆叠上限（0=未知）
	ViewID     int    // 背包分类页 1..4
	EquipType  string // 装备类（武器/防具/时装…）——背包装备行必发真值
	ColorLevel int    // 品阶（0=未知，行缺省 1）
	Photo      string // 图标路径（装备行可携）
}

// Catalog 目录（configID → Entry）。
type Catalog struct {
	entries map[string]Entry
}

// Lookup 按物品 ID 查。
func (c *Catalog) Lookup(configID string) (Entry, bool) {
	if c == nil {
		return Entry{}, false
	}
	entry, ok := c.entries[configID]
	return entry, ok
}

// Count 条目数（诊断面）。
func (c *Catalog) Count() int {
	if c == nil {
		return 0
	}
	return len(c.entries)
}

// LoadDir 装载目录下全部 *.ini（单档解析失败跳过——87 档静态数据，
// 个别脏档不废整目录；同 ID 首档胜出）。
func LoadDir(dir string) (*Catalog, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.ini"))
	if err != nil {
		return nil, fmt.Errorf("itemdata: glob %s: %w", dir, err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("itemdata: 目录 %s 无 ini 档", dir)
	}
	catalog := &Catalog{entries: map[string]Entry{}}
	for _, path := range files {
		entries, perr := LoadFile(path)
		if perr != nil {
			continue
		}
		for id, entry := range entries {
			if _, dup := catalog.entries[id]; dup {
				continue
			}
			catalog.entries[id] = entry
		}
	}
	if len(catalog.entries) == 0 {
		return nil, fmt.Errorf("itemdata: %s 无物品节", dir)
	}
	return catalog, nil
}

// LoadFile 单档装载（合法 UTF-8 直通，否则 GBK 转码；[//注释节] 跳过）。
func LoadFile(path string) (map[string]Entry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(raw) {
		if decoded, derr := simplifiedchinese.GB18030.NewDecoder().Bytes(raw); derr == nil {
			raw = decoded
		}
	}
	viewID := viewIDOfFile(filepath.Base(path))
	entries := map[string]Entry{}
	var current string
	var entry Entry
	flush := func() {
		if current != "" {
			entries[current] = entry
		}
		current, entry = "", Entry{}
	}
	for line := range splitLines(string(raw)) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()
			id := strings.TrimSpace(line[1 : len(line)-1])
			if id == "" || strings.HasPrefix(id, "//") {
				continue // 注释节（[//拜师入门专用] 等）
			}
			current = id
			entry = Entry{ConfigID: id, ViewID: viewID}
			continue
		}
		if current == "" {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		value = strings.TrimSpace(value)
		switch strings.TrimSpace(key) {
		case "ItemType":
			if v, perr := strconv.Atoi(value); perr == nil {
				entry.ItemType = v
			}
		case "MaxAmount":
			if v, perr := strconv.Atoi(value); perr == nil {
				entry.MaxAmount = v
			}
		case "script":
			if strings.EqualFold(value, "Equipment") {
				entry.ViewID = 2
			}
		case "EquipType":
			if value != "" {
				entry.ViewID = 2
				entry.EquipType = value
			}
		case "ColorLevel":
			if v, perr := strconv.Atoi(value); perr == nil {
				entry.ColorLevel = v
			}
		case "Photo":
			entry.Photo = value
		}
	}
	flush()
	if len(entries) == 0 {
		return nil, fmt.Errorf("itemdata: %s 无物品节", path)
	}
	return entries, nil
}

// viewIDOfFile 文件级分类（节内 script/EquipType 可覆盖为装备页）。
func viewIDOfFile(base string) int {
	switch strings.ToLower(base) {
	case "item_material_static.ini":
		return 3
	case "item_task_static.ini":
		return 4
	case "equipment.ini":
		return 2
	default:
		return 1
	}
}

// splitLines 按行迭代（\r\n/\n）。
func splitLines(s string) func(func(string) bool) {
	return func(yield func(string) bool) {
		for len(s) > 0 {
			i := strings.IndexByte(s, '\n')
			if i < 0 {
				if !yield(s) {
					return
				}
				return
			}
			if !yield(s[:i]) {
				return
			}
			s = s[i+1:]
		}
	}
}
