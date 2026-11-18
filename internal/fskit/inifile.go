package fskit

import (
	"bytes"
	"os"
	"strings"
)

// section/item 复刻官方 inifile.cpp 的 section_t/item_t：offset 相对 buffer，
// 名称串（含 trim 后的 key/value）都活在同一块 buffer 里（Build 原地置 NUL）。
type iniSection struct {
	nameOff   uint32
	nameHash  uint32
	firstItem uint32
	itemCount uint32
}

type iniItem struct {
	keyOff   uint32
	keyHash  uint32
	valueOff uint32
}

// IniFile 复刻官方 CIniFile（utils/inifile.cpp）。
type IniFile struct {
	fileName string
	buffer   []byte // 整份文本（Build 后所有字符串的原地存储，尾部恒有 NUL）
	sections []iniSection
	items    []iniItem
}

// NewIniFile 官方 CIniFile::NewInstance(name) = new CIniFile(name)。
func NewIniFile(name string) *IniFile {
	return &IniFile{fileName: name}
}

func (f *IniFile) SetFileName(name string) { f.fileName = name }
func (f *IniFile) GetFileName() string     { return f.fileName }

// GetMemoryUsage 官方返回 buffer 尺寸+容量口径；Go 侧对齐 len(buffer)。
func (f *IniFile) GetMemoryUsage() uint32 { return uint32(len(f.buffer)) }

// cstr 取 buffer 内 offset 处的 C 串（Build/AddToBuffer 维护 NUL 终结）。
func (f *IniFile) cstr(off uint32) string {
	if int(off) >= len(f.buffer) {
		return ""
	}
	end := bytes.IndexByte(f.buffer[off:], 0)
	if end < 0 {
		return string(f.buffer[off:])
	}
	return string(f.buffer[off : int(off)+end])
}

// LoadFromFile 官方 0x10042880：整读 + 尾部哨兵 "\n\0" + Build。
// 短读/打不开 → false。
func (f *IniFile) LoadFromFile() bool {
	data, err := os.ReadFile(f.fileName)
	if err != nil {
		return false
	}
	return f.LoadFromBytes(data)
}

// LoadFromBytes 对齐 LoadFromFile 的内存路径（哨兵等价）。
func (f *IniFile) LoadFromBytes(data []byte) bool {
	buf := make([]byte, len(data)+2)
	copy(buf, data)
	buf[len(data)] = '\n'
	buf[len(data)+1] = 0
	return f.build(buf)
}

// LoadFromString 官方 0x100430a0（Build 直调，同样补哨兵保持行为一致）。
func (f *IniFile) LoadFromString(s string) bool {
	return f.LoadFromBytes([]byte(s))
}

