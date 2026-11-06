package res

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/local/9yin-go-server/internal/mpq"
)

// 三层资源仓：本地 > patch > base，墓碑三层皆 miss。

// createArchive 用 mpq 写入侧构建一个测试归档。
func createArchive(t *testing.T, files []mpq.Addition) []byte {
	t.Helper()
	base, err := mpq.Create(mpq.DefaultCreateOptions())
	if err != nil {
		t.Fatal(err)
	}
	result, err := mpq.Modify(base, nil, files)
	if err != nil {
		t.Fatal(err)
	}
	return result.Bytes
}

func writeArchive(t *testing.T, path string, files []mpq.Addition) {
	t.Helper()
	if err := os.WriteFile(path, createArchive(t, files), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFileOrFatal(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestThreeLayerOverrideAndTombstone(t *testing.T) {
	dir := t.TempDir()

	// base：a.txt / sub\b.txt / gone.txt
	writeArchive(t, filepath.Join(dir, "res-base.rep"), []mpq.Addition{
		{Name: `resources\a.txt`, Data: []byte("base a")},
		{Name: `resources\sub\b.txt`, Data: []byte("base b")},
		{Name: `resources\gone.txt`, Data: []byte("to be deleted")},
	})
	// patch：覆盖 a.txt + 新增 c.txt
	writeArchive(t, filepath.Join(dir, "res-patch.rep"), []mpq.Addition{
		{Name: `resources\a.txt`, Data: []byte("patch a")},
		{Name: `resources\c.txt`, Data: []byte("patch c")},
	})
	// 本地散文件：覆盖 sub/b.txt
	readFileOrFatal(t, filepath.Join(dir, "resources", "sub", "b.txt"), "local b")
	// 清单：墓碑 gone.txt
	manifest := `{
  "base":  { "file": "res-base.rep",  "version": "0001" },
  "patch": { "file": "res-patch.rep", "version": "0003" },
  "deleted": ["resources\\gone.txt"]
}`
	if err := os.WriteFile(filepath.Join(dir, "res-manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// 各层命中。
	for name, want := range map[string]string{
		`resources\sub\b.txt`: "local",
		`resources\a.txt`:     "patch",
		`resources\gone.txt`:  "", // 墓碑：即便 base 有也 miss
		`resources\c.txt`:     "patch",
		`resources\nope.txt`:  "",
	} {
		if got := store.Layer(name); got != want {
			t.Fatalf("Layer(%q) = %q，应为 %q", name, got, want)
		}
	}

	// 内容按层正确。
	for name, want := range map[string]string{
		`resources\sub\b.txt`: "local b",
		`resources\a.txt`:     "patch a",
		`resources\c.txt`:     "patch c",
	} {
		got, err := store.Read(name)
		if err != nil {
			t.Fatalf("Read(%q) 失败：%v", name, err)
		}
		if string(got) != want {
			t.Fatalf("Read(%q) = %q，应为 %q", name, got, want)
		}
	}

	// 名字规范化：/ 与 \、大小写不敏感。
	if got, err := store.Read("resources/sub/B.TXT"); err != nil || string(got) != "local b" {
		t.Fatalf("规范化读取失败：%q（err=%v）", got, err)
	}

	// 墓碑：三层都 miss。
	if store.Exists(`resources\gone.txt`) {
		t.Fatal("墓碑文件不应存在")
	}
	if _, err := store.Read(`resources\gone.txt`); err == nil {
		t.Fatal("读取墓碑文件应报错")
	}

	// 路径穿越被拒绝。
	if got := store.Layer(`..\res-manifest.json`); got != "" {
		t.Fatalf("路径穿越应被拒绝，实得 %q", got)
	}

	// 合并清单：a(patch) + b(local) + c(patch)，gone 被移除。
	files, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("合并清单条目 = %d，应为 3：%v", len(files), files)
	}
	byName := map[string]string{}
	for _, f := range files {
		byName[strings.ToLower(f.Name)] = f.Layer
	}
	for name, want := range map[string]string{
		strings.ToLower(`resources\a.txt`):     "patch",
		strings.ToLower(`resources\sub\b.txt`): "local",
		strings.ToLower(`resources\c.txt`):     "patch",
	} {
		if got := byName[name]; got != want {
			t.Fatalf("%s 层 = %q，应为 %q", name, got, want)
		}
	}

	// 热重载：换 patch 内容后 reload 可见。
	writeArchive(t, filepath.Join(dir, "res-patch.rep"), []mpq.Addition{
		{Name: `resources\a.txt`, Data: []byte("patch a v2")},
		{Name: `resources\c.txt`, Data: []byte("patch c")},
	})
	if err := store.Reload(); err != nil {
		t.Fatal(err)
	}
	if got, err := store.Read(`resources\a.txt`); err != nil || string(got) != "patch a v2" {
		t.Fatalf("reload 后内容不符：%q（err=%v）", got, err)
	}
}

func TestManifestlessBaseOnlyMode(t *testing.T) {
	dir := t.TempDir()
	writeArchive(t, filepath.Join(dir, "res-base.rep"), []mpq.Addition{
		{Name: `resources\a.txt`, Data: []byte("base a")},
	})

	// 无清单：按缺省布局（base + 本地层）打开。
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if store.HasPatch() {
		t.Fatal("无清单时不应有补丁层")
	}
	if got := store.Layer(`resources\a.txt`); got != "base" {
		t.Fatalf("Layer = %q，应为 base", got)
	}
	if data, err := store.Read(`resources\a.txt`); err != nil || string(data) != "base a" {
		t.Fatalf("读取失败：%q（err=%v）", data, err)
	}
}

func TestMissingBaseIsRejected(t *testing.T) {
	if _, err := Open(t.TempDir()); err == nil {
		t.Fatal("缺 base 层应被拒绝")
	}
}
