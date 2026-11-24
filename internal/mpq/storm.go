package mpq

import (
	"encoding/binary"
	"strings"
)

// Storm 风格的加密、哈希与名称处理，复刻暴雪 storm.dll / StormLib 的算法：
// 0x500 项 crypt table、经典 hash table 的四个哈希函数，以及对称流密码。
// 移植自 Java 参考实现的 StormCrypt.java（uint32 环绕语义逐位一致）。

// 经典表哈希函数的表内偏移。
const (
	hashTableIndex = 0x000 // 经典 hash table 索引（hash % table_size）
	hashNameA      = 0x100 // hash table 条目 name1
	hashNameB      = 0x200 // hash table 条目 name2
	hashFileKey    = 0x300 // 文件级加密密钥
	hashKey2Mix    = 0x400 // 流密码 "key2 mix" 子表
)

// 经典 hash table / block table 的加密密钥：HashString("(hash table)", FILE_KEY)。
const (
	KeyHashTable  uint32 = 0xC3AF3770
	KeyBlockTable uint32 = 0xEC83B3A3
)

// cryptTable 即 storm crypt table，包初始化时生成。
var cryptTable [0x500]uint32

func init() {
	seed := uint32(0x00100001)
	for index1 := 0; index1 < 0x100; index1++ {
		for i := 0; i < 5; i++ {
			index2 := index1 + i*0x100
			seed = (seed*125 + 3) % 0x2AAAAB
			temp1 := (seed & 0xFFFF) << 16
			seed = (seed*125 + 3) % 0x2AAAAB
			temp2 := seed & 0xFFFF
			cryptTable[index2] = temp1 | temp2
		}
	}
}

// upperByte 是 storm 大写映射：a-z → A-Z，/ → \，其余原样。
func upperByte(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - 0x20
	}
	if b == '/' {
		return '\\'
	}
	return b
}

// lowerByte 是 storm 小写映射：A-Z → a-z，/ → \（HET/BET 名称哈希用）。
func lowerByte(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 0x20
	}
	if b == '/' {
		return '\\'
	}
	return b
}

// NormalizeName 按经典表方式规范化文件名：ASCII 大写化，/ 转 \。
func NormalizeName(name string) []byte {
	raw := []byte(name)
	out := make([]byte, len(raw))
	for i, b := range raw {
		out[i] = upperByte(b)
	}
	return out
}

// SameName 判断两个 MPQ 名字是否相等（规范化后比较，不分大小写与斜杠方向）。
func SameName(a, b string) bool {
	x, y := NormalizeName(a), NormalizeName(b)
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

// HashBytes 对已规范化的字节串执行 Storm HashString。
func HashBytes(normalized []byte, hashType int) uint32 {
	seed1 := uint32(0x7FED7FED)
	seed2 := uint32(0xEEEEEEEE)
	for _, b := range normalized {
		ch := uint32(b)
		seed1 = cryptTable[hashType+int(ch)] ^ (seed1 + seed2)
		seed2 = ch + seed1 + seed2 + (seed2 << 5) + 3
	}
	return seed1
}

// HashString 对原始名称执行 Storm HashString（内部先规范化）。
func HashString(name string, hashType int) uint32 {
	return HashBytes(NormalizeName(name), hashType)
}

// cryptBytes 是对称流密码：加密与解密是同一运算，
// 区别只在密钥流推进用明文还是密文。以小端 dword 为单位处理。
func cryptBytes(data []byte, key uint32, encrypting bool) {
	key1 := key
	key2 := uint32(0xEEEEEEEE)
	end := len(data) & ^3
	for i := 0; i < end; i += 4 {
		value := binary.LittleEndian.Uint32(data[i:])
		key2 += cryptTable[hashKey2Mix+int(key1&0xFF)]
		out := value ^ (key1 + key2)
		key1 = ((^key1 << 21) + 0x11111111) | (key1 >> 11)
		plain := value
		if !encrypting {
			plain = out
		}
		key2 = plain + key2 + (key2 << 5) + 3
		binary.LittleEndian.PutUint32(data[i:], out)
	}
}

// EncryptBytes 用 storm 流密码就地加密 data 的整体。
func EncryptBytes(data []byte, key uint32) { cryptBytes(data, key, true) }

// DecryptBytes 用 storm 流密码就地解密 data 的整体。
func DecryptBytes(data []byte, key uint32) { cryptBytes(data, key, false) }

// FileKey 派生文件级加密密钥。name 只取基础名（最后一个 \ 或 / 之后）参与哈希；
// fixKey 对应块带 FILE_KEY_V2 标志时，密钥按块偏移与文件大小修正。
func FileKey(name string, fixKey bool, blockPos uint64, fileSize uint32) uint32 {
	plain := name
	if cut := max(strings.LastIndex(name, `\`), strings.LastIndex(name, `/`)); cut >= 0 {
		plain = name[cut+1:]
	}
	key := HashString(plain, hashFileKey)
	if fixKey {
		key = (key + uint32(blockPos)) ^ fileSize
	}
	return key
}
