package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
	"time"
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

	// 按下（0=engage）：官服六件套序（a.pcapng 5.144 定版，2026-11-12 P2）
	// ——MoveSpeed+RunSpeedAdd+InParry+ParryUpSpeedAdd+BufferInfo4 五单属性帧
	// + msg169 + InManualParry，共 7 帧。槽 11 激活；格挡不清怒。
	spBefore := player.skillSP()
	if err := handleBlockStateCustom(conn, player, blockMsg(0), "test"); err != nil {
		t.Fatal(err)
	}
	if got := player.skillSP(); got != spBefore {
		t.Fatalf("engage must not touch SP: %d -> %d", spBefore, got)
	}
	frames := conn.Frames()
	if len(frames) != 7 {
		t.Fatalf("engage frames=%d, want official six-frame order (5 props + notice + InManualParry)", len(frames))
	}
	notice, state := frames[5], frames[0]
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
	if state[0] != 0x10 || binary.LittleEndian.Uint16(state[12:]) != propParryTokenMoveSpeed {
		t.Fatalf("parry state frame=%x (want MoveSpeed single-prop head of official order)", state)
	}
	if !player.parryingSnapshot() {
		t.Fatal("engage did not set parrying")
	}

	// 重复按下：静默无新帧。
	if err := handleBlockStateCustom(conn, player, blockMsg(0), "test"); err != nil {
		t.Fatal(err)
	}
	if got := len(conn.Frames()); got != 7 {
		t.Fatalf("duplicate engage appended frames: %d", got)
	}

	// 松开（非 0）：官服七帧释放序（m169 移除+六单属性）。
	if err := handleBlockStateCustom(conn, player, blockMsg(1), "test"); err != nil {
		t.Fatal(err)
	}
	if got := len(conn.Frames()); got != 14 {
		t.Fatalf("release frames=%d, want official seven-frame release order", got)
	}
	if player.parryingSnapshot() {
		t.Fatal("release kept parrying")
	}

	// 游离的重复松开：重推完整两帧对（169 摘除事件 + 0x10 空态）——
	// 客户端"已格挡"死锁只认 169 摘除事件，单属性帧不解锁（2026-11-12）。
	if err := handleBlockStateCustom(conn, player, blockMsg(1), "test"); err != nil {
		t.Fatal(err)
	}
	frames = conn.Frames()
	if got := len(frames); got != 21 {
		t.Fatalf("stray release frames=%d, want official release order resync", got)
	}
	if frames[14][0] != 0x1E || !bytes.Contains(frames[14], []byte("BuffInParry\x00")) {
		t.Fatalf("stray resync missing 169 remove event: %x", frames[14])
	}
	if flag := binary.LittleEndian.Uint32(frames[14][9:13]); flag != 1 {
		t.Fatalf("stray resync 169 end flag=%d, want 1 (release)", flag)
	}
	if player.parryingSnapshot() {
		t.Fatal("stray release re-engaged parry")
	}
}

// 格挡槽串到期段必须是 1601 纪元毫秒 HEX（55866 '11CE,11000001,7A5C0B,1,
// C38DC7DD957,1' 实证；UnixMilli 会被客户端按 1601 纪元解成 1657 年=
// 即刻过期，非战斗格挡按下即被客户端撤销）。
func TestParryBuffSerialEpoch1601(t *testing.T) {
	player := newPlayerActor("纪元侠", 0)
	engagedAt := time.Date(2026, 11, 12, 11, 30, 0, 0, time.UTC)
	serial := player.engageParryBuff(engagedAt)
	wantExpiry := engagedAt.Add(24 * time.Hour).UnixMilli() + windowsEpochDeltaMs
	want := fmt.Sprintf("11CE,%X,%X,1,%X,1", uint32(playerObjectID), uint32(playerOwnerID), wantExpiry)
	if serial != want {
		t.Fatalf("parry serial=%q want=%q", serial, want)
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
	// 至此连接上只有格挡按下产生的官服七帧序（预警拒绝不回帧）。
	if frames := conn.Frames(); len(frames) != 7 {
		t.Fatalf("warning rejections emitted frames: %x", frames)
	}
}
