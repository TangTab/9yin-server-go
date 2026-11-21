package fssoloauction

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"

	"github.com/lizhanyin/9yin-server-go/internal/proto"
)

// ---- 内网 TCP 帧（CInnerHook，16003）----
//
//	+0x00 u32 N        // 其后字节数 = 12 + len(payload)
//	+0x04 u32 comm_id  // 发送方 comm 身份
//	+0x08 u32 id1      // 一级路由（member 消息 = 目标索引透传）
//	+0x0C u32 id2      // 消息号（0xFFFF=identify / 0xFFFE=退出）
//	+0x10 payload
const (
	innerHeaderSize  = 16 // 4 字节长度前缀 + 12 字节体头
	innerBodyHeader  = 12 // comm_id + id1 + id2
	innerIdentifyID2 = 0xFFFF
	innerExitID2     = 0xFFFE
	maxInnerFrameLen = 0x3FFFFC // 官方 OUT_BUFFER_LEN-4 = 0x400000-4（N 上限）
	maxForwBytes     = 0x100000 // SendAuctionMessage 1MB 告警阈值（超限仅 trace 仍发送）
)

// comm 身份表（COMM_ID_MAX=64，innerhook.cpp 断言）。
const (
	commWorld      = 0
	commSelf       = 2 // SoloAuction 自身（Start @100AAABD push 2 → SetCommIdent）
	commStoreMin   = 4 // 4..7 Store（本服外连）
	commStoreMax   = 8
	commMemberMin  = 0x18 // 0x18..0x2F Member（游戏服连入）
	commMemberMax  = 0x30
	commIDMax      = 64   // COMM_ID_MAX（innerhook.cpp 断言上界）
	memberIndexMax = 0x18 // SendToMember index < MAX_MEMBER_NUM（SendToMember@100AE503 断言）
)

// encodeInner 封一帧（官方 CInnerHook::SendMsg 栈上 16 字节头 + Send2）。
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

// frameAcc 是按连接的半包累积器（官方 comm_t 内 4 字节长度前缀逐帧
// 凑齐再投递，OnReceive 循环直至数据耗尽——断帧安全）。
type frameAcc struct {
	buf []byte
}

// feed 喂入一段字节，凑齐整帧即回调 onFrame；N < 12 视为损坏报错
// （官方 "message size error"）。
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
			return nil // 半帧留缓冲
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

// ---- member 转发消息（auction_forw_message_t）----
//
//	+0x00 u32 param2   // 官方 OnAuctionMessage int2 原样回传位
//	+0x00 u32 count    // vary 元素个数（个数不带在 vary 流内）
//	+0x08 vary[]       // 无数量前缀元素流（proto.EncodeElements）
//
// 入站断言 len ≥ 8（MemberMessage @100B0D39）；出站超 1MB 仅告警
//（SendAuctionMessage @100AE306，jbe 后仍发送）。

// encodeForw 构造 auction_forw_message_t（官方 SendAuctionMessage：
// buf[0]=sub、buf[4]=GetCount()、Seek(8) 后 put_vary_message）。
func encodeForw(param2 uint32, vl *proto.VarList) []byte {
	body := vl.EncodeElements()
	out := make([]byte, 8, 8+len(body))
	binary.LittleEndian.PutUint32(out[0:], param2)
	binary.LittleEndian.PutUint32(out[4:], uint32(len(vl.Values)))
	return append(out, body...)
}

// decodeForw 解析 auction_forw_message_t（<8 字节报错，等价官方断言）。
func decodeForw(b []byte) (param2 uint32, vl *proto.VarList, err error) {
	if len(b) < 8 {
		return 0, nil, fmt.Errorf("forw 太短 %d（须 ≥ sizeof(auction_forw_message_t)=8）", len(b))
	}
	param2 = binary.LittleEndian.Uint32(b[0:])
	count := int(binary.LittleEndian.Uint32(b[4:]))
	vl, rest, err := proto.DecodeElements(b[8:], count)
	if err != nil {
		return 0, nil, err
	}
	if len(rest) != 0 {
		return 0, nil, fmt.Errorf("forw 尾部多 %d 字节", len(rest))
	}
	return param2, vl, nil
}

// ---- UDP 广播（CReactor::CreateBroadcast/SendBroadcast，15000）----
//
// 本服 announce（BroadcastServerInfo @0x100AEC50，每 TimerCheck 1s）：
//
//	+0x00 u32 type = 5（SoloAuction）
//	+0x04 u32 magic = 0x3001
//	+0x08 char addr[0x10]   InnerAddr（NUL 结尾窄串）
//	+0x18 u32 port          InnerPort(16003)
//	+0x1C u32 district_id
//	+0x20 u32 server_id
//
// 收侧 OnBroadcast @0x100AB780 只认 magic 0x3001：type=1 World 自宣
// （0x48 字节，[0x24]=world 序号 <0x10）→ 外连 comm 0；type=2 Store
// 自宣（0x28 字节，[0x24]=store comm_id ∈4..7）→ 外连该 comm。
const (
	announceMagic  = 0x3001
	announceSize   = 0x24
	announceServer = 5
	announceWorld  = 1
	announceStore  = 2
	worldIndexMax  = 0x10
)

// Announce 是广播报文公共前缀（三种 type 共用 +0x00..+0x27 布局；
// World/Store 报文更长，本服只读前缀 + 序号 [0x24]）。
type Announce struct {
	Type     uint32
	Addr     string // [0x08] char[0x10]，NUL 截断
	Port     uint32 // [0x18]
	District uint32 // [0x1C]
	Server   uint32 // [0x20]
	Index    uint32 // [0x24]（World=序号<0x10；Store=裸 StoreID 0..3——收侧 +4 基址勘误）
}

// encodeAnnounce 构造本服 0x24 字节自宣包。
func encodeAnnounce(addr string, port, district, server uint32) []byte {
	out := make([]byte, announceSize)
	binary.LittleEndian.PutUint32(out[0:], announceServer)
	binary.LittleEndian.PutUint32(out[4:], announceMagic)
	copy(out[8:], addr) // CopyString(dst, 0x10, src) 语义：NUL 结尾截断
	if len(addr) >= 0x10 {
		out[8+0xF] = 0
	}
	binary.LittleEndian.PutUint32(out[0x18:], port)
	binary.LittleEndian.PutUint32(out[0x1C:], district)
	binary.LittleEndian.PutUint32(out[0x20:], server)
	return out
}

// decodeAnnounce 解析收到的广播包；magic 不符或长度不足返回 ok=false
// （官方 OnBroadcast：magic≠0x3001 直接忽略；长度按 type 校验）。
func decodeAnnounce(b []byte) (a Announce, ok bool) {
	if len(b) < announceSize {
		return a, false
	}
	a.Type = binary.LittleEndian.Uint32(b[0:])
	if binary.LittleEndian.Uint32(b[4:]) != announceMagic {
		return a, false
	}
	switch a.Type {
	case announceWorld:
		if len(b) < 0x48 {
			return a, false
		}
	case announceStore:
		if len(b) < 0x28 {
			return a, false
		}
	case announceServer:
		// 自宣包不再处理
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

// cstr 取 NUL 截断窄串。
func cstr(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// fixedStr 取定长窄串域（member_forw_scene_add_t 的 char[0x40] 等）。
func fixedStr(b []byte) string { return cstr(b) }

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
