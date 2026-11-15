package main

// 经脉逐层属性与穴位目录（官方式数据链，2026-11-15 经脉对齐工程）。
//
// 数据链（全部 ASCII 字段，字节扫描即可；官方每层段是累积值，直接
// 取值、严禁跨层相加）：
//
//	jingmai_static.ini   StaticData → VarPropNo 区间（正经 216/跷 225/
//	                     余 180；目录侧已解析）
//	jingmai_varprop.ini  VarPropNo → Level + @PropModifyPackRec（可重复）
//	modifypack/proppack.ini  PackRec 段内 r=,AttrName,Value,0 行
//	ini/ui/wuxue/wuxue_jingmai.ini  每经脉 XueWei 有序列表（穴位地图）
//	skill/xuewei/xuewei.ini         穴位 ItemType=1004（StaticData=穴序）
//	skill/xuewei/xuewei_activerule.ini  [内功id] r=<层>;<穴id>,… 开穴规则
//	skill/jingmai/jingmai_active_exclude.ini  激活互斥（跷↔跷·隐）
//
// proppack.ini 3.7MB、varprop/xuewei 表均为静态资源：sync.Once 装载一次
// 全局缓存；loadJingMaiCatalog 每连接调用，只引用不重解析。

import (
	"bufio"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// ---- 逐层属性集合（proppack r= 行 AttrName 原名映射；列名=字段尾名） ----

// jingMaiAttrSet 单条经脉某一周天层的官方累积属性。五维走 effectiveFive
// 推导链；maxHP/maxMP 等为独立 Add 列（与五维公式分量不重复计）。
type jingMaiAttrSet struct {
	str, sta, dex, ing, spi          int32
	maxHPAdd, maxMPAdd               int32
	maxParryAdd                      int32
	minMagicDefAdd                   int32
	meleePowerAdd, shotPowerAdd      int32
	magicPowerAdd                    int32
	phyHitAdd, magicHitAdd, dodgeAdd int32
	phyDefAdd, vaDefAdd              int32
	phyVaAdd, magicVaAdd             int32
	phyVaProbAdd, magicVaProbAdd     int32
	magicParryDamResAdd              int32
	parryUpSpeedAdd                  int32
	hpUpSpeedAdd, mpUpSpeedAdd       int32
	stiDefAdd, negDefAdd             int32
	masDefAdd, jujDefAdd             int32
	phyParryDamResAdd, phyDamageAdd  int32
}

// add 累加另一集合（bonus 汇总用）。
func (set *jingMaiAttrSet) add(other jingMaiAttrSet) {
	set.str += other.str
	set.sta += other.sta
	set.dex += other.dex
	set.ing += other.ing
	set.spi += other.spi
	set.maxHPAdd += other.maxHPAdd
	set.maxMPAdd += other.maxMPAdd
	set.maxParryAdd += other.maxParryAdd
	set.minMagicDefAdd += other.minMagicDefAdd
	set.meleePowerAdd += other.meleePowerAdd
	set.shotPowerAdd += other.shotPowerAdd
	set.magicPowerAdd += other.magicPowerAdd
	set.phyHitAdd += other.phyHitAdd
	set.magicHitAdd += other.magicHitAdd
	set.dodgeAdd += other.dodgeAdd
	set.phyDefAdd += other.phyDefAdd
	set.vaDefAdd += other.vaDefAdd
	set.phyVaAdd += other.phyVaAdd
	set.magicVaAdd += other.magicVaAdd
	set.phyVaProbAdd += other.phyVaProbAdd
	set.magicVaProbAdd += other.magicVaProbAdd
	set.magicParryDamResAdd += other.magicParryDamResAdd
	set.parryUpSpeedAdd += other.parryUpSpeedAdd
	set.hpUpSpeedAdd += other.hpUpSpeedAdd
	set.mpUpSpeedAdd += other.mpUpSpeedAdd
	set.stiDefAdd += other.stiDefAdd
	set.negDefAdd += other.negDefAdd
	set.masDefAdd += other.masDefAdd
	set.jujDefAdd += other.jujDefAdd
	set.phyParryDamResAdd += other.phyParryDamResAdd
	set.phyDamageAdd += other.phyDamageAdd
}

// applyRow 按 proppack AttrName 原名归并一行（未知列静默忽略——与参考
// 生成脚本 ATTR_MAP 同集，表内出现的新列不炸）。
func (set *jingMaiAttrSet) applyRow(name string, value int32) {
	switch name {
	case "StrAdd":
		set.str += value
	case "StaAdd":
		set.sta += value
	case "DexAdd":
		set.dex += value
	case "IngAdd":
		set.ing += value
	case "SpiAdd":
		set.spi += value
	case "MaxHPAdd":
		set.maxHPAdd += value
	case "MaxMPAdd":
		set.maxMPAdd += value
	case "MaxParryAdd":
		set.maxParryAdd += value
	case "MinMagicDefAdd":
		set.minMagicDefAdd += value
	case "MeleePowerAdd":
		set.meleePowerAdd += value
	case "ShotPowerAdd":
		set.shotPowerAdd += value
	case "MagicPowerAdd":
		set.magicPowerAdd += value
	case "PhyHitAdd":
		set.phyHitAdd += value
	case "MagicHitAdd":
		set.magicHitAdd += value
	case "DodgeAdd":
		set.dodgeAdd += value
	case "PhyDefAdd":
		set.phyDefAdd += value
	case "VaDefAdd":
		set.vaDefAdd += value
	case "PhyVaAdd":
		set.phyVaAdd += value
	case "MagicVaAdd":
		set.magicVaAdd += value
	case "PhyVaProbAdd":
		set.phyVaProbAdd += value
	case "MagicVaProbAdd":
		set.magicVaProbAdd += value
	case "MagicParryDamResAdd":
		set.magicParryDamResAdd += value
	case "ParryUpSpeedAdd":
		set.parryUpSpeedAdd += value
	case "HPUpSpeedAdd":
		set.hpUpSpeedAdd += value
	case "MPUpSpeedAdd":
		set.mpUpSpeedAdd += value
	case "StiDefAdd":
		set.stiDefAdd += value
	case "NegDefAdd":
		set.negDefAdd += value
	case "MasDefAdd":
		set.masDefAdd += value
	case "JujDefAdd":
		set.jujDefAdd += value
	case "PhyParryDamResAdd":
		set.phyParryDamResAdd += value
	case "PhyDamageAdd":
		set.phyDamageAdd += value
	}
}

// ---- 官方表全局缓存（sync.Once；缺文件降级为零表，bonus 归零不炸） ----

var (
	defaultJingMaiVarPropINIPath = filepath.Join(defaultModernShareRoot, "skill", "jingmai", "jingmai_varprop.ini")
	defaultPropPackINIPath       = filepath.Join(defaultModernShareRoot, "modifypack", "proppack.ini")
	defaultXueWeiINIPath         = filepath.Join(defaultModernShareRoot, "skill", "xuewei", "xuewei.ini")
	defaultXueWeiRuleINIPath     = filepath.Join(defaultModernShareRoot, "skill", "xuewei", "xuewei_activerule.ini")
	defaultJingMaiExcludePath    = filepath.Join(defaultModernShareRoot, "skill", "jingmai", "jingmai_active_exclude.ini")
	defaultUIWuXueJingMaiPath    = filepath.Join(runtimeProjectPath("resources", "modern", "ini"), "ui", "wuxue", "wuxue_jingmai.ini")

	jingMaiTablesOnce sync.Once
	// jingMaiVarPropPacks: VarPropNo → PackRec 列表（@PropModifyPackRec
	// 可重复出现，保留全部）。
	jingMaiVarPropPacks map[int32][]int32
	// jingMaiPropPackRows: PackRec → r=,AttrName,Value,0 行。
	jingMaiPropPackRows map[int32][]jingMaiPropPackRow

	jingMaiXueWeiMu     sync.RWMutex
	globalXueWeiCatalog *xueWeiCatalog
)

// propPackRow proppack.ini 单行 r=,AttrName,Value,0（按 AttrName 原名
// 归并进 jingMaiAttrSet）。
type jingMaiPropPackRow struct {
	name  string
	value int32
}

// loadJingMaiTablesOnce 装载 varprop + proppack 全表（一次性）。任一文件
// 缺失只记日志，对应查询返回零值——激活加成自动降级为零。
func loadJingMaiTablesOnce() {
	jingMaiTablesOnce.Do(func() {
		packs := make(map[int32][]int32)
		if file, err := openResourceReader(defaultJingMaiVarPropINIPath); err == nil {
			section := int64(0)
			scanner := bufio.NewScanner(file)
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
					continue
				}
				if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
					parsed, parseErr := strconv.ParseInt(strings.Trim(line, "[]"), 10, 32)
					if parseErr != nil {
						section = 0
						continue
					}
					section = parsed
					continue
				}
				key, value, found := strings.Cut(line, "=")
				if !found || section == 0 {
					continue
				}
				if strings.TrimSpace(key) != "@PropModifyPackRec" {
					continue
				}
				parsed, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
				if parseErr != nil {
					continue
				}
				packs[int32(section)] = append(packs[int32(section)], int32(parsed))
			}
			file.Close()
		} else {
			log.Printf("jingmai attrs: varprop table unavailable (%s): %v", defaultJingMaiVarPropINIPath, err)
		}

		rows := make(map[int32][]jingMaiPropPackRow)
		if file, err := openResourceReader(defaultPropPackINIPath); err == nil {
			section := int64(0)
			scanner := bufio.NewScanner(file)
			scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}
				if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
					parsed, parseErr := strconv.ParseInt(strings.Trim(line, "[]"), 10, 32)
					if parseErr != nil {
						section = 0
						continue
					}
					section = parsed
					continue
				}
				key, value, found := strings.Cut(line, "=")
				if !found || section == 0 || strings.TrimSpace(key) != "r" {
					continue
				}
				// r=,AttrName,Value,0
				parts := strings.Split(value, ",")
				if len(parts) < 3 {
					continue
				}
				name := strings.TrimSpace(parts[1])
				parsed, parseErr := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 32)
				if parseErr != nil || name == "" {
					continue
				}
				rows[int32(section)] = append(rows[int32(section)], jingMaiPropPackRow{name: name, value: int32(parsed)})
			}
			file.Close()
		} else {
			log.Printf("jingmai attrs: proppack table unavailable (%s): %v", defaultPropPackINIPath, err)
		}
		jingMaiVarPropPacks = packs
		jingMaiPropPackRows = rows
	})
}

