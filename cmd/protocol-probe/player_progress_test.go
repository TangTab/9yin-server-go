package main

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

func TestPlayerActorPublishesYiHuaFactionAsSchool(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	if got := player.actor.Snapshot().LogicState; got != 0 {
		t.Fatalf("persistent normal faculty starts LogicState=%d, want background state 0", got)
	}
	if player.progress.facultyState != facultyStateConvert || player.progress.facultyStyle != facultyStyleNormal || player.progress.facultyName == "" {
		t.Fatalf("initial persistent faculty=%+v", player.progress)
	}
	properties := make(map[string]string)
	for _, property := range player.stateProperties() {
		if property.Name == "School" || property.Name == "Force" || property.Name == "NewSchool" {
			properties[property.Name] = property.Value.Text
		}
	}
	if got := properties["Force"]; got != defaultRoleFaction {
		t.Fatalf("Force=%q, want %q", got, defaultRoleFaction)
	}
	if got := properties["School"]; got != "" {
		t.Fatalf("force must clear School, got %q", got)
	}
	if got := properties["NewSchool"]; got != "" {
		t.Fatalf("force must clear NewSchool, got %q", got)
	}
}

func TestPlayerActorPublishesNewSchoolSeparately(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	player.faction = "newschool_shenjihui"
	properties := make(map[string]string)
	for _, property := range player.stateProperties() {
		if property.Name == "School" || property.Name == "Force" || property.Name == "NewSchool" {
			properties[property.Name] = property.Value.Text
		}
	}
	if got := properties["NewSchool"]; got != "newschool_shenjihui" {
		t.Fatalf("NewSchool=%q", got)
	}
	if properties["School"] != "" || properties["Force"] != "" {
		t.Fatalf("new school fields=%v", properties)
	}
}

func TestPlayerActorSPAuthorityClampsAndPublishesCurrentValue(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	sp, frame, err := player.setSP(999)
	if err != nil {
		t.Fatal(err)
	}
	if sp != maxPlayerSP || player.skillSP() != maxPlayerSP {
		t.Fatalf("set SP=%d actor=%d, want %d", sp, player.skillSP(), maxPlayerSP)
	}
	if frame[0] != 0x10 || !bytes.Contains(frame, []byte{byte(propSP), byte(propSP >> 8), 100, 0, 0, 0}) {
		t.Fatalf("set SP frame=%x", frame)
	}
	sp, _, err = player.addSP(-150)
	if err != nil {
		t.Fatal(err)
	}
	if sp != 0 || player.skillSP() != 0 {
		t.Fatalf("subtracted SP=%d actor=%d, want 0", sp, player.skillSP())
	}
	sp, _, err = player.addSP(15)
	if err != nil || sp != 15 || player.skillSP() != 15 {
		t.Fatalf("added SP=%d actor=%d err=%v, want 15", sp, player.skillSP(), err)
	}
}

