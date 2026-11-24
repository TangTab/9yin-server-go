package mpq

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"strings"
	"sync"
)

// 一个已打开的 MPQ 归档（只读）：头部定位（含 MPQ\x1B 用户数据与内嵌
// 归档扫描）、经典表与 HET/BET 表、按名查找与读取（sector / 单单元、
// 加密、zlib / PKWARE 解压）。
//
// 与参考 Java 实现一致：不做缺失 (listfile) 时的名字恢复，也不解析
// (attributes)；不支持自定义的密码 AEAD 扩展（遇到即明确报错）。
// 实例可并发读取（读取告警经内部锁保护）。

// SpecialFiles 是可能存在也可能不存在的内部文件名（均为可选）。
var SpecialFiles = []string{
	"(listfile)", "(attributes)", "(signature)", "(userdata)", "(password)",
}

// Archive 是一个已打开的 MPQ 归档。
type Archive struct {
	reader     byteReader
	base       int64
	fileLen    int64
	header     *Header
	version    FormatVersion
	sectorSize uint32
	classic    *ClassicTables // nil = 没有任何经典表内容
	het        *HetTable
	bet        *BetTable
	names      []string // 来自 (listfile) 的文件名
	warnings   []string // 打开归档时发现的非致命问题

	mu           sync.Mutex
	readWarnings []string // 读取文件数据时遇到的非致命问题（有界）
}

// newArchive 从 byteReader 构建归档。
func newArchive(reader byteReader) (*Archive, error) {
	fileLen := reader.Length()
	base, err := locateHeader(reader, fileLen)
	if err != nil {
		return nil, err
	}
	header, err := ParseHeader(reader.ReadAtOrEof(base, maxHeaderRead))
	if err != nil {
		return nil, err
	}
	a := &Archive{
		reader:  reader,
		fileLen: fileLen,
		base:    base,
		header:  header,
		version: FormatVersion(header.FormatVersionV),
	}
	tables, err := a.loadTables()
	if err != nil {
		return nil, err
	}
	a.classic = tables.classic
	a.het = tables.het
	a.bet = tables.bet
	a.sectorSize = tables.sectorSize
	a.loadListfile()
	return a, nil
}

// Open 从文件打开归档。
func Open(path string) (*Archive, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	archive, err := newArchive(&fileReader{f: f})
	if err != nil {
		f.Close()
		return nil, err
	}
	return archive, nil
}

// OpenBytes 从内存字节打开归档。
func OpenBytes(data []byte) (*Archive, error) {
	return newArchive(&memReader{data: data})
}

// Close 释放底层文件句柄（内存归档为无操作）。
func (a *Archive) Close() error {
	if fr, ok := a.reader.(*fileReader); ok {
		return fr.f.Close()
	}
	return nil
}

// -- 访问器 ----------------------------------------------------------------

// Version 返回 MPQ 格式版本（1–4）。
func (a *Archive) Version() int { return int(a.version) + 1 }

// FormatVersion 返回 MPQ 格式版本枚举。
func (a *Archive) FormatVersion() FormatVersion { return a.version }

// SectorSize 返回 sector 大小（字节）。
func (a *Archive) SectorSize() uint32 { return a.sectorSize }

// ArchiveOffset 返回归档在所在文件中的偏移。
func (a *Archive) ArchiveOffset() int64 { return a.base }

// Warnings 返回打开归档时发现的非致命问题。
func (a *Archive) Warnings() []string {
	out := make([]string, len(a.warnings))
	copy(out, a.warnings)
	return out
}

// ReadWarnings 返回读取文件数据时遇到的非致命问题（最近 64 条）。
func (a *Archive) ReadWarnings() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, len(a.readWarnings))
	copy(out, a.readWarnings)
	return out
}

// FileNames 返回来自 (listfile) 的文件名（缺失时为空）。
func (a *Archive) FileNames() []string {
	out := make([]string, len(a.names))
	copy(out, a.names)
	return out
}

// IsPasswordProtected 报告归档是否被自定义密码 AEAD 扩展保护（本模块不支持）。
func (a *Archive) IsPasswordProtected() bool { return a.HasFile("(password)") }

