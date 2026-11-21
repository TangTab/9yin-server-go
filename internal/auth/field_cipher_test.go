// fork oracle 向量测试：全部期望值取自 fork exe（9yin-game-native-menu.exe）
// 实际运行 /api/register 落库的账号键与口令校验子，以及其指令级仿真器
// （逆向期中间产物）的派生密码/密文输出。五组覆盖：1 块/2 块账号、奇偶块
// 掩码、多长度密码（1..19 字符）。
package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

type forkVector struct {
	account         string
	password        string
	derivedPassword string
	acctCiphertext  string // hex
	pwdCiphertext   string // hex
	accountKey      string
	verifierB64     string
}

var forkVectors = []forkVector{
	{
		account: "porttest01", password: "Zk9#verify",
		derivedPassword: "RoNDltbEIvQ1E9PQ==8VMwVjdMMnFQRE8rd1",
		acctCiphertext:  "943a918ed52b34519b8a6184e0a5a7cd",
		pwdCiphertext:   "52d8b7048bcd54878984503cbba3ea3dd6c1aad82bc35956a73c46273cc7504cddb609936eadb36df8934da42df74df3",
		accountKey:      "acct:v1:e8637327ad9d765cac8294034a8488620b576afabda4b8634457b043e378efb4",
		verifierB64:     "Uti3BIvNVIeJhFA8u6PqPdbBqtgrw1lWpzxGJzzHUEzdtgmTbq2zbfiTTaQt903z",
	},
	{
		account: "ABCDEFGHIJKLMNOP", password: "x",
		derivedPassword: "lKNmNuSHQ1R0E9PQ==vzZ4cHJoUy9tSXN2MV",
		acctCiphertext:  "04165b352fa99bc89fbbd73964f6b80e",
		pwdCiphertext:   "436dfb093b009268ee10179ddae1258a00ba3d3377b247e793b68c6499fdb7fc7aaa56f279ac1b86c6a569ee5d88df17",
		accountKey:      "acct:v1:de63ce1693bdf15f5c0bff85bf2dd248f3ff2b5655cd6c3be5455127405501f1",
		verifierB64:     "Q237CTsAkmjuEBed2uEligC6PTN3skfnk7aMZJn9t/x6qlbyeawbhsalae5diN8X",
	},
	{
		account: "a", password: "a",
		derivedPassword: "ZYdHRDOU1yZ3c9PQ==x1RDZjNOd0Mwc0JhcU",
		acctCiphertext:  "823c7e44d44f82c072cfc2ce82c640b8",
		pwdCiphertext:   "d219a61618942ec93c00a2c7aa425390218f1c4bb1fe37d09108bbc0dbfa399d3299060b07cfc3459522d84af3c3b33d",
		accountKey:      "acct:v1:667053f6928c31a4352e5a55bef0978710e803b320e1054abc982b4d397a0250",
		verifierB64:     "0hmmFhiULsk8AKLHqkJTkCGPHEux/jfQkQi7wNv6OZ0ymQYLB8/DRZUi2Erzw7M9",
	},
	{
		account: "twenty-char-acct-x", password: "LongerPassword!2026",
		derivedPassword: "1PaWdYV3kzbTRKUDEvQ05yVWVxQmc9WDhGYWpzdjEvNm9iVlVSdFB3OUZGcG",
		acctCiphertext:  "cfebe3f14e87453be8e2e904785b32277e17ea7121085a078d67f7be02e133ed",
		pwdCiphertext:   "61534830b896d7578563d491df7a4f92bb0dd2c4a314e76eec40a18642f557ead7f9ed70f2d2e6eb03d18af3a311263396599942c6a576998d71b849c088debd",
		accountKey:      "acct:v1:a50ed58b1904cd855de2c6bd1c46247fc504fdb8944606e49f95a2df4d124ff3",
		verifierB64:     "YVNIMLiW11eFY9SR33pPkrsN0sSjFOdu7EChhkL1V+rX+e1w8tLm6wPRivOjESYzllmZQsaldpmNcbhJwIjevQ==",
	},
	{
		account: "blk0000000000004", password: "p",
		derivedPassword: "FLaDl1S2gvZkE9PQ==r3ZHNzBOemNMZmJDbz",
		acctCiphertext:  "6cdba9bd1cc03afaf4107c8977ed8b2a",
		pwdCiphertext:   "10aafc447b6c1d55915e272a01ed80247e32bfda71926e7376f94248bb278881a942ede6af3ab5558345dd6825ca6485",
		accountKey:      "acct:v1:e33b671bead3476a0f11c62ff1e55ce89eb2e44fed44c54e5add2c564878035a",
		verifierB64:     "EKr8RHtsHVWRXicqAe2AJH4yv9pxkm5zdvlCSLsniIGpQu3mrzq1VYNF3WglymSF",
	},
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	raw, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex vector %q: %v", s, err)
	}
	return raw
}

