package main

// 背包四件套（msg30 移动/穿脱、34 丢弃、36 整理；31 的装备穿戴分支也在此），
// 移植 Java LoginFlow.handleMoveItem/unequipFromDoll/wearFromBag/moveBagRow/
// handleDeleteItem（:4559-4821）。内存权威模型在 playerActor（bag/equip/
// bagpackRows），每个操作先落 DB（inventory_mutations.go 写穿，失败拒绝）
// 再改内存并发帧。帧序照 Java：remove(源格) → ViewAdd(新行) → remove(镜像页)。

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

const (
	clientCustomMoveItem   int32 = 30 // CLIENT_CUSTOMMSG_MOVEITEM
	clientCustomUseItem    int32 = 31 // CLIENT_CUSTOMMSG_USEITEM
	clientCustomDeleteItem int32 = 34 // CLIENT_CUSTOMMSG_DELETEITEM
	clientCustomArrangeBag int32 = 36 // CLIENT_CUSTOMMSG_ARANGEITEM
)

// EquipType（小写）→ 纸娃娃候选槽位（Java AppearanceSync.EQUIP_SLOTS 同表；
// 双槽类型给两个，穿戴选首个空槽，全占用替换第一槽）。
var equipSlotsByType = map[string][]int32{
	"hat": {1}, "cloth": {3}, "pants": {4}, "wrist": {6}, "leg": {7}, "shoes": {8},
	"earring": {11, 12}, "necklace": {13}, "fingerring": {14, 15},
	"suit": {16}, "newsuit": {16},
	"fashioncoat": {17}, "newfashioncoat": {17}, "fashionhat": {18}, "fashionshoes": {19},
	"weapon": {22}, "shotweapon": {22}, "innerweapon": {22},
}

// mirrorDollSlot 时装/身份装槽（16-19）→ 新面板镜像槽（40-43）；其余原样。
func mirrorDollSlot(slot int32) int32 {
	if slot >= 16 && slot <= 19 {
		return 40 + slot - 16
	}
	return slot
}

// equipSlotFor 类型对应穿戴槽：双槽类型优先空槽，全占用替换第一槽；
// 类型未收录返回 0（按不可穿戴拒绝）。
func equipSlotFor(equipType string, equips []equipRow) int32 {
	slots := equipSlotsByType[strings.ToLower(strings.TrimSpace(equipType))]
	if len(slots) == 0 {
		return 0
	}
	for _, slot := range slots {
		taken := false
		for _, worn := range equips {
			if worn.slot == slot {
				taken = true
				break
			}
		}
		if !taken {
			return slot
		}
	}
	return slots[0]
}

// bagCategoryOf 线缆视图 → game_items 分类页（1 道具/2 装备/3 材料/4 任务；
// 扩展页 3/122/124/126 归一；非背包页返回 0）。
func bagCategoryOf(view int32) int32 {
	switch view {
	case int32(viewToolPage): // 2
		return 1
	case int32(viewEquipBag): // 121
		return 2
	case int32(viewMaterialPage): // 123
		return 3
	case int32(viewTaskPage): // 125
		return 4
	case 3, 122, 124, 126:
		for page, ext := range pageExtView {
			if int32(ext) == view {
				return page
			}
		}
	}
	return 0
}

// serverDeleteView 编码 SERVER_DELETE_VIEW（0x16）：u16 视图（3B，Java
// SceneMessages.deleteView 同构；View40/43 整视图重开的前置删除帧）。
func serverDeleteView(viewID uint16) []byte {
	msg := make([]byte, 3)
	msg[0] = 0x16
	binary.LittleEndian.PutUint16(msg[1:], viewID)
	return msg
}

// customIntArgs 提取 values[1..] 的整型参数（exactInt32 宽容解析）。
func customIntArgs(custom clientCustomMessage) []int32 {
	args := make([]int32, 0, len(custom.Values)-1)
	for _, value := range custom.Values[1:] {
		if number, ok := value.exactInt32(); ok {
			args = append(args, number)
		}
	}
	return args
}

