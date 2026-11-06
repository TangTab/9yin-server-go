package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

// sceneNPCRegistry keeps the shared template tables in memory and lazily
// materializes one scene's creator catalog for each active world. Loading all
// 224k resolved spawns for every client would be wasteful and would mix NPCs
// from unrelated maps into one coordinate system.
type sceneNPCRegistry struct {
	tables   []*clientdata.NPCTemplateTable
	root     string
	schema   clientdata.Schema
	dirs     map[string]string
	mu       sync.Mutex
	catalogs map[string][]npcSpawn
	stats    map[string]npcCatalogStats
}

// normalizeClientScene repairs legacy persisted scene aliases before they are
// sent to the client. clone004_wl_1 is a gameplay-instance identifier, but
// the current client has no matching terrain directory; its installed terrain
// and base scene resources are named clone004_wl.
func normalizeClientScene(scene role.Scene) role.Scene {
	// scene09's logical configuration has a descriptive suffix, whereas the
	// installed terrain bundle is literally res/map/ter/scene09.
	if strings.EqualFold(scene.Config, `ini\scene\scene09_SaiBeiCaoYuan`) {
		scene.Resource = "scene09"
	}
	if strings.EqualFold(scene.Config, `ini\scene\scene07_MeiHuaLing`) {
		scene.Resource = "scene07"
	}
	if scene.Resource == "clone004_wl_1" || scene.Config == `ini\scene\clone004_wl_1` {
		scene.Config = `ini\scene\clone004_wl`
		scene.Resource = "clone004_wl"
	}
	return scene
}

func newSceneNPCRegistry(tableDir, creatorRoot string, schema clientdata.Schema) (*sceneNPCRegistry, error) {
	tables, err := loadNPCTemplateTables(tableDir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(creatorRoot)
	if err != nil {
		return nil, fmt.Errorf("list scene NPC creators %s: %w", creatorRoot, err)
	}
	dirs := make(map[string]string, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			dirs[strings.ToLower(entry.Name())] = entry.Name()
		}
	}
	return &sceneNPCRegistry{
		tables:   tables,
		root:     creatorRoot,
		schema:   schema,
		dirs:     dirs,
		catalogs: make(map[string][]npcSpawn),
		stats:    make(map[string]npcCatalogStats),
	}, nil
}

func (r *sceneNPCRegistry) catalogFor(scene role.Scene) ([]npcSpawn, npcCatalogStats, error) {
	folder, err := r.resolveFolder(scene)
	if err != nil {
		return nil, npcCatalogStats{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if catalog, ok := r.catalogs[folder]; ok {
		return catalog, r.stats[folder], nil
	}
	catalog, stats, err := loadNPCSceneSpawnsFromTables(r.tables, filepath.Join(r.root, folder), r.schema)
	if err != nil {
		return nil, stats, fmt.Errorf("load NPC catalog for scene %s (%s): %w", scene.Resource, folder, err)
	}
	r.catalogs[folder] = catalog
	r.stats[folder] = stats
	return catalog, stats, nil
}

func (r *sceneNPCRegistry) resolveFolder(scene role.Scene) (string, error) {
	if folder, found := preferredSceneConfigFolder[strings.ToLower(strings.TrimSpace(scene.Config))]; found {
		if actual, exists := r.dirs[folder]; exists {
			return actual, nil
		}
	}
	candidates := sceneResourceCandidates(scene)
	for _, candidate := range candidates {
		if folder, found := r.dirs[candidate]; found {
			return folder, nil
		}
		if folder, found := primarySceneFolder[candidate]; found {
			if actual, exists := r.dirs[folder]; exists {
				return actual, nil
			}
		}
	}
	for _, candidate := range candidates {
		var matches []string
		for key, folder := range r.dirs {
			if strings.HasPrefix(key, candidate+"_") {
				matches = append(matches, folder)
			}
		}
		if len(matches) == 1 {
			return matches[0], nil
		}
		if len(matches) > 1 {
			sort.Strings(matches)
			return "", fmt.Errorf("scene %q matches multiple NPC creator folders %s; add an explicit resource mapping", scene.Resource, strings.Join(matches, ", "))
		}
	}
	return "", fmt.Errorf("no NPC creator folder for scene resource=%q config=%q", scene.Resource, scene.Config)
}

func sceneResourceCandidates(scene role.Scene) []string {
	seen := make(map[string]struct{})
	var result []string
	add := func(value string) {
		value = strings.TrimSpace(strings.ToLower(value))
		value = strings.TrimPrefix(value, "scene_")
		if value == "" {
			return
		}
		if _, exists := seen[value]; !exists {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	add(scene.Resource)
	config := strings.ReplaceAll(scene.Config, "/", `\`)
	add(strings.TrimSuffix(filepath.Base(config), filepath.Ext(config)))
	return result
}

// The client resource ids for the main schools do not include their displayed
// school name, while the creator directories do. These are data aliases, not
// fallbacks: each pair is visible in the current extracted client tree.
var primarySceneFolder = map[string]string{
	// 幽云十六州的客户端资源目录为 scene01，creator 目录使用完整显示名。
	"scene01": "scene01_youyunshiliuzhou",
	// Compatibility for pre-normalization persisted values of the Yanmen Pass
	// instance id; its creator data is stored under the base resource name.
	"clone004_wl_1": "clone004_wl",
	"city01":        "city01_yanjing",
	"city02":        "city02_suzhou",
	"city03":        "city03_jinling",
	"city04":        "city04_luoyang",
	"city05":        "city05_chengdu",
	"school01":      "school01_jinyiwei",
	"school02":      "school02_gaibang",
	"school03":      "school03_junzitang",
	"school04":      "school04_jilegu",
	"school05":      "school05_tangmen",
	"school06":      "school06_emei",
	"school07":      "school07_wudang",
	"school08":      "school08_shaolin",
	"school09":      "school09_yihuagong",
	"school10":      "school10_taohuadao",
	"school11":      "school11_wugenmen",
	"school12":      "school12_wuxianjiao",
	"school13":      "school13_xuedaomen",
	"school14":      "school14_gumupai",
	"school15":      "school15_nianluoba",
	"school17":      "school17_huashanpai",
	"school18":      "school18_damo",
	"school19":      "school19_shenshuigong",
	"school20":      "school20_mingjiao",
	"school21":      "school21_shenjiying",
	"school22":      "school22_tianshan",
	"school23":      "school23_xingmiaoge",
	"school24":      "school24_kunlun",
	"school25":      "school25_tianya",
	"school26":      "school26_xingtianbao",
	"school28":      "school28_wanghui",
	"school30":      "school30_wuxu",
	"school31":      "school31_shenjihui",
}

// Some dungeon variants share one client terrain resource but have distinct
// creator data. Resolve those by their authoritative scene config before the
// generic resource name can select a different variant.
var preferredSceneConfigFolder = map[string]string{
	`ini\scene\clone002_yx`: "clone002_yx", // 龙门客栈 → client resource clone002
}
