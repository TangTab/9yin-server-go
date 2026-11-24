// Package npcdata 客户端 scenes.ini 场景索引（开图位图/迷雾行数据源）。
// 原场景 NPC 装载/联查已迁出：NPC 下发链 2026-11-22 重写为
// internal/npcfeed（服务端 XML 三件 + 2433 帧配方）。
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
//
// 段号即场景 ID：官服 wire ident 的 obj 高 8 位取其低字节（四抓包实证
// 2026-11-24：漠河镇 [4]→0x04 段、姑苏 [14]→0x0E 段、born_single_01 节号
// 317=0x13D→0x3D 段）——摆放目录 → 段号查 SceneIndexOfDir。

import (
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// sceneIndexTables scenes.ini 一次解析的两张索引，均为「同键多段取首现」：
// byResource 键=Config 基名的 resource（city02/born03，city/school/born/
// scene 前缀截到首个 '_'）；byDir 键=Config 基名全名小写（=ini\Scene 下
// 摆放目录名）。
type sceneIndexTables struct {
	byResource map[string]uint32
	byDir      map[string]uint32
}

var (
	sceneIndexMu    sync.Mutex
	sceneIndexRoots = map[string]*sceneIndexTables{}
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
	table := sceneIndexTablesOf(root)
	if table == nil {
		return 0, false
	}
	index, ok := table.byResource[key]
	return index, ok
}

// SceneIndexOfDir 取摆放目录名（ini\Scene 下目录，= scenes.ini 节 Config
// 基名，大小写不敏感）的场景号——官服形态 ident 的 obj 高字节来源
// （npcfeed.IdentScene）。精确基名优先；scenes.ini 未收录的变体目录
// （school01_fangong、scene08 _2_ 族）回落 resource 键的首现段号。
func SceneIndexOfDir(root, dir string) (uint32, bool) {
	key := strings.ToLower(strings.TrimSpace(dir))
	if key == "" {
		return 0, false
	}
	sceneIndexMu.Lock()
	defer sceneIndexMu.Unlock()
	table := sceneIndexTablesOf(root)
	if table == nil {
		return 0, false
	}
	if index, ok := table.byDir[key]; ok {
		return index, true
	}
	index, ok := table.byResource[SceneResourceOfBase(key)]
	return index, ok
}

// sceneIndexTablesOf 数据根的索引表（持 sceneIndexMu 调用）。文件缺失/
// 解码失败缓存 nil（进程级负面结果，不重试）。
func sceneIndexTablesOf(root string) *sceneIndexTables {
	table, ok := sceneIndexRoots[root]
	if !ok {
		table = loadSceneIndexTables(filepath.Join(root, "ini", "scenes.ini"))
		sceneIndexRoots[root] = table
	}
	return table
}

func loadSceneIndexTables(path string) *sceneIndexTables {
	data, err := res.ReadFile(path)
	if err != nil {
		return nil
	}
	decoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), data)
	if err != nil {
		return nil
	}
	return parseSceneIndexTables(string(decoded))
}

// ParseSceneIndexTable 纯解析（测试入口）：[段号] Config=ini\scene\<基名>
// 的 resource 键表，同 resource 多段取首现。
func ParseSceneIndexTable(text string) map[string]uint32 {
	return parseSceneIndexTables(text).byResource
}

func parseSceneIndexTables(text string) *sceneIndexTables {
	tables := &sceneIndexTables{
		byResource: make(map[string]uint32, 300),
		byDir:      make(map[string]uint32, 300),
	}
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
		if _, exists := tables.byDir[strings.ToLower(base)]; !exists {
			tables.byDir[strings.ToLower(base)] = uint32(section)
		}
		key := SceneResourceOfBase(base)
		if _, exists := tables.byResource[key]; !exists {
			tables.byResource[key] = uint32(section)
		}
	}
	return tables
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
