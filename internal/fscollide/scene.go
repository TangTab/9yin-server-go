// CSceneCollide3D（scenecollide3d.cpp）：zone 化 3D 场景碰撞。
//
// 坐标系（反汇编实证）：
//
//	mapLeft  = -(OriginCol+0.5)*zoneWorld   mapTop = -(OriginRow+0.5)*zoneWorld
//	zoneWorld = ZoneScale*UnitSize           cell   = UnitSize/CollidePerUnit
//	x → 列（col），z → 行（row）；zones[row*ZoneCols+col]；
//	zone 文件名 zone_<row>_<col>（LoadZone 第一 %d 对 ZoneRows 校验）。
//
// 派生缓存：invZoneWorld、invCell、mapSizeX=ZoneCols*zoneWorld、
// mapSizeZ=ZoneRows*zoneWorld；装载后 UpdateMaxArea 把边界收缩到实际
// 已加载 zone 的包络。
package fscollide

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// TerrainZone 复刻 CTerrainZone（holder）。
type TerrainZone struct {
	Index    int // row*ZoneCols+col
	Row, Col int
	Left     float32
	Top      float32
	Height   *TerrainHeight
	Walk     *TerrainWalkable
	Region   *TerrainRegion
}

// SceneCollide3D 复刻 CSceneCollide3D。
type SceneCollide3D struct {
	ZoneRows, ZoneCols   int
	OriginRow, OriginCol int
	ZoneScale            int
	CollidePerUnit       int
	InitHeight           float32
	UnitSize             float32
	InvUnit              float32
	ZoneWorld            float32
	InvZoneWorld         float32
	CellSize             float32
	InvCell              float32
	MapLeft, MapTop      float32
	MaxLeft, MaxTop      float32
	MapSizeX, MapSizeZ   float32
	walkPath             string
	Zones                []*TerrainZone // ZoneRows*ZoneCols
}

// LoadScene3D 复刻 CSceneCollide3D::Load：解析 terrain.ini 并装载全部 zone。
func LoadScene3D(mapDir string) (*SceneCollide3D, error) {
	iniPath := filepath.Join(mapDir, "terrain.ini")
	ini, err := loadIni(iniPath)
	if err != nil {
		return nil, fmt.Errorf("fscollide: load ini file failed: %w", err)
	}
	geti := func(k string, def int) int {
		if v, ok := ini["system"][k]; ok {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				return n
			}
		}
		return def
	}
	getf := func(k string, def float32) float32 {
		if v, ok := ini["system"][k]; ok {
			if f, err := strconv.ParseFloat(strings.TrimSpace(v), 32); err == nil {
				return float32(f)
			}
		}
		return def
	}
	s := &SceneCollide3D{
		ZoneRows:       geti("ZoneRows", 100),
		ZoneCols:       geti("ZoneCols", 100),
		OriginRow:      geti("OriginRow", 50),
		OriginCol:      geti("OriginCol", 50),
		ZoneScale:      geti("ZoneScale", 256),
		CollidePerUnit: geti("CollidePerUnit", 2),
		InitHeight:     getf("InitHeight", 0),
		UnitSize:       getf("UnitSize", 1),
	}
	s.walkPath = ini["system"]["WalkablePath"]
	if s.ZoneRows <= 0 || s.ZoneRows > 0x100 || s.ZoneCols <= 0 || s.ZoneCols > 0x100 {
		return nil, fmt.Errorf("fscollide: %s: zone number error", iniPath)
	}
	if s.OriginRow >= s.ZoneRows || s.OriginCol >= s.ZoneCols {
		return nil, fmt.Errorf("fscollide: %s: origin zone error", iniPath)
	}
	if s.ZoneScale <= 0 || s.ZoneScale > 0x400 {
		return nil, fmt.Errorf("fscollide: %s: zone scale error", iniPath)
	}
	switch s.CollidePerUnit {
	case 1, 2, 4, 8:
	default:
		return nil, fmt.Errorf("fscollide: %s: collide per unit error", iniPath)
	}
	switch int(s.UnitSize) {
	case 1, 2, 4, 8:
		s.UnitSize = float32(int(s.UnitSize))
	default:
		return nil, fmt.Errorf("fscollide: %s: unit size error %v", iniPath, s.UnitSize)
	}
	s.InvUnit = 1 / s.UnitSize
	s.ZoneWorld = float32(s.ZoneScale) * s.UnitSize
	s.InvZoneWorld = 1 / s.ZoneWorld
	s.CellSize = s.UnitSize / float32(s.CollidePerUnit)
	s.InvCell = 1 / s.CellSize
	s.MapLeft = -(float32(s.OriginCol) + 0.5) * s.ZoneWorld
	s.MapTop = -(float32(s.OriginRow) + 0.5) * s.ZoneWorld
	s.MaxLeft, s.MaxTop = s.MapLeft, s.MapTop
	s.MapSizeX = float32(s.ZoneCols) * s.ZoneWorld
	s.MapSizeZ = float32(s.ZoneRows) * s.ZoneWorld
	s.Zones = make([]*TerrainZone, s.ZoneRows*s.ZoneCols)

	for row := 0; row < s.ZoneRows; row++ {
		for col := 0; col < s.ZoneCols; col++ {
			_, _, _ = s.loadZone(mapDir, row, col) // 官方对 LoadZone 返回值不敏感
		}
	}
	s.updateMaxArea()
	// ConnectValues：把每 zone 西/北边缘值覆写进左/上邻居的东/南边缘。
	for _, z := range s.Zones {
		if z != nil && z.Height != nil {
			s.connectValues(z)
		}
	}
	return s, nil
}

