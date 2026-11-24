package npcfeed

// load_share.go 直读客户端 share 树的模板 / 战斗装载（source.go 的 A 路）。
//
// 模板：share/npc/npcconfig/**/*.txt——每张表 3 行头（中文注释 / 类型 /
// 列键）的 GBK TAB 表，几乎一表一套列头。行 → 属性的口径与 XML 产物逐条
// 对齐（对拍 TestTemplateSourceParity：两条路读同一份数据根，全部模板行
// 必须相同）：
//
//	ID、script、Name、Desc（=Name）   前置四项（Desc 是九阴中文名）
//	其余列按源列序：
//	  npcColumnBindings 命中 → 服务端属性名（direct 同名 / rename 改名）
//	  列键带 table@ 前缀   → 去前缀保留（输出名撞车追加 _2/_3）
//	  未映射              → 不入行
//	空值不入行；同名列取首见；ID 格为空的行不入（表尾说明 / 排版行）
//
// 「未映射不入行」是刻意的：物化帧的模板层按 0x09 官方场景域字段名同名列
// 直连（resolve.go OrderedProperties），未映射列里有一批正是场景域字段
// （SpringEffect / Material / ActionSoundMode / FixedBeHitAct / DirectSpring
// …）——放开它们等于一次改掉几万个 NPC 的下发帧。要让某个列进模板行，
// 在 npcColumnBindings 加一条绑定即可（重启生效，不用重建数据）；构建器
// tools/datagen/npc/mapping.go 里有一份同用途的表，两边的差异由对拍测试
// 兜住。
//
// 表序（同 ID 多表时谁胜出）见 sharedata.NPCConfigTables。
//
// 战斗：share/task/npctemplate*.tab（1 行头 TAB 表）——npctemplate.tab 主档
// 序前，其余按文件名补充，同 ID 首行胜出。

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/internal/sharedata"
)

// npcColumnKind 一条列绑定的类别（审计 / 测试面）。
type npcColumnKind int

const (
	npcColumnDirect   npcColumnKind = iota // 两边同名同义，直通
	npcColumnRename                        // 改名映射（Note 给 review 依据）
	npcColumnTableRec                      // table@ 前缀引用列，去前缀保留
	npcColumnDropped                       // 未映射，不入行
)

// npcBinding 一条列绑定：客户端列键 → 服务端属性名。
type npcBinding struct {
	Server string
	Kind   npcColumnKind
	Note   string // rename 的 review 依据
}

