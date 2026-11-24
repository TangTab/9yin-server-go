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

// 编号多层资源仓：散文件 > b002 > b001；登记删除对其以下各层生效。

// createArchive 用 mpq 写入侧构建一个 v1 测试归档。
func createArchive(t *testing.T, files []mpq.Addition) []byte {
	t.Helper()
	return archiveOf(t, mpq.V1, files)
}

func archiveOf(t *testing.T, version mpq.FormatVersion, files []mpq.Addition) []byte {
	t.Helper()
	base, err := mpq.Create(mpq.CreateOptionsOfVersion(version))
	if err != nil {
		t.Fatal(err)
	}
	result, err := mpq.Modify(base, nil, files)
	if err != nil {
		t.Fatal(err)
	}
	return result.Bytes
}

// writeArchive 在路径上写一个指定版本的层归档。
func writeArchive(t *testing.T, path string, files []mpq.Addition, versions ...mpq.FormatVersion) {
	t.Helper()
	version := mpq.V1
	if len(versions) > 0 {
		version = versions[0]
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, archiveOf(t, version, files), 0o644); err != nil {
		t.Fatal(err)
	}
}

// deletedMark 构造一层"只登记删除"所需的 (deleted) 条目。
func deletedMark(names ...string) mpq.Addition {
	return mpq.Addition{Name: deletedFile, Data: deletedList(names)}
}

func putFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// openForTest 打开资源根；Windows 上句柄会挡住 TempDir 清理，先关再清。
func openForTest(t *testing.T, dir string) *Store {
	t.Helper()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open(%s)：%v", dir, err)
	}
	t.Cleanup(store.Close)
	return store
}

func mustRead(t *testing.T, s *Store, name, want string) {
	t.Helper()
	data, err := s.Read(name)
	if err != nil {
		t.Fatalf("Read(%q)：%v", name, err)
	}
	if string(data) != want {
		t.Fatalf("Read(%q) = %q，应为 %q", name, data, want)
	}
}

func mustLayer(t *testing.T, s *Store, name, want string) {
	t.Helper()
	if got := s.Layer(name); got != want {
		t.Fatalf("Layer(%q) = %q，应为 %q", name, got, want)
	}
}

func TestNumberedLayersOverride(t *testing.T) {
	dir := t.TempDir()
	writeArchive(t, filepath.Join(dir, "b001.rep"), []mpq.Addition{
		{Name: `ini\a.txt`, Data: []byte("b001 a")},
		{Name: `ini\Sub\b.txt`, Data: []byte("b001 b")},
		{Name: `ini\gone.txt`, Data: []byte("gone by b002")},
		{Name: `ini\keep.txt`, Data: []byte("b001 keep")},
	})
	writeArchive(t, filepath.Join(dir, "b002.rep"), []mpq.Addition{
		{Name: `ini\a.txt`, Data: []byte("b002 a")},
		{Name: `ini\c.txt`, Data: []byte("b002 c")},
		deletedMark(`ini\gone.txt`),
	}, mpq.V4)
	putFile(t, filepath.Join(dir, "ini", "Sub", "b.txt"), "local b")

	store := openForTest(t, dir)
	if got := store.Prefix(); got != PrefixTest {
		t.Fatalf("Prefix = %q，应为 %q", got, PrefixTest)
	}

	// n 覆盖 n-1，散文件覆盖一切；登记删除对其以下各层生效。
	for name, want := range map[string]string{
		`ini\a.txt`:     "b002",
		`ini\Sub\b.txt`: "local",
		`ini\keep.txt`:  "b001",
		`ini\c.txt`:     "b002",
		`ini\gone.txt`:  "",
		`ini\nope.txt`:  "",
		`..\b002.rep`:   "",      // 路径穿越被拒绝
		`b002.rep`:      "local", // 层容器按主机文件对待：可读，但不进合并清单
		"(listfile)":    "",      // 内部文件不是资源
		"(deleted)":     "",
	} {
		mustLayer(t, store, name, want)
	}
	mustRead(t, store, `ini\a.txt`, "b002 a")
	mustRead(t, store, `ini\Sub\b.txt`, "local b")
	mustRead(t, store, `ini\keep.txt`, "b001 keep")
	mustRead(t, store, `ini\c.txt`, "b002 c")

	// 名字规范化：/ 与 \、大小写不敏感。
	mustRead(t, store, "INI/a.Txt", "b002 a")
	mustLayer(t, store, "ini/sub/B.TXT", "local")

	// 登记删除：读取报 ErrNotExist。
	if store.Exists(`ini\gone.txt`) {
		t.Fatal("登记删除的名字不应存在")
	}
	if _, err := store.Read(`ini\gone.txt`); !errors.Is(err, fs.ErrNotExist) || !os.IsNotExist(err) {
		t.Fatalf("读登记删除的名字应报 ErrNotExist，实得：%v", err)
	}

	// Lookup 不读内容即给出大小与层。
	if size, layer, ok := store.Lookup(`ini\keep.txt`); !ok || size != 9 || layer != "b001" {
		t.Fatalf("Lookup(keep) = %d,%q,%v", size, layer, ok)
	}

	// 合并清单：a(b002) + c(b002) + keep(b001) + Sub/b(local)，gone 不在。
	files, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range files {
		got = append(got, f.Layer+":"+f.Name)
	}
	if want := "b002:ini\\a.txt,b002:ini\\c.txt,b001:ini\\keep.txt,local:ini\\Sub\\b.txt"; strings.Join(got, ",") != want {
		t.Fatalf("合并清单 = %v，应为 %s", got, want)
	}

	layers := store.Layers()
	if len(layers) != 2 || layers[0].Name != "b001" || layers[1].Name != "b002" ||
		layers[0].Deleted != 0 || layers[1].Deleted != 1 {
		t.Fatalf("层概览不符：%+v", layers)
	}
}

