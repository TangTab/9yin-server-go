package itemdata

// definition.go 物品模板行（黑金物品配置格式）的取值面：行属性 + 静态包
// 联查。只有构建产物来源（ini/Item，xml.go）有模板行；设计态 ini 回退
// 只有 Entry 目录面。

import "strings"

// Definition 一件物品的模板行：ID（ConfigID）+ 逻辑类（script）+ 静态包
// 引用（ArtPack/LogicPack/FuncPack… 整型 ID）+ 逻辑类自有属性。
type Definition struct {
	id    string
	attrs map[string]string
	packs map[string]packTable
}

// ID 物品 ConfigID。
func (d *Definition) ID() string { return d.id }

// Has 是否带某属性。
func (d *Definition) Has(key string) bool {
	_, ok := d.attrs[key]
	return ok
}

// Str 串属性（缺席为空）。
func (d *Definition) Str(key string) string { return d.attrs[key] }

// Int 整型属性（缺席/非法为 0）。
func (d *Definition) Int(key string) int {
	n, err := atoi(d.attrs[key])
	if err != nil {
		return 0
	}
	return n
}

// Script 逻辑类（script：服务端对象类，使用链的分派键）。
func (d *Definition) Script() string { return d.attrs["script"] }

// ItemType 物品类型（ItemType）。
func (d *Definition) ItemType() int { return d.Int("ItemType") }

// EquipType 装备栏位类型（武器/防具/时装…）。
func (d *Definition) EquipType() string { return d.attrs["EquipType"] }

// entry 目录面（发放行的 ItemType/分类页/渲染属性来源）。
func (d *Definition) entry() Entry {
	view := d.Int("ViewID")
	if view < 1 || view > 4 {
		// 行内页标缺省或越界（11=内功装备页等落不进四页背包）：装备归
		// 装备页，其余道具页（itemdata.ini 回退同口径）。
		view = 1
		if d.EquipType() != "" || d.Script() == "Equipment" || d.Script() == "ShotWeapon" {
			view = 2
		}
	}
	return Entry{
		ConfigID:   d.id,
		ItemType:   d.ItemType(),
		MaxAmount:  d.Int("MaxAmount"),
		ViewID:     view,
		EquipType:  d.EquipType(),
		ColorLevel: d.Int("ColorLevel"),
		Photo:      d.attrs["Photo"],
		Name:       d.attrs["Name"],
		Script:     d.Script(),
	}
}

// Pack 静态包行（ref = 物品行上的引用键，如 "FuncPack"；引用为空/0 或包
// 内无行 → false）。
func (d *Definition) Pack(ref string) (map[string]string, bool) {
	id, ok := d.attrs[ref]
	if !ok || id == "" || id == "0" {
		return nil, false
	}
	table, ok := d.packs[ref]
	if !ok {
		return nil, false
	}
	row, ok := table[id]
	return row, ok
}

// ---- 类型化静态包视图 ----

// FuncPack 功能包行（item_func_static）：PrepareTime/PrepareAction = 使用
// 读条时长/动作（官服 m25[PrepareTime,"flow_prepare","useitem_flow"] 实测
// 吻合）；LuaScript = 官服物品脚本路径（脚本本体未随资源发布）。
type FuncPack struct {
	ID                int
	LuaScript         string
	PrepareAction     string
	PrepareTimeMS     int
	RandomEventPackID string
}

// FuncPack 功能包联查（物品带 FuncPack 且功能包表内有行）。
func (d *Definition) FuncPack() (FuncPack, bool) {
	row, ok := d.Pack(packFunc)
	if !ok {
		return FuncPack{}, false
	}
	return FuncPack{
		ID:                d.Int(packFunc),
		LuaScript:         row["LuaScript"],
		PrepareAction:     row["PrepareAction"],
		PrepareTimeMS:     atoiSafe(row["PrepareTime"]),
		RandomEventPackID: row["RandomEventPackID"],
	}, true
}

// UseSkillPack 使用技能包行（药品=使用技能：SkillID/SkillLevel）。
type UseSkillPack struct {
	ID         int
	SkillID    string
	SkillLevel int
}

// UseSkill 使用技能包联查。
func (d *Definition) UseSkill() (UseSkillPack, bool) {
	row, ok := d.Pack(packUseSkill)
	if !ok {
		return UseSkillPack{}, false
	}
	return UseSkillPack{ID: d.Int(packUseSkill), SkillID: row["SkillID"], SkillLevel: atoiSafe(row["SkillLevel"])}, true
}

// LearnTargets 学习技能包的目标技能（TragetID——官方拼写；逗号/分号混
// 隔；包内无行或无目标为 nil）。
func (d *Definition) LearnTargets() []string {
	row, ok := d.Pack(packLearnSkill)
	if !ok {
		return nil
	}
	return splitList(row["TragetID"])
}

// ArtPack 美术静态包行（图标 + 男女模型 + 动作集）。
type ArtPack struct {
	ID                     int
	Photo, MaleModel       string
	FemaleModel, ActionSet string
}

// Art 美术静态包联查。
func (d *Definition) Art() (ArtPack, bool) {
	row, ok := d.Pack(packArt)
	if !ok {
		return ArtPack{}, false
	}
	return ArtPack{
		ID: d.Int(packArt), Photo: row["Photo"], MaleModel: row["MaleModel"],
		FemaleModel: row["FemaleModel"], ActionSet: row["ActionSet"],
	}, true
}

// LogicPack 逻辑包行（交易/仓库/删除限制族）。
type LogicPack struct {
	ID                             int
	BindType, SpecialItem          int
	CantDepot, CantExchange        int
	CantDelete, CantSell, CantMove int
}

// Logic 逻辑包联查。
func (d *Definition) Logic() (LogicPack, bool) {
	row, ok := d.Pack(packLogic)
	if !ok {
		return LogicPack{}, false
	}
	return LogicPack{
		ID: d.Int(packLogic), BindType: atoiSafe(row["BindType"]), SpecialItem: atoiSafe(row["SpecialItem"]),
		CantDepot: atoiSafe(row["CantDepot"]), CantExchange: atoiSafe(row["CantExchange"]),
		CantDelete: atoiSafe(row["CantDelete"]), CantSell: atoiSafe(row["CantSell"]), CantMove: atoiSafe(row["CantMove"]),
	}, true
}

// splitList 逗号/分号混隔列表（过滤空段）。
func splitList(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' })
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if field = strings.TrimSpace(field); field != "" {
			out = append(out, field)
		}
	}
	return out
}
