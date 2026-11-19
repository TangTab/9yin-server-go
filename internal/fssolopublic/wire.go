package fssolopublic

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf16"
)

// ---- 内网 TCP 帧（CInnerHook，solo 全家同构）----
//
//	+0x00 u32 N        // 其后字节数 = 12 + len(payload)
//	+0x04 u32 comm_id  // 发送方 comm 身份
//	+0x08 u32 id1      // 一级路由（public 收发恒 0）
//	+0x0C u32 id2      // 消息号（0xFFFF=identify / 0xFFFE=退出）
//	+0x10 payload
const (
	innerHeaderSize  = 16 // 4 字节长度前缀 + 12 字节体头
	innerBodyHeader  = 12
	innerIdentifyID2 = 0xFFFF
	innerExitID2     = 0xFFFE
	maxInnerFrameLen = 0x3FFFFC // 官方 N 上限（OUT_BUFFER_LEN-4）
)

// comm 身份表（innerhook.h 断言定案，COMM_ID_MAX=64）。
const (
	CommWorld     = 0 // world（连入）
	CommSelf      = 1 // SoloPublic 自身（Start SetCommIdent(1)）
	CommStoreMin  = 4 // 4..7 Store（外连，MAX_STORE_NUM=4）
	CommStoreMax  = 8
	CommEntryMin  = 8 // 8..0x17 Entry（外连，MAX_ENTRY_NUM=16，solo 特有）
	CommEntryMax  = 0x18
	CommMemberMin = 0x18 // 0x18..0x2F Member（连入，MAX_MEMBER_NUM=24）
	CommMemberMax = 0x30
	CommIDMax     = 64
)

// encodeInner 封一帧（官方 CInnerHook::SendMsg 栈上 16 字节头）。
func encodeInner(comm, id1, id2 uint32, payload []byte) ([]byte, error) {
	n := innerBodyHeader + len(payload)
	if n > maxInnerFrameLen {
		return nil, fmt.Errorf("message too long: %d > %d", n, maxInnerFrameLen)
	}
	out := make([]byte, innerHeaderSize+len(payload))
	binary.LittleEndian.PutUint32(out[0:], uint32(n))
	binary.LittleEndian.PutUint32(out[4:], comm)
	binary.LittleEndian.PutUint32(out[8:], id1)
	binary.LittleEndian.PutUint32(out[12:], id2)
	copy(out[innerHeaderSize:], payload)
	return out, nil
}

// frameAcc 按连接半包累积器（官方 comm_t 4 字节长度前缀逐帧凑齐再投递）。
type frameAcc struct{ buf []byte }

// feed 喂入字节，凑齐整帧即回调；N < 12 视为损坏（官方
// "message size error"）。
func (a *frameAcc) feed(data []byte, onFrame func(comm, id1, id2 uint32, payload []byte)) error {
	a.buf = append(a.buf, data...)
	for {
		if len(a.buf) < 4 {
			return nil
		}
		n := int(binary.LittleEndian.Uint32(a.buf))
		if n < innerBodyHeader {
			return fmt.Errorf("message size error: N=%d", n)
		}
		if len(a.buf) < 4+n {
			return nil
		}
		frame := a.buf[4 : 4+n]
		comm := binary.LittleEndian.Uint32(frame[0:])
		id1 := binary.LittleEndian.Uint32(frame[4:])
		id2 := binary.LittleEndian.Uint32(frame[8:])
		payload := append([]byte(nil), frame[12:]...)
		a.buf = a.buf[4+n:]
		onFrame(comm, id1, id2, payload)
	}
}