// ---- playerActor 会话物品模型（锁纪律：一律经 p.mu） ----

// restoreSessionInventory 登录时装载会话模型（读侧已加载的行直接喂入；
// 加载失败=空模型恒初始化）。
func (p *playerActor) restoreSessionInventory(sex byte, equips []equipRow, items []bagRow, packs []bagpackRow) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visualSex = sex
	p.equip = append([]equipRow(nil), equips...)
	p.bag = append([]bagRow(nil), items...)
	p.bagpackRows = append([]bagpackRow(nil), packs...)
	p.qingGongExtra = make(map[string]struct{})
}

func (p *playerActor) sessionBag() []bagRow {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]bagRow(nil), p.bag...)
}

func (p *playerActor) sessionEquips() []equipRow {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]equipRow(nil), p.equip...)
}

// bagItemAt (主分类视图,槽位) → 行副本与下标（bagViewOf(viewID)==view 匹配）。
func (p *playerActor) bagItemAt(view uint16, slot int32) (bagRow, int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for index := range p.bag {
		if p.bag[index].slot == slot && bagViewOf(p.bag[index].viewID) == view {
			return p.bag[index], index, true
		}
	}
	return bagRow{}, -1, false
}

func (p *playerActor) equipAtSlot(slot int32) (equipRow, int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for index := range p.equip {
		if p.equip[index].slot == slot {
			return p.equip[index], index, true
		}
	}
	return equipRow{}, -1, false
}

// freeBagSlot 分类页第一个空格（1..90；无空位返回 0）。
func (p *playerActor) freeBagSlot(category int32) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	used := make([]bool, int(bagSlotMax)+1)
	for _, item := range p.bag {
		if item.viewID == category && item.slot > 0 && item.slot <= int32(bagSlotMax) {
			used[item.slot] = true
		}
	}
	for slot := int32(1); slot <= int32(bagSlotMax); slot++ {
		if !used[slot] {
			return slot
		}
	}
	return 0
}

// ---- 帧组助手（Java pushBagItemFrames/pushWearFrames 同序） ----

// pushBagItemFrames 入包帧组：0x19(view,slot) 预清 → 0x18 行 → 0x19(镜像页,slot)。
func pushBagItemFrames(link sceneMessageConnection, roleID role.RoleID, item bagRow) error {
	view := bagViewOf(item.viewID)
	if err := link.WriteFrame(serverViewRemove(view, uint16(item.slot))); err != nil {
		return err
	}
	frame, err := bagRowViewAdd(roleID, item)
	if err != nil {
		return fmt.Errorf("encode bag row %s: %w", item.configID, err)
	}
	if err := link.WriteFrame(frame); err != nil {
		return err
	}
	if mirror := bagMirrorOf(view); mirror != view {
		return link.WriteFrame(serverViewRemove(mirror, uint16(item.slot)))
	}
	return nil
}

// pushWearFrames 穿戴帧组：纸娃娃清格 → 行 → 时装镜像槽成对 → 外观
// 0x10（appearanceFrame：ShowEquipType + 三模式裁决模型 + 武器三属性
// 显式清除——Java pushWearFrames 同序；词缀重算 applyEquipBonuses 属
// 词缀专项未移植，跳过）。
func pushWearFrames(link sceneMessageConnection, roleID role.RoleID, sex byte, showEquipType int, equip equipRow, equips []equipRow) error {
	if err := link.WriteFrame(serverViewRemove(viewDoll, uint16(equip.slot))); err != nil {
		return err
	}
	frame, err := equipmentViewAdd(roleID, equip)
	if err != nil {
		return fmt.Errorf("encode equip row %s: %w", equip.configID, err)
	}
	if err := link.WriteFrame(frame); err != nil {
		return err
	}
	if mirror := mirrorDollSlot(equip.slot); mirror != equip.slot {
		if err := link.WriteFrame(serverViewRemove(viewDoll, uint16(mirror))); err != nil {
			return err
		}
		mirrored := equip
		mirrored.slot = mirror
		mirrorFrame, err := equipmentViewAdd(roleID, mirrored)
		if err != nil {
			return fmt.Errorf("encode mirrored equip row %s: %w", equip.configID, err)
		}
		if err := link.WriteFrame(mirrorFrame); err != nil {
			return err
		}
	}
	appearance, err := appearanceFrame(playerObjectID, playerOwnerID, showEquipType, sex, equips)
	if err != nil {
		return fmt.Errorf("encode appearance frame: %w", err)
	}
	return link.WriteFrame(appearance)
}

