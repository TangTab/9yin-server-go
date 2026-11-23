package entergen

// 背包物品视图行（2433 线 inventory_frames.go itemCommonProps/pushBagItemFrames
// 口径）：行属性集按本仓 2433 属性表名解析（ConfigID/ItemType/Amount/
// MaxAmount/BindStatus/UniqueID——idx 与参考端 golden 表不同，名寻址免错位）；
// 行推送「0x19 预清 + 0x18 行」成对方言（客户端对重复 slot 行残留敏感）。

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// 背包容器视图编号（2433 线 bagViewOf：ViewID 分类页 → wire 视图；
// 1 道具/2 装备/3 材料/4 任务，道具页=视图 2）。
const equipDollView uint16 = 1 // View1 纸娃娃（行号=身体槽位）

func bagViewOf(viewID int) uint16 {
	switch viewID {
	case 2:
		return 121 // 装备页
	case 3:
		return 123 // 材料页
	case 4:
		return 125 // 任务页
	default:
		return 2 // 道具页（ViewID 1）
	}
}

// BagViewOf 背包分类页 → wire 视图（bagViewOf 导出面：购销/移除等
// 服务端业务共用）。
func BagViewOf(viewID int) uint16 { return bagViewOf(viewID) }

// ViewRemoveFrame 0x19 视图删行：[19][u16 vid][u16 idx]（5B；行推送
// 预清/删除共用）。
func ViewRemoveFrame(viewID, objectIndex uint16) []byte {
	msg := make([]byte, 5)
	msg[0] = 0x19
	putU16(msg, 1, viewID)
	putU16(msg, 3, objectIndex)
	return msg
}

// bagItemViewRow 背包行属性集（itemCommonProps 六项；Amount 下发实际
// 堆叠数、UniqueID 实例身份——强化/随机属性键控位，空串可省）。
func bagItemViewRow(item rolestate.BagItem) []ViewProperty {
	uid := item.UID
	if uid == "" {
		uid = itemUIDOf(item.Seq, item.Slot)
	}
	props := []ViewProperty{
		{playerIdx("ConfigID"), StringValue(item.ConfigID)},
		{playerIdx("ItemType"), Int32Value(int32(item.ItemType))},
		{playerIdx("Amount"), Int32Value(int32(item.Amount))},
		{playerIdx("MaxAmount"), Int32Value(int32(maxAmountOf(item)))},
		{playerIdx("BindStatus"), Int32Value(0)},
		{playerIdx("UniqueID"), StringValue(uid)},
	}
	// 装备页行补装备域（2433 bagRowViewAdd 同款：EquipType 必须真值
	// 否则不渲染；其余缺省 MaxBuild/CurBuild=2、Hardiness 族=100、
	// ColorLevel=1、Level=0）。
	if item.ViewID == 2 {
		colorLevel := item.ColorLevel
		if colorLevel <= 0 {
			colorLevel = 1
		}
		props = append(props,
			ViewProperty{playerIdx("EquipType"), StringValue(item.EquipType)},
			ViewProperty{playerIdx("MaxBuildCount"), Int32Value(2)},
			ViewProperty{playerIdx("CurBuildCount"), Int32Value(2)},
			ViewProperty{playerIdx("RecoverFlag"), ByteValue(0)},
			ViewProperty{playerIdx("ColorLevel"), Int32Value(int32(colorLevel))},
			ViewProperty{playerIdx("Hardiness"), Int32Value(100)},
			ViewProperty{playerIdx("MaxHardiness"), Int32Value(100)},
			ViewProperty{playerIdx("Level"), ByteValue(0)},
		)
		if item.Photo != "" {
			props = append(props, ViewProperty{playerIdx("Photo"), StringValue(item.Photo)})
		}
	}
	return props
}

