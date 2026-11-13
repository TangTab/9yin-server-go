package main

// GM 面板目录端点补全：console-core.js / console-features.js 启动即
// fetch 的一批 /api/*（副本、场景、礼包、称号、坐骑、待机站姿、洗练
// 词条、训练木桩套路、门派声望、身份系统、极简登录开关）此前后端
// 未注册，浏览器侧一律 404，对应板块显示“加载失败”。本文件补齐这
// 批端点：数据源是游戏服进程内的客户端资源表（GBK ini，与
// gm_item_catalog.go 同款装载方式）或进程内状态；文件缺失按各板块
// 前端已约定的降级形态应答（空数组 / loaded:false），绝不 5xx。
//
// 独立面板进程（cmd/gm-panel）经 internal API 的同名 catalog 端点
// 代理这一批 handler，两种部署形态数据同源。

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/local/9yin-go-server/internal/gmapi"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// gmScanINI 是本文件各表共用的宽松 INI 扫描器：GBK 解码（失败回退
// 原字节）、BOM/CRLF 容错、注释行跳过。visit 对节内每个 key=value
// 回调一次（key 已小写；section 为空表示文件头游离键，调用方自行
// 忽略）。
func gmScanINI(data []byte, visit func(section, key, value string)) {
	if decoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), data); err == nil {
		data = decoded
	}
	section := ""
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") ||
			strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		visit(section, strings.ToLower(strings.TrimSpace(key)), strings.TrimSpace(value))
	}
}

// gmReadINI 读资源树内一张 ini 表；缺失返回错误（调用方按空目录降级）。
func gmReadINI(parts ...string) ([]byte, error) {
	return readResourceBytes(filepath.Join(parts...))
}

// ---------------------------------------------------------------------------
// /api/clones —— 副本目录（console.html「副本」节：列表来自
// share/clone/cloneinfo.ini；难度档 1=普通 2=困难 3=地狱，按键存在性
// 输出；_yx/_qy 等变体是独立副本行，前端原样列出）。
// ---------------------------------------------------------------------------

type gmCloneRow struct {
	Config       string `json:"config"`
	Description  string `json:"description"`
	Difficulties []int  `json:"difficulties"`
}

var (
	gmCloneRowsOnce sync.Once
	gmCloneRows     []gmCloneRow
)

func loadGMCloneRows() []gmCloneRow {
	gmCloneRowsOnce.Do(func() {
		gmCloneRows = []gmCloneRow{}
		if data, err := gmReadINI(defaultModernShareRoot, "clone", "cloneinfo.ini"); err == nil {
			gmCloneRows = parseGMCloneRows(data)
		}
	})
	return gmCloneRows
}

// parseGMCloneRows 纯函数装载：节名即副本 config（ini\scene\clone003
// 原样，enter_clone 命令要原值）；难度键存在性 → [1,2,3] 子集；无任何
// 难度键的畸形行给 [1]（前端 syncCloneFun 以 difficulties[0] 兜底，
// 空档会让它选中 undefined）。
func parseGMCloneRows(data []byte) []gmCloneRow {
	order := []string{}
	byConfig := map[string]*gmCloneRow{}
	current := (*gmCloneRow)(nil)
	gmScanINI(data, func(section, key, value string) {
		if section == "" || strings.HasPrefix(section, "//") {
			return
		}
		if existing, ok := byConfig[section]; ok {
			current = existing
		} else if current == nil || section != current.Config {
			if current != nil {
				order = append(order, current.Config)
			}
			current = &gmCloneRow{Config: section}
			byConfig[section] = current
		}
		switch key {
		case "desc":
			if current.Description == "" {
				current.Description = value
			}
		case "easypowerlevel":
			current.Difficulties = append(current.Difficulties, 1)
		case "difficultpowerlevel":
			current.Difficulties = append(current.Difficulties, 2)
		case "hellpowerlevel":
			current.Difficulties = append(current.Difficulties, 3)
		}
	})
	if current != nil {
		order = append(order, current.Config)
	}
	rows := make([]gmCloneRow, 0, len(order))
	for _, config := range order {
		row := *byConfig[config]
		sort.Ints(row.Difficulties)
		if len(row.Difficulties) == 0 {
			row.Difficulties = []int{1}
		}
		rows = append(rows, row)
	}
	return rows
}

func (hub *gmHub) clonesHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	rows := loadGMCloneRows()
	gmWriteTaggedJSON(writer, request, gmStaticTagOf("clones", rows), rows)
}

