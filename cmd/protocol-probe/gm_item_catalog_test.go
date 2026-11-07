package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// GM「所有道具与换装」目录（/api/items；Java gm-web GmCatalog.items 对齐）。

// injectStubGMItemCatalog 换掉包级 gmItems 目录（gm_catalog_test 惯例）。
func injectStubGMItemCatalog(t *testing.T, catalog *gmItemCatalog) {
	t.Helper()
	old := gmItems
	gmItems = catalog
	t.Cleanup(func() { gmItems = old })
}

// stubGMItemIni 把 UTF-8 样本编码回 GBK——装载器吃的是原始 ini 字节，
// 样本里的中文名必须在 GBK 通道里走一遍才贴近线上。
func stubGMItemIni(t *testing.T, text string, wearable bool) gmItemIniSource {
	t.Helper()
	data, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(text))
	if err != nil {
		t.Fatalf("encode GBK: %v", err)
	}
	return gmItemIniSource{reader: bytes.NewReader(data), wearable: wearable}
}

func TestGMItemCatalogLoadAndSearch(t *testing.T) {
	equipment := "[weapon_jiuxian_01]\n" +
		"Name=jiuxian_dao_placeholder\n" + // 抄错名：idres 应覆盖
		"EquipType=Weapon\nViewID=2\nMaxAmount=1\nColorLevel=6\nNeedSex=0\nscript=Equipment\n\n" +
		"[cloth_sz001]\n" + // idres 无键 → INI 名
		"Name=时装衣1\nEquipType=Cloth\nViewID=2\nNeedSex=2\n\n" +
		"[inner_cloth_hidden]\n" + // ViewID 11 身份装内件：整组跳过
		"EquipType=InnerCloth\nViewID=11\n\n" +
		"[//注释节]\nName=不该出现\nViewID=1\n\n"
	tool := "[//修炼药剂]\nName=注释节\n\n" +
		"[box_gift_01]\nName=惊鸿礼盒\nViewID=1\nMaxAmount=1\nColorLevel=4\nscript=BoxItem\n\n" +
		"[item_num_placeholder]\nName=10101\nViewID=1\n\n" + // 纯数字占位名 → 空
		"[tool_no_viewid]\nName=无名工具\n\n" // 无 ViewID → 默认 1

	names := map[string]string{
		"weapon_jiuxian_01": "久仙刀",
	}
	catalog := loadGMItemCatalogFrom(names,
		stubGMItemIni(t, equipment, true),
		stubGMItemIni(t, tool, false))

	if len(catalog.rows) != 5 {
		t.Fatalf("rows=%d, want 5（注释节×2 与 ViewID 11 不进列表）: %+v", len(catalog.rows), catalog.rows)
	}
	byID := map[string]gmItemRow{}
	for _, row := range catalog.rows {
		byID[row.ConfigID] = row
	}
	// 名链三档：idres 覆盖 → INI 有效名 → 纯数字占位回落空。
	if row := byID["weapon_jiuxian_01"]; row.Name != "久仙刀" || !row.Wearable ||
		row.EquipType != "Weapon" || row.ViewID != 2 || row.MaxAmount != 1 ||
		row.ColorLevel != 6 || row.NeedSex != 0 || row.Script != "Equipment" {
		t.Fatalf("weapon row=%+v", row)
	}
	if row := byID["cloth_sz001"]; row.Name != "时装衣1" || row.NeedSex != 2 {
		t.Fatalf("cloth row=%+v, want INI 名回落+needSex 默认 2", row)
	}
	if row := byID["box_gift_01"]; row.Wearable || row.EquipType != "" || row.ViewID != 1 {
		t.Fatalf("tool row=%+v, want 非装备行 equipType 空", row)
	}
	if row := byID["item_num_placeholder"]; row.Name != "" {
		t.Fatalf("numeric placeholder row=%+v, want 空名", row)
	}
	if row := byID["tool_no_viewid"]; row.ViewID != 1 {
		t.Fatalf("no-viewid row=%+v, want 默认 viewId 1", row)
	}
	if _, hidden := byID["inner_cloth_hidden"]; hidden {
		t.Fatal("ViewID 11 身份装内件不应进目录")
	}

	// 中文名检索（GBK 名 + idres 名都可命中）。
	if rows := catalog.search("礼盒", 0); len(rows) != 1 || rows[0].ConfigID != "box_gift_01" {
		t.Fatalf("search 礼盒=%+v", rows)
	}
	if rows := catalog.search("久仙", 0); len(rows) != 1 || rows[0].ConfigID != "weapon_jiuxian_01" {
		t.Fatalf("search 久仙=%+v", rows)
	}
	// ConfigID 检索 + view 过滤。
	if rows := catalog.search("weapon_jiuxian_01", 1); len(rows) != 0 {
		t.Fatalf("view filter=%+v, want viewId=2 被 view=1 过滤", rows)
	}
	if rows := catalog.search("", 2); len(rows) != 2 {
		t.Fatalf("view=2 rows=%d, want 2 件装备", len(rows))
	}
	// 300 上限。
	bulk := &gmItemCatalog{}
	for i := 0; i < 400; i++ {
		bulk.rows = append(bulk.rows, gmItemRow{ConfigID: "x", ViewID: 1})
	}
	if rows := bulk.search("", 0); len(rows) != gmItemSearchLimit {
		t.Fatalf("limit=%d, want %d", len(rows), gmItemSearchLimit)
	}
}

func TestGMItemsEndpoint(t *testing.T) {
	injectStubGMItemCatalog(t, &gmItemCatalog{rows: []gmItemRow{
		{ConfigID: "weapon_jiuxian_01", Name: "久仙刀", ViewID: 2, Wearable: true, EquipType: "Weapon", NeedSex: 0},
		{ConfigID: "box_gift_01", Name: "惊鸿礼盒", ViewID: 1, Script: "BoxItem", NeedSex: 2},
	}})
	server := httptest.NewServer(newGMMux(newGMHub()))
	t.Cleanup(server.Close)

	response, err := http.Get(server.URL + "/api/items?q=%E4%B9%85%E4%BB%99&view=2") // q=久仙
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", response.StatusCode)
	}
	var rows []gmItemRow
	if err := json.NewDecoder(response.Body).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ConfigID != "weapon_jiuxian_01" {
		t.Fatalf("rows=%+v, want 久仙刀一件", rows)
	}

	// 非 GET 拒绝。
	post, err := http.Post(server.URL+"/api/items", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	post.Body.Close()
	if post.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("POST status=%d, want 405", post.StatusCode)
	}
}
