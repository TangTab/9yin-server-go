package mpq

import "math/bits"

// Jenkins lookup3 hashlittle2 —— MPQ v3+ 归档 HET/BET 表的 64 位名称哈希。
// 移植自 Bob Jenkins 公有领域 lookup3.c（pc=2、pb=1 调用）。

// FileNameHash 计算 HET/BET 表使用的 64 位文件名哈希：
// 小写归一化后以 pc=2、pb=1 做 hashlittle2，组合 (b<<32)|c。
func FileNameHash(name string) uint64 {
	raw := []byte(name)
	normalized := make([]byte, len(raw))
	for i, b := range raw {
		normalized[i] = lowerByte(b)
	}
	c, b := hashlittle2(normalized, 2, 1)
	return uint64(b)<<32 | uint64(c)
}

// hashlittle2 返回 (c, b)（按无符号 u32 语义存放）。
func hashlittle2(key []byte, pc, pb int) (uint32, uint32) {
	// a = b = c = 0xdeadbeef + 长度 + pc; c += pb
	init := uint32(0xDEADBEEF + uint32(len(key)) + uint32(pc))
	a := init
	b := init
	c := init + uint32(pb)

	off := 0
	remaining := len(key)
	for remaining > 12 {
		a += leU32(key, off)
		b += leU32(key, off+4)
		c += leU32(key, off+8)
		a, b, c = mix32(a, b, c)
		off += 12
		remaining -= 12
	}

	// 最后一个（可能不完整的）块，各情况依次向下执行。
	n := remaining
	if n == 0 {
		return c, b
	}
	if n >= 12 {
		c += uint32(key[off+11]) << 24
	}
	if n >= 11 {
		c += uint32(key[off+10]) << 16
	}
	if n >= 10 {
		c += uint32(key[off+9]) << 8
	}
	if n >= 9 {
		c += uint32(key[off+8])
	}
	if n >= 8 {
		b += uint32(key[off+7]) << 24
	}
	if n >= 7 {
		b += uint32(key[off+6]) << 16
	}
	if n >= 6 {
		b += uint32(key[off+5]) << 8
	}
	if n >= 5 {
		b += uint32(key[off+4])
	}
	if n >= 4 {
		a += uint32(key[off+3]) << 24
	}
	if n >= 3 {
		a += uint32(key[off+2]) << 16
	}
	if n >= 2 {
		a += uint32(key[off+1]) << 8
	}
	// 走到这里必然 n >= 1
	a += uint32(key[off])

	a, b, c = finalMix32(a, b, c)
	return c, b
}

func leU32(k []byte, off int) uint32 {
	return uint32(k[off]) | uint32(k[off+1])<<8 | uint32(k[off+2])<<16 | uint32(k[off+3])<<24
}

func mix32(a, b, c uint32) (uint32, uint32, uint32) {
	a -= c
	a ^= bits.RotateLeft32(c, 4)
	c += b
	b -= a
	b ^= bits.RotateLeft32(a, 6)
	a += c
	c -= b
	c ^= bits.RotateLeft32(b, 8)
	b += a
	a -= c
	a ^= bits.RotateLeft32(c, 16)
	c += b
	b -= a
	b ^= bits.RotateLeft32(a, 19)
	a += c
	c -= b
	c ^= bits.RotateLeft32(b, 4)
	b += a
	return a, b, c
}

func finalMix32(a, b, c uint32) (uint32, uint32, uint32) {
	c ^= b
	c -= bits.RotateLeft32(b, 14)
	a ^= c
	a -= bits.RotateLeft32(c, 11)
	b ^= a
	b -= bits.RotateLeft32(a, 25)
	c ^= b
	c -= bits.RotateLeft32(b, 16)
	a ^= c
	a -= bits.RotateLeft32(c, 4)
	b ^= a
	b -= bits.RotateLeft32(a, 14)
	c ^= b
	c -= bits.RotateLeft32(b, 24)
	return a, b, c
}