// ---------------------------------------------------------------------------
// /api/scenes —— 场景目录（「自定义切图」节下拉）。config 保留
// rule/scenes.ini 原始大小写（switch_scene 命令要原值）；显示名复用
// sceneDisplayName 的 idres → resource → scene_name_NNN 链。
// ---------------------------------------------------------------------------

type gmSceneRow struct {
	Config   string `json:"config"`
	Resource string `json:"resource"`
	Name     string `json:"name"`
	ID       int    `json:"id"`
}

var (
	gmSceneRowsOnce sync.Once
	gmSceneRows     []gmSceneRow
)

func loadGMSceneRows() []gmSceneRow {
	gmSceneRowsOnce.Do(func() {
		gmSceneRows = []gmSceneRow{}
		if data, err := gmReadINI(modernScenesRulePath); err == nil {
			gmSceneRows = parseGMSceneRows(data)
		}
	})
	return gmSceneRows
}

// parseGMSceneRows 纯函数装载：数字节名 = 场景 id，Config 值形如
// ini\scene\school09_YiHuaGong；同 id 重复节保留首见 config（与
// loadScenesINI 的 putIfAbsent 同口径），行按 id 升序。
func parseGMSceneRows(data []byte) []gmSceneRow {
	seen := map[int]bool{}
	var rows []gmSceneRow
	gmScanINI(data, func(section, key, value string) {
		if section == "" || key != "config" || value == "" {
			return
		}
		id, err := strconv.Atoi(section)
		if err != nil || seen[id] {
			return
		}
		seen[id] = true
		resource := value
		if slash := strings.LastIndexAny(resource, `\/`); slash >= 0 {
			resource = resource[slash+1:]
		}
		rows = append(rows, gmSceneRow{
			Config:   value,
			Resource: resource,
			Name:     gmSceneDisplayName(value, resource),
			ID:       id,
		})
	})
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

// gmSceneDisplayName 场景目录显示名（GM 面板专用）：idres config/resource
// 两键命中即中文；未命中的（test04/plot2 等测试场景，idres 本无名）回落
// resource 基名——面板侧没人本地化 scene_name_NNN 键，发编码键只会显示
// 「编码」（sceneDisplayName 的 scene_name_NNN 回落是给选角 f24 用的，
// 客户端可本地化，两处口径不同）。
func gmSceneDisplayName(config, resource string) string {
	tables := sceneNameTablesOf()
	if name := tables.nameByKey[strings.ToLower(config)]; name != "" {
		return name
	}
	if name := tables.nameByKey[strings.ToLower(resource)]; name != "" {
		return name
	}
	return resource
}

func (hub *gmHub) scenesHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	rows := loadGMSceneRows()
	gmWriteTaggedJSON(writer, request, gmStaticTagOf("scenes", rows), rows)
}

// ---------------------------------------------------------------------------
// /api/giftboxes —— 礼包目录（「礼包与物品」节）。列表来自
// tool_item.ini 的 ItemType 2102/2103 开箱类（gmItems 目录装载时已
// 保留 ItemType）；hasContent 按 drop 包表/箱内容表是否已有该
// configID 判定（drops_mvp.json / box_contents.json），未配置时前端
// 标「未配置」并降级提示。
// ---------------------------------------------------------------------------

type gmGiftBoxRow struct {
	Config     string `json:"config"`
	Name       string `json:"name"`
	HasContent bool   `json:"hasContent"`
}

func loadGMGiftBoxRows() []gmGiftBoxRow {
	rows := []gmGiftBoxRow{}
	packs := currentDropPacks()
	contents := currentBoxContents()
	for _, item := range gmItems.rows {
		if item.ItemType != 2102 && item.ItemType != 2103 {
			continue
		}
		rows = append(rows, gmGiftBoxRow{
			Config:     item.ConfigID,
			Name:       item.Name,
			HasContent: packs.has(item.ConfigID) || len(contents.byBox(item.ConfigID)) > 0,
		})
	}
	return rows
}

func (hub *gmHub) giftBoxesHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	gmWriteContentTagged(writer, request, "giftboxes", loadGMGiftBoxRows())
}

// ---------------------------------------------------------------------------
// /api/titles —— 称号目录（「称号管理」节）。share/rule/title.ini 给
// 称号 ID → TitleType；中文名 = 客户端本地化键 role_title_<id>（form_
// role_info/form_title.lua GetText 同链），主表在 text/chineses/desc.idres
// （UTF-8，覆盖约 1300/3474，未命中的多为测试/结拜特殊段，客户端同样
// 键回退，这里 name 留空、前端回落显示 ID）。行序：有名称号在前
//（面板首屏可读、可按名称过滤），无名称号殿后，各自 ID 升序。
// ---------------------------------------------------------------------------

