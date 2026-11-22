package clientdata

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

//go:embed official_property_table_2433.txt
var officialPropertyTable2433Text string

// 属性表唯一方言：外服私服 7074 线 2433 项官方真表 + 61 行尾追段
// （2026-11-14 定案，分支 official-2433-only）。官方 2433 行与外服
// 抓包逐字段同方言——可直接对照/回放模拟发送；尾追段（2433..2493，
// PosiX/Y/Z/Orient@2434-2437、Race/FloatingState/BufferListStr 等
// fork 头块缺席名，StopTime 刻意排除）是 fork 客户端渲染头行通道
// ——fxnet2 分发表定案（docs/official/fxnet2-op-dispatch-table.md）：
// fork fxnet2 主表仅 0x38 项（0x46 家族无 handler、0x28 走 LZ 解压），
// 官方式 NPC 详情通道在 fork 客户端先天不通，渲染依赖 0x0D 携带头行。
// 表追加只许末尾（客户端按位置绑定索引，官方行零漂移）。

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

// 默认装载 2433 官方方言（var 初始化先于一切 init()，golden_table.go
// 的 init 可在其上按 env 覆盖为真机镜像方言）。
var officialTable = parsePropertyTable(officialPropertyTable2433Text, "official2433")

// ActivePropertyTableMode 返回当前生效的属性表方言（日志/诊断用）：
// official2433（默认，2433+61 尾追 = 2494 项）或 golden（真机镜像 2086 项）。
func ActivePropertyTableMode() string {
	if goldenTableActive {
		return "golden"
	}
	return "official2433"
}

// OfficialPropertyTableFrame 把官方表条目编码为 0x09 协商帧：
// [0x09][u16 N]{name\0 + u8 typeCode}×N（b-line 真机实证形态，含尾追段）。
func OfficialPropertyTableFrame() []byte {
	entries := officialTable.entries
	msg := make([]byte, 3, 3+len(entries)*16)
	msg[0] = 0x09
	binary.LittleEndian.PutUint16(msg[1:], uint16(len(entries)))
	for _, entry := range entries {
		msg = append(msg, entry.Name...)
		msg = append(msg, 0, byte(entry.TypeCode))
	}
	return msg
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

// OfficialSceneFields officialSceneFields 的导出面（NPC 0x0D 属性按
// 表序合成——2433 线 OrderedProperties 同构）。
func OfficialSceneFields() []PropertyEntry {
	return officialSceneFields()
}