// loadZone 复刻 LoadZone：文件不存在静默返回（地面+行走均缺才算失败）。
func (s *SceneCollide3D) loadZone(mapDir string, row, col int) (*TerrainZone, bool, error) {
	ground := filepath.Join(mapDir, fmt.Sprintf("zone_%d_%d.ground", row, col))
	var walk string
	if s.walkPath == "" {
		walk = filepath.Join(mapDir, fmt.Sprintf("zone_%d_%d.walk", row, col))
	} else {
		walk = filepath.Join(mapDir, s.walkPath, fmt.Sprintf("zone_%d_%d.walk", row, col))
	}
	region := filepath.Join(mapDir, fmt.Sprintf("zone_%d_%d.region", row, col))
	area := filepath.Join(mapDir, fmt.Sprintf("zone_%d_%d.area", row, col))
	groundOK := fileExists(ground)
	walkOK := fileExists(walk)
	if !groundOK && !walkOK {
		return nil, false, nil
	}
	left := float32(col)*s.ZoneWorld + s.MapLeft
	top := float32(row)*s.ZoneWorld + s.MapTop
	z := &TerrainZone{
		Index: row*s.ZoneCols + col,
		Row:   row, Col: col,
		Left: left, Top: top,
	}
	if groundOK {
		h, err := LoadTerrainHeight(ground, s.ZoneScale+1, left, top, s.UnitSize, s.InitHeight)
		if err != nil {
			return nil, false, fmt.Errorf("load height failed: %w", err)
		}
		z.Height = h
	}
	if walkOK {
		w, err := LoadTerrainWalkable(walk, s.ZoneScale*s.CollidePerUnit)
		if err != nil {
			return nil, false, fmt.Errorf("load walkable failed: %w", err)
		}
		z.Walk = w
	}
	if fileExists(region) {
		r, err := LoadTerrainRegion(region, s.ZoneScale, false)
		if err != nil {
			return nil, false, fmt.Errorf("load region failed: %w", err)
		}
		z.Region = r
	}
	if fileExists(area) {
		r, err := LoadTerrainRegion(area, s.ZoneScale, true)
		if err != nil {
			return nil, false, fmt.Errorf("load area failed: %w", err)
		}
		if z.Region == nil {
			z.Region = r
		} else {
			z.Region.Type = r.Type // .area 提供类型栅格
		}
	}
	s.Zones[z.Index] = z
	return z, true, nil
}

// updateMaxArea 复刻：以实际装载 zone 的包络收缩地图边界。
func (s *SceneCollide3D) updateMaxArea() {
	minL, minT := heightNull, heightNull
	maxR, maxB := -heightNull, -heightNull
	for _, z := range s.Zones {
		if z == nil {
			continue
		}
		if z.Left < minL {
			minL = z.Left
		}
		if z.Top < minT {
			minT = z.Top
		}
		if r := z.Left + s.ZoneWorld; r > maxR {
			maxR = r
		}
		if b := z.Top + s.ZoneWorld; b > maxB {
			maxB = b
		}
	}
	if minL < maxR && minT < maxB {
		s.MaxLeft, s.MaxTop = minL, minT
		s.MapSizeX, s.MapSizeZ = maxR-minL, maxB-minT
	}
}

