package mpq

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
)

// MPQ 压缩数据的解压/压缩（sector、单单元、各表）。
// 支持 zlib（0x02）与 PKWARE（0x08）；bzip2 / LZMA 暂不支持（明确报错）。

// 压缩方法字节。
const (
	mHuffman byte = 0x01
	mZlib    byte = 0x02
	mPkware  byte = 0x08
	mBZip2   byte = 0x10
	mLzma    byte = 0x12 // 0x12 是完整方法值，而非标志组合
)

// zlibDecompress 解压 zlib 容器 deflate 流；最多取 maxOut 字节（take 语义）。
func zlibDecompress(data []byte, maxOut int) ([]byte, error) {
	if maxOut <= 0 {
		return nil, nil
	}
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("mpq: zlib: %w", err)
	}
	defer r.Close()
	buf := bytes.NewBuffer(make([]byte, 0, min(maxOut, 1<<20)))
	// io.CopyN 读满 maxOut 即止；流提前干净结束（io.EOF）时返回已解出的部分。
	if _, err := io.CopyN(buf, r, int64(maxOut)); err != nil && err != io.EOF {
		return nil, fmt.Errorf("mpq: zlib: %w", err)
	}
	return buf.Bytes(), nil
}

// zlibCompress 压缩为 zlib 流（写入侧对应 decompressMasked 的 zlib 方法）。
func zlibCompress(data []byte) []byte {
	var buf bytes.Buffer
	w, err := zlib.NewWriterLevel(&buf, zlib.DefaultCompression)
	if err != nil {
		panic(err) // DefaultCompression 恒合法，不可达
	}
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

// decompressOne 单步解压（无方法字节）。
func decompressOne(method byte, data []byte, maxOut int) ([]byte, error) {
	switch method {
	case 0x02:
		return zlibDecompress(data, maxOut)
	case 0x08:
		out := bytes.NewBuffer(make([]byte, 0, min(maxOut, 1<<20)))
		if err := Explode(data, out, maxOut); err != nil {
			return nil, err
		}
		return out.Bytes(), nil
	case 0x10:
		return nil, fmt.Errorf("mpq: 暂不支持 bzip2 压缩方法 0x10（本模块只实现 zlib/PKWARE）")
	case 0x12:
		return nil, fmt.Errorf("mpq: 暂不支持 LZMA 压缩方法 0x12（本模块只实现 zlib/PKWARE）")
	default:
		return nil, fmt.Errorf("mpq: 无法支持的压缩方法 0x%x", method)
	}
}

// decompressMasked 解压带方法字节的 MPQ 压缩块（COMPRESS 标志文件的
// sector、单单元、扩展表）。白名单与 StormLib 的 SCompDecompress2 一致；
// 本模块可解 0x02/0x08，其余（含 bzip2/LZMA）明确报错。
func decompressMasked(data []byte, maxOut int) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}
	mask := data[0]

	if isKnownMethod(mask) {
		if mask == 0x02 || mask == 0x08 {
			return decompressOne(mask, data[1:], maxOut)
		}
		// 0x10 / 0x12 等已知但本模块未实现的方法。
		return nil, fmt.Errorf("mpq: 暂不支持的压缩方法 0x%x（本模块只实现 zlib 0x02 / PKWARE 0x08）", mask)
	}

	// 回退（宽容处理）：方法字节未知或缺失 —— 假定整个块是普通 zlib 流。
	if out, err := zlibDecompress(data, maxOut); err == nil {
		return out, nil
	}
	return nil, fmt.Errorf("mpq: 无法支持的压缩方法 0x%x", mask)
}

func isKnownMethod(mask byte) bool {
	switch mask {
	case 0x01, 0x02, 0x08, 0x10, 0x12, 0x20, 0x22, 0x30, 0x41, 0x81:
		return true
	default:
		return false
	}
}

// decompressImploded 解压旧式 IMPLODE 标志存储的原始 PKWARE 流（无方法字节）。
func decompressImploded(data []byte, maxOut int) ([]byte, error) {
	out := bytes.NewBuffer(make([]byte, 0, min(maxOut, 1<<20)))
	if err := Explode(data, out, maxOut); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
