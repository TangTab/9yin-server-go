package fssolorobot

import "errors"

// QuickLZ level-2 解压器，复刻 FsSoloRobot.dll 内嵌的 quicklz
// （反汇编终局：_qlz_size_header@0x10058380 / _qlz_size_decompressed@
//  0x100581f0 / _qlz_size_compressed@0x100582e0 / _qlz_decompress@
//  0x10058d60 + level-2 核 0x10058e80 + 哈希插入 0x10059250）。
//
// 头布局（两种，flags 决定）：
//
//	flags&2 == 2（9 字节头）：[0]=flags [1..5)=csize u32LE [5..9)=dsize u32LE
//	flags&2 == 0（3 字节头）：[0]=flags [1]=csize u8     [2]=dsize u8
//	flags&1 == 1：数据压缩（level-2 位流）；否则原样存储
//	（memcpy(src+header, dst, dsize)）。
//
// level-2 位流（官方 0x10058e80 核逐指令定案）：
//
//	标签字 cword：进入循环时 cword==1 则 cword = u32LE@src（标签位流，
//	  LSB 先），src += 4；
//	令牌位 cword&1：
//	  0 = 字面量连跑：n = min(cword 尾零数, 4)（批查表 [4,0,1,0,2,0,1,
//	     0,3,0,1,0,2,0,1,0]，下标恒为偶——每 0 位 1 个字面量，0000 批 4）；
//	     dst 距末尾 >10 时走快速道（u32 拷贝后 dst/src 各进 n），否则进
//	     尾道（逐字节，cword==1 时仅 src+=4 且 cword=0x80000000 哨兵，
//	     标签位耗完即尾）；
//	  1 = 匹配：control = u32LE@src（窥视）；hash = (control>>4)&0xFFF；
//	     (control&0xF)!=0 → matchlen=(control&0xF)+2、src+=2，否则
//	     matchlen=src[2]、src+=3；从哈希表指针位置按 3 步 u32 前向重叠
//	     拷贝 matchlen 字节；cword >>= 1。
//	哈希表（scratch）：h = ((v>>12)^v)&0xFFF，v = 位置处 3 字节 u32LE；
//	  每产出一段（字面量后到 dst-3 / 匹配后到 dst-matchlen）从计数器
//	  惰性回填 scratch[h] = 输出位置指针（初值 dst0-1）。
var (
	// ErrQLZShort 输入流过短（头都不足）。
	ErrQLZShort = errors.New("qlz: stream too short")
	// ErrQLZTruncated 压缩体比声明的解压尺寸短。
	ErrQLZTruncated = errors.New("qlz: truncated stream")
)

// qlzSizeHeader 官方 qlz_size_header：flags&2 ? 9 : 3。
func qlzSizeHeader(src []byte) (int, error) {
	if len(src) < 1 {
		return 0, ErrQLZShort
	}
	if src[0]&2 != 0 {
		return 9, nil
	}
	return 3, nil
}

// QLZSizeDecompressed 官方 qlz_size_decompressed：flags&2 →
// u32LE@+5，否则 u8@+2。
func QLZSizeDecompressed(src []byte) (int, error) {
	if len(src) < 3 {
		return 0, ErrQLZShort
	}
	if src[0]&2 != 0 {
		if len(src) < 9 {
			return 0, ErrQLZShort
		}
		return int(leU32(src[5:9])), nil
	}
	return int(src[2]), nil
}

// qlzSizeCompressed 官方 qlz_size_compressed：flags&2 → u32LE@+1，
// 否则 u8@+1。
func qlzSizeCompressed(src []byte) (int, error) {
	if len(src) < 2 {
		return 0, ErrQLZShort
	}
	if src[0]&2 != 0 {
		if len(src) < 9 {
			return 0, ErrQLZShort
		}
		return int(leU32(src[1:5])), nil
	}
	return int(src[1]), nil
}

func leU32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// qlzLiteralTable 官方 0x100c9868 的 16 项字面量批表：
// 偶下标 2^k → k（尾零数），奇下标不可达（令牌位恒为 0 才进字面量）。
var qlzLiteralTable = [16]byte{4, 0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0}

// qlzScratch 是官方 0x1030 字节解压暂存区里被解压器用到的部分：
// 4096 槽指针表（指向输出缓冲内位置，0x4000 字节）+ 槽占用标志
// （解压侧只写不读，Go 版省略）+ @0x5000 流计数（无流模式不用）。
// 值语义用下标代替指针。
type qlzScratch struct {
	hash [4096]int // 输出位置下标；-1 = 空
}

// qlzHash 官方哈希（0x10058d20）：((v>>12)^v)&0xFFF，v=3 字节 u32LE。
func qlzHash(v uint32) uint32 { return ((v >> 12) ^ v) & 0xFFF }

func (s *qlzScratch) insert(out []byte, pos int) {
	if pos+3 > len(out) {
		return // 官方读 3 字节越界容忍；Go 版就近截断为不插入
	}
	v := uint32(out[pos]) | uint32(out[pos+1])<<8 | uint32(out[pos+2])<<16
	s.hash[qlzHash(v)] = pos
}

// lazyHash 官方 0x100591d0：把 counter 推进到 threshold，逐位置插入哈希。
func (s *qlzScratch) lazyHash(out []byte, counter *int, threshold int) {
	for *counter < threshold {
		*counter++
		s.insert(out, *counter)
	}
}

