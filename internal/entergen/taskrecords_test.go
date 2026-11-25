package entergen

// taskrecords_test.go 任务记录编码对拍官服引导抓包（初始账号登陆-完整
// 引导-女.pcapng，8032）：每条用例的期望值是官服帧的明文字节（EE 去转义
// + XOR 之后），帧号是 S2C 流内的 0 基序号。玩家身份 obj=350000f3
// own=0000102a，NPC funnpcborn003035 = 3500004d/00000052。

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

var (
	officialPlayer = Identity{ObjectID: 0x350000F3, OwnerID: 0x0000102A}
	officialNPC    = Identity{ObjectID: 0x3500004D, OwnerID: 0x00000052}
)

// officialTask1340 官服 1340「武林高手」在任务主表里的行（Task_Accepted
// 行的数据源）。
func officialTask1340() taskdata.Task {
	return taskdata.Task{
		ID: 1340, Line: taskdata.LineGuide, TitleKey: "title_1340",
		TargetKey: "target_1340", ContextKey: "context_1340",
		AcceptNpc: "funnpcborn003031", SubmitNpc: "funnpcborn003031",
		AcceptScene: 300, SubmitScene: 300, SubmitNpcEx: "SubmitNpcEx_1340",
		Prize: "1000,0,repute_jianghu,10,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,5,5000,0,0",
	}
}

// mustHex 解官服帧明文。
func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	raw, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// assertFrame 编码结果与官服帧逐字节一致。
func assertFrame(t *testing.T, got []byte, err error, want string, what string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
	expected := mustHex(t, want)
	if !bytes.Equal(got, expected) {
		t.Fatalf("%s 与官服不一致:\n got %x\nwant %s", what, got, want)
	}
}

func TestTaskRecordRowMatchesOfficial(t *testing.T) {
	// 目标行四种形态（剧情 #1537、交互 #1988、杀怪 #2423/#2424、特殊
	// #3575、剧情步居中 #3576）：行号、各列编码（含 0 长度串）与 stamp
	// 低 20 位逐字节一致。stamp 传入官服原值（函数只保留其低 20 位以外
	// 的部分）。
	const story1340 = "1100f30000352a1000001e00000001003c05000006000000e20100000100000000000000010000001100000066756e6e7063626f726e3030333033310001000000000e00000073746f7279696e666f5f343832000000000018000000313334305f365f66756e6e7063626f726e30303330333100068807004f01000000000000000000000100000000"
	frame, err := TaskRecordRowFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, 1340,
		taskdata.Step{Subtype: taskdata.StepStory, Index: 482, Order: 1, MaxNum: 1, Key: "funnpcborn003031", TrackInfo: "storyinfo_482", HasLink: true},
		TaskRecordView{}, 1438814537734)
	assertFrame(t, frame, err, story1340, "1340 剧情步目标行（#1537）")

	const interact558 = "1100f30000352a1000001e000000010043050000030000002e0200000100000000000000010000001100000066756e6e7063626f726e303033303335000d0000004469616c6f6749445f3535380011000000696e746572616374696e666f5f353538000000000018000000313334375f335f66756e6e7063626f726e3030333033350003b808c05001000000000000000000000100000000"
	frame, err = TaskRecordRowFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, 1347,
		taskdata.Step{Subtype: taskdata.StepInteract, Index: 558, Order: 1, MaxNum: 1, Key: "funnpcborn003035", Scene: "DialogID_558", TrackInfo: "interactinfo_558", HasLink: true},
		TaskRecordView{}, 1446330808323)
	assertFrame(t, frame, err, interact558, "1347 交互步目标行（#1988）")

	const hunter522 = "1100f30000352a1000001e00000001003d050000010000000a0200000100000000000000010000000e0000004e7063426f726e3030333031310001000000000f00000068756e746572696e666f5f35323200000000000100000000012808404f01000000000000000000000100000000"
	frame, err = TaskRecordRowFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, 1341,
		taskdata.Step{Subtype: taskdata.StepHunter, Index: 522, Order: 1, MaxNum: 1, Key: "NpcBorn003011", TrackInfo: "hunterinfo_522"},
		TaskRecordView{}, 1439888320513)
	assertFrame(t, frame, err, hunter522, "1341 杀怪步目标行（#2423，scene 为 0 长度串、无超链接）")

	const special2061 = "1100f30000352a1000001e00010001003e05000008000000cb0000000200000000000000010000000500000032303631000100000000110000007370656369616c696e666f5f3230363100000000000c000000313334325f385f3230363100082c03804f01000000000000000000000100000000"
	frame, err = TaskRecordRowFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 1, 1342,
		taskdata.Step{Subtype: taskdata.StepSpecial, Index: 203, Order: 2, MaxNum: 1, Key: "2061", TrackInfo: "specialinfo_2061", HasLink: true},
		TaskRecordView{}, 1440961735688)
	assertFrame(t, frame, err, special2061, "1342 特殊步目标行（#3575，Index=SpecialType）")
}

