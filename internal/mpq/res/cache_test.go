package res

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/mpq"
)

// openCachedFixtureIn 在 dir 下铺 b001（三个文件）+ b002（覆盖 a.txt）
// 并以缓存打开。
func openCachedFixtureIn(t *testing.T, dir string) *CachedStore {
	t.Helper()
	writeArchive(t, filepath.Join(dir, "b001.rep"), []mpq.Addition{
		{Name: `a.txt`, Data: []byte("b001 a")},
		{Name: `Sub\b.txt`, Data: []byte("b001 b")},
		{Name: `Sub\Deep\c.txt`, Data: []byte("b001 c")},
	}, mpq.V4)
	writeArchive(t, filepath.Join(dir, "b002.rep"), []mpq.Addition{
		{Name: `a.txt`, Data: []byte("b002 a")},
	}, mpq.V4)
	cached, err := OpenCachedWith(dir, CacheOptions{MaxBytes: 1 << 20, MaxParsed: 8}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cached.Close)
	return cached
}

func openCachedFixture(t *testing.T) *CachedStore {
	t.Helper()
	return openCachedFixtureIn(t, t.TempDir())
}

func mustReadCached(t *testing.T, c *CachedStore, name, want string) {
	t.Helper()
	data, err := c.Read(name)
	if err != nil || string(data) != want {
		t.Fatalf("Read(%q) = %q（err=%v），应为 %q", name, data, err, want)
	}
}

// dirNames 把目录子项拼成 "a.txt,Sub/" 形式（顺序即返回顺序）。
func dirNames(t *testing.T, fsys *StoreFS, dir string) string {
	t.Helper()
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q)：%v", dir, err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name()+map[bool]string{true: "/", false: ""}[e.IsDir()])
	}
	return strings.Join(names, ",")
}

func TestCacheByteHitAndLayerOverride(t *testing.T) {
	cached := openCachedFixture(t)

	// 首读：miss + 缓存；b002 覆盖 b001。
	mustReadCached(t, cached, "a.txt", "b002 a")
	if s := cached.Stats(); s.ByteMisses != 1 || s.CachedFiles != 1 {
		t.Fatalf("首读后统计不符：%+v", s)
	}
	// 二读：命中缓存。
	mustReadCached(t, cached, `A.TXT`, "b002 a")
	if s := cached.Stats(); s.ByteHits != 1 {
		t.Fatalf("应有一次命中：%+v", s)
	}
}

func TestCacheLoadParsed(t *testing.T) {
	cached := openCachedFixture(t)

	parse := func(data []byte) (int, error) { return len(data), nil }
	first, err := Load(cached, "Sub/b.txt", parse)
	if err != nil || first != 6 {
		t.Fatalf("Load = %d（err=%v）", first, err)
	}
	// 同类型同名字：直接命中解析缓存（parse 不再执行）。
	calls := 0
	counting := func(data []byte) (int, error) { calls++; return len(data), nil }
	second, err := Load(cached, `sub\B.TXT`, counting)
	if err != nil || second != 6 {
		t.Fatalf("二次 Load = %d（err=%v）", second, err)
	}
	if calls != 0 {
		t.Fatalf("解析缓存未命中，parse 执行了 %d 次", calls)
	}
	if s := cached.Stats(); s.ParsedHits != 1 || s.ParsedKeys != 1 {
		t.Fatalf("解析缓存统计不符：%+v", s)
	}
	// 不同解析类型：各自缓存。
	upper, err := Load(cached, "Sub/b.txt", func(data []byte) (string, error) {
		return strings.ToUpper(string(data)), nil
	})
	if err != nil || upper != "B001 B" {
		t.Fatalf("另一类型 Load = %q（err=%v）", upper, err)
	}
}

