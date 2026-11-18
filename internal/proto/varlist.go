package proto

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf16"
)

// TVarList（类型化变量表）线上类型标签——Fs 框架 IVarList 的线路形态，
// 字节布局由旧分支 protocol-probe 对现代客户端实测钉死：
//
//	[u16 数量 N][N × (类型标签 + 值)]
//	  2=int32(4B LE) 3=int64(8B LE) 4=float32(4B LE) 5=float64(8B LE)
//	  6=string([u32 字节长含 NUL][bytes][00])
//	  7=widestr([u32 字节长=(单元数+1)*2][UTF-16LE 单元][0000])
//	  8=object(8B 原样，PERSISTID)
const (
	VarInt32  byte = 2
	VarInt64  byte = 3
	VarFloat  byte = 4
	VarDouble byte = 5
	VarString byte = 6
	VarWide   byte = 7
	VarObject byte = 8
)

// Value 单个类型化变量。
type Value struct {
	Type byte
	I64  int64   // VarInt32 / VarInt64
	F64  float64 // VarFloat / VarDouble
	Text string  // VarString / VarWide
	Raw  [8]byte // VarObject
}

// VarList 类型化变量表（消息体容器）。
type VarList struct {
	Values []Value
}

// AddInt32 追加 int32（链式）。
func (v *VarList) AddInt32(n int32) *VarList {
	v.Values = append(v.Values, Value{Type: VarInt32, I64: int64(n)})
	return v
}

// AddInt64 追加 int64。
func (v *VarList) AddInt64(n int64) *VarList {
	v.Values = append(v.Values, Value{Type: VarInt64, I64: n})
	return v
}

// AddFloat32 追加 float32。
func (v *VarList) AddFloat32(f float32) *VarList {
	v.Values = append(v.Values, Value{Type: VarFloat, F64: float64(f)})
	return v
}

// AddDouble 追加 float64。
func (v *VarList) AddDouble(f float64) *VarList {
	v.Values = append(v.Values, Value{Type: VarDouble, F64: f})
	return v
}

// AddString 追加窄字符串（NUL 结尾计入长度）。
func (v *VarList) AddString(s string) *VarList {
	v.Values = append(v.Values, Value{Type: VarString, Text: s})
	return v
}

// AddWide 追加宽字符串（UTF-16LE，终止 0000 计入字节长度）。
func (v *VarList) AddWide(s string) *VarList {
	v.Values = append(v.Values, Value{Type: VarWide, Text: s})
	return v
}

// AddObject 追加对象引用（8 字节 PERSISTID 原样传输）。
func (v *VarList) AddObject(raw [8]byte) *VarList {
	v.Values = append(v.Values, Value{Type: VarObject, Raw: raw})
	return v
}

// Encode 编码为线上字节。
func (v *VarList) Encode() []byte {
	out := make([]byte, 2, 16+len(v.Values)*12)
	binary.LittleEndian.PutUint16(out, uint16(len(v.Values)))
	for _, val := range v.Values {
		out = append(out, val.Type)
		switch val.Type {
		case VarInt32:
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], uint32(int32(val.I64)))
			out = append(out, b[:]...)
		case VarInt64:
			var b [8]byte
			binary.LittleEndian.PutUint64(b[:], uint64(val.I64))
			out = append(out, b[:]...)
		case VarFloat:
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], math.Float32bits(float32(val.F64)))
			out = append(out, b[:]...)
		case VarDouble:
			var b [8]byte
			binary.LittleEndian.PutUint64(b[:], math.Float64bits(val.F64))
			out = append(out, b[:]...)
		case VarString:
			n := uint32(len(val.Text) + 1)
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], n)
			out = append(out, b[:]...)
			out = append(out, val.Text...)
			out = append(out, 0)
		case VarWide:
			units := utf16.Encode([]rune(val.Text))
			n := uint32((len(units) + 1) * 2)
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], n)
			out = append(out, b[:]...)
			var w [2]byte
			for _, u := range units {
				binary.LittleEndian.PutUint16(w[:], u)
				out = append(out, w[:]...)
			}
			out = append(out, 0, 0)
		case VarObject:
			out = append(out, val.Raw[:]...)
		default:
			panic(fmt.Sprintf("proto: 未知类型标签 %d", val.Type))
		}
	}
	return out
}

// DecodeVarList 从线上字节解码（要求整帧恰好耗尽）。
func DecodeVarList(b []byte) (*VarList, error) {
	if len(b) < 2 {
		return nil, fmt.Errorf("proto: varlist 太短 %d", len(b))
	}
	n := int(binary.LittleEndian.Uint16(b))
	if n > 1024 {
		return nil, fmt.Errorf("proto: varlist 数量超限 %d", n)
	}
	rest := b[2:]
	v := &VarList{Values: make([]Value, 0, n)}
	for i := 0; i < n; i++ {
		if len(rest) == 0 {
			return nil, fmt.Errorf("proto: 第 %d 个值缺类型标签", i)
		}
		val := Value{Type: rest[0]}
		rest = rest[1:]
		var err error
		switch val.Type {
		case VarInt32:
			val.I64, rest, err = takeI64(rest, 4)
		case VarInt64:
			val.I64, rest, err = takeI64(rest, 8)
		case VarFloat:
			var f float64
			f, rest, err = takeFloat(rest, 4)
			val.F64 = f
		case VarDouble:
			val.F64, rest, err = takeFloat(rest, 8)
		case VarString:
			val.Text, rest, err = takeText(rest, false)
		case VarWide:
			val.Text, rest, err = takeText(rest, true)
		case VarObject:
			if len(rest) < 8 {
				err = fmt.Errorf("object 截断")
				break
			}
			copy(val.Raw[:], rest[:8])
			rest = rest[8:]
		default:
			err = fmt.Errorf("第 %d 个值不支持类型 %d", i, val.Type)
		}
		if err != nil {
			return nil, fmt.Errorf("proto: 第 %d 个值: %w", i, err)
		}
		v.Values = append(v.Values, val)
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("proto: varlist 尾部多 %d 字节", len(rest))
	}
	return v, nil
}

func takeI64(b []byte, size int) (int64, []byte, error) {
	if len(b) < size {
		return 0, nil, fmt.Errorf("整数截断")
	}
	if size == 4 {
		return int64(int32(binary.LittleEndian.Uint32(b))), b[4:], nil
	}
	return int64(binary.LittleEndian.Uint64(b)), b[8:], nil
}

func takeFloat(b []byte, size int) (float64, []byte, error) {
	if len(b) < size {
		return 0, nil, fmt.Errorf("浮点截断")
	}
	if size == 4 {
		return float64(math.Float32frombits(binary.LittleEndian.Uint32(b))), b[4:], nil
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(b)), b[8:], nil
}

func takeText(b []byte, wide bool) (string, []byte, error) {
	if len(b) < 4 {
		return "", nil, fmt.Errorf("缺字节长度")
	}
	n := int(binary.LittleEndian.Uint32(b))
	b = b[4:]
	if n <= 0 || n > len(b) {
		return "", nil, fmt.Errorf("非法字节长度 %d", n)
	}
	data := b[:n]
	b = b[n:]
	if wide {
		units := make([]uint16, 0, n/2)
		for i := 0; i+1 < len(data); i += 2 {
			u := binary.LittleEndian.Uint16(data[i:])
			if u == 0 {
				break
			}
			units = append(units, u)
		}
		return string(utf16.Decode(units)), b, nil
	}
	if data[len(data)-1] == 0 {
		data = data[:len(data)-1]
	}
	return string(data), b, nil
}
