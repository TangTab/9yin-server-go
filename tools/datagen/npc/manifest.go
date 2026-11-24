// manifest.go 表清单：npc_type.ini（[file] 段，键名官方拼作 flie<N>）
// 记录每张客户端表的服务端原路径 ini\Npc\NpcConfig\<组>\<表>.txt——
// 客户端表即服务端表镜像，输出树按此布局（.txt → .xml）。
package npc

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// manifestEntry 一张表的登记项。
type manifestEntry struct {
	ClientPath string // npcconfig 相对路径（slash，磁盘实际大小写）
	ServerRel  string // 输出相对路径（slash）：NpcConfig/<组>/<表>.xml
	Declared   bool   // npc_type.ini 是否登记（未登记表按镜像路径兜底）
}

// loadManifest 解析 npc_type.ini 并与目录扫描交叉核对。
// 返回按 ClientPath 排序的清单与差异告警。
func loadManifest(npcconfigDir string) ([]manifestEntry, []string, error) {
	declared, err := loadNPCTypeINI(filepath.Join(npcconfigDir, "npc_type.ini"))
	if err != nil {
		return nil, nil, err
	}

	// 目录扫描收集全部 .txt 数据表（npc_type.txt 是清单的文档镜像，排除）。
	var found []string
	err = res.WalkDir(npcconfigDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".txt") {
			return walkErr
		}
		rel, relErr := filepath.Rel(npcconfigDir, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if strings.EqualFold(rel, "npc_type.txt") {
			return nil
		}
		found = append(found, rel)
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("npc: 扫描 %s: %w", npcconfigDir, err)
	}
	sort.Strings(found)

	var warnings []string
	byLower := map[string]string{} // 小写 ClientPath → 磁盘实际路径
	for _, rel := range found {
		byLower[strings.ToLower(rel)] = rel
	}

	entries := make([]manifestEntry, 0, len(found))
	for _, rel := range found {
		entries = append(entries, manifestEntry{
			ClientPath: rel,
			ServerRel:  mirrorServerPath(rel),
			Declared:   false,
		})
	}
	// npc_type.ini 登记的服务端路径优先（大小写以官方清单为准）。
	declaredLower := map[string]string{} // 小写客户端路径 → 服务端输出路径
	for clientPath, serverPath := range declared {
		declaredLower[strings.ToLower(filepath.ToSlash(clientPath))] = serverPath
	}
	for i := range entries {
		if serverRel, ok := declaredLower[strings.ToLower(entries[i].ClientPath)]; ok {
			entries[i].ServerRel = serverRel
			entries[i].Declared = true
		}
	}

	// 差异报告：清单登记但磁盘缺失。
	for _, clientPath := range sortedKeys(declared) {
		if _, ok := byLower[strings.ToLower(clientPath)]; !ok {
			warnings = append(warnings, fmt.Sprintf("npc_type.ini 登记的表磁盘缺失: %s", clientPath))
		}
	}
	for _, e := range entries {
		if !e.Declared {
			warnings = append(warnings, fmt.Sprintf("表未在 npc_type.ini 登记，按镜像路径输出: %s", e.ClientPath))
		}
	}
	return entries, warnings, nil
}

// loadNPCTypeINI 解析 npc_type.ini（GBK）。行形如
// flie32=ini\Npc\NpcConfig\WorldNpc\city_CommonNpc.txt（官方把 file 拼作
// flie，两种键名都容忍）。返回 客户端相对路径 → 服务端输出相对路径。
func loadNPCTypeINI(path string) (map[string]string, error) {
	raw, err := res.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("npc: 读 npc_type.ini: %w", err)
	}
	decoded, err := toUTF8(raw)
	if err != nil {
		return nil, fmt.Errorf("npc: 解码 npc_type.ini: %w", err)
	}
	text := string(decoded)
	out := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "[") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		value := strings.TrimSpace(line[eq+1:])
		if value == "" {
			continue
		}
		serverRel, clientRel, ok := splitServerNPCPath(value)
		if !ok {
			continue
		}
		out[clientRel] = serverRel
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("npc: npc_type.ini 未解析出任何登记项")
	}
	return out, nil
}

// splitServerNPCPath 把 ini\Npc\NpcConfig\WorldNpc\city_CommonNpc.txt 拆成
// （输出相对路径 NpcConfig/WorldNpc/city_CommonNpc.xml，客户端相对路径
// WorldNpc/city_CommonNpc.txt——即 npcconfig 下的磁盘路径）。
func splitServerNPCPath(value string) (serverRel, clientRel string, ok bool) {
	normalized := filepath.ToSlash(value)
	const marker = "ini/npc/" // 在 ToLower 后的串里搜，必须全小写
	idx := strings.Index(strings.ToLower(normalized), marker)
	if idx < 0 {
		return "", "", false
	}
	after := normalized[idx+len(marker):]
	if !strings.HasSuffix(strings.ToLower(after), ".txt") {
		return "", "", false
	}
	serverRel = strings.TrimSuffix(after, ".txt") + ".xml"
	clientRel = strings.TrimSuffix(after, ".txt") + ".txt"
	// clientRel 去掉 NpcConfig/ 层（npcconfig 磁盘树从组目录开始）。
	const groupLayer = "NpcConfig/"
	if len(clientRel) > len(groupLayer) && strings.EqualFold(clientRel[:len(groupLayer)], groupLayer) {
		clientRel = clientRel[len(groupLayer):]
	}
	return serverRel, clientRel, true
}

// mirrorServerPath 未登记表的兜底：npcconfig 相对路径镜像 + .xml，
// 统一挂到 NpcConfig/ 下（组目录保持磁盘大小写）。
func mirrorServerPath(clientRel string) string {
	return "NpcConfig/" + strings.TrimSuffix(clientRel, ".txt") + ".xml"
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