// connectValues 复刻 CTerrainHeight::ConnectValues(zone)：
// 左邻居东缘列=本区西缘列；上邻居南缘行=本区北缘行；左上角点同步。
func (s *SceneCollide3D) connectValues(z *TerrainZone) {
	h := z.Height
	if z.Col > 0 { // 左邻居东缘 = 本区西缘
		if n := s.Zones[z.Row*s.ZoneCols+z.Col-1]; n != nil && n.Height != nil {
			for i := 0; i < h.Scale; i++ {
				n.Height.Value[(h.Scale-1)*h.Scale+i] = h.Value[i] // SetValue(col=scale-1,row=i) ← GetValue(col=0,row=i)
			}
		}
	}
	if z.Row > 0 { // 上邻居南缘 = 本区北缘
		if u := s.Zones[(z.Row-1)*s.ZoneCols+z.Col]; u != nil && u.Height != nil {
			for i := 0; i < h.Scale; i++ {
				u.Height.Value[i*h.Scale+h.Scale-1] = h.Value[i*h.Scale] // SetValue(col=i,row=scale-1) ← GetValue(col=i,row=0)
			}
		}
	}
	if z.Row > 0 && z.Col > 0 { // 左上角点
		if lu := s.Zones[(z.Row-1)*s.ZoneCols+z.Col-1]; lu != nil && lu.Height != nil {
			lu.Height.Value[(h.Scale-1)*h.Scale+h.Scale-1] = h.Value[0]
		}
	}
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// GetInZone 复刻：fx=(x-mapLeft)*invZoneWorld；fx<=0 或越界 → nil。
func (s *SceneCollide3D) GetInZone(x, z float32) *TerrainZone {
	fx := (x - s.MapLeft) * s.InvZoneWorld
	fz := (z - s.MapTop) * s.InvZoneWorld
	if fx <= 0 || fz <= 0 {
		return nil
	}
	col := int(fx) // 截断
	row := int(fz)
	if row >= s.ZoneRows || col >= s.ZoneCols {
		return nil
	}
	return s.Zones[row*s.ZoneCols+col]
}

// zoneCell 求世界坐标在 zone 内的碰撞格 (col,row)。
func (s *SceneCollide3D) zoneCell(z *TerrainZone, x, zz float32) (col, row int) {
	col = int((x - z.Left) * s.InvCell)
	row = int((zz - z.Top) * s.InvCell)
	return
}

// unitCell 求世界坐标在 zone 内的地形单元格（region 用）。
func (s *SceneCollide3D) unitCell(z *TerrainZone, x, zz float32) (col, row int) {
	col = int((x - z.Left) * s.InvUnit)
	row = int((zz - z.Top) * s.InvUnit)
	return
}

// ---- ISceneCollide 查询（全部按官方语义逐条复刻）----

func (s *SceneCollide3D) GetMultiLayer() bool { return true }

// GetMapLeft/Top/SizeX/SizeZ 返回 UpdateMaxArea 收缩后的实际包络
// （官方 +0x48/+0x4c/+0x50/+0x54 三元组；+0x40/+0x44 仅作网格原点）。
func (s *SceneCollide3D) GetMapLeft() float32  { return s.MaxLeft }
func (s *SceneCollide3D) GetMapTop() float32   { return s.MaxTop }
func (s *SceneCollide3D) GetMapSizeX() float32 { return s.MapSizeX }
func (s *SceneCollide3D) GetMapSizeZ() float32 { return s.MapSizeZ }

func (s *SceneCollide3D) GetMapType(x, z float32) int { return 0 }

// GetWalkType 复刻：GetWalkEnable→1/0。
func (s *SceneCollide3D) GetWalkType(x, z float32) int {
	if s.GetWalkEnable(x, z) {
		return 1
	}
	return 0
}

// GetCollideEnable 复刻：zone 存在且有地形高度。
func (s *SceneCollide3D) GetCollideEnable(x, z float32) bool {
	zone := s.GetInZone(x, z)
	return zone != nil && zone.Height != nil
}

// GetGroundHeight 复刻：无 zone→0；否则地形 GetY。
func (s *SceneCollide3D) GetGroundHeight(x, z float32) float32 {
	zone := s.GetInZone(x, z)
	if zone == nil || zone.Height == nil {
		return 0
	}
	return zone.Height.GetY(x, z)
}

// GetApexHeight 复刻：自顶层向下找首个 bit0 层；空高→地形。
func (s *SceneCollide3D) GetApexHeight(x, z float32) float32 {
	zone := s.GetInZone(x, z)
	if zone == nil {
		return 0
	}
	if zone.Walk == nil {
		return zone.Height.GetY(x, z)
	}
	col, row := s.zoneCell(zone, x, z)
	for f := zone.Walk.GetFloorCount() - 1; f >= 0; f-- {
		if zone.Walk.GetFloorMarker(row, col, f)&1 == 0 {
			continue
		}
		h := zone.Walk.GetFloorHeight(row, col, f)
		if heightIsNull(h) {
			return zone.Height.GetY(x, z)
		}
		return h
	}
	return zone.Height.GetY(x, z)
}

// GetApexFloor 复刻 GetApexFloor(x,z)：自顶层向下首个有效层的层号；无→-1。
func (s *SceneCollide3D) GetApexFloor(x, z float32) int {
	zone := s.GetInZone(x, z)
	if zone == nil || zone.Walk == nil {
		return -1
	}
	col, row := s.zoneCell(zone, x, z)
	for f := zone.Walk.GetFloorCount() - 1; f >= 0; f-- {
		if zone.Walk.GetFloorMarker(row, col, f)&1 != 0 {
			return f
		}
	}
	return -1
}

// GetWalkEnable 复刻：无可行走数据→true；walkMarker!=0。
func (s *SceneCollide3D) GetWalkEnable(x, z float32) bool {
	zone := s.GetInZone(x, z)
	if zone == nil {
		return false
	}
	if zone.Walk == nil {
		return true
	}
	col, row := s.zoneCell(zone, x, z)
	return zone.Walk.GetWalkMarker(row, col) != 0
}

// GetWalkHeight 复刻：walkMarker(=层号+1)>0 → 该层地板高；空高→地形。
func (s *SceneCollide3D) GetWalkHeight(x, z float32) float32 {
	zone := s.GetInZone(x, z)
	if zone == nil {
		return 0
	}
	if zone.Walk == nil {
		return zone.Height.GetY(x, z)
	}
	col, row := s.zoneCell(zone, x, z)
	h := heightNull
	if m := zone.Walk.GetWalkMarker(row, col); m > 0 {
		h = zone.Walk.GetFloorHeight(row, col, m-1)
	}
	if heightIsNull(h) {
		return zone.Height.GetY(x, z)
	}
	return h
}

// GetWalkWaterExists 复刻：水面高非空即有水。
func (s *SceneCollide3D) GetWalkWaterExists(x, z float32) bool {
	zone := s.GetInZone(x, z)
	if zone == nil || zone.Walk == nil {
		return false
	}
	col, row := s.zoneCell(zone, x, z)
	return !heightIsNull(zone.Walk.GetWaterHeight(row, col))
}

// GetWalkWaterHeight 复刻。
func (s *SceneCollide3D) GetWalkWaterHeight(x, z float32) float32 {
	zone := s.GetInZone(x, z)
	if zone == nil || zone.Walk == nil {
		return 0
	}
	col, row := s.zoneCell(zone, x, z)
	return zone.Walk.GetWaterHeight(row, col)
}

// GetFloorCount 复刻。
func (s *SceneCollide3D) GetFloorCount(x, z float32) int {
	zone := s.GetInZone(x, z)
	if zone == nil || zone.Walk == nil {
		return 0
	}
	return zone.Walk.GetFloorCount()
}

func (s *SceneCollide3D) validFloor(x, z float32, floor int) (*TerrainZone, int, int, bool) {
	zone := s.GetInZone(x, z)
	if zone == nil || zone.Walk == nil {
		return nil, 0, 0, false
	}
	if floor < 0 || floor >= zone.Walk.GetFloorCount() {
		return nil, 0, 0, false
	}
	col, row := s.zoneCell(zone, x, z)
	return zone, col, row, true
}

// GetFloorExists 复刻：marker bit0。
func (s *SceneCollide3D) GetFloorExists(x, z float32, floor int) bool {
	zone, col, row, ok := s.validFloor(x, z, floor)
	if !ok {
		return false
	}
	return zone.Walk.GetFloorMarker(row, col, floor)&1 != 0
}

// GetFloorCanMove 复刻：marker bit1。
func (s *SceneCollide3D) GetFloorCanMove(x, z float32, floor int) bool {
	zone, col, row, ok := s.validFloor(x, z, floor)
	if !ok {
		return false
	}
	return zone.Walk.GetFloorMarker(row, col, floor)&2 != 0
}

// GetFloorCanStand 复刻：marker bit2。
func (s *SceneCollide3D) GetFloorCanStand(x, z float32, floor int) bool {
	zone, col, row, ok := s.validFloor(x, z, floor)
	if !ok {
		return false
	}
	return zone.Walk.GetFloorMarker(row, col, floor)&4 != 0
}

// GetFloorHeight 复刻：空高→地形 GetY。
func (s *SceneCollide3D) GetFloorHeight(x, z float32, floor int) float32 {
	zone, col, row, ok := s.validFloor(x, z, floor)
	if !ok {
		return 0
	}
	h := zone.Walk.GetFloorHeight(row, col, floor)
	if heightIsNull(h) {
		return zone.Height.GetY(x, z)
	}
	return h
}

// GetFloorSpace 复刻。
func (s *SceneCollide3D) GetFloorSpace(x, z float32, floor int) float32 {
	zone, col, row, ok := s.validFloor(x, z, floor)
	if !ok {
		return 0
	}
	return zone.Walk.GetFloorSpace(row, col, floor)
}

// GetFloorHasWall 复刻。
func (s *SceneCollide3D) GetFloorHasWall(x, z float32, floor int) bool {
	zone, col, row, ok := s.validFloor(x, z, floor)
	if !ok {
		return false
	}
	return zone.Walk.GetWallExists(row, col, zone.Walk.GetFloorHeight(row, col, floor))
}

// GetWallExists 复刻：GetWallExists(row,col,y)。
func (s *SceneCollide3D) GetWallExists(x, z, y float32) bool {
	zone := s.GetInZone(x, z)
	if zone == nil || zone.Walk == nil {
		return false
	}
	col, row := s.zoneCell(zone, x, z)
	return zone.Walk.GetWallExists(row, col, y)
}

// GetMapRegion 复刻：命名区域位查询。
func (s *SceneCollide3D) GetMapRegion(name string, x, z float32) bool {
	zone := s.GetInZone(x, z)
	if zone == nil || zone.Region == nil {
		return false
	}
	col, row := s.unitCell(zone, x, z)
	return zone.Region.GetRegionEnable(name, row, col)
}

// GetMapArea 复刻：区域名查询。
func (s *SceneCollide3D) GetMapArea(x, z float32) string {
	zone := s.GetInZone(x, z)
	if zone == nil || zone.Region == nil {
		return ""
	}
	col, row := s.unitCell(zone, x, z)
	return zone.Region.GetAreaName(zone.Region.GetAreaType(row, col))
}

// GetMapHeight 复刻（虚槽 0x4c）＝ GetWalkHeight。
func (s *SceneCollide3D) GetMapHeight(x, z float32) float32 { return s.GetWalkHeight(x, z) }

// LineCanWalk 复刻：沿线按 step 采样，逐点行走位 & mask。
func (s *SceneCollide3D) LineCanWalk(mask int, step, x1, z1, x2, z2 float32) bool {
	dx := x2 - x1
	dz := z2 - z1
	dist := float32(math.Sqrt(float64(dx*dx + dz*dz)))
	if step >= dist {
		return s.GetWalkType(x2, z2)&mask != 0
	}
	stepX := dx / dist * step
	stepZ := dz / dist * step
	x, z := x1, z1
	n := int(math.Ceil(float64(dist / step))) // fistp round-up 控制字
	for i := 0; i < n; i++ {
		x += stepX
		z += stepZ
		if s.GetWalkType(x, z)&mask == 0 {
			return false
		}
	}
	return s.GetWalkType(x2, z2)&mask != 0
}