func TestStarterInnerPowerViewUsesNegotiatedBookFields(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	if got := player.actor.Snapshot(); got.BaseMaxHP != 1000 || got.MaxHP != 2630 || got.BaseMaxMP != 500 || got.MaxMP != 990 {
		t.Fatalf("两仪护心功 server caps=%+v, want base 1000/500 total 2630/990", got)
	}
	if got := player.progress.books[0].staticData; got != 5 {
		t.Fatalf("ng_wd_001 StaticData=%d, want classic visible-buff value 5", got)
	}
	if got := player.bufferInfo(innerPowerBuffSlot, time.Now()); !bytes.HasPrefix([]byte(got), []byte("1742,1,1,3,")) {
		t.Fatalf("level-30 neigong BufferInfo=%q", got)
	}
	if got := player.bufferListString(time.Now()); !bytes.Contains([]byte(got), []byte("buf_ng_wd_001,local-4,3,1-1,")) {
		t.Fatalf("level-30 neigong BufferListStr=%q", got)
	}
	wantAdds := map[string]int32{
		"MaxHPAdd":       1630,
		"MaxMPAdd":       490,
		"MaxParryAdd":    840,
		"MinMagicDefAdd": 45,
	}
	for _, property := range player.progressProperties() {
		if want, ok := wantAdds[property.Name]; ok {
			if property.Value.I32 != want {
				t.Fatalf("%s=%d, want %d", property.Name, property.Value.I32, want)
			}
			delete(wantAdds, property.Name)
		}
	}
	if len(wantAdds) != 0 {
		t.Fatalf("missing equipped inner-power additions: %v", wantAdds)
	}
	frames, err := player.progressViewFrames()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(frames), 7; got != want {
		t.Fatalf("View=43 frames=%d, want %d", got, want)
	}
	if frame := frames[0]; frame[0] != 0x15 || binary.LittleEndian.Uint16(frame[1:]) != viewportNeiGong || binary.LittleEndian.Uint16(frame[3:]) != 6 {
		t.Fatalf("CreateView(43)=%x", frame)
	}
	first := frames[1]
	if first[0] != 0x18 || binary.LittleEndian.Uint16(first[1:]) != viewportNeiGong || binary.LittleEndian.Uint16(first[3:]) != 1 {
		t.Fatalf("first ViewAdd=%x", first)
	}
	for _, expected := range [][]byte{
		[]byte("ng_wd_001\x00"),
		{byte(propStaticData), byte(propStaticData >> 8)},
		{byte(propItemType), byte(propItemType >> 8)},
		{byte(propMaxLevel), byte(propMaxLevel >> 8)},
		{byte(propCurFillValue), byte(propCurFillValue >> 8)},
		{byte(propMaxPowerValue), byte(propMaxPowerValue >> 8)},
	} {
		if !bytes.Contains(first, expected) {
			t.Fatalf("first inner-power ViewAdd lacks %x: %x", expected, first)
		}
	}
}

func TestTaoHuaInnerPowerUsesClassicVisibleRoute(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	book := player.progress.book("ng_th_001")
	if book == nil {
		t.Fatal("missing 桃花岛 碧波心法")
	}
	if book.staticData != 113 || book.level != 30 || book.maxLevel != 71 || book.total != 579600 {
		t.Fatalf("碧波心法 metadata=%+v", *book)
	}
	if err := player.equipNeiGong("ng_th_001"); err != nil {
		t.Fatal(err)
	}
	if got := player.bufferInfo(innerPowerBuffSlot, time.Now()); !bytes.HasPrefix([]byte(got), []byte("347A,1,1,3,")) {
		t.Fatalf("碧波心法 BufferInfo=%q", got)
	}
}

func TestYiHuaInnerPowerUsesSharedClassicVisibleRoute(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	book := player.progress.book("ng_yh_001")
	if book == nil {
		t.Fatal("missing 移花宫 明玉神功")
	}
	if book.staticData != 114 || book.level != 30 || book.maxLevel != 71 || book.total != 579600 {
		t.Fatalf("明玉神功 metadata=%+v", *book)
	}
	if err := player.equipNeiGong("ng_yh_001"); err != nil {
		t.Fatal(err)
	}
	if got := player.actor.Snapshot(); got.BaseMaxHP != 1000 || got.MaxHP != 5050 || got.BaseMaxMP != 500 || got.MaxMP != 1630 {
		t.Fatalf("明玉神功 server caps=%+v, want base 1000/500 total 5050/1630", got)
	}
	if got := player.bufferInfo(innerPowerBuffSlot, time.Now()); !bytes.HasPrefix([]byte(got), []byte("3311,1,1,3,")) {
		t.Fatalf("明玉神功 BufferInfo=%q", got)
	}
	if got := player.bufferListString(time.Now()); !bytes.Contains([]byte(got), []byte("buf_ng_yh_001,local-4,3,1-1,")) {
		t.Fatalf("明玉神功 BufferListStr=%q", got)
	}
	wantAdds := map[string]int32{
		"MaxHPAdd":       4050,
		"MaxMPAdd":       1130,
		"MaxParryAdd":    5040,
		"MinMagicDefAdd": 100,
	}
	for _, property := range player.progressProperties() {
		if want, ok := wantAdds[property.Name]; ok {
			if property.Value.I32 != want {
				t.Fatalf("%s=%d, want %d", property.Name, property.Value.I32, want)
			}
			delete(wantAdds, property.Name)
		}
	}
	if len(wantAdds) != 0 {
		t.Fatalf("missing 明玉神功 additions: %v", wantAdds)
	}
}

