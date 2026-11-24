package mpq

import (
	"bytes"
	"fmt"
)

// PKWARE Data Compression Library 的 explode（解压）。
// 忠实移植自 StormLib 所用的公开 PKWARE DCL 实现（pklib/explode.c）。
// MPQ 以方法字节 0x08 或旧式 IMPLODE 标志存储 PKWARE 压缩数据。

const (
	cmpBinary byte = 0
	cmpASCII  byte = 1
)

// 规范码表（提取自 StormLib pklib/explode.c，勿手工修改）。

var distBits = [64]int{2, 4, 4, 5, 5, 5, 5, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8}

var distCode = [64]int{3, 13, 5, 25, 9, 17, 1, 62, 30, 46, 14, 54, 22, 38, 6, 58, 26, 42, 10, 50, 18, 34, 66, 2, 124, 60, 92, 28, 108, 44, 76, 12, 116, 52, 84, 20, 100, 36, 68, 4, 120, 56, 88, 24, 104, 40, 72, 8, 240, 112, 176, 48, 208, 80, 144, 16, 224, 96, 160, 32, 192, 64, 128, 0}

var exLenBits = [16]int{0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8}

var lenBase = [16]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 10, 14, 22, 38, 70, 134, 262}

var lenBits = [16]int{3, 2, 3, 3, 4, 4, 4, 5, 5, 5, 5, 6, 6, 6, 7, 7}

var lenCode = [16]int{5, 3, 1, 6, 10, 2, 12, 20, 4, 24, 8, 48, 16, 32, 64, 0}

var chBitsAsc = [256]int{11, 12, 12, 12, 12, 12, 12, 12, 12, 8, 7, 12, 12, 7, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 13, 12, 12, 12, 12, 12, 4, 10, 8, 12, 10, 12, 10, 8, 7, 7, 8, 9, 7, 6, 7, 8, 7, 6, 7, 7, 7, 7, 8, 7, 7, 8, 8, 12, 11, 7, 9, 11, 12, 6, 7, 6, 6, 5, 7, 8, 8, 6, 11, 9, 6, 7, 6, 6, 7, 11, 6, 6, 6, 7, 9, 8, 9, 9, 11, 8, 11, 9, 12, 8, 12, 5, 6, 6, 6, 5, 6, 6, 6, 5, 11, 7, 5, 6, 5, 5, 6, 10, 5, 5, 5, 5, 8, 7, 8, 8, 10, 11, 11, 12, 12, 12, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 13, 12, 13, 13, 13, 12, 13, 13, 13, 12, 13, 13, 13, 13, 12, 13, 13, 13, 12, 12, 12, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13}

var chCodeAsc = [256]int{1168, 4064, 2016, 3040, 992, 3552, 1504, 2528, 480, 184, 98, 3808, 1760, 34, 2784, 736, 3296, 1248, 2272, 224, 3936, 1888, 2912, 864, 3424, 1376, 4672, 2400, 352, 3680, 1632, 2656, 15, 592, 56, 608, 80, 3168, 912, 216, 66, 2, 88, 432, 124, 41, 60, 152, 92, 9, 28, 108, 44, 76, 24, 12, 116, 232, 104, 1120, 144, 52, 176, 1808, 2144, 49, 84, 17, 33, 23, 20, 168, 40, 1, 784, 304, 62, 100, 30, 46, 36, 1296, 14, 54, 22, 68, 48, 200, 464, 208, 272, 72, 1552, 336, 96, 136, 4000, 7, 38, 6, 58, 27, 26, 42, 10, 11, 528, 4, 19, 50, 3, 29, 18, 400, 13, 21, 5, 25, 8, 120, 240, 112, 656, 1040, 16, 1952, 2976, 928, 576, 7232, 3136, 5184, 1088, 6208, 2112, 4160, 64, 8064, 3968, 6016, 1920, 7040, 2944, 4992, 896, 7552, 3456, 5504, 1408, 6528, 2432, 4480, 384, 7808, 3712, 5760, 1664, 6784, 2688, 4736, 640, 7296, 3200, 5248, 1152, 6272, 2176, 4224, 128, 7936, 3840, 5888, 1792, 6912, 2816, 4864, 3488, 1440, 2464, 416, 3744, 1696, 2720, 672, 3232, 1184, 2208, 160, 3872, 1824, 2848, 800, 3360, 1312, 2336, 288, 3616, 1568, 2592, 544, 3104, 1056, 2080, 32, 4032, 1984, 3008, 960, 3520, 1472, 2496, 448, 3776, 1728, 2752, 704, 3264, 1216, 2240, 192, 3904, 1856, 2880, 832, 768, 3392, 7424, 3328, 5376, 1344, 1280, 6400, 2304, 2368, 4352, 256, 7680, 3584, 320, 5632, 1536, 6656, 3648, 1600, 2624, 2560, 4608, 512, 7168, 3072, 5120, 1024, 6144, 2048, 4096, 0}

