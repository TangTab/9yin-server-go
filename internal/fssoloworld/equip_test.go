package fssoloworld

// msg30 移动物品/穿脱测试：穿=0x19 源行删+op10 模型单推（不发 view1 行）；
// 脱=0x19(view1,槽)+入包行+op10 裸身模型；PhyDef 随装同步；换装退件。

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// moveMsg msg30 线形（[30, src_view, src_slot, dst_view, dst_slot]）。
func moveMsg(srcView, srcSlot, dstView, dstSlot int32) entergen.CustomInbound {
	return entergen.CustomInbound{Values: []entergen.CustomInboundValue{
		{Type: 2, Int32: 30},
		{Type: 2, Int32: srcView},
		{Type: 2, Int32: srcSlot},
		{Type: 2, Int32: dstView},
		{Type: 2, Int32: dstSlot},
	}}
}

// equipTestPlayer 带三件布衣在背包装备页（121 页 slot 1..3）的玩家。
func equipTestPlayer() *Player {
	pl := &Player{Conn1: 21, State: PlOnline}
	pl.haveRoleState = true
	pl.roleState.Sex = 1
	pl.roleState.BagItems = []rolestate.BagItem{
		{Slot: 1, ConfigID: "shoes_b0001", ItemType: 1001, ViewID: 2, Seq: 1},
		{Slot: 2, ConfigID: "pants_b0001", ItemType: 1001, ViewID: 2, Seq: 2},
		{Slot: 3, ConfigID: "cloth_b0001", ItemType: 1001, ViewID: 2, Seq: 3},
	}
	return pl
}

func TestMoveItemEquipUnequip(t *testing.T) {
	s := newNpcTestServer(t)
	pl := equipTestPlayer()

	var sent [][]byte
	clientSendObserver = func(_ *Player, frame []byte) { sent = append(sent, frame) }
	defer func() { clientSendObserver = nil }()

	// 穿鞋：m30 [121,1,1,8]（官服线形）。
	s.handleMoveItem(pl, moveMsg(121, 1, 1, 8))
	if len(pl.roleState.Equips) != 1 || pl.roleState.Equips[0].Slot != 8 ||
		pl.roleState.Equips[0].ConfigID != "shoes_b0001" {
		t.Fatalf("穿后装备列: %+v", pl.roleState.Equips)
	}
	if len(pl.roleState.BagItems) != 2 {
		t.Fatalf("穿后背包行数 = %d", len(pl.roleState.BagItems))
	}
	if !frameContains(sent, []byte("obj\\char\\b_jianghu001\\b_shoes001")) {
		t.Fatalf("穿鞋未推件模型: %d 帧", len(sent))
	}
	if frameContains(sent, []byte("obj\\char\\b_jianghu000")) {
		t.Fatalf("穿鞋不应推裸身模型")
	}
	// 穿戴帧组（2433 参考端）：0x18 view1 纸娃娃行 + appearance（ActionSet
	// 回 1h 基线）。
	doll := false
	for _, f := range sent {
		if f[0] == 0x18 && f[1] == 1 && f[2] == 0 && f[3] == 8 {
			doll = true
		}
	}
	if !doll {
		t.Fatalf("穿鞋未发 view1 槽 8 纸娃娃行")
	}
	if !frameContains(sent, []byte("1h")) {
		t.Fatalf("穿鞋未发 appearance（ActionSet 基线缺失）")
	}
	// PhyDef 同步到 6。
	if !phyDefFrameValue(sent, 6) {
		t.Fatalf("穿鞋后 PhyDef 帧缺失或值非 6")
	}

	sent = nil
	// 脱鞋：m30 [1,8,121,0]（0=自动入位）。
	s.handleMoveItem(pl, moveMsg(1, 8, 121, 0))
	if len(pl.roleState.Equips) != 0 {
		t.Fatalf("脱后装备列: %+v", pl.roleState.Equips)
	}
	if len(pl.roleState.BagItems) != 3 {
		t.Fatalf("脱后背包行数 = %d", len(pl.roleState.BagItems))
	}
	// 裸身模型回推 + 0x19(view1,8) 删行 + 入包行（ConfigID）。
	if !frameContains(sent, []byte("obj\\char\\b_jianghu000\\b_shoes000")) {
		t.Fatalf("脱鞋未推裸身模型")
	}
	remove := entergen.ViewRemoveFrame(1, 8)
	found := false
	for _, f := range sent {
		if bytes.Equal(f, remove) {
			found = true
		}
	}
	if !found {
		t.Fatalf("脱鞋未发 0x19(view1,8)")
	}
	if !frameContains(sent, []byte("shoes_b0001")) {
		t.Fatalf("脱鞋未发背包行")
	}
	if !phyDefFrameValue(sent, 0) {
		t.Fatalf("脱鞋后 PhyDef 帧缺失或值非 0")
	}
}

