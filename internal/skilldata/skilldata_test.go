package skilldata

// 夹具值取自官服设计态表真值（share/skill/*.ini，经 tools/datagen skill 构建），
// 与 docs/official/初始账号登陆-完整引导-女.pcapng 抓包里五个技能的线为
// 逐项对应（冷却类别/分组/射程/消耗）。

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

func loadFixture(t *testing.T) *Catalog {
	t.Helper()
	catalog, err := LoadDir(filepath.Join("testdata", "Skill"))
	if err != nil {
		t.Fatalf("装载夹具: %v", err)
	}
	return catalog
}

// 普攻：扇形射程 2m、无消耗、10 段连招动作、公共冷却 2s（CoolEffectType=2）。
func TestDefinitionNormalAttack(t *testing.T) {
	def, err := loadFixture(t).Definition("zs_normal_00", 1)
	if err != nil {
		t.Fatal(err)
	}
	if def.Script != ScriptNormal || def.ItemType != 1000 || def.StaticData != 13 || def.VarPropNo != 241 || def.Levels != 1 {
		t.Fatalf("对象行联查不符: %+v", def)
	}
	if def.CoolDownMS != 1200 || def.CoolDownCategory != 2012 || def.CoolDownTeam != 50001 ||
		def.CoolEffectType != 2 || def.PublicCoolDownMS != 2000 {
		t.Errorf("冷却族不符: cd=%d cat=%d team=%d effect=%d public=%d",
			def.CoolDownMS, def.CoolDownCategory, def.CoolDownTeam, def.CoolEffectType, def.PublicCoolDownMS)
	}
	if def.TargetType != 4 || !def.NeedsTarget() || def.CastRange != 2 {
		t.Errorf("目标/射程不符: type=%d needs=%t range=%v", def.TargetType, def.NeedsTarget(), def.CastRange)
	}
	if def.TargetShape != (Shape{Kind: 3, Para1: 120, Para2: 2, Comment: "自己正面扇形区域120度半径2米"}) {
		t.Errorf("射程包不符: %+v", def.TargetShape)
	}
	if def.HitShape.Kind != 0 || def.Consume != (Consume{}) || def.Lock.OK || def.CanRush || !def.CanAirUse || def.HeightDiff != 2 {
		t.Errorf("缺省包不符: hit=%+v consume=%+v lock=%+v rush=%t air=%t dy=%v",
			def.HitShape, def.Consume, def.Lock, def.CanRush, def.CanAirUse, def.HeightDiff)
	}
	if n := len(def.Flow.HitActions); n != 10 || def.Flow.HitActions[0] != "zs_normal_00_01" || def.Flow.HitActions[9] != "zs_normal_00_10" {
		t.Errorf("连招动作不符: %v", def.Flow.HitActions)
	}
}

// 自身范围技（长拳 04）：射程 5m 圆、耗 MP 3、正面扇形 5m 作用域、流程包锁定窗 1s。
func TestDefinitionAreaSkill(t *testing.T) {
	def, err := loadFixture(t).Definition("CS_jh_cqgf04", 1)
	if err != nil {
		t.Fatal(err)
	}
	if def.CoolDownMS != 3000 || def.CoolDownCategory != 5830 || def.CoolDownTeam != 50124 ||
		def.CoolEffectType != 1 || def.PublicCoolDownMS != 7000 {
		t.Errorf("冷却族不符: %+v", def)
	}
	if def.TargetType != 8 || def.NeedsTarget() || def.CastRange != 5 {
		t.Errorf("无目标范围技不符: type=%d needs=%t range=%v", def.TargetType, def.NeedsTarget(), def.CastRange)
	}
	if def.Consume.MP != 3 || def.Consume.HP != 0 {
		t.Errorf("消耗包不符: %+v", def.Consume)
	}
	if def.HitShape != (Shape{Kind: 3, Para1: 120, Para2: 5, Comment: "自己正面扇形区域120度半径5米"}) {
		t.Errorf("作用域包不符: %+v", def.HitShape)
	}
	if def.Flow.LockBuffTimeMS != 1000 || len(def.Flow.HitActions) != 1 || def.Flow.HitActions[0] != "CS_jh_cqgf04" {
		t.Errorf("流程包不符: %+v", def.Flow)
	}
	if def.HitCount != 10 || def.EventProcess != 3666 || def.DamageCalculate != 3004 {
		t.Errorf("命中/伤害包引用不符: hits=%d event=%d damage=%d", def.HitCount, def.EventProcess, def.DamageCalculate)
	}
	if def.TaoLu != "CS_jh_cqgf" || def.EffectType != 1 {
		t.Errorf("套路/相性不符: %q %d", def.TaoLu, def.EffectType)
	}
}

