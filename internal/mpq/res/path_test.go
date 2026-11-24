package res

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/mpq"
)

// 路径式访问入口：数据根之外的路径与 os.* 完全一致；数据根之内走合并
// 视图，结果与"同一棵树以散文件存在"时 os.* 的结果逐项相同。

// pathTree 是一棵有代表性的测试树：多级目录、大小写混合的名字。
var pathTree = map[string]string{
	"ini/Scene/born03/npc_creator.xml": "<npc/>",
	"ini/Scene/School01/Readme.txt":    "school",
	"ini/a.txt":                        "a",
	"Zeta.txt":                         "z",
	"lua/task/task19.lua":              "-- task",
	"mpq/keep.bin":                     "fixture",
}

func writeTree(t *testing.T, root string, tree map[string]string) {
	t.Helper()
	for rel, content := range tree {
		putFile(t, filepath.Join(root, filepath.FromSlash(rel)), content)
	}
}

// resetCache 清掉路径式入口的进程内缓存；清理函数注册在 TempDir 之后，
// 先于它执行，释放层文件句柄（Windows 上占用着 TempDir 删不掉）。
func resetCache(t *testing.T) {
	t.Helper()
	ResetPathCache()
	t.Cleanup(ResetPathCache)
}

// packedPair 铺两棵相同的树：loose 保持散文件；packed 打成 b001 并修剪掉
// 散文件（mpq/ 被排除，保持散文件）。
func packedPair(t *testing.T) (loose, packed string) {
	t.Helper()
	base := t.TempDir()
	loose, packed = filepath.Join(base, "loose"), filepath.Join(base, "packed")
	writeTree(t, loose, pathTree)
	writeTree(t, packed, pathTree)
	mustBuild(t, packed, BuildOptions{Prefix: PrefixTest, Prune: true, Exclude: []string{"mpq"}})
	resetCache(t)
	return loose, packed
}

type statView struct {
	name string
	size int64
	dir  bool
}

func viewStat(info fs.FileInfo) statView {
	return statView{name: info.Name(), size: info.Size(), dir: info.IsDir()}
}

type walkFunc = func(root string, fn fs.WalkDirFunc) error

// walkView 记录遍历到达的相对路径（目录带尾斜杠、保持顺序）。
func walkView(root string, walk walkFunc, make func(string) fs.WalkDirFunc) (out []string, err error) {
	var fn fs.WalkDirFunc
	if make == nil {
		fn = func(string, fs.DirEntry, error) error { return nil }
	} else {
		fn = make(root)
	}
	err = walk(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return relErr
		}
		if d.IsDir() {
			rel += "/"
		}
		out = append(out, rel)
		return fn(p, d, err)
	})
	return out, err
}

// skipLua 跳过 lua 目录，用来比对 SkipDir 语义。
func skipLua(string) fs.WalkDirFunc {
	return func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "lua" {
			return fs.SkipDir
		}
		return nil
	}
}

