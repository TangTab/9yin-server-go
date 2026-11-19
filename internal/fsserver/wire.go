package fsserver

import (
	"encoding/binary"
	"fmt"
)

// ident（comm_id）命名空间——CAdapterHook::ReceiveMsg 路由分支定案
// （§2.4；与 fsmember/entry 侧同约定）：
//
//	world  = 0        public = 1        store = 4..7
//	entry  = 8..0x17（8+EntryIdx）      member = 0x18..0x27（0x18+MemberIdx）
const (
	WorldIdent      uint32 = 0
	PublicIdent     uint32 = 1
	StoreIdentBase  uint32 = 4
	EntryIdentBase  uint32 = 8
	MemberIdentBase uint32 = 0x18

	MaxMemberNum = 0x10 // MAX_MEMBER_NUM（SendToMember 断言）
	MaxStoreNum  = 4    // ident 4..7
	MaxEntryNum  = 16   // ident 8..0x17
	ExtraServerN = 16   // EXTRA_SERVER_NUM（MemberMsgToExtra 断言）
)

// IdentRegister 注册帧 type 值（官方 0xFFFF 特例：绑定连接 ↔ 源 ident）。
const IdentRegister uint32 = 0xFFFF

// 内网帧：{src u32, dst u32, type u32} + payload（12B 头，外层 u32 长度
// 前缀帧由 fscomm.Adapter 承担）。
const wireHeaderSize = 12

// InnerHeader 内网帧 12B 头。
type InnerHeader struct {
	Src, Dst, Type uint32
}

// EncodeInner 组一条内网帧。
func EncodeInner(h InnerHeader, payload []byte) []byte {
	out := make([]byte, wireHeaderSize+len(payload))
	binary.LittleEndian.PutUint32(out[0:], h.Src)
	binary.LittleEndian.PutUint32(out[4:], h.Dst)
	binary.LittleEndian.PutUint32(out[8:], h.Type)
	copy(out[wireHeaderSize:], payload)
	return out
}

// DecodeInner 拆一条内网帧（不足 12B 返回错误——官方 "message length error"）。
func DecodeInner(frame []byte) (InnerHeader, []byte, error) {
	if len(frame) < wireHeaderSize {
		return InnerHeader{}, nil, fmt.Errorf("(CAdapterHook::ReceiveMsg)message length error")
	}
	h := InnerHeader{
		Src:  binary.LittleEndian.Uint32(frame[0:]),
		Dst:  binary.LittleEndian.Uint32(frame[4:]),
		Type: binary.LittleEndian.Uint32(frame[8:]),
	}
	return h, frame[wireHeaderSize:], nil
}

// world→member 消息类型（§5 member 面 / fsmember.wire World* 同表对拍；
// SyncAllScene/Player/RoleInfo 的调用点常量已逐个实证）。
const (
	MemberCreateScene  uint32 = 0x0
	MemberCreateClone  uint32 = 0x1
	MemberDeleteClone  uint32 = 0x2
	MemberCheckScene   uint32 = 0x3
	MemberSceneAll     uint32 = 0x4
	MemberSceneAdd     uint32 = 0x5
	MemberSceneRemove  uint32 = 0x6
	MemberPlayerAll    uint32 = 0x7
	MemberPlayerAdd    uint32 = 0x8
	MemberPlayerRemove uint32 = 0x9
	MemberPlayerChange uint32 = 0xA
	MemberRoleInfo     uint32 = 0xB
	MemberRoleAdd      uint32 = 0xC
	MemberRoleRemove   uint32 = 0xD
	MemberCloseMember  uint32 = 0x37
)

// world→entry 消息类型（entry_forw 族；§24 forward 表 + EntryMsgRegister
// 回包实证 type 0 init_connect）。
const (
	ForwInitConnect uint32 = 0 // 4B 开放旗标（worldState==Opened?1:0）
	ForwBreakClient uint32 = 1 // {ident,serial} 踢线
	ForwSetInuse    uint32 = 2 // {ident,serial,bInuse}
	ForwSetOnline   uint32 = 3 // {ident,serial,memberID,memberIdent}
	ForwSendClient  uint32 = 7 // {ident,serial}+帧
	ForwShakeClient uint32 = 10
)

