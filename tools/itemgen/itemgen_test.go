package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/config"
)

// ini 解析：CRLF、三种注释、表头元数据节、节内重复键、残留合并冲突标记。
func TestParseINI(t *testing.T) {
	raw := []byte("\xEF\xBB\xBF; 顶部注释\r\n[-1]\r\nLuaScript=0\r\n[0]\r\nLuaScript=6\r\n" +
		"[book_01]\r\nName=秘籍\r\nscript=SkillBook\r\n// 行注释\r\n# 井号注释\r\n" +
		"<<<<<<< .mine=\r\n[yaopin_01]\r\nUseSkillPack=11\r\nUseSkillPack=12\r\n>>>>>>> .r1=\r\n[带 空格]\r\nK=a=b\r\n")
	table := parseINI(raw, "fixture.ini")
	if len(table.Sections) != 5 {
		t.Fatalf("节数 %d, want 5: %+v", len(table.Sections), table.Sections)
	}
	if len(table.ConflictMarkers) != 2 {
		t.Fatalf("冲突标记应 2 处, got %v", table.ConflictMarkers)
	}
	if name, _ := table.Sections[2].Get("Name"); name != "秘籍" {
		t.Errorf("Name=%q", name)
	}
	if len(table.Sections[3].Keys) != 2 || table.Sections[3].Vals[1] != "12" {
		t.Errorf("重复键应保留两次: %+v", table.Sections[3].Keys)
	}
	if v, _ := table.Sections[4].Get("K"); v != "a=b" {
		t.Errorf("值内等号应保留: %q", v)
	}
}

// 分档：优先序即判定序（武器族 → 装备 → 镶嵌 → 坐骑 → 金钱 → 武学书 →
// 礼盒 → 任务页按 FuncPack 分可用/材料 → 材料页 → 其余道具）。
func TestKindOf(t *testing.T) {
	cases := []struct {
		row  map[string]string
		want string
	}{
		{map[string]string{"script": "Equipment", "EquipType": "Weapon", "ViewID": "2"}, "Weapon"},
		{map[string]string{"script": "Equipment", "EquipType": "innerweapon"}, "Weapon"},
		{map[string]string{"script": "ShotWeapon", "EquipType": "shotweapon"}, "Weapon"},
		{map[string]string{"script": "Equipment", "EquipType": "cloth", "ViewID": "2"}, "Equipment"},
		{map[string]string{"script": "FacultyPaint", "ViewID": "2"}, "Equipment"},
		{map[string]string{"script": "ToolItem", "EquipType": "hat"}, "Equipment"},
		{map[string]string{"script": "Enchase", "ViewID": "1"}, "Rune"},
		{map[string]string{"script": "Sable", "ViewID": "1"}, "ItemMountUse"},
		{map[string]string{"script": "DropMoneyItem", "ViewID": "1"}, "add_money"},
		{map[string]string{"script": "SkillPage", "ViewID": "4"}, "SkillBook"},
		{map[string]string{"script": "BoxItem", "ViewID": "4", "FuncPack": "103"}, "ItemBox"},
		{map[string]string{"script": "ToolItem", "ViewID": "4", "FuncPack": "42"}, "TaskItemUse"},
		{map[string]string{"script": "ToolItem", "ViewID": "4", "FuncPack": "0"}, "TaskMaterial"},
		{map[string]string{"script": "ToolItem", "ViewID": "4"}, "TaskMaterial"},
		{map[string]string{"script": "fish", "ViewID": "3"}, "Material"},
		{map[string]string{"script": "ToolItem", "ViewID": "1"}, "ItemUse"},
		{map[string]string{"script": "Bag", "ViewID": "2"}, "ItemUse"},
		{map[string]string{"ItemType": "2093"}, "ItemUse"},
	}
	for _, c := range cases {
		if got := kindOf(c.row); got != c.want {
			t.Errorf("kindOf(%v)=%s want %s", c.row, got, c.want)
		}
	}
}

