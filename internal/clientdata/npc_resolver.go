package clientdata

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

type Value struct {
	Type WireType
	U8   uint8
	U16  uint16
	I32  int32
	F32  float32
	U64  uint64
	Text string
}

func ByteValue(value uint8) Value        { return Value{Type: WireByte, U8: value} }
func WordValue(value uint16) Value       { return Value{Type: WireWord, U16: value} }
func Int32Value(value int32) Value       { return Value{Type: WireInt32, I32: value} }
func Int64Value(value int64) Value       { return Value{Type: WireInt64, U64: uint64(value)} }
func Float32Value(value float32) Value   { return Value{Type: WireFloat32, F32: value} }
func StringValue(value string) Value     { return Value{Type: WireString, Text: value} }
func WideStringValue(value string) Value { return Value{Type: WireWideString, Text: value} }
func ObjectValue(objectID, ownerID uint32) Value {
	return Value{Type: WireObject, U64: uint64(objectID) | uint64(ownerID)<<32}
}

func (v Value) Native() any {
	switch v.Type {
	case WireByte:
		return v.U8
	case WireWord:
		return v.U16
	case WireInt32:
		return v.I32
	case WireInt64:
		return int64(v.U64)
	case WireFloat32:
		return v.F32
	case WireString, WireWideString:
		return v.Text
	case WireObject:
		return v.U64
	default:
		return nil
	}
}

type Provenance struct {
	Layer  string
	Source string
	Record int
	Field  string
}

type PropertyValue struct {
	Value      Value
	Provenance Provenance
}

type ResolveOptions struct {
	Defaults  map[string]Value
	Runtime   map[string]Value
	Persisted map[string]Value
}

type ResolvedNPC struct {
	InstanceKey string
	ConfigID    string
	ScriptClass string
	Transform   NPCTransform
	Properties  map[string]PropertyValue
	Extensions  map[string]string
}

type IndexedProperty struct {
	Index uint16
	Name  string
	Value Value
}

// OrderedProperties 按官方属性表顺序(每名取首现,即场景域)产出已解析属性。
// 官方方言不认识的名字(旧兼容私有属性)会被静默丢弃——0x09 表是客户端
// 位置绑定的唯一真源,缺席名字无法编码下发。
func (r ResolvedNPC) OrderedProperties() ([]IndexedProperty, error) {
	result := make([]IndexedProperty, 0, len(r.Properties))
	for _, field := range officialSceneFields() {
		property, exists := r.Properties[field.Name]
		if !exists {
			continue
		}
		if property.Value.Type != field.Type {
			return nil, fmt.Errorf("clientdata: resolved NPC property %q type=%s, official table wants %s", field.Name, property.Value.Type, field.Type)
		}
		result = append(result, IndexedProperty{Index: field.Index, Name: field.Name, Value: property.Value})
	}
	return result, nil
}

type templateColumnBinding struct {
	Name       string
	Occurrence int
}

var npcTemplateAliases = map[string]templateColumnBinding{
	"ConfigID": {Name: "ID"},
	"Sex":      {Name: "Gender"},
	// 新版表格中 StopTime 重复出现。NPC 战斗停止时间对应的是
	// 第二次出现（物理列 87，从零开始计数）。
	"StopTime": {Name: "StopTime", Occurrence: 1},
}

// npcTemplateStructuralFields 是模板表的结构列,只经由 npcTemplateAliases
// 审定绑定(如 ConfigID→ID),不参与官方表同名直连——官方表玩家块的
// "ID"@1462 是 Int32,直连会把字符串型模板 ID 列当整数解析而失败。
var npcTemplateStructuralFields = map[string]struct{}{
	"ID": {},
}

var creatorStructuralAttributes = map[string]struct{}{
	"No": {}, "id": {},
	"x": {}, "y": {}, "z": {},
	"ax": {}, "ay": {}, "az": {},
	"sx": {}, "sy": {}, "sz": {},
}

