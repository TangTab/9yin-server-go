package main

// 宝物洗炼计费（用户拍板按表口径）：hhttreasuretaolu.ini 的
//   [taolu]        套路/技能 id → 套路号
//   [special_cost] 套路号 → 火浣台个数（古谱/特殊武学，单位：个火浣台）
//   [normal_cost]  档 → 文（普通武学；1..7 = 10,10,10,10,10,100,100）
// 洗时扣费、存不扣；余额不足拒绝（仅日志，客户端无错误回包方言）。
// 属性洗无表可依，按 [normal_cost] 首档 10 文常数收（口径待官服实证校正）。

import (
	"log"
	"strconv"
	"strings"
	"sync"
)

// tersureHuantaiItem 火浣台（洗炼/副套路系统的消耗道具；idres
// gn_Procon_001=火浣台）。
const tersureHuantaiItem = "gn_Procon_001"

// tersurePropWashCostSilver 属性洗单价（文；[normal_cost] 首档口径）。
const tersurePropWashCostSilver = 10

type tersureCostTables struct {
	taoluNum    map[string]int // [taolu] skill → 套路号
	specialCost map[int]int32  // [special_cost] 套路号 → 火浣台数
	normalCost  []int32        // [normal_cost] 按档序（1 基）
}

var (
	tersureCostOnce   sync.Once
	tersureCostLoaded *tersureCostTables
)

func tersureCostTablesOf() *tersureCostTables {
	tersureCostOnce.Do(func() {
		tables := &tersureCostTables{
			taoluNum:    map[string]int{},
			specialCost: map[int]int32{},
		}
		_ = parseModernINI(modernSharePath("rule", "randomequiprule", "hhttreasuretaolu.ini"),
			func(section, key, value string) error {
				num, err := strconv.Atoi(strings.TrimSpace(value))
				if err != nil {
					return nil
				}
				switch section {
				case "taolu":
					tables.taoluNum[key] = num
				case "special_cost":
					tables.specialCost[keyNum(key)] = int32(num)
				case "normal_cost":
					tables.normalCost = append(tables.normalCost, int32(num))
				}
				return nil
			})
		tersureCostLoaded = tables
	})
	return tersureCostLoaded
}

func keyNum(key string) int {
	num, _ := strconv.Atoi(strings.TrimSpace(key))
	return num
}

// tersureSkillWashCost 武学洗费用：新技能的 skillstore TaoLu/SubSkill 在
// [taolu] 表内且 [special_cost] 有价 ⇒ 火浣台；否则按 [normal_cost] 首档文。
// 返回 (火浣台数, 文, 计费键)。
func tersureSkillWashCost(taoluID string) (huantai, wen int32, billed string) {
	tables := tersureCostTablesOf()
	if num, ok := tables.taoluNum[taoluID]; ok {
		if cost, ok := tables.specialCost[num]; ok && cost > 0 {
			return cost, 0, taoluID
		}
	}
	return 0, tersureNormalTierSilver(1), taoluID
}

// tersureNormalTierSilver 普通档文价（档 1..7；越界取首末档）。
func tersureNormalTierSilver(tier int) int32 {
	tables := tersureCostTablesOf()
	if len(tables.normalCost) == 0 {
		return tersurePropWashCostSilver
	}
	if tier < 1 {
		tier = 1
	}
	if tier > len(tables.normalCost) {
		tier = len(tables.normalCost)
	}
	return tables.normalCost[tier-1]
}

// billTersureWash 洗炼扣费：火浣台走背包跨行扣减（consumeBagConfig，
// 客户端行同步内建），文走碎银（trySpendSilver + 落库）。不足返回 false。
func (h *itemSession) billTersureWash(huantai, wen int32, what, configID string) bool {
	if huantai > 0 {
		if h.player.countBagConfig(tersureHuantaiItem) < huantai {
			log.Printf("%s: tersure %s %s needs huantai %d, has %d", h.remote, what, configID,
				huantai, h.player.countBagConfig(tersureHuantaiItem))
			return false
		}
		if !h.consumeBagConfig(tersureHuantaiItem, huantai) {
			return false
		}
		log.Printf("%s: tersure %s %s billed %d huantai", h.remote, what, configID, huantai)
		return true
	}
	if wen > 0 {
		if !h.player.trySpendSilver(wen) {
			log.Printf("%s: tersure %s %s needs silver %d, has %d", h.remote, what, configID,
				wen, h.player.silverBalance())
			return false
		}
		h.persistCurrency("tersure wash")
		log.Printf("%s: tersure %s %s billed %d silver", h.remote, what, configID, wen)
		return true
	}
	return true
}
