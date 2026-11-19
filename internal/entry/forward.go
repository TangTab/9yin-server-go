package entry

import (
	"encoding/binary"
	"fmt"
)

// entry_forw_* 内网控制协议（CAdapterHook::ProcessMsg 16 型）。
// 编号定案依据 FsEntry.dll 跳表 + 各 case 断言串（adapterhook.cpp）：
//
//	 0 init_connect      ==4   world 就绪旗标：≠0 开放(1)/=0 暂停(0)
//	 1 break_client      ==8   {ident,serial} 踢线
//	 2 set_client_inuse  ==9   {ident,serial,bInuse}
//	 3 set_client_online ==16   {ident,serial,memberID,memberIdent}
//	 4 set_client_posi   ==40   {ident,serial,_,scene,x0,y0,w,h}
//	 5 set_client_move   >=12  {scene,_,nCount}+n×16 {ident,serial,x,y}
//	 6 send_to_sight     >=28  {ident,serial,_,scene,x,y,radius}+payload
//	 7 send_to_client    >=8   {ident,serial}+payload
//	 8 send_all          ——    payload（二级 kind=0xB 全量广播）
//	 9 send_multi        ——    payload（批组播）
//	10 shake_client      ==8   {ident,serial} 保活
//	11 channel_add       ==12  {ident,serial,channel}
//	12 channel_remove    ==12  {ident,serial,channel}
//	13 clear_channel     ==4   {channel}
//	14 send_to_channel   >=4   {channel}+payload
//	15 send_online       ——    payload（二级 kind=2 仅在线广播）
const (
	ForwInitConnect = 0
	ForwBreakClient = 1
	ForwSetInuse    = 2
	ForwSetOnline   = 3
	ForwSetPosi     = 4
	ForwSetMove     = 5
	ForwSendSight   = 6
	ForwSendClient  = 7
	ForwSendAll     = 8
	ForwSendMulti   = 9
	ForwShakeClient = 10
	ForwChannelAdd  = 11
	ForwChannelRm   = 12
	ForwClearChan   = 13
	ForwSendChannel = 14
	ForwSendOnline  = 15
)

// entry→world 上行类型（ProcessEntryMsg 跳表六型——FsServer.pdb 断言
// 逐型钉死，docs/official/FsServer全解与Go复刻.md §5.1）：
//
//	0 register（12B {DistrictID,ServerID,EntryIdx}；ident 校验）
//	1 accept 通知（28B {ident,serial,addr[16],port}）
//	2 client lost（shake 未命中回报 {ident,serial} 8B）
//	3 client 数据（{ident,serial,原始帧} ≥8B）
//	4 非法消息回报
//	5 close entry 确认
//
// 修订记录：旧载 "0=accept" 为 FsSoloEntry 侧误读——world 侧断言实证
// accept=1、0=register；Lost/Data/Illegal 编号不变。
const (
	UpstreamRegister = 0
	UpstreamAccept   = 1
	UpstreamLost     = 2
	UpstreamData     = 3
	UpstreamIllegal  = 4
	UpstreamCloseEntry = 5
)

// innerHeader 内网帧 12 字节头 {src,dst,type}（CAdapterHook::ReceiveMsg）。
type innerHeader struct {
	Src, Dst, Type uint32
}

const innerHeaderSize = 12

// IdentRegister 内网 ident 注册帧的 type 值（官方 0xFFFF 特例）。
const IdentRegister = 0xFFFF

// QuitFrame 内网退出帧 type（CInnerHook::ReceiveMsg 入站 → Core 退出；
// ident 重复时向新连接回发同型帧拒绝注册——FsSoloEntry innerhook.cpp）。
const QuitFrame = 0xFFFE

// WorldIdent world 连接的固定 ident（源校验白名单 0）。
const WorldIdent = 0

// MemberIdentBase member 编号 0..15 映射为内网 ident 0x18+idx。
const MemberIdentBase = 0x18

func memberIdent(idx int) uint32 { return MemberIdentBase + uint32(idx) }

// encodeInner 打包 {src,dst,type}+payload。
func encodeInner(src, dst, typ uint32, payload []byte) []byte {
	out := make([]byte, innerHeaderSize+len(payload))
	binary.LittleEndian.PutUint32(out[0:], src)
	binary.LittleEndian.PutUint32(out[4:], dst)
	binary.LittleEndian.PutUint32(out[8:], typ)
	copy(out[innerHeaderSize:], payload)
	return out
}

// decodeInner 拆包；长度不足返回错误。
func decodeInner(body []byte) (innerHeader, []byte, error) {
	if len(body) < innerHeaderSize {
		return innerHeader{}, nil, fmt.Errorf("inner frame too short: %d", len(body))
	}
	return innerHeader{
		Src:  binary.LittleEndian.Uint32(body[0:]),
		Dst:  binary.LittleEndian.Uint32(body[4:]),
		Type: binary.LittleEndian.Uint32(body[8:]),
	}, body[innerHeaderSize:], nil
}

// validSource 复刻 ReceiveMsg 源校验：0（world）或 0x18..0x27（member）。
func validSource(src uint32) bool {
	return src == WorldIdent || (src >= MemberIdentBase && src < MemberIdentBase+MaxMemberNum)
}

// ---- 上行 payload 构造（entry→world/member） ----

// BuildUpstreamAccept 复刻 OnAccept 28 字节通知 {ident,serial,addr[16],port}。
func BuildUpstreamAccept(ident, serial uint32, addr string, port int) []byte {
	buf := make([]byte, 28)
	binary.LittleEndian.PutUint32(buf[0:], ident)
	binary.LittleEndian.PutUint32(buf[4:], serial)
	copy(buf[8:], addr)
	if len(addr) < 16 {
		buf[8+len(addr)] = 0
	}
	binary.LittleEndian.PutUint32(buf[24:], uint32(port))
	return buf
}

// BuildUpstreamData 复刻转发 {ident,serial,原始客户端帧}。
func BuildUpstreamData(ident, serial uint32, frame []byte) []byte {
	buf := make([]byte, 8+len(frame))
	binary.LittleEndian.PutUint32(buf[0:], ident)
	binary.LittleEndian.PutUint32(buf[4:], serial)
	copy(buf[8:], frame)
	return buf
}

// BuildUpstreamLost 复刻 shake 未命中回报 {ident,serial}。
func BuildUpstreamLost(ident, serial uint32) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint32(buf[0:], ident)
	binary.LittleEndian.PutUint32(buf[4:], serial)
	return buf
}