// HasFile 报告归档中是否存在该名字的文件。
func (a *Archive) HasFile(name string) bool {
	e := a.findEntry(name)
	return e != nil && e.Exists()
}

// FindFile 按名字查找文件；返回附带名字的条目。
func (a *Archive) FindFile(name string) (FileEntry, bool) {
	entry := a.findEntry(name)
	if entry == nil {
		return FileEntry{}, false
	}
	entry.Name = name
	return *entry, true
}

// ReadFile 按名字读取文件内容。
func (a *Archive) ReadFile(name string) ([]byte, error) {
	entry := a.findEntry(name)
	if entry == nil {
		return nil, fmt.Errorf("mpq: 归档中找不到文件：%s", name)
	}
	// 附上名字：加密文件的密钥推导依赖归档内完整文件名。
	entry.Name = name
	return a.ReadEntry(*entry)
}

// Entries 列出所有文件条目（已知名字的附带名字）。
func (a *Archive) Entries() []FileEntry {
	var result []FileEntry

	if a.classic != nil && len(a.classic.BlockTable) > 0 {
		// 经典表：遍历 hash 条目以正确处理 locale 与重复名字。
		seen := make([]bool, len(a.classic.BlockTable))
		for _, hash := range a.classic.HashTable {
			if hash.isFree() {
				continue
			}
			idx := hash.BlockIndex
			if idx >= uint32(len(seen)) || seen[idx] {
				continue
			}
			seen[idx] = true
			result = append(result, a.fileEntryFromClassic(int(idx), hash.Locale))
		}
	} else if a.bet != nil {
		for i := 0; i < int(a.bet.EntryCount); i++ {
			result = append(result, a.fileEntryFromBet(i))
		}
	}

	// 附上名字：(listfile) 条目优先，再是已知特殊文件。
	slotOf := make(map[uint64]int, len(result)*2)
	for i := range result {
		k := tableKey(result[i].Source, result[i].BlockIndex)
		if _, ok := slotOf[k]; !ok {
			slotOf[k] = i
		}
	}
	candidates := append(append([]string(nil), a.names...), SpecialFiles...)
	for _, name := range candidates {
		entry := a.findEntry(name)
		if entry == nil || !entry.Exists() {
			continue
		}
		if slot, ok := slotOf[tableKey(entry.Source, entry.BlockIndex)]; ok && result[slot].Name == "" {
			result[slot].Name = name
		}
	}
	return result
}

func tableKey(source TableSource, index uint32) uint64 {
	return uint64(source)<<32 | uint64(index)
}

// -- 查找 ------------------------------------------------------------------

func (a *Archive) findEntry(name string) *FileEntry {
	// 经典 hash table 优先（与 StormLib 一致）。
	if a.classic != nil && len(a.classic.HashTable) > 0 {
		hash := a.classic.Find(name)
		if hash != nil && hash.BlockIndex < uint32(len(a.classic.BlockTable)) {
			entry := a.fileEntryFromClassic(int(hash.BlockIndex), hash.Locale)
			return &entry
		}
	}

	// HET+BET 回退路径。
	if a.het != nil && a.bet != nil {
		fileHash := FileNameHash(name)
		masked := (fileHash & a.het.AndMask()) | a.het.OrMask()
		hash1 := int(masked >> uint(a.het.NameHashBitSize-8) & 0xFF)
		for _, index := range a.het.Lookup(fileHash) {
			if a.bet.FullNameHash(hash1, int(index)) == masked {
				return a.fileEntryFromBetPtr(int(index))
			}
		}
	}
	return nil
}

func (a *Archive) fileEntryFromClassic(index int, locale uint16) FileEntry {
	block := a.classic.BlockTable[index]
	return FileEntry{
		Locale:         locale,
		BlockIndex:     uint32(index),
		Offset:         a.classic.BlockOffset(index),
		CompressedSize: block.CSize,
		Size:           block.FSize,
		Flags:          block.Flags,
		Source:         SourceClassic,
	}
}