// 记录键改名：@X → XRec，已带 Rec 后缀的不变；非记录键不认。
func TestRecordAttr(t *testing.T) {
	for key, want := range map[string]string{
		"@ItemType": "ItemTypeRec", "@PointPos": "PointPosRec", "@PointPos2": "PointPos2Rec",
		"@CompensateItemRec": "CompensateItemRec", "@DelayBindItemRec": "DelayBindItemRec",
	} {
		got, ok := recordAttr(key)
		if !ok || got != want {
			t.Errorf("recordAttr(%q)=%q,%t want %q", key, got, ok, want)
		}
	}
	if got, ok := recordAttr("ItemType"); ok {
		t.Errorf("标量键不应判为记录键: %q", got)
	}
	if got, ok := recordAttr("@9bad"); ok {
		t.Errorf("数字开头的记录键应判非法: %q", got)
	}
}

func testVocabulary() *vocabulary {
	return &vocabulary{
		structProps: map[string]bool{"ItemType": true, "ExpiredTime": true},
		heijinAttrs: map[string]bool{"script": true, "Name": true},
		defineNames: map[string]bool{"PrepareTime": true},
	}
}

// 模板档：@记录键多行并串且不与标量撞名、前置列序、空值不成列、非法键
// 丢弃、重复键 _2、空节跳过、无 script 记账；并经服务端装载器回读一致。
func TestKindBuilderRowsAndAudits(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"[-1]", "Name=0", "[0]", "Name=6",
		"[//拜师入门专用]", "",
		"[gem_001]", "ItemType=73", "@ItemType=140,141", "@ItemType=142", "script=Enchase", "Name=臂力", "",
		"[Operation_item_0001]", "",
		"[yaopin_01]", "script=ToolItem", "ItemType=1", "Name=回血药", "SellPrice1=", "UseSkillPack=11", "UseSkillPack=12", "]=junk", "",
		"[no_script_item]", "ItemType=2093", "Name=制鞋录", "",
	}, "\r\n"))
	builder := newKindBuilder(kindSpecs[3]) // ItemUse
	for _, section := range parseINI(raw, "tool_item.ini").Sections {
		if section.Name == "-1" || section.Name == "0" || strings.HasPrefix(section.Name, "//") {
			continue
		}
		builder.addSection(section)
	}
	file, report := builder.finish(testVocabulary())
	if report.Rows != 3 || report.EmptySections != 1 {
		t.Fatalf("行 %d 空节 %d, want 3/1: %+v", report.Rows, report.EmptySections, report)
	}
	text := string(file.Data)
	want := "\t<Property ID=\"gem_001\" script=\"Enchase\" ItemType=\"73\" Name=\"臂力\" ItemTypeRec=\"140,141;142\"/>\n"
	if !strings.Contains(text, want) {
		t.Errorf("gem_001 行不符:\n got=%s\nwant=%s", text, want)
	}
	if strings.Contains(text, "ItemType=\"73;") {
		t.Error("@ItemType 不应并入标量 ItemType")
	}
	if !strings.Contains(text, `UseSkillPack="11" UseSkillPack_2="12"`) || strings.Contains(text, "junk") {
		t.Errorf("yaopin_01 行不符: %s", text)
	}
	if strings.Contains(text, "SellPrice1") {
		t.Error("空值属性不应写出")
	}
	if report.NoScript != 1 || report.DuplicateKey != 1 || len(report.InvalidKeys) != 1 || report.InvalidKeys[0] != "]" {
		t.Errorf("审计记账不符: %+v", report)
	}
	kinds := map[string]string{}
	for _, column := range report.Columns {
		kinds[column.Attr] = column.Kind
	}
	if kinds["script"] != kindHeijin || kinds["ItemType"] != kindStruct || kinds["ItemTypeRec"] != kindRename ||
		kinds["UseSkillPack"] != kindNative {
		t.Errorf("列出处不符: %v", kinds)
	}
	assertVerify(t, file)
}

