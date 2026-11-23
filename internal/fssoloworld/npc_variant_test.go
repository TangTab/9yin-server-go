package fssoloworld

// npc_variant_test.go 变体目录权威解析测试（2026-11-23 修复回归）：
// SceneDirOf 字典序首匹配在同前缀多变体目录上选错（school01_fangong
// 抢在 school01_JinYiWei 前——八大门派/襄阳坡/塞外大漠切图 NPC 缺失根因），
// resolveSceneDir 三级解析（roleState.Scene 末段 > BornCJ Config 末段 >
// 字典序兜底）须选权威目录。

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// writeVariantRoot 假数据根：school01 双变体目录（真档布局小写
// jinyiwei——字典序 fangong 靠前抢先，BornCJ Config 则是混合大小写
// JinYiWei，顺带覆盖大小写归一）+ adv900 双变体（非 BornCJ 键域，
// 纯字典序兜底路径）。目录各放一个可联查 NPC（模板行 Resource 非空）。
func writeVariantRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("ini/Npc/NpcConfig/tpl.xml", `<?xml version="1.0" encoding="utf-8"?>
<Root><Object>
  <Property ID="JINYINPC" Resource="npc/jinyiwei" script="CommonNpc"/>
  <Property ID="FANGONGNPC" Resource="npc/fangong" script="CommonNpc"/>
  <Property ID="ADV_A" Resource="npc/adva" script="CommonNpc"/>
</Object></Root>`)
	write("ini/Npc/NpcTemplate.xml", `<Root></Root>`)
	write("ini/Scene/school01_jinyiwei/npc_creator.xml", `<Root><Object>
  <Property id="JINYINPC" File="commonnpc.xml" x="10" y="0" z="10" ay="0"/>
</Object></Root>`)
	write("ini/Scene/school01_fangong/npc_creator.xml", `<Root><Object>
  <Property id="FANGONGNPC" File="commonnpc.xml" x="20" y="0" z="20" ay="0"/>
</Object></Root>`)
	write("ini/Scene/adv900_alpha/npc_creator.xml", `<Root><Object>
  <Property id="ADV_A" File="commonnpc.xml" x="1" y="0" z="1" ay="0"/>
</Object></Root>`)
	write("ini/Scene/adv900_beta/npc_creator.xml", `<Root><Object>
  <Property id="ADV_A" File="commonnpc.xml" x="2" y="0" z="2" ay="0"/>
</Object></Root>`)
	return root
}

// resetNpcCaches 数据根/目录缓存复位（进程级缓存的测试缝）。
func resetNpcCaches() {
	npcfeed.ResetStore()
	npcCatalogMu.Lock()
	npcCatalogCache = map[string]*npcCatalog{}
	npcCatalogMu.Unlock()
}

func TestResolveSceneDirVariantAuthority(t *testing.T) {
	root := writeVariantRoot(t)
	t.Setenv("NINEYIN_DATA_ROOT", root)
	resetNpcCaches()
	t.Cleanup(resetNpcCaches)
	s := newNpcTestServer(t)

	// ① roleState.Scene 末段（权威 config 路径；混合大小写 vs 实目录
	// 小写——大小写不敏感归一到实际目录名）。
	if got := s.resolveSceneDir(`ini\scene\school01_JinYiWei`, "school01"); got != "school01_jinyiwei" {
		t.Errorf("Scene 路径解析 = %q, want school01_jinyiwei", got)
	}
	// ② 旧档 Scene 缺席 → 内嵌 BornCJ 表兜底（school01 → JinYiWei 正图）。
	if got := s.resolveSceneDir("", "school01"); got != "school01_jinyiwei" {
		t.Errorf("BornCJ 兜底解析 = %q, want school01_jinyiwei", got)
	}
	// ③ BornCJ 也没有的键域 → 字典序兜底（仅此路径受变体陷阱影响）。
	if got := s.resolveSceneDir("", "adv900"); got != "adv900_alpha" {
		t.Errorf("字典序兜底解析 = %q, want adv900_alpha", got)
	}
}

func TestSceneNPCEnterUsesAuthoritativeDir(t *testing.T) {
	root := writeVariantRoot(t)
	t.Setenv("NINEYIN_DATA_ROOT", root)
	resetNpcCaches()
	t.Cleanup(resetNpcCaches)
	s := newNpcTestServer(t)

	pl := &Player{Conn1: 7, State: PlOnline}
	pl.roleState = rolestate.RoleState{
		Scene:    `ini\scene\school01_JinYiWei`,
		Resource: "school01",
	}
	frames := s.sceneNPCEnter(pl, "school01", [4]float64{10, 0, 10, 0})
	pl.npcMu.Lock()
	scene := pl.npcScene
	pl.npcMu.Unlock()
	if scene != "school01_jinyiwei" {
		t.Fatalf("npcScene = %q, want school01_jinyiwei（错变体=切图 NPC 缺失根因）", scene)
	}
	if len(frames) == 0 {
		t.Fatal("权威目录 NPC 帧族为空")
	}
	catalog := s.sceneNPCCatalog("school01_JinYiWei")
	if len(catalog.spawns) != 1 || catalog.spawns[0].ConfigID != "JINYINPC" {
		t.Fatalf("权威目录物化集异常: %+v", catalog.spawns)
	}
	// 反证：裸场景键前缀形态（旧路径）确实命中 fangong——权威链存在的意义。
	if leaked := s.sceneNPCCatalog("school01"); len(leaked.spawns) != 1 || leaked.spawns[0].ConfigID != "FANGONGNPC" {
		t.Fatalf("字典序兜底应命中 fangong（反证断言）: dir=%q spawns=%+v", leaked.dir, leaked.spawns)
	}
}
