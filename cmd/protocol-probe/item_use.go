package main

// msg31 使用物品（Java LoginFlow.handleBodyItemUse :4823 + useItemEffect :5188）：
// 装备→右键穿；皮肤卡（CardItem）→fwz 域未接；学习类 script→useSkillBook
//（必须在效果链之前，防 itemType 撞型被当药白吃）；其余→useItemEffect 五步
//（节流→Lua 钩子优先→INI 回退→效果落账→消耗帧对）。简化决策全部照
// docs/item-use-msg31.md §5/§6：temp_stat 仅剩时不消耗、grants 部分失败
// 仍走消耗、不发 sysinfo。

import (
	"log"
	"math/rand"
	"time"
)

// luaItemUseInstruction 是 scripts/item/<configId>.lua on_use 的解析产物
// （ItemModule_ 契约；当前 Lua 宿主未接物品域——钩子默认 nil 走 INI，
// 宿主接入后实现该变量即可，使用链结构不变）。
type luaItemUseInstruction struct {
	healHP     int32
	healMP     int32
	grants     []grantItem
	openDropID string
	rolls      int32
	tempStat   bool
	consume    bool
}

// luaItemUseHook 按钮义注入点（无脚本/引擎坏→返回 nil=INI 回退，永不拒绝）。
var luaItemUseHook func(configID string, amount int32) *luaItemUseInstruction

// itemUseThrottle 使用节流（会话级 800ms 防连点，Java 同值）。
const itemUseThrottle = 800 * time.Millisecond

// handleBodyItemUse：恰好 4 值 [31, view(int), slot(int), count(int≥1)]
// （menu2.exe 字符串钉死），值数不符/count=0 拒绝。
func handleBodyItemUse(h *itemSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomUseItem {
		return false, nil
	}
	if len(custom.Values) != 4 {
		log.Printf("%s: reject malformed native item use values=%d (need exactly 4)", h.remote, len(custom.Values))
		return true, nil
	}
	args := customIntArgs(custom)
	if len(args) < 3 {
		log.Printf("%s: reject malformed native item use: non-int arg", h.remote)
		return true, nil
	}
	view, slot, count := args[0], args[1], args[2]
	if count < 1 {
		log.Printf("%s: reject native item use: count=%d", h.remote, count)
		return true, nil
	}
	item, _, ok := h.player.bagItemAt(uint16(bagViewOf(bagCategoryOf(view))), slot)
	if !ok {
		log.Printf("%s: native item use reject: bag[%d:%d] empty", h.remote, view, slot)
		return true, nil
	}
	script := itemScriptOf(h, item)
	if wearableEquipType(item) != "" {
		wearFromBag(h, view, slot, 0)
		return true, nil
	}
	// 风物志卡片收录（Java 按 CardID>0 分流——script=CardItem 是弱信号，
	// tool_item.ini [card_item_*].CardID 才是权威）：消耗 1 张收录进
	// 衣橱，不自动挂载。
	if _, isCard := skinIdOfItem(item.configID); isCard {
		collectSkinFromItemUse(h, item)
		return true, nil
	}
	switch lowerASCII(script) {
	case "carditem":
		log.Printf("%s: use item %s carditem without CardID mapping", h.remote, item.configID)
		return true, nil
	}
	if isLearnScript(script) {
		useSkillBook(h, item, script)
		return true, nil
	}
	useItemEffect(h, item, view)
	return true, nil
}

// itemScriptOf 解析行为类（game_items.script；无 DB 行回退行内推导为空）。
func itemScriptOf(h *itemSession, item bagRow) string {
	if h.store != nil {
		if entry, ok := h.store.lookupGrantable(h.ctx, item.configID); ok {
			return entry.script
		}
	}
	return ""
}

