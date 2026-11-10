package main

// 镇派兵器动态属性 + 宝物/翡翠娃娃随机属性生成器（第一批）。
//
// 数据源（客户端资源树 <root>/resources/modern/share，GBK；所需字段全为
// ASCII，逐字节解析无损）。权威盘点见 docs/镇派兵器与娃娃宝物随机属性.md：
//   modifypack/proppack.ini            属性包行 r=<序>,<Prop>,<值>,<flag>,<条件ID>
//   rule/condition.ini                 PowerLevel 波段（min/max；实力点数=内功点数）
//   rule/condition_formula.ini         组合条件（倚天剑 24545-24558 = 波段&功能开关）
//   item/equipment.ini                 物品字段（@PropModifyPackRec/TemplateRandomID/…）
//   skill/neigong/neigong_static.ini   StaticData → MinVarPropNo
//   skill/neigong/neigong_varprop.ini  每层 NeiGongLevel = 实力点数（坐忘功等价层数）
//   faculty/facultylevel.ini           点数 → 称号阶梯（0-240 共 32 阶）
//   rule/randomequiprule/*             宝物掷值管线（顶级）
//   rule/randomequiprule/newtersure/*  翡翠娃娃掷值管线
//
// 官服 record 方言（artifacts a-e.pcapng 实证，盘点 §3.6）：0x11 flag=1 =
// 视图对象寻址，objId=viewID、second=objectIndex，一帧可批多行；宝物属性行
// RandomPropRec(400)=[propID,value f64,0,0,0,0]、技能行
// RandomSkillRec(401)=[skillstore下标,0,value f64,0]。

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// ---- 生成结果 ----

const (
	randomKindNone    = 0
	randomKindTreaure = 1 // EquipType=Treasure（ItemType 146）
	randomKindWawa    = 2 // EquipType=NewTreasure（ItemType 193）
)

// randomPropLine 一条随机属性行（RandomPropRec 行镜像；宝物后 4 列官服恒 0，
// 娃娃列 3=有效/4=品质/5=等级）。
type randomPropLine struct {
	PropID  int     `json:"prop"`
	Value   float64 `json:"value"`
	Flag    int     `json:"flag,omitempty"`
	Quality int     `json:"quality,omitempty"`
	Level   int     `json:"level,omitempty"`
}

// randomSkillLine 一条随机技能行（RandomSkillRec=[skillstore下标,SubSkill
// 下标,value,解锁位]——col1 官方实证为 0/1 子技能序号（capture-players
// obj=1-26 同 store 成对 col1=0/1），非技能 id；单 SubSkill 条目恒 0）。
type randomSkillLine struct {
	StoreID int     `json:"store"`
	Sub     int     `json:"sub,omitempty"`
	Value   float64 `json:"value"`
	Unlock  int     `json:"unlock,omitempty"`
}

// itemRandomProps 一件物品的随机属性快照（持久化 + 下发镜像）。
type itemRandomProps struct {
	Kind      int               `json:"kind"`
	PropRows  []randomPropLine  `json:"props,omitempty"`
	SkillRows []randomSkillLine `json:"skills,omitempty"`
}

// neigongPowerBook 实力点数推导输入（learnedNeiGong 的两字段投影）。
type neigongPowerBook struct {
	staticData int32
	level      int32
}

// ---- 目录 ----

type propPackRow struct {
	prop  string
	value float64
	flag  int
	cond  int
}

type powerCond struct {
	name string
	min  int
	max  int // 0 = 无上限
}

type randomItemFields struct {
	equipType  string
	colorLevel int
	level      int // ilvl
	templateID int // TemplateRandomID
	needRandom int
	propPacks  []int // @PropModifyPackRec（条件动态属性包，PropBuilt 类型 1）
	skillPacks []int // @SkillModifyPackRec（套路威力包，PropBuilt 类型 2）
}

type tstoreEntry struct {
	kind   int // 1=propstore / 2=skillstore
	id     int
	ranges []int // 显式 limitelist 行号（alvl 档）；空=-1 通用规则
}

type tstorePack struct {
	num     int
	entries []tstoreEntry
}

type propstoreEntry struct {
	propName string
	limitNo  int
	ratio    int
}

type skillstoreEntry struct {
	subs    [][2]string // (SubSkill, PropName)
	limitNo int
	taolu   string // TaoLu 键（套路 id；洗炼计费查 hhttreasuretaolu [taolu]）
}

type rollRange struct {
	lo, hi, id int
}

type powerTitleBand struct {
	power int
	title string
}

