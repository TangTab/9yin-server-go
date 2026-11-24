package login

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/internal/net"
)

// capturedIdentity 首飞 golden（login-connect-001.bin）的账号身份摘要，
// 旧分支同源 golden 钉死：同密文同键、跨会话稳定。
const capturedIdentity = "acct:v1:112cd36f1e1d2bac408e0428aabef6e399080c2bc33407de6677e60dec97c2a6"

// capturedFirstFlight 读取 golden 线路帧（XOR+EE 编码态）并按线路层解码：
// unescape → XOR(InitialKey)，得 121B 明文 op02。
func capturedFirstFlight(t *testing.T) []byte {
	t.Helper()
	raw, err := res.ReadFile("../../testdata/captures/login-connect-001.bin")
	if err != nil {
		t.Fatal(err)
	}
	framer, err := net.NewEscapeFramer(bytes.NewBuffer(raw), 4096)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := framer.ReadFrame()
	if err != nil {
		t.Fatal(err)
	}
	return net.NewXorCipher(net.InitialKey).Decode(wire)
}

func TestCapturedLoginIdentity(t *testing.T) {
	plain := capturedFirstFlight(t)
	if len(plain) != 121 {
		t.Fatalf("首飞明文长度 %d want 121", len(plain))
	}
	if plain[0] != OpClientLogin {
		t.Fatalf("首飞 opcode %02X want 02", plain[0])
	}
	identity, err := ParseLoginAccountIdentity(plain)
	if err != nil {
		t.Fatal(err)
	}
	if got := identity.String(); got != capturedIdentity {
		t.Fatalf("身份 %s want %s", got, capturedIdentity)
	}
}

// buildLoginFrame 按 auth.go 钉死的 op02 布局造合法帧（官方走位：
// 头 25B → 账号槽 → 密码槽 → client-type u32 → login-string 槽 →
// login-type u32 恰收尾；版本号可注入——版本门测试用）。
func buildLoginFrame(t *testing.T, version uint32, acct, pw, loginStr []byte) []byte {
	t.Helper()
	msg := make([]byte, 0, 25+4+len(acct)+4+len(pw)+4+4+len(loginStr)+4)
	msg = append(msg, OpClientLogin)
	var u32 [4]byte
	put := func(v uint32) { binary.LittleEndian.PutUint32(u32[:], v); msg = append(msg, u32[:]...) }
	put(version) // [0x01:0x05] 版本号（golden=333）
	msg = append(msg, make([]byte, 16)...)
	put(uint32(len(acct)))
	msg = append(msg, acct...)
	put(uint32(len(pw)))
	msg = append(msg, pw...)
	put(1) // client-type（u32 值，不占槽）
	put(uint32(len(loginStr)))
	msg = append(msg, loginStr...)
	put(1) // login-type —— 必须恰好收尾
	return msg
}

// TestLoginSlotBoundOfficial 官方 0x40 上限实证（FsSoloEntry 登录走位）：
// 账号/密码槽超 64 字节应拒绝；login-string 槽维持旧界 512。
func TestLoginSlotBoundOfficial(t *testing.T) {
	acct := make([]byte, 80) // 5×16，%16 合法但超 0x40
	if _, _, err := ParseLoginCredentials(buildLoginFrame(t, 333, acct, make([]byte, 16), make([]byte, 16))); err == nil {
		t.Fatal("账号槽 80B 超 0x40 应拒绝")
	}
	pw := make([]byte, 80)
	if _, _, err := ParseLoginCredentials(buildLoginFrame(t, 333, make([]byte, 16), pw, make([]byte, 16))); err == nil {
		t.Fatal("密码槽 80B 超 0x40 应拒绝")
	}
	// 64B 边界值应通过（官方 jbe=含边界）。
	if _, _, err := ParseLoginCredentials(buildLoginFrame(t, 333, make([]byte, 64), make([]byte, 64), make([]byte, 16))); err != nil {
		t.Fatalf("双槽恰 64B 不应拒绝：%v", err)
	}
	// login-string 384B ≤512：应通过（官方未检该槽上限）。
	long := make([]byte, 384)
	if _, _, err := ParseLoginCredentials(buildLoginFrame(t, 333, make([]byte, 16), make([]byte, 16), long)); err != nil {
		t.Fatalf("login-string 384B 不应拒绝：%v", err)
	}
}

