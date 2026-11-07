package main

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// 现代内功效果是数据驱动的，而不是手写的几本
// 入门内功清单。这些是已安装的现代客户端配置表，定义了
// StaticData -> level varprop -> PropModify/SkillModify/Buffer。
var (
	modernNeiGongIniPath     = filepath.Join(defaultModernShareRoot, "skill", "neigong", "neigong.ini")
	modernNeiGongStaticPath  = filepath.Join(defaultModernShareRoot, "skill", "neigong", "neigong_static.ini")
	modernNeiGongVarPropPath = filepath.Join(defaultModernShareRoot, "skill", "neigong", "neigong_varprop.ini")
	modernBuffNewPath        = filepath.Join(defaultModernShareRoot, "skill", "buff_new.ini")
	modernPropPackPath       = filepath.Join(defaultModernShareRoot, "modifypack", "proppack.ini")
	modernSkillPackPath      = filepath.Join(defaultModernShareRoot, "modifypack", "skillpack.ini")
	modernWuXueBaseInfoPath  = filepath.Join(defaultModernShareRoot, "faculty", "wuxuebaseinfo.ini")
	modernWuXueLevelInfoPath = filepath.Join(defaultModernShareRoot, "faculty", "wuxuelevelinfo.ini")
)

// 五行/属性枚举语义（Java NeiGongCatalog.WUXING_*，客户端本地化键
// ui_wuxue_prop_{N}）：技能 1=刚 / 2=柔 / 3=阴 / 4=阳；内功 5=太极 /
// 6=阴柔 / 7=阳刚（BelongType ng_mp_tj/yr/yg 全量交叉验证吻合）。
// 目录缺行的技能按 5（太极）下发，与 Java StarterSkills.skillWuXing 同款。
const wuxingTaiJi int32 = 5

type modernNeiGongModifier struct {
	Target string
	Name   string
	Value  float64
	Mode   int32
}

type modernNeiGongVarProp struct {
	level      int32
	buffID     string
	buffLevel  int32
	direct     map[string]int32
	propPacks  []int32
	skillPacks []int32
}

type modernNeiGongRange struct{ min, max int32 }

// modernNeiGongBook 是 neigong.ini 一本内功的目录身份：configId（QName，
// 如 ng_wd_001）→ StaticData/ItemType/School。内功不在 skill_new.ini
// （game_skills 的 wuxue 域），快照中超出 starter 目录的书从这里回填。
type modernNeiGongBook struct {
	configID   string
	staticData int32
	itemType   int32
	school     string
}

type modernNeiGongEffect struct {
	staticData     uint32
	buffID         string
	buffLevel      int32
	stats          map[string]int32
	propPacks      []int32
	skillPacks     []int32
	skillModifiers map[string][]modernNeiGongModifier
}

type modernNeiGongCatalog struct {
	ranges     map[int32]modernNeiGongRange
	varProps   map[int32]modernNeiGongVarProp
	books      map[string]modernNeiGongBook
	buffStatic map[string]uint32
	propPacks  map[int32][]modernNeiGongModifier
	skillPacks map[int32][]modernNeiGongModifier
	// wuXing 是 faculty/wuxuebaseinfo.ini 的 configId→nWuXing 表
	//（Java NeiGongCatalog.wuXing 同源；内功书与套路技能共用同一目录，
	// mind_ 重复节按节名收录互不冲突）。缺行 0。
	wuXing map[string]int32
	// levelTotals 是 faculty/wuxuelevelinfo.ini 的 ng_* 官方每级修炼
	// 曲线（r=等级,修炼度,速度；Java 注释同源）：configId→level→total。
	// 全部内功书（含江湖 ng_jh_*）统一走此表。
	levelTotals map[string]map[int32]int32
}

var (
	modernNeiGongCatalogOnce sync.Once
	modernNeiGongCatalogData *modernNeiGongCatalog
	modernNeiGongCatalogErr  error
)

func loadModernNeiGongCatalog() (*modernNeiGongCatalog, error) {
	modernNeiGongCatalogOnce.Do(func() {
		catalog := &modernNeiGongCatalog{
			ranges:      make(map[int32]modernNeiGongRange),
			varProps:    make(map[int32]modernNeiGongVarProp),
			books:       make(map[string]modernNeiGongBook),
			buffStatic:  make(map[string]uint32),
			propPacks:   make(map[int32][]modernNeiGongModifier),
			skillPacks:  make(map[int32][]modernNeiGongModifier),
			wuXing:      make(map[string]int32),
			levelTotals: make(map[string]map[int32]int32),
		}
		if err := catalog.load(); err != nil {
			modernNeiGongCatalogErr = err
			return
		}
		modernNeiGongCatalogData = catalog
	})
	return modernNeiGongCatalogData, modernNeiGongCatalogErr
}

