package npcfeed

// resolve.go 分层属性合成（2433 ResolvedNPC.OrderedProperties 配方）：
//
//	默认值层（2433 VisibleNPCSpawnDefaultsCompatV1 值集，按当前 0x09 表
//	场景域过滤缺席名/线型不符名）
//	  → 模板层（NpcConfig XML 行按官方场景域字段名同名直连，
//	     ConfigID→ID 别名、ID 结构列不直连；单字段脏值跳过不废整条）
//	  → 战斗层（NpcTemplate 行：MaxLife→MaxHP/HP、MaxMana→MaxMP、
//	     Level 真值；表缺席名跳过）
//	  → 摆放层（PosiX/Y/Z/Orient——表尾追段场景域有名才注入）
//
// 输出按 0x09 官方表场景域表序（每名首现），供 0x0D 头行块编码。

import (
	"math"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

// PropValue 按线型的单值（编码不自描述——宽度由 0x09 表条目决定）。
type PropValue struct {
	Type clientdata.WireType
	U8   uint8
	U16  uint16
	I32  int32
	U64  uint64
	F32  float32
	Text string
}

// Prop 一条带表索引的属性值（0x0D 头行块载体）。
type Prop struct {
	Index uint16
	Name  string
	Value PropValue
}

func byteVal(v uint8) PropValue  { return PropValue{Type: clientdata.WireByte, U8: v} }
func wordVal(v uint16) PropValue { return PropValue{Type: clientdata.WireWord, U16: v} }
func int32Val(v int32) PropValue { return PropValue{Type: clientdata.WireInt32, I32: v} }
func floatVal(v float32) PropValue {
	return PropValue{Type: clientdata.WireFloat32, F32: v}
}
func stringVal(v string) PropValue {
	return PropValue{Type: clientdata.WireString, Text: v}
}
func wideVal(v string) PropValue {
	return PropValue{Type: clientdata.WireWideString, Text: v}
}

// spawnDefaults 默认值层（2433 VisibleNPCSpawnDefaultsCompatV1 直抄；
// 装配时按场景域过滤缺席名/线型不符名）。
var spawnDefaults = map[string]PropValue{
	"Type":             byteVal(4),
	"Sex":              byteVal(0),
	"ActionSet":        stringVal(""),
	"State":            stringVal("stand"),
	"LogicState":       byteVal(0),
	"HPRatio":          int32Val(100),
	"MaxHP":            int32Val(1000),
	"HP":               int32Val(1000),
	"Scale":            stringVal(""),
	"CollideType":      int32Val(0),
	"CantShowHeadInfo": byteVal(0),
	"NPCCanSelect":     wordVal(1),
	"RotatePara":       stringVal(""),
	"NeedRotate":       int32Val(0),
	"Collide":          int32Val(0),
	"CollideRadius":    floatVal(0.905),
	"CantAttack":       int32Val(0),
	"CantBeAttack":     int32Val(0),
	"Force":            int32Val(0),
	"MotionNoRotate":   byteVal(0),
	"CanPick":          byteVal(0),
	"VisDataInt":       int32Val(0),
	"VisDataStr":       wideVal(""),
	"HeadEffect":       int32Val(0),
	"NpcNickName":      wideVal(""),
	"RoleHide":         int32Val(0),
	"SpringEffect":     stringVal(""),
	"Material":         stringVal(""),
	"WeaponName":       stringVal(""),
	"BornAction":       stringVal(""),
	"ActionSoundMode":  int32Val(0),
	"Fixed":            int32Val(0),
	"FixedBeHitAct":    stringVal(""),
	"DirectSpring":     int32Val(0),
	"NpcTalkType":      int32Val(0),
	"PreLoad":          int32Val(0),
	"OnlyCareDistance": int32Val(0),
	"RefreshAtOnce":    int32Val(0),
}

// OrderedProperties 头行块合成（按 0x09 官方表场景域表序输出）。
func (s Spawn) OrderedProperties() []Prop {
	props := map[string]PropValue{}

	// 默认值层：按场景域过滤（缺席名与线型不符名不注入）。
	for name, value := range spawnDefaults {
		if entry, ok := clientdata.SceneProperty(name); ok && entry.Type == value.Type {
			props[name] = value
		}
	}

	// 模板层：XML 行同名列绑定（ConfigID→ID 别名、ID 结构列不直连）。
	// 真档含脏值（如 WalkSpeed="#N/A"）——单字段解析失败跳过（默认层兜底）。
	for _, field := range clientdata.OfficialSceneFields() {
		if field.Name == "ID" {
			continue
		}
		column := field.Name
		if field.Name == "ConfigID" {
			column = "ID"
		}
		raw, ok := s.Template[column]
		if !ok || raw == "" {
			continue
		}
		if value, err := parseSceneValue(field, raw); err == nil {
			props[field.Name] = value
		}
	}

	// 战斗层：NpcTemplate 行真值（FightID 未命中整层跳过）。
	if s.Combat != nil {
		applyCombatLayer(props, *s.Combat)
	}

	// 摆放层：坐标四行（尾追表 PosiX/Orient——场景域有名才注入）。
	placement := map[string]PropValue{
		"PosiX":  floatVal(s.X),
		"PosiY":  floatVal(s.Y),
		"PosiZ":  floatVal(s.Z),
		"Orient": floatVal(s.Orient),
	}
	for name, value := range placement {
		if entry, ok := clientdata.SceneProperty(name); ok && entry.Type == value.Type {
			props[name] = value
		}
	}

	// 按表序输出。
	out := make([]Prop, 0, len(props))
	for _, field := range clientdata.OfficialSceneFields() {
		if value, ok := props[field.Name]; ok {
			out = append(out, Prop{Index: field.Index, Name: field.Name, Value: value})
		}
	}
	return out
}

// applyCombatLayer 战斗层映射（值域合法才覆盖；目标名缺席官方表则跳过）。
func applyCombatLayer(props map[string]PropValue, row CombatRow) {
	if maxLife := row.Int("MaxLife"); maxLife > 0 {
		setIfField(props, "MaxHP", int32Val(int32(maxLife)))
		setIfField(props, "HP", int32Val(int32(maxLife)))
	}
	if maxMana := row.Int("MaxMana"); maxMana > 0 {
		setIfField(props, "MaxMP", int32Val(int32(maxMana)))
	}
	if level := row.Int("Level"); level > 0 && level < 65536 {
		setIntByField(props, "Level", int32(level))
	}
}

func setIfField(props map[string]PropValue, name string, value PropValue) {
	if entry, ok := clientdata.SceneProperty(name); ok && entry.Type == value.Type {
		props[name] = value
	}
}

// setIntByField 整数值按表线型分派（byte/word/int32；其它线型跳过）。
func setIntByField(props map[string]PropValue, name string, value int32) {
	entry, ok := clientdata.SceneProperty(name)
	if !ok {
		return
	}
	switch entry.Type {
	case clientdata.WireByte:
		if value >= 0 && value < 256 {
			props[name] = byteVal(uint8(value))
		}
	case clientdata.WireWord:
		if value >= 0 && value < 65536 {
			props[name] = wordVal(uint16(value))
		}
	case clientdata.WireInt32:
		props[name] = int32Val(value)
	}
}

// parseSceneValue 按表线型解析模板/摆放原始串。
func parseSceneValue(field clientdata.PropertyEntry, raw string) (PropValue, error) {
	text := strings.TrimSpace(raw)
	switch field.Type {
	case clientdata.WireByte:
		v, err := strconv.ParseUint(text, 10, 8)
		if err != nil {
			return PropValue{}, err
		}
		return byteVal(uint8(v)), nil
	case clientdata.WireWord:
		v, err := strconv.ParseUint(text, 10, 16)
		if err != nil {
			return PropValue{}, err
		}
		return wordVal(uint16(v)), nil
	case clientdata.WireInt32:
		v, err := parseIntegralInt32(text)
		if err != nil {
			return PropValue{}, err
		}
		return int32Val(v), nil
	case clientdata.WireFloat32:
		v, err := strconv.ParseFloat(text, 32)
		if err != nil {
			return PropValue{}, err
		}
		f := float32(v)
		if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) {
			return PropValue{}, strconv.ErrSyntax
		}
		return floatVal(f), nil
	case clientdata.WireString:
		return stringVal(raw), nil
	case clientdata.WireWideString:
		return wideVal(raw), nil
	default:
		return PropValue{}, strconv.ErrSyntax
	}
}

func parseIntegralInt32(text string) (int32, error) {
	if v, err := strconv.ParseInt(text, 10, 32); err == nil {
		return int32(v), nil
	}
	v, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || math.Trunc(v) != v ||
		v < math.MinInt32 || v > math.MaxInt32 {
		return 0, strconv.ErrSyntax
	}
	return int32(v), nil
}
