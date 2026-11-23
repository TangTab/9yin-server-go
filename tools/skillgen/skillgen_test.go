package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/config"
)

// ini 解析：CRLF、三种注释、表头元数据节、节内重复键、残留合并冲突
// 标记（单侧无 =======）、节外行告警。
func TestParseINI(t *testing.T) {
	raw := []byte("\xEF\xBB\xBF; 顶部注释\r\n节外=1\r\n[-1]\r\nCoolDownTime=0\r\n[0]\r\nCoolDownTime=2\r\n" +
		"[zs_normal_00]\r\nStaticData=13\r\nscript=SkillNormal\r\n// 行注释\r\n# 井号注释\r\n" +
		"<<<<<<< .mine=\r\n[CS_jh_cqgf01]\r\nStaticData=4401\r\nscript=SkillLock\r\n" +
		"UseLimit=25\r\nUseLimit=26\r\n>>>>>>> .r759612=\r\n[带 空格]\r\nK=a=b\r\n")
	table := parseINI(raw, "fixture.ini")
	if len(table.Sections) != 5 {
		t.Fatalf("节 数 %d, want 5: %+v", len(table.Sections), table.Sections)
	}
	if len(table.ConflictMarkers) != 2 {
		t.Fatalf("冲突标记应 2 处, got %v", table.ConflictMarkers)
	}
	if len(table.Warnings) != 2 {
		t.Fatalf("告警应 2 条（冲突标记+节外行）, got %v", table.Warnings)
	}
	basic := table.Sections[2]
	if basic.Name != "zs_normal_00" || basic.Line != 7 {
		t.Errorf("普攻节名/行号 %q/%d", basic.Name, basic.Line)
	}
	if sd, _ := basic.Get("StaticData"); sd != "13" {
		t.Errorf("StaticData=%q", sd)
	}
	lock := table.Sections[3]
	if len(lock.Keys) != 4 || lock.Vals[3] != "26" {
		t.Errorf("重复键应保留两次: %+v", lock.Keys)
	}
	if v, _ := table.Sections[4].Get("K"); v != "a=b" {
		t.Errorf("值内等号应保留: %q", v)
	}
}

// 转换：按逻辑类分组、前置列序、审定改名、空值跳过、重复键 _2、非法键
// 丢弃、无组键归 Unknown；并经 config.LoadObjectFile 回读一致。
func TestConvertTableGroupsAndAudits(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"[a_skill]", "ItemType=1000", "StaticData=13", "script=SkillNormal", "QTypeID=2", "AddCoolDownTime=2000", "UseLimit=25", "",
		"[b_skill]", "StaticData=4401", "script=SkillLock", "ItemType=1001", "AddCoolDownTime=", "PauseTime=30000", "",
		"[no_script]", "StaticData=7", "ItemType=1000", "",
		"[d_skill]", "script=SkillNormal", "StaticData=9", "UseLimit=1", "UseLimit=2", "]=junk", "",
	}, "\r\n"))
	spec := tableSpec{
		Source: "skill_new.ini", Output: "Skill_New/%s.xml", Group: "script",
		Lead:   []string{"script", "StaticData", "ItemType", "QTypeID"},
		Rename: map[string]string{"UseLimit": "Skill_Use_Limit"},
	}
	vocab := &vocabulary{
		structProps: map[string]bool{"ItemType": true, "Skill_Use_Limit": true},
		heijinAttrs: map[string]bool{"script": true},
		defineNames: map[string]bool{"StaticData": true},
	}
	build := convertTable(parseINI(raw, "skill_new.ini"), spec, vocab)
	if build.Rows != 4 || len(build.Files) != 3 {
		t.Fatalf("行 %d 文件 %d, want 4/3: %+v", build.Rows, len(build.Files), build.Files)
	}
	if build.DuplicateKeys != 1 || len(build.InvalidKeys) != 1 || build.InvalidKeys[0] != "]" {
		t.Errorf("重复键 %d 非法键 %v", build.DuplicateKeys, build.InvalidKeys)
	}
	var normal *builtFile
	for _, file := range build.Files {
		switch file.RelPath {
		case "Skill_New/SkillNormal.xml":
			normal = file
		case "Skill_New/SkillLock.xml", "Skill_New/Unknown.xml":
		default:
			t.Errorf("意外输出 %s", file.RelPath)
		}
	}
	if normal == nil {
		t.Fatal("缺 SkillNormal 分档")
	}
	text := string(normal.Data)
	want := "\t<Property ID=\"a_skill\" script=\"SkillNormal\" StaticData=\"13\" ItemType=\"1000\" QTypeID=\"2\" AddCoolDownTime=\"2000\" Skill_Use_Limit=\"25\"/>\n"
	if !strings.Contains(text, want) {
		t.Errorf("a_skill 行不符:\n got=%s\nwant=%s", text, want)
	}
	// 重复键 _2 变体、非法键不出现。
	if !strings.Contains(text, `Skill_Use_Limit="1" Skill_Use_Limit_2="2"`) || strings.Contains(text, "junk") {
		t.Errorf("d_skill 行不符: %s", text)
	}
	// 列出处报告：direct/heijin/define/rename/native。
	kinds := map[string]string{}
	for _, column := range normal.Columns {
		kinds[column.Attr] = column.Kind
	}
	for attr, want := range map[string]string{
		"ItemType": kindDirect, "script": kindHeijin, "StaticData": kindDefine,
		"Skill_Use_Limit": kindRename, "AddCoolDownTime": kindNative,
	} {
		if kinds[attr] != want {
			t.Errorf("列 %s 出处 %s, want %s", attr, kinds[attr], want)
		}
	}
	// 空值属性不写（b_skill 的 AddCoolDownTime）。
	for _, file := range build.Files {
		if file.RelPath == "Skill_New/SkillLock.xml" && strings.Contains(string(file.Data), "AddCoolDownTime") {
			t.Errorf("空值属性不应写出: %s", file.Data)
		}
	}
	// 回读一致（服务端装载器）。
	out := t.TempDir()
	for _, file := range build.Files {
		path := filepath.Join(out, filepath.FromSlash(file.RelPath))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, file.Data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if report := verifyWritten(out, build.Files); len(report.Failures) != 0 {
		t.Fatalf("verify 失败: %v", report.Failures)
	}
	doc, err := config.LoadObjectFile(filepath.Join(out, "Skill_New", "SkillNormal.xml"))
	if err != nil || len(doc.Rows) != 2 || doc.Rows[1]["Skill_Use_Limit_2"] != "2" {
		t.Fatalf("回读 %v %+v", err, doc)
	}
}

