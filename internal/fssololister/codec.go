package fssololister

import (
	"encoding/binary"
	"unicode/utf16"
)

// 注册报文方言（FsSoloLister.dll 双 UDP 口收包门 + FsSoloWorld.dll
// BroadcastWorldInfo 发送方互证，2026-11-19 反汇编终局）：
//
//	+0x00  u32  kind：0=world 信息包（len 必须 0x84），1=entry 信息包（0x48）
//	+0x04  u32  魔数 0x3001（FsLister 大写系为 0x300，solo 系实证不同）
//	+0x1C  u32  district_id（分区准入门：≠ DistrictId 丢弃——仅配置口）
//	world_info（0x84=132B，官方比 FsLister 多 12B）：
//	  +0x08 char ip[16] / +0x20 server_id / +0x24 wchar name[≤0x40]
//	  +0x6C max / +0x70 cur / +0x74 visible(u32≠0) / +0x78 recommend
//	entry_info（0x48=72B）：
//	  +0x20 server_id / +0x24 line_index(<0x10) / +0x28 line_id
//	  +0x2C char ip[16] / +0x3C port / +0x40 max / +0x44 cur
const (
	// RegistrationMagic solo 系注册包魔数（RecvWorld/Entry 收包门实证）。
	RegistrationMagic uint32 = 0x3001

	// KindWorld/KindEntry 包 +0x00 的类型值。
	KindWorld uint32 = 0
	KindEntry uint32 = 1

	// WorldPacketLen/EntryPacketLen 官方收包门的精确长度。
	WorldPacketLen = 0x84
	EntryPacketLen = 0x48

	// MaxNameChars 官方 server_t+0x28 wchar name[0x40]（64 码元）。
	MaxNameChars = 0x40
)

// WorldInfo 是 world 注册包的字段投影（RecvWorldServerInfo@10063B60）。
type WorldInfo struct {
	ServerID  int32
	IP        string // 通告地址（char[16] 定长拷贝）
	Name      string // 宽显示名（wchar[0x40]）
	Max       int32
	Cur       int32
	Visible   bool
	Recommend int32 // +0x78 繁忙度推荐值（>10 直通 busy；solo 独有通道）
}

// EntryInfo 是 entry 注册包的字段投影（RecvEntryServerInfo@10063FB0）。
type EntryInfo struct {
	ServerID  int32
	LineIndex int32 // <0x10 必须
	LineID    int32
	IP        string
	Port      int32
	Max       int32
	Cur       int32
}

// ParseRegistration 官方配置口收包门（broad_cfg_cb@0x10062030）：
// len<8 丢弃 → 魔数 ≠0x3001 丢弃 → kind 0/1 各验精确长度 → 分区门
// （pkt+0x1C == districtID）。返回 kind；ok=false 表示静默丢弃。
func ParseRegistration(pkt []byte, districtID int32) (kind uint32, ok bool) {
	if len(pkt) < 8 {
		return 0, false
	}
	if binary.LittleEndian.Uint32(pkt[4:]) != RegistrationMagic {
		return 0, false
	}
	kind = binary.LittleEndian.Uint32(pkt)
	switch kind {
	case KindWorld:
		if len(pkt) != WorldPacketLen {
			return 0, false
		}
	case KindEntry:
		if len(pkt) != EntryPacketLen {
			return 0, false
		}
	default:
		return 0, false
	}
	if int32(binary.LittleEndian.Uint32(pkt[0x1C:])) != districtID {
		return 0, false
	}
	return kind, true
}

// ParseLoopbackRegistration 官方本地口中继门（broad_local_cb@0x10061F90）：
// len<8 → 魔数 → kind∈{0,1}——**无分区门、无长度门**（中继不解析载荷，
// 长度由目的端配置口再验）。
func ParseLoopbackRegistration(pkt []byte) (kind uint32, ok bool) {
	if len(pkt) < 8 {
		return 0, false
	}
	if binary.LittleEndian.Uint32(pkt[4:]) != RegistrationMagic {
		return 0, false
	}
	kind = binary.LittleEndian.Uint32(pkt)
	if kind != KindWorld && kind != KindEntry {
		return 0, false
	}
	return kind, true
}

