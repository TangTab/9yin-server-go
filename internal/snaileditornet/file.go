package snaileditornet

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

// 文件传输（官方 SendFile/GetFile + _fileworkitem：0x200/0x1000 分块 +
// "Start/End Send File:%s,times:%d!" 计时）。

// ChunkSmall/ChunkLarge 官方观察到的两种分块。
const (
	ChunkSmall  = 0x200
	ChunkLarge  = 0x1000
	FileTimeout = 30 * time.Second
)

// SendFile 流式发送文件：握手 → 数据块（ChunkLarge）→ 结束帧（携带耗时）。
// 帧序列（自 0x3E9 握手 + 0x3EA/0x3ED/0x3EF 块族归纳）：
// FMsgFileHandshake(文件名) → FMsgFileChunk*（载荷分块）→ FMsgFileAck(结束，耗时文本)。
func SendFile(conn net.Conn, name string, r io.Reader) error {
	start := time.Now()
	if err := WriteFrame(conn, FMsgFileHandshake, []byte(name)); err != nil {
		return err
	}
	buf := make([]byte, ChunkLarge)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if werr := WriteFrame(conn, FMsgFileChunk, buf[:n]); werr != nil {
				return werr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return WriteFrame(conn, FMsgFileAck,
		[]byte(fmt.Sprintf("End Send File:%s,times:%d!", name, time.Since(start).Milliseconds())))
}

// GetFile 接收文件（对端 SendFile 镜像）。
func GetFile(conn net.Conn, save func(name string, r io.Reader) error) error {
	f, err := ReadFrame(conn)
	if err != nil {
		return err
	}
	if f.MsgID != FMsgFileHandshake {
		return fmt.Errorf("snaileditornet: expect handshake, got %#x", f.MsgID)
	}
	name := string(f.Payload)
	pr, pw := io.Pipe()
	go func() {
		for {
			f, err := ReadFrame(conn)
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			switch f.MsgID {
			case FMsgFileChunk, FMsgFileChunkVariant, FMsgFileChunkVariant2:
				if _, err := pw.Write(f.Payload); err != nil {
					return
				}
			case FMsgFileAck:
				pw.Close()
				return
			default:
				pw.CloseWithError(fmt.Errorf("unexpected frame %#x", f.MsgID))
				return
			}
		}
	}()
	return save(name, pr)
}

// SendFileFromPath 便利封装。
func SendFileFromPath(conn net.Conn, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return SendFile(conn, path, f)
}

// FileDigest 校验和（对拍辅助；官方无 MD5，仅计时）。
func FileDigest(b []byte) string { return fmt.Sprintf("%x", md5.Sum(b)) }

var _ = bytes.MinRead // 保持 bytes 引用（对拍工具使用）
