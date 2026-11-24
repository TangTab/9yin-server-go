package npcfeed

// source.go NPC 数据来源：客户端 share 树直读，或 tools/datagen npc 构建的
// 服务端 XML 产物。2026-11-25 起两条路并存，逐域按数据根现状选：
//
//	域      share 源（客户端设计态表）                 XML 产物
//	摆放    share/creator/npc_creator/<场景>/*.xml    ini/Scene/<场景>/npc_creator.xml
//	模板    share/npc/npcconfig/**/*.txt              ini/Npc/NpcConfig/**/*.xml
//	战斗    share/task/npctemplate*.tab               ini/Npc/NpcTemplate.xml
//
// 开关 NINEYIN_DATA_SOURCE：share | xml；缺省 auto——该域的 share 源在数据根
// 里就直读，不在就读 XML 产物（不带 share 树的旧数据根行为不变）；auto 下
// share 树看起来比产物旧（摆放树缺了产物已有的场景）也读产物，并在启动日志
// 里说明（sharedata.ShareOlderThanXML / StaleShareNote）。显式 xml 是 A/B
// 基线：两条路读同一份数据根时必须给出逐行相同的结果
// （source_parity_test.go）。
//
// 直读的意义（docs/official/客户端share直读可行性分析.md）：XML 产物是
// 客户端表的有损包装（模板表 57% 的列位被丢弃），且会随客户端树更新而
// 过期；直读拿到的是客户端原表，少一趟「改映射 → 重跑构建器 → 并层 →
// 重启」。键名口径不变：直读给出的属性名与 XML 产物一致（含改名与 table@
// 前缀剥离，见 load_share.go），物化帧 / 巡逻 / 商店 / 任务这些消费方不用改。

import (
	"path/filepath"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/internal/sharedata"
)

const (
	// shareCreatorDir 摆放源：客户端编辑器导出的场景 creator XML 树。
	shareCreatorDir = sharedata.CreatorDir
	// shareNPCConfigDir 模板源：NPC 模板 TAB 表树（每张表 3 行头）。
	shareNPCConfigDir = sharedata.NPCConfigDir
	// shareNpcTemplateFile 战斗模板主档（npctemplate.tab；补充档按同目录 glob 并入）。
	shareNpcTemplateFile = sharedata.NpcTemplateFile
)

// PlacementSource 摆放域的来源。
func PlacementSource(root string) sharedata.Source {
	return sharedata.EffectiveSource(root, shareCreatorDir)
}

// TemplateSource 模板域（NPC 模板）的来源。
func TemplateSource(root string) sharedata.Source {
	return sharedata.EffectiveSource(root, shareNPCConfigDir)
}

// CombatSource 战斗模板域的来源。
func CombatSource(root string) sharedata.Source {
	return sharedata.EffectiveSource(root, shareNpcTemplateFile)
}

// sceneRootDir 摆放目录树的根：share 模式 share/creator/npc_creator，xml
// 模式 ini/Scene。两棵树的场景目录名相同（born03_yanyuzhuang…）。
func sceneRootDir(root string) string {
	if PlacementSource(root) == sharedata.SourceShare {
		return filepath.Join(root, filepath.FromSlash(shareCreatorDir))
	}
	return filepath.Join(root, "ini", "Scene")
}

// SceneDirs 摆放场景目录名清单（字典序；诊断 / 枚举面）。
func SceneDirs(root string) ([]string, error) {
	entries, err := res.ReadDir(sceneRootDir(root))
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs, nil
}

// Sources 三域实际使用的来源（trace / npcprobe 用）。
type Sources struct {
	Placement, Template, Combat sharedata.Source
}

// SourcesOf 数据根三域的来源。
func SourcesOf(root string) Sources {
	return Sources{
		Placement: PlacementSource(root),
		Template:  TemplateSource(root),
		Combat:    CombatSource(root),
	}
}

func (s Sources) String() string {
	return "摆放=" + s.Placement.String() + " 模板=" + s.Template.String() + " 战斗=" + s.Combat.String()
}