// useItemEffect 道具使用效果核心（bak SceneFlow.useItem 五步对齐）。
func useItemEffect(h *itemSession, item bagRow, view int32) {
	now := h.now()
	h.player.mu.Lock()
	throttled := now.Sub(h.player.lastItemUseAt) < itemUseThrottle
	h.player.mu.Unlock()
	if throttled {
		return
	}

	// ① 效果解析：Lua 脚本优先（无脚本/nil 返回 → nil → INI 回退）。
	var instruction *luaItemUseInstruction
	if luaItemUseHook != nil {
		instruction = luaItemUseHook(item.configID, item.amount)
	}

	// ② INI 回退：箱类开包（描述对照优先 → drops_mvp 权重包 → 兜底阶梯）
	// / 使用技能回血阶梯。
	var grants []grantItem
	healHP, healMP := int32(0), int32(0)
	if instruction != nil {
		healHP, healMP = instruction.healHP, instruction.healMP
		if instruction.openDropID != "" {
			grants = currentDropPacks().roll(instruction.openDropID, instruction.rolls, itemUseRandom())
			if len(grants) == 0 {
				log.Printf("item use %s: drop pack %s empty/missing", item.configID, instruction.openDropID)
				return
			}
		} else if len(instruction.grants) > 0 {
			grants = instruction.grants
		}
	} else {
		entry, resolved := currentItemUseCatalog().resolve(item.configID, item.itemType)
		if !resolved {
			log.Printf("%s: use item %s (view=%d pos=%d) use-chain not implemented", h.remote, item.configID, view, item.slot)
			return
		}
		switch entry.kind {
		case itemUseBox:
			// 描述重建对照优先（box_contents.json 固定全发）；未命中回落
			// DropPacks 权重包（authored 真实包 → FALLBACK_LADDER 兜底）。
			if described := currentBoxContents().byBox(item.configID); described != nil {
				grants = described
				log.Printf("item use %s: box_contents hit (%d entries)", item.configID, len(described))
			} else {
				packs := currentDropPacks()
				random := itemUseRandom()
				for _, dropID := range entry.dropIDs {
					grants = append(grants, packs.roll(dropID, 1, random)...)
				}
				if len(grants) == 0 {
					log.Printf("item use %s: drop packs %v empty/missing", item.configID, entry.dropIDs)
					return
				}
			}
		case itemUseSkill:
			healHP = fallbackHealHp(entry.skillLevel)
			healMP = fallbackHealMp(entry.skillLevel)
		}
	}

	// 仅剩未实施的 temp_stat：不消耗防白吃（其余空效果维持 not implemented 口径）。
	tempStat := instruction != nil && instruction.tempStat
	if tempStat && healHP <= 0 && healMP <= 0 && grants == nil {
		log.Printf("item use %s: temp_stat 未接（面板限时属性二期），跳过消耗", item.configID)
		return
	}
	if healHP <= 0 && healMP <= 0 && grants == nil && !tempStat {
		log.Printf("item use %s: instructions empty", item.configID)
		return
	}
	h.player.mu.Lock()
	h.player.lastItemUseAt = now
	h.player.mu.Unlock()

	// ③ 效果落账：给物品（容量感知，失败即断；部分发放后仍走消耗，防重复刷取）。
	granted := make([]string, 0, len(grants))
	for _, grant := range grants {
		entry, ok := h.store.lookupGrantable(h.ctx, grant.configID)
		if !ok {
			log.Printf("item use %s: grant %s unknown config", item.configID, grant.configID)
			break
		}
		if h.grantBagItem(entry, grant.amount) == 0 {
			log.Printf("item use %s: grant %s failed (bag full)", item.configID, grant.configID)
			break
		}
		granted = append(granted, grant.configID)
	}

	// ④ 回血/回蓝（死亡/满血在结算内闸掉）。
	if vitalChanged := h.player.restoreItemVitals(healHP, healMP); vitalChanged {
		if frame, err := h.player.vitalUpdate(); err == nil {
			if err := h.link.WriteFrame(frame); err != nil {
				log.Printf("%s: item heal vital frame: %v", h.remote, err)
			}
		} else {
			log.Printf("%s: item heal vital encode: %v", h.remote, err)
		}
	}

	// ⑤ 消耗 1 个（脚本 consume=false 跳过）+ 背包帧对。
	consume := instruction == nil || instruction.consume
	if consume && !h.consumeBagItemCount(item, 1) {
		log.Printf("item use %s: consume failed (row missing/changed)", item.configID)
	}
	log.Printf("item use %s consumed=%t healHp=%d healMp=%d granted=%v tempStat=%t",
		item.configID, consume, healHP, healMP, granted, tempStat)
}

// globalItemRandom 使用链随机源（roll 权重/数量；Java 取会话 combatRandom，
// Go 进程级源等价——单机服口径）。
var globalItemRandom = rand.New(rand.NewSource(time.Now().UnixNano()))

// itemUseRandom 会话随机源。
func itemUseRandom() *rand.Rand {
	return globalItemRandom
}

// restoreItemVitals 药品回血/回蓝（死亡/满值闸掉；ApplyHeal 同语义清瘀血）。
func (p *playerActor) restoreItemVitals(hp, mp int32) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	changed := p.vitals.ApplyHeal(hp)
	if mp > 0 && p.vitals.MP < p.vitals.MaxMP {
		p.vitals.MP += mp
		if p.vitals.MP > p.vitals.MaxMP {
			p.vitals.MP = p.vitals.MaxMP
		}
		changed = true
	}
	return changed
}
