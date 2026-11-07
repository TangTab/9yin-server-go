package main

import (
	"testing"
	"time"
)

// 这里刻意依赖真实资源：它保护真实的现代链条
// StaticData -> 精确当前等级 varprop -> PropModify 包，而不是
// 保留旧的 30 级入门内功常量。
func TestModernNeiGongEffectUsesCurrentLevelVarProp(t *testing.T) {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		t.Fatalf("load modern inner-power catalog: %v", err)
	}
	effect, err := catalog.effect(114, 32) // ng_yh_001 / 明玉神功，角色 2 持久化的等级。
	if err != nil {
		t.Fatalf("evaluate ng_yh_001 level 32: %v", err)
	}
	if effect.buffID != "buf_ng_yh_001" || effect.staticData != 13073 || effect.buffLevel != 3 {
		t.Fatalf("unexpected buff route: id=%q static=%d level=%d", effect.buffID, effect.staticData, effect.buffLevel)
	}
	want := map[string]int32{
		"StrAdd": 57, "StaAdd": 59, "DexAdd": 72, "IngAdd": 101, "SpiAdd": 69,
		"MaxHPAdd": 4230, "MaxMPAdd": 1180, "MaxParryAdd": 5160, "MinMagicDefAdd": 105,
	}
	for name, value := range want {
		if got := effect.stats[name]; got != value {
			t.Errorf("%s = %d, want %d", name, got, value)
		}
	}
}

func TestPlayerProgressRestoresExactEquippedEffect(t *testing.T) {
	progress := newPlayerProgress()
	saved := progress.snapshot()
	saved.CurNeiGong = "ng_yh_001"
	for index := range saved.Books {
		if saved.Books[index].ConfigID == "ng_yh_001" {
			saved.Books[index].Level = 32
		}
	}
	progress.restore(saved, time.Now().UTC())
	book := progress.book(progress.curNeiGong)
	if book == nil {
		t.Fatal("missing restored equipped book")
	}
	if book.level != 32 || book.maxHPAdd != 4230 || book.maxMPAdd != 1180 || book.ingAdd != 101 {
		t.Fatalf("restored effect = level %d hp %d mp %d ing %d, want 32/4230/1180/101", book.level, book.maxHPAdd, book.maxMPAdd, book.ingAdd)
	}
}

func TestEveryStarterBookResolvesItsExactModernLevel(t *testing.T) {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, book := range newPlayerProgress().books {
		_, err := catalog.effect(book.staticData, book.level)
		if err != nil {
			t.Errorf("%s StaticData=%d level=%d: %v", book.configID, book.staticData, book.level, err)
		}
	}
}

// TestModernNeiGongCatalogWuXingFromWuXueBaseInfo 钉死五行表装载：
// faculty/wuxuebaseinfo.ini 的 nWuXing（Java NeiGongCatalog.wuXing 同源）。
// 套路与内功共用同一目录（CS_yhwq_hsqs01=2 柔 / ng_yh_001=6 阴柔）；
// mind_ 重复节按节名收录，不污染 ng_* 主键。夹具缺该文件时跳过。
func TestModernNeiGongCatalogWuXingFromWuXueBaseInfo(t *testing.T) {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		t.Skipf("load modern inner-power catalog: %v", err)
	}
	if catalog.wuXingOf("ng_wd_001") == 0 {
		t.Skip("wuxuebaseinfo.ini absent from resource store")
	}
	want := map[string]int32{
		"ng_wd_001":      5,
		"ng_yh_001":      6,
		"ng_th_001":      5,
		"CS_yhwq_hsqs01": 2,
		"CS_wd_tjq01":    5,
		"zs_default_01":  5,
	}
	for configID, value := range want {
		if got := catalog.wuXingOf(configID); got != value {
			t.Errorf("wuXingOf(%s)=%d, want %d", configID, got, value)
		}
	}
	if got := catalog.wuXingOf("mind_ng_wd_001"); got != 5 {
		t.Errorf("mind_ duplicate section key =%d, want 5", got)
	}
	if got := catalog.wuXingOf("no_such_config"); got != 0 {
		t.Errorf("wuXingOf(unknown)=%d, want 0", got)
	}
}
