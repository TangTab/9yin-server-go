package neigongdata

// neigongdata_test.go 目录装载单测：夹具 share 树（五行/曲线/varprop
// 档/静态区间/buff StaticData）解析与查询面。

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestCatalogQueries(t *testing.T) {
	root := writeTree(t, map[string]string{
		"share/faculty/wuxuebaseinfo.ini": "[ng_wd_001]\nnWuXing=5\n[ng_jh_201]\nnWuXing=7\n",
		"share/faculty/wuxuelevelinfo.ini": "[ng_wd_001]\nr=1,200,100\nr=2,600,100\n",
		"share/skill/neigong/neigong_static.ini":   "[5459]\nMinVarPropNo=10\nMaxVarPropNo=12\n",
		"share/skill/neigong/neigong_varprop.ini":  "[10]\nLevel=1\nNeiGongLevel=3\n[11]\nLevel=2\nNeiGongLevel=6\nBufferID=buf_x\nBufferLevel=2\nStrAdd=4\n",
		"share/skill/buff_new.ini":                 "[buf_x]\nStaticData=1742\n",
	})
	catalog, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := catalog.WuXingOf("ng_wd_001"); got != 5 {
		t.Fatalf("WuXingOf(ng_wd_001)=%d, want 5", got)
	}
	if got := catalog.WuXingOf("ng_jh_201"); got != 7 {
		t.Fatalf("WuXingOf(ng_jh_201)=%d, want 7", got)
	}
	if got := catalog.WuXingOf("absent"); got != 0 {
		t.Fatalf("未收录应 0, got %d", got)
	}
	if total, ok := catalog.LevelTotal("ng_wd_001", 2); !ok || total != 600 {
		t.Fatalf("LevelTotal(l2)=%d,%v, want 600,true", total, ok)
	}
	if _, ok := catalog.LevelTotal("ng_wd_001", 9); ok {
		t.Fatal("曲线缺级应 ok=false")
	}
	if row, ok := catalog.VarPropRow(5459, 2); !ok || row.BufferID != "buf_x" || row.StrAdd != 4 || row.NeiGongLevel != 6 {
		t.Fatalf("VarPropRow(5459,2)=%+v,%v", row, ok)
	}
	if _, ok := catalog.VarPropRow(5459, 3); ok {
		t.Fatal("档区间无该级应 ok=false")
	}
	if static, ok := catalog.BuffStaticData("buf_x"); !ok || static != 1742 {
		t.Fatalf("BuffStaticData(buf_x)=%d,%v", static, ok)
	}
}

func TestLoadRequiresWuXing(t *testing.T) {
	root := writeTree(t, map[string]string{
		"share/faculty/wuxuebaseinfo.ini": "[x]\nOtherKey=1\n",
	})
	if _, err := Load(root); err == nil {
		t.Fatal("无 nWuXing 行应整体报错")
	}
}

func TestNilCatalogDegraded(t *testing.T) {
	var catalog *Catalog
	if got := catalog.WuXingOf("ng_wd_001"); got != 0 {
		t.Fatalf("nil 目录 WuXingOf=%d, want 0", got)
	}
	if _, ok := catalog.VarPropRow(1, 1); ok {
		t.Fatal("nil 目录 VarPropRow 应 false")
	}
}
