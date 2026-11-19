package fssolopublic

import (
	"sort"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fspublic"
)

// TradeItem 是 trade_item_t（CTrade::Sell 建立、Load 装载）。
// 字段名对应官方 Sell/Load 的 20 形参（str,wstr,str,wstr,str,int,int,
// wstr×4,int×4,double×3 中的业务子集）。
type TradeItem struct {
	ID         string // 寄售单号（Sell 返回 guid）
	Seller     string // 卖家账号（char*）
	SellerName string // 卖家角色名（wstr）
	ItemKey    string // 物品 key（char*）
	ItemName   string // 物品名（wstr）
	Count      int32
	Price      int32
	Currency   string // wstr 币种名
	Desc       string // wstr 物品描述
	Status     int32  // GetStatus：1=在售
	Profit     int32  // GainProfit 累计分成后清零
	TimeoutAt  time.Time
}

// Trade 复刻 CTrade（trade.cpp，SoloPublic+0x134）：跨服寄售行。
// 装载链：store 0x28 应答（Load 20 字段）；超时单由 CheckTimeout 出队。
type Trade struct {
	mu    sync.RWMutex
	items map[string]*TradeItem // id → item
	seq   int
}

func NewTrade() *Trade { return &Trade{items: map[string]*TradeItem{}} }

// Sell 官方 Sell(seller, sellerName, itemKey, count, itemName, currency,
// desc, price, timeoutSecs, ...) → 返回寄售单号。
func (t *Trade) Sell(seller, sellerName, itemKey, itemName string, count, price int32,
	currency, desc string, timeoutSecs float64) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	id := fspublic.CreateGUID()
	it := &TradeItem{
		ID: id, Seller: seller, SellerName: sellerName,
		ItemKey: itemKey, ItemName: itemName, Count: count,
		Price: price, Currency: currency, Desc: desc, Status: 1,
	}
	if timeoutSecs > 0 {
		it.TimeoutAt = time.Now().Add(time.Duration(timeoutSecs * float64(time.Second)))
	}
	t.items[id] = it
	return id
}

// Unsell 官方 Unsell(seller, itemKey, itemName)：下架自己的在售单。
func (t *Trade) Unsell(seller, itemKey, itemName string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	for id, it := range t.items {
		if it.Seller == seller && it.ItemKey == itemKey && it.ItemName == itemName && it.Status == 1 {
			delete(t.items, id)
			return true
		}
	}
	return false
}

// Buy 官方 Buy(buyer, buyerName, id, count, &price, &seller, &profit)：
// 购买在售单（整单），返回卖家/单价。
func (t *Trade) Buy(buyer, buyerName, id string) (ok bool, price int32, seller string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	it, had := t.items[id]
	if !had || it.Status != 1 {
		return false, 0, ""
	}
	price, seller = it.Price, it.Seller
	delete(t.items, id)
	_ = buyer
	_ = buyerName
	return true, price, seller
}

// Search 官方 Search(itemName, key, minLevel, maxLevel, sort) → 在售清单
// （时间序）。
func (t *Trade) Search(itemName, itemKey string) []*TradeItem {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var out []*TradeItem
	for _, it := range t.items {
		if it.Status != 1 {
			continue
		}
		if itemName != "" && it.ItemName != itemName {
			continue
		}
		if itemKey != "" && it.ItemKey != itemKey {
			continue
		}
		cp := *it
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].TimeoutAt.Equal(out[j].TimeoutAt) {
			return out[i].TimeoutAt.Before(out[j].TimeoutAt)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// GetStatus / GetItemName / GetItemInfo / GainProfit 官方查询面。
func (t *Trade) GetStatus(id string) int32 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if it, ok := t.items[id]; ok {
		return it.Status
	}
	return 0
}

func (t *Trade) GetItemName(id string) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if it, ok := t.items[id]; ok {
		return it.ItemName, true
	}
	return "", false
}

func (t *Trade) GetItemInfo(id string) (*TradeItem, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	it, ok := t.items[id]
	if !ok {
		return nil, false
	}
	cp := *it
	return &cp, true
}

func (t *Trade) GainProfit(seller, itemKey, itemName string) int32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, it := range t.items {
		if it.Seller == seller && it.ItemKey == itemKey && it.ItemName == itemName {
			p := it.Profit
			it.Profit = 0
			return p
		}
	}
	return 0
}

// CheckTimeout 官方 CheckTimeout()：弹出全部超时单（寄售时限到）。
func (t *Trade) CheckTimeout(now time.Time) []*TradeItem {
	t.mu.Lock()
	defer t.mu.Unlock()
	var dead []*TradeItem
	for id, it := range t.items {
		if !it.TimeoutAt.IsZero() && now.After(it.TimeoutAt) {
			dead = append(dead, it)
			delete(t.items, id)
		}
	}
	return dead
}

// Load 官方 Load（0x28 装载路径 20 字段）。
func (t *Trade) Load(id string, seller string, sellerName string, itemKey string,
	itemName string, count, price int32, currency, desc, extra string,
	status, profit int32, timeoutSecs, priceD, fee float64) bool {
	if id == "" || seller == "" {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.items[id]; ok {
		return true // 幂等
	}
	it := &TradeItem{
		ID: id, Seller: seller, SellerName: sellerName,
		ItemKey: itemKey, ItemName: itemName, Count: count,
		Price: price, Currency: currency, Desc: desc,
		Status: status, Profit: profit,
	}
	if timeoutSecs > 0 {
		it.TimeoutAt = time.Now().Add(time.Duration(timeoutSecs * float64(time.Second)))
	}
	t.items[id] = it
	return true
}

// Count 呈现用。
func (t *Trade) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.items)
}
