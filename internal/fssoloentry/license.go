package fssoloentry

import (
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// lookup_thread 逐字节写死的 16 字节盐（FsSoloEntry.dll 0x10089F11 起，
// 与网关版 FsEntry 同盐不同拼序）。
var licenseSalt = [16]byte{
	0x77, 0x56, 0xD3, 0x34, 0x2B, 0x8C, 0xE2, 0x2F,
	0xED, 0xAC, 0xB8, 0x15, 0x45, 0x9E, 0x5E, 0xC4,
}

// 官方 lic 形态尺寸（fread 一次读入，尺寸即形态判定）。
const (
	LicenseInternalSize = 256 // 0x100：内部形态（本机 IP 须 127.0.0.1/192.168.*）
	LicenseExternalSize = 272 // 0x110：外部形态（尾部 16B 二段密钥+时间窗）
)

// TimerCheck 复验魔数对（FsSoloEntry.dll 0x100A28EB/0x100A2902 直读；
// 网关版为 0x8E74C52F/0xEB64EB5C——快照 127.0.0.1.lic 实测属网关版）。
const (
	OfficialMagic1 = 0xE7990369
	OfficialMagic2 = 0x87930363
)

// License 是解密后的授权载荷：身份域 8×u32（写 SoloEntry+0x1b4..+0x1d0）
// 与外部形态的时间窗。
type License struct {
	// Internal 内部形态（无时间窗约束）。
	Internal bool
	// Fields 身份域（明文 buf[0..0x1F]；Fields[0]/[1] 即魔数对）。
	Fields [8]uint32
	// T0/T1 外部形态时间窗（明文尾 buf[0x100..0x107]，unix 秒）。
	T0, T1 uint32
}

// Licensed 魔数对是否命中（TimerCheck 门槛的等价判定）。
func (l *License) Licensed() bool {
	return l != nil && l.Fields[0] == OfficialMagic1 && l.Fields[1] == OfficialMagic2
}

// licenseKey 复刻密钥派生：md5(盐16 || IP串截 16B)（CopyString 语义：
// 最多 15 字符 + NUL）。外部形态二段密钥：先用一段密钥解出明文尾
// buf[0x100..0x107]，回填 key[16..23] 再 md5。
func licenseKey(ip string, tailPlain8 []byte) []byte {
	buf := make([]byte, 32)
	copy(buf[:16], licenseSalt[:])
	copy(buf[16:], ip)
	if len(ip) < 16 {
		buf[16+len(ip)] = 0 // CopyString 收尾 NUL
	}
	if len(tailPlain8) == 8 {
		copy(buf[16:24], tailPlain8)
	}
	sum := md5.Sum(buf)
	return sum[:]
}

func tripleDES(key []byte) cipher.Block {
	k24 := make([]byte, 24)
	copy(k24[:8], key[:8])
	copy(k24[8:16], key[8:16])
	copy(k24[16:24], key[:8]) // EDE 等价 K1||K2||K1（des2key 16 字节形态）
	c, err := des.NewTripleDESCipher(k24)
	if err != nil {
		panic(err) // 密钥长度恒定，不可达
	}
	return c
}

func decryptECB(key, src []byte) []byte {
	c := tripleDES(key)
	out := make([]byte, len(src))
	for off := 0; off+8 <= len(src); off += 8 {
		c.Decrypt(out[off:off+8], src[off:off+8])
	}
	return out
}

func encryptECB(key, src []byte) []byte {
	c := tripleDES(key)
	out := make([]byte, len(src))
	for off := 0; off+8 <= len(src); off += 8 {
		c.Encrypt(out[off:off+8], src[off:off+8])
	}
	return out
}

// LocalIPString 复刻 lookup_thread 的 IP 采集：外网监听 socket
// getsockname → 点分串（等价取本机地址；内部形态另需 127.0.0.1 或
// 192.168.* 门槛）。
func LocalIPString(prefer string) string {
	if prefer != "" {
		return prefer
	}
	conn, err := net.Dial("udp", "127.0.0.1:1")
	if err == nil {
		defer conn.Close()
		if host, _, err := net.SplitHostPort(conn.LocalAddr().String()); err == nil {
			return host
		}
	}
	return "127.0.0.1"
}

// DecodeLicense 解密并校验 lic（lookup_thread 全流程）：
//
//	尺寸门（256/272 之外即 "license file error"）→ 内部形态本机 IP 门槛
//	（须 127.0.0.1 或 192.168.*，否则 "not internal address"）→ 密钥
//	派生（外部形态二段）→ 3DES 解密 → 前 252 字节求和 == u32@0xFC →
//	外部形态时间窗（now < t0 "license too young!" / > t1 "too old!"）
//	→ 身份域 8×u32。
func DecodeLicense(data []byte, ip string, now time.Time) (License, error) {
	var lic License
	internal := len(data) == LicenseInternalSize
	if !internal && len(data) != LicenseExternalSize {
		return lic, fmt.Errorf("license file error: %d bytes", len(data))
	}
	if internal && ip != "127.0.0.1" && !strings.HasPrefix(ip, "192.168.") {
		return lic, fmt.Errorf("not internal address: %s", ip)
	}
	key := licenseKey(ip, nil)
	if !internal {
		// 二段密钥：一段密钥解出明文尾 buf[0x100..0x10F]
		tail := decryptECB(key, data[LicenseInternalSize:])
		key = licenseKey(ip, tail[:8])
		lic.T0 = binary.LittleEndian.Uint32(tail[0:])
		lic.T1 = binary.LittleEndian.Uint32(tail[4:])
	}
	plain := decryptECB(key, data[:LicenseInternalSize])
	var sum uint32
	for i := 0; i < 0xFC; i++ {
		sum += uint32(plain[i])
	}
	if sum != binary.LittleEndian.Uint32(plain[0xFC:]) {
		return lic, fmt.Errorf("license checksum error")
	}
	if !internal {
		ts := uint32(now.Unix())
		if ts < lic.T0 {
			return lic, fmt.Errorf("license too young!")
		}
		if ts > lic.T1 {
			return lic, fmt.Errorf("license too old!")
		}
	}
	lic.Internal = internal
	for i := range lic.Fields {
		lic.Fields[i] = binary.LittleEndian.Uint32(plain[i*4:])
	}
	return lic, nil
}

// LoadLicenseFile 读取并解码 `<外网IP>.lic`（缺文件 = "license file
// not found"）。
func LoadLicenseFile(path, ip string, now time.Time) (License, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return License{}, fmt.Errorf("license file not found: %w", err)
	}
	return DecodeLicense(data, ip, now)
}

