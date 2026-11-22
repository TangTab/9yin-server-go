package npcdata

// scenes.ini 全局场景索引表测试：纯解析（段号→Config 基名 resource 键，
// 前缀截断/首现绑定）+ 仓内真档锚点（[3]=born03、[14]=city02、[366]=scene22
// ——与外服 OpenMapRec 行 11 bit14 互证）。

import "testing"

func TestParseSceneIndexTable(t *testing.T) {
	text := "// 注释\r\n[1]\r\nMember=1\r\nConfig=ini\\scene\\born04_QianDengZhen\r\n" +
		"[2]\r\nConfig=ini/scene/city02_SuZhou\r\n[2]\r\nConfig=ini\\scene\\city02_SuZhou2\r\n" +
		"[366]\r\nConfig=ini\\scene\\scene22\r\n[400]\r\nNotConfig=1\r\n"
	table := ParseSceneIndexTable(text)
	for resource, want := range map[string]uint32{
		"born04":  1,   // born 前缀截到 '_'
		"city02":  2,   // city 前缀截断；同键多段取首现
		"scene22": 366, // 无 '_' 全基名
	} {
		if got, ok := table[resource]; !ok || got != want {
			t.Fatalf("%s=%d,%v want %d", resource, got, ok, want)
		}
	}
	if _, ok := table["notconfig=1"]; ok {
		t.Fatal("非 Config 键不应入表")
	}
	if len(table) != 3 {
		t.Fatalf("表大小=%d, want 3", len(table))
	}
}

func TestSceneGlobalIndexFromTestdata(t *testing.T) {
	for resource, want := range map[string]uint32{
		"born03": 3, "Born03 ": 3, // 大小写/空白归一
		"city02":  14,
		"scene22": 366, // 外服位图锚点
	} {
		got, ok := SceneGlobalIndex(testRoot, resource)
		if !ok || got != want {
			t.Fatalf("%s=%d,%v want %d", resource, got, ok, want)
		}
	}
	if _, ok := SceneGlobalIndex(testRoot, "no_such_scene"); ok {
		t.Fatal("未收录场景应 miss")
	}
	if _, ok := SceneGlobalIndex(testRoot, ""); ok {
		t.Fatal("空 resource 应 miss")
	}
}

func TestSceneResourceOfBase(t *testing.T) {
	for base, want := range map[string]string{
		"born03_YanYuZhuang": "born03",
		"city02_SuZhou":      "city02",
		"school01_Emei":      "school01",
		"scene22":            "scene22",
		"WestZQ":             "westzq",
	} {
		if got := SceneResourceOfBase(base); got != want {
			t.Fatalf("%s=%q want %q", base, got, want)
		}
	}
}