// TestVersionGateOverGateway 版本门端到端（官方方言 {0x03, u32 0x4E85}）：
// 帧版本不符回 5B 应答后断连；相符则照常 0x04。
func TestVersionGateOverGateway(t *testing.T) {
	flight := capturedFirstFlight(t)
	bad := append([]byte(nil), flight...)
	binary.LittleEndian.PutUint32(bad[LoginVersionOffset:LoginVersionOffset+4], 999)

	fixed := time.Date(2026, 11, 19, 0, 0, 0, 0, time.Local)
	flow := &Flow{
		ClientVersion: 333,
		Clock:         func() time.Time { return fixed },
	}
	g := &Gateway{Addr: "127.0.0.1:0", OnFrame: flow.Handle}
	addr, err := g.Listen()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { g.Close() })
	go g.Serve()

	// 不符版本：应答 5B 官方版本帧并断连
	conn, err := net.Dial(addr.String(), net.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMsg(bad); err != nil {
		t.Fatal(err)
	}
	reply, err := conn.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if want := versionMismatchFrame(); !bytes.Equal(reply, want) {
		t.Fatalf("版本门应答 %x want %x", reply, want)
	}
	if _, err := conn.ReadMsg(); err == nil {
		t.Fatal("版本不符应断连")
	}

	// 相符版本（golden 原值 333）：照常无角色 0x04
	conn2, err := net.Dial(addr.String(), net.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close()
	if err := conn2.WriteMsg(flight); err != nil {
		t.Fatal(err)
	}
	reply2, err := conn2.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if want := noRoleLoginSuccessFrame(fixed); !bytes.Equal(reply2, want) {
		t.Fatalf("版本相符应回 0x04：got %d 字节", len(reply2))
	}
}

// TestParseLoginCredentialsSecret golden 首飞的密码摘要：md5hex-32、
// 同帧同摘要（确定性），且与账号身份互相独立。
func TestParseLoginCredentialsSecret(t *testing.T) {
	plain := capturedFirstFlight(t)
	identity, secret, err := ParseLoginCredentials(plain)
	if err != nil {
		t.Fatal(err)
	}
	if identity.String() != capturedIdentity {
		t.Fatalf("身份 %s want %s", identity.String(), capturedIdentity)
	}
	if len(secret) != 32 {
		t.Fatalf("密码摘要长度 %d want 32（md5hex）", len(secret))
	}
	for _, c := range secret {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("密码摘要含非 hex 字符 %q", c)
		}
	}
	if _, again, err := ParseLoginCredentials(plain); err != nil || again != secret {
		t.Fatalf("同帧密码摘要不稳定：%q vs %q（err=%v）", secret, again, err)
	}
}

// TestPasswordMutationChangesSecret 密码槽变化必须换摘要
// （凭据比对依赖：同明文同摘要、异明文异摘要）。
func TestPasswordMutationChangesSecret(t *testing.T) {
	plain := capturedFirstFlight(t)
	acctLen := int(binary.LittleEndian.Uint32(plain[0x15:0x19]))
	pwLenOff := 0x19 + acctLen
	pwLen := int(binary.LittleEndian.Uint32(plain[pwLenOff : pwLenOff+4]))
	pwStart := pwLenOff + 4

	mutated := append([]byte(nil), plain...)
	for i := pwStart; i < pwStart+pwLen; i++ {
		mutated[i] ^= 0x5A
	}
	_, first, err := ParseLoginCredentials(plain)
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := ParseLoginCredentials(mutated)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("密码槽变更应改变摘要")
	}
}

