// Package npcfeed 场景 NPC 下发链（2026-11-22 全新实现，替代 npcdata+
// entergen/npc* 旧链）：模板 / 战斗 / 摆放三件装载 → 摆放/模板/战斗联查 →
// 2433 帧配方（0x0D 物化大档 / 0x28 mode=0x46 官方言）。
//
// 数据来源二选一（见 source.go）：
//
//	A. 直读客户端 share 树（2026-11-25 起，数据根带 share 即走这条）
//	   share/creator/npc_creator/<场景>/*.xml  场景摆放（creator 原生 XML）
//	   share/npc/npcconfig/**/*.txt             NPC 模板（3 行头 TAB 表；行带
//	                                            FightID←PowerNo）
//	   share/task/npctemplate*.tab              战斗模板
//	B. tools/datagen npc 离线构建的服务端 XML（NINEYIN_DATA_SOURCE=xml）
//	   ini/Npc/NpcConfig/**/*.xml    NPC 模板（视觉/交互面；行带 FightID）
//	   ini/Npc/NpcTemplate.xml       战斗模板（ID 即 FightID 联查键）
//	   ini/Scene/<场景>/npc_creator.xml  场景摆放（creator 源名方言）
//
// 两条路行方言相同；构建方式以黑金档为样板，见
// docs/official/NPC服务端XML构建链与黑金样板.md，取舍见
// docs/official/客户端share直读可行性分析.md。
//
// 帧配方移植 2433 线 cmd/protocol-probe npcAddObject：0x0D = 63B 头 +
// OrderedProperties（默认值层→模板层→战斗层→摆放层，按 0x09 官方表
// 场景域表序）+ 官方 16-prop 追加块；ident 分配方言见 join.go
// IdentScheme（缺省固定段 0x0E000100+idx；NINEYIN_NPC_IDENT=scene 为官服
// 形态 obj=场景 ID 低字节<<24|场景内序号、own=场景内序列号）。
package npcfeed

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// ResolveRoot 数据根定位：env NINEYIN_DATA_ROOT > 相对 cwd 探测——含
// ini/Npc/NpcConfig（带 XML 产物）或 share/npc/npcconfig（带客户端 share
// 树）的目录都算（正式 $root Res / 测试仓内 testdata）。
func ResolveRoot() string {
	if v := strings.TrimSpace(os.Getenv("NINEYIN_DATA_ROOT")); v != "" {
		return v
	}
	for _, dir := range []string{
		"Res", "../Res", "../../Res", // 正式：$root Res
		".", "..", "../..", // 裸 ini 布局兼容
		"testdata", "../testdata", "../../testdata", // 测试：仓内构建产物
	} {
		for _, probe := range []string{filepath.Join(dir, "ini", "Npc", "NpcConfig"), filepath.Join(dir, shareNPCConfigDir)} {
			if st, err := res.Stat(probe); err == nil && st.IsDir() {
				return dir
			}
		}
	}
	return ""
}

// SceneDirOf 场景键（born03）→ 摆放目录名（born03_yanyuzhuang）：
// 摆放目录树（见 source.go sceneRootDir）下前缀 "<scene>_" 或同名目录
// （大小写不敏感）。
// 巡逻路线装载（mappath 短名前缀）共用同目录名。
// ⚠️ 字典序首匹配——同前缀多变体目录（school01_fangong vs
// school01_JinYiWei）会选中字典序靠前者，权威路径走 canonicalSceneDir。
func SceneDirOf(root, scene string) string {
	entries, err := res.ReadDir(sceneRootDir(root))
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

// CanonicalSceneDir 目录名 → 摆放目录树（见 source.go sceneRootDir）下实际
// 目录名（大小写不敏感精确匹配；非前缀——变体目录不会被吞）。缺席返回空。
// 权威场景目录存在性探针（resolveSceneDir 逐级回落用）。
func CanonicalSceneDir(root, dir string) string {
	if dir == "" {
		return ""
	}
	entries, err := res.ReadDir(sceneRootDir(root))
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() && strings.EqualFold(e.Name(), dir) {
			return e.Name()
		}
	}
	return ""
}
