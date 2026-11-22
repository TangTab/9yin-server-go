// Package npcdata NPC 目录数据源：场景摆放（ini/Scene/<场景>/npc_creator.xml）
// 与 NPC 模板（ini/Npc/NpcConfig 树）的装载与联查，供进场链物化 NPC
// （entergen m229 物化帧族）取数。
//
// 方言锚点（docs/official/外服NPCm229物化方言定版.md）：
//   - npc_creator.xml 行 = <Property File x y z ay id .../>，静态项有
//     x/y/z 坐标；*_creator 随机组只保留量与中心/半径，不发明坐标，
//     本包同样跳过无坐标行（tools/npcgen 同口径）。
//   - NpcConfig 行 = <Property ID script Name Resource NpcType Scale .../>，
//     id 摆放键 → 模板联查取 Resource（客户端模型路径）/Name/Scale。
package npcdata

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// NpcTemplate NpcConfig 模板行（物化所需的字段子集 + 原始行）。
type NpcTemplate struct {
	ID       string
	Script   string // 物化类名（CommonNpc/AttackNpc/...）
	Name     string // 显示名（中文）
	Resource string // 模型资源路径（npc\Guildqizhi01 形）
	Scale    string // 缩放串（"3.100,3.100,3.100" 形；空=不缩放）
	NpcType  int
	Raw      map[string]string // 原始行全字段（NPC 0x0D 模板层按名取）
}

// SceneNPC 一条场景摆放（已联查模板；Hit=false 表示模板缺席）。
type SceneNPC struct {
	ID   string // 摆放 id（模板联查键）
	File string // 摆放类别文件名（commonnpc.xml/attacknpc.xml/...）
	X, Y, Z float64
	AY  float64 // 朝向（yaw 弧度；缺省 0）
	Tpl NpcTemplate
	Hit bool
}

// pushableFiles 下发类别白名单：物件 + 人物静态项。
// attacknpc 族（战斗怪，需 AI/战斗面）与 *_creator 随机组（无坐标）不发。
var pushableFiles = map[string]bool{
	"commonnpc.xml":        true,
	"funcnpc.xml":          true,
	"itemnpc.xml":          true,
	"lightnpc.xml":         true,
	"guildbuildingnpc.xml": true,
	"gathernpc.xml":        true,
	"other.xml":            true,
}

// LoadSceneNPCs 装载场景摆放（testdata/ini/Scene/<sceneDir>/npc_creator.xml）。
// 文件缺失返回 err（调用方决定回退）；只收静态项（File 在白名单且带 x/y/z）。
func LoadSceneNPCs(root, sceneDir string) ([]SceneNPC, error) {
	path := filepath.Join(root, "ini", "Scene", filepath.FromSlash(sceneDir), "npc_creator.xml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("npcdata: 读场景摆放 %s: %w", path, err)
	}
	rows, err := parsePropRows(data)
	if err != nil {
		return nil, fmt.Errorf("npcdata: 解析 %s: %w", path, err)
	}
	out := make([]SceneNPC, 0, len(rows))
	for _, row := range rows {
		file := row["File"]
		if !pushableFiles[file] {
			continue
		}
		x, xerr := parseCoord(row["x"])
		y, yerr := parseCoord(row["y"])
		z, zerr := parseCoord(row["z"])
		if xerr != nil || yerr != nil || zerr != nil {
			continue // 随机组/坐标缺席项
		}
		ay, _ := parseCoord(row["ay"]) // 朝向可缺省
		out = append(out, SceneNPC{
			ID:   row["id"],
			File: file,
			X:    x, Y: y, Z: z, AY: ay,
		})
	}
	return out, nil
}

