package itemdata

// xml.go ini/Item 构建产物（tools/datagen item，黑金物品配置格式）的按需装载。
//
// 产物约 36MB/12.5 万行：装载期逐文件一遍扫描建「ConfigID → (文件, 元素
// 偏移, 长度)」索引后即丢弃文件字节（常驻约 10MB；整表解成 map 要数百
// MB），同 ID 首行胜出。行属性取用时按偏移回读文件解析——不长期持有文件
// 句柄，产物重生成期间不被服务端锁住——解析结果进有界缓存。静态包
// （物品行以 ArtPack/LogicPack/FuncPack… 整型引用的小表）整表入内存。
//
// 文件清单取自 path_config.xml（黑金登记表：ID/Type/Path）；静态包文件名
// 固定（黑金运行时同样按名引用，不入登记表）。表头两行 ID="-1"/"0"（列
// 序/列宽头）跳过。

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/config"
	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// 产物文件名（相对 ini/Item）。
const (
	filePathConfig = "path_config.xml"

	// 静态包：引用键 → 文件（tools/datagen/item/mapping.go packSpecs 同表）。
	packFunc       = "FuncPack"
	packLogic      = "LogicPack"
	packArt        = "ArtPack"
	packUseSkill   = "UseSkillPack"
	packLearnSkill = "LearnSkillPack"
	packTask       = "TaskPack"
	packIngredient = "IngredientPack"
	packPropModify = "PropModifyPack"
	packSeed       = "SeedPack"
)

var packFiles = map[string]string{
	packFunc:       "item_func_static.xml",
	packLogic:      "item_logic_pack.xml",
	packArt:        "ItemArtStatic.xml",
	packUseSkill:   "item_use_skill_static.xml",
	packLearnSkill: "item_learn_skill_static.xml",
	packTask:       "item_task_static.xml",
	packIngredient: "item_ingredient_static.xml",
	packPropModify: "item_prop_modify_static.xml",
	packSeed:       "item_seed_static.xml",
}

// defCacheCap 已解析模板行的缓存上限（每行约 1KB；玩家实际触达的是全集
// 的零头，封顶防长会话无界增长）。
const defCacheCap = 4096

// rowPos 一行在文件中的位置。
type rowPos struct {
	file *xmlFile
	off  int64
	n    int
}

// xmlFile 一个模板档（登记表一行）。
type xmlFile struct {
	path string
	typ  string
}

// xmlCatalog 构建产物的索引态。
type xmlCatalog struct {
	dir     string
	index   map[string]rowPos // ConfigID → 行位置（首行胜出）
	packs   map[string]packTable
	cache   defCache
	missing []string // 缺席的静态包文件（诊断面）
}

// packTable 一个静态包：行 ID → 属性。
type packTable map[string]map[string]string

// defCache 有界行缓存（环形淘汰）。
type defCache struct {
	mu    sync.Mutex
	items map[string]*Definition
	ring  []string
	next  int
}

func (c *defCache) get(id string) (*Definition, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	def, ok := c.items[id]
	return def, ok
}

func (c *defCache) put(id string, def *Definition) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.items) >= cap(c.ring) {
		delete(c.items, c.ring[c.next])
		c.ring[c.next] = id
		c.next = (c.next + 1) % cap(c.ring)
	} else {
		c.ring = append(c.ring, id)
	}
	c.items[id] = def
}

