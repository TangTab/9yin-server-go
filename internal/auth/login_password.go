// 密码域派生链（fork exe internal/auth/login_password.go 逆向移植）。
//
// 客户端登录框输入的密码先经本链变换为「登录密码」，再由 field_cipher 的
// field2 域加密进密码槽；注册端存同链密文，登录时按字节比对（fork oracle
// 五组向量验证，见 field_cipher_test.go）：
//
//	EncryptPassword:      AES-256-CBC(固定键, 零 IV) + PKCS7 → base64
//	ComputeLoginPassword: md5(上步) 字节和低字节作校验头 + 上步 → base64 → 半长旋转
//	ComputePasswordVerifier = EncryptFields(账号, 登录密码)[1]（密码槽密文）
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
)

// passwordKey 是客户端硬编码的 32 字节 AES-256 键（fork exe .rdata 字面量：
// 拼音句子 + 补零）。服务端仅用于重放派生，不做口令存储。
var passwordKey = append([]byte("nengbupojiemexiechengxuhaoxinku"), 0x00)

// ErrPasswordTransform 派生链中不可恢复的加密错误（键长恒合法，理论不可达）。
var ErrPasswordTransform = errors.New("auth: password transform")

// pkcs7Pad 16 字节组 PKCS7 填充（恒补 1..16 字节）。
func pkcs7Pad(data []byte) []byte {
	padLen := 16 - len(data)%16
	padded := make([]byte, len(data)+padLen)
	copy(padded, data)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}
	return padded
}

// EncryptPassword 客户口令加密形态：AES-256-CBC（passwordKey、零 IV）+
// PKCS7 → StdEncoding base64（fork EncryptPassword）。
func EncryptPassword(password string) string {
	block, err := aes.NewCipher(passwordKey)
	if err != nil {
		return "" // 32 字节键恒合法，不可达分支
	}
	padded := pkcs7Pad([]byte(password))
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, make([]byte, aes.BlockSize)).CryptBlocks(ciphertext, padded)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

// rotateHalf 半长旋转（fork login_password.go 内联助手）：后半接前半。
// 输入恒为 base64（长度为 4 的倍数），无奇长分支。
func rotateHalf(s string) string {
	half := len(s) / 2
	return s[half:] + s[:half]
}

// ComputeLoginPassword 派生客户端实际发送的「登录密码」。
func ComputeLoginPassword(password string) string {
	encrypted := EncryptPassword(password)
	if encrypted == "" {
		return ""
	}
	digest := md5.Sum([]byte(encrypted))
	var checksum byte
	for _, b := range digest {
		checksum += b
	}
	return rotateHalf(base64.StdEncoding.EncodeToString(append([]byte{checksum}, encrypted...)))
}

// ComputePasswordVerifier 注册时落库的口令校验子：field2 域加密
// ComputeLoginPassword(密码) 的密文。登录帧密码槽与此逐字节相等即通过。
// account 形参保持 fork 签名对齐（field2 输出只依赖密码本身）。
func ComputePasswordVerifier(account, password string) []byte {
	_, ciphertext := EncryptFields(account, ComputeLoginPassword(password))
	return ciphertext
}

// AccountKeyFor 由明文账号名计算 acct:v1 内部键（fork AccountKeyFor）：
// EncryptFields(名, 占位)[0] 的账号槽密文走 login_identity.go 同一哈希域。
// 与登录侧 ParseLoginAccountIdentity 对同一密文产出完全一致——注册与登录
// 天然对齐，存量账号键零迁移。
func AccountKeyFor(name string) (string, error) {
	if len(name) == 0 {
		return "", errors.New("auth: empty account name")
	}
	if len(name) > 256 {
		return "", errors.New("auth: account name too long")
	}
	accountCiphertext, _ := EncryptFields(name, "x")
	if len(accountCiphertext) == 0 {
		return "", ErrPasswordTransform
	}
	digest := sha256.New()
	digest.Write([]byte(loginAccountDomain))
	digest.Write([]byte{0})
	var encodedLength [4]byte
	binary.LittleEndian.PutUint32(encodedLength[:], uint32(len(accountCiphertext)))
	digest.Write(encodedLength[:])
	digest.Write(accountCiphertext)
	return accountKeyPrefix + hex.EncodeToString(digest.Sum(nil)), nil
}
