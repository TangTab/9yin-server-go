package main

// 镇派兵器动态属性 / 宝物·娃娃随机属性生成器测试：
//   - 纯逻辑（无表）：条件波段评估、roleListTitle 阶梯、record 帧编码；
//   - 真表集成（NINEYIN_SERVER_ROOT 指向部署资源根才生效，G: 仓库内跳过）：
//     共用包/倚天剑波段命中、实力点数推导、黄帝金简宝物掷值、娃娃掷值。
// 数值锚点全部来自 docs/镇派兵器与娃娃宝物随机属性.md §2/§3（对资源
// 原文逐行核实）。

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestCondSatisfiedAtPowerLevelBands(t *testing.T) {
	c := &randomEquipCatalog{
		conds: map[int]powerCond{
			109141: {name: "PowerLevel", min: 1, max: 10},
			109151: {name: "PowerLevel", min: 101},
			16524:  {name: "PowerLevel", min: 1, max: 5},
			16537:  {name: "PowerLevel", min: 66},
		},
		formulas: map[int]string{
			24545: "16524&24559",
			24558: "16537&24559",
			24559: "18418|18419|18420|18456|118500",
		},
	}
	cases := []struct {
		name  string
		cond  int
		power int
		want  bool
	}{
		{"zpsb 波段首 1 命中 109141", 109141, 1, true},
		{"zpsb 波段首 10 命中 109141", 109141, 10, true},
		{"zpsb 波段首 11 不命中", 109141, 11, false},
		{"zpsb 顶档 101 命中 109151", 109151, 101, true},
		{"zpsb 顶档 240 命中 109151（无上限）", 109151, 240, true},
		{"zpsb 顶档 100 不命中", 109151, 100, false},
		{"倚天剑 24545=16524&24559 于 5 点", 24545, 5, true},
		{"倚天剑 24545 于 6 点不命中", 24545, 6, false},
		{"倚天剑 24558=16537&24559 于 66 点", 24558, 66, true},
		{"倚天剑 24558 于 65 点不命中", 24558, 65, false},
		{"无恒真", 0, 0, true},
	}
	for _, testCase := range cases {
		if got := c.condSatisfiedAt(testCase.cond, testCase.power, 0); got != testCase.want {
			t.Errorf("%s: condSatisfiedAt(%d,%d)=%v want %v", testCase.name, testCase.cond, testCase.power, got, testCase.want)
		}
	}
}

func TestZhenpaiDynamicValuesSynthetic(t *testing.T) {
	// 合成目录：三包各两行（10 级/20 级带宽），验证一包只取命中行、
	// 值按 PropName 聚合。
	c := &randomEquipCatalog{
		packRows: map[int][]propPackRow{
			95258: {
				{prop: "MaxHPAdd", value: 1500, cond: 1},
				{prop: "MaxHPAdd", value: 2500, cond: 2},
			},
			95259: {
				{prop: "MaxMPAdd", value: 200, cond: 1},
				{prop: "MaxMPAdd", value: 400, cond: 2},
			},
		},
		conds: map[int]powerCond{
			1: {name: "PowerLevel", min: 1, max: 10},
			2: {name: "PowerLevel", min: 11},
		},
		itemFields: map[string]randomItemFields{
			"blade_leader_test": {propPacks: []int{95258, 95259}},
		},
		formulas: map[int]string{},
	}
	low := c.zhenpaiDynamicValues("blade_leader_test", 10)
	if low["MaxHPAdd"] != 1500 || low["MaxMPAdd"] != 200 {
		t.Errorf("power=10 got %v want HP1500/MP200", low)
	}
	high := c.zhenpaiDynamicValues("blade_leader_test", 11)
	if high["MaxHPAdd"] != 2500 || high["MaxMPAdd"] != 400 {
		t.Errorf("power=11 got %v want HP2500/MP400", high)
	}
}

func TestRoleListTitleLadder(t *testing.T) {
	cases := []struct {
		power int
		want  string
	}{
		{0, "title001"}, {5, "title001"}, {6, "title002"}, {67, "title014"},
		{116, "title024"}, {120, "title024"}, {121, "title025"}, {131, "title025"},
		{135, "title025"}, {136, "title026"}, {150, "title026"}, {151, "title027"},
		{165, "title027"}, {166, "title028"}, {196, "title030"}, {226, "title032"},
		{240, "title032"}, {999, "title032"},
	}
	for _, testCase := range cases {
		if got := roleListTitle(testCase.power); got != testCase.want {
			t.Errorf("roleListTitle(%d)=%q want %q", testCase.power, got, testCase.want)
		}
	}
}

