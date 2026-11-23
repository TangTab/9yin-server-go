package itemdata

// itemdata_test.go share/item 方言解析单测：注释节跳过、script/EquipType
// 装备页覆盖、文件级分类、GBK 自适应。

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

const sampleItemINI = `[//注释节]
SellPrice1=1

[jhzb_xiulian_000]
Amount=1
ItemType=1206
MaxAmount=200

[cloth_demo]
EquipType=FashionCoat
ItemType=141

[weapon_demo]
script=Equipment
ItemType=101
`

func TestLoadDirClassification(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tool_item.ini", sampleItemINI)
	write("item_material_static.ini", "[ore_demo]\nItemType=50\n")
	write("item_task_static.ini", "[task_demo]\nItemType=60\n")
	write("equipment.ini", "[equip_demo]\nItemType=141\n")

	catalog, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		id       string
		itemType int
		viewID   int
		max      int
	}{
		{"jhzb_xiulian_000", 1206, 1, 200},
		{"cloth_demo", 141, 2, 0},
		{"weapon_demo", 101, 2, 0},
		{"ore_demo", 50, 3, 0},
		{"task_demo", 60, 4, 0},
		{"equip_demo", 141, 2, 0},
	}
	for _, c := range cases {
		entry, ok := catalog.Lookup(c.id)
		if !ok {
			t.Fatalf("%s 缺席", c.id)
		}
		if entry.ItemType != c.itemType || entry.ViewID != c.viewID || entry.MaxAmount != c.max {
			t.Fatalf("%s 异常: %+v want type=%d view=%d max=%d", c.id, entry, c.itemType, c.viewID, c.max)
		}
	}
	if _, ok := catalog.Lookup("//注释节"); ok {
		t.Fatal("注释节不应入目录")
	}
}

func TestLoadGBKAdaptive(t *testing.T) {
	dir := t.TempDir()
	raw, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(sampleItemINI))
	if err != nil {
		t.Skipf("GBK 编码器不可用: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tool_item.ini"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if entry, ok := catalog.Lookup("jhzb_xiulian_000"); !ok || entry.ItemType != 1206 {
		t.Fatalf("GBK 转码异常: %+v ok=%v", entry, ok)
	}
}
