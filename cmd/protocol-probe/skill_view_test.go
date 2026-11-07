package main

import (
	"bytes"
	"encoding/binary"
	"math"
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
			{byte(propWuXing & 0xff), byte(propWuXing >> 8)}, // WuXing@2243:官服 55866 每行实证
		} {
			if !bytes.Contains(frame, expected) {
				t.Fatalf("%s ViewAdd lacks %x: %x", skill.configID, expected, frame)
			}
		}
	}
}

// 基础武学→三脚猫招式五件套（bug.txt：官服全角色默认已学，
// Java DefaultBasicSkillsTest.QUINTET 对拍）：官方观测序居 View40 队首。
func TestStarterSkillViewCarriesDefaultBasicQuintet(t *testing.T) {
	quintet := []struct {
		id         string
		staticData int32
	}{
		{"zs_normal_02", 15},
		{"hw_normal_fd", 626},
		{"zs_default_01", 22},
		{"zs_suicide", 55},
		{"riding_start", 21515},
	}
	for index, expected := range quintet {
		skill := starterSkillViews[index]
		if skill.configID != expected.id || skill.staticData != expected.staticData ||
			skill.itemType != 1000 || skill.level != 1 || skill.maxLevel != 1 {
			t.Fatalf("basic skill[%d]=%+v, want %s static=%d itemType=1000 level=1 maxLevel=1",
				index, skill, expected.id, expected.staticData)
		}
	}
	if got, want := len(starterSkillViews), 27; got != want {
		t.Fatalf("starter set=%d, want 27 (五件套 + 花神 7 + 心佛掌 7 + 太极拳 8)", got)
	}
	if last := starterSkillViews[len(starterSkillViews)-1]; last.configID != "CS_wd_tjq08" {
		t.Fatalf("last starter row=%s, want CS_wd_tjq08", last.configID)
	}
}

