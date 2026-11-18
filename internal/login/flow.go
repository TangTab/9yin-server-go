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
	OpClientRoleSel  byte = 0x04 // 选角（重连带会话恢复字段）→ 0x0B 进场
	OpClientRoleNew  byte = 0x05 // 建角
	OpClientReady    byte = 0x09 // ClientReady（进场后就绪，视图/属性簇接入点）
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

// versionMismatchFrame 生成官方版本门不符应答（5B）：{0x03, LE u32
// 0x4E85=20101}（BGserverCJ CheckMessageIllegal 反汇编逐字节实证）。
func versionMismatchFrame() []byte {
	msg := make([]byte, 5)
	msg[0] = 0x03
	binary.LittleEndian.PutUint32(msg[1:], 0x4E85)
	return msg
}

// Flow 是登录链分派器：Gateway.OnFrame 的第一接入点。
// 当前覆盖：op02 账号登录（有/无角色 0x04）、op03 世界信息（五书）、
// op05 建角（回有角色 0x04）。选角 op04 与进场景随主干推进扩展。
type Flow struct {
	// Clock 时间源（应答帧日期字段；测试可注入）。
	Clock func() time.Time
	// Logf 可选日志钩子。
	Logf func(format string, args ...any)
	// Roles 角色仓（nil=Flow 内建内存仓；persist 接 DB 时替换）。
	Roles *RoleStore
	// Accounts 账号仓（nil=不校验凭据；persist 接 MySQL dc_accounts
	// 时注入——首登自动注册、密码摘要不符拒绝）。
	Accounts AccountStore
	// ClientVersion 版本门（0=不校验；>0 时 op02 帧 [1:5] u32 须相等，
	// 不符回官方 {0x03, u32 0x4E85} 并断连。取值见 GameEntryConfig）。
	ClientVersion int
	// OnLogin 账号通过结构校验后的回调（身份摘要入账号域/落库挂点）。
	OnLogin func(sess *Session, identity AccountIdentity)
	// OnEnterScene 选角进场回调（0x0B 已发出；ECS 物化/总线事件挂点）。
	OnEnterScene func(sess *Session, role *Role)

	defaultRoles *RoleStore
}

func (f *Flow) roles() *RoleStore {
	if f.Roles == nil {
		if f.defaultRoles == nil {
			f.defaultRoles = NewRoleStore()
		}
		return f.defaultRoles
	}
	return f.Roles
}

// Handle 处理一帧心跳外的客户端帧，应答直接写回会话连接。
// 非法登录结构拒绝整条连接（旧服同口径：reject client message）。
func (f *Flow) Handle(sess *Session, frame []byte) {
	if len(frame) == 0 {
		return
	}
	switch frame[0] {
	case OpClientLogin:
		identity, secret, err := ParseLoginCredentials(frame)
		if err != nil {
			f.logf("flow: 会话 %d 拒绝登录结构: %v", sess.ID, err)
			sess.Conn.Close() // 拒绝即断连
			return
		}
		if f.ClientVersion > 0 {
			client := binary.LittleEndian.Uint32(frame[LoginVersionOffset : LoginVersionOffset+4])
			if client != uint32(f.ClientVersion) {
				if err := sess.Conn.WriteMsg(versionMismatchFrame()); err != nil {
					return
				}
				sess.Conn.Close()
				f.logf("flow: 会话 %d 版本不符：%d want %d，回 0x4E85 断连", sess.ID, client, f.ClientVersion)
				return
			}
		}
		registered := false
		if f.Accounts != nil {
			if registered, err = f.Accounts.Verify(identity, secret); err != nil {
				f.logf("flow: 会话 %d 账号 %s 凭据不符: %v", sess.ID, identity.Prefix(), err)
				sess.Conn.Close() // 密码不符/存储故障同口径：拒绝即断连
				return
			}
		}
		if f.OnLogin != nil {
			f.OnLogin(sess, identity)
		}
		sess.Account = identity.String()
		sess.Role = f.roles().Get(sess.Account)
		var reply []byte
		if sess.Role != nil {
			reply = oneRoleLoginSuccessFrame(f.clock(), sess.Role)
		} else {
			reply = noRoleLoginSuccessFrame(f.clock())
		}
		if err := sess.Conn.WriteMsg(reply); err != nil {
			return
		}
		f.logf("flow: 会话 %d 账号 %s 登录（角色=%t 注册=%t）",
			sess.ID, identity.Prefix(), sess.Role != nil, registered)
	case OpClientWorldReq:
		if err := sess.Conn.WriteMsg(worldInfoFrame(0, openBooks)); err != nil {
			return
		}
		f.logf("flow: 会话 %d worldInfo（五书开放）", sess.ID)
	case OpClientRoleNew:
		if sess.Account == "" {
			f.logf("flow: 会话 %d 未登录建角，忽略", sess.ID)
			return
		}
		name, appearance := parseCreateRole(frame)
		if name == "" {
			name = "本地角色" // 旧分支同款回退
		}
		sess.Role = f.roles().Create(sess.Account, name, appearance)
		if err := sess.Conn.WriteMsg(oneRoleLoginSuccessFrame(f.clock(), sess.Role)); err != nil {
			return
		}
		f.logf("flow: 会话 %d 建角 %q（外观串 %d 个）", sess.ID, name, len(appearance))
	case OpClientRoleSel:
		if sess.Role == nil {
			f.logf("flow: 会话 %d 无角色却选角，忽略", sess.ID)
			return
		}
		if err := sess.Conn.WriteMsg(entrySceneFrame(sess.Role.Scene)); err != nil {
			return
		}
		sess.EnterCount++
		if !sess.Entered {
			sess.Entered = true
			if f.OnEnterScene != nil {
				f.OnEnterScene(sess, sess.Role)
			}
		} // 已进场重复 op04：0x0B 幂等重发，不再重复物化（真机重试实测）
		f.logf("flow: 会话 %d 选角 %q 进场 %s（第 %d 次）",
			sess.ID, sess.Role.Name, sess.Role.Scene.Resource, sess.EnterCount)
	case OpClientReady:
		if !sess.Entered {
			f.logf("flow: 会话 %d 未进场却 ClientReady，忽略", sess.ID)
			return
		}
		// 就绪后旧服开始技能/视图大簇 + 玩家全量属性集——
		// 0x09 属性表/0x0A 记录表下发是前置，属性全量集环接入。
		f.logf("flow: 会话 %d ClientReady（视图/属性簇待接入）", sess.ID)
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