func TestForkOracleVectors(t *testing.T) {
	for _, vec := range forkVectors {
		acctCt, pwdCt := EncryptFields(vec.account, ComputeLoginPassword(vec.password))
		if got := mustHex(t, vec.acctCiphertext); !bytes.Equal(acctCt, got) {
			t.Errorf("account %q: ct1 mismatch\n got  %x\n want %x", vec.account, acctCt, got)
		}
		if want := mustHex(t, vec.pwdCiphertext); !bytes.Equal(pwdCt, want) {
			t.Errorf("account %q: ct2 mismatch\n got  %x\n want %x", vec.account, pwdCt, want)
		}
		if derived := ComputeLoginPassword(vec.password); derived != vec.derivedPassword {
			t.Errorf("account %q: derived mismatch\n got  %q\n want %q", vec.account, derived, vec.derivedPassword)
		}
		key, err := AccountKeyFor(vec.account)
		if err != nil {
			t.Fatalf("account %q: AccountKeyFor: %v", vec.account, err)
		}
		if key != vec.accountKey {
			t.Errorf("account %q: key mismatch\n got  %s\n want %s", vec.account, key, vec.accountKey)
		}
		verifier := ComputePasswordVerifier(vec.account, vec.password)
		wantVerifier, err := base64.StdEncoding.DecodeString(vec.verifierB64)
		if err != nil {
			t.Fatalf("bad verifier vector: %v", err)
		}
		if !bytes.Equal(verifier, wantVerifier) {
			t.Errorf("account %q: verifier mismatch\n got  %x\n want %x", vec.account, verifier, wantVerifier)
		}
	}
}

// TestEncryptFieldsDeterministic 确定性：同输入恒同输出（登录比对的根基）。
func TestEncryptFieldsDeterministic(t *testing.T) {
	a1, p1 := EncryptFields("determinism", "check")
	a2, p2 := EncryptFields("determinism", "check")
	if !bytes.Equal(a1, a2) || !bytes.Equal(p1, p2) {
		t.Fatal("EncryptFields not deterministic")
	}
}

// TestAccountKeyForValidation 空名/超长名拒绝。
func TestAccountKeyForValidation(t *testing.T) {
	if _, err := AccountKeyFor(""); err == nil {
		t.Fatal("empty name accepted")
	}
	if _, err := AccountKeyFor(string(make([]byte, 257))); err == nil {
		t.Fatal("oversized name accepted")
	}
}

// TestForkOracleDecryptVectors 逆阶梯解密：oracle 密文必须解回明文账号与
// 派生登录密码（field1/field2 双域、1/2 块路径全覆盖）。
func TestForkOracleDecryptVectors(t *testing.T) {
	for _, vec := range forkVectors {
		account, derived := DecryptAccountFields(mustHex(t, vec.acctCiphertext), mustHex(t, vec.pwdCiphertext))
		if account != vec.account {
			t.Errorf("decrypt account: got %q want %q", account, vec.account)
		}
		if derived != vec.derivedPassword {
			t.Errorf("decrypt derived (%s): got %q want %q", vec.account, derived, vec.derivedPassword)
		}
	}
}

// TestDecryptAccountFieldsRejectsNonASCII 非 ASCII（如中文账号名）回空——
// m655/m607 tag6 窄串只放 ASCII（官方方言，曾致账号栏乱码）。
func TestDecryptAccountFieldsRejectsNonASCII(t *testing.T) {
	acctCt, pwdCt := EncryptFields("中文账号", "pw")
	account, _ := DecryptAccountFields(acctCt, pwdCt)
	if account != "" {
		t.Fatalf("non-ASCII account not filtered: %q", account)
	}
}
