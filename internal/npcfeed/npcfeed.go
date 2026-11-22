// Package npcfeed 场景 NPC 下发链（2026-11-22 全新实现，替代 npcdata+
// entergen/npc* 旧链）：服务端 XML 三件装载 → 摆放/模板/战斗联查 →
// 2433 帧配方（0x0D 物化大档 / 0x28 mode=0x46 官方言）。
//
// 数据链（服务端只读构建产物，不直读客户端资源——XML 由 tools/npcgen
// 离线构建，构建方式以黑金档为样板，见
// docs/official/NPC服务端XML构建链与黑金样板.md）：
//
//	ini/Npc/NpcConfig/**/*.xml    NPC 模板（视觉/交互面；行带 FightID）
//	ini/Npc/NpcTemplate.xml       战斗模板（npctemplate.tab 构建；ID 即
//	                              FightID 联查键——黑金实力包引用式解耦）
//	ini/Scene/<场景>/npc_creator.xml  场景摆放（creator 源名方言）
//
// 帧配方移植 2433 线 cmd/protocol-probe npcAddObject：0x0D = 63B 头 +
// OrderedProperties（默认值层→模板层→战斗层→摆放层，按 0x09 官方表
// 场景域表序）+ 官方 16-prop 追加块；ident obj=0x11000000|(idx+2)、
// own=0x00660000|低16位。
package npcfeed

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolveRoot 数据根定位：env NINEYIN_DATA_ROOT > 相对 cwd 探测含
// ini/Npc/NpcConfig 的目录（正式 $root Res / 测试仓内 testdata）。
func ResolveRoot() string {
	if v := strings.TrimSpace(os.Getenv("NINEYIN_DATA_ROOT")); v != "" {
		return v
	}
	for _, dir := range []string{
		"Res", "../Res", "../../Res", // 正式：$root Res
		".", "..", "../..", // 裸 ini 布局兼容
		"testdata", "../testdata", "../../testdata", // 测试：仓内构建产物
	} {
		if st, err := os.Stat(filepath.Join(dir, "ini", "Npc", "NpcConfig")); err == nil && st.IsDir() {
			return dir
		}
	}
	return ""
}

// sceneDirOf 场景键（born03）→ 摆放目录名（born03_yanyuzhuang）：
// ini/Scene 下前缀 "<scene>_" 或同名目录（大小写不敏感）。
func sceneDirOf(root, scene string) string {
	entries, err := os.ReadDir(filepath.Join(root, "ini", "Scene"))
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
