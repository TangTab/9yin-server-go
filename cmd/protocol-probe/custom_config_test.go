package main

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/local/9yin-go-server/internal/role"
)

// customizingFrame 必须与 Java LoginFlow.customizingFrame 逐字节同形：
// 0x1E 旧壳 | u16 count=2 | tag2 u32(705) | tag6 u32(len+1) + UTF-8 + NUL。
func TestCustomizingFrameMatchesJavaLayout(t *testing.T) {
	frame, err := customizingFrame("custom\x01cfg")
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x1E || binary.LittleEndian.Uint16(frame[1:]) != 2 {
		t.Fatalf("header=%x", frame[:3])
	}
	if frame[3] != 2 || binary.LittleEndian.Uint32(frame[4:]) != uint32(serverCustomPushCustomizing) {
		t.Fatalf("message id value=%x", frame[3:8])
	}
	if frame[8] != 6 {
		t.Fatalf("payload tag=%d, want narrow string 6", frame[8])
	}
	config := "custom\x01cfg"
	if got := binary.LittleEndian.Uint32(frame[9:]); got != uint32(len(config)+1) {
		t.Fatalf("payload length=%d, want %d", got, len(config)+1)
	}
	if !strings.Contains(string(frame), config) || frame[len(frame)-1] != 0 {
		t.Fatalf("payload=%x", frame)
	}
}

// 0x10 键位单属性帧：scheme 0/1/2 → 198/199/200，串 = value + " " 尾空格
// （参考端 60310 抓包范本；0x0D 档内 198 无尾空格，勿混用）。
func TestKeybindSchemePropertyFrameAddsTrailingSpace(t *testing.T) {
	for scheme, wantIndex := range map[int32]uint16{0: 198, 1: 199, 2: 200} {
		frame, err := keybindSchemePropertyFrame(scheme, "1,87;2,83")
		if err != nil {
			t.Fatal(err)
		}
		if frame[0] != 0x10 || binary.LittleEndian.Uint32(frame[2:]) != playerObjectID {
			t.Fatalf("scheme %d header=%x", scheme, frame[:6])
		}
		if count := binary.LittleEndian.Uint16(frame[10:]); count != 1 {
			t.Fatalf("scheme %d property count=%d", scheme, count)
		}
		if got := binary.LittleEndian.Uint16(frame[12:]); got != wantIndex {
			t.Fatalf("scheme %d index=%d, want %d", scheme, got, wantIndex)
		}
		if !strings.Contains(string(frame), "1,87;2,83 \x00") {
			t.Fatalf("scheme %d value not trailing-space+NUL encoded: %x", scheme, frame)
		}
	}
	if _, err := keybindSchemePropertyFrame(3, "x"); err == nil {
		t.Fatal("scheme 3 must be rejected")
	}
}

func TestOperateModePropertyFrameEncodesInt32(t *testing.T) {
	frame, err := operateModePropertyFrame(1)
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x10 || binary.LittleEndian.Uint16(frame[10:]) != 1 {
		t.Fatalf("header=%x", frame[:12])
	}
	if got := binary.LittleEndian.Uint16(frame[12:]); got != operateModePropertyIndex {
		t.Fatalf("index=%d, want %d", got, operateModePropertyIndex)
	}
	if got := binary.LittleEndian.Uint32(frame[14:]); got != 1 {
		t.Fatalf("mode=%d", got)
	}
}

// pushKeybindFrames 无档零帧；有档按 198/199/200/223 顺序各一帧。
func TestPushKeybindFramesSkipsEmptyArchive(t *testing.T) {
	conn := &captureMessageConnection{}
	if err := pushKeybindFrames(conn, newPlayerActor("测试", 0)); err != nil {
		t.Fatal(err)
	}
	if len(conn.frames) != 0 {
		t.Fatalf("empty archive produced %d frames", len(conn.frames))
	}

	player := newPlayerActor("测试", 0)
	player.setKeybind(playerKeybind{bind: "1,87", scheme1: "2,83", scheme2: "3,65", operateMode: 1, operateModeSet: true})
	if err := pushKeybindFrames(conn, player); err != nil {
		t.Fatal(err)
	}
	if len(conn.frames) != 4 {
		t.Fatalf("frames=%d, want 4", len(conn.frames))
	}
	for index, frame := range conn.frames[:3] {
		if got := binary.LittleEndian.Uint16(frame[12:]); got != keybindPropertyIndex+uint16(index) {
			t.Fatalf("frame %d index=%d", index, got)
		}
	}
	if got := binary.LittleEndian.Uint16(conn.frames[3][12:]); got != operateModePropertyIndex {
		t.Fatalf("operate mode index=%d", got)
	}
	// 未设置操作模式时不下发（客户端保持本地默认 2.5D）。
	player2 := newPlayerActor("测试", 0)
	player2.setKeybind(playerKeybind{bind: "1,87"})
	conn2 := &captureMessageConnection{}
	if err := pushKeybindFrames(conn2, player2); err != nil {
		t.Fatal(err)
	}
	if len(conn2.frames) != 1 {
		t.Fatalf("frames=%d, want only ShortcutKey", len(conn2.frames))
	}
}

