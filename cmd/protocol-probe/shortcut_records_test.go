package main

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/local/9yin-go-server/internal/entity"
)

func shortcutCustom(messageID int32, values ...clientCustomValue) clientCustomMessage {
	return clientCustomMessage{
		Opcode: 0x0A,
		Values: append([]clientCustomValue{{Type: 2, Int32: messageID}}, values...),
	}
}

func TestSetShortcutAddsAndReplacesRecordRow(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	set := shortcutCustom(clientCustomSetShortcut,
		clientCustomValue{Type: 2, Int32: 3},
		clientCustomValue{Type: 6, Text: "skill"},
		clientCustomValue{Type: 6, Text: "CS_yhwq_hsqs01"},
	)
	handled, err := handleShortcutCustom(conn, player, set, "test", nil, 0)
	if err != nil || !handled || len(conn.frames) != 1 {
		t.Fatalf("first set handled=%t err=%v frames=%d", handled, err, len(conn.frames))
	}
	add := conn.frames[0]
	if add[0] != 0x11 || binary.LittleEndian.Uint16(add[10:]) != recordShortcut ||
		binary.LittleEndian.Uint32(add[16:]) != 3 || !strings.Contains(string(add), "CS_yhwq_hsqs01") {
		t.Fatalf("shortcut add=%x", add)
	}

	set.Values[3].Text = "CS_yhwq_hsqs02"
	handled, err = handleShortcutCustom(conn, player, set, "test", nil, 0)
	if err != nil || !handled || len(conn.frames) != 3 {
		t.Fatalf("replace handled=%t err=%v frames=%d", handled, err, len(conn.frames))
	}
	if conn.frames[1][0] != 0x12 || binary.LittleEndian.Uint16(conn.frames[1][10:]) != recordShortcut ||
		binary.LittleEndian.Uint16(conn.frames[1][12:]) != 0 {
		t.Fatalf("shortcut delete=%x", conn.frames[1])
	}
	if conn.frames[2][0] != 0x11 || !strings.Contains(string(conn.frames[2]), "CS_yhwq_hsqs02") {
		t.Fatalf("replacement add=%x", conn.frames[2])
	}
}

// 客户端 C2S 201 传的是可见栏位索引而非记录行号（shortcut_game.lua
// remove_shortcut 全部调用点实证）。应答走整表重建方言：编辑前的
// 全部行倒序 0x12 清空，再按内存序 0x11 重放（见 rebuildShortcutRecord）。
func TestRemoveShortcutUsesVisibleIndex(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	player.setShortcut(8, "skill", "CS_yhwq_hsqs01")
	player.setShortcut(9, "skill", "CS_yhwq_hsqs02")

	remove := shortcutCustom(clientCustomRemoveShortcut, clientCustomValue{Type: 2, Int32: 9})
	handled, err := handleShortcutCustom(conn, player, remove, "test", nil, 0)
	if err != nil || !handled || len(conn.frames) != 3 {
		t.Fatalf("remove handled=%t err=%v frames=%d", handled, err, len(conn.frames))
	}
	// 倒序清空两行：先 row1（被删的 index=9）再 row0，然后重建剩余行。
	if got := binary.LittleEndian.Uint16(conn.frames[0][12:]); conn.frames[0][0] != 0x12 || got != 1 {
		t.Fatalf("sweep first frame=%x row=%d", conn.frames[0], got)
	}
	if got := binary.LittleEndian.Uint16(conn.frames[1][12:]); conn.frames[1][0] != 0x12 || got != 0 {
		t.Fatalf("sweep second frame=%x row=%d", conn.frames[1], got)
	}
	if conn.frames[2][0] != 0x11 || !strings.Contains(string(conn.frames[2]), "CS_yhwq_hsqs01") {
		t.Fatalf("rebuild frame=%x", conn.frames[2])
	}
	snapshot := player.shortcutSnapshot()
	if len(snapshot) != 1 || snapshot[0].index != 8 {
		t.Fatalf("remaining shortcuts=%+v", snapshot)
	}

	// 删最后一行：清 1 行 + 0 行重建。
	removeFirst := shortcutCustom(clientCustomRemoveShortcut, clientCustomValue{Type: 2, Int32: 8})
	handled, err = handleShortcutCustom(conn, player, removeFirst, "test", nil, 0)
	if err != nil || !handled || len(conn.frames) != 4 {
		t.Fatalf("remove-first handled=%t err=%v frames=%d", handled, err, len(conn.frames))
	}
	if got := binary.LittleEndian.Uint16(conn.frames[3][12:]); conn.frames[3][0] != 0x12 || got != 0 {
		t.Fatalf("remove-first frame=%x row=%d", conn.frames[3], got)
	}
	if snapshot := player.shortcutSnapshot(); len(snapshot) != 0 {
		t.Fatalf("all removed, remaining=%+v", snapshot)
	}

	// 不存在的索引静默忽略（零帧）。
	removeMissing := shortcutCustom(clientCustomRemoveShortcut, clientCustomValue{Type: 2, Int32: 40})
	handled, err = handleShortcutCustom(conn, player, removeMissing, "test", nil, 0)
	if err != nil || !handled || len(conn.frames) != 4 {
		t.Fatalf("remove-missing handled=%t err=%v frames=%d", handled, err, len(conn.frames))
	}
}