type gmTitleRow struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"`
}

var (
	gmTitleRowsOnce sync.Once
	gmTitleRows     []gmTitleRow
)

// gmTitleNamesOnce 装载 desc.idres 的 role_title_<id> → 中文名（只收该
// 前缀，与 stringname.idres 的场景/物品名表互不相干）。
var (
	gmTitleNamesOnce sync.Once
	gmTitleNames     map[int]string
)

func loadGMTitleNames() map[int]string {
	gmTitleNamesOnce.Do(func() {
		gmTitleNames = map[int]string{}
		if content, err := readResourceBytes(filepath.Join(defaultModernTextRoot, "chineses", "desc.idres")); err == nil {
			for _, raw := range strings.Split(strings.TrimPrefix(string(content), "\ufeff"), "\n") {
				line := strings.TrimSpace(raw)
				if !strings.HasPrefix(line, "role_title_") {
					continue
				}
				key, value, found := strings.Cut(line, "=")
				if !found || value == "" {
					continue
				}
				id, err := strconv.Atoi(strings.TrimPrefix(key, "role_title_"))
				if err != nil {
					continue
				}
				if _, exists := gmTitleNames[id]; !exists {
					gmTitleNames[id] = value
				}
			}
		}
	})
	return gmTitleNames
}

func loadGMTitleRows() []gmTitleRow {
	gmTitleRowsOnce.Do(func() {
		gmTitleRows = []gmTitleRow{}
		if data, err := gmReadINI(defaultModernShareRoot, "rule", "title.ini"); err == nil {
			gmTitleRows = parseGMTitleRows(data, loadGMTitleNames())
		}
	})
	return gmTitleRows
}

// parseGMTitleRows 纯函数装载：节名 = 称号 ID，TitleType = 类型；
// 非 TitleType 键与非数字节跳过；行序 = 有名称号在前（ID 升序）、
// 无名称号殿后（ID 升序）。
func parseGMTitleRows(data []byte, names map[int]string) []gmTitleRow {
	var ids []int
	typeByTitle := map[int]int{}
	gmScanINI(data, func(section, key, value string) {
		if key != "titletype" || value == "" {
			return
		}
		id, idErr := strconv.Atoi(section)
		typ, typErr := strconv.Atoi(value)
		if idErr != nil || typErr != nil {
			return
		}
		if _, exists := typeByTitle[id]; !exists {
			ids = append(ids, id)
		}
		typeByTitle[id] = typ
	})
	sort.Ints(ids)
	rows := make([]gmTitleRow, 0, len(ids))
	unnamed := make([]gmTitleRow, 0)
	for _, id := range ids {
		row := gmTitleRow{ID: id, Type: typeByTitle[id], Name: names[id]}
		if row.Name == "" {
			unnamed = append(unnamed, row)
		} else {
			rows = append(rows, row)
		}
	}
	return append(rows, unnamed...)
}

func (hub *gmHub) titlesHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	rows := loadGMTitleRows()
	gmWriteTaggedJSON(writer, request, gmStaticTagOf("titles", rows), rows)
}

// ---------------------------------------------------------------------------
// /api/mounts —— 坐骑目录（「坐骑管理」节）。列表来自
// ini/ui/life/allmount.ini（节名 = 坐骑物品 configID，客户端
// form_allmount.lua 同一入口）；中文名与 ArtPack 走 gmItems 物品目录
//（idres 键 MOUNT_6_003=坐骑·河套马 直取，物品表 Name=河套马 回退，
// ArtPack=9507 是 itemartstatic 物品外观 ID——客户端坐骑渲染同款取值）。
// ---------------------------------------------------------------------------

type gmMountRow struct {
	ConfigID string `json:"configID"`
	Name     string `json:"name"`
	ArtPack  string `json:"artPack"`
}

var (
	gmMountRowsOnce sync.Once
	gmMountRows     []gmMountRow
)

func loadGMMountRows() []gmMountRow {
	gmMountRowsOnce.Do(func() {
		gmMountRows = []gmMountRow{}
		data, err := gmReadINI(runtimeProjectPath("resources", "modern", "ini", "ui", "life", "allmount.ini"))
		if err != nil {
			return
		}
		itemsByConfig := make(map[string]gmItemRow, len(gmItems.rows))
		for _, row := range gmItems.rows {
			itemsByConfig[row.ConfigID] = row
		}
		gmMountRows = parseGMMountRows(data, itemsByConfig, sceneNameTablesOf().nameByKey)
	})
	return gmMountRows
}

