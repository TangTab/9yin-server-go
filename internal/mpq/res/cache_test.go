package res

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/local/9yin-go-server/internal/mpq"
)

func openCachedFixture(t *testing.T) *CachedStore {
	t.Helper()
	dir := t.TempDir()
	writeArchive(t, filepath.Join(dir, "res-base.rep"), []mpq.Addition{
		{Name: `resources\a.txt`, Data: []byte("base a")},
		{Name: `resources\sub\b.txt`, Data: []byte("base b")},
		{Name: `resources\sub\deep\c.txt`, Data: []byte("base c")},
	})
	writeArchive(t, filepath.Join(dir, "res-patch.rep"), []mpq.Addition{
		{Name: `resources\a.txt`, Data: []byte("patch a")},
	})
	if err := os.WriteFile(filepath.Join(dir, "res-manifest.json"), []byte(`{
	  "base":  { "file": "res-base.rep",  "version": "0001" },
	  "patch": { "file": "res-patch.rep", "version": "0001" },
	  "local": "resources"
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := OpenCached(dir, CacheOptions{MaxBytes: 1 << 20, MaxParsed: 8})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	return store
}

func TestCacheByteHitAndLayerOverride(t *testing.T) {
	cached := openCachedFixture(t)

	// 首读：miss + 缓存；patch 覆盖 base。
	data, err := cached.Read("resources/a.txt")
	if err != nil || string(data) != "patch a" {
		t.Fatalf("Read = %q（err=%v）", data, err)
	}
	if s := cached.Stats(); s.ByteMisses != 1 || s.CachedFiles != 1 {
		t.Fatalf("首读后统计不符：%+v", s)
	}
	// 二读：命中缓存。
	data, err = cached.Read("resources\\A.TXT")
	if err != nil || string(data) != "patch a" {
		t.Fatalf("缓存读 = %q（err=%v）", data, err)
	}
	if s := cached.Stats(); s.ByteHits != 1 {
		t.Fatalf("应有一次命中：%+v", s)
	}
}

func TestCacheLoadParsed(t *testing.T) {
	cached := openCachedFixture(t)

	parse := func(data []byte) (int, error) { return len(data), nil }
	first, err := Load(cached, "resources/sub/b.txt", parse)
	if err != nil || first != 6 {
		t.Fatalf("Load = %d（err=%v）", first, err)
	}
	// 同类型同名字：直接命中解析缓存（parse 不再执行）。
	calls := 0
	counting := func(data []byte) (int, error) { calls++; return len(data), nil }
	second, err := Load(cached, `resources\SUB\b.txt`, counting)
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
	upper, err := Load(cached, "resources/sub/b.txt", func(data []byte) (string, error) {
		return strings.ToUpper(string(data)), nil
	})
	if err != nil || upper != "BASE B" {
		t.Fatalf("另一类型 Load = %q（err=%v）", upper, err)
	}
}

func TestCacheReloadInvalidates(t *testing.T) {
	dir := t.TempDir()
	writeArchive(t, filepath.Join(dir, "res-base.rep"), []mpq.Addition{
		{Name: `resources\a.txt`, Data: []byte("v1")},
	})
	cached, err := OpenCached(dir, CacheOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer cached.Close()
	if _, err := cached.Read("resources/a.txt"); err != nil {
		t.Fatal(err)
	}
	writeArchive(t, filepath.Join(dir, "res-base.rep"), []mpq.Addition{
		{Name: `resources\a.txt`, Data: []byte("v2")},
	})
	if err := cached.Reload(); err != nil {
		t.Fatal(err)
	}
	data, err := cached.Read("resources/a.txt")
	if err != nil || string(data) != "v2" {
		t.Fatalf("reload 后读到旧缓存：%q（err=%v）", data, err)
	}
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
			data, err := cached.Read("resources/sub/deep/c.txt")
			if err != nil || string(data) != "base c" {
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
	writeArchive(t, filepath.Join(dir, "res-base.rep"), []mpq.Addition{
		{Name: `resources\big1.txt`, Data: make([]byte, 600)},
		{Name: `resources\big2.txt`, Data: make([]byte, 600)},
		{Name: `resources\big3.txt`, Data: make([]byte, 10)},
	})
	cached, err := OpenCached(dir, CacheOptions{MaxBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	defer cached.Close()
	for _, name := range []string{"resources/big1.txt", "resources/big2.txt", "resources/big3.txt"} {
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

func TestStoreFSBasics(t *testing.T) {
	cached := openCachedFixture(t)
	fsys := cached.FS()

	if _, err := fs.ReadFile(fsys, "resources/a.txt"); err != nil {
		t.Fatalf("ReadFile 失败：%v", err)
	}
	if _, err := fs.ReadFile(fsys, "resources/missing.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("缺失文件应报 ErrNotExist：%v", err)
	}
	info, err := fs.Stat(fsys, "resources/sub")
	if err != nil || !info.IsDir() {
		t.Fatalf("Stat(sub) 应为目录（err=%v）", err)
	}
	if _, err := fs.Stat(fsys, "resources/sub/deep"); err != nil {
		t.Fatalf("嵌套目录 Stat 失败：%v", err)
	}

	entries, err := fs.ReadDir(fsys, "resources")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name()+(map[bool]string{true: "/", false: ""})[e.IsDir()])
	}
	if strings.Join(names, ",") != "a.txt,sub/" {
		t.Fatalf("resources 子项 = %v", names)
	}

	// fs.WalkDir 全量遍历：三个文件 + 中间目录。
	var walked []string
	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		walked = append(walked, p)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".", "resources", "resources/a.txt", "resources/sub", "resources/sub/b.txt", "resources/sub/deep", "resources/sub/deep/c.txt"}
	if strings.Join(walked, "|") != strings.Join(want, "|") {
		t.Fatalf("Walk 序不符：\n got %v\nwant %v", walked, want)
	}
}

func TestStoreFSIndexRefreshAfterReload(t *testing.T) {
	dir := t.TempDir()
	writeArchive(t, filepath.Join(dir, "res-base.rep"), []mpq.Addition{
		{Name: `resources\a.txt`, Data: []byte("v1")},
	})
	cached, err := OpenCached(dir, CacheOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer cached.Close()
	fsys := cached.FS()
	if _, err := fs.Stat(fsys, "resources/b.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("b.txt 初始应不存在：%v", err)
	}
	writeArchive(t, filepath.Join(dir, "res-base.rep"), []mpq.Addition{
		{Name: `resources\a.txt`, Data: []byte("v1")},
		{Name: `resources\b.txt`, Data: []byte("v2")},
	})
	if err := cached.Reload(); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(fsys, "resources/b.txt"); err != nil {
		t.Fatalf("reload 后索引应包含 b.txt：%v", err)
	}
}