// jingMaiAttrSetForVarProp 汇总某 VarPropNo 全部 pack 行成一层属性集合。
func jingMaiAttrSetForVarProp(varPropNo int32) (jingMaiAttrSet, bool) {
	loadJingMaiTablesOnce()
	packs, ok := jingMaiVarPropPacks[varPropNo]
	if !ok {
		return jingMaiAttrSet{}, false
	}
	var set jingMaiAttrSet
	any := false
	for _, pack := range packs {
		for _, row := range jingMaiPropPackRows[pack] {
			set.applyRow(row.name, row.value)
			any = true
		}
	}
	return set, any
}

// attachJingMaiAttrs 按目录区间给每条经脉填 perLevel（逐层累积值直取）。
// 任一层缺表则该层跳过（bonus 查表时按参考服语义回落：该层→满层→零）。
func (catalog *jingMaiCatalog) attachJingMaiAttrs() {
	if catalog == nil {
		return
	}
	loadJingMaiTablesOnce()
	if len(jingMaiVarPropPacks) == 0 || len(jingMaiPropPackRows) == 0 {
		return
	}
	for index := range catalog.order {
		def := &catalog.order[index]
		if def.maxLevel <= 0 || def.minVarPropNo <= 0 {
			continue
		}
		perLevel := make(map[int32]jingMaiAttrSet, def.maxLevel)
		for level := int32(1); level <= def.maxLevel; level++ {
			if set, ok := jingMaiAttrSetForVarProp(def.minVarPropNo + level - 1); ok {
				perLevel[level] = set
			}
		}
		def.perLevel = perLevel
	}
}