func TestLayerOrderIsNumericNotLexical(t *testing.T) {
	dir := t.TempDir()
	writeArchive(t, filepath.Join(dir, "b100.rep"), []mpq.Addition{{Name: `x.txt`, Data: []byte("100")}})
	writeArchive(t, filepath.Join(dir, "b010.rep"), []mpq.Addition{{Name: `x.txt`, Data: []byte("010")}})
	writeArchive(t, filepath.Join(dir, "b009.rep"), []mpq.Addition{{Name: `x.txt`, Data: []byte("009")}})
	store := openForTest(t, dir)
	mustLayer(t, store, `x.txt`, "b100")
	mustRead(t, store, `x.txt`, "100")

	// 序号不补零也认（b1 与 b001 同为第 1 层）。
	other := t.TempDir()
	writeArchive(t, filepath.Join(other, "b1.rep"), []mpq.Addition{{Name: `x.txt`, Data: []byte("one")}})
	mustRead(t, openForTest(t, other), `x.txt`, "one")
}

func TestDeletionIsReinstatedByHigherLayerOrLoose(t *testing.T) {
	dir := t.TempDir()
	writeArchive(t, filepath.Join(dir, "b001.rep"), []mpq.Addition{{Name: `x.txt`, Data: []byte("v1")}})
	writeArchive(t, filepath.Join(dir, "b002.rep"), []mpq.Addition{deletedMark(`x.txt`)})
	store := openForTest(t, dir)
	mustLayer(t, store, `x.txt`, "")

	writeArchive(t, filepath.Join(dir, "b003.rep"), []mpq.Addition{{Name: `x.txt`, Data: []byte("v3")}})
	if err := store.Reload(); err != nil {
		t.Fatal(err)
	}
	mustRead(t, store, `x.txt`, "v3")

	putFile(t, filepath.Join(dir, "x.txt"), "loose")
	mustRead(t, store, `x.txt`, "loose")
}

