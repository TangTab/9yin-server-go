package auction

import (
	"fmt"

	"github.com/lizhanyin/9yin-server-go/internal/config"
)

// 大类（官方前缀表 1..4；ItemType.xml BigType 域同值域）。
const (
	BigTypeWeapon  = 1 // Weapon
	BigTypeArmor   = 2 // Armor
	BigTypeConsume = 3 // Consume
	BigTypeOther   = 4 // Other
	bigTypeMax     = 4
)

// 官方容量常量（AuctionManager 构造器：0x186a0 / 0x61a8）。
const (
	MaxTotalRows = 100000 // 0x186A0 全表上限（CheckConsignArgs err 22）
	MaxPerBig    = 25000  // 0x61A8 单大类上限（CheckConsignArgs err 23）
)

// Manager 对应官方 AuctionManager 单例：
// 持 MainRecord + 4×6 排序视图 + ItemType→BigType 映射。
type Manager struct {
	Main      *MainRecord
	Sorted    [][]*SortedRecord // [bigType 1..4][sortKey 1..6]
	itemTypes map[int32]int32   // ItemType.xml：ID → BigType
}

// NewManager 建管理器并初始化空表与排序视图（InitRecords 语义）。
func NewManager() *Manager {
	m := &Manager{
		Main:      NewMainRecord(MaxTotalRows),
		Sorted:    make([][]*SortedRecord, bigTypeMax+1),
		itemTypes: make(map[int32]int32, 256),
	}
	for bt := 1; bt <= bigTypeMax; bt++ {
		m.Sorted[bt] = make([]*SortedRecord, sortKeyMax+1)
		for key := 1; key <= sortKeyMax; key++ {
			m.Sorted[bt][key] = &SortedRecord{Key: key, main: m.Main, lessFn: rowLess(key)}
		}
	}
	return m
}

// GetBigType ItemType→BigType；未命中返回 Other（官方 LoadResource 缺表时
// 兜底行为不可考，取安全值 4，绝不返回 0 以免落视图越界）。
func (m *Manager) GetBigType(itemType int32) int32 {
	if bt, ok := m.itemTypes[itemType]; ok && bt >= 1 && bt <= bigTypeMax {
		return bt
	}
	return BigTypeOther
}

// GetSortedRecord 取视图（越界返回 nil，等价官方空指针不使用路径）。
func (m *Manager) GetSortedRecord(bigType, sortKey int32) *SortedRecord {
	if bigType < 1 || bigType > bigTypeMax || sortKey < 1 || sortKey > sortKeyMax {
		return nil
	}
	return m.Sorted[bigType][sortKey]
}

// GetItemNum 全表行数（官方 GetItemNum(kernel) = 视图求和口径，等价）。
func (m *Manager) GetItemNum() int { return m.Main.GetItemNum() }

// OnItemAdded 挂全部 6 个排序视图（DoConsign 的 1..6 循环）。
func (m *Manager) OnItemAdded(row int32) {
	bt := m.GetBigType(m.Main.Rows[row].ItemType)
	for key := 1; key <= sortKeyMax; key++ {
		m.Sorted[bt][key].OnItemAdded(row)
	}
}

// OnItemUpdated 竞拍改价后重定位（DoBid 路径）。
func (m *Manager) OnItemUpdated(itemType int32, row int32) {
	bt := m.GetBigType(itemType)
	for key := 1; key <= sortKeyMax; key++ {
		m.Sorted[bt][key].OnItemUpdated(row)
	}
}

// OnItemRemoved 摘全部视图（RemoveItem 路径，先记 ItemType 再删行）。
func (m *Manager) OnItemRemoved(itemType int32, row int32) {
	bt := m.GetBigType(itemType)
	for key := 1; key <= sortKeyMax; key++ {
		m.Sorted[bt][key].OnItemRemoved(row)
	}
}

// RebuildRecs 由 MainRecord 全量重建排序视图（GM msg 5 / 官方 RebuildRecs）。
func (m *Manager) RebuildRecs() {
	for bt := 1; bt <= bigTypeMax; bt++ {
		for key := 1; key <= sortKeyMax; key++ {
			m.Sorted[bt][key].rows = m.Sorted[bt][key].rows[:0]
		}
	}
	for i := range m.Main.Rows {
		r := &m.Main.Rows[i]
		if r.UniqueID == "" {
			continue
		}
		m.OnItemAdded(int32(i))
	}
}

// ClearData 清全表（GM msg 6 / 官方 ClearData）。
func (m *Manager) ClearData() {
	m.Main = NewMainRecord(MaxTotalRows) // 类型映射保留（资源数据非行情数据）
	m.RebuildRecs()
}

// LoadResource 装载 ItemType.xml 建 ItemType→BigType 映射
// （官方 LoadResource：resourcePath + "ini/trade/AuctionHouse/" +
// "ItemType.xml"，逐 section atoi(ID) 读 int(section,"BigType",4)）。
// 失败仅返回错误不阻断起服（官方打 Echo 日志后继续）。
func (m *Manager) LoadResource(path string) error {
	doc, err := config.LoadObjectFile(path)
	if err != nil {
		return fmt.Errorf("auction: not found file %s: %w", path, err)
	}
	for _, row := range doc.Rows {
		id := int32(row.Int("ID", 0))
		bt := int32(row.Int("BigType", bigTypeMax)) // 官方 ReadInteger 默认 4
		if id == 0 || bt < 0 || bt > bigTypeMax {
			continue
		}
		m.itemTypes[id] = bt
	}
	return nil
}
