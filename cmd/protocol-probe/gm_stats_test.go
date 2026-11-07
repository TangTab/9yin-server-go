package main

import (
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"
)

// TestGMStatsThinFrames 逐帧钉死 GM 面板域瘦帧的形（opcode/对象身份/
// 属性序/值编码），全部对齐 Java LoginFlow applyXxx 的下发序。
func TestGMStatsThinFrames(t *testing.T) {
	player := newPlayerActor("测试侠", 0)

	t.Run("guanyin capital frame", func(t *testing.T) {
		player.setGuanYin(4567)
		frame, err := player.guanYinUpdate()
		if err != nil {
			t.Fatal(err)
		}
		if len(frame) != 22 { // 12 头 + 2 索引 + 8 int64
			t.Fatalf("frame len=%d want 22", len(frame))
		}
		if frame[0] != 0x10 || binary.LittleEndian.Uint16(frame[12:]) != propCapitalType2 {
			t.Fatalf("unexpected frame head % x", frame[:14])
		}
		if got := binary.LittleEndian.Uint64(frame[14:]); got != 4567 {
			t.Fatalf("CapitalType2=%d want 4567", got)
		}
	})

	t.Run("power level frame", func(t *testing.T) {
		frame, err := player.setPowerLevel(126)
		if err != nil {
			t.Fatal(err)
		}
		indices, walkErr := propertyFrameIndices(frame)
		if walkErr != nil {
			t.Fatal(walkErr)
		}
		if len(indices) != 2 || indices[0] != propPowerLevel || indices[1] != propLevelTitle {
			t.Fatalf("indices=%v want [PowerLevel LevelTitle]", indices)
		}
		// LevelTitle(126) 按 roleListTitle 分档 = title025（121..130 段）。
		// 布局：[12 头][u16 210][i32][u16 801][u32 len(含 NUL)][UTF-8][NUL]。
		offset := 12 + 2 + 4 + 2
		length := int(binary.LittleEndian.Uint32(frame[offset:]))
		text := string(frame[offset+4 : offset+4+length-1])
		if text != "title025" {
			t.Fatalf("LevelTitle=%q want title025", text)
		}
		if frame[offset+4+length-1] != 0 {
			t.Fatal("LevelTitle missing NUL terminator")
		}
	})

	t.Run("alignment frame ordered subset", func(t *testing.T) {
		frame, err := player.setAlignment(0, 7, 3) // justice=0 跳过设置，但帧常发
		if err != nil {
			t.Fatal(err)
		}
		indices, walkErr := propertyFrameIndices(frame)
		if walkErr != nil {
			t.Fatal(walkErr)
		}
		want := []uint16{propJustice, propEvil, propFreeBase}
		if len(indices) != len(want) {
			t.Fatalf("indices=%v want %v", indices, want)
		}
		for i := range want {
			if indices[i] != want[i] {
				t.Fatalf("indices=%v want %v", indices, want)
			}
		}
	})

	t.Run("pk mode frame is byte wire", func(t *testing.T) {
		frame, err := player.setPKMode(3)
		if err != nil {
			t.Fatal(err)
		}
		// 12 头 + 2 索引 + 1 字节：PKMode 官方线型 t1。
		if len(frame) != 15 {
			t.Fatalf("frame len=%d want 15 (byte payload)", len(frame))
		}
		if frame[14] != 3 {
			t.Fatalf("PKMode byte=%d want 3", frame[14])
		}
	})

	t.Run("vitals frame fixed order", func(t *testing.T) {
		frame, err := player.setGMVitals(gmVitalsState{ene: 80, maxEne: 100, resumeEne: 2, sat: 60, maxSat: 100, pkValue: 4, drunkLevel: 1})
		if err != nil {
			t.Fatal(err)
		}
		indices, walkErr := propertyFrameIndices(frame)
		if walkErr != nil {
			t.Fatal(walkErr)
		}
		want := []uint16{propEne, propMaxEne, propResumeEne, propSat, propMaxSat, propDrunkLevel, propPKValue}
		if len(indices) != len(want) {
			t.Fatalf("indices=%v want %v", indices, want)
		}
		for i := range want {
			if indices[i] != want[i] {
				t.Fatalf("indices=%v want %v", indices, want)
			}
		}
	})

	t.Run("fill speed frame", func(t *testing.T) {
		actual, frame, err := player.setFillSpeed(300, 1.5)
		if err != nil {
			t.Fatal(err)
		}
		if actual != 450 {
			t.Fatalf("actual fill speed=%d want 450", actual)
		}
		indices, walkErr := propertyFrameIndices(frame)
		if walkErr != nil {
			t.Fatal(walkErr)
		}
		if len(indices) != 2 || indices[0] != propFillSpeed || indices[1] != propNorFacultyAdd {
			t.Fatalf("indices=%v want [FillSpeed NorFacultyAdd]", indices)
		}
	})

	t.Run("act day limit frames", func(t *testing.T) {
		// 500 两 → Max=500000 文；已耗/已用重置为 0。
		display, frame, err := player.setActDayLimit(500)
		if err != nil {
			t.Fatal(err)
		}
		if display != 500 {
			t.Fatalf("display=%d want 500", display)
		}
		indices, walkErr := propertyFrameIndices(frame)
		if walkErr != nil {
			t.Fatal(walkErr)
		}
		want := []uint16{propActDayValue, propActCostSuiYin, propActDayValueLast, propActDayValueUsed, propActDayValueMax}
		if len(indices) != len(want) {
			t.Fatalf("indices=%v want %v", indices, want)
		}
		// 不限模式：Max=0、Value=Last=999999999。
		_, frame, err = player.setActDayLimit(0)
		if err != nil {
			t.Fatal(err)
		}
		values := map[uint16]int32{}
		offset := 12
		for i := 0; i < 5; i++ {
			index := binary.LittleEndian.Uint16(frame[offset:])
			offset += 2
			values[index] = int32(binary.LittleEndian.Uint32(frame[offset:]))
			offset += 4
		}
		if values[propActDayValueMax] != 0 || values[propActDayValue] != actDayUnlimitedSentinel || values[propActDayValueLast] != actDayUnlimitedSentinel {
			t.Fatalf("unlimited mode values=%v", values)
		}
	})

	t.Run("faculty full frame carries Faculty", func(t *testing.T) {
		frame, err := player.setFacultyValue(777_888)
		if err != nil {
			t.Fatal(err)
		}
		indices, walkErr := propertyFrameIndices(frame)
		if walkErr != nil {
			t.Fatal(walkErr)
		}
		found := false
		for _, index := range indices {
			if index == propFaculty {
				found = true
			}
		}
		if !found {
			t.Fatalf("full vital frame missing Faculty@%d (indices=%d entries)", propFaculty, len(indices))
		}
	})

	t.Run("base attributes overwrite", func(t *testing.T) {
		if _, err := player.setBaseAttributes(100, 0, 80, 0, 90); err != nil {
			t.Fatal(err)
		}
		player.mu.Lock()
		str, sta, dex := player.progress.attributes.str, player.progress.attributes.sta, player.progress.attributes.dex
		player.mu.Unlock()
		if str != 100 || sta != 30 || dex != 80 {
			t.Fatalf("str=%d sta=%d dex=%d want str=100 sta=30(未跳过维保持) dex=80", str, sta, dex)
		}
	})
}

