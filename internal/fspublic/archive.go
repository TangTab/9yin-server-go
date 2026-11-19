package fspublic

import (
	"encoding/binary"
	"math"
	"fmt"
	"unicode/utf16"
)

// 本文件复刻 CStoreArchive/CLoadArchive（d:\flexis\client\utils\
// {storearchive,loadarchive}.h，反汇编定案 notes-datalayer.md）：
//
//   - 全定长小端，无 varint：Int=4B、Int64=8B、Float=4B IEEE754、
//     Double=8B、Byte=1B、Object=8B（PERSISTID 原样）
//   - WriteString：int32 bytelen + 字节（**含 NUL 一起写**，len=len(s)+1），
//     原始 GBK 字节零转换
//   - WriteWideStr：int32 bytelen + UTF-16LE，bytelen=(码元数+1)*2
//   - 容量上限 16MiB（官方 "Out range" throw）
//   - ReadString 校验末字节 NUL（"String no end symbol"）；官方
//     ReadWideStr 的 NUL 校验按码元语义（原生 bug），Go 侧统一字节语义

const archiveMaxSize = 16 * 1024 * 1024 // 官方 ExpandBuffer 上限

// StoreArchive 写侧归档（对应 CStoreArchive）。
type StoreArchive struct {
	buf []byte
}

// NewStoreArchive capacity 预留（官方 ctor 语义）。
func NewStoreArchive(capacity int) *StoreArchive {
	if capacity < 0 {
		capacity = 0
	}
	return &StoreArchive{buf: make([]byte, 0, capacity)}
}

// Bytes 返回已写内容（GetData/GetLength 等价）。
func (a *StoreArchive) Bytes() []byte { return a.buf }

// Len 已写字节数（GetLength）。
func (a *StoreArchive) Len() int { return len(a.buf) }

// Seek 回退/前移写偏移（官方只置偏移；回填占位用）。
func (a *StoreArchive) Seek(pos int) error {
	if pos < 0 || pos > len(a.buf) {
		return fmt.Errorf("Out range")
	}
	a.buf = a.buf[:pos]
	return nil
}

// PatchInt 就地回填 int32（官方 Seek 回写语义；PUB4 blob 计数占位回填）。
func (a *StoreArchive) PatchInt(pos int, v int32) error {
	if pos < 0 || pos+4 > len(a.buf) {
		return fmt.Errorf("Out range")
	}
	binary.LittleEndian.PutUint32(a.buf[pos:], uint32(v))
	return nil
}

func (a *StoreArchive) ensure(n int) error {
	if len(a.buf)+n > archiveMaxSize {
		return fmt.Errorf("Out range")
	}
	return nil
}

// WriteInt 官方 4B 定长小端。
func (a *StoreArchive) WriteInt(v int32) error {
	if err := a.ensure(4); err != nil {
		return err
	}
	a.buf = binary.LittleEndian.AppendUint32(a.buf, uint32(v))
	return nil
}

// WriteInt64 官方 8B。
func (a *StoreArchive) WriteInt64(v int64) error {
	if err := a.ensure(8); err != nil {
		return err
	}
	a.buf = binary.LittleEndian.AppendUint64(a.buf, uint64(v))
	return nil
}

// WriteFloat 官方 4B IEEE754。
func (a *StoreArchive) WriteFloat(v float32) error {
	if err := a.ensure(4); err != nil {
		return err
	}
	a.buf = binary.LittleEndian.AppendUint32(a.buf, math.Float32bits(v))
	return nil
}

// WriteDouble 官方 8B。
func (a *StoreArchive) WriteDouble(v float64) error {
	if err := a.ensure(8); err != nil {
		return err
	}
	a.buf = binary.LittleEndian.AppendUint64(a.buf, math.Float64bits(v))
	return nil
}

// WriteByte 官方 1B。
func (a *StoreArchive) WriteByte(b byte) error {
	if err := a.ensure(1); err != nil {
		return err
	}
	a.buf = append(a.buf, b)
	return nil
}

// WriteObject 官方 8B PERSISTID 原样。
func (a *StoreArchive) WriteObject(lo, hi uint32) error {
	if err := a.ensure(8); err != nil {
		return err
	}
	a.buf = binary.LittleEndian.AppendUint32(a.buf, lo)
	a.buf = binary.LittleEndian.AppendUint32(a.buf, hi)
	return nil
}

