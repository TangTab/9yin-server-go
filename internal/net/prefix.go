package net

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

// PrefixFramer 是前缀式成帧器（长度前置，无转义无定界符）：
//
//	帧布局 = [4 字节小端长度 N][N 字节帧体]
//
// 对应黑金 FsComm 反汇编确认的内部前缀式帧形态（UserSock::SendMsg 的
// headPtr/headLen 前缀缓冲语义），用于进程内/进程间总线（proc）等
// 可信链路——无转义开销，一次长度读取即可分配精确缓冲。
// 客户端线路（EE 定界）不用它。
type PrefixFramer struct {
	reader *bufio.Reader
	writer io.Writer
	limit  int
}

// NewPrefixFramer 在流上创建前缀式成帧器。
func NewPrefixFramer(rw io.ReadWriter, limit int) (*PrefixFramer, error) {
	if rw == nil {
		return nil, fmt.Errorf("nil prefix framer stream")
	}
	if limit <= 0 {
		return nil, ErrInvalidLimit
	}
	return &PrefixFramer{reader: bufio.NewReader(rw), writer: rw, limit: limit}, nil
}

// PrefixFramerFactory 供 Options 引用。
func PrefixFramerFactory(rw io.ReadWriter, limit int) (Framer, error) {
	return NewPrefixFramer(rw, limit)
}

// ReadFrame 读 4 字节长度头 + 帧体。
func (f *PrefixFramer) ReadFrame() ([]byte, error) {
	var head [4]byte
	if _, err := io.ReadFull(f.reader, head[:]); err != nil {
		if err == io.ErrUnexpectedEOF {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	n := binary.LittleEndian.Uint32(head[:])
	if n == 0 || int(n) > f.limit {
		return nil, fmt.Errorf("%w: size %d, limit %d", ErrFrameTooLarge, n, f.limit)
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(f.reader, body); err != nil {
		return nil, err
	}
	return body, nil
}

// WriteFrame 写 4 字节长度头 + 帧体。
func (f *PrefixFramer) WriteFrame(body []byte) error {
	if len(body) == 0 || len(body) > f.limit {
		return fmt.Errorf("%w: size %d, limit %d", ErrFrameTooLarge, len(body), f.limit)
	}
	var head [4]byte
	binary.LittleEndian.PutUint32(head[:], uint32(len(body)))
	if _, err := f.writer.Write(head[:]); err != nil {
		return err
	}
	_, err := f.writer.Write(body)
	return err
}