// 锁定冲锋技（长拳 01）：射程 3m、CanRush、耗 MP 3 + 轻功 10、锁定包字段。
func TestDefinitionLockSkill(t *testing.T) {
	def, err := loadFixture(t).Definition("CS_jh_cqgf01", 1)
	if err != nil {
		t.Fatal(err)
	}
	if def.Script != ScriptLock || def.ItemType != 1001 || def.PauseTimeMS != 30000 || def.UseLimit != 0 {
		t.Fatalf("对象行不符: %+v", def)
	}
	if !def.CanRush || !def.NeedsTarget() || def.CastRange != 3 || def.TargetShape.Kind != 1 {
		t.Errorf("冲锋技不符: rush=%t needs=%t range=%v shape=%+v", def.CanRush, def.NeedsTarget(), def.CastRange, def.TargetShape)
	}
	if def.CoolDownMS != 6000 || def.CoolDownCategory != 5828 || def.Consume.MP != 3 || def.Consume.QingGong != 10 {
		t.Errorf("冷却/消耗不符: %+v", def)
	}
	lock := def.Lock
	if !lock.OK || lock.Ratio != 100 || lock.SrcTime != 1500 || lock.TargetTime != 1500 ||
		lock.SrcDist != 0.87 || lock.TargetDist != 0.9 || lock.GroundRushAction != "qg_fastrun_0h_01" ||
		lock.AirRushAction != "qg_fastruning_s_0h_01" || lock.DodgeAction != "CS_jh_cqgf01_dodge" ||
		lock.FailedAction != "jb_0h_07" || lock.BindDistance != 0 {
		t.Errorf("锁定包不符: %+v", lock)
	}
}

// 暗器普攻：25m 射程、消耗物品 1、使用限制包 25（必须使用飞镖）。
func TestDefinitionThrown(t *testing.T) {
	def, err := loadFixture(t).Definition("hw_normal_fb", 1)
	if err != nil {
		t.Fatal(err)
	}
	if def.CoolDownMS != 1500 || def.CastRange != 25 || def.Consume.ItemCount != 1 || def.Consume.MP != 0 ||
		def.UseLimit != 25 || !def.NoLimit || def.CanAirUse || def.HeightDiff != 25 {
		t.Errorf("暗器普攻不符: %+v", def)
	}
	if len(def.Flow.HitActions) != 1 || def.Flow.HitActions[0] != "zs_aq070_01_fd" {
		t.Errorf("动作不符: %v", def.Flow.HitActions)
	}
}

// 逐级选行：等级 <1 按 1；区间内取 Level 相符行；越界回落首行；Levels 为实有行数。
func TestDefinitionLevelSelection(t *testing.T) {
	catalog := loadFixture(t)
	for level, want := range map[int]int{0: 91001, 1: 91001, 2: 91002, 3: 91001, 99: 91001} {
		def, err := catalog.Definition("test_levels", level)
		if err != nil {
			t.Fatalf("level %d: %v", level, err)
		}
		if def.VarPropNo != want || def.Levels != 2 {
			t.Errorf("level %d: varprop %d levels %d, want %d/2", level, def.VarPropNo, def.Levels, want)
		}
	}
	if def, _ := catalog.Definition("test_levels", 0); def.Level != 1 {
		t.Errorf("等级 <1 应按 1, got %d", def.Level)
	}
	if def, _ := catalog.Definition("test_levels", 2); def.CoolDownMS != 2000 {
		t.Errorf("2 级行值不符: cd=%d", def.CoolDownMS)
	}
	if def, err := catalog.Definition("CS_jh_cqgf01", 2); err != nil || def.VarPropNo != 88002 || def.Levels != 2 {
		t.Errorf("锁定技 2 级: %+v %v", def, err)
	}
}

// 错误域：未入库 / 非主动技能逻辑类 / 静态包行缺失。
func TestDefinitionErrors(t *testing.T) {
	catalog := loadFixture(t)
	if _, err := catalog.Definition("no_such_skill", 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("未入库技能应 ErrNotFound, got %v", err)
	}
	if _, err := catalog.Definition("mind_ng_cf_001", 1); !errors.Is(err, ErrUnsupported) {
		t.Errorf("内功应 ErrUnsupported, got %v", err)
	}
	if _, err := catalog.Definition("test_nostatic", 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("静态包行缺失应 ErrNotFound, got %v", err)
	}
}

// 属性值反转义、反斜杠路径、非 ASCII 技能 ID、重复 ID 首行胜出。
func TestDefinitionEscapesAndIDs(t *testing.T) {
	catalog := loadFixture(t)
	def, err := catalog.Definition("test_escape", 1)
	if err != nil {
		t.Fatal(err)
	}
	if def.TaoLu != `a "quoted" & <tag>` || def.LuaScript != `skill\wuxue\wx_test.lua` {
		t.Errorf("反转义不符: taolu=%q lua=%q", def.TaoLu, def.LuaScript)
	}
	if def, err := catalog.Definition("裂天拳", 1); err != nil || def.CoolDownMS != 1200 {
		t.Errorf("非 ASCII ID: %+v %v", def, err)
	}
	if def, _ := catalog.Definition("test_dup", 1); def.StaticData != 13 {
		t.Errorf("重复 ID 应首行胜出, got StaticData=%d", def.StaticData)
	}
}

