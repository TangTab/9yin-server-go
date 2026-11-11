// Package auth 的登录域密码学（fork 9yin-game-native-menu.exe 逆向移植）。
//
// 客户端 op02 登录帧的账号/密码槽是确定性加密的（同明文→同密文），客户端与
// 服务端各自加密后比对密文即可完成身份校验——全程无需解密。本文件实现
// fork exe internal/auth/field_cipher.go 的双域 128 位 DES 阶梯（fork oracle
// 五组账号逐字节验证，见 field_cipher_test.go 与 docs/fork-login-crypto.md）：
//
//	块结构：16 字节块 → 4×LE32 → 按块序奇偶 XOR Even/Odd 掩码 → 三级阶梯
//	阶梯：  E(k1a) 独立加密两半 → D(k1b) 交叉混合 → E(k1a) 再独立（EDE）
//	单级：  标准 DES（crypto/des 等价，fork fcDesCore 反汇编+oracle 双证）
//
// 账号槽走 field1 键系，密码槽走 field2 键系；两域互不相干。
package auth

import (
	"crypto/des"
	"encoding/binary"
	"math/bits"
)

// ErrInvalidLogin 由本包各入口复用（login_identity.go 定义）。

// fcField1Key / fcField2Key 是 fork exe .data 段的两个 16 字节固定域键：
// [0:8] 为阶梯第 1/3 级（E 方向）的 DES 键，[8:16] 为第 2 级（D 方向）键。
var (
	fcField1Key = [16]byte{0x66, 0xf9, 0x28, 0xe8, 0xa1, 0x00, 0x9b, 0x76, 0x29, 0xa0, 0x1f, 0x8d, 0xef, 0x45, 0x89, 0x19}
	fcField2Key = [16]byte{0x10, 0x6e, 0xba, 0x55, 0x77, 0x61, 0xc7, 0x7e, 0x1e, 0xea, 0x0d, 0xeb, 0x67, 0x62, 0x93, 0xdc}

	// 块序号偶/奇时的 16 字节 XOR 预白化掩码（4×LE32 与块词异或）。
	fcField1Even = [16]byte{0xa7, 0xd0, 0xeb, 0xf8, 0x9c, 0xdc, 0xba, 0xc4, 0xa2, 0xd0, 0x4f, 0xa8, 0xb7, 0x24, 0xcf, 0x74}
	fcField1Odd  = [16]byte{0x92, 0x4c, 0xe2, 0x26, 0x09, 0xda, 0xef, 0x38, 0x47, 0xd0, 0xb2, 0xc3, 0x9c, 0xd4, 0xef, 0x28}
	fcField2Even = [16]byte{0xe5, 0x83, 0x4d, 0xf2, 0xd8, 0x26, 0x25, 0xab, 0x88, 0x36, 0xb9, 0x32, 0x20, 0xfe, 0xfd, 0xaf}
	fcField2Odd  = [16]byte{0xe5, 0x5a, 0x43, 0xe2, 0xb1, 0x26, 0x61, 0x13, 0x85, 0x6e, 0xba, 0x52, 0x23, 0xfb, 0xa3, 0x3f}
)

// fcField 是一个域的全部会话态：E/D/E 阶梯的三向 DES 块密码加双掩码，
// 外加逆向（untransform128）用的换向组合。
type fcField struct {
	keyA, keyB   desCipher
	keyADecrypt  desCipher // 键 A 解密向（撤销 E 级）
	keyBEncrypt  desCipher // 键 B 加密向（撤销 D 级）
	even, odd    [16]byte
}

type desCipher struct {
	encrypt bool
	block   interface {
		Encrypt(dst, src []byte)
		Decrypt(dst, src []byte)
	}
}