// build 官方 0x10043200 Build 的逐分支直译。入参 buffer 末两位为 "\n\0" 哨兵
// （LoadFromFile 先补好；官方把哨兵补在 fread 之后，行为等价）。
func (f *IniFile) build(buf []byte) bool {
	f.buffer = buf // cstr 依赖 buffer 已就位（官方 Build 全程操作同一 buffer）
	f.sections = f.sections[:0]
	f.items = f.items[:0]

	var curSection *iniSection
	lineStart := 0
	tokenLen := 0
	inBracket := false

	for pos := 0; buf[pos] != 0; pos++ {
		c := buf[pos]
		switch {
		case c == '\r' || c == '\n':
			buf[pos] = 0
			inBracket = false
			if tokenLen == 0 {
				continue
			}
			if buf[lineStart] == ';' { // ';' 注释行
				tokenLen = 0
				continue
			}
			if tokenLen >= 2 && buf[lineStart] == '/' && buf[lineStart+1] == '/' { // '//' 注释行
				tokenLen = 0
				continue
			}
			if curSection == nil {
				// 隐式空名节：nameOff 指向末尾 NUL（官方 len-1 处的 0 字节）
				f.sections = append(f.sections, iniSection{
					nameOff:   uint32(len(buf) - 1),
					nameHash:  HashValue(""),
					firstItem: uint32(len(f.items)),
				})
				curSection = &f.sections[len(f.sections)-1]
			}
			// 整行 rtrim（尾部 ' '/'\t' 逐位置 NUL）
			for end := lineStart + tokenLen - 1; end >= lineStart && (buf[end] == ' ' || buf[end] == '\t'); end-- {
				buf[end] = 0
			}
			// 行内找 '='（行已被 NUL 截断，等价 strchr(lineStart,'=')）
			line := buf[lineStart:] // 到 NUL 为止
			nul := bytes.IndexByte(line, 0)
			eq := bytes.IndexByte(line[:nul], '=')
			f.items = append(f.items, iniItem{})
			it := &f.items[len(f.items)-1]
			if eq < 0 {
				it.keyOff = uint32(lineStart)
				it.keyHash = HashValue(f.cstr(uint32(lineStart)))
				it.valueOff = uint32(len(buf) - 1) // ""：末尾 NUL
			} else {
				eqAbs := lineStart + eq
				buf[eqAbs] = 0
				// key rtrim（'=' 之前）
				for k := eqAbs - 1; k >= lineStart && (buf[k] == ' ' || buf[k] == '\t'); k-- {
					buf[k] = 0
				}
				// value ltrim（'=' 之后，逐位置 NUL 跳过；无 rtrim）
				v := eqAbs + 1
				for buf[v] == ' ' || buf[v] == '\t' {
					buf[v] = 0
					v++
				}
				it.keyOff = uint32(lineStart)
				it.keyHash = HashValue(f.cstr(uint32(lineStart)))
				it.valueOff = uint32(v)
			}
			curSection.itemCount++
			tokenLen = 0
		case c == '[' && tokenLen == 0:
			inBracket = true
			buf[pos] = 0
			lineStart = pos + 1
		case c == ']' && inBracket:
			buf[pos] = 0
			f.sections = append(f.sections, iniSection{
				nameOff:   uint32(lineStart),
				nameHash:  HashValue(f.cstr(uint32(lineStart))),
				firstItem: uint32(len(f.items)),
			})
			curSection = &f.sections[len(f.sections)-1]
			tokenLen = 0
			inBracket = false
		case (c == ' ' || c == '\t') && tokenLen == 0:
			// 行首空白跳过（lineStart 不前移）
		default:
			if tokenLen == 0 {
				lineStart = pos
			}
			tokenLen++
		}
	}
	f.buffer = buf
	return true
}

// SaveToFile 官方 0x10043850：每节 "[%s]\r\n" + "k=v\r\n"… + 空行 "\r\n"。
func (f *IniFile) SaveToFile() bool {
	var b strings.Builder
	for _, sec := range f.sections {
		b.WriteString("[" + f.cstr(sec.nameOff) + "]\r\n")
		for i := uint32(0); i < sec.itemCount; i++ {
			it := f.items[sec.firstItem+i]
			b.WriteString(f.cstr(it.keyOff) + "=" + f.cstr(it.valueOff) + "\r\n")
		}
		b.WriteString("\r\n")
	}
	return os.WriteFile(f.fileName, []byte(b.String()), 0644) == nil
}

// SaveToString 与 SaveToFile 同构（返回序列化文本，便于测试/回放）。
func (f *IniFile) SaveToString() string {
	var b strings.Builder
	for _, sec := range f.sections {
		b.WriteString("[" + f.cstr(sec.nameOff) + "]\r\n")
		for i := uint32(0); i < sec.itemCount; i++ {
			it := f.items[sec.firstItem+i]
			b.WriteString(f.cstr(it.keyOff) + "=" + f.cstr(it.valueOff) + "\r\n")
		}
		b.WriteString("\r\n")
	}
	return b.String()
}

// getSection 官方 0x100439f0：hash 预筛 + stricmp 确认（大小写不敏感）。
func (f *IniFile) getSection(name string) *iniSection {
	h := HashValue(name)
	for i := range f.sections {
		if f.sections[i].nameHash == h && equalFoldASCII(f.cstr(f.sections[i].nameOff), name) {
			return &f.sections[i]
		}
	}
	return nil
}

func (f *IniFile) findSectionIndex(name string) (int, bool) {
	h := HashValue(name)
	for i := range f.sections {
		if f.sections[i].nameHash == h && equalFoldASCII(f.cstr(f.sections[i].nameOff), name) {
			return i, true
		}
	}
	return 0, false
}

// getItem 官方 0x10043b10：节内 item 查找（hash + stricmp）。
func (f *IniFile) getItem(sec *iniSection, key string) (int, bool) {
	h := HashValue(key)
	for i := sec.firstItem; i < sec.firstItem+sec.itemCount; i++ {
		if f.items[i].keyHash == h && equalFoldASCII(f.cstr(f.items[i].keyOff), key) {
			return int(i), true
		}
	}
	return 0, false
}

