// CollideManager（collidemanager.cpp）与 CSceneCollideFlat（scenecollideflat.cpp）。
//
// CollideManager 是 Fs 框架实体（IEntity，名 "CollideManager"，space "system"）：
//	Init(varlist) 官方为空实现（恒 true）
//	CreateSceneCollide(mapDir)：mapDir\walkmap.ini 存在 → Flat；否则
//	mapDir\terrain.ini 存在 → 3D；加载后经 ISceneCollide 虚接口供游戏逻辑查询。
//
// Flat（walkmap.ini + MarkerMap/HeightMap 两文件）：BGserverCJ 部署内无任何
// walkmap.ini 样本（全量扫描仅 terrain.ini），Flat 的块内布局无数据可验证，
// 此处只落 ini 解析与文件路由，加载器返回显式错误（不臆造格式）。
package fscollide

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SceneCollide 是两类场景碰撞的统一门面（官方为 ISceneCollide 虚接口，
// 此处以接口复刻）。
type SceneCollide interface {
	GetMultiLayer() bool
	GetMapLeft() float32
	GetMapTop() float32
	GetMapSizeX() float32
	GetMapSizeZ() float32
	GetMapHeight(x, z float32) float32
	GetMapRegion(name string, x, z float32) bool
	GetMapArea(x, z float32) string
	GetMapType(x, z float32) int
	GetWalkType(x, z float32) int
	LineCanWalk(mask int, step, x1, z1, x2, z2 float32) bool
	GetCollideEnable(x, z float32) bool
	GetGroundHeight(x, z float32) float32
	GetApexHeight(x, z float32) float32
	GetApexFloor(x, z float32) int
	GetWalkEnable(x, z float32) bool
	GetWalkHeight(x, z float32) float32
	GetWalkWaterExists(x, z float32) bool
	GetWalkWaterHeight(x, z float32) float32
	GetFloorCount(x, z float32) int
	GetFloorExists(x, z float32, floor int) bool
	GetFloorCanMove(x, z float32, floor int) bool
	GetFloorCanStand(x, z float32, floor int) bool
	GetFloorHeight(x, z float32, floor int) float32
	GetFloorSpace(x, z float32, floor int) float32
	GetFloorHasWall(x, z float32, floor int) bool
	GetWallExists(x, z, y float32) bool
}

// Manager 复刻 CollideManager。
type Manager struct {
	scenes map[string]SceneCollide
}

func NewManager() *Manager { return &Manager{scenes: map[string]SceneCollide{}} }

// Init 复刻：恒成功。
func (m *Manager) Init() bool { return true }

// CreateSceneCollide 复刻路由：walkmap.ini → Flat；terrain.ini → 3D；缓存复用。
func (m *Manager) CreateSceneCollide(mapDir string) (SceneCollide, error) {
	if c, ok := m.scenes[mapDir]; ok {
		return c, nil
	}
	if fileExists(filepath.Join(mapDir, "walkmap.ini")) {
		return nil, fmt.Errorf("fscollide: SceneCollideFlat（walkmap.ini）无官方样本数据，暂不支持：%s", mapDir)
	}
	if fileExists(filepath.Join(mapDir, "terrain.ini")) {
		s, err := LoadScene3D(mapDir)
		if err != nil {
			return nil, err
		}
		m.scenes[mapDir] = s
		return s, nil
	}
	return nil, fmt.Errorf("fscollide: %s: 地图目录缺少 terrain.ini/walkmap.ini", mapDir)
}

// loadIni 极简 INI：[节] 与 key=value（保留原字节；官方 readini 同样按行）。
func loadIni(path string) (map[string]map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]map[string]string{}
	sec := ""
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimRight(line, "\r")
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, ";") {
			continue
		}
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			sec = t[1 : len(t)-1]
			if out[sec] == nil {
				out[sec] = map[string]string{}
			}
			continue
		}
		if i := strings.IndexByte(t, '='); i > 0 {
			k := strings.TrimSpace(t[:i])
			v := strings.TrimSpace(t[i+1:])
			if out[sec] == nil {
				out[sec] = map[string]string{}
			}
			out[sec][k] = v
		}
	}
	return out, nil
}
