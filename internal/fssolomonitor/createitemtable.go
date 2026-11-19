package fssolomonitor

import "fmt"

// createItemRow 官方 monitor_create_item（0x10+cfg 长）：
// 窗口起点 / 窗口累计数量 / 窗口创建次数。
type createItemRow struct {
	time   int64 // 窗口起点（超 interval 清零重开）
	amount int64 // 窗口内累计创建数量
	count  int64 // 窗口内创建次数
	cfg    string
}

// CreateItemTable 复刻 CCreateItemTable（crateitemtable.cpp，系统产粮
// 滚动窗口）。官方容量 10000；每物品阈值 pod（[SysItemLimit] 落点）；
// 无覆盖阈值的物品只刷窗口时间不告警（无全局默认值）。
type CreateItemTable struct {
	capacity int
	rows     map[string]*createItemRow
	limits   map[string]int64 // 每物品阈值
	interval int64            // 滚动窗口秒（官方缺省 3600）
	warn     warnFunc
	now      func() int64
}

// NewCreateItemTable 建表（官方 ctor cap=10000、interval=3600）。
func NewCreateItemTable(now func() int64, warn warnFunc) *CreateItemTable {
	if now == nil {
		now = func() int64 { return 0 }
	}
	return &CreateItemTable{
		capacity: 10000,
		rows:     map[string]*createItemRow{},
		limits:   map[string]int64{},
		interval: 3600,
		warn:     warn,
		now:      now,
	}
}

// SetCapacity 官方同名。
func (t *CreateItemTable) SetCapacity(n int) {
	if n > 0 {
		t.capacity = n
	}
}

// Capacity 官方同名。
func (t *CreateItemTable) Capacity() int { return t.capacity }

// Count 行数（呈现用）。
func (t *CreateItemTable) Count() int { return len(t.rows) }

// SetItemCounterLimit 官方 SetItemCounterLimit(cfg, v)。
func (t *CreateItemTable) SetItemCounterLimit(cfg string, v int) {
	t.limits[cfg] = int64(v)
}

// ItemCounterLimit 官方 GetItemCounterLimit(cfg)（无记录 0）。
func (t *CreateItemTable) ItemCounterLimit(cfg string) int64 {
	return t.limits[cfg]
}

// SetCheckTimerInterval 官方同名（LoadSystemConfig 下发）。
func (t *CreateItemTable) SetCheckTimerInterval(sec int) { t.interval = int64(sec) }

// CheckTimerInterval 官方同名。
func (t *CreateItemTable) CheckTimerInterval() int { return int(t.interval) }

// get 官方 GetData/NewData/ReplaceTimeoutData 语义（同 RoleData 表型）。
func (t *CreateItemTable) get(cfg string) *createItemRow {
	if row, ok := t.rows[cfg]; ok {
		return row
	}
	if len(t.rows) >= t.capacity {
		var oldest *createItemRow
		for _, r := range t.rows {
			if oldest == nil || r.time < oldest.time {
				oldest = r
			}
		}
		if oldest != nil {
			delete(t.rows, oldest.cfg)
		}
	}
	row := &createItemRow{cfg: cfg, time: t.now()}
	t.rows[cfg] = row
	return row
}

// AddItem 官方 0x10079920：amount+=count、count+=1；有该物品阈值时
// 滚动窗口（now-time>=interval → amount=0、time=now），amount>limit
// 或 窗口次数>limit → type 1 → false；无阈值仅刷新 time。
func (t *CreateItemTable) AddItem(cfg string, count int64, role string) bool {
	row := t.get(cfg)
	row.amount += count
	row.count++
	limit, ok := t.limits[cfg]
	if !ok {
		row.time = t.now()
		return true
	}
	now := t.now()
	if now-row.time >= t.interval {
		row.amount = 0
		row.time = now
	}
	if row.amount > limit || row.count > limit {
		t.warn(role, 1, fmt.Sprintf(
			"系统创建物品超过上限值！物品编号：%s 当前总数量:%d 限制数量:%d 此次增量:%d",
			cfg, row.amount, limit, row.count))
		return false
	}
	return true
}
