package mpq

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
)

// MPQ 归档头部（v1–v4）及 MPQ\x1B 用户数据头部。

const (
	mpqMagic      uint32 = 0x1A51504D // 小端 u32 的 "MPQ\x1A"
	userDataMagic uint32 = 0x1B51504D // 小端 u32 的 "MPQ\x1B"（用户数据头部）

	headerSizeV1 = 0x20
	headerSizeV2 = 0x2C
	headerSizeV3 = 0x44
	headerSizeV4 = 0xD0
	// headerMD5Span 是 v4 头部自身 MD5 覆盖的字节数（签名起至 MD5_HetTable 含）。
	headerMD5Span = 0xC0
	// maxHeaderRead 是检查头部候选时最多需要读取的字节数。
	maxHeaderRead = headerSizeV4
)

// FormatVersion 是头部字段声明的 MPQ 格式版本。
type FormatVersion int

// 支持的 MPQ 格式版本（原始字段值 = 序号）。
const (
	V1 FormatVersion = iota
	V2
	V3
	V4
)

// formatVersionFromRaw 解析头部版本字段（0–3）。
func formatVersionFromRaw(v uint16) (FormatVersion, error) {
	if v > 3 {
		return 0, fmt.Errorf("mpq: 无法支持的 MPQ 格式版本字段：%d", v)
	}
	return FormatVersion(v), nil
}

// Raw 返回头部字段的原始值（序号）。
func (v FormatVersion) Raw() uint16 { return uint16(v) }

// HeaderSize 返回该版本头部的字节数。
func (v FormatVersion) HeaderSize() int {
	switch v {
	case V1:
		return headerSizeV1
	case V2:
		return headerSizeV2
	case V3:
		return headerSizeV3
	default:
		return headerSizeV4
	}
}

// Header 是一个已解析的 MPQ 头部；短于版本的尾字段读作 0。
type Header struct {
	HeaderSize     uint32
	ArchiveSize    uint32
	FormatVersionV uint16
	BlockSize      uint16
	HashTablePos   uint64 // v1 32 位；v2+ 追加高 16 位
	BlockTablePos  uint64
	HashTableSize  uint32
	BlockTableSize uint32

	// v2
	HiBlockTablePos uint64

	// v3
	ArchiveSize64 uint64
	BetTablePos   uint64
	HetTablePos   uint64

	// v4
	HashTableSize64    uint64
	BlockTableSize64   uint64
	HiBlockTableSize64 uint64
	HetTableSize64     uint64
	BetTableSize64     uint64
	RawChunkSize       uint32 // 每个文件级 MD5 覆盖的原始块大小（0 = 无）
	MD5BlockTable      [16]byte
	MD5HashTable       [16]byte
	MD5HiBlockTable    [16]byte
	MD5BetTable        [16]byte
	MD5HetTable        [16]byte
	MD5Header          [16]byte
}

// ParseHeader 从 buf 解析头部（长度 >= 0x20）。
func ParseHeader(buf []byte) (*Header, error) {
	if len(buf) < headerSizeV1 {
		return nil, fmt.Errorf("mpq: MPQ 头部不足 0x20 字节")
	}
	if binary.LittleEndian.Uint32(buf[0:]) != mpqMagic {
		return nil, fmt.Errorf("mpq: MPQ 魔数错误")
	}
	version, err := formatVersionFromRaw(binary.LittleEndian.Uint16(buf[0x0C:]))
	if err != nil {
		return nil, err
	}

	h := &Header{}
	h.HeaderSize = binary.LittleEndian.Uint32(buf[0x04:])
	h.ArchiveSize = binary.LittleEndian.Uint32(buf[0x08:])
	h.FormatVersionV = binary.LittleEndian.Uint16(buf[0x0C:])
	h.BlockSize = binary.LittleEndian.Uint16(buf[0x0E:])
	h.HashTablePos = uint64(binary.LittleEndian.Uint32(buf[0x10:]))
	h.BlockTablePos = uint64(binary.LittleEndian.Uint32(buf[0x14:]))
	h.HashTableSize = binary.LittleEndian.Uint32(buf[0x18:])
	h.BlockTableSize = binary.LittleEndian.Uint32(buf[0x1C:])

	// 宽容处理：某些工具写入的 header_size 小于标准 v1 大小。
	if h.HeaderSize < headerSizeV1 {
		h.HeaderSize = headerSizeV1
	}

	if version >= V2 && len(buf) >= headerSizeV2 {
		h.HiBlockTablePos = binary.LittleEndian.Uint64(buf[0x20:])
		h.HashTablePos |= uint64(binary.LittleEndian.Uint16(buf[0x28:])) << 32
		h.BlockTablePos |= uint64(binary.LittleEndian.Uint16(buf[0x2A:])) << 32
	}
	if version >= V3 && len(buf) >= headerSizeV3 {
		h.ArchiveSize64 = binary.LittleEndian.Uint64(buf[0x2C:])
		h.BetTablePos = binary.LittleEndian.Uint64(buf[0x34:])
		h.HetTablePos = binary.LittleEndian.Uint64(buf[0x3C:])
	}
	if version == V4 && len(buf) >= headerSizeV4 {
		h.HashTableSize64 = binary.LittleEndian.Uint64(buf[0x44:])
		h.BlockTableSize64 = binary.LittleEndian.Uint64(buf[0x4C:])
		h.HiBlockTableSize64 = binary.LittleEndian.Uint64(buf[0x54:])
		h.HetTableSize64 = binary.LittleEndian.Uint64(buf[0x5C:])
		h.BetTableSize64 = binary.LittleEndian.Uint64(buf[0x64:])
		h.RawChunkSize = binary.LittleEndian.Uint32(buf[0x6C:])
		copy(h.MD5BlockTable[:], buf[0x70:])
		copy(h.MD5HashTable[:], buf[0x80:])
		copy(h.MD5HiBlockTable[:], buf[0x90:])
		copy(h.MD5BetTable[:], buf[0xA0:])
		copy(h.MD5HetTable[:], buf[0xB0:])
		copy(h.MD5Header[:], buf[0xC0:])
	}
	return h, nil
}

