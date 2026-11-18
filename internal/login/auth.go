package login

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// op02 ClientLogin（账号登录）的结构方言。golden 钉死：
// testdata/captures/login-connect-001.bin（真实客户端首飞线路帧，
// XOR+EE 解码后 121B）：
//
//	[0x00]      opcode 0x02
//	[0x01:0x05] u32（值 333，语义未证）
//	[0x15:0x19] u32 accountLen（16~256，%16==0）
//	[0x19:..]   account 密文槽
//	u32 pwLen（16~512，%16==0）+ 密文槽
//	u32 client-type 值（不占槽）
//	u32 loginStringLen（16~512，%16==0）+ 槽
//	u32 login-type —— 必须恰好收在帧尾（拒绝尾部数据）
//
// 双槽密文（account/password）不做解密——客户端加扰算法未破（fxcli
// 定位见逆向档案），但密文无 nonce、确定性（test001 162 次登录槽恒定），
// 故身份/凭据均落在密文摘要域：同明文同摘要，跨会话稳定。
const (
	loginHeaderSize = 25
	loginAcctMin    = 16
	loginAcctMax    = 256
	loginSecretMin  = 16
	loginSecretMax  = 512

	// loginAccountDomain / accountKeyPrefix 是 acct:v1 键的哈希域与前缀。
	loginAccountDomain = "nineyin/login-account/v1"
	accountKeyPrefix   = "acct:v1:"

	// loginSecretDomain 是密码槽摘要的哈希域；取 md5hex-32 是为对齐
	// dc_accounts.password varchar(32) 列宽——仅做密文域等值比对，
	// 不承载明文密码语义，抗碰撞性非目标。
	loginSecretDomain = "nineyin/login-secret/v1"
)

// AccountIdentity 是登录方账号的身份摘要（同密文同键，跨会话稳定）。
type AccountIdentity struct {
	id string
}

// String 返回完整身份键（acct:v1:<64 hex>）。
func (identity AccountIdentity) String() string { return identity.id }

// Prefix 返回日志安全前缀（acct:v1: + 前 12 hex）。
func (identity AccountIdentity) Prefix() string {
	const prefixLength = len(accountKeyPrefix) + 12
	if len(identity.id) <= prefixLength {
		return identity.id
	}
	return identity.id[:prefixLength]
}

// ParseLoginCredentials 校验 op02 结构并取账号身份与密码摘要。
// secret 是密码槽密文域摘要（md5hex-32）：密文确定性→同明文密码同
// 摘要，供 AccountStore 等值比对（首登注册、改密换摘要）。
// 结构非法即报错（调用方应拒绝该连接）。
func ParseLoginCredentials(message []byte) (AccountIdentity, string, error) {
	if len(message) < loginHeaderSize || message[0] != OpClientLogin {
		return AccountIdentity{}, "", fmt.Errorf("login: missing login header")
	}
	cursor := loginHeaderSize
	accountLength := binary.LittleEndian.Uint32(message[0x15:0x19])
	if err := validateSlotLength("account", accountLength, loginAcctMin, loginAcctMax); err != nil {
		return AccountIdentity{}, "", err
	}
	accountEnd, ok := advance(cursor, accountLength, len(message))
	if !ok {
		return AccountIdentity{}, "", fmt.Errorf("login: truncated account slot")
	}

	digest := sha256.New()
	_, _ = digest.Write([]byte(loginAccountDomain))
	_, _ = digest.Write([]byte{0})
	var encodedLength [4]byte
	binary.LittleEndian.PutUint32(encodedLength[:], accountLength)
	_, _ = digest.Write(encodedLength[:])
	_, _ = digest.Write(message[cursor:accountEnd])
	cursor = accountEnd

	passwordLength, next, ok := readU32(message, cursor)
	if !ok {
		return AccountIdentity{}, "", fmt.Errorf("login: missing password length")
	}
	if err := validateSlotLength("password", passwordLength, loginSecretMin, loginSecretMax); err != nil {
		return AccountIdentity{}, "", err
	}
	passwordEnd, ok := advance(next, passwordLength, len(message))
	if !ok {
		return AccountIdentity{}, "", fmt.Errorf("login: truncated password slot")
	}
	secretDigest := md5.New()
	_, _ = secretDigest.Write([]byte(loginSecretDomain))
	_, _ = secretDigest.Write([]byte{0})
	binary.LittleEndian.PutUint32(encodedLength[:], passwordLength)
	_, _ = secretDigest.Write(encodedLength[:])
	_, _ = secretDigest.Write(message[next:passwordEnd])
	cursor = passwordEnd

	// client-system/type：u32 值，不属于账号身份。
	if _, cursor, ok = readU32(message, cursor); !ok {
		return AccountIdentity{}, "", fmt.Errorf("login: missing client type")
	}

	loginStringLength, next, ok := readU32(message, cursor)
	if !ok {
		return AccountIdentity{}, "", fmt.Errorf("login: missing login-string length")
	}
	if err := validateSlotLength("login string", loginStringLength, loginSecretMin, loginSecretMax); err != nil {
		return AccountIdentity{}, "", err
	}
	cursor, ok = advance(next, loginStringLength, len(message))
	if !ok {
		return AccountIdentity{}, "", fmt.Errorf("login: truncated login-string slot")
	}

	// login-type 是最后一个 u32，必须恰好收尾——两种布局无法共用同一种解释。
	if _, next, ok = readU32(message, cursor); !ok {
		return AccountIdentity{}, "", fmt.Errorf("login: missing login type")
	}
	if next != len(message) {
		return AccountIdentity{}, "", fmt.Errorf("login: trailing login data")
	}
	return AccountIdentity{
		id: accountKeyPrefix + hex.EncodeToString(digest.Sum(nil)),
	}, hex.EncodeToString(secretDigest.Sum(nil)), nil
}

// ParseLoginAccountIdentity 校验 op02 结构并取账号身份（密码摘要弃用）。
func ParseLoginAccountIdentity(message []byte) (AccountIdentity, error) {
	identity, _, err := ParseLoginCredentials(message)
	return identity, err
}

func validateSlotLength(name string, length, minimum, maximum uint32) error {
	if length < minimum || length > maximum || length%16 != 0 {
		return fmt.Errorf("login: invalid %s slot length", name)
	}
	return nil
}

func readU32(message []byte, cursor int) (uint32, int, bool) {
	if cursor < 0 || len(message)-cursor < 4 {
		return 0, cursor, false
	}
	return binary.LittleEndian.Uint32(message[cursor : cursor+4]), cursor + 4, true
}

func advance(cursor int, length uint32, limit int) (int, bool) {
	end := cursor + int(length)
	if end > limit || end < cursor {
		return cursor, false
	}
	return end, true
}
