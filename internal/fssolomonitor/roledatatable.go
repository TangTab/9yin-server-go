package fssolomonitor

import "fmt"

// warnFunc 官方表级 SetWarningCallback 回调：fn(ctx, wsRole, type, buf)。
// 所有表经 ReportWaring(role, type, format, args…) 格式化（官方
// vsnprintf 上限 0x100）后上抛。
type warnFunc func(role string, typ int, reason string)

// capitalTypes 官方资金类型槽数（row.capital[8]）。
const capitalTypes = 8

// bagSlots 官方 bag[3]（int64，初始 -1）。
const bagSlots = 3

// capitalCounters 官方每币种 6 计数器（0x30 一组）：
// login 基线 / expected 推算 / inc_total / inc_op_count / dec_op_count /
// dec_total。
type capitalCounters struct {
	login      int64 // 上线基准（-1 起）
	expected   int64 // 推算值（login ± 增减）
	incTotal   int64 // 窗口增量（巡检后清零）
	incOpCount int64
	decOpCount int64
	decTotal   int64
}

// roleDataRow 官方 monitor_role_data（0x1c0+名长）。
type roleDataRow struct {
	time        int64                         // 最近活动（所有写入刷新；换行依据）
	itemAmount  int64                         // +0x04 系统取物累计数量
	itemCount   int64                         // +0x08 系统取物次数
	coreItems   map[string]int64              // +0x0c 核心物品计数（懒建，巡检清）
	tradeAmount int64                         // +0x10 交易累计数量
	tradeCount  int64                         // +0x14 交易次数
	tradeCfg    map[string]int64              // +0x18 交易 configId 计数（懒建）
	tradedUUID  map[string]int64              // +0x1c 已交易 uuid 集合（懒建；回购移除）
	capital     [capitalTypes]capitalCounters // +0x20 起每型 0x30
	bag         [bagSlots]int64               // +0x1a0 初始 -1
	lastCheck   int64                         // +0x1b8 上次窗口清算
	name        string
}

// RoleDataTable 复刻 CRoleDataTable（roledatatable.cpp，核心表）。
// 官方容量 100000，MemPool 变长行 + 名→下标索引；Go 用 map + 超容
// 换最老行（ReplaceTimeoutData 语义）。
type RoleDataTable struct {
	capacity int
	rows     map[string]*roleDataRow
	warn     warnFunc
	now      func() int64 // Unix 秒

	capitalIncLimit [capitalTypes]int64 // 巡检窗口增量阀值（0=不检）
	tradeLimit      int64               // 交易次数阀值（0=不检）
	itemLimit       int64               // 取物次数阀值（0=不检）
	itemOverrides   map[string]int64    // 每物品核心物品覆盖阈值
	checkInterval   int64               // 巡检窗口秒（官方缺省 3600）
}

// NewRoleDataTable 建表（官方 ctor cap=100000、interval=3600）。
func NewRoleDataTable(now func() int64, warn warnFunc) *RoleDataTable {
	if now == nil {
		now = func() int64 { return 0 }
	}
	return &RoleDataTable{
		capacity:      100000,
		rows:          map[string]*roleDataRow{},
		warn:          warn,
		now:           now,
		itemOverrides: map[string]int64{},
		checkInterval: 3600,
	}
}

// SetCapacity 官方 SetCapacity（0=停用判定由调用方语义保留；Go 侧
// 仅记录，≤0 视为 1 以保持行存在性）。
func (t *RoleDataTable) SetCapacity(n int) {
	if n > 0 {
		t.capacity = n
	}
}

// Capacity 官方 GetCapacity。
func (t *RoleDataTable) Capacity() int { return t.capacity }

// Count 行数（呈现用）。
func (t *RoleDataTable) Count() int { return len(t.rows) }

// SetCapitalIncLimit 官方 SetCapitalIncLimit(type, v)（type 越界忽略）。
func (t *RoleDataTable) SetCapitalIncLimit(typ int, v int64) {
	if typ < 0 || typ >= capitalTypes {
		return
	}
	t.capitalIncLimit[typ] = v
}

// CapitalIncLimit 官方 GetCapitalIncLimit。
func (t *RoleDataTable) CapitalIncLimit(typ int) int64 {
	if typ < 0 || typ >= capitalTypes {
		return 0
	}
	return t.capitalIncLimit[typ]
}

// SetTradeCounterLimit 官方同名（ExchangeAmountLimit 落点）。
func (t *RoleDataTable) SetTradeCounterLimit(v int) { t.tradeLimit = int64(v) }

// TradeCounterLimit 官方同名。
func (t *RoleDataTable) TradeCounterLimit() int { return int(t.tradeLimit) }

// SetItemCounterLimit 官方两形态：带 name=每物品覆盖阈值，无 name=
// 全局默认取物次数阀值。
func (t *RoleDataTable) SetItemCounterLimit(args ...any) {
	if len(args) >= 2 {
		if name, ok := args[0].(string); ok {
			if v, ok := toInt64(args[1]); ok {
				t.itemOverrides[name] = v
			}
		}
		return
	}
	if len(args) == 1 {
		if v, ok := toInt64(args[0]); ok {
			t.itemLimit = v
		}
	}
}

