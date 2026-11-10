package clientdata

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

//go:embed official_property_table.txt
var officialPropertyTableText string

//go:embed official_property_table_2433.txt
var officialPropertyTable2433Text string

// SCENE_EXTENSION_FLOOR 是商店/运动扩展块在官方表中的起始位置:
// 扩展块名字按「首现且 index >= 99」解析(镜像 Java PropertyTable 语义)。
const SCENE_EXTENSION_FLOOR = 99

// PropertyEntry 是官方 0x09 属性表的一行。TypeCode 保留文件里的原始线型码
// (0/6 不可编码),Type 是映射后的 WireType(不可编码行为 0)。
type PropertyEntry struct {
	Index    uint16
	Name     string
	TypeCode int
	Type     WireType
}

// OfficialWireTypeOf 把官方表线型码映射为 WireType;未知码(0/6)返回 0。
func OfficialWireTypeOf(code int) WireType {
	switch code {
	case 1:
		return WireByte
	case 2:
		return WireWord
	case 3:
		return WireInt32
	case 4:
		return WireInt64
	case 5:
		return WireFloat32
	case 7:
		return WireString
	case 8:
		return WireWideString
	case 9:
		return WireObject
	default:
		return 0
	}
}

type propertyTable struct {
	entries     []PropertyEntry
	scene       map[string]PropertyEntry
	player      map[string]PropertyEntry
	sceneFields []PropertyEntry
}

func (t propertyTable) byNameScene(name string, minIndex int) (PropertyEntry, bool) {
	for _, entry := range t.entries {
		if int(entry.Index) >= minIndex && entry.Name == name && entry.Type != 0 {
			return entry, true
		}
	}
	return PropertyEntry{}, false
}

var (
	officialTable          propertyTable
	activePropertyTableMde string
)

func init() {
	text, mode := activePropertyTableText()
	officialTable = parsePropertyTable(text, mode)
	activePropertyTableMde = mode
}

// 属性表方言档位（NINEYIN_PROPERTY_TABLE，默认 fork）：
//
//	fork         官方 2370 项快照（场景域头块 0..173 + 共享域），与全部
//	             硬编码 prop* 常量配对自洽，现网方言——换表工程
//	             （分支 official-2433-table，docs/官服登录帧对照与缺口
//	             清单.md §14.3）完成前的默认。
//	official2433 官服 7074 线 2433 项真表（头块换血为家园/帮会/VIP 域，
//	             64 名缺席、共享域 +2/+7 插入漂移）——启用后 SceneProperty/
//	             PlayerProperty/0x09 下发帧全部按官表解析；场景域硬编码
//	             与 BufferListStr/NpcNickName 通道在 P3/P4 完成前会断，
//	             仅限换表分支联调。
func activePropertyTableText() (string, string) {
	switch strings.TrimSpace(os.Getenv("NINEYIN_PROPERTY_TABLE")) {
	case "official2433":
		return officialPropertyTable2433Text, "official2433"
	default:
		return officialPropertyTableText, "fork"
	}
}

// ActivePropertyTableMode 返回当前生效的属性表方言（日志/诊断用）。
func ActivePropertyTableMode() string { return activePropertyTableMde }

// SwapPropertyTableForTest 在测试内切换属性表方言并在测试结束后恢复。
// 必须在任意 SceneProperty/PlayerProperty 查询前调用。
func SwapPropertyTableForTest(t testing.TB, mode string) {
	t.Helper()
	var text string
	switch mode {
	case "official2433":
		text = officialPropertyTable2433Text
	case "fork":
		text = officialPropertyTableText
	default:
		t.Fatalf("unknown property table mode %q", mode)
	}
	prev, prevMode := officialTable, activePropertyTableMde
	officialTable = parsePropertyTable(text, mode)
	activePropertyTableMde = mode
	t.Cleanup(func() {
		officialTable = prev
		activePropertyTableMde = prevMode
	})
}