// parseGMMountRows 纯函数装载：节名顺序即坐骑目录（物品目录缺失的
// 条目也保留，name/artPack 留空）；中文名链 = 物品目录 Name（idres→
// INI 名已解析）→ allmount 自身 name 键的 idres 表；artPack = 物品表
// ArtPack 键。
func parseGMMountRows(data []byte, itemsByConfig map[string]gmItemRow, names map[string]string) []gmMountRow {
	var order []string
	uiNameByConfig := map[string]string{}
	gmScanINI(data, func(section, key, value string) {
		if section == "" || strings.HasPrefix(section, "//") || key != "name" {
			return
		}
		if _, exists := uiNameByConfig[section]; !exists {
			uiNameByConfig[section] = value
			order = append(order, section)
		}
	})
	rows := make([]gmMountRow, 0, len(order))
	for _, configID := range order {
		row := gmMountRow{ConfigID: configID}
		if item, ok := itemsByConfig[configID]; ok {
			row.Name = item.Name
			row.ArtPack = item.ArtPack
		}
		if row.Name == "" {
			row.Name = names[strings.ToLower(uiNameByConfig[configID])]
		}
		if row.Name == "" {
			// 最后回退：坐骑物品 configID 本身常是 idres 键
			//（MOUNT_6_003=坐骑·河套马）。
			row.Name = names[strings.ToLower(configID)]
		}
		rows = append(rows, row)
	}
	return rows
}

func (hub *gmHub) mountsHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	rows := loadGMMountRows()
	gmWriteTaggedJSON(writer, request, gmStaticTagOf("mounts", rows), rows)
}

// ---------------------------------------------------------------------------
// /api/idle-stances —— 待机站姿目录（「待机站姿（持久化）」节）。
// 前端约定来源是客户端 b_action_index.ini 的全部含 stand 节；当前
// 资源树未携带该表，按空目录应答（面板显示 0 个候选，快捷站姿按钮
// 仍可用）。资源补齐后此处自动出全量列表。
// ---------------------------------------------------------------------------

type gmIdleStanceRow struct {
	Action   string `json:"action"`
	Skeleton string `json:"skeleton"`
}

var (
	gmIdleStancesOnce sync.Once
	gmIdleStances     []gmIdleStanceRow
)

func loadGMIdleStances() []gmIdleStanceRow {
	gmIdleStancesOnce.Do(func() {
		gmIdleStances = []gmIdleStanceRow{}
		if data, err := gmReadINI(runtimeProjectPath("resources", "modern", "ini", "b_action_index.ini")); err == nil {
			gmIdleStances = parseGMIdleStances(data)
		}
	})
	return gmIdleStances
}

// parseGMIdleStances 纯函数装载：节名含 stand 的节进目录
// （action = 节名，skeleton = Skeleton 键值，缺键留空）。
func parseGMIdleStances(data []byte) []gmIdleStanceRow {
	var order []string
	skeletonByAction := map[string]string{}
	gmScanINI(data, func(section, key, value string) {
		if section == "" || !strings.Contains(strings.ToLower(section), "stand") {
			return
		}
		if _, exists := skeletonByAction[section]; !exists {
			skeletonByAction[section] = ""
			order = append(order, section)
		}
		if key == "skeleton" && skeletonByAction[section] == "" {
			skeletonByAction[section] = value
		}
	})
	rows := make([]gmIdleStanceRow, 0, len(order))
	for _, action := range order {
		rows = append(rows, gmIdleStanceRow{Action: action, Skeleton: skeletonByAction[action]})
	}
	return rows
}

func (hub *gmHub) idleStancesHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	rows := loadGMIdleStances()
	gmWriteTaggedJSON(writer, request, gmStaticTagOf("idle-stances", rows), rows)
}

// ---------------------------------------------------------------------------
// /api/recast-affixes —— 洗练词条目录（「洗练」节）。share/item/
// senior_weapon_dire.ini 的 r=<PropName>,<词条ID>,… 行给出每把武器
// 可洗的普通词条；同一词条多武器出现时 source 合并武器节名，Name 联查
// proppack 首行增强为「属性名+值」。招式词条（skill）取洗练随机池内
// skillpack 归属包（13234 族双表同号——skillpack 侧是招式威力，Name=
// Comment 中文名）；整表缺失时 loaded=false，前端显示「洗练资源未加载」
// 提示。
// ---------------------------------------------------------------------------

