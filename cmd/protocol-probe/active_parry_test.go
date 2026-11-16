package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
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

// msg54 手动格挡（Java handleManualParry 同源）：按下回外服私服六件套序
// （含 BufferInfo4 无时长槽串）+m169；松开七帧清空；重复按下静默；
// 游离松开重推释放序。
func TestBlockStateEngageAndRelease(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("格挡侠", 0)
	blockMsg := func(value int32) clientCustomMessage {
		return clientCustomMessage{Values: []clientCustomValue{
			{Type: 2, Int32: clientCustomBlockState},
			{Type: 2, Int32: value},
		}}
	}

	// 按下（0=engage）：外服私服六件套序（a.pcapng 5.2s 定版）
	// ——MoveSpeed+RunSpeedAdd+InParry+ParryUpSpeedAdd+BufferInfo4 五单属性帧
	// + msg169 + InManualParry，共 7 帧。格挡不清怒。
	spBefore := player.skillSP()
	if err := handleBlockStateCustom(conn, player, blockMsg(0), "test"); err != nil {
		t.Fatal(err)
	}
	if got := player.skillSP(); got != spBefore {
		t.Fatalf("engage must not touch SP: %d -> %d", spBefore, got)
	}
	frames := conn.Frames()
	if len(frames) != 9 {
		t.Fatalf("engage frames=%d, want ack cluster + ParryValue + transient fight stance", len(frames))
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
	// 第 5 帧=BufferInfo4 外服私服无时长槽串（a.pcapng 5.2s
	// '11CE,701E5A4,3AD5FB,1,0,0' 同形：seg5 到期/seg6 剩余跳恒 0）。
	wantSerial := fmt.Sprintf("%X,%X,%X,1,0,0", parryBuffStaticData, playerObjectID, playerOwnerID)
	serial := frames[4]
	if serial[0] != 0x10 || binary.LittleEndian.Uint16(serial[12:]) != propParryTokenBufferInfo4 {
		t.Fatalf("parry serial frame=%x (want BufferInfo4 single-prop)", serial)
	}
	if !bytes.Contains(serial, []byte(wantSerial)) {
		t.Fatalf("parry serial=%x missing official no-expiry CSV %q", serial, wantSerial)
	}
	if !player.parryingSnapshot() {
		t.Fatal("engage did not set parrying")
	}

	// 重复按下：静默无新帧。
	if err := handleBlockStateCustom(conn, player, blockMsg(0), "test"); err != nil {
		t.Fatal(err)
	}
	if got := len(conn.Frames()); got != 9 {
		t.Fatalf("duplicate engage appended frames: %d", got)
	}

	// 松开（非 0）：外服私服七帧释放序（m169 移除+六单属性）。
	if err := handleBlockStateCustom(conn, player, blockMsg(1), "test"); err != nil {
		t.Fatal(err)
	}
	if got := len(conn.Frames()); got != 17 {
		t.Fatalf("release frames=%d, want seven-frame order + stance restore", got)
	}
	// 松开第 2 帧=BufferInfo4 清空串（a.pcapng 5.7s 'BufferInfo4='；
	// u32 长度 1 + NUL → 帧长 12+2+4+1=19，且不含 0x11CE 槽串）。
	frames = conn.Frames()
	clear := frames[10]
	if clear[0] != 0x10 || binary.LittleEndian.Uint16(clear[12:]) != propParryTokenBufferInfo4 {
		t.Fatalf("parry clear frame=%x (want BufferInfo4 single-prop)", clear)
	}
	if len(clear) != 19 || bytes.Contains(clear, []byte(wantSerial)) {
		t.Fatalf("parry clear frame=%x want empty string value", clear)
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
	if got := len(frames); got != 24 {
		t.Fatalf("stray release frames=%d, want official release order resync", got)
	}
	if frames[17][0] != 0x1E || !bytes.Contains(frames[17], []byte("BuffInParry\x00")) {
		t.Fatalf("stray resync missing 169 remove event: %x", frames[17])
	}
	if flag := binary.LittleEndian.Uint32(frames[17][9:13]); flag != 1 {
		t.Fatalf("stray resync 169 end flag=%d, want 1 (release)", flag)
	}
	if player.parryingSnapshot() {
		t.Fatal("stray release re-engaged parry")
	}
}

// 格挡槽串外服私服无时长：seg5 到期/seg6 剩余跳恒 0（a.pcapng 5.2s
// '11CE,701E5A4,3AD5FB,1,0,0' 定版；对照限时 buff '229D,…,8,
// C38EA546FE9,A'——seg5=1601 纪元毫秒、seg6 递减）。格挡 buf 事件驱动：
// 按下登记、松开/被打破摘除。两代同症教训都钉在这条断言上：旧 fork
// 方言 seg5 发 UnixMilli 被客户端按 1601 纪元解出 1657 年=即刻过期；
// 此前字面量 "26" 则令 0x11CE 登记解析失败——两者都表现为非战斗格挡
// 按下即被撤销（战斗态姿态由 LogicState 撑住掩盖）。
func TestParryBuffSerialNoExpiry(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("纪元侠", 0)
	if err := handleBlockStateCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomBlockState},
		{Type: 2, Int32: 0},
	}}, "test"); err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("%X,%X,%X,1,0,0", parryBuffStaticData, playerObjectID, playerOwnerID)
	frames := conn.Frames()
	if len(frames) != 9 || !bytes.Contains(frames[4], []byte(want)) {
		t.Fatalf("parry serial want official no-expiry CSV %q in BufferInfo4 frame, frames=%x", want, frames)
	}
}

// msg292 预警切换（fork active_parry.go:186-250）：零参消息在普通(0)↔
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

	// 格挡按下（HEAD 方言升临时战斗姿态 1）后：m292 落入非 0/2 忽略分支。
	if err := handleBlockStateCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomBlockState},
		{Type: 2, Int32: 0},
	}}, "test"); err != nil {
		t.Fatal(err)
	}
	base := len(conn.Frames())
	warningMsg := clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: clientCustomChangeWarning}}}
	if handled, err := handleChangeWarningCustom(conn, player, warningMsg, "test"); err != nil || !handled {
		t.Fatalf("fight-stance toggle handled=%t err=%v", handled, err)
	}
	if got := player.logicStateSnapshot(); got != 1 {
		t.Fatalf("fight stance must be held, got %d", got)
	}
	if got := len(conn.Frames()) - base; got != 0 {
		t.Fatalf("fight-stance toggle emitted %d frames, want ignore", got)
	}
}
