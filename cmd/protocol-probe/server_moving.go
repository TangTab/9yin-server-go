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

// serverLinkPairFrame 官方对象-主人绑定帧（op 0x31 ServerLinkPair）：
// [op][u16 n][n×(obj u32, owner u32)]。官服与 0x28/0x0D 物化伴生（jn
// 179 条与 0x28 279 条交错；npc.pcapng n=4 批量=0x0E 对象段×0x0A5 玩家
// owner 段），喂养客户端实体-主人表；2026-11-11 官服差集发现本服零
// 下发，物化路径伴发单对。批量形态官服用于登录爆发段，暂不实现。
func serverLinkPairFrame(objectID, ownerID uint32) []byte {
	frame := make([]byte, 0, 11)
	frame = append(frame, 0x31)
	frame = binary.LittleEndian.AppendUint16(frame, 1)
	frame = binary.LittleEndian.AppendUint32(frame, objectID)
	frame = binary.LittleEndian.AppendUint32(frame, ownerID)
	return frame
}