// decodeLit 的结果码。
const (
	litError = 0x306
	litEnd   = 0x305
)

// genDecodeTabs 构建长度码/距离码的规范码查找表。
func genDecodeTabs(startIndexes, lengthBits []int, elements int) []byte {
	positions := make([]byte, 0x100)
	for i := 0; i < elements; i++ {
		length := 1 << lengthBits[i]
		index := startIndexes[i]
		for index < 0x100 {
			positions[index] = byte(i)
			index += length
		}
	}
	return positions
}

// ascTables 是 ASCII 模式的四级解码表。
type ascTables struct {
	chBitsAsc [0x100]byte
	l1        [0x100]byte
	l2        [0x100]byte
	l3        [0x80]byte
	l4        [0x100]byte
}

func genAscTabs() *ascTables {
	t := &ascTables{}
	for i, v := range chBitsAsc {
		t.chBitsAsc[i] = byte(v)
	}

	// 从 0xFF 向 0 迭代各符号；表已清零，与参考实现一致。
	for count := 0xFF; count >= 0; count-- {
		code := chCodeAsc[count]
		bitsAsc := int(t.chBitsAsc[count])

		if bitsAsc <= 8 {
			add := 1 << bitsAsc
			for acc := code; acc < 0x100; acc += add {
				t.l1[acc] = byte(count)
			}
		} else if code&0xFF != 0 {
			t.l1[code&0xFF] = 0xFF
			if code&0x3F != 0 {
				bits := bitsAsc - 4
				t.chBitsAsc[count] = byte(bits)
				add := 1 << bits
				for acc := code >> 4; acc < 0x100; acc += add {
					t.l2[acc] = byte(count)
				}
			} else {
				bits := bitsAsc - 6
				t.chBitsAsc[count] = byte(bits)
				add := 1 << bits
				for acc := code >> 6; acc < 0x80; acc += add {
					t.l3[acc] = byte(count)
				}
			}
		} else {
			bits := bitsAsc - 8
			t.chBitsAsc[count] = byte(bits)
			add := 1 << bits
			for acc := code >> 8; acc < 0x100; acc += add {
				t.l4[acc] = byte(count)
			}
		}
	}
	return t
}

// explodeCtx 保存解压过程的位缓冲与输入位置。
type explodeCtx struct {
	input        []byte
	inPos        int
	bitBuff      uint32 // 16 位位缓冲区；有效位是低 8 + extraBits 位
	extraBits    int
	dsizeBits    int
	dsizeMask    int
	ctype        byte
	lengthCodes  []byte
	distPosCodes []byte
	asc          *ascTables
}

// wasteBits 从位缓冲区移除 nbits 位，必要时从输入补充；返回是否仍在流内。
func (ctx *explodeCtx) wasteBits(nbits int) bool {
	if nbits <= ctx.extraBits {
		ctx.extraBits -= nbits
		ctx.bitBuff >>= uint(nbits)
		return true
	}
	ctx.bitBuff >>= uint(ctx.extraBits)
	if ctx.inPos >= len(ctx.input) {
		return false
	}
	ctx.bitBuff |= uint32(ctx.input[ctx.inPos]) << 8
	ctx.inPos++
	ctx.bitBuff >>= uint(nbits - ctx.extraBits)
	// C 里按无符号回绕计算 (extra - nbits) + 8；调整顺序避免下溢。
	ctx.extraBits = (ctx.extraBits + 8) - nbits
	return true
}

func decodeLit(ctx *explodeCtx) int {
	if ctx.bitBuff&1 != 0 {
		if !ctx.wasteBits(1) {
			return litError
		}
		lengthCode := int(ctx.lengthCodes[ctx.bitBuff&0xFF])

		if !ctx.wasteBits(lenBits[lengthCode]) {
			return litError
		}
		extraBitsCount := exLenBits[lengthCode]
		code := lengthCode
		if extraBitsCount != 0 {
			extra := int(ctx.bitBuff & uint32(1<<extraBitsCount-1))
			// 只有「流结束 且 不是特殊终止组合」才算解码失败。
			if !ctx.wasteBits(extraBitsCount) && (lengthCode+extra) != 0x10E {
				return litError
			}
			code = lenBase[lengthCode] + extra
		}
		return code + 0x100
	}

	if !ctx.wasteBits(1) {
		return litError
	}

	if ctx.ctype == cmpBinary {
		b := int(ctx.bitBuff & 0xFF)
		if !ctx.wasteBits(8) {
			return litError
		}
		return b
	}

	// ASCII 模式
	asc := ctx.asc
	var value int
	if ctx.bitBuff&0xFF != 0 {
		v := int(asc.l1[ctx.bitBuff&0xFF])
		if v == 0xFF {
			if ctx.bitBuff&0x3F != 0 {
				if !ctx.wasteBits(4) {
					return litError
				}
				v = int(asc.l2[ctx.bitBuff&0xFF])
			} else {
				if !ctx.wasteBits(6) {
					return litError
				}
				v = int(asc.l3[ctx.bitBuff&0x7F])
			}
		}
		value = v
	} else {
		if !ctx.wasteBits(8) {
			return litError
		}
		value = int(asc.l4[ctx.bitBuff&0xFF])
	}

	bits := int(asc.chBitsAsc[value])
	if !ctx.wasteBits(bits) {
		return litError
	}
	return value
}

