package main

// 掉落奖励包（share/drop/drops_mvp.json，MVP 自制权重包）与箱子内容物
// 对照表（share/drop/box_contents.json 固定全发）——移植 Java DropPacks/
// BoxContents。结构：packages.{DropId}=[{item,min,max,weight}]；
// boxes.{configId}=[{item,min,max}]。每次 roll 按权重抽 1 条，数量在
// min..max 均匀取；缺包走 FALLBACK_LADDER 兜底（不再静默空返，2026-11-04
// 真机「使用无效果」根因修复同款）。

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"path/filepath"
	"sort"
)

// dropPackEntry 单条掉落（weight ≤0 视为无效跳过）。
type dropPackEntry struct {
	item   string
	min    int32
	max    int32
	weight int32
}

type dropPackEntryJSON struct {
	Item   string `json:"item"`
	Min    int32  `json:"min"`
	Max    int32  `json:"max"`
	Weight int32  `json:"weight"`
}

// grantItem 单条确定性给物品（roll 合并产物同形）。
type grantItem struct {
	configID string
	amount   int32
}

// dropPacks 缺包兜底阶梯（yaopin 阶梯，箱包偏高档；形制对齐 qy_chengdu 系）。
var dropFallbackLadder = []dropPackEntry{
	{"yaopin_00100", 1, 2, 50},
	{"yaopin_00300", 1, 2, 30},
	{"yaopin_00500", 1, 1, 15},
	{"yaopin_00700", 1, 1, 5},
}

type dropPacks struct {
	packages map[string][]dropPackEntry
}

// loadDropPacks 解析 drops_mvp.json；文件缺失保持空目录（开包按缺包兜底走）。
func loadDropPacks(data []byte) (*dropPacks, error) {
	packs := &dropPacks{packages: make(map[string][]dropPackEntry)}
	if len(data) == 0 {
		return packs, nil
	}
	var root struct {
		Packages map[string][]dropPackEntryJSON `json:"packages"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return packs, fmt.Errorf("parse drops_mvp.json: %w", err)
	}
	for id, entries := range root.Packages {
		list := make([]dropPackEntry, 0, len(entries))
		for _, entry := range entries {
			if entry.Item == "" || entry.Weight <= 0 || entry.Min < 1 || entry.Max < entry.Min || entry.Max > 999 {
				continue
			}
			list = append(list, dropPackEntry{item: entry.Item, min: entry.Min, max: entry.Max, weight: entry.Weight})
		}
		if len(list) > 0 {
			packs.packages[id] = list
		}
	}
	return packs, nil
}

func (p *dropPacks) size() int { return len(p.packages) }

func (p *dropPacks) has(dropID string) bool {
	_, exists := p.packages[dropID]
	return exists
}

// roll 权重抽取：rolls 次独立抽取，同物品数量合并（rolls 钳 1..10）。
// 缺包走 FALLBACK_LADDER（日志可追）；空 dropId 返回空。
func (p *dropPacks) roll(dropID string, rolls int32, random *rand.Rand) []grantItem {
	out := make([]grantItem, 0, 4)
	if dropID == "" {
		return out
	}
	entries, exists := p.packages[dropID]
	if !exists {
		log.Printf("drop pack %s missing, rolling fallback ladder", dropID)
		entries = dropFallbackLadder
	}
	clampedRolls := rolls
	if clampedRolls < 1 {
		clampedRolls = 1
	}
	if clampedRolls > 10 {
		clampedRolls = 10
	}
	merged := make(map[string]int32)
	order := make([]string, 0, 4)
	for i := int32(0); i < clampedRolls; i++ {
		picked := pickDropEntry(entries, random.Float64())
		if picked == nil {
			continue
		}
		spread := picked.max - picked.min + 1
		amount := picked.min + int32(random.Float64()*float64(spread))
		if amount < picked.min {
			amount = picked.min
		}
		if amount > picked.max {
			amount = picked.max
		}
		if _, seen := merged[picked.item]; !seen {
			order = append(order, picked.item)
		}
		merged[picked.item] += amount
	}
	for _, item := range order {
		out = append(out, grantItem{configID: item, amount: merged[item]})
	}
	return out
}

func pickDropEntry(entries []dropPackEntry, r float64) *dropPackEntry {
	total := int32(0)
	for index := range entries {
		total += entries[index].weight
	}
	if total <= 0 {
		return nil
	}
	cursor := int32(r * float64(total))
	for index := range entries {
		cursor -= entries[index].weight
		if cursor < 0 {
			return &entries[index]
		}
	}
	return &entries[len(entries)-1]
}

// ---- box_contents.json（描述重建对照，固定全发） ----

type boxContents struct {
	boxes map[string][]grantItem
}

func loadBoxContents(data []byte) (*boxContents, error) {
	contents := &boxContents{boxes: make(map[string][]grantItem)}
	if len(data) == 0 {
		return contents, nil
	}
	var root struct {
		Boxes map[string][]struct {
			Item string `json:"item"`
			Min  int32  `json:"min"`
			Max  int32  `json:"max"`
		} `json:"boxes"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return contents, fmt.Errorf("parse box_contents.json: %w", err)
	}
	for box, entries := range root.Boxes {
		list := make([]grantItem, 0, len(entries))
		for _, entry := range entries {
			if entry.Item == "" {
				continue
			}
			amount := entry.Max
			if amount < 1 {
				amount = 1
			}
			if amount > 999 {
				continue
			}
			list = append(list, grantItem{configID: entry.Item, amount: amount})
		}
		if len(list) > 0 {
			contents.boxes[lowerASCII(box)] = list
		}
	}
	return contents, nil
}

