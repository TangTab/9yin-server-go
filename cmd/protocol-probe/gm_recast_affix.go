package main

// GM 精确包号词条操作（面板「洗练」区词条目录勾选；equip.js 前端契约
// recast_add_affix/recast_clear_affix 的后端实现）：
//
//	recast_add_affix    slot(身上装备槽)+packIDs(词条包号)+affixKind
//	                    (prop→proppack / skill→skillpack) → 对该槽已穿
//	                    装备追加词条。双表同号包（13234 族）的客户端显示
//	                    由 kind 决定：prop=破防、skill=招式威力。
//	recast_clear_affix  slot → 清空该槽装备全部词条。
//
// 词条落地与武器重铸确认（weapon_recast.go recastConfirm 穿戴分支）同链：
// 内存 propBuilt 反解旧词条 → 追加/清空 → affixes JSON 落库（UPDATE
// role_equip_items.affixes）→ 内存行刷新 → 穿戴帧重发（PropBuilt@1954
// tooltip 通道即时生效）。

import (
	"fmt"
	"strconv"
	"strings"
)

// applyRecastAffixCommand 词条操作两动作总入口（processGMCommand 分派）。
func applyRecastAffixCommand(h *itemSession, action string, slot int32, packIDs []int32, affixKind string) (string, error) {
	switch action {
	case "recast_add_affix":
		return applyRecastAddAffix(h, slot, packIDs, affixKind)
	case "recast_clear_affix":
		return applyRecastClearAffix(h, slot)
	}
	return "", fmt.Errorf("unsupported recast affix action %q", action)
}

// applyRecastAddAffix recast_add_affix：对已穿装备追加词条（kind 按
// affixKind 映射，prop→1 / skill→2；13234 族双表同号包的显示归属由
// 此决定）。旧词条从内存 propBuilt 反解（与 DB affixes JSON 一致——
// 生成/重铸/词条操作全链写入时两者同源）。
func applyRecastAddAffix(h *itemSession, slot int32, packIDs []int32, affixKind string) (string, error) {
	row, ok := h.player.wornItemAt(slot)
	if !ok {
		return "", fmt.Errorf("槽 %d 未穿装备，请先穿戴或选择正确部位", slot)
	}
	kind := 1
	if affixKind == "skill" {
		kind = 2
	}
	entries := propBuiltToAffixEntries(row.propBuilt)
	added := make([]gmAffixEntry, 0, len(packIDs))
	for _, id := range packIDs {
		added = append(added, gmAffixEntry{Pack: int(id), Kind: kind})
	}
	entries = append(entries, added...)
	return persistRecastAffixes(h, row, entries,
		fmt.Sprintf("已对槽 %d 追加词条 %d 条（共 %d 条）", slot, len(added), len(entries)))
}

// applyRecastClearAffix recast_clear_affix：清空该槽装备全部词条。
func applyRecastClearAffix(h *itemSession, slot int32) (string, error) {
	row, ok := h.player.wornItemAt(slot)
	if !ok {
		return "", fmt.Errorf("槽 %d 未穿装备，请先穿戴或选择正确部位", slot)
	}
	removed := len(propBuiltToAffixEntries(row.propBuilt))
	return persistRecastAffixes(h, row, nil,
		fmt.Sprintf("已清空槽 %d 词条（原 %d 条）", slot, removed))
}

// persistRecastAffixes 词条落档公共链：affixes JSON 落库 + 内存行刷新 +
// 穿戴帧重发（recastConfirm 穿戴分支同链；无库降级不阻断）。
func persistRecastAffixes(h *itemSession, row equipRow, entries []gmAffixEntry, receipt string) (string, error) {
	affixes := "[]"
	if len(entries) > 0 {
		affixes = gmAffixesJSONEntries(entries)
	}
	if err := h.store.updateEquipAffixes(h.ctx, h.roleID, row.slot, affixes); err != nil {
		return "", fmt.Errorf("词条落库失败: %w", err)
	}
	row.propBuilt = propBuiltOfAffixes(affixes)
	h.player.replaceEquipSlot(row)
	if framesErr := pushWearFrames(h.link, h.roleID, h.player.visualSexOf(),
		effectiveShowEquipType(h.player), row, h.player.sessionEquips()); framesErr != nil {
		return "", framesErr
	}
	return receipt, nil
}

// propBuiltToAffixEntries PropBuilt 串（"kind,pack;kind,pack"）→ 条目
// 列表（propBuiltOfAffixes 互逆；kind 越界按 1 兜底、pack 非正丢弃）。
func propBuiltToAffixEntries(propBuilt string) []gmAffixEntry {
	if propBuilt == "" {
		return nil
	}
	entries := make([]gmAffixEntry, 0, 4)
	for _, segment := range strings.Split(propBuilt, ";") {
		fields := strings.Split(segment, ",")
		if len(fields) != 2 {
			continue
		}
		kind, kindErr := strconv.Atoi(strings.TrimSpace(fields[0]))
		if kindErr != nil || kind < 1 || kind > 5 {
			kind = 1
		}
		pack, packErr := strconv.Atoi(strings.TrimSpace(fields[1]))
		if packErr != nil || pack <= 0 {
			continue
		}
		entries = append(entries, gmAffixEntry{Pack: pack, Kind: kind})
	}
	return entries
}
