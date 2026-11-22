package fssoloworld

// npc.go 场景 NPC 物化目录：场景摆放（npc_creator.xml）+ NpcConfig 模板
// 联查 → 全量物化集（0x28 mode=0x46 官方言，真机验证成功形态），进程
// 级缓存。数据链与方言定版见 docs/official/外服NPCm229物化方言定版.md。

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcdata"
)

// npcPushEnabled NINEYIN_NPC_PUSH=off 一键关停（方言演进期保护，
// 渲染异常时归因用）。
func npcPushEnabled() bool { return os.Getenv("NINEYIN_NPC_PUSH") != "off" }

// npcMark229Enabled m229 设施族开关（默认关——fork 系客户端上是
// EscortRec 记录帧不渲染；外服家族客户端接入时 NINEYIN_NPC_MARK_229=1
// 对照）。
func npcMark229Enabled() bool { return os.Getenv("NINEYIN_NPC_MARK_229") == "1" }

func npc0DEnabled() bool { return os.Getenv("NINEYIN_NPC_0D") != "off" }

// npcCatalog 场景物化条目集（objects0D 为全量联查命中集）。
type npcCatalog struct {
	objects0D []entergen.NpcObject0D
}

var (
	npcCatalogMu    sync.Mutex
	npcCatalogCache = map[string]*npcCatalog{}
)

// sceneNPCObjects 全量物化集（装载+联查+ident，缓存）。
func (s *Server) sceneNPCObjects(scene string, origin [4]float64) []entergen.NpcObject0D {
	if !npcPushEnabled() || !npc0DEnabled() {
		return nil
	}
	return s.sceneNPCScene(scene, origin).objects0D
}

// sceneNPCMarks m229 设施族：已退役恒空（2026-11-22 终局：fork 系
// 客户端把 m229 当 EscortRec 记录帧不渲染）。
func (s *Server) sceneNPCMarks(scene string, origin [4]float64) []entergen.NpcEntry {
	return nil
}

// sceneNPCScene 场景目录（缓存）。装载失败返回空目录——NPC 是增强
// 下发，不阻塞进场。
func (s *Server) sceneNPCScene(scene string, origin [4]float64) *npcCatalog {
	if !npcPushEnabled() {
		return &npcCatalog{}
	}
	npcCatalogMu.Lock()
	defer npcCatalogMu.Unlock()
	if cached, ok := npcCatalogCache[scene]; ok {
		return cached
	}
	catalog := s.loadNPCCatalog(scene, origin)
	npcCatalogCache[scene] = catalog
	return catalog
}

// loadNPCCatalog 装载链（无锁；sceneNPCScene 持锁调用）。
func (s *Server) loadNPCCatalog(scene string, origin [4]float64) *npcCatalog {
	empty := &npcCatalog{}
	root := resolveDataRoot()
	if root == "" {
		s.trace("npc catalog: data root 缺席（NINEYIN_DATA_ROOT / Res / testdata），场景 %s NPC 不下发", scene)
		return empty
	}
	sceneDir := findSceneDir(root, scene)
	if sceneDir == "" {
		s.trace("npc catalog: 场景 %s 无摆放目录", scene)
		return empty
	}
	npcs, err := npcdata.LoadSceneNPCs(root, sceneDir)
	if err != nil {
		s.trace("npc catalog: %v", err)
		return empty
	}
	templates, err := npcdata.LoadTemplates(root)
	if err != nil {
		s.trace("npc catalog: %v", err)
		return empty
	}
	hit := npcdata.Join(npcs, templates)
	catalog := &npcCatalog{
		objects0D: make([]entergen.NpcObject0D, 0, hit),
	}
	ident := 0
	for _, npc := range npcs {
		if !npc.Hit {
			continue
		}
		ident++
		tpl := npc.Tpl.Raw
		if override := os.Getenv("NINEYIN_NPC_RESOURCE_OVERRIDE"); override != "" {
			tpl = make(map[string]string, len(tpl)+1)
			for k, v := range npc.Tpl.Raw {
				tpl[k] = v
			}
			tpl["Resource"] = override
		}
		catalog.objects0D = append(catalog.objects0D, entergen.NpcObject0D{
			ObjectID: entergen.NpcForkObjectID(ident),
			OwnerID:  entergen.NpcForkOwnerID(entergen.NpcForkObjectID(ident)),
			X:        float32(npc.X),
			Y:        float32(npc.Y),
			Z:        float32(npc.Z),
			Orient:   float32(npc.AY),
			ConfigID: npc.ID,
			Template: tpl,
		})
	}
	s.trace("npc catalog: 场景 %s 摆放 %d 联查 %d → 全量物化集 %d 条",
		scene, len(npcs), hit, len(catalog.objects0D))
	return catalog
}

// resolveDataRoot 数据根定位：env NINEYIN_DATA_ROOT > 相对 cwd 探测
// <dir>/ini/Npc/NpcConfig——两级语义：
//   - 正式：$root Res（官方服务端 ini 布局，与 testdata/ini 同构）；
//   - 测试/开发：仓内 testdata。
func resolveDataRoot() string {
	if v := os.Getenv("NINEYIN_DATA_ROOT"); v != "" {
		return v
	}
	for _, dir := range []string{
		"Res", "../Res", "../../Res", // 正式：$root Res
		".", "..", "../..", // 裸 ini 布局兼容
		"testdata", "../testdata", "../../testdata", // 测试：仓内真档
	} {
		if st, err := os.Stat(dir + "/ini/Npc/NpcConfig"); err == nil && st.IsDir() {
			return dir
		}
	}
	return ""
}

// findSceneDir 场景键（born03）→ 摆放目录名（born03_yanyuzhuang）：
// ini/Scene 下前缀 "<scene>_" 或同名目录（大小写不敏感）。
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