func TestCacheReloadInvalidates(t *testing.T) {
	dir := t.TempDir()
	writeArchive(t, filepath.Join(dir, "b001.rep"), []mpq.Addition{
		{Name: `a.txt`, Data: []byte("v1")},
	})
	cached, err := OpenCached(dir, CacheOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer cached.Close()
	if _, err := cached.Read("a.txt"); err != nil {
		t.Fatal(err)
	}
	writeArchive(t, filepath.Join(dir, "b001.rep"), []mpq.Addition{
		{Name: `a.txt`, Data: []byte("v2")},
	})
	if err := cached.Reload(); err != nil {
		t.Fatal(err)
	}
	mustReadCached(t, cached, "a.txt", "v2")
	// reload 清空后重读：又一次 miss，并重新装填缓存。
	if s := cached.Stats(); s.ByteMisses != 2 || s.ByteHits != 0 || s.CachedFiles != 1 {
		t.Fatalf("reload 后应重新装填：%+v", s)
	}
}

func TestCacheConcurrentSingleFlight(t *testing.T) {
	cached := openCachedFixture(t)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := cached.Read("Sub/Deep/c.txt")
			if err != nil || string(data) != "b001 c" {
				t.Errorf("并发读 = %q（err=%v）", data, err)
			}
		}()
	}
	wg.Wait()
	// 16 个读者：1 次 miss（发起者），其余命中或等待同一次读取。
	if s := cached.Stats(); s.ByteMisses != 1 {
		t.Fatalf("并发同键应只 miss 一次：%+v", s)
	}
}