func (c *modernNeiGongCatalog) load() error {
	if err := c.loadBooks(); err != nil {
		return err
	}
	if err := parseModernINI(modernNeiGongStaticPath, func(section, key, value string) error {
		id, err := strconv.ParseInt(section, 10, 32)
		if err != nil {
			return nil
		}
		rangeValue := c.ranges[int32(id)]
		switch key {
		case "MinVarPropNo":
			rangeValue.min, err = parseModernInt32(value)
		case "MaxVarPropNo":
			rangeValue.max, err = parseModernInt32(value)
		}
		if err != nil {
			return fmt.Errorf("neigong static [%s] %s=%q: %w", section, key, value, err)
		}
		c.ranges[int32(id)] = rangeValue
		return nil
	}); err != nil {
		return err
	}
	if err := parseModernINI(modernNeiGongVarPropPath, func(section, key, value string) error {
		id, err := strconv.ParseInt(section, 10, 32)
		if err != nil {
			return nil
		}
		record := c.varProps[int32(id)]
		if record.direct == nil {
			record.direct = make(map[string]int32)
		}
		switch key {
		case "Level":
			record.level, err = parseModernInt32(value)
		case "BufferID":
			record.buffID = strings.TrimSpace(value)
		case "BufferLevel":
			record.buffLevel, err = parseModernInt32(value)
		case "@PropModifyPackRec":
			var pack int32
			pack, err = parseModernInt32(value)
			record.propPacks = append(record.propPacks, pack)
		case "@SkillModifyPackRec":
			var pack int32
			pack, err = parseModernInt32(value)
			record.skillPacks = append(record.skillPacks, pack)
		case "StrAdd", "StaAdd", "DexAdd", "IngAdd", "SpiAdd":
			record.direct[key], err = parseModernInt32(value)
		}
		if err != nil {
			return fmt.Errorf("neigong varprop [%s] %s=%q: %w", section, key, value, err)
		}
		c.varProps[int32(id)] = record
		return nil
	}); err != nil {
		return err
	}
	if err := parseModernINI(modernBuffNewPath, func(section, key, value string) error {
		if key != "StaticData" {
			return nil
		}
		staticData, err := strconv.ParseUint(strings.TrimSpace(value), 10, 32)
		if err != nil {
			return fmt.Errorf("buff [%s] StaticData=%q: %w", section, value, err)
		}
		c.buffStatic[section] = uint32(staticData)
		return nil
	}); err != nil {
		return err
	}
	if err := loadModernModifierPacks(modernPropPackPath, c.propPacks); err != nil {
		return err
	}
	if err := loadModernModifierPacks(modernSkillPackPath, c.skillPacks); err != nil {
		return err
	}
	c.loadWuXing()
	c.loadLevelTotals()
	return nil
}

// loadWuXing 解析 wuxuebaseinfo.ini 的 nWuXing 列（Java NeiGongCatalog
// loadAll 末段同源）。best-effort：夹具仓/精简部署缺该文件时保持空表，
// 不让整个内功目录装载失败——WuXing 下发退回缺省值，其余链路不受影响。
func (c *modernNeiGongCatalog) loadWuXing() {
	_ = parseModernINI(modernWuXueBaseInfoPath, func(section, key, value string) error {
		if key != "nWuXing" {
			return nil
		}
		parsed, err := parseModernInt32(value)
		if err != nil {
			return nil
		}
		c.wuXing[section] = parsed
		return nil
	})
}

// wuXingOf 返回 configId 的五行/属性 int（未收录 0；值语义见
// wuxingTaiJi 处注释）。内修面板 form_wuxue_faculty 按它显示
// 太极/阴柔/阳刚（ui_wuxue_prop_N）。
func (c *modernNeiGongCatalog) wuXingOf(configID string) int32 {
	return c.wuXing[configID]
}

