package main

import (
	"encoding/binary"
	"math"

	worldcore "github.com/local/9yin-go-server/internal/world"
)

// MotionState 是从当前 FxNet2 接收端还原出的九槽位
// ServerMoving (0x20) 负载。最后五个字段刻意保持原始/最简形态：
// 它们的动画语义尚未证实，而目的坐标部分的布局已经证实。
// 随后跟进的 ServerLocation 才是权威的到达确认。
type MotionState struct {
	DestX, DestY, DestZ, DestOrient    float32
	Motion0, Motion1, Motion2, Motion3 float32
	Extra                              uint32
}

func motionFromDest(dest worldcore.Transform) MotionState {
	return MotionState{DestX: dest.X, DestY: dest.Y, DestZ: dest.Z, DestOrient: dest.Orient}
}

// serverMoving 编码精确 45 字节的 GameReceiver::ServerMoving 数据包。
func serverMoving(objectID, ownerID uint32, state MotionState) []byte {
	payload := make([]byte, 45)
	payload[0] = 0x20
	binary.LittleEndian.PutUint32(payload[1:], objectID)
	binary.LittleEndian.PutUint32(payload[5:], ownerID)
	binary.LittleEndian.PutUint32(payload[9:], math.Float32bits(state.DestX))
	binary.LittleEndian.PutUint32(payload[13:], math.Float32bits(state.DestY))
	binary.LittleEndian.PutUint32(payload[17:], math.Float32bits(state.DestZ))
	binary.LittleEndian.PutUint32(payload[21:], math.Float32bits(state.DestOrient))
	binary.LittleEndian.PutUint32(payload[25:], math.Float32bits(state.Motion0))
	binary.LittleEndian.PutUint32(payload[29:], math.Float32bits(state.Motion1))
	binary.LittleEndian.PutUint32(payload[33:], math.Float32bits(state.Motion2))
	binary.LittleEndian.PutUint32(payload[37:], math.Float32bits(state.Motion3))
	binary.LittleEndian.PutUint32(payload[41:], state.Extra)
	return payload
}