func TestServerRecordAddRowsViewEncoding(t *testing.T) {
	// RandomPropRec(400) 六列 [int32,f64,int32,int32,int32,int32]：编码后
	// 头 16 字节 + 逐行 28 字节；官服方言 flag=1 + viewID/objectIndex 寻址。
	rows := [][]recordCell{
		{recordInt(9), recordFloat64(14), recordInt(0), recordInt(0), recordInt(0), recordInt(0)},
		{recordInt(12), recordFloat64(16), recordInt(0), recordInt(0), recordInt(0), recordInt(0)},
	}
	frame, err := serverRecordAddRowsView(1, 25, recordRandomProp, rows)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if frame[0] != 0x11 || frame[1] != 1 {
		t.Fatalf("header = % x", frame[:2])
	}
	if got := binary.LittleEndian.Uint32(frame[2:]); got != 1 {
		t.Errorf("viewID = %d want 1", got)
	}
	if got := binary.LittleEndian.Uint32(frame[6:]); got != 25 {
		t.Errorf("objectIndex = %d want 25", got)
	}
	if got := binary.LittleEndian.Uint16(frame[10:]); got != 400 {
		t.Errorf("recordIndex = %d want 400", got)
	}
	if got := binary.LittleEndian.Uint16(frame[14:]); got != 2 {
		t.Errorf("rowCount = %d want 2", got)
	}
	if len(frame) != 16+2*(4+8+4+4+4+4) {
		t.Errorf("frame length = %d want %d", len(frame), 16+2*28)
	}
	if got := binary.LittleEndian.Uint32(frame[16:]); got != 9 {
		t.Errorf("row0 propID = %d want 9", got)
	}
	if got := float64FromBytes(frame[20:28]); got != 14 {
		t.Errorf("row0 value = %v want 14", got)
	}
}

func float64FromBytes(b []byte) float64 {
	return math.Float64frombits(binary.LittleEndian.Uint64(b))
}

func TestZhenpaiAffixesJSONShape(t *testing.T) {
	c := &randomEquipCatalog{
		itemFields: map[string]randomItemFields{
			"sword_ytj_001": {propPacks: []int{95510, 95511, 95512, 95513}},
		},
		packRows: map[int][]propPackRow{}, conds: map[int]powerCond{}, formulas: map[int]string{},
	}
	encoded := c.zhenpaiAffixesJSON("sword_ytj_001")
	var entries []struct {
		PackRec   int `json:"pack_rec"`
		PropBuilt int `json:"prop_built"`
	}
	if err := json.Unmarshal([]byte(encoded), &entries); err != nil {
		t.Fatalf("decode affixes %q: %v", encoded, err)
	}
	if len(entries) != 4 || entries[0].PropBuilt != 95510 || entries[3].PropBuilt != 95513 {
		t.Errorf("affixes = %+v want 4 packs 95510..95513", entries)
	}
}

// ---- 真表集成（需 NINEYIN_SERVER_ROOT 指向部署资源根） ----

func randomCatalogForTest(t *testing.T) *randomEquipCatalog {
	t.Helper()
	root := os.Getenv("NINEYIN_SERVER_ROOT")
	if root == "" {
		root = ".."
	}
	if _, err := os.Stat(filepath.Join(root, "resources", "modern", "share", "modifypack", "proppack.ini")); err != nil {
		t.Skipf("真表不可达（NINEYIN_SERVER_ROOT=%q）：%v", root, err)
	}
	t.Setenv(serverRootEnvironment, root)
	resetRandomEquipCatalogForTest()
	catalog := randomEquipCatalogOf()
	if catalog.err != nil {
		t.Fatalf("装载目录失败: %v", catalog.err)
	}
	return catalog
}

func TestZhenpaiBandsAgainstRealTables(t *testing.T) {
	c := randomCatalogForTest(t)
	cases := []struct {
		config string
		power  int
		want   map[string]float64
	}{
		{"blade_leader_jyw_001", 67, map[string]float64{ // 冷月刀 61-70 波段
			"MaxHPAdd": 12000, "MaxMPAdd": 1800, "MaxQingGongPointAdd": 200, "MaxParryAdd": 50000,
		}},
		{"blade_leader_jyw_001", 5, map[string]float64{ // 最低档 1-10
			"MaxHPAdd": 1500, "MaxMPAdd": 200, "MaxQingGongPointAdd": 100, "MaxParryAdd": 10000,
		}},
		{"sword_ytj_001", 33, map[string]float64{ // 倚天剑 31-35 波段
			"MaxHPAdd": 5000, "MaxMPAdd": 1400, "MaxParryAdd": 1050, "MaxQingGongPointAdd": 140,
		}},
		{"sword_ytj_001", 66, map[string]float64{ // 66+ 顶档
			"MaxHPAdd": 12000, "MaxMPAdd": 2800, "MaxParryAdd": 2100, "MaxQingGongPointAdd": 280,
		}},
	}
	for _, testCase := range cases {
		got := c.zhenpaiDynamicValues(testCase.config, testCase.power)
		for prop, want := range testCase.want {
			if got[prop] != want {
				t.Errorf("%s@%d %s=%v want %v (all=%v)", testCase.config, testCase.power, prop, got[prop], want, got)
			}
		}
	}
}

