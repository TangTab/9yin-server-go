package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/local/9yin-go-server/internal/mpq/res"
	"strings"
	"testing"
)

// 资源名映射：资源根下命中，根外与穿越路径回退 os。
func TestResourceStoreNameMapping(t *testing.T) {
	// store 只需非 nil（resourceStoreName 只做路径判定）。
	access := &resourceAccessor{root: t.TempDir(), store: &res.CachedStore{}}
	if _, ok := access.resourceStoreName(filepath.Join(access.root, "resources", "a.ini")); !ok {
		t.Fatal("资源根下应命中资源仓")
	}
	if _, ok := access.resourceStoreName(filepath.Join(access.root, "data", "a.json")); ok {
		t.Fatal("data/ 不是资源，应回退 os")
	}
	if _, ok := access.resourceStoreName(filepath.Join(access.root, "..", "elsewhere", "resources", "a.ini")); ok {
		t.Fatal("越界路径不应命中资源仓")
	}
	// 相对路径按工作目录解析，这里只需保证不误报。
	if _, ok := access.resourceStoreName("resources/a.ini"); ok {
		t.Log("工作目录恰在资源根下，视为命中（合符预期）")
	}
}

// 资源仓不可用时（store=nil）：全部回退 os 语义。
func TestResourceFallbackWithoutStore(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("plain"), 0o644); err != nil {
		t.Fatal(err)
	}
	// openResourceReader / readResourceBytes 走全局单例；测试里
	// 用默认访问器（真实环境可能带仓），对临时目录路径必然回退 os。
	data, err := readResourceBytes(filepath.Join(dir, "a.txt"))
	if err != nil || string(data) != "plain" {
		t.Fatalf("回退读 = %q（err=%v）", data, err)
	}
	reader, err := openResourceReader(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	reader.Close()
	if _, err := statResourcePath(filepath.Join(dir, "a.txt")); err != nil {
		t.Fatalf("回退 stat 失败：%v", err)
	}
	if _, err := statResourcePath(filepath.Join(dir, "missing.txt")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("缺失文件应报 ErrNotExist：%v", err)
	}
	entries, err := readResourceDirEntries(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "a.txt" {
		t.Fatalf("回退 ReadDir = %v（err=%v）", entries, err)
	}
}

// 真实仓库数据：本地缺失的文件应由 base MPQ 层提供，且二读命中缓存。
// 缺 res-base.rep（精简环境）时跳过。
func TestResourceReadServesBaseLayerWithCache(t *testing.T) {
	access := resourceAccessorOf()
	if access.store == nil {
		t.Skip("资源仓不可用（缺 res-base.rep），跳过 MPQ 层验证")
	}
	// 该文件只在 base 归档里，本地 resources/ 未解包（若未来本地
	// 补齐则改用 store.Layer 挑一个 base 文件）。
	name := "resources/modern/map/ter/clone015/terrain.ini"
	if layer := access.store.Layer(name); layer == "" {
		t.Skipf("仓库中无 %s，跳过", name)
	}
	before := access.store.Stats()
	data, err := readResourceBytes(filepath.Join(access.root, filepath.FromSlash(name)))
	if err != nil {
		t.Fatalf("经资源仓读取失败：%v", err)
	}
	if len(data) == 0 {
		t.Fatal("base 层文件内容为空")
	}
	if !strings.Contains(string(data[:min(64, len(data))]), "[") {
		t.Logf("预览非节头（可能为二进制），长度=%d", len(data))
	}
	after := access.store.Stats()
	if after.ByteMisses != before.ByteMisses+1 {
		t.Fatalf("首读应记一次 miss：%+v → %+v", before, after)
	}
	if _, err := readResourceBytes(filepath.Join(access.root, filepath.FromSlash(name))); err != nil {
		t.Fatal(err)
	}
	if got := access.store.Stats().ByteHits; got != after.ByteHits+1 {
		t.Fatalf("二读应命中缓存：hits=%d", got)
	}
}