// Plausible 是扫描内嵌归档时的合理性检查。
func (h *Header) Plausible() bool {
	if h.HeaderSize < headerSizeV1 || h.HeaderSize > 0x1000 {
		return false
	}
	// sector 指数合理性检查：storm 实现里只有低字节有意义。
	return h.BlockSize <= 0xFF
}

// SectorSize 返回 sector 大小（字节）：512 << blockSize（低字节），
// 移位语义与 Java int（mod 32）一致。
func (h *Header) SectorSize() uint32 {
	shift := (h.BlockSize & 0xFF) & 31
	return uint32(int32(512) << shift)
}

// EffectiveArchiveSize 返回有效归档大小（相对归档起始处）；
// 宽容回退到实际可用长度。
func (h *Header) EffectiveArchiveSize(fileLenFromBase int64) int64 {
	fromBase := uint64(fileLenFromBase)
	if h.ArchiveSize64 != 0 && h.ArchiveSize64 <= fromBase {
		return int64(h.ArchiveSize64)
	}
	if h.ArchiveSize != 0 && uint64(h.ArchiveSize) <= fromBase {
		return int64(h.ArchiveSize)
	}
	return fileLenFromBase
}

// writeHeader 序列化归档头（v1–v4；v4 校验和由 finishV4Header 填充）。
func writeHeader(version FormatVersion, sectorShift uint16, hashPos uint32,
	hashSize uint32, blockPos uint32, blockCount uint32, archiveSize uint64) []byte {
	h := make([]byte, version.HeaderSize())
	binary.LittleEndian.PutUint32(h[0x00:], mpqMagic)
	binary.LittleEndian.PutUint32(h[0x04:], uint32(len(h)))
	binary.LittleEndian.PutUint32(h[0x08:], uint32(min(archiveSize, 0xFFFFFFFF)))
	binary.LittleEndian.PutUint16(h[0x0C:], version.Raw())
	binary.LittleEndian.PutUint16(h[0x0E:], sectorShift)
	binary.LittleEndian.PutUint32(h[0x10:], hashPos)
	binary.LittleEndian.PutUint32(h[0x14:], blockPos)
	binary.LittleEndian.PutUint32(h[0x18:], hashSize)
	binary.LittleEndian.PutUint32(h[0x1C:], blockCount)
	if version >= V2 {
		binary.LittleEndian.PutUint64(h[0x20:], 0) // hi-block 表位置
		binary.LittleEndian.PutUint16(h[0x28:], 0) // hash table 位置高位
		binary.LittleEndian.PutUint16(h[0x2A:], 0) // block table 位置高位
	}
	if version >= V3 {
		binary.LittleEndian.PutUint64(h[0x2C:], archiveSize) // 64 位归档大小
		binary.LittleEndian.PutUint64(h[0x34:], 0)           // BET 位置
		binary.LittleEndian.PutUint64(h[0x3C:], 0)           // HET 位置
	}
	// v4 独有字段（表大小、MD5）保持为零，由 finishV4Header 填充。
	return h
}

// finishV4Header 填充 v4 独有头字段：显式表存储大小、针对存储后
// （加密、未压缩）字节的表 MD5，以及头部自身 0x00..0xC0 的 MD5，
// 与 StormLib 一致。
func finishV4Header(header, hashBuf, blockBuf []byte) {
	binary.LittleEndian.PutUint64(header[0x44:], uint64(len(hashBuf)))
	binary.LittleEndian.PutUint64(header[0x4C:], uint64(len(blockBuf)))
	// 0x54 hi-block / 0x5C HET / 0x64 BET size64 与 0x6C rawChunkSize 保持 0。
	copy(header[0x70:], md5Sum(blockBuf))
	copy(header[0x80:], md5Sum(hashBuf))
	copy(header[0xC0:], md5Sum(header[:headerMD5Span]))
}

func md5Sum(data []byte) []byte {
	sum := md5.Sum(data)
	return sum[:]
}

// md5Of 以 [16]byte 形式返回 MD5，便于与头部字段直接比较。
func md5Of(data []byte) [16]byte {
	return md5.Sum(data)
}

// md5Present 对应 StormLib IsValidMD5：全零 MD5 表示“不存在”。
func md5Present(md5 [16]byte) bool {
	for _, b := range md5 {
		if b != 0 {
			return true
		}
	}
	return false
}

// userDataHeader 是 MPQ\x1B 用户数据头部（War3/SC2 地图）。
type userDataHeader struct {
	DataSize      uint32
	ArchiveOffset uint32
	HeaderSize    uint32
}

// parseUserDataHeader 在数据不像合理的用户数据头部时返回 nil。
func parseUserDataHeader(buf []byte) *userDataHeader {
	if len(buf) < 16 {
		return nil
	}
	if binary.LittleEndian.Uint32(buf[0:]) != userDataMagic {
		return nil
	}
	ud := &userDataHeader{
		DataSize:      binary.LittleEndian.Uint32(buf[4:]),
		ArchiveOffset: binary.LittleEndian.Uint32(buf[8:]),
		HeaderSize:    binary.LittleEndian.Uint32(buf[12:]),
	}
	// 与 StormLib IsValidMpqUserData 一致的合理性检查。
	if ud.HeaderSize > ud.DataSize || ud.DataSize > ud.ArchiveOffset {
		return nil
	}
	return ud
}