// itemSession 是四件套 handler 的会话上下文（customConfig 惯例的参数包）。
type itemSession struct {
	link     sceneMessageConnection
	player   *playerActor
	store    *inventoryStore  // 可为 nil（纯内存降级）
	skins    *skinDBStore     // 风物志衣橱存档（card 收录链用；可为 nil）
	currency *currencyDBStore // 碎银存档（商店购销后落库；可为 nil）
	ctx      context.Context
	roleID   role.RoleID
	remote   string
	now      func() time.Time
}

// persistCurrency 商店购销后的碎银落库（fork "persist currency after
// shop buy/sell" 同口径；失败只记日志不影响内存权威）。
func (h *itemSession) persistCurrency(reason string) {
	if h.currency == nil || h.roleID == 0 {
		return
	}
	if err := h.currency.Save(h.roleID, h.player.currencySnapshotOf()); err != nil {
		log.Printf("%s: persist currency after %s: %v", h.remote, reason, err)
	}
}

// stackTargetFor 找同 configID 的可叠堆背包行（同页同类；槽位>0）。
func (p *playerActor) stackTargetFor(configID string) (bagRow, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, row := range p.bag {
		if row.configID == configID && row.slot > 0 {
			return row, true
		}
	}
	return bagRow{}, false
}

// countBagConfig 按 configID 全包求和（exchange 兑换计价的数量校验；
// fork (*playerActor).countBagConfig 同义）。
func (p *playerActor) countBagConfig(configID string) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	total := int32(0)
	for _, row := range p.bag {
		if row.configID == configID {
			total += row.amount
		}
	}
	return total
}

// consumeBagConfig 跨行按 configID 扣减（exchange 按物扣费）：逐行走
// consumeBagItemCount（DB 写穿+帧同步），行快照遍历防删行迭代错位。
// 调用方须先以 countBagConfig 校验总量（DB 并发竞态导致的半途失败只
// 记日志，与 fork "material lost after check" TOCTOU 同口径）。
func (h *itemSession) consumeBagConfig(configID string, count int32) bool {
	if h.player == nil {
		return false
	}
	h.player.mu.Lock()
	rows := append([]bagRow(nil), h.player.bag...)
	h.player.mu.Unlock()
	remaining := count
	for _, row := range rows {
		if remaining <= 0 {
			break
		}
		if row.configID != configID || row.amount <= 0 {
			continue
		}
		take := row.amount
		if take > remaining {
			take = remaining
		}
		if !h.consumeBagItemCount(row, take) {
			log.Printf("%s: exchange consume %s row slot=%d db failed", h.remote, configID, row.slot)
			return false
		}
		remaining -= take
	}
	return remaining <= 0
}