// byBox 按箱子 configId 查描述内容物；命中返回全发清单（不可变拷贝）。
func (c *boxContents) byBox(configID string) []grantItem {
	if configID == "" {
		return nil
	}
	entries, exists := c.boxes[lowerASCII(configID)]
	if !exists {
		return nil
	}
	return append([]grantItem(nil), entries...)
}

func lowerASCII(value string) string {
	equal := true
	for index := 0; index < len(value); index++ {
		if value[index] >= 'A' && value[index] <= 'Z' {
			equal = false
			break
		}
	}
	if equal {
		return value
	}
	out := []byte(value)
	for index := range out {
		if out[index] >= 'A' && out[index] <= 'Z' {
			out[index] += 'a' - 'A'
		}
	}
	return string(out)
}

// ---- 装载（资源仓 → 可选加载；测试经 load*FromBytes 注入） ----

var (
	itemDropPacksOnce    func() *dropPacks
	itemBoxContentsOnce  func() *boxContents
	itemCatalogOnce      func() *itemUseCatalog
	itemSkillBooksOnce   func() *skillBookCatalog
	itemSkillGatesOnce   func() *skillGateTable
	itemOverridesApplied bool
)

// applyItemTestOverrides 单测注入合成目录/包表（生产代码不得调用）。
func applyItemTestOverrides(packs *dropPacks, contents *boxContents, use *itemUseCatalog, books *skillBookCatalog, gates *skillGateTable) {
	itemDropPacksOnce = func() *dropPacks { return packs }
	itemBoxContentsOnce = func() *boxContents { return contents }
	itemCatalogOnce = func() *itemUseCatalog { return use }
	itemSkillBooksOnce = func() *skillBookCatalog { return books }
	itemSkillGatesOnce = func() *skillGateTable { return gates }
	itemOverridesApplied = true
}

func defaultDropPacks() *dropPacks {
	data, err := readResourceBytes(filepath.Join(defaultModernShareRoot, "drop", "drops_mvp.json"))
	if err != nil && !errorsIsNotExist(err) {
		log.Printf("drop packs unavailable (%v); fallback ladder covers opens", err)
	}
	packs, parseErr := loadDropPacks(data)
	if parseErr != nil {
		log.Printf("%v", parseErr)
	}
	log.Printf("drop packs ready: %d packages", packs.size())
	return packs
}

func defaultBoxContents() *boxContents {
	data, err := readResourceBytes(filepath.Join(defaultModernShareRoot, "drop", "box_contents.json"))
	if err != nil && !errorsIsNotExist(err) {
		log.Printf("box contents unavailable (%v); described boxes degrade to drop packs", err)
	}
	contents, parseErr := loadBoxContents(data)
	if parseErr != nil {
		log.Printf("%v", parseErr)
	}
	log.Printf("box contents ready: %d boxes", len(contents.boxes))
	return contents
}

func errorsIsNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

var (
	dropPacksLoaded   *dropPacks
	boxContentsLoaded *boxContents
)

func currentDropPacks() *dropPacks {
	if itemDropPacksOnce != nil {
		return itemDropPacksOnce()
	}
	if dropPacksLoaded == nil {
		dropPacksLoaded = defaultDropPacks()
	}
	return dropPacksLoaded
}

func currentBoxContents() *boxContents {
	if itemBoxContentsOnce != nil {
		return itemBoxContentsOnce()
	}
	if boxContentsLoaded == nil {
		boxContentsLoaded = defaultBoxContents()
	}
	return boxContentsLoaded
}

// sortedGrantOrder 输出确定性日志顺序（测试断言辅助）。
func sortedGrantOrder(items []grantItem) []string {
	order := make([]string, 0, len(items))
	for _, item := range items {
		order = append(order, fmt.Sprintf("%sx%d", item.configID, item.amount))
	}
	sort.Strings(order)
	return order
}