func TestCacheEvictionByMaxBytes(t *testing.T) {
	dir := t.TempDir()
	writeArchive(t, filepath.Join(dir, "b001.rep"), []mpq.Addition{
		{Name: `big1.txt`, Data: bytesOf(600)},
		{Name: `big2.txt`, Data: bytesOf(600)},
		{Name: `big3.txt`, Data: bytesOf(10)},
	})
	cached, err := OpenCached(dir, CacheOptions{MaxBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	defer cached.Close()
	for _, name := range []string{"big1.txt", "big2.txt", "big3.txt"} {
		if _, err := cached.Read(name); err != nil {
			t.Fatal(err)
		}
	}
	s := cached.Stats()
	if s.CachedBytes > 1024 {
		t.Fatalf("缓存总量超限：%d", s.CachedBytes)
	}
	if s.Evictions == 0 || s.CachedFiles == 0 {
		t.Fatalf("应有逐出且仍有缓存：%+v", s)
	}
}

// 散文件不进缓存：改完即生效；新落盘的同名散文件立即盖住已缓存的层内容。
func TestLooseFilesBypassCache(t *testing.T) {
	dir := t.TempDir()
	putFile(t, filepath.Join(dir, "loose.txt"), "l1")
	cached := openCachedFixtureIn(t, dir)

	mustReadCached(t, cached, "loose.txt", "l1")
	if s := cached.Stats(); s.LocalReads != 1 || s.CachedFiles != 0 || s.ByteMisses != 0 {
		t.Fatalf("散文件直读不应触碰缓存：%+v", s)
	}
	putFile(t, filepath.Join(dir, "loose.txt"), "l2")
	mustReadCached(t, cached, "loose.txt", "l2")

	mustReadCached(t, cached, "a.txt", "b002 a")
	putFile(t, filepath.Join(dir, "a.txt"), "loose a")
	mustReadCached(t, cached, "a.txt", "loose a")
	if err := os.Remove(filepath.Join(dir, "a.txt")); err != nil {
		t.Fatal(err)
	}
	mustReadCached(t, cached, "a.txt", "b002 a")

	// Load 对散文件每次都重新解析，不进解析缓存。
	calls := 0
	counting := func(data []byte) (int, error) { calls++; return len(data), nil }
	for _, want := range []int{2, 4} {
		n, err := Load(cached, "loose.txt", counting)
		if err != nil || n != want {
			t.Fatalf("Load = %d（err=%v），应为 %d", n, err, want)
		}
		putFile(t, filepath.Join(dir, "loose.txt"), strings.Repeat("x", want+2))
	}
	if calls != 2 || cached.Stats().ParsedKeys != 0 {
		t.Fatalf("散文件不应进解析缓存：calls=%d %+v", calls, cached.Stats())
	}
}

func bytesOf(n int) []byte { return make([]byte, n) }

func TestStoreFSBasics(t *testing.T) {
	cached := openCachedFixture(t)
	fsys := cached.FS()

	if _, err := fs.ReadFile(fsys, "a.txt"); err != nil {
		t.Fatalf("ReadFile 失败：%v", err)
	}
	if _, err := fs.ReadFile(fsys, "missing.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("缺失文件应报 ErrNotExist：%v", err)
	}
	info, err := fs.Stat(fsys, "Sub")
	if err != nil || !info.IsDir() || !info.Mode().IsDir() {
		t.Fatalf("Stat(Sub) 应为目录（info=%+v err=%v）", info, err)
	}
	if _, err := fs.Stat(fsys, "Sub/Deep"); err != nil {
		t.Fatalf("嵌套目录 Stat 失败：%v", err)
	}
	// 只查索引不读内容；名字保留归档内的原样大小写。
	if info, err := fs.Stat(fsys, "sub/DEEP/c.TXT"); err != nil || info.Name() != "c.txt" || info.Size() != 6 {
		t.Fatalf("Stat(c.txt) = %+v（err=%v）", info, err)
	}
	if dirNames(t, fsys, "sub") != "Deep/,b.txt" {
		t.Fatalf("子目录名应保留原样大小写：%s", dirNames(t, fsys, "sub"))
	}

	// 排序同 os.ReadDir（按名字字节序，不折叠大小写）。
	if got := dirNames(t, fsys, "."); got != "Sub/,a.txt" {
		t.Fatalf("根目录子项 = %s", got)
	}
	// 对文件做目录枚举报错，同 os.ReadDir。
	if _, err := fsys.ReadDir("a.txt"); err == nil {
		t.Fatal("对文件 ReadDir 应报错")
	}

	// fs.WalkDir 全量遍历：三个文件 + 中间目录，词法序。
	var walked []string
	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		walked = append(walked, p+map[bool]string{true: "/", false: ""}[d.IsDir()])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "./,Sub/,Sub/Deep/,Sub/Deep/c.txt,Sub/b.txt,a.txt"
	if strings.Join(walked, ",") != want {
		t.Fatalf("Walk 序不符：\n got %v\nwant %v", walked, want)
	}
}

// 散文件 / 散目录在枚举与 Stat 时实时合并，不必 Reload。
func TestStoreFSLiveLooseMerge(t *testing.T) {
	cached := openCachedFixture(t)
	fsys := cached.FS()
	dir := cached.Dir()

	if got := dirNames(t, fsys, "Sub"); got != "Deep/,b.txt" {
		t.Fatalf("初始子项 = %s", got)
	}
	putFile(t, filepath.Join(dir, "Sub", "new.txt"), "n")
	if got := dirNames(t, fsys, "Sub"); got != "Deep/,b.txt,new.txt" {
		t.Fatalf("新落盘的散文件应立即可见：%s", got)
	}
	if got := dirNames(t, fsys, "."); got != "Sub/,a.txt" {
		t.Fatalf("根目录子项 = %s", got)
	}
	putFile(t, filepath.Join(dir, "a.txt"), "loose a")
	if info, err := fs.Stat(fsys, "a.txt"); err != nil || info.Size() != 7 {
		t.Fatalf("散文件应盖住层内同名文件（info=%+v err=%v）", info, err)
	}
	if data, err := fs.ReadFile(fsys, "a.txt"); err != nil || string(data) != "loose a" {
		t.Fatalf("ReadFile = %q（err=%v）", data, err)
	}

	if err := os.Mkdir(filepath.Join(dir, "EmptyDir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := dirNames(t, fsys, "."); got != "EmptyDir/,Sub/,a.txt" {
		t.Fatalf("散目录应出现在枚举里：%s", got)
	}
	// 层容器不是资源，不出现在枚举里。
	if got := dirNames(t, fsys, "."); strings.Contains(got, ".rep") {
		t.Fatalf("层容器不应被枚举：%s", got)
	}
}

func TestStoreFSIndexRefreshAfterReload(t *testing.T) {
	dir := t.TempDir()
	writeArchive(t, filepath.Join(dir, "b001.rep"), []mpq.Addition{
		{Name: `a.txt`, Data: []byte("v1")},
	})
	cached, err := OpenCached(dir, CacheOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer cached.Close()
	fsys := cached.FS()
	if _, err := fs.Stat(fsys, "b.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("b.txt 初始应不存在：%v", err)
	}
	writeArchive(t, filepath.Join(dir, "b001.rep"), []mpq.Addition{
		{Name: `a.txt`, Data: []byte("v1")},
		{Name: `b.txt`, Data: []byte("v2")},
	})
	if err := cached.Reload(); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(fsys, "b.txt"); err != nil {
		t.Fatalf("reload 后索引应包含 b.txt：%v", err)
	}
}