// QLZDecompress 官方 qlz_decompress(src, dst, scratch)：返回解压
// 字节数（== qlz_size_decompressed）。dst 长度必须 ≥ dsize。
func QLZDecompress(src []byte, dst []byte) (int, error) {
	dsize, err := QLZSizeDecompressed(src)
	if err != nil {
		return 0, err
	}
	if dsize > len(dst) {
		return 0, ErrQLZTruncated
	}
	if src[0]&1 == 0 {
		// 官方存储路径：memcpy(dst, src+header, dsize)。
		hdr, err := qlzSizeHeader(src)
		if err != nil {
			return 0, err
		}
		if hdr+dsize > len(src) {
			return 0, ErrQLZTruncated
		}
		copy(dst[:dsize], src[hdr:hdr+dsize])
		return dsize, nil
	}
	hdr, err := qlzSizeHeader(src)
	if err != nil {
		return 0, err
	}
	n, err := qlzDecompressCore(src[hdr:], dst[:dsize])
	if err != nil {
		return 0, err
	}
	_ = n
	return dsize, nil
}

// qlzDecompressCore 官方 level-2 核（0x10058e80）。
// out 为整段输出（哈希可回看），产出量以 dsize 为准。
func qlzDecompressCore(stream, out []byte) (int, error) {
	var scratch qlzScratch
	for i := range scratch.hash {
		scratch.hash[i] = -1
	}
	src := 0              // stream 游标
	dst := 0              // out 游标
	dsize := len(out)     // 产出上限（官方 last_dst = dst0+dsize-1）
	lastDst := dsize - 1  // 官方 [ebp-0x20]
	fastLimit := lastDst - 10 // 官方 [ebp-0x38]（负值则恒走尾道）
	counter := -1         // 官方 [ebp-0x44] = dst0 - 1
	cword := uint32(1)

	peek32 := func(at int) uint32 {
		// 官方直接 u32 窥视（容忍越界读）；Go 版就近补零等价化。
		var b [4]byte
		for i := 0; i < 4; i++ {
			if at+i < len(stream) {
				b[i] = stream[at+i]
			}
		}
		return leU32(b[:])
	}
	read32 := func(at int) uint32 { return peek32(at) }

	if lastDst < 0 { // dsize==0：尾道条件恒假，官方不读任何流字节
		return 0, nil
	}

	for {
		if cword == 1 {
			if src+4 > len(stream) {
				return 0, ErrQLZTruncated
			}
			cword = read32(src)
			src += 4
		}
		control := peek32(src)

		if cword&1 != 0 { // 匹配令牌
			cword >>= 1
			hash := (control >> 4) & 0xFFF
			var matchlen int
			if control&0xF != 0 {
				matchlen = int(control&0xF) + 2
				src += 2
			} else {
				if src+3 > len(stream) {
					return 0, ErrQLZTruncated
				}
				matchlen = int(stream[src+2])
				src += 3
			}
			pos := scratch.hash[hash]
			if pos < 0 || pos >= dst {
				return 0, ErrQLZTruncated // 官方依赖压缩器保证；坏流防御
			}
			// 官方 0x10059170：3 步进 u32 前向重叠拷贝（前瞻第 4 字节
			// 会被后续令牌真实覆写，忠实按 u32 读写）。
			for i := 0; i < matchlen; i += 3 {
				var dw [4]byte
				for j := 0; j < 4; j++ {
					if pos+i+j < dsize {
						dw[j] = out[pos+i+j]
					}
				}
				for j := 0; j < 4; j++ {
					if dst+i+j < dsize {
						out[dst+i+j] = dw[j]
					}
				}
			}
			dst += matchlen
			scratch.lazyHash(out, &counter, dst-matchlen)
			counter = dst - 1
			continue
		}

		// 字面量令牌。
		if dst < fastLimit { // 快速道（官方 0x10058fc5：dst < last_dst-0xA）
			n := int(qlzLiteralTable[cword&0xF])
			if n == 0 {
				return 0, ErrQLZTruncated // 防御：奇下标不可达
			}
			if src+n > len(stream) {
				return 0, ErrQLZTruncated
			}
			// 官方 u32 拷贝（可前瞻覆写，随后覆写校正）；dst+3 距末尾
			// ≥10 字节保证不越界。Go 版逐字节拷等价产出。
			for i := 0; i < n; i++ {
				if dst+i >= dsize {
					return 0, ErrQLZTruncated
				}
				out[dst+i] = stream[src+i]
			}
			cword >>= uint(n)
			dst += n
			src += n
			scratch.lazyHash(out, &counter, dst-3)
			continue
		}

		// 尾道（官方 0x10059029）：距末尾 ≤10 字节后逐字面量收尾。
		for dst <= lastDst {
			if cword == 1 {
				src += 4 // 官方：跳 4 字节标签并以 0x80000000 哨兵续命
				cword = 0x80000000
			}
			if src >= len(stream) {
				return 0, ErrQLZTruncated
			}
			out[dst] = stream[src]
			dst++
			src++
			cword >>= 1
		}
		scratch.lazyHash(out, &counter, lastDst-3)
		return dsize, nil
	}
}
