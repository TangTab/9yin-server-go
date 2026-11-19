package fssolomonitor

import (
	"fmt"
	"math"
)

// ---- detail 环表族（官方"capacity 定长环 + TRefStringIndex 键索引"）----
//
// 通用语义（tmp_research/solomonitor/notes_C_tables.md §通用行为）：
// capacity=0 时 Add/Buy/Login 直接返回不记录；满后 cursor 回绕覆盖最老
// 记录并 Remove 旧键。

// detailRing 官方定长环 + 键索引容器的 Go 投影：槽存任意记录值，
// key→槽号索引，回绕覆盖。
type detailRing[T any] struct {
	capacity int
	cursor   int
	rows     []T // len==capacity（空槽零值）
	index    map[string]int
}

func newDetailRing[T any]() *detailRing[T] {
	return &detailRing[T]{index: map[string]int{}}
}

// SetCapacity 官方 SetCapacity：n<=cap 忽略；首次分配。
func (r *detailRing[T]) SetCapacity(n int) {
	if n <= r.capacity {
		return
	}
	rows := make([]T, n)
	copy(rows, r.rows)
	r.rows = rows
	r.capacity = n
}

// Capacity 官方 GetCapacity。
func (r *detailRing[T]) Capacity() int { return r.capacity }

// lookup 命中返回槽记录指针；未命中 nil。
func (r *detailRing[T]) lookup(key string) *T {
	if slot, ok := r.index[key]; ok {
		return &r.rows[slot]
	}
	return nil
}

// slotKey 槽当前键（覆盖判定用）：由子表提供槽→键提取。
func (r *detailRing[T]) put(key string, rec T, keyOf func(T) string) *T {
	if r.capacity == 0 {
		return nil
	}
	if r.cursor >= r.capacity {
		r.cursor = 0
	}
	old := r.rows[r.cursor]
	if k := keyOf(old); k != "" {
		delete(r.index, k)
	}
	r.rows[r.cursor] = rec
	r.index[key] = r.cursor
	r.cursor++
	return &r.rows[r.index[key]]
}

// ---- CCreateItemDetailTable（注册名 CreateItemDetail，容量 1000000）----
//
// 记录=uuid 定长键（0x2b）；Add 查重 → type 9。

type CreateItemDetailTable struct {
	ring *detailRing[string]
	warn warnFunc
}

func NewCreateItemDetailTable(now func() int64, warn warnFunc) *CreateItemDetailTable {
	return &CreateItemDetailTable{ring: newDetailRing[string](), warn: warn}
}

func (t *CreateItemDetailTable) SetCapacity(n int) { t.ring.SetCapacity(n) }
func (t *CreateItemDetailTable) Capacity() int     { return t.ring.Capacity() }

// Add 官方 0x10078180：cap=0→false；uuid 空→true；已存在 → type 9 →
// false；否则环写 → true。
func (t *CreateItemDetailTable) Add(cfg, uuid, role string) bool {
	if t.ring.capacity == 0 {
		return false
	}
	if uuid == "" {
		return true
	}
	if t.ring.lookup(uuid) != nil {
		t.warn(role, 9, fmt.Sprintf(
			"系统创建了重复序列号的物品！编号：%s 序列号：%s", cfg, uuid))
		return false
	}
	t.ring.put(uuid, uuid, func(s string) string { return s })
	return true
}

// ---- CRecvLetterDetailTable（注册名 RecvLetterDetail，容量 1000000）----
//
// 记录 0x2d={计数A u8, 计数B u8, letterId[0x2b]}；附件类型 10032/10042
// →A++，30010/35002 →B++，其它 →A++；A>1 或 B>1 → type 4。

type recvLetterRec struct {
	countA uint8
	countB uint8
}

type RecvLetterDetailTable struct {
	ring *detailRing[recvLetterKeyed]
	warn warnFunc
}

type recvLetterKeyed struct {
	recvLetterRec
	letterID string
}

func NewRecvLetterDetailTable(now func() int64, warn warnFunc) *RecvLetterDetailTable {
	return &RecvLetterDetailTable{ring: newDetailRing[recvLetterKeyed](), warn: warn}
}

func (t *RecvLetterDetailTable) SetCapacity(n int) { t.ring.SetCapacity(n) }
func (t *RecvLetterDetailTable) Capacity() int     { return t.ring.Capacity() }