func TestMoveItemEquipValidation(t *testing.T) {
	s := newNpcTestServer(t)

	// 错槽拒收：cloth 报槽 8。
	pl := equipTestPlayer()
	s.handleMoveItem(pl, moveMsg(121, 3, 1, 8))
	if len(pl.roleState.Equips) != 0 || len(pl.roleState.BagItems) != 3 {
		t.Fatalf("错槽不应移动物品: equips=%+v bag=%d", pl.roleState.Equips, len(pl.roleState.BagItems))
	}

	// 源格无件：静默拒。
	pl = equipTestPlayer()
	s.handleMoveItem(pl, moveMsg(121, 9, 1, 3))
	if len(pl.roleState.Equips) != 0 {
		t.Fatalf("空格不应穿装")
	}

	// 未知背包页：拒收。
	pl = equipTestPlayer()
	s.handleMoveItem(pl, moveMsg(122, 1, 1, 8))
	if len(pl.roleState.Equips) != 0 || len(pl.roleState.BagItems) != 3 {
		t.Fatalf("未知页不应移动物品")
	}
}

func TestMoveItemSwapAndBagMove(t *testing.T) {
	s := newNpcTestServer(t)
	pl := equipTestPlayer()
	clientSendObserver = func(*Player, []byte) {}
	defer func() { clientSendObserver = nil }()

	// 穿 cloth_b0001 → 槽 3；再造一件 cloth_b0005 换装退件。
	pl.roleState.BagItems = append(pl.roleState.BagItems, rolestate.BagItem{
		Slot: 4, ConfigID: "cloth_b0005", ItemType: 1001, ViewID: 2, Seq: 4})
	s.handleMoveItem(pl, moveMsg(121, 3, 1, 3))
	if got := pl.roleState.Equips[0].ConfigID; got != "cloth_b0001" {
		t.Fatalf("先穿 = %s", got)
	}
	s.handleMoveItem(pl, moveMsg(121, 4, 1, 3))
	if len(pl.roleState.Equips) != 1 || pl.roleState.Equips[0].ConfigID != "cloth_b0005" {
		t.Fatalf("换装后装备列: %+v", pl.roleState.Equips)
	}
	// 退件回源格 4。
	retired := false
	for _, item := range pl.roleState.BagItems {
		if item.ConfigID == "cloth_b0001" && item.Slot == 4 {
			retired = true
		}
	}
	if !retired {
		t.Fatalf("换装未退件回源格: %+v", pl.roleState.BagItems)
	}

	// 背包内挪格 121.1 → 121.7。
	s.handleMoveItem(pl, moveMsg(121, 1, 121, 7))
	moved := false
	for _, item := range pl.roleState.BagItems {
		if item.ConfigID == "shoes_b0001" && item.Slot == 7 {
			moved = true
		}
	}
	if !moved {
		t.Fatalf("背包挪格失败: %+v", pl.roleState.BagItems)
	}
}

// frameContains 帧字节包含子串（GBK 路径/ConfigID 断言用）。
func frameContains(frames [][]byte, needle []byte) bool {
	for _, f := range frames {
		if bytes.Contains(f, needle) {
			return true
		}
	}
	return false
}

// phyDefFrameValue 是否存在 op10 PhyDef=value 单值帧（12B 头 + idx u16 +
// i32 值；idx 按表名解析，免跨方言错位）。
func phyDefFrameValue(frames [][]byte, want int32) bool {
	idx := clientdata.MustPlayerProperty("PhyDef").Index
	for _, f := range frames {
		if len(f) != 18 || f[0] != 0x10 || f[1] != 0 {
			continue
		}
		if binary.LittleEndian.Uint16(f[10:12]) != 1 {
			continue
		}
		if binary.LittleEndian.Uint16(f[12:14]) != idx {
			continue
		}
		if int32(binary.LittleEndian.Uint32(f[14:18])) == want {
			return true
		}
	}
	return false
}