// GenerateLicense 生成内部形态 lic（魔数对 + 零填充 + 求和校验回填）。
// 官方生成器不在快照内；本生成器产物与官方校验序完全互认。
func GenerateLicense(ip string) []byte {
	return generateLicense(ip, License{Internal: true, Fields: [8]uint32{
		OfficialMagic1, OfficialMagic2, 0, 0, 0, 0, 0, 0,
	}}, time.Time{})
}

// GenerateExternalLicense 生成外部形态 lic（时间窗 [t0,t1]，unix 秒）。
func GenerateExternalLicense(ip string, t0, t1 uint32) []byte {
	return generateLicense(ip, License{Fields: [8]uint32{
		OfficialMagic1, OfficialMagic2, 0, 0, 0, 0, 0, 0,
	}, T0: t0, T1: t1}, time.Time{})
}

func generateLicense(ip string, lic License, _ time.Time) []byte {
	// 明文主体：身份域 + 零填充，0xFC 处回填前 252 字节求和
	plain := make([]byte, LicenseInternalSize)
	for i, v := range lic.Fields {
		binary.LittleEndian.PutUint32(plain[i*4:], v)
	}
	var sum uint32
	for i := 0; i < 0xFC; i++ {
		sum += uint32(plain[i])
	}
	binary.LittleEndian.PutUint32(plain[0xFC:], sum)

	key1 := licenseKey(ip, nil)
	if lic.Internal {
		return encryptECB(key1, plain)
	}
	// 外部形态：明文尾 {t0,t1,0,0} 用一段密钥加密；主体用二段密钥
	tail := make([]byte, 16)
	binary.LittleEndian.PutUint32(tail[0:], lic.T0)
	binary.LittleEndian.PutUint32(tail[4:], lic.T1)
	key2 := licenseKey(ip, tail[:8])
	return append(encryptECB(key2, plain), encryptECB(key1, tail)...)
}