// 0x0D addObject 档内嵌：198 无尾空格、223 int32，无档零属性。
func TestAddObjectEmbedsKeybindWithoutTrailingSpace(t *testing.T) {
	location := role.Position{X: 1, Y: 2, Z: 3}
	player := newPlayerActor("测试", 0)
	bare := player.addObject(location, roleVisual{sex: 1})
	if strings.Contains(string(bare), "1,87;2,83") {
		t.Fatal("bare archive must not carry keybind values")
	}

	player.setKeybind(playerKeybind{bind: "1,87;2,83", operateMode: 1, operateModeSet: true})
	msg := player.addObject(location, roleVisual{sex: 1})
	if !strings.Contains(string(msg), "1,87;2,83\x00") || strings.Contains(string(msg), "1,87;2,83 \x00") {
		t.Fatal("archive ShortcutKey must be NUL-terminated without trailing space")
	}
	// 0x0D 属性条目自偏移 0x3f（奇数）起，逐字节扫描索引前缀。
	for offset := 0; offset+6 <= len(msg); offset++ {
		if binary.LittleEndian.Uint16(msg[offset:]) == operateModePropertyIndex && binary.LittleEndian.Uint32(msg[offset+2:]) == 1 {
			return
		}
	}
	t.Fatal("archive does not embed OperateMode(223)=1")
}

// 绑定串合法性：恰好 25 值 + 尾分号，值 0..99（客户端 G_NG_GRID_COUNT=25、
// 绑定框 DataSource 序号；get_svr_ng_bind 按拆段数校验，脏串会整串回落全零）。
func TestValidNeiGongBind(t *testing.T) {
	canonical := "0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;"
	if !validNeiGongBind(canonical) {
		t.Fatal("all-zero canonical bind must be accepted")
	}
	withValues := "0;0;1;2;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;"
	if !validNeiGongBind(withValues) {
		t.Fatal("bind with values must be accepted")
	}
	for name, value := range map[string]string{
		"missing trailing semicolon": strings.TrimSuffix(canonical, ";"),
		"short slot count":           "0;0;0;",
		"extra segment":              canonical + "0;",
		"empty value":                strings.Replace(canonical, "0;0;", ";;", 1),
		"value out of range":         strings.Replace(canonical, "0;0;", "100;", 1),
		"non numeric":                strings.Replace(canonical, "0;0;", "a;", 1),
		"empty string":               "",
	} {
		if validNeiGongBind(value) {
			t.Fatalf("%s must be rejected: %q", name, value)
		}
	}
}

// 199：合法上传回推 0x10 NeiGongBind(201) + 更新内存；非法串静默零帧。
func TestHandleNeiGongBindCustomValidatesAndEchoes(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	upload := func(value string) clientCustomMessage {
		return clientCustomMessage{Values: []clientCustomValue{
			{Type: 2, Int32: clientCustomNeiGongBind},
			{Type: 6, Text: value},
		}}
	}

	bind := "0;1;0;0;2;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;"
	handled, err := handleNeiGongBindCustom(conn, player, upload(bind), "test", nil, 0)
	if err != nil || !handled || len(conn.frames) != 1 {
		t.Fatalf("valid upload handled=%t err=%v frames=%d", handled, err, len(conn.frames))
	}
	frame := conn.frames[0]
	if got := binary.LittleEndian.Uint16(frame[12:]); got != neiGongBindPropertyIndex {
		t.Fatalf("echo index=%d, want %d", got, neiGongBindPropertyIndex)
	}
	if !strings.Contains(string(frame), bind+" \x00") {
		t.Fatalf("echo value must carry trailing space: %x", frame)
	}
	if snapshot := player.keybindSnapshot(); snapshot.neigongBind != bind {
		t.Fatalf("memory neigongBind=%q", snapshot.neigongBind)
	}

	for name, value := range map[string]string{
		"wrong slot count": "0;1;",
		"garbage":          "abc",
		"range":            strings.Replace(bind, "0;1;", "0;777;", 1),
	} {
		connReject := &captureMessageConnection{}
		handled, err := handleNeiGongBindCustom(connReject, player, upload(value), "test", nil, 0)
		if err != nil || !handled || len(connReject.frames) != 0 {
			t.Fatalf("%s handled=%t err=%v frames=%d", name, handled, err, len(connReject.frames))
		}
	}

	// player 未生成（选角前）忽略且零帧。
	connBefore := &captureMessageConnection{}
	handled, err = handleNeiGongBindCustom(connBefore, nil, upload(bind), "test", nil, 0)
	if err != nil || !handled || len(connBefore.frames) != 0 {
		t.Fatalf("before-spawn handled=%t err=%v frames=%d", handled, err, len(connBefore.frames))
	}
}