// 静态包：表头两行照留（列宽 0 也是值）、数据行列序取 [-1] 列序头、表头
// 未列键殿后；回读一致。
func TestBuildPackKeepsHeaderRows(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"[-1]", "PrepareTime=2", "LuaScript=0", "PrepareAction=1", "",
		"[0]", "PrepareTime=2", "LuaScript=6", "PrepareAction=6", "",
		"[1]", "PrepareTime=1000", "",
		"[118]", "LuaScript=item\\useitem_leitai.lua", "PrepareAction=life_ss_01", "PrepareTime=1500", "GuildTask=1", "",
		"//注释", "",
	}, "\r\n"))
	spec := packSpec{Source: "item_func_static.ini", Output: "item_func_static.xml", Ref: "FuncPack", Title: "功能包"}
	file, report, rows := buildPack(parseINI(raw, spec.Source), spec, testVocabulary())
	if report.Rows != 2 || len(file.IDs) != 4 || file.IDs[0] != "-1" || file.IDs[1] != "0" {
		t.Fatalf("表头/行数不符: %+v ids=%v", report, file.IDs)
	}
	text := string(file.Data)
	// 表头行按键在源表的出现序（黑金已发布档同形）；数据行列序按 [-1] 列序头。
	if !strings.Contains(text, `<Property ID="-1" PrepareTime="2" LuaScript="0" PrepareAction="1"/>`) {
		t.Errorf("[-1] 列序头行不符: %s", text)
	}
	want := "\t<Property ID=\"118\" LuaScript=\"item\\useitem_leitai.lua\" PrepareAction=\"life_ss_01\" PrepareTime=\"1500\" GuildTask=\"1\"/>\n"
	if !strings.Contains(text, want) {
		t.Errorf("数据行列序应按 [-1] 列序头:\n got=%s\nwant=%s", text, want)
	}
	if rows["118"]["PrepareTime"] != "1500" || rows["1"]["LuaScript"] != "" {
		t.Errorf("数据行属性不符: %v", rows)
	}
	assertVerify(t, file)
}

// 类型树：最内层包含区间的下界为父；孤立 _MAX 不成区间；同值多名合并。
func TestParseItemTypeList(t *testing.T) {
	text := strings.Join([]string{
		"ITEMTYPE_TOOLITEM_MIN = 1", "ITEMTYPE_ADDHP = 1", "ITEMTYPE_ADDMP = 2",
		"ITEMTYPE_TOOLITEM_MAX = 99", "ITEMTYPE_OFFLINE_MIN = 90", "ITEMTYPE_OFFLINE_MAX = 99",
		"ITEMTYPE_OFFLINE_KEY = 98", "ITEMTYPE_GAME_MARKET_MAX = 499",
		"ITEMTYPE_EQUIP_MIN = 100", "ITEMTYPE_WEAPON_MIN = 100", "ITEMTYPE_WEAPON_MAX = 119",
		"ITEMTYPE_WEAPON_BLADE = 101", "ITEMTYPE_EQUIP_HAT = 140", "ITEMTYPE_EQUIP_MAX = 199",
		"ITEMTYPE_ZHAOSHI = 1000", "-- 注释行 ITEMTYPE_X = 7", "local n = 5",
	}, "\n")
	entries := parseItemTypeList(text)
	got := map[int]int{}
	names := map[int]string{}
	for _, entry := range entries {
		got[entry.ID] = entry.Father
		names[entry.ID] = strings.Join(entry.Names, "/")
	}
	for id, want := range map[int]int{1: itemTypeRoot, 2: 1, 90: 1, 98: 90, 99: 90, 100: itemTypeRoot, 101: 100, 140: 100, 499: itemTypeRoot, 1000: itemTypeRoot, 119: 100, 199: 100} {
		if got[id] != want {
			t.Errorf("类型 %d 父类型 %d, want %d: %v", id, got[id], want, got)
		}
	}
	if names[1] != "ADDHP/TOOLITEM_MIN" || names[100] != "EQUIP_MIN/WEAPON_MIN" {
		t.Errorf("同值多名应合并: %q %q", names[1], names[100])
	}
	if len(entries) != 12 {
		t.Errorf("类型数 %d want 12（注释与局部变量不入表）", len(entries))
	}
	file := buildItemTypeList(entries)
	if !strings.HasSuffix(string(file.Data), `<Property ID="55555" FatherType="55555" Desc="全部"/>`+"\n</Object>\n") {
		t.Errorf("根 55555 应殿后: %s", file.Data)
	}
}

