package entergen

// 背包物品视图行测试：0x19/0x18/0x19 三元组方言（预清→行→镜像页清位）、
// 分类页视图路由（bagViewOf）、行属性集名寻址（ConfigID/ItemType/Amount/
// MaxAmount/BindStatus/UniqueID——2433 表 idx 由 clientdata 解析）、
// 背包视图族官服口径（主页 cap=90+BaseCap=18）、Ready 编排视图族后逐行推送。

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// TestViewRemoveFrameShape 0x19：[19][u16 vid][u16 idx] 5B。
func TestViewRemoveFrameShape(t *testing.T) {
	frame := ViewRemoveFrame(121, 7)
	if len(frame) != 5 || frame[0] != 0x19 {
		t.Fatalf("len=%d op=%#x, want 5/0x19", len(frame), frame[0])
	}
	if vid, idx := binary.LittleEndian.Uint16(frame[1:]), binary.LittleEndian.Uint16(frame[3:]); vid != 121 || idx != 7 {
		t.Fatalf("vid/idx=%d/%d, want 121/7", vid, idx)
	}
}

// TestBagItemPushFramesTriple 三元组方言：预清 0x19 + 行 0x18 + 镜像页
// 清位 0x19 同槽位；分类页路由 ViewID 2→121/3→123/4→125/1→2，镜像
// 174/176/178/180 对应。
func TestBagItemPushFramesTriple(t *testing.T) {
	cases := map[int][2]uint16{1: {2, 174}, 2: {121, 176}, 3: {123, 178}, 4: {125, 180}}
	for viewID, want := range cases {
		wantView, wantMirror := want[0], want[1]
		item := rolestate.BagItem{Slot: 5, ConfigID: "yaopin_00100", ItemType: 8, Amount: 3, ViewID: viewID}
		frames, err := BagItemPushFrames(item)
		if err != nil {
			t.Fatal(err)
		}
		if len(frames) != 3 || frames[0][0] != 0x19 || frames[1][0] != 0x18 || frames[2][0] != 0x19 {
			t.Fatalf("viewID=%d frames=%d/%#x/%#x/%#x, want 0x19+0x18+0x19",
				viewID, len(frames), frames[0][0], frames[1][0], frames[2][0])
		}
		clear, row, mirrorClear := frames[0], frames[1], frames[2]
		if got := binary.LittleEndian.Uint16(clear[1:]); got != wantView {
			t.Fatalf("viewID=%d 清帧视图=%d, want %d", viewID, got, wantView)
		}
		if got := binary.LittleEndian.Uint16(row[1:]); got != wantView {
			t.Fatalf("viewID=%d 行帧视图=%d, want %d", viewID, got, wantView)
		}
		if got := binary.LittleEndian.Uint16(row[3:]); got != 5 {
			t.Fatalf("行 objectIndex=%d, want 5（槽位）", got)
		}
		if got := binary.LittleEndian.Uint16(mirrorClear[1:]); got != wantMirror {
			t.Fatalf("viewID=%d 镜像清帧视图=%d, want %d", viewID, got, wantMirror)
		}
		if got := binary.LittleEndian.Uint16(mirrorClear[3:]); got != 5 {
			t.Fatalf("镜像清帧槽位=%d, want 5", got)
		}
	}
}

