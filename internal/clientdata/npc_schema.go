package clientdata

import (
	"errors"
	"fmt"
)

// WireType is the type tag used by the visible-property schema negotiated in
// the scene bootstrap. Its values intentionally match the current protocol.
type WireType uint8

const (
	WireByte       WireType = 1
	WireWord       WireType = 2
	WireInt32      WireType = 3
	WireFloat32    WireType = 5
	WireString     WireType = 7
	WireWideString WireType = 8
	// WireObject is the native 64-bit scene-object reference.  The client
	// uses it for properties such as the player's LastObject selection.
	WireObject WireType = 9
)

func (t WireType) String() string {
	switch t {
	case WireByte:
		return "byte"
	case WireWord:
		return "word"
	case WireInt32:
		return "int32"
	case WireFloat32:
		return "float32"
	case WireString:
		return "string"
	case WireWideString:
		return "widestr"
	case WireObject:
		return "object"
	default:
		return fmt.Sprintf("wire-type-%d", t)
	}
}

type FieldSpec struct {
	Index uint16
	Name  string
	Type  WireType
}

type Schema struct {
	ID     string
	Fields []FieldSpec
	byName map[string]FieldSpec
}

func NewSchema(id string, fields []FieldSpec) (Schema, error) {
	if id == "" {
		return Schema{}, errors.New("clientdata: NPC schema has no ID")
	}
	result := Schema{ID: id, Fields: append([]FieldSpec(nil), fields...), byName: make(map[string]FieldSpec, len(fields))}
	for i, field := range result.Fields {
		if field.Name == "" {
			return Schema{}, fmt.Errorf("clientdata: NPC schema %q field %d has no name", id, i)
		}
		if int(field.Index) != i {
			return Schema{}, fmt.Errorf("clientdata: NPC schema %q field %q index=%d, want %d", id, field.Name, field.Index, i)
		}
		if _, exists := result.byName[field.Name]; exists {
			return Schema{}, fmt.Errorf("clientdata: NPC schema %q repeats field %q", id, field.Name)
		}
		switch field.Type {
		case WireByte, WireWord, WireInt32, WireFloat32, WireString, WireWideString, WireObject:
		default:
			return Schema{}, fmt.Errorf("clientdata: NPC schema %q field %q has unsupported type %d", id, field.Name, field.Type)
		}
		result.byName[field.Name] = field
	}
	return result, nil
}

func (s Schema) Field(name string) (FieldSpec, bool) {
	field, ok := s.byName[name]
	return field, ok
}

func mustSchema(id string, fields []FieldSpec) Schema {
	schema, err := NewSchema(id, fields)
	if err != nil {
		panic(err)
	}
	return schema
}

