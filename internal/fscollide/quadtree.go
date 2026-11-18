// 四叉树压缩格式（walkbitcompress/walkflagcompress/walkheightcompress/
// heightcompress.cpp）。四类树共用节点骨架（反汇编 GetBitData@0x10063530、
// GetFlagData@0x10065480、GetHeightData@0x10068D80、DecompressData@0x10055B40）：
//
//	节点首字节：低 nibble = 类型
//	  0 分支：高 nibble = 子树尺寸宽度（1=u8/2=u16/4=u32），随后 4 个该宽度的
//	         子树尺寸；子树按 TL,TR,BL,BR 顺序紧挨存放，第 n 子起点 =
//	         本子树数据首 + 前面子树尺寸之和
//	  1 全叶：高 nibble 必须等于本节点 scale；随后是 scale² 格原始数据
//	  2 统一：值直接放高 nibble（bit/flag）或跟随的定宽字段
//
// 值编码差异：
//
//	CWalkBitCompress    统一=0x10 位；全叶=scale²/8 字节位图（第 i 位=(i&7)）
//	CWalkFlagCompress   统一=高 nibble(0..15)；全叶=scale²/2 字节 nibble 对
//	CWalkHeightCompress 头部 {u32 min, u32 delta}，offset 从 8 起；统一/全叶
//	                     值为 u16：0xffff→1e8(空高)、0xfffe→1e6、
//	                     否则 h=(min+q*delta)*0.001（毫米量化）
//	CHeightCompress     统一=跟随 f32；全叶=scale² 裸 f32；另有 3=u8 量化叶
//	                     (f32 base + scale² u8, h=base+q*0.001)、4=u16 量化叶
package fscollide

import (
	"encoding/binary"
	"fmt"
	"math"
)

// heightNull / heightSpecial2 对应官方 0x100d6984=1e8、0x100d72ac=1e6。
const (
	heightNull    = float32(100000000.0)
	heightSpecial = float32(1000000.0)
)

// heightIsNull 复刻 height_is_null@0x100590B0：h > 1e8。
func heightIsNull(h float32) bool { return h > heightNull }

// decodeHeightValue 复刻 CWalkHeightCompress 的 u16 量化值解码。
func decodeHeightValue(q uint16, min, delta uint32) float32 {
	switch q {
	case 0xffff:
		return heightNull
	case 0xfffe:
		return heightSpecial
	}
	return float32(int64(min)+int64(q)*int64(delta)) * 0.001
}

// readChildSizes 读分支节点的 4 个子树尺寸，返回宽度字节数。
func readChildSizes(data []byte, off int) (sizes [4]int, w int, err error) {
	if off >= len(data) {
		return sizes, 0, fmt.Errorf("fscollide: quadtree branch header OOB")
	}
	switch data[off] >> 4 {
	case 1:
		if off+4+4 > len(data) {
			return sizes, 1, fmt.Errorf("fscollide: quadtree child sizes OOB")
		}
		for i := 0; i < 4; i++ {
			sizes[i] = int(data[off+1+i])
		}
		return sizes, 1, nil
	case 2:
		if off+1+8 > len(data) {
			return sizes, 2, fmt.Errorf("fscollide: quadtree child sizes OOB")
		}
		for i := 0; i < 4; i++ {
			sizes[i] = int(binary.LittleEndian.Uint16(data[off+1+i*2:]))
		}
		return sizes, 2, nil
	case 4:
		if off+1+16 > len(data) {
			return sizes, 4, fmt.Errorf("fscollide: quadtree child sizes OOB")
		}
		for i := 0; i < 4; i++ {
			sizes[i] = int(binary.LittleEndian.Uint32(data[off+1+i*4:]))
		}
		return sizes, 4, nil
	}
	return sizes, 0, fmt.Errorf("fscollide: quadtree bad child width %d", data[off]>>4)
}

// quadChildBase 返回第 q 子（0=TL,1=TR,2=BL,3=BR）的数据首偏移。
func quadChildBase(sizes [4]int, q int) int {
	base := 0
	for i := 0; i < q; i++ {
		base += sizes[i]
	}
	return base
}