type randomEquipCatalog struct {
	once sync.Once
	err  error

	itemFields map[string]randomItemFields // equipment.ini 目标条目（configID → 字段）
	packRows   map[int][]propPackRow       // proppack 节号 → 行（只收物品引用到的包）
	conds      map[int]powerCond
	formulas   map[int]string // condition_formula：Formula 原文（&/| 组合）
	staticVar  map[int]int    // neigong_static：StaticData → MinVarPropNo
	ngl        map[int]int    // neigong_varprop：节号 → NeiGongLevel（实力点数）
	titles     []powerTitleBand

	// 宝物管线（randomequiprule 顶级）
	templates map[int][]int          // template.ini：模板 → "2=" 包号列表
	tstore    map[int]tstorePack     // templatestore.ini
	propstore map[int]propstoreEntry // propstore.ini
	skstore   map[int]skillstoreEntry
	limitRows map[int][][2]int // limitelist.ini：节号 → r=min,max 行
	maxalvl   map[int][]int    // maxalvl.ini：[ilvl] → 按行 alvl（行=qlvl）
	boxTrea   map[int]int      // boxlevel [Treasure]：ColorLevel → 条数

	// 娃娃管线（newtersure/）
	newtpl  map[int]int            // newtemplate：模板 → base_rule
	baseRul map[int][]rollRange    // baseproprule：千分 roll
	baseLvl map[int]map[int]int    // baseproplevel_<ColorLevel>：[propID][物品Level] → 固定值
	addRul  map[string][]rollRange // addproprule：[Prop]/[Color]
	addLvl  map[int]map[int][2]int // addproplevel_<品质>：[propID][等级1-10] → min,max
	boxWawa map[int]int            // boxlevel [NewTreasure]
}

var (
	randomEquipCatalogOnce sync.Once
	randomEquipCat         *randomEquipCatalog
)

// randomEquipCatalogOf 懒加载全局目录；装载失败返回带 err 的实例
// （首次失败后不再重试；各查询函数对 err 目录返回零值）。
func randomEquipCatalogOf() *randomEquipCatalog {
	randomEquipCatalogOnce.Do(func() {
		randomEquipCat = loadRandomEquipCatalog()
	})
	return randomEquipCat
}

// resetRandomEquipCatalogForTest 仅供测试：重置懒加载（换资源根后重load）。
func resetRandomEquipCatalogForTest() {
	randomEquipCatalogOnce = sync.Once{}
	randomEquipCat = nil
}

func loadRandomEquipCatalog() *randomEquipCatalog {
	c := &randomEquipCatalog{
		itemFields: map[string]randomItemFields{},
		packRows:   map[int][]propPackRow{},
		conds:      map[int]powerCond{},
		formulas:   map[int]string{},
		staticVar:  map[int]int{},
		ngl:        map[int]int{},
		templates:  map[int][]int{},
		tstore:     map[int]tstorePack{},
		propstore:  map[int]propstoreEntry{},
		skstore:    map[int]skillstoreEntry{},
		limitRows:  map[int][][2]int{},
		maxalvl:    map[int][]int{},
		boxTrea:    map[int]int{},
		newtpl:     map[int]int{},
		baseRul:    map[int][]rollRange{},
		baseLvl:    map[int]map[int]int{},
		addRul:     map[string][]rollRange{},
		addLvl:     map[int]map[int][2]int{},
		boxWawa:    map[int]int{},
	}
	// equipment.ini（77 万行）一次扫描：全节段累积伴随字段（键序不定，
	// EquipType 可能晚于 @PropModifyPackRec 出现），扫完统一剪枝；
	// 同时积累属性包引用集（proppack 二次装载的过滤集）。
	wantedPacks := map[int]bool{}
	if err := parseModernINI(modernSharePath("item", "equipment.ini"), func(section, key, value string) error {
		fields := c.itemFields[section]
		switch key {
		case "@PropModifyPackRec":
			if id, err := strconv.Atoi(value); err == nil {
				fields.propPacks = append(fields.propPacks, id)
				wantedPacks[id] = true
			}
		case "@SkillModifyPackRec":
			if id, err := strconv.Atoi(value); err == nil {
				fields.skillPacks = append(fields.skillPacks, id)
			}
		case "TemplateRandomID":
			if id, err := strconv.Atoi(value); err == nil {
				fields.templateID = id
			}
		case "EquipType":
			fields.equipType = value
		case "ColorLevel":
			if v, err := strconv.Atoi(value); err == nil {
				fields.colorLevel = v
			}
		case "Level":
			if v, err := strconv.Atoi(value); err == nil {
				fields.level = v
			}
		case "NeedRandom":
			if v, err := strconv.Atoi(value); err == nil {
				fields.needRandom = v
			}
		default:
			return nil
		}
		c.itemFields[section] = fields
		return nil
	}); err != nil {
		c.err = fmt.Errorf("equipment.ini: %w", err)
		return c
	}
	// 剪枝：无动态包/技能包、无模板、非宝物/娃娃类型的条目全部丢弃。
	for id, fields := range c.itemFields {
		if len(fields.propPacks) == 0 && len(fields.skillPacks) == 0 && fields.templateID == 0 &&
			fields.equipType != "Treasure" && fields.equipType != "NewTreasure" {
			delete(c.itemFields, id)
		}
	}

	// proppack.ini：只收引用到的包。
	if err := parseModernINI(modernSharePath("modifypack", "proppack.ini"), func(section, key, value string) error {
		if key != "r" {
			return nil
		}
		id, err := strconv.Atoi(section)
		if err != nil || !wantedPacks[id] {
			return nil
		}
		parts := strings.Split(value, ",")
		if len(parts) < 5 {
			return nil
		}
		num, parseErr := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if parseErr != nil {
			return nil // 公式行（$…$）非本批目标，跳过
		}
		flag, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
		cond, _ := strconv.Atoi(strings.TrimSpace(parts[4]))
		c.packRows[id] = append(c.packRows[id], propPackRow{
			prop: strings.TrimSpace(parts[1]), value: num, flag: flag, cond: cond,
		})
		return nil
	}); err != nil {
		c.err = fmt.Errorf("proppack.ini: %w", err)
		return c
	}

	// condition.ini / condition_formula.ini。
	if err := parseModernINI(modernSharePath("rule", "condition.ini"), func(section, key, value string) error {
		id, err := strconv.Atoi(section)
		if err != nil {
			return nil
		}
		entry := c.conds[id]
		switch strings.ToLower(key) {
		case "name":
			entry.name = value
		case "min":
			entry.min, _ = strconv.Atoi(value)
		case "max":
			entry.max, _ = strconv.Atoi(value)
		}
		c.conds[id] = entry
		return nil
	}); err != nil {
		c.err = fmt.Errorf("condition.ini: %w", err)
		return c
	}
	if err := parseModernINI(modernSharePath("rule", "condition_formula.ini"), func(section, key, value string) error {
		if key != "Formula" {
			return nil
		}
		if id, err := strconv.Atoi(section); err == nil {
			c.formulas[id] = value
		}
		return nil
	}); err != nil {
		c.err = fmt.Errorf("condition_formula.ini: %w", err)
		return c
	}

	// 内功实力点数（varprop NeiGongLevel）+ StaticData 映射 + 称号阶梯。
	if err := c.loadPowerTables(); err != nil {
		c.err = err
		return c
	}

	// 宝物/娃娃掷值管线（缺表只降级对应功能，不算整体失败）。
	c.loadTreasureTables()
	c.loadWawaTables()
	return c
}

