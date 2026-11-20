package fskit

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ---- 哈希方言 ----

func TestHashValueFoldCase(t *testing.T) {
	if HashValue("ABC") != HashValue("abc") {
		t.Fatalf("GetHashValue 应大小写不敏感")
	}
	if HashValueCase("ABC") == HashValueCase("abc") {
		t.Fatalf("GetHashValueCase 应大小写敏感")
	}
	// h*131 + tolower 单调性抽查
	if HashValue("a") != uint32('a') {
		t.Fatalf("单字符哈希 = 首字符 tolower")
	}
	if HashValue("ab") != uint32('a')*131+uint32('b') {
		t.Fatalf("双字符哈希 = c1*131+c2")
	}
}

// ---- Build 解析方言（官方 0x10043200 直译验收）----

func buildIni(t *testing.T, src string) *IniFile {
	t.Helper()
	f := NewIniFile("test")
	if !f.LoadFromString(src) {
		t.Fatalf("LoadFromString 失败")
	}
	return f
}

func TestBuildBasicSections(t *testing.T) {
	f := buildIni(t, "[Sec]\r\nk = v\r\n")
	if f.GetSectionCount() != 1 {
		t.Fatalf("节数 %d", f.GetSectionCount())
	}
	if v, ok := f.GetData("sec", "k"); !ok || v != "v" { // key rtrim、value ltrim
		t.Fatalf("GetData(sec,k)=%q ok=%v", v, ok)
	}
	if !f.FindSection("SEC") {
		t.Fatalf("节查找应大小写不敏感")
	}
}

func TestBuildCommentLines(t *testing.T) {
	f := buildIni(t, "; cmt\r\n// cmt2\r\n[Sec]\r\nk=1\r\n")
	if f.GetItemCount("Sec") != 1 {
		t.Fatalf("注释行不应入表")
	}
}

func TestBuildImplicitSection(t *testing.T) {
	f := buildIni(t, "bare=1\r\n[Sec]\r\nk=2\r\n")
	list := f.GetSectionList()
	if len(list) != 2 || list[0] != "" {
		t.Fatalf("裸 item 应落隐式空名节: %q", list)
	}
	if v, _ := f.GetData("", "bare"); v != "1" {
		t.Fatalf("隐式节读取失败: %q", v)
	}
	// SaveToFile 隐式节输出 "[]"
	if !strings.HasPrefix(f.SaveToString(), "[]\r\n") {
		t.Fatalf("隐式节应输出 []")
	}
}

func TestBuildNoEqualLine(t *testing.T) {
	f := buildIni(t, "[S]\r\nnoeq\r\n")
	if v, ok := f.GetData("S", "noeq"); !ok || v != "" {
		t.Fatalf("无 '=' 行应为 key-only: %q %v", v, ok)
	}
}

func TestBuildTrailingValueKept(t *testing.T) {
	// 官方先整行 rtrim 再切分：行尾空白（含 value 侧）被削
	f := buildIni(t, "[S]\r\nk = v w \r\n")
	if v, _ := f.GetData("S", "k"); v != "v w" {
		t.Fatalf("整行 rtrim 应削尾空白: %q", v)
	}
	// 行内空白保留
	f2 := buildIni(t, "[S]\r\nk = v  w\r\n")
	if v, _ := f2.GetData("S", "k"); v != "v  w" {
		t.Fatalf("值内空白应保留: %q", v)
	}
}

func TestBuildBracketTailBecomesItem(t *testing.T) {
	f := buildIni(t, "[a] tail\r\n")
	list := f.GetSectionList()
	if len(list) != 1 || list[0] != "a" {
		t.Fatalf("']' 后残余应为独立 item 而非节名: %q", list)
	}
	if _, ok := f.GetData("a", "tail"); !ok {
		t.Fatalf("']' 尾 token 应落入新节")
	}
}