func ResolveNPC(table *NPCTemplateTable, instance NPCCreatorInstance, options ResolveOptions) (ResolvedNPC, error) {
	if table == nil {
		return ResolvedNPC{}, fmt.Errorf("clientdata: resolve NPC %q with nil template table", instance.TemplateID)
	}
	if OfficialPropertyCount() == 0 {
		return ResolvedNPC{}, errorsf("resolve NPC %q with empty official property table", instance.TemplateID)
	}
	row, err := table.Lookup(instance.TemplateID)
	if err != nil {
		return ResolvedNPC{}, err
	}
	resolved := ResolvedNPC{
		InstanceKey: instance.InstanceKey(),
		ConfigID:    instance.TemplateID,
		ScriptClass: row.ScriptClass,
		Transform:   instance.Transform,
		Properties:  make(map[string]PropertyValue),
		Extensions:  make(map[string]string),
	}
	if err := applyValueLayer(resolved.Properties, options.Defaults, Provenance{Layer: "default", Source: "official-property-table"}); err != nil {
		return ResolvedNPC{}, err
	}

	consumedTemplate := make(map[int]struct{})
	for _, field := range officialSceneFields() {
		if _, structural := npcTemplateStructuralFields[field.Name]; structural {
			continue
		}
		binding, explicit := npcTemplateAliases[field.Name]
		if !explicit {
			binding.Name = field.Name
		}
		indices := table.ColumnIndices(binding.Name)
		if len(indices) == 0 {
			continue
		}
		if !explicit && len(indices) != 1 {
			// 重名需要经过人工审定的绑定。绝不猜测取第一个或最后一个。
			continue
		}
		// 城市表只包含战斗 StopTime 列，而门派表
		// 还包含一个更早出现的移动 StopTime 列。审定绑定的策略是：
		// 第二次出现存在时选用第二次出现，
		// 否则选用唯一的那次出现。
		occurrence := binding.Occurrence
		if explicit && binding.Name == "StopTime" && occurrence == 1 && len(indices) == 1 {
			occurrence = 0
		}
		if occurrence < 0 || occurrence >= len(indices) {
			return ResolvedNPC{}, fmt.Errorf("clientdata: NPC template binding %q occurrence %d not present in %s", binding.Name, binding.Occurrence, table.Source)
		}
		columnIndex := indices[occurrence]
		consumedTemplate[columnIndex] = struct{}{}
		cell := row.Cells[columnIndex]
		if !cell.Set {
			continue
		}
		value, parseErr := parseWireValue(field.Type, cell.Raw)
		if parseErr != nil {
			return ResolvedNPC{}, fmt.Errorf("clientdata: NPC template %s logical record %d field %s=%q: %w", row.Source, row.LogicalRecord, cell.ColumnName, cell.Raw, parseErr)
		}
		resolved.Properties[field.Name] = PropertyValue{Value: value, Provenance: Provenance{Layer: "template", Source: row.Source, Record: row.LogicalRecord, Field: cell.ColumnName}}
	}

	consumedCreator := make(map[string]struct{})
	for _, field := range officialSceneFields() {
		raw, exists := instance.Attributes[field.Name]
		if !exists || !raw.Present {
			continue
		}
		value, parseErr := parseWireValue(field.Type, raw.Raw)
		if parseErr != nil {
			return ResolvedNPC{}, fmt.Errorf("clientdata: NPC creator %s item %d field %s=%q: %w", instance.Source, instance.Ordinal, field.Name, raw.Raw, parseErr)
		}
		resolved.Properties[field.Name] = PropertyValue{Value: value, Provenance: Provenance{Layer: "creator", Source: instance.Source, Record: instance.Ordinal, Field: field.Name}}
		consumedCreator[field.Name] = struct{}{}
	}

	// 摆放坐标是由 creator 数据派生的运行时对象状态，
	// 不是模板属性。下方显式传入的运行时值可以覆盖它们。
	placement := map[string]Value{
		"PosiX": Float32Value(instance.Transform.X),
		"PosiY": Float32Value(instance.Transform.Y),
		"PosiZ": Float32Value(instance.Transform.Z),
	}
	if instance.Transform.AY != nil {
		placement["Orient"] = Float32Value(*instance.Transform.AY)
	}
	// 摆放坐标的权威在 resolved.Transform；属性表副本仅服务 fork 方言
	// 0x0D 属性行（PosiX/Y/Z/Orient 为 fork-only 名）。官服方言下按名
	// 缺席即不注入（0x0D 坐标走固定头，§14.3 场景域处置同策）。
	wirePlacement := make(map[string]Value, len(placement))
	for name, value := range placement {
		if field, ok := SceneProperty(name); ok && field.Type == value.Type {
			wirePlacement[name] = value
		}
	}
	if err := applyValueLayer(resolved.Properties, wirePlacement, Provenance{Layer: "runtime", Source: instance.Source, Record: instance.Ordinal, Field: "transform"}); err != nil {
		return ResolvedNPC{}, err
	}
	if err := applyValueLayer(resolved.Properties, options.Runtime, Provenance{Layer: "runtime", Source: "resolve-options"}); err != nil {
		return ResolvedNPC{}, err
	}
	if err := applyValueLayer(resolved.Properties, options.Persisted, Provenance{Layer: "persisted", Source: "resolve-options"}); err != nil {
		return ResolvedNPC{}, err
	}

	for index, cell := range row.Cells {
		if !cell.Set {
			continue
		}
		if _, consumed := consumedTemplate[index]; consumed {
			continue
		}
		key := "template." + cell.ColumnName
		if len(table.ColumnIndices(cell.ColumnName)) > 1 {
			key = fmt.Sprintf("%s[%d]", key, occurrenceAt(table.ColumnIndices(cell.ColumnName), index))
		}
		resolved.Extensions[key] = cell.Raw
	}
	for name, raw := range instance.Attributes {
		if _, consumed := consumedCreator[name]; consumed {
			continue
		}
		if _, structural := creatorStructuralAttributes[name]; structural {
			continue
		}
		resolved.Extensions["creator."+name] = raw.Raw
	}
	return resolved, nil
}

