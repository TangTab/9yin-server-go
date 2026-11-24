package npcfeed

// load.go 服务端 XML 三件装载：NpcConfig 模板树 / NpcTemplate 战斗表 /
// 场景 npc_creator 摆放。全部 <Object><Property k="v"/> 行方言
//（npcgen 构建产物；UTF-8 声明直读，gb 系声明按 GB18030 超集解码）。

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// TemplateRow NpcConfig 模板行（原始属性 map + 派生键）。
type TemplateRow struct {
	ID     string
	Script string // 物化类名（CommonNpc/AttackNpc/...）
	Attrs  map[string]string
}

// CombatRow NpcTemplate 战斗行（npctemplate.tab 构建产物；原始属性 map，
// 数值列按名取：MaxLife/MaxMana/Level/命中/闪避/SkillID 族）。
type CombatRow struct {
	ID    string
	Attrs map[string]string
}

// Int 战斗行整型列（缺席/脏值 0）。
func (c CombatRow) Int(name string) int {
	v, err := strconv.Atoi(strings.TrimSpace(c.Attrs[name]))
	if err != nil {
		return 0
	}
	return v
}

// Placement 摆放行（creator 源名属性 + 坐标）。
type Placement struct {
	ID      string // 摆放 id（模板联查键）
	File    string // 摆放类别文件名（commonnpc.xml/funcnpc.xml/...）
	X, Y, Z float64
	AY      float64           // 朝向（弧度；缺省 0）
	Attrs   map[string]string // 原始行全字段（闸门判据 TaskSubId/SubSceneNo 等）
}

// ---- 进程级模板/战斗缓存（71MB 档只扫一次） ----

var (
	storeOnce sync.Once
	storeRoot string
	templates map[string]TemplateRow // ID → 模板行（同 ID 多表首档胜出）
	combat    map[string]CombatRow   // ID → 战斗行
	storeErr  error
)

// LoadStore 装载模板树 + 战斗表（进程级缓存；root 变更时以首载为准）。
func LoadStore(root string) (map[string]TemplateRow, map[string]CombatRow, error) {
	storeOnce.Do(func() {
		storeRoot = root
		templates, combat, storeErr = loadStoreOnce(root)
	})
	if root != storeRoot {
		return nil, nil, fmt.Errorf("npcfeed: 数据根已锚定 %s（进程级缓存），拒绝改载 %s", storeRoot, root)
	}
	return templates, combat, storeErr
}

// ResetStore 清缓存（测试面）。
func ResetStore() {
	storeOnce = sync.Once{}
	storeRoot, templates, combat, storeErr = "", nil, nil, nil
}

func loadStoreOnce(root string) (map[string]TemplateRow, map[string]CombatRow, error) {
	tpls, err := loadTemplates(filepath.Join(root, "ini", "Npc", "NpcConfig"))
	if err != nil {
		return nil, nil, err
	}
	cbt, err := loadCombat(filepath.Join(root, "ini", "Npc", "NpcTemplate.xml"))
	if err != nil {
		return nil, nil, err
	}
	return tpls, cbt, nil
}

// loadTemplates 递归装载 NpcConfig 树 → ID → 模板行。
// 黑金原档在参考资源根（testdata/references），不在本树内。
func loadTemplates(dir string) (map[string]TemplateRow, error) {
	templates := map[string]TemplateRow{}
	err := res.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(d.Name()), ".xml") {
			return nil
		}
		rows, parseErr := parsePropRowsFile(path)
		if parseErr != nil {
			return nil // 单档解析失败跳过（模板树大，个别脏档不废整树）
		}
		for _, row := range rows {
			id := row["ID"]
			if id == "" {
				continue
			}
			if _, dup := templates[id]; dup {
				continue // 同 ID 多表：首档胜出（npcgen 输出按玩法域分表，跨表撞 ID 极少）
			}
			templates[id] = TemplateRow{ID: id, Script: row["script"], Attrs: row}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("npcfeed: 扫 NpcConfig 树 %s: %w", dir, err)
	}
	if len(templates) == 0 {
		return nil, fmt.Errorf("npcfeed: NpcConfig 树为空（root 缺席？）")
	}
	return templates, nil
}

// loadCombat 战斗模板表（npctemplate.tab 构建产物）。
func loadCombat(path string) (map[string]CombatRow, error) {
	rows, err := parsePropRowsFile(path)
	if err != nil {
		return nil, fmt.Errorf("npcfeed: 读战斗模板 %s: %w", path, err)
	}
	combat := make(map[string]CombatRow, len(rows))
	for _, row := range rows {
		id := row["ID"]
		if id == "" {
			continue
		}
		combat[id] = CombatRow{ID: id, Attrs: row}
	}
	return combat, nil
}

// LoadPlacements 场景摆放（ini/Scene/<sceneDir>/npc_creator.xml）。
// 只收白名单类别且坐标合法的静态项；随机组（无坐标行）跳过。
func LoadPlacements(root, sceneDir string) ([]Placement, error) {
	path := filepath.Join(root, "ini", "Scene", filepath.FromSlash(sceneDir), "npc_creator.xml")
	rows, err := parsePropRowsFile(path)
	if err != nil {
		return nil, fmt.Errorf("npcfeed: 读场景摆放 %s: %w", path, err)
	}
	out := make([]Placement, 0, len(rows))
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
		out = append(out, Placement{
			ID: row["id"], File: file,
			X: x, Y: y, Z: z, AY: ay,
			Attrs: row,
		})
	}
	return out, nil
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

var errNoCoord = fmt.Errorf("npcfeed: 坐标缺席")

// parsePropRowsFile 解析 <Object><Property k="v" .../></Object> 为属性行
// 列表（XML 声明 utf-8 直读，gb2312/gbk/gb18030 按 GB18030 超集解码）。
func parsePropRowsFile(path string) ([]map[string]string, error) {
	data, err := res.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parsePropRows(data)
}

func parsePropRows(data []byte) ([]map[string]string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch strings.ToLower(charset) {
		case "gb2312", "gbk", "gb18030":
			return simplifiedchinese.GB18030.NewDecoder().Reader(input), nil
		}
		return nil, fmt.Errorf("npcfeed: 不支持的字符集 %q", charset)
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
