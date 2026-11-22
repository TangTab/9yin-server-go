package npcdata

// scenes.go 客户端 ini\scenes.ini 全局场景索引表：场景 resource（born03/
// city02/scene22）→ 全局段号。OpenMapRec 位图行/列与 MapMaskRec col0
// 共用这套编号（b-line map_records.go 同源；form_map_scene.lua
// update_scene_list_by_OpenMapRec 按位点亮城市标签）。
//
// 文件为客户端侧数据（GBK，300 段）；$root\ini\scenes.ini 缺席时
// SceneGlobalIndex 恒 false——调用方降级（只发样本位图、跳过迷雾行），
// 不阻塞进场。段号锚点：[3]=born03_YanYuZhuang、[14]=city02_SuZhou、
// [366]=scene22（与外服 c.pcapng OpenMapRec 行 11 bit14 互证）。

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var (
	sceneIndexMu    sync.Mutex
	sceneIndexRoots = map[string]map[string]uint32{}
)

// SceneGlobalIndex 取场景 resource 在客户端 scenes.ini 的全局段号。
// resource 大小写/空白归一（born03_YanYuZhuang 与 born03 同键——
// city/school/born/scene 前缀截到首个 '_'，其余取全基名）。
func SceneGlobalIndex(root, resource string) (uint32, bool) {
	key := strings.ToLower(strings.TrimSpace(resource))
	if key == "" {
		return 0, false
	}
	sceneIndexMu.Lock()
	defer sceneIndexMu.Unlock()
	table, ok := sceneIndexRoots[root]
	if !ok {
		table = loadSceneIndexTable(filepath.Join(root, "ini", "scenes.ini"))
		sceneIndexRoots[root] = table
	}
	if table == nil {
		return 0, false
	}
	index, ok := table[key]
	return index, ok
}

// loadSceneIndexTable 解析 scenes.ini（GBK）：段号 → Config 基名首现绑定。
// 文件缺失/解码失败返回 nil（进程级缓存负面结果，不重试）。
func loadSceneIndexTable(path string) map[string]uint32 {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	decoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), data)
	if err != nil {
		return nil
	}
	return ParseSceneIndexTable(string(decoded))
}

// ParseSceneIndexTable 纯解析（测试入口）：[段号] Config=ini\scene\<基名>，
// 同 resource 多段取首现。
func ParseSceneIndexTable(text string) map[string]uint32 {
	table := make(map[string]uint32, 300)
	section := -1
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if id, err := strconv.Atoi(line[1 : len(line)-1]); err == nil {
				section = id
			} else {
				section = -1
			}
			continue
		}
		if section < 0 || !strings.HasPrefix(strings.ToLower(line), "config=") {
			continue
		}
		config := strings.ReplaceAll(strings.TrimSpace(line[len("config="):]), "/", `\`)
		base := strings.TrimSuffix(filepath.Base(config), filepath.Ext(config))
		if base == "" {
			continue
		}
		key := SceneResourceOfBase(base)
		if _, exists := table[key]; !exists {
			table[key] = uint32(section)
		}
	}
	return table
}

// SceneResourceOfBase 配置基名 → 地形 resource 键：city/school/born/scene
// 前缀截到首个 '_'（born03_YanYuZhuang → born03、city02_SuZhou → city02），
// 其余取全基名小写（westzq、fbm101）。
func SceneResourceOfBase(base string) string {
	lower := strings.ToLower(base)
	for _, prefix := range []string{"city", "school", "born", "scene"} {
		if strings.HasPrefix(lower, prefix) {
			if separator := strings.IndexByte(lower, '_'); separator > 0 {
				return lower[:separator]
			}
		}
	}
	return lower
}