var visibleNPCCompatV1 = mustSchema("visible-npc-compat-v1", []FieldSpec{
	{0, "Type", WireByte},
	{1, "Sex", WireByte},
	{2, "Photo", WireString},
	{3, "Camp", WireInt32},
	{4, "Race", WireInt32},
	{5, "Job", WireInt32},
	{6, "Level", WireInt32},
	{7, "ConfigID", WireString},
	{8, "Resource", WireString},
	{9, "PosiX", WireFloat32},
	{10, "PosiY", WireFloat32},
	{11, "PosiZ", WireFloat32},
	{12, "Orient", WireFloat32},
	{13, "Hair", WireString},
	{14, "Face", WireString},
	{15, "Cloth", WireString},
	{16, "Pants", WireString},
	{17, "Shoes", WireString},
	{18, "ActionSet", WireString},
	{19, "State", WireString},
	{20, "FloatingState", WireInt32},
	{21, "LogicState", WireByte},
	{22, "JumpSpeed", WireFloat32},
	{23, "MoveSpeed", WireFloat32},
	{24, "WalkSpeed", WireFloat32},
	{25, "RunSpeed", WireFloat32},
	{26, "SpeedRatio", WireInt32},
	{27, "CantJump", WireByte},
	{28, "HPRatio", WireInt32},
	{29, "MPRatio", WireInt32},
	{30, "MaxHP", WireInt32},
	{31, "MaxMP", WireInt32},
	{32, "HP", WireInt32},
	{33, "MP", WireInt32},
	{34, "NpcType", WireByte},
	{35, "Hat", WireString},
	{36, "Name", WireWideString},
	{37, "Scale", WireString},
	{38, "SelectState", WireByte},
	{39, "CollideType", WireInt32},
	{40, "CantShowHeadInfo", WireByte},
	{41, "NPCCanSelect", WireWord},
	{42, "WeaponMode", WireString},
	{43, "RotatePara", WireString},
	{44, "NeedRotate", WireInt32},
	{45, "Collide", WireInt32},
	{46, "CursorShape", WireByte},
	{47, "CollideRadius", WireFloat32},
	{48, "LuaScript", WireString},
	{49, "AITemplate", WireInt32},
	{50, "PatrolMode", WireInt32},
	{51, "PatrolRange", WireFloat32},
	{52, "SpringRange", WireFloat32},
	{53, "School", WireString},
	{54, "Hometown", WireString},
	{55, "OriginPara1", WireInt32},
	{56, "OriginPara2", WireInt32},
	{57, "OriginPara3", WireInt32},
	{58, "NpcTalkRule", WireString},
	{59, "CanQiecuo", WireInt32},
	{60, "PowerNo", WireInt32},
	{61, "BossLevel", WireInt32},
	{62, "NotPositive", WireInt32},
	{63, "CantAttack", WireInt32},
	{64, "CantBeAttack", WireInt32},
	{65, "CantBeKilled", WireInt32},
	{66, "AIFightFreq", WireInt32},
	{67, "ChaseRange", WireFloat32},
	{68, "SkillHoldTime", WireInt32},
	{69, "SelfForceID", WireInt32},
	{70, "SummonIDBeginFight", WireInt32},
	{71, "StopTime", WireInt32},
	{72, "Character", WireInt32},
	{73, "KarmaLevel", WireInt32},
	{74, "Force", WireInt32},
	{75, "MotionNoRotate", WireByte},
	{76, "CanPick", WireByte},
	{77, "VisType", WireInt32},
	{78, "VisDataInt", WireInt32},
	{79, "VisDataStr", WireWideString},
	{80, "HeadEffect", WireInt32},
	{81, "NpcNickName", WireWideString},
	{82, "ExtraInfo", WireString},
	{83, "RoleHide", WireInt32},
	{84, "CKindID", WireInt32},
	{85, "EscortFlag", WireInt32},
	{86, "SpringEffect", WireString},
	{87, "Material", WireString},
	{88, "WeaponName", WireString},
	{89, "BornAction", WireString},
	{90, "ActionSoundMode", WireInt32},
	{91, "Fixed", WireInt32},
	{92, "FixedBeHitAct", WireString},
	{93, "RefreshTime", WireInt32},
	{94, "DirectSpring", WireInt32},
	{95, "NpcTalkType", WireInt32},
	{96, "PreLoad", WireInt32},
	{97, "OnlyCareDistance", WireInt32},
	{98, "RefreshAtOnce", WireInt32},
})

// VisibleNPCCompatV1 returns a copy of the 99-field schema accepted by the
// current modern client. Callers may retain it without sharing mutable slices.
func VisibleNPCCompatV1() Schema {
	copySchema, err := NewSchema(visibleNPCCompatV1.ID, visibleNPCCompatV1.Fields)
	if err != nil {
		panic(err)
	}
	return copySchema
}

var visibleNPCModernV1 = func() Schema {
	fields := append([]FieldSpec(nil), visibleNPCCompatV1.Fields...)
	// The residual 2014 schema called NpcType a byte, but current client data
	// contains values through 1228 and current FxGameLogic compares it with
	// values such as 371 and 401. The modern negotiated property must retain the
	// full int32 value.
	fields[34].Type = WireInt32
	return mustSchema("visible-npc-modern-v1", fields)
}()

// VisibleNPCModernV1 is the schema for materializing modern client templates.
// CompatV1 remains available solely to characterize the existing probe wire.
func VisibleNPCModernV1() Schema {
	copySchema, err := NewSchema(visibleNPCModernV1.ID, visibleNPCModernV1.Fields)
	if err != nil {
		panic(err)
	}
	return copySchema
}