// loadLevelTotals 解析 wuxuelevelinfo.ini 的 ng_* 每级修炼曲线
// （r=level,total,speed；只取 total 列）。best-effort：夹具仓/精简部署
// 缺该文件时保持空表——每级消耗回退 +200/层自然曲线，其余链路不受影响。
func (c *modernNeiGongCatalog) loadLevelTotals() {
	_ = parseModernINI(modernWuXueLevelInfoPath, func(section, key, value string) error {
		if key != "r" || !strings.HasPrefix(section, "ng_") {
			return nil
		}
		parts := strings.Split(value, ",")
		if len(parts) < 2 {
			return nil
		}
		level, err := parseModernInt32(parts[0])
		if err != nil || level <= 0 {
			return nil
		}
		total, err := parseModernInt32(parts[1])
		if err != nil || total <= 0 {
			return nil
		}
		curve, exists := c.levelTotals[section]
		if !exists {
			curve = make(map[int32]int32)
			c.levelTotals[section] = curve
		}
		curve[level] = total
		return nil
	})
}

// levelTotalOf 返回官方 wuxuelevelinfo.ini 曲线里 configId 在 level 层的
// 每级修炼消耗；未收录返回 ok=false。
func (c *modernNeiGongCatalog) levelTotalOf(configID string, level int32) (int32, bool) {
	curve, exists := c.levelTotals[configID]
	if !exists {
		return 0, false
	}
	total, exists := curve[level]
	return total, exists && total > 0
}

// resolveNeiGongLevelTotal 返回 configId 在 level 层的每级修炼消耗
// （TotalFillValue@765，客户端内修进度条/内功提示的每级消耗显示源）。
// 全部内功书（含江湖内功）统一走 wuxuelevelinfo.ini 官方曲线——参考服
// 9yin_server1 抓包里的江湖内功 750×2^(L-1) 翻倍曲线（ng_jh_001 L7=
// 48000、ng_jh_201 L14=6144000）数值过于夸张，刻意不镜像（2026-11-07
// 用户拍板）。ok=false = 无曲线依据（缺资源目录/缺行），调用方保留既有
// 自然曲线。
func resolveNeiGongLevelTotal(configID string, level int32) (int32, bool) {
	if level <= 0 {
		return 0, false
	}
	if catalog, err := loadModernNeiGongCatalog(); err == nil {
		return catalog.levelTotalOf(configID, level)
	}
	return 0, false
}

// loadBooks 解析 neigong.ini 的书目录（QName → StaticData/ItemType/
// School）。文件同时含经典节（[ng_wd_001]）与 mind_ 重复节
// （[mind_ng_wd_001]，QName 键指向同一本书但 Buff 明确隐藏）。
func (c *modernNeiGongCatalog) loadBooks() error {
	fields := make(map[string]string)
	section := ""
	flush := func() {
		if section != "" {
			mergeNeiGongBookSection(c.books, section, fields)
		}
	}
	if err := parseModernINI(modernNeiGongIniPath, func(name, key, value string) error {
		if name != section {
			flush()
			section = name
			fields = make(map[string]string)
		}
		fields[key] = strings.TrimSpace(value)
		return nil
	}); err != nil {
		return err
	}
	flush()
	return nil
}

// mergeNeiGongBookSection 把一个小节并进书目录：经典节（节名=QName）
// 是官方可见 Buff 路线，始终覆盖；mind_ 重复节仅在该书尚无记录时
// 兜底，不覆盖任何已有条目（与 starter 目录对两套记录的取舍一致）。
func mergeNeiGongBookSection(books map[string]modernNeiGongBook, section string, fields map[string]string) {
	qname := section
	classic := !strings.HasPrefix(section, "mind_")
	if !classic {
		qname = fields["QName"]
	}
	if !strings.HasPrefix(qname, "ng_") {
		return
	}
	staticData, err := parseModernInt32(fields["StaticData"])
	if err != nil || staticData <= 0 {
		return
	}
	itemType, err := parseModernInt32(fields["ItemType"])
	if err != nil || itemType <= 0 {
		itemType = 1002
	}
	if !classic {
		if _, exists := books[qname]; exists {
			return
		}
	}
	books[qname] = modernNeiGongBook{
		configID:   qname,
		staticData: staticData,
		itemType:   itemType,
		school:     fields["School"],
	}
}

// book 返回目录身份；快照恢复用它回填 starter 目录之外的书。
func (c *modernNeiGongCatalog) book(configID string) (modernNeiGongBook, bool) {
	entry, exists := c.books[configID]
	return entry, exists
}

