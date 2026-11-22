package fssoloworld

import (
	"encoding/binary"
	"errors"
)

// 内网帧（tools\innerhook.h，家族同源；§六）：
//
//	+0x00 u32 N        // 其后字节数 = 12+len(payload)
//	+0x04 u32 comm_id  // 发送方 comm（world 自身 = 0）
//	+0x08 u32 id1
//	+0x0C u32 id2      // 0xFFFF=identify / 业务消息号
//	+0x10 payload
const (
	innerIdentifyID2 = 0xFFFF
	innerMaxBodyLen  = 0x3FFFFC

	commWorld     = 0
	commPublic    = 1
	commAuction   = 2
	commStoreMin  = 4 // 4..7（MAX_STORE_NUM=4）
	commStoreMax  = 8
	commEntryMin  = 8 // 8..0x17（MAX_ENTRY_NUM=0x10）
	commEntryMax  = 0x18
	commMemberMin = 0x18 // 0x18..0x2F（MAX_MEMBER_NUM=0x18）
	commMemberMax = 0x30
	entryIndexMax = 0x10
	storeNumMax   = 4
	memberNumMax  = 0x18
	commIDMax     = 0x30
)

// UDP 广播方言（§五）：magic 0x3001 + kind 分流。
const (
	announceMagic = 0x3001

	KindWorld   = 0
	KindEntry   = 1 // entry 0x48B，idx@+0x24 <0x10
	KindStore   = 2 // store 0x28B，idx@+0x24
	KindMember  = 3 // member 0x28B，idx@+0x24 <0x18
	KindPublic  = 4 // public 0x24B
	KindAuction = 5 // auction 0x24B

	worldAnnounceSize = 0x84
)

// world → entry 下行 id2（SendOpenConnect=8 等；§七/§十）。
const (
	entryInitConnect = 0 // 开服通知（SendOpenConnect）
	entryBreakClient = 1
	entryClose       = 5 // CloseEntry 状态下发
	entrySendClient  = 6 // 单客户端下行
	entryRedirect    = 2 // 重定向 connect（Online 态切线）
)

// encodeInner 构造内网帧（N=12+len）。
func encodeInner(comm, id1, id2 uint32, payload []byte) ([]byte, error) {
	if len(payload) > innerMaxBodyLen {
		return nil, errors.New("inner frame payload too large")
	}
	buf := make([]byte, 16+len(payload))
	binary.LittleEndian.PutUint32(buf[0:], uint32(12+len(payload)))
	binary.LittleEndian.PutUint32(buf[4:], comm)
	binary.LittleEndian.PutUint32(buf[8:], id1)
	binary.LittleEndian.PutUint32(buf[12:], id2)
	copy(buf[16:], payload)
	return buf, nil
}

// frameAcc 半包累积（官方 OnReceive 逐帧消费语义）。
type frameAcc struct {
	buf []byte
}

func (a *frameAcc) feed(data []byte, onFrame func(comm, id1, id2 uint32, payload []byte)) error {
	a.buf = append(a.buf, data...)
	for {
		if len(a.buf) < 16 {
			return nil
		}
		n := int(binary.LittleEndian.Uint32(a.buf[0:4]))
		if n < 12 || n > 12+innerMaxBodyLen {
			a.buf = a.buf[:0]
			return errors.New("(CInnerHook::ReceiveMsg)frame size error")
		}
		if len(a.buf) < 4+n {
			return nil
		}
		comm := binary.LittleEndian.Uint32(a.buf[4:8])
		id1 := binary.LittleEndian.Uint32(a.buf[8:12])
		id2 := binary.LittleEndian.Uint32(a.buf[12:16])
		payload := make([]byte, n-12)
		copy(payload, a.buf[16:4+n])
		a.buf = a.buf[4+n:]
		onFrame(comm, id1, id2, payload)
	}
}

