package taskdata

// prize.go 任务奖励的取数面：完成奖励取 Prize 域同号行的 23 字段奖励串
// （Task.Prize，排布见 steps.go prizeBase），步骤奖励取 InteractiveList 的
// GetItems/LostItems。这里只做解析，发放与帧序在 fssoloworld。
//
// 官服引导抓包的发放帧序（完成簇开头、完成位图之前）：
//
//	物品：入包帧 → m1 [1,"9645","<物品>",<数量>]（逐件）→
//	      m1 [1,"8","10012,<物品>,<数量>|…"]（整单一张）
//	碎银：0x10 CapitalType1=新余额 → m1 [1,"1303",<数量>]
//	声望：Repute_Record 行（新声望加行，已有则逐格写数值）→
//	      m1 [1,"9693","<声望名>",<数量>]
//	历练：0x10 LiveGroove_<下标>=新余额 → m1 [1,"8055",<累计量/1000>]
//
// 步骤奖励（1347 的交互步 558→武学书）在目标行 curnum 写入之前发放。

import (
	"strconv"
	"strings"
)

// PrizeItem 一件奖励物品。
type PrizeItem struct {
	ID  string
	Num int
}

// PrizeMoney 完成奖励的碎银（prize_base[0] CapitalType1）。
func (t Task) PrizeMoney() int64 {
	fields := strings.Split(t.Prize, ",")
	if len(fields) == 0 {
		return 0
	}
	n, err := strconv.ParseInt(strings.TrimSpace(fields[0]), 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// PrizeItems 完成奖励的物品（prize_base[10..17] Item1,Num1 … Item4,Num4）。
func (t Task) PrizeItems() []PrizeItem {
	fields := strings.Split(t.Prize, ",")
	var items []PrizeItem
	for k := 0; k < 4; k++ {
		id, num := 10+2*k, 11+2*k
		if len(fields) <= num {
			break
		}
		name := strings.TrimSpace(fields[id])
		count, err := strconv.Atoi(strings.TrimSpace(fields[num]))
		if name == "" || err != nil || count <= 0 {
			continue
		}
		items = append(items, PrizeItem{ID: name, Num: count})
	}
	return items
}

// PrizeRepute 完成奖励的声望（prize_base[2]/[3] ReputeName1/ReputeCount1）。
func (t Task) PrizeRepute() (name string, count int32) {
	fields := strings.Split(t.Prize, ",")
	if len(fields) < 4 {
		return "", 0
	}
	n, err := strconv.ParseInt(strings.TrimSpace(fields[3]), 10, 32)
	name = strings.TrimSpace(fields[2])
	if name == "" || err != nil || n <= 0 {
		return "", 0
	}
	return name, int32(n)
}

// PrizeLiveGroove 完成奖励的历练（prize_base[19]/[20]：LiveGroove 下标 1..5
// 与累计量）。下标越界或数量非正视为没有。
func (t Task) PrizeLiveGroove() (index int, count int32) {
	fields := strings.Split(t.Prize, ",")
	if len(fields) < 21 {
		return 0, 0
	}
	index, err := strconv.Atoi(strings.TrimSpace(fields[19]))
	n, err2 := strconv.ParseInt(strings.TrimSpace(fields[20]), 10, 32)
	if err != nil || err2 != nil || index < 1 || index > 5 || n <= 0 {
		return 0, 0
	}
	return index, int32(n)
}

// ParseItemSpec 解析物品清单 "物品-数量|物品-数量" 或 "物品;物品"（无数量
// 按 1）；两种分隔可混用。数量非正的条目丢弃。
func ParseItemSpec(spec string) []PrizeItem {
	var items []PrizeItem
	for _, field := range strings.FieldsFunc(spec, func(r rune) bool { return r == '|' || r == ';' }) {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		item := PrizeItem{ID: field, Num: 1}
		if i := strings.LastIndexByte(field, '-'); i > 0 {
			if n, err := strconv.Atoi(field[i+1:]); err == nil {
				item.ID, item.Num = field[:i], n
			}
		}
		if item.Num > 0 {
			items = append(items, item)
		}
	}
	return items
}