// TestPanelIdentityDomainFrames 钉死面板身份/域块常驻帧：种子实力 151
// （与选人界面 f23 同档）、属性序与 Java PanelState.identityProperties/
// domainProperties 一致；GM 修改后重放帧携带新值（缺发则人物面板整块
// 空白——实力一行不显示的根因）。
func TestPanelIdentityDomainFrames(t *testing.T) {
	player := newPlayerActor("测试侠", 0)
	frames, err := player.gmPanelFrames()
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 { // 身份块 + 域块
		t.Fatalf("panel frames=%d want 2", len(frames))
	}
	for _, frame := range frames {
		if frame[0] != 0x10 {
			t.Fatalf("panel frame opcode=%#x", frame[0])
		}
	}

	identityIndices, walkErr := propertyFrameIndices(frames[0])
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	identityWant := []uint16{
		propPowerLevel, propLevelTitle, propCharacterFlag, propCharacterValue,
		propJustice, propEvil, propFreeBase,
		propEne, propMaxEne, propResumeEne, propSat, propMaxSat,
		propPKValue, propDrunkLevel, propPKMode,
	}
	if len(identityIndices) != len(identityWant) {
		t.Fatalf("identity indices=%v want %v", identityIndices, identityWant)
	}
	for i := range identityWant {
		if identityIndices[i] != identityWant[i] {
			t.Fatalf("identity indices=%v want %v", identityIndices, identityWant)
		}
	}
	// 种子实力 151（title027 返璞归真，选人界面 f23 同档）。
	if got := binary.LittleEndian.Uint32(frames[0][14:18]); got != roleListPowerLevel {
		t.Fatalf("seed PowerLevel=%d want %d", got, roleListPowerLevel)
	}
	if !strings.Contains(string(frames[0]), "title027") {
		t.Fatalf("identity frame missing seed LevelTitle title027: % x", frames[0])
	}

	domainIndices, walkErr := propertyFrameIndices(frames[1])
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	domainWant := []uint16{
		propTitleID, propMount, propCurRideMount,
		propBingLuType, propMaxBingLuLevel, propPrestigeExp,
	}
	if len(domainIndices) != len(domainWant) {
		t.Fatalf("domain indices=%v want %v", domainIndices, domainWant)
	}
	for i := range domainWant {
		if domainIndices[i] != domainWant[i] {
			t.Fatalf("domain indices=%v want %v", domainIndices, domainWant)
		}
	}

	// GM 修改后重放：身份块携带新实力档与 PK 模式（会话内切图不丢状态）。
	if _, err := player.setPowerLevel(60); err != nil {
		t.Fatal(err)
	}
	if _, err := player.setPKMode(1); err != nil {
		t.Fatal(err)
	}
	if _, err := player.setGMVitals(gmVitalsState{ene: 80, maxEne: 100, resumeEne: 2, sat: 60, maxSat: 100, pkValue: 4, drunkLevel: 1}); err != nil {
		t.Fatal(err)
	}
	frames, err = player.gmPanelFrames()
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(frames[0][14:18]); got != 60 {
		t.Fatalf("replayed PowerLevel=%d want 60", got)
	}
	if !strings.Contains(string(frames[0]), "title012") {
		t.Fatalf("replayed LevelTitle want title012（60 级档：(60-1)/5+1=12）: % x", frames[0])
	}
	if strings.Contains(string(frames[0]), "title027") {
		// 重放帧不应残留种子档。
		t.Fatal("replayed identity still carries seed title027")
	}
	// PKMode 线型 t1（byte）：帧尾字节 = 1。
	if frames[0][len(frames[0])-1] != 1 {
		t.Fatalf("replayed PKMode byte=%d want 1", frames[0][len(frames[0])-1])
	}
}

