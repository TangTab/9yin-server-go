package net

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

// DefaultMaxFrameSize 单帧上限（1MB，与原版探测经验一致）。
const DefaultMaxFrameSize = 1 << 20

var (
	// ErrFrameTooLarge 帧体超过上限。
	ErrFrameTooLarge = errors.New("net frame too large")
	// ErrInvalidEscape 非法转义序列 EE xx（xx 既非 00 也非 EE）。
	ErrInvalidEscape = errors.New("net invalid EE escape")
	// ErrInvalidLimit 非法帧上限配置。
	ErrInvalidLimit = errors.New("net invalid frame size limit")
)

// Framer 是成帧层：在字节流上读写完整帧，不解释内容、不做编码。
// 帧体进 Framer 前后均为"已编码"字节（编码由 Cipher 负责，两层正交）。
type Framer interface {
	// ReadFrame 读取一帧并返回帧体（编码态）；流结束返回 io.EOF。
	ReadFrame() ([]byte, error)
	// WriteFrame 写入一帧帧体（编码态）。
	WriteFrame(body []byte) error
}

// FramerFactory 由 Connection 按流构造成帧器。
type FramerFactory func(rw io.ReadWriter, limit int) (Framer, error)

// EscapeFramer 是九阴现代客户端线路格式的成帧器：
//
//	帧体以 EE 定界：EE 00 表示字面 0xEE 字节，EE EE 为帧终止符
//	无长度前缀，流式扫描直至终止符
type EscapeFramer struct {
	reader *bufio.Reader
	writer io.Writer
	limit  int
}

// NewEscapeFramer 在流上创建 EE 转义成帧器。
func NewEscapeFramer(rw io.ReadWriter, limit int) (*EscapeFramer, error) {
	if rw == nil {
		return nil, fmt.Errorf("nil escape framer stream")
	}
	if limit <= 0 {
		return nil, ErrInvalidLimit
	}
	return &EscapeFramer{reader: bufio.NewReader(rw), writer: rw, limit: limit}, nil
}

// EscapeFramerFactory 供 Options 引用。
func EscapeFramerFactory(rw io.ReadWriter, limit int) (Framer, error) {
	return NewEscapeFramer(rw, limit)
}

// ReadFrame 扫描至 EE EE 终止符，返回反转义后的帧体（编码态）。
func (f *EscapeFramer) ReadFrame() ([]byte, error) {
	body := make([]byte, 0, 256)
	started := false
	for {
		b, err := f.reader.ReadByte()
		if err != nil {
			if err == io.EOF && started {
				return nil, io.ErrUnexpectedEOF
			}
			return nil, err
		}
		started = true
		if b != 0xEE {
			body = append(body, b)
		} else {
			next, err := f.reader.ReadByte()
			if err != nil {
				if err == io.EOF {
					return nil, io.ErrUnexpectedEOF
				}
				return nil, err
			}
			switch next {
			case 0x00:
				body = append(body, 0xEE)
			case 0xEE:
				return body, nil
			default:
				return nil, fmt.Errorf("%w: EE %02X", ErrInvalidEscape, next)
			}
		}
		if len(body) > f.limit {
			return nil, fmt.Errorf("%w: limit %d", ErrFrameTooLarge, f.limit)
		}
	}
}

// WriteFrame 转义帧体并追加 EE EE 终止符。
func (f *EscapeFramer) WriteFrame(body []byte) error {
	if len(body) > f.limit {
		return fmt.Errorf("%w: size %d, limit %d", ErrFrameTooLarge, len(body), f.limit)
	}
	wire := make([]byte, 0, len(body)+2)
	for _, b := range body {
		wire = append(wire, b)
		if b == 0xEE {
			wire = append(wire, 0x00)
		}
	}
	wire = append(wire, 0xEE, 0xEE)
	_, err := f.writer.Write(wire)
	return err
}

// EncodeFrameBody 纯函数形态的转义编码（golden 对拍/测试用）。
func EncodeFrameBody(body []byte) []byte {
	wire := make([]byte, 0, len(body)+2)
	for _, b := range body {
		wire = append(wire, b)
		if b == 0xEE {
			wire = append(wire, 0x00)
		}
	}
	return append(wire, 0xEE, 0xEE)
}