func TestTaskAcceptedRowMatchesOfficial(t *testing.T) {
	// 已接行体（官服 0x29 批量帧 15B 头之后的行体，#1538 / #1992）。
	const row1340 = "3c0500000900000007000000536572766572002c0100001100000066756e6e7063626f726e303033303331000b0000007469746c655f3133343000000000000100000000000000ae1d036b0000000000000000000c0000007461726765745f31333430000d000000636f6e746578745f31333430001100000066756e6e7063626f726e303033303331002c010000110000005375626d69744e706345785f313334300042000000313030302c302c7265707574655f6a69616e6768752c31302c302c302c302c302c302c302c302c302c302c302c302c302c302c302c302c352c353030302c302c30000100000000010000000001000000000100000000010000000000000000000100000000000000000000000000000000"
	frame, err := TaskAcceptedRowFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, officialTask1340(),
		TaskAcceptedView{Status: TaskStatusDoing, Tracked: true, Step: 0, Accepted: 1795366318})
	assertFrame(t, frame, err, "1100f30000352a1000001d0000000100"+row1340, "1340 已接行（#1538 行体）")

	// 不在目录里的任务（建号种子的功能门行）：除任务号与固定列 "Server"
	// 外各列皆空/零（行体 134 字节）。
	unknown, err := TaskAcceptedRowFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 3, taskdata.Task{ID: 51002}, TaskAcceptedView{})
	if err != nil {
		t.Fatal(err)
	}
	body := unknown[16:]
	if len(body) != 134 || !bytes.HasPrefix(body, []byte{0x3a, 0xc7, 0x00, 0x00}) {
		t.Fatalf("目录外任务行体长度/任务号异常: %d %x", len(body), body[:8])
	}
	// Server 之后按列构造期望：Z(n)=n 个零字节，E=空串（长度 1 + NUL）。
	Z := func(n int) []byte { return bytes.Repeat([]byte{0}, n) }
	E := []byte{1, 0, 0, 0, 0}
	join := func(parts ...[]byte) []byte { return bytes.Join(parts, nil) }
	var empties [][]byte
	for k := 0; k < 5; k++ {
		empties = append(empties, E)
	}
	want := join(Z(4), E, E, Z(20), Z(5), E, E, E, Z(4), E, E, join(empties...), Z(5), E, Z(12))
	if got := body[19:]; !bytes.Equal(got, want) { // body[8:19] 是 "Server"
		t.Fatalf("目录外任务行体（Server 之后）异常: got %x want %x", got, want)
	}
}

func TestTaskRecordIncrementMatchesOfficial(t *testing.T) {
	// 步进/状态/标志的逐格写与删行帧。
	frame, err := TaskRecordGridFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, GridInt(RecordColCurNum, 1))
	assertFrame(t, frame, err, "1300f30000352a1000001e00010000000401000000", "目标行 curnum=1（#2092）")
	frame, err = TaskAcceptedGridFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, GridInt(TaskColStep, 1))
	assertFrame(t, frame, err, "1300f30000352a1000001d00010000000801000000", "已接行 step=1（#1539）")
	frame, err = TaskAcceptedGridFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, GridInt(TaskColTrack, 1))
	assertFrame(t, frame, err, "1300f30000352a1000001d00010000000701000000", "已接行 track=1（#1558）")
	frame, err = TaskAcceptedGridFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, GridInt(TaskColStatus, TaskStatusCanSubmit))
	assertFrame(t, frame, err, "1300f30000352a1000001d00010000000602000000", "已接行 status=2（#2354）")
	frame, err = TaskRecordGridFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, GridInt(RecordColFlags, 1))
	assertFrame(t, frame, err, "1300f30000352a1000001e00010000000901000000", "目标行 flags=1 杀怪步成为当前步（#2428）")
	frame, err = TaskRecordGridFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, GridInt(RecordColFlags, 32))
	assertFrame(t, frame, err, "1300f30000352a1000001e00010000000920000000", "目标行 flags=32（#2094）")
	frame, err = TaskRecordGridFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, GridInt(RecordColFlags, 48))
	assertFrame(t, frame, err, "1300f30000352a1000001e00010000000930000000", "目标行 flags=48（#2095）")
	frame, err = TaskAcceptedGridFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, GridInt(TaskColTrack, 0))
	assertFrame(t, frame, err, "1300f30000352a1000001d00010000000700000000", "已接行 track=0（#2064）")

	assertFrame(t, TaskRecordDeleteFrames(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, 1)[0], nil,
		"1200f30000352a1000001e000000", "删目标行（#1983）")
	assertFrame(t, TaskAcceptedDeleteFrame(officialPlayer.ObjectID, officialPlayer.OwnerID, 0), nil,
		"1200f30000352a1000001d000000", "删已接行（#1984）")
	if got := TaskRecordDeleteFrames(officialPlayer.ObjectID, officialPlayer.OwnerID, 0, 4); len(got) != 4 {
		t.Fatalf("多行删除应逐行成帧: %d", len(got))
	}
}

