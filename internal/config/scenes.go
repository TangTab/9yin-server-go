package config

import "fmt"

// SceneDef 对应 scenes.xml 的一行场景定义。
// Script 字段引用 logic_class.xml 的 class id（scene/CloneScene/...）。
// ID 保持字符串（原版存在 "202_1" 这类变体后缀写法）。
type SceneDef struct {
	ID          string
	Name        string
	Script      string
	Config      string // 场景 ini 配置根（如 ini\scene\westzq）
	Resource    string
	CollidePath string
	Born        string
	Relive      []string
	Clonable    bool // 副本场景：每队/每人独立实例
	Grouping    bool
	IsNight     bool
	Member      int // 所属进程（Member=0 为公场景）
}

// SceneTable 场景注册表。
type SceneTable struct {
	list []*SceneDef
	byID map[string]*SceneDef
}

// LoadScenes 解析 scenes.xml 并建表。
func LoadScenes(path string) (*SceneTable, error) {
	doc, err := LoadObjectFile(path)
	if err != nil {
		return nil, err
	}
	t := &SceneTable{byID: make(map[string]*SceneDef)}
	for i, row := range doc.Rows {
		def := &SceneDef{
			ID:          row.Str("ID", ""),
			Name:        row.Str("Name", ""),
			Script:      row.Str("Script", "scene"),
			Config:      row.Str("Config", ""),
			Resource:    row.Str("Resource", ""),
			CollidePath: row.Str("CollidePath", ""),
			Born:        row.Str("Born", ""),
			Relive:      row.CSV("ReliveLocation"),
			Clonable:    row.Bool("Clonable", false),
			Grouping:    row.Bool("Grouping", false),
			IsNight:     row.Bool("IsNight", false),
			Member:      row.Int("Member", 0),
		}
		if def.ID == "" {
			return nil, fmt.Errorf("%s: 第 %d 行缺 ID", path, i+1)
		}
		if _, dup := t.byID[def.ID]; dup {
			return nil, fmt.Errorf("%s: 场景 ID %s 重复", path, def.ID)
		}
		t.byID[def.ID] = def
		t.list = append(t.list, def)
	}
	return t, nil
}

// Get 按 ID 取场景；不存在返回 nil。
func (t *SceneTable) Get(id string) *SceneDef { return t.byID[id] }

// Len 场景总数。
func (t *SceneTable) Len() int { return len(t.list) }

// All 全部场景（注册顺序）。
func (t *SceneTable) All() []*SceneDef { return t.list }