// itemUIDOf 实例身份合成（2433 itemUID 同形：md5 前 8 字节 hex=16 字符；
// Seq 全档唯一且随存档持久——重登稳定）。
func itemUIDOf(seq int64, slot int) string {
	seed := fmt.Sprintf("bag:%d:%d", seq, slot)
	digest := md5.Sum([]byte(seed))
	return hex.EncodeToString(digest[:8])
}

// maxAmountOf 堆叠上限：行值（入包时取自物品模板行 MaxAmount——ini/Item
// 构建产物，itemdata 目录）> 无行值回落「与 Amount 等值、单件为 1」（物品
// 目录缺席/旧档行；客户端聚合显示不依赖此值）。
func maxAmountOf(item rolestate.BagItem) int {
	if item.MaxAmount > 0 {
		return item.MaxAmount
	}
	if item.Amount > 1 {
		return item.Amount
	}
	return 1
}

// BagItemPushFrames 单件入包推送（0x19 预清 + 0x18 行成对——GM 发放/
// 后续拾取链共用；slot=行 objectIndex）。
func BagItemPushFrames(item rolestate.BagItem) ([][]byte, error) {
	view := bagViewOf(item.ViewID)
	slot := uint16(item.Slot)
	row, err := ViewAddFrame(view, slot, bagItemViewRow(item))
	if err != nil {
		return nil, err
	}
	return [][]byte{ViewRemoveFrame(view, slot), row}, nil
}

// BagItemViewFrames 全背包行推送（Ready 段空容器族之后；空背包零帧）。
func BagItemViewFrames(state rolestate.RoleState) ([][]byte, error) {
	frames := make([][]byte, 0, len(state.BagItems)*2)
	for _, item := range state.BagItems {
		push, err := BagItemPushFrames(item)
		if err != nil {
			return nil, err
		}
		frames = append(frames, push...)
	}
	return frames, nil
}

// equipItemViewRow View1 纸娃娃行属性集（2433 线 equipmentViewAdd 同款：
// 通用 6 项 + EquipType 真值 + 精炼/耐久/品阶族；行号=身体槽位）。
func equipItemViewRow(item rolestate.EquipItem) []ViewProperty {
	uid := item.UID
	if uid == "" {
		uid = itemUIDOf(item.Seq, item.Slot)
	}
	colorLevel := 1
	props := []ViewProperty{
		{playerIdx("ConfigID"), StringValue(item.ConfigID)},
		{playerIdx("ItemType"), Int32Value(int32(item.ItemType))},
		{playerIdx("Amount"), Int32Value(1)},
		{playerIdx("MaxAmount"), Int32Value(1)},
		{playerIdx("BindStatus"), Int32Value(0)},
		{playerIdx("UniqueID"), StringValue(uid)},
		{playerIdx("EquipType"), StringValue(item.EquipType)},
		{playerIdx("MaxBuildCount"), Int32Value(2)},
		{playerIdx("CurBuildCount"), Int32Value(2)},
		{playerIdx("RecoverFlag"), ByteValue(0)},
		{playerIdx("ColorLevel"), Int32Value(int32(colorLevel))},
		{playerIdx("Hardiness"), Int32Value(100)},
		{playerIdx("MaxHardiness"), Int32Value(100)},
		{playerIdx("Level"), ByteValue(0)},
	}
	return props
}

// EquipRowFrame 单件纸娃娃行（穿戴帧组用：0x19 槽位清 + 本行成对）。
func EquipRowFrame(item rolestate.EquipItem) ([]byte, error) {
	return ViewAddFrame(equipDollView, uint16(item.Slot), equipItemViewRow(item))
}

// EquipViewFrames 全纸娃娃行推送（登录整推——2433 线 inventoryFrames
// 序：纸娃娃行先于背包行；空装备零帧）。
func EquipViewFrames(state rolestate.RoleState) ([][]byte, error) {
	frames := make([][]byte, 0, len(state.Equips))
	for _, item := range state.Equips {
		row, err := EquipRowFrame(item)
		if err != nil {
			return nil, err
		}
		frames = append(frames, row)
	}
	return frames, nil
}