// AddToBuffer 官方 0x10044c00：在旧内容（含旧末尾 NUL）之后追加 s+"\0"，
// 返回**追加前的完整长度**（= 新串 offset）。旧串原样保留（Write 改值
// 不回收旧串 → 孤儿串语义）。
func (f *IniFile) addToBuffer(s string) uint32 {
	off := uint32(len(f.buffer))
	f.buffer = append(f.buffer, s...)
	f.buffer = append(f.buffer, 0)
	return off
}

// GetData 官方 0x10044850。miss 返回 ("", false)。
func (f *IniFile) GetData(section, key string) (string, bool) {
	sec := f.getSection(section)
	if sec == nil {
		return "", false
	}
	if idx, ok := f.getItem(sec, key); ok {
		return f.cstr(f.items[idx].valueOff), true
	}
	return "", false
}

// ReadInteger 官方 0x10044940：atoi(GetData)，miss → def。
func (f *IniFile) ReadInteger(section, key string, def int) int {
	if v, ok := f.GetData(section, key); ok {
		return cAtoi(v)
	}
	return def
}

// ReadFloat 官方 0x10044b70：atof(GetData)，miss → def。
func (f *IniFile) ReadFloat(section, key string, def float64) float64 {
	if v, ok := f.GetData(section, key); ok {
		return cAtof(v)
	}
	return def
}

// ReadString 官方 0x10044ad0（静态串版）：miss → def。
func (f *IniFile) ReadString(section, key, def string) string {
	if v, ok := f.GetData(section, key); ok {
		return v
	}
	return def
}

// ReadStringBuf 官方 0x100449c0（带 maxlen 截断）：n=strlen(v)，n>=maxlen → n=maxlen-1。
// miss → strcpy(buf,def)。返回 dst。
func (f *IniFile) ReadStringBuf(section, key string, dst []byte, def string) []byte {
	v, ok := f.GetData(section, key)
	src := def
	if ok {
		src = v
	}
	n := len(src)
	if n >= len(dst) {
		n = len(dst) - 1
	}
	copy(dst[:n], src[:n])
	dst[n] = 0
	return dst[:n]
}

// setData 官方 0x10044d60 SetData：WriteInteger/WriteString/WriteFloat 底层。
// miss 节 → 建节；miss key → 插到本节末并修正后续节 firstItem；
// 命中且值不同 → append 新值（旧串成孤儿）；同值 → 不动。
func (f *IniFile) setData(section, key, value string) bool {
	si, ok := f.findSectionIndex(section)
	if !ok {
		off := f.addToBuffer(section)
		f.sections = append(f.sections, iniSection{
			nameOff:   off,
			nameHash:  HashValue(section),
			firstItem: uint32(len(f.items)),
		})
		si = len(f.sections) - 1
	}
	sec := &f.sections[si]
	if idx, ok := f.getItem(sec, key); ok {
		if f.cstr(f.items[idx].valueOff) != value {
			f.items[idx].valueOff = f.addToBuffer(value)
		}
		return true
	}
	koff := f.addToBuffer(key)
	voff := uint32(len(f.buffer) - 1) // value 先置 ""，下一行覆盖
	insertAt := int(sec.firstItem + sec.itemCount)
	f.items = append(f.items, iniItem{})
	copy(f.items[insertAt+1:], f.items[insertAt:len(f.items)-1])
	f.items[insertAt] = iniItem{keyOff: koff, keyHash: HashValue(key), valueOff: voff}
	// 值写入（等价官方 AddData 里的 AddToBuffer(data)）
	f.items[insertAt].valueOff = f.addToBuffer(value)
	sec.itemCount++
	for j := si + 1; j < len(f.sections); j++ {
		f.sections[j].firstItem++
	}
	return true
}

// addData 官方 0x10045310 AddData：AddInteger/AddString/AddFloat 底层。
// 不查重，必 append（同 key 可重复）。
func (f *IniFile) addData(section, key, value string) bool {
	si, ok := f.findSectionIndex(section)
	if !ok {
		off := f.addToBuffer(section)
		f.sections = append(f.sections, iniSection{
			nameOff:   off,
			nameHash:  HashValue(section),
			firstItem: uint32(len(f.items)),
		})
		si = len(f.sections) - 1
	}
	sec := &f.sections[si]
	koff := f.addToBuffer(key)
	voff := f.addToBuffer(value)
	insertAt := int(sec.firstItem + sec.itemCount)
	f.items = append(f.items, iniItem{})
	copy(f.items[insertAt+1:], f.items[insertAt:len(f.items)-1])
	f.items[insertAt] = iniItem{keyOff: koff, keyHash: HashValue(key), valueOff: voff}
	sec.itemCount++
	for j := si + 1; j < len(f.sections); j++ {
		f.sections[j].firstItem++
	}
	return true
}

