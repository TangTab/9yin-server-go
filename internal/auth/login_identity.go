// Package auth 包含现代客户端登录消息的兼容身份边界。
// 它从不暴露密码认证材料。
package auth

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
)

const (
	loginHeaderSize = 25
	accountMin      = 16
	accountMax      = 256
	secretMin       = 16
	secretMax       = 512

	// loginAccountDomain / accountKeyPrefix 是 acct:v1 键的哈希域与前缀
	// （登录侧解析与注册侧 AccountKeyFor 共用，保证两侧同密文同键）。
	loginAccountDomain = "nineyin/login-account/v1"
	accountKeyPrefix   = "acct:v1:"
)

var ErrInvalidLogin = errors.New("login: invalid message")

type AccountIdentity struct {
	id string
}

func (identity AccountIdentity) String() string { return identity.id }

func (identity AccountIdentity) Prefix() string {
	const prefixLength = len("acct:v1:") + 12
	if len(identity.id) <= prefixLength {
		return identity.id
	}
	return identity.id[:prefixLength]
}

func ParseLoginAccountIdentity(message []byte) (AccountIdentity, error) {
	if len(message) < loginHeaderSize || message[0] != 0x02 {
		return AccountIdentity{}, fmt.Errorf("%w: missing login header", ErrInvalidLogin)
	}
	cursor := loginHeaderSize
	accountLength := binary.LittleEndian.Uint32(message[0x15:0x19])
	if err := validateSlotLength("account", accountLength, accountMin, accountMax); err != nil {
		return AccountIdentity{}, err
	}
	accountEnd, ok := advance(cursor, accountLength, len(message))
	if !ok {
		return AccountIdentity{}, fmt.Errorf("%w: truncated account slot", ErrInvalidLogin)
	}

	digest := sha256.New()
	_, _ = digest.Write([]byte(loginAccountDomain))
	_, _ = digest.Write([]byte{0})
	var encodedLength [4]byte
	binary.LittleEndian.PutUint32(encodedLength[:], accountLength)
	_, _ = digest.Write(encodedLength[:])
	_, _ = digest.Write(message[cursor:accountEnd])
	accountDigest := digest.Sum(nil)
	cursor = accountEnd

	passwordLength, next, ok := readLength(message, cursor)
	if !ok {
		return AccountIdentity{}, fmt.Errorf("%w: missing password length", ErrInvalidLogin)
	}
	if err := validateSlotLength("password", passwordLength, secretMin, secretMax); err != nil {
		return AccountIdentity{}, err
	}
	cursor, ok = advance(next, passwordLength, len(message))
	if !ok {
		return AccountIdentity{}, fmt.Errorf("%w: truncated password slot", ErrInvalidLogin)
	}

	// client-system/type。其值不属于账号身份的组成部分。
	if _, next, ok = readLength(message, cursor); !ok {
		return AccountIdentity{}, fmt.Errorf("%w: missing client type", ErrInvalidLogin)
	}
	cursor = next

	loginStringLength, next, ok := readLength(message, cursor)
	if !ok {
		return AccountIdentity{}, fmt.Errorf("%w: missing login-string length", ErrInvalidLogin)
	}
	if err := validateSlotLength("login string", loginStringLength, secretMin, secretMax); err != nil {
		return AccountIdentity{}, err
	}
	cursor, ok = advance(next, loginStringLength, len(message))
	if !ok {
		return AccountIdentity{}, fmt.Errorf("%w: truncated login-string slot", ErrInvalidLogin)
	}

	// login type 是最后一个 u32。拒绝尾部数据，使两种布局无法共用同一种
	// 身份解析器的解释。
	if _, next, ok = readLength(message, cursor); !ok {
		return AccountIdentity{}, fmt.Errorf("%w: missing login type", ErrInvalidLogin)
	}
	if next != len(message) {
		return AccountIdentity{}, fmt.Errorf("%w: trailing login data", ErrInvalidLogin)
	}
	return AccountIdentity{id: accountKeyPrefix + hex.EncodeToString(accountDigest)}, nil
}

func validateSlotLength(name string, length uint32, minimum, maximum uint32) error {
	if length < minimum || length > maximum || length%16 != 0 {
		return fmt.Errorf("%w: invalid %s slot length", ErrInvalidLogin, name)
	}
	return nil
}

func readLength(message []byte, cursor int) (uint32, int, bool) {
	if cursor < 0 || len(message)-cursor < 4 {
		return 0, cursor, false
	}
	return binary.LittleEndian.Uint32(message[cursor : cursor+4]), cursor + 4, true
}

func advance(cursor int, length uint32, limit int) (int, bool) {
	if cursor < 0 || cursor > limit || uint64(length) > uint64(limit-cursor) {
		return cursor, false
	}
	return cursor + int(length), true
}