type gmRecastAffix struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source,omitempty"`
}

type gmRecastSkill struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source,omitempty"`
	Detail string `json:"detail,omitempty"`
	Taolu  string `json:"taolu,omitempty"`
	Type   string `json:"type,omitempty"` // 实招/虚招/架招/怒招（skill_static 归属）
	Weapon string `json:"weapon,omitempty"`
}

type gmRecastAffixes struct {
	Loaded bool            `json:"loaded"`
	Prop   []gmRecastAffix `json:"prop"`
	Skill  []gmRecastSkill `json:"skill"`
}

var (
	gmRecastAffixesOnce sync.Once
	gmRecastAffixesData gmRecastAffixes
)

func loadGMRecastAffixes() gmRecastAffixes {
	gmRecastAffixesOnce.Do(func() {
		gmRecastAffixesData = gmRecastAffixes{Prop: []gmRecastAffix{}, Skill: []gmRecastSkill{}}
		if data, err := gmReadINI(defaultModernShareRoot, "item", "senior_weapon_dire.ini"); err == nil {
			gmRecastAffixesData = parseGMRecastAffixes(data)
		}
		gmAttachRecastSkills(&gmRecastAffixesData)
		gmEnhanceRecastPropNames(&gmRecastAffixesData)
	})
	return gmRecastAffixesData
}

// gmAttachRecastSkills 招式词条目录（Skill 数组）：官方洗练随机池内
// skillpack 归属包（13234 族双表同号——skillpack 侧是招式威力，如
// [13234] Comment=神风诀的弄月吟风威力增加7成）。Name 取节内 Comment
// 中文名，Detail 取首条 r= 威力行；随机池/skillpack 缺失时保持空数组。
func gmAttachRecastSkills(data *gmRecastAffixes) {
	randBytes, err := gmReadINI(defaultModernShareRoot, "item", "senior_weapon_rand.ini")
	if err != nil {
		return
	}
	pool := map[int]bool{}
	gmScanINI(randBytes, func(section, key, value string) {
		if key != "r" {
			return
		}
		if _, ok := parseUintSection(section); ok {
			return // 节名须是武器 config 串（非数字）
		}
		for _, id := range splitIntList(value) {
			if id != 0 {
				pool[id] = true
			}
		}
	})
	if len(pool) == 0 {
		return
	}
	type skillAgg struct {
		name     string
		detail   string
		firstKey string // 首条 r= 行的技能 key（类型标注查 skillStatic）
	}
	byID := map[int]*skillAgg{}
	if skillBytes, skillErr := gmReadINI(defaultModernShareRoot, "modifypack", "skillpack.ini"); skillErr == nil {
		gmScanINI(skillBytes, func(section, key, value string) {
			id, ok := parseUintSection(section)
			if !ok || id == 0 || !pool[int(id)] {
				return
			}
			agg, exists := byID[int(id)]
			if !exists {
				agg = &skillAgg{}
				byID[int(id)] = agg
			}
			switch key {
			case "comment":
				if agg.name == "" {
					agg.name = value
				}
			case "r":
				if agg.firstKey == "" && strings.Contains(value, ",") {
					agg.firstKey = strings.ToLower(strings.TrimSpace(strings.Split(value, ",")[0]))
				}
				if agg.detail == "" && strings.Contains(value, ",") {
					if fields := strings.Split(value, ","); len(fields) >= 3 {
						agg.detail = strings.TrimSpace(fields[0]) + " " +
							strings.TrimSpace(fields[1]) + " +" + strings.TrimSpace(fields[2])
					}
				}
			}
		})
	}
	if len(byID) == 0 {
		return
	}
	ids := make([]int, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	affixes := currentSkillAffixCatalog()
	for _, id := range ids {
		agg := byID[id]
		name := agg.name
		if name == "" {
			name = fmt.Sprintf("招式词条 %d", id)
		}
			row := gmRecastSkill{ID: id, Name: name, Detail: agg.detail}
			if static, ok := affixes.skillStatic[agg.firstKey]; ok {
				row.Type = skillAffixTypeLabel(static.EffectType, static.Rage)
				row.Taolu = static.Taolu
			}
		data.Skill = append(data.Skill, row)
	}
}

// gmEnhanceRecastPropNames 普通词条名增强：Name 从裸路由属性名
//（MeleeDamage）增强为 "属性名+值"（如 MinMeleeDamage+7——proppack 该
// 包首条 r= 行；13234 族双定义包在 proppack 侧是 AIPhyDefAdd+100）。
func gmEnhanceRecastPropNames(data *gmRecastAffixes) {
	propBytes, err := gmReadINI(defaultModernShareRoot, "modifypack", "proppack.ini")
	if err != nil {
		return
	}
	firstLine := map[int]string{} // pack → "PropName+value"
	gmScanINI(propBytes, func(section, key, value string) {
		if key != "r" || !strings.Contains(value, ",") {
			return
		}
		pack, ok := parseUintSection(section)
		if !ok || pack == 0 {
			return
		}
		if _, seen := firstLine[int(pack)]; seen {
			return
		}
		fields := strings.Split(value, ",")
		if len(fields) < 3 {
			return
		}
		if propName := strings.TrimSpace(fields[1]); propName != "" {
			firstLine[int(pack)] = propName + "+" + strings.TrimSpace(fields[2])
		}
	})
	for index := range data.Prop {
		if label, ok := firstLine[data.Prop[index].ID]; ok {
			data.Prop[index].Name = label
		}
	}
}

// parseGMRecastAffixes 纯函数装载：r= 行首字段是词条属性名，其后
// 数字列是词条 ID；ID 跨武器去重（name 取首见属性名，source 为武器
// 节名列表保序去重），行按 ID 升序。空表（或全非数据行）时
// loaded=false。
func parseGMRecastAffixes(data []byte) gmRecastAffixes {
	type affixAgg struct {
		name    string
		sources []string
	}
	byID := map[int]*affixAgg{}
	gmScanINI(data, func(section, key, value string) {
		if key != "r" || !strings.Contains(value, ",") {
			return
		}
		fields := strings.Split(value, ",")
		for index, field := range fields {
			id, err := strconv.Atoi(strings.TrimSpace(field))
			if err != nil || id == 0 {
				continue
			}
			if index == 0 {
				// 首字段是属性名（MeleeDamage…），不是词条 ID。
				continue
			}
			agg, exists := byID[id]
			if !exists {
				agg = &affixAgg{name: strings.TrimSpace(fields[0])}
				byID[id] = agg
			}
			agg.sources = append(agg.sources, section)
		}
	})
	result := gmRecastAffixes{Prop: []gmRecastAffix{}, Skill: []gmRecastSkill{}}
	if len(byID) == 0 {
		return result
	}
	ids := make([]int, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		agg := byID[id]
		result.Prop = append(result.Prop, gmRecastAffix{
			ID: id, Name: agg.name, Source: strings.Join(dedupeStrings(agg.sources), " "),
		})
	}
	result.Loaded = true
	return result
}

// dedupeStrings 保序去重（洗练词条的武器来源合并用）。
func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func (hub *gmHub) recastAffixesHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	affixes := loadGMRecastAffixes()
	gmWriteTaggedJSON(writer, request, gmStaticTagOf("recast-affixes", affixes), affixes)
}

