package fssoloworld

// taskreward.go 任务奖励发放（帧序按官服引导抓包；帧编码见
// internal/entergen/taskrewards.go，取数面见 internal/taskdata/prize.go）：
//
//	步骤奖励（交互步记满时，在目标行 curnum 写入之前）
//	  入包帧 → m1 [1,1,"9645","<物品>",<数量>] 逐件 → m1 [1,1,"9697",…]
//	  （LostItems 扣除提示）                              （#2089..#2091 / #6196..#6200）
//	完成奖励（完成簇开头、完成位图之前）
//	  物品：入包帧 → m1 9645 逐件 → m1 [1,8,"10012,<物品>,<数量>|…"]
//	  碎银：0x10 CapitalType1 → m1 [1,1,"1303",<数量>]
//	  声望：Repute_Record 加行/写格 → m1 [1,1,"9693","<声望名>",<数量>]
//	  历练：0x10 LiveGroove_<下标> → m1 [1,1,"8055",<点数>]
//	                                                  （#1931..#1939 / #6827..#6852）
//
// 入包走 npcshop 的同一条链（叠堆/新行/容量预检）；目录里没有的物品或背包
// 放不下时那一件不发也不提示。官银（CapitalType2，prize_base[1]）入账并
// 推余额帧；奖励串其余字段未建模，不发。

import (
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

// npcTaskGrantItems 发放物品清单并追加提示帧。list＝整单再发一张 m1 [1,8,
// "10012,…|"] 清单（官服完成奖励有、步骤奖励没有）。返回是否发出了东西。
// 调用方持 taskMu。
func (s *Server) npcTaskGrantItems(pl *Player, out *taskFrames, items []taskdata.PrizeItem, what string, list bool) bool {
	granted := make([]taskdata.PrizeItem, 0, len(items))
	for _, item := range items {
		frames := make([][]byte, 0, 2)
		if !s.grantItemUnits(pl, item.ID, item.Num, what, func(row rolestate.BagItem, configID string) {
			push, err := entergen.BagItemPushFrames(row)
			if err != nil {
				s.trace("%s: 物品 %s 入包帧编码失败: %v", what, configID, err)
				return
			}
			frames = append(frames, push...)
		}) {
			continue // 目录缺席/背包放不下：拒发留痕在 grantItemUnits
		}
		out.extend(frames)
		out.add(entergen.TaskItemGainNotice(item.ID, item.Num))
		granted = append(granted, item)
	}
	if len(granted) == 0 {
		return false
	}
	if list {
		out.add(entergen.TaskItemListNotice(granted))
	}
	s.trace("%s: 发放物品 %v client=%d", what, granted, pl.Conn1)
	return true
}

// npcTaskTakeItems 扣除物品清单：背包里同 configID 的行逐行扣减（清行发
// 视图删行，减堆发行），随后逐件提示 m1 [1,1,"9697","<物品>",<数量>]。背包
// 里不够的件数只扣现有的——步骤照常推进。调用方持 taskMu。
func (s *Server) npcTaskTakeItems(pl *Player, out *taskFrames, items []taskdata.PrizeItem) {
	for _, item := range items {
		remaining := item.Num
		for i := 0; i < len(pl.roleState.BagItems) && remaining > 0; {
			row := &pl.roleState.BagItems[i]
			if row.ConfigID != item.ID {
				i++
				continue
			}
			take := min(row.Amount, remaining)
			row.Amount -= take
			remaining -= take
			if row.Amount == 0 {
				out.addAll(entergen.BagRowRemoveFrames(entergen.BagViewOf(row.ViewID), uint16(row.Slot)), nil)
				pl.roleState.BagItems = append(pl.roleState.BagItems[:i], pl.roleState.BagItems[i+1:]...)
				continue
			}
			if push, err := entergen.BagItemPushFrames(*row); err == nil {
				out.extend(push)
			}
			i++
		}
		if taken := item.Num - remaining; taken > 0 {
			out.add(entergen.TaskItemLoseNotice(item.ID, taken))
		}
	}
}

// npcTaskStepReward 步骤奖励（InteractiveList GetItems / LostItems）。调用方
// 持 taskMu。
func (s *Server) npcTaskStepReward(pl *Player, out *taskFrames, step taskdata.Step) {
	if items := taskdata.ParseItemSpec(step.GetItems); len(items) > 0 {
		s.npcTaskGrantItems(pl, out, items, "npc task: 步骤奖励 "+step.TrackInfo, false)
	}
	if items := taskdata.ParseItemSpec(step.LostItems); len(items) > 0 {
		s.npcTaskTakeItems(pl, out, items)
	}
}

// npcTaskCompleteReward 完成奖励（Prize 域奖励串）：物品 → 碎银 → 声望 →
// 历练。调用方持 taskMu。
func (s *Server) npcTaskCompleteReward(pl *Player, out *taskFrames, task taskdata.Task) {
	if task.Prize == "" {
		return
	}
	what := "npc task: 完成奖励 " + taskdata.StripRichText(task.Title)
	s.npcTaskGrantItems(pl, out, task.PrizeItems(), what, true)
	id := pl.entergenIdentity()
	if n := task.PrizeMoney(); n > 0 {
		pl.roleState.Money += n
		out.add(entergen.TaskMoneyFrame(id, pl.roleState.Money))
		out.add(entergen.TaskMoneyNotice(n))
	}
	if n := task.PrizeOfficialSilver(); n > 0 {
		pl.roleState.OfficialSilver += n
		out.add(entergen.TaskOfficialSilverFrame(id, pl.roleState.OfficialSilver))
	}
	if name, count := task.PrizeRepute(); count > 0 {
		row, total, added := pl.roleState.AddRepute(name, count)
		if added {
			out.add(entergen.TaskReputeRowFrame(id.ObjectID, id.OwnerID, uint16(row), name, total))
		} else {
			out.add(entergen.TaskReputeCountFrame(id.ObjectID, id.OwnerID, uint16(row), total))
		}
		out.add(entergen.TaskReputeNotice(name, count))
	}
	if index, count := task.PrizeLiveGroove(); count > 0 {
		if total, ok := pl.roleState.AddLiveGroove(index, count); ok {
			out.add(entergen.TaskLiveGrooveFrame(id, index, total))
			out.add(entergen.TaskLiveGrooveNotice(count / 1000))
		}
	}
}