// npcColumnBindings 审定映射表。键为客户端表第 3 行列键。
//
// 服务端词汇来源：黑金档 common_npc*.xml ∪ struct/npc（VisibleNpc 及其
// include）。只列人工核对过的绑定，不猜——没把握的宁可不入行。
var npcColumnBindings = map[string]npcBinding{
	// ---- direct：同名同义 ----
	"ID":             {Server: "ID"},
	"script":         {Server: "script"},
	"Name":           {Server: "Name"},
	"Level":          {Server: "Level"},
	"Camp":           {Server: "Camp"},
	"Resource":       {Server: "Resource"},
	"NpcType":        {Server: "NpcType"},
	"SelectState":    {Server: "SelectState"},
	"NeedRotate":     {Server: "NeedRotate"},
	"WalkSpeed":      {Server: "WalkSpeed"},
	"RunSpeed":       {Server: "RunSpeed"},
	"WeaponMode":     {Server: "WeaponMode"},
	"WeaponName":     {Server: "WeaponName"},
	"CollideRadius":  {Server: "CollideRadius"},
	"AITemplate":     {Server: "AITemplate"},
	"PatrolMode":     {Server: "PatrolMode"},
	"PatrolRange":    {Server: "PatrolRange"},
	"SpringRange":    {Server: "SpringRange"},
	"Call":           {Server: "Call"},
	"ChaseRange":     {Server: "ChaseRange"},
	"CantAttack":     {Server: "CantAttack"},
	"CantBeAttack":   {Server: "CantBeAttack"},
	"CantBeKilled":   {Server: "CantBeKilled"},
	"CantMove":       {Server: "CantMove"},
	"CantUseSkill":   {Server: "CantUseSkill"},
	"PickChangeTime": {Server: "PickChangeTime"},
	"DestroyTime1":   {Server: "DestroyTime1"},
	"DestroyTime2":   {Server: "DestroyTime2"},
	"RefreshTime":    {Server: "RefreshTime"},
	"LifeTime":       {Server: "LifeTime"},
	"Invisible":      {Server: "Invisible"},
	"Scale":          {Server: "Scale"},
	"Fixed":          {Server: "Fixed"}, // struct internals（internals_npc.xml）
	"CallHelpRange":  {Server: "CallHelpRange"},
	"CallHelpType":   {Server: "CallHelpType"},

	// ---- direct：NPC 业务功能列（商店 / 仓库 / 职业 / 驿站 / 落脚点 /
	// 公会功能——商店与仓库 View 接入定案）----
	"ShopID":       {Server: "ShopID"},
	"DepotID":      {Server: "DepotID"},
	"Job":          {Server: "Job"},
	"TransFuncID":  {Server: "TransFuncID"},
	"HomePoint":    {Server: "HomePoint"},
	"GuildFunc":    {Server: "GuildFunc"},
	"GemGameNpcID": {Server: "GemGameNpcID"},

	// ---- direct：NPC 业报列（善恶参数 / 地位评分——对话通用行门控：
	// 官服只对 Character 有值且 KarmaLevel≥1 的 NPC 发「赠送物品 / 侠恶
	// 查询」两行）----
	"Character":  {Server: "Character"},
	"KarmaLevel": {Server: "KarmaLevel"},

	// ---- direct：NPC 出生状态列（姿态 NPC 是动作名，普通为 stand；另两种
	// 同义列名 table@BornActionRec / table@BornAction 走 table@ 去前缀，
	// 输出名 BornActionRec / BornAction）----
	"BornAction": {Server: "BornAction"},

	// ---- rename：改名映射 ----
	"PathID": {Server: "PathId", Kind: npcColumnRename, Note: "大小写对齐黑金档 PathId（巡逻路径 ID）"},
	"FacultyExp": {Server: "Exp", Kind: npcColumnRename,
		Note: "九阴修为经验 → 黑金经验列 Exp（VisibleNpc 无独立属性，common_npc.xml 真档用法）"},
	"DropID": {Server: "WorldDropID", Kind: npcColumnRename, Note: "九阴掉落 → 黑金世界掉落列"},
	"SelfForceID": {Server: "Force", Kind: npcColumnRename,
		Note: "势力归属 → Force（黑金 common_npc.xml 真档列，语义一致）"},
	"PowerNo": {Server: "FightID", Kind: npcColumnRename,
		Note: "review：实力包编号 → FightID。黑金 FightID 形如 ZS_PT_0030，九阴 PowerNo 为实力包引用"},
	"table@SkillRec": {Server: "SkillPackID", Kind: npcColumnRename,
		Note: "review：技能表引用 → SkillPackID"},
}

// npcColumn 一条输出列。
type npcColumn struct {
	index int    // 源列下标
	name  string // 输出属性名（未映射时是源列键，仅审计）
	kind  npcColumnKind
}

// planNPCColumns 一张表的列输出计划（列序保持源表顺序）：审定绑定优先；
// 未绑定但带 table@ 前缀的列去前缀保留（输出名撞车追加 _2/_3）；其余
// dropped。同名列只计划首见列。
func planNPCColumns(t *sharedata.Table) []npcColumn {
	first := t.First()
	seen := map[string]int{} // 输出属性名 → 已占用计数
	plan := make([]npcColumn, 0, len(t.Columns))
	for i, key := range t.Columns {
		if first[key] != i {
			continue
		}
		var pc npcColumn
		if b, ok := npcColumnBindings[key]; ok {
			pc = npcColumn{index: i, name: b.Server, kind: b.Kind} // 未标类别 = direct（零值）
		} else if rest, isTable := strings.CutPrefix(key, "table@"); isTable && rest != "" {
			pc = npcColumn{index: i, name: rest, kind: npcColumnTableRec}
		} else {
			pc = npcColumn{index: i, name: key, kind: npcColumnDropped}
		}
		if pc.kind != npcColumnDropped {
			seen[pc.name]++
			if n := seen[pc.name]; n > 1 {
				pc.name = fmt.Sprintf("%s_%d", pc.name, n)
			}
		}
		plan = append(plan, pc)
	}
	return plan
}

