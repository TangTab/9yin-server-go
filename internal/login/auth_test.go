package login

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/net"
)

// capturedIdentity 首飞 golden（login-connect-001.bin）的账号身份摘要，
// 旧分支同源 golden 钉死：同密文同键、跨会话稳定。
const capturedIdentity = "acct:v1:112cd36f1e1d2bac408e0428aabef6e399080c2bc33407de6677e60dec97c2a6"

// capturedFirstFlight 读取 golden 线路帧（XOR+EE 编码态）并按线路层解码：
// unescape → XOR(InitialKey)，得 121B 明文 op02。
func capturedFirstFlight(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("../../testdata/captures/login-connect-001.bin")
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