func (c *randomEquipCatalog) loadPowerTables() error {
	if err := parseModernINI(modernSharePath("skill", "neigong", "neigong_static.ini"), func(section, key, value string) error {
		if key != "MinVarPropNo" {
			return nil
		}
		id, err := strconv.Atoi(section)
		if err != nil {
			return nil
		}
		if no, err := strconv.Atoi(value); err == nil {
			c.staticVar[id] = no
		}
		return nil
	}); err != nil {
		return fmt.Errorf("neigong_static.ini: %w", err)
	}
	if err := parseModernINI(modernSharePath("skill", "neigong", "neigong_varprop.ini"), func(section, key, value string) error {
		if key != "NeiGongLevel" {
			return nil
		}
		id, err := strconv.Atoi(section)
		if err != nil {
			return nil
		}
		level, err := strconv.Atoi(value)
		if err != nil {
			return nil
		}
		c.ngl[id] = level
		return nil
	}); err != nil {
		return fmt.Errorf("neigong_varprop.ini: %w", err)
	}
	// varprop 节号 = MinVarPropNo + (层数-1)：按节号排序后每本书取连续段，
	// 段内即层数递增的 NeiGongLevel 曲线（220101..220136 实证连续）。
	var ladder []powerTitleBand
	if err := parseModernINI(modernSharePath("faculty", "facultylevel.ini"), func(section, key, value string) error {
		if section != "config" || key != "r" {
			return nil
		}
		parts := strings.Split(value, ",")
		if len(parts) < 3 {
			return nil
		}
		power, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil
		}
		ladder = append(ladder, powerTitleBand{power: power, title: strings.TrimSpace(parts[2])})
		return nil
	}); err != nil {
		return fmt.Errorf("facultylevel.ini: %w", err)
	}
	sort.Slice(ladder, func(i, j int) bool { return ladder[i].power < ladder[j].power })
	c.titles = ladder
	return nil
}

