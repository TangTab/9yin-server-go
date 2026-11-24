// mapping.go 分档规则、静态包规格、审定映射与未转换表清单（本工具的核心）。
//
// 黑金 ini/Item 的分档是「物品大类」：path_config.xml 逐档登记 Type/Path，
// 行自身的 script 决定服务端对象类，文件只管归类。九阴的 script 有 125 个
// 逻辑类，按 kindOf 的优先序归入黑金同名大类（path_config 的 Type 写法
// 沿用黑金原写法，含大小写不一的 material/Taskmaterial/mount）；黑金没有
// 对应物、九阴又各自成类且体量过万的两个大类（礼盒、武学书）单列，其余
// 落 ItemUse——与黑金把 Bag/VipItem/TransItem 行放进 ItemUse.xml 同款。
//
// 分档只影响文件归属：装载与使用链一律按行内 script/FuncPack 等属性取值，
// 挪档不改语义。
package item

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// templateSources 物品模板源表（行 = 一件物品，节名即 ConfigID）。
var templateSources = []string{"tool_item.ini", "equipment.ini", "tool_item_hw.ini", "giftbox.ini"}

// kindSpec 一个输出档（文件名 + path_config 登记的 Type）。顺序即
// path_config.xml 的登记序：黑金已有的大类按其原相对序在前
// （Equipment→Weapon→material→ItemUse→Rune→mount→TaskItemUse→
// Taskmaterial），九阴新增大类殿后。
type kindSpec struct {
	File string // 输出文件名（不含 .xml）
	Type string // path_config Type
	Note string
}

var kindSpecs = []kindSpec{
	{File: "Equipment", Type: "Equipment", Note: "防具/饰器/时装/武学书装备：EquipType 非武器族，或 equipment.ini 的 Equipment/ShotWeapon/FacultyPaint/FacultyBook 行"},
	{File: "Weapon", Type: "Weapon", Note: "武器：EquipType ∈ weapon/shotweapon/innerweapon"},
	{File: "Material", Type: "material", Note: "材料：背包材料页（ViewID=3）"},
	{File: "ItemUse", Type: "ItemUse", Note: "道具：其余全部（ToolItem/Bag/CardItem/TitleItem… 黑金同档亦混装 Bag/VipItem/TransItem）"},
	{File: "Rune", Type: "Rune", Note: "镶嵌：script=Enchase"},
	{File: "ItemMountUse", Type: "mount", Note: "坐骑道具：script ∈ Mount/Sable/Saddle"},
	{File: "TaskItemUse", Type: "TaskItemUse", Note: "任务页可用道具：ViewID=4 且带 FuncPack（黑金 TaskItemUse=ToolItem 行）"},
	{File: "TaskMaterial", Type: "Taskmaterial", Note: "任务材料：ViewID=4 且无 FuncPack（黑金 TaskMaterial=Material 行）"},
	{File: "ItemBox", Type: "ItemBox", Note: "礼盒（九阴新增大类）：script ∈ BoxItem/ChoiceBoxItem/CompensateItem/GCBoxItem/FurnitureBox/TreasureChest"},
	{File: "SkillBook", Type: "SkillBook", Note: "武学书（九阴新增大类）：script ∈ SkillBook/SkillPage/SkillScroll/SkillBookTag/SuperBook"},
	{File: "add_money", Type: "add_money", Note: "掉落金钱显示物：script=DropMoneyItem（黑金同名同义）"},
}

// weaponEquipTypes 武器族 EquipType（九阴 weapon/shotweapon/innerweapon 同占
// 纸娃娃 22 槽）。
var weaponEquipTypes = map[string]bool{"weapon": true, "shotweapon": true, "innerweapon": true}

// equipmentScripts equipment.ini 的逻辑类（无 EquipType 的行也归装备档）。
var equipmentScripts = map[string]bool{"Equipment": true, "ShotWeapon": true, "FacultyPaint": true, "FacultyBook": true}

var (
	mountScripts     = map[string]bool{"Mount": true, "Sable": true, "Saddle": true}
	boxScripts       = map[string]bool{"BoxItem": true, "ChoiceBoxItem": true, "CompensateItem": true, "GCBoxItem": true, "FurnitureBox": true, "TreasureChest": true}
	skillBookScripts = map[string]bool{"SkillBook": true, "SkillPage": true, "SkillScroll": true, "SkillBookTag": true, "SuperBook": true}
)