func (a *Archive) fileEntryFromBet(index int) FileEntry {
	entry := a.fileEntryFromBetPtr(index)
	return *entry
}

func (a *Archive) fileEntryFromBetPtr(index int) *FileEntry {
	if index < 0 || uint32(index) >= a.bet.EntryCount {
		return nil
	}
	rec := a.bet.Record(index)
	return &FileEntry{
		BlockIndex:     uint32(index),
		Offset:         rec.FilePos,
		CompressedSize: rec.CmpSize,
		Size:           rec.FileSize,
		Flags:          rec.Flags,
		Source:         SourceBET,
	}
}

// -- 读取 ------------------------------------------------------------------

// ReadEntry 按给定的 FileEntry（来自 Entries）读取文件内容。并发安全。
func (a *Archive) ReadEntry(entry FileEntry) ([]byte, error) {
	if !entry.Exists() {
		return nil, fmt.Errorf("mpq: 文件不存在：%s", entry.DisplayName())
	}
	a.mu.Lock()
	a.verifyChunkMd5s(&entry)
	a.mu.Unlock()

	if entry.Flags&filePasswordAEAD != 0 {
		return nil, fmt.Errorf("mpq: %s 使用 Rust mpq 库的密码 AEAD 扩展存储，"+
			"本模块不支持（请在 Rust 端 unprotect 后再处理）", entry.DisplayName())
	}

	var key uint32
	if entry.IsEncrypted() {
		if entry.Name == "" {
			return nil, fmt.Errorf("mpq: 文件已加密且名字未知，无法推导解密密钥")
		}
		key = FileKey(entry.Name, entry.Flags&fileKeyV2 != 0, entry.Offset, entry.Size)
	}

	rawStart := a.base + int64(entry.Offset)

	if entry.IsSingleUnit() {
		data := a.reader.ReadAtOrEof(rawStart, int(entry.CompressedSize))
		if entry.IsEncrypted() {
			DecryptBytes(data, key)
		}
		if entry.IsCompressed() {
			var err error
			switch {
			case entry.Flags&fileCompress != 0:
				data, err = decompressMasked(data, int(entry.Size))
			case entry.Flags&fileImplode != 0:
				data, err = decompressImploded(data, int(entry.Size))
			}
			if err != nil {
				return nil, err
			}
		}
		return trimToSize(data, entry.Size), nil
	}

	// 基于 sector 的文件。
	sectorSize := uint64(a.sectorSize)
	nsec := int((uint64(entry.Size) + sectorSize - 1) / sectorSize)
	compressed := entry.IsCompressed()
	encrypted := entry.IsEncrypted()
	hasCrc := entry.Flags&fileSectorCRC != 0

	// sector 偏移表只存在于压缩文件中。
	var offsets []uint32
	if compressed {
		tableDwords := nsec + 1
		if hasCrc {
			tableDwords += nsec
		}
		raw := a.reader.ReadAtOrEof(rawStart, tableDwords*4)
		if encrypted {
			DecryptBytes(raw, key-1)
		}
		offsets = make([]uint32, tableDwords)
		for i := range offsets {
			offsets[i] = binary.LittleEndian.Uint32(raw[i*4:])
		}
	}

	if compressed {
		// 与 StormLib 一致的健全性检查：偏移单调递增、sector 大小合理。
		for i := 0; i < nsec; i++ {
			a32, b32 := ^uint32(0), ^uint32(0) // u32::MAX 语义
			if i < len(offsets) {
				a32 = offsets[i]
			}
			if i+1 < len(offsets) {
				b32 = offsets[i+1]
			}
			if b32 < a32 || uint64(b32)-uint64(a32) > sectorSize || b32 > entry.CompressedSize {
				return nil, fmt.Errorf("mpq: sector 偏移表不合理（已损坏的归档）")
			}
		}
	}

	out := make([]byte, entry.Size)
	outPos := 0
	for i := 0; i < nsec; i++ {
		expected := uint64(entry.Size) - uint64(i)*sectorSize
		if expected > sectorSize {
			expected = sectorSize
		}

		var rawOff, rawLen uint64
		if compressed {
			rawOff = uint64(offsets[i])
			rawLen = uint64(offsets[i+1]) - uint64(offsets[i])
		} else {
			rawOff = uint64(i) * sectorSize
			rawLen = expected
		}

		if rawLen == 0 {
			// 稀疏/缺失的 sector：以零填充（out 已零初始化）。
			outPos += int(expected)
			continue
		}

		raw := a.reader.ReadAtOrEof(rawStart+int64(rawOff), int(rawLen))
		if encrypted {
			DecryptBytes(raw, key+uint32(i))
		}

		sector := raw
		if compressed && uint64(len(raw)) < expected {
			var err error
			if entry.Flags&fileCompress != 0 {
				sector, err = decompressMasked(raw, int(expected))
			} else {
				sector, err = decompressImploded(raw, int(expected))
			}
			if err != nil {
				return nil, err
			}
		}

		take := min(len(sector), int(expected))
		if len(sector) < int(expected) {
			outPos += int(expected) - len(sector) // 零填充在前，与参考实现一致
		}
		copy(out[outPos:], sector[:take])
		outPos += take

		// 可选的逐 sector CRC：不匹配仅告警。
		if hasCrc {
			crcIndex := nsec + 1 + i
			if crcIndex < len(offsets) {
				stored := offsets[crcIndex]
				if stored != 0 && stored != ^uint32(0) {
					if crc := crc32.ChecksumIEEE(sector[:take]); crc != stored {
						a.addReadWarning(fmt.Sprintf("%s: sector %d CRC 不匹配",
							entry.DisplayName(), i))
					}
				}
			}
		}
	}
	return trimToSize(out, entry.Size), nil
}

