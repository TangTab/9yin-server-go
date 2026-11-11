package main

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/local/9yin-go-server/internal/auth"
	"github.com/local/9yin-go-server/internal/role"
)

// craftLoginFrame 按客户端 op02 方言构造登录帧：
// [0]=0x02 + 保留头 + [0x15:0x19]=LE32 账号槽长 + 账号槽密文 +
// u32 密码槽长 + 密码槽密文 + clientType 槽 + loginString 槽（≥16、%16）+
// u32 loginType（帧到此为止，尾部多余字节会被身份解析拒绝）。
func craftLoginFrame(t *testing.T, account, password string) []byte {
	t.Helper()
	derived := auth.ComputeLoginPassword(password)
	accountCiphertext, passwordCiphertext := auth.EncryptFields(account, derived)

	frame := make([]byte, 0, 25+len(accountCiphertext)+4+len(passwordCiphertext)+32)
	frame = append(frame, 0x02)
	frame = append(frame, make([]byte, 20)...) // 保留头（1..0x14），[0x15:0x19] 为账号槽长
	var slotLength [4]byte
	binary.LittleEndian.PutUint32(slotLength[:], uint32(len(accountCiphertext)))
	frame = append(frame, slotLength[:]...)
	frame = append(frame, accountCiphertext...)
	binary.LittleEndian.PutUint32(slotLength[:], uint32(len(passwordCiphertext)))
	frame = append(frame, slotLength[:]...)
	frame = append(frame, passwordCiphertext...)
	// clientType：u32 长 + 内容（0 长度合法）。
	frame = append(frame, 0, 0, 0, 0)
	// loginString：必须 ≥16 且 16 整倍。
	binary.LittleEndian.PutUint32(slotLength[:], 16)
	frame = append(frame, slotLength[:]...)
	frame = append(frame, make([]byte, 16)...)
	// loginType u32。
	frame = append(frame, 1, 0, 0, 0)
	return frame
}

// TestClientStyleLoginFrameRoundTrip 客户端形态的 op02 帧经真实登录链
// （身份解析→凭据槽→store.login 验密）全通：证明注册侧与登录侧键/密文
// 同源对齐，且未注册存量账号仍免密放行。
func TestClientStyleLoginFrameRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := testRoleStore(t)

	const account, password = "frametest01", "frame-pass"
	frame := craftLoginFrame(t, account, password)

	identity, err := auth.ParseLoginAccountIdentity(frame)
	if err != nil {
		t.Fatalf("identity parse: %v", err)
	}
	accountCiphertext, passwordCiphertext, err := auth.ParseLoginCredentials(frame)
	if err != nil {
		t.Fatalf("credentials parse: %v", err)
	}

	// 登录侧键与注册侧键一致。
	registeredKey, err := auth.AccountKeyFor(account)
	if err != nil {
		t.Fatal(err)
	}
	if identity.String() != registeredKey {
		t.Fatalf("key mismatch\n login  %s\n reg    %s", identity.String(), registeredKey)
	}

	// 未注册（默认模式）：自动建号免密放行。
	if _, _, err := store.login(ctx, role.AccountKey(identity.String()), passwordCiphertext); err != nil {
		t.Fatalf("legacy login: %v", err)
	}
	_ = accountCiphertext

	// 注册后：同帧重放必须放行（密码槽==校验子），换密码的帧拒绝。
	verifier := auth.ComputePasswordVerifier(account, password)
	if _, err := store.accounts.RegisterAccount(ctx, role.AccountKey(registeredKey), verifier); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.login(ctx, role.AccountKey(identity.String()), passwordCiphertext); err != nil {
		t.Fatalf("registered login with same frame: %v", err)
	}
	wrongFrame := craftLoginFrame(t, account, "other-pass")
	_, wrongCiphertext, err := auth.ParseLoginCredentials(wrongFrame)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.login(ctx, role.AccountKey(identity.String()), wrongCiphertext); !errors.Is(err, errPasswordMismatch) {
		t.Fatalf("wrong password frame: want errPasswordMismatch, got %v", err)
	}
}
