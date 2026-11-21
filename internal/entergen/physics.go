package entergen

// 登录物理档案流（0x10 单属性 ×36）——b-line physicsLoginFrameSpecs
// 定版（外服私服在表后、建视图前逐帧连发；一次性大档不触发属性更新沿
// 上武装的接近/冲锋子系统）。含 InteractStatus=-1（缺发会让 form_tvt
// 判非 -1 弹刺探技能栏）与 IsRun=0。

import (
	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

type physicsFrameSpec struct {
	name  string
	fval  float64
	ival  int64
	isInt bool
}

var physicsFrameSpecs = []physicsFrameSpec{
	{"PhyParryDamRes", 50, 0, false},
	{"MagicParryDamRes", 50, 0, false},
	{"HPHeartSpeed", 0, 5000, true},
	{"MPHeartSpeed", 0, 5000, true},
	{"GRushRange", 5.0, 0, false},
	{"ARushRange", 8.0, 0, false},
	{"WRushRange", 8.0, 0, false},
	{"UnderWaterSpeed", 10.0, 0, false},
	{"UnderWaterModifySpeed", 1.0, 0, false},
	{"WalkSpeed", 1.281, 0, false},
	{"RunSpeed", 3.9, 0, false},
	{"ActionSpeed", 1.0, 0, false},
	{"MoveSpeed", 3.9, 0, false},
	{"JumpSpeed", 5.5, 0, false},
	{"Gravity", 9.8, 0, false},
	{"DropHeightPub", 10.0, 0, false},
	{"ClimbSpeed", 7.5, 0, false},
	{"DriftSpeed", 7.5, 0, false},
	{"AdvancedJumpSpeed", 23.0, 0, false},
	{"AdvancedGlideSpeed", 5.0, 0, false},
	{"AdvancedFastfallSpeed", 30.0, 0, false},
	{"AdvancedFastfallDist", 40.0, 0, false},
	{"UnderSandSpeed", 5.8, 0, false},
	{"WaterRushSpeed", 25.0, 0, false},
	{"WaterRushDist", 6.5, 0, false},
	{"DriftJumpSpeed", 6.0, 0, false},
	{"SndJumpSpeed", 6.0, 0, false},
	{"ThdJumpSpeed", 6.0, 0, false},
	{"AirRushSpeed", 15.0, 0, false},
	{"AirRushDist", 12.0, 0, false},
	{"LandRushSpeed", 25.0, 0, false},
	{"LandRushDist", 6.5, 0, false},
	{"MaxQingGongPoint", 0, 150, true},
	{"MaxGJQingGongPoint", 0, 150, true},
	{"InteractStatus", 0, -1, true},
	{"IsRun", 0, 0, true},
}

// PhysicsLoginStreamFrames 登录物理档案流（obj/own=玩家身份）。
func PhysicsLoginStreamFrames(objectID, ownerID uint32) ([][]byte, error) {
	frames := make([][]byte, 0, len(physicsFrameSpecs))
	for _, spec := range physicsFrameSpecs {
		entry := clientdata.MustPlayerProperty(spec.name)
		var value PropertyValue
		if spec.isInt {
			value = propertyIntByType(entry.TypeCode, spec.ival)
		} else {
			value = propertyFloatByType(entry.TypeCode, spec.fval)
		}
		frame, err := ObjectPropertyFrame(objectID, ownerID, 0, []IndexedProperty{
			{Index: entry.Index, Name: spec.name, Value: value},
		})
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// propertyIntByType 按表线型承载整值（InteractStatus t4=Int64、
// HeartSpeed t3=Int32、IsRun t3 等各按其表型）。
func propertyIntByType(typeCode int, v int64) PropertyValue {
	switch typeCode {
	case 1:
		return ByteValue(byte(v))
	case 2:
		return WordValue(uint16(v))
	case 4:
		return Int64Value(v)
	default:
		return Int32Value(int32(v))
	}
}

// propertyFloatByType 按表线型承载浮值（速度族 t5 为主）。
func propertyFloatByType(typeCode int, v float64) PropertyValue {
	switch typeCode {
	case 3:
		return Int32Value(int32(v))
	default:
		return Float32Value(float32(v))
	}
}