// Add 官方 0x10092ae0（letterId=备注列，role=源角色）。
func (t *RecvLetterDetailTable) Add(letterID, role string, attachType int, _ int64) bool {
	if t.ring.capacity == 0 {
		return false
	}
	if rec := t.ring.lookup(letterID); rec != nil {
		switch attachType {
		case 10032, 10042:
			rec.countA++
		case 30010, 35002:
			rec.countB++
		default:
			rec.countA++
		}
		if rec.countA > 1 || rec.countB > 1 {
			n := rec.countA
			if n <= 1 {
				n = rec.countB
			}
			t.warn(role, 4, fmt.Sprintf(
				"重复提取系统邮件附件数量：%d 邮件编号：%s", n, letterID))
		}
		return true
	}
	rec := recvLetterKeyed{letterID: letterID}
	switch attachType {
	case 10032, 10042:
		rec.countA = 1
	case 30010, 35002:
		rec.countB = 1
	default:
		rec.countA = 1
	}
	t.ring.put(letterID, rec, func(r recvLetterKeyed) string { return r.letterID })
	return true
}

// ---- CTradeItemDetailTable（注册名 TradeItemDetail，容量 100000）----
//
// 行 0x48={wsName[34], configId→detail 子索引}；detail 0x34={cfg[43],
// 次数 u8, 累计价格 i32, free 链}。Buy/Sell 对称：同玩家同编号已有
// 明细 → 累计价格>0 → type 0xd。

type tradeDetail struct {
	cfg       string
	count     int64 // 累计交易次数（官方 u8，Go 放宽）
	totalCost int64 // 累计价格（每次 += abs(price)；命中告警后重置）
}

type tradeItemRow struct {
	name    string
	details map[string]*tradeDetail // configId → detail
}

type TradeItemDetailTable struct {
	ring *detailRing[*tradeItemRow]
	warn warnFunc
}

func NewTradeItemDetailTable(now func() int64, warn warnFunc) *TradeItemDetailTable {
	return &TradeItemDetailTable{ring: newDetailRing[*tradeItemRow](), warn: warn}
}

func (t *TradeItemDetailTable) SetCapacity(n int) { t.ring.SetCapacity(n) }
func (t *TradeItemDetailTable) Capacity() int     { return t.ring.Capacity() }

// tradeItem 官方 BuyItem/SellItem 共核（0x100ab900/0x100abe60 对称）。
// 键=交易对端 wsTrader；告警对象=wsRole。
func (t *TradeItemDetailTable) tradeItem(cfg string, price int64, role, trader, uuid string, logType int) bool {
	if t.ring.capacity == 0 || cfg == "" || trader == "" {
		return true
	}
	var row *tradeItemRow
	if p := t.ring.lookup(trader); p != nil {
		row = *p
	} else {
		slot := t.ring.put(trader, &tradeItemRow{name: trader, details: map[string]*tradeDetail{}},
			func(r *tradeItemRow) string {
				if r == nil {
					return ""
				}
				return r.name
			})
		if slot == nil {
			return true
		}
		row = *slot
	}
	if row.details == nil {
		row.details = map[string]*tradeDetail{}
	}
	abs := price
	if abs < 0 {
		abs = -abs
	}
	if d, ok := row.details[cfg]; ok {
		d.totalCost += abs
		if d.totalCost > 0 {
			t.warn(role, 0xd, fmt.Sprintf(
				"物品交易获取出错！ 交易编号：%s 物品唯一编号：%s 多数量：%d 物品编号：%s 日志类型：%d",
				trader, uuid, d.totalCost, cfg, logType))
		}
		// 官方无论命中与否都覆写明细：price=abs(price)、次数+1。
		d.totalCost = abs
		d.count++
		return true
	}
	row.details[cfg] = &tradeDetail{cfg: cfg, count: 1, totalCost: abs}
	return true
}

// BuyItem 官方 10031/35001 路径。
func (t *TradeItemDetailTable) BuyItem(cfg string, price int64, role, trader, uuid string, logType int) bool {
	return t.tradeItem(cfg, price, role, trader, uuid, logType)
}

// SellItem 官方 20058/45001 路径（与 Buy 对称）。
func (t *TradeItemDetailTable) SellItem(cfg string, price int64, role, trader, uuid string, logType int) bool {
	return t.tradeItem(cfg, price, role, trader, uuid, logType)
}

// ---- CLoginDetailTable（注册名 LoginDetail，容量 100000）----
//
// 行 0x9c={mac[0x80], ip[0x10], 在线数 u32, 账号→次数 pod}。

type loginRow struct {
	mac      string
	ip       string
	online   int64
	accounts map[string]int64
}

type LoginDetailTable struct {
	ring        *detailRing[*loginRow]
	warn        warnFunc
	amountLimit int // 不同账号数上限（0=不检）
	onlineLimit int // 同时在线账号数上限（0=不检）
}

func NewLoginDetailTable(now func() int64, warn warnFunc) *LoginDetailTable {
	return &LoginDetailTable{ring: newDetailRing[*loginRow](), warn: warn}
}

func (t *LoginDetailTable) SetCapacity(n int) { t.ring.SetCapacity(n) }
func (t *LoginDetailTable) Capacity() int     { return t.ring.Capacity() }

// SetAmountLimit 官方同名（RoleNumLimit 落点）。
func (t *LoginDetailTable) SetAmountLimit(n int) { t.amountLimit = n }

