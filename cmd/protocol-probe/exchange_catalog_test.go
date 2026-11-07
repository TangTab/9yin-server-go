package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 兑换目录解析（msg70 capitalType=3 的计价源；格式见
// resources/modern/share/item/exchangeitem.ini 实证样本）。

func stubExchangeCatalog(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "exchangeitem.ini")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write stub exchangeitem.ini: %v", err)
	}
	original := defaultExchangeINIPath
	defaultExchangeINIPath = path
	resetExchangeCatalogCache()
	t.Cleanup(func() {
		defaultExchangeINIPath = original
		resetExchangeCatalogCache()
	})
}

func TestParseExchangeCatalogEntryShapes(t *testing.T) {
	stubExchangeCatalog(t, `[13093]
Item=item_honor_school13,100;stuff_jiebang_01,1

[13094]
Condition=120958
Prop=CapitalType2,100000

[7202]
Condition2=109035
Item=my_home_duihuan_01_ju,10
Prop=CapitalType1,500;CapitalType2,20
`)
	catalog, err := exchangeCatalogFor(defaultExchangeINIPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	entry, ok := catalog.entry(13093)
	if !ok || len(entry.items) != 2 || entry.items[0].configID != "item_honor_school13" || entry.items[0].count != 100 ||
		entry.items[1].configID != "stuff_jiebang_01" || entry.items[1].count != 1 {
		t.Fatalf("entry 13093=%+v", entry)
	}
	if len(entry.props) != 0 {
		t.Fatalf("entry 13093 unexpected props=%v", entry.props)
	}

	entry, ok = catalog.entry(13094)
	if !ok || len(entry.items) != 0 || len(entry.props) != 1 || entry.props[0].name != "CapitalType2" || entry.props[0].cost != 100000 {
		t.Fatalf("entry 13094=%+v", entry)
	}
	if entry.condition != "120958" {
		t.Fatalf("entry 13094 condition=%q", entry.condition)
	}

	entry, ok = catalog.entry(7202)
	if !ok || len(entry.items) != 1 || len(entry.props) != 2 || entry.props[1].name != "CapitalType2" || entry.props[1].cost != 20 {
		t.Fatalf("entry 7202=%+v", entry)
	}
	if entry.condition != "109035" {
		t.Fatalf("entry 7202 condition=%q", entry.condition)
	}
	if _, ok := catalog.entry(999999); ok {
		t.Fatal("unknown id should miss")
	}
	if _, ok := catalog.entry(0); ok {
		t.Fatal("zero id should miss")
	}
}

// 真实资源树冒烟（NINEYIN_SERVER_ROOT 未配置时跳过）：4.5 万行
// exchangeitem.ini 必须整文件解析成功，并抽检实证样本节。
func TestParseRealExchangeCatalogSmoke(t *testing.T) {
	if _, err := statResourcePath(defaultExchangeINIPath); err != nil {
		t.Skipf("exchange catalog resource unavailable: %v", err)
	}
	resetExchangeCatalogCache()
	t.Cleanup(resetExchangeCatalogCache)
	catalog, err := exchangeCatalogFor(defaultExchangeINIPath)
	if err != nil {
		t.Fatalf("load real catalog: %v", err)
	}
	// 4.5 万行 ≈ 1.05 万节（每节约 4 行键值）。
	if len(catalog.byID) < 10000 {
		t.Fatalf("entries=%d, want >=10000 (real archive)", len(catalog.byID))
	}
	entry, ok := catalog.entry(13093)
	if !ok || len(entry.items) != 2 || entry.items[0].configID != "item_honor_school13" || entry.items[0].count != 100 {
		t.Fatalf("spot check 13093=%+v ok=%t", entry, ok)
	}
	entry, ok = catalog.entry(13094)
	if !ok || len(entry.props) != 1 || entry.props[0].name != "CapitalType2" || entry.props[0].cost != 100000 || entry.condition != "120958" {
		t.Fatalf("spot check 13094=%+v ok=%t", entry, ok)
	}
}

func TestParseExchangeCatalogRejectsBadLines(t *testing.T) {
	for name, content := range map[string]string{
		"bad section":  "[not-a-number]\nItem=a,1\n",
		"item no pair": "[1]\nItem=just-name\n",
		"item bad int": "[1]\nItem=name,x\n",
		"prop no pair": "[1]\nProp=CapitalType1\n",
	} {
		stubExchangeCatalog(t, content)
		if _, err := exchangeCatalogFor(defaultExchangeINIPath); err == nil {
			t.Fatalf("%s: expected parse error", name)
		} else if !strings.Contains(err.Error(), "exchange catalog") {
			t.Fatalf("%s: unexpected error shape: %v", name, err)
		}
	}
	stubExchangeCatalog(t, "// 仅注释\n")
	if _, err := exchangeCatalogFor(defaultExchangeINIPath); err == nil || !strings.Contains(err.Error(), "is empty") {
		t.Fatalf("empty catalog should error, got %v", err)
	}
}