// pushKeybindFrames 尾随 201 帧（有档才发）；0x0D 档内嵌 201 无尾空格。
func TestNeiGongBindPushAndArchiveEmbed(t *testing.T) {
	bind := "1;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;0;"
	player := newPlayerActor("测试", 0)
	player.setKeybind(playerKeybind{neigongBind: bind})
	conn := &captureMessageConnection{}
	if err := pushKeybindFrames(conn, player); err != nil {
		t.Fatal(err)
	}
	if len(conn.frames) != 1 {
		t.Fatalf("frames=%d, want only NeiGongBind", len(conn.frames))
	}
	if got := binary.LittleEndian.Uint16(conn.frames[0][12:]); got != neiGongBindPropertyIndex {
		t.Fatalf("index=%d, want %d", got, neiGongBindPropertyIndex)
	}

	location := role.Position{X: 1, Y: 2, Z: 3}
	msg := player.addObject(location, roleVisual{sex: 1})
	if !strings.Contains(string(msg), bind+"\x00") || strings.Contains(string(msg), bind+" \x00") {
		t.Fatal("archive NeiGongBind must be NUL-terminated without trailing space")
	}
	// 全零档（空串）零属性：bare 档不携带 201。
	barePlayer := newPlayerActor("测试", 0)
	bare := barePlayer.addObject(location, roleVisual{sex: 1})
	for offset := 0; offset+2 <= len(bare); offset++ {
		if binary.LittleEndian.Uint16(bare[offset:]) == neiGongBindPropertyIndex {
			t.Fatal("bare archive must not embed NeiGongBind")
		}
	}
}

func keybindUploadCustom(scheme int32, value string) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomKeybindUpload},
		{Type: 2, Int32: scheme},
		{Type: 6, Text: value},
	}}
}

func customizingUploadCustom(config string) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomSaveCustomizing},
		{Type: 6, Text: config},
	}}
}

// 195：合法上传回推 0x10 + 更新内存；非法 scheme/非可打印串静默零帧。
func TestHandleKeybindCustomValidatesAndEchoes(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)

	handled, err := handleKeybindCustom(conn, player, keybindUploadCustom(1, "1,87;2,83"), "test", nil, 0)
	if err != nil || !handled || len(conn.frames) != 1 {
		t.Fatalf("valid upload handled=%t err=%v frames=%d", handled, err, len(conn.frames))
	}
	if got := binary.LittleEndian.Uint16(conn.frames[0][12:]); got != keybindScheme1PropertyIndex {
		t.Fatalf("echo index=%d", got)
	}
	if snapshot := player.keybindSnapshot(); snapshot.scheme1 != "1,87;2,83" {
		t.Fatalf("memory scheme1=%q", snapshot.scheme1)
	}

	for name, upload := range map[string]clientCustomMessage{
		"scheme out of range": keybindUploadCustom(5, "1,87"),
		"empty string":        keybindUploadCustom(0, ""),
		"control byte":        keybindUploadCustom(0, "1,87;\x1F"),
		"non-ascii":           keybindUploadCustom(0, "1,键"),
	} {
		conn := &captureMessageConnection{}
		handled, err := handleKeybindCustom(conn, player, upload, "test", nil, 0)
		if err != nil || !handled || len(conn.frames) != 0 {
			t.Fatalf("%s handled=%t err=%v frames=%d", name, handled, err, len(conn.frames))
		}
	}

	// player 未生成（选角前）忽略且零帧。
	connBefore := &captureMessageConnection{}
	handled, err = handleKeybindCustom(connBefore, nil, keybindUploadCustom(0, "1,87"), "test", nil, 0)
	if err != nil || !handled || len(connBefore.frames) != 0 {
		t.Fatalf("before-spawn handled=%t err=%v frames=%d", handled, err, len(connBefore.frames))
	}
}