// LoadXML 装载 ini/Item 构建产物目录（不经进程级缓存）。path_config.xml
// 缺席即产物缺席（go run ./tools/datagen item 构建）；静态包缺席只记诊断。
func LoadXML(dir string) (*Catalog, error) {
	doc, err := config.LoadObjectFile(filepath.Join(dir, filePathConfig))
	if err != nil {
		return nil, fmt.Errorf("itemdata: 构建产物缺席 %s（go run ./tools/datagen item 构建）: %w", dir, err)
	}
	xc := &xmlCatalog{
		dir:   dir,
		index: map[string]rowPos{},
		packs: map[string]packTable{},
		cache: defCache{items: map[string]*Definition{}, ring: make([]string, 0, defCacheCap)},
	}
	for _, row := range doc.Rows {
		path := strings.ReplaceAll(row.Str("Path", ""), `\`, string(filepath.Separator))
		if path == "" {
			continue
		}
		file := &xmlFile{path: filepath.Join(dir, path), typ: row.Str("Type", "")}
		if err := xc.indexFile(file); err != nil {
			return nil, err
		}
	}
	if len(xc.index) == 0 {
		return nil, fmt.Errorf("itemdata: %s 登记表下无物品行", dir)
	}
	for ref, name := range packFiles {
		table, err := loadPackTable(filepath.Join(dir, name))
		if err != nil {
			xc.missing = append(xc.missing, name)
			continue
		}
		xc.packs[ref] = table
	}
	return &Catalog{xml: xc, source: "xml:" + dir}, nil
}

// indexFile 一遍扫描建索引（文件字节用完即弃）。
func (x *xmlCatalog) indexFile(file *xmlFile) error {
	raw, err := res.ReadFile(file.path)
	if err != nil {
		return fmt.Errorf("itemdata: 模板档 %s: %w", file.path, err)
	}
	for offset := 0; ; {
		at := bytes.Index(raw[offset:], []byte("<Property"))
		if at < 0 {
			return nil
		}
		start := offset + at
		end, id := scanAttributes(raw, start+len("<Property"), nil)
		if end < 0 {
			break // 畸形元素：索引止于此（产物经 itemgen -verify 回读）
		}
		if id != "" {
			if _, duplicate := x.index[id]; !duplicate {
				x.index[id] = rowPos{file: file, off: int64(start), n: end - start}
			}
		}
		offset = end
	}
	return nil
}

// loadPackTable 静态包整表入内存（跳过表头两行）。
func loadPackTable(path string) (packTable, error) {
	doc, err := config.LoadObjectFile(path)
	if err != nil {
		return nil, err
	}
	table := packTable{}
	for _, row := range doc.Rows {
		id := row["ID"]
		if id == "" || id == "-1" || id == "0" {
			continue
		}
		if _, duplicate := table[id]; duplicate {
			continue // 同 ID 首行胜出（源表重复节见 itemgen 报告）
		}
		attrs := make(map[string]string, len(row))
		for key, value := range row {
			attrs[key] = value
		}
		table[id] = attrs
	}
	return table, nil
}

// lookup 按偏移回读并解析一行（有界缓存）。
func (x *xmlCatalog) lookup(id string) (*Definition, bool) {
	if def, ok := x.cache.get(id); ok {
		return def, true
	}
	pos, ok := x.index[id]
	if !ok {
		return nil, false
	}
	// ReadFileShared 返回的可能是资源仓读缓存的内部对象：buf 只被解析、不被改写。
	raw, err := res.ReadFileShared(pos.file.path)
	if err != nil || pos.off < 0 || pos.off+int64(pos.n) > int64(len(raw)) {
		return nil, false
	}
	buf := raw[pos.off : pos.off+int64(pos.n)]
	attrs := map[string]string{}
	if end, _ := scanAttributes(buf, len("<Property"), attrs); end < 0 {
		return nil, false
	}
	def := &Definition{id: id, attrs: attrs, packs: x.packs}
	x.cache.put(id, def)
	return def, true
}

// scanAttributes 自元素名之后扫描属性直至元素结束：attrs 非 nil 时收集
// 全部属性（值已反转义，同名首见胜出），nil 时只取 ID（建索引零分配）。
// 返回元素结束偏移与 ID 属性值，end<0 为畸形。方言 name="value"，容忍
// name = "value"（黑金档写法）。
func scanAttributes(raw []byte, i int, attrs map[string]string) (end int, id string) {
	for i < len(raw) {
		for i < len(raw) && isXMLSpace(raw[i]) {
			i++
		}
		if i >= len(raw) {
			return -1, ""
		}
		switch raw[i] {
		case '>':
			return i + 1, id
		case '/':
			if i+1 < len(raw) && raw[i+1] == '>' {
				return i + 2, id
			}
			return -1, ""
		}
		nameStart := i
		for i < len(raw) && raw[i] != '=' && raw[i] != '>' && raw[i] != '/' && !isXMLSpace(raw[i]) {
			i++
		}
		if i >= len(raw) {
			return -1, ""
		}
		rawName := raw[nameStart:i]
		for i < len(raw) && isXMLSpace(raw[i]) {
			i++
		}
		if i >= len(raw) || raw[i] != '=' {
			return -1, ""
		}
		i++
		for i < len(raw) && isXMLSpace(raw[i]) {
			i++
		}
		if i >= len(raw) || (raw[i] != '"' && raw[i] != '\'') {
			return -1, ""
		}
		quote := raw[i]
		i++
		valueStart := i
		for i < len(raw) && raw[i] != quote {
			i++
		}
		if i >= len(raw) {
			return -1, ""
		}
		if attrs == nil { // 建索引：只取 ID，其余属性零分配跳过
			if id == "" && string(rawName) == "ID" {
				id = unescapeAttr(string(raw[valueStart:i]))
			}
			i++
			continue
		}
		value := unescapeAttr(string(raw[valueStart:i]))
		if id == "" && string(rawName) == "ID" {
			id = value
		}
		if _, duplicate := attrs[string(rawName)]; !duplicate {
			attrs[string(rawName)] = value
		}
		i++
	}
	return -1, ""
}

func isXMLSpace(b byte) bool { return b == ' ' || b == '\t' || b == '\r' || b == '\n' }

// unescapeAttr 属性值反转义（itemgen attrEscaper 的实体 + 十进制/十六进制
// 字符引用；无 & 时零分配直通）。
func unescapeAttr(value string) string {
	if !strings.Contains(value, "&") {
		return value
	}
	var sb strings.Builder
	sb.Grow(len(value))
	for i := 0; i < len(value); {
		if value[i] != '&' {
			j := strings.IndexByte(value[i:], '&')
			if j < 0 {
				sb.WriteString(value[i:])
				break
			}
			sb.WriteString(value[i : i+j])
			i += j
			continue
		}
		semi := strings.IndexByte(value[i:], ';')
		if semi < 0 {
			sb.WriteString(value[i:])
			break
		}
		entity := value[i+1 : i+semi]
		switch entity {
		case "amp":
			sb.WriteByte('&')
		case "lt":
			sb.WriteByte('<')
		case "gt":
			sb.WriteByte('>')
		case "quot":
			sb.WriteByte('"')
		case "apos":
			sb.WriteByte('\'')
		default:
			digits, base := "", 10
			switch {
			case strings.HasPrefix(entity, "#x"), strings.HasPrefix(entity, "#X"):
				digits, base = entity[2:], 16
			case len(entity) > 1 && entity[0] == '#':
				digits = entity[1:]
			default:
				sb.WriteString(value[i : i+semi+1]) // 未知实体原样保留
				i += semi + 1
				continue
			}
			if n, err := strconv.ParseUint(digits, base, 32); err == nil {
				sb.WriteRune(rune(n))
			}
		}
		i += semi + 1
	}
	return sb.String()
}
