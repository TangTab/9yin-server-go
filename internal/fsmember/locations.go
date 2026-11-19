package fsmember

import (
	"strconv"
	"strings"
	"sync"
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

// Load 复刻 LoadLocations(path)：[Name] Location=x,z,y,orient / Scene=path。
func (l *Locations) Load(data []byte) error {
	section := ""
	loc := Location{}
	flush := func() {
		if section != "" && loc.Scene != "" {
			l.byID[section] = loc
		}
	}
	for line := range eachLine(string(data)) {
		line = trimSpace(line)
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()
			section = strings.TrimSpace(line[1 : len(line)-1])
			loc = Location{Name: section}
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 || section == "" {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:eq]))
		val := strings.TrimSpace(line[eq+1:])
		switch key {
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
	}
	flush()
	return nil
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