func TestGrantShortcutRowsRestoresCurrentRows(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	player.setShortcut(1, "skill", "CS_yhwq_hsqs01")
	player.setShortcut(2, "skill", "CS_yhwq_hsqs02")
	if err := grantShortcutRows(conn, player); err != nil {
		t.Fatal(err)
	}
	if len(conn.frames) != 2 || conn.frames[0][0] != 0x11 || conn.frames[1][0] != 0x11 {
		t.Fatalf("restore frames=%x", conn.frames)
	}
}

// ensureSitcrossShortcut 只补内存不发帧：0x11 行由 grantShortcutRows
// 统一重放，避免同一行重复下发。
func TestEnsureSitcrossShortcutAddsLearnedNativeSkillWithoutOverwritingSlot(t *testing.T) {
	player := newPlayerActor("测试", 0)
	if !ensureSitcrossShortcut(player) {
		t.Fatal("first ensure should add the sitcross row")
	}
	snapshot := player.shortcutSnapshot()
	if len(snapshot) != 1 || snapshot[0].index != sitcrossShortcutIndex || snapshot[0].id != sitcrossShortcutID {
		t.Fatalf("sitcross shortcuts=%+v", snapshot)
	}
	if ensureSitcrossShortcut(player) {
		t.Fatal("second ensure must be idempotent")
	}
	if snapshot := player.shortcutSnapshot(); len(snapshot) != 1 {
		t.Fatalf("duplicate sitcross rows=%+v", snapshot)
	}

	other := newPlayerActor("占位", 0)
	other.setShortcut(sitcrossShortcutIndex, "skill", "CS_yhwq_hsqs01")
	if ensureSitcrossShortcut(other) {
		t.Fatal("occupied slot must not be overwritten")
	}
}

func TestGrantHeartBuddhaPalmShortcutsFillSecondNumericPage(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	player.setShortcut(heartBuddhaPalmShortcutBaseIndex, "skill", "CS_yhwq_hsqs01")
	granted, err := grantHeartBuddhaPalmShortcuts(conn, player)
	if err != nil || granted != len(heartBuddhaPalmShortcutIDs) || len(conn.frames) != len(heartBuddhaPalmShortcutIDs)+1 {
		t.Fatalf("grant granted=%d err=%v frames=%d", granted, err, len(conn.frames))
	}
	if conn.frames[0][0] != 0x12 || conn.frames[1][0] != 0x11 || !strings.Contains(string(conn.frames[1]), "CS_jh_xfz01") {
		t.Fatalf("heart buddha shortcut frames=%x", conn.frames)
	}
	shortcuts := player.shortcutSnapshot()
	if len(shortcuts) != len(heartBuddhaPalmShortcutIDs) {
		t.Fatalf("shortcuts=%+v", shortcuts)
	}
	for offset, shortcut := range shortcuts {
		if shortcut.index != heartBuddhaPalmShortcutBaseIndex+int32(offset) || shortcut.id != heartBuddhaPalmShortcutIDs[offset] {
			t.Fatalf("shortcut[%d]=%+v", offset, shortcut)
		}
	}
}

func TestGrantTaiJiQuanGuPuShortcutsFillThirdNumericPage(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	granted, err := grantTaiJiQuanGuPuShortcuts(conn, player)
	if err != nil || granted != len(taiJiQuanGuPuShortcutIDs) || len(conn.frames) != len(taiJiQuanGuPuShortcutIDs) {
		t.Fatalf("grant granted=%d err=%v frames=%d", granted, err, len(conn.frames))
	}
	shortcuts := player.shortcutSnapshot()
	for offset, shortcut := range shortcuts {
		if shortcut.index != taiJiQuanGuPuShortcutBaseIndex+int32(offset) ||
			shortcut.kind != taiJiQuanGuPuShortcutKind ||
			shortcut.id != taiJiQuanGuPuShortcutIDs[offset] {
			t.Fatalf("shortcut[%d]=%+v", offset, shortcut)
		}
	}
}