func (c *randomEquipCatalog) loadTreasureTables() {
	root := "rule"
	_ = parseModernINI(modernSharePath(root, "randomequiprule", "template.ini"), func(section, key, value string) error {
		if key != "2" {
			return nil // "1="（全随机条数）本批未接：模板族全走 "2=" 包
		}
		id, err := strconv.Atoi(section)
		if err != nil {
			return nil
		}
		if pack, err := strconv.Atoi(value); err == nil {
			c.templates[id] = append(c.templates[id], pack)
		}
		return nil
	})
	_ = parseModernINI(modernSharePath(root, "randomequiprule", "templatestore.ini"), func(section, key, value string) error {
		packID, err := strconv.Atoi(section)
		if err != nil {
			return nil
		}
		entry := c.tstore[packID]
		switch key {
		case "num":
			entry.num, _ = strconv.Atoi(value)
		case "1", "2":
			kind, _ := strconv.Atoi(key)
			parts := strings.SplitN(value, "|", 2)
			id, idErr := strconv.Atoi(strings.TrimSpace(parts[0]))
			if idErr != nil {
				return nil
			}
			var ranges []int
			if len(parts) > 1 && strings.TrimSpace(parts[1]) != "-1" {
				for _, field := range strings.Split(parts[1], ",") {
					if no, err := strconv.Atoi(strings.TrimSpace(field)); err == nil {
						ranges = append(ranges, no)
					}
				}
			}
			entry.entries = append(entry.entries, tstoreEntry{kind: kind, id: id, ranges: ranges})
		}
		c.tstore[packID] = entry
		return nil
	})
	_ = parseModernINI(modernSharePath(root, "randomequiprule", "propstore.ini"), func(section, key, value string) error {
		id, err := strconv.Atoi(section)
		if err != nil {
			return nil
		}
		entry := c.propstore[id]
		switch strings.ToLower(key) {
		case "propname":
			entry.propName = value
		case "limitlistno":
			entry.limitNo, _ = strconv.Atoi(value)
		case "ratio":
			entry.ratio, _ = strconv.Atoi(value)
		}
		c.propstore[id] = entry
		return nil
	})
	_ = parseModernINI(modernSharePath(root, "randomequiprule", "skillstore.ini"), func(section, key, value string) error {
		id, err := strconv.Atoi(section)
		if err != nil {
			return nil
		}
		entry := c.skstore[id]
		switch strings.ToLower(key) {
		case "subskill":
			parts := strings.SplitN(value, ",", 2)
			name := ""
			if len(parts) > 1 {
				name = strings.TrimSpace(parts[1])
			}
			entry.subs = append(entry.subs, [2]string{strings.TrimSpace(parts[0]), name})
		case "limitlistno":
			entry.limitNo, _ = strconv.Atoi(value)
		case "taolu":
			entry.taolu = value
		}
		c.skstore[id] = entry
		return nil
	})
	// limitelist（24.7 万行）整表懒装载成本可接受（行短、一次性）。
	_ = parseModernINI(modernSharePath(root, "randomequiprule", "limitelist.ini"), func(section, key, value string) error {
		if key != "r" {
			return nil
		}
		id, err := strconv.Atoi(section)
		if err != nil {
			return nil
		}
		parts := strings.SplitN(value, ",", 2)
		if len(parts) != 2 {
			return nil
		}
		lo, loErr := strconv.Atoi(strings.TrimSpace(parts[0]))
		hi, hiErr := strconv.Atoi(strings.TrimSpace(parts[1]))
		if loErr != nil || hiErr != nil {
			return nil
		}
		c.limitRows[id] = append(c.limitRows[id], [2]int{lo, hi})
		return nil
	})
	_ = parseModernINI(modernSharePath(root, "randomequiprule", "maxalvl.ini"), func(section, key, value string) error {
		if key != "r" {
			return nil
		}
		ilvl, err := strconv.Atoi(section)
		if err != nil {
			return nil
		}
		alvl, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return nil
		}
		c.maxalvl[ilvl] = append(c.maxalvl[ilvl], alvl)
		return nil
	})
	_ = parseModernINI(modernSharePath(root, "randomequiprule", "boxlevel.ini"), func(section, key, value string) error {
		if key != "r" || (section != "Treasure" && section != "NewTreasure") {
			return nil
		}
		parts := strings.Split(value, ",")
		if len(parts) < 3 {
			return nil
		}
		cl, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		count, err2 := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err1 != nil || err2 != nil {
			return nil
		}
		if section == "Treasure" {
			c.boxTrea[cl] = count
		} else {
			c.boxWawa[cl] = count
		}
		return nil
	})
}