func TestBadLayerSetsAreRejected(t *testing.T) {
	valid := []mpq.Addition{{Name: `x.txt`, Data: []byte("x")}}

	t.Run("同层号两个文件", func(t *testing.T) {
		dir := t.TempDir()
		writeArchive(t, filepath.Join(dir, "b001.rep"), valid)
		writeArchive(t, filepath.Join(dir, "b1.rep"), valid)
		if _, err := Open(dir); err == nil || !strings.Contains(err.Error(), "层号 1 重复") {
			t.Fatalf("应报层号重复，实得：%v", err)
		}
	})
	t.Run("层号从零起", func(t *testing.T) {
		dir := t.TempDir()
		writeArchive(t, filepath.Join(dir, "b000.rep"), valid)
		if _, err := Open(dir); err == nil {
			t.Fatal("b000.rep 应被拒绝")
		}
	})
	t.Run("两族并存须指定前缀", func(t *testing.T) {
		dir := t.TempDir()
		writeArchive(t, filepath.Join(dir, "b001.rep"), valid)
		writeArchive(t, filepath.Join(dir, "data001.rep"), []mpq.Addition{{Name: `x.txt`, Data: []byte("data")}})
		if _, err := Open(dir); err == nil {
			t.Fatal("两族并存且未指定前缀应报错")
		}
		mustRead(t, openStoreWith(t, dir, Options{Prefix: PrefixData}), `x.txt`, "data")
		mustRead(t, openStoreWith(t, dir, Options{Prefix: PrefixTest}), `x.txt`, "x")
	})
	t.Run("b 与 h 并存须指定前缀", func(t *testing.T) {
		dir := t.TempDir()
		writeArchive(t, filepath.Join(dir, "b001.rep"), valid)
		writeArchive(t, filepath.Join(dir, "h001.rep"), []mpq.Addition{{Name: `x.txt`, Data: []byte("ref")}})
		if _, err := Open(dir); err == nil {
			t.Fatal("两族并存且未指定前缀应报错")
		}
		mustRead(t, openStoreWith(t, dir, Options{Prefix: PrefixReference}), `x.txt`, "ref")
	})
	t.Run("同层既收录又登记删除", func(t *testing.T) {
		dir := t.TempDir()
		writeArchive(t, filepath.Join(dir, "b001.rep"), append(append([]mpq.Addition(nil), valid...), deletedMark(`x.txt`)))
		if _, err := Open(dir); err == nil {
			t.Fatal("自相矛盾的层应被拒绝")
		}
	})
	t.Run("层文件损坏", func(t *testing.T) {
		dir := t.TempDir()
		putFile(t, filepath.Join(dir, "b001.rep"), "not an archive")
		if _, err := Open(dir); err == nil {
			t.Fatal("损坏的层应报错")
		}
	})
	t.Run("无效前缀", func(t *testing.T) {
		if _, err := OpenWith(t.TempDir(), Options{Prefix: "res-"}); err == nil {
			t.Fatal("无效前缀应报错")
		}
	})
}

func openStoreWith(t *testing.T, dir string, opt Options) *Store {
	t.Helper()
	store, err := OpenWith(dir, opt)
	if err != nil {
		t.Fatalf("OpenWith(%s, %+v)：%v", dir, opt, err)
	}
	t.Cleanup(store.Close)
	return store
}

func TestLooseOnlyRootAndFamilyDetection(t *testing.T) {
	// 没有任何层文件的数据根：纯散文件仓，打开不报错。
	dir := t.TempDir()
	putFile(t, filepath.Join(dir, "ini", "a.txt"), "loose only")
	store := openForTest(t, dir)
	if store.Prefix() != "" || len(store.Layers()) != 0 {
		t.Fatalf("纯散文件仓不应有层：%q %+v", store.Prefix(), store.Layers())
	}
	mustLayer(t, store, `ini\a.txt`, "local")
	mustRead(t, store, `ini/a.txt`, "loose only")

	// 只有一族时自动探测到它。
	data := t.TempDir()
	writeArchive(t, filepath.Join(data, "DATA002.rep"), []mpq.Addition{{Name: `x.txt`, Data: []byte("data")}})
	s := openForTest(t, data)
	if s.Prefix() != PrefixData {
		t.Fatalf("应探测到 data 族，实得 %q", s.Prefix())
	}
	mustLayer(t, s, `x.txt`, "data002")

	// 不是 .rep 或序号不合法的名字不是层，只是普通散文件。
	other := t.TempDir()
	putFile(t, filepath.Join(other, "b001.rep.bak"), "bak")
	putFile(t, filepath.Join(other, "res-base.rep"), "legacy")
	s = openForTest(t, other)
	if len(s.Layers()) != 0 {
		t.Fatalf("不应识别出层：%+v", s.Layers())
	}
	mustLayer(t, s, `res-base.rep`, "local")
}