// trimToSize 将数据截断或零填充到声明大小（宽容处理，与参考实现一致）。
func trimToSize(data []byte, size uint32) []byte {
	want := int(size)
	if len(data) == want {
		return data
	}
	if len(data) > want {
		return data[:want]
	}
	out := make([]byte, want)
	copy(out, data)
	return out
}

func (a *Archive) addReadWarning(msg string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.readWarnings) < 64 {
		a.readWarnings = append(a.readWarnings, msg)
	}
}

// verifyChunkMd5s 做 v4 raw-chunk 校验：文件存储数据后跟每块 MD5；
// 不匹配仅告警。调用方持有 a.mu。
func (a *Archive) verifyChunkMd5s(entry *FileEntry) {
	chunk := a.header.RawChunkSize
	if a.version != V4 || chunk == 0 || entry.CompressedSize == 0 || len(a.readWarnings) >= 64 {
		return
	}
	start := a.base + int64(entry.Offset)
	rawLen := uint64(entry.CompressedSize)
	count := (rawLen + uint64(chunk) - 1) / uint64(chunk)
	md5Array := rawLen + count*16
	if start+int64(md5Array) > a.fileLen {
		return
	}
	all := a.reader.ReadAtOrEof(start, int(md5Array))
	for i := uint64(0); i < count; i++ {
		from := i * uint64(chunk)
		to := min((i+1)*uint64(chunk), rawLen)
		var expected [16]byte
		copy(expected[:], all[rawLen+i*16:rawLen+(i+1)*16])
		if md5Of(all[from:to]) != expected {
			a.addReadWarning(fmt.Sprintf("%s: raw chunk %d MD5 不匹配", entry.DisplayName(), i))
		}
	}
}

// -- 打开流程 --------------------------------------------------------------

