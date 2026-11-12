package main

// 武器强化/淬炼（C2S 861 BEGIN_QIANGHUA；form_enhance_reduce.lua +
// custom_sender.lua:1505-1511 客户端方言，menu2 weapon_quench.go 语义
// 对齐——连招小游戏（QuenchChef/Taolu/Moves）一期不做，按单发受理）：
//
//	C2S 861 (view, index, jobid:string, itemreduce:string)
//	  view=1 穿戴视图（index=身体槽，须武器 22）/ view=121 背包装备页
//	  （index=槽位）。jobid=生活职业（equip_prop_rand JobName）；
//	  itemreduce=玩家选的降耗融合石 config（空=表默认 Item）。
//
// 规则表 share/rule/equip_prop_rand.ini：[id] Cost=碎银 / Item=融合石,
// 数量,? / JobName / JobLevel / PropPack=词条包池。节→武器映射在官方
// 是物品实例 PropGetPack 属性（当前物品行无此属性——一期按 JobName
// 匹配 + 融合石在包校验近似，真机校准点）。
//
// 消耗：Cost 碎银 + 融合石×数量（check-and-consume 同锁防竞态，menu2
// "quench race: material lost after check" 同义）。前置：淬炼书或
// sh_tj 生活职业（menu2 "no book and no sh_tj job"）——生活职业域未
// 移植、书道具无静态标识，一期放行（单机口径）。
//
// 结果：PropPack 池 roll（数量=现有词条数）→ affixes 重写（穿戴行，
// weapon_recast.go 同通道）。第一期只支持穿戴中武器（背包装备行无
// affixes 列）。

import (
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// clientCustomBeginQiangHua CLIENT_CUSTOMMSG_BEGIN_QIANGHUA（client_
// custom_define.lua:574）。
const clientCustomBeginQiangHua int32 = 861

// qiangHuaRule equip_prop_rand 单节（强化规则）。
type qiangHuaRule struct {
	id       int
	cost     int32  // 碎银
	item     string // 融合石 config（默认）
	itemNum  int32  // 融合石数量
	jobName  string // 生活职业
	jobLevel int32
	packs    []int // PropPack 词条池
}

// qiangHuaCatalog equip_prop_rand 目录（节号 → 规则；JobName → 规则列表）。
type qiangHuaCatalog struct {
	rules    []*qiangHuaRule
	byJob    map[string][]*qiangHuaRule
	itemSets map[string][]string // 融合石 config → 规则 id 列表（近似定位）
}

var (
	qiangHuaOnce sync.Once
	qiangHua     *qiangHuaCatalog
)

// currentQiangHuaCatalog 懒加载（weaponRecast 同模式；测试注入优先）。
func currentQiangHuaCatalog() *qiangHuaCatalog {
	qiangHuaOnce.Do(func() {
		if qiangHua != nil {
			return // 测试注入（setQiangHuaCatalogForTest）优先
		}
		qiangHua = loadQiangHuaCatalog(
			filepath.Join(defaultModernShareRoot, "..", "rule", "equip_prop_rand.ini"))
	})
	return qiangHua
}

// setQiangHuaCatalogForTest 单测注入合成目录（生产代码不得调用）。
func setQiangHuaCatalogForTest(catalog *qiangHuaCatalog) {
	qiangHuaOnce.Do(func() {})
	qiangHua = catalog
}

// loadQiangHuaCatalog 装载强化规则表（GBK 键 ASCII 直读）。
func loadQiangHuaCatalog(path string) *qiangHuaCatalog {
	catalog := &qiangHuaCatalog{
		byJob:    make(map[string][]*qiangHuaRule),
		itemSets: make(map[string][]string),
	}
	file, err := openResourceReader(path)
	if err != nil {
		log.Printf("qianghua catalog: equip_prop_rand unavailable: %v", err)
		return catalog
	}
	defer file.Close()
	pending := map[int]*qiangHuaRule{}
	parseINISections(file, func(section, key, value string) {
		id, ok := parseUintSection(section)
		if !ok {
			return
		}
		rule := pending[int(id)]
		if rule == nil {
			rule = &qiangHuaRule{id: int(id)}
			pending[int(id)] = rule
		}
		switch strings.ToLower(key) {
		case "cost":
			if parsed, err := strconv.ParseInt(value, 10, 32); err == nil {
				rule.cost = int32(parsed)
			}
		case "item":
			parts := strings.Split(value, ",")
			rule.item = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				if parsed, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 32); err == nil {
					rule.itemNum = int32(parsed)
				}
			}
			if rule.itemNum <= 0 {
				rule.itemNum = 1
			}
		case "jobname":
			rule.jobName = strings.TrimSpace(value)
		case "joblevel":
			if parsed, err := strconv.ParseInt(value, 10, 32); err == nil {
				rule.jobLevel = int32(parsed)
			}
		case "proppack":
			rule.packs = splitIntList(value)
		}
	})
	// pending 是 map（解析期按下标聚合），遍历序随机：不排序则 byJob
	// 每职业的规则顺序每次起服洗牌，resolveRule 的「首条在册规则」随机
	// 命中不同档位（强化计费漂移 + 测试 flaky，2026-11-12 定位）。INI
	// 节号升序即官方档位序。
	ordered := make([]*qiangHuaRule, 0, len(pending))
	for _, rule := range pending {
		ordered = append(ordered, rule)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].id < ordered[j].id })
	for _, rule := range ordered {
		if len(rule.packs) == 0 || rule.item == "" {
			continue
		}
		catalog.rules = append(catalog.rules, rule)
		catalog.byJob[rule.jobName] = append(catalog.byJob[rule.jobName], rule)
		catalog.itemSets[rule.item] = append(catalog.itemSets[rule.item], strconv.Itoa(rule.id))
	}
	log.Printf("qianghua catalog ready rules=%d", len(catalog.rules))
	return catalog
}

