package main

import (
	"strings"
	"testing"
	"time"
)

// 镜像 Java RoleVisualSexTest：性别推断（inferSex/resolveRoleVisual）。
// DB 无独立性别列，男女只体现在 role_appearances.appearance 串前缀
// （photo=girl|boy、路径段 g_|b_、_g0xxx|_b0xxx）——线上实证
// role 665=girl01/g_hair9/cloth_g0003、role 618=b_* 全路径。
// 桩表与 Java 同构：同 ArtPack 男女各一模型，验证 resolve 按推断性别取列。
func injectStubEquipmentModels(t *testing.T) {
	t.Helper()
	old := equipmentModels
	equipmentModels = &equipmentResolver{
		artByItem: map[string]string{"cloth_g0003": "7", "cloth_b0002": "7"},
		models:    map[string]itemModels{"7": {male: `obj\m\cloth`, female: `obj\f\cloth`}},
	}
	t.Cleanup(func() { equipmentModels = old })
}

// 线上 role 665 实值：photo=girl01 → sex=1，衣装按女性模型解析。
func TestFemaleRole665InfersFemale(t *testing.T) {
	injectStubEquipmentModels(t)
	appearance := []string{"book2", `icon\players\girl01_create.png`,
		"33333", "1", `obj\char\g_hair\g_hair9`,
		"cloth_g0003", "pants_g0003", "shoes_g0003", "school_junzitang"}
	visual := resolveRoleVisual(appearance)
	if visual.sex != 1 {
		t.Fatalf("sex=%d, want 1（girl01 photo 应判女）", visual.sex)
	}
	if visual.cloth != `obj\f\cloth` {
		t.Fatalf("cloth=%q, want obj\\f\\cloth（取 FemaleModel）", visual.cloth)
	}
}

// 线上 role 618 实值：全路径段 b_* → sex=0，模型原样（含反斜杠不查表）。
func TestMaleRole618InfersMale(t *testing.T) {
	injectStubEquipmentModels(t)
	appearance := []string{"book2", `icon\players\boy01_create.png`,
		"33333", "1", `obj\char\b_shizhuang033\b_Szhelmet033`,
		`obj\char\b_festival_1_106\b_cloth_1_106`,
		`obj\char\b_festival_1_106\b_pants_1_106`,
		`obj\char\b_festival_1_106\b_shoes_1_106`}
	visual := resolveRoleVisual(appearance)
	if visual.sex != 0 {
		t.Fatalf("sex=%d, want 0", visual.sex)
	}
	if visual.cloth != `obj\char\b_festival_1_106\b_cloth_1_106` {
		t.Fatalf("cloth=%q, want 原样透传", visual.cloth)
	}
}

// photo 缺失时逐级回退：hair 路径段 g_ → 女。
func TestHairSegmentIsFallbackEvidence(t *testing.T) {
	injectStubEquipmentModels(t)
	visual := resolveRoleVisual([]string{"book1", "", "", "",
		`obj\char\g_hair\g_hair2`, "", "", ""})
	if visual.sex != 1 {
		t.Fatalf("sex=%d, want 1（g_hair 路径段应判女）", visual.sex)
	}
	if visual.hair != `obj\char\g_hair\g_hair2` {
		t.Fatalf("hair=%q, want 存值原样", visual.hair)
	}
}

// 女性角色空部位回退女默认（photo=girl + hair/衣裤鞋全空）。
func TestFemaleDefaultsWhenPartsEmpty(t *testing.T) {
	injectStubEquipmentModels(t)
	visual := resolveRoleVisual([]string{"book1", `icon\players\girl01_create.png`,
		"", "", "", "", "", ""})
	if visual.sex != 1 {
		t.Fatalf("sex=%d, want 1", visual.sex)
	}
	if visual.photo != `icon\players\girl01_create.png` {
		t.Fatalf("photo=%q, want 存值", visual.photo)
	}
	if visual.hair != `obj\char\g_hair\g_hair1` {
		t.Fatalf("hair=%q, want 女默认发型", visual.hair)
	}
	if visual.cloth != "cloth_g0001" || visual.pants != "pants_g0001" || visual.shoes != "shoes_g0001" {
		t.Fatalf("衣裤鞋=%q/%q/%q, want cloth_g0001/pants_g0001/shoes_g0001 女初始装",
			visual.cloth, visual.pants, visual.shoes)
	}
}

// photo/hair 全缺、只剩裸衣装配置名：cloth_g 前缀 → 女。
func TestBareClothConfigIsLastEvidence(t *testing.T) {
	injectStubEquipmentModels(t)
	visual := resolveRoleVisual([]string{"book1", "", "", "", "", "cloth_g0003", "", ""})
	if visual.sex != 1 {
		t.Fatalf("sex=%d, want 1（cloth_g0003 应判女）", visual.sex)
	}
}

// 空外观（线上 role 674 等）：无证据 → 男（与历史行为一致）。
func TestEmptyAppearanceStaysMale(t *testing.T) {
	injectStubEquipmentModels(t)
	visual := resolveRoleVisual(nil)
	if visual.sex != 0 {
		t.Fatalf("sex=%d, want 0", visual.sex)
	}
	if visual.photo != `icon\players\boy01_create.png` {
		t.Fatalf("photo=%q, want 男默认", visual.photo)
	}
	if visual.cloth != "cloth_b0001" {
		t.Fatalf("cloth=%q, want cloth_b0001", visual.cloth)
	}
}

// photo 证据优先于 hair：boy01 + g_hair → 男。
func TestPhotoBeatsHairEvidence(t *testing.T) {
	injectStubEquipmentModels(t)
	visual := resolveRoleVisual([]string{"book1", `icon\players\boy01_create.png`,
		"", "", `obj\char\g_hair\g_hair2`, "", "", ""})
	if visual.sex != 0 {
		t.Fatalf("sex=%d, want 0（photo=boy 优先于 hair=g_）", visual.sex)
	}
}

// 建档链路端到端：女性外观的 0x04 选角帧 rolePara f0=1（客户端
// form_roles.lua [1]=sex 驱动选人模型），f7/f33 随性别取发型。
func TestFemaleRoleParaCarriesSex(t *testing.T) {
	injectStubEquipmentModels(t)
	msg := oneRoleLoginSuccess(time.Date(2026, 7, 11, 12, 0, 0, 0, time.Local), "女侠",
		[]string{"book2", `icon\players\girl01_create.png`, "33333", "1",
			`obj\char\g_hair\g_hair9`, "cloth_g0003", "pants_g0003", "shoes_g0003"},
		"", "")
	_, rolePara := roleListFrameStrings(t, msg)
	fields := strings.Split(rolePara, ",")
	if len(fields) < 36 {
		t.Fatalf("rolePara fields=%d, want >=36", len(fields))
	}
	if got := fields[0]; got != "1" {
		t.Fatalf("f0 sex=%q, want 1（form.sex 驱动选人组合模型）", got)
	}
	if got := fields[7]; got != `obj\char\g_hair\g_hair9` {
		t.Fatalf("f7 hat/hair=%q, want 女发型存值", got)
	}
	if got := fields[9]; got != `obj\f\cloth` {
		t.Fatalf("f9 cloth=%q, want FemaleModel", got)
	}
	if got := fields[33]; got != `obj\char\g_hair\g_hair9` {
		t.Fatalf("f33 hair=%q, want 与 f7 同口径的女发型", got)
	}
}