// desStep 对 (l, r) 两个大端 32 位半块执行一步标准 DES（fork fcDesCore 等价：
// KAT 向量 85E813540F0AB405 双证），返回新的半块对。
func (c desCipher) step(l, r uint32) (uint32, uint32) {
	var blk [8]byte
	binary.BigEndian.PutUint32(blk[0:4], l)
	binary.BigEndian.PutUint32(blk[4:8], r)
	if c.encrypt {
		c.block.Encrypt(blk[:], blk[:])
	} else {
		c.block.Decrypt(blk[:], blk[:])
	}
	return binary.BigEndian.Uint32(blk[0:4]), binary.BigEndian.Uint32(blk[4:8])
}

var (
	fcAccountField = newFCField(fcField1Key, fcField1Even, fcField1Odd)
	fcPasswordField = newFCField(fcField2Key, fcField2Even, fcField2Odd)
)

func newFCField(key [16]byte, even, odd [16]byte) fcField {
	a, err1 := des.NewCipher(key[0:8])
	b, err2 := des.NewCipher(key[8:16])
	if err1 != nil || err2 != nil {
		// 8 字节键恒有效，仅防御性处理
		panic("auth: field cipher key: " + err1.Error() + "/" + err2.Error())
	}
	return fcField{
		keyA:        desCipher{encrypt: true, block: a},
		keyB:        desCipher{encrypt: false, block: b},
		keyADecrypt: desCipher{encrypt: false, block: a},
		keyBEncrypt: desCipher{encrypt: true, block: b},
		even:        even,
		odd:         odd,
	}
}

// transform128 是单块三级阶梯（fork fcTransform128）：
//
//	(a,b) = E(w0,w1)   (c,d) = E(w2,w3)     ← 两半独立，键 A 加密
//	(e,f) = D(a,c)     (g,h) = D(b,d)       ← 左左/右右交叉，键 B 解密
//	(i,j) = E(e,g)     (k,l) = E(f,h)       ← 再独立，键 A 加密
//	输出 = (i,j,k,l) 大端回填
func (f fcField) transform128(w0, w1, w2, w3 uint32) [16]byte {
	a, b := f.keyA.step(w0, w1)
	c, d := f.keyA.step(w2, w3)
	e, f2 := f.keyB.step(a, c)
	g, h := f.keyB.step(b, d)
	i, j := f.keyA.step(e, g)
	k, l := f.keyA.step(f2, h)
	var out [16]byte
	binary.BigEndian.PutUint32(out[0:4], i)
	binary.BigEndian.PutUint32(out[4:8], j)
	binary.BigEndian.PutUint32(out[8:12], k)
	binary.BigEndian.PutUint32(out[12:16], l)
	return out
}

// encryptField 对 16 字节整倍数据逐块加密：块词（小端）XOR 奇偶掩码后，
// 经一次字节序翻转进阶梯（fork encryptField 的 LE 载入与 fcTransform128 的
// BE 载入天然构成 bswap），阶梯输出以大端回填密文。
func (f fcField) encryptField(padded []byte) []byte {
	out := make([]byte, 0, len(padded))
	for idx := 0; idx*16 < len(padded); idx++ {
		blk := padded[idx*16 : idx*16+16]
		mask := f.even
		if idx%2 == 1 {
			mask = f.odd
		}
		w0 := bits.ReverseBytes32(binary.LittleEndian.Uint32(blk[0:4]) ^ binary.LittleEndian.Uint32(mask[0:4]))
		w1 := bits.ReverseBytes32(binary.LittleEndian.Uint32(blk[4:8]) ^ binary.LittleEndian.Uint32(mask[4:8]))
		w2 := bits.ReverseBytes32(binary.LittleEndian.Uint32(blk[8:12]) ^ binary.LittleEndian.Uint32(mask[8:12]))
		w3 := bits.ReverseBytes32(binary.LittleEndian.Uint32(blk[12:16]) ^ binary.LittleEndian.Uint32(mask[12:16]))
		block := f.transform128(w0, w1, w2, w3)
		out = append(out, block[:]...)
	}
	return out
}

// pad16 零填充到 16 字节整倍（fork pad16；密码域另用 PKCS7，见 login_password.go）。
func pad16(data []byte) []byte {
	if len(data)%16 == 0 {
		return data
	}
	padded := make([]byte, (len(data)+15)&^15)
	copy(padded, data)
	return padded
}

