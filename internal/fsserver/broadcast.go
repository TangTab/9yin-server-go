package fsserver

import (
	"encoding/binary"
	"unicode/utf16"
)

// BroadcastServerInfo 复刻 120B 区服信息 UDP 包（§6.1，与 FsLister
// world 建服包逐域同源）：kind=0/magic=0x300 头。
const (
	BroadcastKind  = 0
	BroadcastMagic = 0x300
	BroadcastSize  = 0x78
)

// BuildBroadcastPacket …
//
// 布局：kind@0 magic@4 addr char[16]@8 port u32@0x18 DistrictId@0x1C
// ServerId@0x20 ServerName wchar[32]@0x24 …@0x6C PlayerCount@0x70
// open@0x74（WorldState==Opened?1:0）。
func BuildBroadcastPacket(addr string, port, district, server uint32, name string, players int, opened bool) []byte {
	b := make([]byte, BroadcastSize)
	binary.LittleEndian.PutUint32(b[0:], BroadcastKind)
	binary.LittleEndian.PutUint32(b[4:], BroadcastMagic)
	copy(b[8:], []byte(addr))
	binary.LittleEndian.PutUint32(b[0x18:], port)
	binary.LittleEndian.PutUint32(b[0x1C:], district)
	binary.LittleEndian.PutUint32(b[0x20:], server)
	units := utf16.Encode([]rune(name))
	if len(units) > 32 {
		units = units[:32]
	}
	for i, u := range units {
		binary.LittleEndian.PutUint16(b[0x24+i*2:], u)
	}
	binary.LittleEndian.PutUint32(b[0x70:], uint32(players))
	if opened {
		binary.LittleEndian.PutUint32(b[0x74:], 1)
	}
	return b
}
