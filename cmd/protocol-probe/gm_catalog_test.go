package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// GM 目录端点（newGMMux httptest 驱动，零 DB/零场景依赖——名字钩子缺席
// 回退 ConfigID）。镜像 gm-panel-adaptation 惯例。

func newCatalogTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	hub := newGMHub()
	hub.itemNames = func(configIDs []string) map[string]string {
		return map[string]string{"shaolin01": "少林俗家一阶"}
	}
	server := httptest.NewServer(newGMMux(hub))
	t.Cleanup(server.Close)
	return server
}

func TestGMFashionCatalogEndpoint(t *testing.T) {
	injectStubFashionCatalog(t)
	server := newCatalogTestServer(t)

	response, err := http.Get(server.URL + "/api/fashion-catalog")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", response.StatusCode)
	}
	var categories []gmCatalogCategory
	if err := json.NewDecoder(response.Body).Decode(&categories); err != nil {
		t.Fatal(err)
	}
	byKey := map[string]gmCatalogCategory{}
	for _, category := range categories {
		byKey[category.Key] = category
	}
	if school, ok := byKey["identity_school"]; !ok || len(school.Items) != 3 {
		t.Fatalf("identity_school=%+v ok=%t, want 3 套（shaolin 2 + wudang 1）", school, ok)
	}
	if coat, ok := byKey["fashion_coat"]; !ok || len(coat.Items) != 1 || coat.Items[0].ConfigID != "FASHION_SUIT_G027" {
		t.Fatalf("fashion_coat=%+v", coat)
	}
	// 名字钩子命中 game_items 名，未命中回退 ConfigID。
	found := false
	for _, item := range byKey["identity_school"].Items {
		if item.ConfigID == "shaolin01" && item.Name == "少林俗家一阶" {
			found = true
		}
	}
	if !found {
		t.Fatalf("item name hook not applied: %+v", byKey["identity_school"].Items)
	}
}

func TestGMCardsEndpoints(t *testing.T) {
	injectStubSkinCatalog(t)
	server := newCatalogTestServer(t)

	response, err := http.Get(server.URL + "/api/cards")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var cards []gmCardRow
	if err := json.NewDecoder(response.Body).Decode(&cards); err != nil {
		t.Fatal(err)
	}
	if len(cards) != 4 {
		t.Fatalf("cards=%d, want 4", len(cards))
	}
	if cards[0].ID != 1 || cards[0].ConfigID != "blade_card_001" || cards[0].Category != "Weapon" {
		t.Fatalf("first card=%+v", cards[0])
	}

	hairResponse, err := http.Get(server.URL + "/api/hair-cards")
	if err != nil {
		t.Fatal(err)
	}
	defer hairResponse.Body.Close()
	var hair []gmCardRow
	if err := json.NewDecoder(hairResponse.Body).Decode(&hair); err != nil {
		t.Fatal(err)
	}
	if len(hair) != 1 || hair[0].ID != 3 || hair[0].Category != "Hat" {
		t.Fatalf("hair cards=%+v, want 仅发型皮肤 3", hair)
	}
}

// 0x04 rolePara 携带显示类型：无档 golden "1"；有档取存档值。
func TestRoleParaCarriesShowEquipTypeLook(t *testing.T) {
	injectStubEquipmentModels(t)
	appearance := []string{"book2", `icon\players\girl01_create.png`, "33333", "1",
		`obj\char\g_hair\g_hair9`, "cloth_g0003", "pants_g0003", "shoes_g0003"}

	// 默认 look（无 DB 外观态）：f6 保持 golden "1"。
	msg := oneRoleLoginSuccess(time.Date(2026, 7, 11, 12, 0, 0, 0, time.Local), "女侠",
		appearance, "", "")
	_, rolePara := roleListFrameStrings(t, msg)
	fields := strings.Split(rolePara, ",")
	if fields[6] != "1" {
		t.Fatalf("default f6=%q, want golden 1", fields[6])
	}

	// 时装偏好 + 套装衣穿戴：f6=2、f9 取时装模型。
	look := defaultRoleListLook(appearance)
	look.showEquipType = "2"
	look.visual.cloth = `obj\f\fashion_coat`
	msg = oneRoleLoginSuccessWithLook(time.Date(2026, 7, 11, 12, 0, 0, 0, time.Local), "女侠",
		appearance, "", "", look)
	_, rolePara = roleListFrameStrings(t, msg)
	fields = strings.Split(rolePara, ",")
	if fields[6] != "2" {
		t.Fatalf("pref f6=%q, want 2", fields[6])
	}
	if fields[9] != `obj\f\fashion_coat` {
		t.Fatalf("f9=%q, want 时装模型", fields[9])
	}
}
