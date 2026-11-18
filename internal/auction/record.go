package auction

import (
	"sort"
	"strings"
)

// Row 是 MainRecord 的 18 列行结构。
//
// 列号由 MainRecord::Get*/Set* 的列下标反汇编钉死
// （GetUniqueID=0 … GetQuality 见 Compare* 前置列读：Level=0xF Quality=0x10）。
// 寄售帧携带 reqid 后按本结构顺序原样透传（wire[2..19] ↔ col[0..17]）。
type Row struct {
	UniqueID    string  // col0  物品唯一号（FindItem 键）
	ConfigID    string  // col1  配置 ID
	SortType    int32   // col2  官方用途未名（int，透传保留）
	ItemString  string  // col3  物品序列化串（原样搬运）
	SellerUid   string  // col4  卖家 uid
	StartPrice  int64   // col5  起拍价（FsGame 寄售帧字段实证）
	FixedPrice  int64   // col6  一口价
	TransFeeCoe float32 // col7  交易手续费系数
	KeepingFee  int64   // col8  保管费（FsGame GetKeepingFee 计算后随帧透传）
	OverTime    int64   // col9  到期时刻（Unix 秒，util_get_time_64 口径）
	CurBiderUid string  // col10 当前最高出价者（""=无）
	CurPrice    int64   // col11 当前价（初始=起拍价）
	State       int32   // col12 状态：1=在拍 2=有出价 12=一口价成交(瞬态)
	ItemType    int32   // col13 物品类型（ItemType.xml → BigType）
	SellerName  string  // col14 卖家名（widestr）
	Level       int32   // col15 等级（CompareLevel 排序键）
	Quality     int32   // col16 品质（CompareQuality 排序键）
	ExtraStr    string  // col17 官方用途未名（string，透传保留）
}

// 行状态常量（MainRecord::SetState 实参 + 超时分支条件）。
const (
	StateConsigned int32 = 1  // 寄售初始
	StateBidded    int32 = 2  // 已有出价（超时→成交 msg 11）
	StateFixedSold int32 = 12 // 一口价成交（随即移除）
)

// MainRecord 全量表 + 空行复用链（官方 FindEmptyRow/pop_back 语义）。
type MainRecord struct {
	Rows  []Row
	Used  int
	free  []int32 // 空行号栈（LIFO）
	index map[string]int32
}

// NewMainRecord 建 maxRows 容量（官方上限 100000）。
func NewMainRecord(maxRows int) *MainRecord {
	m := &MainRecord{
		Rows:  make([]Row, maxRows),
		index: make(map[string]int32, 1024),
	}
	m.InitEmptyRowList()
	return m
}

// InitEmptyRowList 重建空行链（官方 InitEmptyRowList：倒序压栈 → pop 得最小号）。
func (m *MainRecord) InitEmptyRowList() {
	m.free = m.free[:0]
	for i := len(m.Rows) - 1; i >= 0; i-- {
		if m.Rows[i].UniqueID == "" {
			m.free = append(m.free, int32(i))
		}
	}
	// 栈内倒序排列使 pop 取到最小空号：压栈顺序 = 行号降序
	sort.Slice(m.free, func(a, b int) bool { return m.free[a] > m.free[b] })
	m.Used = len(m.Rows) - len(m.free)
}

// IsRecFull 表满（官方 HasEmptyRow 取反）。
func (m *MainRecord) IsRecFull() bool { return len(m.free) == 0 }

// FindItem 按唯一号找行号，无则 -1（官方线性扫描语义，此处走索引等价）。
func (m *MainRecord) FindItem(uid string) int32 {
	if i, ok := m.index[uid]; ok {
		return i
	}
	return -1
}