func TestSitcrossUsesCapturedStartArgumentAndNormalPropertyUpdate(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	start := shortcutCustom(clientCustomSitCross, clientCustomValue{Type: 2, Int32: 1})
	handled, err := handleSitcrossCustom(conn, player, start, "test")
	if err != nil || !handled {
		t.Fatalf("start handled=%t err=%v", handled, err)
	}
	if got := player.stateSnapshot().LogicState; got != entity.LogicStateSitcross {
		t.Fatalf("LogicState=%d, want sitcross=%d", got, entity.LogicStateSitcross)
	}
	// 官服进出簇形态：0x10 属性帧 + msg24 c1=40 "dazuo_old" 动作
	// 飘字同拍成对（FsGame 官服抓包全量成对实证）。
	frames := conn.Frames()
	if len(frames) != 2 || frames[0][0] != 0x10 {
		t.Fatalf("expected 0x10 property + dazuo_old action, got %d frames", len(frames))
	}
	assertSitcrossActionFrame(t, frames[1], 40)

	// 按住按键会产生重复的 240,1 数据包。此时必须保持在状态
	// 102，而不是表现为切换开关。
	handled, err = handleSitcrossCustom(conn, player, start, "test")
	if err != nil || !handled || len(conn.Frames()) != 2 {
		t.Fatalf("duplicate handled=%t err=%v frames=%d", handled, err, len(conn.Frames()))
	}

	stop := shortcutCustom(clientCustomSitCross, clientCustomValue{Type: 2, Int32: 0})
	handled, err = handleSitcrossCustom(conn, player, stop, "test")
	if err != nil || !handled || player.stateSnapshot().LogicState != entity.LogicStateNormal || len(conn.Frames()) != 4 {
		t.Fatalf("stop handled=%t err=%v state=%d frames=%d", handled, err, player.stateSnapshot().LogicState, len(conn.Frames()))
	}
	assertSitcrossActionFrame(t, conn.Frames()[3], 41)

	// 真实客户端抓包：敌方打断会发出 240,2 而不是旧版的
	// 取消值 0。此时必须停止打坐，
	// 而不能关闭连接。
	handled, err = handleSitcrossCustom(conn, player, start, "test")
	if err != nil || !handled || player.stateSnapshot().LogicState != entity.LogicStateSitcross {
		t.Fatalf("second start handled=%t err=%v state=%d", handled, err, player.stateSnapshot().LogicState)
	}
	interrupted := shortcutCustom(clientCustomSitCross, clientCustomValue{Type: 2, Int32: 2})
	handled, err = handleSitcrossCustom(conn, player, interrupted, "test")
	if err != nil || !handled || player.stateSnapshot().LogicState != entity.LogicStateNormal {
		t.Fatalf("interruption handled=%t err=%v state=%d", handled, err, player.stateSnapshot().LogicState)
	}
}

func TestSitcrossTickRestoresAuthoritativeHPAndMP(t *testing.T) {
	player := newPlayerActor("测试", 0)
	before := player.stateSnapshot()
	player.applyDamage(100)
	player.spendMP(100)
	player.setLogicState(entity.LogicStateSitcross)
	settlement := player.sitcrossTick()
	if !settlement.Changed() {
		t.Fatal("sitcross tick did not change damaged actor")
	}
	state := player.stateSnapshot()
	wantHP := minInt32(before.MaxHP, before.HP-100+sitcrossRecoveryPerSecond(before.MaxHP))
	wantMP := minInt32(before.MaxMP, before.MP-100+sitcrossRecoveryPerSecond(before.MaxMP))
	if state.HP != wantHP || state.MP != wantMP {
		t.Fatalf("sitcross state HP=%d MP=%d, want %d/%d", state.HP, state.MP, wantHP, wantMP)
	}
	wantApplied := wantHP - (before.HP - 100)
	wantWasted := sitcrossRecoveryPerSecond(before.MaxHP) - wantApplied
	wantMPApplied := wantMP - (before.MP - 100)
	wantMPWasted := sitcrossRecoveryPerSecond(before.MaxMP) - wantMPApplied
	if settlement.HPApplied != wantApplied || settlement.HPWasted != wantWasted ||
		settlement.MPApplied != wantMPApplied || settlement.MPWasted != wantMPWasted {
		t.Fatalf("settlement=%+v, want HP=%d/%d MP=%d/%d", settlement, wantApplied, wantWasted, wantMPApplied, wantMPWasted)
	}
	properties := player.stateProperties()
	gotRates := map[uint16]int32{}
	for _, property := range properties {
		if property.Index >= propHPUpSpeed && property.Index <= propMPHeartSpeedAdd {
			gotRates[property.Index] = property.Value.I32
		}
	}
	if gotRates[propHPHeartSpeed] != sitcrossHeartMillis || gotRates[propMPHeartSpeed] != sitcrossHeartMillis ||
		gotRates[propHPUpSpeed] != sitcrossRecoveryPerSecond(before.MaxHP) || gotRates[propMPUpSpeed] != sitcrossRecoveryPerSecond(before.MaxMP) {
		t.Fatalf("heart rates=%v", gotRates)
	}
}

