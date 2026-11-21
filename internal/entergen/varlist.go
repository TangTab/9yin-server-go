package entergen

// TVarList 自定义消息（0x1E 外壳）编码器——b-line messages.go
// serverCustomValue 族移植。tag：2=int32 3=int64 4=f32 5=f64
// 6=窄串 7=宽串 8=object。

import (
	"encoding/binary"
	"fmt"
	"math"
)

// CustomValue TVarList 单值。
type CustomValue struct {
	kind byte
	i32  int32
	i64  int64
	f32  float32
	f64  float64
	text string
	obj  uint64
}

func CustomInt(v int32) CustomValue         { return CustomValue{kind: 2, i32: v} }
func CustomInt64(v int64) CustomValue       { return CustomValue{kind: 3, i64: v} }
func CustomFloat32(v float32) CustomValue   { return CustomValue{kind: 4, f32: v} }
func CustomFloat64(v float64) CustomValue   { return CustomValue{kind: 5, f64: v} }
func CustomString(v string) CustomValue     { return CustomValue{kind: 6, text: v} }
func CustomWideString(v string) CustomValue { return CustomValue{kind: 7, text: v} }
func CustomObject(obj, own uint32) CustomValue {
	return CustomValue{kind: 8, obj: uint64(obj) | uint64(own)<<32}
}

// CustomMessageFrame 0x1E 数值路由自定义消息：首值=消息号。
func CustomMessageFrame(messageID int32, values ...CustomValue) ([]byte, error) {
	if messageID < 1 {
		return nil, fmt.Errorf("custom message id %d must be positive", messageID)
	}
	args := append([]CustomValue{CustomInt(messageID)}, values...)
	msg := make([]byte, 3, 64)
	msg[0] = 0x1E
	binary.LittleEndian.PutUint16(msg[1:], uint16(len(args)))
	for _, v := range args {
		switch v.kind {
		case 2:
			msg = append(msg, 2)
			msg = binary.LittleEndian.AppendUint32(msg, uint32(v.i32))
		case 3:
			msg = append(msg, 3)
			msg = binary.LittleEndian.AppendUint64(msg, uint64(v.i64))
		case 4:
			msg = append(msg, 4)
			msg = binary.LittleEndian.AppendUint32(msg, math.Float32bits(v.f32))
		case 5:
			msg = append(msg, 5)
			msg = binary.LittleEndian.AppendUint64(msg, math.Float64bits(v.f64))
		case 6:
			msg = append(msg, 6)
			msg = binary.LittleEndian.AppendUint32(msg, uint32(len(v.text)+1))
			msg = append(msg, v.text...)
			msg = append(msg, 0)
		case 7:
			msg = append(msg, 7)
			units := []rune(v.text)
			msg = binary.LittleEndian.AppendUint32(msg, uint32(len(units)*2+2))
			for _, r := range units {
				msg = binary.LittleEndian.AppendUint16(msg, uint16(r))
			}
			msg = binary.LittleEndian.AppendUint16(msg, 0)
		case 8:
			msg = append(msg, 8)
			msg = binary.LittleEndian.AppendUint64(msg, v.obj)
		default:
			return nil, fmt.Errorf("custom message %d has empty argument", messageID)
		}
	}
	return msg, nil
}
