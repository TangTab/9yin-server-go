package entry

import (
	"crypto/des"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"os"
)

// licenseSalt 是 LookupThread 逐字节写死的 16 字节盐（FsEntry.dll
// 0x1005D827 起），后接 16 字节 0xCC 填充共同参与 md5 密钥派生。
var licenseSalt = [16]byte{
	0x77, 0x56, 0xD3, 0x34, 0x2B, 0x8C, 0xE2, 0x2F,
	0xED, 0xAC, 0xB8, 0x15, 0x45, 0x9E, 0x5E, 0xC4,
}

// LicenseSize 官方 lic 文件固定 256 字节（fread 一次 0x100，少一字节即判错）。
const LicenseSize = 256

// 官方网关版魔数对（快照 127.0.0.1.lic 首块 ECB 解密实测；
// solo 变体 TimerCheck 对照值为 0xE7990369/0x87930363）。
const (
	OfficialMagic1 = 0x8E74C52F
	OfficialMagic2 = 0xEB64EB5C
)

// LicenseIdentity 是解密后写进 GameEntry 身份域（官方 +0xC4..+0xD4）的
// 首 20 字节：两个魔数 u32 + 三个随行 u32（快照实测后三者为随机填充，
// 仅魔数对参与门槛判定）。
type LicenseIdentity struct {
	Magic1 uint32
	Magic2 uint32
	Spare  [3]uint32
}

// licenseKey 复刻密钥派生：md5(IP 字符串 + 盐16 + 0xCC×16)，取 16 字节
// 作双域 3DES 的 K1||K2（EDE 等价 K1||K2||K1）。
func licenseKey(ip string) []byte {
	buf := make([]byte, 0, len(ip)+32)
	buf = append(buf, ip...)
	buf = append(buf, licenseSalt[:]...)
	buf = append(buf, make([]byte, 16)...)
	for i := len(buf) - 16; i < len(buf); i++ {
		buf[i] = 0xCC
	}
	sum := md5.Sum(buf)
	return sum[:]
}

// decryptECB 逐 8 字节块解密（官方 16 次 ×16B 的两块一组等价形态）。
func decryptECB(key []byte, src []byte) []byte {
	k24 := make([]byte, 24)
	copy(k24[:8], key[:8])
	copy(k24[8:16], key[8:16])
	copy(k24[16:24], key[:8])
	c, err := des.NewTripleDESCipher(k24)
	if err != nil {
		panic(err) // 密钥长度恒定，不可达
	}
	out := make([]byte, len(src))
	for off := 0; off+8 <= len(src); off += 8 {
		c.Decrypt(out[off:off+8], src[off:off+8])
	}
	return out
}

func encryptECB(key []byte, src []byte) []byte {
	k24 := make([]byte, 24)
	copy(k24[:8], key[:8])
	copy(k24[8:16], key[8:16])
	copy(k24[16:24], key[:8])
	c, err := des.NewTripleDESCipher(k24)
	if err != nil {
		panic(err)
	}
	out := make([]byte, len(src))
	for off := 0; off+8 <= len(src); off += 8 {
		c.Encrypt(out[off:off+8], src[off:off+8])
	}
	return out
}

// GenerateLicense 为指定 IP 生成官方形态 lic：明文 = 魔数 8B + 零填充，
// 整体 ECB 加密。快照实物尾部为随机填充（生成器无法也不需要复刻随机
// 位——校验只认首块魔数对）。
func GenerateLicense(ip string, magic1, magic2 uint32) []byte {
	plain := make([]byte, LicenseSize)
	binary.LittleEndian.PutUint32(plain[0:], magic1)
	binary.LittleEndian.PutUint32(plain[4:], magic2)
	return encryptECB(licenseKey(ip), plain)
}

// DecodeLicense 解密并校验 lic：长度门 → ECB 解密 → 首块魔数对匹配。
// want 传 0,0 表示仅解出不校验。
func DecodeLicense(data []byte, ip string, want1, want2 uint32) (LicenseIdentity, error) {
	var id LicenseIdentity
	if len(data) != LicenseSize {
		return id, fmt.Errorf("license file error: %d bytes", len(data))
	}
	plain := decryptECB(licenseKey(ip), data)
	id.Magic1 = binary.LittleEndian.Uint32(plain[0:])
	id.Magic2 = binary.LittleEndian.Uint32(plain[4:])
	for i := range id.Spare {
		id.Spare[i] = binary.LittleEndian.Uint32(plain[8+i*4:])
	}
	if want1 != 0 || want2 != 0 {
		if id.Magic1 != want1 || id.Magic2 != want2 {
			return id, fmt.Errorf("license file error: magic %#x/%#x", id.Magic1, id.Magic2)
		}
	}
	return id, nil
}

// LoadLicenseFile 读取并解码 `<外网IP>.lic`（缺文件 = "license file not found"）。
func LoadLicenseFile(path, ip string, want1, want2 uint32) (LicenseIdentity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LicenseIdentity{}, fmt.Errorf("license file not found: %w", err)
	}
	return DecodeLicense(data, ip, want1, want2)
}