// 表头元数据行只在源表存在时记为已丢弃。
func TestConvertTableHeaders(t *testing.T) {
	raw := []byte("[config]\r\nGRushRange=5.0\r\nGRushRangeAdd=1.5\r\n")
	spec := tableSpec{Source: "rush_range.ini", Output: "Skill_RushRange.xml", Headers: []string{"-1", "0"}}
	build := convertTable(parseINI(raw, spec.Source), spec, &vocabulary{})
	if len(build.HeadersDropped) != 0 || build.Rows != 1 {
		t.Fatalf("无表头行不应记账: %+v", build)
	}
	build = convertTable(parseINI([]byte("[-1]\r\nA=0\r\n[0]\r\nA=2\r\n[1]\r\nA=5\r\n"), spec.Source), spec, &vocabulary{})
	if len(build.HeadersDropped) != 2 || build.Rows != 1 {
		t.Fatalf("表头行应丢弃 2 记 2: %+v", build)
	}
}

func TestValidAttrName(t *testing.T) {
	for name, want := range map[string]bool{
		"StaticData": true, "_x": true, "a-b.c_1": true, "": false, "9a": false,
		"]": false, "<<<<<<< .mine": false, "裂天拳": false,
	} {
		if got := validAttrName(name); got != want {
			t.Errorf("validAttrName(%q)=%t want %t", name, got, want)
		}
	}
}

// 服务端 struct schema 属性名装载（skillbase + includes，注释掉的 include
// 不跟随）。
func TestLoadStructProperties(t *testing.T) {
	props, err := loadStructProperties(filepath.Join("..", "..", "testdata", "struct"))
	if err != nil {
		t.Skipf("struct schema 缺席: %v", err)
	}
	for _, name := range []string{"ItemType", "QTypeID", "Level", "Skill_Use_Limit",
		"CoolDownTime", "AddCoolDownTime", "AConsumeCurMP", "TargetShapePara2", "HitCount", "HitTime", "PrepareTime"} {
		if !props[name] {
			t.Errorf("struct schema 应含属性 %s", name)
		}
	}
	if props["BreakTime"] {
		t.Error("注释掉的属性 BreakTime 不应计入")
	}
	// 词汇差异实证：struct（黑金词汇）用 AConsumeCurHP，无九阴运行时列名
	// AConsumeMP（官服 Lua 按后者取属性：buf_jn_lmsj_03.lua:31）。
	if props["AConsumeMP"] || !props["AConsumeCurHP"] {
		t.Error("struct schema 词汇应为 AConsumeCurHP/无 AConsumeMP")
	}
}

// 首跑让位：黑金样本移入 heijin_backup/；重跑（已有产物/已有备份）绝不动
// <out> 根下的文件——九阴静态包产物与黑金样本 skill_static.xml 同名，
// 误搬会把九阴产物当黑金档。
func TestBackupHeiJinFirstRunOnly(t *testing.T) {
	out := t.TempDir()
	for _, name := range heijinSamples {
		if err := os.WriteFile(filepath.Join(out, name), []byte("<Object/>\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	note, err := backupHeiJin(out)
	if err != nil || !strings.Contains(note, "7 项已移入") {
		t.Fatalf("首跑应移入 7 项: %q %v", note, err)
	}
	for _, name := range heijinSamples {
		if _, err := os.Stat(filepath.Join(out, "heijin_backup", name)); err != nil {
			t.Errorf("样本 %s 未进备份", name)
		}
	}
	note, err = backupHeiJin(out)
	if err != nil || !strings.Contains(note, "已存在") {
		t.Fatalf("二次运行应跳过: %q %v", note, err)
	}

	// 无备份目录但已有无歧义产物（非首跑）：跳过。
	fresh := t.TempDir()
	product := filepath.Join(fresh, "Skill_Consume.xml")
	if err := os.WriteFile(product, []byte("<Object/>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	note, err = backupHeiJin(fresh)
	if err != nil || !strings.Contains(note, "非首跑") {
		t.Fatalf("已有产物应跳过备份: %q %v", note, err)
	}

	// 样本残缺（此处只剩与产物同名的 skill_static.xml 一个）：不搬，
	// 同名文件原样保留——它同样可能是九阴静态包产物。
	partial := t.TempDir()
	ambiguousFile := filepath.Join(partial, "skill_static.xml")
	if err := os.WriteFile(ambiguousFile, []byte("<Object/>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	note, err = backupHeiJin(partial)
	if err != nil || !strings.Contains(note, "无完整") {
		t.Fatalf("残缺样本集应跳过: %q %v", note, err)
	}
	if _, err := os.Stat(ambiguousFile); err != nil {
		t.Fatalf("同名文件被误搬: %v", err)
	}
	if _, err := os.Stat(filepath.Join(partial, "heijin_backup")); !os.IsNotExist(err) {
		t.Fatalf("残缺集不应建备份目录: %v", err)
	}
}
