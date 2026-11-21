package entergen

// mode-0x47 全字面量 LZ 包装器（b-line wrapProfile0x28As47Literal 移植，
// op 字节参数化）：字面量 token（k∈{1..4} 字节，n 位型 0010/0100/1000/
// 0000，耗 k bit）+ 尾 11 字节逐字节 1bit；每 flags 词在已用位最高位
// 置哨兵 1。零池依赖——解码输出恒等于原文（lzdec 往返验证）。

import "encoding/binary"

// WrapAs47Literal 把明文体包装为 [op][0x47][len-1 u32][outlen u32][位流]。
func WrapAs47Literal(op byte, body []byte) []byte {
	var out []byte
	var wordBits []bool
	var wordData []byte
	flush := func() {
		w := uint32(0)
		for i, b := range wordBits {
			if b {
				w |= 1 << uint(i)
			}
		}
		w |= 1 << uint(len(wordBits)) // 哨兵位
		out = binary.LittleEndian.AppendUint32(out, w)
		out = append(out, wordData...)
		wordBits, wordData = nil, nil
	}
	emitLiteral := func(k int, chunk []byte) {
		need := k
		if len(wordBits)+need > 31 {
			flush()
		}
		var n uint8
		switch k {
		case 1:
			n = 0b0010
		case 2:
			n = 0b0100
		case 3:
			n = 0b1000
		default:
			n = 0
		}
		for j := 0; j < 4; j++ {
			wordBits = append(wordBits, n&(1<<uint(j)) != 0)
		}
		wordData = append(wordData, chunk...)
	}
	emitTailByte := func(b byte) {
		if len(wordBits)+1 > 31 {
			flush()
		}
		wordBits = append(wordBits, false)
		wordData = append(wordData, b)
	}
	mainLen := len(body) - 0xb
	if mainLen < 0 {
		mainLen = len(body)
	}
	for i := 0; i < mainLen; {
		k := 4
		if r := mainLen - i; r < k {
			k = r
		}
		emitLiteral(k, body[i:i+k])
		i += k
	}
	for _, b := range body[mainLen:] {
		emitTailByte(b)
	}
	flush()
	frame := make([]byte, 0, len(out)+10)
	frame = append(frame, op, 0x47)
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(out)+9)) // len-1
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)))  // outlen
	frame = append(frame, out...)
	return frame
}

// WrapAs46Plain mode-0x46 纯 memcpy 明文包装：[op][0x46][len-1][体长][体]。
func WrapAs46Plain(op byte, body []byte) []byte {
	frame := make([]byte, 0, len(body)+10)
	frame = append(frame, op, 0x46)
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)+9))
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)))
	frame = append(frame, body...)
	return frame
}
