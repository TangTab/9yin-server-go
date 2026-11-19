package fssolomember

import (
	"encoding/binary"
	"errors"
)

// 内网帧（tools\innerhook.h，与 fssoloauction/fssoloentry 同源；§五）：
//
//	+0x00 u32 N        // 其后字节数 = 12+len(payload)；N ≤ 0x3FFFFC
//	+0x04 u32 comm_id  // 发送方 comm 身份（member 自身 = 0x18+MemberID）
//	+0x08 u32 id1      // 一级路由
//	+0x0C u32 id2      // 0xFFFF=identify / 0xFFFE=exit / 业务消息号
//	+0x10 payload
const (
	innerIdentifyID2 = 0xFFFF
	innerExitID2     = 0xFFFE
	innerMaxBodyLen  = 0x3FFFFC

	commWorld     = 0 // world：连入 member 受面，identify 自报
	commPublic    = 1 // public：member 广播发现后外连
	commAuction   = 2 // auction：同上（拍卖行数据服）
	commStoreMin  = 4 // 4..7 store
	commStoreMax  = 8
	commEntryMin  = 8 // 8..0x17 entry（8+EntryIdx）
	commEntryMax  = 0x18
	commMemberMin = 0x18 // 0x18..0x2F member（自身 0x18+MemberID）
	commMemberMax = 0x30
	entryIndexMax = 0x10 // MAX_ENTRY_NUM（CheckEntryConnect 断言）
	storeNumMax   = 4    // MAX_STORE_NUM
	commIDMax     = 64   // COMM_ID_MAX
)

// UDP 15000 服务发现方言（§四）：magic 0x3001 + kind 分流。
const (
	announceMagic = 0x3001

	KindWorld   = 0 // world 自宣 0x84B（member 不处理，world 反向连入）
	KindEntry   = 1 // entry 自宣 0x48B：+0x24 = member idx（<0x10）
	KindStore   = 2 // store 自宣 0x28B：+0x24 = StoreId 原值（CheckStoreConnect 内 +4 得 comm）
	KindMember  = 3 // member 自宣 0x28B（本包为发送方）
	KindPublic  = 4 // public 自宣 0x24B
	KindAuction = 5 // auction 自宣 0x24B

	memberAnnounceSize = 0x28
)

// entry 下行 id2（§八；枚举与 fssoloentry.SoloForw* 对齐——member 是生产方）。
const (
	forwSendClient  = 6  // {serial,type}8B + payload → 单客户端
	forwBroadcast   = 8  // payload → 全量
	forwSendChannel = 13 // {channel}+payload → 频道
	forwSetOnline   = 3  // {ident,serial,member,scene}（场景层产出）
	forwSendMulti   = 4
	forwSendScene   = 5
	forwOnlineSend  = 7
	forwShakeClient = 9
	forwChannelAdd  = 10
	forwChannelRm   = 11
	forwClearChan   = 12
	forwInitConnect = 0
	forwBreakClient = 1
	forwResetInuse  = 2
	forwClose       = 14
)

// 场景命令命名空间（§五/§六；对象内核域，经 SceneRelay 透传）。
const (
	sceneCmdClientMessage = 0x11 // entry 上行统一入口
	sceneCmdLetterBase    = 0x21 // store id2 0x13..0x18 → 0x21..0x26
	sceneCmdStoreBase     = 0x40 // store id2 0xB..0x0E  → 0x40..0x43
)

// store id2 → 场景命令重映射（ProcessInnerMsg 跳表 @0x10295508 定案；
// 0x13..0x18 = store 信件族回执——FsSoloStore MessageSendLetter..RejectLetter
// 的 SendResponse id 逐一闭环）。
var storeID2ToSceneCmd = map[uint32]uint32{
	0xB: 0x40, 0xC: 0x41, 0xD: 0x42, 0xE: 0x43,
	0x13: 0x21, 0x14: 0x22, 0x15: 0x23, 0x16: 0x24, 0x17: 0x25, 0x18: 0x26,
}

// store 本地处理带：ProcessStoreMsg 字节跳表 @0x102A0398 定案——
// 0xF..0x12（chunk 族）+ 0x1E..0x21（manage 族；0x13..0x1D 全是 default
// 空槽）。manage 族 id 与 FsSoloStore MessageManageInfo..Remove 的
// SendResponse id（0x1E..0x21）闭环。
func storeID2Local(id2 uint32) bool {
	return (id2 >= 0xF && id2 <= 0x12) || (id2 >= 0x1E && id2 <= 0x21)
}

// encodeInner 构造内网帧：N(4B) + comm/id1/id2(12B) + payload，
// N = 12+len(payload)（N 之后 的字节数）。
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

// buildMemberAnnounce 官方 BroadcastMemberInfo（@0x10295130）0x28B 自宣包：
//
//	+0x00 kind=3  +0x04 magic=0x3001  +0x08 ip[16]  +0x18 port
//	+0x1C district  +0x20 server  +0x24 member
func buildMemberAnnounce(innerAddr string, innerPort, district, server, member uint32) []byte {
	b := make([]byte, memberAnnounceSize)
	binary.LittleEndian.PutUint32(b[0:], KindMember)
	binary.LittleEndian.PutUint32(b[4:], announceMagic)
	copy(b[8:], innerAddr)
	if len(innerAddr) < 16 {
		b[8+len(innerAddr)] = 0
	}
	binary.LittleEndian.PutUint32(b[0x18:], innerPort)
	binary.LittleEndian.PutUint32(b[0x1C:], district)
	binary.LittleEndian.PutUint32(b[0x20:], server)
	binary.LittleEndian.PutUint32(b[0x24:], member)
	return b
}

// announceInfo 收侧自宣包公共字段的只读视图（on_broadcast @0x10290E70）。
type announceInfo struct {
	kind       uint32
	ip         string
	port       uint32
	district   uint32
	server     uint32
	entryIndex uint32 // kind=1（+0x24）
	comm       uint32 // kind=2（+0x24）
}

// parseAnnounce 门清单：len≥8、magic==0x3001；分区(+0x1C)/服(+0x20) 双门
// 由调用方核对。返回 nil 表示静默丢弃（官方同款）。
func parseAnnounce(p []byte) *announceInfo {
	if len(p) < 8 {
		return nil
	}
	if binary.LittleEndian.Uint32(p[4:]) != announceMagic {
		return nil
	}
	switch binary.LittleEndian.Uint32(p[0:]) {
	case KindPublic:
		if len(p) != 0x24 {
			return nil
		}
	case KindStore:
		if len(p) != 0x28 {
			return nil
		}
	case KindEntry:
		if len(p) != 0x48 {
			return nil
		}
	case KindAuction:
		if len(p) != 0x24 {
			return nil
		}
	default:
		return nil
	}
	info := &announceInfo{
		kind:     binary.LittleEndian.Uint32(p[0:]),
		port:     binary.LittleEndian.Uint32(p[0x18:]),
		district: binary.LittleEndian.Uint32(p[0x1C:]),
		server:   binary.LittleEndian.Uint32(p[0x20:]),
	}
	if end := indexByteZ(p[8:24]); end >= 0 {
		info.ip = string(p[8 : 8+end])
	} else {
		info.ip = string(p[8:24])
	}
	if info.kind == KindEntry {
		info.entryIndex = binary.LittleEndian.Uint32(p[0x24:])
	}
	if info.kind == KindStore {
		info.comm = binary.LittleEndian.Uint32(p[0x24:])
	}
	return info
}

func indexByteZ(b []byte) int {
	for i, c := range b {
		if c == 0 {
			return i
		}
	}
	return -1
}
