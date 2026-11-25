package entergen

// taskrewards_test.go 任务奖励与影像收尾帧对拍官服引导抓包原字节（帧号是
// S2C 流内的 0 基序号；玩家身份 officialPlayer=350000f3/102a）。期望串由
// 抓包帧直接导出（tmp/cf/rawframe.py），不要手抄。

import (
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

const (
	officialMoneyFrame   = "1000f30000352a1000000100a301e803000000000000"                                                                               // #1931
	officialMoneyNotice  = "1e0400020100000002010000000605000000313330330002e8030000"                                                                   // #1932
	officialReputeRow    = "1100f30000352a1000002c00000001000f0000007265707574655f6a69616e676875000a00000001000000"                                     // #1933
	officialReputeNotice = "1e05000201000000020100000006050000003936393300060f0000007265707574655f6a69616e67687500020a000000"                           // #1935
	officialGrooveFrame5 = "1000f30000352a10000001002a0370170000"                                                                                       // #1938
	officialGrooveNotice = "1e040002010000000201000000060500000038303535000205000000"                                                                   // #1939
	officialItemGain     = "1e05000201000000020100000006050000003936343500061300000074626f6f6b5f43535f6a685f637167663033000201000000"                   // #2091
	officialReputeCount  = "1300f30000352a1000002c0001000000011e000000"                                                                                 // #2409
	officialFlagAttack   = "1000f30000352a1000000100cd0001"                                                                                             // #2439
	officialFlagMove     = "1000f30000352a1000000100cb0001"                                                                                             // #2440
	officialGrooveFrame4 = "1000f30000352a10000001002903a8160000"                                                                                       // #3567
	officialItemLose     = "1e0500020100000002010000000605000000393639370006100000006974656d5f6d757365626f726e5f31000201000000"                         // #6200
	officialItemList1    = "1e030002010000000208000000061c00000031303031322c626f6f6b5f71696e67676f6e675f375f66632c317c00"                               // #6848
	officialItemList2    = "1e030002010000000208000000062b00000031303031322c6164646974656d5f303031312c317c31303031322c626f785f63646a685f3030312c317c00" // #12512
	officialMovieEnd     = "1e0100026b000000"                                                                                                           // #1791
)

func TestTaskRewardFramesMatchOfficial(t *testing.T) {
	// 1340 完成奖励（101.701s）：碎银 #1931/#1932 → 声望加行 #1933/#1935 →
	// 历练 #1938/#1939。
	frame, err := TaskMoneyFrame(officialPlayer, 1000)
	assertFrame(t, frame, err, officialMoneyFrame, "碎银余额帧（#1931）")
	frame, err = TaskMoneyNotice(1000)
	assertFrame(t, frame, err, officialMoneyNotice, "碎银提示（#1932）")
	frame, err = TaskReputeRowFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, "repute_jianghu", 10)
	assertFrame(t, frame, err, officialReputeRow, "声望加行（#1933）")
	frame, err = TaskReputeNotice("repute_jianghu", 10)
	assertFrame(t, frame, err, officialReputeNotice, "声望提示（#1935）")
	frame, err = TaskLiveGrooveFrame(officialPlayer, 5, 6000)
	assertFrame(t, frame, err, officialGrooveFrame5, "历练余额帧 LiveGroove_5（#1938）")
	frame, err = TaskLiveGrooveNotice(5)
	assertFrame(t, frame, err, officialGrooveNotice, "历练提示（#1939）")

	// 1347 完成（148.094s）：已有江湖声望逐格写数值；历练下标取奖励串（1341
	// 的 LiveGroove=4 → #3567 写 LiveGroove_4）。
	frame, err = TaskReputeCountFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, 30)
	assertFrame(t, frame, err, officialReputeCount, "声望数值写格（#2409）")
	frame, err = TaskLiveGrooveFrame(officialPlayer, 4, 5800)
	assertFrame(t, frame, err, officialGrooveFrame4, "历练余额帧 LiveGroove_4（#3567）")
	if _, err = TaskLiveGrooveFrame(officialPlayer, 6, 1); err == nil {
		t.Fatal("历练下标越界应报错")
	}

	// 物品：步骤奖励 #2091、完成奖励清单 #6848/#12512、扣除提示 #6200。
	frame, err = TaskItemGainNotice("tbook_CS_jh_cqgf03", 1)
	assertFrame(t, frame, err, officialItemGain, "获得物品提示（#2091）")
	frame, err = TaskItemListNotice([]taskdata.PrizeItem{{ID: "book_qinggong_7_fc", Num: 1}})
	assertFrame(t, frame, err, officialItemList1, "物品清单提示（#6848）")
	frame, err = TaskItemListNotice([]taskdata.PrizeItem{{ID: "additem_0011", Num: 1}, {ID: "box_cdjh_001", Num: 1}})
	assertFrame(t, frame, err, officialItemList2, "两件物品清单提示（#12512）")
	frame, err = TaskItemLoseNotice("item_museborn_1", 1)
	assertFrame(t, frame, err, officialItemLose, "失去物品提示（#6200）")

	// 影像：接取时官服置冻结标志（#2439/#2440），结束应答 m107（#1791）。
	frame, err = TaskPlayerFlagFrame(officialPlayer, "CantBeAttack", 1)
	assertFrame(t, frame, err, officialFlagAttack, "CantBeAttack=1（#2439）")
	frame, err = TaskPlayerFlagFrame(officialPlayer, "CantMove", 1)
	assertFrame(t, frame, err, officialFlagMove, "CantMove=1（#2440）")
	frame, err = TaskMovieEndFrame()
	assertFrame(t, frame, err, officialMovieEnd, "影像结束应答 m107（#1791）")
}