func TestInnerPowerC2SUpdatesNormalPropertyAndViewState(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	conn := &captureMessageConnection{}

	// 215 must accept only learned View=43 books and reply with the normal
	// property archive that the CurNeiGong bind consumes.
	handled, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 215}, {Type: 6, Text: "ng_jh_001"},
	}})
	if err != nil || !handled {
		t.Fatalf("215 handled=%v err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 1 || frames[0][0] != 0x10 || !bytes.Contains(frames[0], []byte("ng_jh_001\x00")) {
		t.Fatalf("215 reply=%x", frames)
	}

	// 1006/1 chooses a book; 1006/11 starts persistent background conversion
	// without entering the visible 演武 LogicState=108 route.
	handled, err = handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 1006}, {Type: 2, Int32: 1}, {Type: 6, Text: "ng_wd_001"},
	}})
	if err != nil || !handled {
		t.Fatalf("1006/1 handled=%v err=%v", handled, err)
	}
	readyFrames := conn.Frames()
	if len(readyFrames) < 3 {
		t.Fatalf("1006/1 must append View=43 refresh and player state: %x", readyFrames)
	}
	readyView := readyFrames[len(readyFrames)-2]
	readyState := readyFrames[len(readyFrames)-1]
	if readyView[0] != 0x18 || binary.LittleEndian.Uint16(readyView[1:]) != viewportNeiGong {
		t.Fatalf("1006/1 first reply must refresh selected View=43 book: %x", readyView)
	}
	if readyState[0] != 0x10 || !bytes.Contains(readyState, []byte("ng_wd_001\x00")) {
		t.Fatalf("1006/1 second reply must publish FacultyName state: %x", readyState)
	}
	if got := player.progress.facultyState; got != facultyStateNone {
		t.Fatalf("FacultyState after 1006/1=%d, want client-rendered idle=%d", got, facultyStateNone)
	}
	beforeRefresh := len(conn.Frames())
	handled, err = handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 1006}, {Type: 2, Int32: 24},
	}})
	if err != nil || !handled {
		t.Fatalf("1006/24 handled=%v err=%v", handled, err)
	}
	refreshFrames := conn.Frames()[beforeRefresh:]
	if len(refreshFrames) != 2 {
		t.Fatalf("1006/24 refresh frames=%d want clear+restore", len(refreshFrames))
	}
	emptyFacultyName := []byte{byte(propFacultyName), byte(propFacultyName >> 8), 1, 0, 0, 0, 0}
	if !bytes.Contains(refreshFrames[0], emptyFacultyName) {
		t.Fatalf("1006/24 first frame must clear FacultyName: %x", refreshFrames[0])
	}
	if !bytes.Contains(refreshFrames[1], []byte("ng_wd_001\x00")) {
		t.Fatalf("1006/24 second frame must restore FacultyName: %x", refreshFrames[1])
	}
	handled, err = handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 1006}, {Type: 2, Int32: 11},
	}})
	if err != nil || !handled {
		t.Fatalf("1006/11 handled=%v err=%v", handled, err)
	}
	if got := player.actor.Snapshot().LogicState; got != 0 {
		t.Fatalf("normal faculty LogicState=%d, want background state 0", got)
	}
	if player.progress.facultyState != facultyStateConvert || player.progress.facultyStyle != facultyStyleNormal {
		t.Fatalf("normal faculty state=%d style=%d", player.progress.facultyState, player.progress.facultyStyle)
	}
	frames = conn.Frames()
	viewUpdated := false
	for _, frame := range frames {
		if len(frame) >= 3 && frame[0] == 0x18 && binary.LittleEndian.Uint16(frame[1:]) == viewportNeiGong {
			viewUpdated = true
			break
		}
	}
	if !viewUpdated {
		t.Fatalf("1006/11 must refresh View=43, got %x", frames)
	}

	_, err = handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 1006}, {Type: 2, Int32: 12},
	}})
	if err != nil {
		t.Fatalf("1006/12: %v", err)
	}
	if got := player.actor.Snapshot().LogicState; got != 0 {
		t.Fatalf("LogicState after exit=%d, want 0", got)
	}
	if got := player.progress.facultyState; got != facultyStateNone {
		t.Fatalf("FacultyState after exit=%d, want client-rendered idle=%d", got, facultyStateNone)
	}

	// The faculty-panel 演武 button emits the same C->S 1006 bus with sub=21,
	// but its native display state is FacultyStyle=2 rather than normal=1.
	handled, err = handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 1006}, {Type: 2, Int32: 21},
	}})
	if err != nil || !handled {
		t.Fatalf("1006/21 handled=%v err=%v", handled, err)
	}
	if got := player.progress.facultyStyle; got != facultyStyleAct {
		t.Fatalf("FacultyStyle after 1006/21=%d, want active=%d", got, facultyStyleAct)
	}
	if got := player.actor.Snapshot().LogicState; got != logicStateFaculty {
		t.Fatalf("LogicState after 1006/21=%d, want faculty=%d", got, logicStateFaculty)
	}
}