// resolveRule 规则定位（一期近似）：itemreduce 非空按融合石匹配，否则
// jobid 的首条在册规则。
func (catalog *qiangHuaCatalog) resolveRule(jobID, itemReduce string) *qiangHuaRule {
	if catalog == nil || len(catalog.rules) == 0 {
		return nil
	}
	if itemReduce != "" {
		for _, rule := range catalog.byJob[jobID] {
			if rule.item == itemReduce {
				return rule
			}
		}
	}
	if rules := catalog.byJob[jobID]; len(rules) > 0 {
		return rules[0]
	}
	return nil
}

// handleQiangHuaCustom 861 入口（直达/活动信封双路）。
func handleQiangHuaCustom(h *itemSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 ||
		custom.Values[0].Int32 != clientCustomBeginQiangHua {
		return false, nil
	}
	if len(custom.Values) < 4 || custom.Values[1].Type != 2 || custom.Values[2].Type != 2 {
		log.Printf("%s: qianghua 861 malformed args=%d", h.remote, len(custom.Values))
		return true, nil
	}
	view, index := custom.Values[1].Int32, custom.Values[2].Int32
	jobID := strings.TrimSpace(custom.Values[3].Text)
	itemReduce := ""
	if len(custom.Values) > 4 {
		itemReduce = strings.TrimSpace(custom.Values[4].Text)
	}

	// 定位武器：穿戴视图（index=身体槽，须武器 22）或背包装备页
	//（00020 affixes 列就位，洗炼二期扩展 121 页）。
	var (
		wornItem  equipRow
		bagItem   bagRow
		isWorn    bool
	)
	switch view {
	case recastViewEquipWorn:
		var ok bool
		wornItem, ok = h.player.wornItemAt(index)
		if !ok {
			log.Printf("%s: qianghua 861 empty body slot %d", h.remote, index)
			return true, nil
		}
		if index != weaponEquipSlot {
			log.Printf("%s: qianghua 861 body slot %d is not a weapon (%s)", h.remote, index, wornItem.configID)
			return true, nil
		}
		isWorn = true
	case recastViewEquipBag:
		var ok bool
		bagItem, _, ok = bagItemAtForView(h.player, uint16(view), index)
		if !ok {
			log.Printf("%s: qianghua 861 empty bag cell view=%d pos=%d", h.remote, view, index)
			return true, nil
		}
		if !strings.EqualFold(bagItem.equipType, "weapon") {
			log.Printf("%s: qianghua 861 target %s is not a weapon (%s)", h.remote, bagItem.configID, bagItem.equipType)
			return true, nil
		}
	default:
		log.Printf("%s: qianghua 861 view=%d not supported", h.remote, view)
		return true, nil
	}
	targetConfig := bagItem.configID
	targetBuilt := bagItem.propBuilt
	if isWorn {
		targetConfig = wornItem.configID
		targetBuilt = wornItem.propBuilt
	}

	// 规则 + 消耗（check-and-consume 同锁——融合石 check 后被并发用掉
	// 的竞态按拒绝收口，menu2 "quench race" 同义）。
	rule := currentQiangHuaCatalog().resolveRule(jobID, itemReduce)
	if rule == nil {
		log.Printf("%s: qianghua 861 no rule for job=%s item=%q", h.remote, jobID, itemReduce)
		return true, nil
	}
	stone := rule.item
	if itemReduce != "" {
		stone = itemReduce
	}
	if row2, inBag := findBagMountRow(h.player, stone); !inBag || row2.amount < rule.itemNum {
		log.Printf("%s: qianghua 861 material %s x%d not in bag", h.remote, stone, rule.itemNum)
		return true, nil
	}
	if h.player.trySpendSilver(rule.cost) {
		// 碎银帧由 GM 货币域同款通道刷新（简化：直接重推货币属性）。
		if err := h.pushCurrencyAfterSpend(); err != nil {
			log.Printf("%s: qianghua 861 currency frame: %v", h.remote, err)
		}
	} else {
		log.Printf("%s: qianghua 861 silver short cost=%d", h.remote, rule.cost)
		return true, nil
	}
	if row2, _ := findBagMountRow(h.player, stone); !h.consumeBagItemCount(row2, rule.itemNum) {
		// 材料竞态丢失：回滚碎银。
		h.player.addSilver(rule.cost)
		log.Printf("%s: quench race: material lost after check", h.remote)
		return true, nil
	}

	// roll：池均匀抽（数量=现有词条数；无词条按 3，recast 同口径）。
	count := 3
	if targetBuilt != "" {
		count = len(strings.Split(targetBuilt, ";"))
	}
	picked := rollQiangHuaPacks(rule.packs, count)
	if len(picked) == 0 {
		log.Printf("%s: qianghua 861 empty pool rule=%d", h.remote, rule.id)
		return true, nil
	}

	// 应用：affixes 落档 + 内存/帧刷新（recastConfirm 同链；穿戴/背包
	// 按定位视图分流——00020 背包通道）。
	affixes := recastAffixesJSON(currentWeaponRecastCatalog(), picked)
	built := propBuiltOfAffixes(affixes)
	if isWorn {
		if err := h.store.updateEquipAffixes(h.ctx, h.roleID, weaponEquipSlot, affixes); err != nil {
			log.Printf("%s: qianghua 861 persist: %v", h.remote, err)
			return true, nil
		}
		wornItem.propBuilt = built
		h.player.replaceEquipSlot(wornItem)
		if err := pushWearFrames(h.link, h.roleID, h.player.visualSexOf(),
			effectiveShowEquipType(h.player), wornItem, h.player.sessionEquips()); err != nil {
			log.Printf("%s: qianghua 861 wear frames: %v", h.remote, err)
		}
	} else {
		if err := h.store.updateBagAffixes(h.ctx, h.roleID, index, affixes); err != nil {
			log.Printf("%s: qianghua 861 bag persist: %v", h.remote, err)
			return true, nil
		}
		h.player.updateBagRowPropBuilt(uint16(view), index, built)
		if fresh, _, ok := bagItemAtForView(h.player, uint16(view), index); ok {
			if err := pushBagItemFrames(h.link, h.roleID, fresh); err != nil {
				log.Printf("%s: qianghua 861 bag frames: %v", h.remote, err)
			}
			// 清格+重发同槽位也会清掉客户端挂着的模板包/随机属性行，
			// 按原槽位补推（普通武器无包无随机零帧）。
			h.pushRandomPropsForBag(fresh.configID, fresh.uid, fresh.viewID, fresh.slot)
		}
	}
	log.Printf("%s: qianghua 861 config=%s view=%d rule=%d packs=%v cost=%d stone=%s x%d",
		h.remote, targetConfig, view, rule.id, picked, rule.cost, stone, rule.itemNum)
	return true, nil
}

// rollQiangHuaPacks 强化池均匀抽 count 条（不放回；池小于 count 时全给）。
func rollQiangHuaPacks(pool []int, count int) []int {
	if len(pool) == 0 || count <= 0 {
		return nil
	}
	picked := make([]int, 0, count)
	used := make(map[int]bool, count)
	for len(picked) < count && len(used) < len(pool) {
		id := pool[recastRandomSource.Intn(len(pool))]
		if used[id] {
			continue
		}
		used[id] = true
		picked = append(picked, id)
	}
	return picked
}

// pushCurrencyAfterSpend 强化扣碎银后的货币属性重推（skill_combat
// silverUpdate 同通道）。
func (h *itemSession) pushCurrencyAfterSpend() error {
	frame, err := h.player.silverUpdate()
	if err != nil {
		return err
	}
	return h.link.WriteFrame(frame)
}