// WriteString 官方：int32 bytelen（含 NUL）+ 原始字节。
// s 为已编码字节（GBK 由调用方负责；此处只做传输语义）。
func (a *StoreArchive) WriteString(s string) error {
	n := len(s) + 1
	if err := a.ensure(4 + n); err != nil {
		return err
	}
	a.buf = binary.LittleEndian.AppendUint32(a.buf, uint32(n))
	a.buf = append(a.buf, s...)
	a.buf = append(a.buf, 0)
	return nil
}

// WriteWideStr 官方：int32 bytelen（(码元+1)*2）+ UTF-16LE + 终结符。
func (a *StoreArchive) WriteWideStr(s string) error {
	units := utf16.Encode([]rune(s))
	n := (len(units) + 1) * 2
	if err := a.ensure(4 + n); err != nil {
		return err
	}
	a.buf = binary.LittleEndian.AppendUint32(a.buf, uint32(n))
	for _, u := range units {
		a.buf = binary.LittleEndian.AppendUint16(a.buf, u)
	}
	a.buf = binary.LittleEndian.AppendUint16(a.buf, 0)
	return nil
}

// LoadArchive 读侧归档（对应 CLoadArchive；越界="Out range"）。
type LoadArchive struct {
	buf []byte
	off int
}

// NewLoadArchive 从字节切片构建。
func NewLoadArchive(b []byte) *LoadArchive { return &LoadArchive{buf: b} }

// Offset 当前读偏移。
func (a *LoadArchive) Offset() int { return a.off }

// Seek 置偏移（LoadSpaces 链式取下一条用）。
func (a *LoadArchive) Seek(pos int) error {
	if pos < 0 || pos > len(a.buf) {
		return fmt.Errorf("Out range")
	}
	a.off = pos
	return nil
}

// Rest 剩余字节。
func (a *LoadArchive) Rest() []byte { return a.buf[a.off:] }

func (a *LoadArchive) take(n int) ([]byte, error) {
	if n < 0 || a.off+n > len(a.buf) {
		return nil, fmt.Errorf("Out range")
	}
	b := a.buf[a.off : a.off+n]
	a.off += n
	return b, nil
}

// ReadInt 4B。
func (a *LoadArchive) ReadInt() (int32, error) {
	b, err := a.take(4)
	if err != nil {
		return 0, err
	}
	return int32(binary.LittleEndian.Uint32(b)), nil
}

// ReadInt64 8B。
func (a *LoadArchive) ReadInt64() (int64, error) {
	b, err := a.take(8)
	if err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(b)), nil
}

// ReadFloat 4B。
func (a *LoadArchive) ReadFloat() (float32, error) {
	b, err := a.take(4)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(b)), nil
}

// ReadDouble 8B。
func (a *LoadArchive) ReadDouble() (float64, error) {
	b, err := a.take(8)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(b)), nil
}

// ReadByte 1B。
func (a *LoadArchive) ReadByte() (byte, error) {
	b, err := a.take(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

// ReadObject 8B PERSISTID。
func (a *LoadArchive) ReadObject() (lo, hi uint32, err error) {
	b, err := a.take(8)
	if err != nil {
		return 0, 0, err
	}
	return binary.LittleEndian.Uint32(b), binary.LittleEndian.Uint32(b[4:]), nil
}

// ReadString 官方：int32 bytelen（含 NUL）→ 校验末字节 0 → 返回去 NUL 字节串。
func (a *LoadArchive) ReadString() (string, error) {
	n, err := a.ReadInt()
	if err != nil {
		return "", err
	}
	if n < 1 {
		return "", fmt.Errorf("String no end symbol")
	}
	b, err := a.take(int(n))
	if err != nil {
		return "", err
	}
	if b[n-1] != 0 {
		return "", fmt.Errorf("String no end symbol")
	}
	return string(b[:n-1]), nil
}

// ReadWideStr 官方：int32 bytelen → UTF-16LE → 校验末码元 0
//（官方校验位原生错位，Go 侧取字节语义正确位置）。
func (a *LoadArchive) ReadWideStr() (string, error) {
	n, err := a.ReadInt()
	if err != nil {
		return "", err
	}
	if n < 2 || n%2 != 0 {
		return "", fmt.Errorf("Wide string no end symbol")
	}
	b, err := a.take(int(n))
	if err != nil {
		return "", err
	}
	units := make([]uint16, n/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	if units[len(units)-1] != 0 {
		return "", fmt.Errorf("Wide string no end symbol")
	}
	return string(utf16.Decode(units[:len(units)-1])), nil
}
