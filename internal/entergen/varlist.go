package entergen

// TVarList 自定义消息（0x1E 外壳）编码器与解码器——b-line messages.go
// serverCustomValue 族移植。tag：2=int32 3=int64 4=f32 5=f64
// 6=窄串 7=宽串 8=object。

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf16"
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

// KeyedMessageFrame 0x1E 字符串路由自定义消息：首值=消息键
// （internal/proto/custom_gen.go 的 SERVER_CUSTOMMSG_* 键族，如
// "task_msg"/"sysinfo"——客户端 custom_handler 按键名分发）。
func KeyedMessageFrame(key string, values ...CustomValue) ([]byte, error) {
	if key == "" {
		return nil, fmt.Errorf("custom message key is empty")
	}
	args := append([]CustomValue{CustomString(key)}, values...)
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
			return nil, fmt.Errorf("custom message %q has empty argument", key)
		}
	}
	return msg, nil
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

// ---- C2S 解码（客户端 GameVisual:CustomSend 上行同壳）----

// CustomInbound 一条 C2S 自定义消息的解码结果（首值恒为消息号）。
type CustomInbound struct {
	Opcode byte // 0x1E / 0x27（0x0A 活动信封携带时为 0x0A）
	Values []CustomInboundValue
}

// CustomInboundValue TVarList 单值（object 型保原始 8B）。
type CustomInboundValue struct {
	Type    byte
	Int32   int32
	Int64   int64
	Float32 float32
	Float64 float64
	Text    string
	Raw     [8]byte
}

// ExactInt32 接受枚举式参数实际使用的数值表示——部分原生表单会把
// 选择值编码为 double（门派 1006/23 发 double:0/1），但不静默截断带
// 小数、NaN 或超范围的值。
func (v CustomInboundValue) ExactInt32() (int32, bool) {
	switch v.Type {
	case 2:
		return v.Int32, true
	case 3:
		if v.Int64 < math.MinInt32 || v.Int64 > math.MaxInt32 {
			return 0, false
		}
		return int32(v.Int64), true
	case 4:
		return exactFloatInt32(float64(v.Float32))
	case 5:
		return exactFloatInt32(v.Float64)
	default:
		return 0, false
	}
}

func exactFloatInt32(value float64) (int32, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value ||
		value < math.MinInt32 || value > math.MaxInt32 {
		return 0, false
	}
	return int32(value), true
}

// ParseCustomMessage 解码裸 0x1E/0x27 TVarList（frame 含 opcode 字节）。
func ParseCustomMessage(frame []byte) (CustomInbound, error) {
	if len(frame) < 3 || (frame[0] != 0x1E && frame[0] != 0x27) {
		return CustomInbound{}, fmt.Errorf("not a custom frame")
	}
	return ParseCustomValues(frame[0], frame[1:])
}

// ParseActivityCustomMessage 解码 0x0A 活动信封携带的 TVarList：
// 0x0A | 16B 会话区 | u32 序列号 | 类型列表体。非自定义的活动流量
// 返回 ok=false（0x0A 还承载其它内容，不可报错）。
func ParseActivityCustomMessage(frame []byte) (message CustomInbound, ok bool, err error) {
	const activityHeaderSize = 21
	if len(frame) < activityHeaderSize+3 || frame[0] != 0x0A {
		return CustomInbound{}, false, nil
	}
	message, err = ParseCustomValues(frame[0], frame[activityHeaderSize:])
	if err != nil {
		return CustomInbound{}, false, nil
	}
	return message, true, nil
}

// ParseCustomValues 解码 opcode 之后的类型列表体：[u16 计数][tag 值]…。
func ParseCustomValues(opcode byte, payload []byte) (CustomInbound, error) {
	if len(payload) < 2 {
		return CustomInbound{}, fmt.Errorf("custom payload has no count")
	}
	count := int(binary.LittleEndian.Uint16(payload[:2]))
	if count == 0 {
		return CustomInbound{}, fmt.Errorf("custom frame has no typed values")
	}
	if count > 1024 {
		return CustomInbound{}, fmt.Errorf("custom typed value count %d exceeds limit", count)
	}
	message := CustomInbound{Opcode: opcode, Values: make([]CustomInboundValue, 0, count)}
	rest := payload[2:]
	for index := 0; index < count; index++ {
		if len(rest) == 0 {
			return CustomInbound{}, fmt.Errorf("custom value %d has no type tag", index)
		}
		value := CustomInboundValue{Type: rest[0]}
		rest = rest[1:]
		switch value.Type {
		case 2:
			if len(rest) < 4 {
				return CustomInbound{}, fmt.Errorf("custom int value %d is truncated", index)
			}
			value.Int32 = int32(binary.LittleEndian.Uint32(rest))
			rest = rest[4:]
		case 3:
			if len(rest) < 8 {
				return CustomInbound{}, fmt.Errorf("custom int64 value %d is truncated", index)
			}
			value.Int64 = int64(binary.LittleEndian.Uint64(rest))
			rest = rest[8:]
		case 4:
			if len(rest) < 4 {
				return CustomInbound{}, fmt.Errorf("custom float value %d is truncated", index)
			}
			value.Float32 = math.Float32frombits(binary.LittleEndian.Uint32(rest))
			rest = rest[4:]
		case 5:
			if len(rest) < 8 {
				return CustomInbound{}, fmt.Errorf("custom double value %d is truncated", index)
			}
			value.Float64 = math.Float64frombits(binary.LittleEndian.Uint64(rest))
			rest = rest[8:]
		case 6:
			var err error
			value.Text, rest, err = parseCustomText(rest, false)
			if err != nil {
				return CustomInbound{}, fmt.Errorf("custom string value %d: %w", index, err)
			}
		case 7:
			var err error
			value.Text, rest, err = parseCustomText(rest, true)
			if err != nil {
				return CustomInbound{}, fmt.Errorf("custom widestr value %d: %w", index, err)
			}
		case 8:
			if len(rest) < len(value.Raw) {
				return CustomInbound{}, fmt.Errorf("custom object value %d is truncated", index)
			}
			copy(value.Raw[:], rest[:len(value.Raw)])
			rest = rest[len(value.Raw):]
		default:
			return CustomInbound{}, fmt.Errorf("custom value %d has unsupported type %d", index, value.Type)
		}
		message.Values = append(message.Values, value)
	}
	if len(rest) != 0 {
		return CustomInbound{}, fmt.Errorf("custom frame has %d trailing bytes", len(rest))
	}
	return message, nil
}

// parseCustomText 解码窄串/宽串（长度含 NUL 终止）。
func parseCustomText(rest []byte, wide bool) (string, []byte, error) {
	if len(rest) < 4 {
		return "", nil, fmt.Errorf("missing byte length")
	}
	length := int(binary.LittleEndian.Uint32(rest))
	rest = rest[4:]
	if length == 0 || length > len(rest) {
		return "", nil, fmt.Errorf("invalid byte length %d", length)
	}
	encoded := rest[:length]
	rest = rest[length:]
	if wide {
		if length < 2 || length%2 != 0 || encoded[length-2] != 0 || encoded[length-1] != 0 {
			return "", nil, fmt.Errorf("wide string is not NUL terminated")
		}
		units := make([]uint16, 0, (length-2)/2)
		for index := 0; index < length-2; index += 2 {
			units = append(units, binary.LittleEndian.Uint16(encoded[index:]))
		}
		return string(utf16.Decode(units)), rest, nil
	}
	if encoded[length-1] != 0 {
		return "", nil, fmt.Errorf("string is not NUL terminated")
	}
	return string(encoded[:length-1]), rest, nil
}