// compareWithOS 对一组文件 / 目录 / 模式逐项比对资源仓入口与直接文件系统。
func compareWithOS(t *testing.T, loose, packed string, files, dirs, globs []string) {
	t.Helper()
	join := func(root, rel string) string {
		if rel == "" {
			return root
		}
		return filepath.Join(root, filepath.FromSlash(rel))
	}

	for _, rel := range files {
		lp, pp := join(loose, rel), join(packed, rel)

		wantData, wantErr := os.ReadFile(lp)
		gotData, gotErr := ReadFile(pp)
		if (wantErr == nil) != (gotErr == nil) || !bytes.Equal(wantData, gotData) {
			t.Errorf("ReadFile(%q) = %q（err=%v），os = %q（err=%v）", rel, gotData, gotErr, wantData, wantErr)
		}
		if gotErr != nil && !errors.Is(gotErr, fs.ErrNotExist) {
			t.Errorf("ReadFile(%q) 的错误应满足 ErrNotExist：%v", rel, gotErr)
		}

		wantInfo, wantErr := os.Stat(lp)
		gotInfo, gotErr := Stat(pp)
		if (wantErr == nil) != (gotErr == nil) {
			t.Errorf("Stat(%q) err=%v，os err=%v", rel, gotErr, wantErr)
		} else if wantErr == nil && viewStat(wantInfo) != viewStat(gotInfo) {
			t.Errorf("Stat(%q) = %+v，os = %+v", rel, viewStat(gotInfo), viewStat(wantInfo))
		}
		if wantErr != nil && (!errors.Is(gotErr, fs.ErrNotExist) || !os.IsNotExist(gotErr)) {
			t.Errorf("Stat(%q) 缺失时应满足 ErrNotExist：%v", rel, gotErr)
		}

		gotInfo, gotErr = Lstat(pp)
		if (wantErr == nil) != (gotErr == nil) || (wantErr == nil && viewStat(gotInfo) != viewStat(wantInfo)) {
			t.Errorf("Lstat(%q) = %+v（err=%v），os.Stat = %+v（err=%v）", rel, gotInfo, gotErr, wantInfo, wantErr)
		}
	}

	for _, rel := range dirs {
		want, wantErr := os.ReadDir(join(loose, rel))
		got, gotErr := ReadDir(join(packed, rel))
		if (wantErr == nil) != (gotErr == nil) {
			t.Errorf("ReadDir(%q) err=%v，os err=%v", rel, gotErr, wantErr)
			continue
		}
		if wantErr != nil {
			continue
		}
		var wantNames, gotNames []string
		for _, e := range want {
			wantNames = append(wantNames, e.Name()+map[bool]string{true: "/", false: ""}[e.IsDir()])
		}
		for _, e := range got {
			if info, err := e.Info(); err != nil || info.IsDir() != e.IsDir() {
				t.Errorf("ReadDir(%q) 的 %v：Info() 与 IsDir() 不一致（err=%v）", rel, e.Name(), err)
			}
			gotNames = append(gotNames, e.Name()+map[bool]string{true: "/", false: ""}[e.IsDir()])
		}
		if strings.Join(wantNames, ",") != strings.Join(gotNames, ",") {
			t.Errorf("ReadDir(%q) = %v，os = %v", rel, gotNames, wantNames)
		}
	}

	for _, make := range []func(string) fs.WalkDirFunc{nil, skipLua} {
		wantWalk, wantErr := walkView(loose, filepath.WalkDir, make)
		gotWalk, gotErr := walkView(packed, WalkDir, make)
		if strings.Join(wantWalk, "|") != strings.Join(gotWalk, "|") || (wantErr == nil) != (gotErr == nil) {
			t.Errorf("Walk = %v（err=%v），filepath.WalkDir = %v（err=%v）",
				gotWalk, gotErr, wantWalk, wantErr)
		}
	}

	for _, pattern := range globs {
		want, wantErr := filepath.Glob(filepath.Join(loose, filepath.FromSlash(pattern)))
		got, gotErr := Glob(filepath.Join(packed, filepath.FromSlash(pattern)))
		for i := range want {
			want[i], _ = filepath.Rel(loose, want[i])
		}
		for i := range got {
			got[i], _ = filepath.Rel(packed, got[i])
		}
		if strings.Join(want, "|") != strings.Join(got, "|") || (wantErr == nil) != (gotErr == nil) {
			t.Errorf("Glob(%q) = %v（err=%v），filepath.Glob = %v（err=%v）",
				pattern, got, gotErr, want, wantErr)
		}
	}
}

func TestPathEntriesMatchDirectFilesystem(t *testing.T) {
	loose, packed := packedPair(t)
	compareWithOS(t, loose, packed,
		[]string{
			"ini/Scene/born03/npc_creator.xml",
			"ini/Scene/School01/Readme.txt",
			"ini/a.txt",
			"Zeta.txt",
			"lua/task/task19.lua",
			"mpq/keep.bin",
			"ini/missing.txt",
			"missing",
			"ini/Scene/missing.xml",
			"ini/a.txt/under", // 文件之下的路径
		},
		[]string{
			"", "ini", "ini/Scene", "ini/Scene/born03", "ini/Scene/School01",
			"lua", "mpq", "missing", "ini/missing", "ini/a.txt",
		},
		[]string{
			"*.txt", "ini/*.txt", "ini/Scene/*", "ini/Scene/*/npc_creator.xml",
			"ini/**/*.xml", "lua/task/*.lua", "ini/*/*.xml", "*.xml", "mpq/*",
			"Zeta.txt", "ini/[ab]*.txt", "ini/Scene/*/npc_*.xml",
		})
}