// ---- store 面 type（集群 FsPublic +6；solo 特有 0x2D/0x30/0x3A）----
const (
	storeRoleOffline   = 0x01 // ←store：下线结果行表 → Callee OnRoleOffline
	storePubInfoBack   = 0x19 // ←store：空间数据名清单
	storePubLoadBack   = 0x1A // ←store：单条 PubData 全量
	storePubSaveBack   = 0x1B // ←store：PubSave ACK
	storePubAddBack    = 0x1C // ←store：PubAdd ACK
	storePubRemoveBack = 0x1D // ←store：PubRemove ACK
	storeRelationBack  = 0x22 // ←store：关系表
	storeFeedBack      = 0x23 // ←store：动态流
	storeTradeBack     = 0x28 // ←store：寄售行

	storePubInfoReq  = 0x24 // →store：空间数据名清单请求 {space[0x24],table[0x24]}
	storePubLoadReq  = 0x25 // →store：单条拉取 {space,table,wname,index}
	storePubSave     = 0x26 // →store：回存 {0x90 头 + PUB4 blob}
	storePubAdd      = 0x27 // →store：新增持久化（+uid）
	storeRelationReq = 0x2D // →store：关系装载请求 {ServerId}
	storeFeedConfig  = 0x30 // →store：feed 存活期配置 {ServerId; double×levels}
	storeFeedReq     = 0x3A // →store：动态装载请求 {ServerId}
)

// world 面 type（ProcessWorldMsg 跳表 1..13）。
const (
	worldServerState    = 0x01 // → Callee OnServerState（0x24B）
	worldClosePublic    = 0x02 // → 状态机 event3
	worldSceneAll       = 0x03 // 场景全量（count + 0x90 项）
	worldSceneAdd       = 0x04 // 0x90B
	worldSceneRemove    = 0x05
	worldPlayerAll      = 0x06 // 玩家全量
	worldPlayerAdd      = 0x07 // 0x9CB
	worldPlayerRemove   = 0x08 // 0x50B
	worldPlayerChange   = 0x09 // 0x54B（CleanPlayerConnect）
	worldRoleInfo       = 0x0A // 0x34B（NameUid 全量 → +0x176）
	worldRoleAdd        = 0x0B // 0x6CB
	worldRoleRemove     = 0x0C // 0x6CB
	worldRoleRecreate   = 0x0D // 0x6CB
	worldTypeMax        = 0x0D
	worldLoadPlayerReq  = 0x00 // →world：LoadPlayer {account[0x21],wname[0x21],fromIdent}
	worldServerStateReq = 0x01 // →world：RequestServerState（1B）
	worldCloseDone      = 0x02 // →world：Closing 收尾 {u32 1}
)

// 官方断言的定长结构尺寸（sizeof 定案）。
const (
	sizeSceneAdd      = 0x90 // member_forw_scene_add_t
	sizePlayerAdd     = 0x9C // member_forw_player_add_t
	sizePlayerRemove  = 0x50 // member_forw_player_remove_t
	sizePlayerChange  = 0x54 // member_forw_player_change_t
	sizeRoleMutate    = 0x6C // member_forw_role_add/remove/recreate_t
	sizeRoleInfo      = 0x34 // member_forw_role_info_t
	sizeServerState   = 0x24 // public_res_server_state_t
	sizeMemberDataReq = 0x74 // public_forw_data_t（solo 加宽；集群 0x29B）
	sizeStoreLoadHdr  = 0xC0 // store_back_pub_load_t 定长前缀（集群 0xB1）
)

// ---- UDP 广播（magic 0x3001 solo 方言）----
//
// 自宣（BroadcastServerInfo @0x100BEC40，每秒）：
//
//	+0x00 u32 kind = 4（SoloPublic；集群 public 同 kind 但 magic 0x300）
//	+0x04 u32 magic = 0x3001
//	+0x08 char addr[0x10]   InnerAddr
//	+0x18 u32 port          InnerPort
//	+0x1C u32 district_id
//	+0x20 u32 server_id
//
// 收听（OnBroadcast @0x100BBAF0，DistrictId/ServerId 双匹配门）：
// type2 store 自宣（≥0x28，[0x24]=store comm 4..7）→ 外连该 store；
// type1 entry 自宣（≥0x48，[0x24]<0x10 序号）→ 外连 comm 8+index；
// type0 world 广播（≥0x84，double@0x7C）→ SetRoleProtectBase。
const (
	announceMagic   = 0x3001
	announceSize    = 0x24
	announcePublic  = 4 // 本服自宣 kind
	announceStore   = 2
	announceEntry   = 1
	announceWorldEx = 0 // world 扩展广播（RoleProtectBase）
	worldIndexMax   = 0x10
)

