// 登录帧凭据槽提取（fork exe internal/auth/login_credentials.go 移植）。
//
// ParseLoginAccountIdentity（login_identity.go）已产出账号键；本文件额外取出
// 两个密文槽的原文切片，供登录链路做密码槽比对（注册时落的
// ComputePasswordVerifier 与客户端密码槽同源确定性密文，逐字节相等即通过）。
package auth

import (
	"encoding/binary"
	"fmt"
)

// ParseLoginCredentials 从 op02 登录帧取出（账号槽密文, 密码槽密文）。
// 结构与 ParseLoginAccountIdentity 同步解析（头 25 字节、[0x15:0x19] 账号槽
// 长、后接密码槽），只到密码槽为止——clientType/loginString/loginType 与
// 凭据校验无关。
func ParseLoginCredentials(message []byte) (accountCiphertext, passwordCiphertext []byte, err error) {
	if len(message) < loginHeaderSize || message[0] != 0x02 {
		return nil, nil, fmt.Errorf("%w: missing login header", ErrInvalidLogin)
	}
	accountLength := binary.LittleEndian.Uint32(message[0x15:0x19])
	if err := validateSlotLength("account", accountLength, accountMin, accountMax); err != nil {
		return nil, nil, err
	}
	cursor := loginHeaderSize
	accountEnd, ok := advance(cursor, accountLength, len(message))
	if !ok {
		return nil, nil, fmt.Errorf("%w: truncated account slot", ErrInvalidLogin)
	}
	accountCiphertext = message[cursor:accountEnd]
	cursor = accountEnd

	passwordLength, next, ok := readLength(message, cursor)
	if !ok {
		return nil, nil, fmt.Errorf("%w: missing password length", ErrInvalidLogin)
	}
	if err := validateSlotLength("password", passwordLength, secretMin, secretMax); err != nil {
		return nil, nil, err
	}
	passwordEnd, ok := advance(next, passwordLength, len(message))
	if !ok {
		return nil, nil, fmt.Errorf("%w: truncated password slot", ErrInvalidLogin)
	}
	passwordCiphertext = message[next:passwordEnd]
	return accountCiphertext, passwordCiphertext, nil
}