func decodeDist(ctx *explodeCtx, repLength int) int {
	distPosCode := int(ctx.distPosCodes[ctx.bitBuff&0xFF])
	distPosBits := distBits[distPosCode]
	if !ctx.wasteBits(distPosBits) {
		return 0
	}

	var distance int
	if repLength == 2 {
		d := distPosCode<<2 | int(ctx.bitBuff&0x03)
		if !ctx.wasteBits(2) {
			return 0
		}
		distance = d
	} else {
		d := distPosCode<<ctx.dsizeBits | int(ctx.bitBuff&uint32(ctx.dsizeMask))
		if !ctx.wasteBits(ctx.dsizeBits) {
			return 0
		}
		distance = d
	}
	return distance + 1
}

// Explode 解压 PKWARE DCL 流；最多 maxOut 字节被追加进 out。
func Explode(input []byte, out *bytes.Buffer, maxOut int) error {
	if len(input) <= 4 {
		return fmt.Errorf("mpq: pkware 流过短")
	}
	ctype := input[0]
	dsizeBits := int(input[1])
	if dsizeBits < 4 || dsizeBits > 6 {
		return fmt.Errorf("mpq: pkware 非法的字典大小")
	}
	if ctype != cmpBinary && ctype != cmpASCII {
		return fmt.Errorf("mpq: pkware 非法的模式")
	}
	dsizeMask := 0xFFFF >> (16 - dsizeBits)

	ctx := &explodeCtx{
		input:        input,
		dsizeBits:    dsizeBits,
		dsizeMask:    dsizeMask,
		ctype:        ctype,
		lengthCodes:  genDecodeTabs(lenCode[:], lenBits[:], 0x10),
		distPosCodes: genDecodeTabs(distCode[:], distBits[:], 0x40),
		asc:          nil,
	}
	if ctype == cmpASCII {
		ctx.asc = genAscTabs()
	}
	ctx.bitBuff = uint32(input[2])
	ctx.inPos = 3
	return expand(ctx, out, maxOut)
}

func expand(ctx *explodeCtx, out *bytes.Buffer, maxOut int) error {
	// 0x2204 字节环形缓冲区；解压从 0x1000 处开始写，越过 0x2000 冲刷 0x1000。
	ring := make([]byte, 0x2204)
	outputPos := 0x1000
	limit := out.Len() + maxOut

	for {
		literal := decodeLit(ctx)
		if literal >= litEnd {
			if literal == litEnd {
				break
			}
			return fmt.Errorf("mpq: pkware 流已损坏")
		}

		if literal >= 0x100 {
			repLength := literal - 0xFE
			minusDist := decodeDist(ctx, repLength)
			if minusDist == 0 {
				return fmt.Errorf("mpq: pkware 流已损坏（距离）")
			}
			if minusDist > outputPos {
				return fmt.Errorf("mpq: pkware 距离越过起点")
			}

			target := outputPos
			source := outputPos - minusDist
			outputPos += repLength
			// 逐字节复制：序列可能重叠（游程式重复）。
			t, s := target, source
			for t < outputPos {
				ring[t] = ring[s]
				t++
				s++
			}
		} else {
			ring[outputPos] = byte(literal)
			outputPos++
		}

		if outputPos >= 0x2000 {
			take := min(0x1000, limit-out.Len())
			if take > 0 {
				out.Write(ring[0x1000 : 0x1000+take])
			}
			keep := outputPos - 0x1000
			copy(ring, ring[0x1000:0x1000+keep])
			outputPos = keep
		}
	}

	// 最终冲刷：输出所有待处理内容（受 maxOut 截断）。
	take := min(outputPos-0x1000, limit-out.Len())
	if take > 0 {
		out.Write(ring[0x1000 : 0x1000+take])
	}
	return nil
}