// 99：mode 0/1 回推并落内存；非法值静默零帧。
func TestHandleOperateModeCustomValidatesMode(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	upload := func(mode int32) clientCustomMessage {
		return clientCustomMessage{Values: []clientCustomValue{
			{Type: 2, Int32: clientCustomSetOperateMode},
			{Type: 2, Int32: mode},
		}}
	}
	handled, err := handleOperateModeCustom(conn, player, upload(1), "test", nil, 0)
	if err != nil || !handled || len(conn.frames) != 1 {
		t.Fatalf("set 2.5D handled=%t err=%v frames=%d", handled, err, len(conn.frames))
	}
	if snapshot := player.keybindSnapshot(); !snapshot.operateModeSet || snapshot.operateMode != 1 {
		t.Fatalf("operate mode=%+v", snapshot)
	}

	connReject := &captureMessageConnection{}
	handled, err = handleOperateModeCustom(connReject, player, upload(2), "test", nil, 0)
	if err != nil || !handled || len(connReject.frames) != 0 {
		t.Fatalf("mode 2 handled=%t err=%v frames=%d", handled, err, len(connReject.frames))
	}
}

// 107 时序闸：会话首传在库里有存档时跳过落库（防默认键位覆盖）并回发
// 存档；无档首传建基线；后续上传落档并回显（Java LoginFlow 同款）。
func TestHandleCustomizingCustomFirstUploadGate(t *testing.T) {
	player := newPlayerActor("测试", 0)

	// 选角前忽略，不消耗首传闸。
	connEarly := &captureMessageConnection{}
	session := customizingSession{}
	handled, err := handleCustomizingCustom(connEarly, nil, customizingUploadCustom("default"), "test", &session, nil, 0)
	if err != nil || !handled || len(connEarly.frames) != 0 || session.seenUpload {
		t.Fatalf("before-spawn handled=%t err=%v frames=%d seen=%t", handled, err, len(connEarly.frames), session.seenUpload)
	}

	// 有存档：首传（默认键位转储）跳过落库；官服方言零 705 回显
	//（on_parser_customizing 强制 game_control_mode=0 打断 2.5D 点击
	// 移动——2026-11-09 停推，见 custom_config.go pushSavedCustomizing）。
	store, mock := newMockCustomizingDB(t)
	mock.ExpectQuery(`SELECT customizing_value FROM role_customizing WHERE role_id = \?`).
		WithArgs(customConfigTestRoleID).
		WillReturnRows(newSQLMockBytesRow("customizing_value", "saved-config"))
	conn := &captureMessageConnection{}
	saved, exists := store.Load(customConfigTestRoleID)
	if !exists {
		t.Fatal("expected saved archive")
	}
	session = customizingSession{saved: saved, hasSaved: exists}
	handled, err = handleCustomizingCustom(conn, player, customizingUploadCustom("default-dump"), "test", &session, store, customConfigTestRoleID)
	if err != nil || !handled || len(conn.frames) != 0 {
		t.Fatalf("first upload handled=%t err=%v frames=%d", handled, err, len(conn.frames))
	}

	// 后续上传落库（官服不回显）。
	mock.ExpectExec(`INSERT INTO role_customizing \(role_id, customizing_value\) VALUES \(\?, \?\)`+
		` ON DUPLICATE KEY UPDATE customizing_value = VALUES\(customizing_value\)`).
		WithArgs(customConfigTestRoleID, []byte("player-edit")).
		WillReturnResult(newSQLMockResult(0, 1))
	connEcho := &captureMessageConnection{}
	handled, err = handleCustomizingCustom(connEcho, player, customizingUploadCustom("player-edit"), "test", &session, store, customConfigTestRoleID)
	if err != nil || !handled || len(connEcho.frames) != 0 {
		t.Fatalf("second upload handled=%t err=%v frames=%d", handled, err, len(connEcho.frames))
	}

	// 无存档会话：首传直接建基线（零回显）。
	store2, mock2 := newMockCustomizingDB(t)
	mock2.ExpectExec(`INSERT INTO role_customizing \(role_id, customizing_value\) VALUES \(\?, \?\)`+
		` ON DUPLICATE KEY UPDATE customizing_value = VALUES\(customizing_value\)`).
		WithArgs(customConfigTestRoleID, []byte("first-baseline")).
		WillReturnResult(newSQLMockResult(0, 1))
	connFresh := &captureMessageConnection{}
	freshSession := customizingSession{}
	handled, err = handleCustomizingCustom(connFresh, player, customizingUploadCustom("first-baseline"), "test", &freshSession, store2, customConfigTestRoleID)
	if err != nil || !handled || len(connFresh.frames) != 0 || !freshSession.hasSaved {
		t.Fatalf("fresh first upload handled=%t err=%v frames=%d saved=%t", handled, err, len(connFresh.frames), freshSession.hasSaved)
	}
}