func TestReloadPicksUpLayerChanges(t *testing.T) {
	dir := t.TempDir()
	writeArchive(t, filepath.Join(dir, "b001.rep"), []mpq.Addition{{Name: `a.txt`, Data: []byte("v1")}})
	store := openForTest(t, dir)
	mustRead(t, store, `a.txt`, "v1")

	// 新增的层在 Reload 之前不可见（层集合在打开时确定）。
	writeArchive(t, filepath.Join(dir, "b002.rep"), []mpq.Addition{{Name: `b.txt`, Data: []byte("new")}})
	writeArchive(t, filepath.Join(dir, "b001.rep"), []mpq.Addition{{Name: `a.txt`, Data: []byte("v2")}})
	mustLayer(t, store, `b.txt`, "")

	if err := store.Reload(); err != nil {
		t.Fatal(err)
	}
	mustRead(t, store, `a.txt`, "v2")
	mustRead(t, store, `b.txt`, "new")

	// 删掉最高层后 Reload：相应名字回到低层。
	store.Close()
	os.Remove(filepath.Join(dir, "b002.rep"))
	if err := store.Reload(); err != nil {
		t.Fatal(err)
	}
	mustLayer(t, store, `b.txt`, "")
	mustRead(t, store, `a.txt`, "v2")
}

// unreadable 层在 Reload 时不应把已打开的好层一并丢掉。
func TestReloadFailureKeepsCurrentLayers(t *testing.T) {
	dir := t.TempDir()
	writeArchive(t, filepath.Join(dir, "b001.rep"), []mpq.Addition{{Name: `a.txt`, Data: []byte("v1")}})
	store := openForTest(t, dir)
	putFile(t, filepath.Join(dir, "b002.rep"), "corrupt")
	if err := store.Reload(); err == nil {
		t.Fatal("损坏的新层应使 Reload 报错")
	}
	mustRead(t, store, `a.txt`, "v1")
}

// 参考资源族 hNNN.rep（黑金原档等）：与 b / data 同等对待——自动探测、
// 按层号覆盖。
func TestReferenceFamily(t *testing.T) {
	dir := t.TempDir()
	writeArchive(t, filepath.Join(dir, "h001.rep"), []mpq.Addition{
		{Name: `ini\Item\a.xml`, Data: []byte("h001")},
	})
	writeArchive(t, filepath.Join(dir, "h002.rep"), []mpq.Addition{
		{Name: `ini\Item\a.xml`, Data: []byte("h002")},
		{Name: `ini\Item\b.xml`, Data: []byte("h002 b")},
	})
	store := openForTest(t, dir)
	if got := store.Prefix(); got != PrefixReference {
		t.Fatalf("Prefix = %q，应为 %q", got, PrefixReference)
	}
	mustRead(t, store, `ini\Item\a.xml`, "h002")
	mustRead(t, store, `ini\Item\b.xml`, "h002 b")
	if l := store.Layers(); len(l) != 2 || l[0].Name != "h001" || l[1].Name != "h002" {
		t.Fatalf("层概览不符：%+v", l)
	}
}

// 全部族前缀都认；大小写与首尾空白不影响判定。
func TestIsPrefix(t *testing.T) {
	for _, ok := range []string{"b", "data", "h", "H", " Data "} {
		if !IsPrefix(ok) {
			t.Errorf("IsPrefix(%q) 应为 true", ok)
		}
	}
	for _, bad := range []string{"", "res-", "a", "b0", "dat"} {
		if IsPrefix(bad) {
			t.Errorf("IsPrefix(%q) 应为 false", bad)
		}
	}
	if got := strings.Join(Families(), ","); got != "data,b,h" {
		t.Errorf("Families = %q", got)
	}
}