// templateRows 一张模板表 → 属性行（口径见文件头）。
func templateRows(t *sharedata.Table) []map[string]string {
	first := t.First()
	plan := planNPCColumns(t)
	idIdx, hasID := first["ID"]
	rows := make([]map[string]string, 0, len(t.Rows))
	for _, cells := range t.Rows {
		id := ""
		if hasID && idIdx < len(cells) {
			id = cells[idIdx]
		}
		if id == "" {
			continue // 无 ID 行（表尾说明 / 排版行里 ID 格空的）不入
		}
		row := make(map[string]string, len(plan))
		row["ID"] = id
		if script, ok := first["script"]; ok && script < len(cells) && cells[script] != "" {
			row["script"] = cells[script]
		}
		if name, ok := first["Name"]; ok && name < len(cells) && cells[name] != "" {
			row["Name"], row["Desc"] = cells[name], cells[name]
		}
		for _, pc := range plan {
			if pc.kind == npcColumnDropped || pc.index >= len(cells) {
				continue
			}
			switch pc.name {
			case "ID", "script", "Name", "Desc":
				continue // 前置项已写
			}
			if value := cells[pc.index]; value != "" {
				row[pc.name] = value
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// loadTemplatesShare 模板树（直读）→ ID → 模板行。表序见
// sharedata.NPCConfigTables（与 XML 产物链的遍历序一致，决定同 ID 多表的
// 胜出行）。单档解析失败跳过（与构建器同：坏档不废整树），返回入表档数
// 供 trace。
func loadTemplatesShare(root string) (map[string]TemplateRow, int, error) {
	tables, err := sharedata.NPCConfigTables(root)
	if err != nil {
		return nil, 0, fmt.Errorf("npcfeed: 扫 %s: %w", sharedata.NPCConfigDir, err)
	}
	templates := map[string]TemplateRow{}
	loaded := 0
	for _, entry := range tables {
		table, parseErr := sharedata.ReadTemplateTable(entry.Path)
		if parseErr != nil {
			continue // 单档解析失败跳过（模板树大，个别脏档不废整树）
		}
		loaded++
		for _, row := range templateRows(table) {
			id := row["ID"]
			if id == "" {
				continue
			}
			if _, dup := templates[id]; dup {
				continue // 同 ID 多表：首档胜出
			}
			templates[id] = TemplateRow{ID: id, Script: row["script"], Attrs: row}
		}
	}
	if len(templates) == 0 {
		return nil, 0, fmt.Errorf("npcfeed: %s 树为空（root 缺席？）", sharedata.NPCConfigDir)
	}
	return templates, loaded, nil
}

// npcTemplateFiles 战斗模板源清单：share/task 下的 npctemplate*.tab——
// npctemplate.tab 精确名主档序前，其余按字典序补充。
func npcTemplateFiles(root string) ([]string, error) {
	matches, err := res.Glob(filepath.Join(root, filepath.FromSlash("share/task"), "npctemplate*.tab"))
	if err != nil {
		return nil, err
	}
	score := func(p string) int {
		if strings.EqualFold(filepath.Base(p), "npctemplate.tab") {
			return 0
		}
		return 1
	}
	sort.SliceStable(matches, func(i, j int) bool { return score(matches[i]) < score(matches[j]) })
	return matches, nil
}

// loadCombatShare 战斗模板（直读）：各档行按主档列头解读，同 ID 首行胜出。
// 口径与构建器 convertNpcTemplate 一致——补充档的行不按自身列头对齐
// （源数据里 npctemplate2.tab 无列键，本就整档跳过）。
func loadCombatShare(root string) (map[string]CombatRow, error) {
	files, err := npcTemplateFiles(root)
	if err != nil {
		return nil, err
	}
	var columns []string
	var rows [][]string
	seen := map[string]bool{}
	for _, path := range files {
		table, loadErr := sharedata.ReadDomainTable(path, 1)
		if loadErr != nil {
			continue // 空壳 / 坏档整档跳过，不废主档
		}
		if len(columns) == 0 {
			columns = table.Columns
		}
		idIdx := table.Column("ID")
		for _, row := range table.Rows {
			id := ""
			if idIdx >= 0 && idIdx < len(row) {
				id = row[idIdx]
			}
			if id != "" {
				if seen[id] {
					continue
				}
				seen[id] = true
			}
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		return nil, os.ErrNotExist
	}
	first := map[string]int{}
	for i, name := range columns {
		if _, ok := first[name]; !ok {
			first[name] = i
		}
	}
	idIdx, hasID := first["ID"]
	combat := make(map[string]CombatRow, len(rows))
	for _, row := range rows {
		if !hasID || idIdx >= len(row) || row[idIdx] == "" {
			continue
		}
		attr := make(map[string]string, len(row))
		attr["ID"] = row[idIdx]
		for i, name := range columns {
			if first[name] != i || name == "ID" || name == "" || i >= len(row) || row[i] == "" {
				continue
			}
			attr[name] = row[i]
		}
		combat[row[idIdx]] = CombatRow{ID: row[idIdx], Attrs: attr}
	}
	return combat, nil
}
