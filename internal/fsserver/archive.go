package fsserver

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf16"
)

// StoreWriter 复刻 CStoreArchive（§6.2）：变长缓冲 + 定长写原语。
// 写串方言：[u32 len][bytes]（窄串原字节 / 宽串 UTF-16LE）。
type StoreWriter struct {
	buf []byte
	pos int
}

// NewStoreWriter …（initial 仅提示容量，Go 侧自动扩容替代 ExpandBuffer）。
func NewStoreWriter(initial int) *StoreWriter {
	if initial <= 0 {
		initial = 0x400
	}
	return &StoreWriter{buf: make([]byte, initial)}
}

func (w *StoreWriter) grow(n int) {
	if end := w.pos + n; end > len(w.buf) {
		nb := make([]byte, end, end*2)
		copy(nb, w.buf)
		w.buf = nb
	}
}

// Seek 定位（构造侧预留 4B 消息头空间的官方惯例）。
func (w *StoreWriter) Seek(pos int) { w.pos = pos }

// Pos …
func (w *StoreWriter) Pos() int { return w.pos }

// Len 返回有效长度（GetLength）。
func (w *StoreWriter) Len() int { return w.pos }

// Bytes 返回有效切片（GetData）。
func (w *StoreWriter) Bytes() []byte { return w.buf[:w.pos] }

// PutByte 写单字节（官方 WriteByte；Go 惯例签名让位于 io.Writer 家族）。
func (w *StoreWriter) PutByte(v byte) { w.grow(1); w.buf[w.pos] = v; w.pos++ }

// WriteBool …
func (w *StoreWriter) WriteBool(v bool) {
	if v {
		w.PutByte(1)
	} else {
		w.PutByte(0)
	}
}

// WriteWord …
func (w *StoreWriter) WriteWord(v uint16) {
	w.grow(2)
	binary.LittleEndian.PutUint16(w.buf[w.pos:], v)
	w.pos += 2
}

// WriteInt …
func (w *StoreWriter) WriteInt(v int32) {
	w.grow(4)
	binary.LittleEndian.PutUint32(w.buf[w.pos:], uint32(v))
	w.pos += 4
}

// WriteInt64 …
func (w *StoreWriter) WriteInt64(v int64) {
	w.grow(8)
	binary.LittleEndian.PutUint64(w.buf[w.pos:], uint64(v))
	w.pos += 8
}

// WriteFloat …
func (w *StoreWriter) WriteFloat(v float32) {
	w.grow(4)
	binary.LittleEndian.PutUint32(w.buf[w.pos:], math.Float32bits(v))
	w.pos += 4
}

// WriteDouble …
func (w *StoreWriter) WriteDouble(v float64) {
	w.grow(8)
	binary.LittleEndian.PutUint64(w.buf[w.pos:], math.Float64bits(v))
	w.pos += 8
}

// WriteString 窄串 [u32 len][bytes]。
func (w *StoreWriter) WriteString(s string) {
	w.WriteInt(int32(len(s)))
	w.grow(len(s))
	copy(w.buf[w.pos:], s)
	w.pos += len(s)
}

// WriteWideStr 宽串 [u32 码元数][UTF-16LE]。
func (w *StoreWriter) WriteWideStr(s string) {
	units := utf16.Encode([]rune(s))
	w.WriteInt(int32(len(units)))
	for _, u := range units {
		w.WriteWord(u)
	}
}

// WriteObject PERSISTID（4B）。
func (w *StoreWriter) WriteObject(v uint32) { w.WriteInt(int32(v)) }

// VaryTag put/get_vary_message 元素类型标签（§6.2）。
const (
	VaryTagInt     byte = 2
	VaryTagInt64   byte = 3
	VaryTagFloat   byte = 4
	VaryTagDouble  byte = 5
	VaryTagString  byte = 6
	VaryTagWideStr byte = 7
	VaryTagObject  byte = 8
)

// VaryValue vary 元素（tag + 值）。
type VaryValue struct {
	Tag byte
	I   int64
	F   float64
	S   string
}

// PutVaryMessage 复刻 put_vary_message：每元素 {u8 tag, 值}（无计数前缀）。
func PutVaryMessage(w *StoreWriter, vals []VaryValue) bool {
	for _, v := range vals {
		w.PutByte(v.Tag)
		switch v.Tag {
		case VaryTagInt, VaryTagObject:
			w.WriteInt(int32(v.I))
		case VaryTagInt64:
			w.WriteInt64(v.I)
		case VaryTagFloat:
			w.WriteFloat(float32(v.F))
		case VaryTagDouble:
			w.WriteDouble(v.F)
		case VaryTagString:
			w.WriteString(v.S)
		case VaryTagWideStr:
			w.WriteWideStr(v.S)
		default:
			return false // "(put_vary_message)error"
		}
	}
	return true
}

