package mpq

import (
	"encoding/binary"
	"fmt"
)

// 经典 hash table / block table / hi-block table。

// 哨兵值：hash 槽位从未使用（查找在此终止）／指向的文件已被删除。
const (
	hashEntryFree    uint32 = 0xFFFFFFFF
	hashEntryDeleted uint32 = 0xFFFFFFFE
)

// HashEntry 是一个经典 hash table 表项（已解密）。
type HashEntry struct {
	Name1      uint32
	Name2      uint32
	Locale     uint16
	Platform   byte
	FlagsByte  byte
	BlockIndex uint32
}

// isFree 报告槽位是否空闲或已删除（blockIndex >= 0xFFFFFFFE）。
func (e HashEntry) isFree() bool { return e.BlockIndex >= hashEntryDeleted }

// serialize 将表项写入 out（至少 16 字节余量）。
func (e HashEntry) serialize(out []byte) {
	binary.LittleEndian.PutUint32(out[0:], e.Name1)
	binary.LittleEndian.PutUint32(out[4:], e.Name2)
	binary.LittleEndian.PutUint16(out[8:], e.Locale)
	out[10] = e.Platform
	out[11] = e.FlagsByte
	binary.LittleEndian.PutUint32(out[12:], e.BlockIndex)
}

// BlockEntry 是一个经典 block table 表项（已解密）。
type BlockEntry struct {
	Pos   uint32
	CSize uint32
	FSize uint32
	Flags uint32
}

// ClassicTables 持有一份已解密解析的经典表。
type ClassicTables struct {
	HashTable  []HashEntry
	BlockTable []BlockEntry
	// HiBlock 是每个块偏移的高 16 位（可能为空切片）。
	HiBlock []uint16
}

// Find 按名称查找（不分 locale，首个匹配生效）；返回对应 hash 表项。
// 找不到时返回 nil。
func (t *ClassicTables) Find(name string) *HashEntry {
	if len(t.HashTable) == 0 {
		return nil
	}
	normalized := NormalizeName(name)
	start := uint64(HashBytes(normalized, hashTableIndex))
	name1 := HashBytes(normalized, hashNameA)
	name2 := HashBytes(normalized, hashNameB)

	// hash table 大小本应是 2 的幂；用取模让格式古怪的归档也能打开。
	size := uint64(len(t.HashTable))
	index := start % size
	startIndex := index

	for {
		entry := &t.HashTable[index]
		if entry.Name1 == name1 && entry.Name2 == name2 && !entry.isFree() &&
			entry.BlockIndex < uint32(len(t.BlockTable)) {
			return entry
		}
		if entry.BlockIndex == hashEntryFree {
			return nil
		}
		index = (index + 1) % size
		if index == startIndex {
			return nil
		}
	}
}

// BlockOffset 将 hi-block 的 16 位字合并进块偏移（48 位结果）。
func (t *ClassicTables) BlockOffset(blockIndex int) uint64 {
	hi := uint64(0)
	if blockIndex < len(t.HiBlock) {
		hi = uint64(t.HiBlock[blockIndex])
	}
	return hi<<32 | uint64(t.BlockTable[blockIndex].Pos)
}

// parseHashTable 将已解密的 hash table 缓冲区解析为表项。
func parseHashTable(buf []byte) []HashEntry {
	entries := make([]HashEntry, len(buf)/16)
	for i := range entries {
		off := i * 16
		entries[i] = HashEntry{
			Name1:      binary.LittleEndian.Uint32(buf[off:]),
			Name2:      binary.LittleEndian.Uint32(buf[off+4:]),
			Locale:     binary.LittleEndian.Uint16(buf[off+8:]),
			Platform:   buf[off+10],
			FlagsByte:  buf[off+11],
			BlockIndex: binary.LittleEndian.Uint32(buf[off+12:]),
		}
	}
	return entries
}

// parseBlockTable 将已解密的 block table 缓冲区解析为表项。
func parseBlockTable(buf []byte) []BlockEntry {
	entries := make([]BlockEntry, len(buf)/16)
	for i := range entries {
		off := i * 16
		entries[i] = BlockEntry{
			Pos:   binary.LittleEndian.Uint32(buf[off:]),
			CSize: binary.LittleEndian.Uint32(buf[off+4:]),
			FSize: binary.LittleEndian.Uint32(buf[off+8:]),
			Flags: binary.LittleEndian.Uint32(buf[off+12:]),
		}
	}
	return entries
}

// loadMpqTable 读取、解密并（需要时）解压一张经典表。
// storedSize < 0 表示未知存储大小（按未压缩处理）；小于 tableSize 表示已压缩。
func loadMpqTable(reader byteReader, offset int64, storedSize int64, tableSize int, key uint32) ([]byte, error) {
	if tableSize == 0 {
		return nil, nil
	}
	stored := int64(tableSize)
	if storedSize >= 0 && storedSize < stored {
		stored = storedSize
	}
	if stored > int64(maxInt) {
		stored = int64(maxInt)
	}

	raw := reader.ReadAtOrEof(offset, int(stored))
	if key != 0 {
		DecryptBytes(raw, key)
	}
	if int(stored) < tableSize {
		table, err := decompressMasked(raw, tableSize)
		if err != nil {
			return nil, err
		}
		if len(table) < tableSize {
			return nil, fmt.Errorf("mpq: 解压后的表过短")
		}
		return table, nil
	}
	return raw, nil
}

// maxInt 用于 int64 → int 收敛（与 Java 的 Integer.MAX_VALUE 截断一致）。
const maxInt = int(^uint(0) >> 1)
