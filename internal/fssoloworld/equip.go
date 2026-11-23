package fssoloworld

// C2S msg30 移动物品/穿脱（2026-11-23 官服真机方言，初始账号登陆-3 +
// 更改快捷键三会话定版，docs/official/初始号无装备身体模型定版.md）：
//
//	[30, src_view, src_slot, dst_view, dst_slot]
//
//	view 1=装备（纸娃娃，槽位 3=衣 4=裤 8=鞋…equipbox 口径）；2/121/123/125
//	=背包分类页（道具/装备/材料/任务）；dst_slot 0=自动入位（首个空格）。
//	应答方言（官服逐帧对齐）：
//	  - 0x19 源行删（脱=0x19(view1,槽)；穿=0x19(背包页,槽)）；
//	  - 脱下入包走 0x19 预清 + 0x18 行（BagItemPushFrames 同购销方言）；
//	  - 穿上**不发 view1 行**（纸娃娃客户端本地拖放自知），改推 op10 模型
//	    单值帧：空槽=jianghu000 裸身族、布衣=jianghu 系号族（equipmodel.go）；
//	  - op10 PhyDef 同步（0 基线 + 布衣每件 +6）。
// 占槽换装：原槽件退回源背包格（行帧 + 模型帧一并下发）。

import (
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// equipViewWire 装备（纸娃娃）wire 视图号。
const equipViewWire int32 = 1

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

// equipSlotForConfig ConfigID 前缀 → 身体槽位（{part}_{b|g}NNNN 形；
// equipbox 口径；其余件 0=按客户端所报槽位收）。
func equipSlotForConfig(configID string) int {
	switch {
	case hasEquipPrefix(configID, "cloth_"):
		return entergen.EquipSlotCloth
	case hasEquipPrefix(configID, "pants_"):
		return entergen.EquipSlotPants
	case hasEquipPrefix(configID, "shoes_"):
		return entergen.EquipSlotShoes
	case hasEquipPrefix(configID, "hat_"):
		return entergen.EquipSlotHat
	}
	return 0
}

// hasEquipPrefix ConfigID 是否 {prefix}{b|g}NNNN 形（性别位 + 数字尾）。
func hasEquipPrefix(configID, prefix string) bool {
	if len(configID) <= len(prefix)+1 || configID[:len(prefix)] != prefix {
		return false
	}
	switch configID[len(prefix)] {
	case 'b', 'g':
	default:
		return false
	}
	for i := len(prefix) + 1; i < len(configID); i++ {
		if configID[i] < '0' || configID[i] > '9' {
			return false
		}
	}
	return true
}

// handleMoveItem C2S 30 移动物品/穿脱。
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

// stackOrAddBagItem 入包：单件且同 configID 同页有行 → 叠堆（购销同方言），
// 否则新行。返回行帧。
func (s *Server) stackOrAddBagItem(pl *Player, item rolestate.BagItem) ([][]byte, error) {
	st := &pl.roleState
	if item.Amount == 1 {
		for i := range st.BagItems {
			if st.BagItems[i].ConfigID == item.ConfigID && st.BagItems[i].ViewID == item.ViewID {
				st.BagItems[i].Amount++
				return entergen.BagItemPushFrames(st.BagItems[i])
			}
		}
	}
	st.BagItems = append(st.BagItems, item)
	return entergen.BagItemPushFrames(item)
}

// pushEquipSlotModel op10 模型单值帧（槽位视觉：穿=件模型、空=裸身 000 族；
// 帽槽=无帽回填发型语义）。
func (s *Server) pushEquipSlotModel(pl *Player, slot int) {
	name := entergen.EquipSlotPropName(slot)
	if name == "" {
		return
	}
	value := entergen.EquipSlotModel(&pl.roleState, slot)
	frame, err := entergen.ObjectPropertyFrame(forkPlayerObjectID, forkPlayerOwnerID, 0,
		[]entergen.IndexedProperty{entergen.NamedProperty(name, entergen.StringValue(value))})
	if err != nil {
		s.trace("equip move: 模型帧编码失败 slot=%d: %v", slot, err)
		return
	}
	s.sendToClient(pl, frame)
}

// pushEquipPhyDef op10 PhyDef 同步（装备合计）。
func (s *Server) pushEquipPhyDef(pl *Player) {
	frame, err := entergen.ObjectPropertyFrame(forkPlayerObjectID, forkPlayerOwnerID, 0,
		[]entergen.IndexedProperty{entergen.NamedProperty("PhyDef", entergen.Int32Value(entergen.EquipPhyDef(&pl.roleState)))})
	if err != nil {
		s.trace("equip move: PhyDef 帧编码失败: %v", err)
		return
	}
	s.sendToClient(pl, frame)
}

// itemTypeWithCatalog 行 ItemType（装备行缺席时查目录兜底）。
func (s *Server) itemTypeWithCatalog(have int, configID string) int {
	if have != 0 {
		return have
	}
	if entry, ok := s.itemCatalogOf().Lookup(configID); ok {
		return entry.ItemType
	}
	return 0
}

// moveUnequip 脱装（equip → 背包）：0x19(view1,槽) + 入包行 + 模型/PhyDef 推送。
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
		Slot:     slot,
		ConfigID: equip.ConfigID,
		ItemType: s.itemTypeWithCatalog(equip.ItemType, equip.ConfigID),
		Amount:   1,
		ViewID:   viewID,
		Seq:      equip.Seq,
		UID:      equip.UID,
	})
	if err != nil {
		s.trace("equip move: 脱装入包帧编码失败 %s: %v", equip.ConfigID, err)
		return
	}
	s.sendToClient(pl, entergen.ViewRemoveFrame(uint16(equipViewWire), uint16(srcSlot)))
	for _, frame := range push {
		s.sendToClient(pl, frame)
	}
	s.pushEquipSlotModel(pl, int(srcSlot))
	s.pushEquipPhyDef(pl)
	s.sendRoleSave(pl)
	s.trace("equip move: 脱 %s 槽 %d → 背包 %d.%d client=%d",
		equip.ConfigID, srcSlot, viewID, slot, pl.Conn1)
}

