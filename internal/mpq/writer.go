package mpq

import (
	"encoding/binary"
	"fmt"
	"math/bits"
	"strings"
)

// MPQ 归档写入：创建空归档、导入（新增/替换）与删除文件、版本转换。
//
// 策略与参考实现一致：保留文件的已存储字节原样复制（压缩方式、
// sector CRC、加密状态不变），只有容器被重写；新文件以 zlib 编码
// （小文件单块 single-unit，永不加密）；(listfile) 重建，
// (attributes)/(signature) 丢弃；v4 头会重算全部 MD5 校验和；
// 归档前后的字节（MPQ\x1B 包装、内嵌前缀、尾部附加）保留。
// 不支持自定义的密码 AEAD 扩展（遇保护归档明确报错）。

// Addition 是一个待导入的文件（保持传入顺序；同名后到覆盖先到）。
type Addition struct {
	Name string
	Data []byte
}

// Deletion 是一次请求的删除操作。用 DeleteByName / DeleteByKey 构造。
type Deletion struct {
	name  string // ByName 模式的名字
	byKey bool   // true = 按表条目精确删除
	Key   struct {
		Source TableSource
		Index  uint32
	}
}

// DeleteByName 删除所有名字匹配的条目（MPQ 规范化：不分大小写与斜杠方向）。
func DeleteByName(name string) Deletion {
	return Deletion{name: name}
}

// DeleteByKey 精确删除一个表条目（寻址无名条目的唯一途径）。
func DeleteByKey(source TableSource, index uint32) Deletion {
	d := Deletion{byKey: true}
	d.Key.Source = source
	d.Key.Index = index
	return d
}

// CreateOptions 是创建空归档的配置。
type CreateOptions struct {
	Version       FormatVersion
	HashTableSize int
	SectorShift   int
}

// DefaultCreateOptions 返回缺省配置（v1、1024 哈希槽、4096 字节 sector）。
func DefaultCreateOptions() CreateOptions {
	return CreateOptions{Version: V1, HashTableSize: 1024, SectorShift: 3}
}

// CreateOptionsOfVersion 返回指定版本的缺省配置。
func CreateOptionsOfVersion(version FormatVersion) CreateOptions {
	return CreateOptions{Version: version, HashTableSize: 1024, SectorShift: 3}
}

// ModifyOptions 是 modify 的可调选项。
type ModifyOptions struct {
	// ReencryptKeyV2 控制 key-v2 块移动后按新偏移重加密（关闭则退化为明文重编码）。
	ReencryptKeyV2 bool
	// TargetVersion 非 nil 时将容器转换为该头版本。
	TargetVersion *FormatVersion
}

// DefaultModifyOptions 返回缺省选项（重加密、保持版本）。
func DefaultModifyOptions() ModifyOptions { return ModifyOptions{ReencryptKeyV2: true} }

// ModifyReport 描述一次 modify 运行所做的变更。
type ModifyReport struct {
	Added     []string
	Replaced  []string
	Deleted   []string
	Dropped   []string
	ReEncoded []string
}

// ModifyResult 是 modify 运行成功的结果。
type ModifyResult struct {
	Bytes  []byte
	Report ModifyReport
}

// == 创建 ==================================================================

// Create 构建一个新的空 MPQ 归档（内存字节）。
func Create(options CreateOptions) ([]byte, error) {
	if options.SectorShift > 30 {
		return nil, fmt.Errorf("mpq: 无效的扇区指数 %d（应在 0–30）", options.SectorShift)
	}
	hashCount := normalizeHashSize(options.HashTableSize)
	out := emptyArchive(options.Version, hashCount, options.SectorShift)
	// 自检：空容器必须能以所请求的版本打开，且不含任何条目。
	check, err := OpenBytes(out)
	if err != nil {
		return nil, err
	}
	defer check.Close()
	if check.Version() != int(options.Version)+1 || len(check.Entries()) != 0 {
		return nil, fmt.Errorf("mpq: 新建归档自检失败")
	}
	return out, nil
}

// normalizeHashSize 将哈希表大小收敛为合法的 2 的幂（4..=1Mi 项）。
func normalizeHashSize(requested int) int {
	clamped := max(4, min(requested, 1<<20))
	return nextPow2(clamped)
}

