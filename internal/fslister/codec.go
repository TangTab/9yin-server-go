package fslister

import (
	"encoding/binary"
	"unicode/utf16"
)

// 注册报文方言（CBroadSockHook::OnReceive@10040840 反汇编定案）：
//
//	+0x00 u32 kind      0=world（建服/刷服） 1=entry（建线/刷线）
//	+0x04 u32 magic     恒 0x300，不符丢弃
//	…kind 特定字段…
//	+0x1C u32 district  必须等于 GameLister.DistrictId，不符丢弃
//
// 长度门：world 0x78(120B)、entry 0x48(72B)，不符丢弃（官方静默，
// 一律不回错）。字段偏移与 FsSoloLister 反汇编记载一致（互证）。
const (
	RegMagic        = 0x300
	KindWorld       = 0
	KindEntry       = 1
	regMinSize      = 8
	districtOff     = 0x1C
	worldPacketSize = 0x78
	entryPacketSize = 0x48

	worldIPOff   = 0x08 // char ip[16]
	worldIDOff   = 0x20 // int server_id
	worldNameOff = 0x24 // wchar name[32]（官方 CopyWideStr 定长 0x40B）
	worldMaxOff  = 0x6C
	worldCurOff  = 0x70
	worldVisOff  = 0x74

	entryIDOff    = 0x20 // int server_id（必须已由 world 包建服）
	entryIndexOff = 0x24 // int line_index < 0x10
	entryLineOff  = 0x28 // int line_id
	entryIPOff    = 0x2C // char ip[16]
	entryPortOff  = 0x3C
	entryMaxOff   = 0x40
	entryCurOff   = 0x44
)

// WorldInfo 是 world 注册包（120B）的解码产物：声明/刷新一个游戏服
// （server_t）：id、显示名（宽）、通告地址、容量、在线、可见性。
type WorldInfo struct {
	ServerID int32
	IP       string
	Name     string
	Max      int32
	Cur      int32
	Visible  bool
}

// EntryInfo 是 entry 注册包（72B）的解码产物：在已存在的服下声明/刷新
// 一条线路（line_t）：线路 id、通告地址、端口、容量、在线。
type EntryInfo struct {
	ServerID  int32
	LineIndex int32 // < MaxLines，官方硬门
	LineID    int32
	IP        string
	Port      int32
	Max       int32
	Cur       int32
}

// ParseRegistration 复刻 OnReceive 报文门：长度 ≥8、魔数 0x300、
// 分区匹配、kind 合法且长度精确。任一不符 → ok=false（静默丢弃）。
func ParseRegistration(pkt []byte, districtID int32) (kind int, ok bool) {
	if len(pkt) < regMinSize {
		return 0, false
	}
	if binary.LittleEndian.Uint32(pkt[4:8]) != RegMagic {
		return 0, false
	}
	k := int(int32(binary.LittleEndian.Uint32(pkt[0:4])))
	var want int
	switch k {
	case KindWorld:
		want = worldPacketSize
	case KindEntry:
		want = entryPacketSize
	default:
		return 0, false
	}
	if len(pkt) != want {
		return 0, false
	}
	if int32(binary.LittleEndian.Uint32(pkt[districtOff:districtOff+4])) != districtID {
		return 0, false
	}
	return k, true
}

// ParseWorldInfo 解码 world 包体（假定已过 ParseRegistration 门）。
func ParseWorldInfo(pkt []byte) WorldInfo {
	return WorldInfo{
		ServerID: int32(binary.LittleEndian.Uint32(pkt[worldIDOff : worldIDOff+4])),
		IP:       fixedString(pkt[worldIPOff : worldIPOff+16]),
		Name:     fixedWideString(pkt[worldNameOff : worldNameOff+0x40]),
		Max:      int32(binary.LittleEndian.Uint32(pkt[worldMaxOff : worldMaxOff+4])),
		Cur:      int32(binary.LittleEndian.Uint32(pkt[worldCurOff : worldCurOff+4])),
		Visible:  binary.LittleEndian.Uint32(pkt[worldVisOff:worldVisOff+4]) != 0,
	}
}

