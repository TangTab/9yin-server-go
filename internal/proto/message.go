package proto

// 引擎级消息 ID（帧首字节，客户端线路）。来自旧分支 protocol-probe
// 对现代客户端/正式服的实测；黑金侧经 PDB 反汇编交叉验证为同族机制。
const (
	MsgWorldInfo    byte = 0x05 // 世界信息：[u16 infoType][UTF-16LE 文本]
	MsgActivity     byte = 0x0A // 活动信封：21B 不透明头 + TVarList
	MsgServerMenu   byte = 0x1C // SERVER_MENU（子菜单树）
	MsgPing         byte = 0x18 // 心跳请求（12B 定长）
	MsgPong         byte = 0x38 // 心跳应答（同体回显）
	MsgCustomDirect byte = 0x1E // 自定义消息直发（TVarList，首值为消息 ID）
	MsgCustomAlt    byte = 0x27 // 自定义消息直发（另一形态）
)

// HeartbeatReply 生成心跳应答：整帧拷贝、首字节 0x18→0x38。
// golden 钉死（fork 49840 抓包 326 对逐字节验证）：
//
//	18 31 54 47 0000 14 00000000 0000 → 38 31 54 47 0000 14 00000000 0000
func HeartbeatReply(ping []byte) []byte {
	reply := append([]byte(nil), ping...)
	if len(reply) > 0 {
		reply[0] = MsgPong
	}
	return reply
}

// IsCustomFrame 报告帧是否为自定义消息（0x1E/0x27 直发形态）。
func IsCustomFrame(frame []byte) bool {
	return len(frame) > 0 && (frame[0] == MsgCustomDirect || frame[0] == MsgCustomAlt)
}

// ParseCustomFrame 解析自定义消息帧：opcode 之后的字节为 TVarList，
// 其首值为自定义消息 ID（数值，C2S 目录见 custom_gen.go）。
func ParseCustomFrame(frame []byte) (opcode byte, body *VarList, err error) {
	if !IsCustomFrame(frame) {
		return 0, nil, fmtMsg("not a custom frame")
	}
	body, err = DecodeVarList(frame[1:])
	if err != nil {
		return 0, nil, err
	}
	return frame[0], body, nil
}

type fmtErr string

func (e fmtErr) Error() string { return "proto: " + string(e) }
func fmtMsg(s string) error    { return fmtErr(s) }
