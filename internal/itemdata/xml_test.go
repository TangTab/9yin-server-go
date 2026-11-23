package itemdata

// xml_test.go 构建产物装载单测：登记表驱动索引、表头行跳过、同 ID 首行
// 胜出、反转义、分类页缺省/越界归一、静态包联查、文件不被长期占用；以及
// 对仓库产物 testdata/ini/Item 的实档抽检。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeProduct(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fixtureProduct(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeProduct(t, dir, "path_config.xml", `<?xml version="1.0" encoding="utf-8"?>
<Object>
	<Property ID="1" Type="ItemUse" Path="ItemUse.xml"/>
	<Property ID="2" Type="SkillBook" Path="SkillBook.xml"/>
</Object>
`)
	writeProduct(t, dir, "ItemUse.xml", `<?xml version="1.0" encoding="utf-8"?>
<Object>
	<Property ID="yaopin_01" script="ToolItem" ItemType="1" Name="回血&#10;药" FuncPack="1" UseSkillPack="11" LogicPack="23" ArtPack="9583" MaxAmount="99"/>
	<Property ID="dup_item" script="ToolItem" ItemType="1" Name="首行"/>
	<Property ID="dup_item" script="ToolItem" ItemType="2" Name="次行"/>
	<Property ID="inner_cloth" script="Equipment" ItemType="171" ViewID="11" EquipType="InnerCloth" Name="内功衣"/>
	<Property ID="no_view" script="ToolItem" ItemType="2093" Name="制鞋录"/>
	<Property ID="no_pack" script="ToolItem" ItemType="1" FuncPack="9999"/>
</Object>
`)
	writeProduct(t, dir, "SkillBook.xml", `<?xml version="1.0" encoding="utf-8"?>
<Object>
	<Property ID="book_01" script="SkillBook" ItemType="23" ViewID="1" Name="招式书 &amp; 秘籍" FuncPack="118" LearnSkillPack="3837"/>
</Object>
`)
	writeProduct(t, dir, "item_func_static.xml", `<Object>
	<Property ID="-1" LuaScript="0" PrepareAction="1" PrepareTime="2"/>
	<Property ID="0" LuaScript="6" PrepareAction="6" PrepareTime="2"/>
	<Property ID="1" PrepareTime="1000"/>
	<Property ID="118" LuaScript="item\useitem_leitai.lua" PrepareAction="life_ss_01" PrepareTime="1500"/>