// ItemCounterLimit 官方两形态 getter。
func (t *RoleDataTable) ItemCounterLimit(args ...any) int64 {
	if len(args) == 1 {
		if name, ok := args[0].(string); ok {
			return t.itemOverrides[name]
		}
	}
	return t.itemLimit
}

// SetCheckTimerInterval 官方同名（秒）。
func (t *RoleDataTable) SetCheckTimerInterval(sec int) { t.checkInterval = int64(sec) }

// CheckTimerInterval 官方同名。
func (t *RoleDataTable) CheckTimerInterval() int { return int(t.checkInterval) }

// get 官方 GetData：命中→行；超容→ReplaceTimeoutData（换 time 最老
// 行名复用）；否则新行。
func (t *RoleDataTable) get(name string) *roleDataRow {
	if row, ok := t.rows[name]; ok {
		return row
	}
	if len(t.rows) >= t.capacity {
		var oldest *roleDataRow
		for _, r := range t.rows {
			if oldest == nil || r.time < oldest.time {
				oldest = r
			}
		}
		if oldest != nil {
			delete(t.rows, oldest.name)
			row := newRoleDataRow(name)
			row.time = t.now()
			t.rows[name] = row
			return row
		}
	}
	row := newRoleDataRow(name)
	row.time = t.now()
	t.rows[name] = row
	return row
}

func newRoleDataRow(name string) *roleDataRow {
	row := &roleDataRow{
		coreItems:  nil, // 懒建
		tradeCfg:   nil,
		tradedUUID: nil,
		name:       name,
	}
	for i := range row.capital {
		row.capital[i].login = -1
		row.capital[i].expected = -1
	}
	for i := range row.bag {
		row.bag[i] = -1
	}
	return row
}

// UpdateCapital 官方 0x100940e0（上线基准，type 7 告警）。
func (t *RoleDataTable) UpdateCapital(name string, idx int, val int64) {
	if idx < 0 || idx >= capitalTypes {
		return
	}
	row := t.get(name)
	row.time = t.now()
	c := &row.capital[idx]
	if c.login != -1 && c.login != val {
		diff := c.expected - val
		if diff < 0 {
			diff = -diff
		}
		if diff != 0 {
			t.warn(name, 7, fmt.Sprintf(
				"玩家上线资金检查异常！资金：CapitalType%d 上线数量：%d 监控服务记录的下线数量：%d 玩家当前拥有数量：%d 资金偏差量：[%d]",
				idx, c.login, c.expected, val, diff))
		}
	}
	c.login = val
	c.expected = val
}

// CheckCapital 官方 0x10094320（下线核对，type 8 告警，只读）。
func (t *RoleDataTable) CheckCapital(name string, idx int, val int64) {
	if idx < 0 || idx >= capitalTypes {
		return
	}
	row := t.get(name)
	row.time = t.now()
	c := &row.capital[idx]
	if c.login != -1 && c.login != val {
		diff := c.expected - val
		if diff < 0 {
			diff = -diff
		}
		if diff != 0 {
			t.warn(name, 8, fmt.Sprintf(
				"玩家下线资金检查异常！资金：CapitalType%d 上线数量：%d 监控服务记录的下线数量：%d 玩家当前拥有数量：%d 资金偏差量：[%d]",
				idx, c.login, c.expected, val, diff))
		}
	}
}

// UpdateItem 官方 0x10094560（上线背包基线，type 0xb 告警）。
func (t *RoleDataTable) UpdateItem(name string, bagIdx int, val int64) {
	if bagIdx < 0 || bagIdx >= bagSlots {
		return
	}
	row := t.get(name)
	row.time = t.now()
	if row.bag[bagIdx] != -1 && row.bag[bagIdx] != val {
		diff := row.bag[bagIdx] - val
		if diff < 0 {
			diff = -diff
		}
		t.warn(name, 0xb, fmt.Sprintf(
			"玩家上线背包物品校验警告！背包编号：%d 监控服务记录的下线数量：%d 玩家当前拥有数量：%d 物品数量偏差量：[%d]",
			bagIdx, row.bag[bagIdx], val, diff))
	}
	row.bag[bagIdx] = val
}

// CheckItem 官方 0x10094750（下线只写值，无告警）。
func (t *RoleDataTable) CheckItem(name string, bagIdx int, val int64) {
	if bagIdx < 0 || bagIdx >= bagSlots {
		return
	}
	row := t.get(name)
	row.time = t.now()
	row.bag[bagIdx] = val
}

// IncCapital 官方 0x10094830：expected+=amt；bInc 时窗口增量累计；
// 尾调 TimerCheckRoleData（惰性巡检）。
func (t *RoleDataTable) IncCapital(name string, idx int, amt int64, bInc bool) {
	if idx < 0 || idx >= capitalTypes {
		return
	}
	row := t.get(name)
	row.time = t.now()
	c := &row.capital[idx]
	c.expected += amt
	if bInc {
		c.incTotal += amt
		c.incOpCount++
	}
	t.timerCheckRoleData(row)
}

