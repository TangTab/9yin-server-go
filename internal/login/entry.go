package login

import (
	"encoding/binary"
)

// 0x0B EntryScene（进场帧）方言。布局钉死（旧分支 fork 真机形态 +
// c.pcapng 官方样本对照，见旧分支 scene_transition.go 注释）：
//
//	[0x00]     0x0B
//	[0x01:05]  u32 场景对象 ID（进场流水，官方样本 0x1521；fork 形恒定）
//	[0x05:09]  u32 玩家线缆 objectID（fork 形 0x1105C0E3）
//	[0x09:0D]  u32 玩家线缆 ownerID（fork 形 0x03DEFBAE）
//	[0x0D:0F]  u16 属性数 = 4
//	随后属性段（[idx u16][值]）：
//	  idx178  Name：[u32 8][8B 遮蔽常量]（六包恒定）
//	  idx1468 MaxVisuals：u16 20000（场景可视表容量=AddObject 注册闸；
//	                        历史恒缺发→客户端对象表未开张）
//	  idx7    ConfigID：窄串 [u32 len 含 NUL][字节][NUL]
//	  idx8    Resource：同上
//
// ⚠️ 属性 idx 按连接收到的 0x09 属性表解析（本服表方言 idx7/8）——
// 照抄外服表 idx（1447/1448）会把 Config 解析成 ClineNpcFile、场景
// 加载失败（旧分支真机实证）。0x09 表下发是属性全量集环的接入点。
const (
	OpEntryScene byte = 0x0B // S2C 进场帧

	propIdxSceneName      uint16 = 178
	propIdxMaxVisuals     uint16 = 1468
	propIdxSceneConfig    uint16 = 7
	propIdxSceneResource  uint16 = 8

	entrySceneSeq   uint32 = 0x1521   // 进场流水（官方样本值）
	playerCableObj  uint32 = 0x1105C0E3 // 玩家线缆 objectID（fork 形）
	playerCableOwn  uint32 = 0x03DEFBAE // 玩家线缆 ownerID（fork 形）
	maxVisualsValue uint16 = 20000
)

// SceneRef 场景标识（0x0B 与角色档共用的最小形态）。
type SceneRef struct {
	Config   string // 如 `ini\scene\school09_YiHuaGong`
	Resource string // 如 `school09`
}

// BirthScene 默认出生场景（移花宫，relive_info.ini 的 force_yihua
// 权威落地出生点 HomePointschool09A）。
var BirthScene = SceneRef{
	Config:   `ini\scene\school09_YiHuaGong`,
	Resource: "school09",
}

// sceneNameShade idx178 Name 的 8B 遮蔽常量（六包恒定）。
var sceneNameShade = [8]byte{0x1b, 0x54, 0x50, 0x5b, 0x02, 0x58, 0x00, 0x00}

// entrySceneFrame 生成 0x0B 进场帧（纯函数，对拍入口）。
func entrySceneFrame(scene SceneRef) []byte {
	entry := make([]byte, 15, 96)
	entry[0] = OpEntryScene
	binary.LittleEndian.PutUint32(entry[1:], entrySceneSeq)
	binary.LittleEndian.PutUint32(entry[5:], playerCableObj)
	binary.LittleEndian.PutUint32(entry[9:], playerCableOwn)
	binary.LittleEndian.PutUint16(entry[13:], 4)

	entry = binary.LittleEndian.AppendUint16(entry, propIdxSceneName)
	entry = binary.LittleEndian.AppendUint32(entry, uint32(len(sceneNameShade)))
	entry = append(entry, sceneNameShade[:]...)

	entry = binary.LittleEndian.AppendUint16(entry, propIdxMaxVisuals)
	entry = binary.LittleEndian.AppendUint16(entry, maxVisualsValue)

	entry = appendSceneStringProp(entry, propIdxSceneConfig, scene.Config)
	entry = appendSceneStringProp(entry, propIdxSceneResource, scene.Resource)
	return entry
}

// appendSceneStringProp 窄串属性段：[idx u16][u32 len 含 NUL][字节][NUL]。
func appendSceneStringProp(msg []byte, index uint16, value string) []byte {
	msg = binary.LittleEndian.AppendUint16(msg, index)
	msg = binary.LittleEndian.AppendUint32(msg, uint32(len(value)+1))
	msg = append(msg, value...)
	return append(msg, 0)
}