// untransform128 是阶梯的逆（服务端展示用：m655/m607 回传真实账号名）。
// 逆向即反序 + 换向：E 级用键 A 解密撤销，D 级用键 B 加密撤销。
func (f fcField) untransform128(i, j, k, l uint32) (w0, w1, w2, w3 uint32) {
	e, g := f.keyADecrypt.step(i, j) // 撤销 (i,j) = E(e,g)
	f2, h := f.keyADecrypt.step(k, l) // 撤销 (k,l) = E(f,h)
	a, c := f.keyBEncrypt.step(e, f2) // 撤销 (e,f) = D(a,c)
	b, d := f.keyBEncrypt.step(g, h)  // 撤销 (g,h) = D(b,d)
	w0, w1 = f.keyADecrypt.step(a, b) // 撤销 (a,b) = E(w0,w1)
	w2, w3 = f.keyADecrypt.step(c, d) // 撤销 (c,d) = E(w2,w3)
	return
}

// decryptField 逐块解密：大端取词 → 逆阶梯 → bswap → XOR 同奇偶掩码 → 小端回填。
func (f fcField) decryptField(ciphertext []byte) []byte {
	out := make([]byte, 0, len(ciphertext))
	for idx := 0; idx*16 < len(ciphertext); idx++ {
		blk := ciphertext[idx*16 : idx*16+16]
		mask := f.even
		if idx%2 == 1 {
			mask = f.odd
		}
		w0, w1, w2, w3 := f.untransform128(
			binary.BigEndian.Uint32(blk[0:4]),
			binary.BigEndian.Uint32(blk[4:8]),
			binary.BigEndian.Uint32(blk[8:12]),
			binary.BigEndian.Uint32(blk[12:16]))
		var plain [16]byte
		binary.LittleEndian.PutUint32(plain[0:4], bits.ReverseBytes32(w0)^binary.LittleEndian.Uint32(mask[0:4]))
		binary.LittleEndian.PutUint32(plain[4:8], bits.ReverseBytes32(w1)^binary.LittleEndian.Uint32(mask[4:8]))
		binary.LittleEndian.PutUint32(plain[8:12], bits.ReverseBytes32(w2)^binary.LittleEndian.Uint32(mask[8:12]))
		binary.LittleEndian.PutUint32(plain[12:16], bits.ReverseBytes32(w3)^binary.LittleEndian.Uint32(mask[12:16]))
		out = append(out, plain[:]...)
	}
	return out
}

// DecryptAccountFields 解出（账号明文, 密码域明文——后者是派生后的登录密码
// 形态，非原始口令）。账号明文仅保留可打印 ASCII（m655/m607 tag6 窄串约束，
// 见 official_login_align.go）；不合规或空串返回空。
func DecryptAccountFields(accountCiphertext, passwordCiphertext []byte) (account, derived string) {
	account = trimPad16(fcAccountField.decryptField(accountCiphertext))
	derived = trimPad16(fcPasswordField.decryptField(passwordCiphertext))
	if !isPrintableASCII(account) {
		account = ""
	}
	return
}

func trimPad16(padded []byte) string {
	end := len(padded)
	for end > 0 && padded[end-1] == 0 {
		end--
	}
	return string(padded[:end])
}

func isPrintableASCII(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 || value[i] > 0x7e {
			return false
		}
	}
	return true
}

// EncryptFields 是双域加密总入口（fork EncryptFields）：
// 返回 (账号槽密文, 密码槽密文)。确定性：同输入恒同输出，服务端凭比对校验。
func EncryptFields(account, password string) (accountCiphertext, passwordCiphertext []byte) {
	accountCiphertext = fcAccountField.encryptField(pad16([]byte(account)))
	passwordCiphertext = fcPasswordField.encryptField(pad16([]byte(password)))
	return accountCiphertext, passwordCiphertext
}
