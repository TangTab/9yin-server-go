package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
	"time"
)

func TestPlayerActorPublishesYiHuaFactionAsSchool(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	if got := player.stateSnapshot().LogicState; got != 0 {
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
	rageCap := player.rageCap()
	sp, frame, err := player.setSP(999)
	if err != nil {
		t.Fatal(err)
	}
	if sp != rageCap || player.skillSP() != rageCap {
		t.Fatalf("set SP=%d actor=%d, want %d", sp, player.skillSP(), rageCap)
	}
	saturatedSP := []byte{byte(propSP & 0xff), byte(propSP >> 8), byte(rageCap), byte(rageCap >> 8), 0, 0}
	if frame[0] != 0x10 || !bytes.Contains(frame, saturatedSP) {
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

func TestPlayerVitalsIncludeMaxSPDisplayCap(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	// 官方方言:常规属性档案(0x0D 与所有周期 0x10 批)不得携带
	// MaxSP/MaxSPAdd——上限只在登录/切图全量批出现一次(49840 帧实证
	// 仅 #16 一处)。
	regular := make(map[string]int32)
	for _, property := range player.stateProperties() {
		regular[property.Name] = property.Value.I32
	}
	rageCap := player.rageCap()
	if got := regular["SP"]; got != rageCap {
		t.Fatalf("SP=%d, want %d", got, rageCap)
	}
	if _, exists := regular["MaxSP"]; exists {
		t.Fatal("regular property profile must not carry MaxSP")
	}
	if _, exists := regular["MaxSPAdd"]; exists {
		t.Fatal("regular property profile must not carry MaxSPAdd")
	}
	// 登录/切图链的首个权威 0x10(=stateProperties 大批 + MaxSP 折算
	// 最终值一次,无独立怒气帧):SP(467)=满值(有效上限),MaxSP(580)=
	// 基础 80 + 运行内功档位加成,MaxSPAdd(583) 按官服口径不出帧。
	frame, err := player.loginVitalUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x10 {
		t.Fatalf("login vital frame opcode=%#x", frame[0])
	}
	spNeedle := []byte{byte(propSP & 0xff), byte(propSP >> 8), byte(rageCap), 0, 0, 0}
	if !bytes.Contains(frame, spNeedle) {
		t.Fatalf("login vital batch misses SP=%d at index %d: %x", rageCap, propSP, frame)
	}
	maxSPNeedle := []byte{byte(580 & 0xff), byte(580 >> 8), byte(rageCap), 0, 0, 0}
	if !bytes.Contains(frame, maxSPNeedle) {
		t.Fatalf("login vital batch misses MaxSP=%d at index 580: %x", rageCap, frame)
	}
	loginIndices, walkErr := propertyFrameIndices(frame)
	if walkErr != nil {
		t.Fatalf("walk login 0x10 frame: %v", walkErr)
	}
	if containsIndex(loginIndices, propMaxSPAdd) {
		t.Fatalf("login vital batch carries MaxSPAdd(%d); official never sends it", propMaxSPAdd)
	}
	// 常规批不再携带上限。
	regularFrame, err := player.vitalUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(regularFrame, []byte{byte(580 & 0xff), byte(580 >> 8)}) {
		t.Fatal("regular vitals batch must not carry MaxSP")
	}
}

func TestStarterInnerPowerViewUsesNegotiatedBookFields(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	if got := player.stateSnapshot(); got.BaseMaxHP != 1000 || got.MaxHP != 3207 || got.BaseMaxMP != 500 || got.MaxMP != 1337 {
		t.Fatalf("两仪护心功 server caps=%+v, want base 1000/500 total 3207/1337", got)
	}
	if got := player.progress.books[0].staticData; got != 5 {
		t.Fatalf("ng_wd_001 StaticData=%d, want classic visible-buff value 5", got)
	}
	if got := player.bufferInfo(innerPowerBuffSlot, time.Now()); !bytes.HasPrefix([]byte(got), []byte("1742,1105C0E3,3DEFBAE,3,")) {
		t.Fatalf("level-30 neigong BufferInfo=%q", got)
	}
	if got := player.bufferListString(time.Now()); !bytes.Contains([]byte(got), []byte("buf_ng_wd_001,local-4,3,285589731-64945070,")) {
		t.Fatalf("level-30 neigong BufferListStr=%q", got)
	}
	// 官方全量公式（55866/12173 抓包解出的系数）：属性分量 + 内功 pack。
	// 五维线上值 75/61/43/76/43（base 30 + L30 varprop）：
	// MaxHPAdd=2·75+7·61+1630=2207、MaxMPAdd=4·76+43+490=837、
	// MinMagicDefAdd=round(0.2·43)+45=54。
	wantAdds := map[string]int32{
		"MaxHPAdd":       2207,
		"MaxMPAdd":       837,
		"MaxParryAdd":    840,
		"MinMagicDefAdd": 54,
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
		{byte(propStaticData & 0xff), byte(propStaticData >> 8)},
		{byte(propItemType & 0xff), byte(propItemType >> 8)},
		{byte(propMaxLevel & 0xff), byte(propMaxLevel >> 8)},
		{byte(propCurFillValue & 0xff), byte(propCurFillValue >> 8)},
		{byte(propMaxPowerValue & 0xff), byte(propMaxPowerValue >> 8)},
		// 内修面板属性列（太极/阴柔/阳刚）的显示来源：目录可用时
		// ng_wd_001→5；缺目录环境回 0（客户端退缺省文案），两者都合法。
		{byte(propWuXing & 0xff), byte(propWuXing >> 8)},
	} {
		if !bytes.Contains(first, expected) {
			t.Fatalf("first inner-power ViewAdd lacks %x: %x", expected, first)
		}
	}
}

// TestNeiGongBooksBackfillWuXingFromCatalog 钉死书行五行值来自
// wuxuebaseinfo.ini（Java NeiGongCatalog.wuXingOf 同源）：两仪护心功/
// 碧波心法=5 太极，明玉神功=6 阴柔。缺目录的夹具环境跳过值断言，
// 只验帧不崩。
func TestNeiGongBooksBackfillWuXingFromCatalog(t *testing.T) {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		t.Skipf("modern neigong catalog unavailable: %v", err)
	}
	if catalog.wuXingOf("ng_wd_001") == 0 {
		t.Skip("wuxuebaseinfo.ini not in resource store; catalog wuXing empty")
	}
	progress := newPlayerProgress()
	want := map[string]int32{
		"ng_wd_001": 5, // 两仪护心功：太极
		"ng_yh_001": 6, // 明玉神功：阴柔
		"ng_th_001": 5, // 碧波心法：太极
		"ng_jh_001": 6,
		"ng_em_001": 6,
		"ng_wd_002": 5,
	}
	for _, book := range progress.books {
		if got, ok := want[book.configID]; ok && book.wuXing != got {
			t.Errorf("%s wuXing=%d, want %d", book.configID, book.wuXing, got)
		}
	}
	frame, err := progress.bookFrame(1)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte{byte(propWuXing & 0xff), byte(propWuXing >> 8), 5, 0, 0, 0}
	if !bytes.Contains(frame, expected) {
		t.Fatalf("ng_wd_001 ViewAdd lacks WuXing=5: %x", frame)
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
	if got := player.bufferInfo(innerPowerBuffSlot, time.Now()); !bytes.HasPrefix([]byte(got), []byte("347A,1105C0E3,3DEFBAE,3,")) {
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
	if got := player.stateSnapshot(); got.BaseMaxHP != 1000 || got.MaxHP != 5829 || got.BaseMaxMP != 500 || got.MaxMP != 2235 {
		t.Fatalf("明玉神功 server caps=%+v, want base 1000/500 total 5829/2235", got)
	}
	if got := player.bufferInfo(innerPowerBuffSlot, time.Now()); !bytes.HasPrefix([]byte(got), []byte("3311,1105C0E3,3DEFBAE,3,")) {
		t.Fatalf("明玉神功 BufferInfo=%q", got)
	}
	if got := player.bufferListString(time.Now()); !bytes.Contains([]byte(got), []byte("buf_ng_yh_001,local-4,3,285589731-64945070,")) {
		t.Fatalf("明玉神功 BufferListStr=%q", got)
	}
	// 五维线上值 85/87/100/127/97：MaxHPAdd=2·85+7·87+4050=4829、
	// MaxMPAdd=4·127+97+1130=1735、MinMagicDefAdd=round(0.2·97)+100=119。
	wantAdds := map[string]int32{
		"MaxHPAdd":       4829,
		"MaxMPAdd":       1735,
		"MaxParryAdd":    5040,
		"MinMagicDefAdd": 119,
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

	// 215 必须只接受已学习的 View=43 秘籍，并用 CurNeiGong 绑定消费的
	// 常规属性档案回复。官服增量口径（2026-11-10 对齐）：MaxSP 专帧
	//（折算最终值,ng_jh_001 坐忘功 +10 → 90）→ 常规状态批 → 换书战斗
	// 面板增量（单属性 0x10：五维 Add/HP·MP/Parry/CurNeiGongWuXing
	// 变化项；官服运行期从不重推全量）。
	handled, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 215}, {Type: 6, Text: "ng_jh_001"},
	}})
	if err != nil || !handled {
		t.Fatalf("215 handled=%v err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) < 3 || frames[0][0] != 0x10 || frames[1][0] != 0x10 {
		t.Fatalf("215 reply=%x", frames)
	}
	capNeedle := []byte{byte(propMaxSP & 0xff), byte(propMaxSP >> 8), byte(maxPlayerSP + 10), 0, 0, 0}
	if !bytes.Contains(frames[0], capNeedle) {
		t.Fatalf("215 first reply must carry MaxSP=%d (坐忘功): %x", maxPlayerSP+10, frames[0])
	}
	if !bytes.Contains(frames[1], []byte("ng_jh_001\x00")) {
		t.Fatalf("215 second reply must bind CurNeiGong: %x", frames[1])
	}
	// 增量帧全部为单属性 0x10（len=18），换书至少改变五维与运行内功
	// 五行（CurNeiGongWuXing 654）。
	deltaProps := make(map[uint16]bool, len(frames)-2)
	for _, frame := range frames[2:] {
		if frame[0] != 0x10 || len(frame) != 18 {
			t.Fatalf("215 delta frame not single-property 0x10: %x", frame)
		}
		deltaProps[binary.LittleEndian.Uint16(frame[12:14])] = true
	}
	if !deltaProps[propCurNeiGongWuXing] {
		t.Fatalf("215 delta missing CurNeiGongWuXing: %x", frames)
	}
	if !deltaProps[propStr] {
		t.Fatalf("215 delta missing Str add change: %x", frames)
	}

	// 1006/1 选择一本秘籍；1006/11 启动持久的后台转换，而不进入可见的
	// 演武 LogicState=108 路线。
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
	// 0x10 isView=1 view=43 行级增量帧（裸 0x18 重发不触发客户端
	// updateitem，武学面板停在旧等级——2026-11-08 50514 抓包实证）。
	if readyView[0] != 0x10 || readyView[1] != 1 ||
		binary.LittleEndian.Uint32(readyView[2:]) != uint32(viewportNeiGong) {
		t.Fatalf("1006/1 first reply must update selected View=43 row: %x", readyView)
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
	emptyFacultyName := []byte{byte(propFacultyName & 0xff), byte(propFacultyName >> 8), 1, 0, 0, 0, 0}
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
	if got := player.stateSnapshot().LogicState; got != 0 {
		t.Fatalf("normal faculty LogicState=%d, want background state 0", got)
	}
	if player.progress.facultyState != facultyStateConvert || player.progress.facultyStyle != facultyStyleNormal {
		t.Fatalf("normal faculty state=%d style=%d", player.progress.facultyState, player.progress.facultyStyle)
	}
	frames = conn.Frames()
	viewUpdated := false
	for _, frame := range frames {
		if len(frame) >= 12 && frame[0] == 0x10 && frame[1] == 1 &&
			binary.LittleEndian.Uint32(frame[2:]) == uint32(viewportNeiGong) {
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
	if got := player.stateSnapshot().LogicState; got != 0 {
		t.Fatalf("LogicState after exit=%d, want 0", got)
	}
	if got := player.progress.facultyState; got != facultyStateNone {
		t.Fatalf("FacultyState after exit=%d, want client-rendered idle=%d", got, facultyStateNone)
	}

	// 修炼面板的演武按钮同样向 C->S 1006 总线发出 sub=21，但其原生显示
	// 状态是 FacultyStyle=2 而非 normal=1。
	handled, err = handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 1006}, {Type: 2, Int32: 21},
	}})
	if err != nil || !handled {
		t.Fatalf("1006/21 handled=%v err=%v", handled, err)
	}
	if got := player.progress.facultyStyle; got != facultyStyleAct {
		t.Fatalf("FacultyStyle after 1006/21=%d, want active=%d", got, facultyStyleAct)
	}
	if got := player.stateSnapshot().LogicState; got != logicStateFaculty {
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
	// 升级必须先从权威的 View=43 和角色属性刷新修炼对话框。之后才能
	// 关闭主动页面；其原生的 exit_flag 仍会抑制过期的 C2S 1006/22
	// 回调。
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
	// 升级路径必须先发 View=43 行级增量帧（0x10 isView=1——裸 0x18 重发
	// 不触发客户端 updateitem，武学面板停在旧等级）和怒气上限专帧，再发
	// 角色状态与原生升级通知。
	updateIndex, capIndex := -1, -1
	for index, frame := range frames {
		if len(frame) >= 12 && frame[0] == 0x10 && frame[1] == 1 &&
			binary.LittleEndian.Uint32(frame[2:]) == uint32(viewportNeiGong) {
			updateIndex = index
		}
		if len(frame) >= 12 && frame[0] == 0x10 && frame[1] == 0 &&
			bytes.Contains(frame, []byte{byte(propMaxSP & 0xff), byte(propMaxSP >> 8), byte(maxPlayerSP + 10), 0, 0, 0}) {
			capIndex = index
		}
	}
	if updateIndex < 0 {
		t.Fatalf("level-up must refresh the View=43 row via 0x10 isView=1: %x", frames)
	}
	if capIndex < 0 || capIndex > levelIndex {
		t.Fatalf("rage cap frame must precede the level-up notice: cap=%d level=%d", capIndex, levelIndex)
	}
	if updateIndex > capIndex || capIndex > levelIndex {
		t.Fatalf("refresh order update=%d cap=%d level=%d", updateIndex, capIndex, levelIndex)
	}
}

// bookUpdateFrame 的方言钉死：0x10 isView=1 view=43 owner=行号，六属性
// 集 Level(byte)/MaxLevel/CurFill/TotalFill/Power/MaxPower——与官服
// e.pcapng 既有视图行更新帧（0x10 isView=1 view=40 行）同构。
func TestNeiGongBookUpdateFrameIsViewRowPropertyUpdate(t *testing.T) {
	progress := newPlayerProgress()
	progress.books[0].level = 31
	progress.books[0].fill = 4321
	frame, err := progress.bookUpdateFrame(1)
	if err != nil {
		t.Fatalf("bookUpdateFrame: %v", err)
	}
	if frame[0] != 0x10 || frame[1] != 1 {
		t.Fatalf("header=%x want 0x10 isView=1", frame[:2])
	}
	if got := binary.LittleEndian.Uint32(frame[2:]); got != uint32(viewportNeiGong) {
		t.Fatalf("view=%d want %d", got, viewportNeiGong)
	}
	if got := binary.LittleEndian.Uint32(frame[6:]); got != 1 {
		t.Fatalf("row=%d want 1", got)
	}
	if count := binary.LittleEndian.Uint16(frame[10:]); count != 6 {
		t.Fatalf("property count=%d want 6", count)
	}
	if !bytes.Contains(frame, []byte{byte(propJMLevel & 0xff), byte(propJMLevel >> 8), 31}) {
		t.Fatalf("frame missing Level=31 byte: %x", frame)
	}
	if !bytes.Contains(frame, []byte{byte(propCurFillValue & 0xff), byte(propCurFillValue >> 8), 0xe1, 0x10, 0, 0}) {
		t.Fatalf("frame missing CurFillValue=4321: %x", frame)
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
	// 第一个类型化值是推导出的当前 C++ 接收器 ID。后两个是 Lua
	// faculty.on_msg 的 arg1=22（打开主动表单）和 arg2=0。
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

// 快照携带 starter 目录之外的书（ng_jh_201，快照值取自参考服抓包：
// L7/total 48000）：从 neigong.ini 目录回填身份后追加，进度按快照覆盖；
// 每级消耗随后对齐 wuxuelevelinfo.ini 官方曲线（L7 → r=7=184400，
// 48000 是参考服翻倍曲线值，不采用），且 snapshot() 回写不丢。
func TestFacultyRestoreAppendsCatalogBackfilledBooks(t *testing.T) {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		t.Skip("modern neigong catalog unavailable")
	}
	entry, ok := catalog.book("ng_jh_201")
	if !ok {
		t.Skip("neigong.ini has no ng_jh_201 (catalog drifted)")
	}
	wantTotal, curveOK := resolveNeiGongLevelTotal("ng_jh_201", 7)
	if !curveOK {
		wantTotal = 48000 // 缺曲线环境保留快照值
	}
	progress := newPlayerProgress()
	starterCount := len(progress.books)
	saved := progress.snapshot()
	saved.CurNeiGong = "ng_jh_201"
	saved.FacultyName = "ng_jh_201"
	saved.Books = append(saved.Books, facultyBookSnapshot{
		ConfigID: "ng_jh_201", Level: 7, Fill: 44254, Total: 48000, Power: 240, MaxPower: 240,
	})
	progress.restore(saved, time.Now())
	if len(progress.books) != starterCount+1 {
		t.Fatalf("books=%d, want %d (+1 backfilled)", len(progress.books), starterCount+1)
	}
	book := progress.book("ng_jh_201")
	if book == nil {
		t.Fatal("ng_jh_201 must be restored as a book")
	}
	if book.staticData != entry.staticData || book.itemType != entry.itemType {
		t.Fatalf("backfilled identity=%+v, want catalog entry %+v", book, entry)
	}
	if book.level != 7 || book.fill != 44254 || book.total != wantTotal || book.power != 240 {
		t.Fatalf("snapshot progress must overlay: %+v", book)
	}
	if book.maxLevel < 7 {
		t.Fatalf("maxLevel=%d must cover snapshot level 7", book.maxLevel)
	}
	if progress.curNeiGong != "ng_jh_201" || progress.facultyName != "ng_jh_201" {
		t.Fatalf("cur/faculty targets must resolve to the backfilled book: cur=%s name=%s", progress.curNeiGong, progress.facultyName)
	}
	roundTrip := progress.snapshot()
	found := false
	for _, savedBook := range roundTrip.Books {
		if savedBook.ConfigID == "ng_jh_201" && savedBook.Level == 7 && savedBook.Total == wantTotal {
			found = true
		}
	}
	if !found {
		t.Fatalf("snapshot round-trip lost ng_jh_201: %+v", roundTrip.Books)
	}
}

// neigong.ini 的经典节（可见 Buff 路线）优先；mind_ 重复节只在经典
// 节缺席时兜底，且不覆盖任何已有条目。
func TestMergeNeiGongBookSectionPrefersClassicRoute(t *testing.T) {
	books := make(map[string]modernNeiGongBook)
	mergeNeiGongBookSection(books, "mind_ng_wd_001", map[string]string{
		"QName": "ng_wd_001", "StaticData": "1006", "ItemType": "1002", "School": "ng_wd",
	})
	if entry, ok := books["ng_wd_001"]; !ok || entry.staticData != 1006 {
		t.Fatalf("mind_-only book must backfill: %+v", entry)
	}
	mergeNeiGongBookSection(books, "ng_wd_001", map[string]string{
		"QName": "ng_wd_001", "StaticData": "5", "ItemType": "1002", "School": "ng_wd",
	})
	if entry := books["ng_wd_001"]; entry.staticData != 5 {
		t.Fatalf("classic section must override mind_ placeholder: %+v", entry)
	}
	mergeNeiGongBookSection(books, "mind_ng_wd_001", map[string]string{
		"QName": "ng_wd_001", "StaticData": "1006", "ItemType": "1002",
	})
	if entry := books["ng_wd_001"]; entry.staticData != 5 {
		t.Fatalf("late mind_ section must not override classic: %+v", entry)
	}
	mergeNeiGongBookSection(books, "[broken", map[string]string{"StaticData": "1"})
	mergeNeiGongBookSection(books, "ng_xx_001", map[string]string{"StaticData": "0"})
	if len(books) != 1 {
		t.Fatalf("non-ng/malformed sections must be ignored: %v", books)
	}
}

func TestSnapshotExtraInt(t *testing.T) {
	parsed := map[string]json.RawMessage{"max_level": json.RawMessage("71"), "name": json.RawMessage(`"x"`), "frac": json.RawMessage("1.5")}
	if got := snapshotExtraInt(parsed, "max_level"); got != 71 {
		t.Fatalf("max_level=%d, want 71", got)
	}
	if got := snapshotExtraInt(parsed, "name"); got != 0 {
		t.Fatalf("string extra must read 0, got %d", got)
	}
	if got := snapshotExtraInt(parsed, "frac"); got != 0 {
		t.Fatalf("fractional extra must read 0, got %d", got)
	}
	if got := snapshotExtraInt(parsed, "missing"); got != 0 {
		t.Fatalf("missing extra must read 0, got %d", got)
	}
}

// 每级修炼消耗统一走 wuxuelevelinfo.ini 官方曲线（含江湖内功；参考服
// 9yin_server1 的 750×2^(L-1) 翻倍曲线数值夸张，刻意不镜像）。锚点：
// ng_jh_001 L1=750/L2=1650/L7=13950、ng_jh_201 L1=92900/L14=330750、
// ng_wd_001 L30=273200。夹具仓缺 wuxuelevelinfo.ini 时跳过。
func TestNeiGongLevelCurveFollowsOfficialINI(t *testing.T) {
	if _, ok := resolveNeiGongLevelTotal("ng_jh_001", 7); !ok {
		t.Skip("wuxuelevelinfo.ini absent from resource store")
	}
	cases := []struct {
		configID string
		level    int32
		want     int32
	}{
		{"ng_jh_001", 1, 750},
		{"ng_jh_001", 2, 1650},
		{"ng_jh_001", 7, 13950},
		{"ng_jh_201", 1, 92900},
		{"ng_jh_201", 14, 330750},
		{"ng_jh_201", 25, 642100},
		{"ng_wd_001", 30, 273200},
		{"ng_yh_001", 30, 579600},
	}
	for _, tc := range cases {
		got, ok := resolveNeiGongLevelTotal(tc.configID, tc.level)
		if !ok || got != tc.want {
			t.Fatalf("%s L%d total=%d ok=%t, want %d", tc.configID, tc.level, got, ok, tc.want)
		}
	}
}

// 升层结算把每级消耗推进到曲线值（ng_jh_001 L2=1650），与 +200 自然
// 曲线（缺曲线环境 950）明确区分。
func TestAdvanceFacultyLevelsJiangHuBookAlongCurve(t *testing.T) {
	progress := newPlayerProgress()
	book := progress.book("ng_jh_001")
	if book == nil {
		t.Fatal("starter must include ng_jh_001")
	}
	wantL2 := int32(950) // 缺曲线回退：750+200
	if _, ok := resolveNeiGongLevelTotal("ng_jh_001", 2); ok {
		wantL2 = 1650 // 官方曲线 wuxuelevelinfo r=2
	}
	if got, want := book.total, int32(750); got != want {
		t.Fatalf("ng_jh_001 L1 total=%d, want 750 (官方曲线 r=1)", got)
	}
	start := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	progress.facultyName = book.configID
	progress.facultyState = facultyStateConvert
	progress.facultyStyle = facultyStyleNormal
	progress.fillSpeed = 200
	progress.lastAdvance = start
	book.fill = book.total - 200 // 一分钟修炼正好攒满本层。
	progress.advanceFaculty(start.Add(time.Minute))
	if got, want := book.level, int32(2); got != want {
		t.Fatalf("level=%d, want %d", got, want)
	}
	if got, want := book.total, wantL2; got != want {
		t.Fatalf("L2 total=%d, want %d", got, want)
	}
	if book.fill != 0 {
		t.Fatalf("fill=%d, want 0 (fill 750 正好攒满本层)", book.fill)
	}
}

// 快照 total 可能来自旧 +200 自然曲线/旧服写盘：restore 后按曲线权威
// 值重求再下发（ng_jh_001 L7=13950）；缺曲线环境保留快照值（1900）。
// fill 进度保留。
func TestFacultyRestoreRealignsLevelTotalsToCurve(t *testing.T) {
	progress := newPlayerProgress()
	restoreAt := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	saved := progress.snapshot()
	saved.LastAdvanceUTC = restoreAt // 结算 0 分钟，别让计时器动 fill。
	for index := range saved.Books {
		if saved.Books[index].ConfigID == "ng_jh_001" {
			saved.Books[index].Level = 7
			saved.Books[index].Total = 1900 // 旧 +200 自然曲线值
			saved.Books[index].Fill = 13950 // 官方 L7 整层，重求后正好攒满。
		}
	}
	progress.restore(saved, restoreAt)
	book := progress.book("ng_jh_001")
	if _, ok := resolveNeiGongLevelTotal("ng_jh_001", 7); ok {
		if book.level != 7 || book.total != 13950 {
			t.Fatalf("ng_jh_001 restored level=%d total=%d, want 7/13950 (wuxuelevelinfo.ini r=7)", book.level, book.total)
		}
	} else if book.total != 1900 {
		t.Fatalf("缺曲线环境须保留快照 total，got %d", book.total)
	}
	if book.fill != 13950 {
		t.Fatalf("fill=%d, want snapshot progress preserved", book.fill)
	}
}

// 官方抓包黄金对齐：五维→衍生 *Add 位的系数由 55866 系列抓包逐点解出
// （base 30×5 + ng_jh_001 装备）。两组真机状态：
//
//	12173（创角会话）: L1 线上五维 32/33/32/32/32 → MaxHPAdd=335、
//	  MaxMPAdd=170、MinMagicDefAdd=6、HPUpSpeed=134、MPUpSpeed=67、
//	  HPUpSpeedAdd=2、MPUpSpeedAdd=6；
//	55866（myjf01）: L7 线上五维 36/39/35/36/35 → MaxHPAdd=495、
//	  MaxMPAdd=204、MinMagicDefAdd=7、HPUpSpeed=150、MPUpSpeed=71、
//	  HPUpSpeedAdd=3、MPUpSpeedAdd=7。
//
// 官方 MaxParryAdd 另有 16/57 的未定位分量（非 ng_jh_001 varprop/pack，
// 疑门派被动），本服 ParryValue 仅 100+装备书分量，该差已知不在此断言。
func TestAttributeDerivedAddsMatchOfficialCapture(t *testing.T) {
	cases := []struct {
		name        string
		level       int32
		five        map[string]int32
		derivedAdds map[string]int32
		maxHP       int32
		maxMP       int32
		hpUpSpeed   int32
		mpUpSpeed   int32
		hpUpAdd     int32
		mpUpAdd     int32
	}{
		{
			name:  "12173 创角 ng_jh_001 L1",
			level: 1,
			five:  map[string]int32{"Str": 32, "Sta": 33, "Dex": 32, "Ing": 32, "Spi": 32},
			derivedAdds: map[string]int32{
				"MeleePowerAdd": 32, "ShotPowerAdd": 32, "MagicPowerAdd": 32,
				"PhyHitAdd": 32, "MagicHitAdd": 32, "DodgeAdd": 32,
				"PhyVaAdd": 32, "MagicVaAdd": 32,
				"MaxHPAdd": 335, "MaxMPAdd": 170, "MinMagicDefAdd": 6,
			},
			maxHP: 1335, maxMP: 670, hpUpSpeed: 134, mpUpSpeed: 67, hpUpAdd: 2, mpUpAdd: 6,
		},
		{
			name:  "55866 myjf01 ng_jh_001 L7",
			level: 7,
			five:  map[string]int32{"Str": 36, "Sta": 39, "Dex": 35, "Ing": 36, "Spi": 35},
			derivedAdds: map[string]int32{
				"MeleePowerAdd": 36, "ShotPowerAdd": 35, "MagicPowerAdd": 36,
				"PhyHitAdd": 35, "MagicHitAdd": 35, "DodgeAdd": 35,
				"PhyVaAdd": 36, "MagicVaAdd": 35,
				"MaxHPAdd": 495, "MaxMPAdd": 204, "MinMagicDefAdd": 7,
			},
			maxHP: 1495, maxMP: 704, hpUpSpeed: 150, mpUpSpeed: 71, hpUpAdd: 3, mpUpAdd: 7,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			player := newPlayerActor("本地角色", 0)
			player.mu.Lock()
			book := player.progress.book("ng_jh_001")
			if book == nil {
				player.mu.Unlock()
				t.Fatal("starter must include ng_jh_001")
			}
			book.level = testCase.level
			applyNeiGongLevelTotal(book)
			_ = book.applyModernEffect()
			player.mu.Unlock()
			if err := player.equipNeiGong("ng_jh_001"); err != nil {
				t.Fatal(err)
			}
			got := map[string]int32{}
			for _, property := range player.progressProperties() {
				got[property.Name] = property.Value.I32
			}
			for name, want := range testCase.five {
				if got[name] != want {
					t.Errorf("%s=%d, want %d", name, got[name], want)
				}
			}
			for name, want := range testCase.derivedAdds {
				if got[name] != want {
					t.Errorf("%s=%d, want %d", name, got[name], want)
				}
			}
			state := player.stateSnapshot()
			if state.MaxHP != testCase.maxHP || state.MaxMP != testCase.maxMP {
				t.Errorf("vitals derived caps HP=%d MP=%d, want %d/%d", state.MaxHP, state.MaxMP, testCase.maxHP, testCase.maxMP)
			}
			stateValues := map[string]int32{}
			for _, property := range player.stateProperties() {
				stateValues[property.Name] = property.Value.I32
			}
			for name, want := range map[string]int32{
				"HPUpSpeed": testCase.hpUpSpeed, "MPUpSpeed": testCase.mpUpSpeed,
				"HPUpSpeedAdd": testCase.hpUpAdd, "MPUpSpeedAdd": testCase.mpUpAdd,
			} {
				if stateValues[name] != want {
					t.Errorf("%s=%d, want %d", name, stateValues[name], want)
				}
			}
		})
	}
}

// TestZhenpaiParryAddLandsInAddColumn 镇派兵器"招架耐力上限"必须落
// MaxParryAdd@477 Add 列（客户端上限=MaxParry+MaxParryAdd 推导），
// 不能只折进 ParryValue@475 总量位——2026-11-12 真机「加的招架耐力
// 的上限没变」根因；475 恒等于 476+477 推导和。
func TestZhenpaiParryAddLandsInAddColumn(t *testing.T) {
	player := newPlayerActor("镇派侠", 0)
	player.mu.Lock()
	player.progress.zhenpaiEquip = zhenpaiEquipBonus{parry: 30000}
	base := player.progress.attributes.maxParry
	bookAdd := int32(0)
	if equipped := player.progress.book(player.progress.curNeiGong); equipped != nil {
		bookAdd = equipped.maxParryAdd
	}
	player.mu.Unlock()

	props := map[string]int32{}
	for _, property := range player.progressProperties() {
		props[property.Name] = property.Value.I32
	}
	if got := props["MaxParryAdd"]; got != bookAdd+30000 {
		t.Fatalf("MaxParryAdd=%d, want 书分量 %d + 镇派 30000（Add 列须含镇派）", got, bookAdd)
	}
	// 475 在 stateProperties() 域（progressProperties 不带 ParryValue）。
	stateProps := map[string]int32{}
	for _, property := range player.stateProperties() {
		stateProps[property.Name] = property.Value.I32
	}
	if got := stateProps["ParryValue"]; got != base+bookAdd+30000 {
		t.Fatalf("ParryValue=%d, want MaxParry %d + MaxParryAdd %d", got, base, bookAdd+30000)
	}
	if got, want := stateProps["ParryValue"], props["MaxParry"]+props["MaxParryAdd"]; got != want {
		t.Fatalf("ParryValue=%d != MaxParry+MaxParryAdd=%d（475 ≠ 476+477 推导和）", got, want)
	}

	// 面板补帧须带 477：换装/换实力档瞬间客户端 Add 列才有刷新源。
	sink := &frameSink{}
	if err := player.pushZhenpaiPanelFrame(sink); err != nil {
		t.Fatal(err)
	}
	if len(sink.frames) != 1 {
		t.Fatalf("panel frames=%d, want 1", len(sink.frames))
	}
	frame := sink.frames[0]
	if frame[0] != 0x10 {
		t.Fatalf("frame head=%#x, want 0x10", frame[0])
	}
	count := int(binary.LittleEndian.Uint16(frame[10:]))
	got := map[uint16]int32{}
	offset := 12
	for i := 0; i < count; i++ {
		if offset+6 > len(frame) {
			t.Fatalf("frame truncated at prop %d", i)
		}
		index := binary.LittleEndian.Uint16(frame[offset:])
		// 本帧五属性（581/582/475/477/694）全 Int32 线型。
		got[index] = int32(binary.LittleEndian.Uint32(frame[offset+2:]))
		offset += 6
	}
	if v, ok := got[propMaxParryAdd]; !ok || v != bookAdd+30000 {
		t.Fatalf("panel frame MaxParryAdd@%d=%d(ok=%t), want %d", propMaxParryAdd, v, ok, bookAdd+30000)
	}
	if v, ok := got[propParryValue]; !ok || v != base+bookAdd+30000 {
		t.Fatalf("panel frame ParryValue@%d=%d(ok=%t), want %d", propParryValue, v, ok, base+bookAdd+30000)
	}
}
