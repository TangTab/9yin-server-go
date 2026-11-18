// 地形三件套：CTerrainHeight（terrainheight.cpp）、CTerrainWalkable
// （terrainwalkable.cpp）、CTerrainRegion（terrainregion.cpp）。
//
// CTerrainHeight：zone 高度网格 scale=ZoneScale+1（含右/下边缘采样），
// GetValue(col,row)=values[col*scale+row]（注意转置：第一下标是 x 列），
// GetY 为对角翻转双线性插值。
//
// CTerrainWalkable：WLK1 块。对象模型（偏移由反汇编实证）：
//
//	+0x180 floorCount      +0x184 water(CWalkHeightMap,单值)
//	+0x188 floorHeight[15] +0x1c4 floorSpace[15]
//	+0x200 walkMarker(单值) +0x204 floorMarker[15]
//
// floor marker 位定义：bit0=存在 bit1=可行走 bit2=可站立 bit3=有墙。
// walk marker 值=可行走地板号+1（0=不可走）。水面高/行走标记为全图单
// 数组，地板高/空间高/地板标记按层各一组（多层实证 1..15 层均存在）。
package fscollide

import (
	"encoding/binary"
	"fmt"
)

const maxFloorNum = 15 // 官方 assert floor < 15；floor amount ≤ 15

// TerrainHeight 复刻 CTerrainHeight。
type TerrainHeight struct {
	Scale int // ZoneScale+1
	Left  float32
	Top   float32
	Unit  float32
	Inv   float32 // 1/Unit
	Value []float32
}

// LoadTerrainHeight 解析 .ground 文件（HGT2 高度四叉树 + CHK0 跳过块）。
func LoadTerrainHeight(path string, scale int, left, top, unit, initHeight float32) (*TerrainHeight, error) {
	th := &TerrainHeight{
		Scale: scale,
		Left:  left,
		Top:   top,
		Unit:  unit,
		Inv:   1 / unit,
		Value: make([]float32, scale*scale),
	}
	for i := range th.Value {
		th.Value[i] = initHeight
	}
	blocks, err := LoadBlockFile(path)
	if err != nil {
		return nil, err
	}
	for _, b := range blocks {
		switch b.ID {
		case blockIDHeight2:
			if len(b.Data) < 12 {
				return nil, fmt.Errorf("fscollide: %s: HGT2 header", path)
			}
			s0 := binary.LittleEndian.Uint32(b.Data)
			s1 := binary.LittleEndian.Uint32(b.Data[4:])
			if int(s0) != scale || int(s1) != scale {
				return nil, fmt.Errorf("fscollide: %s: HGT2 scale %d/%d != %d", path, s0, s1, scale)
			}
			comp := binary.LittleEndian.Uint32(b.Data[8:])
			if uint64(comp) > uint64(len(b.Data)-12) {
				return nil, fmt.Errorf("fscollide: %s: HGT2 compress size error", path)
			}
			vals, err := TerrainHeightQuad(scale, b.Data[12:12+comp])
			if err != nil {
				return nil, fmt.Errorf("fscollide: %s: %w", path, err)
			}
			copy(th.Value, vals)
		case blockIDChunk:
			// CHK0：编辑器 chunk 记录，服务端只读行/列数后整块跳过。
			if len(b.Data) < 8 {
				return nil, fmt.Errorf("fscollide: %s: CHK0 header", path)
			}
			rows := binary.LittleEndian.Uint32(b.Data)
			cols := binary.LittleEndian.Uint32(b.Data[4:])
			need := 8 + uint64(rows)*uint64(cols)*56
			if uint64(len(b.Data)) < need {
				return nil, fmt.Errorf("fscollide: %s: CHK0 unit size error", path)
			}
		default:
			// 未知块：官方打印 block unknown 并跳过。
		}
	}
	return th, nil
}

// GetValue 复刻 CTerrainHeight::GetValue(col,row)=values[col*scale+row]。
func (t *TerrainHeight) GetValue(col, row int) float32 {
	if col < 0 || col >= t.Scale || row < 0 || row >= t.Scale {
		return 0
	}
	return t.Value[col*t.Scale+row]
}