// moveEquip 穿装（背包 → equip）：0x19(源页,槽) + 模型/PhyDef 推送（不发
// view1 行——官服实证）。占槽换装：原件退回源格。
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
	slot := int(dstSlot)
	if expected := equipSlotForConfig(item.ConfigID); expected != 0 {
		if slot != expected {
			s.trace("equip move: %s 应入槽 %d 实报 %d 拒收 client=%d", item.ConfigID, expected, slot, pl.Conn1)
			return
		}
	} else if slot < 1 || slot > 52 {
		s.trace("equip move: 装备槽越界 %d client=%d", slot, pl.Conn1)
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
			Slot:     int(srcSlot),
			ConfigID: swapped.ConfigID,
			ItemType: s.itemTypeWithCatalog(swapped.ItemType, swapped.ConfigID),
			Amount:   1,
			ViewID:   viewID,
			Seq:      swapped.Seq,
			UID:      swapped.UID,
		})
		if err != nil {
			s.trace("equip move: 退件帧编码失败 %s: %v", swapped.ConfigID, err)
		} else {
			for _, frame := range push {
				s.sendToClient(pl, frame)
			}
		}
	}
	st.Equips = append(st.Equips, rolestate.EquipItem{
		Slot:      slot,
		ConfigID:  item.ConfigID,
		ItemType:  item.ItemType,
		EquipType: equipTypeOf(s, item.ConfigID),
		Seq:       item.Seq,
		UID:       item.UID,
	})
	s.pushEquipSlotModel(pl, slot)
	s.pushEquipPhyDef(pl)
	s.sendRoleSave(pl)
	s.trace("equip move: 穿 %s → 槽 %d（换装退件 %t）client=%d",
		item.ConfigID, slot, swapped != nil, pl.Conn1)
}

// equipTypeOf 目录 EquipType（无目录空串）。
func equipTypeOf(s *Server, configID string) string {
	if entry, ok := s.itemCatalogOf().Lookup(configID); ok {
		return entry.EquipType
	}
	return ""
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
	// 同 configID 同页目标格 → 纯叠堆。
	if dstSlot != 0 {
		for i := range st.BagItems {
			if i == index {
				continue
			}
			if st.BagItems[i].ViewID == dstPage && st.BagItems[i].Slot == int(dstSlot) &&
				st.BagItems[i].ConfigID == item.ConfigID {
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
