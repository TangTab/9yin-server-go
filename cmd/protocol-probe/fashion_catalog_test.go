package main

import (
	"os"
	"path/filepath"
	"testing"
)

// 镜像 Java FashionSuitCatalogTest：目录解析、identitySuitsOf 前缀兜底、
// 时装四段去重、大小写不敏感收录判定。桩表复刻真实 fashion_suit.ini 的
// [base_info] + 段名/r= 行结构。

func injectStubFashionCatalog(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	ini := "[base_info]\n" +
		"shengfen=school,force,newschool,shijia,guild,life,activity\n" +
		"fashion=hat,coat,shoes,suit\n" +
		"school=school_shaolin,school_wudang\n" +
		"force=force_yihua\n" +
		"newschool=newschool_tianya\n" +
		"\n[school_shaolin]\nr=shaolin01\nr=shaolin02\n\n" +
		"[school_wudang]\nr=wudang01\n\n" +
		"[force_yihua]\nr=yihua01\n\n" +
		"[newschool_tianya]\nr=tianya01\n\n" +
		"[shijia]\nr=shijia01\n\n" +
		"[hat]\nr=fashion_hat_a\n\n[coat]\nr=FASHION_SUIT_G027\n\n" +
		"[shoes]\nr=fashion_shoes_a\n\n[suit]\nr=g_emei01\nr=FASHION_SUIT_G027\n"
	if err := os.WriteFile(filepath.Join(dir, "fashion_suit.ini"), []byte(ini), 0644); err != nil {
		t.Fatal(err)
	}
	old := fashionSuits
	fashionSuits = loadFashionSuitCatalog(dir)
	t.Cleanup(func() { fashionSuits = old })
}

func TestFashionCatalogIdentitySuitsOf(t *testing.T) {
	injectStubFashionCatalog(t)
	// 旧门派 faction 不带前缀 → school_ 前缀兜底。
	if suits := fashionSuits.identitySuitsOf("shaolin"); len(suits) != 2 || suits[0] != "shaolin01" {
		t.Fatalf("shaolin suits=%v", suits)
	}
	// force_/newschool_ 同名段直命中。
	if suits := fashionSuits.identitySuitsOf("force_yihua"); len(suits) != 1 || suits[0] != "yihua01" {
		t.Fatalf("force_yihua suits=%v", suits)
	}
	if suits := fashionSuits.identitySuitsOf("newschool_tianya"); len(suits) != 1 {
		t.Fatalf("newschool_tianya suits=%v", suits)
	}
	// 未知门派/空 faction → 空。
	for _, faction := range []string{"", "unknown"} {
		if suits := fashionSuits.identitySuitsOf(faction); len(suits) != 0 {
			t.Fatalf("faction %q suits=%v, want 空", faction, suits)
		}
	}
}

func TestFashionCatalogFashionSuitsAllDedup(t *testing.T) {
	injectStubFashionCatalog(t)
	suits := fashionSuits.fashionSuitsAll()
	// 四段去重：FASHION_SUIT_G027 同时在 coat 与 suit 段只留一份
	//（1 hat + 1 coat + 1 shoes + 2 suit - 1 重复 = 4）。
	if len(suits) != 4 {
		t.Fatalf("fashion suits=%v, want 4（去重后）", suits)
	}
	seen := map[string]bool{}
	for _, suit := range suits {
		if seen[suit] {
			t.Fatalf("重复条目 %s", suit)
		}
		seen[suit] = true
	}
}

func TestFashionCatalogIsFashionSuitCaseInsensitive(t *testing.T) {
	injectStubFashionCatalog(t)
	if !fashionSuits.isFashionSuit("fashion_suit_g027") {
		t.Fatal("小写形态应命中")
	}
	if fashionSuits.isFashionSuit("shaolin01") {
		t.Fatal("身份装不在时装目录")
	}
	if fashionSuits.isFashionSuit("") {
		t.Fatal("空配置应拒绝")
	}
}

func TestFashionCatalogAllIdentitySuitsIncludesPlainSections(t *testing.T) {
	injectStubFashionCatalog(t)
	all := fashionSuits.allIdentitySuits()
	// school 2 段 3 套 + force 1 + newschool 1 + shijia 1 = 6。
	if len(all) != 6 {
		t.Fatalf("all identity suits=%v, want 6", all)
	}
}

// 合并/判定辅助：精确去重升序（收录判定才走大小写不敏感）。
func TestMergeFashionSuitsDedupSorted(t *testing.T) {
	merged := mergeFashionSuits([]string{"b", "a"}, []string{"a", "c"})
	if len(merged) != 3 || merged[0] != "a" || merged[2] != "c" {
		t.Fatalf("merged=%v, want [a b c]", merged)
	}
	if !containsIgnoreCase([]string{"FASHION_SUIT_G027"}, "fashion_suit_g027") {
		t.Fatal("大小写不敏感收录判定失败")
	}
}