// Announce 是广播报文公共前缀。
type Announce struct {
	Type     uint32
	Addr     string
	Port     uint32
	District uint32
	Server   uint32
	Index    uint32 // [0x24]：store=comm(4..7)；entry=序号(<0x10)
}

// encodeAnnounce 构造本服 0x24B 自宣包。
func encodeAnnounce(addr string, port, district, server uint32) []byte {
	out := make([]byte, announceSize)
	binary.LittleEndian.PutUint32(out[0:], announcePublic)
	binary.LittleEndian.PutUint32(out[4:], announceMagic)
	copy(out[8:], addr)
	if len(addr) >= 0x10 {
		out[8+0xF] = 0
	}
	binary.LittleEndian.PutUint32(out[0x18:], port)
	binary.LittleEndian.PutUint32(out[0x1C:], district)
	binary.LittleEndian.PutUint32(out[0x20:], server)
	return out
}

// decodeAnnounce 解析广播包；magic 不符或 type 长度不足返回 ok=false。
func decodeAnnounce(b []byte) (a Announce, ok bool) {
	if len(b) < announceSize {
		return a, false
	}
	a.Type = binary.LittleEndian.Uint32(b[0:])
	if binary.LittleEndian.Uint32(b[4:]) != announceMagic {
		return a, false
	}
	switch a.Type {
	case announceStore:
		if len(b) < 0x28 {
			return a, false
		}
	case announceEntry:
		if len(b) < 0x48 {
			return a, false
		}
	case announceWorldEx:
		if len(b) < 0x84 {
			return a, false
		}
	case announcePublic:
	default:
		return a, false
	}
	a.Addr = cstr(b[8 : 8+0x10])
	a.Port = binary.LittleEndian.Uint32(b[0x18:])
	a.District = binary.LittleEndian.Uint32(b[0x1C:])
	a.Server = binary.LittleEndian.Uint32(b[0x20:])
	if len(b) >= 0x28 {
		a.Index = binary.LittleEndian.Uint32(b[0x24:])
	}
	return a, true
}

// worldRoleProtectBase 从 type0 world 广播取 RoleProtectBase(double@0x7C)。
func worldRoleProtectBase(b []byte) (float64, bool) {
	if len(b) < 0x84 {
		return 0, false
	}
	return float64FromBytes(b[0x7C : 0x7C+8]), true
}

func float64FromBytes(b []byte) float64 {
	return math.Float64frombits(binary.LittleEndian.Uint64(b))
}

// ---- 定长域工具 ----

func cstr(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// fixedWide 取定宽宽串域（UCS2-LE，L'\0' 终止）。
func fixedWide(b []byte) string {
	units := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u := binary.LittleEndian.Uint16(b[i:])
		if u == 0 {
			break
		}
		units = append(units, u)
	}
	return string(utf16.Decode(units))
}

// putFixedWide 向定宽域写 UCS2-LE（NUL 填充，超长截断保终结位）。
func putFixedWide(dst []byte, s string) {
	units := utf16.Encode([]rune(s))
	for i := 0; i < len(dst)/2; i++ {
		var u uint16
		if i < len(units) {
			u = units[i]
		}
		binary.LittleEndian.PutUint16(dst[i*2:], u)
	}
	if len(units) >= len(dst)/2 && len(dst) >= 2 {
		binary.LittleEndian.PutUint16(dst[len(dst)-2:], 0)
	}
}

// putFixedStr 向定长窄串域写（NUL 填充，超长截断保终结位）。
func putFixedStr(dst []byte, s string) {
	n := copy(dst, s)
	if n < len(dst) {
		dst[n] = 0
	} else {
		dst[len(dst)-1] = 0
	}
}

func u16at(b []byte, off int) uint16 { return binary.LittleEndian.Uint16(b[off:]) }
func u32at(b []byte, off int) uint32 { return binary.LittleEndian.Uint32(b[off:]) }
func i32at(b []byte, off int) int32  { return int32(binary.LittleEndian.Uint32(b[off:])) }