// DecCapital 官方 0x10094c20：expected-=amt；dec 累计；不触发巡检。
func (t *RoleDataTable) DecCapital(name string, idx int, amt int64) {
	if idx < 0 || idx >= capitalTypes {
		return
	}
	row := t.get(name)
	row.time = t.now()
	c := &row.capital[idx]
	c.expected -= amt
	c.decTotal += amt
	c.decOpCount++
}

// AddItem 官方 0x10093990：srcType==28010 跳过核心物品检查；有该物品
// 覆盖阈值才走 AddCoreItem；行计数累计；尾调巡检。
func (t *RoleDataTable) AddItem(name, cfg string, amt int64, srcType int) {
	row := t.get(name)
	row.time = t.now()
	if srcType != 28010 {
		if limit, ok := t.itemOverrides[cfg]; ok {
			t.addCoreItem(row, cfg, amt, limit)
		}
	}
	row.itemAmount += amt
	row.itemCount++
	t.timerCheckRoleData(row)
}

// addCoreItem 官方 0x10095db0（type 6 告警）。
func (t *RoleDataTable) addCoreItem(row *roleDataRow, cfg string, amt, limit int64) {
	if row.coreItems == nil {
		row.coreItems = map[string]int64{}
	}
	row.coreItems[cfg] += amt
	if row.coreItems[cfg] > limit {
		t.warn(row.name, 6, fmt.Sprintf(
			"玩家获取核心物品数量超过阀值！ 物品编号：%s 已获取数量：%d 阀值数量：%d",
			cfg, row.coreItems[cfg], limit))
	}
}

// TradeItem 官方 0x10094e60：uuid 已交易过 → type 2 且不再累计 configId。
func (t *RoleDataTable) TradeItem(name, cfg, uuid string, count int64) bool {
	row := t.get(name)
	row.time = t.now()
	row.tradeAmount += count
	row.tradeCount++
	if !t.tradeItemUuid(row, uuid, count) {
		t.warn(name, 2, fmt.Sprintf(
			"同一序列号物品玩家重复交易 物品编号：%s 序列号:%s", cfg, uuid))
		return false
	}
	if row.tradeCfg == nil {
		row.tradeCfg = map[string]int64{}
	}
	row.tradeCfg[cfg] += count
	return true
}

// tradeItemUuid 官方 0x10095260：uuid 无记录 → Add 返回 true；有 →
// 计数累计返回 false。
func (t *RoleDataTable) tradeItemUuid(row *roleDataRow, uuid string, count int64) bool {
	if row.tradedUUID == nil {
		row.tradedUUID = map[string]int64{}
	}
	if _, ok := row.tradedUUID[uuid]; ok {
		row.tradedUUID[uuid] += count
		return false
	}
	row.tradedUUID[uuid] = count
	return true
}

// BuyBackItem 官方 0x10094fb0：回购后允许再交易（uuid 解禁）。
func (t *RoleDataTable) BuyBackItem(name, uuid string) {
	row := t.get(name)
	row.time = t.now()
	delete(row.tradedUUID, uuid)
}

// timerCheckRoleData 官方 0x10095aa0：距上次清算 ≥ interval 才执行；
// ①8 币种 inc_total>limit → type 0（清零）②trade_count>limit →
// type 0xa（清零）③item_count>limit → type 5（清零）④清核心物品 pod。
func (t *RoleDataTable) timerCheckRoleData(row *roleDataRow) {
	now := t.now()
	if now-row.lastCheck < t.checkInterval {
		return
	}
	row.lastCheck = now
	for i := 0; i < capitalTypes; i++ {
		if limit := t.capitalIncLimit[i]; limit != 0 && row.capital[i].incTotal > limit {
			t.warn(row.name, 0, fmt.Sprintf(
				"玩家资金增长超过阀值！ 资金类型：CapitalType:%d 资金增量：%d  设定阀值：%d",
				i, row.capital[i].incTotal, limit))
		}
		row.capital[i].incTotal = 0
	}
	if t.tradeLimit > 0 && row.tradeCount > t.tradeLimit {
		t.warn(row.name, 0xa, fmt.Sprintf(
			"玩家交易操作次数超过阀值！总次数：%d 阀值次数：%d 总交易数量:%d",
			row.tradeCount, t.tradeLimit, row.tradeAmount))
	}
	row.tradeCount = 0
	if t.itemLimit > 0 && row.itemCount > t.itemLimit {
		t.warn(row.name, 5, fmt.Sprintf(
			"玩家从系统中获取物品次数超过阀值！获取次数：%d 阀值：%d 总共获取的数量:%d",
			row.itemCount, t.itemLimit, row.itemAmount))
	}
	row.itemCount = 0
	row.coreItems = nil
}

// toInt64 官方 lua 属性面 int/int64 双形态归一。
func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	}
	return 0, false
}