func (c *randomEquipCatalog) loadWawaTables() {
	root := []string{"rule", "randomequiprule", "newtersure"}
	_ = parseModernINI(modernSharePath(append(root, "newtemplate.ini")...), func(section, key, value string) error {
		if key != "base_rule" {
			return nil
		}
		id, err := strconv.Atoi(section)
		if err != nil {
			return nil
		}
		rule, _ := strconv.Atoi(value)
		c.newtpl[id] = rule
		return nil
	})
	_ = parseModernINI(modernSharePath(append(root, "baseproprule.ini")...), func(section, key, value string) error {
		if key != "r" {
			return nil
		}
		id, err := strconv.Atoi(section)
		if err != nil {
			return nil
		}
		c.baseRul[id] = append(c.baseRul[id], parseRollRange(value))
		return nil
	})
	// baseproplevel_<ColorLevel 3-6>.ini：[propID] <物品Level>=固定值。
	for color := 3; color <= 6; color++ {
		table := map[int]int{}
		_ = parseModernINI(modernSharePath(append(root, fmt.Sprintf("baseproplevel_%d.ini", color))...), func(section, key, value string) error {
			propID, err := strconv.Atoi(section)
			if err != nil {
				return nil
			}
			itemLevel, err := strconv.Atoi(key)
			if err != nil {
				return nil
			}
			v, err := strconv.Atoi(value)
			if err != nil {
				return nil
			}
			table[itemLevel*10000+propID] = v
			return nil
		})
		c.baseLvl[color] = table
	}
	_ = parseModernINI(modernSharePath(append(root, "addproprule.ini")...), func(section, key, value string) error {
		if key != "r" {
			return nil
		}
		c.addRul[section] = append(c.addRul[section], parseRollRange(value))
		return nil
	})
	for quality := 1; quality <= 4; quality++ {
		table := map[int][2]int{}
		_ = parseModernINI(modernSharePath(append(root, fmt.Sprintf("addproplevel_%d.ini", quality))...), func(section, key, value string) error {
			propID, err := strconv.Atoi(section)
			if err != nil {
				return nil
			}
			lineLevel, err := strconv.Atoi(key)
			if err != nil {
				return nil
			}
			parts := strings.SplitN(value, ",", 2)
			if len(parts) != 2 {
				return nil
			}
			lo, loErr := strconv.Atoi(strings.TrimSpace(parts[0]))
			hi, hiErr := strconv.Atoi(strings.TrimSpace(parts[1]))
			if loErr != nil || hiErr != nil {
				return nil
			}
			table[lineLevel*10000+propID] = [2]int{lo, hi}
			return nil
		})
		c.addLvl[quality] = table
	}
}

func parseRollRange(value string) rollRange {
	parts := strings.Split(value, ",")
	if len(parts) < 3 {
		return rollRange{}
	}
	lo, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	hi, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	id, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
	return rollRange{lo: lo, hi: hi, id: id}
}

func modernSharePath(parts ...string) string {
	all := append([]string{defaultModernShareRoot}, parts...)
	return filepath.Join(all...)
}

// ---- 条件评估（实力点数波段） ----

// satisfiedSwitchConds 单机服默认放行的功能开关条件（倚天剑 24559 =
// 18418|18419|18420|18456|118500 华山论剑开关族；发放即视为可用）。
var satisfiedSwitchConds = map[int]bool{
	24559: true, 18418: true, 18419: true, 18420: true, 18456: true, 118500: true,
}

// condSatisfiedAt 递归评估条件（深度护栏 8）：公式 &/| 组合；叶子为
// PowerLevel 波段（min<=pl<=max，max=0 无上限）或功能开关白名单。
func (c *randomEquipCatalog) condSatisfiedAt(condID, powerLevel, depth int) bool {
	if condID == 0 || depth > 8 {
		return condID == 0
	}
	if formula, ok := c.formulas[condID]; ok && formula != "" {
		if strings.Contains(formula, "&") {
			for _, part := range strings.Split(formula, "&") {
				id, err := strconv.Atoi(strings.TrimSpace(part))
				if err != nil || !c.condSatisfiedAt(id, powerLevel, depth+1) {
					return false
				}
			}
			return true
		}
		if strings.Contains(formula, "|") {
			for _, part := range strings.Split(formula, "|") {
				id, err := strconv.Atoi(strings.TrimSpace(part))
				if err == nil && c.condSatisfiedAt(id, powerLevel, depth+1) {
					return true
				}
			}
			return false
		}
	}
	cond, ok := c.conds[condID]
	if !ok {
		return satisfiedSwitchConds[condID]
	}
	if !strings.EqualFold(cond.name, "PowerLevel") {
		return satisfiedSwitchConds[condID]
	}
	if powerLevel < cond.min {
		return false
	}
	return cond.max == 0 || powerLevel <= cond.max
}

// ---- 镇派兵器 ----

// zhenpaiPropPacksOf 物品的动态属性包列表（equipment.ini @PropModifyPackRec）。
func (c *randomEquipCatalog) zhenpaiPropPacksOf(configID string) []int {
	fields, ok := c.itemFields[configID]
	if !ok {
		return nil
	}
	return fields.propPacks
}

// zhenpaiDynamicValues 按实力点数命中各包条件行 → PropName→值。
// 13 把共用包（11 行×10 级带宽）与倚天剑专属包（14 行×5 级带宽&功能开关）
// 走同一评估；无一命中行的包不计入（数值 0）。
func (c *randomEquipCatalog) zhenpaiDynamicValues(configID string, powerLevel int) map[string]float64 {
	values := map[string]float64{}
	for _, packID := range c.zhenpaiPropPacksOf(configID) {
		for _, row := range c.packRows[packID] {
			if c.condSatisfiedAt(row.cond, powerLevel, 0) {
				values[row.prop] += row.value
				break // 一包只取一个命中行（波段互斥）
			}
		}
	}
	return values
}

