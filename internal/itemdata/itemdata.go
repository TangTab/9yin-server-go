// Package itemdata 物品目录装载（服务端只读构建产物）。
//
// 主来源是 ini/Item 构建产物（tools/datagen item：九阴设计态 share/item 表 →
// 黑金物品配置格式的服务端 XML，见 docs/official/物品服务端XML构建链与
// msg31使用链.md）——模板行按 path_config.xml 登记分档，行携逻辑类
// script/ItemType/ViewID 与静态包引用（FuncPack/LogicPack/ArtPack/
// UseSkillPack/LearnSkillPack…），xml.go 建偏移索引按需回读，definition.go
// 给取值面与静态包联查。
//
// 产物缺席时回退设计态 share/item/*.ini（GBK ini 方言，物品定义节
// [configID]）——只有 Entry 目录面（发放行的 ItemType/分类页/渲染属性），
// 无模板行。
//
// 分类页（bag 四页：1 道具/2 装备/3 材料/4 任务）：行内 ViewID 为准；
// 缺省时 Equipment 逻辑类或 EquipType 非空归装备页，其余道具页。
package itemdata

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// Entry 一条物品定义（目录面：发放行的 ItemType/分类页/渲染属性来源）。
type Entry struct {
	ConfigID   string
	ItemType   int
	MaxAmount  int    // 堆叠上限（0=未知）
	ViewID     int    // 背包分类页 1..4
	EquipType  string // 装备类（武器/防具/时装…）——背包装备行必发真值
	ColorLevel int    // 品阶（0=未知，行缺省 1）
	Photo      string // 图标路径（装备行可携）
	Name       string // 显示名
	Script     string // 逻辑类（script；使用链分派键）
}

// Catalog 目录（configID → Entry）；构建产物与设计态 ini 两种来源之一。
type Catalog struct {
	entries map[string]Entry // 设计态 ini 来源
	xml     *xmlCatalog      // 构建产物来源（nil = ini 来源）
	source  string
}

// Lookup 按物品 ID 查。
func (c *Catalog) Lookup(configID string) (Entry, bool) {
	if c == nil {
		return Entry{}, false
	}
	if c.xml != nil {
		def, ok := c.xml.lookup(configID)
		if !ok {
			return Entry{}, false
		}
		return def.entry(), true
	}
	entry, ok := c.entries[configID]
	return entry, ok
}

// Definition 模板行（仅构建产物来源；ini 来源恒 false）。
func (c *Catalog) Definition(configID string) (*Definition, bool) {
	if c == nil || c.xml == nil {
		return nil, false
	}
	return c.xml.lookup(configID)
}

// Count 条目数（诊断面）。
func (c *Catalog) Count() int {
	if c == nil {
		return 0
	}
	if c.xml != nil {
		return len(c.xml.index)
	}
	return len(c.entries)
}

// FromXML 是否构建产物来源。
func (c *Catalog) FromXML() bool { return c != nil && c.xml != nil }

// Source 来源描述（"xml:<目录>" / "ini:<目录>"）。
func (c *Catalog) Source() string {
	if c == nil {
		return ""
	}
	return c.source
}

// MissingPacks 缺席的静态包文件（诊断面；ini 来源为空）。
func (c *Catalog) MissingPacks() []string {
	if c == nil || c.xml == nil {
		return nil
	}
	return c.xml.missing
}

// LoadDir 装载目录下全部 *.ini（设计态回退；单档解析失败跳过——87 档
// 静态数据，个别脏档不废整目录；同 ID 首档胜出）。
func LoadDir(dir string) (*Catalog, error) {
	files, err := res.Glob(filepath.Join(dir, "*.ini"))
	if err != nil {
		return nil, fmt.Errorf("itemdata: glob %s: %w", dir, err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("itemdata: 目录 %s 无 ini 档", dir)
	}
	catalog := &Catalog{entries: map[string]Entry{}, source: "ini:" + dir}
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
	raw, err := res.ReadFile(path)
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
	rowView := 0 // 行内 ViewID（1..4 才认；0=未标）
	flush := func() {
		if current != "" {
			if rowView != 0 {
				entry.ViewID = rowView // 行内页标优先于文件级/逻辑类推断
			}
			entries[current] = entry
		}
		current, entry, rowView = "", Entry{}, 0
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
		case "Name":
			entry.Name = value
		case "ViewID":
			if v, perr := strconv.Atoi(value); perr == nil && v >= 1 && v <= 4 {
				rowView = v
			}
		case "script":
			entry.Script = value
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

// viewIDOfFile 文件级分类（节内 ViewID/script/EquipType 可覆盖）。
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

func atoi(value string) (int, error) { return strconv.Atoi(strings.TrimSpace(value)) }

func atoiSafe(value string) int {
	n, err := atoi(value)
	if err != nil {
		return 0
	}
	return n
}