// GetY 复刻 CTerrainHeight::GetY(x,z)：对角翻转双线性插值；越界返回 0。
func (t *TerrainHeight) GetY(x, z float32) float32 {
	fx := x - t.Left
	fz := z - t.Top
	col := int(fx * t.Inv) // 官方 FloatToInt=cvttss2si 截断；负值也截断
	row := int(fz * t.Inv)
	if col >= t.Scale-1 || row >= t.Scale-1 {
		return 0
	}
	if col < 0 || row < 0 {
		return 0
	}
	v1 := t.GetValue(col+1, row) // 右
	v2 := t.GetValue(col, row+1) // 下
	tx := (fx - float32(col)*t.Unit) * t.Inv
	tz := (fz - float32(row)*t.Unit) * t.Inv
	if tx+tz >= 1 { // 上三角：以 (col,row) 为基
		v0 := t.GetValue(col, row)
		return v0 + (v1-v0)*tx + (v2-v0)*tz
	} // 下三角：以 (col+1,row+1) 为基
	v3 := t.GetValue(col+1, row+1)
	tx = 1 - tx
	tz = 1 - tz
	return v3 + (v2-v3)*tx + (v1-v3)*tz
}

// TerrainWalkable 复刻 CTerrainWalkable（解压后的扁平阵等价表示）。
type TerrainWalkable struct {
	Scale int // ZoneScale*CollidePerUnit
	// 每层数组；nil = 该层无此数据
	FloorMarker [][]byte    // [floor] 值：bit0 存在 bit1 可走 bit2 可站 bit3 墙
	FloorHeight [][]float32 // [floor]
	FloorSpace  [][]float32 // [floor]
	WalkMarker  []byte      // 单值（最后一层的数据）
	Water       []float32   // 单值（最后一层的数据）
}

// LoadTerrainWalkable 解析 .walk 文件（WLK1）。
// 块内布局（字符串地址+数组计数 2+3×floors 双重实证）：
//
//	u32 scale, u32 scale, u32 floorCount(≤15)
//	单数组：水面高（高度四叉树）
//	单数组：行走标记（标记四叉树，值=层号+1）
//	每层：地板高（高度四叉树）、空间高（高度四叉树）、地板标记（标记四叉树）
//
// 全部为四叉树直存；官方 Debug 版另做 RLE 回压校验（memcmp），此处从略。
func LoadTerrainWalkable(path string, scale int) (*TerrainWalkable, error) {
	blocks, err := LoadBlockFile(path)
	if err != nil {
		return nil, err
	}
	tw := &TerrainWalkable{Scale: scale}
	for _, b := range blocks {
		if b.ID != blockIDWalk {
			continue
		}
		if len(b.Data) < 12 {
			return nil, fmt.Errorf("fscollide: %s: WLK1 header", path)
		}
		s0 := binary.LittleEndian.Uint32(b.Data)
		s1 := binary.LittleEndian.Uint32(b.Data[4:])
		if int(s0) != scale || int(s1) != scale {
			return nil, fmt.Errorf("fscollide: %s: WLK1 scale %d/%d != %d", path, s0, s1, scale)
		}
		floorCount := int(binary.LittleEndian.Uint32(b.Data[8:]))
		if floorCount > maxFloorNum {
			return nil, fmt.Errorf("fscollide: %s: floor amount error %d", path, floorCount)
		}
		tw.FloorMarker = make([][]byte, floorCount)
		tw.FloorHeight = make([][]float32, floorCount)
		tw.FloorSpace = make([][]float32, floorCount)
		p := 12
		water, np, err := readHeightQuad(b.Data, p, path, scale)
		if err != nil {
			return nil, err
		}
		tw.Water = water
		walk, np2, err := readFlagQuad(b.Data, np, path, scale)
		if err != nil {
			return nil, err
		}
		tw.WalkMarker = walk
		p = np2
		// 分组布局（类型探测实证）：地板高×n、空间高×n、地板标记×n。
		for f := 0; f < floorCount; f++ {
			fh, np3, err := readHeightQuad(b.Data, p, path, scale)
			if err != nil {
				return nil, err
			}
			tw.FloorHeight[f] = fh
			p = np3
		}
		for f := 0; f < floorCount; f++ {
			sp, np4, err := readHeightQuad(b.Data, p, path, scale)
			if err != nil {
				return nil, err
			}
			tw.FloorSpace[f] = sp
			p = np4
		}
		for f := 0; f < floorCount; f++ {
			fm, np5, err := readFlagQuad(b.Data, p, path, scale)
			if err != nil {
				return nil, err
			}
			tw.FloorMarker[f] = fm
			p = np5
		}
		if p != len(b.Data) {
			return nil, fmt.Errorf("fscollide: %s: WLK1 未消费完 %d/%d", path, p, len(b.Data))
		}
	}
	return tw, nil
}

