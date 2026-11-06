package main

import (
	"encoding/binary"
	"math"
)

func parseMotionPosition(msg []byte) (x, y, z, orient float32, ok bool) {
	// 67 字节的坐标更新包在 0x26 处有一段稳定的三十二字节连续浮点：
	// X、Y、Z 和朝向。更长的移动包以同样的带标签格式包含目的地 X/Z、
	// 速度和角度；扫描这类包可能拼出一个貌似合理却错误的坐标元组，
	// 因此绝不能让它们更新存储。
	if len(msg) != 67 {
		return 0, 0, 0, 0, false
	}
	const offset = 0x26
	if msg[offset] != 4 || msg[offset+5] != 4 || msg[offset+10] != 4 || msg[offset+15] != 4 {
		return 0, 0, 0, 0, false
	}
	x = math.Float32frombits(binary.LittleEndian.Uint32(msg[offset+1:]))
	y = math.Float32frombits(binary.LittleEndian.Uint32(msg[offset+6:]))
	z = math.Float32frombits(binary.LittleEndian.Uint32(msg[offset+11:]))
	orient = math.Float32frombits(binary.LittleEndian.Uint32(msg[offset+16:]))
	if !float32Finite(x) || !float32Finite(y) || !float32Finite(z) || !float32Finite(orient) {
		return 0, 0, 0, 0, false
	}
	if math.Abs(float64(x)) > 10000 || math.Abs(float64(y)) > 3000 || math.Abs(float64(z)) > 10000 || math.Abs(float64(orient)) > 20 {
		return 0, 0, 0, 0, false
	}
	return x, y, z, orient, true
}

func float32Finite(value float32) bool {
	v := float64(value)
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
