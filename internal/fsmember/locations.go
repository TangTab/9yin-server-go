package fsmember

import (
	"strconv"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// Location 全局落地点（location.ini [名] 节）：
// Location=x,z,y,orient 四 CSV + Scene=场景配置路径。
type Location struct {
	Name    string
	X, Z, Y float64
	Orient  float64
	Scene   string
}

// Locations 复刻 GameMember::LoadLocations（location.ini 全局点表）+
// SetLandPoint 的查找面（"(CGamePlayer::SetLandPoint)point not found"）。
type Locations struct {
	mu   sync.RWMutex
	byID map[string]Location
}

// NewLocations 建表。
func NewLocations() *Locations { return &Locations{byID: make(map[string]Location)} }

// Load 复刻 LoadLocations(path)：[Name] Location=x,z,y,orient / Scene=path
// （分词统一走 fskit.ParseItems＝官方 CIniFile Build 方言，按文件序成组）。
func (l *Locations) Load(data []byte) error {
	section := ""
	loc := Location{}
	flush := func() {
		if section != "" && loc.Scene != "" {
			l.byID[section] = loc
		}
	}
	err := fskit.ParseItems(data, func(sec, key, val string) error {
		if sec == "" { // 节前裸键不入表（同旧解析器口径）
			return nil
		}
		if sec != section {
			flush()
			section = sec
			loc = Location{Name: sec}
		}
		switch strings.ToLower(key) {
		case "location":
			parts := strings.Split(val, ",")
			nums := make([]float64, 4)
			for i := 0; i < 4 && i < len(parts); i++ {
				n, err := strconv.ParseFloat(strings.TrimSpace(parts[i]), 64)
				if err == nil {
					nums[i] = n
				}
			}
			loc.X, loc.Z, loc.Y, loc.Orient = nums[0], nums[1], nums[2], nums[3]
		case "scene":
			loc.Scene = val
		}
		return nil
	})
	flush()
	return err
}

// Find 按名查点。
func (l *Locations) Find(name string) (Location, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	loc, ok := l.byID[name]
	return loc, ok
}

// Count 点数。
func (l *Locations) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.byID)
}

// Names 点名清单（排序稳定）。
func (l *Locations) Names() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]string, 0, len(l.byID))
	for n := range l.byID {
		out = append(out, n)
	}
	// 排序
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
