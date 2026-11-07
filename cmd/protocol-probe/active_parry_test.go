package main

import (
	"bytes"
	"encoding/binary"
	"testing"
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

// msg54 手动格挡（Java handleManualParry + fork handleBlockStateCustom
// 同源）：按下挂槽 11 + msg169 + InParry 属性帧；松开清槽；重复按下
// 静默；游离松开重推空态帧。
func TestBlockStateEngageAndRelease(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("格挡侠", 0)
	blockMsg := func(value int32) clientCustomMessage {
		return clientCustomMessage{Values: []clientCustomValue{
			{Type: 2, Int32: clientCustomBlockState},
			{Type: 2, Int32: value},
		}}
	}

	// 按下（0=engage）：官方 §3 同构三帧——SP=0 单帧(格挡清怒) +
	// msg169 + 0x10 格挡属性帧，槽 11 激活。
	if err := handleBlockStateCustom(conn, player, blockMsg(0), "test"); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 3 {
		t.Fatalf("engage frames=%d, want rage-clear+notice+state", len(frames))
	}
	spClear, notice, state := frames[0], frames[1], frames[2]
	if spClear[0] != 0x10 {
		t.Fatalf("rage clear frame=%x", spClear)
	}
	spNeedle := []byte{byte(propSP & 0xff), byte(propSP >> 8), 0, 0, 0, 0}
	if !bytes.Contains(spClear, spNeedle) {
		t.Fatalf("engage must clear SP to 0 (index %d): %x", propSP, spClear)
	}
	if notice[0] != 0x1E || binary.LittleEndian.Uint16(notice[1:]) != 9 {
		t.Fatalf("parry notice header=%x", notice)
	}
	if !bytes.Contains(notice, []byte("BuffInParry\x00")) {
		t.Fatalf("parry notice missing BuffInParry: %x", notice)
	}
	// tag7 宽串：UTF-16LE 角色名 + u16 NUL。
	wide := make([]byte, 0, len("格挡侠")*2+2)
	for _, r := range "格挡侠" {
		wide = binary.LittleEndian.AppendUint16(wide, uint16(r))
	}
	wide = binary.LittleEndian.AppendUint16(wide, 0)
	if !bytes.Contains(notice, wide) {
		t.Fatalf("parry notice missing wide name: %x", notice)
	}
	if state[0] != 0x10 || !bytes.Contains(state, []byte{0, 0}) {
		t.Fatalf("parry state frame=%x", state)
	}
	if !player.parryingSnapshot() {
		t.Fatal("engage did not set parrying")
	}

	// 重复按下：静默无新帧。
	if err := handleBlockStateCustom(conn, player, blockMsg(0), "test"); err != nil {
		t.Fatal(err)
	}
	if got := len(conn.Frames()); got != 3 {
		t.Fatalf("duplicate engage appended frames: %d", got)
	}

	// 松开（非 0）：清槽 + 两帧。
	if err := handleBlockStateCustom(conn, player, blockMsg(1), "test"); err != nil {
		t.Fatal(err)
	}
	if got := len(conn.Frames()); got != 5 {
		t.Fatalf("release frames=%d", got)
	}
	if player.parryingSnapshot() {
		t.Fatal("release kept parrying")
	}

	// 游离的重复松开：只重推一帧空态（客户端重同步）。
	if err := handleBlockStateCustom(conn, player, blockMsg(1), "test"); err != nil {
		t.Fatal(err)
	}
	if got := len(conn.Frames()); got != 6 {
		t.Fatalf("stray release frames=%d, want one resync frame", got)
	}
	if player.parryingSnapshot() {
		t.Fatal("stray release re-engaged parry")
	}
}

// msg292 预警切换（fork active_parry.go:186-250）：零参消息在普通(0)↔
// 预警(2)间翻转并回单帧 0x10 LogicState byte；出招中/格挡中/带额外
// 载荷拒绝。
func TestChangeWarningTogglesLogicState(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("预警侠", 0)
	warningMsg := clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: clientCustomChangeWarning}}}

	if handled, err := handleChangeWarningCustom(conn, player, warningMsg, "test"); err != nil || !handled {
		t.Fatalf("first toggle handled=%t err=%v", handled, err)
	}
	if got := player.logicStateSnapshot(); got != 2 {
		t.Fatalf("logic state=%d, want 2 after engage", got)
	}
	frames := conn.Frames()
	if len(frames) != 1 {
		t.Fatalf("warning frames=%d, want single LogicState frame", len(frames))
	}
	frame := frames[0]
	// 0x10 | isView=0 | obj | owner | count=1 | idx 463 | byte 2。
	if len(frame) != 15 || frame[0] != 0x10 || frame[1] != 0 ||
		binary.LittleEndian.Uint32(frame[2:]) != playerObjectID ||
		binary.LittleEndian.Uint32(frame[6:]) != playerOwnerID ||
		binary.LittleEndian.Uint16(frame[10:]) != 1 ||
		binary.LittleEndian.Uint16(frame[12:]) != propLogicState || frame[14] != 2 {
		t.Fatalf("LogicState frame=%x", frame)
	}

	// 翻回普通：同构单帧，byte 0。
	if handled, err := handleChangeWarningCustom(conn, player, warningMsg, "test"); err != nil || !handled {
		t.Fatalf("second toggle handled=%t err=%v", handled, err)
	}
	if got := player.logicStateSnapshot(); got != 0 {
		t.Fatalf("logic state=%d, want 0 after release", got)
	}
	frames = conn.Frames()
	if len(frames) != 2 || frames[1][14] != 0 {
		t.Fatalf("release frames=%x", frames)
	}
}

func TestChangeWarningRejectsExtraPayloadWhileCastingAndWhileParrying(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("预警侠", 0)

	// 带载荷：错误且不翻转。
	extra := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomChangeWarning},
		{Type: 2, Int32: 1},
	}}
	if handled, err := handleChangeWarningCustom(conn, player, extra, "test"); !handled || err == nil {
		t.Fatalf("extra payload handled=%t err=%v", handled, err)
	}
	if got := player.logicStateSnapshot(); got != 0 {
		t.Fatalf("extra payload flipped state to %d", got)
	}

	// 出招中拒绝。
	player.mu.Lock()
	player.currentSkillID = "CS_yhwq_hsqs05"
	player.mu.Unlock()
	warningMsg := clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: clientCustomChangeWarning}}}
	if handled, err := handleChangeWarningCustom(conn, player, warningMsg, "test"); err != nil || !handled {
		t.Fatalf("casting reject handled=%t err=%v", handled, err)
	}
	if got := player.logicStateSnapshot(); got != 0 {
		t.Fatalf("casting toggle flipped state to %d", got)
	}
	player.mu.Lock()
	player.currentSkillID = ""
	player.mu.Unlock()

	// 格挡中拒绝。
	if err := handleBlockStateCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomBlockState},
		{Type: 2, Int32: 0},
	}}, "test"); err != nil {
		t.Fatal(err)
	}
	if handled, err := handleChangeWarningCustom(conn, player, warningMsg, "test"); err != nil || !handled {
		t.Fatalf("parrying reject handled=%t err=%v", handled, err)
	}
	if got := player.logicStateSnapshot(); got != 0 {
		t.Fatalf("parrying toggle flipped state to %d", got)
	}
	if frames := conn.Frames(); len(frames) != 3 {
		t.Fatalf("warning rejections emitted frames: %x", frames)
	}
}