// TestPasswordMutationStableIdentity 密码槽变化不改变账号身份
// （身份只摘要 account 槽——多端改密不改键的方言基础）。
func TestPasswordMutationStableIdentity(t *testing.T) {
	plain := capturedFirstFlight(t)
	acctLen := int(binary.LittleEndian.Uint32(plain[0x15:0x19]))
	pwLenOff := 0x19 + acctLen
	pwLen := int(binary.LittleEndian.Uint32(plain[pwLenOff : pwLenOff+4]))
	pwStart := pwLenOff + 4

	mutated := append([]byte(nil), plain...)
	for i := pwStart; i < pwStart+pwLen; i++ {
		mutated[i] ^= 0x5A
	}
	first, err := ParseLoginAccountIdentity(plain)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ParseLoginAccountIdentity(mutated)
	if err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() {
		t.Fatal("密码槽变更不应改变身份")
	}
}

func TestParseLoginRejectsBadStructure(t *testing.T) {
	plain := capturedFirstFlight(t)
	if _, err := ParseLoginAccountIdentity(append(plain, 0xAA)); err == nil {
		t.Fatal("尾部多余字节应拒绝")
	}
	if _, err := ParseLoginAccountIdentity(plain[:100]); err == nil {
		t.Fatal("截断帧应拒绝")
	}
	bad := append([]byte(nil), plain...)
	binary.LittleEndian.PutUint32(bad[0x15:0x19], 15) // 非 %16
	if _, err := ParseLoginAccountIdentity(bad); err == nil {
		t.Fatal("非法槽长应拒绝")
	}
}

func TestNoRoleLoginSuccessLayout(t *testing.T) {
	now := time.Date(2026, 11, 18, 20, 30, 15, 0, time.Local)
	msg := noRoleLoginSuccessFrame(now)
	if len(msg) != 0x29 {
		t.Fatalf("长度 %d want 41", len(msg))
	}
	if msg[0] != OpServerPlayerRoles {
		t.Fatalf("opcode %02X", msg[0])
	}
	u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(msg[off:]) }
	for off, want := range map[int]uint32{
		0x01: 1, 0x09: 2026, 0x0D: 11, 0x11: 18, 0x15: 20, 0x19: 30, 0x1D: 15,
	} {
		if got := u32(off); got != want {
			t.Fatalf("偏移 %#x=%d want %d", off, got, want)
		}
	}
}

func TestWorldInfoFrameLayout(t *testing.T) {
	msg := worldInfoFrame(0, "AB")
	// 0x05 + u16 0 + 'A' 00 'B' 00 + NUL 00 00
	want := []byte{0x05, 0, 0, 'A', 0, 'B', 0, 0, 0}
	if !bytes.Equal(msg, want) {
		t.Fatalf("worldInfo %x want %x", msg, want)
	}
}

// TestFlowLoginOverGateway 端到端：golden 首飞明文经真 TCP 网关，
// Flow 应回 41B 无角色 0x04；随后 op03 应回五书 worldInfo。
func TestFlowLoginOverGateway(t *testing.T) {
	flight := capturedFirstFlight(t)
	fixed := time.Date(2026, 11, 18, 20, 30, 15, 0, time.Local)
	flow := &Flow{Clock: func() time.Time { return fixed }}

	g := &Gateway{Addr: "127.0.0.1:0", OnFrame: flow.Handle}
	addr, err := g.Listen()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { g.Close() })
	go g.Serve()

	conn, err := net.Dial(addr.String(), net.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err := conn.WriteMsg(flight); err != nil {
		t.Fatal(err)
	}
	reply, err := conn.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if want := noRoleLoginSuccessFrame(fixed); !bytes.Equal(reply, want) {
		t.Fatalf("0x04 应答 %x want %x", reply, want)
	}

	if err := conn.WriteMsg([]byte{OpClientWorldReq}); err != nil {
		t.Fatal(err)
	}
	info, err := conn.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if want := worldInfoFrame(0, openBooks); !bytes.Equal(info, want) {
		t.Fatalf("worldInfo 应答长度 %d want %d", len(info), len(want))
	}
}
