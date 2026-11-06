package mpq

import "fmt"

// 归档内文件条目及其块标志位。

// TableSource 是文件条目的数据来源。
type TableSource int

const (
	SourceClassic TableSource = iota // 经典 block table
	SourceBET                        // BET 表（v3+）
)

// 块标志位。
const (
	fileExists       uint32 = 0x80000000
	fileSingleUnit   uint32 = 0x01000000
	fileDeleteMarker uint32 = 0x02000000
	fileSectorCRC    uint32 = 0x04000000
	fileSignature    uint32 = 0x10000000
	filePatch        uint32 = 0x00100000
	fileEncrypted    uint32 = 0x00010000
	fileKeyV2        uint32 = 0x00020000
	fileCompress     uint32 = 0x00000200
	fileImplode      uint32 = 0x00000100
	fileCompressMask uint32 = 0x0000FF00
	// 自定义扩展（Rust 库的密码 AEAD 方案），本模块不支持。
	filePasswordAEAD uint32 = 0x08000000
)

// FileEntry 是归档内的一个文件，统一表示经典表与 BET 表两种来源。
type FileEntry struct {
	// Name 是文件名；未知时为空（来自 (listfile) 的名字会填入）。
	Name string
	// Locale 是经典 hash 条目的 locale（BET 来源恒为 0）。
	Locale uint16
	// BlockIndex 是在源表中的索引。
	BlockIndex uint32
	// Offset 是相对归档起始处的数据偏移。
	Offset uint64
	// CompressedSize 是压缩后大小。
	CompressedSize uint32
	// Size 是未压缩大小。
	Size uint32
	// Flags 是块标志。
	Flags uint32
	// Source 是来源表。
	Source TableSource
}

// Exists 报告条目是否带 EXISTS 标志。
func (e *FileEntry) Exists() bool { return e.Flags&fileExists != 0 }

// IsSingleUnit 报告文件是否按单单元（不切 sector）存储。
func (e *FileEntry) IsSingleUnit() bool { return e.Flags&fileSingleUnit != 0 }

// IsEncrypted 报告文件是否加密。
func (e *FileEntry) IsEncrypted() bool { return e.Flags&fileEncrypted != 0 }

// IsCompressed 报告文件是否压缩（zlib / PKWARE）。
func (e *FileEntry) IsCompressed() bool { return e.Flags&fileCompressMask != 0 }

// DisplayName 返回用于显示的名字；未知时为 <unnamed #index>。
func (e *FileEntry) DisplayName() string {
	if e.Name != "" {
		return e.Name
	}
	return fmt.Sprintf("<unnamed #%d>", e.BlockIndex)
}
