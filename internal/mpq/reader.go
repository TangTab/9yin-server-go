package mpq

import (
	"os"
)

// byteReader 是随机读抽象：MPQ 各处以绝对偏移读定长字节块，
// 越界部分补零（与 read_exact_or_eof 的零填充语义一致）。
type byteReader interface {
	// Length 返回文件总长。
	Length() int64
	// ReadAtOrEof 从 offset 起读 n 字节；越过 EOF 的部分以零填充。
	ReadAtOrEof(off int64, n int) []byte
}

// memReader 是内存实现。
type memReader struct {
	data []byte
}

func (m *memReader) Length() int64 { return int64(len(m.data)) }

func (m *memReader) ReadAtOrEof(off int64, n int) []byte {
	out := make([]byte, n)
	available := int64(n)
	if available > int64(len(m.data))-off {
		available = int64(len(m.data)) - off
	}
	if available > 0 {
		copy(out, m.data[off:off+available])
	}
	return out
}

// fileReader 是文件实现。os.File.ReadAt 并发安全，无需额外加锁。
type fileReader struct {
	f *os.File
}

func (r *fileReader) Length() int64 {
	st, err := r.f.Stat()
	if err != nil {
		return 0
	}
	return st.Size()
}

func (r *fileReader) ReadAtOrEof(off int64, n int) []byte {
	out := make([]byte, n)
	// ReadAt 在 EOF 处返回短读，剩余保持为零。
	_, _ = r.f.ReadAt(out, off)
	return out
}
