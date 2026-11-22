package entergen

// 背包物品视图行测试：0x19/0x18 成对方言、分类页视图路由
//（bagViewOf）、行属性集名寻址（ConfigID/ItemType/Amount/MaxAmount/
// BindStatus/UniqueID——2433 表 idx 由 clientdata 解析）、Ready 编排
// 空容器族后逐行推送。

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

// TestBagItemPushFramesPair 成对方言：预清 0x19 + 行 0x18 同视图同槽位；
// 分类页路由 ViewID 2→121/3→123/4→125/1→2。
func TestBagItemPushFramesPair(t *testing.T) {
	cases := map[int]uint16{1: 2, 2: 121, 3: 123, 4: 125}
	for viewID, wantView := range cases {
		item := rolestate.BagItem{Slot: 5, ConfigID: "yaopin_00100", ItemType: 8, Amount: 3, ViewID: viewID}
		frames, err := BagItemPushFrames(item)
		if err != nil {
			t.Fatal(err)
		}
		if len(frames) != 2 || frames[0][0] != 0x19 || frames[1][0] != 0x18 {
			t.Fatalf("viewID=%d frames=%d/%#x/%#x, want 0x19+0x18", viewID, len(frames), frames[0][0], frames[1][0])
		}
		clear, row := frames[0], frames[1]
		if got := binary.LittleEndian.Uint16(clear[1:]); got != wantView {
			t.Fatalf("viewID=%d 清帧视图=%d, want %d", viewID, got, wantView)
		}
		if got := binary.LittleEndian.Uint16(row[1:]); got != wantView {
			t.Fatalf("viewID=%d 行帧视图=%d, want %d", viewID, got, wantView)
		}
		if got := binary.LittleEndian.Uint16(row[3:]); got != 5 {
			t.Fatalf("行 objectIndex=%d, want 5（槽位）", got)
		}
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

	// 无 UID 行省 UniqueID（五项）。
	bare := bagItemViewRow(rolestate.BagItem{Slot: 2, ConfigID: "x", ViewID: 1})
	if len(bare) != 5 {
		t.Fatalf("无 UID 行属性数=%d, want 5", len(bare))
	}
}

// propNameOf idx → 表名（测试辅助；反向查 2433 表）。
func propNameOf(t *testing.T, index uint16) string {
	t.Helper()
	for _, name := range []string{"ConfigID", "ItemType", "Amount", "MaxAmount", "BindStatus", "UniqueID"} {
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
	if len(withItems) != len(bare)+4 { // 每件 0x19+0x18 两帧
		t.Fatalf("物品行未编排: bare=%d withItems=%d, want +4", len(bare), len(withItems))
	}
	// 编排位：空容器族尾（0x15 vid=126）之后、武学视图（0x15 vid=40）
	// 之前；两件成对 = 0x19/0x18 view2槽1 + 0x19/0x18 view121槽2。
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
	if lastEmpty < 0 || firstSkill < 0 || firstClear != lastEmpty+1 || firstClear+4 > firstSkill {
		t.Fatalf("物品行编排位错: lastEmpty=%d firstClear=%d firstSkill=%d", lastEmpty, firstClear, firstSkill)
	}
	want := []struct {
		op   byte
		vid  uint16
		slot uint16
	}{
		{0x19, 2, 1}, {0x18, 2, 1}, {0x19, 121, 2}, {0x18, 121, 2},
	}
	for k, w := range want {
		f := withItems[firstClear+k]
		if f[0] != w.op || binary.LittleEndian.Uint16(f[1:]) != w.vid || binary.LittleEndian.Uint16(f[3:]) != w.slot {
			t.Fatalf("第 %d 帧形态错: op=%#x vid=%d slot=%d, want %#x/%d/%d",
				k, f[0], binary.LittleEndian.Uint16(f[1:]), binary.LittleEndian.Uint16(f[3:]), w.op, w.vid, w.slot)
		}
	}
}