func TestBuildNoTrailingNewline(t *testing.T) {
	f := buildIni(t, "[S]\r\nk=v") // 无尾换行
	if v, ok := f.GetData("S", "k"); !ok || v != "v" {
		t.Fatalf("哨兵 \\n 应兜住末行: %q %v", v, ok)
	}
}

func TestBuildLFOnly(t *testing.T) {
	f := buildIni(t, "[S]\nk=v\n")
	if v, _ := f.GetData("S", "k"); v != "v" {
		t.Fatalf("LF 行尾应可解析")
	}
}

func TestSaveToFileDialect(t *testing.T) {
	f := buildIni(t, "[S]\r\nk=v\r\nj=w\r\n")
	want := "[S]\r\nk=v\r\nj=w\r\n\r\n"
	if got := f.SaveToString(); got != want {
		t.Fatalf("Save 方言不符:\n got %q\nwant %q", got, want)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	src := "[S]\r\nk=v\r\n[Empty]\r\n"
	f := buildIni(t, src)
	g := buildIni(t, f.SaveToString())
	if v, ok := g.GetData("S", "k"); !ok || v != "v" {
		t.Fatalf("round-trip 丢数据")
	}
	if g.FindSection("Empty") != true {
		t.Fatalf("空节应保留")
	}
}

// ---- 读 API ----

func TestReadIntegerAtoiPrefix(t *testing.T) {
	f := buildIni(t, "[S]\r\na=12abc\r\nb=-7\r\nc=  9\r\n")
	if f.ReadInteger("S", "a", 0) != 12 {
		t.Fatalf("C atoi 前缀语义")
	}
	if f.ReadInteger("S", "b", 0) != -7 {
		t.Fatalf("负数")
	}
	if f.ReadInteger("S", "c", 0) != 9 {
		t.Fatalf("值 ltrim 后 atoi")
	}
	if f.ReadInteger("S", "missing", 42) != 42 {
		t.Fatalf("miss → def")
	}
}

func TestReadFloat(t *testing.T) {
	f := buildIni(t, "[S]\r\na=1.5\r\nb=2e2\r\n")
	if f.ReadFloat("S", "a", 0) != 1.5 || f.ReadFloat("S", "b", 0) != 200 {
		t.Fatalf("atof 语义: %v %v", f.ReadFloat("S", "a", 0), f.ReadFloat("S", "b", 0))
	}
}

func TestReadStringBufTruncation(t *testing.T) {
	f := buildIni(t, "[S]\r\nk=abcdef\r\n")
	buf := make([]byte, 4)
	res := f.ReadStringBuf("S", "k", buf, "def")
	if string(res) != "abc" || buf[3] != 0 {
		t.Fatalf("maxlen 截断: %q", buf)
	}
	res = f.ReadStringBuf("S", "nope", buf, "xy")
	if string(res) != "xy" {
		t.Fatalf("miss → def 拷贝")
	}
}

// ---- 写 API（SetData/AddData 方言）----

func TestWriteSameValueNoop(t *testing.T) {
	f := buildIni(t, "[S]\r\nk=v1\r\nk2=x\r\n")
	f.WriteString("S", "k", "v1")            // 同值不动
	if got := f.SaveToString(); !strings.Contains(got, "k=v1\r\nk2=x") {
		t.Fatalf("同值 Write 不应改变序/内容: %q", got)
	}
}

func TestWriteNewValueAppends(t *testing.T) {
	f := buildIni(t, "[S]\r\nk=v1\r\n")
	f.WriteString("S", "k", "v2")
	if v, _ := f.GetData("S", "k"); v != "v2" {
		t.Fatalf("改值应生效")
	}
	if f.GetItemCount("S") != 1 {
		t.Fatalf("改值不应加行")
	}
}

func TestWriteCreatesSectionAndItem(t *testing.T) {
	f := buildIni(t, "[S]\r\nk=1\r\n")
	f.WriteInteger("T", "n", 7)
	if f.GetSectionCount() != 2 || !f.FindSection("T") {
		t.Fatalf("Write miss 节应建节")
	}
	if v, ok := f.GetData("T", "n"); !ok || v != "7" {
		t.Fatalf("Write miss key 应插入: %q", v)
	}
	// 新节应在末尾，且不破坏旧节
	if v, _ := f.GetData("S", "k"); v != "1" {
		t.Fatalf("旧节不应受影响")
	}
}

func TestWriteInsertKeepsSectionOrder(t *testing.T) {
	f := buildIni(t, "[A]\r\na=1\r\n[B]\r\nb=2\r\n")
	f.WriteString("A", "a2", "x") // 插到 A 节末（B 节之前）
	got := f.SaveToString()
	ai := strings.Index(got, "a2=x")
	bi := strings.Index(got, "[B]")
	if ai < 0 || ai > bi {
		t.Fatalf("插入应在本节末: %q", got)
	}
	if v, _ := f.GetData("B", "b"); v != "2" {
		t.Fatalf("后续节 firstItem 修正失败")
	}
}

func TestAddDataDuplicates(t *testing.T) {
	f := buildIni(t, "[S]\r\n")
	f.AddString("S", "k", "1")
	f.AddString("S", "k", "2")
	if f.GetItemCount("S") != 2 {
		t.Fatalf("Add* 不查重应重复入表")
	}
	vals := f.GetItemValueList("S")
	if len(vals) != 2 || vals[0] != "1" || vals[1] != "2" {
		t.Fatalf("Add 顺序值: %v", vals)
	}
}

func TestDeleteSectionAndItem(t *testing.T) {
	f := buildIni(t, "[A]\r\na=1\r\na2=2\r\n[B]\r\nb=3\r\n")
	if !f.DeleteItem("A", "a") {
		t.Fatalf("DeleteItem")
	}
	if f.GetItemCount("A") != 1 || f.GetItemCount("B") != 1 {
		t.Fatalf("item 计数")
	}
	if v, _ := f.GetData("B", "b"); v != "3" {
		t.Fatalf("删 item 后后续节读取")
	}
	if !f.DeleteSection("A") {
		t.Fatalf("DeleteSection")
	}
	if f.GetSectionCount() != 1 || !f.FindSection("B") {
		t.Fatalf("删节后剩余节")
	}
	if v, _ := f.GetData("B", "b"); v != "3" {
		t.Fatalf("删节后后续节 firstItem 修正")
	}
}

func TestSectionListOrder(t *testing.T) {
	f := buildIni(t, "[A]\r\n[B]\r\n[C]\r\n")
	if strings.Join(f.GetSectionList(), ",") != "A,B,C" {
		t.Fatalf("节序保持: %v", f.GetSectionList())
	}
}

// ---- split 三兄弟 ----

func TestParseItemsOrderAndDialect(t *testing.T) {
	var got [][3]string
	err := ParseItems([]byte("; cmt\r\n[Sys]\r\n// cmt2\r\nk = v\r\nFsKit.dll\r\n[Main]\r\nIDC=1\r\n"),
		func(sec, key, value string) error {
			got = append(got, [3]string{sec, key, value})
			return nil
		})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	want := [][3]string{
		{"Sys", "k", "v"},
		{"Sys", "FsKit.dll", ""}, // 裸行 key-only（[Module] 模块名方言）
		{"Main", "IDC", "1"},
	}
	if len(got) != len(want) {
		t.Fatalf("条目数 %d: %+v", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("第 %d 条 got %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestParseItemsImplicitSectionAndErrorWrap(t *testing.T) {
	var n int
	err := ParseItems([]byte("bare=1\r\n[S]\r\nk=v\r\n"), func(sec, key, value string) error {
		if n == 0 && (sec != "" || key != "bare" || value != "1") {
			t.Fatalf("隐式空名节应最先: %q/%q=%q", sec, key, value)
		}
		n++
		if n == 2 {
			return errors.New("坏值")
		}
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "S/k: 坏值") {
		t.Fatalf("错误应带节/键前缀: %v", err)
	}
	if n != 2 {
		t.Fatalf("回调应中止: %d", n)
	}
}

func TestSplitStringDefaultComma(t *testing.T) {
	got := SplitString("a,b,c", "")
	want := []string{"a", "b", "c"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("默认分隔符 ,: %v", got)
	}
}

func TestSplitStringCustomSep(t *testing.T) {
	got := SplitString("a|b|c", "|")
	if strings.Join(got, "|") != "a|b|c" {
		t.Fatalf("自定义分隔符: %v", got)
	}
}

func TestSplitStringConsecutiveSkipped(t *testing.T) {
	got := SplitString("a,,b", ",")
	if strings.Join(got, "|") != "a|b" {
		t.Fatalf("连续分隔符空 token 跳过: %v", got)
	}
}

func TestSplitStringTrailingSepEmptyToken(t *testing.T) {
	got := SplitString("a,b,", ",")
	if strings.Join(got, "|") != "a|b|" {
		t.Fatalf("尾随分隔符 → 末尾空串: %v", got)
	}
}

func TestSplitStringEmpty(t *testing.T) {
	if got := SplitString("", ","); len(got) != 0 {
		t.Fatalf("空输入零输出: %v", got)
	}
}

func TestSplitFileName(t *testing.T) {
	cases := []struct{ in, base, ext string }{
		{"a.txt", "a", "txt"},
		{"a.b.c", "a.b", "c"},
		{"noext", "noext", ""},
		{".hidden", "", "hidden"},
		{"dir/file.v", "dir/file", "v"}, // 遇 / 前无点 → 整名；有点 → 正常切
		{"dir.d/file", "dir.d/file", ""},
	}
	for _, c := range cases {
		got := SplitFileName(c.in)
		if got[0] != c.base || got[1] != c.ext {
			t.Fatalf("SplitFileName(%q) = %v, want [%s %s]", c.in, got, c.base, c.ext)
		}
	}
}

func TestSplitFilePath(t *testing.T) {
	got := SplitFilePath("a/b.txt")
	if len(got) != 2 {
		t.Fatalf("应有目录+文件两元素: %v", got)
	}
	if !strings.HasSuffix(got[0], "\\") && !strings.HasSuffix(got[0], "/") {
		t.Fatalf("目录应含尾分隔符: %q", got[0])
	}
	if got[1] != "b.txt" {
		t.Fatalf("文件名: %q", got[1])
	}
	// 无文件部分 → 单元素
	if got := SplitFilePath("dir/"); len(got) != 1 {
		t.Fatalf("尾分隔符路径单元素: %v", got)
	}
}

// ---- get_file_time / gen_unique_name / get_full_path ----

func TestGetFileTime(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.txt")
	os.WriteFile(p, []byte("1"), 0644)
	got := GetFileTime(p)
	// 官方格式 "%04d-%02d-%02d %02d:%02d:%02d" 且为 UTC
	re := regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)
	if !re.MatchString(got) {
		t.Fatalf("格式: %q", got)
	}
	fi, _ := os.Stat(p)
	u := fi.ModTime().UTC()
	want := strings.TrimSpace(u.Format("2006-01-02 15:04:05"))
	if got != want {
		t.Fatalf("应为 UTC 最后写时间: got %s want %s", got, want)
	}
	if GetFileTime(filepath.Join(dir, "nope")) != "" {
		t.Fatalf("失败 → 空串")
	}
}

func TestGenUniqueName(t *testing.T) {
	re := regexp.MustCompile(`^[0-9A-F]{32}$`)
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		s := GenUniqueName()
		if !re.MatchString(s) {
			t.Fatalf("32 位大写 hex 无连字符: %q", s)
		}
		if seen[s] {
			t.Fatalf("重复: %s", s)
		}
		seen[s] = true
	}
}

func TestGetFullPath(t *testing.T) {
	got := GetFullPath("a/b.txt")
	if !filepath.IsAbs(got) {
		t.Fatalf("应为绝对路径: %q", got)
	}
	if !strings.HasSuffix(got, "b.txt") {
		t.Fatalf("应保留文件名: %q", got)
	}
}

// ---- MainConsole ----

func TestMainConsoleColorTable(t *testing.T) {
	if ColorAttr["White"] != 0x0F || ColorAttr["LightRed"] != 0x0C || ColorAttr["Blue"] != 1 {
		t.Fatalf("官方色值表")
	}
}

func TestMainConsoleWriteLineCRLF(t *testing.T) {
	m := NewMainConsole()
	m.Out = &strings.Builder{}
	m.WriteLine("hi")
	if m.Out.(*strings.Builder).String() != "hi\r\n" {
		t.Fatalf("WriteLine 应 CRLF")
	}
}

func TestMainConsoleSetColorUnknown(t *testing.T) {
	m := NewMainConsole()
	m.Out = &strings.Builder{}
	m.SetColor("NotAColor")
	if m.GetColor() != "NotAColor" {
		t.Fatalf("未知名也应存名")
	}
	m.SetColor("lightred") // stricmp 不区分大小写
	if m.GetColor() != "lightred" {
		t.Fatalf("存原名")
	}
}

// ---- FileSearch ----

func TestFileSearchDirAndFile(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "sub1"), 0755)
	os.Mkdir(filepath.Join(dir, "sub2"), 0755)
	os.WriteFile(filepath.Join(dir, "a.ini"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("x"), 0644)

	// SearchDir：目录枚举（排除 . ..）
	fs := NewFileSearch()
	if !fs.SearchDir(dir + "\\") {
		t.Fatalf("SearchDir")
	}
	if fs.GetDirCount() != 2 {
		t.Fatalf("目录数: %v", fs.GetDirList2())
	}
	if fs.GetPath() != dir+"\\" {
		t.Fatalf("Path 属性")
	}

	// 无点文件名 → "*.ini" 扩展名方言
	if !fs.SearchFile(dir+"\\", "ini") {
		t.Fatalf("SearchFile ext 方言")
	}
	if fs.GetFileCount() != 1 {
		t.Fatalf("ini 扩展应只匹配 a.ini: %v", fs.GetFileList2())
	}
	// 空 → *.*（全部文件）
	if !fs.SearchFile(dir+"\\", "") {
		t.Fatalf("SearchFile 全部")
	}
	if fs.GetFileCount() != 2 {
		t.Fatalf("*.* 应匹配全部文件: %v", fs.GetFileList2())
	}
	// 含点 → 原样名
	if !fs.SearchFile(dir+"\\", "a.ini") {
		t.Fatalf("SearchFile 精确")
	}
	if fs.GetFileCount() != 1 {
		t.Fatalf("精确名匹配: %v", fs.GetFileList2())
	}
}

// 测试辅助：直接取 []string（等价 GetDirList/GetFileList 填 VarList）。
func (fs *FileSearch) GetDirList2() []string  { return fs.dirList }
func (fs *FileSearch) GetFileList2() []string { return fs.fileList }

func TestFileSearchVarList(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("x"), 0644)
	fs := NewFileSearch()
	fs.SearchFile(dir+"\\", "")
	var out VarList
	fs.GetFileList(&out)
	if out.Count() != 2 || out.GetKind(0) != VarString {
		t.Fatalf("GetFileList 应填字符串 VarList: n=%d", out.Count())
	}
}

// ---- ArrayList ----

func TestArrayListChildren(t *testing.T) {
	r := NewRegistry()
	a := r.NewArrayList("root")
	id1 := a.CreateChild(r, "A")
	id2 := a.CreateChild(r, "B")
	if id1 == 0 || id2 == 0 || id1 == id2 {
		t.Fatalf("CreateChild 发号")
	}
	if a.GetChildCount() != 2 {
		t.Fatalf("子数")
	}
	if !a.FindChild("A") || a.FindChild("a") {
		t.Fatalf("子名查找应大小写敏感")
	}
	if a.GetChild("B") != id2 || a.GetChildByIndex(0) != id1 {
		t.Fatalf("GetChild/ByIndex")
	}
	// GetChildList 名单
	if strings.Join(a.GetChildList(), ",") != "A,B" {
		t.Fatalf("GetChildList: %v", a.GetChildList())
	}
	// Remove / Clear
	if !a.RemoveChild("A") || a.GetChildCount() != 1 {
		t.Fatalf("RemoveChild")
	}
	if !a.RemoveChildByIndex(0) || a.GetChildCount() != 0 {
		t.Fatalf("RemoveChildByIndex")
	}
	a.CreateChild(r, "C")
	if !a.ClearChild() || a.GetChildCount() != 0 {
		t.Fatalf("ClearChild")
	}
	_ = r.Get(id1)
}

func TestArrayListAddChildForeign(t *testing.T) {
	r := NewRegistry()
	a := r.NewArrayList("a")
	b := r.NewArrayList("b")
	if !a.AddChild(r, b.id) {
		t.Fatalf("AddChild 收养外部实体")
	}
	if a.AddChild(r, b.id) {
		t.Fatalf("重复 AddChild 应失败")
	}
	if a.AddChild(r, 99999) {
		t.Fatalf("未知 id 应失败")
	}
}

// ---- IniDocument 包装 ----

func TestIniDocumentRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.ini")
	os.WriteFile(p, []byte("[S]\r\nk=v\r\n"), 0644)
	d := NewIniDocument(p)
	if d.GetFileName() != p {
		t.Fatalf("FileName 属性")
	}
	if !d.LoadFromFile() || d.ReadString("S", "k", "") != "v" {
		t.Fatalf("Load+Read")
	}
	d.WriteInteger("S", "n", 5)
	if !d.SaveToFile() {
		t.Fatalf("SaveToFile")
	}
	d2 := NewIniDocument(p)
	d2.LoadFromFile()
	if d2.ReadInteger("S", "n", 0) != 5 || d2.ReadString("S", "k", "") != "v" {
		t.Fatalf("落盘 round-trip")
	}
}

