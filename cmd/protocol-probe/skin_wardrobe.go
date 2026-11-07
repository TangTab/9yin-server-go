package main

// 风物志衣橱上行链路（Java LoginFlow.handleWardrobeCard/handleBodyItemUse
// 卡片分支 + SkinStore，docs/装备外观与风物志挂载.md V3）：
//
//   - C2S 180 [180, sub, card_id]：sub 0=USE 挂载（须已收录；同部件互斥
//     替换：武器换武器/衣服换衣服，发型等异部件叠加）、1=CANCEL 卸载、
//     7=COLLECT 收录（背包有对应卡片道具则消耗 1 张，不自动挂载）；
//     其余 sub（2 武技/3 兑换/5 替换/6 显隐/8 领奖/12 套装）记日志待接。
//   - C2S 31 USEITEM 命中卡片道具（script=CardItem，tool_item.ini
//     [card_item_*].CardID 直连皮肤号）→ consumeBagItemCount 1 张收录。
//
// 状态：role_skins（collected/mounted）；整刷走 pushSkinRefreshFrames。

import (
	"log"
	"strconv"
	"strings"
)

// cardItemCatalogOnce 测试注入点（itemCatalogOnce 同款）。
var cardItemCatalogOnce func() map[string]int32

var cardItemCatalogLoaded map[string]int32

// cardItemBySkinCatalog tool_item.ini [configId].CardID 全表
// （configId 小写 → 皮肤号；[card_item_*] 段即卡片道具）。
func cardItemBySkinCatalog() map[string]int32 {
	if cardItemCatalogOnce != nil {
		return cardItemCatalogOnce()
	}
	if cardItemCatalogLoaded == nil {
		loaded := map[string]int32{}
		table := loadOptionalINI("tool_item.ini", defaultModernShareRoot, "item")
		for section, fields := range table {
			for _, field := range fields {
				if !strings.EqualFold(field.key, "CardID") {
					continue
				}
				if id, err := strconv.Atoi(strings.TrimSpace(field.value)); err == nil {
					loaded[strings.ToLower(section)] = int32(id)
				}
			}
		}
		cardItemCatalogLoaded = loaded
		log.Printf("skin card item catalog ready: %d card items", len(loaded))
	}
	return cardItemCatalogLoaded
}

// cardItemOfSkin 皮肤号 → 卡片道具 configId（线性反查，tool_item.ini
// card_item_* 量级千级、低频调用；Java ToolItemCatalog.cardItemBySkinId 同构）。
func cardItemOfSkin(skinID int32) (string, bool) {
	for configID, id := range cardItemBySkinCatalog() {
		if id == skinID {
			return configID, true
		}
	}
	return "", false
}

// skinIdOfItem 道具 configId → 皮肤号（USEITEM 分流用；非卡片返回 false）。
func skinIdOfItem(configID string) (int32, bool) {
	id, ok := cardItemBySkinCatalog()[strings.ToLower(strings.TrimSpace(configID))]
	return id, ok
}