func TestPersistentFacultySettlesElapsedMinutes(t *testing.T) {
	progress := newPlayerProgress()
	start := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	progress.lastAdvance = start
	beforeFill := progress.books[0].fill
	beforeFaculty := progress.faculty
	slot, changed := progress.advanceFaculty(start.Add(2*time.Minute + 30*time.Second))
	if !changed || slot != 1 {
		t.Fatalf("advance changed=%t slot=%d", changed, slot)
	}
	if got, want := progress.books[0].fill, beforeFill+400; got != want {
		t.Fatalf("fill=%d want=%d", got, want)
	}
	if got, want := progress.faculty, beforeFaculty-200; got != want {
		t.Fatalf("faculty=%d want=%d", got, want)
	}
	if got, want := progress.lastAdvance, start.Add(2*time.Minute); !got.Equal(want) {
		t.Fatalf("lastAdvance=%s want=%s", got, want)
	}
}

func TestActiveFacultyPaysOnlyAfterCardSelection(t *testing.T) {
	progress := newPlayerProgress()
	start := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	if err := progress.facultyBegin(facultyStyleAct, start); err != nil {
		t.Fatalf("begin active faculty: %v", err)
	}
	beforeFill := progress.books[0].fill
	slot, changed := progress.advanceFaculty(start.Add(time.Minute + 20*time.Second))
	if changed || slot != 0 {
		t.Fatalf("active minute tick changed=%t slot=%d", changed, slot)
	}
	if got := progress.books[0].fill; got != beforeFill {
		t.Fatalf("active timer fill=%d want unchanged %d", got, beforeFill)
	}
	slot, changed, levelUp, maxLevel, err := progress.awardActiveFaculty()
	if err != nil || !changed || levelUp || maxLevel || slot != 1 {
		t.Fatalf("award active faculty slot=%d changed=%t levelUp=%t maxLevel=%t err=%v", slot, changed, levelUp, maxLevel, err)
	}
	if got, want := progress.books[0].fill, beforeFill+activeFacultyCardReward; got != want {
		t.Fatalf("active card fill=%d want=%d", got, want)
	}
	if progress.facultyStyle != facultyStyleAct || progress.facultyState != facultyStateConvert {
		t.Fatalf("active state=%d style=%d", progress.facultyState, progress.facultyStyle)
	}
}

func TestActiveFacultyAcceptsLiveDoubleCardSelection(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	_, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 1006}, {Type: 2, Int32: 21},
	}})
	if err != nil {
		t.Fatalf("1006/21: %v", err)
	}
	before := player.progress.books[0].fill
	handled, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 1006}, {Type: 2, Int32: 23},
		{Type: 5, Float64: 0}, {Type: 5, Float64: 1},
	}})
	if err != nil || !handled {
		t.Fatalf("live 1006/23 handled=%t err=%v", handled, err)
	}
	if got, want := player.progress.books[0].fill, before+activeFacultyCardReward; got != want {
		t.Fatalf("active double card fill=%d want=%d", got, want)
	}
	frames := conn.Frames()
	if len(frames) == 0 || !bytes.Contains(frames[len(frames)-1], []byte{2, 24, 0, 0, 0}) {
		t.Fatalf("active card reply is missing SUB_SERVER_ACT_PLAY: %x", frames)
	}
}