// zhenpaiPropBuiltString PropBuilt@1954 串（"N,id;N,id"——N 是表类型列：
// 1=proppack 属性包 / 2=skillpack 套路威力包，tips_func_equip.lua:550-559
// get_text_by_table_id 按类型取 tips_proppack_<id>/tips_skillpack_<id> 文案；
// affixes JSON 的协议等价形态，非 DB 回读路径（bag→wear 内存构造行）用）。
func (c *randomEquipCatalog) zhenpaiPropBuiltString(configID string) string {
	fields, ok := c.itemFields[configID]
	if !ok || (len(fields.propPacks) == 0 && len(fields.skillPacks) == 0) {
		return ""
	}
	segments := make([]string, 0, len(fields.propPacks)+len(fields.skillPacks))
	for _, packID := range fields.propPacks {
		segments = append(segments, fmt.Sprintf("1,%d", packID))
	}
	for _, packID := range fields.skillPacks {
		segments = append(segments, fmt.Sprintf("2,%d", packID))
	}
	return strings.Join(segments, ";")
}

// zhenpaiAffixesJSON 生成品实例的 affixes 列 JSON（包列表形态，kind 列
// 1=属性包/2=技能包；propBuiltOf 按 kind 还原 "N,id" 段；客户端按本地
// proppack/skillpack 渲染 tooltip，条件行由客户端按 PowerLevel 自行命中）。
func (c *randomEquipCatalog) zhenpaiAffixesJSON(configID string) string {
	fields, ok := c.itemFields[configID]
	if !ok || (len(fields.propPacks) == 0 && len(fields.skillPacks) == 0) {
		return ""
	}
	type affixEntry struct {
		PackRec   int `json:"pack_rec"`
		PropBuilt int `json:"prop_built"`
		Kind      int `json:"kind"`
	}
	entries := make([]affixEntry, 0, len(fields.propPacks)+len(fields.skillPacks))
	for _, packID := range fields.propPacks {
		entries = append(entries, affixEntry{PackRec: 0, PropBuilt: packID, Kind: 1})
	}
	for _, packID := range fields.skillPacks {
		entries = append(entries, affixEntry{PackRec: 0, PropBuilt: packID, Kind: 2})
	}
	encoded, err := json.Marshal(entries)
	if err != nil {
		return ""
	}
	return string(encoded)
}

// ---- 实力点数推导与称号 ----

// derivePowerLevel 实力点数 = max(已学内功当前层 varprop.NeiGongLevel)
// （varprop 节号 = MinVarPropNo + 层-1；官服"坐忘功等价"口径，盘点 §2.7）。
func (c *randomEquipCatalog) derivePowerLevel(books []neigongPowerBook) int {
	best := 0
	for _, book := range books {
		base, ok := c.staticVar[int(book.staticData)]
		if !ok || book.level <= 0 {
			continue
		}
		if level, ok := c.ngl[base+int(book.level)-1]; ok && level > best {
			best = level
		}
	}
	return best
}

// titleForPowerLevel 称号键（facultylevel 阶梯；点数≥末档取末档）。
func (c *randomEquipCatalog) titleForPowerLevel(powerLevel int) string {
	title := ""
	for _, band := range c.titles {
		if powerLevel >= band.power {
			title = band.title
		} else {
			break
		}
	}
	return title
}

// ---- 宝物掷值 ----

// rollTreasure 按 randomequiprule 管线掷宝物（黄帝金简 treasure_31111..31115
// 等全族通用）：template → 每包独立掷 num 条（候选均匀抽不放回；官方权重
// 口径未见于资源，批一近似）→ 值=limitelist 行区间均匀 roll。
func (c *randomEquipCatalog) rollTreasure(configID string, rnd *rand.Rand) (itemRandomProps, error) {
	snapshot := itemRandomProps{Kind: randomKindTreaure}
	fields, ok := c.itemFields[configID]
	if !ok || fields.templateID == 0 {
		return snapshot, fmt.Errorf("treasure %s: no template", configID)
	}
	packs := c.templates[fields.templateID]
	if len(packs) == 0 {
		return snapshot, fmt.Errorf("treasure %s: template %d absent", configID, fields.templateID)
	}
	for _, packID := range packs {
		pack, ok := c.tstore[packID]
		if !ok || pack.num <= 0 || len(pack.entries) == 0 {
			continue
		}
		chosen := pickEntries(pack.entries, pack.num, rnd)
		for _, entry := range chosen {
			switch entry.kind {
			case 1: // 属性条
				value, err := c.rollLimitValue(c.propstore[entry.id].limitNo, entry.ranges, fields.level, fields.colorLevel, rnd)
				if err != nil {
					continue
				}
				snapshot.PropRows = append(snapshot.PropRows, randomPropLine{PropID: entry.id, Value: value})
			case 2: // 技能条（skillstore）
				value, err := c.rollLimitValue(c.skstore[entry.id].limitNo, nil, fields.level, fields.colorLevel, rnd)
				if err != nil {
					continue
				}
				snapshot.SkillRows = append(snapshot.SkillRows, randomSkillLine{StoreID: entry.id, Value: value})
			}
		}
	}
	if len(snapshot.PropRows) == 0 && len(snapshot.SkillRows) == 0 {
		return snapshot, fmt.Errorf("treasure %s: roll produced nothing", configID)
	}
	return snapshot, nil
}

