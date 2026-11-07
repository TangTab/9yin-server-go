package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestStarterSkillViewUsesNativeSkillContainer(t *testing.T) {
	frames, err := starterSkillViewFrames()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(frames), len(starterSkillViews)+1; got != want {
		t.Fatalf("View=40 frames=%d, want %d", got, want)
	}
	create := frames[0]
	if create[0] != 0x15 ||
		binary.LittleEndian.Uint16(create[1:]) != viewportSkill ||
		binary.LittleEndian.Uint16(create[3:]) != uint16(len(starterSkillViews)) {
		t.Fatalf("CreateView(40)=%x", create)
	}

	for index, skill := range starterSkillViews {
		frame := frames[index+1]
		if frame[0] != 0x18 ||
			binary.LittleEndian.Uint16(frame[1:]) != viewportSkill ||
			binary.LittleEndian.Uint16(frame[3:]) != uint16(index+1) {
			t.Fatalf("%s ViewAdd header=%x", skill.configID, frame)
		}
		for _, expected := range [][]byte{
			[]byte(skill.configID + "\x00"),
			{byte(propStaticData & 0xff), byte(propStaticData >> 8)},
			{byte(propItemType & 0xff), byte(propItemType >> 8)},
			{byte(1465 & 0xff), byte(1465 >> 8), 1}, // Level(官方玩家域,Byte)
			{byte(propMaxLevel & 0xff), byte(propMaxLevel >> 8)},
			{byte(propCurFillValue & 0xff), byte(propCurFillValue >> 8)},
			{byte(propTotalFillValue & 0xff), byte(propTotalFillValue >> 8)},
			{byte(propPauseTime & 0xff), byte(propPauseTime >> 8)},
			{byte(propSkillCanUse & 0xff), byte(propSkillCanUse >> 8), 1},
		} {
			if !bytes.Contains(frame, expected) {
				t.Fatalf("%s ViewAdd lacks %x: %x", skill.configID, expected, frame)
			}
		}
	}
}

func TestYiHuaWuQueSkillMetadataMatchesInstalledClient(t *testing.T) {
	wantIDs := []string{
		"CS_yhwq_hsqs01", "CS_yhwq_hsqs02", "CS_yhwq_hsqs03",
		"CS_yhwq_hsqs04", "CS_yhwq_hsqs05", "CS_yhwq_hsqs06",
		"CS_yhwq_hsqs07",
	}
	wantMax := []int32{11, 11, 11, 9, 4, 11, 11}
	for index, skill := range starterSkillViews[1:8] {
		if skill.configID != wantIDs[index] {
			t.Fatalf("skill[%d]=%s, want %s", index, skill.configID, wantIDs[index])
		}
		if skill.staticData != int32(12636+index) {
			t.Fatalf("%s StaticData=%d", skill.configID, skill.staticData)
		}
		if skill.level != 1 || skill.maxLevel != wantMax[index] || skill.total != 156550 {
			t.Fatalf("%s level metadata=%+v", skill.configID, skill)
		}
		if skill.pauseTime <= 0 {
			t.Fatalf("%s PauseTime=%f, want positive milliseconds", skill.configID, skill.pauseTime)
		}
	}
}

func TestHeartBuddhaPalmSkillMetadataMatchesInstalledClient(t *testing.T) {
	want := []struct {
		id                 string
		static             int32
		itemType, maxLevel int32
	}{
		{"CS_jh_xfz01", 13189, 1001, 12},
		{"CS_jh_xfz02", 13190, 1000, 12},
		{"CS_jh_xfz03", 13205, 1000, 1},
		{"CS_jh_xfz04", 13206, 1000, 1},
		{"CS_jh_xfz05", 13207, 1000, 12},
		{"CS_jh_xfz06", 13208, 1000, 12},
		{"CS_jh_xfz07", 13203, 1000, 1},
	}
	for index, expected := range want {
		skill := starterSkillViews[index+8]
		if skill.configID != expected.id || skill.staticData != expected.static || skill.itemType != expected.itemType || skill.level != 1 || skill.maxLevel != expected.maxLevel {
			t.Fatalf("heart buddha skill[%d]=%+v, want %+v", index, skill, expected)
		}
	}
}

