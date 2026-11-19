package fspublic

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

// ident（comm_id）命名空间——public 视角（CInterConnect 定案，
// 与 member 侧全局一致；SetAdapterID 末尾 SetCommIdent(1) 实证）：
//
//	public = 1（自身）   world = 0        store = 4..7（4+i）
//	member = 0x18..0x27（0x18+i，MAX_MEMBER_NUM=16）
const (
	WorldIdent      uint32 = 0
	PublicIdent     uint32 = 1
	StoreIdentBase  uint32 = 4    // 4..7（MAX_STORE_NUM=4）
	MemberIdentBase uint32 = 0x18 // 0x18..0x27
	MemberIdentMax  uint32 = 0x27
)

// 内网帧：{src u32, dst u32, type u32} + payload（12B 头；外层 u32 长度
// 前缀帧由 fscomm.Adapter 承担——FsComm Adapter 内置 u32 分帧，
// member 实机对拍定案，public 同一 FsComm 类同一线路）。
const wireHeaderSize = 12

// IdentRegister 注册帧 type 值（官方 0xFFFF：绑定连接↔src ident）。
const IdentRegister uint32 = 0xFFFF

// member→public 外层子类型（ProcessMemberMsg cmp 链 0x8AFF0 定案）。
const (
	MemberForwRegister uint32 = 0 // public_forw_register_t（8B）
	MemberForwData     uint32 = 1 // public_forw_data_t（0x29B）
	MemberForwMessage  uint32 = 2 // {msg_id, argc, 标签值列}
)

// public→member 推送 op（member 镜像 14 型；除注明定向外全部经
// SyncPublic→SendToAllMember 广播）。
const (
	PushSpaces       uint32 = 0x0 // 定向应答：空间清单
	PushData         uint32 = 0x1 // 定向应答：单条数据全量快照
	PushAdd          uint32 = 0x2
	PushRemove       uint32 = 0x3
	PushAddAttr      uint32 = 0x4 // AddAttr 与 SetAttr* 复用（upsert）
	PushRemoveAttr   uint32 = 0x5
	PushAddRec       uint32 = 0x6
	PushRemoveRec    uint32 = 0x7
	PushRecAddRow    uint32 = 0x8
	PushRecRemoveRow uint32 = 0x9
	PushRecClear     uint32 = 0xA
	PushRecChange    uint32 = 0xB
	PushRecSetRow    uint32 = 0xC
	PushMessage      uint32 = 0xD // 定向：SendPublicMessage
)

// world→public（ProcessWorldMsg if/else 两型）。
const (
	WorldServerState uint32 = 0 // public_res_server_state_t 0x24B
	WorldClosePublic uint32 = 1
)

// public→world。
const (
	WorldLoadPlayer       uint32 = 0 // {account[33], wname[33], fromIdent} 0x67B
	WorldRequestState     uint32 = 1 // 1B
	WorldCloseDone        uint32 = 2 // {u32 1}（Closing 收尾）
)

// store→public（ProcessStoreMsg 跳表 0x8D4AC 定案）。
const (
	StoreBackRoleOffline uint32 = 0x1  // {result, count, 行列}
	StoreBackPubInfo     uint32 = 0x13 // 空间元信息应答
	StoreBackPubLoad     uint32 = 0x14 // 单条数据应答（含 blob）
	StoreBackPubSave     uint32 = 0x15 // ACK
	StoreBackPubAdd      uint32 = 0x16 // ACK
	StoreBackPubRemove   uint32 = 0x17 // ACK
)

// public→store。
const (
	StoreRoleOfflineReq uint32 = 0x1  // {ServerId, roleID, 时刻, flag} 0x24B
	StorePubInfoReq     uint32 = 0x1E // {space[33], table[33]} 0x42B
	StorePubLoadReq     uint32 = 0x1F // 链式 {count, index, space, table, wname} 0x8CB
	StorePubSave        uint32 = 0x20 // {头 0x84, blob(saveOnly)}
	StorePubAdd         uint32 = 0x21 // {头 0xA5（含 uid）, blob}
	StorePubRemoveReq   uint32 = 0x22 // {space, table, wname} 0x84B
	StoreSaveLog        uint32 = 0xD  // {wtitle[33], wname[256], wtext[256]} 0x44AB
	StoreCustomLog      uint32 = 0x11 // {file[32], u32, varlist blob}
)

