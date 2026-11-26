package entergen

// guide_test.go 新手引导流帧编码对拍官服抓包：女号「初始账号登陆-完整
// 引导-女.pcapng」（8032，obj=350000f3 own=0000102a）与男号「初始账号
// 登陆-跳过引导-男.pcapng」（7114，obj=3d0000f3 own=00000611）。期望值是
// 官服帧的明文字节（EE 去转义 + XOR 之后），帧号是各自 S2C 流内的 0 基
// 序号（tools/pcapevents dumpall 的 #N）。

import (
	"bytes"
	"strings"
	"testing"
)

var officialMale = Identity{ObjectID: 0x3D0000F3, OwnerID: 0x00000611}

// officialCreate 建号.pcapng（端口 7111）男号 book2 的进场身份。
var officialCreate = Identity{ObjectID: 0x3B0000F3, OwnerID: 0x000000F8}

// spacedHex 解带空格的官服帧明文（同包的 mustHex 只收无空格形态，便于逐字段读）。
func spacedHex(t *testing.T, s string) []byte {
	t.Helper()
	return mustHex(t, strings.ReplaceAll(s, " ", ""))
}

func assertOfficial(t *testing.T, got []byte, err error, want string, what string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
	if !bytes.Equal(got, spacedHex(t, want)) {
		t.Fatalf("%s 与官服不一致:\n got  % x\n want % x", what, got, spacedHex(t, want))
	}
}

func assertOfficialFrames(t *testing.T, got [][]byte, err error, want []string, what string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
	if len(got) != len(want) {
		t.Fatalf("%s 帧数 = %d, want %d", what, len(got), len(want))
	}
	for i := range want {
		assertOfficial(t, got[i], nil, want[i], what)
	}
}

// TestTaskTrackRefreshFrameOfficial 任务追踪栏刷新 m390 [5, 0]：女号 #1508
// （首次进场就绪簇）、#13179（切图重入）与男号 #1604、#2258 同一帧。
func TestTaskTrackRefreshFrameOfficial(t *testing.T) {
	const want = "1e 03 00 02 86 01 00 00 02 05 00 00 00 02 00 00 00 00"
	frame, err := TaskTrackRefreshFrame()
	assertOfficial(t, frame, err, want, "m390")
}

// TestGameStepFrameOfficial 引导阶段 m414 ["gamestep", 主步, 5]：女号 #437
// （初出茅庐 4/5）、男号 #547（再战江湖 5/5）。
func TestGameStepFrameOfficial(t *testing.T) {
	single, err := GameStepFrame(4, 5)
	assertOfficial(t, single, err,
		"1e 04 00 02 9e 01 00 00 06 09 00 00 00 67 61 6d 65 73 74 65 70 00 02 04 00 00 00 02 05 00 00 00", "m414 4/5")
	skip, err := GameStepFrame(5, 5)
	assertOfficial(t, skip, err,
		"1e 04 00 02 9e 01 00 00 06 09 00 00 00 67 61 6d 65 73 74 65 70 00 02 05 00 00 00 02 05 00 00 00", "m414 5/5")
}