func parsePropertyTable(text, mode string) propertyTable {
	table := propertyTable{
		entries: nil,
		scene:   map[string]PropertyEntry{},
		player:  map[string]PropertyEntry{},
	}
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			panic(fmt.Sprintf("clientdata(%s): property table line %d: want 3 columns, got %d", mode, i+1, len(parts)))
		}
		index, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			panic(fmt.Sprintf("clientdata(%s): property table line %d: bad index %q", mode, i+1, parts[0]))
		}
		if index != len(table.entries) {
			panic(fmt.Sprintf("clientdata(%s): property table line %d: index %d != position %d", mode, i+1, index, len(table.entries)))
		}
		code, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil {
			panic(fmt.Sprintf("clientdata(%s): property table line %d: bad type %q", mode, i+1, parts[2]))
		}
		name := strings.TrimSpace(parts[1])
		entry := PropertyEntry{
			Index:    uint16(index),
			Name:     name,
			TypeCode: code,
			Type:     OfficialWireTypeOf(code),
		}
		table.entries = append(table.entries, entry)
		// 线型码不可编码(0/6)或名字 None 的行不参与按名解析(镜像 Java 语义)。
		if entry.Type == 0 || name == "None" {
			continue
		}
		if _, exists := table.scene[name]; !exists {
			table.scene[name] = entry
			table.sceneFields = append(table.sceneFields, entry)
		}
		table.player[name] = entry
	}
	if len(table.entries) == 0 {
		panic("clientdata: official property table is empty")
	}
	return table
}

// OfficialPropertyEntries 返回全部声明,文件顺序即 0x09 帧下发顺序。
func OfficialPropertyEntries() []PropertyEntry {
	return officialTable.entries
}

// OfficialPropertyCount 返回官方表条目数。
func OfficialPropertyCount() int {
	return len(officialTable.entries)
}

// SceneProperty 按名取场景域条目(首次出现);NPC 帧、entryScene、NPC 解析使用。
func SceneProperty(name string) (PropertyEntry, bool) {
	entry, ok := officialTable.scene[name]
	return entry, ok
}

// SceneExtensionProperty 按名取场景扩展块(商店/运动)条目:
// 首次出现且 index >= SCENE_EXTENSION_FLOOR。
func SceneExtensionProperty(name string) (PropertyEntry, bool) {
	return officialTable.byNameScene(name, SCENE_EXTENSION_FLOOR)
}

// PlayerProperty 按名取玩家/视图域条目(末次出现);
// 0x0D/0x10 玩家行、0x15/0x18 视图行使用。
func PlayerProperty(name string) (PropertyEntry, bool) {
	entry, ok := officialTable.player[name]
	return entry, ok
}

// MustSceneProperty 未知名直接 panic:编码期暴露而非线上丢帧。
func MustSceneProperty(name string) PropertyEntry {
	entry, ok := SceneProperty(name)
	if !ok {
		panic(fmt.Sprintf("clientdata: property name absent from official table (scene): %s", name))
	}
	return entry
}

// MustPlayerProperty 未知名直接 panic:编码期暴露而非线上丢帧。
func MustPlayerProperty(name string) PropertyEntry {
	entry, ok := PlayerProperty(name)
	if !ok {
		panic(fmt.Sprintf("clientdata: property name absent from official table (player): %s", name))
	}
	return entry
}

// PropertyByIndex 按索引反查(测试/调试)。
func PropertyByIndex(index int) (PropertyEntry, bool) {
	if index < 0 || index >= len(officialTable.entries) {
		return PropertyEntry{}, false
	}
	return officialTable.entries[index], true
}

// officialSceneFields 是按表序去重(每名取首现)后的可编码条目,
// 供 NPC 模板/creator 按名绑定与属性排序遍历使用。
func officialSceneFields() []PropertyEntry {
	return officialTable.sceneFields
}
