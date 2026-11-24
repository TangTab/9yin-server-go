// Package sharedata 客户端 share 树的统一读入口。
//
// 九阴只有设计态表随客户端发布（share/npc、share/creator、share/task…），
// 方言有三类：TAB 表（多为 GBK，1~3 行头）、creator XML（场景摆放）、键值
// 行档（.idres）。此前它们一律先离线转成服务端 XML（tools/datagen npc）
// 再由各装载器读；2026-11-25 起装载器可以直读——本包就是那条读路径。
//
//   - 编码自适应集中在一处（合法 UTF-8 直通，否则 GBK 解码；BOM 剥离）。
//   - 各域的表头方言（NPC 模板 3 行头 / 任务域 2 行头 / 战斗模板 1 行头）
//     解析口径与构建器逐条对齐：同一份源表，直读与 XML 产物必须给出行级
//     相同的结果（对拍见 npcfeed / taskdata 的 parity 测试）。
//   - Source：数据取自 share 树还是 XML 产物（env NINEYIN_DATA_SOURCE）。缺省
//     auto——数据根带 share 源就直读，否则读产物；share 树比产物旧时也读
//     产物（ShareOlderThanXML）。
//
// 读取一律经 internal/mpq/res 的路径式入口——层文件与散文件同一视图。
package sharedata

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// Source NPC / 任务数据的来源。
type Source int

const (
	// SourceAuto 缺省：该域的 share 源在数据根里齐备就直读，否则读 XML
	// 产物；share 树看起来比产物旧时也读产物（见 EffectiveSource）。
	SourceAuto Source = iota
	// SourceShare 直读客户端 share 树。
	SourceShare
	// SourceXML 读构建产物 XML（ini/Npc、ini/Scene、ini/Quest、ini/Text）。
	SourceXML
)

// sourceEnv 来源开关：share | xml；未设或其它值为 auto。
const sourceEnv = "NINEYIN_DATA_SOURCE"

// SourceFromEnv 读开关。
func SourceFromEnv() Source {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(sourceEnv))) {
	case "share":
		return SourceShare
	case "xml":
		return SourceXML
	}
	return SourceAuto
}

func (s Source) String() string {
	switch s {
	case SourceShare:
		return "share"
	case SourceXML:
		return "xml"
	}
	return "auto"
}

// EffectiveSource 定某一域实际使用的来源。sharePath 是该域在数据根下的源
// 路径（/ 分隔的相对路径，文件或目录，如 share/creator/npc_creator）：
//
//	显式 xml     照办。
//	显式 share   路径在即直读；不在回落 XML——数据根没带这份数据时不该让
//	             服务起不来（要提示「被回落」可对照 SourceFromEnv）。
//	auto（缺省） 路径在即直读，不在回落 XML；另有一道保险：share 树与 XML
//	             产物并存而 share 看起来比产物旧（ShareOlderThanXML）时也
//	             回落 XML。
func EffectiveSource(root, sharePath string) Source {
	if root == "" {
		return SourceXML
	}
	want := SourceFromEnv()
	if want == SourceXML {
		return SourceXML
	}
	if _, err := res.Stat(filepath.Join(root, filepath.FromSlash(sharePath))); err != nil {
		return SourceXML
	}
	if want == SourceAuto && ShareOlderThanXML(root) {
		return SourceXML
	}
	return SourceShare
}

// staleInfo 一个数据根的「share 树 vs XML 产物」摆放覆盖比对。
type staleInfo struct {
	checked   bool // 两边都在，做过比对
	shareOnly int  // 只在 share 摆放树里的场景目录数
	xmlOnly   int  // 只在 ini/Scene 里的场景目录数
}

// staleCache 比对结果按数据根缓存（每域每次装载都会来问）。
var staleCache sync.Map // 数据根 → staleInfo

