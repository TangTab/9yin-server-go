package taskdata

import (
	"encoding/json"
	"fmt"
	"time"
)

// SnapshotVersion 每当派生逻辑或字段含义变化时递增；版本不匹配的
// 快照视为过期，调用方应重新派生。
const SnapshotVersion = 1

// Snapshot 是任务目录的版本化持久化形态（data/tasks.json）。
// 任务与剧情链的键在 JSON 中序列化为十进制字符串。
type Snapshot struct {
	Version     int               `json:"version"`
	GeneratedAt time.Time         `json:"generated_at"`
	Tasks       map[int]Task      `json:"tasks"`
	Bindings    []Binding         `json:"bindings"`
	DramaCites  map[int]DramaCite `json:"drama_cites,omitempty"`
}

// SnapshotAt 导出目录的快照；generatedAt 记录派生时刻。
func (c *Catalog) SnapshotAt(generatedAt time.Time) Snapshot {
	snapshot := Snapshot{Version: SnapshotVersion, GeneratedAt: generatedAt}
	if c == nil {
		return snapshot
	}
	snapshot.Tasks = make(map[int]Task, len(c.tasks))
	for id, task := range c.tasks {
		snapshot.Tasks[id] = task
	}
	snapshot.Bindings = append([]Binding(nil), c.bindings...)
	if len(c.dramaCites) > 0 {
		snapshot.DramaCites = make(map[int]DramaCite, len(c.dramaCites))
		for id, cite := range c.dramaCites {
			snapshot.DramaCites[id] = cite
		}
	}
	return snapshot
}

// MarshalSnapshot 编码快照为缩进 JSON（带结尾换行），强制写入当前版本。
func MarshalSnapshot(snapshot Snapshot) ([]byte, error) {
	snapshot.Version = SnapshotVersion
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("taskdata: encode snapshot: %w", err)
	}
	return append(data, '\n'), nil
}

// ParseSnapshot 解码快照并重建目录。版本不匹配或结构非法时报错，
// 调用方应回退到重新派生。
func ParseSnapshot(data []byte) (*Catalog, error) {
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("taskdata: decode snapshot: %w", err)
	}
	if snapshot.Version != SnapshotVersion {
		return nil, fmt.Errorf("taskdata: snapshot version %d does not match %d", snapshot.Version, SnapshotVersion)
	}
	return NewCatalog(snapshot.Tasks, snapshot.Bindings, snapshot.DramaCites), nil
}