// consumeBagItemCount 消耗背包道具 count 个并同步客户端行
// （剩余>0 重推行，=0 删行+镜像清位；Java consumeBagItemCount 同构，
// 内存行 Amount 语义与 DB amount 一致）。
func (h *itemSession) consumeBagItemCount(item bagRow, count int32) bool {
	remaining := item.amount - count
	if h.store != nil {
		dbRemaining, err := h.store.consumeBagItem(h.ctx, h.roleID, item.viewID, item.slot, count)
		if err != nil {
			log.Printf("%s: consume %s x%d db: %v", h.remote, item.configID, count, err)
			return false
		}
		if dbRemaining < 0 {
			return false
		}
		remaining = dbRemaining
	}
	h.player.mu.Lock()
	for index := range h.player.bag {
		if h.player.bag[index].slot == item.slot && h.player.bag[index].viewID == item.viewID {
			if remaining > 0 {
				h.player.bag[index].amount = remaining
			} else {
				h.player.bag = append(h.player.bag[:index], h.player.bag[index+1:]...)
			}
			break
		}
	}
	h.player.mu.Unlock()
	view := bagViewOf(item.viewID)
	if remaining > 0 {
		updated := item
		updated.amount = remaining
		if err := pushBagItemFrames(h.link, h.roleID, updated); err != nil {
			log.Printf("%s: push consumed row: %v", h.remote, err)
		}
	} else {
		if err := h.link.WriteFrame(serverViewRemove(view, uint16(item.slot))); err != nil {
			log.Printf("%s: remove consumed row: %v", h.remote, err)
		}
		if mirror := bagMirrorOf(view); mirror != view {
			if err := h.link.WriteFrame(serverViewRemove(mirror, uint16(item.slot))); err != nil {
				log.Printf("%s: mirror clear consumed row: %v", h.remote, err)
			}
		}
	}
	return true
}

// addToBagItemCount 行内加数并同步客户端行（consumeBagItemCount 的
// 逆操作；商店购买叠堆用）。返回 false=DB 失败拒绝。
func (h *itemSession) addToBagItemCount(item bagRow, count int32) bool {
	total := item.amount + count
	if h.store != nil {
		dbTotal, err := h.store.increaseBagItem(h.ctx, h.roleID, item.viewID, item.slot, count)
		if err != nil {
			log.Printf("%s: stack %s x%d db: %v", h.remote, item.configID, count, err)
			return false
		}
		if dbTotal < 0 {
			return false
		}
		total = dbTotal
	}
	h.player.mu.Lock()
	for index := range h.player.bag {
		if h.player.bag[index].slot == item.slot && h.player.bag[index].viewID == item.viewID {
			h.player.bag[index].amount = total
			break
		}
	}
	h.player.mu.Unlock()
	updated := item
	updated.amount = total
	if err := pushBagItemFrames(h.link, h.roleID, updated); err != nil {
		log.Printf("%s: push stacked row: %v", h.remote, err)
	}
	return true
}

// grantBagItem 发放物品入包（容量感知：1..90 无空位即断）；返回槽位（0=失败）。
// （Java grantBagItem 同构：开箱/脚本 grants 的统一落包路径。）
func (h *itemSession) grantBagItem(entry grantableItem, amount int32) int32 {
	category := entry.viewID
	if category < 1 || category > 4 {
		category = 1
	}
	slot := h.player.freeBagSlot(category)
	if slot <= 0 {
		return 0
	}
	inserted := bagRow{
		slot: slot, configID: entry.configID, itemType: entry.itemType, amount: amount,
		viewID: category, name: entry.name, equipType: entry.equipType,
	}
	if h.store != nil {
		row, err := h.store.insertBagItem(h.ctx, h.roleID, inserted)
		if err != nil {
			log.Printf("%s: insert bag item %s: %v", h.remote, entry.configID, err)
			return 0
		}
		inserted = row
	}
	h.player.mu.Lock()
	h.player.bag = append(h.player.bag, inserted)
	h.player.mu.Unlock()
	if err := pushBagItemFrames(h.link, h.roleID, inserted); err != nil {
		log.Printf("%s: push granted row: %v", h.remote, err)
	}
	return inserted.slot
}

// ---- msg30 移动物品/穿脱装备 ----