func (f *IniFile) WriteInteger(section, key string, v int) bool {
	return f.setData(section, key, itoa(v))
}

func (f *IniFile) WriteString(section, key, value string) bool {
	return f.setData(section, key, value)
}

func (f *IniFile) WriteFloat(section, key string, v float64) bool {
	return f.setData(section, key, formatG(v))
}

func (f *IniFile) AddInteger(section, key string, v int) bool {
	return f.addData(section, key, itoa(v))
}

func (f *IniFile) AddString(section, key, value string) bool {
	return f.addData(section, key, value)
}

func (f *IniFile) AddFloat(section, key string, v float64) bool {
	return f.addData(section, key, formatG(v))
}

// AddSection 官方 0x10043c70：查重后 append。
func (f *IniFile) AddSection(name string) bool {
	if f.getSection(name) != nil {
		return false
	}
	off := f.addToBuffer(name)
	f.sections = append(f.sections, iniSection{
		nameOff:   off,
		nameHash:  HashValue(name),
		firstItem: uint32(len(f.items)),
	})
	return true
}

func (f *IniFile) FindSection(name string) bool {
	return f.getSection(name) != nil
}

// DeleteSection 官方 0x10043f50：删节 + 节内 items + 后续节 firstItem 修正。
func (f *IniFile) DeleteSection(name string) bool {
	si, ok := f.findSectionIndex(name)
	if !ok {
		return false
	}
	sec := f.sections[si]
	base := int(sec.firstItem)
	n := int(sec.itemCount)
	f.items = append(f.items[:base], f.items[base+n:]...)
	f.sections = append(f.sections[:si], f.sections[si+1:]...)
	for j := si; j < len(f.sections); j++ {
		f.sections[j].firstItem -= uint32(n)
	}
	return true
}

func (f *IniFile) GetSectionCount() int { return len(f.sections) }

// GetSectionList 官方 0x10044140：节名逐个 AddString。
func (f *IniFile) GetSectionList() []string {
	out := make([]string, 0, len(f.sections))
	for _, sec := range f.sections {
		out = append(out, f.cstr(sec.nameOff))
	}
	return out
}

func (f *IniFile) FindItem(section, key string) bool {
	sec := f.getSection(section)
	if sec == nil {
		return false
	}
	_, ok := f.getItem(sec, key)
	return ok
}

// DeleteItem 官方 0x10044310：删 item + 本节计数-- + 后续节 firstItem 修正。
func (f *IniFile) DeleteItem(section, key string) bool {
	si, ok := f.findSectionIndex(section)
	if !ok {
		return false
	}
	sec := &f.sections[si]
	idx, ok := f.getItem(sec, key)
	if !ok {
		return false
	}
	f.items = append(f.items[:idx], f.items[idx+1:]...)
	sec.itemCount--
	for j := si; j < len(f.sections); j++ {
		if int(f.sections[j].firstItem) > idx {
			f.sections[j].firstItem--
		}
	}
	return true
}

// GetItemCount 官方 0x10044570：节 miss → 0。
func (f *IniFile) GetItemCount(section string) int {
	sec := f.getSection(section)
	if sec == nil {
		return 0
	}
	return int(sec.itemCount)
}

// GetItemList 官方 0x100445e0：节内 key 列表。
func (f *IniFile) GetItemList(section string) []string {
	sec := f.getSection(section)
	if sec == nil {
		return nil
	}
	out := make([]string, 0, sec.itemCount)
	for i := uint32(0); i < sec.itemCount; i++ {
		out = append(out, f.cstr(f.items[sec.firstItem+i].keyOff))
	}
	return out
}

// GetItemValueList 官方 0x10044700：节内 value 列表。
func (f *IniFile) GetItemValueList(section string) []string {
	sec := f.getSection(section)
	if sec == nil {
		return nil
	}
	out := make([]string, 0, sec.itemCount)
	for i := uint32(0); i < sec.itemCount; i++ {
		out = append(out, f.cstr(f.items[sec.firstItem+i].valueOff))
	}
	return out
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	var b [12]byte
	i := len(b)
	u := int64(v)
	if neg {
		u = -u
	}
	for u > 0 {
		i--
		b[i] = byte('0' + u%10)
		u /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