// TestGuidePropertyFramesOfficial 引导阶段的玩家单属性帧：出生阶段
// BornServerGameStep（女 #183=45、男 #293=55）、帮助步骤 HelpStepID（女
// #466 chengdu_00 / #1792 chengdu_02、男 #576 jimingyi_00）、输入锁
// CanntOperator（女 #881 锁 / #1532 解）、复活点 RelivePositon（男 #1631）、
// 入江湖 PowerLevel/YinLevel（男 #1946/#1947）。
func TestGuidePropertyFramesOfficial(t *testing.T) {
	born, err := BornGameStepFrame(officialPlayer, 45)
	assertOfficial(t, born, err, "10 00 f3 00 00 35 2a 10 00 00 01 00 1d 05 2d 00 00 00", "BornServerGameStep=45")
	born, err = BornGameStepFrame(officialMale, 55)
	assertOfficial(t, born, err, "10 00 f3 00 00 3d 11 06 00 00 01 00 1d 05 37 00 00 00", "BornServerGameStep=55")

	help, err := HelpStepFrame(officialPlayer, "chengdu_00")
	assertOfficial(t, help, err, "10 00 f3 00 00 35 2a 10 00 00 01 00 81 00 0b 00 00 00 63 68 65 6e 67 64 75 5f 30 30 00", "HelpStepID chengdu_00")
	help, err = HelpStepFrame(officialPlayer, "chengdu_02")
	assertOfficial(t, help, err, "10 00 f3 00 00 35 2a 10 00 00 01 00 81 00 0b 00 00 00 63 68 65 6e 67 64 75 5f 30 32 00", "HelpStepID chengdu_02")
	help, err = HelpStepFrame(officialMale, "jimingyi_00")
	assertOfficial(t, help, err, "10 00 f3 00 00 3d 11 06 00 00 01 00 81 00 0c 00 00 00 6a 69 6d 69 6e 67 79 69 5f 30 30 00", "HelpStepID jimingyi_00")

	lock, err := InputLockFrame(officialPlayer, true)
	assertOfficial(t, lock, err, "10 00 f3 00 00 35 2a 10 00 00 01 00 f9 00 01 00 00 00", "CanntOperator=1")
	lock, err = InputLockFrame(officialPlayer, false)
	assertOfficial(t, lock, err, "10 00 f3 00 00 35 2a 10 00 00 01 00 f9 00 00 00 00 00", "CanntOperator=0")

	relive, err := ReliveFrame(officialMale, "4,693.908,24.694,404.35,3.611,ui_pos_ReliveCJ100101")
	assertOfficial(t, relive, err,
		"10 00 f3 00 00 3d 11 06 00 00 01 00 c3 00 34 00 00 00 34 2c 36 39 33 2e 39 30 38 2c 32 34 2e 36 39 34 2c 34 30 34 2e 33 35 2c 33 2e 36 31 31 2c 75 69 5f 70 6f 73 5f 52 65 6c 69 76 65 43 4a 31 30 30 31 30 31 00",
		"RelivePositon")

	frames, err := EnterWorldFrames(officialMale)
	assertOfficialFrames(t, frames, err, []string{
		"10 00 f3 00 00 3d 11 06 00 00 01 00 d2 00 01 00 00 00", // PowerLevel=1
		"10 00 f3 00 00 3d 11 06 00 00 01 00 fb 00 01 00 00 00", // YinLevel=1
	}, "入江湖属性对")
}

// TestGuideSkipRewardFramesOfficial 再战江湖跳过簇的奖励帧（男号）：获得
// 物品记录 #1695 → 历练四帧 #1700..#1703 → 声望行（标志 2）#1720 → m98
// #1721 → 生活新手帮助 m142 ×4 + m98 #1723..#1727 → 链位置位后的 m543
// #1683。
func TestGuideSkipRewardFramesOfficial(t *testing.T) {
	give, err := GiveItemRowFrame(officialMale.ObjectID, officialMale.OwnerID, 0, "item_skipsp_001", 1)
	assertOfficial(t, give, err, "11 00 f3 00 00 3d 11 06 00 00 4a 01 00 00 01 00 10 00 00 00 69 74 65 6d 5f 73 6b 69 70 73 70 5f 30 30 31 00 01 00 00 00", "GiveItemRec 行")

	frames, err := LiveGrooveGainFrames(officialMale, 1, 5000, 5000)
	assertOfficialFrames(t, frames, err, []string{
		"1e 04 00 02 01 00 00 00 02 01 00 00 00 06 05 00 00 00 38 30 35 35 00 02 05 00 00 00",                                  // m1 [1,"8055",5]
		"1e 04 00 02 18 00 00 00 08 f3 00 00 3d 11 06 00 00 02 01 00 00 00 02 05 00 00 00",                                     // m24 [玩家,1,5]
		"1e 04 00 02 18 00 00 00 08 f3 00 00 3d 11 06 00 00 02 28 00 00 00 06 0c 00 00 00 68 75 6f 64 65 6c 69 6c 69 61 6e 00", // m24 [玩家,40,"huodelilian"]
		"10 00 f3 00 00 3d 11 06 00 00 01 00 26 03 88 13 00 00",                                                                // LiveGroove_1=5000
	}, "历练四帧")

	repute, err := ReputeRowFrameWithFlag(officialMale.ObjectID, officialMale.OwnerID, 0, "repute_jianghu", 700, 2)
	assertOfficial(t, repute, err, "11 00 f3 00 00 3d 11 06 00 00 2c 00 00 00 01 00 0f 00 00 00 72 65 70 75 74 65 5f 6a 69 61 6e 67 68 75 00 bc 02 00 00 02 00 00 00", "Repute_Record 行")

	open, err := FunctionOpenNoticeFrame(177)
	assertOfficial(t, open, err, "1e 02 00 02 62 00 00 00 02 b1 00 00 00", "m98 177")
	notice, err := TaskReputeNotice("repute_jianghu", 700)
	assertOfficial(t, notice, err,
		"1e 05 00 02 01 00 00 00 02 01 00 00 00 06 05 00 00 00 39 36 39 33 00 06 0f 00 00 00 72 65 70 75 74 65 5f 6a 69 61 6e 67 68 75 00 02 bc 02 00 00", "m1 9693")

	frames, err = LifeHelpUnlockFrames()
	assertOfficialFrames(t, frames, err, []string{
		"1e 02 00 02 8e 00 00 00 06 18 00 00 00 6c 69 66 65 5f 66 72 65 73 68 6d 61 6e 68 65 6c 70 5f 63 61 69 6a 69 00",
		"1e 02 00 02 8e 00 00 00 06 19 00 00 00 6c 69 66 65 5f 66 72 65 73 68 6d 61 6e 68 65 6c 70 5f 7a 68 69 7a 61 6f 00",
		"1e 02 00 02 8e 00 00 00 06 19 00 00 00 6c 69 66 65 5f 66 72 65 73 68 6d 61 6e 68 65 6c 70 5f 77 65 6e 68 75 61 00",
		"1e 02 00 02 8e 00 00 00 06 1a 00 00 00 6c 69 66 65 5f 66 72 65 73 68 6d 61 6e 68 65 6c 70 5f 73 68 69 6a 69 6e 67 00",
		"1e 02 00 02 62 00 00 00 02 c8 02 00 00",
	}, "生活新手帮助")

	mark, err := SingleStepMarkFrame()
	assertOfficial(t, mark, err, "1e 01 00 02 1f 02 00 00", "m543")
}