// ParseEntryInfo 解码 entry 包体（假定已过 ParseRegistration 门；
// LineIndex 上限门在 RecvEntryInfo 处按官方顺序执行）。
func ParseEntryInfo(pkt []byte) EntryInfo {
	return EntryInfo{
		ServerID:  int32(binary.LittleEndian.Uint32(pkt[entryIDOff : entryIDOff+4])),
		LineIndex: int32(binary.LittleEndian.Uint32(pkt[entryIndexOff : entryIndexOff+4])),
		LineID:    int32(binary.LittleEndian.Uint32(pkt[entryLineOff : entryLineOff+4])),
		IP:        fixedString(pkt[entryIPOff : entryIPOff+16]),
		Port:      int32(binary.LittleEndian.Uint32(pkt[entryPortOff : entryPortOff+4])),
		Max:       int32(binary.LittleEndian.Uint32(pkt[entryMaxOff : entryMaxOff+4])),
		Cur:       int32(binary.LittleEndian.Uint32(pkt[entryCurOff : entryCurOff+4])),
	}
}

// BuildWorldPacket / BuildEntryPacket 是注册包的构造侧（供 world/entry
// 进程与测试使用——注册是持续心跳式，发送方须周期重发）。

func BuildWorldPacket(districtID int32, w WorldInfo) []byte {
	pkt := make([]byte, worldPacketSize)
	binary.LittleEndian.PutUint32(pkt[0:4], KindWorld)
	binary.LittleEndian.PutUint32(pkt[4:8], RegMagic)
	binary.LittleEndian.PutUint32(pkt[districtOff:districtOff+4], uint32(districtID))
	putFixedString(pkt[worldIPOff:worldIPOff+16], w.IP)
	binary.LittleEndian.PutUint32(pkt[worldIDOff:worldIDOff+4], uint32(w.ServerID))
	putFixedWideString(pkt[worldNameOff:worldNameOff+0x40], w.Name)
	binary.LittleEndian.PutUint32(pkt[worldMaxOff:worldMaxOff+4], uint32(w.Max))
	binary.LittleEndian.PutUint32(pkt[worldCurOff:worldCurOff+4], uint32(w.Cur))
	if w.Visible {
		binary.LittleEndian.PutUint32(pkt[worldVisOff:worldVisOff+4], 1)
	}
	return pkt
}

func BuildEntryPacket(districtID int32, e EntryInfo) []byte {
	pkt := make([]byte, entryPacketSize)
	binary.LittleEndian.PutUint32(pkt[0:4], KindEntry)
	binary.LittleEndian.PutUint32(pkt[4:8], RegMagic)
	binary.LittleEndian.PutUint32(pkt[districtOff:districtOff+4], uint32(districtID))
	binary.LittleEndian.PutUint32(pkt[entryIDOff:entryIDOff+4], uint32(e.ServerID))
	binary.LittleEndian.PutUint32(pkt[entryIndexOff:entryIndexOff+4], uint32(e.LineIndex))
	binary.LittleEndian.PutUint32(pkt[entryLineOff:entryLineOff+4], uint32(e.LineID))
	putFixedString(pkt[entryIPOff:entryIPOff+16], e.IP)
	binary.LittleEndian.PutUint32(pkt[entryPortOff:entryPortOff+4], uint32(e.Port))
	binary.LittleEndian.PutUint32(pkt[entryMaxOff:entryMaxOff+4], uint32(e.Max))
	binary.LittleEndian.PutUint32(pkt[entryCurOff:entryCurOff+4], uint32(e.Cur))
	return pkt
}

// fixedString 定长窄串解码：NUL 截断（官方 CopyString 定长拷贝语义）。
func fixedString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// fixedWideString 定宽宽串解码：UTF-16LE NUL 截断。
func fixedWideString(b []byte) string {
	codes := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u := uint16(b[i]) | uint16(b[i+1])<<8
		if u == 0 {
			break
		}
		codes = append(codes, u)
	}
	return string(utf16.Decode(codes))
}

func putFixedString(dst []byte, s string) {
	n := copy(dst, s)
	for i := n; i < len(dst); i++ {
		dst[i] = 0
	}
}

func putFixedWideString(dst []byte, s string) {
	codes := utf16.Encode([]rune(s))
	for i := 0; i < len(dst)/2; i++ {
		var u uint16
		if i < len(codes) {
			u = codes[i]
		}
		dst[i*2] = byte(u)
		dst[i*2+1] = byte(u >> 8)
	}
}