// TestBagViewCreateFramesOfficial 背包视图族官服口径：8 主页 cap=90 且携
// BaseCap=18（初始账号登陆-完整引导-女.pcapng #1347..#1381：`15 <vid>
// 5a 00 01 00 ae 08 12 00 00 00`）、8 扩展页 cap=2、纸娃娃 cap=52。
func TestBagViewCreateFramesOfficial(t *testing.T) {
	frames, err := BagViewCreateFrames()
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 17 {
		t.Fatalf("背包视图族帧数=%d, want 17", len(frames))
	}
	baseCapIdx := clientdata.MustPlayerProperty("BaseCap").Index
	if baseCapIdx != 2222 {
		t.Fatalf("BaseCap idx=%d, want 2222（官服抓包 ae 08）", baseCapIdx)
	}
	mains := map[uint16]bool{}
	for _, frame := range frames {
		if frame[0] != 0x15 {
			t.Fatalf("非建视图帧 %#x", frame[0])
		}
		vid, cap, n := binary.LittleEndian.Uint16(frame[1:]),
			binary.LittleEndian.Uint16(frame[3:]), binary.LittleEndian.Uint16(frame[5:])
		switch vid {
		case equipDollView:
			if cap != 52 || n != 0 {
				t.Fatalf("纸娃娃 cap=%d n=%d, want 52/0", cap, n)
			}
		case 174, 176, 178, 180, 2, 121, 123, 125:
			mains[vid] = true
			if cap != 90 || n != 1 {
				t.Fatalf("主页 %d cap=%d n=%d, want 90/1", vid, cap, n)
			}
			if idx := binary.LittleEndian.Uint16(frame[7:]); idx != baseCapIdx {
				t.Fatalf("主页 %d 属性 idx=%d, want BaseCap %d", vid, idx, baseCapIdx)
			}
			if got := binary.LittleEndian.Uint32(frame[9:]); got != 18 {
				t.Fatalf("主页 %d BaseCap=%d, want 18", vid, got)
			}
		case 175, 177, 179, 181, 3, 122, 124, 126:
			if cap != 2 || n != 0 {
				t.Fatalf("扩展页 %d cap=%d n=%d, want 2/0", vid, cap, n)
			}
		default:
			t.Fatalf("背包族外的视图 %d", vid)
		}
	}
	if len(mains) != 8 {
		t.Fatalf("主页数=%d, want 8", len(mains))
	}
}

// TestBagItemViewRowProps 行属性集：六项通用属性名寻址齐、值随行。
func TestBagItemViewRowProps(t *testing.T) {
	item := rolestate.BagItem{Slot: 1, ConfigID: "wuqi_001", ItemType: 16, Amount: 1, ViewID: 2, UID: "aabb0011"}
	props := bagItemViewRow(item)
	byName := map[string]PropertyValue{}
	for _, p := range props {
		entry, ok := clientdata.PlayerProperty(propNameOf(t, p.Index))
		if !ok {
			t.Fatalf("idx %d 不在 2433 表", p.Index)
		}
		byName[entry.Name] = p.Value
	}
	if got := byName["ConfigID"]; got.Text != "wuqi_001" {
		t.Fatalf("ConfigID=%v", got)
	}
	if got := byName["Amount"]; got.Int32 != 1 {
		t.Fatalf("Amount=%v", got)
	}
	if got := byName["MaxAmount"]; got.Int32 != 1 {
		t.Fatalf("MaxAmount=%v", got)
	}
	if got := byName["BindStatus"]; got.Int32 != 0 {
		t.Fatalf("BindStatus=%v", got)
	}
	if got := byName["UniqueID"]; got.Text != "aabb0011" {
		t.Fatalf("UniqueID=%v", got)
	}

	// 无 UID 行合成 UniqueID（六项；2433 itemUID 同形 16 hex——空缺则
	// 客户端不渲染行，2026-11-23 真机「买了不显示」根因）。
	bare := bagItemViewRow(rolestate.BagItem{Slot: 2, ConfigID: "x", ViewID: 1})
	if len(bare) != 6 {
		t.Fatalf("无 UID 行属性数=%d, want 6", len(bare))
	}
	bareByName := map[string]PropertyValue{}
	for _, p := range bare {
		entry, _ := clientdata.PlayerProperty(propNameOf(t, p.Index))
		bareByName[entry.Name] = p.Value
	}
	synth := bareByName["UniqueID"].Text
	if len(synth) != 16 {
		t.Fatalf("合成 UID 形态=%q, want 16 hex", synth)
	}
	// 装备页行补装备域（EquipType 必发——真值缺失客户端不渲染）。
	equipProps := bagItemViewRow(rolestate.BagItem{Slot: 3, ConfigID: "blade", ViewID: 2, EquipType: "Blade"})
	equipByName := map[string]PropertyValue{}
	for _, p := range equipProps {
		entry, _ := clientdata.PlayerProperty(propNameOf(t, p.Index))
		equipByName[entry.Name] = p.Value
	}
	if equipByName["EquipType"].Text != "Blade" {
		t.Fatalf("EquipType=%v", equipByName["EquipType"])
	}
	if equipByName["Hardiness"].Int32 != 100 || equipByName["ColorLevel"].Int32 != 1 {
		t.Fatalf("装备缺省域异常: %+v", equipByName)
	}
}