// ---------------------------------------------------------------------------
// /api/skill-affixes —— 招式威力词条两级目录（「模板生成」技能词条的
// 套路 → 招式选择）：套路行序与 /api/taolus 一致（school_taolu 文件序），
// 每套路只列有威力词条包（skillpack PackCategory=5）的基座招式，Type
// 标注实招/虚招/架招/怒招（skill_static EffectType + PackCategory==2）。
// 数据缺失按 loaded=false 空表应答，前端显示降级提示。
// ---------------------------------------------------------------------------

type gmSkillAffixSkill struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type gmSkillAffixTaolu struct {
	Key    string              `json:"key"`
	Label  string              `json:"label"`
	Skills []gmSkillAffixSkill `json:"skills"`
}

type gmSkillAffixes struct {
	Loaded bool                `json:"loaded"`
	Taolus []gmSkillAffixTaolu `json:"taolus"`
}

var (
	gmSkillAffixesOnce sync.Once
	gmSkillAffixesData gmSkillAffixes
)

func loadGMSkillAffixes() gmSkillAffixes {
	gmSkillAffixesOnce.Do(func() {
		gmSkillAffixesData = buildGMSkillAffixes(currentSkillAffixCatalog(), gmTaolus, currentSkillDescNames())
	})
	return gmSkillAffixesData
}

