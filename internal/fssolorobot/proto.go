package fssolorobot

import (
	"encoding/binary"
	"math"
	"unicode/utf16"
)

// 线路报文 id（首字节；方向不同而数值可同）。
//
// 官方定案（docs/official/FsSoloRobot全解与Go复刻.md §五）：
// 机器人→服务器：0x02 login / 0x04 choose_char / 0x05 create_char /
// 0x09 entry_scene 应答 / 0x0B move；
// 服务器→机器人：0x03 error_code / 0x04 login_succeed / 0x0B
// entry_scene / 0x0D add_object / 0x1F location / 0x20 moving /
// 0x21 all_dest / 0x28 QuickLZ 压缩包裹（解压体 = add_object 体）。
const (
	msgSendLogin       = 0x02
	msgSendChooseChar  = 0x04
	msgSendCreateChar  = 0x05
	msgSendEntryAck    = 0x09
	msgSendMove        = 0x0B
	msgRecvErrorCode   = 0x03
	msgRecvLoginSucc   = 0x04
	msgRecvEntryScene  = 0x0B
	msgRecvAddObject   = 0x0D
	msgRecvLocation    = 0x1F
	msgRecvMoving      = 0x20
	msgRecvAllDest     = 0x21
	msgRecvCompressed  = 0x28
	loginVersion       = 0x300 // login 报文版本字段（send_login@0x1005e61e）
	moveDestY          = 4.0   // move 目标 y 常量（0x100ca588）
	sendMaxPayload     = 0x1FE // SendToServer payload 上限
	sendStuffedLimit   = 0x400 // 转义后缓冲上限（assert count <= sizeof）
)

// align16FieldLen 官方 login 字符串字段长度域：(strlen+1+0xF)>>4<<4
// （恒 ≥16，len=0 → 16）。
func align16FieldLen(n int) int { return (n + 1 + 15) &^ 15 }

// buildLogin 官方 send_login（OnConnect 内联，0x1005e5f5..0x1005e920）：
//
//	[0]=02 [1..5)=u32 0x300
//	+ 3 个字段（账号 / 密码 / 登录串）：[u32 align16(strlen+1)][ascii+NUL 零垫]
func buildLogin(account, password, loginStr string) []byte {
	buf := make([]byte, 0x15)
	buf[0] = msgSendLogin
	binary.LittleEndian.PutUint32(buf[1:5], loginVersion)
	appendLoginField := func(s string) {
		pad := align16FieldLen(len(s))
		lenPos := len(buf)
		buf = append(buf, 0, 0, 0, 0)
		binary.LittleEndian.PutUint32(buf[lenPos:lenPos+4], uint32(pad))
		start := len(buf)
		buf = append(buf, make([]byte, pad)...)
		copy(buf[start:], s)
		buf[start+len(s)] = 0
	}
	appendLoginField(account)
	appendLoginField(password)
	appendLoginField(loginStr)
	return buf
}

// appendUCS2Fixed 把 name 的 UTF-16LE（含 L'\0'）写进 dst 定长字段：
// 全长 2*chars+2 ≤ size 时整段写入，否则截 size-2 字节并在末字位置
// 补零字（官方 choose：≤0x42 else 0x40+零字@+0x40；create 同型）。
func appendUCS2Fixed(buf []byte, off, size int, name string) {
	u := utf16.Encode([]rune(name))
	full := 2*len(u) + 2
	if full <= size {
		for i, c := range u {
			binary.LittleEndian.PutUint16(buf[off+2*i:], uint16(c))
		}
		binary.LittleEndian.PutUint16(buf[off+2*len(u):], 0)
		return
	}
	n := size - 2
	for i := 0; i < n/2; i++ {
		binary.LittleEndian.PutUint16(buf[off+2*i:], uint16(u[i]))
	}
	binary.LittleEndian.PutUint16(buf[off+n:], 0)
}

// buildChooseChar 官方 choose_char：[0]=04，名字 UCS2@+1（0x42 字段），
// 报文定长 0x54（其余 memset 0）。
func buildChooseChar(name string) []byte {
	buf := make([]byte, 0x54)
	buf[0] = msgSendChooseChar
	appendUCS2Fixed(buf, 1, 0x42, name)
	return buf
}

// buildCreateChar 官方 create_char：[0]=05 [1..5)=u32 0 [5..7)=u16 0，
// 名字 UCS2@+7（0x42 字段），报文定长 0x49。
func buildCreateChar(name string) []byte {
	buf := make([]byte, 0x49)
	buf[0] = msgSendCreateChar
	// [1..5) 与 [5..7) 保持 0
	appendUCS2Fixed(buf, 7, 0x42, name)
	return buf
}

// buildEntryAck 入景应答：单字节 0x09（entry_scene 处理器 0x1005d401）。
func buildEntryAck() []byte { return []byte{msgSendEntryAck} }

// buildMove 官方 move（RobotStateOpened@0x1005f91）：
//
//	[0]=0x0B [1]=1 [2..4)=u16 6 [4]=f32 x [8]=f32 y [12]=f32 z
//	[16]=f32 dest_x [20]=f32 dest_z [24]=f32 4.0，定长 0x60 其余零。
func buildMove(x, y, z, destX, destZ float32) []byte {
	buf := make([]byte, 0x60)
	buf[0] = msgSendMove
	buf[1] = 1
	binary.LittleEndian.PutUint16(buf[2:4], 6)
	binary.LittleEndian.PutUint32(buf[4:8], math.Float32bits(x))
	binary.LittleEndian.PutUint32(buf[8:12], math.Float32bits(y))
	binary.LittleEndian.PutUint32(buf[12:16], math.Float32bits(z))
	binary.LittleEndian.PutUint32(buf[16:20], math.Float32bits(destX))
	binary.LittleEndian.PutUint32(buf[20:24], math.Float32bits(destZ))
	binary.LittleEndian.PutUint32(buf[24:28], math.Float32bits(moveDestY))
	return buf
}

// escapeFrame 官方 SendToServer 转义：0xEE→"EE 00"，帧尾追加 "EE EE"。
func escapeFrame(payload []byte) []byte {
	wire := make([]byte, 0, len(payload)+2)
	for _, b := range payload {
		wire = append(wire, b)
		if b == 0xEE {
			wire = append(wire, 0x00)
		}
	}
	return append(wire, 0xEE, 0xEE)
}
