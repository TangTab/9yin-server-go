package login

import (
	"encoding/binary"
	"time"
	"unicode/utf16"

	"github.com/lizhanyin/9yin-server-go/internal/proto"
)

// 登录链帧号（C2S 请求侧）。S2C 应答侧：0x04 ServerPlayerRoles、
// 0x05 ServerWorldInfo（注意 0x05 双向语义不同，按方向区分——
// 与引擎级 proto.MsgWorldInfo 同号）。
const (
	OpClientLogin    byte = 0x02 // 账号登录（四槽结构，见 auth.go）
	OpClientWorldReq byte = 0x03 // 世界信息请求 → 0x05 worldInfo 应答
	OpClientRoleSel  byte = 0x04 // 选角（重连带会话恢复字段）
	OpClientRoleNew  byte = 0x05 // 建角
)

// noRoleLoginSuccessFrame 生成无角色 0x04 ServerPlayerRoles 应答（41B）：
// 布局 golden 钉死（旧分支 fork 真机可登录形态）——
// [0x00]=0x04、[0x01]u32=1、[0x09/0x0D/0x11/0x15/0x19/0x1D]=年/月/日/时/分/秒。
// 有角色形态在 0x25 处置 1 并追加 TVarList 角色段（选角主干接入时扩展）。
func noRoleLoginSuccessFrame(now time.Time) []byte {
	msg := make([]byte, 0x29)
	msg[0] = OpServerPlayerRoles
	put := func(offset int, value uint32) { binary.LittleEndian.PutUint32(msg[offset:], value) }
	put(0x01, 1)
	put(0x09, uint32(now.Year()))
	put(0x0D, uint32(now.Month()))
	put(0x11, uint32(now.Day()))
	put(0x15, uint32(now.Hour()))
	put(0x19, uint32(now.Minute()))
	put(0x1D, uint32(now.Second()))
	return msg
}

// OpServerPlayerRoles 登录成功/选角列表应答帧号。
const OpServerPlayerRoles byte = 0x04

// worldInfoFrame 生成 0x05 ServerWorldInfo：[0x05][u16 infoType][UTF-16LE
// 文本 + NUL]。books 串为五书开放形态（旧分支真机回放值）。
func worldInfoFrame(infoType uint16, text string) []byte {
	units := utf16.Encode([]rune(text))
	msg := make([]byte, 3+(len(units)+1)*2)
	msg[0] = proto.MsgWorldInfo
	binary.LittleEndian.PutUint16(msg[1:], infoType)
	for i, unit := range units {
		binary.LittleEndian.PutUint16(msg[3+i*2:], unit)
	}
	return msg
}

// openBooks 五书开放世界串（op03 应答载荷，真机回放值）。
const openBooks = "0,book1,0,book2,0,book4,0,book5,0,book8,0,0"

// Flow 是登录链分派器：Gateway.OnFrame 的第一接入点。
// 当前覆盖首段：op02 账号登录（结构校验+身份摘要，回无角色 0x04）、
// op03 世界信息（回五书 worldInfo）。选角/建角/进场景随主干推进扩展。
type Flow struct {
	// Clock 时间源（应答帧日期字段；测试可注入）。
	Clock func() time.Time
	// Logf 可选日志钩子。
	Logf func(format string, args ...any)
	// OnLogin 账号通过结构校验后的回调（身份摘要入账号域/落库挂点）。
	OnLogin func(sess *Session, identity AccountIdentity)

	now func() time.Time
}

// Handle 处理一帧心跳外的客户端帧，应答直接写回会话连接。
// 非法登录结构拒绝整条连接（旧服同口径：reject client message）。
func (f *Flow) Handle(sess *Session, frame []byte) {
	if len(frame) == 0 {
		return
	}
	switch frame[0] {
	case OpClientLogin:
		identity, err := ParseLoginAccountIdentity(frame)
		if err != nil {
			f.logf("flow: 会话 %d 拒绝登录结构: %v", sess.ID, err)
			sess.Conn.Close() // 拒绝即断连
			return
		}
		if f.OnLogin != nil {
			f.OnLogin(sess, identity)
		}
		if err := sess.Conn.WriteMsg(noRoleLoginSuccessFrame(f.clock())); err != nil {
			return
		}
		f.logf("flow: 会话 %d 账号 %s 登录（无角色应答）", sess.ID, identity.Prefix())
	case OpClientWorldReq:
		if err := sess.Conn.WriteMsg(worldInfoFrame(0, openBooks)); err != nil {
			return
		}
		f.logf("flow: 会话 %d worldInfo（五书开放）", sess.ID)
	default:
		f.logf("flow: 会话 %d 登录前未处理帧 op=%02X len=%d", sess.ID, frame[0], len(frame))
	}
}

func (f *Flow) clock() time.Time {
	if f.Clock != nil {
		return f.Clock()
	}
	return time.Now()
}

func (f *Flow) logf(format string, args ...any) {
	if f.Logf != nil {
		f.Logf(format, args...)
	}
}