// SetOnlineAmountLimit 官方同名（OnlineRoleNumLimit 落点）。
func (t *LoginDetailTable) SetOnlineAmountLimit(n int) { t.onlineLimit = n }

// Login 官方 0x10084270：pod 记账号次数；不同账号数>limit（且
// limit!=0）→ type 0xe；在线数>limit（且 limit!=0）→ type 0xf。
func (t *LoginDetailTable) Login(mac, ip, account, role string) bool {
	if t.ring.capacity == 0 {
		return false
	}
	var row *loginRow
	if p := t.ring.lookup(mac); p != nil {
		row = *p
	} else {
		slot := t.ring.put(mac, &loginRow{mac: mac, ip: ip, accounts: map[string]int64{}},
			func(r *loginRow) string {
				if r == nil {
					return ""
				}
				return r.mac
			})
		if slot == nil {
			return false
		}
		row = *slot
	}
	row.accounts[account]++
	n := int64(len(row.accounts)) // 官方 GetCount()=不同 key 数
	if t.amountLimit != 0 && n > int64(t.amountLimit) {
		t.warn(role, 0xe, fmt.Sprintf(
			"同一机器登陆账号过多！ 机器Mac：%s 机器IP：%s 当前账号：%s 总登陆账号个数：%d 阀值：%d",
			mac, ip, role, n, t.amountLimit))
	}
	row.online++
	if t.onlineLimit != 0 && row.online > int64(t.onlineLimit) {
		t.warn(role, 0xf, fmt.Sprintf(
			"同一机器同时登陆账号过多！ 机器Mac：%s 机器IP：%s 当前账号：%s 总登陆账号个数：%d 阀值：%d",
			mac, ip, role, row.online, t.onlineLimit))
	}
	return true
}

// LoginOut 官方 0x100847f0：在线数−1（pod 记录保留——官方键从不删除）。
func (t *LoginDetailTable) LoginOut(mac, ip, account, role string) bool {
	if p := t.ring.lookup(mac); p != nil {
		(*p).online--
		return true
	}
	return false
}

// ---- CGlobalItemDetailTable（注册名 ItemDetail，容量 0=停用）----
//
// 记录 0x30={cfgId, 净数量 i32}；全服净量买加卖减、堆叠与合法性检查。

type globalItemRow struct {
	cfg   string
	count int64 // 全服净数量
}

type GlobalItemDetailTable struct {
	ring *detailRing[globalItemRow]
	warn warnFunc
}

func NewGlobalItemDetailTable(now func() int64, warn warnFunc) *GlobalItemDetailTable {
	return &GlobalItemDetailTable{ring: newDetailRing[globalItemRow](), warn: warn}
}

func (t *GlobalItemDetailTable) SetCapacity(n int) { t.ring.SetCapacity(n) }
func (t *GlobalItemDetailTable) Capacity() int     { return t.ring.Capacity() }

// BuyItem 官方 0x100833c0：count+=|n|；|count|>maxStack → type 4（增加）。
func (t *GlobalItemDetailTable) BuyItem(cfg string, n int64, maxStack int64, role string) bool {
	if t.ring.capacity == 0 {
		return false
	}
	row := t.ring.lookup(cfg)
	if row == nil {
		row = t.ring.put(cfg, globalItemRow{cfg: cfg},
			func(r globalItemRow) string { return r.cfg })
	}
	abs := n
	if abs < 0 {
		abs = -abs
	}
	row.count += abs
	if abs64(row.count) > maxStack {
		t.warn(role, 4, fmt.Sprintf(
			"物品增加的数量已经超过堆叠上限！ 物品编号：%s 当前数量：%d 最大堆叠数量：%d",
			cfg, abs64(row.count), maxStack))
	}
	return true
}

// SellItem 官方 0x10083640：count-=|n|；n<0 → type 0xc；|count|>
// maxStack → type 4（减少）。
func (t *GlobalItemDetailTable) SellItem(cfg string, n int64, maxStack int64, role string) bool {
	if t.ring.capacity == 0 {
		return false
	}
	row := t.ring.lookup(cfg)
	if row == nil {
		row = t.ring.put(cfg, globalItemRow{cfg: cfg},
			func(r globalItemRow) string { return r.cfg })
	}
	abs := n
	if abs < 0 {
		abs = -abs
	}
	row.count -= abs
	if n < 0 {
		t.warn(role, 0xc, fmt.Sprintf(
			"检查物品操作的数量非法！ 物品编号：%s 操作数量：%d", cfg, n))
	}
	if abs64(row.count) > maxStack {
		t.warn(role, 4, fmt.Sprintf(
			"物品减少的数量已经超过堆叠上限！ 物品编号：%s 当前数量：%d 最大堆叠数量：%d",
			cfg, abs64(row.count), maxStack))
	}
	return true
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	if n == math.MinInt64 {
		return math.MaxInt64
	}
	return n
}