// TestFacultySnapshotGMPanelRoundtrip 走一遍 GM 面板域的 role_faculty
// 快照往返：JSON 键必须与 Java FacultyStore 逐字一致（共享 MySQL 三服
// 互读），旧档（无这些键）还原回退默认值。
func TestFacultySnapshotGMPanelRoundtrip(t *testing.T) {
	player := newPlayerActor("测试侠", 0)
	if _, _, err := player.setFillSpeed(300, 1.5); err != nil {
		t.Fatal(err)
	}
	if _, err := player.setBaseAttributes(120, 110, 100, 90, 80); err != nil {
		t.Fatal(err)
	}
	if _, _, err := player.setActDayLimit(500); err != nil {
		t.Fatal(err)
	}
	if _, err := player.setGMVitals(gmVitalsState{ene: 88, maxEne: 100, resumeEne: 3, sat: 66, maxSat: 100, pkValue: 5, drunkLevel: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := player.setFacultyValue(1_234_567); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(player.facultySnapshot())
	if err != nil {
		t.Fatal(err)
	}
	document := string(raw)
	for _, key := range []string{
		`"fill_speed_base":300`, `"fill_speed_multiplier":1.5`,
		`"act_day_value":500000`, `"act_day_value_max":500000`, `"act_day_value_used":0`,
		`"base_str":120`, `"base_sta":110`, `"base_dex":100`, `"base_ing":90`, `"base_spi":80`,
		`"vitals":{"ene":88,"max_ene":100,"resume_ene":3,"sat":66,"max_sat":100,"pk_value":5,"drunk_level":2,`,
		`"faculty":1234567`,
	} {
		if !strings.Contains(document, key) {
			t.Fatalf("snapshot JSON missing %s: %s", key, document)
		}
	}

	// 还原到新 actor：GM 面板域逐值回种。
	var restored facultyProgressSnapshot
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	target := newPlayerActor("还原侠", 0)
	target.restoreFaculty(restored, restored.LastAdvanceUTC)
	if got := target.facultyValue(); got != 1_234_567 {
		t.Fatalf("restored faculty=%d", got)
	}
	target.mu.Lock()
	base, mult := target.progress.fillSpeedBase, target.progress.fillSpeedMultiplier
	maxWen := target.progress.actDayValueMax
	str := target.progress.attributes.str
	vitals, vitalsSet := target.gm.vitals, target.gm.vitalsSet
	target.mu.Unlock()
	if base != 300 || mult != 1.5 {
		t.Fatalf("restored fill speed split=%d×%v", base, mult)
	}
	if maxWen != 500000 {
		t.Fatalf("restored act day max=%d want 500000", maxWen)
	}
	if str != 120 {
		t.Fatalf("restored base str=%d want 120", str)
	}
	if !vitalsSet || vitals.ene != 88 || vitals.pkValue != 5 || vitals.drunkLevel != 2 {
		t.Fatalf("restored vitals=%+v set=%v", vitals, vitalsSet)
	}

	// 旧档（无 GM 面板域键）：还原回退默认，不覆盖运行时值。
	legacy := []byte(`{"cur_neigong":"ng_wd_001","faculty":42,"faculty_state":2,"faculty_style":1,"faculty_name":"ng_wd_001","fill_speed":200,"last_advance_utc":"2026-11-07T00:00:00Z","books":[]}`)
	var legacySnapshot facultyProgressSnapshot
	if err := json.Unmarshal(legacy, &legacySnapshot); err != nil {
		t.Fatal(err)
	}
	legacyTarget := newPlayerActor("旧档侠", 0)
	legacyTarget.restoreFaculty(legacySnapshot, legacySnapshot.LastAdvanceUTC)
	legacyTarget.mu.Lock()
	base, mult = legacyTarget.progress.fillSpeedBase, legacyTarget.progress.fillSpeedMultiplier
	actValue := legacyTarget.progress.actDayValue
	str = legacyTarget.progress.attributes.str
	_, vitalsSet = legacyTarget.gm.vitals, legacyTarget.gm.vitalsSet
	legacyTarget.mu.Unlock()
	if base != 200 || mult != 1 {
		t.Fatalf("legacy fill speed split=%d×%v want 200×1", base, mult)
	}
	if actValue != actDayUnlimitedSentinel {
		t.Fatalf("legacy act day value=%d want unlimited sentinel", actValue)
	}
	if str != 30 {
		t.Fatalf("legacy base str=%d want starter 30 (不覆盖运行时值)", str)
	}
	if vitalsSet {
		t.Fatal("legacy snapshot must not seed vitals")
	}
}