// ShareOlderThanXML auto 模式的保险。一份源加一份产物，产物与源谁新没有
// 时间戳可看，但有一眼可辨的形态：产物按新客户端树构建、数据根里的
// share 却是旧快照——摆放树里缺了产物已有的场景（2026-11-25 的部署根：
// share 389 个场景目录，ini/Scene 405 个）。这时直读等于把 NPC 退回旧
// 快照，不如照旧读产物。判据：只在 ini/Scene 的场景目录多于只在 share
// 的（share 缺的比多的一边多）；任一边缺席（只有一种来源）不判。
//
// 反方向（share 比产物多场景 = 产物过期）不拦，照常直读。确认要以 share
// 为准时设 NINEYIN_DATA_SOURCE=share 绕过本判。
func ShareOlderThanXML(root string) bool {
	info := staleInfoOf(root)
	return info.checked && info.xmlOnly > info.shareOnly
}

// StaleShareNote 数据根的 share 树看起来比 XML 产物旧时的说明（trace 用）；
// 不适用返回空串。显式 share 时照直读，但要讲明读到的是一份旧树。
func StaleShareNote(root string) string {
	info := staleInfoOf(root)
	if !info.checked || info.xmlOnly <= info.shareOnly {
		return ""
	}
	head := fmt.Sprintf("数据根的 share 摆放树比 XML 产物少 %d 个场景（另有 %d 个产物没有的），像是旧快照",
		info.xmlOnly, info.shareOnly)
	if SourceFromEnv() == SourceShare {
		return head + "——当前 " + sourceEnv + "=share 强制直读，NPC / 任务数据取自这份旧树；把数据根的 share 换成当前客户端树后再用"
	}
	return head + "——已回落 xml（读构建产物）；把数据根的 share 树换成当前客户端树后自动改直读"
}

func staleInfoOf(root string) staleInfo {
	key := filepath.Clean(root)
	if v, ok := staleCache.Load(key); ok {
		return v.(staleInfo)
	}
	info := staleInfo{}
	shareDirs, xmlDirs := dirNameSet(root, CreatorDir), dirNameSet(root, "ini/Scene")
	if len(shareDirs) > 0 && len(xmlDirs) > 0 {
		info.checked = true
		for name := range shareDirs {
			if !xmlDirs[name] {
				info.shareOnly++
			}
		}
		for name := range xmlDirs {
			if !shareDirs[name] {
				info.xmlOnly++
			}
		}
	}
	staleCache.Store(key, info)
	return info
}

// dirNameSet 数据根下某目录的直接子目录名集合（小写；目录不在为空）。
func dirNameSet(root, rel string) map[string]bool {
	entries, err := res.ReadDir(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil
	}
	var set map[string]bool
	for _, e := range entries {
		if e.IsDir() {
			if set == nil {
				set = map[string]bool{}
			}
			set[strings.ToLower(e.Name())] = true
		}
	}
	return set
}

// ResetSourceCache 清掉 share 新旧判定缓存（测试面：改了数据根的摆放树后
// 重新判定）。
func ResetSourceCache() {
	staleCache = sync.Map{}
}

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// DecodeText 编码自适应：剥 BOM，合法 UTF-8 直通，否则按 GBK 解码（客户端
// TAB 表与 ini 多为 GBK，摆放 XML 与 .idres 多为带 BOM 的 UTF-8）。
func DecodeText(raw []byte) ([]byte, error) {
	raw = bytes.TrimPrefix(raw, utf8BOM)
	if utf8.Valid(raw) {
		return raw, nil
	}
	return simplifiedchinese.GBK.NewDecoder().Bytes(raw)
}

// ReadTextFile 读一份文本资源并转成 UTF-8（DecodeText 口径）——直读客户端
// share / text 下的 ini、idres、TAB 表都从这里拿原文，读入口不再各自判
// 编码。路径不在返回 fs.ErrNotExist。
func ReadTextFile(path string) ([]byte, error) {
	raw, err := res.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data, err := DecodeText(raw)
	if err != nil {
		return nil, fmt.Errorf("sharedata: 解码 %s: %w", filepath.ToSlash(path), err)
	}
	return data, nil
}