// locateHeader 在流中查找 MPQ 头所在的字节偏移。
func locateHeader(reader byteReader, fileLen int64) (int64, error) {
	if fileLen < 0x20 {
		return 0, fmt.Errorf("mpq: 文件中找不到 MPQ 归档头部")
	}

	probe := reader.ReadAtOrEof(0, maxHeaderRead)
	if binary.LittleEndian.Uint32(probe[0:]) == mpqMagic {
		return 0, nil
	}

	if ud := parseUserDataHeader(probe); ud != nil {
		target := int64(ud.ArchiveOffset)
		if target+0x20 <= fileLen {
			at := reader.ReadAtOrEof(target, maxHeaderRead)
			if binary.LittleEndian.Uint32(at[0:]) == mpqMagic {
				return target, nil
			}
		}
	}

	// 扫描内嵌归档（按 0x200 对齐，与 storm.dll 相同；上限 128 MB）。
	limit := min(fileLen, 0x0800_0000)
	const chunk = 0x10_0000
	offset := int64(0)

scan:
	for offset+0x20 <= limit {
		want := int(min(limit-offset, chunk))
		w := reader.ReadAtOrEof(offset, want+0x200)

		for inOff := 0; inOff+0x20 <= want; inOff += 0x200 {
			magic := binary.LittleEndian.Uint32(w[inOff:])
			if magic == userDataMagic {
				end := inOff + 16
				if end > len(w) {
					end = len(w)
				}
				if ud := parseUserDataHeader(w[inOff:end]); ud != nil {
					target := offset + int64(inOff) + int64(ud.ArchiveOffset)
					if target+0x20 <= fileLen {
						at := reader.ReadAtOrEof(target, maxHeaderRead)
						if binary.LittleEndian.Uint32(at[0:]) == mpqMagic {
							return target, nil
						}
					}
				}
			} else if magic == mpqMagic {
				end := inOff + maxHeaderRead
				if end > len(w) {
					end = len(w)
				}
				if h, err := ParseHeader(w[inOff:end]); err == nil && h.Plausible() {
					return offset + int64(inOff), nil
				}
			}
		}
		if want < chunk {
			break scan
		}
		offset += chunk
	}

	return 0, fmt.Errorf("mpq: 文件中找不到 MPQ 归档头部")
}

// loadedTables 是 loadTables 的产物。
type loadedTables struct {
	classic    *ClassicTables
	het        *HetTable
	bet        *BetTable
	sectorSize uint32
}