// assertSitcrossActionFrame 校验 msg24 四值打坐动作帧：
// [24, obj, i32 40|41, 窄串"dazuo_old"]。
func assertSitcrossActionFrame(t *testing.T, frame []byte, wantAction int32) {
	t.Helper()
	values := decodeCustomValuesFrame(t, frame)
	if len(values) != 4 {
		t.Fatalf("value count = %d, want 4", len(values))
	}
	if values[0].Type != 2 || values[0].Int32 != 24 {
		t.Fatalf("value[0] = %+v, want int 24", values[0])
	}
	if values[1].Type != 8 || values[1].Raw != packTestObject(playerObjectID, playerOwnerID) {
		t.Fatalf("value[1] object mismatch: %+v", values[1])
	}
	if values[2].Type != 2 || values[2].Int32 != wantAction {
		t.Fatalf("value[2] = %+v, want int %d", values[2], wantAction)
	}
	if values[3].Type != 6 || values[3].Text != "dazuo_old" {
		t.Fatalf("value[3] = %+v, want dazuo_old", values[3])
	}
}

// TestSitcrossRecoveryFramesMatchOfficialShape 锚定官服打坐回复
// 帧簇形态（FsGame 官服抓包 00:11:50.408–00:12:00.882）：
// 逐层 n=1 单属性帧 + msg24 c1=16 治疗飘字，全簇绝不含
// LogicState（带状态的整帧重推会让客户端重入坐姿动画）。
func TestSitcrossRecoveryFramesMatchOfficialShape(t *testing.T) {
	player := newPlayerActor("测试", 0)
	before := player.stateSnapshot()
	player.applyDamage(400)
	player.spendMP(50)
	player.setLogicState(entity.LogicStateSitcross)
	settlement := player.sitcrossTick()
	frames, err := player.sitcrossRecoveryFrames(settlement)
	if err != nil {
		t.Fatalf("build recovery frames: %v", err)
	}
	// 官服实证帧序（49.4.14.150:7074 实时 + FsGame timeline）：
	// [HitHP 对] HPRatio/HP + c1=16 飘字 + MPRatio/MP + c1=18 飘字。
	if len(frames) != 6 {
		t.Fatalf("frame count = %d, want 6", len(frames))
	}
	wantHP := minInt32(before.MaxHP, before.HP-400+sitcrossRecoveryPerSecond(before.MaxHP))
	wantMP := minInt32(before.MaxMP, before.MP-50+sitcrossRecoveryPerSecond(before.MaxMP))
	assertIntPropertyFrame(t, frames[0], propHPRatio, int32(int64(wantHP)*100/int64(before.MaxHP)))
	assertIntPropertyFrame(t, frames[1], propHP, wantHP)
	values := decodeCustomValuesFrame(t, frames[2])
	if len(values) != 10 {
		t.Fatalf("float value count = %d, want 10", len(values))
	}
	wantInts := map[int]int32{0: 24, 2: 16, 3: settlement.HPApplied, 4: settlement.HPWasted, 5: 0}
	for index, want := range wantInts {
		if values[index].Type != 2 || values[index].Int32 != want {
			t.Fatalf("HP float value[%d] = %+v, want int %d", index, values[index], want)
		}
	}
	if values[6].Text != sitcrossShortcutID || values[7].Text != "SkillNormal" {
		t.Fatalf("float source = %q/%q, want %s/SkillNormal", values[6].Text, values[7].Text, sitcrossShortcutID)
	}
	if values[8].Type != 7 || values[8].Text != "测试" || values[9].Type != 7 || values[9].Text != "测试" {
		t.Fatalf("float wide names = %+v/%+v, want 测试/测试", values[8], values[9])
	}
	assertIntPropertyFrame(t, frames[3], propMPRatio, int32(int64(wantMP)*100/int64(before.MaxMP)))
	assertIntPropertyFrame(t, frames[4], propMP, wantMP)
	values = decodeCustomValuesFrame(t, frames[5])
	if len(values) != 10 || values[2].Int32 != 18 || values[3].Int32 != settlement.MPApplied || values[4].Int32 != settlement.MPWasted {
		t.Fatalf("MP float = %+v, want c1=18 %d/%d", values, settlement.MPApplied, settlement.MPWasted)
	}
	// 触顶跳：先回满，再留 37 缺口（< 单跳回复量），官服末跳
	// (实际, 溢出) 拆分形态（162+1478=名义 1640 同式）。扣血会把
	// LogicState 切到 Fighting，须重进打坐态再结算。
	for state := player.stateSnapshot(); state.HP < state.MaxHP || state.MP < state.MaxMP; state = player.stateSnapshot() {
		player.sitcrossTick()
	}
	player.applyDamage(37)
	player.setLogicState(entity.LogicStateSitcross)
	settlement = player.sitcrossTick()
	if settlement.HPWasted <= 0 {
		t.Fatalf("expected capped tick, got %+v", settlement)
	}
	frames, err = player.sitcrossRecoveryFrames(settlement)
	if err != nil {
		t.Fatalf("build capped frames: %v", err)
	}
	values = decodeCustomValuesFrame(t, frames[len(frames)-1])
	if values[3].Int32 != 37 || values[4].Int32 != settlement.HPWasted {
		t.Fatalf("capped float = %d/%d, want 37/%d", values[3].Int32, values[4].Int32, settlement.HPWasted)
	}
	if settlement.HPApplied+settlement.HPWasted != sitcrossRecoveryPerSecond(player.stateSnapshot().MaxHP) {
		t.Fatalf("capped split %d+%d != nominal %d", settlement.HPApplied, settlement.HPWasted, sitcrossRecoveryPerSecond(player.stateSnapshot().MaxHP))
	}
}

