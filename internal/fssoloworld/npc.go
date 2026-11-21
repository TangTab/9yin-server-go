package fssoloworld

// npc.go 场景 NPC 物化目录（entergen m229 c130 族数据源）：场景摆放
// （npc_creator.xml）+ NpcConfig 模板联查 + 出生点筛选 → NpcEntry 集，
// 进程级缓存（场景键 → 已分配 ident 的条目集，跨进场稳定——点击路由
// 期的前提）。方言定版见 docs/official/外服NPCm229物化方言定版.md。

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcdata"
)

const (
	npcDefaultRadius = 500.0 // 出生点半径（born03 密度：R300 已 745 条）
	npcDefaultLimit  = 200   // 总量上限（≤0x0B idx1468 MaxVisuals=230）
)

// npcPushEnabled NINEYIN_NPC_PUSH=off 一键关停（方言演进期保护，
// 渲染异常时归因用）。
func npcPushEnabled() bool { return os.Getenv("NINEYIN_NPC_PUSH") != "off" }

var (
	npcCatalogMu    sync.Mutex
	npcCatalogCache = map[string][]entergen.NpcEntry{}
)

// sceneNPCCatalog 场景 NPC 条目集（装载+联查+筛选+ident 分配，缓存）。
// 装载失败（数据根缺席/真档缺失）返回空集——NPC 是增强下发，不阻塞进场。
func (s *Server) sceneNPCCatalog(scene string, origin [4]float64) []entergen.NpcEntry {
	if !npcPushEnabled() {
		return nil
	}
	npcCatalogMu.Lock()
	defer npcCatalogMu.Unlock()
	if cached, ok := npcCatalogCache[scene]; ok {
		return cached
	}
	entries := s.loadNPCCatalog(scene, origin)
	npcCatalogCache[scene] = entries
	return entries
}

// loadNPCCatalog 装载链（无锁；sceneNPCCatalog 持锁调用）。
func (s *Server) loadNPCCatalog(scene string, origin [4]float64) []entergen.NpcEntry {
	root := resolveDataRoot()
	if root == "" {
		s.trace("npc catalog: data root 缺席（NINEYIN_DATA_ROOT 或 testdata/），场景 %s NPC 不下发", scene)
		return nil
	}
	sceneDir := findSceneDir(root, scene)
	if sceneDir == "" {
		s.trace("npc catalog: 场景 %s 无摆放目录", scene)
		return nil
	}
	npcs, err := npcdata.LoadSceneNPCs(root, sceneDir)
	if err != nil {
		s.trace("npc catalog: %v", err)
		return nil
	}
	templates, err := npcdata.LoadTemplates(root)
	if err != nil {
		s.trace("npc catalog: %v", err)
		return nil
	}
	hit := npcdata.Join(npcs, templates)
	radius := envFloat("NINEYIN_NPC_RADIUS", npcDefaultRadius)
	limit := int(envFloat("NINEYIN_NPC_LIMIT", float64(npcDefaultLimit)))
	kept := npcdata.Filter(npcs, origin[0], origin[2], radius, limit)
	// ident 分配（场景内 1 起确定性；0x03 段 obj/0x20xx 段 own——
	// 与玩家 0x11 段及 owner=1 玩家化身位隔离）。
	entries := make([]entergen.NpcEntry, 0, len(kept))
	for i, npc := range kept {
		if !npc.Hit {
			continue
		}
		idx := i + 1
		entries = append(entries, entergen.NpcEntry{
			ObjectID:    entergen.NpcObjectID(idx),
			OwnerID:     entergen.NpcOwnerID(idx),
			Resource:    npc.Tpl.Resource,
			X:           float32(npc.X),
			Y:           float32(npc.Y),
			Z:           float32(npc.Z),
			Yaw:         entergen.WrapYaw(float32(npc.AY)),
			OrientText:  entergen.OrientTriplet(float32(npc.AY)),
			Scale:       npc.Tpl.Scale,
			Layer:       302, // golden 众数（301/302 双发之谜待真机定案）
			Group:       0,
			Name:        npc.Tpl.Name, // 人物名（物件多空名，行为待真机验证）
		})
	}
	s.trace("npc catalog: 场景 %s 摆放 %d 联查 %d → 半径 %.0f 上限 %d 取 %d 条",
		scene, len(npcs), hit, radius, limit, len(entries))
	return entries
}

// resolveDataRoot 数据根定位：env NINEYIN_DATA_ROOT > 相对 cwd 探测
// <dir>/ini/Npc/NpcConfig——覆盖源码树（仓库根/cmd）、bin\\ 部署
// （目标根放 testdata\\ini 真档）、Go 测试（internal/<pkg>）三种工作
// 目录。真档 27MB 不随二进制内嵌（deploy 不发 ini），缺席时 NPC 静默
// 不下发（增强下发，不阻塞进场）。
func resolveDataRoot() string {
	if v := os.Getenv("NINEYIN_DATA_ROOT"); v != "" {
		return v
	}
	for _, dir := range []string{".", "..", "../..", "testdata", "../testdata", "../../testdata"} {
		if st, err := os.Stat(dir + "/ini/Npc/NpcConfig"); err == nil && st.IsDir() {
			return dir
		}
	}
	return ""
}

// findSceneDir 场景键（born03）→ 摆放目录名（born03_yanyuzhuang）：
// ini/Scene 下前缀 "<scene>_" 或同名目录（大小写不敏感——rolestate
// birth 匹配同口径）。
func findSceneDir(root, scene string) string {
	entries, err := os.ReadDir(root + "/ini/Scene")
	if err != nil {
		return ""
	}
	prefix := scene + "_"
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.EqualFold(name, scene) ||
			(len(name) > len(prefix) && strings.EqualFold(name[:len(prefix)], prefix)) {
			return name
		}
	}
	return ""
}

// envFloat env 数值（缺席/非法回落缺省）。
func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
			return f
		}
	}
	return def
}
