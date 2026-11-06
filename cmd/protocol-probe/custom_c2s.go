package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf16"
)

// clientCustomMessage 是 C2S GameVisual:CustomSend 解码后的形式。
// 正式服客户端可能直接发送 TVarList（0x1E/0x27），也可能发送由
// C2S 0x0A 活动信封携带的 TVarList。后者在相同的 u16 计数
// 类型列表体之前还有一个 21 字节的不透明头部。
type clientCustomMessage struct {
	Opcode byte
	Values []clientCustomValue
}

type clientCustomValue struct {
	Type    byte
	Int32   int32
	Int64   int64
	Float32 float32
	Float64 float64
	Text    string
	Raw     [8]byte
}

// exactInt32 接受现代 TVarList 写入方在枚举式活动参数上
// 实际使用的数值表示。一些原生表单会
// 把选择值编码为 double（例如门派 1006/23 发送的是
// double:0、double:1），尽管这些值在语义上是整数。
// 绝不静默截断带小数、NaN 或超出范围的值。
func (value clientCustomValue) exactInt32() (int32, bool) {
	switch value.Type {
	case 2:
		return value.Int32, true
	case 3:
		if value.Int64 < math.MinInt32 || value.Int64 > math.MaxInt32 {
			return 0, false
		}
		return int32(value.Int64), true
	case 4:
		return exactFloatInt32(float64(value.Float32))
	case 5:
		return exactFloatInt32(value.Float64)
	default:
		return 0, false
	}
}

func exactFloatInt32(value float64) (int32, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value || value < math.MinInt32 || value > math.MaxInt32 {
		return 0, false
	}
	return int32(value), true
}

func (value clientCustomValue) String() string {
	switch value.Type {
	case 2:
		return fmt.Sprintf("int:%d", value.Int32)
	case 3:
		return fmt.Sprintf("int64:%d", value.Int64)
	case 4:
		return fmt.Sprintf("float:%g", value.Float32)
	case 5:
		return fmt.Sprintf("double:%g", value.Float64)
	case 6:
		return fmt.Sprintf("string:%q", value.Text)
	case 7:
		return fmt.Sprintf("widestr:%q", value.Text)
	case 8:
		return fmt.Sprintf("object:%x", value.Raw)
	default:
		return fmt.Sprintf("type%d", value.Type)
	}
}

// parseClientCustomMessage 解码直接的 0x1E/0x27 TVarList。其第一个
// 值始终是 C2S 自定义消息 ID，这些消息由
// game_visual:CustomSend 创建。
func parseClientCustomMessage(frame []byte) (clientCustomMessage, error) {
	if len(frame) < 3 || (frame[0] != 0x1E && frame[0] != 0x27) {
		return clientCustomMessage{}, fmt.Errorf("not a custom frame")
	}
	return parseClientCustomValues(frame[0], frame[1:])
}

// parseClientActivityCustomMessage 解码在当前客户端中观察到的
// 0x0A 信封：0x0A | 16 字节会话区 | u32 序列号 | TVarList。
// 当数据包不是完整的类型列表时返回 ok=false，因为
// 0x0A 还承载非自定义的活动流量。
func parseClientActivityCustomMessage(frame []byte) (message clientCustomMessage, ok bool, err error) {
	const activityHeaderSize = 21
	if len(frame) < activityHeaderSize+3 || frame[0] != 0x0A {
		return clientCustomMessage{}, false, nil
	}
	message, err = parseClientCustomValues(frame[0], frame[activityHeaderSize:])
	if err != nil {
		return clientCustomMessage{}, false, nil
	}
	return message, true, nil
}

func parseClientCustomValues(opcode byte, payload []byte) (clientCustomMessage, error) {
	if len(payload) < 2 {
		return clientCustomMessage{}, fmt.Errorf("custom payload has no count")
	}
	count := int(binary.LittleEndian.Uint16(payload[:2]))
	if count == 0 {
		return clientCustomMessage{}, fmt.Errorf("custom frame has no typed values")
	}
	if count > 1024 {
		return clientCustomMessage{}, fmt.Errorf("custom typed value count %d exceeds limit", count)
	}
	message := clientCustomMessage{Opcode: opcode, Values: make([]clientCustomValue, 0, count)}
	rest := payload[2:]
	for index := 0; index < count; index++ {
		if len(rest) == 0 {
			return clientCustomMessage{}, fmt.Errorf("custom value %d has no type tag", index)
		}
		value := clientCustomValue{Type: rest[0]}
		rest = rest[1:]
		switch value.Type {
		case 2:
			if len(rest) < 4 {
				return clientCustomMessage{}, fmt.Errorf("custom int value %d is truncated", index)
			}
			value.Int32 = int32(binary.LittleEndian.Uint32(rest))
			rest = rest[4:]
		case 3:
			if len(rest) < 8 {
				return clientCustomMessage{}, fmt.Errorf("custom int64 value %d is truncated", index)
			}
			value.Int64 = int64(binary.LittleEndian.Uint64(rest))
			rest = rest[8:]
		case 4:
			if len(rest) < 4 {
				return clientCustomMessage{}, fmt.Errorf("custom float value %d is truncated", index)
			}
			value.Float32 = math.Float32frombits(binary.LittleEndian.Uint32(rest))
			rest = rest[4:]
		case 5:
			if len(rest) < 8 {
				return clientCustomMessage{}, fmt.Errorf("custom double value %d is truncated", index)
			}
			value.Float64 = math.Float64frombits(binary.LittleEndian.Uint64(rest))
			rest = rest[8:]
		case 6:
			var err error
			value.Text, rest, err = parseCustomText(rest, false)
			if err != nil {
				return clientCustomMessage{}, fmt.Errorf("custom string value %d: %w", index, err)
			}
		case 7:
			var err error
			value.Text, rest, err = parseCustomText(rest, true)
			if err != nil {
				return clientCustomMessage{}, fmt.Errorf("custom widestr value %d: %w", index, err)
			}
		case 8:
			if len(rest) < len(value.Raw) {
				return clientCustomMessage{}, fmt.Errorf("custom object value %d is truncated", index)
			}
			copy(value.Raw[:], rest[:len(value.Raw)])
			rest = rest[len(value.Raw):]
		default:
			return clientCustomMessage{}, fmt.Errorf("custom value %d has unsupported type %d", index, value.Type)
		}
		message.Values = append(message.Values, value)
	}
	if len(rest) != 0 {
		return clientCustomMessage{}, fmt.Errorf("custom frame has %d trailing bytes", len(rest))
	}
	return message, nil
}

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