// skillStoreTaoLu skillstore 条目的 TaoLu 套路 id（洗炼计费键；空=不在
// hhttreasuretaolu [taolu] 表）。
func (c *randomEquipCatalog) skillStoreTaoLu(storeID int) string {
	return c.skstore[storeID].taolu
}

// treasureWashPool 宝物模板包候选条目并集（kind 1=属性/2=技能）。
// 洗炼与首掷同池（前代服 kept/rolled 行为参照 _r43_full.txt）。
func (c *randomEquipCatalog) treasureWashPool(configID string, kind int) ([]tstoreEntry, error) {
	fields, ok := c.itemFields[configID]
	if !ok || fields.templateID == 0 {
		return nil, fmt.Errorf("treasure %s: no template", configID)
	}
	packs := c.templates[fields.templateID]
	if len(packs) == 0 {
		return nil, fmt.Errorf("treasure %s: template %d absent", configID, fields.templateID)
	}
	var pool []tstoreEntry
	for _, packID := range packs {
		pack, ok := c.tstore[packID]
		if !ok {
			continue
		}
		for _, entry := range pack.entries {
			if entry.kind == kind {
				pool = append(pool, entry)
			}
		}
	}
	if len(pool) == 0 {
		return nil, fmt.Errorf("treasure %s: no kind-%d pool", configID, kind)
	}
	return pool, nil
}

// rollTreasurePropsForWash 属性洗：同一模板池均匀不放回抽 count 条并按
// limitelist 值域重掷（锁定行的保留与前置排序由调用方负责——总数不变，
// 锁定行原样保留）。
func (c *randomEquipCatalog) rollTreasurePropsForWash(configID string, count int, rnd *rand.Rand) ([]randomPropLine, error) {
	if count <= 0 {
		return nil, nil
	}
	fields := c.itemFields[configID]
	pool, err := c.treasureWashPool(configID, 1)
	if err != nil {
		return nil, err
	}
	chosen := pickEntries(pool, count, rnd)
	rolled := make([]randomPropLine, 0, len(chosen))
	for _, entry := range chosen {
		value, err := c.rollLimitValue(c.propstore[entry.id].limitNo, entry.ranges, fields.level, fields.colorLevel, rnd)
		if err != nil {
			continue
		}
		rolled = append(rolled, randomPropLine{PropID: entry.id, Value: value})
	}
	if len(rolled) == 0 {
		return nil, fmt.Errorf("treasure %s: prop wash rolled nothing", configID)
	}
	return rolled, nil
}

// rollTreasureSkillForWash 武学洗：技能池重掷主武学条。skillType 池语义
// （老界面 0/1、新界面 2/3 → 门派/江湖）：江湖=SubSkill 前缀 CS_jh_*，
// 门派=其余——启发式口径，官方过滤规则未见于资源，待抓包校正。
func (c *randomEquipCatalog) rollTreasureSkillForWash(configID string, skillType int, rnd *rand.Rand) (randomSkillLine, error) {
	fields := c.itemFields[configID]
	pool, err := c.treasureWashPool(configID, 2)
	if err != nil {
		return randomSkillLine{}, err
	}
	wantJianghu := skillType == 1 || skillType == 3
	var filtered []tstoreEntry
	for _, entry := range pool {
		sub := ""
		if store, ok := c.skstore[entry.id]; ok && len(store.subs) > 0 {
			sub = store.subs[0][0]
		}
		isJianghu := strings.HasPrefix(sub, "CS_jh_")
		if isJianghu == wantJianghu {
			filtered = append(filtered, entry)
		}
	}
	if len(filtered) == 0 {
		return randomSkillLine{}, fmt.Errorf("treasure %s: no %s skill pool", configID, map[bool]string{true: "jianghu", false: "school"}[wantJianghu])
	}
	entry := filtered[rnd.Intn(len(filtered))]
	value, err := c.rollLimitValue(c.skstore[entry.id].limitNo, nil, fields.level, fields.colorLevel, rnd)
	if err != nil {
		return randomSkillLine{}, err
	}
	return randomSkillLine{StoreID: entry.id, Value: value}, nil
}