// handleWardrobeCardCustom 处理 C2S 180（[180, sub, card_id]）。未知皮肤
// 号忽略；三个已接子操作后统一 saveSkinState + 整刷。
func handleWardrobeCardCustom(h *itemSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomWardrobeCard {
		return false, nil
	}
	if h.player == nil {
		log.Printf("%s: ignore wardrobe card before role selected", h.remote)
		return true, nil
	}
	args := customIntArgs(custom)
	if len(args) < 2 {
		log.Printf("%s: card expects TVarList(180, int sub, int card_id), got %d values", h.remote, len(custom.Values))
		return true, nil
	}
	sub, skinID := args[0], args[1]
	if skinID <= 0 {
		log.Printf("%s: card reject: skin id %d", h.remote, skinID)
		return true, nil
	}
	entry, known := skinCards.lookup(int(skinID))
	if !known {
		log.Printf("%s: card ignore: unknown skin %d", h.remote, skinID)
		return true, nil
	}
	state := h.player.skinSnapshot()
	sex := h.player.visualSexOf()
	switch sub {
	case skinCardSubCollect:
		if state.collectedContains(skinID) {
			return true, nil // 幂等
		}
		h.consumeCardItemOfSkin(skinID)
		state.addCollected(skinID)
		log.Printf("%s: card collect skin=%d (%s)", h.remote, skinID, entry.configID)
	case skinCardSubUse:
		if !state.collectedContains(skinID) {
			log.Printf("%s: card use ignore: skin %d not collected", h.remote, skinID)
			return true, nil // USE 从不自动收录
		}
		if state.mountedContains(skinID) {
			return true, nil // 幂等
		}
		parts := entry.parts(sex)
		mounted := make([]int32, 0, len(state.mounted)+1)
		for _, id := range state.mounted {
			other, ok := skinCards.lookup(int(id))
			if !ok {
				mounted = append(mounted, id) // 未知挂载保留（不破坏存档）
				continue
			}
			// 同部件互斥替换：对方部件与本皮肤有交集则先卸载。
			conflict := false
			for part := range other.parts(sex) {
				if parts[part] {
					conflict = true
					break
				}
			}
			if !conflict {
				mounted = append(mounted, id)
			}
		}
		state.mounted = append(mounted, skinID)
		log.Printf("%s: card use skin=%d parts=[%s]", h.remote, skinID, skinPartSummary(entry, sex))
	case skinCardSubCancel:
		if !state.mountedContains(skinID) {
			return true, nil // 幂等
		}
		state.removeMounted(skinID)
		log.Printf("%s: card cancel skin=%d", h.remote, skinID)
	default:
		log.Printf("%s: card sub %d not wired (skin=%d)", h.remote, sub, skinID)
		return true, nil
	}
	if err := h.skins.Save(h.roleID, state); err != nil {
		log.Printf("%s: persist skins role=%d: %v", h.remote, h.roleID, err)
	}
	h.player.applySkins(state)
	if err := pushSkinRefreshFrames(h.link, h.player); err != nil {
		log.Printf("%s: skin refresh: %v", h.remote, err)
	}
	return true, nil
}

// consumeCardItemOfSkin 收录时消耗背包卡片道具 1 张（无卡片不失败——
// GM grant_all_cards 等直通链路无卡片可耗）。
func (h *itemSession) consumeCardItemOfSkin(skinID int32) {
	configID, ok := cardItemOfSkin(skinID)
	if !ok {
		return
	}
	h.player.mu.Lock()
	var item bagRow
	found := false
	for _, row := range h.player.bag {
		if strings.EqualFold(row.configID, configID) {
			item, found = row, true
			break
		}
	}
	h.player.mu.Unlock()
	if !found {
		return
	}
	if !h.consumeBagItemCount(item, 1) {
		log.Printf("%s: card consume %s failed (row missing/changed)", h.remote, configID)
	}
}

// collectSkinFromItemUse C2S 31 卡片收录分支（item_use.go 路由）：消耗 1
// （失败不收录）→ collected 添加（不自动挂载）→ 存档 + 整刷。
func collectSkinFromItemUse(h *itemSession, item bagRow) {
	skinID, ok := skinIdOfItem(item.configID)
	if !ok || skinID <= 0 {
		return
	}
	if !h.consumeBagItemCount(item, 1) {
		return
	}
	state := h.player.skinSnapshot()
	if state.collectedContains(skinID) {
		// 已收录：卡片照耗（收藏复用），状态不变。
		return
	}
	state.addCollected(skinID)
	if err := h.skins.Save(h.roleID, state); err != nil {
		log.Printf("%s: persist skins role=%d: %v", h.remote, h.roleID, err)
	}
	h.player.applySkins(state)
	if err := pushSkinRefreshFrames(h.link, h.player); err != nil {
		log.Printf("%s: skin refresh: %v", h.remote, err)
	}
	log.Printf("%s: use item %s -> collect skin=%d", h.remote, item.configID, skinID)
}
