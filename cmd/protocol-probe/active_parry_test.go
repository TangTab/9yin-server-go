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

	// 按下（0=engage）：msg169 + 0x10 两帧，槽 11 激活。
	if err := handleBlockStateCustom(conn, player, blockMsg(0), "test"); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 2 {
		t.Fatalf("engage frames=%d, want notice+state", len(frames))
	}
	notice, state := frames[0], frames[1]
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
	if got := len(conn.Frames()); got != 2 {
		t.Fatalf("duplicate engage appended frames: %d", got)
	}

	// 松开（非 0）：清槽 + 两帧。
	if err := handleBlockStateCustom(conn, player, blockMsg(1), "test"); err != nil {
		t.Fatal(err)
	}
	if got := len(conn.Frames()); got != 4 {
		t.Fatalf("release frames=%d", got)
	}
	if player.parryingSnapshot() {
		t.Fatal("release kept parrying")
	}

	// 游离的重复松开：只重推一帧空态（客户端重同步）。
	if err := handleBlockStateCustom(conn, player, blockMsg(1), "test"); err != nil {
		t.Fatal(err)
	}
	if got := len(conn.Frames()); got != 5 {
		t.Fatalf("stray release frames=%d, want one resync frame", got)
	}
	if player.parryingSnapshot() {
		t.Fatal("stray release re-engaged parry")
	}
}
