package fssolorobot

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf16"
)

// archive 复刻官方 CLoadArchive（utils/loadarchive.h：data/maxlen/cursor
// 三元组，读越界抛 CExceptMsg 'Out range'）。Go 版以 error 表达。
type archive struct {
	data []byte
	pos  int
}

func (a *archive) take(n int) ([]byte, error) {
	if a.pos+n > len(a.data) {
		return nil, fmt.Errorf("Out range")
	}
	b := a.data[a.pos : a.pos+n]
	a.pos += n
	return b, nil
}

func (a *archive) readByte() (byte, error) {
	b, err := a.take(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (a *archive) readWord() (uint16, error) {
	b, err := a.take(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b), nil
}

func (a *archive) readInt() (int32, error) {
	b, err := a.take(4)
	if err != nil {
		return 0, err
	}
	return int32(binary.LittleEndian.Uint32(b)), nil
}

// readUCS2 官方 ReadUCS2@0x1005e0a0：[u32 字节长][UTF-16LE 数据]，
// 要求末尾 wchar == 0（否则官方抛 'UCS2 string no end symbol'）。
func (a *archive) readUCS2() (string, error) {
	b, err := a.take(4)
	if err != nil {
		return "", err
	}
	n := int(binary.LittleEndian.Uint32(b))
	if n%2 != 0 {
		return "", fmt.Errorf("UCS2 string no end symbol")
	}
	data, err := a.take(n)
	if err != nil {
		return "", err
	}
	if n >= 2 && binary.LittleEndian.Uint16(data[n-2:]) != 0 {
		return "", fmt.Errorf("UCS2 string no end symbol")
	}
	u := make([]uint16, n/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(data[2*i:])
	}
	if len(u) > 0 && u[len(u)-1] == 0 {
		u = u[:len(u)-1]
	}
	return string(utf16.Decode(u)), nil
}

// receiveServerMsgLocked 官方 SoloRobot::ReceiveServerMsg@0x1005cec0
// （含 0x28 解包与 7 类处理器）。调用方持有 r.mu。
// 返回 false 仅代表本帧解析失败（连接保持，官方同）。
func (r *Robot) receiveServerMsgLocked(id int, c *connect, payload []byte) bool {
	if len(payload) < 1 {
		r.trace("(SoloRobot::ReceiveServerMsg)message too short")
		return false
	}
	if payload[0] == msgRecvCompressed {
		payload = r.unwrapCompressedLocked(payload)
		if payload == nil {
			return false
		}
	}

	switch payload[0] {
	case msgRecvErrorCode: // 0x03 @0x1005d326
		return r.recvErrorCodeLocked(c, payload)
	case msgRecvLoginSucc: // 0x04 @0x1005d011
		return r.recvLoginSucceedLocked(c, payload)
	case msgRecvEntryScene: // 0x0B @0x1005d3a0
		return r.recvEntrySceneLocked(id, c, payload)
	case msgRecvAddObject: // 0x0D @0x1005d46a
		return r.recvAddObjectLocked(c, payload)
	case msgRecvLocation: // 0x1F @0x1005d50a
		return r.recvLocationLocked(c, payload)
	case msgRecvMoving: // 0x20 @0x1005d5aa
		return r.recvMovingLocked(c, payload)
	case msgRecvAllDest: // 0x21 @0x1005d638
		return r.recvAllDestLocked(c, payload)
	default:
		return true // 官方未知 id 忽略（返回 true）
	}
}

// unwrapCompressedLocked 官方 0x28 包裹（0x1005cf2c..0x1005cfd5）：
// qlz_size_decompressed(p+1) ≤0xFFFF；qlz_decompress(p+1, buf+1) 产出
// 必须等长；缓冲[0] 强制 0x0D（解压体 = add_object 报文体，无 id 字节）。
func (r *Robot) unwrapCompressedLocked(payload []byte) []byte {
	dsize, err := QLZSizeDecompressed(payload[1:])
	if err != nil || dsize > 0xFFFF {
		r.trace("(SoloRobot::ReceiveServerMsg)decompress size error")
		return nil
	}
	buf := make([]byte, dsize+1)
	buf[0] = msgRecvAddObject
	n, err := QLZDecompress(payload[1:], buf[1:])
	if err != nil || n != dsize {
		r.trace("(SoloRobot::ReceiveServerMsg)decompress failed")
		return nil
	}
	return buf
}

// robotName 官方账号/角色名口径：AccountPrefix + 连接序号（'%s%d'）。
func (r *Robot) robotName(c *connect) string {
	return fmt.Sprintf("%s%d", r.cfg.AccountPrefix, c.index)
}

// recvErrorCodeLocked len≥5；trace 'robot %s%d receive error code %d'；
// 状态 6 failed（终局，官方不重连）。
func (r *Robot) recvErrorCodeLocked(c *connect, payload []byte) bool {
	if len(payload) < 5 {
		r.trace("(SoloRobot::ReceiveServerMsg)error_code error")
		return false
	}
	code := int32(binary.LittleEndian.Uint32(payload[1:5]))
	r.trace("robot %s%d receive error code %d", r.cfg.AccountPrefix, c.index, code)
	c.state = ConnFailed
	return true
}

// recvLoginSucceedLocked len≥0x29；角色数 int@0x25：
//
//	>0：Seek(0x29) 读角色条目 word,byte,int,byte,int,byte,UCS2(名),
//	    byte,UCS2 → 发 choose_char(名) → 状态 4 choosing；
//	≤0：发 create_char(prefix+序号) → 状态 3 creating；
//	解析异常（官方 CExceptMsg catch @0x1005d215）→ 状态 6 + trace。
func (r *Robot) recvLoginSucceedLocked(c *connect, payload []byte) bool {
	if len(payload) < 0x29 {
		r.trace("(SoloRobot::ReceiveServerMsg)login_succeed error")
		return false
	}
	count := int32(binary.LittleEndian.Uint32(payload[0x25:0x29]))
	if count > 0 {
		ar := &archive{data: payload}
		ar.pos = 0x29
		if _, err := ar.readWord(); err != nil {
			return r.loginParseFailLocked(c, err)
		}
		if _, err := ar.readByte(); err != nil {
			return r.loginParseFailLocked(c, err)
		}
		if _, err := ar.readInt(); err != nil {
			return r.loginParseFailLocked(c, err)
		}
		if _, err := ar.readByte(); err != nil {
			return r.loginParseFailLocked(c, err)
		}
		if _, err := ar.readInt(); err != nil {
			return r.loginParseFailLocked(c, err)
		}
		if _, err := ar.readByte(); err != nil {
			return r.loginParseFailLocked(c, err)
		}
		name, err := ar.readUCS2()
		if err != nil {
			return r.loginParseFailLocked(c, err)
		}
		r.sendToServerLocked(c, buildChooseChar(name))
		c.state = ConnChoosing
		return true
	}
	r.sendToServerLocked(c, buildCreateChar(r.robotName(c)))
	c.state = ConnCreating
	return true
}

func (r *Robot) loginParseFailLocked(c *connect, err error) bool {
	r.trace("(SoloRobot::ReceiveServerMsg)login_succeed")
	r.trace("%s", err.Error())
	c.state = ConnFailed
	return true
}

// recvEntrySceneLocked len≥0xF；PERSISTID = int@5/int@9；回 1 字节
// 0x09；trace 'robot %s%d entry scene'；状态 5 ingame。
func (r *Robot) recvEntrySceneLocked(id int, c *connect, payload []byte) bool {
	if len(payload) < 0xF {
		r.trace("(SoloRobot::ReceiveServerMsg)entry_scene error")
		return false
	}
	c.ident1 = binary.LittleEndian.Uint32(payload[5:9])
	c.ident2 = binary.LittleEndian.Uint32(payload[9:13])
	r.sendToServerLocked(c, buildEntryAck())
	r.trace("robot %s%d entry scene", r.cfg.AccountPrefix, c.index)
	c.state = ConnInGame
	return true
}

func payloadInt32(p []byte, off int) uint32 {
	if off+4 > len(p) {
		return 0
	}
	return binary.LittleEndian.Uint32(p[off : off+4])
}

func payloadFloat32(p []byte, off int) float32 {
	return math.Float32frombits(payloadInt32(p, off))
}

// recvAddObjectLocked len≥0x3F；ident1@1/ident2@5/x@9/y@0xD/z@0x11；
// 匹配自身 → 更新坐标。
func (r *Robot) recvAddObjectLocked(c *connect, payload []byte) bool {
	if len(payload) < 0x3F {
		r.trace("(SoloRobot::ReceiveServerMsg)add_object error")
		return false
	}
	if c.ident1 == payloadInt32(payload, 1) && c.ident2 == payloadInt32(payload, 5) {
		c.x = payloadFloat32(payload, 9)
		c.y = payloadFloat32(payload, 0xD)
		c.z = payloadFloat32(payload, 0x11)
	}
	return true
}

// recvLocationLocked len≥0x19；字段同 add_object；匹配 → 更新 x/y/z。
func (r *Robot) recvLocationLocked(c *connect, payload []byte) bool {
	if len(payload) < 0x19 {
		r.trace("(SoloRobot::ReceiveServerMsg)location error")
		return false
	}
	if c.ident1 == payloadInt32(payload, 1) && c.ident2 == payloadInt32(payload, 5) {
		c.x = payloadFloat32(payload, 9)
		c.y = payloadFloat32(payload, 0xD)
		c.z = payloadFloat32(payload, 0x11)
	}
	return true
}

// recvMovingLocked len≥0x2D；字段同上；匹配 → 只更新 x/z。
func (r *Robot) recvMovingLocked(c *connect, payload []byte) bool {
	if len(payload) < 0x2D {
		r.trace("(SoloRobot::ReceiveServerMsg)moving error")
		return false
	}
	if c.ident1 == payloadInt32(payload, 1) && c.ident2 == payloadInt32(payload, 5) {
		c.x = payloadFloat32(payload, 9)
		c.z = payloadFloat32(payload, 0x11)
	}
	return true
}

// recvAllDestLocked len≥0x2F；u16 数量@1；条目 0x2C 字节自 @3：
// ident1@+0/ident2@+4/x@+8(f32)/z@+0x10(f32)；匹配自身的条目 → 更新 x/z。
func (r *Robot) recvAllDestLocked(c *connect, payload []byte) bool {
	if len(payload) < 0x2F {
		r.trace("(SoloRobot::ReceiveServerMsg)all dest error")
		return false
	}
	count := int(binary.LittleEndian.Uint16(payload[1:3]))
	for i := 0; i < count; i++ {
		base := 3 + i*0x2C
		if base+0x2C > len(payload) {
			break // 官方信压缩器不越界；Go 版防御截断
		}
		e := payload[base : base+0x2C]
		if c.ident1 == payloadInt32(e, 0) && c.ident2 == payloadInt32(e, 4) {
			c.x = payloadFloat32(e, 8)
			c.z = payloadFloat32(e, 0x10)
		}
	}
	return true
}