// parseCoord 坐标属性（十进制，允许负号；空串报错）。
func parseCoord(s string) (float64, error) {
	if s == "" {
		return 0, errNoCoord
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

var errNoCoord = fmt.Errorf("npcdata: 坐标缺席")

// templateCache 进程级模板缓存（NpcConfig 树 71MB 档只扫一次）。
var (
	templateCache   map[string]NpcTemplate
	templateCacheMu sync.Mutex
)

// LoadTemplates 递归装载 ini/Npc/NpcConfig 树 → id → 模板。
// 与 fsgame.NpcResourceModule.LoadResource（npcmods.go）同一树面，但保留
// 物化所需全字段；目录缺失返回 err。
func LoadTemplates(root string) (map[string]NpcTemplate, error) {
	templateCacheMu.Lock()
	defer templateCacheMu.Unlock()
	if templateCache != nil {
		return templateCache, nil
	}
	dir := filepath.Join(root, "ini", "Npc", "NpcConfig")
	templates := map[string]NpcTemplate{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".xml") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil // 单档读败跳过（与 fsgame 容错一致）
		}
		rows, parseErr := parsePropRows(data)
		if parseErr != nil {
			return nil
		}
		for _, row := range rows {
			id := row["ID"]
			if id == "" {
				continue
			}
			npctype, _ := strconv.Atoi(strings.TrimSpace(row["NpcType"]))
			raw := make(map[string]string, len(row))
			for k, v := range row {
				raw[k] = v
			}
			templates[id] = NpcTemplate{
				ID:       id,
				Script:   row["script"],
				Name:     row["Name"],
				Resource: row["Resource"],
				Scale:    row["Scale"],
				NpcType:  npctype,
				Raw:      raw,
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("npcdata: 扫 NpcConfig 树: %w", err)
	}
	if len(templates) == 0 {
		return nil, fmt.Errorf("npcdata: NpcConfig 树为空（root=%s）", root)
	}
	templateCache = templates
	return templateCache, nil
}

// ResetTemplateCache 清模板缓存（测试面）。
func ResetTemplateCache() {
	templateCacheMu.Lock()
	templateCache = nil
	templateCacheMu.Unlock()
}

// Join 模板联查（原地把命中项填 Tpl/Hit）。返回命中条数。
func Join(npcs []SceneNPC, templates map[string]NpcTemplate) int {
	hit := 0
	for i := range npcs {
		if tpl, ok := templates[npcs[i].ID]; ok && tpl.Resource != "" {
			npcs[i].Tpl, npcs[i].Hit = tpl, true
			hit++
		}
	}
	return hit
}

// Filter 距出生点（平面 x/z 距离）半径筛选 + 距离升序截断上限。
// radius<=0 视为不限；limit<=0 不截断。
func Filter(npcs []SceneNPC, ox, oz, radius float64, limit int) []SceneNPC {
	type distNPC struct {
		d float64
		n SceneNPC
	}
	r2 := radius * radius
	kept := make([]distNPC, 0, len(npcs))
	for _, npc := range npcs {
		d := dist2D(ox, oz, npc.X, npc.Z)
		if radius > 0 && d > r2 {
			continue
		}
		kept = append(kept, distNPC{d, npc})
	}
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].d < kept[j].d })
	if limit > 0 && len(kept) > limit {
		kept = kept[:limit]
	}
	out := make([]SceneNPC, len(kept))
	for i, k := range kept {
		out[i] = k.n
	}
	return out
}

func dist2D(ax, az, bx, bz float64) float64 {
	dx, dz := ax-bx, az-bz
	return dx*dx + dz*dz // 平方距离（排序/半径同单调）
}

// gb18030Reader gb2312/gbk 声明统一按 GB18030 超集解码（fsgame 同决策）。
func gb18030Reader(input io.Reader) io.Reader {
	return simplifiedchinese.GB18030.NewDecoder().Reader(input)
}


// parsePropRows 解析 <Object><Property k="v" .../></Object> 为属性行列表
// （fsgame.ParsePropTable 同式；xml 声明 utf-8 直读，gb2312/gbk 按 GB18030
// 超集解码兼容）。
func parsePropRows(data []byte) ([]map[string]string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch strings.ToLower(charset) {
		case "gb2312", "gbk", "gb18030":
			return gb18030Reader(input), nil
		}
		return nil, fmt.Errorf("npcdata: 不支持的字符集 %q", charset)
	}
	var rows []map[string]string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "Property" {
			continue
		}
		row := make(map[string]string, len(se.Attr))
		for _, a := range se.Attr {
			row[a.Name.Local] = a.Value
		}
		rows = append(rows, row)
	}
	return rows, nil
}
