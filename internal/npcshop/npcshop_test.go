package npcshop

// npcshop_test.go shop.ini 方言解析单测：GBK/UTF-8 自适应、元数据键、
// 重复数字键物品行、页数合成、脏行报错。

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

const sampleShopINI = `[Shop_yaopin_00101]
NameID=药堂
PageInfo=Page1,Page2
Type=0
0=drug_hp_small,99,0,500,0,1,0,0,0,
0=drug_mp_small,99,1,800,0,1,1,0,0,
1=exchange_token,1,3,0,0,2,0,13094,543,,

[Shop_weapon_00401]
NameID=铁匠铺
Type=1
// 注释行跳过
0=sword_iron_001,1,0,2500,0,1,0,,
`

func TestParseShopINI(t *testing.T) {
	catalog, err := Parse([]byte(sampleShopINI))
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Count() != 2 {
		t.Fatalf("节数 = %d, want 2", catalog.Count())
	}
	sec, ok := catalog.Section("Shop_yaopin_00101")
	if !ok {
		t.Fatal("缺 Shop_yaopin_00101")
	}
	if sec.NameID != "药堂" || sec.Type != 0 {
		t.Fatalf("元数据异常: %+v", sec)
	}
	if sec.Pages != 2 { // PageInfo 2 页与行最大页 2 取大
		t.Fatalf("页数 = %d, want 2", sec.Pages)
	}
	if len(sec.Items) != 3 {
		t.Fatalf("物品数 = %d, want 3", len(sec.Items))
	}
	first := sec.Items[0]
	if first.ConfigID != "drug_hp_small" || first.Amount != 99 || first.CapitalType != 0 ||
		first.Price != 500 || first.Page != 1 || first.Position != 0 {
		t.Fatalf("行 0 解析异常: %+v", first)
	}
	second := sec.Items[1]
	if second.Position != 1 || second.CapitalType != 1 || second.Price != 800 {
		t.Fatalf("行 1 解析异常: %+v", second)
	}
	third := sec.Items[2]
	if third.Page != 2 || third.Position != 0 || third.CapitalType != 3 ||
		third.ExchangeID != 13094 || third.ConditionID != 543 {
		t.Fatalf("兑换行解析异常: %+v", third)
	}
	wpn, ok := catalog.Section("Shop_weapon_00401")
	if !ok || wpn.Pages != 1 || len(wpn.Items) != 1 || wpn.Type != 1 {
		t.Fatalf("武器店异常: %+v", wpn)
	}
	// 缺节/空档。
	if _, ok := catalog.Section("nope"); ok {
		t.Fatal("未知节应未命中")
	}
	if _, err := Parse([]byte("")); err == nil {
		t.Fatal("空档应报错")
	}
	if _, err := Parse([]byte("[S]\n0=a,1\n")); err == nil {
		t.Fatal("列数不足应报错")
	}
}

func TestLoadGBKAdaptive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shop.ini")
	if err := os.WriteFile(path, []byte(sampleShopINI), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Count() != 2 {
		t.Fatalf("UTF-8 直通节数 = %d", catalog.Count())
	}
	// GBK 档（NameID 中文走 GBK 字节）。
	gbkDir := t.TempDir()
	gbkPath := filepath.Join(gbkDir, "shop.ini")
	raw, encErr := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(sampleShopINI))
	if encErr != nil {
		t.Skipf("GBK 编码器不可用: %v", encErr)
	}
	if err := os.WriteFile(gbkPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	gbkCatalog, err := Load(gbkPath)
	if err != nil {
		t.Fatal(err)
	}
	sec, ok := gbkCatalog.Section("Shop_yaopin_00101")
	if !ok || sec.NameID != "药堂" {
		t.Fatalf("GBK 转码异常: %+v ok=%v", sec, ok)
	}
}