// buildWorldAnnounce 官方 BroadcastWorldInfo（0x100b55c0）0x84B 自宣包：
//
//	+0x00 int 0            +0x04 ident 0x3001
//	+0x08 InnerAddr[0x10]  +0x18 InnerPort
//	+0x1C DistrictId       +0x20 ServerId
//	+0x24 ServerName wchar[0x20]（utf16 定长 32 短）
//	+0x6C MaxPlayers       +0x70 this+0x170 计数
//	+0x74 opened=(state==4)+0x78 在线计数和
func buildWorldAnnounce(innerAddr string, innerPort, district, server uint32,
	name string, maxPlayers int, extraCount uint32, opened bool, online uint32) []byte {
	b := make([]byte, worldAnnounceSize)
	binary.LittleEndian.PutUint32(b[0:], KindWorld)
	binary.LittleEndian.PutUint32(b[4:], announceMagic)
	copy(b[8:], innerAddr)
	if len(innerAddr) < 16 {
		b[8+len(innerAddr)] = 0
	}
	binary.LittleEndian.PutUint32(b[0x18:], innerPort)
	binary.LittleEndian.PutUint32(b[0x1C:], district)
	binary.LittleEndian.PutUint32(b[0x20:], server)
	runes := []rune(name)
	if len(runes) > 31 {
		runes = runes[:31]
	}
	u16 := encodeUTF16(runes)
	for i, v := range u16 {
		binary.LittleEndian.PutUint16(b[0x24+i*2:], v)
	}
	binary.LittleEndian.PutUint32(b[0x6C:], uint32(maxPlayers))
	binary.LittleEndian.PutUint32(b[0x70:], extraCount)
	if opened {
		binary.LittleEndian.PutUint32(b[0x74:], 1)
	}
	binary.LittleEndian.PutUint32(b[0x78:], online)
	return b
}

// announceHead 广播接收帧公共头（§五：kind/ident/addr/port/district/server[,idx]）。
type announceHead struct {
	kind     uint32
	addr     string
	port     uint32
	district uint32
	server   uint32
	idx      uint32
}

// parseAnnounceHead 解析 + 校验 ident==0x3001；返回 ok=false 表示忽略。
func parseAnnounceHead(data []byte) (announceHead, bool) {
	var h announceHead
	if len(data) < 0x24 {
		return h, false
	}
	h.kind = binary.LittleEndian.Uint32(data[0:])
	if binary.LittleEndian.Uint32(data[4:]) != announceMagic {
		return h, false
	}
	if i := indexByteZero(data[8:0x18]); i >= 0 {
		h.addr = string(data[8 : 8+i])
	} else {
		h.addr = string(data[8:0x18])
	}
	h.port = binary.LittleEndian.Uint32(data[0x18:])
	h.district = binary.LittleEndian.Uint32(data[0x1C:])
	h.server = binary.LittleEndian.Uint32(data[0x20:])
	if len(data) >= 0x28 {
		h.idx = binary.LittleEndian.Uint32(data[0x24:])
	}
	return h, true
}

func indexByteZero(b []byte) int {
	for i, v := range b {
		if v == 0 {
			return i
		}
	}
	return -1
}

// encodeUTF16 / decodeUTF16 本地 UCS2 编解码（该 Go 发行版无 encoding/utf16）。
func encodeUTF16(r []rune) []uint16 {
	out := make([]uint16, 0, len(r))
	for _, c := range r {
		if c < 0x10000 {
			out = append(out, uint16(c))
		} else {
			c -= 0x10000
			out = append(out, uint16(0xD800+(c>>10)), uint16(0xDC00+(c&0x3FF)))
		}
	}
	return out
}

func decodeUTF16(u []uint16) []rune {
	out := make([]rune, 0, len(u))
	for i := 0; i < len(u); i++ {
		c := u[i]
		if c >= 0xD800 && c < 0xDC00 && i+1 < len(u) && u[i+1] >= 0xDC00 && u[i+1] < 0xE000 {
			out = append(out, rune(0x10000+(rune(c-0xD800)<<10))+rune(u[i+1]-0xDC00))
			i++
		} else {
			out = append(out, rune(c))
		}
	}
	return out
}

// entry 客户端消息帧（§十 EntryMsgClientMessage）：
// {entry_client_id i32, serial i32, msgtype u8, ...}。
const (
	clientMsgGetVerify  = 0
	clientMsgLogin      = 1
	clientMsgWorldInfo  = 2
	clientMsgChooseRole = 3
	clientMsgCreateRole = 4
	clientMsgDeleteRole = 5
	clientMsgReviveRole = 6
	clientMsgReady      = 8    // op09 ClientReady（进场就绪单字节）
	clientMsgActivity   = 0x09 // op0x0A 活动信封（21B 头 + TVarList 形态才消费）
	clientMsgCustom     = 0x1D // op0x1E 自定义消息（TVarList 数值路由）
	clientMsgPing       = 0x17 // op0x18 心跳（mt 折算后内网号）
)

// decodeClientMsg 拆 8B 头 + msgtype。
func decodeClientMsg(payload []byte) (clientID, serial, msgtype uint32, body []byte, ok bool) {
	if len(payload) < 8 {
		return 0, 0, 0, nil, false
	}
	clientID = binary.LittleEndian.Uint32(payload[0:])
	serial = binary.LittleEndian.Uint32(payload[4:])
	rest := payload[8:]
	if len(rest) < 1 {
		return clientID, serial, 0, nil, false
	}
	return clientID, serial, uint32(rest[0]), rest[1:], true
}