func errorsf(format string, args ...any) error {
	return fmt.Errorf("clientdata: "+format, args...)
}

func occurrenceAt(indices []int, target int) int {
	for occurrence, index := range indices {
		if index == target {
			return occurrence
		}
	}
	return -1
}

func applyValueLayer(target map[string]PropertyValue, values map[string]Value, provenance Provenance) error {
	keys := make([]string, 0, len(values))
	for name := range values {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		value := values[name]
		field, exists := SceneProperty(name)
		if !exists {
			return fmt.Errorf("clientdata: layer %s sets property %q absent from official property table", provenance.Layer, name)
		}
		if value.Type != field.Type {
			return fmt.Errorf("clientdata: layer %s property %q type=%s, official table wants %s", provenance.Layer, name, value.Type, field.Type)
		}
		if value.Type == WireFloat32 && (math.IsNaN(float64(value.F32)) || math.IsInf(float64(value.F32), 0)) {
			return fmt.Errorf("clientdata: layer %s property %q is not finite", provenance.Layer, name)
		}
		propertyProvenance := provenance
		propertyProvenance.Field = name
		target[name] = PropertyValue{Value: value, Provenance: propertyProvenance}
	}
	return nil
}

func parseWireValue(wireType WireType, raw string) (Value, error) {
	switch wireType {
	case WireByte:
		value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 8)
		if err != nil {
			return Value{}, err
		}
		return ByteValue(uint8(value)), nil
	case WireWord:
		value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 16)
		if err != nil {
			return Value{}, err
		}
		return WordValue(uint16(value)), nil
	case WireInt32:
		value, err := parseIntegralInt32(raw)
		if err != nil {
			return Value{}, err
		}
		return Int32Value(value), nil
	case WireFloat32:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 32)
		if err != nil {
			return Value{}, err
		}
		value := float32(parsed)
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return Value{}, fmt.Errorf("not a finite float32")
		}
		return Float32Value(value), nil
	case WireString:
		return StringValue(raw), nil
	case WireWideString:
		return WideStringValue(raw), nil
	default:
		return Value{}, fmt.Errorf("unsupported wire type %d", wireType)
	}
}