// ---- 模块注册表 ----

func TestModuleDecls(t *testing.T) {
	if ModuleType != "FxModule" {
		t.Fatalf("ModuleType")
	}
	names := map[string]bool{}
	for _, e := range EntityTypes {
		names[e.TypeName] = true
	}
	for _, want := range []string{"ArrayList", "FileSearch", "IniDocument", "MainConsole"} {
		if !names[want] {
			t.Fatalf("缺实体类型 %s", want)
		}
	}
	fns := map[string]bool{}
	for _, f := range ExtFuncs {
		fns[f.Name] = true
	}
	for _, want := range []string{"ext_split_string", "ext_split_file_path", "ext_split_file_name", "ext_get_file_time", "ext_get_full_path", "ext_gen_unique_name"} {
		if !fns[want] {
			t.Fatalf("缺扩展函数 %s", want)
		}
	}
	// ext 分发直通
	var args, res VarList
	args.AddString("a,b")
	args.AddString("")
	if !dispatchExt("ext_split_string", &args, &res) || res.GetString(0) != "a" || res.GetString(1) != "b" {
		t.Fatalf("ext_split_string 分发")
	}
}

func dispatchExt(name string, args, res *VarList) bool {
	for _, f := range ExtFuncs {
		if f.Name == name {
			return f.Call(args, res)
		}
	}
	return false
}

// ---- VarList ----

func TestVarListAssign(t *testing.T) {
	var a VarList
	a.AddString("x")
	a.AddString("y")
	var b VarList
	b.Assign(&a, 0, 1)
	if b.Count() != 1 || b.GetString(0) != "x" {
		t.Fatalf("Assign(0,1)")
	}
	b.Clear()
	if b.Count() != 0 {
		t.Fatalf("Clear")
	}
}