// TestTaskCompletedSetFramesSkipOfficial 跳过簇的完成位图逐格写：男号
// #1674..#1694（1339..1343 行 41、1344..1349 行 42、1597 行 49），每帧值
// 是该行累加后的掩码。建号种子不在官服样本里，这里用空已交集从零置位。
func TestTaskCompletedSetFramesSkipOfficial(t *testing.T) {
	chain := []uint32{1339, 1340, 1341, 1342, 1343, 1344, 1345, 1346, 1347, 1348, 1349, 1597}
	want := []string{
		"13 00 f3 00 00 3d 11 06 00 00 1f 00 01 00 29 00 00 00 00 00 08", // 1339 → 0x08000000
		"13 00 f3 00 00 3d 11 06 00 00 1f 00 01 00 29 00 00 00 00 00 18", // 1340 → 0x18000000
		"13 00 f3 00 00 3d 11 06 00 00 1f 00 01 00 29 00 00 00 00 00 38", // 1341 → 0x38000000
		"13 00 f3 00 00 3d 11 06 00 00 1f 00 01 00 29 00 00 00 00 00 78", // 1342 → 0x78000000
		"13 00 f3 00 00 3d 11 06 00 00 1f 00 01 00 29 00 00 00 00 00 f8", // 1343 → 0xF8000000
		"13 00 f3 00 00 3d 11 06 00 00 1f 00 01 00 2a 00 00 01 00 00 00", // 1344 → 0x00000001
		"13 00 f3 00 00 3d 11 06 00 00 1f 00 01 00 2a 00 00 03 00 00 00", // 1345 → 0x00000003
		"13 00 f3 00 00 3d 11 06 00 00 1f 00 01 00 2a 00 00 07 00 00 00", // 1346 → 0x00000007
		"13 00 f3 00 00 3d 11 06 00 00 1f 00 01 00 2a 00 00 0f 00 00 00", // 1347 → 0x0000000F
		"13 00 f3 00 00 3d 11 06 00 00 1f 00 01 00 2a 00 00 1f 00 00 00", // 1348 → 0x0000001F
		"13 00 f3 00 00 3d 11 06 00 00 1f 00 01 00 2a 00 00 3f 00 00 00", // 1349 → 0x0000003F
		"13 00 f3 00 00 3d 11 06 00 00 1f 00 01 00 31 00 00 00 00 00 20", // 1597 → 0x20000000
	}
	var completed []uint32
	var grid [][]byte
	for _, id := range chain {
		frames, err := TaskCompletedSetFrames(officialMale.ObjectID, officialMale.OwnerID, id, completed)
		if err != nil {
			t.Fatalf("%d: %v", id, err)
		}
		// 已交集为空时函数会先把 0 行到目标行的零值行补齐；只比对逐格写。
		grid = append(grid, frames[len(frames)-1])
		completed = append(completed, id)
	}
	assertOfficialFrames(t, grid, nil, want, "完成位图逐格写")
}

