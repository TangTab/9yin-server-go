package entergen

// guide.go 新手引导流的帧编码（官服女号「完整引导」/ 男号「跳过引导」两份
// 抓包逐帧原字节定版；流程编排见 fssoloworld/guide.go）：
//
//	追踪栏刷新  m390 [5, 0]（SERVER_CUSTOMMSG_REFRESH_TASK_TRACKINFO，女号
//	            #1508 / 男号 #1604、#2258）：客户端 custom_refresh_track_info
//	            → form_task_trace.on_ready——追踪栏窗体进场预载后是隐藏的，
//	            只在这一帧之后才显示并按已接表追踪标志重建追踪列表。
//	引导阶段    m414 ["gamestep", 主步, 子步]（#437 女 4/5、#547 男 5/5）
//	            ＋ 0x10 BornServerGameStep=主步×10+子步（#183=45、#293=55）
//	新手帮助    0x10 HelpStepID=<步骤>（#466 "chengdu_00"、#1792 "chengdu_02"）
//	输入锁      0x10 CanntOperator=1/0（待选态 #881 锁、点选后 #1532 解）
//	跳过簇      （男号 #1612..#1989，再战江湖）GiveItemRec 行 → 历练四帧
//	            （m1 8055 → m24 [玩家,1,点数] → m24 [玩家,40,"huodelilian"]
//	            → 0x10 LiveGroove_N）→ Repute_Record 行（标志 2）→ m142 生活
//	            新手帮助 ×4 → 入江湖属性（PowerLevel/YinLevel=1、解锁）。

import "github.com/lizhanyin/9yin-server-go/internal/clientdata"

// 服务器自定义消息号（客户端 custom_handler.lua 处理函数名对应的引擎绑定）。
const (
	serverCustomTaskTrackRefresh int32 = 390 // refresh_track_info
	serverCustomGameStep         int32 = 414 // SwitchManager gamestep
	serverCustomSingleMark       int32 = 543 // 跳过簇里 1346 置位后的空参消息（#1683）
)

var recordGiveItem = clientdata.MustRecordIndexByName("GiveItemRec")

// TaskTrackRefreshFrame 任务追踪栏刷新 m390 [5, 0]：首参 >0 客户端即显示
// 追踪栏（官服两号恒 5），次参是服务器时间（官服恒 0）。
func TaskTrackRefreshFrame() ([]byte, error) {
	return CustomMessageFrame(serverCustomTaskTrackRefresh, CustomInt(5), CustomInt(0))
}

// GameStepFrame 引导阶段 m414 ["gamestep", 主步, 子步]。
func GameStepFrame(main, sub int32) ([]byte, error) {
	return CustomMessageFrame(serverCustomGameStep, CustomString("gamestep"), CustomInt(main), CustomInt(sub))
}

// SingleStepMarkFrame 跳过簇里的空参消息 m543（#1683，1346 置位之后）。
func SingleStepMarkFrame() ([]byte, error) {
	return CustomMessageFrame(serverCustomSingleMark)
}

// BornGameStepFrame 出生服务端阶段 0x10 BornServerGameStep=<主步×10+子步>。
func BornGameStepFrame(player Identity, value int32) ([]byte, error) {
	return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0,
		[]IndexedProperty{NamedProperty("BornServerGameStep", Int32Value(value))})
}

// HelpStepFrame 新手帮助步骤 0x10 HelpStepID=<步骤>。
func HelpStepFrame(player Identity, step string) ([]byte, error) {
	return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0,
		[]IndexedProperty{NamedProperty("HelpStepID", StringValue(step))})
}

// InputLockFrame 输入锁 0x10 CanntOperator=1/0。
func InputLockFrame(player Identity, locked bool) ([]byte, error) {
	value := int32(0)
	if locked {
		value = 1
	}
	return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0,
		[]IndexedProperty{NamedProperty("CanntOperator", Int32Value(value))})
}