// attrSetForLevel 官方逐层累积表查询（参考服语义：该层→满层→零）。
func (def jingMaiDef) attrSetForLevel(level int32) jingMaiAttrSet {
	if level < 1 {
		level = 1
	}
	if set, ok := def.perLevel[level]; ok {
		return set
	}
	if set, ok := def.perLevel[def.maxLevel]; ok {
		return set
	}
	return jingMaiAttrSet{}
}

// ---- 穴位目录（View44 / 开穴规则 / 激活互斥） ----

// xueWeiDef xuewei.ini 单穴（ItemType=1004）。
type xueWeiDef struct {
	configID   string
	staticData int32
}

// xueWeiRule activerule 单条：<内功层>;<穴id>。
type xueWeiRule struct {
	level    int32
	xueWeiID string
}

type xueWeiCatalog struct {
	defs     map[string]xueWeiDef    // 穴 id → 定义
	byStatic map[int32]string        // staticData → 穴 id（仅调试/排序用）
	rules    map[string][]xueWeiRule // 内功 configID → 开穴规则
	excludes map[string][]string     // 经脉 id → 互斥经脉 id 列表
	total    int                     // 穴位总数（View44 容量上限）
}

// xueWeiCatalogOf 穴位/规则/互斥表懒加载入口（双检锁；缺文件降级空
// 目录：无自动开穴、无互斥、View44 仍可装 GM 手开的穴）。测试可经
// setXueWeiCatalogForTest 注入 stub 目录后确定性断言帧数。
func xueWeiCatalogOf() *xueWeiCatalog {
	jingMaiXueWeiMu.RLock()
	catalog := globalXueWeiCatalog
	jingMaiXueWeiMu.RUnlock()
	if catalog != nil {
		return catalog
	}
	jingMaiXueWeiMu.Lock()
	defer jingMaiXueWeiMu.Unlock()
	if globalXueWeiCatalog == nil {
		loadXueWeiCatalogAssign()
	}
	return globalXueWeiCatalog
}