func TestDerivePowerLevelRealTables(t *testing.T) {
	c := randomCatalogForTest(t)
	// 洗髓经(少林五内, StaticData=1027) 49 层 = 111 点；两书取最大。
	got := c.derivePowerLevel([]neigongPowerBook{
		{staticData: 1002, level: 36}, // 少林一内禅定功 36 层 = 42
		{staticData: 1027, level: 49}, // 少林五内洗髓经 49 层 = 111
	})
	if got != 111 {
		t.Errorf("derivePowerLevel = %d want 111", got)
	}
	if title := c.titleForPowerLevel(got); title != "title023" {
		t.Errorf("titleForPowerLevel(111) = %q want title023（空前绝后）", title)
	}
	if title := c.titleForPowerLevel(151); title != "title027" {
		t.Errorf("titleForPowerLevel(151) = %q want title027（返璞归真）", title)
	}
}

func TestRollHuangdiJinjian(t *testing.T) {
	c := randomCatalogForTest(t)
	if kind := c.randomEquipKind("treasure_31111"); kind != randomKindTreaure {
		t.Fatalf("treasure_31111 kind = %d want treasure", kind)
	}
	rnd := rand.New(rand.NewSource(20261108))
	snapshot, err := c.rollTreasure("treasure_31111", rnd)
	if err != nil {
		t.Fatalf("roll treasure_31111: %v", err)
	}
	// 模板 31301 = 11181(气血×1) + 11182(招架耐力×1) + 11183(通用×5) +
	// 2014(打狗技能×4)：4 属性 + 4 技能（黄帝金简，盘点 §3 取证）。
	if len(snapshot.PropRows) != 7 {
		t.Errorf("prop rows = %d want 7（气血1+招架1+通用5）rows=%+v", len(snapshot.PropRows), snapshot.PropRows)
	}
	if len(snapshot.SkillRows) != 4 {
		t.Errorf("skill rows = %d want 4 rows=%+v", len(snapshot.SkillRows), snapshot.SkillRows)
	}
	hasHP, hasParry := false, false
	for _, line := range snapshot.PropRows {
		if line.Value <= 0 {
			t.Errorf("prop %d value %v 非正", line.PropID, line.Value)
		}
		switch line.PropID {
		case 0:
			hasHP = true
		case 4:
			hasParry = true
		}
	}
	if !hasHP || !hasParry {
		t.Errorf("气血/招架耐力缺位 hasHP=%v hasParry=%v", hasHP, hasParry)
	}
	for _, line := range snapshot.SkillRows {
		entry, ok := c.skstore[line.StoreID]
		if !ok || len(entry.subs) == 0 {
			t.Errorf("skill store %d 未装载", line.StoreID)
		}
	}
}

func TestRollWawaGrow(t *testing.T) {
	c := randomCatalogForTest(t)
	if kind := c.randomEquipKind("newtreasure_grow_6_01"); kind != randomKindWawa {
		t.Fatalf("newtreasure_grow_6_01 kind = %d want wawa", kind)
	}
	rnd := rand.New(rand.NewSource(20261108))
	snapshot, err := c.rollWawa("newtreasure_grow_6_01", rnd)
	if err != nil {
		t.Fatalf("roll wawa: %v", err)
	}
	// ColorLevel 6 → 1 基础 + 5 附加；基础来自 grow_6_01（气血恢复速率，
	// propID 2）；附加行品质 1-4、等级 1。
	if len(snapshot.PropRows) != 6 {
		t.Fatalf("prop rows = %d want 6（1 基础 + 5 附加）rows=%+v", len(snapshot.PropRows), snapshot.PropRows)
	}
	base := snapshot.PropRows[0]
	if base.PropID != 2 {
		t.Errorf("基础属性 = %d want 2（气血恢复速率）", base.PropID)
	}
	if base.Value <= 0 {
		t.Errorf("基础值 = %v 非正", base.Value)
	}
	for _, line := range snapshot.PropRows[1:] {
		if line.Quality < 1 || line.Quality > 4 {
			t.Errorf("附加品质 = %d 越界", line.Quality)
		}
		if line.Level != 1 {
			t.Errorf("附加等级 = %d want 1", line.Level)
		}
	}
}