// LoadReader 复刻 CLoadArchive：读侧原语（越界即错，CExceptMsg 语义）。
type LoadReader struct {
	buf []byte
	pos int
}

// NewLoadReader …
func NewLoadReader(data []byte) *LoadReader { return &LoadReader{buf: data} }

// Seek …
func (r *LoadReader) Seek(pos int) { r.pos = pos }

// Pos …
func (r *LoadReader) Pos() int { return r.pos }

// Remaining …
func (r *LoadReader) Remaining() int { return len(r.buf) - r.pos }

// take …
func (r *LoadReader) take(n int) ([]byte, error) {
	if n < 0 || r.pos+n > len(r.buf) {
		return nil, fmt.Errorf("archive out of range")
	}
	b := r.buf[r.pos : r.pos+n]
	r.pos += n
	return b, nil
}

// ReadByte …
func (r *LoadReader) ReadByte() (byte, error) {
	b, err := r.take(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

// ReadBool …
func (r *LoadReader) ReadBool() (bool, error) {
	v, err := r.ReadByte()
	return v != 0, err
}

// ReadWord …
func (r *LoadReader) ReadWord() (uint16, error) {
	b, err := r.take(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b), nil
}

// ReadInt …
func (r *LoadReader) ReadInt() (int32, error) {
	b, err := r.take(4)
	if err != nil {
		return 0, err
	}
	return int32(binary.LittleEndian.Uint32(b)), nil
}

// ReadInt64 …
func (r *LoadReader) ReadInt64() (int64, error) {
	b, err := r.take(8)
	if err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(b)), nil
}

// ReadFloat …
func (r *LoadReader) ReadFloat() (float32, error) {
	v, err := r.ReadInt()
	return math.Float32frombits(uint32(v)), err
}

// ReadDouble …
func (r *LoadReader) ReadDouble() (float64, error) {
	v, err := r.ReadInt64()
	return math.Float64frombits(uint64(v)), err
}

// ReadString 窄串 [u32 len][bytes]。
func (r *LoadReader) ReadString() (string, error) {
	n, err := r.ReadInt()
	if err != nil {
		return "", err
	}
	if n < 0 || int(n) > r.Remaining() {
		return "", fmt.Errorf("string length out of range")
	}
	b, err := r.take(int(n))
	return string(b), err
}

// ReadWideStr 宽串 [u32 码元数][UTF-16LE]。
func (r *LoadReader) ReadWideStr() (string, error) {
	n, err := r.ReadInt()
	if err != nil {
		return "", err
	}
	if n < 0 || int(n)*2 > r.Remaining() {
		return "", fmt.Errorf("wide string length out of range")
	}
	b, err := r.take(int(n) * 2)
	if err != nil {
		return "", err
	}
	units := make([]uint16, n)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	return string(utf16.Decode(units)), nil
}

// ReadObject PERSISTID（4B）。
func (r *LoadReader) ReadObject() (uint32, error) {
	v, err := r.ReadInt()
	return uint32(v), err
}

// GetVaryMessage 复刻 get_vary_message(count, archive, varlist)。
func GetVaryMessage(count int, r *LoadReader) ([]VaryValue, error) {
	out := make([]VaryValue, 0, count)
	for i := 0; i < count; i++ {
		tag, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		v := VaryValue{Tag: tag}
		switch tag {
		case VaryTagInt, VaryTagObject:
			n, err := r.ReadInt()
			if err != nil {
				return nil, err
			}
			v.I = int64(n)
		case VaryTagInt64:
			if v.I, err = r.ReadInt64(); err != nil {
				return nil, err
			}
		case VaryTagFloat:
			f, err := r.ReadFloat()
			if err != nil {
				return nil, err
			}
			v.F = float64(f)
		case VaryTagDouble:
			if v.F, err = r.ReadDouble(); err != nil {
				return nil, err
			}
		case VaryTagString:
			if v.S, err = r.ReadString(); err != nil {
				return nil, err
			}
		case VaryTagWideStr:
			if v.S, err = r.ReadWideStr(); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("error arg type")
		}
		out = append(out, v)
	}
	return out, nil
}
