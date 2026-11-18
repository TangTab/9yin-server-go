package config

import "testing"

func TestLoadScenes(t *testing.T) {
	tbl, err := LoadScenes("../../testdata/ini/scenes.xml")
	if err != nil {
		t.Fatal(err)
	}
	if tbl.Len() < 20 {
		t.Fatalf("场景数过少：%d", tbl.Len())
	}

	main := tbl.Get("1")
	if main == nil || main.Name != "蒸汽大陆" || main.Script != "scene" || main.Clonable {
		t.Fatalf("主城场景字段错误：%+v", main)
	}
	if len(main.Relive) == 0 {
		t.Fatal("蒸汽大陆应有复活点")
	}

	fb := tbl.Get("101")
	if fb == nil || !fb.Clonable || fb.Script != "CloneScene" {
		t.Fatalf("流浪群岛应为副本场景：%+v", fb)
	}
	if v := tbl.Get("202_1"); v == nil { // 变体 ID（字符串写法）
		t.Fatal("202_1 变体场景应存在")
	}

	// 原版 ReliveLocation 存在 "A,,B" 空段写法，应被过滤
	for _, s := range tbl.All() {
		for _, r := range s.Relive {
			if r == "" {
				t.Fatalf("场景 %s 复活点含空段", s.ID)
			}
		}
	}
}

func TestLoadLogicClass(t *testing.T) {
	tree, err := LoadLogicClass("../../testdata/ini/logic_class.xml")
	if err != nil {
		t.Fatal(err)
	}
	scene := tree.Get("scene")
	if scene == nil || scene.Type != "TYPE_SCENE" {
		t.Fatalf("scene 基类缺失或类型错误：%+v", scene)
	}
	if !tree.IsA("CloneScene", "scene") {
		t.Fatal("CloneScene 应继承 scene")
	}
	if !tree.IsA("CastleNpc", "npc") { // 深层链：CastleNpc→FlyNpc→FunctionNpc→VisibleNpc→npc
		t.Fatal("CastleNpc 应继承 npc")
	}
	if tree.IsA("scene", "npc") {
		t.Fatal("scene 与 npc 无继承关系")
	}
	chain := tree.RootsOf("CloneScene")
	if len(chain) != 2 || chain[0] != "CloneScene" || chain[1] != "scene" {
		t.Fatalf("CloneScene 继承链错误：%v", chain)
	}
}