func TestYiHuaWuQueSkillMetadataMatchesInstalledClient(t *testing.T) {
	wantIDs := []string{
		"CS_yhwq_hsqs01", "CS_yhwq_hsqs02", "CS_yhwq_hsqs03",
		"CS_yhwq_hsqs04", "CS_yhwq_hsqs05", "CS_yhwq_hsqs06",
		"CS_yhwq_hsqs07",
	}
	wantMax := []int32{11, 11, 11, 9, 4, 11, 11}
	for index, skill := range starterSkillViews[5:12] {
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
		skill := starterSkillViews[index+12]
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
	const taiJiStart = 19
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

// TestSkillViewRowsCarryCatalogWuXing 钉死 View40 行 WuXing 的值语义：
// 目录值按 wuxuebaseinfo.ini（花神七式=2 柔 / 太极拳=5 太极），未知
// configId 回退 5=太极（Java StarterSkills.skillWuXing 同款），绝不发 0
// ——客户端对 ≤0/>7 会把属性列退回 ui_train_elemt 缺省文案。
func TestSkillViewRowsCarryCatalogWuXing(t *testing.T) {
	want := map[string]int32{
		"zs_default_01":  5,
		"CS_yhwq_hsqs01": 2,
		"CS_jh_xfz01":    5,
		"CS_wd_tjq01":    5,
	}
	for configID, value := range want {
		if got := skillWuXing(configID); got != value {
			t.Errorf("skillWuXing(%s)=%d, want %d", configID, got, value)
		}
	}
	if got := skillWuXing("no_such_skill"); got != wuxingTaiJi {
		t.Errorf("skillWuXing(unknown)=%d, want fallback %d", got, wuxingTaiJi)
	}
	// 目录可用时行内值必须与查表一致（ng 目录缺席的夹具环境会整行退 5，
	// 与 want 兜底值吻合，两种环境同一断言都成立）。
	frames, err := learnedSkillViewFrames(starterSkillViews)
	if err != nil {
		t.Fatal(err)
	}
	for index, skill := range starterSkillViews {
		expected := []byte{
			byte(propWuXing & 0xff), byte(propWuXing >> 8),
			byte(skillWuXing(skill.configID)), 0, 0, 0,
		}
		if !bytes.Contains(frames[index+1], expected) {
			t.Fatalf("%s ViewAdd lacks WuXing=%x: %x", skill.configID, expected, frames[index+1])
		}
	}
}

func viewPropByIndex(properties []serverViewProperty, index uint16) *serverViewProperty {
	for i := range properties {
		if properties[i].index == index {
			return &properties[i]
		}
	}
	return nil
}

func viewPropInt(properties []serverViewProperty, index uint16) (int32, bool) {
	if property := viewPropByIndex(properties, index); property != nil && property.int32 != nil {
		return *property.int32, true
	}
	return 0, false
}

func viewPropFloat(properties []serverViewProperty, index uint16) (float32, bool) {
	if property := viewPropByIndex(properties, index); property != nil && property.real32 != nil {
		return *property.real32, true
	}
	return 0, false
}

// TestSkillRowDataSupplementMatchesInstalledTables 钉死补集取值锚点（真实
// share/skill 表 2026-11-07 提取）：怒招 hsqs07/tjq08 AConsumeSP=50（快捷栏
// 怒气消耗/灰图标源）、太极01 扇形 HitShapePara1=120°/半径 5、射程
// TargetShapePara2（tooltip 距离/施法限距源）、xfz01 CD=6000 与编译目录
// personalCD=6s 互证。
func TestSkillRowDataSupplementMatchesInstalledTables(t *testing.T) {
	anchors := []struct {
		id                           string
		varPropNo, mp, sp, cdMs      int32
		targetPara2, hitPara1, hitPara2 float32
	}{
		{"CS_yhwq_hsqs01", 252701, 21, 0, 2000, 20, 0, 0},
		{"CS_yhwq_hsqs07", 252821, 0, 50, 10000, 20, 0, 0},
		{"CS_jh_xfz01", 263761, 16, 0, 6000, 20, 0, 0},
		{"CS_wd_tjq01", 81121, 21, 0, 2000, 5, 120, 5},
		{"CS_wd_tjq08", 42121, 0, 50, 10000, 5, 0, 5},
	}
	for _, anchor := range anchors {
		supplement := installedCombatSkillCatalog.skillRowData(anchor.id, 1)
		if supplement == nil {
			t.Fatalf("%s: no supplement from installed tables", anchor.id)
		}
		if supplement.varPropNo != anchor.varPropNo || supplement.aConsumeMP != anchor.mp ||
			supplement.aConsumeSP != anchor.sp || supplement.coolDownMs != anchor.cdMs ||
			supplement.targetShapePara2 != anchor.targetPara2 ||
			supplement.hitShapePara1 != anchor.hitPara1 || supplement.hitShapePara2 != anchor.hitPara2 {
			t.Errorf("%s supplement=%+v, want varPropNo=%d mp=%d sp=%d cd=%d tgt2=%.0f hit1=%.0f hit2=%.0f",
				anchor.id, *supplement, anchor.varPropNo, anchor.mp, anchor.sp,
				anchor.cdMs, anchor.targetPara2, anchor.hitPara1, anchor.hitPara2)
		}
	}
	// xfz07 无射程（TargetShapePara2=0）：发射侧按官方稀疏风格不发。
	if xfz07 := installedCombatSkillCatalog.skillRowData("CS_jh_xfz07", 1); xfz07 == nil || xfz07.targetShapePara2 != 0 {
		t.Errorf("xfz07 supplement=%+v, want zero-range sparse row", xfz07)
	}
	if unknown := installedCombatSkillCatalog.skillRowData("no_such_skill", 1); unknown != nil {
		t.Errorf("unknown skill supplement=%+v, want nil", unknown)
	}
}

// TestSkillViewRowsCarryConsumeRangeCdEmission 行级补集下发（Java
// StarterSkills.skillRowProperties 方言）：VarPropNo/CoolDownTime/Quality/
// 倍率固定族/Flow 每行必发，LeadTime/LeadSepTime starter 行必发；UseLimit
// 仅 skill_new 真值 >0 稀疏发（官方 55866 对拍：太极拳/花神/心佛掌整段不发，
// 回落 9 会按 skill_use_limit 规则 9 在入战时把图标判红）；
// 消耗/射程按线上字节序（索引 u16 LE + 值 LE）钉死锚点。
func TestSkillViewRowsCarryConsumeRangeCdEmission(t *testing.T) {
	frames, err := starterSkillViewFrames()
	if err != nil {
		t.Fatal(err)
	}
	rowFrame := func(id string) []byte {
		for index, skill := range starterSkillViews {
			if skill.configID == id {
				return frames[index+1]
			}
		}
		t.Fatalf("%s not in starter set", id)
		return nil
	}
	hasProp := func(frame []byte, propertyIndex uint16) bool {
		return bytes.Contains(frame, []byte{byte(propertyIndex), byte(propertyIndex >> 8)})
	}
	everyRow := []uint16{
		propVarPropNo, propCoolDownTime, propQuality, propFlow,
		propHPHealMul, propMPHealMul, propPhyDamageMul, propStiDamageMul,
		propJujDamageMul, propNegDamageMul, propMasDamageMul,
		propLeadTime, propLeadSepTime,
	}
	for index, skill := range starterSkillViews {
		for _, propertyIndex := range everyRow {
			if !bytes.Contains(frames[index+1], []byte{byte(propertyIndex), byte(propertyIndex >> 8)}) {
				t.Fatalf("%s row lacks supplement prop %d", skill.configID, propertyIndex)
			}
		}
	}
	intSeq := func(index uint16, value int32) []byte {
		seq := make([]byte, 6)
		binary.LittleEndian.PutUint16(seq, index)
		binary.LittleEndian.PutUint32(seq[2:], uint32(value))
		return seq
	}
	floatSeq := func(index uint16, value float32) []byte {
		seq := make([]byte, 6)
		binary.LittleEndian.PutUint16(seq, index)
		binary.LittleEndian.PutUint32(seq[2:], math.Float32bits(value))
		return seq
	}
	anchors := []struct {
		id     string
		expect [][]byte
	}{
		{"CS_yhwq_hsqs01", [][]byte{
			intSeq(propAConsumeMP, 21), intSeq(propCoolDownTime, 2000), floatSeq(propTargetShapePara2, 20)}},
		{"CS_yhwq_hsqs07", [][]byte{
			intSeq(propAConsumeSP, 50), intSeq(propCoolDownTime, 10000), floatSeq(propTargetShapePara2, 20)}},
		{"CS_jh_xfz01", [][]byte{
			intSeq(propAConsumeMP, 16), intSeq(propCoolDownTime, 6000)}},
		{"CS_wd_tjq01", [][]byte{
			floatSeq(propTargetShapePara2, 5), floatSeq(propHitShapePara1, 120), floatSeq(propHitShapePara2, 5)}},
		{"CS_wd_tjq08", [][]byte{
			intSeq(propAConsumeSP, 50), floatSeq(propHitShapePara2, 5)}},
	}
	for _, anchor := range anchors {
		frame := rowFrame(anchor.id)
		for _, expected := range anchor.expect {
			if !bytes.Contains(frame, expected) {
				t.Fatalf("%s ViewAdd lacks %x: %x", anchor.id, expected, frame)
			}
		}
	}
	// 太极08 是圆形（HitShape=1，非扇形）：官方规则 HitShapePara1 不发。
	tjq08 := rowFrame("CS_wd_tjq08")
	if bytes.Contains(tjq08, []byte{byte(propHitShapePara1 & 0xff), byte(propHitShapePara1 >> 8)}) {
		t.Fatalf("tjq08 must not carry non-sector HitShapePara1: %x", tjq08)
	}
	// UseLimit 官方稀疏方言：skill_new 无 UseLimit 的行整列不发（官方 55868
	// 抓包太极拳/花神/心佛掌全段缺席；下发 9 会按 skill_use_limit 规则 9
	// 「无法在战斗状态使用」在入战瞬间把图标判红）；五件套真值行照发。
	for _, id := range []string{"CS_wd_tjq01", "CS_wd_tjq04", "CS_wd_tjq08",
		"CS_yhwq_hsqs01", "CS_jh_xfz01"} {
		if hasProp(rowFrame(id), propSkillUseLimit) {
			t.Fatalf("%s must not carry UseLimit (skill_new has none): %x", id, rowFrame(id))
		}
	}
	if !bytes.Contains(rowFrame("zs_default_01"), intSeq(propSkillUseLimit, 9)) {
		t.Fatalf("zs_default_01 must carry real ini UseLimit=9 (打坐): %x", rowFrame("zs_default_01"))
	}
	if !bytes.Contains(rowFrame("zs_normal_02"), intSeq(propSkillUseLimit, 2)) {
		t.Fatalf("zs_normal_02 must carry real ini UseLimit=2 (单手剑): %x", rowFrame("zs_normal_02"))
	}
}

// TestSkillRowPropertiesStarterVsDBDialect starter 行与 DB 追加行的方言差
// （Java StarterGrants.dbSkillViewAdd 注释口径）：DB 行无链路时 LeadTime/
// LeadSepTime 不发；UseLimit 两方言一致稀疏（仅 skill_new 真值 >0 发——官方
// 55866 对拍，Java starter 回落 9 系把打坐行 zs_default_01 误泛化，下发 9
// 会让客户端按规则「无法在战斗状态使用」在入战时红图标）；倍率固定族两方言都发。
func TestSkillRowPropertiesStarterVsDBDialect(t *testing.T) {
	dbRow := skillRowProperties(learnedSkillView{configID: "no_such_skill", staticData: 1, itemType: 1000, level: 1})
	if viewPropByIndex(dbRow, propSkillUseLimit) != nil ||
		viewPropByIndex(dbRow, propLeadTime) != nil || viewPropByIndex(dbRow, propLeadSepTime) != nil {
		t.Fatalf("db row must not carry starter-only UseLimit/Lead fallbacks: %+v", dbRow)
	}
	if value, ok := viewPropFloat(dbRow, propPhyDamageMul); !ok || value != 1 {
		t.Fatalf("db row fixed multiplier missing: %+v", dbRow)
	}
	// starter 行：hsqs01 的 skill_new UseLimit 为空 → 整列不发；zs_default_01
	// 真值即 9（打坐，全包唯一值 9 行）。
	row := skillRowProperties(learnedSkillView{configID: "CS_yhwq_hsqs01", staticData: 1, itemType: 1000, level: 1})
	if viewPropByIndex(row, propSkillUseLimit) != nil {
		t.Fatalf("hsqs01 must not carry UseLimit (skill_new has none): %+v", row)
	}
	row = skillRowProperties(learnedSkillView{configID: "zs_default_01", staticData: 1, itemType: 1000, level: 1})
	if useLimit, ok := viewPropInt(row, propSkillUseLimit); !ok || useLimit != 9 {
		t.Fatalf("zs_default_01 UseLimit=%d ok=%t, want 9", useLimit, ok)
	}
	for _, id := range []string{"CS_yhwq_hsqs01", "zs_default_01"} {
		row := skillRowProperties(learnedSkillView{configID: id, staticData: 1, itemType: 1000, level: 1})
		if lead, ok := viewPropInt(row, propLeadTime); !ok || lead != 10000 {
			t.Fatalf("%s LeadTime=%d ok=%t, want 10000", id, lead, ok)
		}
	}
}