func TestPathWalkWithFileAndMissingRoots(t *testing.T) {
	loose, packed := packedPair(t)
	for _, rel := range []string{"ini/a.txt", "ini/missing.txt"} {
		want, wantErr := walkView(filepath.Join(loose, rel), filepath.WalkDir, nil)
		got, gotErr := walkView(filepath.Join(packed, rel), WalkDir, nil)
		// 文件根：回调一次；缺失根：回调一次并带回 lstat 错误，返回值仍是 nil。
		if strings.Join(want, "|") != strings.Join(got, "|") || (wantErr == nil) != (gotErr == nil) {
			t.Errorf("WalkDir(%q) = %v（err=%v），filepath.WalkDir = %v（err=%v）",
				rel, got, gotErr, want, wantErr)
		}
	}
}

// 根条目的名字取主机路径的末段，与 filepath.WalkDir 一致。
func TestPathWalkRootEntryName(t *testing.T) {
	_, packed := packedPair(t)
	var names []string
	WalkDir(filepath.Join(packed, "ini")+"/", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Clean(p) == filepath.Join(packed, "ini") {
			names = append(names, d.Name())
		}
		return nil
	})
	if strings.Join(names, ",") != "ini" {
		t.Fatalf("根条目名字 = %v，应为 ini", names)
	}
}

func TestPathRelativePaths(t *testing.T) {
	_, packed := packedPair(t)
	t.Chdir(filepath.Dir(packed))
	if data, err := ReadFile(filepath.Join("packed", "ini", "a.txt")); err != nil || string(data) != "a" {
		t.Fatalf("相对路径读取 = %q（err=%v）", data, err)
	}
	if entries, err := ReadDir(filepath.Join("packed", "ini", "Scene")); err != nil || len(entries) != 2 {
		t.Fatalf("相对路径 ReadDir = %v（err=%v）", entries, err)
	}
	if data, err := ReadFile(filepath.Join("loose", "ini", "a.txt")); err != nil || string(data) != "a" {
		t.Fatalf("同级散文件树读取 = %q（err=%v）", data, err)
	}
}

func TestPathOutsideDataRootIsPlainOS(t *testing.T) {
	resetCache(t)
	dir := t.TempDir()
	putFile(t, filepath.Join(dir, "a.txt"), "plain")
	if data, err := ReadFile(filepath.Join(dir, "a.txt")); err != nil || string(data) != "plain" {
		t.Fatalf("ReadFile = %q（err=%v）", data, err)
	}
	if _, err := Stat(filepath.Join(dir, "missing")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Stat 缺失文件 = %v", err)
	}
	if entries, err := ReadDir(filepath.Join(dir, "a.txt")); err == nil {
		t.Fatalf("对文件 ReadDir 应报错，实得 %v", entries)
	}
}