// emptyArchive 组装空容器：头部 | 加密的全空闲哈希表 | 加密的单空闲块表。
func emptyArchive(version FormatVersion, hashCount, sectorShift int) []byte {
	free := make([]byte, 16)
	binary.LittleEndian.PutUint32(free[0:], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(free[4:], 0xFFFFFFFF)
	// locale / platform / flags 保持 0
	binary.LittleEndian.PutUint32(free[12:], hashEntryFree)

	hashBytes := make([]byte, hashCount*16)
	for i := 0; i < hashCount; i++ {
		copy(hashBytes[i*16:], free)
	}
	EncryptBytes(hashBytes, KeyHashTable)

	// 一个空闲块项（全零）：占位用的未使用块。
	blockBytes := make([]byte, 16)
	EncryptBytes(blockBytes, KeyBlockTable)

	hashPos := version.HeaderSize()
	blockPos := hashPos + len(hashBytes)
	total := int64(blockPos + len(blockBytes))
	header := writeHeader(version, uint16(sectorShift), uint32(hashPos), uint32(hashCount),
		uint32(blockPos), 1, uint64(total))
	if version == V4 {
		finishV4Header(header, hashBytes, blockBytes)
	}

	out := make([]byte, 0, total)
	out = append(out, header...)
	out = append(out, hashBytes...)
	out = append(out, blockBytes...)
	return out
}

// == 修改 ==================================================================

// Modify 向归档导入和/或删除条目，返回完整的修改后字节（缺省选项）。
func Modify(data []byte, deletions []Deletion, additions []Addition) (ModifyResult, error) {
	return ModifyWithOptions(data, deletions, additions, DefaultModifyOptions())
}

// Convert 将归档容器转换为另一种头版本（v1–v4）。
func Convert(data []byte, target FormatVersion) (ModifyResult, error) {
	return ModifyWithOptions(data, nil, nil, ModifyOptions{ReencryptKeyV2: true, TargetVersion: &target})
}

// ModifyWithOptions 是带显式选项的 modify。
func ModifyWithOptions(data []byte, deletions []Deletion, additions []Addition,
	options ModifyOptions) (ModifyResult, error) {
	archive, err := OpenBytes(data)
	if err != nil {
		return ModifyResult{}, err
	}
	defer archive.Close()
	return modifyInner(archive, data, deletions, additions, options)
}

// == modify 主流程 =========================================================

// hashFields 是经典 hash 条目字段，用于按名字（重新）锚定的块。
type hashFields struct {
	name1, name2 uint32
	locale       uint16
	platform     byte
}

func hashFieldsOf(name string) hashFields {
	return hashFields{
		name1: HashString(name, hashNameA),
		name2: HashString(name, hashNameB),
	}
}

// payloadKind 表示块写入新归档的方式。
type payloadKind int

const (
	payloadRaw          payloadKind = iota // 原样复制存储字节
	payloadFresh                           // 待编码的原始内容
	payloadFreshEncoded                    // 已编码的新存储块
	payloadReEncrypted                     // key-v2 以新偏移重加密
)

type outPayload struct {
	kind  payloadKind
	bytes []byte // Raw / FreshEncoded 的存储字节
	flags uint32 // Raw / FreshEncoded 的块标志
	data  []byte // Fresh / ReEncrypted 的原始内容
	name  string // ReEncrypted 的文件名
}

type outBlock struct {
	payload      outPayload
	fsize        uint32
	name         string // "" = 未知
	hashes       []hashFields
	classicRemap bool
}

type placedBlock struct {
	pos   uint64
	bytes []byte
	flags uint32
}

type srcBlock struct {
	source TableSource
	index  uint32
	offset uint64
	csize  uint32
	fsize  uint32
	flags  uint32
	name   string // "" = 未知
}

func modifyInner(archive *Archive, data []byte, deletions []Deletion, additions []Addition,
	options ModifyOptions) (ModifyResult, error) {
	base := archive.ArchiveOffset()
	sectorSize := int(archive.SectorSize())
	sourceVersion := archive.FormatVersion()

	if archive.IsPasswordProtected() {
		return ModifyResult{}, fmt.Errorf("mpq: 归档启用了 Rust mpq 库的密码 AEAD 保护，" +
			"本模块不支持修改（请先在 Rust 端 unprotect）")
	}

	version := sourceVersion
	if options.TargetVersion != nil {
		version = *options.TargetVersion
	}
	converting := options.TargetVersion != nil && *options.TargetVersion != sourceVersion
	sectorShift := archive.header.BlockSize

	if !converting && len(deletions) == 0 && len(additions) == 0 {
		return ModifyResult{Bytes: data}, nil
	}

	// --- 规范化导入项（后面的重复项覆盖先到的，保持首次位置） ---
	type namedData struct {
		name string
		data []byte
	}
	var adds []namedData
	for _, addition := range additions {
		name, err := SanitizeName(addition.Name)
		if err != nil {
			return ModifyResult{}, err
		}
		existing := -1
		for i := range adds {
			if SameName(adds[i].name, name) {
				existing = i
				break
			}
		}
		if existing >= 0 {
			adds[existing] = namedData{name, addition.Data}
		} else {
			adds = append(adds, namedData{name, addition.Data})
		}
	}

	// --- 快照源表 ---
	nameOf := make(map[uint64]string)
	for _, e := range archive.Entries() {
		if e.Name != "" {
			nameOf[tableKey(e.Source, e.BlockIndex)] = e.Name
		}
	}

	classic := archive.classic
	bet := archive.bet
	useClassicBlocks := classic != nil && len(classic.BlockTable) > 0

	// 经典 hash 条目按指向块分组（仅占用项）—— 匹配无名但可寻址的文件。
	hashByBlock := make(map[uint32][]HashEntry)
	if classic != nil {
		for _, entry := range classic.HashTable {
			if !entry.isFree() {
				hashByBlock[entry.BlockIndex] = append(hashByBlock[entry.BlockIndex], entry)
			}
		}
	}

	// --- 枚举源块 ---
	var src []srcBlock
	if useClassicBlocks {
		for i := range classic.BlockTable {
			block := classic.BlockTable[i]
			if block.Flags&fileExists == 0 {
				continue // 幽灵/空闲条目
			}
			src = append(src, srcBlock{
				source: SourceClassic, index: uint32(i),
				offset: classic.BlockOffset(i), csize: block.CSize,
				fsize: block.FSize, flags: block.Flags,
				name: nameOf[tableKey(SourceClassic, uint32(i))],
			})
		}
	} else if bet != nil {
		for i := 0; i < int(bet.EntryCount); i++ {
			rec := bet.Record(i)
			if rec.Flags&fileExists == 0 {
				continue
			}
			src = append(src, srcBlock{
				source: SourceBET, index: uint32(i),
				offset: rec.FilePos, csize: rec.CmpSize,
				fsize: rec.FileSize, flags: rec.Flags,
				name: nameOf[tableKey(SourceBET, uint32(i))],
			})
		}
	}

	// --- 决定丢弃哪些源块 ---
	droppedKeys := make(map[uint64]bool)
	var report ModifyReport

	addDropped := func(k uint64) bool {
		if droppedKeys[k] {
			return false
		}
		droppedKeys[k] = true
		return true
	}

	for _, del := range deletions {
		if !del.byKey {
			if SameName(del.name, "(password)") {
				return ModifyResult{}, fmt.Errorf("mpq: 不能删除 (password)（密码数据）")
			}
			for i := range src {
				b := &src[i]
				if b.name != "" && SameName(b.name, del.name) {
					if addDropped(tableKey(b.source, b.index)) {
						report.Deleted = append(report.Deleted, b.name)
					}
				}
			}
			// 注：无名条目无法按名匹配，由下方归档查找覆盖。
			if e, ok := archive.FindFile(del.name); ok {
				if addDropped(tableKey(e.Source, e.BlockIndex)) {
					report.Deleted = append(report.Deleted, del.name)
				}
			}
		} else {
			for i := range src {
				b := &src[i]
				if b.source == del.Key.Source && b.index == del.Key.Index {
					name := b.name
					if name == "" {
						name = fmt.Sprintf("<unnamed #%d>", b.index)
					}
					report.Deleted = append(report.Deleted, name)
					droppedKeys[tableKey(b.source, b.index)] = true
				}
			}
		}
	}

	// 由修改器管理或因修改而失效的内部文件。
	for i := range src {
		b := &src[i]
		if b.name == "" {
			continue
		}
		if b.name == "(attributes)" || b.name == "(signature)" {
			report.Dropped = append(report.Dropped, b.name)
			droppedKeys[tableKey(b.source, b.index)] = true
		} else if b.name == "(listfile)" {
			// 总是在下方重建；从不复制。
			droppedKeys[tableKey(b.source, b.index)] = true
		}
	}

	// 导入项替换匹配条目 —— 按名字、按经典 hash 对、或经归档查找。
	for _, add := range adds {
		norm := NormalizeName(add.name)
		pair1 := HashBytes(norm, hashNameA)
		pair2 := HashBytes(norm, hashNameB)
		replacedAny := false
		for i := range src {
			b := &src[i]
			nameMatch := b.name != "" && SameName(b.name, add.name)
			entries := hashByBlock[b.index]
			hashMatch := false
			if b.source == SourceClassic && b.name == "" && entries != nil {
				for _, e := range entries {
					if e.Name1 == pair1 && e.Name2 == pair2 {
						hashMatch = true
						break
					}
				}
			}
			if nameMatch || hashMatch {
				droppedKeys[tableKey(b.source, b.index)] = true
				replacedAny = true
			}
		}
		if found, ok := archive.FindFile(add.name); ok {
			if addDropped(tableKey(found.Source, found.BlockIndex)) {
				replacedAny = true
			}
		}
		if replacedAny {
			report.Replaced = append(report.Replaced, add.name)
		} else {
			report.Added = append(report.Added, add.name)
		}
	}

	// --- 构建输出块 ---
	var blocks []outBlock
	remap := make(map[uint32]int)
	var reencoded []namedData

	for i := range src {
		b := &src[i]
		if droppedKeys[tableKey(b.source, b.index)] {
			continue
		}
		start := base + int64(min(b.offset, uint64(maxInt)))
		end := min(start+int64(b.csize), int64(len(data)))
		if start > end {
			start = end
		}
		raw := data[start:end]

		if b.flags&filePasswordAEAD != 0 {
			return ModifyResult{}, fmt.Errorf("mpq: 源归档包含密码 AEAD 块（%s），"+
				"本模块不支持；请先在 Rust 端 unprotect", b.name)
		}

		var payload outPayload
		if b.flags&fileKeyV2 != 0 && b.name != "" {
			// key-v2 密钥派生自块偏移，而偏移会变化：从解码内容重新编码。
			entry := FileEntry{
				Name: b.name, BlockIndex: b.index, Offset: b.offset,
				CompressedSize: b.csize, Size: b.fsize, Flags: b.flags, Source: b.source,
			}
			content, err := archive.ReadEntry(entry)
			if err == nil {
				report.ReEncoded = append(report.ReEncoded, b.name)
				reencoded = append(reencoded, namedData{b.name, content})
				if options.ReencryptKeyV2 {
					payload = outPayload{kind: payloadReEncrypted, name: b.name, data: content}
				} else {
					payload = outPayload{kind: payloadFresh, data: content}
				}
			} else {
				// 源中本就无法读取；原样复制保持现状。
				payload = outPayload{kind: payloadRaw, bytes: raw, flags: b.flags}
			}
		} else {
			payload = outPayload{kind: payloadRaw, bytes: raw, flags: b.flags}
		}

		var hashes []hashFields
		if b.source == SourceClassic {
			for _, e := range hashByBlock[b.index] {
				hashes = append(hashes, hashFields{e.Name1, e.Name2, e.Locale, e.Platform})
			}
		} else if b.name != "" {
			hashes = append(hashes, hashFieldsOf(b.name))
		}
		if b.source == SourceClassic {
			remap[b.index] = len(blocks)
		}
		blocks = append(blocks, outBlock{
			payload: payload, fsize: b.fsize, name: b.name,
			hashes: hashes, classicRemap: b.source == SourceClassic,
		})
	}

	for _, add := range adds {
		blocks = append(blocks, outBlock{
			payload: outPayload{kind: payloadFresh, data: add.data},
			fsize:   uint32(len(add.data)),
			name:    add.name,
			hashes:  []hashFields{hashFieldsOf(add.name)},
		})
	}

	// 重建的 listfile：除内部文件外的所有保留 / 新增名字。
	var listfileNames []string
	for i := range blocks {
		if blocks[i].name == "" || isForbiddenImport(blocks[i].name) {
			continue
		}
		listfileNames = append(listfileNames, blocks[i].name)
	}
	if len(listfileNames) > 0 {
		content := []byte(strings.Join(listfileNames, "\r\n"))
		blocks = append(blocks, outBlock{
			payload: outPayload{kind: payloadFresh, data: content},
			fsize:   uint32(len(content)),
			name:    "(listfile)",
			hashes:  []hashFields{hashFieldsOf("(listfile)")},
		})
	}

	// 转换不得静默产生孤儿文件（无名且无法重新锚定的 BET 条目）。
	if converting {
		orphaned := 0
		for i := range blocks {
			b := &blocks[i]
			if b.name == "" && len(b.hashes) == 0 && !b.classicRemap {
				orphaned++
			}
		}
		if orphaned > 0 {
			return ModifyResult{}, fmt.Errorf("mpq: 无法转换：源归档使用 HET/BET 表且缺少 (listfile)，"+
				"%d 个文件的名字无法恢复，转为经典表后将无法按名访问（可先解包再以目标版本重建）", orphaned)
		}
	}

	if len(blocks) == 0 {
		// 零文件的源连重建的 listfile 都无法承载；纯转换退化为空目标归档。
		if converting && len(adds) == 0 {
			hashCount := normalizeHashSize(max(int(archive.header.HashTableSize), 4))
			inner := emptyArchive(version, hashCount, int(sectorShift))
			out := make([]byte, 0, int(base)+len(inner))
			out = append(out, data[:base]...)
			out = append(out, inner...)
			oldEnd := base + archive.header.EffectiveArchiveSize(int64(len(data))-base)
			out = appendTail(out, data, oldEnd)
			check, err := OpenBytes(out)
			if err != nil {
				return ModifyResult{}, err
			}
			defer check.Close()
			if check.Version() != int(version)+1 {
				return ModifyResult{}, fmt.Errorf("mpq: 转换校验失败：版本不一致")
			}
			return ModifyResult{Bytes: out, Report: report}, nil
		}
		return ModifyResult{}, fmt.Errorf("mpq: 修改后归档不包含任何文件；至少需要保留一个条目")
	}

	// --- 布局与编码 ---
	placed := make([]placedBlock, 0, len(blocks))

	// Fresh 块的编码不依赖位置，先全部预编码。
	for i := range blocks {
		if blocks[i].payload.kind == payloadFresh {
			enc := encodeFresh(blocks[i].payload.data, sectorSize)
			blocks[i].payload = outPayload{kind: payloadFreshEncoded, bytes: enc.bytes, flags: enc.flags}
		}
	}

	cursor := int64(version.HeaderSize())
	for i := range blocks {
		b := &blocks[i]
		pos := cursor
		var bytes []byte
		var flags uint32
		switch b.payload.kind {
		case payloadRaw:
			bytes, flags = b.payload.bytes, b.payload.flags
		case payloadFreshEncoded:
			bytes, flags = b.payload.bytes, b.payload.flags
		case payloadReEncrypted:
			enc := encodeFreshReencrypted(b.payload.name, b.payload.data, sectorSize, pos)
			bytes, flags = enc.bytes, enc.flags
		default:
			return ModifyResult{}, fmt.Errorf("mpq: 不可达的 payload 状态")
		}
		cursor += int64(len(bytes))
		cursor = (cursor + 3) &^ 3 // dword 对齐
		placed = append(placed, placedBlock{pos: uint64(pos), bytes: bytes, flags: flags})
	}

	// --- hash table ---
	// 名字未知的条目只有在原表槽位布局保留时才能存活。
	slotPreservation := classic != nil && len(classic.HashTable) > 0 && func() bool {
		for i := range classic.HashTable {
			e := &classic.HashTable[i]
			if e.isFree() {
				continue
			}
			for j := range src {
				b := &src[j]
				if b.source == SourceClassic && b.index == e.BlockIndex &&
					!droppedKeys[tableKey(b.source, b.index)] && b.name == "" {
					return true
				}
			}
		}
		return false
	}()

	var hashEntries []HashEntry
	if slotPreservation {
		hashEntries = make([]HashEntry, len(classic.HashTable))
		copy(hashEntries, classic.HashTable)
		for i := range hashEntries {
			e := &hashEntries[i]
			if e.isFree() {
				continue
			}
			if mapped, ok := remap[e.BlockIndex]; ok {
				e.BlockIndex = uint32(mapped)
			} else {
				e.BlockIndex = hashEntryDeleted
			}
		}
	} else {
		inserts := 0
		for i := range blocks {
			inserts += len(blocks[i].hashes)
		}
		hashEntries = newHashEntries(max(4, nextPow2(inserts*2)))
	}
	for i := range blocks {
		b := &blocks[i]
		if slotPreservation && b.classicRemap {
			continue // 已由重映射后的原始槽位覆盖
		}
		if b.name == "" {
			continue
		}
		for _, fields := range b.hashes {
			if !insertIntoHash(hashEntries, b.name, fields, i) {
				return ModifyResult{}, fmt.Errorf("mpq: 原始哈希表已满，" +
					"无法插入新文件（且缺少名单无法重建更大的表）")
			}
		}
	}

	// --- 序列化表 ---
	hashBytes := make([]byte, len(hashEntries)*16)
	for i, e := range hashEntries {
		e.serialize(hashBytes[i*16:])
	}
	EncryptBytes(hashBytes, KeyHashTable)

	blockBytes := make([]byte, len(placed)*16)
	for i := range placed {
		p := &placed[i]
		if p.pos > 0xFFFFFFFF {
			return ModifyResult{}, fmt.Errorf("mpq: 归档超过 4 GB，无法用 32 位偏移写出")
		}
		off := i * 16
		binary.LittleEndian.PutUint32(blockBytes[off:], uint32(p.pos))
		binary.LittleEndian.PutUint32(blockBytes[off+4:], uint32(len(p.bytes)))
		binary.LittleEndian.PutUint32(blockBytes[off+8:], blocks[i].fsize)
		binary.LittleEndian.PutUint32(blockBytes[off+12:], p.flags|fileExists)
	}
	EncryptBytes(blockBytes, KeyBlockTable)

	hashPos := cursor
	cursor += int64(len(hashBytes))
	blockPos := cursor
	cursor += int64(len(blockBytes))
	if cursor > 0xFFFFFFFF {
		return ModifyResult{}, fmt.Errorf("mpq: 归档超过 4 GB，无法写出")
	}

	// --- 组装 ---
	header := writeHeader(version, sectorShift, uint32(hashPos), uint32(len(hashEntries)),
		uint32(blockPos), uint32(len(placed)), uint64(cursor))
	if version == V4 {
		finishV4Header(header, hashBytes, blockBytes)
	}

	out := make([]byte, 0, int(base)+int(cursor))
	out = append(out, data[:base]...)
	out = append(out, header...)
	for i := range placed {
		out = append(out, placed[i].bytes...)
		pad := (4 - len(placed[i].bytes)%4) % 4
		out = append(out, make([]byte, pad)...)
	}
	out = append(out, hashBytes...)
	out = append(out, blockBytes...)

	// 原归档之后的字节（追加的数据）保留。
	oldEnd := base + archive.header.EffectiveArchiveSize(int64(len(data))-base)
	out = appendTail(out, data, oldEnd)

	// --- 自检：结果必须能打开并正确读回 ---
	check, err := OpenBytes(out)
	if err != nil {
		return ModifyResult{}, err
	}
	defer check.Close()
	if options.TargetVersion != nil && check.Version() != int(*options.TargetVersion)+1 {
		return ModifyResult{}, fmt.Errorf("mpq: 转换校验失败：写出的版本与目标不一致")
	}
	for _, add := range adds {
		got, err := check.ReadFile(add.name)
		if err != nil || !equalBytes(got, add.data) {
			return ModifyResult{}, fmt.Errorf("mpq: 校验失败：%s 内容不一致", add.name)
		}
	}
	for _, re := range reencoded {
		got, err := check.ReadFile(re.name)
		if err != nil || !equalBytes(got, re.data) {
			return ModifyResult{}, fmt.Errorf("mpq: 校验失败：%s 重编码后内容不一致", re.name)
		}
	}
	for _, del := range deletions {
		if del.byKey {
			continue
		}
		readded := false
		for _, add := range adds {
			if SameName(add.name, del.name) {
				readded = true
				break
			}
		}
		if !readded && check.HasFile(del.name) {
			return ModifyResult{}, fmt.Errorf("mpq: 校验失败：%s 仍然存在", del.name)
		}
	}
	return ModifyResult{Bytes: out, Report: report}, nil
}

// isForbiddenImport 判断名字是否为导入时不可使用的内部文件名。
func isForbiddenImport(name string) bool {
	for _, f := range SpecialFiles {
		if name == f {
			return true
		}
	}
	return false
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// appendTail 保留原归档之后追加的字节。
func appendTail(out, data []byte, oldEnd int64) []byte {
	if oldEnd < int64(len(data)) {
		return append(out, data[oldEnd:]...)
	}
	return out
}

func newHashEntries(count int) []HashEntry {
	entries := make([]HashEntry, count)
	for i := range entries {
		entries[i] = HashEntry{
			Name1:      0xFFFFFFFF,
			Name2:      0xFFFFFFFF,
			BlockIndex: hashEntryFree,
		}
	}
	return entries
}

// SanitizeName 规范化导入名：/ → \，去掉空 / . / .. 部分。
func SanitizeName(name string) (string, error) {
	if isForbiddenImport(name) {
		return "", fmt.Errorf("mpq: 不允许使用保留名 %s", name)
	}
	normalized := strings.ReplaceAll(name, "/", `\`)
	var cleaned []string
	for _, part := range strings.Split(normalized, `\`) {
		if part != "" && part != "." && part != ".." {
			cleaned = append(cleaned, part)
		}
	}
	if len(cleaned) == 0 {
		return "", fmt.Errorf("mpq: 无效的文件名 %q", name)
	}
	return strings.Join(cleaned, `\`), nil
}

func nextPow2(n int) int {
	if n <= 1 {
		return 1
	}
	return 1 << bits.Len(uint(n-1))
}

// insertIntoHash 开放探测插入；同名对 + locale 的占用槽位被替换；
// 表满返回 false。
func insertIntoHash(entries []HashEntry, name string, fields hashFields, blockIndex int) bool {
	norm := NormalizeName(name)
	start := uint64(HashBytes(norm, hashTableIndex)) % uint64(len(entries))
	for off := uint64(0); off < uint64(len(entries)); off++ {
		slot := int((start + off) % uint64(len(entries)))
		e := entries[slot]
		same := !e.isFree() && e.Name1 == fields.name1 && e.Name2 == fields.name2 &&
			e.Locale == fields.locale
		if same || e.isFree() {
			entries[slot] = HashEntry{
				Name1:      fields.name1,
				Name2:      fields.name2,
				Locale:     fields.locale,
				Platform:   fields.platform,
				BlockIndex: uint32(blockIndex),
			}
			return true
		}
	}
	return false
}

// == 编码 ==================================================================

// encoded 是编码产物：存储字节 + 块标志。
type encoded struct {
	bytes []byte
	flags uint32
}

// encodeFresh 将内容编码为新存储块：有收益时每 sector 用 zlib，
// 小文件单块 single-unit，不可压缩时明文 sector。
func encodeFresh(data []byte, sectorSize int) encoded {
	fsize := len(data)

	if fsize == 0 {
		return encoded{flags: fileSingleUnit}
	}

	if fsize < sectorSize {
		z := zlibCompress(data)
		if len(z)+1 < fsize {
			b := make([]byte, len(z)+1)
			b[0] = mZlib
			copy(b[1:], z)
			return encoded{bytes: b, flags: fileSingleUnit | fileCompress}
		}
		return encoded{bytes: append([]byte(nil), data...), flags: fileSingleUnit}
	}

	nsec := (fsize + sectorSize - 1) / sectorSize
	sectors := make([][]byte, nsec)
	total := 0
	for i := 0; i < nsec; i++ {
		chunk := data[min(i*sectorSize, fsize):min((i+1)*sectorSize, fsize)]
		z := zlibCompress(chunk)
		if len(z)+1 < len(chunk) {
			s := make([]byte, len(z)+1)
			s[0] = mZlib
			copy(s[1:], z)
			sectors[i] = s
		} else {
			sectors[i] = chunk
		}
		total += len(sectors[i])
	}
	if total >= fsize {
		// 没有任何内容可压缩：按固定偏移明文存储。
		return encoded{bytes: append([]byte(nil), data...)}
	}

	tableLen := (nsec + 1) * 4
	block := make([]byte, 0, tableLen+total)
	acc := tableLen
	var head [4]byte
	binary.LittleEndian.PutUint32(head[:], uint32(acc))
	block = append(block, head[:]...)
	for _, s := range sectors {
		acc += len(s)
		binary.LittleEndian.PutUint32(head[:], uint32(acc))
		block = append(block, head[:]...)
	}
	for _, s := range sectors {
		block = append(block, s...)
	}
	return encoded{bytes: block, flags: fileCompress}
}

// encodeFreshReencrypted 是 encodeFresh 再加上以块新偏移派生密钥的
// key-v2 加密。磁盘布局约定：single unit 整体加密；压缩 sector 填充到
// dword 边界（偏移含填充），偏移表用 key-1、每个 sector 用 key+i 加密；
// 不可压缩时按固定偏移原样存储、逐个加密。
func encodeFreshReencrypted(name string, data []byte, sectorSize int, pos int64) encoded {
	const enc = fileEncrypted | fileKeyV2
	fsize := len(data)
	key := FileKey(name, true, uint64(pos), uint32(fsize))

	if fsize == 0 {
		return encoded{flags: fileSingleUnit}
	}

	if fsize < sectorSize {
		z := zlibCompress(data)
		compressed := len(z)+1 < fsize
		var payload []byte
		if compressed {
			payload = make([]byte, len(z)+1)
			payload[0] = mZlib
			copy(payload[1:], z)
		} else {
			payload = append([]byte(nil), data...)
		}
		EncryptBytes(payload, key)
		flags := enc | fileSingleUnit
		if compressed {
			flags |= fileCompress
		}
		return encoded{bytes: payload, flags: flags}
	}

	nsec := (fsize + sectorSize - 1) / sectorSize
	sectors := make([][]byte, nsec)
	compressedAny := false
	for i := 0; i < nsec; i++ {
		chunk := data[min(i*sectorSize, fsize):min((i+1)*sectorSize, fsize)]
		z := zlibCompress(chunk)
		// 填充到 dword 后（z+1+3 = z+4）仍小于原文才算有收益（读取方以长度区分）。
		if len(z)+4 < len(chunk) {
			s := make([]byte, len(z)+1)
			s[0] = mZlib
			copy(s[1:], z)
			sectors[i] = s
			compressedAny = true
		} else {
			sectors[i] = chunk
		}
	}
	if !compressedAny {
		// 固定偏移明文 sector，逐个加密。
		payload := make([]byte, 0, fsize)
		for i := range sectors {
			EncryptBytes(sectors[i], key+uint32(i))
			payload = append(payload, sectors[i]...)
		}
		return encoded{bytes: payload, flags: enc}
	}

	// 压缩布局：全部 sector 填充到 dword 边界（偏移含填充），
	// 偏移表 key-1、每个 sector key+i 加密。
	total := 0
	for i := range sectors {
		if pad := (4 - len(sectors[i])%4) % 4; pad != 0 {
			sectors[i] = append(sectors[i], make([]byte, pad)...)
		}
		total += len(sectors[i])
	}
	tableLen := (nsec + 1) * 4
	block := make([]byte, tableLen+total)
	acc := tableLen
	binary.LittleEndian.PutUint32(block[0:], uint32(acc))
	for i, s := range sectors {
		acc += len(s)
		binary.LittleEndian.PutUint32(block[(i+1)*4:], uint32(acc))
	}
	EncryptBytes(block[:tableLen], key-1)
	out := tableLen
	for i := range sectors {
		EncryptBytes(sectors[i], key+uint32(i))
		copy(block[out:], sectors[i])
		out += len(sectors[i])
	}
	return encoded{bytes: block, flags: enc | fileCompress}
}