// setXueWeiCatalogForTest 测试注入（同包；生产不得调用）。
func setXueWeiCatalogForTest(catalog *xueWeiCatalog) {
	jingMaiXueWeiMu.Lock()
	globalXueWeiCatalog = catalog
	jingMaiXueWeiMu.Unlock()
}

func loadXueWeiCatalogAssign() {
	{
		catalog := &xueWeiCatalog{
			defs:     make(map[string]xueWeiDef),
			byStatic: make(map[int32]string),
			rules:    make(map[string][]xueWeiRule),
			excludes: make(map[string][]string),
		}
		if file, err := openResourceReader(defaultXueWeiINIPath); err == nil {
			scanner := bufio.NewScanner(file)
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			var current string
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
					continue
				}
				if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
					current = strings.Trim(line, "[]")
					continue
				}
				key, value, found := strings.Cut(line, "=")
				if !found || current == "" {
					continue
				}
				def := catalog.defs[current]
				def.configID = current
				switch strings.TrimSpace(key) {
				case "StaticData":
					parsed, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
					def.staticData = int32(parsed)
				}
				catalog.defs[current] = def
			}
			file.Close()
			for id, def := range catalog.defs {
				catalog.byStatic[def.staticData] = id
			}
			catalog.total = len(catalog.defs)
		} else {
			log.Printf("jingmai xuewei: catalog unavailable (%s): %v", defaultXueWeiINIPath, err)
		}

		if file, err := openResourceReader(defaultXueWeiRuleINIPath); err == nil {
			scanner := bufio.NewScanner(file)
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			var neigong string
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
					continue
				}
				if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
					neigong = strings.Trim(line, "[]")
					continue
				}
				key, value, found := strings.Cut(line, "=")
				if !found || neigong == "" || strings.TrimSpace(key) != "r" {
					continue
				}
				for _, item := range strings.Split(value, ",") {
					levelText, xwID, itemOK := strings.Cut(strings.TrimSpace(item), ";")
					if !itemOK || xwID == "" {
						continue
					}
					level, parseErr := strconv.ParseInt(strings.TrimSpace(levelText), 10, 32)
					if parseErr != nil {
						continue
					}
					catalog.rules[neigong] = append(catalog.rules[neigong],
						xueWeiRule{level: int32(level), xueWeiID: xwID})
				}
			}
			file.Close()
		} else {
			log.Printf("jingmai xuewei: activerule unavailable (%s): %v", defaultXueWeiRuleINIPath, err)
		}

		if file, err := openResourceReader(defaultJingMaiExcludePath); err == nil {
			scanner := bufio.NewScanner(file)
			inExclude := false
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
					continue
				}
				if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
					inExclude = strings.Trim(line, "[]") == "exclude"
					continue
				}
				if !inExclude {
					continue
				}
				key, value, found := strings.Cut(line, "=")
				if !found {
					continue
				}
				for _, target := range strings.Split(value, ",") {
					target = strings.TrimSpace(target)
					if target != "" {
						catalog.excludes[strings.TrimSpace(key)] = append(catalog.excludes[strings.TrimSpace(key)], target)
					}
				}
			}
			file.Close()
		} else {
			log.Printf("jingmai xuewei: active_exclude unavailable (%s): %v", defaultJingMaiExcludePath, err)
		}
		globalXueWeiCatalog = catalog
	}
}

