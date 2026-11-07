package main

// 身份装/时装目录（Java FashionSuitCatalog 的 Go 移植）：客户端
// ini/form/fashion_suit.ini（角色面板「时装」界面总目录）。
//
// 两组段（[base_info] 钉死组名）：
//   - 身份装（shengfen 组 school/force/newschool/shijia/guild/life/activity）：
//     段内 r=<configid> 按身份等级排布，equipment.ini 里 ItemType=180/181、
//     EquipType=Suit/NewSuit，穿戴占纸娃娃槽 16；角色按 faction 自动拥有
//     所属门派/势力/新门派段（grantIdentitySuitsAtLogin）。
//   - 时装（fashion 组 hat/coat/shoes/suit 四段）：EquipType=FashionHat/
//     FashionCoat/FashionShoes，穿戴占槽 17-19。
//
// configid 均为 ASCII，原始字节读（同 equipment.ini 约定）。目录缺失/
// 未解析时空目录优雅降级（GM 动作据此报错、登录自动发放跳过）。

import (
	"os"
	"path/filepath"
	"strings"
)

// fashionSuitSections 时装四段段名（[base_info] fashion= 行钉死的组名）。
var fashionSuitSections = []string{"hat", "coat", "shoes", "suit"}

// identityPlainSections 身份装通用段名（[base_info] shengfen= 行中不带
// 前缀列表的组：世家/帮会/生活/活动——仅 GM 手动发放）。
var identityPlainSections = []string{"shijia", "guild", "life", "activity"}

type fashionSuitCatalog struct {
	// sections 段名（原样大小写）→ configid 顺序列表（r= 行序）。
	sections map[string][]string
	order    []string
}

// defaultFashionFormRoot 现代资源树的 form 目录（fashion_suit.ini 所在，
// F 服布局 resources/modern/ini/form；G 仓库树缺该文件时目录为空）。
// NINEYIN_FORM_ROOT 可覆盖（NINEYIN_SHARE_ROOT 之于 equipment.ini 同款）。
func defaultFashionFormRoot() string {
	if custom := strings.TrimSpace(os.Getenv("NINEYIN_FORM_ROOT")); custom != "" {
		return custom
	}
	return runtimeProjectPath("resources", "modern", "ini", "form")
}

func loadFashionSuitCatalog(root string) *fashionSuitCatalog {
	catalog := &fashionSuitCatalog{sections: map[string][]string{}}
	parseINI(filepath.Join(root, "fashion_suit.ini"), func(section, key, value string) {
		if !strings.EqualFold(key, "r") || value == "" {
			return
		}
		if _, seen := catalog.sections[section]; !seen {
			catalog.order = append(catalog.order, section)
		}
		catalog.sections[section] = append(catalog.sections[section], value)
	})
	return catalog
}

// fashionSuits 全局目录（equipmentModels 同款包级缓存；测试经
// injectStubFashionCatalog 换表）。
var fashionSuits = loadFashionSuitCatalog(defaultFashionFormRoot())

func (c *fashionSuitCatalog) isEmpty() bool {
	return len(c.sections) == 0
}

// identitySuitsOf 角色所属身份装清单：faction 值即段名（force_yihua→
// [force_yihua]、newschool_tianya→[newschool_tianya]）；旧门派 faction 不带
// 前缀（如 shaolin），补 school_ 前缀命中 [school_shaolin]。未知门派返回空。
func (c *fashionSuitCatalog) identitySuitsOf(faction string) []string {
	if faction == "" {
		return nil
	}
	trimmed := strings.TrimSpace(faction)
	if suits, ok := c.sections[trimmed]; ok {
		return append([]string(nil), suits...)
	}
	if suits, ok := c.sections["school_"+trimmed]; ok {
		return append([]string(nil), suits...)
	}
	return nil
}

// fashionSuitsAll 时装四段合集（hat→coat→shoes→suit 段序，configid 去重）。
func (c *fashionSuitCatalog) fashionSuitsAll() []string {
	seen := map[string]bool{}
	suits := make([]string, 0, 64)
	for _, section := range fashionSuitSections {
		for _, id := range c.sections[section] {
			if !seen[id] {
				seen[id] = true
				suits = append(suits, id)
			}
		}
	}
	return suits
}

// allIdentitySuits 全部身份装（shengfen 组各段合集，去重；GM/校准用）。
func (c *fashionSuitCatalog) allIdentitySuits() []string {
	seen := map[string]bool{}
	suits := make([]string, 0, 64)
	for _, section := range c.order {
		if !isIdentitySection(section) {
			continue
		}
		for _, id := range c.sections[section] {
			if !seen[id] {
				seen[id] = true
				suits = append(suits, id)
			}
		}
	}
	return suits
}

// isFashionSuit 时装目录 configid（四段内；穿戴校验用，大小写不敏感）。
func (c *fashionSuitCatalog) isFashionSuit(configID string) bool {
	if configID == "" {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(configID))
	for _, suit := range c.fashionSuitsAll() {
		if strings.ToLower(suit) == normalized {
			return true
		}
	}
	return false
}

// isIdentitySection 身份装段判定：school_/force_/newschool_ 前缀 + 世家/
// 帮会/生活/活动通用段名。
func isIdentitySection(section string) bool {
	lower := strings.ToLower(section)
	for _, prefix := range []string{"school_", "force_", "newschool_"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	for _, plain := range identityPlainSections {
		if lower == plain {
			return true
		}
	}
	return false
}