func TestTaskCompletedBitmapMatchesOfficial(t *testing.T) {
	// 映射：行=任务号/32、位=任务号%32（#1982：1340 → 行 41 位 28）。
	row, mask := TaskCompletedRow(1340)
	if row != 41 || mask != 1<<28 {
		t.Fatalf("1340 → 行 %d 位掩码 %#x, want 行 41 / 0x10000000", row, mask)
	}
	if row, _ := TaskCompletedRow(1347); row != 42 {
		t.Fatalf("1347 → 行 %d, want 42", row)
	}
	// 官服 #1982 的场景：登录簇位图 0..41 行全零（登录前没有任何已完成
	// 任务），1340 完成后行 41 写入位 28。这里的登录簇只在有已完成任务
	// 时才发行，所以完成前列为空 → 先把 0..41 行逐行加上，再写格。
	frames, err := TaskCompletedSetFrames(officialPlayer.ObjectID, officialPlayer.OwnerID, 1340, nil)
	if err != nil || len(frames) != 43 {
		t.Fatalf("完成前列为空时的置位帧数 = %d err=%v, want 43", len(frames), err)
	}
	assertFrame(t, frames[0], nil, "1100f30000352a1000001f000000010000000000", "完成位图行 0 加行（#1940）")
	assertFrame(t, frames[42], nil, "1300f30000352a1000001f00010029000000000010", "完成位图行 41 写格（#1982）")
	// 行客户端已有（登录簇发过 1340 所在的 0..41 行）：只写格（#3567）。
	frames, err = TaskCompletedSetFrames(officialPlayer.ObjectID, officialPlayer.OwnerID, 1341, []uint32{1340})
	if err != nil || len(frames) != 1 {
		t.Fatalf("已有行置位帧数 = %d err=%v, want 1", len(frames), err)
	}
	assertFrame(t, frames[0], nil, "1300f30000352a1000001f00010029000000000030", "完成位图行 41 写格（#3567）")
	// 行尚不存在（登录簇只下发到 41 行）：先加行再写格（#2413/#2414）。
	frames, err = TaskCompletedSetFrames(officialPlayer.ObjectID, officialPlayer.OwnerID, 1347, []uint32{1340})
	if err != nil || len(frames) != 2 {
		t.Fatalf("新行置位帧数 = %d err=%v, want 2", len(frames), err)
	}
	assertFrame(t, frames[0], nil, "1100f30000352a1000001f002a00010000000000", "完成位图行 42 加行（#2413）")
	assertFrame(t, frames[1], nil, "1300f30000352a1000001f0001002a000008000000", "完成位图行 42 写格（#2414）")
}

func TestTaskNoticeFramesMatchOfficial(t *testing.T) {
	frames, err := TaskAcceptNoticeFrames(officialPlayer, 1340)
	if err != nil || len(frames) != 3 {
		t.Fatalf("接取通知帧数 = %d err=%v, want 3", len(frames), err)
	}
	assertFrame(t, frames[0], nil, "1e030002d10000000209000000023c050000", "m209 [9,1340]（#1550）")
	assertFrame(t, frames[1], nil, "1e0400021800000008f30000352a1000000228000000060c0000007461736b5f61636365707400", "m24 task_accept（#1551）")
	assertFrame(t, frames[2], nil, "1e04000201000000020100000006050000003936323800060b0000007469746c655f3133343000", "m1 9628（#1552）")

	frames, err = TaskCompleteNoticeFrames(1340)
	if err != nil || len(frames) != 2 {
		t.Fatalf("完成通知帧数 = %d err=%v, want 2", len(frames), err)
	}
	assertFrame(t, frames[0], nil, "1e030002d10000000208000000023c050000", "m209 [8,1340]（#1986）")
	assertFrame(t, frames[1], nil, "1e04000201000000020100000006050000003936333100060b0000007469746c655f3133343000", "m1 9631（#1987）")
	action, err := TaskCompleteActionFrame(officialPlayer)
	assertFrame(t, action, err, "1e0400021800000008f30000352a1000000228000000060e0000007461736b5f636f6d706c65746500", "m24 task_complete（#2004）")

	frames, err = TaskStepOkFrames(officialPlayer, officialNPC)
	if err != nil || len(frames) != 3 {
		t.Fatalf("一步完成帧数 = %d err=%v, want 3", len(frames), err)
	}
	assertFrame(t, frames[0], nil, "1e0400021800000008f30000352a100000022800000006080000007461736b5f6f6b00", "m24 task_ok（#2104）")
	assertFrame(t, frames[1], nil, "1e03000212000000084d00003552000000060100000000", "m18 [NPC, \"\"]（#2105）")
	assertFrame(t, frames[2], nil, "1e01000208000000", "m8（#2106）")

	// 提交窗参数帧（#2404：1347 武学之道，Line 9）。
	form, err := TaskSubmitFormFrame(officialNPC, taskdata.Task{ID: 1347, Line: 9, CompleteDialog: "complete_1347;submit_1347"})
	assertFrame(t, form, err, "1e0a0002d10000000202000000084d00003552000000028309fb0b0243050000060e000000636f6d706c6574655f3133343700060c0000007375626d69745f313334370002000000000601000000000209000000", "m209 提交窗参数（#2404）")
}
