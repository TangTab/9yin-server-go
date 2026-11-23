// Package npcshop 商店目录装载：share/trade/shop.ini（GBK ini 方言，
// 2433 loadShopCatalogSection 同源数据——9yin-server-go2 Res 定版）。
//
// 方言要点：节头 [Shop_*]；NameID/PageInfo=Page1,Page2/Type 三元数据键；
// 物品行用刻意重复的数字键（通常 0=），CSV 列序（≥7 列）：
//
//	0=configID 1=amount 2=capitalType 3=price 4=(未用) 5=page 6=position
//	7=exchangeID(可选) 8=conditionID(可选)
//
// capitalType 0=金币 1=银币 2=卡 3=兑换（三价全 0，成本走 exchangeitem）。
// 兑换节偶有 page=0 行（经典档方言），装载时归一为第 1 页。
package npcshop

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// Item 一条出售条目。
type Item struct {
	ConfigID    string
	Amount      int32
	CapitalType int32
	Price       int32
	Page        int32
	Position    int32
	ExchangeID  int32
	ConditionID int32
}

// Section 一个商店节。
type Section struct {
	ID     string
	NameID string
	Type   int32
	Pages  int32 // PageInfo 页数与物品行最大页的较大者
	Items  []Item
}

// Catalog 全档目录。
type Catalog struct {
	sections map[string]*Section
}

// Section 按 ShopID 取节。
func (c *Catalog) Section(id string) (*Section, bool) {
	if c == nil {
		return nil, false
	}
	sec, ok := c.sections[id]
	return sec, ok
}

// Count 节数（诊断面）。
func (c *Catalog) Count() int {
	if c == nil {
		return 0
	}
	return len(c.sections)
}

// Load 读 shop.ini（合法 UTF-8 直通，否则 GBK 转码——与 npcgen toUTF8
// 同款自适应）。
func Load(path string) (*Catalog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("npcshop: read %s: %w", path, err)
	}
	if !utf8.Valid(raw) {
		if decoded, derr := simplifiedchinese.GB18030.NewDecoder().Bytes(raw); derr == nil {
			raw = decoded
		}
	}
	return Parse(raw)
}

// Parse 解析已解码字节。
func Parse(raw []byte) (*Catalog, error) {
	catalog := &Catalog{sections: map[string]*Section{}}
	var current *Section
	for lineNo, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			id := strings.TrimSpace(line[1 : len(line)-1])
			if id == "" {
				current = nil
				continue
			}
			current = &Section{ID: id, Pages: 1}
			catalog.sections[id] = current
			continue
		}
		if current == nil {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "NameID":
			current.NameID = value
			continue
		case "PageInfo":
			if n := int32(strings.Count(value, ",") + 1); n > current.Pages {
				current.Pages = n
			}
			continue
		case "Type":
			if v, perr := strconv.ParseInt(value, 10, 32); perr == nil {
				current.Type = int32(v)
			}
			continue
		}
		// 物品行：数字键（重复的 0= 等）。
		if _, perr := strconv.ParseInt(key, 10, 32); perr != nil {
			continue
		}
		item, ierr := parseItem(current.ID, lineNo+1, value)
		if ierr != nil {
			return nil, ierr
		}
		current.Items = append(current.Items, item)
		if item.Page > current.Pages {
			current.Pages = item.Page
		}
	}
	if len(catalog.sections) == 0 {
		return nil, fmt.Errorf("npcshop: 无商店节")
	}
	return catalog, nil
}

// parseItem 物品行 CSV（列序见包注释；脏行整档报错——商店数据静态，
// 早失败优于静默缺货）。
func parseItem(sectionID string, lineNo int, value string) (Item, error) {
	parts := strings.Split(value, ",")
	if len(parts) < 7 {
		return Item{}, fmt.Errorf("npcshop: 节 %s 行 %d: 物品列 %d < 7", sectionID, lineNo, len(parts))
	}
	col := func(i int, name string) (int32, error) {
		v, err := strconv.ParseInt(strings.TrimSpace(parts[i]), 10, 32)
		if err != nil {
			return 0, fmt.Errorf("npcshop: 节 %s 行 %d: %s 列 %q: %w", sectionID, lineNo, name, parts[i], err)
		}
		return int32(v), nil
	}
	amount, err := col(1, "amount")
	if err != nil {
		return Item{}, err
	}
	capitalType, err := col(2, "capitalType")
	if err != nil {
		return Item{}, err
	}
	price, err := col(3, "price")
	if err != nil {
		return Item{}, err
	}
	page, err := col(5, "page")
	if err != nil {
		return Item{}, err
	}
	position, err := col(6, "position")
	if err != nil {
		return Item{}, err
	}
	if page < 1 {
		// 经典档兑换节存在 page=0 行（Shop_menpai_xtc_01 等 Type=1
		// 节）——归一为第 1 页，objectIndex 公式保持良构。
		page = 1
	}
	if position < 0 {
		return Item{}, fmt.Errorf("npcshop: 节 %s 行 %d: position=%d 越界", sectionID, lineNo, position)
	}
	item := Item{
		ConfigID:    strings.TrimSpace(parts[0]),
		Amount:      amount,
		CapitalType: capitalType,
		Price:       price,
		Page:        page,
		Position:    position,
	}
	opt := func(i int) int32 {
		if i >= len(parts) {
			return 0
		}
		v, err := strconv.ParseInt(strings.TrimSpace(parts[i]), 10, 32)
		if err != nil {
			return 0
		}
		return int32(v)
	}
	item.ExchangeID = opt(7)
	item.ConditionID = opt(8)
	return item, nil
}
