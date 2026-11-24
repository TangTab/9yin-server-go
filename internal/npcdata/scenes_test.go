package npcdata

// scenes.ini 全局场景索引表测试：纯解析（段号→Config 基名 resource 键，
// 前缀截断/首现绑定）+ 仓内真档锚点（[3]=born03、[14]=city02、[366]=scene22
// ——与外服 OpenMapRec 行 11 bit14 互证）。

import (
	"path/filepath"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// testRoot 仓内 testdata 根（含 ini/scenes.ini；npcdata 包 cwd 探测）。
var testRoot = func() string {
	for _, dir := range []string{"../../testdata", "../../../testdata", "testdata"} {
		if _, err := res.Stat(filepath.Join(dir, "ini", "scenes.ini")); err == nil {
			return dir
		}
	}
	return ""
}()

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

func TestParseSceneIndexTablesDir(t *testing.T) {
	text := "// 注释\r\n[4]\r\nConfig=ini\\scene\\born01_MoHeZhen\r\n" +
		"[300]\r\nConfig=ini\\scene\\born_single_01\r\n[302]\r\nConfig=ini\\scene\\born_single_01\r\n" +
		"[14]\r\nConfig=ini/scene/city02_SuZhou\r\n[400]\r\nNotConfig=1\r\n"
	tables := parseSceneIndexTables(text)
	for dir, want := range map[string]uint32{
		"born01_mohezhen":  4,   // 基名全名小写
		"born_single_01":   300, // 同 Config 多节取首现
		"city02_suzhou":    14,  // 正斜杠归一
		"born01_mohezhena": 0,   // 不应命中（前缀相同不算）
	} {
		if got, ok := tables.byDir[dir]; ok != (want != 0) || got != want {
			t.Fatalf("%s=%d,%v want %d", dir, got, ok, want)
		}
	}
	if len(tables.byDir) != 3 {
		t.Fatalf("基名表大小=%d, want 3", len(tables.byDir))
	}
	// resource 表不受影响：born_single_01 截到 'born'。
	if got := tables.byResource["born"]; got != 300 {
		t.Fatalf("born=%d want 300", got)
	}
}

func TestSceneIndexOfDirFromTestdata(t *testing.T) {
	// 官服抓包锚点：obj 高字节 = 段号低字节（漠河镇 0x04、姑苏 0x0E、
	// born_single_01 节号 317 的低字节 0x3D）。
	for dir, want := range map[string]uint32{
		"born03_yanyuzhuang": 3,
		"Born03_YanYuZhuang": 3, // 大小写/空白归一
		" city02_SuZhou ":    14,
		"born01_MoHeZhen":    4,
		"born_single_01":     300, // 每 member 一节，取首现
		"school01_JinYiWei":  5,
		"scene22":            366,
		"school01_fangong":   5, // scenes.ini 未收录的变体目录回落 resource 键
	} {
		got, ok := SceneIndexOfDir(testRoot, dir)
		if !ok || got != want {
			t.Fatalf("%s=%d,%v want %d", dir, got, ok, want)
		}
	}
	for _, dir := range []string{"no_such_dir", ""} {
		if _, ok := SceneIndexOfDir(testRoot, dir); ok {
			t.Fatalf("%q 应 miss", dir)
		}
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
