package clientdata

import "fmt"

// WireType 是场景引导阶段协商的可见属性 schema 所使用的类型标记。
// 其取值有意与当前协议保持一致。
type WireType uint8

const (
	WireByte       WireType = 1
	WireWord       WireType = 2
	WireInt32      WireType = 3
	WireInt64      WireType = 4
	WireFloat32    WireType = 5
	WireString     WireType = 7
	WireWideString WireType = 8
	// WireObject 是原生的 64 位场景对象引用。客户端将其
	// 用于玩家 LastObject 选中目标之类的属性。
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
	case WireInt64:
		return "int64"
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
