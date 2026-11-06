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
	handled, err := handleShortcutCustom(conn, player, set, "test")
	if err != nil || !handled || len(conn.frames) != 1 {
		t.Fatalf("first set handled=%t err=%v frames=%d", handled, err, len(conn.frames))
	}
	add := conn.frames[0]
	if add[0] != 0x11 || binary.LittleEndian.Uint16(add[10:]) != recordShortcut ||
		binary.LittleEndian.Uint32(add[16:]) != 3 || !strings.Contains(string(add), "CS_yhwq_hsqs01") {
		t.Fatalf("shortcut add=%x", add)
	}

	set.Values[3].Text = "CS_yhwq_hsqs02"
	handled, err = handleShortcutCustom(conn, player, set, "test")
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

func TestRemoveShortcutUsesRecordRow(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	player.setShortcut(8, "skill", "CS_yhwq_hsqs01")
	player.setShortcut(9, "skill", "CS_yhwq_hsqs02")

	remove := shortcutCustom(clientCustomRemoveShortcut, clientCustomValue{Type: 2, Int32: 1})
	handled, err := handleShortcutCustom(conn, player, remove, "test")
	if err != nil || !handled || len(conn.frames) != 1 {
		t.Fatalf("remove handled=%t err=%v frames=%d", handled, err, len(conn.frames))
	}
	if got := binary.LittleEndian.Uint16(conn.frames[0][12:]); conn.frames[0][0] != 0x12 || got != 1 {
		t.Fatalf("remove frame=%x", conn.frames[0])
	}
	snapshot := player.shortcutSnapshot()
	if len(snapshot) != 1 || snapshot[0].index != 8 {
		t.Fatalf("remaining shortcuts=%+v", snapshot)
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

func TestGrantSitcrossShortcutAddsLearnedNativeSkillWithoutOverwritingSlot(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	granted, err := grantSitcrossShortcut(conn, player)
	if err != nil || !granted || len(conn.frames) != 1 {
		t.Fatalf("first grant granted=%t err=%v frames=%d", granted, err, len(conn.frames))
	}
	if conn.frames[0][0] != 0x11 || !strings.Contains(string(conn.frames[0]), "zs_default_01") || !strings.Contains(string(conn.frames[0]), "skill") {
		t.Fatalf("sitcross add=%x", conn.frames[0])
	}
	granted, err = grantSitcrossShortcut(conn, player)
	if err != nil || granted || len(conn.frames) != 1 {
		t.Fatalf("second grant granted=%t err=%v frames=%d", granted, err, len(conn.frames))
	}

	other := newPlayerActor("占位", 0)
	other.setShortcut(sitcrossShortcutIndex, "skill", "CS_yhwq_hsqs01")
	granted, err = grantSitcrossShortcut(&captureMessageConnection{}, other)
	if err != nil || granted {
		t.Fatalf("occupied slot grant granted=%t err=%v", granted, err)
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
	if got := player.actor.Snapshot().LogicState; got != entity.LogicStateSitcross {
		t.Fatalf("LogicState=%d, want sitcross=%d", got, entity.LogicStateSitcross)
	}
	if frames := conn.Frames(); len(frames) != 1 || frames[0][0] != 0x10 {
		t.Fatalf("expected one normal 0x10 property update, got %x", frames)
	}

	// 按住按键会产生重复的 240,1 数据包。此时必须保持在状态
	// 102，而不是表现为切换开关。
	handled, err = handleSitcrossCustom(conn, player, start, "test")
	if err != nil || !handled || len(conn.Frames()) != 1 {
		t.Fatalf("duplicate handled=%t err=%v frames=%d", handled, err, len(conn.Frames()))
	}

	stop := shortcutCustom(clientCustomSitCross, clientCustomValue{Type: 2, Int32: 0})
	handled, err = handleSitcrossCustom(conn, player, stop, "test")
	if err != nil || !handled || player.actor.Snapshot().LogicState != entity.LogicStateNormal || len(conn.Frames()) != 2 {
		t.Fatalf("stop handled=%t err=%v state=%d frames=%d", handled, err, player.actor.Snapshot().LogicState, len(conn.Frames()))
	}

	// 真实客户端抓包：敌方打断会发出 240,2 而不是旧版的
	// 取消值 0。此时必须停止打坐，
	// 而不能关闭连接。
	handled, err = handleSitcrossCustom(conn, player, start, "test")
	if err != nil || !handled || player.actor.Snapshot().LogicState != entity.LogicStateSitcross {
		t.Fatalf("second start handled=%t err=%v state=%d", handled, err, player.actor.Snapshot().LogicState)
	}
	interrupted := shortcutCustom(clientCustomSitCross, clientCustomValue{Type: 2, Int32: 2})
	handled, err = handleSitcrossCustom(conn, player, interrupted, "test")
	if err != nil || !handled || player.actor.Snapshot().LogicState != entity.LogicStateNormal {
		t.Fatalf("interruption handled=%t err=%v state=%d", handled, err, player.actor.Snapshot().LogicState)
	}
}

func TestSitcrossTickRestoresAuthoritativeHPAndMP(t *testing.T) {
	player := newPlayerActor("测试", 0)
	before := player.actor.Snapshot()
	player.actor.ApplyDamage(100)
	player.actor.SpendMP(100)
	player.actor.SetLogicState(entity.LogicStateSitcross)
	if changed := player.sitcrossTick(); !changed {
		t.Fatal("sitcross tick did not change damaged actor")
	}
	state := player.actor.Snapshot()
	wantHP := minInt32(before.MaxHP, before.HP-100+sitcrossRecoveryPerSecond(before.MaxHP))
	wantMP := minInt32(before.MaxMP, before.MP-100+sitcrossRecoveryPerSecond(before.MaxMP))
	if state.HP != wantHP || state.MP != wantMP {
		t.Fatalf("sitcross state HP=%d MP=%d, want %d/%d", state.HP, state.MP, wantHP, wantMP)
	}
	properties := player.stateProperties()
	gotRates := map[uint16]int32{}
	for _, property := range properties {
		if property.Index >= propHPHeartSpeed && property.Index <= propMPUpSpeedAdd {
			gotRates[property.Index] = property.Value.I32
		}
	}
	if gotRates[propHPHeartSpeed] != sitcrossHeartMillis || gotRates[propMPHeartSpeed] != sitcrossHeartMillis ||
		gotRates[propHPUpSpeed] != sitcrossRecoveryPerSecond(before.MaxHP) || gotRates[propMPUpSpeed] != sitcrossRecoveryPerSecond(before.MaxMP) {
		t.Fatalf("heart rates=%v", gotRates)
	}
}

func TestSitcrossFullyRestoresAnyDerivedPoolWithinTenTicks(t *testing.T) {
	player := newPlayerActor("测试", 0)
	before := player.actor.Snapshot()
	player.actor.ApplyDamage(before.MaxHP - 1)
	player.actor.SpendMP(before.MaxMP - 1)
	player.actor.SetLogicState(entity.LogicStateSitcross)
	for tick := 0; tick < int(sitcrossFullRecoverySeconds); tick++ {
		player.sitcrossTick()
	}
	state := player.actor.Snapshot()
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
	player.actor.ApplyDamage(100)
	before := player.actor.Snapshot()
	if changed := player.combatBruiseTick(); !changed {
		t.Fatal("combat tick did not settle bruised blood")
	}
	state := player.actor.Snapshot()
	if state.HP != before.HP || state.HitHP != before.HitHP-combatBruiseRecoveryPerSecond {
		t.Fatalf("combat tick HP=%d HitHP=%d, want %d/%d", state.HP, state.HitHP, before.HP, before.HitHP-combatBruiseRecoveryPerSecond)
	}
}