// propNameOf idx → 表名（测试辅助；反向查 2433 表）。
func propNameOf(t *testing.T, index uint16) string {
	t.Helper()
	for _, name := range []string{"ConfigID", "ItemType", "Amount", "MaxAmount", "BindStatus", "UniqueID",
		"EquipType", "MaxBuildCount", "CurBuildCount", "RecoverFlag", "ColorLevel",
		"Hardiness", "MaxHardiness", "Level", "Photo"} {
		if entry, ok := clientdata.PlayerProperty(name); ok && entry.Index == index {
			return name
		}
	}
	t.Fatalf("idx %d 非物品行属性", index)
	return ""
}

// TestReadyFramesContainBagItems 空容器族之后逐行推送（空背包零帧不变量：
// 与既有 ReadyFrames 字节级快照兼容）。
func TestReadyFramesContainBagItems(t *testing.T) {
	ctx := testContext(t)
	bare, err := ReadyFrames(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	ctx.State.BagItems = []rolestate.BagItem{
		{Slot: 1, ConfigID: "yaopin_00100", ItemType: 8, Amount: 2, ViewID: 1},
		{Slot: 2, ConfigID: "wuqi_001", ItemType: 16, Amount: 1, ViewID: 2},
	}
	withItems, err := ReadyFrames(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(withItems) != len(bare)+6 { // 每件 0x19+0x18+0x19（镜像清位）三帧
		t.Fatalf("物品行未编排: bare=%d withItems=%d, want +6", len(bare), len(withItems))
	}
	// 编排位：背包视图族尾（0x15 vid=126）之后、武学视图（0x15 vid=40）
	// 之前；两件三元组 = 0x19/0x18/0x19 view2槽1+174槽1 +
	// 0x19/0x18/0x19 view121槽2+176槽2。
	lastEmpty, firstSkill, firstClear := -1, -1, -1
	for i, f := range withItems {
		switch f[0] {
		case 0x15:
			switch binary.LittleEndian.Uint16(f[1:]) {
			case 126:
				lastEmpty = i
			case ViewportSkill:
				if firstSkill < 0 {
					firstSkill = i
				}
			}
		case 0x19:
			if firstClear < 0 {
				firstClear = i
			}
		}
	}
	if lastEmpty < 0 || firstSkill < 0 || firstClear != lastEmpty+1 || firstClear+6 > firstSkill {
		t.Fatalf("物品行编排位错: lastEmpty=%d firstClear=%d firstSkill=%d", lastEmpty, firstClear, firstSkill)
	}
	want := []struct {
		op   byte
		vid  uint16
		slot uint16
	}{
		{0x19, 2, 1}, {0x18, 2, 1}, {0x19, 174, 1},
		{0x19, 121, 2}, {0x18, 121, 2}, {0x19, 176, 2},
	}
	for k, w := range want {
		f := withItems[firstClear+k]
		if f[0] != w.op || binary.LittleEndian.Uint16(f[1:]) != w.vid || binary.LittleEndian.Uint16(f[3:]) != w.slot {
			t.Fatalf("第 %d 帧形态错: op=%#x vid=%d slot=%d, want %#x/%d/%d",
				k, f[0], binary.LittleEndian.Uint16(f[1:]), binary.LittleEndian.Uint16(f[3:]), w.op, w.vid, w.slot)
		}
	}
}
