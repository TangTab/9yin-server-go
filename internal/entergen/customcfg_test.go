package entergen

// 自定义配置域玩家对象帧/整表重建方言/TVarList 解码的单测（规则源：
// 参考端 cmd/protocol-probe shortcut_records_test.go/custom_config_test.go）。

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// TestPlayerObjectConfigFramesSaved 存档态：键位域 0x10（198 尾空格/
// 223/201 尾空格）→ 快捷栏行；ShortcutsSet 基线不复权 starter 条目；
// 玩家对象域不含连接级 705（0x1E，705 唯一下行位=首传闸 re-assert，
// fssoloworld/custom.go）。
func TestPlayerObjectConfigFramesSaved(t *testing.T) {
	ctx := testContext(t)
	ctx.State.Customizing = "ABC"
	ctx.State.Keybind = "1,65;2,66;"
	ctx.State.OperateMode, ctx.State.OperateModeSet = 1, true
	ctx.State.NeiGongBind = "1;0;0;"
	ctx.State.Shortcuts = []rolestate.ShortcutRow{{Index: 3, Kind: "item", ID: "it_potion"}}
	ctx.State.ShortcutsSet = true
	frames, err := PlayerObjectConfigFrames(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 4 {
		t.Fatalf("saved cluster = %d frames, want 4 (198+223+201+1 行)", len(frames))
	}
	for _, frame := range frames {
		if frame[0] == 0x1E {
			t.Fatalf("player-object domain must not carry connection-level 705: %x", frame[:8])
		}
	}
	// 198 ShortcutKey：串 = 存档值 + " " 尾空格（0x0D 档内嵌无尾空格，
	// 两形态不可混用）。
	if frames[0][0] != 0x10 || binary.LittleEndian.Uint16(frames[0][10:]) != 1 {
		t.Fatalf("keybind frame head: %x", frames[0][:12])
	}
	if idx := binary.LittleEndian.Uint16(frames[0][12:]); idx != clientdata.MustPlayerProperty("ShortcutKey").Index {
		t.Fatalf("keybind property index = %d", idx)
	}
	if !bytes.HasSuffix(frames[0], []byte("1,65;2,66; \x00")) {
		t.Fatalf("keybind value missing trailing space: %x", frames[0])
	}
	// 223 OperateMode int32=1。
	if frames[1][0] != 0x10 || binary.LittleEndian.Uint16(frames[1][12:]) != 223 ||
		binary.LittleEndian.Uint32(frames[1][14:]) != 1 {
		t.Fatalf("operate mode frame: %x", frames[1])
	}
	// 201 NeiGongBind 尾空格。
	if !bytes.HasSuffix(frames[2], []byte("1;0;0; \x00")) {
		t.Fatalf("neigong bind value missing trailing space: %x", frames[2])
	}
	// 快捷栏：基线行原样，无 starter 复活。
	if !bytes.Contains(frames[3], []byte("it_potion\x00")) {
		t.Fatalf("shortcut rows: %x", frames[3])
	}
	if bytes.Contains(frames[3], []byte("zs_default_01")) {
		t.Fatal("starter resurrected over edited baseline")
	}
}

// TestShortcutRebuildFrames 整表重建方言：oldRows=3 倒序 DEL（行 2/1/0，
// 14B）+ 新表 2 行 ADD。
func TestShortcutRebuildFrames(t *testing.T) {
	frames, err := ShortcutRebuildFrames(1, 1, 3, []rolestate.ShortcutRow{
		{Index: 1, Kind: "skill", ID: "CS_wd_tjq01"},
		{Index: 2, Kind: "skill", ID: "CS_wd_tjq02"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 5 {
		t.Fatalf("rebuild = %d frames, want 5 (3 DEL + 2 ADD)", len(frames))
	}
	for row, frame := range frames[:3] {
		wantRow := uint16(2 - row)
		if frame[0] != 0x12 || len(frame) != 14 ||
			binary.LittleEndian.Uint32(frame[2:]) != 1 || binary.LittleEndian.Uint32(frame[6:]) != 1 ||
			binary.LittleEndian.Uint16(frame[10:]) != 93 || binary.LittleEndian.Uint16(frame[12:]) != wantRow {
			t.Fatalf("DEL frame %d: %x (want row %d)", row, frame, wantRow)
		}
	}
	if frames[3][0] != 0x11 || !bytes.Contains(frames[3], []byte("CS_wd_tjq01\x00")) ||
		frames[4][0] != 0x11 || !bytes.Contains(frames[4], []byte("CS_wd_tjq02\x00")) {
		t.Fatalf("ADD frames: %x %x", frames[3], frames[4])
	}
}

// TestParseCustomRoundTrip 编码 → 解码回环 + 0x0A 活动信封形态。
func TestParseCustomRoundTrip(t *testing.T) {
	frame, err := CustomMessageFrame(195, CustomInt(0), CustomString("1,65;2,66;"))
	if err != nil {
		t.Fatal(err)
	}
	message, err := ParseCustomMessage(frame)
	if err != nil {
		t.Fatal(err)
	}
	if message.Opcode != 0x1E || len(message.Values) != 3 {
		t.Fatalf("parsed %+v", message)
	}
	if id, ok := message.Values[0].ExactInt32(); !ok || id != 195 {
		t.Fatalf("message id %+v", message.Values[0])
	}
	if scheme, ok := message.Values[1].ExactInt32(); !ok || scheme != 0 {
		t.Fatalf("scheme %+v", message.Values[1])
	}
	if message.Values[2].Type != 6 || message.Values[2].Text != "1,65;2,66;" {
		t.Fatalf("bind %+v", message.Values[2])
	}
	// double 编码的枚举值（门派 1006/23 方言）按 ExactInt32 收。
	doubleFrame, err := CustomMessageFrame(1006, CustomFloat64(1))
	if err != nil {
		t.Fatal(err)
	}
	doubleMessage, err := ParseCustomMessage(doubleFrame)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := doubleMessage.Values[1].ExactInt32(); !ok || v != 1 {
		t.Fatalf("double enum: %+v", doubleMessage.Values[1])
	}
	// 0x0A 信封：21B 头 + u32 序列号 + 裸类型列表体（不带 0x1E opcode）。
	inner, err := CustomMessageFrame(107, CustomString("CFG"))
	if err != nil {
		t.Fatal(err)
	}
	envelope := make([]byte, 0, 21+len(inner))
	envelope = append(envelope, 0x0A)
	envelope = append(envelope, make([]byte, 16)...)
	envelope = binary.LittleEndian.AppendUint32(envelope, 7)
	envelope = append(envelope, inner[1:]...)
	activity, ok, err := ParseActivityCustomMessage(envelope)
	if err != nil || !ok {
		t.Fatalf("activity parse ok=%v err=%v", ok, err)
	}
	if activity.Values[0].Int32 != 107 || activity.Values[1].Text != "CFG" {
		t.Fatalf("activity values %+v", activity.Values)
	}
	// 非自定义 0x0A 流量：ok=false 不报错。
	if _, ok, err := ParseActivityCustomMessage([]byte{0x0A, 1, 2, 3}); err != nil || ok {
		t.Fatalf("non-custom activity ok=%v err=%v", ok, err)
	}
	// 畸形：尾随字节/空计数/非 NUL 串拒收。
	if _, err := ParseCustomMessage(append(frame, 0x00)); err == nil {
		t.Fatal("trailing bytes accepted")
	}
	if _, err := ParseCustomMessage([]byte{0x1E, 0, 0}); err == nil {
		t.Fatal("empty count accepted")
	}
}

// TestReadyFramesIncludesCustomCluster 就绪段集成：fresh 状态必含
// starter 打坐 0x11 行（缺发即「快捷栏空」表象）。
// 705 已自就绪簇撤下（5633d54；真服对照=登录段单发、上传零回应），
// 簇内只断言 starter 快捷栏行。
func TestReadyFramesIncludesCustomCluster(t *testing.T) {
	frames, err := ReadyFrames(testContext(t), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, frame := range frames {
		if frame[0] == 0x11 && binary.LittleEndian.Uint16(frame[10:]) == 93 &&
			bytes.Contains(frame, []byte("zs_default_01\x00")) {
			found = true
		}
	}
	if !found {
		t.Fatal("ready cluster missing starter shortcut row")
	}
}
