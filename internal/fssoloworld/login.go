package fssoloworld

// 登录链官方通路（store Account 族）：clientLogin 经 internal/auth 解出
// op02 双槽明文（b-line 双域 DES 阶梯移植）→ store 0x36 AccountLogon
// （首登自动建号 nx_district.dc_accounts、密码槽等值、logged 会话门）
// → 回执 0x25 过门后下行 0x04 ServerPlayerRoles（golden 41B 无角形态，
// 布局同 internal/login.noRoleLoginSuccessFrame 真机钉死版；serial@0x21=
// 登录会话 serial，官服实证与 0x0B 的 [1:5] 同值）/ 0x05
// ServerWorldInfo（五书开放串，真机回放值）。断开走 0x37 logoff。

import (
	"encoding/binary"
	"unicode/utf16"

	"github.com/lizhanyin/9yin-server-go/internal/auth"
)

// accountLogonFrame 构造 0x36 请求（官方断言 ==0x98）：
// {client_id@0, serial@4, session@8, name[0x24]@0xC, password[0x24]@0x30,
// addr[0x20]@0x54, 零填至 0x98}。
func accountLogonFrame(clientID, serial, session uint32, name, password, addr string) []byte {
	b := make([]byte, 0x98)
	binary.LittleEndian.PutUint32(b[0:], clientID)
	binary.LittleEndian.PutUint32(b[4:], serial)
	binary.LittleEndian.PutUint32(b[8:], session)
	copy(b[0xC:], name)
	copy(b[0x30:], password)
	copy(b[0x54:], addr)
	return b
}

// sessionFrame 0x38 AccountReset 请求体 {session}。
func sessionFrame(session uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b[0:], session)
	return b
}

// accountLogoffFrame 构造 0x37 请求（官方断言 ==0x7C）。
func accountLogoffFrame(clientID, serial, session uint32, name string) []byte {
	b := make([]byte, 0x7C)
	binary.LittleEndian.PutUint32(b[0:], clientID)
	binary.LittleEndian.PutUint32(b[4:], serial)
	binary.LittleEndian.PutUint32(b[8:], session)
	copy(b[0xC:], name)
	return b
}

// sendAccountLogon clientLogin 主链：op02 帧经 auth 解出明文账号/密码
// 后向全 store 发 0x36。结构非法或解密失败 = 拒绝（断客户端）。
func (s *Server) sendAccountLogon(pl *Player, frame []byte) bool {
	accountCt, passwordCt, err := auth.ParseLoginCredentials(frame)
	if err != nil {
		s.trace("(SoloWorld::PlayerStateUnknown)get account password error")
		return false
	}
	account, password := auth.DecryptAccountFields(accountCt, passwordCt)
	if account == "" {
		s.trace("(SoloWorld::PlayerStateUnknown)account slot not printable")
		return false
	}
	s.mu.Lock()
	session := s.session
	s.mu.Unlock()
	if pl.Account == "" {
		s.players.mu.Lock()
		if pl.Account == "" {
			pl.Account = account
			s.players.byAcc[account] = pl
		}
		s.players.mu.Unlock()
	}
	ok2 := s.inner.SendToAllStore(0, 0x36,
		accountLogonFrame(pl.Conn1, pl.Conn2, session, account, password, ""))
	s.trace("account logon sent(%v) account=%q client=%d", ok2, account, pl.Conn1)
	return ok2
}

// accountLogonReply store 0x25 回执：{client_id, serial, result,
// user_type, points, used_points}。过门 → PlGetting + 下行 0x04（无角
// golden 形态）+ 0x02 角色清单请求；拒 → 断客户端（entryBreakClient）。
func (s *Server) accountLogonReply(p []byte) bool {
	if len(p) < 0x18 {
		return false
	}
	clientID := binary.LittleEndian.Uint32(p[0:])
	serial := binary.LittleEndian.Uint32(p[4:])
	result := binary.LittleEndian.Uint32(p[8:])
	pl := s.players.FindByConnect(clientID, serial)
	s.trace("account logon reply client=%d serial=%d result=%d player=%v", clientID, serial, result, pl != nil)
	if pl == nil {
		return true
	}
	if result == 0 {
		s.trace("account logon denied (client %d)", clientID)
		s.inner.SendToAllEntry(0, entryBreakClient, p[:8])
		return true
	}
	s.players.SetPlayerState(pl, PlGetting)
	s.sendLoginReplyGolden(pl) // golden 方言：外服 op02 应答镜像（会话/场景锚）
	// 账号角色清单请求（0x10 QueryAccount ≥72B；回执 0x0D 经
	// storeRolesList 统一下行 0x04——有角角色列表/无角 41B 基座）。
	// store 未连接（启动竞态/断连）时兜底直发无角 0x04——客户端
	// 不至卡在登录页无应答。
	req := make([]byte, 72)
	binary.LittleEndian.PutUint32(req[0:], clientID)
	binary.LittleEndian.PutUint32(req[4:], serial)
	copy(req[8:], pl.Account)
	if !s.inner.SendToAllStore(0, 0x10, req) {
		s.trace("(SoloWorld::PlayerStateGetting)store absent, no-role fallback client=%d", clientID)
		s.sendRolesFrame(pl, nil)
	}
	return true
}

// accountLogoffReply store 0x26 回执（簿记面）。
func (s *Server) accountLogoffReply(p []byte) bool {
	return true
}

// clientSendObserver 下行帧测试观察缝（nil=无；邮件/切图帧序断言用——
// nil inner 下 SendToAllEntry 静默，帧不可从出口侧观测）。
var clientSendObserver func(pl *Player, frame []byte)

// sendToClient 单客户端下行：entry id2=6（SoloForwSendClient），
// 载荷 {ident, serial, 原始客户端帧}（BuildUpstreamData 布局）。
func (s *Server) sendToClient(pl *Player, frame []byte) {
	s.trace("send to client %d opcode=0x%02X len=%d", pl.Conn1, frame[0], len(frame))
	if clientSendObserver != nil {
		clientSendObserver(pl, frame)
	}
	buf := make([]byte, 8+len(frame))
	binary.LittleEndian.PutUint32(buf[0:], pl.Conn1)
	binary.LittleEndian.PutUint32(buf[4:], pl.Conn2)
	copy(buf[8:], frame)
	s.inner.SendToAllEntry(0, entrySendClient, buf)
}

// worldInfoFrame 0x05 ServerWorldInfo：[0x05][u16 infoType][UTF-16LE 文本
// + NUL]（五书开放串为真机回放值，同 internal/login.openBooks）。
func worldInfoFrame(infoType uint16, text string) []byte {
	units := utf16.Encode([]rune(text))
	msg := make([]byte, 3+(len(units)+1)*2)
	msg[0] = 0x05
	binary.LittleEndian.PutUint16(msg[1:], infoType)
	for i, unit := range units {
		binary.LittleEndian.PutUint16(msg[3+i*2:], unit)
	}
	return msg
}

// openBooks 五书开放世界串（op03 应答载荷，真机回放值）。
const openBooks = "0,book1,0,book2,0,book4,0,book5,0,book8,0,0"