// loadTables 加载 hash/block/hi-block 与 HET/BET 表，逐表容错降级为警告。
func (a *Archive) loadTables() (*loadedTables, error) {
	fromBase := a.fileLen - min(a.base, a.fileLen)
	archiveEnd := a.base + a.header.EffectiveArchiveSize(fromBase)

	// v4：头部自带 0x00..0xC0 的 MD5，不匹配仅告警。
	if a.version == V4 && md5Present(a.header.MD5Header) && a.base+headerMD5Span <= a.fileLen {
		raw := a.reader.ReadAtOrEof(a.base, headerMD5Span)
		if md5Of(raw) != a.header.MD5Header {
			a.warnings = append(a.warnings, "v4 头部 MD5 不匹配（0x00-0xC0）")
		}
	}

	stored := a.storedTableSizes(archiveEnd)
	hashStored, blockStored := stored[0], stored[1]

	var hashTable []HashEntry
	var blockTable []BlockEntry
	var hiBlock []uint16

	if a.header.HashTableSize != 0 {
		tableSize := clampInt(uint64(a.header.HashTableSize) * 16)
		buf, err := loadMpqTable(a.reader, a.base+int64(a.header.HashTablePos),
			hashStored, tableSize, KeyHashTable)
		if err != nil {
			a.warnings = append(a.warnings, fmt.Sprintf("hash table 不可用：%v", err))
		} else {
			hashTable = parseHashTable(buf)
			storedLen := int64(tableSize)
			if hashStored >= 0 {
				storedLen = hashStored
			}
			a.verifyStoredMd5("hash table", a.base+int64(a.header.HashTablePos),
				storedLen, a.header.MD5HashTable)
		}
	}

	if a.header.BlockTableSize != 0 {
		tableSize := clampInt(uint64(a.header.BlockTableSize) * 16)
		buf, err := loadMpqTable(a.reader, a.base+int64(a.header.BlockTablePos),
			blockStored, tableSize, KeyBlockTable)
		if err != nil {
			a.warnings = append(a.warnings, fmt.Sprintf("block table 不可用：%v", err))
		} else {
			blockTable = parseBlockTable(buf)
			storedLen := int64(tableSize)
			if blockStored >= 0 {
				storedLen = blockStored
			}
			a.verifyStoredMd5("block table", a.base+int64(a.header.BlockTablePos),
				storedLen, a.header.MD5BlockTable)
		}
	}

	if a.header.HiBlockTablePos != 0 && len(blockTable) > 0 {
		buf := a.reader.ReadAtOrEof(a.base+int64(a.header.HiBlockTablePos), len(blockTable)*2)
		a.verifyStoredMd5("hi-block table", a.base+int64(a.header.HiBlockTablePos),
			int64(len(blockTable)*2), a.header.MD5HiBlockTable)
		for i := 0; i+1 < len(buf); i += 2 {
			hiBlock = append(hiBlock, binary.LittleEndian.Uint16(buf[i:]))
		}
	}

	var classicTables *ClassicTables
	if len(hashTable) > 0 || len(blockTable) > 0 {
		classicTables = &ClassicTables{
			HashTable:  hashTable,
			BlockTable: blockTable,
			HiBlock:    hiBlock,
		}
	}

	// HET/BET 表（v3+）。
	var hetTable *HetTable
	var betTable *BetTable
	if a.version >= V3 {
		if a.header.HetTablePos != 0 {
			size := a.hetStoredSize(archiveEnd)
			buf, err := loadExtTable(a.reader, a.base+int64(a.header.HetTablePos),
				size, hetSignature, KeyHashTable)
			if err != nil {
				a.warnings = append(a.warnings, fmt.Sprintf("HET table 不可用：%v", err))
			} else {
				if hetTable, err = parseHetTable(buf); err != nil {
					a.warnings = append(a.warnings, fmt.Sprintf("HET table 不可用：%v", err))
					hetTable = nil
				} else {
					a.verifyStoredMd5("HET table", a.base+int64(a.header.HetTablePos),
						size, a.header.MD5HetTable)
				}
			}
		}
		if a.header.BetTablePos != 0 {
			size := a.betStoredSize(archiveEnd)
			buf, err := loadExtTable(a.reader, a.base+int64(a.header.BetTablePos),
				size, betSignature, KeyBlockTable)
			if err != nil {
				a.warnings = append(a.warnings, fmt.Sprintf("BET table 不可用：%v", err))
			} else {
				if betTable, err = parseBetTable(buf); err != nil {
					a.warnings = append(a.warnings, fmt.Sprintf("BET table 不可用：%v", err))
					betTable = nil
				} else {
					a.verifyStoredMd5("BET table", a.base+int64(a.header.BetTablePos),
						size, a.header.MD5BetTable)
				}
			}
		}
	}

	// 可用性检查：至少一个哈希索引 + 一个文件表。
	hasIndex := (classicTables != nil && len(classicTables.HashTable) > 0) || hetTable != nil
	hasFiles := (classicTables != nil && len(classicTables.BlockTable) > 0) || betTable != nil
	if !hasIndex {
		return nil, fmt.Errorf("mpq: 没有可用的哈希索引（经典 hash 表或 HET 表）")
	}
	if !hasFiles {
		return nil, fmt.Errorf("mpq: 没有可用的文件表（经典 block 表或 BET 表）")
	}
	if (classicTables == nil || len(classicTables.HashTable) == 0) && hetTable != nil && betTable == nil {
		return nil, fmt.Errorf("mpq: 存在 HET 表但缺少 BET 表")
	}

	sectorSize := a.header.SectorSize()
	if sectorSize == 0 {
		sectorSize = 512
		a.warnings = append(a.warnings, "无效的 sector 指数，按 512 字节 sector 处理")
	}
	return &loadedTables{classic: classicTables, het: hetTable, bet: betTable, sectorSize: sectorSize}, nil
}

// clampInt 将 u64 长度收敛到 int（与 Java 的 Integer.MAX_VALUE 截断一致）。
func clampInt(v uint64) int {
	if v > uint64(maxInt) {
		return maxInt
	}
	return int(v)
}