func TestRushRange(t *testing.T) {
	got := loadFixture(t).RushRange()
	want := RushRange{Ground: 5, Air: 8, Water: 8, GroundAdd: 1.5, AirAdd: 6.5, WaterAdd: 6.5}
	if got != want {
		t.Errorf("RushRange=%+v want %+v", got, want)
	}
}

// 扫描器容错：伪前缀元素跳过、黑金档 `ID = "-1"` 空格写法、可选包缺席。
func TestScanTolerances(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Skill")
	if err := os.MkdirAll(filepath.Join(dir, dirNew), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<Object>\n" +
		"\t<PropertyX ID=\"999\" script=\"SkillNormal\"/>\n" +
		"\t<Property ID = \"-1\" CoolDownCategory = \"3\"/>\n" +
		"\t<Property ID='quoted' StaticData=\"5\"/>\n" +
		"\t<Property ID=\"zs_normal_00\" script=\"SkillNormal\" StaticData=\"1\" ItemType=\"1000\"/>\n" +
		"\t<Property ID=\"broken\" StaticData=\"x\n" +
		"</Object>\n"
	if err := os.WriteFile(filepath.Join(dir, dirNew, "SkillNormal.xml"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	static := "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<Object>\n" +
		"\t<Property ID=\"1\" MinVarPropNo=\"241\" MaxVarPropNo=\"241\" TargetType=\"8\"/>\n</Object>\n"
	if err := os.WriteFile(filepath.Join(dir, fileStatic), []byte(static), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	classes := catalog.table(dirNew + "/SkillNormal.xml")
	for _, id := range []string{"-1", "quoted", "zs_normal_00"} {
		if _, ok := classes.index[id]; !ok {
			t.Errorf("ID %q 未入索引", id)
		}
	}
	if _, ok := classes.lookup("999"); ok {
		t.Error("伪前缀元素不应入索引")
	}
	if _, ok := classes.lookup("broken"); ok {
		t.Error("畸形元素不应入索引")
	}
	if row, ok := classes.lookup("-1"); !ok || row.int("CoolDownCategory", 0) != 3 {
		t.Errorf("空格写法属性应可解析: %v %v", row, ok)
	}
	// 逐级行文件缺席：定义报错而非 panic。
	if _, err := catalog.Definition("zs_normal_00", 1); err == nil {
		t.Error("逐级行缺席应报错")
	}
	// 可选包缺席：RushRange 零值。
	if got := catalog.RushRange(); got != (RushRange{}) {
		t.Errorf("可选包缺席应零值, got %+v", got)
	}
}

func TestUnescapeAttr(t *testing.T) {
	for in, want := range map[string]string{
		"plain":          "plain",
		"a&amp;b":        "a&b",
		"&lt;x&gt;":      "<x>",
		"&quot;q&quot;":  `"q"`,
		"&#13;&#10;&#9;": "\r\n\t",
		"&#x4E2D;":       "中",
		"&#20013;":       "中",
		"&unknown;":      "&unknown;",
		"tail&":          "tail&",
		"a&b;c":          "a&b;c",
		"&#xZZ;":         "&#xZZ;",  // 非法数字：原样保留
		"&#x4E2D":        "&#x4E2D", // 缺结尾分号：原样保留
	} {
		if got := unescapeAttr(in); got != want {
			t.Errorf("unescapeAttr(%q)=%q want %q", in, got, want)
		}
	}
}

// 对接仓库真实产物（tools/datagen skill 构建；缺席则跳过）：官服真值抽查 +
// 首次装载耗时观测。
func TestRealProducts(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	if _, err := res.Stat(filepath.Join(root, "ini", "Skill", fileStatic)); err != nil {
		t.Skipf("仓库产物缺席: %v", err)
	}
	start := time.Now()
	catalog, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	first, err := catalog.Definition("CS_jh_cqgf04", 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("首次装载+联查 %s", time.Since(start))
	if first.CoolDownMS != 3000 || first.Consume.MP != 3 || first.CastRange != 5 || first.Levels != 16 ||
		first.Flow.LockBuffTimeMS != 1000 || len(catalog.classes) != 2 {
		t.Errorf("真实产物不符: %+v classes=%v", first, catalog.classes)
	}
	if def, err := catalog.Definition("裂天拳", 1); err != nil || def.StaticData != 5675 {
		t.Errorf("非 ASCII ID 真实行: %+v %v", def, err)
	}
	if def, err := catalog.Definition("CS_jh_cqgf01", 16); err != nil || def.Level != 16 || def.Levels != 16 {
		t.Errorf("16 级锁定技: %+v %v", def, err)
	}
	if _, err := catalog.Definition("CS_jh_cqgf99", 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("不存在的技能应 ErrNotFound, got %v", err)
	}
	if got := catalog.RushRange(); got.Ground != 5 || got.GroundAdd != 1.5 {
		t.Errorf("RushRange=%+v", got)
	}
}