// attachXueWeiOrder 把 UI ini 的每经脉穴位有序表挂进目录（穴位地图顺序，
// open_xuewei all 与阶梯档位的计数基准）。
func (catalog *jingMaiCatalog) attachXueWeiOrder() {
	if catalog == nil {
		return
	}
	file, err := openResourceReader(defaultUIWuXueJingMaiPath)
	if err != nil {
		log.Printf("jingmai xuewei: ui map unavailable (%s): %v", defaultUIWuXueJingMaiPath, err)
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	current := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.Trim(line, "[]")
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || current == "" || strings.TrimSpace(key) != "XueWei" {
			continue
		}
		index, ok := catalog.byID[current]
		if !ok {
			continue
		}
		order := make([]string, 0, 12)
		for _, id := range strings.Split(value, ",") {
			if id = strings.TrimSpace(id); id != "" {
				order = append(order, id)
			}
		}
		catalog.order[index].xueWei = order
	}
}

// jingMaiBonusExtraProps 经脉官方逐层表中不属五维/资源/招架/内防既有
// 推导列的战斗面板 Add 列（值 0 不发；0x09 表按名可解析才发——方言
// 换表自动跟随，缺名静默跳过）。HP/MPUpSpeedAdd 走 effectiveFive 分数
// 系数链（stateProperties），不在此直发。
func jingMaiBonusExtraProps(bonus jingMaiAttrSet) []clientdata.IndexedProperty {
	type extraColumn struct {
		name  string
		value int32
	}
	columns := []extraColumn{
		{"PhyVaProbAdd", bonus.phyVaProbAdd},
		{"MagicVaProbAdd", bonus.magicVaProbAdd},
		{"MagicParryDamResAdd", bonus.magicParryDamResAdd},
		{"ParryUpSpeedAdd", bonus.parryUpSpeedAdd},
		{"PhyDefAdd", bonus.phyDefAdd},
		{"VaDefAdd", bonus.vaDefAdd},
		{"StiDefAdd", bonus.stiDefAdd},
		{"NegDefAdd", bonus.negDefAdd},
		{"MasDefAdd", bonus.masDefAdd},
		{"JujDefAdd", bonus.jujDefAdd},
		{"PhyParryDamResAdd", bonus.phyParryDamResAdd},
		{"PhyDamageAdd", bonus.phyDamageAdd},
	}
	props := make([]clientdata.IndexedProperty, 0, len(columns))
	for _, column := range columns {
		if column.value == 0 {
			continue
		}
		entry, ok := clientdata.PlayerProperty(column.name)
		if !ok {
			continue
		}
		props = append(props, clientdata.IndexedProperty{
			Index: entry.Index, Name: column.name, Value: clientdata.Int32Value(column.value),
		})
	}
	return props
}

// ---- 周天上限阶梯（用户拍板 2026-11-15） ----
//
// 初始 9 周天；每开 1 穴 +36，封顶目录上限；跷脉（maxLevel 225）特例
// 3 穴 ×75=225；穴位全开时直取目录上限（带脉 3 穴 36/72/180 这类
// 阶梯缺口由“开满即封顶”兜齐）。
const (
	jingMaiInitialCap  int32 = 9
	jingMaiCapStep     int32 = 36
	jingMaiQiaoCapStep int32 = 75
	jingMaiQiaoMax     int32 = 225
	// jingMaiActLimitDefault 可激活经脉条数缺省（官方初始 3 条；GM
	// set_jingmai_act_count 可调 1..8）。
	jingMaiActLimitDefault int32 = 3
	jingMaiActLimitMax     int32 = 8
)

// jingMaiCapFor 阶梯上限：openedXueWei=该经脉已开穴位数，total=穴位总数。
func jingMaiCapFor(maxLevel, openedXueWei, total int32) int32 {
	if maxLevel <= 0 {
		return jingMaiInitialCap
	}
	if maxLevel <= jingMaiInitialCap {
		return maxLevel
	}
	if openedXueWei <= 0 {
		return jingMaiInitialCap
	}
	if total > 0 && openedXueWei >= total {
		return maxLevel // 开满即封顶（兜齐阶梯缺口）
	}
	step := jingMaiCapStep
	if maxLevel == jingMaiQiaoMax {
		step = jingMaiQiaoCapStep
	}
	cap := step * openedXueWei
	if cap > maxLevel {
		cap = maxLevel
	}
	if cap < jingMaiInitialCap {
		cap = jingMaiInitialCap
	}
	return cap
}
