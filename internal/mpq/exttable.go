package mpq

import (
	"encoding/binary"
	"fmt"
)

// HET / BET 表 —— v3+ 的扩展 hash/block 表，布局与查找语义遵循
// StormLib SBaseFileTable.cpp。

// 扩展表签名：小端 u32 的 "HET\x1A" / "BET\x1A"。
const (
	hetSignature uint32 = 0x1A544548
	betSignature uint32 = 0x1A544542
)

// loadExtTable 加载扩展表（HET/BET）：校验签名、解密，必要时解压。
// size 是 MPQ 头给出的磁盘字节大小。
func loadExtTable(reader byteReader, offset int64, size int64, signature uint32, key uint32) ([]byte, error) {
	if offset == 0 || size == 0 {
		return nil, fmt.Errorf("mpq: 扩展表偏移/大小为零")
	}
	capLen := int64(size)
	if capLen > int64(maxInt) {
		capLen = int64(maxInt)
	}
	raw := reader.ReadAtOrEof(offset, int(capLen))
	if len(raw) < 12 {
		return nil, fmt.Errorf("mpq: 扩展表过短")
	}
	sig := binary.LittleEndian.Uint32(raw[0:])
	version := binary.LittleEndian.Uint32(raw[4:])
	dataSize := binary.LittleEndian.Uint32(raw[8:])

	if sig != signature {
		return nil, fmt.Errorf("mpq: 扩展表签名不匹配")
	}
	if version != 1 {
		return nil, fmt.Errorf("mpq: 扩展表版本不是 1")
	}

	// 头部推算的磁盘大小超出实际时，以声明的 data_size 为准。
	want := uint64(dataSize) + 12
	if want < uint64(len(raw)) {
		raw = raw[:want]
	}

	// 解密 12 字节头部之后的内容。
	DecryptBytes(raw[12:], key)

	// data_size 大于磁盘大小意味着负载已压缩。
	if want > uint64(len(raw)) {
		decompressed, err := decompressMasked(raw[12:], int(dataSize))
		if err != nil {
			return nil, err
		}
		table := make([]byte, 12+min(len(decompressed), int(dataSize)))
		copy(table, raw[:12])
		copy(table[12:], decompressed)
		if uint64(len(table)) < want {
			return nil, fmt.Errorf("mpq: 扩展表解压后过短")
		}
		return table, nil
	}
	return raw, nil
}

// -- HET 表 ----------------------------------------------------------------

// HetTable 是已解析的 HET 表。
type HetTable struct {
	// NameHash1 每槽一个探测字节；0 = 空闲槽位。
	NameHash1 []byte
	// BetIndexes 是按位打包的文件索引，每槽 IndexSizeTotal 位。
	BetIndexes      []byte
	EntryCount      uint32
	TotalCount      uint32
	NameHashBitSize int
	IndexSizeTotal  int
	IndexSize       int
}

// parseHetTable 解析 HET 表缓冲区。
func parseHetTable(table []byte) (*HetTable, error) {
	if len(table) < 12+32 {
		return nil, fmt.Errorf("mpq: HET 表过短")
	}
	u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(table[off:]) }
	entryCount := u32(16)
	totalCount := u32(20)
	nameHashBitSize := int(u32(24))
	indexSizeTotal := int(u32(28))
	indexSize := int(u32(36))
	indexTableSize := u32(40)

	if totalCount == 0 || nameHashBitSize == 0 || nameHashBitSize > 64 {
		return nil, fmt.Errorf("mpq: HET 头部字段超出范围")
	}
	if indexSize > indexSizeTotal || indexSizeTotal == 0 || indexSize > 32 {
		return nil, fmt.Errorf("mpq: HET 索引大小超出范围")
	}

	data := table[44:]
	hashesLen := uint64(totalCount)
	indexesComputed := (hashesLen*uint64(indexSizeTotal) + 7) / 8
	var indexesLen uint64 = indexesComputed
	if indexTableSize > 0 && uint64(len(data)) >= hashesLen+uint64(indexTableSize) &&
		abs64(uint64(indexTableSize), indexesComputed) <= 8 {
		indexesLen = uint64(indexTableSize)
	}
	if uint64(len(data)) < hashesLen || uint64(len(data)) < hashesLen+indexesLen {
		return nil, fmt.Errorf("mpq: HET 表主体被截断")
	}
	return &HetTable{
		NameHash1:       data[:hashesLen],
		BetIndexes:      data[hashesLen : hashesLen+indexesLen],
		EntryCount:      entryCount,
		TotalCount:      totalCount,
		NameHashBitSize: nameHashBitSize,
		IndexSizeTotal:  indexSizeTotal,
		IndexSize:       indexSize,
	}, nil
}

