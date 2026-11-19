package snaileditornet

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// FMSG 命令号（官方 ClientSession::OnRead switch 0x3E8+i，i=0..9）。
// 语义按各 case 处理体定案；精确枚举名需编辑器客户端交叉验证。
const (
	FMsgFileAck           uint32 = 0x3E8 // 0x200 缓冲回发
	FMsgFileHandshake     uint32 = 0x3E9 // 文件传输握手（0x1000 缓冲）
	FMsgFileChunk         uint32 = 0x3EA // 发送文件数据块
	FMsgFileInUse         uint32 = 0x3EB // 文件占用冲突（"[error]%s is used by other peaple!"）
	FMsgEditorCommand     uint32 = 0x3EC // 编辑器命令帧（0x800 载荷 → 游戏命令队列）
	FMsgFileChunkVariant  uint32 = 0x3ED // 发送文件数据块（变体）
	FMsgUnused3EE         uint32 = 0x3EE // 空分支
	FMsgFileChunkVariant2 uint32 = 0x3EF // 发送文件数据块（变体）
	FMsgUnused3F0         uint32 = 0x3F0 // 空分支
	FMsgServerSvnUpdate   uint32 = 0x3F1 // FMSG_SERVER_SVN_UPDATE：路径列表（'/'→'\\'）
)

// FrameHeaderLen 官方帧头 8 字节。
const FrameHeaderLen = 8

// MaxPayload 官方编辑器命令帧/共享内存帧宽 0x800；文件数据块另用
// 0x1000 缓冲，线协议按 1MB 防护上限（ReadFrame 不做 0x800 截断）。
const (
	MaxPayload   = 0x800
	hardFrameCap = 1 << 20
)

// Frame 一条 FMSG 帧：{u32 totalLen(含头); u32 msgId; payload}。
type Frame struct {
	MsgID   uint32
	Payload []byte
}

// EncodeFrame 构造线上字节（ServerSendSvnUpdate 同构：
// buf[0]=8+len; buf[4]=msgId; memcpy(buf+8,payload,len)）。
func EncodeFrame(msgID uint32, payload []byte) []byte {
	out := make([]byte, FrameHeaderLen+len(payload))
	binary.LittleEndian.PutUint32(out[0:], uint32(FrameHeaderLen+len(payload)))
	binary.LittleEndian.PutUint32(out[4:], msgID)
	copy(out[FrameHeaderLen:], payload)
	return out
}

// ErrFrameTooLarge 载荷超防护上限。
var ErrFrameTooLarge = errors.New("snaileditornet: frame payload too large")

// ReadFrame 从流读一帧（头 8B：totalLen+msgId；totalLen 含头）。
// totalLen < 8 或 > 8+0x800 视为对端违约。
func ReadFrame(r io.Reader) (*Frame, error) {
	var head [FrameHeaderLen]byte
	if _, err := io.ReadFull(r, head[:]); err != nil {
		return nil, err
	}
	total := binary.LittleEndian.Uint32(head[0:4])
	msgID := binary.LittleEndian.Uint32(head[4:8])
	if total < FrameHeaderLen {
		return nil, fmt.Errorf("snaileditornet: bad totalLen %d", total)
	}
	n := int(total) - FrameHeaderLen
	if n > hardFrameCap {
		return nil, ErrFrameTooLarge
	}
	f := &Frame{MsgID: msgID}
	if n > 0 {
		f.Payload = make([]byte, n)
		if _, err := io.ReadFull(r, f.Payload); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// WriteFrame 编码并整帧写出。
func WriteFrame(w io.Writer, msgID uint32, payload []byte) error {
	_, err := w.Write(EncodeFrame(msgID, payload))
	return err
}
