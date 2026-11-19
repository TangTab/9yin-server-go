package fssolostore

import (
	"encoding/binary"
	"errors"
)

// ---- 内网帧（tools\innerhook.h，与 fssoloauction/fssoloentry/fssolomember
// 同源；官方 SoloStore 受面 OnReceive 逐帧消费语义）----
//
//	+0x00 u32 N        // 其后字节数 = 12+len(payload)；N ≤ 0x3FFFFC
//	+0x04 u32 comm_id  // 发送方 comm 身份（store 自身 = StoreId+4）
//	+0x08 u32 id1      // 一级路由（请求方序号）
//	+0x0C u32 id2      // 0xFFFF=identify / 0xFFFE=exit / 业务消息号
//	+0x10 payload
const (
	innerIdentifyID2 = 0xFFFF
	innerExitID2     = 0xFFFE
	innerMaxBodyLen  = 0x3FFFFC

	commWorld     = 0 // world（serial==0 直连回执）
	commPublic    = 1 // public（serial==1）
	commAuction   = 2 // auction（serial==2）
	commStoreMin  = 4 // 4..7 store（SetCommIdent(StoreId+4)）
	commStoreMax  = 8
	commEntryMin  = 8 // 8..0x17 entry（8+EntryIdx）
	commEntryMax  = 0x18
	commMemberMin = 0x18 // 0x18..0x2F member（0x18+MemberID）
	commMemberMax = 0x30
	storeNumMax   = 4    // MAX_STORE_NUM（member CheckStoreConnect 断言）
	entryIndexMax = 0x10 // MAX_ENTRY_NUM
	memberNumMax  = 0x18 // 0x30-0x18
)

// commSelf 官方 Start @0x100C016F：SetCommIdent(StoreId+4)。
func commSelf(storeID int) uint32 { return uint32(commStoreMin + storeID) }

// ---- UDP 广播自宣（BroadcastStoreInfo @0x100C4D60，0x28B）----
//
//	+0x00 u32 kind=2（store）   +0x04 u32 magic=0x3001
//	+0x08 char inner_addr[0x10] +0x18 u32 inner_port（实际）
//	+0x1C u32 district          +0x20 u32 server   +0x24 u32 store_id（原值）
//
// member 端 CheckStoreConnect（@0x10291190）断言 <4 后 index+4 得 comm。
const (
	announceMagic     = 0x3001
	kindStore         = 2
	storeAnnounceSize = 0x28
)

func buildStoreAnnounce(innerAddr string, innerPort, district, server, storeID uint32) []byte {
	b := make([]byte, storeAnnounceSize)
	binary.LittleEndian.PutUint32(b[0:], kindStore)
	binary.LittleEndian.PutUint32(b[4:], announceMagic)
	copy(b[8:], innerAddr)
	if len(innerAddr) < 16 {
		b[8+len(innerAddr)] = 0
	}
	binary.LittleEndian.PutUint32(b[0x18:], innerPort)
	binary.LittleEndian.PutUint32(b[0x1C:], district)
	binary.LittleEndian.PutUint32(b[0x20:], server)
	binary.LittleEndian.PutUint32(b[0x24:], storeID)
	return b
}

// ---- serial 编码（CheckResponse @0x100C51C0 定案）----
//
// 高 16 位 = 源 comm：0/1/2（world/public/auction 直连）或 0x18+memberIdx；
// 低 16 位 = 请求方 id1（回包原样带回）。

func encodeSerial(srcComm uint32, reqSerial uint32) uint32 {
	return (srcComm << 16) | (reqSerial & 0xFFFF)
}

// decodeSerial 回执还源：dest=0 world / 1 public / 2 auction /
// memberIdx（0..0x17）+reqSerial。官方 else 分支：
// comm=(serial>>16)-0x18, idx=serial&0xFFFF → SendToMember(comm, idx, …)。
func decodeSerial(serial uint32) (memberIdx int, reqSerial uint32, direct int, ok bool) {
	hi := serial >> 16
	lo := serial & 0xFFFF
	switch hi {
	case commWorld, commPublic, commAuction:
		return 0, lo, int(hi), true
	default:
		idx := int(hi) - commMemberMin
		if idx < 0 || idx >= memberNumMax {
			return 0, 0, 0, false
		}
		return idx, lo, -1, true
	}
}

// encodeInner 构造内网帧（N = 12+len(payload)）。
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
type frameAcc struct{ buf []byte }

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

func le32(b []byte, off int) uint32 { return binary.LittleEndian.Uint32(b[off:]) }

// putLe32 写 u32（构造响应 payload 用）。
func putLe32(b []byte, off int, v uint32) { binary.LittleEndian.PutUint32(b[off:], v) }
