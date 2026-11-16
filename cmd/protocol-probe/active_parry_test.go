package main

import (
	"encoding/binary"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/entity"
)

func TestActiveParryAcceptsModernIntegralSubtype(t *testing.T) {
	player := newPlayerActor("招架测试", 0)
	handled, err := handleActiveParryCustom(player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomActiveParry},
		{Type: 5, Float64: 2},
		{Type: 2, Int32: 77},
	}}, "test")
	if err != nil || !handled {
		t.Fatalf("handle active parry handled=%t err=%v", handled, err)
	}
	request := player.activeParrySnapshot()
	if request.Subtype != 2 || request.Argument.Int32 != 77 || request.Argument.Type != 2 || request.At.IsZero() {
		t.Fatalf("stored request=%+v", request)
	}
}

func TestActiveParryRejectsFractionalSubtypeWithoutMutatingState(t *testing.T) {
	player := newPlayerActor("招架测试", 0)
	handled, err := handleActiveParryCustom(player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomActiveParry},
		{Type: 5, Float64: 1.5},
		{Type: 2, Int32: 77},
	}}, "test")
	if !handled || err == nil {
		t.Fatalf("fractional subtype handled=%t err=%v", handled, err)
	}
	if request := player.activeParrySnapshot(); !request.At.IsZero() {
		t.Fatalf("fractional subtype mutated player state: %+v", request)
	}
}