</Object>
`)
	writeProduct(t, dir, "item_use_skill_static.xml", `<Object><Property ID="11" SkillID="yaoshihp" SkillLevel="3"/></Object>`)
	writeProduct(t, dir, "item_learn_skill_static.xml", `<Object><Property ID="3837" TragetID="CS_base_pgd01"/></Object>`)
	writeProduct(t, dir, "ItemArtStatic.xml", `<Object><Property ID="9583" Photo="icon\prop\ride.png" MaleModel="m" FemaleModel="f"/></Object>`)
	writeProduct(t, dir, "item_logic_pack.xml", `<Object><Property ID="23" BindType="1" CantSell="1"/></Object>`)
	return dir
}

func TestLoadXMLLookupAndPacks(t *testing.T) {
	dir := fixtureProduct(t)
	catalog, err := LoadXML(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.FromXML() || catalog.Count() != 6 || len(catalog.MissingPacks()) != 4 || catalog.Source() != "xml:"+dir {
		t.Fatalf("来源/条目/缺席包不符: %+v missing=%v", catalog, catalog.MissingPacks())
	}
	entry, ok := catalog.Lookup("yaopin_01")
	if !ok || entry.ItemType != 1 || entry.ViewID != 1 || entry.MaxAmount != 99 || entry.Name != "回血\n药" || entry.Script != "ToolItem" {
		t.Fatalf("yaopin_01 异常: %+v ok=%v", entry, ok)
	}
	// 分类页归一：越界页标（11 内功装备页）→ 装备页；未标页标 → 道具页。
	if e, _ := catalog.Lookup("inner_cloth"); e.ViewID != 2 || e.EquipType != "InnerCloth" {
		t.Errorf("越界页标应归装备页: %+v", e)
	}
	if e, _ := catalog.Lookup("no_view"); e.ViewID != 1 {
		t.Errorf("未标页标应归道具页: %+v", e)
	}
	// 同 ID 首行胜出。
	if e, _ := catalog.Lookup("dup_item"); e.Name != "首行" {
		t.Errorf("同 ID 应首行胜出: %+v", e)
	}
	if _, ok := catalog.Lookup("absent"); ok {
		t.Error("不存在的 ID 不应命中")
	}

	def, ok := catalog.Definition("yaopin_01")
	if !ok || def.Script() != "ToolItem" || def.Int("ItemType") != 1 || !def.Has("FuncPack") {
		t.Fatalf("模板行异常: %+v ok=%v", def, ok)
	}
	fp, ok := def.FuncPack()
	if !ok || fp.ID != 1 || fp.PrepareTimeMS != 1000 || fp.PrepareAction != "" || fp.LuaScript != "" {
		t.Errorf("功能包联查异常: %+v ok=%v", fp, ok)
	}
	us, ok := def.UseSkill()
	if !ok || us.SkillID != "yaoshihp" || us.SkillLevel != 3 {
		t.Errorf("使用技能包联查异常: %+v ok=%v", us, ok)
	}
	art, ok := def.Art()
	if !ok || art.Photo != `icon\prop\ride.png` || art.MaleModel != "m" || art.ActionSet != "" {
		t.Errorf("美术包联查异常: %+v ok=%v", art, ok)
	}
	logic, ok := def.Logic()
	if !ok || logic.ID != 23 || logic.BindType != 1 || logic.CantSell != 1 || logic.CantDepot != 0 {
		t.Errorf("逻辑包联查异常: %+v ok=%v", logic, ok)
	}
	if targets := def.LearnTargets(); targets != nil {
		t.Errorf("未引用学习包不应有目标: %v", targets)
	}

	book, _ := catalog.Definition("book_01")
	if book.Str("Name") != "招式书 & 秘籍" {
		t.Errorf("反转义异常: %q", book.Str("Name"))
	}
	if targets := book.LearnTargets(); len(targets) != 1 || targets[0] != "CS_base_pgd01" {
		t.Errorf("学习目标异常: %v", targets)
	}
	if fp, ok := book.FuncPack(); !ok || fp.PrepareTimeMS != 1500 || fp.PrepareAction != "life_ss_01" || fp.LuaScript != `item\useitem_leitai.lua` {
		t.Errorf("功能包 118 应对上官服读条（1500ms life_ss_01）: %+v ok=%v", fp, ok)
	}
	// 包内无行的引用：联查为 false 而非崩溃。
	lost, _ := catalog.Definition("no_pack")
	if _, ok := lost.FuncPack(); ok {
		t.Error("包内无行的引用不应联查成功")
	}

	// 装载后不持有文件句柄：产物可原地重写（服务运行期间重生成）。
	writeProduct(t, dir, "ItemUse.xml", `<Object><Property ID="rewritten" script="ToolItem" ItemType="1"/></Object>`)
}

func TestLoadXMLMissingProduct(t *testing.T) {
	if _, err := LoadXML(t.TempDir()); err == nil {
		t.Fatal("无登记表应报产物缺席")
	}
	var nilCatalog *Catalog
	if _, ok := nilCatalog.Lookup("x"); ok || nilCatalog.FromXML() || nilCatalog.Count() != 0 || nilCatalog.Source() != "" {
		t.Error("nil 目录应零值安全")
	}
	if _, ok := nilCatalog.Definition("x"); ok {
		t.Error("nil 目录不应有模板行")
	}
}

// ini 回退来源：只有目录面、无模板行；行内 ViewID 优先于逻辑类推断。
func TestINIFallbackHasNoDefinition(t *testing.T) {
	dir := t.TempDir()
	writeProduct(t, dir, "tool_item.ini", "[x_1]\nItemType=5\nViewID=4\nscript=Equipment\nName=任务装备\n")
	catalog, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := catalog.Lookup("x_1")
	if !ok || entry.ViewID != 4 || entry.Name != "任务装备" || entry.Script != "Equipment" {
		t.Fatalf("x_1 异常: %+v", entry)
	}
	if catalog.FromXML() {
		t.Error("ini 来源不应判为 XML")
	}
	if _, ok := catalog.Definition("x_1"); ok {
		t.Error("ini 来源无模板行")
	}
}

// 实档抽检：仓库产物（缺席跳过）。官服抓包里用过的三类物品（武学书/
// 菜肴/坐骑）须能按使用链取值面联查到底。
func TestRepositoryProduct(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "ini", "Item")
	if info, err := os.Stat(filepath.Join(dir, filePathConfig)); err != nil || info.IsDir() {
		t.Skipf("仓库产物缺席（go run ./tools/itemgen 构建）: %v", err)
	}
	start := time.Now()
	catalog, err := LoadXML(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("装载 %d 行 %s（缺席包 %v）", catalog.Count(), time.Since(start), catalog.MissingPacks())
	if n := catalog.Count(); n < 100000 {
		t.Errorf("条目数 %d 偏少（应约 10.5 万）", n)
	}
	book, ok := catalog.Definition("book_CS_base_pgd01")
	if !ok || book.Script() != "SkillBook" || book.Int("FuncPack") != 118 || book.Int("ItemType") != 23 {
		t.Fatalf("武学书异常: %+v ok=%v", book, ok)
	}
	if targets := book.LearnTargets(); len(targets) != 1 || targets[0] != "CS_base_pgd01" {
		t.Errorf("学习目标 %v", targets)
	}
	if fp, _ := book.FuncPack(); fp.PrepareTimeMS != 1500 || fp.PrepareAction != "life_ss_01" {
		t.Errorf("书类读条应为 1500ms life_ss_01（官服抓包 m31 #1 实测）: %+v", fp)
	}
	medicine, _ := catalog.Definition("caiyao10001")
	if us, ok := medicine.UseSkill(); !ok || us.SkillID != "csmanzu" {
		t.Errorf("菜肴使用技能异常: %+v ok=%v", us, ok)
	}
	if fp, ok := medicine.FuncPack(); !ok || fp.PrepareTimeMS != 3000 || fp.PrepareAction != "interact053" {
		t.Errorf("菜肴读条异常: %+v ok=%v", fp, ok)
	}
	entry, _ := catalog.Lookup("ride_jh_whz_0001")
	if entry.Script != "Mount" || entry.ViewID != 1 || entry.MaxAmount != 1 {
		t.Errorf("坐骑目录面异常: %+v", entry)
	}
	if e, _ := catalog.Lookup("taskitem_wujuling_passwordfree"); e.ViewID != 4 {
		t.Errorf("任务页物品应落第 4 页（旧目录面按文件归道具页）: %+v", e)
	}
	if !strings.HasPrefix(catalog.Source(), "xml:") {
		t.Errorf("来源 %q", catalog.Source())
	}
}