// readRawArray 读一个 {u32 size, data[size]} 头；size=0 → nil 段。
func readRawArray(data []byte, p int, path string) ([]byte, int, error) {
	if p+4 > len(data) {
		return nil, 0, fmt.Errorf("fscollide: %s: walk array size truncated", path)
	}
	size := int(binary.LittleEndian.Uint32(data[p:]))
	p += 4
	if size < 0 || size > 0x100000 {
		return nil, 0, fmt.Errorf("fscollide: %s: walk array size error %d", path, size)
	}
	if size == 0 {
		return nil, p, nil
	}
	if p+size > len(data) {
		return nil, 0, fmt.Errorf("fscollide: %s: walk array data truncated", path)
	}
	return data[p : p+size], p + size, nil
}

func readFlagQuad(data []byte, p int, path string, scale int) ([]byte, int, error) {
	seg, np, err := readRawArray(data, p, path)
	if err != nil {
		return nil, 0, err
	}
	if seg == nil {
		return nil, np, nil
	}
	out, err := QuadFlagDecompress(scale, seg)
	if err != nil {
		return nil, 0, fmt.Errorf("fscollide: %s: %w", path, err)
	}
	return out, np, nil
}

func readHeightQuad(data []byte, p int, path string, scale int) ([]float32, int, error) {
	seg, np, err := readRawArray(data, p, path)
	if err != nil {
		return nil, 0, err
	}
	if seg == nil {
		return nil, np, nil
	}
	out, err := QuadHeightDecompress(scale, seg)
	if err != nil {
		return nil, 0, fmt.Errorf("fscollide: %s: %w", path, err)
	}
	return out, np, nil
}

// GetFloorCount 返回层数（按 marker/height/space 中最大非空层数）。
func (t *TerrainWalkable) GetFloorCount() int {
	n := 0
	for i := 0; i < maxFloorNum; i++ {
		if i < len(t.FloorMarker) && (t.FloorMarker[i] != nil || t.FloorHeightAt(i) != nil) {
			n = i + 1
		}
	}
	return n
}

func (t *TerrainWalkable) FloorHeightAt(f int) []float32 {
	if f < 0 || f >= len(t.FloorHeight) {
		return nil
	}
	return t.FloorHeight[f]
}

// GetWalkMarker 复刻：无数据 → 0；否则 values[row*scale+col]。
func (t *TerrainWalkable) GetWalkMarker(row, col int) int {
	if t.WalkMarker == nil || row < 0 || row >= t.Scale || col < 0 || col >= t.Scale {
		return 0
	}
	return int(t.WalkMarker[row*t.Scale+col])
}

// GetFloorMarker 复刻：越层或无数据 → 0。
func (t *TerrainWalkable) GetFloorMarker(row, col, floor int) int {
	if floor < 0 || floor >= len(t.FloorMarker) || t.FloorMarker[floor] == nil {
		return 0
	}
	if row < 0 || row >= t.Scale || col < 0 || col >= t.Scale {
		return 0
	}
	return int(t.FloorMarker[floor][row*t.Scale+col])
}

// GetFloorHeight 复刻：无数据 → 1e8（heightNull）。
func (t *TerrainWalkable) GetFloorHeight(row, col, floor int) float32 {
	h := t.FloorHeightAt(floor)
	if h == nil || row < 0 || row >= t.Scale || col < 0 || col >= t.Scale {
		return heightNull
	}
	return h[row*t.Scale+col]
}

// GetFloorSpace 复刻：无数据 → 1e8。
func (t *TerrainWalkable) GetFloorSpace(row, col, floor int) float32 {
	if floor < 0 || floor >= len(t.FloorSpace) || t.FloorSpace[floor] == nil {
		return heightNull
	}
	if row < 0 || row >= t.Scale || col < 0 || col >= t.Scale {
		return heightNull
	}
	return t.FloorSpace[floor][row*t.Scale+col]
}

// GetWaterHeight 复刻：无数据 → 1e8。
func (t *TerrainWalkable) GetWaterHeight(row, col int) float32 {
	if t.Water == nil || row < 0 || row >= t.Scale || col < 0 || col >= t.Scale {
		return heightNull
	}
	return t.Water[row*t.Scale+col]
}

// GetWallExists 复刻：任一层 marker bit3 且该层地板高 < y → 有墙。
func (t *TerrainWalkable) GetWallExists(row, col int, y float32) bool {
	for f := 0; f < t.GetFloorCount(); f++ {
		if t.GetFloorMarker(row, col, f)&8 == 0 {
			continue
		}
		if t.GetFloorHeight(row, col, f) < y {
			return true
		}
	}
	return false
}