func parseIntegralInt32(raw string) (int32, error) {
	text := strings.TrimSpace(raw)
	if value, err := strconv.ParseInt(text, 10, 32); err == nil {
		return int32(value), nil
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value || value < math.MinInt32 || value > math.MaxInt32 {
		return 0, fmt.Errorf("%q is not an integral int32", raw)
	}
	return int32(value), nil
}

// VisibleNPCSpawnDefaultsCompatV1 单独收纳了原先在 protocol-probe 中
// 与模板值混在一起的兼容性默认值。新版客户端数据与 creator
// 属性仍然会覆盖这份默认配置。
//
// 官方属性表迁移(2026-11):旧兼容私有名 VisType/ExtraInfo/EscortFlag
// 在官方 2370 项表中不存在,无法编码下发,已从默认值移除。
func VisibleNPCSpawnDefaultsCompatV1() map[string]Value {
	defaults := map[string]Value{
		"Type":             ByteValue(4),
		"Sex":              ByteValue(0),
		"ActionSet":        StringValue(""),
		"State":            StringValue("stand"),
		"LogicState":       ByteValue(0),
		"HPRatio":          Int32Value(100),
		"MaxHP":            Int32Value(1000),
		"HP":               Int32Value(1000),
		"Scale":            StringValue(""),
		"CollideType":      Int32Value(0),
		"CantShowHeadInfo": ByteValue(0),
		"NPCCanSelect":     WordValue(1),
		"RotatePara":       StringValue(""),
		"NeedRotate":       Int32Value(0),
		"Collide":          Int32Value(0),
		"CollideRadius":    Float32Value(0.905),
		"CantAttack":       Int32Value(0),
		"CantBeAttack":     Int32Value(0),
		"Force":            Int32Value(0),
		"MotionNoRotate":   ByteValue(0),
		"CanPick":          ByteValue(0),
		"VisDataInt":       Int32Value(0),
		"VisDataStr":       WideStringValue(""),
		"HeadEffect":       Int32Value(0),
		"NpcNickName":      WideStringValue(""),
		"RoleHide":         Int32Value(0),
		"SpringEffect":     StringValue(""),
		"Material":         StringValue(""),
		"WeaponName":       StringValue(""),
		"BornAction":       StringValue(""),
		"ActionSoundMode":  Int32Value(0),
		"Fixed":            Int32Value(0),
		"FixedBeHitAct":    StringValue(""),
		"DirectSpring":     Int32Value(0),
		"NpcTalkType":      Int32Value(0),
		"PreLoad":          Int32Value(0),
		"OnlyCareDistance": Int32Value(0),
		"RefreshAtOnce":    Int32Value(0),
	}
	// 0x09 表方言门控（docs 官服登录帧对照 §14.3）：默认层按当前表场景域
	// （首现）过滤——缺席名（fork-only，官服表无此行）与线型不符名
	//（ActionSoundMode fork 场景位=int32、官服唯一现=string）一律不注入，
	// 避免 ResolveNPC 类型校验硬错。2026-11-10 真机：官服方言下该层
	// 5413/5413 实例尽没于 ActionSoundMode 类型错配→NPC 目录空→选角链
	// 断连。fork 表下全部条目类型吻合，行为逐位不变。
	filtered := make(map[string]Value, len(defaults))
	for name, value := range defaults {
		field, ok := SceneProperty(name)
		if !ok || field.Type != value.Type {
			continue
		}
		filtered[name] = value
	}
	return filtered
}