// entry→world 消息类型（ProcessEntryMsg 跳表 0x10084D78 六型；结构名
// 断言钉死）。注意：accept=1/register=0/close_entry=5 为本次实证修正
// （§5.1，旧载 §24 "0=accept" 有误）。
const (
	EntryMsgTypeRegister      uint32 = 0 // entry_back_register_t 12B
	EntryMsgTypeClientAccept  uint32 = 1 // entry_back_client_accept_t 28B
	EntryMsgTypeClientClosed  uint32 = 2
	EntryMsgTypeClientMessage uint32 = 3 // {ident,serial}+帧
	EntryMsgTypeClientIllegal uint32 = 4
	EntryMsgTypeCloseEntry    uint32 = 5
)

// EntryRegister entry 注册帧（12B：校验 District/Server 后回 init_connect）。
type EntryRegister struct {
	DistrictID uint32
	ServerID   uint32
	EntryIdx   uint32
}

// Encode …
func (r EntryRegister) Encode() []byte {
	var b [12]byte
	binary.LittleEndian.PutUint32(b[0:], r.DistrictID)
	binary.LittleEndian.PutUint32(b[4:], r.ServerID)
	binary.LittleEndian.PutUint32(b[8:], r.EntryIdx)
	return b[:]
}

// DecodeEntryRegister …
func DecodeEntryRegister(p []byte) (EntryRegister, error) {
	if len(p) != 12 {
		return EntryRegister{}, fmt.Errorf("len == sizeof(entry_back_register_t) (%d)", len(p))
	}
	return EntryRegister{
		DistrictID: binary.LittleEndian.Uint32(p[0:]),
		ServerID:   binary.LittleEndian.Uint32(p[4:]),
		EntryIdx:   binary.LittleEndian.Uint32(p[8:]),
	}, nil
}

// EntryClientAccept 客户端接入通知（28B，§5.1）。
type EntryClientAccept struct {
	Ident  uint32
	Serial uint32
	Addr   [16]byte
	Port   uint32
}

// Encode …
func (a EntryClientAccept) Encode() []byte {
	var b [28]byte
	binary.LittleEndian.PutUint32(b[0:], a.Ident)
	binary.LittleEndian.PutUint32(b[4:], a.Serial)
	copy(b[8:], a.Addr[:])
	binary.LittleEndian.PutUint32(b[24:], a.Port)
	return b[:]
}

// DecodeEntryClientAccept …
func DecodeEntryClientAccept(p []byte) (EntryClientAccept, error) {
	if len(p) != 28 {
		return EntryClientAccept{}, fmt.Errorf("len == sizeof(entry_back_client_accept_t) (%d)", len(p))
	}
	var a EntryClientAccept
	a.Ident = binary.LittleEndian.Uint32(p[0:])
	a.Serial = binary.LittleEndian.Uint32(p[4:])
	copy(a.Addr[:], p[8:24])
	a.Port = binary.LittleEndian.Uint32(p[24:])
	return a, nil
}

// world→store 消息类型（SendToStorage 族调用点常量实证，§5.3）。
const (
	StoreRoleInfoReq    uint32 = 0x0 // 8B 角色名全量请求（Unknown 态启动）
	StoreAccountLogon   uint32 = 0x2 // 0x8F=143B
	StoreLoadCancel     uint32 = 0x3 // 0x6B
	StoreSetOnline      uint32 = 0x5 // 0x6B
	StoreSetOffline     uint32 = 0x6 // 0x6B
	StoreCreateRole     uint32 = 0xA // 0x9B（SendToAllStorage2）
	StoreRolesReq       uint32 = 0xB // 0x7F
	StoreLoadPlayer     uint32 = 0xC // 0x2D=45B
	StoreChargeLogon    uint32 = 0x24
	StoreChargeLogoff   uint32 = 0x25
	StoreChargeValidate uint32 = 0x26
)

// store→world 消息类型（ProcessStoreMsg 跳表 13 型，§5.3）。
const (
	StoreMsgRoleInfo        uint32 = 0
	StoreMsgLoad            uint32 = 1
	StoreMsgSave            uint32 = 2
	StoreMsgBlock           uint32 = 3
	StoreMsgResume          uint32 = 4
	StoreMsgRoles           uint32 = 7
	StoreMsgCreate          uint32 = 5
	StoreMsgDelete          uint32 = 6
	StoreMsgAccountReset    uint32 = 8
	StoreMsgAccountLogon    uint32 = 9
	StoreMsgAccountLogoff   uint32 = 10
	StoreMsgAccountValidate uint32 = 11
	StoreMsgCloseStore      uint32 = 12
)