// handleMoveItemCustom（[30,fromView,fromSlot,toView,toSlot]，Java handleMoveItem）：
// fromView==1 脱下；toView==1 穿上；背包域之外的视图目标拒绝（bagpack
// addtoolbox 未接，与 Java 同口径）；其余背包页间/页内移动。
func handleMoveItemCustom(h *itemSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomMoveItem {
		return false, nil
	}
	args := customIntArgs(custom)
	if len(args) < 4 {
		log.Printf("%s: move 30 malformed args=%d (need fromView,fromSlot,toView,toSlot)", h.remote, len(args))
		return true, nil
	}
	fromView, fromSlot, toView, toSlot := args[0], args[1], args[2], args[3]
	switch {
	case fromView == int32(viewDoll):
		unequipFromDoll(h, fromSlot, toView, toSlot)
	case toView == int32(viewDoll):
		wearFromBag(h, fromView, fromSlot, toSlot)
	case bagCategoryOf(toView) == 0 || bagCategoryOf(fromView) == 0:
		log.Printf("%s: move 30 ext view (%d,%d)->(%d,%d) rejected (bagpack addtoolbox not implemented)",
			h.remote, fromView, fromSlot, toView, toSlot)
	default:
		moveBagRowBetweenViews(h, fromView, fromSlot, toView, toSlot)
	}
	return true, nil
}

// unequipFromDoll 脱下：装备栏 → 背包页（toView 非背包页归装备页 121；
// toSlot<=0 自动选该页首空格）。
func unequipFromDoll(h *itemSession, equipSlot, toView, toSlot int32) {
	item, index, ok := h.player.equipAtSlot(equipSlot)
	if !ok {
		log.Printf("%s: move 30 equip[%d] empty", h.remote, equipSlot)
		return
	}
	category := bagCategoryOf(toView)
	view := toView
	if category == 0 {
		category = 2
		view = int32(viewEquipBag)
	}
	slot := toSlot
	if slot <= 0 {
		slot = h.player.freeBagSlot(category)
	}
	if slot <= 0 {
		log.Printf("%s: move 30 unequip reject: bag page %d full", h.remote, view)
		return
	}
	if h.store != nil && !h.store.moveEquipToBag(h.ctx, h.roleID, equipSlot, slot, category) {
		log.Printf("%s: move 30 equip[%d]->bag[%d] persist failed", h.remote, equipSlot, slot)
		return
	}
	h.player.mu.Lock()
	h.player.equip = append(h.player.equip[:index], h.player.equip[index+1:]...)
	landed := bagRow{
		slot: slot, configID: item.configID, itemType: item.itemType, amount: 1,
		viewID: category, equipType: item.equipType,
		hardiness: item.hardiness, maxHardiness: item.maxHardiness,
		seq: item.seq, uid: item.uid,
	}
	h.player.bag = append(h.player.bag, landed)
	equipsNow := append([]equipRow(nil), h.player.equip...)
	sex := h.player.visualSex
	h.player.mu.Unlock()

	// 帧组：纸娃娃清格（时装含镜像 40-43）→ 入包成对帧 → 武器外观刷新。
	if err := h.link.WriteFrame(serverViewRemove(viewDoll, uint16(equipSlot))); err != nil {
		log.Printf("%s: unequip clear slot: %v", h.remote, err)
	}
	if mirror := mirrorDollSlot(equipSlot); mirror != equipSlot {
		if err := h.link.WriteFrame(serverViewRemove(viewDoll, uint16(mirror))); err != nil {
			log.Printf("%s: unequip clear mirror: %v", h.remote, err)
		}
	}
	if err := pushBagItemFrames(h.link, h.roleID, landed); err != nil {
		log.Printf("%s: unequip push bag row: %v", h.remote, err)
	}
	// 外观整刷（Java pushAppearanceRefresh 同口径：脱下后按有效显示
	// 类型重推最终模型 + 武器三属性显式清除）。
	if appearance, err := appearanceFrame(playerObjectID, playerOwnerID, effectiveShowEquipType(h.player), sex, equipsNow); err != nil {
		log.Printf("%s: unequip appearance encode: %v", h.remote, err)
	} else if err := h.link.WriteFrame(appearance); err != nil {
		log.Printf("%s: unequip appearance frame: %v", h.remote, err)
	}
	log.Printf("%s: move 30 equip[%d]->bag[%d:%d] config=%s", h.remote, equipSlot, view, slot, item.configID)
}

