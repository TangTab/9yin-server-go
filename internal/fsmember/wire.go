package fsmember

import (
	"encoding/binary"
	"fmt"
)

// ident（comm_id）命名空间——CAdapterHook::ReceiveMsg 路由分支定案
//（docs/official/FsMember全解与Go复刻.md §4.1）：
//
//	world  = 0        public = 1        store = 4..7
//	entry  = 8..0x17（8+EntryIdx）      member = 0x18..0x27（0x18+MemberIdx）
const (
	WorldIdent      uint32 = 0
	PublicIdent     uint32 = 1
	StoreIdentBase  uint32 = 4    // 4..7
	EntryIdentBase  uint32 = 8    // 8..0x17
	MemberIdentBase uint32 = 0x18 // 0x18..0x27（与 entry/forward.go 同约定）
)

// 内网帧：{src u32, dst u32, type u32} + payload（12B 头，外层 u32 长度
// 前缀帧由 fscomm.Adapter/MultiSock 承担）。
const wireHeaderSize = 12

// IdentRegister 注册帧 type 值（官方 0xFFFF 特例：绑定连接↔src ident）。
const IdentRegister uint32 = 0xFFFF

// world 消息表（ProcessWorldMsg 跳表 0x424d0/0x42518 定案，§5）。
const (
	WorldCreateScene    uint32 = 0x0
	WorldCreateClone    uint32 = 0x1
	WorldDeleteClone    uint32 = 0x2
	WorldCheckScene     uint32 = 0x3
	WorldSceneAll       uint32 = 0x4
	WorldSceneAdd       uint32 = 0x5
	WorldSceneRemove    uint32 = 0x6
	WorldPlayerAll      uint32 = 0x7
	WorldPlayerAdd      uint32 = 0x8
	WorldPlayerRemove   uint32 = 0x9
	WorldPlayerChange   uint32 = 0xA
	WorldRoleInfo       uint32 = 0xB
	WorldRoleAdd        uint32 = 0xC
	WorldRoleRemove     uint32 = 0xD
	WorldItemInfo       uint32 = 0x2D
	WorldCloseMember    uint32 = 0x37
	WorldDumpPerf       uint32 = 0x38
)

// public 消息表（ProcessPublicMsg 跳表 0x3ED18 定案，§9）。
const (
	PublicSpaces       uint32 = 0x0
	PublicData         uint32 = 0x1
	PublicAdd          uint32 = 0x2
	PublicRemove       uint32 = 0x3
	PublicAddAttr      uint32 = 0x4
	PublicRemoveAttr   uint32 = 0x5
	PublicAddRec       uint32 = 0x6
	PublicRemoveRec    uint32 = 0x7
	PublicRecAddRow    uint32 = 0x8
	PublicRecRemoveRow uint32 = 0x9
	PublicRecClear     uint32 = 0xA
	PublicRecChange    uint32 = 0xB
	PublicRecSetRow    uint32 = 0xC
	PublicMessage      uint32 = 0xD
)

// store 消息表（ProcessStoreMsg type-9 基址 0x40A04 定案，§10）。
const (
	StoreChunkInfo   uint32 = 0x9
	StoreLoadChunk   uint32 = 0xA
	StoreSaveChunk   uint32 = 0xB
	StoreDeleteChunk uint32 = 0xC
)

// entry_forw 控制面类型（与 entry/forward.go 同表；member 侧仅消费
// 3=set_client_online，其余转场景 0x10；下发用 7=send_to_client）。
const (
	ForwSetOnline  uint32 = 3
	ForwSendClient uint32 = 7
)

// SceneCmdClientMessage 场景命令命名空间里的客户端消息型
//（PutEntryMessage else 分支 SendToScene(…, 0x10, …) 定案，§7）。
const SceneCmdClientMessage uint32 = 0x10

// encodeWire 打包 {src,dst,type}+payload。
func encodeWire(src, dst, typ uint32, payload []byte) []byte {
	out := make([]byte, wireHeaderSize+len(payload))
	binary.LittleEndian.PutUint32(out[0:], src)
	binary.LittleEndian.PutUint32(out[4:], dst)
	binary.LittleEndian.PutUint32(out[8:], typ)
	copy(out[wireHeaderSize:], payload)
	return out
}

// wireHeader 帧头三域。
type wireHeader struct{ Src, Dst, Type uint32 }

// decodeWire 拆包；长度不足报错（官方 "message size error"）。
func decodeWire(body []byte) (wireHeader, []byte, error) {
	if len(body) < wireHeaderSize {
		return wireHeader{}, nil, fmt.Errorf("(CAdapterHook::ReceiveMsg)message size error: %d", len(body))
	}
	return wireHeader{
		Src:  binary.LittleEndian.Uint32(body[0:]),
		Dst:  binary.LittleEndian.Uint32(body[4:]),
		Type: binary.LittleEndian.Uint32(body[8:]),
	}, body[wireHeaderSize:], nil
}

// validSource 复刻 ReceiveMsg 源校验：world(0)/public(1)/store(4..7)/
// entry(8..0x17)；不符丢弃并记 "(CAdapterHook::ReceiveMsg)message source error"。
func validSource(src uint32) bool {
	switch {
	case src == WorldIdent, src == PublicIdent:
		return true
	case src >= StoreIdentBase && src < StoreIdentBase+4:
		return true
	case src >= EntryIdentBase && src < EntryIdentBase+16:
		return true
	}
	return false
}

// EntryIdentOfSerial 从客户端 serial 解出 entry ident：serial 高 16 位
// = EntryIdx → 8+EntryIdx（GameMember::SendToClient 定案，§7）。
func EntryIdentOfSerial(serial uint32) uint32 {
	return EntryIdentBase + (serial>>16)&0xFFFF
}
