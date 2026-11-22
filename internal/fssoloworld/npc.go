package fssoloworld

// npc.go 场景 NPC 下发薄壳（2026-11-22 重写）：npcfeed 目录缓存 +
// 半径/上限过滤 + 帧生成。数据链与帧配方见 internal/npcfeed 包注释及
// docs/official/NPC服务端XML构建链与黑金样板.md。

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
)

// npcPushEnabled NINEYIN_NPC_PUSH=off 一键关停（方言演进期保护，
// 渲染异常时归因用）。
func npcPushEnabled() bool { return os.Getenv("NINEYIN_NPC_PUSH") != "off" }

// npcAs0x28Enabled 物化帧形态：true=0x28 mode=0x46 官方言（默认，
// npc.pcapng 罗塞塔真机验证形态）；false=裸 0x0D 大档（判别）。
func npcAs0x28Enabled() bool { return os.Getenv("NINEYIN_NPC_0X28") != "off" }

// npcCatalog 场景物化目录（联查命中集，进程级缓存）。
type npcCatalog struct {
	spawns []npcfeed.Spawn
	stats  npcfeed.Stats
}

var (
	npcCatalogMu    sync.Mutex
	npcCatalogCache = map[string]*npcCatalog{}
)

// sceneNPCFrames 场景 NPC 物化帧族（装载+联查+ident+过滤+编码，缓存）。
// 装载失败返回空——NPC 是增强下发，不阻塞进场。
func (s *Server) sceneNPCFrames(scene string, origin [4]float64) [][]byte {
	if !npcPushEnabled() {
		return nil
	}
	catalog := s.sceneNPCCatalog(scene)
	spawns := catalog.spawns
	if radius := npcRadius(); radius > 0 {
		spawns = npcfeed.FilterByRadius(spawns, origin[0], origin[2], radius, npcLimit())
	} else if limit := npcLimit(); limit > 0 && len(spawns) > limit {
		spawns = spawns[:limit]
	}
	spawns = applyResourceOverride(spawns)
	frames, err := npcfeed.Frames(spawns, npcAs0x28Enabled())
	if err != nil {
		s.trace("npc frames: %v", err)
		return nil
	}
	return frames
}

// applyResourceOverride NINEYIN_NPC_RESOURCE_OVERRIDE 判别实验开关：
// 强制全 NPC 用同一模型串（模板层克隆改写，不污染目录缓存）。判别
// 「客户端模型库缺失（缺模型的 NPC 不渲染/缺动画呆立）」vs「下发建档
// 问题」——全同模型后若全城显示且会动，坐实前者（2433 线实验设计）。
func applyResourceOverride(spawns []npcfeed.Spawn) []npcfeed.Spawn {
	override := strings.TrimSpace(os.Getenv("NINEYIN_NPC_RESOURCE_OVERRIDE"))
	if override == "" {
		return spawns
	}
	patched := make([]npcfeed.Spawn, len(spawns))
	copy(patched, spawns)
	for i := range patched {
		tpl := make(map[string]string, len(patched[i].Template)+1)
		for k, v := range patched[i].Template {
			tpl[k] = v
		}
		tpl["Resource"] = override
		tpl["Material"] = "" // 防 Material 分支干扰
		patched[i].Template = tpl
	}
	return patched
}

// sceneNPCCatalog 场景目录（缓存）。
func (s *Server) sceneNPCCatalog(scene string) *npcCatalog {
	npcCatalogMu.Lock()
	defer npcCatalogMu.Unlock()
	if cached, ok := npcCatalogCache[scene]; ok {
		return cached
	}
	catalog := &npcCatalog{}
	root := npcfeed.ResolveRoot()
	if root == "" {
		s.trace("npc catalog: 数据根缺席（NINEYIN_DATA_ROOT / Res / testdata），场景 %s NPC 不下发", scene)
		npcCatalogCache[scene] = catalog
		return catalog
	}
	spawns, stats, err := npcfeed.BuildCatalog(root, scene)
	if err != nil {
		s.trace("npc catalog: %v", err)
		npcCatalogCache[scene] = catalog
		return catalog
	}
	catalog.spawns, catalog.stats = spawns, stats
	s.trace("npc catalog: 场景 %s 摆放 %d 联查 %d 战斗命中 %d 闸拦 %d → 物化 %d 条",
		scene, stats.Placements, stats.Resolved, stats.CombatHit, stats.Skipped, len(spawns))
	for _, sample := range stats.Samples {
		s.trace("npc catalog: 样本 %s", sample)
	}
	npcCatalogCache[scene] = catalog
	return catalog
}

// npcRadius NINEYIN_NPC_RADIUS 半径过滤（<=0 全量；默认全量——当前
// 真机验证口径）。
func npcRadius() float64 { return envFloat("NINEYIN_NPC_RADIUS", 0) }

// npcLimit NINEYIN_NPC_LIMIT 截断上限（<=0 不截断）。
func npcLimit() int { return int(envFloat("NINEYIN_NPC_LIMIT", 0)) }

// envFloat env 数值（缺席/非法回落缺省）。
func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
			return f
		}
	}
	return def
}