// storedTableSizes 返回经典表在磁盘上的大小（-1 = 未知）：
// v1 用未压缩大小，v2/v3 由表间隙推导，v4 直接存储。
func (a *Archive) storedTableSizes(archiveEnd int64) [2]int64 {
	hashSize := uint64(a.header.HashTableSize) * 16
	blockSize := uint64(a.header.BlockTableSize) * 16

	switch a.version {
	case V1:
		return [2]int64{-1, -1}
	case V4:
		var hashStored, blockStored int64 = -1, -1
		if a.header.HashTableSize64 != 0 && a.header.HashTableSize64 <= hashSize {
			hashStored = int64(a.header.HashTableSize64)
		}
		if a.header.BlockTableSize64 != 0 && a.header.BlockTableSize64 <= blockSize {
			blockStored = int64(a.header.BlockTableSize64)
		}
		return [2]int64{hashStored, blockStored}
	default: // V2, V3
		hashPos := a.base + int64(a.header.HashTablePos)
		blockPos := a.base + int64(a.header.BlockTablePos)
		hiPos := archiveEnd
		if a.header.HiBlockTablePos != 0 {
			hiPos = a.base + int64(a.header.HiBlockTablePos)
		}

		hashStored := int64(-1)
		if a.header.HashTablePos != 0 && blockPos > hashPos {
			hashStored = min(blockPos-hashPos, int64(hashSize))
		}
		blockStored := int64(-1)
		if a.header.BlockTablePos != 0 && hiPos > blockPos {
			blockStored = min(hiPos-blockPos, int64(blockSize))
		}
		return [2]int64{hashStored, blockStored}
	}
}

func (a *Archive) hetStoredSize(archiveEnd int64) int64 {
	if a.version == V4 {
		return int64(a.header.HetTableSize64)
	}
	upper := a.lowerTableBoundary(archiveEnd)
	return max(0, upper-int64(a.header.HetTablePos))
}

func (a *Archive) betStoredSize(archiveEnd int64) int64 {
	if a.version == V4 {
		return int64(a.header.BetTableSize64)
	}
	// BET 紧贴下一个边界（hash table 或归档结尾）之下。
	upper := int64(a.header.HashTablePos)
	if upper == 0 {
		upper = a.lowerTableBoundary(archiveEnd)
	}
	return max(0, upper-int64(a.header.BetTablePos))
}

// lowerTableBoundary 返回各上界（hash / block / hi-block / 归档结尾）中的最小值。
func (a *Archive) lowerTableBoundary(archiveEnd int64) int64 {
	boundary := archiveEnd
	if a.header.HashTablePos != 0 {
		boundary = min(boundary, int64(a.header.HashTablePos))
	}
	if a.header.BlockTablePos != 0 {
		boundary = min(boundary, int64(a.header.BlockTablePos))
	}
	if a.header.HiBlockTablePos != 0 {
		boundary = min(boundary, int64(a.header.HiBlockTablePos))
	}
	return boundary
}

// verifyStoredMd5 针对存储后的字节校验 v4 表 MD5；不匹配仅告警。
func (a *Archive) verifyStoredMd5(what string, offset, stored int64, expected [16]byte) {
	if a.version != V4 || !md5Present(expected) || stored == 0 {
		return
	}
	if offset >= a.fileLen || a.fileLen-offset < stored {
		return
	}
	raw := a.reader.ReadAtOrEof(offset, clampInt(uint64(stored)))
	if md5Of(raw) != expected {
		a.warnings = append(a.warnings, what+": v4 表 MD5 不匹配")
	}
}

// loadListfile 读取 (listfile)（可选）。
func (a *Archive) loadListfile() {
	if a.HasFile("(listfile)") {
		data, err := a.ReadFile("(listfile)")
		if err != nil {
			a.warnings = append(a.warnings, fmt.Sprintf("(listfile) 存在但不可读：%v", err))
		} else {
			seen := make(map[string]bool)
			for _, part := range strings.FieldsFunc(string(data), func(r rune) bool {
				return r == ';' || r == '\r' || r == '\n'
			}) {
				s := strings.TrimSpace(part)
				if s != "" && !seen[s] {
					seen[s] = true
					a.names = append(a.names, s)
				}
			}
		}
	}
	if len(a.names) == 0 {
		a.warnings = append(a.warnings, "缺少 (listfile)，文件名未知")
	}
}