func TestStaleActiveFacultyCardSelectionDoesNotDisconnect(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	before := player.progress.books[0].fill
	handled, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 1006}, {Type: 2, Int32: 23},
		{Type: 5, Float64: 0}, {Type: 5, Float64: 1},
	}})
	if err != nil || !handled {
		t.Fatalf("stale 1006/23 handled=%t err=%v", handled, err)
	}
	if got := player.progress.books[0].fill; got != before {
		t.Fatalf("stale card selection changed fill=%d want=%d", got, before)
	}
	if frames := conn.Frames(); len(frames) != 2 {
		t.Fatalf("stale card selection replies=%d want state+play", len(frames))
	}
}

func TestActiveFacultyLevelUpNotifiesClientAfterStateAndViewRefresh(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	player.mu.Lock()
	player.progress.books[0].fill = player.progress.books[0].total - 1
	player.mu.Unlock()
	_, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 1006}, {Type: 2, Int32: 21},
	}})
	if err != nil {
		t.Fatalf("1006/21: %v", err)
	}
	handled, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 1006}, {Type: 2, Int32: 23},
		{Type: 5, Float64: 0}, {Type: 5, Float64: 1},
	}})
	if err != nil || !handled {
		t.Fatalf("level-up 1006/23 handled=%t err=%v", handled, err)
	}
	if got, want := player.progress.books[0].level, int32(31); got != want {
		t.Fatalf("level=%d want=%d", got, want)
	}
	frames := conn.Frames()
	levelUp := []byte{2, 14, 0, 0, 0, 2, 0, 0, 0, 0}
	found := false
	for _, frame := range frames {
		if bytes.Contains(frame, levelUp) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("level-up reply is missing faculty sub=14: %x", frames)
	}
	// A level-up must first refresh the faculty dialog from the authoritative
	// View=43 and role properties.  Only then may the active page close; its
	// native exit_flag still suppresses the stale C2S 1006/22 callback.
	exit := []byte{2, 23, 0, 0, 0, 2, 0, 0, 0, 0}
	play := []byte{2, 24, 0, 0, 0, 2, 0, 0, 0, 0}
	exitIndex, levelIndex := -1, -1
	for index, frame := range frames {
		if bytes.Contains(frame, exit) {
			exitIndex = index
		}
		if bytes.Contains(frame, levelUp) {
			levelIndex = index
		}
		if bytes.Contains(frame, play) {
			t.Fatalf("level-up must not leave active ACT_PLAY queued: %x", frame)
		}
	}
	if exitIndex < 0 || levelIndex < 0 || levelIndex >= exitIndex {
		t.Fatalf("active exit must follow level-up state refresh: exit=%d level=%d frames=%x", exitIndex, levelIndex, frames)
	}
}

func TestEncodeFacultyActiveBeginUsesRegisteredHandlerAndActBegin(t *testing.T) {
	frame, err := encodeFacultyActiveBegin()
	if err != nil {
		t.Fatalf("encode active faculty begin: %v", err)
	}
	if len(frame) < 3 || frame[0] != 0x1E || binary.LittleEndian.Uint16(frame[1:]) != 3 {
		t.Fatalf("active faculty frame header=%x", frame)
	}
	// The first typed value is the derived current C++ receiver ID.  The next
	// two are Lua faculty.on_msg's arg1=22 (open act form) and arg2=0.
	needleID := []byte{2, byte(S2CFacultyMessage), 0, 0, 0}
	if !bytes.Contains(frame, needleID) {
		t.Fatalf("active faculty frame misses message id %d: %x", S2CFacultyMessage, frame)
	}
	needle := []byte{2, 22, 0, 0, 0, 2, 0, 0, 0, 0}
	if !bytes.Contains(frame, needle) {
		t.Fatalf("active faculty frame misses 22/0 args: %x", frame)
	}
}

func TestFacultyRestoreSettlesOfflineProgress(t *testing.T) {
	progress := newPlayerProgress()
	start := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	saved := progress.snapshot()
	saved.LastAdvanceUTC = start
	saved.Books[0].Fill = 100
	progress.restore(saved, start.Add(3*time.Minute))
	if got, want := progress.books[0].fill, int32(700); got != want {
		t.Fatalf("offline fill=%d want=%d", got, want)
	}
	if progress.facultyState != facultyStateConvert || progress.facultyStyle != facultyStyleNormal {
		t.Fatalf("restored faculty state=%d style=%d", progress.facultyState, progress.facultyStyle)
	}
}