// assertIntPropertyFrame 校验 0x10 n=1 单属性帧的索引与值。
func assertIntPropertyFrame(t *testing.T, frame []byte, wantIndex uint16, wantValue int32) {
	t.Helper()
	if frame[0] != 0x10 {
		t.Fatalf("frame opcode = %#x, want 0x10", frame[0])
	}
	if got := binary.LittleEndian.Uint32(frame[2:]); got != playerObjectID {
		t.Fatalf("object = %d, want %d", got, playerObjectID)
	}
	if got := binary.LittleEndian.Uint16(frame[10:]); got != 1 {
		t.Fatalf("property count = %d, want 1", got)
	}
	if got := binary.LittleEndian.Uint16(frame[12:]); got != wantIndex {
		t.Fatalf("property index = %d, want %d", got, wantIndex)
	}
	if got := int32(binary.LittleEndian.Uint32(frame[14:])); got != wantValue {
		t.Fatalf("property %d value = %d, want %d", wantIndex, got, wantValue)
	}
}

func TestSitcrossFullyRestoresAnyDerivedPoolWithinTenTicks(t *testing.T) {
	player := newPlayerActor("测试", 0)
	before := player.stateSnapshot()
	player.applyDamage(before.MaxHP - 1)
	player.spendMP(before.MaxMP - 1)
	player.setLogicState(entity.LogicStateSitcross)
	for tick := 0; tick < int(sitcrossFullRecoverySeconds); tick++ {
		player.sitcrossTick()
	}
	state := player.stateSnapshot()
	if state.HP != before.MaxHP || state.MP != before.MaxMP {
		t.Fatalf("ten-second recovery HP=%d/%d MP=%d/%d", state.HP, before.MaxHP, state.MP, before.MaxMP)
	}
}

func minInt32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func TestCombatBruiseTickDoesNotRestoreRealHP(t *testing.T) {
	player := newPlayerActor("测试", 0)
	player.applyDamage(100)
	before := player.stateSnapshot()
	if changed := player.combatBruiseTick(); !changed {
		t.Fatal("combat tick did not settle bruised blood")
	}
	state := player.stateSnapshot()
	if state.HP != before.HP || state.HitHP != before.HitHP-combatBruiseRecoveryPerSecond {
		t.Fatalf("combat tick HP=%d HitHP=%d, want %d/%d", state.HP, state.HitHP, before.HP, before.HitHP-combatBruiseRecoveryPerSecond)
	}
}