// TestFreshmanRecFrameOfficial 黑幕窗门控属性 FinishFreshmanRecStr 初值：
// 建号.pcapng 男号 #134（进场簇、0x0B 前）——idx225、串线型携裸字节
// 45×0x01 尾 09 00、无 NUL。引擎按此属性本地弹「初出茅庐/再战江湖」
// 选择窗（form_single_blackback），缺发＝待选态窗不弹。
func TestFreshmanRecFrameOfficial(t *testing.T) {
	frame, err := FreshmanRecFrame(officialCreate)
	assertOfficial(t, frame, err,
		"10 00 f3 00 00 3b f8 00 00 00 01 00 e1 00 2f 00 00 00 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 09 00",
		"FinishFreshmanRecStr 初值")
}

// TestGuideEntryFramesFreshmanGate 待选态在引导阶段帧之前补门控属性；
// 已定档/存量号不加。
func TestGuideEntryFramesFreshmanGate(t *testing.T) {
	ctx := Context{
		Identity:           officialCreate,
		GuideStep:          5, // 待选性别默认（男）
		GuideChoicePending: true,
		HelpStep:           "yanyuzhuang_00",
	}
	frames, err := GuideEntryFrames(ctx)
	if err != nil {
		t.Fatalf("GuideEntryFrames: %v", err)
	}
	if len(frames) != 4 {
		t.Fatalf("待选态帧数 = %d, want 4（门控+Born+m414+HelpStep）", len(frames))
	}
	want, err := FreshmanRecFrame(officialCreate)
	if err != nil || !bytes.Equal(frames[0], want) {
		t.Fatalf("待选态首帧不是 FinishFreshmanRecStr:\n got  % x\n want % x", frames[0], want)
	}

	ctx.GuideChoicePending = false
	frames, err = GuideEntryFrames(ctx)
	if err != nil {
		t.Fatalf("GuideEntryFrames: %v", err)
	}
	if len(frames) != 3 {
		t.Fatalf("已定档帧数 = %d, want 3", len(frames))
	}

	ctx.GuideStep, ctx.HelpStep = 0, ""
	frames, err = GuideEntryFrames(ctx)
	if err != nil {
		t.Fatalf("GuideEntryFrames: %v", err)
	}
	if len(frames) != 0 {
		t.Fatalf("存量号帧数 = %d, want 0", len(frames))
	}
}

// TestGuideChoiceInboundOfficial 客户端上行：黑幕窗点选 m972 [4]（女号
// C2S #53）/ [5]（男号 C2S #94）——Lua 数值到线是 double；新手帮助步骤
// m950 [0,"chengdu_02"]（女号 C2S #128）。帧是 0x0A 活动信封原字节。
func TestGuideChoiceInboundOfficial(t *testing.T) {
	for _, tc := range []struct {
		name, frame string
		wantStep    int32
	}{
		{"初出茅庐", "0a e0 07 ac 32 d6 4c 32 8a 1c 36 31 a3 55 91 f7 b2 27 00 00 00 02 00 02 cc 03 00 00 05 00 00 00 00 00 00 10 40", 4},
		{"再战江湖", "0a 5e 70 81 55 62 ca 9d 86 57 30 34 d0 64 1d b3 3c 6e 01 00 00 02 00 02 cc 03 00 00 05 00 00 00 00 00 00 14 40", 5},
	} {
		msg, ok, err := ParseActivityCustomMessage([]byte(spacedHex(t, tc.frame)))
		if err != nil || !ok {
			t.Fatalf("%s: 活动信封解帧失败 ok=%v err=%v", tc.name, ok, err)
		}
		if id, ok := msg.Values[0].ExactInt32(); !ok || id != 972 {
			t.Fatalf("%s: 消息号 = %d（want 972）", tc.name, id)
		}
		step, ok := msg.Values[1].ExactInt32()
		if !ok || step != tc.wantStep {
			t.Fatalf("%s: 步骤 = %d ok=%v（want %d；Lua 数值是 double）", tc.name, step, ok, tc.wantStep)
		}
	}

	msg, ok, err := ParseActivityCustomMessage([]byte(spacedHex(t,
		"0a d1 01 73 2b 28 b3 cb 61 bc 6f 33 8b 46 cf c8 3d 58 00 00 00 03 00 02 b6 03 00 00 02 00 00 00 00 06 0b 00 00 00 63 68 65 6e 67 64 75 5f 30 32 00")))
	if err != nil || !ok {
		t.Fatalf("m950 活动信封解帧失败 ok=%v err=%v", ok, err)
	}
	if id, _ := msg.Values[0].ExactInt32(); id != 950 {
		t.Fatalf("m950 消息号 = %d", id)
	}
	if len(msg.Values) < 3 || msg.Values[2].Text != "chengdu_02" {
		t.Fatalf("m950 步骤 = %+v（want chengdu_02）", msg.Values)
	}
}