// 端到端：微型 share/item 夹具 → run() → 全部产物就位、登记表逐档、
// 引用完整性与未转换清单入报告、verify 通过。
func TestRunEndToEnd(t *testing.T) {
	in := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(in, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tool_item.ini", strings.Join([]string{
		"[-1]", "Name=0", "[0]", "Name=6",
		"[book_CS_01]", "Name=招式书", "script=SkillBook", "ItemType=23", "ViewID=1", "ColorLevel=2", "FuncPack=118", "LearnSkillPack=3837", "MaxAmount=1", "",
		"[yaopin_00100]", "Name=药", "script=ToolItem", "ItemType=1", "ViewID=1", "FuncPack=9", "UseSkillPack=4444", "LogicPack=1", "",
		"[task_mark]", "Name=星图", "script=ToolItem", "ItemType=1200", "ViewID=4", "",
		"[money_1]", "Name=钱袋", "script=CapitalItem", "ItemType=37", "ViewID=1", "FuncPack=1", "CapitalType=1", "CapitalValue=100", "",
		"[add_money_item00001]", "Name=金钱掉落", "script=DropMoneyItem", "ItemType=1100", "ViewID=1", "MaxAmount=100000000", "",
	}, "\n"))
	write("equipment.ini", "[cloth_01]\nName=布衣\nscript=Equipment\nItemType=141\nViewID=2\nEquipType=cloth\nArtPack=3243\n")
	write("tool_item_hw.ini", "[haiwai_box]\nName=海外礼盒\nscript=BoxItem\nItemType=2102\nViewID=1\nFuncPack=103\nDropID=drop_x\n")
	write("giftbox.ini", "[box_gift]\nName=礼包\nscript=BoxItem\nItemType=2102\nViewID=1\nFuncPack=103\nDropID=drop_y\n")
	write("item_func_static.ini", "[-1]\nLuaScript=0\nPrepareAction=1\nPrepareTime=2\n[0]\nLuaScript=6\n[1]\nPrepareTime=1000\n[118]\nPrepareAction=life_ss_01\nPrepareTime=1500\n")
	for _, name := range []string{"item_logical_static", "itemartstatic", "item_use_skill_static", "item_learn_skill_static",
		"item_task_static", "item_ingredient_static", "item_prop_modify_static", "item_seed_static"} {
		write(name+".ini", "[-1]\nA=0\n[0]\nA=6\n[1]\nA=1\n")
	}
	write("fuse_formula.ini", "[1]\nComposeResult=x\n")
	write("life_formula.ini", "[hc_01]\nFormulaType=1\n")
	write("item_copy_static.ini", "")
	if err := os.MkdirAll(filepath.Join(in, "innerequip"), 0o755); err != nil {
		t.Fatal(err)
	}
	define := filepath.Join(t.TempDir(), "itemtype_define.lua")
	writeDefine := func() {
		if err := os.WriteFile(define, []byte("ITEMTYPE_WEAPON_MIN = 100\nITEMTYPE_WEAPON_BLADE = 101\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeDefine()
	empty := filepath.Join(t.TempDir(), "share", "item_static_data_define.lua")
	structRoot := filepath.Join("..", "..", "testdata", "struct")

	out := t.TempDir()
	reportPath := filepath.Join(t.TempDir(), "report.json")
	if _, err := run(in, out, "", structRoot, empty, define, reportPath, true); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, name := range []string{"path_config.xml", "ItemTypeList.xml", "Equipment.xml", "ItemUse.xml", "SkillBook.xml",
		"TaskMaterial.xml", "add_money.xml", "ItemBox.xml", "item_func_static.xml", "item_logic_pack.xml", "ItemArtStatic.xml"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("产物缺席 %s: %v", name, err)
		}
	}
	// 空档不产出、不登记（本夹具无武器/镶嵌行）。
	for _, name := range []string{"Weapon.xml", "Rune.xml"} {
		if _, err := os.Stat(filepath.Join(out, name)); err == nil {
			t.Errorf("空档不应产出 %s", name)
		}
	}
	doc, err := config.LoadObjectFile(filepath.Join(out, "path_config.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var types []string
	for _, row := range doc.Rows {
		types = append(types, row["Type"])
	}
	sort.Strings(types)
	if got := strings.Join(types, ","); got != "Equipment,ItemBox,ItemUse,SkillBook,Taskmaterial,add_money" {
		t.Errorf("登记表 Type 集 %q", got)
	}
	// 分档归属。
	box, _ := config.LoadObjectFile(filepath.Join(out, "ItemBox.xml"))
	if len(box.Rows) != 2 || box.Rows[0]["ID"] != "haiwai_box" || box.Rows[0]["DropID"] != "drop_x" {
		t.Errorf("礼盒档应含 hw/giftbox 两行: %+v", box.Rows)
	}
	books, _ := config.LoadObjectFile(filepath.Join(out, "SkillBook.xml"))
	if books.Rows[0]["FuncPack"] != "118" || books.Rows[0]["LearnSkillPack"] != "3837" {
		t.Errorf("武学书行不符: %+v", books.Rows[0])
	}
	// 报告：引用完整性、使用链、未转换清单、verify。
	blob, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report genReport
	if err := json.Unmarshal(blob, &report); err != nil {
		t.Fatal(err)
	}
	if report.Verify == nil || len(report.Verify.Failures) != 0 || report.FilesOut == 0 {
		t.Errorf("verify 应通过: %+v", report.Verify)
	}
	refOf := func(attr string) *refReport {
		for i := range report.Refs {
			if report.Refs[i].Attr == attr {
				return &report.Refs[i]
			}
		}
		return nil
	}
	if ref := refOf("FuncPack"); ref == nil || ref.Refs != 5 || ref.Resolved != 2 || ref.Unresolved != 3 {
		t.Errorf("FuncPack 引用完整性不符: %+v", ref)
	}
	if ref := refOf("ArtPack"); ref == nil || ref.Refs != 1 || ref.Unresolved != 1 {
		t.Errorf("ArtPack 引用完整性不符: %+v", ref)
	}
	uc := report.UseChain
	if uc == nil || uc.WithFuncPack != 5 || uc.FuncPackResolved != 2 || uc.BoxItems != 2 || uc.CapitalItems != 1 ||
		uc.LearnSkillItems != 1 || uc.UseSkillItems != 1 {
		t.Errorf("使用链统计不符: %+v", uc)
	}
	skipped := map[string]string{}
	for _, s := range report.Skipped {
		skipped[s.File] = s.Reason
	}
	if !strings.HasPrefix(skipped["fuse_formula.ini"], "合成域") || !strings.HasPrefix(skipped["life_formula.ini"], "生活技能域") ||
		!strings.HasPrefix(skipped["item_copy_static.ini"], "空表") || !strings.HasPrefix(skipped["innerequip"], "内功装备域") {
		t.Errorf("未转换清单不符: %v", skipped)
	}
	if len(report.Skipped) != 4 {
		t.Errorf("未转换数 %d want 4: %v", len(report.Skipped), skipped)
	}
}

// assertVerify 单文件落盘后经 config.LoadObjectFile 回读一致。
func assertVerify(t *testing.T, file *builtFile) {
	t.Helper()
	out := t.TempDir()
	path := filepath.Join(out, filepath.FromSlash(file.RelPath))
	if err := os.WriteFile(path, file.Data, 0o644); err != nil {
		t.Fatal(err)
	}
	if report := verifyWritten(out, []*builtFile{file}); len(report.Failures) != 0 {
		t.Fatalf("verify 失败: %v", report.Failures)
	}
}