// TerrainRegion 复刻 CTerrainRegion：命名区域位图 + 区域类型栅格。
type TerrainRegion struct {
	Scale int
	Area  []AreaData
	Type  []byte // scale²：区域号+1；0=无
}

// AreaData 对应 area_data_t。
type AreaData struct {
	Name string
	Bits []byte // 压缩位四叉树原始数据（按需查询）
}

// LoadTerrainRegion 解析 .region/.area 文件（两者同为 ARE0 块）。
// .area 额外栅格化 Type；.region 只保留命名位图。区域内数据可多次累加。
func LoadTerrainRegion(path string, scale int, rasterize bool) (*TerrainRegion, error) {
	blocks, err := LoadBlockFile(path)
	if err != nil {
		return nil, err
	}
	tr := &TerrainRegion{Scale: scale}
	for _, b := range blocks {
		if b.ID != blockIDArea {
			continue
		}
		d := b.Data
		if len(d) < 12 {
			return nil, fmt.Errorf("fscollide: %s: ARE0 header", path)
		}
		s0 := binary.LittleEndian.Uint32(d)
		s1 := binary.LittleEndian.Uint32(d[4:])
		if int(s0) != scale || int(s1) != scale {
			return nil, fmt.Errorf("fscollide: %s: ARE0 scale %d/%d != %d", path, s0, s1, scale)
		}
		count := int(binary.LittleEndian.Uint32(d[8:]))
		if count > 0x100 {
			return nil, fmt.Errorf("fscollide: %s: area amount error %d", path, count)
		}
		p := 12
		for i := 0; i < count; i++ {
			if p+4 > len(d) {
				return nil, fmt.Errorf("fscollide: %s: area name len truncated", path)
			}
			nameLen := int(binary.LittleEndian.Uint32(d[p:]))
			p += 4
			if nameLen > 0x100 {
				return nil, fmt.Errorf("fscollide: %s: area name len error", path)
			}
			if p+nameLen > len(d) {
				return nil, fmt.Errorf("fscollide: %s: area name truncated", path)
			}
			name := string(d[p : p+nameLen])
			p += nameLen
			if p+4 > len(d) {
				return nil, fmt.Errorf("fscollide: %s: region size truncated", path)
			}
			dataLen := int(binary.LittleEndian.Uint32(d[p:]))
			p += 4
			if dataLen > 0x100000 {
				return nil, fmt.Errorf("fscollide: %s: region size error", path)
			}
			if p+dataLen > len(d) {
				return nil, fmt.Errorf("fscollide: %s: area data truncated", path)
			}
			data := d[p : p+dataLen]
			p += dataLen
			idx := len(tr.Area)
			tr.Area = append(tr.Area, AreaData{Name: name, Bits: append([]byte(nil), data...)})
			if rasterize && dataLen > 0 {
				if tr.Type == nil {
					tr.Type = make([]byte, scale*scale)
				}
				bits, err := QuadBitDecompress(scale, data)
				if err != nil {
					return nil, fmt.Errorf("fscollide: %s: %w", path, err)
				}
				for c := 0; c < scale*scale; c++ {
					if bits[c] != 0 {
						tr.Type[c] = byte(idx + 1)
					}
				}
			}
		}
		if len(tr.Area) > 0xfe && rasterize {
			return nil, fmt.Errorf("fscollide: %s: too much area", path)
		}
	}
	return tr, nil
}

// GetAreaType 复刻：type==nil → -1；值 0 → -1；否则 值-1。
func (t *TerrainRegion) GetAreaType(row, col int) int {
	if t.Type == nil {
		return -1
	}
	if row < 0 || row >= t.Scale || col < 0 || col >= t.Scale {
		return -1
	}
	v := int(t.Type[row*t.Scale+col])
	if v == 0 {
		return -1
	}
	return v - 1
}

// GetAreaName 复刻：越界 → ""。
func (t *TerrainRegion) GetAreaName(index int) string {
	if index < 0 || index >= len(t.Area) {
		return ""
	}
	return t.Area[index].Name
}

// GetRegionEnable 复刻：按名找区域（哈希+strcmp），位四叉树查询。
func (t *TerrainRegion) GetRegionEnable(name string, row, col int) bool {
	if name == "" {
		return false
	}
	for i := range t.Area {
		if t.Area[i].Name != name || len(t.Area[i].Bits) == 0 {
			continue
		}
		ok, err := QuadGetBit(t.Scale, row, col, t.Area[i].Bits)
		if err == nil && ok {
			return true
		}
	}
	return false
}