// QuadGetBit 复刻 CWalkBitCompress::GetBit 的位查询。
func QuadGetBit(scale, row, col int, data []byte) (bool, error) {
	off := 0
	for {
		if off >= len(data) {
			return false, fmt.Errorf("fscollide: bit quadtree offset OOB")
		}
		tag := data[off] & 0xf
		switch tag {
		case 2:
			return data[off]&0x10 != 0, nil
		case 1:
			if int(data[off]>>4) != scale {
				return false, fmt.Errorf("fscollide: bit quadtree leaf scale mismatch")
			}
			if off+1+(scale*scale+7)/8 > len(data) {
				return false, fmt.Errorf("fscollide: bit quadtree leaf OOB")
			}
			idx := row*scale + col
			return data[off+1+idx/8]&(1<<(idx&7)) != 0, nil
		case 0:
			sizes, w, err := readChildSizes(data, off)
			if err != nil {
				return false, err
			}
			half := scale / 2
			if half == 0 {
				return false, fmt.Errorf("fscollide: bit quadtree scale 0")
			}
			q := 0
			if row >= half {
				q += 2
				row -= half
			}
			if col >= half {
				q++
				col -= half
			}
			off = off + 1 + 4*w + quadChildBase(sizes, q)
			scale = half
		default:
			return false, fmt.Errorf("fscollide: bit quadtree bad tag %d", tag)
		}
	}
}

// QuadGetFlag 复刻 CWalkFlagCompress::GetFlag 的 4bit 查询。
func QuadGetFlag(scale, row, col int, data []byte) (byte, error) {
	off := 0
	for {
		if off >= len(data) {
			return 0, fmt.Errorf("fscollide: flag quadtree offset OOB")
		}
		tag := data[off] & 0xf
		switch tag {
		case 2:
			return (data[off] >> 4) & 0xf, nil
		case 1:
			if int(data[off]>>4) != scale {
				return 0, fmt.Errorf("fscollide: flag quadtree leaf scale mismatch")
			}
			need := scale * scale / 2
			if off+1+need > len(data) {
				return 0, fmt.Errorf("fscollide: flag quadtree leaf OOB")
			}
			idx := row*scale + col
			b := data[off+1+idx/2]
			if idx&1 == 0 {
				return b & 0xf, nil
			}
			return (b >> 4) & 0xf, nil
		case 0:
			sizes, w, err := readChildSizes(data, off)
			if err != nil {
				return 0, err
			}
			half := scale / 2
			if half == 0 {
				return 0, fmt.Errorf("fscollide: flag quadtree scale 0")
			}
			q := 0
			if row >= half {
				q += 2
				row -= half
			}
			if col >= half {
				q++
				col -= half
			}
			off = off + 1 + 4*w + quadChildBase(sizes, q)
			scale = half
		default:
			return 0, fmt.Errorf("fscollide: flag quadtree bad tag %d", tag)
		}
	}
}

// QuadGetHeight 复刻 CWalkHeightCompress::GetHeight 的量化高度查询。
func QuadGetHeight(scale, row, col int, data []byte) (float32, error) {
	if len(data) < 8 {
		return 0, fmt.Errorf("fscollide: height quadtree < 8")
	}
	min := binary.LittleEndian.Uint32(data)
	delta := binary.LittleEndian.Uint32(data[4:])
	off := 8
	for {
		if off >= len(data) {
			return 0, fmt.Errorf("fscollide: height quadtree offset OOB")
		}
		tag := data[off] & 0xf
		switch tag {
		case 2:
			if off+1+2 > len(data) {
				return 0, fmt.Errorf("fscollide: height quadtree uniform OOB")
			}
			return decodeHeightValue(binary.LittleEndian.Uint16(data[off+1:]), min, delta), nil
		case 1:
			if int(data[off]>>4) != scale {
				return 0, fmt.Errorf("fscollide: height quadtree leaf scale mismatch")
			}
			if off+1+scale*scale*2 > len(data) {
				return 0, fmt.Errorf("fscollide: height quadtree leaf OOB")
			}
			q := binary.LittleEndian.Uint16(data[off+1+(row*scale+col)*2:])
			return decodeHeightValue(q, min, delta), nil
		case 0:
			sizes, w, err := readChildSizes(data, off)
			if err != nil {
				return 0, err
			}
			half := scale / 2
			if half == 0 {
				return 0, fmt.Errorf("fscollide: height quadtree scale 0")
			}
			q := 0
			if row >= half {
				q += 2
				row -= half
			}
			if col >= half {
				q++
				col -= half
			}
			off = off + 1 + 4*w + quadChildBase(sizes, q)
			scale = half
		default:
			return 0, fmt.Errorf("fscollide: height quadtree bad tag %d", tag)
		}
	}
}

