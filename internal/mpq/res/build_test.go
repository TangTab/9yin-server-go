package res

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/mpq"
)

func mustBuild(t *testing.T, root string, opt BuildOptions) *BuildResult {
	t.Helper()
	res, err := BuildLayer(root, opt)
	if err != nil {
		t.Fatalf("BuildLayer(%s, %+v)：%v", root, opt, err)
	}
	return res
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func TestBuildFirstLayerFromLooseFiles(t *testing.T) {
	root := t.TempDir()
	putFile(t, filepath.Join(root, "ini", "a.txt"), "a")
	putFile(t, filepath.Join(root, "ini", "Sub", "b.txt"), "b")
	putFile(t, filepath.Join(root, "mpq", "fixture.bin"), "fx")

	res := mustBuild(t, root, BuildOptions{Prefix: PrefixTest, Exclude: []string{"mpq"}})
	if res.Number != 1 || res.File != filepath.Join(root, "b001.rep") || !res.Changed ||
		res.Replaced || res.Added != 2 || res.Excluded != 1 || res.Skipped != 0 || res.Pruned != 0 {
		t.Fatalf("构建结果不符：%+v", res)
	}

	// 不修剪：散文件仍在，合并视图里全部是 local。
	store := openForTest(t, root)
	mustLayer(t, store, `ini\a.txt`, "local")

	// 散文件拿掉后从层里读到，内容逐字节一致。
	for _, rel := range []string{filepath.Join("ini", "a.txt"), filepath.Join("ini", "Sub", "b.txt")} {
		if err := os.Remove(filepath.Join(root, rel)); err != nil {
			t.Fatal(err)
		}
	}
	mustLayer(t, store, `ini\Sub\b.txt`, "b001")
	mustRead(t, store, `ini\a.txt`, "a")

	// 层容器本身不是资源，不会被下一层收进去。
	res = mustBuild(t, root, BuildOptions{Prefix: PrefixTest, Exclude: []string{"mpq"}})
	if res.Changed || res.Excluded != 1 || fileExists(filepath.Join(root, "b002.rep")) {
		t.Fatalf("只有层容器与被排除文件时不应产生新层：%+v", res)
	}
}

func TestBuildIncrementalLayerHoldsOnlyChanges(t *testing.T) {
	root := t.TempDir()
	putFile(t, filepath.Join(root, "a.txt"), "a1")
	putFile(t, filepath.Join(root, "b.txt"), "b")
	mustBuild(t, root, BuildOptions{Prefix: PrefixTest})

	// 改一个、加一个；b 没动。
	putFile(t, filepath.Join(root, "a.txt"), "a2")
	putFile(t, filepath.Join(root, "c.txt"), "c")
	res := mustBuild(t, root, BuildOptions{Prefix: PrefixTest})
	if res.Number != 2 || res.Added != 2 || res.Skipped != 1 || res.Deleted != nil {
		t.Fatalf("增量层应只含改动：%+v", res)
	}

	archive, err := mpq.Open(res.File)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if archive.HasFile(`b.txt`) || !archive.HasFile(`a.txt`) || !archive.HasFile(`c.txt`) {
		t.Fatal("b002 应只含 a.txt 与 c.txt")
	}

	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.Remove(filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	store := openForTest(t, root)
	mustLayer(t, store, `a.txt`, "b002")
	mustLayer(t, store, `c.txt`, "b002")
	mustLayer(t, store, `b.txt`, "b001")

	// 散文件盖在两层之上。
	putFile(t, filepath.Join(root, "a.txt"), "loose")
	mustRead(t, store, `a.txt`, "loose")
}

func TestBuildPruneRemovesAbsorbedLooseFiles(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "ini", "a.txt")
	subB := filepath.Join(root, "ini", "Sub", "b.txt")
	fixture := filepath.Join(root, "mpq", "fixture.bin")
	backup := filepath.Join(root, "ini", "Skill", "heijin_backup", "skill_static.xml")
	putFile(t, a, "a")
	putFile(t, subB, "b")
	putFile(t, fixture, "fx")
	putFile(t, backup, "backup")
	opt := BuildOptions{Prefix: PrefixTest, Prune: true, Exclude: []string{"mpq", "ini/*/heijin_backup"}}

	res := mustBuild(t, root, opt)
	if res.Pruned != 2 || fileExists(a) || fileExists(subB) {
		t.Fatalf("已并入的散文件应被删除：%+v", res)
	}
	if !fileExists(fixture) || !fileExists(backup) {
		t.Fatal("被排除的文件不动")
	}
	// 变空的目录被清掉；仍有内容的目录保留。
	if _, err := os.Stat(filepath.Join(root, "ini", "Sub")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("空目录 ini/Sub 应被清除：%v", err)
	}
	if !fileExists(backup) {
		t.Fatal("ini 目录不应被清除")
	}

	// 再放一个与层内容相同的散文件：被去重并修剪，不产生新层。
	putFile(t, a, "a")
	res = mustBuild(t, root, opt)
	if res.Changed || res.Skipped != 1 || res.Pruned != 1 || fileExists(a) ||
		fileExists(filepath.Join(root, "b002.rep")) {
		t.Fatalf("与下层相同的散文件不应产生新层：%+v", res)
	}

	// -from 指向别处时不修剪源。
	external := t.TempDir()
	putFile(t, filepath.Join(external, "x.txt"), "x")
	res = mustBuild(t, root, BuildOptions{Prefix: PrefixTest, Prune: true, Source: external})
	if res.Pruned != 0 || !fileExists(filepath.Join(external, "x.txt")) {
		t.Fatalf("外部源不应被修剪：%+v", res)
	}
}

func TestBuildReplaceRebuildsWholeLayer(t *testing.T) {
	root, full := t.TempDir(), t.TempDir()
	putFile(t, filepath.Join(full, "a.txt"), "a1")
	putFile(t, filepath.Join(full, "gone.txt"), "gone")
	mustBuild(t, root, BuildOptions{Prefix: PrefixTest, Source: full})

	// 目标层已存在：默认拒绝，免得用不完整的源把它冲掉。
	if _, err := BuildLayer(root, BuildOptions{Prefix: PrefixTest, Number: 1, Source: full}); err == nil {
		t.Fatal("重建已存在的层应要求显式开启 Replace")
	}

	putFile(t, filepath.Join(full, "a.txt"), "a2")
	if err := os.Remove(filepath.Join(full, "gone.txt")); err != nil {
		t.Fatal(err)
	}
	res := mustBuild(t, root, BuildOptions{Prefix: PrefixTest, Number: 1, Replace: true, Source: full})
	if !res.Replaced || !res.Changed || res.Added != 1 {
		t.Fatalf("重建结果不符：%+v", res)
	}
	store := openForTest(t, root)
	if len(store.Layers()) != 1 {
		t.Fatalf("重建不应多出层：%+v", store.Layers())
	}
	mustRead(t, store, `a.txt`, "a2")
	mustLayer(t, store, `gone.txt`, "")
}

func TestBuildDeleteMissingRegistersTombstones(t *testing.T) {
	root, full := t.TempDir(), t.TempDir()
	for _, f := range []string{"a.txt", "b.txt", "gone.txt"} {
		putFile(t, filepath.Join(full, f), "v1")
	}
	mustBuild(t, root, BuildOptions{Prefix: PrefixData, Source: full})

	// 新版完整树：a 没变、b 改了、gone 没了、多了 new。
	putFile(t, filepath.Join(full, "b.txt"), "v2")
	putFile(t, filepath.Join(full, "new.txt"), "n")
	if err := os.Remove(filepath.Join(full, "gone.txt")); err != nil {
		t.Fatal(err)
	}
	res := mustBuild(t, root, BuildOptions{Prefix: PrefixData, Source: full, DeleteMissing: true})
	if res.Number != 2 || res.Added != 2 || res.Skipped != 1 || strings.Join(res.Deleted, ",") != "gone.txt" {
		t.Fatalf("增量层结果不符：%+v", res)
	}
	store := openForTest(t, root)
	if got := store.Prefix(); got != PrefixData {
		t.Fatalf("前缀 = %q，应为 data", got)
	}
	mustLayer(t, store, `a.txt`, "data001")
	mustLayer(t, store, `b.txt`, "data002")
	mustLayer(t, store, `new.txt`, "data002")
	mustLayer(t, store, `gone.txt`, "")

	// 重建 data002：须先关掉自己持有的仓（Windows 上占用着无法替换）。
	// 源里仍没有 gone：原有的删除名单保留。
	putFile(t, filepath.Join(full, "b.txt"), "v3")
	store.Close()
	res = mustBuild(t, root, BuildOptions{Prefix: PrefixData, Number: 2, Replace: true, Source: full})
	if strings.Join(res.Deleted, ",") != "gone.txt" {
		t.Fatalf("重建应保留本层原有的删除名单：%+v", res)
	}
	store = openForTest(t, root)
	mustRead(t, store, `b.txt`, "v3")
	mustLayer(t, store, `gone.txt`, "")

	// 不开 DeleteMissing：缺失的名字不登记删除，低层内容继续可见。
	putFile(t, filepath.Join(full, "gone.txt"), "back")
	if err := os.Remove(filepath.Join(full, "new.txt")); err != nil {
		t.Fatal(err)
	}
	res = mustBuild(t, root, BuildOptions{Prefix: PrefixData, Source: full})
	if res.Deleted != nil {
		t.Fatalf("未开启 DeleteMissing 不应登记删除：%+v", res)
	}
	if err := store.Reload(); err != nil {
		t.Fatal(err)
	}
	mustLayer(t, store, `new.txt`, "data002")
	mustLayer(t, store, `gone.txt`, "data003")
}

func TestBuildRejectsBadOptions(t *testing.T) {
	root := t.TempDir()
	putFile(t, filepath.Join(root, "a.txt"), "a")
	cases := map[string]BuildOptions{
		"缺前缀":    {Exclude: []string{"x"}},
		"未知前缀":   {Prefix: "res-"},
		"格式版本越界": {Prefix: PrefixTest, Format: 5},
		"层号越界":   {Prefix: PrefixTest, Number: -1},
	}
	for name, opt := range cases {
		if _, err := BuildLayer(root, opt); err == nil {
			t.Errorf("%s：应报错", name)
		}
	}
	if _, err := BuildLayer(filepath.Join(root, "a.txt"), BuildOptions{Prefix: PrefixTest}); err == nil {
		t.Error("数据根不是目录应报错")
	}
	if entries, _ := os.ReadDir(root); len(entries) != 1 {
		t.Errorf("失败的构建不应留下文件：%v", entries)
	}
}

func TestBuildAllFormatVersionsRoundTrip(t *testing.T) {
	for format := 1; format <= 4; format++ {
		root := t.TempDir()
		putFile(t, filepath.Join(root, "a.txt"), "a")
		res := mustBuild(t, root, BuildOptions{Prefix: PrefixTest, Format: format})
		archive, err := mpq.Open(res.File)
		if err != nil {
			t.Fatal(err)
		}
		if got := archive.Version(); got != format {
			t.Errorf("格式 v%d：写出 v%d", format, got)
		}
		archive.Close()
		mustRead(t, openForTest(t, root), `a.txt`, "a")
	}
}

func TestDeleteRegistersInTopLayer(t *testing.T) {
	root := t.TempDir()
	looseC := filepath.Join(root, "c.txt")
	writeArchive(t, filepath.Join(root, "b001.rep"), []mpq.Addition{
		{Name: `a.txt`, Data: []byte("a")},
		{Name: `b.txt`, Data: []byte("b")},
	})
	putFile(t, looseC, "c")

	// 同名散文件不受登记约束：报出来让调用方先移走。
	shadowed, err := Delete(root, Options{}, `a.txt`, "c.txt")
	if err != nil || len(shadowed) != 1 || shadowed[0] != looseC {
		t.Fatalf("Delete = %v（err=%v）", shadowed, err)
	}
	store := openForTest(t, root)
	mustLayer(t, store, `a.txt`, "")
	mustLayer(t, store, `c.txt`, "local")
	mustLayer(t, store, `b.txt`, "b001")
	if err := os.Remove(looseC); err != nil {
		t.Fatal(err)
	}
	mustLayer(t, store, `c.txt`, "")

	// 新增一层后，删除登记进最高层；低层内容原样保留。
	// Delete 改写最高层文件：先关掉自己持有的仓，完了再开。
	writeArchive(t, filepath.Join(root, "b002.rep"), []mpq.Addition{{Name: `d.txt`, Data: []byte("d")}})
	store.Close()
	if _, err := Delete(root, Options{Prefix: PrefixTest}, `b.txt`); err != nil {
		t.Fatal(err)
	}
	store = openForTest(t, root)
	layers := store.Layers()
	// b001 里留着最初登记的两条；b.txt 的删除只进当时的最高层 b002。
	if len(layers) != 2 || layers[0].Deleted != 2 || layers[1].Deleted != 1 {
		t.Fatalf("删除应登记在当时的最高层：%+v", layers)
	}
	mustLayer(t, store, `a.txt`, "")
	mustLayer(t, store, `b.txt`, "")
	mustLayer(t, store, `d.txt`, "b002")

	// 更高层重新收录即恢复。
	writeArchive(t, filepath.Join(root, "b003.rep"), []mpq.Addition{{Name: `a.txt`, Data: []byte("back")}})
	if err := store.Reload(); err != nil {
		t.Fatal(err)
	}
	mustRead(t, store, `a.txt`, "back")
}

func TestDeleteNeedsALayerAndNames(t *testing.T) {
	if _, err := Delete(t.TempDir(), Options{}, `a.txt`); err == nil {
		t.Fatal("没有层文件时无法登记删除")
	}
	root := t.TempDir()
	writeArchive(t, filepath.Join(root, "b001.rep"), []mpq.Addition{{Name: `a.txt`, Data: []byte("a")}})
	if _, err := Delete(root, Options{}); err == nil {
		t.Fatal("未给出名字应报错")
	}
}

func TestUnpackWritesMergedView(t *testing.T) {
	root := t.TempDir()
	writeArchive(t, filepath.Join(root, "b001.rep"), []mpq.Addition{
		{Name: `a.txt`, Data: []byte("1")},
		{Name: `Sub\b.txt`, Data: []byte("1")},
		{Name: `gone.txt`, Data: []byte("1")},
	})
	writeArchive(t, filepath.Join(root, "b002.rep"), []mpq.Addition{
		{Name: `a.txt`, Data: []byte("2")},
		deletedMark(`gone.txt`),
	})
	putFile(t, filepath.Join(root, "Sub", "b.txt"), "L")
	store := openForTest(t, root)

	dest := filepath.Join(t.TempDir(), "out")
	if n, err := Unpack(store, dest, ""); err != nil || n != 2 {
		t.Fatalf("Unpack = %d（err=%v）", n, err)
	}
	for rel, want := range map[string]string{
		"a.txt":     "2",
		"Sub/b.txt": "L",
		"gone.txt":  "",
		"b001.rep":  "",
		"b002.rep":  "",
	} {
		data, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(rel)))
		if want == "" {
			if !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("%s 不应被解出：%q（err=%v）", rel, data, err)
			}
			continue
		}
		if err != nil || string(data) != want {
			t.Errorf("解出的 %s = %q（err=%v），应为 %q", rel, data, err, want)
		}
	}

	part := filepath.Join(t.TempDir(), "sub")
	if n, err := Unpack(store, part, "sub/"); err != nil || n != 1 {
		t.Fatalf("按前缀解包 = %d（err=%v）", n, err)
	}
	if !fileExists(filepath.Join(part, "Sub", "b.txt")) {
		t.Fatal("按前缀解包缺少 Sub/b.txt")
	}
}