// rollLimitValue 值域 roll：显式行号集（templatestore "1=0|51,52,…"）均匀取
// 一行；-1 走通用规则 maxalvl[ilvl][qlvl]；行内 [min,max] 均匀取整。
func (c *randomEquipCatalog) rollLimitValue(limitNo int, rows []int, ilvl, qlvl int, rnd *rand.Rand) (float64, error) {
	table := c.limitRows[limitNo]
	if len(table) == 0 {
		return 0, fmt.Errorf("limitelist %d absent", limitNo)
	}
	row := 0
	if len(rows) > 0 {
		row = rows[rnd.Intn(len(rows))]
	} else {
		alvlTable := c.maxalvl[ilvl]
		if len(alvlTable) == 0 {
			return 0, fmt.Errorf("maxalvl ilvl %d absent", ilvl)
		}
		index := qlvl
		if index >= len(alvlTable) {
			index = len(alvlTable) - 1
		}
		if index < 0 {
			index = 0
		}
		row = alvlTable[index]
	}
	if row < 1 || row > len(table) {
		return 0, fmt.Errorf("limitelist %d row %d out of range", limitNo, row)
	}
	band := table[row-1]
	return float64(band[0] + rnd.Intn(band[1]-band[0]+1)), nil
}

func pickEntries(entries []tstoreEntry, num int, rnd *rand.Rand) []tstoreEntry {
	if num >= len(entries) {
		return append([]tstoreEntry(nil), entries...)
	}
	pool := append([]tstoreEntry(nil), entries...)
	chosen := make([]tstoreEntry, 0, num)
	for len(chosen) < num && len(pool) > 0 {
		index := rnd.Intn(len(pool))
		chosen = append(chosen, pool[index])
		pool = append(pool[:index:index], pool[index+1:]...)
	}
	return chosen
}

// ---- 娃娃掷值 ----

// rollWawa 按 newtersure 管线掷翡翠娃娃：基础属性千分 roll → 固定值 +
// 附加属性（boxlevel[NewTreasure] 按 ColorLevel 定条数，每条 [Prop] roll
// 属性 + [Color] roll 品质 → addproplevel_<品质> 等级1 区间 roll）。
func (c *randomEquipCatalog) rollWawa(configID string, rnd *rand.Rand) (itemRandomProps, error) {
	snapshot := itemRandomProps{Kind: randomKindWawa}
	fields, ok := c.itemFields[configID]
	if !ok {
		return snapshot, fmt.Errorf("wawa %s: unknown item", configID)
	}
	baseRule, ok := c.newtpl[fields.templateID]
	if !ok {
		return snapshot, fmt.Errorf("wawa %s: newtemplate %d absent", configID, fields.templateID)
	}
	if base, ok := rollThousandth(c.baseRul[baseRule], rnd); ok {
		if value, ok := c.baseLvl[fields.colorLevel][fields.level*10000+base]; ok {
			// 官方方言（capture-players timeline obj=1-30..33）：基础行
			// [propID,value,0,0,0,0]——flag=0 无品质前缀；flag=1 会让客户端
			// 拼 ui.newtreasure.quality_<品质> 键，quality=0 无词条直接露键名。
			snapshot.PropRows = append(snapshot.PropRows, randomPropLine{PropID: base, Value: float64(value)})
		} else {
			return snapshot, fmt.Errorf("wawa %s: baseproplevel_%d[%d][%d] absent", configID, fields.colorLevel, base, fields.level)
		}
	}
	addCount := c.boxWawa[fields.colorLevel]
	for i := 0; i < addCount; i++ {
		propID, ok := rollThousandth(c.addRul["Prop"], rnd)
		if !ok {
			break
		}
		quality, ok := rollThousandth(c.addRul["Color"], rnd)
		if !ok {
			quality = 1
		}
		band, ok := c.addLvl[quality][1*10000+propID]
		if !ok {
			continue
		}
		value := band[0] + rnd.Intn(band[1]-band[0]+1)
		snapshot.PropRows = append(snapshot.PropRows, randomPropLine{
			PropID: propID, Value: float64(value), Flag: 1, Quality: quality, Level: 1,
		})
	}
	if len(snapshot.PropRows) == 0 {
		return snapshot, fmt.Errorf("wawa %s: roll produced nothing", configID)
	}
	return snapshot, nil
}

func rollThousandth(ranges []rollRange, rnd *rand.Rand) (int, bool) {
	if len(ranges) == 0 {
		return 0, false
	}
	roll := rnd.Intn(10000)
	for _, band := range ranges {
		if roll >= band.lo && roll <= band.hi {
			return band.id, true
		}
	}
	return 0, false
}

// ---- 物品分类 ----

// randomEquipKind 0=普通 / 1=宝物 / 2=娃娃。
func (c *randomEquipCatalog) randomEquipKind(configID string) int {
	fields, ok := c.itemFields[configID]
	if !ok {
		return randomKindNone
	}
	switch fields.equipType {
	case "Treasure":
		return randomKindTreaure
	case "NewTreasure":
		return randomKindWawa
	}
	return randomKindNone
}

// hasTemplatePacks 是否带固定模板包（@Prop/SkillModifyPackRec）——此类物品
// 词条走 record 394/395 官方通道，PropBuilt 让位 EffectPackID（互斥守护）。
func (c *randomEquipCatalog) hasTemplatePacks(configID string) bool {
	fields, ok := c.itemFields[configID]
	return ok && (len(fields.propPacks) > 0 || len(fields.skillPacks) > 0)
}
