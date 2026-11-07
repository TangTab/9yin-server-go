package main

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// 兑换目录（msg70 capitalType=3 分支；docs/fork-p3-exchange-school.md）。
//
// 数据源 `share/item/exchangeitem.ini`（约 4.5 万行，GBK）：节名 = 十进制
// exchangeID（shop.ini 商品行第 8 列引用），键：
//
//	Item=pingzheng001,1;pingzheng002,1   → 按物扣费（分号列表，configID,数量）
//	Prop=CapitalType2,100000             → 按币/属性扣费（CapitalTypeN,数量）
//	Condition=120958 / Condition2=109035 → 任务条件门（本轮不评估，仅记录）
//
// fork（9/14 二进制）语义：Prop 名必须是服务端跟踪的货币，否则
// "uses untracked currency (refused)"；条件未评估仅记日志不拦截。
// 消费的键值均为 ASCII，GBK 高位字节只出现在被忽略的注释/其他键里。
var defaultExchangeINIPath = filepath.Join(defaultModernShareRoot, "item", "exchangeitem.ini")

type exchangeItemCost struct {
	configID string
	count    int32
}

type exchangePropCost struct {
	name string // 如 "CapitalType2"
	cost int32
}

type exchangeCatalogEntry struct {
	items     []exchangeItemCost
	props     []exchangePropCost
	condition string
}

func (entry exchangeCatalogEntry) empty() bool {
	return len(entry.items) == 0 && len(entry.props) == 0
}

type exchangeCatalog struct {
	byID map[int32]exchangeCatalogEntry
}

func (catalog *exchangeCatalog) entry(id int32) (exchangeCatalogEntry, bool) {
	if catalog == nil || id == 0 {
		return exchangeCatalogEntry{}, false
	}
	entry, ok := catalog.byID[id]
	return entry, ok
}

var (
	exchangeCatalogMu    sync.Mutex
	exchangeCatalogCache = map[string]*exchangeCatalog{}
)

// exchangeCatalogFor 懒加载并按路径缓存目录（测试可换 defaultExchangeINIPath
// 指向 stub）。
func exchangeCatalogFor(path string) (*exchangeCatalog, error) {
	exchangeCatalogMu.Lock()
	defer exchangeCatalogMu.Unlock()
	if cached, ok := exchangeCatalogCache[path]; ok {
		return cached, nil
	}
	file, err := openResourceReader(path)
	if err != nil {
		return nil, fmt.Errorf("open exchange catalog %s: %w", path, err)
	}
	defer file.Close()
	catalog, err := parseExchangeCatalog(file, path)
	if err != nil {
		return nil, err
	}
	if len(catalog.byID) == 0 {
		return nil, fmt.Errorf("exchange catalog %s is empty", path)
	}
	exchangeCatalogCache[path] = catalog
	return catalog, nil
}

// resetExchangeCatalogCache 测试辅助：换 stub 路径后清缓存。
func resetExchangeCatalogCache() {
	exchangeCatalogMu.Lock()
	defer exchangeCatalogMu.Unlock()
	exchangeCatalogCache = map[string]*exchangeCatalog{}
}

func parseExchangeCatalog(reader io.Reader, path string) (*exchangeCatalog, error) {
	catalog := &exchangeCatalog{byID: make(map[int32]exchangeCatalogEntry)}
	scanner := bufio.NewScanner(reader)
	lineNumber := 0
	section := int32(0)
	inSection := false
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			parsed, parseErr := strconv.ParseInt(strings.Trim(line, "[]"), 10, 32)
			if parseErr != nil {
				return nil, fmt.Errorf("exchange catalog %s line %d: invalid section %q: %w", path, lineNumber, line, parseErr)
			}
			section = int32(parsed)
			inSection = true
			continue
		}
		if !inSection {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		entry := catalog.byID[section]
		switch strings.TrimSpace(key) {
		case "Item":
			items, err := parseExchangeCostList(value, "Item", path, lineNumber)
			if err != nil {
				return nil, err
			}
			entry.items = append(entry.items, items...)
		case "Prop":
			props, err := parseExchangePropList(value, path, lineNumber)
			if err != nil {
				return nil, err
			}
			entry.props = append(entry.props, props...)
		case "Condition", "Condition2":
			entry.condition = strings.TrimSpace(value)
		}
		catalog.byID[section] = entry
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read exchange catalog %s: %w", path, err)
	}
	return catalog, nil
}

// parseExchangeCostList 解析 `configID,count;configID,count`（fork
// "expected name,amount" 同语义的坏行拒绝）。
func parseExchangeCostList(value, field, path string, lineNumber int) ([]exchangeItemCost, error) {
	var costs []exchangeItemCost
	for _, chunk := range strings.Split(strings.TrimSpace(value), ";") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		name, amount, found := strings.Cut(chunk, ",")
		if !found {
			return nil, fmt.Errorf("exchange catalog %s line %d: invalid %s %q: expected name,amount", path, lineNumber, field, chunk)
		}
		parsed, parseErr := strconv.ParseInt(strings.TrimSpace(amount), 10, 32)
		if parseErr != nil {
			return nil, fmt.Errorf("exchange catalog %s line %d: invalid %s %q: %w", path, lineNumber, field, chunk, parseErr)
		}
		costs = append(costs, exchangeItemCost{configID: strings.TrimSpace(name), count: int32(parsed)})
	}
	return costs, nil
}

func parseExchangePropList(value, path string, lineNumber int) ([]exchangePropCost, error) {
	var costs []exchangePropCost
	for _, chunk := range strings.Split(strings.TrimSpace(value), ";") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		name, amount, found := strings.Cut(chunk, ",")
		if !found {
			return nil, fmt.Errorf("exchange catalog %s line %d: invalid Prop %q: expected name,amount", path, lineNumber, chunk)
		}
		parsed, parseErr := strconv.ParseInt(strings.TrimSpace(amount), 10, 32)
		if parseErr != nil {
			return nil, fmt.Errorf("exchange catalog %s line %d: invalid Prop %q: %w", path, lineNumber, chunk, parseErr)
		}
		costs = append(costs, exchangePropCost{name: strings.TrimSpace(name), cost: int32(parsed)})
	}
	return costs, nil
}