// 定长字段宽度（官方结构体 NUL 填充区）。
const (
	nameLen    = 0x21 // char name[33]
	wnameLen   = 0x42 // wchar name[33]
	pushHdrLen = 0x84 // {space[33], wname[33], key[33]} 公共推送头
)

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

// validSource 复刻 ReceiveMsg 源校验：world(0)/store(4..7)/member(0x18..0x27)。
func validSource(src uint32) bool {
	switch {
	case src == WorldIdent:
		return true
	case src >= StoreIdentBase && src < StoreIdentBase+4:
		return true
	case src >= MemberIdentBase && src <= MemberIdentMax:
		return true
	}
	return false
}

// putFixedString 定长 NUL 填充窄串（GBK 字节截断到 n-1）。
func putFixedString(dst []byte, s string) {
	n := copy(dst, s)
	if n == nameLen {
		n-- // 强制留 NUL 位
	}
	for i := n; i < len(dst); i++ {
		dst[i] = 0
	}
}

// putFixedWideStr 定长 NUL 填充宽串（UTF-16LE，码元截断；尾字节奇余
// 忽略——调用方可能传含后续区的切片）。
func putFixedWideStr(dst []byte, s string) {
	units := utf16.Encode([]rune(s))
	if len(units)*2 > len(dst)-2 {
		units = units[:(len(dst)-2)/2]
	}
	i := 0
	for _, u := range units {
		binary.LittleEndian.PutUint16(dst[i:], u)
		i += 2
	}
	for ; i+1 < len(dst); i += 2 {
		binary.LittleEndian.PutUint16(dst[i:], 0)
	}
	if i < len(dst) { // 奇数尾字节
		dst[i] = 0
	}
}

// getFixedString 读定长窄串（NUL 截断）。
func getFixedString(src []byte) string {
	for i, b := range src {
		if b == 0 {
			return string(src[:i])
		}
	}
	return string(src)
}

// getFixedWideStr 读定长宽串。
func getFixedWideStr(src []byte) string {
	units := make([]uint16, 0, len(src)/2)
	for i := 0; i+1 < len(src); i += 2 {
		u := binary.LittleEndian.Uint16(src[i:])
		if u == 0 {
			break
		}
		units = append(units, u)
	}
	return string(utf16.Decode(units))
}

// PutVaryMessage 官方 put_vary_message：值列线格式 = 每元素
// {byte type, 值按型}（计数在消息头 argc 域，元素区不带计数）。
func PutVaryMessage(a *StoreArchive, vals []Value) error {
	for _, v := range vals {
		if err := a.WriteByte(byte(v.Type)); err != nil {
			return err
		}
		if err := v.PutArchive(a); err != nil {
			return err
		}
	}
	return nil
}

// GetVaryMessage 官方 get_vary_message（argc 从消息头读出后逐元素解）。
func GetVaryMessage(a *LoadArchive, argc int) ([]Value, error) {
	out := make([]Value, 0, argc)
	for i := 0; i < argc; i++ {
		tb, err := a.ReadByte()
		if err != nil {
			return nil, err
		}
		t := VType(tb)
		if !IsValidVType(t) {
			return nil, fmt.Errorf("(get_vary_message)invalid type %d", tb)
		}
		v, err := GetArchive(a, t)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// BuildPushHeader 公共推送头 {space[33], wname[33], key[33]}（0x84B；
// op3 只用前 0x63——调用方截断）。
func BuildPushHeader(space, wname, key string) []byte {
	h := make([]byte, pushHdrLen)
	putFixedString(h[0:], space)
	putFixedWideStr(h[nameLen:nameLen+wnameLen], wname)
	putFixedString(h[nameLen+wnameLen:], key)
	return h
}