// buildGMSkillAffixes 纯函数组装：威力目录的套路分组按 /api/taolus 行
// 序输出（gmTaolus 未收录的威力套路按 key 追加在尾部，仍可用）；招式
// 名缺 string_skill_desc 条目时回退 key 原样。
func buildGMSkillAffixes(catalog *skillAffixCatalog, taolus *gmTaoluCatalog, names map[string]string) gmSkillAffixes {
	result := gmSkillAffixes{Taolus: []gmSkillAffixTaolu{}}
	if catalog == nil || len(catalog.taoluSkills) == 0 {
		return result
	}
	emit := func(taoluKey string) {
		members := catalog.taoluSkills[taoluKey]
		if len(members) == 0 {
			return
		}
		row := gmSkillAffixTaolu{Key: taoluKey, Skills: make([]gmSkillAffixSkill, 0, len(members))}
		if resolved, ok := taolus.resolve(taoluKey); ok {
			row.Label = resolved.Label
		} else {
			row.Label = taoluKey
		}
		for _, key := range members {
			name := names[key]
			if name == "" {
				name = key
			}
			static := catalog.skillStatic[key]
			row.Skills = append(row.Skills, gmSkillAffixSkill{
				Key: key, Name: name, Type: skillAffixTypeLabel(static.EffectType, static.Rage),
			})
		}
		result.Taolus = append(result.Taolus, row)
	}
	emitted := make(map[string]bool, len(catalog.taoluSkills))
	if taolus != nil {
		for _, row := range taolus.rows {
			key := strings.ToLower(row.Key)
			if _, has := catalog.taoluSkills[key]; !has {
				continue // 该套路无威力词条招式：面板两级选择跳过。
			}
			emit(key)
			emitted[key] = true
		}
	}
	// 威力分组中有而 /api/taolus 未收录的套路（如 shijia 之外的分组）按
	// key 序追加，避免目录丢失。
	rest := make([]string, 0)
	for key := range catalog.taoluSkills {
		if !emitted[key] {
			rest = append(rest, key)
		}
	}
	sort.Strings(rest)
	for _, key := range rest {
		emit(key)
	}
	result.Loaded = len(result.Taolus) > 0
	return result
}

func (hub *gmHub) skillAffixesHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	affixes := loadGMSkillAffixes()
	gmWriteTaggedJSON(writer, request, gmStaticTagOf("skill-affixes", affixes), affixes)
}

// ---------------------------------------------------------------------------
// /api/combat-dummy/taolus —— 训练木桩套路预设（「训练木桩」节）。
// 套路预设依赖服务端战斗假人命令（spawn_combat_dummy）与套路静态
// 目录，两者尚未落地；接口先按前端契约返回空数组（下拉仅剩「随机
// 套路」占位），命令实现时补真数据。
// ---------------------------------------------------------------------------

type gmTaoluRow struct {
	Key    string   `json:"key"`
	Label  string   `json:"label"`
	Skills []string `json:"skills"`
}

func (hub *gmHub) combatDummyTaolusHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	gmWriteTaggedJSON(writer, request, gmStaticTagOf("combat-dummy-taolus", []gmTaoluRow{}), []gmTaoluRow{})
}

// ---------------------------------------------------------------------------
// /api/taolus 与 /api/neigongs —— 「整套学习」发放目录（武学套路区块）。
// 套路=school_taolu.ini+taolu_info.ini 精选集；内功=neigong.ini 全目录
// （含 varprop 满层，供面板层数输入参考）。资源缺失时为空数组，面板
// 下拉显示「无」。
// ---------------------------------------------------------------------------

func (hub *gmHub) taolusHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	gmWriteTaggedJSON(writer, request, gmStaticTagOf("taolus", gmTaolus.rows), gmTaolus.rows)
}

func (hub *gmHub) neigongsHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	gmWriteContentTagged(writer, request, "neigongs", loadGMNeiGongRows())
}

// ---------------------------------------------------------------------------
// /api/repute/status 与 /api/origin/status —— 门派声望/身份系统装载
// 状态。两块功能的命令端（repute_apply / origin_activate_all）与数据
// 表（声望势力表、data/origin/origins.json）都尚未接入服务端，按前端
// 约定的降级形态应答（loaded:false），板块显示「未加载」而非报错。
// ---------------------------------------------------------------------------

type gmReputeRow struct {
	Name     string `json:"name"`
	Display  string `json:"display"`
	MaxValue int    `json:"maxValue"`
}

type gmReputeStatus struct {
	Loaded  bool          `json:"loaded"`
	Reputes []gmReputeRow `json:"reputes"`
}

type gmOriginStatus struct {
	Loaded bool `json:"loaded"`
	Total  int  `json:"total"`
}

