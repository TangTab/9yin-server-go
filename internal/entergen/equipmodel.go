package entergen

// 身体槽位模型推导（2026-11-23 官服穿/脱真机定版，docs/official/
// 初始号无装备身体模型定版.md 勘误段）：
//
//	槽空（真无装备）→ jianghu000 裸身族（b/g_cloth000/pants000/shoes000）；
//	穿 {cloth|pants|shoes}_{b|g}NNNN → obj\char\{s}_jianghu{NNN}\{s}_{part}{NNN}
//	（布衣 b0001/g0001 → 001 族全三件；#25 的 001+003 混搭是穿装列，非默认）。
//
// 官服通道口径：玩家外观走 op10 单属性推送（进场物化在 op10 对象属性簇、
// 穿/脱即时单推），玩家不发 0x0D。Hat 槽仍回填发型（帽模型映射未定版）。

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// 身体槽位（equipbox.ini [EquipType] 口径；官服 m30 真机印证 3/4/8）。
const (
	EquipSlotHat   = 1
	EquipSlotCloth = 3
	EquipSlotPants = 4
	EquipSlotShoes = 8
)

// clothPhyDef 布衣族单件物防（官服穿脱推送实证：三件 PhyDef 18↔0）。
const clothPhyDef = 6

// bodySexChar 性别 → 模型族字符（男 b/女 g； Sex 1=男 2=女，非 2 一律男）。
func bodySexChar(sex int) string {
	if sex == 2 {
		return "g"
	}
	return "b"
}

// bareBodyModel 槽空时的裸身模型（jianghu000 族）。
func bareBodyModel(sexChar, part string) string {
	return fmt.Sprintf(`obj\char\%s_jianghu000\%s_%s000`, sexChar, sexChar, part)
}

// clothSeriesModel 布衣系号 → 模型路径（0001→001 族）。series 取
// ConfigID 尾部数字（1..999），越界回落裸身。
func clothSeriesModel(sexChar, part string, series int) string {
	if series < 1 || series > 999 {
		return bareBodyModel(sexChar, part)
	}
	return fmt.Sprintf(`obj\char\%s_jianghu%03d\%s_%s%03d`, sexChar, series, sexChar, part, series)
}

// bodyPartOfSlot 槽位 → 部位名（模型路径与 ConfigID 前缀共用词汇）。
func bodyPartOfSlot(slot int) string {
	switch slot {
	case EquipSlotCloth:
		return "cloth"
	case EquipSlotPants:
		return "pants"
	case EquipSlotShoes:
		return "shoes"
	default:
		return ""
	}
}

// bodySeriesOfConfig 布衣 ConfigID `{part}_{b|g}NNNN` → 系号（NNNN 数字）。
// 前缀与性别不符或无数字尾 → 0（回落裸身）。
func bodySeriesOfConfig(configID string) int {
	under := strings.LastIndexByte(configID, '_')
	if under < 0 || under+2 >= len(configID) {
		return 0
	}
	series, err := strconv.Atoi(configID[under+2:])
	if err != nil {
		return 0
	}
	return series
}

// EquipSlotModel 槽位当前应下发的身体模型路径（装备驱动：穿=件模型、
// 空/未知=裸身 000 族）。帽槽（1）按「无帽回填发型」语义返回发型模型
// （帽件模型映射未定版，穿帽暂仍显发型）。
func EquipSlotModel(state *rolestate.RoleState, slot int) string {
	if slot == EquipSlotHat {
		return HairModel(state)
	}
	part := bodyPartOfSlot(slot)
	if part == "" {
		return ""
	}
	sexChar := bodySexChar(state.Sex)
	for _, equip := range state.Equips {
		if equip.Slot != slot {
			continue
		}
		if !strings.HasPrefix(equip.ConfigID, part+"_") {
			break // 槽上挂着非本部位件（异常档）：回落裸身
		}
		return clothSeriesModel(sexChar, part, bodySeriesOfConfig(equip.ConfigID))
	}
	return bareBodyModel(sexChar, part)
}

// HairModel 当前发型模型（建角外观槽 [4]，缺席回退分支默认）。
func HairModel(state *rolestate.RoleState) string {
	if len(state.Appearance) > 4 && state.Appearance[4] != "" {
		return state.Appearance[4]
	}
	if state.Sex == 2 {
		return appearanceHairG
	}
	return appearanceHairB
}

// EquipPhyDef 装备物防合计（官服白板 0 基线 + 布衣族每件 +6）。
func EquipPhyDef(state *rolestate.RoleState) int32 {
	total := int32(0)
	for _, equip := range state.Equips {
		if bodyPartOfSlot(equip.Slot) != "" && strings.Contains(equip.ConfigID, "_") {
			total += clothPhyDef
		}
	}
	return total
}

// EquipSlotPropName 槽位 → 模型属性名（op10 单推用）。
func EquipSlotPropName(slot int) string {
	switch slot {
	case EquipSlotHat:
		return "Hat"
	case EquipSlotCloth:
		return "Cloth"
	case EquipSlotPants:
		return "Pants"
	case EquipSlotShoes:
		return "Shoes"
	default:
		return ""
	}
}