// wearFromBag 共享穿装核心（msg30 穿上 / msg31 右键穿）：校验可穿
// （EquipType→槽位，toSlot<=0 自动槽=双槽类型首个空槽）→ 目标槽被占时
// 旧装备让位到装备页首空格 → 落库 → 源格清帧 + 旧装备入包帧 + 穿戴帧组。
func wearFromBag(h *itemSession, fromView, fromSlot, toSlot int32) bool {
	category := bagCategoryOf(fromView)
	if category == 0 {
		log.Printf("%s: wear reject: source view %d not a bag page", h.remote, fromView)
		return false
	}
	item, index, ok := h.player.bagItemAt(bagViewOf(category), fromSlot)
	if !ok {
		log.Printf("%s: wear reject: bag[%d:%d] empty", h.remote, fromView, fromSlot)
		return false
	}
	equipType := wearableEquipType(item)
	equips := h.player.sessionEquips()
	slot := toSlot
	if slot <= 0 {
		slot = equipSlotFor(equipType, equips)
	}
	if equipType == "" || slot <= 0 {
		log.Printf("%s: wear requires one wearable equipment item (config=%s type=%q)", h.remote, item.configID, equipType)
		return false
	}
	var old equipRow
	var landedOld bagRow
	hasOld := false
	if worn, _, taken := h.player.equipAtSlot(slot); taken {
		old = worn
		hasOld = true
		oldSlot := h.player.freeBagSlot(2)
		if oldSlot <= 0 {
			log.Printf("%s: wear reject: bag full, cannot swap out %s", h.remote, old.configID)
			return false
		}
		if h.store != nil && !h.store.swapEquipSlot(h.ctx, h.roleID, slot, oldSlot) {
			log.Printf("%s: wear swap equip[%d]->bag[%d] persist failed", h.remote, slot, oldSlot)
			return false
		}
		landedOld = bagRow{
			slot: oldSlot, configID: old.configID, itemType: old.itemType, amount: 1,
			viewID: 2, equipType: old.equipType,
			hardiness: old.hardiness, maxHardiness: old.maxHardiness,
			seq: old.seq, uid: old.uid,
		}
	}
	if h.store != nil && !h.store.moveBagToEquip(h.ctx, h.roleID, fromSlot, slot, category) {
		log.Printf("%s: wear bag[%d]->equip[%d] persist failed", h.remote, fromSlot, slot)
		return false
	}
	hardiness, maxHardiness := item.hardiness, item.maxHardiness
	if hardiness <= 0 {
		hardiness = 100
	}
	if maxHardiness <= 0 {
		maxHardiness = 100
	}
	worn := equipRow{
		configID: item.configID, itemType: item.itemType, equipType: equipType, slot: slot,
		hardiness: hardiness, maxHardiness: maxHardiness, seq: item.seq, uid: item.uid,
	}
	h.player.mu.Lock()
	h.player.bag = append(h.player.bag[:index], h.player.bag[index+1:]...)
	if hasOld {
		for i := range h.player.equip {
			if h.player.equip[i].slot == slot {
				h.player.equip = append(h.player.equip[:i], h.player.equip[i+1:]...)
				break
			}
		}
		h.player.bag = append(h.player.bag, landedOld)
	}
	h.player.equip = append(h.player.equip, worn)
	equipsNow := append([]equipRow(nil), h.player.equip...)
	sex := h.player.visualSex
	h.player.mu.Unlock()

	if err := h.link.WriteFrame(serverViewRemove(uint16(fromView), uint16(fromSlot))); err != nil {
		log.Printf("%s: wear clear source: %v", h.remote, err)
	}
	if fromMirror := bagMirrorOf(uint16(fromView)); fromMirror != uint16(fromView) {
		if err := h.link.WriteFrame(serverViewRemove(fromMirror, uint16(fromSlot))); err != nil {
			log.Printf("%s: wear clear source mirror: %v", h.remote, err)
		}
	}
	if hasOld {
		if err := pushBagItemFrames(h.link, h.roleID, landedOld); err != nil {
			log.Printf("%s: wear push swapped row: %v", h.remote, err)
		}
	}
	if err := pushWearFrames(h.link, h.roleID, sex, effectiveShowEquipType(h.player), worn, equipsNow); err != nil {
		log.Printf("%s: wear push frames: %v", h.remote, err)
	}
	swapNote := ""
	if hasOld {
		swapNote = " (swapped out " + old.configID + ")"
	}
	log.Printf("%s: wear config=%s -> slot=%d%s", h.remote, item.configID, slot, swapNote)
	return true
}