// 散文件实时生效：改完即见；新落盘的同名散文件立即盖住层内容；删掉即回到层。
func TestPathLooseOverrideIsLive(t *testing.T) {
	_, packed := packedPair(t)
	a := filepath.Join(packed, "ini", "a.txt")
	added := filepath.Join(packed, "ini", "added.txt")

	if data, err := ReadFile(a); err != nil || string(data) != "a" {
		t.Fatalf("层内容 = %q（err=%v）", data, err)
	}
	if layer := layerOf(t, a); layer != "b001" {
		t.Fatalf("a.txt 应来自 b001，实得 %q", layer)
	}

	putFile(t, a, "loose a")
	if data, err := ReadFile(a); err != nil || string(data) != "loose a" {
		t.Fatalf("散文件应立即生效：%q（err=%v）", data, err)
	}
	if info, err := Stat(a); err != nil || info.Size() != 7 {
		t.Fatalf("Stat 应反映散文件：%+v（err=%v）", info, err)
	}

	putFile(t, added, "n")
	if names := readDirNames(t, filepath.Join(packed, "ini")); names != "Scene/,a.txt,added.txt" {
		t.Fatalf("新散文件应立即出现在枚举里：%s", names)
	}
	if data, err := ReadFile(added); err != nil || string(data) != "n" {
		t.Fatalf("新散文件应立即可读：%q（err=%v）", data, err)
	}
	if _, err := Stat(filepath.Join(packed, "ini", "NewDir")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("不存在的目录应报 ErrNotExist：%v", err)
	}
	if err := os.Mkdir(filepath.Join(packed, "ini", "NewDir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if info, err := Stat(filepath.Join(packed, "ini", "NewDir")); err != nil || !info.IsDir() {
		t.Fatalf("新散目录应立即可见：%+v（err=%v）", info, err)
	}

	for _, p := range []string{a, added} {
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
	}
	if data, err := ReadFile(a); err != nil || string(data) != "a" {
		t.Fatalf("散文件删掉后应回到层内容：%q（err=%v）", data, err)
	}
	if _, err := Stat(added); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("删掉的散文件不应再存在：%v", err)
	}
	if names := readDirNames(t, filepath.Join(packed, "ini")); names != "NewDir/,Scene/,a.txt" {
		t.Fatalf("删掉的散文件不应再出现在枚举里：%s", names)
	}
}

// ReadFile 返回的切片归调用方所有，改写不会污染读缓存。
func TestReadFileReturnsCallerOwnedCopy(t *testing.T) {
	_, packed := packedPair(t)
	a := filepath.Join(packed, "ini", "a.txt")
	data, err := ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	data[0] = 'X'
	if again, _ := ReadFile(a); string(again) != "a" {
		t.Fatalf("改写返回值污染了读缓存：%q", again)
	}
	if empty, err := ReadFile(filepath.Join(packed, "ini", "Scene", "born03", "npc_creator.xml")); err != nil || empty == nil {
		t.Fatalf("ReadFile = %v（err=%v）", empty, err)
	}
}

// layerOf 报告主机路径经路径式入口解析后命中的层。
func layerOf(t *testing.T, path string) string {
	t.Helper()
	m, name, err := resolve(path, false)
	if err != nil || m == nil {
		t.Fatalf("resolve(%s)：m=%v err=%v", path, m, err)
	}
	return m.store.Layer(name)
}

func readDirNames(t *testing.T, dir string) string {
	t.Helper()
	entries, err := ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%s)：%v", dir, err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name()+map[bool]string{true: "/", false: ""}[e.IsDir()])
	}
	return strings.Join(names, ",")
}

func TestBadDataRootsSurfaceErrors(t *testing.T) {
	resetCache(t)
	valid := []mpq.Addition{{Name: `a.txt`, Data: []byte("a")}}

	ambiguous := t.TempDir()
	writeArchive(t, filepath.Join(ambiguous, "b001.rep"), valid)
	writeArchive(t, filepath.Join(ambiguous, "data001.rep"), valid)
	for name, call := range map[string]func(string) error{
		"ReadFile": func(p string) error { _, err := ReadFile(p); return err },
		"Stat":     func(p string) error { _, err := Stat(p); return err },
		"ReadDir":  func(p string) error { _, err := ReadDir(p); return err },
		// WalkDir 把根的 lstat 错误交给回调（同 filepath.WalkDir）。
		"WalkDir": func(p string) error {
			var got error
			WalkDir(p, func(_ string, _ fs.DirEntry, err error) error { got = err; return nil })
			return got
		},
	} {
		if err := call(filepath.Join(ambiguous, "a.txt")); err == nil {
			t.Errorf("两族并存的数据根：%s 应报错", name)
		}
	}

	corrupt := t.TempDir()
	putFile(t, filepath.Join(corrupt, "b001.rep"), "not an archive")
	if _, err := ReadFile(filepath.Join(corrupt, "a.txt")); err == nil {
		t.Error("层文件损坏时应报错")
	}
}

