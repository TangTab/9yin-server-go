package npcdata

// npcdata_test.go born03 烟雨庄真档装载断言（testdata/ini/Scene 与
// ini/Npc/NpcConfig 树，npcgen 产出的九阴真档）。

import (
	"os"
	"testing"
)

const testRoot = "../../testdata" // 资源根（ini/Scene、ini/Npc 在此；fsgame ResourcePath 同口径）

func TestLoadSceneNPCsBorn03(t *testing.T) {
	if _, err := os.Stat(testRoot + "/ini/Scene/born03_yanyuzhuang/npc_creator.xml"); err != nil {
		t.Skipf("真档缺席: %v", err)
	}
	npcs, err := LoadSceneNPCs(testRoot, "born03_yanyuzhuang")
	if err != nil {
		t.Fatalf("LoadSceneNPCs: %v", err)
	}
	byFile := map[string]int{}
	for _, n := range npcs {
		byFile[n.File]++
		if n.ID == "" {
			t.Fatal("静态项缺 id")
		}
	}
	// 白名单类别条数（npc_creator.xml 真档定数；attacknpc/creator 族被排除）。
	want := map[string]int{
		"commonnpc.xml": 1448, "funcnpc.xml": 229, "itemnpc.xml": 21,
		"lightnpc.xml": 146, "guildbuildingnpc.xml": 1, "gathernpc.xml": 61,
		"other.xml": 1,
	}
	for file, n := range want {
		if byFile[file] != n {
			t.Errorf("%s 条数 = %d, want %d", file, byFile[file], n)
		}
	}
	if total := len(npcs); total != 1907 {
		t.Errorf("静态项总数 = %d, want 1907", total)
	}
}

func TestJoinTemplates(t *testing.T) {
	if _, err := os.Stat(testRoot + "/ini/Npc/NpcConfig"); err != nil {
		t.Skipf("真档缺席: %v", err)
	}
	ResetTemplateCache()
	templates, err := LoadTemplates(testRoot)
	if err != nil {
		t.Fatalf("LoadTemplates: %v", err)
	}
	if len(templates) < 1000 {
		t.Fatalf("模板数 = %d, want ≥1000", len(templates))
	}
	npcs, err := LoadSceneNPCs(testRoot, "born03_yanyuzhuang")
	if err != nil {
		t.Fatalf("LoadSceneNPCs: %v", err)
	}
	hit := Join(npcs, templates)
	t.Logf("born03 静态项 %d 联查命中 %d（%.0f%%；未命中≈lightnpc 火盆族 NpcConfig 无模板，跳过不下发）",
		len(npcs), hit, 100*float64(hit)/float64(len(npcs)))
	if hit < len(npcs)*9/10 {
		t.Errorf("联查命中率 %.0f%% < 90%%", 100*float64(hit)/float64(len(npcs)))
	}
	// 抽样：人物/村庄摆放 id 能查到模型资源。
	for _, want := range []string{"npc_cwqy_linzongren", "yyz066"} {
		var found bool
		for _, n := range npcs {
			if n.ID == want {
				if !n.Hit || n.Tpl.Resource == "" {
					t.Errorf("%s 联查失败（Hit=%v Resource=%q）", want, n.Hit, n.Tpl.Resource)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("摆放 %s 未装载", want)
		}
	}
}

func TestFilterRadiusAndLimit(t *testing.T) {
	npcs := []SceneNPC{
		{ID: "near", X: 0, Z: 10},
		{ID: "mid", X: 0, Z: 100},
		{ID: "far", X: 0, Z: 2000},
	}
	got := Filter(npcs, 0, 0, 150, 0)
	if len(got) != 2 || got[0].ID != "near" || got[1].ID != "mid" {
		t.Errorf("半径筛选 = %v", got)
	}
	got = Filter(npcs, 0, 0, 0, 2)
	if len(got) != 2 || got[0].ID != "near" {
		t.Errorf("上限截断（距离升序）= %v", got)
	}
	got = Filter(npcs, 0, 0, 0, 0)
	if len(got) != 3 {
		t.Errorf("不限 = %d 条", len(got))
	}
}