// kindOf 一行物品的归档（优先序即判定序）。
func kindOf(row map[string]string) string {
	script := row["script"]
	equipType := strings.ToLower(strings.TrimSpace(row["EquipType"]))
	viewID, _ := strconv.Atoi(strings.TrimSpace(row["ViewID"]))
	funcPack, _ := strconv.Atoi(strings.TrimSpace(row["FuncPack"]))
	switch {
	case weaponEquipTypes[equipType]:
		return "Weapon"
	case equipType != "" || equipmentScripts[script]:
		return "Equipment"
	case script == "Enchase":
		return "Rune"
	case mountScripts[script]:
		return "ItemMountUse"
	case script == "DropMoneyItem":
		return "add_money"
	case skillBookScripts[script]:
		return "SkillBook"
	case boxScripts[script]:
		return "ItemBox"
	case viewID == 4:
		if funcPack != 0 {
			return "TaskItemUse"
		}
		return "TaskMaterial"
	case viewID == 3:
		return "Material"
	}
	return "ItemUse"
}

// leadColumns 模板档输出列序的前置列（取源表存在者，按声明序；其余按
// 首见序）。ID 恒居首。
var leadColumns = []string{
	"script", "ItemType", "ViewID", "Name", "ColorLevel", "Photo", "EquipType",
	"Level", "MaxAmount", "Amount", "LogicPack", "FuncPack",
}

// packSpec 一个静态包（物品行以整型 ID 引用：ArtPack="3243" →
// ItemArtStatic.xml 行 3243）。黑金已发布的静态包 XML 保留 [-1]/[0] 两行
// 装载器元数据（列序头/列宽头），本工具对静态包照留；模板档同黑金不
// 留。输出列序取 [-1] 列序头，表头未列的键按首见序殿后。
type packSpec struct {
	Source string // 源 ini（相对 -in）
	Output string // 输出 xml（相对 <out>）
	Ref    string // 物品行引用键
	Title  string
	Note   string
}

var packSpecs = []packSpec{
	{Source: "item_func_static.ini", Output: "item_func_static.xml", Ref: "FuncPack",
		Title: "功能包", Note: "使用链核心：PrepareTime/PrepareAction = 读条时长/动作（官服 m25[PrepareTime,\"flow_prepare\",\"useitem_flow\"] 实测吻合），LuaScript = 官服物品脚本路径"},
	{Source: "item_logical_static.ini", Output: "item_logic_pack.xml", Ref: "LogicPack",
		Title: "逻辑包", Note: "黑金同名档（LogicPack 引用）；列集为九阴自家词汇（BindType/CantDepot…），与黑金 item_logic_pack.xml 的 CantFaceToFaceExchange… 不同"},
	{Source: "itemartstatic.ini", Output: "ItemArtStatic.xml", Ref: "ArtPack",
		Title: "美术静态包", Note: "黑金同名档（ArtPack 引用）：Photo 图标 + 男女模型 + 动作集"},
	{Source: "item_use_skill_static.ini", Output: "item_use_skill_static.xml", Ref: "UseSkillPack",
		Title: "使用技能包", Note: "UseSkillPack 引用（药品=使用技能：SkillID/SkillLevel）；黑金对应物是 SkillPack→item_skill_static.xml，引用键不同名故保留九阴档名"},
	{Source: "item_learn_skill_static.ini", Output: "item_learn_skill_static.xml", Ref: "LearnSkillPack",
		Title: "学习技能包", Note: "武学书学习链：TragetID（官方拼写）= 目标技能，逗号/分号混隔"},
	{Source: "item_task_static.ini", Output: "item_task_static.xml", Ref: "TaskPack", Title: "任务包"},
	{Source: "item_ingredient_static.ini", Output: "item_ingredient_static.xml", Ref: "IngredientPack", Title: "配方包"},
	{Source: "item_prop_modify_static.ini", Output: "item_prop_modify_static.xml", Ref: "PropModifyPack", Title: "属性修正包"},
	{Source: "item_seed_static.ini", Output: "item_seed_static.xml", Ref: "SeedPack", Title: "种子包"},
}