func TestPathCacheTracksRebuiltLayers(t *testing.T) {
	base := t.TempDir()
	resetCache(t)
	root, src := filepath.Join(base, "root"), filepath.Join(base, "src")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	putFile(t, filepath.Join(src, "a.txt"), "v1")
	mustBuild(t, root, BuildOptions{Prefix: PrefixTest, Source: src})
	a := filepath.Join(root, "a.txt")

	if data, _ := ReadFile(a); string(data) != "v1" {
		t.Fatalf("首读 = %q", data)
	}
	// BuildLayer 自带缓存重置：重建后的层立即可见。
	putFile(t, filepath.Join(src, "a.txt"), "v2")
	mustBuild(t, root, BuildOptions{Prefix: PrefixTest, Number: 1, Replace: true, Source: src})
	if data, _ := ReadFile(a); string(data) != "v2" {
		t.Fatalf("重建后应读到 v2：%q", data)
	}
	// 绕过 BuildLayer 手工替换层文件：显式 ResetPathCache 之后才生效。
	writeArchive(t, filepath.Join(root, "b001.rep"), []mpq.Addition{{Name: `a.txt`, Data: []byte("v3")}})
	if data, _ := ReadFile(a); string(data) != "v2" {
		t.Fatalf("手工替换后、重置前不应读到新内容：%q", data)
	}
	ResetPathCache()
	if data, _ := ReadFile(a); string(data) != "v3" {
		t.Fatalf("重置后应读到 v3：%q", data)
	}
}

func TestLayerAppearingLaterIsFoundAfterScanTTL(t *testing.T) {
	late := filepath.Join(t.TempDir(), "late")
	resetCache(t)
	putFile(t, filepath.Join(late, "a.txt"), "loose")
	b := filepath.Join(late, "b.txt")
	if _, err := ReadFile(b); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("尚无层文件时应不存在：%v", err)
	}

	// 不经 BuildLayer 手工放入层：目录判定缓存过期后即被发现。
	writeArchive(t, filepath.Join(late, "b001.rep"), []mpq.Addition{{Name: `b.txt`, Data: []byte("layer")}})
	if _, err := ReadFile(b); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("判定缓存有效期内仍按散文件目录处理：%v", err)
	}
	time.Sleep(mountScanTTL + 200*time.Millisecond)
	if data, err := ReadFile(b); err != nil || string(data) != "layer" {
		t.Fatalf("缓存过期后应从层读到：%q（err=%v）", data, err)
	}
}

// mustReadPath 经路径式入口读取主机路径并核对内容。
func mustReadPath(t *testing.T, path, want string) {
	t.Helper()
	data, err := ReadFile(path)
	if err != nil || string(data) != want {
		t.Fatalf("ReadFile(%s) = %q（err=%v），应为 %q", path, data, err, want)
	}
}

// 仓内实际布局：测试数据根（b001）之下嵌着参考资源根（h001），两边有
// 同名文件。各自按离自己最近的数据根解析，互不串层。
func TestNestedReferenceRoot(t *testing.T) {
	base := t.TempDir()
	resetCache(t)
	outer := filepath.Join(base, "testdata")
	inner := filepath.Join(outer, "references")
	writeArchive(t, filepath.Join(outer, "b001.rep"), []mpq.Addition{
		{Name: `ini\Item\a.xml`, Data: []byte("nineyin")},
	})
	writeArchive(t, filepath.Join(inner, "h001.rep"), []mpq.Addition{
		{Name: `ini\Item\a.xml`, Data: []byte("heijin")},
	})

	mustReadPath(t, filepath.Join(outer, "ini", "Item", "a.xml"), "nineyin")
	mustReadPath(t, filepath.Join(inner, "ini", "Item", "a.xml"), "heijin")

	// 外层的层集合不含内层的 h001；内层的层文件也不会被当成外层的散文件。
	store, err := Open(outer)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if layers := store.Layers(); len(layers) != 1 || layers[0].Name != "b001" {
		t.Fatalf("外层数据根应只有 b001：%+v", layers)
	}
	files, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.Layer == "local" {
			t.Fatalf("内层层文件不应出现在外层的散文件层：%+v", f)
		}
	}
}