// member→world 消息类型（ProcessMemberMsg 跳表 27 型，§5.2）。
const (
	MemberMsgTypeCreateClone uint32 = 0
	MemberMsgTypeDeleteClone uint32 = 1
	MemberMsgTypeCloneScene  uint32 = 2
	MemberMsgTypeSetDownTime uint32 = 3
	MemberMsgTypeSceneState  uint32 = 4
	MemberMsgTypeToScene     uint32 = 5
	MemberMsgTypeToAllScene  uint32 = 6
	MemberMsgTypeBreakPlayer uint32 = 7
	MemberMsgTypeSwitchScene uint32 = 8
	MemberMsgTypeNewRole     uint32 = 9
	MemberMsgTypeWorldInfo   uint32 = 10
	MemberMsgTypeSetDelay    uint32 = 11
	MemberMsgTypeQueryGift   uint32 = 12
	MemberMsgTypeGetGift     uint32 = 13
	MemberMsgTypeSellCard    uint32 = 14
	MemberMsgTypeUnsellCard  uint32 = 15
	MemberMsgTypeBuyCard     uint32 = 16
	MemberMsgTypeBuyItem     uint32 = 17
	MemberMsgTypeGetPoints   uint32 = 18
	MemberMsgTypeItemInfo    uint32 = 19
	MemberMsgTypeToCharge    uint32 = 20
	MemberMsgTypeToGmcc      uint32 = 21
	MemberMsgTypeToStat      uint32 = 22
	MemberMsgTypeToManage    uint32 = 23
	MemberMsgTypeToExtra     uint32 = 24
	MemberMsgTypeCloseMember uint32 = 25
	MemberMsgTypeLeaveGame   uint32 = 26
)

// public→world 消息类型（ProcessPublicMsg 三型，§5.4）。
const (
	PublicMsgLoadPlayer  uint32 = 0
	PublicMsgServerState uint32 = 1
	PublicMsgClosePublic uint32 = 2
)

// inner 消息类型（ProcessInnerMsg 跳表 14 型，§5.5）。
const (
	InnerMsgTypeFromGmcc         uint32 = 0
	InnerMsgTypeFromStat         uint32 = 1
	InnerMsgTypeFromManage       uint32 = 2
	InnerMsgTypeFromExtra        uint32 = 3
	InnerMsgTypeStoreConnected   uint32 = 4
	InnerMsgTypeEntryConnected   uint32 = 5
	InnerMsgTypeMemberConnected  uint32 = 6 // {memberIdx}→三段全量同步
	InnerMsgTypeStoreDisconnect  uint32 = 7
	InnerMsgTypeEntryDisconnect  uint32 = 8
	InnerMsgTypeMemberDisconnect uint32 = 9
	InnerMsgTypeOpenServer       uint32 = 10
	InnerMsgTypeCloseServer      uint32 = 11
	InnerMsgTypeDumpPlayers      uint32 = 12
	InnerMsgTypeDumpScenes       uint32 = 13
)

// 客户端事件号（PlayerStateMachine msgID；ClientMsg* 投入值实证，§四）。
const (
	EventTick       uint32 = 0 // 外部驱动（tick）
	EventClosed     uint32 = 2 // 连接关闭（EntryMsgClientClosed 投入值）
	EventGetVerify  uint32 = 4
	EventLogin      uint32 = 5
	EventWorldInfo  uint32 = 6
	EventChooseRole uint32 = 7
	EventCreateRole uint32 = 8
	EventDeleteRole uint32 = 9
)

// 客户端帧 opcode（entry_back_client_message_t 帧首字节分派，§4.2）。
const (
	ClientOpGetVerify  byte = 0
	ClientOpLogin      byte = 2
	ClientOpWorldInfo  byte = 3
	ClientOpChooseRole byte = 4
	ClientOpCreateRole byte = 5
	ClientOpDeleteRole byte = 6
)

// 世界段客户端错误码（{0x03, LE u32} 5B 应答，PlayerStateUnknown 实证）。
const (
	ErrLoginParse  uint32 = 0x4E90 // get account password error / account empty
	ErrRepeatLogin uint32 = 0x4E91 // account repeat login
	ErrServerFull  uint32 = 0x4E92 // OnlineCount >= MaxPlayers
)

// ErrFrame 构造 {0x03, LE u32} 5B 客户端错误帧。
func ErrFrame(code uint32) []byte {
	b := make([]byte, 5)
	b[0] = 0x03
	binary.LittleEndian.PutUint32(b[1:], code)
	return b
}
