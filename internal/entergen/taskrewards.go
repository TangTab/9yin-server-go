package entergen

// taskrewards.go 任务奖励的帧编码（官服引导抓包逐帧原字节定版）：
//
//	碎银  0x10 CapitalType1=<新余额>（#1931）→ m1 [1,1,"1303",<数量>]（#1932）
//	声望  Repute_Record 行（名称、数值、标志；新声望 0x11 加行 #1933，已有
//	      声望 0x13 写数值列 #2409）→ m1 [1,1,"9693","<声望名>",<数量>]（#1935）
//	历练  0x10 LiveGroove_<下标>=<新余额>（#1938）→ m1 [1,1,"8055",<点数>]（#1939）
//	物品  入包帧（调用方给）→ m1 [1,1,"9645","<物品>",<数量>] 逐件（#2091）→
//	      m1 [1,8,"10012,<物品>,<数量>|…"] 整单一张（#6848）
//	扣除  m1 [1,1,"9697","<物品>",<数量>]（#6200）
//
// 发放与帧序编排见 fssoloworld/taskreward.go。

import (
	"fmt"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

var recordRepute = clientdata.MustRecordIndexByName("Repute_Record")

// TaskReputeFlag 声望记录行的标志列（官服行恒 1）。
const TaskReputeFlag int32 = 1

// TaskMoneyFrame 碎银余额帧 0x10 CapitalType1=<余额>。
func TaskMoneyFrame(player Identity, money int64) ([]byte, error) {
	return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0, []IndexedProperty{
		NamedProperty("CapitalType1", Int64Value(money)),
	})
}

// TaskReputeRowFrame 新声望加行 0x11 Repute_Record row=<行>。
func TaskReputeRowFrame(objectID, ownerID uint32, row uint16, name string, count int32) ([]byte, error) {
	return RecordAddRowFrame(objectID, ownerID, recordRepute, row, []RecordCell{
		RecordString(name), RecordInt(uint32(count)), RecordInt(uint32(TaskReputeFlag)),
	})
}

// TaskReputeCountFrame 已有声望逐格写数值（列 1）。
func TaskReputeCountFrame(objectID, ownerID uint32, row uint16, count int32) ([]byte, error) {
	return RecordGridFrame(objectID, ownerID, recordRepute, row, GridInt(1, uint32(count)))
}

// TaskLiveGrooveFrame 历练余额帧 0x10 LiveGroove_<下标>=<余额>（下标 1..5）。
func TaskLiveGrooveFrame(player Identity, index int, total int32) ([]byte, error) {
	if index < 1 || index > 5 {
		return nil, fmt.Errorf("live groove index %d out of range", index)
	}
	return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0, []IndexedProperty{
		NamedProperty("LiveGroove_"+fmt.Sprint(index), Int32Value(total)),
	})
}

// TaskPlayerFlagFrame 玩家标志属性单写 0x10（CantMove / CantBeAttack；官服
// 影像前后 #2439/#2440 置 1、#1789/#1790 复 0）。
func TaskPlayerFlagFrame(player Identity, name string, value byte) ([]byte, error) {
	return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0, []IndexedProperty{
		NamedProperty(name, ByteValue(value)),
	})
}

// TaskMoneyNotice 获得碎银提示 m1 [1,1,"1303",<数量>]。
func TaskMoneyNotice(amount int64) ([]byte, error) {
	return CustomMessageFrame(1, CustomInt(1), CustomString("1303"), CustomInt(int32(amount)))
}

// TaskReputeNotice 获得声望提示 m1 [1,1,"9693","<声望名>",<数量>]。
func TaskReputeNotice(name string, count int32) ([]byte, error) {
	return CustomMessageFrame(1, CustomInt(1), CustomString("9693"), CustomString(name), CustomInt(count))
}

// TaskLiveGrooveNotice 获得历练提示 m1 [1,1,"8055",<点数>]。
func TaskLiveGrooveNotice(points int32) ([]byte, error) {
	return CustomMessageFrame(1, CustomInt(1), CustomString("8055"), CustomInt(points))
}

// TaskItemGainNotice 获得物品提示 m1 [1,1,"9645","<物品>",<数量>]。
func TaskItemGainNotice(configID string, num int) ([]byte, error) {
	return CustomMessageFrame(1, CustomInt(1), CustomString("9645"), CustomString(configID), CustomInt(int32(num)))
}

// TaskItemLoseNotice 失去物品提示 m1 [1,1,"9697","<物品>",<数量>]。
func TaskItemLoseNotice(configID string, num int) ([]byte, error) {
	return CustomMessageFrame(1, CustomInt(1), CustomString("9697"), CustomString(configID), CustomInt(int32(num)))
}

// TaskItemListNotice 物品清单提示 m1 [1,8,"10012,<物品>,<数量>|…"]。
func TaskItemListNotice(items []taskdata.PrizeItem) ([]byte, error) {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("10012,%s,%d|", item.ID, item.Num))
	}
	return CustomMessageFrame(1, CustomInt(8), CustomString(strings.Join(parts, "")))
}

// TaskMovieEndFrame 影像结束应答 m107（官服 #1791：客户端上报 m806 影像
// 结束后，紧随两帧标志复位之后下发）。
func TaskMovieEndFrame() ([]byte, error) {
	return CustomMessageFrame(107)
}