func abs64(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// AndMask 返回名称哈希的按位与掩码。
func (t *HetTable) AndMask() uint64 {
	if t.NameHashBitSize == 64 {
		return ^uint64(0)
	}
	return uint64(1)<<uint(t.NameHashBitSize) - 1
}

// OrMask 返回名称哈希的按位或掩码。
func (t *HetTable) OrMask() uint64 {
	return uint64(1) << uint(t.NameHashBitSize-1)
}

// Lookup 用完整（掩码后）名称哈希探测；按探测顺序返回候选文件索引。
func (t *HetTable) Lookup(hash uint64) []uint32 {
	fileHash := (hash & t.AndMask()) | t.OrMask()
	hash1 := byte(fileHash >> uint(t.NameHashBitSize-8))

	total := uint64(t.TotalCount)
	slot := fileHash % total
	start := slot
	var found []uint32

	for {
		probe := t.NameHash1[slot]
		if probe == 0 {
			break // 空闲槽位终止探测
		}
		if probe == hash1 {
			idx := getBits(t.BetIndexes, slot*uint64(t.IndexSizeTotal), t.IndexSize)
			found = append(found, uint32(idx))
		}
		slot = (slot + 1) % total
		if slot == start {
			break
		}
	}
	return found
}

// -- BET 表 ----------------------------------------------------------------

// BetFileRecord 是单条 BET 文件记录，已解析成经典结构字段。
type BetFileRecord struct {
	FilePos  uint64
	FileSize uint32
	CmpSize  uint32
	Flags    uint32
}

// BetTable 是已解析的 BET 表。
type BetTable struct {
	Flags             []uint32
	FileTable         []byte
	NameHashes        []byte
	EntryCount        uint32
	TableEntrySize    uint32
	BitIndexFilePos   uint32
	BitIndexFileSize  uint32
	BitIndexCmpSize   uint32
	BitIndexFlagIndex uint32
	BitCountFilePos   uint32
	BitCountFileSize  uint32
	BitCountCmpSize   uint32
	BitCountFlagIndex uint32
	BitTotalNameHash2 uint32
	BitCountNameHash2 uint32
}

// parseBetTable 解析 BET 表缓冲区。
func parseBetTable(table []byte) (*BetTable, error) {
	if len(table) < 12+76 {
		return nil, fmt.Errorf("mpq: BET 表过短")
	}
	u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(table[off:]) }
	entryCount := u32(16)
	flagCount := u32(84)

	if entryCount > 50_000_000 {
		return nil, fmt.Errorf("mpq: BET 条目数超出范围")
	}

	pos := 88
	data := table[pos:]
	flagsLen := uint64(flagCount) * 4
	if uint64(len(data)) < flagsLen {
		return nil, fmt.Errorf("mpq: BET 标志数组被截断")
	}
	flags := make([]uint32, flagCount)
	for i := range flags {
		flags[i] = binary.LittleEndian.Uint32(data[uint64(i)*4:])
	}
	pos += int(flagsLen)

	tableEntrySize := u32(24)
	fileTableBits := uint64(entryCount) * uint64(tableEntrySize)
	fileTableBytes := int((fileTableBits + 7) / 8)
	if len(table) < pos+fileTableBytes {
		return nil, fmt.Errorf("mpq: BET 文件表被截断")
	}
	fileTable := table[pos : pos+fileTableBytes]
	pos += fileTableBytes

	bitTotalNameHash2 := u32(68)
	hashBytes := int((uint64(entryCount)*uint64(bitTotalNameHash2) + 7) / 8)
	if len(table) < pos+hashBytes {
		return nil, fmt.Errorf("mpq: BET 名称哈希数组被截断")
	}
	nameHashes := table[pos : pos+hashBytes]

	return &BetTable{
		Flags:             flags,
		FileTable:         fileTable,
		NameHashes:        nameHashes,
		EntryCount:        entryCount,
		TableEntrySize:    tableEntrySize,
		BitIndexFilePos:   u32(28),
		BitIndexFileSize:  u32(32),
		BitIndexCmpSize:   u32(36),
		BitIndexFlagIndex: u32(40),
		BitCountFilePos:   u32(48),
		BitCountFileSize:  u32(52),
		BitCountCmpSize:   u32(56),
		BitCountFlagIndex: u32(60),
		BitTotalNameHash2: bitTotalNameHash2,
		BitCountNameHash2: u32(76),
	}, nil
}

// Record 将文件记录 index 解码为经典结构字段。
func (t *BetTable) Record(index int) BetFileRecord {
	base := uint64(index) * uint64(t.TableEntrySize)
	var rec BetFileRecord
	if t.BitCountFilePos > 0 {
		rec.FilePos = getBits(t.FileTable, base+uint64(t.BitIndexFilePos), int(t.BitCountFilePos))
	}
	if t.BitCountFileSize > 0 {
		rec.FileSize = uint32(getBits(t.FileTable, base+uint64(t.BitIndexFileSize), int(t.BitCountFileSize)))
	}
	if t.BitCountCmpSize > 0 {
		rec.CmpSize = uint32(getBits(t.FileTable, base+uint64(t.BitIndexCmpSize), int(t.BitCountCmpSize)))
	}
	if t.BitCountFlagIndex > 0 && len(t.Flags) > 0 {
		flagIndex := int(getBits(t.FileTable, base+uint64(t.BitIndexFlagIndex), int(t.BitCountFlagIndex)))
		if flagIndex < len(t.Flags) {
			rec.Flags = t.Flags[flagIndex]
		}
	}
	return rec
}

// FullNameHash 结合 HET 探测字节为某条记录重建完整的 HET 风格名称哈希。
func (t *BetTable) FullNameHash(hash1 int, index int) uint64 {
	return uint64(hash1&0xFFF)<<uint(t.BitCountNameHash2) | t.NameHash2(index)
}

// NameHash2 返回文件记录 index 所存储的（部分）名称哈希。
func (t *BetTable) NameHash2(index int) uint64 {
	return getBits(t.NameHashes, uint64(index)*uint64(t.BitTotalNameHash2), int(t.BitCountNameHash2))
}

// -- 位操作：LSB 优先，与 StormLib TMPQBits 一致 -----------------------------

// getBits 从绝对位偏移 offset 起读 bits 位（LSB 优先）。
func getBits(data []byte, offset uint64, bits int) uint64 {
	if bits == 0 || bits > 64 {
		return 0
	}
	var result uint64
	for i := 0; i < bits; i++ {
		bitPos := offset + uint64(i)
		byteIdx := bitPos / 8
		bit := uint64(0)
		if byteIdx < uint64(len(data)) {
			bit = uint64(data[byteIdx]>>uint(bitPos%8)) & 1
		}
		result |= bit << uint(i)
	}
	return result
}
