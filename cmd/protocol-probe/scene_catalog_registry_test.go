package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

func TestSceneCreatorManifestSkipsEventTriggerAndAbsentOptionalCreator(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.ini"), []byte("[file]\n0=CommonNpc.xml\n1=EventTrigger.xml\n2=AttackNpc_Creator.xml\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"CommonNpc.xml", "EventTrigger.xml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("<object/>"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := sceneCreatorPaths(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || filepath.Base(paths[0]) != "CommonNpc.xml" {
		t.Fatalf("creator paths=%q", paths)
	}
}

func TestSceneCatalogFolderResolution(t *testing.T) {
	registry := &sceneNPCRegistry{dirs: map[string]string{
		"city05_chengdu":           "city05_chengdu",
		"school07_wudang":          "school07_wudang",
		"scene01_youyunshiliuzhou": "scene01_youyunshiliuzhou",
		"clone001":                 "clone001",
		"clone002":                 "clone002",
		"clone002_yx":              "clone002_yx",
	}}
	for _, test := range []struct {
		name  string
		scene role.Scene
		want  string
	}{
		{"city alias", role.Scene{Resource: "city05", Config: `ini\scene\scene_city05`}, "city05_chengdu"},
		{"school alias", role.Scene{Resource: "school07", Config: `ini\scene\school07_WuDang`}, "school07_wudang"},
		{"unique prefix", role.Scene{Resource: "scene01"}, "scene01_youyunshiliuzhou"},
		{"exact clone", role.Scene{Resource: "clone001"}, "clone001"},
		{"longmen config overrides generic resource", role.Scene{Config: `ini\scene\clone002_yx`, Resource: "clone002"}, "clone002_yx"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := registry.resolveFolder(test.scene)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("folder=%q, want %q", got, test.want)
			}
		})
	}
}

func TestSceneCatalogFolderRejectsAmbiguousPrefix(t *testing.T) {
	registry := &sceneNPCRegistry{dirs: map[string]string{
		"scene04_xiangyangpo":      "scene04_xiangyangpo",
		"scene04_2_xiangyangpobei": "scene04_2_xiangyangpobei",
	}}
	if _, err := registry.resolveFolder(role.Scene{Resource: "scene04"}); err == nil {
		t.Fatal("ambiguous scene04 resolved without explicit mapping")
	}
}

func TestNormalizeClientSceneUsesInstalledYanmenPassResource(t *testing.T) {
	got := normalizeClientScene(role.Scene{Config: `ini\scene\clone004_wl_1`, Resource: "clone004_wl_1"})
	if got.Config != `ini\scene\clone004_wl` || got.Resource != "clone004_wl" {
		t.Fatalf("normalized scene=%+v", got)
	}
}

func TestCurrentCitySceneRegistryMaterializesModernCreatorCatalog(t *testing.T) {
	if _, err := os.Stat(defaultNPCTableDir); err != nil {
		t.Skipf("modern client data unavailable: %v", err)
	}
	registry, err := newSceneNPCRegistry(defaultNPCTableDir, filepath.Join(defaultModernShareRoot, "creator", "npc_creator"), clientdata.VisibleNPCModernV1())
	if err != nil {
		t.Fatal(err)
	}
	catalog, stats, err := registry.catalogFor(role.Scene{Config: `ini\scene\scene_city05`, Resource: "city05"})
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) == 0 || stats.Resolved == 0 {
		t.Fatalf("city05 catalog=%d resolved=%d", len(catalog), stats.Resolved)
	}
}

func TestShenJiHuiCatalogSkipsMissingOptionalCreators(t *testing.T) {
	if _, err := os.Stat(defaultNPCTableDir); err != nil {
		t.Skipf("modern client data unavailable: %v", err)
	}
	registry, err := newSceneNPCRegistry(defaultNPCTableDir, filepath.Join(defaultModernShareRoot, "creator", "npc_creator"), clientdata.VisibleNPCModernV1())
	if err != nil {
		t.Fatal(err)
	}
	catalog, stats, err := registry.catalogFor(role.Scene{Config: `ini\scene\school31_shenjihui`, Resource: "school31"})
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) == 0 || stats.Resolved == 0 {
		t.Fatalf("shenjihui catalog=%d stats=%+v", len(catalog), stats)
	}
}

func TestYanYuZhuangCatalogSkipsEmptyZeroAmountCreatorPlaceholder(t *testing.T) {
	if _, err := os.Stat(defaultNPCTableDir); err != nil {
		t.Skipf("modern client data unavailable: %v", err)
	}
	registry, err := newSceneNPCRegistry(defaultNPCTableDir, filepath.Join(defaultModernShareRoot, "creator", "npc_creator"), clientdata.VisibleNPCModernV1())
	if err != nil {
		t.Fatal(err)
	}
	catalog, stats, err := registry.catalogFor(role.Scene{Config: `ini\scene\born03_YanYuZhuang`, Resource: "born03"})
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) == 0 || stats.Resolved == 0 {
		t.Fatalf("born03 catalog=%d stats=%+v", len(catalog), stats)
	}
}