// TerrainHeightQuad 复刻 CHeightCompress::Decompress：地形高度四叉树整树
// 解码为 scale×scale f32 网格（按 [first*scale+second] 即与 CTerrainHeight::
// GetValue(col,row)=values[col*scale+row] 同序）。nodeData 的 row/col 基准由
// 递归传入。
func TerrainHeightQuad(scale int, data []byte) ([]float32, error) {
	out := make([]float32, scale*scale)
	off := 0
	if err := terrainQuadNode(scale, 0, 0, scale, data, &off, out); err != nil {
		return nil, err
	}
	return out, nil
}

func terrainQuadNode(scale, baseA, baseB, stride int, data []byte, off *int, out []float32) error {
	if *off >= len(data) {
		return fmt.Errorf("fscollide: terrain quad offset OOB")
	}
	tag := data[*off] & 0xf
	put := func(a, b int, v float32) { out[(baseA+a)*stride+baseB+b] = v }
	switch tag {
	case 2: // 统一 f32
		if *off+1+4 > len(data) {
			return fmt.Errorf("fscollide: terrain quad uniform OOB")
		}
		v := math.Float32frombits(binary.LittleEndian.Uint32(data[*off+1:]))
		for a := 0; a < scale; a++ {
			for b := 0; b < scale; b++ {
				put(a, b, v)
			}
		}
		*off += 5
	case 3: // u8 量化全叶：f32 base + scale² u8
		if *off+1+4+scale*scale > len(data) {
			return fmt.Errorf("fscollide: terrain quad u8 leaf OOB")
		}
		base := math.Float32frombits(binary.LittleEndian.Uint32(data[*off+1:]))
		p := *off + 5
		for a := 0; a < scale; a++ {
			for b := 0; b < scale; b++ {
				put(a, b, base+float32(data[p+a*scale+b])*0.001)
			}
		}
		*off += 5 + scale*scale
	case 4: // u16 量化全叶
		if *off+1+4+scale*scale*2 > len(data) {
			return fmt.Errorf("fscollide: terrain quad u16 leaf OOB")
		}
		base := math.Float32frombits(binary.LittleEndian.Uint32(data[*off+1:]))
		p := *off + 5
		for a := 0; a < scale; a++ {
			for b := 0; b < scale; b++ {
				q := binary.LittleEndian.Uint16(data[p+(a*scale+b)*2:])
				put(a, b, base+float32(q)*0.001)
			}
		}
		*off += 5 + scale*scale*2
	case 1: // 裸 f32 全叶
		if int(data[*off]>>4) != scale {
			return fmt.Errorf("fscollide: terrain quad leaf scale mismatch")
		}
		if *off+1+scale*scale*4 > len(data) {
			return fmt.Errorf("fscollide: terrain quad f32 leaf OOB")
		}
		p := *off + 1
		for a := 0; a < scale; a++ {
			for b := 0; b < scale; b++ {
				put(a, b, math.Float32frombits(binary.LittleEndian.Uint32(data[p+(a*scale+b)*4:])))
			}
		}
		*off += 1 + scale*scale*4
	case 0: // 分支：无尺寸表，四子顺序递归消费（TL,TR,BL,BR）
		half := scale / 2
		if half == 0 {
			return fmt.Errorf("fscollide: terrain quad scale 0")
		}
		*off++
		quads := [4][2]int{{0, 0}, {0, half}, {half, 0}, {half, half}}
		for q := 0; q < 4; q++ {
			if err := terrainQuadNode(half, baseA+quads[q][0], baseB+quads[q][1], stride, data, off, out); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("fscollide: terrain quad bad tag %d", tag)
	}
	return nil
}

// QuadHeightDecompress 复刻 CWalkHeightCompress::Decompress：整树解码为
// scale² f32 阵（walk 文件各高度数组）。头部 {u32 min, u32 delta}。
func QuadHeightDecompress(scale int, data []byte) ([]float32, error) {
	out := make([]float32, scale*scale)
	if len(data) < 8 {
		return nil, fmt.Errorf("fscollide: height quad < 8")
	}
	min := binary.LittleEndian.Uint32(data)
	delta := binary.LittleEndian.Uint32(data[4:])
	off := 8
	if err := whqNode(scale, 0, 0, scale, data, &off, out, min, delta); err != nil {
		return nil, err
	}
	return out, nil
}

func whqNode(scale, baseA, baseB, stride int, data []byte, off *int, out []float32, min, delta uint32) error {
	if *off >= len(data) {
		return fmt.Errorf("fscollide: height quad offset OOB")
	}
	tag := data[*off] & 0xf
	switch tag {
	case 2: // 统一 u16
		if *off+1+2 > len(data) {
			return fmt.Errorf("fscollide: height quad uniform OOB")
		}
		v := decodeHeightValue(binary.LittleEndian.Uint16(data[*off+1:]), min, delta)
		for a := 0; a < scale; a++ {
			for b := 0; b < scale; b++ {
				out[(baseA+a)*stride+baseB+b] = v
			}
		}
		*off += 3
	case 1: // 全叶 u16[scale²]
		if int(data[*off]>>4) != scale {
			return fmt.Errorf("fscollide: height quad leaf scale mismatch")
		}
		if *off+1+scale*scale*2 > len(data) {
			return fmt.Errorf("fscollide: height quad leaf OOB")
		}
		p := *off + 1
		for a := 0; a < scale; a++ {
			for b := 0; b < scale; b++ {
				q := binary.LittleEndian.Uint16(data[p+(a*scale+b)*2:])
				out[(baseA+a)*stride+baseB+b] = decodeHeightValue(q, min, delta)
			}
		}
		*off += 1 + scale*scale*2
	case 0: // 分支
		sizes, w, err := readChildSizes(data, *off)
		if err != nil {
			return err
		}
		childOff := *off + 1 + 4*w
		half := scale / 2
		if half == 0 {
			return fmt.Errorf("fscollide: height quad scale 0")
		}
		quads := [4][2]int{{0, 0}, {0, half}, {half, 0}, {half, half}}
		for q := 0; q < 4; q++ {
			sub := childOff + quadChildBase(sizes, q)
			if err := whqNode(half, baseA+quads[q][0], baseB+quads[q][1], stride, data, &sub, out, min, delta); err != nil {
				return err
			}
		}
		*off = childOff + sizes[0] + sizes[1] + sizes[2] + sizes[3]
	default:
		return fmt.Errorf("fscollide: height quad bad tag %d", tag)
	}
	return nil
}

// QuadFlagDecompress 复刻 CWalkFlagCompress::Decompress：整树解码为
// scale² 字节阵（值 0..15）。
func QuadFlagDecompress(scale int, data []byte) ([]byte, error) {
	out := make([]byte, scale*scale)
	off := 0
	if err := qfNode(scale, 0, 0, scale, data, &off, out); err != nil {
		return nil, err
	}
	return out, nil
}

func qfNode(scale, baseA, baseB, stride int, data []byte, off *int, out []byte) error {
	if *off >= len(data) {
		return fmt.Errorf("fscollide: flag quad offset OOB")
	}
	tag := data[*off] & 0xf
	switch tag {
	case 2:
		v := (data[*off] >> 4) & 0xf
		for a := 0; a < scale; a++ {
			for b := 0; b < scale; b++ {
				out[(baseA+a)*stride+baseB+b] = v
			}
		}
		*off++
	case 1:
		if int(data[*off]>>4) != scale {
			return fmt.Errorf("fscollide: flag quad leaf scale mismatch")
		}
		need := scale * scale / 2
		if *off+1+need > len(data) {
			return fmt.Errorf("fscollide: flag quad leaf OOB")
		}
		p := *off + 1
		for a := 0; a < scale; a++ {
			for b := 0; b < scale; b++ {
				idx := a*scale + b
				bt := data[p+idx/2]
				if idx&1 == 0 {
					out[(baseA+a)*stride+baseB+b] = bt & 0xf
				} else {
					out[(baseA+a)*stride+baseB+b] = (bt >> 4) & 0xf
				}
			}
		}
		*off += 1 + need
	case 0:
		sizes, w, err := readChildSizes(data, *off)
		if err != nil {
			return err
		}
		childOff := *off + 1 + 4*w
		half := scale / 2
		if half == 0 {
			return fmt.Errorf("fscollide: flag quad scale 0")
		}
		quads := [4][2]int{{0, 0}, {0, half}, {half, 0}, {half, half}}
		for q := 0; q < 4; q++ {
			sub := childOff + quadChildBase(sizes, q)
			if err := qfNode(half, baseA+quads[q][0], baseB+quads[q][1], stride, data, &sub, out); err != nil {
				return err
			}
		}
		*off = childOff + sizes[0] + sizes[1] + sizes[2] + sizes[3]
	default:
		return fmt.Errorf("fscollide: flag quad bad tag %d", tag)
	}
	return nil
}

// QuadBitDecompress 复刻 CWalkBitCompress::Decompress：整树解码为 scale²
// 字节 0/1 阵（LoadArea 栅格化类型图用）。
func QuadBitDecompress(scale int, data []byte) ([]byte, error) {
	out := make([]byte, scale*scale)
	off := 0
	if err := bitQuadNode(scale, 0, 0, scale, data, &off, out); err != nil {
		return nil, err
	}
	return out, nil
}

func bitQuadNode(scale, baseA, baseB, stride int, data []byte, off *int, out []byte) error {
	if *off >= len(data) {
		return fmt.Errorf("fscollide: bit quad offset OOB")
	}
	tag := data[*off] & 0xf
	switch tag {
	case 2:
		v := byte(0)
		if data[*off]&0x10 != 0 {
			v = 1
		}
		for a := 0; a < scale; a++ {
			for b := 0; b < scale; b++ {
				out[(baseA+a)*stride+baseB+b] = v
			}
		}
		*off++
	case 1:
		if int(data[*off]>>4) != scale {
			return fmt.Errorf("fscollide: bit quad leaf scale mismatch")
		}
		if *off+1+(scale*scale+7)/8 > len(data) {
			return fmt.Errorf("fscollide: bit quad leaf OOB")
		}
		p := *off + 1
		for a := 0; a < scale; a++ {
			for b := 0; b < scale; b++ {
				idx := a*scale + b
				if data[p+idx/8]&(1<<(idx&7)) != 0 {
					out[(baseA+a)*stride+baseB+b] = 1
				}
			}
		}
		*off += 1 + (scale*scale+7)/8
	case 0:
		sizes, w, err := readChildSizes(data, *off)
		if err != nil {
			return err
		}
		childOff := *off + 1 + 4*w
		half := scale / 2
		if half == 0 {
			return fmt.Errorf("fscollide: bit quad scale 0")
		}
		quads := [4][2]int{{0, 0}, {0, half}, {half, 0}, {half, half}}
		for q := 0; q < 4; q++ {
			sub := childOff + quadChildBase(sizes, q)
			if err := bitQuadNode(half, baseA+quads[q][0], baseB+quads[q][1], stride, data, &sub, out); err != nil {
				return err
			}
		}
		*off = childOff + sizes[0] + sizes[1] + sizes[2] + sizes[3]
	default:
		return fmt.Errorf("fscollide: bit quad bad tag %d", tag)
	}
	return nil
}
