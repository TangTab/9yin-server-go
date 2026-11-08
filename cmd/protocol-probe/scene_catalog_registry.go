package main

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

// sceneNPCRegistry 将共享的模板表保存在内存中，并为每个活动世界惰性
// 物化单个场景的 creator 目录。为每个客户端加载全部 22.4 万个已解析
// 出生点既浪费，又会把无关地图的 NPC 混进同一个坐标系。
type sceneNPCRegistry struct {
	tables   []*clientdata.NPCTemplateTable
	root     string
	dirs     map[string]string
	mu       sync.Mutex
	catalogs map[string][]npcSpawn
	stats    map[string]npcCatalogStats
}

// normalizeClientScene 在旧持久化场景别名发送给客户端之前将其修复。
// clone004_wl_1 是玩法副本标识，但当前客户端没有对应的地形目录；它
// 安装的地形和基础场景资源名为 clone004_wl。
func normalizeClientScene(scene role.Scene) role.Scene {
	// scene09 的逻辑配置带有一个描述性后缀，而已安装的地形包则直接叫
	// res/map/ter/scene09。
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

func newSceneNPCRegistry(tableDir, creatorRoot string) (*sceneNPCRegistry, error) {
	tables, err := loadNPCTemplateTables(tableDir)
	if err != nil {
		return nil, err
	}
	// 实力包表（staticprop.ini）随 NPC 目录一并装载：守方战斗数值推导用
	//（2026-11-08 NPC 属性块迁移批次）。此前挂在 loadChengduNPCSpawns
	//（无人调用）里，被链接器死代码消除整链裁掉——装载从未执行，
	// deriveNPCCombatStats 恒按零值表推导。表缺失不阻断——推导按零值。
	installedStaticPropTable = loadStaticPropTable()
	if installedStaticPropTable != nil {
		log.Printf("NPC 实力包表 ready packages=%d path=%s", len(installedStaticPropTable), staticPropPath())
	}
	entries, err := readResourceDirEntries(creatorRoot)
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
	catalog, stats, err := loadNPCSceneSpawnsFromTables(r.tables, filepath.Join(r.root, folder))
	if err != nil {
		return nil, stats, fmt.Errorf("load NPC catalog for scene %s (%s): %w", scene.Resource, folder, err)
	}
	// 官方 mappath 巡逻路线：PathId 绑定 + 就近吸附（目录缓存内一次完成，
	// epoch 进缓存 → 场景运行时重建后 NPC 从时间表当前处续走）。
	// mappath 表在 npc_creator 的兄弟目录（share/creator/mappath）。
	bindPatrolRoutes(catalog, loadPatrolRoutes(filepath.Join(filepath.Dir(r.root), "mappath"), folder), time.Now().UnixMilli())
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

// 主要门派的客户端资源 ID 不含显示的门派名，而 creator 目录包含。这些
// 是数据别名而非兜底：每一对映射都能在当前解包的客户端目录树中
// 看到。
var primarySceneFolder = map[string]string{
	// 幽云十六州的客户端资源目录为 scene01，creator 目录使用完整显示名。
	"scene01": "scene01_youyunshiliuzhou",
	// 兼容雁门关副本 ID 在规范化之前持久化的旧值；其 creator 数据存放
	// 在基础资源名之下。
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

// 某些副本变体共用一个客户端地形资源，但拥有不同的 creator 数据。要
// 在通用资源名可能选中其他变体之前，先按其权威场景配置解析这些
// 变体。
var preferredSceneConfigFolder = map[string]string{
	`ini\scene\clone002_yx`: "clone002_yx", // 龙门客栈 → 客户端资源 clone002
}
