// 行 RLE 编解码（walkflagrle.cpp / walkheightrle.cpp）。
//
// 两种行 RLE 共用同一骨架（反汇编 CWalkFlagRLE::Decompress@0x10066A80、
// CWalkHeightRLE::Decompress@0x1006A3A0）：
//
//	u32 offsets[scale]          // 每行：bit31=字面行，低 31 位=行数据在 payload 内偏移
//	payload…
//	字面行: flag=scale/2 字节（低/高 nibble 各一格）；height=scale 个裸 f32
//	RLE 行 : flag 每记录 {u8 run, u8 val&0xf}；height 每记录 {u8 run, f32 val}
//
// 校验（与官方一致，超界/行长不符/展开列数≠scale 均失败）：start+count 不得
// 越界；字面行 count 必须恰为 scale/2（flag）或 scale*4（height）。
package fscollide

import (
	"encoding/binary"
	"fmt"
	"math"
)

// rowSpan 计算第 row 行数据在 payload 内的 [start,count)；行偏移差为负或越界即错。
func rowSpan(scale, row int, data []byte) (start, count, payload int, literal bool, err error) {
	table := scale * 4
	if len(data) < table {
		return 0, 0, 0, false, fmt.Errorf("fscollide: RLE size %d < offset table %d", len(data), table)
	}
	payload = len(data) - table
	off := binary.LittleEndian.Uint32(data[row*4:])
	literal = off&0x80000000 != 0
	start = int(off & 0x7fffffff)
	if row == scale-1 {
		count = payload - start
	} else {
		next := binary.LittleEndian.Uint32(data[(row+1)*4:]) & 0x7fffffff
		count = int(next) - start
	}
	if start < 0 || count < 0 || start+count > payload {
		return 0, 0, 0, false, fmt.Errorf("fscollide: RLE row %d range (%d+%d > %d)", row, start, count, payload)
	}
	return start, count, payload, literal, nil
}

// DecodeFlagRLE 把 4bit 标记 RLE 解为 scale×scale 字节阵（值 0..15）。
func DecodeFlagRLE(scale int, data []byte) ([]byte, error) {
	out := make([]byte, scale*scale)
	for row := 0; row < scale; row++ {
		start, count, _, literal, err := rowSpan(scale, row, data)
		if err != nil {
			return nil, err
		}
		seg := data[scale*4+start : scale*4+start+count]
		dst := out[row*scale : (row+1)*scale]
		if literal { // 字面行：nibble 对（先低后高）
			if count != scale/2 {
				return nil, fmt.Errorf("fscollide: flag RLE literal row len %d != %d", count, scale/2)
			}
			for i := 0; i < scale/2; i++ {
				dst[2*i] = seg[i] & 0xf
				dst[2*i+1] = (seg[i] >> 4) & 0xf
			}
		} else { // RLE 记录 {u8 run, u8 val}
			col := 0
			for i := 0; i < len(seg); i += 2 {
				if i+1 >= len(seg) {
					return nil, fmt.Errorf("fscollide: flag RLE row %d truncated record", row)
				}
				run := int(seg[i])
				if run <= 0 || col+run > scale {
					return nil, fmt.Errorf("fscollide: flag RLE row %d overrun col=%d run=%d", row, col, run)
				}
				val := seg[i+1] & 0xf
				for k := 0; k < run; k++ {
					dst[col] = val
					col++
				}
			}
			if col != scale {
				return nil, fmt.Errorf("fscollide: flag RLE row %d col=%d != %d", row, col, scale)
			}
		}
	}
	return out, nil
}

// DecodeHeightRLE 把 f32 高度 RLE 解为 scale×scale 浮点阵。
func DecodeHeightRLE(scale int, data []byte) ([]float32, error) {
	out := make([]float32, scale*scale)
	for row := 0; row < scale; row++ {
		start, count, _, literal, err := rowSpan(scale, row, data)
		if err != nil {
			return nil, err
		}
		seg := data[scale*4+start : scale*4+start+count]
		dst := out[row*scale : (row+1)*scale]
		if literal { // 字面行：裸 f32
			if count != scale*4 {
				return nil, fmt.Errorf("fscollide: height RLE literal row len %d != %d", count, scale*4)
			}
			for i := 0; i < scale; i++ {
				dst[i] = math.Float32frombits(binary.LittleEndian.Uint32(seg[i*4:]))
			}
		} else { // RLE 记录 {u8 run, f32 val}
			col := 0
			for i := 0; i < len(seg); i += 5 {
				if i+5 > len(seg) {
					return nil, fmt.Errorf("fscollide: height RLE row %d truncated record", row)
				}
				run := int(seg[i])
				if run <= 0 || col+run > scale {
					return nil, fmt.Errorf("fscollide: height RLE row %d overrun col=%d run=%d", row, col, run)
				}
				val := math.Float32frombits(binary.LittleEndian.Uint32(seg[i+1:]))
				for k := 0; k < run; k++ {
					dst[col] = val
					col++
				}
			}
			if col != scale {
				return nil, fmt.Errorf("fscollide: height RLE row %d col=%d != %d", row, col, scale)
			}
		}
	}
	return out, nil
}
