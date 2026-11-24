// mapping.go 客户端列键 → 服务端属性名的审定映射表（本工具的核心）。
//
// 服务端词汇来源（三处并集，均为仓库现有真档/schema）：
//   - testdata/ini/Npc 黑金档（common_npc*.xml 90 属性 ∪ member0 26 属性）
//   - testdata/struct/npc/*.xml（VisibleNpc 及其 6 个 include）
//
// 分类（审计报告四类）：
//   direct   两边同名同义，直通
//   rename   改名映射（注释给 review 依据）
//   tableRec table@ 前缀引用列，去前缀保留（撞名加序号）
//   dropped  未映射，不进输出，仅记录（附中文列注释，便于后续扩词汇）
//
// 只列人工核对过的绑定，不猜：没把握的宁可 dropped 也不 invent。
package npc

import (
	"fmt"
	"strings"
)

// binding 一条列绑定。
type binding struct {
	Server string // 服务端属性名
	Kind   string // direct | rename | tableRec
	Note   string // rename 的 review 依据
}

// columnBindings 审定映射表。键为客户端表第 3 行列键。
var columnBindings = map[string]binding{
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

	// ---- direct：NPC 业务功能列（2433 modernNPCServices 同名方言；源
	// 表中文列注释：商店ID/仓库ID/职业ID/宝石游戏NPC ID/驿站/落脚点/
	// 是否有公会功能——2026-11-23 商店/仓库 View 接入定案）----
	"ShopID":       {Server: "ShopID"},
	"DepotID":      {Server: "DepotID"},
	"Job":          {Server: "Job"},
	"TransFuncID":  {Server: "TransFuncID"},
	"HomePoint":    {Server: "HomePoint"},
	"GuildFunc":    {Server: "GuildFunc"},
	"GemGameNpcID": {Server: "GemGameNpcID"},

	// ---- direct：NPC 业报列（源表中文列注释：善恶参数/地位评分——
	// 2026-11-24 对话通用行门控定案：官服只对 Character 有值且 KarmaLevel≥1
	// 的 NPC 发「赠送物品/侠恶查询」两行）。npcfeed 0x28 尾部常量族按同名
	// 读取，入表后这两个字段按模板真值下发（原恒 1/3）。----
	"Character":  {Server: "Character"},
	"KarmaLevel": {Server: "KarmaLevel"},

	// ---- rename：改名映射 ----
	"PathID": {Server: "PathId", Kind: "rename", Note: "大小写对齐黑金档 PathId（巡逻路径 ID）"},
	"FacultyExp": {Server: "Exp", Kind: "rename",
		Note: "九阴修为经验 → 黑金经验列 Exp（VisibleNpc 无独立属性，common_npc.xml 真档用法）"},
	"DropID": {Server: "WorldDropID", Kind: "rename", Note: "九阴掉落 → 黑金世界掉落列"},
	"SelfForceID": {Server: "Force", Kind: "rename",
		Note: "势力归属 → Force（黑金 common_npc.xml 真档列，语义一致）"},
	"PowerNo": {Server: "FightID", Kind: "rename",
		Note: "review：实力包编号 → FightID。黑金 FightID 形如 ZS_PT_0030，九阴 PowerNo 为实力包引用，量纲存疑，先按语义近邻绑定"},
	"table@SkillRec": {Server: "SkillPackID", Kind: "rename",
		Note: "review：技能表引用 → SkillPackID。黑金 SkillPackID 形如 SkillPack3_1，值域需上线前抽查"},
}

// tablePlan 一张表的列输出计划（列序保持源表顺序）。
type tablePlan struct {
	Columns []plannedColumn
}

// plannedColumn 一条输出列。
type plannedColumn struct {
	SourceIndex int
	SourceName  string // 客户端列键（含 table@ 前缀，报告溯源用）
	OutAttr     string // 输出属性名
	Kind        string // direct | rename | tableRec | dropped
}

// planTable 为一张表计算输出计划：审定绑定优先；未绑定但带 table@ 前缀的
// 列去前缀保留（输出属性撞名时追加 _2/_3）；其余 dropped。
// 同名列取首见（返回重复列清单供报告）。
func planTable(t *npcTable) (*tablePlan, []string) {
	dups := t.duplicateColumns()
	first := t.firstIndex()

	seen := map[string]int{} // 输出属性名 → 已占用计数
	plan := &tablePlan{}
	// 保持源列序：按下标遍历，仅处理每个列键的首见列。
	for i, name := range t.Columns {
		if first[name] != i {
			continue
		}
		var pc plannedColumn
		if b, ok := columnBindings[name]; ok {
			pc = plannedColumn{SourceIndex: i, SourceName: name, OutAttr: b.Server, Kind: b.Kind}
			if pc.Kind == "" {
				pc.Kind = "direct"
			}
		} else if rest, isTable := strings.CutPrefix(name, "table@"); isTable && rest != "" {
			pc = plannedColumn{SourceIndex: i, SourceName: name, OutAttr: rest, Kind: "tableRec"}
		} else {
			pc = plannedColumn{SourceIndex: i, SourceName: name, Kind: "dropped"}
		}
		if pc.Kind != "dropped" {
			seen[pc.OutAttr]++
			if n := seen[pc.OutAttr]; n > 1 {
				pc.OutAttr = fmt.Sprintf("%s_%d", pc.OutAttr, n)
			}
		}
		plan.Columns = append(plan.Columns, pc)
	}
	return plan, dups
}

// droppedNote dropped 列的报告条目（列键 + 中文注释 + 声明类型）。
func droppedNote(t *npcTable, pc plannedColumn) string {
	comment := ""
	if pc.SourceIndex < len(t.Comments) {
		comment = t.Comments[pc.SourceIndex]
	}
	declared := ""
	if pc.SourceIndex < len(t.Types) {
		declared = t.Types[pc.SourceIndex]
	}
	return fmt.Sprintf("%s\t%s\t%s", pc.SourceName, comment, declared)
}