// ReliveFrame 复活点 0x10 RelivePositon=<串>。
func ReliveFrame(player Identity, relive string) ([]byte, error) {
	return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0,
		[]IndexedProperty{NamedProperty("RelivePositon", StringValue(relive))})
}

// GiveItemRowFrame 获得物品记录 0x11 GiveItemRec row=<行>（配置 ID、数量；
// 男号 #1695）。
func GiveItemRowFrame(objectID, ownerID uint32, row uint16, configID string, num int32) ([]byte, error) {
	return RecordAddRowFrame(objectID, ownerID, recordGiveItem, row, []RecordCell{
		RecordString(configID), RecordInt(uint32(num)),
	})
}

// LiveGrooveGainFrames 跳过簇的历练四帧：m1 [1,"8055",<点数>] → m24 [玩家,1,
// <点数>] → m24 [玩家,40,"huodelilian"] → 0x10 LiveGroove_<下标>=<新余额>
// （男号 #1700..#1703）。amount 是本次累加值、total 是累加后的余额（点数
// ＝amount/1000）。
func LiveGrooveGainFrames(player Identity, index int, amount, total int32) ([][]byte, error) {
	points := amount / 1000
	property, err := TaskLiveGrooveFrame(player, index, total)
	if err != nil {
		return nil, err
	}
	frames, err := buildFrames(
		func() ([]byte, error) { return TaskLiveGrooveNotice(points) },
		func() ([]byte, error) {
			return CustomMessageFrame(24, CustomObject(player.ObjectID, player.OwnerID), CustomInt(1), CustomInt(points))
		},
		func() ([]byte, error) {
			return CustomMessageFrame(24, CustomObject(player.ObjectID, player.OwnerID), CustomInt(40), CustomString("huodelilian"))
		},
	)
	if err != nil {
		return nil, err
	}
	return append(frames, property), nil
}

// ReputeRowFrameWithFlag 声望记录行 0x11 Repute_Record row=<行>（标志列
// 取调用方给值：任务完成奖励官服恒 1，跳过引导的再战江湖奖励是 2——男号
// #1720）。
func ReputeRowFrameWithFlag(objectID, ownerID uint32, row uint16, name string, count, flag int32) ([]byte, error) {
	return RecordAddRowFrame(objectID, ownerID, recordRepute, row, []RecordCell{
		RecordString(name), RecordInt(uint32(count)), RecordInt(uint32(flag)),
	})
}

// LifeHelpUnlockFrames 生活新手帮助解锁提示 m142 ×4（采集／制造／文化／
// 视觉）＋收尾 m98 [712]（男号 #1723..#1727；女号完成奖励 #1936..#1937 同
// 形）。
func LifeHelpUnlockFrames() ([][]byte, error) {
	names := []string{"caiji", "zhizao", "wenhua", "shijing"}
	builders := make([]func() ([]byte, error), 0, len(names)+1)
	for _, name := range names {
		key := "life_freshmanhelp_" + name
		builders = append(builders, func() ([]byte, error) {
			return CustomMessageFrame(142, CustomString(key))
		})
	}
	builders = append(builders, func() ([]byte, error) { return CustomMessageFrame(98, CustomInt(712)) })
	return buildFrames(builders...)
}

// FunctionOpenNoticeFrame 功能开启提示 m98 [<功能号>]（男号声望行之后的
// #1721 = 177）。
func FunctionOpenNoticeFrame(function int32) ([]byte, error) {
	return CustomMessageFrame(98, CustomInt(function))
}

// EnterWorldFrames 「进入江湖」属性对 0x10 PowerLevel=1 → 0x10 YinLevel=1
// （男号 #1946..#1947）。
func EnterWorldFrames(player Identity) ([][]byte, error) {
	return buildFrames(
		func() ([]byte, error) {
			return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0,
				[]IndexedProperty{NamedProperty("PowerLevel", Int32Value(1))})
		},
		func() ([]byte, error) {
			return ObjectPropertyFrame(player.ObjectID, player.OwnerID, 0,
				[]IndexedProperty{NamedProperty("YinLevel", Int32Value(1))})
		},
	)
}
