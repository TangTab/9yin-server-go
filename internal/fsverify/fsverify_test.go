package fsverify

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// capturedFirstFlight 读取 golden 线路帧并按线路层解码（与
// internal/login/auth_test.go 同源：EE 转义 → XOR InitialKey）。
func capturedFirstFlight(t *testing.T) []byte {
	t.Helper()
	raw, err := res.ReadFile("../../testdata/captures/login-connect-001.bin")
	if err != nil {
		t.Fatal(err)
	}
	var body []byte
	for i := 0; i < len(raw); {
		if raw[i] == 0xEE && i+1 < len(raw) {
			if raw[i+1] == 0x00 {
				body = append(body, 0xEE)
				i += 2
				continue
			}
			if raw[i+1] == 0xEE {
				break
			}
			t.Fatal("bad escape")
		}
		body = append(body, raw[i])
		i++
	}
	key := binary.LittleEndian.Uint32([]byte{0xCC, 0x3B, 0x6D, 0xFA})
	plain := make([]byte, len(body))
	for i, b := range body {
		plain[i] = b ^ byte(key>>(8*uint(i%4)))
	}
	return plain
}

func loginSlots(t *testing.T, plain []byte) (account, password []byte) {
	t.Helper()
	if len(plain) < 25 || plain[0] != 0x02 {
		t.Fatalf("not an op02 frame")
	}
	acctLen := binary.LittleEndian.Uint32(plain[0x15:0x19])
	account = plain[0x19 : 0x19+acctLen]
	cur := 0x19 + acctLen
	pwLen := binary.LittleEndian.Uint32(plain[cur : cur+4])
	password = plain[cur+4 : cur+4+pwLen]
	return
}

// TestGoldenLoginSlots 官方 golden 线路帧双槽解密：
// 账号槽 → test001；密码槽 → 派生形态（三方实证钉死的期望值）。
func TestGoldenLoginSlots(t *testing.T) {
	plain := capturedFirstFlight(t)
	acct, pw := loginSlots(t, plain)
	if got := DecodeAccount(acct); got != "test001" {
		t.Fatalf("account = %q want test001", got)
	}
	want := "drUGtwMnNPbVE9PQ==VjYzaVZab0xjQ0M3bD"
	if got := DecodePassword(pw); got != want {
		t.Fatalf("password = %q want %q", got, want)
	}
}

// TestSlotRoundTrip 加密→解密回环（两域、多变长）。
func TestSlotRoundTrip(t *testing.T) {
	cases := []struct{ account, derived string }{
		{"test001", "drUGtwMnNPbVE9PQ==VjYzaVZab0xjQ0M3bD"},
		{"a", "x"},
		{"1234567890123456", "0123456789abcdef0123456789abcdef"},
		{"long-account-name-over-32-bytes-xxxx", "p"},
	}
	for _, c := range cases {
		acctCT, pwCT := EncodeAccountFields(c.account, c.derived)
		if got := DecodeAccount(acctCT); got != c.account {
			t.Fatalf("account round trip: %q want %q", got, c.account)
		}
		if got := DecodePassword(pwCT); got != c.derived {
			t.Fatalf("password round trip: %q want %q", got, c.derived)
		}
	}
}

// TestGoldenCiphertextLock 锁定 golden 密文本身（键系回归哨兵）。
func TestGoldenCiphertextLock(t *testing.T) {
	plain := capturedFirstFlight(t)
	acct, _ := loginSlots(t, plain)
	want := "4578627e7a26ccc46b9f5bf48d6dca47"
	if got := hex.EncodeToString(acct); got != want {
		t.Fatalf("golden account ct = %s want %s", got, want)
	}
	ct, _ := EncodeAccountFields("test001", "")
	if hex.EncodeToString(ct) != want {
		t.Fatalf("re-encrypt mismatch: %s want %s", hex.EncodeToString(ct), want)
	}
}

// TestEncodeVectors 全量 2048 向量（tools/fsverify_gen.py 以 unicorn oracle
// 跑官方 DLL 产出的 golden）。
func TestEncodeVectors(t *testing.T) {
	raw, err := res.ReadFile("../../testdata/fsverify/encode-vectors.txt")
	if err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	n := 0
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 {
			t.Fatalf("bad vector line: %q", scanner.Text())
		}
		id := 0
		for _, c := range fields[0] {
			id = id*10 + int(c-'0')
		}
		in, err1 := hex.DecodeString(fields[1])
		want, err2 := hex.DecodeString(fields[2])
		if err1 != nil || err2 != nil || len(in) != 16 || len(want) != 16 {
			t.Fatalf("bad vector %d", id)
		}
		got, err := EncodeFunc(id, in)
		if err != nil {
			t.Fatalf("EncodeFunc(%d): %v", id, err)
		}
		if !bytes.Equal(got[:], want) {
			t.Fatalf("EncodeFunc(%d) = %x want %x", id, got, want)
		}
		n++
	}
	if n != EncodeNum {
		t.Fatalf("vector count %d want %d", n, EncodeNum)
	}
}

// TestTables 三表抽查（提取器输出的已知值）。
func TestTables(t *testing.T) {
	if id, _ := EncodeID(0); id != 0x66f {
		t.Fatalf("EncodeID(0) = %#x want 0x66f", id)
	}
	cond, _ := ClientCond(0)
	if want := "53d6f9ebbdb3d650de6c75130583890c"; hex.EncodeToString(cond) != want {
		t.Fatalf("ClientCond(0) = %s want %s", hex.EncodeToString(cond), want)
	}
	code, _ := ClientCode(0)
	if want := "340100005c440b008000000002000000"; hex.EncodeToString(code) != want {
		t.Fatalf("ClientCode(0) = %s want %s", hex.EncodeToString(code), want)
	}
	if _, ok := ClientCond(EncodeNum); ok {
		t.Fatal("ClientCond(2048) should be out of range")
	}
	if _, ok := EncodeID(EncodeNum); ok {
		t.Fatal("EncodeID(2048) should be out of range")
	}
}

// TestVerifyClientToken 令牌验证（当前命中/回退 prev/全失配）。
func TestVerifyClientToken(t *testing.T) {
	src := bytes.Repeat([]byte{7}, 16)
	cur, _ := EncodeFunc(100, src)
	prev, _ := EncodeFunc(101, src)

	if !VerifyClientToken(100, 101, true, src, cur[:]) {
		t.Fatal("current id should verify")
	}
	if !VerifyClientToken(100, 101, true, src, prev[:]) {
		t.Fatal("prev id fallback should verify")
	}
	if VerifyClientToken(100, 101, false, src, prev[:]) {
		t.Fatal("prev fallback must be gated by prevValid")
	}
	if VerifyClientToken(100, 101, true, src, make([]byte, 16)) {
		t.Fatal("wrong token must fail")
	}
}

// TestModuleConstants 版本/数量常量锁。
func TestModuleConstants(t *testing.T) {
	if ModuleVersion != 300 {
		t.Fatalf("ModuleVersion = %d want 300", ModuleVersion)
	}
	if EncodeNum != 2048 {
		t.Fatalf("EncodeNum = %d want 2048", EncodeNum)
	}
}
