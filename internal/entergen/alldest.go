package entergen

// alldest.go 巡逻广播帧族（0x21 ServerAllDest + 0x10 State 翻转 + 0x0E
// 移除）：移植 2433 线 cmd/protocol-probe patrol.go / 备份 entergen/npc0d.go
// 段（真机抓包定版配方）。
//
// 0x21 帧型 `op | u16 count | count×44B`，官方真机每秒一拍全量广播（含
// 停留中的实体）。NPC 条目 44B：id/own + 目的地 x/y/z/orient + 速度 +
// rotate/motion2/motion3 + 移动标志——行进 spd=WalkSpeed/RunSpeed 且
// flags=1（rotate 恒 +2π——技能.pcapng NPC 条目帧级定版）；停留 spd=0
// 且 flags=0。rotate/motion2/motion3 为玩家回显方言槽（外服私服
// 5.5f/9.9f/msg15 形态），NPC 条目恒零（rotate 除外）。

import (
	"encoding/binary"
	"math"
)

// AllDestEntry 0x21 单条 NPC 移动记录（44B）。
type AllDestEntry struct {
	ObjectID, OwnerID   uint32
	DestX, DestY, DestZ float32
	DestOrient          float32
	Speed               float32
	Rotate              float32
	Motion2, Motion3    float32
	MoveFlags           uint32
}

// ServerAllDestFrame 0x21 批量移动广播帧；空集返回 nil（单帧 ≤8 条分帧
// 由调用方做）。
func ServerAllDestFrame(entries []AllDestEntry) []byte {
	if len(entries) == 0 {
		return nil
	}
	payload := make([]byte, 3, 3+len(entries)*44)
	payload[0] = 0x21
	binary.LittleEndian.PutUint16(payload[1:], uint16(len(entries)))
	for _, e := range entries {
		payload = binary.LittleEndian.AppendUint32(payload, e.ObjectID)
		payload = binary.LittleEndian.AppendUint32(payload, e.OwnerID)
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(e.DestX))
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(e.DestY))
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(e.DestZ))
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(e.DestOrient))
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(e.Speed))
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(e.Rotate))
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(e.Motion2))
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(e.Motion3))
		payload = binary.LittleEndian.AppendUint32(payload, e.MoveFlags)
	}
	return payload
}

// NpcStateFrame 0x10 单属性 State(183)="walk"/"stand"——NPC 行为状态翻转
// 帧（起走 walk/驻停 stand，与 0x21 路点帧成对，e.pcapng 实证 §13.5；
// 先于 0x21 下发——先置动画状态再同步运动目的地）。索引按名经
// clientdata 解析（表漂移免疫）。
func NpcStateFrame(objectID, ownerID uint32, moving bool) ([]byte, error) {
	state := "stand"
	if moving {
		state = "walk"
	}
	return ObjectPropertyFrame(objectID, ownerID, 0, []IndexedProperty{
		NamedProperty("State", StringValue(state)),
	})
}

// SceneRemoveObjectFrame 0x0E 移除对象帧（9B——AOI 视口收缩/场景重入）。
func SceneRemoveObjectFrame(objectID, ownerID uint32) []byte {
	payload := make([]byte, 9)
	payload[0] = 0x0E
	binary.LittleEndian.PutUint32(payload[1:], objectID)
	binary.LittleEndian.PutUint32(payload[5:], ownerID)
	return payload
}
