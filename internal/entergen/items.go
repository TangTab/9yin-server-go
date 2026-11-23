package entergen

// 背包物品视图行（2433 线 inventory_frames.go itemCommonProps/pushBagItemFrames
// 口径）：行属性集按本仓 2433 属性表名解析（ConfigID/ItemType/Amount/
// MaxAmount/BindStatus/UniqueID——idx 与参考端 golden 表不同，名寻址免错位）；
// 行推送「0x19 预清 + 0x18 行」成对方言（客户端对重复 slot 行残留敏感）。

import (
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// 背包容器视图编号（2433 线 bagViewOf：ViewID 分类页 → wire 视图；
// 1 道具/2 装备/3 材料/4 任务，道具页=视图 2）。
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
	props := []ViewProperty{
		{playerIdx("ConfigID"), StringValue(item.ConfigID)},
		{playerIdx("ItemType"), Int32Value(int32(item.ItemType))},
		{playerIdx("Amount"), Int32Value(int32(item.Amount))},
		{playerIdx("MaxAmount"), Int32Value(int32(maxAmountOf(item)))},
		{playerIdx("BindStatus"), Int32Value(0)},
	}
	if item.UID != "" {
		props = append(props, ViewProperty{playerIdx("UniqueID"), StringValue(item.UID)})
	}
	return props
}

// maxAmountOf 堆叠上限（无目录口径：非堆叠物 1，其余与 Amount 等值——
// 客户端聚合显示不依赖此值，ItemUse 目录接入后改真值）。
func maxAmountOf(item rolestate.BagItem) int {
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
