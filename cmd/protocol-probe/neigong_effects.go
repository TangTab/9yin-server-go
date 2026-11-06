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
	modernNeiGongStaticPath  = filepath.Join(defaultModernShareRoot, "skill", "neigong", "neigong_static.ini")
	modernNeiGongVarPropPath = filepath.Join(defaultModernShareRoot, "skill", "neigong", "neigong_varprop.ini")
	modernBuffNewPath        = filepath.Join(defaultModernShareRoot, "skill", "buff_new.ini")
	modernPropPackPath       = filepath.Join(defaultModernShareRoot, "modifypack", "proppack.ini")
	modernSkillPackPath      = filepath.Join(defaultModernShareRoot, "modifypack", "skillpack.ini")
)

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
	buffStatic map[string]uint32
	propPacks  map[int32][]modernNeiGongModifier
	skillPacks map[int32][]modernNeiGongModifier
}

var (
	modernNeiGongCatalogOnce sync.Once
	modernNeiGongCatalogData *modernNeiGongCatalog
	modernNeiGongCatalogErr  error
)

func loadModernNeiGongCatalog() (*modernNeiGongCatalog, error) {
	modernNeiGongCatalogOnce.Do(func() {
		catalog := &modernNeiGongCatalog{
			ranges:     make(map[int32]modernNeiGongRange),
			varProps:   make(map[int32]modernNeiGongVarProp),
			buffStatic: make(map[string]uint32),
			propPacks:  make(map[int32][]modernNeiGongModifier),
			skillPacks: make(map[int32][]modernNeiGongModifier),
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
	return nil
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