// skipReason share/item 下未转换文件的归类理由（全部入审计报告）。
func skipReason(name string) string {
	switch {
	case strings.HasPrefix(name, "fuse_"):
		return "合成域（配方/残卷/书页合成），非物品模板"
	case strings.HasPrefix(name, "life_"):
		return "生活技能域"
	case name == "exchangeitem.ini" || strings.HasPrefix(name, "item_exchange") ||
		name == "item_school_exchange.ini" || name == "item_linxiao_exchange.ini":
		return "兑换域"
	case strings.HasPrefix(name, "senior_") || strings.HasPrefix(name, "propconv") ||
		name == "recast_blacklist.ini" || strings.HasPrefix(name, "inscription_") ||
		name == "equipactivatestatic.ini" || name == "weapon_transfrom.ini" ||
		name == "weapon_transfrom_id.ini" || name == "weapon_equipcategory.ini":
		return "装备精炼/铭刻/转换域"
	case strings.HasPrefix(name, "home_"):
		return "家园域"
	case name == "royal_treasure_map.ini" || name == "qixingjian_item.ini" || strings.HasPrefix(name, "treasure"):
		return "宝箱/寻宝域"
	case strings.HasPrefix(name, "randomevent"):
		return "随机事件域（item_func_static.RandomEventPackID 的下游）"
	case name == "huanpi_suits.ini" || name == "fashion_suit_resource.ini":
		return "换皮/时装域"
	case name == "itemlogicstatic.ini":
		return "旧版逻辑包（3 行；现行表 item_logical_static.ini 已转换）"
	case name == "item_copy_static.ini" || name == "item_material_static.ini":
		return "空表（0 字节）"
	case name == "innerequip":
		return "内功装备域（目录）"
	}
	return "物品域外围配置"
}

// ---- 物品类型总表（黑金 ItemTypeList.xml 同构）----

// itemTypeEntry 类型总表一行：ID / 父类型 / 常量名（同值多名以 / 连接）。
type itemTypeEntry struct {
	ID     int
	Father int
	Names  []string
}

// itemTypeRoot 全部类型的根（黑金同值）。
const itemTypeRoot = 55555

var luaConstPattern = regexp.MustCompile(`(?m)^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(-?\d+)\s*(?:--.*)?$`)

// parseItemTypeList 客户端 lua64/share/itemtype_define.lua → 类型树。
//
// 常量分两类：具名类型（ITEMTYPE_WEAPON_BLADE = 101）与区间界
// （ITEMTYPE_WEAPON_MIN = 100 / _MAX = 119）。父类型取**最内层**包含区间
// 的下界（101 → WEAPON[100,119] → 100）；值恰为某区间下界者不以自身为父
// （TOOLITEM_MIN 与 ADDHP 同值 1：ADDHP 挂根，2..99 挂 1）；不在任何区间
// 内的类型挂根 55555。黑金列 TextID（zx_djtype_<ID> 文案键）九阴无对应
// 键规，不写。
func parseItemTypeList(text string) []itemTypeEntry {
	byID := map[int][]string{}
	for _, m := range luaConstPattern.FindAllStringSubmatch(text, -1) {
		name := strings.TrimPrefix(m[1], "ITEMTYPE_")
		id, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		if !containsString(byID[id], name) {
			byID[id] = append(byID[id], name)
		}
	}
	// 区间：同前缀的 MIN/MAX 对（HUANPIMIN/HUANPIMAX 无下划线，按去尾
	// 归一前缀比对）。缺一半的界不成区间——ITEMTYPE_GAME_MARKET_MAX 孤立
	// 存在，若按 min=0 补齐会把 1..499 全挂到 0 下。
	type span struct{ min, max int }
	type bounds struct {
		min, max       int
		hasMin, hasMax bool
	}
	pairs := map[string]*bounds{}
	for id, names := range byID {
		for _, name := range names {
			trimmed := strings.TrimSuffix(name, "_")
			var key string
			var isMin bool
			switch {
			case strings.HasSuffix(trimmed, "MIN"):
				key, isMin = strings.TrimSuffix(trimmed, "MIN"), true
			case strings.HasSuffix(trimmed, "MAX"):
				key = strings.TrimSuffix(trimmed, "MAX")
			default:
				continue
			}
			if pairs[key] == nil {
				pairs[key] = &bounds{}
			}
			if isMin {
				pairs[key].min, pairs[key].hasMin = id, true
			} else {
				pairs[key].max, pairs[key].hasMax = id, true
			}
		}
	}
	spans := map[string]span{}
	for key, b := range pairs {
		if b.hasMin && b.hasMax {
			spans[key] = span{b.min, b.max}
		}
	}
	fatherOf := func(id int) int {
		best, bestSpan := itemTypeRoot, 0
		for _, s := range spans {
			if s.min >= s.max || id <= s.min || id > s.max {
				continue // 自身下界不算子项；空/倒区间忽略
			}
			if width := s.max - s.min; bestSpan == 0 || width < bestSpan {
				best, bestSpan = s.min, width
			}
		}
		return best
	}
	entries := make([]itemTypeEntry, 0, len(byID))
	for id, names := range byID {
		if id == itemTypeRoot {
			continue
		}
		sort.Strings(names)
		entries = append(entries, itemTypeEntry{ID: id, Father: fatherOf(id), Names: names})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Father != entries[j].Father {
			return entries[i].Father < entries[j].Father
		}
		return entries[i].ID < entries[j].ID
	})
	return entries
}

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
