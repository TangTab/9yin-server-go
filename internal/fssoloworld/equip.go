package fssoloworld

// C2S msg30 移动物品/穿脱。方言双源对齐：
//
// 官服真机（2026-11-23 初始账号登陆-3 + 更改快捷键三会话，docs/official/
// 初始号无装备身体模型定版.md）：脱=0x19(view1,槽)+入包行+op10 裸身模型
// 单值帧+PhyDef；穿=0x19(源页,槽)+op10 件模型。
//
// 2433 参考端（cmd/protocol-probe/item_wear.go，fork 客户端实证）：
// 穿戴帧组=0x19(doll 槽)+0x18 纸娃娃行+appearance 0x10（ShowEquipType+
// 已穿槽位模型+武器三属性，空槽跳过、空手 ActionSet 回 1h）；脱下后同推
// appearance 整刷。两源合流：穿发 doll 行+appearance（参考端），脱额外推
// 裸身模型单值帧（官服——保证视觉回退）。PhyDef 单推（0 基线+布衣 +6/件）。
// 占槽换装：原槽件退回源背包格。

import (
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// equipViewWire 装备（纸娃娃）wire 视图号。
const equipViewWire int32 = 1

// equipSlotsByType EquipType → 穿戴槽（2433 参考端 equipSlotsByType 同表：
// 双槽类型优先空槽）。
var equipSlotsByType = map[string][]int{
	"hat": {1}, "cloth": {3}, "pants": {4}, "wrist": {6}, "leg": {7}, "shoes": {8},
	"earring": {11, 12}, "necklace": {13}, "fingerring": {14, 15},
	"suit": {16}, "newsuit": {16},
	"fashioncoat": {17}, "newfashioncoat": {17}, "fashionhat": {18}, "fashionshoes": {19},
	"weapon": {22}, "shotweapon": {22}, "innerweapon": {22},
	"treasure":    {25, 26, 27, 28, 29},
	"newtreasure": {30, 31, 32, 33, 34},
}

// bagViewIDOfWire wire 视图号 → 背包分类页（bagViewOf 逆映射；未知 0）。
func bagViewIDOfWire(wire int32) int {
	switch wire {
	case 2:
		return 1
	case 121:
		return 2
	case 123:
		return 3
	case 125:
		return 4
	}
	return 0
}

// equipTypeOfItem 行 EquipType（行值 > 目录 > ConfigID 前缀兜底）。
func (s *Server) equipTypeOfItem(itemType, slotFallback int, equipType, configID string) string {
	if equipType != "" {
		return equipType
	}
	if entry, ok := s.itemCatalogOf().Lookup(configID); ok && entry.EquipType != "" {
		return entry.EquipType
	}
	if under := strings.IndexByte(configID, '_'); under > 0 {
		return strings.ToLower(configID[:under])
	}
	return ""
}

// equipSlotForEquipType EquipType → 目标槽：显式槽须在类型槽表内；
// toSlot<=0 自动（双槽类型首个空槽，全占用替换第一槽）。未收录类型 0。
func equipSlotForEquipType(equipType string, toSlot int, equips []rolestate.EquipItem) int {
	slots, ok := equipSlotsByType[strings.ToLower(strings.TrimSpace(equipType))]
	if !ok {
		return 0
	}
	if toSlot > 0 {
		for _, allowed := range slots {
			if allowed == toSlot {
				return toSlot
			}
		}
		return 0 // 类型与所报槽不符：拒收
	}
	for _, slot := range slots {
		taken := false
		for _, worn := range equips {
			if worn.Slot == slot {
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

// handleMoveItem C2S 30 [30, src_view, src_slot, dst_view, dst_slot]。
func (s *Server) handleMoveItem(pl *Player, message entergen.CustomInbound) {
	if !s.requireRoleState(pl, clientCustomMoveItem) {
		return
	}
	if len(message.Values) < 5 {
		s.trace("equip move: 畸形 values=%d client=%d", len(message.Values), pl.Conn1)
		return
	}
	srcView, ok1 := message.Values[1].ExactInt32()
	srcSlot, ok2 := message.Values[2].ExactInt32()
	dstView, ok3 := message.Values[3].ExactInt32()
	dstSlot, ok4 := message.Values[4].ExactInt32()
	if !ok1 || !ok2 || !ok3 || !ok4 {
		s.trace("equip move: 非整型参数 client=%d", pl.Conn1)
		return
	}
	switch {
	case srcView == equipViewWire && dstView != equipViewWire:
		s.moveUnequip(pl, srcSlot, dstView, dstSlot)
	case srcView != equipViewWire && dstView == equipViewWire:
		s.moveEquip(pl, srcView, srcSlot, dstSlot)
	case srcView != equipViewWire && dstView != equipViewWire:
		s.moveInsideBag(pl, srcView, srcSlot, dstView, dstSlot)
	default:
		s.trace("equip move: 拒收 equip→equip %d.%d→%d.%d client=%d",
			srcView, srcSlot, dstView, dstSlot, pl.Conn1)
	}
}

// resolveBagSlot 解析目标背包格：0=自动（首个空格）；显式格须在 1..90 且空闲。
func (s *Server) resolveBagSlot(pl *Player, st *rolestate.RoleState, viewID, slot int, configID, what string) int {
	if slot == 0 {
		if free := st.FreeBagSlot(viewID); free != 0 {
			return free
		}
		s.trace("%s: 分类页 %d 满 configID=%s client=%d", what, viewID, configID, pl.Conn1)
		return 0
	}
	if slot < 1 || slot > 90 {
		s.trace("%s: 槽位越界 %d client=%d", what, slot, pl.Conn1)
		return 0
	}
	for _, item := range st.BagItems {
		if item.ViewID == viewID && item.Slot == slot {
			s.trace("%s: 目标格 %d.%d 非空 client=%d", what, viewID, slot, pl.Conn1)
			return 0
		}
	}
	return slot
}

// bagRowCanAbsorb 背包行能否再并入 add 件（堆叠上限 = 行值 > 物品目录
// MaxAmount；未标不设限）。并入时把目录上限补记到行。
func (s *Server) bagRowCanAbsorb(row *rolestate.BagItem, add int) bool {
	if row.MaxAmount <= 0 {
		if entry, ok := s.itemCatalogOf().Lookup(row.ConfigID); ok {
			row.MaxAmount = entry.MaxAmount
		}
	}
	return row.MaxAmount <= 0 || row.Amount+add <= row.MaxAmount
}

// stackOrAddBagItem 入包单件：同 configID 同页有未满行 → 叠堆（上限 =
// 行值/物品目录 MaxAmount；未标不设限），否则新行。返回行帧。
func (s *Server) stackOrAddBagItem(pl *Player, item rolestate.BagItem) ([][]byte, error) {
	st := &pl.roleState
	if item.Amount == 1 {
		limit := item.MaxAmount
		if limit == 0 {
			if entry, ok := s.itemCatalogOf().Lookup(item.ConfigID); ok {
				limit = entry.MaxAmount
			}
		}
		for i := range st.BagItems {
			if st.BagItems[i].ConfigID != item.ConfigID || st.BagItems[i].ViewID != item.ViewID {
				continue
			}
			capacity := st.BagItems[i].MaxAmount
			if capacity <= 0 {
				capacity = limit
			}
			if capacity > 0 && st.BagItems[i].Amount >= capacity {
				continue // 该行已满：不可堆叠物逐件成行
			}
			st.BagItems[i].Amount++
			return entergen.BagItemPushFrames(st.BagItems[i])
		}
		item.MaxAmount = limit
	}
	st.BagItems = append(st.BagItems, item)
	return entergen.BagItemPushFrames(item)
}

// pushEquipSlotModel op10 模型单值帧（官服方言：空槽显式回推裸身 000 族
// ——保证视觉回退；穿=件模型）。
func (s *Server) pushEquipSlotModel(pl *Player, slot int) {
	name := entergen.EquipSlotPropName(slot)
	if name == "" {
		return
	}
	value := entergen.EquipSlotModel(&pl.roleState, slot)
	frame, err := playerPropertyFrame(pl, entergen.NamedProperty(name, entergen.StringValue(value)))
	if err != nil {
		s.trace("equip move: 模型帧编码失败 slot=%d: %v", slot, err)
		return
	}
	s.sendToClient(pl, frame)
}

// pushEquipAppearance op10 外观整刷（2433 参考端 appearanceFrame 方言：
// ShowEquipType+已穿槽位模型+武器三属性恒写）。
func (s *Server) pushEquipAppearance(pl *Player) {
	frame, err := playerPropertyFrame(pl, entergen.EquipAppearanceProps(&pl.roleState)...)
	if err != nil {
		s.trace("equip move: appearance 帧编码失败: %v", err)
		return
	}
	s.sendToClient(pl, frame)
}

// pushEquipPhyDef op10 PhyDef 同步（装备合计）。
func (s *Server) pushEquipPhyDef(pl *Player) {
	frame, err := playerPropertyFrame(pl, entergen.NamedProperty("PhyDef", entergen.Int32Value(entergen.EquipPhyDef(&pl.roleState))))
	if err != nil {
		s.trace("equip move: PhyDef 帧编码失败: %v", err)
		return
	}
	s.sendToClient(pl, frame)
}

// itemTypeWithCatalog 行 ItemType（缺席时查目录兜底）。
func (s *Server) itemTypeWithCatalog(have int, configID string) int {
	if have != 0 {
		return have
	}
	if entry, ok := s.itemCatalogOf().Lookup(configID); ok {
		return entry.ItemType
	}
	return 0
}

// moveUnequip 脱装（equip → 背包）：0x19(view1,槽) + 入包行 + appearance
// 整刷 + 裸身模型单推（官服方言）+ PhyDef。
func (s *Server) moveUnequip(pl *Player, srcSlot, dstView, dstSlot int32) {
	st := &pl.roleState
	viewID := bagViewIDOfWire(dstView)
	if viewID == 0 {
		s.trace("equip move: 未知背包页 %d client=%d", dstView, pl.Conn1)
		return
	}
	index := -1
	for i := range st.Equips {
		if st.Equips[i].Slot == int(srcSlot) {
			index = i
			break
		}
	}
	if index < 0 {
		s.trace("equip move: 装备槽 %d 无件 client=%d", srcSlot, pl.Conn1)
		return
	}
	equip := st.Equips[index]
	slot := s.resolveBagSlot(pl, st, viewID, int(dstSlot), equip.ConfigID, "equip move 脱装")
	if slot == 0 {
		return
	}
	st.Equips = append(st.Equips[:index], st.Equips[index+1:]...)
	push, err := s.stackOrAddBagItem(pl, rolestate.BagItem{
		Slot:      slot,
		ConfigID:  equip.ConfigID,
		ItemType:  s.itemTypeWithCatalog(equip.ItemType, equip.ConfigID),
		Amount:    1,
		ViewID:    viewID,
		Seq:       equip.Seq,
		UID:       equip.UID,
		EquipType: equip.EquipType, // 装备行渲染闸：脱下入包仍须真值
	})
	if err != nil {
		s.trace("equip move: 脱装入包帧编码失败 %s: %v", equip.ConfigID, err)
		return
	}
	s.sendToClient(pl, entergen.ViewRemoveFrame(uint16(equipViewWire), uint16(srcSlot)))
	for _, frame := range push {
		s.sendToClient(pl, frame)
	}
	s.pushEquipAppearance(pl)
	s.pushEquipSlotModel(pl, int(srcSlot)) // 官服方言：空槽显式回推裸身
	s.pushEquipPhyDef(pl)
	s.sendRoleSave(pl)
	s.trace("equip move: 脱 %s 槽 %d → 背包 %d.%d client=%d",
		equip.ConfigID, srcSlot, viewID, slot, pl.Conn1)
}

// moveEquip 穿装（背包 → equip）：0x19(源页,槽)+[换装退件行]+穿戴帧组
//（0x19(doll 槽)+0x18 纸娃娃行+appearance 整刷）+ PhyDef。槽位按 EquipType
// 槽表校验（显式槽不符/类型未收录拒收；0=自动槽）。
func (s *Server) moveEquip(pl *Player, srcView, srcSlot, dstSlot int32) {
	st := &pl.roleState
	viewID := bagViewIDOfWire(srcView)
	if viewID == 0 {
		s.trace("equip move: 未知背包页 %d client=%d", srcView, pl.Conn1)
		return
	}
	index := -1
	for i := range st.BagItems {
		if st.BagItems[i].ViewID == viewID && st.BagItems[i].Slot == int(srcSlot) {
			index = i
			break
		}
	}
	if index < 0 {
		s.trace("equip move: 背包 %d.%d 无件 client=%d", viewID, srcSlot, pl.Conn1)
		return
	}
	item := st.BagItems[index]
	equipType := s.equipTypeOfItem(item.ItemType, 0, item.EquipType, item.ConfigID)
	slot := equipSlotForEquipType(equipType, int(dstSlot), st.Equips)
	if slot == 0 {
		s.trace("equip move: %s equipType=%q 不可穿/槽不符（报槽 %d）拒收 client=%d",
			item.ConfigID, equipType, dstSlot, pl.Conn1)
		return
	}
	// 占槽换装：原槽件先摘下（稍后退回源格）。
	var swapped *rolestate.EquipItem
	for i := range st.Equips {
		if st.Equips[i].Slot == slot {
			retired := st.Equips[i]
			swapped = &retired
			st.Equips = append(st.Equips[:i], st.Equips[i+1:]...)
			break
		}
	}
	// 源行扣件（堆叠件抽 1，抽空清行）。
	stacked := item.Amount > 1
	if stacked {
		st.BagItems[index].Amount--
	} else {
		st.BagItems = append(st.BagItems[:index], st.BagItems[index+1:]...)
	}
	s.sendToClient(pl, entergen.ViewRemoveFrame(uint16(srcView), uint16(srcSlot)))
	if stacked {
		if push, err := entergen.BagItemPushFrames(st.BagItems[index]); err == nil {
			for _, frame := range push {
				s.sendToClient(pl, frame)
			}
		}
	}
	if swapped != nil {
		push, err := s.stackOrAddBagItem(pl, rolestate.BagItem{
			Slot:      int(srcSlot),
			ConfigID:  swapped.ConfigID,
			ItemType:  s.itemTypeWithCatalog(swapped.ItemType, swapped.ConfigID),
			Amount:    1,
			ViewID:    viewID,
			Seq:       swapped.Seq,
			UID:       swapped.UID,
			EquipType: swapped.EquipType,
		})
		if err != nil {
			s.trace("equip move: 退件帧编码失败 %s: %v", swapped.ConfigID, err)
		} else {
			for _, frame := range push {
				s.sendToClient(pl, frame)
			}
		}
	}
	worn := rolestate.EquipItem{
		Slot:      slot,
		ConfigID:  item.ConfigID,
		ItemType:  item.ItemType,
		EquipType: equipType,
		Seq:       item.Seq,
		UID:       item.UID,
	}
	st.Equips = append(st.Equips, worn)
	// 穿戴帧组（2433 参考端 pushWearFrames 序）：槽位清 → 纸娃娃行 →
	// appearance 整刷。
	s.sendToClient(pl, entergen.ViewRemoveFrame(uint16(equipViewWire), uint16(slot)))
	if row, err := entergen.EquipRowFrame(worn); err == nil {
		s.sendToClient(pl, row)
	} else {
		s.trace("equip move: 纸娃娃行编码失败 %s: %v", worn.ConfigID, err)
	}
	s.pushEquipAppearance(pl)
	s.pushEquipPhyDef(pl)
	s.sendRoleSave(pl)
	s.trace("equip move: 穿 %s（%s）→ 槽 %d（换装退件 %t）client=%d",
		item.ConfigID, equipType, slot, swapped != nil, pl.Conn1)
}

// moveInsideBag 背包内挪格/叠堆：0x19 源 + 0x19+0x18 目标。
func (s *Server) moveInsideBag(pl *Player, srcView, srcSlot, dstView, dstSlot int32) {
	st := &pl.roleState
	srcPage := bagViewIDOfWire(srcView)
	dstPage := bagViewIDOfWire(dstView)
	if srcPage == 0 || dstPage == 0 {
		s.trace("equip move: 未知背包页 %d→%d client=%d", srcView, dstView, pl.Conn1)
		return
	}
	index := -1
	for i := range st.BagItems {
		if st.BagItems[i].ViewID == srcPage && st.BagItems[i].Slot == int(srcSlot) {
			index = i
			break
		}
	}
	if index < 0 {
		s.trace("equip move: 背包 %d.%d 无件 client=%d", srcPage, srcSlot, pl.Conn1)
		return
	}
	item := st.BagItems[index]
	// 同 configID 同页目标格 → 纯叠堆（并入后不得超堆叠上限——超限按
	// 普通挪格走）。
	if dstSlot != 0 {
		for i := range st.BagItems {
			if i == index {
				continue
			}
			if st.BagItems[i].ViewID == dstPage && st.BagItems[i].Slot == int(dstSlot) &&
				st.BagItems[i].ConfigID == item.ConfigID && s.bagRowCanAbsorb(&st.BagItems[i], item.Amount) {
				st.BagItems[i].Amount += item.Amount
				st.BagItems = append(st.BagItems[:index], st.BagItems[index+1:]...)
				s.sendToClient(pl, entergen.ViewRemoveFrame(uint16(srcView), uint16(srcSlot)))
				if push, err := entergen.BagItemPushFrames(st.BagItems[i]); err == nil {
					for _, frame := range push {
						s.sendToClient(pl, frame)
					}
				}
				s.sendRoleSave(pl)
				s.trace("equip move: 叠堆 %s x%d → %d.%d client=%d",
					item.ConfigID, item.Amount, dstPage, dstSlot, pl.Conn1)
				return
			}
		}
	}
	slot := s.resolveBagSlot(pl, st, dstPage, int(dstSlot), item.ConfigID, "equip move 挪格")
	if slot == 0 {
		return
	}
	st.BagItems[index].ViewID = dstPage
	st.BagItems[index].Slot = slot
	s.sendToClient(pl, entergen.ViewRemoveFrame(uint16(srcView), uint16(srcSlot)))
	if push, err := entergen.BagItemPushFrames(st.BagItems[index]); err == nil {
		for _, frame := range push {
			s.sendToClient(pl, frame)
		}
	}
	s.sendRoleSave(pl)
	s.trace("equip move: 挪 %s %d.%d → %d.%d client=%d",
		item.ConfigID, srcPage, srcSlot, dstPage, slot, pl.Conn1)
}