// AddItem 追加行（官方 = 取空行 + IRecord::AddRow 写列 + 摘空行号）。
// 返回行号；表满返回 -1。
func (m *MainRecord) AddItem(r Row) int32 {
	if len(m.free) == 0 {
		return -1
	}
	row := m.free[len(m.free)-1]
	m.free = m.free[:len(m.free)-1]
	m.Rows[row] = r
	m.index[r.UniqueID] = row
	m.Used++
	return row
}

// RemoveItem 按行号清行（ClearRow 语义：置零 + 行号回链）。
func (m *MainRecord) RemoveItem(row int32) {
	if row < 0 || int(row) >= len(m.Rows) {
		return
	}
	uid := m.Rows[row].UniqueID
	if uid == "" {
		return
	}
	delete(m.index, uid)
	m.Rows[row] = Row{}
	m.free = append(m.free, row)
	m.Used--
}

// GetItemNum 非空行数。
func (m *MainRecord) GetItemNum() int { return m.Used }

// 排序视图键位（官方 SortedRecord 槽位 1..6，名字取自构造期名字表）。
const (
	SortByQuality    = 1 // SortedFirstByQuality
	SortByRestTime   = 2 // SortFirstByRestTime（到期升序=剩余短在前）
	SortByLevel      = 3 // SortedFirstByLevel
	SortByCurPrice   = 4 // SortedFirstByCurPrice
	SortByFixedPrice = 5 // SortedFirstByFixedPrice
	SortBySellerName = 6 // SortedFirstBySellerName
	sortKeyMax       = 6
)

// SortedRecord 单个排序视图：MainRecord 行号的有序切片。
// 官方 Compare 系列全部是值差比较（CompareQuality = a-b），
// 差值仅取符号 —— 等价于稳定排序的 less。
type SortedRecord struct {
	Key    int
	rows   []int32
	main   *MainRecord
	lessFn func(a, b *Row) bool
}

// OnItemAdded 插入行号并保持有序（官方二分插入语义）。
func (s *SortedRecord) OnItemAdded(row int32) {
	pos := sort.Search(len(s.rows), func(i int) bool { return s.lessFn(&s.main.Rows[row], &s.main.Rows[s.rows[i]]) })
	s.rows = append(s.rows, 0)
	copy(s.rows[pos+1:], s.rows[pos:])
	s.rows[pos] = row
}

// OnItemUpdated 价格/出价人变动后的重定位（先删后插，官方 OnItemUpdated 语义）。
func (s *SortedRecord) OnItemUpdated(row int32) {
	s.OnItemRemoved(row)
	s.OnItemAdded(row)
}

// OnItemRemoved 摘除行号。
func (s *SortedRecord) OnItemRemoved(row int32) {
	for i, r := range s.rows {
		if r == row {
			s.rows = append(s.rows[:i], s.rows[i+1:]...)
			return
		}
	}
}

// GetItemNum 视图行数。
func (s *SortedRecord) GetItemNum() int { return len(s.rows) }

// Items 返回有序行号快照。
func (s *SortedRecord) Items() []int32 { return append([]int32(nil), s.rows...) }

func rowLess(key int) func(a, b *Row) bool {
	switch key {
	case SortByQuality:
		return func(a, b *Row) bool { return a.Quality < b.Quality }
	case SortByRestTime:
		// 官方 CompareRestTime = 值差升序（OverTime 小 = 剩余时间短 = 先到期在前）
		return func(a, b *Row) bool { return a.OverTime < b.OverTime }
	case SortByLevel:
		return func(a, b *Row) bool { return a.Level < b.Level }
	case SortByCurPrice:
		return func(a, b *Row) bool { return a.CurPrice < b.CurPrice }
	case SortByFixedPrice:
		return func(a, b *Row) bool { return a.FixedPrice < b.FixedPrice }
	case SortBySellerName:
		return func(a, b *Row) bool {
			// 官方 CompareSellerName = strcmp；中文 GBK 字节序即等价Go 字节比较
			return strings.Compare(a.SellerName, b.SellerName) < 0
		}
	}
	return nil
}
