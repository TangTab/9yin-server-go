// combat_calc.go：formula.ini → combatcalc.FormulaSpec 解析与引擎装配
// （2026-11-08 数值重写批次）。formula.ini 与 skill_new.ini 同目录
// （share/skill/），键全 ASCII、GBK 中文只在 Comment 值里，不经解析。
package main

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/local/9yin-go-server/internal/combatcalc"
)

// combatFormulaSpecs 是启动期解析的 formula.ini 规格缓存（GM lua_reload
// 热重载引擎时复用，不重读文件）。
var combatFormulaSpecs []combatcalc.FormulaSpec

// loadCombatFormulaSpecs 解析 formula.ini 全部公式节：[-1]/[0] 为列模板节
// （非数字/无 Formula 值）自动跳过；SrcN/PropN 按 1..12 参数位读入。
func loadCombatFormulaSpecs() ([]combatcalc.FormulaSpec, error) {
	table, err := loadINISections(filepath.Join(modernSkillRoot, "formula.ini"))
	if err != nil {
		return nil, fmt.Errorf("load formula.ini: %w", err)
	}
	specs := make([]combatcalc.FormulaSpec, 0, len(table))
	for section, fields := range table {
		id, err := strconv.ParseInt(section, 10, 32)
		if err != nil || id <= 0 {
			continue // [-1]/[0] 列模板节
		}
		formula := strings.TrimSpace(iniValue(fields, "Formula"))
		if formula == "" {
			continue
		}
		spec := combatcalc.FormulaSpec{
			ID:      int32(id),
			Formula: formula,
			Src:     make(map[int]int, maxFormulaParamCount),
			Prop:    make(map[int]string, maxFormulaParamCount),
		}
		for n := 1; n <= maxFormulaParamCount; n++ {
			if src := strings.TrimSpace(iniValue(fields, fmt.Sprintf("Src%d", n))); src != "" {
				if value, err := strconv.Atoi(src); err == nil {
					spec.Src[n] = value
				}
			}
			if prop := strings.TrimSpace(iniValue(fields, fmt.Sprintf("Prop%d", n))); prop != "" {
				spec.Prop[n] = prop
			}
		}
		specs = append(specs, spec)
	}
	slices.SortFunc(specs, func(a, b combatcalc.FormulaSpec) int { return cmp.Compare(a.ID, b.ID) })
	return specs, nil
}

// maxFormulaParamCount 与 combatcalc 引擎的参数位上限一致（Prop1..Prop12）。
const maxFormulaParamCount = 12