func TestTaiJiQuanGuPuSkillMetadataMatchesInstalledClient(t *testing.T) {
	want := []struct {
		id                 string
		static             int32
		itemType, maxLevel int32
	}{
		{"CS_wd_tjq01", 4057, 1000, 19},
		{"CS_wd_tjq02", 4058, 1001, 19},
		{"CS_wd_tjq03", 4238, 1000, 19},
		{"CS_wd_tjq04", 4045, 1000, 19},
		{"CS_wd_tjq05", 5451, 1000, 19},
		{"CS_wd_tjq06", 5452, 1000, 19},
		{"CS_wd_tjq07", 5453, 1000, 3},
		{"CS_wd_tjq08", 2107, 1000, 19},
	}
	const taiJiStart = 15
	for index, expected := range want {
		skill := starterSkillViews[taiJiStart+index]
		if skill.configID != expected.id || skill.staticData != expected.static ||
			skill.itemType != expected.itemType || skill.level != 1 ||
			skill.maxLevel != expected.maxLevel || skill.pauseTime <= 0 {
			t.Fatalf("tai ji skill[%d]=%+v, want %+v", index, skill, expected)
		}
	}
}

// starter 顺序与数值是快捷栏行号锚点：合并只允许在尾部追加 DB 行，
// 与 starter 重叠的 configID 以 starter 条目优先（去重不覆盖）。
func TestMergeLearnedSkillsAppendsDBRowsAfterStarter(t *testing.T) {
	dbRows := []learnedSkillView{
		{configID: "CS_wd_tjq01", staticData: 4057, itemType: 1000, level: 5}, // 与 starter 重叠
		{configID: "CS_by_xhmdl01", staticData: 13701, itemType: 1000, level: 1, maxLevel: 12},
		{configID: "CS_by_xhmdl01"}, // DB 自身重复也去重
	}
	merged := mergeLearnedSkills(dbRows)
	if got, want := len(merged), len(starterSkillViews)+1; got != want {
		t.Fatalf("merged=%d, want %d", got, want)
	}
	for index, skill := range starterSkillViews {
		if merged[index] != skill {
			t.Fatalf("merged[%d]=%+v must equal starter row (starter wins on overlap)", index, merged[index])
		}
	}
	extra := merged[len(merged)-1]
	if extra.configID != "CS_by_xhmdl01" || extra.staticData != 13701 {
		t.Fatalf("appended db row=%+v", extra)
	}
}

func TestMergeLearnedSkillsEmptyDBKeepsStarter(t *testing.T) {
	merged := mergeLearnedSkills(nil)
	if len(merged) != len(starterSkillViews) {
		t.Fatalf("merged=%d, want starter %d", len(merged), len(starterSkillViews))
	}
}

func TestLearnedSkillViewFramesUsesMergedCapacity(t *testing.T) {
	skills := append([]learnedSkillView(nil), starterSkillViews...)
	skills = append(skills, learnedSkillView{configID: "CS_by_xhmdl01", staticData: 13701, itemType: 1000, level: 1, maxLevel: 12})
	frames, err := learnedSkillViewFrames(skills)
	if err != nil {
		t.Fatal(err)
	}
	create := frames[0]
	if create[0] != 0x15 ||
		binary.LittleEndian.Uint16(create[1:]) != viewportSkill ||
		binary.LittleEndian.Uint16(create[3:]) != uint16(len(skills)) {
		t.Fatalf("CreateView(40) capacity mismatch: %x", create)
	}
	last := frames[len(frames)-1]
	if !bytes.Contains(last, []byte("CS_by_xhmdl01\x00")) {
		t.Fatalf("last row must carry the appended db skill: %x", last)
	}
}