// ParseWorldInfo 解析通过门的 world 包字段（偏移见包注释）。
func ParseWorldInfo(pkt []byte) WorldInfo {
	return WorldInfo{
		ServerID:  int32(binary.LittleEndian.Uint32(pkt[0x20:])),
		IP:        cstr(pkt[0x08:0x18]),
		Name:      wstr(pkt[0x24:], MaxNameChars),
		Max:       int32(binary.LittleEndian.Uint32(pkt[0x6C:])),
		Cur:       int32(binary.LittleEndian.Uint32(pkt[0x70:])),
		Visible:   binary.LittleEndian.Uint32(pkt[0x74:]) != 0,
		Recommend: int32(binary.LittleEndian.Uint32(pkt[0x78:])),
	}
}

// ParseEntryInfo 解析通过门的 entry 包字段。
func ParseEntryInfo(pkt []byte) EntryInfo {
	return EntryInfo{
		ServerID:  int32(binary.LittleEndian.Uint32(pkt[0x20:])),
		LineIndex: int32(binary.LittleEndian.Uint32(pkt[0x24:])),
		LineID:    int32(binary.LittleEndian.Uint32(pkt[0x28:])),
		IP:        cstr(pkt[0x2C:0x3C]),
		Port:      int32(binary.LittleEndian.Uint32(pkt[0x3C:])),
		Max:       int32(binary.LittleEndian.Uint32(pkt[0x40:])),
		Cur:       int32(binary.LittleEndian.Uint32(pkt[0x44:])),
	}
}

// BuildWorldPacket 构造 world 注册包（发送方方言 = FsSoloWorld.dll
// BroadcastWorldInfo@0x100B55C0 互证：0x84 定长、kind 0、魔数 0x3001）。
// world/entry 拆进程部署时直接使用。
func BuildWorldPacket(districtID int32, w WorldInfo) []byte {
	pkt := make([]byte, WorldPacketLen)
	binary.LittleEndian.PutUint32(pkt[0x00:], KindWorld)
	binary.LittleEndian.PutUint32(pkt[0x04:], RegistrationMagic)
	binary.LittleEndian.PutUint32(pkt[0x1C:], uint32(districtID))
	putCStr(pkt[0x08:0x18], w.IP)
	binary.LittleEndian.PutUint32(pkt[0x20:], uint32(w.ServerID))
	putWStr(pkt[0x24:], w.Name)
	binary.LittleEndian.PutUint32(pkt[0x6C:], uint32(w.Max))
	binary.LittleEndian.PutUint32(pkt[0x70:], uint32(w.Cur))
	if w.Visible {
		binary.LittleEndian.PutUint32(pkt[0x74:], 1)
	}
	binary.LittleEndian.PutUint32(pkt[0x78:], uint32(w.Recommend))
	return pkt
}

// BuildEntryPacket 构造 entry 注册包（0x48 定长、kind 1）。
func BuildEntryPacket(districtID int32, e EntryInfo) []byte {
	pkt := make([]byte, EntryPacketLen)
	binary.LittleEndian.PutUint32(pkt[0x00:], KindEntry)
	binary.LittleEndian.PutUint32(pkt[0x04:], RegistrationMagic)
	binary.LittleEndian.PutUint32(pkt[0x1C:], uint32(districtID))
	binary.LittleEndian.PutUint32(pkt[0x20:], uint32(e.ServerID))
	binary.LittleEndian.PutUint32(pkt[0x24:], uint32(e.LineIndex))
	binary.LittleEndian.PutUint32(pkt[0x28:], uint32(e.LineID))
	putCStr(pkt[0x2C:0x3C], e.IP)
	binary.LittleEndian.PutUint32(pkt[0x3C:], uint32(e.Port))
	binary.LittleEndian.PutUint32(pkt[0x40:], uint32(e.Max))
	binary.LittleEndian.PutUint32(pkt[0x44:], uint32(e.Cur))
	return pkt
}

// cstr/wstr 官方 CopyString/CopyWideStr 定长拷贝方言的读取侧。
func cstr(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func wstr(b []byte, maxChars int) string {
	u := make([]uint16, 0, maxChars)
	for i := 0; i+1 < len(b) && len(u) < maxChars; i += 2 {
		c := binary.LittleEndian.Uint16(b[i:])
		if c == 0 {
			break
		}
		u = append(u, c)
	}
	return string(utf16.Decode(u))
}

func putCStr(dst []byte, s string) {
	for i := 0; i < len(dst); i++ {
		if i < len(s) {
			dst[i] = s[i]
		} else {
			dst[i] = 0
		}
	}
}

func putWStr(dst []byte, s string) {
	u := utf16.Encode([]rune(s))
	for i := 0; i+1 < len(dst); i += 2 {
		var c uint16
		if i/2 < len(u) {
			c = u[i/2]
		}
		binary.LittleEndian.PutUint16(dst[i:], c)
	}
}