// wearableEquipType 解析可穿类型：itemStatic 表值 > 行值（bagRowViewAdd 同序）。
func wearableEquipType(item bagRow) string {
	if stat := equipmentModels.itemStatic(item.configID); stat != nil && stat.equipType != "" {
		return stat.equipType
	}
	return item.equipType
}

// moveBagRowBetweenViews 背包页间/页内移动（toSlot<=0 自动选目标页首空格）。
func moveBagRowBetweenViews(h *itemSession, fromView, fromSlot, toView, toSlot int32) {
	fromCategory := bagCategoryOf(fromView)
	toCategory := bagCategoryOf(toView)
	item, index, ok := h.player.bagItemAt(bagViewOf(fromCategory), fromSlot)
	if !ok {
		log.Printf("%s: move 30 bag[%d:%d] empty", h.remote, fromView, fromSlot)
		return
	}
	slot := toSlot
	if slot <= 0 {
		slot = h.player.freeBagSlot(toCategory)
	}
	if slot <= 0 {
		log.Printf("%s: move 30 reject: bag page %d full", h.remote, toView)
		return
	}
	if h.store != nil && !h.store.moveBagSlot(h.ctx, h.roleID, fromSlot, fromCategory, slot, toCategory) {
		log.Printf("%s: move 30 bag[%d]->bag[%d] persist failed", h.remote, fromSlot, slot)
		return
	}
	h.player.mu.Lock()
	h.player.bag[index].slot = slot
	h.player.bag[index].viewID = toCategory
	moved := h.player.bag[index]
	h.player.mu.Unlock()

	if err := h.link.WriteFrame(serverViewRemove(uint16(fromView), uint16(fromSlot))); err != nil {
		log.Printf("%s: move clear source: %v", h.remote, err)
	}
	if fromMirror := bagMirrorOf(uint16(fromView)); fromMirror != uint16(fromView) {
		if err := h.link.WriteFrame(serverViewRemove(fromMirror, uint16(fromSlot))); err != nil {
			log.Printf("%s: move clear source mirror: %v", h.remote, err)
		}
	}
	if err := pushBagItemFrames(h.link, h.roleID, moved); err != nil {
		log.Printf("%s: move push row: %v", h.remote, err)
	}
	log.Printf("%s: move 30 bag[%d:%d]%d->bag[%d:%d] config=%s", h.remote, fromView, fromSlot, fromCategory, toView, slot, item.configID)
}

// ---- msg34 丢弃 ----

// handleDeleteItemCustom（[34,view,slot,count]，Java handleDeleteItem）：
// 非背包页/空格/超量拒绝；回包与消耗链同源（剩余>0 remove+add，=0 双 remove）。
func handleDeleteItemCustom(h *itemSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomDeleteItem {
		return false, nil
	}
	args := customIntArgs(custom)
	if len(args) < 3 {
		log.Printf("%s: delete 34 malformed args=%d (need view,pos,amount)", h.remote, len(args))
		return true, nil
	}
	view, slot, amount := args[0], args[1], args[2]
	category := bagCategoryOf(view)
	if category == 0 {
		log.Printf("%s: delete 34 reject: view=%d not a bag page", h.remote, view)
		return true, nil
	}
	view = int32(bagViewOf(category)) // 扩展页归一主分类页
	if amount < 1 {
		log.Printf("%s: delete 34 reject: amount=%d", h.remote, amount)
		return true, nil
	}
	item, _, ok := h.player.bagItemAt(uint16(view), slot)
	if !ok {
		log.Printf("%s: delete 34 reject: bag[%d:%d] empty", h.remote, view, slot)
		return true, nil
	}
	if amount > item.amount {
		log.Printf("%s: delete 34 reject: %s amount %d > stack %d", h.remote, item.configID, amount, item.amount)
		return true, nil
	}
	if !h.consumeBagItemCount(item, amount) {
		log.Printf("%s: delete 34 %s: consume failed (row missing/changed)", h.remote, item.configID)
		return true, nil
	}
	log.Printf("%s: delete 34 bag[%d:%d] config=%s amount=%d discarded", h.remote, view, slot, item.configID, amount)
	return true, nil
}