// msg54 手动格挡：应答=三属性权威帧（InParry/InManualParry/ParryValue，
// server2 manualParryStateUpdate 同构，2026-11-17）；按下/松开/游离
// 重复松开均回帧；重复按下静默；格挡不清怒；非战斗按下临时升
// LogicState=1 撑持架势、松开回写原值（2026-11-17 恢复）。
func TestBlockStateEngageAndRelease(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("格挡侠", 0)
	blockMsg := func(value int32) clientCustomMessage {
		return clientCustomMessage{Values: []clientCustomValue{
			{Type: 2, Int32: clientCustomBlockState},
			{Type: 2, Int32: value},
		}}
	}

	// 按下（0=engage）：三属性权威帧（active 门槛=parryValue>0）+
	// 463 临时升 1 撑持帧（非战斗架势立住的撑持方言）。
	spBefore := player.skillSP()
	if err := handleBlockStateCustom(conn, player, blockMsg(0), "test"); err != nil {
		t.Fatal(err)
	}
	if got := player.skillSP(); got != spBefore {
		t.Fatalf("engage must not touch SP: %d -> %d", spBefore, got)
	}
	frames := conn.Frames()
	if len(frames) != 2 {
		t.Fatalf("engage frames=%d, want authority frame + stance prop-up", len(frames))
	}
	// 第二帧=LogicState=1 单属性撑持帧。
	stance := frames[1]
	if stance[0] != 0x10 || binary.LittleEndian.Uint16(stance[10:12]) != 1 {
		t.Fatalf("stance prop-up frame head=%x", stance)
	}
	if idx := binary.LittleEndian.Uint16(stance[12:14]); idx != clientdata.MustPlayerProperty("LogicState").Index {
		t.Fatalf("stance prop-up idx=%d, want LogicState", idx)
	}
	if got := stance[14]; got != entity.LogicStateFighting {
		t.Fatalf("stance prop-up LogicState=%d, want fight(1)", got)
	}
	if got := player.logicStateSnapshot(); got != entity.LogicStateFighting {
		t.Fatalf("engage stance=%d, want temporary fight(1)", got)
	}
	frame := frames[0]
	if frame[0] != 0x10 || binary.LittleEndian.Uint32(frame[2:]) != playerObjectID {
		t.Fatalf("parry authority frame head=%x", frame)
	}
	if cnt := binary.LittleEndian.Uint16(frame[10:12]); cnt != 3 {
		t.Fatalf("parry authority prop count=%d, want 3", cnt)
	}
	// 三属性序=InParry/InManualParry(=1)/ParryValue(>0)。
	off := 12
	for _, want := range []struct {
		idx  uint16
		name string
		val  int32
	}{
		{propInParry, "InParry", 1},
		{propInManualParry, "InManualParry", 1},
	} {
		idx := binary.LittleEndian.Uint16(frame[off : off+2])
		off += 2
		if idx != want.idx {
			t.Fatalf("prop order %s: idx=%d want=%d", want.name, idx, want.idx)
		}
		if got := binary.LittleEndian.Uint32(frame[off : off+4]); int32(got) != want.val {
			t.Fatalf("%s=%d, want %d", want.name, int32(got), want.val)
		}
		off += 4
	}
	if idx := binary.LittleEndian.Uint16(frame[off : off+2]); idx != propParryValue {
		t.Fatalf("ParryValue idx=%d", idx)
	}
	off += 2
	if got := int32(binary.LittleEndian.Uint32(frame[off : off+4])); got <= 0 {
		t.Fatalf("ParryValue=%d, want >0 (activate threshold)", got)
	}
	if !player.parryingSnapshot() {
		t.Fatal("engage did not set parrying")
	}

	// 重复按下：静默无新帧。
	if err := handleBlockStateCustom(conn, player, blockMsg(0), "test"); err != nil {
		t.Fatal(err)
	}
	if got := len(conn.Frames()); got != 2 {
		t.Fatalf("duplicate engage appended frames: %d", got)
	}

	// 松开（1）：空态三属性 + 姿态回写帧（0=格挡前原值）。
	if err := handleBlockStateCustom(conn, player, blockMsg(1), "test"); err != nil {
		t.Fatal(err)
	}
	frames = conn.Frames()
	if len(frames) != 4 {
		t.Fatalf("release frames=%d, want authority + stance restore", len(frames))
	}
	rel := frames[2]
	off = 12
	for _, name := range []string{"InParry", "InManualParry"} {
		off += 2
		if got := int32(binary.LittleEndian.Uint32(rel[off : off+4])); got != 0 {
			t.Fatalf("release %s=%d, want 0", name, got)
		}
		off += 4
	}
	restore := frames[3]
	if idx := binary.LittleEndian.Uint16(restore[12:14]); idx != clientdata.MustPlayerProperty("LogicState").Index {
		t.Fatalf("stance restore idx=%d, want LogicState", idx)
	}
	if got := restore[14]; got != entity.LogicStateNormal {
		t.Fatalf("stance restore LogicState=%d, want pre-block normal(0)", got)
	}
	if got := player.logicStateSnapshot(); got != entity.LogicStateNormal {
		t.Fatalf("release stance=%d, want restored normal(0)", got)
	}
	if player.parryingSnapshot() {
		t.Fatal("release kept parrying")
	}

	// 游离的重复松开：也回权威帧（server2「echo duplicate release」），
	// 不再重复回写姿态（还原位已被 take 清零）。
	if err := handleBlockStateCustom(conn, player, blockMsg(1), "test"); err != nil {
		t.Fatal(err)
	}
	if got := len(conn.Frames()); got != 5 {
		t.Fatalf("stray release frames=%d, want echo", got)
	}
	if player.parryingSnapshot() {
		t.Fatal("stray release re-engaged parry")
	}
}