func (hub *gmHub) reputeStatusHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	gmWriteTaggedJSON(writer, request, gmStaticTagOf("repute-status", gmReputeStatus{}), gmReputeStatus{Loaded: false, Reputes: []gmReputeRow{}})
}

func (hub *gmHub) originStatusHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	gmWriteTaggedJSON(writer, request, gmStaticTagOf("origin-status", gmOriginStatus{}), gmOriginStatus{Loaded: false, Total: 0})
}

// ---------------------------------------------------------------------------
// /api/minimal-login —— 极简登录开关（console-features.js
// loadMinimalLogin/toggleMinimalLogin 消费）。GET 读当前值，POST 取反
// 并回新值。开关本体是进程级状态；登录推送链路（H-only 推送跳过）
// 尚未接入消费点，先保证面板读写面完整，接入后无需再动面板。
// ---------------------------------------------------------------------------

var gmMinimalLoginEnabled atomic.Bool

func (hub *gmHub) minimalLoginHandler(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		enabled := gmMinimalLoginEnabled.Load()
		gmWriteTaggedJSON(writer, request, gmapi.HashTag("minimal-login", strconv.FormatBool(enabled), gmBootStamp), map[string]bool{"enabled": enabled})
	case http.MethodPost:
		gmMinimalLoginEnabled.Store(!gmMinimalLoginEnabled.Load())
		writeGMJSON(writer, http.StatusOK, map[string]bool{"enabled": gmMinimalLoginEnabled.Load()})
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// ---------------------------------------------------------------------------
// S2/S3（设计文档 §5.1）：目录启动预热 + 静态目录标签 + manifest 端点。
// ---------------------------------------------------------------------------

// gmStaticTagMemo name → 标签条目。静态目录行装载一次后进程内不变，
// 标签（行哈希+boot 戳）也只算一次。
var gmStaticTagMemo sync.Map

type gmStaticTagEntry struct {
	once sync.Once
	tag  string
}

// gmStaticTagOf 取（首次计算并缓存）静态目录标签。调用方须先触发装载
// （rows 传入已装载结果），once 内只做哈希。
func gmStaticTagOf(name string, rows any) string {
	entryAny, _ := gmStaticTagMemo.LoadOrStore(name, &gmStaticTagEntry{})
	entry := entryAny.(*gmStaticTagEntry)
	entry.once.Do(func() { entry.tag = gmCatalogTag(name, rows) })
	return entry.tag
}

// warmGMCatalogDirs S2 目录预热：boot 后台把懒装载目录全部先解析一遍，
// 首个用户请求永不再等 INI 解析（sync.Once 机制不变，只是提前触发）。
// gmItems/skinCards/fashionSuits/gmTaolus 本就是包级启动装载，无需列入。
func warmGMCatalogDirs() {
	loadGMCloneRows()
	loadGMSceneRows()
	loadGMTitleRows()
	loadGMMountRows()
	loadGMIdleStances()
	loadGMRecastAffixes()
	loadGMNeiGongRows() // 触发 modernNeiGongCatalogOnce
}

// gmCatalogManifest S3 manifest 载荷：面板前端用它做目录缓存键版本。
type gmCatalogManifest struct {
	Version         string            `json:"version"`
	NamesGeneration uint64            `json:"namesGeneration"`
	Catalogs        map[string]string `json:"catalogs"`
}

// manifestHandler GET /api/catalog/manifest（internal 侧同名镜像）：
// version = boot 戳 + 名称代数；catalogs 给静态目录的当前标签。
func (hub *gmHub) manifestHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	namesGen := gmNameGen()
	version := gmBootStamp + "-g" + strconv.FormatUint(namesGen, 10)
	payload := gmCatalogManifest{
		Version:         version,
		NamesGeneration: namesGen,
		Catalogs: map[string]string{
			"clones":         gmStaticTagOf("clones", loadGMCloneRows()),
			"scenes":         gmStaticTagOf("scenes", loadGMSceneRows()),
			"titles":         gmStaticTagOf("titles", loadGMTitleRows()),
			"mounts":         gmStaticTagOf("mounts", loadGMMountRows()),
			"idle-stances":   gmStaticTagOf("idle-stances", loadGMIdleStances()),
			"recast-affixes": gmStaticTagOf("recast-affixes", loadGMRecastAffixes()),
			"taolus":         gmStaticTagOf("taolus", gmTaolus.rows),
			"weather":        gmapi.WeatherTag(),
		},
	}
	gmWriteTaggedJSON(writer, request, gmapi.HashTag("manifest", version), payload)
}