// ---- msg36 整理 ----

// handleArrangeItemCustom（[36,view]，fork 只读 values[1]）：同类聚合+前移
// 压缩——同 ConfigID 聚为相邻组（组间按首现顺序），整体前移填满低位空格；
// 仅对变更格发出 remove(旧)+成对 add(新) 帧。fork 同款不落盘。
func handleArrangeItemCustom(h *itemSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomArrangeBag {
		return false, nil
	}
	args := customIntArgs(custom)
	if len(args) < 1 || args[0] <= 0 {
		log.Printf("%s: arrange 36 malformed args=%d", h.remote, len(args))
		return true, nil
	}
	category := bagCategoryOf(args[0])
	if category == 0 {
		log.Printf("%s: arrange 36 reject: view=%d not a bag page", h.remote, args[0])
		return true, nil
	}
	view := bagViewOf(category)

	type arrangeMove struct {
		from, to int32
		row      bagRow
	}
	h.player.mu.Lock()
	// 组序 = 各 ConfigID 首现（槽位升序）顺序；组内保持原相对顺序。
	rows := make([]bagRow, 0, len(h.player.bag))
	for _, item := range h.player.bag {
		if item.viewID == category {
			rows = append(rows, item)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].slot < rows[j].slot })
	firstSlot := make(map[string]int32)
	groupOrder := make([]string, 0, len(rows))
	for _, item := range rows {
		if _, exists := firstSlot[item.configID]; !exists {
			firstSlot[item.configID] = item.slot
			groupOrder = append(groupOrder, item.configID)
		}
	}
	groupIndex := make(map[string]int, len(groupOrder))
	for i, id := range groupOrder {
		groupIndex[id] = i
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return groupIndex[rows[i].configID] < groupIndex[rows[j].configID]
	})
	// 前移压缩：已定序的行依次取最低空格（不高于原槽位则前移，否则不动）。
	used := make([]bool, int(bagSlotMax)+1)
	moves := make([]arrangeMove, 0, len(rows))
	next := int32(1)
	for index := range rows {
		for next <= int32(bagSlotMax) && used[next] {
			next++
		}
		target := rows[index].slot
		if next < target {
			target = next
		}
		if target > 0 && target <= int32(bagSlotMax) {
			used[target] = true
		}
		if target != rows[index].slot {
			moves = append(moves, arrangeMove{from: rows[index].slot, to: target, row: rows[index]})
			rows[index].slot = target
		}
	}
	if len(moves) == 0 {
		h.player.mu.Unlock()
		log.Printf("%s: arrange 36 view=%d no changes", h.remote, view)
		return true, nil
	}
	// 回写内存模型：非本页行 + 重排后的本页行。
	newBag := make([]bagRow, 0, len(h.player.bag))
	for _, item := range h.player.bag {
		if item.viewID != category {
			newBag = append(newBag, item)
		}
	}
	newBag = append(newBag, rows...)
	h.player.bag = newBag
	h.player.mu.Unlock()

	for _, move := range moves {
		if err := h.link.WriteFrame(serverViewRemove(view, uint16(move.from))); err != nil {
			log.Printf("%s: arrange clear old slot: %v", h.remote, err)
		}
		moved := move.row
		moved.slot = move.to
		if err := pushBagItemFrames(h.link, h.roleID, moved); err != nil {
			log.Printf("%s: arrange push row: %v", h.remote, err)
		}
	}
	log.Printf("%s: arrange 36 view=%d moved=%d cells (grouped by config, compacted)", h.remote, view, len(moves))
	return true, nil
}