// msg292 预警切换（Z 拔/收刀）fork 方言（server2 权威实现，2026-11-17
// 七轮排查真机定版）：受理翻 LogicState 0↔2 并回 vitalUpdate 全量帧
//（面板带 LogicState——FxGameLogic 据此播拔刀挂剑）；战斗等非 0/2 态
// 忽略；带载荷静默忽略。
func TestChangeWarningTogglesLogicState(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("预警侠", 0)
	warningMsg := clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: clientCustomChangeWarning}}}

	if handled, err := handleChangeWarningCustom(conn, player, warningMsg, "test"); err != nil || !handled {
		t.Fatalf("first toggle handled=%t err=%v", handled, err)
	}
	if got := player.logicStateSnapshot(); got != entity.LogicStateWarning {
		t.Fatalf("logic state=%d, want warning after first Z", got)
	}
	frames := conn.Frames()
	if len(frames) != 1 || frames[0][0] != 0x10 {
		t.Fatalf("warning frames=%x, want single vitalUpdate 0x10", frames)
	}
	// vitalUpdate 面板必须带 LogicState=2（fork 客户端据此执行拔刀）。
	// idx 按现行表解析（2494 表=465；旧硬编 463 在该表位是别的属性——
	// idx 错位正是旧单属性帧方言失灵的一半真相）。
	frame := frames[0]
	found := false
	wantIdx := clientdata.MustPlayerProperty("LogicState").Index
	off := 12
	cnt := int(binary.LittleEndian.Uint16(frame[10:12]))
	for i := 0; i < cnt; i++ {
		if off+2 > len(frame) {
			break
		}
		idx := binary.LittleEndian.Uint16(frame[off : off+2])
		off += 2
		if idx == wantIdx {
			found = frame[off] == entity.LogicStateWarning
			break
		}
		off += 4 // 面板其余列宽不逐一解，仅扫描 LogicState 位
	}
	if !found {
		t.Fatalf("vitalUpdate missing LogicState=%d: %x", entity.LogicStateWarning, frame)
	}

	// 翻回收刀：同构 vitalUpdate 单帧。
	if handled, err := handleChangeWarningCustom(conn, player, warningMsg, "test"); err != nil || !handled {
		t.Fatalf("second toggle handled=%t err=%v", handled, err)
	}
	if got := player.logicStateSnapshot(); got != entity.LogicStateNormal {
		t.Fatalf("logic state=%d, want normal after second Z", got)
	}
	if frames := conn.Frames(); len(frames) != 2 {
		t.Fatalf("release frames=%x", frames)
	}
}

func TestChangeWarningIgnoresExtraPayloadAndFightStance(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("预警侠", 0)

	// 带载荷：静默忽略零帧不翻转（server2 方言，不回错不断链）。
	extra := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomChangeWarning},
		{Type: 2, Int32: 1},
	}}
	if handled, err := handleChangeWarningCustom(conn, player, extra, "test"); err != nil || !handled {
		t.Fatalf("extra payload handled=%t err=%v", handled, err)
	}
	if got := player.logicStateSnapshot(); got != 0 {
		t.Fatalf("extra payload flipped state to %d", got)
	}
	if frames := conn.Frames(); len(frames) != 0 {
		t.Fatalf("extra payload emitted frames=%x", frames)
	}

	// 格挡撑持期间姿态=临时战斗(1)：m292 被忽略（Z 持剑所有权不被
	// 侵占，松开回写原值后 Z 才可用——2026-11-16 定版策略恢复）。
	if err := handleBlockStateCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomBlockState},
		{Type: 2, Int32: 0},
	}}, "test"); err != nil {
		t.Fatal(err)
	}
	if got := player.logicStateSnapshot(); got != entity.LogicStateFighting {
		t.Fatalf("parrying stance=%d, want temporary fight(1)", got)
	}
	base := len(conn.Frames())
	warningMsg := clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: clientCustomChangeWarning}}}
	if handled, err := handleChangeWarningCustom(conn, player, warningMsg, "test"); err != nil || !handled {
		t.Fatalf("parrying toggle handled=%t err=%v", handled, err)
	}
	if got := player.logicStateSnapshot(); got != entity.LogicStateFighting {
		t.Fatalf("Z during prop-up must be ignored, stance=%d", got)
	}
	if got := len(conn.Frames()) - base; got != 0 {
		t.Fatalf("Z during prop-up emitted frames=%d, want silence", got)
	}
	// 松开格挡：姿态回写 0，Z 重新可用。
	if err := handleBlockStateCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomBlockState},
		{Type: 2, Int32: 1},
	}}, "test"); err != nil {
		t.Fatal(err)
	}
	if got := player.logicStateSnapshot(); got != entity.LogicStateNormal {
		t.Fatalf("release stance=%d, want restored normal(0)", got)
	}
	if handled, err := handleChangeWarningCustom(conn, player, warningMsg, "test"); err != nil || !handled {
		t.Fatalf("post-release toggle handled=%t err=%v", handled, err)
	}
	if got := player.logicStateSnapshot(); got != entity.LogicStateWarning {
		t.Fatalf("post-release stance=%d, want warning(2)", got)
	}
}