// maxLevelOf 返回该 StaticData varprop 区间内的最大等级，作为快照缺
// max_level 时的显示上限推导。
func (c *modernNeiGongCatalog) maxLevelOf(staticData int32) int32 {
	entry, exists := c.ranges[staticData]
	if !exists || entry.min <= 0 || entry.max < entry.min {
		return 0
	}
	var best int32
	for id := entry.min; id <= entry.max; id++ {
		if candidate, exists := c.varProps[id]; exists && candidate.level > best {
			best = candidate.level
		}
	}
	return best
}

func (c *modernNeiGongCatalog) effect(staticData, level int32) (modernNeiGongEffect, error) {
	entry, exists := c.ranges[staticData]
	if !exists || entry.min <= 0 || entry.max < entry.min {
		return modernNeiGongEffect{}, fmt.Errorf("missing modern neigong StaticData=%d range", staticData)
	}
	var record modernNeiGongVarProp
	found := false
	for id := entry.min; id <= entry.max; id++ {
		candidate, exists := c.varProps[id]
		if exists && candidate.level == level {
			record, found = candidate, true
			break
		}
	}
	if !found {
		return modernNeiGongEffect{}, fmt.Errorf("StaticData=%d has no varprop at level=%d", staticData, level)
	}
	effect := modernNeiGongEffect{
		buffID:         record.buffID,
		buffLevel:      record.buffLevel,
		stats:          make(map[string]int32, len(record.direct)+8),
		propPacks:      append([]int32(nil), record.propPacks...),
		skillPacks:     append([]int32(nil), record.skillPacks...),
		skillModifiers: make(map[string][]modernNeiGongModifier),
	}
	for name, value := range record.direct {
		effect.stats[name] += value
	}
	if effect.buffID != "" {
		effect.staticData = c.buffStatic[effect.buffID]
		if effect.staticData == 0 {
			return modernNeiGongEffect{}, fmt.Errorf("BufferID=%q has no StaticData in buff_new.ini", effect.buffID)
		}
	}
	for _, id := range record.propPacks {
		for _, modifier := range c.propPacks[id] {
			if modifier.Target == "" && modifier.Mode == 0 {
				effect.stats[modifier.Name] += int32(modifier.Value)
			}
		}
	}
	for _, id := range record.skillPacks {
		for _, modifier := range c.skillPacks[id] {
			if modifier.Target == "" {
				continue
			}
			effect.skillModifiers[modifier.Target] = append(effect.skillModifiers[modifier.Target], modifier)
		}
	}
	return effect, nil
}

func parseModernINI(path string, consume func(section, key, value string) error) error {
	file, err := openResourceReader(path)
	if err != nil {
		return err
	}
	defer file.Close()
	section := ""
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || section == "" {
			continue
		}
		if err := consume(section, strings.TrimSpace(key), strings.TrimSpace(value)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func loadModernModifierPacks(path string, destination map[int32][]modernNeiGongModifier) error {
	return parseModernINI(path, func(section, key, value string) error {
		if key != "r" {
			return nil
		}
		id, err := parseModernInt32(section)
		if err != nil {
			return nil
		}
		modifier, ok := parseModernModifier(value)
		if ok {
			destination[id] = append(destination[id], modifier)
		}
		return nil
	})
}

func parseModernModifier(value string) (modernNeiGongModifier, bool) {
	parts := strings.Split(value, ",")
	for len(parts) > 0 && strings.TrimSpace(parts[0]) == "" {
		parts = parts[1:]
	}
	if len(parts) < 3 {
		return modernNeiGongModifier{}, false
	}
	// 属性包为 Name,Value,Mode；技能包为
	// SkillID,Name,Value,Mode。第三个字段用于区分二者。
	if _, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
		amount, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		mode, _ := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 32)
		return modernNeiGongModifier{Name: strings.TrimSpace(parts[0]), Value: amount, Mode: int32(mode)}, true
	}
	if len(parts) < 4 {
		return modernNeiGongModifier{}, false
	}
	amount, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if err != nil {
		return modernNeiGongModifier{}, false
	}
	mode, _ := strconv.ParseInt(strings.TrimSpace(parts[3]), 10, 32)
	return modernNeiGongModifier{Target: strings.TrimSpace(parts[0]), Name: strings.TrimSpace(parts[1]), Value: amount, Mode: int32(mode)}, true
}

func parseModernInt32(value string) (int32, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
	return int32(parsed), err
}