func TestExcludedRuleMatching(t *testing.T) {
	cases := []struct {
		rel   string
		rules []string
		want  bool
	}{
		{"mpq/x.bin", []string{"mpq"}, true},
		{"mpq", []string{"mpq"}, true},
		{"ini/mpq/x.bin", []string{"mpq"}, false},
		{"ini/Skill/heijin_backup/a.xml", []string{"ini/*/heijin_backup"}, true},
		{"ini/heijin_backup/a.xml", []string{"ini/*/heijin_backup"}, false},
		{"a/heijin_backup/deep/x", []string{"*/heijin_backup/"}, true},
		{"ini/Skill/a.xml", []string{"ini/*/heijin_backup"}, false},
		{"a/b/c.txt", []string{"a/*/c.txt"}, true},
		{"a/b/d.txt", []string{"a/*/c.txt"}, false},
		{"README.md", []string{"README.md"}, true},
		{"ini/README.md", []string{"README.md"}, false},
		{"anything", nil, false},
	}
	for _, c := range cases {
		if got := excluded(c.rel, c.rules); got != c.want {
			t.Errorf("excluded(%q, %q) = %v，应为 %v", c.rel, c.rules, got, c.want)
		}
	}
}

// 参考资源族同样可以构建：前缀 h → h001.rep。
func TestBuildReferenceFamily(t *testing.T) {
	root := t.TempDir()
	putFile(t, filepath.Join(root, "ini", "a.txt"), "a")
	res := mustBuild(t, root, BuildOptions{Prefix: PrefixReference})
	if res.File != filepath.Join(root, "h001.rep") {
		t.Fatalf("层文件 = %s，应为 h001.rep", res.File)
	}
	mustLayer(t, openForTest(t, root), `ini\a.txt`, "local")
}
